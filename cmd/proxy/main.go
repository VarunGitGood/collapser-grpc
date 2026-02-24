package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/VarunGitGood/collapser-grpc/internal/collapser"
	"github.com/VarunGitGood/collapser-grpc/internal/config"
	"github.com/VarunGitGood/collapser-grpc/internal/logger"
	"github.com/VarunGitGood/collapser-grpc/internal/proxy"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger
	if err := logger.Init(cfg.LogLevel, cfg.LogFormat); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Collapser Proxy",
		zap.Int("grpc_port", cfg.GRPCPort),
		zap.Int("metrics_port", cfg.MetricsPort),
		zap.String("backend_address", cfg.BackendAddress))

	// Initialize Collapser
	collapserCfg := collapser.Config{
		ResultCacheDuration: cfg.ResultCacheDuration,
		BackendTimeout:      cfg.BackendTimeout,
		CleanupInterval:     cfg.CleanupInterval,
	}
	c := collapser.NewCollapser(collapserCfg)
	if err := c.Start(); err != nil {
		logger.Fatal("failed to start collapser", zap.Error(err))
	}
	defer c.Stop()

	// Initialize Proxy Handler
	proxyHandler := proxy.NewHandler(c, cfg.BackendAddress)

	// Start Metrics Server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		logger.Info("Metrics server starting", zap.Int("port", cfg.MetricsPort))
		if err := http.ListenAndServe(":"+strconv.Itoa(cfg.MetricsPort), mux); err != nil {
			logger.Error("metrics server failed", zap.Error(err))
		}
	}()

	// Start gRPC Proxy Server
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(cfg.GRPCPort))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	// Listen for OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := proxyHandler.Serve(lis); err != nil {
			logger.Error("proxy server failed", zap.Error(err))
		}
	}()

	<-sigCh
	logger.Info("Shutting down gracefully...")
}

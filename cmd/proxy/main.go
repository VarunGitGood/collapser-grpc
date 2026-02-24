package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/VarunGitGood/collapser-grpc/internal/collapser"
	"github.com/VarunGitGood/collapser-grpc/internal/config"
	"github.com/VarunGitGood/collapser-grpc/internal/logger"
	"github.com/VarunGitGood/collapser-grpc/internal/proxy"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger.Setup(cfg.LogLevel, cfg.LogFormat)
	logger.Info("starting proxy", zap.Int("grpc_port", cfg.GRPCPort), zap.Int("metrics_port", cfg.MetricsPort))

	// Start metrics server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		addr := fmt.Sprintf(":%d", cfg.MetricsPort)
		logger.Info("metrics server starting", zap.String("addr", addr))
		if err := http.ListenAndServe(addr, mux); err != nil {
			logger.Fatal("metrics server failed", zap.Error(err))
		}
	}()

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

	// Initialize Proxy Handler
	handler := proxy.NewHandler(cfg.BackendAddress, c)

	// Start gRPC Server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	s := grpc.NewServer(
		grpc.UnknownServiceHandler(handler.Handle),
	)

	go func() {
		logger.Info("proxy gRPC server starting", zap.Int("port", cfg.GRPCPort))
		if err := s.Serve(lis); err != nil {
			logger.Fatal("failed to serve", zap.Error(err))
		}
	}()

	// Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	logger.Info("shutting down...")

	s.GracefulStop()
	c.Stop()

	logger.Info("shutdown complete")
}

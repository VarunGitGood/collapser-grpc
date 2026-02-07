package main

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/VarunGitGood/collapser-grpc/internal/collapser"
	"github.com/VarunGitGood/collapser-grpc/internal/proxy"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Metrics server on http://localhost:2112/metrics")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	config := collapser.Config{
		ResultCacheDuration: 100 * time.Millisecond,
		BackendTimeout:      10 * time.Second,
		CleanupInterval:     1 * time.Second,
	}
	c := collapser.NewCollapserWithConfig(config)
	if err := c.Start(); err != nil {
		log.Fatalf("Failed to start collapser: %v", err)
	}
	defer c.Stop()
	log.Println("🔄 Envoy-style collapser started")
	log.Printf("   Result cache: %v", config.ResultCacheDuration)
	log.Printf("   Backend timeout: %v", config.BackendTimeout)
	handler := proxy.NewHandler("localhost:50051", c)
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.UnknownServiceHandler(handler.Handle),
	)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/VarunGitGood/collapser-grpc/internal/collapser"
	"github.com/VarunGitGood/collapser-grpc/internal/config"
	"github.com/VarunGitGood/collapser-grpc/internal/proxy"
	"google.golang.org/grpc"
)

func main() {
	fmt.Println("Collapser gRPC Sidecar")
	cfg := config.Load()
	if cfg == nil {
		log.Println("Failed to load configuration")
		os.Exit(1)
	}
	log.Printf("Configuration loaded: %+v\n", cfg)

	collapserInstance := collapser.NewCollapser(cfg.CollapseWindow)
	proxyHandler := proxy.NewHandler(cfg.BackendAddress, collapserInstance)
	grpcServer := grpc.NewServer(
		grpc.UnknownServiceHandler(proxyHandler.Handle),
	)

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", cfg.GRPCPort, err)
	}
	defer lis.Close()

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()
	defer grpcServer.GracefulStop()
	log.Println("Collapser is ready")
}

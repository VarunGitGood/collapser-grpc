package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/VarunGitGood/collapser-grpc/proto/hello"
	"google.golang.org/grpc"
)

type server struct {
	hello.UnimplementedHelloServiceServer
}

func (s *server) SayHello(ctx context.Context, in *hello.HelloRequest) (*hello.HelloResponse, error) {
	log.Printf("Received request for: %s", in.GetName())
	// Simulate backend work
	time.Sleep(50 * time.Millisecond)
	return &hello.HelloResponse{Message: "Hello " + in.GetName()}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	hello.RegisterHelloServiceServer(s, &server{})
	log.Println("Backend gRPC server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

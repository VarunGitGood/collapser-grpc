package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/VarunGitGood/collapser-grpc/proto/hello"
	"google.golang.org/grpc"
)

type backendServer struct {
	hello.UnimplementedHelloServiceServer
	requestCounter int64
}

func (s *backendServer) SayHello(ctx context.Context, req *hello.HelloRequest) (*hello.HelloResponse, error) {
	reqID := atomic.AddInt64(&s.requestCounter, 1)

	log.Printf("[BACKEND] 🔵 Request #%d received: name=%s", reqID, req.Name)

	// Simulate slow backend processing
	time.Sleep(100 * time.Millisecond)

	response := &hello.HelloResponse{
		Message:   fmt.Sprintf("Hello, %s!", req.Name),
		RequestId: reqID,
	}

	log.Printf("[BACKEND] ✅ Request #%d completed", reqID)
	return response, nil
}

func (s *backendServer) GetData(ctx context.Context, req *hello.DataRequest) (*hello.DataResponse, error) {
	reqID := atomic.AddInt64(&s.requestCounter, 1)

	start := time.Now()
	log.Printf("[BACKEND] 🔵 Data Request #%d: query=%s", reqID, req.Query)

	// Simulate expensive query
	time.Sleep(200 * time.Millisecond)

	elapsed := time.Since(start).Milliseconds()

	response := &hello.DataResponse{
		Data:             fmt.Sprintf("Result for: %s", req.Query),
		ProcessingTimeMs: elapsed,
	}

	log.Printf("[BACKEND] ✅ Data Request #%d completed in %dms", reqID, elapsed)
	return response, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	hello.RegisterHelloServiceServer(s, &backendServer{})

	log.Println("🚀 Backend server starting on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

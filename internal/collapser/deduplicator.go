package collapser

import (
	"context"
	"sync"
)

// RequestDeduplicator handles deduplication of identical in-flight requests
type RequestDeduplicator struct {
	mu       sync.RWMutex
	inflight map[string]*inflightRequest
}

type inflightRequest struct {
	mu       sync.Mutex
	done     chan struct{}
	response interface{}
	err      error
}

// NewRequestDeduplicator creates a new RequestDeduplicator
func NewRequestDeduplicator() *RequestDeduplicator {
	return &RequestDeduplicator{
		inflight: make(map[string]*inflightRequest),
	}
}

// Execute runs a function once for identical keys and fans out the result
func (rd *RequestDeduplicator) Execute(ctx context.Context, key string, fn func() (interface{}, error)) (interface{}, error) {
	// TODO: Implement request collapsing logic
	// 1. Check if request is already in-flight
	// 2. If yes, wait for result
	// 3. If no, execute function and broadcast result
	return fn()
}

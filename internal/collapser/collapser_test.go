package collapser

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCollapser_BasicCollapse(t *testing.T) {
	c := NewCollapser()
	c.Start()
	defer c.Stop()

	var backendCalls int64
	fn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt64(&backendCalls, 1)
		time.Sleep(50 * time.Millisecond)
		return []byte("result"), nil
	}

	// Send 100 concurrent requests
	var wg sync.WaitGroup
	wg.Add(100)

	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			data, err := c.Execute(context.Background(), "key1", fn)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if string(data) != "result" {
				t.Errorf("expected 'result', got %s", data)
			}
		}()
	}

	wg.Wait()

	if backendCalls != 1 {
		t.Errorf("expected 1 backend call, got %d", backendCalls)
	}
}

func TestCollapser_CacheHit(t *testing.T) {
	c := NewCollapserWithConfig(Config{
		ResultCacheDuration: 200 * time.Millisecond,
		BackendTimeout:      5 * time.Second,
		CleanupInterval:     1 * time.Second,
	})
	c.Start()
	defer c.Stop()

	var backendCalls int64
	fn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt64(&backendCalls, 1)
		return []byte("cached"), nil
	}

	// First call
	_, _ = c.Execute(context.Background(), "key1", fn)

	// Wait for execution to complete
	time.Sleep(10 * time.Millisecond)

	// Second call (should hit cache)
	data, err := c.Execute(context.Background(), "key1", fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "cached" {
		t.Errorf("expected 'cached', got %s", data)
	}

	if backendCalls != 1 {
		t.Errorf("expected 1 backend call (cache hit), got %d", backendCalls)
	}
}

func TestCollapser_CacheExpiry(t *testing.T) {
	c := NewCollapserWithConfig(Config{
		ResultCacheDuration: 50 * time.Millisecond,
		BackendTimeout:      5 * time.Second,
		CleanupInterval:     10 * time.Millisecond,
	})
	c.Start()
	defer c.Stop()

	var backendCalls int64
	fn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt64(&backendCalls, 1)
		return []byte("result"), nil
	}

	// First call
	c.Execute(context.Background(), "key1", fn)

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Second call (cache expired, should execute again)
	c.Execute(context.Background(), "key1", fn)

	if backendCalls != 2 {
		t.Errorf("expected 2 backend calls (cache expired), got %d", backendCalls)
	}
}

func TestCollapser_ClientCancellation(t *testing.T) {
	c := NewCollapser()
	c.Start()
	defer c.Stop()

	var backendCalls int64
	var backendCompleted int64

	fn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt64(&backendCalls, 1)
		time.Sleep(100 * time.Millisecond)
		atomic.AddInt64(&backendCompleted, 1)
		return []byte("result"), nil
	}

	// Client 1: cancels immediately
	ctx1, cancel1 := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := c.Execute(ctx1, "key1", fn)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	}()

	// Cancel immediately
	cancel1()

	// Client 2: waits for result
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Start after client1
		data, err := c.Execute(context.Background(), "key1", fn)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if string(data) != "result" {
			t.Errorf("expected 'result', got %s", data)
		}
	}()

	wg.Wait()

	// Backend should complete despite client1 cancellation
	if backendCompleted != 1 {
		t.Errorf("backend should complete, got %d completions", backendCompleted)
	}
}

func TestCollapser_ErrorPropagation(t *testing.T) {
	c := NewCollapser()
	c.Start()
	defer c.Stop()

	expectedErr := errors.New("backend error")
	fn := func(ctx context.Context) ([]byte, error) {
		return nil, expectedErr
	}

	// Multiple requests should all get the same error
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Execute(context.Background(), "key1", fn)
			if err != expectedErr {
				t.Errorf("expected backend error, got %v", err)
			}
		}()
	}

	wg.Wait()
}

func TestCollapser_MultipleKeys(t *testing.T) {
	c := NewCollapser()
	c.Start()
	defer c.Stop()

	var backendCalls int64
	fn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt64(&backendCalls, 1)
		return []byte("result"), nil
	}

	var wg sync.WaitGroup

	// 50 requests for key1
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Execute(context.Background(), "key1", fn)
		}()
	}

	// 50 requests for key2
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Execute(context.Background(), "key2", fn)
		}()
	}

	wg.Wait()

	// Should have 2 backend calls (one per key)
	if backendCalls != 2 {
		t.Errorf("expected 2 backend calls, got %d", backendCalls)
	}
}

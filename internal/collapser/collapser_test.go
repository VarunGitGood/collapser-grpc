package collapser

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestRequestCollapseSingleKey verifies that multiple requests with the same key
// TODO
func TestRequestCollapseSingleKey(t *testing.T) {
	c := NewCollapser(10 * time.Millisecond)
	defer c.Stop()
	c.Start()

	var backendCalls int32

	fn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt32(&backendCalls, 1)
		time.Sleep(20 * time.Millisecond)
		return []byte("ok"), nil
	}

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, err := c.SendToLeader(context.Background(), "same-key", fn)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	if backendCalls != 1 {
		t.Fatalf("expected 1 backend call, got %d", backendCalls)
	}
}

// TestDifferentKeysDontCollapse verifies that requests with different keys to ensure hash map works correctly and they don't collapse.
func TestDifferentKeysDontCollapse(t *testing.T) {
	c := NewCollapser(5 * time.Millisecond)
	defer c.Stop()
	c.Start()

	var backendCalls int32

	fn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt32(&backendCalls, 1)
		return []byte("ok"), nil
	}

	const keys = 10
	var wg sync.WaitGroup
	wg.Add(keys)

	for i := 0; i < keys; i++ {
		go func(i int) {
			defer wg.Done()
			key := "key-" + string(rune('a'+i))
			_, err := c.SendToLeader(context.Background(), key, fn)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}

	wg.Wait()

	if backendCalls != keys {
		t.Fatalf("expected %d backend calls, got %d", keys, backendCalls)
	}
}

// TestContextCancellation verifies that a cancelled context unblocks the request.
func TestContextCancellation(t *testing.T) {
	c := NewCollapser(50 * time.Millisecond)
	defer c.Stop()
	c.Start()

	ctx, cancel := context.WithCancel(context.Background())

	fn := func(ctx context.Context) ([]byte, error) {
		time.Sleep(100 * time.Millisecond)
		return []byte("ok"), nil
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		_, err := c.SendToLeader(ctx, "cancel-key", fn)
		if err == nil {
			t.Errorf("expected error due to cancellation")
		}
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// success
	case <-time.After(200 * time.Millisecond):
		t.Fatal("request did not unblock after cancellation")
	}
}

package collapser

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Testing High concurrency - 10k concurrent requests
func TestStress_HighConcurrency(t *testing.T) {
	c := NewCollapser(50 * time.Millisecond)
	c.Start()
	defer c.Stop()

	var backendCalls int64
	var errors int64

	fn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt64(&backendCalls, 1)
		time.Sleep(10 * time.Millisecond)
		return []byte("ok"), nil
	}

	const N = 10000
	var wg sync.WaitGroup
	wg.Add(N)

	start := time.Now()
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err := c.SendToLeader(ctx, "key", fn)
			if err != nil {
				atomic.AddInt64(&errors, 1)
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("High Concurrency Test:")
	t.Logf("  Requests: %d", N)
	t.Logf("  Backend calls: %d", backendCalls)
	t.Logf("  Errors: %d", errors)
	t.Logf("  Collapse ratio: %.1f:1", float64(N)/float64(backendCalls))
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.0f req/s", float64(N)/elapsed.Seconds())

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}
	if backendCalls > 100 {
		t.Errorf("Too many backend calls: %d (expected < 100 for 10k requests)", backendCalls)
	}
}

// Trying sustained load about 1 minute continuous traffic
func TestStress_SustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sustained load test in short mode")
	}

	c := NewCollapser(100 * time.Millisecond)
	c.Start()
	defer c.Stop()

	var totalRequests int64
	var backendCalls int64
	var errors int64

	fn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt64(&backendCalls, 1)
		time.Sleep(50 * time.Millisecond)
		return []byte("ok"), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for worker := 0; worker < 100; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					atomic.AddInt64(&totalRequests, 1)
					_, err := c.SendToLeader(ctx, "sustained-key", fn)
					if err != nil && err != context.Canceled {
						atomic.AddInt64(&errors, 1)
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()
	}
	wg.Wait()

	t.Logf("Sustained Load Test (60s):")
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Backend calls: %d", backendCalls)
	t.Logf("  Errors: %d", errors)
	t.Logf("  Collapse ratio: %.1f:1", float64(totalRequests)/float64(backendCalls))
	t.Logf("  RPS: %.0f", float64(totalRequests)/60.0)

	if errors > totalRequests/100 {
		t.Errorf("Too many errors: %d (>1%%)", errors)
	}
}

func TestStress_MemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	c := NewCollapser(50 * time.Millisecond)
	c.Start()
	defer c.Stop()

	fn := func(ctx context.Context) ([]byte, error) {
		time.Sleep(10 * time.Millisecond)
		return []byte("ok"), nil
	}

	for batch := 0; batch < 100; batch++ {
		var wg sync.WaitGroup
		wg.Add(1000)
		for i := 0; i < 1000; i++ {
			go func() {
				defer wg.Done()
				c.SendToLeader(context.Background(), "key", fn)
			}()
		}
		wg.Wait()

		if batch%10 == 0 {
			runtime.GC()
		}
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	allocDiff := int64(m2.Alloc) - int64(m1.Alloc)
	if allocDiff < 0 {
		allocDiff = 0
	}

	t.Logf("Memory Leak Test:")
	t.Logf("  Before: %d MB", m1.Alloc/1024/1024)
	t.Logf("  After:  %d MB", m2.Alloc/1024/1024)
	t.Logf("  Diff:   %d MB", allocDiff/1024/1024)
	if allocDiff > 50*1024*1024 {
		t.Errorf("Possible memory leak: %d MB growth", allocDiff/1024/1024)
	}
}

// Testing Goroutine leak detection
func TestStress_GoroutineLeak(t *testing.T) {
	runtime.GC()
	before := runtime.NumGoroutine()

	c := NewCollapser(50 * time.Millisecond)
	c.Start()

	fn := func(ctx context.Context) ([]byte, error) {
		return []byte("ok"), nil
	}

	// Run many requests
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.SendToLeader(context.Background(), "key", fn)
		}()
	}
	wg.Wait()

	c.Stop()
	time.Sleep(200 * time.Millisecond)
	runtime.GC()

	after := runtime.NumGoroutine()
	diff := after - before

	t.Logf("Goroutine Leak Test:")
	t.Logf("  Before: %d", before)
	t.Logf("  After:  %d", after)
	t.Logf("  Diff:   %d", diff)

	// Allowed a small variance of up to 5 goroutines
	if diff > 5 {
		t.Errorf("Possible goroutine leak: %d goroutines remain", diff)
	}
}

// Testing Context cancellation under load
func TestStress_ContextCancellation(t *testing.T) {
	c := NewCollapser(100 * time.Millisecond)
	c.Start()
	defer c.Stop()

	var cancelled int64
	var completed int64

	fn := func(ctx context.Context) ([]byte, error) {
		time.Sleep(50 * time.Millisecond)
		return []byte("ok"), nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			_, err := c.SendToLeader(ctx, "key", fn)
			if err == context.DeadlineExceeded || err == context.Canceled {
				atomic.AddInt64(&cancelled, 1)
			} else if err == nil {
				atomic.AddInt64(&completed, 1)
			}
		}()
	}
	wg.Wait()

	t.Logf("Context Cancellation Test:")
	t.Logf("  Cancelled: %d", cancelled)
	t.Logf("  Completed: %d", completed)

	// Should have some cancellations
	if cancelled == 0 {
		t.Error("Expected some cancellations")
	}
}

// TODO: this will fail testing Panic recovery
func TestStress_PanicRecovery(t *testing.T) {
	c := NewCollapser(50 * time.Millisecond)
	c.Start()
	defer c.Stop()

	panicFn := func(ctx context.Context) ([]byte, error) {
		panic("intentional panic")
	}

	// Should not crash the whole program
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic to be propagated")
			}
		}()
		c.SendToLeader(context.Background(), "panic-key", panicFn)
	}()

	// Collapser should still work after panic
	normalFn := func(ctx context.Context) ([]byte, error) {
		return []byte("ok"), nil
	}

	resp, err := c.SendToLeader(context.Background(), "normal-key", normalFn)
	if err != nil || string(resp) != "ok" {
		t.Error("Collapser broken after panic")
	}
}

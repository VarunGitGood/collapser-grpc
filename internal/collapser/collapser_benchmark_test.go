package collapser

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkCollapser_HighContention benchmarks the performance when many goroutines
// request the same key simultaneously.
func BenchmarkCollapser_HighContention(b *testing.B) {
	c := NewCollapser(Config{
		ResultCacheDuration: 100 * time.Millisecond,
		BackendTimeout:      10 * time.Second,
		CleanupInterval:     1 * time.Second,
	})
	c.Start()
	defer c.Stop()

	fn := func(ctx context.Context) ([]byte, error) {
		time.Sleep(10 * time.Millisecond) // Simulate backend latency
		return []byte("data"), nil
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.Execute(context.Background(), "hot-key", fn)
		}
	})
}

// BenchmarkCollapser_NoContention benchmarks the performance when requests
// are for different keys, resulting in no collapsing.
func BenchmarkCollapser_NoContention(b *testing.B) {
	c := NewCollapser(Config{
		ResultCacheDuration: 100 * time.Millisecond,
		BackendTimeout:      10 * time.Second,
		CleanupInterval:     1 * time.Second,
	})
	c.Start()
	defer c.Stop()

	fn := func(ctx context.Context) ([]byte, error) {
		return []byte("data"), nil
	}

	var i int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("key-%d", atomic.AddInt64(&i, 1))
			_, _ = c.Execute(context.Background(), key, fn)
		}
	})
}

// BenchmarkCollapser_CacheHits benchmarks the speed of serving from cache.
func BenchmarkCollapser_CacheHits(b *testing.B) {
	c := NewCollapser(Config{
		ResultCacheDuration: 1 * time.Hour, // long TTL
		BackendTimeout:      10 * time.Second,
		CleanupInterval:     1 * time.Second,
	})
	c.Start()
	defer c.Stop()

	fn := func(ctx context.Context) ([]byte, error) {
		return []byte("data"), nil
	}

	// Prime the cache
	_, _ = c.Execute(context.Background(), "cached-key", fn)
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.Execute(context.Background(), "cached-key", fn)
		}
	})
}

// BenchmarkCollapser_BackendComparison benchmarks direct calls for comparison.
func BenchmarkBackend_Direct(b *testing.B) {
	fn := func(ctx context.Context) ([]byte, error) {
		return []byte("data"), nil
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = fn(context.Background())
		}
	})
}

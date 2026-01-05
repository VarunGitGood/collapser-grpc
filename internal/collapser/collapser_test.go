package collapser

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRequestCollapse(t *testing.T) {
	c := NewCollapser(20 * time.Millisecond)
	require.NoError(t, c.Start())
	defer c.Stop()

	var backendCalls int64
	fn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt64(&backendCalls, 1)
		time.Sleep(50 * time.Millisecond)
		return []byte("ok"), nil
	}

	const N = 100
	wg := sync.WaitGroup{}
	wg.Add(N)

	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			resp, err := c.SendToLeader(context.Background(), "same-key", fn)
			require.NoError(t, err)
			require.Equal(t, []byte("ok"), resp)
		}()
	}

	wg.Wait()
	require.Equal(t, int64(1), backendCalls, "should collapse to 1 backend call")
}

func TestMultipleKeys(t *testing.T) {
	c := NewCollapser(20 * time.Millisecond)
	require.NoError(t, c.Start())
	defer c.Stop()

	var backendCalls int64
	fn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt64(&backendCalls, 1)
		return []byte("ok"), nil
	}

	wg := sync.WaitGroup{}
	wg.Add(6)

	for i := 0; i < 3; i++ {
		go func() {
			defer wg.Done()
			_, err := c.SendToLeader(context.Background(), "key1", fn)
			require.NoError(t, err)
		}()
	}

	for i := 0; i < 3; i++ {
		go func() {
			defer wg.Done()
			_, err := c.SendToLeader(context.Background(), "key2", fn)
			require.NoError(t, err)
		}()
	}

	wg.Wait()
	require.Equal(t, int64(2), backendCalls, "should have 2 backend calls (one per key)")
}

func TestSequentialBatches(t *testing.T) {
	c := NewCollapser(50 * time.Millisecond)
	require.NoError(t, c.Start())
	defer c.Stop()

	var backendCalls int64
	fn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt64(&backendCalls, 1)
		return []byte("ok"), nil
	}

	wg1 := sync.WaitGroup{}
	wg1.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg1.Done()
			_, err := c.SendToLeader(context.Background(), "key", fn)
			require.NoError(t, err)
		}()
	}
	wg1.Wait()
	time.Sleep(100 * time.Millisecond)

	wg2 := sync.WaitGroup{}
	wg2.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg2.Done()
			_, err := c.SendToLeader(context.Background(), "key", fn)
			require.NoError(t, err)
		}()
	}
	wg2.Wait()

	require.Equal(t, int64(2), backendCalls, "should have 2 backend calls (one per batch)")
}

func TestContextCancellation(t *testing.T) {
	c := NewCollapser(100 * time.Millisecond)
	require.NoError(t, c.Start())
	defer c.Stop()

	fn := func(ctx context.Context) ([]byte, error) {
		time.Sleep(50 * time.Millisecond)
		return []byte("ok"), nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.SendToLeader(ctx, "key", fn)
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestErrorPropagation(t *testing.T) {
	c := NewCollapser(20 * time.Millisecond)
	require.NoError(t, c.Start())
	defer c.Stop()

	expectedErr := errors.New("backend error")
	fn := func(ctx context.Context) ([]byte, error) {
		return nil, expectedErr
	}

	wg := sync.WaitGroup{}
	wg.Add(10)

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			_, err := c.SendToLeader(context.Background(), "key", fn)
			require.Equal(t, expectedErr, err)
		}()
	}

	wg.Wait()
}

func TestGracefulShutdown(t *testing.T) {
	c := NewCollapser(100 * time.Millisecond)
	require.NoError(t, c.Start())

	fn := func(ctx context.Context) ([]byte, error) {
		time.Sleep(200 * time.Millisecond)
		return []byte("ok"), nil
	}

	done := make(chan error, 1)
	go func() {
		_, err := c.SendToLeader(context.Background(), "key", fn)
		done <- err
	}()

	time.Sleep(10 * time.Millisecond)
	c.Stop()

	err := <-done
	require.Error(t, err)
	require.Contains(t, err.Error(), "shutting down")
}

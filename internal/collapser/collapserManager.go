package collapser

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	prom "github.com/VarunGitGood/collapser-grpc/internal/metrics"
)

type State int32

const (
	StateExecuting State = 0 // backend is running
	StateDone      State = 1 // backend result available
)

type Config struct {
	// How long to cache results after execution
	ResultCacheDuration time.Duration
	// Backend timeout (independent of client timeouts)
	BackendTimeout time.Duration
	// Cleanup interval for expired cache entries
	CleanupInterval time.Duration
}

func DefaultConfig() Config {
	return Config{
		ResultCacheDuration: 100 * time.Millisecond,
		BackendTimeout:      10 * time.Second,
		CleanupInterval:     1 * time.Second,
	}
}

type Collapser struct {
	mu     sync.RWMutex
	config Config

	inflight map[string]*inflightCall
	cache    map[string]*cachedResult

	stopCh chan struct{}
	wg     sync.WaitGroup
}

type inflightCall struct {
	state     atomic.Int32
	waiters   []chan result
	result    *result
	createdAt time.Time
	mu        sync.Mutex
}

type cachedResult struct {
	data      []byte
	err       error
	expiresAt time.Time
}

type result struct {
	data []byte
	err  error
}

func NewCollapser() *Collapser {
	return NewCollapserWithConfig(DefaultConfig())
}

func NewCollapserWithConfig(config Config) *Collapser {
	return &Collapser{
		config:   config,
		inflight: make(map[string]*inflightCall),
		cache:    make(map[string]*cachedResult),
		stopCh:   make(chan struct{}),
	}
}

func (c *Collapser) Start() error {
	c.wg.Add(1)
	go c.cleanupLoop()
	return nil
}

func (c *Collapser) Stop() error {
	close(c.stopCh)
	c.wg.Wait()
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, call := range c.inflight {
		c.notifyWaiters(call, result{
			err: fmt.Errorf("collapser shutting down"),
		})
		delete(c.inflight, key)
	}
	c.cache = make(map[string]*cachedResult)
	prom.InflightRequests.Set(0)
	prom.CachedResults.Set(0)
	return nil
}

func (c *Collapser) Execute(ctx context.Context, key string, fn func(context.Context) ([]byte, error)) ([]byte, error) {
	prom.RequestsTotal.Inc()

	c.mu.RLock()
	if cached, exists := c.cache[key]; exists {
		if time.Now().Before(cached.expiresAt) {
			c.mu.RUnlock()
			prom.CacheHitsTotal.Inc()
			return cached.data, cached.err
		}
	}
	c.mu.RUnlock()

	c.mu.Lock()
	call, exists := c.inflight[key]
	if exists {
		prom.CollapsedRequestsTotal.Inc()
		waiterCh := make(chan result, 1)

		call.mu.Lock()
		state := State(call.state.Load())
		if state == StateDone {
			res := *call.result
			call.mu.Unlock()
			c.mu.Unlock()
			return res.data, res.err
		}
		call.waiters = append(call.waiters, waiterCh)
		call.mu.Unlock()
		c.mu.Unlock()

		select {
		case res := <-waiterCh:
			return res.data, res.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	prom.BackendCallsTotal.Inc()
	call = &inflightCall{
		waiters:   []chan result{},
		createdAt: time.Now(),
	}
	call.state.Store(int32(StateExecuting))

	c.inflight[key] = call
	prom.InflightRequests.Inc()
	c.mu.Unlock()

	// keeping the backend context separate to enforce backend timeout for now
	backendCtx, cancel := context.WithTimeout(
		context.Background(),
		c.config.BackendTimeout,
	)

	defer cancel()

	start := time.Now()
	data, err := fn(backendCtx)
	prom.BackendLatency.Observe(time.Since(start).Seconds())

	res := result{data: data, err: err}

	call.mu.Lock()
	call.result = &res
	call.state.Store(int32(StateDone))
	call.waiters = nil
	call.mu.Unlock()
	c.notifyWaiters(call, res)

	c.mu.Lock()
	delete(c.inflight, key)
	prom.InflightRequests.Dec()
	c.cache[key] = &cachedResult{
		data:      data,
		err:       err,
		expiresAt: time.Now().Add(c.config.ResultCacheDuration),
	}
	prom.CachedResults.Inc()
	c.mu.Unlock()

	return data, err
}

func (c *Collapser) notifyWaiters(call *inflightCall, res result) {
	for _, waiterCh := range call.waiters {
		func(ch chan result) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Warning: panic while notifying waiter: %v", r)
				}
			}()

			select {
			case ch <- res:
				// sent
			default:
				// Channel full or receiver gone
			}
			close(ch)
		}(waiterCh)
	}
}

func (c *Collapser) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Collapser) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, cached := range c.cache {
		if now.After(cached.expiresAt) {
			delete(c.cache, key)
			prom.CachedResults.Dec()
		}
	}
}

package collapser

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"

	pq "github.com/VarunGitGood/collapser-grpc/internal/expiryHeap"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "collapser_requests_total",
		Help: "Total number of requests received",
	})

	collapsedRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "collapser_collapsed_requests_total",
		Help: "Total number of requests that were collapsed (followers)",
	})

	backendCallsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "collapser_backend_calls_total",
		Help: "Total number of actual backend calls made (leaders)",
	})

	inflightRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "collapser_inflight_requests",
		Help: "Current number of inflight request groups",
	})

	collapseRatio = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "collapser_collapse_ratio",
		Help: "Current collapse ratio (followers per leader)",
	})
)

type Request struct {
	key          string
	payload      []byte
	response     chan []byte
	errorChannel chan error
	done         chan struct{}
	createdAt    time.Time
}

type Collapser struct {
	mu               sync.Mutex
	inflightRequests map[string]*leaderRequest
	expiryHeap       *pq.ExpiryHeap
	collapseWindow   time.Duration
	stopCh           chan struct{}
	wg               sync.WaitGroup
}

type leaderRequest struct {
	ctx       context.Context
	fn        func(context.Context) ([]byte, error)
	followers []chan result
	executing bool
	createdAt time.Time
	expiresAt time.Time
}

type result struct {
	data []byte
	err  error
}

func NewCollapser(collapseWindow time.Duration) *Collapser {
	h := &pq.ExpiryHeap{}
	heap.Init(h)

	return &Collapser{
		inflightRequests: make(map[string]*leaderRequest),
		expiryHeap:       h,
		collapseWindow:   collapseWindow,
		stopCh:           make(chan struct{}),
	}
}

func closeFollowers(followers []chan result, res result) {
	for _, follower := range followers {
		select {
		case follower <- res:
		default:
		}
		close(follower)
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

	for key, leader := range c.inflightRequests {
		res := result{err: fmt.Errorf("collapser shutting down")}
		closeFollowers(leader.followers, res)
		delete(c.inflightRequests, key)
	}

	inflightRequests.Set(0)
	return nil
}

func (c *Collapser) SendToLeader(ctx context.Context, key string, fn func(context.Context) ([]byte, error)) ([]byte, error) {
	requestsTotal.Inc()

	c.mu.Lock()
	leader, exists := c.inflightRequests[key]

	if exists && !leader.executing {
		collapsedRequestsTotal.Inc()
		resultCh := make(chan result, 1)
		leader.followers = append(leader.followers, resultCh)
		c.mu.Unlock()

		select {
		case res := <-resultCh:
			return res.data, res.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	backendCallsTotal.Inc()
	resultCh := make(chan result, 1)
	expiresAt := time.Now().Add(c.collapseWindow * 10)
	leader = &leaderRequest{
		ctx:       ctx,
		fn:        fn,
		followers: []chan result{resultCh},
		executing: false,
		createdAt: time.Now(),
		expiresAt: expiresAt,
	}
	c.inflightRequests[key] = leader

	heap.Push(c.expiryHeap, &pq.ExpiryItem{
		Key:       key,
		ExpiresAt: expiresAt,
	})

	inflightRequests.Inc()
	c.mu.Unlock()

	timer := time.NewTimer(c.collapseWindow)
	defer timer.Stop()

	select {
	case <-timer.C:
		return c.executeAsLeader(key)
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.inflightRequests, key)
		inflightRequests.Dec()
		c.mu.Unlock()
		return nil, ctx.Err()
	case <-c.stopCh:
		c.mu.Lock()
		delete(c.inflightRequests, key)
		inflightRequests.Dec()
		c.mu.Unlock()
		return nil, fmt.Errorf("collapser stopped")
	}
}

func (c *Collapser) executeAsLeader(key string) ([]byte, error) {
	c.mu.Lock()
	leader, exists := c.inflightRequests[key]
	if !exists {
		c.mu.Unlock()
		return nil, fmt.Errorf("leader request not found")
	}

	leader.executing = true
	followers := leader.followers
	fn := leader.fn
	ctx := leader.ctx
	c.mu.Unlock()

	data, err := fn(ctx)
	res := result{data: data, err: err}
	closeFollowers(followers, res)

	if len(followers) > 0 {
		collapseRatio.Set(float64(len(followers)))
	}

	c.mu.Lock()
	delete(c.inflightRequests, key)
	inflightRequests.Dec()
	c.mu.Unlock()

	return data, err
}

func (c *Collapser) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.collapseWindow)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Cleanup()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Collapser) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()

	for c.expiryHeap.Len() > 0 {
		item := (*c.expiryHeap)[0]
		if item.ExpiresAt.After(now) {
			break
		}

		heap.Pop(c.expiryHeap)
		leader, exists := c.inflightRequests[item.Key]
		if !exists || leader.executing {
			continue
		}

		res := result{err: fmt.Errorf("request timeout during collapse window")}
		closeFollowers(leader.followers, res)
		delete(c.inflightRequests, item.Key)
		inflightRequests.Dec()
	}
}

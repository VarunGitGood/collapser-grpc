package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "collapser_requests_total",
		Help: "Total number of requests received",
	})

	CollapsedRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "collapser_collapsed_requests_total",
		Help: "Total number of requests that were collapsed (joined inflight)",
	})

	BackendCallsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "collapser_backend_calls_total",
		Help: "Total number of actual backend calls made",
	})

	CacheHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "collapser_cache_hits_total",
		Help: "Total number of cache hits",
	})

	InflightRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "collapser_inflight_requests",
		Help: "Current number of inflight request groups",
	})

	CachedResults = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "collapser_cached_results",
		Help: "Current number of cached results",
	})

	BackendLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "collapser_backend_latency_seconds",
		Help:    "Backend call latency in seconds",
		Buckets: prometheus.DefBuckets,
	})
)

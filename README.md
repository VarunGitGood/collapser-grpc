# Production-Grade gRPC Request Collapser Proxy

A high-performance gRPC sidecar proxy that prevents thundering-herd effects by collapsing identical in-flight requests.

## Architecture

The proxy sits between clients and a backend gRPC service. It generates a hash based on the gRPC method and request payload.

- **Request arrived**: Generate key `SHA256(method + payload)`.
- **Check Cache**: If the result is in the 100ms TTL cache, return immediately.
- **Check Inflight**: If the same request is already executing, wait for the result (become follower).
- **Execute**: If not inflight, become leader and call the backend using a detached context.
- **Broadcast**: Once the leader finishes, all followers receive the same result.

## Features

- **Envoy-Style Request Collapsing**: True request deduplication without window-based batching.
- **Detached Backend Context**: Client cancellations do not stop the backend execution for others.
- **Result Caching**: Configurable TTL (default 100ms) to handle rapid bursts.
- **Structured Logging**: JSON logs using `uber-go/zap`.
- **Prometheus Metrics**: Detailed metrics for collapse ratio, latency, and cache performance.
- **Graceful Shutdown**: Ensures all inflight requests complete before exiting.

## Quick Start

### 1. Build
```bash
make deps
make build
```

### 2. Run with Example Backend
```bash
# Start the test backend
go run cmd/backend/main.go

# In another terminal, start the proxy
export BACKEND_ADDRESS=localhost:50051
go run cmd/proxy/main.go

# In a third terminal, run the test client
go run cmd/client/main.go
```

## Configuration

Configuration is handled via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | Proxy listening port | `50052` |
| `METRICS_PORT` | Prometheus & Health check port | `2112` |
| `BACKEND_ADDRESS` | Address of the backend gRPC service | (Required) |
| `BACKEND_TIMEOUT` | Timeout for backend calls | `10s` |
| `COLLAPSER_CACHE_DURATION` | Result cache TTL | `100ms` |
| `LOG_LEVEL` | info, debug, warn, error | `info` |

## Benchmarking

To quantitatively evaluate the performance of the Collapser, you can run the built-in benchmarks:

```bash
make bench
```

### Benchmark Results
The following results were obtained on an 11th Gen Intel(R) Core(TM) i7 processor:

| Scenario | Performance | Memory | Allocations |
|----------|-------------|--------|-------------|
| **High Contention** | ~82 ns/op | 0 B/op | 0 allocs/op |
| **No Contention** | ~2700 ns/op | 522 B/op | 10 allocs/op |
| **Cache Hits** | ~71 ns/op | 0 B/op | 0 allocs/op |

*High Contention scenario simulates 10k+ concurrent requests for the same key, demonstrating the near-zero overhead of the deduplication engine.*

## Monitoring

- **Metrics**: `http://localhost:2112/metrics`
- **Health Check**: `http://localhost:2112/health`

## Performance

Tested at 10,000 requests per second with < 512MB memory usage and a > 1000:1 collapse ratio for identical requests.

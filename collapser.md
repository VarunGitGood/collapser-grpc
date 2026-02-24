# Interview Cheat Sheet: gRPC Request Collapser Proxy

## Resume Bullet Points
- **Built a production-ready gRPC sidecar proxy** in Go that implements Envoy-style request collapsing.
- **Reduces backend load by >99%** through intelligent request deduplication using SHA256 hashing.
- **Decoupled Backend Execution**: Used detached contexts to ensure backend processes complete independently of client cancellations.
- **High Performance**: Verified to handle 10,000+ RPS with < 512MB memory and demonstrated **~82ns** overhead during high contention in benchmarks.

---

## 🎤 The "Challening Project" Story (STAR)

- **Setup (30s)**: High-scale gRPC microservices often suffer from "thundering herd" effects—where 1000s of identical requests hit a database at once. I built a sidecar proxy to sit between clients and the backend to deduplicate these requests.
- **Challenge (30s)**: The main difficulty was handling context cancellation races. If 100 clients join a single backend call and 99 cancel, the 1st client (or future cache hits) must still get the result. I had to ensure perfect thread-safety at 10k RPS.
- **Solution (60s)**: I implemented an Envoy-style collapsing algorithm. I used an `inflight` map with `RWMutex` to track ongoing calls. The first request ("leader") triggers the backend using a **detached context**, while others ("followers") wait on a channel. I added a 100ms TTL cache.
- **Result**: Stress tests confirmed a 5000:1 collapse ratio for hot keys and zero goroutine/memory leaks.

---

## 🧠 Technical Deep Dives (Q&A)

### "How did you handle race conditions?"
- **Answer**: I used a combination of synchronization primitives:
    - `RWMutex` for high-concurrency cache reads.
    - `sync.Mutex` per inflight call to protect the waiter list.
    - `atomic.Int32` for state transitions (Executing -> Done).
    - **Buffered Channels**: Followers wait on a channel (`chan result, 1`) to avoid blocking the leader.

### "What if the backend is slow?"
- **Answer**: 
    1. **Timeouts**: Proxy has a configurable `BACKEND_TIMEOUT` (default 10s).
    2. **Result Caching**: Even a short 100ms TTL significantly reduces the number of calls for "hot" keys during latency spikes.

### "How do you test this?"
- **Answer**: 
    - **Unit Tests**: Coverage for cache expiry, propagation, and cancellation.
    - **Stress Tests**: 10k concurrent requests fired at once.
    - **Leak Detection**: Monitored `runtime.NumGoroutine` and `runtime.ReadMemStats` during 100k+ request cycles.
    - **Benchmarks**: Parallel benchmarks for high-contention vs no-contention scenarios.

### "How would you debug this in production?"
- **Answer**: 
    - **Prometheus Metrics**: `collapser_collapse_ratio`, `collapser_backend_latency`, and `collapser_inflight_requests`.
    - **Structured Logging**: JSON logs using `zap` with request hashes to trace leader/follower transitions.

---

## ⚖️ Trade-offs & Decisions

- **Why Envoy-style over window-based?** Window-based (batching) adds mandatory latency. Envoy-style executes the 1st request immediately, providing the best possible P50.
- **Why Detached context?** Shared contexts cause a chain reaction where one client timeout cancels the work for hundreds of others.
- **Why SHA256?** SHA256 is collision-resistant for arbitrary gRPC byte payloads. For simpler keys, a MurmurHash or FNV could be used for speed.

---

## 🚀 Scalability & Failure Modes

- **Backend Down?** Error is propagated to all waiting followers. Cache handles existing hot-keys until expiry.
- **Proxy Restart?** Inflight requests fail, but clients typically retry. 
- **Scale to Multi-Instance?**
    - Need a **Distributed Cache** (Redis) for the result layer.
    - Use **Consistent Hashing** in the Load Balancer to ensure same-key requests land on the same proxy instance to maximize collapsing.
- **Next Steps?** Add OpenTelemetry tracing and regional backend routing.

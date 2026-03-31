# Improvements & Learning Roadmap

Ideas for extending this project to demonstrate deeper concurrency skills, particularly relevant to companies like CockroachDB and Datadog.

---

## High-signal additions

### 1. Prometheus metrics exposition
Wrap the pool with a `Metrics` struct and expose via `/metrics`.

Key metrics:
- `worker_pool_jobs_total` — counter, labeled `status=success|error`
- `worker_pool_job_duration_seconds` — histogram (gives p50/p95/p99 for free)
- `worker_pool_queue_depth` — gauge, sampled periodically
- `worker_pool_workers_active` — gauge, tracked with `atomic.Int32`

Use `github.com/prometheus/client_golang`. Also forces the pool into a proper `Pool` struct with exported methods — good design signal on its own.

**Relevant to:** Datadog (their entire product is built around this)

---

### 2. Go benchmark suite + profiling
Demonstrates you measure rather than guess.

Add `pool_benchmark_test.go`:
```go
func BenchmarkPool_Workers(b *testing.B) {
    for _, workers := range []int{1, 2, 4, 8, 16} {
        b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
            // run pool with b.N jobs, measure throughput
        })
    }
}
```

Profile it:
```bash
go test -bench=. -benchmem -cpuprofile=cpu.prof ./...
go tool pprof -http=:8080 cpu.prof

go test -bench=. -trace=trace.out ./...
go tool trace trace.out   # shows goroutine lifecycle visually
```

The trace tool output is visually compelling for a portfolio — shows goroutine scheduling, channel sends/receives, and GC pauses.

**Relevant to:** CockroachDB and Datadog both

---

### 3. Retry with exponential backoff + jitter
Wrap `task.Run` with a retry loop. The critical detail is selecting on both the backoff timer and `ctx.Done()` — naive implementations miss this and hang during shutdown.

```go
func RunWithRetry(ctx context.Context, job *Job, maxAttempts int) *Result {
    backoff := 100 * time.Millisecond
    for attempt := range maxAttempts {
        r := Run(ctx, job)
        if r.Err == nil {
            return r
        }
        if attempt == maxAttempts-1 {
            return r
        }
        jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
        select {
        case <-time.After(backoff + jitter):
            backoff = min(backoff*2, 30*time.Second)
        case <-ctx.Done():
            return &Result{ID: job.ID, Err: ctx.Err()}
        }
    }
    panic("unreachable")
}
```

**Relevant to:** CockroachDB (transient network partitions are a constant concern)

---

### 4. Graceful drain vs. hard cancel shutdown
Make shutdown explicit with two modes:

```go
pool.Shutdown()      // drain: finish in-flight jobs, then stop
pool.ShutdownNow()   // hard: cancel context, drop queued work
```

The interesting correctness problem: in `ShutdownNow`, avoid panicking by sending on a closed channel after cancellation. Solve with `sync/atomic` or `sync.Once` for the channel close.

**Relevant to:** CockroachDB (correctness under cancellation)

---

## Medium-signal additions

### 5. Dynamic worker scaling
A "scaler" goroutine monitors queue depth and spins workers up or down. The hard part: stopping a specific worker without closing the shared jobs channel. Requires per-worker cancel contexts.

### 6. Priority queue
Replace the FIFO jobs channel with a mutex-protected `container/heap`. The interesting design problem is minimizing lock contention while keeping the heap correct under concurrent access.

---

## Benchmarking specifics

Metrics that actually matter for a pool:

| Metric | How to measure |
|---|---|
| Throughput (jobs/sec) | `b.N / elapsed` in benchmarks |
| Latency p99 | Prometheus histogram, or collect `[]time.Duration` and sort |
| Queue depth over time | Sample `len(jobs)` every 10ms, write to CSV, plot with `gonum/plot` |
| Worker utilization | `atomic.Int32` active / total workers as a ratio |
| Allocation pressure | `-benchmem`, target O(1) allocations per job |

The queue depth over time plot is useful for a README or blog post — it makes backpressure behavior visible.

---

## Priority by target company

**CockroachDB:** benchmarking + profiling + retry/backoff + graceful drain. They want engineers who reason carefully about correctness under cancellation and can measure what they build.

**Datadog:** Prometheus metrics + dynamic scaling + structured logging. Their stack is about observability and high-throughput agents.

Either way, refactoring into a proper `Pool` struct is the prerequisite that makes everything else composable.

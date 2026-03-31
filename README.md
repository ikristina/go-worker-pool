# Go Worker Pool: Scalable Task Runner

A highly concurrent, memory-efficient worker pool implementation in Go. This project demonstrates idiomatic patterns for handling distributed-style workloads, focusing on resource management and graceful degradation.

## Architectural Features

- **Fixed-Size Worker Pool:** Limits concurrency to prevent resource exhaustion (e.g., hitting file descriptor limits or overwhelming remote APIs).
- **Backpressure & Scalability:** Uses buffered channels with fixed capacities to regulate the producer's speed, ensuring constant $O(1)$ memory usage regardless of input size.
- **Graceful Shutdown:** Implements a coordinated shutdown sequence using `sync.WaitGroup` and a `chan struct{}` signal to ensure all results are processed and no goroutines leak.
- **Context-Aware Workers:** Every task is tied to a `context.Context`, allowing for immediate cancellation of in-flight network requests if a timeout occurs.
- **Zero-Allocation Signaling:** Utilizes the `chan struct{}` pattern for synchronization, the most memory-efficient signaling primitive in Go.

## Project Structure

```text
.
├── main.go             # Orchestrator (Producer/Consumer/Collector)
├── Makefile            # Standardized build and test commands
├── go.mod              # Dependency management
├── prometheus.yml      # Prometheus scrape config
├── promtail-config.yml # Promtail log shipping config
├── tempo.yml           # Tempo trace storage config
├── docker-compose.yml  # Full observability stack (Prometheus + Loki + Promtail + Tempo + Pyroscope + Grafana)
├── scripts/
│   └── pre-push.sh     # Local Git safety hook
├── metrics/
│   └── main.go         # Prometheus metrics definitions
└── task/
    ├── task.go         # Core business logic (HTTP Engine)
    └── task_test.go    # Table-driven unit tests
```

## Tech Stack

- **Language:** Go
- **Concurrency Primitives:** Goroutines, Channels, `sync.WaitGroup`, `context`
- **Observability:** Prometheus metrics, Loki logs, Tempo traces, Pyroscope profiles, Grafana dashboards
- **Testing:** Run with `go run -race main.go` to verify thread safety.

## Getting Started

### 1. Setup Environment

Install the local git hooks to ensure every push is vetted for race conditions:

```Bash
make setup
```

### 2. Run the Engine

```bash
make run
```

The app exposes metrics at `http://localhost:8080/metrics`.

### 3. Run Tests

Execute the table-driven test suite with the race detector enabled:

```bash
make test
```

## Observability

Spin up Prometheus, Loki, Promtail, Tempo, and Grafana alongside the running app to visualize metrics, logs, and traces.

**Start the stack:**

```bash
mkdir -p logs         # only needed once
make run              # terminal 1 — runs the app
make observability    # terminal 2 — starts the Docker stack
```

**Add data sources in Grafana** (`http://localhost:3000`, login: `admin` / `admin`):

**Prometheus** (metrics):

1. Go to **Connections → Data sources → Add** → select **Prometheus**
2. Set URL to `http://prometheus:9090`
3. Click **Save & test**

**Loki** (logs):

1. Go to **Connections → Data sources → Add** → select **Loki**
2. Set URL to `http://loki:3100`
3. Click **Save & test**

**Tempo** (traces):

1. Go to **Connections → Data sources → Add** → select **Tempo**
2. Set URL to `http://tempo:3200`
3. Click **Save & test**

> Use container service names, not `localhost` - inside Docker, `localhost` refers to the container itself. `grafana/tempo:latest` (v2.10+) requires Kafka configuration not covered here, so the stack pins `tempo:2.6.1` and `grafana:11.2.0`. Dashboards persist across restarts via a named Docker volume (`grafana-storage`). To wipe them, run `docker compose down -v`.

**Metrics** - create a dashboard via **Dashboards → New → Add visualization** and use PromQL:

*Worker pool:*

- `rate(jobs_total{status="success"}[1m])` - jobs completing successfully per second
- `rate(jobs_total{status="failed"}[1m])` - jobs failing per second
- `workers_active` - workers currently executing a job (0 means idle, 3 means saturated)
- `jobs_queue_depth` - jobs waiting in the channel buffer. High values mean workers can't keep up
- `histogram_quantile(0.95, rate(job_duration_seconds_bucket[1m]))` - 95% of jobs complete within this duration

*Go runtime:*

- `go_goroutines` - total live goroutines. A steady climb indicates a leak
- `go_gc_duration_seconds{quantile="1"}` - worst GC pause recorded (summary has fixed quantiles: 0, 0.25, 0.5, 0.75, 1)
- `go_memstats_heap_inuse_bytes` - heap memory actively holding live objects. It oscillates with GC cycles
- `rate(process_cpu_seconds_total[1m])` - CPU cores consumed per second
- `process_resident_memory_bytes` - RAM held by the process (RSS). Go holds freed memory in its runtime pool so this grows until the OS reclaims it, so use `heap_inuse_bytes` to detect application-level leaks

**Logs** - explore via **Explore → Loki** and query:

```logql
{job="go-worker-pool"}
```

**Traces** - explore via **Explore → Tempo**, search by service name `go-worker-pool`. Each job appears as a `process_job` span with `worker.id`, `job.id`, and `job.url` attributes. Failed jobs are marked with error status.

**Profiles** (flame graphs):

1. Go to **Connections → Data sources → Add** → select **Grafana Pyroscope**
2. Set URL to `http://pyroscope:4040`
3. Click **Save & test**
4. Explore via **Explore → Pyroscope**, select service `go-worker-pool` and a profile type (CPU, heap, allocs)

## Scalability Note

Unlike simple scripts that load all work into memory, this engine is designed for scale. By decoupling the Producer, Worker Engine, and Result Collector, each component can be independently tuned or replaced (e.g., swapping the URL list for a message queue like Kafka or RabbitMQ).

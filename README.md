# Go Worker Pool: Scalable Task Runner

A highly concurrent, memory-efficient worker pool implementation in Go. This project demonstrates idiomatic patterns for handling distributed-style workloads, focusing on resource management and graceful degradation.

## Architectural Features
- **Fixed-Size Worker Pool:** Limits concurrency to prevent resource exhaustion (e.g., hitting file descriptor limits or overwhelming remote APIs).
- **Backpressure & Scalability:** Uses buffered channels with fixed capacities to regulate the producer's speed, ensuring constant $O(1)$ memory usage regardless of input size.
- **Graceful Shutdown:** Implements a coordinated shutdown sequence using `sync.WaitGroup` and a `chan struct{}` signal to ensure all results are processed and no goroutines leak.
- **Context-Aware Workers:** Every task is tied to a `context.Context`, allowing for immediate cancellation of in-flight network requests if a timeout occurs.
- **Zero-Allocation Signaling:** Utilizes the `chan struct{}` pattern for synchronization—the most memory-efficient signaling primitive in Go.

## Project Structure

```
.
├── main.go             # Orchestrator (Producer/Consumer/Collector)
├── Makefile            # Standardized build and test commands
├── go.mod              # Dependency management
├── scripts/
│   └── pre-push.sh     # Local Git safety hook
└── task/
    ├── task.go         # Core business logic (HTTP Engine)
    └── task_test.go    # Table-driven unit tests
```

## Tech Stack
- **Language:** Go
- **Concurrency Primitives:** Goroutines, Channels, `sync.WaitGroup`, `context`
- **Testing:** Run with `go run -race main.go` to verify thread safety.


## Getting Started
### 1. Setup Environment
Install the local git hooks to ensure every push is vetted for race conditions:

```Bash
make setup
```

### 2. Run the Engine
```Bash
make run
```

### 3. Run Tests
Execute the table-driven test suite with the race detector enabled:

```Bash
make test
```

## Scalability Note
Unlike simple scripts that load all work into memory, this engine is designed for scale. By decoupling the Producer, Worker Engine, and Result Collector, each component can be independently tuned or replaced (e.g., swapping the URL list for a message queue like Kafka or RabbitMQ).

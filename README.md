# Go Worker Pool: Scalable Task Runner

A highly concurrent, memory-efficient worker pool implementation in Go. This project demonstrates idiomatic patterns for handling distributed-style workloads, focusing on resource management and graceful degradation.

## Architectural Features
- **Fixed-Size Worker Pool:** Limits concurrency to prevent resource exhaustion (e.g., hitting file descriptor limits or overwhelming remote APIs).
- **Backpressure Implementation:** Uses buffered channels with fixed capacities to regulate the producer's speed, ensuring constant $O(1)$ memory usage regardless of input size.
- **Graceful Shutdown:** Implements a coordinated shutdown sequence using `sync.WaitGroup` and a `chan struct{}` signal to ensure all results are processed and no goroutines leak.
- **Context-Aware Workers:** Every task is tied to a `context.Context`, allowing for immediate cancellation of in-flight network requests if a timeout occurs.
- **Zero-Allocation Signaling:** Utilizes the `chan struct{}` pattern for synchronization—the most memory-efficient signaling primitive in Go.

## Tech Stack
- **Language:** Go
- **Concurrency Primitives:** Goroutines, Channels, `sync.WaitGroup`, `context`
- **Testing:** Run with `go run -race main.go` to verify thread safety.

## Scalability Note
Unlike simple scripts that load all work into memory, this engine is designed for scale. By decoupling the Producer, Worker Engine, and Result Collector, each component can be independently tuned or replaced (e.g., swapping the URL list for a message queue like Kafka or RabbitMQ).

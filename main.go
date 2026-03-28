package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/ikristina/go-worker-pool/task"
)

const NUM_JOBS = 5 // total number of jobs (buffer)
const NUM_WORKERS = 3 // concurrency limit

func worker(ctx context.Context, id int, jobs <- chan task.Job, results chan <- task.Result, wg *sync.WaitGroup) {
	defer wg.Done() // the worker must signal that it finished

	for {
		select {
			case <- ctx.Done():
				// In case of context timeout
				return
			case j, ok := <- jobs:
				if !ok {
					return //channel is closed
				}
				// the work
				fmt.Printf("Worker %d started job %d\n", id, j.ID)
				r := task.Run(ctx, &j)
				select {
					case results <- *r: // put the result on the channel
					case <- ctx.Done():
						return
				}
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan task.Job, NUM_JOBS)
	results := make(chan task.Result, NUM_JOBS)

	var wg sync.WaitGroup
	// Start workers
	for w := 1; w <= NUM_WORKERS; w++ {
		wg.Add(1)
		go worker(ctx, w, jobs, results, &wg)
	}
	// Start producer goroutine

	go func() {
		urls := []string{
			"https://go.dev",
			"bobobo",
		    "https://google.com",
		    "https://github.com",
		    "https://pkg.go.dev",
			"https://opensource.org",
		}
		for i, url := range urls {
			jobs <- task.Job{ID: i + 1, URL: url}
		}
		close(jobs)
	}()

	// Result collection (consumer goroutine)
	done := make(chan struct{}) // to signal completion (struct{} is 0 memory while bool or int would take at least 1 or 8 bytes)

	go func() {
		for res := range results {
			if res.Err != nil {
				fmt.Printf("[final] job %d failed: %v\n", res.ID, res.Err)
			} else {
				fmt.Printf("[final] job %d sucess: %v\n", res.ID, res.Value)
			}
		}
		close(done)
	}()

	wg.Wait() // wait for workers to finish
	close(results)

	// wait for the collector to finish printing everything
	<- done
	fmt.Println("all systems shut down")
}

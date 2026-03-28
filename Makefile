.PHONY: run test race clean

# Run the application
run:
	go run main.go

# Run tests
test:
	go test -race ./... -v

# Run with the Race Detector
race:
	go run -race main.go

# Run static analysis and formatting
lint:
	go vet ./...
	go fmt ./...

# Clean up binaries if you build them
clean:
	go clean
	rm -f go-worker-pool

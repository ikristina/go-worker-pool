.PHONY: lint run test race clean

setup:
	git config core.hooksPath scripts
	chmod +x scripts/pre-push

# Run the app
run:
	go run main.go

# Run tests
test:
	go test -race ./... -v

# Run the app with the race detector
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

.PHONY: build test clean install help

# Build the goedgar CLI
build:
	go build -o goedgar ./cmd/goedgar

# Run tests
test:
	go test -v ./...

# Run tests (short mode - skips integration tests)
test-short:
	go test -v -short ./...

# Clean build artifacts
clean:
	rm -f goedgar
	rm -rf output/

# Install goedgar to $GOPATH/bin
install:
	go install ./cmd/goedgar

# Display help
help:
	@echo "Available targets:"
	@echo "  make build       - Build the goedgar CLI"
	@echo "  make test        - Run all tests (including integration tests)"
	@echo "  make test-short  - Run tests in short mode (skip integration tests)"
	@echo "  make clean       - Remove build artifacts and output directory"
	@echo "  make install     - Install goedgar to \$$GOPATH/bin"
	@echo "  make help        - Display this help message"

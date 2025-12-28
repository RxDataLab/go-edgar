.PHONY: build test clean install help snapshot-review snapshot-accept snapshot-reject

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

# Review snapshot changes (shows diffs for all .new files)
snapshot-review:
	@echo "Reviewing snapshot changes..."
	@find testdata -name "*.new" -type f | while read newfile; do \
		original=$${newfile%.new}; \
		echo ""; \
		echo "=== $$original ==="; \
		diff -u "$$original" "$$newfile" || true; \
	done
	@echo ""
	@echo "To accept all changes: make snapshot-accept"
	@echo "To reject all changes: make snapshot-reject"

# Accept all snapshot changes
snapshot-accept:
	@echo "Accepting snapshot changes..."
	@go test -v -run TestForm4Parser -update
	@echo "✓ Snapshots accepted. Review with 'git diff' before committing."

# Reject snapshot changes (remove .new files)
snapshot-reject:
	@echo "Rejecting snapshot changes..."
	@find testdata -name "*.new" -type f -delete
	@echo "✓ Snapshot changes rejected."

# Display help
help:
	@echo "Available targets:"
	@echo "  make build             - Build the goedgar CLI"
	@echo "  make test              - Run all tests (including integration tests)"
	@echo "  make test-short        - Run tests in short mode (skip integration tests)"
	@echo "  make clean             - Remove build artifacts and output directory"
	@echo "  make install           - Install goedgar to \$$GOPATH/bin"
	@echo "  make snapshot-review   - Review snapshot changes (show diffs)"
	@echo "  make snapshot-accept   - Accept all snapshot changes"
	@echo "  make snapshot-reject   - Reject all snapshot changes"
	@echo "  make help              - Display this help message"

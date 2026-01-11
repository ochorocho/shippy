.PHONY: help build test test-quick test-integration test-verbose clean install

# Default target
help:
	@echo "Shippy - TYPO3 Deployment Tool"
	@echo ""
	@echo "Available targets:"
	@echo "  make build              - Build the shippy binary"
	@echo "  make test               - Run all tests (SERIAL - no parallel execution)"
	@echo "  make test-quick         - Run quick tests only (no Docker required)"
	@echo "  make test-integration   - Run integration tests only (requires Docker)"
	@echo "  make test-verbose       - Run all tests with verbose output"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make install            - Install shippy to /usr/local/bin"
	@echo ""
	@echo "⚠️  IMPORTANT: Tests must run serially - never use bats --jobs flag"

# Build the binary
build:
	@echo "Building shippy..."
	@mkdir -p dist
	@go build -o dist/tinnie
	@echo "✓ Build complete: dist/tinnie"

# Run all tests serially (NO parallel execution)
test: build
	@echo "Running all tests (SERIAL execution - sharing resources)..."
	@echo "⚠️  Tests MUST run serially - do NOT use --jobs flag"
	@bats tests/

# Run only quick tests (no Docker required)
test-quick: build
	@echo "Running quick tests (no Docker required)..."
	@bats --filter-tags quick tests/

# Run only integration tests (requires Docker)
test-integration: build
	@echo "Running integration tests (requires Docker)..."
	@bats --filter-tags integration tests/

# Run all tests with verbose output
test-verbose: build
	@echo "Running all tests with verbose output..."
	@bats tests/ --show-output-of-passing-tests --verbose-run --print-output-on-failure

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf dist
	@echo "✓ Clean complete"

# Install to /usr/local/bin
install: build
	@echo "Installing shippy to /usr/local/bin..."
	@cp dist/tinnie /usr/local/bin/shippy
	@chmod +x /usr/local/bin/shippy
	@echo "✓ Installed: /usr/local/bin/shippy"

.PHONY: help build build-release test clean version

# Version information
VERSION := $(shell ./scripts/get-version.sh)
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
LDFLAGS := -X 'tinnie/cmd.Version=$(VERSION)' \
           -X 'tinnie/cmd.GitCommit=$(GIT_COMMIT)' \
           -X 'tinnie/cmd.BuildDate=$(BUILD_DATE)'

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build tinnie binary with version info
	@echo "Building tinnie $(VERSION)..."
	@mkdir -p dist
	@go build -ldflags "$(LDFLAGS)" -o dist/tinnie
	@echo "✓ Built dist/tinnie"

build-release: ## Build release binaries for all platforms
	@echo "Building release binaries for version $(VERSION)..."
	@mkdir -p dist
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/tinnie-darwin-amd64
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/tinnie-darwin-arm64
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/tinnie-linux-amd64
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/tinnie-linux-arm64
	@echo "✓ Built all release binaries"
	@ls -lh dist/

test: ## Run tests
	@echo "Running tests..."
	@bats tests/*.bats

version: ## Show version that would be built
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(GIT_COMMIT)"
	@echo "Date:    $(BUILD_DATE)"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf dist/
	@echo "✓ Cleaned"

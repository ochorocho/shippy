.PHONY: help build build-release test docker-test clean version brew-formula audit

# Don't forward an inherited GOROOT to the go tools. Every go binary already
# knows its own root; a stale/foreign GOROOT (e.g. exported by a version
# manager for a different Go version than the one on PATH) is what causes
# "compile: version ... does not match go tool version ..." failures.
unexport GOROOT

# Version information
VERSION := $(shell ./scripts/get-version.sh)
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
LDFLAGS := -X 'github.com/ochorocho/shippy/cmd.Version=$(VERSION)' \
           -X 'github.com/ochorocho/shippy/cmd.GitCommit=$(GIT_COMMIT)' \
           -X 'github.com/ochorocho/shippy/cmd.BuildDate=$(BUILD_DATE)'

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build shippy binary with version info
	@echo "Building shippy $(VERSION)..."
	@mkdir -p dist
	@go build -ldflags "$(LDFLAGS)" -o dist/shippy
	@echo "✓ Built dist/shippy"

build-release: ## Build release binaries for all platforms
	@echo "Building release binaries for version $(VERSION)..."
	@mkdir -p dist
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/shippy-darwin-amd64
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/shippy-darwin-arm64
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/shippy-linux-amd64
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/shippy-linux-arm64
	@echo "✓ Built all release binaries"
	@ls -lh dist/

test: ## Run tests
	@echo "Running tests..."
	@bats --filter-tags '!integration' tests/*.bats

docker-test: ## Build Docker image and run tests inside it
	@./scripts/docker-test.sh

audit: ## Run security audit (vet, race, govulncheck, gosec)
	@echo "==> go vet"
	@go vet ./...
	@echo "==> go test -race"
	@go test -race ./...
	@echo "==> govulncheck (known CVEs in dependencies)"
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	@echo "==> gosec (static security analysis)"
	@go run github.com/securego/gosec/v2/cmd/gosec@latest -quiet ./...
	@echo "✓ Audit complete"

brew-formula: ## Update Homebrew formula (Formula/shippy.rb) for the latest tag
	@./scripts/update-formula.sh

version: ## Show version that would be built
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(GIT_COMMIT)"
	@echo "Date:    $(BUILD_DATE)"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf dist/
	@echo "✓ Cleaned"

# ─────────────────────────────────────────────────────────────────────────────
# Orbit Makefile
# ─────────────────────────────────────────────────────────────────────────────

BINARY     := orbit
MODULE     := github.com/orbit-sh/orbit
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE       ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
  -X main.version=$(VERSION) \
  -X main.commit=$(COMMIT) \
  -X main.buildDate=$(DATE)

BUILD_DIR := dist

# ─────────────────────────────────────────────────────────────────────────────
# Development targets
# ─────────────────────────────────────────────────────────────────────────────

.PHONY: build
build: ## Build the binary for the current platform
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/orbit

.PHONY: run
run: ## Run orbit (pass ARGS="cmd args" to pass arguments)
	go run ./cmd/orbit $(ARGS)

.PHONY: install
install: ## Install orbit to GOPATH/bin
	go install -ldflags "$(LDFLAGS)" ./cmd/orbit

# ─────────────────────────────────────────────────────────────────────────────
# Testing
# ─────────────────────────────────────────────────────────────────────────────

.PHONY: test
test: ## Run all unit tests
	go test -race -count=1 ./...

.PHONY: test-short
test-short: ## Run tests, skip integration tests
	go test -short -race -count=1 ./...

.PHONY: cover
cover: ## Generate and open an HTML coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ─────────────────────────────────────────────────────────────────────────────
# Code quality
# ─────────────────────────────────────────────────────────────────────────────

.PHONY: lint
lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Install: https://golangci-lint.run/usage/install/"; exit 1; }
	golangci-lint run ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: fmt
fmt: ## Format all Go source files
	gofmt -s -w .

.PHONY: tidy
tidy: ## Tidy and verify go modules
	go mod tidy
	go mod verify

.PHONY: check
check: fmt vet lint test ## Run all quality checks

# ─────────────────────────────────────────────────────────────────────────────
# Cross-platform release builds
# ─────────────────────────────────────────────────────────────────────────────

.PHONY: release
release: ## Build release binaries for all supported platforms
	@mkdir -p $(BUILD_DIR)
	GOOS=linux   GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/orbit-linux-amd64    ./cmd/orbit
	GOOS=linux   GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/orbit-linux-arm64    ./cmd/orbit
	GOOS=darwin  GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/orbit-darwin-amd64   ./cmd/orbit
	GOOS=darwin  GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/orbit-darwin-arm64   ./cmd/orbit
	GOOS=windows GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/orbit-windows-amd64.exe ./cmd/orbit
	@echo ""
	@echo "Release binaries in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/

.PHONY: checksums
checksums: release ## Generate SHA256 checksums for release binaries
	cd $(BUILD_DIR) && sha256sum orbit-* > checksums.txt
	@echo "Checksums written to $(BUILD_DIR)/checksums.txt"

# ─────────────────────────────────────────────────────────────────────────────
# Development helpers
# ─────────────────────────────────────────────────────────────────────────────

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR) coverage.out coverage.html

.PHONY: deps
deps: ## Download and verify all module dependencies
	go mod download
	go mod verify

.PHONY: gen
gen: ## Run go generate (for mock generation, etc.)
	go generate ./...

.PHONY: help
help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

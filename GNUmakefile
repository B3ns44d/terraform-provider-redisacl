# Terraform Redis ACL Provider Makefile
# Version and build configuration
VERSION ?= 0.1.0
BINARY_NAME = terraform-provider-redisacl
PACKAGE = github.com/B3ns44d/terraform-provider-redisacl

# Build configuration
OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)
BUILD_DIR = dist
COVERAGE_DIR = coverage

# Terraform plugin directory for local development
PLUGIN_DIR = ~/.terraform.d/plugins/local/redisacl/redisacl/$(VERSION)/$(OS)_$(ARCH)

# Go build flags
LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(shell git rev-parse HEAD)"
BUILD_FLAGS = -trimpath $(LDFLAGS)

# Tools
GOLANGCI_LINT_VERSION = v1.55.2
TFPLUGINDOCS_VERSION = latest

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[0;33m
BLUE = \033[0;34m
NC = \033[0m # No Color

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo "$(BLUE)Terraform Redis ACL Provider$(NC)"
	@echo "$(BLUE)==============================$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(GREEN)Environment:$(NC)"
	@echo "  Version: $(VERSION)"
	@echo "  OS/Arch: $(OS)/$(ARCH)"
	@echo "  Go Version: $(shell go version | cut -d' ' -f3)"

## clean: Clean build artifacts and caches
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	go clean -cache -testcache -modcache
	@echo "$(GREEN)Clean complete$(NC)"

## deps: Download and verify dependencies
deps:
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	go mod download
	go mod verify
	@echo "$(GREEN)Dependencies ready$(NC)"

## deps-update: Update dependencies to latest versions
deps-update:
	@echo "$(BLUE)Updating dependencies...$(NC)"
	go get -u ./...
	go mod tidy
	cd tools && go get -u ./... && go mod tidy
	@echo "$(GREEN)Dependencies updated$(NC)"

## fmt: Format Go code
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	gofmt -s -w -e .
	go mod tidy
	@echo "$(GREEN)Code formatted$(NC)"

## fmt-check: Check if code is formatted
fmt-check:
	@echo "$(BLUE)Checking code formatting...$(NC)"
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "$(RED)The following files are not formatted:$(NC)"; \
		gofmt -s -l .; \
		echo "$(RED)Please run 'make fmt' to format your code$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Code is properly formatted$(NC)"

## lint: Run linters
lint: tools-golangci-lint
	@echo "$(BLUE)Running linters...$(NC)"
	golangci-lint run --timeout=10m --verbose
	@echo "$(GREEN)Linting complete$(NC)"

## lint-fix: Run linters with auto-fix
lint-fix: tools-golangci-lint
	@echo "$(BLUE)Running linters with auto-fix...$(NC)"
	golangci-lint run --fix --timeout=10m
	@echo "$(GREEN)Linting with fixes complete$(NC)"

## build: Build the provider binary
build: deps
	@echo "$(BLUE)Building provider...$(NC)"
	mkdir -p $(BUILD_DIR)
	go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

## build-all: Build for all supported platforms
build-all: deps
	@echo "$(BLUE)Building for all platforms...$(NC)"
	mkdir -p $(BUILD_DIR)
	@for os in linux windows darwin; do \
		for arch in amd64 arm64; do \
			if [ "$$os" = "windows" ] && [ "$$arch" = "arm64" ]; then continue; fi; \
			echo "Building for $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_$${os}_$${arch} .; \
		done; \
	done
	@echo "$(GREEN)Multi-platform build complete$(NC)"

## install: Install provider locally for development
install: build
	@echo "$(BLUE)Installing provider locally...$(NC)"
	mkdir -p $(PLUGIN_DIR)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(PLUGIN_DIR)/$(BINARY_NAME)
	@echo "$(GREEN)Provider installed to: $(PLUGIN_DIR)$(NC)"

## test: Run unit tests
test: deps
	@echo "$(BLUE)Running unit tests...$(NC)"
	mkdir -p $(COVERAGE_DIR)
	go test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(GREEN)Unit tests complete$(NC)"
	@echo "$(YELLOW)Coverage report: $(COVERAGE_DIR)/coverage.html$(NC)"

## test-short: Run unit tests (short mode)
test-short: deps
	@echo "$(BLUE)Running unit tests (short mode)...$(NC)"
	go test -short -v ./...
	@echo "$(GREEN)Short tests complete$(NC)"

## testacc: Run acceptance tests with testcontainers
testacc: deps
	@echo "$(BLUE)Running acceptance tests...$(NC)"
	@echo "$(YELLOW)Note: This will start Redis containers automatically$(NC)"
	mkdir -p $(COVERAGE_DIR)
	TF_ACC=1 go test -v -timeout=30m -parallel=4 \
		-coverprofile=$(COVERAGE_DIR)/acc-coverage.out \
		-covermode=atomic \
		./internal/provider/
	@echo "$(GREEN)Acceptance tests complete$(NC)"

## test-all: Run all tests (unit + acceptance)
test-all: test testacc
	@echo "$(GREEN)All tests complete$(NC)"

## benchmark: Run benchmarks
benchmark: deps
	@echo "$(BLUE)Running benchmarks...$(NC)"
	go test -bench=. -benchmem ./...
	@echo "$(GREEN)Benchmarks complete$(NC)"

## generate: Generate documentation and other files
generate: tools-tfplugindocs
	@echo "$(BLUE)Generating documentation...$(NC)"
	cd tools && go generate ./...
	@echo "$(GREEN)Documentation generated$(NC)"

## docs: Generate and serve documentation locally
docs: generate
	@echo "$(BLUE)Documentation generated in docs/ directory$(NC)"
	@if command -v python3 >/dev/null 2>&1; then \
		echo "$(YELLOW)Starting local server at http://localhost:8000$(NC)"; \
		cd docs && python3 -m http.server 8000; \
	else \
		echo "$(YELLOW)Install Python 3 to serve docs locally$(NC)"; \
	fi

## validate-examples: Validate all example configurations
validate-examples:
	@echo "$(BLUE)Validating examples...$(NC)"
	@for example in examples/*/; do \
		if [ -f "$$example/main.tf" ]; then \
			echo "Validating $$example"; \
			cd "$$example" && terraform init -backend=false && terraform validate && cd - > /dev/null; \
		fi; \
	done
	@echo "$(GREEN)Examples validation complete$(NC)"

## security: Run security scans
security:
	@echo "$(BLUE)Running security scans...$(NC)"
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		govulncheck ./...; \
	fi
	@echo "$(GREEN)Security scan complete$(NC)"

## release-dry: Dry run of release process
release-dry:
	@echo "$(BLUE)Running release dry run...$(NC)"
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --skip-publish --clean; \
	else \
		echo "$(RED)goreleaser not found. Install it first.$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Release dry run complete$(NC)"

## ci: Run all CI checks locally
ci: fmt-check lint test testacc validate-examples security
	@echo "$(GREEN)All CI checks passed!$(NC)"

## tools: Install all required tools
tools: tools-golangci-lint tools-tfplugindocs tools-goreleaser
	@echo "$(GREEN)All tools installed$(NC)"

## tools-golangci-lint: Install golangci-lint
tools-golangci-lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(BLUE)Installing golangci-lint $(GOLANGCI_LINT_VERSION)...$(NC)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi

## tools-tfplugindocs: Install terraform-plugin-docs
tools-tfplugindocs:
	@if ! command -v tfplugindocs >/dev/null 2>&1; then \
		echo "$(BLUE)Installing tfplugindocs...$(NC)"; \
		go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@$(TFPLUGINDOCS_VERSION); \
	fi

## tools-goreleaser: Install goreleaser
tools-goreleaser:
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "$(BLUE)Installing goreleaser...$(NC)"; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi

## version: Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Binary: $(BINARY_NAME)"
	@echo "Package: $(PACKAGE)"
	@echo "OS/Arch: $(OS)/$(ARCH)"
	@echo "Go Version: $(shell go version)"
	@echo "Git Commit: $(shell git rev-parse HEAD 2>/dev/null || echo 'unknown')"
	@echo "Git Branch: $(shell git branch --show-current 2>/dev/null || echo 'unknown')"

## dev-setup: Set up development environment
dev-setup: tools deps
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@echo "$(GREEN)Development environment ready!$(NC)"
	@echo ""
	@echo "$(YELLOW)Next steps:$(NC)"
	@echo "1. Run 'make build' to build the provider"
	@echo "2. Run 'make install' to install locally"
	@echo "3. Run 'make test' to run tests"
	@echo "4. Check examples/ directory for usage examples"

# Phony targets
.PHONY: help clean deps deps-update fmt fmt-check lint lint-fix build build-all install \
        test test-short testacc test-all benchmark generate docs validate-examples \
        security release-dry ci tools tools-golangci-lint tools-tfplugindocs tools-goreleaser \
        version dev-setup
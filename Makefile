.PHONY: build clean targets install lint test check build-all

# Default target
.DEFAULT_GOAL := targets

# Build the binary
build: ## Build the binary
	go build -o imgmkr .

# Clean build artifacts
clean: ## Clean build artifacts
	rm -f imgmkr

# Run tests
test: ## Run tests
	go test ./...

# Install the binary
install: ## Install the binary
	go install .

# Run linter
lint: ## Run linter
	go vet ./...
	go fmt ./...

# Run all checks
check: lint test ## Run all checks (lint and test)

# Build for multiple platforms
build-all: ## Build for multiple platforms (linux, darwin, windows)
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/imgmkr-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o dist/imgmkr-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o dist/imgmkr-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o dist/imgmkr-windows-amd64.exe .

# Show available targets
targets: ## Show available targets
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
.PHONY: help build test unit-test integration-test clean docker-build docker-push run fmt vet lint

# Variables
APP_NAME := tls-configurator
CONTAINER_IMAGE := quay.io/mdessi/$(APP_NAME)
#VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
VERSION := 1.0.0-RC
GOFILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOVET := $(GOCMD) vet

help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	$(GOBUILD) -o bin/$(APP_NAME) -ldflags="-X main.Version=$(VERSION)" ./cmd/tls-configurator
	@echo "Build complete: bin/$(APP_NAME)"

test: unit-test integration-test ## Run all tests

unit-test: ## Run unit tests
	@echo "Running unit tests..."
	$(GOTEST) -v -race -coverprofile=coverage-unit.out -covermode=atomic ./pkg/...
	@echo "Unit tests complete"

integration-test: ## Run integration tests (suite)
	@echo "Running integration tests..."
	$(GOTEST) -v -race -coverprofile=coverage-integration.out -covermode=atomic ./test/suite/...
	@echo "Integration tests complete"

coverage: test ## Generate coverage report
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage-unit.out -o coverage-unit.html
	$(GOCMD) tool cover -html=coverage-integration.out -o coverage-integration.html
	@echo "Coverage reports generated: coverage-unit.html, coverage-integration.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage*.out coverage*.html
	@echo "Clean complete"

podman-build: ## Build Podman image
	@echo "Building Podman image $(CONTAINER_IMAGE):$(VERSION)..."
	podman build -t $(CONTAINER_IMAGE):$(VERSION) -t $(CONTAINER_IMAGE):latest .
	@echo "Podman image built: $(CONTAINER_IMAGE):$(VERSION)"

podman-push: ## Push Podman image
	@echo "Pushing Podman image $(CONTAINER_IMAGE):$(VERSION)..."
	podman push $(CONTAINER_IMAGE):$(VERSION)
	podman push $(CONTAINER_IMAGE):latest
	@echo "Podman image pushed"

run: build ## Build and run the application
	@echo "Running $(APP_NAME)..."
	./bin/$(APP_NAME) --help

fmt: ## Format Go code
	@echo "Formatting code..."
	$(GOFMT) -s -w $(GOFILES)
	@echo "Formatting complete"

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "go vet complete"

lint: ## Run golangci-lint (requires golangci-lint to be installed)
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...
	@echo "Linting complete"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies downloaded"

verify: fmt vet test ## Run formatting, vetting, and tests
	@echo "Verification complete"

install: build ## Install the binary to GOPATH/bin
	@echo "Installing $(APP_NAME)..."
	cp bin/$(APP_NAME) $(GOPATH)/bin/
	@echo "Installed to $(GOPATH)/bin/$(APP_NAME)"

# Example usage targets
example-get: build ## Example: Get current TLS configuration
	./bin/$(APP_NAME) --action get --ingress-controller default

example-update: build ## Example: Update TLS configuration to TLS 1.3
	./bin/$(APP_NAME) --action update \
		--type Custom \
		--min-tls-version VersionTLS13 \
		--ciphers "ECDHE-RSA-CHACHA20-POLY1305,TLS_AES_128_GCM_SHA256" \
		--curves "X25519MLKEM512,P-256"

example-list: build ## Example: List all IngressControllers
	./bin/$(APP_NAME) --action list

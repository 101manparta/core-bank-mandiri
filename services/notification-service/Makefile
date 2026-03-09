# Notification Service Makefile
# Core Bank Mandiri

.PHONY: help build run test clean docker-build docker-run lint fmt vet check

# Variables
BINARY_NAME=notification-service
BINARY_PATH=bin/$(BINARY_NAME)
CMD_PATH=./cmd
DOCKER_IMAGE=core-bank-mandiri/$(BINARY_NAME)
VERSION?=1.0.0

# Default target
help:
	@echo "Core Bank Mandiri - Notification Service"
	@echo ""
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  run          - Run the service locally"
	@echo "  test         - Run unit tests"
	@echo "  test-verbose - Run unit tests with verbose output"
	@echo "  test-cover   - Run tests with coverage report"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container locally"
	@echo "  docker-push  - Push Docker image to registry"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  check        - Run all checks (lint, vet, test)"
	@echo ""

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-w -s -X main.Version=$(VERSION)" \
		-o $(BINARY_PATH) $(CMD_PATH)
	@echo "Build complete: $(BINARY_PATH)"

# Build for local development (your OS)
build-local:
	@echo "Building $(BINARY_NAME) for local development..."
	@mkdir -p bin
	go build -o $(BINARY_PATH) $(CMD_PATH)
	@echo "Build complete: $(BINARY_PATH)"

# Run the service locally
run: build-local
	@echo "Starting $(BINARY_NAME)..."
	./$(BINARY_PATH)

# Run with hot reload (requires air: go install github.com/air-verse/air@latest)
run-dev:
	@echo "Starting $(BINARY_NAME) with hot reload..."
	air -c .air.toml

# Run unit tests
test:
	@echo "Running tests..."
	go test $(CMD_PATH)

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	go test -v $(CMD_PATH)

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out $(CMD_PATH)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "Clean complete"

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .
	@echo "Docker image built: $(DOCKER_IMAGE):$(VERSION)"

# Run Docker container locally
docker-run:
	@echo "Running Docker container..."
	docker run --rm -p 8086:8086 \
		-e POSTGRES_HOST=host.docker.internal \
		-e KAFKA_BROKERS=host.docker.internal:9092 \
		$(DOCKER_IMAGE):latest

# Push Docker image
docker-push:
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest
	@echo "Docker image pushed"

# Run linter (requires golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
lint:
	@echo "Running linter..."
	golangci-lint run $(CMD_PATH)

# Format code
fmt:
	@echo "Formatting code..."
	go fmt $(CMD_PATH)

# Run go vet
vet:
	@echo "Running go vet..."
	go vet $(CMD_PATH)

# Run all checks
check: fmt vet test
	@echo "All checks passed!"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Generate mocks (requires mockgen: go install github.com/golang/mock/mockgen@latest)
generate:
	@echo "Generating mocks..."
	go generate ./...

# Database migrations (placeholder)
migrate-up:
	@echo "Running database migrations..."
	# Add migration tool command here

migrate-down:
	@echo "Rolling back database migrations..."
	# Add migration tool command here

# View logs (when running in Docker)
logs:
	docker logs -f $(BINARY_NAME)

# Health check
health:
	@echo "Checking service health..."
	curl -s http://localhost:8086/health | jq .

# Ready check
ready:
	@echo "Checking service readiness..."
	curl -s http://localhost:8086/ready | jq .

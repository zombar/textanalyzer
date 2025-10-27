.PHONY: build test clean run install lint fmt coverage help

# Binary name
BINARY_NAME=textanalyzer
SERVER_PATH=./cmd/server

# Build variables
VERSION?=1.0.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

help: ## Display this help message
	@echo "Text Analyzer - Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the server binary
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) $(SERVER_PATH)
	@echo "Build complete: $(BINARY_NAME)"

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 $(SERVER_PATH)
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 $(SERVER_PATH)
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 $(SERVER_PATH)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(SERVER_PATH)
	@echo "Multi-platform build complete"

install: ## Install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod verify
	@echo "Dependencies installed"

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-short: ## Run short tests only
	@go test -short ./...

test-trace: ## Run only trace propagation tests
	@echo "Running trace propagation tests..."
	@go test -v -run ".*Trace.*" ./internal/queue/...

test-trace-e2e: ## Run E2E trace flow tests
	@echo "Running E2E trace flow tests..."
	@go test -v -run ".*E2ETraceFlow.*" ./internal/queue/...

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./internal/analyzer

run: ## Run the server
	@echo "Starting server..."
	@go run $(SERVER_PATH)

run-dev: ## Run server with custom dev settings
	@echo "Starting development server..."
	@go run $(SERVER_PATH) -port 8080 -db dev.db

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatted"

lint: ## Run linter
	@echo "Running linter..."
	@go vet ./...
	@echo "Lint complete"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_NAME)-*
	@rm -f coverage.out coverage.html
	@rm -f *.db
	@rm -f test_*.db
	@echo "Clean complete"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t textanalyzer:$(VERSION) .
	@docker tag textanalyzer:$(VERSION) textanalyzer:latest
	@echo "Docker image built"

docker-run: ## Run in Docker
	@echo "Running Docker container..."
	@docker run -p 8080:8080 textanalyzer:latest

check: fmt lint test ## Run all checks (fmt, lint, test)

example-climate: build ## Run example with climate change text
	@echo "Analyzing climate change example..."
	@./$(BINARY_NAME) &
	@sleep 2
	@curl -X POST http://localhost:8080/api/analyze \
		-H "Content-Type: application/json" \
		-d @examples/climate_change.json | jq
	@killall $(BINARY_NAME)

example-review: build ## Run example with product review
	@echo "Analyzing product review example..."
	@./$(BINARY_NAME) &
	@sleep 2
	@curl -X POST http://localhost:8080/api/analyze \
		-H "Content-Type: application/json" \
		-d @examples/product_review.json | jq
	@killall $(BINARY_NAME)

deps-update: ## Update all dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "Dependencies updated"

deps-graph: ## Show dependency graph
	@go mod graph

.DEFAULT_GOAL := help

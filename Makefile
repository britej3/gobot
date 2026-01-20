.PHONY: build test clean install run-testnet run-prod setup-memory help

# Variables
BINARY_NAME=gobot
MAIN_PATH=cmd/gobot-engine/main.go
GO=go
PYTHON=python3

help: ## Show this help message
@echo "GOBOT - Autonomous Trading Bot"
@echo ""
@echo "Available targets:"
@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the trading bot binary
@echo "Building $(BINARY_NAME)..."
$(GO) build -o $(BINARY_NAME) $(MAIN_PATH)
@echo "Build complete: ./$(BINARY_NAME)"

build-all: ## Build all binaries
@echo "Building all binaries..."
$(GO) build -o gobot-engine cmd/gobot-engine/main.go
$(GO) build -o cobot cmd/cobot/main.go
$(GO) build -o cognee cmd/cognee/main.go
@echo "All builds complete"

test: ## Run all tests
@echo "Running tests..."
$(GO) test -v ./...

test-coverage: ## Run tests with coverage
@echo "Running tests with coverage..."
$(GO) test -v -coverprofile=coverage.out ./...
$(GO) tool cover -html=coverage.out -o coverage.html
@echo "Coverage report: coverage.html"

clean: ## Clean build artifacts
@echo "Cleaning..."
rm -f $(BINARY_NAME) gobot-engine cobot cognee
rm -f coverage.out coverage.html
rm -rf dist/ build/
@echo "Clean complete"

install: ## Install dependencies
@echo "Installing Go dependencies..."
$(GO) mod download
$(GO) mod tidy
@echo "Dependencies installed"

setup-memory: ## Set up the memory system (SimpleMem)
@echo "Setting up memory system..."
cd memory && ./setup.sh
@echo "Memory system setup complete"

run-testnet: build ## Run bot in testnet mode
@echo "Starting bot in testnet mode..."
./$(BINARY_NAME) --testnet

run-prod: build ## Run bot in production mode
@echo "⚠️  Starting bot in PRODUCTION mode..."
@echo "Press Ctrl+C within 5 seconds to cancel..."
@sleep 5
./$(BINARY_NAME)

verify: ## Verify repository setup
@echo "Verifying repository..."
./verify_repositories.sh

format: ## Format Go code
@echo "Formatting code..."
$(GO) fmt ./...
@echo "Format complete"

lint: ## Run linter
@echo "Running linter..."
golangci-lint run ./...

docker-build: ## Build Docker image
@echo "Building Docker image..."
docker build -t gobot:latest .

docker-run: ## Run Docker container
@echo "Running Docker container..."
docker-compose up -d

docker-stop: ## Stop Docker container
@echo "Stopping Docker container..."
docker-compose down

logs: ## View bot logs
@tail -f logs/gobot.log

monitor: ## Monitor trading activity
@./monitor_trades.sh

deps-update: ## Update dependencies
@echo "Updating dependencies..."
$(GO) get -u ./...
$(GO) mod tidy


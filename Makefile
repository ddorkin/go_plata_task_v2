# Makefile для go_plata_task_v2

.PHONY: build run test clean deps help swagger docker-build docker-run

BINARY_NAME=currency-quote-service
BUILD_DIR=build
MAIN_PATH=./cmd/server

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

run:
	@echo "Running $(BINARY_NAME)..."
	@echo "Loading configuration from .env file (if exists) or environment variables"
	@go run $(MAIN_PATH)

run-dev:
	@echo "Running in development mode..."
	@LOG_LEVEL=debug LOG_FORMAT=json SERVER_PORT=8080 WORKER_INTERVAL=10s go run $(MAIN_PATH)

deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

swagger:
	@echo "Generating Swagger documentation..."
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g cmd/server/main.go -o docs/; \
	else \
		echo "swag not found. Installing temporarily..."; \
		go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o docs/; \
	fi

test:
	@echo "Running all tests..."
	@go test -v ./...

test-handlers:
	@echo "Running handlers tests..."
	@go test -v ./internal/handlers/...

test-utils:
	@echo "Running utils tests..."
	@go test -v ./internal/utils/...


clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf docs/
	@go clean

fmt:
	@echo "Formatting code..."
	@go fmt ./...

lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Installing temporarily..."; \
		go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run; \
	fi

docker-build:
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .

docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --env-file .env $(BINARY_NAME):latest

docker-compose-up:
	@echo "Starting services with Docker Compose..."
	@docker-compose up --build

docker-compose-down:
	@echo "Stopping services with Docker Compose..."
	@docker-compose down

install-tools:
	@echo "Installing development tools..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

setup:
	@echo "Setting up development environment..."
	@make deps
	@make install-tools
	@make swagger
	@echo "✅ Development environment ready!"

help:
	@echo "Available commands:"
	@echo "  build              - Build the application"
	@echo "  run                - Run the application"
	@echo "  run-dev            - Run in development mode with debug logging"
	@echo "  deps               - Install dependencies"
	@echo "  swagger            - Generate Swagger documentation"
	@echo "  test               - Run all tests"
	@echo "  test-handlers      - Run handlers tests only"
	@echo "  test-utils         - Run utils tests only"
	@echo "  clean              - Clean build artifacts"
	@echo "  fmt                - Format code"
	@echo "  lint               - Lint code"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-run         - Run Docker container"
	@echo "  docker-compose-up  - Start services with Docker Compose"
	@echo "  docker-compose-down - Stop services with Docker Compose"
	@echo "  install-tools      - Install all development tools"
	@echo "  setup              - Complete development environment setup"
	@echo "  help               - Show this help"

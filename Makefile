# HowsWorkingThere API Makefile

.PHONY: build run test clean fmt lint vet deps docker-build docker-run help

# Binary name
BINARY_NAME=hwt-api
# Docker image name
DOCKER_IMAGE=hwt-api:latest

# Default target
.DEFAULT_GOAL := help

## build: Build the API binary
build:
	@echo "Building API..."
	go build -o $(BINARY_NAME) .

## run: Build and run the API locally
run: build
	@echo "Starting API server on :8080..."
	./$(BINARY_NAME)

## dev: Run the API with hot reload (requires air)
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Installing air for hot reload..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

## test: Run all tests
test:
	@echo "Running tests..."
	go test -v ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	go clean

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## lint: Run golangci-lint (if installed)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

## deps: Download and verify dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify

## tidy: Clean up go.mod and go.sum
tidy:
	@echo "Tidying modules..."
	go mod tidy

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

## docker-run: Run the API in Docker (requires postgres)
docker-run:
	@echo "Running API in Docker..."
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE)

## docker-compose-up: Start all services with docker-compose
docker-compose-up:
	@echo "Starting services with docker-compose..."
	docker-compose up --build

## docker-compose-down: Stop docker-compose services
docker-compose-down:
	@echo "Stopping docker-compose services..."
	docker-compose down

## migrate-up: Run database migrations up (requires golang-migrate)
migrate-up:
	@if command -v migrate >/dev/null 2>&1; then \
		migrate -path docs/migrations -database "$(DATABASE_URL)" up; \
	else \
		echo "migrate not installed. Run: go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		exit 1; \
	fi

## migrate-down: Rollback database migrations (requires golang-migrate)
migrate-down:
	@if command -v migrate >/dev/null 2>&1; then \
		migrate -path docs/migrations -database "$(DATABASE_URL)" down 1; \
	else \
		echo "migrate not installed. Run: go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; \
		exit 1; \
	fi

## check: Run all checks (fmt, vet, test)
check: fmt vet test
	@echo "All checks passed!"

## help: Show this help message
help:
	@echo "Available targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/  /'

# CrawlSmith Makefile

.PHONY: all build test clean install lint fmt coverage docker help

# Variables
BINARY_NAME=crawlsmith
DOCKER_IMAGE=crawlsmith:latest
GO_FILES=$(shell find . -name '*.go' -type f -not -path "./vendor/*")
COVERAGE_FILE=coverage.out

# Default target
all: test build

## help: Display this help message
help:
	@echo "CrawlSmith - SEO Analysis Suite"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@awk '/^##/ { printf("  %-15s %s\n", substr($$1, 4), substr($$0, index($$0, ":") + 2)) }' $(MAKEFILE_LIST) | sort

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -v -o bin/$(BINARY_NAME) ./cmd/crawlsmith

## test: Run all tests
test:
	@echo "Running tests..."
	@go test -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-race: Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	@go test -v -race ./...

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	@golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w $(GO_FILES)

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/ $(COVERAGE_FILE) coverage.html
	@go clean -cache

## install: Install dependencies
install:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

## update: Update dependencies
update:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

## docker: Build Docker image
docker:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	@docker run --rm -it $(DOCKER_IMAGE)

## release: Create a new release
release: clean test build
	@echo "Creating release..."
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 go build -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/crawlsmith
	@GOOS=darwin GOARCH=amd64 go build -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/crawlsmith
	@GOOS=windows GOARCH=amd64 go build -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/crawlsmith
	@echo "Release binaries created in dist/"

## dev: Run in development mode with hot reload
dev:
	@command -v air >/dev/null 2>&1 || { echo "Installing air..."; go install github.com/air-verse/air@latest; }
	@air

## proto: Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/*.proto

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test

## ci: Run CI pipeline locally
ci: clean install check build

.DEFAULT_GOAL := help
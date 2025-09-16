.PHONY: help build test clean

# Variables
APP_NAME=version-service
GO=go

help: ## Display this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the application binary
	$(GO) build -o bin/$(APP_NAME) main.go

test: ## Run tests
	$(GO) test ./...

test-coverage: ## Run tests with coverage
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	rm -rf bin/ coverage.out coverage.html tmp/

.DEFAULT_GOAL := help
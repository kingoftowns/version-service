.PHONY: help build test clean swagger

# Variables
APP_NAME=version-service
GO=go
SWAG=swag

help: ## Display this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

swagger: ## Generate Swagger documentation
	$(SWAG) init --generalInfo main.go --output ./docs

build: swagger ## Build the application binary
	$(GO) build -o bin/$(APP_NAME) main.go

test: ## Run tests
	$(GO) test ./...

test-coverage: ## Run tests with coverage
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	rm -rf bin/ coverage.out coverage.html tmp/ docs/

.DEFAULT_GOAL := help
.PHONY: all build build-release clean run dev test test-coverage \
        fmt vet check install help frontend frontend-install ensure-frontend \
        ensure-configfiles ensure-assets

# Build variables
VERSION ?= dev
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Binary name
BINARY_NAME := verustcode

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GORUN := $(GOCMD) run
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# Directories
BUILD_DIR := .
CMD_DIR := ./cmd/verustcode
COVERAGE_DIR := coverage

# Default target
all: build

## build: Build the binary with version info
build: ensure-frontend ensure-configfiles ensure-assets
	@echo "Building $(BINARY_NAME)..."
	@$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BINARY_NAME)"

## ensure-frontend: Ensure frontend dist directory exists
ensure-frontend:
	@if [ ! -d "internal/admin/dist" ]; then \
		echo "Frontend dist not found, building frontend..."; \
		$(MAKE) frontend; \
	fi

## ensure-configfiles: Ensure configfiles directory exists with required files
ensure-configfiles:
	@echo "Ensuring configfiles directory exists..."
	@echo "Syncing config files to internal/configfiles..."
	@mkdir -p internal/configfiles
	@cp config/bootstrap.example.yaml internal/configfiles/bootstrap.example.yaml
	@cp config/reviews/default.example.yaml internal/configfiles/default.example.yaml
	@rm -rf internal/configfiles/reports
	@mkdir -p internal/configfiles/reports
	@cp config/reports/*.yaml internal/configfiles/reports/
	@echo "Configfiles ready"

## ensure-assets: Copy shared assets from frontend to backend
ensure-assets:
	@echo "Syncing shared assets..."
	@cp frontend/public/logo.svg internal/report/assets/logo.svg
	@echo "Assets ready"

## build-release: Build the binary for release (with custom VERSION)
build-release: ensure-frontend ensure-configfiles ensure-assets
	@echo "Building $(BINARY_NAME) for release..."
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	@$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BINARY_NAME)"

## install: Install the binary to GOPATH/bin
install: ensure-frontend ensure-configfiles ensure-assets
	@echo "Installing $(BINARY_NAME)..."
	@$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) $(CMD_DIR)
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

## run: Build and run the service
run: build
	@echo "Starting $(BINARY_NAME)..."
	@./$(BINARY_NAME) serve

## dev: Run in development mode with debug logging
dev:
	@echo "Starting $(BINARY_NAME) in development mode..."
	@$(GORUN) $(CMD_DIR) serve --debug

## test: Run all tests
test:
	@echo "Running tests..."
	@$(GOTEST) -v -race -timeout 5m ./...

## test-fast: Run all tests without race detector (faster)
test-fast:
	@echo "Running tests (fast mode, no race detector)..."
	@$(GOTEST) -v -timeout 5m ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic $(shell go list ./... | grep -v /scripts)
	@$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"
	@$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total | awk '{print "Total coverage: " $$3}'

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	@$(GOFMT) ./...

## vet: Run go vet
vet: ensure-frontend ensure-configfiles
	@echo "Running go vet..."
	@$(GOVET) ./...

## check: Run format and vet checks
check: fmt vet
	@echo "All checks passed"

## clean: Clean build artifacts and temporary files
clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_NAME)
	@rm -rf $(COVERAGE_DIR)
	@rm -rf reports/
	@rm -rf workspace/
	@rm -rf data/*.db data/*.db-*
	@rm -rf internal/configfiles/*.yaml
	@rm -rf internal/configfiles/reports
	@echo "Clean complete"

## version: Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

## frontend-install: Install frontend dependencies
frontend-install:
	@echo "Installing frontend dependencies..."
	@cd frontend && npm install

## frontend: Build frontend and copy to embed directory
frontend: frontend-install
	@echo "Building frontend..."
	@cd frontend && npm run build
	@echo "Copying frontend dist to internal/admin..."
	@rm -rf internal/admin/dist
	@cp -r frontend/dist internal/admin/dist
	@echo "Frontend build complete"

## build-all: Build frontend and backend
build-all: frontend build
	@echo "Full build complete"

## help: Show this help message
help:
	@echo "VerustCode Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/  /'
	@echo ""
	@echo "Variables:"
	@echo "  VERSION     Version string (default: dev)"
	@echo ""
	@echo "Examples:"
	@echo "  make                          # Build the binary (default)"
	@echo "  make VERSION=v1.0.0 build-release  # Build release version"
	@echo "  make test                     # Run tests"
	@echo "  make dev                      # Run in debug mode"
	@echo "  make frontend                 # Build frontend only"
	@echo "  make build-all                # Build frontend + backend"

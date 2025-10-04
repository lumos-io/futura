VERSION := $(shell git describe --tags --always)
COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# ==================== APIs ====================

.PHONY: apis-build
apis-build: ## Build APIs binary
	go build -ldflags "-X 'github.com/opisvigilant/futura/backend/internal/apis/version.Version=$(VERSION)' \
	                   -X 'github.com/opisvigilant/futura/backend/internal/apis/version.CommitSHA=$(COMMIT)' \
	                   -X 'github.com/opisvigilant/futura/backend/internal/apis/version.BuildTime=$(BUILD_TIME)'" \
	         -o bin/backend 

.PHONY: apis-run
apis-run: ## Run APIs server
	go run -ldflags="-X 'github.com/opisvigilant/futura/backend/internal/apis/version.Version=dev' \
                 -X 'github.com/opisvigilant/futura/backend/internal/apis/version.CommitSHA=$(COMMIT)' \
                 -X 'github.com/opisvigilant/futura/backend/internal/apis/version.BuildTime=$(BUILD_TIME)'" \
		main.go apis

# ==================== Pipeline ====================

.PHONY: pipeline-collect
pipeline-collect: ## Run pipeline collect stage
	go run main.go pipeline collect

.PHONY: pipeline-validate
pipeline-validate: ## Run pipeline validate stage
	go run main.go pipeline validate

.PHONY: pipeline-enrich
pipeline-enrich: ## Run pipeline enrich stage
	go run main.go pipeline enrich

.PHONY: pipeline-store
pipeline-store: ## Run pipeline store stage
	go run main.go pipeline store

.PHONY: pipeline-all
pipeline-all: ## Run all pipeline stages in parallel
	go run main.go pipeline collect & \
	go run main.go pipeline validate & \
	go run main.go pipeline enrich & \
	go run main.go pipeline store & \
	wait

# ==================== Common Targets ====================

.PHONY: build
build: apis-build ## Build all binaries (default: APIs)

.PHONY: run
run: apis-run ## Run server (default: APIs)

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code
	go vet ./...

.PHONY: test
test: ## Run all tests
	go test -v -race -timeout=30s ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test-short
test-short: ## Run tests without verbose output
	go test -race -timeout=30s ./...

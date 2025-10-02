VERSION := $(shell git describe --tags --always)
COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: build
build:
	go build -ldflags "-X 'github.com/opisvigilant/futura/apis/version.Version=$(VERSION)' \
	                   -X 'github.com/opisvigilant/futura/apis/version.CommitSHA=$(COMMIT)' \
	                   -X 'github.com/opisvigilant/futura/apis/version.BuildTime=$(BUILD_TIME)'" \
	         -o bin/apis .


.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
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


.PHONY: run
run:
	go run -ldflags="-X 'github.com/opisvigilant/futura/apis/version.Version=dev' \
                 -X 'github.com/opisvigilant/futura/apis/version.CommitSHA=$(git rev-parse HEAD)' \
                 -X 'github.com/opisvigilant/futura/apis/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
		main.go

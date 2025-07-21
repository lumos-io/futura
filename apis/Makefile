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


.PHONY: run
run: 
	go run -ldflags="-X 'github.com/opisvigilant/futura/apis/version.Version=dev' \
                 -X 'github.com/opisvigilant/futura/apis/version.CommitSHA=$(git rev-parse HEAD)' \
                 -X 'github.com/opisvigilant/futura/apis/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
	main.go

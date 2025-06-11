# Image URL to use all building/pushing image targets
IMG ?= davideberdin/watcher:latest
CONTAINER_TOOL ?= docker

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: e2e
e2e:
	go run e2e/main.go

.PHONY: run
run:
	go run main.go

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# Image URL to use all building/pushing image targets
IMG ?= davideberdin/watcher:latest
CONTAINER_TOOL ?= docker
CLANG ?= clang-14
STRIP ?= llvm-strip-14
OBJCOPY ?= llvm-objcopy-14
TARGET_ARCH ?= x86	# x86 or arm64
CFLAGS := -O2 -g -Wall -Werror -D__TARGET_ARCH_$(TARGET_ARCH) $(CFLAGS)

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: run
run:
	go run main.go

generate: export BPF_CLANG := $(CLANG)
generate: export BPF_CFLAGS := $(CFLAGS)
generate:
	go generate ./...

%-el.elf: %.c
	$(CLANG) $(CFLAGS) -target bpfel -g -c $< -o $@
	$(STRIP) -g $@

%-eb.elf : %.c
	$(CLANG) $(CFLAGS) -target bpfeb -c $< -o $@
	$(STRIP) -g $@

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

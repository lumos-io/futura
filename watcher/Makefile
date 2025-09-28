.PHONY: build container-build build-ebpf build-binary docker-image clean

BINARY := watcher
BIN_DIR := bin
IMAGE_NAME := davideberdin/watcher:latest
BUILDER_IMAGE := $(IMAGE_NAME)-builder

# Host-level build: builds the builder image, then runs it to create local artifacts
build:
	@echo "==> Building builder image from project root..."
	cd .. && docker build --target builder -t $(BUILDER_IMAGE) -f watcher/Dockerfile .
	@echo "==> Running builder container to produce artifacts on host..."
	docker run --rm -v $(PWD):/src/watcher -w /src/watcher $(BUILDER_IMAGE) make container-build
	@echo "==> Artifacts produced: $(BIN_DIR)/$(BINARY)"

# Container-only build (runs inside builder image). This target must NOT run docker.
container-build: build-ebpf build-binary

# Generate (bpf2go) — will produce .o and Go bindings via your internal/ebpf/generate.go
# Must be run in a Linux environment with clang + bpf2go installed (builder image).
build-ebpf:
	@echo "==> Generating eBPF objects and Go bindings..."
	go generate ./internal/ebpf

# Build the Go binary. Depends on eBPF artifacts being present when needed.
build-binary:
	@echo "==> Building Go binary..."
	mkdir -p $(BIN_DIR)
	# build all packages (adjust package path if you prefer)
	CGO_ENABLED=0 GOARCH=arm64 go build -o $(BIN_DIR)/$(BINARY) .

# Build final runtime Docker image (multi-stage Dockerfile)
docker-image:
	cd .. && docker build -t $(IMAGE_NAME):latest -f watcher/Dockerfile .

clean:
	@echo "==> Cleaning bin/"
	rm -rf $(BIN_DIR)/*


# IMG ?= davideberdin/watcher:latest
# CONTAINER_TOOL ?= docker
# BIN_OUT ?= bin

# CLANG ?= clang-14
# STRIP ?= llvm-strip-14
# OBJCOPY ?= llvm-objcopy-14
# TARGET_ARCH ?= arm64	# x86 or arm64
# CFLAGS := -O2 -g -Wall -Werror -D__TARGET_ARCH_$(TARGET_ARCH) $(CFLAGS)

# .PHONY: all fmt vet clean generate build build-go build-docker build-bpf

# all: build

# fmt:
# 	go fmt ./...

# vet:
# 	go vet ./...

# generate: export BPF_CLANG := $(CLANG)
# generate: export BPF_CFLAGS := $(CFLAGS)
# generate:
# 	go mod tidy
# 	go generate ./...

# %-el.elf: %.c
# 	$(CLANG) $(CFLAGS) -target bpfel -g -c $< -o $@
# 	$(STRIP) -g $@

# %-eb.elf : %.c
# 	$(CLANG) $(CFLAGS) -target bpfeb -c $< -o $@
# 	$(STRIP) -g $@

# build-bpf: generate

# build-go:
# 	CGO_ENABLED=0 GOOS=linux GOARCH=$(TARGET_ARCH) go build -o $(BIN_OUT)/watcher main.go

# build: build-bpf build-go

# docker-build:
# 	$(CONTAINER_TOOL) build -t ${IMG} .

# # convenience target to run locally (requires Linux kernel/eBPF)
# run:
# 	sudo ./$(BIN_OUT)/watcher

# .PHONY: docker-push
# docker-push:
# 	$(CONTAINER_TOOL) push ${IMG}
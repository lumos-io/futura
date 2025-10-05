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
	cd .. && docker build -t $(IMAGE_NAME) -f watcher/Dockerfile .

clean:
	@echo "==> Cleaning bin/"
	rm -rf $(BIN_DIR)/*

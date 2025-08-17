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


IMG ?= davideberdin/watcher:latest
CONTAINER_TOOL ?= docker
BIN_OUT ?= bin
TARGET_ARCH ?= arm64  # x86_64 or arm64

.PHONY: all fmt vet clean generate build build-go docker-build docker-push run

all: build

fmt:
	go fmt ./...

vet:
	go vet ./...

# bpf2go handles clang/llvm calls internally
generate:
	go mod tidy
	go generate ./...

build-go:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(TARGET_ARCH) go build -o $(BIN_OUT)/watcher main.go

build: generate build-go

docker-build:
	$(CONTAINER_TOOL) build -t ${IMG} .

docker-push:
	$(CONTAINER_TOOL) push ${IMG}

run:
	sudo ./$(BIN_OUT)/watcher

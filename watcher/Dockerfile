# ---- Stage: Builder ----
FROM debian:bookworm AS builder
ENV DEBIAN_FRONTEND=noninteractive

# Base dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    clang llvm libbpf-dev make git pkg-config ca-certificates curl \
 && rm -rf /var/lib/apt/lists/*

# Install latest Go (1.24+)
ENV GOLANG_VERSION=1.24.4
RUN curl -fsSL https://go.dev/dl/go${GOLANG_VERSION}.linux-$(dpkg --print-architecture).tar.gz | \
    tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:${PATH}"

# Install bpf2go
RUN go install github.com/cilium/ebpf/cmd/bpf2go@latest

WORKDIR /src
# Copy proto module for local dependency
COPY proto/ ./proto/
COPY watcher/go.mod watcher/go.sum ./watcher/
WORKDIR /src/watcher
RUN go env -w GOPROXY=https://proxy.golang.org,direct
RUN go mod download

# Copy watcher source
COPY watcher/ ./

# Run the container-only build (this will run go generate -> bpf2go + go build)
# container-build is a Makefile target (no docker commands)
RUN make container-build

# ---- Stage: Runtime ----
FROM gcr.io/distroless/base
COPY --from=builder /src/bin/watcher /bin/watcher
# include generated eBPF objects if placed under internal/ebpf/bpf/
COPY --from=builder /src/internal/ebpf/bpf/*.o /bpf/
ENTRYPOINT ["/bin/watcher"]


# # build stage
# FROM golang:1.24.1 as builder

# ENV GO111MODULE=on

# WORKDIR /app

# RUN apt-get update && \
#     apt-get install -y bzr

# COPY go.mod .
# COPY go.sum .

# RUN go mod download

# COPY . .

# RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/watcher

# # final stage
# FROM alpine:3.16
# COPY --from=builder /app /app

# RUN apk update && \
#     apk add ca-certificates && \
#     rm -rf /var/cache/apk/*

# WORKDIR /app

# # PPROF port
# EXPOSE 6060 

# ENTRYPOINT ["/app/bin/watcher"]

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
# Pre-copy go.mod/go.sum to speed up dependency resolve
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org,direct
RUN go mod download

COPY . .

# Run the container-only build (this will run go generate -> bpf2go + go build)
# container-build is a Makefile target (no docker commands)
RUN make container-build

# ---- Stage: Runtime ----
FROM gcr.io/distroless/base
COPY --from=builder /src/bin/watcher /bin/watcher
# include generated eBPF objects if placed under internal/ebpf/bpf/
COPY --from=builder /src/internal/ebpf/bpf/*.o /bpf/
ENTRYPOINT ["/bin/watcher"]


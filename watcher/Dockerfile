# ----------- builder stage -----------
FROM ghcr.io/hemslo/docker-bpf:latest AS builder

WORKDIR /app

# Install Go (docker-bpf has clang/llvm/libbpf already, but not Go)
ENV GO_VERSION=1.24.1
RUN curl -sSL https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    | tar -C /usr/local -xz && \
    ln -s /usr/local/go/bin/go /usr/local/bin/go

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GO111MODULE=on

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build BPF objects (via Makefile or bpf2go)
RUN make build-bpf build-go

# ----------- final stage -----------
FROM alpine:3.18

WORKDIR /app

COPY --from=builder /app/bin/collector /app/bin/collector
COPY --from=builder /app/internal/ebpf/bpf/*.o /app/internal/ebpf/bpf/

RUN apk add --no-cache ca-certificates && \
    rm -rf /var/cache/apk/*

EXPOSE 6060

ENTRYPOINT ["/app/bin/collector"]

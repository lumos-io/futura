# Build the pipeline binary
FROM golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
# Copy local module dependencies
COPY proto/ ./proto/
COPY go-lib/ ./go-lib/

# Copy pipeline files with proper directory structure
COPY pipeline/go.mod pipeline/go.sum ./pipeline/
COPY pipeline/cmd ./pipeline/cmd
COPY pipeline/internal ./pipeline/internal
COPY pipeline/main.go ./pipeline/main.go
COPY pipeline/config.toml ./pipeline/config.toml

WORKDIR /src/pipeline
RUN go env -w GOPROXY=https://proxy.golang.org,direct

# Add replace directives for local modules
RUN echo '\nreplace github.com/opisvigilant/futura/proto => ../proto' >> go.mod && \
    echo 'replace github.com/opisvigilant/futura/go-lib => ../go-lib' >> go.mod

# Tidy and download dependencies
RUN go mod tidy && go mod download

# Build
# CGO_ENABLED=0 for static binary compatible with distroless
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o pipeline ./main.go

# Use distroless as minimal base image
FROM gcr.io/distroless/base-debian12:latest
WORKDIR /app
COPY --from=builder /src/pipeline/pipeline /app/pipeline
COPY --from=builder /src/pipeline/config.toml /app/config.toml

# Expose default port if needed
EXPOSE 8080

ENTRYPOINT ["/app/pipeline"]

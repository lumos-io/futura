# Build the analytics binary
FROM golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
# Copy proto module for local dependency
COPY proto/ ./proto/

# Copy analytics files with proper directory structure
COPY analytics/go.mod analytics/go.sum ./analytics/
COPY analytics/internal ./analytics/internal
COPY analytics/db ./analytics/db
COPY analytics/main.go ./analytics/main.go

WORKDIR /src/analytics
RUN go env -w GOPROXY=https://proxy.golang.org,direct

# Add replace directive for local proto module
RUN echo '\nreplace github.com/opisvigilant/futura/proto => ../proto' >> go.mod

# Tidy and download dependencies
RUN go mod tidy && go mod download

# Build
# CGO_ENABLED=0 for static binary compatible with distroless
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o analytics ./main.go

# Use distroless as minimal base image
FROM gcr.io/distroless/base-debian12:latest
WORKDIR /app
COPY --from=builder /src/analytics/analytics /app/analytics

# Expose gRPC port (default: 50051)
EXPOSE 50051

ENTRYPOINT ["/app/analytics"]

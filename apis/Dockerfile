# Build the APIs binary
FROM golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
# Copy local module dependencies
COPY proto/ ./proto/
COPY go-lib/ ./go-lib/

# Copy apis files with proper directory structure
COPY apis/go.mod apis/go.sum ./apis/
COPY apis/controllers ./apis/controllers
COPY apis/internal ./apis/internal
COPY apis/middleware ./apis/middleware
COPY apis/models ./apis/models
COPY apis/pkg ./apis/pkg
COPY apis/routes ./apis/routes
COPY apis/utils ./apis/utils
COPY apis/version ./apis/version
COPY apis/test ./apis/test
COPY apis/main.go ./apis/main.go

# Copy frontend build (must be pre-built: cd frontend && bun run build)
COPY apis/public ./apis/public

WORKDIR /src/apis
RUN go env -w GOPROXY=https://proxy.golang.org,direct

# Add replace directives for local modules
RUN echo '\nreplace github.com/opisvigilant/futura/proto => ../proto' >> go.mod && \
    echo 'replace github.com/opisvigilant/futura/go-lib => ../go-lib' >> go.mod

# Tidy and download dependencies
RUN go mod tidy && go mod download

# Build
# CGO_ENABLED=0 for static binary compatible with distroless
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o apis ./main.go

# Use distroless as minimal base image
FROM gcr.io/distroless/base-debian12:latest
WORKDIR /app
COPY --from=builder /src/apis/apis /app/apis

# Expose API port (default: 8080)
EXPOSE 8080

ENTRYPOINT ["/app/apis"]

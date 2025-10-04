# Stage 1: Build
FROM golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# Copy proto first
COPY proto/ ./proto/

WORKDIR /src/backend

# Copy go.mod/sum first for caching
COPY backend/go.mod backend/go.sum ./

RUN go env -w GOPROXY=https://proxy.golang.org,direct

RUN echo '\nreplace github.com/opisvigilant/futura/proto => ../proto' >> go.mod

RUN go mod tidy && go mod download

# Copy the rest of the backend source
COPY backend/ ./

# Build stripped binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o backend ./main.go

# Stage 2: Minimal runtime
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /src/backend/backend /app/backend

EXPOSE 8080
ENTRYPOINT ["/app/backend"]

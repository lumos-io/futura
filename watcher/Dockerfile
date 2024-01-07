# build stage
FROM golang:1.20 as builder

ENV GO111MODULE=on

WORKDIR /app

RUN apt-get update && \
    apt-get install -y bzr

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/watcher

# final stage
FROM alpine:3.16
COPY --from=builder /app /app

RUN apk update && \
    apk add ca-certificates && \
    rm -rf /var/cache/apk/*

WORKDIR /app

# PPROF port
EXPOSE 6060 

ENTRYPOINT ["/app/bin/watcher"]
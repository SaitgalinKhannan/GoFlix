# syntax=docker/dockerfile:1
# GoFlix/Dockerfile
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache ca-certificates git
WORKDIR /app
COPY go.mod go.sum ./
# Cache Go modules between builds (host-level BuildKit cache).
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o app -ldflags "-s -w" ./cmd/server

FROM alpine:latest
RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=builder /app/app .
RUN mkdir -p /app/data
EXPOSE 8081
ENTRYPOINT ["./app"]

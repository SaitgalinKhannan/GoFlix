# GoFlix/GoFlix/Dockerfile
FROM golang:1.24.5-alpine AS builder
RUN apk add --no-cache ca-certificates git
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o app -ldflags "-s -w" ./cmd/server

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/app .
RUN mkdir -p /app/data
EXPOSE 8081
ENTRYPOINT ["./app"]
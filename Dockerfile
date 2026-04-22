# syntax=docker.io/docker/dockerfile:1

# ========= BUILDER =========
FROM golang:1.26-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o server ./cmd/server/

# ========= RUNNER =========
FROM alpine:3.23
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Jakarta
# Copy binary
COPY --from=builder /app/server /app/server

# Create required directories (no need for chown anymore)
RUN mkdir -p /app/logs /app/uploads

# Run as root explicitly (optional, but clear)
USER 0:0

EXPOSE 4124

CMD ["./server"]
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
FROM alpine:3.23 AS runner
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

ARG UID=1000
ARG GID=1000
ARG userName=ginuser

RUN addgroup --system --gid $GID goapp && \
    adduser --system --uid $UID --ingroup goapp $userName

COPY --from=builder /app/server .

# Create ALL required directories before switching user
RUN mkdir -p logs uploads && chown -R $UID:$GID /app

USER $userName

EXPOSE 4124

CMD ["./server"]
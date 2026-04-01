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

ARG UID=1001
ARG GID=1001
ARG userName=ginuser

RUN addgroup --system --gid $GID goapp && \
    adduser --system --uid $UID --ingroup goapp ginuser

COPY --from=builder /app/server .

USER $userName

EXPOSE 4124

CMD ["./server"]
# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /market-sentinel ./cmd/bot

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1000 appuser

ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /market-sentinel .
COPY configs/config.yaml configs/config.yaml

RUN mkdir -p data && chown -R appuser:appuser /app

USER appuser

ENTRYPOINT ["./market-sentinel"]

# Multi-stage build for Aetheris runtime image (standalone Dockerfile).
# Builds API and Worker binaries, then packages into a minimal Alpine image.
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependency download
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 go build -o /bin/worker ./cmd/worker
RUN CGO_ENABLED=0 go build -o /bin/cli ./cmd/cli

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/api /bin/worker /bin/cli /usr/local/bin/
COPY --from=builder /app/configs /app/configs
COPY --from=builder /app/internal/runtime/jobstore/schema.sql /app/schema.sql

WORKDIR /app
EXPOSE 8080 9093

ENTRYPOINT ["/usr/local/bin/api"]

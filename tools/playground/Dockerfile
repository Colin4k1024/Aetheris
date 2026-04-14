# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the playground server
RUN CGO_ENABLED=0 GOOS=linux go build -o /playground-server ./main.go

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /playground-server /app/playground-server

# Copy static files
COPY --from=builder /app/static /app/static

# Expose port
EXPOSE 8081

# Run the server
ENTRYPOINT ["/app/playground-server"]

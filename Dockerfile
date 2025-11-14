# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o grpc-example .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/grpc-example .

# Copy certificates (will be overridden by volume mounts in production)
COPY --from=builder /build/certs ./certs

# Expose ports
# 10000 - gRPC
# 11000 - HTTP Gateway
EXPOSE 10000 11000

# Set environment variables with defaults
ENV GRPC_PORT=10000 \
    GRPC_GATEWAY_PORT=11000 \
    GRPC_HOST=0.0.0.0 \
    OTEL_ENABLED=true \
    OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317 \
    SERVICE_NAME=grpc-example \
    ENVIRONMENT=production

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider https://localhost:11000/ || exit 1

# Run the application
CMD ["./grpc-example", "-host", "0.0.0.0", "-insecure"]

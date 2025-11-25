# Full Security Stack Example Dockerfile
# Multi-stage build for optimized image size

# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/full-security-stack \
    ./examples/full-security-stack

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user for security
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/full-security-stack /app/

# Copy configuration and policies
COPY examples/full-security-stack/swit.yaml /app/
COPY examples/full-security-stack/policies /app/policies/

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose ports
# HTTP port
EXPOSE 8080
# HTTPS port (for mTLS)
EXPOSE 8443

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set environment variables
ENV HTTP_PORT=8080 \
    CONFIG_PATH=/app/swit.yaml \
    OAUTH2_ENABLED=true \
    OPA_ENABLED=true \
    MTLS_ENABLED=false \
    AUDIT_ENABLED=true \
    METRICS_ENABLED=true \
    POLICY_TYPE=rbac \
    OPA_MODE=embedded \
    OPA_POLICY_DIR=/app/policies

# Run the application
ENTRYPOINT ["/app/full-security-stack"]


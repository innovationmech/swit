# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY ../../../go.mod ../../../go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY ../../../ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o monitoring-service ./examples/prometheus-monitoring/

# Runtime stage
FROM alpine:3.18

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/monitoring-service .

# Copy configuration file
COPY swit.yaml .

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

# Run the binary
CMD ["./monitoring-service"]
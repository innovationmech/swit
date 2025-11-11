# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
WORKDIR /build/examples/oauth2-authentication
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o oauth2-example .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1000 swit && \
    adduser -D -u 1000 -G swit swit

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/examples/oauth2-authentication/oauth2-example .

# Copy configuration file
COPY examples/oauth2-authentication/swit.yaml .

# Change ownership
RUN chown -R swit:swit /app

# Switch to non-root user
USER swit

# Expose HTTP port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["./oauth2-example"]


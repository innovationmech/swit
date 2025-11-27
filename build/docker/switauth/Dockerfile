# syntax=docker/dockerfile:1

# =============================================================================
# Multi-stage build Dockerfile for swit-auth service
# Security optimizations:
# - Non-root user execution
# - Minimal image size with Alpine 3.21
# - Multi-stage build for smaller attack surface
# - Trivy security scan compatible
# =============================================================================

# =============================================================================
# Stage 1: Builder - Compile the Go application
# =============================================================================
FROM golang:1.23.12-alpine AS builder

# Install build dependencies with pinned versions for reproducibility
# hadolint ignore=DL3018
RUN apk add --no-cache git curl ca-certificates

WORKDIR /app

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Install buf tool for proto generation (pinned version)
RUN go install github.com/bufbuild/buf/cmd/buf@v1.47.2

# Generate protobuf code
WORKDIR /app/api
RUN buf dep update && buf generate

# Return to app directory and update dependencies
WORKDIR /app
RUN go mod tidy && \
    mkdir -p _output/build/swit-auth/linux/amd64

# Build arguments for version info
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# Build the binary with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}" \
    -o _output/build/swit-auth/linux/amd64/swit-auth \
    ./cmd/swit-auth

# =============================================================================
# Stage 2: Runtime - Minimal secure runtime image
# =============================================================================
FROM alpine:3.21 AS runtime

# Security: Install only essential runtime packages
# ca-certificates: Required for HTTPS connections
# tzdata: Required for timezone support
# hadolint ignore=DL3018
RUN apk --no-cache add ca-certificates tzdata && \
    rm -rf /var/cache/apk/*

# Security: Create non-root user and group with fixed UID/GID
ARG UID=10001
ARG GID=10001
RUN addgroup -g ${GID} -S swit && \
    adduser -u ${UID} -S -G swit -h /app -s /sbin/nologin swit

# Create application directory
WORKDIR /app
RUN chown -R swit:swit /app

# Copy the binary from builder stage with proper ownership
COPY --from=builder --chown=swit:swit /app/_output/build/swit-auth/linux/amd64/swit-auth /app/

# Copy configuration file with proper ownership
COPY --chown=swit:swit ./switauth.yaml /app/switauth.yaml

# Security: Switch to non-root user BEFORE exposing ports
USER swit

# Expose the service port
EXPOSE 8081

# OCI Image Labels for better traceability
LABEL org.opencontainers.image.source="https://github.com/innovationmech/swit" \
      org.opencontainers.image.vendor="innovationmech" \
      org.opencontainers.image.title="swit-auth" \
      org.opencontainers.image.description="Swit authentication service" \
      org.opencontainers.image.licenses="Apache-2.0"

# Set the entrypoint
ENTRYPOINT ["/app/swit-auth"]

# Default command
CMD ["start"]

# Configuration

This guide covers the comprehensive configuration options available in the Swit framework, including server settings, transport configuration, middleware, and best practices for different environments.

## Overview

Swit uses a hierarchical configuration system with YAML files, environment variables, and programmatic configuration. The `ServerConfig` structure provides type-safe configuration with validation and sensible defaults.

## Configuration Structure

### Main Configuration

```yaml
service_name: "my-service"
shutdown_timeout: "30s"

http:
  # HTTP transport configuration
  
grpc:
  # gRPC transport configuration
  
discovery:
  # Service discovery configuration
  
middleware:
  # Middleware configuration
```

## HTTP Transport Configuration

### Basic HTTP Settings

```yaml
http:
  enabled: true
  port: "8080"
  address: ":8080"
  enable_ready: true
  test_mode: false
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"
  headers:
    X-Service-Name: "my-service"
    X-Version: "1.0.0"
```

### HTTP Middleware Configuration

```yaml
http:
  middleware:
    enable_cors: true
    enable_auth: false
    enable_rate_limit: false
    enable_logging: true
    enable_timeout: true
    
    # CORS Configuration
    cors:
      allow_origins:
        - "http://localhost:3000"
        - "https://myapp.com"
      allow_methods:
        - "GET"
        - "POST"
        - "PUT"
        - "DELETE"
        - "OPTIONS"
      allow_headers:
        - "Origin"
        - "Content-Type"
        - "Accept"
        - "Authorization"
      expose_headers:
        - "X-Total-Count"
      allow_credentials: true
      max_age: 86400
    
    # Rate Limiting Configuration
    rate_limit:
      requests_per_second: 100
      burst_size: 200
      window_size: "60s"
      key_func: "ip"  # "ip", "user", "custom"
    
    # Timeout Configuration
    timeout:
      request_timeout: "30s"
      handler_timeout: "25s"
    
    # Custom Headers
    custom_headers:
      X-Request-ID: "auto"
      X-Service-Version: "1.0.0"
```

### Security Considerations for CORS

**Important:** Never use wildcard (`*`) origins with `allow_credentials: true`. This violates the CORS specification and creates security vulnerabilities.

```yaml
# ✅ Secure CORS configuration
cors:
  allow_origins:
    - "https://app.example.com"
    - "https://admin.example.com"
  allow_credentials: true

# ❌ Insecure configuration - will cause validation error
cors:
  allow_origins: ["*"]
  allow_credentials: true
```

## gRPC Transport Configuration

### Basic gRPC Settings

```yaml
grpc:
  enabled: true
  port: "9080"
  address: ":9080"
  enable_keepalive: true
  enable_reflection: true
  enable_health_service: true
  test_mode: false
  max_recv_msg_size: 4194304  # 4MB
  max_send_msg_size: 4194304  # 4MB
```

### gRPC Keepalive Configuration

```yaml
grpc:
  keepalive_params:
    max_connection_idle: "15s"
    max_connection_age: "30s"
    max_connection_age_grace: "5s"
    time: "5s"
    timeout: "1s"
  
  keepalive_policy:
    min_time: "5s"
    permit_without_stream: true
```

### gRPC Interceptors

```yaml
grpc:
  interceptors:
    enable_auth: false
    enable_logging: true
    enable_metrics: false
    enable_recovery: true
    enable_rate_limit: false
```

### gRPC TLS Configuration

```yaml
grpc:
  tls:
    enabled: true
    cert_file: "/path/to/server.crt"
    key_file: "/path/to/server.key"
    ca_file: "/path/to/ca.crt"
    server_name: "my-service.example.com"
```

## Service Discovery Configuration

### Basic Discovery Settings

```yaml
discovery:
  enabled: true
  address: "127.0.0.1:8500"
  service_name: "my-service"
  tags:
    - "v1"
    - "production"
    - "api"
  health_check_required: false
  registration_timeout: "30s"
```

### Discovery Failure Modes

```yaml
discovery:
  failure_mode: "graceful"  # "graceful", "fail_fast", "strict"
```

- **graceful**: Server continues startup even if discovery registration fails (default)
- **fail_fast**: Server startup fails if discovery registration fails
- **strict**: Requires discovery health check and fails fast on any discovery issues

## Monitoring Configuration (Sentry)

The framework provides comprehensive error monitoring and performance tracking through Sentry integration.

### Basic Sentry Configuration

```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"          # Set via environment variable
  environment: "production"      # deployment environment
  release: "v1.2.3"             # optional release version
  sample_rate: 1.0              # error sampling rate (0.0-1.0)
  traces_sample_rate: 0.1       # performance sampling rate (0.0-1.0)
```

### Complete Sentry Configuration

```yaml
sentry:
  # Basic settings
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "production"
  release: "v1.2.3"
  server_name: "my-server-01"
  
  # Sampling configuration
  sample_rate: 1.0              # Capture all errors
  traces_sample_rate: 0.1       # Capture 10% of performance traces
  profiles_sample_rate: 0.1     # Capture 10% of profiling data
  
  # Performance and tracing
  attach_stacktrace: true       # Include stack traces with errors
  enable_tracing: true          # Enable performance monitoring
  enable_profiling: true        # Enable profiling (requires traces)
  
  # Debug and development
  debug: false                  # Enable debug logging
  
  # Framework integration
  integrate_http: true          # Enable HTTP middleware
  integrate_grpc: true          # Enable gRPC middleware
  capture_panics: true          # Capture and recover from panics
  max_breadcrumbs: 30          # Maximum breadcrumb trail length
  
  # Context and data
  max_request_body_size: 1024   # Max request body to capture (bytes)
  send_default_pii: false       # Don't send personally identifiable info
  
  # Custom tags (added to all events)
  tags:
    service: "user-management"
    version: "1.2.3"
    datacenter: "us-west"
    team: "platform"
  
  # Error filtering
  ignore_errors:
    - "connection timeout"
    - "user not found"
    - "context deadline exceeded"
  
  # HTTP-specific filtering
  http_ignore_paths:
    - "/health"
    - "/metrics"
    - "/favicon.ico"
    - "/robots.txt"
  
  # HTTP status code filtering
  http_ignore_status_codes:
    - 404    # Not found errors
    - 400    # Bad request errors
    - 401    # Unauthorized (expected)
```

### Environment-Specific Sentry Configuration

#### Development
```yaml
sentry:
  enabled: true
  debug: true                   # Verbose logging for troubleshooting
  sample_rate: 1.0             # Capture all errors for testing
  traces_sample_rate: 1.0      # Capture all traces for development
  environment: "development"
  integrate_http: true
  integrate_grpc: true
```

#### Staging
```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN_STAGING}"
  environment: "staging"
  sample_rate: 1.0             # Capture all errors in staging
  traces_sample_rate: 0.5      # 50% performance sampling
  enable_profiling: true       # Test profiling in staging
  ignore_errors:
    - "test error"             # Filter out test-specific errors
```

#### Production
```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN_PRODUCTION}"
  environment: "production"
  release: "${APP_VERSION}"    # Set from CI/CD pipeline
  sample_rate: 1.0             # Capture all errors
  traces_sample_rate: 0.1      # 10% performance sampling
  profiles_sample_rate: 0.1    # 10% profiling
  enable_profiling: true
  
  # Production-specific filtering
  http_ignore_status_codes:
    - 404
    - 400
    - 401
    - 403
  
  ignore_errors:
    - "connection refused"
    - "timeout"
    - "rate limit exceeded"
```

### Sentry Configuration Best Practices

#### Sampling Strategy
- **Development**: High sampling rates (1.0) for comprehensive debugging
- **Staging**: Medium sampling rates (0.5) for realistic testing
- **Production**: Low sampling rates (0.1) to manage volume and costs

#### Error Filtering
- Filter out expected errors (4xx HTTP status codes)
- Ignore health check and metrics endpoints
- Filter noisy connection errors
- Use specific error patterns rather than broad filters

#### Performance Monitoring
- Enable tracing in all environments
- Use conservative sampling in production
- Monitor profiling overhead
- Set appropriate request body size limits

#### Security Considerations
- Never log sensitive data (PII, credentials)
- Use environment variables for DSN configuration
- Filter request bodies and headers containing secrets
- Be cautious with debug mode in production

### Sentry Environment Variables

```bash
# Required
export SENTRY_DSN="https://public@sentry.io/project-id"

# Optional overrides
export SENTRY_ENVIRONMENT="production"
export SENTRY_RELEASE="v1.2.3"
export SENTRY_SAMPLE_RATE="1.0"
export SENTRY_TRACES_SAMPLE_RATE="0.1"
export SENTRY_DEBUG="false"
export SENTRY_SERVER_NAME="my-server-01"
```

### Sentry Validation

The framework validates Sentry configuration on startup:

- **DSN Format**: Validates DSN URL format
- **Sampling Rates**: Ensures values are between 0.0 and 1.0
- **Environment**: Validates environment string format
- **Dependencies**: Checks for required Sentry SDK version

```go
// Configuration validation example
config := &ServerConfig{
    Sentry: SentryConfig{
        Enabled:           true,
        DSN:              "invalid-dsn",  // This will cause validation error
        SampleRate:       2.0,            // This will cause validation error
        TracesSampleRate: -0.1,           // This will cause validation error
    },
}

// Validation will fail with detailed error messages
if err := config.Validate(); err != nil {
    log.Fatal("Configuration validation failed:", err)
}
```

### Sentry Integration Examples

#### Custom Error Context
```go
import "github.com/getsentry/sentry-go"

func (h *Handler) ProcessOrder(c *gin.Context) {
    // Add custom context to Sentry
    sentry.ConfigureScope(func(scope *sentry.Scope) {
        scope.SetTag("operation", "process_order")
        scope.SetContext("order", map[string]interface{}{
            "id":       orderID,
            "amount":   order.Amount,
            "currency": order.Currency,
        })
    })
    
    // Business logic that might error
    if err := h.service.ProcessOrder(orderID); err != nil {
        // Error will include the context above
        sentry.CaptureException(err)
        c.JSON(500, gin.H{"error": "Order processing failed"})
        return
    }
}
```

#### Custom Performance Tracking
```go
func (s *Service) ExpensiveOperation() error {
    // Create custom transaction
    transaction := sentry.StartTransaction(
        context.Background(),
        "expensive-operation",
    )
    defer transaction.Finish()
    
    // Add operation metadata
    transaction.SetTag("operation_type", "data_processing")
    transaction.SetData("batch_size", batchSize)
    
    // Create span for database operation
    dbSpan := transaction.StartChild("database.query")
    dbSpan.SetTag("table", "orders")
    
    result, err := s.db.ProcessBatch()
    
    if err != nil {
        dbSpan.SetStatus(sentry.SpanStatusInternalError)
        transaction.SetStatus(sentry.SpanStatusInternalError)
        return err
    }
    
    dbSpan.SetData("rows_processed", result.Count)
    dbSpan.Finish()
    
    return nil
}
```

## Environment-Specific Configuration

### Development Configuration

```yaml
# config/development.yaml
service_name: "my-service-dev"
http:
  port: "8080"
  middleware:
    enable_cors: true
    cors:
      allow_origins:
        - "http://localhost:3000"
        - "http://localhost:8080"
      allow_credentials: true
grpc:
  enable_reflection: true
  enable_health_service: true
discovery:
  enabled: false  # Disable for local development
```

### Production Configuration

```yaml
# config/production.yaml
service_name: "my-service"
shutdown_timeout: "60s"
http:
  port: "8080"
  read_timeout: "30s"
  write_timeout: "30s"
  middleware:
    enable_cors: true
    enable_rate_limit: true
    enable_auth: true
    cors:
      allow_origins:
        - "https://app.example.com"
        - "https://admin.example.com"
      allow_credentials: true
    rate_limit:
      requests_per_second: 1000
      burst_size: 2000
grpc:
  enable_reflection: false  # Disable in production
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/server.crt"
    key_file: "/etc/ssl/private/server.key"
discovery:
  enabled: true
  failure_mode: "fail_fast"
  health_check_required: true
```

### Testing Configuration

```yaml
# config/test.yaml
service_name: "my-service-test"
http:
  port: "0"  # Dynamic port allocation
  test_mode: true
grpc:
  port: "0"  # Dynamic port allocation
  test_mode: true
discovery:
  enabled: false
```

## Environment Variables

### Override Configuration with Environment Variables

Environment variables follow the pattern: `SWIT_<SECTION>_<FIELD>`

```bash
# Override service name
export SWIT_SERVICE_NAME="my-service-prod"

# Override HTTP port
export SWIT_HTTP_PORT="9000"

# Override gRPC settings
export SWIT_GRPC_ENABLED="true"
export SWIT_GRPC_PORT="9001"

# Override discovery settings
export SWIT_DISCOVERY_ENABLED="true"
export SWIT_DISCOVERY_ADDRESS="consul.example.com:8500"

# Override nested configurations
export SWIT_HTTP_MIDDLEWARE_ENABLE_CORS="false"
export SWIT_GRPC_TLS_ENABLED="true"
```

## Programmatic Configuration

### Basic Programmatic Setup

```go
config := server.NewServerConfig()
config.ServiceName = "my-service"
config.ShutdownTimeout = 30 * time.Second

// Configure HTTP
config.HTTP.Enabled = true
config.HTTP.Port = "8080"
config.HTTP.Middleware.EnableCORS = true
config.HTTP.Middleware.CORSConfig.AllowOrigins = []string{
    "https://app.example.com",
}

// Configure gRPC
config.GRPC.Enabled = true
config.GRPC.Port = "9080"
config.GRPC.EnableReflection = true

// Configure discovery
config.Discovery.Enabled = true
config.Discovery.Address = "127.0.0.1:8500"
config.Discovery.ServiceName = "my-service"

// Validate configuration
if err := config.Validate(); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

### Configuration Builder Pattern

```go
config := server.NewServerConfigBuilder().
    WithServiceName("my-service").
    WithHTTPPort("8080").
    WithGRPCPort("9080").
    WithDiscovery("127.0.0.1:8500", "my-service").
    EnableCORS([]string{"https://app.example.com"}).
    EnableRateLimit(100, 200).
    Build()
```

## Configuration Validation

### Built-in Validation

The framework provides comprehensive validation:

```go
// Validate configuration
if err := config.Validate(); err != nil {
    // Handle validation errors
    fmt.Printf("Configuration validation failed: %v\n", err)
}
```

### Common Validation Errors

```go
// Port conflicts
"http.port and grpc.port must be different"

// Missing required fields
"service_name is required"
"http.port is required when HTTP is enabled"

// Invalid values
"http.read_timeout must be positive"
"discovery.failure_mode must be one of: graceful, fail_fast, strict"

// CORS security violations
"CORS security violation: cannot use wildcard origin '*' with AllowCredentials=true"
```

## Configuration Loading

### Loading from Files

```go
// Load from YAML file
config, err := server.LoadConfigFromFile("config/production.yaml")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// Load with environment overrides
config, err := server.LoadConfigWithEnv("config/production.yaml")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}
```

### Configuration Precedence

1. **Programmatic configuration** (highest priority)
2. **Environment variables**
3. **Configuration file**
4. **Default values** (lowest priority)

## Advanced Configuration

### Custom Middleware Configuration

```go
// Custom rate limiter key function
config.HTTP.Middleware.RateLimitConfig.KeyFunc = "custom"

// Custom headers based on environment
if os.Getenv("ENV") == "production" {
    config.HTTP.Headers["X-Environment"] = "production"
    config.HTTP.Headers["X-Security-Policy"] = "strict"
}
```

### Dynamic Configuration

```go
// Configuration hot reload
configWatcher := server.NewConfigWatcher("config.yaml")
configWatcher.OnChange(func(newConfig *server.ServerConfig) {
    // Handle configuration changes
    if newConfig.HTTP.Middleware.EnableRateLimit != currentConfig.HTTP.Middleware.EnableRateLimit {
        // Update rate limiter
    }
})
```

## Best Practices

### Configuration Management

1. **Use environment-specific files**: Separate configurations for dev, staging, production
2. **Validate early**: Always validate configuration during application startup
3. **Document defaults**: Clearly document all default values and their implications
4. **Secure secrets**: Never store secrets in configuration files; use environment variables or secret management
5. **Version configuration**: Keep configuration files in version control

### Security Best Practices

1. **CORS security**: Never use wildcard origins with credentials
2. **TLS configuration**: Always enable TLS in production environments
3. **Rate limiting**: Configure appropriate rate limits for your use case
4. **Timeouts**: Set reasonable timeouts to prevent resource exhaustion
5. **Reflection**: Disable gRPC reflection in production

### Performance Optimization

1. **Message sizes**: Configure appropriate gRPC message size limits
2. **Keepalive settings**: Tune keepalive parameters for your network conditions
3. **Timeout values**: Balance between user experience and resource usage
4. **Connection limits**: Set appropriate connection limits for your infrastructure

### Monitoring and Observability

1. **Health checks**: Configure meaningful health check endpoints
2. **Metrics**: Enable metrics collection for monitoring
3. **Logging**: Configure appropriate log levels for different environments
4. **Service discovery**: Use service discovery for dynamic environments

## Troubleshooting

### Common Configuration Issues

**Port conflicts:**
```yaml
# Ensure different ports for HTTP and gRPC
http:
  port: "8080"
grpc:
  port: "9080"  # Different from HTTP port
```

**CORS issues:**
```yaml
# Fix CORS security violations
cors:
  allow_origins:
    - "https://app.example.com"  # Specific origins
  allow_credentials: true
```

**Discovery registration failures:**
```yaml
# Use graceful failure mode for non-critical discovery
discovery:
  failure_mode: "graceful"
  registration_timeout: "30s"
```

### Debugging Configuration

```go
// Enable configuration debugging
config.SetDebugMode(true)

// Print effective configuration
config.Print()

// Get configuration summary
summary := config.GetSummary()
fmt.Printf("Configuration: %+v\n", summary)
```

This configuration guide provides comprehensive coverage of all configuration options, best practices, and troubleshooting guidance for the Swit framework.


### Complete Configuration Reference

- See the generated reference: [/en/guide/configuration-reference](/en/guide/configuration-reference)

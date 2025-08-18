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
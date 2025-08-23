# Sentry Integration Guide

This guide provides comprehensive documentation for integrating Sentry error monitoring and performance tracking with the Swit microservice framework.

## Overview

The Swit framework provides seamless Sentry integration through:

- **Automatic Error Capture**: HTTP and gRPC errors are automatically captured
- **Performance Monitoring**: Request tracing and performance metrics collection
- **Lifecycle Management**: Sentry initialization and cleanup handled by the framework
- **Middleware Integration**: Zero-configuration middleware for HTTP and gRPC transports
- **Comprehensive Configuration**: Full control over Sentry behavior and sampling

## Quick Start

### 1. Enable Sentry in Configuration

```yaml
# swit.yaml
service_name: "my-service"

sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"  # Set via environment variable
  environment: "production"
  sample_rate: 1.0
  traces_sample_rate: 0.1
```

### 2. Set Environment Variables

```bash
export SENTRY_DSN="https://your-dsn@sentry.io/your-project-id"
```

### 3. Use in Your Service

```go
package main

import (
    "github.com/innovationmech/swit/pkg/server"
)

func main() {
    config := server.NewServerConfig()
    config.ServiceName = "my-service"
    
    // Sentry will be automatically initialized if enabled in config
    server, err := server.NewBusinessServerCore(config, myService, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start server - Sentry is initialized automatically
    server.Start(context.Background())
}
```

That's it! The framework handles everything else automatically.

## Configuration Reference

### Complete Configuration Example

```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "production"
  release: "v1.2.3"
  sample_rate: 1.0
  traces_sample_rate: 0.1
  attach_stacktrace: true
  enable_tracing: true
  debug: false
  server_name: "my-server-01"
  
  # Custom tags added to all events
  tags:
    service: "user-management"
    version: "1.2.3"
    datacenter: "us-west"
  
  # Framework integration settings
  integrate_http: true
  integrate_grpc: true
  capture_panics: true
  max_breadcrumbs: 30
  
  # Error filtering
  ignore_errors:
    - "connection timeout"
    - "user not found"
  
  # HTTP-specific settings
  http_ignore_paths:
    - "/health"
    - "/metrics"
    - "/favicon.ico"
  
  http_ignore_status_codes:
    - 400  # Bad Request
    - 401  # Unauthorized
    - 403  # Forbidden
    - 404  # Not Found
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable/disable Sentry integration |
| `dsn` | string | `""` | Sentry project DSN (required when enabled) |
| `environment` | string | `"development"` | Environment tag (development, staging, production) |
| `release` | string | `""` | Release version for deployment tracking |
| `sample_rate` | float | `1.0` | Error sampling rate (0.0-1.0) |
| `traces_sample_rate` | float | `0.0` | Performance sampling rate (0.0-1.0) |
| `attach_stacktrace` | boolean | `true` | Include stack traces in error reports |
| `enable_tracing` | boolean | `false` | Enable performance monitoring |
| `debug` | boolean | `false` | Enable Sentry debug logging |
| `server_name` | string | `""` | Server identifier |
| `tags` | map | `{}` | Custom tags for all events |
| `integrate_http` | boolean | `true` | Enable HTTP middleware |
| `integrate_grpc` | boolean | `true` | Enable gRPC interceptors |
| `capture_panics` | boolean | `true` | Capture and report panics |
| `max_breadcrumbs` | integer | `30` | Maximum breadcrumbs per event |
| `ignore_errors` | []string | `[]` | Error message patterns to ignore |
| `http_ignore_paths` | []string | `[]` | HTTP paths to ignore |
| `http_ignore_status_codes` | []int | `[400, 401, 403, 404]` | HTTP status codes to ignore |

## Framework Integration

### Automatic Middleware

When Sentry is enabled, the framework automatically configures middleware:

#### HTTP Middleware Features
- Captures HTTP 5xx errors automatically
- Creates Sentry transactions for performance monitoring
- Adds request context (method, URL, headers, IP)
- Handles panic recovery
- Filters based on ignore patterns

#### gRPC Interceptor Features  
- Captures gRPC errors based on status codes
- Creates transactions for RPC calls
- Adds metadata context
- Handles panic recovery in handlers
- Works with both unary and streaming RPCs

### Server Integration

The Sentry manager is integrated into the server lifecycle:

```go
// Access the Sentry manager
sentryManager := server.GetSentryManager()

// Check if Sentry is enabled
if sentryManager.IsEnabled() {
    // Capture custom errors
    sentryManager.CaptureException(err)
}
```

### Lifecycle Management

1. **Initialization**: Sentry is initialized during `server.Start()`
2. **Context Setup**: Service tags and context are automatically added
3. **Shutdown**: Events are flushed during `server.Shutdown()`

## Manual Usage

### Capturing Errors

```go
import "github.com/getsentry/sentry-go"

// Capture exception with context
sentry.WithScope(func(scope *sentry.Scope) {
    scope.SetTag("operation", "user_creation")
    scope.SetContext("request", map[string]interface{}{
        "user_id": "123",
        "email":   "user@example.com",
    })
    sentry.CaptureException(err)
})

// Capture message
sentry.CaptureMessage("Custom error message", sentry.LevelError)
```

### Performance Monitoring

```go
// Create transaction
transaction := sentry.StartTransaction(ctx, "user_registration")
defer transaction.Finish()

// Add spans for sub-operations
span := transaction.StartChild("validate_email")
// ... validation logic
span.Finish()

span = transaction.StartChild("save_to_database")
// ... database logic
span.Finish()

// Set transaction status
transaction.Status = sentry.SpanStatusOK
```

### Adding Context

```go
// Set user context
sentry.ConfigureScope(func(scope *sentry.Scope) {
    scope.SetUser(sentry.User{
        ID:       "user123",
        Email:    "user@example.com",
        Username: "johndoe",
    })
})

// Add breadcrumbs
sentry.AddBreadcrumb(&sentry.Breadcrumb{
    Type:     "user",
    Category: "auth",
    Message:  "User logged in",
    Level:    sentry.LevelInfo,
    Data: map[string]interface{}{
        "user_id": "123",
        "ip":      "192.168.1.1",
    },
})

// Set custom context
sentry.ConfigureScope(func(scope *sentry.Scope) {
    scope.SetContext("business_data", map[string]interface{}{
        "order_id":    "order-123",
        "customer_id": "customer-456",
        "amount":      99.99,
    })
})
```

## Advanced Configuration

### Environment-Specific Settings

```yaml
# Development
sentry:
  enabled: true
  environment: "development"
  debug: true
  sample_rate: 1.0
  traces_sample_rate: 1.0  # Capture all traces in dev

# Production  
sentry:
  enabled: true
  environment: "production"
  debug: false
  sample_rate: 1.0
  traces_sample_rate: 0.01  # Sample 1% of traces
```

### Custom Tags and Context

```yaml
sentry:
  tags:
    service: "user-service"
    version: "${APP_VERSION}"
    datacenter: "${DATACENTER}"
    team: "backend"
```

### Error Filtering

Configure which errors to ignore:

```yaml
sentry:
  # Ignore specific error messages
  ignore_errors:
    - "connection timeout"
    - "user not authenticated"
    - "rate limit exceeded"
  
  # Ignore health check endpoints
  http_ignore_paths:
    - "/health"
    - "/ready"
    - "/metrics"
  
  # Don't capture client errors
  http_ignore_status_codes:
    - 400  # Bad Request
    - 401  # Unauthorized
    - 403  # Forbidden
    - 404  # Not Found
```

### Performance Sampling

Control performance monitoring overhead:

```yaml
sentry:
  enable_tracing: true
  traces_sample_rate: 0.1  # 10% sampling
  
  # For high-traffic services, use lower rates
  # traces_sample_rate: 0.01  # 1% sampling
```

## Best Practices

### 1. Environment Configuration

Use different configurations per environment:

```yaml
# Use environment variables for sensitive data
sentry:
  dsn: "${SENTRY_DSN}"
  environment: "${ENVIRONMENT}"
  
# Set appropriate sampling rates
sentry:
  sample_rate: 1.0        # Always capture errors
  traces_sample_rate: 0.1 # 10% performance sampling in prod
```

### 2. Error Classification

Configure ignore patterns for expected errors:

```yaml
sentry:
  ignore_errors:
    - "validation failed"      # Expected validation errors
    - "user not found"         # Expected business logic
    - "insufficient privileges" # Expected auth errors
  
  http_ignore_status_codes:
    - 400  # Client errors
    - 404  # Not found
```

### 3. Context and Tags

Add meaningful context for debugging:

```go
sentry.ConfigureScope(func(scope *sentry.Scope) {
    // Business context
    scope.SetTag("feature", "user_management")
    scope.SetTag("operation", "create_user")
    
    // System context
    scope.SetTag("instance_id", os.Getenv("INSTANCE_ID"))
    scope.SetTag("version", version.Version)
})
```

### 4. Performance Monitoring

Use transactions for critical user journeys:

```go
transaction := sentry.StartTransaction(ctx, "user_signup_flow")
defer transaction.Finish()

// Track each step
span := transaction.StartChild("validate_input")
// ... validation
span.Finish()

span = transaction.StartChild("check_existing_user") 
// ... check database
span.Finish()

span = transaction.StartChild("create_user_account")
// ... create user
span.Finish()
```

### 5. Privacy and Security

Be mindful of sensitive data:

```go
// DON'T capture sensitive data
sentry.ConfigureScope(func(scope *sentry.Scope) {
    scope.SetUser(sentry.User{
        ID:       user.ID,
        Username: user.Username,
        // DON'T include: Password, SSN, Credit Card, etc.
    })
})

// Use before-send hook to filter sensitive data
```

## Production Considerations

### Scaling and Performance

1. **Sampling Rates**: Use appropriate sampling rates for your traffic volume
2. **Async Processing**: Sentry uses async processing by default
3. **Memory Usage**: Monitor memory with high error volumes
4. **Network Impact**: Consider bandwidth for error payloads

### Monitoring and Alerting

1. **Set up alerts** for error rate thresholds
2. **Monitor performance** impact of Sentry integration  
3. **Track quota usage** in your Sentry dashboard
4. **Set up notification rules** for critical errors

### Data Retention

1. **Configure data retention** in Sentry project settings
2. **Use sampling** to manage data volume
3. **Archive old releases** to manage storage
4. **Set up data forwarding** if needed for compliance

## Troubleshooting

### Common Issues

#### Sentry Not Receiving Events

```bash
# Check DSN configuration
echo $SENTRY_DSN

# Enable debug mode in config
sentry:
  debug: true

# Check server logs for initialization messages
```

#### High Memory Usage

```yaml
# Reduce sampling rates
sentry:
  sample_rate: 0.8
  traces_sample_rate: 0.01
  max_breadcrumbs: 10
```

#### Performance Impact

```yaml
# Optimize for performance
sentry:
  traces_sample_rate: 0.01  # Lower sampling
  attach_stacktrace: false  # Reduce payload size
  max_breadcrumbs: 10       # Fewer breadcrumbs
```

### Debugging

Enable debug mode to see Sentry SDK activity:

```yaml
sentry:
  debug: true
```

Check server logs for:
- Sentry initialization messages
- Error capture attempts
- Network connectivity issues

### Testing

Test Sentry integration in development:

```bash
# Trigger test error
curl http://localhost:8080/api/test/error

# Check Sentry dashboard for event
```

## Migration Guide

### From Manual Integration

If you have existing manual Sentry integration:

1. **Remove manual initialization** code
2. **Move configuration** to server config
3. **Update error capture** to use framework patterns
4. **Test middleware integration**

### Version Upgrades

When upgrading the framework:

1. **Check configuration changes** in release notes
2. **Update sampling rates** if needed
3. **Test error capture** after upgrade
4. **Monitor performance** impact

## Examples

See the complete example service at:
- `examples/sentry-example-service/` - Full demonstration of Sentry features
- `examples/sentry-example-service/README.md` - Detailed setup instructions

## API Reference

### SentryManager Methods

```go
type SentryManager interface {
    // Lifecycle
    Initialize(ctx context.Context) error
    Close() error
    IsEnabled() bool
    IsInitialized() bool
    
    // Error capture
    CaptureException(error) *sentry.EventID
    CaptureMessage(string, sentry.Level) *sentry.EventID
    CaptureEvent(*sentry.Event) *sentry.EventID
    
    // Context management
    AddBreadcrumb(*sentry.Breadcrumb)
    ConfigureScope(func(*sentry.Scope))
    WithScope(func(*sentry.Scope))
    
    // Filtering
    ShouldCaptureHTTPError(int) bool
    ShouldCaptureHTTPPath(string) bool
    ShouldCaptureError(error) bool
    
    // Utilities
    Flush(time.Duration) bool
    AddContextTags(service, version, env string)
}
```

### Configuration Validation

The framework validates all Sentry configuration at startup:

- DSN format and reachability
- Sample rate ranges (0.0-1.0)  
- HTTP status code validity
- Environment string requirements
- Breadcrumb limits

Invalid configuration will prevent server startup with detailed error messages.

## Support

For issues and questions:

1. Check the troubleshooting section above
2. Review the example service implementation
3. Consult Sentry documentation at https://docs.sentry.io/
4. File issues in the framework repository

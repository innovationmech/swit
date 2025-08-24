# Error Monitoring and Performance Tracking

The Swit framework provides comprehensive error monitoring and performance tracking capabilities through Sentry integration. This guide covers everything from basic setup to advanced configuration and troubleshooting.

## Overview

Swit's monitoring system includes:

- **Automatic Error Capture**: HTTP and gRPC errors are automatically captured with full context
- **Performance Monitoring**: Request tracing, transaction monitoring, and performance metrics
- **Smart Filtering**: Configurable error filtering to reduce noise (ignores 4xx errors by default)
- **Context Enrichment**: Request metadata, user context, and custom tags in error reports
- **Panic Recovery**: Automatic panic capture and recovery with detailed stack traces
- **Zero-Config Defaults**: Works out-of-the-box with sensible production defaults

## Quick Start

### 1. Basic Configuration

Add Sentry configuration to your service config file:

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

### 2. Environment Setup

Set your Sentry DSN:

```bash
export SENTRY_DSN="https://your-dsn@sentry.io/your-project-id"
```

### 3. Framework Integration

The framework handles Sentry automatically:

```go
package main

import (
    "context"
    "log"
    "github.com/innovationmech/swit/pkg/server"
)

func main() {
    config := server.NewServerConfig()
    config.ServiceName = "my-service"
    
    // Sentry will be automatically initialized if enabled in config
    baseServer, err := server.NewBusinessServerCore(config, myService, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start server - Sentry monitoring begins automatically
    baseServer.Start(context.Background())
}
```

That's it! Your service now has comprehensive error monitoring and performance tracking.

## Advanced Configuration

### Complete Configuration Reference

```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "production"
  release: "v1.2.3"
  sample_rate: 1.0              # Error sampling rate (0.0-1.0)
  traces_sample_rate: 0.1       # Performance sampling rate
  attach_stacktrace: true
  enable_tracing: true
  debug: false                  # Enable for troubleshooting
  server_name: "my-server-01"
  
  # Custom tags added to all events
  tags:
    service: "user-management"
    version: "1.2.3"
    datacenter: "us-west"
  
  # Framework integration settings
  integrate_http: true          # Enable HTTP middleware
  integrate_grpc: true          # Enable gRPC middleware
  capture_panics: true          # Capture and recover from panics
  max_breadcrumbs: 30          # Maximum breadcrumb trail length
  
  # Error filtering (reduce noise)
  ignore_errors:
    - "connection timeout"
    - "user not found"
  
  # HTTP-specific filtering
  http_ignore_paths:
    - "/health"
    - "/metrics"
    - "/favicon.ico"
  
  # HTTP status code filtering
  http_ignore_status_codes:
    - 404    # Not found errors
    - 400    # Bad request errors
  
  # Performance monitoring
  enable_profiling: true
  profiles_sample_rate: 0.1
  
  # Context and breadcrumbs
  max_request_body_size: 1024   # Max request body to capture (bytes)
  send_default_pii: false       # Don't send personally identifiable info
```

### Environment-Specific Configuration

Different environments typically need different settings:

#### Development
```yaml
sentry:
  enabled: true
  debug: true
  sample_rate: 1.0
  traces_sample_rate: 1.0       # Capture all traces in dev
  environment: "development"
```

#### Staging
```yaml
sentry:
  enabled: true
  sample_rate: 1.0
  traces_sample_rate: 0.5       # 50% performance sampling
  environment: "staging"
```

#### Production
```yaml
sentry:
  enabled: true
  sample_rate: 1.0              # Capture all errors
  traces_sample_rate: 0.1       # 10% performance sampling
  environment: "production"
  enable_profiling: true
```

## Performance Monitoring

### Transaction Tracking

The framework automatically creates transactions for:

- **HTTP Requests**: Each HTTP request becomes a transaction
- **gRPC Calls**: Each gRPC method call is tracked
- **Custom Operations**: You can create custom transactions

Example of custom transaction:

```go
import "github.com/getsentry/sentry-go"

func processOrder(orderID string) error {
    // Create custom transaction
    transaction := sentry.StartTransaction(
        context.Background(), 
        "process-order",
    )
    defer transaction.Finish()
    
    // Add custom data
    transaction.SetTag("order_id", orderID)
    transaction.SetData("operation", "order_processing")
    
    // Your business logic here
    if err := validateOrder(orderID); err != nil {
        transaction.SetStatus(sentry.SpanStatusInvalidArgument)
        return err
    }
    
    return nil
}
```

### Custom Performance Spans

Create detailed performance spans:

```go
func (h *OrderHandler) CreateOrder(c *gin.Context) {
    span := sentry.StartSpan(c.Request.Context(), "database.query")
    span.SetTag("table", "orders")
    defer span.Finish()
    
    // Database operation
    order, err := h.db.CreateOrder(order)
    if err != nil {
        span.SetStatus(sentry.SpanStatusInternalError)
        sentry.CaptureException(err)
        c.JSON(500, gin.H{"error": "Failed to create order"})
        return
    }
    
    span.SetData("order_id", order.ID)
    c.JSON(201, order)
}
```

## Error Handling Best Practices

### Custom Error Context

Add rich context to errors:

```go
func (s *UserService) GetUser(id string) (*User, error) {
    sentry.ConfigureScope(func(scope *sentry.Scope) {
        scope.SetTag("service", "user-service")
        scope.SetContext("user_lookup", map[string]interface{}{
            "user_id": id,
            "timestamp": time.Now(),
        })
    })
    
    user, err := s.repository.FindByID(id)
    if err != nil {
        // This error will include the context above
        sentry.CaptureException(err)
        return nil, err
    }
    
    return user, nil
}
```

### Panic Recovery

The framework automatically recovers from panics, but you can also handle them manually:

```go
func riskyOperation() {
    defer func() {
        if err := recover(); err != nil {
            // Add custom context before reporting
            sentry.WithScope(func(scope *sentry.Scope) {
                scope.SetLevel(sentry.LevelFatal)
                scope.SetContext("panic_context", map[string]interface{}{
                    "operation": "risky_operation",
                    "timestamp": time.Now(),
                })
                sentry.CaptureException(fmt.Errorf("panic: %v", err))
            })
            // Re-panic if needed
            panic(err)
        }
    }()
    
    // Risky code here
}
```

## Testing with Sentry

### Mock Configuration for Testing

Disable Sentry in tests:

```go
func TestMyService(t *testing.T) {
    config := &server.ServerConfig{
        ServiceName: "test-service",
        Sentry: server.SentryConfig{
            Enabled: false, // Disable for testing
        },
    }
    
    // Your test code here
}
```

### Integration Testing

Test Sentry integration with mock DSN:

```go
func TestSentryIntegration(t *testing.T) {
    config := &server.ServerConfig{
        ServiceName: "test-service",
        Sentry: server.SentryConfig{
            Enabled: true,
            DSN: "http://public@example.com/1", // Mock DSN
            Debug: true,
        },
    }
    
    server, err := server.NewBusinessServerCore(config, service, nil)
    assert.NoError(t, err)
    
    // Test error capture
    err = errors.New("test error")
    sentry.CaptureException(err)
    
    // Flush events for testing
    sentry.Flush(time.Second * 2)
}
```

## Troubleshooting

### Common Issues

#### 1. Events Not Appearing in Sentry

**Check DSN Configuration:**
```bash
# Verify DSN is set
echo $SENTRY_DSN

# Test DSN format
curl -X POST "${SENTRY_DSN%/*}/api/${SENTRY_DSN##*/}/store/" \
  -H "Content-Type: application/json" \
  -d '{"message":"test"}'
```

**Enable Debug Mode:**
```yaml
sentry:
  debug: true  # Enable debug logging
```

#### 2. Too Many Events

**Adjust Sampling Rates:**
```yaml
sentry:
  sample_rate: 0.1        # Reduce error sampling
  traces_sample_rate: 0.05 # Reduce performance sampling
```

**Add More Filters:**
```yaml
sentry:
  ignore_errors:
    - "timeout"
    - "connection refused"
  http_ignore_status_codes:
    - 404
    - 400
    - 401
```

#### 3. Missing Context

**Verify Middleware Registration:**
```go
// Framework handles this automatically, but verify in logs
log.Info("HTTP Sentry middleware registered")
log.Info("gRPC Sentry middleware registered")
```

### Debug Information

Enable debug mode to see what Sentry is doing:

```yaml
sentry:
  debug: true
```

This will log:
- Event capture attempts
- Sampling decisions  
- Transport issues
- Configuration problems

## Best Practices

### 1. Production Configuration

- Use environment variables for sensitive data
- Set appropriate sampling rates
- Enable error filtering
- Configure meaningful tags and context

### 2. Development Workflow

- Use debug mode during development
- Test with mock DSN first
- Verify error capture in staging
- Monitor performance impact

### 3. Error Management

- Don't capture expected errors (4xx HTTP)
- Add meaningful context to errors
- Use appropriate severity levels
- Implement proper error boundaries

### 4. Performance Optimization

- Use low sampling rates in production
- Filter out health checks and metrics endpoints  
- Monitor Sentry SDK overhead
- Use async transport when possible

## Migration from Custom Error Handling

If you're migrating from custom error handling:

1. **Keep Existing Logs**: Sentry complements, doesn't replace logging
2. **Gradual Rollout**: Start with low sampling rates
3. **Context Migration**: Move custom context to Sentry tags/data
4. **Alert Migration**: Gradually move alerts to Sentry

Example migration:

```go
// Before: Custom error handling
func (h *Handler) ProcessRequest(c *gin.Context) {
    if err := h.service.Process(); err != nil {
        h.logger.Error("Process failed", 
            zap.Error(err),
            zap.String("request_id", c.GetHeader("X-Request-ID")),
        )
        c.JSON(500, gin.H{"error": "Internal error"})
        return
    }
}

// After: With Sentry integration
func (h *Handler) ProcessRequest(c *gin.Context) {
    if err := h.service.Process(); err != nil {
        // Keep existing logging
        h.logger.Error("Process failed", zap.Error(err))
        
        // Add Sentry context
        sentry.WithScope(func(scope *sentry.Scope) {
            scope.SetTag("request_id", c.GetHeader("X-Request-ID"))
            scope.SetContext("request", map[string]interface{}{
                "method": c.Request.Method,
                "path":   c.Request.URL.Path,
            })
            sentry.CaptureException(err)
        })
        
        c.JSON(500, gin.H{"error": "Internal error"})
        return
    }
}
```

## Related Topics

- [Configuration Guide](/en/guide/configuration) - Server and service configuration
- [Testing Guide](/en/guide/testing) - Testing strategies and patterns
- [Performance Guide](/en/guide/performance) - Performance optimization
- [Troubleshooting](/en/guide/troubleshooting) - Common issues and solutions

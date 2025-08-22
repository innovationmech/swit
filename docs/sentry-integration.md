# Sentry Integration Guide

The swit framework provides built-in integration with [Sentry](https://sentry.io) for comprehensive error monitoring and performance tracking in production environments.

## Overview

Sentry integration provides:
- **Automatic Error Capture**: HTTP and gRPC errors are automatically reported
- **Performance Monitoring**: Request tracing and transaction performance tracking
- **Panic Recovery**: Application panics are captured and reported while maintaining availability
- **Rich Context**: Request metadata, user context, and environment information
- **Breadcrumbs**: Automatic request lifecycle tracking for better debugging
- **Custom Tags and Metadata**: Service-specific context and categorization

## Quick Start

### 1. Enable Sentry in Configuration

```yaml
# swit.yaml
service_name: "my-service"

sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "${ENVIRONMENT:-production}"
  sample_rate: 1.0
  traces_sample_rate: 0.1
```

### 2. Set Environment Variables

```bash
export SENTRY_DSN="https://your-key@sentry.io/your-project-id"
export ENVIRONMENT="production"
```

### 3. Framework Automatic Integration

The framework automatically:
- Initializes Sentry during server startup
- Configures HTTP and gRPC middleware
- Handles graceful shutdown with event flushing
- Provides error reporting context

```go
// Your service code remains unchanged - Sentry works automatically
func (h *MyHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    ginRouter.GET("/api/v1/data", h.getData) // Errors automatically reported
    return nil
}
```

## Configuration Reference

### Core Sentry Settings

```yaml
sentry:
  enabled: true                    # Enable/disable Sentry integration
  dsn: "https://key@sentry.io/id"  # Sentry Data Source Name
  environment: "production"        # Environment tag (dev/staging/prod)
  release: "v1.2.3"               # Release version for tracking
  debug: false                     # Enable verbose Sentry logs
  server_name: "api-server-01"     # Server identifier
  flush_timeout: "2s"              # Event flush timeout on shutdown
```

### Sampling Configuration

```yaml
sentry:
  sample_rate: 1.0          # Error event sampling (0.0-1.0)
  traces_sample_rate: 0.1   # Performance trace sampling (0.0-1.0)  
  profiles_sample_rate: 0.1 # Profiling data sampling (0.0-1.0)
```

### Performance Monitoring

```yaml
sentry:
  enable_tracing: true      # Enable performance monitoring
  enable_profiling: false   # Enable CPU/memory profiling
  attach_stacktrace: true   # Include stack traces in events
```

### Custom Tags and Metadata

```yaml
sentry:
  tags:
    component: "api-gateway"
    team: "backend"
    version: "v2.1.0"
```

### Middleware Configuration

HTTP and gRPC middleware are automatically configured based on Sentry settings:

```yaml
http:
  middleware:
    enable_sentry: true  # Automatically set to match sentry.enabled

grpc:
  interceptors:
    enable_sentry: true  # Automatically set to match sentry.enabled
```

## Error Reporting Behavior

### HTTP Error Reporting

- **2xx responses**: No Sentry events
- **4xx responses**: Warning-level events (client errors)
- **5xx responses**: Error-level events (server errors) 
- **Panics**: Fatal-level events with full recovery

### gRPC Error Reporting

- **OK status**: No Sentry events
- **Client errors** (InvalidArgument, NotFound): Warning-level events
- **Server errors** (Internal, Unknown): Error-level events
- **Panics**: Fatal-level events with recovery

### Request Context

All error events automatically include:

```json
{
  "request": {
    "method": "POST",
    "url": "https://api.example.com/users",
    "headers": {"..."},
    "user_agent": "...",
    "remote_ip": "192.168.1.1"
  },
  "server": {
    "name": "api-server-01",
    "version": "v1.2.3",
    "environment": "production"
  },
  "tags": {
    "component": "http",
    "service": "user-service"
  }
}
```

## Advanced Usage

### Manual Error Reporting

Access the Sentry manager for custom reporting:

```go
func (s *MyService) someOperation() error {
    server := s.getServer() // Your server instance
    sentryManager := server.GetSentryManager()
    
    if err := riskyOperation(); err != nil {
        // Capture with custom context
        sentryManager.CaptureError(err, 
            map[string]string{"operation": "risky_op"},
            map[string]interface{}{"user_id": 12345})
        return err
    }
    return nil
}
```

### Custom Breadcrumbs

Add custom breadcrumbs for debugging context:

```go
sentryManager.AddBreadcrumb(
    "User authentication started",
    "auth",
    sentry.LevelInfo,
    map[string]interface{}{
        "user_id": userID,
        "method": "oauth2",
    })
```

### Performance Tracking

Create custom transactions:

```go
func (h *Handler) complexOperation(c *gin.Context) {
    sentryManager := h.getSentryManager()
    span := sentryManager.StartTransaction(c.Request.Context(), 
        "complex_operation", "business_logic")
    defer span.Finish()
    
    // Your business logic here
    span.SetData("records_processed", count)
}
```

## Environment-Specific Configuration

### Development Environment

```yaml
sentry:
  enabled: false  # Disable in development
  # OR enable with different settings
  enabled: true
  environment: "development"
  debug: true
  sample_rate: 0.1  # Reduce noise in development
```

### Staging Environment

```yaml
sentry:
  enabled: true
  environment: "staging"
  sample_rate: 1.0
  traces_sample_rate: 0.5  # Higher sampling for testing
```

### Production Environment

```yaml
sentry:
  enabled: true
  environment: "production"
  sample_rate: 1.0
  traces_sample_rate: 0.1  # Balanced performance impact
  enable_profiling: false  # Disabled for performance
```

## Monitoring and Alerting

### Sentry Dashboard Integration

After integration, your Sentry dashboard will show:
- **Error Events**: Categorized by service and environment
- **Performance Data**: Request duration and throughput metrics
- **Release Health**: Error rates and crash statistics
- **User Impact**: Affected user sessions and geographic data

### Alert Configuration

Set up Sentry alerts for:
- Error rate thresholds (e.g., >1% error rate)
- New error types or regressions  
- Performance degradation (e.g., >2s response time)
- High memory usage or resource exhaustion

## Troubleshooting

### Common Issues

1. **Events Not Appearing in Sentry**
   - Verify DSN is correct and accessible
   - Check network connectivity to sentry.io
   - Ensure `sentry.enabled: true` in configuration
   - Look for Sentry initialization logs

2. **Too Many Events**
   - Adjust `sample_rate` to reduce volume
   - Configure before_send filters
   - Review error patterns and fix root causes

3. **Performance Impact**
   - Lower `traces_sample_rate` (default 0.1 = 10%)
   - Disable profiling in production
   - Use async event sending (default behavior)

### Debug Mode

Enable debug logging to troubleshoot:

```yaml
sentry:
  enabled: true
  debug: true  # Enables verbose Sentry logs
```

### Health Check Integration

Sentry status can be included in health checks:

```go
func (h *HealthHandler) CheckSentry() *types.HealthStatus {
    sentryManager := h.getSentryManager()
    if !sentryManager.IsEnabled() {
        return &types.HealthStatus{
            Status: types.HealthStatusHealthy,
            Message: "Sentry disabled",
        }
    }
    // Additional health checks...
}
```

## Example Implementation

See the complete example in `examples/sentry-example-service/` which demonstrates:
- Configuration setup
- Error scenarios (4xx, 5xx, panics)
- Performance monitoring
- Custom context and tagging
- Environment-based configuration

## Best Practices

1. **Use Environment Variables**: Store sensitive DSN in environment variables
2. **Appropriate Sampling**: Balance data collection with performance
3. **Custom Tags**: Add service-specific metadata for better organization
4. **Error Context**: Include relevant business context in error reports
5. **Performance Budget**: Monitor Sentry's impact on application performance
6. **Alert Fatigue**: Configure meaningful alerts, not just error counts
7. **Regular Review**: Periodically review and act on Sentry data

## Security Considerations

- **Sensitive Data**: Sentry automatically scrubs common sensitive fields
- **Custom Scrubbing**: Configure additional data scrubbing if needed
- **Network Security**: Ensure network policies allow outbound HTTPS to sentry.io
- **DSN Security**: Treat DSN as a secret; use environment variables or secret management

The Sentry integration provides comprehensive observability for your swit-based microservices with minimal configuration and automatic error reporting.
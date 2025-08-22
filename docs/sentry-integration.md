# Sentry Integration Guide

The Swit framework includes built-in support for [Sentry](https://sentry.io) error monitoring and performance tracking. This integration provides real-time error reporting, alerting, and debugging capabilities for production environments.

## Features

- **Error Capture**: Automatically captures and reports application errors to Sentry
- **HTTP Middleware**: Captures HTTP request errors and performance data
- **gRPC Interceptors**: Captures gRPC service errors and performance data  
- **Performance Monitoring**: Optional performance trace sampling
- **Graceful Degradation**: Server continues to operate even if Sentry initialization fails
- **Configurable Sampling**: Control error and performance trace sampling rates
- **Custom Tags**: Add custom tags for better error categorization

## Configuration

Add the Sentry configuration section to your server config:

```yaml
sentry:
  enabled: true
  dsn: "https://your-dsn-key@your-org.sentry.io/your-project-id"
  environment: "production"
  release: "v1.0.0"
  sample_rate: 1.0        # Capture 100% of errors
  traces_sample_rate: 0.1 # Capture 10% of performance traces
  debug: false
  attach_stacktrace: true
  tags:
    service: "my-service"
    team: "backend"
    region: "us-west-2"
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable/disable Sentry monitoring |
| `dsn` | string | `""` | Sentry Data Source Name (required when enabled) |
| `environment` | string | `"development"` | Environment name (e.g., production, staging, development) |
| `release` | string | `""` | Release version identifier |
| `sample_rate` | float | `1.0` | Error sampling rate (0.0 to 1.0) |
| `traces_sample_rate` | float | `0.1` | Performance trace sampling rate (0.0 to 1.0) |
| `debug` | boolean | `false` | Enable Sentry debug logging |
| `attach_stacktrace` | boolean | `true` | Attach stack traces to error reports |
| `tags` | map | `{}` | Custom tags to add to all events |

## Getting Your Sentry DSN

1. Sign up for a [Sentry account](https://sentry.io)
2. Create a new project for your service
3. Go to **Settings** > **Projects** > **[Your Project]** > **Client Keys (DSN)**
4. Copy the DSN URL

## Example Usage

### Basic Setup

```go
package main

import (
    "context"
    "github.com/innovationmech/swit/pkg/server"
)

func main() {
    config := server.NewServerConfig()
    config.ServiceName = "my-service"
    
    // Enable Sentry
    config.Sentry.Enabled = true
    config.Sentry.DSN = "https://your-dsn@sentry.io/project-id"
    config.Sentry.Environment = "production"
    config.Sentry.Release = "v1.0.0"
    
    // Create and start server
    srv, err := server.NewBusinessServerCore(config, myServiceRegistrar, nil)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    if err := srv.Start(ctx); err != nil {
        panic(err)
    }
    
    // Server will automatically capture errors and send them to Sentry
}
```

### Manual Error Capture

You can also manually capture errors in your application code:

```go
// Get the Sentry manager from the server (if you need manual capture)
// Note: Usually automatic capture via middleware is sufficient

import "github.com/getsentry/sentry-go"

// Capture an error with context
sentry.CaptureException(err)

// Capture a message with custom data
sentry.WithScope(func(scope *sentry.Scope) {
    scope.SetTag("user_id", "12345")
    scope.SetExtra("additional_data", map[string]interface{}{
        "request_id": "abc-123",
        "user_action": "purchase",
    })
    sentry.CaptureMessage("User completed purchase")
})
```

## What Gets Captured

### HTTP Errors
- 4xx and 5xx HTTP responses
- Request details (method, path, headers, parameters)
- Response status codes
- Client IP addresses
- Request timing information

### gRPC Errors
- Failed gRPC calls with error codes
- Method names and request context
- gRPC status codes and messages
- Request timing information

### Application Errors
- Unhandled panics
- Explicitly captured errors
- Custom messages and events

## Environment-Specific Configuration

### Development
```yaml
sentry:
  enabled: true
  dsn: "https://dev-dsn@sentry.io/dev-project"
  environment: "development"
  debug: true
  sample_rate: 1.0
  traces_sample_rate: 1.0  # Higher sampling for debugging
```

### Production
```yaml
sentry:
  enabled: true
  dsn: "https://prod-dsn@sentry.io/prod-project"
  environment: "production"
  debug: false
  sample_rate: 1.0         # Capture all errors
  traces_sample_rate: 0.1  # Lower sampling to reduce overhead
  tags:
    version: "v1.2.3"
    datacenter: "us-west-2"
```

### Staging
```yaml
sentry:
  enabled: true
  dsn: "https://staging-dsn@sentry.io/staging-project"
  environment: "staging"
  sample_rate: 1.0
  traces_sample_rate: 0.5  # Medium sampling for testing
```

## Performance Considerations

- **Error Sampling**: Set `sample_rate` to less than 1.0 if you have high error volumes
- **Trace Sampling**: Keep `traces_sample_rate` low (0.1 or less) to minimize performance impact
- **Graceful Degradation**: If Sentry fails to initialize, the server continues normally
- **Async Processing**: Error reporting happens asynchronously and won't block requests

## Security Best Practices

1. **Sensitive Data**: Sentry automatically scrubs common sensitive fields, but review your error data
2. **DSN Security**: Keep your DSN secure - it's like an API key
3. **Environment Variables**: Use environment variables for DSN in production:
   ```yaml
   sentry:
     dsn: "${SENTRY_DSN}"
   ```

## Troubleshooting

### Sentry Not Receiving Events

1. **Check DSN**: Verify your DSN is correct and the project exists
2. **Check Network**: Ensure your server can reach sentry.io
3. **Check Sampling**: Verify your sampling rates aren't filtering out events
4. **Check Debug Logs**: Enable `debug: true` to see Sentry debug output

### High Performance Impact

1. **Lower trace sampling**: Reduce `traces_sample_rate`
2. **Reduce error sampling**: Reduce `sample_rate` if you have high error volumes
3. **Check integration overhead**: Monitor CPU and memory usage

### Missing Context

1. **Add custom tags**: Use the `tags` configuration for better categorization
2. **Use manual capture**: Add manual `sentry.CaptureException()` calls for important errors
3. **Configure user context**: Add user identification in your handlers

## Advanced Configuration

### Custom Error Filtering

Currently, the framework captures all errors. For custom filtering, you can:

1. Modify the `BeforeSend` callback in the Sentry configuration
2. Use Sentry's project settings to filter events
3. Implement custom middleware to skip certain error types

### Integration with Logging

The Sentry integration works alongside the framework's existing logging. Errors are:
1. Logged normally via the framework's logger
2. Sent to Sentry for alerting and tracking
3. Available in both systems for comprehensive debugging

## Example Projects

See the `examples/` directory for complete working examples with Sentry integration enabled.

## Support

For issues specific to the Swit framework Sentry integration, please open an issue in the [GitHub repository](https://github.com/innovationmech/swit/issues).

For Sentry-specific questions, refer to the [Sentry documentation](https://docs.sentry.io/).
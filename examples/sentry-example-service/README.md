# Sentry Example Service

This example demonstrates how to integrate Sentry error monitoring and performance tracking with the Swit microservice framework.

## Features Demonstrated

- **Error Capture**: Automatic capture of HTTP 5xx errors and exceptions
- **Panic Recovery**: Automatic panic recovery with Sentry reporting
- **Performance Monitoring**: Request tracing and performance metrics
- **Custom Events**: Manual Sentry event creation and context management
- **User Context**: Setting user information for error reporting
- **Breadcrumbs**: Adding custom breadcrumbs for debugging
- **Transaction Tracking**: Performance monitoring with spans

## Setup

### 1. Set Environment Variables

To enable Sentry integration, set your Sentry DSN:

```bash
export SENTRY_DSN="https://your-dsn@sentry.io/your-project-id"
```

If `SENTRY_DSN` is not set, the service will run without Sentry integration.

### 2. Run the Service

```bash
cd examples/sentry-example-service
go run main.go
```

The service will start on:
- HTTP: `http://localhost:8090`
- gRPC: `localhost:9090`

## Example Endpoints

### Health Check (Ignored by Sentry)
```bash
curl http://localhost:8090/health
```

### Successful Response (Not captured)
```bash
curl http://localhost:8090/api/v1/success
```

### Server Error (Captured by Sentry)
```bash
curl http://localhost:8090/api/v1/error
```

### Custom Sentry Event
```bash
curl http://localhost:8090/api/v1/custom-sentry
```

### User Creation with Context
```bash
curl -X POST http://localhost:8090/api/v1/users
```

### Panic Demonstration (Use with caution)
```bash
curl http://localhost:8090/api/v1/panic
```

## Configuration Options

The example demonstrates various Sentry configuration options:

```go
config.Sentry.Enabled = true                    // Enable/disable Sentry
config.Sentry.DSN = "your-sentry-dsn"          // Sentry project DSN
config.Sentry.Environment = "development"       // Environment tag
config.Sentry.SampleRate = 1.0                 // Error sampling rate (0.0-1.0)
config.Sentry.TracesSampleRate = 0.1           // Performance sampling rate
config.Sentry.Debug = true                     // Debug mode
config.Sentry.AttachStacktrace = true          // Include stack traces
config.Sentry.EnableTracing = true             // Performance monitoring
config.Sentry.HTTPIgnorePaths = []string{      // Paths to ignore
    "/health", "/metrics"
}
config.Sentry.HTTPIgnoreStatusCode = []int{    // Status codes to ignore
    400, 401, 403, 404
}
```

## Framework Integration

The Sentry integration works seamlessly with the Swit framework:

1. **Automatic Middleware**: HTTP and gRPC middleware is automatically configured
2. **Lifecycle Management**: Sentry is initialized during server startup and closed during shutdown
3. **Context Propagation**: Request context is automatically propagated to Sentry
4. **Error Classification**: Different error types are automatically classified with appropriate Sentry levels

## What Gets Captured

### Automatically Captured
- HTTP 5xx responses
- gRPC errors (except ignored status codes)
- Panics and unhandled exceptions
- Request metadata (method, URL, headers, IP)

### Manually Captured
- Custom events and messages
- Business logic errors
- Performance transactions
- User actions and context

## Sentry Dashboard

In your Sentry dashboard, you'll see:

1. **Issues**: Captured errors and exceptions with stack traces
2. **Performance**: Request duration and performance metrics
3. **Releases**: Code deployments (if configured)
4. **User Feedback**: User context and custom data

## Best Practices

1. **Environment Configuration**: Use different environments (development, staging, production)
2. **Sampling Rates**: Use lower sampling rates in production to reduce volume
3. **Ignore Patterns**: Configure ignore patterns for expected errors (404s, validation errors)
4. **Custom Context**: Add relevant business context to help with debugging
5. **Performance Monitoring**: Use transactions to track critical user journeys

## Production Considerations

- Set appropriate sampling rates for your traffic volume
- Configure release tracking for better error attribution  
- Set up alerts for critical error thresholds
- Use tags and contexts to organize and filter errors
- Consider privacy implications when logging user data

## Troubleshooting

### Sentry Not Receiving Events

1. Verify `SENTRY_DSN` environment variable is set correctly
2. Check network connectivity to Sentry servers
3. Enable debug mode: `config.Sentry.Debug = true`
4. Check service logs for Sentry initialization messages

### Performance Impact

1. Adjust sampling rates if performance is affected
2. Use async error reporting (enabled by default)
3. Monitor memory usage with high error volumes
4. Consider using Sentry's rate limiting features

## Documentation

For more information about Sentry integration options, see the main framework documentation in `/docs/sentry-integration.md`.

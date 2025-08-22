# Sentry Integration Example

This example demonstrates how to integrate Sentry error monitoring with the swit microservice framework.

## Features Demonstrated

- **Automatic Error Capture**: HTTP and gRPC errors are automatically reported to Sentry
- **Performance Monitoring**: Request tracing and performance metrics collection
- **Panic Recovery**: Panics are captured and reported while maintaining service availability
- **Custom Context**: Request metadata and user context are included in error reports
- **Breadcrumbs**: Automatic breadcrumb collection for better debugging

## Setup

### 1. Get Sentry DSN

1. Create a Sentry account at [sentry.io](https://sentry.io)
2. Create a new project for your Go application
3. Copy the DSN from your project settings

### 2. Configure Environment Variables

```bash
# Required: Sentry Data Source Name
export SENTRY_DSN="https://your-key@sentry.io/your-project-id"

# Optional: Environment and release tracking
export SENTRY_ENVIRONMENT="development"  # or "production", "staging", etc.
export SENTRY_RELEASE="v1.0.0"
export SENTRY_DEBUG="false"  # Set to "true" for verbose Sentry logs
```

### 3. Run the Example

```bash
# Run the service
go run main.go
```

The service will start on:
- HTTP: `http://localhost:8080`
- gRPC: `localhost:9080`

## Testing Sentry Integration

### Test Endpoints

1. **Health Check** (No Sentry event):
   ```bash
   curl http://localhost:8080/health
   ```

2. **Success Response** (No Sentry event):
   ```bash
   curl http://localhost:8080/success
   ```

3. **Client Error** (Sentry warning):
   ```bash
   curl http://localhost:8080/error
   ```
   - Generates a 400 Bad Request
   - Reported to Sentry as a warning-level event

4. **Server Panic** (Sentry fatal error):
   ```bash
   curl http://localhost:8080/panic
   ```
   - Triggers a panic that's caught and reported to Sentry
   - Service remains available (panic is recovered)
   - Reported as a fatal-level event with full stack trace

5. **Slow Request** (Performance monitoring):
   ```bash
   curl http://localhost:8080/slow
   ```
   - Takes 2 seconds to respond
   - Performance data is tracked in Sentry

### What You'll See in Sentry

1. **Error Events**: Errors from `/error` and `/panic` endpoints
2. **Performance Data**: Transaction traces for all requests
3. **Request Context**: HTTP method, URL, headers, IP address
4. **Environment Data**: Server name, release, environment
5. **Breadcrumbs**: Automatic request lifecycle events
6. **Stack Traces**: Full stack traces for panics and errors

## Configuration Options

The service supports comprehensive Sentry configuration:

```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "${SENTRY_ENVIRONMENT:-production}"
  release: "${SENTRY_RELEASE:-v1.0.0}"
  debug: false
  sample_rate: 1.0          # Error sampling rate (0.0 - 1.0)
  traces_sample_rate: 0.1   # Performance tracing rate (0.0 - 1.0)
  profiles_sample_rate: 0.1 # Profiling rate (0.0 - 1.0)
  attach_stacktrace: true
  server_name: "my-service"
  enable_tracing: true
  enable_profiling: false
  flush_timeout: "2s"
  tags:
    component: "swit-framework"
    version: "v1.0.0"
```

### Middleware Configuration

HTTP and gRPC middleware are automatically configured based on the Sentry settings:

```yaml
http:
  middleware:
    enable_sentry: true  # Automatically matches sentry.enabled

grpc:
  interceptors:
    enable_sentry: true  # Automatically matches sentry.enabled
```

## Framework Integration

This example shows how Sentry integrates seamlessly with the swit framework:

1. **Server Configuration**: Sentry settings in `ServerConfig`
2. **Automatic Middleware**: HTTP and gRPC middleware configured automatically
3. **Lifecycle Management**: Sentry initialization and shutdown handled by framework
4. **Error Context**: Rich context automatically attached to all error reports

## Troubleshooting

### Sentry Not Reporting Events

1. **Check DSN**: Ensure `SENTRY_DSN` is set correctly
2. **Check Network**: Verify the service can reach sentry.io
3. **Check Logs**: Look for Sentry initialization messages
4. **Test DSN**: Use `curl` to test the DSN endpoint

### Performance Impact

- **Error Sampling**: Use `sample_rate` to limit error event volume
- **Trace Sampling**: Use `traces_sample_rate` to control performance overhead
- **Async Reporting**: Sentry events are sent asynchronously by default

### Example Error Event

When you trigger the `/error` endpoint, you'll see something like this in Sentry:

```
BadRequestError: example validation error: missing required parameter

Request:
  Method: GET
  URL: http://localhost:8080/error
  Headers: {...}
  
Environment:
  Server: sentry-example-service
  Release: v1.0.0
  Environment: development
  
Stack Trace:
  main.(*ExampleHTTPHandler).errorResponse
    examples/sentry-example-service/main.go:95
  ...
```

This demonstrates the rich context and debugging information automatically captured by the swit framework's Sentry integration.
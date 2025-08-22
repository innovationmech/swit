# Sentry Integration Example

This example demonstrates how to integrate Sentry error monitoring with the Swit microservice framework.

## Overview

This example shows:
- How to configure Sentry in your server configuration
- Automatic error capture for HTTP requests
- Automatic panic capture
- Proper Sentry initialization and shutdown

## Configuration

```go
config := server.NewServerConfig()
config.ServiceName = "my-service"

// Enable Sentry monitoring
config.Sentry.Enabled = true
config.Sentry.DSN = "https://your-dsn@your-org.sentry.io/your-project"
config.Sentry.Environment = "production"
config.Sentry.Release = "v1.0.0"
config.Sentry.SampleRate = 1.0        // Capture 100% of errors
config.Sentry.TracesSampleRate = 0.1  // Capture 10% of performance traces
```

## Features Demonstrated

### Automatic Error Capture
- HTTP 4xx and 5xx errors are automatically sent to Sentry
- Request context (method, path, headers) is included
- No code changes needed in your handlers

### Panic Recovery
- Panics in HTTP handlers are captured and sent to Sentry
- Stack traces are automatically included
- Server continues to operate after recovery

### Performance Monitoring
- Optional performance trace sampling
- Request timing and throughput metrics
- Configurable sampling rates to control overhead

## Running the Example

1. **Get your Sentry DSN**:
   - Sign up at [sentry.io](https://sentry.io)
   - Create a new project
   - Copy your DSN from project settings

2. **Update the configuration**:
   ```go
   config.Sentry.Enabled = true
   config.Sentry.DSN = "your-actual-dsn-here"
   ```

3. **Run the example**:
   ```bash
   go run main.go
   ```

4. **Test error capture**:
   - Visit `http://localhost:8080/success` - Normal request (not sent to Sentry)
   - Visit `http://localhost:8080/error` - HTTP 500 error (sent to Sentry)
   - Visit `http://localhost:8080/panic` - Panic (captured and sent to Sentry)

## Production Considerations

### Security
- Keep your DSN secure - treat it like an API key
- Use environment variables for sensitive configuration
- Review error data for sensitive information

### Performance
- Start with lower trace sampling rates (0.1 or less) in production
- Monitor CPU and memory impact
- Adjust sampling based on traffic volume

### Error Filtering
- Use Sentry's project settings to filter noise
- Configure custom tags for better error categorization
- Set up alert rules for critical errors

## Configuration via Environment Variables

```yaml
# config.yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "${ENVIRONMENT}"
  release: "${VERSION}"
  sample_rate: 1.0
  traces_sample_rate: 0.1
```

```bash
export SENTRY_DSN="https://your-dsn@sentry.io/project"
export ENVIRONMENT="production"
export VERSION="v1.0.0"
```

## Learn More

- [Sentry Integration Guide](../../docs/sentry-integration.md)
- [Swit Framework Documentation](../../README.md)
- [Sentry Go SDK Documentation](https://docs.sentry.io/platforms/go/)
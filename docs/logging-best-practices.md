# Swit Framework - Structured Logging Best Practices

## Overview

The Swit microservice framework enforces structured logging throughout all components using the Zap logger. This document outlines best practices for implementing and using structured logging in your services.

## Core Principles

### 1. Always Use Structured Logging
- **Never use** `fmt.Print`, `fmt.Println`, `log.Print`, or similar unstructured logging
- **Always use** the framework's logger with structured fields
- **Reason**: Structured logs are machine-readable, searchable, and provide better observability

### 2. Use Appropriate Log Levels

```go
// Debug - Verbose information for debugging
logger.GetLogger().Debug("Processing request details",
    zap.String("user_id", userID),
    zap.Any("payload", requestPayload))

// Info - Normal operational messages
logger.GetLogger().Info("Service started",
    zap.String("version", version),
    zap.String("environment", env))

// Warn - Warning conditions that should be reviewed
logger.GetLogger().Warn("Rate limit approaching",
    zap.Int("current", current),
    zap.Int("limit", limit))

// Error - Error conditions that need attention
logger.GetLogger().Error("Database connection failed",
    zap.Error(err),
    zap.String("database", dbName))

// Fatal - Critical errors that require immediate termination
logger.GetLogger().Fatal("Configuration invalid",
    zap.Error(err))
```

## Configuration

### Server Configuration

The framework automatically configures logging based on your `ServerConfig`:

```yaml
logging:
  level: info                    # debug, info, warn, error, dpanic, panic, fatal
  development: false              # Enable development mode
  encoding: json                  # json or console
  output_paths:
    - stdout
    - /var/log/swit/app.log
  error_output_paths:
    - stderr
    - /var/log/swit/errors.log
  disable_caller: false           # Include caller information
  disable_stacktrace: false       # Include stack traces for errors
  sampling_enabled: true          # Enable sampling for high-volume logs
  sampling_initial: 100           # Log first N occurrences
  sampling_thereafter: 100        # Then log every Nth occurrence
```

### Programmatic Configuration

```go
import "github.com/innovationmech/swit/pkg/logger"

// Initialize with custom configuration
config := &logger.LoggerConfig{
    Level:       "debug",
    Development: true,
    Encoding:    "console",
}
logger.InitLoggerWithConfig(config)
```

## Common Patterns

### 1. Request/Response Logging

```go
// Use structured fields for HTTP requests
logger.LogRequest(
    method,
    path,
    statusCode,
    duration,
    clientIP,
)

// Or manually with fields
logger.GetLogger().Info("HTTP request completed",
    zap.String("method", method),
    zap.String("path", path),
    zap.Int("status", statusCode),
    zap.Float64("latency_ms", latency),
    zap.String("client_ip", clientIP),
    zap.String("request_id", requestID))
```

### 2. Context-Aware Logging

```go
// Add context to all logs in a request
ctx = context.WithValue(ctx, logger.ContextKeyRequestID, requestID)
ctx = context.WithValue(ctx, logger.ContextKeyUserID, userID)

// Use context-aware logger
logger.WithContext(ctx).Info("Processing user request",
    zap.String("action", "update_profile"))
```

### 3. Performance Logging

```go
// Log performance metrics
logger.LogPerformance(
    "database_query",
    duration,
    success,
    map[string]interface{}{
        "query": queryName,
        "rows":  rowCount,
    },
)
```

### 4. Error Logging with Context

```go
// Always include error context
if err != nil {
    logger.GetLogger().Error("Operation failed",
        zap.Error(err),
        zap.String("operation", "user_creation"),
        zap.String("user_email", email),
        zap.Time("timestamp", time.Now()),
        zap.Stack("stacktrace")) // Include stack trace for debugging
}
```

### 5. Service Lifecycle Logging

```go
// Service startup
logger.GetLogger().Info("Service starting",
    zap.String("service", serviceName),
    zap.String("version", version),
    zap.String("build", buildID),
    zap.String("environment", environment))

// Service shutdown
logger.GetLogger().Info("Service shutting down",
    zap.String("reason", reason),
    zap.Duration("uptime", uptime))
```

## Middleware Integration

### HTTP Middleware

The framework provides configurable HTTP request logging:

```go
// Configure request logger middleware
config := &middleware.RequestLoggerConfig{
    SkipPaths:          []string{"/health", "/metrics"},
    LogRequestBody:     false,
    LogResponseBody:    false,
    MaxBodyLogSize:     1024,
    IncludeQueryParams: true,
}

// Apply middleware
router.Use(middleware.RequestLoggerWithConfig(config))
```

### gRPC Interceptors

The framework provides gRPC logging interceptors:

```go
// Configure gRPC logging
config := &middleware.GRPCLoggingConfig{
    SkipMethods:    []string{"/grpc.health.v1.Health/Check"},
    LogPayload:     false,
    MaxPayloadSize: 1024,
}

// Apply interceptor
grpc.UnaryInterceptor(middleware.GRPCLoggingInterceptorWithConfig(config))
```

## Best Practices

### 1. Use Structured Fields

```go
// ✅ Good - Structured fields
logger.GetLogger().Info("User created",
    zap.String("user_id", userID),
    zap.String("email", email),
    zap.Time("created_at", time.Now()))

// ❌ Bad - String concatenation
logger.GetLogger().Info(fmt.Sprintf("User %s with email %s created", userID, email))
```

### 2. Be Consistent with Field Names

```go
// Use consistent field names across your application
zap.String("user_id", userID)      // Always use "user_id"
zap.String("request_id", reqID)    // Always use "request_id"
zap.String("trace_id", traceID)    // Always use "trace_id"
```

### 3. Don't Log Sensitive Information

```go
// ✅ Good - Mask sensitive data
logger.GetLogger().Info("User authenticated",
    zap.String("user_id", userID),
    zap.String("email", email))

// ❌ Bad - Logging passwords or tokens
logger.GetLogger().Info("User authenticated",
    zap.String("password", password),  // Never log passwords!
    zap.String("token", token))         // Never log tokens!
```

### 4. Use Appropriate Log Levels

```go
// Debug - Development and debugging only
logger.GetLogger().Debug("Detailed trace information")

// Info - Normal flow, important events
logger.GetLogger().Info("Service started successfully")

// Warn - Potentially harmful situations
logger.GetLogger().Warn("Deprecated API endpoint called")

// Error - Error events but application continues
logger.GetLogger().Error("Failed to send email, will retry")

// Fatal - Severe errors that cause application exit
logger.GetLogger().Fatal("Cannot connect to database")
```

### 5. Include Request IDs

Always include request IDs for traceability:

```go
// Generate request ID
requestID := uuid.New().String()

// Include in all logs for this request
logger.GetLogger().Info("Processing request",
    zap.String("request_id", requestID),
    zap.String("endpoint", endpoint))
```

### 6. Log at Service Boundaries

```go
// Log when entering a service method
logger.GetLogger().Info("Service method started",
    zap.String("method", "CreateUser"),
    zap.String("request_id", requestID))

// Log when exiting
defer func() {
    logger.GetLogger().Info("Service method completed",
        zap.String("method", "CreateUser"),
        zap.String("request_id", requestID),
        zap.Duration("duration", time.Since(start)))
}()
```

## Performance Considerations

### 1. Use Sampling for High-Volume Logs

```yaml
logging:
  sampling_enabled: true
  sampling_initial: 100      # Log first 100
  sampling_thereafter: 100   # Then every 100th
```

### 2. Avoid Expensive Operations in Log Statements

```go
// ✅ Good - Only compute if logging
if logger.GetLogger().Core().Enabled(zapcore.DebugLevel) {
    expensive := computeExpensiveDebugInfo()
    logger.GetLogger().Debug("Debug info",
        zap.Any("expensive_data", expensive))
}

// ❌ Bad - Always computes even if not logging
logger.GetLogger().Debug("Debug info",
    zap.Any("expensive_data", computeExpensiveDebugInfo()))
```

### 3. Use Lazy Evaluation with Sugared Logger

```go
sugar := logger.GetSugaredLogger()

// Lazy evaluation with sugared logger
sugar.Debugw("Processing items",
    "count", len(items),
    "items", items) // Only serialized if debug enabled
```

## Testing with Logs

### 1. Use Test Logger in Tests

```go
func TestMyFunction(t *testing.T) {
    // Initialize test logger
    logger.InitLogger()
    defer logger.ResetLogger()
    
    // Your test code
    result := MyFunction()
    
    // Assertions
    assert.NotNil(t, result)
}
```

### 2. Capture Logs in Tests

```go
import "go.uber.org/zap/zaptest/observer"

func TestWithLogCapture(t *testing.T) {
    // Create observer to capture logs
    core, recorded := observer.New(zapcore.InfoLevel)
    logger.Logger = zap.New(core)
    
    // Run code that logs
    MyFunction()
    
    // Assert on captured logs
    logs := recorded.All()
    assert.Equal(t, 1, len(logs))
    assert.Equal(t, "expected message", logs[0].Message)
}
```

## Troubleshooting

### 1. Logs Not Appearing

Check your log level configuration:
```go
// Ensure level is set appropriately
config.Logging.Level = "debug" // Will show all logs
```

### 2. Performance Issues

Enable sampling for high-volume logs:
```go
config.Logging.SamplingEnabled = true
config.Logging.SamplingInitial = 10
config.Logging.SamplingThereafter = 100
```

### 3. Large Log Files

Configure log rotation (external tool) or use appropriate log levels:
```bash
# Use logrotate for file rotation
/var/log/swit/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
}
```

## Migration Guide

### Migrating from fmt.Print/log Package

```go
// Before
fmt.Printf("Starting server on port %d\n", port)
log.Printf("Error: %v", err)

// After
logger.GetLogger().Info("Starting server",
    zap.Int("port", port))
logger.GetLogger().Error("Operation failed",
    zap.Error(err))
```

### Migrating from Other Loggers

```go
// Before (logrus)
log.WithFields(log.Fields{
    "user": user,
    "action": "login",
}).Info("User logged in")

// After (zap)
logger.GetLogger().Info("User logged in",
    zap.String("user", user),
    zap.String("action", "login"))
```

## Summary

Structured logging is essential for building observable, maintainable microservices. The Swit framework provides:

1. **Unified logging interface** across all components
2. **Automatic context propagation** for request tracing
3. **Performance-optimized** logging with sampling
4. **Configurable middleware** for HTTP and gRPC
5. **Best practices enforcement** through framework design

By following these guidelines, you ensure your services are production-ready with excellent observability and debugging capabilities.
# Sentry Example Service

The Sentry example service demonstrates comprehensive error monitoring and performance tracking integration with the Swit framework. This example shows real-world usage patterns for production monitoring.

## Overview

**Location**: `examples/sentry-example-service/`

**Features**:
- Complete Sentry integration setup
- Error generation endpoints for testing
- Performance monitoring demonstration
- Custom context and tagging examples
- Recovery from panics
- Different error types and scenarios

## Quick Start

### 1. Prerequisites

Set up Sentry DSN:

```bash
# For testing, you can use a mock DSN
export SENTRY_DSN="https://public@sentry.example.com/1"

# For real testing, use your Sentry project DSN
export SENTRY_DSN="https://your-dsn@sentry.io/your-project-id"
```

### 2. Run the Service

```bash
cd examples/sentry-example-service
go run main.go
```

The service starts on port 8080 with the following endpoints available.

## API Endpoints

### Health and Status

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check endpoint |
| `/api/v1/status` | GET | Service status with Sentry info |

### Error Generation (for Testing)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/error/500` | GET | Generate internal server error |
| `/api/v1/error/400` | GET | Generate bad request error (filtered) |
| `/api/v1/error/custom` | POST | Generate custom error with context |
| `/api/v1/panic` | GET | Trigger panic (recovered by framework) |
| `/api/v1/timeout` | GET | Simulate timeout error |

### Performance Testing

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/slow` | GET | Slow endpoint for performance monitoring |
| `/api/v1/database` | GET | Simulate database operations with spans |
| `/api/v1/external` | GET | Simulate external API calls |

### Data Processing

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/process` | POST | Process data with custom transactions |
| `/api/v1/batch` | POST | Batch processing with performance tracking |

## Example Requests

### 1. Test Error Capture

```bash
# Generate 500 error (will be captured)
curl http://localhost:8080/api/v1/error/500

# Generate 400 error (filtered out by default)
curl http://localhost:8080/api/v1/error/400

# Generate custom error with context
curl -X POST http://localhost:8080/api/v1/error/custom \
  -H "Content-Type: application/json" \
  -d '{"user_id": "123", "operation": "payment", "amount": 100.50}'
```

### 2. Test Performance Monitoring

```bash
# Slow endpoint (will create performance transaction)
curl http://localhost:8080/api/v1/slow

# Database simulation (creates database spans)
curl http://localhost:8080/api/v1/database?table=users&count=100

# External API simulation
curl http://localhost:8080/api/v1/external?service=payment-api
```

### 3. Test Panic Recovery

```bash
# Trigger panic (recovered and reported to Sentry)
curl http://localhost:8080/api/v1/panic
```

### 4. Test Custom Transactions

```bash
# Process data with custom transaction tracking
curl -X POST http://localhost:8080/api/v1/process \
  -H "Content-Type: application/json" \
  -d '{"items": ["item1", "item2", "item3"], "priority": "high"}'

# Batch processing
curl -X POST http://localhost:8080/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{"batch_size": 1000, "operation": "transform"}'
```

## Implementation Examples

### Error Handler with Context

```go
func (h *Handler) CustomErrorHandler(c *gin.Context) {
    var req ErrorRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    // Add custom context to Sentry
    sentry.ConfigureScope(func(scope *sentry.Scope) {
        scope.SetTag("operation", req.Operation)
        scope.SetContext("user_context", map[string]interface{}{
            "user_id": req.UserID,
            "amount":  req.Amount,
        })
        scope.SetLevel(sentry.LevelError)
    })
    
    // Simulate business logic error
    err := fmt.Errorf("failed to process %s for user %s: insufficient funds", 
        req.Operation, req.UserID)
    
    // This error will be captured with the context above
    sentry.CaptureException(err)
    
    c.JSON(500, gin.H{
        "error": "Processing failed",
        "code":  "PROCESSING_ERROR",
    })
}
```

### Performance Tracking

```go
func (h *Handler) DatabaseHandler(c *gin.Context) {
    // Start custom transaction
    transaction := sentry.StartTransaction(
        c.Request.Context(),
        "database-operation",
    )
    defer transaction.Finish()
    
    // Add transaction metadata
    transaction.SetTag("table", c.Query("table"))
    transaction.SetData("count", c.Query("count"))
    
    // Simulate database query with span
    dbSpan := transaction.StartChild("database.query")
    dbSpan.SetTag("operation", "SELECT")
    dbSpan.SetTag("table", c.Query("table"))
    
    // Simulate database work
    time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
    
    // Add results to span
    dbSpan.SetData("rows_returned", 42)
    dbSpan.Finish()
    
    // Simulate processing with another span
    processSpan := transaction.StartChild("data.process")
    time.Sleep(time.Duration(20+rand.Intn(50)) * time.Millisecond)
    processSpan.SetData("processed_count", 42)
    processSpan.Finish()
    
    c.JSON(200, gin.H{
        "status": "success",
        "rows":   42,
        "table":  c.Query("table"),
    })
}
```

### Panic Recovery

```go
func (h *Handler) PanicHandler(c *gin.Context) {
    // Add context before panic
    sentry.ConfigureScope(func(scope *sentry.Scope) {
        scope.SetTag("endpoint", "panic-test")
        scope.SetContext("request", map[string]interface{}{
            "method": c.Request.Method,
            "path":   c.Request.URL.Path,
            "ip":     c.ClientIP(),
        })
    })
    
    // This panic will be caught by Sentry middleware
    panic("intentional panic for Sentry testing")
}
```

## Configuration Examples

### Development Configuration

```yaml
# examples/sentry-example-service/config/development.yaml
service_name: "sentry-example-dev"
http:
  port: "8080"

sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "development"
  debug: true                    # Enable debug logging
  sample_rate: 1.0              # Capture all errors
  traces_sample_rate: 1.0       # Capture all traces
  attach_stacktrace: true
  enable_tracing: true
  integrate_http: true
  capture_panics: true
  
  # Custom tags for this example
  tags:
    service_type: "example"
    version: "1.0.0"
    example: "sentry-demo"
```

### Production Configuration

```yaml
# examples/sentry-example-service/config/production.yaml
service_name: "sentry-example"
http:
  port: "8080"

sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "production"
  debug: false
  sample_rate: 1.0              # Capture all errors in example
  traces_sample_rate: 0.1       # 10% performance sampling
  attach_stacktrace: true
  enable_tracing: true
  enable_profiling: true
  
  # Production filtering
  http_ignore_status_codes:
    - 400    # Filter out bad requests
    - 404    # Filter out not found
  
  http_ignore_paths:
    - "/health"
    - "/favicon.ico"
  
  tags:
    service_type: "example"
    datacenter: "us-west"
    team: "platform"
```

## Code Structure

The example service follows the standard Swit framework structure:

```text
examples/sentry-example-service/
├── cmd/
│   └── main.go                 # Service entry point
├── internal/
│   ├── handler/                # HTTP handlers
│   │   ├── error_handler.go    # Error generation endpoints
│   │   ├── performance_handler.go  # Performance testing endpoints
│   │   └── status_handler.go   # Status and health endpoints
│   ├── service/                # Business logic
│   │   └── example_service.go  # Service implementation
│   └── config/                 # Configuration
│       └── config.go           # Configuration structures
├── config/                     # Configuration files
│   ├── development.yaml
│   └── production.yaml
├── Dockerfile                  # Container configuration
├── Makefile                   # Build commands
├── go.mod                     # Go modules
└── README.md                  # Service documentation
```

## Key Implementation Details

### 1. Service Registration

```go
func (s *ExampleService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Register HTTP handlers
    if err := registry.RegisterBusinessHTTPHandler(s.handler); err != nil {
        return err
    }
    
    // Register health checks
    return registry.RegisterBusinessHealthCheck(s)
}
```

### 2. Middleware Integration

The framework automatically applies Sentry middleware when enabled:

```go
// Framework automatically adds:
// - Sentry error capture middleware
// - Sentry performance middleware  
// - Panic recovery with Sentry reporting
```

### 3. Custom Context Usage

```go
func addUserContext(userID string, operation string) {
    sentry.ConfigureScope(func(scope *sentry.Scope) {
        scope.SetTag("user_id", userID)
        scope.SetTag("operation", operation)
        scope.SetContext("business_context", map[string]interface{}{
            "timestamp":    time.Now(),
            "service":      "sentry-example",
            "request_type": "api",
        })
    })
}
```

## Testing Scenarios

### 1. Error Scenarios

Test different types of errors:

```bash
# Test HTTP errors
for code in 400 401 403 404 500 502; do
    curl "http://localhost:8080/api/v1/error/$code"
done

# Test custom errors with context
curl -X POST http://localhost:8080/api/v1/error/custom \
  -d '{"user_id":"test","operation":"payment","amount":50}'
```

### 2. Performance Scenarios

Test performance monitoring:

```bash
# Test different performance patterns
curl "http://localhost:8080/api/v1/slow?duration=500ms"
curl "http://localhost:8080/api/v1/database?table=orders&count=1000"
curl "http://localhost:8080/api/v1/external?service=user-service&timeout=2s"
```

### 3. Panic Recovery

```bash
# Test panic recovery
curl http://localhost:8080/api/v1/panic

# Should return 500 with recovery message
# Panic details sent to Sentry
```

## Monitoring Dashboard

After running the example and generating events, you can:

1. **View Errors**: Check Sentry dashboard for captured errors
2. **Performance Data**: Review transaction traces and spans
3. **Context Information**: See custom tags and context data
4. **Panic Reports**: View panic stack traces and recovery

## Advanced Usage

### Custom Error Types

```go
type BusinessError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Context map[string]interface{} `json:"context"`
}

func (e *BusinessError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (h *Handler) reportBusinessError(err *BusinessError) {
    sentry.WithScope(func(scope *sentry.Scope) {
        scope.SetTag("error_code", err.Code)
        scope.SetLevel(sentry.LevelError)
        for k, v := range err.Context {
            scope.SetExtra(k, v)
        }
        sentry.CaptureException(err)
    })
}
```

### Performance Profiling

```go
func (h *Handler) ProfiledOperation(c *gin.Context) {
    // Enable profiling for this operation
    profilerOptions := sentry.ProfilerOptions{
        Enabled: true,
    }
    
    transaction := sentry.StartTransaction(
        c.Request.Context(),
        "profiled-operation",
    )
    
    // Set profiler options
    transaction.SetData("profiler_enabled", true)
    
    // Perform work that will be profiled
    result := h.expensiveComputation()
    
    transaction.SetData("result_size", len(result))
    transaction.Finish()
}
```

## Troubleshooting

### Common Issues

**Events not appearing in Sentry:**

1. Check DSN configuration
2. Verify network connectivity
3. Enable debug mode
4. Check sampling rates

**Performance traces missing:**

1. Verify `traces_sample_rate` > 0
2. Check if tracing is enabled
3. Ensure performance middleware is active

**Too many events:**

1. Adjust sampling rates
2. Add more filters
3. Filter expected errors (4xx)

### Debug Mode

Enable debug logging:

```yaml
sentry:
  debug: true
```

This will log:
- Event capture attempts
- Sampling decisions
- Transport status
- Configuration validation

## Next Steps

1. **Customize Configuration**: Adapt configuration for your needs
2. **Add Custom Tags**: Implement service-specific tagging
3. **Performance Tuning**: Adjust sampling rates and filters
4. **Integration Testing**: Test with real Sentry project
5. **Production Deployment**: Deploy with production configuration

## Related Examples

- [Simple HTTP Service](/en/examples/simple-http) - Basic HTTP service patterns
- [Full-Featured Service](/en/examples/full-featured) - Complete framework showcase
- [gRPC Service](/en/examples/grpc-service) - gRPC-specific patterns

## Related Guides

- [Error Monitoring Guide](/en/guide/monitoring) - Complete Sentry integration guide
- [Configuration Guide](/en/guide/configuration) - Configuration options
- [Performance Guide](/en/guide/performance) - Performance optimization

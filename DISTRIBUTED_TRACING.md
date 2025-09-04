# Distributed Tracing Implementation Guide

## Overview

This document explains the OpenTelemetry distributed tracing implementation across the SWIT microservices ecosystem, focusing on cross-service communication tracing between `switauth` and `switserve`.

## Architecture

### Components
- **switauth**: Authentication service that validates user credentials
- **switserve**: User management service that provides user data and internal validation endpoints
- **Jaeger**: Distributed tracing backend for collecting and visualizing traces

### Tracing Flow
```text
Client Request → switauth:Login → HTTP Call → switserve:/internal/validate-user → Database → Response
     ↓               ↓                ↓                    ↓                    ↓         ↓
   Trace ID      Parent Span      Child Span          DB Span             Success    Complete
```

## Implementation Details

### 1. Service-Level Tracing

#### switauth Service
- **AuthService.Login**: Main authentication span with credential validation
- **UserClient.ValidateUserCredentials**: HTTP client span for cross-service calls
- **Database operations**: GORM tracing hooks for token storage

#### switserve Service  
- **HTTP Handler**: Server span for `/internal/validate-user` endpoint
- **UserService.GetUserByUsername**: Service-level span for user lookup
- **Database operations**: GORM tracing hooks for user queries

### 2. Cross-Service Communication

#### HTTP Client (switauth → switserve)
```go
// Inject tracing context into HTTP headers
if c.tracingManager != nil {
    c.tracingManager.InjectHTTPHeaders(ctx, req.Header)
}

// Create client span
ctx, span = c.tracingManager.StartSpan(ctx, "HTTP_POST /internal/validate-user",
    tracing.WithSpanKind(oteltrace.SpanKindClient),
    tracing.WithAttributes(
        attribute.String("http.method", "POST"),
        attribute.String("service.name", "swit-serve"),
        attribute.String("operation.type", "validate_user_credentials"),
    ),
)
```

#### HTTP Server (switserve endpoint)
```go
// Extract tracing context from HTTP headers
if uc.tracingManager != nil {
    tracingCtx := uc.tracingManager.ExtractHTTPHeaders(c.Request.Header)
    if tracingCtx != nil {
        ctx = tracingCtx
    }
    
    // Create server span
    ctx, span = uc.tracingManager.StartSpan(ctx, "HTTP_POST /internal/validate-user",
        tracing.WithSpanKind(oteltrace.SpanKindServer),
        tracing.WithAttributes(
            attribute.String("http.method", "POST"),
            attribute.String("http.route", "/internal/validate-user"),
        ),
    )
}
```

### 3. Configuration

#### Jaeger Exporter (swit.yaml & switauth.yaml)
```yaml
tracing:
  enabled: true
  service_name: "swit-serve"  # or "swit-auth"
  sampling:
    type: "traceidratio"
    rate: 1.0  # Full sampling for development
  exporter:
    type: "jaeger"
    endpoint: "http://localhost:14268/api/traces"
    timeout: "10s"
  resource_attributes:
    environment: "development"
    version: "v1.0.0"
    service.namespace: "swit"
  propagators:
    - "tracecontext"
    - "baggage"
```

## Verification Steps

### 1. Start Jaeger
```bash
# Using Docker
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:latest
```

### 2. Build and Start Services
```bash
# Build services
make build-dev

# Start switserve (Terminal 1)
./_output/build/swit-serve/darwin/arm64/swit-serve

# Start switauth (Terminal 2) 
./_output/build/swit-auth/darwin/arm64/swit-auth
```

### 3. Test Distributed Tracing
```bash
# Make authentication request that triggers cross-service call
curl -X POST http://localhost:9001/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }'
```

### 4. Verify in Jaeger UI
1. Open http://localhost:16686
2. Select service: `swit-auth`
3. Find traces with operation: `AuthService.Login`
4. Verify trace shows:
   - **Parent Span**: `AuthService.Login` (swit-auth)
   - **Child Span**: `HTTP_POST /internal/validate-user` (client side)
   - **Child Span**: `HTTP_POST /internal/validate-user` (server side, swit-serve)
   - **Nested Spans**: `UserService.GetUserByUsername` and database operations

## Trace Attributes

### Common Attributes
- `service.name`: Service identifier
- `operation.success`: Success/failure flag
- `user.username`: Username being validated
- `user.id`: User ID (on success)

### HTTP Client Attributes  
- `http.method`: HTTP method (POST)
- `http.url`: Full request URL
- `net.peer.name`: Target service URL
- `http.status_code`: Response status code
- `error.type`: Error classification (network, service_discovery, etc.)

### HTTP Server Attributes
- `http.method`: HTTP method
- `http.route`: Route pattern
- `error.type`: Error classification (validation, user_not_found, etc.)

### Database Attributes (GORM hooks)
- `db.statement`: SQL statement
- `db.operation`: Operation type (SELECT, INSERT, etc.)
- `db.table`: Table name
- `db.rows_affected`: Number of rows affected

## Testing

### Unit Tests
```bash
# Test cross-service client tracing
go test -v ./internal/switauth/client -run TestUserClient_ValidateUserCredentials_WithTracing

# Test service-level tracing  
go test -v ./internal/switserve/service/user/v1 -run TestUserService_.*_WithTracing
go test -v ./internal/switauth/service/auth/v1 -run TestAuthService_.*_WithTracing
```

### Integration Tests
```bash
# Run all tracing tests
make test-advanced TYPE=tracing
```

## Troubleshooting

### Common Issues
1. **Missing trace headers**: Ensure `InjectHTTPHeaders` is called before HTTP requests
2. **Broken trace chains**: Verify context propagation through all service layers
3. **Missing spans**: Check that TracingManager is properly initialized in dependencies
4. **Jaeger connection errors**: Verify Jaeger is running and endpoint is correct

### Debug Logging
Enable debug logging to see tracing operations:
```yaml
# In service config
logging:
  level: debug
```

### Verification Checklist
- [ ] Jaeger UI shows traces from both services
- [ ] Trace spans are properly nested (parent-child relationships)  
- [ ] HTTP headers contain trace context (traceparent)
- [ ] Database operations appear as child spans
- [ ] Error cases are properly traced with error attributes
- [ ] Service names are correctly set in spans

## Performance Considerations

### Sampling
- **Development**: 100% sampling (`rate: 1.0`)
- **Production**: Reduced sampling (`rate: 0.1` = 10%)

### Batch vs Simple Processors
- **Development**: Simple processor for immediate span export
- **Production**: Batch processor for performance

### Resource Usage
- Tracing adds ~1-5% CPU overhead
- Memory usage increases with span buffer size
- Network overhead for span export (Jaeger endpoint)

## Future Enhancements

1. **gRPC Tracing**: Add distributed tracing for gRPC calls
2. **Custom Metrics**: Add business metrics alongside tracing  
3. **Sampling Strategies**: Implement adaptive sampling
4. **Baggage Propagation**: Use baggage for cross-cutting concerns
5. **Service Mesh Integration**: Integrate with Istio/Linkerd for automatic tracing

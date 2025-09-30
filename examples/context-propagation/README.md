# Context Propagation Example

This example demonstrates cross-transport context propagation in the SWIT framework. It shows how to maintain request context (correlation IDs, user information, tenant IDs, etc.) across HTTP, gRPC, and messaging transports.

## Overview

Context propagation is essential for:
- **Distributed tracing**: Track requests across multiple services
- **Request correlation**: Link related operations across different systems
- **User context**: Maintain user identity and permissions throughout the request flow
- **Multi-tenancy**: Preserve tenant isolation across service boundaries

## Features Demonstrated

1. **HTTP to Messaging**: Extract context from HTTP headers and inject into message headers
2. **gRPC to Messaging**: Extract context from gRPC metadata and propagate to messages
3. **Messaging to HTTP**: Extract context from messages and inject into HTTP headers
4. **Gin to Messaging**: Extract context from Gin framework context and propagate to messages
5. **Round-trip Propagation**: Maintain context through multiple transport hops (HTTP → Message → gRPC → Message → HTTP)

## Running the Example

```bash
# From the project root directory
cd examples/context-propagation
go run main.go
```

## Expected Output

The example will demonstrate five scenarios:

### Example 1: HTTP to Messaging
Shows how HTTP headers are extracted and propagated to message headers.

```
HTTP Headers:
  Correlation-ID: http-corr-12345
  User-ID: user-789
  Tenant-ID: tenant-001

Message Headers:
  Correlation-ID: http-corr-12345
  User-ID: user-789
  Tenant-ID: tenant-001
```

### Example 2: gRPC to Messaging
Shows how gRPC metadata is extracted and propagated to message headers.

### Example 3: Messaging to HTTP
Shows how message headers are extracted and propagated to HTTP headers.

### Example 4: Gin to Messaging
Shows how Gin context values and HTTP headers are combined and propagated.

### Example 5: Round-trip Propagation
Demonstrates end-to-end context propagation through multiple transports:
- HTTP Request → Message Queue → gRPC Service → Message Queue → HTTP Webhook

## Context Keys

The following context keys are propagated across transports:

- `X-Correlation-ID` / `x-correlation-id`: Unique identifier for request correlation
- `X-Trace-ID` / `x-trace-id`: Distributed tracing trace ID
- `X-Span-ID` / `x-span-id`: Distributed tracing span ID
- `X-Request-ID` / `x-request-id`: Unique request identifier
- `X-User-ID` / `x-user-id`: User identity
- `X-Tenant-ID` / `x-tenant-id`: Tenant identifier for multi-tenant systems
- `X-Session-ID` / `x-session-id`: Session identifier

## API Reference

### Context Propagator

```go
// Create a standard context propagator
propagator := messaging.NewStandardContextPropagator()

// Extract from HTTP
ctx := propagator.ExtractFromHTTP(ctx, request)

// Inject to Message
propagator.InjectToMessage(ctx, message)
```

### Helper Functions

```go
// HTTP to Message
ctx := messaging.PropagateHTTPToMessage(ctx, request, message)

// Gin to Message
ctx := messaging.PropagateGinToMessage(ginContext, message)

// gRPC to Message
ctx := messaging.PropagateGRPCToMessage(ctx, message)

// Message to HTTP
ctx := messaging.PropagateMessageToHTTP(ctx, message, httpHeader)

// Message to gRPC
ctx, metadata := messaging.PropagateMessageToGRPC(ctx, message)
```

## Integration with Your Service

### HTTP Handler Example

```go
func handleRequest(c *gin.Context) {
    // Create a message to publish
    message := &messaging.Message{
        ID:      "msg-001",
        Topic:   "user.events",
        Payload: []byte(`{"event": "user_created"}`),
    }

    // Propagate context from HTTP to message
    ctx := messaging.PropagateGinToMessage(c, message)

    // Publish the message
    err := publisher.Publish(ctx, message)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"status": "published"})
}
```

### Message Handler Example

```go
func handleMessage(ctx context.Context, message *messaging.Message) error {
    // Extract context from message
    ctx = messaging.GlobalContextPropagator.ExtractFromMessage(ctx, message)

    // Get correlation ID from context
    if correlationID, ok := ctx.Value(messaging.ContextKeyCorrelationID).(string); ok {
        log.Printf("Processing message with correlation ID: %s", correlationID)
    }

    // Call downstream gRPC service with propagated context
    grpcCtx, md := messaging.GlobalContextPropagator.InjectToGRPC(ctx)
    response, err := grpcClient.Process(grpcCtx, request)
    if err != nil {
        return err
    }

    return nil
}
```

### gRPC Interceptor Example

```go
func contextPropagationInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // Extract context from gRPC metadata
        propagator := messaging.NewStandardContextPropagator()
        ctx = propagator.ExtractFromGRPC(ctx)

        // Process the request with propagated context
        return handler(ctx, req)
    }
}
```

## Custom Header Names

You can customize the header names used for context propagation:

```go
propagator := messaging.NewContextPropagatorWithHeaders(map[string]string{
    "trace_id":       "X-B3-TraceId",      // Use Zipkin B3 headers
    "span_id":        "X-B3-SpanId",
    "correlation_id": "X-Request-ID",      // Use Request-ID for correlation
    "user_id":        "X-Auth-User-ID",
    "tenant_id":      "X-Tenant-Code",
})
```

## Best Practices

1. **Always use correlation IDs**: Ensure every request has a unique correlation ID for tracing
2. **Generate IDs at the edge**: Create correlation IDs at the first entry point (API Gateway, Load Balancer)
3. **Preserve context across async operations**: Always propagate context when publishing messages
4. **Use middleware consistently**: Apply context propagation middleware to all transports
5. **Log with context**: Include correlation IDs in all log messages
6. **Handle missing context gracefully**: Don't fail requests if context is missing, generate new IDs

## Related Documentation

- [Middleware Guide](../../docs/middleware-guide.md)
- [Distributed Tracing](../../docs/tracing-user-guide.md)
- [Messaging Documentation](../../pkg/messaging/README.md)

## See Also

- [Correlation Middleware](../../pkg/messaging/middleware/correlation.go)
- [Tracing Middleware](../../pkg/messaging/middleware/tracing.go)
- [Context Management](../../pkg/messaging/context.go)

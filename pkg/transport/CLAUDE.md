# CLAUDE.md - Transport Layer

This file provides guidance to Claude Code (claude.ai/code) when working with the transport layer (`pkg/transport`) in this repository.

## Transport Layer Overview

The transport layer provides a unified abstraction for managing HTTP and gRPC transports with service registration, lifecycle management, and error handling. It implements a pluggable architecture that allows services to register with multiple transport types seamlessly through a centralized coordination system.

## Core Architecture

### NetworkTransport Interface
The base `NetworkTransport` interface defines the contract for all transport implementations:
```go
type NetworkTransport interface {
    Start(ctx context.Context) error    // Start the transport server
    Stop(ctx context.Context) error     // Gracefully stop the transport
    GetName() string                    // Returns transport name ("http" or "grpc")
    GetAddress() string                 // Returns listening address
}
```

### Key Components

#### 1. Transport Coordinator (`transport.go`)
- **Purpose**: Central coordinator for multiple transport instances and service registries
- **Key Features**:
  - Multi-transport lifecycle management via `NetworkTransport` interface
  - Unified service registration across transports through `MultiTransportRegistry`
  - Error aggregation with `MultiError` and `TransportError` types
  - Thread-safe operations with proper mutex handling
  - Service health checking and metadata management

#### 2. Service Handler System (`handler_register.go`)
- **Purpose**: Unified service registration interface for both HTTP and gRPC
- **Core Interface**: `TransportServiceHandler` - services implement this to work with any transport
- **Key Features**:
  - Service metadata management via `HandlerMetadata`
  - Health check integration with timeout handling
  - Service initialization and shutdown lifecycle
  - Registration order preservation in `TransportServiceRegistry`

#### 3. Multi-Transport Registry (`registry_manager.go`)
- **Purpose**: Manages multiple service registries for different transport types
- **Core Class**: `MultiTransportRegistry` - manages transport-specific service registries
- **Key Features**:
  - Per-transport service registry management with `TransportServiceRegistry`
  - Cross-transport service operations and health checking
  - Transport name validation and registry lifecycle management
  - Graceful shutdown with proper error handling and timeout support

#### 4. HTTP Transport (`http.go`)
- **Purpose**: Gin-based HTTP server implementation
- **Key Features**:
  - Configurable address and port
  - Test mode support with dynamic port allocation
  - Ready channel for testing synchronization
  - Service registry integration

#### 5. gRPC Transport (`grpc.go`)
- **Purpose**: gRPC server implementation with advanced configuration
- **Key Features**:
  - Keepalive configuration
  - Reflection and health service support
  - Interceptor chaining (unary and stream)
  - Message size limits
  - Graceful shutdown with timeout handling

## Transport Implementations

### HTTP Transport
- **File**: `http.go`
- **Framework**: Gin HTTP framework
- **Default Port**: 8080
- **Features**:
  - Dynamic port allocation for testing (use port "0")
  - Configurable readiness signaling
  - Automatic service route registration
  - Graceful shutdown with context cancellation

**Configuration Options**:
```go
type HTTPTransportConfig struct {
    Address     string  // Listen address (e.g., ":8080")
    Port        string  // Port override
    TestMode    bool    // Enable test features
    TestPort    string  // Test port override
    EnableReady bool    // Enable ready channel
}
```

### gRPC Transport  
- **File**: `grpc.go`
- **Framework**: Google gRPC
- **Default Port**: 50051
- **Features**:
  - Advanced keepalive configuration
  - Built-in health check service
  - Service reflection support
  - Interceptor chain management
  - Message size limits

**Configuration Options**:
```go
type GRPCTransportConfig struct {
    Address             string                        // Listen address
    EnableKeepalive     bool                         // Enable keepalive
    EnableReflection    bool                         // Enable reflection
    EnableHealthService bool                         // Enable health service
    MaxRecvMsgSize      int                         // Max receive size
    MaxSendMsgSize      int                         // Max send size
    KeepaliveParams     *keepalive.ServerParameters  // Keepalive settings
    UnaryInterceptors   []grpc.UnaryServerInterceptor
    StreamInterceptors  []grpc.StreamServerInterceptor
}
```

## Service Registration Pattern

### TransportServiceHandler Interface
Services must implement the `TransportServiceHandler` interface to work with the transport layer:

```go
type TransportServiceHandler interface {
    RegisterHTTP(router *gin.Engine) error              // Register HTTP routes
    RegisterGRPC(server *grpc.Server) error            // Register gRPC services
    GetMetadata() *HandlerMetadata                      // Service metadata
    GetHealthEndpoint() string                          // Health check endpoint
    IsHealthy(ctx context.Context) (*types.HealthStatus, error) // Health check
    Initialize(ctx context.Context) error               // Service initialization
    Shutdown(ctx context.Context) error                 // Service shutdown
}
```

### HandlerMetadata Structure
Service metadata is managed through the `HandlerMetadata` structure:

```go
type HandlerMetadata struct {
    Name           string   `json:"name"`             // Service name
    Version        string   `json:"version"`          // Service version
    Description    string   `json:"description"`      // Service description
    HealthEndpoint string   `json:"health_endpoint"`  // Health check endpoint
    Tags           []string `json:"tags,omitempty"`   // Optional tags
    Dependencies   []string `json:"dependencies,omitempty"` // Service dependencies
}
```

### Service Registration Workflow
1. **Service Implementation**: Implement `TransportServiceHandler` interface
2. **Transport Registration**: Register with specific transport via `RegisterHandler()` or use convenience methods `RegisterHTTPService()`/`RegisterGRPCService()`
3. **Registry Management**: Services are managed in transport-specific registries via `MultiTransportRegistry`
4. **Lifecycle Management**: Initialization, health checking, and shutdown are handled automatically with proper timeout handling

## Usage Patterns

### Basic Transport Setup
```go
// Create transport coordinator
coordinator := transport.NewTransportCoordinator()

// Create and configure HTTP transport
httpTransport := transport.NewHTTPTransportWithConfig(&transport.HTTPTransportConfig{
    Address: ":9000",
})

// Create and configure gRPC transport  
grpcTransport := transport.NewGRPCTransportWithConfig(&transport.GRPCTransportConfig{
    Address: ":10000",
    EnableReflection: true,
})

// Register transports with coordinator
coordinator.Register(httpTransport)
coordinator.Register(grpcTransport)
```

### Service Registration
```go
// Register service with HTTP transport
coordinator.RegisterHTTPService(myService)

// Register service with gRPC transport  
coordinator.RegisterGRPCService(myService)

// Register service with specific transport
coordinator.RegisterHandler("http", myService)
```

### Lifecycle Management
```go
// Start all transports
err := coordinator.Start(ctx)

// Initialize all services across all transports
err = coordinator.InitializeTransportServices(ctx)

// Bind HTTP endpoints and gRPC services
err = coordinator.BindAllHTTPEndpoints(router)
err = coordinator.BindAllGRPCServices(grpcServer)

// Check health of all services
healthStatus := coordinator.CheckAllServicesHealth(ctx)

// Graceful shutdown
err = coordinator.Stop(30 * time.Second)
```

## Error Handling

### Error Types
- **`TransportError`**: Single transport error with transport name
- **`MultiError`**: Aggregated errors from multiple transports
- **Error Utilities**: `IsStopError()`, `ExtractStopErrors()` for error analysis

### Error Handling Pattern
```go
if err := coordinator.Stop(timeout); err != nil {
    if multiErr, ok := err.(*transport.MultiError); ok {
        // Handle multiple transport errors
        for _, transportErr := range multiErr.Errors {
            log.Printf("Transport %s error: %v", transportErr.TransportName, transportErr.Err)
        }
    }
}

// Check for specific transport errors
if multiErr, ok := err.(*transport.MultiError); ok {
    if httpErr := multiErr.GetErrorByTransport("http"); httpErr != nil {
        log.Printf("HTTP transport error: %v", httpErr.Err)
    }
}

// Use utility functions for error analysis
if transport.IsStopError(err) {
    stopErrors := transport.ExtractStopErrors(err)
    for _, stopErr := range stopErrors {
        log.Printf("Stop error in %s: %v", stopErr.TransportName, stopErr.Err)
    }
}
```

## Testing Support

### HTTP Transport Testing
- **Dynamic Ports**: Use `SetTestPort("0")` for automatic port allocation
- **Ready Channel**: Use `WaitReady()` to wait for server startup
- **Test Mode**: Enable with `TestMode: true` in configuration

### gRPC Transport Testing
- **Dynamic Ports**: Use `SetTestPort("0")` for automatic port allocation
- **Port Calculation**: `CalculateGRPCPort()` helper for port offset calculation
- **Health Checks**: Built-in health service for testing connectivity

## Performance Considerations

### HTTP Transport
- Uses `gin.New()` instead of `gin.Default()` for minimal overhead
- Configurable ready channels to avoid unnecessary goroutines
- Efficient route registration with service registry

### gRPC Transport
- Configurable keepalive parameters for connection management
- Message size limits to prevent memory issues
- Interceptor chaining for middleware efficiency
- Graceful shutdown with timeout handling

## Integration with Base Server

The transport layer is designed to work seamlessly with the base server framework:

1. **Base Server Integration**: `TransportCoordinator` is embedded in `BaseServerImpl`
2. **Service Registrar Pattern**: Services register via `ServiceRegistrar` interface and are adapted to `TransportServiceHandler`
3. **Adapter Pattern**: Transport layer adapters bridge service interfaces between base server and transport layer
4. **Registry Coordination**: Base server coordinates transport lifecycle through `MultiTransportRegistry` and `TransportServiceRegistry`
5. **Unified Service Management**: All service operations (health checks, initialization, shutdown) are unified across transport types

## Best Practices

### Service Implementation
- Always implement proper health checks in `IsHealthy()` with appropriate timeout handling
- Use context cancellation in `Initialize()` and `Shutdown()` methods
- Return meaningful metadata in `GetMetadata()` including service name, version, and dependencies
- Handle both HTTP and gRPC registration gracefully, even if one transport type is not used
- Implement proper error handling and logging in service methods

### Transport Configuration
- Use dynamic ports (port "0") for testing
- Configure appropriate timeouts for production
- Enable reflection only in development environments
- Set reasonable message size limits for gRPC

### Error Handling
- Always check and handle `MultiError` types using type assertions
- Log transport-specific errors for debugging and use `GetErrorByTransport()` for specific error analysis
- Implement graceful degradation when transports fail
- Use context timeouts for all operations, especially initialization and shutdown
- Leverage error utility functions like `IsStopError()` and `ExtractStopErrors()`

### Testing
- Use separate test configurations for HTTP and gRPC transports
- Leverage ready channels for synchronization in HTTP transport testing
- Test both transport types independently with proper isolation
- Verify health check endpoints work correctly with proper timeout handling
- Use dynamic port allocation for concurrent test execution

## Common Issues and Solutions

### Port Conflicts
- **Issue**: Transport fails to start due to port conflicts
- **Solution**: Use dynamic port allocation (port "0") for testing, configure proper ports for production

### Shutdown Timeouts
- **Issue**: Services don't shutdown gracefully within timeout
- **Solution**: Implement proper context handling in service `Shutdown()` methods, increase timeout values

### Service Registration Order
- **Issue**: Services depend on registration order for proper initialization
- **Solution**: Use service dependencies in `HandlerMetadata`, implement proper initialization order in `TransportServiceRegistry`

### Health Check Failures
- **Issue**: Health checks timeout or fail intermittently
- **Solution**: Implement lightweight health checks with appropriate timeouts (default 5 seconds), ensure proper context handling

### Registry Synchronization Issues
- **Issue**: Concurrent access to service registries causing race conditions
- **Solution**: Use proper mutex locking patterns as implemented in `TransportServiceRegistry` and `MultiTransportRegistry`

### Transport Name Validation
- **Issue**: Invalid transport names causing registration failures
- **Solution**: Use `validateTransportName()` function which enforces alphanumeric characters and hyphens only, with length limits
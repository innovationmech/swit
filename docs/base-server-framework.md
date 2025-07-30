# Base Server Framework Documentation

## Overview

The Base Server Framework provides a reusable foundation for building microservices in Go. It abstracts common patterns for transport management (HTTP/gRPC), service discovery, dependency injection, and lifecycle management while maintaining flexibility for service-specific customizations.

## Key Features

- **Unified Transport Management**: Supports both HTTP and gRPC transports with consistent configuration
- **Service Discovery Integration**: Automatic registration/deregistration with Consul
- **Dependency Injection**: Built-in dependency container support with lifecycle management
- **Middleware Support**: Configurable middleware for both HTTP and gRPC transports
- **Health Checks**: Automatic health check endpoints and service monitoring
- **Graceful Shutdown**: Proper resource cleanup and shutdown sequencing
- **Configuration Validation**: Comprehensive configuration validation with clear error messages

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Your Service  │    │  Base Server     │    │ Transport Layer │
│                 │    │                  │    │                 │
│ ServiceRegistrar├────┤ BaseServerImpl   ├────┤ HTTP Transport  │
│                 │    │                  │    │ gRPC Transport  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                       ┌────────┴────────┐
                       │ Discovery Mgr   │
                       │ Dependency Mgr  │
                       │ Middleware Mgr  │
                       └─────────────────┘
```

## Quick Start

### 1. Define Your Service

```go
package main

import (
    "context"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/server"
)

// MyService implements the ServiceRegistrar interface
type MyService struct {
    // Your service dependencies
}

func (s *MyService) RegisterServices(registry server.ServiceRegistry) error {
    // Register HTTP handlers
    httpHandler := &MyHTTPHandler{}
    if err := registry.RegisterHTTPHandler(httpHandler); err != nil {
        return err
    }
    
    // Register gRPC services
    grpcService := &MyGRPCService{}
    if err := registry.RegisterGRPCService(grpcService); err != nil {
        return err
    }
    
    return nil
}

// MyHTTPHandler implements the HTTPHandler interface
type MyHTTPHandler struct{}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    ginRouter.GET("/api/v1/hello", h.handleHello)
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return "my-service"
}

func (h *MyHTTPHandler) handleHello(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "Hello from my service!"})
}
```

### 2. Configure the Server

```go
func main() {
    config := &server.ServerConfig{
        ServiceName: "my-service",
        HTTP: server.HTTPConfig{
            Port:         "8080",
            EnableReady:  true,
            Enabled:      true,
            ReadTimeout:  30 * time.Second,
            WriteTimeout: 30 * time.Second,
            IdleTimeout:  60 * time.Second,
        },
        GRPC: server.GRPCConfig{
            Port:                "9090",
            EnableReflection:    true,
            EnableHealthService: true,
            Enabled:             true,
            MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
            MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
            KeepaliveParams: server.GRPCKeepaliveParams{
                MaxConnectionIdle:     15 * time.Minute,
                MaxConnectionAge:      30 * time.Minute,
                MaxConnectionAgeGrace: 5 * time.Minute,
                Time:                  5 * time.Minute,
                Timeout:               1 * time.Minute,
            },
        },
        ShutdownTimeout: 30 * time.Second,
        Discovery: server.DiscoveryConfig{
            Enabled:     true,
            Address:     "localhost:8500",
            ServiceName: "my-service",
            Tags:        []string{"api", "v1"},
        },
        Middleware: server.MiddlewareConfig{
            EnableCORS:    true,
            EnableLogging: true,
        },
    }
    
    // Create service and dependencies
    service := &MyService{}
    deps := &MyDependencies{} // Implement server.DependencyContainer
    
    // Create and start server
    baseServer, err := server.NewBaseServer(config, service, deps)
    if err != nil {
        log.Fatal("Failed to create server:", err)
    }
    
    ctx := context.Background()
    if err := baseServer.Start(ctx); err != nil {
        log.Fatal("Failed to start server:", err)
    }
    
    // Wait for shutdown signal
    // ... signal handling code ...
    
    // Graceful shutdown
    if err := baseServer.Shutdown(); err != nil {
        log.Error("Error during shutdown:", err)
    }
}
```

## Configuration Reference

### ServerConfig

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `ServiceName` | `string` | Name of the service | Yes |
| `HTTP` | `HTTPConfig` | HTTP transport configuration | No |
| `GRPC` | `GRPCConfig` | gRPC transport configuration | No |
| `ShutdownTimeout` | `time.Duration` | Maximum time to wait for graceful shutdown | Yes |
| `Discovery` | `DiscoveryConfig` | Service discovery configuration | No |
| `Middleware` | `MiddlewareConfig` | Middleware configuration | No |

### HTTPConfig

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `Port` | `string` | HTTP port to listen on | `"8080"` |
| `Enabled` | `bool` | Enable HTTP transport | `false` |
| `EnableReady` | `bool` | Enable `/ready` endpoint | `true` |
| `ReadTimeout` | `time.Duration` | HTTP read timeout | Required |
| `WriteTimeout` | `time.Duration` | HTTP write timeout | Required |
| `IdleTimeout` | `time.Duration` | HTTP idle timeout | Required |

### GRPCConfig

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `Port` | `string` | gRPC port to listen on | `"9090"` |
| `Enabled` | `bool` | Enable gRPC transport | `false` |
| `EnableReflection` | `bool` | Enable gRPC reflection | `true` |
| `EnableHealthService` | `bool` | Enable gRPC health service | `true` |
| `MaxRecvMsgSize` | `int` | Maximum receive message size | Required |
| `MaxSendMsgSize` | `int` | Maximum send message size | Required |
| `KeepaliveParams` | `GRPCKeepaliveParams` | Keepalive parameters | Required |

### DiscoveryConfig

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `Enabled` | `bool` | Enable service discovery | `false` |
| `Address` | `string` | Consul address | `"localhost:8500"` |
| `ServiceName` | `string` | Service name in discovery | Same as `ServiceName` |
| `Tags` | `[]string` | Service tags | `[]` |

## Interfaces

### ServiceRegistrar

Your service must implement this interface to register handlers:

```go
type ServiceRegistrar interface {
    RegisterServices(registry ServiceRegistry) error
}
```

### HTTPHandler

HTTP handlers must implement this interface:

```go
type HTTPHandler interface {
    RegisterRoutes(router interface{}) error
    GetServiceName() string
}
```

### GRPCService

gRPC services must implement this interface:

```go
type GRPCService interface {
    RegisterGRPC(server interface{}) error
    GetServiceName() string
}
```

### DependencyContainer

Optional dependency injection container:

```go
type DependencyContainer interface {
    Close() error
    GetService(name string) (interface{}, error)
}
```

## Advanced Usage

### Custom Middleware

```go
type MyService struct{}

func (s *MyService) RegisterServices(registry server.ServiceRegistry) error {
    // Register custom HTTP middleware
    httpHandler := &MyHTTPHandler{
        customMiddleware: []gin.HandlerFunc{
            gin.Logger(),
            gin.Recovery(),
            s.authMiddleware(),
        },
    }
    return registry.RegisterHTTPHandler(httpHandler)
}

func (s *MyService) authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Custom authentication logic
        c.Next()
    }
}
```

### Health Checks

```go
type MyHealthCheck struct{}

func (h *MyHealthCheck) Check(ctx context.Context) error {
    // Perform health check logic
    // Return error if unhealthy
    return nil
}

func (h *MyHealthCheck) GetServiceName() string {
    return "my-service"
}

// Register health check
func (s *MyService) RegisterServices(registry server.ServiceRegistry) error {
    healthCheck := &MyHealthCheck{}
    return registry.RegisterHealthCheck(healthCheck)
}
```

### Dependency Injection

```go
type MyDependencies struct {
    db     *sql.DB
    cache  *redis.Client
    closed bool
}

func (d *MyDependencies) GetService(name string) (interface{}, error) {
    switch name {
    case "db":
        return d.db, nil
    case "cache":
        return d.cache, nil
    default:
        return nil, fmt.Errorf("service %s not found", name)
    }
}

func (d *MyDependencies) Close() error {
    if d.closed {
        return nil
    }
    
    if d.db != nil {
        d.db.Close()
    }
    if d.cache != nil {
        d.cache.Close()
    }
    
    d.closed = true
    return nil
}
```

## Best Practices

### 1. Configuration Management

- Use environment variables for configuration
- Validate configuration early in the startup process
- Provide sensible defaults for optional settings

```go
func loadConfig() *server.ServerConfig {
    config := &server.ServerConfig{
        ServiceName: getEnv("SERVICE_NAME", "my-service"),
        HTTP: server.HTTPConfig{
            Port:    getEnv("HTTP_PORT", "8080"),
            Enabled: getBoolEnv("HTTP_ENABLED", true),
        },
        // ... other configuration
    }
    
    if err := config.Validate(); err != nil {
        log.Fatal("Invalid configuration:", err)
    }
    
    return config
}
```

### 2. Error Handling

- Use structured logging for consistent error reporting
- Implement proper error wrapping with context
- Handle partial failures gracefully

```go
func (s *MyService) RegisterServices(registry server.ServiceRegistry) error {
    if err := registry.RegisterHTTPHandler(s.httpHandler); err != nil {
        return fmt.Errorf("failed to register HTTP handler: %w", err)
    }
    
    if err := registry.RegisterGRPCService(s.grpcService); err != nil {
        return fmt.Errorf("failed to register gRPC service: %w", err)
    }
    
    return nil
}
```

### 3. Testing

- Use the integration test patterns provided
- Mock external dependencies
- Test both happy path and error scenarios

```go
func TestMyService(t *testing.T) {
    config := &server.ServerConfig{
        ServiceName: "test-service",
        HTTP: server.HTTPConfig{
            Port:    "0", // Use random port
            Enabled: true,
        },
        Discovery: server.DiscoveryConfig{
            Enabled: false, // Disable for tests
        },
    }
    
    service := &MyService{}
    deps := &MockDependencies{}
    
    baseServer, err := server.NewBaseServer(config, service, deps)
    require.NoError(t, err)
    
    ctx := context.Background()
    err = baseServer.Start(ctx)
    require.NoError(t, err)
    defer baseServer.Shutdown()
    
    // Test your service endpoints
    // ...
}
```

### 4. Monitoring and Observability

- Implement comprehensive health checks
- Use structured logging consistently
- Add metrics collection where appropriate

```go
type MyService struct {
    metrics *prometheus.Registry
    logger  *zap.Logger
}

func (s *MyService) handleRequest(c *gin.Context) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        s.logger.Info("Request processed",
            zap.String("method", c.Request.Method),
            zap.String("path", c.Request.URL.Path),
            zap.Duration("duration", duration),
        )
    }()
    
    // Handle request
}
```

## Migration Guide

See [Migration Guide](migration-guide.md) for detailed instructions on migrating existing services to use the base server framework.

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   - Ensure ports are not conflicting with other services
   - Use port `"0"` for automatic port allocation in tests

2. **Service Discovery Connection Failed**
   - Verify Consul is running and accessible
   - Check network connectivity and firewall settings
   - Service will continue to run without discovery if it fails

3. **Configuration Validation Errors**
   - Check all required fields are provided
   - Ensure timeout values are positive
   - Verify port numbers are valid

4. **gRPC Service Registration Failed**
   - Ensure gRPC transport is enabled
   - Check for duplicate service registrations
   - Verify protobuf definitions are correct

### Debug Logging

Enable debug logging to troubleshoot issues:

```go
import "github.com/innovationmech/swit/pkg/logger"

func main() {
    logger.InitLogger() // Initializes with appropriate log level
    
    // Your server code
}
```

## Examples

See the `examples/` directory for complete working examples:

- [Simple HTTP Service](../examples/simple-http-service/)
- [gRPC Service](../examples/grpc-service/)
- [Full-Featured Service](../examples/full-featured-service/)
- [Service with Discovery](../examples/service-with-discovery/)

## API Reference

For detailed API documentation, see the [API Reference](api-reference.md).
# Simple HTTP Service Example

This example demonstrates how to build a basic HTTP-only service using the Swit framework. It showcases fundamental concepts like service registration, HTTP routing, health checks, and dependency injection in a straightforward implementation.

## Overview

The Simple HTTP Service example (`examples/simple-http-service/`) provides:
- **HTTP-only service** - Focused on HTTP REST API functionality
- **Basic routing** - Essential API endpoints with JSON responses
- **Health monitoring** - Simple health check implementation
- **Environment configuration** - Environment variable support
- **Graceful shutdown** - Proper server lifecycle management

## Key Features

### Service Architecture
- Implements `BusinessServiceRegistrar` for framework integration
- Registers HTTP handlers and health checks
- Uses simple dependency container for basic dependency management
- Demonstrates proper service lifecycle (start/stop/shutdown)

### HTTP Endpoints
- `GET /api/v1/hello?name=YourName` - Greeting endpoint with query parameter
- `GET /api/v1/status` - Service status and health information
- `POST /api/v1/echo` - Echo endpoint that returns request data

### Configuration Features
- Environment variable support for HTTP port
- Optional service discovery integration
- CORS and logging middleware
- Configurable timeouts and server settings

## Code Structure

### Main Service Implementation

```go
// SimpleHTTPService implements the ServiceRegistrar interface
type SimpleHTTPService struct {
    name string
}

func (s *SimpleHTTPService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Register HTTP handler
    httpHandler := &SimpleHTTPHandler{serviceName: s.name}
    if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
        return fmt.Errorf("failed to register HTTP handler: %w", err)
    }

    // Register health check
    healthCheck := &SimpleHealthCheck{serviceName: s.name}
    if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
        return fmt.Errorf("failed to register health check: %w", err)
    }

    return nil
}
```

### HTTP Handler Implementation

```go
type SimpleHTTPHandler struct {
    serviceName string
}

func (h *SimpleHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(gin.IRouter)

    api := ginRouter.Group("/api/v1")
    {
        api.GET("/hello", h.handleHello)
        api.GET("/status", h.handleStatus)
        api.POST("/echo", h.handleEcho)
    }

    return nil
}
```

### Configuration Setup

```go
config := &server.ServerConfig{
    ServiceName: "simple-http-service",
    HTTP: server.HTTPConfig{
        Port:         getEnv("HTTP_PORT", "8080"),
        EnableReady:  true,
        Enabled:      true,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  60 * time.Second,
    },
    GRPC: server.GRPCConfig{
        Enabled: false, // HTTP-only service
    },
    Discovery: server.DiscoveryConfig{
        Enabled: getBoolEnv("DISCOVERY_ENABLED", false),
    },
    Middleware: server.MiddlewareConfig{
        EnableCORS:    true,
        EnableLogging: true,
    },
}
```

## Running the Example

### Prerequisites
- Go 1.24+ installed
- Framework dependencies available

### Quick Start

1. **Navigate to the example directory:**
   ```bash
   cd examples/simple-http-service
   ```

2. **Run the service:**
   ```bash
   go run main.go
   ```

3. **Test the endpoints:**
   ```bash
   # Greeting endpoint
   curl "http://localhost:8080/api/v1/hello?name=Alice"
   
   # Status endpoint  
   curl "http://localhost:8080/api/v1/status"
   
   # Echo endpoint
   curl -X POST "http://localhost:8080/api/v1/echo" \
        -H "Content-Type: application/json" \
        -d '{"message": "Hello, World!"}'
   ```

### Environment Configuration

Configure the service using environment variables:

```bash
# Set custom HTTP port
export HTTP_PORT=9000

# Enable service discovery
export DISCOVERY_ENABLED=true
export CONSUL_ADDRESS=localhost:8500

# Run with custom configuration
go run main.go
```

### Expected Responses

**Hello endpoint:**
```json
{
  "message": "Hello, Alice!",
  "service": "simple-http-service",
  "timestamp": "2025-01-20T10:30:00Z"
}
```

**Status endpoint:**
```json
{
  "status": "healthy",
  "service": "simple-http-service", 
  "timestamp": "2025-01-20T10:30:00Z",
  "uptime": "running"
}
```

**Echo endpoint:**
```json
{
  "echo": "Hello, World!",
  "service": "simple-http-service",
  "timestamp": "2025-01-20T10:30:00Z"
}
```

## Development Patterns

### Adding New Endpoints

1. **Add handler method:**
   ```go
   func (h *SimpleHTTPHandler) handleNewEndpoint(c *gin.Context) {
       c.JSON(http.StatusOK, gin.H{
           "message": "New endpoint response",
           "service": h.serviceName,
       })
   }
   ```

2. **Register in routes:**
   ```go
   func (h *SimpleHTTPHandler) RegisterRoutes(router interface{}) error {
       ginRouter := router.(gin.IRouter)
       api := ginRouter.Group("/api/v1")
       {
           api.GET("/hello", h.handleHello)
           api.GET("/status", h.handleStatus) 
           api.POST("/echo", h.handleEcho)
           api.GET("/new", h.handleNewEndpoint) // Add new route
       }
       return nil
   }
   ```

### Error Handling Pattern

```go
func (h *SimpleHTTPHandler) handleWithValidation(c *gin.Context) {
    var request struct {
        Field string `json:"field" binding:"required"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Invalid request",
            "details": err.Error(),
        })
        return
    }

    // Process valid request...
    c.JSON(http.StatusOK, gin.H{"success": true})
}
```

### Dependency Integration

```go
// Extend dependency container
deps := NewSimpleDependencyContainer()
deps.AddService("config", config)
deps.AddService("version", "1.0.0")
deps.AddService("database", databaseConnection)

// Use in handlers
func (h *SimpleHTTPHandler) handleWithDependency(c *gin.Context) {
    db, err := h.deps.GetService("database")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database unavailable"})
        return
    }
    
    // Use database...
}
```

## Best Practices Demonstrated

1. **Clean Architecture** - Separation of concerns between service, handlers, and dependencies
2. **Error Handling** - Proper HTTP status codes and error responses
3. **Configuration Management** - Environment variable support with sensible defaults
4. **Health Monitoring** - Implementing meaningful health checks
5. **Graceful Shutdown** - Proper cleanup on service termination

## Next Steps

After understanding this simple example:

1. **Explore gRPC Integration** - See `grpc-service` example for Protocol Buffer integration
2. **Multi-Transport Support** - Check `full-featured-service` for HTTP + gRPC combination
3. **Production Features** - Implement database connections, authentication, metrics
4. **Advanced Configuration** - Add YAML configuration files and environment-specific settings

This example provides a solid foundation for building HTTP-based microservices with the Swit framework while demonstrating essential patterns and best practices.
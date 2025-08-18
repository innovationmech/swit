# Full-Featured Service Example

This example demonstrates how to build a comprehensive service that supports both HTTP and gRPC transports simultaneously using the Swit framework. It showcases the complete framework capabilities including dual transport support, unified service registration, and production-ready configurations.

## Overview

The Full-Featured Service example (`examples/full-featured-service/`) provides:
- **Dual transport support** - Both HTTP REST API and gRPC services in one application
- **Unified service interface** - Single service implementation serving both protocols
- **Complete framework integration** - Demonstrates all major framework features
- **Production configurations** - Advanced server settings and middleware
- **Comprehensive examples** - Multiple endpoint types and patterns

## Key Features

### Multi-Transport Architecture
- Implements both HTTP and gRPC handlers in a single service
- Unified business logic serving different transport protocols
- Protocol-specific adapters for optimal client experience
- Shared dependency injection and configuration

### Advanced Configuration
- Keepalive parameters for gRPC connections
- HTTP timeout and middleware configuration
- Service discovery integration ready
- Environment-based configuration management

### Protocol Support
- **HTTP REST API** - JSON-based endpoints with RESTful patterns
- **gRPC Services** - Protocol Buffer-based RPC with type safety
- **Health Services** - Standard health checks for both transports
- **Metrics Endpoint** - Service metrics and monitoring data

## Code Structure

### Multi-Transport Service Implementation

```go
// FullFeaturedService implements both HTTP and gRPC transports
type FullFeaturedService struct {
    name string
}

func (s *FullFeaturedService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Register HTTP handler
    httpHandler := &FullFeaturedHTTPHandler{serviceName: s.name}
    if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
        return fmt.Errorf("failed to register HTTP handler: %w", err)
    }

    // Register gRPC service
    grpcService := &FullFeaturedGRPCService{serviceName: s.name}
    if err := registry.RegisterBusinessGRPCService(grpcService); err != nil {
        return fmt.Errorf("failed to register gRPC service: %w", err)
    }

    // Register health check (shared between transports)
    healthCheck := &FullFeaturedHealthCheck{serviceName: s.name}
    if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
        return fmt.Errorf("failed to register health check: %w", err)
    }

    return nil
}
```

### HTTP Handler Implementation

```go
type FullFeaturedHTTPHandler struct {
    serviceName string
}

func (h *FullFeaturedHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(gin.IRouter)

    api := ginRouter.Group("/api/v1")
    {
        // Greeter endpoints (HTTP versions of gRPC methods)
        api.POST("/greet", h.handleGreet)
        
        // Additional HTTP-only endpoints
        api.GET("/status", h.handleStatus)
        api.GET("/metrics", h.handleMetrics)
        api.POST("/echo", h.handleEcho)
    }

    return nil
}

func (h *FullFeaturedHTTPHandler) handleGreet(c *gin.Context) {
    var request struct {
        Name string `json:"name" binding:"required"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Invalid request",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message":   fmt.Sprintf("Hello, %s!", request.Name),
        "service":   h.serviceName,
        "timestamp": time.Now().UTC(),
        "protocol":  "HTTP",
    })
}
```

### gRPC Service Implementation

```go
type FullFeaturedGRPCService struct {
    serviceName string
    interaction.UnimplementedGreeterServiceServer
}

func (s *FullFeaturedGRPCService) RegisterGRPC(server interface{}) error {
    grpcServer := server.(*grpc.Server)
    interaction.RegisterGreeterServiceServer(grpcServer, s)
    return nil
}

func (s *FullFeaturedGRPCService) SayHello(ctx context.Context, req *interaction.SayHelloRequest) (*interaction.SayHelloResponse, error) {
    // Validate request
    if req.GetName() == "" {
        return nil, status.Error(codes.InvalidArgument, "name cannot be empty")
    }

    // Create response (shared business logic)
    response := &interaction.SayHelloResponse{
        Message: fmt.Sprintf("Hello, %s!", req.GetName()),
    }

    return response, nil
}
```

### Advanced Configuration

```go
config := &server.ServerConfig{
    ServiceName: "full-featured-service",
    HTTP: server.HTTPConfig{
        Port:         getEnv("HTTP_PORT", "8080"),
        EnableReady:  true,
        Enabled:      true,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  60 * time.Second,
    },
    GRPC: server.GRPCConfig{
        Port:                getEnv("GRPC_PORT", "9090"),
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
        KeepalivePolicy: server.GRPCKeepalivePolicy{
            MinTime:             5 * time.Minute,
            PermitWithoutStream: false,
        },
    },
    Discovery: server.DiscoveryConfig{
        Enabled:     getBoolEnv("DISCOVERY_ENABLED", false),
        Address:     getEnv("CONSUL_ADDRESS", "localhost:8500"),
        ServiceName: "full-featured-service",
        Tags:        []string{"http", "grpc", "api", "v1"},
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
- gRPC tools installed (for gRPC testing)
- Framework dependencies available

### Quick Start

1. **Navigate to the example directory:**
   ```bash
   cd examples/full-featured-service
   ```

2. **Run the service:**
   ```bash
   go run main.go
   ```

3. **Test HTTP endpoints:**
   ```bash
   # Greeting endpoint
   curl -X POST "http://localhost:8080/api/v1/greet" \
        -H "Content-Type: application/json" \
        -d '{"name": "Alice"}'
   
   # Status endpoint
   curl "http://localhost:8080/api/v1/status"
   
   # Metrics endpoint
   curl "http://localhost:8080/api/v1/metrics"
   ```

4. **Test gRPC endpoints:**
   ```bash
   # SayHello method
   grpcurl -plaintext -d '{"name": "Bob"}' \
           localhost:9090 \
           swit.interaction.v1.GreeterService/SayHello
   
   # Health check
   grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
   ```

### Environment Configuration

```bash
# Configure both transport ports
export HTTP_PORT=8888
export GRPC_PORT=9999

# Enable service discovery
export DISCOVERY_ENABLED=true
export CONSUL_ADDRESS=localhost:8500

# Set environment
export ENVIRONMENT=production

# Run with custom configuration
go run main.go
```

### Expected Responses

**HTTP Greeting:**
```json
{
  "message": "Hello, Alice!",
  "service": "full-featured-service",
  "timestamp": "2025-01-20T10:30:00Z",
  "protocol": "HTTP"
}
```

**HTTP Status:**
```json
{
  "status": "healthy",
  "service": "full-featured-service",
  "timestamp": "2025-01-20T10:30:00Z",
  "uptime": "running",
  "protocols": ["HTTP", "gRPC"]
}
```

**gRPC Response:**
```json
{
  "message": "Hello, Bob!"
}
```

## Development Patterns

### Shared Business Logic

```go
// BusinessLogic encapsulates shared functionality
type BusinessLogic struct {
    serviceName string
    logger      *zap.Logger
}

func (bl *BusinessLogic) ProcessGreeting(name string) (string, error) {
    if name == "" {
        return "", fmt.Errorf("name cannot be empty")
    }
    
    // Shared business logic
    greeting := fmt.Sprintf("Hello, %s!", name)
    bl.logger.Info("Generated greeting", zap.String("name", name))
    
    return greeting, nil
}

// Use in HTTP handler
func (h *FullFeaturedHTTPHandler) handleGreet(c *gin.Context) {
    var request struct {
        Name string `json:"name" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }
    
    message, err := h.businessLogic.ProcessGreeting(request.Name)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message":  message,
        "protocol": "HTTP",
    })
}

// Use in gRPC service
func (s *FullFeaturedGRPCService) SayHello(ctx context.Context, req *interaction.SayHelloRequest) (*interaction.SayHelloResponse, error) {
    message, err := s.businessLogic.ProcessGreeting(req.GetName())
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, err.Error())
    }
    
    return &interaction.SayHelloResponse{Message: message}, nil
}
```

### Protocol-Specific Adapters

```go
// HTTPAdapter converts between HTTP and business logic
type HTTPAdapter struct {
    businessLogic *BusinessLogic
}

func (a *HTTPAdapter) HandleRequest(c *gin.Context, processor func(string) (interface{}, error)) {
    var request struct {
        Name string `json:"name" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid request",
            "details": err.Error(),
        })
        return
    }
    
    result, err := processor(request.Name)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, result)
}

// GRPCAdapter handles gRPC-specific concerns
type GRPCAdapter struct {
    businessLogic *BusinessLogic
}

func (a *GRPCAdapter) ProcessRequest(ctx context.Context, name string) (string, error) {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return "", status.Error(codes.Canceled, "request canceled")
    default:
    }
    
    // Process with business logic
    return a.businessLogic.ProcessGreeting(name)
}
```

### Advanced Health Checking

```go
type FullFeaturedHealthCheck struct {
    serviceName string
    database    *sql.DB
    redis       *redis.Client
}

func (h *FullFeaturedHealthCheck) Check(ctx context.Context) error {
    // Check database connection
    if err := h.database.PingContext(ctx); err != nil {
        return fmt.Errorf("database health check failed: %w", err)
    }
    
    // Check Redis connection
    if err := h.redis.Ping(ctx).Err(); err != nil {
        return fmt.Errorf("redis health check failed: %w", err)
    }
    
    // Check external dependencies
    if err := h.checkExternalService(ctx); err != nil {
        return fmt.Errorf("external service health check failed: %w", err)
    }
    
    return nil
}

func (h *FullFeaturedHealthCheck) checkExternalService(ctx context.Context) error {
    client := &http.Client{Timeout: 3 * time.Second}
    
    req, err := http.NewRequestWithContext(ctx, "GET", "https://api.external.com/health", nil)
    if err != nil {
        return err
    }
    
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("external service returned status %d", resp.StatusCode)
    }
    
    return nil
}
```

## Production Considerations

### Monitoring Integration

```go
// Add metrics collection
func (h *FullFeaturedHTTPHandler) handleMetrics(c *gin.Context) {
    metrics := map[string]interface{}{
        "service":           h.serviceName,
        "requests_total":    100, // From metrics collector
        "errors_total":      5,   // From metrics collector
        "uptime_seconds":    3600, // From uptime tracker
        "memory_usage_mb":   128,  // From runtime stats
        "goroutines":        50,   // runtime.NumGoroutine()
        "protocols":         []string{"HTTP", "gRPC"},
        "timestamp":         time.Now().UTC(),
    }
    
    c.JSON(http.StatusOK, metrics)
}
```

### Service Discovery Integration

```go
// Configure for production service discovery
discovery: server.DiscoveryConfig{
    Enabled:         true,
    Address:         "consul.internal:8500",
    ServiceName:     "full-featured-service",
    Tags:            []string{"http", "grpc", "api", "v1", "production"},
    FailureMode:     server.DiscoveryFailureModeFailFast,
    HealthCheckRequired: true,
}
```

### Security Configuration

```go
// Add security middleware
middleware: server.MiddlewareConfig{
    EnableCORS:     true,
    EnableLogging:  true,
    EnableSecurity: true,
    SecurityConfig: server.SecurityConfig{
        EnableHTTPS:     true,
        TLSCertFile:     "/etc/ssl/service.crt",
        TLSKeyFile:      "/etc/ssl/service.key",
        EnableJWTAuth:   true,
        JWTSecret:       getEnv("JWT_SECRET", ""),
    },
}
```

## Testing Multi-Transport Services

### Integration Testing

```go
func TestFullFeaturedService(t *testing.T) {
    // Start service with test configuration
    config := createTestConfig()
    service := NewFullFeaturedService("test-service")
    srv, err := server.NewBusinessServerCore(config, service, nil)
    require.NoError(t, err)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    err = srv.Start(ctx)
    require.NoError(t, err)
    defer srv.Shutdown()
    
    // Test HTTP endpoint
    httpAddr := srv.GetHTTPAddress()
    resp, err := http.Post(
        fmt.Sprintf("http://localhost%s/api/v1/greet", httpAddr),
        "application/json",
        strings.NewReader(`{"name": "TestUser"}`),
    )
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    // Test gRPC endpoint
    grpcConn, err := grpc.DialContext(ctx, srv.GetGRPCAddress(),
        grpc.WithInsecure(), grpc.WithBlock())
    require.NoError(t, err)
    defer grpcConn.Close()
    
    grpcClient := interaction.NewGreeterServiceClient(grpcConn)
    grpcResp, err := grpcClient.SayHello(ctx, &interaction.SayHelloRequest{
        Name: "TestUser",
    })
    require.NoError(t, err)
    assert.Contains(t, grpcResp.GetMessage(), "TestUser")
}
```

## Best Practices Demonstrated

1. **Transport Abstraction** - Clean separation between transport and business logic
2. **Unified Configuration** - Single configuration for multiple transports
3. **Shared Components** - Reusable health checks and dependency injection
4. **Protocol Optimization** - Transport-specific optimizations and patterns
5. **Production Readiness** - Advanced configuration and monitoring integration

## Next Steps

After understanding this full-featured example:

1. **Database Integration** - Add database connections and repository patterns
2. **Authentication** - Implement JWT-based authentication for both transports
3. **Advanced Monitoring** - Integrate with Prometheus, Jaeger, or similar tools
4. **Deployment** - Containerize with Docker and deploy with Kubernetes

This example demonstrates the complete capabilities of the Swit framework for building production-ready microservices that serve multiple transport protocols efficiently.
# Base Server Framework Documentation

## Overview

The Base Server Framework provides a unified, production-ready foundation for building microservices in the swit ecosystem. It implements a comprehensive server architecture that handles transport management, service registration, dependency injection, configuration management, performance monitoring, and lifecycle coordination.

## Key Features

- **Unified Transport Management**: Supports both HTTP and gRPC transports through the transport coordinator
- **Service Discovery Integration**: Automatic registration/deregistration with Consul via ServiceDiscoveryManager abstraction
- **Dependency Injection**: Factory-based dependency container with lifecycle management and singleton/transient support
- **Middleware Support**: Configurable middleware for both HTTP and gRPC transports via MiddlewareManager
- **Performance Monitoring**: Built-in metrics collection, performance profiling, and monitoring hooks
- **Health Checks**: Comprehensive health check system with timeout handling and service aggregation
- **Graceful Shutdown**: Phased shutdown sequencing with proper resource cleanup and error handling
- **Configuration Validation**: Extensive configuration validation with sensible defaults and clear error messages
- **Lifecycle Management**: Structured startup/shutdown phases with automatic cleanup on failure

## Architecture

```
┌─────────────────────┐    ┌──────────────────────┐    ┌─────────────────────┐
│   Your Service      │    │  BusinessServerImpl  │    │ Transport Layer     │
│                     │    │                      │    │                     │
│BusinessServiceRegistrar├─┤ • TransportCoordinator├───┤ HTTP Transport      │
│                     │    │ • ServiceDiscoveryMgr│    │ gRPC Transport      │
└─────────────────────┘    │ • PerformanceMonitor │    └─────────────────────┘
                           │ • LifecycleManager   │
                           └──────────┬───────────┘
                                      │
                           ┌──────────┴───────────┐
                           │ BusinessDependencies │
                           │ MiddlewareManager    │
                           │ ServiceRegistrations │
                           └──────────────────────┘
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

// MyService implements the BusinessServiceRegistrar interface
type MyService struct {
    // Your service dependencies
}

func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Register HTTP handlers
    httpHandler := &MyHTTPHandler{}
    if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
        return err
    }
    
    // Register gRPC services
    grpcService := &MyGRPCService{}
    if err := registry.RegisterBusinessGRPCService(grpcService); err != nil {
        return err
    }
    
    return nil
}

// MyHTTPHandler implements the BusinessHTTPHandler interface
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
    deps := &MyDependencies{} // Implement server.BusinessDependencyContainer
    
    // Create and start server
    baseServer, err := server.NewBusinessServerCore(config, service, deps)
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
| `Address` | `string` | HTTP address to bind | `":8080"` |
| `Enabled` | `bool` | Enable HTTP transport | `true` |
| `EnableReady` | `bool` | Enable ready channel for testing | `true` |
| `TestMode` | `bool` | Enable test mode features | `false` |
| `TestPort` | `string` | Test port override | `""` |
| `ReadTimeout` | `time.Duration` | HTTP read timeout | `30s` |
| `WriteTimeout` | `time.Duration` | HTTP write timeout | `30s` |
| `IdleTimeout` | `time.Duration` | HTTP idle timeout | `120s` |
| `Middleware` | `HTTPMiddleware` | HTTP middleware configuration | See HTTPMiddleware |
| `Headers` | `map[string]string` | Default HTTP headers | `{}` |

### GRPCConfig

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `Port` | `string` | gRPC port to listen on | `"9080"` (HTTP port + 1000) |
| `Address` | `string` | gRPC address to bind | `":9080"` |
| `Enabled` | `bool` | Enable gRPC transport | `true` |
| `EnableKeepalive` | `bool` | Enable keepalive parameters | `true` |
| `EnableReflection` | `bool` | Enable gRPC reflection | `true` |
| `EnableHealthService` | `bool` | Enable gRPC health service | `true` |
| `TestMode` | `bool` | Enable test mode features | `false` |
| `TestPort` | `string` | Test port override | `""` |
| `MaxRecvMsgSize` | `int` | Maximum receive message size | `4MB` |
| `MaxSendMsgSize` | `int` | Maximum send message size | `4MB` |
| `KeepaliveParams` | `GRPCKeepaliveParams` | Keepalive parameters | See GRPCKeepaliveParams |
| `KeepalivePolicy` | `GRPCKeepalivePolicy` | Keepalive enforcement policy | See GRPCKeepalivePolicy |
| `Interceptors` | `GRPCInterceptorConfig` | Interceptor configuration | See GRPCInterceptorConfig |
| `TLS` | `GRPCTLSConfig` | TLS configuration | Disabled by default |

### DiscoveryConfig

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `Enabled` | `bool` | Enable service discovery | `true` |
| `Address` | `string` | Consul address | `"127.0.0.1:8500"` |
| `ServiceName` | `string` | Service name in discovery | Same as `ServiceName` |
| `Tags` | `[]string` | Service tags | `["v1"]` |

### Additional Configuration Structures

#### HTTPMiddleware
| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `EnableCORS` | `bool` | Enable CORS middleware | `true` |
| `EnableAuth` | `bool` | Enable authentication middleware | `false` |
| `EnableRateLimit` | `bool` | Enable rate limiting | `false` |
| `EnableLogging` | `bool` | Enable request logging | `true` |
| `EnableTimeout` | `bool` | Enable request timeout | `true` |
| `CORSConfig` | `CORSConfig` | CORS configuration | See CORSConfig |
| `RateLimitConfig` | `RateLimitConfig` | Rate limiting configuration | See RateLimitConfig |
| `TimeoutConfig` | `TimeoutConfig` | Timeout configuration | See TimeoutConfig |

#### GRPCKeepaliveParams
| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `MaxConnectionIdle` | `time.Duration` | Max idle time before ping | `15s` |
| `MaxConnectionAge` | `time.Duration` | Max connection lifetime | `30s` |
| `MaxConnectionAgeGrace` | `time.Duration` | Grace period for connection closure | `5s` |
| `Time` | `time.Duration` | Ping interval | `5s` |
| `Timeout` | `time.Duration` | Ping timeout | `1s` |

#### GRPCInterceptorConfig
| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `EnableAuth` | `bool` | Enable authentication interceptor | `false` |
| `EnableLogging` | `bool` | Enable logging interceptor | `true` |
| `EnableMetrics` | `bool` | Enable metrics interceptor | `false` |
| `EnableRecovery` | `bool` | Enable panic recovery | `true` |
| `EnableRateLimit` | `bool` | Enable rate limiting | `false` |

## Interfaces

### BusinessServiceRegistrar

Your service must implement this interface to register handlers:

```go
type BusinessServiceRegistrar interface {
    RegisterServices(registry BusinessServiceRegistry) error
}
```

### BusinessServiceRegistry

Registry interface for registering different types of services:

```go
type BusinessServiceRegistry interface {
    RegisterBusinessHTTPHandler(handler BusinessHTTPHandler) error
    RegisterBusinessGRPCService(service BusinessGRPCService) error
    RegisterBusinessHealthCheck(check BusinessHealthCheck) error
}
```

### BusinessHTTPHandler

HTTP handlers must implement this interface:

```go
type BusinessHTTPHandler interface {
    RegisterRoutes(router interface{}) error
    GetServiceName() string
}
```

### BusinessGRPCService

gRPC services must implement this interface:

```go
type BusinessGRPCService interface {
    RegisterGRPC(server interface{}) error
    GetServiceName() string
}
```

### BusinessHealthCheck

Health check services must implement this interface:

```go
type BusinessHealthCheck interface {
    Check(ctx context.Context) error
    GetServiceName() string
}
```

### BusinessDependencyContainer

Basic dependency injection container:

```go
type BusinessDependencyContainer interface {
    Close() error
    GetService(name string) (interface{}, error)
}
```

### BusinessDependencyRegistry

Extended dependency container with registration and lifecycle:

```go
type BusinessDependencyRegistry interface {
    BusinessDependencyContainer
    Initialize(ctx context.Context) error
    RegisterSingleton(name string, factory DependencyFactory) error
    RegisterTransient(name string, factory DependencyFactory) error
    RegisterInstance(name string, instance interface{}) error
    GetDependencyNames() []string
    IsInitialized() bool
    IsClosed() bool
}
```

### BusinessServerCore

Main server interface with lifecycle management:

```go
type BusinessServerCore interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Shutdown() error
    GetHTTPAddress() string
    GetGRPCAddress() string
    GetTransports() []transport.NetworkTransport
    GetTransportStatus() map[string]TransportStatus
    GetTransportHealth(ctx context.Context) map[string]map[string]*types.HealthStatus
}
```

### BusinessServerWithPerformance

Extended server interface with performance monitoring:

```go
type BusinessServerWithPerformance interface {
    BusinessServerCore
    GetPerformanceMetrics() *PerformanceMetrics
    GetUptime() time.Duration
    GetPerformanceMonitor() *PerformanceMonitor
}
```

## Advanced Usage

### Custom Middleware

```go
type MyService struct{}

func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Register HTTP handlers with custom middleware support
    httpHandler := &MyHTTPHandler{
        middleware: s.setupMiddleware(),
    }
    return registry.RegisterBusinessHTTPHandler(httpHandler)
}

func (s *MyService) setupMiddleware() []gin.HandlerFunc {
    return []gin.HandlerFunc{
        gin.Logger(),
        gin.Recovery(),
        s.authMiddleware(),
        s.metricsMiddleware(),
    }
}

func (s *MyService) authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Custom authentication logic
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(401, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }
        c.Next()
    }
}

func (s *MyService) metricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        duration := time.Since(start)
        
        // Record metrics
        metrics.RecordHTTPRequest(c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration)
    }
}
```

### Health Checks

```go
type MyHealthCheck struct {
    db    *sql.DB
    redis *redis.Client
}

func (h *MyHealthCheck) Check(ctx context.Context) error {
    // Check database connectivity
    if err := h.db.PingContext(ctx); err != nil {
        return fmt.Errorf("database health check failed: %w", err)
    }
    
    // Check Redis connectivity
    if err := h.redis.Ping(ctx).Err(); err != nil {
        return fmt.Errorf("redis health check failed: %w", err)
    }
    
    // Additional health checks
    if !h.isExternalServiceHealthy(ctx) {
        return fmt.Errorf("external service health check failed")
    }
    
    return nil
}

func (h *MyHealthCheck) GetServiceName() string {
    return "my-service"
}

func (h *MyHealthCheck) isExternalServiceHealthy(ctx context.Context) bool {
    // Check external dependencies
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get("https://api.external-service.com/health")
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    return resp.StatusCode == http.StatusOK
}

// Register health check
func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
    healthCheck := &MyHealthCheck{
        db:    s.db,
        redis: s.redis,
    }
    return registry.RegisterBusinessHealthCheck(healthCheck)
}
```

### Dependency Injection

#### Using SimpleBusinessDependencyContainer

```go
func setupDependencies() server.BusinessDependencyContainer {
    container := server.NewBusinessDependencyContainerBuilder().
        AddSingleton("database", func(container server.BusinessDependencyContainer) (interface{}, error) {
            config := &DatabaseConfig{
                Host:     os.Getenv("DB_HOST"),
                Port:     os.Getenv("DB_PORT"),
                Username: os.Getenv("DB_USER"),
                Password: os.Getenv("DB_PASS"),
                Database: os.Getenv("DB_NAME"),
            }
            return sql.Open("mysql", config.DSN())
        }).
        AddSingleton("redis", func(container server.BusinessDependencyContainer) (interface{}, error) {
            return redis.NewClient(&redis.Options{
                Addr:     os.Getenv("REDIS_ADDR"),
                Password: os.Getenv("REDIS_PASS"),
                DB:       0,
            }), nil
        }).
        AddTransient("user-repository", func(container server.BusinessDependencyContainer) (interface{}, error) {
            db, err := container.GetService("database")
            if err != nil {
                return nil, fmt.Errorf("failed to get database: %w", err)
            }
            return &UserRepository{DB: db.(*sql.DB)}, nil
        }).
        AddTransient("user-service", func(container server.BusinessDependencyContainer) (interface{}, error) {
            userRepo, err := container.GetService("user-repository")
            if err != nil {
                return nil, fmt.Errorf("failed to get user repository: %w", err)
            }
            
            cache, err := container.GetService("redis")
            if err != nil {
                return nil, fmt.Errorf("failed to get redis: %w", err)
            }
            
            return &UserService{
                Repository: userRepo.(*UserRepository),
                Cache:      cache.(*redis.Client),
            }, nil
        }).
        Build()
    
    return container
}

// Use dependencies in service
type MyService struct {
    deps server.BusinessDependencyContainer
}

func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Get user service from container
    userSvc, err := s.deps.GetService("user-service")
    if err != nil {
        return fmt.Errorf("failed to get user service: %w", err)
    }
    
    httpHandler := &UserHTTPHandler{
        userService: userSvc.(*UserService),
    }
    
    return registry.RegisterBusinessHTTPHandler(httpHandler)
}
```

#### Custom Dependency Container Implementation

```go
type MyDependencies struct {
    db           *sql.DB
    cache        *redis.Client
    userService  *UserService
    initialized  bool
    closed       bool
    mu           sync.RWMutex
}

func (d *MyDependencies) Initialize(ctx context.Context) error {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    if d.initialized {
        return nil
    }
    
    // Initialize database connection
    db, err := sql.Open("mysql", d.getDSN())
    if err != nil {
        return fmt.Errorf("failed to open database: %w", err)
    }
    
    if err := db.PingContext(ctx); err != nil {
        return fmt.Errorf("failed to ping database: %w", err)
    }
    
    d.db = db
    
    // Initialize Redis client
    d.cache = redis.NewClient(&redis.Options{
        Addr: os.Getenv("REDIS_ADDR"),
    })
    
    if err := d.cache.Ping(ctx).Err(); err != nil {
        return fmt.Errorf("failed to ping redis: %w", err)
    }
    
    // Initialize user service
    d.userService = &UserService{
        DB:    d.db,
        Cache: d.cache,
    }
    
    d.initialized = true
    return nil
}

func (d *MyDependencies) GetService(name string) (interface{}, error) {
    d.mu.RLock()
    defer d.mu.RUnlock()
    
    if d.closed {
        return nil, fmt.Errorf("dependency container is closed")
    }
    
    switch name {
    case "database":
        return d.db, nil
    case "cache":
        return d.cache, nil
    case "user-service":
        return d.userService, nil
    default:
        return nil, fmt.Errorf("service %s not found", name)
    }
}

func (d *MyDependencies) Close() error {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    if d.closed {
        return nil
    }
    
    var errors []error
    
    if d.db != nil {
        if err := d.db.Close(); err != nil {
            errors = append(errors, fmt.Errorf("failed to close database: %w", err))
        }
    }
    
    if d.cache != nil {
        if err := d.cache.Close(); err != nil {
            errors = append(errors, fmt.Errorf("failed to close redis: %w", err))
        }
    }
    
    d.closed = true
    
    if len(errors) > 0 {
        return fmt.Errorf("errors closing dependencies: %v", errors)
    }
    
    return nil
}
```

### Performance Monitoring

The base server framework includes comprehensive performance monitoring capabilities:

#### Built-in Performance Metrics

```go
// Get performance metrics from server
server, _ := server.NewBusinessServerCore(config, registrar, deps)
metrics := server.GetPerformanceMetrics()

// Access performance data
snapshot := metrics.GetSnapshot()
fmt.Printf("Startup Time: %v\n", snapshot.StartupTime)
fmt.Printf("Memory Usage: %d bytes\n", snapshot.MemoryUsage)
fmt.Printf("Goroutine Count: %d\n", snapshot.GoroutineCount)
fmt.Printf("Request Count: %d\n", snapshot.RequestCount)
```

#### Custom Performance Hooks

```go
func customPerformanceHook(event string, metrics *server.PerformanceMetrics) {
    snapshot := metrics.GetSnapshot()
    
    // Send metrics to external monitoring system
    prometheus.RecordGauge("server_memory_usage", float64(snapshot.MemoryUsage))
    prometheus.RecordGauge("server_goroutine_count", float64(snapshot.GoroutineCount))
    prometheus.RecordCounter("server_requests_total", float64(snapshot.RequestCount))
    
    // Check for threshold violations
    violations := metrics.CheckThresholds()
    if len(violations) > 0 {
        for _, violation := range violations {
            log.Warn("Performance threshold violation", zap.String("violation", violation))
            alerts.SendAlert("performance_violation", violation)
        }
    }
}

// Add custom hook during server setup
srv, err := server.NewBusinessServerCore(config, registrar, deps)
if err != nil {
    return err
}

monitor := srv.GetPerformanceMonitor()
monitor.AddHook(customPerformanceHook)
```

#### Performance Profiling

```go
func profileServerOperations() {
    profiler := server.NewPerformanceProfiler()
    
    // Profile specific operations
    err := profiler.GetMonitor().ProfileOperation("database_query", func() error {
        return database.Query("SELECT * FROM users")
    })
    
    if err != nil {
        log.Error("Database query failed", zap.Error(err))
    }
    
    // Get current metrics
    metrics := profiler.GetMetrics()
    log.Info("Performance metrics", zap.String("metrics", metrics.String()))
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
    // Create test configuration
    config := server.NewServerConfig()
    config.ServiceName = "test-service"
    config.HTTP.Port = "0" // Use random port for testing
    config.HTTP.TestMode = true
    config.GRPC.Port = "0" // Use random port for testing
    config.GRPC.TestMode = true
    config.Discovery.Enabled = false // Disable for tests
    config.ShutdownTimeout = 5 * time.Second
    
    // Create mock dependencies
    deps := server.NewSimpleBusinessDependencyContainer()
    deps.RegisterInstance("test-db", &MockDatabase{})
    deps.RegisterInstance("test-cache", &MockCache{})
    
    // Create test service
    service := &TestServiceRegistrar{
        deps: deps,
    }
    
    // Create server
    baseServer, err := server.NewBusinessServerCore(config, service, deps)
    require.NoError(t, err)
    
    // Start server
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    err = baseServer.Start(ctx)
    require.NoError(t, err)
    defer func() {
        shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer shutdownCancel()
        baseServer.Stop(shutdownCtx)
    }()
    
    // Verify server is running
    assert.True(t, baseServer.GetHTTPAddress() != "")
    assert.True(t, baseServer.GetGRPCAddress() != "")
    
    // Test HTTP endpoints
    httpAddr := baseServer.GetHTTPAddress()
    resp, err := http.Get(fmt.Sprintf("http://localhost%s/api/v1/test", httpAddr))
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    resp.Body.Close()
    
    // Test gRPC endpoints
    grpcAddr := baseServer.GetGRPCAddress()
    conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
    require.NoError(t, err)
    defer conn.Close()
    
    client := testpb.NewTestServiceClient(conn)
    response, err := client.TestMethod(ctx, &testpb.TestRequest{
        Message: "test",
    })
    require.NoError(t, err)
    assert.Equal(t, "test response", response.Message)
    
    // Test health checks
    healthStatus := baseServer.GetTransportHealth(ctx)
    assert.NotEmpty(t, healthStatus)
    
    // Test performance metrics
    if perfServer, ok := baseServer.(server.BusinessServerWithPerformance); ok {
        metrics := perfServer.GetPerformanceMetrics()
        snapshot := metrics.GetSnapshot()
        assert.Greater(t, snapshot.StartupTime, time.Duration(0))
        assert.Greater(t, snapshot.MemoryUsage, uint64(0))
    }
}

type TestServiceRegistrar struct {
    deps server.BusinessDependencyContainer
}

func (s *TestServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Register HTTP handler
    httpHandler := &TestHTTPHandler{}
    if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
        return err
    }
    
    // Register gRPC service
    grpcService := &TestGRPCService{}
    if err := registry.RegisterBusinessGRPCService(grpcService); err != nil {
        return err
    }
    
    // Register health check
    healthCheck := &TestHealthCheck{}
    if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
        return err
    }
    
    return nil
}

type MockDatabase struct{}
type MockCache struct{}

type TestHTTPHandler struct{}

func (h *TestHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    ginRouter.GET("/api/v1/test", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "test response"})
    })
    return nil
}

func (h *TestHTTPHandler) GetServiceName() string {
    return "test-http"
}

type TestGRPCService struct{}

func (s *TestGRPCService) RegisterGRPC(server interface{}) error {
    grpcServer := server.(*grpc.Server)
    testpb.RegisterTestServiceServer(grpcServer, s)
    return nil
}

func (s *TestGRPCService) GetServiceName() string {
    return "test-grpc"
}

func (s *TestGRPCService) TestMethod(ctx context.Context, req *testpb.TestRequest) (*testpb.TestResponse, error) {
    return &testpb.TestResponse{
        Message: "test response",
    }, nil
}

type TestHealthCheck struct{}

func (h *TestHealthCheck) Check(ctx context.Context) error {
    return nil // Always healthy for tests
}

func (h *TestHealthCheck) GetServiceName() string {
    return "test-health"
}
```

### 4. Monitoring and Observability

- Implement comprehensive health checks for all dependencies
- Use structured logging with proper context
- Leverage built-in performance monitoring and custom hooks
- Set appropriate performance thresholds

```go
type MyService struct {
    logger  *zap.Logger
    monitor *server.PerformanceMonitor
}

func (s *MyService) handleRequest(c *gin.Context) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        
        // Record request metrics
        s.monitor.GetMetrics().IncrementRequestCount()
        
        // Log with structured data
        s.logger.Info("Request processed",
            zap.String("method", c.Request.Method),
            zap.String("path", c.Request.URL.Path),
            zap.Duration("duration", duration),
            zap.Int("status", c.Writer.Status()),
            zap.String("user_agent", c.Request.UserAgent()),
        )
    }()
    
    // Profile the operation
    err := s.monitor.ProfileOperation("handle_request", func() error {
        // Handle request logic
        return s.processRequest(c)
    })
    
    if err != nil {
        c.JSON(500, gin.H{"error": "internal server error"})
        return
    }
}

func (s *MyService) setupPerformanceMonitoring() {
    // Add custom performance hook for external monitoring
    s.monitor.AddHook(func(event string, metrics *server.PerformanceMetrics) {
        snapshot := metrics.GetSnapshot()
        
        // Send to monitoring system
        s.sendMetricsToExternal(snapshot)
        
        // Check custom thresholds
        if snapshot.MemoryUsage > 100*1024*1024 { // 100MB threshold
            s.logger.Warn("High memory usage detected",
                zap.Uint64("memory_bytes", snapshot.MemoryUsage))
        }
    })
}
```

### 5. Performance Optimization

- Use the built-in performance monitoring to identify bottlenecks
- Implement proper resource cleanup in dependency containers
- Configure appropriate timeout values for all operations
- Monitor goroutine counts and memory usage

```go
func optimizeServer(server server.BusinessServerWithPerformance) {
    // Get performance monitor
    monitor := server.GetPerformanceMonitor()
    
    // Add performance threshold monitoring
    monitor.AddHook(server.PerformanceThresholdViolationHook)
    
    // Start periodic metrics collection
    ctx := context.Background()
    monitor.StartPeriodicCollection(ctx, 30*time.Second)
    
    // Profile critical operations
    monitor.ProfileOperation("database_migration", func() error {
        return runDatabaseMigration()
    })
}
```

## New Features and Enhancements

### Performance Monitoring System

The base server framework now includes comprehensive performance monitoring:

- **Built-in Metrics Collection**: Automatic collection of startup time, memory usage, goroutine counts, and request metrics
- **Performance Hooks**: Pluggable hook system for custom monitoring integration
- **Threshold Monitoring**: Configurable thresholds with automatic violation detection
- **Operation Profiling**: Profile individual operations with automatic error counting

### Enhanced Dependency Injection

Improved dependency injection with factory patterns:

- **Singleton and Transient Lifecycle**: Support for both singleton and transient dependency patterns
- **Factory Functions**: Flexible dependency creation with dependency injection support
- **Lifecycle Management**: Automatic initialization and cleanup of dependencies
- **Builder Pattern**: Fluent API for dependency container construction

### Lifecycle Management

Structured server lifecycle with proper error handling:

- **Phased Startup**: Dependencies → Transports → Services → Discovery → Started
- **Phased Shutdown**: Discovery → Transports → Dependencies → Stopped
- **Cleanup on Failure**: Automatic cleanup of partially initialized resources
- **Lifecycle Tracking**: Track current server state and lifecycle phase

### Enhanced Configuration System

Comprehensive configuration with validation and defaults:

- **Extensive Validation**: Validate all configuration options with clear error messages
- **Smart Defaults**: Sensible defaults for production deployment
- **Environment Support**: Easy integration with environment variables
- **Testing Support**: Built-in test mode configurations

### Service Discovery Integration

Enhanced service discovery with better abstraction:

- **ServiceDiscoveryManager Interface**: Abstract service discovery operations
- **Multiple Endpoint Support**: Register multiple endpoints per service
- **Health Check Integration**: Automatic health check registration
- **Graceful Failure Handling**: Continue operation even if discovery fails

### Transport Layer Integration

Seamless integration with the transport layer:

- **Adapter Patterns**: Bridge between server interfaces and transport layer
- **Unified Service Registration**: Single interface for HTTP and gRPC registration
- **Health Check Aggregation**: Aggregate health checks across all transports
- **Performance Integration**: Transport-level performance monitoring

## Migration Guide

### From Previous Version

If you're migrating from a previous version of the base server framework:

1. **Update Interface Names**: 
   - `ServiceRegistrar` → `BusinessServiceRegistrar`
   - `HTTPHandler` → `BusinessHTTPHandler`
   - `GRPCService` → `BusinessGRPCService`
   - `DependencyContainer` → `BusinessDependencyContainer`

2. **Update Method Names**:
   - `RegisterHTTPHandler()` → `RegisterBusinessHTTPHandler()`
   - `RegisterGRPCService()` → `RegisterBusinessGRPCService()`
   - `RegisterHealthCheck()` → `RegisterBusinessHealthCheck()`

3. **Update Server Creation**:
   ```go
   // Old
   server, err := server.NewBaseServer(config, registrar, deps)
   
   // New
   server, err := server.NewBusinessServerCore(config, registrar, deps)
   ```

4. **Add Performance Monitoring** (Optional):
   ```go
   // Access performance monitoring features
   if perfServer, ok := server.(server.BusinessServerWithPerformance); ok {
       monitor := perfServer.GetPerformanceMonitor()
       // Add custom hooks, start periodic collection, etc.
   }
   ```

5. **Update Dependency Injection** (Optional):
   ```go
   // Use new factory-based dependency container
   deps := server.NewBusinessDependencyContainerBuilder().
       AddSingleton("database", dbFactory).
       AddTransient("repository", repositoryFactory).
       Build()
   ```

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
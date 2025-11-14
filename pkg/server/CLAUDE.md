# CLAUDE.md - Base Server Framework

This file provides guidance to Claude Code (claude.ai/code) when working with the base server framework (`pkg/server`) in this repository.

## Package Overview

The base server framework provides a unified, production-ready foundation for microservice development in the swit ecosystem. It implements a comprehensive server architecture that handles transport management, service registration, dependency injection, configuration management, performance monitoring, and lifecycle coordination.

### Key Architectural Principles

1. **Interface-Driven Design** - All components are defined through interfaces for maximum testability and flexibility
2. **Transport Abstraction** - Unified handling of HTTP and gRPC transports through the transport coordinator
3. **Dependency Injection** - Factory-based dependency management with lifecycle support
4. **Configuration Validation** - Comprehensive validation with sensible defaults
5. **Performance Monitoring** - Built-in metrics collection and performance profiling
6. **Lifecycle Management** - Phased startup and shutdown with proper error handling

## Core Architecture Components

### 1. BusinessServerCore Interface

The main server interface providing consistent lifecycle management:

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

**Key Features:**
- Unified server lifecycle management
- Transport status monitoring
- Health check aggregation
- Graceful shutdown with timeout handling

### 2. BusinessServerImpl - Core Implementation

The concrete implementation that orchestrates all server components:

**Architecture:**
- **Transport Management** - Manages HTTP and gRPC transports via `TransportCoordinator`
- **Service Registration** - Coordinates service registration through adapter patterns
- **Dependency Container** - Manages service dependencies and their lifecycles
- **Performance Monitoring** - Built-in metrics collection and profiling
- **Service Discovery** - Consul-based service registration and discovery

**Initialization Flow:**
1. Configuration validation
2. Transport initialization (HTTP/gRPC)
3. Service discovery setup
4. Service registration via adapters
5. Performance monitoring setup

### 3. Service Registration System

#### BusinessServiceRegistrar Interface
Services implement this interface to register with the server:

```go
type BusinessServiceRegistrar interface {
    RegisterServices(registry BusinessServiceRegistry) error
}
```

#### BusinessServiceRegistry Interface
Provides registration methods for different service types:

```go
type BusinessServiceRegistry interface {
    RegisterBusinessHTTPHandler(handler BusinessHTTPHandler) error
    RegisterBusinessGRPCService(service BusinessGRPCService) error
    RegisterBusinessHealthCheck(check BusinessHealthCheck) error
}
```

**Adapter Pattern Implementation:**
- `httpHandlerAdapter` - Bridges `BusinessHTTPHandler` to transport layer
- `grpcServiceAdapter` - Bridges `BusinessGRPCService` to transport layer
- Automatic metadata generation and health check integration

### 4. Dependency Injection System

#### BusinessDependencyContainer Interface
```go
type BusinessDependencyContainer interface {
    Close() error
    GetService(name string) (interface{}, error)
}
```

#### BusinessDependencyRegistry Interface
```go
type BusinessDependencyRegistry interface {
    BusinessDependencyContainer
    Initialize(ctx context.Context) error
    RegisterSingleton(name string, factory DependencyFactory) error
    RegisterTransient(name string, factory DependencyFactory) error
    RegisterInstance(name string, instance interface{}) error
}
```

**Features:**
- **Singleton Pattern** - Dependencies created once and reused
- **Transient Pattern** - Fresh instances for each request
- **Factory Pattern** - Flexible dependency creation
- **Lifecycle Management** - Initialize/shutdown hooks
- **Thread-Safe Operations** - Concurrent access protection

### 5. Configuration Management

#### ServerConfig Structure
Comprehensive configuration with validation and defaults:

```go
type ServerConfig struct {
    ServiceName     string           // Service identification
    HTTP            HTTPConfig       // HTTP transport configuration
    GRPC            GRPCConfig       // gRPC transport configuration
    Discovery       DiscoveryConfig  // Service discovery settings
    Middleware      MiddlewareConfig // Middleware configuration
    ShutdownTimeout time.Duration    // Graceful shutdown timeout
}
```

**Configuration Features:**
- **Validation** - Comprehensive validation with meaningful error messages
- **Defaults** - Sensible defaults for production deployment
- **Flexibility** - Support for both HTTP and gRPC transports
- **Middleware Integration** - Built-in support for common middleware patterns

## Key Features

### Transport Integration

The server framework integrates seamlessly with the transport layer through the `TransportCoordinator`:

**HTTP Transport Features:**
- Gin-based HTTP server
- Configurable middleware (CORS, rate limiting, timeouts)
- Dynamic port allocation for testing
- Health check endpoints

**gRPC Transport Features:**
- Advanced keepalive configuration
- Reflection and health services
- Interceptor chain management
- TLS support
- Message size limits

### Service Discovery Integration

**ServiceDiscoveryManager Interface:**
```go
type ServiceDiscoveryManager interface {
    RegisterService(ctx context.Context, registration *ServiceRegistration) error
    DeregisterService(ctx context.Context, registration *ServiceRegistration) error
    RegisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error
    DeregisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error
    IsHealthy(ctx context.Context) bool
}
```

**Features:**
- Consul-based service registration
- Multiple endpoint support
- Graceful failure handling
- Health check integration

### Performance Monitoring

#### PerformanceMetrics
Comprehensive metrics collection:

```go
type PerformanceMetrics struct {
    StartupTime    time.Duration // Server startup time
    ShutdownTime   time.Duration // Server shutdown time
    Uptime         time.Duration // Current uptime
    MemoryUsage    uint64        // Current memory usage
    GoroutineCount int           // Active goroutines
    ServiceCount   int           // Registered services
    TransportCount int           // Active transports
    StartCount     int64         // Start operations count
    RequestCount   int64         // Request counter
    ErrorCount     int64         // Error counter
}
```

#### PerformanceMonitor
Event-driven performance monitoring:

```go
type PerformanceMonitor struct {
    AddHook(hook PerformanceHook)
    RecordEvent(event string)
    ProfileOperation(name string, operation func() error) error
    StartPeriodicCollection(ctx context.Context, interval time.Duration)
}
```

**Built-in Hooks:**
- `PerformanceLoggingHook` - Logs performance events
- `PerformanceThresholdViolationHook` - Alerts on threshold violations
- `PerformanceMetricsCollectionHook` - Triggers metrics collection

### Lifecycle Management

#### LifecycleManager
Manages phased server startup and shutdown:

**Startup Phases:**
1. `PhaseUninitialized` → `PhaseDependenciesInitialized`
2. `PhaseDependenciesInitialized` → `PhaseTransportsInitialized`
3. `PhaseTransportsInitialized` → `PhaseServicesRegistered`
4. `PhaseServicesRegistered` → `PhaseDiscoveryRegistered`
5. `PhaseDiscoveryRegistered` → `PhaseStarted`

**Shutdown Phases:**
1. `PhaseStarted` → `PhaseStopping`
2. Discovery deregistration
3. Transport shutdown
4. Dependency cleanup
5. `PhaseStopping` → `PhaseStopped`

**Error Handling:**
- Partial initialization cleanup on startup failure
- Graceful degradation on shutdown errors
- LIFO cleanup function execution

## Configuration Guide

### Basic Server Configuration

```yaml
service_name: "my-service"
shutdown_timeout: "30s"

http:
  enabled: true
  port: "8080"
  read_timeout: "30s"
  write_timeout: "30s"
  middleware:
    enable_cors: true
    enable_logging: true
    enable_timeout: true

grpc:
  enabled: true
  port: "9080"
  enable_keepalive: true
  enable_reflection: true
  enable_health_service: true

discovery:
  enabled: true
  address: "127.0.0.1:8500"
  service_name: "my-service"
  tags: ["v1", "production"]
```

### HTTP Transport Configuration

```go
type HTTPConfig struct {
    Port         string            // Listen port
    Address      string            // Listen address
    EnableReady  bool              // Ready channel for testing
    Enabled      bool              // Enable HTTP transport
    TestMode     bool              // Test mode settings
    Middleware   HTTPMiddleware    // Middleware configuration
    ReadTimeout  time.Duration     // Read timeout
    WriteTimeout time.Duration     // Write timeout
    IdleTimeout  time.Duration     // Idle timeout
    Headers      map[string]string // Default headers
}
```

**CORS Configuration:**
```go
type CORSConfig struct {
    AllowOrigins     []string // Allowed origins
    AllowMethods     []string // Allowed HTTP methods
    AllowHeaders     []string // Allowed headers
    ExposeHeaders    []string // Exposed headers
    AllowCredentials bool     // Allow credentials
    MaxAge           int      // Preflight cache duration
}
```

### gRPC Transport Configuration

```go
type GRPCConfig struct {
    Port                string                // Listen port
    Address             string                // Listen address
    EnableKeepalive     bool                  // Enable keepalive
    EnableReflection    bool                  // Enable reflection
    EnableHealthService bool                  // Enable health service
    MaxRecvMsgSize      int                   // Max receive message size
    MaxSendMsgSize      int                   // Max send message size
    KeepaliveParams     GRPCKeepaliveParams   // Keepalive parameters
    KeepalivePolicy     GRPCKeepalivePolicy   // Keepalive policy
    Interceptors        GRPCInterceptorConfig // Interceptor config
    TLS                 *tlsconfig.TLSConfig  // TLS/mTLS configuration
}
```

## Usage Patterns

### Basic Server Setup

```go
// 1. Create configuration
config := server.NewServerConfig()
config.ServiceName = "my-service"
config.HTTP.Port = "8080"
config.GRPC.Port = "9080"

// 2. Create service registrar
registrar := &MyServiceRegistrar{}

// 3. Create dependency container (optional)
deps := server.NewSimpleBusinessDependencyContainer()

// 4. Create server
srv, err := server.NewBusinessServerCore(config, registrar, deps)
if err != nil {
    return fmt.Errorf("failed to create server: %w", err)
}

// 5. Start server
ctx := context.Background()
if err := srv.Start(ctx); err != nil {
    return fmt.Errorf("failed to start server: %w", err)
}

// 6. Graceful shutdown
defer func() {
    if err := srv.Shutdown(); err != nil {
        log.Printf("Shutdown error: %v", err)
    }
}()
```

### Service Registration Implementation

```go
type MyServiceRegistrar struct {
    userService *UserService
    authService *AuthService
}

func (r *MyServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Register HTTP handlers
    if err := registry.RegisterBusinessHTTPHandler(r.userService); err != nil {
        return fmt.Errorf("failed to register user HTTP handler: %w", err)
    }

    // Register gRPC services
    if err := registry.RegisterBusinessGRPCService(r.authService); err != nil {
        return fmt.Errorf("failed to register auth gRPC service: %w", err)
    }

    // Register health checks
    healthCheck := &MyHealthCheck{}
    if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
        return fmt.Errorf("failed to register health check: %w", err)
    }

    return nil
}
```

### HTTP Handler Implementation

```go
type UserService struct{}

func (s *UserService) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    
    v1 := ginRouter.Group("/api/v1")
    {
        v1.GET("/users", s.GetUsers)
        v1.POST("/users", s.CreateUser)
        v1.GET("/users/:id", s.GetUser)
        v1.PUT("/users/:id", s.UpdateUser)
        v1.DELETE("/users/:id", s.DeleteUser)
    }
    
    return nil
}

func (s *UserService) GetServiceName() string {
    return "user-service"
}
```

### gRPC Service Implementation

```go
type AuthService struct{}

func (s *AuthService) RegisterGRPC(server interface{}) error {
    grpcServer := server.(*grpc.Server)
    authpb.RegisterAuthServiceServer(grpcServer, s)
    return nil
}

func (s *AuthService) GetServiceName() string {
    return "auth-service"
}

// Implement authpb.AuthServiceServer methods...
```

### Dependency Injection Usage

```go
// Create dependency container
deps := server.NewBusinessDependencyContainerBuilder().
    AddSingleton("database", func(container server.BusinessDependencyContainer) (interface{}, error) {
        return sql.Open("mysql", "connection-string")
    }).
    AddSingleton("redis", func(container server.BusinessDependencyContainer) (interface{}, error) {
        return redis.NewClient(&redis.Options{
            Addr: "localhost:6379",
        }), nil
    }).
    AddTransient("user-repository", func(container server.BusinessDependencyContainer) (interface{}, error) {
        db, err := container.GetService("database")
        if err != nil {
            return nil, err
        }
        return &UserRepository{DB: db.(*sql.DB)}, nil
    }).
    Build()

// Use in service
func (s *UserService) GetUsers(c *gin.Context) {
    userRepo, err := s.deps.GetService("user-repository")
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    users, err := userRepo.(*UserRepository).GetAll()
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, users)
}
```

### Performance Monitoring Integration

```go
// Custom performance hook
func CustomPerformanceHook(event string, metrics *server.PerformanceMetrics) {
    snapshot := metrics.GetSnapshot()
    
    // Log to external monitoring system
    monitoring.RecordMetric("server.startup_time", snapshot.StartupTime)
    monitoring.RecordMetric("server.memory_usage", snapshot.MemoryUsage)
    monitoring.RecordMetric("server.goroutine_count", snapshot.GoroutineCount)
}

// Server setup with custom monitoring
srv, err := server.NewBusinessServerCore(config, registrar, deps)
if err != nil {
    return err
}

// Add custom performance hook
monitor := srv.GetPerformanceMonitor()
monitor.AddHook(CustomPerformanceHook)

// Profile operations
err = monitor.ProfileOperation("user_creation", func() error {
    return userService.CreateUser(userData)
})
```

## Integration with Transport Layer

The base server framework seamlessly integrates with the transport layer through adapter patterns:

### Service Registry Adapter

```go
type serviceRegistryAdapter struct {
    transportManager *transport.TransportCoordinator
    httpTransport    *transport.HTTPNetworkService
    grpcTransport    *transport.GRPCNetworkService
}

func (a *serviceRegistryAdapter) RegisterBusinessHTTPHandler(handler BusinessHTTPHandler) error {
    adapter := &httpHandlerAdapter{handler: handler}
    return a.transportManager.RegisterHTTPService(adapter)
}
```

### Handler Adapters

```go
type httpHandlerAdapter struct {
    handler BusinessHTTPHandler
}

func (a *httpHandlerAdapter) RegisterHTTP(router *gin.Engine) error {
    return a.handler.RegisterRoutes(router)
}

func (a *httpHandlerAdapter) GetMetadata() *transport.HandlerMetadata {
    return &transport.HandlerMetadata{
        Name:        a.handler.GetServiceName(),
        Version:     "v1",
        Description: fmt.Sprintf("HTTP service: %s", a.handler.GetServiceName()),
    }
}
```

## Best Practices

### Server Implementation

1. **Configuration Validation** - Always validate configuration before server creation
2. **Error Handling** - Implement comprehensive error handling with context
3. **Graceful Shutdown** - Use proper shutdown mechanisms with timeouts
4. **Health Checks** - Implement meaningful health checks for all services
5. **Performance Monitoring** - Leverage built-in performance monitoring

### Dependency Management

1. **Interface Design** - Define clear interfaces for all dependencies
2. **Factory Pattern** - Use factories for complex dependency creation
3. **Lifecycle Management** - Implement proper initialization and cleanup
4. **Error Propagation** - Handle dependency errors gracefully
5. **Testing** - Use mock implementations for unit testing

### Configuration Management

1. **Environment-Specific Configs** - Use different configurations for different environments
2. **Validation** - Validate all configuration values at startup
3. **Defaults** - Provide sensible defaults for optional settings
4. **Documentation** - Document all configuration options

### Performance Optimization

1. **Metrics Collection** - Monitor key performance indicators
2. **Resource Management** - Monitor memory usage and goroutine counts
3. **Profiling** - Use built-in profiling for performance bottlenecks
4. **Threshold Monitoring** - Set and monitor performance thresholds

## Testing Strategies

### Unit Testing

```go
func TestServerStartup(t *testing.T) {
    // Create test configuration
    config := server.NewServerConfig()
    config.HTTP.TestMode = true
    config.HTTP.Port = "0" // Dynamic port allocation
    config.GRPC.Port = "0"
    
    // Create mock registrar
    registrar := &MockServiceRegistrar{}
    
    // Create server
    srv, err := server.NewBusinessServerCore(config, registrar, nil)
    require.NoError(t, err)
    
    // Test startup
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    err = srv.Start(ctx)
    require.NoError(t, err)
    
    // Verify server is running
    assert.True(t, srv.GetHTTPAddress() != "")
    assert.True(t, srv.GetGRPCAddress() != "")
    
    // Test shutdown
    err = srv.Shutdown()
    require.NoError(t, err)
}
```

### Integration Testing

```go
func TestEndToEndIntegration(t *testing.T) {
    // Setup test server with real services
    config := server.NewServerConfig()
    config.HTTP.TestMode = true
    config.Discovery.Enabled = false // Disable for testing
    
    registrar := &TestServiceRegistrar{
        userService: NewUserService(testDB),
        authService: NewAuthService(testRedis),
    }
    
    srv, err := server.NewBusinessServerCore(config, registrar, nil)
    require.NoError(t, err)
    
    // Start server
    ctx := context.Background()
    err = srv.Start(ctx)
    require.NoError(t, err)
    defer srv.Shutdown()
    
    // Test HTTP endpoints
    httpAddr := srv.GetHTTPAddress()
    resp, err := http.Get(fmt.Sprintf("http://localhost%s/api/v1/users", httpAddr))
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    // Test gRPC services
    grpcAddr := srv.GetGRPCAddress()
    conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
    require.NoError(t, err)
    defer conn.Close()
    
    client := authpb.NewAuthServiceClient(conn)
    resp, err := client.Login(ctx, &authpb.LoginRequest{
        Username: "test",
        Password: "test",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, resp.Token)
}
```

## Common Issues and Solutions

### Port Conflicts
- **Issue**: Server fails to start due to port conflicts
- **Solution**: Use dynamic port allocation (port "0") for testing, configure different ports for production

### Dependency Initialization Failures
- **Issue**: Dependencies fail to initialize during server startup
- **Solution**: Implement proper error handling in factory functions, use health checks to verify dependencies

### Graceful Shutdown Issues
- **Issue**: Server doesn't shutdown gracefully within timeout
- **Solution**: Implement proper context handling in services, configure appropriate shutdown timeouts

### Performance Degradation
- **Issue**: Server performance degrades over time
- **Solution**: Monitor memory usage and goroutine counts, implement proper resource cleanup

### Configuration Validation Errors
- **Issue**: Server fails to start due to invalid configuration
- **Solution**: Use comprehensive validation, provide meaningful error messages, set sensible defaults

## Advanced Patterns

### Custom Middleware Integration

```go
type CustomMiddlewareManager struct {
    *server.MiddlewareManager
}

func (cm *CustomMiddlewareManager) AddCustomMiddleware(router *gin.Engine) error {
    // Add request ID middleware
    router.Use(func(c *gin.Context) {
        requestID := uuid.New().String()
        c.Header("X-Request-ID", requestID)
        c.Set("request_id", requestID)
        c.Next()
    })
    
    // Add custom authentication
    router.Use(cm.authMiddleware())
    
    return nil
}
```

### Performance Profiling Integration

```go
func SetupPerformanceProfiling(server *server.BusinessServerImpl) {
    profiler := server.NewPerformanceProfiler()
    
    // Profile startup
    startupTime, err := profiler.ProfileServerStartup(server)
    if err != nil {
        log.Printf("Startup profiling failed: %v", err)
    } else {
        log.Printf("Server startup took: %v", startupTime)
    }
    
    // Add custom performance hooks
    monitor := profiler.GetMonitor()
    monitor.AddHook(func(event string, metrics *server.PerformanceMetrics) {
        // Send metrics to external monitoring system
        sendToPrometheus(event, metrics)
    })
}
```

This documentation provides comprehensive guidance for working with the base server framework, covering all major components, usage patterns, and best practices for building robust microservices in the swit ecosystem.
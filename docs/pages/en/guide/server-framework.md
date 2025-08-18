# Server Framework

This guide covers the core server framework in Swit, which provides a comprehensive, production-ready foundation for microservice development with unified lifecycle management, transport coordination, and service registration.

## Overview

The Swit server framework (`pkg/server`) implements a complete server architecture that handles:

- **Transport Management** - Unified HTTP and gRPC transport coordination
- **Service Registration** - Interface-driven service registration patterns
- **Dependency Injection** - Factory-based dependency management
- **Performance Monitoring** - Built-in metrics and profiling
- **Lifecycle Management** - Phased startup and shutdown coordination

## Core Architecture

### BusinessServerCore Interface

The main server interface provides consistent lifecycle management:

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

### BusinessServerImpl - The Core Implementation

The main server implementation orchestrates all framework components:

```go
// Create a new server instance
config := &server.ServerConfig{
    ServiceName: "my-service",
    HTTP: server.HTTPConfig{
        Port:    "8080",
        Enabled: true,
    },
    GRPC: server.GRPCConfig{
        Port:    "9080",
        Enabled: true,
    },
}

srv, err := server.NewBusinessServerCore(config, serviceRegistrar, dependencies)
if err != nil {
    return fmt.Errorf("failed to create server: %w", err)
}
```

## Service Registration System

### BusinessServiceRegistrar Interface

Services implement this interface to register with the server:

```go
type BusinessServiceRegistrar interface {
    RegisterServices(registry BusinessServiceRegistry) error
}
```

### Implementation Example

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
    healthCheck := &ServiceHealthCheck{
        userService: r.userService,
        authService: r.authService,
    }
    if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
        return fmt.Errorf("failed to register health check: %w", err)
    }

    return nil
}
```

### BusinessServiceRegistry Interface

Provides registration methods for different service types:

```go
type BusinessServiceRegistry interface {
    RegisterBusinessHTTPHandler(handler BusinessHTTPHandler) error
    RegisterBusinessGRPCService(service BusinessGRPCService) error
    RegisterBusinessHealthCheck(check BusinessHealthCheck) error
}
```

## HTTP Service Implementation

### BusinessHTTPHandler Interface

```go
type BusinessHTTPHandler interface {
    RegisterRoutes(router interface{}) error
    GetServiceName() string
}
```

### Implementation Example

```go
type UserService struct {
    repository *UserRepository
    logger     *zap.Logger
}

func (s *UserService) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    
    v1 := ginRouter.Group("/api/v1")
    {
        users := v1.Group("/users")
        {
            users.GET("", s.GetUsers)
            users.POST("", s.CreateUser)
            users.GET("/:id", s.GetUser)
            users.PUT("/:id", s.UpdateUser)
            users.DELETE("/:id", s.DeleteUser)
        }
    }
    
    return nil
}

func (s *UserService) GetServiceName() string {
    return "user-service"
}

func (s *UserService) GetUsers(c *gin.Context) {
    users, err := s.repository.GetAll()
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"users": users})
}
```

## gRPC Service Implementation

### BusinessGRPCService Interface

```go
type BusinessGRPCService interface {
    RegisterGRPC(server interface{}) error
    GetServiceName() string
}
```

### Implementation Example

```go
type AuthService struct {
    authpb.UnimplementedAuthServiceServer
    tokenManager *TokenManager
    userRepo     *UserRepository
}

func (s *AuthService) RegisterGRPC(server interface{}) error {
    grpcServer := server.(*grpc.Server)
    authpb.RegisterAuthServiceServer(grpcServer, s)
    return nil
}

func (s *AuthService) GetServiceName() string {
    return "auth-service"
}

func (s *AuthService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
    // Validate user credentials
    user, err := s.userRepo.ValidateCredentials(req.Username, req.Password)
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
    }
    
    // Generate token
    token, err := s.tokenManager.GenerateToken(user.ID)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to generate token")
    }
    
    return &authpb.LoginResponse{
        Token:     token,
        ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
        User: &authpb.User{
            Id:       user.ID,
            Username: user.Username,
            Email:    user.Email,
        },
    }, nil
}
```

## Health Check Implementation

### BusinessHealthCheck Interface

```go
type BusinessHealthCheck interface {
    Check(ctx context.Context) error
    GetServiceName() string
}
```

### Implementation Example

```go
type ServiceHealthCheck struct {
    database *sql.DB
    redis    *redis.Client
    services []HealthChecker
}

func (h *ServiceHealthCheck) Check(ctx context.Context) error {
    // Check database connection
    if err := h.database.PingContext(ctx); err != nil {
        return fmt.Errorf("database health check failed: %w", err)
    }
    
    // Check Redis connection
    if err := h.redis.Ping(ctx).Err(); err != nil {
        return fmt.Errorf("redis health check failed: %w", err)
    }
    
    // Check dependent services
    for _, service := range h.services {
        if err := service.HealthCheck(ctx); err != nil {
            return fmt.Errorf("service %s health check failed: %w", service.GetName(), err)
        }
    }
    
    return nil
}

func (h *ServiceHealthCheck) GetServiceName() string {
    return "service-health-check"
}
```

## Lifecycle Management

### Server Startup Process

The server follows a phased startup process:

1. **Configuration Validation** - Validate all configuration parameters
2. **Dependency Initialization** - Initialize dependency container
3. **Transport Initialization** - Start HTTP and gRPC transports
4. **Service Registration** - Register all services with transports
5. **Discovery Registration** - Register with service discovery
6. **Performance Monitoring** - Start performance monitoring

```go
func (s *BusinessServerImpl) Start(ctx context.Context) error {
    // Phase 1: Initialize dependencies
    if err := s.initializeDependencies(ctx); err != nil {
        return fmt.Errorf("failed to initialize dependencies: %w", err)
    }
    
    // Phase 2: Initialize transports
    if err := s.initializeTransports(ctx); err != nil {
        return fmt.Errorf("failed to initialize transports: %w", err)
    }
    
    // Phase 3: Register services
    if err := s.registerServices(); err != nil {
        return fmt.Errorf("failed to register services: %w", err)
    }
    
    // Phase 4: Start discovery registration
    if err := s.registerWithDiscovery(ctx); err != nil {
        return fmt.Errorf("failed to register with discovery: %w", err)
    }
    
    // Phase 5: Start performance monitoring
    s.startPerformanceMonitoring(ctx)
    
    return nil
}
```

### Graceful Shutdown

The server supports graceful shutdown with proper resource cleanup:

```go
func (s *BusinessServerImpl) Shutdown() error {
    ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
    defer cancel()
    
    // Deregister from service discovery
    if err := s.deregisterFromDiscovery(ctx); err != nil {
        s.logger.Error("Failed to deregister from discovery", zap.Error(err))
    }
    
    // Stop transports gracefully
    if err := s.transportCoordinator.Stop(s.config.ShutdownTimeout); err != nil {
        s.logger.Error("Transport shutdown errors", zap.Error(err))
    }
    
    // Close dependencies
    if s.dependencies != nil {
        if err := s.dependencies.Close(); err != nil {
            s.logger.Error("Failed to close dependencies", zap.Error(err))
        }
    }
    
    return nil
}
```

## Performance Monitoring

### Built-in Performance Metrics

The server framework includes comprehensive performance monitoring:

```go
type PerformanceMetrics struct {
    StartupTime    time.Duration // Server startup time
    ShutdownTime   time.Duration // Server shutdown time
    Uptime         time.Duration // Current uptime
    MemoryUsage    uint64        // Current memory usage
    GoroutineCount int           // Active goroutines
    ServiceCount   int           // Registered services
    TransportCount int           // Active transports
    RequestCount   int64         // Request counter
    ErrorCount     int64         // Error counter
}
```

### Performance Hooks

Add custom performance monitoring:

```go
func CustomPerformanceHook(event string, metrics *server.PerformanceMetrics) {
    // Send metrics to external monitoring system
    switch event {
    case "startup_complete":
        prometheus.SetGauge("server_startup_time_seconds", metrics.StartupTime.Seconds())
    case "request_processed":
        prometheus.IncrementCounter("server_requests_total")
    case "memory_threshold_exceeded":
        alert.Send("High memory usage detected", metrics.MemoryUsage)
    }
}

// Register the hook
srv := server.NewBusinessServerCore(config, registrar, deps)
monitor := srv.GetPerformanceMonitor()
monitor.AddHook(CustomPerformanceHook)
```

## Advanced Usage Patterns

### Multi-Service Server

```go
type MultiServiceRegistrar struct {
    userService    *UserService
    authService    *AuthService
    paymentService *PaymentService
    orderService   *OrderService
}

func (r *MultiServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
    // Register all HTTP handlers
    httpHandlers := []server.BusinessHTTPHandler{
        r.userService,
        r.authService,
        r.paymentService,
        r.orderService,
    }
    
    for _, handler := range httpHandlers {
        if err := registry.RegisterBusinessHTTPHandler(handler); err != nil {
            return fmt.Errorf("failed to register HTTP handler %s: %w", 
                handler.GetServiceName(), err)
        }
    }
    
    // Register gRPC services
    grpcServices := []server.BusinessGRPCService{
        r.authService,
        r.paymentService,
    }
    
    for _, service := range grpcServices {
        if err := registry.RegisterBusinessGRPCService(service); err != nil {
            return fmt.Errorf("failed to register gRPC service %s: %w", 
                service.GetServiceName(), err)
        }
    }
    
    // Register composite health check
    healthCheck := &CompositeHealthCheck{
        checks: []server.BusinessHealthCheck{
            &DatabaseHealthCheck{},
            &RedisHealthCheck{},
            &ExternalServiceHealthCheck{},
        },
    }
    
    return registry.RegisterBusinessHealthCheck(healthCheck)
}
```

### Custom Server Extensions

```go
type EnhancedBusinessServer struct {
    *server.BusinessServerImpl
    metricsCollector *MetricsCollector
    auditLogger      *AuditLogger
}

func NewEnhancedBusinessServer(config *server.ServerConfig, registrar server.BusinessServiceRegistrar, deps server.BusinessDependencyContainer) (*EnhancedBusinessServer, error) {
    baseServer, err := server.NewBusinessServerCore(config, registrar, deps)
    if err != nil {
        return nil, err
    }
    
    enhanced := &EnhancedBusinessServer{
        BusinessServerImpl: baseServer.(*server.BusinessServerImpl),
        metricsCollector:   NewMetricsCollector(),
        auditLogger:       NewAuditLogger(),
    }
    
    // Add custom monitoring
    monitor := enhanced.GetPerformanceMonitor()
    monitor.AddHook(enhanced.metricsCollector.CollectMetrics)
    monitor.AddHook(enhanced.auditLogger.LogAuditEvent)
    
    return enhanced, nil
}

func (s *EnhancedBusinessServer) Start(ctx context.Context) error {
    // Start metrics collection
    s.metricsCollector.Start(ctx)
    
    // Start audit logging
    s.auditLogger.Start(ctx)
    
    // Start base server
    return s.BusinessServerImpl.Start(ctx)
}
```

## Testing the Server Framework

### Unit Testing Services

```go
func TestUserServiceRegistration(t *testing.T) {
    // Create mock registry
    registry := &MockBusinessServiceRegistry{}
    
    // Create service
    userService := &UserService{
        repository: &MockUserRepository{},
        logger:     zap.NewNop(),
    }
    
    registrar := &MyServiceRegistrar{
        userService: userService,
    }
    
    // Test registration
    err := registrar.RegisterServices(registry)
    assert.NoError(t, err)
    
    // Verify registration calls
    assert.True(t, registry.HTTPHandlerRegistered)
    assert.Equal(t, "user-service", registry.RegisteredServiceName)
}
```

### Integration Testing

```go
func TestServerIntegration(t *testing.T) {
    // Create test configuration
    config := server.NewServerConfig()
    config.HTTP.Port = "0" // Dynamic port
    config.GRPC.Port = "0" // Dynamic port
    config.Discovery.Enabled = false
    config.HTTP.TestMode = true
    config.GRPC.TestMode = true
    
    // Create test services
    userService := NewTestUserService()
    authService := NewTestAuthService()
    
    registrar := &TestServiceRegistrar{
        userService: userService,
        authService: authService,
    }
    
    // Create and start server
    srv, err := server.NewBusinessServerCore(config, registrar, nil)
    require.NoError(t, err)
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
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
    loginResp, err := client.Login(ctx, &authpb.LoginRequest{
        Username: "testuser",
        Password: "testpass",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, loginResp.Token)
}
```

## Best Practices

### Service Design

1. **Single Responsibility** - Each service should have a clear, single responsibility
2. **Interface Segregation** - Use specific interfaces rather than large, monolithic ones
3. **Dependency Injection** - Use dependency injection for testability
4. **Error Handling** - Implement comprehensive error handling with context
5. **Logging** - Use structured logging throughout services

### Performance Optimization

1. **Resource Management** - Properly manage database connections, file handles, etc.
2. **Context Propagation** - Always propagate context for cancellation and timeouts
3. **Metrics Collection** - Monitor key performance indicators
4. **Memory Management** - Avoid memory leaks with proper resource cleanup
5. **Concurrency** - Use goroutines and channels appropriately

### Production Readiness

1. **Health Checks** - Implement meaningful health checks for all dependencies
2. **Graceful Shutdown** - Handle shutdown signals properly
3. **Configuration Validation** - Validate configuration at startup
4. **Error Recovery** - Implement proper error recovery mechanisms
5. **Monitoring Integration** - Integrate with monitoring and alerting systems

## Common Patterns

### Service Factory Pattern

```go
type ServiceFactory struct {
    config *Config
    db     *sql.DB
    redis  *redis.Client
}

func (f *ServiceFactory) CreateUserService() *UserService {
    return &UserService{
        repository: &UserRepository{DB: f.db},
        cache:     &UserCache{Redis: f.redis},
        logger:    logger.Named("user-service"),
    }
}

func (f *ServiceFactory) CreateAuthService() *AuthService {
    return &AuthService{
        userRepo:     &UserRepository{DB: f.db},
        tokenManager: &JWTTokenManager{Secret: f.config.JWTSecret},
        logger:      logger.Named("auth-service"),
    }
}
```

### Middleware Integration

```go
type AuthenticatedHTTPHandler struct {
    handler     server.BusinessHTTPHandler
    authService *AuthService
}

func (h *AuthenticatedHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    
    // Add authentication middleware
    ginRouter.Use(h.authMiddleware())
    
    // Register actual handler routes
    return h.handler.RegisterRoutes(router)
}

func (h *AuthenticatedHTTPHandler) authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "missing authorization"})
            return
        }
        
        userID, err := h.authService.ValidateToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }
        
        c.Set("userID", userID)
        c.Next()
    }
}
```

This server framework guide provides comprehensive coverage of the core server functionality, service registration patterns, lifecycle management, and best practices for building robust microservices with the Swit framework.
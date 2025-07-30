# Migration Guide: Base Server Framework

This guide provides step-by-step instructions for migrating existing services to use the base server framework.

## Overview

The base server framework consolidates common server patterns and eliminates code duplication across services. This migration guide covers:

- Pre-migration assessment
- Step-by-step migration process
- Breaking changes and compatibility considerations
- Troubleshooting common issues
- Post-migration validation

## Pre-Migration Assessment

Before starting the migration, assess your current service:

### 1. Identify Current Patterns

Review your existing service for these patterns:

- **Server Initialization**: How is the server created and configured?
- **Transport Management**: HTTP and/or gRPC setup
- **Service Discovery**: Consul registration/deregistration
- **Dependency Injection**: How dependencies are managed
- **Middleware**: CORS, authentication, logging, etc.
- **Health Checks**: Current health check implementation
- **Graceful Shutdown**: How the service handles shutdown

### 2. Document Current Configuration

Create a mapping of your current configuration to the new format:

```go
// Current (example from switauth)
type Config struct {
    Server struct {
        HTTP struct {
            Port string `yaml:"port"`
        } `yaml:"http"`
        GRPC struct {
            Port string `yaml:"port"`
        } `yaml:"grpc"`
    } `yaml:"server"`
    Discovery struct {
        Address string `yaml:"address"`
    } `yaml:"discovery"`
}

// New base server format
type ServerConfig struct {
    ServiceName string      `yaml:"service_name"`
    HTTP        HTTPConfig  `yaml:"http"`
    GRPC        GRPCConfig  `yaml:"grpc"`
    Discovery   DiscoveryConfig `yaml:"discovery"`
    // ... other fields
}
```

### 3. Identify Custom Logic

Document any custom logic that needs to be preserved:

- Custom middleware implementations
- Specific error handling patterns
- Custom health check logic
- Service-specific initialization code

## Migration Process

### Step 1: Create Service Registrar

Create a struct that implements the `ServiceRegistrar` interface:

```go
// Before: Direct server setup
func setupServer() {
    router := gin.New()
    router.GET("/health", healthHandler)
    router.POST("/api/v1/login", loginHandler)
    // ... more routes
}

// After: Service registrar
type MyService struct {
    // Your service dependencies
}

func (s *MyService) RegisterServices(registry server.ServiceRegistry) error {
    // Register HTTP handlers
    httpHandler := &MyHTTPHandler{}
    if err := registry.RegisterHTTPHandler(httpHandler); err != nil {
        return err
    }
    
    // Register health checks
    healthCheck := &MyHealthCheck{}
    if err := registry.RegisterHealthCheck(healthCheck); err != nil {
        return err
    }
    
    return nil
}
```

### Step 2: Implement HTTP Handler

Convert your HTTP routes to implement the `HTTPHandler` interface:

```go
// Before: Direct Gin setup
func setupRoutes(router *gin.Engine) {
    api := router.Group("/api/v1")
    api.POST("/login", loginHandler)
    api.GET("/users", getUsersHandler)
}

// After: HTTPHandler implementation
type MyHTTPHandler struct {
    serviceName string
}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    
    api := ginRouter.Group("/api/v1")
    api.POST("/login", h.loginHandler)
    api.GET("/users", h.getUsersHandler)
    
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return h.serviceName
}

// Convert your existing handlers
func (h *MyHTTPHandler) loginHandler(c *gin.Context) {
    // Your existing login logic
}
```

### Step 3: Implement gRPC Service (if applicable)

Convert your gRPC service registration:

```go
// Before: Direct gRPC setup
func setupGRPC() *grpc.Server {
    server := grpc.NewServer()
    pb.RegisterAuthServiceServer(server, &authService{})
    return server
}

// After: GRPCService implementation
type MyGRPCService struct {
    serviceName string
    pb.UnimplementedAuthServiceServer
}

func (s *MyGRPCService) RegisterGRPC(server interface{}) error {
    grpcServer := server.(*grpc.Server)
    pb.RegisterAuthServiceServer(grpcServer, s)
    return nil
}

func (s *MyGRPCService) GetServiceName() string {
    return s.serviceName
}

// Your existing gRPC method implementations remain the same
func (s *MyGRPCService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
    // Your existing login logic
}
```

### Step 4: Implement Health Checks

Convert your health check logic:

```go
// Before: Direct health endpoint
func healthHandler(c *gin.Context) {
    // Check database, external services, etc.
    c.JSON(200, gin.H{"status": "healthy"})
}

// After: HealthCheck implementation
type MyHealthCheck struct {
    serviceName string
    db          *sql.DB // Your dependencies
}

func (h *MyHealthCheck) Check(ctx context.Context) error {
    // Check database connection
    if err := h.db.PingContext(ctx); err != nil {
        return fmt.Errorf("database unhealthy: %w", err)
    }
    
    // Check other dependencies
    return nil
}

func (h *MyHealthCheck) GetServiceName() string {
    return h.serviceName
}
```

### Step 5: Implement Dependency Container

Convert your dependency management:

```go
// Before: Global variables or manual DI
var (
    db     *sql.DB
    cache  *redis.Client
    config *Config
)

// After: DependencyContainer implementation
type MyDependencyContainer struct {
    db     *sql.DB
    cache  *redis.Client
    config *Config
    closed bool
}

func NewMyDependencyContainer(config *Config) (*MyDependencyContainer, error) {
    // Initialize dependencies
    db, err := sql.Open("mysql", config.Database.DSN)
    if err != nil {
        return nil, err
    }
    
    cache := redis.NewClient(&redis.Options{
        Addr: config.Redis.Address,
    })
    
    return &MyDependencyContainer{
        db:     db,
        cache:  cache,
        config: config,
    }, nil
}

func (d *MyDependencyContainer) GetService(name string) (interface{}, error) {
    switch name {
    case "db":
        return d.db, nil
    case "cache":
        return d.cache, nil
    case "config":
        return d.config, nil
    default:
        return nil, fmt.Errorf("service %s not found", name)
    }
}

func (d *MyDependencyContainer) Close() error {
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

### Step 6: Update Configuration

Convert your configuration to the new format:

```go
// Before: Service-specific config
type Config struct {
    Server struct {
        HTTP struct {
            Port string `yaml:"port"`
        } `yaml:"http"`
        GRPC struct {
            Port string `yaml:"port"`
        } `yaml:"grpc"`
    } `yaml:"server"`
    Database struct {
        DSN string `yaml:"dsn"`
    } `yaml:"database"`
}

// After: Base server config + service-specific config
type Config struct {
    Server   *server.ServerConfig `yaml:"server"`
    Database struct {
        DSN string `yaml:"dsn"`
    } `yaml:"database"`
}

func loadConfig() (*Config, error) {
    config := &Config{
        Server: &server.ServerConfig{
            ServiceName: "my-service",
            HTTP: server.HTTPConfig{
                Port:         getEnv("HTTP_PORT", "8080"),
                Enabled:      true,
                EnableReady:  true,
                ReadTimeout:  30 * time.Second,
                WriteTimeout: 30 * time.Second,
                IdleTimeout:  60 * time.Second,
            },
            GRPC: server.GRPCConfig{
                Port:                getEnv("GRPC_PORT", "9090"),
                Enabled:             true,
                EnableReflection:    true,
                EnableHealthService: true,
                MaxRecvMsgSize:      4 * 1024 * 1024,
                MaxSendMsgSize:      4 * 1024 * 1024,
                // ... keepalive settings
            },
            ShutdownTimeout: 30 * time.Second,
            Discovery: server.DiscoveryConfig{
                Enabled:     getBoolEnv("DISCOVERY_ENABLED", true),
                Address:     getEnv("CONSUL_ADDRESS", "localhost:8500"),
                ServiceName: "my-service",
                Tags:        []string{"api", "v1"},
            },
            Middleware: server.MiddlewareConfig{
                EnableCORS:    true,
                EnableLogging: true,
            },
        },
        Database: struct {
            DSN string `yaml:"dsn"`
        }{
            DSN: getEnv("DATABASE_DSN", "user:pass@tcp(localhost:3306)/db"),
        },
    }
    
    return config, nil
}
```

### Step 7: Update Main Function

Replace your main server setup:

```go
// Before: Manual server setup
func main() {
    config := loadConfig()
    
    // Setup database
    db, err := sql.Open("mysql", config.Database.DSN)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Setup HTTP server
    router := gin.New()
    setupRoutes(router)
    httpServer := &http.Server{
        Addr:    ":" + config.Server.HTTP.Port,
        Handler: router,
    }
    
    // Setup gRPC server
    grpcServer := setupGRPC()
    
    // Start servers...
    // Complex startup and shutdown logic
}

// After: Base server framework
func main() {
    logger.InitLogger()
    
    config, err := loadConfig()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    if err := config.Server.Validate(); err != nil {
        log.Fatal("Invalid configuration:", err)
    }
    
    // Create dependencies
    deps, err := NewMyDependencyContainer(config)
    if err != nil {
        log.Fatal("Failed to create dependencies:", err)
    }
    
    // Create service
    service := NewMyService(deps)
    
    // Create base server
    baseServer, err := server.NewBaseServer(config.Server, service, deps)
    if err != nil {
        log.Fatal("Failed to create server:", err)
    }
    
    // Start server
    ctx := context.Background()
    if err := baseServer.Start(ctx); err != nil {
        log.Fatal("Failed to start server:", err)
    }
    
    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    
    // Graceful shutdown
    if err := baseServer.Shutdown(); err != nil {
        log.Printf("Error during shutdown: %v", err)
    }
}
```

### Step 8: Update Tests

Migrate your tests to use the new patterns:

```go
// Before: Manual test setup
func TestMyService(t *testing.T) {
    router := gin.New()
    setupRoutes(router)
    
    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    assert.Equal(t, 200, w.Code)
}

// After: Base server test patterns
func TestMyService(t *testing.T) {
    config := &server.ServerConfig{
        ServiceName: "test-service",
        HTTP: server.HTTPConfig{
            Port:    "0", // Random port
            Enabled: true,
            // ... other required fields
        },
        Discovery: server.DiscoveryConfig{
            Enabled: false, // Disable for tests
        },
    }
    
    deps := NewTestDependencyContainer()
    service := NewMyService(deps)
    
    baseServer, err := server.NewBaseServer(config, service, deps)
    require.NoError(t, err)
    
    ctx := context.Background()
    err = baseServer.Start(ctx)
    require.NoError(t, err)
    defer baseServer.Shutdown()
    
    // Test your endpoints
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get(fmt.Sprintf("http://%s/health", baseServer.GetHTTPAddress()))
    require.NoError(t, err)
    defer resp.Body.Close()
    
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

## Breaking Changes and Compatibility

### Breaking Changes

1. **Server Initialization**: Manual server setup is replaced with base server framework
2. **Configuration Structure**: New configuration format required
3. **Dependency Management**: Must implement `DependencyContainer` interface
4. **Service Registration**: Must implement `ServiceRegistrar` interface
5. **Health Checks**: Must implement `HealthCheck` interface

### Backward Compatibility

The framework maintains backward compatibility for:

- **API Endpoints**: All existing endpoints continue to work
- **gRPC Methods**: All existing gRPC methods continue to work
- **External Behavior**: Service behavior remains identical from client perspective

### Configuration Migration

| Old Pattern | New Pattern | Notes |
|-------------|-------------|-------|
| `server.http.port` | `http.port` | Direct mapping |
| `server.grpc.port` | `grpc.port` | Direct mapping |
| `discovery.address` | `discovery.address` | Direct mapping |
| Custom middleware setup | `middleware.enable_*` | Standardized middleware |
| Manual health checks | `HealthCheck` interface | Structured health checks |

## Troubleshooting

### Common Issues

#### 1. Port Already in Use

**Error**: `bind: address already in use`

**Solution**: 
- Check if old service is still running
- Use different ports for testing
- Use port `"0"` for automatic port assignment in tests

#### 2. Configuration Validation Errors

**Error**: `invalid server configuration: field X is required`

**Solution**:
- Ensure all required fields are set
- Check timeout values are positive
- Verify port numbers are valid

#### 3. Service Registration Failed

**Error**: `failed to register HTTP handler: ...`

**Solution**:
- Verify `HTTPHandler` interface is correctly implemented
- Check `RegisterRoutes` method doesn't return errors
- Ensure `GetServiceName()` returns non-empty string

#### 4. gRPC Service Registration Failed

**Error**: `failed to register gRPC service: ...`

**Solution**:
- Verify `GRPCService` interface is correctly implemented
- Check protobuf generated code is up to date
- Ensure gRPC server is properly typed in `RegisterGRPC`

#### 5. Health Check Failures

**Error**: Health checks always failing

**Solution**:
- Verify `HealthCheck.Check()` method logic
- Check dependencies are properly initialized
- Ensure context is not cancelled prematurely

#### 6. Discovery Registration Issues

**Error**: Service not appearing in Consul

**Solution**:
- Verify Consul is running and accessible
- Check discovery configuration is correct
- Ensure service name and tags are valid
- Check network connectivity to Consul

### Debugging Tips

1. **Enable Debug Logging**: Set log level to debug for detailed information
2. **Check Server Status**: Use `GetTransportStatus()` to verify transport states
3. **Validate Configuration**: Always call `config.Validate()` early
4. **Test Incrementally**: Migrate one component at a time
5. **Use Integration Tests**: Verify end-to-end functionality

### Migration Checklist

- [ ] Service implements `ServiceRegistrar` interface
- [ ] HTTP handlers implement `HTTPHandler` interface
- [ ] gRPC services implement `GRPCService` interface
- [ ] Health checks implement `HealthCheck` interface
- [ ] Dependencies implement `DependencyContainer` interface
- [ ] Configuration updated to new format
- [ ] All required configuration fields set
- [ ] Tests updated to use base server patterns
- [ ] Integration tests pass
- [ ] Service discovery registration works
- [ ] Health checks return correct status
- [ ] Graceful shutdown works properly
- [ ] All existing API endpoints still work
- [ ] Performance is equivalent or better

## Post-Migration Validation

### 1. Functional Testing

Verify all functionality works as expected:

```bash
# Test HTTP endpoints
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:8080/api/v1/your-endpoint

# Test gRPC endpoints
grpcurl -plaintext localhost:9090 list
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

### 2. Performance Testing

Compare performance before and after migration:

- Response times
- Memory usage
- CPU usage
- Startup time
- Shutdown time

### 3. Integration Testing

Verify integration with external systems:

- Database connections
- Service discovery registration
- Load balancer health checks
- Monitoring systems

### 4. Load Testing

Run load tests to ensure performance is maintained:

```bash
# Example with hey
hey -n 1000 -c 10 http://localhost:8080/api/v1/your-endpoint
```

## Rollback Plan

If issues arise during migration:

1. **Keep Old Code**: Maintain old implementation in separate branch
2. **Feature Flags**: Use feature flags to switch between implementations
3. **Gradual Rollout**: Deploy to subset of instances first
4. **Monitoring**: Monitor key metrics during rollout
5. **Quick Rollback**: Have automated rollback procedures ready

## Best Practices

1. **Migrate Incrementally**: One service at a time
2. **Test Thoroughly**: Unit, integration, and load tests
3. **Monitor Closely**: Watch metrics during and after migration
4. **Document Changes**: Update documentation and runbooks
5. **Train Team**: Ensure team understands new patterns
6. **Review Code**: Have migration reviewed by multiple team members

## Getting Help

If you encounter issues during migration:

1. Check this troubleshooting guide
2. Review the [Configuration Reference](configuration-reference.md)
3. Look at the [examples](../examples/) for reference implementations
4. Check existing service migrations (switauth, switserve) for patterns
5. Consult the team or create an issue for complex problems
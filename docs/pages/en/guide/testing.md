# Testing

This comprehensive guide covers testing strategies for the Swit framework, including unit testing, integration testing, performance testing, and best practices for testing microservices built with the framework.

## Overview

The Swit framework provides extensive testing support through:
- **Test Mode Configuration** - Special configurations for testing environments
- **Dynamic Port Allocation** - Automatic port assignment for concurrent tests
- **Mock Dependencies** - Built-in support for dependency mocking
- **Transport Testing** - Utilities for testing HTTP and gRPC transports
- **Integration Test Patterns** - Patterns for comprehensive integration testing

## Testing Configuration

### Test Mode Configuration

```go
// Create test configuration
config := server.NewServerConfig()
config.ServiceName = "test-service"

// Enable test mode for both transports
config.HTTP.TestMode = true
config.HTTP.Port = "0" // Dynamic port allocation
config.HTTP.EnableReady = true // Enable ready channel for synchronization

config.GRPC.TestMode = true 
config.GRPC.Port = "0" // Dynamic port allocation

// Disable service discovery for tests
config.Discovery.Enabled = false

// Reduce timeouts for faster tests
config.ShutdownTimeout = 5 * time.Second
```

### Environment-Specific Test Configuration

```yaml
# config/test.yaml
service_name: "my-service-test"
shutdown_timeout: "5s"

http:
  enabled: true
  port: "0"  # Dynamic allocation
  test_mode: true
  enable_ready: true
  read_timeout: "5s"
  write_timeout: "5s"
  middleware:
    enable_cors: true
    enable_logging: false  # Reduce test noise

grpc:
  enabled: true
  port: "0"  # Dynamic allocation
  test_mode: true
  enable_reflection: true
  enable_health_service: true

discovery:
  enabled: false  # Disable for unit tests
  
middleware:
  enable_logging: false  # Reduce test output
```

## Unit Testing

### Testing Service Registration

```go
package server_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/innovationmech/swit/pkg/server"
)

func TestServiceRegistrar(t *testing.T) {
    // Create mock registry
    registry := &MockBusinessServiceRegistry{
        httpHandlers:   make(map[string]server.BusinessHTTPHandler),
        grpcServices:   make(map[string]server.BusinessGRPCService), 
        healthChecks:   make(map[string]server.BusinessHealthCheck),
    }
    
    // Create test services
    userService := &TestUserService{}
    authService := &TestAuthService{}
    
    registrar := &TestServiceRegistrar{
        userService: userService,
        authService: authService,
    }
    
    // Test service registration
    err := registrar.RegisterServices(registry)
    require.NoError(t, err)
    
    // Verify HTTP handler registration
    assert.Contains(t, registry.httpHandlers, "user-service")
    assert.Equal(t, userService, registry.httpHandlers["user-service"])
    
    // Verify gRPC service registration
    assert.Contains(t, registry.grpcServices, "auth-service")
    assert.Equal(t, authService, registry.grpcServices["auth-service"])
    
    // Verify health check registration
    assert.Len(t, registry.healthChecks, 1)
}

// Mock implementations
type MockBusinessServiceRegistry struct {
    httpHandlers map[string]server.BusinessHTTPHandler
    grpcServices map[string]server.BusinessGRPCService
    healthChecks map[string]server.BusinessHealthCheck
}

func (m *MockBusinessServiceRegistry) RegisterBusinessHTTPHandler(handler server.BusinessHTTPHandler) error {
    m.httpHandlers[handler.GetServiceName()] = handler
    return nil
}

func (m *MockBusinessServiceRegistry) RegisterBusinessGRPCService(service server.BusinessGRPCService) error {
    m.grpcServices[service.GetServiceName()] = service
    return nil
}

func (m *MockBusinessServiceRegistry) RegisterBusinessHealthCheck(check server.BusinessHealthCheck) error {
    m.healthChecks[check.GetServiceName()] = check
    return nil
}
```

### Testing Individual Services

```go
func TestUserService(t *testing.T) {
    // Create test dependencies
    mockDB := &MockDatabase{
        users: map[string]*User{
            "user1": {ID: "user1", Name: "Test User", Email: "test@example.com"},
        },
    }
    
    mockCache := &MockRedisClient{
        data: make(map[string]string),
    }
    
    // Create service with mocked dependencies
    userService := &UserService{
        repository: &UserRepository{DB: mockDB},
        cache:      mockCache,
        logger:     zap.NewNop(),
    }
    
    // Test GetUser method
    ctx := context.Background()
    user, err := userService.GetUser(ctx, "user1")
    
    require.NoError(t, err)
    assert.Equal(t, "user1", user.ID)
    assert.Equal(t, "Test User", user.Name)
    assert.Equal(t, "test@example.com", user.Email)
    
    // Verify cache was updated
    cached, exists := mockCache.data["user:user1"] 
    assert.True(t, exists)
    assert.Contains(t, cached, "Test User")
    
    // Test non-existent user
    user, err = userService.GetUser(ctx, "nonexistent")
    assert.Error(t, err)
    assert.Nil(t, user)
}
```

### Testing with Dependency Injection

```go
func TestServiceWithDependencyInjection(t *testing.T) {
    // Create test container
    container := server.NewSimpleBusinessDependencyContainer()
    
    // Register test dependencies
    container.RegisterInstance("config", &TestConfig{
        DatabaseURL: "sqlite3::memory:",
        Environment: "test",
    })
    
    container.RegisterInstance("logger", zap.NewNop())
    
    container.RegisterSingleton("database", func(c server.BusinessDependencyContainer) (interface{}, error) {
        return &MockDatabase{users: make(map[string]*User)}, nil
    })
    
    container.RegisterTransient("user-repository", func(c server.BusinessDependencyContainer) (interface{}, error) {
        db, _ := c.GetService("database")
        logger, _ := c.GetService("logger")
        return &UserRepository{
            DB:     db.(*MockDatabase),
            Logger: logger.(*zap.Logger),
        }, nil
    })
    
    // Initialize dependencies
    ctx := context.Background()
    err := container.Initialize(ctx)
    require.NoError(t, err)
    defer container.Close()
    
    // Create service using dependency injection
    repo, err := container.GetService("user-repository")
    require.NoError(t, err)
    
    userRepo := repo.(*UserRepository)
    
    // Test repository operations
    user := &User{ID: "test1", Name: "Test User"}
    err = userRepo.Create(ctx, user)
    require.NoError(t, err)
    
    retrieved, err := userRepo.GetByID(ctx, "test1")
    require.NoError(t, err)
    assert.Equal(t, user.Name, retrieved.Name)
}
```

## Integration Testing

### HTTP Transport Integration Testing

```go
func TestHTTPTransportIntegration(t *testing.T) {
    // Create test configuration
    config := server.NewServerConfig()
    config.HTTP.Port = "0" // Dynamic port
    config.HTTP.TestMode = true
    config.HTTP.EnableReady = true
    config.GRPC.Enabled = false // Test HTTP only
    config.Discovery.Enabled = false
    
    // Create test service registrar
    userService := &TestUserService{
        users: map[string]*User{
            "user1": {ID: "user1", Name: "Test User"},
        },
    }
    
    registrar := &TestServiceRegistrar{userService: userService}
    
    // Create and start server
    srv, err := server.NewBusinessServerCore(config, registrar, nil)
    require.NoError(t, err)
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    err = srv.Start(ctx)
    require.NoError(t, err)
    defer srv.Shutdown()
    
    // Wait for server to be ready
    if httpTransport, ok := srv.GetTransports()[0].(*transport.HTTPTransport); ok {
        err = httpTransport.WaitReady()
        require.NoError(t, err)
    }
    
    // Get server address
    httpAddr := srv.GetHTTPAddress()
    baseURL := fmt.Sprintf("http://localhost%s", httpAddr)
    
    // Test health endpoint
    resp, err := http.Get(baseURL + "/health")
    require.NoError(t, err)
    defer resp.Body.Close()
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    // Test API endpoints
    resp, err = http.Get(baseURL + "/api/v1/users")
    require.NoError(t, err)
    defer resp.Body.Close()
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    var users []User
    err = json.NewDecoder(resp.Body).Decode(&users)
    require.NoError(t, err)
    assert.Len(t, users, 1)
    assert.Equal(t, "user1", users[0].ID)
    
    // Test specific user endpoint
    resp, err = http.Get(baseURL + "/api/v1/users/user1")
    require.NoError(t, err)
    defer resp.Body.Close()
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    var user User
    err = json.NewDecoder(resp.Body).Decode(&user)
    require.NoError(t, err)
    assert.Equal(t, "user1", user.ID)
    assert.Equal(t, "Test User", user.Name)
}
```

### gRPC Transport Integration Testing

```go
func TestGRPCTransportIntegration(t *testing.T) {
    // Create test configuration
    config := server.NewServerConfig()
    config.HTTP.Enabled = false // Test gRPC only
    config.GRPC.Port = "0" // Dynamic port
    config.GRPC.TestMode = true
    config.GRPC.EnableReflection = true
    config.GRPC.EnableHealthService = true
    config.Discovery.Enabled = false
    
    // Create test auth service
    authService := &TestAuthService{
        users: map[string]string{
            "testuser": "testpass",
        },
        tokens: make(map[string]string),
    }
    
    registrar := &TestServiceRegistrar{authService: authService}
    
    // Create and start server
    srv, err := server.NewBusinessServerCore(config, registrar, nil)
    require.NoError(t, err)
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    err = srv.Start(ctx)
    require.NoError(t, err)
    defer srv.Shutdown()
    
    // Get gRPC address
    grpcAddr := srv.GetGRPCAddress()
    
    // Create gRPC client connection
    conn, err := grpc.DialContext(ctx, grpcAddr, 
        grpc.WithInsecure(),
        grpc.WithBlock(),
    )
    require.NoError(t, err)
    defer conn.Close()
    
    // Test health service
    healthClient := healthpb.NewHealthClient(conn)
    healthResp, err := healthClient.Check(ctx, &healthpb.HealthCheckRequest{})
    require.NoError(t, err)
    assert.Equal(t, healthpb.HealthCheckResponse_SERVING, healthResp.Status)
    
    // Test auth service
    authClient := authpb.NewAuthServiceClient(conn)
    
    // Test login
    loginResp, err := authClient.Login(ctx, &authpb.LoginRequest{
        Username: "testuser",
        Password: "testpass",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, loginResp.Token)
    assert.Equal(t, "testuser", loginResp.User.Username)
    
    // Test token validation
    validateResp, err := authClient.ValidateToken(ctx, &authpb.ValidateTokenRequest{
        Token: loginResp.Token,
    })
    require.NoError(t, err)
    assert.True(t, validateResp.Valid)
    assert.Equal(t, "testuser", validateResp.UserId)
    
    // Test invalid login
    _, err = authClient.Login(ctx, &authpb.LoginRequest{
        Username: "invalid",
        Password: "invalid",
    })
    require.Error(t, err)
    assert.Contains(t, err.Error(), "invalid credentials")
}
```

### Multi-Transport Integration Testing

```go
func TestMultiTransportIntegration(t *testing.T) {
    // Create configuration with both transports
    config := server.NewServerConfig()
    config.HTTP.Port = "0"
    config.HTTP.TestMode = true
    config.GRPC.Port = "0"
    config.GRPC.TestMode = true
    config.Discovery.Enabled = false
    
    // Create services that support both transports
    userService := &TestUserService{
        users: map[string]*User{
            "user1": {ID: "user1", Name: "Test User"},
        },
    }
    
    authService := &TestAuthService{
        users: map[string]string{
            "testuser": "testpass",
        },
    }
    
    registrar := &TestServiceRegistrar{
        userService: userService,
        authService: authService,
    }
    
    // Start server
    srv, err := server.NewBusinessServerCore(config, registrar, nil)
    require.NoError(t, err)
    
    ctx := context.Background()
    err = srv.Start(ctx)
    require.NoError(t, err)
    defer srv.Shutdown()
    
    // Test HTTP transport
    httpAddr := srv.GetHTTPAddress()
    resp, err := http.Get(fmt.Sprintf("http://localhost%s/api/v1/users", httpAddr))
    require.NoError(t, err)
    defer resp.Body.Close()
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    // Test gRPC transport
    grpcAddr := srv.GetGRPCAddress()
    conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
    require.NoError(t, err)
    defer conn.Close()
    
    authClient := authpb.NewAuthServiceClient(conn)
    loginResp, err := authClient.Login(ctx, &authpb.LoginRequest{
        Username: "testuser",
        Password: "testpass", 
    })
    require.NoError(t, err)
    assert.NotEmpty(t, loginResp.Token)
    
    // Verify both transports are running
    transports := srv.GetTransports()
    assert.Len(t, transports, 2)
    
    transportNames := make([]string, len(transports))
    for i, transport := range transports {
        transportNames[i] = transport.GetName()
    }
    
    assert.Contains(t, transportNames, "http")
    assert.Contains(t, transportNames, "grpc")
}
```

## Performance Testing

### Load Testing Setup

```go
func TestServerLoadTesting(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }
    
    // Start test server
    srv, httpAddr := startTestServer(t)
    defer srv.Shutdown()
    
    // Configure load test parameters
    const (
        numClients    = 50
        requestsPerClient = 100
        totalRequests = numClients * requestsPerClient
    )
    
    // Create HTTP client with connection pooling
    client := &http.Client{
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     30 * time.Second,
        },
        Timeout: 10 * time.Second,
    }
    
    // Track metrics
    var (
        successCount int64
        errorCount   int64
        totalLatency int64
    )
    
    // Create worker goroutines
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, numClients)
    
    startTime := time.Now()
    
    for i := 0; i < totalRequests; i++ {
        wg.Add(1)
        semaphore <- struct{}{} // Limit concurrency
        
        go func() {
            defer func() {
                <-semaphore
                wg.Done()
            }()
            
            requestStart := time.Now()
            resp, err := client.Get(fmt.Sprintf("http://localhost%s/api/v1/users", httpAddr))
            requestDuration := time.Since(requestStart)
            
            atomic.AddInt64(&totalLatency, requestDuration.Nanoseconds())
            
            if err != nil {
                atomic.AddInt64(&errorCount, 1)
                return
            }
            defer resp.Body.Close()
            
            if resp.StatusCode == http.StatusOK {
                atomic.AddInt64(&successCount, 1)
            } else {
                atomic.AddInt64(&errorCount, 1)
            }
        }()
    }
    
    wg.Wait()
    totalDuration := time.Since(startTime)
    
    // Calculate metrics
    avgLatency := time.Duration(totalLatency / totalRequests)
    requestsPerSecond := float64(totalRequests) / totalDuration.Seconds()
    errorRate := float64(errorCount) / float64(totalRequests) * 100
    
    // Log results
    t.Logf("Load Test Results:")
    t.Logf("Total Requests: %d", totalRequests)
    t.Logf("Successful Requests: %d", successCount)
    t.Logf("Failed Requests: %d", errorCount)
    t.Logf("Error Rate: %.2f%%", errorRate)
    t.Logf("Total Duration: %v", totalDuration)
    t.Logf("Requests/Second: %.2f", requestsPerSecond)
    t.Logf("Average Latency: %v", avgLatency)
    
    // Assert performance thresholds
    assert.Less(t, errorRate, 1.0, "Error rate should be less than 1%")
    assert.Greater(t, requestsPerSecond, 100.0, "Should handle at least 100 requests/second")
    assert.Less(t, avgLatency, 100*time.Millisecond, "Average latency should be less than 100ms")
}
```

### Memory and Resource Testing

```go
func TestServerMemoryUsage(t *testing.T) {
    // Start test server
    srv, _ := startTestServer(t)
    defer srv.Shutdown()
    
    // Get baseline memory
    runtime.GC()
    var baseline runtime.MemStats
    runtime.ReadMemStats(&baseline)
    
    // Simulate workload
    const numOperations = 1000
    for i := 0; i < numOperations; i++ {
        // Simulate memory allocation
        data := make([]byte, 1024)
        _ = data
        
        // Force periodic garbage collection
        if i%100 == 0 {
            runtime.GC()
        }
    }
    
    // Measure final memory
    runtime.GC()
    var final runtime.MemStats
    runtime.ReadMemStats(&final)
    
    // Calculate memory growth
    memoryGrowth := final.Alloc - baseline.Alloc
    
    t.Logf("Memory Usage:")
    t.Logf("Baseline Alloc: %d bytes", baseline.Alloc)
    t.Logf("Final Alloc: %d bytes", final.Alloc)
    t.Logf("Memory Growth: %d bytes", memoryGrowth)
    t.Logf("GC Cycles: %d", final.NumGC-baseline.NumGC)
    
    // Assert memory usage is reasonable
    maxAllowedGrowth := uint64(10 * 1024 * 1024) // 10MB
    assert.Less(t, memoryGrowth, maxAllowedGrowth,
        "Memory growth should be less than %d bytes", maxAllowedGrowth)
}
```

### Goroutine Leak Testing

```go
func TestGoroutineLeaks(t *testing.T) {
    // Get baseline goroutine count
    baselineGoroutines := runtime.NumGoroutine()
    
    // Start and stop server multiple times
    for i := 0; i < 5; i++ {
        srv, _ := startTestServer(t)
        
        // Perform some operations
        time.Sleep(100 * time.Millisecond)
        
        // Shutdown server
        err := srv.Shutdown()
        require.NoError(t, err)
        
        // Wait for cleanup
        time.Sleep(100 * time.Millisecond)
    }
    
    // Force garbage collection
    runtime.GC()
    time.Sleep(100 * time.Millisecond)
    
    // Check final goroutine count
    finalGoroutines := runtime.NumGoroutine()
    goroutineGrowth := finalGoroutines - baselineGoroutines
    
    t.Logf("Goroutine count:")
    t.Logf("Baseline: %d", baselineGoroutines)
    t.Logf("Final: %d", finalGoroutines)
    t.Logf("Growth: %d", goroutineGrowth)
    
    // Assert no significant goroutine leaks
    maxAllowedGrowth := 5
    assert.LessOrEqual(t, goroutineGrowth, maxAllowedGrowth,
        "Goroutine growth should be less than or equal to %d", maxAllowedGrowth)
}
```

## Test Utilities and Helpers

### Test Server Factory

```go
func startTestServer(t *testing.T) (server.BusinessServerCore, string) {
    config := server.NewServerConfig()
    config.HTTP.Port = "0"
    config.HTTP.TestMode = true
    config.HTTP.EnableReady = true
    config.GRPC.Port = "0"
    config.GRPC.TestMode = true
    config.Discovery.Enabled = false
    
    userService := &TestUserService{
        users: map[string]*User{
            "user1": {ID: "user1", Name: "Test User"},
            "user2": {ID: "user2", Name: "Another User"},
        },
    }
    
    registrar := &TestServiceRegistrar{userService: userService}
    
    srv, err := server.NewBusinessServerCore(config, registrar, nil)
    require.NoError(t, err)
    
    ctx := context.Background()
    err = srv.Start(ctx)
    require.NoError(t, err)
    
    // Wait for server to be ready
    httpAddr := srv.GetHTTPAddress()
    
    return srv, httpAddr
}
```

### Test Data Factories

```go
type TestDataFactory struct {
    userCounter int
}

func NewTestDataFactory() *TestDataFactory {
    return &TestDataFactory{}
}

func (f *TestDataFactory) CreateUser() *User {
    f.userCounter++
    return &User{
        ID:    fmt.Sprintf("user%d", f.userCounter),
        Name:  fmt.Sprintf("Test User %d", f.userCounter),
        Email: fmt.Sprintf("user%d@example.com", f.userCounter),
    }
}

func (f *TestDataFactory) CreateUsers(count int) []*User {
    users := make([]*User, count)
    for i := 0; i < count; i++ {
        users[i] = f.CreateUser()
    }
    return users
}
```

### Test Assertion Helpers

```go
func AssertHTTPResponse(t *testing.T, resp *http.Response, expectedStatus int, expectedContentType string) {
    assert.Equal(t, expectedStatus, resp.StatusCode)
    if expectedContentType != "" {
        assert.Equal(t, expectedContentType, resp.Header.Get("Content-Type"))
    }
}

func AssertJSONResponse(t *testing.T, resp *http.Response, target interface{}) {
    defer resp.Body.Close()
    err := json.NewDecoder(resp.Body).Decode(target)
    require.NoError(t, err)
}

func AssertGRPCError(t *testing.T, err error, expectedCode codes.Code) {
    require.Error(t, err)
    status, ok := status.FromError(err)
    require.True(t, ok)
    assert.Equal(t, expectedCode, status.Code())
}
```

## Testing Best Practices

### Test Organization

1. **Test Structure** - Organize tests by component (unit, integration, performance)
2. **Test Naming** - Use descriptive test names that explain what is being tested
3. **Test Data** - Use factories or builders for creating test data
4. **Test Cleanup** - Always clean up resources after tests
5. **Test Isolation** - Ensure tests don't depend on each other

### Mock Strategies

1. **Interface Mocking** - Mock dependencies through interfaces
2. **Dependency Injection** - Use DI to inject mocks
3. **Test Doubles** - Use appropriate test doubles (stubs, mocks, fakes)
4. **Mock Verification** - Verify mock interactions when appropriate
5. **Mock Cleanup** - Reset mocks between tests

### Performance Testing

1. **Baseline Establishment** - Establish performance baselines
2. **Load Patterns** - Test with realistic load patterns
3. **Resource Monitoring** - Monitor memory, CPU, and goroutines
4. **Threshold Setting** - Set realistic performance thresholds
5. **Regression Detection** - Include performance tests in CI/CD

### Integration Testing

1. **Environment Parity** - Keep test environment close to production
2. **External Dependencies** - Use testcontainers for external dependencies
3. **Data Management** - Use appropriate test data management strategies
4. **Test Coverage** - Ensure comprehensive coverage of integration points
5. **Error Scenarios** - Test error handling and edge cases

This comprehensive testing guide covers all aspects of testing Swit framework applications, from unit tests to performance testing, with practical examples and best practices for building robust, well-tested microservices.
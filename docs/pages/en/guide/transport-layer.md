# Transport Layer

This guide covers the transport layer in Swit, which provides a unified abstraction for managing HTTP and gRPC transports with service registration, lifecycle management, and error handling.

## Overview

The transport layer (`pkg/transport`) implements a pluggable architecture that allows services to register with multiple transport types seamlessly through a centralized coordination system. It provides unified lifecycle management, error handling, and service health monitoring across different transport protocols.

## Architecture Components

### NetworkTransport Interface

The base interface for all transport implementations:

```go
type NetworkTransport interface {
    Start(ctx context.Context) error    // Start the transport server
    Stop(ctx context.Context) error     // Gracefully stop the transport
    GetName() string                    // Returns transport name ("http" or "grpc")
    GetAddress() string                 // Returns listening address
}
```

### TransportCoordinator

Central coordinator managing multiple transport instances:

```go
// Create transport coordinator
coordinator := transport.NewTransportCoordinator()

// Register transports
coordinator.Register(httpTransport)
coordinator.Register(grpcTransport)

// Start all transports
err := coordinator.Start(ctx)

// Stop all transports with timeout
err = coordinator.Stop(30 * time.Second)
```

### Service Handler System

Unified service registration interface:

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

## HTTP Transport

### Basic HTTP Transport Setup

```go
import (
    "github.com/innovationmech/swit/pkg/transport"
)

// Create HTTP transport with default settings
httpTransport := transport.NewHTTPTransport(":8080")

// Create HTTP transport with custom configuration
httpConfig := &transport.HTTPTransportConfig{
    Address:     ":8080",
    TestMode:    false,
    EnableReady: true,
}
httpTransport := transport.NewHTTPTransportWithConfig(httpConfig)

// Register with coordinator
coordinator.Register(httpTransport)
```

### HTTP Service Implementation

```go
type UserHTTPService struct {
    userRepo *UserRepository
    logger   *zap.Logger
}

func (s *UserHTTPService) RegisterHTTP(router *gin.Engine) error {
    v1 := router.Group("/api/v1")
    {
        users := v1.Group("/users")
        {
            users.GET("", s.getUsers)
            users.POST("", s.createUser)
            users.GET("/:id", s.getUser)
            users.PUT("/:id", s.updateUser)
            users.DELETE("/:id", s.deleteUser)
        }
    }
    return nil
}

func (s *UserHTTPService) RegisterGRPC(server *grpc.Server) error {
    // Not used for HTTP-only service
    return nil
}

func (s *UserHTTPService) GetMetadata() *transport.HandlerMetadata {
    return &transport.HandlerMetadata{
        Name:           "user-http-service",
        Version:        "v1.0.0",
        Description:    "User management HTTP API",
        HealthEndpoint: "/api/v1/users/health",
        Tags:           []string{"users", "http", "api"},
        Dependencies:   []string{"database", "redis"},
    }
}

func (s *UserHTTPService) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
    // Check database connection
    if err := s.userRepo.Ping(ctx); err != nil {
        return &types.HealthStatus{
            Status:  "unhealthy",
            Message: "Database connection failed",
        }, err
    }
    
    return &types.HealthStatus{
        Status:  "healthy",
        Message: "All dependencies are healthy",
    }, nil
}

func (s *UserHTTPService) Initialize(ctx context.Context) error {
    s.logger.Info("Initializing user HTTP service")
    return s.userRepo.Initialize(ctx)
}

func (s *UserHTTPService) Shutdown(ctx context.Context) error {
    s.logger.Info("Shutting down user HTTP service")
    return s.userRepo.Close()
}
```

### HTTP Route Handlers

```go
func (s *UserHTTPService) getUsers(c *gin.Context) {
    ctx := c.Request.Context()
    
    // Parse query parameters
    limit := c.DefaultQuery("limit", "10")
    offset := c.DefaultQuery("offset", "0")
    
    limitInt, _ := strconv.Atoi(limit)
    offsetInt, _ := strconv.Atoi(offset)
    
    // Get users from repository
    users, total, err := s.userRepo.GetUsers(ctx, limitInt, offsetInt)
    if err != nil {
        s.logger.Error("Failed to get users", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to retrieve users",
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "users": users,
        "total": total,
        "limit": limitInt,
        "offset": offsetInt,
    })
}

func (s *UserHTTPService) createUser(c *gin.Context) {
    ctx := c.Request.Context()
    
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid request format",
            "details": err.Error(),
        })
        return
    }
    
    // Validate request
    if err := req.Validate(); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Validation failed",
            "details": err.Error(),
        })
        return
    }
    
    // Create user
    user, err := s.userRepo.CreateUser(ctx, &req)
    if err != nil {
        s.logger.Error("Failed to create user", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to create user",
        })
        return
    }
    
    c.JSON(http.StatusCreated, gin.H{
        "user": user,
    })
}
```

## gRPC Transport

### Basic gRPC Transport Setup

```go
// Create gRPC transport with default settings
grpcTransport := transport.NewGRPCTransport(":9080")

// Create gRPC transport with custom configuration
grpcConfig := &transport.GRPCTransportConfig{
    Address:             ":9080",
    EnableKeepalive:     true,
    EnableReflection:    true,
    EnableHealthService: true,
    MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
    MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
    KeepaliveParams: &keepalive.ServerParameters{
        MaxConnectionIdle:     15 * time.Second,
        MaxConnectionAge:      30 * time.Second,
        MaxConnectionAgeGrace: 5 * time.Second,
        Time:                  5 * time.Second,
        Timeout:               1 * time.Second,
    },
}
grpcTransport := transport.NewGRPCTransportWithConfig(grpcConfig)

// Register with coordinator
coordinator.Register(grpcTransport)
```

### gRPC Service Implementation

```go
type AuthGRPCService struct {
    authpb.UnimplementedAuthServiceServer
    tokenManager *TokenManager
    userRepo     *UserRepository
    logger       *zap.Logger
}

func (s *AuthGRPCService) RegisterHTTP(router *gin.Engine) error {
    // Not used for gRPC-only service
    return nil
}

func (s *AuthGRPCService) RegisterGRPC(server *grpc.Server) error {
    authpb.RegisterAuthServiceServer(server, s)
    return nil
}

func (s *AuthGRPCService) GetMetadata() *transport.HandlerMetadata {
    return &transport.HandlerMetadata{
        Name:           "auth-grpc-service",
        Version:        "v1.0.0",
        Description:    "Authentication gRPC service",
        HealthEndpoint: "/grpc.health.v1.Health/Check",
        Tags:           []string{"auth", "grpc", "security"},
        Dependencies:   []string{"database", "redis"},
    }
}

func (s *AuthGRPCService) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
    // Check token manager
    if err := s.tokenManager.Ping(ctx); err != nil {
        return &types.HealthStatus{
            Status:  "unhealthy",
            Message: "Token manager not available",
        }, err
    }
    
    // Check user repository
    if err := s.userRepo.Ping(ctx); err != nil {
        return &types.HealthStatus{
            Status:  "unhealthy",
            Message: "User repository not available",
        }, err
    }
    
    return &types.HealthStatus{
        Status:  "healthy",
        Message: "All dependencies are healthy",
    }, nil
}
```

### gRPC Method Implementation

```go
func (s *AuthGRPCService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
    // Validate input
    if req.Username == "" || req.Password == "" {
        return nil, status.Errorf(codes.InvalidArgument, "username and password are required")
    }
    
    // Authenticate user
    user, err := s.userRepo.ValidateCredentials(ctx, req.Username, req.Password)
    if err != nil {
        s.logger.Error("Authentication failed", 
            zap.String("username", req.Username),
            zap.Error(err))
        return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
    }
    
    // Generate token
    token, expiresAt, err := s.tokenManager.GenerateToken(ctx, user.ID)
    if err != nil {
        s.logger.Error("Token generation failed", 
            zap.String("userID", user.ID),
            zap.Error(err))
        return nil, status.Errorf(codes.Internal, "failed to generate token")
    }
    
    return &authpb.LoginResponse{
        Token:     token,
        ExpiresAt: expiresAt.Unix(),
        User: &authpb.User{
            Id:       user.ID,
            Username: user.Username,
            Email:    user.Email,
            Role:     user.Role,
        },
    }, nil
}

func (s *AuthGRPCService) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
    if req.Token == "" {
        return nil, status.Errorf(codes.InvalidArgument, "token is required")
    }
    
    // Validate token
    claims, err := s.tokenManager.ValidateToken(ctx, req.Token)
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "invalid token")
    }
    
    return &authpb.ValidateTokenResponse{
        Valid:  true,
        UserId: claims.UserID,
        Role:   claims.Role,
    }, nil
}
```

## Multi-Transport Services

### Hybrid Service Implementation

Services can support both HTTP and gRPC transports:

```go
type UserManagementService struct {
    userpb.UnimplementedUserServiceServer
    userRepo *UserRepository
    logger   *zap.Logger
}

func (s *UserManagementService) RegisterHTTP(router *gin.Engine) error {
    v1 := router.Group("/api/v1/users")
    {
        v1.GET("", s.getUsersHTTP)
        v1.POST("", s.createUserHTTP)
        v1.GET("/:id", s.getUserHTTP)
        v1.PUT("/:id", s.updateUserHTTP)
        v1.DELETE("/:id", s.deleteUserHTTP)
    }
    return nil
}

func (s *UserManagementService) RegisterGRPC(server *grpc.Server) error {
    userpb.RegisterUserServiceServer(server, s)
    return nil
}

// HTTP handlers
func (s *UserManagementService) getUsersHTTP(c *gin.Context) {
    // HTTP-specific implementation
}

func (s *UserManagementService) createUserHTTP(c *gin.Context) {
    // HTTP-specific implementation
}

// gRPC handlers
func (s *UserManagementService) GetUsers(ctx context.Context, req *userpb.GetUsersRequest) (*userpb.GetUsersResponse, error) {
    // gRPC-specific implementation
}

func (s *UserManagementService) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
    // gRPC-specific implementation
}
```

## Service Registration and Lifecycle

### Complete Service Lifecycle

```go
// Create transport coordinator
coordinator := transport.NewTransportCoordinator()

// Create and register transports
httpTransport := transport.NewHTTPTransport(":8080")
grpcTransport := transport.NewGRPCTransport(":9080")

coordinator.Register(httpTransport)
coordinator.Register(grpcTransport)

// Register services
userService := &UserManagementService{
    userRepo: userRepo,
    logger:   logger,
}

authService := &AuthGRPCService{
    tokenManager: tokenManager,
    userRepo:     userRepo,
    logger:       logger,
}

coordinator.RegisterHTTPService(userService)
coordinator.RegisterGRPCService(userService)
coordinator.RegisterGRPCService(authService)

// Start transports
ctx := context.Background()
if err := coordinator.Start(ctx); err != nil {
    log.Fatalf("Failed to start transports: %v", err)
}

// Initialize services
if err := coordinator.InitializeTransportServices(ctx); err != nil {
    log.Fatalf("Failed to initialize services: %v", err)
}

// Bind services to transports
httpRouter := gin.New()
if err := coordinator.BindAllHTTPEndpoints(httpRouter); err != nil {
    log.Fatalf("Failed to bind HTTP endpoints: %v", err)
}

grpcServer := grpc.NewServer()
if err := coordinator.BindAllGRPCServices(grpcServer); err != nil {
    log.Fatalf("Failed to bind gRPC services: %v", err)
}

// Health checks
healthStatus := coordinator.CheckAllServicesHealth(ctx)
log.Printf("Service health: %+v", healthStatus)

// Graceful shutdown
defer func() {
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    coordinator.ShutdownAllServices(shutdownCtx)
    coordinator.Stop(30 * time.Second)
}()
```

## Error Handling

### Transport Error Types

```go
// Single transport error
type TransportError struct {
    TransportName string
    Err           error
}

// Multiple transport errors
type MultiError struct {
    Errors []TransportError
}
```

### Error Handling Patterns

```go
if err := coordinator.Stop(timeout); err != nil {
    if multiErr, ok := err.(*transport.MultiError); ok {
        // Handle multiple transport errors
        for _, transportErr := range multiErr.Errors {
            log.Printf("Transport %s error: %v", transportErr.TransportName, transportErr.Err)
        }
        
        // Check specific transport errors
        if httpErr := multiErr.GetErrorByTransport("http"); httpErr != nil {
            log.Printf("HTTP transport error: %v", httpErr.Err)
        }
        
        if grpcErr := multiErr.GetErrorByTransport("grpc"); grpcErr != nil {
            log.Printf("gRPC transport error: %v", grpcErr.Err)
        }
    }
}

// Use utility functions
if transport.IsStopError(err) {
    stopErrors := transport.ExtractStopErrors(err)
    for _, stopErr := range stopErrors {
        log.Printf("Stop error in %s: %v", stopErr.TransportName, stopErr.Err)
    }
}
```

## Testing Transport Layer

### HTTP Transport Testing

```go
func TestHTTPTransport(t *testing.T) {
    // Create test transport with dynamic port
    transport := transport.NewHTTPTransportWithConfig(&transport.HTTPTransportConfig{
        Address:     ":0", // Dynamic port allocation
        TestMode:    true,
        EnableReady: true,
    })
    
    // Start transport
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    err := transport.Start(ctx)
    require.NoError(t, err)
    defer transport.Stop(ctx)
    
    // Wait for transport to be ready
    err = transport.WaitReady()
    require.NoError(t, err)
    
    // Test HTTP requests
    address := transport.GetAddress()
    resp, err := http.Get(fmt.Sprintf("http://localhost%s/health", address))
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

### gRPC Transport Testing

```go
func TestGRPCTransport(t *testing.T) {
    // Create test transport with dynamic port
    transport := transport.NewGRPCTransportWithConfig(&transport.GRPCTransportConfig{
        Address:             ":0", // Dynamic port allocation
        EnableHealthService: true,
        EnableReflection:    true,
    })
    
    // Start transport
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    err := transport.Start(ctx)
    require.NoError(t, err)
    defer transport.Stop(ctx)
    
    // Test gRPC connection
    address := transport.GetAddress()
    conn, err := grpc.Dial(address, grpc.WithInsecure())
    require.NoError(t, err)
    defer conn.Close()
    
    // Test health service
    healthClient := healthpb.NewHealthClient(conn)
    resp, err := healthClient.Check(ctx, &healthpb.HealthCheckRequest{})
    require.NoError(t, err)
    assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.Status)
}
```

### Service Integration Testing

```go
func TestServiceIntegration(t *testing.T) {
    // Create coordinator
    coordinator := transport.NewTransportCoordinator()
    
    // Create test transports
    httpTransport := transport.NewHTTPTransportWithConfig(&transport.HTTPTransportConfig{
        Address:  ":0",
        TestMode: true,
    })
    
    grpcTransport := transport.NewGRPCTransportWithConfig(&transport.GRPCTransportConfig{
        Address:             ":0",
        EnableHealthService: true,
    })
    
    coordinator.Register(httpTransport)
    coordinator.Register(grpcTransport)
    
    // Register test service
    testService := &TestMultiTransportService{}
    coordinator.RegisterHTTPService(testService)
    coordinator.RegisterGRPCService(testService)
    
    // Start everything
    ctx := context.Background()
    err := coordinator.Start(ctx)
    require.NoError(t, err)
    defer coordinator.Stop(30 * time.Second)
    
    err = coordinator.InitializeTransportServices(ctx)
    require.NoError(t, err)
    
    // Test both transports
    httpAddr := httpTransport.GetAddress()
    grpcAddr := grpcTransport.GetAddress()
    
    // Test HTTP
    resp, err := http.Get(fmt.Sprintf("http://localhost%s/api/test", httpAddr))
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    // Test gRPC
    conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
    require.NoError(t, err)
    defer conn.Close()
    
    client := testpb.NewTestServiceClient(conn)
    grpcResp, err := client.TestMethod(ctx, &testpb.TestRequest{})
    require.NoError(t, err)
    assert.NotNil(t, grpcResp)
}
```

## Best Practices

### Service Design

1. **Transport Independence** - Design services to work with either transport
2. **Proper Error Handling** - Return appropriate HTTP status codes and gRPC status codes
3. **Context Propagation** - Always use and propagate context
4. **Health Checks** - Implement meaningful health checks
5. **Metadata Management** - Provide accurate service metadata

### Performance Optimization

1. **Connection Pooling** - Use connection pooling for database and external services
2. **Resource Management** - Properly close resources and handle cleanup
3. **Timeout Configuration** - Set appropriate timeouts for operations
4. **Message Size Limits** - Configure appropriate message size limits for gRPC
5. **Keepalive Settings** - Tune keepalive parameters for your network conditions

### Testing Strategy

1. **Unit Testing** - Test service logic independently of transports
2. **Integration Testing** - Test with actual transport implementations
3. **Dynamic Ports** - Use dynamic port allocation for concurrent test runs
4. **Mock Dependencies** - Mock external dependencies for reliable testing
5. **Health Check Testing** - Test health check implementations

### Production Readiness

1. **Graceful Shutdown** - Implement proper shutdown handling
2. **Error Recovery** - Handle transport errors gracefully
3. **Monitoring Integration** - Integrate with monitoring systems
4. **Security Configuration** - Configure TLS and authentication properly
5. **Resource Limits** - Set appropriate resource limits and timeouts

This transport layer guide provides comprehensive coverage of HTTP and gRPC transport implementations, service registration patterns, error handling, testing strategies, and production deployment considerations.
# 传输层

本指南涵盖 Swit 中的传输层，它为管理 HTTP 和 gRPC 传输提供统一抽象，具有服务注册、生命周期管理和错误处理功能。

## 概述

传输层 (`pkg/transport`) 实现了一个可插拔的架构，允许服务通过集中的协调系统无缝地注册到多种传输类型。它提供统一的生命周期管理、错误处理和跨不同传输协议的服务健康监控。

## 架构组件

### NetworkTransport 接口

所有传输实现的基础接口：

```go
type NetworkTransport interface {
    Start(ctx context.Context) error    // 启动传输服务器
    Stop(ctx context.Context) error     // 优雅停止传输
    GetName() string                    // 返回传输名称 ("http" 或 "grpc")
    GetAddress() string                 // 返回监听地址
}
```

### TransportCoordinator

管理多个传输实例的中央协调器：

```go
// 创建传输协调器
coordinator := transport.NewTransportCoordinator()

// 注册传输
coordinator.Register(httpTransport)
coordinator.Register(grpcTransport)

// 启动所有传输
err := coordinator.Start(ctx)

// 带超时停止所有传输
err = coordinator.Stop(30 * time.Second)
```

### 服务处理程序系统

统一的服务注册接口：

```go
type TransportServiceHandler interface {
    RegisterHTTP(router *gin.Engine) error              // 注册 HTTP 路由
    RegisterGRPC(server *grpc.Server) error            // 注册 gRPC 服务
    GetMetadata() *HandlerMetadata                      // 服务元数据
    GetHealthEndpoint() string                          // 健康检查端点
    IsHealthy(ctx context.Context) (*types.HealthStatus, error) // 健康检查
    Initialize(ctx context.Context) error               // 服务初始化
    Shutdown(ctx context.Context) error                 // 服务关闭
}
```

## HTTP 传输

### 基本 HTTP 传输设置

```go
import (
    "github.com/innovationmech/swit/pkg/transport"
)

// 使用默认设置创建 HTTP 传输
httpTransport := transport.NewHTTPTransport(":8080")

// 使用自定义配置创建 HTTP 传输
httpConfig := &transport.HTTPTransportConfig{
    Address:     ":8080",
    TestMode:    false,
    EnableReady: true,
}
httpTransport := transport.NewHTTPTransportWithConfig(httpConfig)

// 向协调器注册
coordinator.Register(httpTransport)
```

### HTTP 服务实现

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
    // HTTP 专用服务不使用
    return nil
}

func (s *UserHTTPService) GetMetadata() *transport.HandlerMetadata {
    return &transport.HandlerMetadata{
        Name:           "user-http-service",
        Version:        "v1.0.0",
        Description:    "用户管理 HTTP API",
        HealthEndpoint: "/api/v1/users/health",
        Tags:           []string{"users", "http", "api"},
        Dependencies:   []string{"database", "redis"},
    }
}

func (s *UserHTTPService) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
    // 检查数据库连接
    if err := s.userRepo.Ping(ctx); err != nil {
        return &types.HealthStatus{
            Status:  "unhealthy",
            Message: "数据库连接失败",
        }, err
    }
    
    return &types.HealthStatus{
        Status:  "healthy",
        Message: "所有依赖项都健康",
    }, nil
}
```

## gRPC 传输

### 基本 gRPC 传输设置

```go
// 使用默认设置创建 gRPC 传输
grpcTransport := transport.NewGRPCTransport(":9080")

// 使用自定义配置创建 gRPC 传输
grpcConfig := &transport.GRPCTransportConfig{
    Address:             ":9080",
    EnableKeepalive:     true,
    EnableReflection:    true,
    EnableHealthService: true,
    MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
    MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
}
grpcTransport := transport.NewGRPCTransportWithConfig(grpcConfig)

// 向协调器注册
coordinator.Register(grpcTransport)
```

### gRPC 服务实现

```go
type AuthGRPCService struct {
    authpb.UnimplementedAuthServiceServer
    tokenManager *TokenManager
    userRepo     *UserRepository
    logger       *zap.Logger
}

func (s *AuthGRPCService) RegisterHTTP(router *gin.Engine) error {
    // gRPC 专用服务不使用
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
        Description:    "认证 gRPC 服务",
        HealthEndpoint: "/grpc.health.v1.Health/Check",
        Tags:           []string{"auth", "grpc", "security"},
        Dependencies:   []string{"database", "redis"},
    }
}
```

## 多传输服务

### 混合服务实现

服务可以同时支持 HTTP 和 gRPC 传输：

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

// HTTP 处理程序
func (s *UserManagementService) getUsersHTTP(c *gin.Context) {
    // HTTP 特定实现
}

// gRPC 处理程序
func (s *UserManagementService) GetUsers(ctx context.Context, req *userpb.GetUsersRequest) (*userpb.GetUsersResponse, error) {
    // gRPC 特定实现
}
```

## 服务注册和生命周期

### 完整的服务生命周期

```go
// 创建传输协调器
coordinator := transport.NewTransportCoordinator()

// 创建并注册传输
httpTransport := transport.NewHTTPTransport(":8080")
grpcTransport := transport.NewGRPCTransport(":9080")

coordinator.Register(httpTransport)
coordinator.Register(grpcTransport)

// 注册服务
userService := &UserManagementService{
    userRepo: userRepo,
    logger:   logger,
}

coordinator.RegisterHTTPService(userService)
coordinator.RegisterGRPCService(userService)

// 启动传输
ctx := context.Background()
if err := coordinator.Start(ctx); err != nil {
    log.Fatalf("启动传输失败: %v", err)
}

// 初始化服务
if err := coordinator.InitializeTransportServices(ctx); err != nil {
    log.Fatalf("初始化服务失败: %v", err)
}

// 将服务绑定到传输
httpRouter := gin.New()
if err := coordinator.BindAllHTTPEndpoints(httpRouter); err != nil {
    log.Fatalf("绑定 HTTP 端点失败: %v", err)
}

grpcServer := grpc.NewServer()
if err := coordinator.BindAllGRPCServices(grpcServer); err != nil {
    log.Fatalf("绑定 gRPC 服务失败: %v", err)
}

// 健康检查
healthStatus := coordinator.CheckAllServicesHealth(ctx)
log.Printf("服务健康状况: %+v", healthStatus)

// 优雅关闭
defer func() {
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    coordinator.ShutdownAllServices(shutdownCtx)
    coordinator.Stop(30 * time.Second)
}()
```

## 错误处理

### 传输错误类型

```go
// 单个传输错误
type TransportError struct {
    TransportName string
    Err           error
}

// 多个传输错误
type MultiError struct {
    Errors []TransportError
}
```

### 错误处理模式

```go
if err := coordinator.Stop(timeout); err != nil {
    if multiErr, ok := err.(*transport.MultiError); ok {
        // 处理多个传输错误
        for _, transportErr := range multiErr.Errors {
            log.Printf("传输 %s 错误: %v", transportErr.TransportName, transportErr.Err)
        }
        
        // 检查特定传输错误
        if httpErr := multiErr.GetErrorByTransport("http"); httpErr != nil {
            log.Printf("HTTP 传输错误: %v", httpErr.Err)
        }
    }
}

// 使用实用函数
if transport.IsStopError(err) {
    stopErrors := transport.ExtractStopErrors(err)
    for _, stopErr := range stopErrors {
        log.Printf("%s 中的停止错误: %v", stopErr.TransportName, stopErr.Err)
    }
}
```

## 测试传输层

### HTTP 传输测试

```go
func TestHTTPTransport(t *testing.T) {
    // 使用动态端口创建测试传输
    transport := transport.NewHTTPTransportWithConfig(&transport.HTTPTransportConfig{
        Address:     ":0", // 动态端口分配
        TestMode:    true,
        EnableReady: true,
    })
    
    // 启动传输
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    err := transport.Start(ctx)
    require.NoError(t, err)
    defer transport.Stop(ctx)
    
    // 等待传输就绪
    err = transport.WaitReady()
    require.NoError(t, err)
    
    // 测试 HTTP 请求
    address := transport.GetAddress()
    resp, err := http.Get(fmt.Sprintf("http://localhost%s/health", address))
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

## 最佳实践

### 服务设计

1. **传输独立性** - 设计服务以适用于任一传输
2. **适当的错误处理** - 返回适当的 HTTP 状态码和 gRPC 状态码
3. **上下文传播** - 始终使用和传播上下文
4. **健康检查** - 实现有意义的健康检查
5. **元数据管理** - 提供准确的服务元数据

### 性能优化

1. **连接池** - 为数据库和外部服务使用连接池
2. **资源管理** - 适当关闭资源并处理清理
3. **超时配置** - 为操作设置适当的超时
4. **消息大小限制** - 为 gRPC 配置适当的消息大小限制
5. **Keepalive 设置** - 为您的网络条件调整 keepalive 参数

这个传输层指南提供了 HTTP 和 gRPC 传输实现、服务注册模式、错误处理、测试策略和生产部署考虑的全面覆盖。
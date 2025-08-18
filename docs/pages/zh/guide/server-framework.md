# 服务器框架

本指南涵盖 Swit 中的核心服务器框架，它为微服务开发提供了全面的、生产就绪的基础，具有统一的生命周期管理、传输协调和服务注册功能。

## 概述

Swit 服务器框架 (`pkg/server`) 实现了完整的服务器架构，处理：

- **传输管理** - 统一的 HTTP 和 gRPC 传输协调
- **服务注册** - 接口驱动的服务注册模式
- **依赖注入** - 基于工厂的依赖管理
- **性能监控** - 内置指标和分析
- **生命周期管理** - 分阶段启动和关闭协调

## 核心架构

### BusinessServerCore 接口

主要服务器接口提供一致的生命周期管理：

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

### BusinessServerImpl - 核心实现

主服务器实现协调所有框架组件：

```go
// 创建新的服务器实例
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
    return fmt.Errorf("创建服务器失败: %w", err)
}
```

## 服务注册系统

### BusinessServiceRegistrar 接口

服务实现此接口以向服务器注册：

```go
type BusinessServiceRegistrar interface {
    RegisterServices(registry BusinessServiceRegistry) error
}
```

### 实现示例

```go
type MyServiceRegistrar struct {
    userService *UserService
    authService *AuthService
}

func (r *MyServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
    // 注册 HTTP 处理程序
    if err := registry.RegisterBusinessHTTPHandler(r.userService); err != nil {
        return fmt.Errorf("注册用户 HTTP 处理程序失败: %w", err)
    }

    // 注册 gRPC 服务
    if err := registry.RegisterBusinessGRPCService(r.authService); err != nil {
        return fmt.Errorf("注册认证 gRPC 服务失败: %w", err)
    }

    // 注册健康检查
    healthCheck := &ServiceHealthCheck{
        userService: r.userService,
        authService: r.authService,
    }
    if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
        return fmt.Errorf("注册健康检查失败: %w", err)
    }

    return nil
}
```

## HTTP 服务实现

### BusinessHTTPHandler 接口

```go
type BusinessHTTPHandler interface {
    RegisterRoutes(router interface{}) error
    GetServiceName() string
}
```

### 实现示例

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

## gRPC 服务实现

### BusinessGRPCService 接口

```go
type BusinessGRPCService interface {
    RegisterGRPC(server interface{}) error
    GetServiceName() string
}
```

### 实现示例

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
    // 验证用户凭据
    user, err := s.userRepo.ValidateCredentials(req.Username, req.Password)
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "无效凭据")
    }
    
    // 生成令牌
    token, err := s.tokenManager.GenerateToken(user.ID)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "生成令牌失败")
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

## 生命周期管理

### 服务器启动过程

服务器遵循分阶段启动过程：

1. **配置验证** - 验证所有配置参数
2. **依赖初始化** - 初始化依赖容器
3. **传输初始化** - 启动 HTTP 和 gRPC 传输
4. **服务注册** - 向传输注册所有服务
5. **发现注册** - 向服务发现注册
6. **性能监控** - 启动性能监控

```go
func (s *BusinessServerImpl) Start(ctx context.Context) error {
    // 阶段 1：初始化依赖
    if err := s.initializeDependencies(ctx); err != nil {
        return fmt.Errorf("初始化依赖失败: %w", err)
    }
    
    // 阶段 2：初始化传输
    if err := s.initializeTransports(ctx); err != nil {
        return fmt.Errorf("初始化传输失败: %w", err)
    }
    
    // 阶段 3：注册服务
    if err := s.registerServices(); err != nil {
        return fmt.Errorf("注册服务失败: %w", err)
    }
    
    // 阶段 4：启动发现注册
    if err := s.registerWithDiscovery(ctx); err != nil {
        return fmt.Errorf("向发现注册失败: %w", err)
    }
    
    // 阶段 5：启动性能监控
    s.startPerformanceMonitoring(ctx)
    
    return nil
}
```

### 优雅关闭

服务器支持带有适当资源清理的优雅关闭：

```go
func (s *BusinessServerImpl) Shutdown() error {
    ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
    defer cancel()
    
    // 从服务发现注销
    if err := s.deregisterFromDiscovery(ctx); err != nil {
        s.logger.Error("从发现注销失败", zap.Error(err))
    }
    
    // 优雅停止传输
    if err := s.transportCoordinator.Stop(s.config.ShutdownTimeout); err != nil {
        s.logger.Error("传输关闭错误", zap.Error(err))
    }
    
    // 关闭依赖
    if s.dependencies != nil {
        if err := s.dependencies.Close(); err != nil {
            s.logger.Error("关闭依赖失败", zap.Error(err))
        }
    }
    
    return nil
}
```

## 性能监控

### 内置性能指标

服务器框架包括全面的性能监控：

```go
type PerformanceMetrics struct {
    StartupTime    time.Duration // 服务器启动时间
    ShutdownTime   time.Duration // 服务器关闭时间
    Uptime         time.Duration // 当前运行时间
    MemoryUsage    uint64        // 当前内存使用
    GoroutineCount int           // 活跃协程
    ServiceCount   int           // 注册服务
    TransportCount int           // 活跃传输
    RequestCount   int64         // 请求计数器
    ErrorCount     int64         // 错误计数器
}
```

### 性能钩子

添加自定义性能监控：

```go
func CustomPerformanceHook(event string, metrics *server.PerformanceMetrics) {
    // 发送指标到外部监控系统
    switch event {
    case "startup_complete":
        prometheus.SetGauge("server_startup_time_seconds", metrics.StartupTime.Seconds())
    case "request_processed":
        prometheus.IncrementCounter("server_requests_total")
    case "memory_threshold_exceeded":
        alert.Send("检测到高内存使用", metrics.MemoryUsage)
    }
}

// 注册钩子
srv := server.NewBusinessServerCore(config, registrar, deps)
monitor := srv.GetPerformanceMonitor()
monitor.AddHook(CustomPerformanceHook)
```

## 最佳实践

### 服务设计

1. **单一职责** - 每个服务应该有清晰的单一职责
2. **接口隔离** - 使用特定接口而不是大型单体接口
3. **依赖注入** - 使用依赖注入提高可测试性
4. **错误处理** - 实现带有上下文的全面错误处理
5. **日志记录** - 在整个服务中使用结构化日志

### 性能优化

1. **资源管理** - 适当管理数据库连接、文件句柄等
2. **上下文传播** - 始终传播上下文以支持取消和超时
3. **指标收集** - 监控关键性能指标
4. **内存管理** - 通过适当的资源清理避免内存泄漏
5. **并发** - 适当使用协程和通道

### 生产就绪

1. **健康检查** - 为所有依赖实现有意义的健康检查
2. **优雅关闭** - 适当处理关闭信号
3. **配置验证** - 在启动时验证配置
4. **错误恢复** - 实现适当的错误恢复机制
5. **监控集成** - 与监控和警报系统集成

这个服务器框架指南提供了核心服务器功能、服务注册模式、生命周期管理和使用 Swit 框架构建健壮微服务的最佳实践的全面覆盖。
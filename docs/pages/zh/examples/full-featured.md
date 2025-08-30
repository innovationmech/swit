# 全功能服务示例

此示例演示了如何构建一个同时支持 HTTP 和 gRPC 传输的综合服务，使用 Swit 框架。它展示了完整的框架功能，包括双传输支持、统一服务注册和生产就绪配置。

## 概述

全功能服务示例（`examples/full-featured-service/`）提供：
- **双传输支持** - 在一个应用程序中同时支持 HTTP REST API 和 gRPC 服务
- **统一服务接口** - 单一服务实现为两种协议提供服务
- **完整框架集成** - 演示所有主要框架功能
- **生产配置** - 高级服务器设置和中间件
- **综合示例** - 多种端点类型和模式

## 主要特性

### 多传输架构
- 在单一服务中实现 HTTP 和 gRPC 处理程序
- 为不同传输协议提供服务的统一业务逻辑
- 针对最佳客户端体验的协议特定适配器
- 共享的依赖注入和配置

### 高级配置
- gRPC 连接的 Keepalive 参数
- HTTP 超时和中间件配置
- 服务发现集成就绪
- 基于环境的配置管理

### 协议支持
- **HTTP REST API** - 基于 JSON 的端点，具有 RESTful 模式
- **gRPC 服务** - 基于 Protocol Buffer 的 RPC，具有类型安全
- **健康服务** - 两种传输的标准健康检查
- **指标端点** - 服务指标和监控数据

## 代码结构

### 多传输服务实现

```go
// FullFeaturedService 实现 HTTP 和 gRPC 传输
type FullFeaturedService struct {
    name string
}

func (s *FullFeaturedService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // 注册 HTTP 处理程序
    httpHandler := &FullFeaturedHTTPHandler{serviceName: s.name}
    if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
        return fmt.Errorf("注册 HTTP 处理程序失败: %w", err)
    }

    // 注册 gRPC 服务
    grpcService := &FullFeaturedGRPCService{serviceName: s.name}
    if err := registry.RegisterBusinessGRPCService(grpcService); err != nil {
        return fmt.Errorf("注册 gRPC 服务失败: %w", err)
    }

    // 注册健康检查（传输间共享）
    healthCheck := &FullFeaturedHealthCheck{serviceName: s.name}
    if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
        return fmt.Errorf("注册健康检查失败: %w", err)
    }

    return nil
}
```

### HTTP 处理程序实现

```go
type FullFeaturedHTTPHandler struct {
    serviceName string
}

func (h *FullFeaturedHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(gin.IRouter)

    api := ginRouter.Group("/api/v1")
    {
        // 问候端点（gRPC 方法的 HTTP 版本）
        api.POST("/greet", h.handleGreet)
        
        // 其他仅 HTTP 端点
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
            "error":   "无效请求",
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

### gRPC 服务实现

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
    // 验证请求
    if req.GetName() == "" {
        return nil, status.Error(codes.InvalidArgument, "姓名不能为空")
    }

    // 创建响应（共享业务逻辑）
    response := &interaction.SayHelloResponse{
        Message: fmt.Sprintf("Hello, %s!", req.GetName()),
    }

    return response, nil
}
```

### 高级配置

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

## 运行示例

### 前置条件
- 已安装 Go 1.23.12+
- 已安装 gRPC 工具（用于 gRPC 测试）
- 框架依赖可用

### 快速开始

1. **导航到示例目录：**
   ```bash
   cd examples/full-featured-service
   ```

2. **运行服务：**
   ```bash
   go run main.go
   ```

3. **测试 HTTP 端点：**
   ```bash
   # 问候端点
   curl -X POST "http://localhost:8080/api/v1/greet" \
        -H "Content-Type: application/json" \
        -d '{"name": "Alice"}'
   
   # 状态端点
   curl "http://localhost:8080/api/v1/status"
   
   # 指标端点
   curl "http://localhost:8080/api/v1/metrics"
   ```

4. **测试 gRPC 端点：**
   ```bash
   # SayHello 方法
   grpcurl -plaintext -d '{"name": "Bob"}' \
           localhost:9090 \
           swit.interaction.v1.GreeterService/SayHello
   
   # 健康检查
   grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
   ```

### 环境配置

```bash
# 配置两个传输端口
export HTTP_PORT=8888
export GRPC_PORT=9999

# 启用服务发现
export DISCOVERY_ENABLED=true
export CONSUL_ADDRESS=localhost:8500

# 设置环境
export ENVIRONMENT=production

# 使用自定义配置运行
go run main.go
```

### 预期响应

**HTTP 问候：**
```json
{
  "message": "Hello, Alice!",
  "service": "full-featured-service",
  "timestamp": "2025-01-20T10:30:00Z",
  "protocol": "HTTP"
}
```

**HTTP 状态：**
```json
{
  "status": "healthy",
  "service": "full-featured-service",
  "timestamp": "2025-01-20T10:30:00Z",
  "uptime": "running",
  "protocols": ["HTTP", "gRPC"]
}
```

**gRPC 响应：**
```json
{
  "message": "Hello, Bob!"
}
```

## 开发模式

### 共享业务逻辑

```go
// BusinessLogic 封装共享功能
type BusinessLogic struct {
    serviceName string
    logger      *zap.Logger
}

func (bl *BusinessLogic) ProcessGreeting(name string) (string, error) {
    if name == "" {
        return "", fmt.Errorf("姓名不能为空")
    }
    
    // 共享业务逻辑
    greeting := fmt.Sprintf("Hello, %s!", name)
    bl.logger.Info("生成问候", zap.String("name", name))
    
    return greeting, nil
}

// 在 HTTP 处理程序中使用
func (h *FullFeaturedHTTPHandler) handleGreet(c *gin.Context) {
    var request struct {
        Name string `json:"name" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
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

// 在 gRPC 服务中使用
func (s *FullFeaturedGRPCService) SayHello(ctx context.Context, req *interaction.SayHelloRequest) (*interaction.SayHelloResponse, error) {
    message, err := s.businessLogic.ProcessGreeting(req.GetName())
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, err.Error())
    }
    
    return &interaction.SayHelloResponse{Message: message}, nil
}
```

### 协议特定适配器

```go
// HTTPAdapter 在 HTTP 和业务逻辑之间转换
type HTTPAdapter struct {
    businessLogic *BusinessLogic
}

func (a *HTTPAdapter) HandleRequest(c *gin.Context, processor func(string) (interface{}, error)) {
    var request struct {
        Name string `json:"name" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "无效请求",
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

// GRPCAdapter 处理 gRPC 特定问题
type GRPCAdapter struct {
    businessLogic *BusinessLogic
}

func (a *GRPCAdapter) ProcessRequest(ctx context.Context, name string) (string, error) {
    // 检查上下文取消
    select {
    case <-ctx.Done():
        return "", status.Error(codes.Canceled, "请求被取消")
    default:
    }
    
    // 使用业务逻辑处理
    return a.businessLogic.ProcessGreeting(name)
}
```

### 高级健康检查

```go
type FullFeaturedHealthCheck struct {
    serviceName string
    database    *sql.DB
    redis       *redis.Client
}

func (h *FullFeaturedHealthCheck) Check(ctx context.Context) error {
    // 检查数据库连接
    if err := h.database.PingContext(ctx); err != nil {
        return fmt.Errorf("数据库健康检查失败: %w", err)
    }
    
    // 检查 Redis 连接
    if err := h.redis.Ping(ctx).Err(); err != nil {
        return fmt.Errorf("redis 健康检查失败: %w", err)
    }
    
    // 检查外部依赖
    if err := h.checkExternalService(ctx); err != nil {
        return fmt.Errorf("外部服务健康检查失败: %w", err)
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
        return fmt.Errorf("外部服务返回状态 %d", resp.StatusCode)
    }
    
    return nil
}
```

## 生产考虑

### 监控集成

```go
// 添加指标收集
func (h *FullFeaturedHTTPHandler) handleMetrics(c *gin.Context) {
    metrics := map[string]interface{}{
        "service":           h.serviceName,
        "requests_total":    100, // 来自指标收集器
        "errors_total":      5,   // 来自指标收集器
        "uptime_seconds":    3600, // 来自运行时间跟踪器
        "memory_usage_mb":   128,  // 来自运行时统计
        "goroutines":        50,   // runtime.NumGoroutine()
        "protocols":         []string{"HTTP", "gRPC"},
        "timestamp":         time.Now().UTC(),
    }
    
    c.JSON(http.StatusOK, metrics)
}
```

### 服务发现集成

```go
// 配置生产服务发现
discovery: server.DiscoveryConfig{
    Enabled:         true,
    Address:         "consul.internal:8500",
    ServiceName:     "full-featured-service",
    Tags:            []string{"http", "grpc", "api", "v1", "production"},
    FailureMode:     server.DiscoveryFailureModeFailFast,
    HealthCheckRequired: true,
}
```

### 安全配置

```go
// 添加安全中间件
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

## 测试多传输服务

### 集成测试

```go
func TestFullFeaturedService(t *testing.T) {
    // 使用测试配置启动服务
    config := createTestConfig()
    service := NewFullFeaturedService("test-service")
    srv, err := server.NewBusinessServerCore(config, service, nil)
    require.NoError(t, err)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    err = srv.Start(ctx)
    require.NoError(t, err)
    defer srv.Shutdown()
    
    // 测试 HTTP 端点
    httpAddr := srv.GetHTTPAddress()
    resp, err := http.Post(
        fmt.Sprintf("http://localhost%s/api/v1/greet", httpAddr),
        "application/json",
        strings.NewReader(`{"name": "TestUser"}`),
    )
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    // 测试 gRPC 端点
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

## 演示的最佳实践

1. **传输抽象** - 传输和业务逻辑之间的清洁分离
2. **统一配置** - 多个传输的单一配置
3. **共享组件** - 可重用的健康检查和依赖注入
4. **协议优化** - 传输特定的优化和模式
5. **生产就绪性** - 高级配置和监控集成

## 下一步

在理解这个全功能示例后：

1. **数据库集成** - 添加数据库连接和存储库模式
2. **身份验证** - 为两种传输实现基于 JWT 的身份验证
3. **高级监控** - 与 Prometheus、Jaeger 或类似工具集成
4. **部署** - 使用 Docker 容器化并使用 Kubernetes 部署

此示例演示了 Swit 框架构建能够高效服务多种传输协议的生产就绪微服务的完整功能。
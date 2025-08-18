# 测试

本综合指南涵盖 Swit 框架的测试策略，包括单元测试、集成测试、性能测试以及使用该框架构建的微服务测试最佳实践。

## 概述

Swit 框架通过以下方式提供广泛的测试支持：
- **测试模式配置** - 测试环境的特殊配置
- **动态端口分配** - 并发测试的自动端口分配
- **模拟依赖** - 对依赖模拟的内置支持
- **传输测试** - 用于测试 HTTP 和 gRPC 传输的实用工具
- **集成测试模式** - 全面集成测试的模式

## 测试配置

### 测试模式配置

```go
// 创建测试配置
config := server.NewServerConfig()
config.ServiceName = "test-service"

// 为两个传输启用测试模式
config.HTTP.TestMode = true
config.HTTP.Port = "0" // 动态端口分配
config.HTTP.EnableReady = true // 启用准备通道用于同步

config.GRPC.TestMode = true 
config.GRPC.Port = "0" // 动态端口分配

// 测试时禁用服务发现
config.Discovery.Enabled = false

// 减少超时以加快测试
config.ShutdownTimeout = 5 * time.Second
```

### 环境特定测试配置

```yaml
# config/test.yaml
service_name: "my-service-test"
shutdown_timeout: "5s"

http:
  enabled: true
  port: "0"  # 动态分配
  test_mode: true
  enable_ready: true
  read_timeout: "5s"
  write_timeout: "5s"
  middleware:
    enable_cors: true
    enable_logging: false  # 减少测试噪音

grpc:
  enabled: true
  port: "0"  # 动态分配
  test_mode: true
  enable_reflection: true
  enable_health_service: true

discovery:
  enabled: false  # 单元测试时禁用
  
middleware:
  enable_logging: false  # 减少测试输出
```

## 单元测试

### 测试服务注册

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
    // 创建模拟注册表
    registry := &MockBusinessServiceRegistry{
        httpHandlers:   make(map[string]server.BusinessHTTPHandler),
        grpcServices:   make(map[string]server.BusinessGRPCService), 
        healthChecks:   make(map[string]server.BusinessHealthCheck),
    }
    
    // 创建测试服务
    userService := &TestUserService{}
    authService := &TestAuthService{}
    
    registrar := &TestServiceRegistrar{
        userService: userService,
        authService: authService,
    }
    
    // 测试服务注册
    err := registrar.RegisterServices(registry)
    require.NoError(t, err)
    
    // 验证 HTTP 处理程序注册
    assert.Contains(t, registry.httpHandlers, "user-service")
    assert.Equal(t, userService, registry.httpHandlers["user-service"])
    
    // 验证 gRPC 服务注册
    assert.Contains(t, registry.grpcServices, "auth-service")
    assert.Equal(t, authService, registry.grpcServices["auth-service"])
    
    // 验证健康检查注册
    assert.Len(t, registry.healthChecks, 1)
}
```

### 测试单独的服务

```go
func TestUserService(t *testing.T) {
    // 创建测试依赖
    mockDB := &MockDatabase{
        users: map[string]*User{
            "user1": {ID: "user1", Name: "测试用户", Email: "test@example.com"},
        },
    }
    
    mockCache := &MockRedisClient{
        data: make(map[string]string),
    }
    
    // 使用模拟依赖创建服务
    userService := &UserService{
        repository: &UserRepository{DB: mockDB},
        cache:      mockCache,
        logger:     zap.NewNop(),
    }
    
    // 测试 GetUser 方法
    ctx := context.Background()
    user, err := userService.GetUser(ctx, "user1")
    
    require.NoError(t, err)
    assert.Equal(t, "user1", user.ID)
    assert.Equal(t, "测试用户", user.Name)
    assert.Equal(t, "test@example.com", user.Email)
    
    // 验证缓存已更新
    cached, exists := mockCache.data["user:user1"] 
    assert.True(t, exists)
    assert.Contains(t, cached, "测试用户")
    
    // 测试不存在的用户
    user, err = userService.GetUser(ctx, "nonexistent")
    assert.Error(t, err)
    assert.Nil(t, user)
}
```

### 使用依赖注入进行测试

```go
func TestServiceWithDependencyInjection(t *testing.T) {
    // 创建测试容器
    container := server.NewSimpleBusinessDependencyContainer()
    
    // 注册测试依赖
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
    
    // 初始化依赖
    ctx := context.Background()
    err := container.Initialize(ctx)
    require.NoError(t, err)
    defer container.Close()
    
    // 使用依赖注入创建服务
    repo, err := container.GetService("user-repository")
    require.NoError(t, err)
    
    userRepo := repo.(*UserRepository)
    
    // 测试存储库操作
    user := &User{ID: "test1", Name: "测试用户"}
    err = userRepo.Create(ctx, user)
    require.NoError(t, err)
    
    retrieved, err := userRepo.GetByID(ctx, "test1")
    require.NoError(t, err)
    assert.Equal(t, user.Name, retrieved.Name)
}
```

## 集成测试

### HTTP 传输集成测试

```go
func TestHTTPTransportIntegration(t *testing.T) {
    // 创建测试配置
    config := server.NewServerConfig()
    config.HTTP.Port = "0" // 动态端口
    config.HTTP.TestMode = true
    config.HTTP.EnableReady = true
    config.GRPC.Enabled = false // 仅测试 HTTP
    config.Discovery.Enabled = false
    
    // 创建测试服务注册器
    userService := &TestUserService{
        users: map[string]*User{
            "user1": {ID: "user1", Name: "测试用户"},
        },
    }
    
    registrar := &TestServiceRegistrar{userService: userService}
    
    // 创建并启动服务器
    srv, err := server.NewBusinessServerCore(config, registrar, nil)
    require.NoError(t, err)
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    err = srv.Start(ctx)
    require.NoError(t, err)
    defer srv.Shutdown()
    
    // 等待服务器就绪
    if httpTransport, ok := srv.GetTransports()[0].(*transport.HTTPTransport); ok {
        err = httpTransport.WaitReady()
        require.NoError(t, err)
    }
    
    // 获取服务器地址
    httpAddr := srv.GetHTTPAddress()
    baseURL := fmt.Sprintf("http://localhost%s", httpAddr)
    
    // 测试健康端点
    resp, err := http.Get(baseURL + "/health")
    require.NoError(t, err)
    defer resp.Body.Close()
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    // 测试 API 端点
    resp, err = http.Get(baseURL + "/api/v1/users")
    require.NoError(t, err)
    defer resp.Body.Close()
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    var users []User
    err = json.NewDecoder(resp.Body).Decode(&users)
    require.NoError(t, err)
    assert.Len(t, users, 1)
    assert.Equal(t, "user1", users[0].ID)
}
```

### gRPC 传输集成测试

```go
func TestGRPCTransportIntegration(t *testing.T) {
    // 创建测试配置
    config := server.NewServerConfig()
    config.HTTP.Enabled = false // 仅测试 gRPC
    config.GRPC.Port = "0" // 动态端口
    config.GRPC.TestMode = true
    config.GRPC.EnableReflection = true
    config.GRPC.EnableHealthService = true
    config.Discovery.Enabled = false
    
    // 创建测试认证服务
    authService := &TestAuthService{
        users: map[string]string{
            "testuser": "testpass",
        },
        tokens: make(map[string]string),
    }
    
    registrar := &TestServiceRegistrar{authService: authService}
    
    // 创建并启动服务器
    srv, err := server.NewBusinessServerCore(config, registrar, nil)
    require.NoError(t, err)
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    err = srv.Start(ctx)
    require.NoError(t, err)
    defer srv.Shutdown()
    
    // 获取 gRPC 地址
    grpcAddr := srv.GetGRPCAddress()
    
    // 创建 gRPC 客户端连接
    conn, err := grpc.DialContext(ctx, grpcAddr, 
        grpc.WithInsecure(),
        grpc.WithBlock(),
    )
    require.NoError(t, err)
    defer conn.Close()
    
    // 测试健康服务
    healthClient := healthpb.NewHealthClient(conn)
    healthResp, err := healthClient.Check(ctx, &healthpb.HealthCheckRequest{})
    require.NoError(t, err)
    assert.Equal(t, healthpb.HealthCheckResponse_SERVING, healthResp.Status)
    
    // 测试认证服务
    authClient := authpb.NewAuthServiceClient(conn)
    
    // 测试登录
    loginResp, err := authClient.Login(ctx, &authpb.LoginRequest{
        Username: "testuser",
        Password: "testpass",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, loginResp.Token)
    assert.Equal(t, "testuser", loginResp.User.Username)
    
    // 测试令牌验证
    validateResp, err := authClient.ValidateToken(ctx, &authpb.ValidateTokenRequest{
        Token: loginResp.Token,
    })
    require.NoError(t, err)
    assert.True(t, validateResp.Valid)
    assert.Equal(t, "testuser", validateResp.UserId)
    
    // 测试无效登录
    _, err = authClient.Login(ctx, &authpb.LoginRequest{
        Username: "invalid",
        Password: "invalid",
    })
    require.Error(t, err)
    assert.Contains(t, err.Error(), "无效凭据")
}
```

## 性能测试

### 负载测试设置

```go
func TestServerLoadTesting(t *testing.T) {
    if testing.Short() {
        t.Skip("在短模式下跳过负载测试")
    }
    
    // 启动测试服务器
    srv, httpAddr := startTestServer(t)
    defer srv.Shutdown()
    
    // 配置负载测试参数
    const (
        numClients    = 50
        requestsPerClient = 100
        totalRequests = numClients * requestsPerClient
    )
    
    // 创建带连接池的 HTTP 客户端
    client := &http.Client{
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     30 * time.Second,
        },
        Timeout: 10 * time.Second,
    }
    
    // 跟踪指标
    var (
        successCount int64
        errorCount   int64
        totalLatency int64
    )
    
    // 创建工作协程
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, numClients)
    
    startTime := time.Now()
    
    for i := 0; i < totalRequests; i++ {
        wg.Add(1)
        semaphore <- struct{}{} // 限制并发
        
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
    
    // 计算指标
    avgLatency := time.Duration(totalLatency / totalRequests)
    requestsPerSecond := float64(totalRequests) / totalDuration.Seconds()
    errorRate := float64(errorCount) / float64(totalRequests) * 100
    
    // 记录结果
    t.Logf("负载测试结果:")
    t.Logf("总请求数: %d", totalRequests)
    t.Logf("成功请求数: %d", successCount)
    t.Logf("失败请求数: %d", errorCount)
    t.Logf("错误率: %.2f%%", errorRate)
    t.Logf("总耗时: %v", totalDuration)
    t.Logf("每秒请求数: %.2f", requestsPerSecond)
    t.Logf("平均延迟: %v", avgLatency)
    
    // 断言性能阈值
    assert.Less(t, errorRate, 1.0, "错误率应小于 1%")
    assert.Greater(t, requestsPerSecond, 100.0, "应处理至少每秒 100 个请求")
    assert.Less(t, avgLatency, 100*time.Millisecond, "平均延迟应小于 100ms")
}
```

### 内存和资源测试

```go
func TestServerMemoryUsage(t *testing.T) {
    // 启动测试服务器
    srv, _ := startTestServer(t)
    defer srv.Shutdown()
    
    // 获取基线内存
    runtime.GC()
    var baseline runtime.MemStats
    runtime.ReadMemStats(&baseline)
    
    // 模拟工作负载
    const numOperations = 1000
    for i := 0; i < numOperations; i++ {
        // 模拟内存分配
        data := make([]byte, 1024)
        _ = data
        
        // 强制定期垃圾收集
        if i%100 == 0 {
            runtime.GC()
        }
    }
    
    // 测量最终内存
    runtime.GC()
    var final runtime.MemStats
    runtime.ReadMemStats(&final)
    
    // 计算内存增长
    memoryGrowth := final.Alloc - baseline.Alloc
    
    t.Logf("内存使用:")
    t.Logf("基线分配: %d 字节", baseline.Alloc)
    t.Logf("最终分配: %d 字节", final.Alloc)
    t.Logf("内存增长: %d 字节", memoryGrowth)
    t.Logf("GC 周期: %d", final.NumGC-baseline.NumGC)
    
    // 断言内存使用合理
    maxAllowedGrowth := uint64(10 * 1024 * 1024) // 10MB
    assert.Less(t, memoryGrowth, maxAllowedGrowth,
        "内存增长应小于 %d 字节", maxAllowedGrowth)
}
```

## 测试工具和助手

### 测试服务器工厂

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
            "user1": {ID: "user1", Name: "测试用户"},
            "user2": {ID: "user2", Name: "另一个用户"},
        },
    }
    
    registrar := &TestServiceRegistrar{userService: userService}
    
    srv, err := server.NewBusinessServerCore(config, registrar, nil)
    require.NoError(t, err)
    
    ctx := context.Background()
    err = srv.Start(ctx)
    require.NoError(t, err)
    
    // 等待服务器就绪
    httpAddr := srv.GetHTTPAddress()
    
    return srv, httpAddr
}
```

### 测试数据工厂

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
        Name:  fmt.Sprintf("测试用户 %d", f.userCounter),
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

### 测试断言助手

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

## 测试最佳实践

### 测试组织

1. **测试结构** - 按组件组织测试（单元、集成、性能）
2. **测试命名** - 使用说明测试内容的描述性测试名称
3. **测试数据** - 使用工厂或构建器创建测试数据
4. **测试清理** - 测试后始终清理资源
5. **测试隔离** - 确保测试不相互依赖

### 模拟策略

1. **接口模拟** - 通过接口模拟依赖
2. **依赖注入** - 使用 DI 注入模拟
3. **测试替身** - 使用适当的测试替身（存根、模拟、伪造）
4. **模拟验证** - 适当时验证模拟交互
5. **模拟清理** - 在测试之间重置模拟

### 性能测试

1. **基线建立** - 建立性能基线
2. **负载模式** - 使用现实的负载模式进行测试
3. **资源监控** - 监控内存、CPU 和协程
4. **阈值设置** - 设置现实的性能阈值
5. **回归检测** - 在 CI/CD 中包含性能测试

这个全面的测试指南涵盖了测试 Swit 框架应用程序的所有方面，从单元测试到性能测试，提供了构建健壮、经过充分测试的微服务的实际示例和最佳实践。
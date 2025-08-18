# 服务发现

本指南涵盖 Swit 中的服务发现系统，它提供基于 Consul 的服务注册和发现，具有可配置的失败模式、健康检查集成和多端点支持。

## 概述

Swit 框架包括通过 Consul 的集成服务发现，允许服务自动注册并发现网络中的其他服务。系统支持多种失败模式、健康检查集成和发现失败的优雅处理。

## 核心组件

### ServiceDiscoveryManager 接口

服务发现操作的主要接口：

```go
type ServiceDiscoveryManager interface {
    RegisterService(ctx context.Context, registration *ServiceRegistration) error
    DeregisterService(ctx context.Context, registration *ServiceRegistration) error
    RegisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error
    DeregisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error
    IsHealthy(ctx context.Context) bool
}
```

### ServiceRegistration 结构

定义服务注册参数：

```go
type ServiceRegistration struct {
    ID              string            // 唯一服务实例 ID
    Name            string            // 服务名称
    Tags            []string          // 服务标签
    Address         string            // 服务地址
    Port            int               // 服务端口
    Check           *HealthCheck      // 健康检查配置
    Meta            map[string]string // 附加元数据
}
```

### HealthCheck 配置

定义 Consul 的健康检查参数：

```go
type HealthCheck struct {
    HTTP                           string        // HTTP 健康检查 URL
    GRPC                           string        // gRPC 健康检查
    TCP                            string        // TCP 健康检查
    Interval                       time.Duration // 检查间隔
    Timeout                        time.Duration // 检查超时
    DeregisterCriticalServiceAfter time.Duration // 自动注销超时
}
```

## 基本配置

### 发现配置

```yaml
discovery:
  enabled: true
  address: "127.0.0.1:8500"  # Consul 地址
  service_name: "my-service"
  tags:
    - "v1"
    - "api"
    - "production"
  failure_mode: "graceful"  # "graceful", "fail_fast", "strict"
  health_check_required: false
  registration_timeout: "30s"
```

### 失败模式

**优雅模式（默认）：**
- 即使发现注册失败，服务器也继续启动
- 适用于开发环境
- 非关键的发现失败不会阻止服务运行

```yaml
discovery:
  failure_mode: "graceful"
```

**快速失败模式：**
- 如果发现注册失败，服务器启动失败
- 推荐用于发现至关重要的生产环境
- 确保服务在提供流量之前正确注册

```yaml
discovery:
  failure_mode: "fail_fast"
```

**严格模式：**
- 需要发现健康检查，对任何发现问题都快速失败
- 最高级别的可靠性
- 确保在继续之前注册和发现健康

```yaml
discovery:
  failure_mode: "strict"
  health_check_required: true
```

## 服务注册

### 自动注册

启用时，框架会自动注册服务：

```go
config := &server.ServerConfig{
    ServiceName: "user-service",
    HTTP: server.HTTPConfig{
        Port:    "8080",
        Enabled: true,
    },
    GRPC: server.GRPCConfig{
        Port:    "9080", 
        Enabled: true,
    },
    Discovery: server.DiscoveryConfig{
        Enabled:     true,
        Address:     "127.0.0.1:8500",
        ServiceName: "user-service",
        Tags:        []string{"v1", "api"},
        FailureMode: server.DiscoveryFailureModeFailFast,
    },
}

srv, err := server.NewBusinessServerCore(config, registrar, deps)
if err != nil {
    return fmt.Errorf("创建服务器失败: %w", err)
}

// 服务在启动期间自动注册
err = srv.Start(ctx)
```

### 手动注册

用于自定义注册场景：

```go
import "github.com/innovationmech/swit/pkg/discovery"

// 创建发现管理器
manager, err := discovery.NewConsulServiceDiscoveryManager("127.0.0.1:8500")
if err != nil {
    return fmt.Errorf("创建发现管理器失败: %w", err)
}

// 创建服务注册
registration := &discovery.ServiceRegistration{
    ID:      "user-service-1",
    Name:    "user-service",
    Address: "192.168.1.100",
    Port:    8080,
    Tags:    []string{"v1", "http", "api"},
    Check: &discovery.HealthCheck{
        HTTP:     "http://192.168.1.100:8080/health",
        Interval: 10 * time.Second,
        Timeout:  3 * time.Second,
        DeregisterCriticalServiceAfter: 30 * time.Second,
    },
    Meta: map[string]string{
        "version":     "1.0.0",
        "environment": "production",
        "region":      "us-west-1",
    },
}

// 注册服务
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err = manager.RegisterService(ctx, registration)
if err != nil {
    return fmt.Errorf("注册服务失败: %w", err)
}
```

### 多传输注册

注册 HTTP 和 gRPC 端点：

```go
// HTTP 服务注册
httpRegistration := &discovery.ServiceRegistration{
    ID:      "user-service-http-1",
    Name:    "user-service",
    Address: "192.168.1.100",
    Port:    8080,
    Tags:    []string{"v1", "http", "api"},
    Check: &discovery.HealthCheck{
        HTTP:     "http://192.168.1.100:8080/health",
        Interval: 10 * time.Second,
        Timeout:  3 * time.Second,
    },
}

// gRPC 服务注册
grpcRegistration := &discovery.ServiceRegistration{
    ID:      "user-service-grpc-1", 
    Name:    "user-service-grpc",
    Address: "192.168.1.100",
    Port:    9080,
    Tags:    []string{"v1", "grpc", "api"},
    Check: &discovery.HealthCheck{
        GRPC:     "192.168.1.100:9080/grpc.health.v1.Health/Check",
        Interval: 10 * time.Second,
        Timeout:  3 * time.Second,
    },
}

// 注册多个端点
registrations := []*discovery.ServiceRegistration{
    httpRegistration,
    grpcRegistration,
}

err = manager.RegisterMultipleEndpoints(ctx, registrations)
if err != nil {
    return fmt.Errorf("注册端点失败: %w", err)
}
```

## 健康检查配置

### HTTP 健康检查

```go
registration := &discovery.ServiceRegistration{
    ID:      "api-service-1",
    Name:    "api-service",
    Address: "10.0.1.100",
    Port:    8080,
    Check: &discovery.HealthCheck{
        HTTP:                           "http://10.0.1.100:8080/health",
        Interval:                       15 * time.Second,
        Timeout:                        5 * time.Second,
        DeregisterCriticalServiceAfter: 60 * time.Second,
    },
}
```

### gRPC 健康检查

```go
registration := &discovery.ServiceRegistration{
    ID:      "grpc-service-1",
    Name:    "grpc-service", 
    Address: "10.0.1.100",
    Port:    9080,
    Check: &discovery.HealthCheck{
        GRPC:                           "10.0.1.100:9080/grpc.health.v1.Health/Check",
        Interval:                       10 * time.Second,
        Timeout:                        3 * time.Second,
        DeregisterCriticalServiceAfter: 30 * time.Second,
    },
}
```

### 自定义健康检查实现

```go
type CustomHealthChecker struct {
    database *sql.DB
    redis    *redis.Client
    external *http.Client
}

func (c *CustomHealthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    health := struct {
        Status      string            `json:"status"`
        Checks      map[string]string `json:"checks"`
        Timestamp   time.Time         `json:"timestamp"`
        Version     string            `json:"version"`
    }{
        Status:    "healthy",
        Checks:    make(map[string]string),
        Timestamp: time.Now(),
        Version:   "1.0.0",
    }
    
    // 检查数据库
    if err := c.database.Ping(); err != nil {
        health.Status = "unhealthy"
        health.Checks["database"] = fmt.Sprintf("失败: %v", err)
    } else {
        health.Checks["database"] = "ok"
    }
    
    // 检查 Redis
    if err := c.redis.Ping(r.Context()).Err(); err != nil {
        health.Status = "unhealthy" 
        health.Checks["redis"] = fmt.Sprintf("失败: %v", err)
    } else {
        health.Checks["redis"] = "ok"
    }
    
    // 检查外部依赖
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()
    
    req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.external.com/health", nil)
    resp, err := c.external.Do(req)
    if err != nil || resp.StatusCode != 200 {
        health.Status = "unhealthy"
        health.Checks["external_api"] = "failed"
    } else {
        health.Checks["external_api"] = "ok"
        resp.Body.Close()
    }
    
    statusCode := 200
    if health.Status != "healthy" {
        statusCode = 503
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(health)
}

// 注册健康检查端点
http.Handle("/health", &CustomHealthChecker{
    database: db,
    redis:    redisClient,
    external: &http.Client{Timeout: 5 * time.Second},
})
```

## 服务发现客户端

### 发现服务

```go
import (
    "github.com/hashicorp/consul/api"
)

// 创建 Consul 客户端
consulConfig := api.DefaultConfig()
consulConfig.Address = "127.0.0.1:8500"
client, err := api.NewClient(consulConfig)
if err != nil {
    return fmt.Errorf("创建 consul 客户端失败: %w", err)
}

// 发现健康服务
services, _, err := client.Health().Service("user-service", "", true, nil)
if err != nil {
    return fmt.Errorf("发现服务失败: %w", err)
}

// 使用发现的服务
for _, service := range services {
    endpoint := fmt.Sprintf("%s:%d", service.Service.Address, service.Service.Port)
    fmt.Printf("发现服务: %s\n", endpoint)
    
    // 创建客户端连接
    conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
    if err != nil {
        continue
    }
    defer conn.Close()
    
    // 使用服务...
}
```

### 带负载均衡的服务发现

```go
type ServiceDiscoveryClient struct {
    consul   *api.Client
    services map[string][]*api.ServiceEntry
    mu       sync.RWMutex
}

func NewServiceDiscoveryClient(consulAddr string) (*ServiceDiscoveryClient, error) {
    config := api.DefaultConfig()
    config.Address = consulAddr
    
    client, err := api.NewClient(config)
    if err != nil {
        return nil, err
    }
    
    sdc := &ServiceDiscoveryClient{
        consul:   client,
        services: make(map[string][]*api.ServiceEntry),
    }
    
    // 启动定期服务刷新
    go sdc.refreshServices()
    
    return sdc, nil
}

func (sdc *ServiceDiscoveryClient) GetService(serviceName string) (*api.ServiceEntry, error) {
    sdc.mu.RLock()
    services, exists := sdc.services[serviceName]
    sdc.mu.RUnlock()
    
    if !exists || len(services) == 0 {
        return nil, fmt.Errorf("未找到服务 %s 的健康实例", serviceName)
    }
    
    // 简单轮询负载均衡
    index := rand.Intn(len(services))
    return services[index], nil
}

func (sdc *ServiceDiscoveryClient) refreshServices() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        // 发现所有服务
        services := []string{"user-service", "auth-service", "payment-service"}
        
        for _, serviceName := range services {
            healthyServices, _, err := sdc.consul.Health().Service(serviceName, "", true, nil)
            if err != nil {
                log.Printf("刷新服务 %s 失败: %v", serviceName, err)
                continue
            }
            
            sdc.mu.Lock()
            sdc.services[serviceName] = healthyServices
            sdc.mu.Unlock()
        }
    }
}
```

## 测试服务发现

### 单元测试发现管理器

```go
func TestServiceRegistration(t *testing.T) {
    // 使用 testcontainers 创建测试 Consul 容器
    ctx := context.Background()
    consulContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "consul:1.15",
            ExposedPorts: []string{"8500/tcp"},
            Cmd:          []string{"consul", "agent", "-dev", "-client", "0.0.0.0"},
        },
        Started: true,
    })
    require.NoError(t, err)
    defer consulContainer.Terminate(ctx)
    
    // 获取 Consul 地址
    consulAddr, err := consulContainer.MappedPort(ctx, "8500")
    require.NoError(t, err)
    
    // 创建发现管理器
    manager, err := discovery.NewConsulServiceDiscoveryManager(fmt.Sprintf("127.0.0.1:%s", consulAddr.Port()))
    require.NoError(t, err)
    
    // 测试服务注册
    registration := &discovery.ServiceRegistration{
        ID:      "test-service-1",
        Name:    "test-service",
        Address: "127.0.0.1",
        Port:    8080,
        Tags:    []string{"test", "v1"},
    }
    
    err = manager.RegisterService(ctx, registration)
    assert.NoError(t, err)
    
    // 验证注册
    consulClient, _ := api.NewClient(&api.Config{Address: fmt.Sprintf("127.0.0.1:%s", consulAddr.Port())})
    services, _, err := consulClient.Catalog().Service("test-service", "", nil)
    require.NoError(t, err)
    assert.Len(t, services, 1)
    assert.Equal(t, "test-service-1", services[0].ServiceID)
    
    // 测试注销
    err = manager.DeregisterService(ctx, registration)
    assert.NoError(t, err)
    
    // 验证注销
    services, _, err = consulClient.Catalog().Service("test-service", "", nil)
    require.NoError(t, err)
    assert.Len(t, services, 0)
}
```

## 最佳实践

### 服务注册

1. **唯一服务 ID** - 使用唯一的服务实例 ID 以避免冲突
2. **有意义的标签** - 使用描述性标签进行服务过滤和发现
3. **健康检查配置** - 配置适当的健康检查间隔和超时
4. **优雅注销** - 在关闭期间始终注销服务
5. **元数据使用** - 使用服务元数据提供附加服务信息

### 健康检查

1. **轻量级检查** - 保持健康检查快速和轻量级
2. **依赖检查** - 在健康检查中检查关键依赖
3. **超时配置** - 为健康检查设置合理的超时
4. **自动注销** - 为失败的服务配置自动注销
5. **状态端点** - 在健康端点提供详细的状态信息

这个服务发现指南涵盖了集成的基于 Consul 的服务发现系统的所有方面，从基本配置到高级模式、测试策略和生产部署考虑。
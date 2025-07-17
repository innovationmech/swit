# SWIT 新子项目架构指导方案

## 概述

本文档基于 `switauth` 和 `switserve` 项目的架构分析，为新增子项目提供标准化的架构指导方案。遵循本指南可以确保新项目与现有项目保持架构一致性，提高代码复用性和可维护性。

## 架构设计原则

### 1. 分层架构模式

新项目必须遵循以下分层架构：

```
┌─────────────────────────────────────────┐
│                Server                   │
├─────────────────────────────────────────┤
│           Transport Layer               │
│  ┌─────────────────┬─────────────────┐  │
│  │  HTTPTransport  │  GRPCTransport  │  │
│  └─────────────────┴─────────────────┘  │
├─────────────────────────────────────────┤
│            Service Layer                │
│  ┌─────────────────────────────────────┐ │
│  │        ServiceRegistry              │ │
│  │  ┌───────────┬───────────────────┐  │ │
│  │  │ Service A │ Service B │ ... │  │ │
│  │  └───────────┴───────────────────┘  │ │
│  └─────────────────────────────────────┘ │
├─────────────────────────────────────────┤
│           Handler Layer                 │
│  ┌─────────────────┬─────────────────┐  │
│  │   HTTP Handler  │   gRPC Handler  │  │
│  └─────────────────┴─────────────────┘  │
└─────────────────────────────────────────┘
```

### 2. 核心设计原则

- **单一职责**: 每个组件只负责一个明确的功能
- **依赖注入**: 通过构造函数注入依赖，便于测试
- **接口隔离**: 定义清晰的接口边界
- **配置外部化**: 所有配置通过配置文件管理
- **优雅关闭**: 支持优雅的服务关闭
- **并发安全**: 所有共享资源必须线程安全

## 项目结构模板

### 目录结构

```
internal/new-service/
├── cmd/
│   └── root.go              # CLI 命令定义
├── config/
│   ├── config.go            # 配置结构定义
│   └── config_test.go       # 配置测试
├── db/
│   ├── db.go               # 数据库连接管理
│   └── migrations/         # 数据库迁移文件
├── handler/
│   └── http/
│       ├── service1/
│       │   ├── service1.go      # HTTP 处理器
│       │   └── service1_test.go # 处理器测试
│       └── health/
│           ├── health.go        # 健康检查处理器
│           └── health_test.go   # 健康检查测试
├── model/
│   ├── entity.go           # 数据模型定义
│   └── dto.go              # 数据传输对象
├── repository/
│   ├── interface.go        # 仓储接口定义
│   ├── impl.go             # 仓储实现
│   └── impl_test.go        # 仓储测试
├── service/
│   ├── service1/
│   │   ├── service_impl.go      # 业务逻辑实现
│   │   ├── service_grpc.go      # gRPC 服务实现
│   │   ├── service_registrar.go # 服务注册器
│   │   └── service_test.go      # 服务测试
│   └── health/
│       ├── health_impl.go       # 健康检查实现
│       ├── health_registrar.go  # 健康检查注册器
│       └── health_test.go       # 健康检查测试
├── transport/
│   ├── transport.go        # 传输层接口
│   ├── http.go             # HTTP 传输实现
│   ├── grpc.go             # gRPC 传输实现
│   ├── manager.go          # 传输管理器
│   ├── registrar.go        # 服务注册器接口
│   └── registry.go         # 服务注册表
├── server.go               # 服务器主文件
└── server_test.go          # 服务器测试
```

### 根目录文件

```
cmd/new-service/
├── new-service.go          # 主程序入口
└── new-service_test.go     # 主程序测试
```

## 核心组件实现指南

### 1. ServiceRegistrar 接口实现

每个服务必须实现 `ServiceRegistrar` 接口：

```go
type ServiceRegistrar interface {
    RegisterGRPC(server *grpc.Server) error
    RegisterHTTP(router *gin.Engine) error
    GetName() string
}
```

**实现模板：**

```go
// internal/new-service/service/example/example_registrar.go
package example

import (
    "google.golang.org/grpc"
    "github.com/gin-gonic/gin"
    "your-project/internal/new-service/handler/http/example"
)

type ServiceRegistrar struct {
    service ExampleService
}

func NewServiceRegistrar(deps ...interface{}) *ServiceRegistrar {
    // 依赖注入逻辑
    return &ServiceRegistrar{
        service: NewExampleService(deps...),
    }
}

func (sr *ServiceRegistrar) RegisterHTTP(router *gin.Engine) error {
    handler := example.NewHandler(sr.service)
    
    v1 := router.Group("/api/v1")
    {
        v1.GET("/example", handler.GetExample)
        v1.POST("/example", handler.CreateExample)
        v1.PUT("/example/:id", handler.UpdateExample)
        v1.DELETE("/example/:id", handler.DeleteExample)
    }
    
    return nil
}

func (sr *ServiceRegistrar) RegisterGRPC(server *grpc.Server) error {
    grpcHandler := NewExampleGRPCHandler(sr.service)
    pb.RegisterExampleServiceServer(server, grpcHandler)
    return nil
}

func (sr *ServiceRegistrar) GetName() string {
    return "example"
}
```

### 2. Transport Layer 实现

**HTTP Transport 模板：**

```go
// internal/new-service/transport/http.go
package transport

import (
    "context"
    "fmt"
    "net/http"
    "sync"
    "time"
    
    "github.com/gin-gonic/gin"
)

type HTTPTransport struct {
    server    *http.Server
    router    *gin.Engine
    address   string
    testPort  string // 测试端口覆盖
    ready     chan struct{}
    readyOnce sync.Once
    mu        sync.RWMutex
}

func NewHTTPTransport() *HTTPTransport {
    router := gin.New()
    router.Use(gin.Recovery())
    
    return &HTTPTransport{
        router: router,
        ready:  make(chan struct{}),
    }
}

func (ht *HTTPTransport) Start(ctx context.Context) error {
    ht.mu.Lock()
    defer ht.mu.Unlock()
    
    address := ht.address
    if ht.testPort != "" {
        address = ":" + ht.testPort
    }
    
    ht.server = &http.Server{
        Addr:    address,
        Handler: ht.router,
    }
    
    go func() {
        ht.readyOnce.Do(func() {
            close(ht.ready)
        })
        
        if err := ht.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            // 记录错误日志
        }
    }()
    
    return nil
}

func (ht *HTTPTransport) Stop(ctx context.Context) error {
    ht.mu.Lock()
    defer ht.mu.Unlock()
    
    if ht.server != nil {
        return ht.server.Shutdown(ctx)
    }
    return nil
}

func (ht *HTTPTransport) Name() string {
    return "http"
}

func (ht *HTTPTransport) Address() string {
    ht.mu.RLock()
    defer ht.mu.RUnlock()
    return ht.address
}

func (ht *HTTPTransport) SetAddress(addr string) {
    ht.mu.Lock()
    defer ht.mu.Unlock()
    ht.address = addr
}

func (ht *HTTPTransport) GetRouter() *gin.Engine {
    return ht.router
}

func (ht *HTTPTransport) WaitReady(timeout time.Duration) error {
    select {
    case <-ht.ready:
        return nil
    case <-time.After(timeout):
        return fmt.Errorf("http transport not ready within %v", timeout)
    }
}
```

**gRPC Transport 模板：**

```go
// internal/new-service/transport/grpc.go
package transport

import (
    "context"
    "net"
    "sync"
    
    "google.golang.org/grpc"
)

type GRPCTransport struct {
    server   *grpc.Server
    listener net.Listener
    address  string
    testPort string
    mu       sync.RWMutex
}

func NewGRPCTransport() *GRPCTransport {
    return &GRPCTransport{
        server: grpc.NewServer(),
    }
}

func (gt *GRPCTransport) Start(ctx context.Context) error {
    gt.mu.Lock()
    defer gt.mu.Unlock()
    
    address := gt.address
    if gt.testPort != "" {
        address = ":" + gt.testPort
    }
    
    listener, err := net.Listen("tcp", address)
    if err != nil {
        return err
    }
    
    gt.listener = listener
    
    go func() {
        if err := gt.server.Serve(listener); err != nil {
            // 记录错误日志
        }
    }()
    
    return nil
}

func (gt *GRPCTransport) Stop(ctx context.Context) error {
    gt.mu.Lock()
    defer gt.mu.Unlock()
    
    if gt.server != nil {
        gt.server.GracefulStop()
    }
    return nil
}

func (gt *GRPCTransport) Name() string {
    return "grpc"
}

func (gt *GRPCTransport) Address() string {
    gt.mu.RLock()
    defer gt.mu.RUnlock()
    return gt.address
}

func (gt *GRPCTransport) SetAddress(addr string) {
    gt.mu.Lock()
    defer gt.mu.Unlock()
    gt.address = addr
}

func (gt *GRPCTransport) GetServer() *grpc.Server {
    return gt.server
}
```

### 3. ServiceRegistry 实现

**注意：必须包含并发安全保护**

```go
// internal/new-service/transport/registry.go
package transport

import (
    "sync"
    
    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"
)

type ServiceRegistry struct {
    mu         sync.RWMutex // 必须包含锁保护
    registrars []ServiceRegistrar
}

func NewServiceRegistry() *ServiceRegistry {
    return &ServiceRegistry{
        registrars: make([]ServiceRegistrar, 0),
    }
}

func (sr *ServiceRegistry) Register(registrar ServiceRegistrar) {
    sr.mu.Lock()
    defer sr.mu.Unlock()
    sr.registrars = append(sr.registrars, registrar)
}

func (sr *ServiceRegistry) RegisterAllHTTP(router *gin.Engine) error {
    sr.mu.RLock()
    defer sr.mu.RUnlock()
    
    for _, registrar := range sr.registrars {
        if err := registrar.RegisterHTTP(router); err != nil {
            return fmt.Errorf("failed to register HTTP for %s: %w", registrar.GetName(), err)
        }
    }
    return nil
}

func (sr *ServiceRegistry) RegisterAllGRPC(server *grpc.Server) error {
    sr.mu.RLock()
    defer sr.mu.RUnlock()
    
    for _, registrar := range sr.registrars {
        if err := registrar.RegisterGRPC(server); err != nil {
            return fmt.Errorf("failed to register gRPC for %s: %w", registrar.GetName(), err)
        }
    }
    return nil
}

func (sr *ServiceRegistry) GetRegistrars() []ServiceRegistrar {
    sr.mu.RLock()
    defer sr.mu.RUnlock()
    
    result := make([]ServiceRegistrar, len(sr.registrars))
    copy(result, sr.registrars)
    return result
}
```

### 4. Server 主文件实现

```go
// internal/new-service/server.go
package newservice

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "your-project/internal/new-service/config"
    "your-project/internal/new-service/service/example"
    "your-project/internal/new-service/service/health"
    "your-project/internal/new-service/transport"
    "your-project/pkg/discovery"
    "your-project/pkg/middleware"
)

type Server struct {
    transportManager *transport.Manager
    serviceRegistry  *transport.ServiceRegistry
    httpTransport    *transport.HTTPTransport
    grpcTransport    *transport.GRPCTransport
    sd               *discovery.ServiceDiscovery
    config           *config.Config
}

func NewServer() (*Server, error) {
    cfg := config.GetConfig()
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }
    
    // 创建传输管理器和服务注册表
    transportManager := transport.NewManager()
    serviceRegistry := transport.NewServiceRegistry()
    
    // 创建传输层
    httpTransport := transport.NewHTTPTransport()
    httpTransport.SetAddress(":" + cfg.Server.Port)
    
    grpcTransport := transport.NewGRPCTransport()
    grpcTransport.SetAddress(":" + cfg.Server.GRPCPort)
    
    // 注册传输层
    transportManager.Register(httpTransport)
    transportManager.Register(grpcTransport)
    
    // 设置服务发现
    sd, err := discovery.GetServiceDiscoveryByAddress(cfg.ServiceDiscovery.Address)
    if err != nil {
        return nil, fmt.Errorf("failed to setup service discovery: %w", err)
    }
    
    server := &Server{
        transportManager: transportManager,
        serviceRegistry:  serviceRegistry,
        httpTransport:    httpTransport,
        grpcTransport:    grpcTransport,
        sd:               sd,
        config:           cfg,
    }
    
    // 注册服务
    if err := server.registerServices(); err != nil {
        return nil, fmt.Errorf("failed to register services: %w", err)
    }
    
    return server, nil
}

func (s *Server) registerServices() error {
    // 注册健康检查服务
    healthService := health.NewServiceRegistrar()
    s.serviceRegistry.Register(healthService)
    
    // 注册业务服务
    exampleService := example.NewServiceRegistrar(/* 依赖注入 */)
    s.serviceRegistry.Register(exampleService)
    
    return nil
}

func (s *Server) configureMiddleware() {
    router := s.httpTransport.GetRouter()
    
    // 注册全局中间件
    middlewareRegistrar := middleware.NewGlobalMiddlewareRegistrar()
    if err := middlewareRegistrar.RegisterMiddleware(router); err != nil {
        // 记录错误日志
    }
}

func (s *Server) Start(ctx context.Context) error {
    // 注册服务到传输层
    if err := s.serviceRegistry.RegisterAllHTTP(s.httpTransport.GetRouter()); err != nil {
        return fmt.Errorf("failed to register HTTP services: %w", err)
    }
    
    if err := s.serviceRegistry.RegisterAllGRPC(s.grpcTransport.GetServer()); err != nil {
        return fmt.Errorf("failed to register gRPC services: %w", err)
    }
    
    // 配置中间件
    s.configureMiddleware()
    
    // 启动传输层
    if err := s.transportManager.Start(ctx); err != nil {
        return fmt.Errorf("failed to start transports: %w", err)
    }
    
    // 注册到服务发现
    if err := s.registerWithDiscovery(); err != nil {
        return fmt.Errorf("failed to register with discovery: %w", err)
    }
    
    return nil
}

func (s *Server) registerWithDiscovery() error {
    // HTTP 服务注册
    httpServiceID := fmt.Sprintf("%s-http-%s", s.config.ServiceName, s.config.Server.Port)
    if err := s.sd.RegisterService(httpServiceID, s.config.ServiceName, s.config.Server.Port, "http"); err != nil {
        return fmt.Errorf("failed to register HTTP service: %w", err)
    }
    
    // gRPC 服务注册
    grpcServiceID := fmt.Sprintf("%s-grpc-%s", s.config.ServiceName, s.config.Server.GRPCPort)
    if err := s.sd.RegisterService(grpcServiceID, s.config.ServiceName, s.config.Server.GRPCPort, "grpc"); err != nil {
        return fmt.Errorf("failed to register gRPC service: %w", err)
    }
    
    return nil
}

func (s *Server) Shutdown() error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // 从服务发现注销
    if err := s.deregisterFromDiscovery(); err != nil {
        // 记录错误日志，但不阻止关闭流程
    }
    
    // 停止传输层
    return s.transportManager.Stop(ctx)
}

func (s *Server) deregisterFromDiscovery() error {
    httpServiceID := fmt.Sprintf("%s-http-%s", s.config.ServiceName, s.config.Server.Port)
    grpcServiceID := fmt.Sprintf("%s-grpc-%s", s.config.ServiceName, s.config.Server.GRPCPort)
    
    if err := s.sd.DeregisterService(httpServiceID); err != nil {
        return fmt.Errorf("failed to deregister HTTP service: %w", err)
    }
    
    if err := s.sd.DeregisterService(grpcServiceID); err != nil {
        return fmt.Errorf("failed to deregister gRPC service: %w", err)
    }
    
    return nil
}

func (s *Server) Run() error {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // 启动服务器
    if err := s.Start(ctx); err != nil {
        return err
    }
    
    // 等待信号
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    <-sigChan
    
    // 优雅关闭
    return s.Shutdown()
}
```

### 5. 配置管理

```go
// internal/new-service/config/config.go
package config

import (
    "errors"
    "sync"
    
    "github.com/spf13/viper"
)

type Config struct {
    ServiceName string `json:"serviceName" yaml:"serviceName"`
    Database    struct {
        Host     string `json:"host" yaml:"host"`
        Port     string `json:"port" yaml:"port"`
        Username string `json:"username" yaml:"username"`
        Password string `json:"password" yaml:"password"`
        DBName   string `json:"dbname" yaml:"dbname"`
    } `json:"database" yaml:"database"`
    Server struct {
        Port     string `json:"port" yaml:"port"`
        GRPCPort string `json:"grpcPort" yaml:"grpcPort"`
    } `json:"server" yaml:"server"`
    ServiceDiscovery struct {
        Address string `json:"address" yaml:"address"`
    } `json:"serviceDiscovery" yaml:"serviceDiscovery"`
}

var (
    config *Config
    once   sync.Once
)

func GetConfig() *Config {
    once.Do(func() {
        config = loadConfig()
    })
    return config
}

func loadConfig() *Config {
    viper.SetConfigName("new-service") // 配置文件名
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("./configs")
    viper.AddConfigPath("/etc/new-service")
    
    // 设置默认值
    viper.SetDefault("serviceName", "new-service")
    viper.SetDefault("server.port", "8080")
    viper.SetDefault("server.grpcPort", "9090")
    viper.SetDefault("serviceDiscovery.address", "localhost:8500")
    
    if err := viper.ReadInConfig(); err != nil {
        // 记录警告日志，使用默认配置
    }
    
    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        panic(err)
    }
    
    return &cfg
}

func (c *Config) Validate() error {
    if c.ServiceName == "" {
        return errors.New("service name is required")
    }
    
    if c.Server.Port == "" {
        return errors.New("server port is required")
    }
    
    if c.Server.GRPCPort == "" {
        return errors.New("gRPC port is required")
    }
    
    if c.ServiceDiscovery.Address == "" {
        return errors.New("service discovery address is required")
    }
    
    return nil
}
```

## 实施步骤

### 第一阶段：项目初始化

1. **创建项目目录结构**
   ```bash
   mkdir -p internal/new-service/{cmd,config,db,handler/http,model,repository,service,transport}
   mkdir -p cmd/new-service
   ```

2. **创建配置文件**
   ```yaml
   # new-service.yaml
   serviceName: "new-service"
   database:
     host: "localhost"
     port: "3306"
     username: "root"
     password: "password"
     dbname: "new_service_db"
   server:
     port: "8080"
     grpcPort: "9090"
   serviceDiscovery:
     address: "localhost:8500"
   ```

3. **实现基础传输层**
   - 复制并修改 `transport` 包
   - 确保包含线程安全保护
   - 添加适当的错误处理

### 第二阶段：服务层开发

1. **定义业务接口**
   ```go
   type ExampleService interface {
       GetExample(ctx context.Context, id string) (*model.Example, error)
       CreateExample(ctx context.Context, example *model.Example) error
       UpdateExample(ctx context.Context, example *model.Example) error
       DeleteExample(ctx context.Context, id string) error
   }
   ```

2. **实现 ServiceRegistrar**
   - 每个服务一个 registrar
   - 支持 HTTP 和 gRPC 双协议
   - 包含依赖注入逻辑

3. **实现处理器层**
   - HTTP 处理器使用 Gin
   - gRPC 处理器实现 protobuf 接口
   - 统一错误处理和响应格式

### 第三阶段：集成和测试

1. **集成服务发现**
   - 使用现有的 `pkg/discovery` 包
   - 实现服务注册和注销
   - 添加健康检查端点

2. **添加中间件**
   - 使用 `pkg/middleware` 包
   - 配置认证、日志、CORS 等中间件
   - 确保中间件顺序正确

3. **编写测试**
   - 单元测试覆盖所有组件
   - 集成测试验证端到端流程
   - 性能测试确保满足要求

### 第四阶段：部署准备

1. **Docker 化**
   ```dockerfile
   FROM golang:1.21-alpine AS builder
   WORKDIR /app
   COPY . .
   RUN go build -o new-service cmd/new-service/new-service.go
   
   FROM alpine:latest
   RUN apk --no-cache add ca-certificates
   WORKDIR /root/
   COPY --from=builder /app/new-service .
   COPY --from=builder /app/new-service.yaml .
   CMD ["./new-service"]
   ```

2. **Kubernetes 配置**
   - 创建 Deployment、Service、ConfigMap
   - 配置健康检查和资源限制
   - 设置环境变量和密钥管理

3. **监控和日志**
   - 集成 Prometheus 指标
   - 配置结构化日志
   - 添加链路追踪

## 最佳实践清单

### ✅ 必须遵循的规范

- [ ] 使用统一的 `ServiceRegistrar` 接口
- [ ] 实现线程安全的 `ServiceRegistry`
- [ ] 支持 HTTP 和 gRPC 双协议
- [ ] 集成服务发现机制
- [ ] 实现优雅关闭
- [ ] 添加配置验证
- [ ] 包含健康检查端点
- [ ] 使用依赖注入模式
- [ ] 实现统一错误处理
- [ ] 添加结构化日志

### ✅ 推荐的增强功能

- [ ] 添加 Prometheus 指标
- [ ] 实现链路追踪
- [ ] 配置熔断器
- [ ] 添加缓存层
- [ ] 实现数据库迁移
- [ ] 配置 API 文档生成
- [ ] 添加性能测试
- [ ] 实现配置热重载

### ⚠️ 常见陷阱避免

- **并发安全**: 确保所有共享资源都有适当的锁保护
- **资源泄漏**: 正确关闭数据库连接、HTTP 客户端等资源
- **错误处理**: 不要忽略错误，使用包装错误提供上下文
- **配置管理**: 不要硬编码配置，使用环境变量或配置文件
- **测试覆盖**: 确保关键路径都有测试覆盖
- **依赖循环**: 避免包之间的循环依赖
- **接口设计**: 保持接口简单和专注

## 示例项目模板

为了快速开始，可以参考以下命令创建项目模板：

```bash
# 1. 创建项目结构
mkdir -p internal/new-service/{cmd,config,db,handler/http,model,repository,service,transport}
mkdir -p cmd/new-service

# 2. 复制基础文件
cp internal/switserve/transport/* internal/new-service/transport/
cp internal/switserve/config/* internal/new-service/config/

# 3. 修改包名和配置
# 使用 sed 或手动修改文件中的包名引用

# 4. 创建配置文件
cp switserve.yaml new-service.yaml
# 修改配置文件中的服务名称和端口

# 5. 创建主程序
cp cmd/swit-serve/swit-serve.go cmd/new-service/new-service.go
# 修改导入路径和服务名称
```

## 总结

遵循本指导方案可以确保新项目：

1. **架构一致性**: 与现有项目保持相同的设计模式
2. **代码复用**: 最大化利用现有的基础设施代码
3. **可维护性**: 清晰的分层和职责分离
4. **可扩展性**: 易于添加新功能和服务
5. **生产就绪**: 包含监控、日志、健康检查等生产环境必需功能

建议在开始新项目开发前，先阅读现有项目的代码，理解其设计思路，然后按照本指南逐步实施。如有疑问，可以参考 `switauth` 和 `switserve` 的具体实现。
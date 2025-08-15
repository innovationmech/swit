---
layout: home
title: Swit Go 微服务框架
titleTemplate: Go 微服务开发框架

hero:
  name: "Swit"
  text: "Go 微服务框架"
  tagline: 生产就绪的微服务开发基础设施
  actions:
    - theme: brand
      text: 快速开始
      link: /zh/guide/getting-started
    - theme: alt
      text: 查看 API
      link: /zh/api/

features:
  - title: 统一的服务器框架
    details: 完整的服务器生命周期管理，包括传输协调和健康监控
    icon: 🚀
  - title: 多传输层支持
    details: 无缝的 HTTP 和 gRPC 传输协调，支持可插拔架构
    icon: 🔄
  - title: 依赖注入系统
    details: 基于工厂的依赖容器，支持自动生命周期管理
    icon: 📦
  - title: 性能监控
    details: 内置指标收集和性能分析，支持阈值监控
    icon: 📊
  - title: 服务发现
    details: 基于 Consul 的服务注册和健康检查集成
    icon: 🔍
  - title: 示例丰富
    details: 完整的参考实现和最佳实践示例
    icon: 📚
---

# Swit

## 项目状态

<div class="project-badges">

[![CI](https://github.com/innovationmech/swit/workflows/CI/badge.svg)](https://github.com/innovationmech/swit/actions/workflows/ci.yml)
[![Security Checks](https://github.com/innovationmech/swit/workflows/Security%20Checks/badge.svg)](https://github.com/innovationmech/swit/actions/workflows/security-checks.yml)
[![codecov](https://codecov.io/gh/innovationmech/swit/branch/master/graph/badge.svg)](https://codecov.io/gh/innovationmech/swit)
[![Go Report Card](https://goreportcard.com/badge/github.com/innovationmech/swit)](https://goreportcard.com/report/github.com/innovationmech/swit)
[![Go Reference](https://pkg.go.dev/badge/github.com/innovationmech/swit.svg)](https://pkg.go.dev/github.com/innovationmech/swit)
[![GitHub release](https://img.shields.io/github/release/innovationmech/swit.svg)](https://github.com/innovationmech/swit/releases)
[![License](https://img.shields.io/github/license/innovationmech/swit.svg)](LICENSE)
[![GitHub issues](https://img.shields.io/github/issues/innovationmech/swit.svg)](https://github.com/innovationmech/swit/issues)
[![GitHub stars](https://img.shields.io/github/stars/innovationmech/swit.svg)](https://github.com/innovationmech/swit/stargazers)

</div>

<style>
.project-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 1rem 0;
}

.project-badges img {
  height: 20px;
}
</style>

![Go Version](https://img.shields.io/badge/go-%3E%3D1.24-blue.svg)





Swit 是一个全面的 Go 微服务框架，为构建可扩展微服务提供统一、生产就绪的基础。该框架专注于开发人员生产力和架构一致性，提供完整的基线服务器框架、统一的传输层、依赖注入系统和全面的工具链，用于快速微服务开发。


## 快速开始

开始使用 Swit 框架构建您的第一个微服务：

```bash
# 克隆项目
git clone https://github.com/innovationmech/swit.git
cd swit

# 安装依赖
go mod tidy

# 构建项目
make build

# 运行示例服务
./bin/swit-serve
```

::: tip 提示
查看我们的 [详细指南](/zh/guide/getting-started) 获取更多信息。
:::

## 框架特性

- **基线服务器框架**: 完整的服务器生命周期管理，通过 `BusinessServerCore` 接口和统一的服务注册模式提供
- **统一传输层**: 通过 `TransportCoordinator` 实现 HTTP 和 gRPC 传输的无缝协调，具有可插拔的传输架构
- **依赖注入系统**: 基于工厂的依赖容器，支持单例/瞬态模式和自动生命周期管理
- **配置管理**: 全面的配置验证，支持基于环境的覆盖和合理的默认值
- **性能监控**: 内置指标收集、性能分析和监控钩子，支持阈值违规检测
- **服务发现集成**: 基于 Consul 的服务注册，支持健康检查集成和自动注销
- **中间件框架**: 可配置的中间件堆栈，支持 HTTP 和 gRPC 传输，包括 CORS、速率限制和超时
- **健康检查系统**: 全面的健康监控，支持服务聚合和超时处理
- **优雅的生命周期管理**: 分阶段的启动/关闭，支持适当的资源清理和错误处理
- **协议缓冲区集成**: Buf 工具链支持 API 版本控制和自动生成文档
- **示例服务**: 完整的参考实现，展示框架使用模式和最佳实践

## 框架架构

Swit 框架由以下核心组件组成：

### 核心框架 (`pkg/server/`)
- **BusinessServerCore**: 主服务器接口，提供生命周期管理、传输协调和服务健康监控
- **BusinessServerImpl**: 完整的服务器实现，包括传输管理、服务发现和性能监控
- **BusinessServiceRegistrar**: 服务注册到框架传输层的接口模式
- **BusinessDependencyContainer**: 基于工厂的依赖注入系统，支持生命周期管理

### 传输层 (`pkg/transport/`)
- **TransportCoordinator**: 中央协调器，管理多个传输实例（HTTP/gRPC），支持统一的服务注册
- **NetworkTransport**: 传输实现的基础接口，具有可插拔架构
- **MultiTransportRegistry**: 服务注册管理器，处理跨传输操作和健康检查

### 示例服务 (`internal/`)
- **switserve**: 用户管理服务，展示完整的框架使用，包括 HTTP/gRPC 端点
- **switauth**: 身份验证服务，展示 JWT 集成和服务间通信
- **switctl**: 命令行工具示例，展示框架集成模式

### 框架支持 (`pkg/`)
- **Discovery**: 基于 Consul 的服务发现，支持自动注册/注销
- **Middleware**: HTTP 和 gRPC 中间件堆栈，支持 CORS、速率限制、超时和身份验证
- **Types**: 常用类型定义和健康检查抽象
- **Utils**: 加密工具、JWT 处理和安全组件

## 框架 API 架构

Swit 框架通过 Buf 工具链为 gRPC API 提供全面的 API 开发支持：

```text {1-10}
api/
├── buf.yaml              # Buf 主配置文件
├── buf.gen.yaml          # 代码生成配置
├── buf.lock              # 依赖锁文件
└── proto/               # Protocol Buffer 定义
    └── swit/
        ├── common/
        │   └── v1/
        │       ├── common.proto
        │       └── health.proto
        ├── communication/
        │   └── v1/
        │       └── notification.proto
        ├── interaction/
        │   └── v1/
        │       └── greeter.proto
        └── user/
            └── v1/
                ├── auth.proto
                └── user.proto
```

如需生成 `api/gen/` 目录和 Swagger 文档，可执行 `make proto` 和 `make swagger`，这些产物不会提交到版本库中。

### API 设计原则

- **版本化**: 所有 API 都有明确的版本号（v1, v2, ...）
- **模块化**: 按服务域组织 proto 文件
- **双协议**: 同时支持 gRPC 和 HTTP/REST
- **自动化**: 使用 Buf 工具链自动生成代码和文档

## 框架示例和服务参考实现

Swit 框架包含全面的示例，展示各种使用模式：

### 简单示例 (`examples/`)

#### `examples/simple-http-service/`
- **目的**: 基本 HTTP-only 服务演示
- **特性**: RESTful API 端点、健康检查、优雅关闭
- **适用场景**: 开始使用框架、仅 HTTP 服务
- **关键概念**: `BusinessServiceRegistrar` 实现、HTTP 路由模式

#### `examples/grpc-service/`
- **目的**: gRPC 服务实现展示
- **特性**: Protocol Buffer 定义、gRPC 服务器设置、流式支持
- **适用场景**: gRPC 专注的微服务、服务间通信
- **关键概念**: `BusinessGRPCService` 实现、Protocol Buffer 集成

#### `examples/full-featured-service/`
- **目的**: 完整框架功能演示
- **特性**: HTTP + gRPC、依赖注入、服务发现、中间件
- **适用场景**: 生产就绪的服务模式、框架评估
- **关键概念**: 多传输服务、高级配置、监控

### 参考服务 (`internal/`)

#### `internal/switserve/` - 用户管理服务
- **目的**: 全面的用户管理微服务
- **架构**: 完整框架集成，包括数据库、缓存和外部服务通信
- **特性**:
  - 用户 CRUD 操作（HTTP REST + gRPC）
  - 带流式支持的问候服务
  - 通知系统集成
  - 健康监控和优雅关闭
  - GORM 数据库集成
  - 中间件堆栈演示

#### `internal/switauth/` - 身份验证服务  
- **目的**: 基于 JWT 的身份验证微服务
- **架构**: 安全的身份验证模式，支持令牌管理
- **特性**:
  - 用户登录/登出（HTTP + gRPC）
  - JWT 令牌生成和验证  
  - 令牌刷新和撤销
  - 密码重置工作流
  - 服务间身份验证
  - Redis 集成用于会话管理

#### `internal/switctl/` - CLI 工具
- **目的**: 命令行管理工具
- **架构**: 框架集成模式用于 CLI 应用程序
- **特性**:
  - 健康检查命令
  - 服务管理操作
  - 版本信息和诊断

### 展示的使用模式

1. **服务注册**: HTTP 和 gRPC 服务的多种实现模式
2. **配置管理**: 基于环境的配置和验证
3. **依赖注入**: 数据库连接、Redis 客户端、外部服务客户端
4. **中间件集成**: 身份验证、CORS、速率限制、日志记录
5. **健康监控**: 服务健康检查和就绪探针
6. **性能监控**: 指标收集和性能分析
7. **服务发现**: Consul 注册和服务查找模式
8. **测试策略**: 单元测试、集成测试和性能基准测试

## 框架接口和模式

### 核心服务器接口

#### `BusinessServerCore`
主服务器接口，提供完整的生命周期管理：
```go {1-10}
type BusinessServerCore interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Shutdown() error
    GetHTTPAddress() string
    GetGRPCAddress() string
    GetTransports() []transport.NetworkTransport
    GetTransportStatus() map[string]TransportStatus
}
```

#### `BusinessServiceRegistrar`
服务注册到框架的接口：
```go {1-10}
type BusinessServiceRegistrar interface {
    RegisterServices(registry BusinessServiceRegistry) error
}
```

#### `BusinessServiceRegistry` 
不同服务类型的注册接口：
```go {1-10}
type BusinessServiceRegistry interface {
    RegisterBusinessHTTPHandler(handler BusinessHTTPHandler) error
    RegisterBusinessGRPCService(service BusinessGRPCService) error
    RegisterBusinessHealthCheck(check BusinessHealthCheck) error
}
```

### 传输层接口

#### `BusinessHTTPHandler`
HTTP 服务实现的接口：
```go {1-10}
type BusinessHTTPHandler interface {
    RegisterRoutes(router interface{}) error
    GetServiceName() string
}
```

#### `BusinessGRPCService`
gRPC 服务实现的接口：
```go {1-10}
type BusinessGRPCService interface {
    RegisterGRPC(server interface{}) error
    GetServiceName() string
}
```

#### `BusinessHealthCheck`
服务健康监控的接口：
```go {1-10}
type BusinessHealthCheck interface {
    Check(ctx context.Context) error
    GetServiceName() string
}
```

### 依赖管理接口

#### `BusinessDependencyContainer`
依赖注入和生命周期管理：
```go {1-10}
type BusinessDependencyContainer interface {
    Close() error
    GetService(name string) (interface{}, error)
}
```

#### `BusinessDependencyRegistry`
扩展的依赖管理，支持工厂模式：
```go {1-10}
type BusinessDependencyRegistry interface {
    BusinessDependencyContainer
    Initialize(ctx context.Context) error
    RegisterSingleton(name string, factory DependencyFactory) error
    RegisterTransient(name string, factory DependencyFactory) error
    RegisterInstance(name string, instance interface{}) error
}
```

### 配置接口

#### `ConfigValidator`
配置验证和默认值：
```go {1-10}
type ConfigValidator interface {
    Validate() error
    SetDefaults()
}
```

### 服务实现示例

框架包括这些接口的工作示例：
- **HTTP 服务**: RESTful API，使用 Gin 路由器集成
- **gRPC 服务**: Protocol Buffer 服务实现
- **健康检查**: 数据库连接、外部服务检查
- **依赖注入**: 数据库连接、Redis 客户端、外部 API
- **配置**: 基于环境的配置和验证

## 要求

### 框架核心要求
- **Go 1.24+** - 支持泛型的现代 Go 版本
- **Git** - 用于框架和示例代码管理

### 可选依赖（服务特定）
- **MySQL 8.0+** - 用于数据库支持的服务（在示例中演示）
- **Redis 6.0+** - 用于缓存和会话管理（在认证示例中使用）
- **Consul 1.12+** - 用于服务发现（可选，可以禁用）

### 开发工具
- **Buf CLI 1.0+** - 用于 Protocol Buffer API 开发
- **Docker 20.10+** - 用于容器化部署和开发
- **Make** - 用于构建自动化（在大多数系统上标准）

## 快速开始

### 1. 获取框架
```bash {1-10}
git clone https://github.com/innovationmech/swit.git
cd swit
go mod download
```

### 2. 创建简单服务
```go {1-10}
// main.go
package main

import (
    "context"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/server"
)

// MyService 实现 BusinessServiceRegistrar 接口
type MyService struct{}

func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
    httpHandler := &MyHTTPHandler{}
    return registry.RegisterBusinessHTTPHandler(httpHandler)
}

// MyHTTPHandler 实现 BusinessHTTPHandler 接口
type MyHTTPHandler struct{}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    ginRouter.GET("/hello", h.handleHello)
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return "my-service"
}

func (h *MyHTTPHandler) handleHello(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "来自 Swit 框架的问候！"})
}

func main() {
    config := &server.ServerConfig{
        ServiceName: "my-service",
        HTTP: server.HTTPConfig{Port: "8080", Enabled: true},
        GRPC: server.GRPCConfig{Enabled: false},
    }
    
    service := &MyService{}
    baseServer, _ := server.NewBusinessServerCore(config, service, nil)
    
    ctx := context.Background()
    baseServer.Start(ctx)
    defer baseServer.Shutdown()
    
    // 服务器运行在 :8080
    select {} // 保持运行
}
```

### 3. 运行服务
```bash {1-10}
go run main.go
curl http://localhost:8080/hello
```

### 4. 探索示例
```bash {1-10}
# 简单 HTTP 服务
cd examples/simple-http-service
go run main.go

# gRPC 服务示例  
cd examples/grpc-service  
go run main.go

# 全功能服务
cd examples/full-featured-service
go run main.go
```

### 5. 构建框架组件
```bash {1-10}
# 构建所有框架组件和示例
make build

# 快速开发构建
make build-dev

# 运行示例服务
./bin/swit-serve    # 用户管理示例
./bin/swit-auth     # 身份验证示例
```

## 框架开发

### 开发环境设置
```bash {1-10}
# 设置完整的框架开发环境
make setup-dev

# 快速设置仅必需的组件
make setup-quick

# 设置单个开发工具
make proto-setup    # Protocol Buffer 工具链
make swagger-setup  # OpenAPI 文档工具
make quality-setup  # 代码质量工具
```

### 框架开发命令

#### 框架 API 开发
```bash {1-10}
# 完整的 API 开发工作流
make proto          # 生成 protobuf 代码 + 文档
make swagger        # 生成 OpenAPI 文档

# 快速开发迭代
make proto-dev      # 跳过依赖检查
make swagger-dev    # 跳过格式化步骤

# 高级 API 操作
make proto-advanced OPERATION=format    # 格式化 proto 文件
make proto-advanced OPERATION=lint      # 检查 proto 定义
make proto-advanced OPERATION=breaking  # 检查破坏性变更
make proto-advanced OPERATION=docs      # 仅生成文档
```

#### 框架扩展开发
```bash {1-10}
# 构建框架组件和示例
make build          # 完整框架构建
make build-dev      # 快速构建（跳过质量检查）
make build-release  # 多平台发布构建

# 框架测试
make test           # 完整测试套件
make test-dev       # 快速测试（跳过代码生成）
make test-coverage  # 生成覆盖率报告
make test-race      # 检测竞争条件
```

### 框架扩展工作流

1. **创建服务**
   ```bash {1-10}
   # 创建服务目录
   mkdir my-service
   cd my-service
   
   # 使用框架依赖初始化
   go mod init my-service
   go get github.com/innovationmech/swit
   ```

2. **实现框架接口**
   ```go {1-10}
   // 实现 BusinessServiceRegistrar
   type MyService struct{}
   
   func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
       // 注册 HTTP/gRPC 处理程序
       return nil
   }
   ```

3. **配置和测试**
   ```bash {1-10}
   # 如果使用 gRPC，生成 proto 代码
   make proto-generate
   
   # 构建和测试服务
   go build .
   go test ./...
   
   # 使用框架运行
   ./my-service
   ```

4. **框架集成测试**
   ```bash {1-10}
   # 使用框架示例进行测试
   cd examples/simple-http-service
   go run main.go
   
   # 集成测试
   make test-integration
   ```

### 贡献框架

1. **框架核心开发**
   ```bash {1-10}
   # 在 pkg/server/ 或 pkg/transport/ 中工作
   vim pkg/server/interfaces.go
   
   # 测试框架更改
   make test-advanced TYPE=unit PACKAGE=pkg
   ```

2. **添加新示例**
   ```bash {1-10}
   # 创建新的示例服务
   mkdir examples/my-example
   # 遵循现有服务的示例模式
   ```

3. **文档更新**
   ```bash {1-10}
   # 更新框架文档
   vim pkg/server/CLAUDE.md
   vim pkg/transport/CLAUDE.md
   ```

## 框架配置

### 核心服务器配置
框架使用 `ServerConfig` 结构进行全面的服务器设置：

```go {1-10}
type ServerConfig struct {
    ServiceName     string           // 服务标识
    HTTP            HTTPConfig       // HTTP 传输配置  
    GRPC            GRPCConfig       // gRPC 传输配置
    Discovery       DiscoveryConfig  // 服务发现设置
    Middleware      MiddlewareConfig // 中间件配置
    ShutdownTimeout time.Duration    // 优雅关闭超时
}
```

### HTTP 传输配置
```go {1-10}
type HTTPConfig struct {
    Port         string            // 监听端口（例如 "8080"）
    Address      string            // 监听地址（例如 ":8080"）
    Enabled      bool              // 启用 HTTP 传输
    EnableReady  bool              // 测试的就绪通道
    TestMode     bool              // 测试模式设置
    ReadTimeout  time.Duration     // 读取超时
    WriteTimeout time.Duration     // 写入超时
    IdleTimeout  time.Duration     // 空闲超时
    Headers      map[string]string // 默认头
    Middleware   HTTPMiddleware    // 中间件配置
}
```

### gRPC 传输配置
```go {1-10}
type GRPCConfig struct {
    Port                string              // 监听端口（例如 "9080"）
    Address             string              // 监听地址
    Enabled             bool                // 启用 gRPC 传输
    EnableKeepalive     bool                // 启用保持连接
    EnableReflection    bool                // 启用反射
    EnableHealthService bool                // 启用健康服务
    MaxRecvMsgSize      int                 // 最大接收消息大小
    MaxSendMsgSize      int                 // 最大发送消息大小
    KeepaliveParams     GRPCKeepaliveParams // 保持连接参数
}
```

### 服务发现配置
```go {1-10}
type DiscoveryConfig struct {
    Enabled     bool     // 启用服务发现
    Address     string   // Consul 地址（例如 "localhost:8500"）
    ServiceName string   // 注册的服务名称
    Tags        []string // 服务标签
    CheckPath   string   // 健康检查路径
    CheckInterval string // 健康检查间隔
}
```

### 框架配置示例（YAML）
```yaml {1-10}
service_name: "my-microservice"
shutdown_timeout: "30s"

http:
  enabled: true
  port: "8080" 
  read_timeout: "30s"
  write_timeout: "30s"
  middleware:
    enable_cors: true
    enable_logging: true
    enable_timeout: true

grpc:
  enabled: true
  port: "9080"
  enable_keepalive: true
  enable_reflection: true
  enable_health_service: true

discovery:
  enabled: true
  address: "127.0.0.1:8500"
  service_name: "my-microservice"
  tags: ["v1", "production"]
  check_path: "/health"
  check_interval: "10s"

middleware:
  enable_cors: true
  enable_logging: true
  cors:
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE"]
    allowed_headers: ["*"]
```

### 环境变量配置
```bash {1-10}
# 服务配置
SERVICE_NAME=my-microservice
SHUTDOWN_TIMEOUT=30s

# HTTP 传输
HTTP_ENABLED=true
HTTP_PORT=8080
HTTP_READ_TIMEOUT=30s

# gRPC 传输  
GRPC_ENABLED=true
GRPC_PORT=9080
GRPC_ENABLE_REFLECTION=true

# 服务发现
DISCOVERY_ENABLED=true
CONSUL_ADDRESS=localhost:8500
DISCOVERY_SERVICE_NAME=my-microservice
```

## Docker 部署

### 构建镜像
```bash {1-10}
make docker
```

### 运行容器
```bash {1-10}
# 运行用户服务
docker run -d -p 9000:9000 -p 10000:10000 --name swit-serve swit-serve:latest

# 运行身份验证服务
docker run -d -p 9001:9001 --name swit-auth swit-auth:latest
```

### 使用 Docker Compose
```bash {1-10}
docker-compose up -d
```

## 测试

### 运行所有测试
```bash {1-10}
make test
```

### 快速开发测试
```bash {1-10}
make test-dev
```

### 测试覆盖率
```bash {1-10}
make test-coverage
```

### 高级测试
```bash {1-10}
# 运行特定类型的测试
make test-advanced TYPE=unit
make test-advanced TYPE=race
make test-advanced TYPE=bench

# 运行特定包的测试
make test-advanced TYPE=unit PACKAGE=internal
make test-advanced TYPE=unit PACKAGE=pkg
```

## 开发环境

### 设置开发环境
```bash {1-10}
# 完整开发设置（推荐）
make setup-dev

# 快速设置最小要求
make setup-quick
```

### 可用服务和端口
- **swit-serve**: HTTP: 9000, gRPC: 10000
- **swit-auth**: HTTP: 9001, gRPC: 50051
- **switctl**: CLI 工具（无 HTTP/gRPC 端点）

### 开发工具

#### 代码质量
```bash {1-10}
# 标准质量检查（推荐用于 CI/CD）
make quality

# 快速质量检查（开发时使用）
make quality-dev

# 设置质量工具
make quality-setup
```

#### 代码格式化和检查
```bash {1-10}
# 格式化代码
make format

# 检查代码
make vet

# 代码规范检查
make lint

# 安全扫描
make security
```

#### 依赖管理
```bash {1-10}
# 整理 Go 模块
make tidy
```

### 构建命令

#### 标准构建
```bash {1-10}
# 构建所有服务（开发模式）
make build

# 快速构建（跳过质量检查）
make build-dev

# 发布构建（所有平台）
make build-release
```

#### 高级构建
```bash {1-10}
# 为特定平台构建特定服务
make build-advanced SERVICE=swit-serve PLATFORM=linux/amd64
make build-advanced SERVICE=swit-auth PLATFORM=darwin/arm64
```

### 清理

```bash {1-10}
# 标准清理（所有生成文件）
make clean

# 快速清理（仅构建输出）
make clean-dev

# 深度清理（重置环境）
make clean-setup

# 高级清理（特定类型）
make clean-advanced TYPE=build
make clean-advanced TYPE=proto
make clean-advanced TYPE=swagger
```

### CI/CD 和版权管理

#### CI 流水线
```bash {1-10}
# 运行 CI 流水线（自动化测试和质量检查）
make ci
```

#### 版权管理
```bash {1-10}
# 检查并修复版权声明
make copyright

# 仅检查版权声明
make copyright-check

# 为新项目设置版权
make copyright-setup
```

### Docker 开发

```bash {1-10}
# 标准 Docker 构建（生产环境）
make docker

# 快速 Docker 构建（开发时使用缓存）
make docker-dev

# 设置 Docker 开发环境
make docker-setup

# 高级 Docker 操作
make docker-advanced OPERATION=build COMPONENT=images SERVICE=auth
```

## Makefile 命令参考

项目使用全面的 Makefile 系统，命令组织清晰。以下是快速参考：

### 核心开发命令
```bash {1-10}
make all              # 完整构建流水线 (proto + swagger + tidy + copyright + build)
make setup-dev        # 设置完整开发环境
make setup-quick      # 快速设置（最小组件）
make ci               # 运行 CI 流水线
```

### 构建命令
```bash {1-10}
make build            # 标准构建（开发模式）
make build-dev        # 快速构建（跳过质量检查）
make build-release    # 发布构建（所有平台）
make build-advanced   # 高级构建（需要 SERVICE 和 PLATFORM 参数）
```

### 测试命令
```bash {1-10}
make test             # 运行所有测试（包含依赖生成）
make test-dev         # 快速开发测试
make test-coverage    # 生成覆盖率报告
make test-advanced    # 高级测试（需要 TYPE 和 PACKAGE 参数）
```

### 质量检查命令
```bash {1-10}
make quality          # 标准质量检查（CI/CD）
make quality-dev      # 快速质量检查（开发）
make quality-setup    # 设置质量检查工具
make tidy             # 整理 Go 模块
make format           # 格式化代码
make vet              # 代码检查
make lint             # 代码规范检查
make security         # 安全扫描
```

### API 开发命令
```bash {1-10}
make proto            # 生成 protobuf 代码
make proto-dev        # 快速 proto 生成
make proto-setup      # 设置 protobuf 工具
make swagger          # 生成 swagger 文档
make swagger-dev      # 快速 swagger 生成
make swagger-setup    # 设置 swagger 工具
```

### 清理命令
```bash {1-10}
make clean            # 标准清理（所有生成文件）
make clean-dev        # 快速清理（仅构建输出）
make clean-setup      # 深度清理（重置环境）
make clean-advanced   # 高级清理（需要 TYPE 参数）
```

### Docker 命令
```bash {1-10}
make docker           # 标准 Docker 构建
make docker-dev       # 快速 Docker 构建（使用缓存）
make docker-setup     # 设置 Docker 开发环境
make docker-advanced  # 高级 Docker 操作
```

### 版权管理命令
```bash {1-10}
make copyright        # 检查并修复版权声明
make copyright-check  # 仅检查版权声明
make copyright-setup  # 为新项目设置版权
```

### 帮助命令
```bash {1-10}
make help             # 显示所有可用命令及说明
```

如需了解详细的命令选项和参数，请运行 `make help` 或参考 `scripts/mk/` 目录下的具体 `.mk` 文件。

### 框架使用示例

#### 示例服务（参考实现）

#### `examples/simple-http-service/`
- **HTTP**: `http://localhost:8080`（可配置）
- **目的**: 基本框架演示
- **端点**:
  - `GET /api/v1/hello?name=<name>` - 问候端点
  - `GET /api/v1/status` - 服务状态
  - `POST /api/v1/echo` - 回显请求体
  - `GET /health` - 健康检查（自动注册）

#### `internal/switserve/` (用户管理参考)
- **HTTP**: `http://localhost:9000`
- **gRPC**: `http://localhost:10000`
- **目的**: 完整框架功能演示
- **框架功能演示**:
  - 多传输服务注册
  - 数据库集成模式
  - 健康检查实现
  - 依赖注入使用
  - 中间件配置

#### `internal/switauth/` (身份验证参考)
- **HTTP**: `http://localhost:9001`
- **gRPC**: `http://localhost:50051`
- **目的**: 身份验证服务模式
- **框架功能演示**:
  - JWT 中间件集成
  - 服务间通信
  - Redis 依赖注入
  - 安全配置模式

#### 框架可用的模式

1. **HTTP 服务注册**
   ```go {1-10}
   func (h *MyHandler) RegisterRoutes(router interface{}) error {
       ginRouter := router.(*gin.Engine)
       ginRouter.GET("/api/v1/my-endpoint", h.handleEndpoint)
       return nil
   }
   ```

2. **gRPC 服务注册**
   ```go {1-10}
   func (s *MyService) RegisterGRPC(server interface{}) error {
       grpcServer := server.(*grpc.Server)
       mypb.RegisterMyServiceServer(grpcServer, s)
       return nil
   }
   ```

3. **健康检查实现**
   ```go {1-10}
   func (h *MyHealthCheck) Check(ctx context.Context) error {
       // 实现健康检查逻辑
       return nil
   }
   ```

## 框架文档

### 核心框架指南
- [基线服务器框架](/zh/guide/docs/base-server-framework.md) - 完整的框架架构和使用模式
- [配置参考](/zh/guide/docs/configuration-reference.md) - 全面的配置文档
- [服务开发指南](/zh/guide/docs/service-development-guide.md) - 如何使用框架构建服务

### 框架组件文档
- [基线服务器框架](/zh/guide/pkg/server/CLAUDE.md) - 核心服务器接口和实现模式
- [传输层](/zh/guide/pkg/transport/CLAUDE.md) - HTTP 和 gRPC 传输协调
- [服务架构分析](/zh/guide/docs/service-architecture-analysis.md) - 框架设计原则

### 示例服务文档
- [示例服务概览](/zh/guide/examples/README.md) - 所有框架示例的指南
- [简单 HTTP 服务](/zh/guide/examples/simple-http-service/README.md) - 基本框架使用
- [gRPC 服务示例](/zh/guide/examples/grpc-service/README.md) - gRPC 集成模式
- [全功能服务](/zh/guide/examples/full-featured-service/README.md) - 完整框架展示

### 参考服务文档
- [SwitServe 服务](/zh/guide/docs/services/switserve/README.md) - 用户管理服务实现
- [SwitAuth 服务](/zh/guide/docs/services/switauth/README.md) - 身份验证服务模式

### API 和协议文档
- [Protocol Buffer 定义](https://github.com/innovationmech/swit/blob/master/api/proto/) - 源 API 规格
- 生成的 Swagger 参考 (`docs/generated/`，通过 `make swagger`)

### 开发和贡献
- [开发指南](/zh/guide/DEVELOPMENT.md) - 框架开发环境设置
- [行为准则](/zh/guide/CODE_OF_CONDUCT.md) - 社区指南
- [安全策略](/zh/guide/SECURITY.md) - 安全实践和报告

## 贡献

我们欢迎对 Swit 微服务框架的贡献！无论您是修复错误、改进文档、添加示例，还是增强框架功能，您的贡献都很有价值。

### 贡献方式

1. **框架核心开发** - 增强 `pkg/server/` 和 `pkg/transport/` 组件
2. **示例服务** - 在 `examples/` 目录中添加新示例
3. **文档** - 改进框架文档和指南
4. **测试** - 为框架组件和示例添加测试
5. **错误报告** - 报告框架功能问题
6. **功能请求** - 建议新的框架功能

### 入门指南

1. Fork 仓库并克隆您的 fork
2. 设置开发环境：`make setup-dev`
3. 运行测试以确保一切正常工作：`make test`
4. 按照现有模式进行更改
5. 为新功能添加测试
6. 提交带有清晰描述的拉取请求

请在贡献前阅读我们的 [行为准则](/zh/guide/CODE_OF_CONDUCT.md) 以确保为所有社区成员提供积极和包容的环境。

## 许可证

MIT 许可证 - 详见 [LICENSE](https://github.com/innovationmech/swit/blob/master/LICENSE) 文件
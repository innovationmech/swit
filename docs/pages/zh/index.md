---
layout: home

title: Swit
titleTemplate: 现代化 Go 微服务框架

hero:
  name: Swit
  text: 现代化 Go 微服务框架
  tagline: 为构建可扩展微服务提供统一的、生产就绪的基础框架
  actions:
    - theme: brand
      text: 快速开始
      link: /zh/guide/getting-started
    - theme: alt
      text: 查看示例
      link: /zh/examples/
    - theme: alt
      text: API 文档
      link: /zh/api/
  image:
    src: /images/hero-logo.svg
    alt: Swit 框架 Logo

features:
  - icon: 🚀
    title: 快速开发
    details: 提供完整的服务器框架和统一传输层，支持 HTTP 和 gRPC 协议，让您专注于业务逻辑
    link: /zh/guide/getting-started

  - icon: ⚡
    title: 高性能
    details: 内置性能监控、优化的并发处理和资源管理，确保生产环境的高性能表现
    link: /zh/guide/performance

  - icon: 🔧
    title: 依赖注入
    details: 工厂模式的依赖容器，支持单例和瞬态依赖，自动生命周期管理
    link: /zh/guide/dependency-injection

  - icon: 🌐
    title: 服务发现
    details: 基于 Consul 的服务注册与发现，支持健康检查和自动注销
    link: /zh/guide/service-discovery

  - icon: 🛡️
    title: 中间件框架
    details: 可配置的中间件堆栈，支持 CORS、限流、超时和身份验证
    link: /zh/guide/middleware

  - icon: 📊
    title: 监控和健康检查
    details: 综合健康监控系统，服务聚合和超时处理，内置性能指标收集
    link: /zh/guide/monitoring

---

## 快速体验

使用 Swit 框架创建您的第一个微服务只需几分钟：

```go
// main.go
package main

import (
    "context"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/server"
)

type MyService struct{}

func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
    return registry.RegisterBusinessHTTPHandler(&MyHTTPHandler{})
}

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
    c.JSON(http.StatusOK, gin.H{"message": "Hello from Swit framework!"})
}

func main() {
    config := &server.ServerConfig{
        ServiceName: "my-service",
        HTTP: server.HTTPConfig{Port: "8080", Enabled: true},
    }
    
    service := &MyService{}
    baseServer, _ := server.NewBusinessServerCore(config, service, nil)
    
    ctx := context.Background()
    baseServer.Start(ctx)
    defer baseServer.Shutdown()
    
    select {} // 保持运行
}
```

```bash
go run main.go
curl http://localhost:8080/hello
```

## 框架特性

### 核心服务器框架

- **BusinessServerCore**: 完整的服务器生命周期管理
- **BusinessServiceRegistrar**: 统一的服务注册模式
- **BusinessDependencyContainer**: 依赖注入和生命周期管理

### 统一传输层

- **TransportCoordinator**: HTTP 和 gRPC 传输协调器
- **NetworkTransport**: 可插拔的传输架构
- **MultiTransportRegistry**: 跨传输服务注册管理

### 完整示例服务

- **switserve**: 用户管理服务，展示完整框架集成
- **switauth**: 身份验证服务，JWT 集成和安全模式
- **switctl**: 命令行工具，框架集成模式

## 开发工具和命令

### 框架开发命令
```bash
make build          # 构建框架组件和示例服务
make test           # 运行完整测试套件
make proto          # 生成 Protocol Buffer 代码
make swagger        # 生成 Swagger 文档
```

### 快速开始命令
```bash
make setup-dev      # 设置完整开发环境
make build-dev      # 快速构建（跳过质量检查）
make test-dev       # 快速测试
```

## 项目统计

<div class="stats">
  <div class="stat">
    <div class="stat-number">{{ $frontmatter.stats?.stars || '⭐' }}</div>
    <div class="stat-label">GitHub Stars</div>
  </div>
  <div class="stat">
    <div class="stat-number">{{ $frontmatter.stats?.version || 'v1.0.0' }}</div>
    <div class="stat-label">最新版本</div>
  </div>
  <div class="stat">
    <div class="stat-number">MIT</div>
    <div class="stat-label">开源许可证</div>
  </div>
  <div class="stat">
    <div class="stat-number">Go 1.24+</div>
    <div class="stat-label">语言版本</div>
  </div>
</div>

## 为什么选择 Swit？

### 🎯 专为生产而设计
Swit 框架基于真实项目经验构建，提供生产环境所需的所有功能，包括监控、健康检查、优雅关闭和错误处理。

### 🔌 插拔式架构
灵活的接口设计允许您轻松扩展功能，更换传输层或添加自定义中间件，而无需修改核心框架。

### 📈 性能优先
内置性能监控和优化，支持高并发场景，提供详细的性能指标和瓶颈分析。

### 👥 开发者友好
完整的示例代码、详细的文档和活跃的社区支持，让您快速上手并持续改进。

## 社区和支持

- **GitHub 仓库**: [innovationmech/swit](https://github.com/innovationmech/swit)
- **问题报告**: [GitHub Issues](https://github.com/innovationmech/swit/issues)
- **贡献指南**: [如何贡献](https://github.com/innovationmech/swit/blob/master/CONTRIBUTING.md)
- **行为准则**: [社区规范](https://github.com/innovationmech/swit/blob/master/CODE_OF_CONDUCT.md)

立即开始您的微服务开发之旅！

<style>
.stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 1rem;
  margin: 2rem 0;
}

.stat {
  text-align: center;
  padding: 1rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  background: var(--vp-c-bg-soft);
}

.stat-number {
  font-size: 2rem;
  font-weight: bold;
  color: var(--vp-c-brand-1);
}

.stat-label {
  font-size: 0.875rem;
  color: var(--vp-c-text-2);
  margin-top: 0.5rem;
}

@media (max-width: 768px) {
  .stats {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
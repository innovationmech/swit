# Swit

[![CI](https://github.com/innovationmech/swit/workflows/CI/badge.svg)](https://github.com/innovationmech/swit/actions/workflows/ci.yml)
[![Test Suite](https://github.com/innovationmech/swit/workflows/Test%20Suite/badge.svg)](https://github.com/innovationmech/swit/actions/workflows/test.yml)
[![Security Checks](https://github.com/innovationmech/swit/workflows/Security%20Checks/badge.svg)](https://github.com/innovationmech/swit/actions/workflows/security-checks.yml)
[![codecov](https://codecov.io/gh/innovationmech/swit/branch/master/graph/badge.svg)](https://codecov.io/gh/innovationmech/swit)
[![Go Report Card](https://goreportcard.com/badge/github.com/innovationmech/swit)](https://goreportcard.com/report/github.com/innovationmech/swit)
[![Go Reference](https://pkg.go.dev/badge/github.com/innovationmech/swit.svg)](https://pkg.go.dev/github.com/innovationmech/swit)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.23.12-blue.svg)
[![GitHub release](https://img.shields.io/github/release/innovationmech/swit.svg)](https://github.com/innovationmech/swit/releases)
[![License](https://img.shields.io/github/license/innovationmech/swit.svg)](LICENSE)

全面的 Go 微服务框架，为构建可扩展微服务提供统一、生产就绪的基础。

📖 **[完整文档](https://innovationmech.github.io/swit/zh/)** | [English Docs](https://innovationmech.github.io/swit/)

## 主要特性

- 🚀 **完整服务器框架**: 统一的 HTTP 和 gRPC 传输协调
- 💉 **依赖注入**: 基于工厂的容器，支持自动生命周期管理
- 📊 **性能监控**: 内置指标收集和监控钩子
- 🔍 **服务发现**: 基于 Consul 的注册，支持健康检查集成
- 🛡️ **中间件堆栈**: 可配置的 CORS、速率限制、身份验证和超时
- ⚡ **Protocol Buffers**: 完整的 Buf 工具链支持 API 开发
- 📱 **示例服务**: 完整的参考实现和使用模式

## 架构概览

### 核心组件
- **`pkg/server/`** - 基础服务器框架和生命周期管理
- **`pkg/transport/`** - HTTP/gRPC 传输协调层
- **`pkg/middleware/`** - 可配置的中间件堆栈
- **`pkg/discovery/`** - 服务发现集成

### 示例服务
- **`examples/`** - 简单的入门示例
- **`internal/switserve/`** - 用户管理参考服务
- **`internal/switauth/`** - 带 JWT 的身份验证服务
- **`internal/switctl/`** - CLI 工具实现

## 环境要求

- **Go 1.23.12+** 支持泛型
- **可选**: MySQL 8.0+、Redis 6.0+、Consul 1.12+（用于示例）
- **开发工具**: Buf CLI、Docker、Make

## 快速开始

### 1. 获取框架
```bash
git clone https://github.com/innovationmech/swit.git
cd swit
go mod download
```

### 2. 创建简单服务
```go
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
    ginRouter.GET("/hello", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "来自 Swit 的问候！"})
    })
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return "my-service"
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
    
    select {} // 保持运行
}
```

### 3. 运行服务
```bash
go run main.go
curl http://localhost:8080/hello
```

## 示例

### 简单示例
```bash
# HTTP 服务
cd examples/simple-http-service && go run main.go

# gRPC 服务
cd examples/grpc-service && go run main.go

# 全功能服务
cd examples/full-featured-service && go run main.go
```

### 参考服务
```bash
# 构建所有服务
make build

# 运行服务
./bin/swit-serve    # 用户管理 (HTTP: 9000, gRPC: 10000)
./bin/swit-auth     # 身份验证 (HTTP: 9001, gRPC: 50051)
./bin/switctl --help # CLI 工具
./bin/saga-migrate --help # 数据库迁移工具
```

### 数据库迁移（Saga 存储）

`saga-migrate` 工具用于管理 Saga 数据库 schema 迁移：

```bash
# 应用所有迁移
saga-migrate -dsn 'postgres://localhost/saga' -action migrate

# 检查迁移状态
saga-migrate -dsn 'postgres://localhost/saga' -action status

# 应用特定版本
saga-migrate -dsn 'postgres://localhost/saga' -action apply -version 2

# 回滚迁移
saga-migrate -dsn 'postgres://localhost/saga' -action rollback -version 2

# 验证 schema 版本
saga-migrate -dsn 'postgres://localhost/saga' -action validate -version 2

# 查看完整文档
cat docs/saga-database-migrations.md
```

## 开发

### 设置开发环境
```bash
# 完整设置
make setup-dev

# 快速设置
make setup-quick
```

### 常用命令
```bash
# 构建
make build          # 完整构建
make build-dev      # 快速构建

# 测试
make test           # 所有测试
make test-dev       # 快速测试
make test-coverage  # 覆盖率报告

# API 开发
make proto          # 生成 protobuf 代码
make swagger        # 生成 API 文档

# 代码质量
make tidy           # 整理模块
make format         # 格式化代码
make quality        # 质量检查
```

## Docker 部署

```bash
# 构建镜像
make docker

# 使用 Docker Compose 运行
docker-compose up -d

# 或单独运行
docker run -p 9000:9000 -p 10000:10000 swit-serve:latest
docker run -p 9001:9001 swit-auth:latest
```

## 贡献

1. Fork 仓库
2. 设置开发环境：`make setup-dev`
3. 运行测试：`make test`
4. 按照现有模式进行更改
5. 为新功能添加测试
6. 提交拉取请求

请在贡献前阅读我们的[行为准则](CODE_OF_CONDUCT.md)。

## 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

---

**完整的文档、示例和高级用法，请访问 [innovationmech.github.io/swit/zh](https://innovationmech.github.io/swit/zh/)**

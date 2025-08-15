# 快速开始

欢迎使用 Swit Go 微服务框架！本指南将帮助您快速上手并创建您的第一个微服务。

## 安装

首先，确保您已经安装了 Go 1.19 或更高版本。

```bash
go get github.com/innovationmech/swit
```

## 创建您的第一个服务

创建一个新的 Go 项目并初始化模块：

```bash
mkdir my-service
cd my-service
go mod init my-service
```

## 基本示例

创建一个简单的 HTTP 服务：

```go
package main

import (
    "context"
    "log"
    
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/server"
)

type MyService struct {
    // 服务字段
}

func (s *MyService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // 注册 HTTP 处理程序
    registry.RegisterBusinessHTTPHandler(&MyHTTPHandler{})
    return nil
}

type MyHTTPHandler struct{}

func (h *MyHTTPHandler) RegisterRoutes(router interface{}) error {
    r := router.(*gin.Engine)
    r.GET("/health", h.Health)
    r.GET("/api/v1/hello", h.Hello)
    return nil
}

func (h *MyHTTPHandler) GetServiceName() string {
    return "my-service"
}

func (h *MyHTTPHandler) Health(c *gin.Context) {
    c.JSON(200, gin.H{"status": "healthy"})
}

func (h *MyHTTPHandler) Hello(c *gin.Context) {
    c.JSON(200, gin.H{"message": "Hello from Swit!"})
}

func main() {
    config := &server.ServerConfig{
        Name:    "my-service",
        Version: "1.0.0",
        HTTP: server.HTTPConfig{
            Port:     8080,
            Enabled:  true,
        },
    }
    
    service := &MyService{}
    
    srv, err := server.NewBusinessServerCore(config, service, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    if err := srv.Start(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## 运行服务

```bash
go run main.go
```

您的服务现在运行在 `http://localhost:8080`。您可以通过以下方式测试：

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/hello
```

## 下一步

- 了解如何[配置服务器](./server-config)
- 学习[创建 gRPC 服务](./grpc-service)
- 探索[中间件](./middleware)功能
- 查看[完整示例](/zh/examples/full-featured)
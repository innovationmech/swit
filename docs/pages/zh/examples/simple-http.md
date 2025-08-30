# 简单 HTTP 服务示例

此示例演示了如何使用 Swit 框架构建基本的纯 HTTP 服务。它展示了服务注册、HTTP 路由、健康检查和依赖注入等基本概念的直接实现。

## 概述

简单 HTTP 服务示例（`examples/simple-http-service/`）提供：
- **纯 HTTP 服务** - 专注于 HTTP REST API 功能
- **基本路由** - 具有 JSON 响应的基本 API 端点
- **健康监控** - 简单的健康检查实现
- **环境配置** - 环境变量支持
- **优雅关闭** - 适当的服务器生命周期管理

## 主要特性

### 服务架构
- 实现 `BusinessServiceRegistrar` 以实现框架集成
- 注册 HTTP 处理程序和健康检查
- 使用简单的依赖容器进行基本依赖管理
- 演示适当的服务生命周期（启动/停止/关闭）

### HTTP 端点
- `GET /api/v1/hello?name=您的姓名` - 带查询参数的问候端点
- `GET /api/v1/status` - 服务状态和健康信息
- `POST /api/v1/echo` - 返回请求数据的回显端点

### 配置特性
- 支持 HTTP 端口的环境变量
- 可选的服务发现集成
- CORS 和日志中间件
- 可配置的超时和服务器设置

## 代码结构

### 主服务实现

```go
// SimpleHTTPService 实现 ServiceRegistrar 接口
type SimpleHTTPService struct {
    name string
}

func (s *SimpleHTTPService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // 注册 HTTP 处理程序
    httpHandler := &SimpleHTTPHandler{serviceName: s.name}
    if err := registry.RegisterBusinessHTTPHandler(httpHandler); err != nil {
        return fmt.Errorf("注册 HTTP 处理程序失败: %w", err)
    }

    // 注册健康检查
    healthCheck := &SimpleHealthCheck{serviceName: s.name}
    if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
        return fmt.Errorf("注册健康检查失败: %w", err)
    }

    return nil
}
```

### HTTP 处理程序实现

```go
type SimpleHTTPHandler struct {
    serviceName string
}

func (h *SimpleHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(gin.IRouter)

    api := ginRouter.Group("/api/v1")
    {
        api.GET("/hello", h.handleHello)
        api.GET("/status", h.handleStatus)
        api.POST("/echo", h.handleEcho)
    }

    return nil
}
```

### 配置设置

```go
config := &server.ServerConfig{
    ServiceName: "simple-http-service",
    HTTP: server.HTTPConfig{
        Port:         getEnv("HTTP_PORT", "8080"),
        EnableReady:  true,
        Enabled:      true,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  60 * time.Second,
    },
    GRPC: server.GRPCConfig{
        Enabled: false, // 仅 HTTP 服务
    },
    Discovery: server.DiscoveryConfig{
        Enabled: getBoolEnv("DISCOVERY_ENABLED", false),
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
- 框架依赖可用

### 快速开始

1. **导航到示例目录：**
   ```bash
   cd examples/simple-http-service
   ```

2. **运行服务：**
   ```bash
   go run main.go
   ```

3. **测试端点：**
   ```bash
   # 问候端点
   curl "http://localhost:8080/api/v1/hello?name=Alice"
   
   # 状态端点  
   curl "http://localhost:8080/api/v1/status"
   
   # 回显端点
   curl -X POST "http://localhost:8080/api/v1/echo" \
        -H "Content-Type: application/json" \
        -d '{"message": "Hello, World!"}'
   ```

### 环境配置

使用环境变量配置服务：

```bash
# 设置自定义 HTTP 端口
export HTTP_PORT=9000

# 启用服务发现
export DISCOVERY_ENABLED=true
export CONSUL_ADDRESS=localhost:8500

# 使用自定义配置运行
go run main.go
```

### 预期响应

**Hello 端点：**
```json
{
  "message": "Hello, Alice!",
  "service": "simple-http-service",
  "timestamp": "2025-01-20T10:30:00Z"
}
```

**状态端点：**
```json
{
  "status": "healthy",
  "service": "simple-http-service", 
  "timestamp": "2025-01-20T10:30:00Z",
  "uptime": "running"
}
```

**回显端点：**
```json
{
  "echo": "Hello, World!",
  "service": "simple-http-service",
  "timestamp": "2025-01-20T10:30:00Z"
}
```

## 开发模式

### 添加新端点

1. **添加处理方法：**
   ```go
   func (h *SimpleHTTPHandler) handleNewEndpoint(c *gin.Context) {
       c.JSON(http.StatusOK, gin.H{
           "message": "新端点响应",
           "service": h.serviceName,
       })
   }
   ```

2. **在路由中注册：**
   ```go
   func (h *SimpleHTTPHandler) RegisterRoutes(router interface{}) error {
       ginRouter := router.(gin.IRouter)
       api := ginRouter.Group("/api/v1")
       {
           api.GET("/hello", h.handleHello)
           api.GET("/status", h.handleStatus) 
           api.POST("/echo", h.handleEcho)
           api.GET("/new", h.handleNewEndpoint) // 添加新路由
       }
       return nil
   }
   ```

### 错误处理模式

```go
func (h *SimpleHTTPHandler) handleWithValidation(c *gin.Context) {
    var request struct {
        Field string `json:"field" binding:"required"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "无效请求",
            "details": err.Error(),
        })
        return
    }

    // 处理有效请求...
    c.JSON(http.StatusOK, gin.H{"success": true})
}
```

### 依赖集成

```go
// 扩展依赖容器
deps := NewSimpleDependencyContainer()
deps.AddService("config", config)
deps.AddService("version", "1.0.0")
deps.AddService("database", databaseConnection)

// 在处理程序中使用
func (h *SimpleHTTPHandler) handleWithDependency(c *gin.Context) {
    db, err := h.deps.GetService("database")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库不可用"})
        return
    }
    
    // 使用数据库...
}
```

## 演示的最佳实践

1. **清洁架构** - 服务、处理程序和依赖之间的关注点分离
2. **错误处理** - 适当的 HTTP 状态码和错误响应
3. **配置管理** - 环境变量支持和合理的默认值
4. **健康监控** - 实现有意义的健康检查
5. **优雅关闭** - 服务终止时的适当清理

## 下一步

在理解这个简单示例后：

1. **探索 gRPC 集成** - 查看 `grpc-service` 示例了解 Protocol Buffer 集成
2. **多传输支持** - 查看 `full-featured-service` 了解 HTTP + gRPC 组合
3. **生产功能** - 实现数据库连接、身份验证、指标
4. **高级配置** - 添加 YAML 配置文件和特定环境设置

此示例为使用 Swit 框架构建基于 HTTP 的微服务提供了坚实的基础，同时演示了基本模式和最佳实践。
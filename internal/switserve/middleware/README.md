# Timeout Middleware

这个包提供了用于 Gin 框架的超时中间件，为整个请求设置超时控制。

## 功能特性

- 🚀 **请求超时控制**: 为HTTP请求设置全局或自定义超时时间
- 🔧 **灵活配置**: 支持自定义超时时间、错误消息和处理器
- 🛤️ **路径跳过**: 可以配置跳过超时检查的特定路径
- 🎯 **Context集成**: 将超时上下文传递给下游服务
- 🔒 **Panic恢复**: 优雅处理处理器中的panic
- 📝 **注册器模式**: 支持中间件注册器模式集成

## 快速开始

### 基本用法

```go
package main

import (
    "time"
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/internal/switserve/middleware"
)

func main() {
    router := gin.New()
    
    // 添加30秒超时中间件
    router.Use(middleware.TimeoutMiddleware(30 * time.Second))
    
    router.GET("/api/data", func(c *gin.Context) {
        // 模拟长时间处理
        time.Sleep(2 * time.Second)
        c.JSON(200, gin.H{"message": "success"})
    })
    
    router.Run(":8080")
}
```

### 自定义配置

```go
// 使用自定义配置
config := middleware.TimeoutConfig{
    Timeout:      15 * time.Second,
    ErrorMessage: "请求处理超时",
    SkipPaths:    []string{"/health", "/metrics"},
    Handler: func(c *gin.Context) {
        c.JSON(408, gin.H{
            "error": "自定义超时响应",
            "timestamp": time.Now().Unix(),
        })
    },
}

router.Use(middleware.TimeoutWithConfig(config))
```

### Context超时中间件

```go
// 为下游服务传递超时上下文
router.Use(middleware.ContextTimeoutMiddleware(10 * time.Second))

router.GET("/api/database", func(c *gin.Context) {
    ctx := c.Request.Context() // 这个context带有超时设置
    
    // 将context传递给数据库调用
    result, err := database.QueryWithContext(ctx, "SELECT * FROM users")
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            c.JSON(408, gin.H{"error": "数据库查询超时"})
            return
        }
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, result)
})
```

## 高级用法

### 不同路由组的不同超时

```go
router := gin.New()

// API路由 - 短超时
api := router.Group("/api/v1")
api.Use(middleware.TimeoutMiddleware(5 * time.Second))

// 文件上传路由 - 长超时
upload := router.Group("/upload")
upload.Use(middleware.TimeoutMiddleware(2 * time.Minute))

// 流式处理 - 无超时或很长超时
stream := router.Group("/stream")
stream.Use(middleware.TimeoutMiddleware(10 * time.Minute))
```

### 使用注册器模式

```go
// 创建超时中间件注册器
timeoutRegistrar := middleware.NewTimeoutRegistrar(middleware.TimeoutConfig{
    Timeout:   20 * time.Second,
    SkipPaths: []string{"/health", "/ready", "/metrics"},
})

// 注册到路由器
err := timeoutRegistrar.RegisterMiddleware(router)
if err != nil {
    log.Fatal("Failed to register timeout middleware:", err)
}
```

### 处理长时间运行的操作

```go
router.GET("/api/process", func(c *gin.Context) {
    ctx := c.Request.Context()
    
    resultChan := make(chan string, 1)
    errorChan := make(chan error, 1)
    
    // 在goroutine中执行长时间操作
    go func() {
        // 模拟处理
        time.Sleep(5 * time.Second)
        resultChan <- "处理完成"
    }()
    
    select {
    case result := <-resultChan:
        c.JSON(200, gin.H{"result": result})
    case err := <-errorChan:
        c.JSON(500, gin.H{"error": err.Error()})
    case <-ctx.Done():
        c.JSON(408, gin.H{"error": "操作超时"})
    }
})
```

## 配置选项

### TimeoutConfig

```go
type TimeoutConfig struct {
    // 超时时间
    Timeout time.Duration
    
    // 自定义错误消息
    ErrorMessage string
    
    // 自定义超时处理器
    Handler gin.HandlerFunc
    
    // 跳过超时检查的路径
    SkipPaths []string
}
```

### 默认配置

```go
config := middleware.DefaultTimeoutConfig()
// Timeout: 30 * time.Second
// ErrorMessage: "Request timeout"
// SkipPaths: []string{"/health", "/metrics"}
```

## 最佳实践

### 1. 超时时间设置

- **API接口**: 5-30秒
- **文件上传**: 1-5分钟
- **数据导出**: 2-10分钟
- **流式处理**: 10分钟以上或无限制

### 2. 路径跳过

建议跳过以下路径的超时检查：
- 健康检查: `/health`, `/ready`
- 监控指标: `/metrics`, `/prometheus`
- 调试接口: `/debug/*`
- 管理接口: `/admin/stop`

### 3. 错误处理

```go
// 推荐的错误响应格式
handler := func(c *gin.Context) {
    c.JSON(http.StatusRequestTimeout, gin.H{
        "error":     "request_timeout",
        "message":   "请求处理时间过长",
        "timestamp": time.Now().Unix(),
        "path":      c.Request.URL.Path,
        "method":    c.Request.Method,
    })
}
```

### 4. 与认证中间件的集成

```go
// 超时中间件应该在认证中间件之前
router.Use(middleware.TimeoutMiddleware(30 * time.Second))
router.Use(middleware.AuthMiddleware())
```

## 性能考虑

- 超时中间件使用goroutine处理请求，会有少量性能开销
- 建议在生产环境中进行性能测试以确定最佳超时时间
- 对于高并发场景，考虑使用连接池和请求限流

## 监控和调试

### 日志记录

```go
config := middleware.TimeoutConfig{
    Timeout: 30 * time.Second,
    Handler: func(c *gin.Context) {
        // 记录超时事件
        log.Printf("Request timeout: %s %s from %s", 
            c.Request.Method, c.Request.URL.Path, c.ClientIP())
        
        c.JSON(408, gin.H{"error": "request timeout"})
    },
}
```

### 指标收集

可以在超时处理器中收集指标：
- 超时请求数量
- 超时的路径分布
- 平均请求处理时间

## 故障排除

### 常见问题

1. **请求总是超时**
   - 检查超时时间设置是否过短
   - 确认处理器逻辑是否有死循环或阻塞

2. **某些路径不应该有超时**
   - 将路径添加到 `SkipPaths` 配置中

3. **超时响应格式不符合预期**
   - 使用自定义的 `Handler` 配置

4. **Context传递问题**
   - 确保使用 `c.Request.Context()` 获取带超时的上下文
   - 将上下文传递给所有下游调用

### 调试技巧

```go
// 添加调试日志
router.Use(func(c *gin.Context) {
    start := time.Now()
    c.Next()
    duration := time.Since(start)
    log.Printf("Request %s %s took %v", c.Request.Method, c.Request.URL.Path, duration)
})
```
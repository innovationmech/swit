# 错误监控和性能跟踪

Swit 框架通过 Sentry 集成提供了全面的错误监控和性能跟踪功能。本指南涵盖了从基本设置到高级配置和故障排除的所有内容。

## 概述

Swit 的监控系统包括：

- **自动错误捕获**：自动捕获 HTTP 和 gRPC 错误，提供完整上下文
- **性能监控**：请求跟踪、事务监控和性能指标
- **智能过滤**：可配置的错误过滤以减少噪音（默认忽略 4xx 错误）
- **上下文丰富**：请求元数据、用户上下文和自定义标签在错误报告中
- **恐慌恢复**：自动恐慌捕获和恢复，提供详细堆栈跟踪
- **零配置默认值**：开箱即用的生产级默认设置

## 快速开始

### 1. 基本配置

在服务配置文件中添加 Sentry 配置：

```yaml
# swit.yaml
service_name: "my-service"

sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"  # 通过环境变量设置
  environment: "production"
  sample_rate: 1.0
  traces_sample_rate: 0.1
```

### 2. 环境设置

设置 Sentry DSN：

```bash
export SENTRY_DSN="https://your-dsn@sentry.io/your-project-id"
```

### 3. 框架集成

框架自动处理 Sentry：

```go
package main

import (
    "context"
    "log"
    "github.com/innovationmech/swit/pkg/server"
)

func main() {
    config := server.NewServerConfig()
    config.ServiceName = "my-service"
    
    // 如果在配置中启用，Sentry 将自动初始化
    baseServer, err := server.NewBusinessServerCore(config, myService, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // 启动服务器 - Sentry 监控自动开始
    baseServer.Start(context.Background())
}
```

就这样！您的服务现在拥有全面的错误监控和性能跟踪。

## 高级配置

### 完整配置参考

```yaml
sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "production"
  release: "v1.2.3"
  sample_rate: 1.0              # 错误采样率 (0.0-1.0)
  traces_sample_rate: 0.1       # 性能采样率
  attach_stacktrace: true
  enable_tracing: true
  debug: false                  # 启用以进行故障排除
  server_name: "my-server-01"
  
  # 添加到所有事件的自定义标签
  tags:
    service: "user-management"
    version: "1.2.3"
    datacenter: "us-west"
  
  # 框架集成设置
  integrate_http: true          # 启用 HTTP 中间件
  integrate_grpc: true          # 启用 gRPC 中间件
  capture_panics: true          # 捕获和恢复恐慌
  max_breadcrumbs: 30          # 最大面包屑轨迹长度
  
  # 错误过滤（减少噪音）
  ignore_errors:
    - "connection timeout"
    - "user not found"
  
  # HTTP 特定过滤
  http_ignore_paths:
    - "/health"
    - "/metrics"
    - "/favicon.ico"
  
  # HTTP 状态码过滤
  http_ignore_status_codes:
    - 404    # 未找到错误
    - 400    # 错误请求
  
  # 性能监控
  enable_profiling: true
  profiles_sample_rate: 0.1
  
  # 上下文和面包屑
  max_request_body_size: 1024   # 捕获的最大请求体（字节）
  send_default_pii: false       # 不发送个人识别信息
```

### 环境特定配置

不同环境通常需要不同的设置：

#### 开发环境
```yaml
sentry:
  enabled: true
  debug: true
  sample_rate: 1.0
  traces_sample_rate: 1.0       # 在开发中捕获所有跟踪
  environment: "development"
```

#### 预发环境
```yaml
sentry:
  enabled: true
  sample_rate: 1.0
  traces_sample_rate: 0.5       # 50% 性能采样
  environment: "staging"
```

#### 生产环境
```yaml
sentry:
  enabled: true
  sample_rate: 1.0              # 捕获所有错误
  traces_sample_rate: 0.1       # 10% 性能采样
  environment: "production"
  enable_profiling: true
```

## 性能监控

### 事务跟踪

框架自动为以下内容创建事务：

- **HTTP 请求**：每个 HTTP 请求成为一个事务
- **gRPC 调用**：跟踪每个 gRPC 方法调用
- **自定义操作**：您可以创建自定义事务

自定义事务示例：

```go
import "github.com/getsentry/sentry-go"

func processOrder(orderID string) error {
    // 创建自定义事务
    transaction := sentry.StartTransaction(
        context.Background(), 
        "process-order",
    )
    defer transaction.Finish()
    
    // 添加自定义数据
    transaction.SetTag("order_id", orderID)
    transaction.SetData("operation", "order_processing")
    
    // 您的业务逻辑在这里
    if err := validateOrder(orderID); err != nil {
        transaction.SetStatus(sentry.SpanStatusInvalidArgument)
        return err
    }
    
    return nil
}
```

### 自定义性能跨度

创建详细的性能跨度：

```go
func (h *OrderHandler) CreateOrder(c *gin.Context) {
    span := sentry.StartSpan(c.Request.Context(), "database.query")
    span.SetTag("table", "orders")
    defer span.Finish()
    
    // 数据库操作
    order, err := h.db.CreateOrder(order)
    if err != nil {
        span.SetStatus(sentry.SpanStatusInternalError)
        sentry.CaptureException(err)
        c.JSON(500, gin.H{"error": "创建订单失败"})
        return
    }
    
    span.SetData("order_id", order.ID)
    c.JSON(201, order)
}
```

## 错误处理最佳实践

### 自定义错误上下文

为错误添加丰富的上下文：

```go
func (s *UserService) GetUser(id string) (*User, error) {
    sentry.ConfigureScope(func(scope *sentry.Scope) {
        scope.SetTag("service", "user-service")
        scope.SetContext("user_lookup", map[string]interface{}{
            "user_id": id,
            "timestamp": time.Now(),
        })
    })
    
    user, err := s.repository.FindByID(id)
    if err != nil {
        // 此错误将包含上述上下文
        sentry.CaptureException(err)
        return nil, err
    }
    
    return user, nil
}
```

### 恐慌恢复

框架自动从恐慌中恢复，但您也可以手动处理：

```go
func riskyOperation() {
    defer func() {
        if err := recover(); err != nil {
            // 在报告之前添加自定义上下文
            sentry.WithScope(func(scope *sentry.Scope) {
                scope.SetLevel(sentry.LevelFatal)
                scope.SetContext("panic_context", map[string]interface{}{
                    "operation": "risky_operation",
                    "timestamp": time.Now(),
                })
                sentry.CaptureException(fmt.Errorf("panic: %v", err))
            })
            // 如果需要，重新恐慌
            panic(err)
        }
    }()
    
    // 风险代码在这里
}
```

## 使用 Sentry 进行测试

### 测试的模拟配置

在测试中禁用 Sentry：

```go
func TestMyService(t *testing.T) {
    config := &server.ServerConfig{
        ServiceName: "test-service",
        Sentry: server.SentryConfig{
            Enabled: false, // 为测试禁用
        },
    }
    
    // 您的测试代码在这里
}
```

### 集成测试

使用模拟 DSN 测试 Sentry 集成：

```go
func TestSentryIntegration(t *testing.T) {
    config := &server.ServerConfig{
        ServiceName: "test-service",
        Sentry: server.SentryConfig{
            Enabled: true,
            DSN: "http://public@example.com/1", // 模拟 DSN
            Debug: true,
        },
    }
    
    server, err := server.NewBusinessServerCore(config, service, nil)
    assert.NoError(t, err)
    
    // 测试错误捕获
    err = errors.New("test error")
    sentry.CaptureException(err)
    
    // 为测试刷新事件
    sentry.Flush(time.Second * 2)
}
```

## 故障排除

### 常见问题

#### 1. 事件未出现在 Sentry 中

**检查 DSN 配置：**
```bash
# 验证 DSN 是否设置
echo $SENTRY_DSN

# 测试 DSN 格式
curl -X POST "${SENTRY_DSN%/*}/api/${SENTRY_DSN##*/}/store/" \
  -H "Content-Type: application/json" \
  -d '{"message":"test"}'
```

**启用调试模式：**
```yaml
sentry:
  debug: true  # 启用调试日志
```

#### 2. 事件过多

**调整采样率：**
```yaml
sentry:
  sample_rate: 0.1        # 减少错误采样
  traces_sample_rate: 0.05 # 减少性能采样
```

**添加更多过滤器：**
```yaml
sentry:
  ignore_errors:
    - "timeout"
    - "connection refused"
  http_ignore_status_codes:
    - 404
    - 400
    - 401
```

#### 3. 缺少上下文

**验证中间件注册：**
```go
// 框架自动处理这个，但在日志中验证
log.Info("HTTP Sentry 中间件已注册")
log.Info("gRPC Sentry 中间件已注册")
```

### 调试信息

启用调试模式查看 Sentry 在做什么：

```yaml
sentry:
  debug: true
```

这将记录：
- 事件捕获尝试
- 采样决策
- 传输问题
- 配置问题

### Prometheus 指标

#### 默认采集器

自 v1.0.0 起，框架默认注册以下 Prometheus 采集器并暴露到 `/metrics` 端点：

- Go 运行时采集器：提供 `go_*` 指标（GC、goroutine、内存等）
- 进程采集器：提供 `process_*` 指标（CPU、内存、FD 等）
- 构建信息采集器：提供 `go_build_info` 指标

无需额外配置，服务启动后即可通过 `/metrics` 获取上述指标。

## 最佳实践

### 1. 生产配置

- 对敏感数据使用环境变量
- 设置适当的采样率
- 启用错误过滤
- 配置有意义的标签和上下文

### 2. 开发工作流

- 在开发期间使用调试模式
- 首先使用模拟 DSN 测试
- 在预发环境中验证错误捕获
- 监控性能影响

### 3. 错误管理

- 不捕获预期错误（4xx HTTP）
- 为错误添加有意义的上下文
- 使用适当的严重级别
- 实现适当的错误边界

### 4. 性能优化

- 在生产环境中使用低采样率
- 过滤掉健康检查和指标端点
- 监控 Sentry SDK 开销
- 尽可能使用异步传输

## 从自定义错误处理迁移

如果您正在从自定义错误处理迁移：

1. **保留现有日志**：Sentry 是补充，而不是替代日志
2. **渐进推出**：从低采样率开始
3. **上下文迁移**：将自定义上下文移动到 Sentry 标签/数据
4. **告警迁移**：逐渐将告警移动到 Sentry

迁移示例：

```go
// 之前：自定义错误处理
func (h *Handler) ProcessRequest(c *gin.Context) {
    if err := h.service.Process(); err != nil {
        h.logger.Error("Process failed", 
            zap.Error(err),
            zap.String("request_id", c.GetHeader("X-Request-ID")),
        )
        c.JSON(500, gin.H{"error": "内部错误"})
        return
    }
}

// 之后：使用 Sentry 集成
func (h *Handler) ProcessRequest(c *gin.Context) {
    if err := h.service.Process(); err != nil {
        // 保留现有日志
        h.logger.Error("Process failed", zap.Error(err))
        
        // 添加 Sentry 上下文
        sentry.WithScope(func(scope *sentry.Scope) {
            scope.SetTag("request_id", c.GetHeader("X-Request-ID"))
            scope.SetContext("request", map[string]interface{}{
                "method": c.Request.Method,
                "path":   c.Request.URL.Path,
            })
            sentry.CaptureException(err)
        })
        
        c.JSON(500, gin.H{"error": "内部错误"})
        return
    }
}
```

## 相关主题

- [配置指南](/zh/guide/configuration) - 服务器和服务配置
- [测试指南](/zh/guide/testing) - 测试策略和模式
- [性能指南](/zh/guide/performance) - 性能优化
- [故障排除](/zh/guide/troubleshooting) - 常见问题和解决方案

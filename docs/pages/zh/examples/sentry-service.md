# Sentry 监控示例服务

Sentry 示例服务演示了 Swit 框架与 Sentry 的全面错误监控和性能跟踪集成。本示例展示了生产环境监控的实际使用模式。

## 概述

**位置**: `examples/sentry-example-service/`

**特性**:
- 完整的 Sentry 集成设置
- 用于测试的错误生成端点
- 性能监控演示
- 自定义上下文和标签示例
- 恐慌恢复
- 不同错误类型和场景

## 快速开始

### 1. 前提条件

设置 Sentry DSN：

```bash
# 用于测试，您可以使用模拟 DSN
export SENTRY_DSN="https://public@sentry.example.com/1"

# 对于真实测试，使用您的 Sentry 项目 DSN
export SENTRY_DSN="https://your-dsn@sentry.io/your-project-id"
```

### 2. 运行服务

```bash
cd examples/sentry-example-service
go run main.go
```

服务在 8080 端口启动，提供以下端点。

## API 端点

### 健康和状态

| 端点 | 方法 | 描述 |
|----------|--------|-------------|
| `/health` | GET | 健康检查端点 |
| `/api/v1/status` | GET | 服务状态（带 Sentry 信息） |

### 错误生成（用于测试）

| 端点 | 方法 | 描述 |
|----------|--------|-------------|
| `/api/v1/error/500` | GET | 生成内部服务器错误 |
| `/api/v1/error/400` | GET | 生成错误请求（已过滤） |
| `/api/v1/error/custom` | POST | 生成带上下文的自定义错误 |
| `/api/v1/panic` | GET | 触发恐慌（框架恢复） |
| `/api/v1/timeout` | GET | 模拟超时错误 |

### 性能测试

| 端点 | 方法 | 描述 |
|----------|--------|-------------|
| `/api/v1/slow` | GET | 慢速端点，用于性能监控 |
| `/api/v1/database` | GET | 模拟数据库操作与跨度 |
| `/api/v1/external` | GET | 模拟外部 API 调用 |

### 数据处理

| 端点 | 方法 | 描述 |
|----------|--------|-------------|
| `/api/v1/process` | POST | 处理数据（自定义事务） |
| `/api/v1/batch` | POST | 批处理（性能跟踪） |

## 示例请求

### 1. 测试错误捕获

```bash
# 生成 500 错误（将被捕获）
curl http://localhost:8080/api/v1/error/500

# 生成 400 错误（默认过滤）
curl http://localhost:8080/api/v1/error/400

# 生成带上下文的自定义错误
curl -X POST http://localhost:8080/api/v1/error/custom \
  -H "Content-Type: application/json" \
  -d '{"user_id": "123", "operation": "payment", "amount": 100.50}'
```

### 2. 测试性能监控

```bash
# 慢速端点（将创建性能事务）
curl http://localhost:8080/api/v1/slow

# 数据库模拟（创建数据库跨度）
curl http://localhost:8080/api/v1/database?table=users&count=100

# 外部 API 模拟
curl http://localhost:8080/api/v1/external?service=payment-api
```

### 3. 测试恐慌恢复

```bash
# 触发恐慌（恢复并报告给 Sentry）
curl http://localhost:8080/api/v1/panic
```

## 配置示例

### 开发配置

```yaml
# examples/sentry-example-service/config/development.yaml
service_name: "sentry-example-dev"
http:
  port: "8080"

sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "development"
  debug: true                    # 启用调试日志
  sample_rate: 1.0              # 捕获所有错误
  traces_sample_rate: 1.0       # 捕获所有跟踪
  attach_stacktrace: true
  enable_tracing: true
  integrate_http: true
  capture_panics: true
  
  # 此示例的自定义标签
  tags:
    service_type: "example"
    version: "1.0.0"
    example: "sentry-demo"
```

### 生产配置

```yaml
# examples/sentry-example-service/config/production.yaml
service_name: "sentry-example"
http:
  port: "8080"

sentry:
  enabled: true
  dsn: "${SENTRY_DSN}"
  environment: "production"
  debug: false
  sample_rate: 1.0              # 在示例中捕获所有错误
  traces_sample_rate: 0.1       # 10% 性能采样
  attach_stacktrace: true
  enable_tracing: true
  enable_profiling: true
  
  # 生产过滤
  http_ignore_status_codes:
    - 400    # 过滤错误请求
    - 404    # 过滤未找到
  
  http_ignore_paths:
    - "/health"
    - "/favicon.ico"
  
  tags:
    service_type: "example"
    datacenter: "us-west"
    team: "platform"
```

## 实现示例

### 带上下文的错误处理器

```go
func (h *Handler) CustomErrorHandler(c *gin.Context) {
    var req ErrorRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "无效请求"})
        return
    }
    
    // 向 Sentry 添加自定义上下文
    sentry.ConfigureScope(func(scope *sentry.Scope) {
        scope.SetTag("operation", req.Operation)
        scope.SetContext("user_context", map[string]interface{}{
            "user_id": req.UserID,
            "amount":  req.Amount,
        })
        scope.SetLevel(sentry.LevelError)
    })
    
    // 模拟业务逻辑错误
    err := fmt.Errorf("用户 %s 的 %s 处理失败：余额不足", 
        req.UserID, req.Operation)
    
    // 此错误将包含上述上下文
    sentry.CaptureException(err)
    
    c.JSON(500, gin.H{
        "error": "处理失败",
        "code":  "PROCESSING_ERROR",
    })
}
```

### 性能跟踪

```go
func (h *Handler) DatabaseHandler(c *gin.Context) {
    // 开始自定义事务
    transaction := sentry.StartTransaction(
        c.Request.Context(),
        "database-operation",
    )
    defer transaction.Finish()
    
    // 添加事务元数据
    transaction.SetTag("table", c.Query("table"))
    transaction.SetData("count", c.Query("count"))
    
    // 用跨度模拟数据库查询
    dbSpan := transaction.StartChild("database.query")
    dbSpan.SetTag("operation", "SELECT")
    dbSpan.SetTag("table", c.Query("table"))
    
    // 模拟数据库工作
    time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
    
    // 向跨度添加结果
    dbSpan.SetData("rows_returned", 42)
    dbSpan.Finish()
    
    c.JSON(200, gin.H{
        "status": "success",
        "rows":   42,
        "table":  c.Query("table"),
    })
}
```

## 测试场景

### 1. 错误场景

测试不同类型的错误：

```bash
# 测试 HTTP 错误
for code in 400 401 403 404 500 502; do
    curl "http://localhost:8080/api/v1/error/$code"
done

# 测试带上下文的自定义错误
curl -X POST http://localhost:8080/api/v1/error/custom \
  -d '{"user_id":"test","operation":"payment","amount":50}'
```

### 2. 性能场景

测试性能监控：

```bash
# 测试不同性能模式
curl "http://localhost:8080/api/v1/slow?duration=500ms"
curl "http://localhost:8080/api/v1/database?table=orders&count=1000"
curl "http://localhost:8080/api/v1/external?service=user-service&timeout=2s"
```

### 3. 恐慌恢复

```bash
# 测试恐慌恢复
curl http://localhost:8080/api/v1/panic

# 应该返回 500 与恢复消息
# 恐慌详细信息发送到 Sentry
```

## 故障排除

### 常见问题

**事件未出现在 Sentry 中：**

1. 检查 DSN 配置
2. 验证网络连接
3. 启用调试模式
4. 检查采样率

**性能跟踪缺失：**

1. 验证 `traces_sample_rate` > 0
2. 检查是否启用跟踪
3. 确保性能中间件处于活动状态

**事件过多：**

1. 调整采样率
2. 添加更多过滤器
3. 过滤预期错误（4xx）

### 调试模式

启用调试日志：

```yaml
sentry:
  debug: true
```

这将记录：
- 事件捕获尝试
- 采样决策
- 传输状态
- 配置验证

## 下一步

1. **自定义配置**：根据您的需求调整配置
2. **添加自定义标签**：实现服务特定的标签
3. **性能调优**：调整采样率和过滤器
4. **集成测试**：使用真实的 Sentry 项目测试
5. **生产部署**：使用生产配置部署

## 相关示例

- [简单 HTTP 服务](/zh/examples/simple-http) - 基本 HTTP 服务模式
- [全功能服务](/zh/examples/full-featured) - 完整框架展示
- [gRPC 服务](/zh/examples/grpc-service) - gRPC 特定模式

## 相关指南

- [错误监控指南](/zh/guide/monitoring) - 完整 Sentry 集成指南
- [配置指南](/zh/guide/configuration) - 配置选项
- [性能指南](/zh/guide/performance) - 性能优化

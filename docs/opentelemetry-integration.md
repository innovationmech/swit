# OpenTelemetry 集成指南

本文档介绍 SWIT 框架中的 OpenTelemetry 集成，包括核心功能、配置选项和使用示例。

## 目录

1. [概述](#概述)
2. [核心功能](#核心功能)
3. [快速开始](#快速开始)
4. [配置选项](#配置选项)
5. [使用示例](#使用示例)
6. [中间件集成](#中间件集成)
7. [数据库追踪](#数据库追踪)
8. [最佳实践](#最佳实践)
9. [故障排查](#故障排查)

## 概述

SWIT 框架提供了完整的 OpenTelemetry 集成，支持分布式追踪、跨服务上下文传播和多种导出器。集成遵循 OpenTelemetry 规范，提供灵活的配置选项和开箱即用的中间件。

### 核心特性

- **多导出器支持**：Console、Jaeger、OTLP (HTTP/gRPC)
- **灵活采样策略**：Always On、Always Off、TraceID Ratio
- **自动上下文传播**：支持 TraceContext 和 Baggage 传播器
- **HTTP/gRPC 中间件**：自动为 HTTP 和 gRPC 请求创建 span
- **数据库追踪**：GORM 集成，自动追踪数据库操作
- **环境变量覆盖**：支持通过环境变量配置所有选项

## 核心功能

### 1. SDK 设置

核心 SDK 位于 `pkg/tracing` 包中，提供：

```go
type TracingManager interface {
    Initialize(ctx context.Context, config *TracingConfig) error
    StartSpan(ctx context.Context, operationName string, opts ...SpanOption) (context.Context, Span)
    SpanFromContext(ctx context.Context) Span
    InjectHTTPHeaders(ctx context.Context, headers http.Header)
    ExtractHTTPHeaders(headers http.Header) context.Context
    Shutdown(ctx context.Context) error
}
```

### 2. 导出器支持

支持三种导出器类型：

- **Console**: 调试用，输出到标准输出
- **Jaeger**: 通过 OTLP 协议导出到 Jaeger
- **OTLP**: 原生 OTLP HTTP/gRPC 导出

### 3. Span 处理器

- **BatchSpanProcessor**: 批量处理，适合生产环境
- **SimpleSpanProcessor**: 实时处理，适合开发调试

## 快速开始

### 基础配置

在服务配置文件中启用追踪：

```yaml
tracing:
  enabled: true
  service_name: "my-service"
  
  sampling:
    type: "traceidratio"
    rate: 1.0  # 100% 采样（开发环境）
  
  exporter:
    type: "jaeger"
    endpoint: "http://localhost:14268/api/traces"
    timeout: "10s"
  
  resource_attributes:
    environment: "development"
    version: "v1.0.0"
    service.namespace: "swit"
  
  propagators:
    - "tracecontext"
    - "baggage"
```

### 初始化追踪

```go
package main

import (
    "context"
    "log"
    
    "github.com/innovationmech/swit/pkg/tracing"
)

func main() {
    // 创建追踪管理器
    tm := tracing.NewTracingManager()
    
    // 加载配置
    config := loadTracingConfig() // 从配置文件加载
    
    // 初始化
    ctx := context.Background()
    if err := tm.Initialize(ctx, config); err != nil {
        log.Fatalf("Failed to initialize tracing: %v", err)
    }
    defer tm.Shutdown(ctx)
    
    // 使用追踪
    ctx, span := tm.StartSpan(ctx, "main-operation")
    defer span.End()
    
    // 业务逻辑...
}
```

## 配置选项

### 完整配置示例

```yaml
tracing:
  # 基础配置
  enabled: true
  service_name: "swit-serve"
  
  # 采样配置
  sampling:
    type: "traceidratio"  # always_on, always_off, traceidratio
    rate: 0.1            # 10% 采样（生产环境）
  
  # 导出器配置
  exporter:
    type: "jaeger"  # console, jaeger, otlp
    endpoint: "http://jaeger:14268/api/traces"
    timeout: "10s"
    
    # Jaeger 特定配置
    jaeger:
      collector_endpoint: "http://jaeger:14268/api/traces"
      username: ""
      password: ""
      rpc_timeout: "5s"
    
    # OTLP 特定配置
    otlp:
      endpoint: "http://otel-collector:4318/v1/traces"
      insecure: true
      compression: "gzip"  # gzip, none
      headers:
        Authorization: "Bearer <token>"
  
  # 资源属性
  resource_attributes:
    service.version: "v1.2.3"
    deployment.environment: "production"
    service.namespace: "swit"
    service.instance.id: "instance-001"
    team: "platform"
    cost.center: "engineering"
  
  # 上下文传播器
  propagators:
    - "tracecontext"
    - "baggage"
```

### 环境变量覆盖

所有配置项都可以通过环境变量覆盖：

```bash
# 基础配置
export SWIT_TRACING_ENABLED=true
export SWIT_TRACING_SERVICE_NAME=my-service

# 采样配置
export SWIT_TRACING_SAMPLING_TYPE=traceidratio
export SWIT_TRACING_SAMPLING_RATE=0.5

# 导出器配置
export SWIT_TRACING_EXPORTER_TYPE=jaeger
export SWIT_TRACING_EXPORTER_ENDPOINT=http://jaeger:14268/api/traces
export SWIT_TRACING_EXPORTER_TIMEOUT=10s

# Jaeger 配置
export SWIT_TRACING_JAEGER_AGENT_ENDPOINT=localhost:6831
export SWIT_TRACING_JAEGER_COLLECTOR_ENDPOINT=http://localhost:14268/api/traces

# OTLP 配置
export SWIT_TRACING_OTLP_ENDPOINT=http://otel:4318/v1/traces
export SWIT_TRACING_OTLP_INSECURE=true

# 资源属性
export SWIT_TRACING_RESOURCE_SERVICE_VERSION=v1.0.0
export SWIT_TRACING_RESOURCE_ENVIRONMENT=production
export SWIT_TRACING_RESOURCE_SERVICE_NAMESPACE=swit

# 自定义属性（使用 CUSTOM_ 前缀）
export SWIT_TRACING_RESOURCE_CUSTOM_TEAM=platform
export SWIT_TRACING_RESOURCE_CUSTOM_COST_CENTER=engineering

# 传播器
export SWIT_TRACING_PROPAGATORS=tracecontext,baggage
```

## 使用示例

### 创建和管理 Span

```go
// 创建 span
ctx, span := tm.StartSpan(ctx, "operation-name",
    tracing.WithSpanKind(trace.SpanKindServer),
    tracing.WithAttributes(
        attribute.String("user.id", userID),
        attribute.String("request.method", "POST"),
    ),
)
defer span.End()

// 添加属性
span.SetAttribute("result.count", 42)
span.SetAttributes(
    attribute.String("status", "success"),
    attribute.Int64("duration_ms", 100),
)

// 添加事件
span.AddEvent("processing.started")
span.AddEvent("cache.hit", trace.WithAttributes(
    attribute.String("cache.key", key),
))

// 记录错误
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, "operation failed")
    return err
}

span.SetStatus(codes.Ok, "")
```

### 嵌套 Span

```go
func ProcessOrder(ctx context.Context, orderID string) error {
    ctx, span := tm.StartSpan(ctx, "ProcessOrder")
    defer span.End()
    
    span.SetAttribute("order.id", orderID)
    
    // 子操作 1
    if err := validateOrder(ctx, orderID); err != nil {
        span.RecordError(err)
        return err
    }
    
    // 子操作 2
    if err := chargePayment(ctx, orderID); err != nil {
        span.RecordError(err)
        return err
    }
    
    span.SetStatus(codes.Ok, "")
    return nil
}

func validateOrder(ctx context.Context, orderID string) error {
    ctx, span := tm.StartSpan(ctx, "ValidateOrder")
    defer span.End()
    
    // 验证逻辑...
    span.SetAttribute("validation.result", "passed")
    return nil
}

func chargePayment(ctx context.Context, orderID string) error {
    ctx, span := tm.StartSpan(ctx, "ChargePayment")
    defer span.End()
    
    // 支付逻辑...
    span.SetAttribute("payment.amount", 99.99)
    span.SetAttribute("payment.currency", "USD")
    return nil
}
```

## 中间件集成

### HTTP 中间件

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/pkg/middleware"
    "github.com/innovationmech/swit/pkg/tracing"
)

func main() {
    // 创建追踪管理器
    tm := tracing.NewTracingManager()
    tm.Initialize(context.Background(), config)
    
    // 创建 Gin 路由
    r := gin.Default()
    
    // 添加追踪中间件（使用默认配置）
    r.Use(middleware.TracingMiddleware(tm))
    
    // 或使用自定义配置
    tracingConfig := &middleware.HTTPTracingConfig{
        SkipPaths:      []string{"/health", "/metrics"},
        RecordReqBody:  false,
        RecordRespBody: false,
        MaxBodySize:    4096,
        CustomAttributes: map[string]func(*gin.Context) string{
            "user.id": func(c *gin.Context) string {
                // 从上下文提取用户 ID
                return c.GetString("user_id")
            },
        },
    }
    r.Use(middleware.TracingMiddlewareWithConfig(tm, tracingConfig))
    
    // 定义路由
    r.GET("/api/users/:id", func(c *gin.Context) {
        // 从上下文获取 span
        span := tm.SpanFromContext(c.Request.Context())
        span.SetAttribute("user.id", c.Param("id"))
        
        // 业务逻辑...
        c.JSON(200, gin.H{"status": "ok"})
    })
    
    r.Run(":8080")
}
```

### gRPC 拦截器

```go
import (
    "google.golang.org/grpc"
    "github.com/innovationmech/swit/pkg/middleware"
    "github.com/innovationmech/swit/pkg/tracing"
)

func main() {
    // 服务端
    tm := tracing.NewTracingManager()
    tm.Initialize(context.Background(), config)
    
    // 创建 gRPC 服务器
    server := grpc.NewServer(
        grpc.UnaryInterceptor(middleware.UnaryServerInterceptor(tm)),
        grpc.StreamInterceptor(middleware.StreamServerInterceptor(tm)),
    )
    
    // 注册服务...
    pb.RegisterYourServiceServer(server, &yourService{})
    
    // 启动服务器
    lis, _ := net.Listen("tcp", ":50051")
    server.Serve(lis)
}

func createGRPCClient() *grpc.ClientConn {
    // 客户端
    tm := tracing.NewTracingManager()
    tm.Initialize(context.Background(), config)
    
    conn, _ := grpc.Dial("localhost:50051",
        grpc.WithInsecure(),
        grpc.WithUnaryInterceptor(middleware.UnaryClientInterceptor(tm)),
        grpc.WithStreamInterceptor(middleware.StreamClientInterceptor(tm)),
    )
    
    return conn
}
```

### HTTP 客户端追踪

```go
import (
    "net/http"
    "github.com/innovationmech/swit/pkg/middleware"
)

func createHTTPClient(tm tracing.TracingManager) *http.Client {
    return &http.Client{
        Transport: middleware.HTTPClientTracingRoundTripper(tm, nil),
    }
}

func makeRequest(ctx context.Context, client *http.Client) error {
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://api.example.com/data", nil)
    
    // 追踪上下文会自动注入到请求头中
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // 处理响应...
    return nil
}
```

## 数据库追踪

### GORM 集成

```go
import (
    "gorm.io/gorm"
    "github.com/innovationmech/swit/pkg/tracing"
)

func setupDatabase(tm tracing.TracingManager) (*gorm.DB, error) {
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, err
    }
    
    // 安装追踪插件
    if err := tracing.InstallGormTracing(db, tm); err != nil {
        return nil, err
    }
    
    return db, nil
}

func queryUser(ctx context.Context, db *gorm.DB, userID string) (*User, error) {
    // 使用带追踪的数据库连接
    var user User
    
    // 从上下文获取追踪信息
    tracedDB := tracing.GormDBWithTracing(ctx, db)
    
    // 执行查询（会自动创建 span）
    if err := tracedDB.Where("id = ?", userID).First(&user).Error; err != nil {
        return nil, err
    }
    
    return &user, nil
}
```

### 自定义配置

```go
config := &tracing.GormTracingConfig{
    RecordSQL:        true,  // 记录 SQL 语句
    RecordParameters: false, // 不记录参数（安全）
    RecordRows:       true,  // 记录受影响行数
    SlowQueryThreshold: 100 * time.Millisecond, // 慢查询阈值
    SkipTables: []string{
        "schema_migrations",
        "internal_cache",
    },
}

tracing.InstallGormTracingWithConfig(db, tm, config)
```

## 最佳实践

### 1. 采样策略

**开发环境**：
```yaml
sampling:
  type: "always_on"  # 100% 采样
```

**生产环境**：
```yaml
sampling:
  type: "traceidratio"
  rate: 0.1  # 10% 采样，降低开销
```

### 2. Span 命名

使用一致的命名约定：

```go
// HTTP: 方法 + 路径
"GET /api/users/:id"

// gRPC: 完整方法名
"/user.UserService/GetUser"

// 数据库: 操作 + 表名
"SELECT users"

// 业务操作: 动词 + 名词
"ProcessOrder"
"ValidatePayment"
"SendNotification"
```

### 3. 属性管理

```go
// 使用语义化属性名
span.SetAttribute("http.method", "GET")
span.SetAttribute("http.status_code", 200)
span.SetAttribute("db.system", "postgresql")
span.SetAttribute("db.statement", "SELECT * FROM users")

// 业务属性使用自定义前缀
span.SetAttribute("app.user.id", userID)
span.SetAttribute("app.order.total", 99.99)
span.SetAttribute("app.tenant.id", tenantID)
```

### 4. 错误处理

```go
func ProcessRequest(ctx context.Context) error {
    ctx, span := tm.StartSpan(ctx, "ProcessRequest")
    defer span.End()
    
    if err := doSomething(ctx); err != nil {
        // 记录错误
        span.RecordError(err)
        
        // 设置错误状态
        span.SetStatus(codes.Error, err.Error())
        
        // 添加错误相关属性
        span.SetAttribute("error.type", "validation_error")
        span.SetAttribute("error.details", err.Error())
        
        return err
    }
    
    // 成功时设置状态
    span.SetStatus(codes.Ok, "")
    return nil
}
```

### 5. 性能优化

```go
// 跳过健康检查等高频端点
config := &middleware.HTTPTracingConfig{
    SkipPaths: []string{
        "/health",
        "/ready",
        "/metrics",
        "/livez",
        "/readyz",
    },
}

// 限制记录的数据大小
config.RecordReqBody = false   // 不记录请求体
config.RecordRespBody = false  // 不记录响应体
config.MaxBodySize = 4096      // 最大 4KB
```

### 6. 生产环境配置

```yaml
tracing:
  enabled: true
  service_name: "swit-serve"
  
  sampling:
    type: "traceidratio"
    rate: 0.1  # 10% 采样
  
  exporter:
    type: "otlp"
    endpoint: "http://otel-collector:4318/v1/traces"
    timeout: "10s"
    otlp:
      insecure: false  # 使用 TLS
      compression: "gzip"  # 启用压缩
  
  resource_attributes:
    deployment.environment: "production"
    service.version: "${APP_VERSION}"
    service.namespace: "swit"
    k8s.pod.name: "${HOSTNAME}"
```

## 故障排查

### 常见问题

#### 1. Trace 上下文未传播

**症状**：跨服务调用时，trace 没有连接起来

**解决方案**：
```go
// HTTP 客户端：确保使用追踪的 RoundTripper
client := &http.Client{
    Transport: middleware.HTTPClientTracingRoundTripper(tm, nil),
}

// HTTP 服务端：确保使用追踪中间件
r.Use(middleware.TracingMiddleware(tm))

// 检查传播器配置
propagators:
  - "tracecontext"
  - "baggage"
```

#### 2. Spans 未导出

**症状**：创建了 span 但在 Jaeger 中看不到

**解决方案**：
```go
// 检查初始化
if err := tm.Initialize(ctx, config); err != nil {
    log.Printf("Tracing init failed: %v", err)
}

// 确保在退出时正确关闭
defer func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := tm.Shutdown(ctx); err != nil {
        log.Printf("Tracing shutdown failed: %v", err)
    }
}()

// 检查导出器端点
exporter:
  type: "jaeger"
  endpoint: "http://localhost:14268/api/traces"  # 确保端口正确
```

#### 3. 采样率问题

**症状**：只能看到部分 trace

**解决方案**：
```yaml
# 开发环境使用 always_on
sampling:
  type: "always_on"

# 或提高采样率
sampling:
  type: "traceidratio"
  rate: 1.0  # 100%
```

#### 4. 性能影响

**症状**：启用追踪后性能下降

**解决方案**：
```yaml
# 降低采样率
sampling:
  rate: 0.1  # 10%

# 使用批处理器（默认）
# 跳过高频端点
skip_paths:
  - "/health"
  - "/metrics"

# 不记录请求/响应体
record_req_body: false
record_resp_body: false
```

### 调试技巧

#### 1. 启用 Console 导出器

```yaml
exporter:
  type: "console"  # 输出到标准输出
```

#### 2. 检查配置

```go
config := tracing.DefaultTracingConfig()
config.ApplyEnvironmentOverrides()

if err := config.Validate(); err != nil {
    log.Printf("Config validation failed: %v", err)
}

log.Printf("Tracing config: %+v", config)
```

#### 3. 手动验证上下文传播

```go
// 检查 span 上下文
spanCtx := span.SpanContext()
log.Printf("TraceID: %s, SpanID: %s, TraceFlags: %s",
    spanCtx.TraceID(),
    spanCtx.SpanID(),
    spanCtx.TraceFlags(),
)

// 检查 HTTP headers
headers := make(http.Header)
tm.InjectHTTPHeaders(ctx, headers)
log.Printf("Tracing headers: %+v", headers)
```

## 参考资源

### 文档

- [DISTRIBUTED_TRACING.md](../DISTRIBUTED_TRACING.md) - 分布式追踪实现指南
- [examples/distributed-tracing/](../examples/distributed-tracing/) - 完整示例项目
- [examples/tracing-config/](../examples/tracing-config/) - 配置示例

### API 文档

- `pkg/tracing` - 核心追踪 API
- `pkg/middleware` - HTTP/gRPC 中间件
- `pkg/tracing/db.go` - 数据库追踪

### 外部资源

- [OpenTelemetry 官方文档](https://opentelemetry.io/docs/)
- [OpenTelemetry Go SDK](https://github.com/open-telemetry/opentelemetry-go)
- [Jaeger 文档](https://www.jaegertracing.io/docs/)
- [OTLP 规范](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md)

## 贡献

如有问题或建议，请提交 Issue 或 Pull Request。

## 许可证

Copyright © 2025 jackelyj <dreamerlyj@gmail.com>

本软件采用 MIT 许可证授权。


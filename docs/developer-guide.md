# 分布式追踪开发者指南

本文档为开发者提供 SWIT 框架分布式追踪功能的深入集成指南，包括 API 使用、自定义追踪实现、框架集成模式和高级配置选项。

## 目录

- [API 使用说明](#api-使用说明)
- [自定义追踪实现](#自定义追踪实现)
- [框架集成模式](#框架集成模式)
- [高级配置选项](#高级配置选项)
- [性能调优](#性能调优)
- [扩展开发](#扩展开发)
- [测试指南](#测试指南)
- [代码示例](#代码示例)

## API 使用说明

### 核心 API 概览

SWIT 框架基于 OpenTelemetry 标准提供追踪功能，主要包含以下核心 API：

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)
```

### 1. Tracer 获取和使用

#### 获取 Tracer 实例

```go
type MyService struct {
    tracer trace.Tracer
}

func NewMyService() *MyService {
    return &MyService{
        tracer: otel.Tracer("my-service"),
    }
}
```

#### 创建 Span

```go
func (s *MyService) ProcessOrder(ctx context.Context, orderID string) error {
    // 创建子 span
    ctx, span := s.tracer.Start(ctx, "ProcessOrder")
    defer span.End()
    
    // 添加属性
    span.SetAttribute(attribute.String("order.id", orderID))
    span.SetAttribute(attribute.Int("order.priority", 1))
    
    // 业务逻辑
    err := s.doProcess(ctx, orderID)
    if err != nil {
        span.SetStatus(codes.Error, err.Error())
        span.RecordError(err)
        return err
    }
    
    span.SetStatus(codes.Ok, "处理完成")
    return nil
}
```

### 2. Span 操作 API

#### 设置 Span 属性

```go
// 字符串属性
span.SetAttribute(attribute.String("user.id", "12345"))

// 数值属性
span.SetAttribute(attribute.Int("order.item_count", 5))
span.SetAttribute(attribute.Float64("order.total", 99.99))

// 布尔属性
span.SetAttribute(attribute.Bool("order.is_premium", true))

// 批量设置属性
span.SetAttributes(
    attribute.String("service.name", "order-service"),
    attribute.String("service.version", "1.0.0"),
    attribute.String("deployment.environment", "production"),
)
```

#### 添加 Span 事件

```go
// 简单事件
span.AddEvent("订单验证开始")

// 带时间戳的事件
span.AddEvent("库存检查", trace.WithTimestamp(time.Now()))

// 带属性的事件
span.AddEvent("支付处理", trace.WithAttributes(
    attribute.String("payment.method", "credit_card"),
    attribute.Float64("payment.amount", 99.99),
))
```

#### 记录错误

```go
func (s *MyService) HandleOrder(ctx context.Context) error {
    span := trace.SpanFromContext(ctx)
    
    if err := s.validateOrder(); err != nil {
        // 记录错误
        span.RecordError(err)
        
        // 设置错误状态
        span.SetStatus(codes.Error, "订单验证失败")
        
        return err
    }
    
    return nil
}
```

### 3. 上下文传播

#### 跨函数调用

```go
func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
    ctx, span := s.tracer.Start(ctx, "CreateOrder")
    defer span.End()
    
    // 传递上下文到子函数
    if err := s.validateOrder(ctx, req); err != nil {
        return err
    }
    
    if err := s.reserveInventory(ctx, req); err != nil {
        return err
    }
    
    return s.processPayment(ctx, req)
}

func (s *OrderService) validateOrder(ctx context.Context, req *CreateOrderRequest) error {
    // 从上下文创建子 span
    ctx, span := s.tracer.Start(ctx, "ValidateOrder")
    defer span.End()
    
    // 验证逻辑...
    return nil
}
```

#### 跨服务调用

```go
import "github.com/your-org/swit/pkg/client"

func (s *OrderService) callPaymentService(ctx context.Context, amount float64) error {
    // 使用支持追踪的 HTTP 客户端
    client := client.NewTracedHTTPClient("payment-service")
    
    req := PaymentRequest{Amount: amount}
    
    // 上下文会自动传播到远程服务
    resp, err := client.Post(ctx, "/api/v1/payments", req)
    if err != nil {
        return err
    }
    
    return s.handleResponse(resp)
}
```

## 自定义追踪实现

### 1. 自定义 Span 处理器

```go
import (
    "go.opentelemetry.io/otel/sdk/trace"
)

// 自定义 Span 处理器
type CustomSpanProcessor struct {
    // 自定义字段
}

func (p *CustomSpanProcessor) OnStart(parent context.Context, s trace.ReadWriteSpan) {
    // Span 开始时的处理逻辑
    s.SetAttribute(attribute.String("custom.processor", "active"))
    
    // 记录开始时间
    s.AddEvent("span.processing.started")
}

func (p *CustomSpanProcessor) OnEnd(s trace.ReadOnlySpan) {
    // Span 结束时的处理逻辑
    duration := s.EndTime().Sub(s.StartTime())
    
    // 记录持续时间
    if duration > time.Second {
        // 记录慢操作
        log.Printf("慢操作检测: %s 耗时 %v", s.Name(), duration)
    }
}

func (p *CustomSpanProcessor) Shutdown(ctx context.Context) error {
    return nil
}

func (p *CustomSpanProcessor) ForceFlush(ctx context.Context) error {
    return nil
}

// 注册自定义处理器
func setupCustomTracing() {
    tp := trace.NewTracerProvider(
        trace.WithSpanProcessor(&CustomSpanProcessor{}),
        // 其他配置...
    )
    otel.SetTracerProvider(tp)
}
```

### 2. 自定义采样器

```go
type BusinessSampler struct {
    defaultSampler trace.Sampler
}

func NewBusinessSampler() *BusinessSampler {
    return &BusinessSampler{
        defaultSampler: trace.TraceIDRatioBased(0.1), // 默认 10% 采样
    }
}

func (s *BusinessSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
    // 检查是否为重要业务操作
    for _, attr := range p.Attributes {
        if attr.Key == "operation.critical" && attr.Value.AsBool() {
            // 重要操作总是采样
            return trace.SamplingResult{
                Decision:   trace.RecordAndSample,
                Attributes: p.Attributes,
            }
        }
    }
    
    // 检查错误情况
    if p.Name == "error-handler" {
        // 错误总是采样
        return trace.SamplingResult{
            Decision:   trace.RecordAndSample,
            Attributes: p.Attributes,
        }
    }
    
    // 使用默认采样策略
    return s.defaultSampler.ShouldSample(p)
}

func (s *BusinessSampler) Description() string {
    return "BusinessSampler"
}
```

### 3. 自定义导出器

```go
type MetricsExporter struct {
    metricsClient MetricsClient
}

func NewMetricsExporter(client MetricsClient) *MetricsExporter {
    return &MetricsExporter{
        metricsClient: client,
    }
}

func (e *MetricsExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
    for _, span := range spans {
        // 提取业务指标
        if span.Name() == "CreateOrder" {
            duration := span.EndTime().Sub(span.StartTime())
            
            // 发送到指标系统
            e.metricsClient.RecordDuration("order.create.duration", duration)
            
            // 检查状态
            if span.Status().Code == codes.Error {
                e.metricsClient.IncrementCounter("order.create.errors")
            } else {
                e.metricsClient.IncrementCounter("order.create.success")
            }
        }
    }
    
    return nil
}

func (e *MetricsExporter) Shutdown(ctx context.Context) error {
    return e.metricsClient.Close()
}
```

## 框架集成模式

### 1. 服务注册器模式

```go
import "github.com/your-org/swit/pkg/server"

type OrderService struct {
    tracer     trace.Tracer
    repository OrderRepository
}

func NewOrderService(repo OrderRepository) *OrderService {
    return &OrderService{
        tracer:     otel.Tracer("order-service"),
        repository: repo,
    }
}

// 实现 BusinessServiceRegistrar 接口
func (s *OrderService) RegisterServices(registry server.BusinessServiceRegistry) error {
    // 注册 HTTP 处理器
    if err := registry.RegisterBusinessHTTPHandler(&OrderHTTPHandler{
        orderService: s,
    }); err != nil {
        return err
    }
    
    // 注册 gRPC 服务
    if err := registry.RegisterBusinessGRPCService(&OrderGRPCService{
        orderService: s,
    }); err != nil {
        return err
    }
    
    // 注册健康检查
    return registry.RegisterBusinessHealthCheck(&OrderHealthCheck{
        orderService: s,
    })
}

// HTTP 处理器
type OrderHTTPHandler struct {
    orderService *OrderService
}

func (h *OrderHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    
    // 路由会自动包含追踪中间件
    ginRouter.POST("/api/v1/orders", h.createOrder)
    ginRouter.GET("/api/v1/orders/:id", h.getOrder)
    
    return nil
}

func (h *OrderHTTPHandler) createOrder(c *gin.Context) {
    // gin.Context 已经包含追踪上下文
    ctx := c.Request.Context()
    
    var req CreateOrderRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    order, err := h.orderService.CreateOrder(ctx, &req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, order)
}
```

### 2. 中间件集成模式

```go
// 自定义追踪中间件
func TracingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tracer := otel.Tracer("http-middleware")
        
        ctx, span := tracer.Start(c.Request.Context(), c.Request.Method+" "+c.Request.URL.Path)
        defer span.End()
        
        // 设置HTTP属性
        span.SetAttribute(attribute.String("http.method", c.Request.Method))
        span.SetAttribute(attribute.String("http.url", c.Request.URL.String()))
        span.SetAttribute(attribute.String("http.user_agent", c.Request.UserAgent()))
        
        // 更新请求上下文
        c.Request = c.Request.WithContext(ctx)
        
        // 继续处理
        c.Next()
        
        // 设置响应属性
        span.SetAttribute(attribute.Int("http.status_code", c.Writer.Status()))
        
        if c.Writer.Status() >= 400 {
            span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", c.Writer.Status()))
        }
    }
}

// 在服务中使用
func (h *OrderHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    
    // 应用追踪中间件
    ginRouter.Use(TracingMiddleware())
    
    ginRouter.POST("/api/v1/orders", h.createOrder)
    return nil
}
```

### 3. 依赖注入模式

```go
import "github.com/your-org/swit/pkg/server"

type OrderDependencies struct {
    tracer     trace.Tracer
    db         *gorm.DB
    redisClient *redis.Client
}

func NewOrderDependencies() *OrderDependencies {
    return &OrderDependencies{
        tracer: otel.Tracer("order-dependencies"),
    }
}

// 实现 BusinessDependencyContainer 接口
func (d *OrderDependencies) InitializeDependencies() error {
    ctx, span := d.tracer.Start(context.Background(), "InitializeDependencies")
    defer span.End()
    
    // 初始化数据库连接
    if err := d.initDatabase(ctx); err != nil {
        span.RecordError(err)
        return err
    }
    
    // 初始化 Redis 连接
    if err := d.initRedis(ctx); err != nil {
        span.RecordError(err)
        return err
    }
    
    return nil
}

func (d *OrderDependencies) GetService(name string) (interface{}, error) {
    switch name {
    case "tracer":
        return d.tracer, nil
    case "db":
        return d.db, nil
    case "redis":
        return d.redisClient, nil
    default:
        return nil, fmt.Errorf("未知的服务: %s", name)
    }
}

func (d *OrderDependencies) Close() error {
    ctx, span := d.tracer.Start(context.Background(), "CloseDependencies")
    defer span.End()
    
    // 关闭资源
    if d.db != nil {
        sqlDB, _ := d.db.DB()
        sqlDB.Close()
    }
    
    if d.redisClient != nil {
        d.redisClient.Close()
    }
    
    return nil
}
```

## 高级配置选项

### 1. 资源配置

```go
import (
    "go.opentelemetry.io/otel/sdk/resource"
    semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

func createResource(serviceName, serviceVersion string) (*resource.Resource, error) {
    return resource.Merge(
        resource.Default(),
        resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName(serviceName),
            semconv.ServiceVersion(serviceVersion),
            semconv.DeploymentEnvironment(os.Getenv("ENVIRONMENT")),
            
            // 自定义属性
            attribute.String("team.name", "platform"),
            attribute.String("service.owner", "platform-team"),
            attribute.String("cost.center", "engineering"),
        ),
    )
}
```

### 2. 批量导出器配置

```go
import "go.opentelemetry.io/otel/exporters/jaeger"

func setupJaegerExporter() (trace.SpanExporter, error) {
    return jaeger.New(
        jaeger.WithCollectorEndpoint(
            jaeger.WithEndpoint("http://jaeger:14268/api/traces"),
            jaeger.WithUsername("user"),
            jaeger.WithPassword("pass"),
        ),
        jaeger.WithProcess(jaeger.Process{
            ServiceName: "order-service",
            Tags: []attribute.KeyValue{
                attribute.String("version", "1.0.0"),
            },
        }),
    )
}

func setupBatchProcessor(exporter trace.SpanExporter) trace.SpanProcessor {
    return trace.NewBatchSpanProcessor(
        exporter,
        trace.WithBatchTimeout(5*time.Second),      // 批处理超时
        trace.WithMaxExportBatchSize(512),          // 最大批量大小
        trace.WithMaxQueueSize(2048),               // 最大队列大小
        trace.WithExportTimeout(30*time.Second),    // 导出超时
    )
}
```

### 3. 多导出器配置

```go
func setupMultipleExporters() trace.SpanProcessor {
    // Jaeger 导出器
    jaegerExporter, _ := jaeger.New(/* config */)
    
    // OTLP 导出器  
    otlpExporter, _ := otlptrace.New(/* config */)
    
    // 控制台导出器（开发环境）
    consoleExporter, _ := stdouttrace.New()
    
    // 创建多个批处理器
    return trace.NewMultiSpanProcessor(
        trace.NewBatchSpanProcessor(jaegerExporter),
        trace.NewBatchSpanProcessor(otlpExporter),
        trace.NewSimpleSpanProcessor(consoleExporter), // 开发环境
    )
}
```

## 性能调优

### 1. 采样策略优化

```go
// 复合采样器
func createCompositeSampler() trace.Sampler {
    return trace.NewParentBasedSampler(
        trace.NewCompositeSampler([]trace.Sampler{
            // 错误总是采样
            NewConditionalSampler(
                func(p trace.SamplingParameters) bool {
                    return p.Name == "error-handler"
                },
                trace.NewAlwaysOnSampler(),
            ),
            
            // 关键业务操作高采样率
            NewConditionalSampler(
                func(p trace.SamplingParameters) bool {
                    for _, attr := range p.Attributes {
                        if attr.Key == "business.critical" {
                            return attr.Value.AsBool()
                        }
                    }
                    return false
                },
                trace.NewTraceIDRatioBasedSampler(0.5), // 50%
            ),
            
            // 默认低采样率
            trace.NewTraceIDRatioBasedSampler(0.01), // 1%
        }),
    )
}
```

### 2. 内存优化

```go
// 限制 Span 属性数量
func limitSpanAttributes(span trace.Span, attrs []attribute.KeyValue) {
    const maxAttributes = 50
    
    if len(attrs) > maxAttributes {
        // 优先保留重要属性
        importantAttrs := filterImportantAttributes(attrs)
        span.SetAttributes(importantAttrs[:maxAttributes]...)
    } else {
        span.SetAttributes(attrs...)
    }
}

// 异步处理大量数据
func (s *OrderService) ProcessLargeOrder(ctx context.Context, order *LargeOrder) error {
    span := trace.SpanFromContext(ctx)
    
    // 只记录关键信息
    span.SetAttribute(attribute.String("order.id", order.ID))
    span.SetAttribute(attribute.Int("order.item_count", len(order.Items)))
    
    // 异步处理详细信息
    go func() {
        detailSpan := s.tracer.Start(context.Background(), "ProcessOrderDetails")
        defer detailSpan.End()
        
        // 处理详细信息
        s.processOrderDetails(order)
    }()
    
    return nil
}
```

### 3. 网络优化

```go
// 使用压缩
func setupOTLPExporter() trace.SpanExporter {
    return otlptracegrpc.New(
        context.Background(),
        otlptracegrpc.WithEndpoint("http://otel-collector:4317"),
        otlptracegrpc.WithInsecure(),
        otlptracegrpc.WithCompressor("gzip"), // 启用压缩
        otlptracegrpc.WithHeaders(map[string]string{
            "api-key": "your-api-key",
        }),
    )
}
```

## 扩展开发

### 1. 自定义仪表化

```go
// 自动仪表化包装器
func WithTracing(serviceName string, fn func(context.Context) error) func(context.Context) error {
    tracer := otel.Tracer(serviceName)
    
    return func(ctx context.Context) error {
        ctx, span := tracer.Start(ctx, "auto-instrumented")
        defer span.End()
        
        err := fn(ctx)
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
        }
        
        return err
    }
}

// 使用示例
func (s *OrderService) ProcessOrder(ctx context.Context) error {
    return WithTracing("order-service", func(ctx context.Context) error {
        // 业务逻辑
        return s.doProcess(ctx)
    })(ctx)
}
```

### 2. 插件系统

```go
// 追踪插件接口
type TracingPlugin interface {
    Name() string
    OnSpanStart(context.Context, trace.ReadWriteSpan)
    OnSpanEnd(trace.ReadOnlySpan)
}

// 安全插件实现
type SecurityPlugin struct{}

func (p *SecurityPlugin) Name() string {
    return "security"
}

func (p *SecurityPlugin) OnSpanStart(ctx context.Context, span trace.ReadWriteSpan) {
    // 检查敏感操作
    if span.Name() == "ProcessPayment" {
        span.SetAttribute(attribute.Bool("security.sensitive", true))
        
        // 记录安全事件
        span.AddEvent("security.check.started")
    }
}

func (p *SecurityPlugin) OnSpanEnd(span trace.ReadOnlySpan) {
    // 安全审计
    if span.Attributes().HasValue("security.sensitive") {
        // 记录到安全日志
        log.Printf("敏感操作完成: %s", span.Name())
    }
}

// 插件管理器
type PluginManager struct {
    plugins []TracingPlugin
}

func (pm *PluginManager) RegisterPlugin(plugin TracingPlugin) {
    pm.plugins = append(pm.plugins, plugin)
}

func (pm *PluginManager) OnSpanStart(ctx context.Context, span trace.ReadWriteSpan) {
    for _, plugin := range pm.plugins {
        plugin.OnSpanStart(ctx, span)
    }
}

func (pm *PluginManager) OnSpanEnd(span trace.ReadOnlySpan) {
    for _, plugin := range pm.plugins {
        plugin.OnSpanEnd(span)
    }
}
```

## 测试指南

### 1. 单元测试

```go
import (
    "go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestOrderService_CreateOrder(t *testing.T) {
    // 创建测试追踪器
    exporter := tracetest.NewInMemoryExporter()
    tp := trace.NewTracerProvider(
        trace.WithSyncer(exporter),
        trace.WithSampler(trace.AlwaysSample()),
    )
    otel.SetTracerProvider(tp)
    
    // 创建服务
    service := NewOrderService()
    
    // 测试
    ctx := context.Background()
    err := service.CreateOrder(ctx, &CreateOrderRequest{
        CustomerID: "123",
        Items:      []OrderItem{{ProductID: "456", Quantity: 2}},
    })
    
    // 断言
    assert.NoError(t, err)
    
    // 验证追踪数据
    spans := exporter.GetSpans()
    assert.Len(t, spans, 3) // 预期的 span 数量
    
    rootSpan := spans[0]
    assert.Equal(t, "CreateOrder", rootSpan.Name)
    assert.Equal(t, "123", rootSpan.Attributes.Get("customer.id").AsString())
}
```

### 2. 集成测试

```go
func TestDistributedTracing_E2E(t *testing.T) {
    // 启动测试容器
    container := startTestContainers(t)
    defer container.Cleanup()
    
    // 配置追踪
    config := &TracingConfig{
        Enabled:  true,
        Exporter: TracingExporterConfig{
            Type:     "jaeger",
            Endpoint: container.JaegerEndpoint(),
        },
        Sampling: TracingSamplingConfig{
            Type: "always_on",
        },
    }
    
    // 启动服务
    services := startTestServices(t, config)
    defer services.Cleanup()
    
    // 执行业务操作
    client := NewTestClient(services.OrderServiceURL())
    resp, err := client.CreateOrder(context.Background(), &CreateOrderRequest{
        CustomerID: "test-customer",
        Items: []OrderItem{
            {ProductID: "product-1", Quantity: 1},
        },
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, resp.OrderID)
    
    // 等待追踪数据传播
    time.Sleep(2 * time.Second)
    
    // 验证追踪数据
    traces := container.GetTraces()
    assert.NotEmpty(t, traces)
    
    // 验证完整的调用链
    assert.Contains(t, traces, "CreateOrder")
    assert.Contains(t, traces, "CheckInventory") 
    assert.Contains(t, traces, "ProcessPayment")
}
```

### 3. 性能测试

```go
func BenchmarkTracingOverhead(b *testing.B) {
    // 无追踪基线
    b.Run("NoTracing", func(b *testing.B) {
        service := NewOrderServiceWithoutTracing()
        ctx := context.Background()
        
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            _ = service.ProcessOrder(ctx)
        }
    })
    
    // 启用追踪
    b.Run("WithTracing", func(b *testing.B) {
        // 设置追踪
        tp := trace.NewTracerProvider(
            trace.WithSampler(trace.AlwaysSample()),
            trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(
                tracetest.NewNoopExporter(),
            )),
        )
        otel.SetTracerProvider(tp)
        
        service := NewOrderServiceWithTracing()
        ctx := context.Background()
        
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            _ = service.ProcessOrder(ctx)
        }
    })
}
```

## 代码示例

### 完整的微服务示例

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/gin-gonic/gin"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
    "go.opentelemetry.io/otel/attribute"
    
    "github.com/your-org/swit/pkg/server"
)

// 订单服务
type OrderService struct {
    tracer     trace.Tracer
    repository *OrderRepository
    client     *PaymentServiceClient
}

func NewOrderService(repo *OrderRepository, client *PaymentServiceClient) *OrderService {
    return &OrderService{
        tracer:     otel.Tracer("order-service"),
        repository: repo,
        client:     client,
    }
}

// 主要业务方法
func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
    // 创建根 span
    ctx, span := s.tracer.Start(ctx, "CreateOrder", 
        trace.WithAttributes(
            attribute.String("customer.id", req.CustomerID),
            attribute.Int("items.count", len(req.Items)),
        ),
    )
    defer span.End()
    
    // 1. 验证订单
    span.AddEvent("validation.started")
    if err := s.validateOrder(ctx, req); err != nil {
        span.AddEvent("validation.failed")
        span.RecordError(err)
        span.SetStatus(codes.Error, "订单验证失败")
        return nil, err
    }
    span.AddEvent("validation.completed")
    
    // 2. 创建订单记录
    order, err := s.createOrderRecord(ctx, req)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }
    
    span.SetAttribute(attribute.String("order.id", order.ID))
    
    // 3. 处理支付
    paymentResult, err := s.processPayment(ctx, order)
    if err != nil {
        span.AddEvent("payment.failed")
        span.RecordError(err)
        return nil, err
    }
    
    span.AddEvent("payment.completed")
    span.SetAttribute(attribute.String("payment.transaction_id", paymentResult.TransactionID))
    
    // 4. 更新订单状态
    order.Status = "confirmed"
    order.PaymentTransactionID = paymentResult.TransactionID
    
    if err := s.repository.Update(ctx, order); err != nil {
        span.RecordError(err)
        return nil, err
    }
    
    span.SetStatus(codes.Ok, "订单创建成功")
    return order, nil
}

func (s *OrderService) validateOrder(ctx context.Context, req *CreateOrderRequest) error {
    ctx, span := s.tracer.Start(ctx, "ValidateOrder")
    defer span.End()
    
    // 验证客户ID
    if req.CustomerID == "" {
        err := errors.New("客户ID不能为空")
        span.RecordError(err)
        return err
    }
    
    // 验证商品
    for i, item := range req.Items {
        if item.ProductID == "" {
            err := fmt.Errorf("商品 %d ID不能为空", i)
            span.RecordError(err)
            return err
        }
        if item.Quantity <= 0 {
            err := fmt.Errorf("商品 %d 数量必须大于0", i)
            span.RecordError(err) 
            return err
        }
    }
    
    return nil
}

func (s *OrderService) processPayment(ctx context.Context, order *Order) (*PaymentResult, error) {
    ctx, span := s.tracer.Start(ctx, "ProcessPayment")
    defer span.End()
    
    span.SetAttribute(attribute.Float64("payment.amount", order.TotalAmount))
    
    // 调用支付服务
    result, err := s.client.ProcessPayment(ctx, &PaymentRequest{
        OrderID:    order.ID,
        CustomerID: order.CustomerID,
        Amount:     order.TotalAmount,
    })
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, "支付处理失败")
        return nil, err
    }
    
    span.SetAttribute(attribute.String("payment.status", result.Status))
    return result, nil
}

// HTTP 处理器
type OrderHTTPHandler struct {
    orderService *OrderService
}

func (h *OrderHTTPHandler) RegisterRoutes(router interface{}) error {
    ginRouter := router.(*gin.Engine)
    ginRouter.POST("/api/v1/orders", h.createOrder)
    ginRouter.GET("/api/v1/orders/:id", h.getOrder)
    return nil
}

func (h *OrderHTTPHandler) createOrder(c *gin.Context) {
    ctx := c.Request.Context()
    
    var req CreateOrderRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    order, err := h.orderService.CreateOrder(ctx, &req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, order)
}

func (h *OrderHTTPHandler) GetServiceName() string {
    return "order-http-handler"
}

// 服务注册
func (s *OrderService) RegisterServices(registry server.BusinessServiceRegistry) error {
    return registry.RegisterBusinessHTTPHandler(&OrderHTTPHandler{
        orderService: s,
    })
}

// 主函数
func main() {
    // 创建配置
    config := &server.ServerConfig{
        Server: server.BaseServerConfig{
            Name: "order-service",
            HTTP: server.HTTPConfig{
                Port: "8080",
            },
        },
        Tracing: server.TracingConfig{
            Enabled:     true,
            ServiceName: "order-service",
            Exporter: server.TracingExporterConfig{
                Type:     "jaeger",
                Endpoint: "http://localhost:14268/api/traces",
            },
            Sampling: server.TracingSamplingConfig{
                Type: "traceidratio",
                Rate: 0.1,
            },
        },
    }
    
    // 创建依赖
    repository := NewOrderRepository()
    paymentClient := NewPaymentServiceClient()
    orderService := NewOrderService(repository, paymentClient)
    
    // 创建服务器
    baseServer, err := server.NewBusinessServerCore(config, orderService, nil)
    if err != nil {
        log.Fatal("创建服务器失败:", err)
    }
    
    // 启动服务器
    ctx := context.Background()
    if err := baseServer.Start(ctx); err != nil {
        log.Fatal("启动服务器失败:", err)
    }
}
```

## 相关资源

- [用户指南](tracing-user-guide.md) - 快速开始和基础配置
- [运维指南](operations-guide.md) - 生产环境部署和监控
- [架构文档](architecture.md) - 系统架构和设计原理
- [示例项目](../examples/distributed-tracing/) - 完整的使用示例
- [OpenTelemetry Go SDK](https://pkg.go.dev/go.opentelemetry.io/otel)

---

如有技术问题或需要支持，请联系开发团队或提交 GitHub Issue。

# 分布式追踪用户指南

本文档为 SWIT 微服务框架的分布式追踪功能提供完整的用户指南，帮助用户快速上手 OpenTelemetry 分布式追踪功能。

## 目录

- [快速开始](#快速开始)
- [环境搭建](#环境搭建)
- [配置详解](#配置详解)
- [基础使用](#基础使用)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)
- [故障排查](#故障排查)

## 快速开始

### 1. 前置条件

- Go 1.21+
- Docker 和 Docker Compose
- 基本的微服务架构理解

### 2. 运行示例

使用我们提供的完整分布式追踪示例：

```bash
# 克隆项目
git clone <repository-url>
cd swit/examples/distributed-tracing

# 一键启动环境
./scripts/setup.sh

# 查看服务状态
./scripts/health-check.sh

# 运行演示场景
./scripts/demo-scenarios.sh
```

### 3. 访问 Jaeger UI

启动成功后，访问 [http://localhost:16686](http://localhost:16686) 打开 Jaeger UI，查看分布式追踪数据。

## 环境搭建

### 开发环境

#### 方式一：Docker Compose（推荐）

```bash
# 进入示例目录
cd examples/distributed-tracing

# 复制环境配置
cp .env.example .env

# 启动所有服务
docker-compose up -d

# 检查服务状态
docker-compose ps
```

#### 方式二：本地开发

```bash
# 启动 Jaeger
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:1.42

# 设置环境变量
export JAEGER_ENDPOINT="http://localhost:14268/api/traces"
export OTEL_SERVICE_NAME="my-service"

# 启动你的服务
go run ./cmd/my-service
```

### 生产环境

生产环境建议使用 Jaeger 的分布式部署模式：

```yaml
# jaeger-production.yml
version: '3.8'
services:
  jaeger-collector:
    image: jaegertracing/jaeger-collector:1.42
    environment:
      - SPAN_STORAGE_TYPE=elasticsearch
      - ES_SERVER_URLS=http://elasticsearch:9200
    ports:
      - "14268:14268"
      - "14250:14250"

  jaeger-query:
    image: jaegertracing/jaeger-query:1.42
    environment:
      - SPAN_STORAGE_TYPE=elasticsearch
      - ES_SERVER_URLS=http://elasticsearch:9200
    ports:
      - "16686:16686"

  jaeger-agent:
    image: jaegertracing/jaeger-agent:1.42
    command: ["--collector.host-port=jaeger-collector:14267"]
    ports:
      - "6831:6831/udp"
      - "6832:6832/udp"
```

## 配置详解

### 基础配置

SWIT 框架通过 `ServerConfig` 结构提供追踪配置：

```go
type TracingConfig struct {
    // 启用追踪
    Enabled bool `yaml:"enabled" mapstructure:"enabled"`
    
    // 服务名称
    ServiceName string `yaml:"service_name" mapstructure:"service_name"`
    
    // 追踪导出器配置
    Exporter TracingExporterConfig `yaml:"exporter" mapstructure:"exporter"`
    
    // 采样配置
    Sampling TracingSamplingConfig `yaml:"sampling" mapstructure:"sampling"`
    
    // 资源属性
    ResourceAttributes map[string]string `yaml:"resource_attributes" mapstructure:"resource_attributes"`
}
```

### 配置文件示例

```yaml
# swit.yaml
tracing:
  enabled: true
  service_name: "my-microservice"
  
  exporter:
    type: "jaeger"  # 支持: jaeger, otlp, zipkin, console
    endpoint: "http://localhost:14268/api/traces"
    timeout: "10s"
    
    # Jaeger 特定配置
    jaeger:
      username: ""
      password: ""
      
    # OTLP 配置
    otlp:
      endpoint: "http://localhost:4318/v1/traces"
      headers:
        authorization: "Bearer token"
        
  sampling:
    type: "traceidratio"  # 支持: always_on, always_off, traceidratio, parentbased
    rate: 0.1  # 10% 采样率
    
  resource_attributes:
    deployment.environment: "production"
    service.version: "1.0.0"
    team.name: "platform"
```

### 环境变量覆盖

所有配置都支持环境变量覆盖：

```bash
# 基础配置
export SWIT_TRACING_ENABLED=true
export SWIT_TRACING_SERVICE_NAME=my-service

# 导出器配置
export SWIT_TRACING_EXPORTER_TYPE=jaeger
export SWIT_TRACING_EXPORTER_ENDPOINT=http://jaeger:14268/api/traces

# 采样配置
export SWIT_TRACING_SAMPLING_TYPE=traceidratio
export SWIT_TRACING_SAMPLING_RATE=0.1

# 资源属性
export SWIT_TRACING_RESOURCE_ATTRIBUTES_DEPLOYMENT_ENVIRONMENT=production
```

## 基础使用

### 1. 框架集成

使用 SWIT 框架的服务会自动启用追踪功能：

```go
package main

import (
    "context"
    "github.com/your-org/swit/pkg/server"
)

func main() {
    config := &server.ServerConfig{
        // ... 其他配置
        Tracing: server.TracingConfig{
            Enabled:     true,
            ServiceName: "order-service",
            Exporter: server.TracingExporterConfig{
                Type:     "jaeger",
                Endpoint: "http://localhost:14268/api/traces",
            },
        },
    }
    
    // 创建服务实例
    orderService := &OrderService{}
    
    // 创建服务器
    baseServer, err := server.NewBusinessServerCore(config, orderService, nil)
    if err != nil {
        panic(err)
    }
    
    // 启动服务器 - 追踪将自动启用
    ctx := context.Background()
    if err := baseServer.Start(ctx); err != nil {
        panic(err)
    }
}
```

### 2. HTTP 请求追踪

框架自动为所有 HTTP 请求创建追踪：

```go
func (h *OrderHandler) CreateOrder(c *gin.Context) {
    // 从 gin.Context 获取追踪上下文
    ctx := c.Request.Context()
    
    // 业务逻辑 - 会自动包含在追踪中
    order, err := h.orderService.CreateOrder(ctx, orderReq)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, order)
}
```

### 3. gRPC 服务追踪

gRPC 服务也会自动启用追踪：

```go
func (s *OrderServiceServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    // ctx 已经包含追踪信息
    
    // 调用业务逻辑
    order, err := s.orderService.CreateOrder(ctx, req)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "创建订单失败: %v", err)
    }
    
    return &pb.CreateOrderResponse{
        OrderId: order.ID,
        Status:  order.Status,
    }, nil
}
```

### 4. 数据库操作追踪

使用 GORM 的数据库操作会自动追踪：

```go
type OrderRepository struct {
    db *gorm.DB
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order *Order) error {
    // 使用带追踪上下文的数据库操作
    return r.db.WithContext(ctx).Create(order).Error
}
```

### 5. 服务间调用追踪

使用框架提供的 HTTP 客户端进行服务间调用：

```go
import "github.com/your-org/swit/pkg/client"

type PaymentService struct {
    httpClient *client.TracedHTTPClient
}

func NewPaymentService() *PaymentService {
    return &PaymentService{
        httpClient: client.NewTracedHTTPClient("payment-service"),
    }
}

func (s *PaymentService) ProcessPayment(ctx context.Context, amount float64) error {
    // 创建请求
    req := PaymentRequest{Amount: amount}
    
    // 发送带追踪的 HTTP 请求
    resp, err := s.httpClient.Post(ctx, "/api/v1/payments", req)
    if err != nil {
        return err
    }
    
    // 处理响应
    return s.handlePaymentResponse(resp)
}
```

## 最佳实践

### 1. 追踪采样策略

#### 开发环境
```yaml
sampling:
  type: "always_on"  # 记录所有追踪
```

#### 测试环境
```yaml
sampling:
  type: "traceidratio"
  rate: 0.5  # 50% 采样率
```

#### 生产环境
```yaml
sampling:
  type: "parentbased"  # 基于父级决策的采样
  rate: 0.01  # 1% 采样率
```

### 2. 关键业务指标

为重要的业务操作添加自定义属性和事件：

```go
func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
    span := trace.SpanFromContext(ctx)
    
    // 添加业务属性
    span.SetAttribute("order.customer_id", req.CustomerID)
    span.SetAttribute("order.product_count", len(req.Items))
    span.SetAttribute("order.total_amount", req.TotalAmount)
    
    // 记录关键事件
    span.AddEvent("validation.started")
    
    if err := s.validateOrder(req); err != nil {
        span.AddEvent("validation.failed", trace.WithEventOptions{
            Timestamp: time.Now(),
            Attributes: []attribute.KeyValue{
                attribute.String("error.reason", err.Error()),
            },
        })
        return nil, err
    }
    
    span.AddEvent("validation.completed")
    
    // 继续业务逻辑...
    return order, nil
}
```

### 3. 错误处理和状态设置

```go
func (s *OrderService) ProcessOrder(ctx context.Context, orderID string) error {
    span := trace.SpanFromContext(ctx)
    
    defer func() {
        if r := recover(); r != nil {
            span.SetStatus(codes.Error, fmt.Sprintf("panic: %v", r))
            span.RecordError(fmt.Errorf("panic: %v", r))
        }
    }()
    
    order, err := s.getOrder(ctx, orderID)
    if err != nil {
        span.SetStatus(codes.Error, "获取订单失败")
        span.RecordError(err)
        return err
    }
    
    // 处理成功
    span.SetStatus(codes.Ok, "订单处理完成")
    return nil
}
```

### 4. 性能优化

- **批量导出**: 使用批处理减少网络开销
- **异步导出**: 避免阻塞业务逻辑
- **采样控制**: 根据环境调整采样率
- **资源限制**: 设置内存和 CPU 限制

```yaml
tracing:
  exporter:
    batch_timeout: "5s"
    max_export_batch_size: 512
    max_queue_size: 2048
    export_timeout: "30s"
```

## 常见问题

### Q1: 为什么在 Jaeger UI 中看不到追踪数据？

**检查清单：**
1. 确认 Jaeger 服务正在运行：`curl http://localhost:16686`
2. 检查服务配置中的 endpoint 地址是否正确
3. 验证网络连接性：`telnet jaeger-host 14268`
4. 查看应用日志中的追踪相关错误信息
5. 确认采样率不为 0

### Q2: 追踪数据丢失或不完整？

**可能原因：**
- 采样率设置过低
- 导出器配置错误
- 网络连接问题
- Jaeger 存储容量不足

**解决方案：**
```yaml
# 临时提高采样率进行调试
sampling:
  type: "always_on"
  
# 启用控制台导出器查看本地输出
exporter:
  type: "console"
```

### Q3: 如何调试追踪配置？

启用调试模式：

```bash
export OTEL_LOG_LEVEL=debug
export SWIT_LOG_LEVEL=debug
```

### Q4: 服务间调用无法关联？

确保使用支持追踪的 HTTP 客户端：

```go
// ✅ 正确方式
client := client.NewTracedHTTPClient("target-service")
resp, err := client.Post(ctx, "/api/endpoint", data)

// ❌ 错误方式
resp, err := http.Post("http://target-service/api/endpoint", "application/json", body)
```

### Q5: 追踪对性能影响如何？

- **CPU 开销**: 通常 < 5%
- **内存开销**: 取决于采样率和批处理设置  
- **网络开销**: 主要在导出阶段

优化建议：
- 生产环境使用适当的采样率（1-10%）
- 启用批量导出
- 监控资源使用情况

## 故障排查

### 1. 连接问题

检查 Jaeger 连接：

```bash
# 检查 Jaeger UI
curl -f http://localhost:16686/api/services

# 检查收集器端口
telnet localhost 14268

# 测试追踪导出
curl -X POST http://localhost:14268/api/traces \
  -H "Content-Type: application/json" \
  -d '{"data":[{"traceID":"test","spans":[]}]}'
```

### 2. 配置验证

验证服务配置：

```bash
# 使用配置验证工具
./tools/config-validator --config swit.yaml --component tracing

# 检查环境变量
env | grep SWIT_TRACING
```

### 3. 日志分析

启用详细日志记录：

```yaml
# swit.yaml
logging:
  level: debug
  
tracing:
  debug: true
```

查看关键日志：

```bash
# 查看追踪初始化日志
grep "tracing" service.log

# 查看导出错误
grep "export.*error" service.log

# 查看采样决策
grep "sampling" service.log
```

### 4. 性能监控

使用内置的监控工具：

```bash
# 查看追踪统计
curl http://localhost:8080/debug/tracing/stats

# 查看内存使用
curl http://localhost:8080/debug/pprof/heap
```

### 5. 常见错误解决

#### 错误: "failed to create tracer provider"
```bash
# 检查导出器配置
export SWIT_TRACING_EXPORTER_TYPE=console  # 临时使用控制台导出
```

#### 错误: "context deadline exceeded"
```bash
# 增加导出超时时间
export SWIT_TRACING_EXPORTER_TIMEOUT=30s
```

#### 错误: "no spans to export"
```bash
# 检查采样配置
export SWIT_TRACING_SAMPLING_TYPE=always_on
```

## 相关资源

- [开发者指南](developer-guide.md) - 高级集成和自定义开发
- [运维指南](operations-guide.md) - 生产环境部署和监控
- [架构文档](architecture.md) - 系统架构和设计原理
- [示例项目](../examples/distributed-tracing/) - 完整的使用示例
- [OpenTelemetry 官方文档](https://opentelemetry.io/docs/)
- [Jaeger 官方文档](https://www.jaegertracing.io/docs/)

---

如有问题或建议，请提交 Issue 或联系开发团队。

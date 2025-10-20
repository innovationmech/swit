# Saga Messaging Package

`pkg/saga/messaging` 包提供了完整的 Saga 事件消息传递基础设施，支持可靠的事件发布、订阅、路由和处理。

## 目录

- [功能概述](#功能概述)
- [快速开始](#快速开始)
- [核心组件](#核心组件)
- [配置指南](#配置指南)
- [使用模式](#使用模式)
- [可靠性保证](#可靠性保证)
- [性能优化](#性能优化)
- [监控与指标](#监控与指标)
- [最佳实践](#最佳实践)
- [故障排查](#故障排查)
- [API 参考](#api-参考)

## 功能概述

### 事件发布器 (Publisher)

- ✅ 单个事件发布
- ✅ 批量事件发布
- ✅ 异步批量发布
- ✅ 事务性消息发送
- ✅ 发布确认机制
- ✅ 自动重试
- ✅ 死信队列（DLQ）
- ✅ 多种序列化格式（JSON、Protobuf）

### 事件处理器 (Handler)

- ✅ 事件路由和分发
- ✅ 事件过滤链
- ✅ 优先级处理
- ✅ 并发控制
- ✅ 错误恢复
- ✅ 事件反序列化

### 可靠性机制 (Reliability)

- ✅ 指数退避重试
- ✅ 发布确认
- ✅ 死信队列
- ✅ 超时控制
- ✅ 可靠性指标

### 其他组件

- ✅ 消息序列化/反序列化
- ✅ 事件路由器
- ✅ 事件过滤器
- ✅ DLQ 处理器

## 快速开始

### 安装

```bash
go get github.com/innovationmech/swit/pkg/saga/messaging
```

### 基础发布示例

```go
package main

import (
    "context"
    "time"
    
    "github.com/innovationmech/swit/pkg/messaging"
    "github.com/innovationmech/swit/pkg/saga"
    sagamessaging "github.com/innovationmech/swit/pkg/saga/messaging"
)

func main() {
    // 1. 创建 broker
    broker, _ := messaging.NewNATSBroker(messaging.BrokerConfig{
        Endpoints: []string{"nats://localhost:4222"},
    })
    defer broker.Close()
    
    // 2. 创建发布器
    publisher, _ := sagamessaging.NewSagaEventPublisher(
        broker,
        &sagamessaging.PublisherConfig{
            TopicPrefix:    "saga.events",
            SerializerType: "json",
            RetryAttempts:  3,
        },
    )
    defer publisher.Close()
    
    // 3. 发布事件
    event := &saga.SagaEvent{
        ID:        "evt-001",
        SagaID:    "saga-001",
        Type:      saga.EventSagaStarted,
        Timestamp: time.Now(),
    }
    
    ctx := context.Background()
    if err := publisher.PublishSagaEvent(ctx, event); err != nil {
        panic(err)
    }
}
```

### 基础订阅示例

```go
// 1. 创建事件处理器
handler, _ := sagamessaging.NewSagaEventHandler(&sagamessaging.EventHandlerConfig{
    HandlerID:   "order-handler",
    HandlerName: "订单处理器",
    SupportedEventTypes: []sagamessaging.SagaEventType{
        sagamessaging.EventTypeSagaStarted,
        sagamessaging.EventTypeSagaCompleted,
    },
})

// 2. 启动处理器
if err := handler.Start(ctx); err != nil {
    panic(err)
}
defer handler.Stop(ctx)

// 3. 处理事件会自动进行
```

## 核心组件

### 1. SagaEventPublisher

事件发布器负责将 Saga 事件发布到消息队列。

**主要接口**:

```go
type SagaEventPublisher interface {
    // 发布单个事件
    PublishSagaEvent(ctx context.Context, event *saga.SagaEvent) error
    
    // 批量发布事件
    PublishBatch(ctx context.Context, events []*saga.SagaEvent) error
    
    // 异步批量发布
    PublishBatchAsync(ctx context.Context, events []*saga.SagaEvent) <-chan *BatchPublishResult
    
    // 事务性发布
    PublishWithTransaction(ctx context.Context, event *saga.SagaEvent) error
    WithTransaction(ctx context.Context, txID string, fn TransactionFunc) error
    
    // 管理方法
    Close() error
    GetMetrics() *PublisherMetrics
    GetReliabilityMetrics() *ReliabilityMetrics
}
```

### 2. SagaEventHandler

事件处理器负责接收和处理 Saga 事件。

**主要接口**:

```go
type SagaEventHandler interface {
    // 处理事件
    HandleSagaEvent(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) error
    
    // 能力查询
    GetSupportedEventTypes() []SagaEventType
    CanHandle(event *saga.SagaEvent) bool
    GetPriority() int
    
    // 生命周期
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // 回调
    OnEventProcessed(ctx context.Context, event *saga.SagaEvent, duration time.Duration) error
    OnEventFailed(ctx context.Context, event *saga.SagaEvent, err error) error
}
```

### 3. ReliabilityHandler

可靠性处理器提供重试、确认和 DLQ 功能。

**主要功能**:

- 自动重试失败的发布
- 发布确认等待
- 死信队列管理
- 可靠性指标收集

### 4. EventRouter

事件路由器将事件路由到合适的处理器。

**路由策略**:

- `RouteToFirst`: 路由到第一个匹配的处理器
- `RouteToAll`: 路由到所有匹配的处理器
- `RouteByPriority`: 按优先级路由

### 5. MessageSerializer

消息序列化器支持多种格式。

**支持的格式**:

- JSON (默认)
- Protobuf (高性能)

## 配置指南

### PublisherConfig

发布器配置选项：

```go
type PublisherConfig struct {
    // 基础配置
    BrokerType      string        // broker 类型: "nats", "rabbitmq", "kafka"
    BrokerEndpoints []string      // broker 端点列表
    TopicPrefix     string        // 主题前缀
    SerializerType  string        // 序列化类型: "json", "protobuf"
    
    // 可靠性配置
    EnableConfirm   bool          // 启用发布确认
    RetryAttempts   int           // 重试次数
    RetryInterval   time.Duration // 重试间隔
    Timeout         time.Duration // 发布超时
    
    // 高级可靠性配置
    Reliability     *ReliabilityConfig
}
```

**默认值**:

- `SerializerType`: "json"
- `RetryAttempts`: 3
- `RetryInterval`: 1s
- `Timeout`: 5s

### ReliabilityConfig

可靠性配置选项：

```go
type ReliabilityConfig struct {
    // 重试配置
    EnableRetry      bool          // 启用重试
    MaxRetryAttempts int           // 最大重试次数
    RetryBackoff     time.Duration // 初始退避时间
    MaxRetryBackoff  time.Duration // 最大退避时间
    
    // 确认配置
    EnableConfirm    bool          // 启用确认
    ConfirmTimeout   time.Duration // 确认超时
    
    // DLQ 配置
    EnableDLQ        bool          // 启用死信队列
    DLQTopic         string        // DLQ 主题
    DLQRetryPolicy   DLQRetryPolicy
}
```

### EventHandlerConfig

处理器配置选项：

```go
type EventHandlerConfig struct {
    // 基础信息
    HandlerID            string
    HandlerName          string
    HandlerDescription   string
    
    // 处理能力
    SupportedEventTypes  []SagaEventType
    Priority             int
    
    // 并发配置
    MaxConcurrency       int
    ProcessingTimeout    time.Duration
    
    // 过滤器
    Filters              []EventFilter
}
```

## 使用模式

### 1. 单个事件发布

适用于发布单个独立事件。

```go
event := &saga.SagaEvent{
    ID:         "evt-001",
    SagaID:     "saga-001",
    Type:       saga.EventSagaStarted,
    Timestamp:  time.Now(),
    InstanceID: "inst-001",
}

err := publisher.PublishSagaEvent(ctx, event)
```

**适用场景**:
- Saga 生命周期事件
- 单步操作完成通知
- 错误或异常事件

### 2. 批量事件发布

适用于发布多个相关事件，提供更好的性能。

```go
events := []*saga.SagaEvent{
    // ... 多个事件
}

err := publisher.PublishBatch(ctx, events)
```

**性能提升**:
- 减少网络往返
- 批量序列化
- 批量确认

**适用场景**:
- 多步骤完成通知
- 批量状态更新
- 数据同步

### 3. 异步批量发布

适用于对延迟不敏感但需要高吞吐量的场景。

```go
resultChan := publisher.PublishBatchAsync(ctx, events)

// 可以继续做其他工作

result := <-resultChan
if result.Error != nil {
    // 处理错误
}
```

**优势**:
- 非阻塞操作
- 更高吞吐量
- 资源利用率高

### 4. 事务性发布

确保多个事件的原子性发布。

```go
err := publisher.WithTransaction(ctx, "tx-001", func(txPub *sagamessaging.TransactionalEventPublisher) error {
    // 在事务中发布多个事件
    for _, event := range events {
        if err := txPub.PublishEvent(ctx, event); err != nil {
            return err // 自动回滚
        }
    }
    return nil // 自动提交
})
```

**保证**:
- 全部成功或全部失败
- 顺序性保证
- 隔离性

**适用场景**:
- 关联事件发布
- 状态转换通知
- 分布式事务

### 5. 带重试的发布

自动重试失败的发布操作。

```go
publisher, _ := sagamessaging.NewSagaEventPublisher(
    broker,
    &sagamessaging.PublisherConfig{
        RetryAttempts: 5,
        RetryInterval: 500 * time.Millisecond,
        Reliability: &sagamessaging.ReliabilityConfig{
            EnableRetry:      true,
            MaxRetryAttempts: 5,
            RetryBackoff:     500 * time.Millisecond,
            MaxRetryBackoff:  5 * time.Second,
        },
    },
)
```

**重试策略**:
- 指数退避
- 可配置的最大重试次数
- 可配置的退避上限

### 6. 死信队列处理

处理无法成功发布的消息。

```go
&sagamessaging.ReliabilityConfig{
    EnableDLQ: true,
    DLQTopic:  "saga.dlq",
    DLQRetryPolicy: &sagamessaging.DLQRetryPolicy{
        MaxRetries:    3,
        RetryInterval: time.Minute,
        TTL:           24 * time.Hour,
    },
}

// 从 DLQ 恢复消息
dlqHandler, _ := sagamessaging.NewDeadLetterQueueHandler(config)
err := dlqHandler.RecoverFromDeadLetterQueue(ctx, messageID, originalHandler)
```

## 可靠性保证

### 1. At-Least-Once 交付

通过以下机制保证至少一次交付：

- **发布确认**: Broker 确认消息已持久化
- **自动重试**: 失败时自动重试
- **死信队列**: 保存无法处理的消息

```go
&sagamessaging.PublisherConfig{
    EnableConfirm: true,
    RetryAttempts: 5,
    Reliability: &sagamessaging.ReliabilityConfig{
        EnableRetry:   true,
        EnableConfirm: true,
        EnableDLQ:     true,
    },
}
```

### 2. 幂等性

消息处理应该是幂等的，以处理重复消息：

```go
func (h *MyHandler) HandleSagaEvent(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) error {
    // 检查事件是否已处理
    if h.isProcessed(event.ID) {
        return nil // 幂等返回
    }
    
    // 处理事件
    if err := h.processEvent(event); err != nil {
        return err
    }
    
    // 记录已处理
    h.markProcessed(event.ID)
    return nil
}
```

### 3. 顺序保证

对于需要顺序的事件，使用相同的 SagaID：

- 同一 SagaID 的事件发送到同一分区
- 使用事务性发布保证顺序
- Handler 中实现顺序检查

### 4. 超时控制

设置合理的超时避免资源泄漏：

```go
&sagamessaging.PublisherConfig{
    Timeout: 5 * time.Second,  // 发布超时
    Reliability: &sagamessaging.ReliabilityConfig{
        ConfirmTimeout: 10 * time.Second, // 确认超时
    },
}

&sagamessaging.EventHandlerConfig{
    ProcessingTimeout: 30 * time.Second, // 处理超时
}
```

## 性能优化

### 1. 批量发布

**性能对比**:
- 单个发布: ~1000 events/sec
- 批量发布: ~5000-10000 events/sec

```go
// 不推荐：逐个发布
for _, event := range events {
    publisher.PublishSagaEvent(ctx, event)
}

// 推荐：批量发布
publisher.PublishBatch(ctx, events)
```

### 2. 选择合适的序列化格式

**性能对比**:
- JSON: 易于调试，跨语言兼容
- Protobuf: 更小的消息体积（30-50%），更快的序列化（2-3x）

```go
// 开发环境
SerializerType: "json"

// 生产环境（高性能要求）
SerializerType: "protobuf"
```

### 3. 异步发布

对延迟不敏感的场景使用异步发布：

```go
// 同步（阻塞）
err := publisher.PublishBatch(ctx, events)

// 异步（非阻塞）
resultChan := publisher.PublishBatchAsync(ctx, events)
// 继续其他工作...
```

### 4. 连接池和复用

复用 broker 连接和 publisher 实例：

```go
// 不推荐：每次创建新连接
func publishEvent(event *saga.SagaEvent) {
    broker, _ := messaging.NewNATSBroker(config)
    publisher, _ := sagamessaging.NewSagaEventPublisher(broker, publisherConfig)
    publisher.PublishSagaEvent(ctx, event)
    publisher.Close()
    broker.Close()
}

// 推荐：复用连接
var (
    globalBroker    messaging.Broker
    globalPublisher *sagamessaging.SagaEventPublisher
)

func init() {
    globalBroker, _ = messaging.NewNATSBroker(config)
    globalPublisher, _ = sagamessaging.NewSagaEventPublisher(globalBroker, publisherConfig)
}

func publishEvent(event *saga.SagaEvent) {
    globalPublisher.PublishSagaEvent(ctx, event)
}
```

### 5. 调整并发度

根据系统资源调整处理并发度：

```go
&sagamessaging.EventHandlerConfig{
    MaxConcurrency: 10, // 根据 CPU 核心数和 I/O 特性调整
}
```

## 监控与指标

### 发布指标

```go
metrics := publisher.GetMetrics()

// 计数指标
metrics.PublishedCount  // 成功发布的事件数
metrics.FailedCount     // 失败的事件数
metrics.BatchCount      // 批次数

// 性能指标
metrics.AverageLatency      // 平均延迟
metrics.AverageBatchSize    // 平均批次大小
metrics.GetPublishRate()    // 发布速率 (events/sec)
```

### 可靠性指标

```go
reliabilityMetrics := publisher.GetReliabilityMetrics()

// 重试指标
reliabilityMetrics.TotalRetries      // 总重试次数
reliabilityMetrics.SuccessfulRetries // 成功的重试次数
reliabilityMetrics.FailedRetries     // 失败的重试次数

// 成功率
reliabilityMetrics.GetSuccessRate()       // 总体成功率
reliabilityMetrics.GetRetrySuccessRate()  // 重试成功率

// DLQ 指标
reliabilityMetrics.DLQMessagesCount  // DLQ 消息数
```

### 处理器指标

```go
handlerMetrics := handler.GetMetrics()

// 处理统计
handlerMetrics.TotalProcessed   // 总处理数
handlerMetrics.SuccessfulCount  // 成功数
handlerMetrics.FailedCount      // 失败数

// 性能
handlerMetrics.AverageProcessingTime  // 平均处理时间
handlerMetrics.MaxProcessingTime      // 最大处理时间
```

### Prometheus 集成

```go
import "github.com/prometheus/client_golang/prometheus"

// 注册 Prometheus 指标
prometheus.MustRegister(
    sagaEventsPublishedTotal,
    sagaEventsFailedTotal,
    sagaPublishLatency,
)

// 在 HTTP handler 中暴露指标
http.Handle("/metrics", promhttp.Handler())
```

## 最佳实践

### 1. 配置管理

使用配置文件管理不同环境：

```yaml
# config/saga-publisher.yaml
saga:
  publisher:
    topic_prefix: "saga.events"
    serializer_type: "json"
    retry_attempts: 3
    retry_interval: 1s
    timeout: 5s
    reliability:
      enable_retry: true
      max_retry_attempts: 5
      enable_dlq: true
      dlq_topic: "saga.dlq"
```

### 2. 错误处理

正确处理各种错误场景：

```go
err := publisher.PublishSagaEvent(ctx, event)
if err != nil {
    switch {
    case errors.Is(err, sagamessaging.ErrPublisherClosed):
        // 发布器已关闭，需要重新创建
    case errors.Is(err, context.DeadlineExceeded):
        // 超时，可能需要重试
    case errors.Is(err, sagamessaging.ErrInvalidEvent):
        // 事件无效，记录日志
    default:
        // 其他错误
    }
}
```

### 3. 优雅关闭

确保程序退出时正确清理资源：

```go
// 设置信号处理
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

go func() {
    <-sigChan
    log.Println("收到关闭信号，开始优雅关闭...")
    
    // 停止接受新事件
    handler.Stop(context.Background())
    
    // 等待正在处理的事件完成
    time.Sleep(5 * time.Second)
    
    // 关闭发布器
    publisher.Close()
    
    // 关闭 broker
    broker.Close()
    
    os.Exit(0)
}()
```

### 4. 日志记录

记录关键操作和错误：

```go
logger.Info("publishing saga event",
    zap.String("event_id", event.ID),
    zap.String("saga_id", event.SagaID),
    zap.String("event_type", string(event.Type)))

if err != nil {
    logger.Error("failed to publish event",
        zap.Error(err),
        zap.String("event_id", event.ID),
        zap.String("saga_id", event.SagaID))
}
```

### 5. 测试

编写完整的单元测试和集成测试：

```go
func TestPublishSagaEvent(t *testing.T) {
    tests := []struct {
        name    string
        event   *saga.SagaEvent
        wantErr bool
    }{
        {
            name: "valid event",
            event: &saga.SagaEvent{
                ID:     "evt-001",
                SagaID: "saga-001",
                Type:   saga.EventSagaStarted,
            },
            wantErr: false,
        },
        // ... 更多测试用例
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := publisher.PublishSagaEvent(context.Background(), tt.event)
            if (err != nil) != tt.wantErr {
                t.Errorf("PublishSagaEvent() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## 故障排查

### 常见问题

#### 1. 连接失败

**症状**:
```
failed to create NATS broker: dial tcp [::1]:4222: connect: connection refused
```

**排查步骤**:
1. 检查 broker 是否启动：`netstat -an | grep 4222`
2. 检查防火墙设置
3. 验证配置的端点地址
4. 查看 broker 日志

**解决方案**:
```bash
# 启动 NATS
docker run -p 4222:4222 nats:latest

# 或使用本地安装
nats-server
```

#### 2. 发布超时

**症状**:
```
context deadline exceeded
```

**排查步骤**:
1. 检查网络延迟：`ping broker-host`
2. 查看消息大小是否过大
3. 检查 broker 负载
4. 查看处理器是否阻塞

**解决方案**:
```go
// 增加超时时间
&sagamessaging.PublisherConfig{
    Timeout: 30 * time.Second,
}

// 或减小消息大小
// 或增加 broker 资源
```

#### 3. 序列化失败

**症状**:
```
failed to serialize event: json: unsupported type
```

**排查步骤**:
1. 检查事件字段类型
2. 验证 Metadata 中的值是否可序列化
3. 检查是否有循环引用

**解决方案**:
```go
// 确保 Metadata 只包含可序列化的类型
event.Metadata = map[string]interface{}{
    "order_id": "ORDER-001",  // ✓ string
    "amount":   100.0,        // ✓ number
    "items":    []string{"A", "B"}, // ✓ array
    // "channel": make(chan int), // ✗ channel 不可序列化
}
```

#### 4. 消息丢失

**排查步骤**:
1. 检查发布器配置：
   ```go
   metrics := publisher.GetMetrics()
   log.Printf("Published: %d, Failed: %d", 
       metrics.PublishedCount, metrics.FailedCount)
   ```
2. 检查 broker 持久化配置
3. 查看 DLQ 是否有消息
4. 检查订阅者是否正常运行

**解决方案**:
```go
// 启用可靠性保证
&sagamessaging.PublisherConfig{
    EnableConfirm: true,
    Reliability: &sagamessaging.ReliabilityConfig{
        EnableRetry:   true,
        EnableConfirm: true,
        EnableDLQ:     true,
    },
}
```

#### 5. 性能问题

**症状**: 发布速度慢，延迟高

**排查步骤**:
1. 检查是否使用批量发布
2. 查看序列化格式
3. 检查网络带宽
4. 分析 CPU 和内存使用

**解决方案**:
```go
// 使用批量发布
publisher.PublishBatch(ctx, events)

// 使用 Protobuf 序列化
&sagamessaging.PublisherConfig{
    SerializerType: "protobuf",
}

// 异步发布
resultChan := publisher.PublishBatchAsync(ctx, events)
```

### 调试技巧

#### 1. 启用详细日志

```go
import "github.com/innovationmech/swit/pkg/logger"

// 设置日志级别为 DEBUG
logger.SetLevel("debug")
```

#### 2. 使用跟踪

```go
import "github.com/innovationmech/swit/pkg/tracing"

// 启用分布式追踪
tracer, _ := tracing.NewJaegerTracer(config)
defer tracer.Close()
```

#### 3. 监控指标

```go
// 定期打印指标
ticker := time.NewTicker(10 * time.Second)
go func() {
    for range ticker.C {
        metrics := publisher.GetMetrics()
        log.Printf("Metrics: published=%d, failed=%d, rate=%.2f/s",
            metrics.PublishedCount,
            metrics.FailedCount,
            metrics.GetPublishRate())
    }
}()
```

## API 参考

完整的 API 文档请参考：

- [GoDoc](https://pkg.go.dev/github.com/innovationmech/swit/pkg/saga/messaging)
- [Examples](../../examples/saga-publisher/)
- [User Guide](../../docs/saga-user-guide.md)

### 主要类型

- `SagaEventPublisher` - 事件发布器接口
- `SagaEventHandler` - 事件处理器接口
- `ReliabilityHandler` - 可靠性处理器
- `EventRouter` - 事件路由器
- `MessageSerializer` - 消息序列化器
- `DeadLetterQueueHandler` - 死信队列处理器

### 配置类型

- `PublisherConfig` - 发布器配置
- `ReliabilityConfig` - 可靠性配置
- `EventHandlerConfig` - 处理器配置
- `DLQRetryPolicy` - DLQ 重试策略

### 事件类型

- `EventTypeSagaStarted` - Saga 已启动
- `EventTypeSagaStepStarted` - 步骤已启动
- `EventTypeSagaStepCompleted` - 步骤已完成
- `EventTypeSagaStepFailed` - 步骤失败
- `EventTypeSagaCompleted` - Saga 已完成
- `EventTypeSagaFailed` - Saga 失败
- `EventTypeCompensation*` - 补偿事件
- `EventTypeRetry*` - 重试事件

## 版本历史

- **v1.0.0** (2025-01) - 初始发布
  - 基础发布和订阅功能
  - JSON 序列化支持
  
- **v1.1.0** (2025-02) - 功能增强
  - 批量发布支持
  - Protobuf 序列化支持
  - 可靠性机制
  
- **v1.2.0** (2025-03) - 性能优化
  - 异步批量发布
  - 事务性消息发送
  - 死信队列处理
  
- **v1.3.0** (2025-04) - 当前版本
  - 完整的测试覆盖 (87.5%)
  - 性能优化
  - 文档完善

## 贡献

欢迎贡献！请参考 [CONTRIBUTING.md](../../../CONTRIBUTING.md)

## 许可证

MIT License - 详见 [LICENSE](../../../LICENSE)


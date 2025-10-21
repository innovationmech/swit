# Saga 消息通信协议文档

## 概述

本文档定义了 Saga 分布式事务系统中的标准消息通信协议。该协议确保跨服务通信的一致性、可靠性和互操作性。

## 协议版本

- **当前版本**: 1.1
- **最低支持版本**: 1.0
- **版本兼容性**: 相同主版本号的协议互相兼容

### 版本历史

- **1.0**: 初始协议版本，包含基本的消息结构和标准头
- **1.1**: 增强的分布式追踪支持，添加父级 Span ID 支持

## 协议结构

### SagaMessageProtocol

标准的 Saga 消息协议结构：

```go
type SagaMessageProtocol struct {
    Version        string                 // 协议版本
    MessageType    string                 // 消息类型
    SagaID         string                 // Saga 实例 ID
    EventID        string                 // 事件 ID
    EventType      string                 // 事件类型
    Timestamp      time.Time              // 时间戳
    Headers        map[string]string      // 消息头
    Payload        interface{}            // 消息体
    StepID         string                 // 步骤 ID（可选）
    CorrelationID  string                 // 关联 ID（可选）
    Source         string                 // 源服务（可选）
    Service        string                 // 目标服务（可选）
    ServiceVersion string                 // 服务版本（可选）
    TraceID        string                 // 追踪 ID（可选）
    SpanID         string                 // Span ID（可选）
    ParentSpanID   string                 // 父级 Span ID（可选）
    Attempt        int                    // 重试次数（可选）
    MaxAttempts    int                    // 最大重试次数（可选）
    Duration       time.Duration          // 持续时间（可选）
    Metadata       map[string]interface{} // 元数据（可选）
}
```

### 必填字段

以下字段在所有协议消息中必须存在：

- `Version`: 协议版本
- `MessageType`: 消息类型
- `SagaID`: Saga 实例标识符
- `EventID`: 事件标识符
- `EventType`: 事件类型
- `Timestamp`: 消息时间戳

## 标准消息头

协议定义了以下标准消息头：

| 头名称 | 键值 | 必填 | 说明 |
|-------|------|------|------|
| Saga ID | `X-Saga-ID` | 是 | Saga 实例标识符 |
| Event ID | `X-Event-ID` | 是 | 事件标识符 |
| Event Type | `X-Event-Type` | 是 | 事件类型 |
| Timestamp | `X-Timestamp` | 是 | RFC3339Nano 格式的时间戳 |
| Protocol Version | `X-Protocol-Version` | 是 | 协议版本 |
| Correlation ID | `X-Correlation-ID` | 否 | 请求关联标识符 |
| Source Service | `X-Source-Service` | 否 | 源服务名称 |
| Service | `X-Service` | 否 | 目标服务名称 |
| Service Version | `X-Service-Version` | 否 | 服务版本 |
| Step ID | `X-Step-ID` | 否 | Saga 步骤标识符 |
| Trace ID | `X-Trace-ID` | 否 | 分布式追踪 ID |
| Span ID | `X-Span-ID` | 否 | Span ID |
| Parent Span ID | `X-Parent-Span-ID` | 否 | 父级 Span ID |
| Content Type | `Content-Type` | 否 | 内容类型（如 application/json） |
| Attempt | `X-Attempt` | 否 | 当前重试次数 |
| Max Attempts | `X-Max-Attempts` | 否 | 最大重试次数 |
| Duration | `X-Duration-Ms` | 否 | 持续时间（毫秒） |

## 消息类型

协议支持以下消息类型：

- `saga.event`: Saga 事件消息
- `saga.command`: Saga 命令消息
- `saga.query`: Saga 查询消息
- `compensation.event`: 补偿事件消息

## 主题命名规则

### 事件主题

格式: `saga.events.<event_type>`

示例:
- `saga.events.saga.started` - Saga 启动事件
- `saga.events.saga.completed` - Saga 完成事件
- `saga.events.saga.step.started` - 步骤启动事件
- `saga.events.saga.step.completed` - 步骤完成事件
- `saga.events.compensation.started` - 补偿启动事件

### 命令主题

格式: `saga.commands.<command_type>`

示例:
- `saga.commands.start` - 启动 Saga 命令
- `saga.commands.cancel` - 取消 Saga 命令

### 补偿主题

格式: `saga.compensation.<event_type>`

示例:
- `saga.compensation.compensation.started` - 补偿启动
- `saga.compensation.compensation.completed` - 补偿完成

## 消息体结构

### JSON 格式示例

```json
{
  "version": "1.1",
  "message_type": "saga.event",
  "saga_id": "saga-001",
  "event_id": "evt-001",
  "event_type": "saga.started",
  "timestamp": "2025-10-20T12:00:00.123456789Z",
  "headers": {
    "X-Saga-ID": "saga-001",
    "X-Event-ID": "evt-001",
    "X-Event-Type": "saga.started",
    "X-Protocol-Version": "1.1",
    "X-Correlation-ID": "corr-001",
    "X-Source-Service": "order-service",
    "X-Trace-ID": "trace-001",
    "X-Span-ID": "span-001"
  },
  "payload": {
    "order_id": "order-123",
    "customer_id": "customer-456",
    "amount": 100.00
  },
  "correlation_id": "corr-001",
  "source": "order-service",
  "trace_id": "trace-001",
  "span_id": "span-001",
  "metadata": {
    "region": "us-east-1"
  }
}
```

## 消息格式转换

协议支持在以下格式之间进行转换：

1. **SagaEvent ↔ SagaMessageProtocol**
   - `ToProtocolMessage()`: SagaEvent → SagaMessageProtocol
   - `FromProtocolMessage()`: SagaMessageProtocol → SagaEvent

2. **SagaEvent ↔ messaging.Message**
   - `ToMessagingMessage()`: SagaEvent → messaging.Message
   - `FromMessagingMessage()`: messaging.Message → SagaEvent

3. **SagaMessageProtocol ↔ messaging.Message**
   - `ProtocolToMessaging()`: SagaMessageProtocol → messaging.Message
   - `MessagingToProtocol()`: messaging.Message → SagaMessageProtocol

## 协议验证

协议实现了多层验证机制：

### 1. 必填字段验证

验证所有必填字段是否存在且非空。

### 2. 版本验证

- 检查协议版本是否在支持的范围内
- 验证版本兼容性

### 3. 头部验证

- 检查必需的标准头是否存在
- 验证头部值的格式

### 4. 格式验证

- 验证时间戳格式（RFC3339Nano）
- 验证数值字段的范围
- 验证枚举值的有效性

## 使用示例

### 创建标准消息转换器

```go
// 创建转换器配置
config := &MessageConverterConfig{
    SerializerType:   "json",
    EnableValidation: true,
    StrictMode:       false,
}

// 创建转换器
converter, err := NewStandardMessageConverter(config)
if err != nil {
    log.Fatal(err)
}
```

### 将 SagaEvent 转换为协议消息

```go
event := &saga.SagaEvent{
    ID:            "event-123",
    SagaID:        "saga-456",
    Type:          saga.EventSagaStarted,
    Version:       "1.1",
    Timestamp:     time.Now(),
    CorrelationID: "corr-001",
    Data: map[string]interface{}{
        "order_id": "order-123",
    },
}

protocol, err := converter.ToProtocolMessage(event)
if err != nil {
    log.Fatal(err)
}
```

### 将协议消息转换为 messaging.Message

```go
msg, err := converter.ProtocolToMessaging(protocol)
if err != nil {
    log.Fatal(err)
}

// 发布消息
err = publisher.Publish(ctx, msg)
```

### 批量转换

```go
// 创建批量转换器
batch := NewBatchMessageConverter(converter)

// 批量转换事件
events := []*saga.SagaEvent{event1, event2, event3}
messages, err := batch.ToMessagingMessages(events)
if err != nil {
    log.Fatal(err)
}
```

### 构建自定义消息头

```go
headers := NewHeaderBuilder().
    WithSagaID("saga-123").
    WithEventID("event-456").
    WithEventType(string(saga.EventSagaStarted)).
    WithTimestamp(time.Now()).
    WithVersion("1.1").
    WithCorrelation("corr-789").
    WithSource("order-service").
    WithTraceID("trace-001").
    WithSpanID("span-001").
    Build()
```

### 验证协议消息

```go
validator := NewStandardProtocolValidator()

err := validator.Validate(protocol)
if err != nil {
    log.Printf("Validation failed: %v", err)
}
```

### 路由消息到正确的主题

```go
router := NewStandardTopicRouter()

// 获取事件主题
topic := router.GetTopicForEvent(saga.EventSagaStarted)
// 结果: "saga.events.saga.started"

// 解析主题信息
info, err := router.ParseTopic(topic)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Prefix: %s, Category: %s\n", info.Prefix, info.Category)
```

### 版本兼容性检查

```go
comparer := NewStandardVersionComparer()

// 检查兼容性
compatible := comparer.IsCompatible("1.0", "1.1")
// 结果: true (相同主版本号)

// 比较版本
result := comparer.Compare("1.0", "1.1")
// 结果: -1 (1.0 < 1.1)

// 检查是否支持
supported := comparer.IsSupported("1.1")
// 结果: true
```

## 最佳实践

### 1. 始终设置关联 ID

关联 ID 用于在分布式系统中跟踪请求链路，建议在所有消息中设置。

```go
event.CorrelationID = generateCorrelationID()
```

### 2. 使用分布式追踪

在启用了分布式追踪的系统中，始终传播追踪信息：

```go
event.TraceID = span.TraceID()
event.SpanID = span.SpanID()
event.ParentSpanID = span.ParentSpanID()
```

### 3. 设置服务信息

明确标识消息的源服务和目标服务：

```go
event.Source = "order-service"
event.Service = "payment-service"
event.ServiceVersion = "1.0.0"
```

### 4. 验证所有传入消息

在处理接收到的消息前，始终验证其格式和内容：

```go
if err := validator.Validate(protocol); err != nil {
    return fmt.Errorf("invalid message: %w", err)
}
```

### 5. 使用结构化的负载

使用结构化的数据格式（如 JSON 对象）而不是原始字符串：

```go
// 好的做法
event.Data = map[string]interface{}{
    "order_id": "order-123",
    "amount":   100.00,
}

// 避免
event.Data = "order-123,100.00"
```

### 6. 处理版本升级

当升级协议版本时，确保向后兼容：

```go
if protocol.Version == "1.0" {
    // 使用 1.0 版本的处理逻辑
} else if protocol.Version == "1.1" {
    // 使用 1.1 版本的处理逻辑
}
```

## 错误处理

### 常见错误及解决方案

1. **验证失败**: 检查所有必填字段是否已设置
2. **版本不兼容**: 确保协议版本在支持的范围内
3. **序列化失败**: 确保负载数据可以被正确序列化
4. **主题无效**: 验证主题名称符合命名规则

## 性能考虑

1. **消息大小**: 保持消息负载在合理大小（建议 < 1MB）
2. **序列化开销**: 对于高频消息，考虑使用 Protobuf 而不是 JSON
3. **头部数量**: 避免添加过多自定义头部
4. **批量处理**: 对于大量消息，使用批量转换器

## 安全性

1. **敏感数据**: 不要在消息头或元数据中包含敏感信息
2. **验证**: 始终验证接收到的消息
3. **加密**: 对于包含敏感数据的负载，考虑使用加密
4. **认证**: 使用服务间认证机制保护消息传输

## 参考资料

- [Saga 分布式事务实现计划](../../../specs/saga-distributed-transactions/implementation-plan.md)
- [消息系统文档](../../messaging/README.md)
- [序列化器文档](./serializer.go)
- [集成适配器文档](./integration.go)


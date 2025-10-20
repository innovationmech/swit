# Saga Event Publisher 示例

本示例演示如何使用 Saga Event Publisher 进行可靠的事件发布。

## 功能展示

1. **基础单个事件发布** - 发布单个 Saga 事件
2. **批量事件发布** - 高效地批量发布多个事件
3. **事务性消息发送** - 在事务中发布多个相关事件
4. **可靠性保证** - 使用重试和确认机制确保消息可靠投递
5. **Protobuf 序列化** - 使用 Protocol Buffers 进行高效序列化
6. **异步批量发布** - 异步方式发布事件批次

## 前置条件

运行示例前，请确保 NATS 服务器已启动：

```bash
# 使用 Docker 启动 NATS
docker run -p 4222:4222 -p 8222:8222 nats:latest

# 或使用本地安装的 NATS
nats-server
```

## 运行示例

```bash
cd examples/saga-publisher
go run main.go
```

## 配置说明

### 基础配置

```go
publisher, err := sagamessaging.NewSagaEventPublisher(
    broker,
    &sagamessaging.PublisherConfig{
        TopicPrefix:    "saga.events",    // 主题前缀
        SerializerType: "json",           // 序列化格式: json 或 protobuf
        RetryAttempts:  3,                // 重试次数
        RetryInterval:  time.Second,      // 重试间隔
        Timeout:        5 * time.Second,  // 发布超时
    },
)
```

### 可靠性配置

```go
&sagamessaging.PublisherConfig{
    EnableConfirm: true,  // 启用发布确认
    Reliability: &sagamessaging.ReliabilityConfig{
        EnableRetry:      true,                  // 启用重试
        MaxRetryAttempts: 5,                     // 最大重试次数
        RetryBackoff:     500 * time.Millisecond, // 重试退避时间
        MaxRetryBackoff:  5 * time.Second,       // 最大退避时间
        EnableDLQ:        true,                  // 启用死信队列
        DLQTopic:         "saga.dlq",            // 死信队列主题
    },
}
```

## 使用模式

### 1. 单个事件发布

```go
event := &saga.SagaEvent{
    ID:         "evt-001",
    SagaID:     "saga-001",
    Type:       saga.EventSagaStarted,
    Timestamp:  time.Now(),
}

err := publisher.PublishSagaEvent(ctx, event)
```

### 2. 批量事件发布

```go
events := []*saga.SagaEvent{
    // ... 多个事件
}

err := publisher.PublishBatch(ctx, events)
```

### 3. 事务性发布

```go
err := publisher.WithTransaction(ctx, "tx-id", func(txPub *sagamessaging.TransactionalEventPublisher) error {
    for _, event := range events {
        if err := txPub.PublishEvent(ctx, event); err != nil {
            return err // 自动回滚
        }
    }
    return nil // 自动提交
})
```

### 4. 异步批量发布

```go
resultChan := publisher.PublishBatchAsync(ctx, events)
result := <-resultChan

if result.Error != nil {
    log.Printf("发布失败: %v", result.Error)
} else {
    log.Printf("发布成功: %d 个事件", result.PublishedCount)
}
```

## 监控指标

```go
// 获取发布指标
metrics := publisher.GetMetrics()
log.Printf("成功: %d, 失败: %d, 批次: %d",
    metrics.PublishedCount,
    metrics.FailedCount,
    metrics.BatchCount)

// 获取可靠性指标
reliabilityMetrics := publisher.GetReliabilityMetrics()
log.Printf("重试次数: %d, 成功率: %.2f%%",
    reliabilityMetrics.TotalRetries,
    reliabilityMetrics.GetSuccessRate()*100)
```

## 最佳实践

1. **使用批量发布** - 当需要发布多个相关事件时，使用批量 API 提高性能
2. **启用可靠性保证** - 在生产环境启用重试和确认机制
3. **合理设置超时** - 根据网络环境和消息大小调整超时配置
4. **监控指标** - 定期检查发布指标，及时发现问题
5. **使用事务** - 对于需要原子性的多个事件，使用事务性发布
6. **选择合适的序列化格式** - JSON 便于调试，Protobuf 性能更好

## 性能优化

- **批量发布**: 比单个发布快 5-10 倍
- **Protobuf 序列化**: 比 JSON 小约 30-50%
- **异步发布**: 适合对延迟不敏感的场景
- **连接池**: 复用 broker 连接减少开销

## 故障排查

### 连接失败

```
❌ 创建 NATS broker 失败: dial tcp [::1]:4222: connect: connection refused
```

**解决方案**: 确保 NATS 服务器已启动并监听在 localhost:4222

### 发布超时

```
❌ 发布事件失败: context deadline exceeded
```

**解决方案**: 增加 Timeout 配置或检查网络连接

### 序列化失败

```
❌ 序列化失败: invalid event data
```

**解决方案**: 检查事件字段是否完整，特别是 SagaID 和事件类型

## 相关文档

- [Saga User Guide](../../docs/saga-user-guide.md)
- [Messaging Package README](../../pkg/saga/messaging/README.md)
- [API Documentation](https://pkg.go.dev/github.com/innovationmech/swit/pkg/saga/messaging)


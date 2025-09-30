# Inbox Pattern

Inbox Pattern 实现提供了一种可靠的幂等消息处理机制，通过去重存储确保消息不会被重复处理。

## 概述

Inbox Pattern 解决了分布式系统中的一个常见问题：如何确保消息的幂等性处理。该模式在处理消息前先检查消息是否已被处理过，如果已处理则直接返回，避免重复处理；如果未处理则处理消息并记录到 inbox 表中。

### 核心概念

- **InboxEntry**: 表示已处理的消息记录
- **InboxStorage**: 存储抽象接口，支持不同的数据库后端
- **InboxProcessor**: 消息处理器，提供幂等性保证
- **MessageHandler**: 实际的业务处理逻辑

## 特性

- ✅ 幂等性消息处理
- ✅ 去重存储，防止重复处理
- ✅ 支持事务性处理
- ✅ 支持多处理器场景
- ✅ 可选的结果缓存
- ✅ 自动清理历史记录
- ✅ 线程安全

## 安装

```go
import "github.com/innovationmech/swit/pkg/patterns/inbox"
```

## 快速开始

### 1. 基本使用

```go
// 创建存储
storage := inbox.NewInMemoryStorage()

// 创建处理器
type OrderHandler struct {
    name string
}

func (h *OrderHandler) Handle(ctx context.Context, msg *messaging.Message) error {
    // 业务逻辑
    fmt.Printf("Processing order: %s\n", msg.ID)
    return nil
}

func (h *OrderHandler) GetName() string {
    return h.name
}

handler := &OrderHandler{name: "order-handler"}
config := inbox.DefaultProcessorConfig()

processor, err := inbox.NewProcessor(storage, handler, config)
if err != nil {
    log.Fatal(err)
}

// 处理消息
msg := &messaging.Message{
    ID:      "msg-123",
    Topic:   "orders",
    Payload: []byte("order data"),
}

err = processor.Process(ctx, msg)
```

### 2. 事务性处理

```go
// 创建支持事务的存储
storage := inbox.NewInMemoryStorage()
txStorage := storage.(inbox.TransactionalStorage)

// 开始事务
tx, err := txStorage.BeginTx(ctx)
if err != nil {
    return err
}

// 在事务中处理消息
err = processor.ProcessWithTransaction(ctx, tx, msg)
if err != nil {
    tx.Rollback(ctx)
    return err
}

// 执行其他业务逻辑
tx.Exec(ctx, "INSERT INTO orders ...")

// 提交事务（业务数据和幂等性记录一起提交）
err = tx.Commit(ctx)
```

### 3. 带结果的处理

```go
// 创建处理器，启用结果存储
config := inbox.DefaultProcessorConfig()
config.StoreResult = true

processor, err := inbox.NewProcessor(storage, handler, config)

// 处理消息并获取结果
result, err := processor.ProcessWithResult(ctx, msg)
if err != nil {
    log.Fatal(err)
}

// 第二次调用会返回缓存的结果
result2, err := processor.ProcessWithResult(ctx, msg)
// result2 与 result 相同，且处理器不会被重复调用
```

## 架构

```
┌─────────────────┐
│  Message        │
│  Broker         │
└────────┬────────┘
         │
         │ 1. Receive Message
         ▼
┌─────────────────┐
│ Inbox Processor │
│                 │
│ ┌─────────────┐ │
│ │  Check if   │ │  2. Query Inbox
│ │  Processed  │ │
│ └─────────────┘ │
└────────┬────────┘
         │
         │ 3a. If Not Processed
         ▼
┌─────────────────┐
│   Database      │
│  ┌───────────┐  │
│  │ Business  │  │  4. Save Business Data
│  │   Data    │  │
│  └───────────┘  │
│  ┌───────────┐  │
│  │  Inbox    │  │  5. Record as Processed
│  │   Table   │  │
│  └───────────┘  │
└─────────────────┘
         │
         │ 3b. If Already Processed
         │
         └──────────► Return (Idempotent)
```

## 接口

### InboxStorage

存储接口定义了基本的 CRUD 操作：

```go
type InboxStorage interface {
    RecordMessage(ctx context.Context, entry *InboxEntry) error
    IsProcessed(ctx context.Context, messageID string) (bool, *InboxEntry, error)
    IsProcessedByHandler(ctx context.Context, messageID, handlerName string) (bool, *InboxEntry, error)
    DeleteBefore(ctx context.Context, before time.Time) error
    DeleteByMessageID(ctx context.Context, messageID string) error
}
```

### MessageHandler

消息处理器接口：

```go
type MessageHandler interface {
    Handle(ctx context.Context, msg *messaging.Message) error
    GetName() string
}
```

### MessageHandlerWithResult

带返回值的消息处理器接口：

```go
type MessageHandlerWithResult interface {
    MessageHandler
    HandleWithResult(ctx context.Context, msg *messaging.Message) (interface{}, error)
}
```

## 配置

### ProcessorConfig

| 字段 | 类型 | 默认值 | 说明 |
|-----|------|-------|------|
| HandlerName | string | handler.GetName() | 处理器名称 |
| EnableAutoCleanup | bool | false | 是否启用自动清理 |
| CleanupInterval | time.Duration | 24h | 清理间隔 |
| CleanupAfter | time.Duration | 168h (7天) | 清理多久前的记录 |
| StoreResult | bool | false | 是否存储处理结果 |

## 存储实现

### 内存存储

用于测试和开发：

```go
storage := inbox.NewInMemoryStorage()
```

### SQL 数据库存储

生产环境建议使用 SQL 数据库存储。需要创建如下表结构：

```sql
CREATE TABLE inbox (
    message_id VARCHAR(255) NOT NULL,
    handler_name VARCHAR(255) NOT NULL,
    topic VARCHAR(255) NOT NULL,
    processed_at TIMESTAMP NOT NULL,
    result BLOB,
    metadata TEXT,
    PRIMARY KEY (message_id, handler_name),
    INDEX idx_processed_at (processed_at)
);
```

## 使用场景

### 1. 防止重复处理

```go
// 当消息可能被重复投递时，使用 Inbox Pattern 确保幂等性
processor.Process(ctx, msg)
```

### 2. 多处理器场景

```go
// 同一消息可以被多个处理器处理
orderProcessor := inbox.NewProcessor(storage, orderHandler, orderConfig)
notificationProcessor := inbox.NewProcessor(storage, notificationHandler, notificationConfig)

// 两个处理器可以独立处理同一消息
orderProcessor.Process(ctx, msg)
notificationProcessor.Process(ctx, msg)
```

### 3. 结果缓存

```go
// 对于计算密集型的处理，可以缓存结果
config.StoreResult = true
processor := inbox.NewProcessor(storage, handler, config)

// 第一次调用会处理并缓存结果
result, _ := processor.ProcessWithResult(ctx, msg)

// 后续调用直接返回缓存的结果
cachedResult, _ := processor.ProcessWithResult(ctx, msg)
```

### 4. 事务一致性

```go
// 确保业务数据和幂等性记录的原子性
tx, _ := txStorage.BeginTx(ctx)

// 业务逻辑
processor.ProcessWithTransaction(ctx, tx, msg)
tx.Exec(ctx, "INSERT INTO orders ...")

// 一起提交
tx.Commit(ctx)
```

## 最佳实践

1. **消息ID唯一性**: 确保消息ID全局唯一，通常使用 UUID
2. **处理器命名**: 为处理器设置有意义的名称，特别是在多处理器场景中
3. **定期清理**: 启用自动清理或定期手动清理历史记录，避免存储无限增长
4. **监控**: 监控 inbox 表的大小和处理延迟
5. **错误处理**: 处理器应该正确区分业务错误和技术错误
6. **事务边界**: 在需要原子性保证时使用事务

## 与 Outbox Pattern 的区别

| 特性 | Inbox Pattern | Outbox Pattern |
|-----|--------------|----------------|
| 目的 | 防止消息重复处理 | 确保消息可靠发布 |
| 位置 | 消息消费端 | 消息生产端 |
| 主要功能 | 去重、幂等性 | 事务性发布 |
| 存储内容 | 已处理的消息ID | 待发布的消息 |
| 使用时机 | 处理消息前 | 发布消息时 |

## 性能考虑

1. **索引**: 在 message_id 和 handler_name 上创建索引
2. **分区**: 对于大量消息，考虑按时间分区
3. **缓存**: 可以使用 Redis 等缓存来加速幂等性检查
4. **批量清理**: 使用批量删除来提高清理效率

## 示例

更多示例请参考 [example_test.go](example_test.go)。

## 常见问题

### Q: 消息ID应该如何生成？

A: 消息ID应该是全局唯一的。对于外部消息，使用消息代理提供的ID；对于内部消息，使用 UUID 或其他唯一标识符。

### Q: 如何处理消息ID冲突？

A: Inbox Pattern 依赖于消息ID的唯一性。如果消息ID可能冲突，可以使用复合键，例如 `{topic}:{partition}:{offset}`。

### Q: 是否支持消息过期？

A: 支持。配置 `EnableAutoCleanup` 和 `CleanupAfter` 来自动清理过期记录。

### Q: 处理失败的消息如何重试？

A: Inbox Pattern 关注幂等性，不负责重试。重试逻辑应该在消息代理或应用层实现。

### Q: 是否可以和 Outbox Pattern 一起使用？

A: 可以。Inbox Pattern 在消费端保证幂等性，Outbox Pattern 在生产端保证可靠性，两者互补。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License

# Outbox Pattern

Outbox Pattern 实现提供了一种可靠的事件发布机制，确保业务数据更新和事件发布的原子性。

## 概述

Outbox Pattern 解决了分布式系统中的一个常见问题：如何确保数据库操作和消息发布的原子性。该模式将待发布的事件存储在与业务数据相同的数据库中，然后由后台 worker 异步发布这些事件。

### 核心概念

- **OutboxEntry**: 表示一个待发布的事件
- **OutboxStorage**: 存储抽象接口，支持不同的数据库后端
- **OutboxPublisher**: 用于保存事件到 outbox
- **OutboxProcessor**: 后台处理器，负责从 outbox 中获取并发布事件

## 特性

- ✅ 事务性消息发布
- ✅ 数据库抽象，支持多种存储后端
- ✅ 自动重试机制
- ✅ 批量处理
- ✅ 并发 worker 支持
- ✅ 自动清理已处理消息
- ✅ 线程安全

## 安装

```go
import "github.com/innovationmech/swit/pkg/patterns/outbox"
```

## 快速开始

### 1. 基本使用

```go
// 创建存储和发布器
storage := outbox.NewInMemoryStorage()
publisher := outbox.NewPublisher(storage)

// 创建事件
entry := &outbox.OutboxEntry{
    ID:          uuid.NewString(),
    AggregateID: "order-123",
    EventType:   "order.created",
    Topic:       "orders.events",
    Payload:     eventData,
}

// 保存到 outbox
err := publisher.SaveForPublish(ctx, entry)
```

### 2. 事务性发布

```go
// 开始事务
tx, err := storage.BeginTx(ctx)
if err != nil {
    return err
}

// 保存业务数据
tx.Exec(ctx, "INSERT INTO orders ...")

// 在同一事务中保存事件
err = publisher.SaveWithTransaction(ctx, tx, entry)
if err != nil {
    tx.Rollback(ctx)
    return err
}

// 提交事务（业务数据和事件一起提交）
err = tx.Commit(ctx)
```

### 3. 启动后台处理器

```go
// 配置处理器
config := outbox.ProcessorConfig{
    PollInterval:    5 * time.Second,
    BatchSize:       100,
    MaxRetries:      3,
    WorkerCount:     1,
    CleanupInterval: 24 * time.Hour,
    CleanupAfter:    7 * 24 * time.Hour,
}

// 创建处理器
processor := outbox.NewProcessor(storage, messagePublisher, config)

// 启动处理器
err := processor.Start(ctx)

// 在应用关闭时停止处理器
defer processor.Stop(ctx)
```

## 架构

```
┌─────────────────┐
│  Application    │
│   Logic         │
└────────┬────────┘
         │
         │ 1. Begin Transaction
         ▼
┌─────────────────┐
│   Database      │
│  ┌───────────┐  │
│  │ Business  │  │  2. Save Business Data
│  │   Data    │  │
│  └───────────┘  │
│  ┌───────────┐  │
│  │  Outbox   │  │  3. Save Event
│  │   Table   │  │
│  └───────────┘  │
└────────┬────────┘
         │
         │ 4. Commit Transaction
         ▼
┌─────────────────┐
│ Outbox Worker   │  5. Poll & Publish
│  (Background)   │
└────────┬────────┘
         │
         │ 6. Publish to Broker
         ▼
┌─────────────────┐
│ Message Broker  │
│  (Kafka/NATS)   │
└─────────────────┘
```

## 接口

### OutboxStorage

存储接口定义了基本的 CRUD 操作：

```go
type OutboxStorage interface {
    Save(ctx context.Context, entry *OutboxEntry) error
    SaveBatch(ctx context.Context, entries []*OutboxEntry) error
    FetchUnprocessed(ctx context.Context, limit int) ([]*OutboxEntry, error)
    MarkAsProcessed(ctx context.Context, id string) error
    MarkAsFailed(ctx context.Context, id string, errMsg string) error
    IncrementRetry(ctx context.Context, id string) error
    Delete(ctx context.Context, id string) error
    DeleteProcessedBefore(ctx context.Context, before time.Time) error
}
```

### TransactionalStorage

支持事务的存储接口：

```go
type TransactionalStorage interface {
    OutboxStorage
    BeginTx(ctx context.Context) (Transaction, error)
}
```

## 配置

### ProcessorConfig

| 字段 | 类型 | 默认值 | 说明 |
|-----|------|-------|------|
| PollInterval | time.Duration | 5s | 轮询间隔 |
| BatchSize | int | 100 | 每批处理的消息数 |
| MaxRetries | int | 3 | 最大重试次数 |
| WorkerCount | int | 1 | 并发 worker 数量 |
| CleanupInterval | time.Duration | 24h | 清理间隔 |
| CleanupAfter | time.Duration | 168h (7天) | 清理多久前的消息 |

## 存储实现

### 内存存储

用于测试和开发：

```go
storage := outbox.NewInMemoryStorage()
```

### SQL 数据库存储

生产环境建议使用 SQL 数据库存储。需要创建如下表结构：

```sql
CREATE TABLE outbox (
    id VARCHAR(255) PRIMARY KEY,
    aggregate_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    topic VARCHAR(255) NOT NULL,
    payload BLOB NOT NULL,
    headers TEXT,
    created_at TIMESTAMP NOT NULL,
    processed_at TIMESTAMP NULL,
    retry_count INT DEFAULT 0,
    last_error TEXT,
    INDEX idx_processed_at (processed_at),
    INDEX idx_created_at (created_at)
);
```

## 最佳实践

1. **使用事务**: 始终在业务事务中保存 outbox 条目，确保原子性
2. **幂等性**: 消费者应该实现幂等处理，因为消息可能被重复投递
3. **监控**: 监控 outbox 表的大小和未处理消息的数量
4. **清理策略**: 定期清理已处理的消息，避免表过大
5. **重试策略**: 根据业务需求配置合适的重试次数和间隔

## 错误处理

- **暂时性错误**: 处理器会自动重试（直到达到 MaxRetries）
- **永久性错误**: 超过最大重试次数后，消息会被标记为失败
- **监控失败消息**: 定期检查失败的消息并手动处理

## 测试

运行测试：

```bash
go test ./pkg/patterns/outbox/... -v
```

运行带覆盖率的测试：

```bash
go test ./pkg/patterns/outbox/... -cover
```

## 依赖

- `github.com/innovationmech/swit/pkg/messaging` - 消息发布接口
- `github.com/google/uuid` - UUID 生成

## 参考资料

- [Microservices Patterns: Transactional Outbox](https://microservices.io/patterns/data/transactional-outbox.html)
- [Event-Driven Architecture Patterns](https://docs.microsoft.com/en-us/azure/architecture/patterns/event-sourcing)

## 许可证

MIT License - 查看 [LICENSE](../../../LICENSE) 文件了解详情

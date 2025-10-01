# Distributed Transaction Coordinator

分布式事务协调器提供了两种可插拔的事务协调策略：2PC（两阶段提交）和 Compensation（补偿型事务）。

## 概述

分布式事务协调器解决了在分布式系统中保持多个服务或资源一致性的问题。它支持两种事务模式：

- **2PC (Two-Phase Commit)**：适用于需要强一致性的场景，提供 ACID 保证
- **Compensation**：适用于长事务和松耦合系统，通过补偿操作撤销已执行的操作

## 核心概念

### TransactionParticipant（事务参与者）

所有参与分布式事务的组件需要实现此接口：

- `Prepare(ctx, txID)`: 准备阶段，确认可以提交（仅 2PC 使用）
- `Commit(ctx, txID)`: 提交阶段，使更改永久生效
- `Rollback(ctx, txID)`: 回滚阶段，撤销所有更改

### TransactionStrategy（事务策略）

可插拔的事务协调策略：

- **TwoPhaseCommitStrategy**: 两阶段提交协议
- **CompensationStrategy**: 补偿型事务

### TransactionCoordinator（事务协调器）

协调分布式事务的执行，管理参与者的生命周期。

## 安装

```go
import "github.com/innovationmech/swit/pkg/patterns/transaction"
```

## 快速开始

### 1. 实现事务参与者

```go
// 数据库参与者
type DatabaseParticipant struct {
    db     *sql.DB
    txData map[string]*sql.Tx
    mu     sync.Mutex
}

func (d *DatabaseParticipant) Prepare(ctx context.Context, txID string) error {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    // 开始数据库事务
    tx, err := d.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    
    // 保存事务以便后续提交或回滚
    d.txData[txID] = tx
    
    return nil
}

func (d *DatabaseParticipant) Commit(ctx context.Context, txID string) error {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    tx, exists := d.txData[txID]
    if !exists {
        // Compensation 策略下，直接执行操作
        tx, err := d.db.BeginTx(ctx, nil)
        if err != nil {
            return err
        }
        // 执行数据库操作
        // ...
        return tx.Commit()
    }
    
    // 2PC 策略下，提交已准备的事务
    defer delete(d.txData, txID)
    return tx.Commit()
}

func (d *DatabaseParticipant) Rollback(ctx context.Context, txID string) error {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    tx, exists := d.txData[txID]
    if exists {
        defer delete(d.txData, txID)
        return tx.Rollback()
    }
    
    // Compensation 策略下，执行补偿操作
    // 例如：删除已插入的记录
    // ...
    
    return nil
}

func (d *DatabaseParticipant) GetName() string {
    return "database"
}
```

### 2. 使用 2PC 策略

```go
// 创建配置
config := &transaction.TransactionConfig{
    Timeout:         30 * time.Second,
    MaxRetries:      3,
    RetryDelay:      time.Second,
    EnableLogging:   true,
    DefaultStrategy: "2pc",
}

// 创建协调器
coordinator := transaction.NewCoordinator(config)

// 定义参与者
participants := []transaction.TransactionParticipant{
    &DatabaseParticipant{db: orderDB},
    &MessagingParticipant{publisher: eventPublisher},
    &CacheParticipant{cache: redis},
}

// 执行事务
err := coordinator.BeginAndExecute(ctx, participants, func() error {
    // 业务逻辑
    order := &Order{
        ID:     "order-123",
        Status: "completed",
    }
    
    // 这些操作将在所有参与者准备好后提交
    if err := saveOrder(order); err != nil {
        return err
    }
    
    if err := publishOrderEvent(order); err != nil {
        return err
    }
    
    if err := updateCache(order); err != nil {
        return err
    }
    
    return nil
})

if err != nil {
    log.Error("Transaction failed:", err)
}
```

### 3. 使用补偿策略

```go
// 创建配置
config := &transaction.TransactionConfig{
    Timeout:         30 * time.Second,
    MaxRetries:      3,
    RetryDelay:      time.Second,
    EnableLogging:   true,
    DefaultStrategy: "compensation",
}

// 创建协调器
coordinator := transaction.NewCoordinator(config)

// 执行事务（无需 Prepare 阶段）
err := coordinator.BeginAndExecute(ctx, participants, func() error {
    // 业务逻辑
    return processLongRunningOperation()
})
```

### 4. 动态切换策略

```go
// 创建协调器（默认 2PC）
coordinator := transaction.NewCoordinator(config)

// 对于短事务，使用 2PC
twoPCStrategy := transaction.NewTwoPhaseCommitStrategy(config)
coordinator.SetStrategy(twoPCStrategy)

err := coordinator.BeginAndExecute(ctx, participants, shortOperation)

// 对于长事务，切换到补偿策略
compensationStrategy := transaction.NewCompensationStrategy(config)
coordinator.SetStrategy(compensationStrategy)

err = coordinator.BeginAndExecute(ctx, participants, longOperation)
```

## 使用场景

### 场景 1：订单处理系统

```go
// 订单服务参与者
type OrderServiceParticipant struct {
    orderRepo OrderRepository
    preparedOrders map[string]*Order
}

func (o *OrderServiceParticipant) Prepare(ctx context.Context, txID string) error {
    // 验证订单可以创建
    order := getCurrentOrder(ctx)
    if err := o.orderRepo.Validate(order); err != nil {
        return err
    }
    o.preparedOrders[txID] = order
    return nil
}

func (o *OrderServiceParticipant) Commit(ctx context.Context, txID string) error {
    order := o.preparedOrders[txID]
    defer delete(o.preparedOrders, txID)
    return o.orderRepo.Create(order)
}

func (o *OrderServiceParticipant) Rollback(ctx context.Context, txID string) error {
    delete(o.preparedOrders, txID)
    return nil
}

// 库存服务参与者
type InventoryServiceParticipant struct {
    inventoryClient InventoryClient
    reservations map[string]string // txID -> reservationID
}

func (i *InventoryServiceParticipant) Prepare(ctx context.Context, txID string) error {
    // 预留库存
    reservationID, err := i.inventoryClient.Reserve(ctx, getOrderItems(ctx))
    if err != nil {
        return err
    }
    i.reservations[txID] = reservationID
    return nil
}

func (i *InventoryServiceParticipant) Commit(ctx context.Context, txID string) error {
    // 确认库存预留
    reservationID := i.reservations[txID]
    defer delete(i.reservations, txID)
    return i.inventoryClient.Confirm(ctx, reservationID)
}

func (i *InventoryServiceParticipant) Rollback(ctx context.Context, txID string) error {
    // 取消库存预留
    if reservationID, exists := i.reservations[txID]; exists {
        defer delete(i.reservations, txID)
        return i.inventoryClient.Cancel(ctx, reservationID)
    }
    return nil
}

// 使用协调器
func ProcessOrder(ctx context.Context, order *Order) error {
    coordinator := transaction.NewCoordinator(transaction.DefaultTransactionConfig())
    
    participants := []transaction.TransactionParticipant{
        &OrderServiceParticipant{orderRepo: orderRepo},
        &InventoryServiceParticipant{inventoryClient: inventoryClient},
        &PaymentServiceParticipant{paymentClient: paymentClient},
    }
    
    return coordinator.BeginAndExecute(ctx, participants, func() error {
        // 业务逻辑在这里执行
        log.Info("Processing order", "order_id", order.ID)
        return nil
    })
}
```

### 场景 2：Saga 模式集成

补偿策略可以与现有的 Saga 模式配合使用：

```go
import (
    "github.com/innovationmech/swit/pkg/patterns/saga"
    "github.com/innovationmech/swit/pkg/patterns/transaction"
)

// Saga 步骤适配器
type SagaStepAdapter struct {
    step saga.SagaStep
}

func (s *SagaStepAdapter) Prepare(ctx context.Context, txID string) error {
    // Saga 不需要 Prepare
    return nil
}

func (s *SagaStepAdapter) Commit(ctx context.Context, txID string) error {
    _, err := s.step.Handler(ctx, nil)
    return err
}

func (s *SagaStepAdapter) Rollback(ctx context.Context, txID string) error {
    if s.step.Compensation != nil {
        return s.step.Compensation(ctx, nil)
    }
    return nil
}

func (s *SagaStepAdapter) GetName() string {
    return s.step.Name
}

// 使用 Saga 与事务协调器
func ExecuteSagaWithCoordinator(ctx context.Context, sagaDef *saga.SagaDefinition) error {
    // 将 Saga 步骤转换为事务参与者
    participants := make([]transaction.TransactionParticipant, len(sagaDef.Steps))
    for i, step := range sagaDef.Steps {
        participants[i] = &SagaStepAdapter{step: step}
    }
    
    // 使用补偿策略
    config := transaction.DefaultTransactionConfig()
    config.DefaultStrategy = "compensation"
    
    coordinator := transaction.NewCoordinator(config)
    
    return coordinator.BeginAndExecute(ctx, participants, func() error {
        return nil
    })
}
```

## 配置选项

```go
type TransactionConfig struct {
    // Timeout 事务超时时间
    Timeout time.Duration
    
    // MaxRetries 最大重试次数
    MaxRetries int
    
    // RetryDelay 重试延迟
    RetryDelay time.Duration
    
    // EnableLogging 是否启用日志
    EnableLogging bool
    
    // DefaultStrategy 默认协调策略 ("2pc" or "compensation")
    DefaultStrategy string
}
```

## 2PC vs Compensation

| 特性 | 2PC | Compensation |
|------|-----|--------------|
| 一致性 | 强一致性 | 最终一致性 |
| Prepare 阶段 | 需要 | 不需要 |
| 性能 | 较慢（两阶段） | 较快（单阶段） |
| 适用场景 | 短事务、强一致性需求 | 长事务、松耦合系统 |
| 阻塞 | 可能阻塞资源 | 不阻塞资源 |
| 回滚方式 | 直接回滚 | 补偿操作 |

## 最佳实践

### 1. 选择合适的策略

- **使用 2PC** 当：
  - 事务执行时间短（< 5秒）
  - 需要强一致性保证
  - 参与者支持事务锁定
  - 可以接受资源阻塞

- **使用 Compensation** 当：
  - 事务执行时间长
  - 可以接受最终一致性
  - 跨多个独立服务
  - 需要高可用性

### 2. 实现幂等性

确保 Commit 和 Rollback 操作是幂等的，可以安全重试：

```go
func (p *MyParticipant) Commit(ctx context.Context, txID string) error {
    // 检查是否已提交
    if p.isCommitted(txID) {
        return nil // 幂等
    }
    
    // 执行提交
    err := p.doCommit(ctx)
    if err != nil {
        return err
    }
    
    // 记录已提交
    p.markCommitted(txID)
    return nil
}
```

### 3. 超时设置

设置合理的超时时间，避免事务长时间挂起：

```go
config := &transaction.TransactionConfig{
    Timeout:    30 * time.Second, // 根据业务调整
    MaxRetries: 3,
    RetryDelay: time.Second,
}
```

### 4. 错误处理

区分临时错误和永久错误，对临时错误进行重试：

```go
func (p *MyParticipant) Commit(ctx context.Context, txID string) error {
    err := p.executeOperation(ctx)
    
    if isTemporaryError(err) {
        // 返回错误，让协调器重试
        return err
    }
    
    if isPermanentError(err) {
        // 永久错误，记录日志
        log.Error("Permanent error", err)
        return err
    }
    
    return nil
}
```

### 5. 日志和监控

启用日志并集成监控系统：

```go
type PrometheusLogger struct {
    logger *zap.Logger
}

func (l *PrometheusLogger) Info(msg string, args ...interface{}) {
    l.logger.Info(msg, argsToZapFields(args)...)
    // 记录 Prometheus 指标
    transactionCounter.Inc()
}

coordinator := transaction.NewCoordinator(config)
coordinator.SetLogger(&PrometheusLogger{logger: zapLogger})
```

## 故障处理

### 协调器故障

如果协调器在执行过程中崩溃：

- **2PC**: 参与者需要实现超时机制，在超时后回滚事务
- **Compensation**: 已执行的操作会保留，需要通过外部机制触发补偿

### 参与者故障

如果参与者在执行过程中崩溃：

- **2PC**: 事务会回滚，所有参与者的更改被撤销
- **Compensation**: 已执行的参与者会被补偿

### 网络分区

在网络分区情况下：

- **2PC**: 可能导致阻塞，需要设置合理的超时
- **Compensation**: 更能容忍网络问题，但可能出现不一致窗口

## 性能考虑

1. **并发控制**: 协调器内部使用读写锁，支持高并发
2. **内存管理**: 定期清理已完成的事务记录

```go
// 定期清理
go func() {
    ticker := time.NewTicker(10 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        // 清理 1 小时前完成的事务
        count := coordinator.CleanupCompletedTransactions(time.Hour)
        log.Info("Cleaned up transactions", "count", count)
    }
}()
```

3. **批量操作**: 对于批量事务，考虑使用工作池

## 测试

运行测试：

```bash
# 运行所有测试
go test ./pkg/patterns/transaction -v

# 运行基准测试
go test ./pkg/patterns/transaction -bench=. -benchmem

# 运行特定测试
go test ./pkg/patterns/transaction -run=TestTwoPhaseCommitSuccess -v
```

## 常见问题

### Q: 2PC 和 Saga 的区别？

A: 2PC 是一种同步的强一致性协议，需要两个阶段（Prepare 和 Commit）。Saga 是一种基于补偿的异步模式，不需要 Prepare 阶段。我们的 Compensation 策略类似于 Saga。

### Q: 如何处理部分提交失败？

A: 在 2PC 中，如果 Commit 阶段部分失败，会尝试回滚所有参与者。在 Compensation 中，会对已执行的操作进行补偿。

### Q: 是否支持嵌套事务？

A: 当前版本不支持嵌套事务。每个事务应该是独立的。

### Q: 如何与现有的 Saga 模式集成？

A: 可以使用 `SagaStepAdapter` 将 Saga 步骤转换为事务参与者，使用 Compensation 策略执行。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License


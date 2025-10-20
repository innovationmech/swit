# Saga 分布式事务用户指南

## 概述

Swit 框架的 Saga 分布式事务系统提供了强大而灵活的分布式事务管理能力。本指南帮助您快速上手并掌握 Saga 的核心概念和使用方法。

## 核心概念

### 什么是 Saga？

Saga 是一种管理分布式事务的设计模式，它将长事务分解为一系列本地事务（步骤）。每个步骤都有对应的补偿操作，当某个步骤失败时，系统会执行已完成步骤的补偿操作，从而保证最终一致性。

### Saga 模式的优势

1. **避免分布式锁**：不需要两阶段提交（2PC）
2. **更好的性能**：步骤可以独立提交，不阻塞其他操作
3. **更高的可用性**：即使部分服务暂时不可用，也能继续执行
4. **灵活的补偿策略**：可以根据业务需求定制补偿逻辑

### 核心组件

- **SagaCoordinator**: 协调器，管理 Saga 实例的生命周期
- **SagaDefinition**: Saga 定义，包含步骤、重试策略、超时配置等
- **SagaStep**: Saga 步骤，包含执行逻辑和补偿逻辑
- **SagaInstance**: Saga 实例，表示一个运行中的 Saga
- **StateStorage**: 状态存储，持久化 Saga 状态
- **EventPublisher**: 事件发布器，发布 Saga 生命周期事件

## 快速开始

### 1. 安装依赖

```bash
go get github.com/innovationmech/swit
```

### 2. 创建第一个 Saga

#### 定义 Saga 步骤

```go
package main

import (
    "context"
    "time"
    "github.com/innovationmech/swit/pkg/saga"
)

// 步骤 1: 创建订单
type CreateOrderStep struct {
    orderService OrderService
}

func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 执行业务逻辑
    order, err := s.orderService.CreateOrder(ctx, orderData)
    if err != nil {
        return nil, err
    }
    
    // 返回结果
    orderData.OrderID = order.ID
    return orderData, nil
}

func (s *CreateOrderStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    // 补偿逻辑：取消订单
    return s.orderService.CancelOrder(ctx, orderData.OrderID)
}

func (s *CreateOrderStep) GetID() string { return "create-order" }
func (s *CreateOrderStep) GetName() string { return "创建订单" }
func (s *CreateOrderStep) GetDescription() string { return "在订单服务中创建新订单" }
func (s *CreateOrderStep) GetTimeout() time.Duration { return 5 * time.Second }
func (s *CreateOrderStep) GetRetryPolicy() saga.RetryPolicy { return nil }
func (s *CreateOrderStep) IsRetryable(err error) bool { return true }
func (s *CreateOrderStep) GetMetadata() map[string]interface{} { return nil }
```

#### 创建 Saga 定义

```go
func NewOrderSagaDefinition() saga.SagaDefinition {
    return &OrderSagaDefinition{
        id:   "order-processing",
        name: "订单处理",
        steps: []saga.SagaStep{
            NewCreateOrderStep(),
            NewReserveInventoryStep(),
            NewProcessPaymentStep(),
            NewConfirmOrderStep(),
        },
        timeout:     30 * time.Minute,
        retryPolicy: saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
        strategy:    saga.NewSequentialCompensationStrategy(30 * time.Second),
    }
}
```

#### 初始化 Coordinator

```go
func main() {
    // 创建状态存储
    stateStorage := coordinator.NewInMemoryStateStorage()
    
    // 创建事件发布器
    eventPublisher := coordinator.NewInMemoryEventPublisher()
    
    // 配置 Coordinator
    config := &coordinator.OrchestratorConfig{
        StateStorage:   stateStorage,
        EventPublisher: eventPublisher,
        RetryPolicy:    saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
    }
    
    // 创建 Coordinator
    sagaCoordinator, err := coordinator.NewOrchestratorCoordinator(config)
    if err != nil {
        log.Fatal(err)
    }
    defer sagaCoordinator.Close()
    
    // 使用 Coordinator
    runSaga(sagaCoordinator)
}
```

#### 启动 Saga

```go
func runSaga(coordinator saga.SagaCoordinator) {
    definition := NewOrderSagaDefinition()
    
    orderData := &OrderData{
        CustomerID: "CUST-123",
        Items:      []OrderItem{{ProductID: "PROD-001", Quantity: 1}},
        TotalAmount: 99.99,
    }
    
    ctx := context.Background()
    instance, err := coordinator.StartSaga(ctx, definition, orderData)
    if err != nil {
        log.Printf("启动 Saga 失败: %v", err)
        return
    }
    
    log.Printf("Saga 已启动: %s", instance.GetID())
}
```

## 核心功能详解

### 1. 步骤执行

Saga 按顺序执行所有步骤，每个步骤的输出作为下一个步骤的输入。

```go
func (s *MyStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 1. 类型断言获取输入数据
    inputData := data.(*MyData)
    
    // 2. 执行业务逻辑
    result, err := s.doWork(ctx, inputData)
    if err != nil {
        return nil, err
    }
    
    // 3. 更新数据并返回
    inputData.Result = result
    return inputData, nil
}
```

### 2. 错误处理和重试

#### 定义重试策略

```go
// 指数退避策略（推荐）
retryPolicy := saga.NewExponentialBackoffRetryPolicy(
    5,              // 最多重试 5 次
    time.Second,    // 基础延迟 1 秒
    time.Minute,    // 最大延迟 1 分钟
)

// 固定延迟策略
retryPolicy := saga.NewFixedDelayRetryPolicy(
    3,              // 最多重试 3 次
    time.Second*5,  // 每次延迟 5 秒
)

// 线性退避策略
retryPolicy := saga.NewLinearBackoffRetryPolicy(
    4,              // 最多重试 4 次
    time.Second,    // 基础延迟 1 秒
    time.Second*2,  // 每次增加 2 秒
    time.Second*30, // 最大延迟 30 秒
)
```

#### 区分可重试错误和不可重试错误

```go
func (s *MyStep) IsRetryable(err error) bool {
    // 网络错误、超时错误可重试
    if IsNetworkError(err) || IsTimeoutError(err) {
        return true
    }
    
    // 业务错误、验证错误不应重试
    if IsBusinessError(err) || IsValidationError(err) {
        return false
    }
    
    return false
}

func (s *MyStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    result, err := s.doWork(ctx, data)
    if err != nil {
        if IsTemporaryError(err) {
            // 标记为可重试错误
            return nil, saga.NewRetryableError(err)
        }
        // 标记为不可重试的业务错误
        return nil, saga.NewBusinessError("INVALID_REQUEST", err.Error())
    }
    return result, nil
}
```

### 3. 补偿逻辑

#### 实现补偿操作

```go
func (s *MyStep) Compensate(ctx context.Context, data interface{}) error {
    // 1. 获取需要补偿的数据
    inputData := data.(*MyData)
    
    // 2. 执行补偿操作（撤销步骤的影响）
    err := s.undoWork(ctx, inputData)
    if err != nil {
        // 记录补偿失败，但不返回错误
        // 这允许其他步骤继续补偿
        log.Printf("补偿失败: %v", err)
    }
    
    // 3. 返回 nil 允许继续补偿流程
    return nil
}
```

#### 选择补偿策略

```go
// 顺序补偿（默认）：逆序执行补偿
strategy := saga.NewSequentialCompensationStrategy(30 * time.Second)

// 并行补偿：同时执行所有补偿
strategy := saga.NewParallelCompensationStrategy(30 * time.Second)

// 尽力而为补偿：即使部分失败也继续
strategy := saga.NewBestEffortCompensationStrategy(30 * time.Second)

// 自定义补偿策略
strategy := saga.NewCustomCompensationStrategy(
    30 * time.Second,
    func(err *saga.SagaError) bool {
        // 自定义是否需要补偿的逻辑
        return err != nil && err.Type != saga.ErrorTypeBusiness
    },
    func(completedSteps []saga.SagaStep) []saga.SagaStep {
        // 自定义补偿顺序
        return reverseSteps(completedSteps)
    },
)
```

### 4. 超时管理

#### Saga 级别超时

```go
func (d *MySagaDefinition) GetTimeout() time.Duration {
    // 整个 Saga 的超时时间
    return 30 * time.Minute
}
```

#### 步骤级别超时

```go
func (s *MyStep) GetTimeout() time.Duration {
    // 单个步骤的超时时间
    return 10 * time.Second
}
```

#### 处理超时

```go
func (s *MyStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 使用 context 检测超时
    select {
    case <-ctx.Done():
        return nil, ctx.Err() // 返回超时错误
    case result := <-s.doAsyncWork(ctx, data):
        return result, nil
    }
}
```

### 5. 并发控制

```go
config := &coordinator.OrchestratorConfig{
    ConcurrencyConfig: &coordinator.ConcurrencyConfig{
        MaxConcurrentSagas: 100,  // 最大并发 Saga 数
        WorkerPoolSize:     20,   // 工作线程池大小
        AcquireTimeout:     5 * time.Second,
        ShutdownTimeout:    30 * time.Second,
    },
}
```

### 6. 事件和监控

#### 订阅 Saga 事件

```go
// 创建事件过滤器
filter := &saga.EventTypeFilter{
    Types: []saga.SagaEventType{
        saga.EventSagaCompleted,
        saga.EventSagaFailed,
        saga.EventCompensationCompleted,
    },
}

// 创建事件处理器
handler := &MySagaEventHandler{}

// 订阅事件
subscription, err := eventPublisher.Subscribe(filter, handler)
if err != nil {
    log.Printf("订阅失败: %v", err)
}
defer eventPublisher.Unsubscribe(subscription)
```

#### 实现事件处理器

```go
type MySagaEventHandler struct{}

func (h *MySagaEventHandler) HandleEvent(ctx context.Context, event *saga.SagaEvent) error {
    switch event.Type {
    case saga.EventSagaCompleted:
        log.Printf("Saga 完成: %s", event.SagaID)
    case saga.EventSagaFailed:
        log.Printf("Saga 失败: %s, 错误: %s", event.SagaID, event.Error.Message)
    case saga.EventCompensationCompleted:
        log.Printf("补偿完成: %s", event.SagaID)
    }
    return nil
}

func (h *MySagaEventHandler) GetHandlerName() string {
    return "MySagaEventHandler"
}
```

#### 查看运行指标

```go
metrics := coordinator.GetMetrics()

log.Printf("总 Saga 数: %d", metrics.TotalSagas)
log.Printf("活跃: %d", metrics.ActiveSagas)
log.Printf("已完成: %d", metrics.CompletedSagas)
log.Printf("失败: %d", metrics.FailedSagas)
log.Printf("平均执行时间: %s", metrics.AverageSagaDuration)
```

### 7. Saga 事件发布器

Saga 事件发布器（`SagaEventPublisher`）提供可靠的事件发布能力，支持多种发布模式、可靠性保证和性能优化。

#### 创建事件发布器

```go
import (
    "github.com/innovationmech/swit/pkg/messaging"
    sagamessaging "github.com/innovationmech/swit/pkg/saga/messaging"
)

// 1. 创建消息 broker
broker, err := messaging.NewNATSBroker(messaging.BrokerConfig{
    Endpoints: []string{"nats://localhost:4222"},
    Timeout:   5 * time.Second,
})
if err != nil {
    return err
}
defer broker.Close()

// 2. 创建 Saga 事件发布器
publisher, err := sagamessaging.NewSagaEventPublisher(
    broker,
    &sagamessaging.PublisherConfig{
        TopicPrefix:    "saga.events",    // 事件主题前缀
        SerializerType: "json",           // 序列化格式: json 或 protobuf
        RetryAttempts:  3,                // 失败重试次数
        RetryInterval:  time.Second,      // 重试间隔
        Timeout:        5 * time.Second,  // 发布超时
        EnableMetrics:  true,             // 启用指标收集
    },
)
if err != nil {
    return err
}
defer publisher.Close()
```

#### 发布单个事件

```go
// 创建 Saga 事件
event := &saga.SagaEvent{
    ID:         "evt-001",
    SagaID:     "saga-001",
    Type:       saga.EventSagaStarted,
    Timestamp:  time.Now(),
    InstanceID: "inst-001",
    Metadata: map[string]interface{}{
        "order_id":    "ORDER-12345",
        "customer_id": "CUST-67890",
    },
}

// 发布事件
ctx := context.Background()
if err := publisher.PublishSagaEvent(ctx, event); err != nil {
    log.Printf("发布事件失败: %v", err)
    return err
}

log.Printf("事件发布成功: %s", event.ID)
```

#### 批量发布事件

批量发布可以显著提高性能（通常快 5-10 倍）：

```go
// 创建多个事件
events := []*saga.SagaEvent{
    {
        ID:         "evt-batch-001",
        SagaID:     "saga-batch-001",
        Type:       saga.EventSagaStepStarted,
        Timestamp:  time.Now(),
        InstanceID: "inst-batch-001",
    },
    {
        ID:         "evt-batch-002",
        SagaID:     "saga-batch-001",
        Type:       saga.EventSagaStepCompleted,
        Timestamp:  time.Now(),
        InstanceID: "inst-batch-001",
    },
    {
        ID:         "evt-batch-003",
        SagaID:     "saga-batch-001",
        Type:       saga.EventSagaCompleted,
        Timestamp:  time.Now(),
        InstanceID: "inst-batch-001",
    },
}

// 批量发布
if err := publisher.PublishBatch(ctx, events); err != nil {
    log.Printf("批量发布失败: %v", err)
    return err
}

log.Printf("批量发布成功: %d 个事件", len(events))
```

#### 异步批量发布

对于延迟不敏感的场景，异步发布可以提供更高的吞吐量：

```go
// 异步发布事件批次
resultChan := publisher.PublishBatchAsync(ctx, events)

// 可以继续做其他工作...

// 等待发布结果
result := <-resultChan
if result.Error != nil {
    log.Printf("异步发布失败: %v", result.Error)
} else {
    log.Printf("异步发布成功: %d 个事件, 耗时: %s",
        result.PublishedCount, result.Duration)
}
```

#### 事务性消息发送

确保多个相关事件的原子性发布：

```go
// 使用事务发布多个相关事件
err := publisher.WithTransaction(ctx, "tx-order-001", func(txPublisher *sagamessaging.TransactionalEventPublisher) error {
    // 在事务中发布多个事件
    events := []*saga.SagaEvent{
        {
            ID:         "evt-tx-001",
            SagaID:     "saga-tx-001",
            Type:       saga.EventSagaStarted,
            Timestamp:  time.Now(),
            InstanceID: "inst-tx-001",
        },
        {
            ID:         "evt-tx-002",
            SagaID:     "saga-tx-001",
            Type:       saga.EventSagaStepStarted,
            Timestamp:  time.Now(),
            InstanceID: "inst-tx-001",
        },
    }

    for _, event := range events {
        if err := txPublisher.PublishEvent(ctx, event); err != nil {
            return err // 自动回滚
        }
    }

    return nil // 自动提交
})

if err != nil {
    log.Printf("事务性发布失败: %v", err)
} else {
    log.Printf("事务提交成功")
}
```

#### 可靠性保证

配置重试、确认和死信队列以确保消息可靠投递：

```go
publisher, err := sagamessaging.NewSagaEventPublisher(
    broker,
    &sagamessaging.PublisherConfig{
        TopicPrefix:    "saga.events",
        SerializerType: "json",
        EnableConfirm:  true, // 启用发布确认
        RetryAttempts:  5,    // 最多重试 5 次
        RetryInterval:  500 * time.Millisecond,
        Timeout:        10 * time.Second,
        Reliability: &sagamessaging.ReliabilityConfig{
            // 重试配置
            EnableRetry:      true,
            MaxRetryAttempts: 5,
            RetryBackoff:     500 * time.Millisecond,  // 初始退避时间
            MaxRetryBackoff:  5 * time.Second,         // 最大退避时间
            
            // 确认配置
            EnableConfirm:  true,
            ConfirmTimeout: 10 * time.Second,
            
            // 死信队列配置
            EnableDLQ:  true,
            DLQTopic:   "saga.dlq",
        },
    },
)
```

#### 选择序列化格式

发布器支持 JSON 和 Protobuf 两种序列化格式：

```go
// JSON 序列化（默认）- 便于调试和跨语言兼容
&sagamessaging.PublisherConfig{
    SerializerType: "json",
}

// Protobuf 序列化 - 更高性能，更小的消息体积（约 30-50%）
&sagamessaging.PublisherConfig{
    SerializerType: "protobuf",
}
```

**性能对比**:
- JSON: 易于调试，跨语言兼容性好
- Protobuf: 序列化速度快 2-3 倍，消息体积小 30-50%

#### 监控发布指标

实时监控发布器的运行状态：

```go
// 获取发布指标
metrics := publisher.GetMetrics()

log.Printf("发布统计:")
log.Printf("  成功: %d", metrics.PublishedCount)
log.Printf("  失败: %d", metrics.FailedCount)
log.Printf("  批次数: %d", metrics.BatchCount)
log.Printf("  平均延迟: %s", metrics.AverageLatency)
log.Printf("  平均批次大小: %.2f", metrics.AverageBatchSize)
log.Printf("  发布速率: %.2f events/sec", metrics.GetPublishRate())

// 获取可靠性指标
if reliabilityMetrics := publisher.GetReliabilityMetrics(); reliabilityMetrics != nil {
    log.Printf("可靠性指标:")
    log.Printf("  总重试次数: %d", reliabilityMetrics.TotalRetries)
    log.Printf("  成功的重试: %d", reliabilityMetrics.SuccessfulRetries)
    log.Printf("  失败的重试: %d", reliabilityMetrics.FailedRetries)
    log.Printf("  成功率: %.2f%%", reliabilityMetrics.GetSuccessRate()*100)
    log.Printf("  重试成功率: %.2f%%", reliabilityMetrics.GetRetrySuccessRate()*100)
    log.Printf("  DLQ 消息数: %d", reliabilityMetrics.DLQMessagesCount)
}
```

#### 处理发布失败

正确处理各种发布错误：

```go
err := publisher.PublishSagaEvent(ctx, event)
if err != nil {
    switch {
    case errors.Is(err, sagamessaging.ErrPublisherClosed):
        // 发布器已关闭，需要重新创建
        log.Error("发布器已关闭，需要重新初始化")
        
    case errors.Is(err, context.DeadlineExceeded):
        // 超时错误，可能需要重试
        log.Warn("发布超时，将重试", zap.Error(err))
        
    case errors.Is(err, sagamessaging.ErrInvalidEvent):
        // 事件无效，记录日志
        log.Error("无效的事件", zap.Error(err))
        
    case errors.Is(err, sagamessaging.ErrSerializationFailed):
        // 序列化失败
        log.Error("序列化失败", zap.Error(err))
        
    default:
        // 其他错误
        log.Error("发布失败", zap.Error(err))
    }
}
```

#### 最佳实践

1. **复用发布器实例**: 避免为每次发布创建新的发布器，应在应用初始化时创建并复用
2. **使用批量发布**: 当需要发布多个相关事件时，使用批量 API 可显著提高性能
3. **启用可靠性保证**: 在生产环境启用重试、确认和 DLQ 机制
4. **选择合适的序列化格式**: 开发环境使用 JSON 便于调试，生产环境考虑使用 Protobuf 提高性能
5. **监控指标**: 定期检查发布指标，及时发现和解决问题
6. **合理设置超时**: 根据网络环境和消息大小调整超时配置
7. **优雅关闭**: 应用退出时先停止接收新事件，等待正在处理的事件完成后再关闭发布器

#### 完整示例

查看完整的使用示例：
- [examples/saga-publisher/](../examples/saga-publisher/) - 包含各种发布模式的完整示例
- [pkg/saga/messaging/README.md](../pkg/saga/messaging/README.md) - 详细的 API 文档

## 最佳实践

### 1. 步骤设计原则

#### 幂等性

确保步骤和补偿操作都是幂等的：

```go
func (s *MyStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 检查是否已执行
    if s.isAlreadyExecuted(ctx, data) {
        return s.getExistingResult(ctx, data), nil
    }
    
    // 执行操作
    result, err := s.doWork(ctx, data)
    if err != nil {
        return nil, err
    }
    
    // 记录执行状态
    s.markAsExecuted(ctx, data, result)
    
    return result, nil
}
```

#### 原子性

每个步骤应该是一个原子操作：

```go
func (s *MyStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 使用数据库事务确保原子性
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()
    
    // 执行操作
    if err := s.doWork1(ctx, tx, data); err != nil {
        return nil, err
    }
    if err := s.doWork2(ctx, tx, data); err != nil {
        return nil, err
    }
    
    // 提交事务
    if err := tx.Commit(); err != nil {
        return nil, err
    }
    
    return data, nil
}
```

### 2. 错误处理策略

```go
// 定义错误类型
const (
    ErrNetworkFailure    = "NETWORK_FAILURE"      // 可重试
    ErrServiceUnavailable = "SERVICE_UNAVAILABLE" // 可重试
    ErrInvalidRequest    = "INVALID_REQUEST"      // 不可重试
    ErrBusinessLogic     = "BUSINESS_LOGIC"       // 不可重试
)

func (s *MyStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    result, err := s.doWork(ctx, data)
    if err != nil {
        // 分类错误
        if isNetworkError(err) {
            return nil, saga.NewRetryableError(err)
        }
        if isValidationError(err) {
            return nil, saga.NewBusinessError(ErrInvalidRequest, err.Error())
        }
        return nil, err
    }
    return result, nil
}
```

### 3. 状态持久化选择

```go
// 开发环境：内存存储
if env == "development" {
    stateStorage = coordinator.NewInMemoryStateStorage()
}

// 生产环境：Redis（高性能）
if env == "production" && requireHighPerformance {
    stateStorage = NewRedisStateStorage(&RedisConfig{
        Addr:     "redis-cluster:6379",
        Password: os.Getenv("REDIS_PASSWORD"),
        PoolSize: 100,
    })
}

// 生产环境：数据库（强一致性）
if env == "production" && requireStrongConsistency {
    stateStorage = NewDatabaseStateStorage(&DatabaseConfig{
        Driver:   "postgres",
        Host:     "db-master",
        Database: "saga_db",
        MaxConns: 50,
    })
}
```

### 4. 监控和可观测性

```go
func (s *MyStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 添加分布式追踪
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(
        attribute.String("step.id", s.GetID()),
        attribute.String("step.name", s.GetName()),
    )
    
    // 记录指标
    startTime := time.Now()
    defer func() {
        duration := time.Since(startTime)
        metrics.RecordStepDuration(s.GetID(), duration)
    }()
    
    // 执行业务逻辑
    result, err := s.doWork(ctx, data)
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        metrics.RecordStepError(s.GetID(), err)
        return nil, err
    }
    
    span.SetStatus(codes.Ok, "success")
    metrics.RecordStepSuccess(s.GetID())
    return result, nil
}
```

### 5. 测试策略

```go
func TestMySagaSuccess(t *testing.T) {
    // 使用内存实现进行测试
    storage := coordinator.NewInMemoryStateStorage()
    publisher := coordinator.NewInMemoryEventPublisher()
    
    config := &coordinator.OrchestratorConfig{
        StateStorage:   storage,
        EventPublisher: publisher,
    }
    
    coordinator, err := coordinator.NewOrchestratorCoordinator(config)
    require.NoError(t, err)
    defer coordinator.Close()
    
    // 创建测试 Saga
    definition := NewTestSagaDefinition()
    data := &TestData{Value: "test"}
    
    // 启动 Saga
    ctx := context.Background()
    instance, err := coordinator.StartSaga(ctx, definition, data)
    require.NoError(t, err)
    
    // 等待完成
    waitForCompletion(t, coordinator, instance.GetID())
    
    // 验证结果
    finalInstance, err := coordinator.GetSagaInstance(instance.GetID())
    require.NoError(t, err)
    assert.Equal(t, saga.StateCompleted, finalInstance.GetState())
}
```

## 故障排查

### 常见问题

#### 1. Saga 一直处于运行状态

**原因**:
- 步骤执行时间过长
- 步骤阻塞等待外部资源
- 超时配置不合理

**解决方案**:
```go
// 检查超时配置
func (s *MyStep) GetTimeout() time.Duration {
    return 10 * time.Second  // 确保设置了合理的超时
}

// 在步骤中正确处理 context
func (s *MyStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()  // 正确响应超时
    case result := <-s.doWork(ctx, data):
        return result, nil
    }
}
```

#### 2. 补偿操作失败

**原因**:
- 补偿逻辑不是幂等的
- 补偿操作依赖已删除的资源
- 补偿超时

**解决方案**:
```go
func (s *MyStep) Compensate(ctx context.Context, data interface{}) error {
    // 1. 检查是否需要补偿
    if s.needsCompensation(ctx, data) == false {
        return nil  // 已经补偿过，直接返回
    }
    
    // 2. 执行补偿（幂等）
    err := s.undoWork(ctx, data)
    if err != nil {
        // 3. 记录错误但不返回，允许其他步骤继续补偿
        log.Printf("补偿失败: %v", err)
    }
    
    return nil
}
```

#### 3. 并发限制达到上限

**原因**:
- `MaxConcurrentSagas` 设置过小
- 步骤执行时间过长
- 系统负载过高

**解决方案**:
```go
// 调整并发配置
config.ConcurrencyConfig = &coordinator.ConcurrencyConfig{
    MaxConcurrentSagas: 200,  // 增加并发数
    WorkerPoolSize:     50,   // 增加工作线程
    AcquireTimeout:     10 * time.Second,
    ShutdownTimeout:    60 * time.Second,
}

// 优化步骤执行
func (s *MyStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 使用异步处理减少阻塞时间
    return s.doWorkAsync(ctx, data)
}
```

## 进阶主题

### 1. 自定义状态存储

```go
type MyStateStorage struct {
    // 实现细节
}

func (s *MyStateStorage) SaveSaga(ctx context.Context, saga saga.SagaInstance) error {
    // 实现保存逻辑
}

// 实现其他 StateStorage 接口方法...
```

### 2. 自定义事件发布器

```go
type MyEventPublisher struct {
    // 实现细节
}

func (p *MyEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
    // 实现发布逻辑
}

// 实现其他 EventPublisher 接口方法...
```

### 3. 集成 OpenTelemetry

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

type OTelTracingManager struct {
    tracer trace.Tracer
}

func NewOTelTracingManager() *OTelTracingManager {
    return &OTelTracingManager{
        tracer: otel.Tracer("saga-coordinator"),
    }
}

func (m *OTelTracingManager) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
    ctx, span := m.tracer.Start(ctx, name)
    return ctx, &otelSpan{span: span}
}
```

## 参考资料

- [完整示例代码](../examples/saga-orchestrator/)
- [API 文档](./generated/saga.md)
- [架构设计](../specs/saga-distributed-transactions/README.md)
- [使用场景](../specs/saga-distributed-transactions/use-cases.md)


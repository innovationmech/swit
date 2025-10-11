# Saga Orchestrator 使用示例

本示例演示如何使用 Swit 框架的 Saga Orchestrator 来管理分布式事务。

## 概述

Saga Orchestrator 提供集中式的分布式事务协调能力，支持：
- 顺序执行多个步骤
- 自动重试失败的步骤
- 失败时自动补偿已完成的步骤
- 分布式追踪和监控
- 并发控制和超时管理

## 快速开始

### 1. 创建 Saga 定义

```go
package main

import (
    "context"
    "time"
    "github.com/innovationmech/swit/pkg/saga"
    "github.com/innovationmech/swit/pkg/saga/coordinator"
)

// 定义订单处理 Saga
type OrderProcessingSaga struct {
    orderService     OrderService
    inventoryService InventoryService
    paymentService   PaymentService
}

func (ops *OrderProcessingSaga) GetDefinition() saga.SagaDefinition {
    return &OrderSagaDefinition{
        id:          "order-processing-saga",
        name:        "订单处理 Saga",
        description: "处理电商订单的完整流程",
        steps: []saga.SagaStep{
            NewCreateOrderStep(ops.orderService),
            NewReserveInventoryStep(ops.inventoryService),
            NewProcessPaymentStep(ops.paymentService),
            NewConfirmOrderStep(ops.orderService),
        },
        timeout:     30 * time.Minute,
        retryPolicy: saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
        strategy:    saga.NewSequentialCompensationStrategy(30 * time.Second),
    }
}
```

### 2. 实现 Saga 步骤

```go
// 创建订单步骤
type CreateOrderStep struct {
    service OrderService
}

func NewCreateOrderStep(service OrderService) *CreateOrderStep {
    return &CreateOrderStep{service: service}
}

func (s *CreateOrderStep) GetID() string {
    return "create-order"
}

func (s *CreateOrderStep) GetName() string {
    return "创建订单"
}

func (s *CreateOrderStep) GetDescription() string {
    return "在订单服务中创建新订单"
}

func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 调用订单服务创建订单
    order, err := s.service.CreateOrder(ctx, &CreateOrderRequest{
        CustomerID: orderData.CustomerID,
        Items:      orderData.Items,
        TotalAmount: orderData.TotalAmount,
        ShippingAddress: orderData.ShippingAddress,
    })
    if err != nil {
        return nil, fmt.Errorf("创建订单失败: %w", err)
    }
    
    // 返回结果数据，传递给下一步
    orderData.OrderID = order.ID
    orderData.Order = order
    return orderData, nil
}

func (s *CreateOrderStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    // 取消订单作为补偿
    err := s.service.CancelOrder(ctx, &CancelOrderRequest{
        OrderID: orderData.OrderID,
        Reason:  "saga_compensation",
    })
    if err != nil {
        log.Printf("警告: 补偿操作失败 - 取消订单: %v", err)
        // 注意：补偿失败不应返回错误，以免阻止其他步骤的补偿
    }
    
    return nil
}

func (s *CreateOrderStep) GetTimeout() time.Duration {
    return 5 * time.Second
}

func (s *CreateOrderStep) GetRetryPolicy() saga.RetryPolicy {
    return saga.NewFixedDelayRetryPolicy(3, time.Second)
}

func (s *CreateOrderStep) IsRetryable(err error) bool {
    // 判断错误是否可重试
    return !IsBusinessError(err)
}

func (s *CreateOrderStep) GetMetadata() map[string]interface{} {
    return map[string]interface{}{
        "service": "order-service",
        "version": "v1",
    }
}
```

### 3. 初始化 Coordinator

```go
func main() {
    // 创建状态存储（可选择内存、Redis、数据库等实现）
    stateStorage := NewRedisStateStorage(&RedisConfig{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })
    
    // 创建事件发布器
    eventPublisher := NewKafkaEventPublisher(&KafkaConfig{
        Brokers: []string{"localhost:9092"},
        Topic:   "saga-events",
    })
    
    // 创建指标收集器
    metricsCollector := NewPrometheusMetricsCollector()
    
    // 创建追踪管理器
    tracingManager := NewOTelTracingManager()
    
    // 配置 Coordinator
    config := &coordinator.OrchestratorConfig{
        StateStorage:     stateStorage,
        EventPublisher:   eventPublisher,
        RetryPolicy:      saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
        MetricsCollector: metricsCollector,
        TracingManager:   tracingManager,
        ConcurrencyConfig: &coordinator.ConcurrencyConfig{
            MaxConcurrentSagas: 100,
            WorkerPoolSize:     20,
            AcquireTimeout:     5 * time.Second,
            ShutdownTimeout:    30 * time.Second,
        },
    }
    
    // 创建 Coordinator
    sagaCoordinator, err := coordinator.NewOrchestratorCoordinator(config)
    if err != nil {
        log.Fatalf("创建 Saga Coordinator 失败: %v", err)
    }
    defer sagaCoordinator.Close()
    
    // 使用 Coordinator
    startSaga(sagaCoordinator)
}
```

### 4. 启动 Saga

```go
func startSaga(coordinator saga.SagaCoordinator) {
    // 创建 Saga 定义
    orderSaga := &OrderProcessingSaga{
        orderService:     NewOrderService(),
        inventoryService: NewInventoryService(),
        paymentService:   NewPaymentService(),
    }
    definition := orderSaga.GetDefinition()
    
    // 准备初始数据
    initialData := &OrderData{
        CustomerID: "CUST-12345",
        Items: []OrderItem{
            {ProductID: "PROD-001", Quantity: 2, Price: 99.99},
            {ProductID: "PROD-002", Quantity: 1, Price: 149.99},
        },
        TotalAmount: 349.97,
        ShippingAddress: Address{
            Street:  "123 Main St",
            City:    "San Francisco",
            State:   "CA",
            ZipCode: "94102",
            Country: "US",
        },
    }
    
    // 启动 Saga
    ctx := context.Background()
    instance, err := coordinator.StartSaga(ctx, definition, initialData)
    if err != nil {
        log.Printf("启动 Saga 失败: %v", err)
        return
    }
    
    log.Printf("Saga 已启动: ID=%s, DefinitionID=%s", instance.GetID(), instance.GetDefinitionID())
    
    // 等待 Saga 完成（可选）
    waitForSagaCompletion(coordinator, instance.GetID())
}
```

### 5. 监控 Saga 状态

```go
func waitForSagaCompletion(coordinator saga.SagaCoordinator, sagaID string) {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    timeout := time.After(5 * time.Minute)
    
    for {
        select {
        case <-timeout:
            log.Printf("Saga 执行超时: %s", sagaID)
            return
        case <-ticker.C:
            instance, err := coordinator.GetSagaInstance(sagaID)
            if err != nil {
                log.Printf("获取 Saga 实例失败: %v", err)
                return
            }
            
            log.Printf("Saga 状态: ID=%s, State=%s, CompletedSteps=%d/%d",
                instance.GetID(),
                instance.GetState().String(),
                instance.GetCompletedSteps(),
                instance.GetTotalSteps())
            
            if instance.IsTerminal() {
                if instance.GetState() == saga.StateCompleted {
                    log.Printf("✓ Saga 执行成功: %s", sagaID)
                    result := instance.GetResult()
                    log.Printf("  结果: %+v", result)
                } else {
                    log.Printf("✗ Saga 执行失败: %s, State=%s",
                        sagaID,
                        instance.GetState().String())
                    if err := instance.GetError(); err != nil {
                        log.Printf("  错误: %s", err.Message)
                    }
                }
                return
            }
        }
    }
}
```

## 完整示例

完整的示例代码请参见 `main.go` 文件。

## 高级特性

### 1. 自定义重试策略

```go
// 指数退避策略
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

// 不重试
retryPolicy := saga.NewNoRetryPolicy()
```

### 2. 自定义补偿策略

```go
// 顺序补偿（默认，逆序执行）
strategy := saga.NewSequentialCompensationStrategy(30 * time.Second)

// 并行补偿
strategy := saga.NewParallelCompensationStrategy(30 * time.Second)

// 尽力而为补偿（即使部分失败也继续）
strategy := saga.NewBestEffortCompensationStrategy(30 * time.Second)

// 自定义补偿策略
strategy := saga.NewCustomCompensationStrategy(
    30 * time.Second,
    func(err *saga.SagaError) bool {
        // 自定义判断是否需要补偿
        return err != nil && err.Type != saga.ErrorTypeBusiness
    },
    func(completedSteps []saga.SagaStep) []saga.SagaStep {
        // 自定义补偿顺序
        return reverseSteps(completedSteps)
    },
)
```

### 3. 事件订阅

```go
// 订阅 Saga 事件
filter := &saga.EventTypeFilter{
    Types: []saga.SagaEventType{
        saga.EventSagaCompleted,
        saga.EventSagaFailed,
    },
}

handler := &MySagaEventHandler{}

subscription, err := eventPublisher.Subscribe(filter, handler)
if err != nil {
    log.Printf("订阅事件失败: %v", err)
}
defer eventPublisher.Unsubscribe(subscription)

// 自定义事件处理器
type MySagaEventHandler struct{}

func (h *MySagaEventHandler) HandleEvent(ctx context.Context, event *saga.SagaEvent) error {
    log.Printf("接收到事件: Type=%s, SagaID=%s", event.Type, event.SagaID)
    
    switch event.Type {
    case saga.EventSagaCompleted:
        // 处理 Saga 完成事件
        return h.handleSagaCompleted(ctx, event)
    case saga.EventSagaFailed:
        // 处理 Saga 失败事件
        return h.handleSagaFailed(ctx, event)
    }
    
    return nil
}

func (h *MySagaEventHandler) GetHandlerName() string {
    return "MySagaEventHandler"
}
```

### 4. 健康检查和指标

```go
// 执行健康检查
ctx := context.Background()
if err := coordinator.HealthCheck(ctx); err != nil {
    log.Printf("健康检查失败: %v", err)
}

// 获取运行指标
metrics := coordinator.GetMetrics()
log.Printf("Saga 指标:")
log.Printf("  总数: %d", metrics.TotalSagas)
log.Printf("  活跃: %d", metrics.ActiveSagas)
log.Printf("  已完成: %d", metrics.CompletedSagas)
log.Printf("  失败: %d", metrics.FailedSagas)
log.Printf("  平均执行时间: %s", metrics.AverageSagaDuration)
```

### 5. 查询活跃 Saga

```go
// 查询所有活跃的 Saga
filter := &saga.SagaFilter{
    States: []saga.SagaState{
        saga.StateRunning,
        saga.StateCompensating,
    },
    Limit:  100,
    Offset: 0,
    SortBy: "created_at",
    SortOrder: "desc",
}

activeSagas, err := coordinator.GetActiveSagas(filter)
if err != nil {
    log.Printf("查询活跃 Saga 失败: %v", err)
}

for _, saga := range activeSagas {
    log.Printf("活跃 Saga: ID=%s, State=%s, Progress=%d/%d",
        saga.GetID(),
        saga.GetState().String(),
        saga.GetCompletedSteps(),
        saga.GetTotalSteps())
}
```

## 最佳实践

### 1. 步骤设计原则

- **幂等性**: 所有步骤和补偿操作都应该是幂等的
- **原子性**: 每个步骤应该是一个原子操作
- **独立性**: 步骤之间应该尽可能独立
- **可补偿性**: 每个步骤都应该有对应的补偿操作

### 2. 错误处理

```go
func (s *MyStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 区分可重试错误和不可重试错误
    result, err := s.doSomething(ctx, data)
    if err != nil {
        if IsTemporaryError(err) {
            // 临时错误，可以重试
            return nil, saga.NewRetryableError(err)
        } else {
            // 业务错误，不应重试
            return nil, saga.NewBusinessError("INVALID_REQUEST", err.Error())
        }
    }
    return result, nil
}

func (s *MyStep) IsRetryable(err error) bool {
    // 根据错误类型判断是否可重试
    if sagaErr, ok := err.(*saga.SagaError); ok {
        return sagaErr.Retryable
    }
    // 默认不重试未知错误
    return false
}
```

### 3. 超时配置

```go
// Saga 级别超时
definition.timeout = 30 * time.Minute

// 步骤级别超时
step.timeout = 5 * time.Second

// 建议：步骤超时 < Saga 超时
// 建议：为长时间运行的操作设置合理的超时
```

### 4. 监控和追踪

```go
// 在步骤中添加追踪信息
func (s *MyStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(
        attribute.String("step.id", s.GetID()),
        attribute.String("step.name", s.GetName()),
    )
    
    // 执行业务逻辑
    result, err := s.doWork(ctx, data)
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    
    span.SetStatus(codes.Ok, "success")
    return result, nil
}
```

### 5. 状态持久化

```go
// 选择合适的状态存储
// - 开发/测试: InMemoryStateStorage
// - 生产环境: RedisStateStorage 或 DatabaseStateStorage

// Redis 实现（推荐用于高性能场景）
stateStorage := NewRedisStateStorage(&RedisConfig{
    Addr:         "localhost:6379",
    Password:     "",
    DB:           0,
    PoolSize:     100,
    MaxRetries:   3,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
})

// 数据库实现（推荐用于强一致性要求）
stateStorage := NewDatabaseStateStorage(&DatabaseConfig{
    Driver:   "postgres",
    Host:     "localhost",
    Port:     5432,
    Database: "saga_db",
    Username: "saga_user",
    Password: "saga_pass",
    MaxConns: 50,
})
```

## 故障排查

### 常见问题

#### 1. Saga 一直处于运行状态

- 检查步骤是否有死循环
- 检查超时配置是否合理
- 检查是否有步骤阻塞等待外部资源

#### 2. 补偿操作失败

- 确保补偿操作是幂等的
- 检查补偿逻辑是否正确
- 考虑使用尽力而为补偿策略

#### 3. 重试次数过多

- 检查重试策略配置
- 区分可重试错误和不可重试错误
- 添加适当的退避延迟

#### 4. 并发限制达到上限

- 调整 `MaxConcurrentSagas` 配置
- 增加 `WorkerPoolSize`
- 优化步骤执行时间

## 性能优化

### 1. 并发配置优化

```go
config.ConcurrencyConfig = &coordinator.ConcurrencyConfig{
    MaxConcurrentSagas: 200,          // 根据系统资源调整
    WorkerPoolSize:     50,           // 工作线程数
    AcquireTimeout:     10 * time.Second,
    ShutdownTimeout:    60 * time.Second,
}
```

### 2. 状态存储优化

- 使用 Redis 集群提高性能
- 启用连接池
- 配置合理的超时时间
- 定期清理过期数据

### 3. 事件发布优化

- 使用异步事件发布
- 批量发布事件
- 配置合理的缓冲区大小

## 更多资源

- [API 文档](../../docs/generated/saga.md)
- [架构设计](../../specs/saga-distributed-transactions/README.md)
- [使用场景](../../specs/saga-distributed-transactions/use-cases.md)
- [实现计划](../../specs/saga-distributed-transactions/implementation-plan.md)


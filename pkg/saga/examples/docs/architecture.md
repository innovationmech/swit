# Saga 示例应用架构文档

## 概述

本文档描述了 Swit 框架 Saga 示例应用的架构设计、设计模式和最佳实践。这些示例展示了如何在实际业务场景中应用 Saga 模式处理复杂的分布式事务。

## 设计原则

### 1. 单一职责原则（SRP）

每个步骤只负责一个明确的业务操作：

```go
// ✅ 好的设计：单一职责
type CreateOrderStep struct {
    service OrderServiceClient
}

func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 只负责创建订单
    return s.service.CreateOrder(ctx, orderData)
}

// ❌ 不好的设计：职责混杂
type CreateOrderAndReserveStep struct {
    orderService     OrderServiceClient
    inventoryService InventoryServiceClient
}

func (s *CreateOrderAndReserveStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 同时处理订单创建和库存预留 - 违反单一职责
    order := s.orderService.CreateOrder(ctx, orderData)
    s.inventoryService.ReserveInventory(ctx, order.Items)
    // ...
}
```

### 2. 幂等性原则

所有步骤和补偿操作都应该是幂等的：

```go
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // ✅ 检查订单是否已存在（实现幂等性）
    if orderData.OrderID != "" {
        existing, err := s.service.GetOrder(ctx, orderData.OrderID)
        if err == nil && existing != nil {
            return &OrderStepResult{OrderID: existing.ID, ...}, nil
        }
    }
    
    // 创建新订单
    return s.service.CreateOrder(ctx, orderData)
}
```

### 3. 向后兼容性

补偿操作应该能够处理步骤的任何成功状态：

```go
func (s *ProcessPaymentStep) Compensate(ctx context.Context, data interface{}) error {
    paymentResult := data.(*PaymentStepResult)
    
    // ✅ 检查支付状态，避免重复退款
    status, _ := s.service.GetPaymentStatus(ctx, paymentResult.TransactionID)
    if status == "refunded" {
        return nil // 已退款，无需重复操作
    }
    
    return s.service.RefundPayment(ctx, paymentResult.TransactionID)
}
```

### 4. 失败快速原则

尽早发现和报告错误：

```go
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // ✅ 尽早验证，快速失败
    if err := s.validateOrderData(orderData); err != nil {
        return nil, fmt.Errorf("订单数据验证失败: %w", err)
    }
    
    // 继续执行业务逻辑
    return s.service.CreateOrder(ctx, orderData)
}
```

## 架构模式

### 1. 编排式 Saga（Orchestration）

所有示例都使用编排式 Saga，由中央协调器管理整个流程。

#### 优势

- 集中管理：流程清晰，易于理解和维护
- 状态可见：可以查询 Saga 的当前状态
- 错误处理：统一的错误处理和补偿机制
- 易于测试：可以独立测试每个步骤

#### 架构图

```
┌─────────────────────────────────────────┐
│         Saga Coordinator                │
│  ┌───────────────────────────────────┐  │
│  │  State Storage                    │  │
│  │  - Saga Instances                 │  │
│  │  - Step Results                   │  │
│  │  - Execution History              │  │
│  └───────────────────────────────────┘  │
│                                          │
│  ┌───────────────────────────────────┐  │
│  │  Event Publisher                  │  │
│  │  - Lifecycle Events               │  │
│  │  - Step Events                    │  │
│  │  - Error Events                   │  │
│  └───────────────────────────────────┘  │
│                                          │
│  ┌───────────────────────────────────┐  │
│  │  Execution Engine                 │  │
│  │  - Step Executor                  │  │
│  │  - Compensation Handler           │  │
│  │  - Retry Manager                  │  │
│  └───────────────────────────────────┘  │
└────────────┬─────────────────────────────┘
             │
    ┌────────┴────────┬─────────┬─────────────┐
    │                 │         │             │
┌───▼────┐    ┌──────▼───┐ ┌───▼────┐  ┌────▼─────┐
│ Step 1 │    │ Step 2   │ │ Step 3 │  │ Step N   │
│        │───>│          │─>│        │─>│          │
└────────┘    └──────────┘ └────────┘  └──────────┘
    │             │            │             │
    ▼             ▼            ▼             ▼
┌───────────────────────────────────────────────┐
│          External Services                     │
│  ┌────────┐  ┌────────┐  ┌────────┐          │
│  │Service1│  │Service2│  │Service3│  ...     │
│  └────────┘  └────────┘  └────────┘          │
└───────────────────────────────────────────────┘
```

### 2. 数据传递模式

#### 链式传递

每个步骤接收上一步的输出作为输入：

```go
// 步骤 1: 创建订单
type CreateOrderStep struct{}
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData) // 输入
    // ...
    return &OrderStepResult{...}, nil // 输出
}

// 步骤 2: 预留库存
type ReserveInventoryStep struct{}
func (s *ReserveInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderResult := data.(*OrderStepResult) // 接收上一步输出
    // ...
    return &InventoryStepResult{...}, nil // 输出给下一步
}
```

#### 元数据传递

通过元数据传递跨步骤的共享信息：

```go
func (s *ReserveInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderResult := data.(*OrderStepResult)
    
    // 在结果中添加元数据
    result := &InventoryStepResult{
        ReservationID: "RES-123",
        // ... 其他字段
        Metadata: map[string]interface{}{
            "order_id":       orderResult.OrderID,       // 传递给后续步骤
            "amount":         orderResult.TotalAmount,   // 支付步骤需要
            "payment_method": orderResult.PaymentMethod, // 支付步骤需要
        },
    }
    
    return result, nil
}
```

### 3. 错误处理模式

#### 错误分类

```go
// 业务错误（不可重试）
type BusinessError struct {
    Code      string
    Message   string
    Retryable bool
    Details   map[string]interface{}
}

// 临时错误（可重试）
type TemporaryError struct {
    Cause   error
    Message string
}

// 在步骤中使用
func (s *Step) IsRetryable(err error) bool {
    if busErr, ok := err.(*BusinessError); ok {
        return busErr.Retryable
    }
    if _, ok := err.(*TemporaryError); ok {
        return true
    }
    return false
}
```

#### 重试策略

```go
// 指数退避
retryPolicy := saga.NewExponentialBackoffRetryPolicy(
    3,                 // 最大重试次数
    1*time.Second,     // 初始延迟
    10*time.Second,    // 最大延迟
)

// 固定间隔
retryPolicy := saga.NewFixedIntervalRetryPolicy(
    5,                 // 最大重试次数
    2*time.Second,     // 重试间隔
)
```

### 4. 补偿策略

#### 顺序补偿

按照步骤执行的相反顺序进行补偿：

```
执行顺序: Step1 → Step2 → Step3 → Step4
补偿顺序: Step4 ← Step3 ← Step2 ← Step1
```

```go
compensationStrategy := saga.NewSequentialCompensationStrategy(
    3 * time.Minute, // 补偿超时
)
```

#### 并行补偿

对于独立的步骤，可以并行执行补偿：

```go
compensationStrategy := saga.NewParallelCompensationStrategy(
    3 * time.Minute, // 补偿超时
    4,               // 并发数
)
```

## 代码组织结构

### 文件组织

每个示例文件遵循统一的结构：

```go
// 1. Copyright 和 Package 声明
// Copyright © 2025 ...
package examples

// 2. Import 声明
import (
    "context"
    "errors"
    // ...
)

// 3. 数据结构定义
type OrderData struct {
    // 输入数据字段
}

type OrderStepResult struct {
    // 步骤结果字段
}

// 4. 服务接口定义
type OrderServiceClient interface {
    CreateOrder(ctx context.Context, data *OrderData) (*OrderStepResult, error)
    CancelOrder(ctx context.Context, orderID string, reason string) error
}

// 5. 步骤实现
type CreateOrderStep struct {
    service OrderServiceClient
}

func (s *CreateOrderStep) GetID() string { return "create-order" }
func (s *CreateOrderStep) GetName() string { return "创建订单" }
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) { /* ... */ }
func (s *CreateOrderStep) Compensate(ctx context.Context, data interface{}) error { /* ... */ }
// ... 其他接口方法

// 6. Saga 定义
type OrderProcessingSagaDefinition struct {
    id          string
    name        string
    steps       []saga.SagaStep
    // ... 其他字段
}

// 实现 SagaDefinition 接口
func (d *OrderProcessingSagaDefinition) GetID() string { /* ... */ }
// ... 其他接口方法

// 7. 工厂函数
func NewOrderProcessingSaga(/* services */) *OrderProcessingSagaDefinition {
    // 创建并返回 Saga 定义
}
```

### 命名约定

#### 步骤命名

- 使用动词+名词：`CreateOrderStep`、`ProcessPaymentStep`
- 清晰表达业务意图
- 避免缩写（除非是通用术语）

#### 结果命名

- 步骤名+Result：`OrderStepResult`、`PaymentStepResult`
- 或具体的业务含义：`ValidationResult`、`AllocationResult`

#### 函数命名

- Execute：执行步骤逻辑
- Compensate：补偿操作
- Get* 系列：获取步骤属性（ID、Name、Timeout等）

## 测试策略

### 1. 单元测试

测试每个步骤的独立功能：

```go
func TestCreateOrderStep_Execute_Success(t *testing.T) {
    // 创建 Mock 服务
    mockService := &MockOrderService{
        CreateOrderFunc: func(ctx context.Context, data *OrderData) (*OrderStepResult, error) {
            return &OrderStepResult{
                OrderID: "ORD-123",
                Status:  "pending",
            }, nil
        },
    }
    
    // 创建步骤
    step := &CreateOrderStep{service: mockService}
    
    // 准备输入数据
    orderData := &OrderData{
        CustomerID:  "CUST-123",
        TotalAmount: 100.00,
        Items: []OrderItem{
            {ProductID: "PROD-001", Quantity: 1},
        },
    }
    
    // 执行步骤
    result, err := step.Execute(context.Background(), orderData)
    
    // 验证结果
    assert.NoError(t, err)
    assert.NotNil(t, result)
    
    orderResult := result.(*OrderStepResult)
    assert.Equal(t, "ORD-123", orderResult.OrderID)
    assert.Equal(t, "pending", orderResult.Status)
}
```

### 2. 集成测试

测试完整的 Saga 流程：

```go
func TestOrderSaga_Success(t *testing.T) {
    // 创建 Coordinator
    coordinator := setupTestCoordinator(t)
    defer coordinator.Close()
    
    // 创建 Saga 定义
    sagaDef := NewOrderProcessingSaga(
        mockOrderService,
        mockInventoryService,
        mockPaymentService,
    )
    
    // 准备数据
    orderData := &OrderData{
        CustomerID:  "CUST-123",
        TotalAmount: 100.00,
        // ...
    }
    
    // 启动 Saga
    instance, err := coordinator.StartSaga(context.Background(), sagaDef, orderData)
    assert.NoError(t, err)
    
    // 等待完成
    waitForSagaCompletion(t, coordinator, instance.GetID(), 30*time.Second)
    
    // 验证最终状态
    finalInstance, _ := coordinator.GetSagaInstance(instance.GetID())
    assert.Equal(t, saga.StatusCompleted, finalInstance.GetStatus())
}
```

### 3. 补偿测试

测试失败和补偿场景：

```go
func TestOrderSaga_PaymentFailure_Compensation(t *testing.T) {
    // 配置支付服务在第3次调用时失败
    mockPaymentService := &MockPaymentService{
        ProcessPaymentFunc: func(ctx context.Context, orderID string, amount float64) (*PaymentStepResult, error) {
            return nil, errors.New("payment failed")
        },
    }
    
    // 创建 Saga 定义
    sagaDef := NewOrderProcessingSaga(
        mockOrderService,
        mockInventoryService,
        mockPaymentService,
    )
    
    // 启动 Saga
    instance, _ := coordinator.StartSaga(context.Background(), sagaDef, orderData)
    
    // 等待完成
    waitForSagaCompletion(t, coordinator, instance.GetID(), 30*time.Second)
    
    // 验证最终状态
    finalInstance, _ := coordinator.GetSagaInstance(instance.GetID())
    assert.Equal(t, saga.StatusCompensated, finalInstance.GetStatus())
    
    // 验证补偿操作被调用
    assert.True(t, mockInventoryService.ReleaseInventoryCalled)
    assert.True(t, mockOrderService.CancelOrderCalled)
}
```

## 性能优化

### 1. 异步执行

对于非关键步骤，可以考虑异步执行：

```go
// 同步执行关键步骤
steps := []saga.SagaStep{
    &CreateOrderStep{},        // 同步
    &ReserveInventoryStep{},   // 同步
    &ProcessPaymentStep{},     // 同步
    &ConfirmOrderStep{},       // 同步
    &SendNotificationStep{     // 可以异步
        async: true,
    },
}
```

### 2. 批量操作

当处理多个相似项目时，使用批量操作：

```go
func (s *LockInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    allocationResult := data.(*AllocationResult)
    
    // ✅ 批量锁定，而不是逐个锁定
    lockResults, err := s.service.BatchLockInventory(ctx, &BatchLockRequest{
        Items: allocationResult.Allocations,
    })
    
    return lockResults, err
}
```

### 3. 缓存

缓存频繁访问的数据：

```go
func (s *QueryWarehousesStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 检查缓存
    if cached := s.cache.Get("active_warehouses"); cached != nil {
        return cached.([]WarehouseInfo), nil
    }
    
    // 查询数据库
    warehouses, err := s.service.QueryWarehouses(ctx, query)
    
    // 缓存结果
    s.cache.Set("active_warehouses", warehouses, 5*time.Minute)
    
    return warehouses, err
}
```

### 4. 连接池

使用连接池管理外部资源：

```go
type OrderService struct {
    db   *sql.DB
    pool *redis.Pool
}

func NewOrderService(dsn string, redisAddr string) *OrderService {
    db, _ := sql.Open("postgres", dsn)
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    
    pool := &redis.Pool{
        MaxIdle:     10,
        IdleTimeout: 240 * time.Second,
        Dial: func() (redis.Conn, error) {
            return redis.Dial("tcp", redisAddr)
        },
    }
    
    return &OrderService{db: db, pool: pool}
}
```

## 监控和可观测性

### 1. 指标收集

```go
type SagaMetrics struct {
    // 计数器
    SagaStarted    prometheus.Counter
    SagaCompleted  prometheus.Counter
    SagaFailed     prometheus.Counter
    SagaCompensated prometheus.Counter
    
    // 直方图
    SagaDuration   prometheus.Histogram
    StepDuration   prometheus.Histogram
    
    // 仪表
    ActiveSagas    prometheus.Gauge
}

// 在 Coordinator 中使用
func (c *Coordinator) StartSaga(ctx context.Context, def SagaDefinition, data interface{}) (*SagaInstance, error) {
    c.metrics.SagaStarted.Inc()
    c.metrics.ActiveSagas.Inc()
    
    start := time.Now()
    defer func() {
        c.metrics.SagaDuration.Observe(time.Since(start).Seconds())
    }()
    
    // ... 执行 Saga
}
```

### 2. 分布式追踪

```go
import "go.opentelemetry.io/otel"

func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    tracer := otel.Tracer("saga")
    ctx, span := tracer.Start(ctx, "CreateOrderStep.Execute")
    defer span.End()
    
    // 添加属性
    span.SetAttributes(
        attribute.String("order.customer_id", orderData.CustomerID),
        attribute.Float64("order.amount", orderData.TotalAmount),
    )
    
    // 执行业务逻辑
    result, err := s.service.CreateOrder(ctx, orderData)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, err
    }
    
    return result, nil
}
```

### 3. 日志记录

```go
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    log := logger.FromContext(ctx)
    
    log.Info("开始执行创建订单步骤",
        "customer_id", orderData.CustomerID,
        "amount", orderData.TotalAmount,
    )
    
    result, err := s.service.CreateOrder(ctx, orderData)
    
    if err != nil {
        log.Error("创建订单失败",
            "error", err,
            "customer_id", orderData.CustomerID,
        )
        return nil, err
    }
    
    log.Info("创建订单成功",
        "order_id", result.OrderID,
        "customer_id", orderData.CustomerID,
    )
    
    return result, nil
}
```

## 安全性考虑

### 1. 数据加密

敏感数据应该加密存储和传输：

```go
func (s *CreateUserStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    regData := data.(*UserRegistrationData)
    
    // 加密敏感字段
    encryptedPhone, _ := encrypt(regData.Phone)
    encryptedEmail, _ := encrypt(regData.Email)
    
    // 创建用户
    user := &User{
        Username: regData.Username,
        Password: hashPassword(regData.Password), // 密码哈希
        Phone:    encryptedPhone,                 // 加密电话
        Email:    encryptedEmail,                 // 加密邮箱
    }
    
    return s.service.CreateUser(ctx, user)
}
```

### 2. 审计日志

记录所有关键操作：

```go
func (s *ExecuteTransferStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 执行转账
    result, err := s.service.ExecuteTransfer(ctx, transferID, freezeID)
    
    // 记录审计日志
    auditLog := &AuditLog{
        Timestamp:   time.Now(),
        Operation:   "transfer",
        UserID:      getUserID(ctx),
        FromAccount: result.FromAccount,
        ToAccount:   result.ToAccount,
        Amount:      result.Amount,
        Status:      "success",
        IPAddress:   getIPAddress(ctx),
    }
    s.auditService.Log(ctx, auditLog)
    
    return result, err
}
```

### 3. 权限验证

在步骤执行前验证权限：

```go
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 验证用户权限
    if !s.authService.CanCreateOrder(ctx, orderData.CustomerID) {
        return nil, &PermissionDeniedError{
            UserID:    getUserID(ctx),
            Operation: "create_order",
        }
    }
    
    // 执行创建订单
    return s.service.CreateOrder(ctx, orderData)
}
```

## 部署考虑

### 1. 配置管理

使用配置文件管理不同环境的配置：

```yaml
# config/production.yaml
saga:
  coordinator:
    state_storage:
      type: redis
      address: redis://prod-redis:6379
      db: 0
    
    event_publisher:
      type: nats
      url: nats://prod-nats:4222
    
    retry:
      max_attempts: 3
      initial_interval: 2s
      max_interval: 30s
```

### 2. 健康检查

实现健康检查接口：

```go
func (c *Coordinator) HealthCheck(ctx context.Context) error {
    // 检查状态存储
    if err := c.stateStorage.Ping(ctx); err != nil {
        return fmt.Errorf("state storage unhealthy: %w", err)
    }
    
    // 检查事件发布器
    if err := c.eventPublisher.Ping(ctx); err != nil {
        return fmt.Errorf("event publisher unhealthy: %w", err)
    }
    
    return nil
}
```

### 3. 优雅关闭

确保 Saga 能够优雅关闭：

```go
func (c *Coordinator) Shutdown(ctx context.Context) error {
    // 停止接受新的 Saga
    c.stopAcceptingNew()
    
    // 等待正在运行的 Saga 完成或超时
    done := make(chan struct{})
    go func() {
        c.waitForActiveSagas()
        close(done)
    }()
    
    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

## 总结

### 关键要点

1. **单一职责**：每个步骤只做一件事
2. **幂等性**：所有操作都应该是幂等的
3. **错误处理**：明确区分可重试和不可重试错误
4. **补偿机制**：确保补偿操作能够正确回滚状态
5. **可观测性**：完整的日志、指标和追踪
6. **安全性**：加密敏感数据，记录审计日志
7. **测试**：全面的单元测试、集成测试和补偿测试

### 扩展阅读

- [Saga 用户指南](../../../../docs/saga-user-guide.md)
- [Saga DSL 指南](../../../../docs/saga-dsl-guide.md)
- [Saga 监控指南](../../../../docs/saga-monitoring-guide.md)
- [订单处理示例](order_saga.md)
- [支付处理示例](payment_saga.md)
- [库存管理示例](inventory_saga.md)
- [用户注册示例](user_registration_saga.md)


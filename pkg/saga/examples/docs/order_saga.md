# 订单处理 Saga 示例文档

## 概述

订单处理 Saga（`order_saga.go`）展示了如何使用 Swit 框架的 Saga 模式处理电商订单的完整流程。这是一个典型的分布式事务场景，涉及多个服务的协调，需要确保数据的一致性。

## 业务场景

### 适用场景

- 电商平台的订单处理流程
- 需要协调多个微服务的业务流程
- 涉及资源预留和释放的场景
- 需要保证最终一致性的分布式事务

### 业务流程

```
客户下单 → 创建订单 → 预留库存 → 处理支付 → 确认订单 → 通知客户
```

当任何一个步骤失败时，系统会自动执行补偿操作，回滚已完成的步骤，确保数据一致性。

## 架构设计

### 服务依赖关系

```
┌─────────────────┐
│  Saga           │
│  Coordinator    │
└────────┬────────┘
         │
    ┌────┴────────────────┬──────────────┬───────────────┐
    │                     │              │               │
┌───▼────────┐   ┌───────▼─────┐  ┌────▼─────────┐  ┌──▼───────────┐
│  订单服务  │   │  库存服务   │  │  支付服务    │  │  通知服务    │
│  (Order)   │   │ (Inventory) │  │  (Payment)   │  │ (Notification)│
└────────────┘   └─────────────┘  └──────────────┘  └──────────────┘
```

### 步骤流程图

```
┌──────────────────────────────────────────────────────────┐
│                    订单处理 Saga                          │
└──────────────────────────────────────────────────────────┘

步骤 1: CreateOrderStep (创建订单)
  ├─ Execute: 在订单系统中创建订单记录
  └─ Compensate: 取消订单

步骤 2: ReserveInventoryStep (预留库存)
  ├─ Execute: 在库存系统中预留商品
  └─ Compensate: 释放预留的库存

步骤 3: ProcessPaymentStep (处理支付)
  ├─ Execute: 通过支付网关处理支付
  └─ Compensate: 执行退款操作

步骤 4: ConfirmOrderStep (确认订单)
  ├─ Execute: 确认订单，更新状态为待发货
  └─ Compensate: 无需补偿（订单取消已处理）
```

### 数据流转

```
OrderData (输入)
    │
    ├─> CreateOrderStep
    │     └─> OrderStepResult
    │           │
    │           ├─> ReserveInventoryStep
    │           │     └─> InventoryStepResult
    │           │           │
    │           │           ├─> ProcessPaymentStep
    │           │           │     └─> PaymentStepResult
    │           │           │           │
    │           │           │           └─> ConfirmOrderStep
    │           │           │                 └─> ConfirmStepResult (输出)
    │           │           │
    │           │           └─ (失败) ─> 补偿 ProcessPaymentStep
    │           │
    │           └─ (失败) ─> 补偿 ReserveInventoryStep
    │
    └─ (失败) ─> 补偿 CreateOrderStep
```

## 数据结构

### 输入数据

#### OrderData

订单处理的主要输入数据结构：

```go
type OrderData struct {
    // 订单基本信息
    OrderID     string      // 订单ID（如果已生成）
    CustomerID  string      // 客户ID
    TotalAmount float64     // 订单总金额
    
    // 订单项列表
    Items []OrderItem      // 订单项
    
    // 配送地址
    Address ShippingAddress // 配送地址
    
    // 支付信息
    PaymentMethod string    // 支付方式
    PaymentToken  string    // 支付令牌
    
    // 元数据
    Metadata map[string]interface{} // 额外元数据
}
```

**字段说明**：
- `OrderID`: 如果调用方已生成订单ID，可预填；否则由创建订单步骤生成
- `CustomerID`: 必填，用于关联客户信息
- `TotalAmount`: 必填，订单总金额，用于支付验证
- `Items`: 必填，订单商品列表
- `Address`: 必填，配送地址信息
- `PaymentMethod`: 支付方式（如 `credit_card`、`alipay`、`wechat_pay`）
- `PaymentToken`: 支付令牌或授权码
- `Metadata`: 可选的扩展信息

#### OrderItem

订单商品项：

```go
type OrderItem struct {
    ProductID string  // 商品ID
    SKU       string  // 库存单位
    Quantity  int     // 数量
    UnitPrice float64 // 单价
    Total     float64 // 小计
}
```

#### ShippingAddress

配送地址：

```go
type ShippingAddress struct {
    RecipientName string // 收件人姓名
    Phone         string // 联系电话
    Province      string // 省份
    City          string // 城市
    District      string // 区/县
    Street        string // 街道地址
    Zipcode       string // 邮编
}
```

### 步骤结果

#### OrderStepResult

创建订单步骤的结果：

```go
type OrderStepResult struct {
    OrderID     string                 // 订单ID
    OrderNumber string                 // 订单号（用于客户查看）
    Status      string                 // 订单状态
    CreatedAt   time.Time              // 创建时间
    Items       []OrderItem            // 订单项
    Metadata    map[string]interface{} // 元数据
}
```

#### InventoryStepResult

预留库存步骤的结果：

```go
type InventoryStepResult struct {
    ReservationID string        // 预留ID
    ReservedItems []ReservedItem // 预留的商品项
    ExpiresAt     time.Time     // 预留过期时间
    WarehouseID   string        // 仓库ID
    Metadata      map[string]interface{} // 元数据
}
```

#### PaymentStepResult

处理支付步骤的结果：

```go
type PaymentStepResult struct {
    TransactionID string    // 交易ID
    PaymentID     string    // 支付ID
    Amount        float64   // 支付金额
    Currency      string    // 货币类型
    Status        string    // 支付状态
    PaidAt        time.Time // 支付时间
    Provider      string    // 支付提供商
    Metadata      map[string]interface{} // 元数据
}
```

#### ConfirmStepResult

确认订单步骤的结果：

```go
type ConfirmStepResult struct {
    OrderID       string    // 订单ID
    ConfirmedAt   time.Time // 确认时间
    Status        string    // 最终状态
    EstimatedShip time.Time // 预计发货时间
    TrackingInfo  string    // 跟踪信息
    Metadata      map[string]interface{} // 元数据
}
```

## 步骤详解

### 步骤 1: CreateOrderStep（创建订单）

#### 功能描述

在订单系统中创建新的订单记录，初始状态为"待支付"。

#### 执行逻辑

```go
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 1. 类型断言，获取 OrderData
    orderData, ok := data.(*OrderData)
    if !ok {
        return nil, errors.New("invalid data type: expected *OrderData")
    }
    
    // 2. 验证订单数据
    if err := s.validateOrderData(orderData); err != nil {
        return nil, fmt.Errorf("订单数据验证失败: %w", err)
    }
    
    // 3. 调用订单服务创建订单
    result, err := s.service.CreateOrder(ctx, orderData)
    if err != nil {
        return nil, fmt.Errorf("创建订单失败: %w", err)
    }
    
    return result, nil
}
```

#### 验证规则

- 客户ID不能为空
- 订单项列表不能为空
- 订单总金额必须大于0
- 收件人姓名不能为空
- 联系电话不能为空

#### 补偿逻辑

```go
func (s *CreateOrderStep) Compensate(ctx context.Context, data interface{}) error {
    result, ok := data.(*OrderStepResult)
    if !ok {
        return errors.New("invalid compensation data type")
    }
    
    // 调用订单服务取消订单
    return s.service.CancelOrder(ctx, result.OrderID, "saga_compensation")
}
```

#### 配置参数

- **超时时间**: 10秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 网络错误和临时服务不可用错误可重试

### 步骤 2: ReserveInventoryStep（预留库存）

#### 功能描述

在库存系统中预留订单所需的商品库存，防止超卖。

#### 执行逻辑

```go
func (s *ReserveInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 1. 获取上一步的订单结果
    orderResult, ok := data.(*OrderStepResult)
    if !ok {
        return nil, errors.New("invalid data type: expected *OrderStepResult")
    }
    
    // 2. 调用库存服务预留库存
    result, err := s.service.ReserveInventory(ctx, orderResult.OrderID, orderResult.Items)
    if err != nil {
        return nil, fmt.Errorf("预留库存失败: %w", err)
    }
    
    return result, nil
}
```

#### 业务规则

- 检查每个商品的可用库存
- 如果库存不足，返回不可重试错误
- 预留成功后，设置过期时间（通常15-30分钟）
- 记录预留ID，用于后续释放

#### 补偿逻辑

```go
func (s *ReserveInventoryStep) Compensate(ctx context.Context, data interface{}) error {
    result, ok := data.(*InventoryStepResult)
    if !ok {
        return errors.New("invalid compensation data type")
    }
    
    // 释放预留的库存
    return s.service.ReleaseInventory(ctx, result.ReservationID)
}
```

**注意**: 库存系统通常有自动过期机制，即使补偿失败，预留也会在过期后自动释放。

#### 配置参数

- **超时时间**: 10秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 网络错误可重试，库存不足不可重试

### 步骤 3: ProcessPaymentStep（处理支付）

#### 功能描述

通过支付网关处理订单支付，扣除客户账户资金。

#### 执行逻辑

```go
func (s *ProcessPaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 1. 获取库存预留结果
    inventoryResult, ok := data.(*InventoryStepResult)
    if !ok {
        return nil, errors.New("invalid data type")
    }
    
    // 2. 从元数据中获取支付信息
    orderID := inventoryResult.Metadata["order_id"].(string)
    amount := inventoryResult.Metadata["amount"].(float64)
    method := inventoryResult.Metadata["payment_method"].(string)
    token := inventoryResult.Metadata["payment_token"].(string)
    
    // 3. 调用支付服务处理支付
    result, err := s.service.ProcessPayment(ctx, orderID, amount, method, token)
    if err != nil {
        return nil, fmt.Errorf("处理支付失败: %w", err)
    }
    
    return result, nil
}
```

#### 业务规则

- 验证支付金额与订单金额一致
- 检查支付方式的有效性
- 调用支付网关进行实际扣款
- 记录支付交易ID，用于对账和退款

#### 补偿逻辑

```go
func (s *ProcessPaymentStep) Compensate(ctx context.Context, data interface{}) error {
    result, ok := data.(*PaymentStepResult)
    if !ok {
        return errors.New("invalid compensation data type")
    }
    
    // 执行退款操作
    return s.service.RefundPayment(ctx, result.TransactionID, "saga_compensation")
}
```

**重要**: 退款是关键操作，如果失败需要人工介入处理。

#### 配置参数

- **超时时间**: 30秒（支付操作可能需要较长时间）
- **重试策略**: 指数退避，最多重试2次（避免重复扣款）
- **可重试**: 非常保守，只有明确的网络临时错误才重试

### 步骤 4: ConfirmOrderStep（确认订单）

#### 功能描述

确认订单已支付，更新订单状态为"待发货"，准备物流信息。

#### 执行逻辑

```go
func (s *ConfirmOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 1. 获取支付结果
    paymentResult, ok := data.(*PaymentStepResult)
    if !ok {
        return nil, errors.New("invalid data type")
    }
    
    // 2. 从元数据中获取订单ID
    orderID := paymentResult.Metadata["order_id"].(string)
    
    // 3. 调用订单服务确认订单
    result, err := s.service.ConfirmOrder(ctx, orderID)
    if err != nil {
        return nil, fmt.Errorf("确认订单失败: %w", err)
    }
    
    return result, nil
}
```

#### 业务规则

- 更新订单状态为"待发货"
- 生成物流跟踪信息
- 记录预计发货时间
- 触发发货流程（异步）

#### 补偿逻辑

```go
func (s *ConfirmOrderStep) Compensate(ctx context.Context, data interface{}) error {
    // 订单确认步骤通常不需要单独补偿
    // 因为订单取消（在 CreateOrderStep 的补偿中执行）已经处理了状态回退
    return nil
}
```

#### 配置参数

- **超时时间**: 10秒
- **重试策略**: 指数退避，最多重试3次
- **可重试**: 网络错误可重试

## 使用示例

### 基本用法

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/innovationmech/swit/pkg/saga"
    "github.com/innovationmech/swit/pkg/saga/coordinator"
    "github.com/innovationmech/swit/pkg/saga/examples"
)

func main() {
    // 1. 创建状态存储和事件发布器
    stateStorage := coordinator.NewInMemoryStateStorage()
    eventPublisher := coordinator.NewInMemoryEventPublisher()
    
    // 2. 配置 Coordinator
    config := &coordinator.OrchestratorConfig{
        StateStorage:   stateStorage,
        EventPublisher: eventPublisher,
        RetryPolicy:    saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
    }
    
    // 3. 创建 Coordinator
    sagaCoordinator, err := coordinator.NewOrchestratorCoordinator(config)
    if err != nil {
        log.Fatal(err)
    }
    defer sagaCoordinator.Close()
    
    // 4. 创建服务客户端（实际环境中应注入真实实现）
    orderService := NewMockOrderService()
    inventoryService := NewMockInventoryService()
    paymentService := NewMockPaymentService()
    
    // 5. 创建 Saga 定义
    sagaDef := examples.NewOrderProcessingSaga(
        orderService,
        inventoryService,
        paymentService,
    )
    
    // 6. 准备订单数据
    orderData := &examples.OrderData{
        CustomerID:  "CUST-12345",
        TotalAmount: 299.99,
        Items: []examples.OrderItem{
            {
                ProductID: "PROD-001",
                SKU:       "SKU-001",
                Quantity:  2,
                UnitPrice: 149.99,
                Total:     299.98,
            },
        },
        Address: examples.ShippingAddress{
            RecipientName: "张三",
            Phone:         "13800138000",
            Province:      "北京市",
            City:          "北京市",
            District:      "朝阳区",
            Street:        "朝阳路100号",
            Zipcode:       "100000",
        },
        PaymentMethod: "credit_card",
        PaymentToken:  "tok_1234567890",
    }
    
    // 7. 启动 Saga
    ctx := context.Background()
    instance, err := sagaCoordinator.StartSaga(ctx, sagaDef, orderData)
    if err != nil {
        log.Fatalf("启动 Saga 失败: %v", err)
    }
    
    log.Printf("Saga 已启动，实例 ID: %s", instance.GetID())
    
    // 8. 等待完成（实际应用中通常是异步的）
    time.Sleep(5 * time.Second)
    
    // 9. 检查最终状态
    finalInstance, err := sagaCoordinator.GetSagaInstance(instance.GetID())
    if err != nil {
        log.Fatalf("获取 Saga 实例失败: %v", err)
    }
    
    log.Printf("Saga 最终状态: %s", finalInstance.GetStatus())
}
```

### 错误处理示例

```go
// 处理库存不足的情况
orderData := &examples.OrderData{
    CustomerID:  "CUST-12345",
    TotalAmount: 999.99,
    Items: []examples.OrderItem{
        {
            ProductID: "PROD-OUT-OF-STOCK",  // 缺货商品
            Quantity:  100,
        },
    },
    // ... 其他字段
}

instance, err := sagaCoordinator.StartSaga(ctx, sagaDef, orderData)
if err != nil {
    log.Printf("Saga 启动失败: %v", err)
    return
}

// 等待执行完成
time.Sleep(5 * time.Second)

// 检查状态
finalInstance, _ := sagaCoordinator.GetSagaInstance(instance.GetID())
if finalInstance.GetStatus() == saga.StatusCompensated {
    log.Println("订单处理失败，已回滚所有操作")
    // 从状态中获取失败原因
    if err := finalInstance.GetError(); err != nil {
        log.Printf("失败原因: %v", err)
    }
}
```

### 监听 Saga 事件

```go
// 订阅 Saga 事件
eventPublisher.Subscribe(func(event *saga.SagaEvent) {
    switch event.Type {
    case saga.EventSagaStarted:
        log.Printf("Saga 已启动: %s", event.SagaID)
        
    case saga.EventStepStarted:
        log.Printf("步骤开始: %s", event.StepID)
        
    case saga.EventStepCompleted:
        log.Printf("步骤完成: %s", event.StepID)
        
    case saga.EventStepFailed:
        log.Printf("步骤失败: %s, 错误: %v", event.StepID, event.Error)
        
    case saga.EventCompensationStarted:
        log.Printf("开始补偿: %s", event.StepID)
        
    case saga.EventSagaCompleted:
        log.Printf("Saga 完成: %s", event.SagaID)
        
    case saga.EventSagaFailed:
        log.Printf("Saga 失败: %s, 错误: %v", event.SagaID, event.Error)
    }
})
```

## 最佳实践

### 1. 数据传递

**推荐做法**：使用元数据传递跨步骤的信息

```go
// 在创建库存预留结果时，添加支付所需信息
result := &InventoryStepResult{
    ReservationID: "RES-123",
    // ... 其他字段
    Metadata: map[string]interface{}{
        "order_id":       orderID,
        "amount":         totalAmount,
        "payment_method": paymentMethod,
        "payment_token":  paymentToken,
    },
}
```

### 2. 幂等性设计

所有步骤和补偿操作都应该是幂等的：

```go
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 检查订单是否已存在（实现幂等性）
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

### 3. 错误分类

明确区分可重试和不可重试的错误：

```go
type OrderError struct {
    Code      string
    Message   string
    Retryable bool
}

func (e *OrderError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// 在步骤中使用
func (s *CreateOrderStep) IsRetryable(err error) bool {
    if orderErr, ok := err.(*OrderError); ok {
        return orderErr.Retryable
    }
    return true // 默认可重试
}
```

### 4. 日志记录

记录详细的执行日志，便于问题排查：

```go
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    log.Printf("[CreateOrderStep] 开始执行，客户ID: %s", orderData.CustomerID)
    
    result, err := s.service.CreateOrder(ctx, orderData)
    
    if err != nil {
        log.Printf("[CreateOrderStep] 执行失败: %v", err)
        return nil, err
    }
    
    log.Printf("[CreateOrderStep] 执行成功，订单ID: %s", result.OrderID)
    return result, nil
}
```

### 5. 超时配置

根据业务特点合理配置超时：

```go
// Saga 总超时
sagaDef.SetTimeout(5 * time.Minute)

// 关键步骤设置较长超时
paymentStep.SetTimeout(30 * time.Second)

// 快速步骤设置较短超时
orderStep.SetTimeout(10 * time.Second)
```

## 测试

### 单元测试

参见 `order_saga_test.go`：

```bash
# 运行订单处理测试
go test -v -run TestOrderSaga

# 运行特定测试用例
go test -v -run TestOrderSagaSuccess
go test -v -run TestOrderSagaInventoryFailure
go test -v -run TestOrderSagaPaymentFailure
```

### 测试场景

1. **成功场景**: 所有步骤成功执行
2. **库存不足**: 在预留库存步骤失败，验证订单被取消
3. **支付失败**: 在支付步骤失败，验证库存被释放、订单被取消
4. **补偿失败**: 模拟补偿操作失败的处理
5. **超时场景**: 测试步骤超时的处理

## 常见问题

### Q1: 如何处理部分商品库存不足的情况？

**A**: 有两种策略：
1. **全或无策略**（推荐）：任何商品库存不足都导致整个订单失败
2. **部分履行策略**：创建部分订单，通知客户缺货情况

```go
// 全或无策略
func (s *ReserveInventoryStep) Execute(...) (interface{}, error) {
    // 先检查所有商品库存
    for _, item := range items {
        available, err := s.service.CheckInventory(ctx, item.SKU)
        if err != nil || available < item.Quantity {
            return nil, &InventoryError{
                Code:      "INSUFFICIENT_STOCK",
                SKU:       item.SKU,
                Required:  item.Quantity,
                Available: available,
                Retryable: false,
            }
        }
    }
    
    // 全部可用，执行预留
    return s.service.ReserveAll(ctx, items)
}
```

### Q2: 支付步骤如何避免重复扣款？

**A**: 
1. 使用幂等性令牌
2. 支付网关通常会防止重复交易
3. 在应用层检查订单支付状态

```go
func (s *ProcessPaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 生成幂等性密钥
    idempotencyKey := fmt.Sprintf("order-%s-%d", orderID, time.Now().Unix())
    
    result, err := s.service.ProcessPayment(ctx, &PaymentRequest{
        OrderID:        orderID,
        Amount:         amount,
        IdempotencyKey: idempotencyKey,
    })
    
    return result, err
}
```

### Q3: 补偿失败怎么办？

**A**: 
1. 配置补偿重试策略
2. 记录补偿失败的详细日志
3. 设置告警，通知运维人员
4. 提供手动补偿工具

```go
// 配置补偿策略
compensationStrategy := saga.NewSequentialCompensationStrategy(
    3 * time.Minute, // 补偿超时
    saga.WithCompensationRetry(3, time.Second),
    saga.WithCompensationAlertOnFailure(true),
)
```

### Q4: 如何监控 Saga 的执行情况？

**A**: 
1. 使用事件发布器监听 Saga 事件
2. 集成 Prometheus 收集指标
3. 使用 Jaeger 进行分布式追踪
4. 实现 Saga 监控面板

参考：[Saga 监控指南](../../../../docs/saga-monitoring-guide.md)

## 生产环境部署

### 配置建议

```yaml
saga:
  order_processing:
    # 全局超时
    timeout: 5m
    
    # 重试策略
    retry:
      max_attempts: 3
      initial_interval: 1s
      max_interval: 30s
      multiplier: 2.0
    
    # 补偿策略
    compensation:
      timeout: 3m
      retry_attempts: 3
      alert_on_failure: true
    
    # 步骤配置
    steps:
      create_order:
        timeout: 10s
        retry_attempts: 3
      reserve_inventory:
        timeout: 10s
        retry_attempts: 3
      process_payment:
        timeout: 30s
        retry_attempts: 2
      confirm_order:
        timeout: 10s
        retry_attempts: 3
```

### 依赖服务

- **状态存储**: Redis 或 PostgreSQL（替换内存存储）
- **消息队列**: NATS 或 RabbitMQ（用于事件发布）
- **监控**: Prometheus + Grafana
- **追踪**: Jaeger
- **日志**: ELK Stack

## 相关资源

- [Saga 用户指南](../../../../docs/saga-user-guide.md)
- [架构设计文档](architecture.md)
- [支付处理 Saga](payment_saga.md)
- [库存管理 Saga](inventory_saga.md)

## 参考文献

- [Saga Pattern - Microsoft](https://docs.microsoft.com/en-us/azure/architecture/reference-architectures/saga/saga)
- [Microservices Patterns - Chris Richardson](https://microservices.io/patterns/data/saga.html)


# Saga 分布式事务最佳实践

本文档总结了使用 Swit 框架实现 Saga 分布式事务的最佳实践、设计模式和常见陷阱。

## 目录

- [设计原则](#设计原则)
- [Saga 设计模式](#saga-设计模式)
- [步骤设计最佳实践](#步骤设计最佳实践)
- [补偿操作设计](#补偿操作设计)
- [错误处理策略](#错误处理策略)
- [性能优化](#性能优化)
- [安全性考虑](#安全性考虑)
- [监控和可观测性](#监控和可观测性)
- [常见陷阱和反模式](#常见陷阱和反模式)
- [Do's and Don'ts 清单](#dos-and-donts-清单)

---

## 设计原则

### 1. 单一职责原则 (SRP)

每个步骤应该只负责一个明确的业务操作。

#### ✅ 好的设计

```go
// 步骤 1: 只负责创建订单
type CreateOrderStep struct {
    service OrderService
}

// 步骤 2: 只负责预留库存
type ReserveInventoryStep struct {
    service InventoryService
}

// 步骤 3: 只负责处理支付
type ProcessPaymentStep struct {
    service PaymentService
}
```

#### ❌ 不好的设计

```go
// 一个步骤做太多事情
type CreateOrderAndProcessPaymentStep struct {
    orderService   OrderService
    inventoryService InventoryService
    paymentService PaymentService
}

func (s *CreateOrderAndProcessPaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 创建订单
    order := s.orderService.CreateOrder(ctx, data)
    
    // 预留库存
    s.inventoryService.Reserve(ctx, order.Items)
    
    // 处理支付
    s.paymentService.Process(ctx, order.Amount)
    
    // 问题: 如果支付失败,前面的操作如何回滚?
    return order, nil
}
```

**为什么**: 
- 单一职责使得步骤更容易测试
- 补偿逻辑更清晰
- 可以独立重试失败的步骤
- 便于并行执行无依赖的步骤

### 2. 幂等性原则

所有步骤和补偿操作都必须是幂等的(多次执行结果相同)。

#### ✅ 好的设计

```go
type ReserveInventoryStep struct {
    service InventoryService
}

func (s *ReserveInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 幂等性检查
    if orderData.ReservationID != "" {
        // 检查预留是否已存在
        existing, err := s.service.GetReservation(ctx, orderData.ReservationID)
        if err == nil && existing != nil && existing.Status == "ACTIVE" {
            // 预留已存在,直接返回
            return orderData, nil
        }
    }
    
    // 执行新预留
    reservation, err := s.service.Reserve(ctx, orderData.Items)
    if err != nil {
        return nil, err
    }
    
    orderData.ReservationID = reservation.ID
    return orderData, nil
}

func (s *ReserveInventoryStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    if orderData.ReservationID == "" {
        // 没有预留,幂等返回成功
        return nil
    }
    
    // 检查预留是否已释放
    reservation, err := s.service.GetReservation(ctx, orderData.ReservationID)
    if err != nil || reservation == nil {
        // 预留不存在,认为已释放
        return nil
    }
    
    if reservation.Status == "RELEASED" {
        // 已释放,幂等返回成功
        return nil
    }
    
    // 执行释放
    return s.service.Release(ctx, orderData.ReservationID)
}
```

#### ❌ 不好的设计

```go
func (s *ReserveInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 直接执行预留,没有幂等性检查
    reservation, err := s.service.Reserve(ctx, orderData.Items)
    if err != nil {
        return nil, err
    }
    
    // 问题: 如果步骤重试,会创建多个预留
    orderData.ReservationID = reservation.ID
    return orderData, nil
}
```

**为什么**:
- 步骤可能因为超时而被重试
- 网络问题可能导致重复执行
- 补偿操作可能被多次调用
- 幂等性确保系统状态一致

### 3. 原子性原则

每个步骤应该是一个原子操作,要么全部成功,要么全部失败。

#### ✅ 好的设计

```go
type ProcessPaymentStep struct {
    service PaymentService
}

func (s *ProcessPaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 在一个事务中完成所有支付相关操作
    transaction, err := s.service.BeginTransaction(ctx)
    if err != nil {
        return nil, err
    }
    defer transaction.Rollback()
    
    // 1. 授权支付
    auth, err := transaction.Authorize(orderData.PaymentMethod, orderData.Amount)
    if err != nil {
        return nil, err
    }
    
    // 2. 捕获资金
    capture, err := transaction.Capture(auth.ID)
    if err != nil {
        return nil, err
    }
    
    // 3. 记录交易
    err = transaction.RecordTransaction(capture)
    if err != nil {
        return nil, err
    }
    
    // 提交事务
    if err := transaction.Commit(); err != nil {
        return nil, err
    }
    
    orderData.TransactionID = capture.ID
    return orderData, nil
}
```

#### ❌ 不好的设计

```go
func (s *ProcessPaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 操作 1
    auth, err := s.service.Authorize(ctx, orderData.PaymentMethod, orderData.Amount)
    if err != nil {
        return nil, err
    }
    
    // 操作 2 - 如果这里失败,前面的授权无法回滚
    capture, err := s.service.Capture(ctx, auth.ID)
    if err != nil {
        // 问题: 授权已完成但捕获失败,资金被锁定
        return nil, err
    }
    
    // 操作 3 - 如果这里失败,前面的操作都无法回滚
    err = s.service.RecordTransaction(ctx, capture)
    if err != nil {
        // 问题: 钱已扣但没记录
        return nil, err
    }
    
    return orderData, nil
}
```

**为什么**:
- 避免部分成功的情况
- 简化错误处理
- 确保数据一致性
- 减少需要补偿的复杂度

### 4. 向后兼容性原则

Saga 定义和步骤应该支持版本演进。

#### ✅ 好的设计

```go
type OrderData struct {
    // 版本 1.0 字段
    OrderID    string
    CustomerID string
    Amount     float64
    
    // 版本 2.0 新增字段 (使用指针,可选)
    Currency          *string
    ShippingAddress   *Address
    PromoCode         *string
    
    // 版本字段
    SchemaVersion string
}

func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 处理版本差异
    currency := "USD"
    if orderData.Currency != nil {
        currency = *orderData.Currency
    }
    
    // 使用默认值处理可选字段
    // ...
}
```

#### ❌ 不好的设计

```go
// 直接修改现有字段,破坏向后兼容性
type OrderData struct {
    OrderID    string
    CustomerID string
    Amount     float64
    Currency   string  // 新增必填字段 - 破坏兼容性!
}
```

**为什么**:
- 生产环境可能同时运行不同版本
- 长时间运行的 Saga 可能跨越版本升级
- 回滚部署时需要向后兼容

---

## Saga 设计模式

### 模式 1: 顺序执行模式

**适用场景**: 步骤之间有严格的依赖关系,必须按顺序执行。

**示例**: 电商订单处理

```
验证订单 → 预留库存 → 处理支付 → 确认订单 → 发送通知
```

**实现**:

```go
func NewOrderProcessingSaga() saga.SagaDefinition {
    return &OrderSaga{
        steps: []saga.SagaStep{
            &ValidateOrderStep{},
            &ReserveInventoryStep{},
            &ProcessPaymentStep{},
            &ConfirmOrderStep{},
            &SendNotificationStep{},
        },
    }
}
```

**优点**:
- 逻辑清晰,易于理解
- 便于调试和追踪
- 状态管理简单

**缺点**:
- 执行时间较长
- 无法利用并行优化

### 模式 2: 并行执行模式

**适用场景**: 多个步骤之间没有依赖关系,可以同时执行。

**示例**: 用户注册

```
               ┌─ 创建用户档案 ─┐
验证用户信息 ──┤                ├── 激活账户
               └─ 发送欢迎邮件 ─┘
```

**DSL 实现**:

```yaml
steps:
  - id: validate-user
    name: Validate User
    # ...

  # 并行步骤 - 无依赖关系
  - id: create-profile
    name: Create User Profile
    dependencies:
      - validate-user

  - id: send-welcome-email
    name: Send Welcome Email
    dependencies:
      - validate-user

  # 等待并行步骤完成
  - id: activate-account
    name: Activate Account
    dependencies:
      - create-profile
      - send-welcome-email
```

**优点**:
- 减少总执行时间
- 提高吞吐量
- 更好地利用资源

**注意事项**:
- 确保步骤真正独立
- 处理并行步骤的部分失败
- 需要更复杂的状态管理

### 模式 3: 条件执行模式

**适用场景**: 根据业务规则动态决定执行哪些步骤。

**示例**: VIP 客户优惠

```
验证订单 → [if VIP] 应用折扣 → 处理支付 → 确认订单
```

**实现**:

```go
type ApplyDiscountStep struct {
    condition func(interface{}) bool
}

func (s *ApplyDiscountStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 检查条件
    if s.condition != nil && !s.condition(data) {
        // 不满足条件,跳过
        return orderData, nil
    }
    
    // 应用折扣
    discount := s.calculateDiscount(orderData)
    orderData.Amount -= discount
    orderData.DiscountApplied = true
    
    return orderData, nil
}
```

**DSL 实现**:

```yaml
steps:
  - id: apply-discount
    name: Apply VIP Discount
    condition:
      expression: "$input.customer_level == 'VIP' && $input.amount > 100"
    action:
      service:
        name: pricing-service
        method: POST
        path: /api/discounts/apply
```

**优点**:
- 灵活适应不同业务场景
- 避免不必要的操作
- 简化流程定义

**注意事项**:
- 条件判断要快速且可靠
- 记录条件评估结果用于审计
- 补偿操作也要考虑条件

### 模式 4: 重试与降级模式

**适用场景**: 对于非关键步骤,失败后可以降级处理。

**示例**: 发送通知

```
处理订单 → [尝试发送邮件] → [失败→降级为短信] → [失败→记录到队列]
```

**实现**:

```go
type NotificationStep struct {
    emailService EmailService
    smsService   SMSService
    queueService QueueService
}

func (s *NotificationStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 首选: 邮件通知
    err := s.emailService.Send(ctx, orderData.Email, "订单确认")
    if err == nil {
        orderData.NotificationSent = "email"
        return orderData, nil
    }
    
    // 降级 1: 短信通知
    err = s.smsService.Send(ctx, orderData.Phone, "订单已确认")
    if err == nil {
        orderData.NotificationSent = "sms"
        return orderData, nil
    }
    
    // 降级 2: 加入异步队列
    err = s.queueService.Enqueue(ctx, &NotificationTask{
        OrderID:  orderData.OrderID,
        Email:    orderData.Email,
        Template: "order-confirmation",
    })
    if err == nil {
        orderData.NotificationSent = "queued"
        return orderData, nil
    }
    
    // 所有方式都失败,记录日志但不阻塞流程
    log.Errorf("Failed to send notification for order %s", orderData.OrderID)
    orderData.NotificationSent = "failed"
    return orderData, nil // 不返回错误,允许 Saga 继续
}

func (s *NotificationStep) Compensate(ctx context.Context, data interface{}) error {
    // 通知步骤无需补偿
    return nil
}

func (s *NotificationStep) IsRetryable(err error) bool {
    // 不重试,直接降级
    return false
}
```

**优点**:
- 提高系统可用性
- 优雅降级
- 非关键操作不阻塞主流程

**适用场景**:
- 通知类操作
- 日志记录
- 非关键数据更新
- 统计和分析

### 模式 5: 补偿链模式

**适用场景**: 复杂的补偿逻辑,需要多个补偿步骤。

**示例**: 取消复杂订单

```
补偿顺序: 
1. 退款 
2. 释放库存 
3. 取消物流 
4. 发送取消通知 
5. 更新订单状态
```

**实现**:

```go
type CompensationChain struct {
    compensators []Compensator
}

type Compensator interface {
    Compensate(context.Context, interface{}) error
    GetName() string
}

func (c *CompensationChain) Compensate(ctx context.Context, data interface{}) error {
    errors := make([]error, 0)
    
    // 按顺序执行所有补偿器
    for _, compensator := range c.compensators {
        err := compensator.Compensate(ctx, data)
        if err != nil {
            log.Errorf("Compensator %s failed: %v", compensator.GetName(), err)
            errors = append(errors, err)
            // 继续执行其他补偿器,不中断
        }
    }
    
    if len(errors) > 0 {
        // 记录所有错误但返回成功
        // 避免阻塞其他补偿操作
        log.Errorf("Compensation chain completed with %d errors", len(errors))
    }
    
    return nil
}
```

**优点**:
- 补偿逻辑模块化
- 可以独立测试每个补偿器
- 支持部分补偿失败

### 模式 6: 两阶段提交模拟 (TCC 模式)

**适用场景**: 需要更强一致性保证的场景。

**TCC 三个阶段**:
1. **Try**: 预留资源
2. **Confirm**: 确认操作
3. **Cancel**: 取消操作(补偿)

**示例**: 资金转账

```go
// 账户服务需要支持 Try-Confirm-Cancel
type TransferFundsStep struct {
    service AccountService
}

func (s *TransferFundsStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    transferData := data.(*TransferData)
    
    // Try 阶段: 冻结资金
    freezeID, err := s.service.FreezeAmount(ctx, &FreezeRequest{
        AccountID: transferData.FromAccount,
        Amount:    transferData.Amount,
        Reason:    "transfer",
    })
    if err != nil {
        return nil, err
    }
    
    transferData.FreezeID = freezeID
    
    // Confirm 阶段: 执行转账
    transaction, err := s.service.Transfer(ctx, &TransferRequest{
        FreezeID:    freezeID,
        ToAccount:   transferData.ToAccount,
        Amount:      transferData.Amount,
    })
    if err != nil {
        // Confirm 失败,自动触发补偿
        return nil, err
    }
    
    transferData.TransactionID = transaction.ID
    return transferData, nil
}

func (s *TransferFundsStep) Compensate(ctx context.Context, data interface{}) error {
    transferData := data.(*TransferData)
    
    // Cancel 阶段: 解冻资金
    if transferData.FreezeID != "" {
        err := s.service.UnfreezeAmount(ctx, transferData.FreezeID)
        if err != nil {
            log.Errorf("Failed to unfreeze: %v", err)
        }
    }
    
    // 如果已经转账,执行反向转账
    if transferData.TransactionID != "" {
        err := s.service.ReverseTransfer(ctx, transferData.TransactionID)
        if err != nil {
            log.Errorf("Failed to reverse transfer: %v", err)
        }
    }
    
    return nil
}
```

**优点**:
- 更强的一致性保证
- 资源锁定时间可控
- 减少不一致的窗口期

**缺点**:
- 实现复杂度较高
- 需要服务支持 TCC 接口
- 性能开销较大

---

## 步骤设计最佳实践

### 1. 步骤粒度

#### ✅ 合适的粒度

```go
// 每个步骤一个职责,粒度适中
steps := []saga.SagaStep{
    &ValidateOrderStep{},        // 验证订单
    &CheckInventoryStep{},       // 检查库存
    &ReserveInventoryStep{},     // 预留库存
    &AuthorizePaymentStep{},     // 授权支付
    &CapturePaymentStep{},       // 捕获支付
    &ConfirmOrderStep{},         // 确认订单
    &NotifyCustomerStep{},       // 通知客户
}
```

#### ❌ 粒度过大

```go
// 一个步骤做太多事情
steps := []saga.SagaStep{
    &ValidateAndReserveStep{},   // 验证+预留,粒度太大
    &PaymentAndConfirmStep{},    // 支付+确认,粒度太大
}
```

#### ❌ 粒度过小

```go
// 步骤过于细碎
steps := []saga.SagaStep{
    &ParseOrderDataStep{},
    &ValidateOrderIDStep{},
    &ValidateCustomerIDStep{},
    &ValidateItemsStep{},
    &ValidatePriceStep{},
    &ValidateAddressStep{},
    // 太多琐碎步骤,难以管理
}
```

**指导原则**:
- 每个步骤对应一个有意义的业务操作
- 步骤应该可以独立重试
- 步骤应该有清晰的补偿语义
- 避免过度拆分和过度聚合

### 2. 步骤命名

#### ✅ 好的命名

```go
&CreateOrderStep{}           // 动词开头,描述操作
&ReserveInventoryStep{}      // 清晰的动作
&ProcessPaymentStep{}        // 表达业务意图
&SendConfirmationEmailStep{} // 完整描述操作
```

#### ❌ 不好的命名

```go
&Step1{}                    // 没有语义
&DoStuff{}                  // 过于模糊
&HandleOrderProcessing{}    // 太宽泛
&XYZStep{}                  // 缩写不清晰
```

**命名规范**:
- 使用动词开头 (Create, Process, Send, Validate, etc.)
- 包含操作的对象 (Order, Payment, Inventory)
- 具体且富有表达力
- 避免缩写和模糊词汇

### 3. 步骤超时设置

```go
type Step struct {
    operation string
}

func (s *Step) GetTimeout() time.Duration {
    switch s.operation {
    case "database":
        return 5 * time.Second   // 数据库操作
    case "payment":
        return 30 * time.Second  // 支付操作
    case "email":
        return 10 * time.Second  // 邮件发送
    case "export":
        return 5 * time.Minute   // 导出操作
    default:
        return 10 * time.Second  // 默认超时
    }
}
```

**超时设置原则**:
- 根据操作类型设置合理超时
- 考虑 P99 延迟而不是平均延迟
- 快速失败优于长时间等待
- 外部 API 调用要有更长的超时
- 定期review和调整超时配置

### 4. 数据传递

#### ✅ 好的设计 - 结构化数据

```go
type OrderData struct {
    // 输入数据
    OrderID    string
    CustomerID string
    Items      []OrderItem
    Amount     float64
    
    // 步骤产生的数据
    OrderNumber      string    // CreateOrderStep 产生
    ReservationID    string    // ReserveInventoryStep 产生
    TransactionID    string    // ProcessPaymentStep 产生
    ConfirmationCode string    // ConfirmOrderStep 产生
    
    // 元数据
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (s *ProcessPaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 使用前面步骤的数据
    order := orderData.OrderNumber
    amount := orderData.Amount
    
    // 执行支付
    transaction, err := s.service.Process(ctx, order, amount)
    if err != nil {
        return nil, err
    }
    
    // 保存步骤结果
    orderData.TransactionID = transaction.ID
    orderData.UpdatedAt = time.Now()
    
    // 返回更新后的数据
    return orderData, nil
}
```

#### ❌ 不好的设计 - 使用 map

```go
// 使用 map[string]interface{} 缺乏类型安全
func (s *ProcessPaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    dataMap := data.(map[string]interface{})
    
    // 类型断言容易出错
    amount := dataMap["amount"].(float64)
    orderId := dataMap["order_id"].(string)
    
    // ...
}
```

**数据传递原则**:
- 使用强类型结构体
- 明确输入和输出字段
- 包含足够的上下文信息
- 避免传递过大的对象
- 考虑数据序列化成本

---

## 补偿操作设计

### 1. 补偿语义

#### 语义型补偿 (推荐)

恢复业务语义,而不是精确撤销操作。

```go
// CreateOrderStep - 创建订单
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    order, err := s.service.CreateOrder(ctx, orderData)
    if err != nil {
        return nil, err
    }
    
    orderData.OrderID = order.ID
    orderData.Status = "CREATED"
    return orderData, nil
}

// 补偿: 不是删除订单,而是标记为已取消
func (s *CreateOrderStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    if orderData.OrderID == "" {
        return nil
    }
    
    // 语义补偿: 取消订单(保留审计记录)
    err := s.service.CancelOrder(ctx, orderData.OrderID, &CancelRequest{
        Reason:    "saga_compensation",
        Timestamp: time.Now(),
    })
    
    if err != nil {
        log.Warnf("Failed to cancel order %s: %v", orderData.OrderID, err)
    }
    
    orderData.Status = "CANCELLED"
    return nil
}
```

#### 精确撤销补偿

只在必要时使用精确撤销。

```go
// ReserveInventoryStep - 预留库存
func (s *ReserveInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    reservation, err := s.service.Reserve(ctx, orderData.Items)
    if err != nil {
        return nil, err
    }
    
    orderData.ReservationID = reservation.ID
    return orderData, nil
}

// 补偿: 精确释放预留的库存
func (s *ReserveInventoryStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    if orderData.ReservationID == "" {
        return nil
    }
    
    // 精确撤销: 释放预留
    return s.service.ReleaseReservation(ctx, orderData.ReservationID)
}
```

### 2. 补偿失败处理

补偿操作可能失败,需要优雅处理。

#### 策略 1: 尽力而为 + 告警

```go
func (s *ProcessPaymentStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    maxAttempts := 3
    var lastErr error
    
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        err := s.service.Refund(ctx, orderData.TransactionID)
        if err == nil {
            log.Infof("Refund successful on attempt %d", attempt)
            return nil
        }
        
        lastErr = err
        log.Warnf("Refund attempt %d failed: %v", attempt, err)
        
        if attempt < maxAttempts {
            time.Sleep(time.Duration(attempt) * time.Second)
        }
    }
    
    // 所有尝试失败,发送告警
    s.alertService.SendAlert(AlertCritical, fmt.Sprintf(
        "Manual refund required: TransactionID=%s, OrderID=%s, Error=%v",
        orderData.TransactionID,
        orderData.OrderID,
        lastErr,
    ))
    
    // 记录到死信队列
    s.dlqService.Enqueue(&CompensationTask{
        SagaID:        orderData.SagaID,
        StepID:        s.GetID(),
        TransactionID: orderData.TransactionID,
        Error:         lastErr.Error(),
        Attempts:      maxAttempts,
    })
    
    // 返回 nil 允许其他补偿继续
    return nil
}
```

#### 策略 2: 补偿队列

```go
type CompensationQueue struct {
    queue chan CompensationTask
}

type CompensationTask struct {
    SagaID        string
    StepID        string
    Data          interface{}
    Compensator   saga.SagaStep
    Attempts      int
    NextRetry     time.Time
}

func (cq *CompensationQueue) Enqueue(task CompensationTask) {
    cq.queue <- task
}

func (cq *CompensationQueue) ProcessLoop() {
    for task := range cq.queue {
        if time.Now().Before(task.NextRetry) {
            // 延迟重试
            time.Sleep(time.Until(task.NextRetry))
        }
        
        err := task.Compensator.Compensate(context.Background(), task.Data)
        if err != nil {
            task.Attempts++
            if task.Attempts < 10 {
                // 指数退避重试
                task.NextRetry = time.Now().Add(time.Duration(1<<task.Attempts) * time.Second)
                cq.Enqueue(task)
            } else {
                // 超过最大重试,人工介入
                log.Errorf("Compensation failed after %d attempts: %v", task.Attempts, err)
            }
        }
    }
}
```

### 3. 补偿顺序

#### 顺序补偿 (默认)

按照执行步骤的逆序进行补偿。

```go
执行顺序: Step1 → Step2 → Step3 → Step4
补偿顺序: Step3 ← Step2 ← Step1
```

**适用场景**:
- 步骤之间有依赖关系
- 需要确保补偿顺序
- 大部分场景

#### 并行补偿

所有步骤同时进行补偿。

```yaml
global_compensation:
  strategy: parallel
  timeout: 2m
```

**适用场景**:
- 步骤完全独立
- 需要快速补偿
- 资源竞争风险低

```go
type ParallelCompensationStrategy struct {
    timeout time.Duration
}

func (s *ParallelCompensationStrategy) Compensate(
    ctx context.Context,
    steps []saga.SagaStep,
    data interface{},
) error {
    var wg sync.WaitGroup
    errors := make(chan error, len(steps))
    
    for _, step := range steps {
        wg.Add(1)
        go func(step saga.SagaStep) {
            defer wg.Done()
            
            err := step.Compensate(ctx, data)
            if err != nil {
                errors <- err
            }
        }(step)
    }
    
    wg.Wait()
    close(errors)
    
    // 收集错误
    var errs []error
    for err := range errors {
        errs = append(errs, err)
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("compensation failed: %v", errs)
    }
    
    return nil
}
```

#### 尽力而为补偿

即使某些补偿失败,也继续执行其他补偿。

```yaml
global_compensation:
  strategy: best_effort
  timeout: 5m
```

**适用场景**:
- 补偿操作可能失败
- 需要尝试所有补偿
- 有后续人工介入流程

---

## 错误处理策略

### 1. 错误分类

根据错误类型采取不同策略:

```go
type ErrorClassifier struct{}

func (c *ErrorClassifier) Classify(err error) ErrorType {
    if err == nil {
        return ErrorTypeNone
    }
    
    // 网络错误 - 可重试
    if errors.Is(err, context.DeadlineExceeded) ||
       errors.Is(err, syscall.ECONNREFUSED) ||
       strings.Contains(err.Error(), "connection") {
        return ErrorTypeNetwork
    }
    
    // 服务不可用 - 可重试
    if errors.Is(err, ErrServiceUnavailable) ||
       strings.Contains(err.Error(), "503") ||
       strings.Contains(err.Error(), "unavailable") {
        return ErrorTypeServiceUnavailable
    }
    
    // 资源锁定 - 可重试
    if errors.Is(err, ErrResourceLocked) ||
       strings.Contains(err.Error(), "locked") {
        return ErrorTypeResourceLocked
    }
    
    // 业务错误 - 不可重试
    if errors.Is(err, ErrInvalidInput) ||
       errors.Is(err, ErrBusinessRule) {
        return ErrorTypeBusiness
    }
    
    // 权限错误 - 不可重试
    if errors.Is(err, ErrUnauthorized) ||
       strings.Contains(err.Error(), "401") ||
       strings.Contains(err.Error(), "403") {
        return ErrorTypePermission
    }
    
    // 默认为未知错误
    return ErrorTypeUnknown
}

func (c *ErrorClassifier) IsRetryable(err error) bool {
    errType := c.Classify(err)
    
    switch errType {
    case ErrorTypeNetwork,
         ErrorTypeServiceUnavailable,
         ErrorTypeResourceLocked,
         ErrorTypeTimeout:
        return true
    default:
        return false
    }
}
```

### 2. 自定义错误类型

定义清晰的错误类型:

```go
// SagaError Saga 错误
type SagaError struct {
    Type       ErrorType
    Code       string
    Message    string
    Cause      error
    Retryable  bool
    StepID     string
    Timestamp  time.Time
    Context    map[string]interface{}
}

func (e *SagaError) Error() string {
    return fmt.Sprintf("[%s] %s: %s (step: %s)",
        e.Type, e.Code, e.Message, e.StepID)
}

func NewBusinessError(code, message string) *SagaError {
    return &SagaError{
        Type:      ErrorTypeBusiness,
        Code:      code,
        Message:   message,
        Retryable: false,
        Timestamp: time.Now(),
    }
}

func NewRetryableError(message string) *SagaError {
    return &SagaError{
        Type:      ErrorTypeTemporary,
        Message:   message,
        Retryable: true,
        Timestamp: time.Now(),
    }
}

// 使用示例
func (s *ValidateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    if orderData.Amount <= 0 {
        return nil, NewBusinessError("INVALID_AMOUNT", "订单金额必须大于0")
    }
    
    if !s.service.CheckInventory(ctx, orderData.Items) {
        return nil, NewBusinessError("INSUFFICIENT_INVENTORY", "库存不足")
    }
    
    return orderData, nil
}
```

### 3. 断路器模式

防止连续失败导致系统雪崩:

```go
type CircuitBreaker struct {
    maxFailures    int
    resetTimeout   time.Duration
    state          CircuitState
    failures       int
    lastFailTime   time.Time
    mu             sync.RWMutex
}

type CircuitState int

const (
    CircuitClosed CircuitState = iota
    CircuitOpen
    CircuitHalfOpen
)

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.RLock()
    state := cb.state
    cb.mu.RUnlock()
    
    switch state {
    case CircuitOpen:
        // 检查是否可以尝试恢复
        cb.mu.Lock()
        if time.Since(cb.lastFailTime) > cb.resetTimeout {
            cb.state = CircuitHalfOpen
            cb.mu.Unlock()
        } else {
            cb.mu.Unlock()
            return ErrCircuitBreakerOpen
        }
        
    case CircuitHalfOpen:
        // 半开状态,尝试调用
        break
        
    case CircuitClosed:
        // 正常状态
        break
    }
    
    // 执行调用
    err := fn()
    
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    if err != nil {
        cb.failures++
        cb.lastFailTime = time.Now()
        
        if cb.failures >= cb.maxFailures {
            cb.state = CircuitOpen
        }
        
        return err
    }
    
    // 成功,重置计数器
    cb.failures = 0
    cb.state = CircuitClosed
    return nil
}

// 使用断路器保护步骤
type ProtectedStep struct {
    step    saga.SagaStep
    breaker *CircuitBreaker
}

func (s *ProtectedStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    var result interface{}
    var err error
    
    breakerErr := s.breaker.Call(func() error {
        result, err = s.step.Execute(ctx, data)
        return err
    })
    
    if breakerErr != nil {
        return nil, breakerErr
    }
    
    return result, err
}
```

---

## 性能优化

### 1. 连接池优化

```go
// 优化 Redis 连接池
redisConfig := &redis.ClusterOptions{
    // 连接池大小
    PoolSize:     200,        // 根据并发量设置
    MinIdleConns: 50,         // 保持最小空闲连接
    
    // 超时设置
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
    PoolTimeout:  4 * time.Second,
    
    // 连接管理
    MaxConnAge:         0,                  // 连接最大生命周期
    IdleTimeout:        5 * time.Minute,    // 空闲连接超时
    IdleCheckFrequency: time.Minute,        // 空闲检查频率
    
    // 重试策略
    MaxRetries:      5,
    MinRetryBackoff: 8 * time.Millisecond,
    MaxRetryBackoff: 512 * time.Millisecond,
}
```

### 2. 批量操作

```go
// 批量保存 Saga 状态
type BatchStateStorage struct {
    storage       saga.StateStorage
    buffer        []*StateSaveRequest
    batchSize     int
    flushInterval time.Duration
    mu            sync.Mutex
}

func (s *BatchStateStorage) SaveSaga(ctx context.Context, instance saga.SagaInstance) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.buffer = append(s.buffer, &StateSaveRequest{
        Instance:  instance,
        Timestamp: time.Now(),
    })
    
    if len(s.buffer) >= s.batchSize {
        return s.flush(ctx)
    }
    
    return nil
}

func (s *BatchStateStorage) flush(ctx context.Context) error {
    if len(s.buffer) == 0 {
        return nil
    }
    
    // 批量保存
    err := s.storage.BatchSave(ctx, s.buffer)
    if err != nil {
        return err
    }
    
    s.buffer = s.buffer[:0]
    return nil
}

// 定期刷新
func (s *BatchStateStorage) StartFlushLoop(ctx context.Context) {
    ticker := time.NewTicker(s.flushInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            s.flush(ctx)
            return
        case <-ticker.C:
            s.flush(ctx)
        }
    }
}
```

### 3. 异步处理

```go
// 异步事件发布
type AsyncEventPublisher struct {
    publisher saga.EventPublisher
    buffer    chan *saga.SagaEvent
    workers   int
}

func NewAsyncEventPublisher(publisher saga.EventPublisher, bufferSize, workers int) *AsyncEventPublisher {
    aep := &AsyncEventPublisher{
        publisher: publisher,
        buffer:    make(chan *saga.SagaEvent, bufferSize),
        workers:   workers,
    }
    
    // 启动多个工作协程
    for i := 0; i < workers; i++ {
        go aep.worker()
    }
    
    return aep
}

func (p *AsyncEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
    select {
    case p.buffer <- event:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (p *AsyncEventPublisher) worker() {
    for event := range p.buffer {
        ctx := context.Background()
        
        maxRetries := 3
        for attempt := 0; attempt < maxRetries; attempt++ {
            err := p.publisher.PublishEvent(ctx, event)
            if err == nil {
                break
            }
            
            if attempt < maxRetries-1 {
                time.Sleep(time.Duration(1<<attempt) * time.Second)
            }
        }
    }
}
```

### 4. 并发控制

```go
// Saga 并发限制
type ConcurrencyLimiter struct {
    semaphore chan struct{}
    active    int64
    mu        sync.Mutex
}

func NewConcurrencyLimiter(maxConcurrent int) *ConcurrencyLimiter {
    return &ConcurrencyLimiter{
        semaphore: make(chan struct{}, maxConcurrent),
    }
}

func (cl *ConcurrencyLimiter) Acquire(ctx context.Context) error {
    select {
    case cl.semaphore <- struct{}{}:
        atomic.AddInt64(&cl.active, 1)
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (cl *ConcurrencyLimiter) Release() {
    <-cl.semaphore
    atomic.AddInt64(&cl.active, -1)
}

func (cl *ConcurrencyLimiter) GetActive() int64 {
    return atomic.LoadInt64(&cl.active)
}

// 使用限流器
func (c *OrchestratorCoordinator) StartSaga(
    ctx context.Context,
    definition saga.SagaDefinition,
    initialData interface{},
) (saga.SagaInstance, error) {
    // 获取许可
    if err := c.limiter.Acquire(ctx); err != nil {
        return nil, fmt.Errorf("failed to acquire saga slot: %w", err)
    }
    
    // 启动 Saga
    instance, err := c.startSagaInternal(ctx, definition, initialData)
    if err != nil {
        c.limiter.Release()
        return nil, err
    }
    
    // 异步释放许可
    go func() {
        // 等待 Saga 完成
        c.waitForCompletion(instance.GetID())
        c.limiter.Release()
    }()
    
    return instance, nil
}
```

---

## 安全性考虑

### 1. 敏感数据保护

```go
// 加密敏感字段
type SecureOrderData struct {
    OrderID      string
    CustomerID   string
    Amount       float64
    
    // 敏感数据加密
    CreditCard   string `json:"-"` // 不序列化
    EncryptedCC  string
    CVV          string `json:"-"`
}

type SecureStep struct {
    step      saga.SagaStep
    encryptor Encryptor
}

func (s *SecureStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    secureData := data.(*SecureOrderData)
    
    // 加密敏感数据
    if secureData.CreditCard != "" {
        encrypted, err := s.encryptor.Encrypt(secureData.CreditCard)
        if err != nil {
            return nil, err
        }
        
        secureData.EncryptedCC = encrypted
        secureData.CreditCard = ""  // 清除明文
    }
    
    // 执行步骤
    result, err := s.step.Execute(ctx, secureData)
    
    // 解密结果中的敏感数据(如需要)
    if err == nil && result != nil {
        resultData := result.(*SecureOrderData)
        if resultData.EncryptedCC != "" {
            decrypted, err := s.encryptor.Decrypt(resultData.EncryptedCC)
            if err != nil {
                return nil, err
            }
            resultData.CreditCard = decrypted
        }
    }
    
    return result, err
}

// 日志脱敏
func (o *SecureOrderData) MarshalJSON() ([]byte, error) {
    type Alias SecureOrderData
    return json.Marshal(&struct {
        CreditCard string `json:"credit_card,omitempty"`
        CVV        string `json:"cvv,omitempty"`
        *Alias
    }{
        CreditCard: maskCreditCard(o.CreditCard),
        CVV:        "***",
        Alias:      (*Alias)(o),
    })
}

func maskCreditCard(cc string) string {
    if len(cc) < 4 {
        return "****"
    }
    return "****" + cc[len(cc)-4:]
}
```

### 2. 权限控制

```go
// 基于角色的访问控制
type AuthorizedCoordinator struct {
    coordinator saga.SagaCoordinator
    authz       Authorizer
}

func (c *AuthorizedCoordinator) StartSaga(
    ctx context.Context,
    definition saga.SagaDefinition,
    initialData interface{},
) (saga.SagaInstance, error) {
    // 提取用户信息
    userID, ok := ctx.Value("user_id").(string)
    if !ok {
        return nil, errors.New("unauthorized: missing user_id")
    }
    
    // 检查权限
    allowed, err := c.authz.CanStartSaga(ctx, userID, definition.GetID())
    if err != nil {
        return nil, err
    }
    if !allowed {
        return nil, fmt.Errorf("unauthorized: user %s cannot start saga %s",
            userID, definition.GetID())
    }
    
    // 审计日志
    c.auditLog(userID, "START_SAGA", definition.GetID(), initialData)
    
    return c.coordinator.StartSaga(ctx, definition, initialData)
}

func (c *AuthorizedCoordinator) AbortSaga(
    ctx context.Context,
    sagaID string,
    reason string,
) error {
    userID, _ := ctx.Value("user_id").(string)
    
    // 检查权限
    allowed, _ := c.authz.CanAbortSaga(ctx, userID, sagaID)
    if !allowed {
        return fmt.Errorf("unauthorized: user %s cannot abort saga %s",
            userID, sagaID)
    }
    
    // 审计日志
    c.auditLog(userID, "ABORT_SAGA", sagaID, reason)
    
    return c.coordinator.AbortSaga(ctx, sagaID, reason)
}
```

### 3. 输入验证

```go
// 输入验证步骤
type ValidateInputStep struct {
    validator Validator
}

func (s *ValidateInputStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 类型断言
    orderData, ok := data.(*OrderData)
    if !ok {
        return nil, NewBusinessError("INVALID_TYPE", "Invalid data type")
    }
    
    // 必填字段验证
    if orderData.CustomerID == "" {
        return nil, NewBusinessError("MISSING_CUSTOMER_ID", "Customer ID is required")
    }
    
    if orderData.Amount <= 0 {
        return nil, NewBusinessError("INVALID_AMOUNT", "Amount must be positive")
    }
    
    if len(orderData.Items) == 0 {
        return nil, NewBusinessError("EMPTY_ITEMS", "Order must have at least one item")
    }
    
    // 格式验证
    if !isValidEmail(orderData.Email) {
        return nil, NewBusinessError("INVALID_EMAIL", "Invalid email format")
    }
    
    // 业务规则验证
    if orderData.Amount > 10000 && orderData.VerificationRequired == false {
        return nil, NewBusinessError("VERIFICATION_REQUIRED",
            "Orders over $10,000 require verification")
    }
    
    // 防止注入攻击
    if containsSQLInjection(orderData.CustomerNote) {
        return nil, NewBusinessError("INVALID_INPUT", "Invalid characters detected")
    }
    
    return orderData, nil
}
```

---

## 监控和可观测性

### 1. 关键指标

```go
var (
    // Saga 总数
    sagaTotalCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "saga_total",
            Help: "Total number of sagas",
        },
        []string{"saga_id", "saga_name"},
    )
    
    // 成功/失败计数
    sagaCompletedCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "saga_completed_total",
            Help: "Number of completed sagas",
        },
        []string{"saga_id", "status"},
    )
    
    // 执行时间
    sagaDurationHistogram = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "saga_duration_seconds",
            Help:    "Saga execution duration",
            Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
        },
        []string{"saga_id", "status"},
    )
    
    // 活跃 Saga 数
    sagaActiveGauge = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "saga_active",
            Help: "Number of active sagas",
        },
    )
    
    // 步骤执行次数
    stepExecutionCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "saga_step_executions_total",
            Help: "Number of step executions",
        },
        []string{"saga_id", "step_id", "result"},
    )
    
    // 补偿执行次数
    compensationCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "saga_compensations_total",
            Help: "Number of compensations",
        },
        []string{"saga_id", "step_id"},
    )
    
    // 重试次数
    retryCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "saga_retries_total",
            Help: "Number of retries",
        },
        []string{"saga_id", "step_id"},
    )
)
```

### 2. 分布式追踪

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// 追踪包装器
type TracedCoordinator struct {
    coordinator saga.SagaCoordinator
    tracer      trace.Tracer
}

func NewTracedCoordinator(coordinator saga.SagaCoordinator) *TracedCoordinator {
    return &TracedCoordinator{
        coordinator: coordinator,
        tracer:      otel.Tracer("saga"),
    }
}

func (c *TracedCoordinator) StartSaga(
    ctx context.Context,
    definition saga.SagaDefinition,
    initialData interface{},
) (saga.SagaInstance, error) {
    ctx, span := c.tracer.Start(ctx, "saga.start",
        trace.WithAttributes(
            attribute.String("saga.id", definition.GetID()),
            attribute.String("saga.name", definition.GetName()),
        ),
    )
    defer span.End()
    
    instance, err := c.coordinator.StartSaga(ctx, definition, initialData)
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    } else {
        span.SetAttributes(
            attribute.String("saga.instance_id", instance.GetID()),
        )
        span.SetStatus(codes.Ok, "started")
    }
    
    return instance, err
}
```

### 3. 结构化日志

```go
import "go.uber.org/zap"

type LoggedStep struct {
    step   saga.SagaStep
    logger *zap.Logger
}

func (s *LoggedStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    logger := s.logger.With(
        zap.String("step_id", s.step.GetID()),
        zap.String("step_name", s.step.GetName()),
        zap.String("saga_id", getSagaID(ctx)),
    )
    
    logger.Info("Step execution started")
    
    startTime := time.Now()
    result, err := s.step.Execute(ctx, data)
    duration := time.Since(startTime)
    
    if err != nil {
        logger.Error("Step execution failed",
            zap.Error(err),
            zap.Duration("duration", duration),
            zap.Bool("retryable", s.step.IsRetryable(err)),
        )
    } else {
        logger.Info("Step execution succeeded",
            zap.Duration("duration", duration),
        )
    }
    
    return result, err
}
```

---

## 常见陷阱和反模式

### ❌ 陷阱 1: 步骤过大

**问题**: 一个步骤做太多事情

```go
// 不好的设计
type ProcessOrderStep struct {
    orderService     OrderService
    inventoryService InventoryService
    paymentService   PaymentService
    notificationService NotificationService
}

func (s *ProcessOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 做太多事情
    order := s.orderService.CreateOrder(ctx, data)
    s.inventoryService.Reserve(ctx, order.Items)
    s.paymentService.Process(ctx, order.Amount)
    s.notificationService.Send(ctx, order.CustomerEmail)
    
    // 问题: 如果某个操作失败,如何回滚?
    return order, nil
}
```

**解决方案**: 拆分为多个独立步骤

```go
steps := []saga.SagaStep{
    &CreateOrderStep{},
    &ReserveInventoryStep{},
    &ProcessPaymentStep{},
    &SendNotificationStep{},
}
```

### ❌ 陷阱 2: 忽略幂等性

**问题**: 步骤不支持重试

```go
// 不好的设计
func (s *ReserveInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 直接执行,没有幂等性检查
    reservation, err := s.service.Reserve(ctx, data)
    // 如果重试,会创建多个预留
    return reservation, err
}
```

**解决方案**: 实现幂等性

```go
func (s *ReserveInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 检查是否已预留
    if orderData.ReservationID != "" {
        existing, _ := s.service.GetReservation(ctx, orderData.ReservationID)
        if existing != nil && existing.Status == "ACTIVE" {
            return orderData, nil
        }
    }
    
    // 执行新预留
    reservation, err := s.service.Reserve(ctx, orderData.Items)
    if err != nil {
        return nil, err
    }
    
    orderData.ReservationID = reservation.ID
    return orderData, nil
}
```

### ❌ 陷阱 3: 补偿操作抛出错误

**问题**: 补偿失败阻止其他补偿

```go
// 不好的设计
func (s *ProcessPaymentStep) Compensate(ctx context.Context, data interface{}) error {
    err := s.service.Refund(ctx, data)
    if err != nil {
        // 返回错误会阻止其他补偿
        return err
    }
    return nil
}
```

**解决方案**: 尽力而为 + 告警

```go
func (s *ProcessPaymentStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    err := s.service.Refund(ctx, orderData.TransactionID)
    if err != nil {
        // 记录错误但不阻塞
        log.Errorf("Refund failed: %v", err)
        
        // 发送告警
        s.alertService.Send(AlertCritical, fmt.Sprintf(
            "Manual refund required: %s", orderData.TransactionID))
        
        // 返回 nil 允许其他补偿继续
    }
    
    return nil
}
```

### ❌ 陷阱 4: 状态未持久化

**问题**: 服务重启后 Saga 状态丢失

```go
// 不好的设计 - 只使用内存存储
stateStorage := &InMemoryStorage{}
```

**解决方案**: 使用持久化存储

```go
// 生产环境使用 Redis 或数据库
stateStorage := NewRedisStorage(&RedisConfig{
    Addrs: []string{"redis-1:6379", "redis-2:6379"},
    // ...
})
```

### ❌ 陷阱 5: 缺少超时控制

**问题**: 步骤可能永久阻塞

```go
// 不好的设计 - 没有超时
func (s *Step) GetTimeout() time.Duration {
    return 0 // 永不超时
}
```

**解决方案**: 设置合理超时

```go
func (s *Step) GetTimeout() time.Duration {
    switch s.operationType {
    case "database":
        return 5 * time.Second
    case "external-api":
        return 30 * time.Second
    case "file-processing":
        return 5 * time.Minute
    default:
        return 10 * time.Second
    }
}
```

### ❌ 陷阱 6: 未区分错误类型

**问题**: 所有错误都重试或都不重试

```go
// 不好的设计
func (s *Step) IsRetryable(err error) bool {
    return true // 盲目重试所有错误
}
```

**解决方案**: 正确分类错误

```go
func (s *Step) IsRetryable(err error) bool {
    // 临时错误可重试
    if errors.Is(err, ErrTimeout) ||
       errors.Is(err, ErrServiceUnavailable) ||
       errors.Is(err, ErrNetworkError) {
        return true
    }
    
    // 业务错误不重试
    if errors.Is(err, ErrInvalidInput) ||
       errors.Is(err, ErrBusinessRule) {
        return false
    }
    
    return false
}
```

### ❌ 陷阱 7: 忽略监控

**问题**: 无法了解 Saga 运行状况

**解决方案**: 添加完整的监控

```go
// 记录指标
metrics.SagaTotal.WithLabelValues(sagaID, sagaName).Inc()
metrics.SagaActive.Inc()

// 记录日志
logger.Info("Saga started",
    zap.String("saga_id", sagaID),
    zap.String("saga_name", sagaName))

// 分布式追踪
ctx, span := tracer.Start(ctx, "saga.execute")
defer span.End()
```

---

## Do's and Don'ts 清单

### ✅ Do's (应该做的)

#### 设计

- ✅ 保持步骤单一职责
- ✅ 确保步骤和补偿的幂等性
- ✅ 为每个步骤设置合理超时
- ✅ 使用强类型数据结构
- ✅ 实现向后兼容的数据结构

#### 错误处理

- ✅ 正确分类错误类型
- ✅ 区分可重试和不可重试错误
- ✅ 使用自定义错误类型
- ✅ 记录详细的错误信息
- ✅ 实现断路器模式

#### 补偿

- ✅ 使用语义补偿而不是精确撤销
- ✅ 补偿操作尽力而为
- ✅ 补偿失败发送告警
- ✅ 记录补偿到死信队列
- ✅ 测试补偿逻辑

#### 性能

- ✅ 使用连接池
- ✅ 实现批量操作
- ✅ 异步处理非关键操作
- ✅ 并发控制和限流
- ✅ 缓存常用数据

#### 安全

- ✅ 加密敏感数据
- ✅ 实施权限控制
- ✅ 验证所有输入
- ✅ 日志脱敏
- ✅ 审计关键操作

#### 监控

- ✅ 记录关键指标
- ✅ 实现分布式追踪
- ✅ 使用结构化日志
- ✅ 配置告警规则
- ✅ 定期review监控数据

#### 测试

- ✅ 单元测试每个步骤
- ✅ 测试补偿逻辑
- ✅ 测试并发场景
- ✅ 测试故障恢复
- ✅ 性能测试

### ❌ Don'ts (不应该做的)

#### 设计

- ❌ 不要让步骤做多件事
- ❌ 不要忽略幂等性
- ❌ 不要硬编码配置
- ❌ 不要使用 `map[string]interface{}`
- ❌ 不要忽略向后兼容性

#### 错误处理

- ❌ 不要盲目重试所有错误
- ❌ 不要忽略错误分类
- ❌ 不要吞掉错误信息
- ❌ 不要无限重试
- ❌ 不要忽略超时

#### 补偿

- ❌ 不要让补偿失败阻塞其他补偿
- ❌ 不要忘记测试补偿逻辑
- ❌ 不要使用精确撤销(除非必要)
- ❌ 不要忽略补偿失败
- ❌ 不要忘记清理资源

#### 性能

- ❌ 不要在步骤中阻塞等待
- ❌ 不要为每个操作创建新连接
- ❌ 不要同步执行所有步骤
- ❌ 不要忽略并发控制
- ❌ 不要过度使用全局锁

#### 安全

- ❌ 不要在日志中记录敏感数据
- ❌ 不要忽略输入验证
- ❌ 不要明文存储密码
- ❌ 不要忽略权限检查
- ❌ 不要跳过审计日志

#### 生产环境

- ❌ 不要在生产使用内存存储
- ❌ 不要忽略监控和告警
- ❌ 不要在生产调试日志过多
- ❌ 不要忽略健康检查
- ❌ 不要忘记故障恢复测试

---

## 总结

遵循这些最佳实践可以帮助您:

1. **设计健壮的 Saga**: 使用正确的模式和原则
2. **处理复杂场景**: 应对各种故障和边界情况
3. **优化性能**: 提高吞吐量和降低延迟
4. **保证安全**: 保护敏感数据和实施访问控制
5. **可观测性**: 全面监控系统运行状况
6. **避免陷阱**: 识别和避免常见错误

## 进一步学习

- 阅读 [教程文档](tutorials.md)
- 查看 [API 参考](../saga-api-reference.md)
- 探索 [示例代码](../../examples/saga-orchestrator/)
- 参考 [用户指南](../saga-user-guide.md)
- 学习 [开发者文档](../saga-developer-guide.md)

## 参考资源

- [Saga Pattern - Martin Fowler](https://martinfowler.com/articles/patterns-of-distributed-systems/saga.html)
- [Microservices Patterns - Saga](https://microservices.io/patterns/data/saga.html)
- [Distributed Sagas: A Protocol for Coordinating Microservices](https://www.cs.cornell.edu/andru/cs711/2002fa/reading/sagas.pdf)
- [Event Sourcing and CQRS](https://martinfowler.com/eaaDev/EventSourcing.html)


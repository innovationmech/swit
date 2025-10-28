# Saga 分布式事务教程

本教程通过循序渐进的方式,帮助您从零开始掌握 Swit 框架的 Saga 分布式事务系统。

## 目录

- [教程 1: Hello Saga - 第一个分布式事务](#教程-1-hello-saga---第一个分布式事务)
- [教程 2: 理解步骤和补偿](#教程-2-理解步骤和补偿)
- [教程 3: 处理错误和重试](#教程-3-处理错误和重试)
- [教程 4: 状态持久化和恢复](#教程-4-状态持久化和恢复)
- [教程 5: 使用 DSL 定义 Saga](#教程-5-使用-dsl-定义-saga)
- [教程 6: 复杂业务流程编排](#教程-6-复杂业务流程编排)
- [教程 7: 监控和可观测性](#教程-7-监控和可观测性)
- [教程 8: 生产环境最佳实践](#教程-8-生产环境最佳实践)

---

## 教程 1: Hello Saga - 第一个分布式事务

### 学习目标

- 理解 Saga 的基本概念
- 创建第一个 Saga 定义
- 实现简单的步骤
- 运行基础的 Saga 流程

### 场景描述

我们将构建一个简单的用户注册流程,包含以下步骤:
1. 创建用户账户
2. 发送欢迎邮件

### 步骤 1: 安装依赖

```bash
go get github.com/innovationmech/swit
```

### 步骤 2: 定义数据结构

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/innovationmech/swit/pkg/saga"
    "github.com/innovationmech/swit/pkg/saga/coordinator"
)

// UserRegistrationData 用户注册数据
type UserRegistrationData struct {
    Username string
    Email    string
    Password string
    UserID   string // 创建成功后填充
}
```

### 步骤 3: 实现第一个步骤 - 创建用户

```go
// CreateUserStep 创建用户步骤
type CreateUserStep struct{}

func (s *CreateUserStep) GetID() string          { return "create-user" }
func (s *CreateUserStep) GetName() string        { return "创建用户" }
func (s *CreateUserStep) GetDescription() string { return "在数据库中创建新用户" }
func (s *CreateUserStep) GetTimeout() time.Duration { return 5 * time.Second }
func (s *CreateUserStep) GetRetryPolicy() saga.RetryPolicy { return nil }
func (s *CreateUserStep) IsRetryable(err error) bool { return true }
func (s *CreateUserStep) GetMetadata() map[string]interface{} { return nil }

// Execute 执行步骤
func (s *CreateUserStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    userData := data.(*UserRegistrationData)
    
    fmt.Printf("正在创建用户: %s\n", userData.Username)
    
    // 模拟数据库操作
    time.Sleep(500 * time.Millisecond)
    
    // 生成用户 ID
    userData.UserID = fmt.Sprintf("USER-%d", time.Now().Unix())
    
    fmt.Printf("用户创建成功: %s (ID: %s)\n", userData.Username, userData.UserID)
    return userData, nil
}

// Compensate 补偿操作 - 删除用户
func (s *CreateUserStep) Compensate(ctx context.Context, data interface{}) error {
    userData := data.(*UserRegistrationData)
    
    fmt.Printf("回滚: 删除用户 %s (ID: %s)\n", userData.Username, userData.UserID)
    
    // 模拟删除操作
    time.Sleep(200 * time.Millisecond)
    
    fmt.Printf("用户已删除: %s\n", userData.UserID)
    return nil
}
```

### 步骤 4: 实现第二个步骤 - 发送邮件

```go
// SendWelcomeEmailStep 发送欢迎邮件步骤
type SendWelcomeEmailStep struct{}

func (s *SendWelcomeEmailStep) GetID() string          { return "send-welcome-email" }
func (s *SendWelcomeEmailStep) GetName() string        { return "发送欢迎邮件" }
func (s *SendWelcomeEmailStep) GetDescription() string { return "向新用户发送欢迎邮件" }
func (s *SendWelcomeEmailStep) GetTimeout() time.Duration { return 10 * time.Second }
func (s *SendWelcomeEmailStep) GetRetryPolicy() saga.RetryPolicy { return nil }
func (s *SendWelcomeEmailStep) IsRetryable(err error) bool { return true }
func (s *SendWelcomeEmailStep) GetMetadata() map[string]interface{} { return nil }

// Execute 发送邮件
func (s *SendWelcomeEmailStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    userData := data.(*UserRegistrationData)
    
    fmt.Printf("正在发送欢迎邮件到: %s\n", userData.Email)
    
    // 模拟邮件发送
    time.Sleep(1 * time.Second)
    
    fmt.Printf("欢迎邮件已发送: %s\n", userData.Email)
    return userData, nil
}

// Compensate 补偿操作 - 发送取消邮件
func (s *SendWelcomeEmailStep) Compensate(ctx context.Context, data interface{}) error {
    userData := data.(*UserRegistrationData)
    
    fmt.Printf("回滚: 发送注册取消通知到: %s\n", userData.Email)
    
    // 模拟发送取消邮件
    time.Sleep(500 * time.Millisecond)
    
    fmt.Printf("取消通知已发送: %s\n", userData.Email)
    return nil
}
```

### 步骤 5: 定义 Saga

```go
// UserRegistrationSaga 用户注册 Saga 定义
type UserRegistrationSaga struct {
    id string
}

func NewUserRegistrationSaga() *UserRegistrationSaga {
    return &UserRegistrationSaga{
        id: "user-registration-saga",
    }
}

func (d *UserRegistrationSaga) GetID() string          { return d.id }
func (d *UserRegistrationSaga) GetName() string        { return "用户注册 Saga" }
func (d *UserRegistrationSaga) GetDescription() string { return "处理用户注册流程" }
func (d *UserRegistrationSaga) GetTimeout() time.Duration { return 2 * time.Minute }

func (d *UserRegistrationSaga) GetRetryPolicy() saga.RetryPolicy {
    return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

func (d *UserRegistrationSaga) GetCompensationStrategy() saga.CompensationStrategy {
    return saga.NewSequentialCompensationStrategy(30 * time.Second)
}

func (d *UserRegistrationSaga) GetMetadata() map[string]interface{} {
    return map[string]interface{}{
        "domain":  "user-management",
        "version": "v1",
    }
}

func (d *UserRegistrationSaga) GetSteps() []saga.SagaStep {
    return []saga.SagaStep{
        &CreateUserStep{},
        &SendWelcomeEmailStep{},
    }
}

func (d *UserRegistrationSaga) Validate() error {
    if d.id == "" {
        return fmt.Errorf("saga ID is required")
    }
    return nil
}
```

### 步骤 6: 运行 Saga

```go
func main() {
    // 创建简单的内存存储和事件发布器
    stateStorage := newInMemoryStateStorage()
    eventPublisher := newInMemoryEventPublisher()
    
    // 配置 Coordinator
    config := &coordinator.OrchestratorConfig{
        StateStorage:   stateStorage,
        EventPublisher: eventPublisher,
        ConcurrencyConfig: &coordinator.ConcurrencyConfig{
            MaxConcurrentSagas: 10,
            WorkerPoolSize:     5,
        },
    }
    
    // 创建 Coordinator
    sagaCoordinator, err := coordinator.NewOrchestratorCoordinator(config)
    if err != nil {
        panic(err)
    }
    defer sagaCoordinator.Close()
    
    // 准备用户数据
    userData := &UserRegistrationData{
        Username: "alice",
        Email:    "alice@example.com",
        Password: "hashed_password_here",
    }
    
    // 启动 Saga
    ctx := context.Background()
    definition := NewUserRegistrationSaga()
    instance, err := sagaCoordinator.StartSaga(ctx, definition, userData)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Saga 已启动: ID=%s\n", instance.GetID())
    
    // 等待完成
    time.Sleep(5 * time.Second)
    
    // 获取最终状态
    finalInstance, _ := sagaCoordinator.GetSagaInstance(instance.GetID())
    fmt.Printf("Saga 最终状态: %s\n", finalInstance.GetState().String())
}
```

### 运行结果

成功的情况:
```
Saga 已启动: ID=saga-xxx
正在创建用户: alice
用户创建成功: alice (ID: USER-1234567890)
正在发送欢迎邮件到: alice@example.com
欢迎邮件已发送: alice@example.com
Saga 最终状态: COMPLETED
```

### 💡 关键要点

1. **步骤接口**: 每个步骤必须实现 `SagaStep` 接口
2. **补偿操作**: 每个步骤都应该有对应的补偿逻辑
3. **数据流转**: 步骤之间通过返回值传递数据
4. **Coordinator**: 负责协调整个 Saga 的执行

---

## 教程 2: 理解步骤和补偿

### 学习目标

- 深入理解步骤的生命周期
- 掌握补偿操作的设计原则
- 学习幂等性实现

### 场景描述

构建一个电商订单处理流程,演示补偿操作如何工作。

### 步骤设计原则

#### 1. 幂等性

步骤应该支持多次执行而不产生副作用:

```go
// ✅ 好的设计: 支持幂等
func (s *ReserveInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 检查是否已经预留
    if orderData.ReservationID != "" {
        existing, err := s.service.GetReservation(ctx, orderData.ReservationID)
        if err == nil && existing != nil {
            // 已经预留,直接返回
            return orderData, nil
        }
    }
    
    // 执行预留操作
    reservation, err := s.service.Reserve(ctx, orderData.Items)
    if err != nil {
        return nil, err
    }
    
    orderData.ReservationID = reservation.ID
    return orderData, nil
}
```

#### 2. 原子性

每个步骤应该是一个原子操作:

```go
// ✅ 好的设计: 原子操作
func (s *ProcessPaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 在一个事务中完成所有支付相关操作
    return s.paymentService.ProcessPaymentInTransaction(ctx, paymentData)
}

// ❌ 不好的设计: 多个非原子操作
func (s *ProcessPaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 操作 1
    s.service.AuthorizePayment(ctx, data)
    
    // 如果这里失败,前面的操作无法回滚
    s.service.CapturePayment(ctx, data)
    
    // 操作 3
    s.service.RecordTransaction(ctx, data)
    
    return data, nil
}
```

#### 3. 完整的补偿

补偿操作应该能够完全撤销步骤的效果:

```go
// CreateOrderStep 创建订单
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    order, err := s.service.CreateOrder(ctx, orderData)
    if err != nil {
        return nil, err
    }
    
    // 保存订单 ID,补偿时需要
    orderData.OrderID = order.ID
    orderData.OrderStatus = "CREATED"
    
    return orderData, nil
}

// Compensate 完整补偿 - 标记订单为取消
func (s *CreateOrderStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    if orderData.OrderID == "" {
        // 没有创建订单,无需补偿
        return nil
    }
    
    // 取消订单
    err := s.service.CancelOrder(ctx, orderData.OrderID, "saga_compensation")
    if err != nil {
        // 记录错误但不阻止其他补偿
        fmt.Printf("警告: 补偿失败 - %v\n", err)
    }
    
    orderData.OrderStatus = "CANCELLED"
    return nil
}
```

### 补偿失败处理

补偿操作可能失败,需要妥善处理:

```go
// ProcessPaymentStep 支付步骤
type ProcessPaymentStep struct {
    service       PaymentService
    alertService  AlertService
    maxRetries    int
}

func (s *ProcessPaymentStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    if orderData.TransactionID == "" {
        return nil // 没有支付记录,无需补偿
    }
    
    // 尝试退款
    for attempt := 1; attempt <= s.maxRetries; attempt++ {
        err := s.service.Refund(ctx, orderData.TransactionID)
        if err == nil {
            return nil // 退款成功
        }
        
        fmt.Printf("退款失败 (尝试 %d/%d): %v\n", attempt, s.maxRetries, err)
        
        if attempt < s.maxRetries {
            // 等待后重试
            time.Sleep(time.Duration(attempt) * time.Second)
        }
    }
    
    // 所有重试失败,发送告警
    s.alertService.Send(AlertCritical, fmt.Sprintf(
        "需要手动退款: TransactionID=%s, OrderID=%s",
        orderData.TransactionID, orderData.OrderID,
    ))
    
    // 返回 nil 允许其他步骤继续补偿
    // 如果返回错误,会阻止补偿流程
    return nil
}
```

### 补偿顺序

Saga 默认按照执行步骤的逆序进行补偿:

```
执行顺序: Step1 → Step2 → Step3 → Step4 → [失败]
补偿顺序: Step3 ← Step2 ← Step1
```

```go
// 示例: 订单处理 Saga
步骤执行:
1. CreateOrder      → 成功
2. ReserveInventory → 成功
3. ProcessPayment   → 成功
4. ConfirmOrder     → 失败!

补偿执行:
3. ProcessPayment.Compensate()   → 退款
2. ReserveInventory.Compensate() → 释放库存
1. CreateOrder.Compensate()      → 取消订单
```

### 💡 关键要点

1. **幂等性**: 步骤可以安全地重复执行
2. **原子性**: 每个步骤是一个独立的原子操作
3. **可补偿性**: 每个步骤都有清晰的补偿逻辑
4. **错误处理**: 补偿失败不应阻止其他步骤的补偿
5. **数据追踪**: 保存必要的 ID 用于补偿操作

---

## 教程 3: 处理错误和重试

### 学习目标

- 理解不同类型的错误
- 配置重试策略
- 实现错误分类

### 错误类型

Saga 系统中有几种错误类型:

#### 1. 临时错误 (Temporary Errors)

可以通过重试解决的错误:
- 网络超时
- 服务暂时不可用
- 资源暂时锁定
- 数据库连接失败

```go
func (s *PaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    result, err := s.service.ProcessPayment(ctx, data)
    if err != nil {
        // 判断是否是临时错误
        if isTemporaryError(err) {
            // 返回可重试错误
            return nil, saga.NewRetryableError(err.Error())
        }
        // 其他错误不重试
        return nil, err
    }
    return result, nil
}

func isTemporaryError(err error) bool {
    // 检查错误类型
    if errors.Is(err, ErrServiceUnavailable) {
        return true
    }
    if errors.Is(err, ErrTimeout) {
        return true
    }
    if strings.Contains(err.Error(), "connection refused") {
        return true
    }
    return false
}
```

#### 2. 永久错误 (Permanent Errors)

无法通过重试解决的错误:
- 验证错误
- 业务规则违反
- 权限错误
- 资源不存在

```go
func (s *ValidateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 验证订单金额
    if orderData.Amount <= 0 {
        // 业务错误,不应重试
        return nil, saga.NewBusinessError("INVALID_AMOUNT", "订单金额必须大于0")
    }
    
    // 验证库存
    if !s.service.CheckInventory(ctx, orderData.Items) {
        // 业务错误,不应重试
        return nil, saga.NewBusinessError("INSUFFICIENT_INVENTORY", "库存不足")
    }
    
    return orderData, nil
}
```

### 重试策略

#### 1. 固定延迟重试

每次重试间隔固定时间:

```go
func (d *OrderSagaDefinition) GetRetryPolicy() saga.RetryPolicy {
    // 固定延迟: 每次等待 2 秒,最多重试 3 次
    return saga.NewFixedDelayRetryPolicy(3, 2*time.Second)
}

// 重试时间线:
// 尝试 1: 立即执行 → 失败
// 等待 2 秒
// 尝试 2: 2秒后执行 → 失败
// 等待 2 秒
// 尝试 3: 4秒后执行 → 失败
// 等待 2 秒
// 尝试 4: 6秒后执行 → 最后一次尝试
```

#### 2. 指数退避重试

重试间隔呈指数增长:

```go
func (d *OrderSagaDefinition) GetRetryPolicy() saga.RetryPolicy {
    // 指数退避:
    // - 最多重试 5 次
    // - 初始延迟 1 秒
    // - 最大延迟 30 秒
    // - 每次延迟翻倍
    return saga.NewExponentialBackoffRetryPolicy(5, time.Second, 30*time.Second)
}

// 重试时间线:
// 尝试 1: 立即执行 → 失败
// 等待 1 秒
// 尝试 2: 1秒后执行 → 失败
// 等待 2 秒
// 尝试 3: 3秒后执行 → 失败
// 等待 4 秒
// 尝试 4: 7秒后执行 → 失败
// 等待 8 秒
// 尝试 5: 15秒后执行 → 失败
// 等待 16 秒(但被限制为30秒)
// 尝试 6: 45秒后执行 → 最后一次尝试
```

添加抖动(Jitter)避免"惊群效应":

```go
func (d *OrderSagaDefinition) GetRetryPolicy() saga.RetryPolicy {
    policy := saga.NewExponentialBackoffRetryPolicy(5, time.Second, 30*time.Second)
    policy.SetJitter(true) // 在延迟时间上添加随机抖动
    policy.SetMultiplier(2.0) // 设置倍增因子
    return policy
}
```

#### 3. 线性退避重试

重试间隔线性增长:

```go
func (d *OrderSagaDefinition) GetRetryPolicy() saga.RetryPolicy {
    // 线性退避:
    // - 最多重试 5 次
    // - 基础延迟 1 秒
    // - 每次增加 2 秒
    // - 最大延迟 30 秒
    return saga.NewLinearBackoffRetryPolicy(5, time.Second, 2*time.Second, 30*time.Second)
}

// 重试时间线:
// 尝试 1: 立即执行 → 失败
// 等待 1 秒 (基础)
// 尝试 2: 1秒后执行 → 失败
// 等待 3 秒 (基础 + 1*增量)
// 尝试 3: 4秒后执行 → 失败
// 等待 5 秒 (基础 + 2*增量)
// 尝试 4: 9秒后执行 → 失败
// 等待 7 秒 (基础 + 3*增量)
// 尝试 5: 16秒后执行 → 失败
```

#### 4. 不重试

某些步骤不应该重试:

```go
func (s *SendNotificationStep) GetRetryPolicy() saga.RetryPolicy {
    // 通知步骤失败不重试,避免重复通知
    return saga.NewNoRetryPolicy()
}
```

### 步骤级别重试策略

可以为特定步骤覆盖全局重试策略:

```go
// PaymentStep 支付步骤
type PaymentStep struct {
    service PaymentService
}

func (s *PaymentStep) GetRetryPolicy() saga.RetryPolicy {
    // 支付步骤使用更激进的重试策略
    return saga.NewExponentialBackoffRetryPolicy(
        7,                // 最多重试 7 次
        500*time.Millisecond, // 初始延迟 500ms
        1*time.Minute,    // 最大延迟 1 分钟
    )
}

func (s *PaymentStep) IsRetryable(err error) bool {
    // 自定义重试判断逻辑
    if errors.Is(err, ErrPaymentGatewayTimeout) {
        return true
    }
    if errors.Is(err, ErrPaymentGatewayUnavailable) {
        return true
    }
    // 支付被拒绝不重试
    if errors.Is(err, ErrPaymentDeclined) {
        return false
    }
    return true
}
```

### 实战示例: 健壮的订单步骤

```go
type CreateOrderStep struct {
    service     OrderService
    maxRetries  int
    retryPolicy saga.RetryPolicy
}

func NewCreateOrderStep(service OrderService) *CreateOrderStep {
    return &CreateOrderStep{
        service:    service,
        maxRetries: 5,
        retryPolicy: saga.NewExponentialBackoffRetryPolicy(
            5,
            time.Second,
            30*time.Second,
        ),
    }
}

func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 幂等性检查
    if orderData.OrderID != "" {
        existing, err := s.service.GetOrder(ctx, orderData.OrderID)
        if err == nil && existing != nil {
            return orderData, nil
        }
    }
    
    // 创建订单
    order, err := s.service.CreateOrder(ctx, &CreateOrderRequest{
        CustomerID: orderData.CustomerID,
        Items:      orderData.Items,
        Amount:     orderData.Amount,
    })
    
    if err != nil {
        // 分类错误
        if s.isRetryable(err) {
            return nil, saga.NewRetryableError(err.Error())
        }
        return nil, saga.NewBusinessError("CREATE_ORDER_FAILED", err.Error())
    }
    
    orderData.OrderID = order.ID
    orderData.CreatedAt = time.Now()
    
    return orderData, nil
}

func (s *CreateOrderStep) isRetryable(err error) bool {
    // 数据库连接错误 - 可重试
    if errors.Is(err, ErrDatabaseConnectionFailed) {
        return true
    }
    
    // 超时错误 - 可重试
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }
    
    // 唯一键冲突 - 不可重试
    if errors.Is(err, ErrDuplicateOrder) {
        return false
    }
    
    // 验证错误 - 不可重试
    if errors.Is(err, ErrInvalidOrderData) {
        return false
    }
    
    // 默认可重试
    return true
}

func (s *CreateOrderStep) GetRetryPolicy() saga.RetryPolicy {
    return s.retryPolicy
}

func (s *CreateOrderStep) IsRetryable(err error) bool {
    return s.isRetryable(err)
}
```

### 💡 关键要点

1. **错误分类**: 区分临时错误和永久错误
2. **合理重试**: 选择适合业务场景的重试策略
3. **避免过度重试**: 设置合理的最大重试次数
4. **幂等性**: 确保步骤可以安全重试
5. **监控告警**: 记录重试次数和失败原因

---

## 教程 4: 状态持久化和恢复

### 学习目标

- 理解状态持久化的重要性
- 实现 Redis 状态存储
- 处理故障恢复

### 为什么需要状态持久化?

当系统崩溃或重启时,内存中的 Saga 状态会丢失。状态持久化确保:
- Saga 可以在失败后恢复
- 长时间运行的 Saga 不受服务重启影响
- 可以追踪和审计 Saga 执行历史

### 实现 Redis 状态存储

```go
package storage

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/innovationmech/swit/pkg/saga"
)

// RedisStateStorage Redis 状态存储实现
type RedisStateStorage struct {
    client      *redis.Client
    keyPrefix   string
    defaultTTL  time.Duration
}

// NewRedisStateStorage 创建 Redis 状态存储
func NewRedisStateStorage(addr, password string, db int) (*RedisStateStorage, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       db,
    })
    
    // 测试连接
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("redis connection failed: %w", err)
    }
    
    return &RedisStateStorage{
        client:     client,
        keyPrefix:  "saga:",
        defaultTTL: 24 * time.Hour,
    }, nil
}

// SaveSaga 保存 Saga 实例
func (s *RedisStateStorage) SaveSaga(ctx context.Context, instance saga.SagaInstance) error {
    key := s.sagaKey(instance.GetID())
    
    // 序列化 Saga 状态
    data, err := json.Marshal(instance)
    if err != nil {
        return fmt.Errorf("failed to marshal saga: %w", err)
    }
    
    // 保存到 Redis
    if err := s.client.Set(ctx, key, data, s.defaultTTL).Err(); err != nil {
        return fmt.Errorf("failed to save saga to redis: %w", err)
    }
    
    // 添加到活跃 Saga 集合
    if !instance.IsTerminal() {
        if err := s.client.SAdd(ctx, s.activeSagasKey(), instance.GetID()).Err(); err != nil {
            return fmt.Errorf("failed to add to active sagas: %w", err)
        }
    }
    
    return nil
}

// GetSaga 获取 Saga 实例
func (s *RedisStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
    key := s.sagaKey(sagaID)
    
    data, err := s.client.Get(ctx, key).Bytes()
    if err == redis.Nil {
        return nil, saga.ErrSagaNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get saga from redis: %w", err)
    }
    
    // 反序列化
    var instance saga.SagaInstance
    if err := json.Unmarshal(data, &instance); err != nil {
        return nil, fmt.Errorf("failed to unmarshal saga: %w", err)
    }
    
    return instance, nil
}

// UpdateSagaState 更新 Saga 状态
func (s *RedisStateStorage) UpdateSagaState(
    ctx context.Context,
    sagaID string,
    state saga.SagaState,
    metadata map[string]interface{},
) error {
    // 获取当前实例
    instance, err := s.GetSaga(ctx, sagaID)
    if err != nil {
        return err
    }
    
    // 更新状态
    instance.SetState(state)
    if metadata != nil {
        for k, v := range metadata {
            instance.SetMetadata(k, v)
        }
    }
    
    // 保存更新
    if err := s.SaveSaga(ctx, instance); err != nil {
        return err
    }
    
    // 如果状态变为终态,从活跃集合中移除
    if state.IsTerminal() {
        s.client.SRem(ctx, s.activeSagasKey(), sagaID)
    }
    
    return nil
}

// GetActiveSagas 获取活跃的 Saga 列表
func (s *RedisStateStorage) GetActiveSagas(
    ctx context.Context,
    filter *saga.SagaFilter,
) ([]saga.SagaInstance, error) {
    // 获取所有活跃 Saga ID
    sagaIDs, err := s.client.SMembers(ctx, s.activeSagasKey()).Result()
    if err != nil {
        return nil, fmt.Errorf("failed to get active sagas: %w", err)
    }
    
    // 批量获取 Saga 实例
    instances := make([]saga.SagaInstance, 0, len(sagaIDs))
    for _, id := range sagaIDs {
        instance, err := s.GetSaga(ctx, id)
        if err != nil {
            continue // 跳过错误的实例
        }
        
        // 应用过滤器
        if filter != nil && !filter.Match(instance) {
            continue
        }
        
        instances = append(instances, instance)
    }
    
    return instances, nil
}

// 辅助方法
func (s *RedisStateStorage) sagaKey(sagaID string) string {
    return fmt.Sprintf("%ssaga:%s", s.keyPrefix, sagaID)
}

func (s *RedisStateStorage) activeSagasKey() string {
    return fmt.Sprintf("%sactive", s.keyPrefix)
}
```

### 使用 Redis 存储

```go
func main() {
    // 创建 Redis 状态存储
    stateStorage, err := storage.NewRedisStateStorage(
        "localhost:6379",  // Redis 地址
        "",                // 密码
        0,                 // 数据库
    )
    if err != nil {
        panic(err)
    }
    
    // 创建事件发布器
    eventPublisher := newKafkaEventPublisher()
    
    // 配置 Coordinator
    config := &coordinator.OrchestratorConfig{
        StateStorage:   stateStorage,
        EventPublisher: eventPublisher,
        ConcurrencyConfig: &coordinator.ConcurrencyConfig{
            MaxConcurrentSagas: 100,
            WorkerPoolSize:     20,
        },
    }
    
    // 创建 Coordinator
    sagaCoordinator, err := coordinator.NewOrchestratorCoordinator(config)
    if err != nil {
        panic(err)
    }
    defer sagaCoordinator.Close()
    
    // 启动 Saga
    ctx := context.Background()
    definition := NewOrderProcessingSaga()
    orderData := &OrderData{
        CustomerID: "CUST-001",
        Amount:     99.99,
    }
    
    instance, err := sagaCoordinator.StartSaga(ctx, definition, orderData)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Saga 已启动: ID=%s\n", instance.GetID())
    fmt.Println("即使服务重启,Saga 状态也会保留在 Redis 中")
}
```

### 故障恢复

当服务重启后,需要恢复未完成的 Saga:

```go
// RecoverUnfinishedSagas 恢复未完成的 Saga
func RecoverUnfinishedSagas(coordinator saga.SagaCoordinator) error {
    ctx := context.Background()
    
    // 获取所有活跃的 Saga
    filter := &saga.SagaFilter{
        States: []saga.SagaState{
            saga.StateRunning,
            saga.StateCompensating,
        },
    }
    
    instances, err := coordinator.GetActiveSagas(filter)
    if err != nil {
        return fmt.Errorf("failed to get active sagas: %w", err)
    }
    
    fmt.Printf("发现 %d 个未完成的 Saga,正在恢复...\n", len(instances))
    
    for _, instance := range instances {
        fmt.Printf("恢复 Saga: ID=%s, State=%s\n",
            instance.GetID(), instance.GetState().String())
        
        // 恢复 Saga 执行
        if err := coordinator.ResumeSaga(ctx, instance.GetID()); err != nil {
            fmt.Printf("恢复 Saga 失败: %v\n", err)
            continue
        }
        
        fmt.Printf("Saga %s 已恢复\n", instance.GetID())
    }
    
    return nil
}

func main() {
    // 创建 Coordinator (使用持久化存储)
    sagaCoordinator := createCoordinator()
    defer sagaCoordinator.Close()
    
    // 服务启动时恢复未完成的 Saga
    if err := RecoverUnfinishedSagas(sagaCoordinator); err != nil {
        panic(err)
    }
    
    // 继续正常服务...
}
```

### 超时检测和处理

检测并处理超时的 Saga:

```go
// MonitorTimeoutSagas 监控超时的 Saga
func MonitorTimeoutSagas(coordinator saga.SagaCoordinator, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for range ticker.C {
        ctx := context.Background()
        
        // 获取所有运行中的 Saga
        filter := &saga.SagaFilter{
            States: []saga.SagaState{saga.StateRunning},
        }
        
        instances, err := coordinator.GetActiveSagas(filter)
        if err != nil {
            fmt.Printf("获取活跃 Saga 失败: %v\n", err)
            continue
        }
        
        now := time.Now()
        for _, instance := range instances {
            // 检查是否超时
            timeout := instance.GetDefinition().GetTimeout()
            startTime := instance.GetStartTime()
            
            if now.Sub(startTime) > timeout {
                fmt.Printf("检测到超时 Saga: ID=%s, 运行时间=%v\n",
                    instance.GetID(), now.Sub(startTime))
                
                // 标记为超时并触发补偿
                err := coordinator.AbortSaga(ctx, instance.GetID(), "timeout")
                if err != nil {
                    fmt.Printf("中止 Saga 失败: %v\n", err)
                }
            }
        }
    }
}

func main() {
    coordinator := createCoordinator()
    defer coordinator.Close()
    
    // 启动超时监控 (每分钟检查一次)
    go MonitorTimeoutSagas(coordinator, time.Minute)
    
    // 继续正常服务...
}
```

### 💡 关键要点

1. **状态持久化**: 使用 Redis 或数据库持久化 Saga 状态
2. **故障恢复**: 服务重启后恢复未完成的 Saga
3. **超时检测**: 定期检查并处理超时的 Saga
4. **数据一致性**: 确保状态更新的原子性
5. **性能优化**: 批量操作和合理的 TTL 设置

---

## 教程 5: 使用 DSL 定义 Saga

### 学习目标

- 学习 Saga DSL 语法
- 用 YAML 定义 Saga 流程
- 理解 DSL 的优势

### 为什么使用 DSL?

Saga DSL (Domain-Specific Language) 的优势:
- ✅ **声明式**: 更容易理解和维护
- ✅ **无需编译**: 修改流程无需重新部署代码
- ✅ **可视化**: 便于生成流程图
- ✅ **版本控制**: YAML 文件易于版本管理
- ✅ **业务友好**: 非开发人员也能理解

### 基础 DSL 示例

创建文件 `user-registration.saga.yaml`:

```yaml
# 用户注册 Saga
saga:
  id: user-registration-saga
  name: User Registration Saga
  description: Handle user registration process
  version: "1.0.0"
  timeout: 5m
  mode: orchestration

# 全局重试策略
global_retry_policy:
  type: exponential_backoff
  max_attempts: 3
  initial_delay: 1s
  max_delay: 30s
  multiplier: 2.0
  jitter: true

# 全局补偿策略
global_compensation:
  strategy: sequential
  timeout: 2m

# 步骤定义
steps:
  # 步骤 1: 验证用户数据
  - id: validate-user
    name: Validate User Data
    description: Validate username, email, and password
    type: service
    action:
      service:
        name: user-service
        endpoint: http://user-service:8080
        method: POST
        path: /api/users/validate
        headers:
          Content-Type: application/json
        body:
          username: "{{.input.username}}"
          email: "{{.input.email}}"
          password: "{{.input.password}}"
    timeout: 10s
    compensation:
      type: skip  # 只读操作,无需补偿

  # 步骤 2: 创建用户账户
  - id: create-user
    name: Create User Account
    description: Create user account in database
    type: service
    action:
      service:
        name: user-service
        endpoint: http://user-service:8080
        method: POST
        path: /api/users
        body:
          username: "{{.input.username}}"
          email: "{{.input.email}}"
          password_hash: "{{.output.validate-user.password_hash}}"
    compensation:
      type: custom
      action:
        service:
          name: user-service
          method: DELETE
          path: /api/users/{{.output.create-user.user_id}}
    timeout: 30s
    dependencies:
      - validate-user

  # 步骤 3: 发送欢迎邮件
  - id: send-welcome-email
    name: Send Welcome Email
    description: Send welcome email to new user
    type: service
    action:
      service:
        name: notification-service
        endpoint: http://notification-service:8080
        method: POST
        path: /api/notifications/email
        body:
          to: "{{.input.email}}"
          template: welcome
          data:
            username: "{{.input.username}}"
            user_id: "{{.output.create-user.user_id}}"
    timeout: 1m
    dependencies:
      - create-user
    async: true  # 异步执行
```

### 加载和执行 DSL

```go
package main

import (
    "context"
    "fmt"
    "io/ioutil"

    "github.com/innovationmech/swit/pkg/saga/dsl"
    "github.com/innovationmech/swit/pkg/saga/coordinator"
)

func main() {
    // 读取 DSL 文件
    yamlData, err := ioutil.ReadFile("user-registration.saga.yaml")
    if err != nil {
        panic(err)
    }
    
    // 解析 DSL
    parser := dsl.NewParser()
    definition, err := parser.Parse(yamlData)
    if err != nil {
        panic(fmt.Errorf("DSL 解析失败: %w", err))
    }
    
    // 验证 DSL
    if err := definition.Validate(); err != nil {
        panic(fmt.Errorf("DSL 验证失败: %w", err))
    }
    
    fmt.Printf("Saga 定义加载成功: %s\n", definition.GetName())
    
    // 创建 Coordinator
    coordinator := createCoordinator()
    defer coordinator.Close()
    
    // 准备输入数据
    inputData := map[string]interface{}{
        "username": "alice",
        "email":    "alice@example.com",
        "password": "SecureP@ss123",
    }
    
    // 启动 Saga
    ctx := context.Background()
    instance, err := coordinator.StartSaga(ctx, definition, inputData)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Saga 已启动: ID=%s\n", instance.GetID())
}
```

### 高级 DSL 特性

#### 1. 条件执行

```yaml
steps:
  - id: apply-discount
    name: Apply Discount
    description: Apply discount if order amount > $100
    type: service
    action:
      service:
        name: pricing-service
        method: POST
        path: /api/discounts/apply
    # 条件: 仅当订单金额大于 100 时执行
    condition:
      expression: "$input.amount > 100"
```

#### 2. 并行步骤

```yaml
steps:
  # 步骤 1: 预留库存
  - id: reserve-inventory
    name: Reserve Inventory
    # ...

  # 步骤 2: 验证支付方式 (与步骤 1 并行)
  - id: validate-payment-method
    name: Validate Payment Method
    dependencies: []  # 无依赖,可并行
    # ...

  # 步骤 3: 处理支付 (依赖前两步)
  - id: process-payment
    name: Process Payment
    dependencies:
      - reserve-inventory
      - validate-payment-method
    # ...
```

#### 3. 循环和重试

```yaml
steps:
  - id: notify-suppliers
    name: Notify Suppliers
    type: service
    # 为每个供应商发送通知
    foreach:
      items: "{{.input.suppliers}}"
      var: supplier
      action:
        service:
          name: notification-service
          method: POST
          path: /api/notify
          body:
            supplier_id: "{{.supplier.id}}"
            message: "New order received"
    # 单个通知失败不影响其他
    continue_on_error: true
```

#### 4. 动态超时

```yaml
steps:
  - id: process-payment
    name: Process Payment
    # 根据支付金额动态设置超时
    timeout: "{{ if gt .input.amount 1000 }}5m{{ else }}2m{{ end }}"
```

### 完整的电商订单 DSL 示例

参考项目中的示例文件:
- `examples/saga-dsl/order-processing.saga.yaml`
- `examples/saga-dsl/payment-flow.saga.yaml`
- `examples/saga-dsl/complex-travel-booking.saga.yaml`

### DSL 验证工具

使用命令行工具验证 DSL 文件:

```bash
# 验证 DSL 语法
swit saga validate user-registration.saga.yaml

# 生成流程图
swit saga visualize user-registration.saga.yaml > flow.dot
dot -Tpng flow.dot > flow.png

# 测试 DSL (干运行)
swit saga dry-run user-registration.saga.yaml --input input.json
```

### 💡 关键要点

1. **声明式定义**: 用 YAML 定义 Saga 流程
2. **模板变量**: 使用 `{{.input.field}}` 和 `{{.output.step.field}}`
3. **条件执行**: 通过 `condition` 实现分支逻辑
4. **依赖管理**: 通过 `dependencies` 控制执行顺序
5. **验证工具**: 使用 CLI 工具验证和测试 DSL

---

## 教程 6: 复杂业务流程编排

### 学习目标

- 处理复杂的依赖关系
- 实现条件分支
- 管理长时间运行的 Saga

### 场景: 复杂旅游预订系统

预订包含:
1. 预订机票
2. 预订酒店
3. 预订租车 (可选)
4. 购买旅游保险 (可选)
5. 发送确认邮件

#### 依赖关系图

```
                    ┌─────────────────┐
                    │  Validate User  │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
    ┌─────────▼────────┐         ┌─────────▼────────┐
    │  Book Flight     │         │  Book Hotel      │
    │  (并行)           │         │  (并行)           │
    └─────────┬────────┘         └─────────┬────────┘
              │                             │
              └──────────────┬──────────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
    ┌─────────▼────────┐         ┌─────────▼────────┐
    │  Book Car        │         │  Buy Insurance   │
    │  (条件:可选)      │         │  (条件:可选)      │
    └─────────┬────────┘         └─────────┬────────┘
              │                             │
              └──────────────┬──────────────┘
                             │
                    ┌────────▼────────┐
                    │  Process Payment │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  Send Confirmation│
                    └─────────────────┘
```

#### 实现代码

```go
// TravelBookingSaga 旅游预订 Saga
type TravelBookingSaga struct {
    id string
}

func NewTravelBookingSaga() *TravelBookingSaga {
    return &TravelBookingSaga{
        id: "travel-booking-saga",
    }
}

func (d *TravelBookingSaga) GetSteps() []saga.SagaStep {
    return []saga.SagaStep{
        // 步骤 1: 验证用户
        &ValidateUserStep{},
        
        // 步骤 2-3: 并行预订机票和酒店
        // 注意: 这两个步骤没有相互依赖,可以并行执行
        &BookFlightStep{},
        &BookHotelStep{},
        
        // 步骤 4: 条件步骤 - 预订租车
        &BookCarStep{
            condition: func(data interface{}) bool {
                booking := data.(*TravelBookingData)
                return booking.NeedsCar
            },
        },
        
        // 步骤 5: 条件步骤 - 购买保险
        &BuyInsuranceStep{
            condition: func(data interface{}) bool {
                booking := data.(*TravelBookingData)
                return booking.NeedsInsurance
            },
        },
        
        // 步骤 6: 处理支付
        &ProcessPaymentStep{},
        
        // 步骤 7: 发送确认
        &SendConfirmationStep{},
    }
}
```

#### 条件步骤实现

```go
// BookCarStep 预订租车步骤 (可选)
type BookCarStep struct {
    condition func(interface{}) bool
}

func (s *BookCarStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    bookingData := data.(*TravelBookingData)
    
    // 检查条件
    if s.condition != nil && !s.condition(data) {
        fmt.Println("跳过租车预订 (用户不需要)")
        return bookingData, nil
    }
    
    fmt.Println("正在预订租车...")
    
    // 调用租车服务
    carBooking, err := s.bookCarService(ctx, bookingData)
    if err != nil {
        return nil, err
    }
    
    bookingData.CarBookingID = carBooking.ID
    bookingData.CarReserved = true
    
    fmt.Printf("租车预订成功: %s\n", carBooking.ID)
    return bookingData, nil
}

func (s *BookCarStep) Compensate(ctx context.Context, data interface{}) error {
    bookingData := data.(*TravelBookingData)
    
    // 如果没有预订租车,跳过补偿
    if !bookingData.CarReserved {
        return nil
    }
    
    fmt.Printf("取消租车预订: %s\n", bookingData.CarBookingID)
    
    // 调用取消 API
    err := s.cancelCarBooking(ctx, bookingData.CarBookingID)
    if err != nil {
        fmt.Printf("警告: 取消租车失败 - %v\n", err)
    }
    
    bookingData.CarReserved = false
    return nil
}
```

### 处理长时间运行的 Saga

某些 Saga 可能需要运行数小时甚至数天:

```go
// LongRunningBookingSaga 长时间运行的预订 Saga
type LongRunningBookingSaga struct {
    id string
}

func (d *LongRunningBookingSaga) GetTimeout() time.Duration {
    // 设置更长的超时时间
    return 24 * time.Hour
}

func (d *LongRunningBookingSaga) GetSteps() []saga.SagaStep {
    return []saga.SagaStep{
        // 步骤 1: 提交预订请求
        &SubmitBookingRequestStep{},
        
        // 步骤 2: 等待供应商确认 (可能需要几个小时)
        &WaitForSupplierConfirmationStep{
            pollInterval: 5 * time.Minute,
            timeout:      6 * time.Hour,
        },
        
        // 步骤 3: 处理支付
        &ProcessPaymentStep{},
        
        // 步骤 4: 发送确认
        &SendConfirmationStep{},
    }
}

// WaitForSupplierConfirmationStep 等待供应商确认
type WaitForSupplierConfirmationStep struct {
    pollInterval time.Duration
    timeout      time.Duration
}

func (s *WaitForSupplierConfirmationStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    bookingData := data.(*BookingData)
    
    fmt.Printf("等待供应商确认: %s\n", bookingData.BookingRequestID)
    
    ticker := time.NewTicker(s.pollInterval)
    defer ticker.Stop()
    
    timeoutCtx, cancel := context.WithTimeout(ctx, s.timeout)
    defer cancel()
    
    for {
        select {
        case <-timeoutCtx.Done():
            return nil, fmt.Errorf("供应商确认超时")
            
        case <-ticker.C:
            // 轮询供应商状态
            status, err := s.checkSupplierStatus(ctx, bookingData.BookingRequestID)
            if err != nil {
                fmt.Printf("检查状态失败: %v, 继续等待...\n", err)
                continue
            }
            
            if status == "CONFIRMED" {
                fmt.Println("供应商已确认预订")
                bookingData.SupplierConfirmed = true
                bookingData.ConfirmationID = status.ConfirmationID
                return bookingData, nil
            }
            
            if status == "REJECTED" {
                return nil, fmt.Errorf("供应商拒绝了预订请求: %s", status.Reason)
            }
            
            // 继续等待
            fmt.Println("预订仍在处理中...")
        }
    }
}
```

### 分支和合并

实现复杂的分支逻辑:

```go
// ConditionalSaga 带条件分支的 Saga
func NewConditionalOrderSaga() saga.SagaDefinition {
    return &ConditionalOrderSaga{}
}

func (d *ConditionalOrderSaga) GetSteps() []saga.SagaStep {
    return []saga.SagaStep{
        // 步骤 1: 验证订单
        &ValidateOrderStep{},
        
        // 步骤 2a: VIP 客户走快速通道
        &VIPFastTrackStep{
            condition: func(data interface{}) bool {
                order := data.(*OrderData)
                return order.CustomerLevel == "VIP"
            },
        },
        
        // 步骤 2b: 普通客户走标准流程
        &StandardProcessStep{
            condition: func(data interface{}) bool {
                order := data.(*OrderData)
                return order.CustomerLevel != "VIP"
            },
        },
        
        // 步骤 3: 合并点 - 处理支付
        &ProcessPaymentStep{},
    }
}
```

### 子 Saga

将复杂流程分解为子 Saga:

```go
// ParentSaga 父 Saga
type ParentSaga struct {
    id string
}

func (d *ParentSaga) GetSteps() []saga.SagaStep {
    return []saga.SagaStep{
        // 普通步骤
        &CreateOrderStep{},
        
        // 子 Saga 步骤
        &SubSagaStep{
            sagaDefinition: NewInventoryManagementSaga(),
        },
        
        // 继续父 Saga
        &ConfirmOrderStep{},
    }
}

// SubSagaStep 执行子 Saga 的步骤
type SubSagaStep struct {
    sagaDefinition saga.SagaDefinition
    coordinator    saga.SagaCoordinator
}

func (s *SubSagaStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    fmt.Println("启动子 Saga...")
    
    // 启动子 Saga
    instance, err := s.coordinator.StartSaga(ctx, s.sagaDefinition, data)
    if err != nil {
        return nil, fmt.Errorf("子 Saga 启动失败: %w", err)
    }
    
    // 等待子 Saga 完成
    for !instance.IsTerminal() {
        time.Sleep(100 * time.Millisecond)
        instance, _ = s.coordinator.GetSagaInstance(instance.GetID())
    }
    
    if instance.GetState() == saga.StateCompleted {
        fmt.Println("子 Saga 执行成功")
        return instance.GetResult(), nil
    }
    
    return nil, fmt.Errorf("子 Saga 执行失败")
}
```

### 💡 关键要点

1. **依赖管理**: 明确步骤之间的依赖关系
2. **并行执行**: 无依赖的步骤可以并行执行
3. **条件逻辑**: 使用条件判断实现分支
4. **长时间运行**: 设置合理的超时和轮询策略
5. **子 Saga**: 将复杂流程分解为可管理的部分

---

## 教程 7: 监控和可观测性

### 学习目标

- 集成 OpenTelemetry 分布式追踪
- 配置 Prometheus 指标
- 实现日志记录
- 构建监控仪表盘

### 分布式追踪

#### 集成 OpenTelemetry

```go
package main

import (
    "context"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
    "github.com/innovationmech/swit/pkg/saga"
)

// TracedStep 带追踪的步骤包装器
type TracedStep struct {
    step   saga.SagaStep
    tracer trace.Tracer
}

func NewTracedStep(step saga.SagaStep) *TracedStep {
    return &TracedStep{
        step:   step,
        tracer: otel.Tracer("saga"),
    }
}

func (s *TracedStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 创建追踪 span
    ctx, span := s.tracer.Start(ctx, s.step.GetName(),
        trace.WithAttributes(
            attribute.String("step.id", s.step.GetID()),
            attribute.String("step.name", s.step.GetName()),
        ),
    )
    defer span.End()
    
    // 执行步骤
    result, err := s.step.Execute(ctx, data)
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    } else {
        span.SetStatus(codes.Ok, "success")
    }
    
    return result, err
}

func (s *TracedStep) Compensate(ctx context.Context, data interface{}) error {
    // 创建补偿追踪 span
    ctx, span := s.tracer.Start(ctx, s.step.GetName()+".compensate",
        trace.WithAttributes(
            attribute.String("step.id", s.step.GetID()),
            attribute.String("operation", "compensate"),
        ),
    )
    defer span.End()
    
    // 执行补偿
    err := s.step.Compensate(ctx, data)
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    } else {
        span.SetStatus(codes.Ok, "compensated")
    }
    
    return err
}

// 委托其他方法给原始步骤
func (s *TracedStep) GetID() string          { return s.step.GetID() }
func (s *TracedStep) GetName() string        { return s.step.GetName() }
func (s *TracedStep) GetDescription() string { return s.step.GetDescription() }
// ... 其他方法
```

#### 使用追踪

```go
func main() {
    // 初始化 OpenTelemetry
    tp := initTracer()
    defer tp.Shutdown(context.Background())
    
    // 创建带追踪的步骤
    tracedSteps := []saga.SagaStep{
        NewTracedStep(&CreateOrderStep{}),
        NewTracedStep(&ReserveInventoryStep{}),
        NewTracedStep(&ProcessPaymentStep{}),
    }
    
    definition := &OrderSaga{
        steps: tracedSteps,
    }
    
    // 启动 Saga
    ctx := context.Background()
    coordinator.StartSaga(ctx, definition, orderData)
}

func initTracer() *sdktrace.TracerProvider {
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint("http://localhost:14268/api/traces"),
    ))
    if err != nil {
        panic(err)
    }
    
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("saga-service"),
        )),
    )
    
    otel.SetTracerProvider(tp)
    return tp
}
```

### Prometheus 指标

#### 定义指标

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Saga 总数计数器
    SagaTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "saga_total",
            Help: "Total number of sagas started",
        },
        []string{"saga_id", "saga_name"},
    )
    
    // Saga 完成计数器
    SagaCompleted = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "saga_completed_total",
            Help: "Total number of completed sagas",
        },
        []string{"saga_id", "status"},
    )
    
    // Saga 执行时间直方图
    SagaDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "saga_duration_seconds",
            Help:    "Saga execution duration in seconds",
            Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
        },
        []string{"saga_id", "status"},
    )
    
    // 步骤执行计数器
    StepExecutions = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "saga_step_executions_total",
            Help: "Total number of step executions",
        },
        []string{"saga_id", "step_id", "result"},
    )
    
    // 补偿执行计数器
    CompensationExecutions = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "saga_compensations_total",
            Help: "Total number of compensation executions",
        },
        []string{"saga_id", "step_id", "result"},
    )
    
    // 重试计数器
    Retries = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "saga_retries_total",
            Help: "Total number of step retries",
        },
        []string{"saga_id", "step_id"},
    )
    
    // 当前活跃 Saga 数量
    ActiveSagas = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "saga_active",
            Help: "Number of currently active sagas",
        },
    )
)
```

#### 记录指标

```go
// MeteredCoordinator 带指标的 Coordinator 包装器
type MeteredCoordinator struct {
    coordinator saga.SagaCoordinator
}

func NewMeteredCoordinator(coordinator saga.SagaCoordinator) *MeteredCoordinator {
    return &MeteredCoordinator{
        coordinator: coordinator,
    }
}

func (c *MeteredCoordinator) StartSaga(
    ctx context.Context,
    definition saga.SagaDefinition,
    initialData interface{},
) (saga.SagaInstance, error) {
    // 记录 Saga 启动
    metrics.SagaTotal.WithLabelValues(
        definition.GetID(),
        definition.GetName(),
    ).Inc()
    
    metrics.ActiveSagas.Inc()
    
    startTime := time.Now()
    
    // 执行 Saga
    instance, err := c.coordinator.StartSaga(ctx, definition, initialData)
    
    if err != nil {
        return nil, err
    }
    
    // 异步监控 Saga 完成
    go c.monitorSaga(instance.GetID(), definition, startTime)
    
    return instance, nil
}

func (c *MeteredCoordinator) monitorSaga(
    sagaID string,
    definition saga.SagaDefinition,
    startTime time.Time,
) {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        instance, err := c.coordinator.GetSagaInstance(sagaID)
        if err != nil {
            return
        }
        
        if instance.IsTerminal() {
            // 记录完成指标
            duration := time.Since(startTime).Seconds()
            status := instance.GetState().String()
            
            metrics.SagaCompleted.WithLabelValues(
                definition.GetID(),
                status,
            ).Inc()
            
            metrics.SagaDuration.WithLabelValues(
                definition.GetID(),
                status,
            ).Observe(duration)
            
            metrics.ActiveSagas.Dec()
            
            return
        }
    }
}
```

### 结构化日志

```go
package logging

import (
    "context"
    
    "go.uber.org/zap"
    "github.com/innovationmech/swit/pkg/saga"
)

// LoggedStep 带日志的步骤包装器
type LoggedStep struct {
    step   saga.SagaStep
    logger *zap.Logger
}

func NewLoggedStep(step saga.SagaStep, logger *zap.Logger) *LoggedStep {
    return &LoggedStep{
        step:   step,
        logger: logger,
    }
}

func (s *LoggedStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    stepLogger := s.logger.With(
        zap.String("step_id", s.step.GetID()),
        zap.String("step_name", s.step.GetName()),
        zap.String("operation", "execute"),
    )
    
    stepLogger.Info("步骤开始执行")
    
    startTime := time.Now()
    result, err := s.step.Execute(ctx, data)
    duration := time.Since(startTime)
    
    if err != nil {
        stepLogger.Error("步骤执行失败",
            zap.Error(err),
            zap.Duration("duration", duration),
        )
    } else {
        stepLogger.Info("步骤执行成功",
            zap.Duration("duration", duration),
        )
    }
    
    return result, err
}

func (s *LoggedStep) Compensate(ctx context.Context, data interface{}) error {
    stepLogger := s.logger.With(
        zap.String("step_id", s.step.GetID()),
        zap.String("step_name", s.step.GetName()),
        zap.String("operation", "compensate"),
    )
    
    stepLogger.Warn("开始补偿操作")
    
    startTime := time.Now()
    err := s.step.Compensate(ctx, data)
    duration := time.Since(startTime)
    
    if err != nil {
        stepLogger.Error("补偿操作失败",
            zap.Error(err),
            zap.Duration("duration", duration),
        )
    } else {
        stepLogger.Info("补偿操作成功",
            zap.Duration("duration", duration),
        )
    }
    
    return err
}
```

### Grafana 仪表盘

创建文件 `grafana-dashboard.json`:

```json
{
  "dashboard": {
    "title": "Saga Monitoring",
    "panels": [
      {
        "title": "Saga Throughput",
        "targets": [
          {
            "expr": "rate(saga_total[5m])"
          }
        ]
      },
      {
        "title": "Success Rate",
        "targets": [
          {
            "expr": "rate(saga_completed_total{status=\"COMPLETED\"}[5m]) / rate(saga_total[5m])"
          }
        ]
      },
      {
        "title": "Average Duration",
        "targets": [
          {
            "expr": "rate(saga_duration_seconds_sum[5m]) / rate(saga_duration_seconds_count[5m])"
          }
        ]
      },
      {
        "title": "Active Sagas",
        "targets": [
          {
            "expr": "saga_active"
          }
        ]
      }
    ]
  }
}
```

### 💡 关键要点

1. **分布式追踪**: 使用 OpenTelemetry 追踪 Saga 执行
2. **指标收集**: 用 Prometheus 记录关键指标
3. **结构化日志**: 使用结构化日志便于查询和分析
4. **可视化**: 构建 Grafana 仪表盘监控 Saga 健康状况
5. **告警**: 基于指标配置告警规则

---

## 教程 8: 生产环境最佳实践

### 学习目标

- 生产环境配置建议
- 性能优化技巧
- 安全性考虑
- 故障处理流程

### 配置最佳实践

#### 1. 环境分离

为不同环境使用不同的配置:

```yaml
# config/production.yaml
saga:
  coordinator:
    # 生产环境使用更大的并发数
    max_concurrent_sagas: 500
    worker_pool_size: 100
    acquire_timeout: 30s
    shutdown_timeout: 2m

  state_storage:
    type: redis
    redis:
      # 使用 Redis 集群
      cluster:
        - redis-1.prod:6379
        - redis-2.prod:6379
        - redis-3.prod:6379
      password: ${REDIS_PASSWORD}
      max_retries: 5
      pool_size: 200

  event_publisher:
    type: kafka
    kafka:
      brokers:
        - kafka-1.prod:9092
        - kafka-2.prod:9092
        - kafka-3.prod:9092
      topic: saga-events-prod
      compression: snappy
      batch_size: 1000

  retry:
    default_policy:
      type: exponential_backoff
      max_attempts: 5
      initial_delay: 1s
      max_delay: 2m
      jitter: true

  monitoring:
    enabled: true
    prometheus:
      port: 9090
      path: /metrics
    tracing:
      enabled: true
      jaeger:
        endpoint: http://jaeger.prod:14268/api/traces
        sample_rate: 0.1  # 生产环境降低采样率
```

```yaml
# config/development.yaml
saga:
  coordinator:
    # 开发环境使用较小的并发数
    max_concurrent_sagas: 10
    worker_pool_size: 5

  state_storage:
    type: memory  # 开发环境使用内存存储

  event_publisher:
    type: memory

  retry:
    default_policy:
      max_attempts: 2  # 开发环境快速失败

  monitoring:
    tracing:
      sample_rate: 1.0  # 开发环境全量采样
```

#### 2. 连接池配置

```go
// 生产环境 Redis 配置
redisConfig := &redis.ClusterOptions{
    Addrs: []string{
        "redis-1:6379",
        "redis-2:6379",
        "redis-3:6379",
    },
    Password: os.Getenv("REDIS_PASSWORD"),
    
    // 连接池配置
    PoolSize:     200,
    MinIdleConns: 50,
    MaxRetries:   5,
    
    // 超时配置
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
    PoolTimeout:  4 * time.Second,
    
    // 心跳检测
    MaxConnAge:     0,
    IdleTimeout:    5 * time.Minute,
    IdleCheckFrequency: time.Minute,
}
```

### 性能优化

#### 1. 批量操作

```go
// 批量保存步骤状态
type BatchStateStorage struct {
    storage   saga.StateStorage
    batchSize int
    flushInterval time.Duration
    
    buffer    []*saga.StepState
    mu        sync.Mutex
}

func (s *BatchStateStorage) SaveStepState(
    ctx context.Context,
    sagaID string,
    step *saga.StepState,
) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.buffer = append(s.buffer, step)
    
    // 达到批量大小,执行刷新
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
    err := s.storage.BatchSaveStepStates(ctx, s.buffer)
    if err != nil {
        return err
    }
    
    s.buffer = s.buffer[:0]
    return nil
}
```

#### 2. 异步事件发布

```go
// 异步事件发布器
type AsyncEventPublisher struct {
    publisher saga.EventPublisher
    buffer    chan *saga.SagaEvent
    batchSize int
}

func NewAsyncEventPublisher(publisher saga.EventPublisher, bufferSize, batchSize int) *AsyncEventPublisher {
    aep := &AsyncEventPublisher{
        publisher: publisher,
        buffer:    make(chan *saga.SagaEvent, bufferSize),
        batchSize: batchSize,
    }
    
    // 启动后台发布协程
    go aep.publishLoop()
    
    return aep
}

func (p *AsyncEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
    select {
    case p.buffer <- event:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    default:
        return fmt.Errorf("event buffer full")
    }
}

func (p *AsyncEventPublisher) publishLoop() {
    batch := make([]*saga.SagaEvent, 0, p.batchSize)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case event := <-p.buffer:
            batch = append(batch, event)
            
            if len(batch) >= p.batchSize {
                p.publishBatch(batch)
                batch = batch[:0]
            }
            
        case <-ticker.C:
            if len(batch) > 0 {
                p.publishBatch(batch)
                batch = batch[:0]
            }
        }
    }
}

func (p *AsyncEventPublisher) publishBatch(events []*saga.SagaEvent) {
    ctx := context.Background()
    for _, event := range events {
        if err := p.publisher.PublishEvent(ctx, event); err != nil {
            // 记录错误但不阻塞
            fmt.Printf("Failed to publish event: %v\n", err)
        }
    }
}
```

### 安全性

#### 1. 敏感数据处理

```go
// SensitiveDataStep 处理敏感数据的步骤
type SensitiveDataStep struct {
    encryptor Encryptor
}

func (s *SensitiveDataStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 加密敏感数据
    encryptedCard, err := s.encryptor.Encrypt(orderData.CreditCard)
    if err != nil {
        return nil, err
    }
    
    // 存储加密后的数据
    orderData.EncryptedCard = encryptedCard
    orderData.CreditCard = ""  // 清除明文
    
    return orderData, nil
}

// 日志中屏蔽敏感信息
func (o *OrderData) MarshalJSON() ([]byte, error) {
    type Alias OrderData
    return json.Marshal(&struct {
        CreditCard string `json:"credit_card,omitempty"`
        *Alias
    }{
        CreditCard: "****",  // 屏蔽信用卡号
        Alias:      (*Alias)(o),
    })
}
```

#### 2. 权限控制

```go
// AuthorizedCoordinator 带权限控制的 Coordinator
type AuthorizedCoordinator struct {
    coordinator saga.SagaCoordinator
    authz       Authorizer
}

func (c *AuthorizedCoordinator) StartSaga(
    ctx context.Context,
    definition saga.SagaDefinition,
    initialData interface{},
) (saga.SagaInstance, error) {
    // 检查权限
    if !c.authz.CanStartSaga(ctx, definition.GetID()) {
        return nil, fmt.Errorf("unauthorized: cannot start saga %s", definition.GetID())
    }
    
    return c.coordinator.StartSaga(ctx, definition, initialData)
}
```

### 故障处理

#### 1. 断路器模式

```go
// CircuitBreakerStep 带断路器的步骤
type CircuitBreakerStep struct {
    step    saga.SagaStep
    breaker *CircuitBreaker
}

func (s *CircuitBreakerStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 检查断路器状态
    if s.breaker.IsOpen() {
        return nil, fmt.Errorf("circuit breaker open for step %s", s.step.GetID())
    }
    
    // 执行步骤
    result, err := s.step.Execute(ctx, data)
    
    if err != nil {
        s.breaker.RecordFailure()
    } else {
        s.breaker.RecordSuccess()
    }
    
    return result, err
}
```

#### 2. 死信队列

```go
// 处理失败的 Saga
func HandleFailedSagas(coordinator saga.SagaCoordinator, dlqHandler DLQHandler) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        ctx := context.Background()
        
        // 查询失败的 Saga
        filter := &saga.SagaFilter{
            States: []saga.SagaState{saga.StateFailed},
        }
        
        instances, _ := coordinator.GetActiveSagas(filter)
        
        for _, instance := range instances {
            // 发送到死信队列
            dlqHandler.Send(&DLQMessage{
                SagaID:    instance.GetID(),
                Error:     instance.GetError(),
                Timestamp: time.Now(),
                Retries:   instance.GetRetryCount(),
            })
            
            // 清理失败的 Saga
            coordinator.DeleteSaga(ctx, instance.GetID())
        }
    }
}
```

### 健康检查

```go
// HealthChecker Saga 系统健康检查
type HealthChecker struct {
    coordinator saga.SagaCoordinator
    storage     saga.StateStorage
    publisher   saga.EventPublisher
}

func (h *HealthChecker) Check(ctx context.Context) error {
    // 检查 Coordinator
    if err := h.coordinator.HealthCheck(ctx); err != nil {
        return fmt.Errorf("coordinator unhealthy: %w", err)
    }
    
    // 检查状态存储
    if err := h.storage.HealthCheck(ctx); err != nil {
        return fmt.Errorf("storage unhealthy: %w", err)
    }
    
    // 检查事件发布器
    if err := h.publisher.HealthCheck(ctx); err != nil {
        return fmt.Errorf("publisher unhealthy: %w", err)
    }
    
    return nil
}

// HTTP 健康检查端点
func healthCheckHandler(checker *HealthChecker) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()
        
        if err := checker.Check(ctx); err != nil {
            w.WriteHeader(http.StatusServiceUnavailable)
            json.NewEncoder(w).Encode(map[string]string{
                "status": "unhealthy",
                "error":  err.Error(),
            })
            return
        }
        
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "healthy",
        })
    }
}
```

### 💡 关键要点

1. **环境分离**: 为不同环境使用不同的配置
2. **性能优化**: 使用批量操作和异步处理
3. **安全性**: 加密敏感数据,实施权限控制
4. **故障处理**: 使用断路器和死信队列
5. **健康检查**: 实现完善的健康检查机制
6. **监控告警**: 配置全面的监控和告警

---

## 总结

通过这 8 个教程,您应该已经掌握了:

✅ Saga 基础概念和第一个 Saga 实现
✅ 步骤设计和补偿操作
✅ 错误处理和重试策略
✅ 状态持久化和故障恢复
✅ 使用 DSL 定义复杂流程
✅ 复杂业务流程编排
✅ 监控和可观测性
✅ 生产环境最佳实践

## 下一步

- 阅读 [最佳实践文档](best-practices.md)
- 查看 [API 参考](../saga-api-reference.md)
- 探索 [示例代码](../../examples/saga-orchestrator/)
- 加入社区讨论

## 参考资源

- [Saga Pattern](https://microservices.io/patterns/data/saga.html)
- [分布式事务](https://martinfowler.com/articles/patterns-of-distributed-systems/saga.html)
- [Swit Framework 文档](../README.md)


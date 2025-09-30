# CQRS Pattern

CQRS (Command Query Responsibility Segregation) Pattern 实现提供了命令和查询的分离处理机制，用于构建更清晰、更易维护的业务逻辑。

## 概述

CQRS 模式将系统的读操作（查询）和写操作（命令）分离，通过不同的模型和处理流程来处理。该模式特别适用于复杂业务逻辑、高性能需求和事件驱动架构。

### 核心概念

- **Command**: 表示系统中的写操作，用于改变系统状态
- **Query**: 表示系统中的读操作，不应改变系统状态
- **CommandBus**: 命令总线，负责将命令路由到对应的处理器
- **QueryBus**: 查询总线，负责将查询路由到对应的处理器
- **Handler**: 处理器，负责执行具体的命令或查询逻辑

## 特性

- ✅ 命令和查询的清晰分离
- ✅ 处理器注册和路由
- ✅ 中间件支持（日志、验证、权限等）
- ✅ 类型安全的接口设计
- ✅ 线程安全的实现
- ✅ 简单易用的 API

## 安装

```go
import "github.com/innovationmech/swit/pkg/patterns/cqrs"
```

## 快速开始

### 1. 定义命令和处理器

```go
// 定义命令
type CreateOrderCommand struct {
    OrderID    string
    CustomerID string
    Amount     float64
}

func (c *CreateOrderCommand) GetCommandName() string {
    return "order.create"
}

// 定义命令处理器
type CreateOrderHandler struct {
    orderRepo OrderRepository
}

func (h *CreateOrderHandler) Handle(ctx context.Context, cmd cqrs.Command) error {
    orderCmd := cmd.(*CreateOrderCommand)
    
    // 执行业务逻辑
    order := &Order{
        ID:         orderCmd.OrderID,
        CustomerID: orderCmd.CustomerID,
        Amount:     orderCmd.Amount,
        Status:     "pending",
    }
    
    return h.orderRepo.Save(ctx, order)
}

func (h *CreateOrderHandler) GetCommandName() string {
    return "order.create"
}
```

### 2. 使用命令总线

```go
// 创建命令总线
commandBus := cqrs.NewCommandBus()

// 注册命令处理器
createOrderHandler := &CreateOrderHandler{orderRepo: repo}
err := commandBus.RegisterHandler(createOrderHandler)
if err != nil {
    log.Fatal(err)
}

// 创建并分发命令
cmd := &CreateOrderCommand{
    OrderID:    "order-001",
    CustomerID: "customer-123",
    Amount:     99.99,
}

err = commandBus.Dispatch(context.Background(), cmd)
if err != nil {
    log.Printf("Command failed: %v", err)
}
```

### 3. 定义查询和处理器

```go
// 定义查询
type GetOrderQuery struct {
    OrderID string
}

func (q *GetOrderQuery) GetQueryName() string {
    return "order.get"
}

// 定义查询处理器
type GetOrderHandler struct {
    readModel OrderReadModel
}

func (h *GetOrderHandler) Handle(ctx context.Context, query cqrs.Query) (interface{}, error) {
    orderQuery := query.(*GetOrderQuery)
    
    // 从读模型中查询
    order, err := h.readModel.FindByID(ctx, orderQuery.OrderID)
    if err != nil {
        return nil, err
    }
    
    return order, nil
}

func (h *GetOrderHandler) GetQueryName() string {
    return "order.get"
}
```

### 4. 使用查询总线

```go
// 创建查询总线
queryBus := cqrs.NewQueryBus()

// 注册查询处理器
getOrderHandler := &GetOrderHandler{readModel: readModel}
err := queryBus.RegisterHandler(getOrderHandler)
if err != nil {
    log.Fatal(err)
}

// 创建并分发查询
query := &GetOrderQuery{
    OrderID: "order-001",
}

result, err := queryBus.Dispatch(context.Background(), query)
if err != nil {
    log.Printf("Query failed: %v", err)
    return
}

order := result.(*OrderDTO)
fmt.Printf("Order: %+v\n", order)
```

## 架构

```
┌─────────────────────────────────────────────────────┐
│                   Application                       │
└───────────────┬─────────────────┬───────────────────┘
                │                 │
        Commands│                 │Queries
                │                 │
                ▼                 ▼
        ┌───────────────┐ ┌───────────────┐
        │  CommandBus   │ │   QueryBus    │
        └───────┬───────┘ └───────┬───────┘
                │                 │
                │ Route           │ Route
                ▼                 ▼
        ┌───────────────┐ ┌───────────────┐
        │Command Handler│ │ Query Handler │
        └───────┬───────┘ └───────┬───────┘
                │                 │
        Write   │                 │ Read
        Model   ▼                 ▼  Model
        ┌───────────────┐ ┌───────────────┐
        │   Database    │ │  Read Model   │
        │  (Write Side) │ │  (Read Side)  │
        └───────────────┘ └───────────────┘
```

## 高级用法

### 中间件

可以为命令总线添加中间件来实现日志、验证、权限检查等功能：

```go
// 日志中间件
loggingMiddleware := func(ctx context.Context, cmd cqrs.Command, next func(context.Context, cqrs.Command) error) error {
    log.Printf("Executing command: %s", cmd.GetCommandName())
    err := next(ctx, cmd)
    if err != nil {
        log.Printf("Command failed: %v", err)
    }
    return err
}

// 验证中间件
validationMiddleware := func(ctx context.Context, cmd cqrs.Command, next func(context.Context, cqrs.Command) error) error {
    // 执行验证逻辑
    if validator, ok := cmd.(Validator); ok {
        if err := validator.Validate(); err != nil {
            return fmt.Errorf("validation failed: %w", err)
        }
    }
    return next(ctx, cmd)
}

// 添加中间件（需要将 CommandBus 转换为具体类型）
bus := cqrs.NewCommandBus().(*cqrs.CommandBus)
bus.AddMiddleware(validationMiddleware)
bus.AddMiddleware(loggingMiddleware)
```

### 与事件溯源集成

CQRS 模式常与事件溯源（Event Sourcing）结合使用：

```go
type CreateOrderHandler struct {
    eventStore EventStore
    eventBus   messaging.EventPublisher
}

func (h *CreateOrderHandler) Handle(ctx context.Context, cmd cqrs.Command) error {
    orderCmd := cmd.(*CreateOrderCommand)
    
    // 创建领域事件
    event := &OrderCreatedEvent{
        OrderID:    orderCmd.OrderID,
        CustomerID: orderCmd.CustomerID,
        Amount:     orderCmd.Amount,
        Timestamp:  time.Now(),
    }
    
    // 存储事件
    if err := h.eventStore.Append(ctx, event); err != nil {
        return err
    }
    
    // 发布事件到消息总线
    if err := h.eventBus.Publish(ctx, event); err != nil {
        log.Printf("Failed to publish event: %v", err)
    }
    
    return nil
}
```

### 处理器组合

可以在一个处理器中组合多个操作：

```go
type CompleteOrderHandler struct {
    commandBus cqrs.CommandBus
    queryBus   cqrs.QueryBus
    orderRepo  OrderRepository
}

func (h *CompleteOrderHandler) Handle(ctx context.Context, cmd cqrs.Command) error {
    completeCmd := cmd.(*CompleteOrderCommand)
    
    // 1. 查询订单状态
    query := &GetOrderQuery{OrderID: completeCmd.OrderID}
    result, err := h.queryBus.Dispatch(ctx, query)
    if err != nil {
        return err
    }
    
    order := result.(*OrderDTO)
    if order.Status != "pending" {
        return errors.New("order is not in pending status")
    }
    
    // 2. 更新订单状态
    if err := h.orderRepo.UpdateStatus(ctx, order.OrderID, "completed"); err != nil {
        return err
    }
    
    // 3. 发送通知命令
    notifyCmd := &SendNotificationCommand{
        CustomerID: order.CustomerID,
        Message:    "Your order is completed",
    }
    
    return h.commandBus.Dispatch(ctx, notifyCmd)
}
```

## 最佳实践

### 命令设计

1. **命令应该是不可变的**
2. **命令名称应该使用动词**，如 `CreateOrder`、`UpdateCustomer`
3. **命令应该包含执行操作所需的所有信息**
4. **使用验证逻辑确保命令的有效性**

```go
type CreateOrderCommand struct {
    OrderID    string
    CustomerID string
    Amount     float64
}

func (c *CreateOrderCommand) Validate() error {
    if c.OrderID == "" {
        return errors.New("order ID is required")
    }
    if c.Amount <= 0 {
        return errors.New("amount must be positive")
    }
    return nil
}
```

### 查询设计

1. **查询应该是只读的**，不应该改变系统状态
2. **查询名称应该使用名词或描述性短语**，如 `GetOrder`、`ListCustomers`
3. **查询应该返回 DTO（数据传输对象）**而不是领域对象
4. **考虑使用专门的读模型优化查询性能**

```go
type ListOrdersQuery struct {
    CustomerID string
    Status     string
    Page       int
    PageSize   int
}

type OrderListDTO struct {
    Orders     []OrderSummaryDTO
    TotalCount int
    Page       int
    PageSize   int
}
```

### 错误处理

```go
func (h *CreateOrderHandler) Handle(ctx context.Context, cmd cqrs.Command) error {
    orderCmd := cmd.(*CreateOrderCommand)
    
    // 业务验证
    if err := h.validateOrder(orderCmd); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // 执行业务逻辑
    if err := h.orderRepo.Save(ctx, order); err != nil {
        return fmt.Errorf("failed to save order: %w", err)
    }
    
    return nil
}
```

## 接口

### Command

```go
type Command interface {
    GetCommandName() string
}
```

### Query

```go
type Query interface {
    GetQueryName() string
}
```

### CommandBus

```go
type CommandBus interface {
    Dispatch(ctx context.Context, cmd Command) error
    RegisterHandler(handler CommandHandler) error
    UnregisterHandler(commandName string) error
    HasHandler(commandName string) bool
}
```

### QueryBus

```go
type QueryBus interface {
    Dispatch(ctx context.Context, query Query) (interface{}, error)
    RegisterHandler(handler QueryHandler) error
    UnregisterHandler(queryName string) error
    HasHandler(queryName string) bool
}
```

## 与其他模式集成

### Outbox Pattern

结合 Outbox Pattern 实现可靠的事件发布：

```go
func (h *CreateOrderHandler) Handle(ctx context.Context, cmd cqrs.Command) error {
    orderCmd := cmd.(*CreateOrderCommand)
    
    // 开始事务
    tx, err := h.storage.BeginTx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    
    // 保存订单
    if err := h.saveOrder(ctx, tx, orderCmd); err != nil {
        return err
    }
    
    // 保存事件到 outbox
    event := &outbox.OutboxEntry{
        AggregateID: orderCmd.OrderID,
        EventType:   "order.created",
        Topic:       "orders.events",
        Payload:     eventData,
    }
    
    if err := h.outboxPublisher.SaveWithTransaction(ctx, tx, event); err != nil {
        return err
    }
    
    // 提交事务
    return tx.Commit(ctx)
}
```

### Saga Pattern

结合 Saga Pattern 实现分布式事务：

```go
type OrderSagaHandler struct {
    commandBus cqrs.CommandBus
    sagaCoord  saga.SagaCoordinator
}

func (h *OrderSagaHandler) Handle(ctx context.Context, cmd cqrs.Command) error {
    orderCmd := cmd.(*CreateOrderCommand)
    
    // 定义 saga 步骤
    sagaDef := &saga.SagaDefinition{
        Name: "CreateOrderSaga",
        Steps: []saga.SagaStep{
            {
                Name: "CreateOrder",
                Handler: func(ctx context.Context, data any) (any, error) {
                    return h.createOrder(ctx, orderCmd)
                },
                Compensation: func(ctx context.Context, data any) error {
                    return h.cancelOrder(ctx, orderCmd.OrderID)
                },
            },
            {
                Name: "ReserveInventory",
                Handler: func(ctx context.Context, data any) (any, error) {
                    return h.reserveInventory(ctx, orderCmd)
                },
                Compensation: func(ctx context.Context, data any) error {
                    return h.releaseInventory(ctx, orderCmd.OrderID)
                },
            },
        },
    }
    
    // 执行 saga
    _, err := h.sagaCoord.StartSaga(ctx, sagaDef, orderCmd)
    return err
}
```

## 性能优化

### 批量处理

对于高频命令，可以实现批量处理：

```go
type BatchCommandBus struct {
    commandBus cqrs.CommandBus
    batchSize  int
    batchChan  chan cqrs.Command
}

func (b *BatchCommandBus) Start(ctx context.Context) {
    go func() {
        batch := make([]cqrs.Command, 0, b.batchSize)
        for {
            select {
            case cmd := <-b.batchChan:
                batch = append(batch, cmd)
                if len(batch) >= b.batchSize {
                    b.processBatch(ctx, batch)
                    batch = make([]cqrs.Command, 0, b.batchSize)
                }
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

### 查询缓存

对于频繁的查询，可以使用缓存：

```go
type CachedQueryHandler struct {
    handler QueryHandler
    cache   Cache
    ttl     time.Duration
}

func (h *CachedQueryHandler) Handle(ctx context.Context, query cqrs.Query) (interface{}, error) {
    // 尝试从缓存获取
    cacheKey := h.getCacheKey(query)
    if result, ok := h.cache.Get(cacheKey); ok {
        return result, nil
    }
    
    // 执行查询
    result, err := h.handler.Handle(ctx, query)
    if err != nil {
        return nil, err
    }
    
    // 缓存结果
    h.cache.Set(cacheKey, result, h.ttl)
    
    return result, nil
}
```

## 故障排查

### 常见问题

1. **处理器未注册**
   ```go
   if !commandBus.HasHandler("order.create") {
       log.Fatal("Handler not registered")
   }
   ```

2. **命令名称不匹配**
   - 确保命令的 `GetCommandName()` 返回值与处理器的 `GetCommandName()` 一致

3. **并发安全问题**
   - 总线实现是线程安全的，但处理器内部需要自行保证线程安全

## 测试

### 单元测试

```go
func TestCreateOrderHandler(t *testing.T) {
    // 创建 mock 依赖
    mockRepo := &MockOrderRepository{}
    handler := &CreateOrderHandler{orderRepo: mockRepo}
    
    // 创建命令
    cmd := &CreateOrderCommand{
        OrderID:    "order-001",
        CustomerID: "customer-123",
        Amount:     99.99,
    }
    
    // 执行测试
    err := handler.Handle(context.Background(), cmd)
    if err != nil {
        t.Fatalf("Handler failed: %v", err)
    }
    
    // 验证结果
    if !mockRepo.SaveCalled {
        t.Error("Save should be called")
    }
}
```

### 集成测试

```go
func TestCommandBusIntegration(t *testing.T) {
    // 创建真实的依赖
    db := setupTestDatabase(t)
    defer db.Close()
    
    commandBus := cqrs.NewCommandBus()
    handler := &CreateOrderHandler{orderRepo: NewOrderRepository(db)}
    
    commandBus.RegisterHandler(handler)
    
    // 执行集成测试
    cmd := &CreateOrderCommand{
        OrderID:    "order-001",
        CustomerID: "customer-123",
        Amount:     99.99,
    }
    
    err := commandBus.Dispatch(context.Background(), cmd)
    if err != nil {
        t.Fatalf("Dispatch failed: %v", err)
    }
    
    // 验证数据库状态
    order, err := NewOrderRepository(db).FindByID(context.Background(), "order-001")
    if err != nil {
        t.Fatalf("Failed to find order: %v", err)
    }
    
    if order.Status != "pending" {
        t.Errorf("Expected status 'pending', got '%s'", order.Status)
    }
}
```

## 参考资料

- [CQRS Pattern - Martin Fowler](https://martinfowler.com/bliki/CQRS.html)
- [Event Sourcing Pattern](https://martinfowler.com/eaaDev/EventSourcing.html)
- [Domain-Driven Design](https://www.domainlanguage.com/ddd/)

## 许可证

MIT License

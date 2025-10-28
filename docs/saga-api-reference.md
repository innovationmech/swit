# Saga 分布式事务 API 参考文档

本文档提供 Swit 框架 Saga 分布式事务系统的完整 API 参考。涵盖所有公开接口、配置选项、错误类型和使用示例。

## 目录

- [核心接口](#核心接口)
  - [SagaCoordinator](#sagacoordinator)
  - [SagaInstance](#sagainstance)
  - [SagaDefinition](#sagadefinition)
  - [SagaStep](#sagastep)
- [状态管理](#状态管理)
  - [StateStorage](#statestorage)
  - [存储实现](#存储实现)
- [事件系统](#事件系统)
  - [EventPublisher](#eventpublisher)
  - [EventFilter](#eventfilter)
  - [EventHandler](#eventhandler)
- [DSL 解析器](#dsl-解析器)
  - [Parser](#parser)
  - [SagaDefinition (DSL)](#sagadefinition-dsl)
- [策略与配置](#策略与配置)
  - [RetryPolicy](#retrypolicy)
  - [CompensationStrategy](#compensationstrategy)
  - [OrchestratorConfig](#orchestratorconfig)
- [类型与常量](#类型与常量)
  - [SagaState](#sagastate)
  - [StepStateEnum](#stepstateenum)
  - [ErrorType](#errortype)
  - [SagaEventType](#sagaeventtype)
- [错误处理](#错误处理)
  - [SagaError](#sagaerror)
  - [错误构造函数](#错误构造函数)
- [数据结构](#数据结构)
  - [SagaInstanceData](#sagainstancedata)
  - [StepState](#stepstate)
  - [SagaEvent](#sagaevent)
- [代码示例](#代码示例)

---

## 核心接口

### SagaCoordinator

`SagaCoordinator` 是 Saga 系统的核心协调器接口，负责管理 Saga 实例的生命周期、协调步骤执行、处理补偿操作和提供监控能力。

#### 包路径

```go
github.com/innovationmech/swit/pkg/saga
```

#### 接口定义

```go
type SagaCoordinator interface {
    StartSaga(ctx context.Context, definition SagaDefinition, initialData interface{}) (SagaInstance, error)
    GetSagaInstance(sagaID string) (SagaInstance, error)
    CancelSaga(ctx context.Context, sagaID string, reason string) error
    GetActiveSagas(filter *SagaFilter) ([]SagaInstance, error)
    GetMetrics() *CoordinatorMetrics
    HealthCheck(ctx context.Context) error
    Close() error
}
```

#### 方法说明

##### StartSaga

启动一个新的 Saga 实例。

**签名**
```go
StartSaga(ctx context.Context, definition SagaDefinition, initialData interface{}) (SagaInstance, error)
```

**参数**
- `ctx` (context.Context): 上下文，用于超时控制和取消操作
- `definition` (SagaDefinition): Saga 定义，包含步骤、重试策略等配置
- `initialData` (interface{}): 初始数据，将传递给第一个步骤

**返回值**
- `SagaInstance`: 创建的 Saga 实例
- `error`: 如果启动失败则返回错误

**错误码**
- `ErrCodeValidationError`: 定义或数据验证失败
- `ErrCodeStorageError`: 状态存储失败
- `ErrCodeCoordinatorStopped`: 协调器已停止

**示例**
```go
// 创建 Saga 定义
definition := &OrderSagaDefinition{
    id:   "order-processing",
    name: "订单处理",
    steps: []saga.SagaStep{
        NewCreateOrderStep(),
        NewReserveInventoryStep(),
        NewProcessPaymentStep(),
    },
    timeout: 30 * time.Minute,
}

// 启动 Saga
instance, err := coordinator.StartSaga(ctx, definition, &OrderData{
    UserID:    "user-123",
    ProductID: "prod-456",
    Quantity:  2,
})
if err != nil {
    log.Fatalf("Failed to start saga: %v", err)
}

fmt.Printf("Saga started with ID: %s\n", instance.GetID())
```

##### GetSagaInstance

根据 ID 获取 Saga 实例的当前状态。

**签名**
```go
GetSagaInstance(sagaID string) (SagaInstance, error)
```

**参数**
- `sagaID` (string): Saga 实例的唯一标识符

**返回值**
- `SagaInstance`: Saga 实例
- `error`: 如果未找到或查询失败则返回错误

**错误码**
- `ErrCodeSagaNotFound`: Saga 不存在
- `ErrCodeStorageError`: 存储查询失败

**示例**
```go
instance, err := coordinator.GetSagaInstance("saga-123")
if err != nil {
    if saga.IsSagaNotFound(err) {
        log.Printf("Saga not found: %s", sagaID)
    } else {
        log.Printf("Failed to get saga: %v", err)
    }
    return
}

fmt.Printf("Saga state: %s, progress: %d/%d\n", 
    instance.GetState().String(),
    instance.GetCompletedSteps(),
    instance.GetTotalSteps())
```

##### CancelSaga

取消正在运行的 Saga 实例，触发补偿操作。

**签名**
```go
CancelSaga(ctx context.Context, sagaID string, reason string) error
```

**参数**
- `ctx` (context.Context): 上下文
- `sagaID` (string): 要取消的 Saga ID
- `reason` (string): 取消原因，用于日志和审计

**返回值**
- `error`: 如果取消失败则返回错误

**错误码**
- `ErrCodeSagaNotFound`: Saga 不存在
- `ErrCodeInvalidSagaState`: Saga 状态不允许取消（已完成或已取消）
- `ErrCodeCompensationFailed`: 补偿操作失败

**示例**
```go
err := coordinator.CancelSaga(ctx, "saga-123", "User requested cancellation")
if err != nil {
    log.Printf("Failed to cancel saga: %v", err)
} else {
    log.Printf("Saga cancelled successfully")
}
```

##### GetActiveSagas

根据过滤条件获取活跃的 Saga 实例列表。

**签名**
```go
GetActiveSagas(filter *SagaFilter) ([]SagaInstance, error)
```

**参数**
- `filter` (*SagaFilter): 过滤条件，可以为 nil（返回所有活跃的 Saga）

**返回值**
- `[]SagaInstance`: Saga 实例列表
- `error`: 如果查询失败则返回错误

**示例**
```go
// 获取所有运行中的 Saga
filter := &saga.SagaFilter{
    States: []saga.SagaState{saga.StateRunning, saga.StateCompensating},
    Limit:  100,
}

instances, err := coordinator.GetActiveSagas(filter)
if err != nil {
    log.Printf("Failed to get active sagas: %v", err)
    return
}

fmt.Printf("Found %d active sagas\n", len(instances))
for _, instance := range instances {
    fmt.Printf("  - %s: %s\n", instance.GetID(), instance.GetState().String())
}
```

##### GetMetrics

获取协调器的运行时指标。

**签名**
```go
GetMetrics() *CoordinatorMetrics
```

**返回值**
- `*CoordinatorMetrics`: 协调器指标数据

**示例**
```go
metrics := coordinator.GetMetrics()
fmt.Printf("Coordinator Metrics:\n")
fmt.Printf("  Total Sagas: %d\n", metrics.TotalSagas)
fmt.Printf("  Active Sagas: %d\n", metrics.ActiveSagas)
fmt.Printf("  Completed Sagas: %d\n", metrics.CompletedSagas)
fmt.Printf("  Failed Sagas: %d\n", metrics.FailedSagas)
fmt.Printf("  Average Duration: %v\n", metrics.AverageSagaDuration)
```

##### HealthCheck

执行健康检查，验证协调器及其依赖项的健康状态。

**签名**
```go
HealthCheck(ctx context.Context) error
```

**参数**
- `ctx` (context.Context): 上下文，用于超时控制

**返回值**
- `error`: 如果不健康则返回错误

**示例**
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := coordinator.HealthCheck(ctx); err != nil {
    log.Printf("Coordinator is unhealthy: %v", err)
} else {
    log.Printf("Coordinator is healthy")
}
```

##### Close

优雅地关闭协调器，释放所有资源。

**签名**
```go
Close() error
```

**返回值**
- `error`: 如果关闭过程出错则返回错误

**示例**
```go
defer func() {
    if err := coordinator.Close(); err != nil {
        log.Printf("Failed to close coordinator: %v", err)
    }
}()
```

---

### SagaInstance

`SagaInstance` 表示一个正在运行的 Saga 实例，维护当前状态、执行进度，并提供访问执行结果和错误信息的方法。

#### 接口定义

```go
type SagaInstance interface {
    GetID() string
    GetDefinitionID() string
    GetState() SagaState
    GetCurrentStep() int
    GetStartTime() time.Time
    GetEndTime() time.Time
    GetResult() interface{}
    GetError() *SagaError
    GetTotalSteps() int
    GetCompletedSteps() int
    GetCreatedAt() time.Time
    GetUpdatedAt() time.Time
    GetTimeout() time.Duration
    GetMetadata() map[string]interface{}
    GetTraceID() string
    IsTerminal() bool
    IsActive() bool
}
```

#### 方法说明

##### GetID

返回 Saga 实例的唯一标识符。

**签名**
```go
GetID() string
```

**返回值**
- `string`: Saga 实例 ID

##### GetState

返回 Saga 实例的当前状态。

**签名**
```go
GetState() SagaState
```

**返回值**
- `SagaState`: 当前状态（Pending、Running、Completed、Failed 等）

**示例**
```go
state := instance.GetState()
switch state {
case saga.StateCompleted:
    fmt.Println("Saga completed successfully")
case saga.StateFailed:
    fmt.Printf("Saga failed: %v\n", instance.GetError())
case saga.StateRunning:
    fmt.Printf("Saga is running: %d/%d steps\n", 
        instance.GetCompletedSteps(), instance.GetTotalSteps())
}
```

##### GetCurrentStep

返回当前正在执行的步骤索引。

**签名**
```go
GetCurrentStep() int
```

**返回值**
- `int`: 当前步骤索引（从 0 开始），如果没有活跃步骤返回 -1

##### GetResult

返回 Saga 执行的最终结果数据。

**签名**
```go
GetResult() interface{}
```

**返回值**
- `interface{}`: 最终结果数据，如果 Saga 未成功完成则为 nil

**示例**
```go
if instance.GetState() == saga.StateCompleted {
    result := instance.GetResult()
    if orderResult, ok := result.(*OrderResult); ok {
        fmt.Printf("Order created: %s\n", orderResult.OrderID)
    }
}
```

##### GetError

返回导致 Saga 失败的错误。

**签名**
```go
GetError() *SagaError
```

**返回值**
- `*SagaError`: 错误详情，如果 Saga 未失败则为 nil

**示例**
```go
if instance.GetState() == saga.StateFailed {
    err := instance.GetError()
    fmt.Printf("Error: %s (Type: %s, Retryable: %v)\n",
        err.Message, err.Type, err.Retryable)
    
    // 打印错误链
    for _, chainErr := range err.GetChain() {
        fmt.Printf("  - %s\n", chainErr.Message)
    }
}
```

##### IsTerminal

判断 Saga 是否处于终止状态（不会再执行）。

**签名**
```go
IsTerminal() bool
```

**返回值**
- `bool`: 如果是终止状态返回 true

##### IsActive

判断 Saga 是否处于活跃状态（正在执行或补偿）。

**签名**
```go
IsActive() bool
```

**返回值**
- `bool`: 如果是活跃状态返回 true

---

### SagaDefinition

`SagaDefinition` 定义 Saga 的结构和行为，包含要执行的所有步骤、重试策略和补偿策略。

#### 接口定义

```go
type SagaDefinition interface {
    GetID() string
    GetName() string
    GetDescription() string
    GetSteps() []SagaStep
    GetTimeout() time.Duration
    GetRetryPolicy() RetryPolicy
    GetCompensationStrategy() CompensationStrategy
    Validate() error
    GetMetadata() map[string]interface{}
}
```

#### 方法说明

##### GetID

返回 Saga 定义的唯一标识符。

**签名**
```go
GetID() string
```

**返回值**
- `string`: Saga 定义 ID

##### GetSteps

返回所有执行步骤（按顺序）。

**签名**
```go
GetSteps() []SagaStep
```

**返回值**
- `[]SagaStep`: 步骤列表

##### GetRetryPolicy

返回失败步骤的重试策略。

**签名**
```go
GetRetryPolicy() RetryPolicy
```

**返回值**
- `RetryPolicy`: 重试策略实例

##### GetCompensationStrategy

返回失败 Saga 的补偿策略。

**签名**
```go
GetCompensationStrategy() CompensationStrategy
```

**返回值**
- `CompensationStrategy`: 补偿策略实例

##### Validate

验证定义的正确性和一致性。

**签名**
```go
Validate() error
```

**返回值**
- `error`: 如果定义无效则返回错误

**示例**
```go
definition := NewOrderSagaDefinition()
if err := definition.Validate(); err != nil {
    log.Fatalf("Invalid saga definition: %v", err)
}
```

---

### SagaStep

`SagaStep` 定义 Saga 中的单个执行步骤，每个步骤都有执行动作和对应的补偿动作。

#### 接口定义

```go
type SagaStep interface {
    GetID() string
    GetName() string
    GetDescription() string
    Execute(ctx context.Context, data interface{}) (interface{}, error)
    Compensate(ctx context.Context, data interface{}) error
    GetTimeout() time.Duration
    GetRetryPolicy() RetryPolicy
    IsRetryable(err error) bool
    GetMetadata() map[string]interface{}
}
```

#### 方法说明

##### Execute

执行步骤的主要逻辑。

**签名**
```go
Execute(ctx context.Context, data interface{}) (interface{}, error)
```

**参数**
- `ctx` (context.Context): 上下文，包含超时和取消信号
- `data` (interface{}): 输入数据，来自上一步骤的输出或初始数据

**返回值**
- `interface{}`: 输出数据，将传递给下一步骤
- `error`: 如果执行失败则返回错误

**示例**
```go
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    // 调用订单服务
    order, err := s.orderService.CreateOrder(ctx, &OrderRequest{
        UserID:    orderData.UserID,
        ProductID: orderData.ProductID,
        Quantity:  orderData.Quantity,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create order: %w", err)
    }
    
    // 更新数据并返回
    orderData.OrderID = order.ID
    return orderData, nil
}
```

##### Compensate

执行补偿动作以撤销步骤的效果。

**签名**
```go
Compensate(ctx context.Context, data interface{}) error
```

**参数**
- `ctx` (context.Context): 上下文
- `data` (interface{}): 执行时的数据，包含需要补偿的信息

**返回值**
- `error`: 如果补偿失败则返回错误

**示例**
```go
func (s *CreateOrderStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    // 取消订单
    if err := s.orderService.CancelOrder(ctx, orderData.OrderID); err != nil {
        return fmt.Errorf("failed to compensate order: %w", err)
    }
    
    return nil
}
```

##### IsRetryable

判断错误是否可重试。

**签名**
```go
IsRetryable(err error) bool
```

**参数**
- `err` (error): 发生的错误

**返回值**
- `bool`: 如果错误可重试返回 true

**示例**
```go
func (s *CreateOrderStep) IsRetryable(err error) bool {
    // 网络错误和超时错误可以重试
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }
    if errors.Is(err, ErrNetworkFailure) {
        return true
    }
    // 业务错误不重试
    if errors.Is(err, ErrInvalidOrder) {
        return false
    }
    return true
}
```

---

## 状态管理

### StateStorage

`StateStorage` 定义了持久化 Saga 状态的接口。实现可以提供内存、Redis、数据库或其他存储后端。

#### 接口定义

```go
type StateStorage interface {
    SaveSaga(ctx context.Context, saga SagaInstance) error
    GetSaga(ctx context.Context, sagaID string) (SagaInstance, error)
    UpdateSagaState(ctx context.Context, sagaID string, state SagaState, metadata map[string]interface{}) error
    DeleteSaga(ctx context.Context, sagaID string) error
    GetActiveSagas(ctx context.Context, filter *SagaFilter) ([]SagaInstance, error)
    GetTimeoutSagas(ctx context.Context, before time.Time) ([]SagaInstance, error)
    SaveStepState(ctx context.Context, sagaID string, step *StepState) error
    GetStepStates(ctx context.Context, sagaID string) ([]*StepState, error)
    CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error
}
```

#### 方法说明

##### SaveSaga

持久化 Saga 实例到存储。

**签名**
```go
SaveSaga(ctx context.Context, saga SagaInstance) error
```

**参数**
- `ctx` (context.Context): 上下文
- `saga` (SagaInstance): 要保存的 Saga 实例

**返回值**
- `error`: 如果保存失败则返回错误

##### GetSaga

根据 ID 检索 Saga 实例。

**签名**
```go
GetSaga(ctx context.Context, sagaID string) (SagaInstance, error)
```

**参数**
- `ctx` (context.Context): 上下文
- `sagaID` (string): Saga ID

**返回值**
- `SagaInstance`: Saga 实例
- `error`: 如果不存在或查询失败则返回错误

##### UpdateSagaState

仅更新 Saga 实例的状态。

**签名**
```go
UpdateSagaState(ctx context.Context, sagaID string, state SagaState, metadata map[string]interface{}) error
```

**参数**
- `ctx` (context.Context): 上下文
- `sagaID` (string): Saga ID
- `state` (SagaState): 新状态
- `metadata` (map[string]interface{}): 可选的元数据更新

**返回值**
- `error`: 如果更新失败则返回错误

##### GetTimeoutSagas

检索在指定时间之前超时的 Saga 实例。

**签名**
```go
GetTimeoutSagas(ctx context.Context, before time.Time) ([]SagaInstance, error)
```

**参数**
- `ctx` (context.Context): 上下文
- `before` (time.Time): 时间阈值

**返回值**
- `[]SagaInstance`: 超时的 Saga 实例列表
- `error`: 如果查询失败则返回错误

---

### 存储实现

#### MemoryStateStorage

内存存储实现，适用于开发、测试和非关键工作负载。

**包路径**
```go
github.com/innovationmech/swit/pkg/saga/state/storage
```

**创建实例**
```go
// 使用默认配置
storage := storage.NewMemoryStateStorage()

// 使用自定义配置
storage := storage.NewMemoryStateStorageWithConfig(&state.MemoryStorageConfig{
    InitialCapacity: 1000,
    MaxCapacity:     10000,
    EnableMetrics:   true,
})
```

**特点**
- 线程安全
- 支持并发访问
- 无持久化（重启后数据丢失）
- 性能高

#### PostgresStateStorage

PostgreSQL 存储实现，提供持久化和高可用性。

**创建实例**
```go
storage, err := storage.NewPostgresStateStorage(&storage.PostgresConfig{
    Host:     "localhost",
    Port:     5432,
    Database: "saga_db",
    Username: "saga_user",
    Password: "saga_pass",
    SSLMode:  "disable",
    MaxConnections:    20,
    MaxIdleConnections: 5,
    ConnectionTimeout:  10 * time.Second,
})
if err != nil {
    log.Fatalf("Failed to create postgres storage: %v", err)
}
defer storage.Close()
```

**特点**
- 持久化存储
- 支持事务
- 索引优化
- 支持复杂查询
- 高可用性

#### RedisStateStorage

Redis 存储实现，提供高性能和分布式支持。

**创建实例**
```go
storage, err := storage.NewRedisStateStorage(&storage.RedisConfig{
    Endpoints: []string{"localhost:6379"},
    Password:  "",
    DB:        0,
    PoolSize:  10,
    EnableCluster: false,
    KeyPrefix:     "saga:",
    DefaultExpiration: 24 * time.Hour,
})
if err != nil {
    log.Fatalf("Failed to create redis storage: %v", err)
}
defer storage.Close()
```

**特点**
- 高性能
- 支持集群模式
- 自动过期
- 适合分布式环境

---

## 事件系统

### EventPublisher

`EventPublisher` 定义了发布 Saga 相关事件的接口，支持事件驱动架构和外部监控。

#### 接口定义

```go
type EventPublisher interface {
    PublishEvent(ctx context.Context, event *SagaEvent) error
    Subscribe(filter EventFilter, handler EventHandler) (EventSubscription, error)
    Unsubscribe(subscription EventSubscription) error
    Close() error
}
```

#### 方法说明

##### PublishEvent

将 Saga 事件发布到配置的消息代理。

**签名**
```go
PublishEvent(ctx context.Context, event *SagaEvent) error
```

**参数**
- `ctx` (context.Context): 上下文
- `event` (*SagaEvent): 要发布的事件

**返回值**
- `error`: 如果发布失败则返回错误

##### Subscribe

订阅匹配过滤器的 Saga 事件。

**签名**
```go
Subscribe(filter EventFilter, handler EventHandler) (EventSubscription, error)
```

**参数**
- `filter` (EventFilter): 事件过滤器
- `handler` (EventHandler): 事件处理器

**返回值**
- `EventSubscription`: 订阅句柄，可用于取消订阅
- `error`: 如果订阅失败则返回错误

**示例**
```go
// 订阅所有 Saga 完成事件
filter := &saga.EventTypeFilter{
    Types: []saga.SagaEventType{
        saga.EventSagaCompleted,
        saga.EventSagaFailed,
    },
}

subscription, err := publisher.Subscribe(filter, &MyEventHandler{})
if err != nil {
    log.Fatalf("Failed to subscribe: %v", err)
}
defer publisher.Unsubscribe(subscription)
```

---

### EventFilter

`EventFilter` 定义过滤 Saga 事件的标准，允许订阅者只接收感兴趣的事件。

#### 接口定义

```go
type EventFilter interface {
    Match(event *SagaEvent) bool
    GetDescription() string
}
```

#### 内置过滤器

##### EventTypeFilter

根据事件类型过滤。

```go
type EventTypeFilter struct {
    Types         []SagaEventType
    ExcludedTypes []SagaEventType
}
```

**示例**
```go
// 只接收步骤相关事件
filter := &saga.EventTypeFilter{
    Types: []saga.SagaEventType{
        saga.EventSagaStepStarted,
        saga.EventSagaStepCompleted,
        saga.EventSagaStepFailed,
    },
}
```

##### SagaIDFilter

根据 Saga ID 过滤。

```go
type SagaIDFilter struct {
    SagaIDs        []string
    ExcludeSagaIDs []string
}
```

**示例**
```go
// 只接收特定 Saga 的事件
filter := &saga.SagaIDFilter{
    SagaIDs: []string{"saga-123", "saga-456"},
}
```

##### CompositeFilter

组合多个过滤器（AND/OR 操作）。

```go
type CompositeFilter struct {
    Filters   []EventFilter
    Operation FilterOperation // "AND" or "OR"
}
```

**示例**
```go
// 组合过滤器：特定 Saga 的完成事件
filter := &saga.CompositeFilter{
    Operation: saga.FilterOperationAND,
    Filters: []saga.EventFilter{
        &saga.SagaIDFilter{SagaIDs: []string{"saga-123"}},
        &saga.EventTypeFilter{Types: []saga.SagaEventType{saga.EventSagaCompleted}},
    },
}
```

---

### EventHandler

`EventHandler` 定义处理 Saga 事件的接口。

#### 接口定义

```go
type EventHandler interface {
    HandleEvent(ctx context.Context, event *SagaEvent) error
    GetHandlerName() string
}
```

#### 实现示例

```go
type MyEventHandler struct {
    logger *zap.Logger
}

func (h *MyEventHandler) HandleEvent(ctx context.Context, event *saga.SagaEvent) error {
    h.logger.Info("Received saga event",
        zap.String("saga_id", event.SagaID),
        zap.String("type", string(event.Type)),
        zap.Time("timestamp", event.Timestamp))
    
    switch event.Type {
    case saga.EventSagaCompleted:
        // 处理完成事件
        return h.handleCompletion(ctx, event)
    case saga.EventSagaFailed:
        // 处理失败事件
        return h.handleFailure(ctx, event)
    default:
        return nil
    }
}

func (h *MyEventHandler) GetHandlerName() string {
    return "MyEventHandler"
}
```

---

## DSL 解析器

### Parser

`Parser` 处理 Saga DSL YAML 文件的解析，支持环境变量替换和文件包含。

#### 包路径

```go
github.com/innovationmech/swit/pkg/saga/dsl
```

#### 创建解析器

```go
// 使用默认选项
parser := dsl.NewParser()

// 使用自定义选项
parser := dsl.NewParser(
    dsl.WithEnvVars(true),
    dsl.WithIncludes(true),
    dsl.WithMaxIncludeDepth(10),
    dsl.WithBasePath("/path/to/sagas"),
    dsl.WithLogger(customLogger),
)
```

#### 解析方法

##### ParseFile

从 YAML 文件解析 Saga 定义。

**签名**
```go
ParseFile(path string) (*SagaDefinition, error)
```

**参数**
- `path` (string): YAML 文件路径

**返回值**
- `*SagaDefinition`: 解析的 Saga 定义
- `error`: 如果解析失败则返回错误

**示例**
```go
definition, err := parser.ParseFile("sagas/order-processing.yaml")
if err != nil {
    log.Fatalf("Failed to parse saga: %v", err)
}

fmt.Printf("Loaded saga: %s (%d steps)\n", 
    definition.Saga.Name, len(definition.Steps))
```

##### ParseBytes

从字节数组解析 Saga 定义。

**签名**
```go
ParseBytes(data []byte) (*SagaDefinition, error)
```

##### ParseReader

从 io.Reader 解析 Saga 定义。

**签名**
```go
ParseReader(r io.Reader) (*SagaDefinition, error)
```

---

### SagaDefinition (DSL)

DSL 格式的 Saga 定义数据结构。

#### 结构定义

```go
type SagaDefinition struct {
    Saga                SagaConfig
    Steps               []StepConfig
    GlobalRetryPolicy   *RetryPolicy
    GlobalCompensation  *CompensationConfig
}
```

#### YAML 示例

```yaml
saga:
  id: order-processing
  name: 订单处理流程
  description: 处理新订单的完整流程
  version: "1.0.0"
  timeout: 30m
  mode: orchestration
  tags:
    - orders
    - payments

global_retry_policy:
  type: exponential_backoff
  max_attempts: 3
  base_delay: 1s
  max_delay: 30s

global_compensation:
  strategy: sequential
  timeout: 5m

steps:
  - id: create-order
    name: 创建订单
    description: 在订单服务中创建新订单
    type: grpc
    timeout: 10s
    action:
      service: order-service
      method: CreateOrder
      input_mapping:
        user_id: "{{ .UserID }}"
        product_id: "{{ .ProductID }}"
        quantity: "{{ .Quantity }}"
    compensation:
      service: order-service
      method: CancelOrder
      input_mapping:
        order_id: "{{ .OrderID }}"
    retry_policy:
      type: fixed_delay
      max_attempts: 5
      delay: 2s

  - id: reserve-inventory
    name: 预留库存
    type: grpc
    timeout: 10s
    action:
      service: inventory-service
      method: ReserveItems
      input_mapping:
        product_id: "{{ .ProductID }}"
        quantity: "{{ .Quantity }}"
        order_id: "{{ .OrderID }}"
    compensation:
      service: inventory-service
      method: ReleaseReservation
      input_mapping:
        reservation_id: "{{ .ReservationID }}"

  - id: process-payment
    name: 处理支付
    type: grpc
    timeout: 30s
    dependencies:
      - create-order
      - reserve-inventory
    action:
      service: payment-service
      method: ProcessPayment
      input_mapping:
        order_id: "{{ .OrderID }}"
        user_id: "{{ .UserID }}"
        amount: "{{ .TotalAmount }}"
    compensation:
      service: payment-service
      method: RefundPayment
      input_mapping:
        payment_id: "{{ .PaymentID }}"
```

---

## 策略与配置

### RetryPolicy

`RetryPolicy` 定义失败操作的重试策略。

#### 接口定义

```go
type RetryPolicy interface {
    ShouldRetry(err error, attempt int) bool
    GetRetryDelay(attempt int) time.Duration
    GetMaxAttempts() int
}
```

#### 内置实现

##### FixedDelayRetryPolicy

固定延迟重试策略。

```go
policy := saga.NewFixedDelayRetryPolicy(
    maxAttempts: 3,
    delay: 2 * time.Second,
)
```

##### ExponentialBackoffRetryPolicy

指数退避重试策略（带抖动）。

```go
policy := saga.NewExponentialBackoffRetryPolicy(
    maxAttempts: 5,
    baseDelay:   1 * time.Second,
    maxDelay:    30 * time.Second,
)

// 自定义配置
policy.SetJitter(true)
policy.SetMultiplier(2.0)
```

##### LinearBackoffRetryPolicy

线性退避重试策略。

```go
policy := saga.NewLinearBackoffRetryPolicy(
    maxAttempts: 4,
    baseDelay:   1 * time.Second,
    increment:   2 * time.Second,
    maxDelay:    10 * time.Second,
)
```

##### NoRetryPolicy

不重试策略。

```go
policy := saga.NewNoRetryPolicy()
```

---

### CompensationStrategy

`CompensationStrategy` 定义 Saga 失败时如何执行补偿。

#### 接口定义

```go
type CompensationStrategy interface {
    ShouldCompensate(err *SagaError) bool
    GetCompensationOrder(completedSteps []SagaStep) []SagaStep
    GetCompensationTimeout() time.Duration
}
```

#### 内置实现

##### SequentialCompensationStrategy

顺序补偿策略（反向顺序）。

```go
strategy := saga.NewSequentialCompensationStrategy(
    timeout: 30 * time.Second,
)
```

##### ParallelCompensationStrategy

并行补偿策略。

```go
strategy := saga.NewParallelCompensationStrategy(
    timeout: 30 * time.Second,
)
```

##### BestEffortCompensationStrategy

尽力补偿策略（即使部分失败也继续）。

```go
strategy := saga.NewBestEffortCompensationStrategy(
    timeout: 60 * time.Second,
)
```

##### CustomCompensationStrategy

自定义补偿策略。

```go
strategy := saga.NewCustomCompensationStrategy(
    timeout: 45 * time.Second,
    shouldCompensateFunc: func(err *saga.SagaError) bool {
        // 自定义逻辑：只对特定错误类型补偿
        return err.Type != saga.ErrorTypeBusiness
    },
    getOrderFunc: func(completedSteps []saga.SagaStep) []saga.SagaStep {
        // 自定义顺序逻辑
        // 例如：按优先级排序
        return customOrderLogic(completedSteps)
    },
)
```

---

### OrchestratorConfig

编排协调器的配置选项。

#### 结构定义

```go
type OrchestratorConfig struct {
    StateStorage           saga.StateStorage         // 必需
    EventPublisher         saga.EventPublisher       // 必需
    RetryPolicy            saga.RetryPolicy          // 可选，默认指数退避
    MetricsCollector       MetricsCollector          // 可选
    TracingManager         TracingManager            // 可选
    TimeoutDetector        *TimeoutDetectorConfig    // 可选
    ConcurrencyController  *ConcurrencyConfig        // 可选
}
```

#### 创建协调器示例

```go
import (
    "github.com/innovationmech/swit/pkg/saga/coordinator"
    "github.com/innovationmech/swit/pkg/saga/state/storage"
    "github.com/innovationmech/swit/pkg/saga/messaging"
)

// 创建存储
stateStorage := storage.NewMemoryStateStorage()

// 创建事件发布器
publisher, err := messaging.NewSagaEventPublisher(&messaging.PublisherConfig{
    BrokerType:      "nats",
    BrokerEndpoints: []string{"nats://localhost:4222"},
    TopicPrefix:     "saga",
    SerializerType:  "json",
    EnableConfirm:   true,
    RetryAttempts:   3,
    RetryInterval:   time.Second,
    Timeout:         30 * time.Second,
})
if err != nil {
    log.Fatalf("Failed to create publisher: %v", err)
}

// 创建协调器
coord, err := coordinator.NewOrchestratorCoordinator(&coordinator.OrchestratorConfig{
    StateStorage:   stateStorage,
    EventPublisher: publisher,
    RetryPolicy:    saga.NewExponentialBackoffRetryPolicy(5, time.Second, 30*time.Second),
    TimeoutDetector: &coordinator.TimeoutDetectorConfig{
        CheckInterval:    10 * time.Second,
        EnableAutoRecover: true,
    },
    ConcurrencyController: &coordinator.ConcurrencyConfig{
        MaxConcurrentSagas: 100,
        MaxConcurrentSteps: 10,
    },
})
if err != nil {
    log.Fatalf("Failed to create coordinator: %v", err)
}
defer coord.Close()
```

---

## 类型与常量

### SagaState

Saga 实例的整体状态。

#### 类型定义

```go
type SagaState int
```

#### 常量

```go
const (
    StatePending      SagaState = iota  // Saga 已创建但未启动
    StateRunning                        // Saga 正在执行
    StateStepCompleted                  // 步骤已完成，移动到下一步
    StateCompleted                      // Saga 成功完成
    StateCompensating                   // Saga 正在执行补偿操作
    StateCompensated                    // 所有补偿操作已完成
    StateFailed                         // Saga 因错误失败
    StateCancelled                      // Saga 被用户请求取消
    StateTimedOut                       // Saga 超过超时限制
)
```

#### 方法

```go
// String 返回状态的字符串表示
func (s SagaState) String() string

// IsTerminal 判断是否为终止状态
func (s SagaState) IsTerminal() bool

// IsActive 判断是否为活跃状态
func (s SagaState) IsActive() bool
```

---

### StepStateEnum

单个步骤的执行状态。

#### 类型定义

```go
type StepStateEnum int
```

#### 常量

```go
const (
    StepStatePending      StepStateEnum = iota  // 步骤等待执行
    StepStateRunning                            // 步骤正在执行
    StepStateCompleted                          // 步骤成功完成
    StepStateFailed                             // 步骤失败，可能会重试
    StepStateCompensating                       // 步骤的补偿正在执行
    StepStateCompensated                        // 步骤的补偿已完成
    StepStateSkipped                            // 步骤被跳过（条件逻辑）
)
```

---

### ErrorType

错误类别。

#### 类型定义

```go
type ErrorType string
```

#### 常量

```go
const (
    ErrorTypeValidation   ErrorType = "validation"    // 验证错误
    ErrorTypeTimeout      ErrorType = "timeout"       // 超时错误
    ErrorTypeNetwork      ErrorType = "network"       // 网络错误
    ErrorTypeService      ErrorType = "service"       // 服务错误
    ErrorTypeData         ErrorType = "data"          // 数据错误
    ErrorTypeSystem       ErrorType = "system"        // 系统错误
    ErrorTypeBusiness     ErrorType = "business"      // 业务错误
    ErrorTypeCompensation ErrorType = "compensation"  // 补偿错误
)
```

---

### SagaEventType

Saga 事件类型。

#### 类型定义

```go
type SagaEventType string
```

#### 常量

```go
const (
    // Saga 生命周期事件
    EventSagaStarted       SagaEventType = "saga.started"
    EventSagaStepStarted   SagaEventType = "saga.step.started"
    EventSagaStepCompleted SagaEventType = "saga.step.completed"
    EventSagaStepFailed    SagaEventType = "saga.step.failed"
    EventSagaCompleted     SagaEventType = "saga.completed"
    EventSagaFailed        SagaEventType = "saga.failed"
    EventSagaCancelled     SagaEventType = "saga.cancelled"
    EventSagaTimedOut      SagaEventType = "saga.timed_out"
    
    // 补偿事件
    EventCompensationStarted       SagaEventType = "compensation.started"
    EventCompensationStepStarted   SagaEventType = "compensation.step.started"
    EventCompensationStepCompleted SagaEventType = "compensation.step.completed"
    EventCompensationStepFailed    SagaEventType = "compensation.step.failed"
    EventCompensationCompleted     SagaEventType = "compensation.completed"
    EventCompensationFailed        SagaEventType = "compensation.failed"
    
    // 重试事件
    EventRetryAttempted SagaEventType = "retry.attempted"
    EventRetryExhausted SagaEventType = "retry.exhausted"
    
    // 状态变更事件
    EventStateChanged SagaEventType = "state.changed"
    
    // 死信事件
    EventDeadLettered SagaEventType = "dead.lettered"
)
```

---

## 错误处理

### SagaError

Saga 执行期间发生的错误表示。

#### 结构定义

```go
type SagaError struct {
    Code       string                 // 错误代码
    Message    string                 // 错误消息
    Type       ErrorType              // 错误类型
    Retryable  bool                   // 是否可重试
    Timestamp  time.Time              // 发生时间
    StackTrace string                 // 堆栈跟踪
    Details    map[string]interface{} // 附加详情
    Cause      *SagaError             // 原因错误（错误链）
}
```

#### 方法

##### Error

实现 error 接口。

```go
func (e *SagaError) Error() string
```

##### WithStackTrace

添加堆栈跟踪。

```go
func (e *SagaError) WithStackTrace() *SagaError
```

##### WithDetail

添加单个详情。

```go
func (e *SagaError) WithDetail(key string, value interface{}) *SagaError
```

##### WithDetails

添加多个详情。

```go
func (e *SagaError) WithDetails(details map[string]interface{}) *SagaError
```

##### GetChain

返回错误链。

```go
func (e *SagaError) GetChain() []*SagaError
```

##### IsRetryable

检查错误或其原因是否可重试。

```go
func (e *SagaError) IsRetryable() bool
```

##### GetRootCause

返回错误链的根原因。

```go
func (e *SagaError) GetRootCause() *SagaError
```

---

### 错误构造函数

#### 预定义错误代码

```go
const (
    ErrCodeSagaNotFound        = "SAGA_NOT_FOUND"
    ErrCodeSagaAlreadyExists   = "SAGA_ALREADY_EXISTS"
    ErrCodeInvalidSagaState    = "INVALID_SAGA_STATE"
    ErrCodeSagaTimeout         = "SAGA_TIMEOUT"
    ErrCodeStepExecutionFailed = "STEP_EXECUTION_FAILED"
    ErrCodeCompensationFailed  = "COMPENSATION_FAILED"
    ErrCodeStorageError        = "STORAGE_ERROR"
    ErrCodeValidationError     = "VALIDATION_ERROR"
    ErrCodeConfigurationError  = "CONFIGURATION_ERROR"
    ErrCodeCoordinatorStopped  = "COORDINATOR_STOPPED"
    ErrCodeRetryExhausted      = "RETRY_EXHAUSTED"
    ErrCodeEventPublishFailed  = "EVENT_PUBLISH_FAILED"
)
```

#### NewSagaError

创建新的 SagaError。

```go
func NewSagaError(code, message string, errorType ErrorType, retryable bool) *SagaError
```

#### WrapError

将现有错误包装为 SagaError。

```go
func WrapError(err error, code, message string, errorType ErrorType, retryable bool) *SagaError
```

#### 特定错误构造函数

```go
// Saga 未找到
func NewSagaNotFoundError(sagaID string) *SagaError

// Saga 已存在
func NewSagaAlreadyExistsError(sagaID string) *SagaError

// 无效状态转换
func NewInvalidSagaStateError(currentState, targetState SagaState) *SagaError

// Saga 超时
func NewSagaTimeoutError(sagaID string, timeout time.Duration) *SagaError

// 步骤执行失败
func NewStepExecutionError(stepID, stepName string, err error) *SagaError

// 补偿失败
func NewCompensationFailedError(stepID, stepName string, err error) *SagaError

// 存储错误
func NewStorageError(operation string, err error) *SagaError

// 验证错误
func NewValidationError(message string) *SagaError

// 配置错误
func NewConfigurationError(message string) *SagaError

// 协调器已停止
func NewCoordinatorStoppedError() *SagaError

// 重试耗尽
func NewRetryExhaustedError(operation string, attempts int) *SagaError

// 事件发布失败
func NewEventPublishError(eventType SagaEventType, err error) *SagaError
```

#### 错误检查函数

```go
// 检查是否为 Saga 未找到错误
func IsSagaNotFound(err error) bool

// 检查是否为超时错误
func IsSagaTimeout(err error) bool

// 检查是否为步骤执行失败
func IsStepExecutionFailed(err error) bool

// 检查是否为补偿失败
func IsCompensationFailed(err error) bool

// 检查是否为重试耗尽
func IsRetryExhausted(err error) bool

// 检查是否为可重试错误
func IsRetryableError(err error) bool
```

---

## 数据结构

### SagaInstanceData

Saga 实例的完整数据结构。

```go
type SagaInstanceData struct {
    // 基本信息
    ID           string
    DefinitionID string
    Name         string
    Description  string
    
    // 状态信息
    State       SagaState
    CurrentStep int
    TotalSteps  int
    
    // 时间信息
    CreatedAt   time.Time
    UpdatedAt   time.Time
    StartedAt   *time.Time
    CompletedAt *time.Time
    TimedOutAt  *time.Time
    
    // 数据和错误信息
    InitialData interface{}
    CurrentData interface{}
    ResultData  interface{}
    Error       *SagaError
    
    // 配置
    Timeout     time.Duration
    RetryPolicy RetryPolicy
    
    // 步骤状态
    StepStates []*StepState
    
    // 元数据
    Metadata map[string]interface{}
    
    // 追踪信息
    TraceID string
    SpanID  string
    
    // 乐观锁
    Version int
}
```

---

### StepState

步骤的状态信息。

```go
type StepState struct {
    // 基本信息
    ID        string
    SagaID    string
    StepIndex int
    Name      string
    
    // 状态信息
    State       StepStateEnum
    Attempts    int
    MaxAttempts int
    
    // 时间信息
    CreatedAt     time.Time
    StartedAt     *time.Time
    CompletedAt   *time.Time
    LastAttemptAt *time.Time
    
    // 数据和错误信息
    InputData  interface{}
    OutputData interface{}
    Error      *SagaError
    
    // 补偿信息
    CompensationState *CompensationState
    
    // 元数据
    Metadata map[string]interface{}
}
```

---

### SagaEvent

Saga 执行期间发生的事件。

```go
type SagaEvent struct {
    // 基本标识
    ID      string
    SagaID  string
    StepID  string
    Type    SagaEventType
    Version string
    
    // 时间信息
    Timestamp     time.Time
    CorrelationID string
    
    // 数据内容
    Data          interface{}
    PreviousState interface{}
    NewState      interface{}
    
    // 错误信息
    Error *SagaError
    
    // 执行信息
    Duration    time.Duration
    Attempt     int
    MaxAttempts int
    
    // 元数据
    Metadata map[string]interface{}
    
    // 来源信息
    Source         string
    Service        string
    ServiceVersion string
    
    // 追踪信息
    TraceID      string
    SpanID       string
    ParentSpanID string
}
```

---

## 代码示例

### 完整示例：订单处理 Saga

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/innovationmech/swit/pkg/saga"
    "github.com/innovationmech/swit/pkg/saga/coordinator"
    "github.com/innovationmech/swit/pkg/saga/state/storage"
    "github.com/innovationmech/swit/pkg/saga/messaging"
)

// ====== 数据结构 ======

type OrderData struct {
    UserID         string
    ProductID      string
    Quantity       int
    OrderID        string
    ReservationID  string
    PaymentID      string
    TotalAmount    float64
}

// ====== 步骤实现 ======

// 步骤 1: 创建订单
type CreateOrderStep struct {
    orderService *OrderService
}

func (s *CreateOrderStep) GetID() string { return "create-order" }
func (s *CreateOrderStep) GetName() string { return "创建订单" }
func (s *CreateOrderStep) GetDescription() string { return "在订单服务中创建新订单" }
func (s *CreateOrderStep) GetTimeout() time.Duration { return 10 * time.Second }
func (s *CreateOrderStep) GetRetryPolicy() saga.RetryPolicy { return nil }
func (s *CreateOrderStep) IsRetryable(err error) bool { return true }
func (s *CreateOrderStep) GetMetadata() map[string]interface{} { return nil }

func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    log.Printf("Creating order for user %s, product %s", orderData.UserID, orderData.ProductID)
    
    order, err := s.orderService.CreateOrder(ctx, orderData)
    if err != nil {
        return nil, fmt.Errorf("failed to create order: %w", err)
    }
    
    orderData.OrderID = order.ID
    orderData.TotalAmount = order.Amount
    
    log.Printf("Order created: %s (amount: %.2f)", order.ID, order.Amount)
    return orderData, nil
}

func (s *CreateOrderStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    log.Printf("Cancelling order: %s", orderData.OrderID)
    
    if err := s.orderService.CancelOrder(ctx, orderData.OrderID); err != nil {
        return fmt.Errorf("failed to cancel order: %w", err)
    }
    
    log.Printf("Order cancelled: %s", orderData.OrderID)
    return nil
}

// 步骤 2: 预留库存
type ReserveInventoryStep struct {
    inventoryService *InventoryService
}

func (s *ReserveInventoryStep) GetID() string { return "reserve-inventory" }
func (s *ReserveInventoryStep) GetName() string { return "预留库存" }
func (s *ReserveInventoryStep) GetDescription() string { return "预留产品库存" }
func (s *ReserveInventoryStep) GetTimeout() time.Duration { return 10 * time.Second }
func (s *ReserveInventoryStep) GetRetryPolicy() saga.RetryPolicy { return nil }
func (s *ReserveInventoryStep) IsRetryable(err error) bool { return true }
func (s *ReserveInventoryStep) GetMetadata() map[string]interface{} { return nil }

func (s *ReserveInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    log.Printf("Reserving %d units of product %s", orderData.Quantity, orderData.ProductID)
    
    reservation, err := s.inventoryService.Reserve(ctx, orderData.ProductID, orderData.Quantity)
    if err != nil {
        return nil, fmt.Errorf("failed to reserve inventory: %w", err)
    }
    
    orderData.ReservationID = reservation.ID
    
    log.Printf("Inventory reserved: %s", reservation.ID)
    return orderData, nil
}

func (s *ReserveInventoryStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    log.Printf("Releasing inventory reservation: %s", orderData.ReservationID)
    
    if err := s.inventoryService.Release(ctx, orderData.ReservationID); err != nil {
        return fmt.Errorf("failed to release inventory: %w", err)
    }
    
    log.Printf("Inventory released: %s", orderData.ReservationID)
    return nil
}

// 步骤 3: 处理支付
type ProcessPaymentStep struct {
    paymentService *PaymentService
}

func (s *ProcessPaymentStep) GetID() string { return "process-payment" }
func (s *ProcessPaymentStep) GetName() string { return "处理支付" }
func (s *ProcessPaymentStep) GetDescription() string { return "处理订单支付" }
func (s *ProcessPaymentStep) GetTimeout() time.Duration { return 30 * time.Second }
func (s *ProcessPaymentStep) GetRetryPolicy() saga.RetryPolicy { 
    return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}
func (s *ProcessPaymentStep) IsRetryable(err error) bool { return true }
func (s *ProcessPaymentStep) GetMetadata() map[string]interface{} { return nil }

func (s *ProcessPaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    orderData := data.(*OrderData)
    
    log.Printf("Processing payment for order %s (amount: %.2f)", orderData.OrderID, orderData.TotalAmount)
    
    payment, err := s.paymentService.Process(ctx, orderData.OrderID, orderData.UserID, orderData.TotalAmount)
    if err != nil {
        return nil, fmt.Errorf("failed to process payment: %w", err)
    }
    
    orderData.PaymentID = payment.ID
    
    log.Printf("Payment processed: %s", payment.ID)
    return orderData, nil
}

func (s *ProcessPaymentStep) Compensate(ctx context.Context, data interface{}) error {
    orderData := data.(*OrderData)
    
    log.Printf("Refunding payment: %s", orderData.PaymentID)
    
    if err := s.paymentService.Refund(ctx, orderData.PaymentID); err != nil {
        return fmt.Errorf("failed to refund payment: %w", err)
    }
    
    log.Printf("Payment refunded: %s", orderData.PaymentID)
    return nil
}

// ====== Saga 定义 ======

type OrderSagaDefinition struct {
    id          string
    name        string
    description string
    steps       []saga.SagaStep
    timeout     time.Duration
    retryPolicy saga.RetryPolicy
    strategy    saga.CompensationStrategy
}

func NewOrderSagaDefinition(
    orderService *OrderService,
    inventoryService *InventoryService,
    paymentService *PaymentService,
) *OrderSagaDefinition {
    return &OrderSagaDefinition{
        id:          "order-processing",
        name:        "订单处理",
        description: "处理新订单的完整流程",
        steps: []saga.SagaStep{
            &CreateOrderStep{orderService: orderService},
            &ReserveInventoryStep{inventoryService: inventoryService},
            &ProcessPaymentStep{paymentService: paymentService},
        },
        timeout:     30 * time.Minute,
        retryPolicy: saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
        strategy:    saga.NewSequentialCompensationStrategy(30 * time.Second),
    }
}

func (d *OrderSagaDefinition) GetID() string { return d.id }
func (d *OrderSagaDefinition) GetName() string { return d.name }
func (d *OrderSagaDefinition) GetDescription() string { return d.description }
func (d *OrderSagaDefinition) GetSteps() []saga.SagaStep { return d.steps }
func (d *OrderSagaDefinition) GetTimeout() time.Duration { return d.timeout }
func (d *OrderSagaDefinition) GetRetryPolicy() saga.RetryPolicy { return d.retryPolicy }
func (d *OrderSagaDefinition) GetCompensationStrategy() saga.CompensationStrategy { return d.strategy }
func (d *OrderSagaDefinition) GetMetadata() map[string]interface{} { return nil }

func (d *OrderSagaDefinition) Validate() error {
    if len(d.steps) == 0 {
        return fmt.Errorf("saga must have at least one step")
    }
    return nil
}

// ====== 主程序 ======

func main() {
    // 1. 创建存储
    stateStorage := storage.NewMemoryStateStorage()
    
    // 2. 创建事件发布器
    publisher, err := messaging.NewSagaEventPublisher(&messaging.PublisherConfig{
        BrokerType:      "nats",
        BrokerEndpoints: []string{"nats://localhost:4222"},
        TopicPrefix:     "saga",
        SerializerType:  "json",
        EnableConfirm:   true,
        RetryAttempts:   3,
        RetryInterval:   time.Second,
        Timeout:         30 * time.Second,
    })
    if err != nil {
        log.Fatalf("Failed to create publisher: %v", err)
    }
    defer publisher.Close()
    
    // 3. 创建协调器
    coord, err := coordinator.NewOrchestratorCoordinator(&coordinator.OrchestratorConfig{
        StateStorage:   stateStorage,
        EventPublisher: publisher,
        RetryPolicy:    saga.NewExponentialBackoffRetryPolicy(5, time.Second, 30*time.Second),
    })
    if err != nil {
        log.Fatalf("Failed to create coordinator: %v", err)
    }
    defer coord.Close()
    
    // 4. 创建服务实例（模拟）
    orderService := &OrderService{}
    inventoryService := &InventoryService{}
    paymentService := &PaymentService{}
    
    // 5. 创建 Saga 定义
    definition := NewOrderSagaDefinition(orderService, inventoryService, paymentService)
    
    // 6. 启动 Saga
    ctx := context.Background()
    instance, err := coord.StartSaga(ctx, definition, &OrderData{
        UserID:    "user-123",
        ProductID: "prod-456",
        Quantity:  2,
    })
    if err != nil {
        log.Fatalf("Failed to start saga: %v", err)
    }
    
    log.Printf("Saga started: %s", instance.GetID())
    
    // 7. 等待完成（实际应用中可以异步处理）
    sagaID := instance.GetID()
    for {
        time.Sleep(1 * time.Second)
        
        instance, err := coord.GetSagaInstance(sagaID)
        if err != nil {
            log.Fatalf("Failed to get saga: %v", err)
        }
        
        state := instance.GetState()
        log.Printf("Saga %s state: %s (%d/%d steps)", 
            sagaID, state.String(), instance.GetCompletedSteps(), instance.GetTotalSteps())
        
        if state.IsTerminal() {
            if state == saga.StateCompleted {
                result := instance.GetResult()
                orderData := result.(*OrderData)
                log.Printf("Saga completed successfully!")
                log.Printf("  Order ID: %s", orderData.OrderID)
                log.Printf("  Payment ID: %s", orderData.PaymentID)
            } else if state == saga.StateFailed {
                sagaErr := instance.GetError()
                log.Printf("Saga failed: %v", sagaErr.Error())
            }
            break
        }
    }
    
    // 8. 打印指标
    metrics := coord.GetMetrics()
    log.Printf("Coordinator Metrics:")
    log.Printf("  Total Sagas: %d", metrics.TotalSagas)
    log.Printf("  Completed Sagas: %d", metrics.CompletedSagas)
    log.Printf("  Failed Sagas: %d", metrics.FailedSagas)
}
```

---

## 相关文档

- [Saga 用户指南](saga-user-guide.md) - 快速开始和使用教程
- [Saga DSL 参考](saga-dsl-reference.md) - DSL 语法和示例
- [Saga 测试指南](saga-testing-guide.md) - 测试最佳实践
- [Saga 监控指南](saga-monitoring-guide.md) - 监控和可观测性
- [Saga 安全指南](saga-security-guide.md) - 安全配置和最佳实践

---

## 许可证

本文档和相关代码遵循 MIT 许可证。详见 [LICENSE](../LICENSE) 文件。


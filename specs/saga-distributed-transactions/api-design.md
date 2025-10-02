# Saga 分布式事务 API 设计文档

## 概述

本文档定义了 Swit 框架中 Saga 分布式事务系统的 API 设计，包括核心接口、数据结构、事件格式和使用模式。

## 核心接口设计

### 1. SagaCoordinator 接口

```go
// SagaCoordinator 是 Saga 系统的核心协调器
type SagaCoordinator interface {
    // 启动一个新的 Saga 实例
    StartSaga(ctx context.Context, definition *SagaDefinition, initialData interface{}) (*SagaInstance, error)
    
    // 获取 Saga 实例的当前状态
    GetSagaInstance(sagaID string) (*SagaInstance, error)
    
    // 取消正在执行的 Saga
    CancelSaga(ctx context.Context, sagaID string, reason string) error
    
    // 获取所有活跃的 Saga 实例
    GetActiveSagas(filter *SagaFilter) ([]*SagaInstance, error)
    
    // 获取协调器运行指标
    GetMetrics() *CoordinatorMetrics
    
    // 执行健康检查
    HealthCheck(ctx context.Context) error
    
    // 关闭协调器，释放资源
    Close() error
}
```

### 2. SagaDefinition 接口

```go
// SagaDefinition 定义了 Saga 的结构和行为
type SagaDefinition interface {
    // 获取 Saga 定义标识符
    GetID() string
    
    // 获取 Saga 名称
    GetName() string
    
    // 获取 Saga 描述
    GetDescription() string
    
    // 获取所有执行步骤
    GetSteps() []SagaStep
    
    // 获取超时时间
    GetTimeout() time.Duration
    
    // 获取重试策略
    GetRetryPolicy() RetryPolicy
    
    // 获取补偿策略
    GetCompensationStrategy() CompensationStrategy
    
    // 验证定义的有效性
    Validate() error
    
    // 获取元数据
    GetMetadata() map[string]interface{}
}
```

### 3. SagaStep 接口

```go
// SagaStep 定义了 Saga 中的单个执行步骤
type SagaStep interface {
    // 获取步骤标识符
    GetID() string
    
    // 获取步骤名称
    GetName() string
    
    // 获取步骤描述
    GetDescription() string
    
    // 执行步骤的主要逻辑
    Execute(ctx context.Context, data interface{}) (interface{}, error)
    
    // 执行补偿操作
    Compensate(ctx context.Context, data interface{}) error
    
    // 获取步骤超时时间
    GetTimeout() time.Duration
    
    // 获取重试策略
    GetRetryPolicy() RetryPolicy
    
    // 判断错误是否可重试
    IsRetryable(err error) bool
    
    // 获取步骤元数据
    GetMetadata() map[string]interface{}
}
```

### 4. StateStorage 接口

```go
// StateStorage 定义了 Saga 状态存储的接口
type StateStorage interface {
    // 保存 Saga 实例
    SaveSaga(ctx context.Context, saga *SagaInstance) error
    
    // 获取 Saga 实例
    GetSaga(ctx context.Context, sagaID string) (*SagaInstance, error)
    
    // 更新 Saga 实例状态
    UpdateSagaState(ctx context.Context, sagaID string, state SagaState, metadata map[string]interface{}) error
    
    // 删除 Saga 实例
    DeleteSaga(ctx context.Context, sagaID string) error
    
    // 查询活跃的 Saga 实例
    GetActiveSagas(ctx context.Context, filter *SagaFilter) ([]*SagaInstance, error)
    
    // 获取超时的 Saga 实例
    GetTimeoutSagas(ctx context.Context, before time.Time) ([]*SagaInstance, error)
    
    // 保存步骤状态
    SaveStepState(ctx context.Context, sagaID string, step *StepState) error
    
    // 获取步骤状态
    GetStepStates(ctx context.Context, sagaID string) ([]*StepState, error)
    
    // 清理过期的 Saga 实例
    CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error
}
```

## 数据结构设计

### 1. SagaInstance 结构

```go
// SagaInstance 表示一个运行中的 Saga 实例
type SagaInstance struct {
    // 基本信息
    ID          string                 `json:"id"`
    DefinitionID string                `json:"definition_id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    
    // 状态信息
    State       SagaState              `json:"state"`
    CurrentStep int                    `json:"current_step"`
    TotalSteps  int                    `json:"total_steps"`
    
    // 时间信息
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    StartedAt   *time.Time             `json:"started_at,omitempty"`
    CompletedAt *time.Time             `json:"completed_at,omitempty"`
    TimedOutAt  *time.Time             `json:"timed_out_at,omitempty"`
    
    // 数据和错误
    InitialData interface{}            `json:"initial_data,omitempty"`
    CurrentData interface{}            `json:"current_data,omitempty"`
    ResultData  interface{}            `json:"result_data,omitempty"`
    Error       *SagaError             `json:"error,omitempty"`
    
    // 配置
    Timeout     time.Duration          `json:"timeout"`
    RetryPolicy RetryPolicy            `json:"retry_policy"`
    
    // 步骤状态
    StepStates  []*StepState           `json:"step_states,omitempty"`
    
    // 元数据
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    
    // 追踪信息
    TraceID     string                 `json:"trace_id,omitempty"`
    SpanID      string                 `json:"span_id,omitempty"`
}
```

### 2. StepState 结构

```go
// StepState 表示单个步骤的执行状态
type StepState struct {
    // 基本信息
    ID          string                 `json:"id"`
    SagaID      string                 `json:"saga_id"`
    StepIndex   int                    `json:"step_index"`
    Name        string                 `json:"name"`
    
    // 状态信息
    State       StepStateEnum          `json:"state"`
    Attempts    int                    `json:"attempts"`
    MaxAttempts int                    `json:"max_attempts"`
    
    // 时间信息
    CreatedAt   time.Time              `json:"created_at"`
    StartedAt   *time.Time             `json:"started_at,omitempty"`
    CompletedAt *time.Time             `json:"completed_at,omitempty"`
    LastAttemptAt *time.Time           `json:"last_attempt_at,omitempty"`
    
    // 数据和错误
    InputData   interface{}            `json:"input_data,omitempty"`
    OutputData  interface{}            `json:"output_data,omitempty"`
    Error       *SagaError             `json:"error,omitempty"`
    
    // 补偿信息
    CompensationState *CompensationState `json:"compensation_state,omitempty"`
    
    // 元数据
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

### 3. CompensationState 结构

```go
// CompensationState 表示补偿操作的执行状态
type CompensationState struct {
    State       CompensationStateEnum  `json:"state"`
    Attempts    int                    `json:"attempts"`
    MaxAttempts int                    `json:"max_attempts"`
    StartedAt   *time.Time             `json:"started_at,omitempty"`
    CompletedAt *time.Time             `json:"completed_at,omitempty"`
    Error       *SagaError             `json:"error,omitempty"`
}
```

### 4. SagaError 结构

```go
// SagaError 表示 Saga 执行过程中的错误
type SagaError struct {
    Code        string                 `json:"code"`
    Message     string                 `json:"message"`
    Type        ErrorType              `json:"type"`
    Retryable   bool                   `json:"retryable"`
    Timestamp   time.Time              `json:"timestamp"`
    StackTrace  string                 `json:"stack_trace,omitempty"`
    Details     map[string]interface{} `json:"details,omitempty"`
    Cause       *SagaError             `json:"cause,omitempty"`
}
```

## 枚举类型定义

### 1. SagaState 枚举

```go
// SagaState 表示 Saga 的整体状态
type SagaState int

const (
    StatePending SagaState = iota
    StateRunning
    StateStepCompleted
    StateCompleted
    StateCompensating
    StateCompensated
    StateFailed
    StateCancelled
    StateTimedOut
)

func (s SagaState) String() string {
    switch s {
    case StatePending:
        return "pending"
    case StateRunning:
        return "running"
    case StateStepCompleted:
        return "step_completed"
    case StateCompleted:
        return "completed"
    case StateCompensating:
        return "compensating"
    case StateCompensated:
        return "compensated"
    case StateFailed:
        return "failed"
    case StateCancelled:
        return "cancelled"
    case StateTimedOut:
        return "timed_out"
    default:
        return "unknown"
    }
}

func (s SagaState) IsTerminal() bool {
    return s == StateCompleted || s == StateCompensated || s == StateFailed || s == StateCancelled || s == StateTimedOut
}

func (s SagaState) IsActive() bool {
    return s == StateRunning || s == StateStepCompleted || s == StateCompensating
}
```

### 2. StepStateEnum 枚举

```go
// StepStateEnum 表示步骤的执行状态
type StepStateEnum int

const (
    StepStatePending StepStateEnum = iota
    StepStateRunning
    StepStateCompleted
    StepStateFailed
    StepStateCompensating
    StepStateCompensated
    StepStateSkipped
)

func (s StepStateEnum) String() string {
    switch s {
    case StepStatePending:
        return "pending"
    case StepStateRunning:
        return "running"
    case StepStateCompleted:
        return "completed"
    case StepStateFailed:
        return "failed"
    case StepStateCompensating:
        return "compensating"
    case StepStateCompensated:
        return "compensated"
    case StepStateSkipped:
        return "skipped"
    default:
        return "unknown"
    }
}
```

### 3. CompensationStateEnum 枚举

```go
// CompensationStateEnum 表示补偿操作的状态
type CompensationStateEnum int

const (
    CompensationStatePending CompensationStateEnum = iota
    CompensationStateRunning
    CompensationStateCompleted
    CompensationStateFailed
    CompensationStateSkipped
)

func (c CompensationStateEnum) String() string {
    switch c {
    case CompensationStatePending:
        return "pending"
    case CompensationStateRunning:
        return "running"
    case CompensationStateCompleted:
        return "completed"
    case CompensationStateFailed:
        return "failed"
    case CompensationStateSkipped:
        return "skipped"
    default:
        return "unknown"
    }
}
```

## 事件系统设计

### 1. SagaEventType 枚举

```go
// SagaEventType 表示 Saga 事件的类型
type SagaEventType string

const (
    // Saga 生命周期事件
    EventSagaStarted         SagaEventType = "saga.started"
    EventSagaStepStarted     SagaEventType = "saga.step.started"
    EventSagaStepCompleted   SagaEventType = "saga.step.completed"
    EventSagaStepFailed      SagaEventType = "saga.step.failed"
    EventSagaCompleted       SagaEventType = "saga.completed"
    EventSagaFailed          SagaEventType = "saga.failed"
    EventSagaCancelled       SagaEventType = "saga.cancelled"
    EventSagaTimedOut        SagaEventType = "saga.timed_out"
    
    // 补偿事件
    EventCompensationStarted SagaEventType = "compensation.started"
    EventCompensationStepStarted SagaEventType = "compensation.step.started"
    EventCompensationStepCompleted SagaEventType = "compensation.step.completed"
    EventCompensationStepFailed SagaEventType = "compensation.step.failed"
    EventCompensationCompleted SagaEventType = "compensation.completed"
    EventCompensationFailed    SagaEventType = "compensation.failed"
    
    // 重试事件
    EventRetryAttempted       SagaEventType = "retry.attempted"
    EventRetryExhausted       SagaEventType = "retry.exhausted"
    
    // 状态变更事件
    EventStateChanged         SagaEventType = "state.changed"
)
```

### 2. SagaEvent 结构

```go
// SagaEvent 表示 Saga 系统中的事件
type SagaEvent struct {
    // 基本标识信息
    ID          string                 `json:"id"`
    SagaID      string                 `json:"saga_id"`
    StepID      string                 `json:"step_id,omitempty"`
    Type        SagaEventType          `json:"type"`
    Version     string                 `json:"version"`
    
    // 时间信息
    Timestamp   time.Time              `json:"timestamp"`
    CorrelationID string               `json:"correlation_id,omitempty"`
    
    // 数据内容
    Data        interface{}            `json:"data,omitempty"`
    PreviousState interface{}          `json:"previous_state,omitempty"`
    NewState    interface{}            `json:"new_state,omitempty"`
    
    // 错误信息
    Error       *SagaError             `json:"error,omitempty"`
    
    // 执行信息
    Duration    time.Duration          `json:"duration,omitempty"`
    Attempt     int                    `json:"attempt,omitempty"`
    MaxAttempts int                    `json:"max_attempts,omitempty"`
    
    // 元数据
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    
    // 来源信息
    Source      string                 `json:"source,omitempty"`
    Service     string                 `json:"service,omitempty"`
    Version     string                 `json:"service_version,omitempty"`
    
    // 追踪信息
    TraceID     string                 `json:"trace_id,omitempty"`
    SpanID      string                 `json:"span_id,omitempty"`
    ParentSpanID string                `json:"parent_span_id,omitempty"`
}
```

## 配置结构设计

### 1. SagaConfig 结构

```go
// SagaConfig 包含 Saga 系统的全局配置
type SagaConfig struct {
    // 协调器配置
    Coordinator CoordinatorConfig `yaml:"coordinator" json:"coordinator"`
    
    // 存储配置
    Storage     StorageConfig     `yaml:"storage" json:"storage"`
    
    // 消息配置
    Messaging   MessagingConfig   `yaml:"messaging" json:"messaging"`
    
    // 监控配置
    Monitoring  MonitoringConfig  `yaml:"monitoring" json:"monitoring"`
    
    // 安全配置
    Security    SecurityConfig    `yaml:"security" json:"security"`
    
    // 日志配置
    Logging     LoggingConfig     `yaml:"logging" json:"logging"`
}
```

### 2. CoordinatorConfig 结构

```go
// CoordinatorConfig 包含协调器的配置
type CoordinatorConfig struct {
    // 模式配置
    Mode                CoordinatorMode `yaml:"mode" json:"mode"`
    
    // 并发配置
    MaxConcurrentSagas  int             `yaml:"max_concurrent_sagas" json:"max_concurrent_sagas"`
    MaxConcurrentSteps  int             `yaml:"max_concurrent_steps" json:"max_concurrent_steps"`
    
    // 超时配置
    DefaultTimeout      time.Duration   `yaml:"default_timeout" json:"default_timeout"`
    StepTimeout         time.Duration   `yaml:"step_timeout" json:"step_timeout"`
    
    // 重试配置
    DefaultRetryPolicy  *RetryPolicy    `yaml:"default_retry_policy" json:"default_retry_policy"`
    
    // 清理配置
    CleanupInterval     time.Duration   `yaml:"cleanup_interval" json:"cleanup_interval"`
    RetentionPeriod     time.Duration   `yaml:"retention_period" json:"retention_period"`
    
    // 恢复配置
    RecoveryEnabled     bool            `yaml:"recovery_enabled" json:"recovery_enabled"`
    RecoveryInterval    time.Duration   `yaml:"recovery_interval" json:"recovery_interval"`
}
```

### 3. StorageConfig 结构

```go
// StorageConfig 包含存储系统的配置
type StorageConfig struct {
    // 存储类型
    Type        StorageType    `yaml:"type" json:"type"`
    
    // 连接配置
    Connection  interface{}     `yaml:"connection" json:"connection"`
    
    // Redis 特定配置
    RedisConfig *RedisConfig   `yaml:"redis,omitempty" json:"redis,omitempty"`
    
    // PostgreSQL 特定配置
    PostgresConfig *PostgresConfig `yaml:"postgres,omitempty" json:"postgres,omitempty"`
    
    // 通用配置
    KeyPrefix   string          `yaml:"key_prefix" json:"key_prefix"`
    TTL         time.Duration   `yaml:"ttl" json:"ttl"`
    MaxRetries  int             `yaml:"max_retries" json:"max_retries"`
}
```

## 使用示例

### 1. 基本使用模式

```go
// 创建 Saga 定义
definition := &StandardSagaDefinition{
    ID:          "order-processing",
    Name:        "Order Processing",
    Description: "Processes customer orders with inventory and payment",
    Steps: []SagaStep{
        &CreateOrderStep{Timeout: 10 * time.Second},
        &ReserveInventoryStep{Timeout: 15 * time.Second},
        &ProcessPaymentStep{Timeout: 20 * time.Second},
        &ConfirmOrderStep{Timeout: 5 * time.Second},
    },
    Timeout: 30 * time.Minute,
    RetryPolicy: &ExponentialBackoffPolicy{
        InitialDelay: 1 * time.Second,
        MaxDelay:     30 * time.Second,
        Multiplier:   2.0,
        MaxAttempts:  5,
    },
}

// 创建协调器
config := &SagaConfig{
    Coordinator: CoordinatorConfig{
        Mode:               OrchestrationMode,
        MaxConcurrentSagas: 1000,
        DefaultTimeout:     30 * time.Minute,
    },
    Storage: StorageConfig{
        Type: RedisStorage,
        RedisConfig: &RedisConfig{
            Address: "localhost:6379",
            DB:      0,
        },
    },
}

coordinator, err := NewOrchestratorCoordinator(config)
if err != nil {
    return err
}
defer coordinator.Close()

// 启动 Saga
orderData := &OrderData{
    CustomerID: "customer-123",
    Items: []OrderItem{
        {ProductID: "product-1", Quantity: 2},
        {ProductID: "product-2", Quantity: 1},
    },
}

instance, err := coordinator.StartSaga(ctx, definition, orderData)
if err != nil {
    return err
}

// 监控 Saga 状态
for {
    saga, err := coordinator.GetSagaInstance(instance.ID)
    if err != nil {
        return err
    }
    
    if saga.State.IsTerminal() {
        break
    }
    
    time.Sleep(1 * time.Second)
}
```

### 2. 事件驱动模式

```go
// 创建事件驱动的协调器
choreography := &ChoreographyCoordinator{
    EventPublisher:  publisher,
    EventSubscriber: subscriber,
    StateStorage:    storage,
}

// 注册事件处理器
handler := &OrderEventHandler{
    OrderService:    orderService,
    InventoryService: inventoryService,
    PaymentService:  paymentService,
}

choreography.RegisterEventHandler("order.created", handler.HandleOrderCreated)
choreography.RegisterEventHandler("inventory.reserved", handler.HandleInventoryReserved)
choreography.RegisterEventHandler("payment.processed", handler.HandlePaymentProcessed)

// 启动协调器
if err := choreography.Start(ctx); err != nil {
    return err
}
```

### 3. 混合模式

```go
// 创建混合模式协调器
hybrid := &HybridCoordinator{
    Orchestrator:  orchestrator,
    Choreography: choreography,
    ModeSelector:  &SmartModeSelector{},
}

// 根据 Saga 特性自动选择模式
instance, err := hybrid.StartSaga(ctx, definition, data)
if err != nil {
    return err
}
```

## 错误处理

### 1. 错误类型定义

```go
// ErrorType 表示错误的类型
type ErrorType string

const (
    ErrorTypeValidation     ErrorType = "validation"
    ErrorTypeTimeout        ErrorType = "timeout"
    ErrorTypeNetwork        ErrorType = "network"
    ErrorTypeService        ErrorType = "service"
    ErrorTypeData           ErrorType = "data"
    ErrorTypeSystem         ErrorType = "system"
    ErrorTypeBusiness       ErrorType = "business"
    ErrorTypeCompensation   ErrorType = "compensation"
)
```

### 2. 错误处理策略

```go
// ErrorHandlingStrategy 定义了错误处理的策略
type ErrorHandlingStrategy interface {
    ShouldRetry(err *SagaError, attempt int) bool
    GetRetryDelay(attempt int) time.Duration
    ShouldCompensate(err *SagaError) bool
    GetTimeoutForError(err *SagaError) time.Duration
}
```

## 性能考虑

### 1. 批量操作

```go
// BatchOperation 表示批量操作
type BatchOperation struct {
    SagaIDs     []string    `json:"saga_ids"`
    Operation   BatchType   `json:"operation"`
    Parameters  interface{} `json:"parameters"`
}

// BatchType 表示批量操作类型
type BatchType string

const (
    BatchTypeCancel     BatchType = "cancel"
    BatchTypeRetry      BatchType = "retry"
    BatchTypeForceCompensation BatchType = "force_compensation"
)
```

### 2. 缓存策略

```go
// CacheConfig 定义了缓存配置
type CacheConfig struct {
    Enabled     bool          `yaml:"enabled" json:"enabled"`
    TTL         time.Duration `yaml:"ttl" json:"ttl"`
    MaxSize     int           `yaml:"max_size" json:"max_size"`
    Policy      CachePolicy   `yaml:"policy" json:"policy"`
}
```

---

**API 设计版本**: 1.0  
**创建日期**: 2025-01-01  
**最后更新**: 2025-01-01  
**负责人**: Swit 开发团队
# Saga 分布式事务开发者文档

## 概述

本文档面向希望深入了解 Swit 框架 Saga 分布式事务系统内部实现的开发者。涵盖架构设计、核心组件、扩展机制、测试策略和贡献指南。

## 目录

- [架构设计](#架构设计)
  - [整体架构](#整体架构)
  - [核心组件](#核心组件)
  - [设计原则](#设计原则)
- [核心组件详解](#核心组件详解)
  - [SagaCoordinator](#sagacoordinator)
  - [状态机](#状态机)
  - [执行引擎](#执行引擎)
  - [补偿机制](#补偿机制)
- [扩展和定制](#扩展和定制)
  - [自定义步骤](#自定义步骤)
  - [自定义存储适配器](#自定义存储适配器)
  - [自定义事件发布器](#自定义事件发布器)
  - [自定义重试策略](#自定义重试策略)
- [内部工作流程](#内部工作流程)
  - [Saga 启动流程](#saga-启动流程)
  - [步骤执行流程](#步骤执行流程)
  - [补偿流程](#补偿流程)
  - [状态恢复流程](#状态恢复流程)
- [测试策略](#测试策略)
  - [单元测试](#单元测试)
  - [集成测试](#集成测试)
  - [端到端测试](#端到端测试)
- [开发环境设置](#开发环境设置)
- [贡献指南](#贡献指南)

---

## 架构设计

### 整体架构

Swit Saga 系统采用分层架构设计，从上到下分为以下几层：

```
┌─────────────────────────────────────────────────────────────┐
│                     应用层 (Application)                     │
│         Saga Definition、Step Implementation              │
└─────────────────────────────────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                   协调层 (Coordination)                      │
│    OrchestratorCoordinator | ChoreographyCoordinator      │
│           HybridCoordinator                                │
└─────────────────────────────────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                   执行层 (Execution)                         │
│     StepExecutor | CompensationExecutor                    │
│     ErrorHandler | TimeoutDetector                         │
└─────────────────────────────────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                   状态层 (State Management)                  │
│    StateStorage | StateManager | Recovery                  │
└─────────────────────────────────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                  基础设施层 (Infrastructure)                 │
│   Messaging | Monitoring | Security | Persistence          │
└─────────────────────────────────────────────────────────────┘
```

### 核心组件

#### 1. Coordinator（协调器）

协调器是 Saga 系统的核心，负责管理 Saga 实例的整个生命周期。

**主要职责**：
- Saga 实例的创建和初始化
- 步骤的顺序执行和协调
- 错误处理和补偿逻辑触发
- 状态持久化和恢复
- 事件发布和监控

**实现类型**：

1. **OrchestratorCoordinator**（编排式协调器）
   - 集中式控制
   - 明确的流程编排
   - 便于追踪和调试
   - 适合复杂业务流程

2. **ChoreographyCoordinator**（协作式协调器）
   - 去中心化协调
   - 事件驱动
   - 松耦合
   - 适合服务间协作

3. **HybridCoordinator**（混合协调器）
   - 结合两种模式的优点
   - 灵活的协调策略
   - 适合复杂场景

#### 2. StateStorage（状态存储）

持久化 Saga 状态，确保故障恢复能力。

**接口定义**：
```go
type StateStorage interface {
    SaveSaga(ctx context.Context, saga SagaInstance) error
    GetSaga(ctx context.Context, sagaID string) (SagaInstance, error)
    UpdateSagaState(ctx context.Context, sagaID string, state SagaState, metadata map[string]interface{}) error
    DeleteSaga(ctx context.Context, sagaID string) error
    GetActiveSagas(ctx context.Context, filter *SagaFilter) ([]SagaInstance, error)
    GetTimeoutSagas(ctx context.Context, before time.Time) ([]SagaInstance, error)
}
```

**内置实现**：
- **MemoryStateStorage**: 内存存储，适合开发和测试
- **RedisStateStorage**: Redis 存储，高性能
- **PostgresStateStorage**: PostgreSQL 存储，强一致性

#### 3. EventPublisher（事件发布器）

发布 Saga 生命周期事件，支持监控和外部系统集成。

**事件类型**：
- Saga 启动/完成/失败事件
- 步骤启动/完成/失败事件
- 补偿启动/完成/失败事件
- 状态变更事件

#### 4. RetryPolicy（重试策略）

定义失败操作的重试行为。

**内置策略**：
- **FixedDelayRetryPolicy**: 固定延迟重试
- **ExponentialBackoffRetryPolicy**: 指数退避重试
- **LinearBackoffRetryPolicy**: 线性退避重试
- **NoRetryPolicy**: 不重试

### 设计原则

1. **接口驱动**: 所有组件基于接口设计，易于测试和扩展
2. **关注点分离**: 清晰的职责划分，降低耦合度
3. **可观测性**: 内置监控、追踪和日志支持
4. **容错性**: 完善的错误处理和恢复机制
5. **性能优化**: 并发控制、批处理和资源池化

---

## 核心组件详解

### SagaCoordinator

#### 结构定义

```go
type OrchestratorCoordinator struct {
    stateStorage           StateStorage
    eventPublisher         EventPublisher
    retryPolicy            RetryPolicy
    metricsCollector       MetricsCollector
    tracingManager         TracingManager
    timeoutDetector        *TimeoutDetector
    concurrencyController  *ConcurrencyController
    instances              sync.Map
    metrics                *CoordinatorMetrics
    closed                 bool
    mu                     sync.RWMutex
}
```

#### 核心方法实现

##### StartSaga

启动一个新的 Saga 实例。

**实现流程**：

1. **验证阶段**
   - 检查协调器状态
   - 验证 Saga 定义
   - 验证初始数据

2. **创建实例**
   - 生成唯一 Saga ID
   - 创建 SagaInstance 对象
   - 初始化状态为 Pending

3. **持久化**
   - 保存初始状态到存储
   - 记录创建时间戳

4. **启动执行**
   - 发布 SagaStarted 事件
   - 启动异步执行goroutine
   - 返回 SagaInstance

**代码示例**：

```go
func (oc *OrchestratorCoordinator) StartSaga(
    ctx context.Context,
    definition saga.SagaDefinition,
    initialData interface{},
) (saga.SagaInstance, error) {
    // 开始追踪
    ctx, span := oc.tracingManager.StartSpan(ctx, "saga.start")
    defer span.End()

    // 验证协调器状态
    oc.mu.RLock()
    if oc.closed {
        oc.mu.RUnlock()
        return nil, ErrCoordinatorClosed
    }
    oc.mu.RUnlock()

    // 验证定义
    if definition == nil || definition.Validate() != nil {
        return nil, ErrInvalidDefinition
    }

    // 生成 Saga ID
    sagaID, err := generateSagaID()
    if err != nil {
        return nil, err
    }

    // 创建实例
    instance := &OrchestratorSagaInstance{
        id:           sagaID,
        definitionID: definition.GetID(),
        state:        saga.StatePending,
        currentData:  initialData,
        steps:        definition.GetSteps(),
        totalSteps:   len(definition.GetSteps()),
        createdAt:    time.Now(),
        timeout:      definition.GetTimeout(),
    }

    // 持久化初始状态
    if err := oc.stateStorage.SaveSaga(ctx, instance); err != nil {
        return nil, fmt.Errorf("failed to save saga: %w", err)
    }

    // 发布启动事件
    oc.eventPublisher.PublishEvent(ctx, &saga.SagaEvent{
        Type:   saga.EventSagaStarted,
        SagaID: sagaID,
        Timestamp: time.Now(),
    })

    // 异步执行步骤
    go oc.executeSteps(context.Background(), instance, definition)

    return instance, nil
}
```

### 状态机

#### 状态定义

```go
type SagaState int

const (
    StatePending      SagaState = iota  // 已创建,未启动
    StateRunning                        // 正在执行
    StateStepCompleted                  // 步骤完成
    StateCompleted                      // 成功完成
    StateCompensating                   // 正在补偿
    StateCompensated                    // 补偿完成
    StateFailed                         // 失败
    StateCancelled                      // 已取消
    StateTimedOut                       // 超时
)
```

#### 状态转换规则

```go
var validTransitions = map[SagaState][]SagaState{
    StatePending: {
        StateRunning,
        StateCancelled,
    },
    StateRunning: {
        StateStepCompleted,
        StateCompleted,
        StateCompensating,
        StateFailed,
        StateCancelled,
        StateTimedOut,
    },
    StateStepCompleted: {
        StateRunning,
        StateCompleted,
        StateCompensating,
        StateFailed,
        StateCancelled,
    },
    StateCompensating: {
        StateCompensated,
        StateFailed,
    },
    // 终止状态不能转换
    StateCompleted:   {},
    StateCompensated: {},
    StateFailed:      {},
    StateCancelled:   {},
    StateTimedOut:    {},
}
```

#### 状态转换验证

```go
func (m *StateManager) TransitionState(
    ctx context.Context,
    sagaID string,
    fromState, toState SagaState,
) error {
    // 验证转换是否合法
    if !isValidTransition(fromState, toState) {
        return fmt.Errorf("invalid transition from %s to %s",
            fromState.String(), toState.String())
    }

    // 获取当前实例验证状态
    instance, err := m.storage.GetSaga(ctx, sagaID)
    if err != nil {
        return err
    }

    currentState := instance.GetState()
    if currentState != fromState {
        return fmt.Errorf("expected state %s but found %s",
            fromState.String(), currentState.String())
    }

    // 更新状态
    if err := m.storage.UpdateSagaState(ctx, sagaID, toState, nil); err != nil {
        return err
    }

    // 通知状态变更监听器
    m.notifyStateChange(&StateChangeEvent{
        SagaID:    sagaID,
        OldState:  fromState,
        NewState:  toState,
        Timestamp: time.Now(),
    })

    return nil
}
```

#### 状态图

```
                          ┌─────────┐
                          │ Pending │
                          └────┬────┘
                               │
                               ▼
                          ┌─────────┐
                    ┌────▶│ Running │◀────┐
                    │     └────┬────┘     │
                    │          │          │
                    │          ▼          │
                    │   ┌──────────────┐  │
                    │   │StepCompleted │──┘
                    │   └──────┬───────┘
                    │          │
                    │          ▼
                    │   ┌───────────┐
                    │   │ Completed │ (终止)
                    │   └───────────┘
                    │
                    │   ┌──────────────┐
                    └───│Compensating  │
                        └──────┬───────┘
                               │
                               ▼
                        ┌─────────────┐
                        │Compensated  │ (终止)
                        └─────────────┘

            (错误路径)
                ┌──────┐      ┌───────────┐      ┌─────────┐
                │Failed│      │ Cancelled │      │TimedOut │
                └──────┘      └───────────┘      └─────────┘
                (终止)            (终止)             (终止)
```

### 执行引擎

#### StepExecutor

步骤执行器负责按顺序执行 Saga 的所有步骤。

**核心逻辑**：

```go
func (se *stepExecutor) executeSteps(ctx context.Context) error {
    steps := se.definition.GetSteps()
    
    // 设置 Saga 级别超时
    sagaCtx, cancel := context.WithTimeout(ctx, se.definition.GetTimeout())
    defer cancel()

    // 更新状态为 Running
    se.updateSagaState(sagaCtx, saga.StateRunning)

    // 顺序执行每个步骤
    currentData := se.instance.currentData
    for i, step := range steps {
        // 检查是否取消
        select {
        case <-sagaCtx.Done():
            return sagaCtx.Err()
        default:
        }

        // 执行步骤
        result, err := se.executeStep(sagaCtx, step, i, currentData)
        if err != nil {
            // 步骤失败，触发补偿
            return se.handleStepFailure(sagaCtx, i, err)
        }

        // 更新当前数据为步骤输出
        currentData = result
        
        // 持久化步骤状态
        se.saveStepState(sagaCtx, i, saga.StepStateCompleted)
    }

    // 所有步骤完成
    se.instance.result = currentData
    se.updateSagaState(sagaCtx, saga.StateCompleted)
    
    return nil
}
```

#### 步骤执行

```go
func (se *stepExecutor) executeStep(
    ctx context.Context,
    step saga.SagaStep,
    stepIndex int,
    data interface{},
) (interface{}, error) {
    // 创建步骤级别的追踪 span
    ctx, span := se.coordinator.tracingManager.StartSpan(
        ctx,
        fmt.Sprintf("step.%s", step.GetID()),
    )
    defer span.End()

    // 设置步骤超时
    stepCtx, cancel := context.WithTimeout(ctx, step.GetTimeout())
    defer cancel()

    // 更新步骤状态为 Running
    se.updateStepState(stepCtx, stepIndex, saga.StepStateRunning)

    // 执行步骤（带重试）
    var result interface{}
    var err error
    
    retryPolicy := step.GetRetryPolicy()
    if retryPolicy == nil {
        retryPolicy = se.coordinator.retryPolicy
    }

    for attempt := 1; attempt <= retryPolicy.GetMaxAttempts(); attempt++ {
        result, err = step.Execute(stepCtx, data)
        
        if err == nil {
            // 步骤执行成功
            se.updateStepState(stepCtx, stepIndex, saga.StepStateCompleted)
            return result, nil
        }

        // 判断是否可重试
        if !step.IsRetryable(err) || attempt >= retryPolicy.GetMaxAttempts() {
            break
        }

        // 等待重试延迟
        delay := retryPolicy.GetRetryDelay(attempt)
        select {
        case <-time.After(delay):
        case <-stepCtx.Done():
            return nil, stepCtx.Err()
        }
    }

    // 步骤失败
    se.updateStepState(stepCtx, stepIndex, saga.StepStateFailed)
    return nil, fmt.Errorf("step %s failed after %d attempts: %w",
        step.GetID(), retryPolicy.GetMaxAttempts(), err)
}
```

### 补偿机制

#### CompensationExecutor

补偿执行器负责在 Saga 失败时撤销已完成步骤的影响。

**补偿策略**：

1. **顺序补偿**（Sequential Compensation）
   - 按步骤反向顺序执行补偿
   - 一个接一个执行
   - 默认策略

2. **并行补偿**（Parallel Compensation）
   - 同时执行所有补偿操作
   - 适合独立步骤
   - 更快但风险更高

3. **尽力补偿**（Best Effort Compensation）
   - 即使部分失败也继续
   - 记录失败但不中断
   - 适合非关键操作

**实现示例**：

```go
func (ce *compensationExecutor) executeCompensation(
    ctx context.Context,
    failedStepIndex int,
) error {
    strategy := ce.definition.GetCompensationStrategy()
    
    // 获取已完成的步骤（需要补偿的步骤）
    completedSteps := ce.getCompletedSteps(failedStepIndex)
    
    // 更新状态为 Compensating
    ce.updateSagaState(ctx, saga.StateCompensating)

    // 根据策略获取补偿顺序
    compensationOrder := strategy.GetCompensationOrder(completedSteps)

    // 设置补偿超时
    compensationCtx, cancel := context.WithTimeout(
        ctx,
        strategy.GetCompensationTimeout(),
    )
    defer cancel()

    // 执行补偿
    if strategy.IsParallel() {
        return ce.executeParallelCompensation(compensationCtx, compensationOrder)
    }
    return ce.executeSequentialCompensation(compensationCtx, compensationOrder)
}

func (ce *compensationExecutor) executeSequentialCompensation(
    ctx context.Context,
    steps []saga.SagaStep,
) error {
    var errors []error

    for i := len(steps) - 1; i >= 0; i-- {
        step := steps[i]
        
        // 执行补偿
        if err := ce.compensateStep(ctx, step, i); err != nil {
            errors = append(errors, err)
            
            // 根据策略决定是否继续
            if !ce.isBestEffort() {
                break
            }
        }
    }

    if len(errors) > 0 {
        ce.updateSagaState(ctx, saga.StateFailed)
        return fmt.Errorf("compensation failed: %v", errors)
    }

    ce.updateSagaState(ctx, saga.StateCompensated)
    return nil
}

func (ce *compensationExecutor) compensateStep(
    ctx context.Context,
    step saga.SagaStep,
    stepIndex int,
) error {
    // 创建补偿追踪 span
    ctx, span := ce.coordinator.tracingManager.StartSpan(
        ctx,
        fmt.Sprintf("compensate.%s", step.GetID()),
    )
    defer span.End()

    // 更新步骤状态
    ce.updateStepState(ctx, stepIndex, saga.StepStateCompensating)

    // 执行补偿
    data := ce.getStepData(stepIndex)
    if err := step.Compensate(ctx, data); err != nil {
        ce.updateStepState(ctx, stepIndex, saga.StepStateFailed)
        return fmt.Errorf("compensation for step %s failed: %w",
            step.GetID(), err)
    }

    ce.updateStepState(ctx, stepIndex, saga.StepStateCompensated)
    return nil
}
```

---

## 扩展和定制

### 自定义步骤

#### 实现自定义步骤

要创建自定义步骤，需要实现 `SagaStep` 接口：

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

#### 完整示例：数据库操作步骤

```go
// DBTransactionStep 是一个执行数据库事务的步骤
type DBTransactionStep struct {
    id          string
    name        string
    description string
    db          *sql.DB
    timeout     time.Duration
    retryPolicy saga.RetryPolicy
}

func NewDBTransactionStep(db *sql.DB) *DBTransactionStep {
    return &DBTransactionStep{
        id:          "db-transaction",
        name:        "数据库事务",
        description: "执行数据库事务操作",
        db:          db,
        timeout:     30 * time.Second,
        retryPolicy: saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
    }
}

// GetID 返回步骤唯一标识符
func (s *DBTransactionStep) GetID() string {
    return s.id
}

// GetName 返回步骤名称
func (s *DBTransactionStep) GetName() string {
    return s.name
}

// GetDescription 返回步骤描述
func (s *DBTransactionStep) GetDescription() string {
    return s.description
}

// Execute 执行数据库事务
func (s *DBTransactionStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    // 类型断言获取输入数据
    txData, ok := data.(*TransactionData)
    if !ok {
        return nil, fmt.Errorf("invalid data type: expected *TransactionData")
    }

    // 开始数据库事务
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // 执行业务逻辑
    result, err := s.executeBusinessLogic(ctx, tx, txData)
    if err != nil {
        return nil, err
    }

    // 提交事务
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit transaction: %w", err)
    }

    // 保存事务 ID 用于补偿
    txData.TransactionID = result.TransactionID
    
    return txData, nil
}

// Compensate 撤销数据库操作
func (s *DBTransactionStep) Compensate(ctx context.Context, data interface{}) error {
    txData, ok := data.(*TransactionData)
    if !ok {
        return fmt.Errorf("invalid data type: expected *TransactionData")
    }

    // 开始补偿事务
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin compensation transaction: %w", err)
    }
    defer tx.Rollback()

    // 执行补偿逻辑（例如回滚之前的更改）
    if err := s.executeCompensationLogic(ctx, tx, txData); err != nil {
        return err
    }

    // 提交补偿事务
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit compensation: %w", err)
    }

    return nil
}

// GetTimeout 返回步骤超时时间
func (s *DBTransactionStep) GetTimeout() time.Duration {
    return s.timeout
}

// GetRetryPolicy 返回重试策略
func (s *DBTransactionStep) GetRetryPolicy() saga.RetryPolicy {
    return s.retryPolicy
}

// IsRetryable 判断错误是否可重试
func (s *DBTransactionStep) IsRetryable(err error) bool {
    // 网络错误和超时错误可以重试
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }
    
    // 数据库死锁可以重试
    if isDeadlockError(err) {
        return true
    }
    
    // 唯一约束违反不应重试
    if isUniqueViolationError(err) {
        return false
    }
    
    return true
}

// GetMetadata 返回步骤元数据
func (s *DBTransactionStep) GetMetadata() map[string]interface{} {
    return map[string]interface{}{
        "database": "postgres",
        "version":  "1.0",
    }
}

// 辅助方法
func (s *DBTransactionStep) executeBusinessLogic(
    ctx context.Context,
    tx *sql.Tx,
    data *TransactionData,
) (*TransactionResult, error) {
    // 实现具体的业务逻辑
    // ...
    return &TransactionResult{}, nil
}

func (s *DBTransactionStep) executeCompensationLogic(
    ctx context.Context,
    tx *sql.Tx,
    data *TransactionData,
) error {
    // 实现具体的补偿逻辑
    // ...
    return nil
}
```

#### 步骤设计最佳实践

1. **幂等性**
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
       
       // 标记为已执行
       s.markAsExecuted(ctx, data, result)
       
       return result, nil
   }
   ```

2. **原子性**
   ```go
   func (s *MyStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
       // 使用事务确保原子性
       tx, err := s.db.BeginTx(ctx, nil)
       if err != nil {
           return nil, err
       }
       defer tx.Rollback()
       
       if err := s.operation1(ctx, tx, data); err != nil {
           return nil, err
       }
       if err := s.operation2(ctx, tx, data); err != nil {
           return nil, err
       }
       
       return data, tx.Commit()
   }
   ```

3. **错误分类**
   ```go
   func (s *MyStep) IsRetryable(err error) bool {
       // 临时错误可重试
       if isTemporaryError(err) {
           return true
       }
       
       // 业务错误不重试
       if isBusinessError(err) {
           return false
       }
       
       // 默认可重试
       return true
   }
   ```

### 自定义存储适配器

#### 实现存储接口

要创建自定义存储后端，需要实现 `StateStorage` 接口：

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

#### 示例：MongoDB 存储适配器

```go
package storage

import (
    "context"
    "fmt"
    "time"

    "github.com/innovationmech/swit/pkg/saga"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

// MongoStateStorage implements StateStorage using MongoDB
type MongoStateStorage struct {
    client     *mongo.Client
    database   *mongo.Database
    collection *mongo.Collection
    ttl        time.Duration
}

// MongoConfig holds MongoDB storage configuration
type MongoConfig struct {
    URI            string
    Database       string
    Collection     string
    TTL            time.Duration
    ConnectTimeout time.Duration
}

// NewMongoStateStorage creates a new MongoDB storage backend
func NewMongoStateStorage(config *MongoConfig) (*MongoStateStorage, error) {
    ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
    defer cancel()

    // 连接 MongoDB
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.URI))
    if err != nil {
        return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
    }

    // 测试连接
    if err := client.Ping(ctx, nil); err != nil {
        return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
    }

    database := client.Database(config.Database)
    collection := database.Collection(config.Collection)

    // 创建索引
    if err := createIndexes(ctx, collection); err != nil {
        return nil, fmt.Errorf("failed to create indexes: %w", err)
    }

    return &MongoStateStorage{
        client:     client,
        database:   database,
        collection: collection,
        ttl:        config.TTL,
    }, nil
}

// SaveSaga saves a Saga instance to MongoDB
func (m *MongoStateStorage) SaveSaga(ctx context.Context, saga saga.SagaInstance) error {
    doc := sagaToDocument(saga)
    doc["_id"] = saga.GetID()
    doc["created_at"] = time.Now()
    doc["updated_at"] = time.Now()
    
    if m.ttl > 0 {
        doc["expire_at"] = time.Now().Add(m.ttl)
    }

    _, err := m.collection.InsertOne(ctx, doc)
    if err != nil {
        return fmt.Errorf("failed to save saga: %w", err)
    }

    return nil
}

// GetSaga retrieves a Saga instance from MongoDB
func (m *MongoStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
    var doc bson.M
    err := m.collection.FindOne(ctx, bson.M{"_id": sagaID}).Decode(&doc)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, fmt.Errorf("saga not found: %s", sagaID)
        }
        return nil, fmt.Errorf("failed to get saga: %w", err)
    }

    return documentToSaga(doc)
}

// UpdateSagaState updates the state of a Saga
func (m *MongoStateStorage) UpdateSagaState(
    ctx context.Context,
    sagaID string,
    state saga.SagaState,
    metadata map[string]interface{},
) error {
    update := bson.M{
        "$set": bson.M{
            "state":      state,
            "updated_at": time.Now(),
        },
    }

    if metadata != nil {
        update["$set"].(bson.M)["metadata"] = metadata
    }

    result, err := m.collection.UpdateOne(
        ctx,
        bson.M{"_id": sagaID},
        update,
    )
    if err != nil {
        return fmt.Errorf("failed to update saga state: %w", err)
    }

    if result.MatchedCount == 0 {
        return fmt.Errorf("saga not found: %s", sagaID)
    }

    return nil
}

// GetActiveSagas retrieves all active Saga instances
func (m *MongoStateStorage) GetActiveSagas(
    ctx context.Context,
    filter *saga.SagaFilter,
) ([]saga.SagaInstance, error) {
    query := buildMongoQuery(filter)
    
    cursor, err := m.collection.Find(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to query sagas: %w", err)
    }
    defer cursor.Close(ctx)

    var instances []saga.SagaInstance
    for cursor.Next(ctx) {
        var doc bson.M
        if err := cursor.Decode(&doc); err != nil {
            return nil, err
        }
        
        instance, err := documentToSaga(doc)
        if err != nil {
            return nil, err
        }
        
        instances = append(instances, instance)
    }

    return instances, nil
}

// Close closes the MongoDB connection
func (m *MongoStateStorage) Close() error {
    return m.client.Disconnect(context.Background())
}

// 辅助函数
func createIndexes(ctx context.Context, collection *mongo.Collection) error {
    indexes := []mongo.IndexModel{
        {
            Keys: bson.D{{Key: "state", Value: 1}},
        },
        {
            Keys: bson.D{{Key: "created_at", Value: 1}},
        },
        {
            Keys: bson.D{{Key: "expire_at", Value: 1}},
            Options: options.Index().SetExpireAfterSeconds(0),
        },
    }

    _, err := collection.Indexes().CreateMany(ctx, indexes)
    return err
}

func sagaToDocument(saga saga.SagaInstance) bson.M {
    return bson.M{
        "definition_id": saga.GetDefinitionID(),
        "state":         saga.GetState(),
        "current_step":  saga.GetCurrentStep(),
        "total_steps":   saga.GetTotalSteps(),
        "created_at":    saga.GetCreatedAt(),
        "metadata":      saga.GetMetadata(),
    }
}

func documentToSaga(doc bson.M) (saga.SagaInstance, error) {
    // 实现文档到 SagaInstance 的转换
    // ...
    return nil, nil
}

func buildMongoQuery(filter *saga.SagaFilter) bson.M {
    query := bson.M{}
    
    if filter != nil && len(filter.States) > 0 {
        query["state"] = bson.M{"$in": filter.States}
    }
    
    return query
}
```

### 自定义事件发布器

#### 实现事件发布接口

```go
type EventPublisher interface {
    PublishEvent(ctx context.Context, event *SagaEvent) error
    Subscribe(filter EventFilter, handler EventHandler) (EventSubscription, error)
    Unsubscribe(subscription EventSubscription) error
    Close() error
}
```

#### 示例：Kafka 事件发布器

```go
package messaging

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/innovationmech/swit/pkg/saga"
    "github.com/segmentio/kafka-go"
)

// KafkaEventPublisher implements EventPublisher using Kafka
type KafkaEventPublisher struct {
    writer     *kafka.Writer
    config     *KafkaConfig
    serializer Serializer
}

type KafkaConfig struct {
    Brokers     []string
    Topic       string
    Compression kafka.CompressionCodec
}

func NewKafkaEventPublisher(config *KafkaConfig) (*KafkaEventPublisher, error) {
    writer := &kafka.Writer{
        Addr:         kafka.TCP(config.Brokers...),
        Topic:        config.Topic,
        Balancer:     &kafka.LeastBytes{},
        Compression:  config.Compression,
        RequiredAcks: kafka.RequireAll,
    }

    return &KafkaEventPublisher{
        writer:     writer,
        config:     config,
        serializer: &JSONSerializer{},
    }, nil
}

func (k *KafkaEventPublisher) PublishEvent(
    ctx context.Context,
    event *saga.SagaEvent,
) error {
    // 序列化事件
    data, err := k.serializer.Serialize(event)
    if err != nil {
        return fmt.Errorf("failed to serialize event: %w", err)
    }

    // 创建 Kafka 消息
    message := kafka.Message{
        Key:   []byte(event.SagaID),
        Value: data,
        Headers: []kafka.Header{
            {Key: "event_type", Value: []byte(event.Type)},
            {Key: "saga_id", Value: []byte(event.SagaID)},
            {Key: "timestamp", Value: []byte(event.Timestamp.Format(time.RFC3339))},
        },
    }

    // 发送消息
    if err := k.writer.WriteMessages(ctx, message); err != nil {
        return fmt.Errorf("failed to publish event: %w", err)
    }

    return nil
}

func (k *KafkaEventPublisher) Close() error {
    return k.writer.Close()
}

// Serializer interface
type Serializer interface {
    Serialize(event *saga.SagaEvent) ([]byte, error)
    Deserialize(data []byte) (*saga.SagaEvent, error)
}

// JSONSerializer implements Serializer using JSON
type JSONSerializer struct{}

func (j *JSONSerializer) Serialize(event *saga.SagaEvent) ([]byte, error) {
    return json.Marshal(event)
}

func (j *JSONSerializer) Deserialize(data []byte) (*saga.SagaEvent, error) {
    var event saga.SagaEvent
    err := json.Unmarshal(data, &event)
    return &event, err
}
```

### 自定义重试策略

#### 实现重试策略接口

```go
type RetryPolicy interface {
    ShouldRetry(err error, attempt int) bool
    GetRetryDelay(attempt int) time.Duration
    GetMaxAttempts() int
}
```

#### 示例：自适应重试策略

```go
package retry

import (
    "math"
    "math/rand"
    "time"

    "github.com/innovationmech/swit/pkg/saga"
)

// AdaptiveRetryPolicy adjusts retry delays based on success/failure history
type AdaptiveRetryPolicy struct {
    baseDelay      time.Duration
    maxDelay       time.Duration
    maxAttempts    int
    successCount   int
    failureCount   int
    adjustmentRate float64
}

func NewAdaptiveRetryPolicy(
    baseDelay, maxDelay time.Duration,
    maxAttempts int,
) *AdaptiveRetryPolicy {
    return &AdaptiveRetryPolicy{
        baseDelay:      baseDelay,
        maxDelay:       maxDelay,
        maxAttempts:    maxAttempts,
        adjustmentRate: 1.0,
    }
}

func (a *AdaptiveRetryPolicy) ShouldRetry(err error, attempt int) bool {
    if attempt >= a.maxAttempts {
        return false
    }

    // 根据错误类型决定是否重试
    if saga.IsBusinessError(err) {
        return false
    }

    return true
}

func (a *AdaptiveRetryPolicy) GetRetryDelay(attempt int) time.Duration {
    // 计算基础延迟（指数退避）
    baseDelay := float64(a.baseDelay) * math.Pow(2, float64(attempt-1))
    
    // 根据历史调整延迟
    adjustedDelay := baseDelay * a.adjustmentRate
    
    // 添加抖动
    jitter := rand.Float64() * 0.3 * adjustedDelay
    delay := adjustedDelay + jitter

    // 限制最大延迟
    if delay > float64(a.maxDelay) {
        delay = float64(a.maxDelay)
    }

    return time.Duration(delay)
}

func (a *AdaptiveRetryPolicy) GetMaxAttempts() int {
    return a.maxAttempts
}

// RecordSuccess 记录成功，降低延迟
func (a *AdaptiveRetryPolicy) RecordSuccess() {
    a.successCount++
    a.adjustmentRate = math.Max(0.5, a.adjustmentRate*0.9)
}

// RecordFailure 记录失败，增加延迟
func (a *AdaptiveRetryPolicy) RecordFailure() {
    a.failureCount++
    a.adjustmentRate = math.Min(2.0, a.adjustmentRate*1.1)
}
```

---

## 内部工作流程

### Saga 启动流程

```
用户调用 StartSaga
        │
        ▼
┌───────────────────┐
│ 1. 验证协调器状态 │
│   - 检查是否关闭  │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│ 2. 验证 Saga 定义 │
│   - 定义非空      │
│   - 步骤有效      │
│   - 配置正确      │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│ 3. 创建 Saga 实例│
│   - 生成唯一 ID   │
│   - 初始化状态    │
│   - 设置超时      │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│ 4. 持久化初始状态 │
│   - 保存到存储    │
│   - 记录时间戳    │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│ 5. 发布启动事件   │
│   - EventSagaStarted │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│ 6. 启动异步执行   │
│   - 创建 goroutine │
│   - 执行步骤      │
└─────────┬─────────┘
          │
          ▼
   返回 SagaInstance
```

### 步骤执行流程

```
开始执行步骤
      │
      ▼
┌──────────────────┐
│ 1. 更新状态      │
│    StateRunning  │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ 2. 设置超时      │
│    - Saga 超时   │
│    - 步骤超时    │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐     YES    ┌──────────────────┐
│ 3. 遍历所有步骤  │─────────▶│ 执行单个步骤     │
└────────┬─────────┘            └────────┬─────────┘
         │                               │
         │ NO (所有步骤完成)               │
         │                               ▼
         │                      ┌──────────────────┐
         │                      │ 3.1 创建追踪 span│
         │                      └────────┬─────────┘
         │                               │
         │                               ▼
         │                      ┌──────────────────┐
         │                      │ 3.2 更新步骤状态 │
         │                      │   StepStateRunning│
         │                      └────────┬─────────┘
         │                               │
         │                               ▼
         │                      ┌──────────────────┐
         │                      │ 3.3 执行重试循环 │◀─┐
         │                      └────────┬─────────┘  │
         │                               │            │
         │                               ▼            │
         │                      ┌──────────────────┐  │
         │                      │ 调用 Execute()  │  │
         │                      └────────┬─────────┘  │
         │                               │            │
         │                    ┌─────────┴──────────┐ │
         │                    │                    │ │
         │                    ▼                    ▼ │
         │            ┌──────────────┐    ┌──────────────┐
         │            │  步骤成功    │    │  步骤失败    │
         │            └──────┬───────┘    └──────┬───────┘
         │                   │                   │
         │                   │                   ▼
         │                   │          ┌──────────────────┐
         │                   │          │ 判断是否可重试    │
         │                   │          └────────┬─────────┘
         │                   │                   │
         │                   │     ┌─────────────┴───────────┐
         │                   │     │                         │
         │                   │     ▼ YES                     ▼ NO
         │                   │  等待延迟──────────────────┘ 触发补偿
         │                   │     │                           │
         │                   ▼     └───────────────────────────┘
         │          ┌──────────────────┐
         │          │ 3.4 保存步骤结果│
         │          └────────┬─────────┘
         │                   │
         │                   ▼
         │          ┌──────────────────┐
         │          │ 3.5 传递数据到  │
         │          │     下一步骤     │
         │          └────────┬─────────┘
         │                   │
         └───────────────────┘
         │
         ▼
┌──────────────────┐
│ 4. 所有步骤完成  │
│   更新状态为     │
│   StateCompleted │
└──────────────────┘
```

### 补偿流程

```
步骤失败触发补偿
        │
        ▼
┌─────────────────────┐
│ 1. 更新状态         │
│   StateCompensating │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ 2. 获取已完成步骤   │
│   (需要补偿的步骤)  │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ 3. 确定补偿顺序     │
│   - 顺序 (reverse)  │
│   - 并行            │
│   - 自定义          │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ 4. 执行补偿操作     │
└──────────┬──────────┘
           │
   ┌───────┴────────┐
   │                │
   ▼ 顺序补偿        ▼ 并行补偿
┌──────────┐    ┌──────────┐
│从后向前逐│    │同时执行所│
│个执行补偿│    │有补偿操作│
└────┬─────┘    └────┬─────┘
     │               │
     └───────┬───────┘
             │
             ▼
    ┌────────────────┐
    │ 遍历每个步骤   │
    └────────┬───────┘
             │
             ▼
    ┌────────────────┐
    │ 调用 Compensate│
    └────────┬───────┘
             │
     ┌───────┴────────┐
     │                │
     ▼ 成功            ▼ 失败
┌──────────┐     ┌──────────┐
│记录成功  │     │记录失败  │
│继续下一个│     │根据策略  │
│          │     │决定是否  │
│          │     │继续      │
└────┬─────┘     └────┬─────┘
     │                │
     └────────┬───────┘
              │
              ▼
    ┌─────────────────┐
    │ 5. 完成补偿     │
    └─────────┬───────┘
              │
      ┌───────┴────────┐
      │                │
      ▼ 全部成功        ▼ 部分/全部失败
┌──────────────┐  ┌──────────────┐
│ 更新状态为   │  │ 更新状态为   │
│StateCompensated│  │ StateFailed  │
└──────────────┘  └──────────────┘
```

### 状态恢复流程

```
系统启动/定时检查
        │
        ▼
┌─────────────────────┐
│ 1. 查询超时/中断的  │
│    Saga 实例        │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ 2. 遍历每个实例     │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ 3. 检查状态         │
└──────────┬──────────┘
           │
    ┌──────┴────────────────┐
    │                       │
    ▼ StateRunning          ▼ StateCompensating
┌──────────────┐       ┌──────────────┐
│ 恢复执行流程 │       │ 恢复补偿流程 │
└──────┬───────┘       └──────┬───────┘
       │                      │
       ▼                      ▼
┌──────────────┐       ┌──────────────┐
│ 找到当前步骤 │       │ 找到已补偿的 │
│ 继续执行     │       │ 步骤，继续补偿│
└──────┬───────┘       └──────┬───────┘
       │                      │
       └──────────┬───────────┘
                  │
                  ▼
        ┌─────────────────┐
        │ 4. 更新恢复时间 │
        └─────────┬───────┘
                  │
                  ▼
        ┌─────────────────┐
        │ 5. 发布恢复事件 │
        └─────────────────┘
```

---

## 测试策略

### 单元测试

#### 测试结构

```
pkg/saga/
├── coordinator/
│   ├── orchestrator_test.go
│   ├── choreography_test.go
│   └── hybrid_test.go
├── state/
│   ├── manager_test.go
│   └── storage/
│       ├── memory_test.go
│       ├── redis_test.go
│       └── postgres_test.go
├── retry/
│   ├── policy_test.go
│   └── backoff_test.go
└── messaging/
    ├── publisher_test.go
    └── handler_test.go
```

#### 测试示例

##### 测试步骤执行

```go
package coordinator

import (
    "context"
    "testing"
    "time"

    "github.com/innovationmech/swit/pkg/saga"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestStepExecutor_ExecuteStep_Success(t *testing.T) {
    // 创建测试协调器
    coordinator := newTestCoordinator(t)
    defer coordinator.Close()

    // 创建测试步骤
    step := &mockSagaStep{
        id: "test-step",
        executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
            return "success", nil
        },
    }

    // 创建测试数据
    testData := map[string]interface{}{
        "test_key": "test_value",
    }

    // 执行步骤
    executor := newStepExecutor(coordinator, nil, nil)
    result, err := executor.executeStep(context.Background(), step, 0, testData)

    // 验证结果
    require.NoError(t, err)
    assert.Equal(t, "success", result)
}

func TestStepExecutor_ExecuteStep_WithRetry(t *testing.T) {
    attemptCount := 0
    step := &mockSagaStep{
        id: "retry-step",
        executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
            attemptCount++
            if attemptCount < 3 {
                return nil, fmt.Errorf("temporary error")
            }
            return "success after retry", nil
        },
        retryable: true,
        retryPolicy: saga.NewFixedDelayRetryPolicy(3, 100*time.Millisecond),
    }

    coordinator := newTestCoordinator(t)
    executor := newStepExecutor(coordinator, nil, nil)
    
    result, err := executor.executeStep(context.Background(), step, 0, nil)

    require.NoError(t, err)
    assert.Equal(t, "success after retry", result)
    assert.Equal(t, 3, attemptCount)
}
```

##### 测试状态转换

```go
func TestStateManager_TransitionState_Valid(t *testing.T) {
    tests := []struct {
        name      string
        fromState saga.SagaState
        toState   saga.SagaState
        wantErr   bool
    }{
        {
            name:      "Pending to Running",
            fromState: saga.StatePending,
            toState:   saga.StateRunning,
            wantErr:   false,
        },
        {
            name:      "Running to Completed",
            fromState: saga.StateRunning,
            toState:   saga.StateCompleted,
            wantErr:   false,
        },
        {
            name:      "Completed to Running (invalid)",
            fromState: saga.StateCompleted,
            toState:   saga.StateRunning,
            wantErr:   true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            manager := newTestStateManager(t)
            
            // 创建测试 Saga
            sagaID := createTestSaga(t, manager, tt.fromState)
            
            // 尝试状态转换
            err := manager.TransitionState(
                context.Background(),
                sagaID,
                tt.fromState,
                tt.toState,
            )

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                
                // 验证状态已更新
                instance, _ := manager.GetSaga(context.Background(), sagaID)
                assert.Equal(t, tt.toState, instance.GetState())
            }
        })
    }
}
```

##### 测试补偿逻辑

```go
func TestCompensationExecutor_Execute_Sequential(t *testing.T) {
    compensatedSteps := []string{}
    
    // 创建三个已完成的步骤
    steps := []saga.SagaStep{
        &mockSagaStep{
            id: "step-1",
            compensateFunc: func(ctx context.Context, data interface{}) error {
                compensatedSteps = append(compensatedSteps, "step-1")
                return nil
            },
        },
        &mockSagaStep{
            id: "step-2",
            compensateFunc: func(ctx context.Context, data interface{}) error {
                compensatedSteps = append(compensatedSteps, "step-2")
                return nil
            },
        },
        &mockSagaStep{
            id: "step-3",
            compensateFunc: func(ctx context.Context, data interface{}) error {
                compensatedSteps = append(compensatedSteps, "step-3")
                return nil
            },
        },
    }

    coordinator := newTestCoordinator(t)
    executor := newCompensationExecutor(coordinator, nil, steps)
    
    // 执行补偿（step-3 失败，需要补偿 step-1 和 step-2）
    err := executor.executeCompensation(context.Background(), 2)
    
    require.NoError(t, err)
    // 验证补偿顺序是反向的
    assert.Equal(t, []string{"step-2", "step-1"}, compensatedSteps)
}
```

#### 使用模拟对象

```go
// mockSagaStep 是测试用的模拟步骤
type mockSagaStep struct {
    id             string
    name           string
    executeFunc    func(context.Context, interface{}) (interface{}, error)
    compensateFunc func(context.Context, interface{}) error
    timeout        time.Duration
    retryPolicy    saga.RetryPolicy
    retryable      bool
}

func (m *mockSagaStep) GetID() string { return m.id }
func (m *mockSagaStep) GetName() string { return m.name }
func (m *mockSagaStep) GetDescription() string { return "" }
func (m *mockSagaStep) GetTimeout() time.Duration { return m.timeout }
func (m *mockSagaStep) GetRetryPolicy() saga.RetryPolicy { return m.retryPolicy }
func (m *mockSagaStep) IsRetryable(err error) bool { return m.retryable }
func (m *mockSagaStep) GetMetadata() map[string]interface{} { return nil }

func (m *mockSagaStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
    if m.executeFunc != nil {
        return m.executeFunc(ctx, data)
    }
    return data, nil
}

func (m *mockSagaStep) Compensate(ctx context.Context, data interface{}) error {
    if m.compensateFunc != nil {
        return m.compensateFunc(ctx, data)
    }
    return nil
}

// mockStateStorage 是测试用的模拟存储
type mockStateStorage struct {
    sagas map[string]saga.SagaInstance
    mu    sync.RWMutex
}

func newMockStateStorage() *mockStateStorage {
    return &mockStateStorage{
        sagas: make(map[string]saga.SagaInstance),
    }
}

func (m *mockStateStorage) SaveSaga(ctx context.Context, saga saga.SagaInstance) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.sagas[saga.GetID()] = saga
    return nil
}

func (m *mockStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    saga, ok := m.sagas[sagaID]
    if !ok {
        return nil, fmt.Errorf("saga not found: %s", sagaID)
    }
    return saga, nil
}
```

### 集成测试

#### 端到端测试示例

```go
func TestSagaE2E_OrderProcessing_Success(t *testing.T) {
    // 1. 设置测试环境
    ctx := context.Background()
    
    // 创建真实的存储（使用 Docker 容器或内存）
    storage := createTestStorage(t)
    defer storage.Close()
    
    // 创建真实的事件发布器
    publisher := createTestPublisher(t)
    defer publisher.Close()
    
    // 创建协调器
    coordinator, err := coordinator.NewOrchestratorCoordinator(&coordinator.OrchestratorConfig{
        StateStorage:   storage,
        EventPublisher: publisher,
        RetryPolicy:    saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
    })
    require.NoError(t, err)
    defer coordinator.Close()

    // 2. 创建测试服务
    orderService := newTestOrderService()
    inventoryService := newTestInventoryService()
    paymentService := newTestPaymentService()

    // 3. 创建 Saga 定义
    definition := createOrderProcessingSaga(orderService, inventoryService, paymentService)

    // 4. 准备测试数据
    orderData := &OrderData{
        UserID:    "test-user-123",
        ProductID: "test-product-456",
        Quantity:  2,
        Amount:    199.98,
    }

    // 5. 启动 Saga
    instance, err := coordinator.StartSaga(ctx, definition, orderData)
    require.NoError(t, err)
    sagaID := instance.GetID()

    // 6. 等待 Saga 完成
    err = waitForSagaCompletion(t, coordinator, sagaID, 30*time.Second)
    require.NoError(t, err)

    // 7. 验证最终状态
    finalInstance, err := coordinator.GetSaga(sagaID)
    require.NoError(t, err)
    assert.Equal(t, saga.StateCompleted, finalInstance.GetState())

    // 8. 验证业务结果
    result := finalInstance.GetResult().(*OrderData)
    assert.NotEmpty(t, result.OrderID)
    assert.NotEmpty(t, result.ReservationID)
    assert.NotEmpty(t, result.PaymentID)
    assert.Equal(t, "CONFIRMED", result.Status)

    // 9. 验证服务调用
    assert.True(t, orderService.OrderCreated(result.OrderID))
    assert.True(t, inventoryService.InventoryReserved(result.ReservationID))
    assert.True(t, paymentService.PaymentProcessed(result.PaymentID))
}

func TestSagaE2E_OrderProcessing_PaymentFailure(t *testing.T) {
    // ... 设置协调器 ...

    // 创建会失败的支付服务
    paymentService := newTestPaymentService()
    paymentService.SetFailureMode(true) // 模拟支付失败

    definition := createOrderProcessingSaga(orderService, inventoryService, paymentService)
    
    // 启动 Saga
    instance, err := coordinator.StartSaga(ctx, definition, orderData)
    require.NoError(t, err)

    // 等待完成（应该是补偿完成）
    err = waitForSagaCompletion(t, coordinator, instance.GetID(), 30*time.Second)
    require.NoError(t, err)

    // 验证补偿已执行
    finalInstance, err := coordinator.GetSaga(instance.GetID())
    require.NoError(t, err)
    assert.Equal(t, saga.StateCompensated, finalInstance.GetState())

    // 验证补偿效果
    result := finalInstance.GetResult().(*OrderData)
    assert.False(t, orderService.OrderCreated(result.OrderID)) // 订单应该已取消
    assert.False(t, inventoryService.InventoryReserved(result.ReservationID)) // 库存应该已释放
}
```

#### 辅助测试函数

```go
func waitForSagaCompletion(
    t *testing.T,
    coordinator saga.SagaCoordinator,
    sagaID string,
    timeout time.Duration,
) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("timeout waiting for saga completion")
        case <-ticker.C:
            instance, err := coordinator.GetSagaInstance(sagaID)
            if err != nil {
                return err
            }
            
            state := instance.GetState()
            if state.IsTerminal() {
                return nil
            }
        }
    }
}

func createTestStorage(t *testing.T) saga.StateStorage {
    // 选项 1: 使用内存存储
    return storage.NewMemoryStateStorage()
    
    // 选项 2: 使用 Docker 容器中的 Redis
    // return storage.NewRedisStateStorage(&storage.RedisConfig{
    //     Endpoints: []string{getTestRedisEndpoint(t)},
    // })
}

func createTestPublisher(t *testing.T) saga.EventPublisher {
    // 使用内存发布器用于测试
    return NewInMemoryEventPublisher()
}
```

### 端到端测试

#### 使用 Docker Compose

```yaml
# test/docker-compose.yml
version: '3.8'

services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: saga_test
      POSTGRES_USER: saga
      POSTGRES_PASSWORD: saga_test
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U saga"]
      interval: 5s
      timeout: 3s
      retries: 5

  nats:
    image: nats:2-alpine
    ports:
      - "4222:4222"
    command: ["-js"]
```

#### 测试脚本

```bash
#!/bin/bash
# test/run-e2e-tests.sh

set -e

echo "Starting test infrastructure..."
docker-compose -f test/docker-compose.yml up -d

echo "Waiting for services to be healthy..."
docker-compose -f test/docker-compose.yml wait

echo "Running database migrations..."
go run cmd/saga-migrate/main.go --config test/test-config.yaml

echo "Running E2E tests..."
go test -v -tags=e2e ./pkg/saga/integration/...

echo "Cleaning up..."
docker-compose -f test/docker-compose.yml down -v
```

---

## 开发环境设置

### 前置要求

- Go 1.23 或更高版本
- Docker 和 Docker Compose（用于集成测试）
- Make（可选，用于构建脚本）
- Git

### 克隆仓库

```bash
git clone https://github.com/innovationmech/swit.git
cd swit
```

### 安装依赖

```bash
# 安装 Go 依赖
go mod download

# 安装开发工具
make setup-dev
```

### 配置开发环境

1. **创建本地配置文件**

```bash
cp swit.yaml.example swit-dev.yaml
```

2. **启动本地依赖服务**

```bash
docker-compose up -d redis postgres nats
```

3. **运行数据库迁移**

```bash
go run cmd/saga-migrate/main.go --config swit-dev.yaml
```

### 运行测试

```bash
# 运行所有单元测试
make test

# 运行特定包的测试
go test ./pkg/saga/coordinator/...

# 运行带覆盖率的测试
make test-coverage

# 查看覆盖率报告
open coverage.html
```

### 代码质量检查

```bash
# 运行完整的质量检查
make quality

# 或者运行单独的检查
make fmt      # 格式化代码
make vet      # 静态分析
make lint     # Linter 检查
```

### 构建项目

```bash
# 构建所有二进制文件
make build

# 构建特定二进制
make build-swit-serve
make build-swit-auth
make build-switctl
```

### IDE 配置

#### VS Code

安装推荐的扩展：
- Go (golang.go)
- GitLens
- Docker

创建 `.vscode/launch.json`：

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Saga Tests",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${workspaceFolder}/pkg/saga/coordinator",
            "args": ["-test.v"]
        }
    ]
}
```

#### GoLand / IntelliJ IDEA

1. 打开项目
2. IDE 会自动识别 Go 模块
3. 配置 Go SDK（Go 1.23+）
4. 启用 Go Modules 支持

---

## 贡献指南

### 开发流程

1. **Fork 仓库**

访问 [GitHub](https://github.com/innovationmech/swit) 并 Fork 仓库。

2. **克隆你的 Fork**

```bash
git clone https://github.com/YOUR_USERNAME/swit.git
cd swit
git remote add upstream https://github.com/innovationmech/swit.git
```

3. **创建特性分支**

```bash
git checkout -b feat/saga-my-feature
```

4. **开发和测试**

- 编写代码
- 添加测试
- 运行测试：`make test`
- 检查代码质量：`make quality`

5. **提交更改**

遵循 Conventional Commits 规范：

```bash
git add .
git commit -m "feat(saga): add new retry strategy"
```

提交信息格式：
- `feat(scope): description` - 新功能
- `fix(scope): description` - Bug 修复
- `docs(scope): description` - 文档更新
- `test(scope): description` - 测试相关
- `refactor(scope): description` - 重构
- `perf(scope): description` - 性能优化
- `chore(scope): description` - 构建/工具相关

6. **推送到你的 Fork**

```bash
git push origin feat/saga-my-feature
```

7. **创建 Pull Request**

- 访问你的 Fork 页面
- 点击 "New Pull Request"
- 填写 PR 描述，包括：
  - 变更摘要
  - 相关 Issue（使用 `Closes #<issue-number>`）
  - 测试说明
  - 截图（如适用）

### 代码规范

#### Go 代码风格

遵循官方 Go 代码风格：
- 使用 `gofmt` 格式化代码
- 使用 `goimports` 管理导入
- 导出的函数和类型需要有文档注释
- 保持函数简短，单一职责

#### 命名约定

- **包名**: 小写，无下划线，简短且有意义
- **文件名**: 小写，使用下划线分隔单词
- **接口**: 名词或形容词，通常以 `-er` 结尾
- **函数**: 驼峰命名，导出函数首字母大写
- **变量**: 驼峰命名，局部变量简短，全局变量描述性强

示例：
```go
// 好的命名
type UserRepository interface {}
func (r *Repository) GetUser(id string) (*User, error) {}
var ErrUserNotFound = errors.New("user not found")

// 不好的命名
type user_repo interface {}
func (r *Repository) get_user(ID string) (*User, error) {}
var errorUserNotFound = errors.New("user not found")
```

#### 错误处理

- 总是检查错误
- 使用 `fmt.Errorf` 包装错误以添加上下文
- 使用 `errors.Is` 和 `errors.As` 进行错误检查
- 定义可导出的错误变量用于常见错误

```go
// 好的错误处理
result, err := doSomething()
if err != nil {
    return nil, fmt.Errorf("failed to do something: %w", err)
}

// 错误检查
if errors.Is(err, ErrNotFound) {
    // 处理特定错误
}
```

### 测试要求

- 所有新功能必须有单元测试
- 复杂逻辑需要集成测试
- 测试覆盖率应 > 80%
- 使用表驱动测试
- 测试应该快速、可靠、独立

示例测试结构：
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   Input
        want    Output
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   validInput,
            want:    expectedOutput,
            wantErr: false,
        },
        // 更多测试用例...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 文档要求

- 所有导出的类型、函数和方法需要文档注释
- 文档注释应该以类型/函数名开头
- 包级别的文档在 `doc.go` 文件中
- 复杂的功能需要有使用示例

```go
// SagaCoordinator manages the lifecycle of Saga instances.
// It provides centralized control over Saga execution, state management,
// and compensation operations.
//
// Example:
//   coordinator := NewOrchestratorCoordinator(config)
//   instance, err := coordinator.StartSaga(ctx, definition, data)
type SagaCoordinator interface {
    // StartSaga initiates a new Saga instance with the given definition and data.
    // Returns the created instance or an error if the Saga cannot be started.
    StartSaga(ctx context.Context, definition SagaDefinition, data interface{}) (SagaInstance, error)
}
```

### PR 检查清单

在提交 PR 前，确保：

- [ ] 代码遵循项目风格指南
- [ ] 所有测试通过：`make test`
- [ ] 代码质量检查通过：`make quality`
- [ ] 添加了必要的测试
- [ ] 更新了相关文档
- [ ] 提交信息遵循 Conventional Commits
- [ ] PR 描述清晰，包含相关 Issue 链接
- [ ] 没有引入新的 linter 警告
- [ ] 生成的代码已提交（如 proto 文件）

### 获取帮助

- 查看 [GitHub Issues](https://github.com/innovationmech/swit/issues)
- 阅读 [用户指南](saga-user-guide.md)
- 阅读 [API 参考](saga-api-reference.md)
- 加入社区讨论

### 行为准则

请遵守我们的 [行为准则](../CODE_OF_CONDUCT.md)，保持友好和专业。

---

## 附录

### 相关文档

- [Saga 用户指南](saga-user-guide.md)
- [Saga API 参考](saga-api-reference.md)
- [Saga DSL 参考](saga-dsl-reference.md)
- [Saga 监控指南](saga-monitoring-guide.md)
- [Saga 安全指南](saga-security-guide.md)
- [Saga 测试指南](saga-testing-guide.md)

### 外部资源

- [Saga 模式介绍](https://microservices.io/patterns/data/saga.html)
- [分布式事务最佳实践](https://www.infoq.com/articles/saga-orchestration-outbox/)
- [Go 编码规范](https://golang.org/doc/effective_go)

---

**文档版本**: 1.0  
**创建日期**: 2025-10-28  
**最后更新**: 2025-10-28  
**维护者**: Swit 开发团队

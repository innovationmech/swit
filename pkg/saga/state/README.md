# Saga State Management

`pkg/saga/state` 包提供了完整的 Saga 状态管理解决方案，包括状态存储、序列化和状态管理器。该包支持 Saga 协调器进行可靠的分布式事务处理。

## 目录

- [概述](#概述)
- [核心组件](#核心组件)
- [快速开始](#快速开始)
- [存储后端](#存储后端)
- [序列化](#序列化)
- [状态管理器](#状态管理器)
- [配置选项](#配置选项)
- [最佳实践](#最佳实践)
- [示例](#示例)

## 概述

Saga 状态管理包提供了三个核心功能：

1. **状态存储（State Storage）**：持久化和检索 Saga 实例及其步骤状态
2. **序列化（Serialization）**：将 Saga 状态序列化为不同格式（JSON、Protobuf 等）
3. **状态管理器（State Manager）**：提供统一的高级状态管理 API，包含验证和事件通知

## 核心组件

### StateStorage 接口

`StateStorage` 接口定义了 Saga 状态持久化的核心操作：

```go
type StateStorage interface {
    // Saga 实例操作
    SaveSaga(ctx context.Context, saga SagaInstance) error
    GetSaga(ctx context.Context, sagaID string) (SagaInstance, error)
    UpdateSagaState(ctx context.Context, sagaID string, state SagaState, metadata map[string]interface{}) error
    DeleteSaga(ctx context.Context, sagaID string) error
    
    // 查询操作
    GetActiveSagas(ctx context.Context, filter *SagaFilter) ([]SagaInstance, error)
    GetTimeoutSagas(ctx context.Context, before time.Time) ([]SagaInstance, error)
    
    // 步骤状态操作
    SaveStepState(ctx context.Context, sagaID string, step *StepState) error
    GetStepStates(ctx context.Context, sagaID string) ([]*StepState, error)
    
    // 维护操作
    CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error
}
```

### StateManager 接口

`StateManager` 在 `StateStorage` 之上提供更高级的功能：

```go
type StateManager interface {
    // 基础操作
    SaveSaga(ctx context.Context, instance saga.SagaInstance) error
    GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error)
    DeleteSaga(ctx context.Context, sagaID string) error
    
    // 状态转换（带验证）
    UpdateSagaState(ctx context.Context, sagaID string, newState saga.SagaState, metadata map[string]interface{}) error
    TransitionState(ctx context.Context, sagaID string, fromState, toState saga.SagaState, metadata map[string]interface{}) error
    
    // 批量操作
    BatchUpdateSagas(ctx context.Context, updates []SagaUpdate) error
    
    // 事件订阅
    SubscribeToStateChanges(listener StateChangeListener) error
    UnsubscribeFromStateChanges(listener StateChangeListener) error
    
    // 其他操作...
}
```

### Serializer 接口

`Serializer` 接口支持 Saga 状态的序列化和反序列化：

```go
type Serializer interface {
    SerializeSaga(instance saga.SagaInstance) ([]byte, error)
    DeserializeSaga(data []byte) (saga.SagaInstance, error)
    SerializeStepState(step *saga.StepState) ([]byte, error)
    DeserializeStepState(data []byte) (*saga.StepState, error)
}
```

## 快速开始

### 基础用法

```go
package main

import (
    "context"
    "time"
    
    "github.com/innovationmech/swit/pkg/saga"
    "github.com/innovationmech/swit/pkg/saga/state"
    "github.com/innovationmech/swit/pkg/saga/state/storage"
)

func main() {
    // 1. 创建存储后端
    memStorage := storage.NewMemoryStateStorage()
    defer memStorage.Close()
    
    // 2. 创建状态管理器
    manager := state.NewStateManager(memStorage)
    defer manager.Close()
    
    // 3. 保存 Saga 实例
    ctx := context.Background()
    sagaInstance := createSagaInstance() // 你的 Saga 实例
    
    if err := manager.SaveSaga(ctx, sagaInstance); err != nil {
        panic(err)
    }
    
    // 4. 更新状态
    metadata := map[string]interface{}{
        "updated_by": "system",
        "timestamp": time.Now(),
    }
    
    if err := manager.UpdateSagaState(ctx, sagaInstance.GetID(), saga.StateRunning, metadata); err != nil {
        panic(err)
    }
    
    // 5. 查询 Saga
    retrieved, err := manager.GetSaga(ctx, sagaInstance.GetID())
    if err != nil {
        panic(err)
    }
    
    println("Saga state:", retrieved.GetState().String())
}
```

### 使用状态转换验证

```go
// 使用 TransitionState 进行带验证的状态转换
err := manager.TransitionState(
    ctx,
    sagaID,
    saga.StatePending,    // 期望的当前状态
    saga.StateRunning,    // 目标状态
    metadata,
)

if err != nil {
    // 处理无效的状态转换
    log.Printf("Invalid state transition: %v", err)
}
```

### 订阅状态变更事件

```go
// 订阅状态变更
listener := func(event *state.StateChangeEvent) {
    log.Printf("Saga %s transitioned from %s to %s",
        event.SagaID,
        event.OldState.String(),
        event.NewState.String())
}

if err := manager.SubscribeToStateChanges(listener); err != nil {
    panic(err)
}

// 后续的状态更新会触发事件通知
```

## 存储后端

### 内存存储（Memory Storage）

内存存储适用于开发、测试和单实例部署：

```go
// 使用默认配置
storage := storage.NewMemoryStateStorage()

// 使用自定义配置
config := &state.MemoryStorageConfig{
    InitialCapacity: 1000,
    MaxCapacity:     10000,
    EnableMetrics:   true,
}
storage := storage.NewMemoryStateStorageWithConfig(config)
```

**特点：**
- ✅ 高性能（无网络开销）
- ✅ 并发安全（使用 RWMutex）
- ✅ 支持所有查询操作
- ❌ 不持久化（重启后数据丢失）
- ❌ 不支持分布式部署

### Redis 存储（即将支持）

计划中的 Redis 存储后端将支持：
- 持久化存储
- 分布式部署
- 高可用性
- TTL 自动清理

### 数据库存储（即将支持）

计划中的数据库存储后端将支持：
- PostgreSQL
- MySQL
- 复杂查询
- 事务支持

## 序列化

### JSON 序列化器

默认的 JSON 序列化器适用于大多数场景：

```go
serializer := state.NewJSONSerializer()

// 序列化
data, err := serializer.SerializeSaga(sagaInstance)
if err != nil {
    panic(err)
}

// 反序列化
deserialized, err := serializer.DeserializeSaga(data)
if err != nil {
    panic(err)
}
```

**特点：**
- 人类可读
- 调试友好
- 支持复杂数据结构
- 性能适中

### 自定义序列化器

你可以实现自己的序列化器：

```go
type MySerializer struct{}

func (s *MySerializer) SerializeSaga(instance saga.SagaInstance) ([]byte, error) {
    // 自定义序列化逻辑
}

func (s *MySerializer) DeserializeSaga(data []byte) (saga.SagaInstance, error) {
    // 自定义反序列化逻辑
}
```

## 状态管理器

### 状态验证

StateManager 提供内置的状态验证：

```go
// 验证状态转换
err := manager.TransitionState(ctx, sagaID, fromState, toState, metadata)

// 有效的状态转换：
// - Pending -> Running
// - Running -> StepCompleted
// - Running -> Completed
// - Running -> Compensating
// - Compensating -> Compensated
// ...

// 无效的状态转换会返回错误
// 例如：Completed -> Running（终止状态不能转换）
```

### 批量更新

```go
updates := []state.SagaUpdate{
    {
        SagaID:   "saga-1",
        NewState: saga.StateCompleted,
        Metadata: map[string]interface{}{"batch": "001"},
    },
    {
        SagaID:   "saga-2",
        NewState: saga.StateCompleted,
        Metadata: map[string]interface{}{"batch": "001"},
    },
}

err := manager.BatchUpdateSagas(ctx, updates)
if err != nil {
    // 处理错误（操作会在第一个失败时停止）
}
```

### 事件监听

```go
var eventCount int
listener := func(event *state.StateChangeEvent) {
    eventCount++
    log.Printf("Event %d: %s -> %s",
        eventCount,
        event.OldState.String(),
        event.NewState.String())
}

manager.SubscribeToStateChanges(listener)
```

## 配置选项

### 存储配置

```go
config := &state.StorageConfig{
    Type:                state.StorageTypeMemory,
    SerializationFormat: state.SerializationFormatJSON,
    EnableCompression:   false,
    MaxRetentionDays:    30,
    Memory: &state.MemoryStorageConfig{
        InitialCapacity: 100,
        MaxCapacity:     0, // 无限制
        EnableMetrics:   true,
    },
}
```

### 内存存储配置

```go
memConfig := &state.MemoryStorageConfig{
    InitialCapacity: 1000,    // 初始容量
    MaxCapacity:     100000,  // 最大容量（0 = 无限制）
    EnableMetrics:   true,    // 启用指标收集
}
```

## 最佳实践

### 1. 使用 Context 超时

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := manager.SaveSaga(ctx, sagaInstance)
```

### 2. 正确处理错误

```go
saga, err := manager.GetSaga(ctx, sagaID)
if err == state.ErrSagaNotFound {
    // Saga 不存在
    return nil
} else if err != nil {
    // 其他错误
    return err
}
```

### 3. 定期清理过期 Saga

```go
// 每小时清理一次超过 30 天的 Saga
ticker := time.NewTicker(1 * time.Hour)
defer ticker.Stop()

for range ticker.C {
    cutoff := time.Now().Add(-30 * 24 * time.Hour)
    if err := manager.CleanupExpiredSagas(ctx, cutoff); err != nil {
        log.Printf("Cleanup failed: %v", err)
    }
}
```

### 4. 使用状态过滤器

```go
filter := &saga.SagaFilter{
    States:         []saga.SagaState{saga.StateRunning, saga.StatePending},
    DefinitionIDs:  []string{"order-saga", "payment-saga"},
    MinCreatedAt:   time.Now().Add(-24 * time.Hour),
}

activeSagas, err := manager.GetActiveSagas(ctx, filter)
```

### 5. 监控超时 Saga

```go
// 查找所有超时的 Saga
timeoutSagas, err := manager.GetTimeoutSagas(ctx, time.Now())
if err != nil {
    return err
}

for _, saga := range timeoutSagas {
    // 处理超时的 Saga（例如触发补偿）
    log.Printf("Saga %s has timed out", saga.GetID())
}
```

## 示例

### 完整的工作流示例

参见 `examples/` 目录中的完整示例：

- `examples/saga-state-basic/` - 基础状态管理示例
- `examples/saga-state-events/` - 事件监听示例
- `examples/saga-state-cleanup/` - 清理和维护示例

### 与 Coordinator 集成

```go
import (
    "github.com/innovationmech/swit/pkg/saga/coordinator"
    "github.com/innovationmech/swit/pkg/saga/state/storage"
)

// 创建存储
stateStorage := storage.NewMemoryStateStorage()

// 创建事件发布器
eventPublisher := NewMyEventPublisher()

// 配置协调器
config := &coordinator.OrchestratorConfig{
    StateStorage:   stateStorage,
    EventPublisher: eventPublisher,
    RetryPolicy:    saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
}

// 创建协调器
coord, err := coordinator.NewOrchestratorCoordinator(config)
if err != nil {
    panic(err)
}
defer coord.Close()

// 启动 Saga
instance, err := coord.StartSaga(ctx, definition, initialData)
if err != nil {
    panic(err)
}
```

## 性能考虑

### 内存存储性能

- **SaveSaga**: ~100,000 ops/sec
- **GetSaga**: ~500,000 ops/sec
- **UpdateSagaState**: ~80,000 ops/sec
- **GetActiveSagas**: ~10,000 ops/sec（取决于过滤条件）

详细的性能报告参见 `storage/PERFORMANCE_REPORT.md`。

### 并发安全

所有存储实现都是并发安全的：

```go
// 安全地并发操作
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        saga := createSaga(fmt.Sprintf("saga-%d", id))
        manager.SaveSaga(ctx, saga)
    }(i)
}
wg.Wait()
```

## 测试

运行测试套件：

```bash
# 运行所有测试
make test

# 运行状态包测试
go test ./pkg/saga/state/...

# 运行存储测试
go test ./pkg/saga/state/storage/...

# 运行集成测试
go test ./pkg/saga/state/storage -run TestIntegration

# 运行并发测试
go test ./pkg/saga/state/storage -run Concurrent

# 生成覆盖率报告
make test-coverage
```

## API 文档

完整的 API 文档可通过 godoc 查看：

```bash
# 启动 godoc 服务器
godoc -http=:6060

# 访问
open http://localhost:6060/pkg/github.com/innovationmech/swit/pkg/saga/state/
```

## 相关文档

- [Saga 架构文档](../../../docs/saga-user-guide.md)
- [分布式事务规范](../../../specs/saga-distributed-transactions/)
- [Coordinator 文档](../coordinator/README.md)
- [性能报告](storage/PERFORMANCE_REPORT.md)

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](../../../LICENSE) 文件。


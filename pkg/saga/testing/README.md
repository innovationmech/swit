# Saga Testing Tools

`pkg/saga/testing` 包提供了一套完整的测试工具和 Mock 实现，简化 Saga 测试编写和提高测试质量。

## 目录

- [概述](#概述)
- [核心组件](#核心组件)
- [快速开始](#快速开始)
- [Mock 实现](#mock-实现)
- [测试构建器](#测试构建器)
- [断言函数](#断言函数)
- [测试配置](#测试配置)
- [测试日志](#测试日志)
- [测试数据 Fixtures](#测试数据-fixtures)
- [预定义场景](#预定义场景)
- [最佳实践](#最佳实践)
- [示例](#示例)

## 概述

该包提供以下核心功能：

1. **Mock 实现** - StateStorage、EventPublisher、SagaStep、SagaDefinition 的 Mock
2. **测试构建器** - 流畅的 API 用于构建测试场景
3. **断言函数** - 丰富的断言辅助函数
4. **测试配置** - 灵活的测试配置管理
5. **测试日志** - 日志收集和分析工具
6. **测试 Fixtures** - 标准化的测试数据

## 核心组件

### 1. Mock 实现 (mocks.go)

提供所有核心 Saga 接口的 Mock 实现：

- `MockStateStorage` - 状态存储 Mock
- `MockEventPublisher` - 事件发布器 Mock
- `MockSagaStep` - Saga 步骤 Mock
- `MockSagaDefinition` - Saga 定义 Mock

### 2. 测试构建器 (builder.go)

提供流畅的 API 构建测试场景：

- `SagaTestBuilder` - 构建 Saga 定义
- `ScenarioBuilder` - 构建完整测试场景
- `TestScenario` - 测试场景封装
- `TestResult` - 测试结果封装

### 3. 断言函数 (assertions.go)

提供丰富的断言辅助函数，验证测试结果。

### 4. 测试配置 (config.go)

提供灵活的测试配置管理。

### 5. 测试日志 (logger.go)

提供日志收集和分析工具。

### 6. 测试数据 Fixtures (fixtures.go)

提供标准化的测试数据和加载器：

- `FixtureLoader` - Fixture 加载器
- `FixtureGenerator` - Fixture 生成器
- `FixtureBuilder` - Fixture 构建器
- 预定义的 YAML fixtures (fixtures/*.yaml)

详见 [Fixtures README](fixtures/README.md)。

## 快速开始

### 基本测试示例

```go
package mytest

import (
    "context"
    "testing"
    
    sagatesting "github.com/innovationmech/swit/pkg/saga/testing"
)

func TestSimpleSaga(t *testing.T) {
    // 创建 Saga 定义
    saga := sagatesting.NewSagaTestBuilder("test-saga", "Test Saga").
        WithTimeout(1 * time.Minute).
        AddStep("step1", "First Step").
        AddStep("step2", "Second Step").
        AddStep("step3", "Third Step").
        Build()
    
    // 创建测试场景
    scenario := sagatesting.NewScenarioBuilder("Simple Test").
        WithSaga(saga).
        WithInputData(map[string]interface{}{"test": "data"}).
        AddAssertion(sagatesting.AssertStepCount(3)).
        AddAssertion(sagatesting.AssertAllStepsCompleted()).
        Build()
    
    // 验证场景（需要结合实际的 Saga 执行）
    // ...
}
```

### 使用预定义场景

```go
func TestSuccessfulSaga(t *testing.T) {
    scenario := sagatesting.SuccessfulSagaScenario()
    
    // 执行和验证
    // ...
}
```

## Mock 实现

### MockStateStorage

模拟 Saga 状态存储，支持配置行为和响应。

```go
storage := sagatesting.NewMockStateStorage()

// 配置自定义行为
storage.SaveSagaFunc = func(ctx context.Context, s saga.SagaInstance) error {
    // 自定义逻辑
    return nil
}

// 使用默认的内存存储
err := storage.SaveSaga(ctx, sagaInstance)

// 检查调用次数
if storage.SaveSagaCalls != 1 {
    t.Errorf("Expected 1 SaveSaga call, got %d", storage.SaveSagaCalls)
}

// 重置
storage.Reset()
```

### MockEventPublisher

模拟事件发布器，追踪发布的事件。

```go
publisher := sagatesting.NewMockEventPublisher()

// 发布事件
err := publisher.PublishEvent(ctx, event)

// 检查发布的事件
events := publisher.GetEventsByType(saga.EventSagaStarted)
if len(events) != 1 {
    t.Errorf("Expected 1 SagaStarted event, got %d", len(events))
}

// 获取所有事件
allEvents := publisher.PublishedEvents

// 检查调用次数
if publisher.PublishEventCalls != 5 {
    t.Errorf("Expected 5 publish calls, got %d", publisher.PublishEventCalls)
}
```

### MockSagaStep

模拟 Saga 步骤，配置执行和补偿行为。

```go
step := sagatesting.NewMockSagaStep("step1", "Test Step")

// 配置执行函数
step.ExecuteFunc = func(ctx context.Context, data interface{}) (interface{}, error) {
    return map[string]interface{}{"result": "success"}, nil
}

// 配置补偿函数
step.CompensateFunc = func(ctx context.Context, data interface{}) error {
    return nil
}

// 配置失败
step.ExecuteError = fmt.Errorf("simulated failure")

// 执行
result, err := step.Execute(ctx, inputData)

// 检查调用次数
if step.ExecuteCalls != 1 {
    t.Errorf("Expected 1 execute call, got %d", step.ExecuteCalls)
}
```

### MockSagaDefinition

模拟 Saga 定义，包含步骤和策略。

```go
definition := sagatesting.NewMockSagaDefinition("saga1", "Test Saga")

// 添加步骤
step1 := sagatesting.NewMockSagaStep("step1", "Step 1")
step2 := sagatesting.NewMockSagaStep("step2", "Step 2")
definition.AddStep(step1)
definition.AddStep(step2)

// 配置策略
definition.TimeoutValue = 5 * time.Minute
definition.RetryPolicyValue = retryPolicy

// 使用
steps := definition.GetSteps()
timeout := definition.GetTimeout()
```

## 测试构建器

### SagaTestBuilder

流畅地构建 Saga 定义用于测试。

```go
saga := sagatesting.NewSagaTestBuilder("order-saga", "Order Processing").
    WithDescription("Process customer orders").
    WithTimeout(5 * time.Minute).
    WithRetryPolicy(retryPolicy).
    AddStep("validate", "Validate Order").
    AddStep("reserve", "Reserve Inventory").
    AddStep("payment", "Process Payment").
    AddStep("confirm", "Confirm Order").
    Build()
```

### 添加自定义步骤

```go
saga := sagatesting.NewSagaTestBuilder("test-saga", "Test").
    AddStepWithExecute("custom", "Custom Step",
        func(ctx context.Context, data interface{}) (interface{}, error) {
            // 自定义逻辑
            return data, nil
        }).
    AddStepWithCompensate("compensable", "Compensable Step",
        func(ctx context.Context, data interface{}) (interface{}, error) {
            return data, nil
        },
        func(ctx context.Context, data interface{}) error {
            // 补偿逻辑
            return nil
        }).
    AddFailingStep("fail", "Failing Step", fmt.Errorf("expected failure")).
    AddSlowStep("slow", "Slow Step", 500*time.Millisecond).
    Build()
```

### ScenarioBuilder

构建完整的测试场景。

```go
scenario := sagatesting.NewScenarioBuilder("Payment Flow Test").
    WithDescription("Test payment processing with retry").
    WithSaga(sagaDefinition).
    WithStorage(mockStorage).
    WithPublisher(mockPublisher).
    WithInputData(map[string]interface{}{
        "order_id": "ORD-123",
        "amount": 199.99,
    }).
    AddAssertion(sagatesting.AssertAllStepsCompleted()).
    AddAssertion(sagatesting.AssertEventPublished(saga.EventSagaCompleted)).
    AddAssertion(sagatesting.AssertDurationLessThan(10 * time.Second)).
    Build()
```

## 断言函数

### 步骤断言

```go
// 断言步骤数量
sagatesting.AssertStepCount(3)

// 断言所有步骤完成
sagatesting.AssertAllStepsCompleted()

// 断言特定步骤完成
sagatesting.AssertStepCompleted(0)

// 断言特定步骤失败
sagatesting.AssertStepFailed(1)

// 断言步骤状态
sagatesting.AssertStepState(2, saga.StepStateCompleted)

// 断言步骤被补偿
sagatesting.AssertStepCompensated(1)

// 断言重试次数
sagatesting.AssertRetryAttempts(1, 3)
```

### 执行断言

```go
// 断言执行成功
sagatesting.AssertExecutionSucceeded()

// 断言执行失败
sagatesting.AssertExecutionFailed()

// 断言错误内容
sagatesting.AssertError(expectedError)
sagatesting.AssertErrorContains("timeout")

// 断言执行时间
sagatesting.AssertDuration(100*time.Millisecond, 5*time.Second)
sagatesting.AssertDurationLessThan(10 * time.Second)
```

### 补偿断言

```go
// 断言没有补偿
sagatesting.AssertNoCompensation()

// 断言有补偿
sagatesting.AssertCompensationPerformed()

// 断言特定步骤补偿
sagatesting.AssertStepCompensated(0)
```

### 事件断言

```go
// 断言事件发布
sagatesting.AssertEventPublished(saga.EventSagaStarted)

// 断言事件数量
sagatesting.AssertEventCount(5)

// 断言特定类型事件数量
sagatesting.AssertEventCountByType(saga.EventStepCompleted, 3)

// 断言发布器调用次数
sagatesting.AssertPublisherCalls(5)
```

### 存储断言

```go
// 断言保存次数
sagatesting.AssertStorageSaveCount(3)

// 断言获取次数
sagatesting.AssertStorageGetCount(1)
```

### 组合断言

```go
// And - 所有断言都必须通过
sagatesting.And(
    sagatesting.AssertStepCount(3),
    sagatesting.AssertAllStepsCompleted(),
    sagatesting.AssertNoCompensation(),
)

// Or - 至少一个断言通过
sagatesting.Or(
    sagatesting.AssertExecutionSucceeded(),
    sagatesting.AssertCompensationPerformed(),
)

// Not - 断言取反
sagatesting.Not(sagatesting.AssertExecutionFailed())
```

### 自定义断言

```go
customAssertion := sagatesting.AssertCustom(
    "Payment Amount Check",
    func(result *sagatesting.TestResult) bool {
        metadata := result.SagaInstance.GetMetadata()
        amount, ok := metadata["amount"].(float64)
        return ok && amount == 199.99
    },
    "payment amount does not match",
)
```

## 测试配置

### 预定义配置

```go
// 默认配置
config := sagatesting.DefaultTestConfig()

// 快速测试配置（用于单元测试）
config := sagatesting.QuickTestConfig()

// 集成测试配置
config := sagatesting.IntegrationTestConfig()

// 性能测试配置
config := sagatesting.PerformanceTestConfig()

// 调试配置（最大调试信息）
config := sagatesting.DebugTestConfig()
```

### 自定义配置

```go
config := sagatesting.NewTestConfigBuilder().
    WithTimeout(10 * time.Minute).
    WithStepTimeout(1 * time.Minute).
    WithRetry(5, 500*time.Millisecond).
    WithConcurrency(50).
    WithVerboseLogging().
    WithMetrics().
    WithTracing().
    WithAutoCleanup(true).
    Build()

// 验证配置
if err := config.Validate(); err != nil {
    t.Fatalf("Invalid config: %v", err)
}

// 获取重试策略
retryPolicy := config.GetRetryPolicy()
```

## 测试日志

### 基本使用

```go
logger := sagatesting.NewTestLogger()

// 设置日志级别
logger.SetLevel(sagatesting.LogLevelDebug)

// 记录日志
logger.Info("Saga started")
logger.Debugf("Processing step %d", stepIndex)
logger.Error("Saga failed", map[string]interface{}{"error": err})

// 检查日志
if logger.HasErrors() {
    t.Error("Unexpected errors in logs")
}

// 获取特定级别的日志
errors := logger.GetEntriesByLevel(sagatesting.LogLevelError)
if len(errors) > 0 {
    t.Errorf("Found %d errors", len(errors))
}

// 清空日志
logger.Clear()
```

### 缓冲日志

```go
logger := sagatesting.NewBufferedLogger()

// 记录日志
logger.Info("Test message")

// 获取输出
output := logger.GetOutput()
fmt.Println(output)

// 清空缓冲区
logger.ClearBuffer()
```

### 文件日志

```go
logger, err := sagatesting.NewFileLogger("/tmp/saga-test.log")
if err != nil {
    t.Fatal(err)
}
defer logger.Close()

logger.Info("Test started")
logger.Info("Test completed")
```

### 多输出日志

```go
consoleLogger := sagatesting.NewTestLoggerWithOutput(os.Stdout)
fileLogger, _ := sagatesting.NewFileLogger("/tmp/test.log")

multiLogger := sagatesting.NewMultiLogger(consoleLogger, fileLogger)
multiLogger.Info("This goes to both console and file")
```

### 辅助函数

```go
logger := sagatesting.NewTestLogger()

// 记录步骤执行
sagatesting.LogStepExecution(logger, sagaID, "step1", 1)
sagatesting.LogStepSuccess(logger, sagaID, "step1", duration)
sagatesting.LogStepFailure(logger, sagaID, "step2", err)

// 记录补偿
sagatesting.LogCompensation(logger, sagaID, "step1")

// 记录 Saga 生命周期
sagatesting.LogSagaStarted(logger, sagaID, "Order Saga")
sagatesting.LogSagaCompleted(logger, sagaID, duration)
sagatesting.LogSagaFailed(logger, sagaID, err)
```

## 测试数据 Fixtures

### 概述

Fixtures 提供标准化的、可复用的测试数据，包括 Saga 定义、实例状态、事件、配置和错误场景等。所有 fixtures 以 YAML 格式存储在 `fixtures/` 目录。

详细文档请参阅 [Fixtures README](fixtures/README.md)。

### 加载 Fixtures

```go
// 使用默认加载器
loader := sagatesting.GetDefaultFixtureLoader()

// 加载单个 fixture
fixture, err := loader.Load("successful-saga")
if err != nil {
    t.Fatal(err)
}

// 按类型加载
sagaDefFixtures, err := loader.LoadByType(sagatesting.FixtureTypeSagaDefinition)
eventFixtures, err := loader.LoadByType(sagatesting.FixtureTypeSagaEvent)

// 按标签加载
successFixtures, err := loader.LoadByTags("success")
errorFixtures, err := loader.LoadByTags("error", "payment")
```

### 预定义 Fixtures

```go
// 成功场景
fixture, err := sagatesting.LoadSuccessfulSagaFixture()

// 失败场景
fixture, err := sagatesting.LoadFailingSagaFixture()

// 补偿场景
fixture, err := sagatesting.LoadCompensationSagaFixture()

// 超时场景
fixture, err := sagatesting.LoadTimeoutSagaFixture()
```

### 生成 Fixtures

```go
generator := sagatesting.NewFixtureGenerator()

// 生成 Saga 定义
sagaDef := generator.GenerateSagaDefinition("test-saga", "Test Saga", 5)

// 生成不同状态的实例
runningInstance := generator.GenerateSagaInstance("saga-001", "test-saga", saga.StateRunning)
completedInstance := generator.GenerateSagaInstance("saga-002", "test-saga", saga.StateCompleted)

// 生成步骤状态
stepState := generator.GenerateStepState("saga-001", 0, saga.StepStateCompleted)

// 生成事件
event := generator.GenerateEvent("saga-001", saga.EventSagaStarted)

// 生成错误
error := generator.GenerateError("TEST_ERROR", "Test error", saga.ErrorTypeService, false)
```

### 创建自定义 Fixtures

```go
// 使用 FixtureBuilder
fixture := sagatesting.NewFixtureBuilder(
    "my-saga",
    "My Saga",
    sagatesting.FixtureTypeSagaDefinition,
).
    WithDescription("Custom saga for testing").
    WithTags("custom", "test").
    WithData(myData).
    WithMetadata("author", "test-team").
    Build()

// 保存到文件
err := builder.SaveToFile("fixtures/my-saga.yaml")
```

### 可用 Fixtures 类型

- **Saga 定义**: successful-saga, failing-saga, compensation-saga, timeout-saga
- **Saga 实例**: saga-instance-running, saga-instance-completed, saga-instance-failed, saga-instance-compensating
- **事件**: saga-event-started, saga-event-step-completed, saga-event-step-failed, saga-event-compensation-started, saga-event-completed
- **配置**: config-quick-test, config-integration-test
- **错误**: error-network-timeout, error-business-logic, error-database-connection, error-compensation-failure

## 预定义场景

包提供了多个预定义的测试场景：

```go
// 成功的 Saga
scenario := sagatesting.SuccessfulSagaScenario()

// 失败的步骤
scenario := sagatesting.FailingStepScenario()

// 补偿场景
scenario := sagatesting.CompensationScenario()

// 超时场景
scenario := sagatesting.TimeoutScenario()

// 重试场景
scenario := sagatesting.RetryScenario()

// 并发场景
scenario := sagatesting.ParallelStepsScenario(10)

// 复杂场景
scenario := sagatesting.ComplexSagaScenario()
```

## 最佳实践

### 1. 使用表驱动测试

```go
func TestSagaScenarios(t *testing.T) {
    scenarios := []struct {
        name     string
        scenario *sagatesting.TestScenario
    }{
        {"Success", sagatesting.SuccessfulSagaScenario()},
        {"Failure", sagatesting.FailingStepScenario()},
        {"Compensation", sagatesting.CompensationScenario()},
        {"Timeout", sagatesting.TimeoutScenario()},
    }
    
    for _, tc := range scenarios {
        t.Run(tc.name, func(t *testing.T) {
            // 执行场景测试
            // ...
        })
    }
}
```

### 2. 隔离测试依赖

```go
func TestWithMocks(t *testing.T) {
    // 为每个测试创建新的 Mock
    storage := sagatesting.NewMockStateStorage()
    publisher := sagatesting.NewMockEventPublisher()
    
    defer func() {
        storage.Reset()
        publisher.Reset()
    }()
    
    // 运行测试
    // ...
}
```

### 3. 使用断言组合

```go
// 创建可复用的断言组合
successfulOrderProcessing := sagatesting.And(
    sagatesting.AssertStepCount(4),
    sagatesting.AssertAllStepsCompleted(),
    sagatesting.AssertEventPublished(saga.EventSagaCompleted),
    sagatesting.AssertNoCompensation(),
    sagatesting.AssertDurationLessThan(5 * time.Second),
)

// 在多个测试中使用
scenario.AddAssertion(successfulOrderProcessing)
```

### 4. 配置测试环境

```go
func setupTestEnvironment(t *testing.T) (*sagatesting.TestConfig, *sagatesting.TestLogger) {
    config := sagatesting.QuickTestConfig()
    logger := sagatesting.NewTestLogger()
    
    t.Cleanup(func() {
        if logger.HasErrors() {
            t.Log("Test logs:\n" + logger.String())
        }
    })
    
    return config, logger
}

func TestWithSetup(t *testing.T) {
    config, logger := setupTestEnvironment(t)
    // 使用 config 和 logger
}
```

### 5. 验证调用和行为

```go
func TestMockInteractions(t *testing.T) {
    storage := sagatesting.NewMockStateStorage()
    publisher := sagatesting.NewMockEventPublisher()
    
    // 执行 Saga
    // ...
    
    // 验证交互
    if storage.SaveSagaCalls != 1 {
        t.Errorf("Expected 1 SaveSaga call, got %d", storage.SaveSagaCalls)
    }
    
    if publisher.PublishEventCalls != 3 {
        t.Errorf("Expected 3 PublishEvent calls, got %d", publisher.PublishEventCalls)
    }
    
    // 验证发布的事件
    events := publisher.GetEventsByType(saga.EventStepCompleted)
    if len(events) != 2 {
        t.Errorf("Expected 2 StepCompleted events, got %d", len(events))
    }
}
```

## 示例

完整的测试示例：

```go
package mypackage_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/innovationmech/swit/pkg/saga"
    sagatesting "github.com/innovationmech/swit/pkg/saga/testing"
)

func TestOrderProcessingSaga(t *testing.T) {
    // 创建测试配置和日志
    config := sagatesting.QuickTestConfig()
    logger := sagatesting.NewTestLogger()
    
    // 创建 Mock
    storage := sagatesting.NewMockStateStorage()
    publisher := sagatesting.NewMockEventPublisher()
    
    // 构建 Saga
    orderSaga := sagatesting.NewSagaTestBuilder("order-saga", "Order Processing").
        WithDescription("Process customer order with payment").
        WithTimeout(config.DefaultTimeout).
        WithRetryPolicy(config.GetRetryPolicy()).
        AddStep("validate", "Validate Order").
        AddStep("reserve", "Reserve Inventory").
        AddStep("payment", "Process Payment").
        AddStep("confirm", "Confirm Order").
        Build()
    
    // 创建测试场景
    scenario := sagatesting.NewScenarioBuilder("Successful Order").
        WithDescription("Order processing completes successfully").
        WithSaga(orderSaga).
        WithStorage(storage).
        WithPublisher(publisher).
        WithInputData(map[string]interface{}{
            "order_id": "ORD-12345",
            "amount": 199.99,
        }).
        AddAssertion(sagatesting.AssertStepCount(4)).
        AddAssertion(sagatesting.AssertAllStepsCompleted()).
        AddAssertion(sagatesting.AssertEventPublished(saga.EventSagaStarted)).
        AddAssertion(sagatesting.AssertEventPublished(saga.EventSagaCompleted)).
        AddAssertion(sagatesting.AssertNoCompensation()).
        Build()
    
    // 执行 Saga (示例，需要实际的执行逻辑)
    // coordinator := createCoordinator(storage, publisher)
    // result := executeScenario(coordinator, scenario)
    
    // 验证断言
    // for _, assertion := range scenario.Assertions {
    //     if err := assertion(result); err != nil {
    //         t.Errorf("Assertion failed: %v", err)
    //     }
    // }
    
    // 验证日志
    if logger.HasErrors() {
        t.Error("Unexpected errors in logs")
        t.Log(logger.String())
    }
    
    // 验证 Mock 交互
    if storage.SaveSagaCalls == 0 {
        t.Error("Expected saga to be saved")
    }
    
    if publisher.GetPublishedEventCount() == 0 {
        t.Error("Expected events to be published")
    }
}
```

## 参考

- [Saga 核心接口](../interfaces.go)
- [Saga 协调器](../coordinator/)
- [Saga 状态管理](../state/)
- [Saga 重试策略](../retry/)

## 贡献

欢迎贡献新的测试工具、Mock 实现和断言函数。请遵循项目的测试规范和代码风格。


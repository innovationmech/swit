# Saga 测试指南

本指南提供了 Saga 系统测试的全面说明，包括单元测试、集成测试、性能测试和混沌测试的最佳实践。

## 目录

- [概述](#概述)
- [测试工具包](#测试工具包)
- [单元测试](#单元测试)
- [集成测试](#集成测试)
- [性能基准测试](#性能基准测试)
- [混沌测试](#混沌测试)
- [测试数据 Fixtures](#测试数据-fixtures)
- [测试配置](#测试配置)
- [最佳实践](#最佳实践)
- [CI/CD 集成](#cicd-集成)
- [常见问题](#常见问题)

## 概述

Saga 测试套件提供了完整的测试工具和框架，确保分布式事务的可靠性和性能。测试套件包括：

- **单元测试** - 测试单个组件和函数
- **集成测试** - 测试组件之间的交互
- **性能测试** - 验证吞吐量和延迟指标
- **混沌测试** - 模拟故障场景验证容错性
- **测试工具** - Mock、Fixtures、断言和构建器

### 测试覆盖率

当前 Saga 系统的测试覆盖率：

| 模块 | 覆盖率 | 说明 |
|-----|-------|------|
| saga core | 96.0% | 核心 Saga 接口和类型 |
| coordinator | 87.4% | Orchestrator 协调器 |
| dsl | 85.5% | DSL 解析和验证 |
| examples | 85.7% | 示例代码 |
| messaging | 86.1% | 消息发布和订阅 |
| monitoring | 75.1% | 监控和指标 |
| retry | 91.6% | 重试策略 |
| security | 86.2% | 安全和认证 |
| state | 75.1% | 状态管理 |
| state/storage | 46.9% | 状态存储实现 |
| testing | 44.6% | 测试工具包 |

**总体覆盖率: 73.6%**

## 测试工具包

Saga 测试工具包位于 `pkg/saga/testing`，提供以下核心组件：

### 1. Mock 实现 (mocks.go)

完整的 Mock 实现，支持自定义行为和验证：

```go
import sagatesting "github.com/innovationmech/swit/pkg/saga/testing"

// MockStateStorage - 状态存储 Mock
storage := sagatesting.NewMockStateStorage()
storage.SaveSagaFunc = func(ctx context.Context, s saga.SagaInstance) error {
    // 自定义逻辑
    return nil
}

// MockEventPublisher - 事件发布器 Mock
publisher := sagatesting.NewMockEventPublisher()

// MockSagaStep - Saga 步骤 Mock
step := sagatesting.NewMockSagaStep("step1", "Test Step")
step.ExecuteFunc = func(ctx context.Context, data interface{}) (interface{}, error) {
    return map[string]interface{}{"result": "success"}, nil
}

// MockSagaDefinition - Saga 定义 Mock
definition := sagatesting.NewMockSagaDefinition("saga1", "Test Saga")
definition.AddStep(step)
```

### 2. 测试构建器 (builder.go)

流畅的 API 构建测试场景：

```go
// 构建 Saga 定义
saga := sagatesting.NewSagaTestBuilder("order-saga", "Order Processing").
    WithTimeout(5 * time.Minute).
    WithRetryPolicy(retryPolicy).
    AddStep("validate", "Validate Order").
    AddStep("reserve", "Reserve Inventory").
    AddStep("payment", "Process Payment").
    Build()

// 构建测试场景
scenario := sagatesting.NewScenarioBuilder("Payment Flow Test").
    WithSaga(saga).
    WithStorage(mockStorage).
    WithPublisher(mockPublisher).
    WithInputData(map[string]interface{}{
        "order_id": "ORD-123",
        "amount": 199.99,
    }).
    AddAssertion(sagatesting.AssertAllStepsCompleted()).
    Build()
```

### 3. 断言函数 (assertions.go)

丰富的断言库验证测试结果：

```go
// 步骤断言
sagatesting.AssertStepCount(3)
sagatesting.AssertAllStepsCompleted()
sagatesting.AssertStepCompleted(0)
sagatesting.AssertStepFailed(1)

// 执行断言
sagatesting.AssertExecutionSucceeded()
sagatesting.AssertExecutionFailed()
sagatesting.AssertDurationLessThan(10 * time.Second)

// 补偿断言
sagatesting.AssertNoCompensation()
sagatesting.AssertCompensationPerformed()
sagatesting.AssertStepCompensated(0)

// 事件断言
sagatesting.AssertEventPublished(saga.EventSagaStarted)
sagatesting.AssertEventCount(5)

// 组合断言
sagatesting.And(
    sagatesting.AssertStepCount(3),
    sagatesting.AssertAllStepsCompleted(),
    sagatesting.AssertNoCompensation(),
)
```

### 4. 测试配置 (config.go)

预定义和自定义测试配置：

```go
// 预定义配置
config := sagatesting.QuickTestConfig()        // 快速单元测试
config := sagatesting.IntegrationTestConfig()  // 集成测试
config := sagatesting.PerformanceTestConfig()  // 性能测试
config := sagatesting.DebugTestConfig()        // 调试配置

// 自定义配置
config := sagatesting.NewTestConfigBuilder().
    WithTimeout(10 * time.Minute).
    WithStepTimeout(1 * time.Minute).
    WithRetry(5, 500*time.Millisecond).
    WithConcurrency(50).
    WithVerboseLogging().
    Build()
```

### 5. 测试日志 (logger.go)

日志收集和分析工具：

```go
logger := sagatesting.NewTestLogger()
logger.SetLevel(sagatesting.LogLevelDebug)

logger.Info("Saga started")
logger.Debugf("Processing step %d", stepIndex)

// 检查日志
if logger.HasErrors() {
    t.Error("Unexpected errors in logs")
}

// 获取特定级别的日志
errors := logger.GetEntriesByLevel(sagatesting.LogLevelError)
```

### 6. 测试数据 Fixtures (fixtures.go)

标准化的可复用测试数据：

```go
// 加载预定义 Fixtures
loader := sagatesting.GetDefaultFixtureLoader()
fixture, err := loader.Load("successful-saga")

// 按类型加载
sagaDefFixtures, err := loader.LoadByType(sagatesting.FixtureTypeSagaDefinition)

// 按标签加载
successFixtures, err := loader.LoadByTags("success")
errorFixtures, err := loader.LoadByTags("error", "payment")
```

详细信息请参阅 [测试数据 Fixtures](#测试数据-fixtures) 章节。

## 单元测试

单元测试验证单个组件的功能。

### 测试 Saga 步骤执行

```go
func TestStepExecution(t *testing.T) {
    // 创建测试步骤
    step := sagatesting.NewMockSagaStep("payment", "Process Payment")
    
    // 配置执行行为
    step.ExecuteFunc = func(ctx context.Context, data interface{}) (interface{}, error) {
        orderData := data.(map[string]interface{})
        amount := orderData["amount"].(float64)
        
        if amount <= 0 {
            return nil, fmt.Errorf("invalid amount")
        }
        
        return map[string]interface{}{
            "transaction_id": "TXN-123",
            "status": "success",
        }, nil
    }
    
    // 测试成功场景
    ctx := context.Background()
    result, err := step.Execute(ctx, map[string]interface{}{
        "amount": 199.99,
    })
    
    require.NoError(t, err)
    assert.Equal(t, "TXN-123", result.(map[string]interface{})["transaction_id"])
    
    // 测试失败场景
    result, err = step.Execute(ctx, map[string]interface{}{
        "amount": -10.0,
    })
    
    require.Error(t, err)
    assert.Contains(t, err.Error(), "invalid amount")
}
```

### 测试补偿逻辑

```go
func TestCompensation(t *testing.T) {
    step := sagatesting.NewMockSagaStep("inventory", "Reserve Inventory")
    
    // 配置补偿函数
    var compensated bool
    step.CompensateFunc = func(ctx context.Context, data interface{}) error {
        compensated = true
        return nil
    }
    
    // 执行补偿
    ctx := context.Background()
    err := step.Compensate(ctx, map[string]interface{}{"sku": "PROD-001"})
    
    require.NoError(t, err)
    assert.True(t, compensated, "Compensation should be called")
    assert.Equal(t, 1, step.CompensateCalls, "Compensation should be called once")
}
```

### 测试重试策略

```go
func TestRetryPolicy(t *testing.T) {
    tests := []struct {
        name          string
        maxAttempts   int
        delay         time.Duration
        failureCount  int
        expectSuccess bool
    }{
        {
            name:          "Success on first try",
            maxAttempts:   3,
            delay:         100 * time.Millisecond,
            failureCount:  0,
            expectSuccess: true,
        },
        {
            name:          "Success after retry",
            maxAttempts:   3,
            delay:         100 * time.Millisecond,
            failureCount:  2,
            expectSuccess: true,
        },
        {
            name:          "Failure after max retries",
            maxAttempts:   2,
            delay:         100 * time.Millisecond,
            failureCount:  3,
            expectSuccess: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            attemptCount := 0
            step := sagatesting.NewMockSagaStep("retry-step", "Retry Test")
            
            step.ExecuteFunc = func(ctx context.Context, data interface{}) (interface{}, error) {
                attemptCount++
                if attemptCount <= tt.failureCount {
                    return nil, fmt.Errorf("temporary failure")
                }
                return "success", nil
            }
            
            // 执行带重试
            ctx := context.Background()
            result, err := executeWithRetry(ctx, step, tt.maxAttempts, tt.delay)
            
            if tt.expectSuccess {
                require.NoError(t, err)
                assert.Equal(t, "success", result)
            } else {
                require.Error(t, err)
            }
            
            expectedAttempts := tt.failureCount + 1
            if expectedAttempts > tt.maxAttempts {
                expectedAttempts = tt.maxAttempts
            }
            assert.Equal(t, expectedAttempts, attemptCount)
        })
    }
}
```

### 表驱动测试

```go
func TestSagaScenarios(t *testing.T) {
    scenarios := []struct {
        name       string
        steps      []string
        failAt     int  // -1 表示不失败
        expectComp bool // 期望补偿
    }{
        {
            name:       "All steps succeed",
            steps:      []string{"step1", "step2", "step3"},
            failAt:     -1,
            expectComp: false,
        },
        {
            name:       "Second step fails",
            steps:      []string{"step1", "step2", "step3"},
            failAt:     1,
            expectComp: true,
        },
        {
            name:       "Last step fails",
            steps:      []string{"step1", "step2", "step3"},
            failAt:     2,
            expectComp: true,
        },
    }
    
    for _, tc := range scenarios {
        t.Run(tc.name, func(t *testing.T) {
            // 构建 Saga
            builder := sagatesting.NewSagaTestBuilder("test-saga", tc.name)
            
            for i, stepName := range tc.steps {
                if i == tc.failAt {
                    builder.AddFailingStep(stepName, stepName, fmt.Errorf("planned failure"))
                } else {
                    builder.AddStep(stepName, stepName)
                }
            }
            
            saga := builder.Build()
            
            // 执行测试
            // ... 执行 saga 逻辑
            
            // 验证结果
            if tc.expectComp {
                sagatesting.AssertCompensationPerformed()
            } else {
                sagatesting.AssertNoCompensation()
            }
        })
    }
}
```

## 集成测试

集成测试验证多个组件的协同工作。

### 端到端 Saga 执行

```go
func TestE2ESagaExecution(t *testing.T) {
    // 创建真实的依赖
    storage := memory.NewInMemoryStateStorage()
    publisher := memory.NewInMemoryEventPublisher()
    
    // 创建 Orchestrator
    orchestrator := coordinator.NewOrchestratorCoordinator(
        storage,
        publisher,
        coordinator.WithMaxConcurrentSagas(10),
    )
    
    // 定义 Saga
    sagaDef := &saga.Definition{
        ID:          "order-saga",
        Name:        "Order Processing",
        Description: "Process customer order",
        Steps: []saga.Step{
            &ValidateOrderStep{},
            &ReserveInventoryStep{},
            &ProcessPaymentStep{},
            &ConfirmOrderStep{},
        },
    }
    
    // 启动 Saga
    ctx := context.Background()
    instance, err := orchestrator.StartSaga(ctx, sagaDef, map[string]interface{}{
        "order_id": "ORD-12345",
        "items": []string{"ITEM-001", "ITEM-002"},
        "amount": 299.99,
    })
    
    require.NoError(t, err)
    
    // 等待完成
    err = orchestrator.WaitForCompletion(ctx, instance.ID, 30*time.Second)
    require.NoError(t, err)
    
    // 验证最终状态
    finalInstance, err := storage.GetSaga(ctx, instance.ID)
    require.NoError(t, err)
    assert.Equal(t, saga.StateCompleted, finalInstance.State)
    
    // 验证所有步骤完成
    for i, step := range finalInstance.Steps {
        assert.Equal(t, saga.StepStateCompleted, step.State, 
            "Step %d (%s) should be completed", i, step.ID)
    }
    
    // 验证事件发布
    events := publisher.GetEventsByType(saga.EventSagaCompleted)
    assert.Len(t, events, 1, "Should publish saga completed event")
}
```

### 故障恢复测试

```go
func TestSagaRecovery(t *testing.T) {
    storage := memory.NewInMemoryStateStorage()
    publisher := memory.NewInMemoryEventPublisher()
    
    orchestrator := coordinator.NewOrchestratorCoordinator(storage, publisher)
    
    // 创建失败的步骤
    sagaDef := sagatesting.NewSagaTestBuilder("recovery-saga", "Recovery Test").
        AddStep("step1", "Step 1").
        AddFailingStep("step2", "Step 2 (Fails)", fmt.Errorf("simulated failure")).
        AddStep("step3", "Step 3").
        Build()
    
    // 启动 Saga
    ctx := context.Background()
    instance, err := orchestrator.StartSaga(ctx, sagaDef, nil)
    require.NoError(t, err)
    
    // 等待失败
    time.Sleep(1 * time.Second)
    
    // 验证状态
    currentInstance, err := storage.GetSaga(ctx, instance.ID)
    require.NoError(t, err)
    assert.Equal(t, saga.StateFailed, currentInstance.State)
    
    // 验证补偿执行
    assert.Equal(t, saga.StepStateCompensated, currentInstance.Steps[0].State,
        "First step should be compensated")
    
    // 验证事件
    compensationEvents := publisher.GetEventsByType(saga.EventCompensationStarted)
    assert.NotEmpty(t, compensationEvents, "Should publish compensation events")
}
```

### 并发 Saga 执行

```go
func TestConcurrentSagaExecution(t *testing.T) {
    storage := memory.NewInMemoryStateStorage()
    publisher := memory.NewInMemoryEventPublisher()
    
    orchestrator := coordinator.NewOrchestratorCoordinator(
        storage,
        publisher,
        coordinator.WithMaxConcurrentSagas(50),
    )
    
    sagaDef := sagatesting.NewSagaTestBuilder("concurrent-saga", "Concurrent Test").
        AddStep("step1", "Step 1").
        AddStep("step2", "Step 2").
        AddStep("step3", "Step 3").
        Build()
    
    // 并发启动多个 Saga
    const numSagas = 100
    var wg sync.WaitGroup
    var mu sync.Mutex
    successCount := 0
    failureCount := 0
    
    ctx := context.Background()
    startTime := time.Now()
    
    for i := 0; i < numSagas; i++ {
        wg.Add(1)
        go func(index int) {
            defer wg.Done()
            
            instance, err := orchestrator.StartSaga(ctx, sagaDef, map[string]interface{}{
                "saga_index": index,
            })
            
            if err != nil {
                mu.Lock()
                failureCount++
                mu.Unlock()
                return
            }
            
            // 等待完成
            err = orchestrator.WaitForCompletion(ctx, instance.ID, 30*time.Second)
            
            mu.Lock()
            if err == nil {
                successCount++
            } else {
                failureCount++
            }
            mu.Unlock()
        }(i)
    }
    
    wg.Wait()
    duration := time.Since(startTime)
    
    // 验证结果
    t.Logf("Completed %d sagas in %v", numSagas, duration)
    t.Logf("Success: %d, Failure: %d", successCount, failureCount)
    
    assert.Equal(t, numSagas, successCount, "All sagas should complete successfully")
    assert.Equal(t, 0, failureCount, "No sagas should fail")
    
    // 验证性能
    throughput := float64(numSagas) / duration.Seconds()
    t.Logf("Throughput: %.2f sagas/sec", throughput)
    
    assert.Greater(t, throughput, 10.0, "Should process at least 10 sagas/sec")
}
```

## 性能基准测试

性能基准测试验证系统的吞吐量和延迟特性。详细信息请参阅 [性能基准报告](saga-performance-benchmarks.md)。

### 运行基准测试

```bash
# 运行所有基准测试
cd pkg/saga/testing/benchmarks
go test -bench=. -benchmem -benchtime=5s

# 运行特定基准测试
go test -bench=BenchmarkOrchestrator -benchmem

# 生成性能分析
go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof
go tool pprof cpu.prof
```

### 编写自定义基准测试

```go
func BenchmarkSagaExecution(b *testing.B) {
    storage := memory.NewInMemoryStateStorage()
    publisher := memory.NewInMemoryEventPublisher()
    orchestrator := coordinator.NewOrchestratorCoordinator(storage, publisher)
    
    sagaDef := sagatesting.NewSagaTestBuilder("bench-saga", "Benchmark").
        AddStep("step1", "Step 1").
        AddStep("step2", "Step 2").
        AddStep("step3", "Step 3").
        Build()
    
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        instance, err := orchestrator.StartSaga(ctx, sagaDef, nil)
        if err != nil {
            b.Fatal(err)
        }
        
        err = orchestrator.WaitForCompletion(ctx, instance.ID, 10*time.Second)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// 并发基准测试
func BenchmarkSagaExecutionParallel(b *testing.B) {
    storage := memory.NewInMemoryStateStorage()
    publisher := memory.NewInMemoryEventPublisher()
    orchestrator := coordinator.NewOrchestratorCoordinator(storage, publisher)
    
    sagaDef := sagatesting.NewSagaTestBuilder("bench-saga", "Benchmark").
        AddStep("step1", "Step 1").
        AddStep("step2", "Step 2").
        Build()
    
    b.RunParallel(func(pb *testing.PB) {
        ctx := context.Background()
        for pb.Next() {
            instance, err := orchestrator.StartSaga(ctx, sagaDef, nil)
            if err != nil {
                b.Fatal(err)
            }
            
            err = orchestrator.WaitForCompletion(ctx, instance.ID, 10*time.Second)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

## 混沌测试

混沌测试通过主动注入故障来验证系统的容错能力。详细信息请参阅 `pkg/saga/testing/chaos/README.md`。

### 故障注入器

```go
import "github.com/innovationmech/swit/pkg/saga/testing/chaos"

// 创建故障注入器
injector := chaos.NewFaultInjector()

// 添加网络故障
injector.AddFault("network-fault", &chaos.FaultConfig{
    Type:        chaos.FaultTypeNetworkPartition,
    Probability: 0.3,  // 30% 概率
    TargetStep:  "payment-step",
})

// 添加超时故障
injector.AddFault("timeout-fault", &chaos.FaultConfig{
    Type:        chaos.FaultTypeTimeout,
    Probability: 0.2,
    Duration:    5 * time.Second,
})

// 包装组件
chaosStorage := chaos.NewChaosStateStorage(storage, injector)
chaosPublisher := chaos.NewChaosEventPublisher(publisher, injector)
```

### 网络故障测试

```go
func TestNetworkPartition(t *testing.T) {
    injector := chaos.NewFaultInjector()
    
    // 配置网络分区故障
    injector.AddFault("network", &chaos.FaultConfig{
        Type:        chaos.FaultTypeNetworkPartition,
        Probability: 1.0,  // 100% 触发
        TargetStep:  "step1",
    })
    
    storage := memory.NewInMemoryStateStorage()
    chaosStorage := chaos.NewChaosStateStorage(storage, injector)
    
    publisher := memory.NewInMemoryEventPublisher()
    orchestrator := coordinator.NewOrchestratorCoordinator(chaosStorage, publisher)
    
    sagaDef := sagatesting.NewSagaTestBuilder("chaos-saga", "Chaos Test").
        AddStep("step1", "Step 1").
        AddStep("step2", "Step 2").
        Build()
    
    // 启动 Saga
    ctx := context.Background()
    instance, err := orchestrator.StartSaga(ctx, sagaDef, nil)
    
    // 可能因网络故障而失败
    if err != nil {
        assert.Contains(t, err.Error(), "network partition")
        return
    }
    
    // 等待完成或失败
    time.Sleep(2 * time.Second)
    
    // 验证系统处理了故障
    finalInstance, err := storage.GetSaga(ctx, instance.ID)
    if err == nil {
        // Saga 应该处于终止状态
        assert.True(t, finalInstance.IsTerminal(), 
            "Saga should reach terminal state despite network failures")
    }
}
```

### 服务崩溃测试

```go
func TestServiceCrash(t *testing.T) {
    injector := chaos.NewFaultInjector()
    
    injector.AddFault("crash", &chaos.FaultConfig{
        Type:        chaos.FaultTypeServiceCrash,
        Probability: 1.0,
        TargetStep:  "step2",
    })
    
    storage := memory.NewInMemoryStateStorage()
    publisher := memory.NewInMemoryEventPublisher()
    
    sagaDef := sagatesting.NewSagaTestBuilder("crash-saga", "Crash Test").
        AddStep("step1", "Step 1").
        AddStep("step2", "Step 2 (Will Crash)").
        AddStep("step3", "Step 3").
        Build()
    
    orchestrator := coordinator.NewOrchestratorCoordinator(storage, publisher)
    
    // 包装步骤
    for i, step := range sagaDef.Steps {
        sagaDef.Steps[i] = chaos.WrapStepWithInjector(step, injector)
    }
    
    // 启动 Saga
    ctx := context.Background()
    instance, err := orchestrator.StartSaga(ctx, sagaDef, nil)
    require.NoError(t, err)
    
    // 等待处理
    time.Sleep(2 * time.Second)
    
    // 验证故障处理
    finalInstance, err := storage.GetSaga(ctx, instance.ID)
    require.NoError(t, err)
    
    // 应该触发补偿
    assert.Equal(t, saga.StateFailed, finalInstance.State)
    assert.Equal(t, saga.StepStateCompensated, finalInstance.Steps[0].State)
}
```

## 测试数据 Fixtures

Fixtures 提供标准化的测试数据。详细信息请参阅 `pkg/saga/testing/fixtures/README.md`。

### 可用 Fixtures

#### Saga 定义

- `successful-saga` - 成功的订单处理
- `failing-saga` - 支付失败场景
- `compensation-saga` - 订单取消与补偿
- `timeout-saga` - 支付超时场景

#### Saga 实例状态

- `saga-instance-running` - 运行中的实例
- `saga-instance-completed` - 已完成的实例
- `saga-instance-failed` - 失败的实例
- `saga-instance-compensating` - 补偿中的实例

#### 事件

- `saga-event-started` - Saga 启动事件
- `saga-event-step-completed` - 步骤完成事件
- `saga-event-step-failed` - 步骤失败事件
- `saga-event-compensation-started` - 补偿开始事件
- `saga-event-completed` - Saga 完成事件

#### 配置

- `config-quick-test` - 快速测试配置
- `config-integration-test` - 集成测试配置

#### 错误场景

- `error-network-timeout` - 网络超时
- `error-business-logic` - 业务逻辑错误
- `error-database-connection` - 数据库连接错误
- `error-compensation-failure` - 补偿失败

### 使用 Fixtures

```go
// 加载单个 fixture
loader := sagatesting.GetDefaultFixtureLoader()
fixture, err := loader.Load("successful-saga")
require.NoError(t, err)

// 按类型加载
sagaDefs, err := loader.LoadByType(sagatesting.FixtureTypeSagaDefinition)

// 按标签加载
successFixtures, err := loader.LoadByTags("success")
errorFixtures, err := loader.LoadByTags("error", "payment")

// 使用预定义加载函数
successFixture, err := sagatesting.LoadSuccessfulSagaFixture()
failureFixture, err := sagatesting.LoadFailingSagaFixture()
compensationFixture, err := sagatesting.LoadCompensationSagaFixture()
```

### 创建自定义 Fixtures

```go
// 使用 FixtureBuilder
fixture := sagatesting.NewFixtureBuilder(
    "custom-saga",
    "Custom Saga",
    sagatesting.FixtureTypeSagaDefinition,
).
    WithDescription("Custom saga for specific test").
    WithTags("custom", "test").
    WithData(myData).
    Build()

// 保存到文件
err := fixture.SaveToFile("pkg/saga/testing/fixtures/custom-saga.yaml")
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
```

## 最佳实践

### 1. 使用表驱动测试

表驱动测试提高代码复用和可维护性：

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

为每个测试创建新的 Mock，避免状态污染：

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

### 3. 验证调用和行为

验证 Mock 的交互：

```go
func TestMockInteractions(t *testing.T) {
    storage := sagatesting.NewMockStateStorage()
    publisher := sagatesting.NewMockEventPublisher()
    
    // 执行 Saga
    // ...
    
    // 验证交互
    assert.Equal(t, 1, storage.SaveSagaCalls, "Should save saga once")
    assert.Equal(t, 3, publisher.PublishEventCalls, "Should publish 3 events")
    
    // 验证发布的事件
    events := publisher.GetEventsByType(saga.EventStepCompleted)
    assert.Len(t, events, 2, "Should publish 2 step completed events")
}
```

### 4. 使用断言组合

创建可复用的断言组合：

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

### 5. 配置测试环境

使用辅助函数设置和清理测试环境：

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

### 6. 渐进式故障注入

从低概率开始，逐步增加故障注入：

```go
// 开始时低概率
injector.AddFault("fault-1", &chaos.FaultConfig{Probability: 0.1})

// 验证通过后增加概率
injector.AddFault("fault-2", &chaos.FaultConfig{Probability: 0.5})
```

### 7. 测试覆盖率目标

- 单元测试覆盖率 > 80%
- 集成测试覆盖主要场景
- 性能测试建立基线和阈值
- 混沌测试验证容错性

## CI/CD 集成

### GitHub Actions 集成

```yaml
name: Saga Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      - name: Run Unit Tests
        run: go test -v -coverprofile=coverage.out ./pkg/saga/...
      
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
  
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      - name: Run Integration Tests
        run: go test -v -tags=integration ./pkg/saga/integration/...
  
  performance-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      - name: Run Benchmark Tests
        run: go test -bench=. -benchmem ./pkg/saga/testing/benchmarks/...
```

### Makefile 集成

```makefile
.PHONY: test-saga
test-saga:
	@echo "Running Saga tests..."
	go test -v -coverprofile=saga_coverage.out ./pkg/saga/...

.PHONY: test-saga-integration
test-saga-integration:
	@echo "Running Saga integration tests..."
	go test -v -tags=integration ./pkg/saga/integration/...

.PHONY: test-saga-benchmark
test-saga-benchmark:
	@echo "Running Saga benchmark tests..."
	go test -bench=. -benchmem -benchtime=5s ./pkg/saga/testing/benchmarks/...

.PHONY: test-saga-chaos
test-saga-chaos:
	@echo "Running Saga chaos tests..."
	go test -v ./pkg/saga/testing/chaos/...

.PHONY: test-saga-coverage
test-saga-coverage:
	@echo "Generating Saga test coverage report..."
	go test -coverprofile=saga_coverage.out ./pkg/saga/...
	go tool cover -html=saga_coverage.out -o saga_coverage.html
	@echo "Coverage report: saga_coverage.html"
```

## 常见问题

### Q: 如何调试失败的测试？

A: 启用详细日志记录：

```go
logger := sagatesting.NewTestLogger()
logger.SetLevel(sagatesting.LogLevelDebug)

// 或使用调试配置
config := sagatesting.DebugTestConfig()
```

### Q: 测试不稳定怎么办？

A: 检查以下几点：
- 确保等待时间足够
- 验证并发访问控制
- 使用确定性的测试数据
- 检查资源清理

### Q: 如何提高测试性能？

A: 使用以下策略：
- 使用 `QuickTestConfig()` 进行快速单元测试
- 并行运行独立测试
- 使用内存存储而非真实数据库
- 减少不必要的等待时间

### Q: 如何测试超时场景？

A: 使用短超时和慢步骤：

```go
config := sagatesting.QuickTestConfig()
config.DefaultTimeout = 1 * time.Second
config.StepTimeout = 500 * time.Millisecond

saga := sagatesting.NewSagaTestBuilder("timeout-saga", "Timeout Test").
    WithTimeout(config.DefaultTimeout).
    AddSlowStep("slow", "Slow Step", 2*time.Second).
    Build()
```

### Q: 如何模拟网络故障？

A: 使用混沌测试的故障注入器：

```go
injector := chaos.NewFaultInjector()
injector.AddFault("network", &chaos.FaultConfig{
    Type:        chaos.FaultTypeNetworkPartition,
    Probability: 0.3,
})

chaosStorage := chaos.NewChaosStateStorage(storage, injector)
```

## 相关文档

- [测试覆盖率报告](saga-test-coverage.md)
- [性能基准报告](saga-performance-benchmarks.md)
- [Saga 用户指南](saga-user-guide.md)
- [Saga 监控指南](saga-monitoring-guide.md)
- [性能监控和告警](performance-monitoring-alerting.md)

## 贡献

欢迎贡献新的测试用例、工具和最佳实践！请：

1. 遵循现有的测试规范
2. 提供清晰的文档
3. 确保测试稳定可靠
4. 更新相关文档

## 许可证

Copyright © 2025 jackelyj <dreamerlyj@gmail.com>

本项目采用 MIT 许可证。


# Saga Test Fixtures

本目录包含标准化的测试数据 fixtures，为 Saga 测试提供一致、可复用的测试数据。

## 目录

- [概述](#概述)
- [Fixture 类型](#fixture-类型)
- [可用 Fixtures](#可用-fixtures)
- [使用方式](#使用方式)
- [创建自定义 Fixtures](#创建自定义-fixtures)

## 概述

Fixtures 是预定义的测试数据，用于：

- **一致性**: 确保测试使用相同的基准数据
- **可复用性**: 在多个测试中重复使用相同的场景
- **可维护性**: 集中管理测试数据，便于更新
- **文档化**: YAML 格式易于阅读和理解

## Fixture 类型

### 1. Saga 定义 (saga_definition)

定义完整的 Saga 流程，包括步骤、超时、重试策略等。

### 2. Saga 实例状态 (saga_instance)

不同执行状态下的 Saga 实例快照。

### 3. 步骤状态 (step_state)

单个步骤的状态数据。

### 4. Saga 事件 (saga_event)

Saga 生命周期中产生的各类事件。

### 5. 配置 (config)

测试环境的配置数据。

### 6. 错误场景 (error)

各种错误和异常情况。

## 可用 Fixtures

### Saga 定义

| ID | 名称 | 描述 | 标签 |
|----|------|------|------|
| `successful-saga` | 成功的订单处理 | 所有步骤成功完成的 Saga | success, order, complete |
| `failing-saga` | 支付失败 | 支付步骤失败的 Saga | failure, payment, error |
| `compensation-saga` | 订单取消与补偿 | 触发补偿的 Saga | compensation, rollback, order |
| `timeout-saga` | 支付超时 | 超时场景的 Saga | timeout, payment, slow |

### Saga 实例状态

| ID | 名称 | 状态 | 描述 | 标签 |
|----|------|------|------|------|
| `saga-instance-running` | 运行中的订单 | running | 正在执行的 Saga | running, in_progress, state |
| `saga-instance-completed` | 已完成的订单 | completed | 成功完成的 Saga | completed, success, state |
| `saga-instance-failed` | 失败的支付 | failed | 支付失败的 Saga | failed, error, state |
| `saga-instance-compensating` | 补偿中 | compensating | 正在执行补偿的 Saga | compensating, rollback, state |

### Saga 事件

| ID | 名称 | 事件类型 | 描述 | 标签 |
|----|------|----------|------|------|
| `saga-event-started` | Saga 启动 | saga_started | Saga 开始执行 | event, started, lifecycle |
| `saga-event-step-completed` | 步骤完成 | step_completed | 步骤成功完成 | event, step, completed |
| `saga-event-step-failed` | 步骤失败 | step_failed | 步骤执行失败 | event, step, failed, error |
| `saga-event-compensation-started` | 补偿开始 | compensation_started | 开始执行补偿 | event, compensation, started |
| `saga-event-completed` | Saga 完成 | saga_completed | Saga 成功完成 | event, completed, success, lifecycle |

### 配置

| ID | 名称 | 用途 | 标签 |
|----|------|------|------|
| `config-quick-test` | 快速测试配置 | 单元测试 | config, quick, unit_test |
| `config-integration-test` | 集成测试配置 | 集成测试 | config, integration, full_stack |

### 错误场景

| ID | 名称 | 错误类型 | 标签 |
|----|------|----------|------|
| `error-network-timeout` | 网络超时 | timeout | error, timeout, network |
| `error-business-logic` | 业务逻辑错误 | validation | error, validation, business_logic |
| `error-database-connection` | 数据库连接错误 | infrastructure | error, database, infrastructure |
| `error-compensation-failure` | 补偿失败 | compensation | error, compensation, critical |

## 使用方式

### 基本加载

```go
import sagatesting "github.com/innovationmech/swit/pkg/saga/testing"

// 使用默认加载器
loader := sagatesting.GetDefaultFixtureLoader()

// 加载单个 fixture
fixture, err := loader.Load("successful-saga")
if err != nil {
    t.Fatal(err)
}

// 使用 fixture 数据
sagaDef := fixture.Data.(sagatesting.SagaDefinitionFixture)
```

### 预定义加载函数

```go
// 加载成功场景
fixture, err := sagatesting.LoadSuccessfulSagaFixture()

// 加载失败场景
fixture, err := sagatesting.LoadFailingSagaFixture()

// 加载补偿场景
fixture, err := sagatesting.LoadCompensationSagaFixture()

// 加载超时场景
fixture, err := sagatesting.LoadTimeoutSagaFixture()
```

### 按类型加载

```go
loader := sagatesting.GetDefaultFixtureLoader()

// 加载所有事件 fixtures
events, err := loader.LoadByType(sagatesting.FixtureTypeSagaEvent)

// 加载所有错误 fixtures
errors, err := loader.LoadByType(sagatesting.FixtureTypeError)
```

### 按标签加载

```go
loader := sagatesting.GetDefaultFixtureLoader()

// 加载所有带 "success" 标签的 fixtures
successFixtures, err := loader.LoadByTags("success")

// 加载带多个标签的 fixtures
fixtures, err := loader.LoadByTags("error", "payment")
```

### 从文件系统加载

```go
// 指定自定义 fixtures 目录
loader := sagatesting.NewFixtureLoader("/path/to/custom/fixtures")

fixture, err := loader.Load("custom-saga")
```

### 转换为 Mock

```go
// 加载 Saga 定义 fixture
fixture, err := loader.Load("successful-saga")
if err != nil {
    t.Fatal(err)
}

// 解析为 SagaDefinitionFixture
var sagaDef sagatesting.SagaDefinitionFixture
data, _ := yaml.Marshal(fixture.Data)
yaml.Unmarshal(data, &sagaDef)

// 转换为 Mock
mock, err := sagatesting.CreateMockFromFixture(&sagaDef)
if err != nil {
    t.Fatal(err)
}

// 在测试中使用
// ...
```

### 在测试场景中使用

```go
func TestOrderProcessing(t *testing.T) {
    // 加载 fixture
    fixture, err := sagatesting.LoadSuccessfulSagaFixture()
    require.NoError(t, err)

    // 创建测试场景
    scenario := sagatesting.NewScenarioBuilder("Order Processing").
        WithDescription("Test successful order processing").
        WithSaga(createSagaFromFixture(fixture)).
        WithInputData(map[string]interface{}{
            "order_id": "TEST-001",
            "amount": 299.99,
        }).
        AddAssertion(sagatesting.AssertAllStepsCompleted()).
        Build()

    // 执行测试
    // ...
}
```

## 创建自定义 Fixtures

### 使用 Fixture Builder

```go
import sagatesting "github.com/innovationmech/swit/pkg/saga/testing"

// 创建自定义 fixture
fixture := sagatesting.NewFixtureBuilder(
    "my-custom-saga",
    "My Custom Saga",
    sagatesting.FixtureTypeSagaDefinition,
).
    WithDescription("A custom saga for specific testing").
    WithTags("custom", "special").
    WithData(myCustomData).
    WithMetadata("author", "test-team").
    Build()

// 保存到文件
err := fixture.SaveToFile("fixtures/my-custom-saga.yaml")
```

### 手动创建 YAML 文件

创建新的 `.yaml` 文件在 `fixtures/` 目录:

```yaml
id: my-custom-saga
name: My Custom Saga
description: Description of the saga
type: saga_definition
tags:
  - custom
  - tag1
  - tag2
metadata:
  scenario: custom_scenario
  complexity: simple

data:
  id: custom-saga-001
  name: Custom Saga
  description: Detailed description
  timeout: 5m
  compensation_strategy: backward
  steps:
    - id: step-1
      name: First Step
      type: validation
      timeout: 30s
      retryable: true
      compensatable: false
```

### 使用 Fixture Generator

```go
generator := sagatesting.NewFixtureGenerator()

// 生成 Saga 定义
sagaDef := generator.GenerateSagaDefinition("test-saga", "Test Saga", 5)

// 生成 Saga 实例
instance := generator.GenerateSagaInstance(
    "saga-001",
    "test-saga",
    saga.SagaStateRunning,
)

// 生成步骤状态
stepState := generator.GenerateStepState("saga-001", 0, saga.StepStateCompleted)

// 生成事件
event := generator.GenerateEvent("saga-001", saga.EventSagaStarted)

// 生成错误
error := generator.GenerateError(
    "TEST_ERROR",
    "Test error message",
    saga.ErrorTypeStepExecution,
    false,
)
```

## Fixture 格式

所有 fixtures 遵循统一的格式:

```yaml
id: unique-fixture-id          # 唯一标识符
name: Display Name              # 显示名称
description: Detailed description  # 详细描述
type: fixture_type              # fixture 类型
tags:                           # 标签列表
  - tag1
  - tag2
metadata:                       # 元数据
  key: value
data:                          # 实际数据
  # 类型特定的数据结构
```

## 最佳实践

1. **命名约定**
   - 使用 kebab-case 命名 fixture ID
   - ID 应该描述性强且唯一
   - 文件名与 ID 保持一致

2. **标签使用**
   - 使用标签分类 fixtures
   - 常用标签: success, failure, error, timeout, compensation
   - 添加领域标签: order, payment, inventory

3. **文档化**
   - 提供清晰的描述
   - 在 metadata 中记录场景和用途
   - 注释复杂的数据结构

4. **版本控制**
   - 所有 fixtures 纳入版本控制
   - 更新时保持向后兼容
   - 使用 metadata.version 跟踪版本

5. **数据真实性**
   - 使用真实的数据格式
   - 避免过于简化的示例
   - 包含边界情况

## 验证

验证 fixture 的有效性:

```go
fixture, err := loader.Load("my-fixture")
if err != nil {
    t.Fatal(err)
}

if err := sagatesting.ValidateFixture(fixture); err != nil {
    t.Errorf("Invalid fixture: %v", err)
}
```

## 贡献

欢迎贡献新的 fixtures！请确保：

1. 遵循命名约定和格式
2. 提供清晰的文档
3. 添加适当的标签
4. 验证 YAML 语法
5. 更新本 README

## 相关文档

- [Saga Testing Tools](../README.md)
- [Test Builder](../builder.go)
- [Assertions](../assertions.go)
- [Mock Implementations](../mocks.go)


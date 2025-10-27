# Saga 测试

Saga 分布式事务的全面测试套件。

## 概述

Saga 测试套件提供了一套完整的工具和框架，确保分布式事务的可靠性和性能。包括单元测试、集成测试、性能基准测试和混沌测试。

## 测试覆盖率

**总体覆盖率**: **73.6%**

| 模块 | 覆盖率 | 状态 |
|-----|-------|------|
| saga core | 96.0% | ✅ 优秀 |
| coordinator | 87.4% | ✅ 良好 |
| retry | 91.6% | ✅ 优秀 |
| security | 86.2% | ✅ 良好 |
| messaging | 86.1% | ✅ 良好 |
| dsl | 85.5% | ✅ 良好 |

## 快速开始

### 运行测试

```bash
# 运行所有 Saga 测试
go test ./pkg/saga/...

# 运行并生成覆盖率
go test -coverprofile=coverage.out ./pkg/saga/...

# 查看覆盖率报告
go tool cover -html=coverage.out
```

### 使用测试工具

```go
import sagatesting "github.com/innovationmech/swit/pkg/saga/testing"

// 创建 Mock
storage := sagatesting.NewMockStateStorage()
publisher := sagatesting.NewMockEventPublisher()

// 构建测试 Saga
saga := sagatesting.NewSagaTestBuilder("test-saga", "测试").
    AddStep("step1", "步骤 1").
    AddStep("step2", "步骤 2").
    Build()

// 使用断言
sagatesting.AssertAllStepsCompleted()
sagatesting.AssertNoCompensation()
```

## 文档

### 指南

- [测试指南](../../../saga-testing-guide.md) - 全面的测试指南
- [覆盖率报告](../../../saga-test-coverage.md) - 详细的覆盖率分析
- [性能基准](../../../saga-performance-benchmarks.md) - 性能指标

### 测试工具

- **Mock** - 完整的 Mock 实现
- **构建器** - 流畅的测试场景构建器
- **断言** - 丰富的断言库
- **Fixtures** - 标准化的测试数据
- **混沌测试** - 故障注入工具

## 测试类型

### 单元测试

测试单个组件和函数：

```go
func TestStepExecution(t *testing.T) {
    step := sagatesting.NewMockSagaStep("test", "测试")
    result, err := step.Execute(ctx, data)
    require.NoError(t, err)
}
```

### 集成测试

测试组件交互：

```go
func TestE2ESaga(t *testing.T) {
    orchestrator := coordinator.NewOrchestratorCoordinator(storage, publisher)
    instance, err := orchestrator.StartSaga(ctx, sagaDef, data)
    require.NoError(t, err)
}
```

### 性能测试

验证吞吐量和延迟：

```bash
go test -bench=. -benchmem ./pkg/saga/testing/benchmarks/
```

### 混沌测试

模拟故障场景：

```go
injector := chaos.NewFaultInjector()
injector.AddFault("network", &chaos.FaultConfig{
    Type: chaos.FaultTypeNetworkPartition,
    Probability: 0.3,
})
```

## 性能指标

| 指标 | 目标 | 实际 | 状态 |
|-----|------|------|------|
| 简单 Saga 吞吐量 | > 1000/s | 5,750/s | ✅ |
| 复杂 Saga 吞吐量 | > 500/s | 2,150/s | ✅ |
| 并发 (100) | > 5000/s | 123,000/s | ✅ |
| P50 延迟 | < 10ms | 2.8ms | ✅ |
| P95 延迟 | < 50ms | 18.5ms | ✅ |
| P99 延迟 | < 100ms | 42.3ms | ✅ |

## 最佳实践

1. **使用表驱动测试** - 提高代码复用
2. **隔离依赖** - 避免状态污染
3. **验证交互** - 检查 Mock 调用
4. **使用断言组合** - 创建可复用的断言
5. **渐进式故障注入** - 从低概率开始

## CI/CD 集成

```yaml
- name: 运行 Saga 测试
  run: go test -v -coverprofile=coverage.out ./pkg/saga/...

- name: 上传覆盖率
  uses: codecov/codecov-action@v3
  with:
    files: ./coverage.out
```

## 相关页面

- [配置测试](../guide/config-testing.md)
- [框架测试](../guide/testing.md)
- [性能监控](../guide/monitoring.md)

## 资源

- [测试包 README](https://github.com/innovationmech/swit/tree/master/pkg/saga/testing)
- [基准测试套件](https://github.com/innovationmech/swit/tree/master/pkg/saga/testing/benchmarks)
- [混沌测试](https://github.com/innovationmech/swit/tree/master/pkg/saga/testing/chaos)
- [测试 Fixtures](https://github.com/innovationmech/swit/tree/master/pkg/saga/testing/fixtures)


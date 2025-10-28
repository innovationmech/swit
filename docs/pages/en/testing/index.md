# Saga Testing

Comprehensive testing suite for Saga distributed transactions.

## Overview

The Saga testing suite provides a complete set of tools and frameworks to ensure the reliability and performance of distributed transactions. It includes unit tests, integration tests, performance benchmarks, and chaos testing.

## Testing Coverage

**Overall Coverage**: **73.6%**

| Module | Coverage | Status |
|--------|----------|--------|
| saga core | 96.0% | ✅ Excellent |
| coordinator | 87.4% | ✅ Good |
| retry | 91.6% | ✅ Excellent |
| security | 86.2% | ✅ Good |
| messaging | 86.1% | ✅ Good |
| dsl | 85.5% | ✅ Good |

## Quick Start

### Running Tests

```bash
# Run all Saga tests
go test ./pkg/saga/...

# Run with coverage
go test -coverprofile=coverage.out ./pkg/saga/...

# View coverage report
go tool cover -html=coverage.out
```

### Using Test Tools

```go
import sagatesting "github.com/innovationmech/swit/pkg/saga/testing"

// Create Mock
storage := sagatesting.NewMockStateStorage()
publisher := sagatesting.NewMockEventPublisher()

// Build Test Saga
saga := sagatesting.NewSagaTestBuilder("test-saga", "Test").
    AddStep("step1", "Step 1").
    AddStep("step2", "Step 2").
    Build()

// Use Assertions
sagatesting.AssertAllStepsCompleted()
sagatesting.AssertNoCompensation()
```

## Documentation

### Guides

- [Testing Guide](../../../saga-testing-guide.md) - Comprehensive testing guide
- [Coverage Report](../../../saga-test-coverage.md) - Detailed coverage analysis
- [Performance Benchmarks](../../../saga-performance-benchmarks.md) - Performance metrics

### Testing Tools

- **Mocks** - Complete mock implementations
- **Builders** - Fluent test scenario builders
- **Assertions** - Rich assertion library
- **Fixtures** - Standardized test data
- **Chaos Testing** - Fault injection tools

## Test Types

### Unit Tests

Test individual components and functions:

```go
func TestStepExecution(t *testing.T) {
    step := sagatesting.NewMockSagaStep("test", "Test")
    result, err := step.Execute(ctx, data)
    require.NoError(t, err)
}
```

### Integration Tests

Test component interactions:

```go
func TestE2ESaga(t *testing.T) {
    orchestrator := coordinator.NewOrchestratorCoordinator(storage, publisher)
    instance, err := orchestrator.StartSaga(ctx, sagaDef, data)
    require.NoError(t, err)
}
```

### Performance Tests

Verify throughput and latency:

```bash
go test -bench=. -benchmem ./pkg/saga/testing/benchmarks/
```

### Chaos Tests

Simulate failure scenarios:

```go
injector := chaos.NewFaultInjector()
injector.AddFault("network", &chaos.FaultConfig{
    Type: chaos.FaultTypeNetworkPartition,
    Probability: 0.3,
})
```

## Performance Metrics

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| Simple Saga Throughput | > 1000/s | 5,750/s | ✅ |
| Complex Saga Throughput | > 500/s | 2,150/s | ✅ |
| Concurrent (100) | > 5000/s | 123,000/s | ✅ |
| P50 Latency | < 10ms | 2.8ms | ✅ |
| P95 Latency | < 50ms | 18.5ms | ✅ |
| P99 Latency | < 100ms | 42.3ms | ✅ |

## Best Practices

1. **Use Table-Driven Tests** - Improve code reuse
2. **Isolate Dependencies** - Avoid state pollution
3. **Verify Interactions** - Check mock calls
4. **Use Assertion Combinations** - Create reusable assertions
5. **Progressive Fault Injection** - Start with low probability

## CI/CD Integration

```yaml
- name: Run Saga Tests
  run: go test -v -coverprofile=coverage.out ./pkg/saga/...

- name: Upload Coverage
  uses: codecov/codecov-action@v3
  with:
    files: ./coverage.out
```

## Related Pages

- [Configuration Testing](../guide/config-testing.md)
- [Framework Testing](../guide/testing.md)
- [Performance Monitoring](../guide/monitoring.md)

## Resources

- [Testing Package README](https://github.com/innovationmech/swit/tree/master/pkg/saga/testing)
- [Benchmark Suite](https://github.com/innovationmech/swit/tree/master/pkg/saga/testing/benchmarks)
- [Chaos Testing](https://github.com/innovationmech/swit/tree/master/pkg/saga/testing/chaos)
- [Test Fixtures](https://github.com/innovationmech/swit/tree/master/pkg/saga/testing/fixtures)


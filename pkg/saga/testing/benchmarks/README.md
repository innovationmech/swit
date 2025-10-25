# Saga 性能基准测试套件

本目录包含 Saga 系统的性能基准测试，用于验证并发处理能力和资源使用效率。

## 概述

基准测试套件包含以下组件的性能测试：

1. **Orchestrator 执行性能** - 测试 Saga 编排器的执行性能
2. **并发 Saga 执行** - 测试不同并发级别下的性能表现
3. **状态持久化** - 测试状态存储的读写性能
4. **DSL 解析** - 测试 DSL 文件的解析和验证性能
5. **消息发布** - 测试事件消息的发布性能

## 运行基准测试

### 运行所有基准测试

```bash
cd pkg/saga/testing/benchmarks
go test -bench=. -benchmem -benchtime=5s
```

### 运行特定基准测试

```bash
# Orchestrator 性能测试
go test -bench=BenchmarkOrchestrator -benchmem

# 并发执行测试
go test -bench=BenchmarkConcurrent -benchmem

# 状态持久化测试
go test -bench=BenchmarkState -benchmem

# DSL 解析测试
go test -bench=BenchmarkDSL -benchmem

# 消息发布测试
go test -bench=BenchmarkMessage -benchmem
```

### 生成性能分析报告

```bash
# CPU 分析
go test -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof

# 内存分析
go test -bench=. -memprofile=mem.prof
go tool pprof mem.prof

# 完整分析
go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof -trace=trace.out
```

## 基准测试文件说明

### orchestrator_bench_test.go

测试 Orchestrator 编排器的核心性能：

- `BenchmarkOrchestratorSimpleSaga` - 简单 Saga 执行
- `BenchmarkOrchestratorComplexSaga` - 复杂多步骤 Saga
- `BenchmarkOrchestratorWithCompensation` - 带补偿的 Saga
- `BenchmarkOrchestratorStateTransitions` - 状态转换
- `BenchmarkOrchestratorMemoryAllocation` - 内存分配

**关键指标**：
- 执行时间（ns/op）
- 内存分配（B/op）
- 分配次数（allocs/op）

### concurrent_bench_test.go

测试并发场景下的性能表现：

- `BenchmarkConcurrentSagaExecution` - 不同并发级别（1, 10, 50, 100, 500）
- `BenchmarkConcurrentSagaWithSharedState` - 共享状态访问
- `BenchmarkConcurrentSagaStartStop` - 高并发启停
- `BenchmarkConcurrentSagaWithContention` - 高竞争场景
- `BenchmarkConcurrentSagaThroughput` - 吞吐量测量
- `BenchmarkConcurrentSagaLatency` - 延迟分布（P50/P95/P99）

**关键指标**：
- 吞吐量（sagas/sec）
- 延迟（P50/P95/P99 微秒）
- 并发扩展性

### state_bench_test.go

测试状态持久化性能：

- `BenchmarkStateStorageSave` - 状态保存
- `BenchmarkStateStorageLoad` - 状态加载
- `BenchmarkStateStorageUpdate` - 状态更新
- `BenchmarkStateStorageDelete` - 状态删除
- `BenchmarkStateStorageList` - 状态列表查询
- `BenchmarkStateStorageBatchSave` - 批量保存（10, 50, 100, 500）
- `BenchmarkStateStorageConcurrent` - 并发操作
- `BenchmarkStateStorageWithLargeData` - 大数据存储（1KB, 10KB, 100KB）

**关键指标**：
- CRUD 操作延迟
- 批量操作吞吐量
- 大数据处理性能

### dsl_bench_test.go

测试 DSL 解析和验证性能：

- `BenchmarkDSLParserSimple` - 简单 DSL 解析
- `BenchmarkDSLParserComplex` - 复杂 DSL 解析（5, 10, 20, 50 步骤）
- `BenchmarkDSLValidation` - DSL 验证
- `BenchmarkDSLParserWithEnvVars` - 环境变量替换
- `BenchmarkDSLGenerator` - DSL 生成
- `BenchmarkDSLParserConcurrent` - 并发解析
- `BenchmarkDSLParserLargeFile` - 大型文件解析（100, 200, 500 步骤）

**关键指标**：
- 解析时间
- 验证开销
- 内存使用

### messaging_bench_test.go

测试消息发布性能：

- `BenchmarkMessagePublishSingle` - 单消息发布
- `BenchmarkMessagePublishBatch` - 批量发布（10, 50, 100, 500, 1000）
- `BenchmarkMessagePublishAsync` - 异步发布
- `BenchmarkMessageSerialization` - 消息序列化（JSON, Protobuf, MsgPack）
- `BenchmarkMessageDeserialization` - 消息反序列化
- `BenchmarkMessagePublishConcurrent` - 并发发布
- `BenchmarkMessagePublishWithRetry` - 带重试的发布
- `BenchmarkMessagePublishWithConfirmation` - 带确认的发布

**关键指标**：
- 发布延迟
- 批量操作吞吐量
- 序列化/反序列化性能
- 重试开销

## 性能目标

### 吞吐量目标

- **简单 Saga**: > 1000 sagas/sec（单核）
- **复杂 Saga**: > 500 sagas/sec（单核）
- **并发执行（100 并发）**: > 5000 sagas/sec

### 延迟目标

- **P50**: < 10ms
- **P95**: < 50ms
- **P99**: < 100ms

### 资源使用

- **内存**: 单个 Saga < 10KB
- **CPU**: 单核处理 > 500 sagas/sec

### 状态持久化

- **保存**: < 5ms/op
- **加载**: < 3ms/op
- **批量保存（100）**: < 50ms/op

### DSL 解析

- **简单 DSL**: < 1ms
- **复杂 DSL（50 步骤）**: < 10ms

### 消息发布

- **单消息**: < 1ms
- **批量（100）**: < 10ms

## 基准结果分析

### 查看基准结果

基准测试结果保存在 `baseline.txt` 文件中，包含：

- 测试名称
- 执行次数
- 每次操作耗时（ns/op）
- 内存分配（B/op）
- 分配次数（allocs/op）
- 自定义指标（吞吐量、延迟等）

### 比较基准结果

使用 `benchstat` 工具比较两次基准测试结果：

```bash
# 安装 benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# 比较结果
benchstat baseline.txt new_results.txt
```

### 性能回归检测

在 CI/CD 中集成性能回归检测：

```bash
# 运行基准测试并保存结果
go test -bench=. -benchmem > new_baseline.txt

# 与历史基准比较
benchstat baseline.txt new_baseline.txt

# 如果性能下降超过 10%，失败构建
```

## 优化建议

### Orchestrator 优化

1. 减少锁竞争，使用细粒度锁
2. 优化状态转换逻辑
3. 使用对象池减少内存分配
4. 异步执行非关键操作

### 并发优化

1. 调整 `MaxConcurrentSagas` 参数
2. 使用工作池管理协程
3. 优化共享状态访问
4. 减少上下文切换

### 状态持久化优化

1. 使用批量操作减少 I/O
2. 启用缓存层
3. 优化数据序列化
4. 使用连接池

### DSL 解析优化

1. 缓存已解析的 DSL
2. 延迟验证非关键字段
3. 优化正则表达式
4. 减少内存拷贝

### 消息发布优化

1. 使用批量发布减少网络开销
2. 启用消息压缩
3. 优化序列化格式（Protobuf > MsgPack > JSON）
4. 使用连接池

## 持续监控

### 设置性能监控

```bash
# 启动 Prometheus 监控
cd examples/saga-monitoring
docker-compose up -d

# 查看性能指标
open http://localhost:3000  # Grafana
open http://localhost:9090  # Prometheus
```

### 关键性能指标

- `saga_execution_duration_seconds` - Saga 执行时间
- `saga_throughput_total` - Saga 吞吐量
- `saga_concurrent_executions` - 并发执行数
- `state_operation_duration_seconds` - 状态操作延迟
- `message_publish_duration_seconds` - 消息发布延迟

## 故障排查

### 性能问题诊断

1. **高延迟**
   - 检查数据库连接池配置
   - 查看是否有慢查询
   - 检查网络延迟

2. **低吞吐量**
   - 增加并发级别
   - 优化热点代码路径
   - 检查 CPU 和内存使用

3. **内存泄漏**
   - 使用 pprof 分析内存分配
   - 检查协程泄漏
   - 验证资源正确释放

## 相关文档

- [Saga 用户指南](../../../../docs/saga-user-guide.md)
- [Saga 监控指南](../../../../docs/saga-monitoring-guide.md)
- [性能监控和告警](../../../../docs/performance-monitoring-alerting.md)
- [Saga 示例](../../examples/README.md)

## 贡献

如果你发现性能问题或有优化建议，请：

1. 创建基准测试复现问题
2. 提交 Issue 描述性能问题
3. 提供性能分析报告（pprof）
4. 提交 PR 包含优化和基准测试结果

## 许可证

Copyright © 2025 jackelyj <dreamerlyj@gmail.com>

本项目采用 MIT 许可证。


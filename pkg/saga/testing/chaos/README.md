# Saga 混沌测试套件

本目录包含 Saga 系统的混沌测试套件，用于模拟各种故障场景，验证系统的可靠性和容错能力。

## 概述

混沌测试（Chaos Testing）通过主动注入故障来验证系统在异常情况下的行为。本测试套件模拟了以下故障场景：

- **网络故障** - 网络分区、连接失败
- **服务崩溃** - 服务中断、进程终止
- **消息丢失** - 消息队列故障、事件丢失
- **超时** - 操作超时、响应延迟
- **数据库故障** - 连接失败、事务回滚、部分写入
- **部分失败** - 分布式系统中的部分组件失败

## 文件结构

```
pkg/saga/testing/chaos/
├── README.md                     # 本文档
├── fault_injector.go             # 故障注入器核心实现
├── network_failure_test.go       # 网络故障测试
├── service_crash_test.go         # 服务崩溃测试
├── message_loss_test.go          # 消息丢失测试
├── timeout_test.go               # 超时场景测试
└── database_failure_test.go      # 数据库故障测试
```

## 核心组件

### FaultInjector（故障注入器）

故障注入器提供了统一的故障注入接口，支持以下功能：

- **概率性故障注入** - 根据配置的概率决定是否注入故障
- **目标步骤选择** - 可以针对特定步骤或全局注入故障
- **多种故障类型** - 支持网络、服务、消息、超时、数据库等故障
- **统计收集** - 记录故障注入的统计信息
- **动态启用/禁用** - 可以在运行时启用或禁用故障注入

### 故障类型

```go
const (
    FaultTypeNetworkPartition    // 网络分区
    FaultTypeServiceCrash         // 服务崩溃
    FaultTypeMessageLoss          // 消息丢失
    FaultTypeTimeout              // 超时
    FaultTypeDatabaseFailure      // 数据库故障
    FaultTypePartialFailure       // 部分失败
    FaultTypeDelay                // 延迟
    FaultTypeRandomError          // 随机错误
)
```

## 使用示例

### 基本用法

```go
// 创建故障注入器
injector := NewFaultInjector()

// 添加网络分区故障
injector.AddFault("network-fault", &FaultConfig{
    Type:        FaultTypeNetworkPartition,
    Probability: 0.3,  // 30% 概率
    TargetStep:  "step1",
})

// 包装 Saga 步骤
wrappedStep := injector.WrapStep(originalStep)

// 包装状态存储
chaosStorage := NewChaosStateStorage(storage, injector)

// 包装事件发布器
chaosPublisher := NewChaosEventPublisher(publisher, injector)
```

### 故障配置

```go
config := &FaultConfig{
    Type:         FaultTypeTimeout,        // 故障类型
    Probability:  1.0,                     // 注入概率 (0.0-1.0)
    Duration:     5 * time.Second,         // 故障持续时间
    Delay:        2 * time.Second,         // 延迟时间
    TargetStep:   "payment-step",          // 目标步骤（空表示全部）
    ErrorMessage: "custom error message",  // 自定义错误信息
    Metadata:     map[string]interface{}{  // 额外元数据
        "region": "us-east-1",
    },
}
```

## 测试场景

### 1. 网络故障测试（network_failure_test.go）

测试 Saga 在网络故障下的行为：

- **网络分区导致步骤失败** - 验证步骤执行失败时的补偿机制
- **状态存储网络故障** - 验证状态持久化失败的处理
- **事件发布网络故障** - 验证事件发布失败的处理
- **网络恢复后的 Saga 恢复** - 验证网络恢复后的自动恢复
- **并发 Saga 的网络故障** - 验证多个 Saga 同时遇到网络故障

**运行测试：**
```bash
go test -v -run TestNetworkPartition ./pkg/saga/testing/chaos
```

### 2. 服务崩溃测试（service_crash_test.go）

测试服务崩溃场景：

- **步骤执行时服务崩溃** - 验证崩溃后的状态一致性
- **服务重启后的状态恢复** - 验证从持久化存储恢复
- **补偿阶段的服务崩溃** - 验证补偿逻辑的容错性
- **部分补偿后的崩溃** - 验证部分补偿的处理

**运行测试：**
```bash
go test -v -run TestServiceCrash ./pkg/saga/testing/chaos
```

### 3. 消息丢失测试（message_loss_test.go）

测试消息系统故障：

- **事件发布消息丢失** - 验证事件丢失后的系统行为
- **消息丢失不违反顺序保证** - 验证事件顺序的一致性
- **消息重试机制** - 验证失败后的重试逻辑
- **幂等性检查** - 验证重复消息的去重
- **死信队列恢复** - 验证从 DLQ 恢复消息

**运行测试：**
```bash
go test -v -run TestMessageLoss ./pkg/saga/testing/chaos
```

### 4. 超时测试（timeout_test.go）

测试各种超时场景：

- **步骤执行超时** - 验证步骤超时的处理
- **Saga 级别超时** - 验证整体 Saga 超时
- **补偿超时** - 验证补偿操作超时的处理
- **超时重试机制** - 验证超时后的重试
- **Context 取消** - 验证 context 取消的处理

**运行测试：**
```bash
go test -v -run TestTimeout ./pkg/saga/testing/chaos
```

### 5. 数据库故障测试（database_failure_test.go）

测试数据库故障场景：

- **保存操作失败** - 验证状态保存失败的处理
- **查询操作失败** - 验证状态查询失败的处理
- **事务回滚** - 验证事务失败时的回滚
- **连接丢失** - 验证数据库连接丢失的恢复
- **部分写入** - 验证部分状态更新的一致性
- **数据一致性** - 验证故障后的数据一致性
- **恢复机制** - 验证数据库恢复后的系统恢复

**运行测试：**
```bash
go test -v -run TestDatabaseFailure ./pkg/saga/testing/chaos
```

## 运行所有混沌测试

```bash
# 运行所有混沌测试
go test -v ./pkg/saga/testing/chaos

# 运行特定测试
go test -v -run TestNetworkPartition_StepExecution ./pkg/saga/testing/chaos

# 生成覆盖率报告
go test -v -coverprofile=chaos_coverage.out ./pkg/saga/testing/chaos
go tool cover -html=chaos_coverage.out -o chaos_coverage.html
```

## 验收标准

混沌测试验证以下关键属性：

### 1. 最终一致性
- ✅ 所有 Saga 最终达到终止状态（完成、失败或补偿）
- ✅ 没有 Saga 停留在中间状态

### 2. 补偿逻辑
- ✅ 失败时正确触发补偿
- ✅ 补偿按正确顺序执行
- ✅ 补偿失败时有适当的错误处理

### 3. 状态持久化
- ✅ 故障后能从持久化状态恢复
- ✅ 状态更新是原子的或可恢复的
- ✅ 并发访问不破坏状态一致性

### 4. 重试机制
- ✅ 可重试的错误会自动重试
- ✅ 重试次数受限，避免无限重试
- ✅ 重试使用适当的退避策略

### 5. 超时处理
- ✅ 超时被正确检测和处理
- ✅ 超时后触发适当的清理和补偿
- ✅ Context 取消被正确传播

### 6. 幂等性
- ✅ 重复的操作不会产生副作用
- ✅ 消息去重机制正常工作
- ✅ 状态更新是幂等的

## 故障注入最佳实践

### 1. 渐进式注入
从低概率开始，逐步增加故障注入概率：
```go
// 开始时低概率
injector.AddFault("fault-1", &FaultConfig{Probability: 0.1})
// 验证通过后增加概率
injector.AddFault("fault-2", &FaultConfig{Probability: 0.5})
```

### 2. 隔离故障
一次测试一种故障类型，便于定位问题：
```go
// 好的做法：单一故障类型
injector.AddFault("network", &FaultConfig{Type: FaultTypeNetworkPartition})

// 避免：同时多种故障
// injector.AddFault("multi", &FaultConfig{Type: ...})
// injector.AddFault("chaos", &FaultConfig{Type: ...})
```

### 3. 记录和监控
启用日志记录故障注入事件：
```go
injector.SetLogFunc(func(format string, args ...interface{}) {
    t.Logf(format, args...)
})
```

### 4. 验证最终状态
始终验证系统达到一致的最终状态：
```go
// 等待 Saga 完成
time.Sleep(waitTime)

// 验证最终状态
if !instance.IsTerminal() {
    t.Error("Saga 未达到终止状态")
}
```

## 故障注入统计

故障注入器收集以下统计信息：

```go
stats := injector.GetStats()
// stats.TotalInjections        - 总注入次数
// stats.SuccessfulInjections   - 成功注入次数
// stats.FailedInjections       - 失败注入次数
// stats.InjectionsByType       - 按类型统计
// stats.LastInjectionTime      - 最后注入时间
```

## 调试技巧

### 1. 启用详细日志
```go
injector.SetLogFunc(func(format string, args ...interface{}) {
    log.Printf("[CHAOS] "+format, args...)
})
```

### 2. 使用确定性故障
在调试时使用 100% 概率：
```go
config := &FaultConfig{
    Probability: 1.0,  // 确保每次都触发
}
```

### 3. 减少并发
先用单个 Saga 测试，然后逐步增加并发：
```go
// 调试阶段
numSagas := 1

// 验证通过后
numSagas := 100
```

### 4. 增加等待时间
给系统足够的时间处理故障和恢复：
```go
// 调试时使用较长等待时间
time.Sleep(5 * time.Second)

// 确认后可以减少
time.Sleep(1 * time.Second)
```

## 贡献指南

添加新的混沌测试时：

1. **创建新的测试文件** - 按故障类型组织
2. **实现测试场景** - 使用表驱动测试模式
3. **验证最终一致性** - 确保系统达到一致状态
4. **添加文档** - 更新本 README 说明新测试
5. **运行所有测试** - 确保不影响现有测试

## 参考资料

- [混沌工程原则](https://principlesofchaos.org/)
- [Saga 模式文档](../../../docs/saga-user-guide.md)
- [分布式系统测试](../../../docs/service-development-guide.md)

## 问题排查

### 测试不稳定
- 检查等待时间是否足够
- 验证故障概率配置
- 确认资源清理正确

### 死锁或挂起
- 检查 context 超时设置
- 验证锁的正确使用
- 使用 pprof 分析

### 状态不一致
- 检查并发访问控制
- 验证状态更新的原子性
- 确认补偿逻辑正确

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](../../../../LICENSE) 文件。


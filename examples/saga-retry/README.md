# Saga Retry 机制示例

这个示例演示了如何使用 Swit 的重试机制来处理分布式系统中的临时性故障。

## 功能演示

这个示例程序展示了以下特性：

1. **指数退避重试**：适用于网络请求和外部API调用
2. **线性退避重试**：适用于资源竞争场景
3. **固定间隔重试**：适用于定期轮询和状态检查
4. **断路器模式**：防止级联故障，实现快速失败
5. **错误分类**：区分可重试错误和永久性错误
6. **回调函数**：监控重试过程和收集指标
7. **超时控制**：防止无限期重试
8. **抖动类型比较**：展示不同抖动算法的效果

## 运行示例

```bash
# 进入示例目录
cd examples/saga-retry

# 运行示例
go run main.go
```

## 示例输出

```
=== Saga Retry 机制示例 ===

1. 指数退避重试示例
-------------------
✅ 操作成功: Response from payment-api (call #3)
📊 尝试次数: 3, 总耗时: 456ms
📊 payment-api: 3 calls, 1 success (33.3%), 2 failures

2. 线性退避重试示例
-------------------
✅ 操作成功: Response from inventory-api (call #2)
📊 尝试次数: 2, 总耗时: 234ms
📊 inventory-api: 2 calls, 1 success (50.0%), 1 failures

...
```

## 代码结构

### SimulatedService

模拟一个可能失败的外部服务：

- `failureRate`：配置失败率（0.0-1.0）
- `avgLatency`：平均响应延迟
- `Call()`：执行服务调用（可能失败）
- `GetStats()`：获取调用统计信息

### 示例函数

1. `runExponentialBackoffExample()`
   - 演示指数退避策略
   - 适用场景：网络请求、外部API调用
   - 配置：2倍增长，10%抖动

2. `runLinearBackoffExample()`
   - 演示线性退避策略
   - 适用场景：资源竞争、限流
   - 配置：每次增加150ms

3. `runFixedIntervalExample()`
   - 演示固定间隔策略
   - 适用场景：定期轮询、状态检查
   - 配置：固定200ms间隔

4. `runCircuitBreakerExample()`
   - 演示断路器模式
   - 展示状态转换：Closed → Open → Half-Open
   - 监控断路器状态变化

5. `runErrorClassificationExample()`
   - 演示错误分类
   - 区分可重试错误和永久性错误
   - 配置白名单和黑名单

6. `runCallbacksExample()`
   - 演示回调函数的使用
   - OnRetry、OnSuccess、OnFailure
   - 适用于日志记录和指标收集

7. `runTimeoutExample()`
   - 演示超时控制
   - 防止无限期重试
   - Context 取消机制

8. `runJitterComparisonExample()`
   - 比较不同抖动类型
   - Full、Equal、Decorrelated Jitter
   - 展示延迟分布差异

## 关键概念

### 重试策略

#### 指数退避（Exponential Backoff）
```go
policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.1)
```
- 延迟按指数增长：100ms → 200ms → 400ms → 800ms
- 适合网络临时故障
- 快速降低系统负载

#### 线性退避（Linear Backoff）
```go
policy := retry.NewLinearBackoffPolicy(config, 150*time.Millisecond, 0.1)
```
- 延迟线性增长：100ms → 250ms → 400ms → 550ms
- 适合资源竞争场景
- 可预测的延迟增长

#### 固定间隔（Fixed Interval）
```go
policy := retry.NewFixedIntervalPolicy(config, 200*time.Millisecond, 0)
```
- 固定延迟：200ms → 200ms → 200ms
- 最简单直接
- 适合定期检查

### 断路器（Circuit Breaker）

断路器有三种状态：

1. **Closed（关闭）**
   - 正常工作，允许所有请求
   - 跟踪连续失败次数

2. **Open（打开）**
   - 快速失败，拒绝所有请求
   - 等待恢复时间（ResetTimeout）

3. **Half-Open（半开）**
   - 试探性恢复
   - 允许少量请求测试服务
   - 成功则关闭，失败则重新打开

状态转换规则：
```
Closed --[MaxFailures达到]--> Open
Open --[ResetTimeout后]--> Half-Open
Half-Open --[SuccessThreshold达到]--> Closed
Half-Open --[任何失败]--> Open
```

### 抖动（Jitter）

抖动用于防止"惊群效应"：

- **Full Jitter**：`random(0, delay)`
  - 完全随机化延迟
  - 最大程度避免同步重试

- **Equal Jitter**：`delay/2 + random(0, delay/2)`
  - 保留一半固定延迟
  - 平衡可预测性和随机性

- **Decorrelated Jitter**：`random(InitialDelay, delay * 3)`
  - 基于前次延迟计算
  - 去相关化重试时间

## 配置建议

### 网络请求场景
```go
config := &retry.RetryConfig{
    MaxAttempts:  5,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     30 * time.Second,
}
policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.2)
```

### 资源竞争场景
```go
config := &retry.RetryConfig{
    MaxAttempts:  10,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     5 * time.Second,
}
policy := retry.NewLinearBackoffPolicy(config, 200*time.Millisecond, 0.1)
```

### 定期检查场景
```go
config := &retry.RetryConfig{
    MaxAttempts:  100,
    InitialDelay: 1 * time.Second,
}
policy := retry.NewFixedIntervalPolicy(config, 1*time.Second, 0)
```

### 断路器配置
```go
cbConfig := &retry.CircuitBreakerConfig{
    MaxFailures:         5,                // 5次失败后打开
    ResetTimeout:        60 * time.Second, // 60秒后尝试恢复
    HalfOpenMaxRequests: 3,                // 半开时允许3个请求
    SuccessThreshold:    2,                // 2次成功后关闭
}
```

## 最佳实践

1. **选择合适的策略**
   - 网络请求：指数退避 + 抖动
   - 资源竞争：线性退避
   - 定期检查：固定间隔

2. **设置合理的限制**
   - MaxAttempts：避免无限重试
   - MaxDelay：防止延迟过长
   - Timeout：设置总体超时

3. **使用断路器**
   - 保护下游服务
   - 快速失败
   - 避免级联故障

4. **错误分类**
   - 明确哪些错误可重试
   - 避免重试永久性错误
   - 使用 NonRetryableErrors 黑名单

5. **监控和日志**
   - 使用回调记录重试
   - 监控断路器状态
   - 收集指标数据

6. **使用抖动**
   - 避免惊群效应
   - 推荐 10%-30% 抖动
   - 高并发场景必须使用

## 性能考虑

1. **避免过度重试**
   - 合理设置 MaxAttempts
   - 使用断路器快速失败

2. **控制延迟增长**
   - 设置 MaxDelay 上限
   - 考虑业务 SLA

3. **资源管理**
   - Context 取消支持
   - 及时释放资源

4. **并发场景**
   - 使用抖动避免峰值
   - 考虑限流

## 故障排查

### 问题：重试次数过多

**原因**：
- 下游服务持续不可用
- MaxAttempts 设置过高
- 错误分类不当

**解决**：
- 使用断路器
- 降低 MaxAttempts
- 正确配置 NonRetryableErrors

### 问题：延迟时间过长

**原因**：
- 指数退避增长过快
- MaxDelay 设置过高

**解决**：
- 降低 Multiplier
- 设置合理的 MaxDelay
- 考虑使用线性退避

### 问题：惊群效应

**原因**：
- 没有使用抖动
- 多个客户端同时重试

**解决**：
- 启用抖动（Jitter >= 0.1）
- 使用 Decorrelated Jitter
- 错开初始重试时间

## 扩展示例

### 与 Prometheus 集成

```go
registry := prometheus.NewRegistry()
collector, _ := retry.NewRetryMetricsCollector("saga", "retry", registry)

executor := retry.NewExecutor(policy, retry.WithMetrics(collector))
```

### 自定义错误分类器

```go
type MyErrorClassifier struct{}

func (c *MyErrorClassifier) IsRetryable(err error) bool {
    // 自定义逻辑
    return true
}

func (c *MyErrorClassifier) IsPermanent(err error) bool {
    return errors.Is(err, ErrBadRequest)
}

executor := retry.NewExecutor(policy,
    retry.WithErrorClassifier(&MyErrorClassifier{}),
)
```

## 相关文档

- [Retry Package 文档](../../pkg/saga/retry/README.md)
- [Saga 用户指南](../../docs/saga-user-guide.md)
- [分布式事务规范](../../specs/saga-distributed-transactions/README.md)

## 许可证

Copyright © 2025 jackelyj <dreamerlyj@gmail.com>

根据 MIT 许可证授权。


# Saga 重试机制

本文档描述 Swit Saga 框架中的重试机制设计和实现。

## 目录

- [概述](#概述)
- [架构设计](#架构设计)
- [重试策略](#重试策略)
- [断路器](#断路器)
- [使用指南](#使用指南)
- [最佳实践](#最佳实践)
- [性能考虑](#性能考虑)
- [故障排查](#故障排查)

## 概述

重试机制是分布式系统中处理临时性故障的关键能力。Swit Saga 框架提供了灵活的重试策略和断路器模式，帮助开发者构建更可靠的分布式事务系统。

### 设计目标

1. **可靠性**：自动处理临时性故障，提高系统可用性
2. **灵活性**：支持多种重试策略，适应不同场景
3. **可观测性**：提供完善的日志、指标和回调机制
4. **性能**：最小化重试开销，避免雪崩效应
5. **易用性**：简单直观的 API，开箱即用的默认配置

### 关键特性

- **多种重试策略**：指数退避、线性退避、固定间隔
- **断路器模式**：防止级联故障，快速失败
- **抖动（Jitter）**：避免惊群效应
- **错误分类**：区分可重试错误和永久性错误
- **超时控制**：防止无限期重试
- **指标收集**：Prometheus 集成
- **回调支持**：灵活的生命周期钩子

## 架构设计

### 组件结构

```
┌─────────────────────────────────────────────────┐
│                   Executor                      │
│  - 执行重试逻辑                                   │
│  - 管理回调                                      │
│  - 收集指标                                      │
└──────────────┬──────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────┐
│            RetryPolicy Interface                │
│  - ShouldRetry(err, attempt) bool               │
│  - GetRetryDelay(attempt) Duration              │
│  - GetMaxAttempts() int                         │
└──────────────┬──────────────────────────────────┘
               │
       ┌───────┴────────┬─────────────┬────────────┐
       ▼                ▼             ▼            ▼
┌──────────────┐ ┌──────────┐ ┌─────────┐ ┌──────────────┐
│ Exponential  │ │ Linear   │ │ Fixed   │ │ Circuit      │
│ Backoff      │ │ Backoff  │ │Interval │ │ Breaker      │
└──────────────┘ └──────────┘ └─────────┘ └──────────────┘
```

### 核心接口

#### RetryPolicy

所有重试策略实现的接口：

```go
type RetryPolicy interface {
    // ShouldRetry 判断是否应该重试
    ShouldRetry(err error, attempt int) bool
    
    // GetRetryDelay 计算重试延迟
    GetRetryDelay(attempt int) time.Duration
    
    // GetMaxAttempts 返回最大重试次数
    GetMaxAttempts() int
}
```

#### Executor

执行器负责实际的重试逻辑：

```go
type Executor struct {
    policy          RetryPolicy
    circuitBreaker  *CircuitBreaker
    errorClassifier ErrorClassifier
    logger          *zap.Logger
    metricsCollector *RetryMetricsCollector
}
```

### 数据流

1. **执行请求**：用户通过 Executor 提交待重试的函数
2. **首次尝试**：执行函数，记录结果
3. **错误判断**：
   - 检查是否为永久性错误
   - 检查断路器状态
   - 判断是否应该重试
4. **延迟计算**：根据策略计算重试延迟
5. **等待重试**：应用延迟，支持 Context 取消
6. **重试执行**：重复步骤 2-5，直到成功或达到最大次数
7. **返回结果**：返回详细的执行结果

## 重试策略

### 指数退避（Exponential Backoff）

延迟按指数增长，快速降低系统负载。

#### 公式

```
delay = InitialDelay * (Multiplier ^ (attempt - 1)) + jitter
delay = min(delay, MaxDelay)
```

#### 配置参数

```go
type ExponentialBackoffPolicy struct {
    Config     *RetryConfig
    Multiplier float64    // 增长倍数，通常 2.0
    Jitter     float64    // 抖动系数，0.0-1.0
    JitterType JitterType // 抖动类型
}
```

#### 适用场景

- 网络请求和 HTTP API 调用
- 数据库连接
- 消息队列操作
- 外部服务调用

#### 示例

```go
config := &retry.RetryConfig{
    MaxAttempts:  5,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     30 * time.Second,
}

policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.1)

// 延迟序列（约）：100ms, 200ms, 400ms, 800ms, 1600ms
```

#### 优缺点

**优点**：
- 快速降低负载
- 适合临时性故障
- 业界广泛使用

**缺点**：
- 延迟增长快，可能导致总时间过长
- 需要合理设置 MaxDelay

### 线性退避（Linear Backoff）

延迟线性增长，提供可预测的重试间隔。

#### 公式

```
delay = InitialDelay + (Increment * (attempt - 1)) + jitter
delay = min(delay, MaxDelay)
```

#### 配置参数

```go
type LinearBackoffPolicy struct {
    Config    *RetryConfig
    Increment time.Duration // 每次增加的延迟
    Jitter    float64       // 抖动系数
}
```

#### 适用场景

- 资源锁竞争
- 限流（Rate Limiting）场景
- 需要可预测延迟的场景
- 轮询状态变化

#### 示例

```go
config := &retry.RetryConfig{
    MaxAttempts:  5,
    InitialDelay: 100 * time.Millisecond,
}

policy := retry.NewLinearBackoffPolicy(config, 200*time.Millisecond, 0.1)

// 延迟序列（约）：100ms, 300ms, 500ms, 700ms, 900ms
```

#### 优缺点

**优点**：
- 可预测的延迟增长
- 适合资源竞争场景
- 总延迟相对可控

**缺点**：
- 降低负载的速度较慢
- 可能不如指数退避高效

### 固定间隔（Fixed Interval）

使用恒定的重试间隔，最简单直接。

#### 配置参数

```go
type FixedIntervalPolicy struct {
    Config   *RetryConfig
    Interval time.Duration // 固定延迟
    Jitter   float64       // 抖动系数
}
```

#### 适用场景

- 定期轮询
- 状态检查
- 简单的重试逻辑
- 不关心负载的场景

#### 示例

```go
config := &retry.RetryConfig{
    MaxAttempts:  10,
    InitialDelay: 1 * time.Second,
}

policy := retry.NewFixedIntervalPolicy(config, 1*time.Second, 0)

// 延迟序列：1s, 1s, 1s, 1s, ...
```

#### 优缺点

**优点**：
- 最简单
- 可预测
- 易于理解和调试

**缺点**：
- 不能动态调整
- 不适合高负载场景

### 抖动（Jitter）

抖动用于防止多个客户端同时重试导致的"惊群效应"。

#### 抖动类型

1. **Full Jitter**
   ```
   delay = random(0, baseDelay)
   ```
   - 完全随机化延迟
   - 最大程度避免同步重试

2. **Equal Jitter**
   ```
   delay = baseDelay/2 + random(0, baseDelay/2)
   ```
   - 保留一半固定延迟
   - 平衡可预测性和随机性

3. **Decorrelated Jitter**
   ```
   delay = random(InitialDelay, baseDelay * 3)
   ```
   - 基于前次延迟计算
   - 去相关化重试时间

#### 配置建议

- **高并发场景**：使用 Full Jitter 或 Decorrelated Jitter
- **抖动系数**：推荐 0.1-0.3（10%-30%）
- **低并发场景**：可以不使用抖动或使用 Equal Jitter

## 断路器

断路器模式用于防止级联故障，在服务不可用时快速失败。

### 状态机

```
                MaxFailures 达到
    ┌─────────────────────────────────┐
    │                                 │
    │                                 ▼
┌────────┐                        ┌──────┐
│ Closed │                        │ Open │
│ (关闭) │◄───┐                   │ (打开)│
└────────┘    │                   └──────┘
              │ SuccessThreshold     │
              │ 达到                  │ ResetTimeout
              │                      │ 超时
         ┌────────────┐              │
         │ Half-Open  │◄─────────────┘
         │ (半开)     │
         └────────────┘
              │
              │ 任何失败
              │
              ▼
         (返回 Open)
```

### 状态说明

#### Closed（关闭）

- **行为**：正常工作，允许所有请求
- **状态**：跟踪连续失败次数
- **转换**：连续失败达到 MaxFailures → Open

#### Open（打开）

- **行为**：快速失败，拒绝所有请求
- **状态**：等待恢复时间
- **转换**：等待 ResetTimeout → Half-Open

#### Half-Open（半开）

- **行为**：试探性恢复，允许少量请求
- **状态**：限制请求数量，跟踪成功次数
- **转换**：
  - 连续成功达到 SuccessThreshold → Closed
  - 任何失败 → Open

### 配置参数

```go
type CircuitBreakerConfig struct {
    // MaxFailures 打开断路器的失败阈值
    MaxFailures int
    
    // ResetTimeout 从 Open 到 Half-Open 的等待时间
    ResetTimeout time.Duration
    
    // HalfOpenMaxRequests 半开状态允许的最大请求数
    HalfOpenMaxRequests int
    
    // SuccessThreshold 关闭断路器的成功阈值
    SuccessThreshold int
    
    // OnStateChange 状态变化回调（可选）
    OnStateChange func(from, to CircuitState)
}
```

### 配置建议

根据不同场景选择合适的配置：

#### 保守配置（快速熔断）

```go
config := &retry.CircuitBreakerConfig{
    MaxFailures:         3,  // 3次失败就打开
    ResetTimeout:        30 * time.Second,
    HalfOpenMaxRequests: 1,  // 只允许1个测试请求
    SuccessThreshold:    2,  // 需要2次成功才关闭
}
```

适用于：
- 关键业务路径
- 下游服务不稳定
- 需要快速保护

#### 宽松配置（较慢熔断）

```go
config := &retry.CircuitBreakerConfig{
    MaxFailures:         10, // 10次失败才打开
    ResetTimeout:        60 * time.Second,
    HalfOpenMaxRequests: 5,  // 允许5个测试请求
    SuccessThreshold:    3,  // 需要3次成功才关闭
}
```

适用于：
- 非关键路径
- 下游服务相对稳定
- 允许更多重试

#### 生产环境推荐

```go
config := &retry.CircuitBreakerConfig{
    MaxFailures:         5,
    ResetTimeout:        60 * time.Second,
    HalfOpenMaxRequests: 3,
    SuccessThreshold:    2,
    OnStateChange: func(from, to retry.CircuitState) {
        // 记录状态变化到监控系统
        logger.Warn("circuit breaker state changed",
            zap.String("from", from.String()),
            zap.String("to", to.String()))
    },
}
```

## 使用指南

### 基本使用

#### 1. 创建重试配置

```go
config := &retry.RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     30 * time.Second,
    Timeout:      5 * time.Minute,
}
```

#### 2. 选择重试策略

```go
// 指数退避
policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.1)

// 或线性退避
policy := retry.NewLinearBackoffPolicy(config, 200*time.Millisecond, 0.1)

// 或固定间隔
policy := retry.NewFixedIntervalPolicy(config, 500*time.Millisecond, 0)
```

#### 3. 创建执行器

```go
executor := retry.NewExecutor(policy)
```

#### 4. 执行带重试的函数

```go
result, err := executor.Do(ctx, func(ctx context.Context) (interface{}, error) {
    // 你的业务逻辑
    return service.Call(ctx)
})
```

### 高级使用

#### 使用断路器

```go
cbConfig := &retry.CircuitBreakerConfig{
    MaxFailures:         5,
    ResetTimeout:        60 * time.Second,
    HalfOpenMaxRequests: 3,
    SuccessThreshold:    2,
}

basePolicy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.1)
cb := retry.NewCircuitBreaker(cbConfig, basePolicy)

executor := retry.NewExecutor(cb)
```

#### 配置错误分类

```go
config := &retry.RetryConfig{
    MaxAttempts: 3,
    // 可重试错误白名单
    RetryableErrors: []error{
        ErrNetworkTimeout,
        ErrTemporaryUnavailable,
    },
    // 永久性错误黑名单（优先级更高）
    NonRetryableErrors: []error{
        ErrInvalidInput,
        ErrAuthenticationFailed,
    },
}
```

#### 添加回调

```go
executor := retry.NewExecutor(policy).
    OnRetry(func(attempt int, err error, delay time.Duration) {
        log.Printf("重试 #%d: %v (延迟 %v)", attempt+1, err, delay)
    }).
    OnSuccess(func(attempt int, duration time.Duration, result interface{}) {
        log.Printf("成功 (尝试 %d 次)", attempt)
    }).
    OnFailure(func(attempts int, duration time.Duration, lastErr error) {
        log.Printf("所有重试失败: %v", lastErr)
    })
```

#### 集成 Prometheus 指标

```go
registry := prometheus.NewRegistry()
collector, err := retry.NewRetryMetricsCollector("saga", "retry", registry)
if err != nil {
    log.Fatal(err)
}

executor := retry.NewExecutor(policy, retry.WithMetrics(collector))
```

### 与 Saga 集成

#### 全局默认重试策略

```go
sagaConfig := &saga.Config{
    DefaultRetryPolicy: retry.NewExponentialBackoffPolicy(
        &retry.RetryConfig{
            MaxAttempts:  3,
            InitialDelay: 100 * time.Millisecond,
        },
        2.0, 0.1,
    ),
}

orchestrator := saga.NewOrchestrator(sagaConfig)
```

#### 步骤级别重试策略

```go
step := saga.NewStep("payment").
    WithAction(paymentAction).
    WithCompensation(paymentCompensation).
    WithRetryPolicy(retry.NewFixedIntervalPolicy(
        &retry.RetryConfig{MaxAttempts: 5},
        1*time.Second, 0,
    ))
```

## 最佳实践

### 1. 选择合适的策略

| 场景 | 推荐策略 | 配置建议 |
|------|---------|---------|
| HTTP API 调用 | 指数退避 + 抖动 | Multiplier: 2.0, Jitter: 0.1-0.3 |
| 数据库操作 | 指数退避 | MaxAttempts: 3-5 |
| 资源竞争 | 线性退避 | Increment: 根据竞争程度 |
| 状态轮询 | 固定间隔 | Interval: 根据变化频率 |
| 关键服务调用 | 指数退避 + 断路器 | 快速熔断配置 |

### 2. 设置合理的限制

```go
config := &retry.RetryConfig{
    MaxAttempts:  3-5,           // 避免过多重试
    InitialDelay: 100ms-1s,      // 根据服务响应时间
    MaxDelay:     30s-60s,       // 防止延迟过长
    Timeout:      5m-10m,        // 考虑业务 SLA
}
```

### 3. 错误分类最佳实践

```go
// 明确定义错误类型
var (
    ErrNetworkTimeout    = errors.New("network timeout")
    ErrServiceUnavailable = errors.New("service unavailable")
    ErrBadRequest        = errors.New("bad request")
    ErrUnauthorized      = errors.New("unauthorized")
)

// 配置重试策略
config := &retry.RetryConfig{
    RetryableErrors: []error{
        ErrNetworkTimeout,
        ErrServiceUnavailable,
    },
    NonRetryableErrors: []error{
        ErrBadRequest,
        ErrUnauthorized,
    },
}
```

### 4. 监控和告警

必须监控的指标：
- 重试次数和成功率
- 断路器状态变化频率
- 重试延迟分布
- 永久性错误次数

告警设置：
```go
// 高重试率告警
if retryRate > 0.3 {  // 超过30%的请求需要重试
    alert("High retry rate detected")
}

// 断路器频繁打开告警
if circuitBreakerOpenCount > threshold {
    alert("Circuit breaker frequently opening")
}
```

### 5. 日志记录

```go
executor := retry.NewExecutor(policy, retry.WithLogger(logger)).
    OnRetry(func(attempt int, err error, delay time.Duration) {
        logger.Info("retrying",
            zap.Int("attempt", attempt),
            zap.Error(err),
            zap.Duration("delay", delay))
    }).
    OnFailure(func(attempts int, duration time.Duration, lastErr error) {
        logger.Error("all retries exhausted",
            zap.Int("attempts", attempts),
            zap.Duration("total_duration", duration),
            zap.Error(lastErr))
    })
```

### 6. 测试建议

```go
// 单元测试：使用 NoRetryConfig 禁用重试
func TestBusinessLogic(t *testing.T) {
    config := retry.NoRetryConfig()
    policy := retry.NewFixedIntervalPolicy(config, 0, 0)
    executor := retry.NewExecutor(policy)
    
    // 测试业务逻辑
}

// 集成测试：模拟真实重试场景
func TestRetryBehavior(t *testing.T) {
    config := &retry.RetryConfig{
        MaxAttempts:  3,
        InitialDelay: 10 * time.Millisecond,
    }
    policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0)
    
    // 测试重试行为
}
```

## 性能考虑

### 1. 延迟开销

重试机制引入的延迟：
- 重试判断：< 1μs
- 延迟计算：< 1μs
- 实际等待：根据策略配置

基准测试结果（参考）：
```
BenchmarkExponentialBackoffPolicy_GetRetryDelay    5000000    250 ns/op
BenchmarkLinearBackoffPolicy_GetRetryDelay         10000000   150 ns/op
BenchmarkFixedIntervalPolicy_GetRetryDelay         20000000   80 ns/op
BenchmarkCircuitBreaker_ShouldRetry                10000000   200 ns/op
```

### 2. 内存开销

每个 Executor 实例的内存占用：
- 基本开销：~200 bytes
- 带断路器：~300 bytes
- 带指标收集器：~500 bytes

建议：
- 重用 Executor 实例
- 合理设置 MaxAttempts
- 使用对象池（高并发场景）

### 3. 并发性能

所有组件都是线程安全的：
- RetryPolicy：无状态，可并发使用
- CircuitBreaker：使用 RWMutex 保护
- Executor：支持并发执行

基准测试结果：
```
BenchmarkExecutor_Concurrent-8          1000000    1200 ns/op
BenchmarkCircuitBreaker_Concurrent-8    5000000    300 ns/op
```

### 4. 优化建议

1. **避免过度重试**
   ```go
   // ❌ 不好
   config.MaxAttempts = 20  // 太多了
   
   // ✅ 好
   config.MaxAttempts = 3-5  // 合理范围
   ```

2. **使用抖动**
   ```go
   // ✅ 避免惊群效应
   policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.2)
   ```

3. **设置 MaxDelay**
   ```go
   // ✅ 防止延迟无限增长
   config.MaxDelay = 30 * time.Second
   ```

4. **使用断路器**
   ```go
   // ✅ 快速失败，节省资源
   executor := retry.NewExecutor(circuitBreaker)
   ```

## 故障排查

### 问题 1：重试次数过多

**症状**：
- 日志显示频繁重试
- 响应时间长
- 系统负载高

**可能原因**：
- 下游服务持续不可用
- MaxAttempts 设置过高
- 永久性错误被错误地重试

**排查步骤**：
1. 检查下游服务健康状态
2. 查看错误类型分布
3. 检查重试配置

**解决方案**：
```go
// 1. 使用断路器快速失败
cb := retry.NewCircuitBreaker(cbConfig, basePolicy)

// 2. 降低 MaxAttempts
config.MaxAttempts = 3

// 3. 正确配置永久性错误
config.NonRetryableErrors = []error{
    ErrBadRequest,
    ErrNotFound,
}
```

### 问题 2：断路器频繁打开/关闭

**症状**：
- 断路器状态频繁变化
- 日志显示 state change 事件很多
- 服务可用性不稳定

**可能原因**：
- MaxFailures 设置过低
- ResetTimeout 过短
- 下游服务不稳定

**排查步骤**：
1. 监控断路器指标
2. 检查下游服务稳定性
3. 分析状态转换模式

**解决方案**：
```go
// 提高失败阈值
config.MaxFailures = 10  // 从 5 增加到 10

// 增加重置超时
config.ResetTimeout = 120 * time.Second  // 从 60s 增加到 120s

// 提高成功阈值
config.SuccessThreshold = 5  // 从 2 增加到 5
```

### 问题 3：延迟时间过长

**症状**：
- 重试总耗时很长
- 用户体验差
- 超时频繁发生

**可能原因**：
- 指数退避增长过快
- MaxDelay 设置过高
- 重试次数过多

**排查步骤**：
1. 查看延迟分布
2. 检查重试策略配置
3. 分析超时日志

**解决方案**：
```go
// 1. 降低增长倍数
policy := retry.NewExponentialBackoffPolicy(config, 1.5, 0.1)  // 从 2.0 降到 1.5

// 2. 设置合理的 MaxDelay
config.MaxDelay = 10 * time.Second  // 从 30s 降到 10s

// 3. 考虑使用线性退避
policy := retry.NewLinearBackoffPolicy(config, 200*time.Millisecond, 0.1)

// 4. 设置总超时
result, err := executor.ExecuteWithTimeout(ctx, 5*time.Second, fn)
```

### 问题 4：惊群效应

**症状**：
- 多个客户端同时重试
- 下游服务出现流量峰值
- 系统负载呈现周期性峰值

**可能原因**：
- 没有使用抖动
- 抖动系数过小
- 客户端时钟同步

**排查步骤**：
1. 检查重试时间分布
2. 分析流量模式
3. 查看抖动配置

**解决方案**：
```go
// 1. 启用抖动
policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.3)  // Jitter: 30%

// 2. 使用 Decorrelated Jitter
policy.JitterType = retry.JitterTypeDecorelated

// 3. 错开初始延迟
initialDelay := baseDelay + randomOffset()
config.InitialDelay = initialDelay
```

### 问题 5：内存泄漏

**症状**：
- 内存持续增长
- GC 频繁
- 系统性能下降

**可能原因**：
- Executor 实例未复用
- 回调函数持有大对象
- Context 未正确取消

**排查步骤**：
1. 使用 pprof 分析内存
2. 检查 Executor 创建位置
3. 查看 goroutine 数量

**解决方案**：
```go
// 1. 复用 Executor
var executor *retry.Executor
func init() {
    executor = retry.NewExecutor(policy)
}

// 2. 避免回调持有大对象
executor.OnRetry(func(attempt int, err error, delay time.Duration) {
    // 只记录必要信息，不持有大对象
    logger.Info("retry", zap.Int("attempt", attempt))
})

// 3. 正确使用 Context
ctx, cancel := context.WithTimeout(context.Background(), timeout)
defer cancel()
result, err := executor.Execute(ctx, fn)
```

## 相关资源

### 文档

- [Retry Package README](../../pkg/saga/retry/README.md)
- [Saga 用户指南](../../docs/saga-user-guide.md)
- [示例代码](../../examples/saga-retry/)

### 代码

- [pkg/saga/retry/policy.go](../../pkg/saga/retry/policy.go) - 重试策略
- [pkg/saga/retry/backoff.go](../../pkg/saga/retry/backoff.go) - 退避算法
- [pkg/saga/retry/circuit_breaker.go](../../pkg/saga/retry/circuit_breaker.go) - 断路器
- [pkg/saga/retry/executor.go](../../pkg/saga/retry/executor.go) - 执行器

### 测试

- [pkg/saga/retry/*_test.go](../../pkg/saga/retry/) - 单元测试
- [pkg/saga/retry/benchmark_test.go](../../pkg/saga/retry/benchmark_test.go) - 基准测试

### 外部参考

- [AWS Architecture Blog: Exponential Backoff And Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)
- [Microsoft Azure: Retry pattern](https://docs.microsoft.com/en-us/azure/architecture/patterns/retry)
- [Martin Fowler: CircuitBreaker](https://martinfowler.com/bliki/CircuitBreaker.html)
- [Google SRE Book: Handling Overload](https://sre.google/sre-book/handling-overload/)

## 许可证

Copyright © 2025 jackelyj <dreamerlyj@gmail.com>

根据 MIT 许可证授权。


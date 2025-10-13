# Retry Package

灵活的重试策略和退避算法实现，为 Saga 分布式事务提供可靠的错误恢复能力。

## 特性

- **多种重试策略**：指数退避、线性退避、固定间隔
- **断路器模式**：防止级联故障，实现快速失败
- **抖动支持**：避免惊群效应（thundering herd）
- **灵活配置**：可自定义最大重试次数、延迟、超时等
- **执行器**：提供高级重试执行逻辑，支持回调和上下文
- **高测试覆盖率**：89% 的测试覆盖率

## 快速开始

### 指数退避重试

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/innovationmech/swit/pkg/saga/retry"
)

func main() {
    // 创建重试配置
    config := &retry.RetryConfig{
        MaxAttempts:  5,
        InitialDelay: 100 * time.Millisecond,
        MaxDelay:     30 * time.Second,
    }
    
    // 创建指数退避策略
    policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.1) // 倍数2.0，抖动0.1
    
    // 创建执行器
    executor := retry.NewExecutor(policy)
    
    // 执行带重试的函数
    result, err := executor.Do(context.Background(), func(ctx context.Context) (interface{}, error) {
        // 你的业务逻辑
        return doSomething()
    })
    
    if err != nil {
        fmt.Printf("操作失败: %v\n", err)
        return
    }
    
    fmt.Printf("操作成功: %v\n", result)
}

func doSomething() (interface{}, error) {
    // 模拟业务逻辑
    return "success", nil
}
```

### 使用断路器

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/innovationmech/swit/pkg/saga/retry"
)

func main() {
    // 创建断路器配置
    cbConfig := &retry.CircuitBreakerConfig{
        MaxFailures:         5,              // 5次失败后打开断路器
        ResetTimeout:        60 * time.Second, // 60秒后尝试半开状态
        HalfOpenMaxRequests: 3,              // 半开状态允许3个请求
        SuccessThreshold:    2,              // 2次成功后关闭断路器
    }
    
    // 创建基础重试策略
    retryConfig := &retry.RetryConfig{
        MaxAttempts:  3,
        InitialDelay: 1 * time.Second,
    }
    basePolicy := retry.NewFixedIntervalPolicy(retryConfig, 1*time.Second, 0)
    
    // 创建带断路器的策略
    cb := retry.NewCircuitBreaker(cbConfig, basePolicy)
    
    // 使用断路器
    executor := retry.NewExecutor(cb)
    
    result, err := executor.Do(context.Background(), func(ctx context.Context) (interface{}, error) {
        return callExternalService()
    })
    
    if err != nil {
        fmt.Printf("调用失败: %v\n", err)
        return
    }
    
    fmt.Printf("调用成功: %v\n", result)
}

func callExternalService() (interface{}, error) {
    // 模拟外部服务调用
    return "response", nil
}
```

### 回调和监控

```go
executor := retry.NewExecutor(policy).
    OnRetry(func(attempt int, err error, delay time.Duration) {
        fmt.Printf("重试 #%d: %v (延迟 %v)\n", attempt, err, delay)
    }).
    OnSuccess(func(attempt int, duration time.Duration, result interface{}) {
        fmt.Printf("成功 (尝试 %d 次, 耗时 %v)\n", attempt, duration)
    }).
    OnFailure(func(attempts int, duration time.Duration, lastErr error) {
        fmt.Printf("失败 (尝试 %d 次, 耗时 %v, 错误: %v)\n", attempts, duration, lastErr)
    })
```

## 重试策略

### ExponentialBackoffPolicy（指数退避）

延迟按指数增长：`delay = InitialDelay * (Multiplier ^ (attempt - 1))`

- **适用场景**：网络请求、数据库连接、外部API调用
- **优点**：快速降低负载，适合临时性故障
- **配置**：
  - `Multiplier`：增长倍数（通常2.0）
  - `Jitter`：抖动系数（0.0-1.0）
  - `JitterType`：抖动类型（Full/Equal/Decorrelated）

### LinearBackoffPolicy（线性退避）

延迟线性增长：`delay = InitialDelay + (Increment * (attempt - 1))`

- **适用场景**：资源竞争、限流场景
- **优点**：可预测的延迟增长
- **配置**：
  - `Increment`：每次增加的延迟

### FixedIntervalPolicy（固定间隔）

固定的重试间隔

- **适用场景**：定期轮询、状态检查
- **优点**：简单、可预测
- **配置**：
  - `Interval`：固定延迟时间

## 断路器（Circuit Breaker）

断路器有三种状态：

1. **Closed（关闭）**：正常工作，允许所有请求
2. **Open（打开）**：快速失败，拒绝所有请求
3. **Half-Open（半开）**：试探性恢复，允许少量请求

### 状态转换

- Closed → Open：连续失败达到 `MaxFailures`
- Open → Half-Open：等待 `ResetTimeout` 后
- Half-Open → Closed：连续成功达到 `SuccessThreshold`
- Half-Open → Open：任何失败

### 配置参数

```go
type CircuitBreakerConfig struct {
    MaxFailures         int           // 打开断路器的失败阈值
    ResetTimeout        time.Duration // 重置超时时间
    HalfOpenMaxRequests int           // 半开状态允许的最大请求数
    SuccessThreshold    int           // 关闭断路器的成功阈值
    OnStateChange       func(from, to CircuitState) // 状态变化回调
}
```

## RetryConfig（重试配置）

```go
type RetryConfig struct {
    MaxAttempts        int             // 最大尝试次数（包括首次）
    InitialDelay       time.Duration   // 初始延迟
    MaxDelay           time.Duration   // 最大延迟上限
    Timeout            time.Duration   // 总体超时时间
    RetryableErrors    []error         // 可重试的错误列表
    NonRetryableErrors []error         // 不可重试的错误列表（优先级更高）
}
```

## Executor（执行器）

执行器提供高级重试逻辑：

### 方法

- `Execute(ctx, fn)`: 执行并返回详细结果
- `ExecuteWithTimeout(ctx, timeout, fn)`: 带超时的执行
- `Do(ctx, fn)`: 简化执行（仅返回结果和错误）
- `DoWithTimeout(ctx, timeout, fn)`: 简化的带超时执行

### 回调

- `OnRetry(callback)`: 重试前调用
- `OnSuccess(callback)`: 成功时调用
- `OnFailure(callback)`: 所有重试耗尽后调用

## 抖动（Jitter）

抖动用于防止多个客户端同时重试导致的"惊群效应"：

### JitterType

- **Full**：`random(0, delay)` - 完全随机
- **Equal**：`delay/2 + random(0, delay/2)` - 平均分布
- **Decorrelated**：`random(InitialDelay, delay * 3)` - 去相关

## 错误处理

### 可重试错误判断

```go
config := &retry.RetryConfig{
    MaxAttempts: 3,
    RetryableErrors: []error{
        ErrNetworkTimeout,
        ErrTemporaryFailure,
    },
    NonRetryableErrors: []error{
        ErrInvalidInput,
        ErrAuthenticationFailed,
    },
}
```

## 最佳实践

1. **选择合适的策略**：
   - 网络请求：指数退避 + 抖动
   - 资源竞争：线性退避
   - 定期检查：固定间隔

2. **设置合理的超时**：
   - 避免无限期重试
   - 考虑下游服务的SLA

3. **使用断路器**：
   - 保护下游服务
   - 快速失败，避免级联故障

4. **监控和日志**：
   - 使用回调记录重试次数
   - 监控断路器状态变化

5. **错误分类**：
   - 明确哪些错误可重试
   - 避免重试永久性错误

## 测试

```bash
# 运行测试
go test ./pkg/saga/retry/...

# 查看覆盖率
go test ./pkg/saga/retry/... -cover

# 生成覆盖率报告
go test ./pkg/saga/retry/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 性能考虑

1. **避免过度重试**：设置合理的 `MaxAttempts`
2. **使用抖动**：减少同时重试的峰值
3. **设置MaxDelay**：防止延迟无限增长
4. **断路器**：在服务不可用时快速失败

## 示例

更多示例请参考：
- `backoff_test.go` - 退避算法测试
- `circuit_breaker_test.go` - 断路器测试
- `executor_test.go` - 执行器测试


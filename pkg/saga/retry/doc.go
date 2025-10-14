// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

/*
Package retry 提供灵活的重试策略和退避算法，为 Saga 分布式事务提供可靠的错误恢复能力。

# 概述

retry 包实现了多种重试策略和断路器模式，支持在分布式系统中处理临时性故障。
该包提供了可配置的重试逻辑、智能的退避算法和完善的监控能力。

# 特性

  - 多种重试策略：指数退避、线性退避、固定间隔
  - 断路器模式：防止级联故障，实现快速失败
  - 抖动支持：避免惊群效应（thundering herd problem）
  - 灵活配置：可自定义最大重试次数、延迟、超时等
  - 高级执行器：支持回调、上下文、指标收集
  - 错误分类：区分可重试错误和永久性错误
  - Prometheus 指标：内置指标收集支持

# 快速开始

最简单的使用方式是创建一个重试策略和执行器：

	config := &retry.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
	}

	policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.1)
	executor := retry.NewExecutor(policy)

	result, err := executor.Do(ctx, func(ctx context.Context) (interface{}, error) {
		// 你的业务逻辑
		return doSomething()
	})

# 重试策略

retry 包提供了三种主要的重试策略：

# 指数退避（ExponentialBackoffPolicy）

指数退避是最常用的重试策略，延迟按指数增长，适合处理网络请求、外部API调用等场景。

延迟计算公式：

	delay = InitialDelay * (Multiplier ^ (attempt - 1))

示例：

	config := &retry.RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
	}

	// 创建指数退避策略：倍数2.0，抖动10%
	policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.1)

	// 重试延迟序列（约）：100ms, 200ms, 400ms, 800ms, 1600ms

适用场景：
  - HTTP API 调用
  - 数据库连接
  - 消息队列操作
  - 网络临时故障

# 线性退避（LinearBackoffPolicy）

线性退避策略的延迟线性增长，提供可预测的重试间隔，适合资源竞争场景。

延迟计算公式：

	delay = InitialDelay + (Increment * (attempt - 1))

示例：

	config := &retry.RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
	}

	// 每次增加200ms
	policy := retry.NewLinearBackoffPolicy(config, 200*time.Millisecond, 0)

	// 重试延迟序列：100ms, 300ms, 500ms, 700ms, 900ms

适用场景：
  - 资源锁竞争
  - 限流（rate limiting）场景
  - 需要可预测延迟的场景

# 固定间隔（FixedIntervalPolicy）

固定间隔策略使用恒定的重试间隔，最简单直接。

示例：

	config := &retry.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 500 * time.Millisecond,
	}

	policy := retry.NewFixedIntervalPolicy(config, 500*time.Millisecond, 0)

	// 重试延迟序列：500ms, 500ms, 500ms

适用场景：
  - 定期轮询
  - 状态检查
  - 简单的重试逻辑

# 断路器（Circuit Breaker）

断路器模式用于防止级联故障，在服务不可用时快速失败，避免浪费资源。

断路器有三种状态：

 1. Closed（关闭）：正常工作，允许所有请求通过
 2. Open（打开）：服务故障，拒绝所有请求，快速失败
 3. Half-Open（半开）：试探性恢复，允许少量请求测试服务是否恢复

状态转换：

  - Closed → Open：连续失败次数达到 MaxFailures
  - Open → Half-Open：等待 ResetTimeout 时间后
  - Half-Open → Closed：连续成功次数达到 SuccessThreshold
  - Half-Open → Open：任何失败

示例：

	cbConfig := &retry.CircuitBreakerConfig{
		MaxFailures:         5,                // 5次失败后打开断路器
		ResetTimeout:        60 * time.Second, // 60秒后尝试半开
		HalfOpenMaxRequests: 3,                // 半开状态允许3个请求
		SuccessThreshold:    2,                // 2次成功后关闭断路器
		OnStateChange: func(from, to retry.CircuitState) {
			log.Printf("断路器状态: %s -> %s", from, to)
		},
	}

	basePolicy := retry.NewExponentialBackoffPolicy(retryConfig, 2.0, 0.1)
	cb := retry.NewCircuitBreaker(cbConfig, basePolicy)

	executor := retry.NewExecutor(cb)

断路器适用于：
  - 保护下游服务
  - 防止雪崩效应
  - 快速失败场景
  - 服务熔断降级

# 抖动（Jitter）

抖动用于防止多个客户端同时重试导致的"惊群效应"，通过在延迟中加入随机性来分散重试时间。

retry 包支持三种抖动类型：

  - Full Jitter：完全随机，delay = random(0, baseDelay)
  - Equal Jitter：平均分布，delay = baseDelay/2 + random(0, baseDelay/2)
  - Decorrelated Jitter：去相关，delay = random(InitialDelay, baseDelay * 3)

示例：

	policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.1)
	policy.JitterType = retry.JitterTypeEqual

建议：在高并发场景下始终使用抖动，推荐 10%-30% 的抖动系数。

# 高级功能

# 执行器（Executor）

执行器提供了高级的重试执行逻辑，支持回调、上下文取消、指标收集等。

创建执行器：

	policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.1)

	executor := retry.NewExecutor(
		policy,
		retry.WithLogger(logger),
		retry.WithMetrics(metricsCollector),
		retry.WithCircuitBreaker(circuitBreaker),
	)

执行器方法：

	// Execute: 返回详细的重试结果
	result, err := executor.Execute(ctx, fn)
	fmt.Printf("尝试次数: %d, 总耗时: %v\n", result.Attempts, result.TotalDuration)

	// ExecuteWithTimeout: 带总超时时间
	result, err := executor.ExecuteWithTimeout(ctx, 5*time.Second, fn)

	// Do: 简化版本，仅返回结果和错误
	result, err := executor.Do(ctx, fn)

	// DoWithTimeout: 简化版本 + 超时
	result, err := executor.DoWithTimeout(ctx, 5*time.Second, fn)

# 回调函数

执行器支持三种回调，用于监控和日志记录：

	executor := retry.NewExecutor(policy).
		OnRetry(func(attempt int, err error, delay time.Duration) {
			log.Printf("重试 #%d: %v (延迟 %v)", attempt+1, err, delay)
		}).
		OnSuccess(func(attempt int, duration time.Duration, result interface{}) {
			log.Printf("成功 (尝试 %d 次, 耗时 %v)", attempt, duration)
		}).
		OnFailure(func(attempts int, duration time.Duration, lastErr error) {
			log.Printf("失败 (尝试 %d 次, 耗时 %v, 错误: %v)", attempts, duration, lastErr)
		})

# 错误分类

retry 包支持区分可重试错误和永久性错误：

	config := &retry.RetryConfig{
		MaxAttempts: 3,
		// 白名单：只重试这些错误
		RetryableErrors: []error{
			ErrNetworkTimeout,
			ErrTemporaryUnavailable,
		},
		// 黑名单：不重试这些错误（优先级更高）
		NonRetryableErrors: []error{
			ErrInvalidInput,
			ErrAuthenticationFailed,
			ErrNotFound,
		},
	}

自定义错误分类器：

	type MyErrorClassifier struct{}

	func (c *MyErrorClassifier) IsRetryable(err error) bool {
		// 自定义逻辑判断错误是否可重试
		return true
	}

	func (c *MyErrorClassifier) IsPermanent(err error) bool {
		// 判断是否为永久性错误
		return false
	}

	executor := retry.NewExecutor(policy,
		retry.WithErrorClassifier(&MyErrorClassifier{}),
	)

# 指标收集

retry 包内置支持 Prometheus 指标收集：

	registry := prometheus.NewRegistry()
	collector, err := retry.NewRetryMetricsCollector("saga", "retry", registry)
	if err != nil {
		log.Fatal(err)
	}

	executor := retry.NewExecutor(policy, retry.WithMetrics(collector))

收集的指标：
  - saga_retry_attempts_total：重试尝试总次数
  - saga_retry_success_total：成功总次数
  - saga_retry_failure_total：失败总次数
  - saga_retry_aborted_total：中止总次数
  - saga_retry_attempts_current：当前正在进行的重试数
  - saga_retry_duration_seconds：重试执行耗时

# 配置最佳实践

# 选择合适的策略

 1. 网络请求：使用指数退避 + 抖动
    - Multiplier: 2.0
    - Jitter: 0.1-0.3
    - MaxAttempts: 3-5

 2. 资源竞争：使用线性退避
    - Increment: 根据竞争程度调整
    - MaxAttempts: 5-10

 3. 定期检查：使用固定间隔
    - Interval: 根据检查频率需求
    - MaxAttempts: 较高值或无限制

# 设置合理的超时

  - InitialDelay: 100ms-1s（根据服务响应时间）
  - MaxDelay: 30s-60s（防止延迟过长）
  - Timeout: 5m-10m（考虑业务SLA）
  - MaxAttempts: 3-5（避免无限重试）

# 使用断路器的建议

  - MaxFailures: 5-10（根据服务稳定性）
  - ResetTimeout: 30s-60s（给服务恢复时间）
  - HalfOpenMaxRequests: 1-3（谨慎测试）
  - SuccessThreshold: 2-3（确保服务确实恢复）

# 监控和告警

 1. 监控重试次数和成功率
 2. 监控断路器状态变化
 3. 为高重试率设置告警
 4. 记录永久性错误

# 故障排查

# 问题：重试次数过多

可能原因：
  - 下游服务持续不可用
  - MaxAttempts 设置过高
  - 错误分类不当，永久性错误被重试

解决方案：
  - 检查下游服务健康状态
  - 使用断路器快速失败
  - 正确配置 NonRetryableErrors
  - 降低 MaxAttempts

# 问题：断路器频繁打开/关闭

可能原因：
  - MaxFailures 设置过低
  - ResetTimeout 过短
  - 服务不稳定

解决方案：
  - 提高 MaxFailures 阈值
  - 增加 ResetTimeout
  - 提高 SuccessThreshold
  - 检查服务稳定性

# 问题：延迟时间过长

可能原因：
  - 指数退避增长过快
  - MaxDelay 设置过高
  - 抖动系数过大

解决方案：
  - 降低 Multiplier（如 1.5）
  - 设置合理的 MaxDelay
  - 调整 Jitter 系数
  - 考虑使用线性退避

# 问题：惊群效应

可能原因：
  - 没有使用抖动
  - 多个客户端同时重试

解决方案：
  - 启用抖动（Jitter >= 0.1）
  - 使用 Decorrelated Jitter
  - 错开初始重试时间

# 示例代码

完整的示例代码请参考：
  - examples/saga-retry/：完整的示例程序
  - pkg/saga/retry/example_test.go：Go 示例测试
  - pkg/saga/retry/*_test.go：单元测试和集成测试

# 性能考虑

 1. 避免过度重试：合理设置 MaxAttempts
 2. 使用抖动：减少同时重试的峰值负载
 3. 设置 MaxDelay：防止延迟无限增长
 4. 使用断路器：快速失败，避免资源浪费
 5. 异步重试：对非关键路径使用异步重试
 6. 批量操作：考虑批量重试而非单个重试

# 与 Saga 集成

retry 包与 Saga 协调器无缝集成，可以为 Saga 步骤和补偿操作配置重试策略：

	// 为整个 Saga 配置默认重试策略
	sagaConfig := &saga.Config{
		DefaultRetryPolicy: retry.NewExponentialBackoffPolicy(
			&retry.RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 100 * time.Millisecond,
			},
			2.0, 0.1,
		),
	}

	// 为特定步骤配置重试策略
	step := saga.NewStep("payment").
		WithRetryPolicy(retry.NewFixedIntervalPolicy(
			&retry.RetryConfig{MaxAttempts: 5},
			1*time.Second, 0,
		))

详细的 Saga 集成文档请参考：
  - specs/saga-distributed-transactions/README.md
  - docs/saga-user-guide.md

# 线程安全

retry 包的所有类型都是线程安全的，可以在并发环境中安全使用：
  - RetryPolicy 实现是无状态的，可以被多个 goroutine 共享
  - CircuitBreaker 使用互斥锁保护内部状态
  - Executor 可以被多个 goroutine 并发使用

# 测试支持

retry 包提供了测试友好的接口：

	// 在测试中使用 NoRetryConfig 禁用重试
	config := retry.NoRetryConfig()
	policy := retry.NewFixedIntervalPolicy(config, 0, 0)

	// 使用模拟的错误分类器
	executor := retry.NewExecutor(policy,
		retry.WithErrorClassifier(&mockClassifier{}),
	)

	// 断路器支持手动重置
	circuitBreaker.Reset()

# 参考资源

  - AWS Architecture Blog: Exponential Backoff And Jitter
  - Microsoft Azure: Retry pattern
  - Martin Fowler: CircuitBreaker pattern
  - Google SRE Book: Handling Overload

# 许可证

Copyright © 2025 jackelyj <dreamerlyj@gmail.com>

根据 MIT 许可证授权。详见 LICENSE 文件。
*/
package retry

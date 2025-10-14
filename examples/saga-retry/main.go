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

package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/innovationmech/swit/pkg/saga/retry"
	"go.uber.org/zap"
)

// SimulatedService 模拟一个可能失败的外部服务
type SimulatedService struct {
	name          string
	failureRate   float64 // 失败率 (0.0 - 1.0)
	callCount     int
	successCount  int
	failureCount  int
	avgLatency    time.Duration
	logger        *zap.Logger
}

func NewSimulatedService(name string, failureRate float64, avgLatency time.Duration, logger *zap.Logger) *SimulatedService {
	return &SimulatedService{
		name:        name,
		failureRate: failureRate,
		avgLatency:  avgLatency,
		logger:      logger,
	}
}

// Call 调用服务（可能失败）
func (s *SimulatedService) Call(ctx context.Context) (string, error) {
	s.callCount++

	// 模拟网络延迟
	latency := s.avgLatency + time.Duration(rand.Int63n(int64(s.avgLatency/2)))
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(latency):
	}

	// 根据失败率决定是否失败
	if rand.Float64() < s.failureRate {
		s.failureCount++
		s.logger.Warn("service call failed",
			zap.String("service", s.name),
			zap.Int("call_count", s.callCount),
			zap.Int("failure_count", s.failureCount),
		)
		return "", fmt.Errorf("service %s temporarily unavailable", s.name)
	}

	s.successCount++
	s.logger.Info("service call succeeded",
		zap.String("service", s.name),
		zap.Int("call_count", s.callCount),
		zap.Int("success_count", s.successCount),
	)
	return fmt.Sprintf("Response from %s (call #%d)", s.name, s.callCount), nil
}

// GetStats 获取服务统计信息
func (s *SimulatedService) GetStats() string {
	successRate := 0.0
	if s.callCount > 0 {
		successRate = float64(s.successCount) / float64(s.callCount) * 100
	}
	return fmt.Sprintf("%s: %d calls, %d success (%.1f%%), %d failures",
		s.name, s.callCount, s.successCount, successRate, s.failureCount)
}

func main() {
	// 初始化日志
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	fmt.Println("=== Saga Retry 机制示例 ===")

	// 运行所有示例
	runExponentialBackoffExample(logger)
	fmt.Println()

	runLinearBackoffExample(logger)
	fmt.Println()

	runFixedIntervalExample(logger)
	fmt.Println()

	runCircuitBreakerExample(logger)
	fmt.Println()

	runErrorClassificationExample(logger)
	fmt.Println()

	runCallbacksExample(logger)
	fmt.Println()

	runTimeoutExample(logger)
	fmt.Println()

	runJitterComparisonExample(logger)
	fmt.Println()

	fmt.Println("=== 所有示例完成 ===")
}

// runExponentialBackoffExample 演示指数退避重试
func runExponentialBackoffExample(logger *zap.Logger) {
	fmt.Println("1. 指数退避重试示例")
	fmt.Println("-------------------")

	service := NewSimulatedService("payment-api", 0.6, 50*time.Millisecond, logger)

	config := &retry.RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     2 * time.Second,
	}

	policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.1)
	executor := retry.NewExecutor(policy, retry.WithLogger(logger))

	ctx := context.Background()
	result, err := executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return service.Call(ctx)
	})

	if err != nil {
		fmt.Printf("❌ 操作失败: %v\n", err)
	} else {
		fmt.Printf("✅ 操作成功: %v\n", result.Result)
	}
	fmt.Printf("📊 尝试次数: %d, 总耗时: %v\n", result.Attempts, result.TotalDuration)
	fmt.Printf("📊 %s\n", service.GetStats())
}

// runLinearBackoffExample 演示线性退避重试
func runLinearBackoffExample(logger *zap.Logger) {
	fmt.Println("2. 线性退避重试示例")
	fmt.Println("-------------------")

	service := NewSimulatedService("inventory-api", 0.5, 30*time.Millisecond, logger)

	config := &retry.RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	policy := retry.NewLinearBackoffPolicy(config, 150*time.Millisecond, 0.1)
	executor := retry.NewExecutor(policy, retry.WithLogger(logger))

	ctx := context.Background()
	result, err := executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return service.Call(ctx)
	})

	if err != nil {
		fmt.Printf("❌ 操作失败: %v\n", err)
	} else {
		fmt.Printf("✅ 操作成功: %v\n", result.Result)
	}
	fmt.Printf("📊 尝试次数: %d, 总耗时: %v\n", result.Attempts, result.TotalDuration)
	fmt.Printf("📊 %s\n", service.GetStats())
}

// runFixedIntervalExample 演示固定间隔重试
func runFixedIntervalExample(logger *zap.Logger) {
	fmt.Println("3. 固定间隔重试示例")
	fmt.Println("-------------------")

	service := NewSimulatedService("notification-api", 0.4, 40*time.Millisecond, logger)

	config := &retry.RetryConfig{
		MaxAttempts:  4,
		InitialDelay: 200 * time.Millisecond,
	}

	policy := retry.NewFixedIntervalPolicy(config, 200*time.Millisecond, 0)
	executor := retry.NewExecutor(policy, retry.WithLogger(logger))

	ctx := context.Background()
	result, err := executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return service.Call(ctx)
	})

	if err != nil {
		fmt.Printf("❌ 操作失败: %v\n", err)
	} else {
		fmt.Printf("✅ 操作成功: %v\n", result.Result)
	}
	fmt.Printf("📊 尝试次数: %d, 总耗时: %v\n", result.Attempts, result.TotalDuration)
	fmt.Printf("📊 %s\n", service.GetStats())
}

// runCircuitBreakerExample 演示断路器模式
func runCircuitBreakerExample(logger *zap.Logger) {
	fmt.Println("4. 断路器模式示例")
	fmt.Println("-------------------")

	// 创建一个高失败率的服务
	service := NewSimulatedService("unstable-api", 0.8, 30*time.Millisecond, logger)

	cbConfig := &retry.CircuitBreakerConfig{
		MaxFailures:         3,
		ResetTimeout:        2 * time.Second,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
		OnStateChange: func(from, to retry.CircuitState) {
			fmt.Printf("🔄 断路器状态变化: %s -> %s\n", from, to)
		},
	}

	retryConfig := &retry.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
	}
	basePolicy := retry.NewExponentialBackoffPolicy(retryConfig, 2.0, 0)

	cb := retry.NewCircuitBreaker(cbConfig, basePolicy)
	executor := retry.NewExecutor(cb, retry.WithLogger(logger))

	ctx := context.Background()

	// 尝试多次调用，观察断路器行为
	for i := 1; i <= 8; i++ {
		fmt.Printf("\n第 %d 次调用 (断路器状态: %s):\n", i, cb.GetState())

		result, err := executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return service.Call(ctx)
		})

		if err != nil {
			if errors.Is(err, retry.ErrCircuitBreakerOpen) {
				fmt.Printf("⚡ 断路器打开，快速失败\n")
			} else {
				fmt.Printf("❌ 操作失败: %v (尝试 %d 次)\n", err, result.Attempts)
			}
		} else {
			fmt.Printf("✅ 操作成功: %v (尝试 %d 次)\n", result.Result, result.Attempts)
		}

		// 短暂延迟
		time.Sleep(300 * time.Millisecond)
	}

	fmt.Printf("\n📊 最终断路器状态: %s\n", cb.GetState())
	fmt.Printf("📊 %s\n", service.GetStats())
	metrics := cb.GetMetrics()
	fmt.Printf("📊 连续失败: %d, 连续成功: %d\n",
		metrics.ConsecutiveFailures, metrics.ConsecutiveSuccesses)
}

// runErrorClassificationExample 演示错误分类
func runErrorClassificationExample(logger *zap.Logger) {
	fmt.Println("5. 错误分类示例")
	fmt.Println("-------------------")

	var (
		errTemporary  = errors.New("temporary error")
		errPermanent  = errors.New("permanent error")
		errValidation = errors.New("validation error")
	)

	config := &retry.RetryConfig{
		MaxAttempts: 3,
		RetryableErrors: []error{
			errTemporary,
		},
		NonRetryableErrors: []error{
			errPermanent,
			errValidation,
		},
		InitialDelay: 100 * time.Millisecond,
	}

	policy := retry.NewFixedIntervalPolicy(config, 100*time.Millisecond, 0)
	executor := retry.NewExecutor(policy, retry.WithLogger(logger))

	ctx := context.Background()

	// 测试可重试错误
	fmt.Println("测试可重试错误 (temporary error):")
	attemptCount := 0
	result, err := executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		attemptCount++
		if attemptCount < 2 {
			return nil, errTemporary
		}
		return "success", nil
	})
	if err != nil {
		fmt.Printf("❌ 失败: %v (尝试 %d 次)\n", err, result.Attempts)
	} else {
		fmt.Printf("✅ 成功: %v (尝试 %d 次)\n", result.Result, result.Attempts)
	}

	// 测试永久性错误
	fmt.Println("\n测试永久性错误 (permanent error):")
	result, err = executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errPermanent
	})
	if err != nil {
		fmt.Printf("❌ 失败: %v (尝试 %d 次 - 不重试)\n", err, result.Attempts)
	}

	// 测试验证错误
	fmt.Println("\n测试验证错误 (validation error):")
	result, err = executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, errValidation
	})
	if err != nil {
		fmt.Printf("❌ 失败: %v (尝试 %d 次 - 不重试)\n", err, result.Attempts)
	}
}

// runCallbacksExample 演示回调函数
func runCallbacksExample(logger *zap.Logger) {
	fmt.Println("6. 回调函数示例")
	fmt.Println("-------------------")

	service := NewSimulatedService("callback-api", 0.6, 30*time.Millisecond, logger)

	config := &retry.RetryConfig{
		MaxAttempts:  4,
		InitialDelay: 100 * time.Millisecond,
	}

	policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0)

	executor := retry.NewExecutor(policy, retry.WithLogger(logger)).
		OnRetry(func(attempt int, err error, delay time.Duration) {
			fmt.Printf("🔄 准备重试: 尝试 #%d, 错误: %v, 延迟: %v\n", attempt+1, err, delay)
		}).
		OnSuccess(func(attempt int, duration time.Duration, result interface{}) {
			fmt.Printf("✅ 操作成功: 尝试 %d 次, 总耗时: %v\n", attempt, duration)
		}).
		OnFailure(func(attempts int, duration time.Duration, lastErr error) {
			fmt.Printf("❌ 所有重试失败: 尝试 %d 次, 总耗时: %v, 最后错误: %v\n",
				attempts, duration, lastErr)
		})

	ctx := context.Background()
	result, err := executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return service.Call(ctx)
	})

	if err == nil {
		fmt.Printf("\n📊 结果: %v\n", result.Result)
	}
	fmt.Printf("📊 %s\n", service.GetStats())
}

// runTimeoutExample 演示超时控制
func runTimeoutExample(logger *zap.Logger) {
	fmt.Println("7. 超时控制示例")
	fmt.Println("-------------------")

	// 创建一个慢速服务
	service := NewSimulatedService("slow-api", 0.7, 200*time.Millisecond, logger)

	config := &retry.RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
	}

	policy := retry.NewFixedIntervalPolicy(config, 200*time.Millisecond, 0)
	executor := retry.NewExecutor(policy, retry.WithLogger(logger))

	ctx := context.Background()

	// 设置较短的超时时间
	timeout := 800 * time.Millisecond
	fmt.Printf("设置超时: %v\n", timeout)

	result, err := executor.ExecuteWithTimeout(ctx, timeout, func(ctx context.Context) (interface{}, error) {
		return service.Call(ctx)
	})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("⏱️ 超时: 尝试了 %d 次后超时\n", result.Attempts)
		} else {
			fmt.Printf("❌ 失败: %v (尝试 %d 次)\n", err, result.Attempts)
		}
	} else {
		fmt.Printf("✅ 成功: %v (尝试 %d 次)\n", result.Result, result.Attempts)
	}
	fmt.Printf("📊 总耗时: %v\n", result.TotalDuration)
	fmt.Printf("📊 %s\n", service.GetStats())
}

// runJitterComparisonExample 演示不同抖动类型的效果
func runJitterComparisonExample(logger *zap.Logger) {
	fmt.Println("8. 抖动类型比较示例")
	fmt.Println("-------------------")

	config := &retry.RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     2 * time.Second,
	}

	jitterTypes := []struct {
		name       string
		jitterType retry.JitterType
	}{
		{"无抖动", -1},
		{"Full Jitter", retry.JitterTypeFull},
		{"Equal Jitter", retry.JitterTypeEqual},
		{"Decorrelated Jitter", retry.JitterTypeDecorelated},
	}

	for _, jt := range jitterTypes {
		fmt.Printf("\n%s:\n", jt.name)

		policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.3)
		if jt.jitterType >= 0 {
			policy.JitterType = jt.jitterType
		} else {
			policy.Jitter = 0 // 无抖动
		}

		// 打印前5次重试的延迟
		fmt.Print("  延迟序列: ")
		for i := 1; i <= 5; i++ {
			delay := policy.GetRetryDelay(i)
			fmt.Printf("%v ", delay.Round(time.Millisecond))
		}
		fmt.Println()
	}
}

// 辅助函数：运行示例时检查环境
func init() {
	// 设置随机种子
	rand.Seed(time.Now().UnixNano())

	// 检查是否在 CI 环境中
	if os.Getenv("CI") != "" {
		fmt.Println("注意：在 CI 环境中运行，某些示例可能表现不同")
	}
}


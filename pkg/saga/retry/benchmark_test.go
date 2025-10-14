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

package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

// BenchmarkExponentialBackoffPolicy_GetRetryDelay 测试指数退避策略的延迟计算性能
func BenchmarkExponentialBackoffPolicy_GetRetryDelay(b *testing.B) {
	config := &RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
	}

	policy := NewExponentialBackoffPolicy(config, 2.0, 0.1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = policy.GetRetryDelay(5)
	}
}

// BenchmarkLinearBackoffPolicy_GetRetryDelay 测试线性退避策略的延迟计算性能
func BenchmarkLinearBackoffPolicy_GetRetryDelay(b *testing.B) {
	config := &RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
	}

	policy := NewLinearBackoffPolicy(config, 200*time.Millisecond, 0.1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = policy.GetRetryDelay(5)
	}
}

// BenchmarkFixedIntervalPolicy_GetRetryDelay 测试固定间隔策略的延迟计算性能
func BenchmarkFixedIntervalPolicy_GetRetryDelay(b *testing.B) {
	config := &RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 100*time.Millisecond, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = policy.GetRetryDelay(5)
	}
}

// BenchmarkJitterTypes 比较不同抖动类型的性能
func BenchmarkJitterTypes(b *testing.B) {
	config := &RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
	}

	jitterTypes := []struct {
		name string
		jt   JitterType
	}{
		{"Full", JitterTypeFull},
		{"Equal", JitterTypeEqual},
		{"Decorrelated", JitterTypeDecorelated},
	}

	for _, tt := range jitterTypes {
		b.Run(tt.name, func(b *testing.B) {
			policy := NewExponentialBackoffPolicy(config, 2.0, 0.3)
			policy.JitterType = tt.jt

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = policy.GetRetryDelay(5)
			}
		})
	}
}

// BenchmarkCircuitBreaker_ShouldRetry 测试断路器判断性能
func BenchmarkCircuitBreaker_ShouldRetry(b *testing.B) {
	cbConfig := &CircuitBreakerConfig{
		MaxFailures:         5,
		ResetTimeout:        60 * time.Second,
		HalfOpenMaxRequests: 3,
		SuccessThreshold:    2,
	}

	retryConfig := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
	}
	basePolicy := NewFixedIntervalPolicy(retryConfig, 100*time.Millisecond, 0)

	cb := NewCircuitBreaker(cbConfig, basePolicy)

	err := errors.New("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.ShouldRetry(err, 1)
	}
}

// BenchmarkCircuitBreaker_StateTransitions 测试断路器状态转换性能
func BenchmarkCircuitBreaker_StateTransitions(b *testing.B) {
	cbConfig := &CircuitBreakerConfig{
		MaxFailures:         3,
		ResetTimeout:        100 * time.Millisecond,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	retryConfig := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}
	basePolicy := NewFixedIntervalPolicy(retryConfig, 10*time.Millisecond, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb := NewCircuitBreaker(cbConfig, basePolicy)

		// 触发失败，打开断路器
		err := errors.New("test error")
		for j := 0; j < 3; j++ {
			cb.ShouldRetry(err, j+1)
		}

		// 等待重置
		time.Sleep(110 * time.Millisecond)

		// 触发成功，关闭断路器
		cb.ShouldRetry(nil, 0)
		cb.ShouldRetry(nil, 0)
	}
}

// BenchmarkExecutor_Execute_Success 测试成功执行的性能
func BenchmarkExecutor_Execute_Success(b *testing.B) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 1*time.Millisecond, 0)
	executor := NewExecutor(policy)

	ctx := context.Background()
	fn := func(ctx context.Context) (interface{}, error) {
		return "success", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(ctx, fn)
	}
}

// BenchmarkExecutor_Execute_WithRetries 测试带重试的执行性能
func BenchmarkExecutor_Execute_WithRetries(b *testing.B) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 1*time.Millisecond, 0)
	executor := NewExecutor(policy)

	ctx := context.Background()
	err := errors.New("temporary error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempt := 0
		fn := func(ctx context.Context) (interface{}, error) {
			attempt++
			if attempt < 2 {
				return nil, err
			}
			return "success", nil
		}
		_, _ = executor.Execute(ctx, fn)
	}
}

// BenchmarkExecutor_Execute_AllStrategies 比较不同策略的执行性能
func BenchmarkExecutor_Execute_AllStrategies(b *testing.B) {
	strategies := []struct {
		name   string
		policy RetryPolicy
	}{
		{
			"ExponentialBackoff",
			NewExponentialBackoffPolicy(
				&RetryConfig{MaxAttempts: 3, InitialDelay: 1 * time.Millisecond},
				2.0, 0,
			),
		},
		{
			"LinearBackoff",
			NewLinearBackoffPolicy(
				&RetryConfig{MaxAttempts: 3, InitialDelay: 1 * time.Millisecond},
				1*time.Millisecond, 0,
			),
		},
		{
			"FixedInterval",
			NewFixedIntervalPolicy(
				&RetryConfig{MaxAttempts: 3, InitialDelay: 1 * time.Millisecond},
				1*time.Millisecond, 0,
			),
		},
	}

	for _, s := range strategies {
		b.Run(s.name, func(b *testing.B) {
			executor := NewExecutor(s.policy)
			ctx := context.Background()

			err := errors.New("temporary error")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				attempt := 0
				fn := func(ctx context.Context) (interface{}, error) {
					attempt++
					if attempt < 2 {
						return nil, err
					}
					return "success", nil
				}
				_, _ = executor.Execute(ctx, fn)
			}
		})
	}
}

// BenchmarkExecutor_WithCallbacks 测试带回调的执行性能
func BenchmarkExecutor_WithCallbacks(b *testing.B) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 1*time.Millisecond, 0)

	executor := NewExecutor(policy).
		OnRetry(func(attempt int, err error, delay time.Duration) {
			// 模拟日志记录
		}).
		OnSuccess(func(attempt int, duration time.Duration, result interface{}) {
			// 模拟成功回调
		}).
		OnFailure(func(attempts int, duration time.Duration, lastErr error) {
			// 模拟失败回调
		})

	ctx := context.Background()
	err := errors.New("temporary error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempt := 0
		fn := func(ctx context.Context) (interface{}, error) {
			attempt++
			if attempt < 2 {
				return nil, err
			}
			return "success", nil
		}
		_, _ = executor.Execute(ctx, fn)
	}
}

// BenchmarkRetryConfig_IsRetryableError 测试错误判断性能
func BenchmarkRetryConfig_IsRetryableError(b *testing.B) {
	errRetryable := errors.New("retryable error")
	errNonRetryable := errors.New("non-retryable error")

	config := &RetryConfig{
		MaxAttempts: 3,
		RetryableErrors: []error{
			errRetryable,
		},
		NonRetryableErrors: []error{
			errNonRetryable,
		},
	}

	b.Run("Retryable", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = config.IsRetryableError(errRetryable)
		}
	})

	b.Run("NonRetryable", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = config.IsRetryableError(errNonRetryable)
		}
	})
}

// BenchmarkCircuitBreaker_Concurrent 测试并发场景下的断路器性能
func BenchmarkCircuitBreaker_Concurrent(b *testing.B) {
	cbConfig := &CircuitBreakerConfig{
		MaxFailures:         10,
		ResetTimeout:        60 * time.Second,
		HalfOpenMaxRequests: 5,
		SuccessThreshold:    3,
	}

	retryConfig := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
	}
	basePolicy := NewFixedIntervalPolicy(retryConfig, 1*time.Millisecond, 0)

	cb := NewCircuitBreaker(cbConfig, basePolicy)

	err := errors.New("test error")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.ShouldRetry(err, 1)
		}
	})
}

// BenchmarkExecutor_Concurrent 测试并发执行性能
func BenchmarkExecutor_Concurrent(b *testing.B) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 1*time.Millisecond, 0)
	executor := NewExecutor(policy)

	fn := func(ctx context.Context) (interface{}, error) {
		return "success", nil
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			_, _ = executor.Execute(ctx, fn)
		}
	})
}

// BenchmarkMemoryAllocation 测试内存分配
func BenchmarkMemoryAllocation(b *testing.B) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
	}

	b.Run("CreatePolicy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewExponentialBackoffPolicy(config, 2.0, 0.1)
		}
	})

	b.Run("CreateExecutor", func(b *testing.B) {
		policy := NewExponentialBackoffPolicy(config, 2.0, 0.1)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewExecutor(policy)
		}
	})

	b.Run("Execute", func(b *testing.B) {
		policy := NewExponentialBackoffPolicy(config, 2.0, 0.1)
		executor := NewExecutor(policy)
		ctx := context.Background()
		fn := func(ctx context.Context) (interface{}, error) {
			return "success", nil
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = executor.Execute(ctx, fn)
		}
	})
}

// BenchmarkRetryScenarios 测试真实重试场景
func BenchmarkRetryScenarios(b *testing.B) {
	scenarios := []struct {
		name         string
		maxAttempts  int
		failAttempts int
		delay        time.Duration
	}{
		{"QuickSuccess", 3, 0, 1 * time.Millisecond},
		{"OneRetry", 3, 1, 1 * time.Millisecond},
		{"TwoRetries", 3, 2, 1 * time.Millisecond},
		{"AllFailed", 3, 3, 1 * time.Millisecond},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			config := &RetryConfig{
				MaxAttempts:  s.maxAttempts,
				InitialDelay: s.delay,
			}
			policy := NewFixedIntervalPolicy(config, s.delay, 0)
			executor := NewExecutor(policy)

			ctx := context.Background()
			err := errors.New("temporary error")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				attempt := 0
				fn := func(ctx context.Context) (interface{}, error) {
					attempt++
					if attempt <= s.failAttempts {
						return nil, err
					}
					return "success", nil
				}
				_, _ = executor.Execute(ctx, fn)
			}
		})
	}
}

// BenchmarkCircuitBreakerStates 测试断路器不同状态下的性能
func BenchmarkCircuitBreakerStates(b *testing.B) {
	states := []struct {
		name  string
		setup func(*CircuitBreaker)
	}{
		{
			"Closed",
			func(cb *CircuitBreaker) {
				// 默认状态是 closed
			},
		},
		{
			"Open",
			func(cb *CircuitBreaker) {
				// 触发多次失败，打开断路器
				err := errors.New("test error")
				for i := 0; i < 5; i++ {
					cb.ShouldRetry(err, i+1)
				}
			},
		},
	}

	for _, s := range states {
		b.Run(s.name, func(b *testing.B) {
			cbConfig := &CircuitBreakerConfig{
				MaxFailures:         5,
				ResetTimeout:        60 * time.Second,
				HalfOpenMaxRequests: 3,
				SuccessThreshold:    2,
			}

			retryConfig := &RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 1 * time.Millisecond,
			}
			basePolicy := NewFixedIntervalPolicy(retryConfig, 1*time.Millisecond, 0)

			cb := NewCircuitBreaker(cbConfig, basePolicy)
			s.setup(cb)

			err := errors.New("test error")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = cb.ShouldRetry(err, 1)
			}
		})
	}
}


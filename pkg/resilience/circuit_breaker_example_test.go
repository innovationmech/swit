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

package resilience_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/innovationmech/swit/pkg/resilience"
)

// ExampleCircuitBreaker_basic 演示熔断器的基本使用。
func ExampleCircuitBreaker_basic() {
	// 创建熔断器配置
	config := resilience.CircuitBreakerConfig{
		Name:                 "api-service",
		MaxRequests:          1,
		Interval:             60 * time.Second,
		Timeout:              30 * time.Second,
		FailureThreshold:     5,
		FailureRateThreshold: 0.5,
		MinimumRequests:      10,
	}

	// 创建熔断器
	cb, err := resilience.NewCircuitBreaker(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// 执行操作
	err = cb.Execute(ctx, func() error {
		// 你的业务逻辑
		return nil
	})

	if err != nil {
		if errors.Is(err, resilience.ErrOpenState) {
			fmt.Println("Circuit breaker is open")
		} else {
			fmt.Println("Operation failed:", err)
		}
	} else {
		fmt.Println("Operation succeeded")
	}

	// Output: Operation succeeded
}

// ExampleCircuitBreaker_withResult 演示如何使用熔断器执行有返回值的操作。
func ExampleCircuitBreaker_withResult() {
	config := resilience.DefaultCircuitBreakerConfig()
	cb, err := resilience.NewCircuitBreaker(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// 执行带返回值的操作
	result, err := resilience.ExecuteWithResult(ctx, cb, func() (string, error) {
		// 模拟 API 调用
		return "success response", nil
	})

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Result:", result)
	// Output: Result: success response
}

// ExampleCircuitBreaker_stateTransitions 演示熔断器的状态转换。
func ExampleCircuitBreaker_stateTransitions() {
	config := resilience.CircuitBreakerConfig{
		Name:                 "demo",
		MaxRequests:          1,
		Interval:             time.Minute,
		Timeout:              100 * time.Millisecond,
		FailureThreshold:     3,
		FailureRateThreshold: 0.5,
		MinimumRequests:      3,
		OnStateChange: func(name string, from resilience.State, to resilience.State) {
			fmt.Printf("State changed from %s to %s\n", from, to)
		},
	}

	cb, _ := resilience.NewCircuitBreaker(config)
	ctx := context.Background()

	// 触发失败，打开熔断器
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func() error {
			return errors.New("service unavailable")
		})
	}

	// 尝试执行请求（将被拒绝）
	err := cb.Execute(ctx, func() error {
		return nil
	})

	if errors.Is(err, resilience.ErrOpenState) {
		fmt.Println("Request rejected: circuit breaker is open")
	}

	// Output:
	// State changed from closed to open
	// Request rejected: circuit breaker is open
}

// ExampleCircuitBreaker_customReadyToTrip 演示自定义熔断触发逻辑。
func ExampleCircuitBreaker_customReadyToTrip() {
	config := resilience.CircuitBreakerConfig{
		Name:                 "custom",
		MaxRequests:          1,
		Interval:             time.Minute,
		Timeout:              time.Second,
		FailureRateThreshold: 0.5,
		MinimumRequests:      5,
		// 自定义触发逻辑：连续3次失败就打开
		ReadyToTrip: func(counts resilience.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	}

	cb, _ := resilience.NewCircuitBreaker(config)
	ctx := context.Background()

	// 连续3次失败会触发熔断
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func() error {
			return errors.New("failed")
		})
	}

	fmt.Println("State:", cb.GetState())
	// Output: State: open
}

// ExampleCircuitBreaker_metrics 演示如何获取和使用指标。
func ExampleCircuitBreaker_metrics() {
	config := resilience.DefaultCircuitBreakerConfig()
	cb, _ := resilience.NewCircuitBreaker(config)
	ctx := context.Background()

	// 执行一些操作
	_ = cb.Execute(ctx, func() error { return nil })
	_ = cb.Execute(ctx, func() error { return errors.New("error") })

	// 获取指标
	metrics := cb.GetMetrics()
	fmt.Printf("Total requests: %d\n", metrics.GetTotalRequests())
	fmt.Printf("Total successes: %d\n", metrics.GetTotalSuccesses())
	fmt.Printf("Total failures: %d\n", metrics.GetTotalFailures())
	fmt.Printf("Failure rate: %.2f\n", metrics.GetFailureRate())

	// Output:
	// Total requests: 2
	// Total successes: 1
	// Total failures: 1
	// Failure rate: 0.50
}

// ExampleCircuitBreaker_reset 演示如何重置熔断器。
func ExampleCircuitBreaker_reset() {
	config := resilience.CircuitBreakerConfig{
		Name:                 "resetable",
		MaxRequests:          1,
		Interval:             time.Minute,
		Timeout:              time.Hour, // 很长的超时
		FailureThreshold:     2,
		FailureRateThreshold: 0.5,
		MinimumRequests:      2,
	}

	cb, _ := resilience.NewCircuitBreaker(config)
	ctx := context.Background()

	// 触发熔断
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func() error {
			return errors.New("error")
		})
	}

	fmt.Println("State before reset:", cb.GetState())

	// 手动重置熔断器（例如在服务恢复后）
	cb.Reset()

	fmt.Println("State after reset:", cb.GetState())

	// 现在可以正常执行请求
	err := cb.Execute(ctx, func() error {
		return nil
	})

	if err == nil {
		fmt.Println("Request succeeded after reset")
	}

	// Output:
	// State before reset: open
	// State after reset: closed
	// Request succeeded after reset
}

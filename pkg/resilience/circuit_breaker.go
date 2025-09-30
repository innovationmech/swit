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

package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrOpenState 表示熔断器处于打开状态
	ErrOpenState = errors.New("circuit breaker is open")

	// ErrTooManyRequests 表示半开状态下请求过多
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// CircuitBreaker 实现熔断器模式。
//
// 熔断器有三种状态：
// - Closed: 正常处理请求，统计失败次数/失败率
// - Open: 直接拒绝请求，快速失败
// - Half-Open: 允许有限数量的请求通过以测试服务是否恢复
type CircuitBreaker struct {
	name   string
	config CircuitBreakerConfig

	mu              sync.Mutex
	state           State
	generation      uint64
	counts          Counts
	expiry          time.Time
	halfOpenSuccess uint32

	metrics *CircuitBreakerMetrics
}

// NewCircuitBreaker 创建一个新的熔断器实例。
func NewCircuitBreaker(config CircuitBreakerConfig) (*CircuitBreaker, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// 如果没有提供 ReadyToTrip 函数，使用默认实现
	if config.ReadyToTrip == nil {
		config.ReadyToTrip = func(counts Counts) bool {
			// 请求数必须达到最小阈值
			if counts.Requests < config.MinimumRequests {
				return false
			}
			// 失败次数超过阈值或失败率超过阈值
			return counts.TotalFailures >= config.FailureThreshold ||
				counts.FailureRate() >= config.FailureRateThreshold
		}
	}

	// 如果没有提供 IsSuccessful 函数，使用默认实现
	if config.IsSuccessful == nil {
		config.IsSuccessful = func(err error) bool {
			return err == nil
		}
	}

	cb := &CircuitBreaker{
		name:    config.Name,
		config:  config,
		state:   StateClosed,
		expiry:  time.Now().Add(config.Interval),
		metrics: NewCircuitBreakerMetrics(),
	}

	return cb, nil
}

// Execute 执行给定的操作，并根据结果更新熔断器状态。
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	generation, err := cb.beforeRequest()
	if err != nil {
		cb.metrics.RecordRejection()
		return err
	}

	defer func() {
		if e := recover(); e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result := operation()
	success := cb.config.IsSuccessful(result)
	cb.afterRequest(generation, success)

	return result
}

// ExecuteWithResult 执行给定的操作并返回结果。
func ExecuteWithResult[T any](ctx context.Context, cb *CircuitBreaker, operation func() (T, error)) (T, error) {
	var zero T

	// 检查 context 是否已取消
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	default:
	}

	generation, err := cb.beforeRequest()
	if err != nil {
		cb.metrics.RecordRejection()
		return zero, err
	}

	defer func() {
		if e := recover(); e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, opErr := operation()
	success := cb.config.IsSuccessful(opErr)
	cb.afterRequest(generation, success)

	return result, opErr
}

// GetState 返回当前熔断器状态。
func (cb *CircuitBreaker) GetState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// GetCounts 返回当前统计计数。
func (cb *CircuitBreaker) GetCounts() Counts {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return cb.counts
}

// GetMetrics 返回熔断器的指标统计。
func (cb *CircuitBreaker) GetMetrics() *CircuitBreakerMetrics {
	return cb.metrics
}

// Reset 重置熔断器到关闭状态。
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.toNewGeneration(time.Now(), StateClosed)
}

// beforeRequest 在请求执行前检查熔断器状态。
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, ErrOpenState
	}

	if state == StateHalfOpen && cb.halfOpenSuccess >= cb.config.MaxRequests {
		return generation, ErrTooManyRequests
	}

	if state == StateHalfOpen {
		cb.halfOpenSuccess++
	}

	cb.metrics.RecordRequest()
	return generation, nil
}

// afterRequest 在请求执行后更新熔断器状态。
func (cb *CircuitBreaker) afterRequest(generation uint64, success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, currentGeneration := cb.currentState(now)

	// 如果代数不匹配，说明状态已经改变，忽略这次结果
	if generation != currentGeneration {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// onSuccess 处理成功的请求。
func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	cb.metrics.RecordSuccess()

	switch state {
	case StateClosed:
		cb.counts.onSuccess()
	case StateHalfOpen:
		cb.counts.onSuccess()
		// 如果半开状态下所有测试请求都成功，转换到关闭状态
		if cb.counts.ConsecutiveSuccesses >= cb.config.MaxRequests {
			cb.toNewGeneration(now, StateClosed)
		}
	}
}

// onFailure 处理失败的请求。
func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	cb.metrics.RecordFailure()

	switch state {
	case StateClosed:
		cb.counts.onFailure()
		// 检查是否应该打开熔断器
		if cb.config.ReadyToTrip(cb.counts) {
			cb.toNewGeneration(now, StateOpen)
		}
	case StateHalfOpen:
		// 半开状态下任何失败都会重新打开熔断器
		cb.toNewGeneration(now, StateOpen)
	}
}

// currentState 返回当前状态和代数。
// 如果处于打开状态且超时，则自动转换到半开状态。
func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		// 检查是否需要重置统计窗口
		if cb.expiry.Before(now) {
			cb.toNewGeneration(now, StateClosed)
		}
	case StateOpen:
		// 检查是否应该进入半开状态
		if cb.expiry.Before(now) {
			cb.toNewGeneration(now, StateHalfOpen)
		}
	}
	return cb.state, cb.generation
}

// toNewGeneration 转换到新的状态和代数。
func (cb *CircuitBreaker) toNewGeneration(now time.Time, newState State) {
	if cb.state != newState {
		cb.beforeStateChange(cb.state, newState)
	}

	cb.generation++
	cb.counts.clear()
	cb.halfOpenSuccess = 0

	var expiry time.Time
	switch newState {
	case StateClosed:
		expiry = now.Add(cb.config.Interval)
		cb.metrics.RecordStateChange("closed")
	case StateOpen:
		expiry = now.Add(cb.config.Timeout)
		cb.metrics.RecordStateChange("open")
	case StateHalfOpen:
		expiry = now.Add(cb.config.Timeout)
		cb.metrics.RecordStateChange("half_open")
	}

	cb.state = newState
	cb.expiry = expiry
}

// beforeStateChange 在状态改变前调用回调函数。
func (cb *CircuitBreaker) beforeStateChange(from State, to State) {
	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(cb.name, from, to)
	}
}

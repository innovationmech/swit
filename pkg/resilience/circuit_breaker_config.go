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
	"fmt"
	"time"
)

// CircuitBreakerConfig 定义熔断器的配置参数。
//
// 熔断器通过监控失败次数和失败率来决定是否打开电路，
// 在打开状态下快速失败，避免对故障服务施加更多压力。
type CircuitBreakerConfig struct {
	// Name 熔断器名称，用于日志和指标标识
	Name string `json:"name" yaml:"name"`

	// MaxRequests 半开状态下允许通过的最大请求数
	MaxRequests uint32 `json:"max_requests" yaml:"max_requests"`

	// Interval 统计窗口期，在此期间内计算失败率
	Interval time.Duration `json:"interval" yaml:"interval"`

	// Timeout 打开状态持续时间，超时后进入半开状态
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// ReadyToTrip 决定是否应该打开熔断器的函数
	// 参数为当前计数器状态（请求总数、失败数）
	ReadyToTrip func(counts Counts) bool `json:"-" yaml:"-"`

	// OnStateChange 状态变化时的回调函数
	OnStateChange func(name string, from State, to State) `json:"-" yaml:"-"`

	// IsSuccessful 判断结果是否成功的函数
	// 如果未设置，则将 error == nil 视为成功
	IsSuccessful func(err error) bool `json:"-" yaml:"-"`

	// FailureThreshold 失败次数阈值（当 ReadyToTrip 未设置时使用）
	FailureThreshold uint32 `json:"failure_threshold" yaml:"failure_threshold"`

	// FailureRateThreshold 失败率阈值（0-1），当 ReadyToTrip 未设置时使用
	FailureRateThreshold float64 `json:"failure_rate_threshold" yaml:"failure_rate_threshold"`

	// MinimumRequests 最小请求数，只有达到此数量才会计算失败率
	MinimumRequests uint32 `json:"minimum_requests" yaml:"minimum_requests"`
}

// DefaultCircuitBreakerConfig 返回熔断器的默认配置。
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:                 "default",
		MaxRequests:          1,
		Interval:             60 * time.Second,
		Timeout:              60 * time.Second,
		FailureThreshold:     5,
		FailureRateThreshold: 0.5,
		MinimumRequests:      10,
		ReadyToTrip:          nil, // 使用默认实现
		OnStateChange:        nil,
		IsSuccessful:         nil, // 使用默认实现（error == nil）
	}
}

// Validate 校验配置合法性。
func (c *CircuitBreakerConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("circuit breaker name cannot be empty")
	}
	if c.MaxRequests == 0 {
		return fmt.Errorf("max_requests must be greater than 0")
	}
	if c.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.FailureRateThreshold < 0 || c.FailureRateThreshold > 1 {
		return fmt.Errorf("failure_rate_threshold must be between 0 and 1")
	}
	if c.MinimumRequests == 0 {
		return fmt.Errorf("minimum_requests must be greater than 0")
	}
	return nil
}

// State 表示熔断器的状态。
type State int

const (
	// StateClosed 关闭状态，正常处理请求
	StateClosed State = iota
	// StateHalfOpen 半开状态，允许有限数量的请求通过以测试服务是否恢复
	StateHalfOpen
	// StateOpen 打开状态，直接拒绝请求
	StateOpen
)

// String 返回状态的字符串表示。
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return fmt.Sprintf("unknown state: %d", s)
	}
}

// Counts 记录熔断器的请求统计。
type Counts struct {
	// Requests 总请求数
	Requests uint32
	// TotalSuccesses 总成功数
	TotalSuccesses uint32
	// TotalFailures 总失败数
	TotalFailures uint32
	// ConsecutiveSuccesses 连续成功数
	ConsecutiveSuccesses uint32
	// ConsecutiveFailures 连续失败数
	ConsecutiveFailures uint32
}

// FailureRate 计算失败率（0-1）。
func (c *Counts) FailureRate() float64 {
	if c.Requests == 0 {
		return 0
	}
	return float64(c.TotalFailures) / float64(c.Requests)
}

// onSuccess 记录一次成功。
func (c *Counts) onSuccess() {
	c.Requests++
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

// onFailure 记录一次失败。
func (c *Counts) onFailure() {
	c.Requests++
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

// clear 清空计数器。
func (c *Counts) clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

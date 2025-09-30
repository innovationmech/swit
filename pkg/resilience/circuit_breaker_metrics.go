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
	"sync/atomic"
)

// CircuitBreakerMetrics 包含熔断器的度量指标。
type CircuitBreakerMetrics struct {
	// TotalRequests 总请求数
	TotalRequests atomic.Int64

	// TotalSuccesses 总成功数
	TotalSuccesses atomic.Int64

	// TotalFailures 总失败数
	TotalFailures atomic.Int64

	// TotalRejections 总拒绝数（熔断器打开或半开状态下的请求）
	TotalRejections atomic.Int64

	// StateChanges 状态变更次数（按状态分类）
	StateChangesToClosed   atomic.Int64
	StateChangesToOpen     atomic.Int64
	StateChangesToHalfOpen atomic.Int64
}

// NewCircuitBreakerMetrics 创建新的度量指标实例。
func NewCircuitBreakerMetrics() *CircuitBreakerMetrics {
	return &CircuitBreakerMetrics{}
}

// RecordRequest 记录一次请求。
func (m *CircuitBreakerMetrics) RecordRequest() {
	m.TotalRequests.Add(1)
}

// RecordSuccess 记录一次成功。
func (m *CircuitBreakerMetrics) RecordSuccess() {
	m.TotalSuccesses.Add(1)
}

// RecordFailure 记录一次失败。
func (m *CircuitBreakerMetrics) RecordFailure() {
	m.TotalFailures.Add(1)
}

// RecordRejection 记录一次拒绝。
func (m *CircuitBreakerMetrics) RecordRejection() {
	m.TotalRejections.Add(1)
}

// RecordStateChange 记录状态变更。
func (m *CircuitBreakerMetrics) RecordStateChange(state string) {
	switch state {
	case "closed":
		m.StateChangesToClosed.Add(1)
	case "open":
		m.StateChangesToOpen.Add(1)
	case "half_open":
		m.StateChangesToHalfOpen.Add(1)
	}
}

// GetTotalRequests 返回总请求数。
func (m *CircuitBreakerMetrics) GetTotalRequests() int64 {
	return m.TotalRequests.Load()
}

// GetTotalSuccesses 返回总成功数。
func (m *CircuitBreakerMetrics) GetTotalSuccesses() int64 {
	return m.TotalSuccesses.Load()
}

// GetTotalFailures 返回总失败数。
func (m *CircuitBreakerMetrics) GetTotalFailures() int64 {
	return m.TotalFailures.Load()
}

// GetTotalRejections 返回总拒绝数。
func (m *CircuitBreakerMetrics) GetTotalRejections() int64 {
	return m.TotalRejections.Load()
}

// GetStateChangesToClosed 返回转换到关闭状态的次数。
func (m *CircuitBreakerMetrics) GetStateChangesToClosed() int64 {
	return m.StateChangesToClosed.Load()
}

// GetStateChangesToOpen 返回转换到打开状态的次数。
func (m *CircuitBreakerMetrics) GetStateChangesToOpen() int64 {
	return m.StateChangesToOpen.Load()
}

// GetStateChangesToHalfOpen 返回转换到半开状态的次数。
func (m *CircuitBreakerMetrics) GetStateChangesToHalfOpen() int64 {
	return m.StateChangesToHalfOpen.Load()
}

// GetFailureRate 返回失败率（0-1）。
func (m *CircuitBreakerMetrics) GetFailureRate() float64 {
	total := m.TotalRequests.Load()
	if total == 0 {
		return 0
	}
	failures := m.TotalFailures.Load()
	return float64(failures) / float64(total)
}

// Reset 重置所有度量指标。
func (m *CircuitBreakerMetrics) Reset() {
	m.TotalRequests.Store(0)
	m.TotalSuccesses.Store(0)
	m.TotalFailures.Store(0)
	m.TotalRejections.Store(0)
	m.StateChangesToClosed.Store(0)
	m.StateChangesToOpen.Store(0)
	m.StateChangesToHalfOpen.Store(0)
}

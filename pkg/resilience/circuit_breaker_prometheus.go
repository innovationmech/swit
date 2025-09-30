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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// CircuitBreakerPrometheusMetrics 包含熔断器的 Prometheus 指标。
type CircuitBreakerPrometheusMetrics struct {
	Requests        *prometheus.CounterVec
	Successes       *prometheus.CounterVec
	Failures        *prometheus.CounterVec
	Rejections      *prometheus.CounterVec
	StateChanges    *prometheus.CounterVec
	State           *prometheus.GaugeVec
	FailureRate     *prometheus.GaugeVec
	RequestDuration *prometheus.HistogramVec
}

// NewCircuitBreakerPrometheusMetrics 创建并注册熔断器 Prometheus 指标。
func NewCircuitBreakerPrometheusMetrics(namespace string, subsystem string) *CircuitBreakerPrometheusMetrics {
	if namespace == "" {
		namespace = "swit"
	}
	if subsystem == "" {
		subsystem = "circuit_breaker"
	}

	return &CircuitBreakerPrometheusMetrics{
		Requests: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "requests_total",
			Help:      "Total number of requests processed by the circuit breaker",
		}, []string{"name"}),

		Successes: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "successes_total",
			Help:      "Total number of successful requests",
		}, []string{"name"}),

		Failures: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "failures_total",
			Help:      "Total number of failed requests",
		}, []string{"name"}),

		Rejections: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "rejections_total",
			Help:      "Total number of rejected requests (circuit breaker open or half-open)",
		}, []string{"name"}),

		StateChanges: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "state_changes_total",
			Help:      "Total number of state changes",
		}, []string{"name", "state"}),

		State: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "state",
			Help:      "Current state of the circuit breaker (0=closed, 1=half-open, 2=open)",
		}, []string{"name"}),

		FailureRate: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "failure_rate",
			Help:      "Current failure rate of the circuit breaker",
		}, []string{"name"}),

		RequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "request_duration_seconds",
			Help:      "Duration of requests processed by the circuit breaker",
			Buckets:   prometheus.DefBuckets,
		}, []string{"name", "result"}),
	}
}

// RecordRequest 记录一次请求。
func (p *CircuitBreakerPrometheusMetrics) RecordRequest(name string) {
	p.Requests.WithLabelValues(name).Inc()
}

// RecordSuccess 记录一次成功。
func (p *CircuitBreakerPrometheusMetrics) RecordSuccess(name string) {
	p.Successes.WithLabelValues(name).Inc()
}

// RecordFailure 记录一次失败。
func (p *CircuitBreakerPrometheusMetrics) RecordFailure(name string) {
	p.Failures.WithLabelValues(name).Inc()
}

// RecordRejection 记录一次拒绝。
func (p *CircuitBreakerPrometheusMetrics) RecordRejection(name string) {
	p.Rejections.WithLabelValues(name).Inc()
}

// RecordStateChange 记录状态变更。
func (p *CircuitBreakerPrometheusMetrics) RecordStateChange(name string, toState string) {
	p.StateChanges.WithLabelValues(name, toState).Inc()
}

// UpdateState 更新当前状态。
func (p *CircuitBreakerPrometheusMetrics) UpdateState(name string, state State) {
	p.State.WithLabelValues(name).Set(float64(state))
}

// UpdateFailureRate 更新失败率。
func (p *CircuitBreakerPrometheusMetrics) UpdateFailureRate(name string, rate float64) {
	p.FailureRate.WithLabelValues(name).Set(rate)
}

// RecordDuration 记录请求持续时间。
func (p *CircuitBreakerPrometheusMetrics) RecordDuration(name string, result string, duration float64) {
	p.RequestDuration.WithLabelValues(name, result).Observe(duration)
}

// UpdateFromCircuitBreakerMetrics 从 CircuitBreakerMetrics 更新 Prometheus 指标。
func (p *CircuitBreakerPrometheusMetrics) UpdateFromCircuitBreakerMetrics(name string, metrics *CircuitBreakerMetrics, state State) {
	// 更新状态
	p.UpdateState(name, state)

	// 更新失败率
	p.UpdateFailureRate(name, metrics.GetFailureRate())
}

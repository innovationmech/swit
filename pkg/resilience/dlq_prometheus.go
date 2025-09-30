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

// DLQPrometheusMetrics 包含 DLQ 的 Prometheus 指标。
type DLQPrometheusMetrics struct {
	TotalMessages     prometheus.Counter
	StoredMessages    prometheus.Counter
	DiscardedMessages prometheus.Counter
	RetriedMessages   prometheus.Counter
	RemovedMessages   prometheus.Counter
	ExpiredMessages   prometheus.Counter
	EvictedMessages   prometheus.Counter
	NotificationsSent prometheus.Counter
	QueueSize         prometheus.Gauge
}

// NewDLQPrometheusMetrics 创建并注册 DLQ Prometheus 指标。
func NewDLQPrometheusMetrics(namespace string, subsystem string) *DLQPrometheusMetrics {
	if namespace == "" {
		namespace = "swit"
	}
	if subsystem == "" {
		subsystem = "dlq"
	}

	return &DLQPrometheusMetrics{
		TotalMessages: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_total",
			Help:      "Total number of messages received by the DLQ",
		}),
		StoredMessages: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_stored_total",
			Help:      "Total number of messages stored in the DLQ",
		}),
		DiscardedMessages: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_discarded_total",
			Help:      "Total number of messages discarded",
		}),
		RetriedMessages: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_retried_total",
			Help:      "Total number of messages retried",
		}),
		RemovedMessages: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_removed_total",
			Help:      "Total number of messages manually removed from the DLQ",
		}),
		ExpiredMessages: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_expired_total",
			Help:      "Total number of messages expired due to TTL",
		}),
		EvictedMessages: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_evicted_total",
			Help:      "Total number of messages evicted due to queue size limit",
		}),
		NotificationsSent: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "notifications_sent_total",
			Help:      "Total number of notifications sent",
		}),
		QueueSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "queue_size",
			Help:      "Current number of messages in the DLQ",
		}),
	}
}

// UpdateFromDLQMetrics 从 DLQMetrics 更新 Prometheus 指标。
func (p *DLQPrometheusMetrics) UpdateFromDLQMetrics(metrics *DLQMetrics, queueSize int) {
	// Prometheus 的 Counter 是只增不减的，我们需要记录增量
	// 这里简单地设置为当前值（在生产环境中应该记录上次值并计算差值）
	p.TotalMessages.Add(float64(metrics.GetTotalMessages()))
	p.StoredMessages.Add(float64(metrics.GetMessagesStored()))
	p.DiscardedMessages.Add(float64(metrics.GetMessagesDiscarded()))
	p.RetriedMessages.Add(float64(metrics.GetMessagesRetried()))
	p.RemovedMessages.Add(float64(metrics.GetMessagesRemoved()))
	p.ExpiredMessages.Add(float64(metrics.GetMessagesExpired()))
	p.EvictedMessages.Add(float64(metrics.GetMessagesEvicted()))
	p.NotificationsSent.Add(float64(metrics.GetNotificationsSent()))
	p.QueueSize.Set(float64(queueSize))
}

// RecordMessage 记录收到一条消息。
func (p *DLQPrometheusMetrics) RecordMessage() {
	p.TotalMessages.Inc()
}

// RecordStore 记录存储一条消息。
func (p *DLQPrometheusMetrics) RecordStore(queueSize int) {
	p.StoredMessages.Inc()
	p.QueueSize.Set(float64(queueSize))
}

// RecordDiscard 记录丢弃一条消息。
func (p *DLQPrometheusMetrics) RecordDiscard() {
	p.DiscardedMessages.Inc()
}

// RecordRetry 记录重试一条消息。
func (p *DLQPrometheusMetrics) RecordRetry() {
	p.RetriedMessages.Inc()
}

// RecordRemoval 记录移除一条消息。
func (p *DLQPrometheusMetrics) RecordRemoval(queueSize int) {
	p.RemovedMessages.Inc()
	p.QueueSize.Set(float64(queueSize))
}

// RecordExpiration 记录过期一条消息。
func (p *DLQPrometheusMetrics) RecordExpiration(queueSize int) {
	p.ExpiredMessages.Inc()
	p.QueueSize.Set(float64(queueSize))
}

// RecordEviction 记录驱逐一条消息。
func (p *DLQPrometheusMetrics) RecordEviction() {
	p.EvictedMessages.Inc()
}

// RecordNotification 记录发送一条通知。
func (p *DLQPrometheusMetrics) RecordNotification() {
	p.NotificationsSent.Inc()
}

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

package integration

import (
	"sync/atomic"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/server"
)

// MessagingMetrics represents comprehensive metrics for the messaging system.
// It extends the existing performance monitoring framework with messaging-specific metrics.
type MessagingMetrics struct {
	// Message processing metrics
	MessagesPublished    int64 `json:"messages_published"`
	MessagesConsumed     int64 `json:"messages_consumed"`
	MessagesFailed       int64 `json:"messages_failed"`
	MessagesRetried      int64 `json:"messages_retried"`
	MessagesDeadLettered int64 `json:"messages_dead_lettered"`

	// Latency metrics
	AvgPublishLatency time.Duration `json:"avg_publish_latency"`
	MaxPublishLatency time.Duration `json:"max_publish_latency"`
	MinPublishLatency time.Duration `json:"min_publish_latency"`
	AvgConsumeLatency time.Duration `json:"avg_consume_latency"`
	MaxConsumeLatency time.Duration `json:"max_consume_latency"`
	MinConsumeLatency time.Duration `json:"min_consume_latency"`

	// Throughput metrics
	MessagesPerSecond float64 `json:"messages_per_second"`
	BytesPerSecond    float64 `json:"bytes_per_second"`

	// Broker health metrics
	ActiveBrokers            int64 `json:"active_brokers"`
	UnhealthyBrokers         int64 `json:"unhealthy_brokers"`
	BrokerConnectionFailures int64 `json:"broker_connection_failures"`

	// Queue depth metrics
	AvgQueueDepth float64 `json:"avg_queue_depth"`
	MaxQueueDepth int64   `json:"max_queue_depth"`

	// Consumer lag metrics
	AvgConsumerLag int64 `json:"avg_consumer_lag"`
	MaxConsumerLag int64 `json:"max_consumer_lag"`

	// Error rate metrics
	PublishErrorRate float64 `json:"publish_error_rate"`
	ConsumeErrorRate float64 `json:"consume_error_rate"`

	// Memory usage for messaging components
	MessagingMemoryUsage uint64 `json:"messaging_memory_usage"`

	// Last update timestamp
	LastUpdated time.Time `json:"last_updated"`
}

// MessagingPerformanceMonitor extends the server performance monitoring system
// with messaging-specific metrics and monitoring capabilities.
type MessagingPerformanceMonitor struct {
	serverMonitor *server.PerformanceMonitor
	metrics       *MessagingMetrics
	coordinator   messaging.MessagingCoordinator

	// Performance thresholds for messaging
	thresholds *MessagingPerformanceThresholds
}

// MessagingPerformanceThresholds defines performance thresholds for messaging components.
type MessagingPerformanceThresholds struct {
	MaxPublishLatency time.Duration `json:"max_publish_latency"`
	MaxConsumeLatency time.Duration `json:"max_consume_latency"`
	MaxErrorRate      float64       `json:"max_error_rate"`
	MaxQueueDepth     int64         `json:"max_queue_depth"`
	MaxConsumerLag    int64         `json:"max_consumer_lag"`
	MaxMemoryUsage    uint64        `json:"max_memory_usage"`
}

// DefaultMessagingPerformanceThresholds returns sensible defaults for messaging performance thresholds.
func DefaultMessagingPerformanceThresholds() *MessagingPerformanceThresholds {
	return &MessagingPerformanceThresholds{
		MaxPublishLatency: 100 * time.Millisecond,
		MaxConsumeLatency: 500 * time.Millisecond,
		MaxErrorRate:      0.05, // 5%
		MaxQueueDepth:     10000,
		MaxConsumerLag:    1000,
		MaxMemoryUsage:    100 * 1024 * 1024, // 100MB
	}
}

// NewMessagingPerformanceMonitor creates a new messaging performance monitor
// that integrates with the server's performance monitoring system.
func NewMessagingPerformanceMonitor(
	serverMonitor *server.PerformanceMonitor,
	coordinator messaging.MessagingCoordinator,
) *MessagingPerformanceMonitor {
	return &MessagingPerformanceMonitor{
		serverMonitor: serverMonitor,
		coordinator:   coordinator,
		metrics:       &MessagingMetrics{LastUpdated: time.Now()},
		thresholds:    DefaultMessagingPerformanceThresholds(),
	}
}

// GetMetrics returns a snapshot of the current messaging metrics.
func (mpm *MessagingPerformanceMonitor) GetMetrics() *MessagingMetrics {
	// Create a copy to avoid race conditions
	snapshot := *mpm.metrics
	return &snapshot
}

// SetThresholds updates the performance thresholds.
func (mpm *MessagingPerformanceMonitor) SetThresholds(thresholds *MessagingPerformanceThresholds) {
	mpm.thresholds = thresholds
}

// GetThresholds returns the current performance thresholds.
func (mpm *MessagingPerformanceMonitor) GetThresholds() *MessagingPerformanceThresholds {
	return mpm.thresholds
}

// RecordMessagePublished records metrics for a published message.
func (mpm *MessagingPerformanceMonitor) RecordMessagePublished(latency time.Duration, messageSize int64) {
	atomic.AddInt64(&mpm.metrics.MessagesPublished, 1)
	mpm.updateLatencyMetrics(&mpm.metrics.AvgPublishLatency, &mpm.metrics.MaxPublishLatency, &mpm.metrics.MinPublishLatency, latency)
	mpm.updateThroughputMetrics(messageSize)

	// Trigger performance hook
	mpm.serverMonitor.RecordEvent("message_published")
}

// RecordMessageConsumed records metrics for a consumed message.
func (mpm *MessagingPerformanceMonitor) RecordMessageConsumed(latency time.Duration, messageSize int64) {
	atomic.AddInt64(&mpm.metrics.MessagesConsumed, 1)
	mpm.updateLatencyMetrics(&mpm.metrics.AvgConsumeLatency, &mpm.metrics.MaxConsumeLatency, &mpm.metrics.MinConsumeLatency, latency)
	mpm.updateThroughputMetrics(messageSize)

	// Trigger performance hook
	mpm.serverMonitor.RecordEvent("message_consumed")
}

// RecordMessageFailed records metrics for a failed message.
func (mpm *MessagingPerformanceMonitor) RecordMessageFailed(isPublish bool) {
	atomic.AddInt64(&mpm.metrics.MessagesFailed, 1)

	// Update error rates
	mpm.updateErrorRates(isPublish)

	// Trigger performance hook
	mpm.serverMonitor.RecordEvent("message_failed")
}

// RecordMessageRetried records metrics for a retried message.
func (mpm *MessagingPerformanceMonitor) RecordMessageRetried() {
	atomic.AddInt64(&mpm.metrics.MessagesRetried, 1)
	mpm.serverMonitor.RecordEvent("message_retried")
}

// RecordMessageDeadLettered records metrics for a dead-lettered message.
func (mpm *MessagingPerformanceMonitor) RecordMessageDeadLettered() {
	atomic.AddInt64(&mpm.metrics.MessagesDeadLettered, 1)
	mpm.serverMonitor.RecordEvent("message_dead_lettered")
}

// RecordBrokerConnectionFailure records broker connection failures.
func (mpm *MessagingPerformanceMonitor) RecordBrokerConnectionFailure() {
	atomic.AddInt64(&mpm.metrics.BrokerConnectionFailures, 1)
	mpm.serverMonitor.RecordEvent("broker_connection_failure")
}

// UpdateBrokerHealth updates broker health metrics.
func (mpm *MessagingPerformanceMonitor) UpdateBrokerHealth(active, unhealthy int64) {
	atomic.StoreInt64(&mpm.metrics.ActiveBrokers, active)
	atomic.StoreInt64(&mpm.metrics.UnhealthyBrokers, unhealthy)
}

// UpdateQueueMetrics updates queue depth metrics.
func (mpm *MessagingPerformanceMonitor) UpdateQueueMetrics(avgDepth float64, maxDepth int64) {
	mpm.metrics.AvgQueueDepth = avgDepth
	atomic.StoreInt64(&mpm.metrics.MaxQueueDepth, maxDepth)
}

// UpdateConsumerLag updates consumer lag metrics.
func (mpm *MessagingPerformanceMonitor) UpdateConsumerLag(avgLag, maxLag int64) {
	atomic.StoreInt64(&mpm.metrics.AvgConsumerLag, avgLag)
	atomic.StoreInt64(&mpm.metrics.MaxConsumerLag, maxLag)
}

// UpdateMemoryUsage updates messaging memory usage.
func (mpm *MessagingPerformanceMonitor) UpdateMemoryUsage(usage uint64) {
	atomic.StoreUint64(&mpm.metrics.MessagingMemoryUsage, usage)
}

// CheckThresholds checks if any messaging performance thresholds are exceeded.
func (mpm *MessagingPerformanceMonitor) CheckThresholds() []string {
	var violations []string

	if mpm.metrics.AvgPublishLatency > mpm.thresholds.MaxPublishLatency {
		violations = append(violations,
			"average publish latency "+mpm.metrics.AvgPublishLatency.String()+
				" exceeds threshold "+mpm.thresholds.MaxPublishLatency.String())
	}

	if mpm.metrics.AvgConsumeLatency > mpm.thresholds.MaxConsumeLatency {
		violations = append(violations,
			"average consume latency "+mpm.metrics.AvgConsumeLatency.String()+
				" exceeds threshold "+mpm.thresholds.MaxConsumeLatency.String())
	}

	if mpm.metrics.PublishErrorRate > mpm.thresholds.MaxErrorRate {
		violations = append(violations,
			"publish error rate exceeds threshold")
	}

	if mpm.metrics.ConsumeErrorRate > mpm.thresholds.MaxErrorRate {
		violations = append(violations,
			"consume error rate exceeds threshold")
	}

	if mpm.metrics.MaxQueueDepth > mpm.thresholds.MaxQueueDepth {
		violations = append(violations,
			"queue depth exceeds threshold")
	}

	if mpm.metrics.MaxConsumerLag > mpm.thresholds.MaxConsumerLag {
		violations = append(violations,
			"consumer lag exceeds threshold")
	}

	if mpm.metrics.MessagingMemoryUsage > mpm.thresholds.MaxMemoryUsage {
		violations = append(violations,
			"messaging memory usage exceeds threshold")
	}

	return violations
}

// CollectMetrics collects current metrics from the messaging coordinator and brokers.
func (mpm *MessagingPerformanceMonitor) CollectMetrics() {
	if mpm.coordinator == nil {
		return
	}

	coordMetrics := mpm.coordinator.GetMetrics()
	if coordMetrics == nil {
		return
	}

	// Update broker health metrics
	activeBrokers := int64(coordMetrics.BrokerCount)
	unhealthyBrokers := int64(0)

	// Calculate aggregate metrics from broker metrics
	var totalPublished, totalConsumed, totalErrors int64
	var totalMemoryUsage uint64
	var maxQueueDepth int64
	var avgLatency time.Duration

	for _, brokerMetrics := range coordMetrics.BrokerMetrics {
		if brokerMetrics != nil {
			totalPublished += brokerMetrics.MessagesPublished
			totalConsumed += brokerMetrics.MessagesConsumed
			totalErrors += brokerMetrics.PublishErrors + brokerMetrics.ConsumeErrors
			totalMemoryUsage += uint64(brokerMetrics.MemoryUsage)

			if brokerMetrics.AvgPublishLatency > avgLatency {
				avgLatency = brokerMetrics.AvgPublishLatency
			}

			// Check if broker is unhealthy based on connection status
			if brokerMetrics.ConnectionStatus != "connected" {
				unhealthyBrokers++
			}
		}
	}

	// Update metrics
	mpm.UpdateBrokerHealth(activeBrokers, unhealthyBrokers)
	mpm.UpdateMemoryUsage(totalMemoryUsage)
	mpm.UpdateQueueMetrics(0, maxQueueDepth) // Queue depth would come from broker-specific metrics

	// Update error rates
	if totalPublished > 0 {
		mpm.metrics.PublishErrorRate = float64(totalErrors) / float64(totalPublished)
	}
	if totalConsumed > 0 {
		mpm.metrics.ConsumeErrorRate = float64(totalErrors) / float64(totalConsumed)
	}

	mpm.metrics.LastUpdated = time.Now()
}

// updateLatencyMetrics updates average, max, and min latency metrics.
func (mpm *MessagingPerformanceMonitor) updateLatencyMetrics(avg, max, min *time.Duration, newLatency time.Duration) {
	// Simple moving average (in a real implementation, you might want a more sophisticated approach)
	if *avg == 0 {
		*avg = newLatency
	} else {
		*avg = (*avg + newLatency) / 2
	}

	if newLatency > *max {
		*max = newLatency
	}

	if *min == 0 || newLatency < *min {
		*min = newLatency
	}
}

// updateThroughputMetrics updates throughput metrics.
func (mpm *MessagingPerformanceMonitor) updateThroughputMetrics(messageSize int64) {
	// Update throughput (simplified calculation)
	now := time.Now()
	if !mpm.metrics.LastUpdated.IsZero() {
		duration := now.Sub(mpm.metrics.LastUpdated)
		if duration > 0 {
			mpm.metrics.MessagesPerSecond = 1.0 / duration.Seconds()
			mpm.metrics.BytesPerSecond = float64(messageSize) / duration.Seconds()
		}
	}
}

// updateErrorRates updates error rate metrics.
func (mpm *MessagingPerformanceMonitor) updateErrorRates(isPublish bool) {
	totalMessages := atomic.LoadInt64(&mpm.metrics.MessagesPublished) + atomic.LoadInt64(&mpm.metrics.MessagesConsumed)
	totalFailures := atomic.LoadInt64(&mpm.metrics.MessagesFailed)

	if totalMessages > 0 {
		overallErrorRate := float64(totalFailures) / float64(totalMessages)

		if isPublish {
			mpm.metrics.PublishErrorRate = overallErrorRate
		} else {
			mpm.metrics.ConsumeErrorRate = overallErrorRate
		}
	}
}

// MessagingPerformanceHook creates a performance hook for messaging events.
func MessagingPerformanceHook(monitor *MessagingPerformanceMonitor) server.PerformanceHook {
	return func(event string, metrics *server.PerformanceMetrics) {
		// Collect messaging metrics when performance events occur
		monitor.CollectMetrics()

		// Check for threshold violations
		violations := monitor.CheckThresholds()
		if len(violations) > 0 {
			// In a real implementation, you might want to log warnings or trigger alerts
			for _, violation := range violations {
				_ = violation // Suppress unused variable warning
				// logger.Logger.Warn("Messaging performance threshold violation", zap.String("violation", violation))
			}
		}
	}
}

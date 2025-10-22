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

// Package monitoring provides metrics collection and monitoring capabilities for Saga execution.
// It includes support for Prometheus metrics, thread-safe operations, and extensible interfaces
// for custom metrics implementations.
package monitoring

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsCollector defines the interface for collecting Saga-related metrics.
// Implementations must be thread-safe and support concurrent access.
type MetricsCollector interface {
	// RecordSagaStarted increments the count of started Sagas.
	RecordSagaStarted(sagaID string)

	// RecordSagaCompleted increments the count of completed Sagas and records the duration.
	RecordSagaCompleted(sagaID string, duration time.Duration)

	// RecordSagaFailed increments the count of failed Sagas with the given reason.
	RecordSagaFailed(sagaID string, reason string)

	// GetActiveSagasCount returns the current number of active Sagas.
	GetActiveSagasCount() int64

	// GetMetrics returns a snapshot of all collected metrics.
	GetMetrics() *Metrics

	// Reset clears all collected metrics (for testing).
	Reset()
}

// Metrics contains a snapshot of collected Saga metrics.
type Metrics struct {
	// Total number of Sagas started
	SagasStarted int64

	// Total number of Sagas completed successfully
	SagasCompleted int64

	// Total number of Sagas that failed
	SagasFailed int64

	// Current number of active Sagas
	ActiveSagas int64

	// Total duration of all completed Sagas (in seconds)
	TotalDuration float64

	// Average duration per Saga (in seconds)
	AvgDuration float64

	// Failure reasons with their counts
	FailureReasons map[string]int64

	// Timestamp when metrics were collected
	Timestamp time.Time
}

// SagaMetricsCollector is the base implementation of MetricsCollector
// using Prometheus metrics for production use.
type SagaMetricsCollector struct {
	// Prometheus registry for metric registration
	registry prometheus.Registerer

	// Counter for total Sagas started
	sagaStartedCounter prometheus.Counter

	// Counter for total Sagas completed
	sagaCompletedCounter prometheus.Counter

	// Counter for total Sagas failed (with reason label)
	sagaFailedCounter *prometheus.CounterVec

	// Histogram for Saga execution duration
	sagaDurationHistogram prometheus.Histogram

	// Gauge for active Sagas count
	activeSagasGauge prometheus.Gauge

	// Mutex for thread-safe operations on internal state
	mu sync.RWMutex

	// Internal counters for non-Prometheus metrics retrieval
	internalStarted       int64
	internalCompleted     int64
	internalFailed        int64
	internalActive        int64
	internalTotalDuration float64 // Running sum of all durations to avoid unbounded slice storage
	failureReasons        map[string]int64
}

// Config contains configuration options for SagaMetricsCollector.
type Config struct {
	// Namespace for Prometheus metrics (default: "saga")
	Namespace string

	// Subsystem for Prometheus metrics (default: "monitoring")
	Subsystem string

	// Registry for Prometheus metrics. If nil, uses prometheus.DefaultRegisterer.
	Registry prometheus.Registerer

	// DurationBuckets for histogram (default: [0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30, 60])
	DurationBuckets []float64
}

// DefaultConfig returns a default configuration for SagaMetricsCollector.
func DefaultConfig() *Config {
	return &Config{
		Namespace: "saga",
		Subsystem: "monitoring",
		Registry:  prometheus.DefaultRegisterer,
		DurationBuckets: []float64{
			0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0,
		},
	}
}

// NewSagaMetricsCollector creates a new SagaMetricsCollector with the given configuration.
// It initializes all Prometheus metrics and registers them with the provided registry.
//
// Parameters:
//   - config: Configuration for the metrics collector. If nil, default config is used.
//
// Returns:
//   - A configured SagaMetricsCollector ready to collect metrics.
//   - An error if metric registration fails.
//
// Example:
//
//	collector, err := NewSagaMetricsCollector(nil)
//	if err != nil {
//	    return err
//	}
//	collector.RecordSagaStarted("saga-123")
func NewSagaMetricsCollector(config *Config) (*SagaMetricsCollector, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Apply defaults
	if config.Namespace == "" {
		config.Namespace = "saga"
	}
	if config.Subsystem == "" {
		config.Subsystem = "monitoring"
	}
	if config.Registry == nil {
		config.Registry = prometheus.DefaultRegisterer
	}
	if config.DurationBuckets == nil {
		config.DurationBuckets = []float64{
			0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0,
		}
	}

	collector := &SagaMetricsCollector{
		registry:       config.Registry,
		failureReasons: make(map[string]int64),
	}

	// Initialize Saga started counter
	collector.sagaStartedCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "saga_started_total",
			Help:      "Total number of Sagas started",
		},
	)

	// Initialize Saga completed counter
	collector.sagaCompletedCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "saga_completed_total",
			Help:      "Total number of Sagas completed successfully",
		},
	)

	// Initialize Saga failed counter with reason label
	collector.sagaFailedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "saga_failed_total",
			Help:      "Total number of Sagas that failed",
		},
		[]string{"reason"},
	)

	// Initialize Saga duration histogram
	collector.sagaDurationHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "saga_duration_seconds",
			Help:      "Duration of Saga execution in seconds",
			Buckets:   config.DurationBuckets,
		},
	)

	// Initialize active Sagas gauge
	collector.activeSagasGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "active_sagas",
			Help:      "Current number of active Sagas",
		},
	)

	// Register all metrics
	if err := config.Registry.Register(collector.sagaStartedCounter); err != nil {
		return nil, err
	}
	if err := config.Registry.Register(collector.sagaCompletedCounter); err != nil {
		return nil, err
	}
	if err := config.Registry.Register(collector.sagaFailedCounter); err != nil {
		return nil, err
	}
	if err := config.Registry.Register(collector.sagaDurationHistogram); err != nil {
		return nil, err
	}
	if err := config.Registry.Register(collector.activeSagasGauge); err != nil {
		return nil, err
	}

	return collector, nil
}

// RecordSagaStarted increments the count of started Sagas and the active Sagas gauge.
// This method is thread-safe and can be called concurrently.
func (smc *SagaMetricsCollector) RecordSagaStarted(sagaID string) {
	smc.sagaStartedCounter.Inc()
	smc.activeSagasGauge.Inc()

	smc.mu.Lock()
	smc.internalStarted++
	smc.internalActive++
	smc.mu.Unlock()
}

// RecordSagaCompleted increments the count of completed Sagas, decrements the active Sagas gauge,
// and records the duration in the histogram.
// This method is thread-safe and can be called concurrently.
func (smc *SagaMetricsCollector) RecordSagaCompleted(sagaID string, duration time.Duration) {
	smc.sagaCompletedCounter.Inc()
	smc.activeSagasGauge.Dec()
	smc.sagaDurationHistogram.Observe(duration.Seconds())

	durationSeconds := duration.Seconds()
	smc.mu.Lock()
	smc.internalCompleted++
	smc.internalActive--
	smc.internalTotalDuration += durationSeconds
	smc.mu.Unlock()
}

// RecordSagaFailed increments the count of failed Sagas with the given reason
// and decrements the active Sagas gauge.
// This method is thread-safe and can be called concurrently.
func (smc *SagaMetricsCollector) RecordSagaFailed(sagaID string, reason string) {
	smc.sagaFailedCounter.WithLabelValues(reason).Inc()
	smc.activeSagasGauge.Dec()

	smc.mu.Lock()
	smc.internalFailed++
	smc.internalActive--
	smc.failureReasons[reason]++
	smc.mu.Unlock()
}

// GetActiveSagasCount returns the current number of active Sagas.
// This method is thread-safe and can be called concurrently.
func (smc *SagaMetricsCollector) GetActiveSagasCount() int64 {
	smc.mu.RLock()
	defer smc.mu.RUnlock()
	return smc.internalActive
}

// GetMetrics returns a snapshot of all collected metrics.
// This method is thread-safe and can be called concurrently.
func (smc *SagaMetricsCollector) GetMetrics() *Metrics {
	smc.mu.RLock()
	defer smc.mu.RUnlock()

	// Calculate average duration using running sum to avoid O(n) iteration
	var avgDuration float64
	var totalDuration float64
	if smc.internalCompleted > 0 {
		totalDuration = smc.internalTotalDuration
		avgDuration = totalDuration / float64(smc.internalCompleted)
	}

	// Copy failure reasons map
	failureReasons := make(map[string]int64, len(smc.failureReasons))
	for reason, count := range smc.failureReasons {
		failureReasons[reason] = count
	}

	return &Metrics{
		SagasStarted:   smc.internalStarted,
		SagasCompleted: smc.internalCompleted,
		SagasFailed:    smc.internalFailed,
		ActiveSagas:    smc.internalActive,
		TotalDuration:  totalDuration,
		AvgDuration:    avgDuration,
		FailureReasons: failureReasons,
		Timestamp:      time.Now(),
	}
}

// Reset clears all collected metrics. This is primarily useful for testing.
// Note: This only resets internal counters, not the registered Prometheus metrics.
// This method is thread-safe and can be called concurrently.
func (smc *SagaMetricsCollector) Reset() {
	smc.mu.Lock()
	defer smc.mu.Unlock()

	smc.internalStarted = 0
	smc.internalCompleted = 0
	smc.internalFailed = 0
	smc.internalActive = 0
	smc.internalTotalDuration = 0
	smc.failureReasons = make(map[string]int64)
}

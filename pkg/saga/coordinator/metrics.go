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

package coordinator

import (
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetricsCollector implements MetricsCollector using Prometheus metrics.
// It collects and exposes metrics for Saga coordinator operations, including
// Saga lifecycle events, step execution, compensation, and timing information.
type PrometheusMetricsCollector struct {
	// Saga lifecycle metrics
	sagaStartedTotal   *prometheus.CounterVec
	sagaCompletedTotal *prometheus.CounterVec
	sagaFailedTotal    *prometheus.CounterVec
	sagaCancelledTotal *prometheus.CounterVec
	sagaTimedOutTotal  *prometheus.CounterVec
	sagaDuration       *prometheus.HistogramVec

	// Step execution metrics
	stepExecutedTotal *prometheus.CounterVec
	stepRetriedTotal  *prometheus.CounterVec
	stepDuration      *prometheus.HistogramVec

	// Compensation metrics
	compensationExecutedTotal *prometheus.CounterVec
	compensationDuration      *prometheus.HistogramVec

	// Prometheus registry
	registry *prometheus.Registry
}

// PrometheusMetricsConfig contains configuration for Prometheus metrics.
type PrometheusMetricsConfig struct {
	// Namespace is the Prometheus namespace for all metrics (default: "saga")
	Namespace string

	// Subsystem is the Prometheus subsystem for all metrics (default: "coordinator")
	Subsystem string

	// Registry is the Prometheus registry to use. If nil, a new registry is created.
	Registry *prometheus.Registry

	// DurationBuckets defines the buckets for duration histograms.
	// If nil, default buckets are used: [0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30, 60]
	DurationBuckets []float64
}

// DefaultPrometheusMetricsConfig returns a default configuration for Prometheus metrics.
func DefaultPrometheusMetricsConfig() *PrometheusMetricsConfig {
	return &PrometheusMetricsConfig{
		Namespace: "saga",
		Subsystem: "coordinator",
		Registry:  prometheus.NewRegistry(),
		DurationBuckets: []float64{
			0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0,
		},
	}
}

// NewPrometheusMetricsCollector creates a new Prometheus-based metrics collector.
// It initializes all metrics and registers them with the provided or default registry.
//
// Parameters:
//   - config: Configuration for the metrics collector. If nil, default config is used.
//
// Returns:
//   - A configured PrometheusMetricsCollector ready to collect metrics.
//   - An error if metric registration fails.
//
// Example:
//
//	collector, err := NewPrometheusMetricsCollector(nil)
//	if err != nil {
//	    return err
//	}
//	config := &OrchestratorConfig{
//	    MetricsCollector: collector,
//	    // ... other config
//	}
func NewPrometheusMetricsCollector(config *PrometheusMetricsConfig) (*PrometheusMetricsCollector, error) {
	if config == nil {
		config = DefaultPrometheusMetricsConfig()
	}

	if config.Registry == nil {
		config.Registry = prometheus.NewRegistry()
	}

	if config.Namespace == "" {
		config.Namespace = "saga"
	}

	if config.Subsystem == "" {
		config.Subsystem = "coordinator"
	}

	if config.DurationBuckets == nil {
		config.DurationBuckets = []float64{
			0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0,
		}
	}

	collector := &PrometheusMetricsCollector{
		registry: config.Registry,
	}

	// Initialize Saga lifecycle metrics
	collector.sagaStartedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "saga_started_total",
			Help:      "Total number of Sagas started",
		},
		[]string{"definition_id"},
	)

	collector.sagaCompletedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "saga_completed_total",
			Help:      "Total number of Sagas completed successfully",
		},
		[]string{"definition_id"},
	)

	collector.sagaFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "saga_failed_total",
			Help:      "Total number of Sagas that failed",
		},
		[]string{"definition_id", "error_type"},
	)

	collector.sagaCancelledTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "saga_cancelled_total",
			Help:      "Total number of Sagas that were cancelled",
		},
		[]string{"definition_id"},
	)

	collector.sagaTimedOutTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "saga_timed_out_total",
			Help:      "Total number of Sagas that timed out",
		},
		[]string{"definition_id"},
	)

	collector.sagaDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "saga_duration_seconds",
			Help:      "Duration of Saga execution in seconds",
			Buckets:   config.DurationBuckets,
		},
		[]string{"definition_id", "status"},
	)

	// Initialize step execution metrics
	collector.stepExecutedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "step_executed_total",
			Help:      "Total number of steps executed",
		},
		[]string{"definition_id", "step_id", "success"},
	)

	collector.stepRetriedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "step_retried_total",
			Help:      "Total number of step retry attempts",
		},
		[]string{"definition_id", "step_id", "attempt"},
	)

	collector.stepDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "step_duration_seconds",
			Help:      "Duration of step execution in seconds",
			Buckets:   config.DurationBuckets,
		},
		[]string{"definition_id", "step_id", "success"},
	)

	// Initialize compensation metrics
	collector.compensationExecutedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "compensation_executed_total",
			Help:      "Total number of compensation operations executed",
		},
		[]string{"definition_id", "step_id", "success"},
	)

	collector.compensationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "compensation_duration_seconds",
			Help:      "Duration of compensation execution in seconds",
			Buckets:   config.DurationBuckets,
		},
		[]string{"definition_id", "step_id", "success"},
	)

	// Register all metrics with the registry
	metrics := []prometheus.Collector{
		collector.sagaStartedTotal,
		collector.sagaCompletedTotal,
		collector.sagaFailedTotal,
		collector.sagaCancelledTotal,
		collector.sagaTimedOutTotal,
		collector.sagaDuration,
		collector.stepExecutedTotal,
		collector.stepRetriedTotal,
		collector.stepDuration,
		collector.compensationExecutedTotal,
		collector.compensationDuration,
	}

	for _, metric := range metrics {
		if err := config.Registry.Register(metric); err != nil {
			return nil, err
		}
	}

	return collector, nil
}

// RecordSagaStarted increments the count of started Sagas.
func (pmc *PrometheusMetricsCollector) RecordSagaStarted(definitionID string) {
	pmc.sagaStartedTotal.WithLabelValues(definitionID).Inc()
}

// RecordSagaCompleted increments the count of completed Sagas and records the duration.
func (pmc *PrometheusMetricsCollector) RecordSagaCompleted(definitionID string, duration time.Duration) {
	pmc.sagaCompletedTotal.WithLabelValues(definitionID).Inc()
	pmc.sagaDuration.WithLabelValues(definitionID, "completed").Observe(duration.Seconds())
}

// RecordSagaFailed increments the count of failed Sagas and records the duration.
func (pmc *PrometheusMetricsCollector) RecordSagaFailed(
	definitionID string,
	errorType saga.ErrorType,
	duration time.Duration,
) {
	pmc.sagaFailedTotal.WithLabelValues(definitionID, string(errorType)).Inc()
	pmc.sagaDuration.WithLabelValues(definitionID, "failed").Observe(duration.Seconds())
}

// RecordSagaCancelled increments the count of cancelled Sagas and records the duration.
func (pmc *PrometheusMetricsCollector) RecordSagaCancelled(definitionID string, duration time.Duration) {
	pmc.sagaCancelledTotal.WithLabelValues(definitionID).Inc()
	pmc.sagaDuration.WithLabelValues(definitionID, "cancelled").Observe(duration.Seconds())
}

// RecordSagaTimedOut increments the count of timed out Sagas and records the duration.
func (pmc *PrometheusMetricsCollector) RecordSagaTimedOut(definitionID string, duration time.Duration) {
	pmc.sagaTimedOutTotal.WithLabelValues(definitionID).Inc()
	pmc.sagaDuration.WithLabelValues(definitionID, "timed_out").Observe(duration.Seconds())
}

// RecordStepExecuted increments the count of executed steps and records the duration.
func (pmc *PrometheusMetricsCollector) RecordStepExecuted(
	definitionID, stepID string,
	success bool,
	duration time.Duration,
) {
	successLabel := "false"
	if success {
		successLabel = "true"
	}
	pmc.stepExecutedTotal.WithLabelValues(definitionID, stepID, successLabel).Inc()
	pmc.stepDuration.WithLabelValues(definitionID, stepID, successLabel).Observe(duration.Seconds())
}

// RecordStepRetried increments the count of step retries.
func (pmc *PrometheusMetricsCollector) RecordStepRetried(definitionID, stepID string, attempt int) {
	attemptLabel := "0"
	if attempt > 0 {
		// Limit attempt labels to reduce cardinality (1, 2, 3, 4+)
		if attempt <= 3 {
			attemptLabel = string(rune('0' + attempt))
		} else {
			attemptLabel = "4+"
		}
	}
	pmc.stepRetriedTotal.WithLabelValues(definitionID, stepID, attemptLabel).Inc()
}

// RecordCompensationExecuted increments the count of compensation executions and records the duration.
func (pmc *PrometheusMetricsCollector) RecordCompensationExecuted(
	definitionID, stepID string,
	success bool,
	duration time.Duration,
) {
	successLabel := "false"
	if success {
		successLabel = "true"
	}
	pmc.compensationExecutedTotal.WithLabelValues(definitionID, stepID, successLabel).Inc()
	pmc.compensationDuration.WithLabelValues(definitionID, stepID, successLabel).Observe(duration.Seconds())
}

// GetRegistry returns the Prometheus registry used by this collector.
// This can be used to expose the metrics via an HTTP handler.
//
// Example:
//
//	http.Handle("/metrics", promhttp.HandlerFor(
//	    collector.GetRegistry(),
//	    promhttp.HandlerOpts{},
//	))
func (pmc *PrometheusMetricsCollector) GetRegistry() *prometheus.Registry {
	return pmc.registry
}

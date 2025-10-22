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

// Package monitoring provides Prometheus metrics integration for Saga execution.
package monitoring

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsExporter provides Prometheus metrics export functionality.
// It wraps a Prometheus registry and provides HTTP handlers for metrics exposition.
type MetricsExporter struct {
	// registry is the Prometheus registry used to collect metrics
	registry prometheus.Registerer

	// gatherer is used to gather metrics for export
	gatherer prometheus.Gatherer
}

// NewMetricsExporter creates a new MetricsExporter with the given registry.
// If registry is nil, it uses prometheus.DefaultRegisterer.
//
// Parameters:
//   - registry: The Prometheus registerer to use. If nil, uses the default registerer.
//
// Returns:
//   - A configured MetricsExporter ready to export metrics.
//
// Example:
//
//	exporter := NewMetricsExporter(prometheus.NewRegistry())
//	http.Handle("/metrics", exporter.HTTPHandler())
func NewMetricsExporter(registry prometheus.Registerer) *MetricsExporter {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	// Try to get the gatherer from the registry
	var gatherer prometheus.Gatherer
	if reg, ok := registry.(*prometheus.Registry); ok {
		gatherer = reg
	} else {
		// Fallback to default gatherer if registry doesn't support gathering
		gatherer = prometheus.DefaultGatherer
	}

	return &MetricsExporter{
		registry: registry,
		gatherer: gatherer,
	}
}

// HTTPHandler returns an HTTP handler that serves Prometheus metrics.
// This handler can be used directly with http.Handle or gin.WrapH.
//
// Returns:
//   - An http.Handler that serves metrics in Prometheus text format.
//
// Example:
//
//	exporter := NewMetricsExporter(nil)
//	http.Handle("/metrics", exporter.HTTPHandler())
//	http.ListenAndServe(":8080", nil)
func (me *MetricsExporter) HTTPHandler() http.Handler {
	return promhttp.HandlerFor(me.gatherer, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
		ErrorHandling:     promhttp.ContinueOnError,
	})
}

// Registry returns the underlying Prometheus registry.
// This can be used to register additional metrics or for advanced usage.
//
// Returns:
//   - The Prometheus registerer used by this exporter.
func (me *MetricsExporter) Registry() prometheus.Registerer {
	return me.registry
}

// MetricsLabels defines labels that can be attached to Saga metrics.
// Labels provide additional dimensions for filtering and aggregating metrics.
type MetricsLabels struct {
	// SagaType is the type of the Saga (e.g., "order", "payment", "inventory")
	SagaType string

	// Status is the current status of the Saga (e.g., "started", "completed", "failed", "compensated")
	Status string

	// Additional custom labels can be added via the Extra map
	Extra map[string]string
}

// ToSlice converts MetricsLabels to a slice of label values.
// The order matches the label keys: [saga_type, status, ...extra keys in sorted order].
//
// Returns:
//   - A slice of label values for use with Prometheus labeled metrics.
func (ml *MetricsLabels) ToSlice() []string {
	labels := []string{ml.SagaType, ml.Status}
	// Note: For extra labels, we would need to maintain consistent ordering
	// This is a simplified version - in production, you'd want to ensure
	// label keys are sorted consistently
	return labels
}

// LabeledMetricsCollector extends MetricsCollector with support for custom labels.
// This allows for more granular metric collection based on Saga characteristics.
type LabeledMetricsCollector interface {
	MetricsCollector

	// RecordSagaStartedWithLabels records a Saga start with custom labels
	RecordSagaStartedWithLabels(sagaID string, labels *MetricsLabels)

	// RecordSagaCompletedWithLabels records a Saga completion with custom labels
	RecordSagaCompletedWithLabels(sagaID string, duration float64, labels *MetricsLabels)

	// RecordSagaFailedWithLabels records a Saga failure with custom labels
	RecordSagaFailedWithLabels(sagaID string, reason string, labels *MetricsLabels)
}

// PrometheusMetrics encapsulates all Prometheus metrics for Saga monitoring.
// This structure can be used to create metrics with custom configurations.
type PrometheusMetrics struct {
	// SagaStartedTotal counts total Sagas started
	SagaStartedTotal *prometheus.CounterVec

	// SagaCompletedTotal counts total Sagas completed
	SagaCompletedTotal *prometheus.CounterVec

	// SagaFailedTotal counts total Sagas failed
	SagaFailedTotal *prometheus.CounterVec

	// SagaDurationSeconds records Saga execution duration
	SagaDurationSeconds *prometheus.HistogramVec

	// ActiveSagas tracks current active Sagas
	ActiveSagas *prometheus.GaugeVec
}

// NewPrometheusMetrics creates a new PrometheusMetrics with default configurations.
// All metrics include saga_type and status labels for better observability.
//
// Parameters:
//   - namespace: The namespace for metrics (e.g., "saga")
//   - subsystem: The subsystem for metrics (e.g., "monitoring")
//
// Returns:
//   - A PrometheusMetrics instance with all metrics initialized.
//
// Example:
//
//	metrics := NewPrometheusMetrics("saga", "monitoring")
//	metrics.SagaStartedTotal.WithLabelValues("order", "started").Inc()
func NewPrometheusMetrics(namespace, subsystem string) *PrometheusMetrics {
	return &PrometheusMetrics{
		SagaStartedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "saga_started_labeled_total",
				Help:      "Total number of Sagas started with labels",
			},
			[]string{"saga_type", "status"},
		),
		SagaCompletedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "saga_completed_labeled_total",
				Help:      "Total number of Sagas completed successfully with labels",
			},
			[]string{"saga_type", "status"},
		),
		SagaFailedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "saga_failed_labeled_total",
				Help:      "Total number of Sagas that failed with labels",
			},
			[]string{"saga_type", "reason"},
		),
		SagaDurationSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "saga_duration_labeled_seconds",
				Help:      "Duration of Saga execution in seconds with labels",
				Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0},
			},
			[]string{"saga_type", "status"},
		),
		ActiveSagas: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "active_sagas_labeled",
				Help:      "Current number of active Sagas with labels",
			},
			[]string{"saga_type"},
		),
	}
}

// Register registers all metrics with the given Prometheus registry.
//
// Parameters:
//   - registry: The Prometheus registerer to register metrics with.
//
// Returns:
//   - An error if any metric registration fails.
//
// Example:
//
//	metrics := NewPrometheusMetrics("saga", "monitoring")
//	err := metrics.Register(prometheus.DefaultRegisterer)
func (pm *PrometheusMetrics) Register(registry prometheus.Registerer) error {
	if err := registry.Register(pm.SagaStartedTotal); err != nil {
		return err
	}
	if err := registry.Register(pm.SagaCompletedTotal); err != nil {
		return err
	}
	if err := registry.Register(pm.SagaFailedTotal); err != nil {
		return err
	}
	if err := registry.Register(pm.SagaDurationSeconds); err != nil {
		return err
	}
	if err := registry.Register(pm.ActiveSagas); err != nil {
		return err
	}
	return nil
}

// Unregister removes all metrics from the given Prometheus registry.
// This is useful for testing or when recreating metrics.
//
// Parameters:
//   - registry: The Prometheus registerer to unregister metrics from.
//
// Returns:
//   - true if all metrics were successfully unregistered, false otherwise.
func (pm *PrometheusMetrics) Unregister(registry prometheus.Registerer) bool {
	success := true
	success = registry.Unregister(pm.SagaStartedTotal) && success
	success = registry.Unregister(pm.SagaCompletedTotal) && success
	success = registry.Unregister(pm.SagaFailedTotal) && success
	success = registry.Unregister(pm.SagaDurationSeconds) && success
	success = registry.Unregister(pm.ActiveSagas) && success
	return success
}

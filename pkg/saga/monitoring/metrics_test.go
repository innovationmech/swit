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

package monitoring

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMetricsExporter tests the creation of MetricsExporter.
func TestNewMetricsExporter(t *testing.T) {
	tests := []struct {
		name     string
		registry prometheus.Registerer
		wantNil  bool
	}{
		{
			name:     "with nil registry",
			registry: nil,
			wantNil:  false,
		},
		{
			name:     "with custom registry",
			registry: prometheus.NewRegistry(),
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := NewMetricsExporter(tt.registry)
			if tt.wantNil {
				assert.Nil(t, exporter)
			} else {
				assert.NotNil(t, exporter)
				assert.NotNil(t, exporter.registry)
				assert.NotNil(t, exporter.gatherer)
			}
		})
	}
}

// TestMetricsExporter_HTTPHandler tests the HTTP handler for metrics export.
func TestMetricsExporter_HTTPHandler(t *testing.T) {
	registry := prometheus.NewRegistry()
	exporter := NewMetricsExporter(registry)

	// Create a test counter and register it
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "A test counter",
	})
	err := registry.Register(counter)
	require.NoError(t, err)

	// Increment the counter
	counter.Add(42)

	// Create a test HTTP request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Get the handler and serve the request
	handler := exporter.HTTPHandler()
	handler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")

	// Check that the metric appears in the output
	body := w.Body.String()
	assert.Contains(t, body, "test_counter")
	assert.Contains(t, body, "42")
}

// TestMetricsExporter_Registry tests the Registry method.
func TestMetricsExporter_Registry(t *testing.T) {
	registry := prometheus.NewRegistry()
	exporter := NewMetricsExporter(registry)

	assert.Equal(t, registry, exporter.Registry())
}

// TestMetricsLabels_ToSlice tests the ToSlice method.
func TestMetricsLabels_ToSlice(t *testing.T) {
	labels := &MetricsLabels{
		SagaType: "order",
		Status:   "completed",
		Extra:    map[string]string{"region": "us-east-1"},
	}

	slice := labels.ToSlice()
	assert.Equal(t, []string{"order", "completed"}, slice)
}

// TestNewPrometheusMetrics tests the creation of PrometheusMetrics.
func TestNewPrometheusMetrics(t *testing.T) {
	metrics := NewPrometheusMetrics("saga", "monitoring")

	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.SagaStartedTotal)
	assert.NotNil(t, metrics.SagaCompletedTotal)
	assert.NotNil(t, metrics.SagaFailedTotal)
	assert.NotNil(t, metrics.SagaDurationSeconds)
	assert.NotNil(t, metrics.ActiveSagas)
}

// TestPrometheusMetrics_Register tests metric registration.
func TestPrometheusMetrics_Register(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewPrometheusMetrics("saga", "monitoring")

	err := metrics.Register(registry)
	assert.NoError(t, err)

	// Test duplicate registration should fail
	err = metrics.Register(registry)
	assert.Error(t, err)
}

// TestPrometheusMetrics_Unregister tests metric unregistration.
func TestPrometheusMetrics_Unregister(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewPrometheusMetrics("saga", "monitoring")

	err := metrics.Register(registry)
	require.NoError(t, err)

	success := metrics.Unregister(registry)
	assert.True(t, success)

	// Re-registration should succeed after unregistration
	err = metrics.Register(registry)
	assert.NoError(t, err)
}

// TestSagaMetricsCollector_WithLabeledMetrics tests labeled metrics functionality.
func TestSagaMetricsCollector_WithLabeledMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry:             registry,
		EnableLabeledMetrics: true,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)
	require.NotNil(t, collector.labeledMetrics)

	// Record metrics with labels
	labels := &MetricsLabels{
		SagaType: "order",
		Status:   "started",
	}

	collector.RecordSagaStartedWithLabels("saga-1", labels)

	// Change status for completion
	labels.Status = "completed"
	collector.RecordSagaCompletedWithLabels("saga-1", 1.5, labels)

	// Verify metrics can be gathered
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Check for our labeled metrics
	foundStarted := false
	foundCompleted := false
	foundDuration := false
	foundActive := false

	for _, mf := range metricFamilies {
		switch *mf.Name {
		case "saga_monitoring_saga_started_labeled_total":
			foundStarted = true
			assert.Equal(t, 1, len(mf.Metric))
		case "saga_monitoring_saga_completed_labeled_total":
			foundCompleted = true
			assert.Equal(t, 1, len(mf.Metric))
		case "saga_monitoring_saga_duration_labeled_seconds":
			foundDuration = true
			assert.Equal(t, 1, len(mf.Metric))
		case "saga_monitoring_active_sagas_labeled":
			foundActive = true
		}
	}

	assert.True(t, foundStarted, "saga_started_total metric not found")
	assert.True(t, foundCompleted, "saga_completed_total metric not found")
	assert.True(t, foundDuration, "saga_duration_seconds metric not found")
	assert.True(t, foundActive, "active_sagas metric not found")
}

// TestSagaMetricsCollector_WithoutLabeledMetrics tests that labeled methods do nothing when disabled.
func TestSagaMetricsCollector_WithoutLabeledMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry:             registry,
		EnableLabeledMetrics: false, // Disabled
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)
	assert.Nil(t, collector.labeledMetrics)

	// These should do nothing and not panic
	labels := &MetricsLabels{
		SagaType: "order",
		Status:   "started",
	}

	collector.RecordSagaStartedWithLabels("saga-1", labels)
	collector.RecordSagaCompletedWithLabels("saga-1", 1.5, labels)
	collector.RecordSagaFailedWithLabels("saga-1", "error", labels)

	// Should not panic and should work fine
	assert.Nil(t, collector.labeledMetrics)
}

// TestSagaMetricsCollector_LabeledMetricsWithNilLabels tests handling of nil labels.
func TestSagaMetricsCollector_LabeledMetricsWithNilLabels(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry:             registry,
		EnableLabeledMetrics: true,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	// Call with nil labels - should not panic
	collector.RecordSagaStartedWithLabels("saga-1", nil)
	collector.RecordSagaCompletedWithLabels("saga-1", 1.5, nil)
	collector.RecordSagaFailedWithLabels("saga-1", "error", nil)

	// Verify no metrics were recorded with nil labels
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if strings.HasPrefix(*mf.Name, "saga_monitoring_saga_") && strings.Contains(*mf.Name, "_labeled_") {
			// Labeled metrics should have no samples since we passed nil labels
			assert.Equal(t, 0, len(mf.Metric), "Metric %s should have no samples", *mf.Name)
		}
	}
}

// TestSagaMetricsCollector_MixedLabeledAndUnlabeled tests using both labeled and unlabeled metrics.
func TestSagaMetricsCollector_MixedLabeledAndUnlabeled(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry:             registry,
		EnableLabeledMetrics: true,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	// Record unlabeled metrics
	collector.RecordSagaStarted("saga-1")
	collector.RecordSagaStarted("saga-2")

	// Record labeled metrics
	orderLabels := &MetricsLabels{SagaType: "order", Status: "started"}
	paymentLabels := &MetricsLabels{SagaType: "payment", Status: "started"}

	collector.RecordSagaStartedWithLabels("saga-3", orderLabels)
	collector.RecordSagaStartedWithLabels("saga-4", paymentLabels)

	// Verify both types of metrics exist
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	foundUnlabeled := false
	foundLabeled := false

	for _, mf := range metricFamilies {
		if *mf.Name == "saga_monitoring_saga_started_total" && !strings.Contains(*mf.Name, "_labeled_") {
			// This is the unlabeled counter
			foundUnlabeled = true
		}
		if *mf.Name == "saga_monitoring_saga_started_labeled_total" {
			// This is the labeled counter
			foundLabeled = true
		}
	}

	assert.True(t, foundUnlabeled, "Unlabeled metrics should exist")
	assert.True(t, foundLabeled, "Labeled metrics should exist")

	// Verify internal counters only count unlabeled metrics
	metrics := collector.GetMetrics()
	assert.Equal(t, int64(2), metrics.SagasStarted)
}

// TestPrometheusMetricsIntegration tests full integration with HTTP export.
func TestPrometheusMetricsIntegration(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry:             registry,
		EnableLabeledMetrics: true,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	// Simulate a saga lifecycle
	labels := &MetricsLabels{
		SagaType: "order",
		Status:   "started",
	}

	// Start saga
	collector.RecordSagaStarted("saga-1")
	collector.RecordSagaStartedWithLabels("saga-1", labels)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Complete saga
	labels.Status = "completed"
	collector.RecordSagaCompleted("saga-1", 100*time.Millisecond)
	collector.RecordSagaCompletedWithLabels("saga-1", 0.1, labels)

	// Fail another saga
	failLabels := &MetricsLabels{
		SagaType: "payment",
		Status:   "failed",
	}
	collector.RecordSagaStarted("saga-2")
	collector.RecordSagaStartedWithLabels("saga-2", failLabels)
	collector.RecordSagaFailed("saga-2", "timeout")
	collector.RecordSagaFailedWithLabels("saga-2", "timeout", failLabels)

	// Export metrics via HTTP
	exporter := NewMetricsExporter(registry)
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler := exporter.HTTPHandler()
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	// Check for key metrics (both labeled and unlabeled)
	assert.Contains(t, body, "saga_monitoring_saga_started_total")
	assert.Contains(t, body, "saga_monitoring_saga_started_labeled_total")
	assert.Contains(t, body, "saga_monitoring_saga_completed_total")
	assert.Contains(t, body, "saga_monitoring_saga_completed_labeled_total")
	assert.Contains(t, body, "saga_monitoring_saga_failed_total")
	assert.Contains(t, body, "saga_monitoring_saga_failed_labeled_total")
	assert.Contains(t, body, "saga_monitoring_saga_duration_seconds")
	assert.Contains(t, body, "saga_monitoring_saga_duration_labeled_seconds")
	assert.Contains(t, body, "saga_monitoring_active_sagas")

	// Check for labels
	assert.Contains(t, body, `saga_type="order"`)
	assert.Contains(t, body, `saga_type="payment"`)
	assert.Contains(t, body, `status="completed"`)
	assert.Contains(t, body, `reason="timeout"`)
}

// TestPrometheusMetrics_DifferentSagaTypes tests metrics with different saga types.
func TestPrometheusMetrics_DifferentSagaTypes(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry:             registry,
		EnableLabeledMetrics: true,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	// Record different saga types
	sagaTypes := []string{"order", "payment", "inventory", "shipping"}
	for _, sagaType := range sagaTypes {
		labels := &MetricsLabels{
			SagaType: sagaType,
			Status:   "started",
		}
		collector.RecordSagaStartedWithLabels("saga-"+sagaType, labels)

		labels.Status = "completed"
		collector.RecordSagaCompletedWithLabels("saga-"+sagaType, 0.5, labels)
	}

	// Gather metrics
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Find and verify saga_started_labeled_total
	var startedMetric *dto.MetricFamily
	for _, mf := range metricFamilies {
		if *mf.Name == "saga_monitoring_saga_started_labeled_total" {
			startedMetric = mf
			break
		}
	}

	require.NotNil(t, startedMetric)
	// Each saga type should have its own metric with labels
	assert.GreaterOrEqual(t, len(startedMetric.Metric), len(sagaTypes))
}

// TestPrometheusMetrics_CustomBuckets tests histogram with custom buckets.
func TestPrometheusMetrics_CustomBuckets(t *testing.T) {
	registry := prometheus.NewRegistry()
	customBuckets := []float64{0.1, 0.5, 1.0, 2.0, 5.0}

	config := &Config{
		Registry:             registry,
		EnableLabeledMetrics: true,
		DurationBuckets:      customBuckets,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	// Record some durations
	labels := &MetricsLabels{
		SagaType: "test",
		Status:   "completed",
	}

	durations := []float64{0.05, 0.3, 0.7, 1.5, 3.0}
	for i, duration := range durations {
		collector.RecordSagaCompletedWithLabels("saga-"+string(rune(i)), duration, labels)
	}

	// Export and verify buckets exist in output
	exporter := NewMetricsExporter(registry)
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler := exporter.HTTPHandler()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "saga_monitoring_saga_duration_labeled_seconds")

	// Check for bucket boundaries in the output
	for range customBuckets {
		// Prometheus exports buckets with le (less than or equal) label
		assert.Contains(t, body, `le="`)
	}
}

// BenchmarkMetricsExport benchmarks metrics export performance.
func BenchmarkMetricsExport(b *testing.B) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry:             registry,
		EnableLabeledMetrics: true,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(b, err)

	// Populate with some metrics
	for i := 0; i < 100; i++ {
		labels := &MetricsLabels{
			SagaType: "test",
			Status:   "completed",
		}
		collector.RecordSagaStartedWithLabels("saga", labels)
		collector.RecordSagaCompletedWithLabels("saga", 1.0, labels)
	}

	exporter := NewMetricsExporter(registry)
	handler := exporter.HTTPHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkLabeledMetricsCollection benchmarks labeled metrics collection.
func BenchmarkLabeledMetricsCollection(b *testing.B) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry:             registry,
		EnableLabeledMetrics: true,
	}

	collector, err := NewSagaMetricsCollector(config)
	require.NoError(b, err)

	labels := &MetricsLabels{
		SagaType: "order",
		Status:   "started",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordSagaStartedWithLabels("saga", labels)
	}
}

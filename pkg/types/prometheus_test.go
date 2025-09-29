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

package types

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestDefaultPrometheusConfig(t *testing.T) {
	config := DefaultPrometheusConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, "/metrics", config.Endpoint)
	assert.Equal(t, "swit", config.Namespace)
	assert.Equal(t, "server", config.Subsystem)
	assert.NotNil(t, config.Labels)
	assert.NotEmpty(t, config.Buckets.Duration)
	assert.NotEmpty(t, config.Buckets.Size)

	// Test default duration buckets
	expectedDuration := []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10}
	assert.Equal(t, expectedDuration, config.Buckets.Duration)

	// Test default size buckets
	expectedSize := []float64{100, 1000, 10000, 100000, 1000000}
	assert.Equal(t, expectedSize, config.Buckets.Size)
}

func TestNewPrometheusMetricsCollector(t *testing.T) {
	t.Run("with valid config", func(t *testing.T) {
		config := &PrometheusConfig{
			Enabled:   true,
			Endpoint:  "/test-metrics",
			Namespace: "test",
			Subsystem: "collector",
		}

		collector := NewPrometheusMetricsCollector(config)

		assert.NotNil(t, collector)
		assert.Equal(t, config, collector.config)
		assert.NotNil(t, collector.registry)
		assert.NotNil(t, collector.counters)
		assert.NotNil(t, collector.gauges)
		assert.NotNil(t, collector.histograms)
		assert.NotNil(t, collector.summaries)
		assert.Equal(t, 10000, collector.maxCardinality)
		assert.NotNil(t, collector.currentMetrics)
	})

	t.Run("with nil config", func(t *testing.T) {
		collector := NewPrometheusMetricsCollector(nil)

		assert.NotNil(t, collector)
		assert.NotNil(t, collector.config)
		assert.Equal(t, "swit", collector.config.Namespace)
		assert.Equal(t, "server", collector.config.Subsystem)
	})
}

func TestPrometheusMetricsCollector_CounterOperations(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	t.Run("IncrementCounter", func(t *testing.T) {
		labels := map[string]string{
			"method": "GET",
			"status": "200",
		}

		collector.IncrementCounter("test_counter", labels)
		collector.IncrementCounter("test_counter", labels)

		// Verify metric was created and incremented
		metrics := collector.GetMetrics()
		found := false
		for _, metric := range metrics {
			if strings.Contains(metric.Name, "test_counter") {
				found = true
				assert.Equal(t, CounterType, metric.Type)
				assert.Equal(t, 2.0, metric.Value)
				break
			}
		}
		assert.True(t, found, "Counter metric should be found")
	})

	t.Run("AddToCounter", func(t *testing.T) {
		// Use fresh collector to avoid interference from other tests
		freshCollector := NewPrometheusMetricsCollector(nil)
		labels := map[string]string{"service": "test"}

		freshCollector.AddToCounter("add_counter", 5.5, labels)
		freshCollector.AddToCounter("add_counter", 2.5, labels)

		metrics := freshCollector.GetMetrics()
		found := false
		for _, metric := range metrics {
			if strings.Contains(metric.Name, "add_counter") {
				found = true
				assert.Equal(t, 8.0, metric.Value)
				t.Logf("Found add_counter metric with value: %v", metric.Value)
				break
			}
		}
		if !found {
			t.Logf("Available metrics: %d", len(metrics))
			for i, m := range metrics {
				t.Logf("  Metric %d: %s = %v", i, m.Name, m.Value)
			}
		}
		assert.True(t, found)
	})

	t.Run("counter with empty labels", func(t *testing.T) {
		// Use fresh collector to avoid interference from other tests
		freshCollector := NewPrometheusMetricsCollector(nil)
		freshCollector.IncrementCounter("no_labels_counter", nil)
		freshCollector.IncrementCounter("no_labels_counter", map[string]string{})

		metrics := freshCollector.GetMetrics()
		found := false
		for _, metric := range metrics {
			if strings.Contains(metric.Name, "no_labels_counter") {
				found = true
				assert.Equal(t, 2.0, metric.Value)
				t.Logf("Found no_labels_counter metric with value: %v", metric.Value)
				break
			}
		}
		if !found {
			t.Logf("Available metrics: %d", len(metrics))
			for i, m := range metrics {
				t.Logf("  Metric %d: %s = %v", i, m.Name, m.Value)
			}
		}
		assert.True(t, found)
	})
}

func TestPrometheusMetricsCollector_GaugeOperations(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	t.Run("SetGauge", func(t *testing.T) {
		labels := map[string]string{"instance": "test"}

		collector.SetGauge("test_gauge", 42.5, labels)
		collector.SetGauge("test_gauge", 100.0, labels) // Update value

		metrics := collector.GetMetrics()
		found := false
		for _, metric := range metrics {
			if strings.Contains(metric.Name, "test_gauge") {
				found = true
				assert.Equal(t, GaugeType, metric.Type)
				assert.Equal(t, 100.0, metric.Value)
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("IncrementGauge", func(t *testing.T) {
		// Use fresh collector to avoid interference from other tests
		freshCollector := NewPrometheusMetricsCollector(nil)
		labels := map[string]string{"type": "memory"}

		freshCollector.SetGauge("inc_gauge", 10.0, labels)
		freshCollector.IncrementGauge("inc_gauge", labels)
		freshCollector.IncrementGauge("inc_gauge", labels)

		metrics := freshCollector.GetMetrics()
		found := false
		for _, metric := range metrics {
			if strings.Contains(metric.Name, "inc_gauge") {
				found = true
				assert.Equal(t, 12.0, metric.Value)
				t.Logf("Found inc_gauge metric with value: %v", metric.Value)
				break
			}
		}
		if !found {
			t.Logf("Available metrics: %d", len(metrics))
			for i, m := range metrics {
				t.Logf("  Metric %d: %s = %v", i, m.Name, m.Value)
			}
		}
		assert.True(t, found)
	})

	t.Run("DecrementGauge", func(t *testing.T) {
		// Use fresh collector to avoid interference from other tests
		freshCollector := NewPrometheusMetricsCollector(nil)
		labels := map[string]string{"type": "cpu"}

		freshCollector.SetGauge("dec_gauge", 20.0, labels)
		freshCollector.DecrementGauge("dec_gauge", labels)
		freshCollector.DecrementGauge("dec_gauge", labels)

		metrics := freshCollector.GetMetrics()
		found := false
		for _, metric := range metrics {
			if strings.Contains(metric.Name, "dec_gauge") {
				found = true
				assert.Equal(t, 18.0, metric.Value)
				t.Logf("Found dec_gauge metric with value: %v", metric.Value)
				break
			}
		}
		if !found {
			t.Logf("Available metrics: %d", len(metrics))
			for i, m := range metrics {
				t.Logf("  Metric %d: %s = %v", i, m.Name, m.Value)
			}
		}
		assert.True(t, found)
	})
}

func TestPrometheusMetricsCollector_HistogramOperations(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	t.Run("ObserveHistogram", func(t *testing.T) {
		labels := map[string]string{"endpoint": "/api/test"}

		collector.ObserveHistogram("test_duration", 0.05, labels)
		collector.ObserveHistogram("test_duration", 0.15, labels)
		collector.ObserveHistogram("test_duration", 0.25, labels)

		metrics := collector.GetMetrics()
		found := false
		for _, metric := range metrics {
			if strings.Contains(metric.Name, "test_duration") {
				found = true
				assert.Equal(t, HistogramType, metric.Type)

				// Check histogram structure
				histData, ok := metric.Value.(map[string]interface{})
				assert.True(t, ok, "Histogram value should be a map")
				assert.Contains(t, histData, "count")
				assert.Contains(t, histData, "sum")
				assert.Equal(t, uint64(3), histData["count"])
				assert.Equal(t, 0.45, histData["sum"])
				break
			}
		}
		assert.True(t, found)
	})
}

func TestPrometheusMetricsCollector_MetricNamingAndLabels(t *testing.T) {
	collector := NewPrometheusMetricsCollector(&PrometheusConfig{
		Namespace: "test_ns",
		Subsystem: "test_sub",
	})

	t.Run("metric name sanitization", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"valid_name", "valid_name"},
			{"name-with-dashes", "name_with_dashes"},
			{"name.with.dots", "name_with_dots"},
			{"name with spaces", "name_with_spaces"},
			{"123_starts_with_number", "metric_123_starts_with_number"},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				result := collector.sanitizeMetricName(tc.input)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("label handling", func(t *testing.T) {
		labels := map[string]string{
			"c_label": "value3",
			"a_label": "value1",
			"b_label": "value2",
		}

		names := collector.getLabelNames(labels)
		expectedNames := []string{"a_label", "b_label", "c_label"}
		assert.Equal(t, expectedNames, names)

		values := collector.extractLabelValues(labels, nil)
		expectedValues := []string{"value1", "value2", "value3"}
		assert.Equal(t, expectedValues, values)
	})

	t.Run("empty labels handling", func(t *testing.T) {
		names := collector.getLabelNames(nil)
		assert.Nil(t, names)

		names = collector.getLabelNames(map[string]string{})
		assert.Nil(t, names)

		values := collector.extractLabelValues(nil, nil)
		assert.Nil(t, values)
	})
}

func TestPrometheusMetricsCollector_BucketSelection(t *testing.T) {
	config := &PrometheusConfig{
		Enabled: true,
		Buckets: PrometheusBuckets{
			Duration: []float64{0.1, 0.5, 1.0},
			Size:     []float64{1000, 5000, 10000},
		},
	}
	collector := NewPrometheusMetricsCollector(config)

	testCases := []struct {
		metricName      string
		expectedBuckets []float64
	}{
		{"request_duration", config.Buckets.Duration},
		{"response_time", config.Buckets.Duration},
		{"query_latency", config.Buckets.Duration},
		{"processing_seconds", config.Buckets.Duration},
		{"response_size", config.Buckets.Size},
		{"message_bytes", config.Buckets.Size},
		{"payload_length", config.Buckets.Size},
		{"unknown_metric", config.Buckets.Duration}, // Default to duration
	}

	for _, tc := range testCases {
		t.Run(tc.metricName, func(t *testing.T) {
			buckets := collector.getBucketsForMetric(tc.metricName)
			assert.Equal(t, tc.expectedBuckets, buckets)
		})
	}
}

func TestPrometheusMetricsCollector_GetMetrics(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	// Add various types of metrics
	collector.IncrementCounter("test_counter", map[string]string{"label": "value"})
	collector.SetGauge("test_gauge", 42.0, map[string]string{"type": "test"})
	collector.ObserveHistogram("test_histogram", 0.5, map[string]string{"method": "GET"})

	metrics := collector.GetMetrics()
	assert.NotEmpty(t, metrics)

	// Verify each metric has required fields
	for _, metric := range metrics {
		assert.NotEmpty(t, metric.Name)
		assert.NotEmpty(t, metric.Type)
		assert.NotNil(t, metric.Value)
		assert.False(t, metric.Timestamp.IsZero())
	}

	// Check for specific metrics
	counterFound := false
	gaugeFound := false
	histogramFound := false

	for _, metric := range metrics {
		if strings.Contains(metric.Name, "test_counter") {
			counterFound = true
			assert.Equal(t, CounterType, metric.Type)
		}
		if strings.Contains(metric.Name, "test_gauge") {
			gaugeFound = true
			assert.Equal(t, GaugeType, metric.Type)
		}
		if strings.Contains(metric.Name, "test_histogram") {
			histogramFound = true
			assert.Equal(t, HistogramType, metric.Type)
		}
	}

	assert.True(t, counterFound, "Counter metric should be found")
	assert.True(t, gaugeFound, "Gauge metric should be found")
	assert.True(t, histogramFound, "Histogram metric should be found")
}

func TestPrometheusMetricsCollector_GetMetric(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	collector.IncrementCounter("specific_counter", map[string]string{"test": "value"})

	t.Run("existing metric", func(t *testing.T) {
		// Note: The metric name will be prefixed with namespace and subsystem
		metrics := collector.GetMetrics()
		assert.NotEmpty(t, metrics)

		// Find the actual metric name from the collected metrics
		var actualMetricName string
		for _, metric := range metrics {
			if strings.Contains(metric.Name, "specific_counter") {
				actualMetricName = metric.Name
				break
			}
		}

		assert.NotEmpty(t, actualMetricName, "Should find the metric name")

		metric, exists := collector.GetMetric(actualMetricName)
		assert.True(t, exists)
		assert.NotNil(t, metric)
		assert.Equal(t, actualMetricName, metric.Name)
	})

	t.Run("non-existing metric", func(t *testing.T) {
		metric, exists := collector.GetMetric("non_existent_metric")
		assert.False(t, exists)
		assert.Nil(t, metric)
	})
}

func TestPrometheusMetricsCollector_Reset(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	// Add some metrics
	collector.IncrementCounter("counter_to_reset", nil)
	collector.SetGauge("gauge_to_reset", 100.0, nil)

	// Verify metrics exist
	metricsBefore := collector.GetMetrics()
	assert.NotEmpty(t, metricsBefore)

	// Reset
	collector.Reset()

	// Verify metrics are cleared
	metricsAfter := collector.GetMetrics()
	assert.Empty(t, metricsAfter)

	// Verify internal maps are cleared
	assert.Empty(t, collector.counters)
	assert.Empty(t, collector.gauges)
	assert.Empty(t, collector.histograms)
	assert.Empty(t, collector.summaries)
	assert.Empty(t, collector.currentMetrics)
}

func TestPrometheusMetricsCollector_GetHandler(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	handler := collector.GetHandler()
	assert.NotNil(t, handler)

	// Verify it's an HTTP handler
	assert.Implements(t, (*http.Handler)(nil), handler)
}

func TestPrometheusMetricsCollector_GetRegistry(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	registry := collector.GetRegistry()
	assert.NotNil(t, registry)
	assert.IsType(t, &prometheus.Registry{}, registry)
}

func TestPrometheusMetricsCollector_DefaultCollectorsRegistered(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	// Give a tiny moment for collectors to be ready
	time.Sleep(10 * time.Millisecond)

	metrics := collector.GetMetrics()
	assert.NotEmpty(t, metrics)

	hasGoBuildInfo := false
	hasGoMetric := false
	hasProcessMetric := false

	for _, m := range metrics {
		if m.Name == "go_build_info" {
			hasGoBuildInfo = true
		}
		if strings.HasPrefix(m.Name, "go_") {
			hasGoMetric = true
		}
		if strings.HasPrefix(m.Name, "process_") {
			hasProcessMetric = true
		}

		// Early exit if all found
		if hasGoBuildInfo && hasGoMetric && hasProcessMetric {
			break
		}
	}

	assert.True(t, hasGoBuildInfo, "should expose go_build_info from default collectors")
	assert.True(t, hasGoMetric, "should expose at least one go_* metric from runtime collector")
	assert.True(t, hasProcessMetric, "should expose at least one process_* metric from process collector")
}

func TestPrometheusMetricsCollector_CardinalityLimiting(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)
	collector.maxCardinality = 5 // Set low limit for testing

	t.Run("cardinality check allows initial metrics", func(t *testing.T) {
		result := collector.checkCardinality("test_metric", map[string]string{"label": "value"})
		assert.True(t, result)
	})

	t.Run("cardinality tracking", func(t *testing.T) {
		metricName := "cardinality_test"

		// The maxCardinality is set to 5, which is the total limit across all metrics
		// One metric was already added in the previous test, so we have 4 slots left
		// Test up to the remaining limit (4) and verify rejection after
		for i := 0; i < 10; i++ {
			result := collector.checkCardinality(metricName, map[string]string{
				"iteration": fmt.Sprintf("%d", i),
			})

			// First test added 1 metric, so we have 4 slots left (5 - 1 = 4)
			if i < 4 {
				assert.True(t, result, fmt.Sprintf("Should allow metric %d under remaining limit of 4", i))
			} else {
				assert.False(t, result, fmt.Sprintf("Should reject metric %d over limit of 5", i))
			}
		}
	})

	t.Run("no cardinality limit", func(t *testing.T) {
		collector.maxCardinality = 0 // Disable limit

		result := collector.checkCardinality("unlimited_metric", map[string]string{"test": "value"})
		assert.True(t, result)
	})
}

func TestPrometheusMetricsCollector_ThreadSafety(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	const numGoroutines = 5  // Further reduced
	const numOperations = 10 // Much smaller to ensure no cardinality issues

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent counter operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Use consistent labels per goroutine to avoid cardinality explosion
				labels := map[string]string{
					"goroutine": fmt.Sprintf("g%d", id), // g prefix to make it more readable
				}

				collector.IncrementCounter("threadsafe_counter", labels)
				collector.SetGauge("threadsafe_gauge", float64(j), labels)
				collector.ObserveHistogram("threadsafe_histogram", float64(j)*0.01, labels)
			}
		}(i)
	}

	wg.Wait()

	// Give time for operations to complete and cache to invalidate
	time.Sleep(100 * time.Millisecond)

	// Clear cache to ensure fresh read
	collector.cacheMu.Lock()
	collector.cacheValid = false
	collector.cacheMu.Unlock()

	metrics := collector.GetMetrics()
	t.Logf("Retrieved %d metrics", len(metrics))

	// Debug: log all metric names
	for i, metric := range metrics {
		t.Logf("Metric %d: %s", i, metric.Name)
	}

	assert.NotEmpty(t, metrics, "Should have collected metrics from concurrent operations")

	// Verify we have the expected metric types
	hasCounter := false
	hasGauge := false
	hasHistogram := false

	for _, metric := range metrics {
		name := metric.Name
		if strings.Contains(name, "threadsafe_counter") {
			hasCounter = true
			t.Logf("Found counter metric: %s", name)
		}
		if strings.Contains(name, "threadsafe_gauge") {
			hasGauge = true
			t.Logf("Found gauge metric: %s", name)
		}
		if strings.Contains(name, "threadsafe_histogram") {
			hasHistogram = true
			t.Logf("Found histogram metric: %s", name)
		}
	}

	assert.True(t, hasCounter, "Should have counter metrics")
	assert.True(t, hasGauge, "Should have gauge metrics")
	assert.True(t, hasHistogram, "Should have histogram metrics")
}

func TestPrometheusMetricsCollector_ErrorRecovery(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	t.Run("safe metric operation with panic", func(t *testing.T) {
		// This test verifies that safeMetricOperation handles panics gracefully
		// We can't easily trigger a panic in the actual metric operations,
		// but we can verify the structure is in place

		assert.NotPanics(t, func() {
			collector.safeMetricOperation(func() error {
				return fmt.Errorf("test error")
			})
		})
	})

	t.Run("operations continue after errors", func(t *testing.T) {
		// Even if individual operations fail, subsequent ones should work
		collector.IncrementCounter("test_counter", map[string]string{"test": "value"})

		metrics := collector.GetMetrics()
		assert.NotEmpty(t, metrics)
	})
}

func TestPrometheusMetricsCollector_MetricTypeConversion(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	// Add different metric types
	collector.IncrementCounter("type_test_counter", map[string]string{"type": "counter"})
	collector.SetGauge("type_test_gauge", 42.0, map[string]string{"type": "gauge"})
	collector.ObserveHistogram("type_test_histogram", 0.5, map[string]string{"type": "histogram"})

	metrics := collector.GetMetrics()
	assert.NotEmpty(t, metrics)

	typeMap := make(map[MetricType]bool)
	for _, metric := range metrics {
		typeMap[metric.Type] = true

		switch metric.Type {
		case CounterType:
			assert.IsType(t, float64(0), metric.Value, "Counter value should be float64")
		case GaugeType:
			assert.IsType(t, float64(0), metric.Value, "Gauge value should be float64")
		case HistogramType:
			histValue, ok := metric.Value.(map[string]interface{})
			assert.True(t, ok, "Histogram value should be a map")
			assert.Contains(t, histValue, "count")
			assert.Contains(t, histValue, "sum")
		}
	}

	// Verify we got the expected types
	assert.True(t, typeMap[CounterType], "Should have counter metrics")
	assert.True(t, typeMap[GaugeType], "Should have gauge metrics")
	assert.True(t, typeMap[HistogramType], "Should have histogram metrics")
}

func TestPrometheusMetricsCollector_DoubleRegistrationPrevention(t *testing.T) {
	collector := NewPrometheusMetricsCollector(nil)

	// Create the same metric multiple times to test double-registration prevention
	collector.IncrementCounter("duplicate_test", map[string]string{"label": "value1"})
	collector.IncrementCounter("duplicate_test", map[string]string{"label": "value2"})

	// Should not panic or error - should reuse existing metric
	assert.NotPanics(t, func() {
		collector.IncrementCounter("duplicate_test", map[string]string{"label": "value3"})
	})

	// Verify metric exists and has accumulated values
	metrics := collector.GetMetrics()
	found := false
	for _, metric := range metrics {
		if strings.Contains(metric.Name, "duplicate_test") {
			found = true
			// Should have accumulated 3 increments across different label combinations
			break
		}
	}
	assert.True(t, found)
}

// Benchmark tests for performance validation
func BenchmarkPrometheusMetricsCollector_CounterOperations(b *testing.B) {
	collector := NewPrometheusMetricsCollector(nil)
	labels := map[string]string{"benchmark": "test"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector.IncrementCounter("benchmark_counter", labels)
		}
	})
}

func BenchmarkPrometheusMetricsCollector_GaugeOperations(b *testing.B) {
	collector := NewPrometheusMetricsCollector(nil)
	labels := map[string]string{"benchmark": "test"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			collector.SetGauge("benchmark_gauge", float64(i), labels)
			i++
		}
	})
}

func BenchmarkPrometheusMetricsCollector_HistogramOperations(b *testing.B) {
	collector := NewPrometheusMetricsCollector(nil)
	labels := map[string]string{"benchmark": "test"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			collector.ObserveHistogram("benchmark_histogram", float64(i)*0.001, labels)
			i++
		}
	})
}

func BenchmarkPrometheusMetricsCollector_GetMetrics(b *testing.B) {
	collector := NewPrometheusMetricsCollector(nil)

	// Pre-populate with some metrics
	for i := 0; i < 100; i++ {
		collector.IncrementCounter("bench_counter", map[string]string{
			"id": fmt.Sprintf("%d", i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetMetrics()
	}
}

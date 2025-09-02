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

package server

import (
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/innovationmech/swit/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsRegistry(t *testing.T) {
	registry := NewMetricsRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.predefinedMetrics)
	assert.NotNil(t, registry.customMetrics)

	// Verify predefined metrics are loaded
	predefinedCount, customCount, totalCount := registry.GetMetricsCount()
	assert.Greater(t, predefinedCount, 0, "Should have predefined metrics")
	assert.Equal(t, 0, customCount, "Should start with no custom metrics")
	assert.Equal(t, predefinedCount, totalCount)

	// Verify some expected predefined metrics exist
	expectedPredefined := []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"grpc_server_started_total",
		"server_uptime_seconds",
		"errors_total",
	}

	for _, name := range expectedPredefined {
		def, exists := registry.GetMetricDefinition(name)
		assert.True(t, exists, "Predefined metric %s should exist", name)
		assert.NotNil(t, def)
		assert.Equal(t, name, def.Name)
	}
}

func TestMetricsRegistry_RegisterMetric(t *testing.T) {
	registry := NewMetricsRegistry()

	t.Run("valid custom metric registration", func(t *testing.T) {
		definition := MetricDefinition{
			Name:        "custom_counter",
			Type:        types.CounterType,
			Description: "A custom counter metric",
			Labels:      []string{"service", "method"},
		}

		err := registry.RegisterMetric(definition)
		assert.NoError(t, err)

		// Verify metric was registered
		def, exists := registry.GetMetricDefinition("custom_counter")
		assert.True(t, exists)
		assert.Equal(t, definition, *def)

		// Verify counts updated
		_, customCount, _ := registry.GetMetricsCount()
		assert.Equal(t, 1, customCount)
	})

	t.Run("invalid metric - empty name", func(t *testing.T) {
		definition := MetricDefinition{
			Name: "",
			Type: types.CounterType,
		}

		err := registry.RegisterMetric(definition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metric name cannot be empty")
	})

	t.Run("invalid metric - empty type", func(t *testing.T) {
		definition := MetricDefinition{
			Name: "test_metric",
			Type: "",
		}

		err := registry.RegisterMetric(definition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metric type cannot be empty")
	})

	t.Run("duplicate custom metric", func(t *testing.T) {
		definition := MetricDefinition{
			Name: "duplicate_custom",
			Type: types.GaugeType,
		}

		err := registry.RegisterMetric(definition)
		assert.NoError(t, err)

		// Try to register again
		err = registry.RegisterMetric(definition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists as custom metric")
	})

	t.Run("conflict with predefined metric", func(t *testing.T) {
		definition := MetricDefinition{
			Name: "http_requests_total", // This is a predefined metric
			Type: types.CounterType,
		}

		err := registry.RegisterMetric(definition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists as predefined metric")
	})
}

func TestMetricsRegistry_GetMetricDefinition(t *testing.T) {
	registry := NewMetricsRegistry()

	// Register a custom metric
	customDef := MetricDefinition{
		Name:        "custom_test_metric",
		Type:        types.HistogramType,
		Description: "Test histogram",
		Labels:      []string{"endpoint", "status"},
		Buckets:     []float64{0.1, 0.5, 1.0, 5.0},
	}
	err := registry.RegisterMetric(customDef)
	require.NoError(t, err)

	t.Run("get predefined metric", func(t *testing.T) {
		def, exists := registry.GetMetricDefinition("http_requests_total")
		assert.True(t, exists)
		assert.NotNil(t, def)
		assert.Equal(t, "http_requests_total", def.Name)
		assert.Equal(t, types.CounterType, def.Type)
		assert.Contains(t, def.Labels, "method")
		assert.Contains(t, def.Labels, "status")
	})

	t.Run("get custom metric", func(t *testing.T) {
		def, exists := registry.GetMetricDefinition("custom_test_metric")
		assert.True(t, exists)
		assert.NotNil(t, def)
		assert.Equal(t, customDef, *def)
	})

	t.Run("get non-existent metric", func(t *testing.T) {
		def, exists := registry.GetMetricDefinition("non_existent_metric")
		assert.False(t, exists)
		assert.Nil(t, def)
	})
}

func TestMetricsRegistry_ListMetrics(t *testing.T) {
	registry := NewMetricsRegistry()

	// Register some custom metrics
	customMetrics := []MetricDefinition{
		{
			Name: "custom_gauge",
			Type: types.GaugeType,
		},
		{
			Name: "custom_counter",
			Type: types.CounterType,
		},
		{
			Name:    "another_custom",
			Type:    types.HistogramType,
			Buckets: []float64{0.1, 1.0},
		},
	}

	for _, metric := range customMetrics {
		err := registry.RegisterMetric(metric)
		require.NoError(t, err)
	}

	t.Run("list all metrics", func(t *testing.T) {
		metrics := registry.ListMetrics()
		assert.NotEmpty(t, metrics)

		_, _, totalCount := registry.GetMetricsCount()
		assert.Len(t, metrics, totalCount)

		// Verify sorting
		names := make([]string, len(metrics))
		for i, metric := range metrics {
			names[i] = metric.Name
		}
		assert.True(t, sort.StringsAreSorted(names), "Metrics should be sorted by name")

		// Verify custom metrics are included
		customFound := 0
		for _, metric := range metrics {
			for _, custom := range customMetrics {
				if metric.Name == custom.Name {
					customFound++
					assert.Equal(t, custom.Type, metric.Type)
					break
				}
			}
		}
		assert.Equal(t, len(customMetrics), customFound, "All custom metrics should be found")
	})

	t.Run("list predefined metrics only", func(t *testing.T) {
		predefinedMetrics := registry.ListPredefinedMetrics()
		assert.NotEmpty(t, predefinedMetrics)

		predefinedCount, _, _ := registry.GetMetricsCount()
		assert.Len(t, predefinedMetrics, predefinedCount)

		// Verify sorting
		names := make([]string, len(predefinedMetrics))
		for i, metric := range predefinedMetrics {
			names[i] = metric.Name
		}
		assert.True(t, sort.StringsAreSorted(names), "Predefined metrics should be sorted")

		// Ensure no custom metrics are included
		for _, metric := range predefinedMetrics {
			for _, custom := range customMetrics {
				assert.NotEqual(t, custom.Name, metric.Name, "Custom metric should not be in predefined list")
			}
		}
	})

	t.Run("list custom metrics only", func(t *testing.T) {
		customMetricsList := registry.ListCustomMetrics()
		assert.Len(t, customMetricsList, len(customMetrics))

		// Verify sorting
		names := make([]string, len(customMetricsList))
		for i, metric := range customMetricsList {
			names[i] = metric.Name
		}
		assert.True(t, sort.StringsAreSorted(names), "Custom metrics should be sorted")

		// Verify only custom metrics are included
		for _, metric := range customMetricsList {
			found := false
			for _, custom := range customMetrics {
				if metric.Name == custom.Name {
					found = true
					assert.Equal(t, custom.Type, metric.Type)
					break
				}
			}
			assert.True(t, found, "Listed custom metric should match registered custom metric")
		}
	})
}

func TestMetricsRegistry_UnregisterCustomMetric(t *testing.T) {
	registry := NewMetricsRegistry()

	// Register a custom metric
	customMetric := MetricDefinition{
		Name: "metric_to_unregister",
		Type: types.CounterType,
	}
	err := registry.RegisterMetric(customMetric)
	require.NoError(t, err)

	// Verify it exists
	_, exists := registry.GetMetricDefinition("metric_to_unregister")
	assert.True(t, exists)

	t.Run("successfully unregister custom metric", func(t *testing.T) {
		err := registry.UnregisterCustomMetric("metric_to_unregister")
		assert.NoError(t, err)

		// Verify it's gone
		_, exists := registry.GetMetricDefinition("metric_to_unregister")
		assert.False(t, exists)

		// Verify counts updated
		_, customCount, _ := registry.GetMetricsCount()
		assert.Equal(t, 0, customCount)
	})

	t.Run("unregister non-existent custom metric", func(t *testing.T) {
		err := registry.UnregisterCustomMetric("non_existent_metric")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("cannot unregister predefined metric", func(t *testing.T) {
		err := registry.UnregisterCustomMetric("http_requests_total")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestMetricsRegistry_IsRegistered(t *testing.T) {
	registry := NewMetricsRegistry()

	// Register a custom metric
	customMetric := MetricDefinition{
		Name: "registered_custom",
		Type: types.GaugeType,
	}
	err := registry.RegisterMetric(customMetric)
	require.NoError(t, err)

	t.Run("predefined metric is registered", func(t *testing.T) {
		assert.True(t, registry.IsRegistered("http_requests_total"))
	})

	t.Run("custom metric is registered", func(t *testing.T) {
		assert.True(t, registry.IsRegistered("registered_custom"))
	})

	t.Run("unregistered metric is not registered", func(t *testing.T) {
		assert.False(t, registry.IsRegistered("unregistered_metric"))
	})
}

func TestMetricsRegistry_ValidateMetric(t *testing.T) {
	registry := NewMetricsRegistry()

	t.Run("valid counter metric", func(t *testing.T) {
		definition := MetricDefinition{
			Name:        "valid_counter",
			Type:        types.CounterType,
			Description: "A valid counter",
			Labels:      []string{"service", "method"},
		}

		err := registry.ValidateMetric(definition)
		assert.NoError(t, err)
	})

	t.Run("valid gauge metric", func(t *testing.T) {
		definition := MetricDefinition{
			Name:   "valid_gauge",
			Type:   types.GaugeType,
			Labels: []string{"instance"},
		}

		err := registry.ValidateMetric(definition)
		assert.NoError(t, err)
	})

	t.Run("valid histogram metric", func(t *testing.T) {
		definition := MetricDefinition{
			Name:    "valid_histogram",
			Type:    types.HistogramType,
			Buckets: []float64{0.1, 0.5, 1.0, 5.0, 10.0},
			Labels:  []string{"endpoint"},
		}

		err := registry.ValidateMetric(definition)
		assert.NoError(t, err)
	})

	t.Run("valid summary metric", func(t *testing.T) {
		definition := MetricDefinition{
			Name: "valid_summary",
			Type: types.SummaryType,
			Objectives: map[float64]float64{
				0.5:  0.05,
				0.9:  0.01,
				0.99: 0.001,
			},
			Labels: []string{"service"},
		}

		err := registry.ValidateMetric(definition)
		assert.NoError(t, err)
	})

	t.Run("invalid metric - empty name", func(t *testing.T) {
		definition := MetricDefinition{
			Name: "",
			Type: types.CounterType,
		}

		err := registry.ValidateMetric(definition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metric name is required")
	})

	t.Run("invalid metric - empty type", func(t *testing.T) {
		definition := MetricDefinition{
			Name: "test_metric",
			Type: "",
		}

		err := registry.ValidateMetric(definition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metric type is required")
	})

	t.Run("invalid metric - invalid type", func(t *testing.T) {
		definition := MetricDefinition{
			Name: "invalid_type_metric",
			Type: MetricType("invalid_type"),
		}

		err := registry.ValidateMetric(definition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid metric type")
	})

	t.Run("invalid histogram - no buckets", func(t *testing.T) {
		definition := MetricDefinition{
			Name: "histogram_no_buckets",
			Type: types.HistogramType,
		}

		err := registry.ValidateMetric(definition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "histogram metric must have buckets defined")
	})

	t.Run("invalid summary - no objectives", func(t *testing.T) {
		definition := MetricDefinition{
			Name: "summary_no_objectives",
			Type: types.SummaryType,
		}

		err := registry.ValidateMetric(definition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "summary metric must have objectives defined")
	})

	t.Run("invalid labels - empty label name", func(t *testing.T) {
		definition := MetricDefinition{
			Name:   "invalid_labels",
			Type:   types.CounterType,
			Labels: []string{"valid_label", "", "another_valid"},
		}

		err := registry.ValidateMetric(definition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "label name cannot be empty")
	})
}

func TestMetricsRegistry_GetMetricNames(t *testing.T) {
	registry := NewMetricsRegistry()

	// Register some custom metrics
	customMetrics := []string{"custom_a", "custom_b", "custom_c"}
	for _, name := range customMetrics {
		err := registry.RegisterMetric(MetricDefinition{
			Name: name,
			Type: types.CounterType,
		})
		require.NoError(t, err)
	}

	t.Run("get predefined metric names", func(t *testing.T) {
		names := registry.GetPredefinedMetricNames()
		assert.NotEmpty(t, names)
		assert.True(t, sort.StringsAreSorted(names), "Names should be sorted")

		// Verify some expected names
		expectedNames := []string{
			"http_requests_total",
			"grpc_server_started_total",
			"server_uptime_seconds",
		}

		for _, expected := range expectedNames {
			assert.Contains(t, names, expected)
		}

		// Ensure no custom metrics
		for _, custom := range customMetrics {
			assert.NotContains(t, names, custom)
		}
	})

	t.Run("get custom metric names", func(t *testing.T) {
		names := registry.GetCustomMetricNames()
		assert.Len(t, names, len(customMetrics))
		assert.True(t, sort.StringsAreSorted(names), "Names should be sorted")

		for _, custom := range customMetrics {
			assert.Contains(t, names, custom)
		}
	})
}

func TestMetricsRegistry_GetMetricsCount(t *testing.T) {
	registry := NewMetricsRegistry()

	initialPredefined, initialCustom, initialTotal := registry.GetMetricsCount()
	assert.Greater(t, initialPredefined, 0)
	assert.Equal(t, 0, initialCustom)
	assert.Equal(t, initialPredefined, initialTotal)

	// Add custom metrics
	customMetrics := []MetricDefinition{
		{Name: "custom_1", Type: types.CounterType},
		{Name: "custom_2", Type: types.GaugeType},
		{Name: "custom_3", Type: types.HistogramType, Buckets: []float64{1, 5}},
	}

	for _, metric := range customMetrics {
		err := registry.RegisterMetric(metric)
		require.NoError(t, err)
	}

	finalPredefined, finalCustom, finalTotal := registry.GetMetricsCount()
	assert.Equal(t, initialPredefined, finalPredefined, "Predefined count should not change")
	assert.Equal(t, len(customMetrics), finalCustom)
	assert.Equal(t, finalPredefined+finalCustom, finalTotal)
}

func TestMetricsRegistry_PredefinedMetricsContent(t *testing.T) {
	registry := NewMetricsRegistry()

	testCases := []struct {
		name           string
		expectedType   MetricType
		expectedLabels []string
	}{
		{
			name:           "http_requests_total",
			expectedType:   types.CounterType,
			expectedLabels: []string{"method", "endpoint", "status"},
		},
		{
			name:           "http_request_duration_seconds",
			expectedType:   types.HistogramType,
			expectedLabels: []string{"method", "endpoint"},
		},
		{
			name:           "grpc_server_started_total",
			expectedType:   types.CounterType,
			expectedLabels: []string{"method"},
		},
		{
			name:           "server_uptime_seconds",
			expectedType:   types.GaugeType,
			expectedLabels: []string{"service"},
		},
		{
			name:           "transport_status",
			expectedType:   types.GaugeType,
			expectedLabels: []string{"transport", "status"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			def, exists := registry.GetMetricDefinition(tc.name)
			assert.True(t, exists, "Predefined metric %s should exist", tc.name)
			assert.NotNil(t, def)
			assert.Equal(t, tc.name, def.Name)
			assert.Equal(t, tc.expectedType, def.Type)
			assert.Equal(t, tc.expectedLabels, def.Labels)
			assert.NotEmpty(t, def.Description)

			// Verify histogram buckets for histogram metrics
			if tc.expectedType == types.HistogramType {
				assert.NotEmpty(t, def.Buckets, "Histogram should have buckets")
			}
		})
	}
}

func TestMetricsRegistry_ThreadSafety(t *testing.T) {
	registry := NewMetricsRegistry()

	const numGoroutines = 10
	const operationsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // Three types of operations

	// Test concurrent registrations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				metricName := fmt.Sprintf("concurrent_metric_%d_%d", id, j)
				err := registry.RegisterMetric(MetricDefinition{
					Name: metricName,
					Type: types.CounterType,
				})
				assert.NoError(t, err)
			}
		}(i)
	}

	// Test concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				registry.ListMetrics()
				registry.GetMetricsCount()
				registry.IsRegistered("http_requests_total")
			}
		}(i)
	}

	// Test concurrent unregistrations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// First register some metrics to unregister
			for j := 0; j < operationsPerGoroutine/2; j++ {
				metricName := fmt.Sprintf("to_unregister_%d_%d", id, j)
				registry.RegisterMetric(MetricDefinition{
					Name: metricName,
					Type: types.GaugeType,
				})

				// Then try to unregister it
				registry.UnregisterCustomMetric(metricName)
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	metrics := registry.ListMetrics()
	assert.NotEmpty(t, metrics)

	predefined, custom, total := registry.GetMetricsCount()
	assert.Equal(t, len(metrics), total)
	assert.Equal(t, predefined+custom, total)
}

func TestMetricsRegistry_EdgeCases(t *testing.T) {
	registry := NewMetricsRegistry()

	t.Run("empty buckets slice for histogram", func(t *testing.T) {
		definition := MetricDefinition{
			Name:    "empty_buckets_histogram",
			Type:    types.HistogramType,
			Buckets: []float64{}, // Empty slice
		}

		err := registry.ValidateMetric(definition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "histogram metric must have buckets defined")
	})

	t.Run("empty objectives map for summary", func(t *testing.T) {
		definition := MetricDefinition{
			Name:       "empty_objectives_summary",
			Type:       types.SummaryType,
			Objectives: map[float64]float64{}, // Empty map
		}

		err := registry.ValidateMetric(definition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "summary metric must have objectives defined")
	})

	t.Run("nil labels slice", func(t *testing.T) {
		definition := MetricDefinition{
			Name:   "nil_labels",
			Type:   types.CounterType,
			Labels: nil,
		}

		err := registry.ValidateMetric(definition)
		assert.NoError(t, err) // nil labels should be valid
	})

	t.Run("valid empty labels slice", func(t *testing.T) {
		definition := MetricDefinition{
			Name:   "empty_labels",
			Type:   types.GaugeType,
			Labels: []string{},
		}

		err := registry.ValidateMetric(definition)
		assert.NoError(t, err) // Empty labels should be valid
	})
}

// Benchmark tests for performance validation
func BenchmarkMetricsRegistry_RegisterMetric(b *testing.B) {
	registry := NewMetricsRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		definition := MetricDefinition{
			Name: fmt.Sprintf("benchmark_metric_%d", i),
			Type: types.CounterType,
		}
		registry.RegisterMetric(definition)
	}
}

func BenchmarkMetricsRegistry_GetMetricDefinition(b *testing.B) {
	registry := NewMetricsRegistry()

	// Pre-register some metrics
	for i := 0; i < 1000; i++ {
		registry.RegisterMetric(MetricDefinition{
			Name: fmt.Sprintf("benchmark_metric_%d", i),
			Type: types.GaugeType,
		})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			registry.GetMetricDefinition(fmt.Sprintf("benchmark_metric_%d", i%1000))
			i++
		}
	})
}

func BenchmarkMetricsRegistry_ListMetrics(b *testing.B) {
	registry := NewMetricsRegistry()

	// Pre-register metrics
	for i := 0; i < 100; i++ {
		registry.RegisterMetric(MetricDefinition{
			Name: fmt.Sprintf("list_benchmark_%d", i),
			Type: types.CounterType,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.ListMetrics()
	}
}

func BenchmarkMetricsRegistry_ValidateMetric(b *testing.B) {
	registry := NewMetricsRegistry()

	definition := MetricDefinition{
		Name:    "benchmark_validation",
		Type:    types.HistogramType,
		Labels:  []string{"service", "method", "endpoint"},
		Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.ValidateMetric(definition)
	}
}

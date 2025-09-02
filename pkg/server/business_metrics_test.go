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

package server

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBusinessMetricsManager(t *testing.T) {
	t.Run("with custom collector and registry", func(t *testing.T) {
		collector := NewSimpleMetricsCollector()
		registry := NewMetricsRegistry()

		bmm := NewBusinessMetricsManager("test-service", collector, registry)

		assert.NotNil(t, bmm)
		assert.Equal(t, "test-service", bmm.serviceName)
		assert.Same(t, collector, bmm.collector)
		assert.Same(t, registry, bmm.registry)
		assert.Empty(t, bmm.GetRegisteredHooks())
	})

	t.Run("with nil collector and registry", func(t *testing.T) {
		bmm := NewBusinessMetricsManager("test-service", nil, nil)

		assert.NotNil(t, bmm)
		assert.Equal(t, "test-service", bmm.serviceName)
		assert.NotNil(t, bmm.collector)
		assert.NotNil(t, bmm.registry)
	})
}

func TestBusinessMetricsManager_RegisterHook(t *testing.T) {
	bmm := NewBusinessMetricsManager("test-service", nil, nil)

	t.Run("register valid hook", func(t *testing.T) {
		hook := NewLoggingBusinessMetricsHook()

		err := bmm.RegisterHook(hook)

		assert.NoError(t, err)
		assert.Contains(t, bmm.GetRegisteredHooks(), hook.GetHookName())
	})

	t.Run("register nil hook", func(t *testing.T) {
		err := bmm.RegisterHook(nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hook cannot be nil")
	})

	t.Run("register hook with empty name", func(t *testing.T) {
		hook := &testBusinessMetricsHook{name: ""}

		err := bmm.RegisterHook(hook)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hook name cannot be empty")
	})

	t.Run("register duplicate hook", func(t *testing.T) {
		hook1 := &testBusinessMetricsHook{name: "test_hook"}
		hook2 := &testBusinessMetricsHook{name: "test_hook"}

		err1 := bmm.RegisterHook(hook1)
		err2 := bmm.RegisterHook(hook2)

		assert.NoError(t, err1)
		assert.Error(t, err2)
		assert.Contains(t, err2.Error(), "already registered")
	})
}

func TestBusinessMetricsManager_UnregisterHook(t *testing.T) {
	bmm := NewBusinessMetricsManager("test-service", nil, nil)
	hook := NewLoggingBusinessMetricsHook()

	t.Run("unregister existing hook", func(t *testing.T) {
		_ = bmm.RegisterHook(hook)

		err := bmm.UnregisterHook(hook.GetHookName())

		assert.NoError(t, err)
		assert.NotContains(t, bmm.GetRegisteredHooks(), hook.GetHookName())
	})

	t.Run("unregister non-existent hook", func(t *testing.T) {
		err := bmm.UnregisterHook("non-existent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestBusinessMetricsManager_RecordMetrics(t *testing.T) {
	collector := NewSimpleMetricsCollector()
	bmm := NewBusinessMetricsManager("test-service", collector, nil)

	// Add a test hook to verify events are triggered
	testHook := &testBusinessMetricsHook{name: "test_hook"}
	_ = bmm.RegisterHook(testHook)

	t.Run("record counter metric", func(t *testing.T) {
		labels := map[string]string{"endpoint": "/api/test"}

		bmm.RecordCounter("api_requests_total", 1.0, labels)

		// Allow time for goroutine to execute
		time.Sleep(10 * time.Millisecond)

		// Verify metric was recorded in collector
		metrics := collector.GetMetrics()
		var found bool
		for _, metric := range metrics {
			if metric.Name == "api_requests_total" {
				found = true
				assert.Equal(t, MetricTypeCounter, metric.Type)
				assert.Equal(t, int64(1), metric.Value)
				assert.Equal(t, "test-service", metric.Labels["service"])
				assert.Equal(t, "/api/test", metric.Labels["endpoint"])
				break
			}
		}
		assert.True(t, found, "Counter metric not found")

		// Verify hook was triggered
		testHook.mu.RLock()
		assert.True(t, len(testHook.events) > 0)
		lastEvent := testHook.events[len(testHook.events)-1]
		assert.Equal(t, "api_requests_total", lastEvent.Name)
		assert.Equal(t, MetricTypeCounter, lastEvent.Type)
		testHook.mu.RUnlock()
	})

	t.Run("record gauge metric", func(t *testing.T) {
		labels := map[string]string{"queue": "processing"}

		bmm.RecordGauge("queue_size", 42.0, labels)

		// Allow time for goroutine to execute
		time.Sleep(10 * time.Millisecond)

		// Verify metric was recorded in collector
		metrics := collector.GetMetrics()
		var found bool
		for _, metric := range metrics {
			if metric.Name == "queue_size" {
				found = true
				assert.Equal(t, MetricTypeGauge, metric.Type)
				assert.Equal(t, 42.0, metric.Value)
				assert.Equal(t, "test-service", metric.Labels["service"])
				assert.Equal(t, "processing", metric.Labels["queue"])
				break
			}
		}
		assert.True(t, found, "Gauge metric not found")
	})

	t.Run("record histogram metric", func(t *testing.T) {
		labels := map[string]string{"method": "GET"}

		bmm.RecordHistogram("http_request_duration_seconds", 0.123, labels)

		// Allow time for goroutine to execute
		time.Sleep(10 * time.Millisecond)

		// Verify metric was recorded in collector
		metrics := collector.GetMetrics()
		var found bool
		for _, metric := range metrics {
			if metric.Name == "http_request_duration_seconds" {
				found = true
				assert.Equal(t, MetricTypeHistogram, metric.Type)
				assert.Equal(t, "test-service", metric.Labels["service"])
				assert.Equal(t, "GET", metric.Labels["method"])
				break
			}
		}
		assert.True(t, found, "Histogram metric not found")
	})

	t.Run("automatic service label addition", func(t *testing.T) {
		// Test with nil labels
		bmm.RecordCounter("test_metric_nil", 1.0, nil)

		// Test with empty labels
		bmm.RecordCounter("test_metric_empty", 1.0, map[string]string{})

		// Test with existing service label
		bmm.RecordCounter("test_metric_existing", 1.0, map[string]string{"service": "other-service"})

		time.Sleep(10 * time.Millisecond)

		metrics := collector.GetMetrics()

		for _, metric := range metrics {
			switch metric.Name {
			case "test_metric_nil", "test_metric_empty":
				assert.Equal(t, "test-service", metric.Labels["service"])
			case "test_metric_existing":
				// Should not override existing service label
				assert.Equal(t, "other-service", metric.Labels["service"])
			}
		}
	})
}

func TestBusinessMetricsManager_CustomMetrics(t *testing.T) {
	bmm := NewBusinessMetricsManager("test-service", nil, nil)

	t.Run("register and retrieve custom metric", func(t *testing.T) {
		definition := MetricDefinition{
			Name:        "custom_business_metric",
			Type:        MetricTypeGauge,
			Description: "Custom business metric for testing",
			Labels:      []string{"department", "region"},
		}

		err := bmm.RegisterCustomMetric(definition)

		assert.NoError(t, err)

		retrieved, found := bmm.GetCustomMetric("custom_business_metric")
		assert.True(t, found)
		assert.Equal(t, definition.Name, retrieved.Name)
		assert.Equal(t, definition.Type, retrieved.Type)
		assert.Equal(t, definition.Description, retrieved.Description)
	})

	t.Run("get all metrics includes predefined and custom", func(t *testing.T) {
		allMetrics := bmm.GetAllMetrics()

		// Should include predefined server metrics
		_, hasServerUptime := allMetrics["server_uptime_seconds"]
		assert.True(t, hasServerUptime)

		// Should include the custom metric we registered
		_, hasCustom := allMetrics["custom_business_metric"]
		assert.True(t, hasCustom)

		// Should have a reasonable number of metrics
		assert.GreaterOrEqual(t, len(allMetrics), 20) // At least 20+ predefined metrics
	})
}

func TestBusinessMetricsManager_ConcurrentAccess(t *testing.T) {
	bmm := NewBusinessMetricsManager("test-service", nil, nil)

	const numGoroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup

	// Test concurrent hook registration/unregistration
	t.Run("concurrent hook management", func(t *testing.T) {
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					hookName := fmt.Sprintf("hook_%d_%d", id, j)
					hook := &testBusinessMetricsHook{name: hookName}

					err := bmm.RegisterHook(hook)
					if err == nil {
						_ = bmm.UnregisterHook(hookName)
					}
				}
			}(i)
		}

		wg.Wait()

		// Should not crash and should have consistent state
		hooks := bmm.GetRegisteredHooks()
		assert.NotNil(t, hooks) // Should not be nil
	})

	// Test concurrent metric recording
	t.Run("concurrent metric recording", func(t *testing.T) {
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					labels := map[string]string{"goroutine": fmt.Sprintf("%d", id)}

					bmm.RecordCounter("concurrent_counter", 1.0, labels)
					bmm.RecordGauge("concurrent_gauge", float64(j), labels)
					bmm.RecordHistogram("concurrent_histogram", float64(j)*0.1, labels)
				}
			}(i)
		}

		wg.Wait()

		// Should not crash and metrics should be recorded
		metrics := bmm.GetCollector().GetMetrics()
		assert.NotEmpty(t, metrics)
	})
}

// Test hook implementations

func TestLoggingBusinessMetricsHook(t *testing.T) {
	hook := NewLoggingBusinessMetricsHook()

	assert.Equal(t, "logging_hook", hook.GetHookName())

	event := BusinessMetricEvent{
		Name:      "test_metric",
		Type:      MetricTypeCounter,
		Value:     1.0,
		Labels:    map[string]string{"test": "value"},
		Timestamp: time.Now(),
		Source:    "test-service",
	}

	// Should not panic
	assert.NotPanics(t, func() {
		hook.OnMetricRecorded(event)
	})
}

func TestAggregationBusinessMetricsHook(t *testing.T) {
	hook := NewAggregationBusinessMetricsHook()

	assert.Equal(t, "aggregation_hook", hook.GetHookName())

	t.Run("aggregate counters", func(t *testing.T) {
		event1 := BusinessMetricEvent{
			Name:   "test_counter",
			Type:   MetricTypeCounter,
			Value:  5.0,
			Source: "service1",
		}
		event2 := BusinessMetricEvent{
			Name:   "test_counter",
			Type:   MetricTypeCounter,
			Value:  3.0,
			Source: "service1",
		}

		hook.OnMetricRecorded(event1)
		hook.OnMetricRecorded(event2)

		totals := hook.GetCounterTotals()
		expected := "service1_test_counter"
		assert.Equal(t, 8.0, totals[expected])
	})

	t.Run("set gauges", func(t *testing.T) {
		event1 := BusinessMetricEvent{
			Name:   "test_gauge",
			Type:   MetricTypeGauge,
			Value:  42.0,
			Source: "service1",
		}
		event2 := BusinessMetricEvent{
			Name:   "test_gauge",
			Type:   MetricTypeGauge,
			Value:  84.0,
			Source: "service1",
		}

		hook.OnMetricRecorded(event1)
		hook.OnMetricRecorded(event2)

		values := hook.GetGaugeValues()
		expected := "service1_test_gauge"
		assert.Equal(t, 84.0, values[expected]) // Latest value for gauge
	})

	t.Run("reset aggregation", func(t *testing.T) {
		hook.Reset()

		assert.Empty(t, hook.GetCounterTotals())
		assert.Empty(t, hook.GetGaugeValues())
	})
}

// Helper types for testing

type testBusinessMetricsHook struct {
	name   string
	events []BusinessMetricEvent
	mu     sync.RWMutex
}

func (t *testBusinessMetricsHook) OnMetricRecorded(event BusinessMetricEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}

func (t *testBusinessMetricsHook) GetHookName() string {
	return t.name
}

// Benchmark tests

func BenchmarkBusinessMetricsManager_RecordCounter(b *testing.B) {
	bmm := NewBusinessMetricsManager("test-service", nil, nil)
	labels := map[string]string{"endpoint": "/api/test"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bmm.RecordCounter("api_requests_total", 1.0, labels)
		}
	})
}

func BenchmarkBusinessMetricsManager_RecordGauge(b *testing.B) {
	bmm := NewBusinessMetricsManager("test-service", nil, nil)
	labels := map[string]string{"queue": "processing"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bmm.RecordGauge("queue_size", float64(b.N), labels)
		}
	})
}

func BenchmarkBusinessMetricsManager_WithHooks(b *testing.B) {
	bmm := NewBusinessMetricsManager("test-service", nil, nil)

	// Add multiple hooks
	_ = bmm.RegisterHook(NewLoggingBusinessMetricsHook())
	_ = bmm.RegisterHook(NewAggregationBusinessMetricsHook())

	labels := map[string]string{"test": "benchmark"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bmm.RecordCounter("benchmark_counter", 1.0, labels)
		}
	})
}

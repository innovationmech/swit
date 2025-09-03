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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/innovationmech/swit/pkg/types"
)

// TestPrometheusMetricsEndToEnd tests the complete Prometheus metrics flow
func TestPrometheusMetricsEndToEnd(t *testing.T) {
	// Create Prometheus configuration
	prometheusConfig := &types.PrometheusConfig{
		Enabled:   true,
		Endpoint:  "/metrics",
		Namespace: "integration_test",
		Subsystem: "server",
		Buckets: types.PrometheusBuckets{
			Duration: []float64{0.01, 0.1, 1.0, 5.0},
			Size:     []float64{100, 1000, 10000},
		},
		Labels:           map[string]string{"env": "test"},
		CardinalityLimit: 1000,
	}

	// Create PrometheusMetricsCollector
	collector := types.NewPrometheusMetricsCollector(prometheusConfig)
	require.NotNil(t, collector)

	// Create metrics registry
	registry := NewMetricsRegistry()

	// Create business metrics manager
	bmm := NewBusinessMetricsManager("integration-test-service", collector, registry)

	t.Run("end-to-end metric collection and exposure", func(t *testing.T) {
		// Record various types of metrics
		bmm.RecordCounter("http_requests_total", 1.0, map[string]string{
			"method":   "GET",
			"endpoint": "/api/users",
			"status":   "200",
		})

		bmm.RecordCounter("http_requests_total", 1.0, map[string]string{
			"method":   "POST",
			"endpoint": "/api/users",
			"status":   "201",
		})

		bmm.RecordGauge("active_connections", 25.0, map[string]string{
			"server": "web-01",
		})

		bmm.RecordHistogram("http_request_duration_seconds", 0.15, map[string]string{
			"method":   "GET",
			"endpoint": "/api/users",
		})

		bmm.RecordHistogram("http_request_duration_seconds", 0.45, map[string]string{
			"method":   "POST",
			"endpoint": "/api/users",
		})

		bmm.RecordHistogram("response_size_bytes", 1024.0, map[string]string{
			"endpoint": "/api/users",
		})

		// Allow time for async operations
		time.Sleep(20 * time.Millisecond)

		// Verify metrics are in collector
		metrics := collector.GetMetrics()
		assert.NotEmpty(t, metrics)

		// Check for specific metrics
		var counterFound, gaugeFound, durationHistFound, sizeHistFound bool
		for _, metric := range metrics {
			switch {
			case strings.Contains(metric.Name, "http_requests_total"):
				counterFound = true
				assert.Equal(t, types.CounterType, metric.Type)
			case strings.Contains(metric.Name, "active_connections"):
				gaugeFound = true
				assert.Equal(t, types.GaugeType, metric.Type)
				assert.Equal(t, 25.0, metric.Value)
			case strings.Contains(metric.Name, "http_request_duration_seconds"):
				durationHistFound = true
				assert.Equal(t, types.HistogramType, metric.Type)
				histData := metric.Value.(map[string]interface{})
				assert.Contains(t, histData, "count")
				assert.Contains(t, histData, "sum")
			case strings.Contains(metric.Name, "response_size_bytes"):
				sizeHistFound = true
				assert.Equal(t, types.HistogramType, metric.Type)
			}
		}

		assert.True(t, counterFound, "Counter metrics should be found")
		assert.True(t, gaugeFound, "Gauge metrics should be found")
		assert.True(t, durationHistFound, "Duration histogram should be found")
		assert.True(t, sizeHistFound, "Size histogram should be found")

		// Test HTTP metrics endpoint
		handler := collector.GetHandler()
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()

		// Verify Prometheus format output
		assert.Contains(t, body, "integration_test_server_http_requests_total")
		assert.Contains(t, body, "integration_test_server_active_connections")
		assert.Contains(t, body, "integration_test_server_http_request_duration_seconds")
		assert.Contains(t, body, "integration_test_server_response_size_bytes")

		// Verify metric labels are present
		assert.Contains(t, body, `method="GET"`)
		assert.Contains(t, body, `endpoint="/api/users"`)
		assert.Contains(t, body, `status="200"`)
		assert.Contains(t, body, `server="web-01"`)
	})
}

// TestPrometheusMetricsHighLoad tests metrics collection under high load
func TestPrometheusMetricsHighLoad(t *testing.T) {
	config := &types.PrometheusConfig{
		Enabled:          true,
		Namespace:        "load_test",
		Subsystem:        "server",
		CardinalityLimit: 5000,
	}

	collector := types.NewPrometheusMetricsCollector(config)
	bmm := NewBusinessMetricsManager("load-test-service", collector, nil)

	const numGoroutines = 20
	const operationsPerGoroutine = 500
	const totalOperations = numGoroutines * operationsPerGoroutine

	t.Run("high load concurrent metrics", func(t *testing.T) {
		var wg sync.WaitGroup
		start := time.Now()

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					labels := map[string]string{
						"goroutine": fmt.Sprintf("g%d", goroutineID),
						"iteration": fmt.Sprintf("i%d", j),
					}

					// Mix different metric types
					switch j % 4 {
					case 0:
						bmm.RecordCounter("load_test_counter", 1.0, labels)
					case 1:
						bmm.RecordGauge("load_test_gauge", float64(j), labels)
					case 2:
						bmm.RecordHistogram("load_test_duration", float64(j)*0.001, labels)
					case 3:
						bmm.RecordHistogram("load_test_size", float64(j*100), labels)
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		// Performance requirements: should handle >10k operations per second
		opsPerSecond := float64(totalOperations) / duration.Seconds()
		t.Logf("Processed %d operations in %v (%.2f ops/sec)", totalOperations, duration, opsPerSecond)

		// Should be able to handle at least 5000 ops/sec under test conditions
		assert.GreaterOrEqual(t, opsPerSecond, 5000.0, "Should handle high load efficiently")

		// Verify metrics were recorded
		metrics := collector.GetMetrics()
		assert.NotEmpty(t, metrics)

		// Test that HTTP endpoint still works under load
		handler := collector.GetHandler()
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		metricsStart := time.Now()
		handler.ServeHTTP(w, req)
		metricsTime := time.Since(metricsStart)

		assert.Equal(t, http.StatusOK, w.Code)
		// Metrics endpoint should respond quickly even with many metrics
		assert.Less(t, metricsTime, 500*time.Millisecond, "Metrics endpoint should be fast")

		body := w.Body.String()
		assert.Contains(t, body, "load_test_server_load_test_counter")
		assert.Contains(t, body, "load_test_server_load_test_gauge")
		assert.Contains(t, body, "load_test_server_load_test_duration")
	})
}

// TestPrometheusMetricsMemoryUsage tests memory usage and cleanup
func TestPrometheusMetricsMemoryUsage(t *testing.T) {
	config := &types.PrometheusConfig{
		Enabled:          true,
		Namespace:        "memory_test",
		Subsystem:        "server",
		CardinalityLimit: 1000,
	}

	collector := types.NewPrometheusMetricsCollector(config)
	bmm := NewBusinessMetricsManager("memory-test-service", collector, nil)

	t.Run("memory cleanup after reset", func(t *testing.T) {
		// Add many metrics to build up memory usage
		for i := 0; i < 1000; i++ {
			labels := map[string]string{
				"id":     fmt.Sprintf("metric_%d", i),
				"shard":  fmt.Sprintf("shard_%d", i%10),
				"region": fmt.Sprintf("region_%d", i%5),
			}

			bmm.RecordCounter("memory_test_counter", 1.0, labels)
			bmm.RecordGauge("memory_test_gauge", float64(i), labels)
			bmm.RecordHistogram("memory_test_histogram", float64(i)*0.01, labels)
		}

		// Verify metrics exist
		metricsBefore := collector.GetMetrics()
		assert.NotEmpty(t, metricsBefore)
		assert.Greater(t, len(metricsBefore), 100)

		// Reset collector
		collector.Reset()

		// Verify cleanup
		metricsAfter := collector.GetMetrics()
		assert.Empty(t, metricsAfter)

		// Verify metrics endpoint is clean
		handler := collector.GetHandler()
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		// Should not contain our test metrics
		assert.NotContains(t, body, "memory_test_counter")
		assert.NotContains(t, body, "memory_test_gauge")
	})
}

// TestPrometheusMetricsCardinalityLimits tests cardinality limiting functionality
func TestPrometheusMetricsCardinalityLimits(t *testing.T) {
	config := &types.PrometheusConfig{
		Enabled:          true,
		Namespace:        "cardinality_test",
		Subsystem:        "server",
		CardinalityLimit: 100, // Low limit for testing
	}

	collector := types.NewPrometheusMetricsCollector(config)

	t.Run("cardinality limiting prevents metric explosion", func(t *testing.T) {
		// First create metrics up to the limit
		metricsUpToLimit := 95 // Leave some room for other metrics
		for i := 0; i < metricsUpToLimit; i++ {
			labels := map[string]string{
				"unique_id": fmt.Sprintf("id_%d", i),
				"category":  fmt.Sprintf("cat_%d", i%5),
			}
			collector.IncrementCounter("cardinality_test_metric", labels)
		}

		// Get metrics count after creating up to limit
		metricsAfterLimit := len(collector.GetMetrics())

		// Try to create more metrics beyond the limit
		for i := metricsUpToLimit; i < 150; i++ {
			labels := map[string]string{
				"unique_id": fmt.Sprintf("id_%d", i),
				"category":  fmt.Sprintf("cat_%d", i%5),
			}
			// These should be dropped due to cardinality limit
			collector.IncrementCounter("cardinality_test_metric", labels)
		}

		// Get final metric count - should not have increased significantly
		finalMetrics := len(collector.GetMetrics())

		// Should have created metrics up to the limit but not beyond
		assert.Greater(t, metricsAfterLimit, metricsUpToLimit-10, "Should create most metrics up to limit")
		assert.LessOrEqual(t, finalMetrics-metricsAfterLimit, 10, "Should not create many metrics beyond limit")

		t.Logf("Metrics after %d attempts: %d, after %d more attempts: %d", metricsUpToLimit, metricsAfterLimit, 150-metricsUpToLimit, finalMetrics)

		// Verify actual cardinality test metrics count
		metrics := collector.GetMetrics()
		cardinalityMetrics := 0
		for _, metric := range metrics {
			if strings.Contains(metric.Name, "cardinality_test_metric") {
				cardinalityMetrics++
			}
		}
		t.Logf("Found %d cardinality test metrics in collector", cardinalityMetrics)
		assert.Greater(t, cardinalityMetrics, 0, "Should have created some cardinality test metrics")
	})
}

// TestPrometheusMetricsWithBusinessHooks tests integration with business metrics hooks
func TestPrometheusMetricsWithBusinessHooks(t *testing.T) {
	collector := types.NewPrometheusMetricsCollector(nil)
	registry := NewMetricsRegistry()
	bmm := NewBusinessMetricsManager("hooks-test-service", collector, registry)

	// Add hooks for testing
	loggingHook := NewLoggingBusinessMetricsHook()
	aggregationHook := NewAggregationBusinessMetricsHook()

	err := bmm.RegisterHook(loggingHook)
	require.NoError(t, err)

	err = bmm.RegisterHook(aggregationHook)
	require.NoError(t, err)

	t.Run("hooks receive events from Prometheus collector", func(t *testing.T) {
		// Record various metrics
		testMetrics := []struct {
			name   string
			mType  types.MetricType
			value  float64
			labels map[string]string
		}{
			{
				name:   "hook_test_counter",
				mType:  types.CounterType,
				value:  5.0,
				labels: map[string]string{"endpoint": "/api/test"},
			},
			{
				name:   "hook_test_gauge",
				mType:  types.GaugeType,
				value:  42.0,
				labels: map[string]string{"server": "web-01"},
			},
			{
				name:   "hook_test_histogram",
				mType:  types.HistogramType,
				value:  0.123,
				labels: map[string]string{"method": "POST"},
			},
		}

		for _, tm := range testMetrics {
			switch tm.mType {
			case types.CounterType:
				bmm.RecordCounter(tm.name, tm.value, tm.labels)
			case types.GaugeType:
				bmm.RecordGauge(tm.name, tm.value, tm.labels)
			case types.HistogramType:
				bmm.RecordHistogram(tm.name, tm.value, tm.labels)
			}
		}

		// Allow time for async hook processing
		time.Sleep(50 * time.Millisecond)

		// Verify aggregation hook received events
		counterTotals := aggregationHook.GetCounterTotals()
		gaugeValues := aggregationHook.GetGaugeValues()

		assert.Contains(t, counterTotals, "hooks-test-service_hook_test_counter")
		assert.Equal(t, 5.0, counterTotals["hooks-test-service_hook_test_counter"])

		assert.Contains(t, gaugeValues, "hooks-test-service_hook_test_gauge")
		assert.Equal(t, 42.0, gaugeValues["hooks-test-service_hook_test_gauge"])

		// Verify metrics are also in Prometheus collector
		metrics := collector.GetMetrics()
		assert.NotEmpty(t, metrics)

		var foundCounter, foundGauge, foundHistogram bool
		for _, metric := range metrics {
			switch {
			case strings.Contains(metric.Name, "hook_test_counter"):
				foundCounter = true
			case strings.Contains(metric.Name, "hook_test_gauge"):
				foundGauge = true
			case strings.Contains(metric.Name, "hook_test_histogram"):
				foundHistogram = true
			}
		}

		assert.True(t, foundCounter, "Counter should be in Prometheus collector")
		assert.True(t, foundGauge, "Gauge should be in Prometheus collector")
		assert.True(t, foundHistogram, "Histogram should be in Prometheus collector")
	})
}

// TestPrometheusMetricsCustomRegistry tests custom metric definitions
func TestPrometheusMetricsCustomRegistry(t *testing.T) {
	collector := types.NewPrometheusMetricsCollector(nil)
	registry := NewMetricsRegistry()
	bmm := NewBusinessMetricsManager("custom-registry-test", collector, registry)

	t.Run("custom metric definitions integration", func(t *testing.T) {
		// Register custom metrics
		customMetrics := []MetricDefinition{
			{
				Name:        "custom_business_operations_total",
				Type:        types.CounterType,
				Description: "Total business operations processed",
				Labels:      []string{"operation_type", "department"},
			},
			{
				Name:        "custom_queue_depth",
				Type:        types.GaugeType,
				Description: "Current queue depth",
				Labels:      []string{"queue_name", "priority"},
			},
			{
				Name:        "custom_processing_duration_seconds",
				Type:        types.HistogramType,
				Description: "Processing time distribution",
				Labels:      []string{"process_type"},
				Buckets:     []float64{0.1, 0.5, 1.0, 2.0, 5.0},
			},
		}

		for _, metric := range customMetrics {
			err := bmm.RegisterCustomMetric(metric)
			require.NoError(t, err, "Should register custom metric %s", metric.Name)
		}

		// Verify custom metrics are registered
		allMetrics := bmm.GetAllMetrics()
		for _, metric := range customMetrics {
			def, exists := allMetrics[metric.Name]
			assert.True(t, exists, "Custom metric %s should be in registry", metric.Name)
			assert.Equal(t, metric.Description, def.Description)
			assert.Equal(t, metric.Labels, def.Labels)
		}

		// Record metrics using custom definitions
		bmm.RecordCounter("custom_business_operations_total", 1.0, map[string]string{
			"operation_type": "user_creation",
			"department":     "engineering",
		})

		bmm.RecordGauge("custom_queue_depth", 15.0, map[string]string{
			"queue_name": "processing",
			"priority":   "high",
		})

		bmm.RecordHistogram("custom_processing_duration_seconds", 0.75, map[string]string{
			"process_type": "data_validation",
		})

		// Verify metrics are collected
		collectedMetrics := collector.GetMetrics()
		assert.NotEmpty(t, collectedMetrics)

		var foundCustomCounter, foundCustomGauge, foundCustomHist bool
		for _, metric := range collectedMetrics {
			switch {
			case strings.Contains(metric.Name, "custom_business_operations_total"):
				foundCustomCounter = true
				assert.Equal(t, types.CounterType, metric.Type)
			case strings.Contains(metric.Name, "custom_queue_depth"):
				foundCustomGauge = true
				assert.Equal(t, types.GaugeType, metric.Type)
			case strings.Contains(metric.Name, "custom_processing_duration_seconds"):
				foundCustomHist = true
				assert.Equal(t, types.HistogramType, metric.Type)
			}
		}

		assert.True(t, foundCustomCounter, "Custom counter should be collected")
		assert.True(t, foundCustomGauge, "Custom gauge should be collected")
		assert.True(t, foundCustomHist, "Custom histogram should be collected")

		// Test HTTP endpoint includes custom metrics
		handler := collector.GetHandler()
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()

		assert.Contains(t, body, "custom_business_operations_total")
		assert.Contains(t, body, "custom_queue_depth")
		assert.Contains(t, body, "custom_processing_duration_seconds")
	})
}

// TestPrometheusMetricsServerIntegration tests integration with actual server
func TestPrometheusMetricsServerIntegration(t *testing.T) {
	// Create a simple HTTP server with Prometheus metrics
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create Prometheus collector
	collector := types.NewPrometheusMetricsCollector(&types.PrometheusConfig{
		Enabled:   true,
		Endpoint:  "/metrics",
		Namespace: "server_integration",
		Subsystem: "api",
	})

	bmm := NewBusinessMetricsManager("integration-server", collector, nil)

	// Add metrics middleware
	router.Use(func(c *gin.Context) {
		start := time.Now()

		// Record request counter with proper labels
		bmm.RecordCounter("http_requests_total", 1.0, map[string]string{
			"method": c.Request.Method,
			"path":   c.FullPath(),
		})

		bmm.RecordGauge("active_requests", 1.0, nil)
		defer func() {
			bmm.RecordGauge("active_requests", -1.0, nil)
		}()

		c.Next()

		// Record request duration after request completes
		duration := time.Since(start).Seconds()
		bmm.RecordHistogram("http_request_duration_seconds", duration, map[string]string{
			"method": c.Request.Method,
			"path":   c.FullPath(),
			"status": fmt.Sprintf("%d", c.Writer.Status()),
		})
	})

	// Add routes
	router.GET("/api/users", func(c *gin.Context) {
		time.Sleep(10 * time.Millisecond) // Simulate processing
		c.JSON(200, gin.H{"users": []string{"user1", "user2"}})
	})

	router.POST("/api/users", func(c *gin.Context) {
		time.Sleep(20 * time.Millisecond) // Simulate processing
		c.JSON(201, gin.H{"id": "123", "name": "new user"})
	})

	// Add metrics endpoint
	router.GET("/metrics", gin.WrapH(collector.GetHandler()))

	t.Run("server metrics collection", func(t *testing.T) {
		server := httptest.NewServer(router)
		defer server.Close()

		// Make requests to generate metrics
		requests := []struct {
			method string
			path   string
		}{
			{"GET", "/api/users"},
			{"GET", "/api/users"},
			{"POST", "/api/users"},
			{"GET", "/api/users"},
		}

		for _, req := range requests {
			requestURL, err := url.Parse(server.URL + req.path)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(&http.Request{
				Method: req.method,
				URL:    requestURL,
			})
			require.NoError(t, err)
			resp.Body.Close()
		}

		// Allow time for metrics processing (goroutines in BusinessMetricsManager)
		time.Sleep(100 * time.Millisecond)

		// Fetch metrics
		resp, err := http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Read entire response body
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		metricsBody := string(body)

		// Verify metrics are present (with proper prefixes)
		assert.Contains(t, metricsBody, "server_integration_api_http_requests_total")
		assert.Contains(t, metricsBody, "server_integration_api_http_request_duration_seconds")
		assert.Contains(t, metricsBody, "server_integration_api_active_requests")

		// Verify labels
		assert.Contains(t, metricsBody, `method="GET"`)
		assert.Contains(t, metricsBody, `method="POST"`)
		assert.Contains(t, metricsBody, `path="/api/users"`)
		assert.Contains(t, metricsBody, `service="integration-server"`)

		// Verify we have the expected number of requests
		getRequestCount := strings.Count(metricsBody, `method="GET",path="/api/users"`)
		postRequestCount := strings.Count(metricsBody, `method="POST",path="/api/users"`)
		assert.Greater(t, getRequestCount, 0, "Should have GET request metrics")
		assert.Greater(t, postRequestCount, 0, "Should have POST request metrics")

		t.Logf("Metrics endpoint response:\n%s", metricsBody)
	})
}

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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMetricsAPI tests the creation of MetricsAPI.
func TestNewMetricsAPI(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	api := NewMetricsAPI(collector, nil)
	assert.NotNil(t, api)
	assert.NotNil(t, api.collector)
}

// TestMetricsAPI_GetMetrics tests the GetMetrics endpoint.
func TestMetricsAPI_GetMetrics(t *testing.T) {
	tests := []struct {
		name               string
		setupMetrics       func(*SagaMetricsCollector)
		queryParams        string
		expectedStatusCode int
		checkResponse      func(*testing.T, *MetricsResponse)
	}{
		{
			name: "basic metrics retrieval",
			setupMetrics: func(collector *SagaMetricsCollector) {
				collector.RecordSagaStarted("saga-1")
				collector.RecordSagaStarted("saga-2")
				collector.RecordSagaCompleted("saga-1", 1500*time.Millisecond)
			},
			queryParams:        "",
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *MetricsResponse) {
				assert.Equal(t, int64(2), resp.Summary.Total)
				assert.Equal(t, int64(1), resp.Summary.Active)
				assert.Equal(t, int64(1), resp.Summary.Completed)
				assert.Equal(t, int64(0), resp.Summary.Failed)
				assert.InDelta(t, 50.0, resp.Summary.SuccessRate, 0.1)
			},
		},
		{
			name: "metrics with failures",
			setupMetrics: func(collector *SagaMetricsCollector) {
				collector.RecordSagaStarted("saga-1")
				collector.RecordSagaStarted("saga-2")
				collector.RecordSagaStarted("saga-3")
				collector.RecordSagaCompleted("saga-1", 1000*time.Millisecond)
				collector.RecordSagaFailed("saga-2", "timeout")
			},
			queryParams:        "",
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *MetricsResponse) {
				assert.Equal(t, int64(3), resp.Summary.Total)
				assert.Equal(t, int64(1), resp.Summary.Active)
				assert.Equal(t, int64(1), resp.Summary.Completed)
				assert.Equal(t, int64(1), resp.Summary.Failed)
				assert.InDelta(t, 33.3, resp.Summary.SuccessRate, 0.5)
				assert.InDelta(t, 33.3, resp.Summary.FailureRate, 0.5)
				assert.Contains(t, resp.Summary.FailureReasons, "timeout")
			},
		},
		{
			name: "metrics with groupBy parameter",
			setupMetrics: func(collector *SagaMetricsCollector) {
				collector.RecordSagaStarted("saga-1")
				collector.RecordSagaCompleted("saga-1", 500*time.Millisecond)
			},
			queryParams:        "?groupBy=state",
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *MetricsResponse) {
				// Aggregations should be present when groupBy is specified
				assert.NotNil(t, resp.Aggregations)
			},
		},
		{
			name:               "invalid groupBy parameter",
			setupMetrics:       func(collector *SagaMetricsCollector) {},
			queryParams:        "?groupBy=invalid",
			expectedStatusCode: http.StatusBadRequest,
			checkResponse:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			gin.SetMode(gin.TestMode)
			registry := prometheus.NewRegistry()
			config := &Config{
				Registry: registry,
			}
			collector, err := NewSagaMetricsCollector(config)
			require.NoError(t, err)

			// Setup metrics
			if tt.setupMetrics != nil {
				tt.setupMetrics(collector)
			}

			api := NewMetricsAPI(collector, nil)

			// Create test request
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest("GET", "/api/metrics"+tt.queryParams, nil)
			c.Request = req

			// Call handler
			api.GetMetrics(c)

			// Check status code
			assert.Equal(t, tt.expectedStatusCode, w.Code)

			// Check response
			if tt.checkResponse != nil && w.Code == http.StatusOK {
				var resp MetricsResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				tt.checkResponse(t, &resp)
			}
		})
	}
}

// TestMetricsAPI_GetRealtimeMetrics tests the GetRealtimeMetrics endpoint.
func TestMetricsAPI_GetRealtimeMetrics(t *testing.T) {
	tests := []struct {
		name               string
		setupMetrics       func(*SagaMetricsCollector)
		expectedStatusCode int
		checkResponse      func(*testing.T, map[string]interface{})
	}{
		{
			name: "basic realtime metrics",
			setupMetrics: func(collector *SagaMetricsCollector) {
				collector.RecordSagaStarted("saga-1")
				collector.RecordSagaStarted("saga-2")
				collector.RecordSagaCompleted("saga-1", 2000*time.Millisecond)
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, float64(2), resp["total"])
				assert.Equal(t, float64(1), resp["active"])
				assert.Equal(t, float64(1), resp["completed"])
				assert.Equal(t, float64(0), resp["failed"])
				assert.Contains(t, resp, "successRate")
				assert.Contains(t, resp, "avgDuration")
				assert.Contains(t, resp, "throughput")
				assert.Contains(t, resp, "timestamp")
			},
		},
		{
			name: "realtime metrics with no sagas",
			setupMetrics: func(collector *SagaMetricsCollector) {
				// No sagas recorded
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, float64(0), resp["total"])
				assert.Equal(t, float64(0), resp["active"])
				assert.Equal(t, float64(0), resp["completed"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			gin.SetMode(gin.TestMode)
			registry := prometheus.NewRegistry()
			config := &Config{
				Registry: registry,
			}
			collector, err := NewSagaMetricsCollector(config)
			require.NoError(t, err)

			// Setup metrics
			if tt.setupMetrics != nil {
				tt.setupMetrics(collector)
			}

			api := NewMetricsAPI(collector, nil)

			// Create test request
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest("GET", "/api/metrics/realtime", nil)
			c.Request = req

			// Call handler
			api.GetRealtimeMetrics(c)

			// Check status code
			assert.Equal(t, tt.expectedStatusCode, w.Code)

			// Check response
			if tt.checkResponse != nil {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				tt.checkResponse(t, resp)
			}
		})
	}
}

// TestCalculateMetricsSummary tests the calculateMetricsSummary function.
func TestCalculateMetricsSummary(t *testing.T) {
	tests := []struct {
		name     string
		metrics  *Metrics
		expected MetricsSummary
	}{
		{
			name: "all completed successfully",
			metrics: &Metrics{
				SagasStarted:   10,
				SagasCompleted: 10,
				SagasFailed:    0,
				ActiveSagas:    0,
				TotalDuration:  25.0,
				AvgDuration:    2.5,
				FailureReasons: map[string]int64{},
			},
			expected: MetricsSummary{
				Total:          10,
				Active:         0,
				Completed:      10,
				Failed:         0,
				SuccessRate:    100.0,
				FailureRate:    0.0,
				AvgDuration:    2.5,
				TotalDuration:  25.0,
				FailureReasons: map[string]int64{},
			},
		},
		{
			name: "mixed results",
			metrics: &Metrics{
				SagasStarted:   100,
				SagasCompleted: 85,
				SagasFailed:    10,
				ActiveSagas:    5,
				TotalDuration:  212.5,
				AvgDuration:    2.5,
				FailureReasons: map[string]int64{
					"timeout": 7,
					"error":   3,
				},
			},
			expected: MetricsSummary{
				Total:         100,
				Active:        5,
				Completed:     85,
				Failed:        10,
				SuccessRate:   85.0,
				FailureRate:   10.0,
				AvgDuration:   2.5,
				TotalDuration: 212.5,
				FailureReasons: map[string]int64{
					"timeout": 7,
					"error":   3,
				},
			},
		},
		{
			name: "no sagas",
			metrics: &Metrics{
				SagasStarted:   0,
				SagasCompleted: 0,
				SagasFailed:    0,
				ActiveSagas:    0,
				TotalDuration:  0,
				AvgDuration:    0,
				FailureReasons: map[string]int64{},
			},
			expected: MetricsSummary{
				Total:          0,
				Active:         0,
				Completed:      0,
				Failed:         0,
				SuccessRate:    0.0,
				FailureRate:    0.0,
				AvgDuration:    0,
				TotalDuration:  0,
				FailureReasons: map[string]int64{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateMetricsSummary(tt.metrics)

			assert.Equal(t, tt.expected.Total, result.Total)
			assert.Equal(t, tt.expected.Active, result.Active)
			assert.Equal(t, tt.expected.Completed, result.Completed)
			assert.Equal(t, tt.expected.Failed, result.Failed)
			assert.InDelta(t, tt.expected.SuccessRate, result.SuccessRate, 0.01)
			assert.InDelta(t, tt.expected.FailureRate, result.FailureRate, 0.01)
			assert.InDelta(t, tt.expected.AvgDuration, result.AvgDuration, 0.01)
			assert.InDelta(t, tt.expected.TotalDuration, result.TotalDuration, 0.01)
			assert.Equal(t, tt.expected.FailureReasons, result.FailureReasons)
		})
	}
}

// TestMetricsAPI_Integration tests the full integration with collector.
func TestMetricsAPI_Integration(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(config)
	require.NoError(t, err)

	api := NewMetricsAPI(collector, nil)

	// Simulate saga lifecycle
	for i := 0; i < 10; i++ {
		sagaID := "saga-" + string(rune(i))
		collector.RecordSagaStarted(sagaID)

		if i < 8 {
			// Complete 8 sagas
			collector.RecordSagaCompleted(sagaID, time.Duration(i+1)*time.Second)
		} else if i == 8 {
			// Fail 1 saga
			collector.RecordSagaFailed(sagaID, "timeout")
		}
		// Leave 1 saga active
	}

	// Test GetMetrics
	t.Run("GetMetrics integration", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/metrics", nil)
		c.Request = req

		api.GetMetrics(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp MetricsResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, int64(10), resp.Summary.Total)
		assert.Equal(t, int64(1), resp.Summary.Active)
		assert.Equal(t, int64(8), resp.Summary.Completed)
		assert.Equal(t, int64(1), resp.Summary.Failed)
		assert.InDelta(t, 80.0, resp.Summary.SuccessRate, 0.1)
		assert.InDelta(t, 10.0, resp.Summary.FailureRate, 0.1)
		assert.NotZero(t, resp.Timestamp)
	})

	// Test GetRealtimeMetrics
	t.Run("GetRealtimeMetrics integration", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/metrics/realtime", nil)
		c.Request = req

		api.GetRealtimeMetrics(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, float64(10), resp["total"])
		assert.Equal(t, float64(1), resp["active"])
		assert.Equal(t, float64(8), resp["completed"])
		assert.Equal(t, float64(1), resp["failed"])
	})
}

// BenchmarkMetricsAPI_GetMetrics benchmarks the GetMetrics endpoint.
func BenchmarkMetricsAPI_GetMetrics(b *testing.B) {
	// Setup
	gin.SetMode(gin.ReleaseMode)
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(config)
	require.NoError(b, err)

	// Populate with some metrics
	for i := 0; i < 100; i++ {
		collector.RecordSagaStarted("saga")
		collector.RecordSagaCompleted("saga", 1*time.Second)
	}

	api := NewMetricsAPI(collector, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/metrics", nil)
		c.Request = req

		api.GetMetrics(c)
	}
}

// BenchmarkMetricsAPI_GetRealtimeMetrics benchmarks the GetRealtimeMetrics endpoint.
func BenchmarkMetricsAPI_GetRealtimeMetrics(b *testing.B) {
	// Setup
	gin.SetMode(gin.ReleaseMode)
	registry := prometheus.NewRegistry()
	config := &Config{
		Registry: registry,
	}
	collector, err := NewSagaMetricsCollector(config)
	require.NoError(b, err)

	// Populate with some metrics
	for i := 0; i < 100; i++ {
		collector.RecordSagaStarted("saga")
		collector.RecordSagaCompleted("saga", 1*time.Second)
	}

	api := NewMetricsAPI(collector, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/metrics/realtime", nil)
		c.Request = req

		api.GetRealtimeMetrics(c)
	}
}

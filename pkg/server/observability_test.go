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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/transport"
	"github.com/innovationmech/swit/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewSimpleMetricsCollector(t *testing.T) {
	collector := NewSimpleMetricsCollector()

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.metrics)
	assert.Empty(t, collector.metrics)
}

func TestSimpleMetricsCollector_Counter(t *testing.T) {
	collector := NewSimpleMetricsCollector()

	t.Run("increment counter", func(t *testing.T) {
		labels := map[string]string{"service": "test"}

		collector.IncrementCounter("test_counter", labels)
		collector.IncrementCounter("test_counter", labels)

		metrics := collector.GetMetrics()
		require.Len(t, metrics, 1)

		metric := metrics[0]
		assert.Equal(t, "test_counter", metric.Name)
		assert.Equal(t, MetricTypeCounter, metric.Type)
		assert.Equal(t, int64(2), metric.Value)
		assert.Equal(t, labels, metric.Labels)
	})

	t.Run("add to counter", func(t *testing.T) {
		labels := map[string]string{"service": "test"}

		collector.AddToCounter("add_counter", 5.0, labels)
		collector.AddToCounter("add_counter", 3.0, labels)

		metric, exists := collector.GetMetric("add_counter")
		require.True(t, exists)
		assert.Equal(t, int64(8), metric.Value)
	})
}

func TestSimpleMetricsCollector_Gauge(t *testing.T) {
	collector := NewSimpleMetricsCollector()

	t.Run("set gauge", func(t *testing.T) {
		labels := map[string]string{"service": "test"}

		collector.SetGauge("test_gauge", 42.5, labels)

		metric, exists := collector.GetMetric("test_gauge")
		require.True(t, exists)
		assert.Equal(t, MetricTypeGauge, metric.Type)
		assert.Equal(t, 42.5, metric.Value)
	})

	t.Run("increment gauge", func(t *testing.T) {
		labels := map[string]string{"service": "test"}

		collector.SetGauge("inc_gauge", 10.0, labels)
		collector.IncrementGauge("inc_gauge", labels)
		collector.IncrementGauge("inc_gauge", labels)

		metric, exists := collector.GetMetric("inc_gauge")
		require.True(t, exists)
		assert.Equal(t, 12.0, metric.Value)
	})

	t.Run("decrement gauge", func(t *testing.T) {
		labels := map[string]string{"service": "test"}

		collector.SetGauge("dec_gauge", 10.0, labels)
		collector.DecrementGauge("dec_gauge", labels)

		metric, exists := collector.GetMetric("dec_gauge")
		require.True(t, exists)
		assert.Equal(t, 9.0, metric.Value)
	})
}

func TestSimpleMetricsCollector_Histogram(t *testing.T) {
	collector := NewSimpleMetricsCollector()
	labels := map[string]string{"service": "test"}

	collector.ObserveHistogram("test_histogram", 1.0, labels)
	collector.ObserveHistogram("test_histogram", 2.0, labels)
	collector.ObserveHistogram("test_histogram", 3.0, labels)

	metric, exists := collector.GetMetric("test_histogram")
	require.True(t, exists)
	assert.Equal(t, MetricTypeHistogram, metric.Type)

	value, ok := metric.Value.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 3, value["count"])
	assert.Equal(t, 6.0, value["sum"])
	assert.Equal(t, 2.0, value["avg"])
}

func TestSimpleMetricsCollector_Reset(t *testing.T) {
	collector := NewSimpleMetricsCollector()

	collector.IncrementCounter("test_counter", nil)
	collector.SetGauge("test_gauge", 42.0, nil)

	assert.Len(t, collector.GetMetrics(), 2)

	collector.Reset()

	assert.Empty(t, collector.GetMetrics())
}

func TestSimpleMetricsCollector_GetMetricKey(t *testing.T) {
	collector := NewSimpleMetricsCollector()

	t.Run("no labels", func(t *testing.T) {
		key := collector.getMetricKey("test_metric", nil)
		assert.Equal(t, "test_metric", key)
	})

	t.Run("with labels", func(t *testing.T) {
		labels := map[string]string{
			"service": "test",
			"env":     "dev",
		}
		key := collector.getMetricKey("test_metric", labels)
		// Key should contain metric name and all label key-value pairs
		assert.Contains(t, key, "test_metric")
		assert.Contains(t, key, "service_test")
		assert.Contains(t, key, "env_dev")
	})
}

func TestNewServerMetrics(t *testing.T) {
	t.Run("with custom collector", func(t *testing.T) {
		collector := NewSimpleMetricsCollector()
		metrics := NewServerMetrics("test-service", collector)

		assert.NotNil(t, metrics)
		assert.Equal(t, "test-service", metrics.serviceName)
		assert.Same(t, collector, metrics.collector)
	})

	t.Run("with nil collector creates default", func(t *testing.T) {
		metrics := NewServerMetrics("test-service", nil)

		assert.NotNil(t, metrics)
		assert.NotNil(t, metrics.collector)
	})
}

func TestServerMetrics_RecordMethods(t *testing.T) {
	t.Run("RecordServerStart", func(t *testing.T) {
		collector := NewSimpleMetricsCollector()
		metrics := NewServerMetrics("test-service", collector)

		metrics.RecordServerStart()

		allMetrics := collector.GetMetrics()
		assert.Len(t, allMetrics, 2) // counter and gauge

		// Check counter
		counter, exists := collector.GetMetric("server_starts_total")
		require.True(t, exists)
		assert.Equal(t, int64(1), counter.Value)

		// Check gauge
		gauge, exists := collector.GetMetric("server_start_time")
		require.True(t, exists)
		assert.IsType(t, float64(0), gauge.Value)
	})

	t.Run("RecordServerStop", func(t *testing.T) {
		collector := NewSimpleMetricsCollector()
		metrics := NewServerMetrics("test-service", collector)

		metrics.RecordServerStop()

		counter, exists := collector.GetMetric("server_stops_total")
		require.True(t, exists)
		assert.Equal(t, int64(1), counter.Value)

		histogram, exists := collector.GetMetric("server_uptime_seconds")
		require.True(t, exists)
		assert.Equal(t, MetricTypeHistogram, histogram.Type)
	})

	t.Run("RecordTransportStart", func(t *testing.T) {
		collector := NewSimpleMetricsCollector()
		metrics := NewServerMetrics("test-service", collector)

		metrics.RecordTransportStart("http")

		// Check all metrics to find the ones with correct labels
		allMetrics := collector.GetMetrics()

		// Find transport_starts_total metric
		var startsCounter *Metric
		for _, metric := range allMetrics {
			if metric.Name == "transport_starts_total" {
				startsCounter = &metric
				break
			}
		}
		require.NotNil(t, startsCounter)
		assert.Equal(t, int64(1), startsCounter.Value)

		// Find active_transports gauge
		var activeGauge *Metric
		for _, metric := range allMetrics {
			if metric.Name == "active_transports" {
				activeGauge = &metric
				break
			}
		}
		require.NotNil(t, activeGauge)
		assert.Equal(t, 1.0, activeGauge.Value)
	})

	t.Run("RecordTransportStop", func(t *testing.T) {
		collector := NewSimpleMetricsCollector()
		metrics := NewServerMetrics("test-service", collector)

		// First start a transport to have something to stop
		metrics.RecordTransportStart("http")
		// Then stop it
		metrics.RecordTransportStop("http")

		// Check all metrics to find the ones with correct labels
		allMetrics := collector.GetMetrics()

		// Find transport_stops_total metric
		var stopsCounter *Metric
		for _, metric := range allMetrics {
			if metric.Name == "transport_stops_total" {
				stopsCounter = &metric
				break
			}
		}
		require.NotNil(t, stopsCounter)
		assert.Equal(t, int64(1), stopsCounter.Value)

		// Find active_transports gauge with the correct labels
		var activeGauge *Metric
		for _, metric := range allMetrics {
			if metric.Name == "active_transports" &&
				metric.Labels["service"] == "test-service" &&
				metric.Labels["transport"] == "http" {
				activeGauge = &metric
				break
			}
		}
		require.NotNil(t, activeGauge, "Should find active_transports metric with correct labels")
		assert.Equal(t, 0.0, activeGauge.Value) // Should be 0 after start then stop
	})

	t.Run("RecordServiceRegistration", func(t *testing.T) {
		collector := NewSimpleMetricsCollector()
		metrics := NewServerMetrics("test-service", collector)

		metrics.RecordServiceRegistration("auth-service", "http")

		counter, exists := collector.GetMetric("service_registrations_total")
		require.True(t, exists)
		assert.Equal(t, int64(1), counter.Value)

		gauge, exists := collector.GetMetric("registered_services")
		require.True(t, exists)
		assert.Equal(t, 1.0, gauge.Value)
	})

	t.Run("RecordError", func(t *testing.T) {
		collector := NewSimpleMetricsCollector()
		metrics := NewServerMetrics("test-service", collector)

		metrics.RecordError("config_error", "startup")

		counter, exists := collector.GetMetric("errors_total")
		require.True(t, exists)
		assert.Equal(t, int64(1), counter.Value)
	})

	t.Run("RecordOperationDuration", func(t *testing.T) {
		collector := NewSimpleMetricsCollector()
		metrics := NewServerMetrics("test-service", collector)

		duration := 100 * time.Millisecond
		metrics.RecordOperationDuration("startup", duration)

		histogram, exists := collector.GetMetric("operation_duration_seconds")
		require.True(t, exists)
		assert.Equal(t, MetricTypeHistogram, histogram.Type)
	})
}

func TestNewObservabilityManager(t *testing.T) {
	collector := NewSimpleMetricsCollector()
	manager := NewObservabilityManager("test-service", collector)

	assert.NotNil(t, manager)
	assert.Equal(t, "test-service", manager.serviceName)
	assert.NotNil(t, manager.metrics)
	assert.NotNil(t, manager.buildInfo)
}

func TestObservabilityManager_SetMethods(t *testing.T) {
	manager := NewObservabilityManager("test-service", nil)

	t.Run("SetVersion", func(t *testing.T) {
		manager.SetVersion("1.0.0")
		assert.Equal(t, "1.0.0", manager.version)
	})

	t.Run("SetBuildInfo", func(t *testing.T) {
		buildInfo := map[string]string{
			"commit": "abc123",
			"date":   "2025-01-01",
		}
		manager.SetBuildInfo(buildInfo)
		assert.Equal(t, buildInfo, manager.buildInfo)
	})
}

// mockNetworkTransport implements NetworkTransport for testing
type mockNetworkTransport struct {
	name    string
	address string
}

func (m *mockNetworkTransport) Start(ctx context.Context) error {
	return nil
}

func (m *mockNetworkTransport) Stop(ctx context.Context) error {
	return nil
}

func (m *mockNetworkTransport) GetName() string {
	return m.name
}

func (m *mockNetworkTransport) GetAddress() string {
	return m.address
}

// MockBusinessServer for testing
type MockBusinessServer struct {
	mock.Mock
}

// Implement BusinessServerCore interface

func (m *MockBusinessServer) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockBusinessServer) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockBusinessServer) Shutdown() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockBusinessServer) GetHTTPAddress() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockBusinessServer) GetGRPCAddress() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockBusinessServer) GetTransports() []transport.NetworkTransport {
	args := m.Called()
	return args.Get(0).([]transport.NetworkTransport)
}

func (m *MockBusinessServer) GetTransportStatus() map[string]TransportStatus {
	args := m.Called()
	return args.Get(0).(map[string]TransportStatus)
}

func (m *MockBusinessServer) GetTransportHealth(ctx context.Context) map[string]map[string]*types.HealthStatus {
	args := m.Called(ctx)
	return args.Get(0).(map[string]map[string]*types.HealthStatus)
}

func (m *MockBusinessServer) GetPerformanceMetrics() *PerformanceMetrics {
	args := m.Called()
	return args.Get(0).(*PerformanceMetrics)
}

func (m *MockBusinessServer) GetUptime() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func (m *MockBusinessServer) GetPerformanceMonitor() *PerformanceMonitor {
	args := m.Called()
	return args.Get(0).(*PerformanceMonitor)
}

func TestObservabilityManager_GetServerStatus(t *testing.T) {
	manager := NewObservabilityManager("test-service", NewSimpleMetricsCollector())
	manager.SetVersion("1.0.0")
	manager.SetBuildInfo(map[string]string{"commit": "abc123"})

	mockServer := new(MockBusinessServer)

	// Mock transport status
	transportStatus := map[string]TransportStatus{
		"http": {
			Name:    "http",
			Address: "localhost:8080",
			Running: true,
		},
	}
	mockServer.On("GetTransportStatus").Return(transportStatus)

	// Mock health checks
	healthChecks := map[string]map[string]*types.HealthStatus{
		"http": {
			"test-service": {
				Status:    types.HealthStatusHealthy,
				Timestamp: time.Now(),
				Version:   "1.0.0",
			},
		},
	}
	mockServer.On("GetTransportHealth", mock.Anything).Return(healthChecks)

	status := manager.GetServerStatus(mockServer)

	assert.Equal(t, "test-service", status.ServiceName)
	assert.Equal(t, "running", status.Status)
	assert.Equal(t, "1.0.0", status.Version)
	assert.Equal(t, map[string]string{"commit": "abc123"}, status.BuildInfo)
	assert.Len(t, status.Transports, 1)
	assert.Equal(t, "http", status.Transports[0].Name)
	assert.NotNil(t, status.HealthChecks)
	assert.Contains(t, status.HealthChecks, "http_test-service")
	assert.NotEmpty(t, status.SystemInfo.GoVersion)
	assert.Greater(t, status.SystemInfo.NumCPU, 0)

	mockServer.AssertExpectations(t)
}

func TestObservabilityManager_RegisterDebugEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	manager := NewObservabilityManager("test-service", nil)
	manager.SetVersion("1.0.0")

	mockServer := new(MockBusinessServer)

	// Setup mock expectations for all endpoints
	transportStatus := map[string]TransportStatus{
		"http": {Name: "http", Address: "localhost:8080", Running: true},
	}
	healthChecks := map[string]map[string]*types.HealthStatus{
		"http": {
			"test-service": {
				Status:    types.HealthStatusHealthy,
				Timestamp: time.Now(),
				Version:   "1.0.0",
			},
		},
	}

	mockServer.On("GetTransportStatus").Return(transportStatus)
	mockServer.On("GetTransportHealth", mock.Anything).Return(healthChecks)

	manager.RegisterDebugEndpoints(router, mockServer)

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
	}{
		{"status endpoint", "/debug/status", http.StatusOK},
		{"metrics endpoint", "/debug/metrics", http.StatusOK},
		{"health endpoint", "/debug/health", http.StatusOK},
		{"config endpoint", "/debug/config", http.StatusOK},
		{"system endpoint", "/debug/system", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify response is valid JSON
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.NotEmpty(t, response)
		})
	}
}

func TestObservabilityManager_RegisterHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	manager := NewObservabilityManager("test-service", nil)
	manager.SetVersion("1.0.0")

	t.Run("healthy server", func(t *testing.T) {
		mockServer := new(MockBusinessServer)

		healthChecks := map[string]map[string]*types.HealthStatus{
			"http": {
				"test-service": {
					Status:    types.HealthStatusHealthy,
					Timestamp: time.Now(),
					Version:   "1.0.0",
				},
			},
		}
		mockServer.On("GetTransportHealth", mock.Anything).Return(healthChecks)

		manager.RegisterHealthEndpoint(router, mockServer)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response types.HealthStatus
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, types.HealthStatusHealthy, response.Status)
		assert.Equal(t, "1.0.0", response.Version)
		assert.NotNil(t, response.Dependencies)

		mockServer.AssertExpectations(t)
	})

	t.Run("unhealthy server", func(t *testing.T) {
		router := gin.New() // Create new router to avoid conflicts
		mockServer := new(MockBusinessServer)

		healthChecks := map[string]map[string]*types.HealthStatus{
			"http": {
				"test-service": {
					Status:    types.HealthStatusUnhealthy,
					Timestamp: time.Now(),
					Version:   "1.0.0",
				},
			},
		}
		mockServer.On("GetTransportHealth", mock.Anything).Return(healthChecks)

		manager.RegisterHealthEndpoint(router, mockServer)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response types.HealthStatus
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, types.HealthStatusUnhealthy, response.Status)

		mockServer.AssertExpectations(t)
	})

	t.Run("nil server", func(t *testing.T) {
		router := gin.New() // Create new router to avoid conflicts
		manager.RegisterHealthEndpoint(router, nil)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response types.HealthStatus
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, types.HealthStatusUnhealthy, response.Status)
		assert.Contains(t, response.Dependencies, "server")
	})
}

func TestServerMetrics_GetCollector(t *testing.T) {
	collector := NewSimpleMetricsCollector()
	metrics := NewServerMetrics("test-service", collector)

	assert.Same(t, collector, metrics.GetCollector())
}

func TestMetricTypes(t *testing.T) {
	assert.Equal(t, MetricType("counter"), MetricTypeCounter)
	assert.Equal(t, MetricType("gauge"), MetricTypeGauge)
	assert.Equal(t, MetricType("histogram"), MetricTypeHistogram)
	assert.Equal(t, MetricType("summary"), MetricTypeSummary)
}

func TestSystemInfo(t *testing.T) {
	manager := NewObservabilityManager("test-service", nil)
	status := manager.GetServerStatus(nil)

	assert.NotEmpty(t, status.SystemInfo.GoVersion)
	assert.Greater(t, status.SystemInfo.NumCPU, 0)
	assert.GreaterOrEqual(t, status.SystemInfo.NumGoroutine, 1)
	assert.GreaterOrEqual(t, status.SystemInfo.MemStats.Alloc, uint64(0))
	assert.GreaterOrEqual(t, status.SystemInfo.MemStats.TotalAlloc, uint64(0))
	assert.GreaterOrEqual(t, status.SystemInfo.MemStats.Sys, uint64(0))
	assert.GreaterOrEqual(t, status.SystemInfo.MemStats.NumGC, uint32(0))
}

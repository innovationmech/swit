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
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhase3Integration_ServerWithPrometheusMetrics(t *testing.T) {
	// Create test configuration
	config := &ServerConfig{
		ServiceName: "test-phase3-service",
		HTTP: HTTPConfig{
			Port:         "0", // Dynamic port allocation
			Enabled:      true,
			TestMode:     true,
			EnableReady:  true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: GRPCConfig{
			Port:    "0",   // Dynamic port allocation
			Enabled: false, // Disable gRPC for this test
		},
		Discovery: DiscoveryConfig{
			Enabled: false, // Disable for testing
		},
		ShutdownTimeout: 5 * time.Second,
	}

	// Create test service registrar
	registrar := &Phase3TestServiceRegistrar{}

	// Create server
	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err)
	require.NotNil(t, server)

	// Verify observability manager is initialized
	observabilityManager := server.GetObservabilityManager()
	assert.NotNil(t, observabilityManager)

	// Verify Prometheus collector is available
	prometheusCollector := server.GetPrometheusCollector()
	assert.NotNil(t, prometheusCollector)

	// Verify business metrics manager is available
	businessMetrics := server.GetBusinessMetricsManager()
	assert.NotNil(t, businessMetrics)

	t.Run("server startup records metrics", func(t *testing.T) {
		// Start server
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := server.Start(ctx)
		require.NoError(t, err)

		// Give the server a moment to record metrics
		time.Sleep(100 * time.Millisecond)

		// Verify HTTP address is available
		httpAddr := server.GetHTTPAddress()
		assert.NotEmpty(t, httpAddr)

		// Test Prometheus metrics endpoint
		resp, err := http.Get(fmt.Sprintf("http://%s/metrics", httpAddr))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")

		// Test debug endpoints
		debugResp, err := http.Get(fmt.Sprintf("http://%s/debug/status", httpAddr))
		require.NoError(t, err)
		defer debugResp.Body.Close()

		assert.Equal(t, http.StatusOK, debugResp.StatusCode)
		assert.Contains(t, debugResp.Header.Get("Content-Type"), "application/json")

		// Test health endpoint
		healthResp, err := http.Get(fmt.Sprintf("http://%s/health", httpAddr))
		require.NoError(t, err)
		defer healthResp.Body.Close()

		assert.Equal(t, http.StatusOK, healthResp.StatusCode)

		// Shutdown server
		err = server.Shutdown()
		assert.NoError(t, err)
	})

	t.Run("business metrics recording", func(t *testing.T) {
		// Test business metrics functionality
		hook := NewAggregationBusinessMetricsHook()
		err := businessMetrics.RegisterHook(hook)
		require.NoError(t, err)

		// Record some business metrics
		businessMetrics.RecordCounter("test_api_calls", 5, map[string]string{"endpoint": "/test"})
		businessMetrics.RecordGauge("test_active_users", 42, map[string]string{"region": "us-west"})
		businessMetrics.RecordHistogram("test_request_duration", 0.123, map[string]string{"method": "GET"})

		// Allow hooks to process
		time.Sleep(50 * time.Millisecond)

		// Verify metrics were recorded
		collector := businessMetrics.GetCollector()
		metrics := collector.GetMetrics()
		assert.NotEmpty(t, metrics)

		// Check aggregation hook received events
		counterTotals := hook.GetCounterTotals()
		gaugeValues := hook.GetGaugeValues()

		assert.Contains(t, counterTotals, "test-phase3-service_test_api_calls")
		assert.Equal(t, 5.0, counterTotals["test-phase3-service_test_api_calls"])

		assert.Contains(t, gaugeValues, "test-phase3-service_test_active_users")
		assert.Equal(t, 42.0, gaugeValues["test-phase3-service_test_active_users"])
	})

	t.Run("system metrics collection", func(t *testing.T) {
		// Trigger system metrics update
		observabilityManager.UpdateSystemMetrics()

		// Get Prometheus collector metrics
		collector := observabilityManager.GetPrometheusCollector()
		metrics := collector.GetMetrics()

		// Verify system metrics are present
		var foundMemory, foundGoroutines, foundUptime bool

		for _, metric := range metrics {
			switch metric.Name {
			case "swit_server_memory_bytes":
				foundMemory = true
				assert.Equal(t, MetricTypeGauge, metric.Type)
				assert.Contains(t, metric.Labels, "service")
				assert.Contains(t, metric.Labels, "type")
			case "swit_server_goroutines":
				foundGoroutines = true
				assert.Equal(t, MetricTypeGauge, metric.Type)
				assert.Contains(t, metric.Labels, "service")
			case "swit_server_uptime_seconds":
				foundUptime = true
				assert.Equal(t, MetricTypeGauge, metric.Type)
				assert.Contains(t, metric.Labels, "service")
			}
		}

		assert.True(t, foundMemory, "server_memory_bytes metric not found")
		assert.True(t, foundGoroutines, "server_goroutines metric not found")
		assert.True(t, foundUptime, "server_uptime_seconds metric not found")
	})
}

func TestPhase3Integration_MetricsRegistry(t *testing.T) {
	registry := NewMetricsRegistry()

	t.Run("predefined server metrics are available", func(t *testing.T) {
		expectedMetrics := []string{
			"server_uptime_seconds",
			"server_startup_duration_seconds",
			"server_shutdown_duration_seconds",
			"server_goroutines",
			"server_memory_bytes",
			"server_gc_duration_seconds",
		}

		for _, metricName := range expectedMetrics {
			definition, found := registry.GetMetricDefinition(metricName)
			assert.True(t, found, "Metric %s not found", metricName)
			assert.Equal(t, metricName, definition.Name)
			assert.NotEmpty(t, definition.Description)
			assert.Contains(t, definition.Labels, "service")
		}
	})

	t.Run("transport and service metrics are available", func(t *testing.T) {
		transportMetrics := []string{
			"transport_starts_total",
			"transport_stops_total",
			"active_transports",
			"service_registrations_total",
			"registered_services",
			"transport_status",
			"transport_connections_active",
			"transport_connections_total",
		}

		for _, metricName := range transportMetrics {
			definition, found := registry.GetMetricDefinition(metricName)
			assert.True(t, found, "Transport metric %s not found", metricName)
			assert.Equal(t, metricName, definition.Name)
			assert.NotEmpty(t, definition.Description)
		}
	})
}

// Test helper for custom service registration
type Phase3TestServiceRegistrar struct{}

func (t *Phase3TestServiceRegistrar) RegisterServices(registry BusinessServiceRegistry) error {
	// Register a simple test handler
	handler := &Phase3TestHTTPHandler{}
	return registry.RegisterBusinessHTTPHandler(handler)
}

type Phase3TestHTTPHandler struct{}

func (t *Phase3TestHTTPHandler) RegisterRoutes(router interface{}) error {
	// No routes needed for this test
	return nil
}

func (t *Phase3TestHTTPHandler) GetServiceName() string {
	return "phase3-test-handler"
}

// Benchmark test for Phase 3 integration
func BenchmarkPhase3_MetricsCollection(b *testing.B) {
	config := &ServerConfig{
		ServiceName: "benchmark-service",
		HTTP:        HTTPConfig{Enabled: false},
		GRPC:        GRPCConfig{Enabled: false},
		Discovery:   DiscoveryConfig{Enabled: false},
	}

	registrar := &Phase3TestServiceRegistrar{}
	server, err := NewBusinessServerCore(config, registrar, nil)
	if err != nil {
		b.Fatal(err)
	}

	observabilityManager := server.GetObservabilityManager()
	businessMetrics := server.GetBusinessMetricsManager()

	b.ResetTimer()

	b.Run("system_metrics_update", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				observabilityManager.UpdateSystemMetrics()
			}
		})
	})

	b.Run("business_metrics_recording", func(b *testing.B) {
		labels := map[string]string{"endpoint": "/api/test"}
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				businessMetrics.RecordCounter("benchmark_counter", 1.0, labels)
			}
		})
	})

	b.Run("prometheus_metrics_export", func(b *testing.B) {
		collector := observabilityManager.GetPrometheusCollector()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = collector.GetMetrics()
			}
		})
	})
}

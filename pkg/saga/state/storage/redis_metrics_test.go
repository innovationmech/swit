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

package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestDefaultRedisMetricsConfig(t *testing.T) {
	config := DefaultRedisMetricsConfig()

	if config == nil {
		t.Fatal("DefaultRedisMetricsConfig() returned nil")
	}

	if config.Namespace != "saga" {
		t.Errorf("Expected namespace to be 'saga', got %s", config.Namespace)
	}

	if config.Subsystem != "redis_storage" {
		t.Errorf("Expected subsystem to be 'redis_storage', got %s", config.Subsystem)
	}

	if !config.EnableDetailedMetrics {
		t.Error("Expected detailed metrics to be enabled by default")
	}

	if config.InfoUpdateInterval != 30*time.Second {
		t.Errorf("Expected info update interval to be 30s, got %v", config.InfoUpdateInterval)
	}

	if len(config.DurationBuckets) == 0 {
		t.Error("Expected default duration buckets to be non-empty")
	}
}

func TestNewRedisStorageMetrics(t *testing.T) {
	config := DefaultRedisMetricsConfig()
	config.Registry = prometheus.NewRegistry()

	metrics := NewRedisStorageMetrics(config)

	if metrics == nil {
		t.Fatal("NewRedisStorageMetrics() returned nil")
	}

	if !metrics.IsEnabled() {
		t.Error("Expected metrics to be enabled by default")
	}

	if metrics.GetRegistry() != config.Registry {
		t.Error("Expected metrics registry to match config registry")
	}
}

func TestNewRedisStorageMetrics_NilConfig(t *testing.T) {
	metrics := NewRedisStorageMetrics(nil)

	if metrics == nil {
		t.Fatal("NewRedisStorageMetrics(nil) returned nil")
	}

	if !metrics.IsEnabled() {
		t.Error("Expected metrics to be enabled by default")
	}
}

func TestRedisStorageMetrics_RecordOperation(t *testing.T) {
	config := DefaultRedisMetricsConfig()
	config.Registry = prometheus.NewRegistry()

	metrics := NewRedisStorageMetrics(config)

	tests := []struct {
		name      string
		operation string
		duration  time.Duration
		err       error
	}{
		{
			name:      "successful operation",
			operation: "save_saga",
			duration:  10 * time.Millisecond,
			err:       nil,
		},
		{
			name:      "failed operation",
			operation: "get_saga",
			duration:  5 * time.Millisecond,
			err:       errors.New("not found"),
		},
		{
			name:      "timeout operation",
			operation: "update_saga",
			duration:  100 * time.Millisecond,
			err:       context.DeadlineExceeded,
		},
		{
			name:      "canceled operation",
			operation: "delete_saga",
			duration:  2 * time.Millisecond,
			err:       context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.RecordOperation(tt.operation, tt.duration, tt.err)

			// Verify metrics were recorded
			metricFamilies, err := config.Registry.Gather()
			if err != nil {
				t.Fatalf("Failed to gather metrics: %v", err)
			}

			// Check that we have metrics
			if len(metricFamilies) == 0 {
				t.Error("Expected metrics to be recorded")
			}
		})
	}
}

func TestRedisStorageMetrics_RecordSlowQuery(t *testing.T) {
	config := DefaultRedisMetricsConfig()
	config.Registry = prometheus.NewRegistry()

	metrics := NewRedisStorageMetrics(config)

	tests := []struct {
		name       string
		operation  string
		duration   time.Duration
		threshold  time.Duration
		expectSlow bool
	}{
		{
			name:       "fast query",
			operation:  "save_saga",
			duration:   50 * time.Millisecond,
			threshold:  100 * time.Millisecond,
			expectSlow: false,
		},
		{
			name:       "slow query",
			operation:  "get_saga",
			duration:   150 * time.Millisecond,
			threshold:  100 * time.Millisecond,
			expectSlow: true,
		},
		{
			name:       "exactly at threshold",
			operation:  "update_saga",
			duration:   100 * time.Millisecond,
			threshold:  100 * time.Millisecond,
			expectSlow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.RecordSlowQuery(tt.operation, tt.duration, tt.threshold)

			if tt.expectSlow {
				// Verify slow query counter was incremented
				metricFamilies, err := config.Registry.Gather()
				if err != nil {
					t.Fatalf("Failed to gather metrics: %v", err)
				}

				found := false
				for _, mf := range metricFamilies {
					if mf.GetName() == "saga_redis_storage_slow_queries_total" {
						found = true
						break
					}
				}

				if !found {
					t.Error("Expected slow_queries_total metric to be recorded")
				}
			}
		})
	}
}

func TestRedisStorageMetrics_RecordSagaOperation(t *testing.T) {
	config := DefaultRedisMetricsConfig()
	config.Registry = prometheus.NewRegistry()

	metrics := NewRedisStorageMetrics(config)

	actions := []string{"save", "get", "update", "delete"}

	for _, action := range actions {
		metrics.RecordSagaOperation(action)
	}

	// Verify metrics were recorded
	metricFamilies, err := config.Registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "saga_redis_storage_sagas_total" {
			found = true
			if len(mf.GetMetric()) != len(actions) {
				t.Errorf("Expected %d saga metrics, got %d", len(actions), len(mf.GetMetric()))
			}
		}
	}

	if !found {
		t.Error("Expected sagas_total metric to be recorded")
	}
}

func TestRedisStorageMetrics_RecordStepOperation(t *testing.T) {
	config := DefaultRedisMetricsConfig()
	config.Registry = prometheus.NewRegistry()

	metrics := NewRedisStorageMetrics(config)

	actions := []string{"save", "get"}

	for _, action := range actions {
		metrics.RecordStepOperation(action)
	}

	// Verify metrics were recorded
	metricFamilies, err := config.Registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "saga_redis_storage_saga_steps_total" {
			found = true
		}
	}

	if !found {
		t.Error("Expected saga_steps_total metric to be recorded")
	}
}

func TestRedisStorageMetrics_RecordHealthCheck(t *testing.T) {
	config := DefaultRedisMetricsConfig()
	config.Registry = prometheus.NewRegistry()

	metrics := NewRedisStorageMetrics(config)

	tests := []struct {
		name     string
		healthy  bool
		duration time.Duration
	}{
		{
			name:     "healthy check",
			healthy:  true,
			duration: 10 * time.Millisecond,
		},
		{
			name:     "unhealthy check",
			healthy:  false,
			duration: 50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics.RecordHealthCheck(tt.healthy, tt.duration)

			// Verify metrics were recorded
			metricFamilies, err := config.Registry.Gather()
			if err != nil {
				t.Fatalf("Failed to gather metrics: %v", err)
			}

			// Check health check status gauge
			foundStatus := false
			for _, mf := range metricFamilies {
				if mf.GetName() == "saga_redis_storage_health_check_status" {
					foundStatus = true
					if len(mf.GetMetric()) > 0 {
						gauge := mf.GetMetric()[0].GetGauge()
						expectedValue := 1.0
						if !tt.healthy {
							expectedValue = 0.0
						}
						if gauge.GetValue() != expectedValue {
							t.Errorf("Expected health check status to be %f, got %f", expectedValue, gauge.GetValue())
						}
					}
				}
			}

			if !foundStatus {
				t.Error("Expected health_check_status metric to be recorded")
			}
		})
	}
}

func TestRedisStorageMetrics_UpdatePoolMetrics(t *testing.T) {
	config := DefaultRedisMetricsConfig()
	config.Registry = prometheus.NewRegistry()

	metrics := NewRedisStorageMetrics(config)

	stats := &PoolStats{
		TotalConns: 10,
		IdleConns:  5,
		StaleConns: 1,
		Hits:       100,
		Misses:     10,
		Timeouts:   2,
	}

	metrics.UpdatePoolMetrics(stats)

	// Verify metrics were recorded
	metricFamilies, err := config.Registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	expectedMetrics := map[string]float64{
		"saga_redis_storage_pool_active_connections": float64(stats.TotalConns - stats.IdleConns),
		"saga_redis_storage_pool_idle_connections":   float64(stats.IdleConns),
	}

	for _, mf := range metricFamilies {
		if expectedValue, ok := expectedMetrics[mf.GetName()]; ok {
			if len(mf.GetMetric()) > 0 {
				gauge := mf.GetMetric()[0].GetGauge()
				if gauge.GetValue() != expectedValue {
					t.Errorf("For metric %s, expected value %f, got %f", mf.GetName(), expectedValue, gauge.GetValue())
				}
			}
		}
	}
}

func TestRedisStorageMetrics_UpdatePoolMetrics_Nil(t *testing.T) {
	config := DefaultRedisMetricsConfig()
	config.Registry = prometheus.NewRegistry()

	metrics := NewRedisStorageMetrics(config)

	// Should not panic with nil stats
	metrics.UpdatePoolMetrics(nil)
}

func TestRedisStorageMetrics_UpdateRedisInfo(t *testing.T) {
	config := DefaultRedisMetricsConfig()
	config.Registry = prometheus.NewRegistry()
	config.EnableDetailedMetrics = true

	metrics := NewRedisStorageMetrics(config)

	details := &HealthDetails{
		ConnectedClients:   50,
		UsedMemory:         1024 * 1024 * 100, // 100 MB
		MaxMemory:          1024 * 1024 * 200, // 200 MB
		MemoryUsagePercent: 50.0,
		KeyspaceHits:       1000,
		KeyspaceMisses:     100,
		HitRate:            90.9,
		UptimeSeconds:      86400,
		Version:            "7.0.0",
		Mode:               "standalone",
	}

	metrics.UpdateRedisInfo(details)

	// Verify metrics were recorded
	metricFamilies, err := config.Registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	expectedMetrics := map[string]float64{
		"saga_redis_storage_redis_connected_clients":    float64(details.ConnectedClients),
		"saga_redis_storage_redis_used_memory_bytes":    float64(details.UsedMemory),
		"saga_redis_storage_redis_max_memory_bytes":     float64(details.MaxMemory),
		"saga_redis_storage_redis_memory_usage_percent": details.MemoryUsagePercent,
		"saga_redis_storage_redis_hit_rate_percent":     details.HitRate,
		"saga_redis_storage_redis_uptime_seconds":       float64(details.UptimeSeconds),
	}

	for _, mf := range metricFamilies {
		if expectedValue, ok := expectedMetrics[mf.GetName()]; ok {
			if len(mf.GetMetric()) > 0 {
				gauge := mf.GetMetric()[0].GetGauge()
				if gauge.GetValue() != expectedValue {
					t.Errorf("For metric %s, expected value %f, got %f", mf.GetName(), expectedValue, gauge.GetValue())
				}
			}
		}
	}
}

func TestRedisStorageMetrics_UpdateRedisInfo_Nil(t *testing.T) {
	config := DefaultRedisMetricsConfig()
	config.Registry = prometheus.NewRegistry()

	metrics := NewRedisStorageMetrics(config)

	// Should not panic with nil details
	metrics.UpdateRedisInfo(nil)
}

func TestRedisStorageMetrics_DisableEnable(t *testing.T) {
	config := DefaultRedisMetricsConfig()
	config.Registry = prometheus.NewRegistry()

	metrics := NewRedisStorageMetrics(config)

	if !metrics.IsEnabled() {
		t.Error("Expected metrics to be enabled initially")
	}

	metrics.Disable()

	if metrics.IsEnabled() {
		t.Error("Expected metrics to be disabled after Disable()")
	}

	// Recording should not panic when disabled
	metrics.RecordOperation("test", 10*time.Millisecond, nil)

	metrics.Enable()

	if !metrics.IsEnabled() {
		t.Error("Expected metrics to be enabled after Enable()")
	}
}

func TestRedisStorageMetrics_ConcurrentUpdates(t *testing.T) {
	config := DefaultRedisMetricsConfig()
	config.Registry = prometheus.NewRegistry()

	metrics := NewRedisStorageMetrics(config)

	done := make(chan bool)
	count := 100

	// Concurrent operation recordings
	for i := 0; i < count; i++ {
		go func() {
			metrics.RecordOperation("concurrent_test", 10*time.Millisecond, nil)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < count; i++ {
		<-done
	}

	// Verify metrics were recorded
	metricFamilies, err := config.Registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "saga_redis_storage_operations_total" {
			found = true
			for _, m := range mf.GetMetric() {
				counter := m.GetCounter()
				if counter.GetValue() == float64(count) {
					return // Success
				}
			}
		}
	}

	if !found {
		t.Error("Expected operations_total metric to be recorded")
	}
}

func getMetricValue(mf *dto.MetricFamily) float64 {
	if len(mf.GetMetric()) == 0 {
		return 0
	}

	m := mf.GetMetric()[0]
	if m.GetCounter() != nil {
		return m.GetCounter().GetValue()
	}
	if m.GetGauge() != nil {
		return m.GetGauge().GetValue()
	}
	if m.GetHistogram() != nil {
		return float64(m.GetHistogram().GetSampleCount())
	}
	return 0
}

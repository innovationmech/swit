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
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewPoolMonitor(t *testing.T) {
	config := DefaultRedisConfig()
	config.Addr = "localhost:6379"

	client := redis.NewClient(&redis.Options{
		Addr: config.Addr,
	})
	defer client.Close()

	monitor := NewPoolMonitor(client, config)

	if monitor == nil {
		t.Fatal("Expected monitor to be created")
	}

	if monitor.client != client {
		t.Error("Expected monitor client to match")
	}

	if monitor.config != config {
		t.Error("Expected monitor config to match")
	}

	if !monitor.metricsEnabled {
		t.Error("Expected metrics to be enabled by default")
	}

	if monitor.healthThresholds == nil {
		t.Error("Expected health thresholds to be initialized")
	}
}

func TestCollectStats(t *testing.T) {
	config := DefaultRedisConfig()
	config.Addr = "localhost:6379"

	client := redis.NewClient(&redis.Options{
		Addr: config.Addr,
	})
	defer client.Close()

	monitor := NewPoolMonitor(client, config)
	ctx := context.Background()

	stats, err := monitor.CollectStats(ctx)
	if err != nil {
		t.Fatalf("Failed to collect stats: %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	if stats.PoolSize != config.PoolSize {
		t.Errorf("Expected pool size %d, got %d", config.PoolSize, stats.PoolSize)
	}

	if stats.MinIdleConns != config.MinIdleConns {
		t.Errorf("Expected min idle conns %d, got %d", config.MinIdleConns, stats.MinIdleConns)
	}

	if stats.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	// Verify stats are stored
	lastStats := monitor.GetLastStats()
	if lastStats == nil {
		t.Fatal("Expected last stats to be stored")
	}

	if lastStats.PoolSize != stats.PoolSize {
		t.Error("Expected last stats to match collected stats")
	}
}

func TestGetLastStats(t *testing.T) {
	config := DefaultRedisConfig()
	config.Addr = "localhost:6379"

	client := redis.NewClient(&redis.Options{
		Addr: config.Addr,
	})
	defer client.Close()

	monitor := NewPoolMonitor(client, config)

	// Should return nil if no stats collected yet
	stats := monitor.GetLastStats()
	if stats != nil {
		t.Error("Expected nil when no stats collected")
	}

	// Collect stats
	ctx := context.Background()
	_, err := monitor.CollectStats(ctx)
	if err != nil {
		t.Fatalf("Failed to collect stats: %v", err)
	}

	// Should return stats now
	stats = monitor.GetLastStats()
	if stats == nil {
		t.Error("Expected stats after collection")
	}

	// Verify we get a copy (not the same instance)
	stats2 := monitor.GetLastStats()
	if stats == stats2 {
		t.Error("Expected to get a copy, not the same instance")
	}
}

func TestCheckHealth(t *testing.T) {
	config := DefaultRedisConfig()
	config.Addr = "localhost:6379"
	config.PoolSize = 10

	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		PoolSize: config.PoolSize,
	})
	defer client.Close()

	monitor := NewPoolMonitor(client, config)
	ctx := context.Background()

	health, err := monitor.CheckHealth(ctx)
	if err != nil {
		t.Fatalf("Failed to check health: %v", err)
	}

	if health == nil {
		t.Fatal("Expected health status to be returned")
	}

	if health.Stats == nil {
		t.Error("Expected stats to be included in health status")
	}

	if health.Message == "" {
		t.Error("Expected health message to be set")
	}

	// Initially should be healthy with no warnings
	if !health.Healthy {
		t.Logf("Health status: %s, warnings: %v", health.Message, health.Warnings)
	}
}

func TestHealthThresholds(t *testing.T) {
	config := DefaultRedisConfig()
	config.Addr = "localhost:6379"

	client := redis.NewClient(&redis.Options{
		Addr: config.Addr,
	})
	defer client.Close()

	monitor := NewPoolMonitor(client, config)

	// Get default thresholds
	thresholds := monitor.GetHealthThresholds()
	if thresholds == nil {
		t.Fatal("Expected thresholds to be returned")
	}

	if thresholds.MaxUtilization != 0.9 {
		t.Errorf("Expected default max utilization 0.9, got %f", thresholds.MaxUtilization)
	}

	// Set custom thresholds
	customThresholds := &PoolHealthThresholds{
		MaxUtilization: 0.8,
		MinIdleRatio:   0.2,
		MaxMissRate:    0.1,
		MaxTimeoutRate: 0.01,
	}

	monitor.SetHealthThresholds(customThresholds)

	// Verify thresholds were updated
	updatedThresholds := monitor.GetHealthThresholds()
	if updatedThresholds.MaxUtilization != 0.8 {
		t.Errorf("Expected max utilization 0.8, got %f", updatedThresholds.MaxUtilization)
	}

	if updatedThresholds.MinIdleRatio != 0.2 {
		t.Errorf("Expected min idle ratio 0.2, got %f", updatedThresholds.MinIdleRatio)
	}
}

func TestGetUtilizationMetrics(t *testing.T) {
	config := DefaultRedisConfig()
	config.Addr = "localhost:6379"

	client := redis.NewClient(&redis.Options{
		Addr: config.Addr,
	})
	defer client.Close()

	monitor := NewPoolMonitor(client, config)

	// Should return nil if no stats collected
	metrics := monitor.GetUtilizationMetrics()
	if metrics != nil {
		t.Error("Expected nil metrics when no stats collected")
	}

	// Collect stats
	ctx := context.Background()
	_, err := monitor.CollectStats(ctx)
	if err != nil {
		t.Fatalf("Failed to collect stats: %v", err)
	}

	// Should return metrics now
	metrics = monitor.GetUtilizationMetrics()
	if metrics == nil {
		t.Fatal("Expected metrics after stats collection")
	}

	// Verify metric fields
	expectedFields := []string{
		"total_conns", "idle_conns", "stale_conns",
		"hits", "misses", "timeouts",
	}

	for _, field := range expectedFields {
		if _, ok := metrics[field]; !ok {
			t.Errorf("Expected metric field %s to be present", field)
		}
	}
}

func TestStartMonitoring(t *testing.T) {
	config := DefaultRedisConfig()
	config.Addr = "localhost:6379"
	config.EnableMetrics = true

	client := redis.NewClient(&redis.Options{
		Addr: config.Addr,
	})
	defer client.Close()

	monitor := NewPoolMonitor(client, config)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// Start monitoring in background
	done := make(chan struct{})
	go func() {
		monitor.StartMonitoring(ctx, 100*time.Millisecond)
		close(done)
	}()

	// Wait for context to be cancelled
	<-done

	// Verify stats were collected
	stats := monitor.GetLastStats()
	if stats == nil {
		t.Error("Expected stats to be collected during monitoring")
	}
}

func TestStartMonitoring_MetricsDisabled(t *testing.T) {
	config := DefaultRedisConfig()
	config.Addr = "localhost:6379"
	config.EnableMetrics = false

	client := redis.NewClient(&redis.Options{
		Addr: config.Addr,
	})
	defer client.Close()

	monitor := NewPoolMonitor(client, config)
	monitor.metricsEnabled = false

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start monitoring - should return immediately
	done := make(chan struct{})
	go func() {
		monitor.StartMonitoring(ctx, 50*time.Millisecond)
		close(done)
	}()

	// Should complete quickly
	select {
	case <-done:
		// Expected - function returned immediately
	case <-time.After(150 * time.Millisecond):
		t.Error("StartMonitoring should return immediately when metrics disabled")
	}
}

func TestDefaultPoolHealthThresholds(t *testing.T) {
	thresholds := DefaultPoolHealthThresholds()

	if thresholds == nil {
		t.Fatal("Expected thresholds to be returned")
	}

	tests := []struct {
		name     string
		value    float64
		expected float64
	}{
		{"MaxUtilization", thresholds.MaxUtilization, 0.9},
		{"MinIdleRatio", thresholds.MinIdleRatio, 0.1},
		{"MaxMissRate", thresholds.MaxMissRate, 0.2},
		{"MaxTimeoutRate", thresholds.MaxTimeoutRate, 0.05},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %s to be %f, got %f", tt.name, tt.expected, tt.value)
			}
		})
	}
}

func TestCollectStats_CancelledContext(t *testing.T) {
	config := DefaultRedisConfig()
	config.Addr = "localhost:6379"

	client := redis.NewClient(&redis.Options{
		Addr: config.Addr,
	})
	defer client.Close()

	monitor := NewPoolMonitor(client, config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := monitor.CollectStats(ctx)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestCheckHealth_CancelledContext(t *testing.T) {
	config := DefaultRedisConfig()
	config.Addr = "localhost:6379"

	client := redis.NewClient(&redis.Options{
		Addr: config.Addr,
	})
	defer client.Close()

	monitor := NewPoolMonitor(client, config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	health, err := monitor.CheckHealth(ctx)
	if err != nil {
		t.Errorf("CheckHealth should not return error, got %v", err)
	}

	if health == nil {
		t.Fatal("Expected health status even with cancelled context")
	}

	if health.Healthy {
		t.Error("Expected unhealthy status when stats collection fails")
	}
}

func TestPoolHealthStatus_HighUtilization(t *testing.T) {
	config := DefaultRedisConfig()
	config.Addr = "localhost:6379"
	config.PoolSize = 5

	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		PoolSize: config.PoolSize,
	})
	defer client.Close()

	monitor := NewPoolMonitor(client, config)

	// Set a very low threshold to trigger warnings
	monitor.SetHealthThresholds(&PoolHealthThresholds{
		MaxUtilization: 0.01, // Very low threshold
		MinIdleRatio:   0.01,
		MaxMissRate:    1.0,
		MaxTimeoutRate: 1.0,
	})

	ctx := context.Background()

	// Collect some stats first
	_, _ = monitor.CollectStats(ctx)

	health, err := monitor.CheckHealth(ctx)
	if err != nil {
		t.Fatalf("Failed to check health: %v", err)
	}

	// With the low threshold and some connections, we might get warnings
	if health.Healthy {
		t.Logf("Pool is healthy, warnings: %v", health.Warnings)
	}
}


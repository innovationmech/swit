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
	"strings"
	"testing"
	"time"
)

func TestDefaultHealthCheckConfig(t *testing.T) {
	config := DefaultHealthCheckConfig()

	if config == nil {
		t.Fatal("DefaultHealthCheckConfig() returned nil")
	}

	if !config.Enabled {
		t.Error("Expected health checks to be enabled by default")
	}

	if config.Timeout != 3*time.Second {
		t.Errorf("Expected timeout to be 3s, got %v", config.Timeout)
	}

	if config.Interval != 30*time.Second {
		t.Errorf("Expected interval to be 30s, got %v", config.Interval)
	}

	if !config.PingEnabled {
		t.Error("Expected ping to be enabled by default")
	}

	if !config.InfoEnabled {
		t.Error("Expected info to be enabled by default")
	}

	if config.SlowQueryThreshold != 100*time.Millisecond {
		t.Errorf("Expected slow query threshold to be 100ms, got %v", config.SlowQueryThreshold)
	}

	if config.MaxMemoryWarningPercent != 90 {
		t.Errorf("Expected max memory warning percent to be 90, got %d", config.MaxMemoryWarningPercent)
	}

	if config.MaxConnectionWarningPercent != 90 {
		t.Errorf("Expected max connection warning percent to be 90, got %d", config.MaxConnectionWarningPercent)
	}
}

func TestRedisHealthChecker_GetName(t *testing.T) {
	// Create a mock storage
	storage := &RedisStateStorage{
		config: DefaultRedisConfig(),
	}

	checker := NewRedisHealthChecker(storage, nil)

	name := checker.GetName()
	if name != "redis-saga-storage" {
		t.Errorf("Expected name to be 'redis-saga-storage', got %s", name)
	}
}

func TestRedisHealthChecker_CheckHealth_Disabled(t *testing.T) {
	storage := &RedisStateStorage{
		config: DefaultRedisConfig(),
	}

	config := DefaultHealthCheckConfig()
	config.Enabled = false

	checker := NewRedisHealthChecker(storage, config)

	ctx := context.Background()
	status := checker.CheckHealth(ctx)

	if !status.Healthy {
		t.Error("Expected health check to be healthy when disabled")
	}

	if status.Message != "health checks disabled" {
		t.Errorf("Expected message 'health checks disabled', got %s", status.Message)
	}
}

func TestRedisHealthChecker_GetLastStatus(t *testing.T) {
	storage := &RedisStateStorage{
		config: DefaultRedisConfig(),
	}

	checker := NewRedisHealthChecker(storage, nil)

	// Get status before any check
	status := checker.GetLastStatus()
	if status == nil {
		t.Fatal("GetLastStatus() returned nil")
	}

	if !status.Healthy {
		t.Error("Expected initial status to be healthy")
	}

	if status.Message != "no health check performed yet" {
		t.Errorf("Expected message 'no health check performed yet', got %s", status.Message)
	}
}

func TestParseRedisInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:  "single key-value",
			input: "redis_version:7.0.0\r\n",
			expected: map[string]string{
				"redis_version": "7.0.0",
			},
		},
		{
			name:  "multiple key-values",
			input: "redis_version:7.0.0\r\nredis_mode:standalone\r\nuptime_in_seconds:1000\r\n",
			expected: map[string]string{
				"redis_version":     "7.0.0",
				"redis_mode":        "standalone",
				"uptime_in_seconds": "1000",
			},
		},
		{
			name:  "with comments",
			input: "# Server\r\nredis_version:7.0.0\r\n# Clients\r\nconnected_clients:10\r\n",
			expected: map[string]string{
				"redis_version":     "7.0.0",
				"connected_clients": "10",
			},
		},
		{
			name:  "with newlines only",
			input: "redis_version:7.0.0\nredis_mode:standalone\n",
			expected: map[string]string{
				"redis_version": "7.0.0",
				"redis_mode":    "standalone",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRedisInfo(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				actualValue, ok := result[key]
				if !ok {
					t.Errorf("Expected key %s not found in result", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("For key %s, expected value %s, got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestParseRedisInfoInt(t *testing.T) {
	tests := []struct {
		name     string
		info     map[string]string
		key      string
		expected int64
		found    bool
	}{
		{
			name:     "valid integer",
			info:     map[string]string{"uptime_in_seconds": "1000"},
			key:      "uptime_in_seconds",
			expected: 1000,
			found:    true,
		},
		{
			name:     "zero value",
			info:     map[string]string{"count": "0"},
			key:      "count",
			expected: 0,
			found:    true,
		},
		{
			name:     "large number",
			info:     map[string]string{"used_memory": "123456789"},
			key:      "used_memory",
			expected: 123456789,
			found:    true,
		},
		{
			name:     "non-existent key",
			info:     map[string]string{"other_key": "100"},
			key:      "missing_key",
			expected: 0,
			found:    false,
		},
		{
			name:     "non-numeric value",
			info:     map[string]string{"version": "7.0.0"},
			key:      "version",
			expected: 7,
			found:    true,
		},
		{
			name:     "value with suffix",
			info:     map[string]string{"memory": "1000M"},
			key:      "memory",
			expected: 1000,
			found:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := parseRedisInfoInt(tt.info, tt.key)

			if found != tt.found {
				t.Errorf("Expected found to be %v, got %v", tt.found, found)
			}

			if value != tt.expected {
				t.Errorf("Expected value to be %d, got %d", tt.expected, value)
			}
		})
	}
}

func TestHealthDetails_Warnings(t *testing.T) {
	tests := []struct {
		name                    string
		maxMemoryWarningPercent int
		details                 *HealthDetails
		expected                []string
	}{
		{
			name:                    "high memory usage",
			maxMemoryWarningPercent: 80,
			details: &HealthDetails{
				UsedMemory:         800,
				MaxMemory:          1000,
				MemoryUsagePercent: 80.0,
			},
			expected: []string{"memory usage is high: 80.00%"},
		},
		{
			name:                    "very high memory usage",
			maxMemoryWarningPercent: 90,
			details: &HealthDetails{
				UsedMemory:         950,
				MaxMemory:          1000,
				MemoryUsagePercent: 95.0,
			},
			expected: []string{"memory usage is high: 95.00%"},
		},
		{
			name:                    "low hit rate but insufficient operations",
			maxMemoryWarningPercent: 90,
			details: &HealthDetails{
				UsedMemory:         500,
				MaxMemory:          1000,
				MemoryUsagePercent: 50.0,
				KeyspaceHits:       40,
				KeyspaceMisses:     60,
				HitRate:            40.0,
			},
			expected: nil,
		},
		{
			name:                    "normal memory and good hit rate",
			maxMemoryWarningPercent: 90,
			details: &HealthDetails{
				UsedMemory:         500,
				MaxMemory:          1000,
				MemoryUsagePercent: 50.0,
				KeyspaceHits:       800,
				KeyspaceMisses:     200,
				HitRate:            80.0,
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &RedisStateStorage{
				config: DefaultRedisConfig(),
			}

			config := DefaultHealthCheckConfig()
			config.MaxMemoryWarningPercent = tt.maxMemoryWarningPercent

			checker := NewRedisHealthChecker(storage, config)
			checker.checkWarnings(tt.details)

			if len(tt.details.Warnings) != len(tt.expected) {
				t.Errorf("Expected %d warnings, got %d: %v", len(tt.expected), len(tt.details.Warnings), tt.details.Warnings)
			}

			for i, expectedWarning := range tt.expected {
				if i >= len(tt.details.Warnings) {
					t.Errorf("Missing expected warning: %s", expectedWarning)
					continue
				}
				if !strings.Contains(tt.details.Warnings[i], expectedWarning) {
					t.Errorf("Expected warning to contain %s, got %s", expectedWarning, tt.details.Warnings[i])
				}
			}
		})
	}
}

func TestRedisHealthChecker_StopPeriodicCheck(t *testing.T) {
	storage := &RedisStateStorage{
		config: DefaultRedisConfig(),
	}

	config := DefaultHealthCheckConfig()
	config.Interval = 100 * time.Millisecond

	checker := NewRedisHealthChecker(storage, config)

	// Stop before starting should not panic
	checker.StopPeriodicCheck()

	// Stop multiple times should not panic
	checker.StopPeriodicCheck()
	checker.StopPeriodicCheck()
}

func TestRedisHealthChecker_PeriodicCheck_DisabledByConfig(t *testing.T) {
	storage := &RedisStateStorage{
		config: DefaultRedisConfig(),
	}

	config := DefaultHealthCheckConfig()
	config.Enabled = false

	checker := NewRedisHealthChecker(storage, config)

	ctx := context.Background()
	doneCh := checker.StartPeriodicCheck(ctx)

	// Should complete immediately when disabled
	select {
	case <-doneCh:
		// Expected - channel should be closed immediately
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected periodic check to stop immediately when disabled")
	}
}

func TestRedisHealthChecker_PeriodicCheck_ZeroInterval(t *testing.T) {
	storage := &RedisStateStorage{
		config: DefaultRedisConfig(),
	}

	config := DefaultHealthCheckConfig()
	config.Interval = 0

	checker := NewRedisHealthChecker(storage, config)

	ctx := context.Background()
	doneCh := checker.StartPeriodicCheck(ctx)

	// Should complete immediately when interval is zero
	select {
	case <-doneCh:
		// Expected - channel should be closed immediately
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected periodic check to stop immediately when interval is zero")
	}
}

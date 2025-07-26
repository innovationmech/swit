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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHealthStatusConstants(t *testing.T) {
	// Test that constants are properly defined
	assert.Equal(t, "healthy", HealthStatusHealthy)
	assert.Equal(t, "unhealthy", HealthStatusUnhealthy)
	assert.Equal(t, "degraded", HealthStatusDegraded)
}

func TestDependencyStatusConstants(t *testing.T) {
	// Test that dependency status constants are properly defined
	assert.Equal(t, "up", DependencyStatusUp)
	assert.Equal(t, "down", DependencyStatusDown)
	assert.Equal(t, "slow", DependencyStatusSlow)
}

func TestNewHealthStatus(t *testing.T) {
	version := "v1.2.3"
	uptime := 5 * time.Minute

	status := NewHealthStatus(HealthStatusHealthy, version, uptime)

	assert.NotNil(t, status)
	assert.Equal(t, HealthStatusHealthy, status.Status)
	assert.Equal(t, version, status.Version)
	assert.Equal(t, uptime, status.Uptime)
	assert.NotZero(t, status.Timestamp)
	assert.True(t, time.Since(status.Timestamp) < time.Second) // Should be recent
	assert.NotNil(t, status.Dependencies)
	assert.Equal(t, 0, len(status.Dependencies))
}

func TestNewHealthStatus_WithDifferentStatuses(t *testing.T) {
	testCases := []struct {
		name    string
		status  string
		version string
		uptime  time.Duration
	}{
		{
			name:    "healthy status",
			status:  HealthStatusHealthy,
			version: "v1.0.0",
			uptime:  time.Hour,
		},
		{
			name:    "unhealthy status",
			status:  HealthStatusUnhealthy,
			version: "v2.0.0",
			uptime:  time.Minute,
		},
		{
			name:    "degraded status",
			status:  HealthStatusDegraded,
			version: "v3.0.0",
			uptime:  time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status := NewHealthStatus(tc.status, tc.version, tc.uptime)

			assert.Equal(t, tc.status, status.Status)
			assert.Equal(t, tc.version, status.Version)
			assert.Equal(t, tc.uptime, status.Uptime)
		})
	}
}

func TestNewDependencyStatus(t *testing.T) {
	status := DependencyStatusUp
	latency := 50 * time.Millisecond

	depStatus := NewDependencyStatus(status, latency)

	assert.NotNil(t, depStatus)
	assert.Equal(t, status, depStatus.Status)
	assert.Equal(t, latency, depStatus.Latency)
	assert.NotZero(t, depStatus.Timestamp)
	assert.True(t, time.Since(depStatus.Timestamp) < time.Second) // Should be recent
}

func TestNewDependencyStatus_WithDifferentStatuses(t *testing.T) {
	testCases := []struct {
		name    string
		status  string
		latency time.Duration
	}{
		{
			name:    "up status",
			status:  DependencyStatusUp,
			latency: 10 * time.Millisecond,
		},
		{
			name:    "down status",
			status:  DependencyStatusDown,
			latency: 0,
		},
		{
			name:    "slow status",
			status:  DependencyStatusSlow,
			latency: 500 * time.Millisecond,
		},
		{
			name:    "zero latency",
			status:  DependencyStatusUp,
			latency: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			depStatus := NewDependencyStatus(tc.status, tc.latency)

			assert.Equal(t, tc.status, depStatus.Status)
			assert.Equal(t, tc.latency, depStatus.Latency)
			assert.NotZero(t, depStatus.Timestamp)
		})
	}
}

func TestNewDependencyStatusWithError(t *testing.T) {
	status := DependencyStatusDown
	errorMsg := "Connection timeout"

	depStatus := NewDependencyStatusWithError(status, errorMsg)

	assert.NotNil(t, depStatus)
	assert.Equal(t, status, depStatus.Status)
	assert.Equal(t, errorMsg, depStatus.Error)
	assert.NotZero(t, depStatus.Timestamp)
	assert.True(t, time.Since(depStatus.Timestamp) < time.Second) // Should be recent
}

func TestHealthStatus_WithDependencies(t *testing.T) {
	// Create a health status and add dependencies
	status := NewHealthStatus(HealthStatusHealthy, "v1.0.0", time.Hour)

	// Add some dependencies
	status.Dependencies["database"] = NewDependencyStatus(DependencyStatusUp, 10*time.Millisecond)
	status.Dependencies["cache"] = NewDependencyStatus(DependencyStatusSlow, 500*time.Millisecond)
	status.Dependencies["external-api"] = NewDependencyStatusWithError(DependencyStatusDown, "Timeout")

	assert.Equal(t, 3, len(status.Dependencies))
	assert.Equal(t, DependencyStatusUp, status.Dependencies["database"].Status)
	assert.Equal(t, 10*time.Millisecond, status.Dependencies["database"].Latency)
	assert.Equal(t, DependencyStatusSlow, status.Dependencies["cache"].Status)
	assert.Equal(t, 500*time.Millisecond, status.Dependencies["cache"].Latency)
	assert.Equal(t, DependencyStatusDown, status.Dependencies["external-api"].Status)
	assert.Equal(t, "Timeout", status.Dependencies["external-api"].Error)
}

func TestHealthStatus_TimestampAccuracy(t *testing.T) {
	start := time.Now()
	status := NewHealthStatus(HealthStatusHealthy, "v1.0.0", time.Hour)
	end := time.Now()

	// Timestamp should be between start and end
	assert.True(t, status.Timestamp.After(start) || status.Timestamp.Equal(start))
	assert.True(t, status.Timestamp.Before(end) || status.Timestamp.Equal(end))
}

func TestDependencyStatus_TimestampAccuracy(t *testing.T) {
	start := time.Now()
	depStatus := NewDependencyStatus(DependencyStatusUp, 10*time.Millisecond)
	end := time.Now()

	// Timestamp should be between start and end
	assert.True(t, depStatus.Timestamp.After(start) || depStatus.Timestamp.Equal(start))
	assert.True(t, depStatus.Timestamp.Before(end) || depStatus.Timestamp.Equal(end))
}

func TestHealthStatus_ZeroUptime(t *testing.T) {
	status := NewHealthStatus(HealthStatusHealthy, "v1.0.0", 0)

	assert.Equal(t, time.Duration(0), status.Uptime)
	assert.Equal(t, HealthStatusHealthy, status.Status)
}

func TestHealthStatus_EmptyVersion(t *testing.T) {
	status := NewHealthStatus(HealthStatusHealthy, "", time.Hour)

	assert.Equal(t, "", status.Version)
	assert.Equal(t, HealthStatusHealthy, status.Status)
}

func TestHealthStatus_ConcurrentAccess(t *testing.T) {
	// Test concurrent creation of health statuses
	const numGoroutines = 100
	statuses := make([]*HealthStatus, numGoroutines)

	done := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			statuses[index] = NewHealthStatus(HealthStatusHealthy, "v1.0.0", time.Duration(index)*time.Second)
			done <- index
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all statuses were created properly
	for i, status := range statuses {
		assert.NotNil(t, status)
		assert.Equal(t, HealthStatusHealthy, status.Status)
		assert.Equal(t, "v1.0.0", status.Version)
		assert.Equal(t, time.Duration(i)*time.Second, status.Uptime)
	}
}

func TestDependencyStatus_ConcurrentAccess(t *testing.T) {
	// Test concurrent creation of dependency statuses
	const numGoroutines = 100
	statuses := make([]DependencyStatus, numGoroutines)

	done := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			statuses[index] = NewDependencyStatus(DependencyStatusUp, 10*time.Millisecond)
			done <- index
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all statuses were created properly
	for _, status := range statuses {
		assert.Equal(t, DependencyStatusUp, status.Status)
		assert.Equal(t, 10*time.Millisecond, status.Latency)
		assert.NotZero(t, status.Timestamp)
	}
}

func TestHealthStatus_Dependencies_ModificationSafety(t *testing.T) {
	// Test that modifying the dependencies map works correctly
	status := NewHealthStatus(HealthStatusHealthy, "v1.0.0", time.Hour)

	// Initially empty
	assert.Equal(t, 0, len(status.Dependencies))

	// Add a dependency
	status.Dependencies["test"] = NewDependencyStatus(DependencyStatusUp, 10*time.Millisecond)
	assert.Equal(t, 1, len(status.Dependencies))

	// Modify the dependency
	status.Dependencies["test"] = NewDependencyStatusWithError(DependencyStatusDown, "Failed")
	assert.Equal(t, 1, len(status.Dependencies))
	assert.Equal(t, DependencyStatusDown, status.Dependencies["test"].Status)
	assert.Equal(t, "Failed", status.Dependencies["test"].Error)

	// Delete the dependency
	delete(status.Dependencies, "test")
	assert.Equal(t, 0, len(status.Dependencies))
}

// Benchmark tests
func BenchmarkNewHealthStatus(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewHealthStatus(HealthStatusHealthy, "v1.0.0", time.Hour)
	}
}

func BenchmarkNewDependencyStatus(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewDependencyStatus(DependencyStatusUp, 10*time.Millisecond)
	}
}

func BenchmarkHealthStatus_WithDependencies(b *testing.B) {
	for i := 0; i < b.N; i++ {
		status := NewHealthStatus(HealthStatusHealthy, "v1.0.0", time.Hour)
		status.Dependencies["db"] = NewDependencyStatus(DependencyStatusUp, 10*time.Millisecond)
		status.Dependencies["cache"] = NewDependencyStatus(DependencyStatusUp, 20*time.Millisecond)
		status.Dependencies["api"] = NewDependencyStatus(DependencyStatusUp, 30*time.Millisecond)
	}
}

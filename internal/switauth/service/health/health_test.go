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

package health

import (
	"context"
	"testing"
	"time"

	"github.com/innovationmech/swit/internal/switauth/interfaces"
	"github.com/innovationmech/swit/internal/switauth/types"
	"github.com/stretchr/testify/assert"
)

// Test helper functions
func createTestHealthService() interfaces.HealthService {
	return NewHealthService()
}

// TestNewHealthService tests the constructor
func TestNewHealthService(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "success_create_service",
			description: "Should create health service successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewHealthService()

			assert.NotNil(t, service)
			assert.Implements(t, (*interfaces.HealthService)(nil), service)

			// Verify service can be cast to concrete type
			healthSvc, ok := service.(*healthService)
			assert.True(t, ok)
			assert.Equal(t, "switauth", healthSvc.name)
			assert.Equal(t, "1.0.0", healthSvc.version)
		})
	}
}

// TestHealthService_CheckHealth tests the CheckHealth method
func TestHealthService_CheckHealth(t *testing.T) {
	tests := []struct {
		name            string
		ctx             context.Context
		expectedStatus  string
		expectedVersion string
		description     string
	}{
		{
			name:            "success_check_health",
			ctx:             context.Background(),
			expectedStatus:  types.HealthStatusHealthy,
			expectedVersion: "1.0.0",
			description:     "Should return healthy status",
		},
		{
			name:            "success_with_timeout_context",
			ctx:             createTimeoutContext(5 * time.Second),
			expectedStatus:  types.HealthStatusHealthy,
			expectedVersion: "1.0.0",
			description:     "Should return healthy status with timeout context",
		},
		{
			name:            "success_with_cancelled_context",
			ctx:             createCancelledContext(),
			expectedStatus:  types.HealthStatusHealthy,
			expectedVersion: "1.0.0",
			description:     "Should return healthy status even with cancelled context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := createTestHealthService()

			status := service.CheckHealth(tt.ctx)

			assert.NotNil(t, status)
			assert.Equal(t, tt.expectedStatus, status.Status)
			assert.Equal(t, tt.expectedVersion, status.Version)
			assert.Equal(t, time.Duration(0), status.Uptime)
			assert.WithinDuration(t, time.Now(), status.Timestamp, time.Second)
			assert.NotNil(t, status.Dependencies)
			assert.Empty(t, status.Dependencies)
			assert.True(t, status.IsHealthy())
		})
	}
}

// TestHealthService_GetServiceInfo tests the GetServiceInfo method
func TestHealthService_GetServiceInfo(t *testing.T) {
	tests := []struct {
		name            string
		ctx             context.Context
		expectedName    string
		expectedVersion string
		expectedDesc    string
		expectedEnv     string
		description     string
	}{
		{
			name:            "success_get_service_info",
			ctx:             context.Background(),
			expectedName:    "switauth",
			expectedVersion: "1.0.0",
			expectedDesc:    "Authentication service",
			expectedEnv:     "production",
			description:     "Should return correct service information",
		},
		{
			name:            "success_with_timeout_context",
			ctx:             createTimeoutContext(5 * time.Second),
			expectedName:    "switauth",
			expectedVersion: "1.0.0",
			expectedDesc:    "Authentication service",
			expectedEnv:     "production",
			description:     "Should return service info with timeout context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := createTestHealthService()

			info := service.GetServiceInfo(tt.ctx)

			assert.NotNil(t, info)
			assert.Equal(t, tt.expectedName, info.Name)
			assert.Equal(t, tt.expectedVersion, info.Version)
			assert.Equal(t, tt.expectedDesc, info.Description)
			assert.Equal(t, tt.expectedEnv, info.Environment)
			assert.WithinDuration(t, time.Now(), info.StartTime, time.Second)
			assert.GreaterOrEqual(t, info.Uptime, time.Duration(0))
			assert.Nil(t, info.BuildInfo) // BuildInfo is not set in the implementation
		})
	}
}

// TestHealthService_IsHealthy tests the IsHealthy method
func TestHealthService_IsHealthy(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		expected    bool
		description string
	}{
		{
			name:        "success_is_healthy",
			ctx:         context.Background(),
			expected:    true,
			description: "Should return true for healthy service",
		},
		{
			name:        "success_with_timeout_context",
			ctx:         createTimeoutContext(5 * time.Second),
			expected:    true,
			description: "Should return true with timeout context",
		},
		{
			name:        "success_with_cancelled_context",
			ctx:         createCancelledContext(),
			expected:    true,
			description: "Should return true even with cancelled context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := createTestHealthService()

			result := service.IsHealthy(tt.ctx)

			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestHealthService_GetHealthDetails tests the GetHealthDetails method
func TestHealthService_GetHealthDetails(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		description string
	}{
		{
			name:        "success_get_health_details",
			ctx:         context.Background(),
			description: "Should return comprehensive health details",
		},
		{
			name:        "success_with_timeout_context",
			ctx:         createTimeoutContext(5 * time.Second),
			description: "Should return health details with timeout context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := createTestHealthService()

			details := service.GetHealthDetails(tt.ctx)

			assert.NotNil(t, details)
			assert.Contains(t, details, "service")
			assert.Contains(t, details, "version")
			assert.Contains(t, details, "status")
			assert.Contains(t, details, "timestamp")
			assert.Contains(t, details, "uptime")
			assert.Contains(t, details, "checks")

			// Verify values
			assert.Equal(t, "switauth", details["service"])
			assert.Equal(t, "1.0.0", details["version"])
			assert.Equal(t, "healthy", details["status"])
			assert.Equal(t, "running", details["uptime"])

			// Verify timestamp is recent
			timestamp, ok := details["timestamp"].(int64)
			assert.True(t, ok)
			assert.WithinDuration(t, time.Now(), time.Unix(timestamp, 0), time.Second)

			// Verify checks
			checks, ok := details["checks"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "ok", checks["database"])
			assert.Equal(t, "ok", checks["cache"])
			assert.Equal(t, "ok", checks["external"])
		})
	}
}

// TestHealthService_InterfaceCompliance tests interface compliance
func TestHealthService_InterfaceCompliance(t *testing.T) {
	t.Run("implements_health_service_interface", func(t *testing.T) {
		service := NewHealthService()
		assert.Implements(t, (*interfaces.HealthService)(nil), service)
	})
}

// TestHealthService_ConcurrentAccess tests concurrent access to methods
func TestHealthService_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent_health_checks", func(t *testing.T) {
		service := createTestHealthService()
		ctx := context.Background()
		const numGoroutines = 100

		done := make(chan bool, numGoroutines)

		// Start multiple goroutines calling different methods
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				defer func() { done <- true }()

				switch index % 4 {
				case 0:
					status := service.CheckHealth(ctx)
					assert.NotNil(t, status)
				case 1:
					info := service.GetServiceInfo(ctx)
					assert.NotNil(t, info)
				case 2:
					healthy := service.IsHealthy(ctx)
					assert.True(t, healthy)
				case 3:
					details := service.GetHealthDetails(ctx)
					assert.NotNil(t, details)
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// Success
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for goroutines to complete")
			}
		}
	})
}

// TestHealthService_ContextHandling tests context handling
func TestHealthService_ContextHandling(t *testing.T) {
	tests := []struct {
		name        string
		ctxFunc     func() context.Context
		description string
	}{
		{
			name:        "nil_context",
			ctxFunc:     func() context.Context { return nil },
			description: "Should handle nil context gracefully",
		},
		{
			name:        "background_context",
			ctxFunc:     func() context.Context { return context.Background() },
			description: "Should work with background context",
		},
		{
			name:        "todo_context",
			ctxFunc:     func() context.Context { return context.TODO() },
			description: "Should work with TODO context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := createTestHealthService()
			ctx := tt.ctxFunc()

			// Test all methods with the context
			assert.NotPanics(t, func() {
				status := service.CheckHealth(ctx)
				assert.NotNil(t, status)

				info := service.GetServiceInfo(ctx)
				assert.NotNil(t, info)

				healthy := service.IsHealthy(ctx)
				assert.True(t, healthy)

				details := service.GetHealthDetails(ctx)
				assert.NotNil(t, details)
			})
		})
	}
}

// TestHealthService_Integration tests complete workflow
func TestHealthService_Integration(t *testing.T) {
	t.Run("complete_health_check_workflow", func(t *testing.T) {
		service := createTestHealthService()
		ctx := context.Background()

		// Step 1: Check if service is healthy
		healthy := service.IsHealthy(ctx)
		assert.True(t, healthy)

		// Step 2: Get basic health status
		status := service.CheckHealth(ctx)
		assert.NotNil(t, status)
		assert.Equal(t, types.HealthStatusHealthy, status.Status)
		assert.True(t, status.IsHealthy())

		// Step 3: Get service information
		info := service.GetServiceInfo(ctx)
		assert.NotNil(t, info)
		assert.Equal(t, "switauth", info.Name)
		assert.Equal(t, "1.0.0", info.Version)

		// Step 4: Get detailed health information
		details := service.GetHealthDetails(ctx)
		assert.NotNil(t, details)
		assert.Equal(t, "switauth", details["service"])
		assert.Equal(t, "healthy", details["status"])

		// Verify consistency between methods
		assert.Equal(t, status.Status, details["status"])
		assert.Equal(t, info.Name, details["service"])
		assert.Equal(t, info.Version, details["version"])
	})
}

// Helper functions for creating test contexts
func createTimeoutContext(timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// Note: In real usage, cancel should be called when done
	// For testing purposes, we'll defer cancel to avoid context leak
	go func() {
		time.Sleep(timeout)
		cancel()
	}()
	return ctx
}

func createCancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	return ctx
}

// Benchmark tests
func BenchmarkHealthService_CheckHealth(b *testing.B) {
	service := createTestHealthService()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.CheckHealth(ctx)
	}
}

func BenchmarkHealthService_GetServiceInfo(b *testing.B) {
	service := createTestHealthService()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.GetServiceInfo(ctx)
	}
}

func BenchmarkHealthService_IsHealthy(b *testing.B) {
	service := createTestHealthService()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.IsHealthy(ctx)
	}
}

func BenchmarkHealthService_GetHealthDetails(b *testing.B) {
	service := createTestHealthService()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.GetHealthDetails(ctx)
	}
}

func BenchmarkHealthService_ConcurrentAccess(b *testing.B) {
	service := createTestHealthService()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = service.CheckHealth(ctx)
			_ = service.IsHealthy(ctx)
		}
	})
}

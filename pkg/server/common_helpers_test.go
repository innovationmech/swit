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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Mock types for testing

type mockCloser struct {
	closeError error
}

func (m *mockCloser) Close() error {
	return m.closeError
}

type mockHealthChecker struct {
	healthError error
}

func (m *mockHealthChecker) HealthCheck(ctx context.Context) error {
	return m.healthError
}

// Test CommonServicePatterns

func TestNewCommonServicePatterns(t *testing.T) {
	patterns := NewCommonServicePatterns()
	assert.NotNil(t, patterns)
}

func TestCommonServicePatterns_CreateDatabaseHealthCheck(t *testing.T) {
	patterns := NewCommonServicePatterns()

	tests := []struct {
		name        string
		db          *gorm.DB
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil database",
			db:          nil,
			expectError: true,
			errorMsg:    "database connection is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthCheck := patterns.CreateDatabaseHealthCheck(tt.db)
			ctx := context.Background()

			err := healthCheck(ctx)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Note: TestCommonServicePatterns_CreateDatabaseHealthCheck_WithValidDB would require
// actual database setup which is complex for unit tests. The nil database test above
// covers the main error path.

func TestCommonServicePatterns_CreateServiceAvailabilityCheck(t *testing.T) {
	patterns := NewCommonServicePatterns()

	tests := []struct {
		name        string
		service     interface{}
		serviceName string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil service",
			service:     nil,
			serviceName: "test-service",
			expectError: true,
			errorMsg:    "test-service is not available",
		},
		{
			name:        "valid service without health check",
			service:     "some-service",
			serviceName: "test-service",
			expectError: false,
		},
		{
			name:        "service with successful health check",
			service:     &mockHealthChecker{healthError: nil},
			serviceName: "test-service",
			expectError: false,
		},
		{
			name:        "service with failing health check",
			service:     &mockHealthChecker{healthError: errors.New("health check failed")},
			serviceName: "test-service",
			expectError: true,
			errorMsg:    "health check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthCheck := patterns.CreateServiceAvailabilityCheck(tt.service, tt.serviceName)
			ctx := context.Background()

			err := healthCheck(ctx)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCommonServicePatterns_SafeCloseDatabase(t *testing.T) {
	patterns := NewCommonServicePatterns()

	tests := []struct {
		name        string
		db          *gorm.DB
		expectError bool
	}{
		{
			name:        "nil database",
			db:          nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := patterns.SafeCloseDatabase(tt.db, "test-service")

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Note: SafeCloseDatabase with valid DB would require actual database setup

func TestCommonServicePatterns_CreateGenericDependencyContainer(t *testing.T) {
	patterns := NewCommonServicePatterns()

	services := map[string]interface{}{
		"service1": "test-service-1",
		"service2": "test-service-2",
	}

	// Test without database
	container := patterns.CreateGenericDependencyContainer("test-service", services, nil)

	require.NotNil(t, container)

	// Test that services were registered
	service1, err := container.GetService("service1")
	require.NoError(t, err)
	assert.Equal(t, "test-service-1", service1)

	service2, err := container.GetService("service2")
	require.NoError(t, err)
	assert.Equal(t, "test-service-2", service2)

	// Test that database was not registered (since it was nil)
	_, err = container.GetService("database")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service database not found")

	// Test closing container
	err = container.Close()
	require.NoError(t, err)
}

func TestCommonServicePatterns_CreateGenericDependencyContainer_WithoutDB(t *testing.T) {
	patterns := NewCommonServicePatterns()

	services := map[string]interface{}{
		"service1": "test-service-1",
	}

	container := patterns.CreateGenericDependencyContainer("test-service", services, nil)

	require.NotNil(t, container)

	// Test that service was registered
	service1, err := container.GetService("service1")
	require.NoError(t, err)
	assert.Equal(t, "test-service-1", service1)

	// Test that database service doesn't exist
	_, err = container.GetService("database")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service database not found")

	// Test closing container
	err = container.Close()
	require.NoError(t, err)
}

func TestCommonServicePatterns_ApplyStandardDefaults(t *testing.T) {
	patterns := NewCommonServicePatterns()

	defaults := StandardConfigDefaults{
		ServiceName:          "test-service",
		HTTPPort:             "8080",
		GRPCPort:             "9080",
		EnableHTTPReady:      true,
		EnableGRPCKeepalive:  true,
		EnableGRPCReflection: true,
		EnableGRPCHealth:     true,
		DiscoveryTags:        []string{"api", "v1"},
		EnableCORS:           true,
		EnableAuth:           false,
		EnableRateLimit:      true,
		EnableLogging:        true,
	}

	config := &ServerConfig{}
	patterns.ApplyStandardDefaults(config, defaults)

	// Verify server configuration
	assert.Equal(t, "test-service", config.ServiceName)
	assert.Equal(t, 5*time.Second, config.ShutdownTimeout)

	// Verify HTTP configuration
	assert.Equal(t, "8080", config.HTTP.Port)
	assert.Equal(t, ":8080", config.HTTP.Address)
	assert.True(t, config.HTTP.Enabled)
	assert.True(t, config.HTTP.EnableReady)

	// Verify gRPC configuration
	assert.Equal(t, "9080", config.GRPC.Port)
	assert.Equal(t, ":9080", config.GRPC.Address)
	assert.True(t, config.GRPC.Enabled)
	assert.True(t, config.GRPC.EnableKeepalive)
	assert.True(t, config.GRPC.EnableReflection)
	assert.True(t, config.GRPC.EnableHealthService)

	// Verify discovery configuration
	assert.Equal(t, "test-service", config.Discovery.ServiceName)
	assert.Equal(t, []string{"api", "v1"}, config.Discovery.Tags)
	assert.True(t, config.Discovery.Enabled)

	// Verify middleware configuration
	assert.True(t, config.Middleware.EnableCORS)
	assert.False(t, config.Middleware.EnableAuth)
	assert.True(t, config.Middleware.EnableRateLimit)
	assert.True(t, config.Middleware.EnableLogging)

	// Verify HTTP middleware configuration
	assert.True(t, config.HTTP.Middleware.EnableCORS)
	assert.False(t, config.HTTP.Middleware.EnableAuth)
	assert.True(t, config.HTTP.Middleware.EnableRateLimit)
	assert.True(t, config.HTTP.Middleware.EnableLogging)
}

// Test ServiceRegistrationBatch

func TestNewServiceRegistrationBatch(t *testing.T) {
	registry := &mockBusinessServiceRegistry{}
	batch := NewServiceRegistrationBatch(registry)

	assert.NotNil(t, batch)
	assert.NotNil(t, batch.helper)
	assert.Equal(t, registry, batch.helper.registry)
}

func TestServiceRegistrationBatch_RegisterHTTPServices(t *testing.T) {
	tests := []struct {
		name        string
		services    []HTTPServiceDefinition
		setupMock   func(*mockBusinessServiceRegistry)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful registration",
			services: []HTTPServiceDefinition{
				{
					ServiceName:     "service1",
					Registrar:       &mockHTTPHandlerRegistrar{},
					HealthCheckFunc: func(ctx context.Context) error { return nil },
				},
				{
					ServiceName:     "service2",
					Registrar:       &mockHTTPHandlerRegistrar{},
					HealthCheckFunc: func(ctx context.Context) error { return nil },
				},
			},
			setupMock: func(m *mockBusinessServiceRegistry) {
				m.On("RegisterBusinessHTTPHandler", mock.Anything).Return(nil).Times(2)
				m.On("RegisterBusinessHealthCheck", mock.Anything).Return(nil).Times(2)
			},
			expectError: false,
		},
		{
			name: "registration failure",
			services: []HTTPServiceDefinition{
				{
					ServiceName:     "service1",
					Registrar:       &mockHTTPHandlerRegistrar{},
					HealthCheckFunc: func(ctx context.Context) error { return nil },
				},
			},
			setupMock: func(m *mockBusinessServiceRegistry) {
				m.On("RegisterBusinessHTTPHandler", mock.Anything).Return(errors.New("registration failed"))
			},
			expectError: true,
			errorMsg:    "failed to register HTTP service service1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &mockBusinessServiceRegistry{}
			tt.setupMock(registry)

			batch := NewServiceRegistrationBatch(registry)
			err := batch.RegisterHTTPServices(tt.services)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}

			registry.AssertExpectations(t)
		})
	}
}

func TestServiceRegistrationBatch_RegisterGRPCServices(t *testing.T) {
	tests := []struct {
		name        string
		services    []GRPCServiceDefinition
		setupMock   func(*mockBusinessServiceRegistry)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful registration",
			services: []GRPCServiceDefinition{
				{
					ServiceName:     "grpc-service1",
					Registrar:       &mockGRPCServiceRegistrar{},
					HealthCheckFunc: func(ctx context.Context) error { return nil },
				},
			},
			setupMock: func(m *mockBusinessServiceRegistry) {
				m.On("RegisterBusinessGRPCService", mock.Anything).Return(nil)
				m.On("RegisterBusinessHealthCheck", mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name: "registration failure",
			services: []GRPCServiceDefinition{
				{
					ServiceName:     "grpc-service1",
					Registrar:       &mockGRPCServiceRegistrar{},
					HealthCheckFunc: func(ctx context.Context) error { return nil },
				},
			},
			setupMock: func(m *mockBusinessServiceRegistry) {
				m.On("RegisterBusinessGRPCService", mock.Anything).Return(errors.New("grpc registration failed"))
			},
			expectError: true,
			errorMsg:    "failed to register gRPC service grpc-service1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &mockBusinessServiceRegistry{}
			tt.setupMock(registry)

			batch := NewServiceRegistrationBatch(registry)
			err := batch.RegisterGRPCServices(tt.services)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}

			registry.AssertExpectations(t)
		})
	}
}

// Test PerformanceOptimizations

func TestNewPerformanceOptimizations(t *testing.T) {
	perf := NewPerformanceOptimizations()
	assert.NotNil(t, perf)
}

func TestPerformanceOptimizations_OptimizeGORMConnection(t *testing.T) {
	perf := NewPerformanceOptimizations()

	tests := []struct {
		name        string
		db          *gorm.DB
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil database",
			db:          nil,
			expectError: true,
			errorMsg:    "database connection is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := perf.OptimizeGORMConnection(tt.db)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Note: OptimizeGORMConnection with valid DB would require actual database setup

func TestPerformanceOptimizations_PreallocateSlices(t *testing.T) {
	perf := NewPerformanceOptimizations()
	slices := perf.PreallocateSlices()

	assert.NotNil(t, slices)
	assert.Contains(t, slices, "small_slice")
	assert.Contains(t, slices, "medium_slice")
	assert.Contains(t, slices, "large_slice")

	// Verify slice capacities
	smallSlice := slices["small_slice"].([]interface{})
	assert.Equal(t, 0, len(smallSlice))
	assert.Equal(t, 10, cap(smallSlice))

	mediumSlice := slices["medium_slice"].([]interface{})
	assert.Equal(t, 0, len(mediumSlice))
	assert.Equal(t, 50, cap(mediumSlice))

	largeSlice := slices["large_slice"].([]interface{})
	assert.Equal(t, 0, len(largeSlice))
	assert.Equal(t, 100, cap(largeSlice))
}

// Test ResourceCleanup

func TestNewResourceCleanup(t *testing.T) {
	cleanup := NewResourceCleanup()
	assert.NotNil(t, cleanup)
}

func TestResourceCleanup_CleanupDatabase(t *testing.T) {
	cleanup := NewResourceCleanup()

	tests := []struct {
		name        string
		db          *gorm.DB
		expectError bool
	}{
		{
			name:        "nil database",
			db:          nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cleanup.CleanupDatabase(tt.db, "test-service")

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Note: CleanupDatabase with valid DB would require actual database setup

func TestResourceCleanup_CleanupResources(t *testing.T) {
	cleanup := NewResourceCleanup()

	tests := []struct {
		name        string
		resources   map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty resources",
			resources:   map[string]interface{}{},
			expectError: false,
		},
		{
			name: "resources without Close method",
			resources: map[string]interface{}{
				"service1": "test-service",
				"service2": 123,
			},
			expectError: false,
		},
		{
			name: "resources with successful Close",
			resources: map[string]interface{}{
				"closer1": &mockCloser{closeError: nil},
				"closer2": &mockCloser{closeError: nil},
			},
			expectError: false,
		},
		{
			name: "resources with Close error",
			resources: map[string]interface{}{
				"closer1": &mockCloser{closeError: errors.New("close failed")},
				"closer2": &mockCloser{closeError: errors.New("another close error")},
			},
			expectError: true,
			errorMsg:    "cleanup errors:",
		},
		{
			name: "mixed resources",
			resources: map[string]interface{}{
				"service":     "test-service",
				"good_closer": &mockCloser{closeError: nil},
				"bad_closer":  &mockCloser{closeError: errors.New("close failed")},
			},
			expectError: true,
			errorMsg:    "cleanup errors:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cleanup.CleanupResources(tt.resources, "test-service")

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

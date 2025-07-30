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
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/transport"
	"github.com/innovationmech/swit/pkg/types"
)

// Mock service that implements multiple interfaces
type mockMultiService struct {
	name string
}

func (m *mockMultiService) GetServiceName() string {
	return m.name
}

func (m *mockMultiService) RegisterRoutes(router gin.IRouter) error {
	return nil
}

func (m *mockMultiService) GetVersion() string {
	return "1.0.0"
}

func (m *mockMultiService) RegisterService(server *grpc.Server) error {
	return nil
}

func (m *mockMultiService) GetServiceDesc() *grpc.ServiceDesc {
	return &grpc.ServiceDesc{
		ServiceName: m.name,
		Methods:     []grpc.MethodDesc{},
		Streams:     []grpc.StreamDesc{},
	}
}

func (m *mockMultiService) GetName() string {
	return m.name
}

func (m *mockMultiService) Check(ctx context.Context) *types.HealthStatus {
	return &types.HealthStatus{
		Status:    types.HealthStatusHealthy,
		Timestamp: time.Now(),
	}
}

func (m *mockMultiService) IsHealthy() bool {
	return true
}

func TestServiceRegistry_AutoRegistration(t *testing.T) {
	transportManager := &transport.Manager{}
	registry := newServiceRegistry(transportManager)

	// Test auto registration of a service that implements multiple interfaces
	service := &mockMultiService{name: "test-service"}
	err := registry.AutoRegisterService(service)
	require.NoError(t, err)

	// Verify service was registered in all categories
	assert.True(t, registry.IsServiceRegistered("http", "test-service"))
	assert.True(t, registry.IsServiceRegistered("grpc", "test-service"))
	assert.Len(t, registry.GetHTTPHandlers(), 1)
	assert.Len(t, registry.GetGRPCServices(), 1)
	assert.Len(t, registry.GetHealthChecks(), 1)

	// Test service counts
	counts := registry.GetServiceCounts()
	assert.Equal(t, 1, counts["http"])
	assert.Equal(t, 1, counts["grpc"])
	assert.Equal(t, 1, counts["healthCheck"])
	assert.Equal(t, 2, counts["total"]) // HTTP + gRPC registrations
}

func TestServiceRegistry_DuplicateRegistrationPrevention(t *testing.T) {
	transportManager := &transport.Manager{}
	registry := newServiceRegistry(transportManager)

	// Register a service
	service1 := &mockMultiService{name: "duplicate-service"}
	err := registry.AutoRegisterService(service1)
	require.NoError(t, err)

	// Try to register another service with the same name
	service2 := &mockMultiService{name: "duplicate-service"}
	err = registry.AutoRegisterService(service2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Verify only one service is registered per type
	counts := registry.GetServiceCounts()
	assert.Equal(t, 2, counts["total"]) // HTTP + gRPC registrations for first service
}

func TestServiceRegistry_ServiceNameValidation(t *testing.T) {
	transportManager := &transport.Manager{}
	registry := newServiceRegistry(transportManager)

	tests := []struct {
		name        string
		serviceName string
		expectError bool
	}{
		{"valid name", "valid-service", false},
		{"empty name", "", true},
		{"name with space", "invalid service", true},
		{"name with tab", "invalid\tservice", true},
		{"name with newline", "invalid\nservice", true},
		{"too long name", string(make([]byte, 65)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockMultiService{name: tt.serviceName}
			err := registry.AutoRegisterService(service)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServiceRegistry_HealthCheckAggregation(t *testing.T) {
	transportManager := &transport.Manager{}
	registry := newServiceRegistry(transportManager)

	// Register multiple services with health checks
	service1 := &mockMultiService{name: "service1"}
	service2 := &mockMultiService{name: "service2"}

	err := registry.AutoRegisterServices(service1, service2)
	require.NoError(t, err)

	// Test aggregated health check
	ctx := context.Background()
	health := registry.CheckAggregatedHealth(ctx)

	assert.NotNil(t, health)
	assert.Equal(t, types.HealthStatusHealthy, health.Status)
	assert.Len(t, health.Dependencies, 2)
	assert.Contains(t, health.Dependencies, "service1")
	assert.Contains(t, health.Dependencies, "service2")
}

func TestServiceRegistry_HealthCheckCaching(t *testing.T) {
	transportManager := &transport.Manager{}
	registry := newServiceRegistry(transportManager)

	service := &mockMultiService{name: "cached-service"}
	err := registry.AutoRegisterService(service)
	require.NoError(t, err)

	ctx := context.Background()

	// First call should perform actual health check
	health1 := registry.CheckAggregatedHealth(ctx)
	assert.NotNil(t, health1)

	// Second call should return cached result
	health2 := registry.CheckAggregatedHealth(ctx)
	assert.NotNil(t, health2)

	// Results should be identical (cached)
	assert.Equal(t, health1.Timestamp, health2.Timestamp)
}

func TestServiceRegistry_BatchAutoRegistration(t *testing.T) {
	transportManager := &transport.Manager{}
	registry := newServiceRegistry(transportManager)

	// Create multiple services
	services := []interface{}{
		&mockMultiService{name: "service1"},
		&mockMultiService{name: "service2"},
		&mockMultiService{name: "service3"},
	}

	// Register all services at once
	err := registry.AutoRegisterServices(services...)
	require.NoError(t, err)

	// Verify all services are registered
	names := registry.GetRegisteredServiceNames()
	assert.Len(t, names["http"], 3)
	assert.Len(t, names["grpc"], 3)
	assert.Contains(t, names["http"], "service1")
	assert.Contains(t, names["http"], "service2")
	assert.Contains(t, names["http"], "service3")
	assert.Contains(t, names["grpc"], "service1")
	assert.Contains(t, names["grpc"], "service2")
	assert.Contains(t, names["grpc"], "service3")

	// Verify service counts
	counts := registry.GetServiceCounts()
	assert.Equal(t, 3, counts["http"])
	assert.Equal(t, 3, counts["grpc"])
	assert.Equal(t, 3, counts["healthCheck"])
	assert.Equal(t, 6, counts["total"]) // 3 HTTP + 3 gRPC registrations
}

func TestServiceRegistry_Clear(t *testing.T) {
	transportManager := &transport.Manager{}
	registry := newServiceRegistry(transportManager)

	// Register some services
	service := &mockMultiService{name: "test-service"}
	err := registry.AutoRegisterService(service)
	require.NoError(t, err)

	// Verify services are registered
	assert.True(t, registry.IsServiceRegistered("http", "test-service"))
	assert.True(t, registry.IsServiceRegistered("grpc", "test-service"))
	// Count() returns HTTP handlers + gRPC services
	assert.Equal(t, 2, registry.Count()) // 1 HTTP + 1 gRPC

	// Verify initial counts
	counts := registry.GetServiceCounts()
	assert.Equal(t, 1, counts["http"])
	assert.Equal(t, 1, counts["grpc"])
	assert.Equal(t, 1, counts["healthCheck"])
	assert.Equal(t, 2, counts["total"]) // HTTP + gRPC registrations

	// Clear registry
	registry.Clear()

	// Verify all services are cleared
	assert.False(t, registry.IsServiceRegistered("http", "test-service"))
	assert.False(t, registry.IsServiceRegistered("grpc", "test-service"))
	assert.Equal(t, 0, registry.Count())
	names := registry.GetRegisteredServiceNames()
	assert.Len(t, names, 0)

	counts = registry.GetServiceCounts()
	assert.Equal(t, 0, counts["total"])
}

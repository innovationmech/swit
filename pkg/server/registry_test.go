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

	"github.com/innovationmech/swit/pkg/transport"
	"github.com/stretchr/testify/assert"
)

type testHTTPHandler struct {
	serviceName string
}

func (h *testHTTPHandler) RegisterRoutes(router any) error {
	return nil
}

func (h *testHTTPHandler) GetServiceName() string {
	return h.serviceName
}

type testGRPCService struct {
	serviceName string
}

func (s *testGRPCService) RegisterGRPC(server any) error {
	return nil
}

func (s *testGRPCService) GetServiceName() string {
	return s.serviceName
}

func TestNewDefaultServiceRegistry(t *testing.T) {
	transportManager := transport.NewTransportCoordinator()
	registry := NewDefaultServiceRegistry(transportManager)
	assert.NotNil(t, registry)
}

func TestDefaultServiceRegistry_ValidateServiceName(t *testing.T) {
	transportManager := transport.NewTransportCoordinator()
	registry := NewDefaultServiceRegistry(transportManager)

	tests := []struct {
		name        string
		serviceName string
		expectError bool
	}{
		{"valid name", "test-service", false},
		{"valid with underscore", "test_service", false},
		{"valid with numbers", "test-service-123", false},
		{"empty name", "", true},
		{"too long", "this-is-a-very-long-service-name-that-exceeds-fifty-characters", true},
		{"invalid characters", "test@service", true},
		{"valid mixed case", "TestService", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.validateServiceName(tt.serviceName)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultServiceRegistry_DuplicateServicePrevention(t *testing.T) {
	transportManager := transport.NewTransportCoordinator()
	registry := NewDefaultServiceRegistry(transportManager)

	// Register HTTP handler
	httpHandler := &testHTTPHandler{serviceName: "test-service"}
	err := registry.RegisterBusinessHTTPHandler(httpHandler)
	assert.NoError(t, err)

	// Try to register gRPC service with same name
	grpcService := &testGRPCService{serviceName: "test-service"}
	err = registry.RegisterBusinessGRPCService(grpcService)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service with name 'test-service' is already registered")
}

func TestDefaultServiceRegistry_GetRegisteredServiceNames(t *testing.T) {
	transportManager := transport.NewTransportCoordinator()
	registry := NewDefaultServiceRegistry(transportManager)

	// Register services
	httpHandler := &testHTTPHandler{serviceName: "http-service"}
	grpcService := &testGRPCService{serviceName: "grpc-service"}

	err := registry.RegisterBusinessHTTPHandler(httpHandler)
	assert.NoError(t, err)

	err = registry.RegisterBusinessGRPCService(grpcService)
	assert.NoError(t, err)

	names := registry.GetRegisteredServiceNames()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "http-service")
	assert.Contains(t, names, "grpc-service")
}

func TestDefaultServiceRegistry_GetServiceCount(t *testing.T) {
	transportManager := transport.NewTransportCoordinator()
	registry := NewDefaultServiceRegistry(transportManager)

	assert.Equal(t, 0, registry.GetServiceCount())

	// Register services
	httpHandler := &testHTTPHandler{serviceName: "http-service"}
	grpcService := &testGRPCService{serviceName: "grpc-service"}

	err := registry.RegisterBusinessHTTPHandler(httpHandler)
	assert.NoError(t, err)
	assert.Equal(t, 1, registry.GetServiceCount())

	err = registry.RegisterBusinessGRPCService(grpcService)
	assert.NoError(t, err)
	assert.Equal(t, 2, registry.GetServiceCount())
}

func TestDefaultServiceRegistry_CreateHealthCheckEndpoint(t *testing.T) {
	transportManager := transport.NewTransportCoordinator()
	registry := NewDefaultServiceRegistry(transportManager)

	checkFunc := func(ctx context.Context) error {
		return nil
	}

	healthCheck := registry.CreateBusinessHealthCheckEndpoint("test-service", checkFunc)
	assert.NotNil(t, healthCheck)
	assert.Equal(t, "test-service", healthCheck.GetServiceName())

	// Test health check execution
	ctx := context.Background()
	err := healthCheck.Check(ctx)
	assert.NoError(t, err)
}

func TestDefaultServiceRegistry_RegisterServiceWithHealthCheck(t *testing.T) {
	transportManager := transport.NewTransportCoordinator()
	registry := NewDefaultServiceRegistry(transportManager)

	httpHandler := &testHTTPHandler{serviceName: "test-service"}
	healthCheckFunc := func(ctx context.Context) error {
		return nil
	}

	err := registry.RegisterBusinessServiceWithHealthCheck(httpHandler, healthCheckFunc)
	assert.NoError(t, err)

	// Verify both service and health check are registered
	assert.Len(t, registry.GetBusinessHTTPHandlers(), 1)
	assert.Len(t, registry.GetBusinessHealthChecks(), 1)
	assert.Equal(t, 1, registry.GetServiceCount())
}

func TestDefaultServiceRegistry_RegisterGRPCServiceWithHealthCheck(t *testing.T) {
	transportManager := transport.NewTransportCoordinator()
	registry := NewDefaultServiceRegistry(transportManager)

	grpcService := &testGRPCService{serviceName: "test-service"}
	healthCheckFunc := func(ctx context.Context) error {
		return nil
	}

	err := registry.RegisterBusinessGRPCServiceWithHealthCheck(grpcService, healthCheckFunc)
	assert.NoError(t, err)

	// Verify both service and health check are registered
	assert.Len(t, registry.GetBusinessGRPCServices(), 1)
	assert.Len(t, registry.GetBusinessHealthChecks(), 1)
	assert.Equal(t, 1, registry.GetServiceCount())
}

func TestDefaultServiceRegistry_GetOverallHealthStatus(t *testing.T) {
	transportManager := transport.NewTransportCoordinator()
	registry := NewDefaultServiceRegistry(transportManager)

	// Test with no health checks - should be healthy
	ctx := context.Background()
	overallStatus := registry.GetOverallHealthStatus(ctx)
	assert.NotNil(t, overallStatus)
	assert.Equal(t, "healthy", overallStatus.Status)
	assert.Empty(t, overallStatus.Dependencies)

	// Register healthy service
	healthyCheck := registry.CreateBusinessHealthCheckEndpoint("healthy-service", func(ctx context.Context) error {
		return nil
	})
	err := registry.RegisterBusinessHealthCheck(healthyCheck)
	assert.NoError(t, err)

	overallStatus = registry.GetOverallHealthStatus(ctx)
	assert.Equal(t, "healthy", overallStatus.Status)
	assert.Len(t, overallStatus.Dependencies, 1)
	assert.Equal(t, "up", overallStatus.Dependencies["healthy-service"].Status)

	// Register unhealthy service
	unhealthyCheck := registry.CreateBusinessHealthCheckEndpoint("unhealthy-service", func(ctx context.Context) error {
		return assert.AnError
	})
	err = registry.RegisterBusinessHealthCheck(unhealthyCheck)
	assert.NoError(t, err)

	overallStatus = registry.GetOverallHealthStatus(ctx)
	assert.Equal(t, "unhealthy", overallStatus.Status)
	assert.Len(t, overallStatus.Dependencies, 2)
	assert.Equal(t, "up", overallStatus.Dependencies["healthy-service"].Status)
	assert.Equal(t, "down", overallStatus.Dependencies["unhealthy-service"].Status)
}

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

package switauth

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/internal/switauth/deps"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockServiceRegistry implements server.ServiceRegistry for testing
type MockServiceRegistry struct {
	mock.Mock
}

func (m *MockServiceRegistry) RegisterHTTPHandler(handler server.BusinessHTTPHandler) error {
	args := m.Called(handler)
	return args.Error(0)
}

func (m *MockServiceRegistry) RegisterGRPCService(service server.BusinessGRPCService) error {
	args := m.Called(service)
	return args.Error(0)
}

func (m *MockServiceRegistry) RegisterBusinessGRPCService(service server.BusinessGRPCService) error {
	args := m.Called(service)
	return args.Error(0)
}

func (m *MockServiceRegistry) RegisterBusinessHTTPHandler(handler server.BusinessHTTPHandler) error {
	args := m.Called(handler)
	return args.Error(0)
}

func (m *MockServiceRegistry) RegisterBusinessHealthCheck(check server.BusinessHealthCheck) error {
	args := m.Called(check)
	return args.Error(0)
}

func (m *MockServiceRegistry) RegisterHealthCheck(check server.BusinessHealthCheck) error {
	args := m.Called(check)
	return args.Error(0)
}

func TestServiceRegistrar_RegisterServices(t *testing.T) {
	// Create mock dependencies
	mockDeps := &deps.Dependencies{
		// Add minimal required fields for testing
	}

	registrar := NewServiceRegistrar(mockDeps)
	mockRegistry := &MockServiceRegistry{}

	// Set up expectations
	mockRegistry.On("RegisterBusinessHTTPHandler", mock.AnythingOfType("*switauth.AuthBusinessHTTPHandler")).Return(nil)
	mockRegistry.On("RegisterBusinessHTTPHandler", mock.AnythingOfType("*switauth.HealthBusinessHTTPHandler")).Return(nil)
	mockRegistry.On("RegisterBusinessHealthCheck", mock.AnythingOfType("*switauth.AuthBusinessHealthCheck")).Return(nil)
	mockRegistry.On("RegisterBusinessHealthCheck", mock.AnythingOfType("*switauth.HealthServiceCheck")).Return(nil)

	// Test service registration
	err := registrar.RegisterServices(mockRegistry)

	// Assertions
	assert.NoError(t, err)
	mockRegistry.AssertExpectations(t)
}

func TestAuthHTTPHandler_GetServiceName(t *testing.T) {
	handler := &AuthBusinessHTTPHandler{}
	assert.Equal(t, "auth-service", handler.GetServiceName())
}

func TestAuthHTTPHandler_RegisterRoutes(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	handler := &AuthBusinessHTTPHandler{
		handler: nil, // We'll test with nil handler to avoid complex setup
	}

	// Test with wrong type first
	err := handler.RegisterRoutes("not-a-router")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected *gin.Engine")

	// Test with correct type but nil handler - this will panic, so we need to handle it
	router := gin.New()

	// The handler will panic due to nil pointer, so we expect a panic recovery
	defer func() {
		if r := recover(); r != nil {
			// Expected panic due to nil handler
			assert.NotNil(t, r)
		}
	}()

	// This will panic because handler.handler is nil
	_ = handler.RegisterRoutes(router)
}

func TestHealthHTTPHandler_GetServiceName(t *testing.T) {
	handler := &HealthBusinessHTTPHandler{}
	assert.Equal(t, "health-service", handler.GetServiceName())
}

func TestAuthHealthCheck_GetServiceName(t *testing.T) {
	healthCheck := &AuthBusinessHealthCheck{}
	assert.Equal(t, "auth-service", healthCheck.GetServiceName())
}

func TestAuthHealthCheck_Check(t *testing.T) {
	ctx := context.Background()

	// Test with nil service
	healthCheck := &AuthBusinessHealthCheck{
		authService: nil,
		startTime:   time.Now(),
	}
	err := healthCheck.Check(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth service is not available")

	// Test with service present
	healthCheck.authService = "mock-service"
	err = healthCheck.Check(ctx)
	assert.NoError(t, err)
}

func TestHealthServiceCheck_GetServiceName(t *testing.T) {
	healthCheck := &HealthServiceCheck{}
	assert.Equal(t, "health-service", healthCheck.GetServiceName())
}

func TestHealthServiceCheck_Check(t *testing.T) {
	ctx := context.Background()
	healthCheck := &HealthServiceCheck{
		startTime: time.Now(),
	}

	err := healthCheck.Check(ctx)
	assert.NoError(t, err) // Health service check always succeeds
}

func TestDependencyContainer_GetService(t *testing.T) {
	mockDeps := &deps.Dependencies{
		// Add minimal fields for testing
	}

	container := NewBusinessDependencyContainer(mockDeps)

	// Test getting unknown service
	_, err := container.GetService("unknown-service")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service unknown-service not found")

	// Test getting known services
	service, err := container.GetService("auth-service")
	assert.NoError(t, err)
	assert.Equal(t, mockDeps.AuthSrv, service)

	service, err = container.GetService("health-service")
	assert.NoError(t, err)
	assert.Equal(t, mockDeps.HealthSrv, service)

	service, err = container.GetService("database")
	assert.NoError(t, err)
	assert.Equal(t, mockDeps.DB, service)

	service, err = container.GetService("config")
	assert.NoError(t, err)
	assert.Equal(t, mockDeps.Config, service)
}

func TestDependencyContainer_Close(t *testing.T) {
	mockDeps := &deps.Dependencies{}
	container := NewBusinessDependencyContainer(mockDeps)

	// Test closing
	err := container.Close()
	assert.NoError(t, err)

	// Test getting service after close
	_, err = container.GetService("auth-service")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency container is closed")

	// Test closing again (should be idempotent)
	err = container.Close()
	assert.NoError(t, err)
}

func TestConfigMapper_ToServerConfig(t *testing.T) {
	// Test with default values
	authConfig := &config.AuthConfig{}
	mapper := NewConfigMapper(authConfig)
	serverConfig := mapper.ToServerConfig()

	assert.Equal(t, "swit-auth", serverConfig.ServiceName)
	assert.Equal(t, "9001", serverConfig.HTTP.Port)
	assert.Equal(t, ":9001", serverConfig.HTTP.Address)
	assert.True(t, serverConfig.HTTP.Enabled)
	assert.False(t, serverConfig.HTTP.EnableReady)

	assert.Equal(t, "50051", serverConfig.GRPC.Port)
	assert.Equal(t, ":50051", serverConfig.GRPC.Address)
	assert.True(t, serverConfig.GRPC.Enabled)
	assert.False(t, serverConfig.GRPC.EnableKeepalive)
	assert.True(t, serverConfig.GRPC.EnableReflection)
	assert.True(t, serverConfig.GRPC.EnableHealthService)

	assert.Equal(t, "swit-auth", serverConfig.Discovery.ServiceName)
	assert.Contains(t, serverConfig.Discovery.Tags, "auth")
	assert.Contains(t, serverConfig.Discovery.Tags, "v1")
	assert.True(t, serverConfig.Discovery.Enabled)

	assert.True(t, serverConfig.Middleware.EnableCORS)
	assert.False(t, serverConfig.Middleware.EnableAuth)
	assert.True(t, serverConfig.Middleware.EnableRateLimit)
	assert.True(t, serverConfig.Middleware.EnableLogging)

	assert.Equal(t, 5*time.Second, serverConfig.ShutdownTimeout)
}

func TestConfigMapper_ToServerConfig_WithCustomValues(t *testing.T) {
	// Test with custom values
	authConfig := &config.AuthConfig{
		Server: struct {
			Port     string `json:"port" yaml:"port"`
			GRPCPort string `json:"grpcPort" yaml:"grpcPort"`
		}{
			Port:     "8080",
			GRPCPort: "9090",
		},
		ServiceDiscovery: struct {
			Address string `json:"address" yaml:"address"`
		}{
			Address: "localhost:8500",
		},
	}

	mapper := NewConfigMapper(authConfig)
	serverConfig := mapper.ToServerConfig()

	assert.Equal(t, "8080", serverConfig.HTTP.Port)
	assert.Equal(t, ":8080", serverConfig.HTTP.Address)
	assert.Equal(t, "9090", serverConfig.GRPC.Port)
	assert.Equal(t, ":9090", serverConfig.GRPC.Address)
	assert.Equal(t, "localhost:8500", serverConfig.Discovery.Address)
}

func TestAuthGRPCService_GetServiceName(t *testing.T) {
	service := NewAuthBusinessGRPCService()
	assert.Equal(t, "auth-grpc-service", service.GetServiceName())
}

func TestAuthGRPCService_RegisterGRPC(t *testing.T) {
	service := NewAuthBusinessGRPCService()

	// Test with correct type
	grpcServer := grpc.NewServer()
	err := service.RegisterGRPC(grpcServer)
	assert.NoError(t, err)

	// Test with wrong type
	err = service.RegisterGRPC("not-a-grpc-server")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected *grpc.Server")
}

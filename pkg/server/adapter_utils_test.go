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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// Mock implementations for adapter testing

type mockHTTPHandlerRegistrar struct {
	mock.Mock
}

func (m *mockHTTPHandlerRegistrar) RegisterHTTP(router *gin.Engine) error {
	args := m.Called(router)
	return args.Error(0)
}

type mockGRPCServiceRegistrar struct {
	mock.Mock
}

func (m *mockGRPCServiceRegistrar) RegisterGRPC(server *grpc.Server) error {
	args := m.Called(server)
	return args.Error(0)
}

type mockBusinessServiceRegistry struct {
	mock.Mock
}

func (m *mockBusinessServiceRegistry) RegisterBusinessHTTPHandler(handler BusinessHTTPHandler) error {
	args := m.Called(handler)
	return args.Error(0)
}

func (m *mockBusinessServiceRegistry) RegisterBusinessGRPCService(service BusinessGRPCService) error {
	args := m.Called(service)
	return args.Error(0)
}

func (m *mockBusinessServiceRegistry) RegisterBusinessHealthCheck(healthCheck BusinessHealthCheck) error {
	args := m.Called(healthCheck)
	return args.Error(0)
}

// Test GenericHTTPHandler

func TestNewGenericHTTPHandler(t *testing.T) {
	serviceName := "test-service"
	registrar := &mockHTTPHandlerRegistrar{}

	handler := NewGenericHTTPHandler(serviceName, registrar)

	assert.NotNil(t, handler)
	assert.Equal(t, serviceName, handler.serviceName)
	assert.Equal(t, registrar, handler.registrar)
}

func TestGenericHTTPHandler_RegisterRoutes(t *testing.T) {
	tests := []struct {
		name        string
		router      interface{}
		setupMock   func(*mockHTTPHandlerRegistrar)
		expectError bool
		errorMsg    string
	}{
		{
			name:   "successful registration",
			router: gin.New(),
			setupMock: func(m *mockHTTPHandlerRegistrar) {
				m.On("RegisterHTTP", mock.AnythingOfType("*gin.Engine")).Return(nil)
			},
			expectError: false,
		},
		{
			name:   "registrar error",
			router: gin.New(),
			setupMock: func(m *mockHTTPHandlerRegistrar) {
				m.On("RegisterHTTP", mock.AnythingOfType("*gin.Engine")).Return(errors.New("registration failed"))
			},
			expectError: true,
			errorMsg:    "registration failed",
		},
		{
			name:        "invalid router type",
			router:      "not-a-gin-engine",
			setupMock:   func(m *mockHTTPHandlerRegistrar) {},
			expectError: true,
			errorMsg:    "expected *gin.Engine, got string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrar := &mockHTTPHandlerRegistrar{}
			tt.setupMock(registrar)

			handler := NewGenericHTTPHandler("test-service", registrar)
			err := handler.RegisterRoutes(tt.router)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}

			registrar.AssertExpectations(t)
		})
	}
}

func TestGenericHTTPHandler_GetServiceName(t *testing.T) {
	serviceName := "test-service"
	registrar := &mockHTTPHandlerRegistrar{}
	handler := NewGenericHTTPHandler(serviceName, registrar)

	result := handler.GetServiceName()
	assert.Equal(t, serviceName, result)
}

// Test GenericGRPCService

func TestNewGenericGRPCService(t *testing.T) {
	serviceName := "test-grpc-service"
	registrar := &mockGRPCServiceRegistrar{}

	service := NewGenericGRPCService(serviceName, registrar)

	assert.NotNil(t, service)
	assert.Equal(t, serviceName, service.serviceName)
	assert.Equal(t, registrar, service.registrar)
}

func TestGenericGRPCService_RegisterGRPC(t *testing.T) {
	tests := []struct {
		name        string
		server      interface{}
		setupMock   func(*mockGRPCServiceRegistrar)
		expectError bool
		errorMsg    string
	}{
		{
			name:   "successful registration",
			server: grpc.NewServer(),
			setupMock: func(m *mockGRPCServiceRegistrar) {
				m.On("RegisterGRPC", mock.AnythingOfType("*grpc.Server")).Return(nil)
			},
			expectError: false,
		},
		{
			name:   "registrar error",
			server: grpc.NewServer(),
			setupMock: func(m *mockGRPCServiceRegistrar) {
				m.On("RegisterGRPC", mock.AnythingOfType("*grpc.Server")).Return(errors.New("grpc registration failed"))
			},
			expectError: true,
			errorMsg:    "grpc registration failed",
		},
		{
			name:        "invalid server type",
			server:      "not-a-grpc-server",
			setupMock:   func(m *mockGRPCServiceRegistrar) {},
			expectError: true,
			errorMsg:    "expected *grpc.Server, got string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrar := &mockGRPCServiceRegistrar{}
			tt.setupMock(registrar)

			service := NewGenericGRPCService("test-service", registrar)
			err := service.RegisterGRPC(tt.server)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}

			registrar.AssertExpectations(t)
		})
	}
}

func TestGenericGRPCService_GetServiceName(t *testing.T) {
	serviceName := "test-grpc-service"
	registrar := &mockGRPCServiceRegistrar{}
	service := NewGenericGRPCService(serviceName, registrar)

	result := service.GetServiceName()
	assert.Equal(t, serviceName, result)
}

// Test GenericHealthCheck

func TestNewGenericHealthCheck(t *testing.T) {
	serviceName := "test-health-service"
	checkFunc := func(ctx context.Context) error { return nil }

	start := time.Now()
	healthCheck := NewGenericHealthCheck(serviceName, checkFunc)

	assert.NotNil(t, healthCheck)
	assert.Equal(t, serviceName, healthCheck.serviceName)
	assert.NotNil(t, healthCheck.checkFunc)
	assert.True(t, healthCheck.startTime.After(start) || healthCheck.startTime.Equal(start))
}

func TestGenericHealthCheck_Check(t *testing.T) {
	tests := []struct {
		name        string
		checkFunc   HealthCheckFunc
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil check function - default healthy",
			checkFunc:   nil,
			expectError: false,
		},
		{
			name: "successful check function",
			checkFunc: func(ctx context.Context) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "failing check function",
			checkFunc: func(ctx context.Context) error {
				return errors.New("health check failed")
			},
			expectError: true,
			errorMsg:    "health check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthCheck := NewGenericHealthCheck("test-service", tt.checkFunc)
			ctx := context.Background()

			err := healthCheck.Check(ctx)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGenericHealthCheck_GetServiceName(t *testing.T) {
	serviceName := "test-health-service"
	healthCheck := NewGenericHealthCheck(serviceName, nil)

	result := healthCheck.GetServiceName()
	assert.Equal(t, serviceName, result)
}

func TestGenericHealthCheck_GetUptime(t *testing.T) {
	healthCheck := NewGenericHealthCheck("test-service", nil)

	// Wait a small amount to ensure uptime is measurable
	time.Sleep(1 * time.Millisecond)

	uptime := healthCheck.GetUptime()
	assert.True(t, uptime > 0)
	assert.True(t, uptime < 1*time.Second) // Should be very small for this test
}

// Test ServiceAvailabilityChecker

func TestServiceAvailabilityChecker(t *testing.T) {
	tests := []struct {
		name        string
		service     interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil service",
			service:     nil,
			expectError: true,
			errorMsg:    "service is not available",
		},
		{
			name:        "valid service",
			service:     "some-service",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkFunc := ServiceAvailabilityChecker(tt.service)
			ctx := context.Background()

			err := checkFunc(ctx)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test ConfigMapperBase

func TestNewConfigMapperBase(t *testing.T) {
	mapper := NewConfigMapperBase()
	assert.NotNil(t, mapper)
}

func TestConfigMapperBase_SetDefaultHTTPConfig(t *testing.T) {
	tests := []struct {
		name         string
		port         string
		defaultPort  string
		enableReady  bool
		expectedPort string
		expectedAddr string
	}{
		{
			name:         "custom port",
			port:         "8090",
			defaultPort:  "8080",
			enableReady:  true,
			expectedPort: "8090",
			expectedAddr: ":8090",
		},
		{
			name:         "default port when empty",
			port:         "",
			defaultPort:  "8080",
			enableReady:  false,
			expectedPort: "8080",
			expectedAddr: ":8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewConfigMapperBase()
			config := &ServerConfig{}

			mapper.SetDefaultHTTPConfig(config, tt.port, tt.defaultPort, tt.enableReady)

			assert.Equal(t, tt.expectedPort, config.HTTP.Port)
			assert.Equal(t, tt.expectedAddr, config.HTTP.Address)
			assert.True(t, config.HTTP.Enabled)
			assert.Equal(t, tt.enableReady, config.HTTP.EnableReady)
		})
	}
}

func TestConfigMapperBase_SetDefaultGRPCConfig(t *testing.T) {
	tests := []struct {
		name                string
		port                string
		defaultPort         string
		enableKeepalive     bool
		enableReflection    bool
		enableHealthService bool
		expectedPort        string
		expectedAddr        string
	}{
		{
			name:                "custom port with all features",
			port:                "9090",
			defaultPort:         "9080",
			enableKeepalive:     true,
			enableReflection:    true,
			enableHealthService: true,
			expectedPort:        "9090",
			expectedAddr:        ":9090",
		},
		{
			name:                "default port when empty",
			port:                "",
			defaultPort:         "9080",
			enableKeepalive:     false,
			enableReflection:    false,
			enableHealthService: false,
			expectedPort:        "9080",
			expectedAddr:        ":9080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewConfigMapperBase()
			config := &ServerConfig{}

			mapper.SetDefaultGRPCConfig(config, tt.port, tt.defaultPort,
				tt.enableKeepalive, tt.enableReflection, tt.enableHealthService)

			assert.Equal(t, tt.expectedPort, config.GRPC.Port)
			assert.Equal(t, tt.expectedAddr, config.GRPC.Address)
			assert.True(t, config.GRPC.Enabled)
			assert.Equal(t, tt.enableKeepalive, config.GRPC.EnableKeepalive)
			assert.Equal(t, tt.enableReflection, config.GRPC.EnableReflection)
			assert.Equal(t, tt.enableHealthService, config.GRPC.EnableHealthService)
		})
	}
}

func TestConfigMapperBase_SetDefaultDiscoveryConfig(t *testing.T) {
	tests := []struct {
		name         string
		address      string
		serviceName  string
		tags         []string
		expectedAddr string
	}{
		{
			name:         "with custom address",
			address:      "consul:8500",
			serviceName:  "test-service",
			tags:         []string{"api", "v1"},
			expectedAddr: "consul:8500",
		},
		{
			name:         "empty address",
			address:      "",
			serviceName:  "test-service",
			tags:         []string{"api"},
			expectedAddr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewConfigMapperBase()
			config := &ServerConfig{}

			mapper.SetDefaultDiscoveryConfig(config, tt.address, tt.serviceName, tt.tags)

			assert.Equal(t, tt.expectedAddr, config.Discovery.Address)
			assert.Equal(t, tt.serviceName, config.Discovery.ServiceName)
			assert.Equal(t, tt.tags, config.Discovery.Tags)
			assert.True(t, config.Discovery.Enabled)
		})
	}
}

func TestConfigMapperBase_SetDefaultMiddlewareConfig(t *testing.T) {
	mapper := NewConfigMapperBase()
	config := &ServerConfig{}

	mapper.SetDefaultMiddlewareConfig(config, true, false, true, false)

	// Check global middleware
	assert.True(t, config.Middleware.EnableCORS)
	assert.False(t, config.Middleware.EnableAuth)
	assert.True(t, config.Middleware.EnableRateLimit)
	assert.False(t, config.Middleware.EnableLogging)

	// Check HTTP-specific middleware
	assert.True(t, config.HTTP.Middleware.EnableCORS)
	assert.False(t, config.HTTP.Middleware.EnableAuth)
	assert.True(t, config.HTTP.Middleware.EnableRateLimit)
	assert.False(t, config.HTTP.Middleware.EnableLogging)
}

func TestConfigMapperBase_SetDefaultServerConfig(t *testing.T) {
	mapper := NewConfigMapperBase()
	config := &ServerConfig{}
	serviceName := "test-service"

	mapper.SetDefaultServerConfig(config, serviceName)

	assert.Equal(t, serviceName, config.ServiceName)
	assert.Equal(t, 5*time.Second, config.ShutdownTimeout)
}

// Test GenericDependencyContainer

func TestNewGenericDependencyContainer(t *testing.T) {
	closeFunc := func() error { return nil }
	container := NewGenericDependencyContainer(closeFunc)

	assert.NotNil(t, container)
	assert.NotNil(t, container.services)
	assert.False(t, container.closed)
	assert.NotNil(t, container.closeFunc)
}

func TestGenericDependencyContainer_RegisterService(t *testing.T) {
	container := NewGenericDependencyContainer(nil)
	service := "test-service"

	container.RegisterService("test", service)

	retrievedService, err := container.GetService("test")
	require.NoError(t, err)
	assert.Equal(t, service, retrievedService)
}

func TestGenericDependencyContainer_GetService(t *testing.T) {
	tests := []struct {
		name         string
		setupService bool
		serviceName  string
		closed       bool
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "successful retrieval",
			setupService: true,
			serviceName:  "test",
			closed:       false,
			expectError:  false,
		},
		{
			name:         "service not found",
			setupService: false,
			serviceName:  "nonexistent",
			closed:       false,
			expectError:  true,
			errorMsg:     "service nonexistent not found",
		},
		{
			name:         "container closed",
			setupService: true,
			serviceName:  "test",
			closed:       true,
			expectError:  true,
			errorMsg:     "dependency container is closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := NewGenericDependencyContainer(nil)

			if tt.setupService {
				container.RegisterService(tt.serviceName, "test-service")
			}

			if tt.closed {
				container.closed = true
			}

			service, err := container.GetService(tt.serviceName)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, service)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestGenericDependencyContainer_Close(t *testing.T) {
	tests := []struct {
		name          string
		closeFunc     func() error
		alreadyClosed bool
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "successful close",
			closeFunc:     func() error { return nil },
			alreadyClosed: false,
			expectError:   false,
		},
		{
			name:          "close function error",
			closeFunc:     func() error { return errors.New("close error") },
			alreadyClosed: false,
			expectError:   true,
			errorMsg:      "close error",
		},
		{
			name:          "already closed",
			closeFunc:     func() error { return nil },
			alreadyClosed: true,
			expectError:   false,
		},
		{
			name:          "nil close function",
			closeFunc:     nil,
			alreadyClosed: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := NewGenericDependencyContainer(tt.closeFunc)

			if tt.alreadyClosed {
				container.closed = true
			}

			err := container.Close()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}

			assert.True(t, container.closed)
		})
	}
}

// Test ServiceRegistrationHelper

func TestNewServiceRegistrationHelper(t *testing.T) {
	registry := &mockBusinessServiceRegistry{}
	helper := NewServiceRegistrationHelper(registry)

	assert.NotNil(t, helper)
	assert.Equal(t, registry, helper.registry)
}

func TestServiceRegistrationHelper_RegisterHTTPHandlerWithHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mockBusinessServiceRegistry)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful registration",
			setupMock: func(m *mockBusinessServiceRegistry) {
				m.On("RegisterBusinessHTTPHandler", mock.Anything).Return(nil)
				m.On("RegisterBusinessHealthCheck", mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name: "HTTP handler registration fails",
			setupMock: func(m *mockBusinessServiceRegistry) {
				m.On("RegisterBusinessHTTPHandler", mock.Anything).Return(errors.New("http registration failed"))
			},
			expectError: true,
			errorMsg:    "failed to register HTTP handler for test-service: http registration failed",
		},
		{
			name: "health check registration fails",
			setupMock: func(m *mockBusinessServiceRegistry) {
				m.On("RegisterBusinessHTTPHandler", mock.Anything).Return(nil)
				m.On("RegisterBusinessHealthCheck", mock.Anything).Return(errors.New("health check registration failed"))
			},
			expectError: true,
			errorMsg:    "failed to register health check for test-service: health check registration failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &mockBusinessServiceRegistry{}
			tt.setupMock(registry)

			helper := NewServiceRegistrationHelper(registry)
			registrar := &mockHTTPHandlerRegistrar{}
			healthCheckFunc := func(ctx context.Context) error { return nil }

			err := helper.RegisterHTTPHandlerWithHealthCheck("test-service", registrar, healthCheckFunc)

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

func TestServiceRegistrationHelper_RegisterGRPCServiceWithHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mockBusinessServiceRegistry)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful registration",
			setupMock: func(m *mockBusinessServiceRegistry) {
				m.On("RegisterBusinessGRPCService", mock.Anything).Return(nil)
				m.On("RegisterBusinessHealthCheck", mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name: "gRPC service registration fails",
			setupMock: func(m *mockBusinessServiceRegistry) {
				m.On("RegisterBusinessGRPCService", mock.Anything).Return(errors.New("grpc registration failed"))
			},
			expectError: true,
			errorMsg:    "failed to register gRPC service for test-service: grpc registration failed",
		},
		{
			name: "health check registration fails",
			setupMock: func(m *mockBusinessServiceRegistry) {
				m.On("RegisterBusinessGRPCService", mock.Anything).Return(nil)
				m.On("RegisterBusinessHealthCheck", mock.Anything).Return(errors.New("health check registration failed"))
			},
			expectError: true,
			errorMsg:    "failed to register health check for test-service: health check registration failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &mockBusinessServiceRegistry{}
			tt.setupMock(registry)

			helper := NewServiceRegistrationHelper(registry)
			registrar := &mockGRPCServiceRegistrar{}
			healthCheckFunc := func(ctx context.Context) error { return nil }

			err := helper.RegisterGRPCServiceWithHealthCheck("test-service", registrar, healthCheckFunc)

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

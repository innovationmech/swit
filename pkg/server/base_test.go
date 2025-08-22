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
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() {
	// Initialize logger for tests
	logger.InitLogger()
	// Set gin to test mode to reduce noise
	gin.SetMode(gin.TestMode)
}

// Mock implementations for testing

type mockBusinessServiceRegistrar struct {
	mock.Mock
}

func (m *mockBusinessServiceRegistrar) RegisterServices(registry BusinessServiceRegistry) error {
	args := m.Called(registry)
	return args.Error(0)
}

type mockBusinessDependencyContainer struct {
	mock.Mock
}

func (m *mockBusinessDependencyContainer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockBusinessDependencyContainer) GetService(name string) (interface{}, error) {
	args := m.Called(name)
	return args.Get(0), args.Error(1)
}

type mockHTTPHandler struct {
	mock.Mock
}

func (m *mockHTTPHandler) RegisterRoutes(router interface{}) error {
	args := m.Called(router)
	return args.Error(0)
}

func (m *mockHTTPHandler) GetServiceName() string {
	args := m.Called()
	return args.String(0)
}

type mockGRPCService struct {
	mock.Mock
}

func (m *mockGRPCService) RegisterGRPC(server interface{}) error {
	args := m.Called(server)
	return args.Error(0)
}

func (m *mockGRPCService) GetServiceName() string {
	args := m.Called()
	return args.String(0)
}

type mockHealthCheck struct {
	mock.Mock
}

func (m *mockHealthCheck) Check(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockHealthCheck) GetServiceName() string {
	args := m.Called()
	return args.String(0)
}

func TestNewBaseServer(t *testing.T) {
	tests := []struct {
		name        string
		config      *ServerConfig
		registrar   BusinessServiceRegistrar
		deps        BusinessDependencyContainer
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			registrar:   &mockBusinessServiceRegistrar{},
			deps:        &mockBusinessDependencyContainer{},
			expectError: true,
			errorMsg:    "server config cannot be nil",
		},
		{
			name:        "nil registrar",
			config:      NewServerConfig(),
			registrar:   nil,
			deps:        &mockBusinessDependencyContainer{},
			expectError: true,
			errorMsg:    "service registrar cannot be nil",
		},
		{
			name: "invalid config",
			config: &ServerConfig{
				ServiceName: "", // Invalid: empty service name
			},
			registrar:   &mockBusinessServiceRegistrar{},
			deps:        &mockBusinessDependencyContainer{},
			expectError: true,
			errorMsg:    "invalid server configuration",
		},
		{
			name:      "valid configuration",
			config:    NewServerConfig(),
			registrar: &mockBusinessServiceRegistrar{},
			deps:      &mockBusinessDependencyContainer{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			if mockReg, ok := tt.registrar.(*mockBusinessServiceRegistrar); ok {
				mockReg.On("RegisterServices", mock.Anything).Return(nil)
			}
			if mockDeps, ok := tt.deps.(*mockBusinessDependencyContainer); ok {
				// Mock GetService call for logger (return nil to indicate no logger service)
				mockDeps.On("GetService", "logger").Return(nil, fmt.Errorf("service not found"))
			}

			server, err := NewBusinessServerCore(tt.config, tt.registrar, tt.deps)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, server)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, server)
				assert.Equal(t, tt.config, server.config)
				assert.Equal(t, tt.registrar, server.serviceRegistrar)
				assert.Equal(t, tt.deps, server.dependencies)
				assert.NotNil(t, server.transportManager)
			}
		})
	}
}

func TestBaseServer_TransportInitialization(t *testing.T) {
	tests := []struct {
		name             string
		config           *ServerConfig
		expectHTTP       bool
		expectGRPC       bool
		expectedHTTPAddr string
		expectedGRPCAddr string
	}{
		{
			name: "both transports enabled",
			config: &ServerConfig{
				ServiceName: "test-service",
				HTTP: HTTPConfig{
					Port:         "8080",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				GRPC: GRPCConfig{
					Port:           "9080",
					Enabled:        true,
					MaxRecvMsgSize: 4 * 1024 * 1024,
					MaxSendMsgSize: 4 * 1024 * 1024,
				},
				Discovery:       DiscoveryConfig{Enabled: false},
				ShutdownTimeout: 5 * time.Second,
			},
			expectHTTP:       true,
			expectGRPC:       true,
			expectedHTTPAddr: ":8080",
			expectedGRPCAddr: ":9080",
		},
		{
			name: "only HTTP enabled",
			config: &ServerConfig{
				ServiceName: "test-service",
				HTTP: HTTPConfig{
					Port:         "8080",
					Enabled:      true,
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  120 * time.Second,
				},
				GRPC:            GRPCConfig{Enabled: false},
				Discovery:       DiscoveryConfig{Enabled: false},
				ShutdownTimeout: 5 * time.Second,
			},
			expectHTTP:       true,
			expectGRPC:       false,
			expectedHTTPAddr: ":8080",
			expectedGRPCAddr: "",
		},
		{
			name: "only gRPC enabled",
			config: &ServerConfig{
				ServiceName: "test-service",
				HTTP:        HTTPConfig{Enabled: false},
				GRPC: GRPCConfig{
					Port:           "9080",
					Enabled:        true,
					MaxRecvMsgSize: 4 * 1024 * 1024,
					MaxSendMsgSize: 4 * 1024 * 1024,
				},
				Discovery:       DiscoveryConfig{Enabled: false},
				ShutdownTimeout: 5 * time.Second,
			},
			expectHTTP:       false,
			expectGRPC:       true,
			expectedHTTPAddr: "",
			expectedGRPCAddr: ":9080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRegistrar := &mockBusinessServiceRegistrar{}
			mockRegistrar.On("RegisterServices", mock.Anything).Return(nil)

			mockDeps := &mockBusinessDependencyContainer{}

			server, err := NewBusinessServerCore(tt.config, mockRegistrar, mockDeps)
			require.NoError(t, err)

			if tt.expectHTTP {
				assert.NotNil(t, server.httpTransport)
				assert.Equal(t, tt.expectedHTTPAddr, server.GetHTTPAddress())
			} else {
				assert.Nil(t, server.httpTransport)
				assert.Equal(t, "", server.GetHTTPAddress())
			}

			if tt.expectGRPC {
				assert.NotNil(t, server.grpcTransport)
				assert.Equal(t, tt.expectedGRPCAddr, server.GetGRPCAddress())
			} else {
				assert.Nil(t, server.grpcTransport)
				assert.Equal(t, "", server.GetGRPCAddress())
			}
		})
	}
}

func TestBaseServer_StartStop(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false // Disable discovery for simpler testing

	mockRegistrar := &mockBusinessServiceRegistrar{}
	mockRegistrar.On("RegisterServices", mock.Anything).Return(nil)

	mockDeps := &mockBusinessDependencyContainer{}
	mockDeps.On("Close").Return(nil)

	server, err := NewBusinessServerCore(config, mockRegistrar, mockDeps)
	require.NoError(t, err)

	ctx := context.Background()

	// Test starting the server
	err = server.Start(ctx)
	require.NoError(t, err)
	assert.True(t, server.started)

	// Test starting already started server
	err = server.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server is already started")

	// Test stopping the server
	err = server.Stop(ctx)
	require.NoError(t, err)
	assert.False(t, server.started)

	// Test stopping already stopped server (should not error)
	err = server.Stop(ctx)
	require.NoError(t, err)
}

func TestBaseServer_Shutdown(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false

	mockRegistrar := &mockBusinessServiceRegistrar{}
	mockRegistrar.On("RegisterServices", mock.Anything).Return(nil)

	mockDeps := &mockBusinessDependencyContainer{}
	mockDeps.On("Close").Return(nil)

	server, err := NewBusinessServerCore(config, mockRegistrar, mockDeps)
	require.NoError(t, err)

	// Start the server
	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)

	// Test shutdown
	err = server.Shutdown()
	require.NoError(t, err)
	assert.False(t, server.started)

	// Verify dependencies were closed
	mockDeps.AssertExpectations(t)
}

func TestServiceRegistryAdapter(t *testing.T) {
	// This test verifies the adapter pattern works correctly
	config := NewServerConfig()
	config.Discovery.Enabled = false

	mockRegistrar := &mockBusinessServiceRegistrar{}
	mockDeps := &mockBusinessDependencyContainer{}

	// Setup the registrar to just return nil for simplicity
	mockRegistrar.On("RegisterServices", mock.Anything).Return(nil)

	server, err := NewBusinessServerCore(config, mockRegistrar, mockDeps)
	require.NoError(t, err)
	assert.NotNil(t, server)

	// Verify that the service registrar was called
	mockRegistrar.AssertExpectations(t)
}

func TestBaseServer_TransportStatus(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false

	mockRegistrar := &mockBusinessServiceRegistrar{}
	mockRegistrar.On("RegisterServices", mock.Anything).Return(nil)

	mockDeps := &mockBusinessDependencyContainer{}

	server, err := NewBusinessServerCore(config, mockRegistrar, mockDeps)
	require.NoError(t, err)

	// Get transport status
	status := server.GetTransportStatus()

	// Should have both HTTP and gRPC transports
	assert.Len(t, status, 2)

	// Check HTTP transport status
	httpStatus, exists := status["http"]
	assert.True(t, exists)
	assert.Equal(t, "http", httpStatus.Name)
	assert.Equal(t, ":8080", httpStatus.Address)
	assert.False(t, httpStatus.Running) // Not started yet

	// Check gRPC transport status
	grpcStatus, exists := status["grpc"]
	assert.True(t, exists)
	assert.Equal(t, "grpc", grpcStatus.Name)
	assert.Equal(t, ":9080", grpcStatus.Address)
	assert.False(t, grpcStatus.Running) // Not started yet
}

func TestBaseServer_TransportHealth(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false

	mockRegistrar := &mockBusinessServiceRegistrar{}
	mockRegistrar.On("RegisterServices", mock.Anything).Return(nil)

	mockDeps := &mockBusinessDependencyContainer{}

	server, err := NewBusinessServerCore(config, mockRegistrar, mockDeps)
	require.NoError(t, err)

	ctx := context.Background()

	// Get transport health
	health := server.GetTransportHealth(ctx)

	// Should return health status map (may be empty if no services are registered)
	assert.NotNil(t, health)
}

// Test missing coverage methods

func TestBaseServer_GetPerformanceMonitor(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false

	mockRegistrar := &mockBusinessServiceRegistrar{}
	mockRegistrar.On("RegisterServices", mock.Anything).Return(nil)

	mockDeps := &mockBusinessDependencyContainer{}

	server, err := NewBusinessServerCore(config, mockRegistrar, mockDeps)
	require.NoError(t, err)

	monitor := server.GetPerformanceMonitor()
	assert.NotNil(t, monitor)
}

func TestBaseServer_CastToBusinessServerWithPerformance(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false

	mockRegistrar := &mockBusinessServiceRegistrar{}
	mockRegistrar.On("RegisterServices", mock.Anything).Return(nil)

	mockDeps := &mockBusinessDependencyContainer{}

	server, err := NewBusinessServerCore(config, mockRegistrar, mockDeps)
	require.NoError(t, err)

	// Test that server has performance methods
	metrics := server.GetPerformanceMetrics()
	assert.NotNil(t, metrics)

	monitor := server.GetPerformanceMonitor()
	assert.NotNil(t, monitor)
}

func TestBaseServer_ServiceRegistration_ErrorPaths(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false

	// Test with a registrar that returns an error
	errorRegistrar := &mockBusinessServiceRegistrar{}
	errorRegistrar.On("RegisterServices", mock.Anything).Return(fmt.Errorf("registration error"))

	mockDeps := &mockBusinessDependencyContainer{}

	_, err := NewBusinessServerCore(config, errorRegistrar, mockDeps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register services")
}

func TestBaseServer_InterfaceCompliance(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false

	mockRegistrar := &mockBusinessServiceRegistrar{}
	mockRegistrar.On("RegisterServices", mock.Anything).Return(nil)

	mockDeps := &mockBusinessDependencyContainer{}

	server, err := NewBusinessServerCore(config, mockRegistrar, mockDeps)
	require.NoError(t, err)

	// Verify server was created successfully and has expected interface methods
	assert.NotNil(t, server)

	// Test basic interface methods
	assert.Equal(t, ":8080", server.GetHTTPAddress())
	assert.Equal(t, ":9080", server.GetGRPCAddress())

	transports := server.GetTransports()
	assert.NotNil(t, transports)

	status := server.GetTransportStatus()
	assert.NotNil(t, status)

	ctx := context.Background()
	health := server.GetTransportHealth(ctx)
	assert.NotNil(t, health)
}

func TestBaseServer_DiscoveryRegistration_ErrorPaths(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = true
	config.Discovery.Address = "invalid-consul-address"
	config.Discovery.ServiceName = "test-service"

	mockRegistrar := &mockBusinessServiceRegistrar{}
	mockRegistrar.On("RegisterServices", mock.Anything).Return(nil)

	mockDeps := &mockBusinessDependencyContainer{}

	server, err := NewBusinessServerCore(config, mockRegistrar, mockDeps)
	require.NoError(t, err)

	ctx := context.Background()

	// Starting with invalid discovery should not fail due to graceful failure mode
	err = server.Start(ctx)
	require.NoError(t, err)

	err = server.Stop(ctx)
	require.NoError(t, err)
}

func TestBaseServer_Uptime(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false

	mockRegistrar := &mockBusinessServiceRegistrar{}
	mockRegistrar.On("RegisterServices", mock.Anything).Return(nil)

	mockDeps := &mockBusinessDependencyContainer{}

	server, err := NewBusinessServerCore(config, mockRegistrar, mockDeps)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)

	// Wait a tiny bit and check uptime
	time.Sleep(1 * time.Millisecond)
	uptime := server.GetUptime()
	assert.True(t, uptime > 0)

	err = server.Stop(ctx)
	require.NoError(t, err)
}

func TestBaseServer_HTTPMiddlewareConfiguration_ErrorPath(t *testing.T) {
	config := NewServerConfig()
	config.Discovery.Enabled = false
	config.HTTP.Middleware.EnableCORS = true

	mockRegistrar := &mockBusinessServiceRegistrar{}
	mockRegistrar.On("RegisterServices", mock.Anything).Return(nil)

	mockDeps := &mockBusinessDependencyContainer{}

	server, err := NewBusinessServerCore(config, mockRegistrar, mockDeps)
	require.NoError(t, err)

	ctx := context.Background()

	// Should start successfully with default middleware configuration
	err = server.Start(ctx)
	require.NoError(t, err)

	err = server.Stop(ctx)
	require.NoError(t, err)
}

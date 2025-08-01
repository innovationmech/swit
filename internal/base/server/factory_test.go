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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
)

func init() {
	// Initialize logger for tests
	logger.InitLogger()
}

// mockServiceRegistrar implements server.ServiceRegistrar for testing
type mockServiceRegistrar struct {
	registerServicesCalled bool
	registerServicesError  error
}

func (m *mockServiceRegistrar) RegisterServices(registry server.ServiceRegistry) error {
	m.registerServicesCalled = true
	return m.registerServicesError
}

// mockDependencyContainer implements server.DependencyContainer for testing
type mockDependencyContainer struct {
	services map[string]interface{}
	closed   bool
}

func (m *mockDependencyContainer) Close() error {
	m.closed = true
	return nil
}

func (m *mockDependencyContainer) GetService(name string) (interface{}, error) {
	if service, exists := m.services[name]; exists {
		return service, nil
	}
	return nil, nil
}

func newMockDependencyContainer() *mockDependencyContainer {
	return &mockDependencyContainer{
		services: make(map[string]interface{}),
		closed:   false,
	}
}

func TestNewServerFactory(t *testing.T) {
	tests := []struct {
		name        string
		config      *server.ServerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "server config cannot be nil",
		},
		{
			name: "invalid config - empty service name",
			config: &server.ServerConfig{
				ServiceName: "",
			},
			expectError: true,
			errorMsg:    "invalid server configuration",
		},
		{
			name: "valid config",
			config: &server.ServerConfig{
				ServiceName: "test-service",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set defaults if config is not nil and we expect success
			if tt.config != nil && !tt.expectError {
				tt.config.SetDefaults()
			}

			factory, err := NewServerFactory(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, factory)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, factory)
				assert.Equal(t, tt.config, factory.config)
			}
		})
	}
}

func TestServerFactory_CreateServer(t *testing.T) {
	config := server.NewServerConfig()
	config.ServiceName = "test-service"

	factory, err := NewServerFactory(config)
	require.NoError(t, err)

	tests := []struct {
		name        string
		registrar   server.ServiceRegistrar
		deps        server.DependencyContainer
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil registrar",
			registrar:   nil,
			deps:        newMockDependencyContainer(),
			expectError: true,
			errorMsg:    "service registrar cannot be nil",
		},
		{
			name:        "valid parameters",
			registrar:   &mockServiceRegistrar{},
			deps:        newMockDependencyContainer(),
			expectError: false,
		},
		{
			name:        "nil dependencies (should be allowed)",
			registrar:   &mockServiceRegistrar{},
			deps:        nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := factory.CreateServer(tt.registrar, tt.deps)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, server)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, server)
			}
		})
	}
}

func TestServerFactory_CreateHTTPOnlyServer(t *testing.T) {
	config := server.NewServerConfig()
	config.ServiceName = "test-service"
	config.HTTP.Enabled = true
	config.GRPC.Enabled = true // Initially both enabled

	factory, err := NewServerFactory(config)
	require.NoError(t, err)

	registrar := &mockServiceRegistrar{}
	deps := newMockDependencyContainer()

	server, err := factory.CreateHTTPOnlyServer(registrar, deps)
	require.NoError(t, err)
	require.NotNil(t, server)

	// Verify that only HTTP address is available
	assert.NotEmpty(t, server.GetHTTPAddress())
	assert.Empty(t, server.GetGRPCAddress())
}

func TestServerFactory_CreateGRPCOnlyServer(t *testing.T) {
	config := server.NewServerConfig()
	config.ServiceName = "test-service"
	config.HTTP.Enabled = true
	config.GRPC.Enabled = true // Initially both enabled

	factory, err := NewServerFactory(config)
	require.NoError(t, err)

	registrar := &mockServiceRegistrar{}
	deps := newMockDependencyContainer()

	server, err := factory.CreateGRPCOnlyServer(registrar, deps)
	require.NoError(t, err)
	require.NotNil(t, server)

	// Verify that only gRPC address is available
	assert.Empty(t, server.GetHTTPAddress())
	assert.NotEmpty(t, server.GetGRPCAddress())
}

func TestServerFactory_CreateTestServer(t *testing.T) {
	config := server.NewServerConfig()
	config.ServiceName = "test-service"
	config.HTTP.TestPort = "8081"
	config.GRPC.TestPort = "9081"
	config.Discovery.Enabled = true // Initially enabled

	factory, err := NewServerFactory(config)
	require.NoError(t, err)

	registrar := &mockServiceRegistrar{}
	deps := newMockDependencyContainer()

	server, err := factory.CreateTestServer(registrar, deps)
	require.NoError(t, err)
	require.NotNil(t, server)

	// Verify test ports are used
	assert.Equal(t, ":8081", server.GetHTTPAddress())
	assert.Equal(t, ":9081", server.GetGRPCAddress())
}

func TestServerFactory_CreateServerWithCustomConfig(t *testing.T) {
	originalConfig := server.NewServerConfig()
	originalConfig.ServiceName = "original-service"

	factory, err := NewServerFactory(originalConfig)
	require.NoError(t, err)

	customConfig := server.NewServerConfig()
	customConfig.ServiceName = "custom-service"
	customConfig.HTTP.Port = "8082"

	registrar := &mockServiceRegistrar{}
	deps := newMockDependencyContainer()

	tests := []struct {
		name         string
		customConfig *server.ServerConfig
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "nil custom config",
			customConfig: nil,
			expectError:  true,
			errorMsg:     "custom config cannot be nil",
		},
		{
			name: "invalid custom config",
			customConfig: &server.ServerConfig{
				ServiceName: "", // Invalid
			},
			expectError: true,
			errorMsg:    "invalid custom configuration",
		},
		{
			name:         "valid custom config",
			customConfig: customConfig,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := factory.CreateServerWithCustomConfig(tt.customConfig, registrar, deps)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, server)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, server)
			}
		})
	}
}

func TestServerFactory_GetConfig(t *testing.T) {
	originalConfig := server.NewServerConfig()
	originalConfig.ServiceName = "test-service"

	factory, err := NewServerFactory(originalConfig)
	require.NoError(t, err)

	// Get config copy
	configCopy := factory.GetConfig()

	// Verify it's a copy, not the same instance
	assert.Equal(t, originalConfig.ServiceName, configCopy.ServiceName)
	assert.NotSame(t, originalConfig, configCopy)

	// Modify the copy and verify original is unchanged
	configCopy.ServiceName = "modified-service"
	assert.Equal(t, "test-service", factory.config.ServiceName)
	assert.Equal(t, "modified-service", configCopy.ServiceName)
}

func TestServerFactory_ValidateFactoryParameters(t *testing.T) {
	config := server.NewServerConfig()
	config.ServiceName = "test-service"

	factory, err := NewServerFactory(config)
	require.NoError(t, err)

	tests := []struct {
		name        string
		registrar   server.ServiceRegistrar
		deps        server.DependencyContainer
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil registrar",
			registrar:   nil,
			deps:        newMockDependencyContainer(),
			expectError: true,
			errorMsg:    "service registrar cannot be nil",
		},
		{
			name:        "valid registrar with deps",
			registrar:   &mockServiceRegistrar{},
			deps:        newMockDependencyContainer(),
			expectError: false,
		},
		{
			name:        "valid registrar without deps",
			registrar:   &mockServiceRegistrar{},
			deps:        nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := factory.ValidateFactoryParameters(tt.registrar, tt.deps)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServerFactory_ErrorHandling(t *testing.T) {
	config := server.NewServerConfig()
	config.ServiceName = "test-service"

	factory, err := NewServerFactory(config)
	require.NoError(t, err)

	// Test with registrar that returns error during service registration
	errorRegistrar := &mockServiceRegistrar{
		registerServicesError: assert.AnError,
	}
	deps := newMockDependencyContainer()

	server, err := factory.CreateServer(errorRegistrar, deps)
	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), "failed to create base server")
}

func TestServerFactory_Integration(t *testing.T) {
	// Test complete factory workflow
	config := server.NewServerConfig()
	config.ServiceName = "integration-test-service"
	config.HTTP.Port = "8083"
	config.GRPC.Port = "9083"

	factory, err := NewServerFactory(config)
	require.NoError(t, err)

	registrar := &mockServiceRegistrar{}
	deps := newMockDependencyContainer()

	// Create server
	server, err := factory.CreateServer(registrar, deps)
	require.NoError(t, err)
	require.NotNil(t, server)

	// Verify addresses
	assert.Equal(t, ":8083", server.GetHTTPAddress())
	assert.Equal(t, ":9083", server.GetGRPCAddress())

	// Verify transports are available
	transports := server.GetTransports()
	assert.Len(t, transports, 2) // HTTP and gRPC

	// Test shutdown (should not error even if not started)
	err = server.Shutdown()
	assert.NoError(t, err)

	// Verify dependencies were closed
	assert.True(t, deps.closed)
}

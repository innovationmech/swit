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

// mockServiceRegistrar implements server.BusinessServiceRegistrar for testing
type mockServiceRegistrar struct {
	registerServicesCalled bool
	registerServicesError  error
}

func (m *mockServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
	m.registerServicesCalled = true
	return m.registerServicesError
}

// mockDependencyContainer implements server.BusinessDependencyContainer for testing
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
		registrar   server.BusinessServiceRegistrar
		deps        server.BusinessDependencyContainer
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
		registrar   server.BusinessServiceRegistrar
		deps        server.BusinessDependencyContainer
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

func TestServerFactory_DeepCopyIsolation(t *testing.T) {
	// Create a base configuration with nested structures
	baseConfig := server.NewServerConfig()
	baseConfig.ServiceName = "test-service"
	baseConfig.HTTP.Headers = map[string]string{
		"X-Original": "original-value",
	}
	baseConfig.HTTP.Middleware.CustomHeaders = map[string]string{
		"X-Middleware": "middleware-value",
	}
	baseConfig.HTTP.Middleware.CORSConfig.AllowOrigins = []string{
		"http://localhost:3000",
		"https://example.com",
	}
	baseConfig.Discovery.Tags = []string{"api", "v1"}

	// Create factory
	factory, err := NewServerFactory(baseConfig)
	require.NoError(t, err)
	require.NotNil(t, factory)

	// Create mock dependencies
	registrar := &mockServiceRegistrar{}
	deps := newMockDependencyContainer()

	t.Run("HTTP-only server config isolation", func(t *testing.T) {
		// Get original config state
		originalHeaders := make(map[string]string)
		for k, v := range baseConfig.HTTP.Headers {
			originalHeaders[k] = v
		}
		originalCustomHeaders := make(map[string]string)
		for k, v := range baseConfig.HTTP.Middleware.CustomHeaders {
			originalCustomHeaders[k] = v
		}
		originalOrigins := make([]string, len(baseConfig.HTTP.Middleware.CORSConfig.AllowOrigins))
		copy(originalOrigins, baseConfig.HTTP.Middleware.CORSConfig.AllowOrigins)
		originalTags := make([]string, len(baseConfig.Discovery.Tags))
		copy(originalTags, baseConfig.Discovery.Tags)

		// Create HTTP-only server (this should use deep copy)
		_, err := factory.CreateHTTPOnlyServer(registrar, deps)
		require.NoError(t, err)

		// Verify original config is unchanged
		assert.Equal(t, originalHeaders, baseConfig.HTTP.Headers)
		assert.Equal(t, originalCustomHeaders, baseConfig.HTTP.Middleware.CustomHeaders)
		assert.Equal(t, originalOrigins, baseConfig.HTTP.Middleware.CORSConfig.AllowOrigins)
		assert.Equal(t, originalTags, baseConfig.Discovery.Tags)

		// Verify original config transport settings are unchanged
		assert.True(t, baseConfig.HTTP.Enabled, "Original HTTP enabled should be unchanged")
		assert.True(t, baseConfig.GRPC.Enabled, "Original GRPC enabled should be unchanged")
	})

	t.Run("GRPC-only server config isolation", func(t *testing.T) {
		// Get original config state
		originalHTTPEnabled := baseConfig.HTTP.Enabled
		originalGRPCEnabled := baseConfig.GRPC.Enabled

		// Create GRPC-only server (this should use deep copy)
		_, err := factory.CreateGRPCOnlyServer(registrar, deps)
		require.NoError(t, err)

		// Verify original config transport settings are unchanged
		assert.Equal(t, originalHTTPEnabled, baseConfig.HTTP.Enabled, "Original HTTP enabled should be unchanged")
		assert.Equal(t, originalGRPCEnabled, baseConfig.GRPC.Enabled, "Original GRPC enabled should be unchanged")
	})

	t.Run("Test server config isolation", func(t *testing.T) {
		// Get original config state
		originalHTTPTestMode := baseConfig.HTTP.TestMode
		originalGRPCTestMode := baseConfig.GRPC.TestMode

		// Create test server (this should use deep copy)
		_, err := factory.CreateTestServer(registrar, deps)
		require.NoError(t, err)

		// Verify original config test mode settings are unchanged
		assert.Equal(t, originalHTTPTestMode, baseConfig.HTTP.TestMode, "Original HTTP test mode should be unchanged")
		assert.Equal(t, originalGRPCTestMode, baseConfig.GRPC.TestMode, "Original GRPC test mode should be unchanged")
	})

	t.Run("Multiple server creation isolation", func(t *testing.T) {
		// Create multiple servers and verify they don't affect each other
		_, err1 := factory.CreateHTTPOnlyServer(registrar, deps)
		require.NoError(t, err1)

		_, err2 := factory.CreateGRPCOnlyServer(registrar, deps)
		require.NoError(t, err2)

		_, err3 := factory.CreateTestServer(registrar, deps)
		require.NoError(t, err3)

		// Verify original config is still intact
		assert.Equal(t, "test-service", baseConfig.ServiceName)
		assert.Contains(t, baseConfig.HTTP.Headers, "X-Original")
		assert.Contains(t, baseConfig.HTTP.Middleware.CustomHeaders, "X-Middleware")
		assert.Contains(t, baseConfig.HTTP.Middleware.CORSConfig.AllowOrigins, "http://localhost:3000")
		assert.Contains(t, baseConfig.Discovery.Tags, "api")
	})
}

func TestServerFactory_ConfigModificationIsolation(t *testing.T) {
	// Create a configuration with nested structures
	config := server.NewServerConfig()
	config.HTTP.Headers = map[string]string{"X-Test": "test"}
	config.HTTP.Middleware.CORSConfig.AllowOrigins = []string{"http://localhost:3000"}
	config.Discovery.Tags = []string{"test"}

	factory, err := NewServerFactory(config)
	require.NoError(t, err)

	// Create mock dependencies
	registrar := &mockServiceRegistrar{}
	deps := newMockDependencyContainer()

	// Create a server (this should create a deep copy)
	_, err = factory.CreateHTTPOnlyServer(registrar, deps)
	require.NoError(t, err)

	// Modify the original config's nested structures
	config.HTTP.Headers["X-Modified"] = "modified"
	config.HTTP.Middleware.CORSConfig.AllowOrigins = append(config.HTTP.Middleware.CORSConfig.AllowOrigins, "https://modified.com")
	config.Discovery.Tags = append(config.Discovery.Tags, "modified")

	// Create another server and verify it uses the current state of the original config
	_, err = factory.CreateGRPCOnlyServer(registrar, deps)
	require.NoError(t, err)

	// The factory should still reference the original config, so modifications should be visible
	// in subsequent server creations (this tests that the factory itself doesn't create
	// unintended copies of the original config)
	factoryConfig := factory.GetConfig()
	assert.Contains(t, factoryConfig.HTTP.Headers, "X-Modified")
	assert.Contains(t, factoryConfig.HTTP.Middleware.CORSConfig.AllowOrigins, "https://modified.com")
	assert.Contains(t, factoryConfig.Discovery.Tags, "modified")
}

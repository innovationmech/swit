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

	"github.com/innovationmech/swit/pkg/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock factory for testing
type MockServerFactory struct {
	supportedTypes []string
	createFunc     func(*ServerConfig) (BaseServer, error)
}

func (m *MockServerFactory) CreateServer(config *ServerConfig) (BaseServer, error) {
	if m.createFunc != nil {
		return m.createFunc(config)
	}
	return nil, errors.New("mock factory not implemented")
}

func (m *MockServerFactory) GetSupportedTypes() []string {
	return m.supportedTypes
}

func (m *MockServerFactory) SupportsType(serverType string) bool {
	for _, t := range m.supportedTypes {
		if t == serverType {
			return true
		}
	}
	return false
}

// CustomMockFactory for testing with custom SupportsType behavior
type CustomMockFactory struct {
	MockServerFactory
	customSupportsType func(string) bool
}

func (c *CustomMockFactory) SupportsType(serverType string) bool {
	if c.customSupportsType != nil {
		return c.customSupportsType(serverType)
	}
	return c.MockServerFactory.SupportsType(serverType)
}

// Mock server for testing
type MockServer struct {
	config *ServerConfig
}

func (m *MockServer) Start(ctx context.Context) error      { return nil }
func (m *MockServer) Stop(ctx context.Context) error       { return nil }
func (m *MockServer) Shutdown() error                      { return nil }
func (m *MockServer) GetHTTPAddress() string               { return "" }
func (m *MockServer) GetGRPCAddress() string               { return "" }
func (m *MockServer) GetTransports() []transport.Transport { return nil }
func (m *MockServer) GetServiceName() string               { return m.config.ServiceName }

func TestServerFactoryFunc(t *testing.T) {
	// Test successful creation
	factory := ServerFactoryFunc(func(config *ServerConfig) (BaseServer, error) {
		return &MockServer{config: config}, nil
	})

	config := &ServerConfig{ServiceName: "test"}
	server, err := factory.CreateServer(config)
	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, "test", server.GetServiceName())

	// Test supported types
	types := factory.GetSupportedTypes()
	assert.Contains(t, types, "generic")

	// Test supports type
	assert.True(t, factory.SupportsType("any-type"))

	// Test error case
	errorFactory := ServerFactoryFunc(func(config *ServerConfig) (BaseServer, error) {
		return nil, errors.New("creation failed")
	})

	_, err = errorFactory.CreateServer(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creation failed")
}

func TestNewDefaultServerFactory(t *testing.T) {
	factory := NewDefaultServerFactory()

	assert.NotNil(t, factory)
	assert.NotEmpty(t, factory.GetSupportedTypes())
	assert.Contains(t, factory.GetSupportedTypes(), "http")
	assert.Contains(t, factory.GetSupportedTypes(), "grpc")
	assert.Contains(t, factory.GetSupportedTypes(), "mixed")
	assert.Contains(t, factory.GetSupportedTypes(), "generic")
}

func TestDefaultServerFactory_CreateServer(t *testing.T) {
	factory := NewDefaultServerFactory()

	// Test nil config
	_, err := factory.CreateServer(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")

	// Test valid config
	config := NewServerConfig()
	config.SetDefaults()

	server, err := factory.CreateServer(config)
	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, config.ServiceName, server.GetServiceName())

	// Test invalid config
	invalidConfig := &ServerConfig{
		HTTPPort: "invalid-port",
	}

	_, err = factory.CreateServer(invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid server configuration")
}

func TestDefaultServerFactory_SupportsType(t *testing.T) {
	factory := NewDefaultServerFactory()

	assert.True(t, factory.SupportsType("http"))
	assert.True(t, factory.SupportsType("grpc"))
	assert.True(t, factory.SupportsType("mixed"))
	assert.True(t, factory.SupportsType("generic"))
	assert.False(t, factory.SupportsType("unsupported"))
}

func TestNewFactoryRegistry(t *testing.T) {
	registry := NewFactoryRegistry()

	assert.NotNil(t, registry)
	assert.Equal(t, 0, registry.Count())
	assert.Empty(t, registry.ListFactories())
}

func TestFactoryRegistry_Register(t *testing.T) {
	registry := NewFactoryRegistry()
	mockFactory := &MockServerFactory{
		supportedTypes: []string{"test"},
	}

	// Test successful registration
	err := registry.Register("test", mockFactory)
	assert.NoError(t, err)
	assert.Equal(t, 1, registry.Count())
	assert.True(t, registry.HasFactory("test"))

	// Test duplicate registration
	err = registry.Register("test", mockFactory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test empty server type
	err = registry.Register("", mockFactory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server type cannot be empty")

	// Test nil factory
	err = registry.Register("nil-test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "factory cannot be nil")
}

func TestFactoryRegistry_GetFactory(t *testing.T) {
	registry := NewFactoryRegistry()
	mockFactory := &MockServerFactory{
		supportedTypes: []string{"test"},
	}

	// Test getting non-existent factory
	_, err := registry.GetFactory("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no factory registered")

	// Test getting existing factory
	err = registry.Register("test", mockFactory)
	require.NoError(t, err)

	factory, err := registry.GetFactory("test")
	assert.NoError(t, err)
	assert.Equal(t, mockFactory, factory)
}

func TestFactoryRegistry_CreateServer(t *testing.T) {
	registry := NewFactoryRegistry()

	// Test nil config
	_, err := registry.CreateServer(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config cannot be nil")

	// Test with no registered factories
	config := &ServerConfig{ServiceName: "test"}
	_, err = registry.CreateServer(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no factory found")

	// Test with registered factory
	mockServer := &MockServer{config: config}
	mockFactory := &MockServerFactory{
		supportedTypes: []string{"generic"},
		createFunc: func(c *ServerConfig) (BaseServer, error) {
			return mockServer, nil
		},
	}

	err = registry.Register("generic", mockFactory)
	require.NoError(t, err)

	server, err := registry.CreateServer(config)
	assert.NoError(t, err)
	assert.Equal(t, mockServer, server)

	// Test with supporting factory (not exact match)
	registry2 := NewFactoryRegistry()
	supportingFactory := &CustomMockFactory{
		MockServerFactory: MockServerFactory{
			supportedTypes: []string{"other"},
			createFunc: func(c *ServerConfig) (BaseServer, error) {
				return mockServer, nil
			},
		},
		customSupportsType: func(serverType string) bool {
			return serverType == "generic"
		},
	}

	err = registry2.Register("other", supportingFactory)
	require.NoError(t, err)

	server, err = registry2.CreateServer(config)
	assert.NoError(t, err)
	assert.Equal(t, mockServer, server)
}

func TestFactoryRegistry_Unregister(t *testing.T) {
	registry := NewFactoryRegistry()
	mockFactory := &MockServerFactory{}

	err := registry.Register("test", mockFactory)
	require.NoError(t, err)
	assert.Equal(t, 1, registry.Count())

	registry.Unregister("test")
	assert.Equal(t, 0, registry.Count())
	assert.False(t, registry.HasFactory("test"))

	// Unregistering non-existent factory should not error
	registry.Unregister("non-existent")
}

func TestFactoryRegistry_ListFactories(t *testing.T) {
	registry := NewFactoryRegistry()

	// Empty registry
	assert.Empty(t, registry.ListFactories())

	// Add factories
	mockFactory1 := &MockServerFactory{}
	mockFactory2 := &MockServerFactory{}

	err := registry.Register("test1", mockFactory1)
	require.NoError(t, err)
	err = registry.Register("test2", mockFactory2)
	require.NoError(t, err)

	factories := registry.ListFactories()
	assert.Len(t, factories, 2)
	assert.Contains(t, factories, "test1")
	assert.Contains(t, factories, "test2")
}

func TestFactoryRegistry_Clear(t *testing.T) {
	registry := NewFactoryRegistry()
	mockFactory := &MockServerFactory{}

	err := registry.Register("test", mockFactory)
	require.NoError(t, err)
	assert.Equal(t, 1, registry.Count())

	registry.Clear()
	assert.Equal(t, 0, registry.Count())
	assert.Empty(t, registry.ListFactories())
}

func TestGetGlobalFactoryRegistry(t *testing.T) {
	// Test singleton behavior
	registry1 := GetGlobalFactoryRegistry()
	registry2 := GetGlobalFactoryRegistry()

	assert.Equal(t, registry1, registry2)
	assert.NotNil(t, registry1)

	// Should have default factories registered
	assert.True(t, registry1.Count() > 0)
	assert.True(t, registry1.HasFactory("http"))
	assert.True(t, registry1.HasFactory("grpc"))
	assert.True(t, registry1.HasFactory("mixed"))
	assert.True(t, registry1.HasFactory("generic"))
}

func TestRegisterServerFactory(t *testing.T) {
	// Clear global registry for clean test
	GetGlobalFactoryRegistry().Clear()

	mockFactory := &MockServerFactory{
		supportedTypes: []string{"custom"},
	}

	err := RegisterServerFactory("custom", mockFactory)
	assert.NoError(t, err)

	registry := GetGlobalFactoryRegistry()
	assert.True(t, registry.HasFactory("custom"))

	factory, err := registry.GetFactory("custom")
	assert.NoError(t, err)
	assert.Equal(t, mockFactory, factory)
}

func TestCreateServerWithFactory(t *testing.T) {
	// Clear and setup global registry
	GetGlobalFactoryRegistry().Clear()

	mockServer := &MockServer{}
	mockFactory := &MockServerFactory{
		supportedTypes: []string{"generic"},
		createFunc: func(c *ServerConfig) (BaseServer, error) {
			mockServer.config = c
			return mockServer, nil
		},
	}

	err := RegisterServerFactory("generic", mockFactory)
	require.NoError(t, err)

	config := &ServerConfig{ServiceName: "test"}
	server, err := CreateServerWithFactory(config)
	assert.NoError(t, err)
	assert.Equal(t, mockServer, server)
	assert.Equal(t, config, mockServer.config)
}

func TestNewServerBuilder(t *testing.T) {
	builder := NewServerBuilder()

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.config)
}

func TestServerBuilder_WithMethods(t *testing.T) {
	builder := NewServerBuilder()

	// Test method chaining
	result := builder.
		WithName("test-server").
		WithVersion("1.0.0").
		WithHTTPPort("8080").
		WithGRPCPort("9090")

	assert.Equal(t, builder, result) // Should return same instance for chaining
	assert.Equal(t, "test-server", builder.config.ServiceName)
	assert.Equal(t, "1.0.0", builder.config.ServiceVersion)
	assert.Equal(t, "8080", builder.config.HTTPPort)
	assert.Equal(t, "9090", builder.config.GRPCPort)
}

func TestServerBuilder_WithConfig(t *testing.T) {
	builder := NewServerBuilder()
	customConfig := &ServerConfig{
		ServiceName:    "custom",
		ServiceVersion: "2.0.0",
	}

	result := builder.WithConfig(customConfig)
	assert.Equal(t, builder, result)
	assert.Equal(t, customConfig, builder.config)
}

func TestServerBuilder_WithFactory(t *testing.T) {
	builder := NewServerBuilder()
	mockFactory := &MockServerFactory{}

	result := builder.WithFactory(mockFactory)
	assert.Equal(t, builder, result)
	assert.Equal(t, mockFactory, builder.factory)
}

func TestServerBuilder_Build(t *testing.T) {
	// Test with nil config
	builder := &ServerBuilder{}
	_, err := builder.Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")

	// Test with custom factory
	mockServer := &MockServer{}
	mockFactory := &MockServerFactory{
		createFunc: func(c *ServerConfig) (BaseServer, error) {
			mockServer.config = c
			return mockServer, nil
		},
	}

	builder = NewServerBuilder().
		WithName("test").
		WithFactory(mockFactory)

	server, err := builder.Build()
	assert.NoError(t, err)
	assert.Equal(t, mockServer, server)
	assert.Equal(t, "test", mockServer.config.ServiceName)

	// Test with global factory (setup required)
	GetGlobalFactoryRegistry().Clear()
	err = RegisterServerFactory("generic", mockFactory)
	require.NoError(t, err)

	builder2 := NewServerBuilder().WithName("test2")
	server2, err := builder2.Build()
	assert.NoError(t, err)
	assert.Equal(t, mockServer, server2)
}

func TestServerBuilder_Integration(t *testing.T) {
	// Integration test with real default factory
	builder := NewServerBuilder().
		WithName("integration-test").
		WithVersion("1.0.0").
		WithHTTPPort("8080").
		WithGRPCPort("9090")

	server, err := builder.Build()
	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, "integration-test", server.GetServiceName())
}

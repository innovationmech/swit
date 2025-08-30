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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigBuilder(t *testing.T) {
	builder := NewConfigBuilder()

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.config)
	assert.Empty(t, builder.errors)

	// Should have default values
	assert.Equal(t, "swit-service", builder.config.ServiceName)
	assert.True(t, builder.config.HTTP.Enabled)
	assert.True(t, builder.config.GRPC.Enabled)
}

func TestConfigBuilder_WithServiceName(t *testing.T) {
	builder := NewConfigBuilder()

	// Test valid service name
	result := builder.WithServiceName("test-service")
	assert.Same(t, builder, result) // Should return same instance for chaining
	assert.Equal(t, "test-service", builder.config.ServiceName)
	assert.Empty(t, builder.errors)

	// Test empty service name
	builder.WithServiceName("")
	assert.Len(t, builder.errors, 1)
	assert.Contains(t, builder.errors[0].Error(), "service name cannot be empty")
}

func TestConfigBuilder_WithPorts(t *testing.T) {
	builder := NewConfigBuilder()

	// Test HTTP port
	builder.WithHTTPPort("8080")
	assert.Equal(t, "8080", builder.config.HTTP.Port)
	assert.Equal(t, ":8080", builder.config.HTTP.Address)

	// Test gRPC port
	builder.WithGRPCPort("9080")
	assert.Equal(t, "9080", builder.config.GRPC.Port)
	assert.Equal(t, ":9080", builder.config.GRPC.Address)

	// Test empty ports
	builder.WithHTTPPort("").WithGRPCPort("")
	assert.Len(t, builder.errors, 2)
}

func TestConfigBuilder_TransportControl(t *testing.T) {
	builder := NewConfigBuilder()

	// Test enabling/disabling HTTP
	builder.EnableHTTP()
	assert.True(t, builder.config.HTTP.Enabled)

	builder.DisableHTTP()
	assert.False(t, builder.config.HTTP.Enabled)

	// Test enabling/disabling gRPC
	builder.EnableGRPC()
	assert.True(t, builder.config.GRPC.Enabled)

	builder.DisableGRPC()
	assert.False(t, builder.config.GRPC.Enabled)
}

func TestConfigBuilder_WithDiscovery(t *testing.T) {
	builder := NewConfigBuilder().WithServiceName("test-service")

	// Test valid discovery configuration
	builder.WithDiscovery("localhost:8500", "custom-service", []string{"tag1", "tag2"})
	assert.True(t, builder.config.Discovery.Enabled)
	assert.Equal(t, "localhost:8500", builder.config.Discovery.Address)
	assert.Equal(t, "custom-service", builder.config.Discovery.ServiceName)
	assert.Equal(t, []string{"tag1", "tag2"}, builder.config.Discovery.Tags)

	// Test with empty service name (should use default)
	builder2 := NewConfigBuilder().WithServiceName("test-service")
	builder2.WithDiscovery("localhost:8500", "", []string{"tag1"})
	assert.Equal(t, "test-service", builder2.config.Discovery.ServiceName)

	// Test empty address
	builder.WithDiscovery("", "service", []string{})
	assert.Len(t, builder.errors, 1)
	assert.Contains(t, builder.errors[0].Error(), "discovery address cannot be empty")

	// Test disable discovery
	builder.DisableDiscovery()
	assert.False(t, builder.config.Discovery.Enabled)
}

func TestConfigBuilder_WithShutdownTimeout(t *testing.T) {
	builder := NewConfigBuilder()

	// Test valid timeout
	builder.WithShutdownTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, builder.config.ShutdownTimeout)

	// Test invalid timeout
	builder.WithShutdownTimeout(-1 * time.Second)
	assert.Len(t, builder.errors, 1)
	assert.Contains(t, builder.errors[0].Error(), "shutdown timeout must be positive")
}

func TestConfigBuilder_WithMiddleware(t *testing.T) {
	builder := NewConfigBuilder()

	// Test CORS
	builder.WithCORS([]string{"https://example.com"})
	assert.True(t, builder.config.HTTP.Middleware.EnableCORS)
	assert.Equal(t, []string{"https://example.com"}, builder.config.HTTP.Middleware.CORSConfig.AllowOrigins)

	// Test Auth
	builder.WithAuth()
	assert.True(t, builder.config.HTTP.Middleware.EnableAuth)
	assert.True(t, builder.config.GRPC.Interceptors.EnableAuth)

	// Test Rate Limit
	builder.WithRateLimit(100, 200)
	assert.True(t, builder.config.HTTP.Middleware.EnableRateLimit)
	assert.Equal(t, 100, builder.config.HTTP.Middleware.RateLimitConfig.RequestsPerSecond)
	assert.Equal(t, 200, builder.config.HTTP.Middleware.RateLimitConfig.BurstSize)
	assert.True(t, builder.config.GRPC.Interceptors.EnableRateLimit)

	// Test invalid rate limit (only first error is captured due to early return)
	builder.WithRateLimit(-1, 0)
	assert.Len(t, builder.errors, 1) // Only one error for requests per second

	// Test Logging
	builder.WithLogging()
	assert.True(t, builder.config.HTTP.Middleware.EnableLogging)
	assert.True(t, builder.config.GRPC.Interceptors.EnableLogging)

	// Test Timeout
	builder.WithTimeout(30 * time.Second)
	assert.True(t, builder.config.HTTP.Middleware.EnableTimeout)
	assert.Equal(t, 30*time.Second, builder.config.HTTP.Middleware.TimeoutConfig.RequestTimeout)
	assert.Equal(t, 25*time.Second, builder.config.HTTP.Middleware.TimeoutConfig.HandlerTimeout)

	// Test invalid timeout
	builder.WithTimeout(-1 * time.Second)
	assert.Len(t, builder.errors, 2) // Previous 1 + this 1
}

func TestConfigBuilder_WithGRPCOptions(t *testing.T) {
	builder := NewConfigBuilder()

	// Test keepalive
	builder.WithGRPCKeepalive(5*time.Second, 1*time.Second)
	assert.True(t, builder.config.GRPC.EnableKeepalive)
	assert.Equal(t, 5*time.Second, builder.config.GRPC.KeepaliveParams.Time)
	assert.Equal(t, 1*time.Second, builder.config.GRPC.KeepaliveParams.Timeout)

	// Test invalid keepalive
	builder.WithGRPCKeepalive(-1*time.Second, 0)
	assert.Len(t, builder.errors, 1)

	// Test reflection
	builder.WithGRPCReflection()
	assert.True(t, builder.config.GRPC.EnableReflection)

	// Test health service
	builder.WithGRPCHealthService()
	assert.True(t, builder.config.GRPC.EnableHealthService)
}

func TestConfigBuilder_WithTestMode(t *testing.T) {
	builder := NewConfigBuilder()

	builder.WithTestMode("8081", "9081")
	assert.True(t, builder.config.HTTP.TestMode)
	assert.True(t, builder.config.GRPC.TestMode)
	assert.Equal(t, "8081", builder.config.HTTP.TestPort)
	assert.Equal(t, "9081", builder.config.GRPC.TestPort)
	assert.False(t, builder.config.Discovery.Enabled)
}

func TestConfigBuilder_Build(t *testing.T) {
	// Test successful build
	builder := NewConfigBuilder().
		WithServiceName("test-service").
		WithHTTPPort("8080").
		WithGRPCPort("9080")

	config, err := builder.Build()
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "test-service", config.ServiceName)

	// Test build with builder errors
	builder2 := NewConfigBuilder().WithServiceName("")
	config2, err2 := builder2.Build()
	assert.Error(t, err2)
	assert.Nil(t, config2)
	assert.Contains(t, err2.Error(), "configuration builder errors")

	// Test build with validation errors
	builder3 := NewConfigBuilder().
		WithServiceName("test-service").
		DisableHTTP().
		DisableGRPC() // Both transports disabled should fail validation

	config3, err3 := builder3.Build()
	assert.Error(t, err3)
	assert.Nil(t, config3)
	assert.Contains(t, err3.Error(), "configuration validation failed")
}

func TestConfigBuilder_MustBuild(t *testing.T) {
	// Test successful MustBuild
	builder := NewConfigBuilder().WithServiceName("test-service")

	assert.NotPanics(t, func() {
		config := builder.MustBuild()
		assert.NotNil(t, config)
	})

	// Test MustBuild with errors should panic
	builder2 := NewConfigBuilder().WithServiceName("")

	assert.Panics(t, func() {
		builder2.MustBuild()
	})
}

func TestConfigBuilder_Chaining(t *testing.T) {
	// Test method chaining
	config := NewConfigBuilder().
		WithServiceName("chained-service").
		WithHTTPPort("8080").
		WithGRPCPort("9080").
		EnableHTTP().
		EnableGRPC().
		WithCORS([]string{"http://localhost:3000"}).
		WithAuth().
		WithLogging().
		WithTimeout(30*time.Second).
		WithGRPCReflection().
		WithGRPCHealthService().
		WithDiscovery("localhost:8500", "", []string{"test"}).
		MustBuild()

	assert.Equal(t, "chained-service", config.ServiceName)
	assert.Equal(t, "8080", config.HTTP.Port)
	assert.Equal(t, "9080", config.GRPC.Port)
	assert.True(t, config.HTTP.Enabled)
	assert.True(t, config.GRPC.Enabled)
	assert.True(t, config.HTTP.Middleware.EnableCORS)
	assert.True(t, config.HTTP.Middleware.EnableAuth)
	assert.True(t, config.HTTP.Middleware.EnableLogging)
	assert.True(t, config.HTTP.Middleware.EnableTimeout)
	assert.True(t, config.GRPC.EnableReflection)
	assert.True(t, config.GRPC.EnableHealthService)
	assert.True(t, config.Discovery.Enabled)
}

func TestWebAPITemplate(t *testing.T) {
	template := &WebAPITemplate{
		ServiceName: "web-api-service",
		HTTPPort:    "8080",
		GRPCPort:    "9080",
	}

	assert.Equal(t, "web-api", template.GetName())
	assert.NotEmpty(t, template.GetDescription())

	config := template.Build()
	assert.Equal(t, "web-api-service", config.ServiceName)
	assert.Equal(t, "8080", config.HTTP.Port)
	assert.Equal(t, "9080", config.GRPC.Port)
	assert.True(t, config.HTTP.Enabled)
	assert.True(t, config.GRPC.Enabled)
	assert.True(t, config.HTTP.Middleware.EnableCORS)
	assert.True(t, config.HTTP.Middleware.EnableLogging)
	assert.True(t, config.GRPC.EnableReflection)
	assert.True(t, config.GRPC.EnableHealthService)
	assert.True(t, config.Discovery.Enabled)
}

func TestAuthServiceTemplate(t *testing.T) {
	template := &AuthServiceTemplate{
		ServiceName: "auth-service",
		HTTPPort:    "9001",
		GRPCPort:    "50051",
	}

	assert.Equal(t, "auth-service", template.GetName())
	assert.NotEmpty(t, template.GetDescription())

	config := template.Build()
	assert.Equal(t, "auth-service", config.ServiceName)
	assert.Equal(t, "9001", config.HTTP.Port)
	assert.Equal(t, "50051", config.GRPC.Port)
	assert.True(t, config.HTTP.Middleware.EnableAuth)
	assert.True(t, config.HTTP.Middleware.EnableRateLimit)
	assert.True(t, config.GRPC.Interceptors.EnableAuth)
	assert.True(t, config.GRPC.Interceptors.EnableRateLimit)
	assert.Equal(t, 10*time.Second, config.ShutdownTimeout)
}

func TestMicroserviceTemplate(t *testing.T) {
	template := &MicroserviceTemplate{
		ServiceName: "micro-service",
		HTTPPort:    "8080",
		GRPCPort:    "9080",
	}

	assert.Equal(t, "microservice", template.GetName())
	assert.NotEmpty(t, template.GetDescription())

	config := template.Build()
	assert.Equal(t, "micro-service", config.ServiceName)
	assert.True(t, config.HTTP.Enabled)
	assert.True(t, config.GRPC.Enabled)
	assert.True(t, config.GRPC.EnableKeepalive)
}

func TestHTTPOnlyTemplate(t *testing.T) {
	template := &HTTPOnlyTemplate{
		ServiceName: "http-service",
		HTTPPort:    "8080",
	}

	assert.Equal(t, "http-only", template.GetName())
	assert.NotEmpty(t, template.GetDescription())

	config := template.Build()
	assert.Equal(t, "http-service", config.ServiceName)
	assert.True(t, config.HTTP.Enabled)
	assert.False(t, config.GRPC.Enabled)
}

func TestGRPCOnlyTemplate(t *testing.T) {
	template := &GRPCOnlyTemplate{
		ServiceName: "grpc-service",
		GRPCPort:    "9080",
	}

	assert.Equal(t, "grpc-only", template.GetName())
	assert.NotEmpty(t, template.GetDescription())

	config := template.Build()
	assert.Equal(t, "grpc-service", config.ServiceName)
	assert.False(t, config.HTTP.Enabled)
	assert.True(t, config.GRPC.Enabled)
}

func TestTestServiceTemplate(t *testing.T) {
	template := &TestServiceTemplate{
		ServiceName:  "test-service",
		HTTPTestPort: "8081",
		GRPCTestPort: "9081",
	}

	assert.Equal(t, "test-service", template.GetName())
	assert.NotEmpty(t, template.GetDescription())

	config := template.Build()
	assert.Equal(t, "test-service", config.ServiceName)
	assert.True(t, config.HTTP.TestMode)
	assert.True(t, config.GRPC.TestMode)
	assert.Equal(t, "8081", config.HTTP.TestPort)
	assert.Equal(t, "9081", config.GRPC.TestPort)
	assert.False(t, config.Discovery.Enabled)
	assert.Equal(t, 1*time.Second, config.ShutdownTimeout)
}

func TestNewTemplateRegistry(t *testing.T) {
	registry := NewTemplateRegistry()

	assert.NotNil(t, registry)
	assert.NotEmpty(t, registry.templates)

	// Should have default templates
	names := registry.GetTemplateNames()
	expectedTemplates := []string{"web-api", "auth-service", "microservice", "http-only", "grpc-only", "test-service"}

	for _, expected := range expectedTemplates {
		assert.Contains(t, names, expected)
	}
}

// customTestTemplate for testing template registration
type customTestTemplate struct {
	name string
}

func (t *customTestTemplate) GetName() string {
	return t.name
}

func (t *customTestTemplate) GetDescription() string {
	return "Custom test template"
}

func (t *customTestTemplate) Build() *ServerConfig {
	return NewConfigBuilder().
		WithServiceName("custom-service").
		WithHTTPPort("8080").
		MustBuild()
}

func TestTemplateRegistry_RegisterTemplate(t *testing.T) {
	registry := NewTemplateRegistry()

	// Create custom template
	customTemplate := &customTestTemplate{
		name: "custom-template",
	}

	registry.RegisterTemplate(customTemplate)

	template, exists := registry.GetTemplate("custom-template")
	assert.True(t, exists)
	assert.Equal(t, customTemplate, template)
}

func TestTemplateRegistry_GetTemplate(t *testing.T) {
	registry := NewTemplateRegistry()

	// Test existing template
	template, exists := registry.GetTemplate("web-api")
	assert.True(t, exists)
	assert.NotNil(t, template)
	assert.Equal(t, "web-api", template.GetName())

	// Test non-existing template
	template2, exists2 := registry.GetTemplate("non-existent")
	assert.False(t, exists2)
	assert.Nil(t, template2)
}

func TestTemplateRegistry_GetAllTemplates(t *testing.T) {
	registry := NewTemplateRegistry()

	templates := registry.GetAllTemplates()
	assert.NotEmpty(t, templates)

	// Should be a copy, not the original map
	assert.NotSame(t, &registry.templates, &templates)

	// Should contain expected templates
	assert.Contains(t, templates, "web-api")
	assert.Contains(t, templates, "auth-service")
}

func TestTemplateRegistry_CreateConfigFromTemplate(t *testing.T) {
	registry := NewTemplateRegistry()

	// Test with web-api template
	params := map[string]interface{}{
		"service_name": "custom-web-api",
		"http_port":    "8081",
		"grpc_port":    "9081",
	}

	config, err := registry.CreateConfigFromTemplate("web-api", params)
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "custom-web-api", config.ServiceName)
	assert.Equal(t, "8081", config.HTTP.Port)
	assert.Equal(t, "9081", config.GRPC.Port)

	// Test with non-existent template
	config2, err2 := registry.CreateConfigFromTemplate("non-existent", params)
	assert.Error(t, err2)
	assert.Nil(t, config2)
	assert.Contains(t, err2.Error(), "template 'non-existent' not found")

	// Test with HTTP-only template
	httpParams := map[string]interface{}{
		"service_name": "custom-http",
		"http_port":    "8082",
	}

	config3, err3 := registry.CreateConfigFromTemplate("http-only", httpParams)
	assert.NoError(t, err3)
	assert.NotNil(t, config3)
	assert.Equal(t, "custom-http", config3.ServiceName)
	assert.Equal(t, "8082", config3.HTTP.Port)
	assert.True(t, config3.HTTP.Enabled)
	assert.False(t, config3.GRPC.Enabled)
}

func TestNewConfigBuilderFromTemplate(t *testing.T) {
	template := &WebAPITemplate{
		ServiceName: "template-service",
		HTTPPort:    "8080",
		GRPCPort:    "9080",
	}

	builder := NewConfigBuilderFromTemplate(template)
	assert.NotNil(t, builder)
	assert.Equal(t, "template-service", builder.config.ServiceName)
	assert.Equal(t, "8080", builder.config.HTTP.Port)
	assert.Equal(t, "9080", builder.config.GRPC.Port)

	// Should be able to further customize
	config := builder.
		WithServiceName("customized-service").
		WithHTTPPort("8081").
		MustBuild()

	assert.Equal(t, "customized-service", config.ServiceName)
	assert.Equal(t, "8081", config.HTTP.Port)
}

func TestConfigBuilder_ConfigurationImmutability(t *testing.T) {
	builder := NewConfigBuilder().WithServiceName("test-service")

	config1, err1 := builder.Build()
	require.NoError(t, err1)

	config2, err2 := builder.Build()
	require.NoError(t, err2)

	// Should be different instances
	assert.NotSame(t, config1, config2)

	// Modifying one should not affect the other
	config1.ServiceName = "modified"
	assert.Equal(t, "test-service", config2.ServiceName)
}

func TestConfigBuilder_ErrorAccumulation(t *testing.T) {
	builder := NewConfigBuilder().
		WithServiceName("").                      // Error 1
		WithHTTPPort("").                         // Error 2
		WithGRPCPort("").                         // Error 3
		WithDiscovery("", "service", []string{}). // Error 4
		WithShutdownTimeout(-1*time.Second).      // Error 5
		WithRateLimit(-1, 0).                     // Error 6 (only first error due to early return)
		WithTimeout(-1*time.Second).              // Error 7
		WithGRPCKeepalive(-1*time.Second, 0)      // Error 8

	assert.Len(t, builder.errors, 8)

	config, err := builder.Build()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "configuration builder errors")
}

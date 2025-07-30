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

func TestConfigBuilder(t *testing.T) {
	t.Run("basic configuration building", func(t *testing.T) {
		config := NewConfigBuilder().
			WithServiceName("test-service").
			WithServiceVersion("v1.0.0").
			WithHTTPPort("8080").
			WithGRPCPort("9090").
			WithDiscovery(true, "localhost:8500").
			WithTimeouts(30*time.Second, 60*time.Second).
			WithMiddleware(true, true, true).
			Build()

		assert.Equal(t, "test-service", config.ServiceName)
		assert.Equal(t, "v1.0.0", config.ServiceVersion)
		assert.Equal(t, "8080", config.HTTPPort)
		assert.Equal(t, "9090", config.GRPCPort)
		assert.True(t, config.EnableHTTP)
		assert.True(t, config.EnableGRPC)
		assert.True(t, config.Discovery.Enabled)
		assert.Equal(t, "localhost:8500", config.Discovery.Consul.Address)
		assert.Equal(t, 30*time.Second, config.StartupTimeout)
		assert.Equal(t, 60*time.Second, config.ShutdownTimeout)
		assert.True(t, config.Middleware.EnableCORS)
		assert.True(t, config.Middleware.EnableAuth)
		assert.True(t, config.Middleware.EnableRateLimit)
	})

	t.Run("minimal configuration", func(t *testing.T) {
		config := NewConfigBuilder().
			WithServiceName("minimal-service").
			Build()

		assert.Equal(t, "minimal-service", config.ServiceName)
		assert.Equal(t, "v1.0.0", config.ServiceVersion) // default
		assert.True(t, config.EnableHTTP)                // default
		assert.False(t, config.EnableGRPC)               // default when only HTTP is enabled
	})
}

func TestHTTPOnlyTemplate(t *testing.T) {
	template := &HTTPOnlyTemplate{}

	t.Run("template metadata", func(t *testing.T) {
		assert.Equal(t, "http-only", template.GetName())
		assert.Contains(t, template.GetDescription(), "HTTP-only")
	})

	t.Run("default configuration", func(t *testing.T) {
		config := template.CreateConfig()

		assert.True(t, config.EnableHTTP)
		assert.False(t, config.EnableGRPC)
		assert.Equal(t, "8080", config.HTTPPort)
		assert.True(t, config.Middleware.EnableCORS)
		assert.True(t, config.Middleware.EnableRequestLogger)
		assert.True(t, config.Middleware.EnableRateLimit)
		assert.Equal(t, 100, config.Middleware.RateLimitRPS)
	})

	t.Run("with custom options", func(t *testing.T) {
		config := template.CreateConfig(
			WithServiceName("custom-http-service"),
			WithHTTPPort("9080"),
			WithRateLimit(true, 200),
		)

		assert.Equal(t, "custom-http-service", config.ServiceName)
		assert.Equal(t, "9080", config.HTTPPort)
		assert.Equal(t, 200, config.Middleware.RateLimitRPS)
	})
}

func TestGRPCOnlyTemplate(t *testing.T) {
	template := &GRPCOnlyTemplate{}

	t.Run("template metadata", func(t *testing.T) {
		assert.Equal(t, "grpc-only", template.GetName())
		assert.Contains(t, template.GetDescription(), "gRPC-only")
	})

	t.Run("default configuration", func(t *testing.T) {
		config := template.CreateConfig()

		assert.False(t, config.EnableHTTP)
		assert.True(t, config.EnableGRPC)
		assert.Equal(t, "9090", config.GRPCPort)
		assert.True(t, config.GRPCConfig.EnableReflection)
		assert.Equal(t, 4<<20, config.GRPCConfig.MaxRecvMsgSize)
		assert.Equal(t, 4<<20, config.GRPCConfig.MaxSendMsgSize)
	})

	t.Run("with custom options", func(t *testing.T) {
		config := template.CreateConfig(
			WithServiceName("custom-grpc-service"),
			WithGRPCPort("9091"),
			WithGRPCMessageSize(8<<20, 8<<20),
		)

		assert.Equal(t, "custom-grpc-service", config.ServiceName)
		assert.Equal(t, "9091", config.GRPCPort)
		assert.Equal(t, 8<<20, config.GRPCConfig.MaxRecvMsgSize)
		assert.Equal(t, 8<<20, config.GRPCConfig.MaxSendMsgSize)
	})
}

func TestMicroserviceTemplate(t *testing.T) {
	template := &MicroserviceTemplate{}

	t.Run("template metadata", func(t *testing.T) {
		assert.Equal(t, "microservice", template.GetName())
		assert.Contains(t, template.GetDescription(), "microservice")
	})

	t.Run("default configuration", func(t *testing.T) {
		config := template.CreateConfig()

		assert.True(t, config.EnableHTTP)
		assert.True(t, config.EnableGRPC)
		assert.Equal(t, "8080", config.HTTPPort)
		assert.Equal(t, "9090", config.GRPCPort)
		assert.True(t, config.Discovery.Enabled)
		assert.Equal(t, "localhost:8500", config.Discovery.Consul.Address)
		assert.Equal(t, "/health", config.Discovery.HealthCheckPath)
		assert.Equal(t, 10*time.Second, config.Discovery.HealthCheckInterval)
		assert.True(t, config.Middleware.EnableCORS)
		assert.True(t, config.Middleware.EnableRequestLogger)
		assert.True(t, config.Middleware.EnableRateLimit)
		assert.Equal(t, 100, config.Middleware.RateLimitRPS)
		assert.True(t, config.GRPCConfig.EnableReflection)
	})

	t.Run("with custom options", func(t *testing.T) {
		config := template.CreateConfig(
			WithServiceName("custom-microservice"),
			WithServiceDiscovery(true, "consul.example.com:8500"),
			WithRateLimit(true, 500),
		)

		assert.Equal(t, "custom-microservice", config.ServiceName)
		assert.Equal(t, "consul.example.com:8500", config.Discovery.Consul.Address)
		assert.Equal(t, 500, config.Middleware.RateLimitRPS)
	})
}

func TestGatewayTemplate(t *testing.T) {
	template := &GatewayTemplate{}

	t.Run("template metadata", func(t *testing.T) {
		assert.Equal(t, "gateway", template.GetName())
		assert.Contains(t, template.GetDescription(), "Gateway")
	})

	t.Run("default configuration", func(t *testing.T) {
		config := template.CreateConfig()

		assert.True(t, config.EnableHTTP)
		assert.False(t, config.EnableGRPC)
		assert.Equal(t, "8080", config.HTTPPort)
		assert.True(t, config.Middleware.EnableCORS)
		assert.True(t, config.Middleware.EnableAuth)
		assert.True(t, config.Middleware.EnableRequestLogger)
		assert.True(t, config.Middleware.EnableRateLimit)
		assert.True(t, config.Middleware.EnableTimeout)
		assert.Equal(t, 1000, config.Middleware.RateLimitRPS) // Higher for gateway
		assert.Equal(t, 30*time.Second, config.Middleware.TimeoutDuration)
		assert.Equal(t, 30*time.Second, config.HTTPConfig.ReadTimeout)
		assert.Equal(t, 30*time.Second, config.HTTPConfig.WriteTimeout)
		assert.Equal(t, 120*time.Second, config.HTTPConfig.IdleTimeout)
	})

	t.Run("with custom options", func(t *testing.T) {
		config := template.CreateConfig(
			WithServiceName("api-gateway"),
			WithRateLimit(true, 2000),
			WithTimeout(60*time.Second),
		)

		assert.Equal(t, "api-gateway", config.ServiceName)
		assert.Equal(t, 2000, config.Middleware.RateLimitRPS)
		assert.Equal(t, 60*time.Second, config.Middleware.TimeoutDuration)
	})
}

func TestTemplateRegistry(t *testing.T) {
	t.Run("create and register templates", func(t *testing.T) {
		registry := NewTemplateRegistry()

		// Check built-in templates are registered
		templates := registry.List()
		assert.Contains(t, templates, "http-only")
		assert.Contains(t, templates, "grpc-only")
		assert.Contains(t, templates, "microservice")
		assert.Contains(t, templates, "gateway")
		assert.Len(t, templates, 4)
	})

	t.Run("get existing template", func(t *testing.T) {
		registry := NewTemplateRegistry()

		template, err := registry.Get("http-only")
		require.NoError(t, err)
		assert.Equal(t, "http-only", template.GetName())
	})

	t.Run("get non-existing template", func(t *testing.T) {
		registry := NewTemplateRegistry()

		_, err := registry.Get("non-existing")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("create config from template", func(t *testing.T) {
		registry := NewTemplateRegistry()

		config, err := registry.CreateConfig("microservice",
			WithServiceName("test-microservice"),
			WithHTTPPort("8081"),
		)
		require.NoError(t, err)
		assert.Equal(t, "test-microservice", config.ServiceName)
		assert.Equal(t, "8081", config.HTTPPort)
		assert.True(t, config.EnableHTTP)
		assert.True(t, config.EnableGRPC)
		assert.True(t, config.Discovery.Enabled)
	})

	t.Run("create config from non-existing template", func(t *testing.T) {
		registry := NewTemplateRegistry()

		_, err := registry.CreateConfig("non-existing")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestGlobalTemplateRegistry(t *testing.T) {
	t.Run("global registry functions", func(t *testing.T) {
		// Test global registry access
		templates := GetAvailableTemplates()
		assert.Contains(t, templates, "http-only")
		assert.Contains(t, templates, "grpc-only")
		assert.Contains(t, templates, "microservice")
		assert.Contains(t, templates, "gateway")

		// Test creating config from global registry
		config, err := CreateConfigFromTemplate("http-only",
			WithServiceName("global-test-service"),
		)
		require.NoError(t, err)
		assert.Equal(t, "global-test-service", config.ServiceName)
		assert.True(t, config.EnableHTTP)
		assert.False(t, config.EnableGRPC)
	})
}

func TestConfigOptions(t *testing.T) {
	t.Run("service name option", func(t *testing.T) {
		config := NewServerConfig()
		WithServiceName("test-service")(config)
		assert.Equal(t, "test-service", config.ServiceName)
	})

	t.Run("service version option", func(t *testing.T) {
		config := NewServerConfig()
		WithServiceVersion("v2.0.0")(config)
		assert.Equal(t, "v2.0.0", config.ServiceVersion)
	})

	t.Run("HTTP port option", func(t *testing.T) {
		config := NewServerConfig()
		WithHTTPPort("8081")(config)
		assert.Equal(t, "8081", config.HTTPPort)
		assert.True(t, config.EnableHTTP)
	})

	t.Run("gRPC port option", func(t *testing.T) {
		config := NewServerConfig()
		WithGRPCPort("9091")(config)
		assert.Equal(t, "9091", config.GRPCPort)
		assert.True(t, config.EnableGRPC)
	})

	t.Run("service discovery option", func(t *testing.T) {
		config := NewServerConfig()
		WithServiceDiscovery(true, "consul.example.com:8500")(config)
		assert.True(t, config.Discovery.Enabled)
		assert.Equal(t, "consul.example.com:8500", config.Discovery.Consul.Address)
	})

	t.Run("CORS option", func(t *testing.T) {
		config := NewServerConfig()
		WithCORS(true)(config)
		assert.True(t, config.Middleware.EnableCORS)
	})

	t.Run("auth option", func(t *testing.T) {
		config := NewServerConfig()
		WithAuth(true)(config)
		assert.True(t, config.Middleware.EnableAuth)
	})

	t.Run("rate limit option", func(t *testing.T) {
		config := NewServerConfig()
		WithRateLimit(true, 500)(config)
		assert.True(t, config.Middleware.EnableRateLimit)
		assert.Equal(t, 500, config.Middleware.RateLimitRPS)
	})

	t.Run("timeout option", func(t *testing.T) {
		config := NewServerConfig()
		WithTimeout(45 * time.Second)(config)
		assert.True(t, config.Middleware.EnableTimeout)
		assert.Equal(t, 45*time.Second, config.Middleware.TimeoutDuration)
	})

	t.Run("gRPC reflection option", func(t *testing.T) {
		config := NewServerConfig()
		WithGRPCReflection(true)(config)
		assert.True(t, config.GRPCConfig.EnableReflection)
	})

	t.Run("gRPC message size option", func(t *testing.T) {
		config := NewServerConfig()
		WithGRPCMessageSize(8<<20, 8<<20)(config)
		assert.Equal(t, 8<<20, config.GRPCConfig.MaxRecvMsgSize)
		assert.Equal(t, 8<<20, config.GRPCConfig.MaxSendMsgSize)
	})
}

func TestTemplateValidation(t *testing.T) {
	t.Run("HTTP-only template validation", func(t *testing.T) {
		template := &HTTPOnlyTemplate{}
		config := template.CreateConfig(WithServiceName("test-service"))
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("gRPC-only template validation", func(t *testing.T) {
		template := &GRPCOnlyTemplate{}
		config := template.CreateConfig(WithServiceName("test-service"))
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("microservice template validation", func(t *testing.T) {
		template := &MicroserviceTemplate{}
		config := template.CreateConfig(WithServiceName("test-service"))
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("gateway template validation", func(t *testing.T) {
		template := &GatewayTemplate{}
		config := template.CreateConfig(WithServiceName("test-service"))
		err := config.Validate()
		assert.NoError(t, err)
	})
}

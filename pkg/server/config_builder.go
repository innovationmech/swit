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
	"fmt"
	"time"
)

// ConfigBuilder provides a fluent interface for building server configurations
// It supports method chaining and provides validation during the build process
type ConfigBuilder struct {
	config *ServerConfig
	errors []error
}

// NewConfigBuilder creates a new ConfigBuilder with default configuration
func NewConfigBuilder() *ConfigBuilder {
	config := NewServerConfig()
	return &ConfigBuilder{
		config: config,
		errors: make([]error, 0),
	}
}

// NewConfigBuilderFromTemplate creates a new ConfigBuilder from a predefined template
func NewConfigBuilderFromTemplate(template ConfigTemplate) *ConfigBuilder {
	config := template.Build()
	return &ConfigBuilder{
		config: config,
		errors: make([]error, 0),
	}
}

// WithServiceName sets the service name
func (b *ConfigBuilder) WithServiceName(name string) *ConfigBuilder {
	if name == "" {
		b.errors = append(b.errors, fmt.Errorf("service name cannot be empty"))
		return b
	}
	b.config.ServiceName = name
	return b
}

// WithHTTPPort sets the HTTP port
func (b *ConfigBuilder) WithHTTPPort(port string) *ConfigBuilder {
	if port == "" {
		b.errors = append(b.errors, fmt.Errorf("HTTP port cannot be empty"))
		return b
	}
	b.config.HTTP.Port = port
	b.config.HTTP.Address = ":" + port
	return b
}

// WithGRPCPort sets the gRPC port
func (b *ConfigBuilder) WithGRPCPort(port string) *ConfigBuilder {
	if port == "" {
		b.errors = append(b.errors, fmt.Errorf("gRPC port cannot be empty"))
		return b
	}
	b.config.GRPC.Port = port
	b.config.GRPC.Address = ":" + port
	return b
}

// EnableHTTP enables HTTP transport
func (b *ConfigBuilder) EnableHTTP() *ConfigBuilder {
	b.config.HTTP.Enabled = true
	return b
}

// DisableHTTP disables HTTP transport
func (b *ConfigBuilder) DisableHTTP() *ConfigBuilder {
	b.config.HTTP.Enabled = false
	return b
}

// EnableGRPC enables gRPC transport
func (b *ConfigBuilder) EnableGRPC() *ConfigBuilder {
	b.config.GRPC.Enabled = true
	return b
}

// DisableGRPC disables gRPC transport
func (b *ConfigBuilder) DisableGRPC() *ConfigBuilder {
	b.config.GRPC.Enabled = false
	return b
}

// WithDiscovery configures service discovery
func (b *ConfigBuilder) WithDiscovery(address, serviceName string, tags []string) *ConfigBuilder {
	if address == "" {
		b.errors = append(b.errors, fmt.Errorf("discovery address cannot be empty"))
		return b
	}
	if serviceName == "" {
		serviceName = b.config.ServiceName
	}

	b.config.Discovery.Enabled = true
	b.config.Discovery.Address = address
	b.config.Discovery.ServiceName = serviceName
	b.config.Discovery.Tags = tags
	return b
}

// DisableDiscovery disables service discovery
func (b *ConfigBuilder) DisableDiscovery() *ConfigBuilder {
	b.config.Discovery.Enabled = false
	return b
}

// WithShutdownTimeout sets the shutdown timeout
func (b *ConfigBuilder) WithShutdownTimeout(timeout time.Duration) *ConfigBuilder {
	if timeout <= 0 {
		b.errors = append(b.errors, fmt.Errorf("shutdown timeout must be positive"))
		return b
	}
	b.config.ShutdownTimeout = timeout
	return b
}

// WithCORS enables CORS middleware with optional configuration
func (b *ConfigBuilder) WithCORS(allowOrigins []string) *ConfigBuilder {
	b.config.HTTP.Middleware.EnableCORS = true
	if len(allowOrigins) > 0 {
		b.config.HTTP.Middleware.CORSConfig.AllowOrigins = allowOrigins
	}
	return b
}

// WithAuth enables authentication middleware
func (b *ConfigBuilder) WithAuth() *ConfigBuilder {
	b.config.HTTP.Middleware.EnableAuth = true
	b.config.GRPC.Interceptors.EnableAuth = true
	return b
}

// WithRateLimit enables rate limiting with specified requests per second
func (b *ConfigBuilder) WithRateLimit(requestsPerSecond, burstSize int) *ConfigBuilder {
	if requestsPerSecond <= 0 {
		b.errors = append(b.errors, fmt.Errorf("requests per second must be positive"))
		return b
	}
	if burstSize <= 0 {
		b.errors = append(b.errors, fmt.Errorf("burst size must be positive"))
		return b
	}

	b.config.HTTP.Middleware.EnableRateLimit = true
	b.config.HTTP.Middleware.RateLimitConfig.RequestsPerSecond = requestsPerSecond
	b.config.HTTP.Middleware.RateLimitConfig.BurstSize = burstSize

	b.config.GRPC.Interceptors.EnableRateLimit = true
	return b
}

// WithLogging enables request logging
func (b *ConfigBuilder) WithLogging() *ConfigBuilder {
	b.config.HTTP.Middleware.EnableLogging = true
	b.config.GRPC.Interceptors.EnableLogging = true
	return b
}

// WithTimeout sets request timeout
func (b *ConfigBuilder) WithTimeout(timeout time.Duration) *ConfigBuilder {
	if timeout <= 0 {
		b.errors = append(b.errors, fmt.Errorf("timeout must be positive"))
		return b
	}

	b.config.HTTP.Middleware.EnableTimeout = true
	b.config.HTTP.Middleware.TimeoutConfig.RequestTimeout = timeout
	b.config.HTTP.Middleware.TimeoutConfig.HandlerTimeout = timeout - (5 * time.Second)
	return b
}

// WithGRPCKeepalive enables gRPC keepalive with custom parameters
func (b *ConfigBuilder) WithGRPCKeepalive(time, timeout time.Duration) *ConfigBuilder {
	if time <= 0 || timeout <= 0 {
		b.errors = append(b.errors, fmt.Errorf("keepalive time and timeout must be positive"))
		return b
	}

	b.config.GRPC.EnableKeepalive = true
	b.config.GRPC.KeepaliveParams.Time = time
	b.config.GRPC.KeepaliveParams.Timeout = timeout
	return b
}

// WithGRPCReflection enables gRPC reflection
func (b *ConfigBuilder) WithGRPCReflection() *ConfigBuilder {
	b.config.GRPC.EnableReflection = true
	return b
}

// WithGRPCHealthService enables gRPC health service
func (b *ConfigBuilder) WithGRPCHealthService() *ConfigBuilder {
	b.config.GRPC.EnableHealthService = true
	return b
}

// WithTestMode enables test mode with optional test ports
func (b *ConfigBuilder) WithTestMode(httpTestPort, grpcTestPort string) *ConfigBuilder {
	b.config.HTTP.TestMode = true
	b.config.GRPC.TestMode = true

	if httpTestPort != "" {
		b.config.HTTP.TestPort = httpTestPort
	}
	if grpcTestPort != "" {
		b.config.GRPC.TestPort = grpcTestPort
	}

	// Disable discovery in test mode by default
	b.config.Discovery.Enabled = false
	return b
}

// Build builds and validates the configuration
func (b *ConfigBuilder) Build() (*ServerConfig, error) {
	// Check for builder errors first
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("configuration builder errors: %v", b.errors)
	}

	// Validate the final configuration
	if err := b.config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Return a copy to prevent external modification
	configCopy := *b.config
	return &configCopy, nil
}

// MustBuild builds the configuration and panics on error
// This is useful for static configuration where errors should not occur
func (b *ConfigBuilder) MustBuild() *ServerConfig {
	config, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build configuration: %v", err))
	}
	return config
}

// ConfigTemplate defines an interface for configuration templates
type ConfigTemplate interface {
	// Build creates a server configuration from the template
	Build() *ServerConfig
	// GetName returns the template name
	GetName() string
	// GetDescription returns the template description
	GetDescription() string
}

// WebAPITemplate provides a template for web API services
type WebAPITemplate struct {
	ServiceName string
	HTTPPort    string
	GRPCPort    string
}

func (t *WebAPITemplate) GetName() string {
	return "web-api"
}

func (t *WebAPITemplate) GetDescription() string {
	return "Template for web API services with HTTP and gRPC support, CORS, logging, and service discovery"
}

func (t *WebAPITemplate) Build() *ServerConfig {
	return NewConfigBuilder().
		WithServiceName(t.ServiceName).
		WithHTTPPort(t.HTTPPort).
		WithGRPCPort(t.GRPCPort).
		EnableHTTP().
		EnableGRPC().
		WithCORS([]string{"http://localhost:3000", "http://localhost:8080"}).
		WithLogging().
		WithTimeout(30*time.Second).
		WithGRPCReflection().
		WithGRPCHealthService().
		WithDiscovery("127.0.0.1:8500", "", []string{"api", "v1"}).
		MustBuild()
}

// AuthServiceTemplate provides a template for authentication services
type AuthServiceTemplate struct {
	ServiceName string
	HTTPPort    string
	GRPCPort    string
}

func (t *AuthServiceTemplate) GetName() string {
	return "auth-service"
}

func (t *AuthServiceTemplate) GetDescription() string {
	return "Template for authentication services with enhanced security, rate limiting, and monitoring"
}

func (t *AuthServiceTemplate) Build() *ServerConfig {
	return NewConfigBuilder().
		WithServiceName(t.ServiceName).
		WithHTTPPort(t.HTTPPort).
		WithGRPCPort(t.GRPCPort).
		EnableHTTP().
		EnableGRPC().
		WithCORS([]string{"https://localhost:3000", "https://app.example.com"}).
		WithAuth().
		WithRateLimit(100, 200).
		WithLogging().
		WithTimeout(15*time.Second).
		WithGRPCKeepalive(5*time.Second, 1*time.Second).
		WithGRPCReflection().
		WithGRPCHealthService().
		WithDiscovery("127.0.0.1:8500", "", []string{"auth", "security", "v1"}).
		WithShutdownTimeout(10 * time.Second).
		MustBuild()
}

// MicroserviceTemplate provides a template for general microservices
type MicroserviceTemplate struct {
	ServiceName string
	HTTPPort    string
	GRPCPort    string
}

func (t *MicroserviceTemplate) GetName() string {
	return "microservice"
}

func (t *MicroserviceTemplate) GetDescription() string {
	return "Template for general microservices with balanced configuration"
}

func (t *MicroserviceTemplate) Build() *ServerConfig {
	return NewConfigBuilder().
		WithServiceName(t.ServiceName).
		WithHTTPPort(t.HTTPPort).
		WithGRPCPort(t.GRPCPort).
		EnableHTTP().
		EnableGRPC().
		WithCORS([]string{"http://localhost:3000", "http://localhost:8080"}).
		WithLogging().
		WithTimeout(30*time.Second).
		WithGRPCKeepalive(15*time.Second, 5*time.Second).
		WithGRPCReflection().
		WithGRPCHealthService().
		WithDiscovery("127.0.0.1:8500", "", []string{"microservice", "v1"}).
		MustBuild()
}

// HTTPOnlyTemplate provides a template for HTTP-only services
type HTTPOnlyTemplate struct {
	ServiceName string
	HTTPPort    string
}

func (t *HTTPOnlyTemplate) GetName() string {
	return "http-only"
}

func (t *HTTPOnlyTemplate) GetDescription() string {
	return "Template for HTTP-only services like web servers or REST APIs"
}

func (t *HTTPOnlyTemplate) Build() *ServerConfig {
	return NewConfigBuilder().
		WithServiceName(t.ServiceName).
		WithHTTPPort(t.HTTPPort).
		EnableHTTP().
		DisableGRPC().
		WithCORS([]string{"http://localhost:3000", "http://localhost:8080"}).
		WithLogging().
		WithTimeout(30*time.Second).
		WithDiscovery("127.0.0.1:8500", "", []string{"http", "web", "v1"}).
		MustBuild()
}

// GRPCOnlyTemplate provides a template for gRPC-only services
type GRPCOnlyTemplate struct {
	ServiceName string
	GRPCPort    string
}

func (t *GRPCOnlyTemplate) GetName() string {
	return "grpc-only"
}

func (t *GRPCOnlyTemplate) GetDescription() string {
	return "Template for gRPC-only services like internal APIs or service-to-service communication"
}

func (t *GRPCOnlyTemplate) Build() *ServerConfig {
	return NewConfigBuilder().
		WithServiceName(t.ServiceName).
		WithGRPCPort(t.GRPCPort).
		DisableHTTP().
		EnableGRPC().
		WithLogging().
		WithGRPCKeepalive(10*time.Second, 2*time.Second).
		WithGRPCReflection().
		WithGRPCHealthService().
		WithDiscovery("127.0.0.1:8500", "", []string{"grpc", "internal", "v1"}).
		MustBuild()
}

// TestServiceTemplate provides a template for test services
type TestServiceTemplate struct {
	ServiceName  string
	HTTPTestPort string
	GRPCTestPort string
}

func (t *TestServiceTemplate) GetName() string {
	return "test-service"
}

func (t *TestServiceTemplate) GetDescription() string {
	return "Template for test services with discovery disabled and test ports"
}

func (t *TestServiceTemplate) Build() *ServerConfig {
	return NewConfigBuilder().
		WithServiceName(t.ServiceName).
		WithHTTPPort("8080").
		WithGRPCPort("9080").
		EnableHTTP().
		EnableGRPC().
		WithTestMode(t.HTTPTestPort, t.GRPCTestPort).
		WithLogging().
		WithGRPCReflection().
		WithGRPCHealthService().
		WithShutdownTimeout(1 * time.Second).
		MustBuild()
}

// TemplateRegistry manages available configuration templates
type TemplateRegistry struct {
	templates map[string]ConfigTemplate
}

// NewTemplateRegistry creates a new template registry with default templates
func NewTemplateRegistry() *TemplateRegistry {
	registry := &TemplateRegistry{
		templates: make(map[string]ConfigTemplate),
	}

	// Register default templates
	registry.RegisterTemplate(&WebAPITemplate{})
	registry.RegisterTemplate(&AuthServiceTemplate{})
	registry.RegisterTemplate(&MicroserviceTemplate{})
	registry.RegisterTemplate(&HTTPOnlyTemplate{})
	registry.RegisterTemplate(&GRPCOnlyTemplate{})
	registry.RegisterTemplate(&TestServiceTemplate{})

	return registry
}

// RegisterTemplate registers a new template
func (r *TemplateRegistry) RegisterTemplate(template ConfigTemplate) {
	r.templates[template.GetName()] = template
}

// GetTemplate retrieves a template by name
func (r *TemplateRegistry) GetTemplate(name string) (ConfigTemplate, bool) {
	template, exists := r.templates[name]
	return template, exists
}

// GetAllTemplates returns all registered templates
func (r *TemplateRegistry) GetAllTemplates() map[string]ConfigTemplate {
	// Return a copy to prevent external modification
	templates := make(map[string]ConfigTemplate)
	for name, template := range r.templates {
		templates[name] = template
	}
	return templates
}

// GetTemplateNames returns the names of all registered templates
func (r *TemplateRegistry) GetTemplateNames() []string {
	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}

// CreateConfigFromTemplate creates a configuration from a template with custom parameters
func (r *TemplateRegistry) CreateConfigFromTemplate(templateName string, params map[string]interface{}) (*ServerConfig, error) {
	template, exists := r.GetTemplate(templateName)
	if !exists {
		return nil, fmt.Errorf("template '%s' not found", templateName)
	}

	// Apply parameters to template based on template type
	switch t := template.(type) {
	case *WebAPITemplate:
		if serviceName, ok := params["service_name"].(string); ok {
			t.ServiceName = serviceName
		}
		if httpPort, ok := params["http_port"].(string); ok {
			t.HTTPPort = httpPort
		}
		if grpcPort, ok := params["grpc_port"].(string); ok {
			t.GRPCPort = grpcPort
		}
	case *AuthServiceTemplate:
		if serviceName, ok := params["service_name"].(string); ok {
			t.ServiceName = serviceName
		}
		if httpPort, ok := params["http_port"].(string); ok {
			t.HTTPPort = httpPort
		}
		if grpcPort, ok := params["grpc_port"].(string); ok {
			t.GRPCPort = grpcPort
		}
	case *MicroserviceTemplate:
		if serviceName, ok := params["service_name"].(string); ok {
			t.ServiceName = serviceName
		}
		if httpPort, ok := params["http_port"].(string); ok {
			t.HTTPPort = httpPort
		}
		if grpcPort, ok := params["grpc_port"].(string); ok {
			t.GRPCPort = grpcPort
		}
	case *HTTPOnlyTemplate:
		if serviceName, ok := params["service_name"].(string); ok {
			t.ServiceName = serviceName
		}
		if httpPort, ok := params["http_port"].(string); ok {
			t.HTTPPort = httpPort
		}
	case *GRPCOnlyTemplate:
		if serviceName, ok := params["service_name"].(string); ok {
			t.ServiceName = serviceName
		}
		if grpcPort, ok := params["grpc_port"].(string); ok {
			t.GRPCPort = grpcPort
		}
	case *TestServiceTemplate:
		if serviceName, ok := params["service_name"].(string); ok {
			t.ServiceName = serviceName
		}
		if httpTestPort, ok := params["http_test_port"].(string); ok {
			t.HTTPTestPort = httpTestPort
		}
		if grpcTestPort, ok := params["grpc_test_port"].(string); ok {
			t.GRPCTestPort = grpcTestPort
		}
	}

	return template.Build(), nil
}

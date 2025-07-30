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

// ConfigTemplate defines a template for server configuration
type ConfigTemplate interface {
	// GetName returns the template name
	GetName() string
	// GetDescription returns the template description
	GetDescription() string
	// CreateConfig creates a new configuration based on this template
	CreateConfig(options ...ConfigOption) *ServerConfig
	// GetDefaultOptions returns the default options for this template
	GetDefaultOptions() []ConfigOption
}

// ConfigOption defines a configuration option function
type ConfigOption func(*ServerConfig)

// ConfigBuilder provides a fluent interface for building server configurations
type ConfigBuilder struct {
	config *ServerConfig
}

// NewConfigBuilder creates a new configuration builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: NewServerConfig(),
	}
}

// WithServiceName sets the service name
func (b *ConfigBuilder) WithServiceName(name string) *ConfigBuilder {
	b.config.ServiceName = name
	return b
}

// WithServiceVersion sets the service version
func (b *ConfigBuilder) WithServiceVersion(version string) *ConfigBuilder {
	b.config.ServiceVersion = version
	return b
}

// WithHTTPPort sets the HTTP port
func (b *ConfigBuilder) WithHTTPPort(port string) *ConfigBuilder {
	b.config.HTTPPort = port
	b.config.EnableHTTP = true
	return b
}

// WithGRPCPort sets the gRPC port
func (b *ConfigBuilder) WithGRPCPort(port string) *ConfigBuilder {
	b.config.GRPCPort = port
	b.config.EnableGRPC = true
	return b
}

// WithDiscovery enables service discovery with the given configuration
func (b *ConfigBuilder) WithDiscovery(enabled bool, consulAddress string) *ConfigBuilder {
	b.config.Discovery.Enabled = enabled
	if consulAddress != "" {
		b.config.Discovery.Consul.Address = consulAddress
	}
	return b
}

// WithTimeouts sets the server timeouts
func (b *ConfigBuilder) WithTimeouts(startup, shutdown time.Duration) *ConfigBuilder {
	b.config.StartupTimeout = startup
	b.config.ShutdownTimeout = shutdown
	return b
}

// WithMiddleware configures middleware settings
func (b *ConfigBuilder) WithMiddleware(enableCORS, enableAuth, enableRateLimit bool) *ConfigBuilder {
	b.config.Middleware.EnableCORS = enableCORS
	b.config.Middleware.EnableAuth = enableAuth
	b.config.Middleware.EnableRateLimit = enableRateLimit
	return b
}

// Build creates the final configuration
func (b *ConfigBuilder) Build() *ServerConfig {
	b.config.SetDefaults()
	return b.config
}

// HTTPOnlyTemplate provides a template for HTTP-only services
type HTTPOnlyTemplate struct{}

func (t *HTTPOnlyTemplate) GetName() string {
	return "http-only"
}

func (t *HTTPOnlyTemplate) GetDescription() string {
	return "HTTP-only service template for REST APIs"
}

func (t *HTTPOnlyTemplate) CreateConfig(options ...ConfigOption) *ServerConfig {
	config := NewServerConfig()
	config.EnableHTTP = true
	config.EnableGRPC = false
	config.HTTPPort = "8080"
	config.Middleware.EnableCORS = true
	config.Middleware.EnableRequestLogger = true
	config.Middleware.EnableRateLimit = true
	config.Middleware.RateLimitRPS = 100

	// Apply custom options
	for _, option := range options {
		option(config)
	}

	config.SetDefaults()
	return config
}

func (t *HTTPOnlyTemplate) GetDefaultOptions() []ConfigOption {
	return []ConfigOption{
		WithServiceName("http-service"),
		WithHTTPPort("8080"),
		WithCORS(true),
		WithRateLimit(true, 100),
	}
}

// GRPCOnlyTemplate provides a template for gRPC-only services
type GRPCOnlyTemplate struct{}

func (t *GRPCOnlyTemplate) GetName() string {
	return "grpc-only"
}

func (t *GRPCOnlyTemplate) GetDescription() string {
	return "gRPC-only service template for high-performance APIs"
}

func (t *GRPCOnlyTemplate) CreateConfig(options ...ConfigOption) *ServerConfig {
	config := NewServerConfig()
	config.EnableHTTP = false
	config.EnableGRPC = true
	config.GRPCPort = "9090"
	config.GRPCConfig.EnableReflection = true
	config.GRPCConfig.MaxRecvMsgSize = 4 << 20 // 4MB
	config.GRPCConfig.MaxSendMsgSize = 4 << 20 // 4MB

	// Apply custom options
	for _, option := range options {
		option(config)
	}

	config.SetDefaults()
	return config
}

func (t *GRPCOnlyTemplate) GetDefaultOptions() []ConfigOption {
	return []ConfigOption{
		WithServiceName("grpc-service"),
		WithGRPCPort("9090"),
		WithGRPCReflection(true),
	}
}

// MicroserviceTemplate provides a template for microservices with both HTTP and gRPC
type MicroserviceTemplate struct{}

func (t *MicroserviceTemplate) GetName() string {
	return "microservice"
}

func (t *MicroserviceTemplate) GetDescription() string {
	return "Full microservice template with HTTP, gRPC, and service discovery"
}

func (t *MicroserviceTemplate) CreateConfig(options ...ConfigOption) *ServerConfig {
	config := NewServerConfig()
	config.EnableHTTP = true
	config.EnableGRPC = true
	config.HTTPPort = "8080"
	config.GRPCPort = "9090"

	// Enable service discovery
	config.Discovery.Enabled = true
	config.Discovery.Consul.Address = "localhost:8500"
	config.Discovery.HealthCheckPath = "/health"
	config.Discovery.HealthCheckInterval = 10 * time.Second

	// Enable middleware
	config.Middleware.EnableCORS = true
	config.Middleware.EnableRequestLogger = true
	config.Middleware.EnableRateLimit = true
	config.Middleware.RateLimitRPS = 100

	// Configure gRPC
	config.GRPCConfig.EnableReflection = true

	// Apply custom options
	for _, option := range options {
		option(config)
	}

	config.SetDefaults()
	return config
}

func (t *MicroserviceTemplate) GetDefaultOptions() []ConfigOption {
	return []ConfigOption{
		WithServiceName("microservice"),
		WithHTTPPort("8080"),
		WithGRPCPort("9090"),
		WithServiceDiscovery(true, "localhost:8500"),
		WithCORS(true),
		WithRateLimit(true, 100),
	}
}

// GatewayTemplate provides a template for API gateways
type GatewayTemplate struct{}

func (t *GatewayTemplate) GetName() string {
	return "gateway"
}

func (t *GatewayTemplate) GetDescription() string {
	return "API Gateway template with enhanced middleware and routing"
}

func (t *GatewayTemplate) CreateConfig(options ...ConfigOption) *ServerConfig {
	config := NewServerConfig()
	config.EnableHTTP = true
	config.EnableGRPC = false
	config.HTTPPort = "8080"

	// Enhanced middleware for gateway
	config.Middleware.EnableCORS = true
	config.Middleware.EnableAuth = true
	config.Middleware.EnableRequestLogger = true
	config.Middleware.EnableRateLimit = true
	config.Middleware.EnableTimeout = true
	config.Middleware.RateLimitRPS = 1000 // Higher rate limit for gateway
	config.Middleware.TimeoutDuration = 30 * time.Second

	// Enhanced HTTP configuration
	config.HTTPConfig.ReadTimeout = 30 * time.Second
	config.HTTPConfig.WriteTimeout = 30 * time.Second
	config.HTTPConfig.IdleTimeout = 120 * time.Second

	// Apply custom options
	for _, option := range options {
		option(config)
	}

	config.SetDefaults()
	return config
}

func (t *GatewayTemplate) GetDefaultOptions() []ConfigOption {
	return []ConfigOption{
		WithServiceName("api-gateway"),
		WithHTTPPort("8080"),
		WithCORS(true),
		WithAuth(true),
		WithRateLimit(true, 1000),
		WithTimeout(30 * time.Second),
	}
}

// Configuration option functions

// WithServiceName sets the service name
func WithServiceName(name string) ConfigOption {
	return func(c *ServerConfig) {
		c.ServiceName = name
	}
}

// WithServiceVersion sets the service version
func WithServiceVersion(version string) ConfigOption {
	return func(c *ServerConfig) {
		c.ServiceVersion = version
	}
}

// WithHTTPPort sets the HTTP port
func WithHTTPPort(port string) ConfigOption {
	return func(c *ServerConfig) {
		c.HTTPPort = port
		c.EnableHTTP = true
	}
}

// WithGRPCPort sets the gRPC port
func WithGRPCPort(port string) ConfigOption {
	return func(c *ServerConfig) {
		c.GRPCPort = port
		c.EnableGRPC = true
	}
}

// WithServiceDiscovery enables service discovery
func WithServiceDiscovery(enabled bool, consulAddress string) ConfigOption {
	return func(c *ServerConfig) {
		c.Discovery.Enabled = enabled
		if consulAddress != "" {
			c.Discovery.Consul.Address = consulAddress
		}
	}
}

// WithCORS enables or disables CORS
func WithCORS(enabled bool) ConfigOption {
	return func(c *ServerConfig) {
		c.Middleware.EnableCORS = enabled
	}
}

// WithAuth enables or disables authentication
func WithAuth(enabled bool) ConfigOption {
	return func(c *ServerConfig) {
		c.Middleware.EnableAuth = enabled
	}
}

// WithRateLimit configures rate limiting
func WithRateLimit(enabled bool, rps int) ConfigOption {
	return func(c *ServerConfig) {
		c.Middleware.EnableRateLimit = enabled
		c.Middleware.RateLimitRPS = rps
	}
}

// WithTimeout configures request timeout
func WithTimeout(duration time.Duration) ConfigOption {
	return func(c *ServerConfig) {
		c.Middleware.EnableTimeout = true
		c.Middleware.TimeoutDuration = duration
	}
}

// WithGRPCReflection enables or disables gRPC reflection
func WithGRPCReflection(enabled bool) ConfigOption {
	return func(c *ServerConfig) {
		c.GRPCConfig.EnableReflection = enabled
	}
}

// WithGRPCMessageSize sets gRPC message size limits
func WithGRPCMessageSize(maxRecv, maxSend int) ConfigOption {
	return func(c *ServerConfig) {
		c.GRPCConfig.MaxRecvMsgSize = maxRecv
		c.GRPCConfig.MaxSendMsgSize = maxSend
	}
}

// TemplateRegistry manages configuration templates
type TemplateRegistry struct {
	templates map[string]ConfigTemplate
}

// NewTemplateRegistry creates a new template registry
func NewTemplateRegistry() *TemplateRegistry {
	registry := &TemplateRegistry{
		templates: make(map[string]ConfigTemplate),
	}

	// Register built-in templates
	registry.Register(&HTTPOnlyTemplate{})
	registry.Register(&GRPCOnlyTemplate{})
	registry.Register(&MicroserviceTemplate{})
	registry.Register(&GatewayTemplate{})

	return registry
}

// Register registers a new template
func (r *TemplateRegistry) Register(template ConfigTemplate) {
	r.templates[template.GetName()] = template
}

// Get retrieves a template by name
func (r *TemplateRegistry) Get(name string) (ConfigTemplate, error) {
	template, exists := r.templates[name]
	if !exists {
		return nil, fmt.Errorf("template '%s' not found", name)
	}
	return template, nil
}

// List returns all available template names
func (r *TemplateRegistry) List() []string {
	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}

// CreateConfig creates a configuration using the specified template
func (r *TemplateRegistry) CreateConfig(templateName string, options ...ConfigOption) (*ServerConfig, error) {
	template, err := r.Get(templateName)
	if err != nil {
		return nil, err
	}
	return template.CreateConfig(options...), nil
}

// Global template registry
var globalTemplateRegistry *TemplateRegistry

// GetGlobalTemplateRegistry returns the global template registry
func GetGlobalTemplateRegistry() *TemplateRegistry {
	if globalTemplateRegistry == nil {
		globalTemplateRegistry = NewTemplateRegistry()
	}
	return globalTemplateRegistry
}

// CreateConfigFromTemplate creates a configuration using a global template
func CreateConfigFromTemplate(templateName string, options ...ConfigOption) (*ServerConfig, error) {
	return GetGlobalTemplateRegistry().CreateConfig(templateName, options...)
}

// GetAvailableTemplates returns all available template names
func GetAvailableTemplates() []string {
	return GetGlobalTemplateRegistry().List()
}

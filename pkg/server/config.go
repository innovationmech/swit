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
	"net"
	"strconv"
	"strings"
	"time"
)

// ServerConfig holds the configuration for the base server
type ServerConfig struct {
	// Service identification
	ServiceName    string `yaml:"service_name" json:"service_name"`
	ServiceVersion string `yaml:"service_version" json:"service_version"`

	// Transport configuration
	HTTPPort    string `yaml:"http_port" json:"http_port"`
	GRPCPort    string `yaml:"grpc_port" json:"grpc_port"`
	EnableHTTP  bool   `yaml:"enable_http" json:"enable_http"`
	EnableGRPC  bool   `yaml:"enable_grpc" json:"enable_grpc"`
	EnableReady bool   `yaml:"enable_ready" json:"enable_ready"`

	// Lifecycle configuration
	ShutdownTimeout  time.Duration `yaml:"shutdown_timeout" json:"shutdown_timeout"`
	StartupTimeout   time.Duration `yaml:"startup_timeout" json:"startup_timeout"`
	GracefulShutdown bool          `yaml:"graceful_shutdown" json:"graceful_shutdown"`

	// Service discovery configuration
	Discovery DiscoveryConfig `yaml:"discovery" json:"discovery"`

	// Middleware configuration
	Middleware MiddlewareConfig `yaml:"middleware" json:"middleware"`

	// Transport specific configuration
	HTTPConfig HTTPTransportConfig `yaml:"http_config" json:"http_config"`
	GRPCConfig GRPCTransportConfig `yaml:"grpc_config" json:"grpc_config"`
}

// DiscoveryConfig holds service discovery configuration
type DiscoveryConfig struct {
	Enabled             bool          `yaml:"enabled" json:"enabled"`
	ServiceName         string        `yaml:"service_name" json:"service_name"`
	Tags                []string      `yaml:"tags" json:"tags"`
	HealthCheckPath     string        `yaml:"health_check_path" json:"health_check_path"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
	RetryAttempts       int           `yaml:"retry_attempts" json:"retry_attempts"`
	RetryInterval       time.Duration `yaml:"retry_interval" json:"retry_interval"`
	Consul              ConsulConfig  `yaml:"consul" json:"consul"`
}

// ConsulConfig holds Consul specific configuration
type ConsulConfig struct {
	Address    string `yaml:"address" json:"address"`
	Scheme     string `yaml:"scheme" json:"scheme"`
	Datacenter string `yaml:"datacenter" json:"datacenter"`
	Token      string `yaml:"token" json:"token"`
}

// MiddlewareConfig holds middleware configuration
type MiddlewareConfig struct {
	EnableCORS          bool          `yaml:"enable_cors" json:"enable_cors"`
	EnableAuth          bool          `yaml:"enable_auth" json:"enable_auth"`
	EnableRateLimit     bool          `yaml:"enable_rate_limit" json:"enable_rate_limit"`
	EnableRequestLogger bool          `yaml:"enable_request_logger" json:"enable_request_logger"`
	EnableTimeout       bool          `yaml:"enable_timeout" json:"enable_timeout"`
	TimeoutDuration     time.Duration `yaml:"timeout_duration" json:"timeout_duration"`
	RateLimitRPS        int           `yaml:"rate_limit_rps" json:"rate_limit_rps"`
}

// HTTPTransportConfig holds HTTP transport specific configuration
type HTTPTransportConfig struct {
	ReadTimeout    time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout    time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes" json:"max_header_bytes"`
	TrustedProxies []string      `yaml:"trusted_proxies" json:"trusted_proxies"`
}

// GRPCTransportConfig holds gRPC transport specific configuration
type GRPCTransportConfig struct {
	MaxRecvMsgSize    int           `yaml:"max_recv_msg_size" json:"max_recv_msg_size"`
	MaxSendMsgSize    int           `yaml:"max_send_msg_size" json:"max_send_msg_size"`
	KeepaliveTime     time.Duration `yaml:"keepalive_time" json:"keepalive_time"`
	KeepaliveTimeout  time.Duration `yaml:"keepalive_timeout" json:"keepalive_timeout"`
	MaxConnectionIdle time.Duration `yaml:"max_connection_idle" json:"max_connection_idle"`
	EnableReflection  bool          `yaml:"enable_reflection" json:"enable_reflection"`
}

// NewServerConfig creates a new server configuration with default values
func NewServerConfig() *ServerConfig {
	config := &ServerConfig{}
	config.SetDefaults()
	return config
}

// SetDefaults sets default values for the server configuration
func (c *ServerConfig) SetDefaults() {
	if c.ServiceName == "" {
		c.ServiceName = "unknown-service"
	}

	if c.ServiceVersion == "" {
		c.ServiceVersion = "v1.0.0"
	}

	if c.HTTPPort == "" {
		c.HTTPPort = "8080"
	}

	if c.GRPCPort == "" {
		c.GRPCPort = "9090"
	}

	// Enable HTTP by default
	if !c.EnableHTTP && !c.EnableGRPC {
		c.EnableHTTP = true
	}

	c.EnableReady = true
	c.GracefulShutdown = true

	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = 30 * time.Second
	}

	if c.StartupTimeout == 0 {
		c.StartupTimeout = 30 * time.Second
	}

	// Set discovery defaults
	c.setDiscoveryDefaults()

	// Set middleware defaults
	c.setMiddlewareDefaults()

	// Set transport defaults
	c.setHTTPDefaults()
	c.setGRPCDefaults()
}

// setDiscoveryDefaults sets default values for discovery configuration
func (c *ServerConfig) setDiscoveryDefaults() {
	if c.Discovery.ServiceName == "" {
		c.Discovery.ServiceName = c.ServiceName
	}

	if c.Discovery.HealthCheckPath == "" {
		c.Discovery.HealthCheckPath = "/health"
	}

	if c.Discovery.HealthCheckInterval == 0 {
		c.Discovery.HealthCheckInterval = 10 * time.Second
	}

	if c.Discovery.RetryAttempts == 0 {
		c.Discovery.RetryAttempts = 3
	}

	if c.Discovery.RetryInterval == 0 {
		c.Discovery.RetryInterval = 5 * time.Second
	}
}

// setMiddlewareDefaults sets default values for middleware configuration
func (c *ServerConfig) setMiddlewareDefaults() {
	c.Middleware.EnableCORS = true
	c.Middleware.EnableRequestLogger = true

	if c.Middleware.TimeoutDuration == 0 {
		c.Middleware.TimeoutDuration = 30 * time.Second
	}

	if c.Middleware.RateLimitRPS == 0 {
		c.Middleware.RateLimitRPS = 100
	}
}

// setHTTPDefaults sets default values for HTTP transport configuration
func (c *ServerConfig) setHTTPDefaults() {
	if c.HTTPConfig.ReadTimeout == 0 {
		c.HTTPConfig.ReadTimeout = 15 * time.Second
	}

	if c.HTTPConfig.WriteTimeout == 0 {
		c.HTTPConfig.WriteTimeout = 15 * time.Second
	}

	if c.HTTPConfig.IdleTimeout == 0 {
		c.HTTPConfig.IdleTimeout = 60 * time.Second
	}

	if c.HTTPConfig.MaxHeaderBytes == 0 {
		c.HTTPConfig.MaxHeaderBytes = 1 << 20 // 1MB
	}
}

// setGRPCDefaults sets default values for gRPC transport configuration
func (c *ServerConfig) setGRPCDefaults() {
	if c.GRPCConfig.MaxRecvMsgSize == 0 {
		c.GRPCConfig.MaxRecvMsgSize = 4 << 20 // 4MB
	}

	if c.GRPCConfig.MaxSendMsgSize == 0 {
		c.GRPCConfig.MaxSendMsgSize = 4 << 20 // 4MB
	}

	if c.GRPCConfig.KeepaliveTime == 0 {
		c.GRPCConfig.KeepaliveTime = 2 * time.Hour
	}

	if c.GRPCConfig.KeepaliveTimeout == 0 {
		c.GRPCConfig.KeepaliveTimeout = 20 * time.Second
	}

	if c.GRPCConfig.MaxConnectionIdle == 0 {
		c.GRPCConfig.MaxConnectionIdle = 15 * time.Minute
	}
}

// Validate validates the server configuration
func (c *ServerConfig) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}

	if !c.EnableHTTP && !c.EnableGRPC {
		return fmt.Errorf("at least one transport (HTTP or gRPC) must be enabled")
	}

	if c.EnableHTTP {
		if err := c.validatePort(c.HTTPPort, "HTTP"); err != nil {
			return err
		}
	}

	if c.EnableGRPC {
		if err := c.validatePort(c.GRPCPort, "gRPC"); err != nil {
			return err
		}
	}

	if c.EnableHTTP && c.EnableGRPC && c.HTTPPort == c.GRPCPort {
		return fmt.Errorf("HTTP and gRPC ports cannot be the same")
	}

	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}

	if c.StartupTimeout <= 0 {
		return fmt.Errorf("startup timeout must be positive")
	}

	return c.validateDiscoveryConfig()
}

// validatePort validates that a port string is valid
func (c *ServerConfig) validatePort(port, transportType string) error {
	if port == "" {
		return fmt.Errorf("%s port is required", transportType)
	}

	// Check if it's just a port number
	if portNum, err := strconv.Atoi(port); err == nil {
		if portNum < 1 || portNum > 65535 {
			return fmt.Errorf("%s port must be between 1 and 65535", transportType)
		}
		return nil
	}

	// Check if it's a valid address:port format
	if strings.Contains(port, ":") {
		host, portStr, err := net.SplitHostPort(port)
		if err != nil {
			return fmt.Errorf("invalid %s address format: %v", transportType, err)
		}

		if host != "" {
			if ip := net.ParseIP(host); ip == nil {
				return fmt.Errorf("invalid %s host address: %s", transportType, host)
			}
		}

		if portNum, err := strconv.Atoi(portStr); err != nil {
			return fmt.Errorf("invalid %s port number: %v", transportType, err)
		} else if portNum < 1 || portNum > 65535 {
			return fmt.Errorf("%s port must be between 1 and 65535", transportType)
		}
	}

	return nil
}

// validateDiscoveryConfig validates the discovery configuration
func (c *ServerConfig) validateDiscoveryConfig() error {
	if !c.Discovery.Enabled {
		return nil
	}

	if c.Discovery.ServiceName == "" {
		return fmt.Errorf("discovery service name is required when discovery is enabled")
	}

	if c.Discovery.HealthCheckInterval <= 0 {
		return fmt.Errorf("discovery health check interval must be positive")
	}

	if c.Discovery.RetryAttempts < 0 {
		return fmt.Errorf("discovery retry attempts cannot be negative")
	}

	if c.Discovery.RetryInterval <= 0 {
		return fmt.Errorf("discovery retry interval must be positive")
	}

	return nil
}

// GetHTTPAddress returns the full HTTP address
func (c *ServerConfig) GetHTTPAddress() string {
	if strings.Contains(c.HTTPPort, ":") {
		return c.HTTPPort
	}
	return ":" + c.HTTPPort
}

// GetGRPCAddress returns the full gRPC address
func (c *ServerConfig) GetGRPCAddress() string {
	if strings.Contains(c.GRPCPort, ":") {
		return c.GRPCPort
	}
	return ":" + c.GRPCPort
}

// Clone creates a deep copy of the server configuration
func (c *ServerConfig) Clone() *ServerConfig {
	clone := *c

	// Deep copy slices
	if c.Discovery.Tags != nil {
		clone.Discovery.Tags = make([]string, len(c.Discovery.Tags))
		copy(clone.Discovery.Tags, c.Discovery.Tags)
	}

	if c.HTTPConfig.TrustedProxies != nil {
		clone.HTTPConfig.TrustedProxies = make([]string, len(c.HTTPConfig.TrustedProxies))
		copy(clone.HTTPConfig.TrustedProxies, c.HTTPConfig.TrustedProxies)
	}

	return &clone
}

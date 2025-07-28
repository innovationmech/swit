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
	"strconv"
	"time"
)

// ServerConfig holds the complete configuration for a base server instance
// It includes transport, discovery, and middleware configuration
type ServerConfig struct {
	ServiceName     string           `yaml:"service_name" json:"service_name"`
	HTTP            HTTPConfig       `yaml:"http" json:"http"`
	GRPC            GRPCConfig       `yaml:"grpc" json:"grpc"`
	Discovery       DiscoveryConfig  `yaml:"discovery" json:"discovery"`
	Middleware      MiddlewareConfig `yaml:"middleware" json:"middleware"`
	ShutdownTimeout time.Duration    `yaml:"shutdown_timeout" json:"shutdown_timeout"`
}

// HTTPConfig holds HTTP transport specific configuration
type HTTPConfig struct {
	Port        string `yaml:"port" json:"port"`
	EnableReady bool   `yaml:"enable_ready" json:"enable_ready"`
	Enabled     bool   `yaml:"enabled" json:"enabled"`
}

// GRPCConfig holds gRPC transport specific configuration
type GRPCConfig struct {
	Port                string `yaml:"port" json:"port"`
	EnableKeepalive     bool   `yaml:"enable_keepalive" json:"enable_keepalive"`
	EnableReflection    bool   `yaml:"enable_reflection" json:"enable_reflection"`
	EnableHealthService bool   `yaml:"enable_health_service" json:"enable_health_service"`
	Enabled             bool   `yaml:"enabled" json:"enabled"`
}

// DiscoveryConfig holds service discovery configuration
type DiscoveryConfig struct {
	Address     string   `yaml:"address" json:"address"`
	ServiceName string   `yaml:"service_name" json:"service_name"`
	Tags        []string `yaml:"tags" json:"tags"`
	Enabled     bool     `yaml:"enabled" json:"enabled"`
}

// MiddlewareConfig holds middleware configuration flags
type MiddlewareConfig struct {
	EnableCORS      bool `yaml:"enable_cors" json:"enable_cors"`
	EnableAuth      bool `yaml:"enable_auth" json:"enable_auth"`
	EnableRateLimit bool `yaml:"enable_rate_limit" json:"enable_rate_limit"`
	EnableLogging   bool `yaml:"enable_logging" json:"enable_logging"`
}

// NewServerConfig creates a new ServerConfig with default values
func NewServerConfig() *ServerConfig {
	config := &ServerConfig{}
	config.SetDefaults()
	return config
}

// SetDefaults sets default values for all configuration options
func (c *ServerConfig) SetDefaults() {
	if c.ServiceName == "" {
		c.ServiceName = "swit-service"
	}

	// HTTP defaults
	if c.HTTP.Port == "" {
		c.HTTP.Port = "8080"
	}
	c.HTTP.Enabled = true
	c.HTTP.EnableReady = true

	// gRPC defaults
	if c.GRPC.Port == "" {
		c.GRPC.Port = c.calculateGRPCPort()
	}
	c.GRPC.Enabled = true
	c.GRPC.EnableKeepalive = true
	c.GRPC.EnableReflection = true
	c.GRPC.EnableHealthService = true

	// Discovery defaults
	if c.Discovery.Address == "" {
		c.Discovery.Address = "127.0.0.1:8500"
	}
	if c.Discovery.ServiceName == "" {
		c.Discovery.ServiceName = c.ServiceName
	}
	c.Discovery.Enabled = true
	if len(c.Discovery.Tags) == 0 {
		c.Discovery.Tags = []string{"v1"}
	}

	// Middleware defaults
	c.Middleware.EnableCORS = true
	c.Middleware.EnableAuth = false
	c.Middleware.EnableRateLimit = false
	c.Middleware.EnableLogging = true

	// Server defaults
	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = 5 * time.Second
	}
}

// Validate validates the configuration and returns any errors
func (c *ServerConfig) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}

	// Validate HTTP configuration
	if c.HTTP.Enabled {
		if c.HTTP.Port == "" {
			return fmt.Errorf("http.port is required when HTTP is enabled")
		}
		if err := c.validatePort(c.HTTP.Port, "http.port"); err != nil {
			return err
		}
	}

	// Validate gRPC configuration
	if c.GRPC.Enabled {
		if c.GRPC.Port == "" {
			return fmt.Errorf("grpc.port is required when gRPC is enabled")
		}
		if err := c.validatePort(c.GRPC.Port, "grpc.port"); err != nil {
			return err
		}
	}

	// Validate that at least one transport is enabled
	if !c.HTTP.Enabled && !c.GRPC.Enabled {
		return fmt.Errorf("at least one transport (HTTP or gRPC) must be enabled")
	}

	// Validate ports are different if both transports are enabled
	if c.HTTP.Enabled && c.GRPC.Enabled && c.HTTP.Port == c.GRPC.Port {
		return fmt.Errorf("http.port and grpc.port must be different")
	}

	// Validate discovery configuration
	if c.Discovery.Enabled {
		if c.Discovery.Address == "" {
			return fmt.Errorf("discovery.address is required when discovery is enabled")
		}
		if c.Discovery.ServiceName == "" {
			return fmt.Errorf("discovery.service_name is required when discovery is enabled")
		}
	}

	// Validate shutdown timeout
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown_timeout must be positive")
	}

	return nil
}

// validatePort validates that a port string is a valid port number
func (c *ServerConfig) validatePort(port, fieldName string) error {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("%s must be a valid port number: %w", fieldName, err)
	}
	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535, got %d", fieldName, portNum)
	}
	return nil
}

// calculateGRPCPort calculates a default gRPC port based on HTTP port
// Uses the same logic as the existing transport package
func (c *ServerConfig) calculateGRPCPort() string {
	if c.HTTP.Port == "" {
		return "9080" // Default if no HTTP port set
	}

	httpPort, err := strconv.Atoi(c.HTTP.Port)
	if err != nil {
		return "9080" // Fallback to default
	}

	// Add 1000 to HTTP port for gRPC (e.g., 8080 -> 9080)
	grpcPort := httpPort + 1000
	return strconv.Itoa(grpcPort)
}

// GetHTTPAddress returns the full HTTP address with colon prefix
func (c *ServerConfig) GetHTTPAddress() string {
	if !c.HTTP.Enabled {
		return ""
	}
	return ":" + c.HTTP.Port
}

// GetGRPCAddress returns the full gRPC address with colon prefix
func (c *ServerConfig) GetGRPCAddress() string {
	if !c.GRPC.Enabled {
		return ""
	}
	return ":" + c.GRPC.Port
}

// IsHTTPEnabled returns true if HTTP transport is enabled
func (c *ServerConfig) IsHTTPEnabled() bool {
	return c.HTTP.Enabled
}

// IsGRPCEnabled returns true if gRPC transport is enabled
func (c *ServerConfig) IsGRPCEnabled() bool {
	return c.GRPC.Enabled
}

// IsDiscoveryEnabled returns true if service discovery is enabled
func (c *ServerConfig) IsDiscoveryEnabled() bool {
	return c.Discovery.Enabled
}

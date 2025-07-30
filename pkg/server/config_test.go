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

func TestNewServerConfig(t *testing.T) {
	config := NewServerConfig()

	// Test default values are set
	assert.Equal(t, "unknown-service", config.ServiceName)
	assert.Equal(t, "v1.0.0", config.ServiceVersion)
	assert.Equal(t, "8080", config.HTTPPort)
	assert.Equal(t, "9090", config.GRPCPort)
	assert.True(t, config.EnableHTTP)
	assert.False(t, config.EnableGRPC)
	assert.True(t, config.EnableReady)
	assert.True(t, config.GracefulShutdown)
	assert.Equal(t, 30*time.Second, config.ShutdownTimeout)
	assert.Equal(t, 30*time.Second, config.StartupTimeout)
}

func TestServerConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config *ServerConfig
		want   func(*ServerConfig)
	}{
		{
			name:   "empty config",
			config: &ServerConfig{},
			want: func(c *ServerConfig) {
				assert.Equal(t, "unknown-service", c.ServiceName)
				assert.Equal(t, "v1.0.0", c.ServiceVersion)
				assert.Equal(t, "8080", c.HTTPPort)
				assert.Equal(t, "9090", c.GRPCPort)
				assert.True(t, c.EnableHTTP)
			},
		},
		{
			name: "partial config",
			config: &ServerConfig{
				ServiceName: "test-service",
				EnableGRPC:  true,
			},
			want: func(c *ServerConfig) {
				assert.Equal(t, "test-service", c.ServiceName)
				assert.Equal(t, "v1.0.0", c.ServiceVersion)
				assert.True(t, c.EnableGRPC)
				assert.False(t, c.EnableHTTP) // Should remain false since gRPC is explicitly enabled
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.SetDefaults()
			tt.want(tt.config)
		})
	}
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &ServerConfig{
				ServiceName:     "test-service",
				HTTPPort:        "8080",
				GRPCPort:        "9090",
				EnableHTTP:      true,
				EnableGRPC:      true,
				ShutdownTimeout: 30 * time.Second,
				StartupTimeout:  30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty service name",
			config: &ServerConfig{
				EnableHTTP:      true,
				ShutdownTimeout: 30 * time.Second,
				StartupTimeout:  30 * time.Second,
			},
			wantErr: true,
			errMsg:  "service name is required",
		},
		{
			name: "no transports enabled",
			config: &ServerConfig{
				ServiceName:     "test-service",
				EnableHTTP:      false,
				EnableGRPC:      false,
				ShutdownTimeout: 30 * time.Second,
				StartupTimeout:  30 * time.Second,
			},
			wantErr: true,
			errMsg:  "at least one transport (HTTP or gRPC) must be enabled",
		},
		{
			name: "invalid HTTP port",
			config: &ServerConfig{
				ServiceName:     "test-service",
				HTTPPort:        "invalid:port",
				EnableHTTP:      true,
				ShutdownTimeout: 30 * time.Second,
				StartupTimeout:  30 * time.Second,
			},
			wantErr: true,
			errMsg:  "invalid HTTP host address",
		},
		{
			name: "same ports for HTTP and gRPC",
			config: &ServerConfig{
				ServiceName:     "test-service",
				HTTPPort:        "8080",
				GRPCPort:        "8080",
				EnableHTTP:      true,
				EnableGRPC:      true,
				ShutdownTimeout: 30 * time.Second,
				StartupTimeout:  30 * time.Second,
			},
			wantErr: true,
			errMsg:  "HTTP and gRPC ports cannot be the same",
		},
		{
			name: "negative shutdown timeout",
			config: &ServerConfig{
				ServiceName:     "test-service",
				HTTPPort:        "8080",
				EnableHTTP:      true,
				ShutdownTimeout: -1 * time.Second,
				StartupTimeout:  30 * time.Second,
			},
			wantErr: true,
			errMsg:  "shutdown timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServerConfig_ValidatePort(t *testing.T) {
	tests := []struct {
		name          string
		port          string
		transportType string
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "valid port number",
			port:          "8080",
			transportType: "HTTP",
			wantErr:       false,
		},
		{
			name:          "valid address:port",
			port:          "127.0.0.1:8080",
			transportType: "HTTP",
			wantErr:       false,
		},
		{
			name:          "valid :port format",
			port:          ":8080",
			transportType: "HTTP",
			wantErr:       false,
		},
		{
			name:          "port too low",
			port:          "0",
			transportType: "HTTP",
			wantErr:       true,
			errMsg:        "HTTP port must be between 1 and 65535",
		},
		{
			name:          "port too high",
			port:          "65536",
			transportType: "HTTP",
			wantErr:       true,
			errMsg:        "HTTP port must be between 1 and 65535",
		},
		{
			name:          "invalid address format",
			port:          "invalid:port:format",
			transportType: "HTTP",
			wantErr:       true,
			errMsg:        "invalid HTTP address format",
		},
		{
			name:          "invalid IP address",
			port:          "999.999.999.999:8080",
			transportType: "HTTP",
			wantErr:       true,
			errMsg:        "invalid HTTP host address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ServerConfig{}
			err := config.validatePort(tt.port, tt.transportType)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServerConfig_GetAddresses(t *testing.T) {
	config := &ServerConfig{
		HTTPPort: "8080",
		GRPCPort: "127.0.0.1:9090",
	}

	assert.Equal(t, ":8080", config.GetHTTPAddress())
	assert.Equal(t, "127.0.0.1:9090", config.GetGRPCAddress())
}

func TestServerConfig_Clone(t *testing.T) {
	original := &ServerConfig{
		ServiceName: "test-service",
		HTTPPort:    "8080",
		Discovery: DiscoveryConfig{
			Tags: []string{"tag1", "tag2"},
		},
		HTTPConfig: HTTPTransportConfig{
			TrustedProxies: []string{"127.0.0.1", "192.168.1.1"},
		},
	}

	clone := original.Clone()

	// Test that clone is a separate instance
	assert.NotSame(t, original, clone)

	// Test that values are copied
	assert.Equal(t, original.ServiceName, clone.ServiceName)
	assert.Equal(t, original.HTTPPort, clone.HTTPPort)

	// Test that slices are deep copied
	assert.NotSame(t, original.Discovery.Tags, clone.Discovery.Tags)
	assert.Equal(t, original.Discovery.Tags, clone.Discovery.Tags)

	assert.NotSame(t, original.HTTPConfig.TrustedProxies, clone.HTTPConfig.TrustedProxies)
	assert.Equal(t, original.HTTPConfig.TrustedProxies, clone.HTTPConfig.TrustedProxies)

	// Test that modifying clone doesn't affect original
	clone.ServiceName = "modified-service"
	clone.Discovery.Tags[0] = "modified-tag"
	clone.HTTPConfig.TrustedProxies[0] = "modified-proxy"

	assert.Equal(t, "test-service", original.ServiceName)
	assert.Equal(t, "tag1", original.Discovery.Tags[0])
	assert.Equal(t, "127.0.0.1", original.HTTPConfig.TrustedProxies[0])
}

func TestDiscoveryConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *ServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "discovery disabled",
			config: &ServerConfig{
				ServiceName:     "test-service",
				HTTPPort:        "8080",
				EnableHTTP:      true,
				ShutdownTimeout: 30 * time.Second,
				StartupTimeout:  30 * time.Second,
				Discovery: DiscoveryConfig{
					Enabled: false,
				},
			},
			wantErr: false,
		},
		{
			name: "discovery enabled with valid config",
			config: &ServerConfig{
				ServiceName:     "test-service",
				HTTPPort:        "8080",
				EnableHTTP:      true,
				ShutdownTimeout: 30 * time.Second,
				StartupTimeout:  30 * time.Second,
				Discovery: DiscoveryConfig{
					Enabled:             true,
					ServiceName:         "test-service",
					HealthCheckInterval: 10 * time.Second,
					RetryAttempts:       3,
					RetryInterval:       5 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "discovery enabled without service name",
			config: &ServerConfig{
				ServiceName:     "test-service",
				HTTPPort:        "8080",
				EnableHTTP:      true,
				ShutdownTimeout: 30 * time.Second,
				StartupTimeout:  30 * time.Second,
				Discovery: DiscoveryConfig{
					Enabled:             true,
					HealthCheckInterval: 10 * time.Second,
					RetryAttempts:       3,
					RetryInterval:       5 * time.Second,
				},
			},
			wantErr: true,
			errMsg:  "discovery service name is required when discovery is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

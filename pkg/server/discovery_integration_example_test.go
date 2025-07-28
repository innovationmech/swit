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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiscoveryFailureHandling_CompleteExample demonstrates the complete solution
// showing how different failure modes work in practice
func TestDiscoveryFailureHandling_CompleteExample(t *testing.T) {
	t.Run("complete_example_development_environment", func(t *testing.T) {
		// Development configuration - graceful failure handling
		config := NewServerConfig()
		config.ServiceName = "swit-serve-dev"
		config.HTTP.Port = "0" // Dynamic port allocation
		config.GRPC.Port = "0"

		// Configure discovery for development
		config.Discovery.Enabled = true
		config.Discovery.ServiceName = "swit-serve-dev"
		config.Discovery.FailureMode = DiscoveryFailureModeGraceful
		config.Discovery.HealthCheckRequired = false
		config.Discovery.RegistrationTimeout = 30 * time.Second
		config.Discovery.Tags = []string{"dev", "v1"}

		// Validate configuration
		err := config.Validate()
		require.NoError(t, err, "Development configuration should be valid")

		// Check configuration helpers
		assert.True(t, config.IsDiscoveryFailureModeGraceful())
		assert.False(t, config.IsDiscoveryFailureModeFailFast())
		assert.False(t, config.IsDiscoveryFailureModeStrict())

		// Check description
		desc := config.GetDiscoveryFailureModeDescription()
		assert.Contains(t, strings.ToLower(desc), "graceful")

		// Create server with failing discovery
		registrar := &MockServiceRegistrar{}
		mockDiscovery := &MockDiscoveryManager{shouldFailRegistration: true}

		server, err := NewBusinessServerCore(config, registrar, nil)
		require.NoError(t, err)
		server.SetDiscoveryManager(mockDiscovery)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Server should start successfully despite discovery failure
		err = server.Start(ctx)
		assert.NoError(t, err, "Development server should start despite discovery failure")

		err = server.Stop(ctx)
		assert.NoError(t, err, "Should stop cleanly")
	})

	t.Run("complete_example_production_environment", func(t *testing.T) {
		// Production configuration - fail-fast behavior
		config := NewServerConfig()
		config.ServiceName = "swit-serve-prod"
		config.HTTP.Port = "0" // Dynamic port allocation
		config.GRPC.Port = "0"

		// Configure discovery for production
		config.Discovery.Enabled = true
		config.Discovery.ServiceName = "swit-serve-prod"
		config.Discovery.FailureMode = DiscoveryFailureModeFailFast
		config.Discovery.HealthCheckRequired = false
		config.Discovery.RegistrationTimeout = 60 * time.Second
		config.Discovery.Tags = []string{"prod", "v1"}

		// Validate configuration
		err := config.Validate()
		require.NoError(t, err, "Production configuration should be valid")

		// Check configuration helpers
		assert.False(t, config.IsDiscoveryFailureModeGraceful())
		assert.True(t, config.IsDiscoveryFailureModeFailFast())
		assert.False(t, config.IsDiscoveryFailureModeStrict())

		// Check description
		desc := config.GetDiscoveryFailureModeDescription()
		assert.Contains(t, strings.ToLower(desc), "fail")

		// Test with failing discovery
		registrar := &MockServiceRegistrar{}
		mockDiscovery := &MockDiscoveryManager{shouldFailRegistration: true}

		server, err := NewBusinessServerCore(config, registrar, nil)
		require.NoError(t, err)
		server.SetDiscoveryManager(mockDiscovery)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Server should fail to start due to discovery failure
		err = server.Start(ctx)
		assert.Error(t, err, "Production server should fail to start on discovery failure")
		assert.Contains(t, err.Error(), "failed to register with service discovery")

		// Test with working discovery
		mockDiscovery.shouldFailRegistration = false
		server2, err := NewBusinessServerCore(config, registrar, nil)
		require.NoError(t, err)
		server2.SetDiscoveryManager(mockDiscovery)

		// Should start successfully
		err = server2.Start(ctx)
		assert.NoError(t, err, "Production server should start with working discovery")

		err = server2.Stop(ctx)
		assert.NoError(t, err, "Should stop cleanly")
	})

	t.Run("complete_example_strict_mode", func(t *testing.T) {
		// Critical production configuration - strict behavior
		config := NewServerConfig()
		config.ServiceName = "swit-serve-critical"
		config.HTTP.Port = "0" // Dynamic port allocation
		config.GRPC.Port = "0"

		// Configure discovery for critical production
		config.Discovery.Enabled = true
		config.Discovery.ServiceName = "swit-serve-critical"
		config.Discovery.FailureMode = DiscoveryFailureModeStrict
		config.Discovery.HealthCheckRequired = true
		config.Discovery.RegistrationTimeout = 45 * time.Second
		config.Discovery.Tags = []string{"prod", "critical", "v1"}

		// Validate configuration
		err := config.Validate()
		require.NoError(t, err, "Critical configuration should be valid")

		// Check configuration helpers
		assert.False(t, config.IsDiscoveryFailureModeGraceful())
		assert.True(t, config.IsDiscoveryFailureModeFailFast()) // Strict mode is also fail-fast
		assert.True(t, config.IsDiscoveryFailureModeStrict())

		// Check description
		desc := config.GetDiscoveryFailureModeDescription()
		assert.Contains(t, strings.ToLower(desc), "strict")

		// Test with unhealthy discovery
		registrar := &MockServiceRegistrar{}
		mockDiscovery := &MockDiscoveryManager{
			shouldFailHealth:       true, // Unhealthy discovery
			shouldFailRegistration: false,
		}

		server, err := NewBusinessServerCore(config, registrar, nil)
		require.NoError(t, err)
		server.SetDiscoveryManager(mockDiscovery)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Server should fail to start due to unhealthy discovery
		err = server.Start(ctx)
		assert.Error(t, err, "Strict mode should fail on unhealthy discovery")
		assert.Contains(t, err.Error(), "discovery service is not healthy")

		// Test with healthy discovery and working registration
		mockDiscovery.shouldFailHealth = false
		mockDiscovery.shouldFailRegistration = false

		server2, err := NewBusinessServerCore(config, registrar, nil)
		require.NoError(t, err)
		server2.SetDiscoveryManager(mockDiscovery)

		// Should start successfully
		err = server2.Start(ctx)
		assert.NoError(t, err, "Strict mode should start with healthy discovery")

		// Verify health check was called (once for the failed case, once for the successful case)
		assert.GreaterOrEqual(t, mockDiscovery.healthCallCount, 1, "Health check should be called in strict mode")

		err = server2.Stop(ctx)
		assert.NoError(t, err, "Should stop cleanly")
	})
}

// TestDiscoveryFailureHandling_ConfigurationValidation tests comprehensive validation
func TestDiscoveryFailureHandling_ConfigurationValidation(t *testing.T) {
	baseValidConfig := func() *ServerConfig {
		config := NewServerConfig()
		config.ServiceName = "test-service"
		config.Discovery.Enabled = true
		config.Discovery.Address = "localhost:8500"
		config.Discovery.ServiceName = "test-service"
		return config
	}

	tests := []struct {
		name          string
		modifyConfig  func(*ServerConfig)
		shouldBeValid bool
		errorContains string
	}{
		{
			name:          "valid_default_configuration",
			modifyConfig:  func(c *ServerConfig) {}, // No changes
			shouldBeValid: true,
		},
		{
			name: "valid_graceful_mode",
			modifyConfig: func(c *ServerConfig) {
				c.Discovery.FailureMode = DiscoveryFailureModeGraceful
			},
			shouldBeValid: true,
		},
		{
			name: "valid_fail_fast_mode",
			modifyConfig: func(c *ServerConfig) {
				c.Discovery.FailureMode = DiscoveryFailureModeFailFast
			},
			shouldBeValid: true,
		},
		{
			name: "valid_strict_mode",
			modifyConfig: func(c *ServerConfig) {
				c.Discovery.FailureMode = DiscoveryFailureModeStrict
				c.Discovery.HealthCheckRequired = true
			},
			shouldBeValid: true,
		},
		{
			name: "invalid_failure_mode",
			modifyConfig: func(c *ServerConfig) {
				c.Discovery.FailureMode = "invalid_mode"
			},
			shouldBeValid: false,
			errorContains: "failure_mode must be one of",
		},
		{
			name: "invalid_negative_timeout",
			modifyConfig: func(c *ServerConfig) {
				c.Discovery.RegistrationTimeout = -1 * time.Second
			},
			shouldBeValid: false,
			errorContains: "registration_timeout must be positive",
		},
		{
			name: "invalid_excessive_timeout",
			modifyConfig: func(c *ServerConfig) {
				c.Discovery.RegistrationTimeout = 10 * time.Minute
			},
			shouldBeValid: false,
			errorContains: "registration_timeout should not exceed 5 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := baseValidConfig()
			tt.modifyConfig(config)

			err := config.Validate()

			if tt.shouldBeValid {
				assert.NoError(t, err, "Configuration should be valid")
			} else {
				assert.Error(t, err, "Configuration should be invalid")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			}
		})
	}
}

// TestDiscoveryFailureHandling_BackwardCompatibility ensures existing configurations work
func TestDiscoveryFailureHandling_BackwardCompatibility(t *testing.T) {
	// Simulate a configuration file from before the feature was added
	config := &ServerConfig{
		ServiceName: "legacy-service",
		HTTP: HTTPConfig{
			Enabled: true,
			Port:    "9000",
		},
		GRPC: GRPCConfig{
			Enabled: true,
			Port:    "9001",
		},
		Discovery: DiscoveryConfig{
			Enabled:     true,
			Address:     "localhost:8500",
			ServiceName: "legacy-service",
			Tags:        []string{"v1"},
			// Note: No FailureMode, HealthCheckRequired, or RegistrationTimeout set
		},
	}

	// Apply defaults (simulates loading from YAML)
	config.SetDefaults()

	// Should be valid after applying defaults
	err := config.Validate()
	assert.NoError(t, err, "Legacy configuration should be valid with defaults")

	// Check that backward-compatible defaults are applied
	assert.Equal(t, DiscoveryFailureModeGraceful, config.Discovery.FailureMode,
		"Should default to graceful mode for backward compatibility")
	assert.False(t, config.Discovery.HealthCheckRequired,
		"Should default to no health check requirement")
	assert.Equal(t, 30*time.Second, config.Discovery.RegistrationTimeout,
		"Should have reasonable default timeout")

	// Behavior should be backward compatible (graceful)
	assert.True(t, config.IsDiscoveryFailureModeGraceful())
	assert.False(t, config.IsDiscoveryFailureModeFailFast())
}

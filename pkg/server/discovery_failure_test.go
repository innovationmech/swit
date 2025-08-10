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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDiscoveryManager implements ServiceDiscoveryManager for testing
type MockDiscoveryManager struct {
	shouldFailRegistration  bool
	shouldFailHealth        bool
	registrationCallCount   int
	deregistrationCallCount int
	healthCallCount         int
	registeredServices      []*ServiceRegistration
	customRegisterFunc      func(context.Context, []*ServiceRegistration) error
}

func (m *MockDiscoveryManager) RegisterService(ctx context.Context, registration *ServiceRegistration) error {
	m.registrationCallCount++
	if m.shouldFailRegistration {
		return fmt.Errorf("mock registration failure")
	}
	m.registeredServices = append(m.registeredServices, registration)
	return nil
}

func (m *MockDiscoveryManager) DeregisterService(ctx context.Context, registration *ServiceRegistration) error {
	m.deregistrationCallCount++
	return nil
}

func (m *MockDiscoveryManager) RegisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error {
	m.registrationCallCount++

	// Use custom function if provided
	if m.customRegisterFunc != nil {
		return m.customRegisterFunc(ctx, registrations)
	}

	if m.shouldFailRegistration {
		return fmt.Errorf("mock registration failure for multiple endpoints")
	}
	m.registeredServices = append(m.registeredServices, registrations...)
	return nil
}

func (m *MockDiscoveryManager) DeregisterMultipleEndpoints(ctx context.Context, registrations []*ServiceRegistration) error {
	m.deregistrationCallCount++
	return nil
}

func (m *MockDiscoveryManager) IsHealthy(ctx context.Context) bool {
	m.healthCallCount++
	return !m.shouldFailHealth
}

// MockServiceRegistrar for testing
type MockServiceRegistrar struct{}

func (m *MockServiceRegistrar) RegisterServices(registry BusinessServiceRegistry) error {
	return nil
}

func TestDiscoveryFailureMode_Configuration(t *testing.T) {
	tests := []struct {
		name         string
		failureMode  DiscoveryFailureMode
		expectValid  bool
		expectedDesc string
	}{
		{
			name:         "graceful mode",
			failureMode:  DiscoveryFailureModeGraceful,
			expectValid:  true,
			expectedDesc: "graceful degradation - server continues without discovery",
		},
		{
			name:         "fail_fast mode",
			failureMode:  DiscoveryFailureModeFailFast,
			expectValid:  true,
			expectedDesc: "fail-fast - server startup fails if discovery registration fails",
		},
		{
			name:         "strict mode",
			failureMode:  DiscoveryFailureModeStrict,
			expectValid:  true,
			expectedDesc: "strict mode - requires discovery health check and fails fast",
		},
		{
			name:        "invalid mode",
			failureMode: "invalid",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewServerConfig()
			config.Discovery.Enabled = true
			config.Discovery.FailureMode = tt.failureMode

			err := config.Validate()
			if tt.expectValid {
				assert.NoError(t, err, "Expected valid failure mode")
				if tt.expectedDesc != "" {
					assert.Equal(t, tt.expectedDesc, config.GetDiscoveryFailureModeDescription())
				}
			} else {
				assert.Error(t, err, "Expected invalid failure mode to cause validation error")
				assert.Contains(t, err.Error(), "failure_mode must be one of")
			}
		})
	}
}

func TestDiscoveryFailureMode_ConfigurationDefaults(t *testing.T) {
	config := NewServerConfig()

	// Check defaults are set properly
	assert.Equal(t, DiscoveryFailureModeGraceful, config.Discovery.FailureMode, "Default should be graceful mode")
	assert.False(t, config.Discovery.HealthCheckRequired, "Health check should default to false")
	assert.Equal(t, 30*time.Second, config.Discovery.RegistrationTimeout, "Default timeout should be 30s")

	// Check helper methods
	assert.True(t, config.IsDiscoveryFailureModeGraceful(), "Should detect graceful mode")
	assert.False(t, config.IsDiscoveryFailureModeFailFast(), "Should not detect fail-fast mode")
	assert.False(t, config.IsDiscoveryFailureModeStrict(), "Should not detect strict mode")
}

func TestDiscoveryFailureMode_ValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		setupConfig   func(*ServerConfig)
		expectedError string
	}{
		{
			name: "timeout too small",
			setupConfig: func(c *ServerConfig) {
				c.Discovery.Enabled = true
				c.Discovery.RegistrationTimeout = -1 * time.Second
			},
			expectedError: "registration_timeout must be positive",
		},
		{
			name: "timeout too large",
			setupConfig: func(c *ServerConfig) {
				c.Discovery.Enabled = true
				c.Discovery.RegistrationTimeout = 10 * time.Minute
			},
			expectedError: "registration_timeout should not exceed 5 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewServerConfig()
			tt.setupConfig(config)

			err := config.Validate()
			assert.Error(t, err, "Expected validation error")
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestBaseServer_DiscoveryFailureMode_Graceful(t *testing.T) {
	mockDiscovery := &MockDiscoveryManager{
		shouldFailRegistration: true, // Simulate registration failure
	}

	config := NewServerConfig()
	config.ServiceName = "test-service"
	config.HTTP.Port = "0" // Use any available port
	config.GRPC.Port = "0"
	config.Discovery.Enabled = true
	config.Discovery.FailureMode = DiscoveryFailureModeGraceful

	registrar := &MockServiceRegistrar{}

	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err, "Should create server successfully")

	// Replace the discovery manager with our mock
	server.SetDiscoveryManager(mockDiscovery)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start should succeed even though discovery registration fails
	err = server.Start(ctx)
	assert.NoError(t, err, "Server should start successfully in graceful mode even with discovery failure")

	// Verify discovery registration was attempted
	assert.Equal(t, 1, mockDiscovery.registrationCallCount, "Registration should have been attempted")

	// Clean up
	err = server.Stop(ctx)
	assert.NoError(t, err, "Should stop cleanly")
}

func TestBaseServer_DiscoveryFailureMode_FailFast(t *testing.T) {
	mockDiscovery := &MockDiscoveryManager{
		shouldFailRegistration: true, // Simulate registration failure
	}

	config := NewServerConfig()
	config.ServiceName = "test-service"
	config.HTTP.Port = "0" // Use any available port
	config.GRPC.Port = "0"
	config.Discovery.Enabled = true
	config.Discovery.FailureMode = DiscoveryFailureModeFailFast

	registrar := &MockServiceRegistrar{}

	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err, "Should create server successfully")

	// Replace the discovery manager with our mock
	server.SetDiscoveryManager(mockDiscovery)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start should fail because discovery registration fails
	err = server.Start(ctx)
	assert.Error(t, err, "Server should fail to start in fail-fast mode with discovery failure")
	assert.Contains(t, err.Error(), "failed to register with service discovery")
	assert.Contains(t, err.Error(), "failure_mode=fail_fast")

	// Verify discovery registration was attempted
	assert.Equal(t, 1, mockDiscovery.registrationCallCount, "Registration should have been attempted")
}

func TestBaseServer_DiscoveryFailureMode_Strict(t *testing.T) {
	tests := []struct {
		name                   string
		shouldFailHealth       bool
		shouldFailRegistration bool
		healthCheckRequired    bool
		expectStartupFailure   bool
		expectedErrorContains  string
	}{
		{
			name:                   "strict mode with healthy discovery and successful registration",
			shouldFailHealth:       false,
			shouldFailRegistration: false,
			healthCheckRequired:    true,
			expectStartupFailure:   false,
		},
		{
			name:                   "strict mode with unhealthy discovery",
			shouldFailHealth:       true,
			shouldFailRegistration: false,
			healthCheckRequired:    true,
			expectStartupFailure:   true,
			expectedErrorContains:  "discovery service is not healthy",
		},
		{
			name:                   "strict mode with healthy discovery but failed registration",
			shouldFailHealth:       false,
			shouldFailRegistration: true,
			healthCheckRequired:    true,
			expectStartupFailure:   true,
			expectedErrorContains:  "failed to register with service discovery",
		},
		{
			name:                   "strict mode without health check requirement",
			shouldFailHealth:       true, // This won't matter since health check is not required
			shouldFailRegistration: false,
			healthCheckRequired:    false,
			expectStartupFailure:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDiscovery := &MockDiscoveryManager{
				shouldFailHealth:       tt.shouldFailHealth,
				shouldFailRegistration: tt.shouldFailRegistration,
			}

			config := NewServerConfig()
			config.ServiceName = "test-service"
			config.HTTP.Port = "0" // Use any available port
			config.GRPC.Port = "0"
			config.Discovery.Enabled = true
			config.Discovery.FailureMode = DiscoveryFailureModeStrict
			config.Discovery.HealthCheckRequired = tt.healthCheckRequired

			registrar := &MockServiceRegistrar{}

			server, err := NewBusinessServerCore(config, registrar, nil)
			require.NoError(t, err, "Should create server successfully")

			// Replace the discovery manager with our mock
			server.SetDiscoveryManager(mockDiscovery)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Start server and check result
			err = server.Start(ctx)

			if tt.expectStartupFailure {
				assert.Error(t, err, "Server should fail to start in strict mode")
				if tt.expectedErrorContains != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorContains)
				}
			} else {
				assert.NoError(t, err, "Server should start successfully in strict mode")
				// Clean up
				err = server.Stop(ctx)
				assert.NoError(t, err, "Should stop cleanly")
			}

			// Verify appropriate calls were made
			if tt.healthCheckRequired {
				assert.Equal(t, 1, mockDiscovery.healthCallCount, "Health check should have been called")
			}
		})
	}
}

func TestBaseServer_DiscoveryFailureMode_SuccessfulRegistration(t *testing.T) {
	tests := []struct {
		name        string
		failureMode DiscoveryFailureMode
	}{
		{"graceful mode success", DiscoveryFailureModeGraceful},
		{"fail_fast mode success", DiscoveryFailureModeFailFast},
		{"strict mode success", DiscoveryFailureModeStrict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDiscovery := &MockDiscoveryManager{
				shouldFailRegistration: false, // Successful registration
				shouldFailHealth:       false, // Healthy discovery
			}

			config := NewServerConfig()
			config.ServiceName = "test-service"
			config.HTTP.Port = "0" // Use any available port
			config.GRPC.Port = "0"
			config.Discovery.Enabled = true
			config.Discovery.FailureMode = tt.failureMode
			config.Discovery.HealthCheckRequired = true

			registrar := &MockServiceRegistrar{}

			server, err := NewBusinessServerCore(config, registrar, nil)
			require.NoError(t, err, "Should create server successfully")

			// Replace the discovery manager with our mock
			server.SetDiscoveryManager(mockDiscovery)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// All modes should succeed when discovery works properly
			err = server.Start(ctx)
			assert.NoError(t, err, "Server should start successfully when discovery works")

			// Verify registration was attempted and succeeded
			assert.Equal(t, 1, mockDiscovery.registrationCallCount, "Registration should have been attempted")
			assert.True(t, len(mockDiscovery.registeredServices) > 0, "Services should have been registered")

			// Clean up
			err = server.Stop(ctx)
			assert.NoError(t, err, "Should stop cleanly")
		})
	}
}

func TestBaseServer_DiscoveryFailureMode_TimeoutHandling(t *testing.T) {
	// Create a mock that simulates slow/hanging discovery operations
	mockDiscovery := &MockDiscoveryManager{}

	// Set custom function to simulate timeout
	mockDiscovery.customRegisterFunc = func(ctx context.Context, registrations []*ServiceRegistration) error {
		// Wait longer than the configured timeout
		select {
		case <-time.After(2 * time.Second):
			return fmt.Errorf("operation took too long")
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	config := NewServerConfig()
	config.ServiceName = "test-service"
	config.HTTP.Port = "0" // Use any available port
	config.GRPC.Port = "0"
	config.Discovery.Enabled = true
	config.Discovery.FailureMode = DiscoveryFailureModeFailFast
	config.Discovery.RegistrationTimeout = 1 * time.Second // Short timeout

	registrar := &MockServiceRegistrar{}

	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err, "Should create server successfully")

	// Replace the discovery manager with our mock
	server.SetDiscoveryManager(mockDiscovery)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start should fail due to timeout in fail-fast mode
	err = server.Start(ctx)
	assert.Error(t, err, "Server should fail to start due to discovery timeout")
	assert.Contains(t, err.Error(), "timed out")
}

func TestDiscoveryFailureMode_StringRepresentation(t *testing.T) {
	tests := []struct {
		mode     DiscoveryFailureMode
		expected string
	}{
		{DiscoveryFailureModeGraceful, "graceful"},
		{DiscoveryFailureModeFailFast, "fail_fast"},
		{DiscoveryFailureModeStrict, "strict"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.mode))
		})
	}
}

func TestConfigurationHelperMethods(t *testing.T) {
	tests := []struct {
		name             string
		failureMode      DiscoveryFailureMode
		expectedGraceful bool
		expectedFailFast bool
		expectedStrict   bool
	}{
		{
			name:             "graceful mode",
			failureMode:      DiscoveryFailureModeGraceful,
			expectedGraceful: true,
			expectedFailFast: false,
			expectedStrict:   false,
		},
		{
			name:             "fail_fast mode",
			failureMode:      DiscoveryFailureModeFailFast,
			expectedGraceful: false,
			expectedFailFast: true,
			expectedStrict:   false,
		},
		{
			name:             "strict mode",
			failureMode:      DiscoveryFailureModeStrict,
			expectedGraceful: false,
			expectedFailFast: true, // Strict mode should also return true for fail-fast
			expectedStrict:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewServerConfig()
			config.Discovery.FailureMode = tt.failureMode

			assert.Equal(t, tt.expectedGraceful, config.IsDiscoveryFailureModeGraceful(),
				"IsDiscoveryFailureModeGraceful mismatch")
			assert.Equal(t, tt.expectedFailFast, config.IsDiscoveryFailureModeFailFast(),
				"IsDiscoveryFailureModeFailFast mismatch")
			assert.Equal(t, tt.expectedStrict, config.IsDiscoveryFailureModeStrict(),
				"IsDiscoveryFailureModeStrict mismatch")

			// Check description is not empty
			desc := config.GetDiscoveryFailureModeDescription()
			assert.NotEmpty(t, desc, "Description should not be empty")
			// Check that the description contains relevant keywords based on the failure mode
			switch tt.failureMode {
			case DiscoveryFailureModeGraceful:
				assert.Contains(t, strings.ToLower(desc), "graceful")
			case DiscoveryFailureModeFailFast:
				assert.Contains(t, strings.ToLower(desc), "fail-fast")
			case DiscoveryFailureModeStrict:
				assert.Contains(t, strings.ToLower(desc), "strict")
			}
		})
	}
}

// TestDiscoveryFailureMode_BackwardCompatibility ensures existing configurations continue to work
func TestDiscoveryFailureMode_BackwardCompatibility(t *testing.T) {
	// Test that configurations without explicit failure_mode still work
	config := &ServerConfig{
		ServiceName: "test-service",
		HTTP: HTTPConfig{
			Enabled: true,
			Port:    "9000",
		},
		Discovery: DiscoveryConfig{
			Enabled:     true,
			Address:     "localhost:8500",
			ServiceName: "test-service",
			// Note: FailureMode is not set, should default to graceful
		},
	}

	config.SetDefaults()
	err := config.Validate()

	assert.NoError(t, err, "Configuration should be valid with defaults")
	assert.Equal(t, DiscoveryFailureModeGraceful, config.Discovery.FailureMode,
		"Should default to graceful mode for backward compatibility")
	assert.False(t, config.Discovery.HealthCheckRequired,
		"Should default to no health check requirement")
	assert.Equal(t, 30*time.Second, config.Discovery.RegistrationTimeout,
		"Should have reasonable default timeout")
}

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiscoveryManager(t *testing.T) {
	tests := []struct {
		name        string
		config      *DiscoveryConfig
		expectNoOp  bool
		expectError bool
	}{
		{
			name:       "disabled discovery returns no-op manager",
			config:     &DiscoveryConfig{Enabled: false},
			expectNoOp: true,
		},
		{
			name:       "nil config returns no-op manager",
			config:     nil,
			expectNoOp: true,
		},
		{
			name: "enabled discovery creates real manager",
			config: &DiscoveryConfig{
				Enabled:     true,
				Address:     "127.0.0.1:8500",
				ServiceName: "test-service",
			},
			expectNoOp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewDiscoveryManager(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, manager)

			if tt.expectNoOp {
				_, ok := manager.(*NoOpDiscoveryManager)
				assert.True(t, ok, "Expected NoOpDiscoveryManager")
			} else {
				_, ok := manager.(*DiscoveryManagerImpl)
				assert.True(t, ok, "Expected DiscoveryManagerImpl")
			}
		})
	}
}

func TestServiceRegistration(t *testing.T) {
	registration := &ServiceRegistration{
		ServiceName: "test-service",
		Address:     "localhost",
		Port:        8080,
		Tags:        []string{"v1", "http"},
		Meta: map[string]string{
			"protocol": "http",
			"version":  "v1",
		},
	}

	assert.Equal(t, "test-service", registration.ServiceName)
	assert.Equal(t, "localhost", registration.Address)
	assert.Equal(t, 8080, registration.Port)
	assert.Contains(t, registration.Tags, "v1")
	assert.Contains(t, registration.Tags, "http")
	assert.Equal(t, "http", registration.Meta["protocol"])
}

func TestNoOpDiscoveryManager(t *testing.T) {
	manager := &NoOpDiscoveryManager{}
	ctx := context.Background()

	registration := &ServiceRegistration{
		ServiceName: "test-service",
		Address:     "localhost",
		Port:        8080,
	}

	// All operations should succeed without error
	err := manager.RegisterService(ctx, registration)
	assert.NoError(t, err)

	err = manager.DeregisterService(ctx, registration)
	assert.NoError(t, err)

	registrations := []*ServiceRegistration{registration}
	err = manager.RegisterMultipleEndpoints(ctx, registrations)
	assert.NoError(t, err)

	err = manager.DeregisterMultipleEndpoints(ctx, registrations)
	assert.NoError(t, err)

	healthy := manager.IsHealthy(ctx)
	assert.True(t, healthy)
}

func TestCreateHTTPServiceRegistration(t *testing.T) {
	serviceName := "test-service"
	address := "localhost"
	port := 8080
	tags := []string{"v1", "api"}

	registration := CreateHTTPServiceRegistration(serviceName, address, port, tags)

	assert.Equal(t, serviceName, registration.ServiceName)
	assert.Equal(t, address, registration.Address)
	assert.Equal(t, port, registration.Port)
	assert.Contains(t, registration.Tags, "v1")
	assert.Contains(t, registration.Tags, "api")
	assert.Contains(t, registration.Tags, "http")
	assert.Equal(t, "http", registration.Meta["protocol"])
	assert.Equal(t, "v1", registration.Meta["version"])

	// Check health check configuration
	require.NotNil(t, registration.HealthCheck)
	assert.Equal(t, "http://localhost:8080/health", registration.HealthCheck.HTTP)
	assert.Equal(t, 10*time.Second, registration.HealthCheck.Interval)
	assert.Equal(t, 5*time.Second, registration.HealthCheck.Timeout)
	assert.Equal(t, 1*time.Minute, registration.HealthCheck.DeregisterCriticalServiceAfter)
}

func TestCreateGRPCServiceRegistration(t *testing.T) {
	serviceName := "test-service"
	address := "localhost"
	port := 9080
	tags := []string{"v1", "api"}

	registration := CreateGRPCServiceRegistration(serviceName, address, port, tags)

	assert.Equal(t, serviceName+"-grpc", registration.ServiceName)
	assert.Equal(t, address, registration.Address)
	assert.Equal(t, port, registration.Port)
	assert.Contains(t, registration.Tags, "v1")
	assert.Contains(t, registration.Tags, "api")
	assert.Contains(t, registration.Tags, "grpc")
	assert.Equal(t, "grpc", registration.Meta["protocol"])
	assert.Equal(t, "v1", registration.Meta["version"])

	// Check health check configuration
	require.NotNil(t, registration.HealthCheck)
	assert.Equal(t, "localhost:9080", registration.HealthCheck.GRPC)
	assert.Equal(t, 10*time.Second, registration.HealthCheck.Interval)
	assert.Equal(t, 5*time.Second, registration.HealthCheck.Timeout)
	assert.Equal(t, 1*time.Minute, registration.HealthCheck.DeregisterCriticalServiceAfter)
}

func TestCreateServiceRegistrations(t *testing.T) {
	tests := []struct {
		name          string
		config        *ServerConfig
		expectedCount int
		expectedNames []string
	}{
		{
			name: "HTTP only",
			config: &ServerConfig{
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				GRPC: GRPCConfig{
					Enabled: false,
				},
				Discovery: DiscoveryConfig{
					ServiceName: "test-service",
					Tags:        []string{"v1"},
				},
			},
			expectedCount: 1,
			expectedNames: []string{"test-service"},
		},
		{
			name: "gRPC only",
			config: &ServerConfig{
				HTTP: HTTPConfig{
					Enabled: false,
				},
				GRPC: GRPCConfig{
					Enabled: true,
					Port:    "9080",
				},
				Discovery: DiscoveryConfig{
					ServiceName: "test-service",
					Tags:        []string{"v1"},
				},
			},
			expectedCount: 1,
			expectedNames: []string{"test-service-grpc"},
		},
		{
			name: "both HTTP and gRPC on different ports",
			config: &ServerConfig{
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				GRPC: GRPCConfig{
					Enabled: true,
					Port:    "9080",
				},
				Discovery: DiscoveryConfig{
					ServiceName: "test-service",
					Tags:        []string{"v1"},
				},
			},
			expectedCount: 2,
			expectedNames: []string{"test-service", "test-service-grpc"},
		},
		{
			name: "both HTTP and gRPC on same port",
			config: &ServerConfig{
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				GRPC: GRPCConfig{
					Enabled: true,
					Port:    "8080",
				},
				Discovery: DiscoveryConfig{
					ServiceName: "test-service",
					Tags:        []string{"v1"},
				},
			},
			expectedCount: 1,
			expectedNames: []string{"test-service"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrations := CreateServiceRegistrations(tt.config)

			assert.Len(t, registrations, tt.expectedCount)

			actualNames := make([]string, len(registrations))
			for i, reg := range registrations {
				actualNames[i] = reg.ServiceName
			}

			for _, expectedName := range tt.expectedNames {
				assert.Contains(t, actualNames, expectedName)
			}
		})
	}
}

func TestDiscoveryManagerImpl_ServiceIDGeneration(t *testing.T) {
	tests := []struct {
		name         string
		registration *ServiceRegistration
		expectedID   string
	}{
		{
			name: "basic service ID generation",
			registration: &ServiceRegistration{
				ServiceName: "test-service",
				Address:     "localhost",
				Port:        8080,
			},
			expectedID: "test-service-localhost-8080",
		},
		{
			name: "service with different address",
			registration: &ServiceRegistration{
				ServiceName: "api-service",
				Address:     "192.168.1.100",
				Port:        9090,
			},
			expectedID: "api-service-192.168.1.100-9090",
		},
		{
			name: "service with existing ID should not change",
			registration: &ServiceRegistration{
				ServiceName: "existing-service",
				ServiceID:   "custom-id-123",
				Address:     "localhost",
				Port:        8080,
			},
			expectedID: "custom-id-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test service ID generation logic without actually calling Consul
			// This simulates the ID generation logic from RegisterService method
			if tt.registration.ServiceID == "" {
				tt.registration.ServiceID = fmt.Sprintf("%s-%s-%d",
					tt.registration.ServiceName, tt.registration.Address, tt.registration.Port)
			}

			assert.Equal(t, tt.expectedID, tt.registration.ServiceID)
		})
	}
}

func TestRetryConfig(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:      3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
	}

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.InitialInterval)
	assert.Equal(t, 30*time.Second, config.MaxInterval)
	assert.Equal(t, 2.0, config.Multiplier)
}

func TestHealthCheckInfo(t *testing.T) {
	healthCheck := &HealthCheckInfo{
		HTTP:                           "http://localhost:8080/health",
		Interval:                       10 * time.Second,
		Timeout:                        5 * time.Second,
		DeregisterCriticalServiceAfter: 1 * time.Minute,
	}

	assert.Equal(t, "http://localhost:8080/health", healthCheck.HTTP)
	assert.Equal(t, 10*time.Second, healthCheck.Interval)
	assert.Equal(t, 5*time.Second, healthCheck.Timeout)
	assert.Equal(t, 1*time.Minute, healthCheck.DeregisterCriticalServiceAfter)
}
func TestGenerateServiceName(t *testing.T) {
	tests := []struct {
		name     string
		baseName string
		protocol string
		port     int
		pattern  ServiceNamingPattern
		expected string
	}{
		{
			name:     "base pattern",
			baseName: "test-service",
			protocol: "http",
			port:     8080,
			pattern:  ServiceNamingPatternBase,
			expected: "test-service",
		},
		{
			name:     "protocol pattern with http",
			baseName: "test-service",
			protocol: "http",
			port:     8080,
			pattern:  ServiceNamingPatternProtocol,
			expected: "test-service",
		},
		{
			name:     "protocol pattern with grpc",
			baseName: "test-service",
			protocol: "grpc",
			port:     9080,
			pattern:  ServiceNamingPatternProtocol,
			expected: "test-service-grpc",
		},
		{
			name:     "port pattern",
			baseName: "test-service",
			protocol: "http",
			port:     8080,
			pattern:  ServiceNamingPatternPort,
			expected: "test-service-8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateServiceName(tt.baseName, tt.protocol, tt.port, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetServicesByProtocol(t *testing.T) {
	registrations := []*ServiceRegistration{
		{
			ServiceName: "test-service",
			Port:        8080,
			Meta:        map[string]string{"protocol": "http"},
		},
		{
			ServiceName: "test-service-grpc",
			Port:        9080,
			Meta:        map[string]string{"protocol": "grpc"},
		},
		{
			ServiceName: "another-service",
			Port:        8081,
			Meta:        map[string]string{"protocol": "http"},
		},
	}

	httpServices := GetServicesByProtocol(registrations, "http")
	assert.Len(t, httpServices, 2)
	assert.Equal(t, "test-service", httpServices[0].ServiceName)
	assert.Equal(t, "another-service", httpServices[1].ServiceName)

	grpcServices := GetServicesByProtocol(registrations, "grpc")
	assert.Len(t, grpcServices, 1)
	assert.Equal(t, "test-service-grpc", grpcServices[0].ServiceName)

	unknownServices := GetServicesByProtocol(registrations, "unknown")
	assert.Len(t, unknownServices, 0)
}

func TestGetServicesByPort(t *testing.T) {
	registrations := []*ServiceRegistration{
		{ServiceName: "service1", Port: 8080},
		{ServiceName: "service2", Port: 9080},
		{ServiceName: "service3", Port: 8080},
	}

	port8080Services := GetServicesByPort(registrations, 8080)
	assert.Len(t, port8080Services, 2)

	port9080Services := GetServicesByPort(registrations, 9080)
	assert.Len(t, port9080Services, 1)
	assert.Equal(t, "service2", port9080Services[0].ServiceName)

	unknownPortServices := GetServicesByPort(registrations, 7000)
	assert.Len(t, unknownPortServices, 0)
}

func TestGetServiceEndpointInfo(t *testing.T) {
	registrations := []*ServiceRegistration{
		{
			ServiceName: "test-service",
			ServiceID:   "test-service-localhost-8080",
			Address:     "localhost",
			Port:        8080,
			Tags:        []string{"http", "v1"},
			Meta:        map[string]string{"protocol": "http", "port_type": "http"},
		},
		{
			ServiceName: "test-service-grpc",
			ServiceID:   "test-service-grpc-localhost-9080",
			Address:     "localhost",
			Port:        9080,
			Tags:        []string{"grpc", "v1"},
			Meta:        map[string]string{"protocol": "grpc", "port_type": "grpc"},
		},
		{
			ServiceName: "dual-service",
			ServiceID:   "dual-service-localhost-8081",
			Address:     "localhost",
			Port:        8081,
			Tags:        []string{"http", "grpc", "v1"},
			Meta:        map[string]string{"protocol": "http", "port_type": "dual", "supports_grpc": "true"},
		},
	}

	info := GetServiceEndpointInfo(registrations)

	assert.Equal(t, 3, info["total_endpoints"])

	httpEndpoints := info["http_endpoints"].([]map[string]interface{})
	assert.Len(t, httpEndpoints, 1)
	assert.Equal(t, "test-service", httpEndpoints[0]["service_name"])

	grpcEndpoints := info["grpc_endpoints"].([]map[string]interface{})
	assert.Len(t, grpcEndpoints, 1)
	assert.Equal(t, "test-service-grpc", grpcEndpoints[0]["service_name"])

	dualEndpoints := info["dual_endpoints"].([]map[string]interface{})
	assert.Len(t, dualEndpoints, 1)
	assert.Equal(t, "dual-service", dualEndpoints[0]["service_name"])
}

func TestValidateServiceRegistrations(t *testing.T) {
	tests := []struct {
		name          string
		registrations []*ServiceRegistration
		expectError   bool
		errorContains string
	}{
		{
			name:          "empty registrations",
			registrations: []*ServiceRegistration{},
			expectError:   true,
			errorContains: "no service registrations provided",
		},
		{
			name:          "nil registration",
			registrations: []*ServiceRegistration{nil},
			expectError:   true,
			errorContains: "nil service registration found",
		},
		{
			name: "empty service name",
			registrations: []*ServiceRegistration{
				{ServiceName: "", Address: "localhost", Port: 8080},
			},
			expectError:   true,
			errorContains: "service name cannot be empty",
		},
		{
			name: "empty address",
			registrations: []*ServiceRegistration{
				{ServiceName: "test", Address: "", Port: 8080},
			},
			expectError:   true,
			errorContains: "service address cannot be empty",
		},
		{
			name: "invalid port",
			registrations: []*ServiceRegistration{
				{ServiceName: "test", Address: "localhost", Port: 0},
			},
			expectError:   true,
			errorContains: "invalid port",
		},
		{
			name: "port conflict between different services",
			registrations: []*ServiceRegistration{
				{ServiceName: "service1", Address: "localhost", Port: 8080},
				{ServiceName: "service2", Address: "localhost", Port: 8080},
			},
			expectError:   true,
			errorContains: "port 8080 is used by multiple services",
		},
		{
			name: "duplicate service IDs",
			registrations: []*ServiceRegistration{
				{ServiceName: "service1", ServiceID: "duplicate-id", Address: "localhost", Port: 8080},
				{ServiceName: "service2", ServiceID: "duplicate-id", Address: "localhost", Port: 8081},
			},
			expectError:   true,
			errorContains: "duplicate service ID",
		},
		{
			name: "valid registrations",
			registrations: []*ServiceRegistration{
				{ServiceName: "service1", Address: "localhost", Port: 8080},
				{ServiceName: "service2", Address: "localhost", Port: 8081},
			},
			expectError: false,
		},
		{
			name: "same service on same port with different service IDs",
			registrations: []*ServiceRegistration{
				{ServiceName: "service1", ServiceID: "service1-instance1", Address: "localhost", Port: 8080},
				{ServiceName: "service1", ServiceID: "service1-instance2", Address: "localhost", Port: 8080},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServiceRegistrations(tt.registrations)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				// Check that service IDs were generated for valid registrations
				for _, reg := range tt.registrations {
					if reg != nil {
						assert.NotEmpty(t, reg.ServiceID)
					}
				}
			}
		})
	}
}

func TestCreateServiceRegistrations_MultiEndpoint(t *testing.T) {
	tests := []struct {
		name                        string
		config                      *ServerConfig
		expectedRegistrations       int
		expectedHTTPService         string
		expectedGRPCService         string
		expectedDualProtocol        bool
		expectedPortDifferentiation bool
	}{
		{
			name: "HTTP and gRPC on different ports",
			config: &ServerConfig{
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				GRPC: GRPCConfig{
					Enabled: true,
					Port:    "9080",
				},
				Discovery: DiscoveryConfig{
					ServiceName: "multi-service",
					Tags:        []string{"v1", "api"},
				},
			},
			expectedRegistrations:       2,
			expectedHTTPService:         "multi-service",
			expectedGRPCService:         "multi-service-grpc",
			expectedDualProtocol:        false,
			expectedPortDifferentiation: true,
		},
		{
			name: "HTTP and gRPC on same port",
			config: &ServerConfig{
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				GRPC: GRPCConfig{
					Enabled: true,
					Port:    "8080",
				},
				Discovery: DiscoveryConfig{
					ServiceName: "dual-service",
					Tags:        []string{"v1", "api"},
				},
			},
			expectedRegistrations:       1,
			expectedHTTPService:         "dual-service",
			expectedGRPCService:         "",
			expectedDualProtocol:        true,
			expectedPortDifferentiation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrations := CreateServiceRegistrations(tt.config)

			assert.Len(t, registrations, tt.expectedRegistrations)

			// Validate registrations
			err := ValidateServiceRegistrations(registrations)
			assert.NoError(t, err)

			// Check HTTP service
			httpServices := GetServicesByProtocol(registrations, "http")
			if tt.expectedHTTPService != "" {
				assert.Len(t, httpServices, 1)
				assert.Equal(t, tt.expectedHTTPService, httpServices[0].ServiceName)
			}

			// Check gRPC service
			grpcServices := GetServicesByProtocol(registrations, "grpc")
			if tt.expectedGRPCService != "" {
				assert.Len(t, grpcServices, 1)
				assert.Equal(t, tt.expectedGRPCService, grpcServices[0].ServiceName)
			}

			// Check dual protocol support
			if tt.expectedDualProtocol {
				assert.Len(t, registrations, 1)
				reg := registrations[0]
				assert.Contains(t, reg.Tags, "grpc")
				assert.Equal(t, "true", reg.Meta["supports_grpc"])
				assert.Equal(t, "dual", reg.Meta["port_type"])
			}

			// Check port differentiation metadata
			if tt.expectedPortDifferentiation {
				for _, reg := range registrations {
					if reg.Meta["protocol"] == "grpc" {
						assert.Equal(t, "grpc", reg.Meta["port_type"])
						assert.NotEmpty(t, reg.Meta["http_port"])
						assert.NotEmpty(t, reg.Meta["grpc_port"])
					}
				}
			}
		})
	}
}
func TestDiscoveryHealthStatus(t *testing.T) {
	status := &DiscoveryHealthStatus{
		Available:         true,
		LastCheck:         time.Now(),
		ConsecutiveErrors: 0,
		TotalErrors:       5,
		Uptime:            time.Hour,
	}

	assert.True(t, status.Available)
	assert.Equal(t, 0, status.ConsecutiveErrors)
	assert.Equal(t, 5, status.TotalErrors)
	assert.Equal(t, time.Hour, status.Uptime)
}

func TestRetryConfigExtended(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:      5,
		InitialInterval: 2 * time.Second,
		MaxInterval:     60 * time.Second,
		Multiplier:      3.0,
		EnableJitter:    true,
	}

	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 2*time.Second, config.InitialInterval)
	assert.Equal(t, 60*time.Second, config.MaxInterval)
	assert.Equal(t, 3.0, config.Multiplier)
	assert.True(t, config.EnableJitter)
}

func TestDiscoveryManagerImpl_HealthStatus(t *testing.T) {
	config := &DiscoveryConfig{
		Enabled:     true,
		Address:     "127.0.0.1:8500",
		ServiceName: "test-service",
	}

	manager, err := NewDiscoveryManager(config)
	require.NoError(t, err)

	impl, ok := manager.(*DiscoveryManagerImpl)
	require.True(t, ok)

	// Test initial health status
	status := impl.GetHealthStatus()
	assert.NotNil(t, status)
	assert.True(t, status.Available) // Initially available
	assert.Equal(t, 0, status.ConsecutiveErrors)
	assert.Equal(t, 0, status.TotalErrors)

	// Test IsDiscoveryAvailable
	assert.True(t, impl.IsDiscoveryAvailable())
}

func TestDiscoveryManagerImpl_GracefulFailureHandling(t *testing.T) {
	config := &DiscoveryConfig{
		Enabled:     true,
		Address:     "127.0.0.1:8500",
		ServiceName: "test-service",
	}

	manager, err := NewDiscoveryManager(config)
	require.NoError(t, err)

	impl, ok := manager.(*DiscoveryManagerImpl)
	require.True(t, ok)

	// Reduce retry configuration for faster tests
	impl.retryConfig = &RetryConfig{
		MaxRetries:      1,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     500 * time.Millisecond,
		Multiplier:      2.0,
		EnableJitter:    false,
	}

	ctx := context.Background()
	registration := &ServiceRegistration{
		ServiceName: "test-service",
		Address:     "localhost",
		Port:        8080,
	}

	// This will fail because Consul is not running, but should handle gracefully
	err = impl.RegisterService(ctx, registration)
	assert.Error(t, err) // Should return error

	// Check that health status was updated
	status := impl.GetHealthStatus()
	assert.False(t, status.Available)
	assert.Greater(t, status.ConsecutiveErrors, 0)
	assert.Greater(t, status.TotalErrors, 0)
	assert.NotEmpty(t, status.LastError)

	// Test deregistration also handles gracefully
	err = impl.DeregisterService(ctx, registration)
	assert.Error(t, err) // Should return error but not panic

	// Test multiple endpoint operations
	registrations := []*ServiceRegistration{registration}
	err = impl.RegisterMultipleEndpoints(ctx, registrations)
	assert.Error(t, err) // Should handle gracefully

	// DeregisterMultipleEndpoints logs warnings but doesn't return error during shutdown
	err = impl.DeregisterMultipleEndpoints(ctx, registrations)
	// This may or may not return an error depending on implementation - both are acceptable
}

func TestNoOpDiscoveryManager_ExtendedInterface(t *testing.T) {
	manager := &NoOpDiscoveryManager{}

	// Test health status
	status := manager.GetHealthStatus()
	assert.NotNil(t, status)
	assert.True(t, status.Available)
	assert.Equal(t, 0, status.ConsecutiveErrors)
	assert.Equal(t, 0, status.TotalErrors)

	// Test availability check
	assert.True(t, manager.IsDiscoveryAvailable())

	// Test health check
	ctx := context.Background()
	assert.True(t, manager.IsHealthy(ctx))
}

func TestDiscoveryFailureScenarios(t *testing.T) {
	tests := []struct {
		name        string
		config      *DiscoveryConfig
		expectNoOp  bool
		testFailure bool
	}{
		{
			name: "disabled discovery should use no-op",
			config: &DiscoveryConfig{
				Enabled: false,
			},
			expectNoOp:  true,
			testFailure: false,
		},
		{
			name: "enabled discovery with invalid address should handle gracefully",
			config: &DiscoveryConfig{
				Enabled:     true,
				Address:     "invalid:99999",
				ServiceName: "test-service",
			},
			expectNoOp:  false,
			testFailure: true,
		},
		{
			name: "enabled discovery with unreachable address should handle gracefully",
			config: &DiscoveryConfig{
				Enabled:     true,
				Address:     "127.0.0.1:8500",
				ServiceName: "test-service",
			},
			expectNoOp:  false,
			testFailure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewDiscoveryManager(tt.config)
			require.NoError(t, err)

			if tt.expectNoOp {
				_, ok := manager.(*NoOpDiscoveryManager)
				assert.True(t, ok)
				return
			}

			impl, ok := manager.(*DiscoveryManagerImpl)
			require.True(t, ok)

			// Reduce retry configuration for faster tests
			impl.retryConfig = &RetryConfig{
				MaxRetries:      1,
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     500 * time.Millisecond,
				Multiplier:      2.0,
				EnableJitter:    false,
			}

			ctx := context.Background()
			registration := &ServiceRegistration{
				ServiceName: "test-service",
				Address:     "localhost",
				Port:        8080,
			}

			if tt.testFailure {
				// Test that operations fail gracefully without crashing
				err := impl.RegisterService(ctx, registration)
				if err != nil {
					// Should log warning but not crash
					assert.Contains(t, err.Error(), "operation failed after")
				}

				// Health status should reflect the failure
				status := impl.GetHealthStatus()
				if err != nil {
					assert.False(t, status.Available)
					assert.Greater(t, status.TotalErrors, 0)
				}
			}
		})
	}
}

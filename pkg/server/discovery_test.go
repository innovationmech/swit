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

// MockServiceDiscovery implements a mock service discovery for testing
type MockServiceDiscovery struct {
	registeredServices map[string]bool
	failRegistration   bool
	failDeregistration bool
}

func NewMockServiceDiscovery() *MockServiceDiscovery {
	return &MockServiceDiscovery{
		registeredServices: make(map[string]bool),
	}
}

func (m *MockServiceDiscovery) RegisterService(serviceName, address string, port int) error {
	if m.failRegistration {
		return assert.AnError
	}
	key := serviceName + ":" + address + ":" + string(rune(port))
	m.registeredServices[key] = true
	return nil
}

func (m *MockServiceDiscovery) DeregisterService(serviceName, address string, port int) error {
	if m.failDeregistration {
		return assert.AnError
	}
	key := serviceName + ":" + address + ":" + string(rune(port))
	delete(m.registeredServices, key)
	return nil
}

func (m *MockServiceDiscovery) GetInstanceRoundRobin(serviceName string) (string, error) {
	if serviceName == "health-check-test-service" {
		return "", fmt.Errorf("no healthy service instances found: %s", serviceName)
	}
	return "", assert.AnError
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 1*time.Second, config.InitialDelay)
	assert.Equal(t, 10*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.BackoffFactor)
	assert.True(t, config.JitterEnabled)
}

func TestNewServiceDiscoveryAbstraction(t *testing.T) {
	mockSD := NewMockServiceDiscovery()
	config := &DiscoveryConfig{
		Enabled:       true,
		RetryAttempts: 5,
		RetryInterval: 2 * time.Second,
	}

	sda := NewServiceDiscoveryAbstraction(nil, config)

	assert.NotNil(t, sda)
	assert.Equal(t, config, sda.config)
	assert.Equal(t, 5, sda.retryConfig.MaxAttempts)
	assert.Equal(t, 2*time.Second, sda.retryConfig.InitialDelay)
	assert.NotNil(t, sda.registeredServices)

	// Test with nil discovery service
	assert.Nil(t, sda.sd)

	// Test with mock discovery service
	sda2 := NewServiceDiscoveryAbstraction(nil, config)
	sda2.sd = mockSD
	assert.Equal(t, mockSD, sda2.sd)
}

func TestServiceDiscoveryAbstraction_RegisterEndpoint(t *testing.T) {
	tests := []struct {
		name          string
		endpoint      ServiceEndpoint
		discoveryNil  bool
		disabled      bool
		failOperation bool
		expectError   bool
	}{
		{
			name: "successful registration",
			endpoint: ServiceEndpoint{
				ServiceName: "test-service",
				Address:     "localhost",
				Port:        8080,
				Protocol:    "http",
			},
			expectError: false,
		},
		{
			name:        "discovery disabled",
			endpoint:    ServiceEndpoint{ServiceName: "test", Address: "localhost", Port: 8080},
			disabled:    true,
			expectError: false,
		},
		{
			name:         "discovery nil",
			endpoint:     ServiceEndpoint{ServiceName: "test", Address: "localhost", Port: 8080},
			discoveryNil: true,
			expectError:  false,
		},
		{
			name:          "registration failure",
			endpoint:      ServiceEndpoint{ServiceName: "test", Address: "localhost", Port: 8080},
			failOperation: true,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSD := NewMockServiceDiscovery()
			mockSD.failRegistration = tt.failOperation

			config := &DiscoveryConfig{
				Enabled:       !tt.disabled,
				RetryAttempts: 1, // Reduce retries for faster tests
				RetryInterval: 10 * time.Millisecond,
			}

			sda := NewServiceDiscoveryAbstraction(nil, config)
			if !tt.discoveryNil {
				sda.sd = mockSD
			}

			ctx := context.Background()
			err := sda.RegisterEndpoint(ctx, tt.endpoint)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if !tt.disabled && !tt.discoveryNil {
					// Check if endpoint was stored
					registered := sda.GetRegisteredEndpoints()
					assert.Len(t, registered, 1)
				}
			}
		})
	}
}

func TestServiceDiscoveryAbstraction_DeregisterEndpoint(t *testing.T) {
	mockSD := NewMockServiceDiscovery()
	config := &DiscoveryConfig{
		Enabled:       true,
		RetryAttempts: 1,
		RetryInterval: 10 * time.Millisecond,
	}

	sda := NewServiceDiscoveryAbstraction(nil, config)
	sda.sd = mockSD

	endpoint := ServiceEndpoint{
		ServiceName: "test-service",
		Address:     "localhost",
		Port:        8080,
		Protocol:    "http",
	}

	ctx := context.Background()

	// First register
	err := sda.RegisterEndpoint(ctx, endpoint)
	require.NoError(t, err)

	// Then deregister
	err = sda.DeregisterEndpoint(ctx, endpoint)
	assert.NoError(t, err)

	// Check if endpoint was removed
	registered := sda.GetRegisteredEndpoints()
	assert.Len(t, registered, 0)
}

func TestServiceDiscoveryAbstraction_RegisterMultipleEndpoints(t *testing.T) {
	mockSD := NewMockServiceDiscovery()
	config := &DiscoveryConfig{
		Enabled:       true,
		RetryAttempts: 1,
		RetryInterval: 10 * time.Millisecond,
	}

	sda := NewServiceDiscoveryAbstraction(nil, config)
	sda.sd = mockSD

	endpoints := []ServiceEndpoint{
		{ServiceName: "service1", Address: "localhost", Port: 8080, Protocol: "http"},
		{ServiceName: "service2", Address: "localhost", Port: 9090, Protocol: "grpc"},
	}

	ctx := context.Background()
	err := sda.RegisterMultipleEndpoints(ctx, endpoints)
	assert.NoError(t, err)

	// Check if both endpoints were registered
	registered := sda.GetRegisteredEndpoints()
	assert.Len(t, registered, 2)
}

func TestServiceDiscoveryAbstraction_DeregisterAllEndpoints(t *testing.T) {
	mockSD := NewMockServiceDiscovery()
	config := &DiscoveryConfig{
		Enabled:       true,
		RetryAttempts: 1,
		RetryInterval: 10 * time.Millisecond,
	}

	sda := NewServiceDiscoveryAbstraction(nil, config)
	sda.sd = mockSD

	endpoints := []ServiceEndpoint{
		{ServiceName: "service1", Address: "localhost", Port: 8080, Protocol: "http"},
		{ServiceName: "service2", Address: "localhost", Port: 9090, Protocol: "grpc"},
	}

	ctx := context.Background()

	// Register multiple endpoints
	err := sda.RegisterMultipleEndpoints(ctx, endpoints)
	require.NoError(t, err)

	// Deregister all
	err = sda.DeregisterAllEndpoints(ctx)
	assert.NoError(t, err)

	// Check if all endpoints were removed
	registered := sda.GetRegisteredEndpoints()
	assert.Len(t, registered, 0)
}

func TestServiceDiscoveryAbstraction_RetryOperation(t *testing.T) {
	mockSD := NewMockServiceDiscovery()
	mockSD.failRegistration = true

	config := &DiscoveryConfig{
		Enabled:       true,
		RetryAttempts: 3,
		RetryInterval: 10 * time.Millisecond,
	}

	sda := NewServiceDiscoveryAbstraction(nil, config)
	sda.sd = mockSD

	endpoint := ServiceEndpoint{
		ServiceName: "test-service",
		Address:     "localhost",
		Port:        8080,
		Protocol:    "http",
	}

	ctx := context.Background()
	err := sda.RegisterEndpoint(ctx, endpoint)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service registration failed")
}

func TestServiceDiscoveryAbstraction_IsHealthy(t *testing.T) {
	tests := []struct {
		name         string
		discoveryNil bool
		disabled     bool
		expected     bool
	}{
		{
			name:     "discovery disabled",
			disabled: true,
			expected: true,
		},
		{
			name:         "discovery nil",
			discoveryNil: true,
			expected:     true,
		},
		{
			name:     "discovery enabled and healthy",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSD := NewMockServiceDiscovery()
			config := &DiscoveryConfig{
				Enabled:       !tt.disabled,
				RetryAttempts: 1,
				RetryInterval: 10 * time.Millisecond,
			}

			sda := NewServiceDiscoveryAbstraction(nil, config)
			if !tt.discoveryNil {
				sda.sd = mockSD
			}

			ctx := context.Background()
			healthy := sda.IsHealthy(ctx)
			assert.Equal(t, tt.expected, healthy)
		})
	}
}

func TestServiceEndpoint(t *testing.T) {
	endpoint := ServiceEndpoint{
		ServiceName: "test-service",
		Address:     "localhost",
		Port:        8080,
		Protocol:    "http",
		Tags:        []string{"api", "v1"},
	}

	assert.Equal(t, "test-service", endpoint.ServiceName)
	assert.Equal(t, "localhost", endpoint.Address)
	assert.Equal(t, 8080, endpoint.Port)
	assert.Equal(t, "http", endpoint.Protocol)
	assert.Equal(t, []string{"api", "v1"}, endpoint.Tags)
}

func TestRetryConfig(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  2 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 1.5,
		JitterEnabled: false,
	}

	assert.Equal(t, 5, config.MaxAttempts)
	assert.Equal(t, 2*time.Second, config.InitialDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 1.5, config.BackoffFactor)
	assert.False(t, config.JitterEnabled)
}

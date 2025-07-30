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
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// ServiceDiscoveryInterface defines the interface for service discovery operations
type ServiceDiscoveryInterface interface {
	RegisterService(serviceName, address string, port int) error
	DeregisterService(serviceName, address string, port int) error
	GetInstanceRoundRobin(serviceName string) (string, error)
}

// ServiceDiscoveryAbstraction provides an abstraction layer for service discovery operations
type ServiceDiscoveryAbstraction struct {
	sd                 ServiceDiscoveryInterface
	config             *DiscoveryConfig
	retryConfig        *RetryConfig
	registeredServices map[string]ServiceEndpoint
}

// ServiceEndpoint represents a registered service endpoint
type ServiceEndpoint struct {
	ServiceName string
	Address     string
	Port        int
	Protocol    string // "http" or "grpc"
	Tags        []string
}

// RetryConfig defines retry behavior for service discovery operations
type RetryConfig struct {
	MaxAttempts   int           `json:"maxAttempts" yaml:"maxAttempts"`
	InitialDelay  time.Duration `json:"initialDelay" yaml:"initialDelay"`
	MaxDelay      time.Duration `json:"maxDelay" yaml:"maxDelay"`
	BackoffFactor float64       `json:"backoffFactor" yaml:"backoffFactor"`
	JitterEnabled bool          `json:"jitterEnabled" yaml:"jitterEnabled"`
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
		JitterEnabled: true,
	}
}

// NewServiceDiscoveryAbstraction creates a new service discovery abstraction
func NewServiceDiscoveryAbstraction(sd ServiceDiscoveryInterface, config *DiscoveryConfig) *ServiceDiscoveryAbstraction {
	retryConfig := DefaultRetryConfig()
	if config.RetryAttempts > 0 {
		retryConfig.MaxAttempts = config.RetryAttempts
	}
	if config.RetryInterval > 0 {
		retryConfig.InitialDelay = config.RetryInterval
	}

	return &ServiceDiscoveryAbstraction{
		sd:                 sd,
		config:             config,
		retryConfig:        retryConfig,
		registeredServices: make(map[string]ServiceEndpoint),
	}
}

// RegisterEndpoint registers a service endpoint with retry logic
func (sda *ServiceDiscoveryAbstraction) RegisterEndpoint(ctx context.Context, endpoint ServiceEndpoint) error {
	if sda.sd == nil || !sda.config.Enabled {
		return nil
	}

	// Generate service ID
	serviceID := fmt.Sprintf("%s-%s-%d", endpoint.ServiceName, endpoint.Address, endpoint.Port)

	// Attempt registration with retry
	err := sda.retryOperation(ctx, func() error {
		return sda.sd.RegisterService(endpoint.ServiceName, endpoint.Address, endpoint.Port)
	})

	if err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Failed to register service endpoint after retries",
				zap.String("serviceID", serviceID),
				zap.String("protocol", endpoint.Protocol),
				zap.Error(err))
		}
		return fmt.Errorf("service registration failed for %s: %w", serviceID, err)
	}

	// Store registered service for cleanup
	sda.registeredServices[serviceID] = endpoint

	if logger.Logger != nil {
		logger.Logger.Info("Successfully registered service endpoint",
			zap.String("serviceID", serviceID),
			zap.String("protocol", endpoint.Protocol),
			zap.String("address", fmt.Sprintf("%s:%d", endpoint.Address, endpoint.Port)))
	}

	return nil
}

// DeregisterEndpoint deregisters a service endpoint with retry logic
func (sda *ServiceDiscoveryAbstraction) DeregisterEndpoint(ctx context.Context, endpoint ServiceEndpoint) error {
	if sda.sd == nil || !sda.config.Enabled {
		return nil
	}

	serviceID := fmt.Sprintf("%s-%s-%d", endpoint.ServiceName, endpoint.Address, endpoint.Port)

	// Attempt deregistration with retry
	err := sda.retryOperation(ctx, func() error {
		return sda.sd.DeregisterService(endpoint.ServiceName, endpoint.Address, endpoint.Port)
	})

	if err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Failed to deregister service endpoint after retries",
				zap.String("serviceID", serviceID),
				zap.String("protocol", endpoint.Protocol),
				zap.Error(err))
		}
		return fmt.Errorf("service deregistration failed for %s: %w", serviceID, err)
	}

	// Remove from registered services
	delete(sda.registeredServices, serviceID)

	if logger.Logger != nil {
		logger.Logger.Info("Successfully deregistered service endpoint",
			zap.String("serviceID", serviceID),
			zap.String("protocol", endpoint.Protocol))
	}

	return nil
}

// RegisterMultipleEndpoints registers multiple service endpoints
func (sda *ServiceDiscoveryAbstraction) RegisterMultipleEndpoints(ctx context.Context, endpoints []ServiceEndpoint) error {
	var errors []error

	for _, endpoint := range endpoints {
		if err := sda.RegisterEndpoint(ctx, endpoint); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to register %d out of %d endpoints: %v", len(errors), len(endpoints), errors)
	}

	return nil
}

// DeregisterAllEndpoints deregisters all registered service endpoints
func (sda *ServiceDiscoveryAbstraction) DeregisterAllEndpoints(ctx context.Context) error {
	var errors []error

	for _, endpoint := range sda.registeredServices {
		if err := sda.DeregisterEndpoint(ctx, endpoint); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to deregister %d endpoints: %v", len(errors), errors)
	}

	return nil
}

// GetRegisteredEndpoints returns all currently registered endpoints
func (sda *ServiceDiscoveryAbstraction) GetRegisteredEndpoints() map[string]ServiceEndpoint {
	result := make(map[string]ServiceEndpoint)
	for k, v := range sda.registeredServices {
		result[k] = v
	}
	return result
}

// retryOperation executes an operation with exponential backoff retry logic
func (sda *ServiceDiscoveryAbstraction) retryOperation(ctx context.Context, operation func() error) error {
	var lastErr error
	delay := sda.retryConfig.InitialDelay

	for attempt := 1; attempt <= sda.retryConfig.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute operation
		lastErr = operation()
		if lastErr == nil {
			return nil // Success
		}

		// Log retry attempt
		if logger.Logger != nil {
			logger.Logger.Debug("Service discovery operation failed, retrying",
				zap.Int("attempt", attempt),
				zap.Int("maxAttempts", sda.retryConfig.MaxAttempts),
				zap.Duration("delay", delay),
				zap.Error(lastErr))
		}

		// Don't wait after the last attempt
		if attempt == sda.retryConfig.MaxAttempts {
			break
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * sda.retryConfig.BackoffFactor)
		if delay > sda.retryConfig.MaxDelay {
			delay = sda.retryConfig.MaxDelay
		}

		// Add jitter if enabled
		if sda.retryConfig.JitterEnabled {
			jitterFactor := float64(2*time.Now().UnixNano()%2 - 1)
			jitter := time.Duration(float64(delay) * 0.1 * jitterFactor)
			delay += jitter
			if delay < 0 {
				delay = sda.retryConfig.InitialDelay
			}
		}
	}

	return lastErr
}

// IsHealthy checks if the service discovery is healthy
func (sda *ServiceDiscoveryAbstraction) IsHealthy(ctx context.Context) bool {
	if sda.sd == nil || !sda.config.Enabled {
		return true // Consider disabled discovery as healthy
	}

	// Try a simple operation to check health
	err := sda.retryOperation(ctx, func() error {
		// Try to get a non-existent service to test connectivity
		_, err := sda.sd.GetInstanceRoundRobin("health-check-test-service")
		// We expect this to fail with "no healthy service instances found"
		// Any other error indicates connectivity issues
		if err != nil && err.Error() != "no healthy service instances found: health-check-test-service" {
			return err
		}
		return nil
	})

	return err == nil
}

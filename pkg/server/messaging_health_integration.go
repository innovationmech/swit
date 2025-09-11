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

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// MessagingHealthIntegrator integrates messaging health checks with the SWIT framework
type MessagingHealthIntegrator struct {
	coordinator interface{} // MessagingCoordinator interface from messaging package
	config      *MessagingHealthConfig
}

// MessagingHealthConfig contains configuration for messaging health integration
type MessagingHealthConfig struct {
	// Enabled controls whether messaging health checks are integrated
	Enabled bool
	// RegisterCoordinatorHealth controls whether to register coordinator health check
	RegisterCoordinatorHealth bool
	// RegisterBrokerHealth controls whether to register individual broker health checks
	RegisterBrokerHealth bool
	// RegisterSubsystemHealth controls whether to register subsystem health check
	RegisterSubsystemHealth bool
}

// DefaultMessagingHealthConfig returns default messaging health configuration
func DefaultMessagingHealthConfig() *MessagingHealthConfig {
	return &MessagingHealthConfig{
		Enabled:                   true,
		RegisterCoordinatorHealth: true,
		RegisterBrokerHealth:      true,
		RegisterSubsystemHealth:   true,
	}
}

// NewMessagingHealthIntegrator creates a new messaging health integrator
func NewMessagingHealthIntegrator(coordinator interface{}, config *MessagingHealthConfig) *MessagingHealthIntegrator {
	if config == nil {
		config = DefaultMessagingHealthConfig()
	}

	return &MessagingHealthIntegrator{
		coordinator: coordinator,
		config:      config,
	}
}

// IntegrateWithFramework integrates messaging health checks with the framework health system
func (m *MessagingHealthIntegrator) IntegrateWithFramework(registry BusinessServiceRegistry) error {
	if !m.config.Enabled {
		logger.Logger.Info("Messaging health integration is disabled")
		return nil
	}

	if m.coordinator == nil {
		logger.Logger.Warn("No messaging coordinator provided for health integration")
		return nil
	}

	logger.Logger.Info("Integrating messaging health checks with framework",
		zap.Bool("coordinator_health", m.config.RegisterCoordinatorHealth),
		zap.Bool("broker_health", m.config.RegisterBrokerHealth),
		zap.Bool("subsystem_health", m.config.RegisterSubsystemHealth))

	// Create messaging health checkers based on configuration
	if m.config.RegisterCoordinatorHealth {
		healthCheck := &MessagingCoordinatorHealthCheck{
			coordinator: m.coordinator,
			serviceName: "messaging-coordinator",
		}
		if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
			return fmt.Errorf("failed to register messaging coordinator health check: %w", err)
		}
		logger.Logger.Info("Registered messaging coordinator health check")
	}

	if m.config.RegisterSubsystemHealth {
		healthCheck := &MessagingSubsystemHealthCheck{
			coordinator: m.coordinator,
			serviceName: "messaging-subsystem",
		}
		if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
			return fmt.Errorf("failed to register messaging subsystem health check: %w", err)
		}
		logger.Logger.Info("Registered messaging subsystem health check")
	}

	return nil
}

// MessagingCoordinatorHealthCheck implements BusinessHealthCheck for messaging coordinator
type MessagingCoordinatorHealthCheck struct {
	coordinator interface{}
	serviceName string
}

// Check implements BusinessHealthCheck interface
func (h *MessagingCoordinatorHealthCheck) Check(ctx context.Context) error {
	// Use reflection or type assertion to call health check
	// This avoids circular imports by using interface{}
	if coordinator, ok := h.coordinator.(interface {
		HealthCheck(context.Context) (interface{}, error)
	}); ok {
		status, err := coordinator.HealthCheck(ctx)
		if err != nil {
			logger.Logger.Error("Messaging coordinator health check failed",
				zap.String("service", h.serviceName),
				zap.Error(err))
			return fmt.Errorf("messaging coordinator health check failed: %w", err)
		}

		// Check if status indicates healthy state
		if healthStatus, ok := status.(interface {
			GetOverall() string
		}); ok {
			if healthStatus.GetOverall() != "healthy" {
				return fmt.Errorf("messaging coordinator is unhealthy: %s", healthStatus.GetOverall())
			}
		}

		// For simpler status checking, try to extract Overall field via type assertion
		if statusMap, ok := status.(map[string]interface{}); ok {
			if overall, exists := statusMap["overall"]; exists {
				if overallStr, ok := overall.(string); ok && overallStr != "healthy" {
					return fmt.Errorf("messaging coordinator status: %s", overallStr)
				}
			}
		}

		logger.Logger.Debug("Messaging coordinator health check passed",
			zap.String("service", h.serviceName))
		return nil
	}

	return fmt.Errorf("messaging coordinator does not support health checks")
}

// GetServiceName implements BusinessHealthCheck interface
func (h *MessagingCoordinatorHealthCheck) GetServiceName() string {
	return h.serviceName
}

// MessagingSubsystemHealthCheck implements BusinessHealthCheck for the entire messaging subsystem
type MessagingSubsystemHealthCheck struct {
	coordinator interface{}
	serviceName string
}

// Check implements BusinessHealthCheck interface
func (h *MessagingSubsystemHealthCheck) Check(ctx context.Context) error {
	// First check coordinator health
	if coordinator, ok := h.coordinator.(interface {
		HealthCheck(context.Context) (interface{}, error)
	}); ok {
		status, err := coordinator.HealthCheck(ctx)
		if err != nil {
			logger.Logger.Error("Messaging subsystem health check failed at coordinator level",
				zap.String("service", h.serviceName),
				zap.Error(err))
			return fmt.Errorf("messaging subsystem coordinator check failed: %w", err)
		}

		// Check if coordinator is healthy
		if statusMap, ok := status.(map[string]interface{}); ok {
			if overall, exists := statusMap["overall"]; exists {
				if overallStr, ok := overall.(string); ok && overallStr != "healthy" {
					return fmt.Errorf("messaging subsystem unhealthy: %s", overallStr)
				}
			}

			// Check broker health if available
			if brokerHealth, exists := statusMap["broker_health"]; exists {
				if brokerMap, ok := brokerHealth.(map[string]interface{}); ok {
					for brokerName, health := range brokerMap {
						if healthStr, ok := health.(string); ok && healthStr != "healthy" {
							logger.Logger.Warn("Broker unhealthy",
								zap.String("broker", brokerName),
								zap.String("status", healthStr))
							// We don't fail the entire subsystem for individual broker issues
							// but we log them for monitoring
						}
					}
				}
			}
		}

		logger.Logger.Debug("Messaging subsystem health check passed",
			zap.String("service", h.serviceName))
		return nil
	}

	return fmt.Errorf("messaging coordinator does not support health checks")
}

// GetServiceName implements BusinessHealthCheck interface
func (h *MessagingSubsystemHealthCheck) GetServiceName() string {
	return h.serviceName
}

// GetMessagingHealthStatus retrieves detailed messaging health status from transport coordinator
func GetMessagingHealthStatus(ctx context.Context, transportCoordinator interface{}) (interface{}, error) {
	if coordinator, ok := transportCoordinator.(interface {
		CheckMessagingHealth(context.Context) (interface{}, error)
	}); ok {
		return coordinator.CheckMessagingHealth(ctx)
	}
	return nil, fmt.Errorf("transport coordinator does not support messaging health checks")
}

// IsMessagingHealthy checks if messaging subsystem is healthy
func IsMessagingHealthy(ctx context.Context, transportCoordinator interface{}) bool {
	status, err := GetMessagingHealthStatus(ctx, transportCoordinator)
	if err != nil {
		logger.Logger.Error("Failed to get messaging health status", zap.Error(err))
		return false
	}

	if statusMap, ok := status.(map[string]interface{}); ok {
		if overall, exists := statusMap["overall"]; exists {
			if overallStr, ok := overall.(string); ok {
				return overallStr == "healthy"
			}
		}
	}

	// If we can't determine status, assume unhealthy for safety
	return false
}

// CreateMessagingHealthIntegrationFromConfig creates messaging health integration from server config
func CreateMessagingHealthIntegrationFromConfig(config *ServerConfig, coordinator interface{}) *MessagingHealthIntegrator {
	if !config.IsMessagingEnabled() {
		return nil
	}

	healthConfig := &MessagingHealthConfig{
		Enabled:                   config.Messaging.Monitoring.HealthCheckEnabled,
		RegisterCoordinatorHealth: true,
		RegisterBrokerHealth:      true,
		RegisterSubsystemHealth:   true,
	}

	return NewMessagingHealthIntegrator(coordinator, healthConfig)
}

// RegisterMessagingHealthChecks is a convenience function to register all messaging health checks
func RegisterMessagingHealthChecks(registry BusinessServiceRegistry, coordinator interface{}, config *MessagingHealthConfig) error {
	integrator := NewMessagingHealthIntegrator(coordinator, config)
	return integrator.IntegrateWithFramework(registry)
}

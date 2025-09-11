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

package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// MessagingHealthChecker provides health check integration for messaging components
// with the existing SWIT framework health check system. It implements the
// BusinessHealthCheck interface to integrate seamlessly with the framework.
type MessagingHealthChecker interface {
	// Check performs comprehensive health check on messaging components
	Check(ctx context.Context) error
	// GetServiceName returns the health check service name
	GetServiceName() string
	// GetHealthCheckConfig returns health check configuration
	GetHealthCheckConfig() *HealthCheckConfig
}

// HealthCheckConfig contains configuration for messaging health checks
type HealthCheckConfig struct {
	// Enabled controls whether health checks are active
	Enabled bool `json:"enabled" yaml:"enabled"`
	// Interval specifies how often to perform health checks
	Interval time.Duration `json:"interval" yaml:"interval"`
	// Timeout specifies maximum time for a health check
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
	// BrokerChecksEnabled controls broker-specific health checks
	BrokerChecksEnabled bool `json:"broker_checks_enabled" yaml:"broker_checks_enabled"`
	// HandlerChecksEnabled controls handler-specific health checks
	HandlerChecksEnabled bool `json:"handler_checks_enabled" yaml:"handler_checks_enabled"`
	// FailureThreshold specifies consecutive failures before marking unhealthy
	FailureThreshold int `json:"failure_threshold" yaml:"failure_threshold"`
}

// DefaultHealthCheckConfig returns default health check configuration
func DefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Enabled:              true,
		Interval:             30 * time.Second,
		Timeout:              10 * time.Second,
		BrokerChecksEnabled:  true,
		HandlerChecksEnabled: true,
		FailureThreshold:     3,
	}
}

// MessagingCoordinatorHealthChecker implements health check integration for MessagingCoordinator
type MessagingCoordinatorHealthChecker struct {
	coordinator MessagingCoordinator
	config      *HealthCheckConfig
	serviceName string
}

// NewMessagingCoordinatorHealthChecker creates a new health checker for MessagingCoordinator
func NewMessagingCoordinatorHealthChecker(coordinator MessagingCoordinator, serviceName string, config *HealthCheckConfig) *MessagingCoordinatorHealthChecker {
	if config == nil {
		config = DefaultHealthCheckConfig()
	}
	if serviceName == "" {
		serviceName = "messaging-coordinator"
	}

	return &MessagingCoordinatorHealthChecker{
		coordinator: coordinator,
		config:      config,
		serviceName: serviceName,
	}
}

// Check implements the BusinessHealthCheck interface for framework integration
func (h *MessagingCoordinatorHealthChecker) Check(ctx context.Context) error {
	if !h.config.Enabled {
		return nil // Health checks disabled
	}

	// Create timeout context for health check
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	status, err := h.coordinator.HealthCheck(checkCtx)
	if err != nil {
		logger.Logger.Error("Messaging health check failed",
			zap.String("service", h.serviceName),
			zap.Error(err))
		return fmt.Errorf("messaging coordinator health check failed: %w", err)
	}

	// Check overall health status
	if status.Overall != "healthy" {
		logger.Logger.Warn("Messaging coordinator is unhealthy",
			zap.String("service", h.serviceName),
			zap.String("status", status.Overall),
			zap.String("details", status.Details))
		return fmt.Errorf("messaging coordinator status: %s - %s", status.Overall, status.Details)
	}

	logger.Logger.Debug("Messaging health check passed",
		zap.String("service", h.serviceName),
		zap.Int("broker_count", len(status.BrokerHealth)),
		zap.Int("handler_count", len(status.HandlerHealth)))

	return nil
}

// GetServiceName implements the BusinessHealthCheck interface
func (h *MessagingCoordinatorHealthChecker) GetServiceName() string {
	return h.serviceName
}

// GetHealthStatus returns detailed messaging health status
func (h *MessagingCoordinatorHealthChecker) GetHealthStatus(ctx context.Context) (*MessagingHealthStatus, error) {
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	return h.coordinator.HealthCheck(checkCtx)
}

// GetHealthCheckConfig returns the health check configuration
func (h *MessagingCoordinatorHealthChecker) GetHealthCheckConfig() *HealthCheckConfig {
	return h.config
}

// BrokerHealthChecker provides health check integration for individual message brokers
type BrokerHealthChecker struct {
	broker      MessageBroker
	config      *HealthCheckConfig
	serviceName string
}

// NewBrokerHealthChecker creates a new health checker for a message broker
func NewBrokerHealthChecker(broker MessageBroker, brokerName string, config *HealthCheckConfig) *BrokerHealthChecker {
	if config == nil {
		config = DefaultHealthCheckConfig()
	}
	serviceName := fmt.Sprintf("messaging-broker-%s", brokerName)

	return &BrokerHealthChecker{
		broker:      broker,
		config:      config,
		serviceName: serviceName,
	}
}

// Check implements the BusinessHealthCheck interface for broker health checks
func (h *BrokerHealthChecker) Check(ctx context.Context) error {
	if !h.config.Enabled || !h.config.BrokerChecksEnabled {
		return nil // Broker health checks disabled
	}

	// Create timeout context for health check
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	// Check broker connection status
	if !h.broker.IsConnected() {
		logger.Logger.Error("Broker is not connected",
			zap.String("service", h.serviceName))
		return fmt.Errorf("broker %s is not connected", h.serviceName)
	}

	// Perform broker health check
	status, err := h.broker.HealthCheck(checkCtx)
	if err != nil {
		logger.Logger.Error("Broker health check failed",
			zap.String("service", h.serviceName),
			zap.Error(err))
		return fmt.Errorf("broker %s health check failed: %w", h.serviceName, err)
	}

	// Check health status
	if status.Status != HealthStatusHealthy {
		logger.Logger.Warn("Broker is unhealthy",
			zap.String("service", h.serviceName),
			zap.String("status", string(status.Status)),
			zap.String("message", status.Message))
		return fmt.Errorf("broker %s status: %s - %s", h.serviceName, status.Status, status.Message)
	}

	logger.Logger.Debug("Broker health check passed",
		zap.String("service", h.serviceName),
		zap.Duration("response_time", status.ResponseTime))

	return nil
}

// GetServiceName implements the BusinessHealthCheck interface
func (h *BrokerHealthChecker) GetServiceName() string {
	return h.serviceName
}

// GetHealthStatus returns detailed broker health status
func (h *BrokerHealthChecker) GetHealthStatus(ctx context.Context) (*HealthStatus, error) {
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	return h.broker.HealthCheck(checkCtx)
}

// GetHealthCheckConfig returns the health check configuration
func (h *BrokerHealthChecker) GetHealthCheckConfig() *HealthCheckConfig {
	return h.config
}

// MessagingSubsystemHealthChecker provides aggregated health checking for the entire messaging subsystem
type MessagingSubsystemHealthChecker struct {
	coordinator     MessagingCoordinator
	brokerCheckers  map[string]*BrokerHealthChecker
	config          *HealthCheckConfig
	serviceName     string
	failureCounters map[string]int
}

// NewMessagingSubsystemHealthChecker creates a comprehensive health checker for the messaging subsystem
func NewMessagingSubsystemHealthChecker(coordinator MessagingCoordinator, config *HealthCheckConfig) *MessagingSubsystemHealthChecker {
	if config == nil {
		config = DefaultHealthCheckConfig()
	}

	return &MessagingSubsystemHealthChecker{
		coordinator:     coordinator,
		brokerCheckers:  make(map[string]*BrokerHealthChecker),
		config:          config,
		serviceName:     "messaging-subsystem",
		failureCounters: make(map[string]int),
	}
}

// RegisterBrokerChecker adds a broker health checker to the subsystem checker
func (h *MessagingSubsystemHealthChecker) RegisterBrokerChecker(brokerName string, checker *BrokerHealthChecker) {
	h.brokerCheckers[brokerName] = checker
	h.failureCounters[brokerName] = 0
}

// Check implements the BusinessHealthCheck interface for subsystem health checks
func (h *MessagingSubsystemHealthChecker) Check(ctx context.Context) error {
	if !h.config.Enabled {
		return nil // Health checks disabled
	}

	// Create timeout context for health check
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	var healthErrors []string

	// Check coordinator health
	coordinatorStatus, err := h.coordinator.HealthCheck(checkCtx)
	if err != nil {
		h.incrementFailureCounter("coordinator")
		healthErrors = append(healthErrors, fmt.Sprintf("coordinator: %v", err))
	} else if coordinatorStatus.Overall != "healthy" {
		h.incrementFailureCounter("coordinator")
		healthErrors = append(healthErrors, fmt.Sprintf("coordinator: %s - %s", coordinatorStatus.Overall, coordinatorStatus.Details))
	} else {
		h.resetFailureCounter("coordinator")
	}

	// Check individual broker health if enabled
	if h.config.BrokerChecksEnabled {
		for brokerName, checker := range h.brokerCheckers {
			if err := checker.Check(checkCtx); err != nil {
				h.incrementFailureCounter(brokerName)
				healthErrors = append(healthErrors, fmt.Sprintf("broker %s: %v", brokerName, err))
			} else {
				h.resetFailureCounter(brokerName)
			}
		}
	}

	// Check if any component exceeds failure threshold
	for component, count := range h.failureCounters {
		if count >= h.config.FailureThreshold {
			healthErrors = append(healthErrors, fmt.Sprintf("%s exceeded failure threshold (%d)", component, count))
		}
	}

	if len(healthErrors) > 0 {
		logger.Logger.Error("Messaging subsystem health check failed",
			zap.String("service", h.serviceName),
			zap.Strings("errors", healthErrors))
		return fmt.Errorf("messaging subsystem health issues: %v", healthErrors)
	}

	logger.Logger.Debug("Messaging subsystem health check passed",
		zap.String("service", h.serviceName),
		zap.Int("broker_checkers", len(h.brokerCheckers)))

	return nil
}

// GetServiceName implements the BusinessHealthCheck interface
func (h *MessagingSubsystemHealthChecker) GetServiceName() string {
	return h.serviceName
}

// GetHealthStatus returns detailed subsystem health status
func (h *MessagingSubsystemHealthChecker) GetHealthStatus(ctx context.Context) (*MessagingSubsystemHealthStatus, error) {
	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
	defer cancel()

	status := &MessagingSubsystemHealthStatus{
		Overall:           "healthy",
		Timestamp:         time.Now(),
		CoordinatorHealth: make(map[string]string),
		BrokerHealth:      make(map[string]string),
		FailureCounters:   make(map[string]int),
		SubsystemDetails:  make(map[string]interface{}),
	}

	// Get coordinator health
	coordinatorStatus, err := h.coordinator.HealthCheck(checkCtx)
	if err != nil {
		status.Overall = "unhealthy"
		status.CoordinatorHealth["error"] = err.Error()
	} else {
		status.CoordinatorHealth["status"] = coordinatorStatus.Overall
		status.CoordinatorHealth["details"] = coordinatorStatus.Details
		if coordinatorStatus.Overall != "healthy" {
			status.Overall = "degraded"
		}
	}

	// Get broker health
	for brokerName, checker := range h.brokerCheckers {
		brokerStatus, err := checker.GetHealthStatus(checkCtx)
		if err != nil {
			status.BrokerHealth[brokerName] = "error: " + err.Error()
			if status.Overall == "healthy" {
				status.Overall = "degraded"
			}
		} else {
			status.BrokerHealth[brokerName] = string(brokerStatus.Status)
			if brokerStatus.Status != HealthStatusHealthy && status.Overall == "healthy" {
				status.Overall = "degraded"
			}
		}
	}

	// Copy failure counters
	for component, count := range h.failureCounters {
		status.FailureCounters[component] = count
		if count >= h.config.FailureThreshold && status.Overall != "unhealthy" {
			status.Overall = "unhealthy"
		}
	}

	// Add subsystem details
	status.SubsystemDetails["broker_count"] = len(h.brokerCheckers)
	status.SubsystemDetails["config"] = h.config

	return status, nil
}

// GetHealthCheckConfig returns the health check configuration
func (h *MessagingSubsystemHealthChecker) GetHealthCheckConfig() *HealthCheckConfig {
	return h.config
}

// incrementFailureCounter increments the failure counter for a component
func (h *MessagingSubsystemHealthChecker) incrementFailureCounter(component string) {
	h.failureCounters[component]++
}

// resetFailureCounter resets the failure counter for a component
func (h *MessagingSubsystemHealthChecker) resetFailureCounter(component string) {
	h.failureCounters[component] = 0
}

// MessagingSubsystemHealthStatus represents the health status of the entire messaging subsystem
type MessagingSubsystemHealthStatus struct {
	// Overall health status of the messaging subsystem
	Overall string `json:"overall"`
	// Timestamp when the health check was performed
	Timestamp time.Time `json:"timestamp"`
	// CoordinatorHealth contains coordinator-specific health information
	CoordinatorHealth map[string]string `json:"coordinator_health"`
	// BrokerHealth contains broker-specific health information
	BrokerHealth map[string]string `json:"broker_health"`
	// FailureCounters tracks consecutive failures per component
	FailureCounters map[string]int `json:"failure_counters"`
	// SubsystemDetails contains additional subsystem information
	SubsystemDetails map[string]interface{} `json:"subsystem_details"`
}

// CreateMessagingHealthCheckers creates health checkers for a messaging coordinator and its brokers
func CreateMessagingHealthCheckers(coordinator MessagingCoordinator, config *HealthCheckConfig) (*MessagingCoordinatorHealthChecker, []*BrokerHealthChecker, *MessagingSubsystemHealthChecker, error) {
	if config == nil {
		config = DefaultHealthCheckConfig()
	}

	// Create coordinator health checker
	coordinatorChecker := NewMessagingCoordinatorHealthChecker(coordinator, "messaging-coordinator", config)

	// Create broker health checkers for each registered broker
	var brokerCheckers []*BrokerHealthChecker
	brokerNames := coordinator.GetRegisteredBrokers()
	for _, brokerName := range brokerNames {
		broker, err := coordinator.GetBroker(brokerName)
		if err != nil {
			logger.Logger.Warn("Failed to get broker for health checker",
				zap.String("broker", brokerName),
				zap.Error(err))
			continue
		}

		brokerChecker := NewBrokerHealthChecker(broker, brokerName, config)
		brokerCheckers = append(brokerCheckers, brokerChecker)
	}

	// Create subsystem health checker
	subsystemChecker := NewMessagingSubsystemHealthChecker(coordinator, config)
	for _, brokerChecker := range brokerCheckers {
		// Extract broker name from service name (remove "messaging-broker-" prefix)
		brokerName := brokerChecker.GetServiceName()[len("messaging-broker-"):]
		subsystemChecker.RegisterBrokerChecker(brokerName, brokerChecker)
	}

	return coordinatorChecker, brokerCheckers, subsystemChecker, nil
}

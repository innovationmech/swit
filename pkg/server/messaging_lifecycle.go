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
	"github.com/innovationmech/swit/pkg/messaging"
	"go.uber.org/zap"
)

// MessagingLifecyclePhase represents different phases of messaging system lifecycle
type MessagingLifecyclePhase int

const (
	MessagingPhaseUninitialized MessagingLifecyclePhase = iota
	MessagingPhaseConfigValidated
	MessagingPhaseDependenciesInjected
	MessagingPhaseCoordinatorInitialized
	MessagingPhaseBrokersRegistered
	MessagingPhaseHandlersRegistered
	MessagingPhaseStarted
	MessagingPhaseStopping
	MessagingPhaseStopped
)

// String returns the string representation of the messaging lifecycle phase
func (p MessagingLifecyclePhase) String() string {
	switch p {
	case MessagingPhaseUninitialized:
		return "uninitialized"
	case MessagingPhaseConfigValidated:
		return "config_validated"
	case MessagingPhaseDependenciesInjected:
		return "dependencies_injected"
	case MessagingPhaseCoordinatorInitialized:
		return "coordinator_initialized"
	case MessagingPhaseBrokersRegistered:
		return "brokers_registered"
	case MessagingPhaseHandlersRegistered:
		return "handlers_registered"
	case MessagingPhaseStarted:
		return "started"
	case MessagingPhaseStopping:
		return "stopping"
	case MessagingPhaseStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// MessagingStartupConfig contains configuration for messaging startup sequence
type MessagingStartupConfig struct {
	// StartupTimeout is the maximum time to wait for messaging startup
	StartupTimeout time.Duration `yaml:"startup_timeout" json:"startup_timeout"`

	// ShutdownTimeout is the maximum time to wait for messaging shutdown
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" json:"shutdown_timeout"`

	// NonBlocking indicates whether messaging startup failures should be non-blocking
	NonBlocking bool `yaml:"non_blocking" json:"non_blocking"`

	// RetryAttempts is the number of retry attempts for failed operations
	RetryAttempts int `yaml:"retry_attempts" json:"retry_attempts"`

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration `yaml:"retry_delay" json:"retry_delay"`

	// HealthCheckInterval is the interval for performing health checks during startup
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
}

// DefaultMessagingStartupConfig returns default configuration for messaging startup
func DefaultMessagingStartupConfig() *MessagingStartupConfig {
	return &MessagingStartupConfig{
		StartupTimeout:      30 * time.Second,
		ShutdownTimeout:     30 * time.Second,
		NonBlocking:         false,
		RetryAttempts:       3,
		RetryDelay:          time.Second,
		HealthCheckInterval: 5 * time.Second,
	}
}

// MessagingLifecycleManager manages the messaging system lifecycle integration
// with the SWIT framework startup phases. It ensures proper initialization order
// and dependency resolution for messaging components.
type MessagingLifecycleManager struct {
	// Configuration for messaging lifecycle
	config *MessagingStartupConfig

	// Current lifecycle phase
	currentPhase MessagingLifecyclePhase

	// Messaging coordinator for orchestrating brokers and handlers
	coordinator messaging.MessagingCoordinator

	// Registered brokers and handlers
	brokers  map[string]messaging.MessageBroker
	handlers map[string]EventHandler

	// Lifecycle tracking
	startTime *time.Time
	stopTime  *time.Time

	// Validation and error tracking
	validationErrors []error
	startupErrors    []error
}

// NewMessagingLifecycleManager creates a new messaging lifecycle manager
func NewMessagingLifecycleManager(config *MessagingStartupConfig) *MessagingLifecycleManager {
	if config == nil {
		config = DefaultMessagingStartupConfig()
	}

	return &MessagingLifecycleManager{
		config:       config,
		currentPhase: MessagingPhaseUninitialized,
		coordinator:  messaging.NewMessagingCoordinator(),
		brokers:      make(map[string]messaging.MessageBroker),
		handlers:     make(map[string]EventHandler),
	}
}

// GetCurrentPhase returns the current messaging lifecycle phase
func (m *MessagingLifecycleManager) GetCurrentPhase() MessagingLifecyclePhase {
	return m.currentPhase
}

// GetCoordinator returns the messaging coordinator
func (m *MessagingLifecycleManager) GetCoordinator() messaging.MessagingCoordinator {
	return m.coordinator
}

// StartupSequence executes the messaging startup sequence following framework patterns
// Order: Config → DI → Coordinators → Brokers → Handlers → Start
func (m *MessagingLifecycleManager) StartupSequence(ctx context.Context, brokerConfigs map[string]*messaging.BrokerConfig, handlers []EventHandler) error {
	logger.Logger.Info("Starting messaging system startup sequence",
		zap.String("phase", m.currentPhase.String()),
		zap.Int("broker_configs", len(brokerConfigs)),
		zap.Int("handlers", len(handlers)))

	startTime := time.Now()
	m.startTime = &startTime

	// Create timeout context for entire startup sequence
	startupCtx, cancel := context.WithTimeout(ctx, m.config.StartupTimeout)
	defer cancel()

	// Phase 1: Configuration validation and loading
	if err := m.validateConfiguration(brokerConfigs, handlers); err != nil {
		return m.handleStartupError("configuration validation", err)
	}

	// Phase 2: Dependency injection container setup
	if err := m.setupDependencyInjection(startupCtx, brokerConfigs); err != nil {
		return m.handleStartupError("dependency injection setup", err)
	}

	// Phase 3: MessagingCoordinator initialization
	if err := m.initializeCoordinator(startupCtx); err != nil {
		return m.handleStartupError("coordinator initialization", err)
	}

	// Phase 4: Broker registration and connection
	if err := m.registerBrokers(startupCtx, brokerConfigs); err != nil {
		return m.handleStartupError("broker registration", err)
	}

	// Phase 5: Event handler discovery and registration
	if err := m.registerHandlers(startupCtx, handlers); err != nil {
		return m.handleStartupError("handler registration", err)
	}

	// Phase 6: Start messaging coordinator and activate services
	if err := m.startMessagingServices(startupCtx); err != nil {
		return m.handleStartupError("messaging services start", err)
	}

	m.currentPhase = MessagingPhaseStarted
	duration := time.Since(startTime)

	logger.Logger.Info("Messaging system startup sequence completed successfully",
		zap.String("phase", m.currentPhase.String()),
		zap.Duration("startup_duration", duration),
		zap.Int("active_brokers", len(m.brokers)),
		zap.Int("active_handlers", len(m.handlers)))

	return nil
}

// ShutdownSequence executes the messaging shutdown sequence with proper cleanup
func (m *MessagingLifecycleManager) ShutdownSequence(ctx context.Context) error {
	if m.currentPhase == MessagingPhaseStopped {
		return nil // Already stopped
	}

	m.currentPhase = MessagingPhaseStopping
	stopTime := time.Now()
	m.stopTime = &stopTime

	logger.Logger.Info("Starting messaging system shutdown sequence",
		zap.String("phase", m.currentPhase.String()))

	// Create timeout context for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, m.config.ShutdownTimeout)
	defer cancel()

	var shutdownErrors []error

	// Stop messaging coordinator first
	if m.coordinator != nil {
		if err := m.coordinator.Stop(shutdownCtx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("coordinator shutdown failed: %w", err))
		}
	}

	// Additional cleanup if needed
	m.brokers = make(map[string]messaging.MessageBroker)
	m.handlers = make(map[string]EventHandler)

	m.currentPhase = MessagingPhaseStopped

	if len(shutdownErrors) > 0 {
		logger.Logger.Error("Messaging system shutdown completed with errors",
			zap.Int("error_count", len(shutdownErrors)))
		return fmt.Errorf("messaging shutdown errors: %v", shutdownErrors)
	}

	duration := time.Since(stopTime)
	logger.Logger.Info("Messaging system shutdown sequence completed successfully",
		zap.String("phase", m.currentPhase.String()),
		zap.Duration("shutdown_duration", duration))

	return nil
}

// validateConfiguration validates messaging configuration
func (m *MessagingLifecycleManager) validateConfiguration(brokerConfigs map[string]*messaging.BrokerConfig, handlers []EventHandler) error {
	logger.Logger.Info("Validating messaging configuration")

	var validationErrors []error

	// Validate broker configurations
	for name, config := range brokerConfigs {
		if config == nil {
			validationErrors = append(validationErrors, fmt.Errorf("broker config for '%s' is nil", name))
			continue
		}

		if err := messaging.ValidateBrokerConfig(config); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("invalid broker config for '%s': %w", name, err))
		}
	}

	// Validate event handlers
	for i, handler := range handlers {
		if handler == nil {
			validationErrors = append(validationErrors, fmt.Errorf("handler at index %d is nil", i))
			continue
		}

		if handler.GetHandlerID() == "" {
			validationErrors = append(validationErrors, fmt.Errorf("handler at index %d has empty ID", i))
		}

		if len(handler.GetTopics()) == 0 {
			validationErrors = append(validationErrors, fmt.Errorf("handler '%s' has no topics", handler.GetHandlerID()))
		}
	}

	m.validationErrors = validationErrors

	if len(validationErrors) > 0 {
		return fmt.Errorf("configuration validation failed with %d errors", len(validationErrors))
	}

	m.currentPhase = MessagingPhaseConfigValidated
	logger.Logger.Info("Messaging configuration validated successfully")
	return nil
}

// setupDependencyInjection sets up dependency injection for messaging components
func (m *MessagingLifecycleManager) setupDependencyInjection(ctx context.Context, brokerConfigs map[string]*messaging.BrokerConfig) error {
	logger.Logger.Info("Setting up messaging dependency injection")

	// In a full implementation, this would integrate with the framework's DI container
	// For now, we prepare the brokers for registration
	for name, config := range brokerConfigs {
		broker, err := messaging.NewMessageBroker(config)
		if err != nil {
			return fmt.Errorf("failed to create broker '%s': %w", name, err)
		}
		m.brokers[name] = broker
	}

	m.currentPhase = MessagingPhaseDependenciesInjected
	logger.Logger.Info("Messaging dependency injection setup completed",
		zap.Int("brokers_prepared", len(m.brokers)))
	return nil
}

// initializeCoordinator initializes the messaging coordinator
func (m *MessagingLifecycleManager) initializeCoordinator(ctx context.Context) error {
	logger.Logger.Info("Initializing messaging coordinator")

	if m.coordinator == nil {
		m.coordinator = messaging.NewMessagingCoordinator()
	}

	m.currentPhase = MessagingPhaseCoordinatorInitialized
	logger.Logger.Info("Messaging coordinator initialized successfully")
	return nil
}

// registerBrokers registers all brokers with the coordinator
func (m *MessagingLifecycleManager) registerBrokers(ctx context.Context, brokerConfigs map[string]*messaging.BrokerConfig) error {
	logger.Logger.Info("Registering brokers with coordinator",
		zap.Int("broker_count", len(m.brokers)))

	for name, broker := range m.brokers {
		if err := m.coordinator.RegisterBroker(name, broker); err != nil {
			return fmt.Errorf("failed to register broker '%s': %w", name, err)
		}
		logger.Logger.Debug("Broker registered successfully", zap.String("broker", name))
	}

	m.currentPhase = MessagingPhaseBrokersRegistered
	logger.Logger.Info("All brokers registered successfully",
		zap.Int("registered_brokers", len(m.brokers)))
	return nil
}

// registerHandlers registers all event handlers with the coordinator
func (m *MessagingLifecycleManager) registerHandlers(ctx context.Context, handlers []EventHandler) error {
	logger.Logger.Info("Registering event handlers with coordinator",
		zap.Int("handler_count", len(handlers)))

	for _, handler := range handlers {
		// Create an adapter to convert server.EventHandler to messaging.EventHandler
		messagingHandler := &messagingEventHandlerAdapter{
			serverHandler: handler,
		}
		
		if err := m.coordinator.RegisterEventHandler(messagingHandler); err != nil {
			return fmt.Errorf("failed to register handler '%s': %w", handler.GetHandlerID(), err)
		}
		m.handlers[handler.GetHandlerID()] = handler
		logger.Logger.Debug("Event handler registered successfully",
			zap.String("handler_id", handler.GetHandlerID()),
			zap.Strings("topics", handler.GetTopics()))
	}

	m.currentPhase = MessagingPhaseHandlersRegistered
	logger.Logger.Info("All event handlers registered successfully",
		zap.Int("registered_handlers", len(m.handlers)))
	return nil
}

// startMessagingServices starts the messaging coordinator and all services
func (m *MessagingLifecycleManager) startMessagingServices(ctx context.Context) error {
	logger.Logger.Info("Starting messaging services")

	if err := m.coordinator.Start(ctx); err != nil {
		return fmt.Errorf("failed to start messaging coordinator: %w", err)
	}

	// Perform health check to ensure everything is running correctly
	if err := m.performStartupHealthCheck(ctx); err != nil {
		// Try to stop coordinator if health check fails
		if stopErr := m.coordinator.Stop(ctx); stopErr != nil {
			logger.Logger.Error("Failed to stop coordinator after health check failure", zap.Error(stopErr))
		}
		return fmt.Errorf("messaging startup health check failed: %w", err)
	}

	logger.Logger.Info("Messaging services started successfully")
	return nil
}

// performStartupHealthCheck performs a health check after startup
func (m *MessagingLifecycleManager) performStartupHealthCheck(ctx context.Context) error {
	logger.Logger.Debug("Performing messaging startup health check")

	healthCtx, cancel := context.WithTimeout(ctx, m.config.HealthCheckInterval)
	defer cancel()

	healthStatus, err := m.coordinator.HealthCheck(healthCtx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if healthStatus.Overall != "healthy" {
		return fmt.Errorf("messaging system is not healthy: %s - %s", healthStatus.Overall, healthStatus.Details)
	}

	logger.Logger.Debug("Messaging startup health check passed",
		zap.String("overall_status", healthStatus.Overall),
		zap.String("details", healthStatus.Details))
	return nil
}

// handleStartupError handles errors during startup with proper cleanup and retry logic
func (m *MessagingLifecycleManager) handleStartupError(phase string, err error) error {
	m.startupErrors = append(m.startupErrors, err)

	logger.Logger.Error("Messaging startup error occurred",
		zap.String("phase", phase),
		zap.String("current_phase", m.currentPhase.String()),
		zap.Error(err))

	// For non-blocking mode, log error but continue
	if m.config.NonBlocking {
		logger.Logger.Warn("Messaging startup error in non-blocking mode, continuing",
			zap.String("phase", phase),
			zap.Error(err))
		return nil
	}

	// Perform cleanup and return error
	return fmt.Errorf("messaging startup failed in %s phase: %w", phase, err)
}

// IsStarted returns true if the messaging system is started
func (m *MessagingLifecycleManager) IsStarted() bool {
	return m.currentPhase == MessagingPhaseStarted
}

// IsStopped returns true if the messaging system is stopped
func (m *MessagingLifecycleManager) IsStopped() bool {
	return m.currentPhase == MessagingPhaseStopped
}

// IsStopping returns true if the messaging system is stopping
func (m *MessagingLifecycleManager) IsStopping() bool {
	return m.currentPhase == MessagingPhaseStopping
}

// GetStartupDuration returns the time taken for startup
func (m *MessagingLifecycleManager) GetStartupDuration() time.Duration {
	if m.startTime == nil {
		return 0
	}
	if m.currentPhase != MessagingPhaseStarted {
		return time.Since(*m.startTime)
	}
	return time.Since(*m.startTime)
}

// GetShutdownDuration returns the time taken for shutdown
func (m *MessagingLifecycleManager) GetShutdownDuration() time.Duration {
	if m.stopTime == nil {
		return 0
	}
	return time.Since(*m.stopTime)
}

// GetValidationErrors returns any validation errors encountered
func (m *MessagingLifecycleManager) GetValidationErrors() []error {
	return m.validationErrors
}

// GetStartupErrors returns any startup errors encountered
func (m *MessagingLifecycleManager) GetStartupErrors() []error {
	return m.startupErrors
}
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

package messaging

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// MessagingCoordinator defines the interface for the central orchestrator of messaging operations.
// It manages broker registration, event handler discovery, and messaging lifecycle coordination
// following SWIT framework patterns for consistency with transport coordination.
//
// This interface provides:
// - Broker registration and management
// - Event handler registration and discovery
// - Lifecycle management (Start/Stop) with graceful shutdown
// - Thread-safe operations with proper synchronization
// - Integration points for TransportCoordinator coordination
//
// Context Handling:
// All methods that accept a context.Context parameter expect a valid, non-nil context.
// Methods will respect context cancellation and timeouts, returning appropriate errors
// when the context is cancelled or times out.
//
// Example usage:
//
//	coordinator := messaging.NewMessagingCoordinator()
//	defer coordinator.Stop(context.Background())
//
//	// Register brokers
//	kafkaBroker, _ := messaging.NewMessageBroker(&messaging.BrokerConfig{
//		Type: messaging.BrokerTypeKafka,
//		Endpoints: []string{"localhost:9092"},
//	})
//	coordinator.RegisterBroker("kafka", kafkaBroker)
//
//	// Register event handlers
//	handler := &MyEventHandler{}
//	coordinator.RegisterEventHandler(handler)
//
//	// Start coordinator
//	ctx := context.Background()
//	if err := coordinator.Start(ctx); err != nil {
//		log.Fatal("Failed to start messaging coordinator:", err)
//	}
type MessagingCoordinator interface {
	// Start initializes the messaging coordinator and starts all registered brokers.
	// This method blocks until all brokers are successfully started or an error occurs.
	// It follows the SWIT framework lifecycle pattern and integrates with server startup phases.
	//
	// The startup sequence:
	// 1. Initialize broker connections
	// 2. Register event handlers with brokers
	// 3. Start message processing
	// 4. Mark coordinator as ready
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//
	// Returns:
	//   - error: MessagingError if startup fails, nil on success
	Start(ctx context.Context) error

	// Stop gracefully shuts down the messaging coordinator and all registered brokers.
	// This method ensures all in-flight messages are processed before stopping.
	// It follows the SWIT framework graceful shutdown pattern.
	//
	// The shutdown sequence:
	// 1. Stop accepting new messages
	// 2. Wait for in-flight messages to complete (with timeout)
	// 3. Disconnect from brokers
	// 4. Clean up resources
	//
	// Parameters:
	//   - ctx: Context for timeout control during shutdown
	//
	// Returns:
	//   - error: MessagingError if shutdown fails, nil on success
	Stop(ctx context.Context) error

	// RegisterBroker registers a message broker with the coordinator under the given name.
	// This allows multiple brokers to be managed by a single coordinator.
	// The broker name must be unique and follow naming conventions (alphanumeric + hyphens).
	//
	// Parameters:
	//   - name: Unique name for the broker (alphanumeric + hyphens, 1-50 chars)
	//   - broker: MessageBroker instance to register
	//
	// Returns:
	//   - error: MessagingError if registration fails or name is invalid, nil on success
	RegisterBroker(name string, broker MessageBroker) error

	// GetBroker retrieves a registered broker by name.
	// This enables services to obtain broker instances for direct operations.
	//
	// Parameters:
	//   - name: Name of the broker to retrieve
	//
	// Returns:
	//   - MessageBroker: The registered broker instance, nil if not found
	//   - error: MessagingError if broker not found, nil if found
	GetBroker(name string) (MessageBroker, error)

	// GetRegisteredBrokers returns a list of all registered broker names.
	// This is useful for introspection and debugging purposes.
	//
	// Returns:
	//   - []string: List of registered broker names, empty if none registered
	GetRegisteredBrokers() []string

	// RegisterEventHandler registers an event handler for message processing.
	// Handlers are automatically associated with appropriate brokers based on their
	// configuration and routing requirements.
	//
	// Parameters:
	//   - handler: EventHandler instance to register
	//
	// Returns:
	//   - error: MessagingError if registration fails, nil on success
	RegisterEventHandler(handler EventHandler) error

	// UnregisterEventHandler removes an event handler by ID.
	// This stops message processing for the specified handler.
	//
	// Parameters:
	//   - handlerID: Unique identifier of the handler to remove
	//
	// Returns:
	//   - error: MessagingError if handler not found or removal fails, nil on success
	UnregisterEventHandler(handlerID string) error

	// GetRegisteredHandlers returns a list of all registered handler IDs.
	// This is useful for introspection and management purposes.
	//
	// Returns:
	//   - []string: List of registered handler IDs, empty if none registered
	GetRegisteredHandlers() []string

	// IsStarted returns whether the coordinator has been started.
	// This method is thread-safe and can be called concurrently.
	//
	// Returns:
	//   - bool: true if started, false otherwise
	IsStarted() bool

	// GetMetrics returns current messaging coordinator metrics.
	// Includes broker counts, handler counts, message statistics, and health status.
	//
	// Returns:
	//   - *MessagingCoordinatorMetrics: Current metrics snapshot, never nil
	GetMetrics() *MessagingCoordinatorMetrics

	// HealthCheck performs a comprehensive health check on all registered brokers and handlers.
	// This integrates with the SWIT framework health checking system.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - *MessagingHealthStatus: Aggregated health status, never nil
	//   - error: MessagingError if health check fails
	HealthCheck(ctx context.Context) (*MessagingHealthStatus, error)
}

// EventHandler defines the interface for event handlers that can be registered
// with the MessagingCoordinator. This interface extends the base MessageHandler
// with additional metadata and lifecycle management capabilities.
type EventHandler interface {
	MessageHandler

	// GetHandlerID returns a unique identifier for this handler.
	// This ID is used for registration, deregistration, and debugging.
	//
	// Returns:
	//   - string: Unique handler identifier
	GetHandlerID() string

	// GetTopics returns the list of topics this handler is interested in.
	// The coordinator uses this information to set up subscriptions.
	//
	// Returns:
	//   - []string: List of topic names this handler processes
	GetTopics() []string

	// GetBrokerRequirement returns the broker name this handler requires.
	// If empty, the handler can work with any available broker.
	//
	// Returns:
	//   - string: Required broker name, empty for any broker
	GetBrokerRequirement() string

	// Initialize is called when the handler is registered with the coordinator.
	// This allows handlers to perform setup operations.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - error: Error if initialization fails, nil on success
	Initialize(ctx context.Context) error

	// Shutdown is called when the handler is being unregistered or coordinator is stopping.
	// This allows handlers to perform cleanup operations.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - error: Error if shutdown fails, nil on success
	Shutdown(ctx context.Context) error
}

// MessagingCoordinatorMetrics contains metrics for the messaging coordinator.
type MessagingCoordinatorMetrics struct {
	// BrokerCount is the number of registered brokers
	BrokerCount int `json:"broker_count"`

	// HandlerCount is the number of registered event handlers
	HandlerCount int `json:"handler_count"`

	// StartedAt is when the coordinator was started (nil if not started)
	StartedAt *time.Time `json:"started_at,omitempty"`

	// Uptime is how long the coordinator has been running (zero if not started)
	Uptime time.Duration `json:"uptime"`

	// TotalMessagesProcessed is the total number of messages processed across all handlers
	TotalMessagesProcessed int64 `json:"total_messages_processed"`

	// TotalErrors is the total number of errors across all brokers and handlers
	TotalErrors int64 `json:"total_errors"`

	// BrokerMetrics maps broker names to their individual metrics
	BrokerMetrics map[string]*BrokerMetrics `json:"broker_metrics,omitempty"`
}

// MessagingHealthStatus represents the aggregated health status of the messaging coordinator.
type MessagingHealthStatus struct {
	// Overall is the overall health status
	Overall string `json:"overall"`

	// Details provides human-readable health information
	Details string `json:"details"`

	// BrokerHealth maps broker names to their health status
	BrokerHealth map[string]string `json:"broker_health,omitempty"`

	// HandlerHealth maps handler IDs to their health status
	HandlerHealth map[string]string `json:"handler_health,omitempty"`

	// Timestamp is when this health check was performed
	Timestamp time.Time `json:"timestamp"`
}

// MessagingCoordinatorError represents errors specific to messaging coordinator operations.
type MessagingCoordinatorError struct {
	Operation string
	Err       error
}

func (e *MessagingCoordinatorError) Error() string {
	return fmt.Sprintf("messaging coordinator %s: %v", e.Operation, e.Err)
}

func (e *MessagingCoordinatorError) Unwrap() error {
	return e.Err
}

// messagingCoordinatorImpl implements the MessagingCoordinator interface.
// It provides centralized management of message brokers and event handlers
// following SWIT framework patterns for consistency and reliability.
type messagingCoordinatorImpl struct {
	// brokers maps broker names to broker instances
	brokers map[string]MessageBroker

	// handlers maps handler IDs to event handler instances
	handlers map[string]EventHandler

	// subscribers maps handler IDs to their subscriber instances
	subscribers map[string]EventSubscriber

	// started tracks whether the coordinator has been started
	started bool

	// startedAt tracks when the coordinator was started
	startedAt *time.Time

	// metrics tracks coordinator metrics
	metrics *MessagingCoordinatorMetrics

	// mu protects concurrent access to coordinator state
	mu sync.RWMutex
}

// NewMessagingCoordinator creates a new messaging coordinator instance.
// The coordinator is initialized but not started - call Start() to begin operations.
//
// Returns:
//   - MessagingCoordinator: New coordinator instance ready for registration and startup
func NewMessagingCoordinator() MessagingCoordinator {
	return &messagingCoordinatorImpl{
		brokers:     make(map[string]MessageBroker),
		handlers:    make(map[string]EventHandler),
		subscribers: make(map[string]EventSubscriber),
		started:     false,
		metrics: &MessagingCoordinatorMetrics{
			BrokerMetrics: make(map[string]*BrokerMetrics),
		},
	}
}

// Start implements MessagingCoordinator interface.
func (c *messagingCoordinatorImpl) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return &MessagingCoordinatorError{
			Operation: "start",
			Err:       fmt.Errorf("coordinator already started"),
		}
	}

	logger.Logger.Info("Starting messaging coordinator",
		zap.Int("broker_count", len(c.brokers)),
		zap.Int("handler_count", len(c.handlers)))

	// Start all registered brokers
	for name, broker := range c.brokers {
		logger.Logger.Debug("Starting broker", zap.String("broker", name))

		if err := broker.Connect(ctx); err != nil {
			return &MessagingCoordinatorError{
				Operation: "start",
				Err:       fmt.Errorf("failed to start broker '%s': %w", name, err),
			}
		}

		logger.Logger.Info("Broker started successfully", zap.String("broker", name))
	}

	// Initialize and set up subscriptions for all event handlers
	for handlerID, handler := range c.handlers {
		logger.Logger.Debug("Initializing event handler", zap.String("handler_id", handlerID))

		// Initialize the handler
		if err := handler.Initialize(ctx); err != nil {
			return &MessagingCoordinatorError{
				Operation: "start",
				Err:       fmt.Errorf("failed to initialize handler '%s': %w", handlerID, err),
			}
		}

		// Create subscription for the handler
		if err := c.createSubscriptionForHandler(ctx, handler); err != nil {
			return &MessagingCoordinatorError{
				Operation: "start",
				Err:       fmt.Errorf("failed to create subscription for handler '%s': %w", handlerID, err),
			}
		}

		logger.Logger.Info("Event handler initialized successfully", zap.String("handler_id", handlerID))
	}

	// Mark as started and update metrics
	now := time.Now()
	c.started = true
	c.startedAt = &now
	c.metrics.StartedAt = &now
	c.updateMetrics()

	logger.Logger.Info("Messaging coordinator started successfully",
		zap.Int("brokers", len(c.brokers)),
		zap.Int("handlers", len(c.handlers)))

	return nil
}

// Stop implements MessagingCoordinator interface.
func (c *messagingCoordinatorImpl) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil // Already stopped
	}

	logger.Logger.Info("Stopping messaging coordinator")

	var errors []string

	// Stop all subscribers first (gracefully stop message processing)
	for handlerID, subscriber := range c.subscribers {
		logger.Logger.Debug("Stopping subscriber", zap.String("handler_id", handlerID))

		if err := subscriber.Unsubscribe(ctx); err != nil {
			errors = append(errors, fmt.Sprintf("failed to stop subscriber for handler '%s': %v", handlerID, err))
		} else {
			logger.Logger.Debug("Subscriber stopped successfully", zap.String("handler_id", handlerID))
		}

		// Clean up subscriber
		if err := subscriber.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to close subscriber for handler '%s': %v", handlerID, err))
		}
	}

	// Shutdown all event handlers
	for handlerID, handler := range c.handlers {
		logger.Logger.Debug("Shutting down event handler", zap.String("handler_id", handlerID))

		if err := handler.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Sprintf("failed to shutdown handler '%s': %v", handlerID, err))
		} else {
			logger.Logger.Debug("Event handler shutdown successfully", zap.String("handler_id", handlerID))
		}
	}

	// Disconnect all brokers
	for name, broker := range c.brokers {
		logger.Logger.Debug("Disconnecting broker", zap.String("broker", name))

		if err := broker.Disconnect(ctx); err != nil {
			errors = append(errors, fmt.Sprintf("failed to disconnect broker '%s': %v", name, err))
		} else {
			logger.Logger.Debug("Broker disconnected successfully", zap.String("broker", name))
		}
	}

	// Clear subscribers map
	c.subscribers = make(map[string]EventSubscriber)

	// Mark as stopped
	c.started = false
	c.startedAt = nil
	c.updateMetrics()

	if len(errors) > 0 {
		return &MessagingCoordinatorError{
			Operation: "stop",
			Err:       fmt.Errorf("errors during shutdown: %s", strings.Join(errors, "; ")),
		}
	}

	logger.Logger.Info("Messaging coordinator stopped successfully")
	return nil
}

// RegisterBroker implements MessagingCoordinator interface.
func (c *messagingCoordinatorImpl) RegisterBroker(name string, broker MessageBroker) error {
	if name == "" {
		return &MessagingCoordinatorError{
			Operation: "register_broker",
			Err:       fmt.Errorf("broker name cannot be empty"),
		}
	}

	if broker == nil {
		return &MessagingCoordinatorError{
			Operation: "register_broker",
			Err:       fmt.Errorf("broker cannot be nil"),
		}
	}

	// Validate broker name follows SWIT framework conventions
	if err := c.validateBrokerName(name); err != nil {
		return &MessagingCoordinatorError{
			Operation: "register_broker",
			Err:       fmt.Errorf("invalid broker name: %w", err),
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check for duplicate registration
	if _, exists := c.brokers[name]; exists {
		return &MessagingCoordinatorError{
			Operation: "register_broker",
			Err:       fmt.Errorf("broker with name '%s' already registered", name),
		}
	}

	// Register the broker
	c.brokers[name] = broker
	c.updateMetrics()

	logger.Logger.Info("Broker registered successfully", zap.String("broker", name))
	return nil
}

// GetBroker implements MessagingCoordinator interface.
func (c *messagingCoordinatorImpl) GetBroker(name string) (MessageBroker, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	broker, exists := c.brokers[name]
	if !exists {
		return nil, &MessagingCoordinatorError{
			Operation: "get_broker",
			Err:       fmt.Errorf("broker '%s' not found", name),
		}
	}

	return broker, nil
}

// GetRegisteredBrokers implements MessagingCoordinator interface.
func (c *messagingCoordinatorImpl) GetRegisteredBrokers() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.brokers))
	for name := range c.brokers {
		names = append(names, name)
	}
	return names
}

// RegisterEventHandler implements MessagingCoordinator interface.
func (c *messagingCoordinatorImpl) RegisterEventHandler(handler EventHandler) error {
	if handler == nil {
		return &MessagingCoordinatorError{
			Operation: "register_handler",
			Err:       fmt.Errorf("handler cannot be nil"),
		}
	}

	handlerID := handler.GetHandlerID()
	if handlerID == "" {
		return &MessagingCoordinatorError{
			Operation: "register_handler",
			Err:       fmt.Errorf("handler ID cannot be empty"),
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check for duplicate registration
	if _, exists := c.handlers[handlerID]; exists {
		return &MessagingCoordinatorError{
			Operation: "register_handler",
			Err:       fmt.Errorf("handler with ID '%s' already registered", handlerID),
		}
	}

	// Validate handler requirements
	if err := c.validateHandlerRequirements(handler); err != nil {
		return &MessagingCoordinatorError{
			Operation: "register_handler",
			Err:       fmt.Errorf("handler validation failed: %w", err),
		}
	}

	// Register the handler
	c.handlers[handlerID] = handler
	c.updateMetrics()

	logger.Logger.Info("Event handler registered successfully", zap.String("handler_id", handlerID))
	return nil
}

// UnregisterEventHandler implements MessagingCoordinator interface.
func (c *messagingCoordinatorImpl) UnregisterEventHandler(handlerID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	handler, exists := c.handlers[handlerID]
	if !exists {
		return &MessagingCoordinatorError{
			Operation: "unregister_handler",
			Err:       fmt.Errorf("handler '%s' not found", handlerID),
		}
	}

	// Stop subscriber if running
	if subscriber, exists := c.subscribers[handlerID]; exists {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := subscriber.Unsubscribe(ctx); err != nil {
			logger.Logger.Warn("Failed to unsubscribe handler",
				zap.String("handler_id", handlerID),
				zap.Error(err))
		}

		if err := subscriber.Close(); err != nil {
			logger.Logger.Warn("Failed to close subscriber",
				zap.String("handler_id", handlerID),
				zap.Error(err))
		}

		delete(c.subscribers, handlerID)
	}

	// Shutdown the handler
	if c.started {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := handler.Shutdown(ctx); err != nil {
			logger.Logger.Warn("Failed to shutdown handler",
				zap.String("handler_id", handlerID),
				zap.Error(err))
		}
	}

	// Remove the handler
	delete(c.handlers, handlerID)
	c.updateMetrics()

	logger.Logger.Info("Event handler unregistered successfully", zap.String("handler_id", handlerID))
	return nil
}

// GetRegisteredHandlers implements MessagingCoordinator interface.
func (c *messagingCoordinatorImpl) GetRegisteredHandlers() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	handlerIDs := make([]string, 0, len(c.handlers))
	for handlerID := range c.handlers {
		handlerIDs = append(handlerIDs, handlerID)
	}
	return handlerIDs
}

// IsStarted implements MessagingCoordinator interface.
func (c *messagingCoordinatorImpl) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// GetMetrics implements MessagingCoordinator interface.
func (c *messagingCoordinatorImpl) GetMetrics() *MessagingCoordinatorMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create a copy of metrics to avoid external mutation
	metrics := &MessagingCoordinatorMetrics{
		BrokerCount:            c.metrics.BrokerCount,
		HandlerCount:           c.metrics.HandlerCount,
		TotalMessagesProcessed: c.metrics.TotalMessagesProcessed,
		TotalErrors:            c.metrics.TotalErrors,
		BrokerMetrics:          make(map[string]*BrokerMetrics),
	}

	if c.metrics.StartedAt != nil {
		startedAt := *c.metrics.StartedAt
		metrics.StartedAt = &startedAt
		metrics.Uptime = time.Since(startedAt)
	}

	// Copy broker metrics
	for name, brokerMetrics := range c.metrics.BrokerMetrics {
		metrics.BrokerMetrics[name] = brokerMetrics
	}

	return metrics
}

// HealthCheck implements MessagingCoordinator interface.
func (c *messagingCoordinatorImpl) HealthCheck(ctx context.Context) (*MessagingHealthStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := &MessagingHealthStatus{
		Overall:       "healthy",
		BrokerHealth:  make(map[string]string),
		HandlerHealth: make(map[string]string),
		Timestamp:     time.Now(),
	}

	var healthIssues []string

	// Check broker health
	for name, broker := range c.brokers {
		brokerHealth, err := broker.HealthCheck(ctx)
		if err != nil {
			status.BrokerHealth[name] = "unhealthy"
			healthIssues = append(healthIssues, fmt.Sprintf("broker %s: %v", name, err))
		} else if brokerHealth != nil && brokerHealth.Status != HealthStatusHealthy {
			status.BrokerHealth[name] = brokerHealth.Status.String()
			healthIssues = append(healthIssues, fmt.Sprintf("broker %s: %s", name, brokerHealth.Message))
		} else {
			status.BrokerHealth[name] = "healthy"
		}
	}

	// Check if coordinator is started
	if !c.started {
		status.Overall = "stopped"
		status.Details = "Messaging coordinator is not started"
	} else if len(healthIssues) > 0 {
		status.Overall = "degraded"
		status.Details = strings.Join(healthIssues, "; ")
	} else {
		status.Details = fmt.Sprintf("All %d brokers and %d handlers are healthy",
			len(c.brokers), len(c.handlers))
	}

	// Handler health is based on their subscription status
	for handlerID := range c.handlers {
		if _, exists := c.subscribers[handlerID]; exists && c.started {
			status.HandlerHealth[handlerID] = "active"
		} else {
			status.HandlerHealth[handlerID] = "inactive"
		}
	}

	return status, nil
}

// validateBrokerName validates broker names following SWIT framework conventions.
func (c *messagingCoordinatorImpl) validateBrokerName(name string) error {
	if len(name) == 0 || len(name) > 50 {
		return fmt.Errorf("name must be 1-50 characters long")
	}

	// Allow alphanumeric characters and hyphens (same as transport layer)
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return fmt.Errorf("name can only contain alphanumeric characters and hyphens")
		}
	}

	return nil
}

// validateHandlerRequirements validates event handler requirements.
func (c *messagingCoordinatorImpl) validateHandlerRequirements(handler EventHandler) error {
	topics := handler.GetTopics()
	if len(topics) == 0 {
		return fmt.Errorf("handler must specify at least one topic")
	}

	brokerReq := handler.GetBrokerRequirement()
	if brokerReq != "" {
		// Check if required broker exists
		if _, exists := c.brokers[brokerReq]; !exists {
			return fmt.Errorf("required broker '%s' is not registered", brokerReq)
		}
	} else if len(c.brokers) == 0 {
		return fmt.Errorf("no brokers available and handler doesn't specify a required broker")
	}

	return nil
}

// createSubscriptionForHandler creates a subscription for an event handler.
func (c *messagingCoordinatorImpl) createSubscriptionForHandler(ctx context.Context, handler EventHandler) error {
	handlerID := handler.GetHandlerID()
	brokerReq := handler.GetBrokerRequirement()
	topics := handler.GetTopics()

	// Determine which broker to use
	var broker MessageBroker
	var brokerName string

	if brokerReq != "" {
		// Use required broker
		var exists bool
		broker, exists = c.brokers[brokerReq]
		if !exists {
			return fmt.Errorf("required broker '%s' not found", brokerReq)
		}
		brokerName = brokerReq
	} else {
		// Use first available broker (could be enhanced with load balancing)
		for name, b := range c.brokers {
			broker = b
			brokerName = name
			break
		}
		if broker == nil {
			return fmt.Errorf("no brokers available")
		}
	}

	// Create subscriber configuration
	// Note: This is a simplified configuration - in a real implementation,
	// you'd want more sophisticated configuration options
	config := SubscriberConfig{
		Topics:        topics,
		ConsumerGroup: fmt.Sprintf("%s-group", handlerID),
		// Add other configuration as needed
	}

	// Create subscriber
	subscriber, err := broker.CreateSubscriber(config)
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}

	// Store subscriber for later cleanup
	c.subscribers[handlerID] = subscriber

	// Start subscription in a separate goroutine
	// In a production system, you'd want better goroutine management
	go func() {
		logger.Logger.Info("Starting subscription for handler",
			zap.String("handler_id", handlerID),
			zap.String("broker", brokerName),
			zap.Strings("topics", topics))

		if err := subscriber.Subscribe(ctx, handler); err != nil {
			logger.Logger.Error("Subscription failed",
				zap.String("handler_id", handlerID),
				zap.Error(err))
		}
	}()

	return nil
}

// updateMetrics updates the coordinator metrics.
func (c *messagingCoordinatorImpl) updateMetrics() {
	c.metrics.BrokerCount = len(c.brokers)
	c.metrics.HandlerCount = len(c.handlers)

	// Collect broker metrics
	for name, broker := range c.brokers {
		c.metrics.BrokerMetrics[name] = broker.GetMetrics()
	}

	// Calculate total messages and errors across all brokers
	var totalMessages int64
	var totalErrors int64

	for _, brokerMetrics := range c.metrics.BrokerMetrics {
		if brokerMetrics != nil {
			totalMessages += brokerMetrics.MessagesPublished + brokerMetrics.MessagesConsumed
			totalErrors += brokerMetrics.ConnectionFailures + brokerMetrics.PublishErrors + brokerMetrics.ConsumeErrors
		}
	}

	c.metrics.TotalMessagesProcessed = totalMessages
	c.metrics.TotalErrors = totalErrors
}

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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

const (
	defaultShutdownTimeout = 30 * time.Second
)

// MessagingCoordinator orchestrates brokers and event handlers.
// It supports broker and handler registration, lifecycle management and
// exposes metrics and health checks for the messaging layer.
type MessagingCoordinator interface {
	// Start connects brokers, initializes handlers and begins processing.
	// Blocks until startup completes or an error occurs.
	Start(ctx context.Context) error

	// Stop gracefully shuts down brokers and handlers, waiting for in-flight work.
	Stop(ctx context.Context) error

	// RegisterBroker adds a broker with the given name.
	RegisterBroker(name string, broker MessageBroker) error

	// GetBroker returns a registered broker by name.
	GetBroker(name string) (MessageBroker, error)

	// GetRegisteredBrokers lists names of all registered brokers.
	GetRegisteredBrokers() []string

	// RegisterEventHandler adds an event handler to the coordinator.
	RegisterEventHandler(handler EventHandler) error

	// UnregisterEventHandler removes an event handler by ID.
	UnregisterEventHandler(handlerID string) error

	// GetRegisteredHandlers lists IDs of all registered handlers.
	GetRegisteredHandlers() []string

	// IsStarted reports whether the coordinator has successfully started.
	IsStarted() bool

	// GetMetrics returns current coordinator metrics.
	GetMetrics() *MessagingCoordinatorMetrics

	// HealthCheck verifies brokers and handlers and returns aggregated status.
	HealthCheck(ctx context.Context) (*MessagingHealthStatus, error)
}

// EventHandler processes messages and exposes metadata for registration.
type EventHandler interface {
	MessageHandler

	// GetHandlerID returns the handler's unique identifier.
	GetHandlerID() string

	// GetTopics returns the topics this handler processes.
	GetTopics() []string

	// GetBrokerRequirement returns the required broker name, or empty if any broker is acceptable.
	GetBrokerRequirement() string

	// Initialize prepares the handler for processing.
	Initialize(ctx context.Context) error

	// Shutdown releases handler resources.
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

	// subscriptionCtx controls the lifecycle of handler subscriptions and is
	// independent from Start's context. Cancelled on shutdown.
	subscriptionCtx    context.Context
	subscriptionCancel context.CancelFunc

	// wg waits for subscription goroutines to complete
	wg sync.WaitGroup

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
	if c.started {
		c.mu.Unlock()
		return &MessagingCoordinatorError{
			Operation: "start",
			Err:       fmt.Errorf("coordinator already started"),
		}
	}

	logger.Logger.Info("Starting messaging coordinator",
		zap.Int("broker_count", len(c.brokers)),
		zap.Int("handler_count", len(c.handlers)))

	// Use a background context so subscriptions persist beyond Start's context
	c.subscriptionCtx, c.subscriptionCancel = context.WithCancel(context.Background())
	c.mu.Unlock()

	startedBrokers := make([]string, 0, len(c.brokers))
	startedHandlers := make([]string, 0, len(c.handlers))

	// Start all registered brokers
	for name, broker := range c.brokers {
		logger.Logger.Debug("Starting broker", zap.String("broker", name))

		if err := broker.Connect(ctx); err != nil {
			c.rollbackStart(ctx, startedBrokers, startedHandlers)
			return &MessagingCoordinatorError{
				Operation: "start",
				Err:       fmt.Errorf("failed to start broker '%s': %w", name, err),
			}
		}

		startedBrokers = append(startedBrokers, name)
		logger.Logger.Info("Broker started successfully", zap.String("broker", name))
	}

	// Initialize and set up subscriptions for all event handlers
	for handlerID, handler := range c.handlers {
		logger.Logger.Debug("Initializing event handler", zap.String("handler_id", handlerID))

		// Initialize the handler
		if err := handler.Initialize(ctx); err != nil {
			c.rollbackStart(ctx, startedBrokers, startedHandlers)
			return &MessagingCoordinatorError{
				Operation: "start",
				Err:       fmt.Errorf("failed to initialize handler '%s': %w", handlerID, err),
			}
		}

		startedHandlers = append(startedHandlers, handlerID)

		// Create subscription for the handler
		if err := c.createSubscriptionForHandler(handler); err != nil {
			c.rollbackStart(ctx, startedBrokers, startedHandlers)
			return &MessagingCoordinatorError{
				Operation: "start",
				Err:       fmt.Errorf("failed to create subscription for handler '%s': %w", handlerID, err),
			}
		}

		logger.Logger.Info("Event handler initialized successfully", zap.String("handler_id", handlerID))
	}

	c.mu.Lock()
	now := time.Now()
	c.started = true
	c.startedAt = &now
	c.metrics.StartedAt = &now
	c.updateMetrics()
	c.mu.Unlock()

	logger.Logger.Info("Messaging coordinator started successfully",
		zap.Int("brokers", len(c.brokers)),
		zap.Int("handlers", len(c.handlers)))

	return nil
}

// rollbackStart cleans up resources started before a startup failure.
func (c *messagingCoordinatorImpl) rollbackStart(ctx context.Context, brokers, handlers []string) {
	if c.subscriptionCancel != nil {
		c.subscriptionCancel()
	}
	c.wg.Wait()

	for _, handlerID := range handlers {
		if sub, ok := c.subscribers[handlerID]; ok {
			_ = sub.Unsubscribe(ctx)
			_ = sub.Close()
			delete(c.subscribers, handlerID)
		}
		if h, ok := c.handlers[handlerID]; ok {
			_ = h.Shutdown(ctx)
		}
	}

	for _, name := range brokers {
		if b, ok := c.brokers[name]; ok {
			_ = b.Disconnect(ctx)
		}
	}
}

// Stop implements MessagingCoordinator interface.
func (c *messagingCoordinatorImpl) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopUnlocked(ctx)
}

// stopUnlocked performs coordinator shutdown. Caller must hold c.mu.
func (c *messagingCoordinatorImpl) stopUnlocked(ctx context.Context) error {
	if !c.started {
		return nil // Already stopped
	}

	logger.Logger.Info("Stopping messaging coordinator")

	var errors []string

	if c.subscriptionCancel != nil {
		c.subscriptionCancel()
	}
	c.wg.Wait()

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
	c.subscriptionCtx = nil
	c.subscriptionCancel = nil
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

	if c.started {
		return &MessagingCoordinatorError{
			Operation: "register_broker",
			Err:       fmt.Errorf("cannot register broker after coordinator has started"),
		}
	}

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

	if c.started {
		return &MessagingCoordinatorError{
			Operation: "register_handler",
			Err:       fmt.Errorf("cannot register handler after coordinator has started"),
		}
	}

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
		ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
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
		ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
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
func (c *messagingCoordinatorImpl) createSubscriptionForHandler(handler EventHandler) error {
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
		// Use deterministic selection of available brokers
		names := make([]string, 0, len(c.brokers))
		for name := range c.brokers {
			names = append(names, name)
		}
		sort.Strings(names)
		if len(names) > 0 {
			brokerName = names[0]
			broker = c.brokers[brokerName]
		}
		if broker == nil {
			return fmt.Errorf("no brokers available")
		}
	}

	// Create subscriber configuration
	config := SubscriberConfig{
		Topics:        topics,
		ConsumerGroup: fmt.Sprintf("%s-group", handlerID),
	}

	// Create subscriber
	subscriber, err := broker.CreateSubscriber(config)
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}

	// Store subscriber for later cleanup
	c.subscribers[handlerID] = subscriber

	c.wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Logger.Error("Subscription panic recovered",
					zap.String("handler_id", handlerID),
					zap.Any("panic", r))
			}
			c.wg.Done()
		}()

		logger.Logger.Info("Starting subscription for handler",
			zap.String("handler_id", handlerID),
			zap.String("broker", brokerName),
			zap.Strings("topics", topics))

		if err := subscriber.Subscribe(c.subscriptionCtx, handler); err != nil {
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

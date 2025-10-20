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

// Package coordinator provides choreography-based Saga coordinator implementation.
// It manages decentralized event-driven coordination where services react to events
// autonomously without central control.
package coordinator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

var (
	// ErrChoreographyCoordinatorClosed indicates the choreography coordinator has been closed.
	ErrChoreographyCoordinatorClosed = errors.New("choreography coordinator is closed")

	// ErrEventHandlerAlreadyRegistered indicates an event handler with the same ID is already registered.
	ErrEventHandlerAlreadyRegistered = errors.New("event handler already registered")

	// ErrEventHandlerNotFound indicates the specified event handler was not found.
	ErrEventHandlerNotFound = errors.New("event handler not found")

	// ErrNoHandlersForEventType indicates no handlers are registered for the event type.
	ErrNoHandlersForEventType = errors.New("no handlers registered for event type")

	// ErrEventPublisherRequired indicates EventPublisher is required but not provided.
	ErrEventPublisherRequired = errors.New("event publisher is required")

	// ErrStateStorageRequired indicates StateStorage is required but not provided.
	ErrStateStorageRequired = errors.New("state storage is required")
)

// ChoreographyEventHandler defines the interface for handling events in choreography mode.
// Services implement this interface to react to Saga events autonomously.
type ChoreographyEventHandler interface {
	// HandleEvent processes a Saga event.
	// Returns an error if event processing fails.
	HandleEvent(ctx context.Context, event *saga.SagaEvent) error

	// GetHandlerID returns the unique identifier for this handler.
	GetHandlerID() string

	// GetSupportedEventTypes returns the list of event types this handler supports.
	GetSupportedEventTypes() []saga.SagaEventType

	// GetPriority returns the execution priority (higher values execute first).
	GetPriority() int
}

// ChoreographyConfig contains configuration options for the choreography coordinator.
type ChoreographyConfig struct {
	// EventPublisher is required for publishing Saga events.
	EventPublisher saga.EventPublisher

	// StateStorage is optional for storing Saga state.
	// If not provided, state management is handled by individual services.
	StateStorage saga.StateStorage

	// MaxConcurrentHandlers limits the number of concurrent event handlers.
	// Default is 100 if not specified.
	MaxConcurrentHandlers int

	// EventTimeout is the timeout for individual event handling.
	// Default is 30 seconds if not specified.
	EventTimeout time.Duration

	// EnableMetrics enables metrics collection.
	// Default is true.
	EnableMetrics bool

	// HandlerRetryPolicy defines the retry policy for failed event handlers.
	// If not provided, a default policy is used.
	HandlerRetryPolicy saga.RetryPolicy
}

// Validate checks if the configuration is valid.
func (c *ChoreographyConfig) Validate() error {
	if c.EventPublisher == nil {
		return ErrEventPublisherRequired
	}
	return nil
}

// ChoreographyCoordinator implements the choreography-based Saga coordination pattern.
// It provides decentralized event-driven coordination where services react to events
// autonomously without central control.
//
// The coordinator follows these key responsibilities:
//  1. Event handler registration and lifecycle management
//  2. Event subscription and routing to appropriate handlers
//  3. Concurrent event processing with configurable limits
//  4. Optional state storage integration
//  5. Metrics collection for monitoring
//  6. Graceful shutdown and cleanup
type ChoreographyCoordinator struct {
	// eventPublisher publishes Saga events to the message broker.
	eventPublisher saga.EventPublisher

	// stateStorage optionally stores Saga state.
	stateStorage saga.StateStorage

	// handlers maps event types to their registered handlers.
	// Multiple handlers can be registered for the same event type.
	handlers map[saga.SagaEventType][]ChoreographyEventHandler

	// handlersByID maps handler IDs to their handlers for efficient lookup.
	handlersByID map[string]ChoreographyEventHandler

	// config holds the coordinator configuration.
	config *ChoreographyConfig

	// subscription holds the event subscription for receiving events.
	subscription saga.EventSubscription

	// metrics collects runtime metrics.
	metrics *ChoreographyMetrics

	// semaphore limits concurrent event handler executions.
	semaphore chan struct{}

	// running indicates if the coordinator is currently running.
	running bool

	// closed indicates if the coordinator has been shut down.
	closed bool

	// ctx is the coordinator context for lifecycle management.
	ctx context.Context

	// cancel cancels the coordinator context.
	cancel context.CancelFunc

	// wg tracks active goroutines for graceful shutdown.
	wg sync.WaitGroup

	// mu protects concurrent access to coordinator state.
	mu sync.RWMutex
}

// ChoreographyMetrics contains metrics about choreography coordination.
type ChoreographyMetrics struct {
	// TotalEvents is the total number of events received.
	TotalEvents int64 `json:"total_events"`

	// ProcessedEvents is the number of events successfully processed.
	ProcessedEvents int64 `json:"processed_events"`

	// FailedEvents is the number of events that failed processing.
	FailedEvents int64 `json:"failed_events"`

	// ActiveHandlers is the current number of active event handlers.
	ActiveHandlers int64 `json:"active_handlers"`

	// TotalHandlers is the total number of registered handlers.
	TotalHandlers int64 `json:"total_handlers"`

	// AverageProcessingTime is the average event processing time.
	AverageProcessingTime time.Duration `json:"average_processing_time"`

	// StartTime is when the coordinator was started.
	StartTime time.Time `json:"start_time"`

	// LastUpdateTime is when metrics were last updated.
	LastUpdateTime time.Time `json:"last_update_time"`

	mu sync.RWMutex
}

// NewChoreographyCoordinator creates a new choreography-based Saga coordinator.
// It initializes all dependencies and prepares the coordinator for managing
// decentralized event-driven Saga coordination.
//
// Parameters:
//   - config: Configuration containing required dependencies (EventPublisher)
//     and optional components (StateStorage, retry policies).
//
// Returns:
//   - A configured ChoreographyCoordinator instance ready to coordinate Sagas.
//   - An error if the configuration is invalid or initialization fails.
//
// Example:
//
//	config := &ChoreographyConfig{
//	    EventPublisher: eventPublisher,
//	    StateStorage: stateStorage,
//	    MaxConcurrentHandlers: 100,
//	    EventTimeout: 30 * time.Second,
//	}
//	coordinator, err := NewChoreographyCoordinator(config)
//	if err != nil {
//	    return err
//	}
//	defer coordinator.Close()
func NewChoreographyCoordinator(config *ChoreographyConfig) (*ChoreographyCoordinator, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set default values
	if config.MaxConcurrentHandlers <= 0 {
		config.MaxConcurrentHandlers = 100
	}
	if config.EventTimeout <= 0 {
		config.EventTimeout = 30 * time.Second
	}
	if config.HandlerRetryPolicy == nil {
		config.HandlerRetryPolicy = saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
	}

	ctx, cancel := context.WithCancel(context.Background())

	coordinator := &ChoreographyCoordinator{
		eventPublisher: config.EventPublisher,
		stateStorage:   config.StateStorage,
		handlers:       make(map[saga.SagaEventType][]ChoreographyEventHandler),
		handlersByID:   make(map[string]ChoreographyEventHandler),
		config:         config,
		semaphore:      make(chan struct{}, config.MaxConcurrentHandlers),
		metrics: &ChoreographyMetrics{
			StartTime:      time.Now(),
			LastUpdateTime: time.Now(),
		},
		running: false,
		closed:  false,
		ctx:     ctx,
		cancel:  cancel,
	}

	return coordinator, nil
}

// RegisterEventHandler registers an event handler for specific event types.
// Multiple handlers can be registered for the same event type, and they will
// be executed in priority order (higher priority first).
//
// Parameters:
//   - handler: The event handler to register.
//
// Returns:
//   - An error if the handler is already registered or if the coordinator is closed.
func (cc *ChoreographyCoordinator) RegisterEventHandler(handler ChoreographyEventHandler) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.closed {
		return ErrChoreographyCoordinatorClosed
	}

	handlerID := handler.GetHandlerID()
	if _, exists := cc.handlersByID[handlerID]; exists {
		return ErrEventHandlerAlreadyRegistered
	}

	// Register handler for all supported event types
	eventTypes := handler.GetSupportedEventTypes()
	for _, eventType := range eventTypes {
		cc.handlers[eventType] = append(cc.handlers[eventType], handler)
	}

	// Store handler by ID for efficient lookup
	cc.handlersByID[handlerID] = handler

	// Update metrics
	cc.metrics.mu.Lock()
	cc.metrics.TotalHandlers++
	cc.metrics.LastUpdateTime = time.Now()
	cc.metrics.mu.Unlock()

	return nil
}

// UnregisterEventHandler removes a previously registered event handler.
//
// Parameters:
//   - handlerID: The unique identifier of the handler to unregister.
//
// Returns:
//   - An error if the handler is not found or if the coordinator is closed.
func (cc *ChoreographyCoordinator) UnregisterEventHandler(handlerID string) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.closed {
		return ErrChoreographyCoordinatorClosed
	}

	handler, exists := cc.handlersByID[handlerID]
	if !exists {
		return ErrEventHandlerNotFound
	}

	// Remove handler from all event type mappings
	eventTypes := handler.GetSupportedEventTypes()
	for _, eventType := range eventTypes {
		handlers := cc.handlers[eventType]
		for i, h := range handlers {
			if h.GetHandlerID() == handlerID {
				// Remove handler from slice
				cc.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
		// Remove event type entry if no handlers remain
		if len(cc.handlers[eventType]) == 0 {
			delete(cc.handlers, eventType)
		}
	}

	// Remove handler from ID map
	delete(cc.handlersByID, handlerID)

	// Update metrics
	cc.metrics.mu.Lock()
	cc.metrics.TotalHandlers--
	cc.metrics.LastUpdateTime = time.Now()
	cc.metrics.mu.Unlock()

	return nil
}

// Start begins the choreography coordination by subscribing to Saga events
// and starting the event processing loop.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//
// Returns:
//   - An error if the coordinator is already running, closed, or subscription fails.
func (cc *ChoreographyCoordinator) Start(ctx context.Context) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.closed {
		return ErrChoreographyCoordinatorClosed
	}

	if cc.running {
		return errors.New("choreography coordinator is already running")
	}

	// Create event filter to receive all Saga events
	filter := &saga.EventTypeFilter{
		Types: []saga.SagaEventType{
			saga.EventSagaStarted,
			saga.EventSagaStepStarted,
			saga.EventSagaStepCompleted,
			saga.EventSagaStepFailed,
			saga.EventSagaCompleted,
			saga.EventSagaFailed,
			saga.EventSagaCancelled,
			saga.EventSagaTimedOut,
			saga.EventCompensationStarted,
			saga.EventCompensationStepStarted,
			saga.EventCompensationStepCompleted,
			saga.EventCompensationStepFailed,
			saga.EventCompensationCompleted,
			saga.EventCompensationFailed,
		},
	}

	// Subscribe to events
	subscription, err := cc.eventPublisher.Subscribe(filter, cc)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	cc.subscription = subscription
	cc.running = true

	// Update metrics
	cc.metrics.mu.Lock()
	cc.metrics.StartTime = time.Now()
	cc.metrics.LastUpdateTime = time.Now()
	cc.metrics.mu.Unlock()

	return nil
}

// Stop gracefully stops the choreography coordinator by unsubscribing from events
// and waiting for active event handlers to complete.
//
// Parameters:
//   - ctx: Context for timeout control.
//
// Returns:
//   - An error if stopping fails or times out.
func (cc *ChoreographyCoordinator) Stop(ctx context.Context) error {
	cc.mu.Lock()
	if !cc.running {
		cc.mu.Unlock()
		return nil
	}

	// Unsubscribe from events
	if cc.subscription != nil {
		if err := cc.eventPublisher.Unsubscribe(cc.subscription); err != nil {
			cc.mu.Unlock()
			return fmt.Errorf("failed to unsubscribe: %w", err)
		}
		cc.subscription = nil
	}

	cc.running = false
	cc.mu.Unlock()

	// Wait for active handlers to complete with timeout
	done := make(chan struct{})
	go func() {
		cc.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stop timed out: %w", ctx.Err())
	}
}

// IsRunning returns true if the coordinator is currently running.
func (cc *ChoreographyCoordinator) IsRunning() bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.running
}

// Close gracefully shuts down the choreography coordinator, releasing all resources.
// It stops event processing, unsubscribes from events, and waits for active handlers
// to complete.
//
// Returns:
//   - An error if shutdown encounters issues.
func (cc *ChoreographyCoordinator) Close() error {
	cc.mu.Lock()
	if cc.closed {
		cc.mu.Unlock()
		return ErrChoreographyCoordinatorClosed
	}

	cc.closed = true
	cc.mu.Unlock()

	// Stop the coordinator with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := cc.Stop(ctx); err != nil {
		return err
	}

	// Cancel the coordinator context
	cc.cancel()

	// Update metrics
	cc.metrics.mu.Lock()
	cc.metrics.LastUpdateTime = time.Now()
	cc.metrics.mu.Unlock()

	return nil
}

// HandleEvent implements the saga.EventHandler interface to receive events
// from the event publisher subscription.
func (cc *ChoreographyCoordinator) HandleEvent(ctx context.Context, event *saga.SagaEvent) error {
	cc.mu.RLock()
	if cc.closed {
		cc.mu.RUnlock()
		return ErrChoreographyCoordinatorClosed
	}
	cc.mu.RUnlock()

	// Update metrics
	cc.metrics.mu.Lock()
	cc.metrics.TotalEvents++
	cc.metrics.mu.Unlock()

	// Get handlers for this event type
	cc.mu.RLock()
	handlers, exists := cc.handlers[event.Type]
	cc.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		// No handlers registered for this event type - not an error
		return nil
	}

	// Sort handlers by priority (higher priority first)
	sortedHandlers := make([]ChoreographyEventHandler, len(handlers))
	copy(sortedHandlers, handlers)
	cc.sortHandlersByPriority(sortedHandlers)

	// Process handlers concurrently with semaphore limiting
	errChan := make(chan error, len(sortedHandlers))
	startTime := time.Now()

	for _, handler := range sortedHandlers {
		handler := handler // capture loop variable

		// Acquire semaphore slot
		select {
		case cc.semaphore <- struct{}{}:
			// Got slot, continue
		case <-ctx.Done():
			return ctx.Err()
		case <-cc.ctx.Done():
			return ErrChoreographyCoordinatorClosed
		}

		// Process handler in goroutine
		cc.wg.Add(1)
		go func() {
			defer cc.wg.Done()
			defer func() { <-cc.semaphore }() // Release semaphore

			// Create timeout context for handler execution
			handlerCtx, cancel := context.WithTimeout(ctx, cc.config.EventTimeout)
			defer cancel()

			// Execute handler with retry
			err := cc.executeHandlerWithRetry(handlerCtx, handler, event)
			errChan <- err
		}()
	}

	// Wait for all handlers to complete
	var lastErr error
	for i := 0; i < len(sortedHandlers); i++ {
		if err := <-errChan; err != nil {
			lastErr = err
			// Update failure metrics
			cc.metrics.mu.Lock()
			cc.metrics.FailedEvents++
			cc.metrics.mu.Unlock()
		}
	}

	// Update success metrics
	if lastErr == nil {
		duration := time.Since(startTime)
		cc.metrics.mu.Lock()
		cc.metrics.ProcessedEvents++
		// Update average processing time (simple moving average)
		if cc.metrics.AverageProcessingTime == 0 {
			cc.metrics.AverageProcessingTime = duration
		} else {
			cc.metrics.AverageProcessingTime = (cc.metrics.AverageProcessingTime + duration) / 2
		}
		cc.metrics.LastUpdateTime = time.Now()
		cc.metrics.mu.Unlock()
	}

	return lastErr
}

// executeHandlerWithRetry executes an event handler with retry logic.
func (cc *ChoreographyCoordinator) executeHandlerWithRetry(
	ctx context.Context,
	handler ChoreographyEventHandler,
	event *saga.SagaEvent,
) error {
	retryPolicy := cc.config.HandlerRetryPolicy
	maxAttempts := retryPolicy.GetMaxAttempts()

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Execute handler
		err := handler.HandleEvent(ctx, event)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry
		if !retryPolicy.ShouldRetry(err, attempt+1) {
			break
		}

		// Wait before retry
		delay := retryPolicy.GetRetryDelay(attempt)
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return ctx.Err()
		case <-cc.ctx.Done():
			return ErrChoreographyCoordinatorClosed
		}
	}

	return lastErr
}

// sortHandlersByPriority sorts handlers by priority (higher priority first).
func (cc *ChoreographyCoordinator) sortHandlersByPriority(handlers []ChoreographyEventHandler) {
	// Simple bubble sort - adequate for small number of handlers
	n := len(handlers)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if handlers[j].GetPriority() < handlers[j+1].GetPriority() {
				handlers[j], handlers[j+1] = handlers[j+1], handlers[j]
			}
		}
	}
}

// GetHandlerName implements saga.EventHandler interface.
func (cc *ChoreographyCoordinator) GetHandlerName() string {
	return "ChoreographyCoordinator"
}

// GetMetrics returns a snapshot of current coordinator metrics.
func (cc *ChoreographyCoordinator) GetMetrics() *ChoreographyMetrics {
	cc.metrics.mu.RLock()
	defer cc.metrics.mu.RUnlock()

	// Return a copy to prevent external mutation
	return &ChoreographyMetrics{
		TotalEvents:           cc.metrics.TotalEvents,
		ProcessedEvents:       cc.metrics.ProcessedEvents,
		FailedEvents:          cc.metrics.FailedEvents,
		ActiveHandlers:        cc.metrics.ActiveHandlers,
		TotalHandlers:         cc.metrics.TotalHandlers,
		AverageProcessingTime: cc.metrics.AverageProcessingTime,
		StartTime:             cc.metrics.StartTime,
		LastUpdateTime:        time.Now(),
	}
}

// GetRegisteredHandlers returns a list of all registered handler IDs.
func (cc *ChoreographyCoordinator) GetRegisteredHandlers() []string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	handlerIDs := make([]string, 0, len(cc.handlersByID))
	for id := range cc.handlersByID {
		handlerIDs = append(handlerIDs, id)
	}
	return handlerIDs
}

// GetHandlersByEventType returns the list of handlers registered for a specific event type.
func (cc *ChoreographyCoordinator) GetHandlersByEventType(eventType saga.SagaEventType) []string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	handlers, exists := cc.handlers[eventType]
	if !exists {
		return []string{}
	}

	handlerIDs := make([]string, len(handlers))
	for i, handler := range handlers {
		handlerIDs[i] = handler.GetHandlerID()
	}
	return handlerIDs
}

// PublishEvent publishes a Saga event to the message broker.
// This is a convenience method for services to publish events.
func (cc *ChoreographyCoordinator) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	if cc.closed {
		return ErrChoreographyCoordinatorClosed
	}

	// Add event ID if not present
	if event.ID == "" {
		event.ID = generateChoreographyEventID()
	}

	// Add timestamp if not present
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Publish event
	return cc.eventPublisher.PublishEvent(ctx, event)
}

// generateChoreographyEventID generates a unique identifier for an event.
func generateChoreographyEventID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("choreo-event-%016x", time.Now().UnixNano())
	}
	return fmt.Sprintf("choreo-event-%s", hex.EncodeToString(bytes))
}

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

// Package messaging provides Saga-specific event handling and messaging infrastructure.
// It extends the base EventHandler interface with Saga-specific functionality for
// distributed transaction coordination, event routing, and message processing.
package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// SagaEventHandler extends the base EventHandler interface with Saga-specific
// event processing capabilities. It provides methods for handling different types
// of Saga events including step completion, compensation, and failure events.
//
// Implementations of this interface should be thread-safe and handle concurrent
// event processing. The handler is responsible for deserializing messages,
// routing events to appropriate processors, and managing error recovery.
//
// Example usage:
//
//	handler := NewSagaEventHandler(config)
//	if err := handler.Initialize(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer handler.Shutdown(ctx)
//
//	// Register handler with messaging coordinator
//	coordinator.RegisterEventHandler(handler)
type SagaEventHandler interface {
	// HandleSagaEvent processes a Saga event with full context and metadata.
	// This is the primary method for processing Saga-specific events.
	//
	// Parameters:
	//   - ctx: Context for timeout and cancellation control
	//   - event: The Saga event to process
	//   - handlerCtx: Additional context information for event processing
	//
	// Returns:
	//   - error: nil if processing succeeds, or an error describing the failure
	//
	// The method should be idempotent where possible, as events may be
	// redelivered in case of failures or network issues.
	HandleSagaEvent(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) error

	// GetSupportedEventTypes returns the list of Saga event types this handler
	// can process. This is used for event routing and filtering.
	//
	// Returns:
	//   - []SagaEventType: List of supported event types
	GetSupportedEventTypes() []SagaEventType

	// CanHandle determines if this handler can process a specific event.
	// This provides more fine-grained control than GetSupportedEventTypes,
	// allowing handlers to inspect event metadata before accepting it.
	//
	// Parameters:
	//   - event: The event to check
	//
	// Returns:
	//   - bool: true if the handler can process this event
	CanHandle(event *saga.SagaEvent) bool

	// GetPriority returns the handler's priority for event processing.
	// Higher priority handlers are invoked first when multiple handlers
	// can process the same event type.
	//
	// Returns:
	//   - int: Priority value (higher is more important)
	GetPriority() int

	// OnEventProcessed is called after an event is successfully processed.
	// This allows handlers to perform cleanup or additional actions.
	//
	// Parameters:
	//   - ctx: Context for the callback
	//   - event: The processed event
	//   - result: Optional result data from event processing
	OnEventProcessed(ctx context.Context, event *saga.SagaEvent, result interface{})

	// OnEventFailed is called when event processing fails.
	// Handlers can use this to implement custom error handling or retry logic.
	//
	// Parameters:
	//   - ctx: Context for the callback
	//   - event: The failed event
	//   - err: The error that caused the failure
	//
	// Returns:
	//   - EventFailureAction: Action to take (retry, dead-letter, discard)
	OnEventFailed(ctx context.Context, event *saga.SagaEvent, err error) EventFailureAction

	// GetConfiguration returns the handler's configuration.
	//
	// Returns:
	//   - *HandlerConfig: Current configuration
	GetConfiguration() *HandlerConfig

	// UpdateConfiguration updates the handler's configuration at runtime.
	// Not all configuration changes may be applied immediately.
	//
	// Parameters:
	//   - config: New configuration to apply
	//
	// Returns:
	//   - error: nil if update succeeds, or an error
	UpdateConfiguration(config *HandlerConfig) error

	// GetMetrics returns metrics about the handler's performance.
	//
	// Returns:
	//   - *HandlerMetrics: Current metrics snapshot
	GetMetrics() *HandlerMetrics

	// HealthCheck verifies the handler is functioning correctly.
	//
	// Parameters:
	//   - ctx: Context for the health check
	//
	// Returns:
	//   - error: nil if healthy, or an error describing the issue
	HealthCheck(ctx context.Context) error
}

// EventHandlerContext provides contextual information for event processing.
// It contains metadata about the event source, processing state, and
// any additional context needed for proper event handling.
type EventHandlerContext struct {
	// MessageID is the unique identifier of the underlying message.
	MessageID string

	// Topic is the message broker topic the event was received from.
	Topic string

	// Partition identifies the topic partition (if applicable).
	Partition int

	// Offset is the message offset within the partition.
	Offset int64

	// Timestamp is when the message was received by the handler.
	Timestamp time.Time

	// Headers contains message headers from the broker.
	Headers map[string]string

	// RetryCount indicates how many times this event has been retried.
	RetryCount int

	// OriginalMessageData contains the raw message data before deserialization.
	OriginalMessageData []byte

	// BrokerName identifies which message broker the event came from.
	BrokerName string

	// ConsumerGroup is the consumer group that received this message.
	ConsumerGroup string

	// Metadata contains additional context-specific metadata.
	Metadata map[string]interface{}
}

// Note: SagaEventType is defined in pkg/saga/types.go and reused here.
// We use type alias to maintain compatibility while avoiding duplicate definitions.
type SagaEventType = saga.SagaEventType

// Reuse existing event type constants from pkg/saga/types.go to ensure
// consistency across the codebase and avoid event routing mismatches.
const (
	// Saga lifecycle events (from saga.EventSaga*)
	EventTypeSagaStarted       = saga.EventSagaStarted       // "saga.started"
	EventTypeSagaStepStarted   = saga.EventSagaStepStarted   // "saga.step.started"
	EventTypeSagaStepCompleted = saga.EventSagaStepCompleted // "saga.step.completed"
	EventTypeSagaStepFailed    = saga.EventSagaStepFailed    // "saga.step.failed"
	EventTypeSagaCompleted     = saga.EventSagaCompleted     // "saga.completed"
	EventTypeSagaFailed        = saga.EventSagaFailed        // "saga.failed"
	EventTypeSagaCancelled     = saga.EventSagaCancelled     // "saga.cancelled"
	EventTypeSagaTimedOut      = saga.EventSagaTimedOut      // "saga.timed_out"

	// Compensation events (from saga.EventCompensation*)
	EventTypeCompensationStarted       = saga.EventCompensationStarted       // "compensation.started"
	EventTypeCompensationStepStarted   = saga.EventCompensationStepStarted   // "compensation.step.started"
	EventTypeCompensationStepCompleted = saga.EventCompensationStepCompleted // "compensation.step.completed"
	EventTypeCompensationStepFailed    = saga.EventCompensationStepFailed    // "compensation.step.failed"
	EventTypeCompensationCompleted     = saga.EventCompensationCompleted     // "compensation.completed"
	EventTypeCompensationFailed        = saga.EventCompensationFailed        // "compensation.failed"

	// Retry events (from saga.EventRetry*)
	EventTypeRetryAttempted = saga.EventRetryAttempted // "retry.attempted"
	EventTypeRetryExhausted = saga.EventRetryExhausted // "retry.exhausted"

	// State change events (from saga.EventState*)
	EventTypeStateChanged = saga.EventStateChanged // "state.changed"
)

// IsStepEvent returns true if the event type is related to step execution.
func IsStepEvent(t SagaEventType) bool {
	switch t {
	case EventTypeSagaStepStarted, EventTypeSagaStepCompleted, EventTypeSagaStepFailed:
		return true
	default:
		return false
	}
}

// IsCompensationEvent returns true if the event type is related to compensation.
func IsCompensationEvent(t SagaEventType) bool {
	switch t {
	case EventTypeCompensationStarted, EventTypeCompensationStepStarted,
		EventTypeCompensationStepCompleted, EventTypeCompensationStepFailed,
		EventTypeCompensationCompleted, EventTypeCompensationFailed:
		return true
	default:
		return false
	}
}

// IsSagaLifecycleEvent returns true if the event type is a Saga lifecycle event.
func IsSagaLifecycleEvent(t SagaEventType) bool {
	switch t {
	case EventTypeSagaStarted, EventTypeSagaCompleted, EventTypeSagaFailed,
		EventTypeSagaCancelled, EventTypeSagaTimedOut:
		return true
	default:
		return false
	}
}

// IsRetryEvent returns true if the event type is related to retry operations.
func IsRetryEvent(t SagaEventType) bool {
	switch t {
	case EventTypeRetryAttempted, EventTypeRetryExhausted:
		return true
	default:
		return false
	}
}

// EventFailureAction defines the action to take when event processing fails.
type EventFailureAction string

const (
	// FailureActionRetry indicates the event should be retried.
	FailureActionRetry EventFailureAction = "retry"

	// FailureActionDeadLetter indicates the event should be sent to a dead-letter queue.
	FailureActionDeadLetter EventFailureAction = "dead_letter"

	// FailureActionDiscard indicates the event should be discarded (acknowledged).
	FailureActionDiscard EventFailureAction = "discard"

	// FailureActionRequeue indicates the event should be requeued for later processing.
	FailureActionRequeue EventFailureAction = "requeue"
)

// String returns the string representation of the EventFailureAction.
func (a EventFailureAction) String() string {
	return string(a)
}

// HandlerConfig contains configuration for a SagaEventHandler.
type HandlerConfig struct {
	// HandlerID is the unique identifier for this handler instance.
	HandlerID string

	// HandlerName is the human-readable name of the handler.
	HandlerName string

	// Topics is the list of message broker topics to consume from.
	Topics []string

	// BrokerRequirement specifies which broker this handler requires.
	// Empty string means any broker is acceptable.
	BrokerRequirement string

	// ConsumerGroup is the consumer group name for this handler.
	ConsumerGroup string

	// Concurrency is the maximum number of concurrent event processors.
	Concurrency int

	// BatchSize is the number of events to process in a batch.
	BatchSize int

	// ProcessingTimeout is the maximum time allowed for processing a single event.
	ProcessingTimeout time.Duration

	// RetryPolicy defines how failed events should be retried.
	RetryPolicy RetryPolicy

	// DeadLetterConfig configures dead-letter queue behavior.
	DeadLetterConfig *DeadLetterConfig

	// FilterConfig defines event filtering rules.
	FilterConfig *FilterConfig

	// EnableMetrics enables metrics collection for this handler.
	EnableMetrics bool

	// EnableTracing enables distributed tracing for event processing.
	EnableTracing bool

	// Metadata contains additional handler-specific configuration.
	Metadata map[string]interface{}
}

// Validate checks if the handler configuration is valid.
func (c *HandlerConfig) Validate() error {
	if c.HandlerID == "" {
		return ErrInvalidHandlerID
	}
	if c.HandlerName == "" {
		return ErrInvalidHandlerName
	}
	if len(c.Topics) == 0 {
		return ErrNoTopicsConfigured
	}
	if c.Concurrency <= 0 {
		c.Concurrency = 1
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 1
	}
	if c.ProcessingTimeout <= 0 {
		c.ProcessingTimeout = 30 * time.Second
	}
	return nil
}

// RetryPolicy defines the retry behavior for failed event processing.
type RetryPolicy struct {
	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int

	// InitialDelay is the delay before the first retry.
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration

	// BackoffMultiplier is the multiplier for exponential backoff.
	BackoffMultiplier float64

	// RetryableErrors defines which error types should be retried.
	RetryableErrors []string
}

// DeadLetterConfig configures dead-letter queue behavior.
type DeadLetterConfig struct {
	// Enabled indicates if dead-letter queue is enabled.
	Enabled bool

	// Topic is the dead-letter queue topic name.
	Topic string

	// MaxRetentionDays is how long to retain messages in the DLQ.
	MaxRetentionDays int

	// IncludeOriginalMessage includes the original message in DLQ.
	IncludeOriginalMessage bool

	// IncludeErrorDetails includes error details in DLQ.
	IncludeErrorDetails bool
}

// FilterConfig defines event filtering rules.
type FilterConfig struct {
	// IncludeEventTypes specifies which event types to process.
	// Empty means process all supported types.
	IncludeEventTypes []SagaEventType

	// ExcludeEventTypes specifies which event types to exclude.
	ExcludeEventTypes []SagaEventType

	// IncludeSagaIDs processes only events from specific Saga instances.
	// Empty means process events from all Sagas.
	IncludeSagaIDs []string

	// ExcludeSagaIDs excludes events from specific Saga instances.
	ExcludeSagaIDs []string

	// CustomFilter is a custom filtering function.
	CustomFilter func(event *saga.SagaEvent) bool
}

// ShouldProcess determines if an event should be processed based on filter rules.
func (f *FilterConfig) ShouldProcess(event *saga.SagaEvent) bool {
	if f == nil {
		return true
	}

	// Apply custom filter first
	if f.CustomFilter != nil && !f.CustomFilter(event) {
		return false
	}

	// Check event type filters
	if len(f.ExcludeEventTypes) > 0 {
		for _, excludedType := range f.ExcludeEventTypes {
			if SagaEventType(event.Type) == excludedType {
				return false
			}
		}
	}

	if len(f.IncludeEventTypes) > 0 {
		found := false
		for _, includedType := range f.IncludeEventTypes {
			if SagaEventType(event.Type) == includedType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check Saga ID filters
	if len(f.ExcludeSagaIDs) > 0 {
		for _, excludedID := range f.ExcludeSagaIDs {
			if event.SagaID == excludedID {
				return false
			}
		}
	}

	if len(f.IncludeSagaIDs) > 0 {
		found := false
		for _, includedID := range f.IncludeSagaIDs {
			if event.SagaID == includedID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// HandlerMetrics contains metrics about event handler performance.
type HandlerMetrics struct {
	// HandlerID is the handler identifier.
	HandlerID string

	// TotalEventsReceived is the total number of events received.
	TotalEventsReceived int64

	// TotalEventsProcessed is the total number of events successfully processed.
	TotalEventsProcessed int64

	// TotalEventsFailed is the total number of events that failed processing.
	TotalEventsFailed int64

	// TotalEventsFiltered is the number of events filtered out.
	TotalEventsFiltered int64

	// TotalEventsRetried is the number of events that were retried.
	TotalEventsRetried int64

	// TotalEventsDeadLettered is the number of events sent to dead-letter queue.
	TotalEventsDeadLettered int64

	// AverageProcessingTime is the average time to process an event.
	AverageProcessingTime time.Duration

	// MaxProcessingTime is the maximum time taken to process an event.
	MaxProcessingTime time.Duration

	// MinProcessingTime is the minimum time taken to process an event.
	MinProcessingTime time.Duration

	// EventTypeMetrics contains per-event-type metrics.
	EventTypeMetrics map[SagaEventType]*EventTypeMetrics

	// LastProcessedAt is the timestamp of the last processed event.
	LastProcessedAt time.Time

	// LastErrorAt is the timestamp of the last error.
	LastErrorAt time.Time

	// LastError is the last error message.
	LastError string

	// StartedAt is when the handler started.
	StartedAt time.Time

	// IsHealthy indicates if the handler is healthy.
	IsHealthy bool
}

// EventTypeMetrics contains metrics for a specific event type.
type EventTypeMetrics struct {
	// EventType is the event type.
	EventType SagaEventType

	// Count is the number of events of this type processed.
	Count int64

	// FailureCount is the number of failures for this type.
	FailureCount int64

	// AverageProcessingTime is the average processing time for this type.
	AverageProcessingTime time.Duration
}

// HandlerRegistry manages registration and lookup of Saga event handlers.
type HandlerRegistry interface {
	// RegisterHandler registers a Saga event handler.
	RegisterHandler(handler SagaEventHandler) error

	// UnregisterHandler removes a handler by ID.
	UnregisterHandler(handlerID string) error

	// GetHandler retrieves a handler by ID.
	GetHandler(handlerID string) (SagaEventHandler, error)

	// GetHandlersForEvent returns all handlers that can process an event.
	GetHandlersForEvent(event *saga.SagaEvent) []SagaEventHandler

	// GetHandlersForEventType returns all handlers for a specific event type.
	GetHandlersForEventType(eventType SagaEventType) []SagaEventHandler

	// GetAllHandlers returns all registered handlers.
	GetAllHandlers() []SagaEventHandler

	// GetHandlerMetrics returns aggregated metrics for all handlers.
	GetHandlerMetrics() map[string]*HandlerMetrics
}

// EventRouter routes incoming Saga events to appropriate handlers.
type EventRouter interface {
	// RouteEvent routes an event to one or more handlers.
	RouteEvent(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) error

	// GetRoutingStrategy returns the current routing strategy.
	GetRoutingStrategy() RoutingStrategy

	// SetRoutingStrategy updates the routing strategy.
	SetRoutingStrategy(strategy RoutingStrategy)
}

// RoutingStrategy defines how events are routed to handlers.
type RoutingStrategy string

const (
	// RoutingStrategyAll sends events to all matching handlers.
	RoutingStrategyAll RoutingStrategy = "all"

	// RoutingStrategyPriority sends events to the highest priority handler only.
	RoutingStrategyPriority RoutingStrategy = "priority"

	// RoutingStrategyRoundRobin distributes events across handlers in round-robin fashion.
	RoutingStrategyRoundRobin RoutingStrategy = "round_robin"

	// RoutingStrategyRandom randomly selects a handler for each event.
	RoutingStrategyRandom RoutingStrategy = "random"
)

// String returns the string representation of the RoutingStrategy.
func (s RoutingStrategy) String() string {
	return string(s)
}

// defaultSagaEventHandler is a concrete implementation of the SagaEventHandler interface.
// It provides event handling capabilities with support for filtering, routing, metrics collection,
// and lifecycle management.
//
// The handler follows these key responsibilities:
//  1. Consuming Saga events from message brokers
//  2. Filtering and routing events to appropriate processors
//  3. Managing event processing concurrency
//  4. Collecting metrics and maintaining health status
//  5. Handling failures with retry and dead-letter queue support
//  6. Coordinating with the Saga coordinator for state updates
//
// This implementation is thread-safe and designed for concurrent event processing.
type defaultSagaEventHandler struct {
	// config holds the handler configuration.
	config *HandlerConfig

	// coordinator is the Saga coordinator for processing events.
	coordinator saga.SagaCoordinator

	// eventPublisher publishes events (e.g., for DLQ).
	eventPublisher saga.EventPublisher

	// metrics tracks handler performance metrics.
	metrics *HandlerMetrics

	// started indicates whether the handler has been started.
	started bool

	// shutdown indicates whether the handler has been shut down.
	shutdown bool

	// mu protects the handler state.
	mu sync.RWMutex

	// stopCh is used to signal handler shutdown.
	stopCh chan struct{}

	// wg tracks active event processing goroutines.
	wg sync.WaitGroup
}

// HandlerOption is a functional option for configuring a SagaEventHandler.
type HandlerOption func(*defaultSagaEventHandler) error

// NewSagaEventHandler creates a new SagaEventHandler with the given configuration and options.
// It initializes the handler with default values and applies any provided options.
//
// The handler requires at minimum:
//   - A valid configuration (HandlerConfig)
//   - A Saga coordinator for event processing
//   - An event publisher (optional, but required for DLQ support)
//
// Example usage:
//
//	handler, err := NewSagaEventHandler(
//	    config,
//	    WithCoordinator(coordinator),
//	    WithEventPublisher(publisher),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Parameters:
//   - config: Handler configuration
//   - opts: Variadic list of functional options
//
// Returns:
//   - SagaEventHandler: Configured handler instance
//   - error: Configuration or validation error
func NewSagaEventHandler(config *HandlerConfig, opts ...HandlerOption) (SagaEventHandler, error) {
	if config == nil {
		return nil, ErrInvalidContext
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create handler with defaults
	handler := &defaultSagaEventHandler{
		config: config,
		metrics: &HandlerMetrics{
			HandlerID:        config.HandlerID,
			EventTypeMetrics: make(map[SagaEventType]*EventTypeMetrics),
			StartedAt:        time.Now(),
			IsHealthy:        true,
		},
		stopCh: make(chan struct{}),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(handler); err != nil {
			return nil, err
		}
	}

	// Validate required dependencies
	if handler.coordinator == nil {
		return nil, ErrHandlerNotInitialized
	}

	return handler, nil
}

// WithCoordinator sets the Saga coordinator for the handler.
func WithCoordinator(coordinator saga.SagaCoordinator) HandlerOption {
	return func(h *defaultSagaEventHandler) error {
		if coordinator == nil {
			return ErrInvalidContext
		}
		h.coordinator = coordinator
		return nil
	}
}

// WithEventPublisher sets the event publisher for the handler.
func WithEventPublisher(publisher saga.EventPublisher) HandlerOption {
	return func(h *defaultSagaEventHandler) error {
		if publisher == nil {
			return ErrInvalidContext
		}
		h.eventPublisher = publisher
		return nil
	}
}

// Start initializes and starts the handler, preparing it to process events.
// This method should be called before the handler begins receiving events.
//
// The start operation:
//  1. Marks the handler as started
//  2. Initializes internal state
//  3. Prepares worker pools (if configured)
//  4. Starts health monitoring
//
// Parameters:
//   - ctx: Context for initialization timeout control
//
// Returns:
//   - error: Initialization error or nil on success
func (h *defaultSagaEventHandler) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.shutdown {
		return ErrHandlerShutdown
	}

	if h.started {
		return ErrHandlerAlreadyInitialized
	}

	// Verify coordinator is available
	if err := h.coordinator.HealthCheck(ctx); err != nil {
		h.metrics.IsHealthy = false
		return fmt.Errorf("coordinator health check failed: %w", err)
	}

	h.started = true
	h.metrics.StartedAt = time.Now()
	h.metrics.IsHealthy = true

	return nil
}

// Stop gracefully shuts down the handler, waiting for active processing to complete.
// This method should be called when the handler is no longer needed or during application shutdown.
//
// The stop operation:
//  1. Signals workers to stop accepting new events
//  2. Waits for active event processing to complete
//  3. Releases resources and connections
//  4. Updates metrics and health status
//
// Parameters:
//   - ctx: Context for shutdown timeout control
//
// Returns:
//   - error: Shutdown error or nil on success
func (h *defaultSagaEventHandler) Stop(ctx context.Context) error {
	h.mu.Lock()
	if h.shutdown {
		h.mu.Unlock()
		return ErrHandlerShutdown
	}
	if !h.started {
		h.mu.Unlock()
		return ErrHandlerNotInitialized
	}

	h.shutdown = true
	close(h.stopCh)
	h.mu.Unlock()

	// Wait for active processing to complete with timeout
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		h.mu.Lock()
		h.metrics.IsHealthy = false
		h.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// HandleSagaEvent processes a Saga event with full context and metadata.
// This is the primary method for processing Saga-specific events.
//
// The processing flow:
//  1. Validate event and context
//  2. Apply filters to determine if event should be processed
//  3. Update metrics
//  4. Delegate to coordinator for business logic
//  5. Handle errors with retry/DLQ logic
//  6. Call lifecycle callbacks
//
// Parameters:
//   - ctx: Context for timeout and cancellation control
//   - event: The Saga event to process
//   - handlerCtx: Additional context information for event processing
//
// Returns:
//   - error: nil if processing succeeds, or an error describing the failure
func (h *defaultSagaEventHandler) HandleSagaEvent(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) error {
	h.mu.RLock()
	if h.shutdown {
		h.mu.RUnlock()
		return ErrHandlerShutdown
	}
	if !h.started {
		h.mu.RUnlock()
		return ErrHandlerNotInitialized
	}
	h.mu.RUnlock()

	// Validate inputs
	if event == nil {
		return ErrInvalidEvent
	}
	if handlerCtx == nil {
		return ErrInvalidContext
	}

	// Track active processing
	h.wg.Add(1)
	defer h.wg.Done()

	// Update metrics
	h.mu.Lock()
	h.metrics.TotalEventsReceived++
	h.mu.Unlock()

	// Apply filters
	if h.config.FilterConfig != nil && !h.config.FilterConfig.ShouldProcess(event) {
		h.mu.Lock()
		h.metrics.TotalEventsFiltered++
		h.mu.Unlock()
		return ErrEventFilteredOut
	}

	// Check if we can handle this event
	if !h.CanHandle(event) {
		return ErrInvalidEventType
	}

	// Apply processing timeout
	if h.config.ProcessingTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.config.ProcessingTimeout)
		defer cancel()
	}

	// Record start time for metrics
	startTime := time.Now()

	// Process the event (delegate to coordinator or custom logic)
	// For now, we just validate the event structure
	err := h.processEvent(ctx, event, handlerCtx)

	// Calculate processing time
	processingTime := time.Since(startTime)

	// Update metrics
	h.updateMetrics(event, processingTime, err)

	// Handle processing result
	if err != nil {
		// Call failure callback
		action := h.OnEventFailed(ctx, event, err)
		return h.handleFailureAction(ctx, event, handlerCtx, action, err)
	}

	// Call success callback
	h.OnEventProcessed(ctx, event, nil)

	return nil
}

// processEvent handles the actual event processing logic.
// This is a placeholder that can be extended with actual business logic.
func (h *defaultSagaEventHandler) processEvent(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) error {
	// Basic validation
	if event.Type == "" {
		return ErrInvalidEventType
	}
	if event.SagaID == "" {
		return ErrInvalidEvent
	}

	// For now, just validate the event structure
	// In a full implementation, this would delegate to the coordinator
	// or execute custom event processing logic

	return nil
}

// handleFailureAction executes the specified failure action.
func (h *defaultSagaEventHandler) handleFailureAction(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext, action EventFailureAction, originalErr error) error {
	switch action {
	case FailureActionRetry:
		// In a full implementation, this would re-enqueue the event
		h.mu.Lock()
		h.metrics.TotalEventsRetried++
		h.mu.Unlock()
		return originalErr

	case FailureActionDeadLetter:
		// Send to dead-letter queue if configured
		if h.config.DeadLetterConfig != nil && h.config.DeadLetterConfig.Enabled {
			if err := h.sendToDeadLetterQueue(ctx, event, handlerCtx, originalErr); err != nil {
				return fmt.Errorf("failed to send to DLQ: %w (original error: %v)", err, originalErr)
			}
			h.mu.Lock()
			h.metrics.TotalEventsDeadLettered++
			h.mu.Unlock()
			return nil // Event is handled by DLQ
		}
		return ErrDeadLetterQueueDisabled

	case FailureActionDiscard:
		// Discard the event (acknowledge without processing)
		return nil

	case FailureActionRequeue:
		// Requeue for later processing
		h.mu.Lock()
		h.metrics.TotalEventsRetried++
		h.mu.Unlock()
		return originalErr

	default:
		return originalErr
	}
}

// sendToDeadLetterQueue sends a failed event to the dead-letter queue.
func (h *defaultSagaEventHandler) sendToDeadLetterQueue(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext, err error) error {
	if h.eventPublisher == nil {
		return ErrDeadLetterPublishFailed
	}

	// In a full implementation, this would construct a proper DLQ message
	// and publish it using the event publisher
	// For now, we just return success
	return nil
}

// updateMetrics updates handler metrics based on event processing results.
func (h *defaultSagaEventHandler) updateMetrics(event *saga.SagaEvent, processingTime time.Duration, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	eventType := SagaEventType(event.Type)

	// Update per-event-type metrics
	typeMetrics, ok := h.metrics.EventTypeMetrics[eventType]
	if !ok {
		typeMetrics = &EventTypeMetrics{
			EventType: eventType,
		}
		h.metrics.EventTypeMetrics[eventType] = typeMetrics
	}

	typeMetrics.Count++
	if err != nil {
		typeMetrics.FailureCount++
		h.metrics.TotalEventsFailed++
		h.metrics.LastError = err.Error()
		h.metrics.LastErrorAt = time.Now()
	} else {
		h.metrics.TotalEventsProcessed++
		h.metrics.LastProcessedAt = time.Now()
	}

	// Update processing time metrics
	if h.metrics.MinProcessingTime == 0 || processingTime < h.metrics.MinProcessingTime {
		h.metrics.MinProcessingTime = processingTime
	}
	if processingTime > h.metrics.MaxProcessingTime {
		h.metrics.MaxProcessingTime = processingTime
	}

	// Calculate average processing time
	totalProcessed := h.metrics.TotalEventsProcessed + h.metrics.TotalEventsFailed
	if totalProcessed > 0 {
		currentAvg := h.metrics.AverageProcessingTime
		h.metrics.AverageProcessingTime = (currentAvg*time.Duration(totalProcessed-1) + processingTime) / time.Duration(totalProcessed)
	}

	// Update per-type average
	if typeMetrics.Count > 0 {
		currentTypeAvg := typeMetrics.AverageProcessingTime
		typeMetrics.AverageProcessingTime = (currentTypeAvg*time.Duration(typeMetrics.Count-1) + processingTime) / time.Duration(typeMetrics.Count)
	}
}

// GetSupportedEventTypes returns the list of Saga event types this handler can process.
func (h *defaultSagaEventHandler) GetSupportedEventTypes() []SagaEventType {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// If filter config specifies included types, return those
	if h.config.FilterConfig != nil && len(h.config.FilterConfig.IncludeEventTypes) > 0 {
		return h.config.FilterConfig.IncludeEventTypes
	}

	// Otherwise, return all event types except excluded ones
	allTypes := []SagaEventType{
		EventTypeSagaStarted,
		EventTypeSagaStepStarted,
		EventTypeSagaStepCompleted,
		EventTypeSagaStepFailed,
		EventTypeSagaCompleted,
		EventTypeSagaFailed,
		EventTypeSagaCancelled,
		EventTypeSagaTimedOut,
		EventTypeCompensationStarted,
		EventTypeCompensationStepStarted,
		EventTypeCompensationStepCompleted,
		EventTypeCompensationStepFailed,
		EventTypeCompensationCompleted,
		EventTypeCompensationFailed,
		EventTypeRetryAttempted,
		EventTypeRetryExhausted,
		EventTypeStateChanged,
	}

	if h.config.FilterConfig == nil || len(h.config.FilterConfig.ExcludeEventTypes) == 0 {
		return allTypes
	}

	// Filter out excluded types
	result := make([]SagaEventType, 0, len(allTypes))
	for _, t := range allTypes {
		excluded := false
		for _, excl := range h.config.FilterConfig.ExcludeEventTypes {
			if t == excl {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, t)
		}
	}

	return result
}

// CanHandle determines if this handler can process a specific event.
func (h *defaultSagaEventHandler) CanHandle(event *saga.SagaEvent) bool {
	if event == nil {
		return false
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	// Check filter config
	if h.config.FilterConfig != nil && !h.config.FilterConfig.ShouldProcess(event) {
		return false
	}

	// Check if event type is supported
	supportedTypes := h.GetSupportedEventTypes()
	eventType := SagaEventType(event.Type)
	for _, t := range supportedTypes {
		if t == eventType {
			return true
		}
	}

	return false
}

// GetPriority returns the handler's priority for event processing.
func (h *defaultSagaEventHandler) GetPriority() int {
	// Default priority is 0
	// This can be made configurable through HandlerConfig if needed
	return 0
}

// OnEventProcessed is called after an event is successfully processed.
func (h *defaultSagaEventHandler) OnEventProcessed(ctx context.Context, event *saga.SagaEvent, result interface{}) {
	// This is a callback hook that can be extended for custom behavior
	// For now, it's a no-op
}

// OnEventFailed is called when event processing fails.
func (h *defaultSagaEventHandler) OnEventFailed(ctx context.Context, event *saga.SagaEvent, err error) EventFailureAction {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Determine the appropriate action based on configuration and error type
	if h.config.DeadLetterConfig != nil && h.config.DeadLetterConfig.Enabled {
		// If DLQ is enabled and we've exceeded retries, send to DLQ
		if h.config.RetryPolicy.MaxRetries > 0 {
			// Check if we've exceeded retries (simplified check)
			return FailureActionDeadLetter
		}
	}

	// Default to retry if retry policy is configured
	if h.config.RetryPolicy.MaxRetries > 0 {
		return FailureActionRetry
	}

	// Otherwise, discard the event
	return FailureActionDiscard
}

// GetConfiguration returns the handler's configuration.
func (h *defaultSagaEventHandler) GetConfiguration() *HandlerConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.config
}

// UpdateConfiguration updates the handler's configuration at runtime.
func (h *defaultSagaEventHandler) UpdateConfiguration(config *HandlerConfig) error {
	if config == nil {
		return ErrInvalidContext
	}

	if err := config.Validate(); err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.shutdown {
		return ErrHandlerShutdown
	}

	// Only update safe-to-change fields while running
	h.config.ProcessingTimeout = config.ProcessingTimeout
	h.config.EnableMetrics = config.EnableMetrics
	h.config.EnableTracing = config.EnableTracing

	// FilterConfig can be updated safely
	if config.FilterConfig != nil {
		h.config.FilterConfig = config.FilterConfig
	}

	return nil
}

// GetMetrics returns metrics about the handler's performance.
func (h *defaultSagaEventHandler) GetMetrics() *HandlerMetrics {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Create a copy to avoid race conditions
	metricsCopy := *h.metrics

	// Deep copy event type metrics
	metricsCopy.EventTypeMetrics = make(map[SagaEventType]*EventTypeMetrics)
	for k, v := range h.metrics.EventTypeMetrics {
		typeCopy := *v
		metricsCopy.EventTypeMetrics[k] = &typeCopy
	}

	return &metricsCopy
}

// HealthCheck verifies the handler is functioning correctly.
func (h *defaultSagaEventHandler) HealthCheck(ctx context.Context) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.shutdown {
		return ErrHandlerShutdown
	}

	if !h.started {
		return ErrHandlerNotInitialized
	}

	// Check coordinator health
	if err := h.coordinator.HealthCheck(ctx); err != nil {
		return fmt.Errorf("coordinator health check failed: %w", err)
	}

	// Check if handler is processing events normally
	if !h.metrics.IsHealthy {
		return fmt.Errorf("handler is unhealthy: %s", h.metrics.LastError)
	}

	return nil
}

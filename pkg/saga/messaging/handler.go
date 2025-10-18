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

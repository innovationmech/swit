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

package saga

import (
	"math"
	"math/rand"
	"time"
)

// SagaState represents the overall state of a Saga instance.
type SagaState int

const (
	// StatePending indicates the Saga is created but not yet started.
	StatePending SagaState = iota

	// StateRunning indicates the Saga is currently executing.
	StateRunning

	// StateStepCompleted indicates a step has completed and the Saga is moving to the next step.
	StateStepCompleted

	// StateCompleted indicates the Saga has completed successfully.
	StateCompleted

	// StateCompensating indicates the Saga is executing compensation operations.
	StateCompensating

	// StateCompensated indicates all compensation operations have completed.
	StateCompensated

	// StateFailed indicates the Saga has failed due to an error.
	StateFailed

	// StateCancelled indicates the Saga was cancelled by a user request.
	StateCancelled

	// StateTimedOut indicates the Saga exceeded its timeout limit.
	StateTimedOut
)

// String returns the string representation of the SagaState.
func (s SagaState) String() string {
	switch s {
	case StatePending:
		return "pending"
	case StateRunning:
		return "running"
	case StateStepCompleted:
		return "step_completed"
	case StateCompleted:
		return "completed"
	case StateCompensating:
		return "compensating"
	case StateCompensated:
		return "compensated"
	case StateFailed:
		return "failed"
	case StateCancelled:
		return "cancelled"
	case StateTimedOut:
		return "timed_out"
	default:
		return "unknown"
	}
}

// IsTerminal returns true if the state is a terminal state (no further execution possible).
func (s SagaState) IsTerminal() bool {
	return s == StateCompleted || s == StateCompensated || s == StateFailed || s == StateCancelled || s == StateTimedOut
}

// IsActive returns true if the Saga is currently active (running or processing).
func (s SagaState) IsActive() bool {
	return s == StateRunning || s == StateStepCompleted || s == StateCompensating
}

// StepStateEnum represents the execution state of an individual step.
type StepStateEnum int

const (
	// StepStatePending indicates the step is waiting to be executed.
	StepStatePending StepStateEnum = iota

	// StepStateRunning indicates the step is currently executing.
	StepStateRunning

	// StepStateCompleted indicates the step has completed successfully.
	StepStateCompleted

	// StepStateFailed indicates the step has failed and may be retried.
	StepStateFailed

	// StepStateCompensating indicates the step's compensation is currently executing.
	StepStateCompensating

	// StepStateCompensated indicates the step's compensation has completed.
	StepStateCompensated

	// StepStateSkipped indicates the step was skipped (e.g., due to conditional logic).
	StepStateSkipped
)

// String returns the string representation of the StepStateEnum.
func (s StepStateEnum) String() string {
	switch s {
	case StepStatePending:
		return "pending"
	case StepStateRunning:
		return "running"
	case StepStateCompleted:
		return "completed"
	case StepStateFailed:
		return "failed"
	case StepStateCompensating:
		return "compensating"
	case StepStateCompensated:
		return "compensated"
	case StepStateSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// CompensationStateEnum represents the state of a compensation operation.
type CompensationStateEnum int

const (
	// CompensationStatePending indicates compensation is waiting to start.
	CompensationStatePending CompensationStateEnum = iota

	// CompensationStateRunning indicates compensation is currently executing.
	CompensationStateRunning

	// CompensationStateCompleted indicates compensation has completed successfully.
	CompensationStateCompleted

	// CompensationStateFailed indicates compensation has failed.
	CompensationStateFailed

	// CompensationStateSkipped indicates compensation was skipped.
	CompensationStateSkipped
)

// String returns the string representation of the CompensationStateEnum.
func (c CompensationStateEnum) String() string {
	switch c {
	case CompensationStatePending:
		return "pending"
	case CompensationStateRunning:
		return "running"
	case CompensationStateCompleted:
		return "completed"
	case CompensationStateFailed:
		return "failed"
	case CompensationStateSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// ErrorType represents the category of an error.
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeTimeout      ErrorType = "timeout"
	ErrorTypeNetwork      ErrorType = "network"
	ErrorTypeService      ErrorType = "service"
	ErrorTypeData         ErrorType = "data"
	ErrorTypeSystem       ErrorType = "system"
	ErrorTypeBusiness     ErrorType = "business"
	ErrorTypeCompensation ErrorType = "compensation"
)

// SagaEventType represents the type of a Saga event.
type SagaEventType string

const (
	// Saga lifecycle events
	EventSagaStarted       SagaEventType = "saga.started"
	EventSagaStepStarted   SagaEventType = "saga.step.started"
	EventSagaStepCompleted SagaEventType = "saga.step.completed"
	EventSagaStepFailed    SagaEventType = "saga.step.failed"
	EventSagaCompleted     SagaEventType = "saga.completed"
	EventSagaFailed        SagaEventType = "saga.failed"
	EventSagaCancelled     SagaEventType = "saga.cancelled"
	EventSagaTimedOut      SagaEventType = "saga.timed_out"

	// Compensation events
	EventCompensationStarted       SagaEventType = "compensation.started"
	EventCompensationStepStarted   SagaEventType = "compensation.step.started"
	EventCompensationStepCompleted SagaEventType = "compensation.step.completed"
	EventCompensationStepFailed    SagaEventType = "compensation.step.failed"
	EventCompensationCompleted     SagaEventType = "compensation.completed"
	EventCompensationFailed        SagaEventType = "compensation.failed"

	// Retry events
	EventRetryAttempted SagaEventType = "retry.attempted"
	EventRetryExhausted SagaEventType = "retry.exhausted"

	// State change events
	EventStateChanged SagaEventType = "state.changed"

	// Dead-letter events
	EventDeadLettered SagaEventType = "dead.lettered"
)

// SagaInstanceData represents the complete data structure for a Saga instance.
type SagaInstanceData struct {
	// Basic information
	ID           string `json:"id"`
	DefinitionID string `json:"definition_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`

	// State information
	State       SagaState `json:"state"`
	CurrentStep int       `json:"current_step"`
	TotalSteps  int       `json:"total_steps"`

	// Timing information
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	TimedOutAt  *time.Time `json:"timed_out_at,omitempty"`

	// Data and error information
	InitialData interface{} `json:"initial_data,omitempty"`
	CurrentData interface{} `json:"current_data,omitempty"`
	ResultData  interface{} `json:"result_data,omitempty"`
	Error       *SagaError  `json:"error,omitempty"`

	// Configuration
	Timeout     time.Duration `json:"timeout"`
	RetryPolicy RetryPolicy   `json:"retry_policy"`

	// Step states
	StepStates []*StepState `json:"step_states,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Tracing information
	TraceID string `json:"trace_id,omitempty"`
	SpanID  string `json:"span_id,omitempty"`

	// Optimistic locking
	Version int `json:"version"`
}

// StepState represents the state of an individual step within a Saga.
type StepState struct {
	// Basic information
	ID        string `json:"id"`
	SagaID    string `json:"saga_id"`
	StepIndex int    `json:"step_index"`
	Name      string `json:"name"`

	// State information
	State       StepStateEnum `json:"state"`
	Attempts    int           `json:"attempts"`
	MaxAttempts int           `json:"max_attempts"`

	// Timing information
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`

	// Data and error information
	InputData  interface{} `json:"input_data,omitempty"`
	OutputData interface{} `json:"output_data,omitempty"`
	Error      *SagaError  `json:"error,omitempty"`

	// Compensation information
	CompensationState *CompensationState `json:"compensation_state,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CompensationState represents the state of a compensation operation.
type CompensationState struct {
	State       CompensationStateEnum `json:"state"`
	Attempts    int                   `json:"attempts"`
	MaxAttempts int                   `json:"max_attempts"`
	StartedAt   *time.Time            `json:"started_at,omitempty"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
	Error       *SagaError            `json:"error,omitempty"`
}

// SagaError represents an error that occurred during Saga execution.
type SagaError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Type       ErrorType              `json:"type"`
	Retryable  bool                   `json:"retryable"`
	Timestamp  time.Time              `json:"timestamp"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Cause      *SagaError             `json:"cause,omitempty"`
}

// SagaEvent represents an event that occurs during Saga execution.
type SagaEvent struct {
	// Basic identification
	ID      string        `json:"id"`
	SagaID  string        `json:"saga_id"`
	StepID  string        `json:"step_id,omitempty"`
	Type    SagaEventType `json:"type"`
	Version string        `json:"version"`

	// Timing information
	Timestamp     time.Time `json:"timestamp"`
	CorrelationID string    `json:"correlation_id,omitempty"`

	// Data content
	Data          interface{} `json:"data,omitempty"`
	PreviousState interface{} `json:"previous_state,omitempty"`
	NewState      interface{} `json:"new_state,omitempty"`

	// Error information
	Error *SagaError `json:"error,omitempty"`

	// Execution information
	Duration    time.Duration `json:"duration,omitempty"`
	Attempt     int           `json:"attempt,omitempty"`
	MaxAttempts int           `json:"max_attempts,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Source information
	Source         string `json:"source,omitempty"`
	Service        string `json:"service,omitempty"`
	ServiceVersion string `json:"service_version,omitempty"`

	// Tracing information
	TraceID      string `json:"trace_id,omitempty"`
	SpanID       string `json:"span_id,omitempty"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
}

// CoordinatorMetrics contains metrics about the coordinator's performance.
type CoordinatorMetrics struct {
	// Saga counts
	TotalSagas     int64 `json:"total_sagas"`
	ActiveSagas    int64 `json:"active_sagas"`
	CompletedSagas int64 `json:"completed_sagas"`
	FailedSagas    int64 `json:"failed_sagas"`
	CancelledSagas int64 `json:"cancelled_sagas"`
	TimedOutSagas  int64 `json:"timed_out_sagas"`

	// Step counts
	TotalSteps     int64 `json:"total_steps"`
	CompletedSteps int64 `json:"completed_steps"`
	FailedSteps    int64 `json:"failed_steps"`

	// Timing metrics
	AverageSagaDuration time.Duration `json:"average_saga_duration"`
	AverageStepDuration time.Duration `json:"average_step_duration"`

	// Retry metrics
	TotalRetries      int64 `json:"total_retries"`
	SuccessfulRetries int64 `json:"successful_retries"`

	// Compensation metrics
	TotalCompensations      int64 `json:"total_compensations"`
	SuccessfulCompensations int64 `json:"successful_compensations"`

	// System metrics
	StartTime      time.Time `json:"start_time"`
	LastUpdateTime time.Time `json:"last_update_time"`
}

// SagaFilter provides filtering options for querying Saga instances.
type SagaFilter struct {
	// State filter - include only Sagas in these states
	States []SagaState `json:"states,omitempty"`

	// Definition filter - include only Sagas with these definition IDs
	DefinitionIDs []string `json:"definition_ids,omitempty"`

	// Time range filter - include only Sagas created within this range
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`

	// Limit and offset for pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Sort order
	SortBy    string `json:"sort_by,omitempty"`    // e.g., "created_at", "updated_at", "state"
	SortOrder string `json:"sort_order,omitempty"` // "asc" or "desc"

	// Metadata filter - include only Sagas with matching metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// EventTypeFilter filters events based on their type.
type EventTypeFilter struct {
	// Types specifies which event types to include.
	Types []SagaEventType `json:"types,omitempty"`

	// ExcludedTypes specifies which event types to exclude.
	ExcludedTypes []SagaEventType `json:"excluded_types,omitempty"`
}

// Match determines if an event matches the event type filter criteria.
func (f *EventTypeFilter) Match(event *SagaEvent) bool {
	// If no types specified, include all events except excluded ones
	if len(f.Types) == 0 {
		for _, excludedType := range f.ExcludedTypes {
			if event.Type == excludedType {
				return false
			}
		}
		return true
	}

	// Check if event type is in the included types
	for _, includedType := range f.Types {
		if event.Type == includedType {
			// Also check it's not in excluded types
			for _, excludedType := range f.ExcludedTypes {
				if event.Type == excludedType {
					return false
				}
			}
			return true
		}
	}

	return false
}

// GetDescription returns a human-readable description of the event type filter.
func (f *EventTypeFilter) GetDescription() string {
	if len(f.Types) == 0 && len(f.ExcludedTypes) == 0 {
		return "all events"
	}

	desc := "events matching types: "
	for i, eventType := range f.Types {
		if i > 0 {
			desc += ", "
		}
		desc += string(eventType)
	}

	if len(f.ExcludedTypes) > 0 {
		desc += " (excluding: "
		for i, excludedType := range f.ExcludedTypes {
			if i > 0 {
				desc += ", "
			}
			desc += string(excludedType)
		}
		desc += ")"
	}

	return desc
}

// SagaIDFilter filters events based on their Saga ID.
type SagaIDFilter struct {
	// SagaIDs specifies which Saga IDs to include.
	SagaIDs []string `json:"saga_ids,omitempty"`

	// ExcludeSagaIDs specifies which Saga IDs to exclude.
	ExcludeSagaIDs []string `json:"exclude_saga_ids,omitempty"`
}

// Match determines if an event matches the Saga ID filter criteria.
func (f *SagaIDFilter) Match(event *SagaEvent) bool {
	// If no Saga IDs specified, include all events except excluded ones
	if len(f.SagaIDs) == 0 {
		for _, excludedID := range f.ExcludeSagaIDs {
			if event.SagaID == excludedID {
				return false
			}
		}
		return true
	}

	// Check if event Saga ID is in the included IDs
	for _, includedID := range f.SagaIDs {
		if event.SagaID == includedID {
			// Also check it's not in excluded IDs
			for _, excludedID := range f.ExcludeSagaIDs {
				if event.SagaID == excludedID {
					return false
				}
			}
			return true
		}
	}

	return false
}

// GetDescription returns a human-readable description of the Saga ID filter.
func (f *SagaIDFilter) GetDescription() string {
	if len(f.SagaIDs) == 0 && len(f.ExcludeSagaIDs) == 0 {
		return "all Sagas"
	}

	desc := "events for Sagas: "
	for i, sagaID := range f.SagaIDs {
		if i > 0 {
			desc += ", "
		}
		desc += sagaID
	}

	if len(f.ExcludeSagaIDs) > 0 {
		desc += " (excluding: "
		for i, excludedID := range f.ExcludeSagaIDs {
			if i > 0 {
				desc += ", "
			}
			desc += excludedID
		}
		desc += ")"
	}

	return desc
}

// CompositeFilter combines multiple filters with logical AND/OR operations.
type CompositeFilter struct {
	// Filters are the individual filters to combine.
	Filters []EventFilter `json:"filters,omitempty"`

	// Operation specifies how to combine the filters (AND or OR).
	Operation FilterOperation `json:"operation"`
}

// FilterOperation represents the logical operation for combining filters.
type FilterOperation string

const (
	// FilterOperationAND requires all filters to match.
	FilterOperationAND FilterOperation = "AND"

	// FilterOperationOR requires at least one filter to match.
	FilterOperationOR FilterOperation = "OR"
)

// Match determines if an event matches the composite filter criteria.
func (f *CompositeFilter) Match(event *SagaEvent) bool {
	if len(f.Filters) == 0 {
		return true
	}

	switch f.Operation {
	case FilterOperationAND:
		// All filters must match
		for _, filter := range f.Filters {
			if !filter.Match(event) {
				return false
			}
		}
		return true

	case FilterOperationOR:
		// At least one filter must match
		for _, filter := range f.Filters {
			if filter.Match(event) {
				return true
			}
		}
		return false

	default:
		// Default to AND operation
		for _, filter := range f.Filters {
			if !filter.Match(event) {
				return false
			}
		}
		return true
	}
}

// GetDescription returns a human-readable description of the composite filter.
func (f *CompositeFilter) GetDescription() string {
	if len(f.Filters) == 0 {
		return "all events"
	}

	operation := string(f.Operation)
	if operation == "" {
		operation = string(FilterOperationAND)
	}

	desc := "events matching ("
	for i, filter := range f.Filters {
		if i > 0 {
			desc += " " + operation + " "
		}
		desc += filter.GetDescription()
	}
	desc += ")"

	return desc
}

// BasicEventSubscription provides a basic implementation of EventSubscription.
type BasicEventSubscription struct {
	ID        string                 `json:"id"`
	Filter    EventFilter            `json:"filter"`
	Handler   EventHandler           `json:"-"`
	Active    bool                   `json:"active"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// GetID returns the unique identifier of this subscription.
func (s *BasicEventSubscription) GetID() string {
	return s.ID
}

// GetFilter returns the filter associated with this subscription.
func (s *BasicEventSubscription) GetFilter() EventFilter {
	return s.Filter
}

// GetHandler returns the event handler for this subscription.
func (s *BasicEventSubscription) GetHandler() EventHandler {
	return s.Handler
}

// IsActive returns true if the subscription is currently active.
func (s *BasicEventSubscription) IsActive() bool {
	return s.Active
}

// GetCreatedAt returns the time when this subscription was created.
func (s *BasicEventSubscription) GetCreatedAt() time.Time {
	return s.CreatedAt
}

// GetMetadata returns the metadata associated with this subscription.
func (s *BasicEventSubscription) GetMetadata() map[string]interface{} {
	return s.Metadata
}

// SetActive sets the active state of the subscription.
func (s *BasicEventSubscription) SetActive(active bool) {
	s.Active = active
}

// mathRand provides a random number function for jitter calculation.
func mathRand(float64) float64 {
	return rand.Float64()
}

// ==========================
// Strategy Implementations
// ==========================

// RetryPolicyType represents the type of retry policy.
type RetryPolicyType string

const (
	// RetryPolicyFixedDelay uses a fixed delay between retries.
	RetryPolicyFixedDelay RetryPolicyType = "fixed_delay"

	// RetryPolicyExponentialBackoff uses exponential backoff with jitter.
	RetryPolicyExponentialBackoff RetryPolicyType = "exponential_backoff"

	// RetryPolicyLinearBackoff uses linear backoff.
	RetryPolicyLinearBackoff RetryPolicyType = "linear_backoff"

	// RetryPolicyNoRetry disables retries.
	RetryPolicyNoRetry RetryPolicyType = "no_retry"
)

// FixedDelayRetryPolicy implements a retry policy with fixed delay between attempts.
type FixedDelayRetryPolicy struct {
	maxAttempts int
	delay       time.Duration
}

// NewFixedDelayRetryPolicy creates a new fixed delay retry policy.
func NewFixedDelayRetryPolicy(maxAttempts int, delay time.Duration) *FixedDelayRetryPolicy {
	return &FixedDelayRetryPolicy{
		maxAttempts: maxAttempts,
		delay:       delay,
	}
}

// ShouldRetry determines if an operation should be retried.
func (p *FixedDelayRetryPolicy) ShouldRetry(err error, attempt int) bool {
	return attempt < p.maxAttempts
}

// GetRetryDelay returns the fixed delay before the next retry.
func (p *FixedDelayRetryPolicy) GetRetryDelay(attempt int) time.Duration {
	return p.delay
}

// GetMaxAttempts returns the maximum number of retry attempts.
func (p *FixedDelayRetryPolicy) GetMaxAttempts() int {
	return p.maxAttempts
}

// ExponentialBackoffRetryPolicy implements exponential backoff with jitter.
type ExponentialBackoffRetryPolicy struct {
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
	multiplier  float64
	jitter      bool
	random      func(float64) float64 // For testing injection
}

// NewExponentialBackoffRetryPolicy creates a new exponential backoff retry policy.
func NewExponentialBackoffRetryPolicy(maxAttempts int, baseDelay, maxDelay time.Duration) *ExponentialBackoffRetryPolicy {
	return &ExponentialBackoffRetryPolicy{
		maxAttempts: maxAttempts,
		baseDelay:   baseDelay,
		maxDelay:    maxDelay,
		multiplier:  2.0,
		jitter:      true,
		random:      mathRand,
	}
}

// ShouldRetry determines if an operation should be retried.
func (p *ExponentialBackoffRetryPolicy) ShouldRetry(err error, attempt int) bool {
	return attempt < p.maxAttempts
}

// GetRetryDelay returns exponential backoff delay with optional jitter.
func (p *ExponentialBackoffRetryPolicy) GetRetryDelay(attempt int) time.Duration {
	delay := time.Duration(float64(p.baseDelay) * math.Pow(p.multiplier, float64(attempt)))

	// Cap at max delay
	if delay > p.maxDelay {
		delay = p.maxDelay
	}

	// Add jitter if enabled
	if p.jitter {
		// Add random jitter of ±25% of the delay
		jitterAmount := float64(delay) * 0.25 * (p.random(0)*2 - 1)
		delay = time.Duration(float64(delay) + jitterAmount)
	}

	return delay
}

// GetMaxAttempts returns the maximum number of retry attempts.
func (p *ExponentialBackoffRetryPolicy) GetMaxAttempts() int {
	return p.maxAttempts
}

// SetJitter enables or disables jitter for retry delays.
func (p *ExponentialBackoffRetryPolicy) SetJitter(enabled bool) {
	p.jitter = enabled
}

// SetMultiplier sets the backoff multiplier.
func (p *ExponentialBackoffRetryPolicy) SetMultiplier(multiplier float64) {
	p.multiplier = multiplier
}

// LinearBackoffRetryPolicy implements linear backoff retry policy.
type LinearBackoffRetryPolicy struct {
	maxAttempts int
	baseDelay   time.Duration
	increment   time.Duration
	maxDelay    time.Duration
}

// NewLinearBackoffRetryPolicy creates a new linear backoff retry policy.
func NewLinearBackoffRetryPolicy(maxAttempts int, baseDelay, increment, maxDelay time.Duration) *LinearBackoffRetryPolicy {
	return &LinearBackoffRetryPolicy{
		maxAttempts: maxAttempts,
		baseDelay:   baseDelay,
		increment:   increment,
		maxDelay:    maxDelay,
	}
}

// ShouldRetry determines if an operation should be retried.
func (p *LinearBackoffRetryPolicy) ShouldRetry(err error, attempt int) bool {
	return attempt < p.maxAttempts
}

// GetRetryDelay returns linear backoff delay.
func (p *LinearBackoffRetryPolicy) GetRetryDelay(attempt int) time.Duration {
	delay := p.baseDelay + time.Duration(attempt)*p.increment

	// Cap at max delay
	if delay > p.maxDelay {
		delay = p.maxDelay
	}

	return delay
}

// GetMaxAttempts returns the maximum number of retry attempts.
func (p *LinearBackoffRetryPolicy) GetMaxAttempts() int {
	return p.maxAttempts
}

// NoRetryPolicy implements a retry policy that never retries.
type NoRetryPolicy struct{}

// NewNoRetryPolicy creates a new no retry policy.
func NewNoRetryPolicy() *NoRetryPolicy {
	return &NoRetryPolicy{}
}

// ShouldRetry always returns false.
func (p *NoRetryPolicy) ShouldRetry(err error, attempt int) bool {
	return false
}

// GetRetryDelay returns zero duration.
func (p *NoRetryPolicy) GetRetryDelay(attempt int) time.Duration {
	return 0
}

// GetMaxAttempts returns 1 (no retries).
func (p *NoRetryPolicy) GetMaxAttempts() int {
	return 1
}

// CompensationStrategyType represents the type of compensation strategy.
type CompensationStrategyType string

const (
	// CompensationStrategySequential compensates steps in reverse order sequentially.
	CompensationStrategySequential CompensationStrategyType = "sequential"

	// CompensationStrategyParallel compensates all steps in parallel.
	CompensationStrategyParallel CompensationStrategyType = "parallel"

	// CompensationStrategyCustom allows custom compensation logic.
	CompensationStrategyCustom CompensationStrategyType = "custom"

	// CompensationStrategyBestEffort attempts to compensate all steps even if some fail.
	CompensationStrategyBestEffort CompensationStrategyType = "best_effort"
)

// SequentialCompensationStrategy compensates steps in reverse sequential order.
type SequentialCompensationStrategy struct {
	timeout time.Duration
}

// NewSequentialCompensationStrategy creates a new sequential compensation strategy.
func NewSequentialCompensationStrategy(timeout time.Duration) *SequentialCompensationStrategy {
	return &SequentialCompensationStrategy{
		timeout: timeout,
	}
}

// ShouldCompensate determines if compensation should be performed.
func (s *SequentialCompensationStrategy) ShouldCompensate(err *SagaError) bool {
	// Compensate for all non-cancellation errors
	return err != nil && err.Type != ErrorTypeBusiness
}

// GetCompensationOrder returns the steps in reverse order for sequential compensation.
func (s *SequentialCompensationStrategy) GetCompensationOrder(completedSteps []SagaStep) []SagaStep {
	// Return steps in reverse order
	reversed := make([]SagaStep, len(completedSteps))
	for i, step := range completedSteps {
		reversed[len(completedSteps)-1-i] = step
	}
	return reversed
}

// GetCompensationTimeout returns the timeout for compensation operations.
func (s *SequentialCompensationStrategy) GetCompensationTimeout() time.Duration {
	return s.timeout
}

// ParallelCompensationStrategy compensates all steps in parallel.
type ParallelCompensationStrategy struct {
	timeout time.Duration
}

// NewParallelCompensationStrategy creates a new parallel compensation strategy.
func NewParallelCompensationStrategy(timeout time.Duration) *ParallelCompensationStrategy {
	return &ParallelCompensationStrategy{
		timeout: timeout,
	}
}

// ShouldCompensate determines if compensation should be performed.
func (s *ParallelCompensationStrategy) ShouldCompensate(err *SagaError) bool {
	// Compensate for all non-cancellation errors
	return err != nil && err.Type != ErrorTypeBusiness
}

// GetCompensationOrder returns the steps in original order for parallel compensation.
func (s *ParallelCompensationStrategy) GetCompensationOrder(completedSteps []SagaStep) []SagaStep {
	// Return steps as-is for parallel execution
	steps := make([]SagaStep, len(completedSteps))
	copy(steps, completedSteps)
	return steps
}

// GetCompensationTimeout returns the timeout for compensation operations.
func (s *ParallelCompensationStrategy) GetCompensationTimeout() time.Duration {
	return s.timeout
}

// BestEffortCompensationStrategy attempts to compensate all steps even if some fail.
type BestEffortCompensationStrategy struct {
	timeout time.Duration
}

// NewBestEffortCompensationStrategy creates a new best effort compensation strategy.
func NewBestEffortCompensationStrategy(timeout time.Duration) *BestEffortCompensationStrategy {
	return &BestEffortCompensationStrategy{
		timeout: timeout,
	}
}

// ShouldCompensate determines if compensation should be performed.
func (s *BestEffortCompensationStrategy) ShouldCompensate(err *SagaError) bool {
	// Always attempt compensation
	return err != nil
}

// GetCompensationOrder returns the steps in reverse order for best effort compensation.
func (s *BestEffortCompensationStrategy) GetCompensationOrder(completedSteps []SagaStep) []SagaStep {
	// Return steps in reverse order
	reversed := make([]SagaStep, len(completedSteps))
	for i, step := range completedSteps {
		reversed[len(completedSteps)-1-i] = step
	}
	return reversed
}

// GetCompensationTimeout returns the timeout for compensation operations.
func (s *BestEffortCompensationStrategy) GetCompensationTimeout() time.Duration {
	return s.timeout
}

// CustomCompensationStrategy allows custom compensation logic.
type CustomCompensationStrategy struct {
	timeout              time.Duration
	shouldCompensateFunc func(err *SagaError) bool
	getOrderFunc         func(completedSteps []SagaStep) []SagaStep
}

// NewCustomCompensationStrategy creates a new custom compensation strategy.
func NewCustomCompensationStrategy(
	timeout time.Duration,
	shouldCompensateFunc func(err *SagaError) bool,
	getOrderFunc func(completedSteps []SagaStep) []SagaStep,
) *CustomCompensationStrategy {
	return &CustomCompensationStrategy{
		timeout:              timeout,
		shouldCompensateFunc: shouldCompensateFunc,
		getOrderFunc:         getOrderFunc,
	}
}

// ShouldCompensate determines if compensation should be performed using custom logic.
func (s *CustomCompensationStrategy) ShouldCompensate(err *SagaError) bool {
	if s.shouldCompensateFunc != nil {
		return s.shouldCompensateFunc(err)
	}
	return err != nil
}

// GetCompensationOrder returns the compensation order using custom logic.
func (s *CustomCompensationStrategy) GetCompensationOrder(completedSteps []SagaStep) []SagaStep {
	if s.getOrderFunc != nil {
		return s.getOrderFunc(completedSteps)
	}
	// Default to reverse order
	reversed := make([]SagaStep, len(completedSteps))
	for i, step := range completedSteps {
		reversed[len(completedSteps)-1-i] = step
	}
	return reversed
}

// GetCompensationTimeout returns the timeout for compensation operations.
func (s *CustomCompensationStrategy) GetCompensationTimeout() time.Duration {
	return s.timeout
}

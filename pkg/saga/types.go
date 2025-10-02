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

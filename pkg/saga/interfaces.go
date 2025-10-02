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

// Package saga provides distributed transaction support using the Saga pattern.
// It offers both orchestration and choreography-based coordination for complex
// multi-service workflows with built-in compensation and error handling.
package saga

import (
	"context"
	"time"
)

// SagaCoordinator is the core coordinator of the Saga system.
// It manages the lifecycle of Saga instances, coordinates step execution,
// handles compensation operations, and provides monitoring capabilities.
type SagaCoordinator interface {
	// StartSaga starts a new Saga instance with the given definition and initial data.
	// Returns the created Saga instance or an error if the Saga could not be started.
	StartSaga(ctx context.Context, definition SagaDefinition, initialData interface{}) (SagaInstance, error)

	// GetSagaInstance retrieves the current state of a Saga instance by its ID.
	// Returns the Saga instance or an error if not found.
	GetSagaInstance(sagaID string) (SagaInstance, error)

	// CancelSaga cancels a running Saga instance with the specified reason.
	// This triggers compensation operations for completed steps.
	CancelSaga(ctx context.Context, sagaID string, reason string) error

	// GetActiveSagas retrieves all currently active Saga instances based on the filter.
	// Returns a slice of active Saga instances or an error.
	GetActiveSagas(filter *SagaFilter) ([]SagaInstance, error)

	// GetMetrics returns runtime metrics about the coordinator's performance.
	// Includes statistics about active, completed, and failed Sagas.
	GetMetrics() *CoordinatorMetrics

	// HealthCheck performs a health check of the coordinator and its dependencies.
	// Returns an error if any component is unhealthy.
	HealthCheck(ctx context.Context) error

	// Close gracefully shuts down the coordinator, releasing all resources.
	// This method should be called when the coordinator is no longer needed.
	Close() error
}

// SagaInstance represents a running instance of a Saga definition.
// It maintains the current state, execution progress, and provides access to
// execution results and error information.
type SagaInstance interface {
	// GetID returns the unique identifier of this Saga instance.
	GetID() string

	// GetDefinitionID returns the identifier of the Saga definition this instance follows.
	GetDefinitionID() string

	// GetState returns the current state of the Saga instance.
	GetState() SagaState

	// GetCurrentStep returns the index of the currently executing step.
	// Returns -1 if no step is currently active.
	GetCurrentStep() int

	// GetStartTime returns the time when the Saga instance was created.
	GetStartTime() time.Time

	// GetEndTime returns the time when the Saga instance reached a terminal state.
	// Returns a zero time value if the Saga is still running.
	GetEndTime() time.Time

	// GetResult returns the final result data of the Saga execution.
	// Returns nil if the Saga has not completed successfully.
	GetResult() interface{}

	// GetError returns the error that caused the Saga to fail.
	// Returns nil if the Saga has not failed or completed successfully.
	GetError() *SagaError

	// GetTotalSteps returns the total number of steps in the Saga definition.
	GetTotalSteps() int

	// GetCompletedSteps returns the number of steps that have completed successfully.
	GetCompletedSteps() int

	// GetCreatedAt returns the creation time of the Saga instance.
	GetCreatedAt() time.Time

	// GetUpdatedAt returns the last update time of the Saga instance.
	GetUpdatedAt() time.Time

	// GetTimeout returns the timeout duration for this Saga instance.
	GetTimeout() time.Duration

	// GetMetadata returns the metadata associated with this Saga instance.
	GetMetadata() map[string]interface{}

	// GetTraceID returns the distributed tracing identifier for this Saga.
	GetTraceID() string

	// IsTerminal returns true if the Saga is in a terminal state (completed, failed, compensated, etc.).
	IsTerminal() bool

	// IsActive returns true if the Saga is currently active (running, executing steps, or compensating).
	IsActive() bool
}

// SagaDefinition defines the structure and behavior of a Saga.
// It contains all the steps to execute, retry policies, and compensation strategies.
type SagaDefinition interface {
	// GetID returns the unique identifier of the Saga definition.
	GetID() string

	// GetName returns the human-readable name of the Saga.
	GetName() string

	// GetDescription returns a description of what the Saga does.
	GetDescription() string

	// GetSteps returns all the execution steps in order.
	GetSteps() []SagaStep

	// GetTimeout returns the overall timeout for the Saga execution.
	GetTimeout() time.Duration

	// GetRetryPolicy returns the retry policy for failed steps.
	GetRetryPolicy() RetryPolicy

	// GetCompensationStrategy returns the compensation strategy for failed Sagas.
	GetCompensationStrategy() CompensationStrategy

	// Validate validates the definition for correctness and consistency.
	// Returns an error if the definition is invalid.
	Validate() error

	// GetMetadata returns the metadata associated with this Saga definition.
	GetMetadata() map[string]interface{}
}

// SagaStep defines a single execution step within a Saga.
// Each step has an execute action and a corresponding compensation action.
type SagaStep interface {
	// GetID returns the unique identifier of the step.
	GetID() string

	// GetName returns the human-readable name of the step.
	GetName() string

	// GetDescription returns a description of what the step does.
	GetDescription() string

	// Execute executes the main logic of the step with the provided data.
	// Returns the output data or an error if execution fails.
	Execute(ctx context.Context, data interface{}) (interface{}, error)

	// Compensate executes the compensation action to undo the step's effects.
	// Returns an error if compensation fails.
	Compensate(ctx context.Context, data interface{}) error

	// GetTimeout returns the timeout duration for this step execution.
	GetTimeout() time.Duration

	// GetRetryPolicy returns the retry policy specific to this step.
	GetRetryPolicy() RetryPolicy

	// IsRetryable determines if an error is retryable for this step.
	IsRetryable(err error) bool

	// GetMetadata returns the metadata associated with this step.
	GetMetadata() map[string]interface{}
}

// StateStorage defines the interface for persisting Saga state.
// Implementations can provide in-memory, Redis, database, or other storage backends.
type StateStorage interface {
	// SaveSaga persists a Saga instance to storage.
	SaveSaga(ctx context.Context, saga SagaInstance) error

	// GetSaga retrieves a Saga instance by its ID.
	GetSaga(ctx context.Context, sagaID string) (SagaInstance, error)

	// UpdateSagaState updates only the state of a Saga instance.
	UpdateSagaState(ctx context.Context, sagaID string, state SagaState, metadata map[string]interface{}) error

	// DeleteSaga removes a Saga instance from storage.
	DeleteSaga(ctx context.Context, sagaID string) error

	// GetActiveSagas retrieves all active Saga instances based on the filter.
	GetActiveSagas(ctx context.Context, filter *SagaFilter) ([]SagaInstance, error)

	// GetTimeoutSagas retrieves Saga instances that have timed out before the specified time.
	GetTimeoutSagas(ctx context.Context, before time.Time) ([]SagaInstance, error)

	// SaveStepState persists the state of a specific step within a Saga.
	SaveStepState(ctx context.Context, sagaID string, step *StepState) error

	// GetStepStates retrieves all step states for a Saga instance.
	GetStepStates(ctx context.Context, sagaID string) ([]*StepState, error)

	// CleanupExpiredSagas removes Saga instances that are older than the specified time.
	CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error
}

// EventPublisher defines the interface for publishing Saga-related events.
// This enables event-driven architecture and external monitoring.
type EventPublisher interface {
	// PublishEvent publishes a Saga event to the configured message broker.
	PublishEvent(ctx context.Context, event *SagaEvent) error

	// Close gracefully shuts down the event publisher.
	Close() error
}

// RetryPolicy defines the strategy for retrying failed operations.
type RetryPolicy interface {
	// ShouldRetry determines if an operation should be retried based on the error and attempt count.
	ShouldRetry(err error, attempt int) bool

	// GetRetryDelay returns the delay before the next retry attempt.
	GetRetryDelay(attempt int) time.Duration

	// GetMaxAttempts returns the maximum number of retry attempts.
	GetMaxAttempts() int
}

// CompensationStrategy defines how compensation is performed when a Saga fails.
type CompensationStrategy interface {
	// ShouldCompensate determines if compensation should be performed for the given error.
	ShouldCompensate(err *SagaError) bool

	// GetCompensationOrder returns the order in which steps should be compensated.
	GetCompensationOrder(completedSteps []SagaStep) []SagaStep

	// GetCompensationTimeout returns the timeout for compensation operations.
	GetCompensationTimeout() time.Duration
}

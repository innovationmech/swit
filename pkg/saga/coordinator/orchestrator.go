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

// Package coordinator provides the orchestration-based Saga coordinator implementation.
// It manages centralized control of Saga execution, step coordination, and compensation handling.
package coordinator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

var (
	// ErrCoordinatorClosed indicates the coordinator has been closed.
	ErrCoordinatorClosed = errors.New("coordinator is closed")

	// ErrInvalidDefinition indicates the Saga definition is invalid.
	ErrInvalidDefinition = errors.New("invalid Saga definition")

	// ErrInvalidInitialData indicates the initial data is invalid.
	ErrInvalidInitialData = errors.New("invalid initial data")

	// ErrStateStorageNotConfigured indicates StateStorage is not configured.
	ErrStateStorageNotConfigured = errors.New("state storage not configured")

	// ErrEventPublisherNotConfigured indicates EventPublisher is not configured.
	ErrEventPublisherNotConfigured = errors.New("event publisher not configured")

	// ErrRetryPolicyNotConfigured indicates RetryPolicy is not configured.
	ErrRetryPolicyNotConfigured = errors.New("retry policy not configured")

	// ErrMetricsCollectorNotConfigured indicates MetricsCollector is not configured.
	ErrMetricsCollectorNotConfigured = errors.New("metrics collector not configured")
)

// OrchestratorCoordinator implements the orchestration-based Saga coordination pattern.
// It provides centralized control over Saga execution, managing the lifecycle of Saga instances,
// coordinating step execution, handling compensation operations, and providing monitoring capabilities.
//
// The coordinator follows these key responsibilities:
//  1. Saga instance lifecycle management (create, start, stop, cancel)
//  2. Sequential step execution with retry support
//  3. Compensation logic execution on failure
//  4. State persistence and recovery
//  5. Event publishing for observability
//  6. Metrics collection for monitoring
//  7. Timeout detection and handling
//  8. Concurrency control for safe parallel execution
type OrchestratorCoordinator struct {
	// stateStorage persists Saga state for recovery and querying.
	stateStorage saga.StateStorage

	// eventPublisher publishes Saga lifecycle events for observability.
	eventPublisher saga.EventPublisher

	// retryPolicy defines the default retry strategy for failed steps.
	retryPolicy saga.RetryPolicy

	// metricsCollector collects runtime metrics for monitoring.
	metricsCollector MetricsCollector

	// instances tracks active Saga instances.
	instances sync.Map

	// metrics holds aggregated coordinator metrics.
	metrics *saga.CoordinatorMetrics

	// closed indicates if the coordinator has been shut down.
	closed bool

	// mu protects concurrent access to coordinator state.
	mu sync.RWMutex
}

// MetricsCollector defines the interface for collecting coordinator metrics.
// Implementations can send metrics to Prometheus, StatsD, or other monitoring systems.
type MetricsCollector interface {
	// RecordSagaStarted increments the count of started Sagas.
	RecordSagaStarted(definitionID string)

	// RecordSagaCompleted increments the count of completed Sagas.
	RecordSagaCompleted(definitionID string, duration time.Duration)

	// RecordSagaFailed increments the count of failed Sagas.
	RecordSagaFailed(definitionID string, errorType saga.ErrorType, duration time.Duration)

	// RecordSagaCancelled increments the count of cancelled Sagas.
	RecordSagaCancelled(definitionID string, duration time.Duration)

	// RecordSagaTimedOut increments the count of timed out Sagas.
	RecordSagaTimedOut(definitionID string, duration time.Duration)

	// RecordStepExecuted increments the count of executed steps.
	RecordStepExecuted(definitionID, stepID string, success bool, duration time.Duration)

	// RecordStepRetried increments the count of step retries.
	RecordStepRetried(definitionID, stepID string, attempt int)

	// RecordCompensationExecuted increments the count of compensation executions.
	RecordCompensationExecuted(definitionID, stepID string, success bool, duration time.Duration)
}

// OrchestratorConfig contains configuration options for the orchestrator coordinator.
type OrchestratorConfig struct {
	// StateStorage is required for persisting Saga state.
	StateStorage saga.StateStorage

	// EventPublisher is required for publishing Saga events.
	EventPublisher saga.EventPublisher

	// RetryPolicy defines the default retry strategy. If not provided, a default policy is used.
	RetryPolicy saga.RetryPolicy

	// MetricsCollector collects runtime metrics. If not provided, a no-op collector is used.
	MetricsCollector MetricsCollector
}

// Validate checks if the configuration is valid.
func (c *OrchestratorConfig) Validate() error {
	if c.StateStorage == nil {
		return ErrStateStorageNotConfigured
	}
	if c.EventPublisher == nil {
		return ErrEventPublisherNotConfigured
	}
	return nil
}

// NewOrchestratorCoordinator creates a new orchestration-based Saga coordinator.
// It initializes all dependencies and prepares the coordinator for managing Saga instances.
//
// Parameters:
//   - config: Configuration containing required dependencies (StateStorage, EventPublisher)
//     and optional components (RetryPolicy, MetricsCollector).
//
// Returns:
//   - A configured OrchestratorCoordinator instance ready to manage Sagas.
//   - An error if the configuration is invalid or initialization fails.
//
// Example:
//
//	config := &OrchestratorConfig{
//	    StateStorage: memoryStorage,
//	    EventPublisher: eventPublisher,
//	    RetryPolicy: saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
//	    MetricsCollector: prometheusCollector,
//	}
//	coordinator, err := NewOrchestratorCoordinator(config)
//	if err != nil {
//	    return err
//	}
//	defer coordinator.Close()
func NewOrchestratorCoordinator(config *OrchestratorConfig) (*OrchestratorCoordinator, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Use default retry policy if not provided
	retryPolicy := config.RetryPolicy
	if retryPolicy == nil {
		retryPolicy = saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
	}

	// Use no-op metrics collector if not provided
	metricsCollector := config.MetricsCollector
	if metricsCollector == nil {
		metricsCollector = &noOpMetricsCollector{}
	}

	return &OrchestratorCoordinator{
		stateStorage:     config.StateStorage,
		eventPublisher:   config.EventPublisher,
		retryPolicy:      retryPolicy,
		metricsCollector: metricsCollector,
		metrics: &saga.CoordinatorMetrics{
			StartTime:      time.Now(),
			LastUpdateTime: time.Now(),
		},
		closed: false,
	}, nil
}

// StartSaga starts a new Saga instance with the given definition and initial data.
// It validates the definition, creates a new instance, persists it, and begins execution.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - definition: The Saga definition containing steps and configuration.
//   - initialData: Initial data to pass to the first step.
//
// Returns:
//   - The created Saga instance if successful.
//   - An error if validation fails, persistence fails, or the coordinator is closed.
//
// The method performs the following steps:
//  1. Validates the coordinator state and Saga definition
//  2. Creates a new Saga instance with a unique ID
//  3. Persists the initial state to storage
//  4. Publishes a SagaStarted event
//  5. Initiates asynchronous step execution
func (oc *OrchestratorCoordinator) StartSaga(
	ctx context.Context,
	definition saga.SagaDefinition,
	initialData interface{},
) (saga.SagaInstance, error) {
	oc.mu.RLock()
	if oc.closed {
		oc.mu.RUnlock()
		return nil, ErrCoordinatorClosed
	}
	oc.mu.RUnlock()

	// Validate definition
	if definition == nil {
		return nil, ErrInvalidDefinition
	}
	if err := definition.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Saga definition: %w", err)
	}

	// Create Saga instance placeholder
	// TODO: Implementation will be completed in issue #507
	return nil, errors.New("not yet implemented: issue #507 will implement Saga instance creation")
}

// GetSagaInstance retrieves the current state of a Saga instance by its ID.
// It first checks the in-memory cache, then queries the state storage if necessary.
//
// Parameters:
//   - sagaID: The unique identifier of the Saga instance.
//
// Returns:
//   - The Saga instance if found.
//   - An error if the instance does not exist or the coordinator is closed.
func (oc *OrchestratorCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	if oc.closed {
		return nil, ErrCoordinatorClosed
	}

	// Check in-memory cache first
	if instance, ok := oc.instances.Load(sagaID); ok {
		return instance.(saga.SagaInstance), nil
	}

	// Query from state storage
	// TODO: Implementation will be completed in future issues
	return nil, saga.NewSagaNotFoundError(sagaID)
}

// CancelSaga cancels a running Saga instance with the specified reason.
// This triggers compensation operations for all completed steps.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaID: The unique identifier of the Saga instance to cancel.
//   - reason: Human-readable reason for cancellation.
//
// Returns:
//   - An error if the Saga is not found, already terminal, or cancellation fails.
func (oc *OrchestratorCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	oc.mu.RLock()
	if oc.closed {
		oc.mu.RUnlock()
		return ErrCoordinatorClosed
	}
	oc.mu.RUnlock()

	// TODO: Implementation will be completed in future issues
	return errors.New("not yet implemented: cancellation logic pending")
}

// GetActiveSagas retrieves all currently active Saga instances based on the filter.
// Active Sagas are those in Running, StepCompleted, or Compensating states.
//
// Parameters:
//   - filter: Optional filter to narrow results. Pass nil to get all active Sagas.
//
// Returns:
//   - A slice of active Saga instances.
//   - An error if the query fails or the coordinator is closed.
func (oc *OrchestratorCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	if oc.closed {
		return nil, ErrCoordinatorClosed
	}

	// TODO: Implementation will query from state storage with filter
	return []saga.SagaInstance{}, nil
}

// GetMetrics returns runtime metrics about the coordinator's performance.
// The metrics include counts of Sagas in various states, timing information,
// and retry/compensation statistics.
//
// Returns:
//   - A snapshot of current coordinator metrics.
func (oc *OrchestratorCoordinator) GetMetrics() *saga.CoordinatorMetrics {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	// Return a copy to prevent external mutation
	metricsCopy := *oc.metrics
	metricsCopy.LastUpdateTime = time.Now()
	return &metricsCopy
}

// HealthCheck performs a health check of the coordinator and its dependencies.
// It verifies that the state storage and event publisher are operational.
//
// Parameters:
//   - ctx: Context for timeout control.
//
// Returns:
//   - nil if all components are healthy.
//   - An error describing which component is unhealthy.
func (oc *OrchestratorCoordinator) HealthCheck(ctx context.Context) error {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	if oc.closed {
		return ErrCoordinatorClosed
	}

	// TODO: Implement actual health checks for dependencies
	// For now, just verify the coordinator is not closed
	return nil
}

// Close gracefully shuts down the coordinator, releasing all resources.
// It waits for in-flight operations to complete (with timeout), then closes
// the event publisher and cleans up internal state.
//
// Returns:
//   - An error if shutdown encounters issues.
func (oc *OrchestratorCoordinator) Close() error {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	if oc.closed {
		return ErrCoordinatorClosed
	}

	oc.closed = true

	// TODO: Wait for in-flight Sagas to complete or timeout
	// TODO: Close event publisher
	// TODO: Clean up resources

	return nil
}

// StopSaga gracefully stops a running Saga instance.
// Unlike CancelSaga, this allows the current step to complete before stopping.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaID: The unique identifier of the Saga instance to stop.
//
// Returns:
//   - An error if the Saga is not found, already terminal, or stopping fails.
func (oc *OrchestratorCoordinator) StopSaga(ctx context.Context, sagaID string) error {
	oc.mu.RLock()
	if oc.closed {
		oc.mu.RUnlock()
		return ErrCoordinatorClosed
	}
	oc.mu.RUnlock()

	// TODO: Implementation will be completed in future issues
	return errors.New("not yet implemented: stop logic pending")
}

// noOpMetricsCollector is a no-op implementation of MetricsCollector.
// It is used when no metrics collector is provided in the configuration.
type noOpMetricsCollector struct{}

func (n *noOpMetricsCollector) RecordSagaStarted(definitionID string)                           {}
func (n *noOpMetricsCollector) RecordSagaCompleted(definitionID string, duration time.Duration) {}
func (n *noOpMetricsCollector) RecordSagaFailed(definitionID string, errorType saga.ErrorType, duration time.Duration) {
}
func (n *noOpMetricsCollector) RecordSagaCancelled(definitionID string, duration time.Duration) {}
func (n *noOpMetricsCollector) RecordSagaTimedOut(definitionID string, duration time.Duration)  {}
func (n *noOpMetricsCollector) RecordStepExecuted(definitionID, stepID string, success bool, duration time.Duration) {
}
func (n *noOpMetricsCollector) RecordStepRetried(definitionID, stepID string, attempt int) {}
func (n *noOpMetricsCollector) RecordCompensationExecuted(definitionID, stepID string, success bool, duration time.Duration) {
}

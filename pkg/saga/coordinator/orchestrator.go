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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/saga"
	oteltrace "go.opentelemetry.io/otel/trace"
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

	// tracingManager manages distributed tracing with OpenTelemetry.
	tracingManager TracingManager

	// timeoutDetector detects and handles timeouts for Sagas and steps.
	timeoutDetector *TimeoutDetector

	// concurrencyController manages concurrent Saga execution limits.
	concurrencyController *ConcurrencyController

	// instances tracks active Saga instances.
	instances sync.Map

	// definitions tracks Saga definitions by Saga ID. The definition is
	// required to drive compensation and lifecycle operations (cancel/stop)
	// for active instances.
	definitions sync.Map

	// cancelFuncs tracks per-Saga execution cancellation functions, enabling
	// graceful interruption of in-flight step execution.
	cancelFuncs sync.Map

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

// TracingManager defines the interface for distributed tracing management.
// It provides methods for creating and managing tracing spans for Saga operations.
type TracingManager interface {
	// StartSpan creates a new span with the given operation name.
	// Returns the updated context containing the span and the span itself.
	StartSpan(ctx context.Context, operationName string, opts ...SpanOption) (context.Context, Span)

	// SpanFromContext retrieves the current span from the context.
	SpanFromContext(ctx context.Context) Span
}

// Span defines the interface for a tracing span.
// It provides methods for adding attributes, events, and status to spans.
type Span interface {
	// SetAttribute sets a single attribute on the span.
	SetAttribute(key string, value interface{})

	// AddEvent adds an event to the span.
	AddEvent(name string)

	// SetStatus sets the status of the span.
	SetStatus(code int, description string)

	// End ends the span.
	End()

	// RecordError records an error as an event on the span.
	RecordError(err error)
}

// SpanOption represents an option for creating spans.
type SpanOption interface{}

// noOpTracingManager is a no-op implementation of TracingManager.
type noOpTracingManager struct{}

func (n *noOpTracingManager) StartSpan(ctx context.Context, operationName string, opts ...SpanOption) (context.Context, Span) {
	return ctx, &noOpSpan{}
}

func (n *noOpTracingManager) SpanFromContext(ctx context.Context) Span {
	return &noOpSpan{}
}

// noOpSpan is a no-op implementation of Span.
type noOpSpan struct{}

func (n *noOpSpan) SetAttribute(key string, value interface{}) {}
func (n *noOpSpan) AddEvent(name string)                       {}
func (n *noOpSpan) SetStatus(code int, description string)     {}
func (n *noOpSpan) End()                                       {}
func (n *noOpSpan) RecordError(err error)                      {}

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

	// TracingManager manages distributed tracing. If not provided, a no-op manager is used.
	TracingManager TracingManager

	// ConcurrencyConfig defines concurrency limits and worker pool settings.
	// If not provided, default configuration is used.
	ConcurrencyConfig *ConcurrencyConfig
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

	// Use no-op tracing manager if not provided
	tracingManager := config.TracingManager
	if tracingManager == nil {
		tracingManager = &noOpTracingManager{}
	}

	// Use default concurrency config if not provided
	concurrencyConfig := config.ConcurrencyConfig
	if concurrencyConfig == nil {
		concurrencyConfig = DefaultConcurrencyConfig()
	}

	// Initialize concurrency controller
	concurrencyController, err := NewConcurrencyController(concurrencyConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create concurrency controller: %w", err)
	}

	coordinator := &OrchestratorCoordinator{
		stateStorage:          config.StateStorage,
		eventPublisher:        config.EventPublisher,
		retryPolicy:           retryPolicy,
		metricsCollector:      metricsCollector,
		tracingManager:        tracingManager,
		concurrencyController: concurrencyController,
		metrics: &saga.CoordinatorMetrics{
			StartTime:      time.Now(),
			LastUpdateTime: time.Now(),
		},
		closed: false,
	}

	// Initialize timeout detector
	coordinator.timeoutDetector = newTimeoutDetector(coordinator)

	return coordinator, nil
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
	// Start tracing span for Saga execution
	ctx, span := oc.tracingManager.StartSpan(ctx, "saga.start")
	defer span.End()

	oc.mu.RLock()
	if oc.closed {
		oc.mu.RUnlock()
		span.SetStatus(2, "coordinator closed") // status code 2 = error
		return nil, ErrCoordinatorClosed
	}
	oc.mu.RUnlock()

	// Validate definition
	if definition == nil {
		span.SetStatus(2, "invalid definition")
		return nil, ErrInvalidDefinition
	}

	// Set span attributes after validating definition is not nil
	span.SetAttribute("saga.definition_id", definition.GetID())
	span.SetAttribute("saga.definition_name", definition.GetName())
	if err := definition.Validate(); err != nil {
		span.SetStatus(2, fmt.Sprintf("invalid definition: %v", err))
		span.RecordError(err)
		return nil, fmt.Errorf("invalid Saga definition: %w", err)
	}

	// Validate initial data (basic check)
	if initialData == nil {
		span.SetStatus(2, "invalid initial data")
		return nil, ErrInvalidInitialData
	}

	// Generate unique Saga ID
	sagaID, err := generateSagaID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Saga ID: %w", err)
	}

	// Extract tracing information from context if available
	traceID := extractTraceID(ctx)
	spanID := extractSpanID(ctx)

	// Get retry policy (use definition's policy or coordinator's default)
	retryPolicy := definition.GetRetryPolicy()
	if retryPolicy == nil {
		retryPolicy = oc.retryPolicy
	}

	// Create Saga instance
	now := time.Now()
	instance := &OrchestratorSagaInstance{
		id:             sagaID,
		definitionID:   definition.GetID(),
		name:           definition.GetName(),
		description:    definition.GetDescription(),
		state:          saga.StatePending,
		currentStep:    -1, // No step is currently executing
		completedSteps: 0,
		totalSteps:     len(definition.GetSteps()),
		createdAt:      now,
		updatedAt:      now,
		startedAt:      &now,
		initialData:    initialData,
		currentData:    initialData,
		timeout:        definition.GetTimeout(),
		retryPolicy:    retryPolicy,
		metadata:       copyMetadata(definition.GetMetadata()),
		traceID:        traceID,
		spanID:         spanID,
	}

	// Persist initial state to storage
	if err := oc.stateStorage.SaveSaga(ctx, instance); err != nil {
		return nil, saga.NewStorageError("SaveSaga", err)
	}

	// Store instance in memory cache
	oc.instances.Store(sagaID, instance)

	// Update coordinator metrics
	oc.mu.Lock()
	oc.metrics.TotalSagas++
	oc.metrics.ActiveSagas++
	oc.metrics.LastUpdateTime = time.Now()
	oc.mu.Unlock()

	// Record metrics
	oc.metricsCollector.RecordSagaStarted(definition.GetID())

	// Publish Saga started event
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    sagaID,
		Type:      saga.EventSagaStarted,
		Version:   "1.0",
		Timestamp: now,
		Data:      initialData,
		NewState:  saga.StatePending,
		TraceID:   traceID,
		SpanID:    spanID,
		Source:    "OrchestratorCoordinator",
		Metadata:  instance.metadata,
	}

	if err := oc.eventPublisher.PublishEvent(ctx, event); err != nil {
		// Log error but don't fail the Saga creation
		// The Saga is already persisted, so we continue
		_ = saga.NewEventPublishError(saga.EventSagaStarted, err)
	}

	// Track the definition and a cancellable execution context so that the
	// lifecycle APIs (CancelSaga / StopSaga) can interrupt in-flight execution
	// and trigger compensation when required.
	oc.definitions.Store(sagaID, definition)
	execCtx, execCancel := context.WithCancel(context.Background())
	oc.cancelFuncs.Store(sagaID, execCancel)

	// Acquire concurrency slot before starting execution
	if err := oc.concurrencyController.AcquireSlot(ctx, sagaID); err != nil {
		// Failed to acquire slot, clean up tracking state and persist instance
		oc.releaseExecution(sagaID)
		_ = oc.stateStorage.SaveSaga(ctx, instance)
		return nil, fmt.Errorf("failed to acquire concurrency slot: %w", err)
	}

	// Start asynchronous step execution using worker pool
	execTask := func(taskCtx context.Context) error {
		defer oc.concurrencyController.ReleaseSlot(sagaID)

		// Execute steps
		executor := newStepExecutor(oc, instance, definition)
		return executor.executeSteps(taskCtx)
	}

	// Submit task to worker pool using the cancellable execution context.
	submitErr := oc.concurrencyController.SubmitTask(execCtx, execTask)

	// SubmitTask blocks until the worker reports completion, so the per-Saga
	// execution tracking can be released now. Releasing here (rather than in a
	// deferred cleanup inside the task) avoids cancelling the execution context
	// before the worker has reported its result.
	oc.releaseExecution(sagaID)

	if submitErr != nil {
		// Failed to submit task, release slot and return error
		oc.concurrencyController.ReleaseSlot(sagaID)
		span.SetStatus(2, fmt.Sprintf("failed to submit task: %v", submitErr))
		span.RecordError(submitErr)
		return nil, fmt.Errorf("failed to submit execution task: %w", submitErr)
	}

	// Mark span as successful
	span.SetAttribute("saga.id", sagaID)
	span.SetAttribute("saga.total_steps", instance.totalSteps)
	span.SetStatus(1, "saga started successfully") // status code 1 = ok
	span.AddEvent("saga_started")

	return instance, nil
}

// executeStepsAsync executes Saga steps asynchronously in a goroutine.
// This allows StartSaga to return immediately while steps execute in the background.
func (oc *OrchestratorCoordinator) executeStepsAsync(
	ctx context.Context,
	instance *OrchestratorSagaInstance,
	definition saga.SagaDefinition,
) {
	// Create step executor
	executor := newStepExecutor(oc, instance, definition)

	// Execute steps
	if err := executor.executeSteps(ctx); err != nil {
		// Steps failed, error already handled by executor
		// Nothing more to do here
		return
	}

	// Steps completed successfully, result already handled by executor
}

// generateSagaID generates a unique identifier for a Saga instance.
// It uses crypto/rand to generate a random hex string in UUID-like format.
func generateSagaID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("saga-%016x", time.Now().UnixNano()), nil
	}

	// Format as UUID-like string with "saga-" prefix
	return fmt.Sprintf("saga-%08x-%04x-%04x-%04x-%012x",
		bytes[0:4],
		bytes[4:6],
		bytes[6:8],
		bytes[8:10],
		bytes[10:16]), nil
}

// generateEventID generates a unique identifier for an event.
func generateEventID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("event-%016x", time.Now().UnixNano())
	}
	return fmt.Sprintf("event-%s", hex.EncodeToString(bytes))
}

// extractTraceID extracts the trace ID from the context using OpenTelemetry.
// It retrieves the span context from the active span and returns the trace ID as a hex string.
func extractTraceID(ctx context.Context) string {
	span := oteltrace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	spanContext := span.SpanContext()
	if !spanContext.IsValid() {
		return ""
	}
	return spanContext.TraceID().String()
}

// extractSpanID extracts the span ID from the context using OpenTelemetry.
// It retrieves the span context from the active span and returns the span ID as a hex string.
func extractSpanID(ctx context.Context) string {
	span := oteltrace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	spanContext := span.SpanContext()
	if !spanContext.IsValid() {
		return ""
	}
	return spanContext.SpanID().String()
}

// copyMetadata creates a deep copy of metadata map to prevent external mutation.
func copyMetadata(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		return make(map[string]interface{})
	}
	copy := make(map[string]interface{}, len(metadata))
	for k, v := range metadata {
		copy[k] = v
	}
	return copy
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
	closed := oc.closed
	oc.mu.RUnlock()

	if closed {
		return nil, ErrCoordinatorClosed
	}

	// Check in-memory cache first for the live runtime instance.
	if instance, ok := oc.instances.Load(sagaID); ok {
		return instance.(saga.SagaInstance), nil
	}

	// Fall back to the configured state storage. This allows querying Sagas
	// that are no longer cached in memory (for example after a restart).
	instance, err := oc.stateStorage.GetSaga(context.Background(), sagaID)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, saga.NewSagaNotFoundError(sagaID)
	}
	return instance, nil
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

	ctx, span := oc.tracingManager.StartSpan(ctx, "saga.cancel")
	defer span.End()
	span.SetAttribute("saga.id", sagaID)

	// Resolve the instance. In-memory instances are preferred because they
	// carry the live runtime state required to drive compensation.
	instance, inMemory := oc.loadInstance(ctx, sagaID)
	if instance == nil {
		err := saga.NewSagaNotFoundError(sagaID)
		span.RecordError(err)
		span.SetStatus(2, "saga not found")
		return err
	}

	if instance.GetState().IsTerminal() {
		err := saga.NewInvalidSagaStateError(instance.GetState(), saga.StateCancelled)
		span.RecordError(err)
		span.SetStatus(2, "saga already terminal")
		return err
	}

	startTime := instance.GetStartTime()

	// Signal any in-flight execution to stop before compensation begins.
	oc.cancelExecution(sagaID)

	// Trigger compensation for completed steps when the live instance and its
	// definition are available.
	if inMemory {
		if orchInstance, ok := instance.(*OrchestratorSagaInstance); ok {
			oc.compensateForCancellation(ctx, orchInstance)
		}
	}

	// Transition to the Cancelled terminal state and persist it.
	if orchInstance, ok := instance.(*OrchestratorSagaInstance); ok {
		now := time.Now()
		orchInstance.mu.Lock()
		orchInstance.state = saga.StateCancelled
		orchInstance.completedAt = &now
		orchInstance.updatedAt = now
		if orchInstance.sagaError == nil {
			orchInstance.sagaError = saga.NewSagaError(
				saga.ErrCodeInvalidSagaState, reason, saga.ErrorTypeBusiness, false)
		}
		orchInstance.mu.Unlock()

		if err := oc.stateStorage.SaveSaga(ctx, orchInstance); err != nil {
			span.RecordError(err)
			span.SetStatus(2, "failed to persist cancelled state")
			return saga.NewStorageError("SaveSaga", err)
		}
	} else {
		// Storage-only instance: update the persisted state directly.
		metadata := map[string]interface{}{"cancellation_reason": reason}
		if err := oc.stateStorage.UpdateSagaState(ctx, sagaID, saga.StateCancelled, metadata); err != nil {
			span.RecordError(err)
			span.SetStatus(2, "failed to persist cancelled state")
			return saga.NewStorageError("UpdateSagaState", err)
		}
	}

	// Publish the cancellation event for observability.
	oc.publishCancellationEvent(ctx, instance, reason, time.Since(startTime))

	// Update aggregate metrics.
	oc.mu.Lock()
	oc.metrics.CancelledSagas++
	if oc.metrics.ActiveSagas > 0 {
		oc.metrics.ActiveSagas--
	}
	oc.metrics.LastUpdateTime = time.Now()
	oc.mu.Unlock()

	oc.metricsCollector.RecordSagaCancelled(instance.GetDefinitionID(), time.Since(startTime))

	// Clean up tracking state.
	oc.instances.Delete(sagaID)
	oc.releaseExecution(sagaID)

	span.SetStatus(1, "saga cancelled")
	span.AddEvent("saga_cancelled")
	return nil
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
	closed := oc.closed
	oc.mu.RUnlock()

	if closed {
		return nil, ErrCoordinatorClosed
	}

	// Constrain the query to active states. If the caller supplied explicit
	// states we respect them, otherwise default to the set of states that are
	// considered active (Running, StepCompleted, Compensating).
	effectiveFilter := buildActiveSagaFilter(filter)

	instances, err := oc.stateStorage.GetActiveSagas(context.Background(), effectiveFilter)
	if err != nil {
		return nil, saga.NewStorageError("GetActiveSagas", err)
	}

	// Storage may lag behind the live runtime state when persistence fails
	// (for example during a network partition). Prefer the in-memory
	// instance when available and re-check it against the active state
	// filter so callers observe the authoritative state.
	result := make([]saga.SagaInstance, 0, len(instances))
	for _, instance := range instances {
		live, ok := oc.instances.Load(instance.GetID())
		if !ok {
			result = append(result, instance)
			continue
		}
		liveInstance := live.(saga.SagaInstance)
		if stateInSet(liveInstance.GetState(), effectiveFilter.States) {
			result = append(result, liveInstance)
		}
	}
	return result, nil
}

// stateInSet reports whether the given state is contained in the set of states.
func stateInSet(state saga.SagaState, states []saga.SagaState) bool {
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

// buildActiveSagaFilter returns a filter constrained to active Saga states.
// It never mutates the caller-supplied filter.
func buildActiveSagaFilter(filter *saga.SagaFilter) *saga.SagaFilter {
	activeStates := []saga.SagaState{
		saga.StateRunning,
		saga.StateStepCompleted,
		saga.StateCompensating,
	}

	if filter == nil {
		return &saga.SagaFilter{States: activeStates}
	}

	copied := *filter
	if len(copied.States) == 0 {
		copied.States = activeStates
	}
	return &copied
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
	closed := oc.closed
	stateStorage := oc.stateStorage
	eventPublisher := oc.eventPublisher
	oc.mu.RUnlock()

	if closed {
		return ErrCoordinatorClosed
	}

	// Respect caller cancellation or deadline.
	if err := ctx.Err(); err != nil {
		return err
	}

	// Probe the state storage dependency.
	if stateStorage == nil {
		return ErrStateStorageNotConfigured
	}
	if err := probeHealth(ctx, stateStorage, func() error {
		_, err := stateStorage.GetActiveSagas(ctx, &saga.SagaFilter{Limit: 1})
		return err
	}); err != nil {
		return fmt.Errorf("state storage unhealthy: %w", err)
	}

	// Probe the event publisher dependency.
	if eventPublisher == nil {
		return ErrEventPublisherNotConfigured
	}
	if err := probeHealth(ctx, eventPublisher, nil); err != nil {
		return fmt.Errorf("event publisher unhealthy: %w", err)
	}

	return nil
}

// healthCheckable is implemented by dependencies that expose an explicit
// health probe (such as the Redis and Postgres state storage backends).
type healthCheckable interface {
	HealthCheck(ctx context.Context) error
}

// probeHealth checks the health of a dependency. It prefers an explicit
// HealthCheck method when the dependency implements healthCheckable, otherwise
// it falls back to the provided lightweight probe (if any).
func probeHealth(ctx context.Context, dependency interface{}, fallback func() error) error {
	if hc, ok := dependency.(healthCheckable); ok {
		return hc.HealthCheck(ctx)
	}
	if fallback != nil {
		return fallback()
	}
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
	if oc.closed {
		oc.mu.Unlock()
		return ErrCoordinatorClosed
	}
	oc.closed = true
	oc.mu.Unlock()

	// Create context for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown concurrency controller - waits for active Sagas to complete
	if oc.concurrencyController != nil {
		if err := oc.concurrencyController.Shutdown(ctx); err != nil {
			// Log error but continue with cleanup
			_ = err
		}
	}

	// Clean up timeout detector
	if oc.timeoutDetector != nil {
		oc.timeoutDetector.CleanupTimeouts()
	}

	// Update metrics
	oc.mu.Lock()
	oc.metrics.LastUpdateTime = time.Now()
	oc.mu.Unlock()

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

	ctx, span := oc.tracingManager.StartSpan(ctx, "saga.stop")
	defer span.End()
	span.SetAttribute("saga.id", sagaID)

	instance, _ := oc.loadInstance(ctx, sagaID)
	if instance == nil {
		err := saga.NewSagaNotFoundError(sagaID)
		span.RecordError(err)
		span.SetStatus(2, "saga not found")
		return err
	}

	if instance.GetState().IsTerminal() {
		err := saga.NewInvalidSagaStateError(instance.GetState(), saga.StateCancelled)
		span.RecordError(err)
		span.SetStatus(2, "saga already terminal")
		return err
	}

	// Signal in-flight execution to stop gracefully. The currently executing
	// step is allowed to finish; no new steps will be started once the
	// execution context is cancelled.
	oc.cancelExecution(sagaID)

	// Persist the current state so the Saga can be inspected or recovered
	// after the stop request.
	if orchInstance, ok := instance.(*OrchestratorSagaInstance); ok {
		orchInstance.mu.Lock()
		orchInstance.updatedAt = time.Now()
		orchInstance.mu.Unlock()

		if err := oc.stateStorage.SaveSaga(ctx, orchInstance); err != nil {
			span.RecordError(err)
			span.SetStatus(2, "failed to persist current state")
			return saga.NewStorageError("SaveSaga", err)
		}
	} else {
		if err := oc.stateStorage.UpdateSagaState(ctx, sagaID, instance.GetState(), nil); err != nil {
			span.RecordError(err)
			span.SetStatus(2, "failed to persist current state")
			return saga.NewStorageError("UpdateSagaState", err)
		}
	}

	span.SetStatus(1, "saga stopped")
	span.AddEvent("saga_stopped")
	return nil
}

// loadInstance resolves a Saga instance by ID. It prefers the in-memory cache
// (which holds the live runtime instance) and falls back to the configured
// state storage. The boolean return value reports whether the instance was
// resolved from the in-memory cache.
func (oc *OrchestratorCoordinator) loadInstance(ctx context.Context, sagaID string) (saga.SagaInstance, bool) {
	if v, ok := oc.instances.Load(sagaID); ok {
		return v.(saga.SagaInstance), true
	}

	instance, err := oc.stateStorage.GetSaga(ctx, sagaID)
	if err != nil || instance == nil {
		return nil, false
	}
	return instance, false
}

// cancelExecution signals any in-flight execution for the given Saga to stop
// by cancelling its execution context. It is a no-op if no execution is
// currently tracked.
func (oc *OrchestratorCoordinator) cancelExecution(sagaID string) {
	if cancel, ok := oc.cancelFuncs.Load(sagaID); ok {
		cancel.(context.CancelFunc)()
	}
}

// releaseExecution releases the per-Saga execution tracking state. It cancels
// and removes the execution context and removes the cached definition.
func (oc *OrchestratorCoordinator) releaseExecution(sagaID string) {
	if cancel, ok := oc.cancelFuncs.LoadAndDelete(sagaID); ok {
		cancel.(context.CancelFunc)()
	}
	oc.definitions.Delete(sagaID)
}

// compensateForCancellation drives compensation for the steps that have
// already completed when a Saga is cancelled. It reuses the per-step
// compensation logic from the compensation executor without performing the
// terminal state transition, which the caller controls.
func (oc *OrchestratorCoordinator) compensateForCancellation(ctx context.Context, instance *OrchestratorSagaInstance) {
	definitionValue, ok := oc.definitions.Load(instance.id)
	if !ok {
		return
	}
	definition, ok := definitionValue.(saga.SagaDefinition)
	if !ok {
		return
	}

	completedCount := instance.GetCompletedSteps()
	if completedCount <= 0 {
		return
	}

	steps := definition.GetSteps()
	if completedCount > len(steps) {
		completedCount = len(steps)
	}
	completedSteps := steps[:completedCount]

	strategy := definition.GetCompensationStrategy()
	if strategy == nil {
		strategy = saga.NewSequentialCompensationStrategy(30 * time.Second)
	}

	// Mark the instance as compensating for observability.
	instance.mu.Lock()
	instance.state = saga.StateCompensating
	instance.updatedAt = time.Now()
	instance.mu.Unlock()

	ce := newCompensationExecutor(oc, instance, definition)
	ce.publishCompensationStartedEvent(ctx, len(completedSteps))

	for compIndex, step := range strategy.GetCompensationOrder(completedSteps) {
		originalIndex := ce.findOriginalStepIndex(step, completedSteps)
		if originalIndex < 0 {
			continue
		}
		if err := ce.compensateStep(ctx, step, originalIndex, compIndex); err != nil {
			logger.GetSugaredLogger().Warnf(
				"Cancellation compensation failed for step %d of Saga %s: %v",
				originalIndex, instance.id, err)
		}
	}
}

// publishCancellationEvent publishes a Saga cancelled event for observability.
func (oc *OrchestratorCoordinator) publishCancellationEvent(
	ctx context.Context,
	instance saga.SagaInstance,
	reason string,
	duration time.Duration,
) {
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    instance.GetID(),
		Type:      saga.EventSagaCancelled,
		Version:   "1.0",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"reason": reason,
		},
		NewState: saga.StateCancelled,
		Duration: duration,
		TraceID:  instance.GetTraceID(),
		Source:   "OrchestratorCoordinator",
		Metadata: instance.GetMetadata(),
	}

	if err := oc.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish saga cancelled event: %v", err)
	}
}

// OrchestratorSagaInstance is the concrete implementation of saga.SagaInstance for orchestrator-based Sagas.
// It holds all runtime state and metadata for a Saga instance managed by OrchestratorCoordinator.
type OrchestratorSagaInstance struct {
	// Basic information
	id           string
	definitionID string
	name         string
	description  string

	// State information
	state          saga.SagaState
	currentStep    int
	completedSteps int
	totalSteps     int

	// Timing information
	createdAt   time.Time
	updatedAt   time.Time
	startedAt   *time.Time
	completedAt *time.Time
	timedOutAt  *time.Time

	// Data
	initialData interface{}
	currentData interface{}
	resultData  interface{}

	// Error information
	sagaError *saga.SagaError

	// Configuration
	timeout     time.Duration
	retryPolicy saga.RetryPolicy

	// Metadata
	metadata map[string]interface{}

	// Tracing
	traceID string
	spanID  string

	// Internal state management
	mu sync.RWMutex
}

// GetID returns the unique identifier of this Saga instance.
func (o *OrchestratorSagaInstance) GetID() string {
	return o.id
}

// GetDefinitionID returns the identifier of the Saga definition this instance follows.
func (o *OrchestratorSagaInstance) GetDefinitionID() string {
	return o.definitionID
}

// GetState returns the current state of the Saga instance.
func (o *OrchestratorSagaInstance) GetState() saga.SagaState {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.state
}

// GetCurrentStep returns the index of the currently executing step.
func (o *OrchestratorSagaInstance) GetCurrentStep() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.currentStep
}

// GetStartTime returns the time when the Saga instance was created.
func (o *OrchestratorSagaInstance) GetStartTime() time.Time {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.startedAt != nil {
		return *o.startedAt
	}
	return time.Time{}
}

// GetEndTime returns the time when the Saga instance reached a terminal state.
func (o *OrchestratorSagaInstance) GetEndTime() time.Time {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.completedAt != nil {
		return *o.completedAt
	}
	if o.timedOutAt != nil {
		return *o.timedOutAt
	}
	return time.Time{}
}

// GetResult returns the final result data of the Saga execution.
func (o *OrchestratorSagaInstance) GetResult() interface{} {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.resultData
}

// GetError returns the error that caused the Saga to fail.
func (o *OrchestratorSagaInstance) GetError() *saga.SagaError {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.sagaError
}

// GetTotalSteps returns the total number of steps in the Saga definition.
func (o *OrchestratorSagaInstance) GetTotalSteps() int {
	return o.totalSteps
}

// GetCompletedSteps returns the number of steps that have completed successfully.
func (o *OrchestratorSagaInstance) GetCompletedSteps() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.completedSteps
}

// GetCreatedAt returns the creation time of the Saga instance.
func (o *OrchestratorSagaInstance) GetCreatedAt() time.Time {
	return o.createdAt
}

// GetUpdatedAt returns the last update time of the Saga instance.
func (o *OrchestratorSagaInstance) GetUpdatedAt() time.Time {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.updatedAt
}

// GetTimeout returns the timeout duration for this Saga instance.
func (o *OrchestratorSagaInstance) GetTimeout() time.Duration {
	return o.timeout
}

// GetMetadata returns the metadata associated with this Saga instance.
func (o *OrchestratorSagaInstance) GetMetadata() map[string]interface{} {
	o.mu.RLock()
	defer o.mu.RUnlock()
	// Return a copy to prevent external mutation
	metadata := make(map[string]interface{}, len(o.metadata))
	for k, v := range o.metadata {
		metadata[k] = v
	}
	return metadata
}

// GetTraceID returns the distributed tracing identifier for this Saga.
func (o *OrchestratorSagaInstance) GetTraceID() string {
	return o.traceID
}

// IsTerminal returns true if the Saga is in a terminal state.
func (o *OrchestratorSagaInstance) IsTerminal() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.state.IsTerminal()
}

// IsActive returns true if the Saga is currently active.
func (o *OrchestratorSagaInstance) IsActive() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.state.IsActive()
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

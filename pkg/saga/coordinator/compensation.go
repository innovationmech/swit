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

package coordinator

import (
	"context"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/saga"
)

// compensationExecutor encapsulates the logic for executing compensation in reverse order.
// It manages compensation execution, state tracking, retry logic, and event publishing.
type compensationExecutor struct {
	coordinator *OrchestratorCoordinator
	instance    *OrchestratorSagaInstance
	definition  saga.SagaDefinition
}

// newCompensationExecutor creates a new compensation executor for the given Saga instance.
func newCompensationExecutor(
	coordinator *OrchestratorCoordinator,
	instance *OrchestratorSagaInstance,
	definition saga.SagaDefinition,
) *compensationExecutor {
	return &compensationExecutor{
		coordinator: coordinator,
		instance:    instance,
		definition:  definition,
	}
}

// executeCompensation executes compensation for all completed steps in reverse order.
// It manages state transitions, error handling, retry logic, and event publishing.
// Returns an error if compensation fails critically and cannot continue.
func (ce *compensationExecutor) executeCompensation(ctx context.Context, completedSteps []saga.SagaStep) error {
	// Start tracing span for compensation execution
	ctx, span := ce.coordinator.tracingManager.StartSpan(ctx, "saga.execute_compensation")
	defer span.End()

	span.SetAttribute("saga.id", ce.instance.id)
	span.SetAttribute("saga.definition_id", ce.definition.GetID())
	span.SetAttribute("compensation.steps_count", len(completedSteps))

	if len(completedSteps) == 0 {
		logger.GetSugaredLogger().Infof("No completed steps to compensate for Saga %s", ce.instance.id)
		span.SetStatus(1, "no steps to compensate")
		span.AddEvent("compensation_skipped_no_steps")
		return ce.handleCompensationCompletion(ctx)
	}

	// Update state to Compensating
	if err := ce.updateSagaState(ctx, saga.StateCompensating); err != nil {
		span.SetStatus(2, fmt.Sprintf("failed to update state: %v", err))
		span.RecordError(err)
		return fmt.Errorf("failed to update Saga state to Compensating: %w", err)
	}

	// Get compensation strategy
	strategy := ce.definition.GetCompensationStrategy()
	if strategy == nil {
		// Use default sequential strategy
		strategy = saga.NewSequentialCompensationStrategy(30 * time.Second)
	}

	// Check if compensation should be performed
	if !strategy.ShouldCompensate(ce.instance.sagaError) {
		logger.GetSugaredLogger().Infof("Compensation not required for Saga %s based on strategy", ce.instance.id)
		return ce.handleCompensationSkipped(ctx)
	}

	// Publish compensation started event
	ce.publishCompensationStartedEvent(ctx, len(completedSteps))

	// Get compensation order (typically reverse order)
	compensationSteps := strategy.GetCompensationOrder(completedSteps)

	// Track compensation failures
	var compensationErrors []error

	// Execute compensation for each step
	for i, step := range compensationSteps {
		select {
		case <-ctx.Done():
			// Context cancelled, stop compensation
			return ce.handleContextCancellation(ctx, i)
		default:
		}

		// Get original step index (for proper step state tracking)
		originalIndex := ce.findOriginalStepIndex(step, completedSteps)
		if originalIndex < 0 {
			logger.GetSugaredLogger().Warnf("Could not find original index for step %s", step.GetName())
			continue
		}

		// Execute compensation for this step
		err := ce.compensateStep(ctx, step, originalIndex, i)
		if err != nil {
			// Check if error is due to context cancellation
			if err == context.Canceled || err == context.DeadlineExceeded {
				return ce.handleContextCancellation(ctx, i)
			}

			logger.GetSugaredLogger().Errorf("Compensation failed for step %d (%s): %v", originalIndex, step.GetName(), err)
			compensationErrors = append(compensationErrors, err)

			// Depending on strategy, we may continue or stop
			// For now, continue with best-effort approach
			continue
		}
	}

	// Handle compensation completion
	if len(compensationErrors) > 0 {
		span.SetAttribute("compensation.errors_count", len(compensationErrors))
		span.SetStatus(2, fmt.Sprintf("compensation failed with %d errors", len(compensationErrors)))
		span.AddEvent("compensation_failed")
		return ce.handleCompensationFailure(ctx, compensationErrors)
	}

	span.SetStatus(1, "compensation completed successfully")
	span.AddEvent("compensation_completed")
	return ce.handleCompensationCompletion(ctx)
}

// compensateStep executes compensation for a single step with retry logic.
// Returns an error if all compensation attempts fail.
func (ce *compensationExecutor) compensateStep(
	ctx context.Context,
	step saga.SagaStep,
	stepIndex int,
	compensationIndex int,
) error {
	// Get step state from storage
	stepStates, err := ce.coordinator.stateStorage.GetStepStates(ctx, ce.instance.id)
	if err != nil {
		logger.GetSugaredLogger().Warnf("Failed to load step states: %v", err)
	}

	var stepState *saga.StepState
	if stepIndex < len(stepStates) {
		stepState = stepStates[stepIndex]
	}

	// If no step state exists, create a minimal one
	if stepState == nil {
		now := time.Now()
		stepState = &saga.StepState{
			ID:        fmt.Sprintf("%s-step-%d", ce.instance.id, stepIndex),
			SagaID:    ce.instance.id,
			StepIndex: stepIndex,
			Name:      step.GetName(),
			State:     saga.StepStateCompleted, // Assuming it was completed before
			CreatedAt: now,
		}
	}

	// Initialize compensation state if not exists
	if stepState.CompensationState == nil {
		stepState.CompensationState = &saga.CompensationState{
			State:       saga.CompensationStatePending,
			Attempts:    0,
			MaxAttempts: ce.getMaxCompensationAttempts(step),
		}
	}

	// Update step state to compensating
	startTime := time.Now()
	stepState.State = saga.StepStateCompensating
	stepState.CompensationState.State = saga.CompensationStateRunning
	stepState.CompensationState.StartedAt = &startTime

	// Persist updated step state
	if err := ce.coordinator.stateStorage.SaveStepState(ctx, ce.instance.id, stepState); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to save compensating step state: %v", err)
	}

	// Publish compensation step started event
	ce.publishCompensationStepStartedEvent(ctx, step, stepIndex, compensationIndex)

	// Log compensation start
	logger.GetSugaredLogger().Infof("Starting compensation for step %d (%s) in Saga %s", stepIndex, step.GetName(), ce.instance.id)

	// Execute compensation with retry logic
	var lastErr error
	maxAttempts := stepState.CompensationState.MaxAttempts

	for attempt := 0; attempt < maxAttempts; attempt++ {
		stepState.CompensationState.Attempts = attempt + 1

		// Create compensation context with timeout
		compensationCtx, cancel := ce.createCompensationContext(ctx, step)

		// Execute the compensation
		err := step.Compensate(compensationCtx, stepState.OutputData)
		cancel()

		duration := time.Since(startTime)

		if err == nil {
			// Compensation succeeded
			completedTime := time.Now()
			stepState.State = saga.StepStateCompensated
			stepState.CompensationState.State = saga.CompensationStateCompleted
			stepState.CompensationState.CompletedAt = &completedTime

			// Persist final step state
			if saveErr := ce.coordinator.stateStorage.SaveStepState(ctx, ce.instance.id, stepState); saveErr != nil {
				logger.GetSugaredLogger().Warnf("Failed to save compensated step state: %v", saveErr)
			}

			// Record metrics
			ce.coordinator.metricsCollector.RecordCompensationExecuted(
				ce.definition.GetID(),
				step.GetID(),
				true,
				duration,
			)

			// Publish compensation step completed event
			ce.publishCompensationStepCompletedEvent(ctx, step, stepIndex, compensationIndex, duration)

			// Log success
			logger.GetSugaredLogger().Infof("Compensation for step %d (%s) completed successfully in %v", stepIndex, step.GetName(), duration)

			return nil
		}

		// Compensation failed
		lastErr = err
		logger.GetSugaredLogger().Warnf("Compensation for step %d (%s) attempt %d failed: %v", stepIndex, step.GetName(), attempt+1, err)

		// Create SagaError
		sagaError := ce.createCompensationError(err, step)
		stepState.CompensationState.Error = sagaError

		// Check if we should retry
		if attempt+1 < maxAttempts {
			// Calculate retry delay (use exponential backoff)
			retryDelay := time.Duration(attempt+1) * time.Second
			logger.GetSugaredLogger().Infof("Retrying compensation for step %d (%s) after %v (attempt %d/%d)",
				stepIndex, step.GetName(), retryDelay, attempt+1, maxAttempts)

			// Wait before retry
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				stepState.CompensationState.State = saga.CompensationStateFailed
				return ctx.Err()
			}
		} else {
			// No more retries, mark as failed
			stepState.State = saga.StepStateCompensating // Keep in compensating state
			stepState.CompensationState.State = saga.CompensationStateFailed

			// Record metrics
			ce.coordinator.metricsCollector.RecordCompensationExecuted(
				ce.definition.GetID(),
				step.GetID(),
				false,
				duration,
			)

			// Publish compensation step failed event
			ce.publishCompensationStepFailedEvent(ctx, step, stepIndex, compensationIndex, sagaError, attempt+1)

			// Persist final step state
			if saveErr := ce.coordinator.stateStorage.SaveStepState(ctx, ce.instance.id, stepState); saveErr != nil {
				logger.GetSugaredLogger().Warnf("Failed to save failed compensation step state: %v", saveErr)
			}

			return lastErr
		}
	}

	return lastErr
}

// createCompensationContext creates a context with timeout for compensation execution.
func (ce *compensationExecutor) createCompensationContext(ctx context.Context, step saga.SagaStep) (context.Context, context.CancelFunc) {
	// Get compensation strategy timeout
	strategy := ce.definition.GetCompensationStrategy()
	if strategy == nil {
		// Use default timeout
		return context.WithTimeout(ctx, 30*time.Second)
	}

	timeout := strategy.GetCompensationTimeout()
	if timeout == 0 {
		// No timeout specified, return context without timeout
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

// getMaxCompensationAttempts returns the maximum number of compensation attempts.
func (ce *compensationExecutor) getMaxCompensationAttempts(step saga.SagaStep) int {
	// Default to 3 attempts for compensation
	// This can be made configurable in the future
	return 3
}

// findOriginalStepIndex finds the original index of a step in the completed steps list.
func (ce *compensationExecutor) findOriginalStepIndex(step saga.SagaStep, completedSteps []saga.SagaStep) int {
	for i, s := range completedSteps {
		if s.GetID() == step.GetID() {
			return i
		}
	}
	return -1
}

// createCompensationError creates a SagaError from a compensation error.
func (ce *compensationExecutor) createCompensationError(err error, step saga.SagaStep) *saga.SagaError {
	errorType := saga.ErrorTypeCompensation

	// Check for specific error types
	if err == context.DeadlineExceeded {
		errorType = saga.ErrorTypeTimeout
	} else if err == context.Canceled {
		errorType = saga.ErrorTypeSystem
	}

	return &saga.SagaError{
		Code:      "COMPENSATION_FAILED",
		Message:   err.Error(),
		Type:      errorType,
		Retryable: true, // Compensation failures are typically retryable
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"step_id":   step.GetID(),
			"step_name": step.GetName(),
		},
	}
}

// updateSagaState updates the Saga instance state and persists it.
func (ce *compensationExecutor) updateSagaState(ctx context.Context, newState saga.SagaState) error {
	ce.instance.mu.Lock()
	oldState := ce.instance.state
	ce.instance.state = newState
	ce.instance.updatedAt = time.Now()
	ce.instance.mu.Unlock()

	// Persist state change
	if err := ce.coordinator.stateStorage.UpdateSagaState(ctx, ce.instance.id, newState, nil); err != nil {
		return err
	}

	// Publish state change event
	event := &saga.SagaEvent{
		ID:            generateEventID(),
		SagaID:        ce.instance.id,
		Type:          saga.EventStateChanged,
		Version:       "1.0",
		Timestamp:     time.Now(),
		PreviousState: oldState,
		NewState:      newState,
		TraceID:       ce.instance.traceID,
		SpanID:        ce.instance.spanID,
		Source:        "OrchestratorCoordinator",
		Metadata:      ce.instance.metadata,
	}

	if err := ce.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish state change event: %v", err)
	}

	return nil
}

// handleContextCancellation handles context cancellation during compensation.
func (ce *compensationExecutor) handleContextCancellation(ctx context.Context, currentStep int) error {
	logger.GetSugaredLogger().Warnf("Saga %s compensation cancelled at step %d", ce.instance.id, currentStep)

	// Update state to Cancelled
	if err := ce.updateSagaState(context.Background(), saga.StateCancelled); err != nil {
		logger.GetSugaredLogger().Errorf("Failed to update state to Cancelled: %v", err)
	}

	// Publish cancellation event
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    ce.instance.id,
		Type:      saga.EventSagaCancelled,
		Version:   "1.0",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"reason":                  "compensation_context_cancelled",
			"compensation_step_index": currentStep,
		},
		TraceID:  ce.instance.traceID,
		SpanID:   ce.instance.spanID,
		Source:   "OrchestratorCoordinator",
		Metadata: ce.instance.metadata,
	}

	if err := ce.coordinator.eventPublisher.PublishEvent(context.Background(), event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish cancellation event: %v", err)
	}

	return ctx.Err()
}

// handleCompensationFailure handles compensation failure.
func (ce *compensationExecutor) handleCompensationFailure(ctx context.Context, errors []error) error {
	logger.GetSugaredLogger().Errorf("Saga %s compensation failed with %d errors", ce.instance.id, len(errors))

	// Update state to Failed (compensation failed)
	if err := ce.updateSagaState(ctx, saga.StateFailed); err != nil {
		logger.GetSugaredLogger().Errorf("Failed to update state to Failed: %v", err)
	}

	// Update coordinator metrics
	ce.coordinator.mu.Lock()
	ce.coordinator.metrics.FailedSagas++
	ce.coordinator.metrics.LastUpdateTime = time.Now()
	ce.coordinator.mu.Unlock()

	// Calculate duration
	duration := time.Since(ce.instance.GetStartTime())

	// Create aggregate error
	aggregateError := &saga.SagaError{
		Code:      "COMPENSATION_FAILED",
		Message:   fmt.Sprintf("Compensation failed with %d errors", len(errors)),
		Type:      saga.ErrorTypeCompensation,
		Retryable: false,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"error_count": len(errors),
			"errors":      errors,
		},
	}

	// Publish compensation failed event
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    ce.instance.id,
		Type:      saga.EventCompensationFailed,
		Version:   "1.0",
		Timestamp: time.Now(),
		Error:     aggregateError,
		Duration:  duration,
		TraceID:   ce.instance.traceID,
		SpanID:    ce.instance.spanID,
		Source:    "OrchestratorCoordinator",
		Metadata:  ce.instance.metadata,
	}

	if err := ce.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish compensation failed event: %v", err)
	}

	return fmt.Errorf("compensation failed: %d errors occurred", len(errors))
}

// handleCompensationCompletion handles successful completion of compensation.
func (ce *compensationExecutor) handleCompensationCompletion(ctx context.Context) error {
	logger.GetSugaredLogger().Infof("Saga %s compensation completed successfully", ce.instance.id)

	// Update instance state
	completedTime := time.Now()
	ce.instance.mu.Lock()
	ce.instance.state = saga.StateCompensated
	ce.instance.completedAt = &completedTime
	ce.instance.updatedAt = completedTime
	ce.instance.mu.Unlock()

	// Persist final state
	if err := ce.coordinator.stateStorage.SaveSaga(ctx, ce.instance); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to save compensated Saga state: %v", err)
	}

	// Update coordinator metrics
	ce.coordinator.mu.Lock()
	ce.coordinator.metrics.TotalCompensations++
	ce.coordinator.metrics.SuccessfulCompensations++
	ce.coordinator.metrics.ActiveSagas--
	ce.coordinator.metrics.LastUpdateTime = time.Now()
	ce.coordinator.mu.Unlock()

	// Calculate duration
	duration := time.Since(ce.instance.GetStartTime())

	// Publish compensation completed event
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    ce.instance.id,
		Type:      saga.EventCompensationCompleted,
		Version:   "1.0",
		Timestamp: completedTime,
		NewState:  saga.StateCompensated,
		Duration:  duration,
		TraceID:   ce.instance.traceID,
		SpanID:    ce.instance.spanID,
		Source:    "OrchestratorCoordinator",
		Metadata:  ce.instance.metadata,
	}

	if err := ce.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish compensation completed event: %v", err)
	}

	return nil
}

// handleCompensationSkipped handles the case where compensation is not required.
func (ce *compensationExecutor) handleCompensationSkipped(ctx context.Context) error {
	logger.GetSugaredLogger().Infof("Saga %s compensation skipped per strategy", ce.instance.id)

	// Update state to Failed (but with compensation skipped)
	if err := ce.updateSagaState(ctx, saga.StateFailed); err != nil {
		logger.GetSugaredLogger().Errorf("Failed to update state to Failed: %v", err)
	}

	return nil
}

// publishCompensationStartedEvent publishes a compensation started event.
func (ce *compensationExecutor) publishCompensationStartedEvent(ctx context.Context, stepCount int) {
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    ce.instance.id,
		Type:      saga.EventCompensationStarted,
		Version:   "1.0",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"steps_to_compensate": stepCount,
		},
		TraceID:  ce.instance.traceID,
		SpanID:   ce.instance.spanID,
		Source:   "OrchestratorCoordinator",
		Metadata: ce.instance.metadata,
	}

	if err := ce.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish compensation started event: %v", err)
	}
}

// publishCompensationStepStartedEvent publishes a compensation step started event.
func (ce *compensationExecutor) publishCompensationStepStartedEvent(
	ctx context.Context,
	step saga.SagaStep,
	stepIndex int,
	compensationIndex int,
) {
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    ce.instance.id,
		StepID:    step.GetID(),
		Type:      saga.EventCompensationStepStarted,
		Version:   "1.0",
		Timestamp: time.Now(),
		TraceID:   ce.instance.traceID,
		SpanID:    ce.instance.spanID,
		Source:    "OrchestratorCoordinator",
		Metadata: map[string]interface{}{
			"step_index":         stepIndex,
			"step_name":          step.GetName(),
			"compensation_index": compensationIndex,
		},
	}

	if err := ce.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish compensation step started event: %v", err)
	}
}

// publishCompensationStepCompletedEvent publishes a compensation step completed event.
func (ce *compensationExecutor) publishCompensationStepCompletedEvent(
	ctx context.Context,
	step saga.SagaStep,
	stepIndex int,
	compensationIndex int,
	duration time.Duration,
) {
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    ce.instance.id,
		StepID:    step.GetID(),
		Type:      saga.EventCompensationStepCompleted,
		Version:   "1.0",
		Timestamp: time.Now(),
		Duration:  duration,
		TraceID:   ce.instance.traceID,
		SpanID:    ce.instance.spanID,
		Source:    "OrchestratorCoordinator",
		Metadata: map[string]interface{}{
			"step_index":         stepIndex,
			"step_name":          step.GetName(),
			"compensation_index": compensationIndex,
		},
	}

	if err := ce.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish compensation step completed event: %v", err)
	}
}

// publishCompensationStepFailedEvent publishes a compensation step failed event.
func (ce *compensationExecutor) publishCompensationStepFailedEvent(
	ctx context.Context,
	step saga.SagaStep,
	stepIndex int,
	compensationIndex int,
	sagaError *saga.SagaError,
	attempts int,
) {
	event := &saga.SagaEvent{
		ID:          generateEventID(),
		SagaID:      ce.instance.id,
		StepID:      step.GetID(),
		Type:        saga.EventCompensationStepFailed,
		Version:     "1.0",
		Timestamp:   time.Now(),
		Error:       sagaError,
		Attempt:     attempts,
		MaxAttempts: ce.getMaxCompensationAttempts(step),
		TraceID:     ce.instance.traceID,
		SpanID:      ce.instance.spanID,
		Source:      "OrchestratorCoordinator",
		Metadata: map[string]interface{}{
			"step_index":         stepIndex,
			"step_name":          step.GetName(),
			"compensation_index": compensationIndex,
		},
	}

	if err := ce.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish compensation step failed event: %v", err)
	}
}

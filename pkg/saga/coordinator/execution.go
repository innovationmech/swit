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

// stepExecutor encapsulates the logic for executing Saga steps sequentially.
// It manages step execution, state tracking, data passing, retry logic, and event publishing.
type stepExecutor struct {
	coordinator *OrchestratorCoordinator
	instance    *OrchestratorSagaInstance
	definition  saga.SagaDefinition
}

// newStepExecutor creates a new step executor for the given Saga instance.
func newStepExecutor(
	coordinator *OrchestratorCoordinator,
	instance *OrchestratorSagaInstance,
	definition saga.SagaDefinition,
) *stepExecutor {
	return &stepExecutor{
		coordinator: coordinator,
		instance:    instance,
		definition:  definition,
	}
}

// executeSteps executes all steps in the Saga definition sequentially.
// It manages state transitions, data passing between steps, retry logic,
// and error handling. Returns an error if any step fails after all retries.
func (se *stepExecutor) executeSteps(ctx context.Context) error {
	steps := se.definition.GetSteps()
	if len(steps) == 0 {
		return fmt.Errorf("no steps defined in Saga definition")
	}

	// Update state to Running
	if err := se.updateSagaState(ctx, saga.StateRunning); err != nil {
		return fmt.Errorf("failed to update Saga state to Running: %w", err)
	}

	// Track step states for this execution
	stepStates := make([]*saga.StepState, len(steps))

	// Execute each step in sequence
	currentData := se.instance.currentData
	for i, step := range steps {
		select {
		case <-ctx.Done():
			// Context cancelled, stop execution
			return se.handleContextCancellation(ctx, i, stepStates)
		default:
		}

		// Execute the current step
		stepState, outputData, err := se.executeStep(ctx, step, i, currentData)
		stepStates[i] = stepState

		if err != nil {
			// Step execution failed after all retries
			logger.GetSugaredLogger().Errorf("Step %d (%s) failed: %v", i, step.GetName(), err)
			return se.handleStepFailure(ctx, i, stepState, err)
		}

		// Step succeeded, update current data for next step
		currentData = outputData
		se.updateCurrentData(currentData)

		// Update state after step completion
		if err := se.updateSagaState(ctx, saga.StateStepCompleted); err != nil {
			logger.GetSugaredLogger().Warnf("Failed to update state to StepCompleted: %v", err)
		}
	}

	// All steps completed successfully
	return se.handleSagaCompletion(ctx, currentData)
}

// executeStep executes a single step with retry logic.
// Returns the step state, output data, and any error that occurred.
func (se *stepExecutor) executeStep(
	ctx context.Context,
	step saga.SagaStep,
	stepIndex int,
	inputData interface{},
) (*saga.StepState, interface{}, error) {
	// Initialize step state
	now := time.Now()
	stepState := &saga.StepState{
		ID:          fmt.Sprintf("%s-step-%d", se.instance.id, stepIndex),
		SagaID:      se.instance.id,
		StepIndex:   stepIndex,
		Name:        step.GetName(),
		State:       saga.StepStatePending,
		Attempts:    0,
		MaxAttempts: se.getMaxAttempts(step),
		CreatedAt:   now,
		InputData:   inputData,
		Metadata:    step.GetMetadata(),
	}

	// Persist step state
	if err := se.coordinator.stateStorage.SaveStepState(ctx, se.instance.id, stepState); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to save initial step state: %v", err)
	}

	// Update instance current step
	se.updateCurrentStep(stepIndex)

	// Publish step started event
	se.publishStepStartedEvent(ctx, step, stepIndex, inputData)

	// Log step execution start
	logger.GetSugaredLogger().Infof("Starting step %d (%s) for Saga %s", stepIndex, step.GetName(), se.instance.id)

	// Execute with retry logic
	var outputData interface{}
	var lastErr error

	retryPolicy := step.GetRetryPolicy()
	if retryPolicy == nil {
		retryPolicy = se.coordinator.retryPolicy
	}

	maxAttempts := retryPolicy.GetMaxAttempts()
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Update step state to running
		startTime := time.Now()
		stepState.State = saga.StepStateRunning
		stepState.Attempts = attempt + 1
		stepState.StartedAt = &startTime
		stepState.LastAttemptAt = &startTime

		// Persist updated step state
		if err := se.coordinator.stateStorage.SaveStepState(ctx, se.instance.id, stepState); err != nil {
			logger.GetSugaredLogger().Warnf("Failed to save step state: %v", err)
		}

		// Create step execution context with timeout
		stepCtx, cancel := se.createStepContext(ctx, step)

		// Execute the step
		result, err := step.Execute(stepCtx, inputData)
		cancel()

		duration := time.Since(startTime)

		if err == nil {
			// Step succeeded
			completedTime := time.Now()
			stepState.State = saga.StepStateCompleted
			stepState.CompletedAt = &completedTime
			stepState.OutputData = result

			// Persist final step state
			if saveErr := se.coordinator.stateStorage.SaveStepState(ctx, se.instance.id, stepState); saveErr != nil {
				logger.GetSugaredLogger().Warnf("Failed to save completed step state: %v", saveErr)
			}

			// Update instance metrics
			se.incrementCompletedSteps()

			// Record metrics
			se.coordinator.metricsCollector.RecordStepExecuted(
				se.definition.GetID(),
				step.GetID(),
				true,
				duration,
			)

			// Publish step completed event
			se.publishStepCompletedEvent(ctx, step, stepIndex, result, duration)

			// Log success
			logger.GetSugaredLogger().Infof("Step %d (%s) completed successfully in %v", stepIndex, step.GetName(), duration)

			outputData = result
			break
		}

		// Step failed
		lastErr = err
		logger.GetSugaredLogger().Warnf("Step %d (%s) attempt %d failed: %v", stepIndex, step.GetName(), attempt+1, err)

		// Create SagaError
		sagaError := se.createSagaError(err, step)
		stepState.Error = sagaError

		// Check if we should retry
		if attempt+1 < maxAttempts && retryPolicy.ShouldRetry(err, attempt+1) {
			// Record retry metrics
			se.coordinator.metricsCollector.RecordStepRetried(
				se.definition.GetID(),
				step.GetID(),
				attempt+1,
			)

			// Calculate retry delay
			retryDelay := retryPolicy.GetRetryDelay(attempt)
			logger.GetSugaredLogger().Infof("Retrying step %d (%s) after %v (attempt %d/%d)",
				stepIndex, step.GetName(), retryDelay, attempt+1, maxAttempts)

			// Wait before retry
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				stepState.State = saga.StepStateFailed
				return stepState, nil, ctx.Err()
			}
		} else {
			// No more retries, mark as failed
			stepState.State = saga.StepStateFailed

			// Record metrics
			se.coordinator.metricsCollector.RecordStepExecuted(
				se.definition.GetID(),
				step.GetID(),
				false,
				duration,
			)

			// Publish step failed event
			se.publishStepFailedEvent(ctx, step, stepIndex, sagaError, attempt+1)

			// Persist final step state
			if saveErr := se.coordinator.stateStorage.SaveStepState(ctx, se.instance.id, stepState); saveErr != nil {
				logger.GetSugaredLogger().Warnf("Failed to save failed step state: %v", saveErr)
			}

			return stepState, nil, lastErr
		}
	}

	return stepState, outputData, nil
}

// createStepContext creates a context with timeout for step execution.
func (se *stepExecutor) createStepContext(ctx context.Context, step saga.SagaStep) (context.Context, context.CancelFunc) {
	timeout := step.GetTimeout()
	if timeout == 0 {
		// Use default timeout from Saga definition
		timeout = se.definition.GetTimeout()
	}
	if timeout == 0 {
		// No timeout specified, return context without timeout
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

// getMaxAttempts returns the maximum number of attempts for a step.
func (se *stepExecutor) getMaxAttempts(step saga.SagaStep) int {
	retryPolicy := step.GetRetryPolicy()
	if retryPolicy == nil {
		retryPolicy = se.coordinator.retryPolicy
	}
	return retryPolicy.GetMaxAttempts()
}

// createSagaError creates a SagaError from a standard error.
func (se *stepExecutor) createSagaError(err error, step saga.SagaStep) *saga.SagaError {
	// Determine error type and retryability
	errorType := saga.ErrorTypeService
	retryable := step.IsRetryable(err)

	// Check for specific error types
	if err == context.DeadlineExceeded {
		errorType = saga.ErrorTypeTimeout
	} else if err == context.Canceled {
		errorType = saga.ErrorTypeSystem
	}

	return &saga.SagaError{
		Code:      "STEP_EXECUTION_FAILED",
		Message:   err.Error(),
		Type:      errorType,
		Retryable: retryable,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"step_id":   step.GetID(),
			"step_name": step.GetName(),
		},
	}
}

// updateSagaState updates the Saga instance state and persists it.
func (se *stepExecutor) updateSagaState(ctx context.Context, newState saga.SagaState) error {
	se.instance.mu.Lock()
	oldState := se.instance.state
	se.instance.state = newState
	se.instance.updatedAt = time.Now()
	se.instance.mu.Unlock()

	// Persist state change
	if err := se.coordinator.stateStorage.UpdateSagaState(ctx, se.instance.id, newState, nil); err != nil {
		return err
	}

	// Publish state change event
	event := &saga.SagaEvent{
		ID:            generateEventID(),
		SagaID:        se.instance.id,
		Type:          saga.EventStateChanged,
		Version:       "1.0",
		Timestamp:     time.Now(),
		PreviousState: oldState,
		NewState:      newState,
		TraceID:       se.instance.traceID,
		SpanID:        se.instance.spanID,
		Source:        "OrchestratorCoordinator",
		Metadata:      se.instance.metadata,
	}

	if err := se.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish state change event: %v", err)
	}

	return nil
}

// updateCurrentStep updates the current step index in the instance.
func (se *stepExecutor) updateCurrentStep(stepIndex int) {
	se.instance.mu.Lock()
	se.instance.currentStep = stepIndex
	se.instance.updatedAt = time.Now()
	se.instance.mu.Unlock()
}

// updateCurrentData updates the current data in the instance.
func (se *stepExecutor) updateCurrentData(data interface{}) {
	se.instance.mu.Lock()
	se.instance.currentData = data
	se.instance.updatedAt = time.Now()
	se.instance.mu.Unlock()
}

// incrementCompletedSteps increments the completed steps counter.
func (se *stepExecutor) incrementCompletedSteps() {
	se.instance.mu.Lock()
	se.instance.completedSteps++
	se.instance.updatedAt = time.Now()
	se.instance.mu.Unlock()
}

// handleContextCancellation handles context cancellation during execution.
func (se *stepExecutor) handleContextCancellation(
	ctx context.Context,
	currentStep int,
	stepStates []*saga.StepState,
) error {
	logger.GetSugaredLogger().Warnf("Saga %s execution cancelled at step %d", se.instance.id, currentStep)

	// Update state to Cancelled
	if err := se.updateSagaState(context.Background(), saga.StateCancelled); err != nil {
		logger.GetSugaredLogger().Errorf("Failed to update state to Cancelled: %v", err)
	}

	// Publish cancellation event
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    se.instance.id,
		Type:      saga.EventSagaCancelled,
		Version:   "1.0",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"reason":       "context_cancelled",
			"current_step": currentStep,
		},
		TraceID:  se.instance.traceID,
		SpanID:   se.instance.spanID,
		Source:   "OrchestratorCoordinator",
		Metadata: se.instance.metadata,
	}

	if err := se.coordinator.eventPublisher.PublishEvent(context.Background(), event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish cancellation event: %v", err)
	}

	return ctx.Err()
}

// handleStepFailure handles a step execution failure.
// It triggers compensation for all completed steps before the failure.
func (se *stepExecutor) handleStepFailure(
	ctx context.Context,
	stepIndex int,
	stepState *saga.StepState,
	err error,
) error {
	// Update instance with error
	se.instance.mu.Lock()
	se.instance.sagaError = stepState.Error
	se.instance.mu.Unlock()

	// Calculate duration
	duration := time.Since(se.instance.GetStartTime())

	// Publish Saga failed event
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    se.instance.id,
		Type:      saga.EventSagaFailed,
		Version:   "1.0",
		Timestamp: time.Now(),
		Error:     stepState.Error,
		Data: map[string]interface{}{
			"failed_step":      stepIndex,
			"failed_step_name": stepState.Name,
			"completed_steps":  se.instance.completedSteps,
			"total_steps":      se.instance.totalSteps,
		},
		Duration: duration,
		TraceID:  se.instance.traceID,
		SpanID:   se.instance.spanID,
		Source:   "OrchestratorCoordinator",
		Metadata: se.instance.metadata,
	}

	if pubErr := se.coordinator.eventPublisher.PublishEvent(ctx, event); pubErr != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish Saga failed event: %v", pubErr)
	}

	// Record metrics
	se.coordinator.metricsCollector.RecordSagaFailed(
		se.definition.GetID(),
		stepState.Error.Type,
		duration,
	)

	// Trigger compensation for completed steps
	if stepIndex > 0 {
		// Get completed steps up to the failed step
		allSteps := se.definition.GetSteps()
		completedSteps := allSteps[:stepIndex]

		// Create compensation executor
		compensationExecutor := newCompensationExecutor(se.coordinator, se.instance, se.definition)

		// Execute compensation
		if compErr := compensationExecutor.executeCompensation(ctx, completedSteps); compErr != nil {
			logger.GetSugaredLogger().Errorf("Compensation execution failed: %v", compErr)
			// Continue even if compensation fails - the saga is already failed
		}
	} else {
		// No steps completed, just update state to Failed
		if updateErr := se.updateSagaState(ctx, saga.StateFailed); updateErr != nil {
			logger.GetSugaredLogger().Errorf("Failed to update state to Failed: %v", updateErr)
		}

		// Update coordinator metrics
		se.coordinator.mu.Lock()
		se.coordinator.metrics.FailedSagas++
		se.coordinator.metrics.ActiveSagas--
		se.coordinator.metrics.LastUpdateTime = time.Now()
		se.coordinator.mu.Unlock()
	}

	return err
}

// handleSagaCompletion handles successful completion of all steps.
func (se *stepExecutor) handleSagaCompletion(ctx context.Context, finalData interface{}) error {
	// Update instance with result
	completedTime := time.Now()
	se.instance.mu.Lock()
	se.instance.state = saga.StateCompleted
	se.instance.resultData = finalData
	se.instance.completedAt = &completedTime
	se.instance.updatedAt = completedTime
	se.instance.mu.Unlock()

	// Persist final state
	if err := se.coordinator.stateStorage.SaveSaga(ctx, se.instance); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to save completed Saga state: %v", err)
	}

	// Update coordinator metrics
	se.coordinator.mu.Lock()
	se.coordinator.metrics.CompletedSagas++
	se.coordinator.metrics.ActiveSagas--
	se.coordinator.metrics.LastUpdateTime = time.Now()
	se.coordinator.mu.Unlock()

	// Calculate duration
	duration := time.Since(se.instance.GetStartTime())

	// Record metrics
	se.coordinator.metricsCollector.RecordSagaCompleted(
		se.definition.GetID(),
		duration,
	)

	// Publish Saga completed event
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    se.instance.id,
		Type:      saga.EventSagaCompleted,
		Version:   "1.0",
		Timestamp: completedTime,
		Data:      finalData,
		NewState:  saga.StateCompleted,
		Duration:  duration,
		TraceID:   se.instance.traceID,
		SpanID:    se.instance.spanID,
		Source:    "OrchestratorCoordinator",
		Metadata:  se.instance.metadata,
	}

	if err := se.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish Saga completed event: %v", err)
	}

	// Log completion
	logger.GetSugaredLogger().Infof("Saga %s completed successfully in %v", se.instance.id, duration)

	return nil
}

// publishStepStartedEvent publishes a step started event.
func (se *stepExecutor) publishStepStartedEvent(
	ctx context.Context,
	step saga.SagaStep,
	stepIndex int,
	inputData interface{},
) {
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    se.instance.id,
		StepID:    step.GetID(),
		Type:      saga.EventSagaStepStarted,
		Version:   "1.0",
		Timestamp: time.Now(),
		Data:      inputData,
		TraceID:   se.instance.traceID,
		SpanID:    se.instance.spanID,
		Source:    "OrchestratorCoordinator",
		Metadata: map[string]interface{}{
			"step_index": stepIndex,
			"step_name":  step.GetName(),
		},
	}

	if err := se.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish step started event: %v", err)
	}
}

// publishStepCompletedEvent publishes a step completed event.
func (se *stepExecutor) publishStepCompletedEvent(
	ctx context.Context,
	step saga.SagaStep,
	stepIndex int,
	outputData interface{},
	duration time.Duration,
) {
	event := &saga.SagaEvent{
		ID:        generateEventID(),
		SagaID:    se.instance.id,
		StepID:    step.GetID(),
		Type:      saga.EventSagaStepCompleted,
		Version:   "1.0",
		Timestamp: time.Now(),
		Data:      outputData,
		Duration:  duration,
		TraceID:   se.instance.traceID,
		SpanID:    se.instance.spanID,
		Source:    "OrchestratorCoordinator",
		Metadata: map[string]interface{}{
			"step_index": stepIndex,
			"step_name":  step.GetName(),
		},
	}

	if err := se.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish step completed event: %v", err)
	}
}

// publishStepFailedEvent publishes a step failed event.
func (se *stepExecutor) publishStepFailedEvent(
	ctx context.Context,
	step saga.SagaStep,
	stepIndex int,
	sagaError *saga.SagaError,
	attempts int,
) {
	event := &saga.SagaEvent{
		ID:          generateEventID(),
		SagaID:      se.instance.id,
		StepID:      step.GetID(),
		Type:        saga.EventSagaStepFailed,
		Version:     "1.0",
		Timestamp:   time.Now(),
		Error:       sagaError,
		Attempt:     attempts,
		MaxAttempts: se.getMaxAttempts(step),
		TraceID:     se.instance.traceID,
		SpanID:      se.instance.spanID,
		Source:      "OrchestratorCoordinator",
		Metadata: map[string]interface{}{
			"step_index": stepIndex,
			"step_name":  step.GetName(),
		},
	}

	if err := se.coordinator.eventPublisher.PublishEvent(ctx, event); err != nil {
		logger.GetSugaredLogger().Warnf("Failed to publish step failed event: %v", err)
	}
}

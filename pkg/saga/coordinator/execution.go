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
	coordinator  *OrchestratorCoordinator
	instance     *OrchestratorSagaInstance
	definition   saga.SagaDefinition
	errorHandler *ErrorHandler
}

// newStepExecutor creates a new step executor for the given Saga instance.
func newStepExecutor(
	coordinator *OrchestratorCoordinator,
	instance *OrchestratorSagaInstance,
	definition saga.SagaDefinition,
) *stepExecutor {
	return &stepExecutor{
		coordinator:  coordinator,
		instance:     instance,
		definition:   definition,
		errorHandler: newErrorHandler(coordinator, instance),
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

	// Start Saga-level timeout tracking
	sagaTimeout := se.definition.GetTimeout()
	if sagaTimeout <= 0 {
		sagaTimeout = se.instance.GetTimeout()
	}
	
	var sagaTimeoutCancel context.CancelFunc
	if sagaTimeout > 0 {
		ctx, sagaTimeoutCancel = se.coordinator.timeoutDetector.StartSagaTimeout(
			ctx,
			se.instance,
			sagaTimeout,
		)
		defer sagaTimeoutCancel()
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

	// Execute with retry logic using ErrorHandler
	var outputData interface{}
	var lastErr error

	maxAttempts := se.errorHandler.GetMaxAttempts(step)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Update step state to running
		startTime := time.Now()
		stepState.State = saga.StepStateRunning
		stepState.Attempts = attempt
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

		endTime := time.Now()
		duration := endTime.Sub(startTime)

		if err == nil {
			// Step succeeded
			se.errorHandler.RecordAttempt(step, attempt, startTime, endTime, nil, false)

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

			outputData = result
			break
		}

		// Step failed - use ErrorHandler to determine retry strategy
		lastErr = err
		sagaError := se.errorHandler.ClassifyError(err, step)
		stepState.Error = sagaError

		// Determine if we should retry
		shouldRetry, retryDelay := se.errorHandler.HandleStepError(ctx, step, err, attempt)

		// Record the attempt
		se.errorHandler.RecordAttempt(step, attempt, startTime, endTime, err, shouldRetry)

		if shouldRetry && attempt < maxAttempts {
			// Record retry metrics
			se.coordinator.metricsCollector.RecordStepRetried(
				se.definition.GetID(),
				step.GetID(),
				attempt,
			)

			// Wait before retry
			select {
			case <-time.After(retryDelay):
				// Continue to next attempt
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

			// Handle max retries exceeded
			if attempt >= maxAttempts {
				lastErr = se.errorHandler.HandleMaxRetriesExceeded(ctx, step, lastErr, attempt)
			}

			// Publish step failed event
			se.publishStepFailedEvent(ctx, step, stepIndex, sagaError, attempt)

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
	
	// Use timeout detector to track step timeout
	stepIndex := se.instance.GetCurrentStep()
	if timeout > 0 {
		return se.coordinator.timeoutDetector.StartStepTimeout(
			ctx,
			se.instance.GetID(),
			stepIndex,
			step.GetName(),
			timeout,
		)
	}
	
	// No timeout specified, return context without timeout
	return context.WithCancel(ctx)
}

// getMaxAttempts returns the maximum number of attempts for a step.
func (se *stepExecutor) getMaxAttempts(step saga.SagaStep) int {
	return se.errorHandler.GetMaxAttempts(step)
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

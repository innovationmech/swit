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

package state

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

var (
	// ErrRecoverySkipped is returned when recovery is skipped for a Saga.
	ErrRecoverySkipped = errors.New("recovery skipped")

	// ErrNoStrategyAvailable is returned when no recovery strategy is available for a Saga.
	ErrNoStrategyAvailable = errors.New("no recovery strategy available")
)

// recoveryHistory tracks the recovery attempts for a Saga.
type recoveryHistory struct {
	sagaID        string
	attemptCount  int
	lastAttempt   time.Time
	attempts      []recoveryAttemptRecord
	totalDuration time.Duration
}

// recoveryAttemptRecord records a single recovery attempt.
type recoveryAttemptRecord struct {
	attemptTime time.Time
	strategy    RecoveryStrategyType
	result      string
	error       error
	duration    time.Duration
}

// recoverSagaInstance is the main method that attempts to recover a specific Saga instance.
// It replaces the stub implementation in recovery.go.
func (rm *RecoveryManager) recoverSagaInstance(ctx context.Context, sagaID string) error {
	startTime := time.Now()

	rm.logger.Info("attempting to recover saga", zap.String("saga_id", sagaID))

	// Update stats
	rm.statsMu.Lock()
	rm.stats.TotalRecoveryAttempts++
	rm.stats.CurrentlyRecovering++
	rm.statsMu.Unlock()

	defer func() {
		rm.statsMu.Lock()
		rm.stats.CurrentlyRecovering--
		rm.statsMu.Unlock()
	}()

	// Record metrics - will be updated with actual strategy later
	strategy := "unknown"
	sagaType := "unknown"
	if rm.metrics != nil {
		rm.metrics.IncrementRecoveringInProgress(strategy)
		defer rm.metrics.DecrementRecoveringInProgress(strategy)
	}

	// Get Saga instance from storage
	sagaInst, err := rm.stateStorage.GetSaga(ctx, sagaID)
	if err != nil {
		rm.logger.Error("failed to get saga for recovery",
			zap.String("saga_id", sagaID),
			zap.Error(err),
		)
		rm.recordFailedRecovery(sagaID, startTime, err)
		return fmt.Errorf("failed to get saga: %w", err)
	}

	// Check if Saga is already in terminal state
	if sagaInst.GetState().IsTerminal() {
		rm.logger.Debug("saga already in terminal state, skipping recovery",
			zap.String("saga_id", sagaID),
			zap.String("state", sagaInst.GetState().String()),
		)
		rm.recordSkippedRecovery(sagaID, startTime)
		return ErrRecoverySkipped
	}

	// Note: RecoverSaga already checks if the saga is being recovered,
	// so we don't need to check again here

	// Get recovery attempt from the recovering map (set by RecoverSaga)
	// Check if this exceeds max attempts
	rm.recoveringMu.Lock()
	attempt, exists := rm.recovering[sagaID]
	if exists && attempt.attemptCount > rm.config.MaxRecoveryAttempts {
		rm.recoveringMu.Unlock()
		rm.logger.Warn("max recovery attempts exceeded for saga",
			zap.String("saga_id", sagaID),
			zap.Int("attempts", attempt.attemptCount),
		)

		// Emit manual intervention required event
		rm.emitEvent(&RecoveryEvent{
			Type:      RecoveryEventManualInterventionRequired,
			Timestamp: time.Now(),
			SagaID:    sagaID,
			Message:   "Max recovery attempts exceeded, manual intervention required",
			Metadata: map[string]interface{}{
				"attempts": attempt.attemptCount,
				"max":      rm.config.MaxRecoveryAttempts,
			},
		})

		// Mark as failed
		return rm.markSagaAsFailed(ctx, sagaInst, ErrMaxRecoveryAttemptsExceeded)
	}

	// Check recovery backoff
	if exists && !attempt.lastAttempt.IsZero() && attempt.attemptCount > 1 {
		timeSinceLastAttempt := time.Since(attempt.lastAttempt)
		if timeSinceLastAttempt < rm.config.RecoveryBackoff {
			rm.recoveringMu.Unlock()
			rm.logger.Debug("recovery backoff not elapsed, skipping",
				zap.String("saga_id", sagaID),
				zap.Duration("time_since_last_attempt", timeSinceLastAttempt),
				zap.Duration("backoff", rm.config.RecoveryBackoff),
			)
			return ErrRecoverySkipped
		}
	}
	rm.recoveringMu.Unlock()

	// Select recovery strategy
	strategySelector := NewRecoveryStrategySelector()
	selectedStrategy := strategySelector.SelectStrategy(sagaInst)
	if selectedStrategy == nil {
		rm.logger.Warn("no recovery strategy available for saga",
			zap.String("saga_id", sagaID),
			zap.String("state", sagaInst.GetState().String()),
		)
		rm.recordFailedRecovery(sagaID, startTime, ErrNoStrategyAvailable)
		return ErrNoStrategyAvailable
	}

	// Update strategy and saga type for metrics
	strategy = string(selectedStrategy.GetStrategyType())
	sagaType = sagaInst.GetDefinitionID()

	// Record metrics for recovery attempt
	if rm.metrics != nil {
		rm.metrics.RecordRecoveryAttempt(strategy, sagaType)
	}

	rm.logger.Info("selected recovery strategy",
		zap.String("saga_id", sagaID),
		zap.String("strategy", strategy),
		zap.String("description", selectedStrategy.GetDescription()),
	)

	// Execute recovery based on strategy and Saga state
	var recoveryErr error
	switch sagaInst.GetState() {
	case saga.StateRunning, saga.StateStepCompleted:
		recoveryErr = rm.recoverRunningSaga(ctx, sagaInst, selectedStrategy)
	case saga.StateCompensating:
		recoveryErr = rm.recoverCompensatingSaga(ctx, sagaInst)
	default:
		rm.logger.Warn("cannot recover saga in current state",
			zap.String("saga_id", sagaID),
			zap.String("state", sagaInst.GetState().String()),
		)
		recoveryErr = fmt.Errorf("cannot recover saga in state %s", sagaInst.GetState().String())
	}

	duration := time.Since(startTime)

	// Record recovery attempt
	rm.recordRecoveryAttempt(sagaID, selectedStrategy.GetStrategyType(), recoveryErr, duration)

	if recoveryErr != nil {
		rm.logger.Error("saga recovery failed",
			zap.String("saga_id", sagaID),
			zap.String("strategy", strategy),
			zap.Duration("duration", duration),
			zap.Error(recoveryErr),
		)

		rm.recordFailedRecovery(sagaID, startTime, recoveryErr)

		// Record metrics for failure
		if rm.metrics != nil {
			errorType := "unknown"
			if errors.Is(recoveryErr, ErrMaxRecoveryAttemptsExceeded) {
				errorType = "max_attempts_exceeded"
			} else if errors.Is(recoveryErr, ErrNoStrategyAvailable) {
				errorType = "no_strategy"
			} else if errors.Is(recoveryErr, context.DeadlineExceeded) {
				errorType = "timeout"
			} else {
				errorType = "execution_error"
			}
			rm.metrics.RecordRecoveryFailure(strategy, sagaType, errorType, duration)
		}

		rm.emitEvent(&RecoveryEvent{
			Type:      RecoveryEventSagaRecoveryFailed,
			Timestamp: time.Now(),
			SagaID:    sagaID,
			Message:   "Saga recovery failed",
			Error:     recoveryErr,
			Metadata: map[string]interface{}{
				"strategy":    strategy,
				"duration_ms": duration.Milliseconds(),
			},
		})

		return recoveryErr
	}

	// Recovery succeeded
	rm.recordSuccessfulRecovery(sagaID, startTime)

	// Record metrics for success
	if rm.metrics != nil {
		rm.metrics.RecordRecoverySuccess(strategy, sagaType, duration)
	}

	rm.logger.Info("saga recovery completed successfully",
		zap.String("saga_id", sagaID),
		zap.String("strategy", strategy),
		zap.Duration("duration", duration),
	)

	rm.emitEvent(&RecoveryEvent{
		Type:      RecoveryEventSagaRecovered,
		Timestamp: time.Now(),
		SagaID:    sagaID,
		Message:   "Saga recovered successfully",
		Metadata: map[string]interface{}{
			"strategy":    strategy,
			"duration_ms": duration.Milliseconds(),
		},
	})

	return nil
}

// recoverRunningSaga recovers a Saga that is in Running or StepCompleted state.
func (rm *RecoveryManager) recoverRunningSaga(ctx context.Context, sagaInst saga.SagaInstance, strategy RecoveryStrategy) error {
	rm.logger.Info("recovering running saga",
		zap.String("saga_id", sagaInst.GetID()),
		zap.String("state", sagaInst.GetState().String()),
		zap.Int("current_step", sagaInst.GetCurrentStep()),
	)

	// Get step states to check progress
	stepStates, err := rm.stateStorage.GetStepStates(ctx, sagaInst.GetID())
	if err != nil {
		return fmt.Errorf("failed to get step states: %w", err)
	}

	currentStep := sagaInst.GetCurrentStep()

	// Check if current step is valid
	if currentStep < 0 || currentStep >= sagaInst.GetTotalSteps() {
		rm.logger.Warn("invalid current step index",
			zap.String("saga_id", sagaInst.GetID()),
			zap.Int("current_step", currentStep),
			zap.Int("total_steps", sagaInst.GetTotalSteps()),
		)
		return rm.markSagaAsFailed(ctx, sagaInst, errors.New("invalid step index"))
	}

	// Get current step state
	var currentStepState *saga.StepState
	if currentStep < len(stepStates) {
		currentStepState = stepStates[currentStep]
	}

	// Decide recovery action based on strategy and step state
	switch strategy.GetStrategyType() {
	case RecoveryStrategyRetry:
		return rm.retryCurrentStep(ctx, sagaInst, currentStepState)

	case RecoveryStrategyCompensate:
		return strategy.Recover(ctx, sagaInst, rm.coordinator)

	case RecoveryStrategyMarkFailed:
		return rm.markSagaAsFailed(ctx, sagaInst, errors.New("recovery strategy: mark as failed"))

	case RecoveryStrategyContinue:
		return rm.continueNextStep(ctx, sagaInst)

	default:
		return fmt.Errorf("unsupported recovery strategy: %s", strategy.GetStrategyType())
	}
}

// recoverCompensatingSaga recovers a Saga that is in Compensating state.
func (rm *RecoveryManager) recoverCompensatingSaga(ctx context.Context, sagaInst saga.SagaInstance) error {
	rm.logger.Info("recovering compensating saga",
		zap.String("saga_id", sagaInst.GetID()),
		zap.String("state", sagaInst.GetState().String()),
	)

	// Get step states to check compensation progress
	stepStates, err := rm.stateStorage.GetStepStates(ctx, sagaInst.GetID())
	if err != nil {
		return fmt.Errorf("failed to get step states: %w", err)
	}

	// Find incomplete compensation steps
	incompleteSteps := make([]*saga.StepState, 0)
	for _, stepState := range stepStates {
		if stepState.CompensationState != nil {
			switch stepState.CompensationState.State {
			case saga.CompensationStatePending, saga.CompensationStateRunning:
				incompleteSteps = append(incompleteSteps, stepState)
			case saga.CompensationStateFailed:
				// Check if we should retry compensation
				if stepState.CompensationState.Attempts < stepState.CompensationState.MaxAttempts {
					incompleteSteps = append(incompleteSteps, stepState)
				}
			}
		}
	}

	if len(incompleteSteps) == 0 {
		// All compensation steps completed, update Saga state
		rm.logger.Info("all compensation steps completed, updating saga state",
			zap.String("saga_id", sagaInst.GetID()),
		)
		return rm.updateSagaState(ctx, sagaInst.GetID(), saga.StateCompensated)
	}

	// Trigger compensation retry via coordinator
	// Note: The coordinator will handle the actual compensation logic
	rm.logger.Info("retriggering compensation via coordinator",
		zap.String("saga_id", sagaInst.GetID()),
		zap.Int("incomplete_steps", len(incompleteSteps)),
	)

	// We can't directly trigger compensation, so we cancel the saga which will trigger it
	return rm.coordinator.CancelSaga(ctx, sagaInst.GetID(), "Recovery retriggering compensation")
}

// retryCurrentStep retries the current step execution.
func (rm *RecoveryManager) retryCurrentStep(ctx context.Context, sagaInst saga.SagaInstance, stepState *saga.StepState) error {
	currentStep := sagaInst.GetCurrentStep()

	// If stepState is nil, get it from storage
	if stepState == nil {
		stepStates, err := rm.stateStorage.GetStepStates(ctx, sagaInst.GetID())
		if err != nil {
			return fmt.Errorf("failed to get step states: %w", err)
		}
		if currentStep >= 0 && currentStep < len(stepStates) {
			stepState = stepStates[currentStep]
		}
	}

	if stepState == nil {
		// Create a new step state if none exists
		now := time.Now()
		stepState = &saga.StepState{
			ID:            fmt.Sprintf("%s-step-%d", sagaInst.GetID(), currentStep),
			SagaID:        sagaInst.GetID(),
			StepIndex:     currentStep,
			Name:          fmt.Sprintf("step-%d", currentStep),
			State:         saga.StepStatePending,
			Attempts:      0,
			CreatedAt:     now,
			LastAttemptAt: &now,
		}
	}

	rm.logger.Info("retrying current step",
		zap.String("saga_id", sagaInst.GetID()),
		zap.Int("step_index", stepState.StepIndex),
		zap.String("step_name", stepState.Name),
		zap.Int("attempts", stepState.Attempts),
	)

	// Update step state to pending for retry
	now := time.Now()
	stepState.State = saga.StepStatePending
	stepState.Attempts++
	stepState.LastAttemptAt = &now

	// Save updated step state
	if err := rm.stateStorage.SaveStepState(ctx, sagaInst.GetID(), stepState); err != nil {
		return fmt.Errorf("failed to save step state for retry: %w", err)
	}

	// Update Saga state to running
	if err := rm.updateSagaState(ctx, sagaInst.GetID(), saga.StateRunning); err != nil {
		return fmt.Errorf("failed to update saga state: %w", err)
	}

	rm.logger.Info("step marked for retry",
		zap.String("saga_id", sagaInst.GetID()),
		zap.Int("step_index", stepState.StepIndex),
	)

	// Note: The actual step execution will be triggered by the coordinator's main loop
	// or by an external trigger. We just reset the state to allow retry.
	return nil
}

// continueNextStep marks the current step as completed and moves to the next step.
func (rm *RecoveryManager) continueNextStep(ctx context.Context, sagaInst saga.SagaInstance) error {
	currentStep := sagaInst.GetCurrentStep()
	nextStep := currentStep + 1

	if nextStep >= sagaInst.GetTotalSteps() {
		// All steps completed
		return rm.updateSagaState(ctx, sagaInst.GetID(), saga.StateCompleted)
	}

	rm.logger.Info("continuing to next step",
		zap.String("saga_id", sagaInst.GetID()),
		zap.Int("current_step", currentStep),
		zap.Int("next_step", nextStep),
	)

	// Update Saga to continue with next step
	// This is a simplified approach - actual implementation may need to update more fields
	return rm.updateSagaState(ctx, sagaInst.GetID(), saga.StateRunning)
}

// markSagaAsFailed marks a Saga as failed and updates its state.
func (rm *RecoveryManager) markSagaAsFailed(ctx context.Context, sagaInst saga.SagaInstance, err error) error {
	rm.logger.Warn("marking saga as failed",
		zap.String("saga_id", sagaInst.GetID()),
		zap.Error(err),
	)

	metadata := map[string]interface{}{
		"recovery_failed": true,
		"failure_reason":  err.Error(),
		"failed_at":       time.Now().Format(time.RFC3339),
	}

	return rm.stateStorage.UpdateSagaState(ctx, sagaInst.GetID(), saga.StateFailed, metadata)
}

// updateSagaState updates the Saga state in storage.
func (rm *RecoveryManager) updateSagaState(ctx context.Context, sagaID string, state saga.SagaState) error {
	metadata := map[string]interface{}{
		"recovered_at": time.Now().Format(time.RFC3339),
		"recovered_by": "recovery_manager",
	}

	return rm.stateStorage.UpdateSagaState(ctx, sagaID, state, metadata)
}

// isAlreadyRecovering checks if a Saga is currently being recovered.
func (rm *RecoveryManager) isAlreadyRecovering(sagaID string) bool {
	rm.recoveringMu.RLock()
	defer rm.recoveringMu.RUnlock()

	_, exists := rm.recovering[sagaID]
	return exists
}

// getRecoveryHistory retrieves the recovery history for a Saga.
func (rm *RecoveryManager) getRecoveryHistory(sagaID string) *recoveryHistory {
	// In a production system, this would be persisted in storage
	// For now, we check the in-memory recovering map
	rm.recoveringMu.RLock()
	defer rm.recoveringMu.RUnlock()

	if attempt, exists := rm.recovering[sagaID]; exists {
		return &recoveryHistory{
			sagaID:       sagaID,
			attemptCount: attempt.attemptCount,
			lastAttempt:  attempt.lastAttempt,
		}
	}

	return nil
}

// recordRecoveryAttempt records a recovery attempt in history.
func (rm *RecoveryManager) recordRecoveryAttempt(sagaID string, strategy RecoveryStrategyType, err error, duration time.Duration) {
	rm.recoveringMu.Lock()
	defer rm.recoveringMu.Unlock()

	attempt, exists := rm.recovering[sagaID]
	if !exists {
		attempt = &recoveryAttempt{
			sagaID:      sagaID,
			startTime:   time.Now(),
			lastAttempt: time.Now(),
		}
		rm.recovering[sagaID] = attempt
	}

	attempt.attemptCount++
	attempt.lastAttempt = time.Now()

	// In a production system, we would persist this to storage
	rm.logger.Debug("recorded recovery attempt",
		zap.String("saga_id", sagaID),
		zap.String("strategy", string(strategy)),
		zap.Int("attempt_count", attempt.attemptCount),
		zap.Duration("duration", duration),
		zap.Bool("success", err == nil),
	)
}

// recordSuccessfulRecovery updates stats for a successful recovery.
func (rm *RecoveryManager) recordSuccessfulRecovery(sagaID string, startTime time.Time) {
	duration := time.Since(startTime)

	rm.statsMu.Lock()
	rm.stats.SuccessfulRecoveries++
	rm.stats.LastRecoveryTime = time.Now()

	// Update average duration
	if rm.stats.SuccessfulRecoveries > 0 {
		totalDuration := rm.stats.AverageRecoveryDuration * time.Duration(rm.stats.SuccessfulRecoveries-1)
		rm.stats.AverageRecoveryDuration = (totalDuration + duration) / time.Duration(rm.stats.SuccessfulRecoveries)
	}
	rm.statsMu.Unlock()
}

// recordFailedRecovery updates stats for a failed recovery.
func (rm *RecoveryManager) recordFailedRecovery(sagaID string, startTime time.Time, err error) {
	rm.statsMu.Lock()
	rm.stats.FailedRecoveries++
	rm.statsMu.Unlock()
}

// recordSkippedRecovery updates stats for a skipped recovery.
func (rm *RecoveryManager) recordSkippedRecovery(sagaID string, startTime time.Time) {
	// Skipped recoveries don't update the main counters
	rm.logger.Debug("recovery skipped",
		zap.String("saga_id", sagaID),
	)
}

// performRecoveryBatch processes a batch of detected Sagas for recovery.
// It respects concurrency limits and implements exponential backoff for retries.
func (rm *RecoveryManager) performRecoveryBatch(ctx context.Context, detectionResults []*DetectionResult) error {
	if len(detectionResults) == 0 {
		return nil
	}

	// Limit batch size
	batchSize := rm.config.RecoveryBatchSize
	if len(detectionResults) > batchSize {
		detectionResults = detectionResults[:batchSize]
	}

	rm.logger.Info("processing recovery batch",
		zap.Int("batch_size", len(detectionResults)),
	)

	// Process each detected Saga
	for _, result := range detectionResults {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Acquire semaphore for concurrency control
		select {
		case rm.semaphore <- struct{}{}:
			// Got semaphore, proceed with recovery
			go func(sagaID string, priority DetectionPriority) {
				defer func() { <-rm.semaphore }()

				recoveryCtx, cancel := context.WithTimeout(ctx, rm.config.RecoveryTimeout)
				defer cancel()

				if err := rm.RecoverSaga(recoveryCtx, sagaID); err != nil {
					if errors.Is(err, ErrRecoverySkipped) || errors.Is(err, ErrRecoveryInProgress) {
						// Expected errors, log at debug level
						rm.logger.Debug("saga recovery skipped",
							zap.String("saga_id", sagaID),
							zap.Error(err),
						)
					} else {
						rm.logger.Error("saga recovery failed in batch",
							zap.String("saga_id", sagaID),
							zap.Int("priority", int(priority)),
							zap.Error(err),
						)
					}
				}
			}(result.SagaID, result.Priority)

		case <-time.After(5 * time.Second):
			// Couldn't acquire semaphore within timeout, skip this Saga
			rm.logger.Warn("skipping saga recovery due to concurrency limit",
				zap.String("saga_id", result.SagaID),
			)
			continue
		}
	}

	return nil
}

// calculateBackoff calculates exponential backoff duration.
func (rm *RecoveryManager) calculateBackoff(attemptCount int) time.Duration {
	baseBackoff := rm.config.RecoveryBackoff
	maxBackoff := 30 * time.Minute

	// Exponential backoff: baseBackoff * 2^attemptCount
	backoff := time.Duration(float64(baseBackoff) * math.Pow(2, float64(attemptCount)))

	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}

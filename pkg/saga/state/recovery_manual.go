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
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

var (
	// ErrEmptySagaIDs is returned when an empty saga ID list is provided.
	ErrEmptySagaIDs = errors.New("saga IDs list cannot be empty")

	// ErrInvalidStepIndex is returned when an invalid step index is provided.
	ErrInvalidStepIndex = errors.New("invalid step index")
)

// RecoveryOperationType represents the type of manual recovery operation.
type RecoveryOperationType string

const (
	// OperationRecover is a manual recovery operation.
	OperationRecover RecoveryOperationType = "recover"

	// OperationForceCompensate is a forced compensation operation.
	OperationForceCompensate RecoveryOperationType = "force_compensate"

	// OperationMarkFailed is a mark-as-failed operation.
	OperationMarkFailed RecoveryOperationType = "mark_failed"

	// OperationRetry is a retry from specific step operation.
	OperationRetry RecoveryOperationType = "retry"
)

// ManualOperation records a manual recovery operation.
type ManualOperation struct {
	OperationID   string                `json:"operation_id"`
	OperationType RecoveryOperationType `json:"operation_type"`
	SagaID        string                `json:"saga_id"`
	Timestamp     time.Time             `json:"timestamp"`
	Operator      string                `json:"operator,omitempty"`
	Reason        string                `json:"reason,omitempty"`
	FromStep      int                   `json:"from_step,omitempty"`
	Success       bool                  `json:"success"`
	Error         string                `json:"error,omitempty"`
	Duration      time.Duration         `json:"duration"`
}

// RecoveryBatchResult holds the result of a batch recovery operation.
type RecoveryBatchResult struct {
	TotalCount    int                `json:"total_count"`
	SuccessCount  int                `json:"success_count"`
	FailedCount   int                `json:"failed_count"`
	SkippedCount  int                `json:"skipped_count"`
	Results       map[string]error   `json:"results"`
	StartTime     time.Time          `json:"start_time"`
	EndTime       time.Time          `json:"end_time"`
	TotalDuration time.Duration      `json:"total_duration"`
	Operations    []*ManualOperation `json:"operations,omitempty"`
}

// RecoverSagaBatch attempts to recover multiple Saga instances in batch.
// It processes recoveries concurrently while respecting the max concurrent recoveries limit.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaIDs: The IDs of the Sagas to recover.
//
// Returns:
//   - *RecoveryBatchResult: The result of the batch operation.
//   - error: An error if the batch operation setup fails.
func (rm *RecoveryManager) RecoverSagaBatch(ctx context.Context, sagaIDs []string) (*RecoveryBatchResult, error) {
	if rm.closed.Load() {
		return nil, ErrRecoveryManagerClosed
	}

	if len(sagaIDs) == 0 {
		return nil, ErrEmptySagaIDs
	}

	rm.logger.Info("batch recovery triggered",
		zap.Int("saga_count", len(sagaIDs)),
	)

	result := &RecoveryBatchResult{
		TotalCount: len(sagaIDs),
		Results:    make(map[string]error),
		StartTime:  time.Now(),
		Operations: make([]*ManualOperation, 0),
	}

	// Use a WaitGroup to track completion
	var wg sync.WaitGroup
	var resultMu sync.Mutex

	// Process each saga
	for _, sagaID := range sagaIDs {
		wg.Add(1)

		// Acquire semaphore to limit concurrency
		rm.semaphore <- struct{}{}

		go func(id string) {
			defer wg.Done()
			defer func() { <-rm.semaphore }()

			opStart := time.Now()
			op := &ManualOperation{
				OperationID:   fmt.Sprintf("%s-%d", id, time.Now().UnixNano()),
				OperationType: OperationRecover,
				SagaID:        id,
				Timestamp:     opStart,
			}

			err := rm.RecoverSaga(ctx, id)

			op.Duration = time.Since(opStart)
			op.Success = err == nil
			if err != nil {
				op.Error = err.Error()
			}

			resultMu.Lock()
			result.Results[id] = err
			result.Operations = append(result.Operations, op)
			if err == nil {
				result.SuccessCount++
			} else if errors.Is(err, ErrRecoverySkipped) {
				result.SkippedCount++
			} else {
				result.FailedCount++
			}
			resultMu.Unlock()

			rm.logger.Debug("batch recovery attempt completed",
				zap.String("saga_id", id),
				zap.Bool("success", err == nil),
				zap.Duration("duration", op.Duration),
			)
		}(sagaID)
	}

	// Wait for all recoveries to complete
	wg.Wait()

	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)

	// Record the batch operation
	rm.recordManualOperation(result.Operations...)

	rm.logger.Info("batch recovery completed",
		zap.Int("total", result.TotalCount),
		zap.Int("success", result.SuccessCount),
		zap.Int("failed", result.FailedCount),
		zap.Int("skipped", result.SkippedCount),
		zap.Duration("duration", result.TotalDuration),
	)

	return result, nil
}

// ForceCompensate forces a Saga to start compensation immediately.
// This is useful when a Saga is stuck and needs manual intervention.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaID: The ID of the Saga to compensate.
//   - reason: The reason for forcing compensation (for audit purposes).
//
// Returns:
//   - error: An error if the operation fails.
func (rm *RecoveryManager) ForceCompensate(ctx context.Context, sagaID string, reason string) error {
	if rm.closed.Load() {
		return ErrRecoveryManagerClosed
	}

	opStart := time.Now()
	rm.logger.Info("force compensate triggered",
		zap.String("saga_id", sagaID),
		zap.String("reason", reason),
	)

	op := &ManualOperation{
		OperationID:   fmt.Sprintf("force-comp-%s-%d", sagaID, time.Now().UnixNano()),
		OperationType: OperationForceCompensate,
		SagaID:        sagaID,
		Timestamp:     opStart,
		Reason:        reason,
	}

	// Get Saga instance
	sagaInst, err := rm.stateStorage.GetSaga(ctx, sagaID)
	if err != nil {
		op.Duration = time.Since(opStart)
		op.Success = false
		op.Error = err.Error()
		rm.recordManualOperation(op)
		return fmt.Errorf("failed to get saga: %w", err)
	}

	// Check if already in terminal state
	if sagaInst.GetState().IsTerminal() {
		op.Duration = time.Since(opStart)
		op.Success = false
		op.Error = "saga already in terminal state"
		rm.recordManualOperation(op)
		return errors.New("saga already in terminal state")
	}

	// Check if already compensating
	if sagaInst.GetState() == saga.StateCompensating {
		rm.logger.Info("saga already compensating, retriggering via cancel",
			zap.String("saga_id", sagaID),
		)
	}

	// Trigger compensation by canceling the saga
	err = rm.coordinator.CancelSaga(ctx, sagaID, reason)

	op.Duration = time.Since(opStart)
	op.Success = err == nil
	if err != nil {
		op.Error = err.Error()
		rm.recordManualOperation(op)
		return fmt.Errorf("failed to force compensate: %w", err)
	}

	rm.recordManualOperation(op)

	// Record manual intervention metric
	if rm.metrics != nil {
		rm.metrics.RecordManualIntervention()
	}

	rm.logger.Info("force compensate completed",
		zap.String("saga_id", sagaID),
		zap.Duration("duration", op.Duration),
	)

	return nil
}

// MarkAsFailed marks a Saga as failed with a specific reason.
// This is useful when manual investigation determines a Saga cannot be recovered.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaID: The ID of the Saga to mark as failed.
//   - reason: The reason for marking as failed (for audit purposes).
//
// Returns:
//   - error: An error if the operation fails.
func (rm *RecoveryManager) MarkAsFailed(ctx context.Context, sagaID string, reason string) error {
	if rm.closed.Load() {
		return ErrRecoveryManagerClosed
	}

	opStart := time.Now()
	rm.logger.Info("mark as failed triggered",
		zap.String("saga_id", sagaID),
		zap.String("reason", reason),
	)

	op := &ManualOperation{
		OperationID:   fmt.Sprintf("mark-failed-%s-%d", sagaID, time.Now().UnixNano()),
		OperationType: OperationMarkFailed,
		SagaID:        sagaID,
		Timestamp:     opStart,
		Reason:        reason,
	}

	// Get Saga instance
	sagaInst, err := rm.stateStorage.GetSaga(ctx, sagaID)
	if err != nil {
		op.Duration = time.Since(opStart)
		op.Success = false
		op.Error = err.Error()
		rm.recordManualOperation(op)
		return fmt.Errorf("failed to get saga: %w", err)
	}

	// Check if already in terminal state
	if sagaInst.GetState().IsTerminal() {
		op.Duration = time.Since(opStart)
		op.Success = false
		op.Error = "saga already in terminal state"
		rm.recordManualOperation(op)
		return errors.New("saga already in terminal state")
	}

	// Mark as failed
	err = rm.markSagaAsFailed(ctx, sagaInst, errors.New(reason))

	op.Duration = time.Since(opStart)
	op.Success = err == nil
	if err != nil {
		op.Error = err.Error()
		rm.recordManualOperation(op)
		return fmt.Errorf("failed to mark as failed: %w", err)
	}

	rm.recordManualOperation(op)

	// Record manual intervention metric
	if rm.metrics != nil {
		rm.metrics.RecordManualIntervention()
	}

	rm.logger.Info("mark as failed completed",
		zap.String("saga_id", sagaID),
		zap.Duration("duration", op.Duration),
	)

	return nil
}

// RetrySaga retries a Saga from a specific step.
// This is useful when a step failed and you want to retry from that point.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaID: The ID of the Saga to retry.
//   - fromStep: The step index to retry from (0-based).
//
// Returns:
//   - error: An error if the operation fails.
func (rm *RecoveryManager) RetrySaga(ctx context.Context, sagaID string, fromStep int) error {
	if rm.closed.Load() {
		return ErrRecoveryManagerClosed
	}

	if fromStep < 0 {
		return ErrInvalidStepIndex
	}

	opStart := time.Now()
	rm.logger.Info("retry saga triggered",
		zap.String("saga_id", sagaID),
		zap.Int("from_step", fromStep),
	)

	op := &ManualOperation{
		OperationID:   fmt.Sprintf("retry-%s-%d", sagaID, time.Now().UnixNano()),
		OperationType: OperationRetry,
		SagaID:        sagaID,
		Timestamp:     opStart,
		FromStep:      fromStep,
	}

	// Get Saga instance
	sagaInst, err := rm.stateStorage.GetSaga(ctx, sagaID)
	if err != nil {
		op.Duration = time.Since(opStart)
		op.Success = false
		op.Error = err.Error()
		rm.recordManualOperation(op)
		return fmt.Errorf("failed to get saga: %w", err)
	}

	// Check if already in terminal state
	if sagaInst.GetState().IsTerminal() {
		op.Duration = time.Since(opStart)
		op.Success = false
		op.Error = "saga already in terminal state"
		rm.recordManualOperation(op)
		return errors.New("saga already in terminal state")
	}

	// Validate step index
	if fromStep >= sagaInst.GetTotalSteps() {
		op.Duration = time.Since(opStart)
		op.Success = false
		op.Error = fmt.Sprintf("step index %d out of range (total steps: %d)", fromStep, sagaInst.GetTotalSteps())
		rm.recordManualOperation(op)
		return fmt.Errorf("%w: step %d out of range (total: %d)", ErrInvalidStepIndex, fromStep, sagaInst.GetTotalSteps())
	}

	// Update the saga state with the new step index
	// We use metadata to store the recovery information
	metadata := map[string]interface{}{
		"recovery_from_step": fromStep,
		"recovery_timestamp": time.Now(),
		"recovery_reason":    "manual_retry",
	}

	// Update to Running state if not already
	targetState := saga.StateRunning
	if sagaInst.GetState() == saga.StateRunning {
		targetState = saga.StateStepCompleted
	}

	if err := rm.stateStorage.UpdateSagaState(ctx, sagaID, targetState, metadata); err != nil {
		op.Duration = time.Since(opStart)
		op.Success = false
		op.Error = err.Error()
		rm.recordManualOperation(op)
		return fmt.Errorf("failed to update saga: %w", err)
	}

	// Note: In a real implementation, the coordinator would need a method to continue
	// execution from a specific step. For now, we just update the state and log.
	rm.logger.Info("saga state updated for retry",
		zap.String("saga_id", sagaID),
		zap.Int("from_step", fromStep),
		zap.String("new_state", targetState.String()),
	)

	op.Duration = time.Since(opStart)
	op.Success = true

	rm.recordManualOperation(op)

	// Record manual intervention metric
	if rm.metrics != nil {
		rm.metrics.RecordManualIntervention()
	}

	rm.logger.Info("retry saga completed",
		zap.String("saga_id", sagaID),
		zap.Int("from_step", fromStep),
		zap.Duration("duration", op.Duration),
	)

	return nil
}

// recordManualOperation records manual operations for audit purposes.
func (rm *RecoveryManager) recordManualOperation(operations ...*ManualOperation) {
	if len(operations) == 0 {
		return
	}

	rm.recoveringMu.Lock()
	defer rm.recoveringMu.Unlock()

	// In a production system, this would write to an audit log or database
	// For now, we just log it
	for _, op := range operations {
		rm.logger.Info("manual operation recorded",
			zap.String("operation_id", op.OperationID),
			zap.String("operation_type", string(op.OperationType)),
			zap.String("saga_id", op.SagaID),
			zap.Time("timestamp", op.Timestamp),
			zap.String("operator", op.Operator),
			zap.String("reason", op.Reason),
			zap.Bool("success", op.Success),
			zap.String("error", op.Error),
			zap.Duration("duration", op.Duration),
		)
	}
}

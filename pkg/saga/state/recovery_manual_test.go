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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newMockCoordinator creates a simple mock coordinator for testing
func newMockCoordinator() *mockCoordinator {
	return &mockCoordinator{
		cancelSagaFunc: func(ctx context.Context, sagaID string, reason string) error {
			return nil
		},
	}
}

func TestRecoverSagaBatch(t *testing.T) {
	storage := newMockStateStorage()
	coordinator := newMockCoordinator()
	rm, err := NewRecoveryManager(nil, storage, coordinator, zap.NewNop())
	require.NoError(t, err)

	// Add some test sagas
	saga1 := &mockSagaInstance{id: "saga-1", state: saga.StateRunning, totalSteps: 3}
	saga2 := &mockSagaInstance{id: "saga-2", state: saga.StateRunning, totalSteps: 3}
	saga3 := &mockSagaInstance{id: "saga-3", state: saga.StateFailed, totalSteps: 3}

	storage.sagas["saga-1"] = saga1
	storage.sagas["saga-2"] = saga2
	storage.sagas["saga-3"] = saga3

	ctx := context.Background()

	result, err := rm.RecoverSagaBatch(ctx, []string{"saga-1", "saga-2", "saga-3"})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.TotalCount)
	assert.True(t, result.SuccessCount+result.FailedCount+result.SkippedCount == result.TotalCount)
}

func TestRecoverSagaBatch_EmptySagaIDs(t *testing.T) {
	storage := newMockStateStorage()
	coordinator := newMockCoordinator()
	rm, err := NewRecoveryManager(nil, storage, coordinator, zap.NewNop())
	require.NoError(t, err)

	ctx := context.Background()

	result, err := rm.RecoverSagaBatch(ctx, []string{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptySagaIDs)
	assert.Nil(t, result)
}

func TestForceCompensate(t *testing.T) {
	storage := newMockStateStorage()
	coordinator := newMockCoordinator()
	rm, err := NewRecoveryManager(nil, storage, coordinator, zap.NewNop())
	require.NoError(t, err)

	// Create a running saga
	sagaInst := &mockSagaInstance{id: "saga-1", state: saga.StateRunning, totalSteps: 3}
	storage.sagas["saga-1"] = sagaInst

	ctx := context.Background()

	err = rm.ForceCompensate(ctx, "saga-1", "manual compensation")
	assert.NoError(t, err)
}

func TestForceCompensate_TerminalState(t *testing.T) {
	storage := newMockStateStorage()
	coordinator := newMockCoordinator()
	rm, err := NewRecoveryManager(nil, storage, coordinator, zap.NewNop())
	require.NoError(t, err)

	// Create a completed saga
	sagaInst := &mockSagaInstance{id: "saga-1", state: saga.StateCompleted, totalSteps: 3}
	storage.sagas["saga-1"] = sagaInst

	ctx := context.Background()

	err = rm.ForceCompensate(ctx, "saga-1", "manual compensation")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "terminal state")
}

func TestMarkAsFailed(t *testing.T) {
	storage := newMockStateStorage()
	coordinator := newMockCoordinator()
	rm, err := NewRecoveryManager(nil, storage, coordinator, zap.NewNop())
	require.NoError(t, err)

	// Create a running saga
	sagaInst := &mockSagaInstance{id: "saga-1", state: saga.StateRunning, totalSteps: 3}
	storage.sagas["saga-1"] = sagaInst

	ctx := context.Background()

	err = rm.MarkAsFailed(ctx, "saga-1", "manual mark as failed")
	assert.NoError(t, err)

	// Verify state was updated
	updated, err := storage.GetSaga(ctx, "saga-1")
	require.NoError(t, err)
	assert.Equal(t, saga.StateFailed, updated.GetState())
}

func TestMarkAsFailed_TerminalState(t *testing.T) {
	storage := newMockStateStorage()
	coordinator := newMockCoordinator()
	rm, err := NewRecoveryManager(nil, storage, coordinator, zap.NewNop())
	require.NoError(t, err)

	// Create a completed saga
	sagaInst := &mockSagaInstance{id: "saga-1", state: saga.StateCompleted, totalSteps: 3}
	storage.sagas["saga-1"] = sagaInst

	ctx := context.Background()

	err = rm.MarkAsFailed(ctx, "saga-1", "manual mark as failed")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "terminal state")
}

func TestRetrySaga(t *testing.T) {
	storage := newMockStateStorage()
	coordinator := newMockCoordinator()
	rm, err := NewRecoveryManager(nil, storage, coordinator, zap.NewNop())
	require.NoError(t, err)

	// Create a running saga with multiple steps
	sagaInst := &mockSagaInstance{id: "saga-1", state: saga.StateRunning, currentStep: 2, totalSteps: 5}
	storage.sagas["saga-1"] = sagaInst

	ctx := context.Background()

	err = rm.RetrySaga(ctx, "saga-1", 1)
	assert.NoError(t, err)
}

func TestRetrySaga_InvalidStepIndex(t *testing.T) {
	storage := newMockStateStorage()
	coordinator := newMockCoordinator()
	rm, err := NewRecoveryManager(nil, storage, coordinator, zap.NewNop())
	require.NoError(t, err)

	// Create a running saga with 5 steps
	sagaInst := &mockSagaInstance{id: "saga-1", state: saga.StateRunning, totalSteps: 5}
	storage.sagas["saga-1"] = sagaInst

	ctx := context.Background()

	// Try to retry from invalid step
	err = rm.RetrySaga(ctx, "saga-1", 10)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidStepIndex)
}

func TestRetrySaga_NegativeStepIndex(t *testing.T) {
	storage := newMockStateStorage()
	coordinator := newMockCoordinator()
	rm, err := NewRecoveryManager(nil, storage, coordinator, zap.NewNop())
	require.NoError(t, err)

	ctx := context.Background()

	err = rm.RetrySaga(ctx, "saga-1", -1)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidStepIndex)
}

func TestRecoveryBatchResult(t *testing.T) {
	result := &RecoveryBatchResult{
		TotalCount:   3,
		SuccessCount: 2,
		FailedCount:  1,
		Results:      make(map[string]error),
		StartTime:    time.Now(),
		Operations:   make([]*ManualOperation, 0),
	}

	result.Results["saga-1"] = nil
	result.Results["saga-2"] = nil
	result.Results["saga-3"] = assert.AnError

	assert.Equal(t, 3, len(result.Results))
	assert.Equal(t, 2, result.SuccessCount)
	assert.Equal(t, 1, result.FailedCount)
}

func TestManualOperation(t *testing.T) {
	op := &ManualOperation{
		OperationID:   "op-1",
		OperationType: OperationRecover,
		SagaID:        "saga-1",
		Timestamp:     time.Now(),
		Operator:      "admin",
		Reason:        "manual recovery",
		Success:       true,
		Duration:      100 * time.Millisecond,
	}

	assert.Equal(t, "op-1", op.OperationID)
	assert.Equal(t, OperationRecover, op.OperationType)
	assert.True(t, op.Success)
	assert.Equal(t, "saga-1", op.SagaID)
}

func TestRecoveryOperationType_Constants(t *testing.T) {
	assert.Equal(t, RecoveryOperationType("recover"), OperationRecover)
	assert.Equal(t, RecoveryOperationType("force_compensate"), OperationForceCompensate)
	assert.Equal(t, RecoveryOperationType("mark_failed"), OperationMarkFailed)
	assert.Equal(t, RecoveryOperationType("retry"), OperationRetry)
}

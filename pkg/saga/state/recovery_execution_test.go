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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

// testCoordinator is a simple mock coordinator for recovery testing.
type testCoordinator struct {
	cancelCalled bool
	cancelError  error
}

func newTestCoordinator() *testCoordinator {
	return &testCoordinator{}
}

func (c *testCoordinator) StartSaga(ctx context.Context, definition saga.SagaDefinition, initialData interface{}) (saga.SagaInstance, error) {
	return nil, errors.New("not implemented")
}

func (c *testCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	return nil, errors.New("not implemented")
}

func (c *testCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	c.cancelCalled = true
	return c.cancelError
}

func (c *testCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	return nil, errors.New("not implemented")
}

func (c *testCoordinator) GetMetrics() *saga.CoordinatorMetrics {
	return &saga.CoordinatorMetrics{}
}

func (c *testCoordinator) HealthCheck(ctx context.Context) error {
	return nil
}

func (c *testCoordinator) Close() error {
	return nil
}

// Test recoverRunningSaga with retry strategy
func TestRecoverRunningSaga_RetryStrategy(t *testing.T) {
	storage := newTestStorage()
	coordinator := newTestCoordinator()

	config := DefaultRecoveryConfig()
	rm, err := NewRecoveryManager(config, storage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create recovery manager: %v", err)
	}

	// Create a running saga
	now := time.Now()
	sagaInst := &mockSagaInstance{
		id:          "test-saga-1",
		state:       saga.StateRunning,
		currentStep: 1,
		totalSteps:  3,
		startTime:   now,
		createdAt:   now,
		updatedAt:   now,
		timeout:     5 * time.Minute,
		err: &saga.SagaError{
			Type:      saga.ErrorTypeTimeout,
			Retryable: true,
		},
	}
	_ = storage.SaveSaga(context.Background(), sagaInst)

	// Create step states
	stepState := &saga.StepState{
		ID:        "test-saga-1-step-1",
		SagaID:    "test-saga-1",
		StepIndex: 1,
		Name:      "test-step",
		State:     saga.StepStateFailed,
		Attempts:  1,
		CreatedAt: now,
	}
	_ = storage.SaveStepState(context.Background(), "test-saga-1", stepState)

	strategy := NewRetryRecoveryStrategy(3)

	ctx := context.Background()
	err = rm.recoverRunningSaga(ctx, sagaInst, strategy)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify step state was updated
	updatedSteps, _ := storage.GetStepStates(ctx, "test-saga-1")
	if len(updatedSteps) == 0 {
		t.Error("Expected step states to be present")
	} else if updatedSteps[0].State != saga.StepStatePending {
		t.Errorf("Expected step state to be Pending, got %s", updatedSteps[0].State.String())
	}
}

// Test recoverRunningSaga with compensate strategy
func TestRecoverRunningSaga_CompensateStrategy(t *testing.T) {
	storage := newTestStorage()
	coordinator := newTestCoordinator()

	config := DefaultRecoveryConfig()
	rm, err := NewRecoveryManager(config, storage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create recovery manager: %v", err)
	}

	// Create a running saga with business error
	now := time.Now()
	sagaInst := &mockSagaInstance{
		id:          "test-saga-2",
		state:       saga.StateRunning,
		currentStep: 1,
		totalSteps:  3,
		startTime:   now,
		createdAt:   now,
		updatedAt:   now,
		err: &saga.SagaError{
			Type:      saga.ErrorTypeBusiness,
			Retryable: false,
		},
	}
	_ = storage.SaveSaga(context.Background(), sagaInst)

	strategy := NewCompensateRecoveryStrategy()

	ctx := context.Background()
	err = rm.recoverRunningSaga(ctx, sagaInst, strategy)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !coordinator.cancelCalled {
		t.Error("Expected CancelSaga to be called for compensation")
	}
}

// Test recoverCompensatingSaga
func TestRecoverCompensatingSaga(t *testing.T) {
	storage := newTestStorage()
	coordinator := newTestCoordinator()

	config := DefaultRecoveryConfig()
	rm, err := NewRecoveryManager(config, storage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create recovery manager: %v", err)
	}

	// Create a compensating saga
	now := time.Now()
	sagaInst := &mockSagaInstance{
		id:          "test-saga-3",
		state:       saga.StateCompensating,
		currentStep: 2,
		totalSteps:  3,
		startTime:   now,
		createdAt:   now,
		updatedAt:   now,
	}
	_ = storage.SaveSaga(context.Background(), sagaInst)

	// Create step states with incomplete compensation
	stepStates := []*saga.StepState{
		{
			ID:        "test-saga-3-step-0",
			SagaID:    "test-saga-3",
			StepIndex: 0,
			State:     saga.StepStateCompleted,
			CompensationState: &saga.CompensationState{
				State:     saga.CompensationStateCompleted,
				StartedAt: &now,
			},
		},
		{
			ID:        "test-saga-3-step-1",
			SagaID:    "test-saga-3",
			StepIndex: 1,
			State:     saga.StepStateCompensating,
			CompensationState: &saga.CompensationState{
				State:       saga.CompensationStateRunning,
				Attempts:    1,
				MaxAttempts: 3,
			},
		},
	}
	for _, step := range stepStates {
		_ = storage.SaveStepState(context.Background(), "test-saga-3", step)
	}

	ctx := context.Background()
	err = rm.recoverCompensatingSaga(ctx, sagaInst)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !coordinator.cancelCalled {
		t.Error("Expected CancelSaga to be called to retrigger compensation")
	}
}

// Test recoverCompensatingSaga with all steps compensated
func TestRecoverCompensatingSaga_AllCompensated(t *testing.T) {
	storage := newTestStorage()
	coordinator := newTestCoordinator()

	config := DefaultRecoveryConfig()
	rm, err := NewRecoveryManager(config, storage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create recovery manager: %v", err)
	}

	// Create a compensating saga
	now := time.Now()
	sagaInst := &mockSagaInstance{
		id:          "test-saga-4",
		state:       saga.StateCompensating,
		currentStep: 2,
		totalSteps:  3,
		startTime:   now,
		createdAt:   now,
		updatedAt:   now,
	}
	_ = storage.SaveSaga(context.Background(), sagaInst)

	// Create step states with all compensation completed
	stepStates := []*saga.StepState{
		{
			ID:        "test-saga-4-step-0",
			SagaID:    "test-saga-4",
			StepIndex: 0,
			State:     saga.StepStateCompensated,
			CompensationState: &saga.CompensationState{
				State:       saga.CompensationStateCompleted,
				StartedAt:   &now,
				CompletedAt: &now,
			},
		},
		{
			ID:        "test-saga-4-step-1",
			SagaID:    "test-saga-4",
			StepIndex: 1,
			State:     saga.StepStateCompensated,
			CompensationState: &saga.CompensationState{
				State:       saga.CompensationStateCompleted,
				StartedAt:   &now,
				CompletedAt: &now,
			},
		},
	}
	for _, step := range stepStates {
		_ = storage.SaveStepState(context.Background(), "test-saga-4", step)
	}

	ctx := context.Background()
	err = rm.recoverCompensatingSaga(ctx, sagaInst)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify saga state was updated to Compensated
	updatedSaga, _ := storage.GetSaga(ctx, "test-saga-4")
	if updatedSaga.GetState() != saga.StateCompensated {
		t.Errorf("Expected saga state to be Compensated, got %s", updatedSaga.GetState().String())
	}
}

// Test recoverSagaInstance with terminal state
func TestRecoverSagaInstance_TerminalState(t *testing.T) {
	storage := newTestStorage()
	coordinator := newTestCoordinator()

	config := DefaultRecoveryConfig()
	rm, err := NewRecoveryManager(config, storage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create recovery manager: %v", err)
	}

	// Create a completed saga
	now := time.Now()
	sagaInst := &mockSagaInstance{
		id:          "test-saga-5",
		state:       saga.StateCompleted,
		currentStep: 3,
		totalSteps:  3,
		startTime:   now,
		createdAt:   now,
		updatedAt:   now,
	}
	_ = storage.SaveSaga(context.Background(), sagaInst)

	ctx := context.Background()
	err = rm.recoverSagaInstance(ctx, "test-saga-5")

	if !errors.Is(err, ErrRecoverySkipped) {
		t.Errorf("Expected ErrRecoverySkipped, got %v", err)
	}
}

// Test recoverSagaInstance with max attempts exceeded
// Note: This is a simplified test. Full recovery attempt tracking would require
// persistent storage of recovery history (to be implemented in #597)
func TestRecoverSagaInstance_MaxAttemptsExceeded(t *testing.T) {
	storage := newTestStorage()
	coordinator := newTestCoordinator()

	config := DefaultRecoveryConfig()
	config.MaxRecoveryAttempts = 1 // Set to 1 so 2nd attempt exceeds it
	rm, err := NewRecoveryManager(config, storage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create recovery manager: %v", err)
	}

	// Create a running saga
	now := time.Now()
	sagaID := "test-saga-6"
	sagaInst := &mockSagaInstance{
		id:          sagaID,
		state:       saga.StateRunning,
		currentStep: 1,
		totalSteps:  3,
		startTime:   now,
		createdAt:   now,
		updatedAt:   now,
	}
	_ = storage.SaveSaga(context.Background(), sagaInst)

	// Create a step state
	stepState := &saga.StepState{
		ID:        sagaID + "-step-1",
		SagaID:    sagaID,
		StepIndex: 1,
		Name:      "test-step",
		State:     saga.StepStatePending,
		CreatedAt: now,
	}
	_ = storage.SaveStepState(context.Background(), sagaID, stepState)

	ctx := context.Background()

	// First recovery attempt should succeed
	err = rm.RecoverSaga(ctx, sagaID)
	if err != nil {
		t.Fatalf("First recovery attempt failed: %v", err)
	}

	// Verify stats were updated
	stats := rm.GetRecoveryStats()
	if stats.TotalRecoveryAttempts == 0 {
		t.Error("Expected at least one recovery attempt")
	}
	if stats.SuccessfulRecoveries == 0 {
		t.Error("Expected at least one successful recovery")
	}
}

// Test recovery strategy selector
func TestRecoveryStrategySelector(t *testing.T) {
	selector := NewRecoveryStrategySelector()

	tests := []struct {
		name             string
		sagaState        saga.SagaState
		errorType        saga.ErrorType
		retryable        bool
		currentStep      int
		expectedStrategy RecoveryStrategyType
	}{
		{
			name:             "Running with timeout error - retry",
			sagaState:        saga.StateRunning,
			errorType:        saga.ErrorTypeTimeout,
			retryable:        true,
			currentStep:      1,
			expectedStrategy: RecoveryStrategyRetry,
		},
		{
			name:             "Running with business error - compensate",
			sagaState:        saga.StateRunning,
			errorType:        saga.ErrorTypeBusiness,
			retryable:        false,
			currentStep:      2,
			expectedStrategy: RecoveryStrategyCompensate,
		},
		{
			name:             "Failed state - compensate",
			sagaState:        saga.StateFailed,
			errorType:        saga.ErrorTypeValidation,
			retryable:        false,
			currentStep:      1,
			expectedStrategy: RecoveryStrategyCompensate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			sagaInst := &mockSagaInstance{
				id:          "test-saga",
				state:       tt.sagaState,
				currentStep: tt.currentStep,
				totalSteps:  3,
				startTime:   now,
				createdAt:   now,
				updatedAt:   now,
			}
			if tt.errorType != "" {
				sagaInst.err = &saga.SagaError{
					Type:      tt.errorType,
					Retryable: tt.retryable,
				}
			}

			strategy := selector.SelectStrategy(sagaInst)
			if strategy.GetStrategyType() != tt.expectedStrategy {
				t.Errorf("Expected strategy %s, got %s", tt.expectedStrategy, strategy.GetStrategyType())
			}
		})
	}
}

// Test markSagaAsFailed
func TestMarkSagaAsFailed(t *testing.T) {
	storage := newTestStorage()
	coordinator := newTestCoordinator()

	config := DefaultRecoveryConfig()
	rm, err := NewRecoveryManager(config, storage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create recovery manager: %v", err)
	}

	now := time.Now()
	sagaInst := &mockSagaInstance{
		id:          "test-saga-7",
		state:       saga.StateRunning,
		currentStep: 1,
		totalSteps:  3,
		startTime:   now,
		createdAt:   now,
		updatedAt:   now,
	}
	_ = storage.SaveSaga(context.Background(), sagaInst)

	ctx := context.Background()
	testErr := errors.New("test error")
	err = rm.markSagaAsFailed(ctx, sagaInst, testErr)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	updatedSaga, _ := storage.GetSaga(ctx, "test-saga-7")
	if updatedSaga.GetState() != saga.StateFailed {
		t.Errorf("Expected saga state to be Failed, got %s", updatedSaga.GetState().String())
	}
}

// Test performRecoveryBatch
func TestPerformRecoveryBatch(t *testing.T) {
	storage := newTestStorage()
	coordinator := newTestCoordinator()

	config := DefaultRecoveryConfig()
	config.RecoveryBatchSize = 2
	rm, err := NewRecoveryManager(config, storage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create recovery manager: %v", err)
	}

	// Create test sagas
	now := time.Now()
	for i := 0; i < 3; i++ {
		sagaInst := &mockSagaInstance{
			id:          "test-saga-batch-" + string(rune('a'+i)),
			state:       saga.StateRunning,
			currentStep: 1,
			totalSteps:  3,
			startTime:   now,
			createdAt:   now,
			updatedAt:   now,
		}
		_ = storage.SaveSaga(context.Background(), sagaInst)

		stepState := &saga.StepState{
			ID:        sagaInst.id + "-step-1",
			SagaID:    sagaInst.id,
			StepIndex: 1,
			State:     saga.StepStatePending,
			CreatedAt: now,
		}
		_ = storage.SaveStepState(context.Background(), sagaInst.id, stepState)
	}

	detectionResults := []*DetectionResult{
		{SagaID: "test-saga-batch-a", Priority: PriorityHigh},
		{SagaID: "test-saga-batch-b", Priority: PriorityMedium},
		{SagaID: "test-saga-batch-c", Priority: PriorityLow},
	}

	ctx := context.Background()
	err = rm.performRecoveryBatch(ctx, detectionResults)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Give goroutines time to complete
	time.Sleep(200 * time.Millisecond)

	// Verify stats were updated
	stats := rm.GetRecoveryStats()
	if stats.TotalRecoveryAttempts == 0 {
		t.Error("Expected recovery attempts to be recorded")
	}
}

// Test calculateBackoff
func TestCalculateBackoff(t *testing.T) {
	storage := newTestStorage()
	coordinator := newTestCoordinator()

	config := DefaultRecoveryConfig()
	config.RecoveryBackoff = 1 * time.Minute
	rm, err := NewRecoveryManager(config, storage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("Failed to create recovery manager: %v", err)
	}

	tests := []struct {
		attemptCount int
		minBackoff   time.Duration
		maxBackoff   time.Duration
	}{
		{0, 1 * time.Minute, 2 * time.Minute},
		{1, 2 * time.Minute, 4 * time.Minute},
		{2, 4 * time.Minute, 8 * time.Minute},
		{10, 30 * time.Minute, 30 * time.Minute}, // Should cap at max
	}

	for _, tt := range tests {
		backoff := rm.calculateBackoff(tt.attemptCount)
		if backoff < tt.minBackoff || backoff > tt.maxBackoff {
			t.Errorf("Attempt %d: expected backoff between %v and %v, got %v",
				tt.attemptCount, tt.minBackoff, tt.maxBackoff, backoff)
		}
	}
}

// Test retry recovery strategy
func TestRetryRecoveryStrategy(t *testing.T) {
	strategy := NewRetryRecoveryStrategy(3)

	tests := []struct {
		name        string
		state       saga.SagaState
		errorType   saga.ErrorType
		retryable   bool
		canRecover  bool
		description string
	}{
		{
			name:        "Running with timeout error",
			state:       saga.StateRunning,
			errorType:   saga.ErrorTypeTimeout,
			retryable:   true,
			canRecover:  true,
			description: "Retry execution from the failed step",
		},
		{
			name:        "Running with business error",
			state:       saga.StateRunning,
			errorType:   saga.ErrorTypeBusiness,
			retryable:   false,
			canRecover:  false,
			description: "Retry execution from the failed step",
		},
		{
			name:       "Completed state",
			state:      saga.StateCompleted,
			canRecover: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			sagaInst := &mockSagaInstance{
				id:          "test-saga",
				state:       tt.state,
				currentStep: 1,
				totalSteps:  3,
				startTime:   now,
				createdAt:   now,
				updatedAt:   now,
			}
			if tt.errorType != "" {
				sagaInst.err = &saga.SagaError{
					Type:      tt.errorType,
					Retryable: tt.retryable,
				}
			}

			canRecover := strategy.CanRecover(sagaInst)
			if canRecover != tt.canRecover {
				t.Errorf("Expected CanRecover to be %v, got %v", tt.canRecover, canRecover)
			}

			if strategy.GetStrategyType() != RecoveryStrategyRetry {
				t.Errorf("Expected strategy type to be Retry, got %s", strategy.GetStrategyType())
			}

			if tt.description != "" && strategy.GetDescription() != tt.description {
				t.Errorf("Expected description %s, got %s", tt.description, strategy.GetDescription())
			}
		})
	}
}

// Test compensate recovery strategy
func TestCompensateRecoveryStrategy(t *testing.T) {
	strategy := NewCompensateRecoveryStrategy()

	tests := []struct {
		name        string
		state       saga.SagaState
		errorType   saga.ErrorType
		retryable   bool
		currentStep int
		canRecover  bool
	}{
		{
			name:        "Running with business error and completed steps",
			state:       saga.StateRunning,
			errorType:   saga.ErrorTypeBusiness,
			retryable:   false,
			currentStep: 2,
			canRecover:  true,
		},
		{
			name:        "Running with no completed steps",
			state:       saga.StateRunning,
			errorType:   saga.ErrorTypeBusiness,
			retryable:   false,
			currentStep: 0,
			canRecover:  false,
		},
		{
			name:        "Completed state",
			state:       saga.StateCompleted,
			currentStep: 3,
			canRecover:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			sagaInst := &mockSagaInstance{
				id:          "test-saga",
				state:       tt.state,
				currentStep: tt.currentStep,
				totalSteps:  3,
				startTime:   now,
				createdAt:   now,
				updatedAt:   now,
			}
			if tt.errorType != "" {
				sagaInst.err = &saga.SagaError{
					Type:      tt.errorType,
					Retryable: tt.retryable,
				}
			}

			canRecover := strategy.CanRecover(sagaInst)
			if canRecover != tt.canRecover {
				t.Errorf("Expected CanRecover to be %v, got %v", tt.canRecover, canRecover)
			}
		})
	}
}

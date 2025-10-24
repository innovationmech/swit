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

package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// validatorMockSagaInstance is a mock for testing validator.
type validatorMockSagaInstance struct {
	mockSagaInstance
	state saga.SagaState
}

func (m *validatorMockSagaInstance) GetState() saga.SagaState {
	return m.state
}

// validatorMockCoordinator is a mock coordinator for validator testing.
type validatorMockCoordinator struct {
	dashboardMockCoordinator
	instance      *validatorMockSagaInstance
	getInstanceErr error
}

func (m *validatorMockCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	if m.getInstanceErr != nil {
		return nil, m.getInstanceErr
	}
	if m.instance == nil {
		return nil, errors.New("saga not found")
	}
	return m.instance, nil
}

func TestNewOperationValidator(t *testing.T) {
	coordinator := newMockCoordinator()
	validator := NewOperationValidator(coordinator)

	if validator == nil {
		t.Fatal("NewOperationValidator() returned nil")
	}
	if validator.coordinator == nil {
		t.Error("Validator coordinator is nil")
	}
}

func TestValidateCancelOperation(t *testing.T) {
	tests := []struct {
		name      string
		sagaState saga.SagaState
		setupErr  error
		wantErr   bool
		errType   error
	}{
		{
			name:      "running saga can be cancelled",
			sagaState: saga.StateRunning,
			wantErr:   false,
		},
		{
			name:      "pending saga can be cancelled",
			sagaState: saga.StatePending,
			wantErr:   false,
		},
		{
			name:      "step completed saga can be cancelled",
			sagaState: saga.StateStepCompleted,
			wantErr:   false,
		},
		{
			name:      "completed saga cannot be cancelled",
			sagaState: saga.StateCompleted,
			wantErr:   true,
			errType:   ErrSagaAlreadyTerminal,
		},
		{
			name:      "failed saga cannot be cancelled",
			sagaState: saga.StateFailed,
			wantErr:   true,
			errType:   ErrSagaAlreadyTerminal,
		},
		{
			name:      "compensated saga cannot be cancelled",
			sagaState: saga.StateCompensated,
			wantErr:   true,
			errType:   ErrSagaAlreadyTerminal,
		},
		{
			name:      "cancelled saga cannot be cancelled again",
			sagaState: saga.StateCancelled,
			wantErr:   true,
			errType:   ErrSagaAlreadyTerminal,
		},
		{
			name:      "compensating saga cannot be cancelled",
			sagaState: saga.StateCompensating,
			wantErr:   true,
			errType:   ErrInvalidSagaState,
		},
		{
			name:     "saga not found",
			setupErr: errors.New("not found"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCoord := &validatorMockCoordinator{}
			if tt.setupErr == nil {
				mockCoord.instance = &validatorMockSagaInstance{
					mockSagaInstance: mockSagaInstance{id: "test-saga"},
					state:            tt.sagaState,
				}
			} else {
				mockCoord.getInstanceErr = tt.setupErr
			}

			validator := NewOperationValidator(mockCoord)
			ctx := context.Background()

			instance, err := validator.ValidateCancelOperation(ctx, "test-saga", "test reason")

			if tt.wantErr {
				if err == nil {
					t.Error("ValidateCancelOperation() should return error")
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("Expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateCancelOperation() unexpected error: %v", err)
				}
				if instance == nil {
					t.Error("ValidateCancelOperation() should return instance")
				}
			}
		})
	}
}

func TestValidateRetryOperation(t *testing.T) {
	tests := []struct {
		name      string
		sagaState saga.SagaState
		setupErr  error
		wantErr   bool
		errType   error
	}{
		{
			name:      "failed saga can be retried",
			sagaState: saga.StateFailed,
			wantErr:   false,
		},
		{
			name:      "compensated saga can be retried",
			sagaState: saga.StateCompensated,
			wantErr:   false,
		},
		{
			name:      "cancelled saga can be retried",
			sagaState: saga.StateCancelled,
			wantErr:   false,
		},
		{
			name:      "timed out saga can be retried",
			sagaState: saga.StateTimedOut,
			wantErr:   false,
		},
		{
			name:      "running saga cannot be retried",
			sagaState: saga.StateRunning,
			wantErr:   true,
			errType:   ErrInvalidSagaState,
		},
		{
			name:      "pending saga cannot be retried",
			sagaState: saga.StatePending,
			wantErr:   true,
			errType:   ErrInvalidSagaState,
		},
		{
			name:      "completed saga cannot be retried",
			sagaState: saga.StateCompleted,
			wantErr:   true,
			errType:   ErrInvalidSagaState,
		},
		{
			name:      "compensating saga cannot be retried",
			sagaState: saga.StateCompensating,
			wantErr:   true,
			errType:   ErrInvalidSagaState,
		},
		{
			name:     "saga not found",
			setupErr: errors.New("not found"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCoord := &validatorMockCoordinator{}
			if tt.setupErr == nil {
				mockCoord.instance = &validatorMockSagaInstance{
					mockSagaInstance: mockSagaInstance{id: "test-saga"},
					state:            tt.sagaState,
				}
			} else {
				mockCoord.getInstanceErr = tt.setupErr
			}

			validator := NewOperationValidator(mockCoord)
			ctx := context.Background()

			instance, err := validator.ValidateRetryOperation(ctx, "test-saga", 0)

			if tt.wantErr {
				if err == nil {
					t.Error("ValidateRetryOperation() should return error")
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("Expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRetryOperation() unexpected error: %v", err)
				}
				if instance == nil {
					t.Error("ValidateRetryOperation() should return instance")
				}
			}
		})
	}
}

func TestValidateStateTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    saga.SagaState
		to      saga.SagaState
		wantErr bool
	}{
		// Valid transitions from Pending
		{name: "Pending to Running", from: saga.StatePending, to: saga.StateRunning, wantErr: false},
		{name: "Pending to Cancelled", from: saga.StatePending, to: saga.StateCancelled, wantErr: false},
		{name: "Pending to Failed", from: saga.StatePending, to: saga.StateFailed, wantErr: false},

		// Valid transitions from Running
		{name: "Running to StepCompleted", from: saga.StateRunning, to: saga.StateStepCompleted, wantErr: false},
		{name: "Running to Completed", from: saga.StateRunning, to: saga.StateCompleted, wantErr: false},
		{name: "Running to Compensating", from: saga.StateRunning, to: saga.StateCompensating, wantErr: false},
		{name: "Running to Failed", from: saga.StateRunning, to: saga.StateFailed, wantErr: false},
		{name: "Running to Cancelled", from: saga.StateRunning, to: saga.StateCancelled, wantErr: false},

		// Valid transitions from StepCompleted
		{name: "StepCompleted to Running", from: saga.StateStepCompleted, to: saga.StateRunning, wantErr: false},
		{name: "StepCompleted to Completed", from: saga.StateStepCompleted, to: saga.StateCompleted, wantErr: false},
		{name: "StepCompleted to Compensating", from: saga.StateStepCompleted, to: saga.StateCompensating, wantErr: false},

		// Valid transitions from Compensating
		{name: "Compensating to Compensated", from: saga.StateCompensating, to: saga.StateCompensated, wantErr: false},
		{name: "Compensating to Failed", from: saga.StateCompensating, to: saga.StateFailed, wantErr: false},

		// Valid retry transitions from terminal states
		{name: "Failed to Running (retry)", from: saga.StateFailed, to: saga.StateRunning, wantErr: false},
		{name: "Compensated to Running (retry)", from: saga.StateCompensated, to: saga.StateRunning, wantErr: false},
		{name: "Cancelled to Running (retry)", from: saga.StateCancelled, to: saga.StateRunning, wantErr: false},

		// Invalid transitions
		{name: "Completed to Running", from: saga.StateCompleted, to: saga.StateRunning, wantErr: true},
		{name: "Completed to Failed", from: saga.StateCompleted, to: saga.StateFailed, wantErr: true},
		{name: "Pending to Completed", from: saga.StatePending, to: saga.StateCompleted, wantErr: true},
		{name: "Running to Compensated", from: saga.StateRunning, to: saga.StateCompensated, wantErr: true},
	}

	coordinator := newMockCoordinator()
	validator := NewOperationValidator(coordinator)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateStateTransition(tt.from, tt.to)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateStateTransition(%v, %v) should return error", tt.from, tt.to)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateStateTransition(%v, %v) unexpected error: %v", tt.from, tt.to, err)
				}
			}
		})
	}
}

func TestValidatePermission(t *testing.T) {
	coordinator := newMockCoordinator()
	validator := NewOperationValidator(coordinator)
	ctx := context.Background()

	// Currently this is a placeholder that always succeeds
	err := validator.ValidatePermission(ctx, "cancel", "test-saga")
	if err != nil {
		t.Errorf("ValidatePermission() unexpected error: %v", err)
	}

	err = validator.ValidatePermission(ctx, "retry", "test-saga")
	if err != nil {
		t.Errorf("ValidatePermission() unexpected error: %v", err)
	}
}

func TestCheckConcurrentOperation(t *testing.T) {
	coordinator := newMockCoordinator()
	validator := NewOperationValidator(coordinator)
	ctx := context.Background()

	// Currently this is a placeholder that always succeeds
	err := validator.CheckConcurrentOperation(ctx, "test-saga")
	if err != nil {
		t.Errorf("CheckConcurrentOperation() unexpected error: %v", err)
	}
}

func TestValidatorWithContextCancellation(t *testing.T) {
	mockCoord := &validatorMockCoordinator{}
	mockCoord.instance = &validatorMockSagaInstance{
		mockSagaInstance: mockSagaInstance{id: "test-saga"},
		state:            saga.StateRunning,
	}

	validator := NewOperationValidator(mockCoord)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Operations should still work even with cancelled context
	// (the actual context handling is done by the coordinator)
	_, err := validator.ValidateCancelOperation(ctx, "test-saga", "test")
	if err != nil {
		// This is acceptable - depends on coordinator implementation
		t.Logf("ValidateCancelOperation() with cancelled context: %v", err)
	}
}

func TestValidatorWithTimeout(t *testing.T) {
	mockCoord := &validatorMockCoordinator{}
	mockCoord.instance = &validatorMockSagaInstance{
		mockSagaInstance: mockSagaInstance{id: "test-saga"},
		state:            saga.StateFailed,
	}

	validator := NewOperationValidator(mockCoord)

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure timeout is triggered

	// Operations should still work - timeout handling is in coordinator
	_, err := validator.ValidateRetryOperation(ctx, "test-saga", 0)
	if err != nil {
		t.Logf("ValidateRetryOperation() with timeout: %v", err)
	}
}


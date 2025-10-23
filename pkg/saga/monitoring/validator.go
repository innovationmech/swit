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
	"fmt"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

var (
	// ErrSagaNotFound is returned when a Saga instance is not found.
	ErrSagaNotFound = errors.New("saga not found")

	// ErrInvalidSagaState is returned when a Saga is in an invalid state for the operation.
	ErrInvalidSagaState = errors.New("invalid saga state for operation")

	// ErrSagaAlreadyTerminal is returned when trying to operate on a terminal Saga.
	ErrSagaAlreadyTerminal = errors.New("saga is already in terminal state")

	// ErrUnauthorized is returned when the user is not authorized for the operation.
	ErrUnauthorized = errors.New("unauthorized operation")

	// ErrConcurrentOperation is returned when a concurrent operation is detected.
	ErrConcurrentOperation = errors.New("concurrent operation detected")
)

// OperationValidator validates Saga control operations before execution.
type OperationValidator struct {
	coordinator saga.SagaCoordinator
}

// NewOperationValidator creates a new operation validator.
func NewOperationValidator(coordinator saga.SagaCoordinator) *OperationValidator {
	return &OperationValidator{
		coordinator: coordinator,
	}
}

// ValidateCancelOperation validates if a Saga can be cancelled.
//
// Validation rules:
//  1. Saga must exist
//  2. Saga must not be in a terminal state (Completed, Compensated, Failed, Cancelled)
//  3. Saga must be in a cancellable state (Running, StepCompleted, Pending)
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaID: The ID of the Saga to validate.
//   - reason: The reason for cancellation (for logging).
//
// Returns:
//   - The Saga instance if validation passes.
//   - An error if validation fails.
func (v *OperationValidator) ValidateCancelOperation(ctx context.Context, sagaID string, reason string) (saga.SagaInstance, error) {
	if logger.Logger != nil {
		logger.Logger.Debug("Validating cancel operation",
			zap.String("saga_id", sagaID),
			zap.String("reason", reason))
	}

	// Get Saga instance
	instance, err := v.coordinator.GetSagaInstance(sagaID)
	if err != nil {
		if saga.IsSagaNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrSagaNotFound, sagaID)
		}
		return nil, fmt.Errorf("failed to get saga instance: %w", err)
	}

	// Check if Saga is already in terminal state
	if instance.IsTerminal() {
		return nil, fmt.Errorf("%w: current state is %s", ErrSagaAlreadyTerminal, instance.GetState().String())
	}

	// Check if Saga is in a cancellable state
	state := instance.GetState()
	switch state {
	case saga.StateRunning, saga.StateStepCompleted, saga.StatePending:
		// These states are cancellable
		if logger.Logger != nil {
			logger.Logger.Debug("Saga is in cancellable state",
				zap.String("saga_id", sagaID),
				zap.String("state", state.String()))
		}
		return instance, nil

	case saga.StateCompensating:
		// Already compensating, might want to allow this
		if logger.Logger != nil {
			logger.Logger.Warn("Saga is already compensating",
				zap.String("saga_id", sagaID))
		}
		return instance, nil

	default:
		return nil, fmt.Errorf("%w: cannot cancel saga in state %s", ErrInvalidSagaState, state.String())
	}
}

// ValidateRetryOperation validates if a Saga can be retried.
//
// Validation rules:
//  1. Saga must exist
//  2. Saga must be in a retryable state (Failed, Compensated, or Running with failed step)
//  3. If fromStep is specified, it must be within valid range
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaID: The ID of the Saga to validate.
//   - fromStep: The step index to retry from (optional, -1 means retry from current/last failed step).
//
// Returns:
//   - The Saga instance if validation passes.
//   - An error if validation fails.
func (v *OperationValidator) ValidateRetryOperation(ctx context.Context, sagaID string, fromStep int) (saga.SagaInstance, error) {
	if logger.Logger != nil {
		logger.Logger.Debug("Validating retry operation",
			zap.String("saga_id", sagaID),
			zap.Int("from_step", fromStep))
	}

	// Get Saga instance
	instance, err := v.coordinator.GetSagaInstance(sagaID)
	if err != nil {
		if saga.IsSagaNotFound(err) {
			return nil, fmt.Errorf("%w: %s", ErrSagaNotFound, sagaID)
		}
		return nil, fmt.Errorf("failed to get saga instance: %w", err)
	}

	// Validate step index if specified
	if fromStep >= 0 {
		if fromStep >= instance.GetTotalSteps() {
			return nil, fmt.Errorf("invalid from_step: %d is out of range (total steps: %d)", fromStep, instance.GetTotalSteps())
		}
	}

	// Check if Saga is in a retryable state
	state := instance.GetState()
	switch state {
	case saga.StateFailed:
		// Failed Sagas can be retried
		if logger.Logger != nil {
			logger.Logger.Debug("Saga is in failed state, can be retried",
				zap.String("saga_id", sagaID))
		}
		return instance, nil

	case saga.StateCompensated:
		// Compensated Sagas can potentially be retried
		if logger.Logger != nil {
			logger.Logger.Debug("Saga is in compensated state, can be retried",
				zap.String("saga_id", sagaID))
		}
		return instance, nil

	case saga.StateRunning, saga.StateStepCompleted:
		// Running Sagas can be retried from a specific step if that step failed
		if logger.Logger != nil {
			logger.Logger.Debug("Saga is running, can retry specific step",
				zap.String("saga_id", sagaID),
				zap.String("state", state.String()))
		}
		return instance, nil

	case saga.StateCompleted:
		// Completed Sagas cannot be retried
		return nil, fmt.Errorf("%w: cannot retry completed saga", ErrInvalidSagaState)

	case saga.StateCancelled:
		// Cancelled Sagas might be retried in some scenarios
		if logger.Logger != nil {
			logger.Logger.Warn("Attempting to retry cancelled saga",
				zap.String("saga_id", sagaID))
		}
		return instance, nil

	default:
		return nil, fmt.Errorf("%w: cannot retry saga in state %s", ErrInvalidSagaState, state.String())
	}
}

// ValidateStateTransition validates if a state transition is allowed.
// This is a helper method for more complex validation scenarios.
//
// Parameters:
//   - from: The current state.
//   - to: The target state.
//
// Returns:
//   - An error if the transition is not allowed.
func (v *OperationValidator) ValidateStateTransition(from, to saga.SagaState) error {
	// Terminal states cannot transition to other states (except for manual intervention)
	if from.IsTerminal() && to != saga.StateRunning {
		return fmt.Errorf("%w: cannot transition from terminal state %s to %s", ErrInvalidSagaState, from.String(), to.String())
	}

	// Define allowed transitions
	allowedTransitions := map[saga.SagaState][]saga.SagaState{
		saga.StatePending: {
			saga.StateRunning,
			saga.StateCancelled,
			saga.StateFailed,
		},
		saga.StateRunning: {
			saga.StateStepCompleted,
			saga.StateCompleted,
			saga.StateCompensating,
			saga.StateFailed,
			saga.StateCancelled,
			saga.StateTimedOut,
		},
		saga.StateStepCompleted: {
			saga.StateRunning,
			saga.StateCompleted,
			saga.StateCompensating,
			saga.StateFailed,
			saga.StateCancelled,
		},
		saga.StateCompensating: {
			saga.StateCompensated,
			saga.StateFailed,
		},
		// Terminal states
		saga.StateCompleted:   {},
		saga.StateCompensated: {saga.StateRunning}, // Allow retry
		saga.StateFailed:      {saga.StateRunning}, // Allow retry
		saga.StateCancelled:   {saga.StateRunning}, // Allow retry
		saga.StateTimedOut:    {saga.StateRunning}, // Allow retry
	}

	allowed, exists := allowedTransitions[from]
	if !exists {
		return fmt.Errorf("%w: unknown state %s", ErrInvalidSagaState, from.String())
	}

	for _, allowedState := range allowed {
		if to == allowedState {
			return nil
		}
	}

	return fmt.Errorf("%w: transition from %s to %s is not allowed", ErrInvalidSagaState, from.String(), to.String())
}

// ValidatePermission validates if the user has permission to perform the operation.
// This is a placeholder for future authentication/authorization integration.
//
// Parameters:
//   - ctx: Context containing user information.
//   - operation: The operation being performed (e.g., "cancel", "retry").
//   - sagaID: The ID of the Saga being operated on.
//
// Returns:
//   - An error if the user is not authorized.
func (v *OperationValidator) ValidatePermission(ctx context.Context, operation string, sagaID string) error {
	// TODO: Implement actual permission checking when authentication/authorization is added
	// For now, we'll just log the operation for audit purposes

	if logger.Logger != nil {
		logger.Logger.Info("Permission check requested",
			zap.String("operation", operation),
			zap.String("saga_id", sagaID))
	}

	// In a real implementation, this would:
	// 1. Extract user identity from context
	// 2. Check user permissions against operation and resource
	// 3. Return ErrUnauthorized if permission is denied

	return nil
}

// CheckConcurrentOperation checks if there's a concurrent operation on the same Saga.
// This helps prevent race conditions when multiple control operations are attempted simultaneously.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaID: The ID of the Saga to check.
//
// Returns:
//   - An error if a concurrent operation is detected.
func (v *OperationValidator) CheckConcurrentOperation(ctx context.Context, sagaID string) error {
	// TODO: Implement actual concurrent operation detection
	// This would typically use:
	// 1. Distributed locks (Redis, etcd)
	// 2. Database-level locks
	// 3. In-memory lock tracking

	if logger.Logger != nil {
		logger.Logger.Debug("Checking for concurrent operations",
			zap.String("saga_id", sagaID))
	}

	// For now, we'll just log and return success
	// In a production system, this would implement proper locking
	return nil
}

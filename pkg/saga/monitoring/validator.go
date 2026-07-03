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
	"sync"

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

// PermissionChecker decides whether the caller identified by the context is
// allowed to perform a control operation on a Saga. Implementations typically
// extract the caller identity from the context (e.g. from authentication
// middleware) and evaluate it against the operation and resource.
type PermissionChecker interface {
	// CheckPermission returns nil if the operation is allowed, or an error
	// describing why it is denied.
	CheckPermission(ctx context.Context, operation string, sagaID string) error
}

// PermissionCheckerFunc adapts a function to the PermissionChecker interface.
type PermissionCheckerFunc func(ctx context.Context, operation string, sagaID string) error

// CheckPermission implements PermissionChecker.
func (f PermissionCheckerFunc) CheckPermission(ctx context.Context, operation string, sagaID string) error {
	return f(ctx, operation, sagaID)
}

// OperationValidator validates Saga control operations before execution.
type OperationValidator struct {
	coordinator saga.SagaCoordinator

	// permissionChecker is the pluggable authorization hook. When nil, all
	// operations are allowed (see ValidatePermission for the design boundary).
	permissionChecker PermissionChecker

	// inFlightOps tracks Saga IDs with an operation currently in progress,
	// used by BeginOperation/CheckConcurrentOperation to detect concurrent
	// control operations within this process.
	inFlightOps sync.Map
}

// NewOperationValidator creates a new operation validator.
func NewOperationValidator(coordinator saga.SagaCoordinator) *OperationValidator {
	return &OperationValidator{
		coordinator: coordinator,
	}
}

// SetPermissionChecker configures the authorization hook used by
// ValidatePermission. Passing nil restores the default allow-all behavior.
func (v *OperationValidator) SetPermissionChecker(checker PermissionChecker) {
	v.permissionChecker = checker
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

// ValidatePermission validates if the caller has permission to perform the operation.
//
// Design boundary: the monitoring package does not ship an authentication or
// authorization system. Authorization is delegated to the PermissionChecker
// configured via SetPermissionChecker; when no checker is configured, all
// operations are allowed and the request is logged for audit purposes.
//
// Parameters:
//   - ctx: Context containing caller information (as understood by the configured checker).
//   - operation: The operation being performed (e.g., "cancel", "retry").
//   - sagaID: The ID of the Saga being operated on.
//
// Returns:
//   - An error wrapping ErrUnauthorized if the caller is not authorized.
func (v *OperationValidator) ValidatePermission(ctx context.Context, operation string, sagaID string) error {
	if logger.Logger != nil {
		logger.Logger.Info("Permission check requested",
			zap.String("operation", operation),
			zap.String("saga_id", sagaID))
	}

	if v.permissionChecker == nil {
		// No authorization configured: allow by design (see doc comment).
		return nil
	}

	if err := v.permissionChecker.CheckPermission(ctx, operation, sagaID); err != nil {
		return fmt.Errorf("%w: %s on saga %s: %v", ErrUnauthorized, operation, sagaID, err)
	}

	return nil
}

// BeginOperation marks the start of a control operation on the given Saga and
// returns a release function that must be called when the operation finishes
// (typically via defer). If another operation started through BeginOperation
// is still in progress for the same Saga, ErrConcurrentOperation is returned.
//
// Design boundary: detection is process-local (in-memory). Deployments running
// multiple control-plane replicas need an external coordination mechanism
// (e.g. distributed locks) layered on top.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaID: The ID of the Saga being operated on.
//
// Returns:
//   - A release function and nil on success, or nil and an error if a
//     concurrent operation is detected.
func (v *OperationValidator) BeginOperation(ctx context.Context, sagaID string) (func(), error) {
	if _, loaded := v.inFlightOps.LoadOrStore(sagaID, struct{}{}); loaded {
		return nil, fmt.Errorf("%w: saga %s", ErrConcurrentOperation, sagaID)
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			v.inFlightOps.Delete(sagaID)
		})
	}
	return release, nil
}

// CheckConcurrentOperation checks if there's a concurrent operation on the same Saga.
// It reports operations tracked via BeginOperation within this process.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - sagaID: The ID of the Saga to check.
//
// Returns:
//   - An error wrapping ErrConcurrentOperation if a concurrent operation is detected.
func (v *OperationValidator) CheckConcurrentOperation(ctx context.Context, sagaID string) error {
	if logger.Logger != nil {
		logger.Logger.Debug("Checking for concurrent operations",
			zap.String("saga_id", sagaID))
	}

	if _, inFlight := v.inFlightOps.Load(sagaID); inFlight {
		return fmt.Errorf("%w: saga %s", ErrConcurrentOperation, sagaID)
	}

	return nil
}

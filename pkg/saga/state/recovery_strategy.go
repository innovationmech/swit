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

	"github.com/innovationmech/swit/pkg/saga"
)

// RecoveryStrategyType represents the type of recovery strategy.
type RecoveryStrategyType string

const (
	// RecoveryStrategyRetry indicates the Saga should be retried from the failed step.
	RecoveryStrategyRetry RecoveryStrategyType = "retry"

	// RecoveryStrategyCompensate indicates the Saga should trigger compensation.
	RecoveryStrategyCompensate RecoveryStrategyType = "compensate"

	// RecoveryStrategyMarkFailed indicates the Saga should be marked as failed without further action.
	RecoveryStrategyMarkFailed RecoveryStrategyType = "mark_failed"

	// RecoveryStrategyContinue indicates the Saga should continue from the next step.
	RecoveryStrategyContinue RecoveryStrategyType = "continue"
)

// RecoveryStrategy defines how to recover a failed Saga.
// Different strategies can be applied based on the Saga's state, error type,
// and business requirements.
type RecoveryStrategy interface {
	// GetStrategyType returns the type of this recovery strategy.
	GetStrategyType() RecoveryStrategyType

	// CanRecover determines if this strategy can recover the given Saga.
	// Returns true if the strategy is applicable to the Saga's current state.
	CanRecover(sagaInst saga.SagaInstance) bool

	// Recover attempts to recover the Saga using this strategy.
	// Returns an error if recovery fails.
	Recover(ctx context.Context, sagaInst saga.SagaInstance, coordinator saga.SagaCoordinator) error

	// GetDescription returns a human-readable description of what this strategy does.
	GetDescription() string
}

// RetryRecoveryStrategy attempts to retry the failed step execution.
// This is suitable for transient failures like network timeouts or temporary service unavailability.
type RetryRecoveryStrategy struct {
	// maxAttempts is the maximum number of retry attempts.
	maxAttempts int

	// retryableStates are the Saga states for which retry is applicable.
	retryableStates []saga.SagaState

	// retryableErrorTypes are the error types that are retryable.
	retryableErrorTypes []saga.ErrorType
}

// NewRetryRecoveryStrategy creates a new retry recovery strategy.
func NewRetryRecoveryStrategy(maxAttempts int) *RetryRecoveryStrategy {
	return &RetryRecoveryStrategy{
		maxAttempts: maxAttempts,
		retryableStates: []saga.SagaState{
			saga.StateRunning,
			saga.StateStepCompleted,
		},
		retryableErrorTypes: []saga.ErrorType{
			saga.ErrorTypeTimeout,
			saga.ErrorTypeNetwork,
			saga.ErrorTypeService,
		},
	}
}

// GetStrategyType returns the type of this recovery strategy.
func (s *RetryRecoveryStrategy) GetStrategyType() RecoveryStrategyType {
	return RecoveryStrategyRetry
}

// CanRecover determines if this strategy can recover the given Saga.
func (s *RetryRecoveryStrategy) CanRecover(sagaInst saga.SagaInstance) bool {
	// Check if Saga state is retryable
	state := sagaInst.GetState()
	stateRetryable := false
	for _, retryableState := range s.retryableStates {
		if state == retryableState {
			stateRetryable = true
			break
		}
	}

	if !stateRetryable {
		return false
	}

	// Check if error type is retryable
	sagaError := sagaInst.GetError()
	if sagaError != nil {
		for _, retryableType := range s.retryableErrorTypes {
			if sagaError.Type == retryableType && sagaError.Retryable {
				return true
			}
		}
		return false
	}

	// No error means Saga might be stuck, retry is applicable
	return true
}

// Recover attempts to recover the Saga by retrying the failed step.
func (s *RetryRecoveryStrategy) Recover(ctx context.Context, sagaInst saga.SagaInstance, coordinator saga.SagaCoordinator) error {
	// The actual retry logic will be implemented in recoverRunningSaga
	// This is a validation method to check if retry is allowed
	return nil
}

// GetDescription returns a human-readable description of this strategy.
func (s *RetryRecoveryStrategy) GetDescription() string {
	return "Retry execution from the failed step"
}

// CompensateRecoveryStrategy triggers compensation for the Saga.
// This is suitable for non-transient failures where the business transaction cannot continue.
type CompensateRecoveryStrategy struct {
	// compensatableStates are the Saga states for which compensation is applicable.
	compensatableStates []saga.SagaState

	// compensatableErrorTypes are the error types that require compensation.
	compensatableErrorTypes []saga.ErrorType
}

// NewCompensateRecoveryStrategy creates a new compensate recovery strategy.
func NewCompensateRecoveryStrategy() *CompensateRecoveryStrategy {
	return &CompensateRecoveryStrategy{
		compensatableStates: []saga.SagaState{
			saga.StateRunning,
			saga.StateStepCompleted,
			saga.StateFailed,
		},
		compensatableErrorTypes: []saga.ErrorType{
			saga.ErrorTypeBusiness,
			saga.ErrorTypeValidation,
			saga.ErrorTypeData,
		},
	}
}

// GetStrategyType returns the type of this recovery strategy.
func (s *CompensateRecoveryStrategy) GetStrategyType() RecoveryStrategyType {
	return RecoveryStrategyCompensate
}

// CanRecover determines if this strategy can recover the given Saga.
func (s *CompensateRecoveryStrategy) CanRecover(sagaInst saga.SagaInstance) bool {
	// Check if Saga state allows compensation
	state := sagaInst.GetState()
	stateCompensatable := false
	for _, compensatableState := range s.compensatableStates {
		if state == compensatableState {
			stateCompensatable = true
			break
		}
	}

	if !stateCompensatable {
		return false
	}

	// Check if there are completed steps to compensate
	if sagaInst.GetCompletedSteps() == 0 {
		return false
	}

	// Check error type if available
	sagaError := sagaInst.GetError()
	if sagaError != nil {
		for _, compensatableType := range s.compensatableErrorTypes {
			if sagaError.Type == compensatableType {
				return true
			}
		}
		// If error type is not in compensatable list, check if error is non-retryable
		return !sagaError.Retryable
	}

	return true
}

// Recover attempts to recover the Saga by triggering compensation.
func (s *CompensateRecoveryStrategy) Recover(ctx context.Context, sagaInst saga.SagaInstance, coordinator saga.SagaCoordinator) error {
	// Trigger compensation via coordinator
	return coordinator.CancelSaga(ctx, sagaInst.GetID(), "Recovery triggered compensation")
}

// GetDescription returns a human-readable description of this strategy.
func (s *CompensateRecoveryStrategy) GetDescription() string {
	return "Trigger compensation to rollback completed steps"
}

// MarkFailedStrategy marks the Saga as failed without further action.
// This is used when recovery is not possible or when the Saga has exceeded retry limits.
type MarkFailedStrategy struct {
	// failableStates are the Saga states that can be marked as failed.
	failableStates []saga.SagaState
}

// NewMarkFailedStrategy creates a new mark failed recovery strategy.
func NewMarkFailedStrategy() *MarkFailedStrategy {
	return &MarkFailedStrategy{
		failableStates: []saga.SagaState{
			saga.StateRunning,
			saga.StateStepCompleted,
			saga.StateCompensating,
			saga.StateTimedOut,
		},
	}
}

// GetStrategyType returns the type of this recovery strategy.
func (s *MarkFailedStrategy) GetStrategyType() RecoveryStrategyType {
	return RecoveryStrategyMarkFailed
}

// CanRecover determines if this strategy can recover the given Saga.
func (s *MarkFailedStrategy) CanRecover(sagaInst saga.SagaInstance) bool {
	// Check if Saga state can be marked as failed
	state := sagaInst.GetState()
	for _, failableState := range s.failableStates {
		if state == failableState {
			return true
		}
	}
	return false
}

// Recover attempts to recover the Saga by marking it as failed.
func (s *MarkFailedStrategy) Recover(ctx context.Context, sagaInst saga.SagaInstance, coordinator saga.SagaCoordinator) error {
	// Mark as failed - this will be handled by the recovery execution logic
	return nil
}

// GetDescription returns a human-readable description of this strategy.
func (s *MarkFailedStrategy) GetDescription() string {
	return "Mark Saga as failed without further recovery attempts"
}

// RecoveryStrategySelector selects the appropriate recovery strategy based on Saga state and error.
type RecoveryStrategySelector struct {
	strategies []RecoveryStrategy
}

// NewRecoveryStrategySelector creates a new recovery strategy selector with default strategies.
func NewRecoveryStrategySelector() *RecoveryStrategySelector {
	return &RecoveryStrategySelector{
		strategies: []RecoveryStrategy{
			NewRetryRecoveryStrategy(3),
			NewCompensateRecoveryStrategy(),
			NewMarkFailedStrategy(),
		},
	}
}

// SelectStrategy selects the most appropriate recovery strategy for the given Saga.
// It evaluates strategies in order and returns the first one that can recover the Saga.
func (s *RecoveryStrategySelector) SelectStrategy(sagaInst saga.SagaInstance) RecoveryStrategy {
	for _, strategy := range s.strategies {
		if strategy.CanRecover(sagaInst) {
			return strategy
		}
	}

	// Default to mark failed if no strategy matches
	return NewMarkFailedStrategy()
}

// AddStrategy adds a custom recovery strategy to the selector.
// Strategies are evaluated in the order they are added.
func (s *RecoveryStrategySelector) AddStrategy(strategy RecoveryStrategy) {
	s.strategies = append([]RecoveryStrategy{strategy}, s.strategies...)
}

// GetAvailableStrategies returns all available recovery strategies.
func (s *RecoveryStrategySelector) GetAvailableStrategies() []RecoveryStrategy {
	return s.strategies
}

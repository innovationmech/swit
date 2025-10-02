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

package saga

import (
	"fmt"
	"runtime"
	"time"
)

// predefined error codes
const (
	ErrCodeSagaNotFound        = "SAGA_NOT_FOUND"
	ErrCodeSagaAlreadyExists   = "SAGA_ALREADY_EXISTS"
	ErrCodeInvalidSagaState    = "INVALID_SAGA_STATE"
	ErrCodeSagaTimeout         = "SAGA_TIMEOUT"
	ErrCodeStepExecutionFailed = "STEP_EXECUTION_FAILED"
	ErrCodeCompensationFailed  = "COMPENSATION_FAILED"
	ErrCodeStorageError        = "STORAGE_ERROR"
	ErrCodeValidationError     = "VALIDATION_ERROR"
	ErrCodeConfigurationError  = "CONFIGURATION_ERROR"
	ErrCodeCoordinatorStopped  = "COORDINATOR_STOPPED"
	ErrCodeRetryExhausted      = "RETRY_EXHAUSTED"
	ErrCodeEventPublishFailed  = "EVENT_PUBLISH_FAILED"
)

// NewSagaError creates a new SagaError with the specified parameters.
func NewSagaError(code, message string, errorType ErrorType, retryable bool) *SagaError {
	return &SagaError{
		Code:      code,
		Message:   message,
		Type:      errorType,
		Retryable: retryable,
		Timestamp: time.Now(),
	}
}

// WrapError wraps an existing error into a SagaError.
func WrapError(err error, code, message string, errorType ErrorType, retryable bool) *SagaError {
	if err == nil {
		return nil
	}

	sagaErr := NewSagaError(code, message, errorType, retryable)

	// If the original error is already a SagaError, preserve it as the cause
	if originalSagaErr, ok := err.(*SagaError); ok {
		sagaErr.Cause = originalSagaErr
	} else {
		// For non-SagaError errors, create a simple SagaError as the cause
		sagaErr.Cause = NewSagaError("WRAPPED_ERROR", err.Error(), ErrorTypeSystem, false)
	}

	return sagaErr
}

// Error implements the error interface for SagaError.
func (e *SagaError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %s)", e.Code, e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// WithStackTrace adds a stack trace to the SagaError.
// Returns a new SagaError with stack trace added.
func (e *SagaError) WithStackTrace() *SagaError {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)

	// Create a copy of the error
	errCopy := &SagaError{
		Code:       e.Code,
		Message:    e.Message,
		Type:       e.Type,
		Retryable:  e.Retryable,
		Timestamp:  e.Timestamp,
		StackTrace: string(buf[:n]),
		Details:    make(map[string]interface{}),
		Cause:      e.Cause,
	}

	// Copy details
	for k, v := range e.Details {
		errCopy.Details[k] = v
	}

	return errCopy
}

// WithDetail adds a detail to the SagaError.
func (e *SagaError) WithDetail(key string, value interface{}) *SagaError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithDetails adds multiple details to the SagaError.
func (e *SagaError) WithDetails(details map[string]interface{}) *SagaError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// GetChain returns the error chain as a slice of SagaError.
func (e *SagaError) GetChain() []*SagaError {
	chain := []*SagaError{e}
	current := e
	for current.Cause != nil {
		cause := current.Cause
		chain = append(chain, cause)
		current = cause
	}
	return chain
}

// IsRetryable checks if the error or any of its causes is retryable.
func (e *SagaError) IsRetryable() bool {
	if e.Retryable {
		return true
	}
	if e.Cause != nil {
		return e.Cause.IsRetryable()
	}
	return false
}

// GetRootCause returns the root cause of the error chain.
func (e *SagaError) GetRootCause() *SagaError {
	if e.Cause == nil {
		return e
	}
	return e.Cause.GetRootCause()
}

// Common error constructors

// NewSagaNotFoundError creates an error for when a Saga is not found.
func NewSagaNotFoundError(sagaID string) *SagaError {
	return NewSagaError(ErrCodeSagaNotFound, fmt.Sprintf("Saga with ID '%s' not found", sagaID), ErrorTypeData, false).
		WithDetail("saga_id", sagaID)
}

// NewSagaAlreadyExistsError creates an error for when a Saga already exists.
func NewSagaAlreadyExistsError(sagaID string) *SagaError {
	return NewSagaError(ErrCodeSagaAlreadyExists, fmt.Sprintf("Saga with ID '%s' already exists", sagaID), ErrorTypeData, false).
		WithDetail("saga_id", sagaID)
}

// NewInvalidSagaStateError creates an error for invalid Saga state transitions.
func NewInvalidSagaStateError(currentState, targetState SagaState) *SagaError {
	return NewSagaError(ErrCodeInvalidSagaState,
		fmt.Sprintf("Invalid state transition from %s to %s", currentState.String(), targetState.String()),
		ErrorTypeValidation, false).
		WithDetail("current_state", currentState.String()).
		WithDetail("target_state", targetState.String())
}

// NewSagaTimeoutError creates an error for Saga timeout.
func NewSagaTimeoutError(sagaID string, timeout time.Duration) *SagaError {
	return NewSagaError(ErrCodeSagaTimeout, fmt.Sprintf("Saga '%s' timed out after %v", sagaID, timeout), ErrorTypeTimeout, false).
		WithDetail("saga_id", sagaID).
		WithDetail("timeout", timeout.String())
}

// NewStepExecutionError creates an error for step execution failure.
func NewStepExecutionError(stepID, stepName string, err error) *SagaError {
	return WrapError(err, ErrCodeStepExecutionFailed,
		fmt.Sprintf("Step '%s' (%s) execution failed", stepName, stepID),
		ErrorTypeService, true).
		WithDetail("step_id", stepID).
		WithDetail("step_name", stepName)
}

// NewCompensationFailedError creates an error for compensation failure.
func NewCompensationFailedError(stepID, stepName string, err error) *SagaError {
	return WrapError(err, ErrCodeCompensationFailed,
		fmt.Sprintf("Compensation for step '%s' (%s) failed", stepName, stepID),
		ErrorTypeCompensation, true).
		WithDetail("step_id", stepID).
		WithDetail("step_name", stepName)
}

// NewStorageError creates an error for storage operation failures.
func NewStorageError(operation string, err error) *SagaError {
	return WrapError(err, ErrCodeStorageError,
		fmt.Sprintf("Storage operation '%s' failed", operation),
		ErrorTypeSystem, true).
		WithDetail("operation", operation)
}

// NewValidationError creates an error for validation failures.
func NewValidationError(message string) *SagaError {
	return NewSagaError(ErrCodeValidationError, message, ErrorTypeValidation, false)
}

// NewConfigurationError creates an error for configuration issues.
func NewConfigurationError(message string) *SagaError {
	return NewSagaError(ErrCodeConfigurationError, message, ErrorTypeSystem, false)
}

// NewCoordinatorStoppedError creates an error when the coordinator is stopped.
func NewCoordinatorStoppedError() *SagaError {
	return NewSagaError(ErrCodeCoordinatorStopped, "Saga coordinator is stopped", ErrorTypeSystem, false)
}

// NewRetryExhaustedError creates an error when retry attempts are exhausted.
func NewRetryExhaustedError(operation string, attempts int) *SagaError {
	return NewSagaError(ErrCodeRetryExhausted,
		fmt.Sprintf("Retry attempts exhausted for operation '%s' after %d attempts", operation, attempts),
		ErrorTypeSystem, false).
		WithDetail("operation", operation).
		WithDetail("attempts", attempts)
}

// NewEventPublishError creates an error for event publishing failures.
func NewEventPublishError(eventType SagaEventType, err error) *SagaError {
	return WrapError(err, ErrCodeEventPublishFailed,
		fmt.Sprintf("Failed to publish event of type '%s'", string(eventType)),
		ErrorTypeSystem, true).
		WithDetail("event_type", string(eventType))
}

// IsSagaNotFound checks if an error is a SagaNotFoundError.
func IsSagaNotFound(err error) bool {
	if sagaErr, ok := err.(*SagaError); ok {
		return sagaErr.Code == ErrCodeSagaNotFound
	}
	return false
}

// IsSagaTimeout checks if an error is a SagaTimeoutError.
func IsSagaTimeout(err error) bool {
	if sagaErr, ok := err.(*SagaError); ok {
		return sagaErr.Code == ErrCodeSagaTimeout
	}
	return false
}

// IsStepExecutionFailed checks if an error is a StepExecutionError.
func IsStepExecutionFailed(err error) bool {
	if sagaErr, ok := err.(*SagaError); ok {
		return sagaErr.Code == ErrCodeStepExecutionFailed
	}
	return false
}

// IsCompensationFailed checks if an error is a CompensationFailedError.
func IsCompensationFailed(err error) bool {
	if sagaErr, ok := err.(*SagaError); ok {
		return sagaErr.Code == ErrCodeCompensationFailed
	}
	return false
}

// IsRetryExhausted checks if an error is a RetryExhaustedError.
func IsRetryExhausted(err error) bool {
	if sagaErr, ok := err.(*SagaError); ok {
		return sagaErr.Code == ErrCodeRetryExhausted
	}
	return false
}

// IsRetryableError checks if an error should be retried.
func IsRetryableError(err error) bool {
	if sagaErr, ok := err.(*SagaError); ok {
		return sagaErr.IsRetryable()
	}
	return false
}

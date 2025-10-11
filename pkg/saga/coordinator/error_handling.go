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

// RetryAttempt represents a single retry attempt with its outcome.
type RetryAttempt struct {
	// Attempt is the attempt number (1-indexed)
	Attempt int

	// StartTime is when this attempt started
	StartTime time.Time

	// EndTime is when this attempt ended
	EndTime time.Time

	// Duration is how long this attempt took
	Duration time.Duration

	// Error is the error that occurred during this attempt (nil if successful)
	Error *saga.SagaError

	// Success indicates if this attempt was successful
	Success bool

	// WillRetry indicates if another retry will be attempted after this one
	WillRetry bool
}

// RetryHistory tracks the history of retry attempts for a step.
type RetryHistory struct {
	// StepID is the ID of the step being retried
	StepID string

	// StepName is the name of the step being retried
	StepName string

	// Attempts is the list of all retry attempts
	Attempts []*RetryAttempt

	// TotalAttempts is the total number of attempts made
	TotalAttempts int

	// SuccessfulAttempt is the attempt number that succeeded (0 if none)
	SuccessfulAttempt int

	// FinalError is the final error if all retries failed
	FinalError *saga.SagaError

	// CreatedAt is when retry history was created
	CreatedAt time.Time

	// UpdatedAt is when retry history was last updated
	UpdatedAt time.Time
}

// AddAttempt adds a retry attempt to the history.
func (rh *RetryHistory) AddAttempt(attempt *RetryAttempt) {
	rh.Attempts = append(rh.Attempts, attempt)
	rh.TotalAttempts++
	rh.UpdatedAt = time.Now()

	if attempt.Success {
		rh.SuccessfulAttempt = attempt.Attempt
	} else {
		rh.FinalError = attempt.Error
	}
}

// GetLastAttempt returns the most recent retry attempt.
func (rh *RetryHistory) GetLastAttempt() *RetryAttempt {
	if len(rh.Attempts) == 0 {
		return nil
	}
	return rh.Attempts[len(rh.Attempts)-1]
}

// ErrorHandler handles error classification, retry logic, and error history tracking.
type ErrorHandler struct {
	// coordinator is a reference to the orchestrator coordinator
	coordinator *OrchestratorCoordinator

	// instance is the Saga instance being handled
	instance *OrchestratorSagaInstance

	// retryHistories tracks retry history for each step
	retryHistories map[string]*RetryHistory
}

// newErrorHandler creates a new error handler for the given Saga instance.
func newErrorHandler(coordinator *OrchestratorCoordinator, instance *OrchestratorSagaInstance) *ErrorHandler {
	return &ErrorHandler{
		coordinator:    coordinator,
		instance:       instance,
		retryHistories: make(map[string]*RetryHistory),
	}
}

// ClassifyError classifies an error into a SagaError with appropriate type and retryability.
func (eh *ErrorHandler) ClassifyError(err error, step saga.SagaStep) *saga.SagaError {
	if err == nil {
		return nil
	}

	// If it's already a SagaError, return it
	if sagaErr, ok := err.(*saga.SagaError); ok {
		return sagaErr
	}

	// Determine error type based on error characteristics
	errorType := eh.determineErrorType(err)
	retryable := eh.isRetryable(err, step)

	// Create SagaError with details
	sagaErr := &saga.SagaError{
		Code:      eh.determineErrorCode(err, errorType),
		Message:   err.Error(),
		Type:      errorType,
		Retryable: retryable,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"step_id":   step.GetID(),
			"step_name": step.GetName(),
			"saga_id":   eh.instance.id,
		},
	}

	return sagaErr
}

// determineErrorType determines the ErrorType based on error characteristics.
func (eh *ErrorHandler) determineErrorType(err error) saga.ErrorType {
	if err == nil {
		return saga.ErrorTypeSystem
	}

	// Check for specific error types
	errStr := err.Error()

	// Context errors
	if err == context.DeadlineExceeded {
		return saga.ErrorTypeTimeout
	}
	if err == context.Canceled {
		return saga.ErrorTypeSystem
	}

	// Network errors (simple heuristic - can be enhanced)
	if containsAny(errStr, []string{"connection", "network", "timeout", "unreachable", "refused"}) {
		return saga.ErrorTypeNetwork
	}

	// Validation errors
	if containsAny(errStr, []string{"invalid", "validation", "missing required", "malformed"}) {
		return saga.ErrorTypeValidation
	}

	// Data errors
	if containsAny(errStr, []string{"not found", "already exists", "duplicate", "constraint"}) {
		return saga.ErrorTypeData
	}

	// Default to service error
	return saga.ErrorTypeService
}

// determineErrorCode determines the error code based on error type and content.
func (eh *ErrorHandler) determineErrorCode(err error, errorType saga.ErrorType) string {
	if err == nil {
		return "UNKNOWN_ERROR"
	}

	switch errorType {
	case saga.ErrorTypeTimeout:
		return saga.ErrCodeSagaTimeout
	case saga.ErrorTypeValidation:
		return saga.ErrCodeValidationError
	case saga.ErrorTypeData:
		return "DATA_ERROR"
	case saga.ErrorTypeNetwork:
		return "NETWORK_ERROR"
	case saga.ErrorTypeService:
		return saga.ErrCodeStepExecutionFailed
	case saga.ErrorTypeSystem:
		return "SYSTEM_ERROR"
	case saga.ErrorTypeBusiness:
		return "BUSINESS_ERROR"
	case saga.ErrorTypeCompensation:
		return saga.ErrCodeCompensationFailed
	default:
		return "UNKNOWN_ERROR"
	}
}

// isRetryable determines if an error should be retried.
func (eh *ErrorHandler) isRetryable(err error, step saga.SagaStep) bool {
	if err == nil {
		return false
	}

	// Check if step defines custom retryability
	if step.IsRetryable(err) {
		return true
	}

	// Non-retryable error types
	if err == context.Canceled {
		return false
	}

	// Check error type for retryability
	errorType := eh.determineErrorType(err)
	switch errorType {
	case saga.ErrorTypeValidation, saga.ErrorTypeBusiness, saga.ErrorTypeData:
		// These typically shouldn't be retried as they require different input
		return false
	case saga.ErrorTypeTimeout, saga.ErrorTypeNetwork, saga.ErrorTypeService, saga.ErrorTypeSystem:
		// These can potentially be retried
		return true
	default:
		return false
	}
}

// ShouldRetry determines if a step should be retried based on the retry policy.
func (eh *ErrorHandler) ShouldRetry(err error, step saga.SagaStep, attemptCount int) bool {
	if err == nil {
		return false
	}

	// Get retry policy
	retryPolicy := step.GetRetryPolicy()
	if retryPolicy == nil {
		retryPolicy = eh.coordinator.retryPolicy
	}

	// Check max attempts
	if attemptCount >= retryPolicy.GetMaxAttempts() {
		return false
	}

	// Classify error
	sagaErr := eh.ClassifyError(err, step)

	// Check if error is retryable
	if !sagaErr.Retryable {
		logger.GetSugaredLogger().Infof(
			"Error is not retryable: %s (type: %s, code: %s)",
			sagaErr.Message, sagaErr.Type, sagaErr.Code,
		)
		return false
	}

	// Use policy's ShouldRetry method
	return retryPolicy.ShouldRetry(err, attemptCount)
}

// GetRetryDelay calculates the delay before the next retry attempt.
func (eh *ErrorHandler) GetRetryDelay(step saga.SagaStep, attemptCount int) time.Duration {
	retryPolicy := step.GetRetryPolicy()
	if retryPolicy == nil {
		retryPolicy = eh.coordinator.retryPolicy
	}

	return retryPolicy.GetRetryDelay(attemptCount)
}

// GetMaxAttempts returns the maximum number of retry attempts for a step.
func (eh *ErrorHandler) GetMaxAttempts(step saga.SagaStep) int {
	retryPolicy := step.GetRetryPolicy()
	if retryPolicy == nil {
		retryPolicy = eh.coordinator.retryPolicy
	}

	return retryPolicy.GetMaxAttempts()
}

// RecordAttempt records a retry attempt in the history.
func (eh *ErrorHandler) RecordAttempt(
	step saga.SagaStep,
	attempt int,
	startTime time.Time,
	endTime time.Time,
	err error,
	willRetry bool,
) {
	stepID := step.GetID()

	// Get or create retry history
	history, exists := eh.retryHistories[stepID]
	if !exists {
		history = &RetryHistory{
			StepID:    stepID,
			StepName:  step.GetName(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		eh.retryHistories[stepID] = history
	}

	// Create attempt record
	attemptRecord := &RetryAttempt{
		Attempt:   attempt,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Success:   err == nil,
		WillRetry: willRetry,
	}

	if err != nil {
		attemptRecord.Error = eh.ClassifyError(err, step)
	}

	// Add to history
	history.AddAttempt(attemptRecord)

	// Log the attempt
	if err == nil {
		logger.GetSugaredLogger().Infof(
			"Step %s (attempt %d/%d) succeeded in %v",
			step.GetName(),
			attempt,
			eh.GetMaxAttempts(step),
			attemptRecord.Duration,
		)
	} else {
		logger.GetSugaredLogger().Warnf(
			"Step %s (attempt %d/%d) failed in %v: %v (will retry: %v)",
			step.GetName(),
			attempt,
			eh.GetMaxAttempts(step),
			attemptRecord.Duration,
			err,
			willRetry,
		)
	}
}

// GetRetryHistory returns the retry history for a specific step.
func (eh *ErrorHandler) GetRetryHistory(stepID string) *RetryHistory {
	return eh.retryHistories[stepID]
}

// GetAllRetryHistories returns all retry histories.
func (eh *ErrorHandler) GetAllRetryHistories() map[string]*RetryHistory {
	// Return a copy to prevent external mutation
	histories := make(map[string]*RetryHistory, len(eh.retryHistories))
	for k, v := range eh.retryHistories {
		histories[k] = v
	}
	return histories
}

// HandleStepError handles a step execution error with retry logic.
// Returns true if the error was handled (retry will occur), false if it should fail.
func (eh *ErrorHandler) HandleStepError(
	ctx context.Context,
	step saga.SagaStep,
	err error,
	attemptCount int,
) (shouldRetry bool, retryDelay time.Duration) {
	// Classify the error
	sagaErr := eh.ClassifyError(err, step)

	// Log the error
	logger.GetSugaredLogger().Errorf(
		"Step %s failed (attempt %d): %s (type: %s, retryable: %v)",
		step.GetName(),
		attemptCount,
		sagaErr.Message,
		sagaErr.Type,
		sagaErr.Retryable,
	)

	// Check if we should retry
	shouldRetry = eh.ShouldRetry(err, step, attemptCount)
	if !shouldRetry {
		logger.GetSugaredLogger().Errorf(
			"Step %s will not be retried (attempt %d/%d, error type: %s, retryable: %v)",
			step.GetName(),
			attemptCount,
			eh.GetMaxAttempts(step),
			sagaErr.Type,
			sagaErr.Retryable,
		)
		return false, 0
	}

	// Calculate retry delay
	retryDelay = eh.GetRetryDelay(step, attemptCount)
	logger.GetSugaredLogger().Infof(
		"Step %s will be retried after %v (attempt %d/%d)",
		step.GetName(),
		retryDelay,
		attemptCount,
		eh.GetMaxAttempts(step),
	)

	return true, retryDelay
}

// HandleMaxRetriesExceeded handles the case where max retries have been exceeded.
// This triggers compensation for all completed steps.
func (eh *ErrorHandler) HandleMaxRetriesExceeded(
	ctx context.Context,
	step saga.SagaStep,
	finalError error,
	attemptCount int,
) error {
	// Classify the final error
	sagaErr := eh.ClassifyError(finalError, step)

	// Log the exhaustion
	logger.GetSugaredLogger().Errorf(
		"Step %s exhausted all retry attempts (%d/%d): %s",
		step.GetName(),
		attemptCount,
		eh.GetMaxAttempts(step),
		sagaErr.Message,
	)

	// Create retry exhausted error
	retryExhaustedErr := saga.NewRetryExhaustedError(
		fmt.Sprintf("step_%s", step.GetID()),
		attemptCount,
	)
	retryExhaustedErr.Cause = sagaErr
	retryExhaustedErr.WithDetails(map[string]interface{}{
		"step_id":          step.GetID(),
		"step_name":        step.GetName(),
		"saga_id":          eh.instance.id,
		"final_error":      sagaErr.Message,
		"final_error_type": sagaErr.Type,
		"attempts":         attemptCount,
		"max_attempts":     eh.GetMaxAttempts(step),
		"retry_history":    eh.GetRetryHistory(step.GetID()),
	})

	// Update instance error
	eh.instance.mu.Lock()
	eh.instance.sagaError = retryExhaustedErr
	eh.instance.mu.Unlock()

	return retryExhaustedErr
}

// containsAny checks if a string contains any of the given substrings.
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	// Simple case-insensitive check
	sLower := toLower(s)
	substrLower := toLower(substr)
	return indexOf(sLower, substrLower) >= 0
}

// toLower converts a string to lowercase.
func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

// indexOf finds the index of a substring in a string.
func indexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		found := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}

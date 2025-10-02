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
	"testing"
	"time"
)

func TestSagaState_String(t *testing.T) {
	tests := []struct {
		state    SagaState
		expected string
	}{
		{StatePending, "pending"},
		{StateRunning, "running"},
		{StateStepCompleted, "step_completed"},
		{StateCompleted, "completed"},
		{StateCompensating, "compensating"},
		{StateCompensated, "compensated"},
		{StateFailed, "failed"},
		{StateCancelled, "cancelled"},
		{StateTimedOut, "timed_out"},
		{SagaState(999), "unknown"}, // Invalid state
	}

	for _, test := range tests {
		if result := test.state.String(); result != test.expected {
			t.Errorf("Expected %s, got %s for state %d", test.expected, result, test.state)
		}
	}
}

func TestSagaState_IsTerminal(t *testing.T) {
	tests := []struct {
		state    SagaState
		expected bool
	}{
		{StatePending, false},
		{StateRunning, false},
		{StateStepCompleted, false},
		{StateCompleted, true},
		{StateCompensating, false},
		{StateCompensated, true},
		{StateFailed, true},
		{StateCancelled, true},
		{StateTimedOut, true},
	}

	for _, test := range tests {
		if result := test.state.IsTerminal(); result != test.expected {
			t.Errorf("Expected %v, got %v for state %d", test.expected, result, test.state)
		}
	}
}

func TestSagaState_IsActive(t *testing.T) {
	tests := []struct {
		state    SagaState
		expected bool
	}{
		{StatePending, false},
		{StateRunning, true},
		{StateStepCompleted, true},
		{StateCompleted, false},
		{StateCompensating, true},
		{StateCompensated, false},
		{StateFailed, false},
		{StateCancelled, false},
		{StateTimedOut, false},
	}

	for _, test := range tests {
		if result := test.state.IsActive(); result != test.expected {
			t.Errorf("Expected %v, got %v for state %d", test.expected, result, test.state)
		}
	}
}

func TestStepStateEnum_String(t *testing.T) {
	tests := []struct {
		state    StepStateEnum
		expected string
	}{
		{StepStatePending, "pending"},
		{StepStateRunning, "running"},
		{StepStateCompleted, "completed"},
		{StepStateFailed, "failed"},
		{StepStateCompensating, "compensating"},
		{StepStateCompensated, "compensated"},
		{StepStateSkipped, "skipped"},
		{StepStateEnum(999), "unknown"}, // Invalid state
	}

	for _, test := range tests {
		if result := test.state.String(); result != test.expected {
			t.Errorf("Expected %s, got %s for state %d", test.expected, result, test.state)
		}
	}
}

func TestCompensationStateEnum_String(t *testing.T) {
	tests := []struct {
		state    CompensationStateEnum
		expected string
	}{
		{CompensationStatePending, "pending"},
		{CompensationStateRunning, "running"},
		{CompensationStateCompleted, "completed"},
		{CompensationStateFailed, "failed"},
		{CompensationStateSkipped, "skipped"},
		{CompensationStateEnum(999), "unknown"}, // Invalid state
	}

	for _, test := range tests {
		if result := test.state.String(); result != test.expected {
			t.Errorf("Expected %s, got %s for state %d", test.expected, result, test.state)
		}
	}
}

func TestSagaError_Error(t *testing.T) {
	// Test basic error
	err := NewSagaError("TEST_CODE", "Test message", ErrorTypeValidation, false)
	expected := "TEST_CODE: Test message"
	if result := err.Error(); result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test error with cause
	cause := NewSagaError("CAUSE_CODE", "Cause message", ErrorTypeSystem, true)
	errWithCause := WrapError(cause, "WRAPPER_CODE", "Wrapper message", ErrorTypeValidation, false)

	result := errWithCause.Error()
	if result != "WRAPPER_CODE: Wrapper message (caused by: CAUSE_CODE: Cause message)" {
		t.Errorf("Unexpected error message with cause: %s", result)
	}
}

func TestSagaError_WithStackTrace(t *testing.T) {
	err := NewSagaError("TEST_CODE", "Test message", ErrorTypeValidation, false)
	errWithStack := err.WithStackTrace()

	if errWithStack.StackTrace == "" {
		t.Error("Expected non-empty stack trace")
	}

	// Original error should be unchanged
	if err.StackTrace != "" {
		t.Error("Original error should not have stack trace")
	}
}

func TestSagaError_WithDetail(t *testing.T) {
	err := NewSagaError("TEST_CODE", "Test message", ErrorTypeValidation, false)
	err.WithDetail("key1", "value1")

	if err.Details["key1"] != "value1" {
		t.Errorf("Expected value1, got %v", err.Details["key1"])
	}

	// Test adding another detail
	err.WithDetail("key2", 42)
	if err.Details["key2"] != 42 {
		t.Errorf("Expected 42, got %v", err.Details["key2"])
	}
}

func TestSagaError_WithDetails(t *testing.T) {
	err := NewSagaError("TEST_CODE", "Test message", ErrorTypeValidation, false)
	details := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	err.WithDetails(details)

	if err.Details["key1"] != "value1" {
		t.Errorf("Expected value1, got %v", err.Details["key1"])
	}
	if err.Details["key2"] != 42 {
		t.Errorf("Expected 42, got %v", err.Details["key2"])
	}
}

func TestSagaError_GetChain(t *testing.T) {
	// Create error chain
	cause1 := NewSagaError("CAUSE1", "Cause 1", ErrorTypeSystem, true)
	cause2 := WrapError(cause1, "CAUSE2", "Cause 2", ErrorTypeValidation, false)
	mainErr := WrapError(cause2, "MAIN", "Main error", ErrorTypeService, true)

	chain := mainErr.GetChain()

	if len(chain) != 3 {
		t.Errorf("Expected 3 errors in chain, got %d", len(chain))
	}

	if chain[0].Code != "MAIN" {
		t.Errorf("Expected MAIN, got %s", chain[0].Code)
	}

	if chain[1].Code != "CAUSE2" {
		t.Errorf("Expected CAUSE2, got %s", chain[1].Code)
	}

	if chain[2].Code != "CAUSE1" {
		t.Errorf("Expected CAUSE1, got %s", chain[2].Code)
	}
}

func TestSagaError_IsRetryable(t *testing.T) {
	// Test retryable error
	retryableErr := NewSagaError("RETRYABLE", "Retryable error", ErrorTypeNetwork, true)
	if !retryableErr.IsRetryable() {
		t.Error("Expected retryable error to be retryable")
	}

	// Test non-retryable error
	nonRetryableErr := NewSagaError("NON_RETRYABLE", "Non-retryable error", ErrorTypeValidation, false)
	if nonRetryableErr.IsRetryable() {
		t.Error("Expected non-retryable error to not be retryable")
	}

	// Test error chain with retryable cause
	cause := NewSagaError("RETRYABLE_CAUSE", "Retryable cause", ErrorTypeNetwork, true)
	wrappedErr := WrapError(cause, "WRAPPER", "Wrapper error", ErrorTypeValidation, false)
	if !wrappedErr.IsRetryable() {
		t.Error("Expected wrapped error to be retryable due to retryable cause")
	}
}

func TestSagaError_GetRootCause(t *testing.T) {
	// Test error without cause
	err := NewSagaError("ROOT", "Root error", ErrorTypeSystem, false)
	root := err.GetRootCause()
	if root != err {
		t.Error("Expected root cause to be the error itself when no cause exists")
	}

	// Test error with cause
	cause := NewSagaError("ROOT_CAUSE", "Root cause", ErrorTypeSystem, false)
	wrappedErr := WrapError(cause, "WRAPPER", "Wrapper error", ErrorTypeValidation, false)
	root = wrappedErr.GetRootCause()
	if root != cause {
		t.Error("Expected root cause to be the cause error")
	}
}

func TestErrorConstructors(t *testing.T) {
	// Test NewSagaNotFoundError
	err := NewSagaNotFoundError("test-saga-id")
	if err.Code != ErrCodeSagaNotFound {
		t.Errorf("Expected %s, got %s", ErrCodeSagaNotFound, err.Code)
	}
	if err.Details["saga_id"] != "test-saga-id" {
		t.Errorf("Expected test-saga-id, got %v", err.Details["saga_id"])
	}

	// Test NewSagaAlreadyExistsError
	err = NewSagaAlreadyExistsError("test-saga-id")
	if err.Code != ErrCodeSagaAlreadyExists {
		t.Errorf("Expected %s, got %s", ErrCodeSagaAlreadyExists, err.Code)
	}

	// Test NewInvalidSagaStateError
	err = NewInvalidSagaStateError(StateRunning, StateCompleted)
	if err.Code != ErrCodeInvalidSagaState {
		t.Errorf("Expected %s, got %s", ErrCodeInvalidSagaState, err.Code)
	}

	// Test NewSagaTimeoutError
	err = NewSagaTimeoutError("test-saga-id", 30*time.Second)
	if err.Code != ErrCodeSagaTimeout {
		t.Errorf("Expected %s, got %s", ErrCodeSagaTimeout, err.Code)
	}
	if err.Details["timeout"] != "30s" {
		t.Errorf("Expected 30s, got %v", err.Details["timeout"])
	}

	// Test NewValidationError
	err = NewValidationError("Invalid input")
	if err.Code != ErrCodeValidationError {
		t.Errorf("Expected %s, got %s", ErrCodeValidationError, err.Code)
	}
	if err.Type != ErrorTypeValidation {
		t.Errorf("Expected %s, got %s", ErrorTypeValidation, err.Type)
	}
}

func TestErrorCheckers(t *testing.T) {
	// Test IsSagaNotFound
	err := NewSagaNotFoundError("test-id")
	if !IsSagaNotFound(err) {
		t.Error("Expected true for SagaNotFoundError")
	}

	otherErr := NewValidationError("test")
	if IsSagaNotFound(otherErr) {
		t.Error("Expected false for non-SagaNotFoundError")
	}

	// Test IsSagaTimeout
	err = NewSagaTimeoutError("test-id", 30*time.Second)
	if !IsSagaTimeout(err) {
		t.Error("Expected true for SagaTimeoutError")
	}

	// Test IsRetryableError
	retryableErr := NewSagaError("RETRYABLE", "test", ErrorTypeNetwork, true)
	if !IsRetryableError(retryableErr) {
		t.Error("Expected true for retryable error")
	}

	nonRetryableErr := NewSagaError("NON_RETRYABLE", "test", ErrorTypeValidation, false)
	if IsRetryableError(nonRetryableErr) {
		t.Error("Expected false for non-retryable error")
	}
}

func TestSagaFilter(t *testing.T) {
	filter := &SagaFilter{
		States:        []SagaState{StateRunning, StateCompleted},
		DefinitionIDs: []string{"def1", "def2"},
		Limit:         10,
		Offset:        0,
		SortBy:        "created_at",
		SortOrder:     "desc",
		Metadata:      map[string]interface{}{"key": "value"},
	}

	if len(filter.States) != 2 {
		t.Errorf("Expected 2 states, got %d", len(filter.States))
	}

	if filter.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", filter.Limit)
	}

	if filter.SortBy != "created_at" {
		t.Errorf("Expected sort by created_at, got %s", filter.SortBy)
	}
}

func TestCoordinatorMetrics(t *testing.T) {
	metrics := &CoordinatorMetrics{
		TotalSagas:              100,
		ActiveSagas:             10,
		CompletedSagas:          80,
		FailedSagas:             5,
		CancelledSagas:          3,
		TimedOutSagas:           2,
		TotalSteps:              500,
		CompletedSteps:          450,
		FailedSteps:             30,
		AverageSagaDuration:     5 * time.Minute,
		AverageStepDuration:     30 * time.Second,
		TotalRetries:            25,
		SuccessfulRetries:       20,
		TotalCompensations:      8,
		SuccessfulCompensations: 7,
		StartTime:               time.Now().Add(-1 * time.Hour),
		LastUpdateTime:          time.Now(),
	}

	if metrics.TotalSagas != 100 {
		t.Errorf("Expected 100 total sagas, got %d", metrics.TotalSagas)
	}

	if metrics.ActiveSagas != 10 {
		t.Errorf("Expected 10 active sagas, got %d", metrics.ActiveSagas)
	}

	if metrics.AverageSagaDuration != 5*time.Minute {
		t.Errorf("Expected 5m average duration, got %v", metrics.AverageSagaDuration)
	}
}

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
	"errors"
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

// TestMoreErrorConstructors tests additional error constructors not covered elsewhere
func TestMoreErrorConstructors(t *testing.T) {
	t.Run("NewStepExecutionError", func(t *testing.T) {
		cause := NewSagaError("CAUSE", "original error", ErrorTypeSystem, false)
		err := NewStepExecutionError("step-1", "Test Step", cause)

		if err.Code != ErrCodeStepExecutionFailed {
			t.Errorf("Expected %s, got %s", ErrCodeStepExecutionFailed, err.Code)
		}
		if err.Details["step_id"] != "step-1" {
			t.Errorf("Expected step-1, got %v", err.Details["step_id"])
		}
		if err.Details["step_name"] != "Test Step" {
			t.Errorf("Expected Test Step, got %v", err.Details["step_name"])
		}
		if err.Cause == nil {
			t.Error("Expected cause to be wrapped")
		}
	})

	t.Run("NewCompensationFailedError", func(t *testing.T) {
		cause := NewSagaError("COMPENSATION_CAUSE", "compensation failed", ErrorTypeSystem, false)
		err := NewCompensationFailedError("step-2", "Compensation Step", cause)

		if err.Code != ErrCodeCompensationFailed {
			t.Errorf("Expected %s, got %s", ErrCodeCompensationFailed, err.Code)
		}
		if err.Details["step_id"] != "step-2" {
			t.Errorf("Expected step-2, got %v", err.Details["step_id"])
		}
		if err.Details["step_name"] != "Compensation Step" {
			t.Errorf("Expected Compensation Step, got %v", err.Details["step_name"])
		}
		if err.Type != ErrorTypeCompensation {
			t.Errorf("Expected %s, got %s", ErrorTypeCompensation, err.Type)
		}
	})

	t.Run("NewStorageError", func(t *testing.T) {
		cause := NewSagaError("DB_ERROR", "database error", ErrorTypeSystem, true)
		err := NewStorageError("save", cause)

		if err.Code != ErrCodeStorageError {
			t.Errorf("Expected %s, got %s", ErrCodeStorageError, err.Code)
		}
		if err.Details["operation"] != "save" {
			t.Errorf("Expected save, got %v", err.Details["operation"])
		}
		if !err.Retryable {
			t.Error("Expected storage error to be retryable")
		}
	})

	t.Run("NewConfigurationError", func(t *testing.T) {
		err := NewConfigurationError("Invalid timeout value")

		if err.Code != ErrCodeConfigurationError {
			t.Errorf("Expected %s, got %s", ErrCodeConfigurationError, err.Code)
		}
		if err.Message != "Invalid timeout value" {
			t.Errorf("Expected 'Invalid timeout value', got '%s'", err.Message)
		}
		if err.Type != ErrorTypeSystem {
			t.Errorf("Expected %s, got %s", ErrorTypeSystem, err.Type)
		}
	})

	t.Run("NewCoordinatorStoppedError", func(t *testing.T) {
		err := NewCoordinatorStoppedError()

		if err.Code != ErrCodeCoordinatorStopped {
			t.Errorf("Expected %s, got %s", ErrCodeCoordinatorStopped, err.Code)
		}
		if err.Type != ErrorTypeSystem {
			t.Errorf("Expected %s, got %s", ErrorTypeSystem, err.Type)
		}
	})

	t.Run("NewRetryExhaustedError", func(t *testing.T) {
		err := NewRetryExhaustedError("execute-step", 5)

		if err.Code != ErrCodeRetryExhausted {
			t.Errorf("Expected %s, got %s", ErrCodeRetryExhausted, err.Code)
		}
		if err.Details["operation"] != "execute-step" {
			t.Errorf("Expected execute-step, got %v", err.Details["operation"])
		}
		if err.Details["attempts"] != 5 {
			t.Errorf("Expected 5, got %v", err.Details["attempts"])
		}
	})

	t.Run("NewEventPublishError", func(t *testing.T) {
		cause := NewSagaError("PUBLISH_FAILED", "broker unavailable", ErrorTypeNetwork, true)
		err := NewEventPublishError(EventSagaStarted, cause)

		if err.Code != ErrCodeEventPublishFailed {
			t.Errorf("Expected %s, got %s", ErrCodeEventPublishFailed, err.Code)
		}
		if err.Details["event_type"] != string(EventSagaStarted) {
			t.Errorf("Expected %s, got %v", string(EventSagaStarted), err.Details["event_type"])
		}
		if !err.Retryable {
			t.Error("Expected event publish error to be retryable")
		}
	})
}

// TestMoreErrorCheckers tests additional error checker functions
func TestMoreErrorCheckers(t *testing.T) {
	t.Run("IsStepExecutionFailed", func(t *testing.T) {
		cause := NewSagaError("CAUSE", "error", ErrorTypeSystem, false)
		err := NewStepExecutionError("step-1", "Test", cause)

		if !IsStepExecutionFailed(err) {
			t.Error("Expected true for StepExecutionError")
		}

		otherErr := NewValidationError("test")
		if IsStepExecutionFailed(otherErr) {
			t.Error("Expected false for non-StepExecutionError")
		}
	})

	t.Run("IsCompensationFailed", func(t *testing.T) {
		cause := NewSagaError("CAUSE", "error", ErrorTypeSystem, false)
		err := NewCompensationFailedError("step-1", "Test", cause)

		if !IsCompensationFailed(err) {
			t.Error("Expected true for CompensationFailedError")
		}

		otherErr := NewValidationError("test")
		if IsCompensationFailed(otherErr) {
			t.Error("Expected false for non-CompensationFailedError")
		}
	})

	t.Run("IsRetryExhausted", func(t *testing.T) {
		err := NewRetryExhaustedError("operation", 3)

		if !IsRetryExhausted(err) {
			t.Error("Expected true for RetryExhaustedError")
		}

		otherErr := NewValidationError("test")
		if IsRetryExhausted(otherErr) {
			t.Error("Expected false for non-RetryExhaustedError")
		}
	})
}

// TestWrapErrorEdgeCases tests edge cases for WrapError
func TestWrapErrorEdgeCases(t *testing.T) {
	t.Run("WrapError with nil error", func(t *testing.T) {
		err := WrapError(nil, "CODE", "message", ErrorTypeSystem, false)
		if err != nil {
			t.Error("Expected nil when wrapping nil error")
		}
	})

	t.Run("WrapError with SagaError", func(t *testing.T) {
		originalErr := NewSagaError("ORIGINAL", "original message", ErrorTypeSystem, true)
		wrappedErr := WrapError(originalErr, "WRAPPER", "wrapper message", ErrorTypeValidation, false)

		if wrappedErr.Cause != originalErr {
			t.Error("Expected original SagaError to be preserved as cause")
		}
	})

	t.Run("WrapError with non-SagaError", func(t *testing.T) {
		originalErr := errors.New("standard error")
		wrappedErr := WrapError(originalErr, "WRAPPER", "wrapper message", ErrorTypeValidation, false)

		if wrappedErr.Cause == nil {
			t.Error("Expected cause to be created")
		}
		if wrappedErr.Cause.Code != "WRAPPED_ERROR" {
			t.Errorf("Expected WRAPPED_ERROR, got %s", wrappedErr.Cause.Code)
		}
	})
}

// TestEventTypeFilterGetDescription tests GetDescription method
func TestEventTypeFilterGetDescription(t *testing.T) {
	tests := []struct {
		name     string
		filter   *EventTypeFilter
		expected string
	}{
		{
			name:     "Empty filter",
			filter:   &EventTypeFilter{},
			expected: "all events",
		},
		{
			name: "Filter with types only",
			filter: &EventTypeFilter{
				Types: []SagaEventType{EventSagaStarted, EventSagaCompleted},
			},
			expected: "events matching types: saga.started, saga.completed",
		},
		{
			name: "Filter with exclusions only",
			filter: &EventTypeFilter{
				ExcludedTypes: []SagaEventType{EventSagaFailed},
			},
			expected: "events matching types:  (excluding: saga.failed)",
		},
		{
			name: "Filter with types and exclusions",
			filter: &EventTypeFilter{
				Types:         []SagaEventType{EventSagaStarted},
				ExcludedTypes: []SagaEventType{EventSagaFailed},
			},
			expected: "events matching types: saga.started (excluding: saga.failed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := tt.filter.GetDescription()
			if desc != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, desc)
			}
		})
	}
}

// TestSagaIDFilterGetDescription tests GetDescription method for SagaIDFilter
func TestSagaIDFilterGetDescription(t *testing.T) {
	tests := []struct {
		name     string
		filter   *SagaIDFilter
		expected string
	}{
		{
			name:     "Empty filter",
			filter:   &SagaIDFilter{},
			expected: "all Sagas",
		},
		{
			name: "Filter with IDs only",
			filter: &SagaIDFilter{
				SagaIDs: []string{"saga-1", "saga-2"},
			},
			expected: "events for Sagas: saga-1, saga-2",
		},
		{
			name: "Filter with exclusions only",
			filter: &SagaIDFilter{
				ExcludeSagaIDs: []string{"saga-3"},
			},
			expected: "events for Sagas:  (excluding: saga-3)",
		},
		{
			name: "Filter with IDs and exclusions",
			filter: &SagaIDFilter{
				SagaIDs:        []string{"saga-1"},
				ExcludeSagaIDs: []string{"saga-2"},
			},
			expected: "events for Sagas: saga-1 (excluding: saga-2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := tt.filter.GetDescription()
			if desc != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, desc)
			}
		})
	}
}

// TestCompositeFilterGetDescription tests GetDescription method for CompositeFilter
func TestCompositeFilterGetDescription(t *testing.T) {
	tests := []struct {
		name     string
		filter   *CompositeFilter
		expected string
	}{
		{
			name:     "Empty composite filter",
			filter:   &CompositeFilter{},
			expected: "all events",
		},
		{
			name: "AND composite filter",
			filter: &CompositeFilter{
				Filters: []EventFilter{
					&EventTypeFilter{Types: []SagaEventType{EventSagaStarted}},
					&SagaIDFilter{SagaIDs: []string{"saga-1"}},
				},
				Operation: FilterOperationAND,
			},
			expected: "events matching (events matching types: saga.started AND events for Sagas: saga-1)",
		},
		{
			name: "OR composite filter",
			filter: &CompositeFilter{
				Filters: []EventFilter{
					&EventTypeFilter{Types: []SagaEventType{EventSagaStarted}},
					&SagaIDFilter{SagaIDs: []string{"saga-1"}},
				},
				Operation: FilterOperationOR,
			},
			expected: "events matching (events matching types: saga.started OR events for Sagas: saga-1)",
		},
		{
			name: "Default operation (empty)",
			filter: &CompositeFilter{
				Filters: []EventFilter{
					&EventTypeFilter{Types: []SagaEventType{EventSagaStarted}},
				},
				Operation: "",
			},
			expected: "events matching (events matching types: saga.started)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := tt.filter.GetDescription()
			if desc != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, desc)
			}
		})
	}
}

// TestBasicEventSubscriptionGetCreatedAt tests GetCreatedAt method
func TestBasicEventSubscriptionGetCreatedAt(t *testing.T) {
	now := time.Now()
	subscription := &BasicEventSubscription{
		ID:        "test-sub",
		CreatedAt: now,
	}

	if subscription.GetCreatedAt() != now {
		t.Errorf("Expected %v, got %v", now, subscription.GetCreatedAt())
	}
}

// TestExponentialBackoffRetryPolicySetMultiplier tests SetMultiplier method
func TestExponentialBackoffRetryPolicySetMultiplier(t *testing.T) {
	policy := NewExponentialBackoffRetryPolicy(3, 100*time.Millisecond, 10*time.Second)

	// Set custom multiplier
	policy.SetMultiplier(3.0)
	policy.SetJitter(false)

	// Test with custom multiplier
	delay1 := policy.GetRetryDelay(0) // 100ms
	delay2 := policy.GetRetryDelay(1) // 100ms * 3 = 300ms
	delay3 := policy.GetRetryDelay(2) // 100ms * 9 = 900ms

	if delay1 != 100*time.Millisecond {
		t.Errorf("Expected 100ms, got %v", delay1)
	}
	if delay2 != 300*time.Millisecond {
		t.Errorf("Expected 300ms, got %v", delay2)
	}
	if delay3 != 900*time.Millisecond {
		t.Errorf("Expected 900ms, got %v", delay3)
	}
}

// TestExponentialBackoffRetryPolicyWithJitter tests jitter behavior
func TestExponentialBackoffRetryPolicyWithJitter(t *testing.T) {
	policy := NewExponentialBackoffRetryPolicy(3, 100*time.Millisecond, 10*time.Second)
	policy.SetJitter(true)

	// With jitter enabled, delays should vary
	// We can't test exact values, but we can ensure it doesn't crash
	delay := policy.GetRetryDelay(1)
	if delay <= 0 {
		t.Error("Expected positive delay")
	}
}

// TestEventTypeFilterMatch_EdgeCases tests edge cases for EventTypeFilter.Match
func TestEventTypeFilterMatch_EdgeCases(t *testing.T) {
	t.Run("Match with empty exclusions", func(t *testing.T) {
		filter := &EventTypeFilter{
			Types:         []SagaEventType{EventSagaStarted},
			ExcludedTypes: []SagaEventType{},
		}
		event := &SagaEvent{Type: EventSagaStarted}

		if !filter.Match(event) {
			t.Error("Expected match when exclusions are empty")
		}
	})

	t.Run("Match with no types but with exclusions", func(t *testing.T) {
		filter := &EventTypeFilter{
			Types:         []SagaEventType{},
			ExcludedTypes: []SagaEventType{EventSagaFailed},
		}
		event1 := &SagaEvent{Type: EventSagaStarted}
		event2 := &SagaEvent{Type: EventSagaFailed}

		if !filter.Match(event1) {
			t.Error("Expected match for non-excluded event")
		}
		if filter.Match(event2) {
			t.Error("Expected no match for excluded event")
		}
	})
}

// TestCompositeFilterMatch_EdgeCases tests edge cases for CompositeFilter.Match
func TestCompositeFilterMatch_EdgeCases(t *testing.T) {
	t.Run("Match with unknown operation defaults to AND", func(t *testing.T) {
		filter := &CompositeFilter{
			Filters: []EventFilter{
				&EventTypeFilter{Types: []SagaEventType{EventSagaStarted}},
				&SagaIDFilter{SagaIDs: []string{"saga-1"}},
			},
			Operation: "UNKNOWN",
		}
		event := &SagaEvent{Type: EventSagaStarted, SagaID: "saga-1"}

		if !filter.Match(event) {
			t.Error("Expected match when both filters match (default AND)")
		}
	})
}

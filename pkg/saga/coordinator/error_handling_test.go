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
	"errors"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// mockStep is a mock implementation of saga.SagaStep for testing.
type mockStep struct {
	id          string
	name        string
	retryable   bool
	retryPolicy saga.RetryPolicy
	executeFunc func(ctx context.Context, data interface{}) (interface{}, error)
}

func (m *mockStep) GetID() string                            { return m.id }
func (m *mockStep) GetName() string                          { return m.name }
func (m *mockStep) GetDescription() string                   { return "mock step" }
func (m *mockStep) GetTimeout() time.Duration                { return 5 * time.Second }
func (m *mockStep) GetRetryPolicy() saga.RetryPolicy         { return m.retryPolicy }
func (m *mockStep) GetMetadata() map[string]interface{}      { return nil }
func (m *mockStep) IsRetryable(err error) bool               { return m.retryable }
func (m *mockStep) Compensate(ctx context.Context, data interface{}) error { return nil }

func (m *mockStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, data)
	}
	return nil, nil
}

// TestErrorHandler_ClassifyError tests error classification.
func TestErrorHandler_ClassifyError(t *testing.T) {
	coordinator := createTestCoordinator(t)
	instance := createTestInstance()
	eh := newErrorHandler(coordinator, instance)

	tests := []struct {
		name          string
		err           error
		expectedType  saga.ErrorType
		expectedCode  string
		stepRetryable bool
	}{
		{
			name:          "timeout error",
			err:           context.DeadlineExceeded,
			expectedType:  saga.ErrorTypeTimeout,
			expectedCode:  saga.ErrCodeSagaTimeout,
			stepRetryable: true,
		},
		{
			name:          "cancelled error",
			err:           context.Canceled,
			expectedType:  saga.ErrorTypeSystem,
			expectedCode:  "SYSTEM_ERROR",
			stepRetryable: true,
		},
		{
			name:          "network error",
			err:           errors.New("connection refused"),
			expectedType:  saga.ErrorTypeNetwork,
			expectedCode:  "NETWORK_ERROR",
			stepRetryable: true,
		},
		{
			name:          "validation error",
			err:           errors.New("invalid input data"),
			expectedType:  saga.ErrorTypeValidation,
			expectedCode:  saga.ErrCodeValidationError,
			stepRetryable: true,
		},
		{
			name:          "data error",
			err:           errors.New("record not found"),
			expectedType:  saga.ErrorTypeData,
			expectedCode:  "DATA_ERROR",
			stepRetryable: true,
		},
		{
			name:          "service error",
			err:           errors.New("service error occurred"),
			expectedType:  saga.ErrorTypeService,
			expectedCode:  saga.ErrCodeStepExecutionFailed,
			stepRetryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &mockStep{
				id:        "test-step",
				name:      "Test Step",
				retryable: tt.stepRetryable,
			}
			
			sagaErr := eh.ClassifyError(tt.err, step)

			if sagaErr == nil {
				t.Fatal("expected SagaError, got nil")
			}

			if sagaErr.Type != tt.expectedType {
				t.Errorf("expected error type %s, got %s", tt.expectedType, sagaErr.Type)
			}

			if sagaErr.Code != tt.expectedCode {
				t.Errorf("expected error code %s, got %s", tt.expectedCode, sagaErr.Code)
			}
		})
	}
}

// TestErrorHandler_ShouldRetry tests retry decision logic.
func TestErrorHandler_ShouldRetry(t *testing.T) {
	coordinator := createTestCoordinator(t)
	instance := createTestInstance()
	eh := newErrorHandler(coordinator, instance)

	retryPolicy := saga.NewFixedDelayRetryPolicy(3, time.Second)

	tests := []struct {
		name          string
		err           error
		attemptCount  int
		stepRetryable bool
		shouldRetry   bool
	}{
		{
			name:          "retryable error within max attempts",
			err:           errors.New("temporary failure"),
			attemptCount:  1,
			stepRetryable: true,
			shouldRetry:   true,
		},
		{
			name:          "retryable error at max attempts",
			err:           errors.New("temporary failure"),
			attemptCount:  3,
			stepRetryable: true,
			shouldRetry:   false,
		},
		{
			name:          "non-retryable error",
			err:           errors.New("invalid data"),
			attemptCount:  1,
			stepRetryable: false,
			shouldRetry:   false,
		},
		{
			name:          "context cancelled",
			err:           context.Canceled,
			attemptCount:  1,
			stepRetryable: false,
			shouldRetry:   false,
		},
		{
			name:          "nil error",
			err:           nil,
			attemptCount:  1,
			stepRetryable: true,
			shouldRetry:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &mockStep{
				id:          "test-step",
				name:        "Test Step",
				retryable:   tt.stepRetryable,
				retryPolicy: retryPolicy,
			}

			result := eh.ShouldRetry(tt.err, step, tt.attemptCount)
			if result != tt.shouldRetry {
				t.Errorf("expected ShouldRetry %v, got %v", tt.shouldRetry, result)
			}
		})
	}
}

// TestErrorHandler_RecordAttempt tests retry attempt recording.
func TestErrorHandler_RecordAttempt(t *testing.T) {
	coordinator := createTestCoordinator(t)
	instance := createTestInstance()
	eh := newErrorHandler(coordinator, instance)

	step := &mockStep{
		id:        "test-step",
		name:      "Test Step",
		retryable: true,
	}

	// Record successful attempt
	startTime := time.Now()
	time.Sleep(10 * time.Millisecond)
	endTime := time.Now()

	eh.RecordAttempt(step, 1, startTime, endTime, nil, false)

	history := eh.GetRetryHistory(step.GetID())
	if history == nil {
		t.Fatal("expected retry history, got nil")
	}

	if history.TotalAttempts != 1 {
		t.Errorf("expected 1 attempt, got %d", history.TotalAttempts)
	}

	lastAttempt := history.GetLastAttempt()
	if lastAttempt == nil {
		t.Fatal("expected last attempt, got nil")
	}

	if !lastAttempt.Success {
		t.Error("expected successful attempt")
	}

	if lastAttempt.Error != nil {
		t.Errorf("expected no error, got %v", lastAttempt.Error)
	}

	// Record failed attempt
	testErr := errors.New("test error")
	eh.RecordAttempt(step, 2, startTime, endTime, testErr, true)

	history = eh.GetRetryHistory(step.GetID())
	if history.TotalAttempts != 2 {
		t.Errorf("expected 2 attempts, got %d", history.TotalAttempts)
	}

	lastAttempt = history.GetLastAttempt()
	if lastAttempt.Success {
		t.Error("expected failed attempt")
	}

	if lastAttempt.Error == nil {
		t.Error("expected error in attempt")
	}

	if !lastAttempt.WillRetry {
		t.Error("expected WillRetry to be true")
	}
}

// TestErrorHandler_GetRetryDelay tests retry delay calculation.
func TestErrorHandler_GetRetryDelay(t *testing.T) {
	coordinator := createTestCoordinator(t)
	instance := createTestInstance()
	eh := newErrorHandler(coordinator, instance)

	tests := []struct {
		name         string
		retryPolicy  saga.RetryPolicy
		attempt      int
		minDelay     time.Duration
		maxDelay     time.Duration
	}{
		{
			name:         "fixed delay policy",
			retryPolicy:  saga.NewFixedDelayRetryPolicy(3, 2*time.Second),
			attempt:      1,
			minDelay:     2 * time.Second,
			maxDelay:     2 * time.Second,
		},
		{
			name:         "exponential backoff - attempt 0",
			retryPolicy:  saga.NewExponentialBackoffRetryPolicy(5, time.Second, 10*time.Second),
			attempt:      0,
			minDelay:     500 * time.Millisecond,
			maxDelay:     2 * time.Second,
		},
		{
			name:         "exponential backoff - attempt 2",
			retryPolicy:  saga.NewExponentialBackoffRetryPolicy(5, time.Second, 10*time.Second),
			attempt:      2,
			minDelay:     2 * time.Second,
			maxDelay:     6 * time.Second,
		},
		{
			name:         "linear backoff",
			retryPolicy:  saga.NewLinearBackoffRetryPolicy(3, time.Second, 500*time.Millisecond, 5*time.Second),
			attempt:      2,
			minDelay:     2 * time.Second,
			maxDelay:     2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &mockStep{
				id:          "test-step",
				name:        "Test Step",
				retryPolicy: tt.retryPolicy,
			}

			delay := eh.GetRetryDelay(step, tt.attempt)

			if delay < tt.minDelay || delay > tt.maxDelay {
				t.Errorf("expected delay between %v and %v, got %v", tt.minDelay, tt.maxDelay, delay)
			}
		})
	}
}

// TestErrorHandler_HandleStepError tests step error handling.
func TestErrorHandler_HandleStepError(t *testing.T) {
	coordinator := createTestCoordinator(t)
	instance := createTestInstance()
	eh := newErrorHandler(coordinator, instance)

	retryPolicy := saga.NewFixedDelayRetryPolicy(3, time.Second)

	tests := []struct {
		name             string
		err              error
		attemptCount     int
		stepRetryable    bool
		expectedRetry    bool
		expectedMinDelay time.Duration
	}{
		{
			name:             "retryable error - should retry",
			err:              errors.New("temporary error"),
			attemptCount:     1,
			stepRetryable:    true,
			expectedRetry:    true,
			expectedMinDelay: time.Second,
		},
		{
			name:             "retryable error - max attempts reached",
			err:              errors.New("temporary error"),
			attemptCount:     3,
			stepRetryable:    true,
			expectedRetry:    false,
			expectedMinDelay: 0,
		},
		{
			name:             "non-retryable error",
			err:              errors.New("invalid input"),
			attemptCount:     1,
			stepRetryable:    false,
			expectedRetry:    false,
			expectedMinDelay: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &mockStep{
				id:          "test-step",
				name:        "Test Step",
				retryable:   tt.stepRetryable,
				retryPolicy: retryPolicy,
			}

			shouldRetry, retryDelay := eh.HandleStepError(context.Background(), step, tt.err, tt.attemptCount)

			if shouldRetry != tt.expectedRetry {
				t.Errorf("expected shouldRetry %v, got %v", tt.expectedRetry, shouldRetry)
			}

			if shouldRetry && retryDelay < tt.expectedMinDelay {
				t.Errorf("expected retry delay >= %v, got %v", tt.expectedMinDelay, retryDelay)
			}

			if !shouldRetry && retryDelay != 0 {
				t.Errorf("expected retry delay 0, got %v", retryDelay)
			}
		})
	}
}

// TestErrorHandler_HandleMaxRetriesExceeded tests max retries exceeded handling.
func TestErrorHandler_HandleMaxRetriesExceeded(t *testing.T) {
	coordinator := createTestCoordinator(t)
	instance := createTestInstance()
	eh := newErrorHandler(coordinator, instance)

	step := &mockStep{
		id:          "test-step",
		name:        "Test Step",
		retryable:   true,
		retryPolicy: saga.NewFixedDelayRetryPolicy(3, time.Second),
	}

	originalErr := errors.New("persistent failure")
	err := eh.HandleMaxRetriesExceeded(context.Background(), step, originalErr, 3)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	sagaErr, ok := err.(*saga.SagaError)
	if !ok {
		t.Fatalf("expected *saga.SagaError, got %T", err)
	}

	if sagaErr.Code != saga.ErrCodeRetryExhausted {
		t.Errorf("expected code %s, got %s", saga.ErrCodeRetryExhausted, sagaErr.Code)
	}

	if sagaErr.Cause == nil {
		t.Error("expected cause to be set")
	}

	// Check that instance error was updated
	if instance.sagaError == nil {
		t.Error("expected instance saga error to be set")
	}

	if instance.sagaError.Code != saga.ErrCodeRetryExhausted {
		t.Errorf("expected instance error code %s, got %s", saga.ErrCodeRetryExhausted, instance.sagaError.Code)
	}
}

// TestRetryHistory_AddAttempt tests adding attempts to retry history.
func TestRetryHistory_AddAttempt(t *testing.T) {
	history := &RetryHistory{
		StepID:    "test-step",
		StepName:  "Test Step",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add successful attempt
	successAttempt := &RetryAttempt{
		Attempt:   1,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Duration:  10 * time.Millisecond,
		Success:   true,
		WillRetry: false,
	}
	history.AddAttempt(successAttempt)

	if history.TotalAttempts != 1 {
		t.Errorf("expected 1 total attempt, got %d", history.TotalAttempts)
	}

	if history.SuccessfulAttempt != 1 {
		t.Errorf("expected successful attempt 1, got %d", history.SuccessfulAttempt)
	}

	// Add failed attempt
	failedAttempt := &RetryAttempt{
		Attempt:   2,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Duration:  10 * time.Millisecond,
		Success:   false,
		WillRetry: true,
		Error: &saga.SagaError{
			Code:    "TEST_ERROR",
			Message: "test error",
		},
	}
	history.AddAttempt(failedAttempt)

	if history.TotalAttempts != 2 {
		t.Errorf("expected 2 total attempts, got %d", history.TotalAttempts)
	}

	if history.FinalError == nil {
		t.Error("expected final error to be set")
	}

	lastAttempt := history.GetLastAttempt()
	if lastAttempt != failedAttempt {
		t.Error("expected last attempt to match failed attempt")
	}
}

// TestErrorHandler_GetAllRetryHistories tests retrieving all retry histories.
func TestErrorHandler_GetAllRetryHistories(t *testing.T) {
	coordinator := createTestCoordinator(t)
	instance := createTestInstance()
	eh := newErrorHandler(coordinator, instance)

	step1 := &mockStep{id: "step-1", name: "Step 1"}
	step2 := &mockStep{id: "step-2", name: "Step 2"}

	// Record attempts for multiple steps
	eh.RecordAttempt(step1, 1, time.Now(), time.Now(), nil, false)
	eh.RecordAttempt(step2, 1, time.Now(), time.Now(), errors.New("error"), true)

	histories := eh.GetAllRetryHistories()

	if len(histories) != 2 {
		t.Errorf("expected 2 histories, got %d", len(histories))
	}

	if _, exists := histories["step-1"]; !exists {
		t.Error("expected history for step-1")
	}

	if _, exists := histories["step-2"]; !exists {
		t.Error("expected history for step-2")
	}
}

// Helper functions

func createTestCoordinator(t *testing.T) *OrchestratorCoordinator {
	config := &OrchestratorConfig{
		StateStorage:   &mockStateStorage{},
		EventPublisher: &mockEventPublisher{},
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(3, time.Second),
	}

	coordinator, err := NewOrchestratorCoordinator(config)
	if err != nil {
		t.Fatalf("failed to create coordinator: %v", err)
	}

	return coordinator
}

func createTestInstance() *OrchestratorSagaInstance {
	now := time.Now()
	return &OrchestratorSagaInstance{
		id:           "test-saga-001",
		definitionID: "test-definition",
		name:         "Test Saga",
		state:        saga.StatePending,
		createdAt:    now,
		updatedAt:    now,
		startedAt:    &now,
	}
}

// Mock implementations are reused from orchestrator_test.go


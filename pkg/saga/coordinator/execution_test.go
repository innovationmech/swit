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

// mockSagaStep is a mock implementation of saga.SagaStep for testing.
type mockSagaStep struct {
	id          string
	name        string
	description string
	executeFunc func(ctx context.Context, data interface{}) (interface{}, error)
	compensate  func(ctx context.Context, data interface{}) error
	timeout     time.Duration
	retryPolicy saga.RetryPolicy
	retryable   bool
	metadata    map[string]interface{}
}

func (m *mockSagaStep) GetID() string {
	return m.id
}

func (m *mockSagaStep) GetName() string {
	return m.name
}

func (m *mockSagaStep) GetDescription() string {
	return m.description
}

func (m *mockSagaStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, data)
	}
	return data, nil
}

func (m *mockSagaStep) Compensate(ctx context.Context, data interface{}) error {
	if m.compensate != nil {
		return m.compensate(ctx, data)
	}
	return nil
}

func (m *mockSagaStep) GetTimeout() time.Duration {
	return m.timeout
}

func (m *mockSagaStep) GetRetryPolicy() saga.RetryPolicy {
	return m.retryPolicy
}

func (m *mockSagaStep) IsRetryable(err error) bool {
	return m.retryable
}

func (m *mockSagaStep) GetMetadata() map[string]interface{} {
	return m.metadata
}

// TestStepExecutorExecuteStepsSuccess tests successful execution of all steps.
func TestStepExecutorExecuteStepsSuccess(t *testing.T) {
	// Setup mocks
	storage := &mockStateStorage{}
	publisher := &mockEventPublisher{}
	metrics := &mockMetricsCollector{}

	coordinator, err := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:     storage,
		EventPublisher:   publisher,
		MetricsCollector: metrics,
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}

	// Create test steps
	step1 := &mockSagaStep{
		id:   "step-1",
		name: "Step 1",
		executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
			return "step1-result", nil
		},
		retryable: true,
	}

	step2 := &mockSagaStep{
		id:   "step-2",
		name: "Step 2",
		executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
			// Verify data from previous step
			if data != "step1-result" {
				return nil, errors.New("unexpected data")
			}
			return "step2-result", nil
		},
		retryable: true,
	}

	step3 := &mockSagaStep{
		id:   "step-3",
		name: "Step 3",
		executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
			// Verify data from previous step
			if data != "step2-result" {
				return nil, errors.New("unexpected data")
			}
			return "final-result", nil
		},
		retryable: true,
	}

	definition := &mockSagaDefinition{
		id:          "test-saga",
		name:        "Test Saga",
		description: "Test saga for execution",
		steps:       []saga.SagaStep{step1, step2, step3},
		timeout:     30 * time.Second,
		retryPolicy: saga.NewFixedDelayRetryPolicy(3, 100*time.Millisecond),
	}

	// Create instance
	instance := &OrchestratorSagaInstance{
		id:             "test-instance-1",
		definitionID:   definition.id,
		state:          saga.StatePending,
		currentStep:    -1,
		completedSteps: 0,
		totalSteps:     3,
		createdAt:      time.Now(),
		updatedAt:      time.Now(),
		initialData:    "initial-data",
		currentData:    "initial-data",
		timeout:        30 * time.Second,
		retryPolicy:    definition.retryPolicy,
		metadata:       make(map[string]interface{}),
	}

	// Create executor
	executor := newStepExecutor(coordinator, instance, definition)

	// Execute steps
	ctx := context.Background()
	err = executor.executeSteps(ctx)

	// Verify no error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify instance state
	if instance.GetState() != saga.StateCompleted {
		t.Errorf("Expected state Completed, got: %s", instance.GetState())
	}

	// Verify completed steps
	if instance.GetCompletedSteps() != 3 {
		t.Errorf("Expected 3 completed steps, got: %d", instance.GetCompletedSteps())
	}

	// Verify result data
	if instance.GetResult() != "final-result" {
		t.Errorf("Expected result 'final-result', got: %v", instance.GetResult())
	}

	// Verify metrics
	if metrics.sagaCompletedCount != 1 {
		t.Errorf("Expected 1 saga completed, got: %d", metrics.sagaCompletedCount)
	}

	if metrics.stepExecutedCount != 3 {
		t.Errorf("Expected 3 steps executed, got: %d", metrics.stepExecutedCount)
	}
}

// TestStepExecutorExecuteStepsWithStepFailure tests step failure handling.
func TestStepExecutorExecuteStepsWithStepFailure(t *testing.T) {
	// Setup mocks
	storage := &mockStateStorage{}
	publisher := &mockEventPublisher{}
	metrics := &mockMetricsCollector{}

	coordinator, err := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:     storage,
		EventPublisher:   publisher,
		MetricsCollector: metrics,
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}

	// Create test steps
	step1 := &mockSagaStep{
		id:   "step-1",
		name: "Step 1",
		executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
			return "step1-result", nil
		},
		retryable: true,
	}

	expectedErr := errors.New("step 2 failed")
	step2 := &mockSagaStep{
		id:   "step-2",
		name: "Step 2",
		executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
			return nil, expectedErr
		},
		retryable: false,
	}

	definition := &mockSagaDefinition{
		id:          "test-saga",
		name:        "Test Saga",
		steps:       []saga.SagaStep{step1, step2},
		timeout:     30 * time.Second,
		retryPolicy: saga.NewFixedDelayRetryPolicy(1, 10*time.Millisecond),
	}

	instance := &OrchestratorSagaInstance{
		id:             "test-instance-2",
		definitionID:   definition.id,
		state:          saga.StatePending,
		currentStep:    -1,
		completedSteps: 0,
		totalSteps:     2,
		createdAt:      time.Now(),
		updatedAt:      time.Now(),
		initialData:    "initial-data",
		currentData:    "initial-data",
		timeout:        30 * time.Second,
		retryPolicy:    definition.retryPolicy,
		metadata:       make(map[string]interface{}),
	}

	// Create executor
	executor := newStepExecutor(coordinator, instance, definition)

	// Execute steps
	ctx := context.Background()
	err = executor.executeSteps(ctx)

	// Verify error
	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Verify instance state - should be Compensated after automatic compensation
	// Note: After implementing compensation logic in #509, step failures now trigger
	// automatic compensation, so the final state is Compensated instead of Failed
	expectedState := saga.StateCompensated
	if instance.GetState() != expectedState {
		t.Errorf("Expected state %s, got: %s", expectedState, instance.GetState())
	}

	// Verify completed steps (only step 1 should complete)
	if instance.GetCompletedSteps() != 1 {
		t.Errorf("Expected 1 completed step, got: %d", instance.GetCompletedSteps())
	}

	// Verify error is set
	if instance.GetError() == nil {
		t.Error("Expected saga error to be set")
	}

	// Verify metrics - saga failed count should still be 1 (failure recorded before compensation)
	if metrics.sagaFailedCount != 1 {
		t.Errorf("Expected 1 saga failed, got: %d", metrics.sagaFailedCount)
	}

	// Verify compensation was executed for step 1
	if metrics.compensationCount < 1 {
		t.Errorf("Expected at least 1 compensation, got: %d", metrics.compensationCount)
	}
}

// TestStepExecutorExecuteStepsWithRetry tests retry logic for failed steps.
func TestStepExecutorExecuteStepsWithRetry(t *testing.T) {
	// Setup mocks
	storage := &mockStateStorage{}
	publisher := &mockEventPublisher{}
	metrics := &mockMetricsCollector{}

	coordinator, err := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:     storage,
		EventPublisher:   publisher,
		MetricsCollector: metrics,
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}

	// Create test step that fails twice then succeeds
	attemptCount := 0
	step1 := &mockSagaStep{
		id:   "step-1",
		name: "Step 1",
		executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
			attemptCount++
			if attemptCount < 3 {
				return nil, errors.New("temporary failure")
			}
			return "success-after-retry", nil
		},
		retryable: true,
	}

	definition := &mockSagaDefinition{
		id:          "test-saga",
		name:        "Test Saga",
		steps:       []saga.SagaStep{step1},
		timeout:     30 * time.Second,
		retryPolicy: saga.NewFixedDelayRetryPolicy(3, 10*time.Millisecond),
	}

	instance := &OrchestratorSagaInstance{
		id:             "test-instance-3",
		definitionID:   definition.id,
		state:          saga.StatePending,
		currentStep:    -1,
		completedSteps: 0,
		totalSteps:     1,
		createdAt:      time.Now(),
		updatedAt:      time.Now(),
		initialData:    "initial-data",
		currentData:    "initial-data",
		timeout:        30 * time.Second,
		retryPolicy:    definition.retryPolicy,
		metadata:       make(map[string]interface{}),
	}

	// Create executor
	executor := newStepExecutor(coordinator, instance, definition)

	// Execute steps
	ctx := context.Background()
	err = executor.executeSteps(ctx)

	// Verify no error (should succeed after retries)
	if err != nil {
		t.Errorf("Expected no error after retries, got: %v", err)
	}

	// Verify instance state
	if instance.GetState() != saga.StateCompleted {
		t.Errorf("Expected state Completed, got: %s", instance.GetState())
	}

	// Verify attempt count
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got: %d", attemptCount)
	}

	// Verify retry metrics
	if metrics.stepRetriedCount != 2 {
		t.Errorf("Expected 2 retries, got: %d", metrics.stepRetriedCount)
	}
}

// TestStepExecutorExecuteStepsWithContextCancellation tests context cancellation handling.
func TestStepExecutorExecuteStepsWithContextCancellation(t *testing.T) {
	// Setup mocks
	storage := &mockStateStorage{}
	publisher := &mockEventPublisher{}
	metrics := &mockMetricsCollector{}

	coordinator, err := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:     storage,
		EventPublisher:   publisher,
		MetricsCollector: metrics,
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}

	// Create test steps
	step1 := &mockSagaStep{
		id:   "step-1",
		name: "Step 1",
		executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
			return "step1-result", nil
		},
		retryable: true,
	}

	step2 := &mockSagaStep{
		id:   "step-2",
		name: "Step 2",
		executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
			// This should not be called due to cancellation
			return "step2-result", nil
		},
		retryable: true,
	}

	definition := &mockSagaDefinition{
		id:          "test-saga",
		name:        "Test Saga",
		steps:       []saga.SagaStep{step1, step2},
		timeout:     30 * time.Second,
		retryPolicy: saga.NewFixedDelayRetryPolicy(1, 10*time.Millisecond),
	}

	instance := &OrchestratorSagaInstance{
		id:             "test-instance-4",
		definitionID:   definition.id,
		state:          saga.StatePending,
		currentStep:    -1,
		completedSteps: 0,
		totalSteps:     2,
		createdAt:      time.Now(),
		updatedAt:      time.Now(),
		initialData:    "initial-data",
		currentData:    "initial-data",
		timeout:        30 * time.Second,
		retryPolicy:    definition.retryPolicy,
		metadata:       make(map[string]interface{}),
	}

	// Create executor
	executor := newStepExecutor(coordinator, instance, definition)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Execute steps
	err = executor.executeSteps(ctx)

	// Verify error
	if err == nil {
		t.Error("Expected error due to cancellation, got nil")
	}

	// Verify it's a context error
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// TestStepExecutorDataPassing tests data passing between steps.
func TestStepExecutorDataPassing(t *testing.T) {
	// Setup mocks
	storage := &mockStateStorage{}
	publisher := &mockEventPublisher{}
	metrics := &mockMetricsCollector{}

	coordinator, err := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:     storage,
		EventPublisher:   publisher,
		MetricsCollector: metrics,
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}

	// Track data flow through steps
	dataFlow := []interface{}{}

	// Create test steps
	step1 := &mockSagaStep{
		id:   "step-1",
		name: "Step 1",
		executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
			dataFlow = append(dataFlow, data)
			return map[string]interface{}{"step": 1, "value": 100}, nil
		},
		retryable: true,
	}

	step2 := &mockSagaStep{
		id:   "step-2",
		name: "Step 2",
		executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
			dataFlow = append(dataFlow, data)
			dataMap := data.(map[string]interface{})
			dataMap["step"] = 2
			dataMap["value"] = dataMap["value"].(int) + 50
			return dataMap, nil
		},
		retryable: true,
	}

	step3 := &mockSagaStep{
		id:   "step-3",
		name: "Step 3",
		executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
			dataFlow = append(dataFlow, data)
			dataMap := data.(map[string]interface{})
			return map[string]interface{}{
				"final_step":  3,
				"final_value": dataMap["value"].(int) * 2,
			}, nil
		},
		retryable: true,
	}

	definition := &mockSagaDefinition{
		id:          "test-saga",
		name:        "Test Saga",
		steps:       []saga.SagaStep{step1, step2, step3},
		timeout:     30 * time.Second,
		retryPolicy: saga.NewFixedDelayRetryPolicy(1, 10*time.Millisecond),
	}

	instance := &OrchestratorSagaInstance{
		id:             "test-instance-5",
		definitionID:   definition.id,
		state:          saga.StatePending,
		currentStep:    -1,
		completedSteps: 0,
		totalSteps:     3,
		createdAt:      time.Now(),
		updatedAt:      time.Now(),
		initialData:    "initial-data",
		currentData:    "initial-data",
		timeout:        30 * time.Second,
		retryPolicy:    definition.retryPolicy,
		metadata:       make(map[string]interface{}),
	}

	// Create executor
	executor := newStepExecutor(coordinator, instance, definition)

	// Execute steps
	ctx := context.Background()
	err = executor.executeSteps(ctx)

	// Verify no error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify data flow
	if len(dataFlow) != 3 {
		t.Errorf("Expected 3 data items, got: %d", len(dataFlow))
	}

	// Verify initial data in step 1
	if dataFlow[0] != "initial-data" {
		t.Errorf("Expected initial data in step 1, got: %v", dataFlow[0])
	}

	// Verify step 1 output in step 2
	step1Data, ok := dataFlow[1].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map in step 2 input, got: %T", dataFlow[1])
	}
	// Note: step2 modifies the data in-place, so we verify it received the right initial values
	// by checking that it was able to produce the correct output
	if step1Data["value"] != 150 { // After step 2 modification
		t.Errorf("Unexpected value in step 2 output: %v", step1Data["value"])
	}

	// Verify final result
	finalResult := instance.GetResult().(map[string]interface{})
	if finalResult["final_step"] != 3 {
		t.Errorf("Expected final_step 3, got: %v", finalResult["final_step"])
	}
	if finalResult["final_value"] != 300 { // (100 + 50) * 2
		t.Errorf("Expected final_value 300, got: %v", finalResult["final_value"])
	}
}

// TestStepExecutorNoSteps tests handling of empty step list.
func TestStepExecutorNoSteps(t *testing.T) {
	// Setup mocks
	storage := &mockStateStorage{}
	publisher := &mockEventPublisher{}
	metrics := &mockMetricsCollector{}

	coordinator, err := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:     storage,
		EventPublisher:   publisher,
		MetricsCollector: metrics,
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}

	definition := &mockSagaDefinition{
		id:          "test-saga",
		name:        "Test Saga",
		steps:       []saga.SagaStep{}, // Empty steps
		timeout:     30 * time.Second,
		retryPolicy: saga.NewFixedDelayRetryPolicy(1, 10*time.Millisecond),
	}

	instance := &OrchestratorSagaInstance{
		id:             "test-instance-6",
		definitionID:   definition.id,
		state:          saga.StatePending,
		currentStep:    -1,
		completedSteps: 0,
		totalSteps:     0,
		createdAt:      time.Now(),
		updatedAt:      time.Now(),
		initialData:    "initial-data",
		currentData:    "initial-data",
		timeout:        30 * time.Second,
		retryPolicy:    definition.retryPolicy,
		metadata:       make(map[string]interface{}),
	}

	// Create executor
	executor := newStepExecutor(coordinator, instance, definition)

	// Execute steps
	ctx := context.Background()
	err = executor.executeSteps(ctx)

	// Verify error
	if err == nil {
		t.Error("Expected error for empty steps, got nil")
	}
}

// TestStepExecutorStepTimeout tests step timeout handling.
func TestStepExecutorStepTimeout(t *testing.T) {
	// Setup mocks
	storage := &mockStateStorage{}
	publisher := &mockEventPublisher{}
	metrics := &mockMetricsCollector{}

	coordinator, err := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:     storage,
		EventPublisher:   publisher,
		MetricsCollector: metrics,
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}

	// Create test step that takes too long
	step1 := &mockSagaStep{
		id:   "step-1",
		name: "Step 1",
		executeFunc: func(ctx context.Context, data interface{}) (interface{}, error) {
			// Wait for context to timeout
			<-ctx.Done()
			return nil, ctx.Err()
		},
		timeout:   50 * time.Millisecond, // Very short timeout
		retryable: false,
	}

	definition := &mockSagaDefinition{
		id:          "test-saga",
		name:        "Test Saga",
		steps:       []saga.SagaStep{step1},
		timeout:     30 * time.Second,
		retryPolicy: saga.NewFixedDelayRetryPolicy(1, 10*time.Millisecond),
	}

	instance := &OrchestratorSagaInstance{
		id:             "test-instance-7",
		definitionID:   definition.id,
		state:          saga.StatePending,
		currentStep:    -1,
		completedSteps: 0,
		totalSteps:     1,
		createdAt:      time.Now(),
		updatedAt:      time.Now(),
		initialData:    "initial-data",
		currentData:    "initial-data",
		timeout:        30 * time.Second,
		retryPolicy:    definition.retryPolicy,
		metadata:       make(map[string]interface{}),
	}

	// Create executor
	executor := newStepExecutor(coordinator, instance, definition)

	// Execute steps
	ctx := context.Background()
	err = executor.executeSteps(ctx)

	// Verify error (should be timeout error)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Verify instance state
	if instance.GetState() != saga.StateFailed {
		t.Errorf("Expected state Failed, got: %s", instance.GetState())
	}

	// Verify error type
	if instance.GetError() == nil {
		t.Error("Expected saga error to be set")
	} else if instance.GetError().Type != saga.ErrorTypeTimeout {
		t.Errorf("Expected timeout error type, got: %s", instance.GetError().Type)
	}
}

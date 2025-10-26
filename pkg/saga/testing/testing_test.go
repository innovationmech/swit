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

package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestMockStateStorage tests the MockStateStorage implementation.
func TestMockStateStorage(t *testing.T) {
	storage := NewMockStateStorage()
	ctx := context.Background()

	// Test SaveSaga and GetSaga with default behavior
	mockInstance := &mockSagaInstance{
		id:        "saga-1",
		defID:     "def-1",
		state:     saga.StatePending,
		startTime: time.Now(),
		timeout:   5 * time.Minute,
		metadata:  make(map[string]interface{}),
	}

	err := storage.SaveSaga(ctx, mockInstance)
	if err != nil {
		t.Errorf("SaveSaga failed: %v", err)
	}

	if storage.SaveSagaCalls != 1 {
		t.Errorf("Expected 1 SaveSaga call, got %d", storage.SaveSagaCalls)
	}

	retrieved, err := storage.GetSaga(ctx, "saga-1")
	if err != nil {
		t.Errorf("GetSaga failed: %v", err)
	}

	if retrieved.GetID() != "saga-1" {
		t.Errorf("Expected saga ID 'saga-1', got '%s'", retrieved.GetID())
	}

	// Test custom behavior
	storage.GetSagaFunc = func(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
		return nil, fmt.Errorf("custom error")
	}

	_, err = storage.GetSaga(ctx, "saga-1")
	if err == nil || err.Error() != "custom error" {
		t.Errorf("Expected custom error, got %v", err)
	}

	// Test Reset
	storage.Reset()
	if storage.SaveSagaCalls != 0 {
		t.Errorf("Expected 0 calls after reset, got %d", storage.SaveSagaCalls)
	}
}

// TestMockEventPublisher tests the MockEventPublisher implementation.
func TestMockEventPublisher(t *testing.T) {
	publisher := NewMockEventPublisher()
	ctx := context.Background()

	// Test PublishEvent
	event := &saga.SagaEvent{
		ID:        "event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}

	err := publisher.PublishEvent(ctx, event)
	if err != nil {
		t.Errorf("PublishEvent failed: %v", err)
	}

	if publisher.PublishEventCalls != 1 {
		t.Errorf("Expected 1 PublishEvent call, got %d", publisher.PublishEventCalls)
	}

	if len(publisher.PublishedEvents) != 1 {
		t.Errorf("Expected 1 published event, got %d", len(publisher.PublishedEvents))
	}

	// Test GetEventsByType
	events := publisher.GetEventsByType(saga.EventSagaStarted)
	if len(events) != 1 {
		t.Errorf("Expected 1 SagaStarted event, got %d", len(events))
	}

	// Test error simulation
	publisher.PublishEventError = fmt.Errorf("publish error")
	err = publisher.PublishEvent(ctx, event)
	if err == nil {
		t.Error("Expected publish error, got nil")
	}

	// Test Reset
	publisher.Reset()
	if len(publisher.PublishedEvents) != 0 {
		t.Errorf("Expected 0 events after reset, got %d", len(publisher.PublishedEvents))
	}
}

// TestMockSagaStep tests the MockSagaStep implementation.
func TestMockSagaStep(t *testing.T) {
	step := NewMockSagaStep("step-1", "Test Step")
	ctx := context.Background()

	// Test basic properties
	if step.GetID() != "step-1" {
		t.Errorf("Expected ID 'step-1', got '%s'", step.GetID())
	}

	if step.GetName() != "Test Step" {
		t.Errorf("Expected name 'Test Step', got '%s'", step.GetName())
	}

	// Test Execute with default behavior
	step.ExecuteResult = "success"
	result, err := step.Execute(ctx, "input")
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	if result != "success" {
		t.Errorf("Expected result 'success', got '%v'", result)
	}

	if step.ExecuteCalls != 1 {
		t.Errorf("Expected 1 Execute call, got %d", step.ExecuteCalls)
	}

	// Test Execute with custom function
	step.ExecuteFunc = func(ctx context.Context, data interface{}) (interface{}, error) {
		return "custom", nil
	}
	result, err = step.Execute(ctx, "input")
	if err != nil {
		t.Errorf("Execute with custom func failed: %v", err)
	}
	if result != "custom" {
		t.Errorf("Expected result 'custom', got '%v'", result)
	}

	// Test Compensate
	step.CompensateFunc = func(ctx context.Context, data interface{}) error {
		return nil
	}
	err = step.Compensate(ctx, "data")
	if err != nil {
		t.Errorf("Compensate failed: %v", err)
	}

	if step.CompensateCalls != 1 {
		t.Errorf("Expected 1 Compensate call, got %d", step.CompensateCalls)
	}
}

// TestSagaTestBuilder tests the SagaTestBuilder.
func TestSagaTestBuilder(t *testing.T) {
	saga := NewSagaTestBuilder("test-saga", "Test Saga").
		WithDescription("A test saga").
		WithTimeout(5*time.Minute).
		AddStep("step1", "Step 1").
		AddStep("step2", "Step 2").
		AddFailingStep("step3", "Failing Step", fmt.Errorf("expected error")).
		AddSlowStep("step4", "Slow Step", 100*time.Millisecond).
		WithMetadata("key", "value").
		Build()

	if saga.GetID() != "test-saga" {
		t.Errorf("Expected ID 'test-saga', got '%s'", saga.GetID())
	}

	if saga.GetName() != "Test Saga" {
		t.Errorf("Expected name 'Test Saga', got '%s'", saga.GetName())
	}

	steps := saga.GetSteps()
	if len(steps) != 4 {
		t.Errorf("Expected 4 steps, got %d", len(steps))
	}

	if saga.GetTimeout() != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", saga.GetTimeout())
	}

	metadata := saga.GetMetadata()
	if metadata["key"] != "value" {
		t.Errorf("Expected metadata key='value', got '%v'", metadata["key"])
	}
}

// TestScenarioBuilder tests the ScenarioBuilder.
func TestScenarioBuilder(t *testing.T) {
	sagaDef := NewMockSagaDefinition("saga-1", "Test Saga")
	storage := NewMockStateStorage()
	publisher := NewMockEventPublisher()

	scenario := NewScenarioBuilder("Test Scenario").
		WithDescription("A test scenario").
		WithSaga(sagaDef).
		WithStorage(storage).
		WithPublisher(publisher).
		WithInputData(map[string]interface{}{"key": "value"}).
		AddAssertion(func(result *TestResult) error {
			return nil
		}).
		Build()

	if scenario.Name != "Test Scenario" {
		t.Errorf("Expected name 'Test Scenario', got '%s'", scenario.Name)
	}

	if scenario.Saga != sagaDef {
		t.Error("Saga definition not set correctly")
	}

	if scenario.Storage != storage {
		t.Error("Storage not set correctly")
	}

	if scenario.Publisher != publisher {
		t.Error("Publisher not set correctly")
	}

	if len(scenario.Assertions) != 1 {
		t.Errorf("Expected 1 assertion, got %d", len(scenario.Assertions))
	}
}

// TestAssertions tests various assertion functions.
func TestAssertions(t *testing.T) {
	// Create a test result
	result := &TestResult{
		Success:  true,
		Duration: 100 * time.Millisecond,
		StepStates: []*saga.StepState{
			{
				ID:    "step-1",
				Name:  "Step 1",
				State: saga.StepStateCompleted,
			},
			{
				ID:    "step-2",
				Name:  "Step 2",
				State: saga.StepStateCompleted,
			},
		},
		Events: []*saga.SagaEvent{
			{Type: saga.EventSagaStarted},
			{Type: saga.EventSagaStepCompleted},
			{Type: saga.EventSagaStepCompleted},
			{Type: saga.EventSagaCompleted},
		},
		Storage:   NewMockStateStorage(),
		Publisher: NewMockEventPublisher(),
	}

	result.Storage.SaveSagaCalls = 2
	result.Publisher.PublishEventCalls = 4

	// Test step assertions
	t.Run("AssertStepCount", func(t *testing.T) {
		err := AssertStepCount(2)(result)
		if err != nil {
			t.Errorf("AssertStepCount failed: %v", err)
		}

		err = AssertStepCount(3)(result)
		if err == nil {
			t.Error("Expected AssertStepCount to fail, but it passed")
		}
	})

	t.Run("AssertAllStepsCompleted", func(t *testing.T) {
		err := AssertAllStepsCompleted()(result)
		if err != nil {
			t.Errorf("AssertAllStepsCompleted failed: %v", err)
		}
	})

	t.Run("AssertExecutionSucceeded", func(t *testing.T) {
		err := AssertExecutionSucceeded()(result)
		if err != nil {
			t.Errorf("AssertExecutionSucceeded failed: %v", err)
		}
	})

	t.Run("AssertDurationLessThan", func(t *testing.T) {
		err := AssertDurationLessThan(200 * time.Millisecond)(result)
		if err != nil {
			t.Errorf("AssertDurationLessThan failed: %v", err)
		}
	})

	t.Run("AssertEventPublished", func(t *testing.T) {
		err := AssertEventPublished(saga.EventSagaStarted)(result)
		if err != nil {
			t.Errorf("AssertEventPublished failed: %v", err)
		}
	})

	t.Run("AssertEventCount", func(t *testing.T) {
		err := AssertEventCount(4)(result)
		if err != nil {
			t.Errorf("AssertEventCount failed: %v", err)
		}
	})

	t.Run("AssertStorageSaveCount", func(t *testing.T) {
		err := AssertStorageSaveCount(2)(result)
		if err != nil {
			t.Errorf("AssertStorageSaveCount failed: %v", err)
		}
	})

	t.Run("AssertPublisherCalls", func(t *testing.T) {
		err := AssertPublisherCalls(4)(result)
		if err != nil {
			t.Errorf("AssertPublisherCalls failed: %v", err)
		}
	})

	// Test composite assertions
	t.Run("And", func(t *testing.T) {
		err := And(
			AssertStepCount(2),
			AssertAllStepsCompleted(),
			AssertExecutionSucceeded(),
		)(result)
		if err != nil {
			t.Errorf("And assertion failed: %v", err)
		}
	})
}

// TestTestConfig tests the test configuration.
func TestTestConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultTestConfig()
		if err := config.Validate(); err != nil {
			t.Errorf("Default config validation failed: %v", err)
		}
	})

	t.Run("QuickTestConfig", func(t *testing.T) {
		config := QuickTestConfig()
		if config.DefaultTimeout != 10*time.Second {
			t.Errorf("Expected default timeout 10s, got %v", config.DefaultTimeout)
		}
	})

	t.Run("IntegrationTestConfig", func(t *testing.T) {
		config := IntegrationTestConfig()
		if config.DefaultTimeout != 10*time.Minute {
			t.Errorf("Expected default timeout 10m, got %v", config.DefaultTimeout)
		}
	})

	t.Run("ConfigBuilder", func(t *testing.T) {
		config := NewTestConfigBuilder().
			WithTimeout(1*time.Minute).
			WithStepTimeout(10*time.Second).
			WithRetry(5, 100*time.Millisecond).
			WithConcurrency(20).
			WithVerboseLogging().
			Build()

		if config.DefaultTimeout != 1*time.Minute {
			t.Errorf("Expected timeout 1m, got %v", config.DefaultTimeout)
		}

		if config.MaxRetries != 5 {
			t.Errorf("Expected max retries 5, got %d", config.MaxRetries)
		}

		if !config.EnableVerboseLogging {
			t.Error("Expected verbose logging to be enabled")
		}
	})
}

// TestTestLogger tests the test logger.
func TestTestLogger(t *testing.T) {
	logger := NewTestLogger()
	logger.SetLevel(LogLevelDebug) // Set level to capture debug messages

	// Test logging
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")
	logger.Debugf("Debug message %d", 123)

	// Test entry counts
	if logger.GetEntriesCount() != 4 {
		t.Errorf("Expected 4 entries, got %d", logger.GetEntriesCount())
	}

	// Test entries by level
	errors := logger.GetEntriesByLevel(LogLevelError)
	if len(errors) != 1 {
		t.Errorf("Expected 1 error entry, got %d", len(errors))
	}

	// Test HasErrors
	if !logger.HasErrors() {
		t.Error("Expected HasErrors to return true")
	}

	// Test Clear
	logger.Clear()
	if logger.GetEntriesCount() != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", logger.GetEntriesCount())
	}
}

// TestBufferedLogger tests the buffered logger.
func TestBufferedLogger(t *testing.T) {
	logger := NewBufferedLogger()

	logger.Info("Test message")
	logger.Error("Error message")

	output := logger.GetOutput()
	if !contains(output, "Test message") {
		t.Error("Output does not contain 'Test message'")
	}

	if !contains(output, "Error message") {
		t.Error("Output does not contain 'Error message'")
	}

	logger.ClearBuffer()
	if logger.GetOutput() != "" {
		t.Error("Expected empty output after clear")
	}
}

// mockSagaInstance is a simple mock for testing
type mockSagaInstance struct {
	id        string
	defID     string
	state     saga.SagaState
	startTime time.Time
	endTime   time.Time
	timeout   time.Duration
	metadata  map[string]interface{}
}

func (m *mockSagaInstance) GetID() string {
	return m.id
}

func (m *mockSagaInstance) GetDefinitionID() string {
	return m.defID
}

func (m *mockSagaInstance) GetState() saga.SagaState {
	return m.state
}

func (m *mockSagaInstance) GetCurrentStep() int {
	return 0
}

func (m *mockSagaInstance) GetStartTime() time.Time {
	return m.startTime
}

func (m *mockSagaInstance) GetEndTime() time.Time {
	return m.endTime
}

func (m *mockSagaInstance) GetResult() interface{} {
	return nil
}

func (m *mockSagaInstance) GetError() *saga.SagaError {
	return nil
}

func (m *mockSagaInstance) GetTotalSteps() int {
	return 0
}

func (m *mockSagaInstance) GetCompletedSteps() int {
	return 0
}

func (m *mockSagaInstance) GetCreatedAt() time.Time {
	return m.startTime
}

func (m *mockSagaInstance) GetUpdatedAt() time.Time {
	return m.startTime
}

func (m *mockSagaInstance) GetTimeout() time.Duration {
	return m.timeout
}

func (m *mockSagaInstance) GetMetadata() map[string]interface{} {
	return m.metadata
}

func (m *mockSagaInstance) GetTraceID() string {
	return ""
}

func (m *mockSagaInstance) IsTerminal() bool {
	return false
}

func (m *mockSagaInstance) IsActive() bool {
	return true
}

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

package chaos

import (
	"context"
	"errors"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/coordinator"
	"github.com/innovationmech/swit/pkg/saga/state/storage"
)

// createTestCoordinator creates a coordinator with chaos state storage.
func createTestCoordinator(chaosStorage saga.StateStorage, retryPolicy saga.RetryPolicy) (*coordinator.OrchestratorCoordinator, error) {
	mockPublisher := &mockEventPublisher{}
	
	config := &coordinator.OrchestratorConfig{
		StateStorage:   chaosStorage,
		EventPublisher: mockPublisher,
		RetryPolicy:    retryPolicy,
	}
	
	return coordinator.NewOrchestratorCoordinator(config)
}

// createTestSagaDefinition creates a simple test Saga definition.
func createTestSagaDefinition(injector *FaultInjector) saga.SagaDefinition {
	return &mockSagaDefinition{
		id:   "test-saga",
		name: "Test Saga",
		steps: []saga.SagaStep{
			injector.WrapStep(&mockStep{id: "step1", name: "Step 1"}),
			injector.WrapStep(&mockStep{id: "step2", name: "Step 2"}),
		},
		timeout:     5 * time.Second,
		retryPolicy: saga.NewFixedDelayRetryPolicy(2, 100*time.Millisecond),
	}
}

// createMultiStepSagaDefinition creates a Saga definition with multiple steps.
func createMultiStepSagaDefinition(injector *FaultInjector) saga.SagaDefinition {
	return &mockSagaDefinition{
		id:   "multi-step-saga",
		name: "Multi-Step Test Saga",
		steps: []saga.SagaStep{
			injector.WrapStep(&mockStep{id: "step1", name: "Step 1", timeout: 1 * time.Second}),
			injector.WrapStep(&mockStep{id: "step2", name: "Step 2", timeout: 1 * time.Second}),
			injector.WrapStep(&mockStep{id: "step3", name: "Step 3", timeout: 1 * time.Second}),
		},
		timeout:     10 * time.Second,
		retryPolicy: saga.NewFixedDelayRetryPolicy(2, 100*time.Millisecond),
	}
}

// createFailingSagaDefinition creates a Saga definition that will fail to trigger compensation.
func createFailingSagaDefinition(injector *FaultInjector) saga.SagaDefinition {
	return &mockSagaDefinition{
		id:   "failing-saga",
		name: "Failing Test Saga",
		steps: []saga.SagaStep{
			injector.WrapStep(&mockStep{id: "step1", name: "Step 1", timeout: 1 * time.Second}),
			injector.WrapStep(&mockStep{id: "step2", name: "Step 2", timeout: 1 * time.Second}),
			injector.WrapStep(&mockStep{id: "step3", name: "Step 3 (Failing)", timeout: 1 * time.Second, shouldFail: true}),
		},
		timeout:     10 * time.Second,
		retryPolicy: saga.NewNoRetryPolicy(),
	}
}

// createMultiStepFailingSagaDefinition creates a multi-step Saga that fails after several steps.
func createMultiStepFailingSagaDefinition(injector *FaultInjector) saga.SagaDefinition {
	return &mockSagaDefinition{
		id:   "multi-step-failing-saga",
		name: "Multi-Step Failing Test Saga",
		steps: []saga.SagaStep{
			injector.WrapStep(&mockStep{id: "step1", name: "Step 1", timeout: 1 * time.Second}),
			injector.WrapStep(&mockStep{id: "step2", name: "Step 2", timeout: 1 * time.Second}),
			injector.WrapStep(&mockStep{id: "step3", name: "Step 3", timeout: 1 * time.Second}),
			injector.WrapStep(&mockStep{id: "step4", name: "Step 4", timeout: 1 * time.Second}),
			injector.WrapStep(&mockStep{id: "step5", name: "Step 5 (Failing)", timeout: 1 * time.Second, shouldFail: true}),
		},
		timeout:     10 * time.Second,
		retryPolicy: saga.NewNoRetryPolicy(),
	}
}

// Mock implementations for testing

// mockSagaInstance implements saga.SagaInstance for testing.
type mockSagaInstance struct {
	id    string
	state saga.SagaState
}

func (m *mockSagaInstance) GetID() string                            { return m.id }
func (m *mockSagaInstance) GetDefinitionID() string                  { return "test-def" }
func (m *mockSagaInstance) GetState() saga.SagaState                 { return m.state }
func (m *mockSagaInstance) GetCurrentStep() int                      { return 0 }
func (m *mockSagaInstance) GetStartTime() time.Time                  { return time.Now() }
func (m *mockSagaInstance) GetEndTime() time.Time                    { return time.Time{} }
func (m *mockSagaInstance) GetResult() interface{}                   { return nil }
func (m *mockSagaInstance) GetError() *saga.SagaError                { return nil }
func (m *mockSagaInstance) GetTotalSteps() int                       { return 2 }
func (m *mockSagaInstance) GetCompletedSteps() int                   { return 0 }
func (m *mockSagaInstance) GetCreatedAt() time.Time                  { return time.Now() }
func (m *mockSagaInstance) GetUpdatedAt() time.Time                  { return time.Now() }
func (m *mockSagaInstance) GetTimeout() time.Duration                { return 5 * time.Second }
func (m *mockSagaInstance) GetMetadata() map[string]interface{}      { return nil }
func (m *mockSagaInstance) GetTraceID() string                       { return "" }
func (m *mockSagaInstance) IsTerminal() bool                         { return m.state.IsTerminal() }
func (m *mockSagaInstance) IsActive() bool                           { return m.state.IsActive() }

// mockSagaDefinition implements saga.SagaDefinition for testing.
type mockSagaDefinition struct {
	id          string
	name        string
	description string
	steps       []saga.SagaStep
	timeout     time.Duration
	retryPolicy saga.RetryPolicy
}

func (m *mockSagaDefinition) GetID() string                                     { return m.id }
func (m *mockSagaDefinition) GetName() string                                   { return m.name }
func (m *mockSagaDefinition) GetDescription() string                            { return m.description }
func (m *mockSagaDefinition) GetSteps() []saga.SagaStep                         { return m.steps }
func (m *mockSagaDefinition) GetTimeout() time.Duration                         { return m.timeout }
func (m *mockSagaDefinition) GetRetryPolicy() saga.RetryPolicy                  { return m.retryPolicy }
func (m *mockSagaDefinition) GetCompensationStrategy() saga.CompensationStrategy { return nil }
func (m *mockSagaDefinition) Validate() error                                   { return nil }
func (m *mockSagaDefinition) GetMetadata() map[string]interface{}               { return nil }

// mockStep implements saga.SagaStep for testing.
type mockStep struct {
	id          string
	name        string
	description string
	timeout     time.Duration
	shouldFail  bool
}

func (m *mockStep) GetID() string                          { return m.id }
func (m *mockStep) GetName() string                        { return m.name }
func (m *mockStep) GetDescription() string                 { return m.description }
func (m *mockStep) GetTimeout() time.Duration              { return m.timeout }
func (m *mockStep) GetRetryPolicy() saga.RetryPolicy       { return nil }
func (m *mockStep) IsRetryable(err error) bool             { return true }
func (m *mockStep) GetMetadata() map[string]interface{}    { return nil }

func (m *mockStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	if m.shouldFail {
		return nil, errors.New("step execution failed")
	}
	return data, nil
}

func (m *mockStep) Compensate(ctx context.Context, data interface{}) error {
	return nil
}

// mockEventPublisher implements saga.EventPublisher for testing.
type mockEventPublisher struct {
	publishCount int
}

func (m *mockEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	m.publishCount++
	return nil
}

func (m *mockEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	return nil, nil
}

func (m *mockEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	return nil
}

func (m *mockEventPublisher) Close() error {
	return nil
}

// createTestStorage creates a test memory storage wrapped with chaos.
func createTestStorage(injector *FaultInjector) saga.StateStorage {
	memStorage := storage.NewMemoryStateStorage()
	return NewChaosStateStorage(memStorage, injector)
}


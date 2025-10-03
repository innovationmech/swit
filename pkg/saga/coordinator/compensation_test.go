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
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// mockStateStorageWithSteps extends mockStateStorage to store step states
type mockStateStorageWithSteps struct {
	saveSagaErr            error
	getSagaErr             error
	updateSagaStateErr     error
	deleteSagaErr          error
	getActiveSagasErr      error
	getTimeoutSagasErr     error
	saveStepStateErr       error
	getStepStatesErr       error
	cleanupExpiredSagasErr error
	stepStates             map[string][]*saga.StepState // sagaID -> step states
	mu                     sync.RWMutex
}

func newMockStateStorage() *mockStateStorageWithSteps {
	return &mockStateStorageWithSteps{
		stepStates: make(map[string][]*saga.StepState),
	}
}

func (m *mockStateStorageWithSteps) SaveSaga(ctx context.Context, saga saga.SagaInstance) error {
	return m.saveSagaErr
}

func (m *mockStateStorageWithSteps) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	return nil, m.getSagaErr
}

func (m *mockStateStorageWithSteps) UpdateSagaState(ctx context.Context, sagaID string, state saga.SagaState, metadata map[string]interface{}) error {
	return m.updateSagaStateErr
}

func (m *mockStateStorageWithSteps) DeleteSaga(ctx context.Context, sagaID string) error {
	return m.deleteSagaErr
}

func (m *mockStateStorageWithSteps) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	if m.getActiveSagasErr != nil {
		return nil, m.getActiveSagasErr
	}
	return []saga.SagaInstance{}, nil
}

func (m *mockStateStorageWithSteps) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	return nil, m.getTimeoutSagasErr
}

func (m *mockStateStorageWithSteps) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	if m.saveStepStateErr != nil {
		return m.saveStepStateErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stepStates[sagaID] == nil {
		m.stepStates[sagaID] = []*saga.StepState{}
	}

	// Find and update existing step or append new one
	found := false
	for i, s := range m.stepStates[sagaID] {
		if s.StepIndex == step.StepIndex {
			m.stepStates[sagaID][i] = step
			found = true
			break
		}
	}

	if !found {
		m.stepStates[sagaID] = append(m.stepStates[sagaID], step)
	}

	return nil
}

func (m *mockStateStorageWithSteps) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	if m.getStepStatesErr != nil {
		return nil, m.getStepStatesErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	states := m.stepStates[sagaID]
	if states == nil {
		return []*saga.StepState{}, nil
	}

	return states, nil
}

func (m *mockStateStorageWithSteps) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	return m.cleanupExpiredSagasErr
}

// mockEventPublisherWithTracking extends mockEventPublisher to track published events
type mockEventPublisherWithTracking struct {
	publishEventErr error
	subscribeErr    error
	unsubscribeErr  error
	closeErr        error
	events          []*saga.SagaEvent
	mu              sync.RWMutex
}

func newMockEventPublisher() *mockEventPublisherWithTracking {
	return &mockEventPublisherWithTracking{
		events: []*saga.SagaEvent{},
	}
}

func (m *mockEventPublisherWithTracking) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	if m.publishEventErr != nil {
		return m.publishEventErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = append(m.events, event)
	return nil
}

func (m *mockEventPublisherWithTracking) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	return nil, m.subscribeErr
}

func (m *mockEventPublisherWithTracking) Unsubscribe(subscription saga.EventSubscription) error {
	return m.unsubscribeErr
}

func (m *mockEventPublisherWithTracking) Close() error {
	return m.closeErr
}

func (m *mockEventPublisherWithTracking) GetEvents() []*saga.SagaEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	events := make([]*saga.SagaEvent, len(m.events))
	copy(events, m.events)
	return events
}

// mockCompensationStep implements saga.SagaStep with compensation tracking
type mockCompensationStep struct {
	id                   string
	name                 string
	executeFunc          func(ctx context.Context, data interface{}) (interface{}, error)
	compensateFunc       func(ctx context.Context, data interface{}) error
	compensationCalled   bool
	compensationAttempts int
	mu                   sync.Mutex
}

func (m *mockCompensationStep) GetID() string                       { return m.id }
func (m *mockCompensationStep) GetName() string                     { return m.name }
func (m *mockCompensationStep) GetDescription() string              { return "mock step" }
func (m *mockCompensationStep) GetTimeout() time.Duration           { return 5 * time.Second }
func (m *mockCompensationStep) GetRetryPolicy() saga.RetryPolicy    { return nil }
func (m *mockCompensationStep) IsRetryable(err error) bool          { return true }
func (m *mockCompensationStep) GetMetadata() map[string]interface{} { return nil }

func (m *mockCompensationStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, data)
	}
	return data, nil
}

func (m *mockCompensationStep) Compensate(ctx context.Context, data interface{}) error {
	m.mu.Lock()
	m.compensationCalled = true
	m.compensationAttempts++
	m.mu.Unlock()

	if m.compensateFunc != nil {
		return m.compensateFunc(ctx, data)
	}
	return nil
}

func (m *mockCompensationStep) WasCompensationCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.compensationCalled
}

func (m *mockCompensationStep) GetCompensationAttempts() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.compensationAttempts
}

// mockCompensationDefinition implements saga.SagaDefinition with compensation strategy
type mockCompensationDefinition struct {
	id                   string
	name                 string
	steps                []saga.SagaStep
	compensationStrategy saga.CompensationStrategy
}

func (m *mockCompensationDefinition) GetID() string                    { return m.id }
func (m *mockCompensationDefinition) GetName() string                  { return m.name }
func (m *mockCompensationDefinition) GetDescription() string           { return "mock definition" }
func (m *mockCompensationDefinition) GetSteps() []saga.SagaStep        { return m.steps }
func (m *mockCompensationDefinition) GetTimeout() time.Duration        { return 30 * time.Second }
func (m *mockCompensationDefinition) GetRetryPolicy() saga.RetryPolicy { return nil }
func (m *mockCompensationDefinition) GetCompensationStrategy() saga.CompensationStrategy {
	return m.compensationStrategy
}
func (m *mockCompensationDefinition) Validate() error                     { return nil }
func (m *mockCompensationDefinition) GetMetadata() map[string]interface{} { return nil }

// TestCompensationExecutor_ExecuteCompensation_Success tests successful compensation in reverse order
func TestCompensationExecutor_ExecuteCompensation_Success(t *testing.T) {
	storage := newMockStateStorage()
	publisher := newMockEventPublisher()
	collector := &noOpMetricsCollector{}

	coordinator := &OrchestratorCoordinator{
		stateStorage:     storage,
		eventPublisher:   publisher,
		retryPolicy:      saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
		metricsCollector: collector,
		metrics:          &saga.CoordinatorMetrics{},
	}

	// Create mock steps
	step1 := &mockCompensationStep{id: "step1", name: "Step 1"}
	step2 := &mockCompensationStep{id: "step2", name: "Step 2"}
	step3 := &mockCompensationStep{id: "step3", name: "Step 3"}

	completedSteps := []saga.SagaStep{step1, step2, step3}

	definition := &mockCompensationDefinition{
		id:                   "test-saga",
		name:                 "Test Saga",
		steps:                completedSteps,
		compensationStrategy: saga.NewSequentialCompensationStrategy(30 * time.Second),
	}

	now := time.Now()
	instance := &OrchestratorSagaInstance{
		id:           "saga-123",
		definitionID: "test-saga",
		state:        saga.StateFailed,
		startedAt:    &now,
		sagaError: &saga.SagaError{
			Code:    "TEST_ERROR",
			Message: "test error",
			Type:    saga.ErrorTypeService,
		},
	}

	// Store step states
	for i, step := range completedSteps {
		stepState := &saga.StepState{
			ID:         step.GetID(),
			SagaID:     instance.id,
			StepIndex:  i,
			Name:       step.GetName(),
			State:      saga.StepStateCompleted,
			OutputData: map[string]interface{}{"result": i + 1},
		}
		_ = storage.SaveStepState(context.Background(), instance.id, stepState)
	}

	executor := newCompensationExecutor(coordinator, instance, definition)

	// Execute compensation
	err := executor.executeCompensation(context.Background(), completedSteps)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify all steps were compensated in reverse order
	if !step1.WasCompensationCalled() {
		t.Error("Step 1 compensation was not called")
	}
	if !step2.WasCompensationCalled() {
		t.Error("Step 2 compensation was not called")
	}
	if !step3.WasCompensationCalled() {
		t.Error("Step 3 compensation was not called")
	}

	// Verify final state is Compensated
	if instance.state != saga.StateCompensated {
		t.Errorf("Expected state Compensated, got: %v", instance.state)
	}

	// Verify events were published
	events := publisher.GetEvents()
	if len(events) == 0 {
		t.Error("Expected events to be published")
	}

	// Check for compensation events
	hasCompensationStarted := false
	hasCompensationCompleted := false
	for _, event := range events {
		if event.Type == saga.EventCompensationStarted {
			hasCompensationStarted = true
		}
		if event.Type == saga.EventCompensationCompleted {
			hasCompensationCompleted = true
		}
	}

	if !hasCompensationStarted {
		t.Error("Expected compensation started event")
	}
	if !hasCompensationCompleted {
		t.Error("Expected compensation completed event")
	}
}

// TestCompensationExecutor_CompensationFailure tests compensation failure handling
func TestCompensationExecutor_CompensationFailure(t *testing.T) {
	storage := newMockStateStorage()
	publisher := newMockEventPublisher()
	collector := &noOpMetricsCollector{}

	coordinator := &OrchestratorCoordinator{
		stateStorage:     storage,
		eventPublisher:   publisher,
		retryPolicy:      saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
		metricsCollector: collector,
		metrics:          &saga.CoordinatorMetrics{},
	}

	// Create mock steps - step2 will fail compensation
	step1 := &mockCompensationStep{id: "step1", name: "Step 1"}
	step2 := &mockCompensationStep{
		id:   "step2",
		name: "Step 2",
		compensateFunc: func(ctx context.Context, data interface{}) error {
			return errors.New("compensation failed")
		},
	}
	step3 := &mockCompensationStep{id: "step3", name: "Step 3"}

	completedSteps := []saga.SagaStep{step1, step2, step3}

	definition := &mockCompensationDefinition{
		id:                   "test-saga",
		name:                 "Test Saga",
		steps:                completedSteps,
		compensationStrategy: saga.NewSequentialCompensationStrategy(30 * time.Second),
	}

	now := time.Now()
	instance := &OrchestratorSagaInstance{
		id:           "saga-456",
		definitionID: "test-saga",
		state:        saga.StateFailed,
		startedAt:    &now,
		sagaError: &saga.SagaError{
			Code:    "TEST_ERROR",
			Message: "test error",
			Type:    saga.ErrorTypeService,
		},
	}

	// Store step states
	for i, step := range completedSteps {
		stepState := &saga.StepState{
			ID:        step.GetID(),
			SagaID:    instance.id,
			StepIndex: i,
			Name:      step.GetName(),
			State:     saga.StepStateCompleted,
		}
		_ = storage.SaveStepState(context.Background(), instance.id, stepState)
	}

	executor := newCompensationExecutor(coordinator, instance, definition)

	// Execute compensation - should continue despite failures
	err := executor.executeCompensation(context.Background(), completedSteps)
	if err == nil {
		t.Fatal("Expected error due to compensation failure")
	}

	// Verify step2 attempted compensation (with retries)
	if step2.GetCompensationAttempts() != 3 {
		t.Errorf("Expected 3 compensation attempts for step2, got: %d", step2.GetCompensationAttempts())
	}

	// Verify final state is Failed
	if instance.state != saga.StateFailed {
		t.Errorf("Expected state Failed, got: %v", instance.state)
	}

	// Verify compensation failed event was published
	events := publisher.GetEvents()
	hasCompensationFailed := false
	for _, event := range events {
		if event.Type == saga.EventCompensationFailed {
			hasCompensationFailed = true
		}
	}

	if !hasCompensationFailed {
		t.Error("Expected compensation failed event")
	}
}

// TestCompensationExecutor_NoCompletedSteps tests compensation with no completed steps
func TestCompensationExecutor_NoCompletedSteps(t *testing.T) {
	storage := newMockStateStorage()
	publisher := newMockEventPublisher()
	collector := &noOpMetricsCollector{}

	coordinator := &OrchestratorCoordinator{
		stateStorage:     storage,
		eventPublisher:   publisher,
		retryPolicy:      saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
		metricsCollector: collector,
		metrics:          &saga.CoordinatorMetrics{},
	}

	definition := &mockCompensationDefinition{
		id:                   "test-saga",
		name:                 "Test Saga",
		steps:                []saga.SagaStep{},
		compensationStrategy: saga.NewSequentialCompensationStrategy(30 * time.Second),
	}

	now := time.Now()
	instance := &OrchestratorSagaInstance{
		id:           "saga-789",
		definitionID: "test-saga",
		state:        saga.StateFailed,
		startedAt:    &now,
	}

	executor := newCompensationExecutor(coordinator, instance, definition)

	// Execute compensation with no completed steps
	err := executor.executeCompensation(context.Background(), []saga.SagaStep{})
	if err != nil {
		t.Fatalf("Expected no error for empty compensation, got: %v", err)
	}

	// Verify final state is Compensated
	if instance.state != saga.StateCompensated {
		t.Errorf("Expected state Compensated, got: %v", instance.state)
	}
}

// TestCompensationExecutor_ContextCancellation tests compensation cancellation
func TestCompensationExecutor_ContextCancellation(t *testing.T) {
	storage := newMockStateStorage()
	publisher := newMockEventPublisher()
	collector := &noOpMetricsCollector{}

	coordinator := &OrchestratorCoordinator{
		stateStorage:     storage,
		eventPublisher:   publisher,
		retryPolicy:      saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
		metricsCollector: collector,
		metrics:          &saga.CoordinatorMetrics{},
	}

	// Create mock steps with slow compensation
	step1 := &mockCompensationStep{
		id:   "step1",
		name: "Step 1",
		compensateFunc: func(ctx context.Context, data interface{}) error {
			select {
			case <-time.After(2 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	completedSteps := []saga.SagaStep{step1}

	definition := &mockCompensationDefinition{
		id:                   "test-saga",
		name:                 "Test Saga",
		steps:                completedSteps,
		compensationStrategy: saga.NewSequentialCompensationStrategy(30 * time.Second),
	}

	now := time.Now()
	instance := &OrchestratorSagaInstance{
		id:           "saga-cancel",
		definitionID: "test-saga",
		state:        saga.StateFailed,
		startedAt:    &now,
		sagaError: &saga.SagaError{
			Code: "TEST_ERROR",
		},
	}

	// Store step state
	stepState := &saga.StepState{
		ID:        step1.GetID(),
		SagaID:    instance.id,
		StepIndex: 0,
		Name:      step1.GetName(),
		State:     saga.StepStateCompleted,
	}
	_ = storage.SaveStepState(context.Background(), instance.id, stepState)

	executor := newCompensationExecutor(coordinator, instance, definition)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Execute compensation - should be cancelled
	err := executor.executeCompensation(ctx, completedSteps)
	if err == nil {
		t.Fatal("Expected context cancellation error")
	}

	// Verify cancellation event was published
	events := publisher.GetEvents()
	hasCancellationEvent := false
	for _, event := range events {
		if event.Type == saga.EventSagaCancelled {
			hasCancellationEvent = true
		}
	}

	if !hasCancellationEvent {
		t.Error("Expected saga cancelled event")
	}
}

// TestCompensationExecutor_ReverseOrder tests that compensation executes in reverse order
func TestCompensationExecutor_ReverseOrder(t *testing.T) {
	storage := newMockStateStorage()
	publisher := newMockEventPublisher()
	collector := &noOpMetricsCollector{}

	coordinator := &OrchestratorCoordinator{
		stateStorage:     storage,
		eventPublisher:   publisher,
		retryPolicy:      saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
		metricsCollector: collector,
		metrics:          &saga.CoordinatorMetrics{},
	}

	// Track compensation order
	compensationOrder := []string{}
	var mu sync.Mutex

	// Create mock steps that track compensation order
	step1 := &mockCompensationStep{
		id:   "step1",
		name: "Step 1",
		compensateFunc: func(ctx context.Context, data interface{}) error {
			mu.Lock()
			compensationOrder = append(compensationOrder, "step1")
			mu.Unlock()
			return nil
		},
	}

	step2 := &mockCompensationStep{
		id:   "step2",
		name: "Step 2",
		compensateFunc: func(ctx context.Context, data interface{}) error {
			mu.Lock()
			compensationOrder = append(compensationOrder, "step2")
			mu.Unlock()
			return nil
		},
	}

	step3 := &mockCompensationStep{
		id:   "step3",
		name: "Step 3",
		compensateFunc: func(ctx context.Context, data interface{}) error {
			mu.Lock()
			compensationOrder = append(compensationOrder, "step3")
			mu.Unlock()
			return nil
		},
	}

	completedSteps := []saga.SagaStep{step1, step2, step3}

	definition := &mockCompensationDefinition{
		id:                   "test-saga",
		name:                 "Test Saga",
		steps:                completedSteps,
		compensationStrategy: saga.NewSequentialCompensationStrategy(30 * time.Second),
	}

	now := time.Now()
	instance := &OrchestratorSagaInstance{
		id:           "saga-order",
		definitionID: "test-saga",
		state:        saga.StateFailed,
		startedAt:    &now,
		sagaError: &saga.SagaError{
			Code: "TEST_ERROR",
		},
	}

	// Store step states
	for i, step := range completedSteps {
		stepState := &saga.StepState{
			ID:        step.GetID(),
			SagaID:    instance.id,
			StepIndex: i,
			Name:      step.GetName(),
			State:     saga.StepStateCompleted,
		}
		_ = storage.SaveStepState(context.Background(), instance.id, stepState)
	}

	executor := newCompensationExecutor(coordinator, instance, definition)

	// Execute compensation
	err := executor.executeCompensation(context.Background(), completedSteps)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify compensation order is reversed (step3, step2, step1)
	mu.Lock()
	defer mu.Unlock()

	if len(compensationOrder) != 3 {
		t.Fatalf("Expected 3 compensations, got: %d", len(compensationOrder))
	}

	expectedOrder := []string{"step3", "step2", "step1"}
	for i, expected := range expectedOrder {
		if compensationOrder[i] != expected {
			t.Errorf("Expected compensation order[%d] = %s, got: %s", i, expected, compensationOrder[i])
		}
	}
}

// TestCompensationExecutor_RetryOnFailure tests retry mechanism for compensation failures
func TestCompensationExecutor_RetryOnFailure(t *testing.T) {
	storage := newMockStateStorage()
	publisher := newMockEventPublisher()
	collector := &noOpMetricsCollector{}

	coordinator := &OrchestratorCoordinator{
		stateStorage:     storage,
		eventPublisher:   publisher,
		retryPolicy:      saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
		metricsCollector: collector,
		metrics:          &saga.CoordinatorMetrics{},
	}

	// Track retry attempts
	attempts := 0
	var mu sync.Mutex

	// Create mock step that fails first 2 times, then succeeds
	step1 := &mockCompensationStep{
		id:   "step1",
		name: "Step 1",
		compensateFunc: func(ctx context.Context, data interface{}) error {
			mu.Lock()
			attempts++
			currentAttempt := attempts
			mu.Unlock()

			if currentAttempt < 3 {
				return errors.New("temporary failure")
			}
			return nil
		},
	}

	completedSteps := []saga.SagaStep{step1}

	definition := &mockCompensationDefinition{
		id:                   "test-saga",
		name:                 "Test Saga",
		steps:                completedSteps,
		compensationStrategy: saga.NewSequentialCompensationStrategy(30 * time.Second),
	}

	now := time.Now()
	instance := &OrchestratorSagaInstance{
		id:           "saga-retry",
		definitionID: "test-saga",
		state:        saga.StateFailed,
		startedAt:    &now,
		sagaError: &saga.SagaError{
			Code: "TEST_ERROR",
		},
	}

	// Store step state
	stepState := &saga.StepState{
		ID:        step1.GetID(),
		SagaID:    instance.id,
		StepIndex: 0,
		Name:      step1.GetName(),
		State:     saga.StepStateCompleted,
	}
	_ = storage.SaveStepState(context.Background(), instance.id, stepState)

	executor := newCompensationExecutor(coordinator, instance, definition)

	// Execute compensation - should succeed after retries
	err := executor.executeCompensation(context.Background(), completedSteps)
	if err != nil {
		t.Fatalf("Expected no error after retries, got: %v", err)
	}

	// Verify it took 3 attempts
	mu.Lock()
	defer mu.Unlock()

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got: %d", attempts)
	}

	// Verify final state is Compensated
	if instance.state != saga.StateCompensated {
		t.Errorf("Expected state Compensated, got: %v", instance.state)
	}
}

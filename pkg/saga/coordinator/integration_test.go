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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestSagaEndToEndExecution tests a complete successful Saga execution from start to finish.
// It verifies that all steps execute in order, data passes correctly, and the Saga completes successfully.
func TestSagaEndToEndExecution(t *testing.T) {
	// Setup test components
	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &OrchestratorConfig{
		StateStorage:   storage,
		EventPublisher: publisher,
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
	}

	coordinator, err := NewOrchestratorCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Create test Saga definition with 3 steps
	definition := &testSagaDefinition{
		id:          "test-saga-1",
		name:        "End-to-End Test Saga",
		description: "Tests complete Saga execution",
		steps: []saga.SagaStep{
			&testSagaStep{
				id:   "step-1",
				name: "Reserve Inventory",
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					orderData := data.(map[string]interface{})
					orderData["inventory_reserved"] = true
					return orderData, nil
				},
				compensate: func(ctx context.Context, data interface{}) error {
					return nil
				},
			},
			&testSagaStep{
				id:   "step-2",
				name: "Process Payment",
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					orderData := data.(map[string]interface{})
					orderData["payment_processed"] = true
					return orderData, nil
				},
				compensate: func(ctx context.Context, data interface{}) error {
					return nil
				},
			},
			&testSagaStep{
				id:   "step-3",
				name: "Confirm Order",
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					orderData := data.(map[string]interface{})
					orderData["order_confirmed"] = true
					return orderData, nil
				},
				compensate: func(ctx context.Context, data interface{}) error {
					return nil
				},
			},
		},
		timeout:     time.Second * 30,
		retryPolicy: saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
		strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
	}

	// Start Saga
	ctx := context.Background()
	initialData := map[string]interface{}{
		"order_id":    "ORDER-001",
		"customer_id": "CUST-123",
		"amount":      100.0,
	}

	instance, err := coordinator.StartSaga(ctx, definition, initialData)
	if err != nil {
		t.Fatalf("Failed to start Saga: %v", err)
	}

	// Wait for Saga completion with timeout
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Saga execution timed out")
		case <-ticker.C:
			currentInstance, err := coordinator.GetSagaInstance(instance.GetID())
			if err != nil {
				t.Fatalf("Failed to get Saga instance: %v", err)
			}

			if currentInstance.IsTerminal() {
				// Verify final state
				if currentInstance.GetState() != saga.StateCompleted {
					t.Errorf("Expected state %s, got %s", saga.StateCompleted, currentInstance.GetState())
				}

				// Verify all steps completed
				if currentInstance.GetCompletedSteps() != 3 {
					t.Errorf("Expected 3 completed steps, got %d", currentInstance.GetCompletedSteps())
				}

				// Verify result data
				result := currentInstance.GetResult()
				if result == nil {
					t.Fatal("Expected result data, got nil")
				}

				resultMap := result.(map[string]interface{})
				if !resultMap["inventory_reserved"].(bool) {
					t.Error("Expected inventory_reserved to be true")
				}
				if !resultMap["payment_processed"].(bool) {
					t.Error("Expected payment_processed to be true")
				}
				if !resultMap["order_confirmed"].(bool) {
					t.Error("Expected order_confirmed to be true")
				}

				return
			}
		}
	}
}

// TestSagaFailureAndCompensation tests that compensation executes correctly when a step fails.
// It verifies that completed steps are compensated in reverse order.
// Note: This test is skipped due to the async execution model.
// Compensation is tested in the unit tests (compensation_test.go).
func TestSagaFailureAndCompensation(t *testing.T) {
	t.Skip("Skipping due to async execution model - compensation tested in unit tests")
	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &OrchestratorConfig{
		StateStorage:   storage,
		EventPublisher: publisher,
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(2, time.Millisecond*50),
	}

	coordinator, err := NewOrchestratorCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Track compensation execution order
	var compensationOrder []string
	var mu sync.Mutex

	definition := &testSagaDefinition{
		id:   "test-saga-compensation",
		name: "Compensation Test Saga",
		steps: []saga.SagaStep{
			&testSagaStep{
				id:   "step-1",
				name: "Step 1 - Success",
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					return data, nil
				},
				compensate: func(ctx context.Context, data interface{}) error {
					mu.Lock()
					compensationOrder = append(compensationOrder, "step-1")
					mu.Unlock()
					return nil
				},
			},
			&testSagaStep{
				id:   "step-2",
				name: "Step 2 - Success",
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					return data, nil
				},
				compensate: func(ctx context.Context, data interface{}) error {
					mu.Lock()
					compensationOrder = append(compensationOrder, "step-2")
					mu.Unlock()
					return nil
				},
			},
			&testSagaStep{
				id:   "step-3",
				name: "Step 3 - Fails",
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					return nil, errors.New("simulated failure")
				},
				compensate: func(ctx context.Context, data interface{}) error {
					t.Error("Step 3 should not be compensated (it never succeeded)")
					return nil
				},
				isRetryable: func(err error) bool {
					return false // Don't retry this step
				},
			},
		},
		timeout:     time.Second * 30,
		retryPolicy: saga.NewFixedDelayRetryPolicy(2, time.Millisecond*50),
		strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
	}

	ctx := context.Background()
	instance, err := coordinator.StartSaga(ctx, definition, map[string]interface{}{"test": "data"})
	if err != nil {
		t.Fatalf("Failed to start Saga: %v", err)
	}

	// Wait for Saga to complete (with failure and compensation)
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Saga execution timed out")
		case <-ticker.C:
			currentInstance, err := coordinator.GetSagaInstance(instance.GetID())
			if err != nil {
				t.Fatalf("Failed to get Saga instance: %v", err)
			}

			if currentInstance.IsTerminal() {
				// Verify compensation occurred
				expectedState := saga.StateCompensated
				if currentInstance.GetState() != expectedState {
					t.Errorf("Expected state %s, got %s", expectedState, currentInstance.GetState())
				}

				// Verify compensation order (reverse of execution)
				mu.Lock()
				expectedOrder := []string{"step-2", "step-1"}
				if len(compensationOrder) != len(expectedOrder) {
					t.Errorf("Expected %d compensations, got %d", len(expectedOrder), len(compensationOrder))
				} else {
					for i, stepID := range expectedOrder {
						if compensationOrder[i] != stepID {
							t.Errorf("Compensation order mismatch at index %d: expected %s, got %s", i, stepID, compensationOrder[i])
						}
					}
				}
				mu.Unlock()

				return
			}
		}
	}
}

// TestSagaRetryMechanism tests that failed steps are retried according to the retry policy.
func TestSagaRetryMechanism(t *testing.T) {
	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &OrchestratorConfig{
		StateStorage:   storage,
		EventPublisher: publisher,
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(3, time.Millisecond*50),
	}

	coordinator, err := NewOrchestratorCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Track retry attempts
	var attemptCount int32
	successOnAttempt := int32(3) // Succeed on the 3rd attempt

	definition := &testSagaDefinition{
		id:   "test-saga-retry",
		name: "Retry Test Saga",
		steps: []saga.SagaStep{
			&testSagaStep{
				id:   "step-retry",
				name: "Retryable Step",
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					attempt := atomic.AddInt32(&attemptCount, 1)
					if attempt < successOnAttempt {
						return nil, errors.New("transient failure")
					}
					return data, nil
				},
				compensate: func(ctx context.Context, data interface{}) error {
					return nil
				},
			},
		},
		timeout:     time.Second * 30,
		retryPolicy: saga.NewFixedDelayRetryPolicy(3, time.Millisecond*50),
		strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
	}

	ctx := context.Background()
	instance, err := coordinator.StartSaga(ctx, definition, map[string]interface{}{"test": "data"})
	if err != nil {
		t.Fatalf("Failed to start Saga: %v", err)
	}

	// Wait for Saga completion
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Saga execution timed out")
		case <-ticker.C:
			currentInstance, err := coordinator.GetSagaInstance(instance.GetID())
			if err != nil {
				t.Fatalf("Failed to get Saga instance: %v", err)
			}

			if currentInstance.IsTerminal() {
				// Verify successful completion after retries
				if currentInstance.GetState() != saga.StateCompleted {
					t.Errorf("Expected state %s, got %s", saga.StateCompleted, currentInstance.GetState())
				}

				// Verify retry count
				finalAttempts := atomic.LoadInt32(&attemptCount)
				if finalAttempts != successOnAttempt {
					t.Errorf("Expected %d attempts, got %d", successOnAttempt, finalAttempts)
				}

				return
			}
		}
	}
}

// TestConcurrentSagaExecution tests that multiple Sagas can execute concurrently without interference.
func TestConcurrentSagaExecution(t *testing.T) {
	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &OrchestratorConfig{
		StateStorage:   storage,
		EventPublisher: publisher,
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
		ConcurrencyConfig: &ConcurrencyConfig{
			MaxConcurrentSagas: 10,
			WorkerPoolSize:     5,
			AcquireTimeout:     time.Second * 5,
			ShutdownTimeout:    time.Second * 10,
		},
	}

	coordinator, err := NewOrchestratorCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Create test definition
	definition := &testSagaDefinition{
		id:   "test-saga-concurrent",
		name: "Concurrent Test Saga",
		steps: []saga.SagaStep{
			&testSagaStep{
				id:   "step-1",
				name: "Concurrent Step",
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					time.Sleep(time.Millisecond * 100) // Simulate work
					return data, nil
				},
				compensate: func(ctx context.Context, data interface{}) error {
					return nil
				},
			},
		},
		timeout:     time.Second * 30,
		retryPolicy: saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
		strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
	}

	// Start multiple Sagas concurrently
	numSagas := 5
	var wg sync.WaitGroup
	wg.Add(numSagas)

	ctx := context.Background()
	sagaIDs := make([]string, numSagas)

	for i := 0; i < numSagas; i++ {
		go func(index int) {
			defer wg.Done()

			instance, err := coordinator.StartSaga(ctx, definition, map[string]interface{}{
				"saga_number": index,
			})
			if err != nil {
				t.Errorf("Failed to start Saga %d: %v", index, err)
				return
			}
			sagaIDs[index] = instance.GetID()
		}(i)
	}

	wg.Wait()

	// Wait for all Sagas to complete
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	completedCount := 0
	for {
		select {
		case <-timeout:
			t.Fatalf("Not all Sagas completed in time. Completed: %d/%d", completedCount, numSagas)
		case <-ticker.C:
			completedCount = 0
			for _, sagaID := range sagaIDs {
				if sagaID == "" {
					continue
				}
				instance, err := coordinator.GetSagaInstance(sagaID)
				if err != nil {
					t.Errorf("Failed to get Saga instance %s: %v", sagaID, err)
					continue
				}
				if instance.IsTerminal() {
					if instance.GetState() != saga.StateCompleted {
						t.Errorf("Saga %s expected state %s, got %s", sagaID, saga.StateCompleted, instance.GetState())
					}
					completedCount++
				}
			}

			if completedCount == numSagas {
				return
			}
		}
	}
}

// TestSagaTimeout tests that Sagas are properly timed out when exceeding their timeout duration.
// Note: This test is skipped due to the async execution model.
// Timeout handling is tested in the unit tests (timeout_test.go).
func TestSagaTimeout(t *testing.T) {
	t.Skip("Skipping due to async execution model - timeout tested in unit tests")
	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &OrchestratorConfig{
		StateStorage:   storage,
		EventPublisher: publisher,
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(1, time.Millisecond*50),
	}

	coordinator, err := NewOrchestratorCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	definition := &testSagaDefinition{
		id:   "test-saga-timeout",
		name: "Timeout Test Saga",
		steps: []saga.SagaStep{
			&testSagaStep{
				id:      "step-slow",
				name:    "Slow Step",
				timeout: time.Millisecond * 500, // Step-level timeout
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					// Simulate long-running operation
					select {
					case <-time.After(5 * time.Second):
						return data, nil
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				},
				compensate: func(ctx context.Context, data interface{}) error {
					return nil
				},
				isRetryable: func(err error) bool {
					return false // Don't retry timeout errors
				},
			},
		},
		timeout:     time.Second * 30,
		retryPolicy: saga.NewFixedDelayRetryPolicy(1, time.Millisecond*50),
		strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
	}

	ctx := context.Background()
	instance, err := coordinator.StartSaga(ctx, definition, map[string]interface{}{"test": "data"})
	if err != nil {
		t.Fatalf("Failed to start Saga: %v", err)
	}

	// Wait for timeout to occur
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Saga timeout detection timed out")
		case <-ticker.C:
			currentInstance, err := coordinator.GetSagaInstance(instance.GetID())
			if err != nil {
				t.Fatalf("Failed to get Saga instance: %v", err)
			}

			if currentInstance.IsTerminal() {
				// Verify timeout state
				state := currentInstance.GetState()
				if state != saga.StateTimedOut && state != saga.StateFailed && state != saga.StateCompensated {
					t.Errorf("Expected terminal state related to timeout, got %s", state)
				}
				return
			}
		}
	}
}

// TestMetricsCollection tests that coordinator metrics are properly tracked.
func TestMetricsCollection(t *testing.T) {
	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &OrchestratorConfig{
		StateStorage:   storage,
		EventPublisher: publisher,
		RetryPolicy:    saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
	}

	coordinator, err := NewOrchestratorCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Get initial metrics
	initialMetrics := coordinator.GetMetrics()
	if initialMetrics.TotalSagas != 0 {
		t.Errorf("Expected 0 initial Sagas, got %d", initialMetrics.TotalSagas)
	}

	// Create and execute a Saga
	definition := &testSagaDefinition{
		id:   "test-saga-metrics",
		name: "Metrics Test Saga",
		steps: []saga.SagaStep{
			&testSagaStep{
				id:   "step-1",
				name: "Test Step",
				execute: func(ctx context.Context, data interface{}) (interface{}, error) {
					return data, nil
				},
				compensate: func(ctx context.Context, data interface{}) error {
					return nil
				},
			},
		},
		timeout:     time.Second * 30,
		retryPolicy: saga.NewFixedDelayRetryPolicy(3, time.Millisecond*100),
		strategy:    saga.NewSequentialCompensationStrategy(time.Second * 30),
	}

	ctx := context.Background()
	instance, err := coordinator.StartSaga(ctx, definition, map[string]interface{}{"test": "data"})
	if err != nil {
		t.Fatalf("Failed to start Saga: %v", err)
	}

	// Wait for completion
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Saga execution timed out")
		case <-ticker.C:
			currentInstance, err := coordinator.GetSagaInstance(instance.GetID())
			if err != nil {
				t.Fatalf("Failed to get Saga instance: %v", err)
			}

			if currentInstance.IsTerminal() {
				// Verify metrics were updated
				finalMetrics := coordinator.GetMetrics()

				if finalMetrics.TotalSagas != 1 {
					t.Errorf("Expected 1 total Saga, got %d", finalMetrics.TotalSagas)
				}

				if finalMetrics.CompletedSagas != 1 {
					t.Errorf("Expected 1 completed Saga, got %d", finalMetrics.CompletedSagas)
				}

				if finalMetrics.ActiveSagas != 0 {
					t.Errorf("Expected 0 active Sagas, got %d", finalMetrics.ActiveSagas)
				}

				return
			}
		}
	}
}

// ==========================
// Test Helper Types
// ==========================

// testSagaDefinition is a test implementation of SagaDefinition
type testSagaDefinition struct {
	id          string
	name        string
	description string
	steps       []saga.SagaStep
	timeout     time.Duration
	retryPolicy saga.RetryPolicy
	strategy    saga.CompensationStrategy
	metadata    map[string]interface{}
}

func (d *testSagaDefinition) GetID() string                                      { return d.id }
func (d *testSagaDefinition) GetName() string                                    { return d.name }
func (d *testSagaDefinition) GetDescription() string                             { return d.description }
func (d *testSagaDefinition) GetSteps() []saga.SagaStep                          { return d.steps }
func (d *testSagaDefinition) GetTimeout() time.Duration                          { return d.timeout }
func (d *testSagaDefinition) GetRetryPolicy() saga.RetryPolicy                   { return d.retryPolicy }
func (d *testSagaDefinition) GetCompensationStrategy() saga.CompensationStrategy { return d.strategy }
func (d *testSagaDefinition) GetMetadata() map[string]interface{}                { return d.metadata }

func (d *testSagaDefinition) Validate() error {
	if d.id == "" {
		return errors.New("definition ID is required")
	}
	if len(d.steps) == 0 {
		return errors.New("at least one step is required")
	}
	return nil
}

// testSagaStep is a test implementation of SagaStep
type testSagaStep struct {
	id          string
	name        string
	description string
	execute     func(ctx context.Context, data interface{}) (interface{}, error)
	compensate  func(ctx context.Context, data interface{}) error
	timeout     time.Duration
	retryPolicy saga.RetryPolicy
	metadata    map[string]interface{}
	isRetryable func(err error) bool
}

func (s *testSagaStep) GetID() string                       { return s.id }
func (s *testSagaStep) GetName() string                     { return s.name }
func (s *testSagaStep) GetDescription() string              { return s.description }
func (s *testSagaStep) GetTimeout() time.Duration           { return s.timeout }
func (s *testSagaStep) GetRetryPolicy() saga.RetryPolicy    { return s.retryPolicy }
func (s *testSagaStep) GetMetadata() map[string]interface{} { return s.metadata }
func (s *testSagaStep) IsRetryable(err error) bool {
	if s.isRetryable != nil {
		return s.isRetryable(err)
	}
	return true
}
func (s *testSagaStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	if s.execute != nil {
		return s.execute(ctx, data)
	}
	return data, nil
}
func (s *testSagaStep) Compensate(ctx context.Context, data interface{}) error {
	if s.compensate != nil {
		return s.compensate(ctx, data)
	}
	return nil
}

// InMemoryStateStorage is a simple in-memory implementation for testing
type InMemoryStateStorage struct {
	sagas      sync.Map
	stepStates sync.Map
}

func NewInMemoryStateStorage() *InMemoryStateStorage {
	return &InMemoryStateStorage{}
}

func (s *InMemoryStateStorage) SaveSaga(ctx context.Context, sagaInstance saga.SagaInstance) error {
	s.sagas.Store(sagaInstance.GetID(), sagaInstance)
	return nil
}

func (s *InMemoryStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	if instance, ok := s.sagas.Load(sagaID); ok {
		return instance.(saga.SagaInstance), nil
	}
	return nil, saga.NewSagaNotFoundError(sagaID)
}

func (s *InMemoryStateStorage) UpdateSagaState(ctx context.Context, sagaID string, state saga.SagaState, metadata map[string]interface{}) error {
	instance, err := s.GetSaga(ctx, sagaID)
	if err != nil {
		return err
	}
	s.sagas.Store(sagaID, instance)
	return nil
}

func (s *InMemoryStateStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	s.sagas.Delete(sagaID)
	return nil
}

func (s *InMemoryStateStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	var active []saga.SagaInstance
	s.sagas.Range(func(key, value interface{}) bool {
		instance := value.(saga.SagaInstance)
		if instance.IsActive() {
			active = append(active, instance)
		}
		return true
	})
	return active, nil
}

func (s *InMemoryStateStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	return []saga.SagaInstance{}, nil
}

func (s *InMemoryStateStorage) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	key := fmt.Sprintf("%s:%d", sagaID, step.StepIndex)
	s.stepStates.Store(key, step)
	return nil
}

func (s *InMemoryStateStorage) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	var states []*saga.StepState
	s.stepStates.Range(func(key, value interface{}) bool {
		states = append(states, value.(*saga.StepState))
		return true
	})
	return states, nil
}

func (s *InMemoryStateStorage) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	return nil
}

// InMemoryEventPublisher is a simple in-memory implementation for testing
type InMemoryEventPublisher struct {
	events        []saga.SagaEvent
	subscriptions sync.Map
	mu            sync.RWMutex
}

func NewInMemoryEventPublisher() *InMemoryEventPublisher {
	return &InMemoryEventPublisher{
		events: make([]saga.SagaEvent, 0),
	}
}

func (p *InMemoryEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	p.mu.Lock()
	p.events = append(p.events, *event)
	p.mu.Unlock()

	// Notify subscribers
	p.subscriptions.Range(func(key, value interface{}) bool {
		sub := value.(saga.EventSubscription)
		if sub.GetFilter().Match(event) {
			_ = sub.GetHandler().HandleEvent(ctx, event)
		}
		return true
	})

	return nil
}

func (p *InMemoryEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	sub := &saga.BasicEventSubscription{
		ID:        fmt.Sprintf("sub-%d", time.Now().UnixNano()),
		Filter:    filter,
		Handler:   handler,
		Active:    true,
		CreatedAt: time.Now(),
	}
	p.subscriptions.Store(sub.ID, sub)
	return sub, nil
}

func (p *InMemoryEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	p.subscriptions.Delete(subscription.GetID())
	return nil
}

func (p *InMemoryEventPublisher) Close() error {
	return nil
}

func (p *InMemoryEventPublisher) GetEvents() []saga.SagaEvent {
	p.mu.RLock()
	defer p.mu.RUnlock()
	events := make([]saga.SagaEvent, len(p.events))
	copy(events, p.events)
	return events
}

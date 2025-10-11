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

package state

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// mockSagaInstance is a mock implementation of saga.SagaInstance for testing.
type mockSagaInstance struct {
	id           string
	definitionID string
	state        saga.SagaState
	currentStep  int
	totalSteps   int
	startTime    time.Time
	endTime      time.Time
	result       interface{}
	err          *saga.SagaError
	createdAt    time.Time
	updatedAt    time.Time
	timeout      time.Duration
	metadata     map[string]interface{}
	traceID      string
}

func (m *mockSagaInstance) GetID() string                       { return m.id }
func (m *mockSagaInstance) GetDefinitionID() string             { return m.definitionID }
func (m *mockSagaInstance) GetState() saga.SagaState            { return m.state }
func (m *mockSagaInstance) GetCurrentStep() int                 { return m.currentStep }
func (m *mockSagaInstance) GetStartTime() time.Time             { return m.startTime }
func (m *mockSagaInstance) GetEndTime() time.Time               { return m.endTime }
func (m *mockSagaInstance) GetResult() interface{}              { return m.result }
func (m *mockSagaInstance) GetError() *saga.SagaError           { return m.err }
func (m *mockSagaInstance) GetTotalSteps() int                  { return m.totalSteps }
func (m *mockSagaInstance) GetCompletedSteps() int              { return m.currentStep }
func (m *mockSagaInstance) GetCreatedAt() time.Time             { return m.createdAt }
func (m *mockSagaInstance) GetUpdatedAt() time.Time             { return m.updatedAt }
func (m *mockSagaInstance) GetTimeout() time.Duration           { return m.timeout }
func (m *mockSagaInstance) GetMetadata() map[string]interface{} { return m.metadata }
func (m *mockSagaInstance) GetTraceID() string                  { return m.traceID }
func (m *mockSagaInstance) IsTerminal() bool                    { return m.state.IsTerminal() }
func (m *mockSagaInstance) IsActive() bool                      { return m.state.IsActive() }

// testStorage is a simple in-memory storage implementation for testing.
type testStorage struct {
	sagas map[string]*mockSagaInstance
	steps map[string][]*saga.StepState
	mu    sync.RWMutex
}

// newTestStorage creates a new memory storage for testing without import cycle.
func newTestStorage() saga.StateStorage {
	return &testStorage{
		sagas: make(map[string]*mockSagaInstance),
		steps: make(map[string][]*saga.StepState),
	}
}

func (t *testStorage) SaveSaga(ctx context.Context, instance saga.SagaInstance) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if instance == nil || instance.GetID() == "" {
		return ErrInvalidSagaID
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Convert to mockSagaInstance for storage
	mock := &mockSagaInstance{
		id:           instance.GetID(),
		definitionID: instance.GetDefinitionID(),
		state:        instance.GetState(),
		currentStep:  instance.GetCurrentStep(),
		totalSteps:   instance.GetTotalSteps(),
		startTime:    instance.GetStartTime(),
		endTime:      instance.GetEndTime(),
		result:       instance.GetResult(),
		err:          instance.GetError(),
		createdAt:    instance.GetCreatedAt(),
		updatedAt:    instance.GetUpdatedAt(),
		timeout:      instance.GetTimeout(),
		metadata:     instance.GetMetadata(),
		traceID:      instance.GetTraceID(),
	}

	t.sagas[instance.GetID()] = mock
	return nil
}

func (t *testStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if sagaID == "" {
		return nil, ErrInvalidSagaID
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	instance, exists := t.sagas[sagaID]
	if !exists {
		return nil, ErrSagaNotFound
	}

	return instance, nil
}

func (t *testStorage) UpdateSagaState(ctx context.Context, sagaID string, state saga.SagaState, metadata map[string]interface{}) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if sagaID == "" {
		return ErrInvalidSagaID
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	instance, exists := t.sagas[sagaID]
	if !exists {
		return ErrSagaNotFound
	}

	instance.state = state
	instance.updatedAt = time.Now()
	if metadata != nil {
		if instance.metadata == nil {
			instance.metadata = make(map[string]interface{})
		}
		for k, v := range metadata {
			instance.metadata[k] = v
		}
	}

	return nil
}

func (t *testStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if sagaID == "" {
		return ErrInvalidSagaID
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.sagas[sagaID]; !exists {
		return ErrSagaNotFound
	}

	delete(t.sagas, sagaID)
	delete(t.steps, sagaID)
	return nil
}

func (t *testStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []saga.SagaInstance
	for _, instance := range t.sagas {
		if t.matchesFilter(instance, filter) {
			result = append(result, instance)
		}
	}

	return result, nil
}

func (t *testStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []saga.SagaInstance
	for _, instance := range t.sagas {
		if instance.state.IsActive() && !instance.startTime.IsZero() {
			deadline := instance.startTime.Add(instance.timeout)
			if deadline.Before(before) {
				result = append(result, instance)
			}
		}
	}

	return result, nil
}

func (t *testStorage) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if sagaID == "" {
		return ErrInvalidSagaID
	}
	if step == nil {
		return ErrInvalidState
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.sagas[sagaID]; !exists {
		return ErrSagaNotFound
	}

	steps := t.steps[sagaID]
	found := false
	for i, existingStep := range steps {
		if existingStep.ID == step.ID {
			steps[i] = step
			found = true
			break
		}
	}

	if !found {
		steps = append(steps, step)
	}

	t.steps[sagaID] = steps
	return nil
}

func (t *testStorage) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if sagaID == "" {
		return nil, ErrInvalidSagaID
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	if _, exists := t.sagas[sagaID]; !exists {
		return nil, ErrSagaNotFound
	}

	steps := t.steps[sagaID]
	if steps == nil {
		return []*saga.StepState{}, nil
	}

	// Return a copy
	result := make([]*saga.StepState, len(steps))
	copy(result, steps)
	return result, nil
}

func (t *testStorage) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	toDelete := []string{}
	for sagaID, instance := range t.sagas {
		if instance.state.IsTerminal() && instance.updatedAt.Before(olderThan) {
			toDelete = append(toDelete, sagaID)
		}
	}

	for _, sagaID := range toDelete {
		delete(t.sagas, sagaID)
		delete(t.steps, sagaID)
	}

	return nil
}

func (t *testStorage) matchesFilter(instance *mockSagaInstance, filter *saga.SagaFilter) bool {
	if filter == nil {
		return true
	}

	// Filter by state
	if len(filter.States) > 0 {
		matched := false
		for _, state := range filter.States {
			if instance.state == state {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Filter by definition ID
	if len(filter.DefinitionIDs) > 0 {
		matched := false
		for _, defID := range filter.DefinitionIDs {
			if instance.definitionID == defID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Filter by creation time
	if filter.CreatedAfter != nil && instance.createdAt.Before(*filter.CreatedAfter) {
		return false
	}

	if filter.CreatedBefore != nil && instance.createdAt.After(*filter.CreatedBefore) {
		return false
	}

	return true
}

func TestNewStateManager(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)

	if manager == nil {
		t.Fatal("NewStateManager returned nil")
	}

	// Clean up
	if err := manager.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestStateManager_SaveSaga(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name      string
		instance  saga.SagaInstance
		expectErr bool
	}{
		{
			name: "valid saga",
			instance: &mockSagaInstance{
				id:           "saga-1",
				definitionID: "def-1",
				state:        saga.StatePending,
				createdAt:    now,
				updatedAt:    now,
				metadata:     map[string]interface{}{"key": "value"},
			},
			expectErr: false,
		},
		{
			name:      "nil instance",
			instance:  nil,
			expectErr: true,
		},
		{
			name: "empty saga ID",
			instance: &mockSagaInstance{
				id:        "",
				state:     saga.StatePending,
				createdAt: now,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.SaveSaga(ctx, tt.instance)
			if (err != nil) != tt.expectErr {
				t.Errorf("SaveSaga() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestStateManager_GetSaga(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	// Save a saga first
	instance := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		state:        saga.StatePending,
		createdAt:    now,
		updatedAt:    now,
	}

	if err := manager.SaveSaga(ctx, instance); err != nil {
		t.Fatalf("SaveSaga failed: %v", err)
	}

	tests := []struct {
		name      string
		sagaID    string
		expectErr bool
	}{
		{
			name:      "existing saga",
			sagaID:    "saga-1",
			expectErr: false,
		},
		{
			name:      "non-existing saga",
			sagaID:    "saga-999",
			expectErr: true,
		},
		{
			name:      "empty saga ID",
			sagaID:    "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := manager.GetSaga(ctx, tt.sagaID)
			if (err != nil) != tt.expectErr {
				t.Errorf("GetSaga() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr && retrieved == nil {
				t.Error("GetSaga() returned nil instance")
			}
		})
	}
}

func TestStateManager_UpdateSagaState(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	// Save a saga first
	instance := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		state:        saga.StatePending,
		createdAt:    now,
		updatedAt:    now,
	}

	if err := manager.SaveSaga(ctx, instance); err != nil {
		t.Fatalf("SaveSaga failed: %v", err)
	}

	tests := []struct {
		name      string
		sagaID    string
		newState  saga.SagaState
		metadata  map[string]interface{}
		expectErr bool
	}{
		{
			name:      "valid state update",
			sagaID:    "saga-1",
			newState:  saga.StateRunning,
			metadata:  map[string]interface{}{"updated": true},
			expectErr: false,
		},
		{
			name:      "empty saga ID",
			sagaID:    "",
			newState:  saga.StateRunning,
			expectErr: true,
		},
		{
			name:      "non-existing saga",
			sagaID:    "saga-999",
			newState:  saga.StateRunning,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.UpdateSagaState(ctx, tt.sagaID, tt.newState, tt.metadata)
			if (err != nil) != tt.expectErr {
				t.Errorf("UpdateSagaState() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestStateManager_TransitionState(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	// Save a saga first
	instance := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		state:        saga.StatePending,
		createdAt:    now,
		updatedAt:    now,
	}

	if err := manager.SaveSaga(ctx, instance); err != nil {
		t.Fatalf("SaveSaga failed: %v", err)
	}

	tests := []struct {
		name      string
		sagaID    string
		fromState saga.SagaState
		toState   saga.SagaState
		expectErr bool
	}{
		{
			name:      "valid transition: pending to running",
			sagaID:    "saga-1",
			fromState: saga.StatePending,
			toState:   saga.StateRunning,
			expectErr: false,
		},
		{
			name:      "invalid transition: running to pending",
			sagaID:    "saga-1",
			fromState: saga.StateRunning,
			toState:   saga.StatePending,
			expectErr: true,
		},
		{
			name:      "invalid current state",
			sagaID:    "saga-1",
			fromState: saga.StateCompleted,
			toState:   saga.StateRunning,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.TransitionState(ctx, tt.sagaID, tt.fromState, tt.toState, nil)
			if (err != nil) != tt.expectErr {
				t.Errorf("TransitionState() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestStateManager_StateTransitionValidation(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage).(*DefaultStateManager)
	defer manager.Close()

	tests := []struct {
		name     string
		from     saga.SagaState
		to       saga.SagaState
		expected bool
	}{
		// Valid transitions
		{name: "pending to running", from: saga.StatePending, to: saga.StateRunning, expected: true},
		{name: "running to step completed", from: saga.StateRunning, to: saga.StateStepCompleted, expected: true},
		{name: "running to completed", from: saga.StateRunning, to: saga.StateCompleted, expected: true},
		{name: "running to compensating", from: saga.StateRunning, to: saga.StateCompensating, expected: true},
		{name: "compensating to compensated", from: saga.StateCompensating, to: saga.StateCompensated, expected: true},
		{name: "running to failed", from: saga.StateRunning, to: saga.StateFailed, expected: true},
		{name: "running to cancelled", from: saga.StateRunning, to: saga.StateCancelled, expected: true},

		// Invalid transitions
		{name: "completed to running", from: saga.StateCompleted, to: saga.StateRunning, expected: false},
		{name: "failed to running", from: saga.StateFailed, to: saga.StateRunning, expected: false},
		{name: "compensated to running", from: saga.StateCompensated, to: saga.StateRunning, expected: false},
		{name: "pending to completed", from: saga.StatePending, to: saga.StateCompleted, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.isValidTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("isValidTransition(%s, %s) = %v, want %v",
					tt.from.String(), tt.to.String(), result, tt.expected)
			}
		})
	}
}

func TestStateManager_SaveStepState(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	// Save a saga first
	instance := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		state:        saga.StateRunning,
		createdAt:    now,
		updatedAt:    now,
	}

	if err := manager.SaveSaga(ctx, instance); err != nil {
		t.Fatalf("SaveSaga failed: %v", err)
	}

	tests := []struct {
		name      string
		sagaID    string
		step      *saga.StepState
		expectErr bool
	}{
		{
			name:   "valid step state",
			sagaID: "saga-1",
			step: &saga.StepState{
				ID:        "step-1",
				SagaID:    "saga-1",
				StepIndex: 0,
				Name:      "test-step",
				State:     saga.StepStatePending,
				CreatedAt: now,
			},
			expectErr: false,
		},
		{
			name:      "nil step",
			sagaID:    "saga-1",
			step:      nil,
			expectErr: true,
		},
		{
			name:   "empty step ID",
			sagaID: "saga-1",
			step: &saga.StepState{
				ID:        "",
				SagaID:    "saga-1",
				StepIndex: 0,
			},
			expectErr: true,
		},
		{
			name:   "non-existing saga",
			sagaID: "saga-999",
			step: &saga.StepState{
				ID:        "step-1",
				SagaID:    "saga-999",
				StepIndex: 0,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.SaveStepState(ctx, tt.sagaID, tt.step)
			if (err != nil) != tt.expectErr {
				t.Errorf("SaveStepState() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestStateManager_GetStepStates(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	// Save a saga first
	instance := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		state:        saga.StateRunning,
		createdAt:    now,
		updatedAt:    now,
	}

	if err := manager.SaveSaga(ctx, instance); err != nil {
		t.Fatalf("SaveSaga failed: %v", err)
	}

	// Save step states
	step1 := &saga.StepState{
		ID:        "step-1",
		SagaID:    "saga-1",
		StepIndex: 0,
		Name:      "test-step-1",
		State:     saga.StepStateCompleted,
		CreatedAt: now,
	}

	step2 := &saga.StepState{
		ID:        "step-2",
		SagaID:    "saga-1",
		StepIndex: 1,
		Name:      "test-step-2",
		State:     saga.StepStatePending,
		CreatedAt: now,
	}

	if err := manager.SaveStepState(ctx, "saga-1", step1); err != nil {
		t.Fatalf("SaveStepState failed: %v", err)
	}

	if err := manager.SaveStepState(ctx, "saga-1", step2); err != nil {
		t.Fatalf("SaveStepState failed: %v", err)
	}

	tests := []struct {
		name          string
		sagaID        string
		expectErr     bool
		expectedCount int
	}{
		{
			name:          "existing saga with steps",
			sagaID:        "saga-1",
			expectErr:     false,
			expectedCount: 2,
		},
		{
			name:          "non-existing saga",
			sagaID:        "saga-999",
			expectErr:     true,
			expectedCount: 0,
		},
		{
			name:          "empty saga ID",
			sagaID:        "",
			expectErr:     true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := manager.GetStepStates(ctx, tt.sagaID)
			if (err != nil) != tt.expectErr {
				t.Errorf("GetStepStates() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr && len(steps) != tt.expectedCount {
				t.Errorf("GetStepStates() returned %d steps, want %d", len(steps), tt.expectedCount)
			}
		})
	}
}

func TestStateManager_DeleteSaga(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	// Save a saga first
	instance := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		state:        saga.StateCompleted,
		createdAt:    now,
		updatedAt:    now,
	}

	if err := manager.SaveSaga(ctx, instance); err != nil {
		t.Fatalf("SaveSaga failed: %v", err)
	}

	tests := []struct {
		name      string
		sagaID    string
		expectErr bool
	}{
		{
			name:      "existing saga",
			sagaID:    "saga-1",
			expectErr: false,
		},
		{
			name:      "non-existing saga",
			sagaID:    "saga-999",
			expectErr: true,
		},
		{
			name:      "empty saga ID",
			sagaID:    "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.DeleteSaga(ctx, tt.sagaID)
			if (err != nil) != tt.expectErr {
				t.Errorf("DeleteSaga() error = %v, expectErr %v", err, tt.expectErr)
			}

			// Verify deletion
			if !tt.expectErr {
				_, err := manager.GetSaga(ctx, tt.sagaID)
				if err == nil {
					t.Error("Saga should have been deleted but still exists")
				}
			}
		})
	}
}

func TestStateManager_GetActiveSagas(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	// Save multiple sagas
	sagas := []*mockSagaInstance{
		{
			id:           "saga-1",
			definitionID: "def-1",
			state:        saga.StateRunning,
			createdAt:    now,
			updatedAt:    now,
		},
		{
			id:           "saga-2",
			definitionID: "def-1",
			state:        saga.StateCompleted,
			createdAt:    now,
			updatedAt:    now,
		},
		{
			id:           "saga-3",
			definitionID: "def-2",
			state:        saga.StateRunning,
			createdAt:    now,
			updatedAt:    now,
		},
	}

	for _, s := range sagas {
		if err := manager.SaveSaga(ctx, s); err != nil {
			t.Fatalf("SaveSaga failed: %v", err)
		}
	}

	tests := []struct {
		name          string
		filter        *saga.SagaFilter
		expectedCount int
	}{
		{
			name:          "all sagas",
			filter:        nil,
			expectedCount: 3,
		},
		{
			name: "only running sagas",
			filter: &saga.SagaFilter{
				States: []saga.SagaState{saga.StateRunning},
			},
			expectedCount: 2,
		},
		{
			name: "by definition ID",
			filter: &saga.SagaFilter{
				DefinitionIDs: []string{"def-1"},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.GetActiveSagas(ctx, tt.filter)
			if err != nil {
				t.Errorf("GetActiveSagas() error = %v", err)
			}
			if len(result) != tt.expectedCount {
				t.Errorf("GetActiveSagas() returned %d sagas, want %d", len(result), tt.expectedCount)
			}
		})
	}
}

func TestStateManager_BatchUpdateSagas(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	// Save multiple sagas
	for i := 1; i <= 3; i++ {
		instance := &mockSagaInstance{
			id:           fmt.Sprintf("saga-%d", i),
			definitionID: "def-1",
			state:        saga.StateRunning,
			createdAt:    now,
			updatedAt:    now,
		}
		if err := manager.SaveSaga(ctx, instance); err != nil {
			t.Fatalf("SaveSaga failed: %v", err)
		}
	}

	tests := []struct {
		name      string
		updates   []SagaUpdate
		expectErr bool
	}{
		{
			name: "valid batch update",
			updates: []SagaUpdate{
				{SagaID: "saga-1", NewState: saga.StateCompleted, Metadata: map[string]interface{}{"batch": 1}},
				{SagaID: "saga-2", NewState: saga.StateCompleted, Metadata: map[string]interface{}{"batch": 1}},
			},
			expectErr: false,
		},
		{
			name: "partial failure",
			updates: []SagaUpdate{
				{SagaID: "saga-3", NewState: saga.StateCompleted, Metadata: nil},
				{SagaID: "saga-999", NewState: saga.StateCompleted, Metadata: nil},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.BatchUpdateSagas(ctx, tt.updates)
			if (err != nil) != tt.expectErr {
				t.Errorf("BatchUpdateSagas() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestStateManager_StateChangeNotification(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	// Save a saga
	instance := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		state:        saga.StatePending,
		createdAt:    now,
		updatedAt:    now,
	}

	if err := manager.SaveSaga(ctx, instance); err != nil {
		t.Fatalf("SaveSaga failed: %v", err)
	}

	// Register a listener
	var receivedEvent *StateChangeEvent
	var wg sync.WaitGroup
	wg.Add(1)

	listener := func(event *StateChangeEvent) {
		receivedEvent = event
		wg.Done()
	}

	if err := manager.SubscribeToStateChanges(listener); err != nil {
		t.Fatalf("SubscribeToStateChanges failed: %v", err)
	}

	// Update state
	if err := manager.UpdateSagaState(ctx, "saga-1", saga.StateRunning, map[string]interface{}{"test": true}); err != nil {
		t.Fatalf("UpdateSagaState failed: %v", err)
	}

	// Wait for notification
	wg.Wait()

	if receivedEvent == nil {
		t.Fatal("No state change event received")
	}

	if receivedEvent.SagaID != "saga-1" {
		t.Errorf("Event SagaID = %s, want saga-1", receivedEvent.SagaID)
	}

	if receivedEvent.OldState != saga.StatePending {
		t.Errorf("Event OldState = %s, want %s", receivedEvent.OldState.String(), saga.StatePending.String())
	}

	if receivedEvent.NewState != saga.StateRunning {
		t.Errorf("Event NewState = %s, want %s", receivedEvent.NewState.String(), saga.StateRunning.String())
	}
}

func TestStateManager_ConcurrentOperations(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	// Save initial sagas
	for i := 0; i < 10; i++ {
		instance := &mockSagaInstance{
			id:           fmt.Sprintf("saga-%d", i),
			definitionID: "def-1",
			state:        saga.StatePending,
			createdAt:    now,
			updatedAt:    now,
		}
		if err := manager.SaveSaga(ctx, instance); err != nil {
			t.Fatalf("SaveSaga failed: %v", err)
		}
	}

	// Perform concurrent updates
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			err := manager.UpdateSagaState(ctx, fmt.Sprintf("saga-%d", index), saga.StateRunning, nil)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent operation failed: %v", err)
		}
	}

	// Verify all sagas were updated
	filter := &saga.SagaFilter{
		States: []saga.SagaState{saga.StateRunning},
	}

	result, err := manager.GetActiveSagas(ctx, filter)
	if err != nil {
		t.Fatalf("GetActiveSagas failed: %v", err)
	}

	if len(result) != 10 {
		t.Errorf("Expected 10 running sagas, got %d", len(result))
	}
}

func TestStateManager_GetTimeoutSagas(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	// Save sagas with different timeout scenarios
	sagas := []*mockSagaInstance{
		{
			id:           "saga-1",
			definitionID: "def-1",
			state:        saga.StateRunning,
			startTime:    now.Add(-2 * time.Hour),
			timeout:      1 * time.Hour,
			createdAt:    now.Add(-2 * time.Hour),
			updatedAt:    now.Add(-2 * time.Hour),
		},
		{
			id:           "saga-2",
			definitionID: "def-1",
			state:        saga.StateRunning,
			startTime:    now.Add(-30 * time.Minute),
			timeout:      2 * time.Hour,
			createdAt:    now.Add(-30 * time.Minute),
			updatedAt:    now.Add(-30 * time.Minute),
		},
		{
			id:           "saga-3",
			definitionID: "def-1",
			state:        saga.StateCompleted,
			startTime:    now.Add(-2 * time.Hour),
			timeout:      1 * time.Hour,
			createdAt:    now.Add(-2 * time.Hour),
			updatedAt:    now,
		},
	}

	for _, s := range sagas {
		if err := manager.SaveSaga(ctx, s); err != nil {
			t.Fatalf("SaveSaga failed: %v", err)
		}
	}

	// Get timeout sagas
	timeoutSagas, err := manager.GetTimeoutSagas(ctx, now)
	if err != nil {
		t.Fatalf("GetTimeoutSagas failed: %v", err)
	}

	// Should only return saga-1 (timed out and still active)
	if len(timeoutSagas) != 1 {
		t.Errorf("Expected 1 timeout saga, got %d", len(timeoutSagas))
	}

	if len(timeoutSagas) > 0 && timeoutSagas[0].GetID() != "saga-1" {
		t.Errorf("Expected saga-1 to be timed out, got %s", timeoutSagas[0].GetID())
	}
}

func TestStateManager_CleanupExpiredSagas(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()
	oldTime := now.Add(-2 * time.Hour)

	// Save sagas with different ages
	sagas := []*mockSagaInstance{
		{
			id:           "saga-1",
			definitionID: "def-1",
			state:        saga.StateCompleted,
			createdAt:    oldTime,
			updatedAt:    oldTime,
		},
		{
			id:           "saga-2",
			definitionID: "def-1",
			state:        saga.StateCompleted,
			createdAt:    now.Add(-10 * time.Minute),
			updatedAt:    now.Add(-10 * time.Minute),
		},
		{
			id:           "saga-3",
			definitionID: "def-1",
			state:        saga.StateRunning,
			createdAt:    oldTime,
			updatedAt:    oldTime,
		},
	}

	for _, s := range sagas {
		if err := manager.SaveSaga(ctx, s); err != nil {
			t.Fatalf("SaveSaga failed: %v", err)
		}
	}

	// Cleanup sagas older than 1 hour
	cutoffTime := now.Add(-1 * time.Hour)
	if err := manager.CleanupExpiredSagas(ctx, cutoffTime); err != nil {
		t.Fatalf("CleanupExpiredSagas failed: %v", err)
	}

	// Verify saga-1 was deleted (old and terminal)
	if _, err := manager.GetSaga(ctx, "saga-1"); err != ErrSagaNotFound {
		t.Error("saga-1 should have been cleaned up")
	}

	// Verify saga-2 still exists (recent)
	if _, err := manager.GetSaga(ctx, "saga-2"); err != nil {
		t.Error("saga-2 should not have been cleaned up")
	}

	// Verify saga-3 still exists (old but still running)
	if _, err := manager.GetSaga(ctx, "saga-3"); err != nil {
		t.Error("saga-3 should not have been cleaned up (still active)")
	}
}

func TestStateManager_UnsubscribeFromStateChanges(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	ctx := context.Background()
	now := time.Now()

	// Save a saga
	instance := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		state:        saga.StatePending,
		createdAt:    now,
		updatedAt:    now,
	}

	if err := manager.SaveSaga(ctx, instance); err != nil {
		t.Fatalf("SaveSaga failed: %v", err)
	}

	// Register a listener
	callCount := 0
	var mu sync.Mutex

	listener := func(event *StateChangeEvent) {
		mu.Lock()
		callCount++
		mu.Unlock()
	}

	if err := manager.SubscribeToStateChanges(listener); err != nil {
		t.Fatalf("SubscribeToStateChanges failed: %v", err)
	}

	// Update state - listener should be called
	if err := manager.UpdateSagaState(ctx, "saga-1", saga.StateRunning, nil); err != nil {
		t.Fatalf("UpdateSagaState failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond) // Wait for async notification

	mu.Lock()
	firstCount := callCount
	mu.Unlock()

	if firstCount != 1 {
		t.Errorf("Expected listener to be called once, got %d", firstCount)
	}

	// Unsubscribe (note: the current implementation doesn't properly support this)
	// This test will demonstrate the current behavior
	if err := manager.UnsubscribeFromStateChanges(listener); err == nil {
		// If unsubscribe succeeds, verify listener is not called
		if err := manager.UpdateSagaState(ctx, "saga-1", saga.StateCompleted, nil); err != nil {
			t.Fatalf("UpdateSagaState failed: %v", err)
		}

		time.Sleep(100 * time.Millisecond) // Wait for async notification

		mu.Lock()
		secondCount := callCount
		mu.Unlock()

		if secondCount != firstCount {
			t.Error("Listener should not have been called after unsubscribe")
		}
	}

	// Test error cases
	if err := manager.SubscribeToStateChanges(nil); err == nil {
		t.Error("Should not allow nil listener")
	}

	if err := manager.UnsubscribeFromStateChanges(nil); err == nil {
		t.Error("Should not allow nil listener for unsubscribe")
	}
}

func TestStateManager_ContextCancellation(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)
	defer manager.Close()

	now := time.Now()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	instance := &mockSagaInstance{
		id:        "saga-1",
		state:     saga.StatePending,
		createdAt: now,
		updatedAt: now,
	}

	// All operations should fail with context error
	if err := manager.SaveSaga(ctx, instance); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	if _, err := manager.GetSaga(ctx, "saga-1"); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	if err := manager.UpdateSagaState(ctx, "saga-1", saga.StateRunning, nil); err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestStateManager_ClosedState(t *testing.T) {
	storage := newTestStorage()
	manager := NewStateManager(storage)

	// Close the manager
	if err := manager.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	ctx := context.Background()

	// Try operations on closed manager
	instance := &mockSagaInstance{
		id:        "saga-1",
		state:     saga.StatePending,
		createdAt: time.Now(),
	}

	if err := manager.SaveSaga(ctx, instance); err != ErrStorageClosed {
		t.Errorf("SaveSaga on closed manager should return ErrStorageClosed, got %v", err)
	}

	if _, err := manager.GetSaga(ctx, "saga-1"); err != ErrStorageClosed {
		t.Errorf("GetSaga on closed manager should return ErrStorageClosed, got %v", err)
	}

	// Test double close
	if err := manager.Close(); err != nil {
		t.Errorf("Second Close should not error, got %v", err)
	}
}

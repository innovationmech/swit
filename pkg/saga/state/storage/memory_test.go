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

package storage

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
)

// mockSagaInstance implements saga.SagaInstance for testing purposes.
type mockSagaInstance struct {
	id             string
	definitionID   string
	sagaState      saga.SagaState
	currentStep    int
	totalSteps     int
	startTime      time.Time
	endTime        time.Time
	result         interface{}
	err            *saga.SagaError
	completedSteps int
	createdAt      time.Time
	updatedAt      time.Time
	timeout        time.Duration
	metadata       map[string]interface{}
	traceID        string
}

func (m *mockSagaInstance) GetID() string                       { return m.id }
func (m *mockSagaInstance) GetDefinitionID() string             { return m.definitionID }
func (m *mockSagaInstance) GetState() saga.SagaState            { return m.sagaState }
func (m *mockSagaInstance) GetCurrentStep() int                 { return m.currentStep }
func (m *mockSagaInstance) GetStartTime() time.Time             { return m.startTime }
func (m *mockSagaInstance) GetEndTime() time.Time               { return m.endTime }
func (m *mockSagaInstance) GetResult() interface{}              { return m.result }
func (m *mockSagaInstance) GetError() *saga.SagaError           { return m.err }
func (m *mockSagaInstance) GetTotalSteps() int                  { return m.totalSteps }
func (m *mockSagaInstance) GetCompletedSteps() int              { return m.completedSteps }
func (m *mockSagaInstance) GetCreatedAt() time.Time             { return m.createdAt }
func (m *mockSagaInstance) GetUpdatedAt() time.Time             { return m.updatedAt }
func (m *mockSagaInstance) GetTimeout() time.Duration           { return m.timeout }
func (m *mockSagaInstance) GetMetadata() map[string]interface{} { return m.metadata }
func (m *mockSagaInstance) GetTraceID() string                  { return m.traceID }
func (m *mockSagaInstance) IsTerminal() bool                    { return m.sagaState.IsTerminal() }
func (m *mockSagaInstance) IsActive() bool                      { return m.sagaState.IsActive() }

// TestNewMemoryStateStorage tests the creation of a new memory storage instance.
func TestNewMemoryStateStorage(t *testing.T) {
	tests := []struct {
		name   string
		config *state.MemoryStorageConfig
	}{
		{
			name:   "with nil config",
			config: nil,
		},
		{
			name: "with custom config",
			config: &state.MemoryStorageConfig{
				InitialCapacity: 50,
				MaxCapacity:     1000,
				EnableMetrics:   true,
			},
		},
		{
			name: "with zero initial capacity",
			config: &state.MemoryStorageConfig{
				InitialCapacity: 0,
				MaxCapacity:     0,
				EnableMetrics:   false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var storage *MemoryStateStorage
			if tt.config == nil {
				storage = NewMemoryStateStorage()
			} else {
				storage = NewMemoryStateStorageWithConfig(tt.config)
			}

			if storage == nil {
				t.Fatal("expected non-nil storage")
			}

			if storage.sagas == nil {
				t.Error("expected non-nil sagas map")
			}

			if storage.steps == nil {
				t.Error("expected non-nil steps map")
			}

			if storage.config == nil {
				t.Error("expected non-nil config")
			}

			if storage.closed {
				t.Error("expected storage to not be closed on creation")
			}
		})
	}
}

// TestMemoryStateStorage_SaveSaga tests the SaveSaga method.
func TestMemoryStateStorage_SaveSaga(t *testing.T) {
	tests := []struct {
		name        string
		instance    saga.SagaInstance
		wantErr     error
		description string
	}{
		{
			name: "save valid saga",
			instance: &mockSagaInstance{
				id:           "saga-1",
				definitionID: "def-1",
				sagaState:    saga.StateRunning,
				currentStep:  1,
				totalSteps:   3,
				createdAt:    time.Now(),
				updatedAt:    time.Now(),
				timeout:      5 * time.Minute,
			},
			wantErr:     nil,
			description: "should successfully save a valid saga instance",
		},
		{
			name:        "save nil instance",
			instance:    nil,
			wantErr:     state.ErrInvalidSagaID,
			description: "should return error when instance is nil",
		},
		{
			name: "save instance with empty ID",
			instance: &mockSagaInstance{
				id:           "",
				definitionID: "def-1",
				sagaState:    saga.StateRunning,
			},
			wantErr:     state.ErrInvalidSagaID,
			description: "should return error when instance ID is empty",
		},
		{
			name: "overwrite existing saga",
			instance: &mockSagaInstance{
				id:           "saga-overwrite",
				definitionID: "def-1",
				sagaState:    saga.StateCompleted,
				currentStep:  3,
				totalSteps:   3,
				createdAt:    time.Now(),
				updatedAt:    time.Now(),
				timeout:      5 * time.Minute,
			},
			wantErr:     nil,
			description: "should successfully overwrite an existing saga",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStateStorage()
			ctx := context.Background()

			// For overwrite test, pre-save a saga
			if tt.name == "overwrite existing saga" {
				existing := &mockSagaInstance{
					id:           "saga-overwrite",
					definitionID: "def-1",
					sagaState:    saga.StateRunning,
					currentStep:  1,
					totalSteps:   3,
					createdAt:    time.Now(),
					updatedAt:    time.Now(),
				}
				if err := storage.SaveSaga(ctx, existing); err != nil {
					t.Fatalf("failed to pre-save saga: %v", err)
				}
			}

			err := storage.SaveSaga(ctx, tt.instance)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify the saga was saved
				if tt.instance != nil && tt.instance.GetID() != "" {
					_, err := storage.GetSaga(ctx, tt.instance.GetID())
					if err != nil {
						t.Errorf("failed to retrieve saved saga: %v", err)
					}
				}
			}
		})
	}
}

// TestMemoryStateStorage_SaveSaga_ContextCancellation tests SaveSaga with cancelled context.
func TestMemoryStateStorage_SaveSaga_ContextCancellation(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	instance := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
	}

	err := storage.SaveSaga(ctx, instance)
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestMemoryStateStorage_GetSaga tests the GetSaga method.
func TestMemoryStateStorage_GetSaga(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate storage with a saga
	testSaga := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
		currentStep:  1,
		totalSteps:   3,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
		timeout:      5 * time.Minute,
		metadata: map[string]interface{}{
			"key1": "value1",
		},
	}

	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("failed to save test saga: %v", err)
	}

	tests := []struct {
		name        string
		sagaID      string
		wantErr     error
		description string
	}{
		{
			name:        "get existing saga",
			sagaID:      "saga-1",
			wantErr:     nil,
			description: "should successfully retrieve an existing saga",
		},
		{
			name:        "get non-existent saga",
			sagaID:      "saga-nonexistent",
			wantErr:     state.ErrSagaNotFound,
			description: "should return ErrSagaNotFound for non-existent saga",
		},
		{
			name:        "get with empty ID",
			sagaID:      "",
			wantErr:     state.ErrInvalidSagaID,
			description: "should return ErrInvalidSagaID for empty ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance, err := storage.GetSaga(ctx, tt.sagaID)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if instance == nil {
					t.Error("expected non-nil instance")
				} else {
					if instance.GetID() != tt.sagaID {
						t.Errorf("expected ID %s, got %s", tt.sagaID, instance.GetID())
					}
				}
			}
		})
	}
}

// TestMemoryStateStorage_GetSaga_ContextCancellation tests GetSaga with cancelled context.
func TestMemoryStateStorage_GetSaga_ContextCancellation(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := storage.GetSaga(ctx, "saga-1")
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestMemoryStateStorage_DeleteSaga tests the DeleteSaga method.
func TestMemoryStateStorage_DeleteSaga(t *testing.T) {
	tests := []struct {
		name        string
		sagaID      string
		setupFunc   func(*MemoryStateStorage, context.Context) error
		wantErr     error
		description string
	}{
		{
			name:   "delete existing saga",
			sagaID: "saga-1",
			setupFunc: func(s *MemoryStateStorage, ctx context.Context) error {
				return s.SaveSaga(ctx, &mockSagaInstance{
					id:           "saga-1",
					definitionID: "def-1",
					sagaState:    saga.StateCompleted,
				})
			},
			wantErr:     nil,
			description: "should successfully delete an existing saga",
		},
		{
			name:        "delete non-existent saga",
			sagaID:      "saga-nonexistent",
			setupFunc:   nil,
			wantErr:     state.ErrSagaNotFound,
			description: "should return ErrSagaNotFound for non-existent saga",
		},
		{
			name:        "delete with empty ID",
			sagaID:      "",
			setupFunc:   nil,
			wantErr:     state.ErrInvalidSagaID,
			description: "should return ErrInvalidSagaID for empty ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryStateStorage()
			ctx := context.Background()

			if tt.setupFunc != nil {
				if err := tt.setupFunc(storage, ctx); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			err := storage.DeleteSaga(ctx, tt.sagaID)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify the saga was deleted
				_, err := storage.GetSaga(ctx, tt.sagaID)
				if !errors.Is(err, state.ErrSagaNotFound) {
					t.Errorf("expected saga to be deleted, but it still exists")
				}
			}
		})
	}
}

// TestMemoryStateStorage_DeleteSaga_ContextCancellation tests DeleteSaga with cancelled context.
func TestMemoryStateStorage_DeleteSaga_ContextCancellation(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := storage.DeleteSaga(ctx, "saga-1")
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestMemoryStateStorage_UpdateSagaState tests the UpdateSagaState method.
func TestMemoryStateStorage_UpdateSagaState(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate storage with a saga
	testSaga := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
		currentStep:  1,
		totalSteps:   3,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
	}

	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("failed to save test saga: %v", err)
	}

	tests := []struct {
		name        string
		sagaID      string
		newState    saga.SagaState
		metadata    map[string]interface{}
		wantErr     error
		description string
	}{
		{
			name:     "update state successfully",
			sagaID:   "saga-1",
			newState: saga.StateCompleted,
			metadata: map[string]interface{}{
				"updated": true,
			},
			wantErr:     nil,
			description: "should successfully update saga state",
		},
		{
			name:        "update non-existent saga",
			sagaID:      "saga-nonexistent",
			newState:    saga.StateCompleted,
			metadata:    nil,
			wantErr:     state.ErrSagaNotFound,
			description: "should return ErrSagaNotFound for non-existent saga",
		},
		{
			name:        "update with empty ID",
			sagaID:      "",
			newState:    saga.StateCompleted,
			metadata:    nil,
			wantErr:     state.ErrInvalidSagaID,
			description: "should return ErrInvalidSagaID for empty ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.UpdateSagaState(ctx, tt.sagaID, tt.newState, tt.metadata)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify the state was updated
				instance, err := storage.GetSaga(ctx, tt.sagaID)
				if err != nil {
					t.Fatalf("failed to retrieve updated saga: %v", err)
				}

				if instance.GetState() != tt.newState {
					t.Errorf("expected state %v, got %v", tt.newState, instance.GetState())
				}

				// Verify metadata was updated
				if tt.metadata != nil {
					for key, expectedValue := range tt.metadata {
						if actualValue := instance.GetMetadata()[key]; actualValue != expectedValue {
							t.Errorf("expected metadata[%s] = %v, got %v", key, expectedValue, actualValue)
						}
					}
				}
			}
		})
	}
}

// TestMemoryStateStorage_SaveStepState tests the SaveStepState method.
func TestMemoryStateStorage_SaveStepState(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate storage with a saga
	testSaga := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
	}

	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("failed to save test saga: %v", err)
	}

	tests := []struct {
		name        string
		sagaID      string
		stepState   *saga.StepState
		wantErr     error
		description string
	}{
		{
			name:   "save step state successfully",
			sagaID: "saga-1",
			stepState: &saga.StepState{
				ID:        "step-1",
				SagaID:    "saga-1",
				StepIndex: 0,
				Name:      "Step 1",
				State:     saga.StepStateCompleted,
			},
			wantErr:     nil,
			description: "should successfully save a step state",
		},
		{
			name:        "save nil step state",
			sagaID:      "saga-1",
			stepState:   nil,
			wantErr:     state.ErrInvalidState,
			description: "should return ErrInvalidState for nil step state",
		},
		{
			name:   "save step state for non-existent saga",
			sagaID: "saga-nonexistent",
			stepState: &saga.StepState{
				ID:        "step-1",
				SagaID:    "saga-nonexistent",
				StepIndex: 0,
				Name:      "Step 1",
				State:     saga.StepStateCompleted,
			},
			wantErr:     state.ErrSagaNotFound,
			description: "should return ErrSagaNotFound for non-existent saga",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.SaveStepState(ctx, tt.sagaID, tt.stepState)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify the step state was saved
				steps, err := storage.GetStepStates(ctx, tt.sagaID)
				if err != nil {
					t.Fatalf("failed to retrieve step states: %v", err)
				}

				found := false
				for _, step := range steps {
					if step.ID == tt.stepState.ID {
						found = true
						break
					}
				}

				if !found {
					t.Error("expected step state to be saved")
				}
			}
		})
	}
}

// TestMemoryStateStorage_GetStepStates tests the GetStepStates method.
func TestMemoryStateStorage_GetStepStates(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate storage with a saga and step states
	testSaga := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
	}

	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("failed to save test saga: %v", err)
	}

	step1 := &saga.StepState{
		ID:        "step-1",
		SagaID:    "saga-1",
		StepIndex: 0,
		Name:      "Step 1",
		State:     saga.StepStateCompleted,
	}

	step2 := &saga.StepState{
		ID:        "step-2",
		SagaID:    "saga-1",
		StepIndex: 1,
		Name:      "Step 2",
		State:     saga.StepStateRunning,
	}

	if err := storage.SaveStepState(ctx, "saga-1", step1); err != nil {
		t.Fatalf("failed to save step 1: %v", err)
	}

	if err := storage.SaveStepState(ctx, "saga-1", step2); err != nil {
		t.Fatalf("failed to save step 2: %v", err)
	}

	tests := []struct {
		name        string
		sagaID      string
		expectCount int
		wantErr     error
		description string
	}{
		{
			name:        "get step states for saga with steps",
			sagaID:      "saga-1",
			expectCount: 2,
			wantErr:     nil,
			description: "should retrieve all step states for the saga",
		},
		{
			name:        "get step states for non-existent saga",
			sagaID:      "saga-nonexistent",
			expectCount: 0,
			wantErr:     state.ErrSagaNotFound,
			description: "should return ErrSagaNotFound for non-existent saga",
		},
		{
			name:        "get step states with empty ID",
			sagaID:      "",
			expectCount: 0,
			wantErr:     state.ErrInvalidSagaID,
			description: "should return ErrInvalidSagaID for empty ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := storage.GetStepStates(ctx, tt.sagaID)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if len(steps) != tt.expectCount {
					t.Errorf("expected %d step states, got %d", tt.expectCount, len(steps))
				}
			}
		})
	}
}

// TestMemoryStateStorage_GetActiveSagas tests the GetActiveSagas method.
func TestMemoryStateStorage_GetActiveSagas(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate storage with multiple sagas
	sagas := []*mockSagaInstance{
		{
			id:           "saga-1",
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
		},
		{
			id:           "saga-2",
			definitionID: "def-1",
			sagaState:    saga.StateCompleted,
			createdAt:    time.Now(),
		},
		{
			id:           "saga-3",
			definitionID: "def-2",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
		},
	}

	for _, s := range sagas {
		if err := storage.SaveSaga(ctx, s); err != nil {
			t.Fatalf("failed to save saga %s: %v", s.id, err)
		}
	}

	tests := []struct {
		name        string
		filter      *saga.SagaFilter
		expectCount int
		description string
	}{
		{
			name:        "get all sagas without filter",
			filter:      nil,
			expectCount: 3,
			description: "should retrieve all sagas when no filter is provided",
		},
		{
			name: "filter by state",
			filter: &saga.SagaFilter{
				States: []saga.SagaState{saga.StateRunning},
			},
			expectCount: 2,
			description: "should retrieve only running sagas",
		},
		{
			name: "filter by definition ID",
			filter: &saga.SagaFilter{
				DefinitionIDs: []string{"def-1"},
			},
			expectCount: 2,
			description: "should retrieve sagas with matching definition ID",
		},
		{
			name: "filter by state and definition ID",
			filter: &saga.SagaFilter{
				States:        []saga.SagaState{saga.StateRunning},
				DefinitionIDs: []string{"def-1"},
			},
			expectCount: 1,
			description: "should retrieve sagas matching both state and definition ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instances, err := storage.GetActiveSagas(ctx, tt.filter)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(instances) != tt.expectCount {
				t.Errorf("expected %d sagas, got %d", tt.expectCount, len(instances))
			}
		})
	}
}

// TestMemoryStateStorage_GetTimeoutSagas tests the GetTimeoutSagas method.
func TestMemoryStateStorage_GetTimeoutSagas(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-10 * time.Minute)

	// Pre-populate storage with sagas
	sagas := []*mockSagaInstance{
		{
			id:           "saga-timeout-1",
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			startTime:    past,
			timeout:      5 * time.Minute,
		},
		{
			id:           "saga-timeout-2",
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			startTime:    now,
			timeout:      5 * time.Minute,
		},
		{
			id:           "saga-completed",
			definitionID: "def-1",
			sagaState:    saga.StateCompleted,
			startTime:    past,
			timeout:      5 * time.Minute,
		},
	}

	for _, s := range sagas {
		if err := storage.SaveSaga(ctx, s); err != nil {
			t.Fatalf("failed to save saga %s: %v", s.id, err)
		}
	}

	tests := []struct {
		name        string
		before      time.Time
		expectCount int
		description string
	}{
		{
			name:        "get timed out sagas",
			before:      now,
			expectCount: 1,
			description: "should retrieve sagas that have exceeded their timeout",
		},
		{
			name:        "get timed out sagas with earlier cutoff",
			before:      past.Add(1 * time.Minute),
			expectCount: 0,
			description: "should retrieve no sagas with earlier cutoff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instances, err := storage.GetTimeoutSagas(ctx, tt.before)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(instances) != tt.expectCount {
				t.Errorf("expected %d timed out sagas, got %d", tt.expectCount, len(instances))
			}
		})
	}
}

// TestMemoryStateStorage_CleanupExpiredSagas tests the CleanupExpiredSagas method.
func TestMemoryStateStorage_CleanupExpiredSagas(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	now := time.Now()
	old := now.Add(-2 * time.Hour)

	// Pre-populate storage with sagas
	sagas := []*mockSagaInstance{
		{
			id:           "saga-old-completed",
			definitionID: "def-1",
			sagaState:    saga.StateCompleted,
			createdAt:    old,
			updatedAt:    old,
		},
		{
			id:           "saga-old-running",
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    old,
			updatedAt:    old,
		},
		{
			id:           "saga-recent-completed",
			definitionID: "def-1",
			sagaState:    saga.StateCompleted,
			createdAt:    now,
			updatedAt:    now,
		},
	}

	for _, s := range sagas {
		if err := storage.SaveSaga(ctx, s); err != nil {
			t.Fatalf("failed to save saga %s: %v", s.id, err)
		}
	}

	cutoff := now.Add(-1 * time.Hour)
	err := storage.CleanupExpiredSagas(ctx, cutoff)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify that only the old completed saga was removed
	_, err = storage.GetSaga(ctx, "saga-old-completed")
	if !errors.Is(err, state.ErrSagaNotFound) {
		t.Error("expected old completed saga to be cleaned up")
	}

	// Verify that the old running saga was NOT removed
	_, err = storage.GetSaga(ctx, "saga-old-running")
	if err != nil {
		t.Error("old running saga should not be cleaned up")
	}

	// Verify that the recent completed saga was NOT removed
	_, err = storage.GetSaga(ctx, "saga-recent-completed")
	if err != nil {
		t.Error("recent completed saga should not be cleaned up")
	}
}

// TestMemoryStateStorage_Close tests the Close method.
func TestMemoryStateStorage_Close(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate storage with a saga
	testSaga := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
	}

	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("failed to save test saga: %v", err)
	}

	// Close the storage
	err := storage.Close()
	if err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}

	// Verify that operations fail after close
	err = storage.SaveSaga(ctx, testSaga)
	if !errors.Is(err, state.ErrStorageClosed) {
		t.Errorf("expected ErrStorageClosed after close, got %v", err)
	}

	_, err = storage.GetSaga(ctx, "saga-1")
	if !errors.Is(err, state.ErrStorageClosed) {
		t.Errorf("expected ErrStorageClosed after close, got %v", err)
	}

	// Verify that closing again is idempotent
	err = storage.Close()
	if err != nil {
		t.Errorf("unexpected error on second close: %v", err)
	}
}

// TestMemoryStateStorage_Concurrency tests concurrent operations on the storage.
func TestMemoryStateStorage_Concurrency(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Number of concurrent goroutines
	numGoroutines := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // Save, Get, Update

	// Concurrent saves
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			instance := &mockSagaInstance{
				id:           "saga-" + string(rune(id)),
				definitionID: "def-1",
				sagaState:    saga.StateRunning,
				createdAt:    time.Now(),
				updatedAt:    time.Now(),
			}
			_ = storage.SaveSaga(ctx, instance)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			_, _ = storage.GetSaga(ctx, "saga-"+string(rune(id)))
		}(i)
	}

	// Concurrent updates
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			_ = storage.UpdateSagaState(ctx, "saga-"+string(rune(id)), saga.StateCompleted, nil)
		}(i)
	}

	wg.Wait()

	// Test should complete without data races (checked by -race flag)
}

// TestMemorySagaInstance_Methods tests all methods of memorySagaInstance.
func TestMemorySagaInstance_Methods(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now

	// Create a comprehensive test saga
	testSaga := &mockSagaInstance{
		id:             "saga-1",
		definitionID:   "def-1",
		sagaState:      saga.StateCompleted,
		currentStep:    3,
		totalSteps:     5,
		startTime:      startTime,
		endTime:        endTime,
		result:         "test-result",
		err:            &saga.SagaError{Code: "TEST_ERROR", Message: "test error"},
		completedSteps: 3,
		createdAt:      now.Add(-2 * time.Hour),
		updatedAt:      now.Add(-30 * time.Minute),
		timeout:        5 * time.Minute,
		metadata: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
		traceID: "trace-123",
	}

	// Save the saga
	err := storage.SaveSaga(ctx, testSaga)
	if err != nil {
		t.Fatalf("failed to save saga: %v", err)
	}

	// Retrieve it to get the memorySagaInstance
	instance, err := storage.GetSaga(ctx, "saga-1")
	if err != nil {
		t.Fatalf("failed to get saga: %v", err)
	}

	// Test all getter methods
	if instance.GetID() != "saga-1" {
		t.Errorf("expected ID 'saga-1', got %s", instance.GetID())
	}

	if instance.GetDefinitionID() != "def-1" {
		t.Errorf("expected DefinitionID 'def-1', got %s", instance.GetDefinitionID())
	}

	if instance.GetState() != saga.StateCompleted {
		t.Errorf("expected State StateCompleted, got %v", instance.GetState())
	}

	if instance.GetCurrentStep() != 3 {
		t.Errorf("expected CurrentStep 3, got %d", instance.GetCurrentStep())
	}

	if instance.GetTotalSteps() != 5 {
		t.Errorf("expected TotalSteps 5, got %d", instance.GetTotalSteps())
	}

	if !instance.GetStartTime().Equal(startTime) {
		t.Errorf("expected StartTime %v, got %v", startTime, instance.GetStartTime())
	}

	if !instance.GetEndTime().Equal(endTime) {
		t.Errorf("expected EndTime %v, got %v", endTime, instance.GetEndTime())
	}

	if instance.GetResult() != "test-result" {
		t.Errorf("expected Result 'test-result', got %v", instance.GetResult())
	}

	if instance.GetError() == nil {
		t.Error("expected non-nil Error")
	} else if instance.GetError().Code != "TEST_ERROR" {
		t.Errorf("expected Error.Code 'TEST_ERROR', got %s", instance.GetError().Code)
	}

	if instance.GetCompletedSteps() != 0 { // Note: CompletedSteps is calculated from StepStates
		t.Errorf("expected CompletedSteps 0 (no step states saved), got %d", instance.GetCompletedSteps())
	}

	if instance.GetCreatedAt().IsZero() {
		t.Error("expected non-zero CreatedAt")
	}

	if instance.GetUpdatedAt().IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}

	if instance.GetTimeout() != 5*time.Minute {
		t.Errorf("expected Timeout 5m, got %v", instance.GetTimeout())
	}

	metadata := instance.GetMetadata()
	if metadata == nil {
		t.Error("expected non-nil Metadata")
	} else {
		if metadata["key1"] != "value1" {
			t.Errorf("expected Metadata['key1'] = 'value1', got %v", metadata["key1"])
		}
	}

	if instance.GetTraceID() != "trace-123" {
		t.Errorf("expected TraceID 'trace-123', got %s", instance.GetTraceID())
	}

	if !instance.IsTerminal() {
		t.Error("expected IsTerminal to be true for StateCompleted")
	}

	if instance.IsActive() {
		t.Error("expected IsActive to be false for StateCompleted")
	}
}

// TestMemorySagaInstance_CompletedStepsCalculation tests GetCompletedSteps calculation from step states.
func TestMemorySagaInstance_CompletedStepsCalculation(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Create a test saga
	testSaga := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
		totalSteps:   3,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
	}

	// Save the saga
	err := storage.SaveSaga(ctx, testSaga)
	if err != nil {
		t.Fatalf("failed to save saga: %v", err)
	}

	// Add some step states
	steps := []*saga.StepState{
		{
			ID:        "step-1",
			SagaID:    "saga-1",
			StepIndex: 0,
			State:     saga.StepStateCompleted,
		},
		{
			ID:        "step-2",
			SagaID:    "saga-1",
			StepIndex: 1,
			State:     saga.StepStateCompleted,
		},
		{
			ID:        "step-3",
			SagaID:    "saga-1",
			StepIndex: 2,
			State:     saga.StepStateRunning,
		},
	}

	for _, step := range steps {
		if err := storage.SaveStepState(ctx, "saga-1", step); err != nil {
			t.Fatalf("failed to save step state: %v", err)
		}
	}

	// Need to update the saga instance to include the step states
	// First retrieve step states
	stepStates, err := storage.GetStepStates(ctx, "saga-1")
	if err != nil {
		t.Fatalf("failed to get step states: %v", err)
	}

	// Create a new saga instance with step states
	storage.mu.Lock()
	if data, exists := storage.sagas["saga-1"]; exists {
		data.StepStates = stepStates
	}
	storage.mu.Unlock()

	// Retrieve it again
	instance, err := storage.GetSaga(ctx, "saga-1")
	if err != nil {
		t.Fatalf("failed to get saga: %v", err)
	}

	// Now GetCompletedSteps should return 2
	completedSteps := instance.GetCompletedSteps()
	if completedSteps != 2 {
		t.Errorf("expected 2 completed steps, got %d", completedSteps)
	}
}

// TestMemorySagaInstance_ActiveState tests IsActive for active states.
func TestMemorySagaInstance_ActiveState(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	activeStates := []saga.SagaState{
		saga.StateRunning,
		saga.StateStepCompleted,
		saga.StateCompensating,
	}

	for _, state := range activeStates {
		t.Run(state.String(), func(t *testing.T) {
			sagaID := "saga-" + state.String()
			testSaga := &mockSagaInstance{
				id:           sagaID,
				definitionID: "def-1",
				sagaState:    state,
				createdAt:    time.Now(),
				updatedAt:    time.Now(),
			}

			err := storage.SaveSaga(ctx, testSaga)
			if err != nil {
				t.Fatalf("failed to save saga: %v", err)
			}

			instance, err := storage.GetSaga(ctx, sagaID)
			if err != nil {
				t.Fatalf("failed to get saga: %v", err)
			}

			if !instance.IsActive() {
				t.Errorf("expected IsActive to be true for state %s", state.String())
			}

			if instance.IsTerminal() {
				t.Errorf("expected IsTerminal to be false for state %s", state.String())
			}
		})
	}
}

// TestMemorySagaInstance_TerminalState tests IsTerminal for terminal states.
func TestMemorySagaInstance_TerminalState(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	terminalStates := []saga.SagaState{
		saga.StateCompleted,
		saga.StateCompensated,
		saga.StateFailed,
		saga.StateCancelled,
		saga.StateTimedOut,
	}

	for _, state := range terminalStates {
		t.Run(state.String(), func(t *testing.T) {
			sagaID := "saga-" + state.String()
			testSaga := &mockSagaInstance{
				id:           sagaID,
				definitionID: "def-1",
				sagaState:    state,
				createdAt:    time.Now(),
				updatedAt:    time.Now(),
			}

			err := storage.SaveSaga(ctx, testSaga)
			if err != nil {
				t.Fatalf("failed to save saga: %v", err)
			}

			instance, err := storage.GetSaga(ctx, sagaID)
			if err != nil {
				t.Fatalf("failed to get saga: %v", err)
			}

			if !instance.IsTerminal() {
				t.Errorf("expected IsTerminal to be true for state %s", state.String())
			}

			if instance.IsActive() {
				t.Errorf("expected IsActive to be false for state %s", state.String())
			}
		})
	}
}

// TestMemorySagaInstance_EndTimeForTimedOut tests GetEndTime for timed out sagas.
func TestMemorySagaInstance_EndTimeForTimedOut(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	timedOutTime := time.Now()
	testSaga := &mockSagaInstance{
		id:           "saga-timeout",
		definitionID: "def-1",
		sagaState:    saga.StateTimedOut,
		endTime:      timedOutTime,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
	}

	err := storage.SaveSaga(ctx, testSaga)
	if err != nil {
		t.Fatalf("failed to save saga: %v", err)
	}

	instance, err := storage.GetSaga(ctx, "saga-timeout")
	if err != nil {
		t.Fatalf("failed to get saga: %v", err)
	}

	endTime := instance.GetEndTime()
	if endTime.IsZero() {
		t.Error("expected non-zero EndTime for timed out saga")
	}
}

// TestMemoryStateStorage_StepStateUpdate tests updating an existing step state.
func TestMemoryStateStorage_StepStateUpdate(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Create a test saga
	testSaga := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
	}

	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("failed to save saga: %v", err)
	}

	// Add a step state
	step1 := &saga.StepState{
		ID:        "step-1",
		SagaID:    "saga-1",
		StepIndex: 0,
		Name:      "Step 1",
		State:     saga.StepStateRunning,
	}

	if err := storage.SaveStepState(ctx, "saga-1", step1); err != nil {
		t.Fatalf("failed to save initial step state: %v", err)
	}

	// Update the step state
	step1Updated := &saga.StepState{
		ID:        "step-1",
		SagaID:    "saga-1",
		StepIndex: 0,
		Name:      "Step 1 Updated",
		State:     saga.StepStateCompleted,
	}

	if err := storage.SaveStepState(ctx, "saga-1", step1Updated); err != nil {
		t.Fatalf("failed to update step state: %v", err)
	}

	// Verify the update
	steps, err := storage.GetStepStates(ctx, "saga-1")
	if err != nil {
		t.Fatalf("failed to get step states: %v", err)
	}

	if len(steps) != 1 {
		t.Errorf("expected 1 step state, got %d", len(steps))
	}

	if steps[0].State != saga.StepStateCompleted {
		t.Errorf("expected step state to be updated to Completed, got %v", steps[0].State)
	}

	if steps[0].Name != "Step 1 Updated" {
		t.Errorf("expected step name to be updated, got %s", steps[0].Name)
	}
}

// TestMemoryStateStorage_FilterByMetadata tests filtering sagas by metadata.
func TestMemoryStateStorage_FilterByMetadata(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Create sagas with different metadata
	sagas := []*mockSagaInstance{
		{
			id:           "saga-1",
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			metadata: map[string]interface{}{
				"environment": "production",
				"priority":    "high",
			},
		},
		{
			id:           "saga-2",
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			metadata: map[string]interface{}{
				"environment": "staging",
				"priority":    "low",
			},
		},
		{
			id:           "saga-3",
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			metadata: map[string]interface{}{
				"environment": "production",
				"priority":    "low",
			},
		},
	}

	for _, s := range sagas {
		if err := storage.SaveSaga(ctx, s); err != nil {
			t.Fatalf("failed to save saga %s: %v", s.id, err)
		}
	}

	// Filter by environment
	filter := &saga.SagaFilter{
		Metadata: map[string]interface{}{
			"environment": "production",
		},
	}

	instances, err := storage.GetActiveSagas(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get active sagas: %v", err)
	}

	if len(instances) != 2 {
		t.Errorf("expected 2 sagas with environment=production, got %d", len(instances))
	}

	// Filter by environment and priority
	filter = &saga.SagaFilter{
		Metadata: map[string]interface{}{
			"environment": "production",
			"priority":    "high",
		},
	}

	instances, err = storage.GetActiveSagas(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get active sagas: %v", err)
	}

	if len(instances) != 1 {
		t.Errorf("expected 1 saga with environment=production and priority=high, got %d", len(instances))
	}
}

// TestMemoryStateStorage_FilterByTimeRange tests filtering sagas by creation time.
func TestMemoryStateStorage_FilterByTimeRange(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	// Create sagas with different creation times
	sagas := []*mockSagaInstance{
		{
			id:           "saga-old",
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    lastWeek,
			updatedAt:    lastWeek,
		},
		{
			id:           "saga-medium",
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    yesterday,
			updatedAt:    yesterday,
		},
		{
			id:           "saga-recent",
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    now,
			updatedAt:    now,
		},
	}

	for _, s := range sagas {
		if err := storage.SaveSaga(ctx, s); err != nil {
			t.Fatalf("failed to save saga %s: %v", s.id, err)
		}
	}

	// Filter by created after
	twoDaysAgo := now.Add(-48 * time.Hour)
	filter := &saga.SagaFilter{
		CreatedAfter: &twoDaysAgo,
	}

	instances, err := storage.GetActiveSagas(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get active sagas: %v", err)
	}

	if len(instances) != 2 {
		t.Errorf("expected 2 sagas created after two days ago, got %d", len(instances))
	}

	// Filter by created before (should include sagas created before yesterday, i.e., lastWeek)
	// Note: Yesterday's saga might also be included depending on exact timing
	theDayBeforeYesterday := yesterday.Add(-1 * time.Hour)
	filter = &saga.SagaFilter{
		CreatedBefore: &theDayBeforeYesterday,
	}

	instances, err = storage.GetActiveSagas(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get active sagas: %v", err)
	}

	// Should only get the lastWeek saga
	if len(instances) != 1 {
		t.Errorf("expected 1 saga created before the day before yesterday, got %d", len(instances))
	}
}

// TestMemoryStateStorage_MaxCapacity tests the max capacity limit.
func TestMemoryStateStorage_MaxCapacity(t *testing.T) {
	config := &state.MemoryStorageConfig{
		InitialCapacity: 10,
		MaxCapacity:     2,
		EnableMetrics:   false,
	}

	storage := NewMemoryStateStorageWithConfig(config)
	ctx := context.Background()

	// Save up to max capacity
	for i := 0; i < 2; i++ {
		instance := &mockSagaInstance{
			id:           "saga-" + string(rune(i)),
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			createdAt:    time.Now(),
			updatedAt:    time.Now(),
		}
		err := storage.SaveSaga(ctx, instance)
		if err != nil {
			t.Fatalf("failed to save saga %d: %v", i, err)
		}
	}

	// Try to save one more beyond capacity
	overflowSaga := &mockSagaInstance{
		id:           "saga-overflow",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
	}

	err := storage.SaveSaga(ctx, overflowSaga)
	if !errors.Is(err, state.ErrStorageFailure) {
		t.Errorf("expected ErrStorageFailure when exceeding max capacity, got %v", err)
	}

	// Verify that updating existing saga is still allowed
	existingSaga := &mockSagaInstance{
		id:           "saga-" + string(rune(0)),
		definitionID: "def-1",
		sagaState:    saga.StateCompleted,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
	}
	err = storage.SaveSaga(ctx, existingSaga)
	if err != nil {
		t.Errorf("updating existing saga should be allowed: %v", err)
	}
}

// TestMemoryStateStorage_UpdateSagaState_ContextCancellation tests UpdateSagaState with cancelled context.
func TestMemoryStateStorage_UpdateSagaState_ContextCancellation(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := storage.UpdateSagaState(ctx, "saga-1", saga.StateCompleted, nil)
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestMemoryStateStorage_GetActiveSagas_ContextCancellation tests GetActiveSagas with cancelled context.
func TestMemoryStateStorage_GetActiveSagas_ContextCancellation(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := storage.GetActiveSagas(ctx, nil)
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestMemoryStateStorage_GetTimeoutSagas_ContextCancellation tests GetTimeoutSagas with cancelled context.
func TestMemoryStateStorage_GetTimeoutSagas_ContextCancellation(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := storage.GetTimeoutSagas(ctx, time.Now())
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestMemoryStateStorage_CleanupExpiredSagas_ContextCancellation tests CleanupExpiredSagas with cancelled context.
func TestMemoryStateStorage_CleanupExpiredSagas_ContextCancellation(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := storage.CleanupExpiredSagas(ctx, time.Now())
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestMemoryStateStorage_SaveStepState_ContextCancellation tests SaveStepState with cancelled context.
func TestMemoryStateStorage_SaveStepState_ContextCancellation(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	step := &saga.StepState{
		ID:        "step-1",
		SagaID:    "saga-1",
		StepIndex: 0,
		Name:      "Step 1",
		State:     saga.StepStateCompleted,
	}

	err := storage.SaveStepState(ctx, "saga-1", step)
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestMemoryStateStorage_GetStepStates_ContextCancellation tests GetStepStates with cancelled context.
func TestMemoryStateStorage_GetStepStates_ContextCancellation(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := storage.GetStepStates(ctx, "saga-1")
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestMemoryStateStorage_ClosedOperations tests operations on closed storage.
func TestMemoryStateStorage_ClosedOperations(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Close the storage
	if err := storage.Close(); err != nil {
		t.Fatalf("failed to close storage: %v", err)
	}

	// Test all operations return ErrStorageClosed
	operations := []struct {
		name string
		op   func() error
	}{
		{
			name: "UpdateSagaState",
			op: func() error {
				return storage.UpdateSagaState(ctx, "saga-1", saga.StateCompleted, nil)
			},
		},
		{
			name: "GetActiveSagas",
			op: func() error {
				_, err := storage.GetActiveSagas(ctx, nil)
				return err
			},
		},
		{
			name: "GetTimeoutSagas",
			op: func() error {
				_, err := storage.GetTimeoutSagas(ctx, time.Now())
				return err
			},
		},
		{
			name: "CleanupExpiredSagas",
			op: func() error {
				return storage.CleanupExpiredSagas(ctx, time.Now())
			},
		},
		{
			name: "SaveStepState",
			op: func() error {
				step := &saga.StepState{
					ID:        "step-1",
					SagaID:    "saga-1",
					StepIndex: 0,
					Name:      "Step 1",
					State:     saga.StepStateCompleted,
				}
				return storage.SaveStepState(ctx, "saga-1", step)
			},
		},
		{
			name: "GetStepStates",
			op: func() error {
				_, err := storage.GetStepStates(ctx, "saga-1")
				return err
			},
		},
	}

	for _, test := range operations {
		t.Run(test.name, func(t *testing.T) {
			err := test.op()
			if !errors.Is(err, state.ErrStorageClosed) {
				t.Errorf("expected ErrStorageClosed for %s after close, got %v", test.name, err)
			}
		})
	}
}

// TestMemoryStateStorage_UpdateSagaState_WithNilMetadata tests UpdateSagaState with nil metadata.
func TestMemoryStateStorage_UpdateSagaState_WithNilMetadata(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Pre-populate storage with a saga that has no metadata
	testSaga := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
		currentStep:  1,
		totalSteps:   3,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
		metadata:     nil, // No initial metadata
	}

	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("failed to save test saga: %v", err)
	}

	// Update with new metadata
	newMetadata := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	err := storage.UpdateSagaState(ctx, "saga-1", saga.StateCompleted, newMetadata)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the metadata was added
	instance, err := storage.GetSaga(ctx, "saga-1")
	if err != nil {
		t.Fatalf("failed to retrieve updated saga: %v", err)
	}

	if instance.GetMetadata() == nil {
		t.Error("expected metadata to be initialized")
	} else {
		if instance.GetMetadata()["key1"] != "value1" {
			t.Errorf("expected metadata[key1] = 'value1', got %v", instance.GetMetadata()["key1"])
		}
		if instance.GetMetadata()["key2"] != 123 {
			t.Errorf("expected metadata[key2] = 123, got %v", instance.GetMetadata()["key2"])
		}
	}
}

// TestMemoryStateStorage_GetStepStates_EmptySteps tests GetStepStates for saga with no steps.
func TestMemoryStateStorage_GetStepStates_EmptySteps(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Create a saga without step states
	testSaga := &mockSagaInstance{
		id:           "saga-empty",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
		createdAt:    time.Now(),
		updatedAt:    time.Now(),
	}

	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("failed to save test saga: %v", err)
	}

	steps, err := storage.GetStepStates(ctx, "saga-empty")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(steps))
	}
}

// TestMemoryStateStorage_GetTimeoutSagas_WithNonActiveStates tests GetTimeoutSagas filtering non-active sagas.
func TestMemoryStateStorage_GetTimeoutSagas_WithNonActiveStates(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-10 * time.Minute)

	// Create sagas with various states
	sagas := []*mockSagaInstance{
		{
			id:           "saga-pending",
			definitionID: "def-1",
			sagaState:    saga.StatePending,
			startTime:    past,
			timeout:      5 * time.Minute,
			createdAt:    past,
			updatedAt:    past,
		},
		{
			id:           "saga-completed",
			definitionID: "def-1",
			sagaState:    saga.StateCompleted,
			startTime:    past,
			timeout:      5 * time.Minute,
			createdAt:    past,
			updatedAt:    past,
		},
		{
			id:           "saga-running-timedout",
			definitionID: "def-1",
			sagaState:    saga.StateRunning,
			startTime:    past,
			timeout:      5 * time.Minute,
			createdAt:    past,
			updatedAt:    past,
		},
	}

	for _, s := range sagas {
		if err := storage.SaveSaga(ctx, s); err != nil {
			t.Fatalf("failed to save saga %s: %v", s.id, err)
		}
	}

	// Only active sagas that timed out should be returned
	instances, err := storage.GetTimeoutSagas(ctx, now)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should only get the running saga that timed out
	if len(instances) != 1 {
		t.Errorf("expected 1 timed out saga, got %d", len(instances))
	}

	if len(instances) > 0 && instances[0].GetID() != "saga-running-timedout" {
		t.Errorf("expected saga-running-timedout, got %s", instances[0].GetID())
	}
}

// TestMemoryStateStorage_GetTimeoutSagas_WithoutStartTime tests GetTimeoutSagas with sagas without start time.
func TestMemoryStateStorage_GetTimeoutSagas_WithoutStartTime(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	now := time.Now()

	// Create saga without start time
	testSaga := &mockSagaInstance{
		id:           "saga-no-start",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
		startTime:    time.Time{}, // Zero time
		timeout:      5 * time.Minute,
		createdAt:    now,
		updatedAt:    now,
	}

	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("failed to save test saga: %v", err)
	}

	// Should not include sagas without start time
	instances, err := storage.GetTimeoutSagas(ctx, now)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(instances) != 0 {
		t.Errorf("expected 0 timed out sagas (no start time), got %d", len(instances))
	}
}

// TestMemoryStateStorage_FilterByMetadata_NoMatch tests filtering with no matching metadata.
func TestMemoryStateStorage_FilterByMetadata_NoMatch(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Create saga with metadata
	testSaga := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
		createdAt:    time.Now(),
		metadata: map[string]interface{}{
			"environment": "production",
		},
	}

	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("failed to save saga: %v", err)
	}

	// Filter by non-matching metadata
	filter := &saga.SagaFilter{
		Metadata: map[string]interface{}{
			"environment": "staging", // Different value
		},
	}

	instances, err := storage.GetActiveSagas(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get active sagas: %v", err)
	}

	if len(instances) != 0 {
		t.Errorf("expected 0 sagas with non-matching metadata, got %d", len(instances))
	}
}

// TestMemoryStateStorage_FilterByMetadata_NoMetadata tests filtering against sagas without metadata.
func TestMemoryStateStorage_FilterByMetadata_NoMetadata(t *testing.T) {
	storage := NewMemoryStateStorage()
	ctx := context.Background()

	// Create saga without metadata
	testSaga := &mockSagaInstance{
		id:           "saga-1",
		definitionID: "def-1",
		sagaState:    saga.StateRunning,
		createdAt:    time.Now(),
		metadata:     nil,
	}

	if err := storage.SaveSaga(ctx, testSaga); err != nil {
		t.Fatalf("failed to save saga: %v", err)
	}

	// Filter by metadata
	filter := &saga.SagaFilter{
		Metadata: map[string]interface{}{
			"environment": "production",
		},
	}

	instances, err := storage.GetActiveSagas(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get active sagas: %v", err)
	}

	if len(instances) != 0 {
		t.Errorf("expected 0 sagas when filtering metadata against saga with no metadata, got %d", len(instances))
	}
}

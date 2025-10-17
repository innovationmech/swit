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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

// mockStateStorage is a simple mock implementation of saga.StateStorage for testing.
type mockStateStorage struct {
	mu     sync.RWMutex
	sagas  map[string]saga.SagaInstance
	steps  map[string][]*saga.StepState
	closed bool
}

func newMockStateStorage() *mockStateStorage {
	return &mockStateStorage{
		sagas: make(map[string]saga.SagaInstance),
		steps: make(map[string][]*saga.StepState),
	}
}

func (m *mockStateStorage) SaveSaga(ctx context.Context, instance saga.SagaInstance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("storage closed")
	}
	m.sagas[instance.GetID()] = instance
	return nil
}

func (m *mockStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return nil, errors.New("storage closed")
	}
	instance, exists := m.sagas[sagaID]
	if !exists {
		return nil, errors.New("saga not found")
	}
	return instance, nil
}

func (m *mockStateStorage) UpdateSagaState(ctx context.Context, sagaID string, state saga.SagaState, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("storage closed")
	}
	sagaInst, exists := m.sagas[sagaID]
	if !exists {
		return errors.New("saga not found")
	}
	// Update the saga state if it's a mockSagaInstance
	if mockInst, ok := sagaInst.(*mockSagaInstance); ok {
		mockInst.state = state
		mockInst.updatedAt = time.Now()
		if metadata != nil {
			if mockInst.metadata == nil {
				mockInst.metadata = make(map[string]interface{})
			}
			for k, v := range metadata {
				mockInst.metadata[k] = v
			}
		}
	}
	return nil
}

func (m *mockStateStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("storage closed")
	}
	delete(m.sagas, sagaID)
	delete(m.steps, sagaID)
	return nil
}

func (m *mockStateStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return nil, errors.New("storage closed")
	}
	var result []saga.SagaInstance
	for _, instance := range m.sagas {
		result = append(result, instance)
	}
	return result, nil
}

func (m *mockStateStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return nil, errors.New("storage closed")
	}
	return []saga.SagaInstance{}, nil
}

func (m *mockStateStorage) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("storage closed")
	}
	m.steps[sagaID] = append(m.steps[sagaID], step)
	return nil
}

func (m *mockStateStorage) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return nil, errors.New("storage closed")
	}
	return m.steps[sagaID], nil
}

func (m *mockStateStorage) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("storage closed")
	}
	return nil
}

// mockCoordinator is a mock implementation of saga.SagaCoordinator for testing.
type mockCoordinator struct {
	startSagaFunc       func(ctx context.Context, definition saga.SagaDefinition, initialData interface{}) (saga.SagaInstance, error)
	getSagaInstanceFunc func(sagaID string) (saga.SagaInstance, error)
	cancelSagaFunc      func(ctx context.Context, sagaID string, reason string) error
	getActiveSagasFunc  func(filter *saga.SagaFilter) ([]saga.SagaInstance, error)
	getMetricsFunc      func() *saga.CoordinatorMetrics
	healthCheckFunc     func(ctx context.Context) error
	closeFunc           func() error
}

func (m *mockCoordinator) StartSaga(ctx context.Context, definition saga.SagaDefinition, initialData interface{}) (saga.SagaInstance, error) {
	if m.startSagaFunc != nil {
		return m.startSagaFunc(ctx, definition, initialData)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	if m.getSagaInstanceFunc != nil {
		return m.getSagaInstanceFunc(sagaID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	if m.cancelSagaFunc != nil {
		return m.cancelSagaFunc(ctx, sagaID, reason)
	}
	return errors.New("not implemented")
}

func (m *mockCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	if m.getActiveSagasFunc != nil {
		return m.getActiveSagasFunc(filter)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCoordinator) GetMetrics() *saga.CoordinatorMetrics {
	if m.getMetricsFunc != nil {
		return m.getMetricsFunc()
	}
	return &saga.CoordinatorMetrics{}
}

func (m *mockCoordinator) HealthCheck(ctx context.Context) error {
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return nil
}

func (m *mockCoordinator) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestNewRecoveryManager(t *testing.T) {
	tests := []struct {
		name         string
		config       *RecoveryConfig
		stateStorage saga.StateStorage
		coordinator  saga.SagaCoordinator
		logger       *zap.Logger
		wantErr      bool
		expectedErr  error
	}{
		{
			name:         "success with all parameters",
			config:       DefaultRecoveryConfig(),
			stateStorage: newMockStateStorage(),
			coordinator:  &mockCoordinator{},
			logger:       zap.NewNop(),
			wantErr:      false,
		},
		{
			name:         "success with nil config uses default",
			config:       nil,
			stateStorage: newMockStateStorage(),
			coordinator:  &mockCoordinator{},
			logger:       zap.NewNop(),
			wantErr:      false,
		},
		{
			name:         "success with nil logger uses nop logger",
			config:       DefaultRecoveryConfig(),
			stateStorage: newMockStateStorage(),
			coordinator:  &mockCoordinator{},
			logger:       nil,
			wantErr:      false,
		},
		{
			name:         "error when state storage is nil",
			config:       DefaultRecoveryConfig(),
			stateStorage: nil,
			coordinator:  &mockCoordinator{},
			logger:       zap.NewNop(),
			wantErr:      true,
			expectedErr:  ErrStateStorageRequired,
		},
		{
			name:         "error when coordinator is nil",
			config:       DefaultRecoveryConfig(),
			stateStorage: newMockStateStorage(),
			coordinator:  nil,
			logger:       zap.NewNop(),
			wantErr:      true,
			expectedErr:  ErrCoordinatorRequired,
		},
		{
			name: "error when config is invalid",
			config: &RecoveryConfig{
				CheckInterval: -1 * time.Second, // Invalid
			},
			stateStorage: newMockStateStorage(),
			coordinator:  &mockCoordinator{},
			logger:       zap.NewNop(),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm, err := NewRecoveryManager(tt.config, tt.stateStorage, tt.coordinator, tt.logger)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if rm == nil {
				t.Error("expected recovery manager but got nil")
				return
			}

			// Verify fields are initialized
			if rm.config == nil {
				t.Error("config should not be nil")
			}
			if rm.stateStorage == nil {
				t.Error("stateStorage should not be nil")
			}
			if rm.coordinator == nil {
				t.Error("coordinator should not be nil")
			}
			if rm.logger == nil {
				t.Error("logger should not be nil")
			}
			if rm.stats == nil {
				t.Error("stats should not be nil")
			}
			if rm.recovering == nil {
				t.Error("recovering map should not be nil")
			}

			// Clean up
			if err := rm.Close(); err != nil {
				t.Errorf("failed to close recovery manager: %v", err)
			}
		})
	}
}

func TestRecoveryManagerLifecycle(t *testing.T) {
	stateStorage := newMockStateStorage()
	coordinator := &mockCoordinator{}
	config := DefaultRecoveryConfig()
	config.CheckInterval = 2 * time.Second // Faster for testing but still valid
	config.EnableAutoRecovery = false      // Disable auto-recovery for lifecycle test

	rm, err := NewRecoveryManager(config, stateStorage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create recovery manager: %v", err)
	}

	ctx := context.Background()

	// Test Start
	if err := rm.Start(ctx); err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// Test double Start
	if err := rm.Start(ctx); err == nil {
		t.Error("expected error on double Start(), got nil")
	}

	// Test Stop
	if err := rm.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Test Stop when not started
	if err := rm.Stop(); err == nil {
		t.Error("expected error on Stop() when not started, got nil")
	}

	// Test Close
	if err := rm.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Test double Close (should not error)
	if err := rm.Close(); err != nil {
		t.Errorf("double Close() error = %v", err)
	}

	// Test operations after Close
	if err := rm.Start(ctx); !errors.Is(err, ErrRecoveryManagerClosed) {
		t.Errorf("expected ErrRecoveryManagerClosed after Close(), got %v", err)
	}

	if err := rm.RecoverSaga(ctx, "test-saga"); !errors.Is(err, ErrRecoveryManagerClosed) {
		t.Errorf("expected ErrRecoveryManagerClosed after Close(), got %v", err)
	}
}

func TestRecoveryManagerAutoRecovery(t *testing.T) {
	stateStorage := newMockStateStorage()
	coordinator := &mockCoordinator{}
	config := DefaultRecoveryConfig()
	config.CheckInterval = 1 * time.Second // Minimum valid interval for testing
	config.EnableAutoRecovery = true

	rm, err := NewRecoveryManager(config, stateStorage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create recovery manager: %v", err)
	}
	defer rm.Close()

	ctx := context.Background()

	// Start the recovery manager
	if err := rm.Start(ctx); err != nil {
		t.Fatalf("failed to start recovery manager: %v", err)
	}

	// Wait for a few check cycles
	time.Sleep(3 * time.Second)

	// Get stats
	stats := rm.GetRecoveryStats()
	if stats.TotalChecks == 0 {
		t.Error("expected at least one recovery check, got 0")
	}

	// Stop the recovery manager
	if err := rm.Stop(); err != nil {
		t.Errorf("failed to stop recovery manager: %v", err)
	}
}

func TestRecoverSaga(t *testing.T) {
	stateStorage := newMockStateStorage()
	coordinator := &mockCoordinator{}
	config := DefaultRecoveryConfig()

	rm, err := NewRecoveryManager(config, stateStorage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create recovery manager: %v", err)
	}
	defer rm.Close()

	ctx := context.Background()

	// Create a saga instance for recovery
	sagaID := "test-saga-001"
	now := time.Now()
	sagaInstance := &mockSagaInstance{
		id:           sagaID,
		definitionID: "test-definition",
		state:        saga.StateRunning,
		currentStep:  1,
		totalSteps:   3,
		startTime:    now,
		createdAt:    now,
		updatedAt:    now,
		timeout:      5 * time.Minute,
	}
	if err := stateStorage.SaveSaga(ctx, sagaInstance); err != nil {
		t.Fatalf("failed to save saga: %v", err)
	}

	// Create a step state
	stepState := &saga.StepState{
		ID:        sagaID + "-step-1",
		SagaID:    sagaID,
		StepIndex: 1,
		Name:      "test-step",
		State:     saga.StepStatePending,
		CreatedAt: now,
	}
	if err := stateStorage.SaveStepState(ctx, sagaID, stepState); err != nil {
		t.Fatalf("failed to save step state: %v", err)
	}

	// Test manual recovery
	if err := rm.RecoverSaga(ctx, sagaID); err != nil {
		t.Errorf("RecoverSaga() error = %v", err)
	}

	// Verify stats were updated
	stats := rm.GetRecoveryStats()
	if stats.TotalRecoveryAttempts == 0 {
		t.Error("expected at least one recovery attempt")
	}
	if stats.SuccessfulRecoveries == 0 {
		t.Error("expected at least one successful recovery")
	}
}

func TestRecoveryEventListeners(t *testing.T) {
	stateStorage := newMockStateStorage()
	coordinator := &mockCoordinator{}
	config := DefaultRecoveryConfig()
	config.CheckInterval = 1 * time.Second // Minimum valid interval
	config.EnableAutoRecovery = true

	rm, err := NewRecoveryManager(config, stateStorage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create recovery manager: %v", err)
	}
	defer rm.Close()

	// Add event listener
	eventChan := make(chan *RecoveryEvent, 10)
	rm.AddEventListener(func(event *RecoveryEvent) {
		eventChan <- event
	})

	ctx := context.Background()

	// Start the recovery manager
	if err := rm.Start(ctx); err != nil {
		t.Fatalf("failed to start recovery manager: %v", err)
	}

	// Wait for some events
	timeout := time.After(3 * time.Second)
	eventsReceived := 0

	for eventsReceived < 2 { // Reduce expectation since we have slower interval
		select {
		case event := <-eventChan:
			eventsReceived++
			if event == nil {
				t.Error("received nil event")
			}
			if event.Timestamp.IsZero() {
				t.Error("event timestamp is zero")
			}
		case <-timeout:
			if eventsReceived == 0 {
				t.Error("no events received within timeout")
			}
			goto done
		}
	}

done:
	// Stop the recovery manager
	if err := rm.Stop(); err != nil {
		t.Errorf("failed to stop recovery manager: %v", err)
	}
}

func TestGetRecoveryStats(t *testing.T) {
	stateStorage := newMockStateStorage()
	coordinator := &mockCoordinator{}
	config := DefaultRecoveryConfig()

	rm, err := NewRecoveryManager(config, stateStorage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create recovery manager: %v", err)
	}
	defer rm.Close()

	// Get initial stats
	stats := rm.GetRecoveryStats()
	if stats == nil {
		t.Fatal("expected stats but got nil")
	}

	// Verify initial values
	if stats.TotalChecks != 0 {
		t.Errorf("expected TotalChecks = 0, got %d", stats.TotalChecks)
	}
	if stats.TotalRecoveryAttempts != 0 {
		t.Errorf("expected TotalRecoveryAttempts = 0, got %d", stats.TotalRecoveryAttempts)
	}
	if stats.SuccessfulRecoveries != 0 {
		t.Errorf("expected SuccessfulRecoveries = 0, got %d", stats.SuccessfulRecoveries)
	}
	if stats.FailedRecoveries != 0 {
		t.Errorf("expected FailedRecoveries = 0, got %d", stats.FailedRecoveries)
	}
	if stats.CurrentlyRecovering != 0 {
		t.Errorf("expected CurrentlyRecovering = 0, got %d", stats.CurrentlyRecovering)
	}

	// Test that we get a copy (not a reference)
	stats.TotalChecks = 999
	newStats := rm.GetRecoveryStats()
	if newStats.TotalChecks == 999 {
		t.Error("GetRecoveryStats() should return a copy, not a reference")
	}
}

func TestRecoverSagaInProgress(t *testing.T) {
	stateStorage := newMockStateStorage()
	coordinator := &mockCoordinator{}
	config := DefaultRecoveryConfig()
	config.RecoveryTimeout = 2 * time.Second // Longer timeout

	rm, err := NewRecoveryManager(config, stateStorage, coordinator, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create recovery manager: %v", err)
	}
	defer rm.Close()

	ctx := context.Background()
	sagaID := "test-saga-concurrent"

	// Create a saga instance for recovery
	now := time.Now()
	sagaInstance := &mockSagaInstance{
		id:           sagaID,
		definitionID: "test-definition",
		state:        saga.StateRunning,
		currentStep:  1,
		totalSteps:   3,
		startTime:    now,
		createdAt:    now,
		updatedAt:    now,
		timeout:      5 * time.Minute,
	}
	if err := stateStorage.SaveSaga(ctx, sagaInstance); err != nil {
		t.Fatalf("failed to save saga: %v", err)
	}

	// Create a step state
	stepState := &saga.StepState{
		ID:        sagaID + "-step-1",
		SagaID:    sagaID,
		StepIndex: 1,
		Name:      "test-step",
		State:     saga.StepStatePending,
		CreatedAt: now,
	}
	if err := stateStorage.SaveStepState(ctx, sagaID, stepState); err != nil {
		t.Fatalf("failed to save step state: %v", err)
	}

	// Since the recovery implementation is a stub that completes immediately,
	// we test that concurrent calls to RecoverSaga with the same ID
	// are properly tracked in the recovering map.

	// Manually add an entry to the recovering map to simulate an in-progress recovery
	rm.recoveringMu.Lock()
	rm.recovering[sagaID] = &recoveryAttempt{
		sagaID:       sagaID,
		startTime:    time.Now(),
		attemptCount: 1,
		lastAttempt:  time.Now(),
	}
	rm.recoveringMu.Unlock()

	// Try to recover the same saga while it's marked as recovering
	err = rm.RecoverSaga(ctx, sagaID)
	if !errors.Is(err, ErrRecoveryInProgress) {
		t.Errorf("expected ErrRecoveryInProgress, got %v", err)
	}

	// Clean up the manual entry
	rm.recoveringMu.Lock()
	delete(rm.recovering, sagaID)
	rm.recoveringMu.Unlock()

	// Now recovery should succeed
	err = rm.RecoverSaga(ctx, sagaID)
	if err != nil {
		t.Errorf("expected successful recovery after cleanup, got %v", err)
	}
}

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
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// StateManager provides a unified interface for managing Saga state operations.
// It coordinates state transitions, validation, and transactional operations,
// ensuring consistency and reliability of state management.
//
// The StateManager acts as a facade over the underlying StateStorage,
// providing higher-level operations with built-in validation, event notification,
// and transaction-like semantics.
type StateManager interface {
	// SaveSaga persists a Saga instance to storage with validation.
	SaveSaga(ctx context.Context, instance saga.SagaInstance) error

	// GetSaga retrieves a Saga instance by its ID.
	GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error)

	// UpdateSagaState updates the Saga state with validation and events.
	UpdateSagaState(ctx context.Context, sagaID string, newState saga.SagaState, metadata map[string]interface{}) error

	// TransitionState performs a validated state transition with event notification.
	TransitionState(ctx context.Context, sagaID string, fromState, toState saga.SagaState, metadata map[string]interface{}) error

	// SaveStepState persists a step state with validation.
	SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error

	// GetStepStates retrieves all step states for a Saga.
	GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error)

	// DeleteSaga removes a Saga instance from storage.
	DeleteSaga(ctx context.Context, sagaID string) error

	// GetActiveSagas retrieves all active Saga instances.
	GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error)

	// GetTimeoutSagas retrieves Saga instances that have timed out.
	GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error)

	// CleanupExpiredSagas removes expired Saga instances.
	CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error

	// BatchUpdateSagas performs a batch update operation on multiple Sagas.
	BatchUpdateSagas(ctx context.Context, updates []SagaUpdate) error

	// SubscribeToStateChanges registers a listener for state change events.
	SubscribeToStateChanges(listener StateChangeListener) error

	// UnsubscribeFromStateChanges removes a state change listener.
	UnsubscribeFromStateChanges(listener StateChangeListener) error

	// Close closes the manager and releases resources.
	Close() error
}

// SagaUpdate represents a single Saga state update operation.
type SagaUpdate struct {
	SagaID   string
	NewState saga.SagaState
	Metadata map[string]interface{}
}

// StateChangeEvent represents a state change event.
type StateChangeEvent struct {
	SagaID    string
	OldState  saga.SagaState
	NewState  saga.SagaState
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// StateChangeListener is a callback function for state change notifications.
type StateChangeListener func(event *StateChangeEvent)

// DefaultStateManager provides the default implementation of StateManager.
type DefaultStateManager struct {
	storage   saga.StateStorage
	listeners []StateChangeListener
	mu        sync.RWMutex
	closed    bool
}

// NewStateManager creates a new state manager with the given storage backend.
func NewStateManager(storage saga.StateStorage) StateManager {
	return &DefaultStateManager{
		storage:   storage,
		listeners: make([]StateChangeListener, 0),
		closed:    false,
	}
}

// SaveSaga persists a Saga instance to storage with validation.
func (m *DefaultStateManager) SaveSaga(ctx context.Context, instance saga.SagaInstance) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return ErrStorageClosed
	}

	if instance == nil {
		return ErrInvalidSagaID
	}

	// Validate Saga ID
	if instance.GetID() == "" {
		return ErrInvalidSagaID
	}

	// Validate state
	if !m.isValidState(instance.GetState()) {
		return ErrInvalidState
	}

	// Save to storage
	return m.storage.SaveSaga(ctx, instance)
}

// GetSaga retrieves a Saga instance by its ID.
func (m *DefaultStateManager) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStorageClosed
	}

	if sagaID == "" {
		return nil, ErrInvalidSagaID
	}

	return m.storage.GetSaga(ctx, sagaID)
}

// UpdateSagaState updates the Saga state with validation and events.
func (m *DefaultStateManager) UpdateSagaState(ctx context.Context, sagaID string, newState saga.SagaState, metadata map[string]interface{}) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return ErrStorageClosed
	}

	if sagaID == "" {
		return ErrInvalidSagaID
	}

	// Validate new state
	if !m.isValidState(newState) {
		return ErrInvalidState
	}

	// Get current state for event notification
	currentInstance, err := m.storage.GetSaga(ctx, sagaID)
	if err != nil {
		return err
	}

	oldState := currentInstance.GetState()

	// Update state in storage
	if err := m.storage.UpdateSagaState(ctx, sagaID, newState, metadata); err != nil {
		return err
	}

	// Notify listeners
	m.notifyStateChange(&StateChangeEvent{
		SagaID:    sagaID,
		OldState:  oldState,
		NewState:  newState,
		Timestamp: time.Now(),
		Metadata:  metadata,
	})

	return nil
}

// TransitionState performs a validated state transition with event notification.
func (m *DefaultStateManager) TransitionState(ctx context.Context, sagaID string, fromState, toState saga.SagaState, metadata map[string]interface{}) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return ErrStorageClosed
	}

	if sagaID == "" {
		return ErrInvalidSagaID
	}

	// Validate state transition
	if !m.isValidTransition(fromState, toState) {
		return fmt.Errorf("invalid state transition from %s to %s", fromState.String(), toState.String())
	}

	// Get current state to verify it matches fromState
	currentInstance, err := m.storage.GetSaga(ctx, sagaID)
	if err != nil {
		return err
	}

	currentState := currentInstance.GetState()
	if currentState != fromState {
		return fmt.Errorf("expected state %s but found %s", fromState.String(), currentState.String())
	}

	// Update state in storage
	if err := m.storage.UpdateSagaState(ctx, sagaID, toState, metadata); err != nil {
		return err
	}

	// Notify listeners
	m.notifyStateChange(&StateChangeEvent{
		SagaID:    sagaID,
		OldState:  fromState,
		NewState:  toState,
		Timestamp: time.Now(),
		Metadata:  metadata,
	})

	return nil
}

// SaveStepState persists a step state with validation.
func (m *DefaultStateManager) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return ErrStorageClosed
	}

	if sagaID == "" {
		return ErrInvalidSagaID
	}

	if step == nil {
		return ErrInvalidState
	}

	// Validate step data
	if step.ID == "" {
		return fmt.Errorf("step ID cannot be empty")
	}

	return m.storage.SaveStepState(ctx, sagaID, step)
}

// GetStepStates retrieves all step states for a Saga.
func (m *DefaultStateManager) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStorageClosed
	}

	if sagaID == "" {
		return nil, ErrInvalidSagaID
	}

	return m.storage.GetStepStates(ctx, sagaID)
}

// DeleteSaga removes a Saga instance from storage.
func (m *DefaultStateManager) DeleteSaga(ctx context.Context, sagaID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return ErrStorageClosed
	}

	if sagaID == "" {
		return ErrInvalidSagaID
	}

	return m.storage.DeleteSaga(ctx, sagaID)
}

// GetActiveSagas retrieves all active Saga instances.
func (m *DefaultStateManager) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStorageClosed
	}

	return m.storage.GetActiveSagas(ctx, filter)
}

// GetTimeoutSagas retrieves Saga instances that have timed out.
func (m *DefaultStateManager) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStorageClosed
	}

	return m.storage.GetTimeoutSagas(ctx, before)
}

// CleanupExpiredSagas removes expired Saga instances.
func (m *DefaultStateManager) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return ErrStorageClosed
	}

	return m.storage.CleanupExpiredSagas(ctx, olderThan)
}

// BatchUpdateSagas performs a batch update operation on multiple Sagas.
func (m *DefaultStateManager) BatchUpdateSagas(ctx context.Context, updates []SagaUpdate) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return ErrStorageClosed
	}

	// Process each update
	for _, update := range updates {
		if err := m.UpdateSagaState(ctx, update.SagaID, update.NewState, update.Metadata); err != nil {
			return fmt.Errorf("failed to update saga %s: %w", update.SagaID, err)
		}
	}

	return nil
}

// SubscribeToStateChanges registers a listener for state change events.
func (m *DefaultStateManager) SubscribeToStateChanges(listener StateChangeListener) error {
	if listener == nil {
		return fmt.Errorf("listener cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStorageClosed
	}

	m.listeners = append(m.listeners, listener)
	return nil
}

// UnsubscribeFromStateChanges removes a state change listener.
func (m *DefaultStateManager) UnsubscribeFromStateChanges(listener StateChangeListener) error {
	if listener == nil {
		return fmt.Errorf("listener cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStorageClosed
	}

	// Find and remove the listener
	for i, l := range m.listeners {
		// Compare function pointers
		if &l == &listener {
			m.listeners = append(m.listeners[:i], m.listeners[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("listener not found")
}

// Close closes the manager and releases resources.
func (m *DefaultStateManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	m.listeners = nil

	return nil
}

// notifyStateChange notifies all registered listeners about a state change.
func (m *DefaultStateManager) notifyStateChange(event *StateChangeEvent) {
	// Note: This should be called with read lock held
	for _, listener := range m.listeners {
		// Call listener in a goroutine to avoid blocking
		go func(l StateChangeListener) {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but don't crash
					// In a real implementation, this would use a logger
				}
			}()
			l(event)
		}(listener)
	}
}

// isValidState checks if a state is valid.
func (m *DefaultStateManager) isValidState(state saga.SagaState) bool {
	// Check if state is within valid range
	return state >= saga.StatePending && state <= saga.StateTimedOut
}

// isValidTransition validates if a state transition is allowed.
func (m *DefaultStateManager) isValidTransition(from, to saga.SagaState) bool {
	// Define valid state transitions
	validTransitions := map[saga.SagaState][]saga.SagaState{
		saga.StatePending: {
			saga.StateRunning,
			saga.StateCancelled,
		},
		saga.StateRunning: {
			saga.StateStepCompleted,
			saga.StateCompleted,
			saga.StateCompensating,
			saga.StateFailed,
			saga.StateCancelled,
			saga.StateTimedOut,
		},
		saga.StateStepCompleted: {
			saga.StateRunning,
			saga.StateCompleted,
			saga.StateCompensating,
			saga.StateFailed,
			saga.StateCancelled,
		},
		saga.StateCompensating: {
			saga.StateCompensated,
			saga.StateFailed,
		},
		// Terminal states cannot transition
		saga.StateCompleted:   {},
		saga.StateCompensated: {},
		saga.StateFailed:      {},
		saga.StateCancelled:   {},
		saga.StateTimedOut:    {},
	}

	allowedStates, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedState := range allowedStates {
		if allowedState == to {
			return true
		}
	}

	return false
}

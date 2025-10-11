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
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
)

// MemoryStateStorage provides an in-memory implementation of saga.StateStorage.
// It stores Saga instances and their states in memory using a map protected by a read-write mutex.
// This implementation is suitable for development, testing, and non-critical workloads where
// durability across restarts is not required.
//
// The storage is thread-safe and supports concurrent access from multiple goroutines.
// All state modifications are protected by a sync.RWMutex to ensure data consistency.
type MemoryStateStorage struct {
	// mu protects concurrent access to the sagas and steps maps
	mu sync.RWMutex

	// sagas stores Saga instances indexed by their ID
	sagas map[string]*saga.SagaInstanceData

	// steps stores step states indexed by Saga ID
	steps map[string][]*saga.StepState

	// config holds the memory storage configuration
	config *state.MemoryStorageConfig

	// closed indicates whether the storage has been closed
	closed bool
}

// NewMemoryStateStorage creates a new in-memory state storage with default configuration.
func NewMemoryStateStorage() *MemoryStateStorage {
	return NewMemoryStateStorageWithConfig(nil)
}

// NewMemoryStateStorageWithConfig creates a new in-memory state storage with the specified configuration.
// If config is nil, default configuration is used.
func NewMemoryStateStorageWithConfig(config *state.MemoryStorageConfig) *MemoryStateStorage {
	if config == nil {
		config = &state.MemoryStorageConfig{
			InitialCapacity: 100,
			MaxCapacity:     0, // No limit
			EnableMetrics:   true,
		}
	}

	initialCapacity := config.InitialCapacity
	if initialCapacity <= 0 {
		initialCapacity = 100
	}

	return &MemoryStateStorage{
		sagas:  make(map[string]*saga.SagaInstanceData, initialCapacity),
		steps:  make(map[string][]*saga.StepState, initialCapacity),
		config: config,
		closed: false,
	}
}

// SaveSaga persists a Saga instance to memory storage.
// This method converts the SagaInstance interface to SagaInstanceData and stores it.
// If a Saga with the same ID already exists, it will be overwritten.
func (m *MemoryStateStorage) SaveSaga(ctx context.Context, instance saga.SagaInstance) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return state.ErrStorageClosed
	}

	if instance == nil {
		return state.ErrInvalidSagaID
	}

	sagaID := instance.GetID()
	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	// Check capacity limit if configured
	if m.config.MaxCapacity > 0 && len(m.sagas) >= m.config.MaxCapacity {
		// Only reject if this is a new Saga (not an update)
		if _, exists := m.sagas[sagaID]; !exists {
			return state.ErrStorageFailure
		}
	}

	// Convert SagaInstance to SagaInstanceData
	data := m.convertToSagaInstanceData(instance)

	// Store the Saga instance
	m.sagas[sagaID] = data

	return nil
}

// GetSaga retrieves a Saga instance by its ID from memory storage.
// Returns state.ErrSagaNotFound if the Saga does not exist.
func (m *MemoryStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, state.ErrStorageClosed
	}

	if sagaID == "" {
		return nil, state.ErrInvalidSagaID
	}

	data, exists := m.sagas[sagaID]
	if !exists {
		return nil, state.ErrSagaNotFound
	}

	// Return a copy of the stored data wrapped in a SagaInstance implementation
	return m.convertToSagaInstance(data), nil
}

// DeleteSaga removes a Saga instance from memory storage.
// Returns state.ErrSagaNotFound if the Saga does not exist.
func (m *MemoryStateStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return state.ErrStorageClosed
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	if _, exists := m.sagas[sagaID]; !exists {
		return state.ErrSagaNotFound
	}

	// Delete the Saga and its associated step states
	delete(m.sagas, sagaID)
	delete(m.steps, sagaID)

	return nil
}

// UpdateSagaState updates only the state of a Saga instance.
// This is a lightweight operation that doesn't require loading the entire Saga.
func (m *MemoryStateStorage) UpdateSagaState(ctx context.Context, sagaID string, sagaState saga.SagaState, metadata map[string]interface{}) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return state.ErrStorageClosed
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	data, exists := m.sagas[sagaID]
	if !exists {
		return state.ErrSagaNotFound
	}

	// Update the state
	data.State = sagaState
	data.UpdatedAt = time.Now()

	// Update metadata if provided
	if metadata != nil {
		if data.Metadata == nil {
			data.Metadata = make(map[string]interface{})
		}
		for key, value := range metadata {
			data.Metadata[key] = value
		}
	}

	return nil
}

// GetActiveSagas retrieves all active Saga instances based on the filter.
func (m *MemoryStateStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, state.ErrStorageClosed
	}

	var result []saga.SagaInstance

	for _, data := range m.sagas {
		if m.matchesFilter(data, filter) {
			result = append(result, m.convertToSagaInstance(data))
		}
	}

	return result, nil
}

// GetTimeoutSagas retrieves Saga instances that have timed out before the specified time.
func (m *MemoryStateStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, state.ErrStorageClosed
	}

	var result []saga.SagaInstance

	for _, data := range m.sagas {
		// Check if Saga is active and has exceeded its timeout
		if data.State.IsActive() && data.StartedAt != nil {
			deadline := data.StartedAt.Add(data.Timeout)
			if deadline.Before(before) {
				result = append(result, m.convertToSagaInstance(data))
			}
		}
	}

	return result, nil
}

// SaveStepState persists the state of a specific step within a Saga.
func (m *MemoryStateStorage) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return state.ErrStorageClosed
	}

	if sagaID == "" {
		return state.ErrInvalidSagaID
	}

	if step == nil {
		return state.ErrInvalidState
	}

	// Verify that the Saga exists
	if _, exists := m.sagas[sagaID]; !exists {
		return state.ErrSagaNotFound
	}

	// Get or create the step states slice for this Saga
	steps := m.steps[sagaID]

	// Find and update existing step or append new one
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

	m.steps[sagaID] = steps

	return nil
}

// GetStepStates retrieves all step states for a Saga instance.
func (m *MemoryStateStorage) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, state.ErrStorageClosed
	}

	if sagaID == "" {
		return nil, state.ErrInvalidSagaID
	}

	// Verify that the Saga exists
	if _, exists := m.sagas[sagaID]; !exists {
		return nil, state.ErrSagaNotFound
	}

	steps, exists := m.steps[sagaID]
	if !exists || len(steps) == 0 {
		return []*saga.StepState{}, nil
	}

	// Return a copy of the step states
	result := make([]*saga.StepState, len(steps))
	copy(result, steps)

	return result, nil
}

// CleanupExpiredSagas removes Saga instances that are older than the specified time.
func (m *MemoryStateStorage) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return state.ErrStorageClosed
	}

	toDelete := []string{}

	for sagaID, data := range m.sagas {
		// Only cleanup terminal state Sagas
		if data.State.IsTerminal() && data.UpdatedAt.Before(olderThan) {
			toDelete = append(toDelete, sagaID)
		}
	}

	for _, sagaID := range toDelete {
		delete(m.sagas, sagaID)
		delete(m.steps, sagaID)
	}

	return nil
}

// Close closes the storage and releases resources.
func (m *MemoryStateStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	m.sagas = nil
	m.steps = nil

	return nil
}

// convertToSagaInstanceData converts a SagaInstance to SagaInstanceData for storage.
func (m *MemoryStateStorage) convertToSagaInstanceData(instance saga.SagaInstance) *saga.SagaInstanceData {
	now := time.Now()
	startTime := instance.GetStartTime()
	endTime := instance.GetEndTime()

	data := &saga.SagaInstanceData{
		ID:           instance.GetID(),
		DefinitionID: instance.GetDefinitionID(),
		State:        instance.GetState(),
		CurrentStep:  instance.GetCurrentStep(),
		TotalSteps:   instance.GetTotalSteps(),
		CreatedAt:    instance.GetCreatedAt(),
		UpdatedAt:    instance.GetUpdatedAt(),
		Timeout:      instance.GetTimeout(),
		Metadata:     instance.GetMetadata(),
		TraceID:      instance.GetTraceID(),
		Error:        instance.GetError(),
		ResultData:   instance.GetResult(),
	}

	// Set started time if available
	if !startTime.IsZero() {
		data.StartedAt = &startTime
	}

	// Set completed time if available
	if !endTime.IsZero() {
		if instance.GetState() == saga.StateTimedOut {
			data.TimedOutAt = &endTime
		} else {
			data.CompletedAt = &endTime
		}
	}

	// Update UpdatedAt if it's zero
	if data.UpdatedAt.IsZero() {
		data.UpdatedAt = now
	}

	return data
}

// convertToSagaInstance converts SagaInstanceData to a SagaInstance implementation.
func (m *MemoryStateStorage) convertToSagaInstance(data *saga.SagaInstanceData) saga.SagaInstance {
	return &memorySagaInstance{
		data: copyInstanceData(data),
	}
}

// matchesFilter checks if a Saga instance matches the given filter criteria.
func (m *MemoryStateStorage) matchesFilter(data *saga.SagaInstanceData, filter *saga.SagaFilter) bool {
	if filter == nil {
		return true
	}

	// Filter by state
	if len(filter.States) > 0 {
		matched := false
		for _, state := range filter.States {
			if data.State == state {
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
			if data.DefinitionID == defID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Filter by creation time
	if filter.CreatedAfter != nil && data.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}

	if filter.CreatedBefore != nil && data.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}

	// Filter by metadata
	if len(filter.Metadata) > 0 {
		if data.Metadata == nil {
			return false
		}
		for key, value := range filter.Metadata {
			if data.Metadata[key] != value {
				return false
			}
		}
	}

	return true
}

// copyInstanceData creates a deep copy of SagaInstanceData.
func copyInstanceData(src *saga.SagaInstanceData) *saga.SagaInstanceData {
	if src == nil {
		return nil
	}

	dst := &saga.SagaInstanceData{
		ID:           src.ID,
		DefinitionID: src.DefinitionID,
		Name:         src.Name,
		Description:  src.Description,
		State:        src.State,
		CurrentStep:  src.CurrentStep,
		TotalSteps:   src.TotalSteps,
		CreatedAt:    src.CreatedAt,
		UpdatedAt:    src.UpdatedAt,
		InitialData:  src.InitialData,
		CurrentData:  src.CurrentData,
		ResultData:   src.ResultData,
		Error:        src.Error,
		Timeout:      src.Timeout,
		RetryPolicy:  src.RetryPolicy,
		TraceID:      src.TraceID,
		SpanID:       src.SpanID,
	}

	if src.StartedAt != nil {
		t := *src.StartedAt
		dst.StartedAt = &t
	}

	if src.CompletedAt != nil {
		t := *src.CompletedAt
		dst.CompletedAt = &t
	}

	if src.TimedOutAt != nil {
		t := *src.TimedOutAt
		dst.TimedOutAt = &t
	}

	if src.Metadata != nil {
		dst.Metadata = make(map[string]interface{}, len(src.Metadata))
		for k, v := range src.Metadata {
			dst.Metadata[k] = v
		}
	}

	if src.StepStates != nil {
		dst.StepStates = make([]*saga.StepState, len(src.StepStates))
		copy(dst.StepStates, src.StepStates)
	}

	return dst
}

// memorySagaInstance is an in-memory implementation of saga.SagaInstance.
type memorySagaInstance struct {
	data *saga.SagaInstanceData
}

// GetID returns the unique identifier of this Saga instance.
func (m *memorySagaInstance) GetID() string {
	return m.data.ID
}

// GetDefinitionID returns the identifier of the Saga definition this instance follows.
func (m *memorySagaInstance) GetDefinitionID() string {
	return m.data.DefinitionID
}

// GetState returns the current state of the Saga instance.
func (m *memorySagaInstance) GetState() saga.SagaState {
	return m.data.State
}

// GetCurrentStep returns the index of the currently executing step.
func (m *memorySagaInstance) GetCurrentStep() int {
	return m.data.CurrentStep
}

// GetStartTime returns the time when the Saga instance was created.
func (m *memorySagaInstance) GetStartTime() time.Time {
	if m.data.StartedAt != nil {
		return *m.data.StartedAt
	}
	return time.Time{}
}

// GetEndTime returns the time when the Saga instance reached a terminal state.
func (m *memorySagaInstance) GetEndTime() time.Time {
	if m.data.CompletedAt != nil {
		return *m.data.CompletedAt
	}
	if m.data.TimedOutAt != nil {
		return *m.data.TimedOutAt
	}
	return time.Time{}
}

// GetResult returns the final result data of the Saga execution.
func (m *memorySagaInstance) GetResult() interface{} {
	return m.data.ResultData
}

// GetError returns the error that caused the Saga to fail.
func (m *memorySagaInstance) GetError() *saga.SagaError {
	return m.data.Error
}

// GetTotalSteps returns the total number of steps in the Saga definition.
func (m *memorySagaInstance) GetTotalSteps() int {
	return m.data.TotalSteps
}

// GetCompletedSteps returns the number of steps that have completed successfully.
func (m *memorySagaInstance) GetCompletedSteps() int {
	count := 0
	for _, step := range m.data.StepStates {
		if step.State == saga.StepStateCompleted {
			count++
		}
	}
	return count
}

// GetCreatedAt returns the creation time of the Saga instance.
func (m *memorySagaInstance) GetCreatedAt() time.Time {
	return m.data.CreatedAt
}

// GetUpdatedAt returns the last update time of the Saga instance.
func (m *memorySagaInstance) GetUpdatedAt() time.Time {
	return m.data.UpdatedAt
}

// GetTimeout returns the timeout duration for this Saga instance.
func (m *memorySagaInstance) GetTimeout() time.Duration {
	return m.data.Timeout
}

// GetMetadata returns the metadata associated with this Saga instance.
func (m *memorySagaInstance) GetMetadata() map[string]interface{} {
	return m.data.Metadata
}

// GetTraceID returns the distributed tracing identifier for this Saga.
func (m *memorySagaInstance) GetTraceID() string {
	return m.data.TraceID
}

// IsTerminal returns true if the Saga is in a terminal state.
func (m *memorySagaInstance) IsTerminal() bool {
	return m.data.State.IsTerminal()
}

// IsActive returns true if the Saga is currently active.
func (m *memorySagaInstance) IsActive() bool {
	return m.data.State.IsActive()
}

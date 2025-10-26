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
	"errors"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// ==========================
// MockStateStorage
// ==========================

// MockStateStorage provides a mock implementation of saga.StateStorage for testing.
// It allows configuring behavior and responses for all StateStorage methods.
type MockStateStorage struct {
	mu sync.RWMutex

	// Storage for sagas and step states
	sagas      map[string]saga.SagaInstance
	stepStates map[string][]*saga.StepState

	// Configurable function handlers
	SaveSagaFunc            func(ctx context.Context, s saga.SagaInstance) error
	GetSagaFunc             func(ctx context.Context, sagaID string) (saga.SagaInstance, error)
	UpdateSagaStateFunc     func(ctx context.Context, sagaID string, state saga.SagaState, metadata map[string]interface{}) error
	DeleteSagaFunc          func(ctx context.Context, sagaID string) error
	GetActiveSagasFunc      func(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error)
	GetTimeoutSagasFunc     func(ctx context.Context, before time.Time) ([]saga.SagaInstance, error)
	SaveStepStateFunc       func(ctx context.Context, sagaID string, step *saga.StepState) error
	GetStepStatesFunc       func(ctx context.Context, sagaID string) ([]*saga.StepState, error)
	CleanupExpiredSagasFunc func(ctx context.Context, olderThan time.Time) error

	// Call tracking
	SaveSagaCalls            int
	GetSagaCalls             int
	UpdateSagaStateCalls     int
	DeleteSagaCalls          int
	GetActiveSagasCalls      int
	GetTimeoutSagasCalls     int
	SaveStepStateCalls       int
	GetStepStatesCalls       int
	CleanupExpiredSagasCalls int
}

// NewMockStateStorage creates a new mock state storage with default in-memory behavior.
func NewMockStateStorage() *MockStateStorage {
	return &MockStateStorage{
		sagas:      make(map[string]saga.SagaInstance),
		stepStates: make(map[string][]*saga.StepState),
	}
}

// SaveSaga persists a Saga instance to storage.
func (m *MockStateStorage) SaveSaga(ctx context.Context, s saga.SagaInstance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SaveSagaCalls++

	if m.SaveSagaFunc != nil {
		return m.SaveSagaFunc(ctx, s)
	}

	m.sagas[s.GetID()] = s
	return nil
}

// GetSaga retrieves a Saga instance by its ID.
func (m *MockStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetSagaCalls++

	if m.GetSagaFunc != nil {
		return m.GetSagaFunc(ctx, sagaID)
	}

	s, ok := m.sagas[sagaID]
	if !ok {
		return nil, errors.New("saga not found")
	}
	return s, nil
}

// UpdateSagaState updates only the state of a Saga instance.
func (m *MockStateStorage) UpdateSagaState(ctx context.Context, sagaID string, state saga.SagaState, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdateSagaStateCalls++

	if m.UpdateSagaStateFunc != nil {
		return m.UpdateSagaStateFunc(ctx, sagaID, state, metadata)
	}

	s, ok := m.sagas[sagaID]
	if !ok {
		return errors.New("saga not found")
	}

	// Update the state in the stored instance
	// Note: This is a simplified version, actual implementation may vary
	m.sagas[sagaID] = s
	return nil
}

// DeleteSaga removes a Saga instance from storage.
func (m *MockStateStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteSagaCalls++

	if m.DeleteSagaFunc != nil {
		return m.DeleteSagaFunc(ctx, sagaID)
	}

	delete(m.sagas, sagaID)
	delete(m.stepStates, sagaID)
	return nil
}

// GetActiveSagas retrieves all active Saga instances based on the filter.
func (m *MockStateStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetActiveSagasCalls++

	if m.GetActiveSagasFunc != nil {
		return m.GetActiveSagasFunc(ctx, filter)
	}

	var result []saga.SagaInstance
	for _, s := range m.sagas {
		// Simple filter: return all sagas if no filter provided
		if filter == nil || m.matchesFilter(s, filter) {
			result = append(result, s)
		}
	}
	return result, nil
}

// GetTimeoutSagas retrieves Saga instances that have timed out before the specified time.
func (m *MockStateStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetTimeoutSagasCalls++

	if m.GetTimeoutSagasFunc != nil {
		return m.GetTimeoutSagasFunc(ctx, before)
	}

	var result []saga.SagaInstance
	for _, s := range m.sagas {
		// Check if saga has timed out
		if s.GetStartTime().Add(s.GetTimeout()).Before(before) {
			result = append(result, s)
		}
	}
	return result, nil
}

// SaveStepState persists the state of a specific step within a Saga.
func (m *MockStateStorage) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SaveStepStateCalls++

	if m.SaveStepStateFunc != nil {
		return m.SaveStepStateFunc(ctx, sagaID, step)
	}

	steps := m.stepStates[sagaID]
	// Update existing or append new
	found := false
	for i, s := range steps {
		if s.ID == step.ID {
			steps[i] = step
			found = true
			break
		}
	}
	if !found {
		steps = append(steps, step)
	}
	m.stepStates[sagaID] = steps
	return nil
}

// GetStepStates retrieves all step states for a Saga instance.
func (m *MockStateStorage) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.GetStepStatesCalls++

	if m.GetStepStatesFunc != nil {
		return m.GetStepStatesFunc(ctx, sagaID)
	}

	steps, ok := m.stepStates[sagaID]
	if !ok {
		return []*saga.StepState{}, nil
	}
	return steps, nil
}

// CleanupExpiredSagas removes Saga instances that are older than the specified time.
func (m *MockStateStorage) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CleanupExpiredSagasCalls++

	if m.CleanupExpiredSagasFunc != nil {
		return m.CleanupExpiredSagasFunc(ctx, olderThan)
	}

	for id, s := range m.sagas {
		if s.GetStartTime().Before(olderThan) {
			delete(m.sagas, id)
			delete(m.stepStates, id)
		}
	}
	return nil
}

// Reset clears all stored data and call counters.
func (m *MockStateStorage) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sagas = make(map[string]saga.SagaInstance)
	m.stepStates = make(map[string][]*saga.StepState)
	m.SaveSagaCalls = 0
	m.GetSagaCalls = 0
	m.UpdateSagaStateCalls = 0
	m.DeleteSagaCalls = 0
	m.GetActiveSagasCalls = 0
	m.GetTimeoutSagasCalls = 0
	m.SaveStepStateCalls = 0
	m.GetStepStatesCalls = 0
	m.CleanupExpiredSagasCalls = 0
}

// matchesFilter checks if a saga instance matches the given filter.
func (m *MockStateStorage) matchesFilter(s saga.SagaInstance, filter *saga.SagaFilter) bool {
	// Simplified filter matching logic
	return true
}

// ==========================
// MockEventPublisher
// ==========================

// MockEventPublisher provides a mock implementation of saga.EventPublisher for testing.
type MockEventPublisher struct {
	mu sync.RWMutex

	// Published events
	PublishedEvents []*saga.SagaEvent

	// Subscriptions
	subscriptions map[string]*mockSubscription

	// Configurable function handlers
	PublishEventFunc func(ctx context.Context, event *saga.SagaEvent) error
	SubscribeFunc    func(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error)
	UnsubscribeFunc  func(subscription saga.EventSubscription) error
	CloseFunc        func() error

	// Call tracking
	PublishEventCalls int
	SubscribeCalls    int
	UnsubscribeCalls  int
	CloseCalls        int

	// Error simulation
	PublishEventError error
	SubscribeError    error
	UnsubscribeError  error
	CloseError        error

	// Closed state
	closed bool
}

// NewMockEventPublisher creates a new mock event publisher.
func NewMockEventPublisher() *MockEventPublisher {
	return &MockEventPublisher{
		PublishedEvents: make([]*saga.SagaEvent, 0),
		subscriptions:   make(map[string]*mockSubscription),
	}
}

// PublishEvent publishes a Saga event.
func (m *MockEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PublishEventCalls++

	if m.closed {
		return errors.New("publisher is closed")
	}

	if m.PublishEventError != nil {
		return m.PublishEventError
	}

	if m.PublishEventFunc != nil {
		return m.PublishEventFunc(ctx, event)
	}

	m.PublishedEvents = append(m.PublishedEvents, event)

	// Notify subscribers
	for _, sub := range m.subscriptions {
		if sub.filter == nil || sub.filter.Match(event) {
			go func(h saga.EventHandler) {
				_ = h.HandleEvent(ctx, event)
			}(sub.handler)
		}
	}

	return nil
}

// Subscribe subscribes to Saga events that match the given filter.
func (m *MockEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SubscribeCalls++

	if m.SubscribeError != nil {
		return nil, m.SubscribeError
	}

	if m.SubscribeFunc != nil {
		return m.SubscribeFunc(filter, handler)
	}

	sub := &mockSubscription{
		id:        generateID(),
		filter:    filter,
		handler:   handler,
		active:    true,
		createdAt: time.Now(),
		metadata:  make(map[string]interface{}),
	}
	m.subscriptions[sub.id] = sub
	return sub, nil
}

// Unsubscribe cancels a previous subscription.
func (m *MockEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UnsubscribeCalls++

	if m.UnsubscribeError != nil {
		return m.UnsubscribeError
	}

	if m.UnsubscribeFunc != nil {
		return m.UnsubscribeFunc(subscription)
	}

	delete(m.subscriptions, subscription.GetID())
	return nil
}

// Close gracefully shuts down the event publisher.
func (m *MockEventPublisher) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CloseCalls++

	if m.CloseError != nil {
		return m.CloseError
	}

	if m.CloseFunc != nil {
		return m.CloseFunc()
	}

	m.closed = true
	m.subscriptions = make(map[string]*mockSubscription)
	return nil
}

// Reset clears all published events and call counters.
func (m *MockEventPublisher) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PublishedEvents = make([]*saga.SagaEvent, 0)
	m.subscriptions = make(map[string]*mockSubscription)
	m.PublishEventCalls = 0
	m.SubscribeCalls = 0
	m.UnsubscribeCalls = 0
	m.CloseCalls = 0
	m.closed = false
}

// GetPublishedEventCount returns the number of published events.
func (m *MockEventPublisher) GetPublishedEventCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.PublishedEvents)
}

// GetEventsByType returns all published events of the given type.
func (m *MockEventPublisher) GetEventsByType(eventType saga.SagaEventType) []*saga.SagaEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*saga.SagaEvent
	for _, event := range m.PublishedEvents {
		if event.Type == eventType {
			result = append(result, event)
		}
	}
	return result
}

// mockSubscription implements saga.EventSubscription for testing.
type mockSubscription struct {
	id        string
	filter    saga.EventFilter
	handler   saga.EventHandler
	active    bool
	createdAt time.Time
	metadata  map[string]interface{}
}

func (s *mockSubscription) GetID() string {
	return s.id
}

func (s *mockSubscription) GetFilter() saga.EventFilter {
	return s.filter
}

func (s *mockSubscription) GetHandler() saga.EventHandler {
	return s.handler
}

func (s *mockSubscription) IsActive() bool {
	return s.active
}

func (s *mockSubscription) GetCreatedAt() time.Time {
	return s.createdAt
}

func (s *mockSubscription) GetMetadata() map[string]interface{} {
	return s.metadata
}

// ==========================
// MockSagaStep
// ==========================

// MockSagaStep provides a mock implementation of saga.SagaStep for testing.
type MockSagaStep struct {
	IDValue          string
	NameValue        string
	DescriptionValue string
	TimeoutValue     time.Duration
	RetryPolicyValue saga.RetryPolicy
	MetadataValue    map[string]interface{}

	// Configurable function handlers
	ExecuteFunc     func(ctx context.Context, data interface{}) (interface{}, error)
	CompensateFunc  func(ctx context.Context, data interface{}) error
	IsRetryableFunc func(err error) bool

	// Call tracking
	ExecuteCalls     int
	CompensateCalls  int
	IsRetryableCalls int

	// Default behavior
	ExecuteResult     interface{}
	ExecuteError      error
	CompensateError   error
	IsRetryableResult bool
}

// NewMockSagaStep creates a new mock saga step with sensible defaults.
func NewMockSagaStep(id, name string) *MockSagaStep {
	return &MockSagaStep{
		IDValue:           id,
		NameValue:         name,
		DescriptionValue:  "Mock step: " + name,
		TimeoutValue:      30 * time.Second,
		MetadataValue:     make(map[string]interface{}),
		IsRetryableResult: true,
	}
}

func (m *MockSagaStep) GetID() string {
	return m.IDValue
}

func (m *MockSagaStep) GetName() string {
	return m.NameValue
}

func (m *MockSagaStep) GetDescription() string {
	return m.DescriptionValue
}

func (m *MockSagaStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	m.ExecuteCalls++

	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, data)
	}

	if m.ExecuteError != nil {
		return nil, m.ExecuteError
	}

	return m.ExecuteResult, nil
}

func (m *MockSagaStep) Compensate(ctx context.Context, data interface{}) error {
	m.CompensateCalls++

	if m.CompensateFunc != nil {
		return m.CompensateFunc(ctx, data)
	}

	return m.CompensateError
}

func (m *MockSagaStep) GetTimeout() time.Duration {
	return m.TimeoutValue
}

func (m *MockSagaStep) GetRetryPolicy() saga.RetryPolicy {
	return m.RetryPolicyValue
}

func (m *MockSagaStep) IsRetryable(err error) bool {
	m.IsRetryableCalls++

	if m.IsRetryableFunc != nil {
		return m.IsRetryableFunc(err)
	}

	return m.IsRetryableResult
}

func (m *MockSagaStep) GetMetadata() map[string]interface{} {
	return m.MetadataValue
}

// ==========================
// MockSagaDefinition
// ==========================

// MockSagaDefinition provides a mock implementation of saga.SagaDefinition for testing.
type MockSagaDefinition struct {
	IDValue                   string
	NameValue                 string
	DescriptionValue          string
	StepsValue                []saga.SagaStep
	TimeoutValue              time.Duration
	RetryPolicyValue          saga.RetryPolicy
	CompensationStrategyValue saga.CompensationStrategy
	MetadataValue             map[string]interface{}

	// Configurable function handlers
	ValidateFunc func() error

	// Call tracking
	ValidateCalls int

	// Default behavior
	ValidateError error
}

// NewMockSagaDefinition creates a new mock saga definition.
func NewMockSagaDefinition(id, name string) *MockSagaDefinition {
	return &MockSagaDefinition{
		IDValue:          id,
		NameValue:        name,
		DescriptionValue: "Mock saga: " + name,
		StepsValue:       make([]saga.SagaStep, 0),
		TimeoutValue:     5 * time.Minute,
		MetadataValue:    make(map[string]interface{}),
	}
}

func (m *MockSagaDefinition) GetID() string {
	return m.IDValue
}

func (m *MockSagaDefinition) GetName() string {
	return m.NameValue
}

func (m *MockSagaDefinition) GetDescription() string {
	return m.DescriptionValue
}

func (m *MockSagaDefinition) GetSteps() []saga.SagaStep {
	return m.StepsValue
}

func (m *MockSagaDefinition) GetTimeout() time.Duration {
	return m.TimeoutValue
}

func (m *MockSagaDefinition) GetRetryPolicy() saga.RetryPolicy {
	return m.RetryPolicyValue
}

func (m *MockSagaDefinition) GetCompensationStrategy() saga.CompensationStrategy {
	return m.CompensationStrategyValue
}

func (m *MockSagaDefinition) Validate() error {
	m.ValidateCalls++

	if m.ValidateFunc != nil {
		return m.ValidateFunc()
	}

	return m.ValidateError
}

func (m *MockSagaDefinition) GetMetadata() map[string]interface{} {
	return m.MetadataValue
}

// AddStep adds a step to the mock definition.
func (m *MockSagaDefinition) AddStep(step saga.SagaStep) {
	m.StepsValue = append(m.StepsValue, step)
}

// ==========================
// Helper Functions
// ==========================

var idCounter int64

func generateID() string {
	idCounter++
	return "mock-" + time.Now().Format("20060102150405") + "-" + string(rune(idCounter))
}

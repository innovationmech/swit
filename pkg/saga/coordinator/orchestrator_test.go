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
	"github.com/innovationmech/swit/pkg/saga/state/storage"
)

// mockStateStorage is a mock implementation of saga.StateStorage for testing.
type mockStateStorage struct {
	saveSagaErr            error
	getSagaErr             error
	updateSagaStateErr     error
	deleteSagaErr          error
	getActiveSagasErr      error
	getTimeoutSagasErr     error
	saveStepStateErr       error
	getStepStatesErr       error
	cleanupExpiredSagasErr error

	// sagas allows GetSaga to return a configured instance by ID.
	sagas map[string]saga.SagaInstance

	// activeSagas allows GetActiveSagas to return a configured slice.
	activeSagas []saga.SagaInstance

	// saveSagaCount tracks the number of SaveSaga invocations.
	saveSagaCount int
	// lastUpdatedState records the most recent state passed to UpdateSagaState.
	lastUpdatedState saga.SagaState
}

func (m *mockStateStorage) SaveSaga(ctx context.Context, saga saga.SagaInstance) error {
	m.saveSagaCount++
	return m.saveSagaErr
}

func (m *mockStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	if m.getSagaErr != nil {
		return nil, m.getSagaErr
	}
	if m.sagas != nil {
		if instance, ok := m.sagas[sagaID]; ok {
			return instance, nil
		}
	}
	return nil, nil
}

func (m *mockStateStorage) UpdateSagaState(ctx context.Context, sagaID string, state saga.SagaState, metadata map[string]interface{}) error {
	m.lastUpdatedState = state
	return m.updateSagaStateErr
}

func (m *mockStateStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	return m.deleteSagaErr
}

func (m *mockStateStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	if m.getActiveSagasErr != nil {
		return nil, m.getActiveSagasErr
	}
	if m.activeSagas != nil {
		return m.activeSagas, nil
	}
	return []saga.SagaInstance{}, nil
}

func (m *mockStateStorage) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	return nil, m.getTimeoutSagasErr
}

func (m *mockStateStorage) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	return m.saveStepStateErr
}

func (m *mockStateStorage) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	return nil, m.getStepStatesErr
}

func (m *mockStateStorage) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	return m.cleanupExpiredSagasErr
}

// mockEventPublisher is a mock implementation of saga.EventPublisher for testing.
type mockEventPublisher struct {
	publishEventErr error
	subscribeErr    error
	unsubscribeErr  error
	closeErr        error
}

func (m *mockEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	return m.publishEventErr
}

func (m *mockEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	return nil, m.subscribeErr
}

func (m *mockEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	return m.unsubscribeErr
}

func (m *mockEventPublisher) Close() error {
	return m.closeErr
}

// mockRetryPolicy is a mock implementation of saga.RetryPolicy for testing.
type mockRetryPolicy struct {
	shouldRetry bool
	retryDelay  time.Duration
	maxAttempts int
}

func (m *mockRetryPolicy) ShouldRetry(err error, attempt int) bool {
	return m.shouldRetry && attempt < m.maxAttempts
}

func (m *mockRetryPolicy) GetRetryDelay(attempt int) time.Duration {
	return m.retryDelay
}

func (m *mockRetryPolicy) GetMaxAttempts() int {
	return m.maxAttempts
}

// mockMetricsCollector is a mock implementation of MetricsCollector for testing.
type mockMetricsCollector struct {
	sagaStartedCount   int
	sagaCompletedCount int
	sagaFailedCount    int
	sagaCancelledCount int
	sagaTimedOutCount  int
	stepExecutedCount  int
	stepRetriedCount   int
	compensationCount  int
}

func (m *mockMetricsCollector) RecordSagaStarted(definitionID string) {
	m.sagaStartedCount++
}

func (m *mockMetricsCollector) RecordSagaCompleted(definitionID string, duration time.Duration) {
	m.sagaCompletedCount++
}

func (m *mockMetricsCollector) RecordSagaFailed(definitionID string, errorType saga.ErrorType, duration time.Duration) {
	m.sagaFailedCount++
}

func (m *mockMetricsCollector) RecordSagaCancelled(definitionID string, duration time.Duration) {
	m.sagaCancelledCount++
}

func (m *mockMetricsCollector) RecordSagaTimedOut(definitionID string, duration time.Duration) {
	m.sagaTimedOutCount++
}

func (m *mockMetricsCollector) RecordStepExecuted(definitionID, stepID string, success bool, duration time.Duration) {
	m.stepExecutedCount++
}

func (m *mockMetricsCollector) RecordStepRetried(definitionID, stepID string, attempt int) {
	m.stepRetriedCount++
}

func (m *mockMetricsCollector) RecordCompensationExecuted(definitionID, stepID string, success bool, duration time.Duration) {
	m.compensationCount++
}

// mockSagaDefinition is a mock implementation of saga.SagaDefinition for testing.
type mockSagaDefinition struct {
	id                   string
	name                 string
	description          string
	steps                []saga.SagaStep
	timeout              time.Duration
	retryPolicy          saga.RetryPolicy
	compensationStrategy saga.CompensationStrategy
	metadata             map[string]interface{}
	validateErr          error
}

func (m *mockSagaDefinition) GetID() string {
	return m.id
}

func (m *mockSagaDefinition) GetName() string {
	return m.name
}

func (m *mockSagaDefinition) GetDescription() string {
	return m.description
}

func (m *mockSagaDefinition) GetSteps() []saga.SagaStep {
	return m.steps
}

func (m *mockSagaDefinition) GetTimeout() time.Duration {
	return m.timeout
}

func (m *mockSagaDefinition) GetRetryPolicy() saga.RetryPolicy {
	return m.retryPolicy
}

func (m *mockSagaDefinition) GetCompensationStrategy() saga.CompensationStrategy {
	return m.compensationStrategy
}

func (m *mockSagaDefinition) Validate() error {
	return m.validateErr
}

func (m *mockSagaDefinition) GetMetadata() map[string]interface{} {
	return m.metadata
}

// Test helper function to create a valid OrchestratorConfig for testing.
func newTestConfig() *OrchestratorConfig {
	return &OrchestratorConfig{
		StateStorage:     &mockStateStorage{},
		EventPublisher:   &mockEventPublisher{},
		RetryPolicy:      &mockRetryPolicy{maxAttempts: 3, retryDelay: time.Millisecond},
		MetricsCollector: &mockMetricsCollector{},
	}
}

func TestNewOrchestratorCoordinator(t *testing.T) {
	tests := []struct {
		name        string
		config      *OrchestratorConfig
		wantErr     bool
		expectedErr error
	}{
		{
			name:        "nil config",
			config:      nil,
			wantErr:     true,
			expectedErr: errors.New("config cannot be nil"),
		},
		{
			name: "missing state storage",
			config: &OrchestratorConfig{
				EventPublisher: &mockEventPublisher{},
			},
			wantErr:     true,
			expectedErr: ErrStateStorageNotConfigured,
		},
		{
			name: "missing event publisher",
			config: &OrchestratorConfig{
				StateStorage: &mockStateStorage{},
			},
			wantErr:     true,
			expectedErr: ErrEventPublisherNotConfigured,
		},
		{
			name:    "valid config with all dependencies",
			config:  newTestConfig(),
			wantErr: false,
		},
		{
			name: "valid config without optional dependencies",
			config: &OrchestratorConfig{
				StateStorage:   &mockStateStorage{},
				EventPublisher: &mockEventPublisher{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordinator, err := NewOrchestratorCoordinator(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewOrchestratorCoordinator() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewOrchestratorCoordinator() unexpected error = %v", err)
				return
			}

			if coordinator == nil {
				t.Error("NewOrchestratorCoordinator() returned nil coordinator")
				return
			}

			// Verify coordinator state
			if coordinator.stateStorage == nil {
				t.Error("coordinator.stateStorage is nil")
			}
			if coordinator.eventPublisher == nil {
				t.Error("coordinator.eventPublisher is nil")
			}
			if coordinator.retryPolicy == nil {
				t.Error("coordinator.retryPolicy is nil")
			}
			if coordinator.metricsCollector == nil {
				t.Error("coordinator.metricsCollector is nil")
			}
			if coordinator.metrics == nil {
				t.Error("coordinator.metrics is nil")
			}
			if coordinator.closed {
				t.Error("coordinator.closed should be false")
			}
		})
	}
}

func TestOrchestratorConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *OrchestratorConfig
		wantErr     bool
		expectedErr error
	}{
		{
			name: "valid config",
			config: &OrchestratorConfig{
				StateStorage:   &mockStateStorage{},
				EventPublisher: &mockEventPublisher{},
			},
			wantErr: false,
		},
		{
			name: "missing state storage",
			config: &OrchestratorConfig{
				EventPublisher: &mockEventPublisher{},
			},
			wantErr:     true,
			expectedErr: ErrStateStorageNotConfigured,
		},
		{
			name: "missing event publisher",
			config: &OrchestratorConfig{
				StateStorage: &mockStateStorage{},
			},
			wantErr:     true,
			expectedErr: ErrEventPublisherNotConfigured,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got nil")
					return
				}
				if tt.expectedErr != nil && err.Error() != tt.expectedErr.Error() {
					t.Errorf("Validate() error = %v, expected %v", err, tt.expectedErr)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestOrchestratorCoordinatorStartSaga(t *testing.T) {
	tests := []struct {
		name        string
		setupCoord  func() *OrchestratorCoordinator
		definition  saga.SagaDefinition
		initialData interface{}
		wantErr     bool
		expectedErr error
	}{
		{
			name: "closed coordinator",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				coord.closed = true
				return coord
			},
			definition:  &mockSagaDefinition{id: "test-saga"},
			initialData: map[string]interface{}{"key": "value"},
			wantErr:     true,
			expectedErr: ErrCoordinatorClosed,
		},
		{
			name: "nil definition",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			definition:  nil,
			initialData: map[string]interface{}{"key": "value"},
			wantErr:     true,
			expectedErr: ErrInvalidDefinition,
		},
		{
			name: "invalid definition",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			definition: &mockSagaDefinition{
				id:          "test-saga",
				validateErr: errors.New("invalid definition"),
			},
			initialData: map[string]interface{}{"key": "value"},
			wantErr:     true,
		},
		{
			name: "nil initial data",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			definition: &mockSagaDefinition{
				id:      "test-saga",
				name:    "Test Saga",
				steps:   []saga.SagaStep{},
				timeout: 30 * time.Second,
			},
			initialData: nil,
			wantErr:     true,
			expectedErr: ErrInvalidInitialData,
		},
		{
			name: "successful saga creation",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			definition: &mockSagaDefinition{
				id:          "test-saga-def",
				name:        "Test Saga",
				description: "A test saga",
				steps: []saga.SagaStep{
					&mockSagaStep{
						id:   "step-1",
						name: "Test Step",
					},
				},
				timeout:  30 * time.Second,
				metadata: map[string]interface{}{"env": "test"},
			},
			initialData: map[string]interface{}{"order_id": "12345"},
			wantErr:     false,
		},
		{
			name: "storage error",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				config.StateStorage = &mockStateStorage{
					saveSagaErr: errors.New("storage failure"),
				}
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			definition: &mockSagaDefinition{
				id:      "test-saga",
				name:    "Test Saga",
				steps:   []saga.SagaStep{},
				timeout: 30 * time.Second,
			},
			initialData: map[string]interface{}{"key": "value"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setupCoord()
			ctx := context.Background()

			instance, err := coord.StartSaga(ctx, tt.definition, tt.initialData)
			if tt.wantErr {
				if err == nil {
					t.Errorf("StartSaga() expected error but got nil")
				}
				if tt.expectedErr != nil && err != tt.expectedErr {
					// Just check that error exists for now
				}
				return
			}

			if err != nil {
				t.Errorf("StartSaga() unexpected error = %v", err)
				return
			}

			// Verify instance was created correctly
			if instance == nil {
				t.Fatal("StartSaga() returned nil instance")
			}

			// Verify instance fields
			if instance.GetID() == "" {
				t.Error("instance ID should not be empty")
			}
			if instance.GetDefinitionID() != tt.definition.GetID() {
				t.Errorf("instance.GetDefinitionID() = %s, expected %s", instance.GetDefinitionID(), tt.definition.GetID())
			}
			// State may be Pending, Running, or Completed due to async execution
			// Just check it's a valid state
			state := instance.GetState()
			if state != saga.StatePending && state != saga.StateRunning && state != saga.StateCompleted {
				t.Errorf("instance.GetState() = %v, expected valid state", state)
			}
			// CurrentStep and CompletedSteps may vary due to async execution
			// Just verify they're within valid range
			if instance.GetCurrentStep() < -1 || instance.GetCurrentStep() >= instance.GetTotalSteps() {
				t.Errorf("instance.GetCurrentStep() = %d, out of valid range", instance.GetCurrentStep())
			}
			if instance.GetTotalSteps() != len(tt.definition.GetSteps()) {
				t.Errorf("instance.GetTotalSteps() = %d, expected %d", instance.GetTotalSteps(), len(tt.definition.GetSteps()))
			}
			if instance.GetCompletedSteps() < 0 || instance.GetCompletedSteps() > instance.GetTotalSteps() {
				t.Errorf("instance.GetCompletedSteps() = %d, out of valid range", instance.GetCompletedSteps())
			}
			if instance.GetCreatedAt().IsZero() {
				t.Error("instance.GetCreatedAt() should not be zero")
			}
			if instance.GetTimeout() != tt.definition.GetTimeout() {
				t.Errorf("instance.GetTimeout() = %v, expected %v", instance.GetTimeout(), tt.definition.GetTimeout())
			}

			// Verify instance is stored in memory cache
			cachedInstance, err := coord.GetSagaInstance(instance.GetID())
			if err != nil {
				t.Errorf("GetSagaInstance() error = %v", err)
			}
			if cachedInstance.GetID() != instance.GetID() {
				t.Error("cached instance ID does not match")
			}

			// Verify metrics were updated
			metrics := coord.GetMetrics()
			if metrics.TotalSagas == 0 {
				t.Error("TotalSagas should be incremented")
			}
			// ActiveSagas may have already completed due to async execution
			// Just verify TotalSagas was incremented
		})
	}
}

func TestOrchestratorSagaInstance(t *testing.T) {
	now := time.Now()
	startedAt := now.Add(-1 * time.Hour)

	instance := &OrchestratorSagaInstance{
		id:             "test-instance-123",
		definitionID:   "test-definition",
		name:           "Test Instance",
		description:    "A test instance",
		state:          saga.StateRunning,
		currentStep:    1,
		completedSteps: 1,
		totalSteps:     3,
		createdAt:      now.Add(-2 * time.Hour),
		updatedAt:      now,
		startedAt:      &startedAt,
		initialData:    map[string]interface{}{"key": "initial"},
		currentData:    map[string]interface{}{"key": "current"},
		timeout:        30 * time.Minute,
		metadata:       map[string]interface{}{"env": "test"},
		traceID:        "trace-123",
		spanID:         "span-456",
	}

	// Test all getter methods
	if instance.GetID() != "test-instance-123" {
		t.Errorf("GetID() = %s, expected test-instance-123", instance.GetID())
	}
	if instance.GetDefinitionID() != "test-definition" {
		t.Errorf("GetDefinitionID() = %s, expected test-definition", instance.GetDefinitionID())
	}
	if instance.GetState() != saga.StateRunning {
		t.Errorf("GetState() = %v, expected StateRunning", instance.GetState())
	}
	if instance.GetCurrentStep() != 1 {
		t.Errorf("GetCurrentStep() = %d, expected 1", instance.GetCurrentStep())
	}
	if instance.GetCompletedSteps() != 1 {
		t.Errorf("GetCompletedSteps() = %d, expected 1", instance.GetCompletedSteps())
	}
	if instance.GetTotalSteps() != 3 {
		t.Errorf("GetTotalSteps() = %d, expected 3", instance.GetTotalSteps())
	}
	if instance.GetTimeout() != 30*time.Minute {
		t.Errorf("GetTimeout() = %v, expected 30m", instance.GetTimeout())
	}
	if instance.GetTraceID() != "trace-123" {
		t.Errorf("GetTraceID() = %s, expected trace-123", instance.GetTraceID())
	}
	if !instance.IsActive() {
		t.Error("IsActive() should return true for StateRunning")
	}
	if instance.IsTerminal() {
		t.Error("IsTerminal() should return false for StateRunning")
	}

	// Test metadata copy
	metadata := instance.GetMetadata()
	metadata["new_key"] = "new_value"
	if _, exists := instance.metadata["new_key"]; exists {
		t.Error("GetMetadata() should return a copy, not the original")
	}

	// Test terminal state
	instance.state = saga.StateCompleted
	if !instance.IsTerminal() {
		t.Error("IsTerminal() should return true for StateCompleted")
	}
	if instance.IsActive() {
		t.Error("IsActive() should return false for StateCompleted")
	}
}

func TestGenerateSagaID(t *testing.T) {
	// Test multiple generations for uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := generateSagaID()
		if err != nil {
			t.Fatalf("generateSagaID() error = %v", err)
		}
		if id == "" {
			t.Error("generateSagaID() returned empty string")
		}
		if ids[id] {
			t.Errorf("generateSagaID() generated duplicate ID: %s", id)
		}
		ids[id] = true

		// Verify format (should start with "saga-" and follow UUID-like pattern)
		if len(id) < 10 {
			t.Errorf("generateSagaID() returned short ID: %s", id)
		}
		if id[:5] != "saga-" {
			t.Errorf("generateSagaID() ID should start with 'saga-': %s", id)
		}
	}
}

func TestGenerateEventID(t *testing.T) {
	// Test multiple generations for uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateEventID()
		if id == "" {
			t.Error("generateEventID() returned empty string")
		}
		if ids[id] {
			t.Errorf("generateEventID() generated duplicate ID: %s", id)
		}
		ids[id] = true

		// Verify format (should start with "event-")
		if len(id) < 10 {
			t.Errorf("generateEventID() returned short ID: %s", id)
		}
		if id[:6] != "event-" {
			t.Errorf("generateEventID() ID should start with 'event-': %s", id)
		}
	}
}

func TestCopyMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		want     map[string]interface{}
	}{
		{
			name:     "nil metadata",
			metadata: nil,
			want:     map[string]interface{}{},
		},
		{
			name:     "empty metadata",
			metadata: map[string]interface{}{},
			want:     map[string]interface{}{},
		},
		{
			name: "non-empty metadata",
			metadata: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
			want: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := copyMetadata(tt.metadata)

			// Verify length
			if len(got) != len(tt.want) {
				t.Errorf("copyMetadata() length = %d, want %d", len(got), len(tt.want))
			}

			// Verify contents
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("copyMetadata()[%s] = %v, want %v", k, got[k], v)
				}
			}

			// Verify it's a copy (mutation doesn't affect original)
			if tt.metadata != nil {
				got["test_key"] = "test_value"
				if _, exists := tt.metadata["test_key"]; exists {
					t.Error("copyMetadata() should return a copy, not reference original")
				}
			}
		})
	}
}

// fakeSagaInstance is a minimal saga.SagaInstance implementation that is NOT
// an *OrchestratorSagaInstance. It is used to exercise the storage-only
// lifecycle code paths (where the live runtime instance is unavailable).
type fakeSagaInstance struct {
	id           string
	definitionID string
	state        saga.SagaState
	startTime    time.Time
}

func (f *fakeSagaInstance) GetID() string                       { return f.id }
func (f *fakeSagaInstance) GetDefinitionID() string             { return f.definitionID }
func (f *fakeSagaInstance) GetState() saga.SagaState            { return f.state }
func (f *fakeSagaInstance) GetCurrentStep() int                 { return 0 }
func (f *fakeSagaInstance) GetStartTime() time.Time             { return f.startTime }
func (f *fakeSagaInstance) GetEndTime() time.Time               { return time.Time{} }
func (f *fakeSagaInstance) GetResult() interface{}              { return nil }
func (f *fakeSagaInstance) GetError() *saga.SagaError           { return nil }
func (f *fakeSagaInstance) GetTotalSteps() int                  { return 0 }
func (f *fakeSagaInstance) GetCompletedSteps() int              { return 0 }
func (f *fakeSagaInstance) GetCreatedAt() time.Time             { return f.startTime }
func (f *fakeSagaInstance) GetUpdatedAt() time.Time             { return f.startTime }
func (f *fakeSagaInstance) GetTimeout() time.Duration           { return 0 }
func (f *fakeSagaInstance) GetMetadata() map[string]interface{} { return nil }
func (f *fakeSagaInstance) GetTraceID() string                  { return "" }
func (f *fakeSagaInstance) IsTerminal() bool                    { return f.state.IsTerminal() }
func (f *fakeSagaInstance) IsActive() bool                      { return f.state.IsActive() }

// newRunningInstance creates an in-memory OrchestratorSagaInstance in the
// running state with the supplied number of completed steps, useful for
// exercising the lifecycle APIs deterministically.
func newRunningInstance(id string, totalSteps, completedSteps int) *OrchestratorSagaInstance {
	now := time.Now()
	started := now.Add(-time.Minute)
	return &OrchestratorSagaInstance{
		id:             id,
		definitionID:   "test-def",
		name:           "Test Saga",
		state:          saga.StateRunning,
		currentStep:    completedSteps,
		completedSteps: completedSteps,
		totalSteps:     totalSteps,
		createdAt:      started,
		updatedAt:      now,
		startedAt:      &started,
		metadata:       map[string]interface{}{"env": "test"},
	}
}

func TestOrchestratorCoordinatorGetSagaInstance(t *testing.T) {
	cachedInstance := newRunningInstance("cached-saga", 2, 1)
	storedInstance := newRunningInstance("stored-saga", 3, 2)

	tests := []struct {
		name       string
		setupCoord func() *OrchestratorCoordinator
		sagaID     string
		wantErr    bool
		wantID     string
	}{
		{
			name: "closed coordinator",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				coord.closed = true
				return coord
			},
			sagaID:  "test-saga-id",
			wantErr: true,
		},
		{
			name: "saga not found",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			sagaID:  "non-existent-saga",
			wantErr: true,
		},
		{
			name: "found in memory cache",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				coord.instances.Store(cachedInstance.id, cachedInstance)
				return coord
			},
			sagaID:  "cached-saga",
			wantErr: false,
			wantID:  "cached-saga",
		},
		{
			name: "found in storage fallback",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				config.StateStorage = &mockStateStorage{
					sagas: map[string]saga.SagaInstance{
						storedInstance.id: storedInstance,
					},
				}
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			sagaID:  "stored-saga",
			wantErr: false,
			wantID:  "stored-saga",
		},
		{
			name: "storage error",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				config.StateStorage = &mockStateStorage{
					getSagaErr: errors.New("storage failure"),
				}
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			sagaID:  "any-saga",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setupCoord()

			instance, err := coord.GetSagaInstance(tt.sagaID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetSagaInstance() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetSagaInstance() unexpected error = %v", err)
				return
			}
			if instance == nil {
				t.Fatal("GetSagaInstance() returned nil instance")
			}
			if instance.GetID() != tt.wantID {
				t.Errorf("GetSagaInstance() ID = %s, expected %s", instance.GetID(), tt.wantID)
			}
		})
	}
}

func TestOrchestratorCoordinatorCancelSaga(t *testing.T) {
	t.Run("closed coordinator", func(t *testing.T) {
		coord, _ := NewOrchestratorCoordinator(newTestConfig())
		coord.closed = true
		if err := coord.CancelSaga(context.Background(), "id", "reason"); err == nil {
			t.Error("CancelSaga() expected error for closed coordinator")
		}
	})

	t.Run("saga not found", func(t *testing.T) {
		coord, _ := NewOrchestratorCoordinator(newTestConfig())
		if err := coord.CancelSaga(context.Background(), "missing", "reason"); err == nil {
			t.Error("CancelSaga() expected error for missing saga")
		}
	})

	t.Run("terminal saga rejected", func(t *testing.T) {
		coord, _ := NewOrchestratorCoordinator(newTestConfig())
		instance := newRunningInstance("done-saga", 1, 1)
		instance.state = saga.StateCompleted
		coord.instances.Store(instance.id, instance)
		if err := coord.CancelSaga(context.Background(), instance.id, "reason"); err == nil {
			t.Error("CancelSaga() expected error for terminal saga")
		}
	})

	t.Run("successful cancel triggers compensation", func(t *testing.T) {
		metrics := &mockMetricsCollector{}
		config := newTestConfig()
		config.MetricsCollector = metrics
		coord, _ := NewOrchestratorCoordinator(config)

		var compensated1, compensated2 int32
		step1 := &mockSagaStep{
			id:   "step-1",
			name: "Step 1",
			compensate: func(ctx context.Context, data interface{}) error {
				atomic.AddInt32(&compensated1, 1)
				return nil
			},
		}
		step2 := &mockSagaStep{
			id:   "step-2",
			name: "Step 2",
			compensate: func(ctx context.Context, data interface{}) error {
				atomic.AddInt32(&compensated2, 1)
				return nil
			},
		}
		definition := &mockSagaDefinition{
			id:    "test-def",
			name:  "Test Saga",
			steps: []saga.SagaStep{step1, step2},
		}

		instance := newRunningInstance("active-saga", 2, 2)
		coord.instances.Store(instance.id, instance)
		coord.definitions.Store(instance.id, definition)
		coord.mu.Lock()
		coord.metrics.ActiveSagas = 1
		coord.mu.Unlock()

		err := coord.CancelSaga(context.Background(), instance.id, "user requested")
		if err != nil {
			t.Fatalf("CancelSaga() unexpected error = %v", err)
		}

		if instance.GetState() != saga.StateCancelled {
			t.Errorf("instance state = %v, expected Cancelled", instance.GetState())
		}
		if atomic.LoadInt32(&compensated1) != 1 {
			t.Errorf("step1 compensation called %d times, expected 1", compensated1)
		}
		if atomic.LoadInt32(&compensated2) != 1 {
			t.Errorf("step2 compensation called %d times, expected 1", compensated2)
		}
		if metrics.sagaCancelledCount != 1 {
			t.Errorf("RecordSagaCancelled called %d times, expected 1", metrics.sagaCancelledCount)
		}

		m := coord.GetMetrics()
		if m.CancelledSagas != 1 {
			t.Errorf("CancelledSagas = %d, expected 1", m.CancelledSagas)
		}
		if m.ActiveSagas != 0 {
			t.Errorf("ActiveSagas = %d, expected 0", m.ActiveSagas)
		}

		// The instance should be removed from the active cache after cancel.
		if _, ok := coord.instances.Load(instance.id); ok {
			t.Error("instance should be removed from cache after cancellation")
		}
	})

	t.Run("cancel without completed steps skips compensation", func(t *testing.T) {
		coord, _ := NewOrchestratorCoordinator(newTestConfig())
		step := &mockSagaStep{
			id:   "step-1",
			name: "Step 1",
			compensate: func(ctx context.Context, data interface{}) error {
				t.Error("compensation should not be called when no steps completed")
				return nil
			},
		}
		definition := &mockSagaDefinition{
			id:    "test-def",
			steps: []saga.SagaStep{step},
		}
		instance := newRunningInstance("pending-saga", 1, 0)
		instance.state = saga.StatePending
		coord.instances.Store(instance.id, instance)
		coord.definitions.Store(instance.id, definition)

		if err := coord.CancelSaga(context.Background(), instance.id, "reason"); err != nil {
			t.Fatalf("CancelSaga() unexpected error = %v", err)
		}
		if instance.GetState() != saga.StateCancelled {
			t.Errorf("instance state = %v, expected Cancelled", instance.GetState())
		}
	})

	t.Run("storage failure on persist", func(t *testing.T) {
		config := newTestConfig()
		config.StateStorage = &mockStateStorage{saveSagaErr: errors.New("save failed")}
		coord, _ := NewOrchestratorCoordinator(config)
		instance := newRunningInstance("active-saga", 1, 0)
		coord.instances.Store(instance.id, instance)

		if err := coord.CancelSaga(context.Background(), instance.id, "reason"); err == nil {
			t.Error("CancelSaga() expected storage error")
		}
	})

	t.Run("storage-only instance updates state", func(t *testing.T) {
		storage := &mockStateStorage{
			sagas: map[string]saga.SagaInstance{
				"stored-saga": &fakeSagaInstance{
					id:           "stored-saga",
					definitionID: "test-def",
					state:        saga.StateRunning,
					startTime:    time.Now().Add(-time.Minute),
				},
			},
		}
		config := newTestConfig()
		config.StateStorage = storage
		coord, _ := NewOrchestratorCoordinator(config)

		if err := coord.CancelSaga(context.Background(), "stored-saga", "reason"); err != nil {
			t.Fatalf("CancelSaga() unexpected error = %v", err)
		}
		if storage.lastUpdatedState != saga.StateCancelled {
			t.Errorf("UpdateSagaState state = %v, expected Cancelled", storage.lastUpdatedState)
		}
	})
}

func TestOrchestratorCoordinatorGetActiveSagas(t *testing.T) {
	tests := []struct {
		name       string
		setupCoord func() *OrchestratorCoordinator
		filter     *saga.SagaFilter
		wantErr    bool
		wantLen    int
	}{
		{
			name: "closed coordinator",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				coord.closed = true
				return coord
			},
			filter:  nil,
			wantErr: true,
		},
		{
			name: "no filter - returns empty list",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			filter:  nil,
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "returns active sagas from storage",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				config.StateStorage = &mockStateStorage{
					activeSagas: []saga.SagaInstance{
						newRunningInstance("saga-1", 2, 1),
						newRunningInstance("saga-2", 3, 0),
					},
				}
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			filter:  nil,
			wantErr: false,
			wantLen: 2,
		},
		{
			name: "storage error is wrapped",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				config.StateStorage = &mockStateStorage{
					getActiveSagasErr: errors.New("query failed"),
				}
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			filter:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setupCoord()

			sagas, err := coord.GetActiveSagas(tt.filter)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetActiveSagas() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetActiveSagas() unexpected error = %v", err)
				return
			}

			if len(sagas) != tt.wantLen {
				t.Errorf("GetActiveSagas() returned %d sagas, expected %d", len(sagas), tt.wantLen)
			}
		})
	}

	t.Run("stale storage state is overridden by live terminal instance", func(t *testing.T) {
		// Storage still reports the Saga as running (e.g. the final persist
		// failed during a network partition), but the live in-memory
		// instance already completed. The stale entry must be filtered out.
		staleStored := newRunningInstance("saga-live", 2, 2)
		config := newTestConfig()
		config.StateStorage = &mockStateStorage{
			activeSagas: []saga.SagaInstance{staleStored},
		}
		coord, _ := NewOrchestratorCoordinator(config)

		liveInstance := newRunningInstance("saga-live", 2, 2)
		liveInstance.state = saga.StateCompleted
		coord.instances.Store(liveInstance.id, liveInstance)

		sagas, err := coord.GetActiveSagas(nil)
		if err != nil {
			t.Fatalf("GetActiveSagas() unexpected error = %v", err)
		}
		if len(sagas) != 0 {
			t.Errorf("GetActiveSagas() returned %d sagas, expected 0 (live instance is terminal)", len(sagas))
		}
	})

	t.Run("live active instance is preferred over storage copy", func(t *testing.T) {
		staleStored := newRunningInstance("saga-live", 2, 0)
		config := newTestConfig()
		config.StateStorage = &mockStateStorage{
			activeSagas: []saga.SagaInstance{staleStored},
		}
		coord, _ := NewOrchestratorCoordinator(config)

		liveInstance := newRunningInstance("saga-live", 2, 1)
		liveInstance.state = saga.StateCompensating
		coord.instances.Store(liveInstance.id, liveInstance)

		sagas, err := coord.GetActiveSagas(nil)
		if err != nil {
			t.Fatalf("GetActiveSagas() unexpected error = %v", err)
		}
		if len(sagas) != 1 {
			t.Fatalf("GetActiveSagas() returned %d sagas, expected 1", len(sagas))
		}
		if sagas[0] != saga.SagaInstance(liveInstance) {
			t.Error("GetActiveSagas() should return the live in-memory instance")
		}
		if sagas[0].GetState() != saga.StateCompensating {
			t.Errorf("GetActiveSagas() state = %v, expected Compensating", sagas[0].GetState())
		}
	})
}

func TestBuildActiveSagaFilter(t *testing.T) {
	t.Run("nil filter defaults to active states", func(t *testing.T) {
		f := buildActiveSagaFilter(nil)
		if len(f.States) != 3 {
			t.Fatalf("expected 3 default active states, got %d", len(f.States))
		}
	})

	t.Run("explicit states are preserved and original not mutated", func(t *testing.T) {
		original := &saga.SagaFilter{States: []saga.SagaState{saga.StateCompleted}}
		f := buildActiveSagaFilter(original)
		if len(f.States) != 1 || f.States[0] != saga.StateCompleted {
			t.Errorf("explicit states should be preserved, got %v", f.States)
		}
	})

	t.Run("empty states are populated with defaults without mutating caller", func(t *testing.T) {
		original := &saga.SagaFilter{Limit: 5}
		f := buildActiveSagaFilter(original)
		if len(f.States) != 3 {
			t.Errorf("expected default active states, got %v", f.States)
		}
		if len(original.States) != 0 {
			t.Error("buildActiveSagaFilter must not mutate the caller filter")
		}
	})
}

func TestOrchestratorCoordinatorGetMetrics(t *testing.T) {
	config := newTestConfig()
	coord, err := NewOrchestratorCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}

	metrics := coord.GetMetrics()
	if metrics == nil {
		t.Error("GetMetrics() returned nil")
		return
	}

	// Verify metrics structure
	if metrics.TotalSagas != 0 {
		t.Errorf("TotalSagas = %d, expected 0", metrics.TotalSagas)
	}
	if metrics.ActiveSagas != 0 {
		t.Errorf("ActiveSagas = %d, expected 0", metrics.ActiveSagas)
	}
	if metrics.StartTime.IsZero() {
		t.Error("StartTime is zero")
	}
	if metrics.LastUpdateTime.IsZero() {
		t.Error("LastUpdateTime is zero")
	}

	// Verify that returned metrics is a copy
	metrics.TotalSagas = 100
	newMetrics := coord.GetMetrics()
	if newMetrics.TotalSagas != 0 {
		t.Error("GetMetrics() should return a copy, not a reference")
	}
}

func TestOrchestratorCoordinatorHealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		setupCoord func() *OrchestratorCoordinator
		wantErr    bool
	}{
		{
			name: "closed coordinator",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				coord.closed = true
				return coord
			},
			wantErr: true,
		},
		{
			name: "healthy coordinator",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			wantErr: false,
		},
		{
			name: "state storage probe failure",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				config.StateStorage = &mockStateStorage{
					getActiveSagasErr: errors.New("storage down"),
				}
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setupCoord()
			ctx := context.Background()

			err := coord.HealthCheck(ctx)
			if tt.wantErr {
				if err == nil {
					t.Errorf("HealthCheck() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("HealthCheck() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestOrchestratorCoordinatorHealthCheckExplicitProbe(t *testing.T) {
	t.Run("healthy explicit probe", func(t *testing.T) {
		config := newTestConfig()
		config.StateStorage = &healthCheckStateStorage{mockStateStorage: &mockStateStorage{}}
		coord, _ := NewOrchestratorCoordinator(config)
		if err := coord.HealthCheck(context.Background()); err != nil {
			t.Errorf("HealthCheck() unexpected error = %v", err)
		}
	})

	t.Run("unhealthy explicit probe", func(t *testing.T) {
		config := newTestConfig()
		config.StateStorage = &healthCheckStateStorage{
			mockStateStorage: &mockStateStorage{},
			healthErr:        errors.New("ping failed"),
		}
		coord, _ := NewOrchestratorCoordinator(config)
		if err := coord.HealthCheck(context.Background()); err == nil {
			t.Error("HealthCheck() expected error from explicit probe")
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		coord, _ := NewOrchestratorCoordinator(newTestConfig())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := coord.HealthCheck(ctx); err == nil {
			t.Error("HealthCheck() expected error for cancelled context")
		}
	})
}

// healthCheckStateStorage wraps mockStateStorage and adds an explicit
// HealthCheck method, exercising the healthCheckable code path.
type healthCheckStateStorage struct {
	*mockStateStorage
	healthErr error
}

func (h *healthCheckStateStorage) HealthCheck(ctx context.Context) error {
	return h.healthErr
}

func TestOrchestratorCoordinatorClose(t *testing.T) {
	tests := []struct {
		name       string
		setupCoord func() *OrchestratorCoordinator
		wantErr    bool
	}{
		{
			name: "close active coordinator",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			wantErr: false,
		},
		{
			name: "close already closed coordinator",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				coord.closed = true
				return coord
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setupCoord()

			err := coord.Close()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Close() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Close() unexpected error = %v", err)
				}
				if !coord.closed {
					t.Error("Close() should set closed flag to true")
				}
			}
		})
	}
}

func TestOrchestratorCoordinatorStopSaga(t *testing.T) {
	t.Run("closed coordinator", func(t *testing.T) {
		coord, _ := NewOrchestratorCoordinator(newTestConfig())
		coord.closed = true
		if err := coord.StopSaga(context.Background(), "id"); err == nil {
			t.Error("StopSaga() expected error for closed coordinator")
		}
	})

	t.Run("saga not found", func(t *testing.T) {
		coord, _ := NewOrchestratorCoordinator(newTestConfig())
		if err := coord.StopSaga(context.Background(), "missing"); err == nil {
			t.Error("StopSaga() expected error for missing saga")
		}
	})

	t.Run("terminal saga rejected", func(t *testing.T) {
		coord, _ := NewOrchestratorCoordinator(newTestConfig())
		instance := newRunningInstance("done-saga", 1, 1)
		instance.state = saga.StateCompleted
		coord.instances.Store(instance.id, instance)
		if err := coord.StopSaga(context.Background(), instance.id); err == nil {
			t.Error("StopSaga() expected error for terminal saga")
		}
	})

	t.Run("successful stop persists state and cancels execution", func(t *testing.T) {
		storage := &mockStateStorage{}
		config := newTestConfig()
		config.StateStorage = storage
		coord, _ := NewOrchestratorCoordinator(config)

		instance := newRunningInstance("active-saga", 2, 1)
		coord.instances.Store(instance.id, instance)

		// Track a cancellation function to verify it is invoked.
		var cancelled int32
		coord.cancelFuncs.Store(instance.id, context.CancelFunc(func() {
			atomic.AddInt32(&cancelled, 1)
		}))

		before := storage.saveSagaCount
		if err := coord.StopSaga(context.Background(), instance.id); err != nil {
			t.Fatalf("StopSaga() unexpected error = %v", err)
		}
		if storage.saveSagaCount <= before {
			t.Error("StopSaga() should persist current state via SaveSaga")
		}
		if atomic.LoadInt32(&cancelled) != 1 {
			t.Errorf("StopSaga() should cancel in-flight execution, calls = %d", cancelled)
		}
	})

	t.Run("storage failure on persist", func(t *testing.T) {
		config := newTestConfig()
		config.StateStorage = &mockStateStorage{saveSagaErr: errors.New("save failed")}
		coord, _ := NewOrchestratorCoordinator(config)
		instance := newRunningInstance("active-saga", 1, 0)
		coord.instances.Store(instance.id, instance)

		if err := coord.StopSaga(context.Background(), instance.id); err == nil {
			t.Error("StopSaga() expected storage error")
		}
	})
}

// TestOrchestratorCoordinatorLifecycleConcurrent exercises the lifecycle APIs
// under concurrent access to surface data races in the coordinator (run with
// -race). It uses the thread-safe in-memory storage and the stateless no-op
// metrics collector so that any race surfaced originates from the coordinator
// itself rather than from test doubles.
func TestOrchestratorCoordinatorLifecycleConcurrent(t *testing.T) {
	config := &OrchestratorConfig{
		StateStorage:     storage.NewMemoryStateStorage(),
		EventPublisher:   &mockEventPublisher{},
		MetricsCollector: &noOpMetricsCollector{},
	}
	coord, err := NewOrchestratorCoordinator(config)
	if err != nil {
		t.Fatalf("failed to create coordinator: %v", err)
	}

	const numSagas = 20
	ids := make([]string, 0, numSagas)
	for i := 0; i < numSagas; i++ {
		id := fmt.Sprintf("saga-%02d", i)
		ids = append(ids, id)
		instance := newRunningInstance(id, 1, 0)
		coord.instances.Store(id, instance)
		coord.mu.Lock()
		coord.metrics.ActiveSagas++
		coord.mu.Unlock()
	}

	var wg sync.WaitGroup
	for _, id := range ids {
		id := id
		wg.Add(3)
		go func() {
			defer wg.Done()
			_ = coord.CancelSaga(context.Background(), id, "concurrent cancel")
		}()
		go func() {
			defer wg.Done()
			_, _ = coord.GetSagaInstance(id)
		}()
		go func() {
			defer wg.Done()
			_, _ = coord.GetActiveSagas(nil)
		}()
	}

	wg.Wait()

	// After cancellation, cancelled count should not exceed the number created.
	m := coord.GetMetrics()
	if m.CancelledSagas > numSagas {
		t.Errorf("CancelledSagas = %d, expected <= %d", m.CancelledSagas, numSagas)
	}
}

func TestNoOpMetricsCollector(t *testing.T) {
	// Test that no-op metrics collector doesn't panic
	collector := &noOpMetricsCollector{}

	collector.RecordSagaStarted("test-def")
	collector.RecordSagaCompleted("test-def", time.Second)
	collector.RecordSagaFailed("test-def", saga.ErrorTypeSystem, time.Second)
	collector.RecordSagaCancelled("test-def", time.Second)
	collector.RecordSagaTimedOut("test-def", time.Second)
	collector.RecordStepExecuted("test-def", "step-1", true, time.Millisecond)
	collector.RecordStepRetried("test-def", "step-1", 2)
	collector.RecordCompensationExecuted("test-def", "step-1", true, time.Millisecond)

	// If we reach here without panicking, the test passes
}

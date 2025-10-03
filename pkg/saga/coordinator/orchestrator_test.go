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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
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
}

func (m *mockStateStorage) SaveSaga(ctx context.Context, saga saga.SagaInstance) error {
	return m.saveSagaErr
}

func (m *mockStateStorage) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	return nil, m.getSagaErr
}

func (m *mockStateStorage) UpdateSagaState(ctx context.Context, sagaID string, state saga.SagaState, metadata map[string]interface{}) error {
	return m.updateSagaStateErr
}

func (m *mockStateStorage) DeleteSaga(ctx context.Context, sagaID string) error {
	return m.deleteSagaErr
}

func (m *mockStateStorage) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	if m.getActiveSagasErr != nil {
		return nil, m.getActiveSagasErr
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setupCoord()
			ctx := context.Background()

			_, err := coord.StartSaga(ctx, tt.definition, tt.initialData)
			if tt.wantErr {
				if err == nil {
					t.Errorf("StartSaga() expected error but got nil")
				}
				return
			}

			if err != nil && err.Error() != "not yet implemented: issue #507 will implement Saga instance creation" {
				t.Errorf("StartSaga() unexpected error = %v", err)
			}
		})
	}
}

func TestOrchestratorCoordinatorGetSagaInstance(t *testing.T) {
	tests := []struct {
		name        string
		setupCoord  func() *OrchestratorCoordinator
		sagaID      string
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
			sagaID:      "test-saga-id",
			wantErr:     true,
			expectedErr: ErrCoordinatorClosed,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setupCoord()

			_, err := coord.GetSagaInstance(tt.sagaID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetSagaInstance() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetSagaInstance() unexpected error = %v", err)
			}
		})
	}
}

func TestOrchestratorCoordinatorCancelSaga(t *testing.T) {
	tests := []struct {
		name        string
		setupCoord  func() *OrchestratorCoordinator
		sagaID      string
		reason      string
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
			sagaID:      "test-saga-id",
			reason:      "test cancellation",
			wantErr:     true,
			expectedErr: ErrCoordinatorClosed,
		},
		{
			name: "not implemented yet",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			sagaID:  "test-saga-id",
			reason:  "test cancellation",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setupCoord()
			ctx := context.Background()

			err := coord.CancelSaga(ctx, tt.sagaID, tt.reason)
			if tt.wantErr {
				if err == nil {
					t.Errorf("CancelSaga() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("CancelSaga() unexpected error = %v", err)
			}
		})
	}
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
	tests := []struct {
		name       string
		setupCoord func() *OrchestratorCoordinator
		sagaID     string
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
			sagaID:  "test-saga-id",
			wantErr: true,
		},
		{
			name: "not implemented yet",
			setupCoord: func() *OrchestratorCoordinator {
				config := newTestConfig()
				coord, _ := NewOrchestratorCoordinator(config)
				return coord
			},
			sagaID:  "test-saga-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := tt.setupCoord()
			ctx := context.Background()

			err := coord.StopSaga(ctx, tt.sagaID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("StopSaga() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("StopSaga() unexpected error = %v", err)
				}
			}
		})
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

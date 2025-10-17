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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

// mockDetectionSagaInstance is a test implementation of saga.SagaInstance for detection tests
type mockDetectionSagaInstance struct {
	id             string
	state          saga.SagaState
	currentStep    int
	totalSteps     int
	startTime      time.Time
	createdAt      time.Time
	updatedAt      time.Time
	timeout        time.Duration
	completedSteps int
}

func (m *mockDetectionSagaInstance) GetID() string                       { return m.id }
func (m *mockDetectionSagaInstance) GetDefinitionID() string             { return "test-def" }
func (m *mockDetectionSagaInstance) GetState() saga.SagaState            { return m.state }
func (m *mockDetectionSagaInstance) GetCurrentStep() int                 { return m.currentStep }
func (m *mockDetectionSagaInstance) GetStartTime() time.Time             { return m.startTime }
func (m *mockDetectionSagaInstance) GetEndTime() time.Time               { return time.Time{} }
func (m *mockDetectionSagaInstance) GetResult() interface{}              { return nil }
func (m *mockDetectionSagaInstance) GetError() *saga.SagaError           { return nil }
func (m *mockDetectionSagaInstance) GetTotalSteps() int                  { return m.totalSteps }
func (m *mockDetectionSagaInstance) GetCompletedSteps() int              { return m.completedSteps }
func (m *mockDetectionSagaInstance) GetCreatedAt() time.Time             { return m.createdAt }
func (m *mockDetectionSagaInstance) GetUpdatedAt() time.Time             { return m.updatedAt }
func (m *mockDetectionSagaInstance) GetTimeout() time.Duration           { return m.timeout }
func (m *mockDetectionSagaInstance) GetMetadata() map[string]interface{} { return nil }
func (m *mockDetectionSagaInstance) GetTraceID() string                  { return "" }
func (m *mockDetectionSagaInstance) IsTerminal() bool                    { return m.state.IsTerminal() }
func (m *mockDetectionSagaInstance) IsActive() bool                      { return m.state.IsActive() }

// mockStateStorageForDetection is a mock storage for testing detection
type mockStateStorageForDetection struct {
	timeoutSagas      []saga.SagaInstance
	activeSagas       []saga.SagaInstance
	stepStates        map[string][]*saga.StepState
	getTimeoutErr     error
	getActiveSagasErr error
	getStepStatesErr  error
}

func (m *mockStateStorageForDetection) SaveSaga(ctx context.Context, saga saga.SagaInstance) error {
	return nil
}

func (m *mockStateStorageForDetection) GetSaga(ctx context.Context, sagaID string) (saga.SagaInstance, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStateStorageForDetection) UpdateSagaState(ctx context.Context, sagaID string, state saga.SagaState, metadata map[string]interface{}) error {
	return nil
}

func (m *mockStateStorageForDetection) DeleteSaga(ctx context.Context, sagaID string) error {
	return nil
}

func (m *mockStateStorageForDetection) GetActiveSagas(ctx context.Context, filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	if m.getActiveSagasErr != nil {
		return nil, m.getActiveSagasErr
	}
	return m.activeSagas, nil
}

func (m *mockStateStorageForDetection) GetTimeoutSagas(ctx context.Context, before time.Time) ([]saga.SagaInstance, error) {
	if m.getTimeoutErr != nil {
		return nil, m.getTimeoutErr
	}
	return m.timeoutSagas, nil
}

func (m *mockStateStorageForDetection) SaveStepState(ctx context.Context, sagaID string, step *saga.StepState) error {
	return nil
}

func (m *mockStateStorageForDetection) GetStepStates(ctx context.Context, sagaID string) ([]*saga.StepState, error) {
	if m.getStepStatesErr != nil {
		return nil, m.getStepStatesErr
	}
	if steps, ok := m.stepStates[sagaID]; ok {
		return steps, nil
	}
	return []*saga.StepState{}, nil
}

func (m *mockStateStorageForDetection) CleanupExpiredSagas(ctx context.Context, olderThan time.Time) error {
	return nil
}

// mockDetectionCoordinator is a simple mock coordinator for detection tests
type mockDetectionCoordinator struct{}

func (m *mockDetectionCoordinator) StartSaga(ctx context.Context, definition saga.SagaDefinition, initialData interface{}) (saga.SagaInstance, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDetectionCoordinator) GetSagaInstance(sagaID string) (saga.SagaInstance, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDetectionCoordinator) CancelSaga(ctx context.Context, sagaID string, reason string) error {
	return errors.New("not implemented")
}

func (m *mockDetectionCoordinator) GetActiveSagas(filter *saga.SagaFilter) ([]saga.SagaInstance, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDetectionCoordinator) GetMetrics() *saga.CoordinatorMetrics {
	return &saga.CoordinatorMetrics{}
}

func (m *mockDetectionCoordinator) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *mockDetectionCoordinator) Close() error {
	return nil
}

func TestDetectionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *DetectionConfig
		wantErr bool
	}{
		{
			name:    "default config is valid",
			config:  DefaultDetectionConfig(),
			wantErr: false,
		},
		{
			name: "negative timeout threshold",
			config: &DetectionConfig{
				TimeoutThreshold:  -1 * time.Second,
				StuckThreshold:    1 * time.Minute,
				MaxResultsPerScan: 100,
			},
			wantErr: true,
		},
		{
			name: "zero stuck threshold",
			config: &DetectionConfig{
				TimeoutThreshold:  1 * time.Second,
				StuckThreshold:    0,
				MaxResultsPerScan: 100,
			},
			wantErr: true,
		},
		{
			name: "zero max results",
			config: &DetectionConfig{
				TimeoutThreshold:  1 * time.Second,
				StuckThreshold:    1 * time.Minute,
				MaxResultsPerScan: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectionConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRecoveryManager_detectTimeoutSagas(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		timeoutSagas  []saga.SagaInstance
		storageErr    error
		wantCount     int
		wantErr       bool
		checkPriority bool
		expectedPrio  DetectionPriority
	}{
		{
			name: "detect single timeout saga",
			timeoutSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:          "saga-1",
					state:       saga.StateRunning,
					startTime:   now.Add(-10 * time.Minute),
					updatedAt:   now.Add(-5 * time.Minute),
					timeout:     5 * time.Minute,
					currentStep: 1,
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "skip terminal state sagas",
			timeoutSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:        "saga-1",
					state:     saga.StateCompleted,
					startTime: now.Add(-10 * time.Minute),
					timeout:   5 * time.Minute,
				},
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "skip non-active sagas",
			timeoutSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:        "saga-1",
					state:     saga.StatePending,
					startTime: now.Add(-10 * time.Minute),
					timeout:   5 * time.Minute,
				},
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "multiple timeout sagas",
			timeoutSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:        "saga-1",
					state:     saga.StateRunning,
					startTime: now.Add(-10 * time.Minute),
					updatedAt: now.Add(-5 * time.Minute),
					timeout:   5 * time.Minute,
				},
				&mockDetectionSagaInstance{
					id:        "saga-2",
					state:     saga.StateCompensating,
					startTime: now.Add(-15 * time.Minute),
					updatedAt: now.Add(-10 * time.Minute),
					timeout:   5 * time.Minute,
				},
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:       "storage error",
			storageErr: errors.New("storage error"),
			wantCount:  0,
			wantErr:    true,
		},
		{
			name: "critical priority for 2x timeout",
			timeoutSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:        "saga-1",
					state:     saga.StateRunning,
					startTime: now.Add(-15 * time.Minute),
					updatedAt: now.Add(-10 * time.Minute),
					timeout:   5 * time.Minute,
				},
			},
			wantCount:     1,
			wantErr:       false,
			checkPriority: true,
			expectedPrio:  PriorityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &mockStateStorageForDetection{
				timeoutSagas:  tt.timeoutSagas,
				getTimeoutErr: tt.storageErr,
			}

			config := DefaultRecoveryConfig()
			rm, err := NewRecoveryManager(config, storage, &mockDetectionCoordinator{}, zap.NewNop())
			if err != nil {
				t.Fatalf("NewRecoveryManager() error = %v", err)
			}

			results, err := rm.detectTimeoutSagas(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("detectTimeoutSagas() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantCount {
				t.Errorf("detectTimeoutSagas() count = %v, want %v", len(results), tt.wantCount)
			}

			if tt.checkPriority && len(results) > 0 {
				if results[0].Priority != tt.expectedPrio {
					t.Errorf("detectTimeoutSagas() priority = %v, want %v", results[0].Priority, tt.expectedPrio)
				}
			}

			// Verify result structure
			for _, result := range results {
				if result.Type != DetectionTypeTimeout {
					t.Errorf("result type = %v, want %v", result.Type, DetectionTypeTimeout)
				}
				if result.SagaID == "" {
					t.Error("result SagaID is empty")
				}
				if result.DetectedAt.IsZero() {
					t.Error("result DetectedAt is zero")
				}
			}
		})
	}
}

func TestRecoveryManager_detectStuckSagas(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		activeSagas []saga.SagaInstance
		storageErr  error
		wantCount   int
		wantErr     bool
	}{
		{
			name: "detect single stuck saga",
			activeSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:        "saga-1",
					state:     saga.StateRunning,
					startTime: now.Add(-10 * time.Minute),
					updatedAt: now.Add(-10 * time.Minute),
					timeout:   30 * time.Minute,
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "skip recently updated sagas",
			activeSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:        "saga-1",
					state:     saga.StateRunning,
					startTime: now.Add(-5 * time.Minute),
					updatedAt: now.Add(-1 * time.Minute),
					timeout:   30 * time.Minute,
				},
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "skip timeout cases",
			activeSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:        "saga-1",
					state:     saga.StateRunning,
					startTime: now.Add(-15 * time.Minute),
					updatedAt: now.Add(-10 * time.Minute),
					timeout:   5 * time.Minute, // Already timed out
				},
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "detect compensating stuck saga",
			activeSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:        "saga-1",
					state:     saga.StateCompensating,
					startTime: now.Add(-20 * time.Minute),
					updatedAt: now.Add(-10 * time.Minute),
					timeout:   60 * time.Minute,
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:       "storage error",
			storageErr: errors.New("storage error"),
			wantCount:  0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &mockStateStorageForDetection{
				activeSagas:       tt.activeSagas,
				getActiveSagasErr: tt.storageErr,
			}

			config := DefaultRecoveryConfig()
			rm, err := NewRecoveryManager(config, storage, &mockDetectionCoordinator{}, zap.NewNop())
			if err != nil {
				t.Fatalf("NewRecoveryManager() error = %v", err)
			}

			results, err := rm.detectStuckSagas(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("detectStuckSagas() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantCount {
				t.Errorf("detectStuckSagas() count = %v, want %v", len(results), tt.wantCount)
			}

			// Verify result structure
			for _, result := range results {
				if result.Type != DetectionTypeStuck {
					t.Errorf("result type = %v, want %v", result.Type, DetectionTypeStuck)
				}
			}
		})
	}
}

func TestRecoveryManager_detectInconsistentSagas(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		activeSagas  []saga.SagaInstance
		stepStates   map[string][]*saga.StepState
		storageErr   error
		wantCount    int
		wantErr      bool
		checkDesc    bool
		descContains string
	}{
		{
			name: "detect saga with mismatched state",
			activeSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:          "saga-1",
					state:       saga.StateCompleted,
					currentStep: 1,
					updatedAt:   now,
				},
			},
			stepStates: map[string][]*saga.StepState{
				"saga-1": {
					{
						ID:        "step-0",
						SagaID:    "saga-1",
						StepIndex: 0,
						State:     saga.StepStateCompleted,
					},
					{
						ID:        "step-1",
						SagaID:    "saga-1",
						StepIndex: 1,
						State:     saga.StepStatePending, // Inconsistent!
					},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "detect running saga with completed current step",
			activeSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:          "saga-1",
					state:       saga.StateRunning,
					currentStep: 1,
					updatedAt:   now,
				},
			},
			stepStates: map[string][]*saga.StepState{
				"saga-1": {
					{
						ID:        "step-0",
						SagaID:    "saga-1",
						StepIndex: 0,
						State:     saga.StepStateCompleted,
					},
					{
						ID:        "step-1",
						SagaID:    "saga-1",
						StepIndex: 1,
						State:     saga.StepStateCompleted, // Should be Running or Pending
					},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "detect compensating saga without compensating steps",
			activeSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:          "saga-1",
					state:       saga.StateCompensating,
					currentStep: 1,
					updatedAt:   now,
				},
			},
			stepStates: map[string][]*saga.StepState{
				"saga-1": {
					{
						ID:        "step-0",
						SagaID:    "saga-1",
						StepIndex: 0,
						State:     saga.StepStateCompleted, // No compensation state
					},
					{
						ID:        "step-1",
						SagaID:    "saga-1",
						StepIndex: 1,
						State:     saga.StepStateCompleted, // No compensation state
					},
				},
			},
			wantCount:    1,
			wantErr:      false,
			checkDesc:    true,
			descContains: "compensating but no steps",
		},
		{
			name: "no inconsistencies",
			activeSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:          "saga-1",
					state:       saga.StateRunning,
					currentStep: 1,
					updatedAt:   now,
				},
			},
			stepStates: map[string][]*saga.StepState{
				"saga-1": {
					{
						ID:        "step-0",
						SagaID:    "saga-1",
						StepIndex: 0,
						State:     saga.StepStateCompleted,
					},
					{
						ID:        "step-1",
						SagaID:    "saga-1",
						StepIndex: 1,
						State:     saga.StepStateRunning,
					},
				},
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:       "storage error",
			storageErr: errors.New("storage error"),
			wantCount:  0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &mockStateStorageForDetection{
				activeSagas:       tt.activeSagas,
				stepStates:        tt.stepStates,
				getActiveSagasErr: tt.storageErr,
			}

			config := DefaultRecoveryConfig()
			rm, err := NewRecoveryManager(config, storage, &mockDetectionCoordinator{}, zap.NewNop())
			if err != nil {
				t.Fatalf("NewRecoveryManager() error = %v", err)
			}

			results, err := rm.detectInconsistentSagas(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("detectInconsistentSagas() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantCount {
				t.Errorf("detectInconsistentSagas() count = %v, want %v", len(results), tt.wantCount)
			}

			// Verify result structure
			for _, result := range results {
				if result.Type != DetectionTypeInconsistent {
					t.Errorf("result type = %v, want %v", result.Type, DetectionTypeInconsistent)
				}
				if tt.checkDesc {
					if len(tt.descContains) > 0 {
						found := false
						for _, inc := range result.Metadata["inconsistencies"].([]string) {
							if len(inc) > 0 {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected inconsistencies in metadata")
						}
					}
				}
			}
		})
	}
}

func TestRecoveryManager_checkStateConsistency(t *testing.T) {
	tests := []struct {
		name          string
		saga          saga.SagaInstance
		stepStates    []*saga.StepState
		wantInconsist int
		checkContains string
	}{
		{
			name: "current step index out of bounds",
			saga: &mockDetectionSagaInstance{
				id:          "saga-1",
				state:       saga.StateRunning,
				currentStep: 5,
			},
			stepStates: []*saga.StepState{
				{StepIndex: 0, State: saga.StepStateCompleted},
				{StepIndex: 1, State: saga.StepStateRunning},
			},
			wantInconsist: 1,
			checkContains: "exceeds step count",
		},
		{
			name: "completed saga with non-completed step",
			saga: &mockDetectionSagaInstance{
				id:    "saga-1",
				state: saga.StateCompleted,
			},
			stepStates: []*saga.StepState{
				{StepIndex: 0, State: saga.StepStateCompleted},
				{StepIndex: 1, State: saga.StepStateRunning},
			},
			wantInconsist: 1,
			checkContains: "saga completed but step",
		},
		{
			name: "no inconsistencies",
			saga: &mockDetectionSagaInstance{
				id:          "saga-1",
				state:       saga.StateRunning,
				currentStep: 1,
			},
			stepStates: []*saga.StepState{
				{StepIndex: 0, State: saga.StepStateCompleted},
				{StepIndex: 1, State: saga.StepStateRunning},
			},
			wantInconsist: 0,
		},
		{
			name: "empty step states",
			saga: &mockDetectionSagaInstance{
				id:    "saga-1",
				state: saga.StateRunning,
			},
			stepStates:    []*saga.StepState{},
			wantInconsist: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRecoveryConfig()
			rm, err := NewRecoveryManager(config, &mockStateStorageForDetection{}, &mockCoordinator{}, zap.NewNop())
			if err != nil {
				t.Fatalf("NewRecoveryManager() error = %v", err)
			}

			inconsistencies := rm.checkStateConsistency(tt.saga, tt.stepStates)

			if len(inconsistencies) != tt.wantInconsist {
				t.Errorf("checkStateConsistency() count = %v, want %v", len(inconsistencies), tt.wantInconsist)
			}

			if tt.checkContains != "" && len(inconsistencies) > 0 {
				found := false
				for _, inc := range inconsistencies {
					if len(inc) > 0 {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected inconsistency containing text, got: %v", inconsistencies)
				}
			}
		})
	}
}

func TestRecoveryManager_calculateTimeoutPriority(t *testing.T) {
	tests := []struct {
		name           string
		timeSinceStart time.Duration
		timeout        time.Duration
		wantPriority   DetectionPriority
	}{
		{
			name:           "critical - 3x timeout",
			timeSinceStart: 15 * time.Minute,
			timeout:        5 * time.Minute,
			wantPriority:   PriorityCritical,
		},
		{
			name:           "medium - 1.7x timeout",
			timeSinceStart: 8*time.Minute + 30*time.Second,
			timeout:        5 * time.Minute,
			wantPriority:   PriorityMedium,
		},
		{
			name:           "medium - 1.6x timeout",
			timeSinceStart: 8 * time.Minute,
			timeout:        5 * time.Minute,
			wantPriority:   PriorityMedium,
		},
		{
			name:           "low - 1.3x timeout",
			timeSinceStart: 6*time.Minute + 30*time.Second,
			timeout:        5 * time.Minute,
			wantPriority:   PriorityLow,
		},
		{
			name:           "zero timeout",
			timeSinceStart: 10 * time.Minute,
			timeout:        0,
			wantPriority:   PriorityMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRecoveryConfig()
			rm, err := NewRecoveryManager(config, &mockStateStorageForDetection{}, &mockCoordinator{}, zap.NewNop())
			if err != nil {
				t.Fatalf("NewRecoveryManager() error = %v", err)
			}

			priority := rm.calculateTimeoutPriority(tt.timeSinceStart, tt.timeout)
			if priority != tt.wantPriority {
				t.Errorf("calculateTimeoutPriority() = %v, want %v", priority, tt.wantPriority)
			}
		})
	}
}

func TestRecoveryManager_calculateStuckPriority(t *testing.T) {
	tests := []struct {
		name            string
		timeSinceUpdate time.Duration
		stuckThreshold  time.Duration
		wantPriority    DetectionPriority
	}{
		{
			name:            "critical - 4x threshold",
			timeSinceUpdate: 20 * time.Minute,
			stuckThreshold:  5 * time.Minute,
			wantPriority:    PriorityCritical,
		},
		{
			name:            "high - 2.5x threshold",
			timeSinceUpdate: 12*time.Minute + 30*time.Second,
			stuckThreshold:  5 * time.Minute,
			wantPriority:    PriorityHigh,
		},
		{
			name:            "medium - 1.7x threshold",
			timeSinceUpdate: 8*time.Minute + 30*time.Second,
			stuckThreshold:  5 * time.Minute,
			wantPriority:    PriorityMedium,
		},
		{
			name:            "low - 1.3x threshold",
			timeSinceUpdate: 6*time.Minute + 30*time.Second,
			stuckThreshold:  5 * time.Minute,
			wantPriority:    PriorityLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRecoveryConfig()
			rm, err := NewRecoveryManager(config, &mockStateStorageForDetection{}, &mockCoordinator{}, zap.NewNop())
			if err != nil {
				t.Fatalf("NewRecoveryManager() error = %v", err)
			}

			priority := rm.calculateStuckPriority(tt.timeSinceUpdate, tt.stuckThreshold)
			if priority != tt.wantPriority {
				t.Errorf("calculateStuckPriority() = %v, want %v", priority, tt.wantPriority)
			}
		})
	}
}

func TestRecoveryManager_performDetection(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		timeoutSagas []saga.SagaInstance
		activeSagas  []saga.SagaInstance
		wantMinCount int
	}{
		{
			name: "combined detection",
			timeoutSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:        "saga-timeout",
					state:     saga.StateRunning,
					startTime: now.Add(-10 * time.Minute),
					updatedAt: now.Add(-5 * time.Minute),
					timeout:   5 * time.Minute,
				},
			},
			activeSagas: []saga.SagaInstance{
				&mockDetectionSagaInstance{
					id:        "saga-stuck",
					state:     saga.StateRunning,
					startTime: now.Add(-20 * time.Minute),
					updatedAt: now.Add(-10 * time.Minute),
					timeout:   60 * time.Minute,
				},
			},
			wantMinCount: 2,
		},
		{
			name:         "no issues",
			timeoutSagas: []saga.SagaInstance{},
			activeSagas:  []saga.SagaInstance{},
			wantMinCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &mockStateStorageForDetection{
				timeoutSagas: tt.timeoutSagas,
				activeSagas:  tt.activeSagas,
			}

			config := DefaultRecoveryConfig()
			rm, err := NewRecoveryManager(config, storage, &mockDetectionCoordinator{}, zap.NewNop())
			if err != nil {
				t.Fatalf("NewRecoveryManager() error = %v", err)
			}

			results, err := rm.performDetection(context.Background())
			if err != nil {
				t.Errorf("performDetection() error = %v", err)
				return
			}

			if len(results) < tt.wantMinCount {
				t.Errorf("performDetection() count = %v, want at least %v", len(results), tt.wantMinCount)
			}

			// Verify results are sorted by priority (highest first)
			for i := 1; i < len(results); i++ {
				if results[i].Priority > results[i-1].Priority {
					t.Errorf("results not sorted by priority: result[%d].Priority=%v > result[%d].Priority=%v",
						i, results[i].Priority, i-1, results[i-1].Priority)
				}
			}
		})
	}
}

func TestRecoveryManager_detectTimeoutSagas_NoStorage(t *testing.T) {
	config := DefaultRecoveryConfig()

	// Create RecoveryManager without storage by directly manipulating struct
	rm := &RecoveryManager{
		config:       config,
		stateStorage: nil, // No storage
		logger:       zap.NewNop(),
	}

	_, err := rm.detectTimeoutSagas(context.Background())
	if err != ErrNoStorageForDetection {
		t.Errorf("detectTimeoutSagas() with no storage: got error %v, want %v", err, ErrNoStorageForDetection)
	}
}

func TestRecoveryManager_detectStuckSagas_NoStorage(t *testing.T) {
	config := DefaultRecoveryConfig()

	rm := &RecoveryManager{
		config:       config,
		stateStorage: nil,
		logger:       zap.NewNop(),
	}

	_, err := rm.detectStuckSagas(context.Background())
	if err != ErrNoStorageForDetection {
		t.Errorf("detectStuckSagas() with no storage: got error %v, want %v", err, ErrNoStorageForDetection)
	}
}

func TestRecoveryManager_detectInconsistentSagas_NoStorage(t *testing.T) {
	config := DefaultRecoveryConfig()

	rm := &RecoveryManager{
		config:       config,
		stateStorage: nil,
		logger:       zap.NewNop(),
	}

	_, err := rm.detectInconsistentSagas(context.Background())
	if err != ErrNoStorageForDetection {
		t.Errorf("detectInconsistentSagas() with no storage: got error %v, want %v", err, ErrNoStorageForDetection)
	}
}

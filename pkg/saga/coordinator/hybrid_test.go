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
	"testing"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestNewHybridCoordinator tests the creation of a hybrid coordinator.
func TestNewHybridCoordinator(t *testing.T) {
	// Create properly initialized test coordinators
	mockStorage := &mockStateStorage{}
	mockPublisher := &mockEventPublisher{}

	validOrchestrator, _ := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:   mockStorage,
		EventPublisher: mockPublisher,
	})
	validChoreography, _ := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: mockPublisher,
	})

	tests := []struct {
		name    string
		config  *HybridConfig
		wantErr bool
		errType error
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errType: nil,
		},
		{
			name: "missing orchestrator in auto mode",
			config: &HybridConfig{
				Orchestrator: nil,
				Choreography: validChoreography,
				ModeSelector: NewManualModeSelector(ModeOrchestration, "test"),
			},
			wantErr: true,
			errType: ErrOrchestratorNotConfigured,
		},
		{
			name: "missing choreography in auto mode",
			config: &HybridConfig{
				Orchestrator: validOrchestrator,
				Choreography: nil,
				ModeSelector: NewManualModeSelector(ModeOrchestration, "test"),
			},
			wantErr: true,
			errType: ErrChoreographyNotConfigured,
		},
		{
			name: "missing mode selector in auto mode",
			config: &HybridConfig{
				Orchestrator: validOrchestrator,
				Choreography: validChoreography,
				ModeSelector: nil,
			},
			wantErr: true,
			errType: ErrModeSelectorNotConfigured,
		},
		{
			name: "valid auto mode configuration",
			config: &HybridConfig{
				Orchestrator: validOrchestrator,
				Choreography: validChoreography,
				ModeSelector: NewManualModeSelector(ModeOrchestration, "test"),
			},
			wantErr: false,
		},
		{
			name: "valid forced orchestration mode",
			config: &HybridConfig{
				Orchestrator: validOrchestrator,
				ForceMode:    ModeOrchestration,
			},
			wantErr: false,
		},
		{
			name: "valid forced choreography mode",
			config: &HybridConfig{
				Choreography: validChoreography,
				ForceMode:    ModeChoreography,
			},
			wantErr: false,
		},
		{
			name: "unsupported mode",
			config: &HybridConfig{
				Orchestrator: validOrchestrator,
				ForceMode:    "invalid",
			},
			wantErr: true,
			errType: ErrUnsupportedMode,
		},
	}

	defer validOrchestrator.Close()
	defer validChoreography.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordinator, err := NewHybridCoordinator(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHybridCoordinator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("NewHybridCoordinator() error = %v, want error type %v", err, tt.errType)
				}
			}
			if !tt.wantErr && coordinator == nil {
				t.Error("NewHybridCoordinator() returned nil coordinator")
			}
			if !tt.wantErr && coordinator != nil {
				// Clean up
				_ = coordinator.Close()
			}
		})
	}
}

// TestHybridCoordinatorStartSaga tests starting a Saga with mode selection.
func TestHybridCoordinatorStartSaga(t *testing.T) {
	// Create mock coordinators
	mockStorage := &mockStateStorage{}
	mockPublisher := &mockEventPublisher{}

	orchestrator, err := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:   mockStorage,
		EventPublisher: mockPublisher,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}
	defer orchestrator.Close()

	choreography, err := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: mockPublisher,
	})
	if err != nil {
		t.Fatalf("failed to create choreography coordinator: %v", err)
	}
	defer choreography.Close()

	// Start choreography coordinator
	ctx := context.Background()
	if err := choreography.Start(ctx); err != nil {
		t.Fatalf("failed to start choreography coordinator: %v", err)
	}

	tests := []struct {
		name       string
		selector   ModeSelector
		definition saga.SagaDefinition
		wantMode   CoordinationMode
		wantErr    bool
	}{
		{
			name:     "selects orchestration for simple workflow",
			selector: NewManualModeSelector(ModeOrchestration, "test"),
			definition: &mockSagaDefinition{
				id:   "simple-saga",
				name: "Simple Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
				},
			},
			wantMode: ModeOrchestration,
			wantErr:  false,
		},
		{
			name:     "selects choreography for complex workflow",
			selector: NewManualModeSelector(ModeChoreography, "test"),
			definition: &mockSagaDefinition{
				id:   "complex-saga",
				name: "Complex Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
					&mockSagaStep{id: "step2"},
				},
			},
			wantMode: ModeChoreography,
			wantErr:  false,
		},
		{
			name:       "fails with nil definition",
			selector:   NewManualModeSelector(ModeOrchestration, "test"),
			definition: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh choreography coordinator for this test case
			testChoreography, _ := NewChoreographyCoordinator(&ChoreographyConfig{
				EventPublisher: mockPublisher,
			})
			defer testChoreography.Close()

			// Start it if needed for choreography mode
			if tt.wantMode == ModeChoreography {
				if err := testChoreography.Start(ctx); err != nil {
					t.Fatalf("failed to start test choreography coordinator: %v", err)
				}
			}

			coordinator, err := NewHybridCoordinator(&HybridConfig{
				Orchestrator: orchestrator,
				Choreography: testChoreography,
				ModeSelector: tt.selector,
			})
			if err != nil {
				t.Fatalf("NewHybridCoordinator() error = %v", err)
			}
			defer coordinator.Close()

			instance, err := coordinator.StartSaga(ctx, tt.definition, map[string]interface{}{"test": "data"})

			if (err != nil) != tt.wantErr {
				t.Errorf("StartSaga() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if instance == nil {
					t.Error("StartSaga() returned nil instance")
					return
				}

				// Verify mode was tracked
				mode, exists := coordinator.GetModeForSaga(instance.GetID())
				if !exists {
					t.Error("mode was not tracked for Saga")
				}
				if mode != tt.wantMode {
					t.Errorf("tracked mode = %v, want %v", mode, tt.wantMode)
				}

				// Verify metrics were updated
				metrics := coordinator.GetHybridMetrics()
				if metrics.TotalSagas == 0 {
					t.Error("metrics not updated after starting Saga")
				}
			}
		})
	}
}

// TestHybridCoordinatorModeSelection tests mode selection logic.
func TestHybridCoordinatorModeSelection(t *testing.T) {
	mockStorage := &mockStateStorage{}
	mockPublisher := &mockEventPublisher{}

	orchestrator, _ := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:   mockStorage,
		EventPublisher: mockPublisher,
	})
	defer orchestrator.Close()

	choreography, _ := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: mockPublisher,
	})
	defer choreography.Close()

	tests := []struct {
		name      string
		forceMode CoordinationMode
		metadata  map[string]interface{}
		wantMode  CoordinationMode
	}{
		{
			name:      "forced orchestration mode",
			forceMode: ModeOrchestration,
			wantMode:  ModeOrchestration,
		},
		{
			name:      "forced choreography mode",
			forceMode: ModeChoreography,
			wantMode:  ModeChoreography,
		},
		{
			name:      "explicit mode in metadata",
			forceMode: "",
			metadata: map[string]interface{}{
				"coordination_mode": "choreography",
			},
			wantMode: ModeChoreography,
		},
		{
			name:      "selector decides mode",
			forceMode: "",
			metadata:  nil,
			wantMode:  ModeOrchestration, // based on smart selector default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordinator, err := NewHybridCoordinator(&HybridConfig{
				Orchestrator: orchestrator,
				Choreography: choreography,
				ModeSelector: NewSmartModeSelector([]ModeSelectionRule{
					NewSimpleLinearRule(5),
				}, ModeOrchestration),
				ForceMode: tt.forceMode,
			})
			if err != nil {
				t.Fatalf("NewHybridCoordinator() error = %v", err)
			}
			defer coordinator.Close()

			definition := &mockSagaDefinition{
				id:   "test-saga",
				name: "Test Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
				},
				metadata: tt.metadata,
			}

			mode := coordinator.selectMode(definition)
			if mode != tt.wantMode {
				t.Errorf("selectMode() = %v, want %v", mode, tt.wantMode)
			}
		})
	}
}

// TestHybridCoordinatorGetSagaInstance tests retrieving Saga instances.
func TestHybridCoordinatorGetSagaInstance(t *testing.T) {
	mockStorage := &mockStateStorage{}
	mockPublisher := &mockEventPublisher{}

	orchestrator, _ := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:   mockStorage,
		EventPublisher: mockPublisher,
	})
	defer orchestrator.Close()

	choreography, _ := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: mockPublisher,
	})
	defer choreography.Close()

	coordinator, err := NewHybridCoordinator(&HybridConfig{
		Orchestrator: orchestrator,
		Choreography: choreography,
		ModeSelector: NewManualModeSelector(ModeOrchestration, "test"),
	})
	if err != nil {
		t.Fatalf("NewHybridCoordinator() error = %v", err)
	}
	defer coordinator.Close()

	// Try to get non-existent Saga
	_, err = coordinator.GetSagaInstance("non-existent")
	if err == nil {
		t.Error("GetSagaInstance() should return error for non-existent Saga")
	}

	// Close coordinator and try again
	_ = coordinator.Close()
	_, err = coordinator.GetSagaInstance("test-saga")
	if !errors.Is(err, ErrHybridCoordinatorClosed) {
		t.Errorf("GetSagaInstance() after close should return ErrHybridCoordinatorClosed, got %v", err)
	}
}

// TestHybridCoordinatorCancelSaga tests canceling Saga instances.
func TestHybridCoordinatorCancelSaga(t *testing.T) {
	mockStorage := &mockStateStorage{}
	mockPublisher := &mockEventPublisher{}

	orchestrator, _ := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:   mockStorage,
		EventPublisher: mockPublisher,
	})
	defer orchestrator.Close()

	choreography, _ := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: mockPublisher,
	})
	defer choreography.Close()

	coordinator, err := NewHybridCoordinator(&HybridConfig{
		Orchestrator: orchestrator,
		Choreography: choreography,
		ModeSelector: NewManualModeSelector(ModeOrchestration, "test"),
	})
	if err != nil {
		t.Fatalf("NewHybridCoordinator() error = %v", err)
	}
	defer coordinator.Close()

	ctx := context.Background()

	// Try to cancel non-existent Saga
	err = coordinator.CancelSaga(ctx, "non-existent", "test reason")
	if err == nil {
		t.Error("CancelSaga() should return error for non-existent Saga")
	}
}

// TestHybridCoordinatorGetActiveSagas tests retrieving active Sagas.
func TestHybridCoordinatorGetActiveSagas(t *testing.T) {
	mockStorage := &mockStateStorage{}
	mockPublisher := &mockEventPublisher{}

	orchestrator, _ := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:   mockStorage,
		EventPublisher: mockPublisher,
	})
	defer orchestrator.Close()

	choreography, _ := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: mockPublisher,
	})
	defer choreography.Close()

	coordinator, err := NewHybridCoordinator(&HybridConfig{
		Orchestrator: orchestrator,
		Choreography: choreography,
		ModeSelector: NewManualModeSelector(ModeOrchestration, "test"),
	})
	if err != nil {
		t.Fatalf("NewHybridCoordinator() error = %v", err)
	}
	defer coordinator.Close()

	// Get active Sagas
	instances, err := coordinator.GetActiveSagas(nil)
	if err != nil {
		t.Errorf("GetActiveSagas() error = %v", err)
	}
	// instances should be an empty slice, not nil
	if len(instances) != 0 {
		t.Errorf("GetActiveSagas() expected empty slice, got %d instances", len(instances))
	}
}

// TestHybridCoordinatorGetMetrics tests retrieving coordinator metrics.
func TestHybridCoordinatorGetMetrics(t *testing.T) {
	mockStorage := &mockStateStorage{}
	mockPublisher := &mockEventPublisher{}

	orchestrator, _ := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:   mockStorage,
		EventPublisher: mockPublisher,
	})
	defer orchestrator.Close()

	choreography, _ := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: mockPublisher,
	})
	defer choreography.Close()

	coordinator, err := NewHybridCoordinator(&HybridConfig{
		Orchestrator: orchestrator,
		Choreography: choreography,
		ModeSelector: NewManualModeSelector(ModeOrchestration, "test"),
	})
	if err != nil {
		t.Fatalf("NewHybridCoordinator() error = %v", err)
	}
	defer coordinator.Close()

	// Get standard metrics
	metrics := coordinator.GetMetrics()
	if metrics == nil {
		t.Error("GetMetrics() returned nil")
	}

	// Get hybrid-specific metrics
	hybridMetrics := coordinator.GetHybridMetrics()
	if hybridMetrics == nil {
		t.Error("GetHybridMetrics() returned nil")
	}
}

// TestHybridCoordinatorHealthCheck tests health checking.
func TestHybridCoordinatorHealthCheck(t *testing.T) {
	mockStorage := &mockStateStorage{}
	mockPublisher := &mockEventPublisher{}

	orchestrator, _ := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:   mockStorage,
		EventPublisher: mockPublisher,
	})
	defer orchestrator.Close()

	choreography, _ := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: mockPublisher,
	})
	defer choreography.Close()

	// Start choreography coordinator for health check
	ctx := context.Background()
	if err := choreography.Start(ctx); err != nil {
		t.Fatalf("failed to start choreography coordinator: %v", err)
	}

	coordinator, err := NewHybridCoordinator(&HybridConfig{
		Orchestrator: orchestrator,
		Choreography: choreography,
		ModeSelector: NewManualModeSelector(ModeOrchestration, "test"),
	})
	if err != nil {
		t.Fatalf("NewHybridCoordinator() error = %v", err)
	}

	// Health check should pass initially
	if err := coordinator.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}

	// Close coordinator
	_ = coordinator.Close()

	// Health check should fail after close
	if err := coordinator.HealthCheck(ctx); !errors.Is(err, ErrHybridCoordinatorClosed) {
		t.Errorf("HealthCheck() after close should return ErrHybridCoordinatorClosed, got %v", err)
	}
}

// TestHybridCoordinatorClose tests closing the coordinator.
func TestHybridCoordinatorClose(t *testing.T) {
	mockStorage := &mockStateStorage{}
	mockPublisher := &mockEventPublisher{}

	orchestrator, _ := NewOrchestratorCoordinator(&OrchestratorConfig{
		StateStorage:   mockStorage,
		EventPublisher: mockPublisher,
	})

	choreography, _ := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: mockPublisher,
	})

	coordinator, err := NewHybridCoordinator(&HybridConfig{
		Orchestrator: orchestrator,
		Choreography: choreography,
		ModeSelector: NewManualModeSelector(ModeOrchestration, "test"),
	})
	if err != nil {
		t.Fatalf("NewHybridCoordinator() error = %v", err)
	}

	// Close should succeed
	if err := coordinator.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Second close should return error
	if err := coordinator.Close(); !errors.Is(err, ErrHybridCoordinatorClosed) {
		t.Errorf("Close() second call should return ErrHybridCoordinatorClosed, got %v", err)
	}
}

// TestHybridConfigValidate tests configuration validation.
func TestHybridConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *HybridConfig
		wantErr bool
		errType error
	}{
		{
			name: "valid auto mode",
			config: &HybridConfig{
				Orchestrator: &OrchestratorCoordinator{},
				Choreography: &ChoreographyCoordinator{},
				ModeSelector: NewManualModeSelector(ModeOrchestration, "test"),
			},
			wantErr: false,
		},
		{
			name: "missing orchestrator in auto mode",
			config: &HybridConfig{
				Choreography: &ChoreographyCoordinator{},
				ModeSelector: NewManualModeSelector(ModeOrchestration, "test"),
			},
			wantErr: true,
			errType: ErrOrchestratorNotConfigured,
		},
		{
			name: "missing choreography in auto mode",
			config: &HybridConfig{
				Orchestrator: &OrchestratorCoordinator{},
				ModeSelector: NewManualModeSelector(ModeOrchestration, "test"),
			},
			wantErr: true,
			errType: ErrChoreographyNotConfigured,
		},
		{
			name: "missing selector in auto mode",
			config: &HybridConfig{
				Orchestrator: &OrchestratorCoordinator{},
				Choreography: &ChoreographyCoordinator{},
			},
			wantErr: true,
			errType: ErrModeSelectorNotConfigured,
		},
		{
			name: "valid forced orchestration",
			config: &HybridConfig{
				Orchestrator: &OrchestratorCoordinator{},
				ForceMode:    ModeOrchestration,
			},
			wantErr: false,
		},
		{
			name: "valid forced choreography",
			config: &HybridConfig{
				Choreography: &ChoreographyCoordinator{},
				ForceMode:    ModeChoreography,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Validate() error = %v, want error type %v", err, tt.errType)
				}
			}
		})
	}
}

// TestHybridCoordinatorChoreographyUniqueIDs tests that multiple choreography saga instances
// get unique saga IDs instead of sharing the same definition-based ID.
func TestHybridCoordinatorChoreographyUniqueIDs(t *testing.T) {
	// Create mock coordinators
	mockPublisher := &mockEventPublisher{}

	choreography, err := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: mockPublisher,
	})
	if err != nil {
		t.Fatalf("failed to create choreography coordinator: %v", err)
	}
	defer choreography.Close()

	// Create hybrid coordinator with forced choreography mode
	hybrid, err := NewHybridCoordinator(&HybridConfig{
		Choreography:  choreography,
		ForceMode:     ModeChoreography,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("failed to create hybrid coordinator: %v", err)
	}
	defer hybrid.Close()

	// Create a simple saga definition
	definition := &mockSagaDefinition{
		id: "test-saga-definition",
	}

	// Start multiple choreography saga instances concurrently
	const numInstances = 5
	instances := make([]saga.SagaInstance, numInstances)
	instanceIDs := make([]string, numInstances)

	for i := 0; i < numInstances; i++ {
		instance, err := hybrid.StartSaga(context.Background(), definition, "test-data")
		if err != nil {
			t.Fatalf("failed to start saga instance %d: %v", i, err)
		}
		instances[i] = instance
		instanceIDs[i] = instance.GetID()
	}

	// Verify that all instance IDs are unique
	idSet := make(map[string]bool)
	for i, id := range instanceIDs {
		if idSet[id] {
			t.Errorf("duplicate saga ID found: %s (instance %d)", id, i)
		}
		idSet[id] = true

		// Verify that the ID is not just the definition-based ID
		expectedDefID := fmt.Sprintf("saga-%s", definition.GetID())
		if id == expectedDefID {
			t.Errorf("saga ID %q should not be equal to definition-based ID %q", id, expectedDefID)
		}

		// Verify that the ID follows the expected pattern from generateSagaID
		if !matchesSagaIDPattern(id) {
			t.Errorf("saga ID %q does not match expected saga ID pattern", id)
		}
	}

	// Verify that mode tracking works correctly with unique IDs
	for i, id := range instanceIDs {
		mode, exists := hybrid.GetModeForSaga(id)
		if !exists {
			t.Errorf("mode tracking not found for saga instance %d with ID %s", i, id)
		} else if mode != ModeChoreography {
			t.Errorf("expected mode %s for saga instance %d, got %s", ModeChoreography, i, mode)
		}
	}

	// Verify metrics count
	metrics := hybrid.GetHybridMetrics()
	if metrics.ChoreographySagas != int64(numInstances) {
		t.Errorf("expected %d choreography sagas in metrics, got %d", numInstances, metrics.ChoreographySagas)
	}
}

// matchesSagaIDPattern checks if a saga ID follows the pattern from generateSagaID
func matchesSagaIDPattern(id string) bool {
	// Expected pattern: saga-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (UUID-like)
	if len(id) < 5 {
		return false
	}
	return id[:5] == "saga-" && len(id) == 41 // saga- + 36 char UUID
}

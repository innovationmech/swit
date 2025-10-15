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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/state"
)

// TestNewPostgresStateStorage_InvalidConfig tests that NewPostgresStateStorage
// returns an error with invalid configuration.
func TestNewPostgresStateStorage_InvalidConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *PostgresConfig
		wantError bool
	}{
		{
			name:      "empty DSN",
			config:    &PostgresConfig{},
			wantError: true,
		},
		{
			name: "invalid max connections",
			config: &PostgresConfig{
				DSN:          "postgres://localhost/test",
				MaxOpenConns: -1,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewPostgresStateStorage(tt.config)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if storage != nil {
					t.Error("Expected nil storage on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if storage != nil {
					defer storage.Close()
				}
			}
		})
	}
}

// TestNewPostgresStateStorage_WithDefaults tests that default config works.
func TestNewPostgresStateStorage_WithDefaults(t *testing.T) {
	// Skip this test if no PostgreSQL is available
	// This is expected to fail without a real database connection
	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_Close tests the Close method.
func TestPostgresStateStorage_Close(t *testing.T) {
	// Create a storage with invalid DSN (so we can test Close without connection)
	config := DefaultPostgresConfig()
	config.DSN = "postgres://localhost/nonexistent"

	// We can't actually create the storage without a valid connection
	// So this test verifies the concept
	t.Skip("Requires PostgreSQL connection - testing close behavior requires valid connection")
}

// TestPostgresStateStorage_ConvertToSagaInstanceData tests the conversion
// from SagaInstance to SagaInstanceData.
func TestPostgresStateStorage_ConvertToSagaInstanceData(t *testing.T) {
	// This test doesn't require database connection
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config: config,
		closed: false,
	}

	testInstance := createTestSagaInstance("test-saga-123", "payment-saga", saga.StateRunning)

	data := storage.convertToSagaInstanceData(testInstance)

	if data.ID != testInstance.id {
		t.Errorf("Expected ID %s, got %s", testInstance.id, data.ID)
	}

	if data.DefinitionID != testInstance.definitionID {
		t.Errorf("Expected DefinitionID %s, got %s", testInstance.definitionID, data.DefinitionID)
	}

	if data.State != testInstance.state {
		t.Errorf("Expected State %v, got %v", testInstance.state, data.State)
	}

	if data.CurrentStep != testInstance.currentStep {
		t.Errorf("Expected CurrentStep %d, got %d", testInstance.currentStep, data.CurrentStep)
	}

	if data.TotalSteps != testInstance.totalSteps {
		t.Errorf("Expected TotalSteps %d, got %d", testInstance.totalSteps, data.TotalSteps)
	}

	if data.TraceID != testInstance.traceID {
		t.Errorf("Expected TraceID %s, got %s", testInstance.traceID, data.TraceID)
	}

	if data.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}
}

// TestPostgresStateStorage_ConvertToSagaInstance tests the conversion
// from SagaInstanceData to SagaInstance.
func TestPostgresStateStorage_ConvertToSagaInstance(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config: config,
		closed: false,
	}

	now := time.Now()
	data := &saga.SagaInstanceData{
		ID:           "test-saga-123",
		DefinitionID: "payment-saga",
		State:        saga.StateRunning,
		CurrentStep:  1,
		TotalSteps:   3,
		CreatedAt:    now,
		UpdatedAt:    now,
		StartedAt:    &now,
		TraceID:      "trace-123",
		Timeout:      5 * time.Minute,
	}

	instance := storage.convertToSagaInstance(data)

	if instance.GetID() != data.ID {
		t.Errorf("Expected ID %s, got %s", data.ID, instance.GetID())
	}

	if instance.GetDefinitionID() != data.DefinitionID {
		t.Errorf("Expected DefinitionID %s, got %s", data.DefinitionID, instance.GetDefinitionID())
	}

	if instance.GetState() != data.State {
		t.Errorf("Expected State %v, got %v", data.State, instance.GetState())
	}

	if instance.GetCurrentStep() != data.CurrentStep {
		t.Errorf("Expected CurrentStep %d, got %d", data.CurrentStep, instance.GetCurrentStep())
	}

	if instance.GetTotalSteps() != data.TotalSteps {
		t.Errorf("Expected TotalSteps %d, got %d", data.TotalSteps, instance.GetTotalSteps())
	}
}

// TestPostgresStateStorage_MarshalUnmarshalJSON tests JSON marshaling and unmarshaling.
func TestPostgresStateStorage_MarshalUnmarshalJSON(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config: config,
		closed: false,
	}

	tests := []struct {
		name  string
		value interface{}
	}{
		{
			name:  "nil value",
			value: nil,
		},
		{
			name:  "simple map",
			value: map[string]interface{}{"key": "value"},
		},
		{
			name:  "nested structure",
			value: map[string]interface{}{"nested": map[string]interface{}{"inner": "value"}},
		},
		{
			name: "saga error",
			value: &saga.SagaError{
				Code:    "ERR001",
				Message: "Test error",
				Type:    saga.ErrorTypeTimeout,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := storage.marshalJSON(tt.value)
			if err != nil {
				t.Errorf("marshalJSON failed: %v", err)
				return
			}

			if tt.value == nil && data != nil {
				t.Error("Expected nil data for nil value")
				return
			}

			if tt.value == nil {
				return
			}

			// Unmarshal
			var result interface{}
			err = storage.unmarshalJSON(data, &result)
			if err != nil {
				t.Errorf("unmarshalJSON failed: %v", err)
			}
		})
	}
}

// TestPostgresStateStorage_CheckClosed tests the checkClosed method.
func TestPostgresStateStorage_CheckClosed(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config: config,
		closed: false,
	}

	// Should not return error when not closed
	if err := storage.checkClosed(); err != nil {
		t.Errorf("Expected no error when not closed, got %v", err)
	}

	// Should return error when closed
	storage.closed = true
	if err := storage.checkClosed(); err == nil {
		t.Error("Expected error when closed")
	} else if err != state.ErrStorageClosed {
		t.Errorf("Expected ErrStorageClosed, got %v", err)
	}
}

// TestPostgresStateStorage_SaveSaga_InvalidInput tests SaveSaga with invalid input.
func TestPostgresStateStorage_SaveSaga_InvalidInput(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config:         config,
		closed:         false,
		instancesTable: "saga_instances",
	}

	ctx := context.Background()

	tests := []struct {
		name      string
		instance  saga.SagaInstance
		wantError error
	}{
		{
			name:      "nil instance",
			instance:  nil,
			wantError: state.ErrInvalidSagaID,
		},
		{
			name:      "empty saga ID",
			instance:  createTestSagaInstance("", "payment-saga", saga.StateRunning),
			wantError: state.ErrInvalidSagaID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.SaveSaga(ctx, tt.instance)
			if err == nil {
				t.Error("Expected error but got nil")
			} else if err != tt.wantError {
				t.Errorf("Expected error %v, got %v", tt.wantError, err)
			}
		})
	}
}

// TestPostgresStateStorage_GetSaga_InvalidInput tests GetSaga with invalid input.
func TestPostgresStateStorage_GetSaga_InvalidInput(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config:         config,
		closed:         false,
		instancesTable: "saga_instances",
	}

	ctx := context.Background()

	_, err := storage.GetSaga(ctx, "")
	if err == nil {
		t.Error("Expected error but got nil")
	} else if err != state.ErrInvalidSagaID {
		t.Errorf("Expected ErrInvalidSagaID, got %v", err)
	}
}

// TestPostgresStateStorage_UpdateSagaState_InvalidInput tests UpdateSagaState with invalid input.
func TestPostgresStateStorage_UpdateSagaState_InvalidInput(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config:         config,
		closed:         false,
		instancesTable: "saga_instances",
	}

	ctx := context.Background()

	err := storage.UpdateSagaState(ctx, "", saga.StateRunning, nil)
	if err == nil {
		t.Error("Expected error but got nil")
	} else if err != state.ErrInvalidSagaID {
		t.Errorf("Expected ErrInvalidSagaID, got %v", err)
	}
}

// TestPostgresStateStorage_DeleteSaga_InvalidInput tests DeleteSaga with invalid input.
func TestPostgresStateStorage_DeleteSaga_InvalidInput(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config:         config,
		closed:         false,
		instancesTable: "saga_instances",
	}

	ctx := context.Background()

	err := storage.DeleteSaga(ctx, "")
	if err == nil {
		t.Error("Expected error but got nil")
	} else if err != state.ErrInvalidSagaID {
		t.Errorf("Expected ErrInvalidSagaID, got %v", err)
	}
}

// TestPostgresStateStorage_SaveStepState_InvalidInput tests SaveStepState with invalid input.
func TestPostgresStateStorage_SaveStepState_InvalidInput(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config:     config,
		closed:     false,
		stepsTable: "saga_steps",
	}

	ctx := context.Background()

	tests := []struct {
		name      string
		sagaID    string
		step      *saga.StepState
		wantError error
	}{
		{
			name:      "empty saga ID",
			sagaID:    "",
			step:      &saga.StepState{ID: "step-1"},
			wantError: state.ErrInvalidSagaID,
		},
		{
			name:      "nil step",
			sagaID:    "saga-123",
			step:      nil,
			wantError: state.ErrInvalidState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.SaveStepState(ctx, tt.sagaID, tt.step)
			if err == nil {
				t.Error("Expected error but got nil")
			} else if err != tt.wantError {
				t.Errorf("Expected error %v, got %v", tt.wantError, err)
			}
		})
	}
}

// TestPostgresStateStorage_GetStepStates_InvalidInput tests GetStepStates with invalid input.
func TestPostgresStateStorage_GetStepStates_InvalidInput(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config:     config,
		closed:     false,
		stepsTable: "saga_steps",
	}

	ctx := context.Background()

	_, err := storage.GetStepStates(ctx, "")
	if err == nil {
		t.Error("Expected error but got nil")
	} else if err != state.ErrInvalidSagaID {
		t.Errorf("Expected ErrInvalidSagaID, got %v", err)
	}
}

// TestPostgresSagaInstance_Methods tests all methods of postgresSagaInstance.
func TestPostgresSagaInstance_Methods(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now

	data := &saga.SagaInstanceData{
		ID:           "test-saga-123",
		DefinitionID: "payment-saga",
		State:        saga.StateCompleted,
		CurrentStep:  3,
		TotalSteps:   3,
		CreatedAt:    now,
		UpdatedAt:    now,
		StartedAt:    &startTime,
		CompletedAt:  &endTime,
		ResultData:   map[string]interface{}{"result": "success"},
		Error:        nil,
		Timeout:      5 * time.Minute,
		Metadata:     map[string]interface{}{"key": "value"},
		TraceID:      "trace-123",
	}

	instance := &postgresSagaInstance{data: data}

	// Test all getter methods
	if instance.GetID() != data.ID {
		t.Errorf("GetID: expected %s, got %s", data.ID, instance.GetID())
	}

	if instance.GetDefinitionID() != data.DefinitionID {
		t.Errorf("GetDefinitionID: expected %s, got %s", data.DefinitionID, instance.GetDefinitionID())
	}

	if instance.GetState() != data.State {
		t.Errorf("GetState: expected %v, got %v", data.State, instance.GetState())
	}

	if instance.GetCurrentStep() != data.CurrentStep {
		t.Errorf("GetCurrentStep: expected %d, got %d", data.CurrentStep, instance.GetCurrentStep())
	}

	if instance.GetTotalSteps() != data.TotalSteps {
		t.Errorf("GetTotalSteps: expected %d, got %d", data.TotalSteps, instance.GetTotalSteps())
	}

	if !instance.GetStartTime().Equal(startTime) {
		t.Errorf("GetStartTime: expected %v, got %v", startTime, instance.GetStartTime())
	}

	if !instance.GetEndTime().Equal(endTime) {
		t.Errorf("GetEndTime: expected %v, got %v", endTime, instance.GetEndTime())
	}

	if instance.GetResult() == nil {
		t.Error("GetResult: expected non-nil result")
	}

	if instance.GetError() != nil {
		t.Errorf("GetError: expected nil, got %v", instance.GetError())
	}

	if instance.GetCompletedSteps() != data.CurrentStep {
		t.Errorf("GetCompletedSteps: expected %d, got %d", data.CurrentStep, instance.GetCompletedSteps())
	}

	if !instance.GetCreatedAt().Equal(data.CreatedAt) {
		t.Errorf("GetCreatedAt: expected %v, got %v", data.CreatedAt, instance.GetCreatedAt())
	}

	if !instance.GetUpdatedAt().Equal(data.UpdatedAt) {
		t.Errorf("GetUpdatedAt: expected %v, got %v", data.UpdatedAt, instance.GetUpdatedAt())
	}

	if instance.GetTimeout() != data.Timeout {
		t.Errorf("GetTimeout: expected %v, got %v", data.Timeout, instance.GetTimeout())
	}

	if instance.GetMetadata() == nil {
		t.Error("GetMetadata: expected non-nil metadata")
	}

	if instance.GetTraceID() != data.TraceID {
		t.Errorf("GetTraceID: expected %s, got %s", data.TraceID, instance.GetTraceID())
	}

	if !instance.IsTerminal() {
		t.Error("IsTerminal: expected true for completed state")
	}

	if instance.IsActive() {
		t.Error("IsActive: expected false for completed state")
	}
}

// TestPostgresStateStorage_SaveSaga_NotImplemented tests that SaveSaga returns
// ErrNotImplemented since it's not yet implemented.
func TestPostgresStateStorage_SaveSaga_NotImplemented(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will implement with actual DB")
}

// TestPostgresStateStorage_GetSaga_NotImplemented tests that GetSaga returns
// ErrNotImplemented since it's not yet implemented.
func TestPostgresStateStorage_GetSaga_NotImplemented(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will implement with actual DB")
}

// TestPostgresStateStorage_UpdateSagaState_NotImplemented tests that UpdateSagaState
// returns ErrNotImplemented since it's not yet implemented.
func TestPostgresStateStorage_UpdateSagaState_NotImplemented(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will implement with actual DB")
}

// TestPostgresStateStorage_DeleteSaga_NotImplemented tests that DeleteSaga returns
// ErrNotImplemented since it's not yet implemented.
func TestPostgresStateStorage_DeleteSaga_NotImplemented(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will implement with actual DB")
}

// TestPostgresStateStorage_GetActiveSagas_NotImplemented tests that GetActiveSagas
// returns ErrNotImplemented since it's not yet implemented.
func TestPostgresStateStorage_GetActiveSagas_NotImplemented(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will implement with actual DB")
}

// TestPostgresStateStorage_GetTimeoutSagas_NotImplemented tests that GetTimeoutSagas
// returns ErrNotImplemented since it's not yet implemented.
func TestPostgresStateStorage_GetTimeoutSagas_NotImplemented(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will implement with actual DB")
}

// TestPostgresStateStorage_SaveStepState_NotImplemented tests that SaveStepState
// returns ErrNotImplemented since it's not yet implemented.
func TestPostgresStateStorage_SaveStepState_NotImplemented(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will implement with actual DB")
}

// TestPostgresStateStorage_GetStepStates_NotImplemented tests that GetStepStates
// returns ErrNotImplemented since it's not yet implemented.
func TestPostgresStateStorage_GetStepStates_NotImplemented(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will implement with actual DB")
}

// TestPostgresStateStorage_CleanupExpiredSagas_NotImplemented tests that CleanupExpiredSagas
// returns ErrNotImplemented since it's not yet implemented.
func TestPostgresStateStorage_CleanupExpiredSagas_NotImplemented(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will implement with actual DB")
}

// TestPostgresStateStorage_ImplementsInterface verifies that PostgresStateStorage
// implements the saga.StateStorage interface at compile time.
func TestPostgresStateStorage_ImplementsInterface(t *testing.T) {
	// This test ensures PostgresStateStorage implements saga.StateStorage interface
	var _ saga.StateStorage = (*PostgresStateStorage)(nil)
	t.Log("PostgresStateStorage implements saga.StateStorage interface")
}

// TestPostgresStateStorage_ValidationChecks tests basic validation without database.
func TestPostgresStateStorage_ValidationChecks(t *testing.T) {
	// Create a mock storage that's already closed
	storage := &PostgresStateStorage{
		closed: true,
	}

	ctx := context.Background()

	// Test that closed storage returns appropriate errors
	t.Run("SaveSaga with closed storage", func(t *testing.T) {
		err := storage.SaveSaga(ctx, nil)
		if err != state.ErrStorageClosed {
			t.Errorf("Expected ErrStorageClosed, got %v", err)
		}
	})

	t.Run("GetSaga with closed storage", func(t *testing.T) {
		_, err := storage.GetSaga(ctx, "test-id")
		if err != state.ErrStorageClosed {
			t.Errorf("Expected ErrStorageClosed, got %v", err)
		}
	})

	t.Run("UpdateSagaState with closed storage", func(t *testing.T) {
		err := storage.UpdateSagaState(ctx, "test-id", saga.StateRunning, nil)
		if err != state.ErrStorageClosed {
			t.Errorf("Expected ErrStorageClosed, got %v", err)
		}
	})

	t.Run("DeleteSaga with closed storage", func(t *testing.T) {
		err := storage.DeleteSaga(ctx, "test-id")
		if err != state.ErrStorageClosed {
			t.Errorf("Expected ErrStorageClosed, got %v", err)
		}
	})

	t.Run("GetActiveSagas with closed storage", func(t *testing.T) {
		_, err := storage.GetActiveSagas(ctx, nil)
		if err != state.ErrStorageClosed {
			t.Errorf("Expected ErrStorageClosed, got %v", err)
		}
	})

	t.Run("GetTimeoutSagas with closed storage", func(t *testing.T) {
		_, err := storage.GetTimeoutSagas(ctx, time.Now())
		if err != state.ErrStorageClosed {
			t.Errorf("Expected ErrStorageClosed, got %v", err)
		}
	})

	t.Run("SaveStepState with closed storage", func(t *testing.T) {
		err := storage.SaveStepState(ctx, "test-id", nil)
		if err != state.ErrStorageClosed {
			t.Errorf("Expected ErrStorageClosed, got %v", err)
		}
	})

	t.Run("GetStepStates with closed storage", func(t *testing.T) {
		_, err := storage.GetStepStates(ctx, "test-id")
		if err != state.ErrStorageClosed {
			t.Errorf("Expected ErrStorageClosed, got %v", err)
		}
	})

	t.Run("CleanupExpiredSagas with closed storage", func(t *testing.T) {
		err := storage.CleanupExpiredSagas(ctx, time.Now())
		if err != state.ErrStorageClosed {
			t.Errorf("Expected ErrStorageClosed, got %v", err)
		}
	})

	// Test Close on already closed storage
	t.Run("Close already closed storage", func(t *testing.T) {
		err := storage.Close()
		if err != nil {
			t.Errorf("Expected no error on double close, got %v", err)
		}
	})
}

// TestPostgresStateStorage_ContextCancellation tests context cancellation handling.
func TestPostgresStateStorage_ContextCancellation(t *testing.T) {
	storage := &PostgresStateStorage{
		closed: false,
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Create a mock instance for testing
	mockInstance := &mockSagaInstance{
		id:           "test-id",
		definitionID: "test-def",
		sagaState:    saga.StateRunning,
	}

	t.Run("SaveSaga with cancelled context", func(t *testing.T) {
		err := storage.SaveSaga(ctx, mockInstance)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})

	t.Run("GetSaga with cancelled context", func(t *testing.T) {
		_, err := storage.GetSaga(ctx, "test-id")
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})

	t.Run("UpdateSagaState with cancelled context", func(t *testing.T) {
		err := storage.UpdateSagaState(ctx, "test-id", saga.StateRunning, nil)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})
}

// TestPostgresStateStorage_InvalidInputs tests handling of invalid inputs.
func TestPostgresStateStorage_InvalidInputs(t *testing.T) {
	storage := &PostgresStateStorage{
		closed: false,
	}

	ctx := context.Background()

	t.Run("SaveSaga with nil instance", func(t *testing.T) {
		err := storage.SaveSaga(ctx, nil)
		if err != state.ErrInvalidSagaID {
			t.Errorf("Expected ErrInvalidSagaID, got %v", err)
		}
	})

	t.Run("SaveSaga with empty ID", func(t *testing.T) {
		mockInstance := &mockSagaInstance{
			id:           "",
			definitionID: "test-def",
			sagaState:    saga.StateRunning,
		}
		err := storage.SaveSaga(ctx, mockInstance)
		if err != state.ErrInvalidSagaID {
			t.Errorf("Expected ErrInvalidSagaID, got %v", err)
		}
	})

	t.Run("GetSaga with empty ID", func(t *testing.T) {
		_, err := storage.GetSaga(ctx, "")
		if err != state.ErrInvalidSagaID {
			t.Errorf("Expected ErrInvalidSagaID, got %v", err)
		}
	})

	t.Run("UpdateSagaState with empty ID", func(t *testing.T) {
		err := storage.UpdateSagaState(ctx, "", saga.StateRunning, nil)
		if err != state.ErrInvalidSagaID {
			t.Errorf("Expected ErrInvalidSagaID, got %v", err)
		}
	})

	t.Run("DeleteSaga with empty ID", func(t *testing.T) {
		err := storage.DeleteSaga(ctx, "")
		if err != state.ErrInvalidSagaID {
			t.Errorf("Expected ErrInvalidSagaID, got %v", err)
		}
	})

	t.Run("SaveStepState with empty saga ID", func(t *testing.T) {
		step := &saga.StepState{ID: "step-1"}
		err := storage.SaveStepState(ctx, "", step)
		if err != state.ErrInvalidSagaID {
			t.Errorf("Expected ErrInvalidSagaID, got %v", err)
		}
	})

	t.Run("SaveStepState with nil step", func(t *testing.T) {
		err := storage.SaveStepState(ctx, "saga-1", nil)
		if err != state.ErrInvalidState {
			t.Errorf("Expected ErrInvalidState, got %v", err)
		}
	})

	t.Run("GetStepStates with empty saga ID", func(t *testing.T) {
		_, err := storage.GetStepStates(ctx, "")
		if err != state.ErrInvalidSagaID {
			t.Errorf("Expected ErrInvalidSagaID, got %v", err)
		}
	})
}

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

// ==========================
// Complex Query Tests
// ==========================

// TestPostgresStateStorage_BuildWhereClause tests the WHERE clause builder with various filters.
func TestPostgresStateStorage_BuildWhereClause(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config:         config,
		closed:         false,
		instancesTable: "saga_instances",
	}

	tests := []struct {
		name          string
		filter        *saga.SagaFilter
		wantCondition bool // whether WHERE clause should be present
		wantArgCount  int
	}{
		{
			name:          "nil filter",
			filter:        nil,
			wantCondition: false,
			wantArgCount:  0,
		},
		{
			name: "filter by single state",
			filter: &saga.SagaFilter{
				States: []saga.SagaState{saga.StateRunning},
			},
			wantCondition: true,
			wantArgCount:  1,
		},
		{
			name: "filter by multiple states",
			filter: &saga.SagaFilter{
				States: []saga.SagaState{saga.StateRunning, saga.StateCompleted, saga.StateFailed},
			},
			wantCondition: true,
			wantArgCount:  3,
		},
		{
			name: "filter by definition IDs",
			filter: &saga.SagaFilter{
				DefinitionIDs: []string{"payment-saga", "order-saga"},
			},
			wantCondition: true,
			wantArgCount:  2,
		},
		{
			name: "filter by time range",
			filter: &saga.SagaFilter{
				CreatedAfter:  timePtr(time.Now().Add(-24 * time.Hour)),
				CreatedBefore: timePtr(time.Now()),
			},
			wantCondition: true,
			wantArgCount:  2,
		},
		{
			name: "filter by metadata",
			filter: &saga.SagaFilter{
				Metadata: map[string]interface{}{
					"tenant": "test-tenant",
					"region": "us-east-1",
				},
			},
			wantCondition: true,
			wantArgCount:  1, // metadata is marshaled as one JSONB argument
		},
		{
			name: "combined filters",
			filter: &saga.SagaFilter{
				States:        []saga.SagaState{saga.StateRunning, saga.StateCompleted},
				DefinitionIDs: []string{"payment-saga"},
				CreatedAfter:  timePtr(time.Now().Add(-24 * time.Hour)),
			},
			wantCondition: true,
			wantArgCount:  4, // 2 states + 1 definition + 1 time
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			whereClause, args := storage.buildWhereClause(tt.filter)

			if tt.wantCondition {
				if whereClause == "" {
					t.Error("Expected WHERE clause but got empty string")
				}
			} else {
				if whereClause != "" {
					t.Errorf("Expected empty WHERE clause but got: %s", whereClause)
				}
			}

			if len(args) != tt.wantArgCount {
				t.Errorf("Expected %d arguments, got %d", tt.wantArgCount, len(args))
			}
		})
	}
}

// TestPostgresStateStorage_BuildOrderByClause tests the ORDER BY clause builder.
func TestPostgresStateStorage_BuildOrderByClause(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config:         config,
		closed:         false,
		instancesTable: "saga_instances",
	}

	tests := []struct {
		name         string
		filter       *saga.SagaFilter
		wantContains string
	}{
		{
			name:         "nil filter - default sorting",
			filter:       nil,
			wantContains: "created_at DESC",
		},
		{
			name: "sort by created_at asc",
			filter: &saga.SagaFilter{
				SortBy:    "created_at",
				SortOrder: "asc",
			},
			wantContains: "created_at ASC",
		},
		{
			name: "sort by updated_at desc",
			filter: &saga.SagaFilter{
				SortBy:    "updated_at",
				SortOrder: "desc",
			},
			wantContains: "updated_at DESC",
		},
		{
			name: "sort by state",
			filter: &saga.SagaFilter{
				SortBy:    "state",
				SortOrder: "asc",
			},
			wantContains: "state ASC",
		},
		{
			name: "invalid sort field - fallback to created_at",
			filter: &saga.SagaFilter{
				SortBy:    "invalid_field",
				SortOrder: "asc",
			},
			wantContains: "created_at ASC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderBy := storage.buildOrderByClause(tt.filter)

			if orderBy == "" {
				t.Error("Expected ORDER BY clause but got empty string")
			}

			if !containsString(orderBy, tt.wantContains) {
				t.Errorf("Expected ORDER BY clause to contain '%s', got: %s", tt.wantContains, orderBy)
			}
		})
	}
}

// TestPostgresStateStorage_BuildPaginationClause tests pagination clause builder.
func TestPostgresStateStorage_BuildPaginationClause(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config:         config,
		closed:         false,
		instancesTable: "saga_instances",
	}

	tests := []struct {
		name         string
		baseQuery    string
		baseArgs     []interface{}
		filter       *saga.SagaFilter
		wantLimit    bool
		wantOffset   bool
		wantArgCount int
	}{
		{
			name:         "nil filter",
			baseQuery:    "SELECT * FROM saga_instances",
			baseArgs:     []interface{}{},
			filter:       nil,
			wantLimit:    false,
			wantOffset:   false,
			wantArgCount: 0,
		},
		{
			name:      "with limit only",
			baseQuery: "SELECT * FROM saga_instances",
			baseArgs:  []interface{}{},
			filter: &saga.SagaFilter{
				Limit: 10,
			},
			wantLimit:    true,
			wantOffset:   false,
			wantArgCount: 1,
		},
		{
			name:      "with limit and offset",
			baseQuery: "SELECT * FROM saga_instances",
			baseArgs:  []interface{}{},
			filter: &saga.SagaFilter{
				Limit:  10,
				Offset: 20,
			},
			wantLimit:    true,
			wantOffset:   true,
			wantArgCount: 2,
		},
		{
			name:      "with existing args",
			baseQuery: "SELECT * FROM saga_instances WHERE state = $1",
			baseArgs:  []interface{}{saga.StateRunning},
			filter: &saga.SagaFilter{
				Limit:  5,
				Offset: 10,
			},
			wantLimit:    true,
			wantOffset:   true,
			wantArgCount: 3, // 1 existing + 2 new
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args := storage.buildPaginationClause(tt.baseQuery, tt.baseArgs, tt.filter)

			if tt.wantLimit && !containsString(query, "LIMIT") {
				t.Error("Expected query to contain LIMIT")
			}

			if tt.wantOffset && !containsString(query, "OFFSET") {
				t.Error("Expected query to contain OFFSET")
			}

			if len(args) != tt.wantArgCount {
				t.Errorf("Expected %d arguments, got %d", tt.wantArgCount, len(args))
			}
		})
	}
}

// TestPostgresStateStorage_SanitizeSortField tests sort field sanitization.
func TestPostgresStateStorage_SanitizeSortField(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config: config,
		closed: false,
	}

	tests := []struct {
		name      string
		field     string
		wantField string
	}{
		{
			name:      "valid field - id",
			field:     "id",
			wantField: "id",
		},
		{
			name:      "valid field - created_at",
			field:     "created_at",
			wantField: "created_at",
		},
		{
			name:      "valid field - state",
			field:     "state",
			wantField: "state",
		},
		{
			name:      "invalid field - SQL injection attempt",
			field:     "id; DROP TABLE saga_instances;--",
			wantField: "created_at", // should fallback to default
		},
		{
			name:      "invalid field - random string",
			field:     "invalid_column",
			wantField: "created_at", // should fallback to default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := storage.sanitizeSortField(tt.field)
			if result != tt.wantField {
				t.Errorf("Expected field %s, got %s", tt.wantField, result)
			}
		})
	}
}

// TestPostgresStateStorage_JoinStrings tests the string joining utility.
func TestPostgresStateStorage_JoinStrings(t *testing.T) {
	tests := []struct {
		name string
		strs []string
		sep  string
		want string
	}{
		{
			name: "empty slice",
			strs: []string{},
			sep:  ", ",
			want: "",
		},
		{
			name: "single element",
			strs: []string{"one"},
			sep:  ", ",
			want: "one",
		},
		{
			name: "multiple elements",
			strs: []string{"one", "two", "three"},
			sep:  ", ",
			want: "one, two, three",
		},
		{
			name: "different separator",
			strs: []string{"a", "b", "c"},
			sep:  " AND ",
			want: "a AND b AND c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinStrings(tt.strs, tt.sep)
			if result != tt.want {
				t.Errorf("Expected '%s', got '%s'", tt.want, result)
			}
		})
	}
}

// TestPostgresStateStorage_BuildListQuery tests the complete query builder.
func TestPostgresStateStorage_BuildListQuery(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config:         config,
		closed:         false,
		instancesTable: "saga_instances",
	}

	tests := []struct {
		name            string
		filter          *saga.SagaFilter
		wantContains    []string
		wantNotContains []string
		wantArgCount    int
	}{
		{
			name:   "empty filter - default query",
			filter: &saga.SagaFilter{},
			wantContains: []string{
				"SELECT",
				"FROM saga_instances",
				"ORDER BY created_at DESC",
			},
			wantNotContains: []string{"WHERE", "LIMIT", "OFFSET"},
			wantArgCount:    0,
		},
		{
			name: "filter with states and pagination",
			filter: &saga.SagaFilter{
				States: []saga.SagaState{saga.StateRunning, saga.StateCompleted},
				Limit:  10,
				Offset: 5,
			},
			wantContains: []string{
				"SELECT",
				"WHERE",
				"state IN",
				"LIMIT",
				"OFFSET",
			},
			wantArgCount: 4, // 2 states + limit + offset
		},
		{
			name: "complex filter with all options",
			filter: &saga.SagaFilter{
				States:        []saga.SagaState{saga.StateRunning},
				DefinitionIDs: []string{"payment-saga"},
				CreatedAfter:  timePtr(time.Now().Add(-24 * time.Hour)),
				SortBy:        "updated_at",
				SortOrder:     "asc",
				Limit:         20,
			},
			wantContains: []string{
				"SELECT",
				"WHERE",
				"state IN",
				"definition_id IN",
				"created_at >=",
				"ORDER BY updated_at ASC",
				"LIMIT",
			},
			wantArgCount: 4, // 1 state + 1 definition + 1 time + limit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args := storage.buildListQuery(tt.filter)

			if query == "" {
				t.Error("Expected non-empty query")
			}

			for _, want := range tt.wantContains {
				if !containsString(query, want) {
					t.Errorf("Expected query to contain '%s', got: %s", want, query)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if containsString(query, notWant) {
					t.Errorf("Expected query NOT to contain '%s', got: %s", notWant, query)
				}
			}

			if len(args) != tt.wantArgCount {
				t.Errorf("Expected %d arguments, got %d", tt.wantArgCount, len(args))
			}
		})
	}
}

// ==========================
// Helper Functions for Tests
// ==========================

// timePtr returns a pointer to a time.Time value.
func timePtr(t time.Time) *time.Time {
	return &t
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr)))
}

// containsSubstring checks if substr exists within s.
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ==========================
// Transaction Tests
// ==========================

// TestPostgresStateStorage_TransactionBasics tests basic transaction operations.
func TestPostgresStateStorage_TransactionBasics(t *testing.T) {
	// Skip if no real database - transactions require actual PostgreSQL
	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_TransactionCommit tests transaction commit behavior.
func TestPostgresStateStorage_TransactionCommit(t *testing.T) {
	// Test will verify:
	// 1. BeginTransaction succeeds
	// 2. Operations within transaction work
	// 3. Commit makes changes permanent
	// 4. Changes visible after commit

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_TransactionRollback tests transaction rollback behavior.
func TestPostgresStateStorage_TransactionRollback(t *testing.T) {
	// Test will verify:
	// 1. BeginTransaction succeeds
	// 2. Operations within transaction work
	// 3. Rollback discards changes
	// 4. Changes not visible after rollback

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_TransactionTimeout tests transaction timeout behavior.
func TestPostgresStateStorage_TransactionTimeout(t *testing.T) {
	// Test will verify:
	// 1. Transaction timeout is enforced
	// 2. Operations fail after timeout
	// 3. Proper error returned on timeout

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_TransactionClosed tests operations on closed transactions.
func TestPostgresStateStorage_TransactionClosed(t *testing.T) {
	// Test will verify:
	// 1. Operations fail on closed transaction
	// 2. Proper error returned (ErrTransactionClosed)
	// 3. Double commit/rollback is safe

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestSagaTransaction_SaveSaga tests SaveSaga within transaction.
func TestSagaTransaction_SaveSaga(t *testing.T) {
	// Test will verify:
	// 1. SaveSaga works within transaction
	// 2. Multiple sagas can be saved in one transaction
	// 3. Changes are isolated until commit

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestSagaTransaction_SaveStepState tests SaveStepState within transaction.
func TestSagaTransaction_SaveStepState(t *testing.T) {
	// Test will verify:
	// 1. SaveStepState works within transaction
	// 2. Multiple steps can be saved in one transaction
	// 3. Changes are isolated until commit

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestSagaTransaction_UpdateSagaState tests UpdateSagaState within transaction.
func TestSagaTransaction_UpdateSagaState(t *testing.T) {
	// Test will verify:
	// 1. UpdateSagaState works within transaction
	// 2. State updates are atomic
	// 3. Metadata merging works correctly

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestSagaTransaction_DeleteSaga tests DeleteSaga within transaction.
func TestSagaTransaction_DeleteSaga(t *testing.T) {
	// Test will verify:
	// 1. DeleteSaga works within transaction
	// 2. Cascade delete works for steps
	// 3. Rollback restores deleted saga

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestSagaTransaction_Exec tests custom SQL execution within transaction.
func TestSagaTransaction_Exec(t *testing.T) {
	// Test will verify:
	// 1. Exec works within transaction
	// 2. Custom queries can be executed
	// 3. Changes are atomic with saga operations

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_BatchSaveSagas tests batch save operations.
func TestPostgresStateStorage_BatchSaveSagas(t *testing.T) {
	// Test will verify:
	// 1. Multiple sagas saved atomically
	// 2. All-or-nothing behavior on error
	// 3. Timeout parameter works

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_BatchSaveStepStates tests batch step save operations.
func TestPostgresStateStorage_BatchSaveStepStates(t *testing.T) {
	// Test will verify:
	// 1. Multiple steps saved atomically
	// 2. All-or-nothing behavior on error
	// 3. Timeout parameter works

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// ==========================
// Optimistic Locking Tests
// ==========================

// TestPostgresStateStorage_OptimisticLocking tests optimistic locking with version.
func TestPostgresStateStorage_OptimisticLocking(t *testing.T) {
	// Test will verify:
	// 1. Version field is tracked correctly
	// 2. Concurrent updates detect conflicts
	// 3. ErrOptimisticLockFailed returned on version mismatch

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_UpdateSagaWithOptimisticLock tests optimistic lock updates.
func TestPostgresStateStorage_UpdateSagaWithOptimisticLock(t *testing.T) {
	// Test will verify:
	// 1. Update succeeds with correct version
	// 2. Update fails with wrong version
	// 3. Version is incremented on update
	// 4. Proper error differentiation (not found vs version mismatch)

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_UpdateSagaStateWithOptimisticLock tests state update with locking.
func TestPostgresStateStorage_UpdateSagaStateWithOptimisticLock(t *testing.T) {
	// Test will verify:
	// 1. State update succeeds with correct version
	// 2. State update fails with wrong version
	// 3. Metadata merge works with version check
	// 4. Version is incremented on update

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// ==========================
// Concurrency Tests
// ==========================

// TestPostgresStateStorage_ConcurrentTransactions tests concurrent transaction handling.
func TestPostgresStateStorage_ConcurrentTransactions(t *testing.T) {
	// Test will verify:
	// 1. Multiple transactions can run concurrently
	// 2. Transaction isolation is maintained
	// 3. No data corruption occurs
	// 4. Performance is acceptable with concurrent load

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_ConcurrentOptimisticLocking tests concurrent optimistic locking.
func TestPostgresStateStorage_ConcurrentOptimisticLocking(t *testing.T) {
	// Test will verify:
	// 1. Multiple goroutines attempting updates
	// 2. Only one succeeds per version
	// 3. Others get ErrOptimisticLockFailed
	// 4. Retry logic can eventually succeed
	// 5. No lost updates occur

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_ConcurrentMixedOperations tests mixed concurrent operations.
func TestPostgresStateStorage_ConcurrentMixedOperations(t *testing.T) {
	// Test will verify:
	// 1. Mix of reads, writes, updates, deletes
	// 2. Transaction and non-transaction operations
	// 3. No deadlocks occur
	// 4. Data consistency maintained
	// 5. All operations complete successfully

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_TransactionDeadlockHandling tests deadlock scenarios.
func TestPostgresStateStorage_TransactionDeadlockHandling(t *testing.T) {
	// Test will verify:
	// 1. Deadlock detection and handling
	// 2. Proper error reporting
	// 3. Retry mechanism can resolve deadlocks

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// ==========================
// Integration Test Scenarios
// ==========================

// TestPostgresStateStorage_TransactionScenario_OrderProcessing tests a realistic scenario.
func TestPostgresStateStorage_TransactionScenario_OrderProcessing(t *testing.T) {
	// Simulates order processing saga:
	// 1. Create order saga
	// 2. Save multiple step states (inventory check, payment, shipping)
	// 3. Update saga state as steps complete
	// 4. All operations in transaction
	// 5. Commit on success or rollback on failure

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_TransactionScenario_Compensation tests compensation flow.
func TestPostgresStateStorage_TransactionScenario_Compensation(t *testing.T) {
	// Simulates compensation scenario:
	// 1. Saga progresses through multiple steps
	// 2. A step fails mid-way
	// 3. Compensation steps are executed
	// 4. State updates tracked transactionally
	// 5. Final state reflects compensation

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_VersionIncrement tests version field behavior.
func TestPostgresStateStorage_VersionIncrement(t *testing.T) {
	// Test will verify:
	// 1. Version starts at 1 for new sagas
	// 2. Version increments on each update
	// 3. Version is correctly read back
	// 4. Database trigger increments version

	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// ==========================
// Health Check and Monitoring Tests
// ==========================

// TestPostgresStateStorage_HealthCheck tests the HealthCheck method.
func TestPostgresStateStorage_HealthCheck(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_HealthCheckWithRetry tests health check with retry.
func TestPostgresStateStorage_HealthCheckWithRetry(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_GetPoolStats tests getting connection pool statistics.
func TestPostgresStateStorage_GetPoolStats(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_DetectConnectionLeaks tests connection leak detection.
func TestPostgresStateStorage_DetectConnectionLeaks(t *testing.T) {
	t.Skip("Requires PostgreSQL connection - will be tested in integration tests")
}

// TestPostgresStateStorage_CalculateBackoff tests the backoff calculation.
func TestPostgresStateStorage_CalculateBackoff(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"
	config.RetryBackoff = 100 * time.Millisecond
	config.MaxRetryBackoff = 5 * time.Second

	// Create a mock storage to test the method
	// Note: We can't fully initialize without DB connection, but we can test the logic
	storage := &PostgresStateStorage{
		config: config,
	}

	tests := []struct {
		name           string
		attempt        int
		wantMin        time.Duration
		wantMax        time.Duration
		shouldBeCapped bool
	}{
		{
			name:    "first retry",
			attempt: 1,
			wantMin: 100 * time.Millisecond,
			wantMax: 100 * time.Millisecond,
		},
		{
			name:    "second retry",
			attempt: 2,
			wantMin: 200 * time.Millisecond,
			wantMax: 200 * time.Millisecond,
		},
		{
			name:    "third retry",
			attempt: 3,
			wantMin: 400 * time.Millisecond,
			wantMax: 400 * time.Millisecond,
		},
		{
			name:           "large attempt (should be capped)",
			attempt:        10,
			wantMin:        5 * time.Second,
			wantMax:        5 * time.Second,
			shouldBeCapped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backoff := storage.calculateBackoff(tt.attempt)

			if backoff < tt.wantMin {
				t.Errorf("calculateBackoff() = %v, want at least %v", backoff, tt.wantMin)
			}
			if backoff > tt.wantMax {
				t.Errorf("calculateBackoff() = %v, want at most %v", backoff, tt.wantMax)
			}

			if tt.shouldBeCapped && backoff != config.MaxRetryBackoff {
				t.Errorf("calculateBackoff() = %v, want capped at %v", backoff, config.MaxRetryBackoff)
			}
		})
	}
}

// TestPostgresStateStorage_IsRetriableError tests error classification.
func TestPostgresStateStorage_IsRetriableError(t *testing.T) {
	config := DefaultPostgresConfig()
	config.DSN = "postgres://test"

	storage := &PostgresStateStorage{
		config: config,
	}

	tests := []struct {
		name          string
		err           error
		wantRetriable bool
	}{
		{
			name:          "nil error",
			err:           nil,
			wantRetriable: false,
		},
		{
			name:          "context canceled",
			err:           context.Canceled,
			wantRetriable: false,
		},
		{
			name:          "context deadline exceeded",
			err:           context.DeadlineExceeded,
			wantRetriable: false,
		},
		{
			name:          "connection refused",
			err:           &testError{msg: "connection refused"},
			wantRetriable: true,
		},
		{
			name:          "connection reset",
			err:           &testError{msg: "connection reset by peer"},
			wantRetriable: true,
		},
		{
			name:          "timeout error",
			err:           &testError{msg: "timeout waiting for connection"},
			wantRetriable: true,
		},
		{
			name:          "deadlock error",
			err:           &testError{msg: "deadlock detected"},
			wantRetriable: true,
		},
		{
			name:          "too many connections",
			err:           &testError{msg: "too many connections"},
			wantRetriable: true,
		},
		{
			name:          "serialization error",
			err:           &testError{msg: "could not serialize access due to concurrent update"},
			wantRetriable: true,
		},
		{
			name:          "non-retriable error",
			err:           &testError{msg: "syntax error at or near SELECT"},
			wantRetriable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := storage.isRetriableError(tt.err)
			if got != tt.wantRetriable {
				t.Errorf("isRetriableError() = %v, want %v for error: %v", got, tt.wantRetriable, tt.err)
			}
		})
	}
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestPostgresStateStorage_Contains tests the string contains helper.
func TestPostgresStateStorage_Contains(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "exact match",
			s:      "timeout",
			substr: "timeout",
			want:   true,
		},
		{
			name:   "substring at start",
			s:      "timeout error",
			substr: "timeout",
			want:   true,
		},
		{
			name:   "substring in middle",
			s:      "connection timeout error",
			substr: "timeout",
			want:   true,
		},
		{
			name:   "substring at end",
			s:      "connection timeout",
			substr: "timeout",
			want:   true,
		},
		{
			name:   "not found",
			s:      "connection error",
			substr: "timeout",
			want:   false,
		},
		{
			name:   "empty substring",
			s:      "any string",
			substr: "",
			want:   true,
		},
		{
			name:   "empty string",
			s:      "",
			substr: "timeout",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

// TestPostgresStateStorage_HealthCheck_ClosedStorage tests health check on closed storage.
func TestPostgresStateStorage_HealthCheck_ClosedStorage(t *testing.T) {
	// Create a mock closed storage
	storage := &PostgresStateStorage{
		closed: true,
		config: DefaultPostgresConfig(),
	}

	ctx := context.Background()
	err := storage.HealthCheck(ctx)

	if err != state.ErrStorageClosed {
		t.Errorf("HealthCheck() on closed storage = %v, want %v", err, state.ErrStorageClosed)
	}
}

// TestPostgresStateStorage_GetPoolStats_ClosedStorage tests getting stats from closed storage.
func TestPostgresStateStorage_GetPoolStats_ClosedStorage(t *testing.T) {
	// Create a mock closed storage
	storage := &PostgresStateStorage{
		closed: true,
		config: DefaultPostgresConfig(),
	}

	stats, err := storage.GetPoolStats()

	if err != state.ErrStorageClosed {
		t.Errorf("GetPoolStats() on closed storage error = %v, want %v", err, state.ErrStorageClosed)
	}
	if stats != nil {
		t.Errorf("GetPoolStats() on closed storage = %v, want nil", stats)
	}
}

// TestPostgresStateStorage_DetectConnectionLeaks_ClosedStorage tests leak detection on closed storage.
func TestPostgresStateStorage_DetectConnectionLeaks_ClosedStorage(t *testing.T) {
	// Create a mock closed storage
	storage := &PostgresStateStorage{
		closed: true,
		config: DefaultPostgresConfig(),
	}

	leaked, msg, err := storage.DetectConnectionLeaks()

	if err != state.ErrStorageClosed {
		t.Errorf("DetectConnectionLeaks() on closed storage error = %v, want %v", err, state.ErrStorageClosed)
	}
	if leaked {
		t.Errorf("DetectConnectionLeaks() leaked = %v, want false", leaked)
	}
	if msg != "" {
		t.Errorf("DetectConnectionLeaks() msg = %q, want empty", msg)
	}
}

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

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

package security

import (
	"context"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditEntry_ToJSON(t *testing.T) {
	tests := []struct {
		name    string
		entry   *AuditEntry
		wantErr bool
	}{
		{
			name: "valid entry",
			entry: &AuditEntry{
				ID:           "audit-1",
				Timestamp:    time.Now(),
				Level:        AuditLevelInfo,
				Action:       AuditActionSagaStarted,
				ResourceType: "saga",
				ResourceID:   "saga-123",
				UserID:       "user-1",
				Message:      "Saga started",
			},
			wantErr: false,
		},
		{
			name: "entry with details",
			entry: &AuditEntry{
				ID:           "audit-2",
				Timestamp:    time.Now(),
				Level:        AuditLevelWarning,
				Action:       AuditActionAuthFailure,
				ResourceType: "auth",
				UserID:       "user-2",
				Message:      "Authentication failed",
				Details: map[string]interface{}{
					"reason": "invalid_credentials",
					"ip":     "192.168.1.1",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr, err := tt.entry.ToJSON()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, jsonStr)
			}
		})
	}
}

func TestAuditFilter_Match(t *testing.T) {
	now := time.Now()
	hourAgo := now.Add(-time.Hour)
	hourLater := now.Add(time.Hour)

	tests := []struct {
		name   string
		filter *AuditFilter
		entry  *AuditEntry
		want   bool
	}{
		{
			name:   "nil filter matches all",
			filter: nil,
			entry: &AuditEntry{
				ID:        "audit-1",
				Timestamp: now,
			},
			want: true,
		},
		{
			name: "time range match",
			filter: &AuditFilter{
				StartTime: &hourAgo,
				EndTime:   &hourLater,
			},
			entry: &AuditEntry{
				ID:        "audit-1",
				Timestamp: now,
			},
			want: true,
		},
		{
			name: "time range no match - before start",
			filter: &AuditFilter{
				StartTime: &now,
			},
			entry: &AuditEntry{
				ID:        "audit-1",
				Timestamp: hourAgo,
			},
			want: false,
		},
		{
			name: "level match",
			filter: &AuditFilter{
				Levels: []AuditLevel{AuditLevelInfo, AuditLevelWarning},
			},
			entry: &AuditEntry{
				ID:    "audit-1",
				Level: AuditLevelInfo,
			},
			want: true,
		},
		{
			name: "level no match",
			filter: &AuditFilter{
				Levels: []AuditLevel{AuditLevelError},
			},
			entry: &AuditEntry{
				ID:    "audit-1",
				Level: AuditLevelInfo,
			},
			want: false,
		},
		{
			name: "action match",
			filter: &AuditFilter{
				Actions: []AuditAction{AuditActionSagaStarted, AuditActionSagaCompleted},
			},
			entry: &AuditEntry{
				ID:     "audit-1",
				Action: AuditActionSagaStarted,
			},
			want: true,
		},
		{
			name: "resource type match",
			filter: &AuditFilter{
				ResourceType: "saga",
			},
			entry: &AuditEntry{
				ID:           "audit-1",
				ResourceType: "saga",
			},
			want: true,
		},
		{
			name: "resource id match",
			filter: &AuditFilter{
				ResourceID: "saga-123",
			},
			entry: &AuditEntry{
				ID:         "audit-1",
				ResourceID: "saga-123",
			},
			want: true,
		},
		{
			name: "user id match",
			filter: &AuditFilter{
				UserID: "user-1",
			},
			entry: &AuditEntry{
				ID:     "audit-1",
				UserID: "user-1",
			},
			want: true,
		},
		{
			name: "source match",
			filter: &AuditFilter{
				Source: "saga-coordinator",
			},
			entry: &AuditEntry{
				ID:     "audit-1",
				Source: "saga-coordinator",
			},
			want: true,
		},
		{
			name: "multiple criteria match",
			filter: &AuditFilter{
				Levels:       []AuditLevel{AuditLevelInfo},
				Actions:      []AuditAction{AuditActionSagaStarted},
				ResourceType: "saga",
				UserID:       "user-1",
			},
			entry: &AuditEntry{
				ID:           "audit-1",
				Level:        AuditLevelInfo,
				Action:       AuditActionSagaStarted,
				ResourceType: "saga",
				UserID:       "user-1",
			},
			want: true,
		},
		{
			name: "multiple criteria no match",
			filter: &AuditFilter{
				Levels:  []AuditLevel{AuditLevelInfo},
				Actions: []AuditAction{AuditActionSagaCompleted}, // Wrong action
			},
			entry: &AuditEntry{
				ID:     "audit-1",
				Level:  AuditLevelInfo,
				Action: AuditActionSagaStarted,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Match(tt.entry)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDefaultAuditLogger_Log(t *testing.T) {
	storage := NewMemoryAuditStorage()
	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage: storage,
		Source:  "test-source",
	})
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		entry   *AuditEntry
		wantErr bool
	}{
		{
			name: "log basic entry",
			entry: &AuditEntry{
				Level:        AuditLevelInfo,
				Action:       AuditActionSagaStarted,
				ResourceType: "saga",
				ResourceID:   "saga-123",
				Message:      "Saga started",
			},
			wantErr: false,
		},
		{
			name: "log entry with details",
			entry: &AuditEntry{
				Level:        AuditLevelWarning,
				Action:       AuditActionStepFailed,
				ResourceType: "saga",
				ResourceID:   "saga-123",
				Message:      "Step failed",
				Details: map[string]interface{}{
					"step_id": "step-1",
					"error":   "network timeout",
				},
			},
			wantErr: false,
		},
		{
			name: "log entry auto-fills defaults",
			entry: &AuditEntry{
				Level:   AuditLevelInfo,
				Action:  AuditActionSagaCompleted,
				Message: "Saga completed",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := logger.Log(ctx, tt.entry)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify defaults were set
				assert.NotEmpty(t, tt.entry.ID)
				assert.False(t, tt.entry.Timestamp.IsZero())
				assert.Equal(t, "test-source", tt.entry.Source)
			}
		})
	}

	// Verify entries were stored
	entries, err := storage.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, len(tests), len(entries))
}

func TestDefaultAuditLogger_LogAction(t *testing.T) {
	storage := NewMemoryAuditStorage()
	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage: storage,
		Source:  "test-source",
	})
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()

	err = logger.LogAction(ctx,
		AuditLevelInfo,
		AuditActionSagaStarted,
		"saga",
		"saga-123",
		"Saga started successfully",
		map[string]interface{}{"definition_id": "order-saga"},
	)
	assert.NoError(t, err)

	// Verify entry was logged
	entries, err := storage.Query(ctx, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, AuditLevelInfo, entry.Level)
	assert.Equal(t, AuditActionSagaStarted, entry.Action)
	assert.Equal(t, "saga", entry.ResourceType)
	assert.Equal(t, "saga-123", entry.ResourceID)
	assert.Equal(t, "Saga started successfully", entry.Message)
	assert.NotNil(t, entry.Details)
}

func TestDefaultAuditLogger_LogStateChange(t *testing.T) {
	storage := NewMemoryAuditStorage()
	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage: storage,
		Source:  "test-source",
	})
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()

	err = logger.LogStateChange(ctx,
		"saga",
		"saga-123",
		saga.StatePending,
		saga.StateRunning,
		"user-1",
		map[string]interface{}{"trigger": "api_call"},
	)
	assert.NoError(t, err)

	// Verify entry was logged
	entries, err := storage.Query(ctx, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, AuditLevelInfo, entry.Level)
	assert.Equal(t, AuditActionStateChanged, entry.Action)
	assert.Equal(t, "saga", entry.ResourceType)
	assert.Equal(t, "saga-123", entry.ResourceID)
	assert.Equal(t, "user-1", entry.UserID)
	assert.Equal(t, "pending", entry.OldState)
	assert.Equal(t, "running", entry.NewState)
}

func TestDefaultAuditLogger_LogAuthEvent(t *testing.T) {
	storage := NewMemoryAuditStorage()
	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage: storage,
		Source:  "test-source",
	})
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		action  AuditAction
		userID  string
		success bool
		level   AuditLevel
	}{
		{
			name:    "successful auth",
			action:  AuditActionAuthSuccess,
			userID:  "user-1",
			success: true,
			level:   AuditLevelInfo,
		},
		{
			name:    "failed auth",
			action:  AuditActionAuthFailure,
			userID:  "user-2",
			success: false,
			level:   AuditLevelWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := logger.LogAuthEvent(ctx, tt.action, tt.userID, tt.success, map[string]interface{}{
				"ip": "192.168.1.1",
			})
			assert.NoError(t, err)
		})
	}

	// Verify entries were logged
	entries, err := storage.Query(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, len(tests), len(entries))
}

func TestDefaultAuditLogger_Query(t *testing.T) {
	storage := NewMemoryAuditStorage()
	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage: storage,
		Source:  "test-source",
	})
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()

	// Log multiple entries
	entries := []*AuditEntry{
		{
			Level:        AuditLevelInfo,
			Action:       AuditActionSagaStarted,
			ResourceType: "saga",
			ResourceID:   "saga-1",
			UserID:       "user-1",
			Message:      "Saga 1 started",
		},
		{
			Level:        AuditLevelInfo,
			Action:       AuditActionSagaCompleted,
			ResourceType: "saga",
			ResourceID:   "saga-1",
			UserID:       "user-1",
			Message:      "Saga 1 completed",
		},
		{
			Level:        AuditLevelWarning,
			Action:       AuditActionAuthFailure,
			ResourceType: "auth",
			UserID:       "user-2",
			Message:      "Auth failed",
		},
	}

	for _, entry := range entries {
		err := logger.Log(ctx, entry)
		require.NoError(t, err)
	}

	tests := []struct {
		name      string
		filter    *AuditFilter
		wantCount int
	}{
		{
			name:      "query all",
			filter:    nil,
			wantCount: 3,
		},
		{
			name: "filter by level",
			filter: &AuditFilter{
				Levels: []AuditLevel{AuditLevelWarning},
			},
			wantCount: 1,
		},
		{
			name: "filter by action",
			filter: &AuditFilter{
				Actions: []AuditAction{AuditActionSagaStarted},
			},
			wantCount: 1,
		},
		{
			name: "filter by resource type",
			filter: &AuditFilter{
				ResourceType: "saga",
			},
			wantCount: 2,
		},
		{
			name: "filter by user",
			filter: &AuditFilter{
				UserID: "user-1",
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := logger.Query(ctx, tt.filter)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, len(results))
		})
	}
}

func TestDefaultAuditLogger_Count(t *testing.T) {
	storage := NewMemoryAuditStorage()
	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage: storage,
		Source:  "test-source",
	})
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()

	// Log some entries
	for i := 0; i < 5; i++ {
		err := logger.LogAction(ctx,
			AuditLevelInfo,
			AuditActionSagaStarted,
			"saga",
			"saga-123",
			"Saga started",
			nil,
		)
		require.NoError(t, err)
	}

	// Count all entries
	count, err := logger.Count(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)

	// Count with filter
	count, err = logger.Count(ctx, &AuditFilter{
		Levels: []AuditLevel{AuditLevelInfo},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestDefaultAuditLogger_WithAuthContext(t *testing.T) {
	storage := NewMemoryAuditStorage()
	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage: storage,
		Source:  "test-source",
	})
	require.NoError(t, err)
	defer logger.Close()

	// Create auth context
	authCtx := &AuthContext{
		Credentials: &AuthCredentials{
			UserID: "user-123",
		},
		SagaID:        "saga-456",
		CorrelationID: "corr-789",
	}

	// Add auth context to context
	ctx := ContextWithAuth(context.Background(), authCtx)

	// Log an entry
	err = logger.LogAction(ctx,
		AuditLevelInfo,
		AuditActionSagaStarted,
		"saga",
		"saga-456",
		"Saga started",
		nil,
	)
	require.NoError(t, err)

	// Verify entry has auth context info
	entries, err := storage.Query(ctx, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, "user-123", entry.UserID)
	assert.Equal(t, "saga-456", entry.TraceID)
	assert.Equal(t, "corr-789", entry.CorrelationID)
}

func TestDefaultAuditLogger_WithEnricher(t *testing.T) {
	storage := NewMemoryAuditStorage()

	// Create enricher function
	enricher := func(entry *AuditEntry) {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]string)
		}
		entry.Metadata["enriched"] = "true"
		entry.Metadata["version"] = "1.0"
	}

	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage:  storage,
		Source:   "test-source",
		Enricher: enricher,
	})
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()

	// Log an entry
	err = logger.LogAction(ctx,
		AuditLevelInfo,
		AuditActionSagaStarted,
		"saga",
		"saga-123",
		"Saga started",
		nil,
	)
	require.NoError(t, err)

	// Verify entry was enriched
	entries, err := storage.Query(ctx, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.NotNil(t, entry.Metadata)
	assert.Equal(t, "true", entry.Metadata["enriched"])
	assert.Equal(t, "1.0", entry.Metadata["version"])
}

func TestDefaultAuditLogger_Close(t *testing.T) {
	storage := NewMemoryAuditStorage()
	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage: storage,
		Source:  "test-source",
	})
	require.NoError(t, err)

	// Close the logger
	err = logger.Close()
	assert.NoError(t, err)

	// Verify subsequent operations fail
	ctx := context.Background()
	err = logger.LogAction(ctx,
		AuditLevelInfo,
		AuditActionSagaStarted,
		"saga",
		"saga-123",
		"Saga started",
		nil,
	)
	assert.Error(t, err)
}

func TestSagaAuditHook_OnStateChange(t *testing.T) {
	storage := NewMemoryAuditStorage()
	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage: storage,
		Source:  "test-source",
	})
	require.NoError(t, err)
	defer logger.Close()

	hook := NewSagaAuditHook(logger)

	ctx := context.Background()

	// Simulate state change
	hook.OnStateChange(ctx, "saga-123", saga.StatePending, saga.StateRunning, map[string]interface{}{
		"trigger": "api_call",
	})

	// Verify audit entry was created
	entries, err := storage.Query(ctx, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, AuditActionStateChanged, entry.Action)
	assert.Equal(t, "saga", entry.ResourceType)
	assert.Equal(t, "saga-123", entry.ResourceID)
	assert.Equal(t, "pending", entry.OldState)
	assert.Equal(t, "running", entry.NewState)
}

func TestSagaAuditHook_OnStateChangeWithAuth(t *testing.T) {
	storage := NewMemoryAuditStorage()
	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage: storage,
		Source:  "test-source",
	})
	require.NoError(t, err)
	defer logger.Close()

	hook := NewSagaAuditHook(logger)

	// Create auth context
	authCtx := &AuthContext{
		Credentials: &AuthCredentials{
			UserID: "user-456",
		},
	}
	ctx := ContextWithAuth(context.Background(), authCtx)

	// Simulate state change
	hook.OnStateChange(ctx, "saga-123", saga.StateRunning, saga.StateCompleted, nil)

	// Verify audit entry includes user info
	entries, err := storage.Query(ctx, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, "user-456", entry.UserID)
}

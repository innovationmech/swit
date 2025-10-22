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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryAuditStorage_Write(t *testing.T) {
	storage := NewMemoryAuditStorage()
	ctx := context.Background()

	entry := &AuditEntry{
		ID:           "audit-1",
		Timestamp:    time.Now(),
		Level:        AuditLevelInfo,
		Action:       AuditActionSagaStarted,
		ResourceType: "saga",
		ResourceID:   "saga-123",
		Message:      "Saga started",
	}

	err := storage.Write(ctx, entry)
	assert.NoError(t, err)

	// Verify entry was stored
	assert.Len(t, storage.entries, 1)
	assert.Equal(t, entry.ID, storage.entries[0].ID)
}

func TestMemoryAuditStorage_Query(t *testing.T) {
	storage := NewMemoryAuditStorage()
	ctx := context.Background()

	// Add test entries
	entries := []*AuditEntry{
		{
			ID:           "audit-1",
			Timestamp:    time.Now().Add(-2 * time.Hour),
			Level:        AuditLevelInfo,
			Action:       AuditActionSagaStarted,
			ResourceType: "saga",
			ResourceID:   "saga-1",
			UserID:       "user-1",
			Message:      "Saga 1 started",
		},
		{
			ID:           "audit-2",
			Timestamp:    time.Now().Add(-1 * time.Hour),
			Level:        AuditLevelWarning,
			Action:       AuditActionStepFailed,
			ResourceType: "saga",
			ResourceID:   "saga-2",
			UserID:       "user-2",
			Message:      "Step failed",
		},
		{
			ID:           "audit-3",
			Timestamp:    time.Now(),
			Level:        AuditLevelInfo,
			Action:       AuditActionSagaCompleted,
			ResourceType: "saga",
			ResourceID:   "saga-1",
			UserID:       "user-1",
			Message:      "Saga 1 completed",
		},
	}

	for _, entry := range entries {
		err := storage.Write(ctx, entry)
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
				Actions: []AuditAction{AuditActionSagaStarted, AuditActionSagaCompleted},
			},
			wantCount: 2,
		},
		{
			name: "filter by resource id",
			filter: &AuditFilter{
				ResourceID: "saga-1",
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
		{
			name: "pagination - limit",
			filter: &AuditFilter{
				Limit: 2,
			},
			wantCount: 2,
		},
		{
			name: "pagination - offset",
			filter: &AuditFilter{
				Offset: 1,
			},
			wantCount: 2,
		},
		{
			name: "pagination - limit and offset",
			filter: &AuditFilter{
				Limit:  1,
				Offset: 1,
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := storage.Query(ctx, tt.filter)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, len(results))
		})
	}
}

func TestMemoryAuditStorage_Count(t *testing.T) {
	storage := NewMemoryAuditStorage()
	ctx := context.Background()

	// Add test entries
	for i := 0; i < 5; i++ {
		entry := &AuditEntry{
			ID:           "audit-" + string(rune('1'+i)),
			Timestamp:    time.Now(),
			Level:        AuditLevelInfo,
			Action:       AuditActionSagaStarted,
			ResourceType: "saga",
			Message:      "Test entry",
		}
		err := storage.Write(ctx, entry)
		require.NoError(t, err)
	}

	// Count all entries
	count, err := storage.Count(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)

	// Count with filter
	count, err = storage.Count(ctx, &AuditFilter{
		Levels: []AuditLevel{AuditLevelInfo},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestMemoryAuditStorage_Cleanup(t *testing.T) {
	storage := NewMemoryAuditStorage()
	ctx := context.Background()

	// Add old and new entries
	oldEntry := &AuditEntry{
		ID:        "audit-old",
		Timestamp: time.Now().Add(-2 * time.Hour),
		Level:     AuditLevelInfo,
		Action:    AuditActionSagaStarted,
		Message:   "Old entry",
	}
	newEntry := &AuditEntry{
		ID:        "audit-new",
		Timestamp: time.Now(),
		Level:     AuditLevelInfo,
		Action:    AuditActionSagaStarted,
		Message:   "New entry",
	}

	err := storage.Write(ctx, oldEntry)
	require.NoError(t, err)
	err = storage.Write(ctx, newEntry)
	require.NoError(t, err)

	// Cleanup entries older than 1 hour
	err = storage.Cleanup(ctx, 1*time.Hour)
	assert.NoError(t, err)

	// Verify only new entry remains
	entries, err := storage.Query(ctx, nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "audit-new", entries[0].ID)
}

func TestFileAuditStorage_Write(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	storage, err := NewFileAuditStorage(&FileAuditStorageConfig{
		FilePath:    filePath,
		MaxFileSize: 1024 * 1024, // 1MB
		MaxBackups:  3,
	})
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	entry := &AuditEntry{
		ID:           "audit-1",
		Timestamp:    time.Now(),
		Level:        AuditLevelInfo,
		Action:       AuditActionSagaStarted,
		ResourceType: "saga",
		ResourceID:   "saga-123",
		Message:      "Saga started",
	}

	err = storage.Write(ctx, entry)
	assert.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(filePath)
	assert.NoError(t, err)
}

func TestFileAuditStorage_Query(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	storage, err := NewFileAuditStorage(&FileAuditStorageConfig{
		FilePath:    filePath,
		MaxFileSize: 1024 * 1024,
		MaxBackups:  3,
	})
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Write test entries
	entries := []*AuditEntry{
		{
			ID:           "audit-1",
			Timestamp:    time.Now(),
			Level:        AuditLevelInfo,
			Action:       AuditActionSagaStarted,
			ResourceType: "saga",
			ResourceID:   "saga-1",
			Message:      "Saga 1 started",
		},
		{
			ID:           "audit-2",
			Timestamp:    time.Now(),
			Level:        AuditLevelWarning,
			Action:       AuditActionStepFailed,
			ResourceType: "saga",
			ResourceID:   "saga-2",
			Message:      "Step failed",
		},
	}

	for _, entry := range entries {
		err := storage.Write(ctx, entry)
		require.NoError(t, err)
	}

	// Query all entries
	results, err := storage.Query(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Query with filter
	results, err = storage.Query(ctx, &AuditFilter{
		Levels: []AuditLevel{AuditLevelWarning},
	})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "audit-2", results[0].ID)
}

func TestFileAuditStorage_Rotate(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	storage, err := NewFileAuditStorage(&FileAuditStorageConfig{
		FilePath:    filePath,
		MaxFileSize: 100, // Small size to force rotation
		MaxBackups:  3,
	})
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Write entries to trigger rotation
	for i := 0; i < 10; i++ {
		entry := &AuditEntry{
			ID:        "audit-" + string(rune('0'+i)),
			Timestamp: time.Now(),
			Level:     AuditLevelInfo,
			Action:    AuditActionSagaStarted,
			Message:   "This is a test message that should fill up the file quickly",
			Details: map[string]interface{}{
				"index": i,
			},
		}
		err := storage.Write(ctx, entry)
		require.NoError(t, err)
	}

	// Verify backup files were created
	backupPath := filePath + ".1"
	_, err = os.Stat(backupPath)
	assert.NoError(t, err)
}

func TestFileAuditStorage_Cleanup(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	storage, err := NewFileAuditStorage(&FileAuditStorageConfig{
		FilePath:    filePath,
		MaxFileSize: 1024 * 1024,
		MaxBackups:  3,
	})
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Create a backup file with old timestamp
	backupPath := filePath + ".1"
	file, err := os.Create(backupPath)
	require.NoError(t, err)
	file.Close()

	// Change file modification time to 2 hours ago
	oldTime := time.Now().Add(-2 * time.Hour)
	err = os.Chtimes(backupPath, oldTime, oldTime)
	require.NoError(t, err)

	// Cleanup old backups
	err = storage.Cleanup(ctx, 1*time.Hour)
	assert.NoError(t, err)

	// Verify backup was removed
	_, err = os.Stat(backupPath)
	assert.True(t, os.IsNotExist(err))
}

func TestFileAuditStorage_SortingAndPagination(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	storage, err := NewFileAuditStorage(&FileAuditStorageConfig{
		FilePath:    filePath,
		MaxFileSize: 1024 * 1024,
		MaxBackups:  3,
	})
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Write entries with different timestamps
	baseTime := time.Now()
	for i := 0; i < 5; i++ {
		entry := &AuditEntry{
			ID:        "audit-" + string(rune('1'+i)),
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Level:     AuditLevelInfo,
			Action:    AuditActionSagaStarted,
			Message:   "Test entry " + string(rune('1'+i)),
		}
		err := storage.Write(ctx, entry)
		require.NoError(t, err)
	}

	// Test sorting by timestamp ascending
	results, err := storage.Query(ctx, &AuditFilter{
		SortBy:    "timestamp",
		SortOrder: "asc",
	})
	require.NoError(t, err)
	require.Len(t, results, 5)
	assert.Equal(t, "audit-1", results[0].ID)
	assert.Equal(t, "audit-5", results[4].ID)

	// Test sorting by timestamp descending
	results, err = storage.Query(ctx, &AuditFilter{
		SortBy:    "timestamp",
		SortOrder: "desc",
	})
	require.NoError(t, err)
	require.Len(t, results, 5)
	assert.Equal(t, "audit-5", results[0].ID)
	assert.Equal(t, "audit-1", results[4].ID)

	// Test pagination
	results, err = storage.Query(ctx, &AuditFilter{
		Limit:  2,
		Offset: 1,
	})
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestMemoryAuditStorage_Sorting(t *testing.T) {
	storage := NewMemoryAuditStorage()
	ctx := context.Background()

	// Add entries with different levels
	entries := []*AuditEntry{
		{
			ID:        "audit-1",
			Timestamp: time.Now(),
			Level:     AuditLevelWarning,
			Action:    AuditActionSagaStarted,
			Message:   "Entry 1",
		},
		{
			ID:        "audit-2",
			Timestamp: time.Now().Add(1 * time.Minute),
			Level:     AuditLevelInfo,
			Action:    AuditActionSagaCompleted,
			Message:   "Entry 2",
		},
		{
			ID:        "audit-3",
			Timestamp: time.Now().Add(2 * time.Minute),
			Level:     AuditLevelError,
			Action:    AuditActionSagaFailed,
			Message:   "Entry 3",
		},
	}

	for _, entry := range entries {
		err := storage.Write(ctx, entry)
		require.NoError(t, err)
	}

	// Test sort by timestamp ascending
	results, err := storage.Query(ctx, &AuditFilter{
		SortBy:    "timestamp",
		SortOrder: "asc",
	})
	require.NoError(t, err)
	assert.Equal(t, "audit-1", results[0].ID)
	assert.Equal(t, "audit-3", results[2].ID)

	// Test sort by timestamp descending
	results, err = storage.Query(ctx, &AuditFilter{
		SortBy:    "timestamp",
		SortOrder: "desc",
	})
	require.NoError(t, err)
	assert.Equal(t, "audit-3", results[0].ID)
	assert.Equal(t, "audit-1", results[2].ID)

	// Test sort by level
	results, err = storage.Query(ctx, &AuditFilter{
		SortBy:    "level",
		SortOrder: "asc",
	})
	require.NoError(t, err)
	// Error < Info < Warning (lexicographically)
	assert.Equal(t, AuditLevelError, results[0].Level)
}

func TestMemoryAuditStorage_ConcurrentAccess(t *testing.T) {
	storage := NewMemoryAuditStorage()
	ctx := context.Background()

	// Write entries concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			entry := &AuditEntry{
				ID:        "audit-" + string(rune('0'+index)),
				Timestamp: time.Now(),
				Level:     AuditLevelInfo,
				Action:    AuditActionSagaStarted,
				Message:   "Concurrent entry",
			}
			err := storage.Write(ctx, entry)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all entries were written
	count, err := storage.Count(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)
}

func TestFileAuditStorage_InvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *FileAuditStorageConfig
		wantErr bool
	}{
		{
			name: "empty file path",
			config: &FileAuditStorageConfig{
				FilePath: "",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &FileAuditStorageConfig{
				FilePath:    filepath.Join(t.TempDir(), "audit.log"),
				MaxFileSize: 1024 * 1024,
				MaxBackups:  3,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewFileAuditStorage(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if storage != nil {
					storage.Close()
				}
			}
		})
	}
}

func TestFileAuditStorage_MaxBackups(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	maxBackups := 2
	storage, err := NewFileAuditStorage(&FileAuditStorageConfig{
		FilePath:    filePath,
		MaxFileSize: 50, // Very small to force multiple rotations
		MaxBackups:  maxBackups,
	})
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Write many entries to force multiple rotations
	for i := 0; i < 50; i++ {
		entry := &AuditEntry{
			ID:        "audit-" + string(rune('0'+i%10)),
			Timestamp: time.Now(),
			Level:     AuditLevelInfo,
			Action:    AuditActionSagaStarted,
			Message:   "Entry to force rotation with some extra text",
		}
		err := storage.Write(ctx, entry)
		require.NoError(t, err)
	}

	// Count backup files
	backupCount := 0
	for i := 1; i <= maxBackups+5; i++ {
		backupPath := filePath + "." + string(rune('0'+i))
		if _, err := os.Stat(backupPath); err == nil {
			backupCount++
		}
	}

	// Verify we don't have more than maxBackups
	assert.LessOrEqual(t, backupCount, maxBackups)
}

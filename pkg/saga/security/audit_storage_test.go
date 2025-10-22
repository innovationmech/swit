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
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver for testing
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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

func TestDatabaseAuditStorage_TimeFilters(t *testing.T) {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create database audit storage
	storage, err := NewDatabaseAuditStorage(&DatabaseAuditStorageConfig{
		DB:         db,
		TableName:  "test_audit_logs",
		DriverName: "sqlite3",
	})
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Create test entries with different timestamps
	entries := []*AuditEntry{
		{
			ID:        "audit-1",
			Timestamp: now.Add(-2 * time.Hour), // 2 hours ago
			Level:     AuditLevelInfo,
			Action:    AuditActionSagaStarted,
			Message:   "Old entry",
		},
		{
			ID:        "audit-2",
			Timestamp: now.Add(-1 * time.Hour), // 1 hour ago
			Level:     AuditLevelInfo,
			Action:    AuditActionSagaCompleted,
			Message:   "Medium entry",
		},
		{
			ID:        "audit-3",
			Timestamp: now, // now
			Level:     AuditLevelInfo,
			Action:    AuditActionSagaStarted,
			Message:   "Recent entry",
		},
	}

	// Write test entries
	for _, entry := range entries {
		err := storage.Write(ctx, entry)
		require.NoError(t, err)
	}

	// Test Query with time range filter
	startTime := now.Add(-90 * time.Minute) // 90 minutes ago
	endTime := now.Add(30 * time.Minute)    // 30 minutes from now

	filter := &AuditFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	results, err := storage.Query(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, results, 2, "Should return 2 entries within time range")

	// Verify the entries are within the time range
	for _, entry := range results {
		assert.True(t, entry.Timestamp.After(startTime) || entry.Timestamp.Equal(startTime))
		assert.True(t, entry.Timestamp.Before(endTime) || entry.Timestamp.Equal(endTime))
	}

	// Test Count with time range filter
	count, err := storage.Count(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count, "Count should return 2 entries within time range")

	// Test Query with only start time
	filterOnlyStart := &AuditFilter{
		StartTime: &startTime,
	}

	results, err = storage.Query(ctx, filterOnlyStart)
	require.NoError(t, err)
	assert.Len(t, results, 2, "Should return 2 entries after start time")

	count, err = storage.Count(ctx, filterOnlyStart)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count, "Count should return 2 entries after start time")

	// Test Query with only end time
	filterOnlyEnd := &AuditFilter{
		EndTime: &startTime,
	}

	results, err = storage.Query(ctx, filterOnlyEnd)
	require.NoError(t, err)
	assert.Len(t, results, 1, "Should return 1 entry before end time")

	count, err = storage.Count(ctx, filterOnlyEnd)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "Count should return 1 entry before end time")
}

func TestDatabaseAuditStorage_SQLDialects(t *testing.T) {
	testCases := []struct {
		name       string
		driverName string
		wantQuery  string
	}{
		{
			name:       "MySQL/SQLite dialect",
			driverName: "mysql",
			wantQuery:  "INSERT INTO audit_logs (id, timestamp, level) VALUES (?, ?, ?)",
		},
		{
			name:       "PostgreSQL dialect",
			driverName: "postgres",
			wantQuery:  "INSERT INTO audit_logs (id, timestamp, level) VALUES ($1, $2, $3)",
		},
		{
			name:       "pgx driver dialect",
			driverName: "pgx",
			wantQuery:  "INSERT INTO audit_logs (id, timestamp, level) VALUES ($1, $2, $3)",
		},
		{
			name:       "lib/pq driver dialect",
			driverName: "pq",
			wantQuery:  "INSERT INTO audit_logs (id, timestamp, level) VALUES ($1, $2, $3)",
		},
		{
			name:       "SQLite dialect",
			driverName: "sqlite3",
			wantQuery:  "INSERT INTO audit_logs (id, timestamp, level) VALUES (?, ?, ?)",
		},
		{
			name:       "Unknown driver defaults to ?",
			driverName: "unknown_driver",
			wantQuery:  "INSERT INTO audit_logs (id, timestamp, level) VALUES (?, ?, ?)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create storage instance with specific driver
			storage := &DatabaseAuditStorage{
				driverName: tc.driverName,
				tableName:  "audit_logs",
				logger:     zap.NewNop(),
			}

			// Test buildPlaceholders method
			gotPlaceholders := storage.buildPlaceholders(3)
			expectedPlaceholders := "?"
			if tc.driverName == "postgres" || tc.driverName == "pgx" || tc.driverName == "pq" {
				expectedPlaceholders = "$1, $2, $3"
			} else {
				expectedPlaceholders = "?, ?, ?"
			}
			assert.Equal(t, expectedPlaceholders, gotPlaceholders,
				"buildPlaceholders should generate correct placeholders for driver %s", tc.driverName)

			// Test getPlaceholder method
			gotPlaceholder1 := storage.getPlaceholder(1)
			gotPlaceholder2 := storage.getPlaceholder(2)

			if tc.driverName == "postgres" || tc.driverName == "pgx" || tc.driverName == "pq" {
				assert.Equal(t, "$1", gotPlaceholder1)
				assert.Equal(t, "$2", gotPlaceholder2)
			} else {
				assert.Equal(t, "?", gotPlaceholder1)
				assert.Equal(t, "?", gotPlaceholder2)
			}

			// Test that INSERT query generation would work
			query := fmt.Sprintf("INSERT INTO %s (id, timestamp, level) VALUES (%s)",
				storage.tableName, storage.buildPlaceholders(3))
			assert.Equal(t, tc.wantQuery, query,
				"INSERT query should use correct placeholders for driver %s", tc.driverName)
		})
	}
}

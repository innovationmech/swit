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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// MemoryAuditStorage implements in-memory audit log storage
type MemoryAuditStorage struct {
	entries []*AuditEntry
	mu      sync.RWMutex
}

// NewMemoryAuditStorage creates a new in-memory audit storage
func NewMemoryAuditStorage() *MemoryAuditStorage {
	return &MemoryAuditStorage{
		entries: make([]*AuditEntry, 0),
	}
}

// Write writes an audit entry to memory
func (s *MemoryAuditStorage) Write(ctx context.Context, entry *AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a copy to avoid external modifications
	entryCopy := *entry
	s.entries = append(s.entries, &entryCopy)

	return nil
}

// Query retrieves audit entries based on filter criteria
func (s *MemoryAuditStorage) Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*AuditEntry

	for _, entry := range s.entries {
		if filter == nil || filter.Match(entry) {
			// Create a copy to avoid external modifications
			entryCopy := *entry
			results = append(results, &entryCopy)
		}
	}

	// Sort results
	if filter != nil && filter.SortBy != "" {
		s.sortResults(results, filter.SortBy, filter.SortOrder)
	}

	// Apply pagination
	if filter != nil && (filter.Limit > 0 || filter.Offset > 0) {
		results = s.paginate(results, filter.Limit, filter.Offset)
	}

	return results, nil
}

// Count returns the total number of audit entries matching the filter
func (s *MemoryAuditStorage) Count(ctx context.Context, filter *AuditFilter) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if filter == nil {
		return int64(len(s.entries)), nil
	}

	count := int64(0)
	for _, entry := range s.entries {
		if filter.Match(entry) {
			count++
		}
	}

	return count, nil
}

// Rotate is a no-op for memory storage
func (s *MemoryAuditStorage) Rotate(ctx context.Context) error {
	return nil
}

// Cleanup removes audit entries older than the specified duration
func (s *MemoryAuditStorage) Cleanup(ctx context.Context, olderThan time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoffTime := time.Now().Add(-olderThan)
	newEntries := make([]*AuditEntry, 0)

	for _, entry := range s.entries {
		if entry.Timestamp.After(cutoffTime) {
			newEntries = append(newEntries, entry)
		}
	}

	s.entries = newEntries
	return nil
}

// Close closes the storage
func (s *MemoryAuditStorage) Close() error {
	return nil
}

// sortResults sorts audit entries based on the specified field
func (s *MemoryAuditStorage) sortResults(entries []*AuditEntry, sortBy, sortOrder string) {
	sort.Slice(entries, func(i, j int) bool {
		var result bool

		switch sortBy {
		case "timestamp":
			result = entries[i].Timestamp.Before(entries[j].Timestamp)
		case "level":
			result = entries[i].Level < entries[j].Level
		case "action":
			result = entries[i].Action < entries[j].Action
		default:
			result = entries[i].Timestamp.Before(entries[j].Timestamp)
		}

		if sortOrder == "desc" {
			return !result
		}
		return result
	})
}

// paginate applies pagination to the results
func (s *MemoryAuditStorage) paginate(entries []*AuditEntry, limit, offset int) []*AuditEntry {
	if offset >= len(entries) {
		return []*AuditEntry{}
	}

	start := offset
	end := len(entries)

	if limit > 0 && start+limit < end {
		end = start + limit
	}

	return entries[start:end]
}

// FileAuditStorage implements file-based audit log storage
type FileAuditStorage struct {
	filePath       string
	maxFileSize    int64 // Maximum file size in bytes before rotation
	maxBackups     int   // Maximum number of backup files to keep
	file           *os.File
	currentSize    int64
	mu             sync.Mutex
	logger         *zap.Logger
	rotateOnWrite  bool
	compressionFmt string // "gzip" or empty
}

// FileAuditStorageConfig configures file-based audit storage
type FileAuditStorageConfig struct {
	FilePath       string
	MaxFileSize    int64  // in bytes, default 100MB
	MaxBackups     int    // default 10
	CompressionFmt string // "gzip" or empty
}

// NewFileAuditStorage creates a new file-based audit storage
func NewFileAuditStorage(config *FileAuditStorageConfig) (*FileAuditStorage, error) {
	if config.FilePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	if config.MaxFileSize <= 0 {
		config.MaxFileSize = 100 * 1024 * 1024 // 100MB default
	}

	if config.MaxBackups <= 0 {
		config.MaxBackups = 10
	}

	// Ensure directory exists
	dir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open or create the file
	file, err := os.OpenFile(config.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	// Get current file size
	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	storage := &FileAuditStorage{
		filePath:       config.FilePath,
		maxFileSize:    config.MaxFileSize,
		maxBackups:     config.MaxBackups,
		file:           file,
		currentSize:    fileInfo.Size(),
		logger:         logger.Logger,
		compressionFmt: config.CompressionFmt,
	}

	// Initialize logger if not set
	if storage.logger == nil {
		storage.logger = zap.NewNop()
	}

	return storage, nil
}

// Write writes an audit entry to the file
func (s *FileAuditStorage) Write(ctx context.Context, entry *AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if rotation is needed
	if s.currentSize >= s.maxFileSize {
		if err := s.rotateNoLock(); err != nil {
			s.logger.Error("Failed to rotate audit log file", zap.Error(err))
			// Continue writing even if rotation fails
		}
	}

	// Convert entry to JSON
	data, err := entry.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize audit entry: %w", err)
	}

	// Write to file with newline
	line := data + "\n"
	n, err := s.file.WriteString(line)
	if err != nil {
		return fmt.Errorf("failed to write audit entry: %w", err)
	}

	s.currentSize += int64(n)

	// Sync to disk
	if err := s.file.Sync(); err != nil {
		s.logger.Warn("Failed to sync audit log file", zap.Error(err))
	}

	return nil
}

// Query retrieves audit entries from the file based on filter criteria
func (s *FileAuditStorage) Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read all entries from file
	entries, err := s.readAllEntriesNoLock()
	if err != nil {
		return nil, err
	}

	// Filter entries
	var results []*AuditEntry
	for _, entry := range entries {
		if filter == nil || filter.Match(entry) {
			results = append(results, entry)
		}
	}

	// Sort and paginate
	storage := &MemoryAuditStorage{entries: results}
	if filter != nil && filter.SortBy != "" {
		storage.sortResults(results, filter.SortBy, filter.SortOrder)
	}

	if filter != nil && (filter.Limit > 0 || filter.Offset > 0) {
		results = storage.paginate(results, filter.Limit, filter.Offset)
	}

	return results, nil
}

// Count returns the total number of audit entries matching the filter
func (s *FileAuditStorage) Count(ctx context.Context, filter *AuditFilter) (int64, error) {
	entries, err := s.Query(ctx, filter)
	if err != nil {
		return 0, err
	}
	return int64(len(entries)), nil
}

// Rotate performs log rotation
func (s *FileAuditStorage) Rotate(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.rotateNoLock()
}

// rotateNoLock performs rotation without acquiring lock (caller must hold lock)
func (s *FileAuditStorage) rotateNoLock() error {
	// Close current file
	if err := s.file.Close(); err != nil {
		return fmt.Errorf("failed to close current file: %w", err)
	}

	// Rotate backup files
	for i := s.maxBackups - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", s.filePath, i)
		newPath := fmt.Sprintf("%s.%d", s.filePath, i+1)

		if _, err := os.Stat(oldPath); err == nil {
			if err := os.Rename(oldPath, newPath); err != nil {
				s.logger.Warn("Failed to rotate backup file",
					zap.String("old", oldPath),
					zap.String("new", newPath),
					zap.Error(err))
			}
		}
	}

	// Move current file to .1
	backupPath := fmt.Sprintf("%s.1", s.filePath)
	if err := os.Rename(s.filePath, backupPath); err != nil {
		return fmt.Errorf("failed to rename current file: %w", err)
	}

	// Remove oldest backup if exceeds maxBackups
	oldestBackup := fmt.Sprintf("%s.%d", s.filePath, s.maxBackups+1)
	if _, err := os.Stat(oldestBackup); err == nil {
		if err := os.Remove(oldestBackup); err != nil {
			s.logger.Warn("Failed to remove oldest backup",
				zap.String("path", oldestBackup),
				zap.Error(err))
		}
	}

	// Create new file
	file, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new file: %w", err)
	}

	s.file = file
	s.currentSize = 0

	s.logger.Info("Audit log file rotated", zap.String("path", s.filePath))

	return nil
}

// Cleanup removes old backup files
func (s *FileAuditStorage) Cleanup(ctx context.Context, olderThan time.Duration) error {
	// For file storage, this removes backup files older than the specified duration
	cutoffTime := time.Now().Add(-olderThan)

	for i := 1; i <= s.maxBackups; i++ {
		backupPath := fmt.Sprintf("%s.%d", s.filePath, i)
		fileInfo, err := os.Stat(backupPath)
		if err != nil {
			continue // File doesn't exist
		}

		if fileInfo.ModTime().Before(cutoffTime) {
			if err := os.Remove(backupPath); err != nil {
				s.logger.Warn("Failed to remove old backup file",
					zap.String("path", backupPath),
					zap.Error(err))
			} else {
				s.logger.Info("Removed old audit log backup",
					zap.String("path", backupPath))
			}
		}
	}

	return nil
}

// Close closes the file storage
func (s *FileAuditStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

// readAllEntriesNoLock reads all entries from the current file and backups
func (s *FileAuditStorage) readAllEntriesNoLock() ([]*AuditEntry, error) {
	var entries []*AuditEntry

	// Read from current file
	currentEntries, err := s.readFileNoLock(s.filePath)
	if err != nil {
		return nil, err
	}
	entries = append(entries, currentEntries...)

	// Read from backup files
	for i := 1; i <= s.maxBackups; i++ {
		backupPath := fmt.Sprintf("%s.%d", s.filePath, i)
		if _, err := os.Stat(backupPath); err == nil {
			backupEntries, err := s.readFileNoLock(backupPath)
			if err != nil {
				s.logger.Warn("Failed to read backup file",
					zap.String("path", backupPath),
					zap.Error(err))
				continue
			}
			entries = append(entries, backupEntries...)
		}
	}

	return entries, nil
}

// readFileNoLock reads audit entries from a specific file
func (s *FileAuditStorage) readFileNoLock(path string) ([]*AuditEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*AuditEntry{}, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var entries []*AuditEntry
	lines := string(data)

	// Split by newline and parse each entry
	for i, line := range splitLines(lines) {
		if line == "" {
			continue
		}

		var entry AuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			s.logger.Warn("Failed to parse audit entry",
				zap.Int("line", i+1),
				zap.String("file", path),
				zap.Error(err))
			continue
		}

		entries = append(entries, &entry)
	}

	return entries, nil
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0

	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}

// DatabaseAuditStorage implements database-based audit log storage
type DatabaseAuditStorage struct {
	db         *sql.DB
	tableName  string
	logger     *zap.Logger
	mu         sync.RWMutex
	driverName string
}

// DatabaseAuditStorageConfig configures database-based audit storage
type DatabaseAuditStorageConfig struct {
	DB         *sql.DB
	TableName  string
	DriverName string // "postgres", "mysql", etc.
}

// NewDatabaseAuditStorage creates a new database-based audit storage
func NewDatabaseAuditStorage(config *DatabaseAuditStorageConfig) (*DatabaseAuditStorage, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	if config.TableName == "" {
		config.TableName = "audit_logs"
	}

	storage := &DatabaseAuditStorage{
		db:         config.DB,
		tableName:  config.TableName,
		logger:     logger.Logger,
		driverName: config.DriverName,
	}

	// Initialize logger if not set
	if storage.logger == nil {
		storage.logger = zap.NewNop()
	}

	// Initialize table
	if err := storage.initTable(); err != nil {
		return nil, fmt.Errorf("failed to initialize table: %w", err)
	}

	return storage, nil
}

// initTable creates the audit log table if it doesn't exist
func (s *DatabaseAuditStorage) initTable() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) PRIMARY KEY,
			timestamp TIMESTAMP NOT NULL,
			level VARCHAR(50) NOT NULL,
			action VARCHAR(100) NOT NULL,
			resource_type VARCHAR(100),
			resource_id VARCHAR(255),
			user_id VARCHAR(255),
			username VARCHAR(255),
			source VARCHAR(255),
			ip_address VARCHAR(45),
			old_state VARCHAR(100),
			new_state VARCHAR(100),
			message TEXT,
			details TEXT,
			error TEXT,
			trace_id VARCHAR(255),
			span_id VARCHAR(255),
			correlation_id VARCHAR(255),
			metadata TEXT,
			INDEX idx_timestamp (timestamp),
			INDEX idx_action (action),
			INDEX idx_resource (resource_type, resource_id),
			INDEX idx_user (user_id)
		)
	`, s.tableName)

	_, err := s.db.Exec(query)
	return err
}

// Write writes an audit entry to the database
func (s *DatabaseAuditStorage) Write(ctx context.Context, entry *AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize complex fields
	detailsJSON, _ := json.Marshal(entry.Details)
	metadataJSON, _ := json.Marshal(entry.Metadata)

	query := fmt.Sprintf(`
		INSERT INTO %s (
			id, timestamp, level, action, resource_type, resource_id,
			user_id, username, source, ip_address, old_state, new_state,
			message, details, error, trace_id, span_id, correlation_id, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.tableName)

	_, err := s.db.ExecContext(ctx, query,
		entry.ID, entry.Timestamp, entry.Level, entry.Action,
		entry.ResourceType, entry.ResourceID, entry.UserID, entry.Username,
		entry.Source, entry.IPAddress, entry.OldState, entry.NewState,
		entry.Message, string(detailsJSON), entry.Error,
		entry.TraceID, entry.SpanID, entry.CorrelationID, string(metadataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to insert audit entry: %w", err)
	}

	return nil
}

// Query retrieves audit entries from the database based on filter criteria
func (s *DatabaseAuditStorage) Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := fmt.Sprintf("SELECT * FROM %s WHERE 1=1", s.tableName)
	args := []interface{}{}

	// Build WHERE clause
	if filter != nil {
		if filter.StartTime != nil {
			query += " AND timestamp >= ?"
			args = append(args, filter.StartTime)
		}
		if filter.EndTime != nil {
			query += " AND timestamp <= ?"
			args = append(args, filter.EndTime)
		}
		if filter.ResourceType != "" {
			query += " AND resource_type = ?"
			args = append(args, filter.ResourceType)
		}
		if filter.ResourceID != "" {
			query += " AND resource_id = ?"
			args = append(args, filter.ResourceID)
		}
		if filter.UserID != "" {
			query += " AND user_id = ?"
			args = append(args, filter.UserID)
		}
		if filter.Source != "" {
			query += " AND source = ?"
			args = append(args, filter.Source)
		}
	}

	// Add sorting
	if filter != nil && filter.SortBy != "" {
		order := "ASC"
		if filter.SortOrder == "desc" {
			order = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", filter.SortBy, order)
	} else {
		query += " ORDER BY timestamp DESC"
	}

	// Add pagination
	if filter != nil && filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit entries: %w", err)
	}
	defer rows.Close()

	var entries []*AuditEntry
	for rows.Next() {
		entry := &AuditEntry{}
		var detailsJSON, metadataJSON string

		err := rows.Scan(
			&entry.ID, &entry.Timestamp, &entry.Level, &entry.Action,
			&entry.ResourceType, &entry.ResourceID, &entry.UserID, &entry.Username,
			&entry.Source, &entry.IPAddress, &entry.OldState, &entry.NewState,
			&entry.Message, &detailsJSON, &entry.Error,
			&entry.TraceID, &entry.SpanID, &entry.CorrelationID, &metadataJSON,
		)
		if err != nil {
			s.logger.Warn("Failed to scan audit entry", zap.Error(err))
			continue
		}

		// Deserialize JSON fields
		if detailsJSON != "" {
			json.Unmarshal([]byte(detailsJSON), &entry.Details)
		}
		if metadataJSON != "" {
			json.Unmarshal([]byte(metadataJSON), &entry.Metadata)
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// Count returns the total number of audit entries matching the filter
func (s *DatabaseAuditStorage) Count(ctx context.Context, filter *AuditFilter) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE 1=1", s.tableName)
	args := []interface{}{}

	// Build WHERE clause (same as Query)
	if filter != nil {
		if filter.StartTime != nil {
			query += " AND timestamp >= ?"
			args = append(args, filter.StartTime)
		}
		if filter.EndTime != nil {
			query += " AND timestamp <= ?"
			args = append(args, filter.EndTime)
		}
		if filter.ResourceType != "" {
			query += " AND resource_type = ?"
			args = append(args, filter.ResourceType)
		}
		if filter.ResourceID != "" {
			query += " AND resource_id = ?"
			args = append(args, filter.ResourceID)
		}
		if filter.UserID != "" {
			query += " AND user_id = ?"
			args = append(args, filter.UserID)
		}
	}

	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit entries: %w", err)
	}

	return count, nil
}

// Rotate is a no-op for database storage
func (s *DatabaseAuditStorage) Rotate(ctx context.Context) error {
	return nil
}

// Cleanup removes audit entries older than the specified duration
func (s *DatabaseAuditStorage) Cleanup(ctx context.Context, olderThan time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoffTime := time.Now().Add(-olderThan)
	query := fmt.Sprintf("DELETE FROM %s WHERE timestamp < ?", s.tableName)

	result, err := s.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup audit entries: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	s.logger.Info("Cleaned up old audit entries",
		zap.Int64("rows_affected", rowsAffected),
		zap.Time("cutoff_time", cutoffTime))

	return nil
}

// Close closes the database storage
func (s *DatabaseAuditStorage) Close() error {
	// Don't close the DB connection as it may be shared
	return nil
}

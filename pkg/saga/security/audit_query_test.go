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
)

func TestQueryBuilder(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now

	builder := NewQueryBuilder().
		WithTimeRange(startTime, endTime).
		WithLevels(AuditLevelInfo, AuditLevelWarning).
		WithActions(AuditActionSagaStarted).
		WithResource("saga", "saga-123").
		WithUserID("user-1").
		WithSource("test-source").
		WithPagination(10, 0).
		WithSorting("timestamp", "desc")

	filter := builder.Build()

	if filter.StartTime == nil || !filter.StartTime.Equal(startTime) {
		t.Error("StartTime not set correctly")
	}
	if filter.EndTime == nil || !filter.EndTime.Equal(endTime) {
		t.Error("EndTime not set correctly")
	}
	if len(filter.Levels) != 2 {
		t.Errorf("Levels count = %d, want 2", len(filter.Levels))
	}
	if len(filter.Actions) != 1 {
		t.Errorf("Actions count = %d, want 1", len(filter.Actions))
	}
	if filter.ResourceType != "saga" {
		t.Errorf("ResourceType = %v, want saga", filter.ResourceType)
	}
	if filter.ResourceID != "saga-123" {
		t.Errorf("ResourceID = %v, want saga-123", filter.ResourceID)
	}
	if filter.UserID != "user-1" {
		t.Errorf("UserID = %v, want user-1", filter.UserID)
	}
	if filter.Source != "test-source" {
		t.Errorf("Source = %v, want test-source", filter.Source)
	}
	if filter.Limit != 10 {
		t.Errorf("Limit = %d, want 10", filter.Limit)
	}
	if filter.Offset != 0 {
		t.Errorf("Offset = %d, want 0", filter.Offset)
	}
	if filter.SortBy != "timestamp" {
		t.Errorf("SortBy = %v, want timestamp", filter.SortBy)
	}
	if filter.SortOrder != "desc" {
		t.Errorf("SortOrder = %v, want desc", filter.SortOrder)
	}
}

func TestQueryBuilder_IndividualSetters(t *testing.T) {
	now := time.Now()

	builder := NewQueryBuilder().
		WithStartTime(now).
		WithEndTime(now.Add(time.Hour)).
		WithResourceType("saga").
		WithResourceID("saga-123").
		WithLimit(5).
		WithOffset(10)

	filter := builder.Build()

	if filter.StartTime == nil {
		t.Error("StartTime is nil")
	}
	if filter.EndTime == nil {
		t.Error("EndTime is nil")
	}
	if filter.ResourceType != "saga" {
		t.Errorf("ResourceType = %v, want saga", filter.ResourceType)
	}
	if filter.ResourceID != "saga-123" {
		t.Errorf("ResourceID = %v, want saga-123", filter.ResourceID)
	}
	if filter.Limit != 5 {
		t.Errorf("Limit = %d, want 5", filter.Limit)
	}
	if filter.Offset != 10 {
		t.Errorf("Offset = %d, want 10", filter.Offset)
	}
}

func TestAuditQueryService_Query(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add test entries
	entries := createTestEntries(t, ctx, auditLogger)

	// Query all
	filter := NewQueryBuilder().Build()
	results, err := service.Query(ctx, filter)
	if err != nil {
		t.Errorf("Query() error = %v", err)
	}
	if len(results) != len(entries) {
		t.Errorf("Query() returned %d entries, want %d", len(results), len(entries))
	}
}

func TestAuditQueryService_Count(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add test entries
	entries := createTestEntries(t, ctx, auditLogger)

	// Count all
	filter := NewQueryBuilder().Build()
	count, err := service.Count(ctx, filter)
	if err != nil {
		t.Errorf("Count() error = %v", err)
	}
	if count != int64(len(entries)) {
		t.Errorf("Count() = %d, want %d", count, len(entries))
	}
}

func TestAuditQueryService_SearchByText(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add test entries with specific messages
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Order creation started", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaCompleted, "saga", "saga-2", "Payment processing completed", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelError, AuditActionSagaFailed, "saga", "saga-3", "Inventory update failed", nil)

	tests := []struct {
		name       string
		searchText string
		wantCount  int
	}{
		{
			name:       "search 'order'",
			searchText: "order",
			wantCount:  1,
		},
		{
			name:       "search 'failed'",
			searchText: "failed",
			wantCount:  1,
		},
		{
			name:       "search 'completed'",
			searchText: "completed",
			wantCount:  1,
		},
		{
			name:       "search 'saga'",
			searchText: "saga",
			wantCount:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := service.SearchByText(ctx, tt.searchText, nil)
			if err != nil {
				t.Errorf("SearchByText() error = %v", err)
			}
			if len(results) != tt.wantCount {
				t.Errorf("SearchByText() returned %d entries, want %d", len(results), tt.wantCount)
			}
		})
	}
}

func TestAuditQueryService_GetEntriesByCategory(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add entries with categories
	sagaEntry := &AuditEntry{
		Level:        AuditLevelInfo,
		Action:       AuditActionSagaStarted,
		ResourceType: "saga",
		ResourceID:   "saga-1",
		Message:      "Saga started",
	}
	AddTagsToEntry(sagaEntry, CategorySaga, nil)
	_ = auditLogger.Log(ctx, sagaEntry)

	authEntry := &AuditEntry{
		Level:        AuditLevelInfo,
		Action:       AuditActionAuthSuccess,
		ResourceType: "auth",
		ResourceID:   "user-1",
		Message:      "Auth success",
	}
	AddTagsToEntry(authEntry, CategoryAuth, nil)
	_ = auditLogger.Log(ctx, authEntry)

	// Get saga entries
	sagaEntries, err := service.GetEntriesByCategory(ctx, CategorySaga, nil)
	if err != nil {
		t.Errorf("GetEntriesByCategory() error = %v", err)
	}
	if len(sagaEntries) != 1 {
		t.Errorf("GetEntriesByCategory() returned %d entries, want 1", len(sagaEntries))
	}

	// Get auth entries
	authEntries, err := service.GetEntriesByCategory(ctx, CategoryAuth, nil)
	if err != nil {
		t.Errorf("GetEntriesByCategory() error = %v", err)
	}
	if len(authEntries) != 1 {
		t.Errorf("GetEntriesByCategory() returned %d entries, want 1", len(authEntries))
	}
}

func TestAuditQueryService_GetEntriesByTag(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add entries with tags
	criticalEntry := &AuditEntry{
		Level:   AuditLevelError,
		Action:  AuditActionSagaFailed,
		Message: "Critical failure",
	}
	AddTagsToEntry(criticalEntry, CategorySaga, []AuditTag{TagCritical})
	_ = auditLogger.Log(ctx, criticalEntry)

	sensitiveEntry := &AuditEntry{
		Level:   AuditLevelInfo,
		Action:  AuditActionAuthSuccess,
		Message: "Sensitive operation",
	}
	AddTagsToEntry(sensitiveEntry, CategoryAuth, []AuditTag{TagSensitive})
	_ = auditLogger.Log(ctx, sensitiveEntry)

	// Get critical entries
	criticalEntries, err := service.GetEntriesByTag(ctx, TagCritical, nil)
	if err != nil {
		t.Errorf("GetEntriesByTag() error = %v", err)
	}
	if len(criticalEntries) != 1 {
		t.Errorf("GetEntriesByTag() returned %d entries, want 1", len(criticalEntries))
	}

	// Get sensitive entries
	sensitiveEntries, err := service.GetEntriesByTag(ctx, TagSensitive, nil)
	if err != nil {
		t.Errorf("GetEntriesByTag() error = %v", err)
	}
	if len(sensitiveEntries) != 1 {
		t.Errorf("GetEntriesByTag() returned %d entries, want 1", len(sensitiveEntries))
	}
}

func TestAuditQueryService_GetRecentEntries(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add multiple entries
	for i := 0; i < 10; i++ {
		_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Test", nil)
		time.Sleep(1 * time.Millisecond)
	}

	// Get recent 5 entries
	entries, err := service.GetRecentEntries(ctx, 5)
	if err != nil {
		t.Errorf("GetRecentEntries() error = %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("GetRecentEntries() returned %d entries, want 5", len(entries))
	}
}

func TestAuditQueryService_GetEntriesSince(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	now := time.Now()
	since := now.Add(-5 * time.Minute)

	// Add entries
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Test 1", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-2", "Test 2", nil)

	entries, err := service.GetEntriesSince(ctx, since)
	if err != nil {
		t.Errorf("GetEntriesSince() error = %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("GetEntriesSince() returned %d entries, want 2", len(entries))
	}
}

func TestAuditQueryService_GetEntriesForUser(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add entries for different users
	entry1 := &AuditEntry{
		Level:   AuditLevelInfo,
		Action:  AuditActionSagaStarted,
		UserID:  "user-1",
		Message: "User 1 action",
	}
	_ = auditLogger.Log(ctx, entry1)

	entry2 := &AuditEntry{
		Level:   AuditLevelInfo,
		Action:  AuditActionSagaStarted,
		UserID:  "user-2",
		Message: "User 2 action",
	}
	_ = auditLogger.Log(ctx, entry2)

	// Get entries for user-1
	user1Entries, err := service.GetEntriesForUser(ctx, "user-1", nil)
	if err != nil {
		t.Errorf("GetEntriesForUser() error = %v", err)
	}
	if len(user1Entries) != 1 {
		t.Errorf("GetEntriesForUser() returned %d entries, want 1", len(user1Entries))
	}
	if user1Entries[0].UserID != "user-1" {
		t.Errorf("GetEntriesForUser() returned entry with UserID %v, want user-1", user1Entries[0].UserID)
	}
}

func TestAuditQueryService_GetEntriesForResource(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add entries for different resources
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Test 1", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-2", "Test 2", nil)

	// Get entries for saga-1
	entries, err := service.GetEntriesForResource(ctx, "saga", "saga-1", nil)
	if err != nil {
		t.Errorf("GetEntriesForResource() error = %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("GetEntriesForResource() returned %d entries, want 1", len(entries))
	}
	if entries[0].ResourceID != "saga-1" {
		t.Errorf("GetEntriesForResource() returned entry with ResourceID %v, want saga-1", entries[0].ResourceID)
	}
}

func TestAuditQueryService_GetErrorEntries(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add entries with different levels
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Info", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelError, AuditActionSagaFailed, "saga", "saga-2", "Error", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelCritical, AuditActionSagaFailed, "saga", "saga-3", "Critical", nil)

	// Get error entries
	errorEntries, err := service.GetErrorEntries(ctx, nil)
	if err != nil {
		t.Errorf("GetErrorEntries() error = %v", err)
	}
	if len(errorEntries) != 2 {
		t.Errorf("GetErrorEntries() returned %d entries, want 2", len(errorEntries))
	}
}

func TestAuditQueryService_GetCriticalEntries(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add entries with different levels
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Info", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelError, AuditActionSagaFailed, "saga", "saga-2", "Error", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelCritical, AuditActionSagaFailed, "saga", "saga-3", "Critical", nil)

	// Get critical entries
	criticalEntries, err := service.GetCriticalEntries(ctx, nil)
	if err != nil {
		t.Errorf("GetCriticalEntries() error = %v", err)
	}
	if len(criticalEntries) != 1 {
		t.Errorf("GetCriticalEntries() returned %d entries, want 1", len(criticalEntries))
	}
}

func TestAuditQueryService_GetAuthEvents(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add auth and non-auth events
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionAuthSuccess, "auth", "user-1", "Auth success", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelWarning, AuditActionAuthFailure, "auth", "user-2", "Auth failure", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Saga started", nil)

	// Get auth events
	authEvents, err := service.GetAuthEvents(ctx, nil)
	if err != nil {
		t.Errorf("GetAuthEvents() error = %v", err)
	}
	if len(authEvents) != 2 {
		t.Errorf("GetAuthEvents() returned %d entries, want 2", len(authEvents))
	}
}

func TestAuditQueryService_GetFailedAuthAttempts(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add auth events
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionAuthSuccess, "auth", "user-1", "Auth success", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelWarning, AuditActionAuthFailure, "auth", "user-2", "Auth failure", nil)
	_ = auditLogger.LogAction(ctx, AuditLevelWarning, AuditActionAccessDenied, "auth", "user-3", "Access denied", nil)

	// Get failed auth attempts
	failedAttempts, err := service.GetFailedAuthAttempts(ctx, nil)
	if err != nil {
		t.Errorf("GetFailedAuthAttempts() error = %v", err)
	}
	if len(failedAttempts) != 2 {
		t.Errorf("GetFailedAuthAttempts() returned %d entries, want 2", len(failedAttempts))
	}
}

func TestAuditQueryService_GetStatistics(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	// Add various entries
	entry1 := &AuditEntry{
		Level:        AuditLevelInfo,
		Action:       AuditActionSagaStarted,
		ResourceType: "saga",
		ResourceID:   "saga-1",
		UserID:       "user-1",
		Message:      "Test 1",
	}
	AddTagsToEntry(entry1, CategorySaga, []AuditTag{TagCritical})
	_ = auditLogger.Log(ctx, entry1)

	entry2 := &AuditEntry{
		Level:        AuditLevelError,
		Action:       AuditActionSagaFailed,
		ResourceType: "saga",
		ResourceID:   "saga-2",
		UserID:       "user-2",
		Message:      "Test 2",
	}
	AddTagsToEntry(entry2, CategorySaga, []AuditTag{TagCritical})
	_ = auditLogger.Log(ctx, entry2)

	entry3 := &AuditEntry{
		Level:        AuditLevelInfo,
		Action:       AuditActionAuthSuccess,
		ResourceType: "auth",
		ResourceID:   "user-1",
		UserID:       "user-1",
		Message:      "Test 3",
	}
	AddTagsToEntry(entry3, CategoryAuth, []AuditTag{TagSensitive})
	_ = auditLogger.Log(ctx, entry3)

	// Get statistics
	stats, err := service.GetStatistics(ctx, nil)
	if err != nil {
		t.Errorf("GetStatistics() error = %v", err)
	}

	if stats.TotalEntries != 3 {
		t.Errorf("TotalEntries = %d, want 3", stats.TotalEntries)
	}
	if stats.UniqueUsers != 2 {
		t.Errorf("UniqueUsers = %d, want 2", stats.UniqueUsers)
	}
	if stats.UniqueResources != 3 {
		t.Errorf("UniqueResources = %d, want 3", stats.UniqueResources)
	}
	if stats.ErrorRate == 0 {
		t.Error("ErrorRate is zero, expected non-zero")
	}
	if len(stats.EntriesByLevel) != 2 {
		t.Errorf("EntriesByLevel count = %d, want 2", len(stats.EntriesByLevel))
	}
	if len(stats.EntriesByCategory) != 2 {
		t.Errorf("EntriesByCategory count = %d, want 2", len(stats.EntriesByCategory))
	}
}

func TestAuditQueryService_GetTimeline(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryAuditStorage()
	auditLogger := createTestAuditLoggerWithStorage(t, storage)
	service := NewAuditQueryService(auditLogger)

	now := time.Now()
	startTime := now.Add(-2 * time.Hour)
	endTime := now

	// Add entries at different times
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaStarted, "saga", "saga-1", "Test 1", nil)
	time.Sleep(10 * time.Millisecond)
	_ = auditLogger.LogAction(ctx, AuditLevelInfo, AuditActionSagaCompleted, "saga", "saga-1", "Test 2", nil)

	// Get timeline with 1-hour intervals
	timeline, err := service.GetTimeline(ctx, startTime, endTime, time.Hour, nil)
	if err != nil {
		t.Errorf("GetTimeline() error = %v", err)
	}

	if timeline.Interval != time.Hour {
		t.Errorf("Timeline interval = %v, want 1 hour", timeline.Interval)
	}
	if len(timeline.Buckets) != 2 {
		t.Errorf("Timeline buckets count = %d, want 2", len(timeline.Buckets))
	}
}

// Helper functions

func createTestAuditLoggerWithStorage(t *testing.T, storage AuditStorage) AuditLogger {
	t.Helper()
	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage: storage,
		Source:  "test",
	})
	if err != nil {
		t.Fatalf("Failed to create test audit logger: %v", err)
	}
	return logger
}

func createTestEntries(t *testing.T, ctx context.Context, auditLogger AuditLogger) []*AuditEntry {
	t.Helper()

	entries := []*AuditEntry{
		{
			Level:        AuditLevelInfo,
			Action:       AuditActionSagaStarted,
			ResourceType: "saga",
			ResourceID:   "saga-1",
			UserID:       "user-1",
			Message:      "Saga started",
		},
		{
			Level:        AuditLevelInfo,
			Action:       AuditActionSagaCompleted,
			ResourceType: "saga",
			ResourceID:   "saga-1",
			UserID:       "user-1",
			Message:      "Saga completed",
		},
		{
			Level:        AuditLevelError,
			Action:       AuditActionSagaFailed,
			ResourceType: "saga",
			ResourceID:   "saga-2",
			UserID:       "user-2",
			Message:      "Saga failed",
		},
	}

	for _, entry := range entries {
		if err := auditLogger.Log(ctx, entry); err != nil {
			t.Fatalf("Failed to log entry: %v", err)
		}
	}

	return entries
}

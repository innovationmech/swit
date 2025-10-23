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

// Package security provides audit query and filtering functionality.
package security

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// QueryBuilder helps build complex audit queries
type QueryBuilder struct {
	filter *AuditFilter
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		filter: &AuditFilter{},
	}
}

// WithTimeRange sets the time range filter
func (qb *QueryBuilder) WithTimeRange(startTime, endTime time.Time) *QueryBuilder {
	qb.filter.StartTime = &startTime
	qb.filter.EndTime = &endTime
	return qb
}

// WithStartTime sets the start time filter
func (qb *QueryBuilder) WithStartTime(startTime time.Time) *QueryBuilder {
	qb.filter.StartTime = &startTime
	return qb
}

// WithEndTime sets the end time filter
func (qb *QueryBuilder) WithEndTime(endTime time.Time) *QueryBuilder {
	qb.filter.EndTime = &endTime
	return qb
}

// WithLevels sets the level filter
func (qb *QueryBuilder) WithLevels(levels ...AuditLevel) *QueryBuilder {
	qb.filter.Levels = levels
	return qb
}

// WithActions sets the action filter
func (qb *QueryBuilder) WithActions(actions ...AuditAction) *QueryBuilder {
	qb.filter.Actions = actions
	return qb
}

// WithResource sets the resource filter
func (qb *QueryBuilder) WithResource(resourceType, resourceID string) *QueryBuilder {
	qb.filter.ResourceType = resourceType
	qb.filter.ResourceID = resourceID
	return qb
}

// WithResourceType sets the resource type filter
func (qb *QueryBuilder) WithResourceType(resourceType string) *QueryBuilder {
	qb.filter.ResourceType = resourceType
	return qb
}

// WithResourceID sets the resource ID filter
func (qb *QueryBuilder) WithResourceID(resourceID string) *QueryBuilder {
	qb.filter.ResourceID = resourceID
	return qb
}

// WithUserID sets the user ID filter
func (qb *QueryBuilder) WithUserID(userID string) *QueryBuilder {
	qb.filter.UserID = userID
	return qb
}

// WithSource sets the source filter
func (qb *QueryBuilder) WithSource(source string) *QueryBuilder {
	qb.filter.Source = source
	return qb
}

// WithPagination sets pagination parameters
func (qb *QueryBuilder) WithPagination(limit, offset int) *QueryBuilder {
	qb.filter.Limit = limit
	qb.filter.Offset = offset
	return qb
}

// WithLimit sets the limit
func (qb *QueryBuilder) WithLimit(limit int) *QueryBuilder {
	qb.filter.Limit = limit
	return qb
}

// WithOffset sets the offset
func (qb *QueryBuilder) WithOffset(offset int) *QueryBuilder {
	qb.filter.Offset = offset
	return qb
}

// WithSorting sets sorting parameters
func (qb *QueryBuilder) WithSorting(sortBy, sortOrder string) *QueryBuilder {
	qb.filter.SortBy = sortBy
	qb.filter.SortOrder = sortOrder
	return qb
}

// Build builds and returns the audit filter
func (qb *QueryBuilder) Build() *AuditFilter {
	return qb.filter
}

// AuditQueryService provides high-level audit query operations
type AuditQueryService struct {
	auditLogger AuditLogger
}

// NewAuditQueryService creates a new audit query service
func NewAuditQueryService(auditLogger AuditLogger) *AuditQueryService {
	return &AuditQueryService{
		auditLogger: auditLogger,
	}
}

// Query executes a query using the provided filter
func (s *AuditQueryService) Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error) {
	return s.auditLogger.Query(ctx, filter)
}

// Count returns the count of entries matching the filter
func (s *AuditQueryService) Count(ctx context.Context, filter *AuditFilter) (int64, error) {
	return s.auditLogger.Count(ctx, filter)
}

// SearchByText searches audit entries by text in message or details
// Note: This is an in-memory search and may be slow for large datasets
func (s *AuditQueryService) SearchByText(ctx context.Context, searchText string, filter *AuditFilter) ([]*AuditEntry, error) {
	// Get all entries matching the filter
	entries, err := s.auditLogger.Query(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Filter by text search
	var results []*AuditEntry
	searchTextLower := strings.ToLower(searchText)

	for _, entry := range entries {
		// Search in message
		if strings.Contains(strings.ToLower(entry.Message), searchTextLower) {
			results = append(results, entry)
			continue
		}

		// Search in error
		if entry.Error != "" && strings.Contains(strings.ToLower(entry.Error), searchTextLower) {
			results = append(results, entry)
			continue
		}

		// Search in resource type and ID
		if strings.Contains(strings.ToLower(entry.ResourceType), searchTextLower) {
			results = append(results, entry)
			continue
		}

		if strings.Contains(strings.ToLower(entry.ResourceID), searchTextLower) {
			results = append(results, entry)
			continue
		}
	}

	return results, nil
}

// GetEntriesByCategory retrieves entries by category
func (s *AuditQueryService) GetEntriesByCategory(ctx context.Context, category AuditCategory, filter *AuditFilter) ([]*AuditEntry, error) {
	entries, err := s.auditLogger.Query(ctx, filter)
	if err != nil {
		return nil, err
	}

	var results []*AuditEntry
	for _, entry := range entries {
		if GetCategoryFromEntry(entry) == category {
			results = append(results, entry)
		}
	}

	return results, nil
}

// GetEntriesByTag retrieves entries by tag
func (s *AuditQueryService) GetEntriesByTag(ctx context.Context, tag AuditTag, filter *AuditFilter) ([]*AuditEntry, error) {
	entries, err := s.auditLogger.Query(ctx, filter)
	if err != nil {
		return nil, err
	}

	var results []*AuditEntry
	for _, entry := range entries {
		tags := GetTagsFromEntry(entry)
		for _, t := range tags {
			if t == tag {
				results = append(results, entry)
				break
			}
		}
	}

	return results, nil
}

// GetRecentEntries retrieves the most recent audit entries
func (s *AuditQueryService) GetRecentEntries(ctx context.Context, limit int) ([]*AuditEntry, error) {
	filter := NewQueryBuilder().
		WithSorting("timestamp", "desc").
		WithLimit(limit).
		Build()

	return s.auditLogger.Query(ctx, filter)
}

// GetEntriesSince retrieves entries since a specific time
func (s *AuditQueryService) GetEntriesSince(ctx context.Context, since time.Time) ([]*AuditEntry, error) {
	filter := NewQueryBuilder().
		WithStartTime(since).
		WithSorting("timestamp", "desc").
		Build()

	return s.auditLogger.Query(ctx, filter)
}

// GetEntriesForUser retrieves all entries for a specific user
func (s *AuditQueryService) GetEntriesForUser(ctx context.Context, userID string, filter *AuditFilter) ([]*AuditEntry, error) {
	if filter == nil {
		filter = &AuditFilter{}
	}
	filter.UserID = userID

	return s.auditLogger.Query(ctx, filter)
}

// GetEntriesForResource retrieves all entries for a specific resource
func (s *AuditQueryService) GetEntriesForResource(ctx context.Context, resourceType, resourceID string, filter *AuditFilter) ([]*AuditEntry, error) {
	if filter == nil {
		filter = &AuditFilter{}
	}
	filter.ResourceType = resourceType
	filter.ResourceID = resourceID

	return s.auditLogger.Query(ctx, filter)
}

// GetErrorEntries retrieves all entries with errors
func (s *AuditQueryService) GetErrorEntries(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error) {
	if filter == nil {
		filter = &AuditFilter{}
	}
	filter.Levels = []AuditLevel{AuditLevelError, AuditLevelCritical}

	return s.auditLogger.Query(ctx, filter)
}

// GetCriticalEntries retrieves all critical entries
func (s *AuditQueryService) GetCriticalEntries(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error) {
	if filter == nil {
		filter = &AuditFilter{}
	}
	filter.Levels = []AuditLevel{AuditLevelCritical}

	return s.auditLogger.Query(ctx, filter)
}

// GetAuthEvents retrieves authentication/authorization events
func (s *AuditQueryService) GetAuthEvents(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error) {
	if filter == nil {
		filter = &AuditFilter{}
	}
	filter.Actions = []AuditAction{
		AuditActionAuthSuccess,
		AuditActionAuthFailure,
		AuditActionAuthExpired,
		AuditActionAuthRevoked,
		AuditActionAccessGranted,
		AuditActionAccessDenied,
	}

	return s.auditLogger.Query(ctx, filter)
}

// GetFailedAuthAttempts retrieves failed authentication attempts
func (s *AuditQueryService) GetFailedAuthAttempts(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error) {
	if filter == nil {
		filter = &AuditFilter{}
	}
	filter.Actions = []AuditAction{AuditActionAuthFailure, AuditActionAccessDenied}

	return s.auditLogger.Query(ctx, filter)
}

// AuditStatistics provides statistics about audit entries
type AuditStatistics struct {
	TotalEntries      int64                   `json:"total_entries"`
	EntriesByLevel    map[AuditLevel]int64    `json:"entries_by_level"`
	EntriesByAction   map[AuditAction]int64   `json:"entries_by_action"`
	EntriesByCategory map[AuditCategory]int64 `json:"entries_by_category"`
	EntriesByTag      map[AuditTag]int64      `json:"entries_by_tag"`
	UniqueUsers       int64                   `json:"unique_users"`
	UniqueResources   int64                   `json:"unique_resources"`
	ErrorRate         float64                 `json:"error_rate"`
	StartTime         time.Time               `json:"start_time"`
	EndTime           time.Time               `json:"end_time"`
}

// GetStatistics calculates statistics for audit entries
func (s *AuditQueryService) GetStatistics(ctx context.Context, filter *AuditFilter) (*AuditStatistics, error) {
	// Get all entries matching the filter
	entries, err := s.auditLogger.Query(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}

	stats := &AuditStatistics{
		EntriesByLevel:    make(map[AuditLevel]int64),
		EntriesByAction:   make(map[AuditAction]int64),
		EntriesByCategory: make(map[AuditCategory]int64),
		EntriesByTag:      make(map[AuditTag]int64),
	}

	uniqueUsers := make(map[string]bool)
	uniqueResources := make(map[string]bool)
	errorCount := int64(0)

	for _, entry := range entries {
		stats.TotalEntries++

		// Count by level
		stats.EntriesByLevel[entry.Level]++

		// Count by action
		stats.EntriesByAction[entry.Action]++

		// Count by category
		category := GetCategoryFromEntry(entry)
		if category != "" {
			stats.EntriesByCategory[category]++
		}

		// Count by tag
		tags := GetTagsFromEntry(entry)
		for _, tag := range tags {
			stats.EntriesByTag[tag]++
		}

		// Track unique users
		if entry.UserID != "" {
			uniqueUsers[entry.UserID] = true
		}

		// Track unique resources
		if entry.ResourceType != "" && entry.ResourceID != "" {
			resourceKey := fmt.Sprintf("%s:%s", entry.ResourceType, entry.ResourceID)
			uniqueResources[resourceKey] = true
		}

		// Count errors
		if entry.Level == AuditLevelError || entry.Level == AuditLevelCritical {
			errorCount++
		}

		// Track time range
		if stats.StartTime.IsZero() || entry.Timestamp.Before(stats.StartTime) {
			stats.StartTime = entry.Timestamp
		}
		if stats.EndTime.IsZero() || entry.Timestamp.After(stats.EndTime) {
			stats.EndTime = entry.Timestamp
		}
	}

	stats.UniqueUsers = int64(len(uniqueUsers))
	stats.UniqueResources = int64(len(uniqueResources))

	if stats.TotalEntries > 0 {
		stats.ErrorRate = float64(errorCount) / float64(stats.TotalEntries)
	}

	return stats, nil
}

// AuditTimeline represents a timeline of audit events
type AuditTimeline struct {
	Interval  time.Duration          `json:"interval"`
	Buckets   []*AuditTimelineBucket `json:"buckets"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
}

// AuditTimelineBucket represents a time bucket in the timeline
type AuditTimelineBucket struct {
	StartTime time.Time             `json:"start_time"`
	EndTime   time.Time             `json:"end_time"`
	Count     int64                 `json:"count"`
	ByLevel   map[AuditLevel]int64  `json:"by_level"`
	ByAction  map[AuditAction]int64 `json:"by_action"`
}

// GetTimeline generates a timeline of audit events
func (s *AuditQueryService) GetTimeline(ctx context.Context, startTime, endTime time.Time, interval time.Duration, filter *AuditFilter) (*AuditTimeline, error) {
	// Update filter with time range
	if filter == nil {
		filter = &AuditFilter{}
	}
	filter.StartTime = &startTime
	filter.EndTime = &endTime

	// Get all entries in the time range
	entries, err := s.auditLogger.Query(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}

	// Create timeline buckets
	timeline := &AuditTimeline{
		Interval:  interval,
		StartTime: startTime,
		EndTime:   endTime,
		Buckets:   make([]*AuditTimelineBucket, 0),
	}

	// Generate buckets
	for t := startTime; t.Before(endTime); t = t.Add(interval) {
		bucket := &AuditTimelineBucket{
			StartTime: t,
			EndTime:   t.Add(interval),
			ByLevel:   make(map[AuditLevel]int64),
			ByAction:  make(map[AuditAction]int64),
		}
		timeline.Buckets = append(timeline.Buckets, bucket)
	}

	// Fill buckets with entries
	for _, entry := range entries {
		// Find the appropriate bucket
		for _, bucket := range timeline.Buckets {
			if (entry.Timestamp.Equal(bucket.StartTime) || entry.Timestamp.After(bucket.StartTime)) &&
				entry.Timestamp.Before(bucket.EndTime) {
				bucket.Count++
				bucket.ByLevel[entry.Level]++
				bucket.ByAction[entry.Action]++
				break
			}
		}
	}

	return timeline, nil
}

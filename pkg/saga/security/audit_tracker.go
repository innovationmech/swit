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

// Package security provides audit tracking functionality for operation tracing.
package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// AuditCategory represents the category of an audit event
type AuditCategory string

const (
	// CategorySaga represents saga-related events
	CategorySaga AuditCategory = "saga"
	// CategoryAuth represents authentication/authorization events
	CategoryAuth AuditCategory = "auth"
	// CategoryData represents data access events
	CategoryData AuditCategory = "data"
	// CategoryConfig represents configuration change events
	CategoryConfig AuditCategory = "config"
	// CategorySecurity represents security-related events
	CategorySecurity AuditCategory = "security"
	// CategorySystem represents system-level events
	CategorySystem AuditCategory = "system"
)

// AuditTag represents a tag for categorizing audit events
type AuditTag string

const (
	// TagSensitive marks events involving sensitive data
	TagSensitive AuditTag = "sensitive"
	// TagCompliance marks events relevant for compliance
	TagCompliance AuditTag = "compliance"
	// TagCritical marks critical operations
	TagCritical AuditTag = "critical"
	// TagAdministrative marks administrative operations
	TagAdministrative AuditTag = "administrative"
	// TagAutomated marks automated operations
	TagAutomated AuditTag = "automated"
	// TagManual marks manual operations
	TagManual AuditTag = "manual"
	// TagRetry marks retry attempts
	TagRetry AuditTag = "retry"
	// TagRollback marks rollback operations
	TagRollback AuditTag = "rollback"
)

// OperationTrace represents a trace of related operations
type OperationTrace struct {
	// Unique identifier for the trace
	TraceID string `json:"trace_id"`

	// Parent trace ID for nested operations
	ParentTraceID string `json:"parent_trace_id,omitempty"`

	// Operation name
	Operation string `json:"operation"`

	// Category of the operation
	Category AuditCategory `json:"category"`

	// Tags associated with the operation
	Tags []AuditTag `json:"tags,omitempty"`

	// Start time of the operation
	StartTime time.Time `json:"start_time"`

	// End time of the operation
	EndTime time.Time `json:"end_time,omitempty"`

	// Duration of the operation
	Duration time.Duration `json:"duration,omitempty"`

	// Status of the operation (success, failed, in_progress)
	Status string `json:"status"`

	// User who initiated the operation
	UserID string `json:"user_id,omitempty"`

	// Resource being operated on
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`

	// Steps in the operation
	Steps []*OperationStep `json:"steps,omitempty"`

	// Related audit entry IDs
	AuditEntryIDs []string `json:"audit_entry_ids,omitempty"`

	// Additional context
	Context map[string]interface{} `json:"context,omitempty"`

	// Error information if operation failed
	Error string `json:"error,omitempty"`

	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`
}

// OperationStep represents a single step in an operation trace
type OperationStep struct {
	// Step name
	Name string `json:"name"`

	// Step number in the sequence
	Sequence int `json:"sequence"`

	// Start time
	StartTime time.Time `json:"start_time"`

	// End time
	EndTime time.Time `json:"end_time,omitempty"`

	// Duration
	Duration time.Duration `json:"duration,omitempty"`

	// Status (success, failed, in_progress, skipped)
	Status string `json:"status"`

	// Associated audit entry ID
	AuditEntryID string `json:"audit_entry_id,omitempty"`

	// Error information if step failed
	Error string `json:"error,omitempty"`

	// Details about the step
	Details map[string]interface{} `json:"details,omitempty"`
}

// AuditTracker tracks operation traces and links them with audit entries
type AuditTracker interface {
	// StartTrace starts a new operation trace
	StartTrace(ctx context.Context, operation string, category AuditCategory, tags []AuditTag) (*OperationTrace, error)

	// StartChildTrace starts a child operation trace
	StartChildTrace(ctx context.Context, parentTraceID string, operation string, category AuditCategory, tags []AuditTag) (*OperationTrace, error)

	// AddStep adds a step to an operation trace
	AddStep(ctx context.Context, traceID string, stepName string) (*OperationStep, error)

	// CompleteStep marks a step as completed
	CompleteStep(ctx context.Context, traceID string, stepSequence int, auditEntryID string) error

	// FailStep marks a step as failed
	FailStep(ctx context.Context, traceID string, stepSequence int, err error, auditEntryID string) error

	// CompleteTrace marks a trace as completed
	CompleteTrace(ctx context.Context, traceID string) error

	// FailTrace marks a trace as failed
	FailTrace(ctx context.Context, traceID string, err error) error

	// GetTrace retrieves an operation trace
	GetTrace(ctx context.Context, traceID string) (*OperationTrace, error)

	// GetTracesByCategory retrieves all traces for a category
	GetTracesByCategory(ctx context.Context, category AuditCategory) ([]*OperationTrace, error)

	// GetTracesByTag retrieves all traces with a specific tag
	GetTracesByTag(ctx context.Context, tag AuditTag) ([]*OperationTrace, error)

	// GetTracesByTimeRange retrieves traces within a time range
	GetTracesByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*OperationTrace, error)

	// LinkAuditEntry links an audit entry to a trace
	LinkAuditEntry(ctx context.Context, traceID string, auditEntryID string) error

	// Close closes the tracker and releases resources
	Close() error
}

// DefaultAuditTracker implements the AuditTracker interface
type DefaultAuditTracker struct {
	traces      map[string]*OperationTrace
	auditLogger AuditLogger
	mu          sync.RWMutex
	closed      bool
	logger      *zap.Logger
}

// AuditTrackerConfig configures the audit tracker
type AuditTrackerConfig struct {
	AuditLogger AuditLogger
}

// NewAuditTracker creates a new audit tracker
func NewAuditTracker(config *AuditTrackerConfig) (*DefaultAuditTracker, error) {
	if config.AuditLogger == nil {
		return nil, fmt.Errorf("audit logger is required")
	}

	tracker := &DefaultAuditTracker{
		traces:      make(map[string]*OperationTrace),
		auditLogger: config.AuditLogger,
		logger:      logger.Logger,
	}

	// Initialize logger if not set
	if tracker.logger == nil {
		tracker.logger = zap.NewNop()
	}

	return tracker, nil
}

// StartTrace starts a new operation trace
func (t *DefaultAuditTracker) StartTrace(ctx context.Context, operation string, category AuditCategory, tags []AuditTag) (*OperationTrace, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil, fmt.Errorf("tracker is closed")
	}

	// Generate trace ID
	traceID := fmt.Sprintf("trace-%d", time.Now().UnixNano())

	// Extract user ID from context if available
	userID := ""
	if authCtx, ok := AuthFromContext(ctx); ok && authCtx.Credentials != nil {
		userID = authCtx.Credentials.UserID
	}

	trace := &OperationTrace{
		TraceID:       traceID,
		Operation:     operation,
		Category:      category,
		Tags:          tags,
		StartTime:     time.Now(),
		Status:        "in_progress",
		UserID:        userID,
		Steps:         make([]*OperationStep, 0),
		AuditEntryIDs: make([]string, 0),
		Context:       make(map[string]interface{}),
		Metadata:      make(map[string]string),
	}

	t.traces[traceID] = trace

	// Log the trace start
	t.logger.Debug("Operation trace started",
		zap.String("trace_id", traceID),
		zap.String("operation", operation),
		zap.String("category", string(category)))

	// Create an audit entry for trace start
	err := t.auditLogger.LogAction(ctx, AuditLevelInfo, AuditAction("trace.started"),
		"trace", traceID, fmt.Sprintf("Operation trace started: %s", operation),
		map[string]interface{}{
			"operation": operation,
			"category":  string(category),
			"tags":      tags,
		})
	if err != nil {
		t.logger.Warn("Failed to log trace start", zap.Error(err))
	}

	return trace, nil
}

// StartChildTrace starts a child operation trace
func (t *DefaultAuditTracker) StartChildTrace(ctx context.Context, parentTraceID string, operation string, category AuditCategory, tags []AuditTag) (*OperationTrace, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil, fmt.Errorf("tracker is closed")
	}

	// Verify parent trace exists
	if _, exists := t.traces[parentTraceID]; !exists {
		return nil, fmt.Errorf("parent trace not found: %s", parentTraceID)
	}

	// Generate trace ID
	traceID := fmt.Sprintf("trace-%d", time.Now().UnixNano())

	// Extract user ID from context if available
	userID := ""
	if authCtx, ok := AuthFromContext(ctx); ok && authCtx.Credentials != nil {
		userID = authCtx.Credentials.UserID
	}

	trace := &OperationTrace{
		TraceID:       traceID,
		ParentTraceID: parentTraceID,
		Operation:     operation,
		Category:      category,
		Tags:          tags,
		StartTime:     time.Now(),
		Status:        "in_progress",
		UserID:        userID,
		Steps:         make([]*OperationStep, 0),
		AuditEntryIDs: make([]string, 0),
		Context:       make(map[string]interface{}),
		Metadata:      make(map[string]string),
	}

	t.traces[traceID] = trace

	t.logger.Debug("Child operation trace started",
		zap.String("trace_id", traceID),
		zap.String("parent_trace_id", parentTraceID),
		zap.String("operation", operation))

	return trace, nil
}

// AddStep adds a step to an operation trace
func (t *DefaultAuditTracker) AddStep(ctx context.Context, traceID string, stepName string) (*OperationStep, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil, fmt.Errorf("tracker is closed")
	}

	trace, exists := t.traces[traceID]
	if !exists {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	step := &OperationStep{
		Name:      stepName,
		Sequence:  len(trace.Steps) + 1,
		StartTime: time.Now(),
		Status:    "in_progress",
		Details:   make(map[string]interface{}),
	}

	trace.Steps = append(trace.Steps, step)

	t.logger.Debug("Step added to trace",
		zap.String("trace_id", traceID),
		zap.String("step_name", stepName),
		zap.Int("sequence", step.Sequence))

	return step, nil
}

// CompleteStep marks a step as completed
func (t *DefaultAuditTracker) CompleteStep(ctx context.Context, traceID string, stepSequence int, auditEntryID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("tracker is closed")
	}

	trace, exists := t.traces[traceID]
	if !exists {
		return fmt.Errorf("trace not found: %s", traceID)
	}

	if stepSequence < 1 || stepSequence > len(trace.Steps) {
		return fmt.Errorf("invalid step sequence: %d", stepSequence)
	}

	step := trace.Steps[stepSequence-1]
	step.Status = "success"
	step.EndTime = time.Now()
	step.Duration = step.EndTime.Sub(step.StartTime)
	step.AuditEntryID = auditEntryID

	if auditEntryID != "" {
		trace.AuditEntryIDs = append(trace.AuditEntryIDs, auditEntryID)
	}

	t.logger.Debug("Step completed",
		zap.String("trace_id", traceID),
		zap.Int("sequence", stepSequence),
		zap.Duration("duration", step.Duration))

	return nil
}

// FailStep marks a step as failed
func (t *DefaultAuditTracker) FailStep(ctx context.Context, traceID string, stepSequence int, err error, auditEntryID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("tracker is closed")
	}

	trace, exists := t.traces[traceID]
	if !exists {
		return fmt.Errorf("trace not found: %s", traceID)
	}

	if stepSequence < 1 || stepSequence > len(trace.Steps) {
		return fmt.Errorf("invalid step sequence: %d", stepSequence)
	}

	step := trace.Steps[stepSequence-1]
	step.Status = "failed"
	step.EndTime = time.Now()
	step.Duration = step.EndTime.Sub(step.StartTime)
	step.Error = err.Error()
	step.AuditEntryID = auditEntryID

	if auditEntryID != "" {
		trace.AuditEntryIDs = append(trace.AuditEntryIDs, auditEntryID)
	}

	t.logger.Debug("Step failed",
		zap.String("trace_id", traceID),
		zap.Int("sequence", stepSequence),
		zap.Error(err))

	return nil
}

// CompleteTrace marks a trace as completed
func (t *DefaultAuditTracker) CompleteTrace(ctx context.Context, traceID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("tracker is closed")
	}

	trace, exists := t.traces[traceID]
	if !exists {
		return fmt.Errorf("trace not found: %s", traceID)
	}

	trace.Status = "success"
	trace.EndTime = time.Now()
	trace.Duration = trace.EndTime.Sub(trace.StartTime)

	t.logger.Debug("Operation trace completed",
		zap.String("trace_id", traceID),
		zap.Duration("duration", trace.Duration))

	// Create an audit entry for trace completion
	err := t.auditLogger.LogAction(ctx, AuditLevelInfo, AuditAction("trace.completed"),
		"trace", traceID, fmt.Sprintf("Operation trace completed: %s", trace.Operation),
		map[string]interface{}{
			"operation": trace.Operation,
			"duration":  trace.Duration.String(),
			"steps":     len(trace.Steps),
		})
	if err != nil {
		t.logger.Warn("Failed to log trace completion", zap.Error(err))
	}

	return nil
}

// FailTrace marks a trace as failed
func (t *DefaultAuditTracker) FailTrace(ctx context.Context, traceID string, err error) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("tracker is closed")
	}

	trace, exists := t.traces[traceID]
	if !exists {
		return fmt.Errorf("trace not found: %s", traceID)
	}

	trace.Status = "failed"
	trace.EndTime = time.Now()
	trace.Duration = trace.EndTime.Sub(trace.StartTime)
	trace.Error = err.Error()

	t.logger.Debug("Operation trace failed",
		zap.String("trace_id", traceID),
		zap.Error(err))

	// Create an audit entry for trace failure
	logErr := t.auditLogger.LogAction(ctx, AuditLevelError, AuditAction("trace.failed"),
		"trace", traceID, fmt.Sprintf("Operation trace failed: %s", trace.Operation),
		map[string]interface{}{
			"operation": trace.Operation,
			"duration":  trace.Duration.String(),
			"error":     err.Error(),
		})
	if logErr != nil {
		t.logger.Warn("Failed to log trace failure", zap.Error(logErr))
	}

	return nil
}

// GetTrace retrieves an operation trace
func (t *DefaultAuditTracker) GetTrace(ctx context.Context, traceID string) (*OperationTrace, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return nil, fmt.Errorf("tracker is closed")
	}

	trace, exists := t.traces[traceID]
	if !exists {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	// Return a deep copy to avoid external modifications
	return deepCopyTrace(trace), nil
}

// GetTracesByCategory retrieves all traces for a category
func (t *DefaultAuditTracker) GetTracesByCategory(ctx context.Context, category AuditCategory) ([]*OperationTrace, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return nil, fmt.Errorf("tracker is closed")
	}

	var results []*OperationTrace
	for _, trace := range t.traces {
		if trace.Category == category {
			results = append(results, deepCopyTrace(trace))
		}
	}

	return results, nil
}

// GetTracesByTag retrieves all traces with a specific tag
func (t *DefaultAuditTracker) GetTracesByTag(ctx context.Context, tag AuditTag) ([]*OperationTrace, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return nil, fmt.Errorf("tracker is closed")
	}

	var results []*OperationTrace
	for _, trace := range t.traces {
		for _, traceTag := range trace.Tags {
			if traceTag == tag {
				results = append(results, deepCopyTrace(trace))
				break
			}
		}
	}

	return results, nil
}

// GetTracesByTimeRange retrieves traces within a time range
func (t *DefaultAuditTracker) GetTracesByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*OperationTrace, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return nil, fmt.Errorf("tracker is closed")
	}

	var results []*OperationTrace
	for _, trace := range t.traces {
		// Check if trace started within the time range
		if trace.StartTime.Before(startTime) || trace.StartTime.After(endTime) {
			continue
		}
		results = append(results, deepCopyTrace(trace))
	}

	return results, nil
}

// LinkAuditEntry links an audit entry to a trace
func (t *DefaultAuditTracker) LinkAuditEntry(ctx context.Context, traceID string, auditEntryID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("tracker is closed")
	}

	trace, exists := t.traces[traceID]
	if !exists {
		return fmt.Errorf("trace not found: %s", traceID)
	}

	trace.AuditEntryIDs = append(trace.AuditEntryIDs, auditEntryID)

	t.logger.Debug("Audit entry linked to trace",
		zap.String("trace_id", traceID),
		zap.String("audit_entry_id", auditEntryID))

	return nil
}

// Close closes the tracker and releases resources
func (t *DefaultAuditTracker) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true
	t.traces = nil

	return nil
}

// AddTagsToEntry adds category and tags to an audit entry
func AddTagsToEntry(entry *AuditEntry, category AuditCategory, tags []AuditTag) {
	if entry.Metadata == nil {
		entry.Metadata = make(map[string]string)
	}

	entry.Metadata["category"] = string(category)

	if len(tags) > 0 {
		tagStr := ""
		for i, tag := range tags {
			if i > 0 {
				tagStr += ","
			}
			tagStr += string(tag)
		}
		entry.Metadata["tags"] = tagStr
	}
}

// GetCategoryFromEntry extracts the category from an audit entry
func GetCategoryFromEntry(entry *AuditEntry) AuditCategory {
	if entry.Metadata == nil {
		return ""
	}
	return AuditCategory(entry.Metadata["category"])
}

// GetTagsFromEntry extracts tags from an audit entry
func GetTagsFromEntry(entry *AuditEntry) []AuditTag {
	if entry.Metadata == nil {
		return nil
	}

	tagStr := entry.Metadata["tags"]
	if tagStr == "" {
		return nil
	}

	var tags []AuditTag
	for _, t := range splitString(tagStr, ",") {
		if t != "" {
			tags = append(tags, AuditTag(t))
		}
	}

	return tags
}

// splitString splits a string by delimiter
func splitString(s string, delimiter string) []string {
	if s == "" {
		return nil
	}

	var parts []string
	start := 0

	for i := 0; i < len(s); i++ {
		if i+len(delimiter) <= len(s) && s[i:i+len(delimiter)] == delimiter {
			parts = append(parts, s[start:i])
			start = i + len(delimiter)
			i += len(delimiter) - 1
		}
	}

	if start < len(s) {
		parts = append(parts, s[start:])
	}

	return parts
}

// deepCopyTrace creates a deep copy of an OperationTrace to prevent concurrent mutation
func deepCopyTrace(original *OperationTrace) *OperationTrace {
	if original == nil {
		return nil
	}

	// Deep copy Tags slice
	tagsCopy := make([]AuditTag, len(original.Tags))
	copy(tagsCopy, original.Tags)

	// Deep copy Steps slice
	stepsCopy := make([]*OperationStep, len(original.Steps))
	for i, step := range original.Steps {
		stepsCopy[i] = deepCopyStep(step)
	}

	// Deep copy AuditEntryIDs slice
	auditIDsCopy := make([]string, len(original.AuditEntryIDs))
	copy(auditIDsCopy, original.AuditEntryIDs)

	// Deep copy Context map
	contextCopy := make(map[string]interface{})
	for k, v := range original.Context {
		contextCopy[k] = v
	}

	// Deep copy Metadata map
	metadataCopy := make(map[string]string)
	for k, v := range original.Metadata {
		metadataCopy[k] = v
	}

	return &OperationTrace{
		TraceID:       original.TraceID,
		ParentTraceID: original.ParentTraceID,
		Operation:     original.Operation,
		Category:      original.Category,
		Tags:          tagsCopy,
		StartTime:     original.StartTime,
		EndTime:       original.EndTime,
		Duration:      original.Duration,
		Status:        original.Status,
		UserID:        original.UserID,
		ResourceType:  original.ResourceType,
		ResourceID:    original.ResourceID,
		Steps:         stepsCopy,
		AuditEntryIDs: auditIDsCopy,
		Context:       contextCopy,
		Error:         original.Error,
		Metadata:      metadataCopy,
	}
}

// deepCopyStep creates a deep copy of an OperationStep to prevent concurrent mutation
func deepCopyStep(original *OperationStep) *OperationStep {
	if original == nil {
		return nil
	}

	// Deep copy Details map
	detailsCopy := make(map[string]interface{})
	for k, v := range original.Details {
		detailsCopy[k] = v
	}

	return &OperationStep{
		Name:         original.Name,
		Sequence:     original.Sequence,
		StartTime:    original.StartTime,
		EndTime:      original.EndTime,
		Duration:     original.Duration,
		Status:       original.Status,
		AuditEntryID: original.AuditEntryID,
		Error:        original.Error,
		Details:      detailsCopy,
	}
}

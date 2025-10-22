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

// Package security provides audit logging functionality for Saga operations.
package security

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

// AuditLevel represents the severity level of an audit event
type AuditLevel string

const (
	// AuditLevelInfo represents informational audit events
	AuditLevelInfo AuditLevel = "info"
	// AuditLevelWarning represents warning audit events
	AuditLevelWarning AuditLevel = "warning"
	// AuditLevelError represents error audit events
	AuditLevelError AuditLevel = "error"
	// AuditLevelCritical represents critical audit events
	AuditLevelCritical AuditLevel = "critical"
)

// AuditAction represents the type of action being audited
type AuditAction string

const (
	// Saga lifecycle actions
	AuditActionSagaCreated   AuditAction = "saga.created"
	AuditActionSagaStarted   AuditAction = "saga.started"
	AuditActionSagaCompleted AuditAction = "saga.completed"
	AuditActionSagaFailed    AuditAction = "saga.failed"
	AuditActionSagaCancelled AuditAction = "saga.cancelled"
	AuditActionSagaTimedOut  AuditAction = "saga.timed_out"

	// State transition actions
	AuditActionStateChanged AuditAction = "saga.state.changed"

	// Step actions
	AuditActionStepStarted   AuditAction = "saga.step.started"
	AuditActionStepCompleted AuditAction = "saga.step.completed"
	AuditActionStepFailed    AuditAction = "saga.step.failed"

	// Compensation actions
	AuditActionCompensationStarted   AuditAction = "saga.compensation.started"
	AuditActionCompensationCompleted AuditAction = "saga.compensation.completed"
	AuditActionCompensationFailed    AuditAction = "saga.compensation.failed"

	// Authentication and authorization actions
	AuditActionAuthSuccess AuditAction = "auth.success"
	AuditActionAuthFailure AuditAction = "auth.failure"
	AuditActionAuthExpired AuditAction = "auth.expired"
	AuditActionAuthRevoked AuditAction = "auth.revoked"

	// Authorization actions
	AuditActionAccessGranted AuditAction = "access.granted"
	AuditActionAccessDenied  AuditAction = "access.denied"

	// Configuration changes
	AuditActionConfigChanged AuditAction = "config.changed"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	// Unique identifier for the audit entry
	ID string `json:"id"`

	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Level of the audit event
	Level AuditLevel `json:"level"`

	// Action that was performed
	Action AuditAction `json:"action"`

	// Resource being acted upon (e.g., saga ID, step ID)
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`

	// User information
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`

	// Source information
	Source    string `json:"source"`
	IPAddress string `json:"ip_address,omitempty"`

	// State information
	OldState string `json:"old_state,omitempty"`
	NewState string `json:"new_state,omitempty"`

	// Message describing the event
	Message string `json:"message"`

	// Additional details
	Details map[string]interface{} `json:"details,omitempty"`

	// Error information (if applicable)
	Error string `json:"error,omitempty"`

	// Distributed tracing information
	TraceID       string `json:"trace_id,omitempty"`
	SpanID        string `json:"span_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`

	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ToJSON converts the audit entry to JSON string
func (e *AuditEntry) ToJSON() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", fmt.Errorf("failed to marshal audit entry: %w", err)
	}
	return string(data), nil
}

// AuditFilter represents filtering criteria for querying audit logs
type AuditFilter struct {
	// Time range
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`

	// Filter by level
	Levels []AuditLevel `json:"levels,omitempty"`

	// Filter by action
	Actions []AuditAction `json:"actions,omitempty"`

	// Filter by resource
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`

	// Filter by user
	UserID string `json:"user_id,omitempty"`

	// Filter by source
	Source string `json:"source,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`    // e.g., "timestamp", "level"
	SortOrder string `json:"sort_order,omitempty"` // "asc" or "desc"
}

// Match determines if an audit entry matches the filter criteria
func (f *AuditFilter) Match(entry *AuditEntry) bool {
	// Nil filter matches everything
	if f == nil {
		return true
	}

	// Check time range
	if f.StartTime != nil && entry.Timestamp.Before(*f.StartTime) {
		return false
	}
	if f.EndTime != nil && entry.Timestamp.After(*f.EndTime) {
		return false
	}

	// Check level
	if len(f.Levels) > 0 {
		matched := false
		for _, level := range f.Levels {
			if entry.Level == level {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check action
	if len(f.Actions) > 0 {
		matched := false
		for _, action := range f.Actions {
			if entry.Action == action {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check resource
	if f.ResourceType != "" && entry.ResourceType != f.ResourceType {
		return false
	}
	if f.ResourceID != "" && entry.ResourceID != f.ResourceID {
		return false
	}

	// Check user
	if f.UserID != "" && entry.UserID != f.UserID {
		return false
	}

	// Check source
	if f.Source != "" && entry.Source != f.Source {
		return false
	}

	return true
}

// AuditStorage defines the interface for storing and retrieving audit logs
type AuditStorage interface {
	// Write writes an audit entry to storage
	Write(ctx context.Context, entry *AuditEntry) error

	// Query retrieves audit entries based on filter criteria
	Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error)

	// Count returns the total number of audit entries matching the filter
	Count(ctx context.Context, filter *AuditFilter) (int64, error)

	// Rotate performs log rotation (for file-based storage)
	Rotate(ctx context.Context) error

	// Cleanup removes audit entries older than the specified duration
	Cleanup(ctx context.Context, olderThan time.Duration) error

	// Close closes the storage and releases resources
	Close() error
}

// AuditLogger is the main interface for audit logging
type AuditLogger interface {
	// Log logs an audit entry
	Log(ctx context.Context, entry *AuditEntry) error

	// LogAction logs an action with automatic entry creation
	LogAction(ctx context.Context, level AuditLevel, action AuditAction, resourceType, resourceID, message string, details map[string]interface{}) error

	// LogStateChange logs a state change event
	LogStateChange(ctx context.Context, resourceType, resourceID string, oldState, newState saga.SagaState, userID string, details map[string]interface{}) error

	// LogAuthEvent logs an authentication or authorization event
	LogAuthEvent(ctx context.Context, action AuditAction, userID string, success bool, details map[string]interface{}) error

	// Query retrieves audit entries based on filter criteria
	Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error)

	// Count returns the total number of audit entries matching the filter
	Count(ctx context.Context, filter *AuditFilter) (int64, error)

	// Close closes the audit logger and releases resources
	Close() error
}

// DefaultAuditLogger implements the AuditLogger interface
type DefaultAuditLogger struct {
	storage  AuditStorage
	source   string
	logger   *zap.Logger
	mu       sync.RWMutex
	closed   bool
	enricher func(*AuditEntry) // Optional enricher function
}

// AuditLoggerConfig configures the audit logger
type AuditLoggerConfig struct {
	Storage  AuditStorage
	Source   string
	Enricher func(*AuditEntry) // Optional function to enrich entries with additional context
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config *AuditLoggerConfig) (*DefaultAuditLogger, error) {
	if config.Storage == nil {
		return nil, fmt.Errorf("storage is required")
	}

	if config.Source == "" {
		config.Source = "saga"
	}

	auditLogger := &DefaultAuditLogger{
		storage:  config.Storage,
		source:   config.Source,
		logger:   logger.Logger,
		enricher: config.Enricher,
	}

	// Initialize logger if not set
	if auditLogger.logger == nil {
		auditLogger.logger = zap.NewNop()
	}

	return auditLogger, nil
}

// Log logs an audit entry
func (l *DefaultAuditLogger) Log(ctx context.Context, entry *AuditEntry) error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.closed {
		return fmt.Errorf("audit logger is closed")
	}

	// Set default values
	if entry.ID == "" {
		entry.ID = generateAuditID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	if entry.Source == "" {
		entry.Source = l.source
	}

	// Extract auth context if available
	if authCtx, ok := AuthFromContext(ctx); ok {
		if entry.UserID == "" && authCtx.Credentials != nil {
			entry.UserID = authCtx.Credentials.UserID
		}
		if entry.TraceID == "" {
			entry.TraceID = authCtx.SagaID
		}
		if entry.CorrelationID == "" {
			entry.CorrelationID = authCtx.CorrelationID
		}
	}

	// Apply enricher if configured
	if l.enricher != nil {
		l.enricher(entry)
	}

	// Write to storage
	if err := l.storage.Write(ctx, entry); err != nil {
		l.logger.Error("Failed to write audit entry",
			zap.String("entry_id", entry.ID),
			zap.String("action", string(entry.Action)),
			zap.Error(err))
		return fmt.Errorf("failed to write audit entry: %w", err)
	}

	l.logger.Debug("Audit entry logged",
		zap.String("id", entry.ID),
		zap.String("action", string(entry.Action)),
		zap.String("resource", entry.ResourceID),
		zap.String("user", entry.UserID))

	return nil
}

// LogAction logs an action with automatic entry creation
func (l *DefaultAuditLogger) LogAction(ctx context.Context, level AuditLevel, action AuditAction, resourceType, resourceID, message string, details map[string]interface{}) error {
	entry := &AuditEntry{
		Level:        level,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Message:      message,
		Details:      details,
	}

	return l.Log(ctx, entry)
}

// LogStateChange logs a state change event
func (l *DefaultAuditLogger) LogStateChange(ctx context.Context, resourceType, resourceID string, oldState, newState saga.SagaState, userID string, details map[string]interface{}) error {
	entry := &AuditEntry{
		Level:        AuditLevelInfo,
		Action:       AuditActionStateChanged,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		UserID:       userID,
		OldState:     oldState.String(),
		NewState:     newState.String(),
		Message:      fmt.Sprintf("State changed from %s to %s", oldState.String(), newState.String()),
		Details:      details,
	}

	return l.Log(ctx, entry)
}

// LogAuthEvent logs an authentication or authorization event
func (l *DefaultAuditLogger) LogAuthEvent(ctx context.Context, action AuditAction, userID string, success bool, details map[string]interface{}) error {
	level := AuditLevelInfo
	if !success {
		level = AuditLevelWarning
	}

	message := fmt.Sprintf("Authentication action: %s", action)
	if !success {
		message = fmt.Sprintf("Failed %s", message)
	}

	entry := &AuditEntry{
		Level:        level,
		Action:       action,
		ResourceType: "auth",
		UserID:       userID,
		Message:      message,
		Details:      details,
	}

	return l.Log(ctx, entry)
}

// Query retrieves audit entries based on filter criteria
func (l *DefaultAuditLogger) Query(ctx context.Context, filter *AuditFilter) ([]*AuditEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.closed {
		return nil, fmt.Errorf("audit logger is closed")
	}

	return l.storage.Query(ctx, filter)
}

// Count returns the total number of audit entries matching the filter
func (l *DefaultAuditLogger) Count(ctx context.Context, filter *AuditFilter) (int64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.closed {
		return 0, fmt.Errorf("audit logger is closed")
	}

	return l.storage.Count(ctx, filter)
}

// Close closes the audit logger and releases resources
func (l *DefaultAuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true
	return l.storage.Close()
}

// generateAuditID generates a unique identifier for an audit entry
func generateAuditID() string {
	return fmt.Sprintf("audit-%d", time.Now().UnixNano())
}

// SagaAuditHook implements saga.StateChangeListener for automatic audit logging
type SagaAuditHook struct {
	auditLogger AuditLogger
	logger      *zap.Logger
}

// NewSagaAuditHook creates a new saga audit hook
func NewSagaAuditHook(auditLogger AuditLogger) *SagaAuditHook {
	hook := &SagaAuditHook{
		auditLogger: auditLogger,
		logger:      logger.Logger,
	}

	// Initialize logger if not set
	if hook.logger == nil {
		hook.logger = zap.NewNop()
	}

	return hook
}

// OnStateChange handles state change events from saga state manager
// This implements the StateChangeListener interface
func (h *SagaAuditHook) OnStateChange(ctx context.Context, sagaID string, oldState, newState saga.SagaState, metadata map[string]interface{}) {
	// Extract user ID from context if available
	userID := ""
	if authCtx, ok := AuthFromContext(ctx); ok && authCtx.Credentials != nil {
		userID = authCtx.Credentials.UserID
	}

	// Log the state change
	if err := h.auditLogger.LogStateChange(ctx, "saga", sagaID, oldState, newState, userID, metadata); err != nil {
		h.logger.Error("Failed to log state change audit entry",
			zap.String("saga_id", sagaID),
			zap.String("old_state", oldState.String()),
			zap.String("new_state", newState.String()),
			zap.Error(err))
	}
}

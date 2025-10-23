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
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewAuditTracker(t *testing.T) {
	tests := []struct {
		name    string
		config  *AuditTrackerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &AuditTrackerConfig{
				AuditLogger: createTestAuditLogger(t),
			},
			wantErr: false,
		},
		{
			name: "missing audit logger",
			config: &AuditTrackerConfig{
				AuditLogger: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker, err := NewAuditTracker(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAuditTracker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tracker == nil {
				t.Error("NewAuditTracker() returned nil tracker")
			}
			if tracker != nil {
				defer tracker.Close()
			}
		})
	}
}

func TestAuditTracker_StartTrace(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()

	tests := []struct {
		name      string
		operation string
		category  AuditCategory
		tags      []AuditTag
		wantErr   bool
	}{
		{
			name:      "start saga trace",
			operation: "create-order",
			category:  CategorySaga,
			tags:      []AuditTag{TagCritical, TagManual},
			wantErr:   false,
		},
		{
			name:      "start auth trace",
			operation: "user-login",
			category:  CategoryAuth,
			tags:      []AuditTag{TagSensitive},
			wantErr:   false,
		},
		{
			name:      "start trace with no tags",
			operation: "data-sync",
			category:  CategoryData,
			tags:      nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trace, err := tracker.StartTrace(ctx, tt.operation, tt.category, tt.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartTrace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if trace == nil {
					t.Fatal("StartTrace() returned nil trace")
				}
				if trace.TraceID == "" {
					t.Error("StartTrace() trace ID is empty")
				}
				if trace.Operation != tt.operation {
					t.Errorf("StartTrace() operation = %v, want %v", trace.Operation, tt.operation)
				}
				if trace.Category != tt.category {
					t.Errorf("StartTrace() category = %v, want %v", trace.Category, tt.category)
				}
				if trace.Status != "in_progress" {
					t.Errorf("StartTrace() status = %v, want in_progress", trace.Status)
				}
				if len(trace.Tags) != len(tt.tags) {
					t.Errorf("StartTrace() tags length = %v, want %v", len(trace.Tags), len(tt.tags))
				}
			}
		})
	}
}

func TestAuditTracker_StartChildTrace(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()

	// Create parent trace
	parent, err := tracker.StartTrace(ctx, "parent-op", CategorySaga, nil)
	if err != nil {
		t.Fatalf("Failed to create parent trace: %v", err)
	}

	tests := []struct {
		name          string
		parentTraceID string
		operation     string
		category      AuditCategory
		tags          []AuditTag
		wantErr       bool
	}{
		{
			name:          "valid child trace",
			parentTraceID: parent.TraceID,
			operation:     "child-op",
			category:      CategorySaga,
			tags:          []AuditTag{TagAutomated},
			wantErr:       false,
		},
		{
			name:          "invalid parent trace ID",
			parentTraceID: "non-existent",
			operation:     "child-op",
			category:      CategorySaga,
			tags:          nil,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trace, err := tracker.StartChildTrace(ctx, tt.parentTraceID, tt.operation, tt.category, tt.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartChildTrace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if trace == nil {
					t.Fatal("StartChildTrace() returned nil trace")
				}
				if trace.ParentTraceID != tt.parentTraceID {
					t.Errorf("StartChildTrace() parentTraceID = %v, want %v", trace.ParentTraceID, tt.parentTraceID)
				}
			}
		})
	}
}

func TestAuditTracker_AddStep(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()

	// Create a trace
	trace, err := tracker.StartTrace(ctx, "test-op", CategorySaga, nil)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	tests := []struct {
		name     string
		traceID  string
		stepName string
		wantErr  bool
	}{
		{
			name:     "add first step",
			traceID:  trace.TraceID,
			stepName: "step-1",
			wantErr:  false,
		},
		{
			name:     "add second step",
			traceID:  trace.TraceID,
			stepName: "step-2",
			wantErr:  false,
		},
		{
			name:     "invalid trace ID",
			traceID:  "non-existent",
			stepName: "step-3",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step, err := tracker.AddStep(ctx, tt.traceID, tt.stepName)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddStep() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if step == nil {
					t.Fatal("AddStep() returned nil step")
				}
				if step.Name != tt.stepName {
					t.Errorf("AddStep() name = %v, want %v", step.Name, tt.stepName)
				}
				if step.Status != "in_progress" {
					t.Errorf("AddStep() status = %v, want in_progress", step.Status)
				}
			}
		})
	}
}

func TestAuditTracker_CompleteStep(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()

	// Create a trace and add a step
	trace, err := tracker.StartTrace(ctx, "test-op", CategorySaga, nil)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	step, err := tracker.AddStep(ctx, trace.TraceID, "test-step")
	if err != nil {
		t.Fatalf("Failed to add step: %v", err)
	}

	tests := []struct {
		name         string
		traceID      string
		stepSequence int
		auditEntryID string
		wantErr      bool
	}{
		{
			name:         "complete valid step",
			traceID:      trace.TraceID,
			stepSequence: step.Sequence,
			auditEntryID: "audit-123",
			wantErr:      false,
		},
		{
			name:         "invalid trace ID",
			traceID:      "non-existent",
			stepSequence: 1,
			auditEntryID: "audit-456",
			wantErr:      true,
		},
		{
			name:         "invalid step sequence",
			traceID:      trace.TraceID,
			stepSequence: 999,
			auditEntryID: "audit-789",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tracker.CompleteStep(ctx, tt.traceID, tt.stepSequence, tt.auditEntryID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompleteStep() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Verify step was completed
	updatedTrace, err := tracker.GetTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("Failed to get trace: %v", err)
	}
	if len(updatedTrace.Steps) > 0 {
		if updatedTrace.Steps[0].Status != "success" {
			t.Errorf("Step status = %v, want success", updatedTrace.Steps[0].Status)
		}
	}
}

func TestAuditTracker_FailStep(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()

	// Create a trace and add a step
	trace, err := tracker.StartTrace(ctx, "test-op", CategorySaga, nil)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	step, err := tracker.AddStep(ctx, trace.TraceID, "test-step")
	if err != nil {
		t.Fatalf("Failed to add step: %v", err)
	}

	testErr := errors.New("test error")

	err = tracker.FailStep(ctx, trace.TraceID, step.Sequence, testErr, "audit-123")
	if err != nil {
		t.Errorf("FailStep() error = %v", err)
	}

	// Verify step was failed
	updatedTrace, err := tracker.GetTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("Failed to get trace: %v", err)
	}
	if len(updatedTrace.Steps) > 0 {
		if updatedTrace.Steps[0].Status != "failed" {
			t.Errorf("Step status = %v, want failed", updatedTrace.Steps[0].Status)
		}
		if updatedTrace.Steps[0].Error != testErr.Error() {
			t.Errorf("Step error = %v, want %v", updatedTrace.Steps[0].Error, testErr.Error())
		}
	}
}

func TestAuditTracker_CompleteTrace(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()

	// Create a trace
	trace, err := tracker.StartTrace(ctx, "test-op", CategorySaga, nil)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	// Wait a bit to ensure duration is measurable
	time.Sleep(10 * time.Millisecond)

	err = tracker.CompleteTrace(ctx, trace.TraceID)
	if err != nil {
		t.Errorf("CompleteTrace() error = %v", err)
	}

	// Verify trace was completed
	updatedTrace, err := tracker.GetTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("Failed to get trace: %v", err)
	}
	if updatedTrace.Status != "success" {
		t.Errorf("Trace status = %v, want success", updatedTrace.Status)
	}
	if updatedTrace.Duration == 0 {
		t.Error("Trace duration is zero")
	}
	if updatedTrace.EndTime.IsZero() {
		t.Error("Trace end time is zero")
	}
}

func TestAuditTracker_FailTrace(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()

	// Create a trace
	trace, err := tracker.StartTrace(ctx, "test-op", CategorySaga, nil)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	testErr := errors.New("trace failed")

	err = tracker.FailTrace(ctx, trace.TraceID, testErr)
	if err != nil {
		t.Errorf("FailTrace() error = %v", err)
	}

	// Verify trace was failed
	updatedTrace, err := tracker.GetTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("Failed to get trace: %v", err)
	}
	if updatedTrace.Status != "failed" {
		t.Errorf("Trace status = %v, want failed", updatedTrace.Status)
	}
	if updatedTrace.Error != testErr.Error() {
		t.Errorf("Trace error = %v, want %v", updatedTrace.Error, testErr.Error())
	}
}

func TestAuditTracker_GetTracesByCategory(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()

	// Create traces in different categories
	_, err := tracker.StartTrace(ctx, "saga-op", CategorySaga, nil)
	if err != nil {
		t.Fatalf("Failed to create saga trace: %v", err)
	}

	_, err = tracker.StartTrace(ctx, "auth-op", CategoryAuth, nil)
	if err != nil {
		t.Fatalf("Failed to create auth trace: %v", err)
	}

	_, err = tracker.StartTrace(ctx, "another-saga-op", CategorySaga, nil)
	if err != nil {
		t.Fatalf("Failed to create another saga trace: %v", err)
	}

	// Get saga traces
	sagaTraces, err := tracker.GetTracesByCategory(ctx, CategorySaga)
	if err != nil {
		t.Errorf("GetTracesByCategory() error = %v", err)
	}
	if len(sagaTraces) != 2 {
		t.Errorf("GetTracesByCategory() returned %d traces, want 2", len(sagaTraces))
	}

	// Get auth traces
	authTraces, err := tracker.GetTracesByCategory(ctx, CategoryAuth)
	if err != nil {
		t.Errorf("GetTracesByCategory() error = %v", err)
	}
	if len(authTraces) != 1 {
		t.Errorf("GetTracesByCategory() returned %d traces, want 1", len(authTraces))
	}
}

func TestAuditTracker_GetTracesByTag(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()

	// Create traces with different tags
	_, err := tracker.StartTrace(ctx, "op1", CategorySaga, []AuditTag{TagCritical, TagManual})
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	_, err = tracker.StartTrace(ctx, "op2", CategoryAuth, []AuditTag{TagSensitive})
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	_, err = tracker.StartTrace(ctx, "op3", CategoryData, []AuditTag{TagCritical})
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	// Get critical traces
	criticalTraces, err := tracker.GetTracesByTag(ctx, TagCritical)
	if err != nil {
		t.Errorf("GetTracesByTag() error = %v", err)
	}
	if len(criticalTraces) != 2 {
		t.Errorf("GetTracesByTag() returned %d traces, want 2", len(criticalTraces))
	}

	// Get sensitive traces
	sensitiveTraces, err := tracker.GetTracesByTag(ctx, TagSensitive)
	if err != nil {
		t.Errorf("GetTracesByTag() error = %v", err)
	}
	if len(sensitiveTraces) != 1 {
		t.Errorf("GetTracesByTag() returned %d traces, want 1", len(sensitiveTraces))
	}
}

func TestAuditTracker_GetTracesByTimeRange(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()

	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now.Add(1 * time.Hour)

	// Create some traces
	_, err := tracker.StartTrace(ctx, "op1", CategorySaga, nil)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_, err = tracker.StartTrace(ctx, "op2", CategoryAuth, nil)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	// Get traces in time range
	traces, err := tracker.GetTracesByTimeRange(ctx, startTime, endTime)
	if err != nil {
		t.Errorf("GetTracesByTimeRange() error = %v", err)
	}
	if len(traces) != 2 {
		t.Errorf("GetTracesByTimeRange() returned %d traces, want 2", len(traces))
	}

	// Get traces in past time range (should return 0)
	pastStart := now.Add(-2 * time.Hour)
	pastEnd := now.Add(-1 * time.Hour)
	pastTraces, err := tracker.GetTracesByTimeRange(ctx, pastStart, pastEnd)
	if err != nil {
		t.Errorf("GetTracesByTimeRange() error = %v", err)
	}
	if len(pastTraces) != 0 {
		t.Errorf("GetTracesByTimeRange() returned %d traces, want 0", len(pastTraces))
	}
}

func TestAuditTracker_LinkAuditEntry(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()

	// Create a trace
	trace, err := tracker.StartTrace(ctx, "test-op", CategorySaga, nil)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	auditEntryID := "audit-123"

	err = tracker.LinkAuditEntry(ctx, trace.TraceID, auditEntryID)
	if err != nil {
		t.Errorf("LinkAuditEntry() error = %v", err)
	}

	// Verify entry was linked
	updatedTrace, err := tracker.GetTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("Failed to get trace: %v", err)
	}

	found := false
	for _, id := range updatedTrace.AuditEntryIDs {
		if id == auditEntryID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Audit entry ID was not linked to trace")
	}
}

func TestAuditTracker_Close(t *testing.T) {
	tracker := createTestTracker(t)

	err := tracker.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Operations after close should fail
	ctx := context.Background()
	_, err = tracker.StartTrace(ctx, "test-op", CategorySaga, nil)
	if err == nil {
		t.Error("StartTrace() should fail after Close()")
	}
}

func TestAddTagsToEntry(t *testing.T) {
	entry := &AuditEntry{
		ID:      "test-1",
		Message: "test message",
	}

	category := CategorySaga
	tags := []AuditTag{TagCritical, TagManual}

	AddTagsToEntry(entry, category, tags)

	if entry.Metadata == nil {
		t.Fatal("Metadata is nil")
	}

	gotCategory := GetCategoryFromEntry(entry)
	if gotCategory != category {
		t.Errorf("GetCategoryFromEntry() = %v, want %v", gotCategory, category)
	}

	gotTags := GetTagsFromEntry(entry)
	if len(gotTags) != len(tags) {
		t.Errorf("GetTagsFromEntry() returned %d tags, want %d", len(gotTags), len(tags))
	}
}

func TestGetCategoryFromEntry(t *testing.T) {
	tests := []struct {
		name     string
		entry    *AuditEntry
		expected AuditCategory
	}{
		{
			name: "with category",
			entry: &AuditEntry{
				Metadata: map[string]string{"category": string(CategorySaga)},
			},
			expected: CategorySaga,
		},
		{
			name: "without metadata",
			entry: &AuditEntry{
				Metadata: nil,
			},
			expected: "",
		},
		{
			name: "without category",
			entry: &AuditEntry{
				Metadata: map[string]string{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCategoryFromEntry(tt.entry)
			if result != tt.expected {
				t.Errorf("GetCategoryFromEntry() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetTagsFromEntry(t *testing.T) {
	tests := []struct {
		name     string
		entry    *AuditEntry
		expected []AuditTag
	}{
		{
			name: "with tags",
			entry: &AuditEntry{
				Metadata: map[string]string{"tags": "critical,manual"},
			},
			expected: []AuditTag{TagCritical, TagManual},
		},
		{
			name: "without metadata",
			entry: &AuditEntry{
				Metadata: nil,
			},
			expected: nil,
		},
		{
			name: "without tags",
			entry: &AuditEntry{
				Metadata: map[string]string{},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTagsFromEntry(tt.entry)
			if len(result) != len(tt.expected) {
				t.Errorf("GetTagsFromEntry() returned %d tags, want %d", len(result), len(tt.expected))
			}
		})
	}
}

// Helper functions

func createTestTracker(t *testing.T) *DefaultAuditTracker {
	t.Helper()
	auditLogger := createTestAuditLogger(t)
	tracker, err := NewAuditTracker(&AuditTrackerConfig{
		AuditLogger: auditLogger,
	})
	if err != nil {
		t.Fatalf("Failed to create test tracker: %v", err)
	}
	return tracker
}

func TestAuditTracker_ImmutabilityAndThreadSafety(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()

	// Create a trace with complex nested data
	tags := []AuditTag{TagCritical, TagSensitive, TagManual}
	trace, err := tracker.StartTrace(ctx, "test-operation", CategorySaga, tags)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	// Add a step with details
	step, err := tracker.AddStep(ctx, trace.TraceID, "test-step")
	if err != nil {
		t.Fatalf("Failed to add step: %v", err)
	}

	// Complete the step
	err = tracker.CompleteStep(ctx, trace.TraceID, step.Sequence, "audit-123")
	if err != nil {
		t.Fatalf("Failed to complete step: %v", err)
	}

	// Add some context and metadata to the trace
	// Note: We need to access the internal trace directly to add context for testing
	tracker.mu.Lock()
	if internalTrace, exists := tracker.traces[trace.TraceID]; exists {
		internalTrace.Context["key1"] = "value1"
		internalTrace.Context["key2"] = map[string]interface{}{"nested": "data"}
		internalTrace.Metadata["meta1"] = "metavalue1"
		internalTrace.Metadata["meta2"] = "metavalue2"
	}
	tracker.mu.Unlock()

	// Test 1: Verify GetTrace returns a deep copy
	retrievedTrace, err := tracker.GetTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("Failed to get trace: %v", err)
	}

	// Modify the retrieved trace
	retrievedTrace.Status = "modified"
	retrievedTrace.Tags[0] = "modified-tag"
	if len(retrievedTrace.Steps) > 0 {
		retrievedTrace.Steps[0].Status = "modified-step"
		if retrievedTrace.Steps[0].Details != nil {
			retrievedTrace.Steps[0].Details["modified"] = true
		}
	}
	retrievedTrace.AuditEntryIDs = append(retrievedTrace.AuditEntryIDs, "modified-id")
	if retrievedTrace.Context != nil {
		retrievedTrace.Context["modified"] = true
	}
	if retrievedTrace.Metadata != nil {
		retrievedTrace.Metadata["modified"] = "true"
	}

	// Verify the original trace is unchanged
	originalTrace, err := tracker.GetTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("Failed to get original trace: %v", err)
	}

	if originalTrace.Status == "modified" {
		t.Error("Original trace status was modified through returned copy")
	}
	if len(originalTrace.Tags) > 0 && originalTrace.Tags[0] == "modified-tag" {
		t.Error("Original trace tags were modified through returned copy")
	}
	if len(originalTrace.Steps) > 0 && originalTrace.Steps[0].Status == "modified-step" {
		t.Error("Original step status was modified through returned copy")
	}
	if len(originalTrace.AuditEntryIDs) > 0 && originalTrace.AuditEntryIDs[len(originalTrace.AuditEntryIDs)-1] == "modified-id" {
		t.Error("Original audit entry IDs were modified through returned copy")
	}
	if originalTrace.Context != nil && originalTrace.Context["modified"] == true {
		t.Error("Original context was modified through returned copy")
	}
	if originalTrace.Metadata != nil && originalTrace.Metadata["modified"] == "true" {
		t.Error("Original metadata was modified through returned copy")
	}

	// Test 2: Verify GetTracesByCategory returns deep copies
	categoryTraces, err := tracker.GetTracesByCategory(ctx, CategorySaga)
	if err != nil {
		t.Fatalf("Failed to get traces by category: %v", err)
	}
	if len(categoryTraces) == 0 {
		t.Fatal("No traces returned for category")
	}

	// Modify the first trace from the category results
	categoryTraces[0].Status = "category-modified"

	// Verify the original is unchanged
	originalTraceAgain, err := tracker.GetTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("Failed to get trace again: %v", err)
	}
	if originalTraceAgain.Status == "category-modified" {
		t.Error("Original trace was modified through category query results")
	}

	// Test 3: Verify GetTracesByTag returns deep copies
	tagTraces, err := tracker.GetTracesByTag(ctx, TagCritical)
	if err != nil {
		t.Fatalf("Failed to get traces by tag: %v", err)
	}
	if len(tagTraces) == 0 {
		t.Fatal("No traces returned for tag")
	}

	// Modify the first trace from the tag results
	tagTraces[0].Status = "tag-modified"

	// Verify the original is unchanged
	originalTraceThird, err := tracker.GetTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("Failed to get trace third time: %v", err)
	}
	if originalTraceThird.Status == "tag-modified" {
		t.Error("Original trace was modified through tag query results")
	}

	// Test 4: Verify GetTracesByTimeRange returns deep copies
	now := time.Now()
	timeTraces, err := tracker.GetTracesByTimeRange(ctx, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to get traces by time range: %v", err)
	}
	if len(timeTraces) == 0 {
		t.Fatal("No traces returned for time range")
	}

	// Modify the first trace from the time range results
	timeTraces[0].Status = "time-modified"

	// Verify the original is unchanged
	originalTraceFourth, err := tracker.GetTrace(ctx, trace.TraceID)
	if err != nil {
		t.Fatalf("Failed to get trace fourth time: %v", err)
	}
	if originalTraceFourth.Status == "time-modified" {
		t.Error("Original trace was modified through time range query results")
	}
}

func TestAuditTracker_ConcurrentAccess(t *testing.T) {
	tracker := createTestTracker(t)
	defer tracker.Close()

	ctx := context.Background()
	numGoroutines := 10
	numOperations := 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*2)

	// Create traces concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				trace, err := tracker.StartTrace(ctx, "concurrent-op", CategorySaga, []AuditTag{TagCritical})
				if err != nil {
					errors <- err
					return
				}

				// Try to read the trace immediately
				retrieved, err := tracker.GetTrace(ctx, trace.TraceID)
				if err != nil {
					errors <- err
					return
				}

				// Modify the retrieved copy (should not affect the original)
				retrieved.Status = "modified-copy"

				// Add a step
				step, err := tracker.AddStep(ctx, trace.TraceID, "concurrent-step")
				if err != nil {
					errors <- err
					return
				}

				// Complete the step
				err = tracker.CompleteStep(ctx, trace.TraceID, step.Sequence, "audit-concurrent")
				if err != nil {
					errors <- err
					return
				}

				// Complete the trace
				err = tracker.CompleteTrace(ctx, trace.TraceID)
				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	// Read traces concurrently while writes are happening
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Get traces by category
				traces, err := tracker.GetTracesByCategory(ctx, CategorySaga)
				if err != nil {
					errors <- err
					return
				}

				// Modify all returned traces (should not affect originals)
				for _, trace := range traces {
					trace.Status = "concurrent-read-modified"
					if len(trace.Tags) > 0 {
						trace.Tags[0] = "modified-tag"
					}
					if len(trace.Steps) > 0 {
						trace.Steps[0].Status = "modified-step"
					}
					if trace.Context != nil {
						trace.Context["concurrent-modified"] = true
					}
					if trace.Metadata != nil {
						trace.Metadata["concurrent"] = "modified"
					}
				}

				// Get traces by tag
				tagTraces, err := tracker.GetTracesByTag(ctx, TagCritical)
				if err != nil {
					errors <- err
					return
				}

				// Modify tag traces too
				for _, trace := range tagTraces {
					trace.Status = "tag-concurrent-modified"
				}

				// Small delay to allow interleaving
				time.Sleep(time.Microsecond)
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}

	// Verify all traces have correct status (should be "success", not "modified")
	sagaTraces, err := tracker.GetTracesByCategory(ctx, CategorySaga)
	if err != nil {
		t.Fatalf("Failed to get final saga traces: %v", err)
	}

	expectedCount := numGoroutines * numOperations
	if len(sagaTraces) != expectedCount {
		t.Errorf("Expected %d traces, got %d", expectedCount, len(sagaTraces))
	}

	incorrectStatusCount := 0
	for _, trace := range sagaTraces {
		if trace.Status != "success" {
			incorrectStatusCount++
		}
	}
	if incorrectStatusCount > 0 {
		t.Errorf("Found %d traces with incorrect status (should all be 'success')", incorrectStatusCount)
	}
}

func TestDeepCopyTraceFunction(t *testing.T) {
	// Test the deepCopyTrace function directly
	original := &OperationTrace{
		TraceID:       "test-trace",
		ParentTraceID: "parent-trace",
		Operation:     "test-operation",
		Category:      CategorySaga,
		Tags:          []AuditTag{TagCritical, TagSensitive},
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(time.Hour),
		Duration:      time.Hour,
		Status:        "success",
		UserID:        "user123",
		ResourceType:  "resource",
		ResourceID:    "resource123",
		AuditEntryIDs: []string{"audit1", "audit2"},
		Context: map[string]interface{}{
			"key1": "value1",
			"key2": map[string]interface{}{
				"nested": "data",
				"number": 42,
			},
		},
		Error: "test error",
		Metadata: map[string]string{
			"meta1": "value1",
			"meta2": "value2",
		},
		Steps: []*OperationStep{
			{
				Name:         "step1",
				Sequence:     1,
				StartTime:    time.Now(),
				EndTime:      time.Now().Add(time.Minute),
				Duration:     time.Minute,
				Status:       "success",
				AuditEntryID: "step-audit1",
				Error:        "step error",
				Details: map[string]interface{}{
					"stepKey1": "stepValue1",
					"stepKey2": map[string]interface{}{
						"stepNested": "stepData",
					},
				},
			},
		},
	}

	// Create deep copy
	copy := deepCopyTrace(original)

	// Verify copy has same values
	if copy.TraceID != original.TraceID {
		t.Errorf("Copy TraceID = %v, want %v", copy.TraceID, original.TraceID)
	}
	if copy.Category != original.Category {
		t.Errorf("Copy Category = %v, want %v", copy.Category, original.Category)
	}
	if len(copy.Tags) != len(original.Tags) {
		t.Errorf("Copy Tags length = %d, want %d", len(copy.Tags), len(original.Tags))
	}
	if len(copy.Steps) != len(original.Steps) {
		t.Errorf("Copy Steps length = %d, want %d", len(copy.Steps), len(original.Steps))
	}
	if len(copy.AuditEntryIDs) != len(original.AuditEntryIDs) {
		t.Errorf("Copy AuditEntryIDs length = %d, want %d", len(copy.AuditEntryIDs), len(original.AuditEntryIDs))
	}
	if len(copy.Context) != len(original.Context) {
		t.Errorf("Copy Context length = %d, want %d", len(copy.Context), len(original.Context))
	}
	if len(copy.Metadata) != len(original.Metadata) {
		t.Errorf("Copy Metadata length = %d, want %d", len(copy.Metadata), len(original.Metadata))
	}

	// Modify the copy
	copy.Status = "modified"
	copy.Tags[0] = "modified-tag"
	copy.AuditEntryIDs[0] = "modified-audit"
	copy.Context["key1"] = "modified-value"
	copy.Context["newKey"] = "newValue"
	copy.Metadata["meta1"] = "modified-value"
	copy.Metadata["newMeta"] = "newValue"
	if len(copy.Steps) > 0 {
		copy.Steps[0].Status = "modified-step"
		copy.Steps[0].Details["stepKey1"] = "modified-step-value"
		copy.Steps[0].Details["newStepKey"] = "newStepValue"
	}

	// Verify original is unchanged
	if original.Status == "modified" {
		t.Error("Original status was modified through copy")
	}
	if original.Tags[0] == "modified-tag" {
		t.Error("Original tags were modified through copy")
	}
	if original.AuditEntryIDs[0] == "modified-audit" {
		t.Error("Original audit entry IDs were modified through copy")
	}
	if original.Context["key1"] == "modified-value" {
		t.Error("Original context was modified through copy")
	}
	if _, exists := original.Context["newKey"]; exists {
		t.Error("New key was added to original context through copy")
	}
	if original.Metadata["meta1"] == "modified-value" {
		t.Error("Original metadata was modified through copy")
	}
	if _, exists := original.Metadata["newMeta"]; exists {
		t.Error("New metadata was added to original through copy")
	}
	if len(original.Steps) > 0 {
		if original.Steps[0].Status == "modified-step" {
			t.Error("Original step status was modified through copy")
		}
		if original.Steps[0].Details["stepKey1"] == "modified-step-value" {
			t.Error("Original step details were modified through copy")
		}
		if _, exists := original.Steps[0].Details["newStepKey"]; exists {
			t.Error("New step detail was added to original through copy")
		}
	}
}

func TestDeepCopyStepFunction(t *testing.T) {
	// Test the deepCopyStep function directly
	original := &OperationStep{
		Name:         "test-step",
		Sequence:     1,
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(time.Minute),
		Duration:     time.Minute,
		Status:       "success",
		AuditEntryID: "step-audit",
		Error:        "step error",
		Details: map[string]interface{}{
			"key1": "value1",
			"key2": map[string]interface{}{
				"nested": "data",
				"number": 42,
			},
		},
	}

	// Create deep copy
	copy := deepCopyStep(original)

	// Verify copy has same values
	if copy.Name != original.Name {
		t.Errorf("Copy Name = %v, want %v", copy.Name, original.Name)
	}
	if copy.Sequence != original.Sequence {
		t.Errorf("Copy Sequence = %v, want %v", copy.Sequence, original.Sequence)
	}
	if copy.Status != original.Status {
		t.Errorf("Copy Status = %v, want %v", copy.Status, original.Status)
	}
	if len(copy.Details) != len(original.Details) {
		t.Errorf("Copy Details length = %d, want %d", len(copy.Details), len(original.Details))
	}

	// Modify the copy
	copy.Status = "modified"
	copy.Details["key1"] = "modified-value"
	copy.Details["newKey"] = "newValue"

	// Verify original is unchanged
	if original.Status == "modified" {
		t.Error("Original status was modified through copy")
	}
	if original.Details["key1"] == "modified-value" {
		t.Error("Original details were modified through copy")
	}
	if _, exists := original.Details["newKey"]; exists {
		t.Error("New detail was added to original through copy")
	}
}

func createTestAuditLogger(t *testing.T) AuditLogger {
	t.Helper()
	storage := NewMemoryAuditStorage()
	logger, err := NewAuditLogger(&AuditLoggerConfig{
		Storage: storage,
		Source:  "test",
	})
	if err != nil {
		t.Fatalf("Failed to create test audit logger: %v", err)
	}
	return logger
}

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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestPostgresStateStorage_ErrOptimisticLockFailed tests the optimistic lock error
func TestPostgresStateStorage_ErrOptimisticLockFailed(t *testing.T) {
	if ErrOptimisticLockFailed == nil {
		t.Error("ErrOptimisticLockFailed should not be nil")
	}
}

// TestPostgresPoolStats tests the PostgresPoolStats structure
func TestPostgresPoolStats(t *testing.T) {
	stats := &PostgresPoolStats{
		MaxOpenConnections:  10,
		OpenConnections:     5,
		InUse:               3,
		Idle:                2,
		WaitCount:           100,
		WaitDuration:        5 * time.Second,
		MaxIdleClosed:       10,
		MaxIdleTimeClosed:   20,
		MaxLifetimeClosed:   30,
	}

	if stats.MaxOpenConnections != 10 {
		t.Errorf("Expected MaxOpenConnections 10, got %d", stats.MaxOpenConnections)
	}
	if stats.OpenConnections != 5 {
		t.Errorf("Expected OpenConnections 5, got %d", stats.OpenConnections)
	}
}

// TestPostgresSagaInstance_Interface tests the postgresSagaInstance interface compliance
func TestPostgresSagaInstance_Interface(t *testing.T) {
	now := time.Now()
	data := &saga.SagaInstanceData{
		ID:           "test-id",
		DefinitionID: "test-def",
		State:        saga.StateRunning,
		CurrentStep:  1,
		TotalSteps:   3,
		CreatedAt:    now,
		UpdatedAt:    now,
		StartedAt:    &now,
		Timeout:      5 * time.Minute,
		Metadata:     map[string]interface{}{"key": "value"},
		TraceID:      "trace-123",
	}

	instance := &postgresSagaInstance{data: data}

	// Test interface compliance
	var _ saga.SagaInstance = instance

	// Test all getters
	if instance.GetID() != "test-id" {
		t.Errorf("GetID() = %s, want test-id", instance.GetID())
	}
	if instance.GetDefinitionID() != "test-def" {
		t.Errorf("GetDefinitionID() = %s, want test-def", instance.GetDefinitionID())
	}
	if instance.GetState() != saga.StateRunning {
		t.Errorf("GetState() = %v, want %v", instance.GetState(), saga.StateRunning)
	}
	if instance.GetCurrentStep() != 1 {
		t.Errorf("GetCurrentStep() = %d, want 1", instance.GetCurrentStep())
	}
	if instance.GetTotalSteps() != 3 {
		t.Errorf("GetTotalSteps() = %d, want 3", instance.GetTotalSteps())
	}
	if instance.GetCompletedSteps() != 1 {
		t.Errorf("GetCompletedSteps() = %d, want 1", instance.GetCompletedSteps())
	}
	if instance.GetTimeout() != 5*time.Minute {
		t.Errorf("GetTimeout() = %v, want %v", instance.GetTimeout(), 5*time.Minute)
	}
	if instance.GetTraceID() != "trace-123" {
		t.Errorf("GetTraceID() = %s, want trace-123", instance.GetTraceID())
	}
	if instance.IsTerminal() {
		t.Error("IsTerminal() = true, want false for Running state")
	}
	if !instance.GetStartTime().Equal(now) {
		t.Errorf("GetStartTime() = %v, want %v", instance.GetStartTime(), now)
	}

	// Test GetEndTime with nil values
	if !instance.GetEndTime().IsZero() {
		t.Error("GetEndTime() should return zero time when both CompletedAt and TimedOutAt are nil")
	}

	// Test GetEndTime with CompletedAt
	completedTime := now.Add(1 * time.Hour)
	data.CompletedAt = &completedTime
	if !instance.GetEndTime().Equal(completedTime) {
		t.Errorf("GetEndTime() = %v, want %v", instance.GetEndTime(), completedTime)
	}

	// Test GetEndTime with TimedOutAt
	data.CompletedAt = nil
	timedOutTime := now.Add(2 * time.Hour)
	data.TimedOutAt = &timedOutTime
	if !instance.GetEndTime().Equal(timedOutTime) {
		t.Errorf("GetEndTime() = %v, want %v", instance.GetEndTime(), timedOutTime)
	}

	// Test IsActive for different states
	data.State = saga.StateRunning
	if !instance.IsActive() {
		t.Error("IsActive() should return true for Running state")
	}

	data.State = saga.StateCompleted
	if instance.IsActive() {
		t.Error("IsActive() should return false for Completed state")
	}

	if !instance.IsTerminal() {
		t.Error("IsTerminal() should return true for Completed state")
	}
}

// TestPostgresStateStorage_JoinStrings tests the joinStrings helper
func TestPostgresStateStorage_JoinStrings_Unit(t *testing.T) {
	tests := []struct {
		name string
		strs []string
		sep  string
		want string
	}{
		{
			name: "empty slice",
			strs: []string{},
			sep:  ",",
			want: "",
		},
		{
			name: "single element",
			strs: []string{"a"},
			sep:  ",",
			want: "a",
		},
		{
			name: "two elements",
			strs: []string{"a", "b"},
			sep:  ",",
			want: "a,b",
		},
		{
			name: "three elements with AND",
			strs: []string{"x", "y", "z"},
			sep:  " AND ",
			want: "x AND y AND z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinStrings(tt.strs, tt.sep)
			if got != tt.want {
				t.Errorf("joinStrings() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestPostgresStateStorage_Contains_Unit tests the contains helper
func TestPostgresStateStorage_Contains_Unit(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "exact match",
			s:      "hello",
			substr: "hello",
			want:   true,
		},
		{
			name:   "substring at start",
			s:      "hello world",
			substr: "hello",
			want:   true,
		},
		{
			name:   "substring in middle",
			s:      "hello world",
			substr: "lo wo",
			want:   true,
		},
		{
			name:   "substring at end",
			s:      "hello world",
			substr: "world",
			want:   true,
		},
		{
			name:   "not found",
			s:      "hello world",
			substr: "foo",
			want:   false,
		},
		{
			name:   "empty substring",
			s:      "hello",
			substr: "",
			want:   true,
		},
		{
			name:   "empty string",
			s:      "",
			substr: "hello",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

// TestSagaTransaction_Interface tests transaction interface compliance
func TestSagaTransaction_Interface(t *testing.T) {
	// Create a minimal transaction for interface testing
	tx := &SagaTransaction{
		tx:        nil,
		storage:   nil,
		closed:    true,
		timeout:   30 * time.Second,
		startTime: time.Now(),
	}

	// Test checkClosed
	err := tx.checkClosed()
	if err != ErrTransactionClosed {
		t.Errorf("checkClosed() on closed transaction = %v, want %v", err, ErrTransactionClosed)
	}

	// Test checkClosed on open transaction
	tx.closed = false
	err = tx.checkClosed()
	if err != nil {
		t.Errorf("checkClosed() on open transaction = %v, want nil", err)
	}

	// Test checkTimeout
	tx.startTime = time.Now().Add(-1 * time.Hour)
	tx.timeout = 1 * time.Second
	err = tx.checkTimeout()
	if err == nil {
		t.Error("checkTimeout() should return error when timeout exceeded")
	}
}


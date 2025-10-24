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

package monitoring

import (
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// Note: TestSagaListRequest_ApplyDefaults already exists in api_query_test.go

func TestParseSagaState(t *testing.T) {
	tests := []struct {
		name     string
		stateStr string
		expected saga.SagaState
	}{
		// Capitalized versions
		{"Pending", "Pending", saga.StatePending},
		{"Running", "Running", saga.StateRunning},
		{"StepCompleted", "StepCompleted", saga.StateStepCompleted},
		{"Completed", "Completed", saga.StateCompleted},
		{"Compensating", "Compensating", saga.StateCompensating},
		{"Compensated", "Compensated", saga.StateCompensated},
		{"Failed", "Failed", saga.StateFailed},
		{"Cancelled", "Cancelled", saga.StateCancelled},

		// Lowercase versions
		{"pending lowercase", "pending", saga.StatePending},
		{"running lowercase", "running", saga.StateRunning},
		{"step_completed", "step_completed", saga.StateStepCompleted},
		{"completed lowercase", "completed", saga.StateCompleted},
		{"compensating lowercase", "compensating", saga.StateCompensating},
		{"compensated lowercase", "compensated", saga.StateCompensated},
		{"failed lowercase", "failed", saga.StateFailed},
		{"cancelled lowercase", "cancelled", saga.StateCancelled},

		// Invalid state
		{"invalid state", "invalid", saga.SagaState(-1)},
		{"empty string", "", saga.SagaState(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSagaState(tt.stateStr)
			if result != tt.expected {
				t.Errorf("parseSagaState(%q) = %v, want %v", tt.stateStr, result, tt.expected)
			}
		})
	}
}

func TestSagaListRequest_ToSagaFilter(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	tests := []struct {
		name    string
		request *SagaListRequest
		check   func(*testing.T, *saga.SagaFilter)
	}{
		{
			name: "basic pagination",
			request: &SagaListRequest{
				Page:     2,
				PageSize: 10,
			},
			check: func(t *testing.T, filter *saga.SagaFilter) {
				if filter.Limit != 10 {
					t.Errorf("Limit = %d, want 10", filter.Limit)
				}
				if filter.Offset != 10 {
					t.Errorf("Offset = %d, want 10", filter.Offset)
				}
			},
		},
		{
			name: "with states",
			request: &SagaListRequest{
				Page:     1,
				PageSize: 20,
				States:   []string{"running", "Failed", "completed"},
			},
			check: func(t *testing.T, filter *saga.SagaFilter) {
				if len(filter.States) != 3 {
					t.Errorf("States length = %d, want 3", len(filter.States))
				}
				expectedStates := map[saga.SagaState]bool{
					saga.StateRunning:   true,
					saga.StateFailed:    true,
					saga.StateCompleted: true,
				}
				for _, state := range filter.States {
					if !expectedStates[state] {
						t.Errorf("Unexpected state: %v", state)
					}
				}
			},
		},
		{
			name: "with invalid states filtered out",
			request: &SagaListRequest{
				Page:     1,
				PageSize: 20,
				States:   []string{"running", "invalid", "completed"},
			},
			check: func(t *testing.T, filter *saga.SagaFilter) {
				if len(filter.States) != 2 {
					t.Errorf("States length = %d, want 2 (invalid state should be filtered)", len(filter.States))
				}
			},
		},
		{
			name: "with definition IDs",
			request: &SagaListRequest{
				Page:          1,
				PageSize:      20,
				DefinitionIDs: []string{"def1", "def2", "def3"},
			},
			check: func(t *testing.T, filter *saga.SagaFilter) {
				if len(filter.DefinitionIDs) != 3 {
					t.Errorf("DefinitionIDs length = %d, want 3", len(filter.DefinitionIDs))
				}
			},
		},
		{
			name: "with time range",
			request: &SagaListRequest{
				Page:          1,
				PageSize:      20,
				CreatedAfter:  &yesterday,
				CreatedBefore: &now,
			},
			check: func(t *testing.T, filter *saga.SagaFilter) {
				if filter.CreatedAfter == nil {
					t.Error("CreatedAfter should not be nil")
				}
				if filter.CreatedBefore == nil {
					t.Error("CreatedBefore should not be nil")
				}
			},
		},
		{
			name: "with sort options",
			request: &SagaListRequest{
				Page:      1,
				PageSize:  20,
				SortBy:    "updated_at",
				SortOrder: "asc",
			},
			check: func(t *testing.T, filter *saga.SagaFilter) {
				if filter.SortBy != "updated_at" {
					t.Errorf("SortBy = %s, want updated_at", filter.SortBy)
				}
				if filter.SortOrder != "asc" {
					t.Errorf("SortOrder = %s, want asc", filter.SortOrder)
				}
			},
		},
		{
			name: "comprehensive filter",
			request: &SagaListRequest{
				Page:          3,
				PageSize:      15,
				States:        []string{"running", "pending"},
				DefinitionIDs: []string{"order-saga", "payment-saga"},
				CreatedAfter:  &yesterday,
				CreatedBefore: &now,
				SortBy:        "created_at",
				SortOrder:     "desc",
			},
			check: func(t *testing.T, filter *saga.SagaFilter) {
				if filter.Limit != 15 {
					t.Errorf("Limit = %d, want 15", filter.Limit)
				}
				if filter.Offset != 30 {
					t.Errorf("Offset = %d, want 30", filter.Offset)
				}
				if len(filter.States) != 2 {
					t.Errorf("States length = %d, want 2", len(filter.States))
				}
				if len(filter.DefinitionIDs) != 2 {
					t.Errorf("DefinitionIDs length = %d, want 2", len(filter.DefinitionIDs))
				}
				if filter.CreatedAfter == nil || filter.CreatedBefore == nil {
					t.Error("Time range should be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := tt.request.ToSagaFilter()
			if filter == nil {
				t.Fatal("ToSagaFilter() returned nil")
			}
			tt.check(t, filter)
		})
	}
}

func TestSagaListRequest_ToSagaFilter_Offset_Calculation(t *testing.T) {
	tests := []struct {
		page           int
		pageSize       int
		expectedOffset int
	}{
		{page: 1, pageSize: 10, expectedOffset: 0},
		{page: 2, pageSize: 10, expectedOffset: 10},
		{page: 3, pageSize: 20, expectedOffset: 40},
		{page: 5, pageSize: 25, expectedOffset: 100},
		{page: 10, pageSize: 15, expectedOffset: 135},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			request := &SagaListRequest{
				Page:     tt.page,
				PageSize: tt.pageSize,
			}
			filter := request.ToSagaFilter()
			if filter.Offset != tt.expectedOffset {
				t.Errorf("For page=%d, pageSize=%d: Offset = %d, want %d",
					tt.page, tt.pageSize, filter.Offset, tt.expectedOffset)
			}
		})
	}
}

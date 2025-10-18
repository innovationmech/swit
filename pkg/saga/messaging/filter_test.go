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

package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestFilterChain_AddFilter tests adding filters to the chain
func TestFilterChain_AddFilter(t *testing.T) {
	tests := []struct {
		name        string
		filters     []EventFilter
		expectError bool
		errorMsg    string
	}{
		{
			name: "add single filter",
			filters: []EventFilter{
				NewEventTypeFilter("type-filter", []SagaEventType{EventTypeSagaCompleted}, nil),
			},
			expectError: false,
		},
		{
			name: "add multiple filters",
			filters: []EventFilter{
				NewEventTypeFilter("type-filter", []SagaEventType{EventTypeSagaCompleted}, nil),
				NewSagaIDFilter("id-filter", []string{"saga-1"}, nil),
			},
			expectError: false,
		},
		{
			name: "add nil filter",
			filters: []EventFilter{
				nil,
			},
			expectError: true,
			errorMsg:    "filter cannot be nil",
		},
		{
			name: "add duplicate filter",
			filters: []EventFilter{
				NewEventTypeFilter("type-filter", []SagaEventType{EventTypeSagaCompleted}, nil),
				NewEventTypeFilter("type-filter", []SagaEventType{EventTypeSagaFailed}, nil),
			},
			expectError: true,
			errorMsg:    "filter with name type-filter already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := NewFilterChain()

			var err error
			for _, filter := range tt.filters {
				err = chain.AddFilter(filter)
				if err != nil {
					break
				}
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(chain.GetFilters()) != len(tt.filters) {
					t.Errorf("expected %d filters, got %d", len(tt.filters), len(chain.GetFilters()))
				}
			}
		})
	}
}

// TestFilterChain_RemoveFilter tests removing filters from the chain
func TestFilterChain_RemoveFilter(t *testing.T) {
	chain := NewFilterChain()
	filter1 := NewEventTypeFilter("filter-1", []SagaEventType{EventTypeSagaCompleted}, nil)
	filter2 := NewEventTypeFilter("filter-2", []SagaEventType{EventTypeSagaFailed}, nil)

	_ = chain.AddFilter(filter1)
	_ = chain.AddFilter(filter2)

	tests := []struct {
		name        string
		filterName  string
		expectError bool
	}{
		{
			name:        "remove existing filter",
			filterName:  "filter-1",
			expectError: false,
		},
		{
			name:        "remove non-existent filter",
			filterName:  "non-existent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := chain.RemoveFilter(tt.filterName)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestFilterChain_ShouldFilter tests the filter chain filtering logic
func TestFilterChain_ShouldFilter(t *testing.T) {
	tests := []struct {
		name           string
		filters        []EventFilter
		event          *saga.SagaEvent
		expectedFilter bool
	}{
		{
			name: "event passes all filters",
			filters: []EventFilter{
				NewEventTypeFilter("type-filter", []SagaEventType{EventTypeSagaCompleted}, nil),
			},
			event: &saga.SagaEvent{
				Type:    EventTypeSagaCompleted,
				SagaID:  "saga-1",
				Timestamp: time.Now(),
			},
			expectedFilter: false,
		},
		{
			name: "event filtered by type",
			filters: []EventFilter{
				NewEventTypeFilter("type-filter", []SagaEventType{EventTypeSagaCompleted}, nil),
			},
			event: &saga.SagaEvent{
				Type:    EventTypeSagaFailed,
				SagaID:  "saga-1",
				Timestamp: time.Now(),
			},
			expectedFilter: true,
		},
		{
			name: "event filtered by excluded type",
			filters: []EventFilter{
				NewEventTypeFilter("type-filter", nil, []SagaEventType{EventTypeSagaFailed}),
			},
			event: &saga.SagaEvent{
				Type:    EventTypeSagaFailed,
				SagaID:  "saga-1",
				Timestamp: time.Now(),
			},
			expectedFilter: true,
		},
		{
			name: "event filtered by saga ID",
			filters: []EventFilter{
				NewSagaIDFilter("id-filter", []string{"saga-1"}, nil),
			},
			event: &saga.SagaEvent{
				Type:    EventTypeSagaCompleted,
				SagaID:  "saga-2",
				Timestamp: time.Now(),
			},
			expectedFilter: true,
		},
		{
			name: "empty filter chain passes all events",
			filters: []EventFilter{},
			event: &saga.SagaEvent{
				Type:    EventTypeSagaCompleted,
				SagaID:  "saga-1",
				Timestamp: time.Now(),
			},
			expectedFilter: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := NewFilterChain()
			for _, filter := range tt.filters {
				_ = chain.AddFilter(filter)
			}

			ctx := context.Background()
			handlerCtx := &EventHandlerContext{
				MessageID: "msg-1",
				Timestamp: time.Now(),
			}

			shouldFilter, err := chain.ShouldFilter(ctx, tt.event, handlerCtx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if shouldFilter != tt.expectedFilter {
				t.Errorf("expected shouldFilter=%v, got %v", tt.expectedFilter, shouldFilter)
			}
		})
	}
}

// TestEventTypeFilter tests the event type filter
func TestEventTypeFilter(t *testing.T) {
	tests := []struct {
		name           string
		allowedTypes   []SagaEventType
		excludedTypes  []SagaEventType
		eventType      SagaEventType
		expectedFilter bool
	}{
		{
			name:           "allowed type passes",
			allowedTypes:   []SagaEventType{EventTypeSagaCompleted, EventTypeSagaFailed},
			excludedTypes:  nil,
			eventType:      EventTypeSagaCompleted,
			expectedFilter: false,
		},
		{
			name:           "non-allowed type filtered",
			allowedTypes:   []SagaEventType{EventTypeSagaCompleted},
			excludedTypes:  nil,
			eventType:      EventTypeSagaFailed,
			expectedFilter: true,
		},
		{
			name:           "excluded type filtered",
			allowedTypes:   nil,
			excludedTypes:  []SagaEventType{EventTypeSagaFailed},
			eventType:      EventTypeSagaFailed,
			expectedFilter: true,
		},
		{
			name:           "no restrictions passes all",
			allowedTypes:   nil,
			excludedTypes:  nil,
			eventType:      EventTypeSagaCompleted,
			expectedFilter: false,
		},
		{
			name:           "excluded takes precedence",
			allowedTypes:   []SagaEventType{EventTypeSagaCompleted},
			excludedTypes:  []SagaEventType{EventTypeSagaCompleted},
			eventType:      EventTypeSagaCompleted,
			expectedFilter: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewEventTypeFilter("test-filter", tt.allowedTypes, tt.excludedTypes)
			event := &saga.SagaEvent{
				Type:    tt.eventType,
				SagaID:  "saga-1",
				Timestamp: time.Now(),
			}

			ctx := context.Background()
			handlerCtx := &EventHandlerContext{MessageID: "msg-1"}

			shouldFilter, err := filter.ShouldFilter(ctx, event, handlerCtx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if shouldFilter != tt.expectedFilter {
				t.Errorf("expected shouldFilter=%v, got %v", tt.expectedFilter, shouldFilter)
			}
		})
	}
}

// TestSagaIDFilter tests the saga ID filter
func TestSagaIDFilter(t *testing.T) {
	tests := []struct {
		name           string
		allowedIDs     []string
		excludedIDs    []string
		sagaID         string
		expectedFilter bool
	}{
		{
			name:           "allowed ID passes",
			allowedIDs:     []string{"saga-1", "saga-2"},
			excludedIDs:    nil,
			sagaID:         "saga-1",
			expectedFilter: false,
		},
		{
			name:           "non-allowed ID filtered",
			allowedIDs:     []string{"saga-1"},
			excludedIDs:    nil,
			sagaID:         "saga-2",
			expectedFilter: true,
		},
		{
			name:           "excluded ID filtered",
			allowedIDs:     nil,
			excludedIDs:    []string{"saga-3"},
			sagaID:         "saga-3",
			expectedFilter: true,
		},
		{
			name:           "no restrictions passes all",
			allowedIDs:     nil,
			excludedIDs:    nil,
			sagaID:         "saga-any",
			expectedFilter: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewSagaIDFilter("test-filter", tt.allowedIDs, tt.excludedIDs)
			event := &saga.SagaEvent{
				Type:    EventTypeSagaCompleted,
				SagaID:  tt.sagaID,
				Timestamp: time.Now(),
			}

			ctx := context.Background()
			handlerCtx := &EventHandlerContext{MessageID: "msg-1"}

			shouldFilter, err := filter.ShouldFilter(ctx, event, handlerCtx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if shouldFilter != tt.expectedFilter {
				t.Errorf("expected shouldFilter=%v, got %v", tt.expectedFilter, shouldFilter)
			}
		})
	}
}

// TestMetadataFilter tests the metadata filter
func TestMetadataFilter(t *testing.T) {
	tests := []struct {
		name           string
		rules          []MetadataRule
		metadata       map[string]interface{}
		expectedFilter bool
		expectError    bool
	}{
		{
			name: "equals match filters",
			rules: []MetadataRule{
				{
					Key:           "environment",
					MatchType:     MetadataMatchEquals,
					ExpectedValue: "production",
				},
			},
			metadata: map[string]interface{}{
				"environment": "production",
			},
			expectedFilter: true,
		},
		{
			name: "equals no match passes",
			rules: []MetadataRule{
				{
					Key:           "environment",
					MatchType:     MetadataMatchEquals,
					ExpectedValue: "production",
				},
			},
			metadata: map[string]interface{}{
				"environment": "development",
			},
			expectedFilter: false,
		},
		{
			name: "regex match filters",
			rules: []MetadataRule{
				{
					Key:          "version",
					MatchType:    MetadataMatchRegex,
					ValuePattern: "^v1\\.",
				},
			},
			metadata: map[string]interface{}{
				"version": "v1.2.3",
			},
			expectedFilter: true,
		},
		{
			name: "no metadata key passes",
			rules: []MetadataRule{
				{
					Key:           "environment",
					MatchType:     MetadataMatchEquals,
					ExpectedValue: "production",
				},
			},
			metadata:       map[string]interface{}{},
			expectedFilter: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewMetadataFilter("test-filter", tt.rules)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error creating filter: %v", err)
				return
			}

			event := &saga.SagaEvent{
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
				Metadata:  tt.metadata,
			}

			ctx := context.Background()
			handlerCtx := &EventHandlerContext{MessageID: "msg-1"}

			shouldFilter, err := filter.ShouldFilter(ctx, event, handlerCtx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if shouldFilter != tt.expectedFilter {
				t.Errorf("expected shouldFilter=%v, got %v", tt.expectedFilter, shouldFilter)
			}
		})
	}
}

// TestCustomFilter tests the custom filter
func TestCustomFilter(t *testing.T) {
	tests := []struct {
		name           string
		filterFn       func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error)
		event          *saga.SagaEvent
		expectedFilter bool
	}{
		{
			name: "custom filter returns true",
			filterFn: func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
				return true, nil
			},
			event: &saga.SagaEvent{
				Type:    EventTypeSagaCompleted,
				SagaID:  "saga-1",
				Timestamp: time.Now(),
			},
			expectedFilter: true,
		},
		{
			name: "custom filter returns false",
			filterFn: func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
				return false, nil
			},
			event: &saga.SagaEvent{
				Type:    EventTypeSagaCompleted,
				SagaID:  "saga-1",
				Timestamp: time.Now(),
			},
			expectedFilter: false,
		},
		{
			name: "custom filter based on attempt count",
			filterFn: func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
				return event.Attempt > 5, nil
			},
			event: &saga.SagaEvent{
				Type:      EventTypeSagaStepCompleted,
				SagaID:    "saga-1",
				Attempt:   10,
				Timestamp: time.Now(),
			},
			expectedFilter: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewCustomFilter("test-filter", 50, tt.filterFn)

			ctx := context.Background()
			handlerCtx := &EventHandlerContext{MessageID: "msg-1"}

			shouldFilter, err := filter.ShouldFilter(ctx, tt.event, handlerCtx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if shouldFilter != tt.expectedFilter {
				t.Errorf("expected shouldFilter=%v, got %v", tt.expectedFilter, shouldFilter)
			}
		})
	}
}

// TestCompositeFilter tests the composite filter
func TestCompositeFilter(t *testing.T) {
	typeFilter := NewEventTypeFilter("type-filter", []SagaEventType{EventTypeSagaCompleted}, nil)
	idFilter := NewSagaIDFilter("id-filter", []string{"saga-1"}, nil)

	tests := []struct {
		name           string
		operator       FilterOperator
		filters        []EventFilter
		event          *saga.SagaEvent
		expectedFilter bool
	}{
		{
			name:     "AND operator - all filters match",
			operator: FilterOperatorAND,
			filters:  []EventFilter{typeFilter, idFilter},
			event: &saga.SagaEvent{
				Type:    EventTypeSagaCompleted,
				SagaID:  "saga-1",
				Timestamp: time.Now(),
			},
			expectedFilter: false, // Both pass, so not filtered
		},
		{
			name:     "AND operator - one filter fails",
			operator: FilterOperatorAND,
			filters:  []EventFilter{typeFilter, idFilter},
			event: &saga.SagaEvent{
				Type:    EventTypeSagaCompleted,
				SagaID:  "saga-2",
				Timestamp: time.Now(),
			},
			expectedFilter: false, // ID filter would filter, but we need both to filter for AND
		},
		{
			name:     "OR operator - one filter matches",
			operator: FilterOperatorOR,
			filters:  []EventFilter{typeFilter, idFilter},
			event: &saga.SagaEvent{
				Type:    EventTypeSagaFailed,
				SagaID:  "saga-1",
				Timestamp: time.Now(),
			},
			expectedFilter: true, // Type filter filters it
		},
		{
			name:     "OR operator - no filters match",
			operator: FilterOperatorOR,
			filters:  []EventFilter{typeFilter, idFilter},
			event: &saga.SagaEvent{
				Type:    EventTypeSagaCompleted,
				SagaID:  "saga-1",
				Timestamp: time.Now(),
			},
			expectedFilter: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewCompositeFilter("composite-filter", 70, tt.filters, tt.operator)

			ctx := context.Background()
			handlerCtx := &EventHandlerContext{MessageID: "msg-1"}

			shouldFilter, err := filter.ShouldFilter(ctx, tt.event, handlerCtx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if shouldFilter != tt.expectedFilter {
				t.Errorf("expected shouldFilter=%v, got %v", tt.expectedFilter, shouldFilter)
			}
		})
	}
}

// TestFilterChain_Metrics tests filter chain metrics collection
func TestFilterChain_Metrics(t *testing.T) {
	chain := NewFilterChain()
	filter1 := NewEventTypeFilter("filter-1", []SagaEventType{EventTypeSagaCompleted}, nil)
	filter2 := NewSagaIDFilter("filter-2", []string{"saga-1"}, nil)

	_ = chain.AddFilter(filter1)
	_ = chain.AddFilter(filter2)

	ctx := context.Background()
	handlerCtx := &EventHandlerContext{MessageID: "msg-1"}

	// Test events
	event1 := &saga.SagaEvent{
		Type:    EventTypeSagaCompleted,
		SagaID:  "saga-1",
		Timestamp: time.Now(),
	}
	event2 := &saga.SagaEvent{
		Type:    EventTypeSagaFailed,
		SagaID:  "saga-2",
		Timestamp: time.Now(),
	}

	// Process events
	_, _ = chain.ShouldFilter(ctx, event1, handlerCtx)
	_, _ = chain.ShouldFilter(ctx, event2, handlerCtx)

	metrics := chain.GetMetrics()

	if metrics.TotalEventsFiltered != 2 {
		t.Errorf("expected 2 events filtered, got %d", metrics.TotalEventsFiltered)
	}

	if metrics.FilterExecutions["filter-1"] != 2 {
		t.Errorf("expected filter-1 executed 2 times, got %d", metrics.FilterExecutions["filter-1"])
	}
}

// TestFilterPriority tests that filters are executed in priority order
func TestFilterPriority(t *testing.T) {
	chain := NewFilterChain()

	executionOrder := []string{}

	// Create filters with different priorities
	filter1 := NewCustomFilter("low-priority", 10, func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
		executionOrder = append(executionOrder, "low-priority")
		return false, nil
	})

	filter2 := NewCustomFilter("high-priority", 100, func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
		executionOrder = append(executionOrder, "high-priority")
		return false, nil
	})

	filter3 := NewCustomFilter("medium-priority", 50, func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
		executionOrder = append(executionOrder, "medium-priority")
		return false, nil
	})

	// Add filters in random order
	_ = chain.AddFilter(filter1)
	_ = chain.AddFilter(filter2)
	_ = chain.AddFilter(filter3)

	event := &saga.SagaEvent{
		Type:    EventTypeSagaCompleted,
		SagaID:  "saga-1",
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	handlerCtx := &EventHandlerContext{MessageID: "msg-1"}

	_, _ = chain.ShouldFilter(ctx, event, handlerCtx)

	// Verify execution order (highest priority first)
	if len(executionOrder) != 3 {
		t.Fatalf("expected 3 filters executed, got %d", len(executionOrder))
	}

	if executionOrder[0] != "high-priority" {
		t.Errorf("expected first filter to be high-priority, got %s", executionOrder[0])
	}
	if executionOrder[1] != "medium-priority" {
		t.Errorf("expected second filter to be medium-priority, got %s", executionOrder[1])
	}
	if executionOrder[2] != "low-priority" {
		t.Errorf("expected third filter to be low-priority, got %s", executionOrder[2])
	}
}


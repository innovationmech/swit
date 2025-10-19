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
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-1",
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
				Type:      EventTypeSagaFailed,
				SagaID:    "saga-1",
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
				Type:      EventTypeSagaFailed,
				SagaID:    "saga-1",
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
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-2",
				Timestamp: time.Now(),
			},
			expectedFilter: true,
		},
		{
			name:    "empty filter chain passes all events",
			filters: []EventFilter{},
			event: &saga.SagaEvent{
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-1",
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
				Type:      tt.eventType,
				SagaID:    "saga-1",
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
				Type:      EventTypeSagaCompleted,
				SagaID:    tt.sagaID,
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
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-1",
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
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-1",
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
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			expectedFilter: false, // Both pass, so not filtered
		},
		{
			name:     "AND operator - one filter fails",
			operator: FilterOperatorAND,
			filters:  []EventFilter{typeFilter, idFilter},
			event: &saga.SagaEvent{
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-2",
				Timestamp: time.Now(),
			},
			expectedFilter: false, // ID filter would filter, but we need both to filter for AND
		},
		{
			name:     "OR operator - one filter matches",
			operator: FilterOperatorOR,
			filters:  []EventFilter{typeFilter, idFilter},
			event: &saga.SagaEvent{
				Type:      EventTypeSagaFailed,
				SagaID:    "saga-1",
				Timestamp: time.Now(),
			},
			expectedFilter: true, // Type filter filters it
		},
		{
			name:     "OR operator - no filters match",
			operator: FilterOperatorOR,
			filters:  []EventFilter{typeFilter, idFilter},
			event: &saga.SagaEvent{
				Type:      EventTypeSagaCompleted,
				SagaID:    "saga-1",
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
		Type:      EventTypeSagaCompleted,
		SagaID:    "saga-1",
		Timestamp: time.Now(),
	}
	event2 := &saga.SagaEvent{
		Type:      EventTypeSagaFailed,
		SagaID:    "saga-2",
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
		Type:      EventTypeSagaCompleted,
		SagaID:    "saga-1",
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

// TestFilterChain_Clear tests clearing all filters from the chain
func TestFilterChain_Clear(t *testing.T) {
	tests := []struct {
		name           string
		initialFilters []EventFilter
		expectedCount  int
	}{
		{
			name:           "clear empty chain",
			initialFilters: []EventFilter{},
			expectedCount:  0,
		},
		{
			name: "clear chain with single filter",
			initialFilters: []EventFilter{
				NewEventTypeFilter("type-filter", []SagaEventType{EventTypeSagaCompleted}, nil),
			},
			expectedCount: 0,
		},
		{
			name: "clear chain with multiple filters",
			initialFilters: []EventFilter{
				NewEventTypeFilter("type-filter", []SagaEventType{EventTypeSagaCompleted}, nil),
				NewSagaIDFilter("id-filter", []string{"saga-1"}, nil),
				NewCustomFilter("custom-filter", 50, func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
					return false, nil
				}),
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := NewFilterChain()

			// Add initial filters
			for _, filter := range tt.initialFilters {
				_ = chain.AddFilter(filter)
			}

			// Verify filters were added
			if len(tt.initialFilters) > 0 && len(chain.GetFilters()) != len(tt.initialFilters) {
				t.Errorf("expected %d initial filters, got %d", len(tt.initialFilters), len(chain.GetFilters()))
			}

			// Clear the chain
			chain.Clear()

			// Verify chain is empty
			if len(chain.GetFilters()) != tt.expectedCount {
				t.Errorf("expected %d filters after clear, got %d", tt.expectedCount, len(chain.GetFilters()))
			}

			// Verify metrics are also cleared
			metrics := chain.GetMetrics()
			if len(metrics.FilterExecutions) != 0 {
				t.Errorf("expected 0 filter executions after clear, got %d", len(metrics.FilterExecutions))
			}
			if len(metrics.FilterMatches) != 0 {
				t.Errorf("expected 0 filter matches after clear, got %d", len(metrics.FilterMatches))
			}
		})
	}
}

// TestFilterChain_ClearWithMetrics tests that clearing also resets metrics
func TestFilterChain_ClearWithMetrics(t *testing.T) {
	chain := NewFilterChain()
	filter1 := NewEventTypeFilter("filter-1", []SagaEventType{EventTypeSagaCompleted}, nil)
	filter2 := NewSagaIDFilter("filter-2", []string{"saga-1"}, nil)

	_ = chain.AddFilter(filter1)
	_ = chain.AddFilter(filter2)

	// Process some events to generate metrics
	ctx := context.Background()
	handlerCtx := &EventHandlerContext{MessageID: "msg-1"}
	event := &saga.SagaEvent{
		Type:      EventTypeSagaCompleted,
		SagaID:    "saga-1",
		Timestamp: time.Now(),
	}

	_, _ = chain.ShouldFilter(ctx, event, handlerCtx)

	// Verify metrics exist before clear
	metricsBefore := chain.GetMetrics()
	if metricsBefore.TotalEventsFiltered == 0 {
		t.Errorf("expected metrics to have events before clear")
	}
	if len(metricsBefore.FilterExecutions) == 0 {
		t.Errorf("expected filter executions to exist before clear")
	}

	// Clear the chain
	chain.Clear()

	// Verify filter-specific metrics are reset (TotalEventsFiltered is preserved)
	metricsAfter := chain.GetMetrics()
	// Note: TotalEventsFiltered is not reset by Clear() implementation - this is intentional
	// to maintain cumulative metrics about the chain's lifetime
	if len(metricsAfter.FilterExecutions) != 0 {
		t.Errorf("expected FilterExecutions to be empty after clear, got %d entries", len(metricsAfter.FilterExecutions))
	}
	if len(metricsAfter.FilterMatches) != 0 {
		t.Errorf("expected FilterMatches to be empty after clear, got %d entries", len(metricsAfter.FilterMatches))
	}
}

// TestEventTypeFilter_SetPriority tests setting priority on EventTypeFilter
func TestEventTypeFilter_SetPriority(t *testing.T) {
	tests := []struct {
		name             string
		filter           *EventTypeFilter
		newPriority      int
		expectedPriority int
	}{
		{
			name:             "set priority to positive value",
			filter:           NewEventTypeFilter("test-filter", []SagaEventType{EventTypeSagaCompleted}, nil),
			newPriority:      150,
			expectedPriority: 150,
		},
		{
			name:             "set priority to zero",
			filter:           NewEventTypeFilter("test-filter", []SagaEventType{EventTypeSagaCompleted}, nil),
			newPriority:      0,
			expectedPriority: 0,
		},
		{
			name:             "set priority to negative value",
			filter:           NewEventTypeFilter("test-filter", []SagaEventType{EventTypeSagaCompleted}, nil),
			newPriority:      -50,
			expectedPriority: -50,
		},
		{
			name:             "set priority to maximum int",
			filter:           NewEventTypeFilter("test-filter", []SagaEventType{EventTypeSagaCompleted}, nil),
			newPriority:      2147483647,
			expectedPriority: 2147483647,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify initial priority
			initialPriority := tt.filter.GetPriority()
			if initialPriority == 0 {
				t.Logf("initial priority was %d", initialPriority)
			}

			// Set new priority
			tt.filter.SetPriority(tt.newPriority)

			// Verify priority was set
			if tt.filter.GetPriority() != tt.expectedPriority {
				t.Errorf("expected priority %d, got %d", tt.expectedPriority, tt.filter.GetPriority())
			}
		})
	}
}

// TestSagaIDFilter_SetPriority tests setting priority on SagaIDFilter
func TestSagaIDFilter_SetPriority(t *testing.T) {
	tests := []struct {
		name             string
		filter           *SagaIDFilter
		newPriority      int
		expectedPriority int
	}{
		{
			name:             "set priority to positive value",
			filter:           NewSagaIDFilter("test-filter", []string{"saga-1"}, nil),
			newPriority:      120,
			expectedPriority: 120,
		},
		{
			name:             "set priority to zero",
			filter:           NewSagaIDFilter("test-filter", []string{"saga-1"}, nil),
			newPriority:      0,
			expectedPriority: 0,
		},
		{
			name:             "set priority to negative value",
			filter:           NewSagaIDFilter("test-filter", []string{"saga-1"}, nil),
			newPriority:      -10,
			expectedPriority: -10,
		},
		{
			name:             "set priority to large value",
			filter:           NewSagaIDFilter("test-filter", []string{"saga-1"}, nil),
			newPriority:      1000,
			expectedPriority: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set new priority
			tt.filter.SetPriority(tt.newPriority)

			// Verify priority was set
			if tt.filter.GetPriority() != tt.expectedPriority {
				t.Errorf("expected priority %d, got %d", tt.expectedPriority, tt.filter.GetPriority())
			}
		})
	}
}

// TestCustomFilter_SetPriority tests setting priority on CustomFilter
func TestCustomFilter_SetPriority(t *testing.T) {
	customFn := func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
		return false, nil
	}

	tests := []struct {
		name             string
		filter           *CustomFilter
		newPriority      int
		expectedPriority int
	}{
		{
			name:             "set priority to positive value",
			filter:           NewCustomFilter("test-filter", 50, customFn),
			newPriority:      200,
			expectedPriority: 200,
		},
		{
			name:             "set priority to zero",
			filter:           NewCustomFilter("test-filter", 50, customFn),
			newPriority:      0,
			expectedPriority: 0,
		},
		{
			name:             "set priority to negative value",
			filter:           NewCustomFilter("test-filter", 50, customFn),
			newPriority:      -100,
			expectedPriority: -100,
		},
		{
			name:             "set priority to maximum value",
			filter:           NewCustomFilter("test-filter", 50, customFn),
			newPriority:      999999,
			expectedPriority: 999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set new priority
			tt.filter.SetPriority(tt.newPriority)

			// Verify priority was set
			if tt.filter.GetPriority() != tt.expectedPriority {
				t.Errorf("expected priority %d, got %d", tt.expectedPriority, tt.filter.GetPriority())
			}
		})
	}
}

// TestCompositeFilter_SetPriority tests setting priority on CompositeFilter
func TestCompositeFilter_SetPriority(t *testing.T) {
	typeFilter := NewEventTypeFilter("type-filter", []SagaEventType{EventTypeSagaCompleted}, nil)
	idFilter := NewSagaIDFilter("id-filter", []string{"saga-1"}, nil)

	tests := []struct {
		name             string
		filter           *CompositeFilter
		newPriority      int
		expectedPriority int
	}{
		{
			name:             "set priority to positive value",
			filter:           NewCompositeFilter("composite-filter", 70, []EventFilter{typeFilter, idFilter}, FilterOperatorAND),
			newPriority:      180,
			expectedPriority: 180,
		},
		{
			name:             "set priority to zero",
			filter:           NewCompositeFilter("composite-filter", 70, []EventFilter{typeFilter, idFilter}, FilterOperatorAND),
			newPriority:      0,
			expectedPriority: 0,
		},
		{
			name:             "set priority to negative value",
			filter:           NewCompositeFilter("composite-filter", 70, []EventFilter{typeFilter, idFilter}, FilterOperatorAND),
			newPriority:      -20,
			expectedPriority: -20,
		},
		{
			name:             "set priority on empty composite filter",
			filter:           NewCompositeFilter("empty-composite", 70, []EventFilter{}, FilterOperatorOR),
			newPriority:      250,
			expectedPriority: 250,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set new priority
			tt.filter.SetPriority(tt.newPriority)

			// Verify priority was set
			if tt.filter.GetPriority() != tt.expectedPriority {
				t.Errorf("expected priority %d, got %d", tt.expectedPriority, tt.filter.GetPriority())
			}
		})
	}
}

// TestMetadataFilter_GetName_GetPriority tests GetName and GetPriority methods on MetadataFilter
func TestMetadataFilter_GetName_GetPriority(t *testing.T) {
	tests := []struct {
		name             string
		filterName       string
		expectedName     string
		expectedPriority int
	}{
		{
			name:             "filter with simple name",
			filterName:       "metadata-filter",
			expectedName:     "metadata-filter",
			expectedPriority: 80, // Default priority from implementation
		},
		{
			name:             "filter with complex name",
			filterName:       "complex-metadata-filter-v1",
			expectedName:     "complex-metadata-filter-v1",
			expectedPriority: 80,
		},
		{
			name:             "filter with empty name",
			filterName:       "",
			expectedName:     "",
			expectedPriority: 80,
		},
		{
			name:             "filter with special characters",
			filterName:       "metadata-filter_test-123",
			expectedName:     "metadata-filter_test-123",
			expectedPriority: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := []MetadataRule{
				{
					Key:           "environment",
					MatchType:     MetadataMatchEquals,
					ExpectedValue: "production",
				},
			}

			filter, err := NewMetadataFilter(tt.filterName, rules)
			if err != nil {
				t.Fatalf("unexpected error creating filter: %v", err)
			}

			// Test GetName
			if filter.GetName() != tt.expectedName {
				t.Errorf("expected name %s, got %s", tt.expectedName, filter.GetName())
			}

			// Test GetPriority
			if filter.GetPriority() != tt.expectedPriority {
				t.Errorf("expected priority %d, got %d", tt.expectedPriority, filter.GetPriority())
			}
		})
	}
}

// TestCompositeFilter_GetName_GetPriority tests GetName and GetPriority methods on CompositeFilter
func TestCompositeFilter_GetName_GetPriority(t *testing.T) {
	typeFilter := NewEventTypeFilter("type-filter", []SagaEventType{EventTypeSagaCompleted}, nil)
	idFilter := NewSagaIDFilter("id-filter", []string{"saga-1"}, nil)

	tests := []struct {
		name             string
		filterName       string
		initialPriority  int
		expectedName     string
		expectedPriority int
	}{
		{
			name:             "composite filter with simple name",
			filterName:       "composite-filter",
			initialPriority:  75,
			expectedName:     "composite-filter",
			expectedPriority: 75,
		},
		{
			name:             "composite filter with AND operator",
			filterName:       "and-composite",
			initialPriority:  100,
			expectedName:     "and-composite",
			expectedPriority: 100,
		},
		{
			name:             "composite filter with OR operator",
			filterName:       "or-composite",
			initialPriority:  50,
			expectedName:     "or-composite",
			expectedPriority: 50,
		},
		{
			name:             "empty composite filter",
			filterName:       "empty-composite",
			initialPriority:  25,
			expectedName:     "empty-composite",
			expectedPriority: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filter *CompositeFilter
			if tt.name == "empty composite filter" {
				filter = NewCompositeFilter(tt.filterName, tt.initialPriority, []EventFilter{}, FilterOperatorOR)
			} else {
				filter = NewCompositeFilter(tt.filterName, tt.initialPriority, []EventFilter{typeFilter, idFilter}, FilterOperatorAND)
			}

			// Test GetName
			if filter.GetName() != tt.expectedName {
				t.Errorf("expected name %s, got %s", tt.expectedName, filter.GetName())
			}

			// Test GetPriority
			if filter.GetPriority() != tt.expectedPriority {
				t.Errorf("expected priority %d, got %d", tt.expectedPriority, filter.GetPriority())
			}
		})
	}
}

// TestFilter_SetPriorityIntegration tests that setting priorities works correctly at the integration level
func TestFilter_SetPriorityIntegration(t *testing.T) {
	chain := NewFilterChain()

	// Create filters with different priorities
	filter1 := NewEventTypeFilter("type-filter", []SagaEventType{EventTypeSagaCompleted}, nil)
	filter2 := NewSagaIDFilter("id-filter", []string{"saga-1"}, nil)

	// Test initial priorities
	if filter1.GetPriority() != 100 {
		t.Errorf("expected EventTypeFilter initial priority 100, got %d", filter1.GetPriority())
	}
	if filter2.GetPriority() != 90 {
		t.Errorf("expected SagaIDFilter initial priority 90, got %d", filter2.GetPriority())
	}

	// Add filters to chain
	_ = chain.AddFilter(filter1)
	_ = chain.AddFilter(filter2)

	// Verify they are in the chain
	filters := chain.GetFilters()
	if len(filters) != 2 {
		t.Errorf("expected 2 filters in chain, got %d", len(filters))
	}

	// Change priorities
	filter1.SetPriority(50)
	filter2.SetPriority(150)

	// Verify priorities were changed
	if filter1.GetPriority() != 50 {
		t.Errorf("expected EventTypeFilter new priority 50, got %d", filter1.GetPriority())
	}
	if filter2.GetPriority() != 150 {
		t.Errorf("expected SagaIDFilter new priority 150, got %d", filter2.GetPriority())
	}

	// Verify that the filters are still in the chain after priority change
	filters = chain.GetFilters()
	if len(filters) != 2 {
		t.Errorf("expected 2 filters in chain after priority change, got %d", len(filters))
	}
}

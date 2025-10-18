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

// Package messaging provides message filtering capabilities for Saga events.
// This file implements a flexible filter chain pattern for fine-grained event filtering.
package messaging

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// EventFilter defines the interface for event filtering.
// Filters can inspect events and decide whether they should be processed.
//
// Example usage:
//
//	filter := NewEventTypeFilter([]SagaEventType{EventTypeSagaCompleted})
//	if filter.ShouldFilter(ctx, event, handlerCtx) {
//	    // Event should be filtered out
//	}
type EventFilter interface {
	// ShouldFilter returns true if the event should be filtered out (not processed).
	//
	// Parameters:
	//   - ctx: Context for the filter operation
	//   - event: The event to filter
	//   - handlerCtx: Additional context about the event
	//
	// Returns:
	//   - bool: true if the event should be filtered out
	//   - error: Error if filtering fails
	ShouldFilter(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error)

	// GetName returns the name of this filter.
	GetName() string

	// GetPriority returns the filter's priority.
	// Higher priority filters are evaluated first.
	GetPriority() int
}

// FilterChain manages a chain of filters and applies them in priority order.
// The chain follows the "fail-fast" principle: if any filter decides to filter out
// the event, processing stops immediately.
//
// Example usage:
//
//	chain := NewFilterChain()
//	chain.AddFilter(typeFilter)
//	chain.AddFilter(sagaIDFilter)
//
//	shouldFilter, err := chain.ShouldFilter(ctx, event, handlerCtx)
type FilterChain interface {
	// AddFilter adds a filter to the chain.
	AddFilter(filter EventFilter) error

	// RemoveFilter removes a filter by name.
	RemoveFilter(name string) error

	// ShouldFilter applies all filters in the chain.
	// Returns true if any filter decides to filter the event.
	ShouldFilter(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error)

	// GetFilters returns all filters in the chain.
	GetFilters() []EventFilter

	// Clear removes all filters from the chain.
	Clear()

	// GetMetrics returns metrics about filter operations.
	GetMetrics() *FilterChainMetrics
}

// FilterChainMetrics contains metrics about filter chain operations.
type FilterChainMetrics struct {
	// TotalEventsFiltered is the total number of events checked by the filter chain.
	TotalEventsFiltered int64

	// EventsPassedThrough is the number of events that passed all filters.
	EventsPassedThrough int64

	// EventsFilteredOut is the number of events that were filtered out.
	EventsFilteredOut int64

	// FilterExecutions tracks how many times each filter was executed.
	FilterExecutions map[string]int64

	// FilterMatches tracks how many times each filter filtered an event.
	FilterMatches map[string]int64

	// LastFilteredAt is when the last event was filtered.
	LastFilteredAt time.Time
}

// defaultFilterChain is the default implementation of FilterChain.
type defaultFilterChain struct {
	filters []EventFilter
	metrics *FilterChainMetrics
	mu      sync.RWMutex
}

// NewFilterChain creates a new filter chain.
func NewFilterChain() FilterChain {
	return &defaultFilterChain{
		filters: make([]EventFilter, 0),
		metrics: &FilterChainMetrics{
			FilterExecutions: make(map[string]int64),
			FilterMatches:    make(map[string]int64),
		},
	}
}

// AddFilter adds a filter to the chain.
func (fc *defaultFilterChain) AddFilter(filter EventFilter) error {
	if filter == nil {
		return fmt.Errorf("filter cannot be nil")
	}

	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Check for duplicate filters
	for _, f := range fc.filters {
		if f.GetName() == filter.GetName() {
			return fmt.Errorf("filter with name %s already exists", filter.GetName())
		}
	}

	fc.filters = append(fc.filters, filter)

	// Sort by priority (highest first)
	fc.sortFilters()

	// Initialize metrics
	fc.metrics.FilterExecutions[filter.GetName()] = 0
	fc.metrics.FilterMatches[filter.GetName()] = 0

	return nil
}

// RemoveFilter removes a filter by name.
func (fc *defaultFilterChain) RemoveFilter(name string) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	for i, filter := range fc.filters {
		if filter.GetName() == name {
			fc.filters = append(fc.filters[:i], fc.filters[i+1:]...)
			delete(fc.metrics.FilterExecutions, name)
			delete(fc.metrics.FilterMatches, name)
			return nil
		}
	}

	return fmt.Errorf("filter %s not found", name)
}

// ShouldFilter applies all filters in the chain.
func (fc *defaultFilterChain) ShouldFilter(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
	fc.mu.Lock()
	fc.metrics.TotalEventsFiltered++
	filters := make([]EventFilter, len(fc.filters))
	copy(filters, fc.filters)
	fc.mu.Unlock()

	// Apply each filter in priority order
	for _, filter := range filters {
		fc.mu.Lock()
		fc.metrics.FilterExecutions[filter.GetName()]++
		fc.mu.Unlock()

		shouldFilter, err := filter.ShouldFilter(ctx, event, handlerCtx)
		if err != nil {
			return false, fmt.Errorf("filter %s failed: %w", filter.GetName(), err)
		}

		if shouldFilter {
			fc.mu.Lock()
			fc.metrics.FilterMatches[filter.GetName()]++
			fc.metrics.EventsFilteredOut++
			fc.metrics.LastFilteredAt = time.Now()
			fc.mu.Unlock()
			return true, nil
		}
	}

	fc.mu.Lock()
	fc.metrics.EventsPassedThrough++
	fc.mu.Unlock()

	return false, nil
}

// GetFilters returns all filters in the chain.
func (fc *defaultFilterChain) GetFilters() []EventFilter {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	result := make([]EventFilter, len(fc.filters))
	copy(result, fc.filters)
	return result
}

// Clear removes all filters from the chain.
func (fc *defaultFilterChain) Clear() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.filters = make([]EventFilter, 0)
	fc.metrics.FilterExecutions = make(map[string]int64)
	fc.metrics.FilterMatches = make(map[string]int64)
}

// GetMetrics returns metrics about filter operations.
func (fc *defaultFilterChain) GetMetrics() *FilterChainMetrics {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	// Deep copy metrics
	metrics := &FilterChainMetrics{
		TotalEventsFiltered: fc.metrics.TotalEventsFiltered,
		EventsPassedThrough: fc.metrics.EventsPassedThrough,
		EventsFilteredOut:   fc.metrics.EventsFilteredOut,
		LastFilteredAt:      fc.metrics.LastFilteredAt,
		FilterExecutions:    make(map[string]int64),
		FilterMatches:       make(map[string]int64),
	}

	for k, v := range fc.metrics.FilterExecutions {
		metrics.FilterExecutions[k] = v
	}
	for k, v := range fc.metrics.FilterMatches {
		metrics.FilterMatches[k] = v
	}

	return metrics
}

// sortFilters sorts filters by priority (highest first).
// Must be called with lock held.
func (fc *defaultFilterChain) sortFilters() {
	// Simple bubble sort is fine for small filter lists
	for i := 0; i < len(fc.filters); i++ {
		for j := i + 1; j < len(fc.filters); j++ {
			if fc.filters[j].GetPriority() > fc.filters[i].GetPriority() {
				fc.filters[i], fc.filters[j] = fc.filters[j], fc.filters[i]
			}
		}
	}
}

// EventTypeFilter filters events based on their type.
type EventTypeFilter struct {
	name          string
	priority      int
	allowedTypes  map[SagaEventType]bool
	excludedTypes map[SagaEventType]bool
}

// NewEventTypeFilter creates a filter that allows only specific event types.
// If allowedTypes is empty, all types are allowed unless explicitly excluded.
func NewEventTypeFilter(name string, allowedTypes []SagaEventType, excludedTypes []SagaEventType) *EventTypeFilter {
	filter := &EventTypeFilter{
		name:          name,
		priority:      100, // High priority
		allowedTypes:  make(map[SagaEventType]bool),
		excludedTypes: make(map[SagaEventType]bool),
	}

	for _, t := range allowedTypes {
		filter.allowedTypes[t] = true
	}
	for _, t := range excludedTypes {
		filter.excludedTypes[t] = true
	}

	return filter
}

// ShouldFilter implements EventFilter.
func (f *EventTypeFilter) ShouldFilter(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
	eventType := SagaEventType(event.Type)

	// Check excluded types first
	if f.excludedTypes[eventType] {
		return true, nil
	}

	// If allowedTypes is not empty, check if event type is in the list
	if len(f.allowedTypes) > 0 {
		return !f.allowedTypes[eventType], nil
	}

	// Allow by default if no allowed types specified
	return false, nil
}

// GetName implements EventFilter.
func (f *EventTypeFilter) GetName() string {
	return f.name
}

// GetPriority implements EventFilter.
func (f *EventTypeFilter) GetPriority() int {
	return f.priority
}

// SetPriority sets the filter priority.
func (f *EventTypeFilter) SetPriority(priority int) {
	f.priority = priority
}

// SagaIDFilter filters events based on Saga ID.
type SagaIDFilter struct {
	name        string
	priority    int
	allowedIDs  map[string]bool
	excludedIDs map[string]bool
}

// NewSagaIDFilter creates a filter for specific Saga IDs.
func NewSagaIDFilter(name string, allowedIDs []string, excludedIDs []string) *SagaIDFilter {
	filter := &SagaIDFilter{
		name:        name,
		priority:    90,
		allowedIDs:  make(map[string]bool),
		excludedIDs: make(map[string]bool),
	}

	for _, id := range allowedIDs {
		filter.allowedIDs[id] = true
	}
	for _, id := range excludedIDs {
		filter.excludedIDs[id] = true
	}

	return filter
}

// ShouldFilter implements EventFilter.
func (f *SagaIDFilter) ShouldFilter(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
	// Check excluded IDs first
	if f.excludedIDs[event.SagaID] {
		return true, nil
	}

	// If allowedIDs is not empty, check if saga ID is in the list
	if len(f.allowedIDs) > 0 {
		return !f.allowedIDs[event.SagaID], nil
	}

	// Allow by default
	return false, nil
}

// GetName implements EventFilter.
func (f *SagaIDFilter) GetName() string {
	return f.name
}

// GetPriority implements EventFilter.
func (f *SagaIDFilter) GetPriority() int {
	return f.priority
}

// SetPriority sets the filter priority.
func (f *SagaIDFilter) SetPriority(priority int) {
	f.priority = priority
}

// MetadataFilter filters events based on metadata patterns.
type MetadataFilter struct {
	name     string
	priority int
	rules    []MetadataRule
}

// MetadataRule defines a rule for matching event metadata.
type MetadataRule struct {
	// Key is the metadata key to match.
	Key string

	// ValuePattern is a regex pattern for matching the value.
	ValuePattern string

	// compiledPattern is the compiled regex.
	compiledPattern *regexp.Regexp

	// MatchType determines how to match (equals, contains, regex).
	MatchType MetadataMatchType

	// ExpectedValue is the expected value for equals match.
	ExpectedValue string
}

// MetadataMatchType defines how metadata values should be matched.
type MetadataMatchType string

const (
	// MetadataMatchEquals matches exact values.
	MetadataMatchEquals MetadataMatchType = "equals"

	// MetadataMatchContains matches if the value contains the pattern.
	MetadataMatchContains MetadataMatchType = "contains"

	// MetadataMatchRegex matches using regular expressions.
	MetadataMatchRegex MetadataMatchType = "regex"
)

// NewMetadataFilter creates a filter based on event metadata.
func NewMetadataFilter(name string, rules []MetadataRule) (*MetadataFilter, error) {
	filter := &MetadataFilter{
		name:     name,
		priority: 80,
		rules:    rules,
	}

	// Compile regex patterns
	for i := range filter.rules {
		if filter.rules[i].MatchType == MetadataMatchRegex {
			pattern, err := regexp.Compile(filter.rules[i].ValuePattern)
			if err != nil {
				return nil, fmt.Errorf("invalid regex pattern %s: %w", filter.rules[i].ValuePattern, err)
			}
			filter.rules[i].compiledPattern = pattern
		}
	}

	return filter, nil
}

// ShouldFilter implements EventFilter.
func (f *MetadataFilter) ShouldFilter(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
	// If any rule matches, filter the event
	for _, rule := range f.rules {
		if f.matchesRule(event, rule) {
			return true, nil
		}
	}
	return false, nil
}

// matchesRule checks if an event matches a metadata rule.
func (f *MetadataFilter) matchesRule(event *saga.SagaEvent, rule MetadataRule) bool {
	// Get metadata value
	value, exists := event.Metadata[rule.Key]
	if !exists {
		return false
	}

	valueStr, ok := value.(string)
	if !ok {
		return false
	}

	// Match based on type
	switch rule.MatchType {
	case MetadataMatchEquals:
		return valueStr == rule.ExpectedValue

	case MetadataMatchContains:
		return regexp.MustCompile(rule.ValuePattern).MatchString(valueStr)

	case MetadataMatchRegex:
		if rule.compiledPattern != nil {
			return rule.compiledPattern.MatchString(valueStr)
		}
		return false

	default:
		return false
	}
}

// GetName implements EventFilter.
func (f *MetadataFilter) GetName() string {
	return f.name
}

// GetPriority implements EventFilter.
func (f *MetadataFilter) GetPriority() int {
	return f.priority
}

// SetPriority sets the filter priority.
func (f *MetadataFilter) SetPriority(priority int) {
	f.priority = priority
}

// CustomFilter wraps a custom filtering function.
type CustomFilter struct {
	name     string
	priority int
	filterFn func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error)
}

// NewCustomFilter creates a filter with a custom function.
func NewCustomFilter(name string, priority int, filterFn func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error)) *CustomFilter {
	return &CustomFilter{
		name:     name,
		priority: priority,
		filterFn: filterFn,
	}
}

// ShouldFilter implements EventFilter.
func (f *CustomFilter) ShouldFilter(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
	if f.filterFn == nil {
		return false, nil
	}
	return f.filterFn(ctx, event, handlerCtx)
}

// GetName implements EventFilter.
func (f *CustomFilter) GetName() string {
	return f.name
}

// GetPriority implements EventFilter.
func (f *CustomFilter) GetPriority() int {
	return f.priority
}

// SetPriority sets the filter priority.
func (f *CustomFilter) SetPriority(priority int) {
	f.priority = priority
}

// CompositeFilter combines multiple filters with AND or OR logic.
type CompositeFilter struct {
	name     string
	priority int
	filters  []EventFilter
	operator FilterOperator
}

// FilterOperator defines how multiple filters are combined.
type FilterOperator string

const (
	// FilterOperatorAND requires all filters to agree to filter.
	FilterOperatorAND FilterOperator = "AND"

	// FilterOperatorOR requires any filter to agree to filter.
	FilterOperatorOR FilterOperator = "OR"
)

// NewCompositeFilter creates a filter that combines multiple filters.
func NewCompositeFilter(name string, priority int, filters []EventFilter, operator FilterOperator) *CompositeFilter {
	return &CompositeFilter{
		name:     name,
		priority: priority,
		filters:  filters,
		operator: operator,
	}
}

// ShouldFilter implements EventFilter.
func (f *CompositeFilter) ShouldFilter(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) (bool, error) {
	if len(f.filters) == 0 {
		return false, nil
	}

	switch f.operator {
	case FilterOperatorAND:
		// All filters must agree to filter
		for _, filter := range f.filters {
			shouldFilter, err := filter.ShouldFilter(ctx, event, handlerCtx)
			if err != nil {
				return false, err
			}
			if !shouldFilter {
				return false, nil
			}
		}
		return true, nil

	case FilterOperatorOR:
		// Any filter can decide to filter
		for _, filter := range f.filters {
			shouldFilter, err := filter.ShouldFilter(ctx, event, handlerCtx)
			if err != nil {
				return false, err
			}
			if shouldFilter {
				return true, nil
			}
		}
		return false, nil

	default:
		return false, fmt.Errorf("unknown filter operator: %s", f.operator)
	}
}

// GetName implements EventFilter.
func (f *CompositeFilter) GetName() string {
	return f.name
}

// GetPriority implements EventFilter.
func (f *CompositeFilter) GetPriority() int {
	return f.priority
}

// SetPriority sets the filter priority.
func (f *CompositeFilter) SetPriority(priority int) {
	f.priority = priority
}


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
	"fmt"
	"sort"
	"sync"

	"github.com/innovationmech/swit/pkg/saga"
)

// ExtendedEventRouter extends the EventRouter interface with additional
// routing capabilities including handler management, routing rules, and metrics.
//
// This interface provides more advanced routing features beyond the basic
// EventRouter interface defined in handler.go.
//
// Example usage:
//
//	router := NewDefaultEventRouter()
//	router.RegisterHandler(handler1)
//	router.RegisterHandler(handler2)
//
//	err := router.RouteEvent(ctx, event, handlerCtx)
//	if err != nil {
//	    log.Error("Failed to route event", zap.Error(err))
//	}
type ExtendedEventRouter interface {
	EventRouter

	// RegisterHandler registers a Saga event handler with the router.
	// Handlers are stored in priority order for efficient routing.
	//
	// Parameters:
	//   - handler: The event handler to register
	//
	// Returns:
	//   - error: Registration error if the handler is invalid
	RegisterHandler(handler SagaEventHandler) error

	// UnregisterHandler removes a handler from the router.
	//
	// Parameters:
	//   - handlerID: Unique identifier of the handler to remove
	//
	// Returns:
	//   - error: Error if the handler is not found
	UnregisterHandler(handlerID string) error

	// FindHandlers returns all handlers that can process the given event.
	// Handlers are returned in priority order (highest priority first).
	//
	// Parameters:
	//   - event: The event to match handlers for
	//
	// Returns:
	//   - []SagaEventHandler: List of matching handlers
	FindHandlers(event *saga.SagaEvent) []SagaEventHandler

	// GetHandlerCount returns the total number of registered handlers.
	//
	// Returns:
	//   - int: Number of registered handlers
	GetHandlerCount() int

	// GetHandlerByID retrieves a handler by its unique identifier.
	//
	// Parameters:
	//   - handlerID: Unique identifier of the handler
	//
	// Returns:
	//   - SagaEventHandler: The handler if found
	//   - error: Error if handler not found
	GetHandlerByID(handlerID string) (SagaEventHandler, error)

	// GetHandlersByEventType returns all handlers that support a specific event type.
	//
	// Parameters:
	//   - eventType: The event type to match
	//
	// Returns:
	//   - []SagaEventHandler: List of handlers supporting the event type
	GetHandlersByEventType(eventType SagaEventType) []SagaEventHandler

	// AddRoutingRule adds a custom routing rule to the router.
	// Routing rules allow fine-grained control over event routing.
	//
	// Parameters:
	//   - rule: The routing rule to add
	//
	// Returns:
	//   - error: Error if the rule is invalid
	AddRoutingRule(rule RoutingRule) error

	// RemoveRoutingRule removes a routing rule by its name.
	//
	// Parameters:
	//   - ruleName: Name of the rule to remove
	//
	// Returns:
	//   - error: Error if the rule is not found
	RemoveRoutingRule(ruleName string) error

	// GetRoutingRules returns all configured routing rules.
	//
	// Returns:
	//   - []RoutingRule: List of routing rules
	GetRoutingRules() []RoutingRule

	// GetRouterMetrics returns routing metrics for monitoring.
	//
	// Returns:
	//   - *RoutingMetrics: Current routing metrics
	GetRouterMetrics() *RoutingMetrics
}

// RoutingRule defines a rule for custom event routing logic.
// Rules can be used to implement complex routing scenarios based on event content,
// metadata, or external conditions.
type RoutingRule interface {
	// GetName returns the unique name of the routing rule.
	GetName() string

	// Matches determines if this rule applies to the given event.
	//
	// Parameters:
	//   - event: The event to check
	//
	// Returns:
	//   - bool: true if the rule matches the event
	Matches(event *saga.SagaEvent) bool

	// SelectHandlers selects which handlers should process the event.
	// This allows rules to override the default handler selection logic.
	//
	// Parameters:
	//   - event: The event to route
	//   - availableHandlers: All handlers that can potentially process the event
	//
	// Returns:
	//   - []SagaEventHandler: Selected handlers (may be a subset of available handlers)
	SelectHandlers(event *saga.SagaEvent, availableHandlers []SagaEventHandler) []SagaEventHandler

	// GetPriority returns the priority of this rule.
	// Higher priority rules are evaluated first.
	//
	// Returns:
	//   - int: Rule priority
	GetPriority() int
}

// RoutingMetrics contains metrics about routing operations.
type RoutingMetrics struct {
	// TotalEventsRouted is the total number of events routed
	TotalEventsRouted int64 `json:"total_events_routed"`

	// TotalRoutingErrors is the total number of routing errors
	TotalRoutingErrors int64 `json:"total_routing_errors"`

	// EventsRoutedByType tracks routing counts by event type
	EventsRoutedByType map[SagaEventType]int64 `json:"events_routed_by_type"`

	// HandlerInvocations tracks how many times each handler was invoked
	HandlerInvocations map[string]int64 `json:"handler_invocations"`

	// UnroutedEvents tracks events that couldn't be routed
	UnroutedEvents int64 `json:"unrouted_events"`

	// RuleMatches tracks how many times each rule matched
	RuleMatches map[string]int64 `json:"rule_matches"`
}

// defaultEventRouter is the default implementation of EventRouter and ExtendedEventRouter.
type defaultEventRouter struct {
	// handlers stores registered handlers by their ID
	handlers map[string]SagaEventHandler

	// handlersByType stores handlers grouped by supported event type
	handlersByType map[SagaEventType][]SagaEventHandler

	// sortedHandlers maintains handlers in priority order
	sortedHandlers []SagaEventHandler

	// routingRules stores custom routing rules
	routingRules []RoutingRule

	// metrics tracks routing statistics
	metrics *RoutingMetrics

	// continueOnError determines if routing should continue after a handler error
	continueOnError bool

	// routingStrategy defines how events are routed to handlers
	routingStrategy RoutingStrategy

	// mu protects concurrent access
	mu sync.RWMutex
}

// RouterOption is a functional option for configuring the event router.
type RouterOption func(*defaultEventRouter)

// WithContinueOnError configures the router to continue routing to other handlers
// even if a handler returns an error.
func WithContinueOnError(continueOnError bool) RouterOption {
	return func(r *defaultEventRouter) {
		r.continueOnError = continueOnError
	}
}

// WithRoutingStrategy configures the routing strategy for the router.
func WithRoutingStrategy(strategy RoutingStrategy) RouterOption {
	return func(r *defaultEventRouter) {
		r.routingStrategy = strategy
	}
}

// NewDefaultEventRouter creates a new event router with default configuration.
//
// Parameters:
//   - opts: Optional configuration options
//
// Returns:
//   - ExtendedEventRouter: Configured router instance
func NewDefaultEventRouter(opts ...RouterOption) ExtendedEventRouter {
	router := &defaultEventRouter{
		handlers:        make(map[string]SagaEventHandler),
		handlersByType:  make(map[SagaEventType][]SagaEventHandler),
		sortedHandlers:  make([]SagaEventHandler, 0),
		routingRules:    make([]RoutingRule, 0),
		routingStrategy: RoutingStrategyAll, // Default strategy
		metrics: &RoutingMetrics{
			EventsRoutedByType: make(map[SagaEventType]int64),
			HandlerInvocations: make(map[string]int64),
			RuleMatches:        make(map[string]int64),
		},
		continueOnError: false,
	}

	// Apply options
	for _, opt := range opts {
		opt(router)
	}

	return router
}

// RegisterHandler implements EventRouter interface.
func (r *defaultEventRouter) RegisterHandler(handler SagaEventHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	config := handler.GetConfiguration()
	if config == nil {
		return fmt.Errorf("handler configuration is nil")
	}
	if config.HandlerID == "" {
		return fmt.Errorf("handler ID is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if handler already registered
	if _, exists := r.handlers[config.HandlerID]; exists {
		return fmt.Errorf("handler with ID %s already registered", config.HandlerID)
	}

	// Register handler
	r.handlers[config.HandlerID] = handler

	// Group by event type
	for _, eventType := range handler.GetSupportedEventTypes() {
		r.handlersByType[eventType] = append(r.handlersByType[eventType], handler)
	}

	// Rebuild sorted handlers list
	r.rebuildSortedHandlers()

	// Initialize metrics
	r.metrics.HandlerInvocations[config.HandlerID] = 0

	return nil
}

// UnregisterHandler implements EventRouter interface.
func (r *defaultEventRouter) UnregisterHandler(handlerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	handler, exists := r.handlers[handlerID]
	if !exists {
		return fmt.Errorf("handler %s not found", handlerID)
	}

	// Remove from main handlers map
	delete(r.handlers, handlerID)

	// Remove from event type groups
	for _, eventType := range handler.GetSupportedEventTypes() {
		handlers := r.handlersByType[eventType]
		for i, h := range handlers {
			if h.GetConfiguration().HandlerID == handlerID {
				r.handlersByType[eventType] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
		// Clean up empty slices
		if len(r.handlersByType[eventType]) == 0 {
			delete(r.handlersByType, eventType)
		}
	}

	// Rebuild sorted handlers list
	r.rebuildSortedHandlers()

	// Remove from metrics
	delete(r.metrics.HandlerInvocations, handlerID)

	return nil
}

// Route implements EventRouter interface.
func (r *defaultEventRouter) Route(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) error {
	if event == nil {
		return ErrInvalidEvent
	}
	if handlerCtx == nil {
		return ErrInvalidContext
	}

	r.mu.Lock()
	r.metrics.TotalEventsRouted++
	r.metrics.EventsRoutedByType[event.Type]++
	r.mu.Unlock()

	// Find matching handlers
	handlers := r.FindHandlers(event)
	if len(handlers) == 0 {
		r.mu.Lock()
		r.metrics.UnroutedEvents++
		r.mu.Unlock()
		return fmt.Errorf("no handler found for event type: %s", event.Type)
	}

	// Apply routing rules to filter/reorder handlers
	handlers = r.applyRoutingRules(event, handlers)

	// Route to all matching handlers
	var lastError error
	handledSuccessfully := false

	for _, handler := range handlers {
		// Update metrics
		config := handler.GetConfiguration()
		r.mu.Lock()
		r.metrics.HandlerInvocations[config.HandlerID]++
		r.mu.Unlock()

		// Invoke handler
		err := handler.HandleSagaEvent(ctx, event, handlerCtx)
		if err != nil {
			lastError = err
			r.mu.Lock()
			r.metrics.TotalRoutingErrors++
			r.mu.Unlock()

			// Call failure callback
			handler.OnEventFailed(ctx, event, err)

			// Stop routing if continueOnError is false
			if !r.continueOnError {
				return fmt.Errorf("handler %s failed to process event: %w", config.HandlerID, err)
			}
		} else {
			handledSuccessfully = true

			// Call success callback
			handler.OnEventProcessed(ctx, event, nil)
		}
	}

	// If continueOnError is true and no handler succeeded, return the last error
	if r.continueOnError && !handledSuccessfully && lastError != nil {
		return fmt.Errorf("all handlers failed to process event: %w", lastError)
	}

	return nil
}

// FindHandlers implements EventRouter interface.
func (r *defaultEventRouter) FindHandlers(event *saga.SagaEvent) []SagaEventHandler {
	if event == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get handlers by event type
	handlers := r.handlersByType[event.Type]
	if len(handlers) == 0 {
		return nil
	}

	// Filter handlers using CanHandle method
	matchingHandlers := make([]SagaEventHandler, 0, len(handlers))
	for _, handler := range handlers {
		if handler.CanHandle(event) {
			matchingHandlers = append(matchingHandlers, handler)
		}
	}

	// Sort by priority (highest first)
	sort.Slice(matchingHandlers, func(i, j int) bool {
		return matchingHandlers[i].GetPriority() > matchingHandlers[j].GetPriority()
	})

	return matchingHandlers
}

// GetHandlerCount implements EventRouter interface.
func (r *defaultEventRouter) GetHandlerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.handlers)
}

// GetHandlerByID implements EventRouter interface.
func (r *defaultEventRouter) GetHandlerByID(handlerID string) (SagaEventHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[handlerID]
	if !exists {
		return nil, fmt.Errorf("handler %s not found", handlerID)
	}

	return handler, nil
}

// GetHandlersByEventType implements EventRouter interface.
func (r *defaultEventRouter) GetHandlersByEventType(eventType SagaEventType) []SagaEventHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlers := r.handlersByType[eventType]
	result := make([]SagaEventHandler, len(handlers))
	copy(result, handlers)

	return result
}

// AddRoutingRule implements EventRouter interface.
func (r *defaultEventRouter) AddRoutingRule(rule RoutingRule) error {
	if rule == nil {
		return fmt.Errorf("routing rule cannot be nil")
	}
	if rule.GetName() == "" {
		return fmt.Errorf("routing rule name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate rule names
	for _, existingRule := range r.routingRules {
		if existingRule.GetName() == rule.GetName() {
			return fmt.Errorf("routing rule with name %s already exists", rule.GetName())
		}
	}

	// Add rule
	r.routingRules = append(r.routingRules, rule)

	// Sort rules by priority
	sort.Slice(r.routingRules, func(i, j int) bool {
		return r.routingRules[i].GetPriority() > r.routingRules[j].GetPriority()
	})

	// Initialize metrics
	r.metrics.RuleMatches[rule.GetName()] = 0

	return nil
}

// RemoveRoutingRule implements EventRouter interface.
func (r *defaultEventRouter) RemoveRoutingRule(ruleName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, rule := range r.routingRules {
		if rule.GetName() == ruleName {
			r.routingRules = append(r.routingRules[:i], r.routingRules[i+1:]...)
			delete(r.metrics.RuleMatches, ruleName)
			return nil
		}
	}

	return fmt.Errorf("routing rule %s not found", ruleName)
}

// GetRoutingRules implements EventRouter interface.
func (r *defaultEventRouter) GetRoutingRules() []RoutingRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rules := make([]RoutingRule, len(r.routingRules))
	copy(rules, r.routingRules)
	return rules
}

// RouteEvent implements EventRouter interface.
// This is an alias for the Route method to satisfy the EventRouter interface.
func (r *defaultEventRouter) RouteEvent(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) error {
	return r.Route(ctx, event, handlerCtx)
}

// GetRoutingStrategy implements EventRouter interface.
func (r *defaultEventRouter) GetRoutingStrategy() RoutingStrategy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.routingStrategy
}

// SetRoutingStrategy implements EventRouter interface.
func (r *defaultEventRouter) SetRoutingStrategy(strategy RoutingStrategy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routingStrategy = strategy
}

// GetRouterMetrics implements ExtendedEventRouter interface.
func (r *defaultEventRouter) GetRouterMetrics() *RoutingMetrics {
	return r.getMetrics()
}

// getMetrics is an internal method for getting metrics.
func (r *defaultEventRouter) getMetrics() *RoutingMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a deep copy
	metrics := &RoutingMetrics{
		TotalEventsRouted:  r.metrics.TotalEventsRouted,
		TotalRoutingErrors: r.metrics.TotalRoutingErrors,
		UnroutedEvents:     r.metrics.UnroutedEvents,
		EventsRoutedByType: make(map[SagaEventType]int64),
		HandlerInvocations: make(map[string]int64),
		RuleMatches:        make(map[string]int64),
	}

	for k, v := range r.metrics.EventsRoutedByType {
		metrics.EventsRoutedByType[k] = v
	}
	for k, v := range r.metrics.HandlerInvocations {
		metrics.HandlerInvocations[k] = v
	}
	for k, v := range r.metrics.RuleMatches {
		metrics.RuleMatches[k] = v
	}

	return metrics
}

// rebuildSortedHandlers rebuilds the sorted handlers list.
// Must be called with lock held.
func (r *defaultEventRouter) rebuildSortedHandlers() {
	r.sortedHandlers = make([]SagaEventHandler, 0, len(r.handlers))
	for _, handler := range r.handlers {
		r.sortedHandlers = append(r.sortedHandlers, handler)
	}

	// Sort by priority
	sort.Slice(r.sortedHandlers, func(i, j int) bool {
		return r.sortedHandlers[i].GetPriority() > r.sortedHandlers[j].GetPriority()
	})
}

// applyRoutingRules applies routing rules to filter and reorder handlers.
func (r *defaultEventRouter) applyRoutingRules(event *saga.SagaEvent, handlers []SagaEventHandler) []SagaEventHandler {
	r.mu.RLock()
	rules := r.routingRules
	r.mu.RUnlock()

	result := handlers

	// Apply each matching rule
	for _, rule := range rules {
		if rule.Matches(event) {
			// Update metrics
			r.mu.Lock()
			r.metrics.RuleMatches[rule.GetName()]++
			r.mu.Unlock()

			// Apply rule
			result = rule.SelectHandlers(event, result)
		}
	}

	return result
}

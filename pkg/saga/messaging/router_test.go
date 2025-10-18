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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// mockSagaEventHandler is a mock implementation of SagaEventHandler for testing.
type mockSagaEventHandler struct {
	config            *HandlerConfig
	supportedTypes    []SagaEventType
	canHandleFunc     func(*saga.SagaEvent) bool
	handleFunc        func(context.Context, *saga.SagaEvent, *EventHandlerContext) error
	onProcessedFunc   func(context.Context, *saga.SagaEvent, interface{})
	onFailedFunc      func(context.Context, *saga.SagaEvent, error) EventFailureAction
	priority          int
	handledEvents     []*saga.SagaEvent
	failedEvents      []*saga.SagaEvent
	processedCallback bool
	failedCallback    bool
}

func newMockHandler(id string, priority int, eventTypes ...SagaEventType) *mockSagaEventHandler {
	return &mockSagaEventHandler{
		config: &HandlerConfig{
			HandlerID: id,
		},
		supportedTypes: eventTypes,
		priority:       priority,
		handledEvents:  make([]*saga.SagaEvent, 0),
		failedEvents:   make([]*saga.SagaEvent, 0),
		canHandleFunc: func(event *saga.SagaEvent) bool {
			for _, t := range eventTypes {
				if event.Type == t {
					return true
				}
			}
			return false
		},
		handleFunc: func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) error {
			return nil
		},
	}
}

func (m *mockSagaEventHandler) HandleSagaEvent(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) error {
	if m.handleFunc != nil {
		err := m.handleFunc(ctx, event, handlerCtx)
		if err != nil {
			m.failedEvents = append(m.failedEvents, event)
			return err
		}
	}
	m.handledEvents = append(m.handledEvents, event)
	return nil
}

func (m *mockSagaEventHandler) GetSupportedEventTypes() []SagaEventType {
	return m.supportedTypes
}

func (m *mockSagaEventHandler) CanHandle(event *saga.SagaEvent) bool {
	if m.canHandleFunc != nil {
		return m.canHandleFunc(event)
	}
	return false
}

func (m *mockSagaEventHandler) GetPriority() int {
	return m.priority
}

func (m *mockSagaEventHandler) OnEventProcessed(ctx context.Context, event *saga.SagaEvent, result interface{}) {
	m.processedCallback = true
	if m.onProcessedFunc != nil {
		m.onProcessedFunc(ctx, event, result)
	}
}

func (m *mockSagaEventHandler) OnEventFailed(ctx context.Context, event *saga.SagaEvent, err error) EventFailureAction {
	m.failedCallback = true
	if m.onFailedFunc != nil {
		return m.onFailedFunc(ctx, event, err)
	}
	return FailureActionRetry
}

func (m *mockSagaEventHandler) GetConfiguration() *HandlerConfig {
	return m.config
}

func (m *mockSagaEventHandler) UpdateConfiguration(config *HandlerConfig) error {
	m.config = config
	return nil
}

func (m *mockSagaEventHandler) GetMetrics() *HandlerMetrics {
	return &HandlerMetrics{}
}

func (m *mockSagaEventHandler) HealthCheck(ctx context.Context) error {
	return nil
}

func TestNewDefaultEventRouter(t *testing.T) {
	router := NewDefaultEventRouter()
	if router == nil {
		t.Fatal("Expected non-nil router")
	}

	if router.GetHandlerCount() != 0 {
		t.Errorf("Expected 0 handlers, got %d", router.GetHandlerCount())
	}

	strategy := router.GetRoutingStrategy()
	if strategy != RoutingStrategyAll {
		t.Errorf("Expected default strategy 'all', got '%s'", strategy)
	}
}

func TestEventRouter_RegisterHandler(t *testing.T) {
	router := NewDefaultEventRouter()

	handler1 := newMockHandler("handler-1", 10, saga.EventSagaStarted)
	err := router.RegisterHandler(handler1)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	if router.GetHandlerCount() != 1 {
		t.Errorf("Expected 1 handler, got %d", router.GetHandlerCount())
	}

	// Test duplicate registration
	err = router.RegisterHandler(handler1)
	if err == nil {
		t.Error("Expected error when registering duplicate handler")
	}

	// Test nil handler
	err = router.RegisterHandler(nil)
	if err == nil {
		t.Error("Expected error when registering nil handler")
	}
}

func TestEventRouter_UnregisterHandler(t *testing.T) {
	router := NewDefaultEventRouter()

	handler := newMockHandler("handler-1", 10, saga.EventSagaStarted)
	_ = router.RegisterHandler(handler)

	err := router.UnregisterHandler("handler-1")
	if err != nil {
		t.Fatalf("Failed to unregister handler: %v", err)
	}

	if router.GetHandlerCount() != 0 {
		t.Errorf("Expected 0 handlers, got %d", router.GetHandlerCount())
	}

	// Test unregistering non-existent handler
	err = router.UnregisterHandler("non-existent")
	if err == nil {
		t.Error("Expected error when unregistering non-existent handler")
	}
}

func TestEventRouter_Route(t *testing.T) {
	tests := []struct {
		name        string
		handlers    []*mockSagaEventHandler
		event       *saga.SagaEvent
		expectError bool
		validate    func(*testing.T, []*mockSagaEventHandler)
	}{
		{
			name: "route to single handler",
			handlers: []*mockSagaEventHandler{
				newMockHandler("handler-1", 10, saga.EventSagaStarted),
			},
			event: &saga.SagaEvent{
				ID:        "event-1",
				SagaID:    "saga-1",
				Type:      saga.EventSagaStarted,
				Timestamp: time.Now(),
			},
			expectError: false,
			validate: func(t *testing.T, handlers []*mockSagaEventHandler) {
				if len(handlers[0].handledEvents) != 1 {
					t.Errorf("Expected 1 handled event, got %d", len(handlers[0].handledEvents))
				}
				if !handlers[0].processedCallback {
					t.Error("Expected processed callback to be called")
				}
			},
		},
		{
			name: "route to multiple handlers by priority",
			handlers: []*mockSagaEventHandler{
				newMockHandler("handler-1", 5, saga.EventSagaStarted),
				newMockHandler("handler-2", 10, saga.EventSagaStarted),
				newMockHandler("handler-3", 1, saga.EventSagaStarted),
			},
			event: &saga.SagaEvent{
				ID:        "event-1",
				SagaID:    "saga-1",
				Type:      saga.EventSagaStarted,
				Timestamp: time.Now(),
			},
			expectError: false,
			validate: func(t *testing.T, handlers []*mockSagaEventHandler) {
				// All handlers should have been invoked
				for i, handler := range handlers {
					if len(handler.handledEvents) != 1 {
						t.Errorf("Handler %d: Expected 1 handled event, got %d", i, len(handler.handledEvents))
					}
				}
			},
		},
		{
			name: "no matching handler",
			handlers: []*mockSagaEventHandler{
				newMockHandler("handler-1", 10, saga.EventSagaCompleted),
			},
			event: &saga.SagaEvent{
				ID:        "event-1",
				SagaID:    "saga-1",
				Type:      saga.EventSagaStarted,
				Timestamp: time.Now(),
			},
			expectError: true,
		},
		{
			name: "handler returns error",
			handlers: []*mockSagaEventHandler{
				func() *mockSagaEventHandler {
					h := newMockHandler("handler-1", 10, saga.EventSagaStarted)
					h.handleFunc = func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) error {
						return errors.New("handler error")
					}
					return h
				}(),
			},
			event: &saga.SagaEvent{
				ID:        "event-1",
				SagaID:    "saga-1",
				Type:      saga.EventSagaStarted,
				Timestamp: time.Now(),
			},
			expectError: true,
			validate: func(t *testing.T, handlers []*mockSagaEventHandler) {
				if !handlers[0].failedCallback {
					t.Error("Expected failed callback to be called")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewDefaultEventRouter()

			// Register handlers
			for _, handler := range tt.handlers {
				_ = router.RegisterHandler(handler)
			}

			ctx := context.Background()
			handlerCtx := &EventHandlerContext{
				MessageID: "msg-1",
				Topic:     "saga.events",
				Timestamp: time.Now(),
			}

			err := router.RouteEvent(ctx, tt.event, handlerCtx)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			if tt.validate != nil {
				tt.validate(t, tt.handlers)
			}
		})
	}
}

func TestEventRouter_RouteEvent(t *testing.T) {
	router := NewDefaultEventRouter()

	handler := newMockHandler("handler-1", 10, saga.EventSagaStarted)
	_ = router.RegisterHandler(handler)

	ctx := context.Background()
	event := &saga.SagaEvent{
		ID:        "event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}
	handlerCtx := &EventHandlerContext{
		MessageID: "msg-1",
		Topic:     "saga.events",
		Timestamp: time.Now(),
	}

	// Test RouteEvent (alias for Route)
	err := router.RouteEvent(ctx, event, handlerCtx)
	if err != nil {
		t.Fatalf("RouteEvent failed: %v", err)
	}

	if len(handler.handledEvents) != 1 {
		t.Errorf("Expected 1 handled event, got %d", len(handler.handledEvents))
	}
}

func TestEventRouter_FindHandlers(t *testing.T) {
	router := NewDefaultEventRouter()

	handler1 := newMockHandler("handler-1", 10, saga.EventSagaStarted, saga.EventSagaCompleted)
	handler2 := newMockHandler("handler-2", 5, saga.EventSagaStarted)
	handler3 := newMockHandler("handler-3", 1, saga.EventSagaCompleted)

	_ = router.RegisterHandler(handler1)
	_ = router.RegisterHandler(handler2)
	_ = router.RegisterHandler(handler3)

	event := &saga.SagaEvent{
		ID:        "event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}

	handlers := router.FindHandlers(event)
	if len(handlers) != 2 {
		t.Errorf("Expected 2 matching handlers, got %d", len(handlers))
	}

	// Check priority ordering (highest first)
	if handlers[0].GetConfiguration().HandlerID != "handler-1" {
		t.Errorf("Expected first handler to be 'handler-1', got '%s'", handlers[0].GetConfiguration().HandlerID)
	}
	if handlers[1].GetConfiguration().HandlerID != "handler-2" {
		t.Errorf("Expected second handler to be 'handler-2', got '%s'", handlers[1].GetConfiguration().HandlerID)
	}
}

func TestEventRouter_GetHandlerByID(t *testing.T) {
	router := NewDefaultEventRouter()

	handler := newMockHandler("handler-1", 10, saga.EventSagaStarted)
	_ = router.RegisterHandler(handler)

	retrieved, err := router.GetHandlerByID("handler-1")
	if err != nil {
		t.Fatalf("Failed to get handler: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected non-nil handler")
	}
	if retrieved.GetConfiguration().HandlerID != "handler-1" {
		t.Errorf("Expected handler ID 'handler-1', got '%s'", retrieved.GetConfiguration().HandlerID)
	}

	// Test non-existent handler
	_, err = router.GetHandlerByID("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent handler")
	}
}

func TestEventRouter_GetHandlersByEventType(t *testing.T) {
	router := NewDefaultEventRouter()

	handler1 := newMockHandler("handler-1", 10, saga.EventSagaStarted, saga.EventSagaCompleted)
	handler2 := newMockHandler("handler-2", 5, saga.EventSagaStarted)
	handler3 := newMockHandler("handler-3", 1, saga.EventSagaCompleted)

	_ = router.RegisterHandler(handler1)
	_ = router.RegisterHandler(handler2)
	_ = router.RegisterHandler(handler3)

	handlers := router.GetHandlersByEventType(saga.EventSagaStarted)
	if len(handlers) != 2 {
		t.Errorf("Expected 2 handlers for EventSagaStarted, got %d", len(handlers))
	}

	handlers = router.GetHandlersByEventType(saga.EventSagaCompleted)
	if len(handlers) != 2 {
		t.Errorf("Expected 2 handlers for EventSagaCompleted, got %d", len(handlers))
	}

	handlers = router.GetHandlersByEventType(saga.EventSagaFailed)
	if len(handlers) != 0 {
		t.Errorf("Expected 0 handlers for EventSagaFailed, got %d", len(handlers))
	}
}

func TestEventRouter_RoutingStrategy(t *testing.T) {
	router := NewDefaultEventRouter(WithRoutingStrategy(RoutingStrategyPriority))

	strategy := router.GetRoutingStrategy()
	if strategy != RoutingStrategyPriority {
		t.Errorf("Expected strategy 'priority', got '%s'", strategy)
	}

	router.SetRoutingStrategy(RoutingStrategyRoundRobin)
	strategy = router.GetRoutingStrategy()
	if strategy != RoutingStrategyRoundRobin {
		t.Errorf("Expected strategy 'round_robin', got '%s'", strategy)
	}
}

func TestEventRouter_RoutingStrategy_Priority(t *testing.T) {
	router := NewDefaultEventRouter(WithRoutingStrategy(RoutingStrategyPriority))

	// Register handlers with different priorities
	handler1 := newMockHandler("handler-1", 5, saga.EventSagaStarted)
	handler2 := newMockHandler("handler-2", 10, saga.EventSagaStarted) // Highest priority
	handler3 := newMockHandler("handler-3", 1, saga.EventSagaStarted)

	_ = router.RegisterHandler(handler1)
	_ = router.RegisterHandler(handler2)
	_ = router.RegisterHandler(handler3)

	ctx := context.Background()
	event := &saga.SagaEvent{
		ID:        "event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}
	handlerCtx := &EventHandlerContext{
		MessageID: "msg-1",
		Topic:     "saga.events",
		Timestamp: time.Now(),
	}

	err := router.RouteEvent(ctx, event, handlerCtx)
	if err != nil {
		t.Fatalf("RouteEvent failed: %v", err)
	}

	// Only the highest priority handler (handler2) should have processed the event
	if len(handler1.handledEvents) != 0 {
		t.Errorf("Handler1 should not have processed event, got %d events", len(handler1.handledEvents))
	}
	if len(handler2.handledEvents) != 1 {
		t.Errorf("Handler2 should have processed event, got %d events", len(handler2.handledEvents))
	}
	if len(handler3.handledEvents) != 0 {
		t.Errorf("Handler3 should not have processed event, got %d events", len(handler3.handledEvents))
	}
}

func TestEventRouter_RoutingStrategy_RoundRobin(t *testing.T) {
	router := NewDefaultEventRouter(WithRoutingStrategy(RoutingStrategyRoundRobin))

	// Register multiple handlers
	handler1 := newMockHandler("handler-1", 5, saga.EventSagaStarted)
	handler2 := newMockHandler("handler-2", 5, saga.EventSagaStarted)
	handler3 := newMockHandler("handler-3", 5, saga.EventSagaStarted)

	_ = router.RegisterHandler(handler1)
	_ = router.RegisterHandler(handler2)
	_ = router.RegisterHandler(handler3)

	ctx := context.Background()
	handlerCtx := &EventHandlerContext{
		MessageID: "msg-1",
		Topic:     "saga.events",
		Timestamp: time.Now(),
	}

	// Send 6 events - each handler should get 2 events
	for i := 0; i < 6; i++ {
		event := &saga.SagaEvent{
			ID:        fmt.Sprintf("event-%d", i),
			SagaID:    "saga-1",
			Type:      saga.EventSagaStarted,
			Timestamp: time.Now(),
		}
		err := router.RouteEvent(ctx, event, handlerCtx)
		if err != nil {
			t.Fatalf("RouteEvent %d failed: %v", i, err)
		}
	}

	// Each event should be handled by exactly one handler
	totalHandled := len(handler1.handledEvents) + len(handler2.handledEvents) + len(handler3.handledEvents)
	if totalHandled != 6 {
		t.Errorf("Expected 6 total handled events, got %d", totalHandled)
	}

	// Each handler should have handled at least one event (round-robin distribution)
	if len(handler1.handledEvents) == 0 {
		t.Error("Handler1 should have handled at least one event")
	}
	if len(handler2.handledEvents) == 0 {
		t.Error("Handler2 should have handled at least one event")
	}
	if len(handler3.handledEvents) == 0 {
		t.Error("Handler3 should have handled at least one event")
	}
}

func TestEventRouter_RoutingStrategy_All(t *testing.T) {
	router := NewDefaultEventRouter(WithRoutingStrategy(RoutingStrategyAll))

	// Register multiple handlers
	handler1 := newMockHandler("handler-1", 5, saga.EventSagaStarted)
	handler2 := newMockHandler("handler-2", 10, saga.EventSagaStarted)

	_ = router.RegisterHandler(handler1)
	_ = router.RegisterHandler(handler2)

	ctx := context.Background()
	event := &saga.SagaEvent{
		ID:        "event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}
	handlerCtx := &EventHandlerContext{
		MessageID: "msg-1",
		Topic:     "saga.events",
		Timestamp: time.Now(),
	}

	err := router.RouteEvent(ctx, event, handlerCtx)
	if err != nil {
		t.Fatalf("RouteEvent failed: %v", err)
	}

	// Both handlers should have processed the event
	if len(handler1.handledEvents) != 1 {
		t.Errorf("Handler1 should have processed event, got %d events", len(handler1.handledEvents))
	}
	if len(handler2.handledEvents) != 1 {
		t.Errorf("Handler2 should have processed event, got %d events", len(handler2.handledEvents))
	}
}

func TestEventRouter_GetRouterMetrics(t *testing.T) {
	router := NewDefaultEventRouter()

	handler := newMockHandler("handler-1", 10, saga.EventSagaStarted)
	_ = router.RegisterHandler(handler)

	ctx := context.Background()
	event := &saga.SagaEvent{
		ID:        "event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}
	handlerCtx := &EventHandlerContext{
		MessageID: "msg-1",
		Topic:     "saga.events",
		Timestamp: time.Now(),
	}

	_ = router.RouteEvent(ctx, event, handlerCtx)

	metrics := router.GetRouterMetrics()
	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	if metrics.TotalEventsRouted != 1 {
		t.Errorf("Expected 1 routed event, got %d", metrics.TotalEventsRouted)
	}

	if metrics.HandlerInvocations["handler-1"] != 1 {
		t.Errorf("Expected 1 handler invocation, got %d", metrics.HandlerInvocations["handler-1"])
	}
}

func TestEventRouter_ContinueOnError(t *testing.T) {
	router := NewDefaultEventRouter(WithContinueOnError(true))

	handler1 := func() *mockSagaEventHandler {
		h := newMockHandler("handler-1", 10, saga.EventSagaStarted)
		h.handleFunc = func(ctx context.Context, event *saga.SagaEvent, handlerCtx *EventHandlerContext) error {
			return errors.New("handler 1 error")
		}
		return h
	}()

	handler2 := newMockHandler("handler-2", 5, saga.EventSagaStarted)

	_ = router.RegisterHandler(handler1)
	_ = router.RegisterHandler(handler2)

	ctx := context.Background()
	event := &saga.SagaEvent{
		ID:        "event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}
	handlerCtx := &EventHandlerContext{
		MessageID: "msg-1",
		Topic:     "saga.events",
		Timestamp: time.Now(),
	}

	// With continueOnError=true, routing should continue despite handler1 error
	err := router.RouteEvent(ctx, event, handlerCtx)
	if err != nil {
		t.Logf("RouteEvent returned error (expected with continueOnError): %v", err)
	}

	// handler2 should still have processed the event
	if len(handler2.handledEvents) != 1 {
		t.Errorf("Expected handler2 to process event despite handler1 error, got %d processed events", len(handler2.handledEvents))
	}
}

// mockRoutingRule is a mock implementation of RoutingRule for testing.
type mockRoutingRule struct {
	name           string
	priority       int
	matchFunc      func(*saga.SagaEvent) bool
	selectFunc     func(*saga.SagaEvent, []SagaEventHandler) []SagaEventHandler
	matchCount     int
	lastMatchEvent *saga.SagaEvent
}

func (m *mockRoutingRule) GetName() string {
	return m.name
}

func (m *mockRoutingRule) Matches(event *saga.SagaEvent) bool {
	if m.matchFunc != nil {
		matched := m.matchFunc(event)
		if matched {
			m.matchCount++
			m.lastMatchEvent = event
		}
		return matched
	}
	return false
}

func (m *mockRoutingRule) SelectHandlers(event *saga.SagaEvent, availableHandlers []SagaEventHandler) []SagaEventHandler {
	if m.selectFunc != nil {
		return m.selectFunc(event, availableHandlers)
	}
	return availableHandlers
}

func (m *mockRoutingRule) GetPriority() int {
	return m.priority
}

func TestEventRouter_AddRoutingRule(t *testing.T) {
	router := NewDefaultEventRouter()

	rule := &mockRoutingRule{
		name:     "test-rule",
		priority: 10,
		matchFunc: func(event *saga.SagaEvent) bool {
			return event.Type == saga.EventSagaStarted
		},
		selectFunc: func(event *saga.SagaEvent, handlers []SagaEventHandler) []SagaEventHandler {
			return handlers
		},
	}

	err := router.AddRoutingRule(rule)
	if err != nil {
		t.Fatalf("Failed to add routing rule: %v", err)
	}

	rules := router.GetRoutingRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 routing rule, got %d", len(rules))
	}

	// Test duplicate rule
	err = router.AddRoutingRule(rule)
	if err == nil {
		t.Error("Expected error when adding duplicate rule")
	}

	// Test nil rule
	err = router.AddRoutingRule(nil)
	if err == nil {
		t.Error("Expected error when adding nil rule")
	}
}

func TestEventRouter_RemoveRoutingRule(t *testing.T) {
	router := NewDefaultEventRouter()

	rule := &mockRoutingRule{
		name:     "test-rule",
		priority: 10,
	}

	_ = router.AddRoutingRule(rule)

	err := router.RemoveRoutingRule("test-rule")
	if err != nil {
		t.Fatalf("Failed to remove routing rule: %v", err)
	}

	rules := router.GetRoutingRules()
	if len(rules) != 0 {
		t.Errorf("Expected 0 routing rules, got %d", len(rules))
	}

	// Test removing non-existent rule
	err = router.RemoveRoutingRule("non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent rule")
	}
}

func BenchmarkEventRouter_Route(b *testing.B) {
	router := NewDefaultEventRouter()

	handler := newMockHandler("handler-1", 10, saga.EventSagaStarted)
	_ = router.RegisterHandler(handler)

	ctx := context.Background()
	event := &saga.SagaEvent{
		ID:        "event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}
	handlerCtx := &EventHandlerContext{
		MessageID: "msg-1",
		Topic:     "saga.events",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.RouteEvent(ctx, event, handlerCtx)
	}
}

func BenchmarkEventRouter_FindHandlers(b *testing.B) {
	router := NewDefaultEventRouter()

	for i := 0; i < 10; i++ {
		handler := newMockHandler("handler-"+string(rune(i)), i, saga.EventSagaStarted)
		_ = router.RegisterHandler(handler)
	}

	event := &saga.SagaEvent{
		ID:        "event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.FindHandlers(event)
	}
}

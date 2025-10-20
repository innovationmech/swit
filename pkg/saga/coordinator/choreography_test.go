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

package coordinator

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// Mock event handler for testing
type mockChoreographyHandler struct {
	id              string
	supportedTypes  []saga.SagaEventType
	priority        int
	handleFunc      func(context.Context, *saga.SagaEvent) error
	handleCallCount int
	mu              sync.Mutex
}

func (m *mockChoreographyHandler) HandleEvent(ctx context.Context, event *saga.SagaEvent) error {
	m.mu.Lock()
	m.handleCallCount++
	m.mu.Unlock()

	if m.handleFunc != nil {
		return m.handleFunc(ctx, event)
	}
	return nil
}

func (m *mockChoreographyHandler) GetHandlerID() string {
	return m.id
}

func (m *mockChoreographyHandler) GetSupportedEventTypes() []saga.SagaEventType {
	return m.supportedTypes
}

func (m *mockChoreographyHandler) GetPriority() int {
	return m.priority
}

func (m *mockChoreographyHandler) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.handleCallCount
}

// Mock event publisher for choreography testing
type choreographyMockEventPublisher struct {
	events        []*saga.SagaEvent
	subscriptions map[string]*choreographyMockSubscription
	mu            sync.RWMutex
}

type choreographyMockSubscription struct {
	id      string
	filter  saga.EventFilter
	handler saga.EventHandler
	active  bool
}

func (s *choreographyMockSubscription) GetID() string                       { return s.id }
func (s *choreographyMockSubscription) GetFilter() saga.EventFilter         { return s.filter }
func (s *choreographyMockSubscription) GetHandler() saga.EventHandler       { return s.handler }
func (s *choreographyMockSubscription) IsActive() bool                      { return s.active }
func (s *choreographyMockSubscription) GetCreatedAt() time.Time             { return time.Now() }
func (s *choreographyMockSubscription) GetMetadata() map[string]interface{} { return nil }

func newChoreographyMockEventPublisher() *choreographyMockEventPublisher {
	return &choreographyMockEventPublisher{
		events:        make([]*saga.SagaEvent, 0),
		subscriptions: make(map[string]*choreographyMockSubscription),
	}
}

func (m *choreographyMockEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	m.mu.Lock()
	m.events = append(m.events, event)
	m.mu.Unlock()

	// Notify subscribers
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, sub := range m.subscriptions {
		if sub.active && sub.filter.Match(event) {
			_ = sub.handler.HandleEvent(ctx, event)
		}
	}
	return nil
}

func (m *choreographyMockEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub := &choreographyMockSubscription{
		id:      generateChoreographyEventID(),
		filter:  filter,
		handler: handler,
		active:  true,
	}
	m.subscriptions[sub.id] = sub
	return sub, nil
}

func (m *choreographyMockEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sub, exists := m.subscriptions[subscription.GetID()]; exists {
		sub.active = false
		delete(m.subscriptions, subscription.GetID())
	}
	return nil
}

func (m *choreographyMockEventPublisher) Close() error {
	return nil
}

func (m *choreographyMockEventPublisher) GetEvents() []*saga.SagaEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	events := make([]*saga.SagaEvent, len(m.events))
	copy(events, m.events)
	return events
}

// TestNewChoreographyCoordinator tests the creation of a choreography coordinator.
func TestNewChoreographyCoordinator(t *testing.T) {
	tests := []struct {
		name      string
		config    *ChoreographyConfig
		expectErr bool
	}{
		{
			name:      "nil config",
			config:    nil,
			expectErr: true,
		},
		{
			name: "missing event publisher",
			config: &ChoreographyConfig{
				EventPublisher: nil,
			},
			expectErr: true,
		},
		{
			name: "valid minimal config",
			config: &ChoreographyConfig{
				EventPublisher: newChoreographyMockEventPublisher(),
			},
			expectErr: false,
		},
		{
			name: "valid full config",
			config: &ChoreographyConfig{
				EventPublisher:        newChoreographyMockEventPublisher(),
				MaxConcurrentHandlers: 50,
				EventTimeout:          10 * time.Second,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordinator, err := NewChoreographyCoordinator(tt.config)
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if coordinator == nil {
					t.Error("Expected coordinator but got nil")
				}
				if coordinator != nil {
					_ = coordinator.Close()
				}
			}
		})
	}
}

// TestRegisterEventHandler tests event handler registration.
func TestRegisterEventHandler(t *testing.T) {
	coordinator, err := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: newChoreographyMockEventPublisher(),
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	handler := &mockChoreographyHandler{
		id:             "test-handler-1",
		supportedTypes: []saga.SagaEventType{saga.EventSagaStarted},
		priority:       10,
	}

	// Test successful registration
	err = coordinator.RegisterEventHandler(handler)
	if err != nil {
		t.Errorf("Failed to register handler: %v", err)
	}

	// Verify handler is registered
	handlers := coordinator.GetRegisteredHandlers()
	if len(handlers) != 1 {
		t.Errorf("Expected 1 registered handler, got %d", len(handlers))
	}

	// Test duplicate registration
	err = coordinator.RegisterEventHandler(handler)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}

	// Test registration after close
	_ = coordinator.Close()
	handler2 := &mockChoreographyHandler{
		id:             "test-handler-2",
		supportedTypes: []saga.SagaEventType{saga.EventSagaCompleted},
		priority:       5,
	}
	err = coordinator.RegisterEventHandler(handler2)
	if err == nil {
		t.Error("Expected error when registering after close")
	}
}

// TestUnregisterEventHandler tests event handler unregistration.
func TestUnregisterEventHandler(t *testing.T) {
	coordinator, err := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: newChoreographyMockEventPublisher(),
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	handler := &mockChoreographyHandler{
		id:             "test-handler-1",
		supportedTypes: []saga.SagaEventType{saga.EventSagaStarted},
		priority:       10,
	}

	// Register handler
	err = coordinator.RegisterEventHandler(handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Test successful unregistration
	err = coordinator.UnregisterEventHandler(handler.id)
	if err != nil {
		t.Errorf("Failed to unregister handler: %v", err)
	}

	// Verify handler is unregistered
	handlers := coordinator.GetRegisteredHandlers()
	if len(handlers) != 0 {
		t.Errorf("Expected 0 registered handlers, got %d", len(handlers))
	}

	// Test unregistering non-existent handler
	err = coordinator.UnregisterEventHandler("non-existent")
	if err == nil {
		t.Error("Expected error when unregistering non-existent handler")
	}
}

// TestStartStop tests starting and stopping the coordinator.
func TestStartStop(t *testing.T) {
	coordinator, err := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: newChoreographyMockEventPublisher(),
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Test starting
	ctx := context.Background()
	err = coordinator.Start(ctx)
	if err != nil {
		t.Errorf("Failed to start coordinator: %v", err)
	}

	if !coordinator.IsRunning() {
		t.Error("Expected coordinator to be running")
	}

	// Test starting again (should fail)
	err = coordinator.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running coordinator")
	}

	// Test stopping
	err = coordinator.Stop(ctx)
	if err != nil {
		t.Errorf("Failed to stop coordinator: %v", err)
	}

	if coordinator.IsRunning() {
		t.Error("Expected coordinator to not be running")
	}
}

// TestEventHandling tests event handling with registered handlers.
func TestEventHandling(t *testing.T) {
	publisher := newChoreographyMockEventPublisher()
	coordinator, err := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: publisher,
		EventTimeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Create mock handler
	var handledEvent *saga.SagaEvent
	var handlerWg sync.WaitGroup
	handlerWg.Add(1)

	handler := &mockChoreographyHandler{
		id:             "test-handler",
		supportedTypes: []saga.SagaEventType{saga.EventSagaStarted},
		priority:       10,
		handleFunc: func(ctx context.Context, event *saga.SagaEvent) error {
			handledEvent = event
			handlerWg.Done()
			return nil
		},
	}

	// Register handler
	err = coordinator.RegisterEventHandler(handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Start coordinator
	ctx := context.Background()
	err = coordinator.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}

	// Publish event
	event := &saga.SagaEvent{
		ID:        "test-event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"key": "value"},
	}

	err = publisher.PublishEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// Wait for handler to process event
	done := make(chan struct{})
	go func() {
		handlerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event handler")
	}

	// Verify event was handled
	if handledEvent == nil {
		t.Fatal("Event was not handled")
	}
	if handledEvent.ID != event.ID {
		t.Errorf("Expected event ID %s, got %s", event.ID, handledEvent.ID)
	}
}

// TestMultipleHandlers tests handling events with multiple registered handlers.
func TestMultipleHandlers(t *testing.T) {
	publisher := newChoreographyMockEventPublisher()
	coordinator, err := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: publisher,
		EventTimeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Create multiple handlers
	var handlerWg sync.WaitGroup
	handlerWg.Add(3)

	handler1 := &mockChoreographyHandler{
		id:             "handler-1",
		supportedTypes: []saga.SagaEventType{saga.EventSagaStarted},
		priority:       10,
		handleFunc: func(ctx context.Context, event *saga.SagaEvent) error {
			handlerWg.Done()
			return nil
		},
	}

	handler2 := &mockChoreographyHandler{
		id:             "handler-2",
		supportedTypes: []saga.SagaEventType{saga.EventSagaStarted},
		priority:       20,
		handleFunc: func(ctx context.Context, event *saga.SagaEvent) error {
			handlerWg.Done()
			return nil
		},
	}

	handler3 := &mockChoreographyHandler{
		id:             "handler-3",
		supportedTypes: []saga.SagaEventType{saga.EventSagaStarted},
		priority:       5,
		handleFunc: func(ctx context.Context, event *saga.SagaEvent) error {
			handlerWg.Done()
			return nil
		},
	}

	// Register handlers
	_ = coordinator.RegisterEventHandler(handler1)
	_ = coordinator.RegisterEventHandler(handler2)
	_ = coordinator.RegisterEventHandler(handler3)

	// Start coordinator
	ctx := context.Background()
	err = coordinator.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}

	// Publish event
	event := &saga.SagaEvent{
		ID:        "test-event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}

	err = publisher.PublishEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// Wait for all handlers to process event
	done := make(chan struct{})
	go func() {
		handlerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event handlers")
	}

	// Verify all handlers were called
	if handler1.GetCallCount() != 1 {
		t.Errorf("Expected handler1 to be called once, got %d", handler1.GetCallCount())
	}
	if handler2.GetCallCount() != 1 {
		t.Errorf("Expected handler2 to be called once, got %d", handler2.GetCallCount())
	}
	if handler3.GetCallCount() != 1 {
		t.Errorf("Expected handler3 to be called once, got %d", handler3.GetCallCount())
	}
}

// TestHandlerRetry tests retry logic for failed handlers.
func TestHandlerRetry(t *testing.T) {
	publisher := newChoreographyMockEventPublisher()
	coordinator, err := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher:     publisher,
		EventTimeout:       5 * time.Second,
		HandlerRetryPolicy: saga.NewFixedDelayRetryPolicy(3, 100*time.Millisecond),
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Create handler that fails twice then succeeds
	attemptCount := 0
	var mu sync.Mutex
	var handlerWg sync.WaitGroup
	handlerWg.Add(1)

	handler := &mockChoreographyHandler{
		id:             "retry-handler",
		supportedTypes: []saga.SagaEventType{saga.EventSagaStarted},
		priority:       10,
		handleFunc: func(ctx context.Context, event *saga.SagaEvent) error {
			mu.Lock()
			attemptCount++
			currentAttempt := attemptCount
			mu.Unlock()

			if currentAttempt < 3 {
				return errors.New("temporary failure")
			}

			handlerWg.Done()
			return nil
		},
	}

	// Register handler
	err = coordinator.RegisterEventHandler(handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Start coordinator
	ctx := context.Background()
	err = coordinator.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}

	// Publish event
	event := &saga.SagaEvent{
		ID:        "test-event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}

	err = publisher.PublishEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// Wait for handler to succeed after retries
	done := make(chan struct{})
	go func() {
		handlerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for handler to succeed after retries")
	}

	// Verify handler was retried
	mu.Lock()
	finalAttemptCount := attemptCount
	mu.Unlock()

	if finalAttemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", finalAttemptCount)
	}
}

// TestGetMetrics tests metrics collection.
func TestGetMetrics(t *testing.T) {
	publisher := newChoreographyMockEventPublisher()
	coordinator, err := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: publisher,
		EnableMetrics:  true,
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Create handler
	var handlerWg sync.WaitGroup
	handlerWg.Add(2)

	handler := &mockChoreographyHandler{
		id:             "test-handler",
		supportedTypes: []saga.SagaEventType{saga.EventSagaStarted, saga.EventSagaCompleted},
		priority:       10,
		handleFunc: func(ctx context.Context, event *saga.SagaEvent) error {
			handlerWg.Done()
			return nil
		},
	}

	// Register handler
	err = coordinator.RegisterEventHandler(handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Start coordinator
	ctx := context.Background()
	err = coordinator.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}

	// Publish events
	event1 := &saga.SagaEvent{
		ID:        "event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}
	event2 := &saga.SagaEvent{
		ID:        "event-2",
		SagaID:    "saga-1",
		Type:      saga.EventSagaCompleted,
		Timestamp: time.Now(),
	}

	_ = publisher.PublishEvent(ctx, event1)
	_ = publisher.PublishEvent(ctx, event2)

	// Wait for handlers
	done := make(chan struct{})
	go func() {
		handlerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for handlers")
	}

	// Get metrics
	metrics := coordinator.GetMetrics()
	if metrics == nil {
		t.Fatal("Expected metrics but got nil")
	}

	if metrics.TotalHandlers != 1 {
		t.Errorf("Expected 1 total handler, got %d", metrics.TotalHandlers)
	}

	if metrics.TotalEvents < 2 {
		t.Errorf("Expected at least 2 total events, got %d", metrics.TotalEvents)
	}

	if metrics.ProcessedEvents < 2 {
		t.Errorf("Expected at least 2 processed events, got %d", metrics.ProcessedEvents)
	}
}

// TestPublishEvent tests publishing events through the coordinator.
func TestPublishEvent(t *testing.T) {
	publisher := newChoreographyMockEventPublisher()
	coordinator, err := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: publisher,
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	ctx := context.Background()

	// Test publishing event without ID
	event := &saga.SagaEvent{
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Time{}, // Will be set by coordinator
	}

	err = coordinator.PublishEvent(ctx, event)
	if err != nil {
		t.Errorf("Failed to publish event: %v", err)
	}

	// Verify event was published
	events := publisher.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 published event, got %d", len(events))
	}

	// Verify event has ID and timestamp
	publishedEvent := events[0]
	if publishedEvent.ID == "" {
		t.Error("Expected event to have ID")
	}
	if publishedEvent.Timestamp.IsZero() {
		t.Error("Expected event to have timestamp")
	}
}

// TestGetHandlersByEventType tests retrieving handlers by event type.
func TestGetHandlersByEventType(t *testing.T) {
	coordinator, err := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher: newChoreographyMockEventPublisher(),
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Register handlers for different event types
	handler1 := &mockChoreographyHandler{
		id:             "handler-1",
		supportedTypes: []saga.SagaEventType{saga.EventSagaStarted, saga.EventSagaCompleted},
		priority:       10,
	}
	handler2 := &mockChoreographyHandler{
		id:             "handler-2",
		supportedTypes: []saga.SagaEventType{saga.EventSagaStarted},
		priority:       20,
	}
	handler3 := &mockChoreographyHandler{
		id:             "handler-3",
		supportedTypes: []saga.SagaEventType{saga.EventSagaFailed},
		priority:       5,
	}

	_ = coordinator.RegisterEventHandler(handler1)
	_ = coordinator.RegisterEventHandler(handler2)
	_ = coordinator.RegisterEventHandler(handler3)

	// Test getting handlers for EventSagaStarted
	handlers := coordinator.GetHandlersByEventType(saga.EventSagaStarted)
	if len(handlers) != 2 {
		t.Errorf("Expected 2 handlers for EventSagaStarted, got %d", len(handlers))
	}

	// Test getting handlers for EventSagaCompleted
	handlers = coordinator.GetHandlersByEventType(saga.EventSagaCompleted)
	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler for EventSagaCompleted, got %d", len(handlers))
	}

	// Test getting handlers for EventSagaFailed
	handlers = coordinator.GetHandlersByEventType(saga.EventSagaFailed)
	if len(handlers) != 1 {
		t.Errorf("Expected 1 handler for EventSagaFailed, got %d", len(handlers))
	}

	// Test getting handlers for unregistered event type
	handlers = coordinator.GetHandlersByEventType(saga.EventSagaTimedOut)
	if len(handlers) != 0 {
		t.Errorf("Expected 0 handlers for EventSagaTimedOut, got %d", len(handlers))
	}
}

// TestConcurrentHandlerExecution tests concurrent execution limits.
func TestConcurrentHandlerExecution(t *testing.T) {
	publisher := newChoreographyMockEventPublisher()
	coordinator, err := NewChoreographyCoordinator(&ChoreographyConfig{
		EventPublisher:        publisher,
		MaxConcurrentHandlers: 2,
		EventTimeout:          5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Create slow handlers
	var handlerWg sync.WaitGroup
	handlerWg.Add(3)

	var activeMu sync.Mutex
	activeCount := 0
	maxActive := 0

	createSlowHandler := func(id string) *mockChoreographyHandler {
		return &mockChoreographyHandler{
			id:             id,
			supportedTypes: []saga.SagaEventType{saga.EventSagaStarted},
			priority:       10,
			handleFunc: func(ctx context.Context, event *saga.SagaEvent) error {
				activeMu.Lock()
				activeCount++
				if activeCount > maxActive {
					maxActive = activeCount
				}
				activeMu.Unlock()

				time.Sleep(100 * time.Millisecond)

				activeMu.Lock()
				activeCount--
				activeMu.Unlock()

				handlerWg.Done()
				return nil
			},
		}
	}

	// Register handlers
	_ = coordinator.RegisterEventHandler(createSlowHandler("handler-1"))
	_ = coordinator.RegisterEventHandler(createSlowHandler("handler-2"))
	_ = coordinator.RegisterEventHandler(createSlowHandler("handler-3"))

	// Start coordinator
	ctx := context.Background()
	err = coordinator.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}

	// Publish event
	event := &saga.SagaEvent{
		ID:        "event-1",
		SagaID:    "saga-1",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}

	err = publisher.PublishEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// Wait for handlers
	done := make(chan struct{})
	go func() {
		handlerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for handlers")
	}

	// Verify concurrent execution was limited
	activeMu.Lock()
	finalMaxActive := maxActive
	activeMu.Unlock()

	if finalMaxActive > 2 {
		t.Errorf("Expected max 2 concurrent handlers, got %d", finalMaxActive)
	}
}

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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestChoreographyCoordinator_Integration tests the choreography coordinator with event-driven workflow.
// This test verifies:
// - Event publishing and subscription
// - Multiple event handlers coordination
// - Event order guarantees
// - State consistency
func TestChoreographyCoordinator_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 10,
		EventTimeout:          5 * time.Second,
		EnableMetrics:         true,
	}

	coordinator, err := NewChoreographyCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create choreography coordinator: %v", err)
	}
	defer coordinator.Close()

	// Track event processing order
	var eventOrder []string
	var mu sync.Mutex

	// Register event handlers for different event types
	orderCreatedHandler := &testEventHandler{
		id:       "order-created-handler",
		types:    []saga.SagaEventType{saga.EventSagaStarted},
		priority: 10,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			mu.Lock()
			eventOrder = append(eventOrder, "order-created")
			mu.Unlock()

			// Publish inventory reserved event
			inventoryEvent := &saga.SagaEvent{
				ID:        "inventory-reserved-" + event.SagaID,
				SagaID:    event.SagaID,
				Type:      saga.EventSagaStepCompleted,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"step":   "inventory-reserved",
					"status": "completed",
				},
			}
			return publisher.PublishEvent(ctx, inventoryEvent)
		},
	}

	inventoryHandler := &testEventHandler{
		id:       "inventory-handler",
		types:    []saga.SagaEventType{saga.EventSagaStepCompleted},
		priority: 5,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			data := event.Data.(map[string]interface{})
			if step, ok := data["step"].(string); ok && step == "inventory-reserved" {
				mu.Lock()
				eventOrder = append(eventOrder, "inventory-reserved")
				mu.Unlock()

				// Publish payment processed event
				paymentEvent := &saga.SagaEvent{
					ID:        "payment-processed-" + event.SagaID,
					SagaID:    event.SagaID,
					Type:      saga.EventSagaStepCompleted,
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"step":   "payment-processed",
						"status": "completed",
					},
				}
				return publisher.PublishEvent(ctx, paymentEvent)
			}
			return nil
		},
	}

	paymentHandler := &testEventHandler{
		id:       "payment-handler",
		types:    []saga.SagaEventType{saga.EventSagaStepCompleted},
		priority: 3,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			data := event.Data.(map[string]interface{})
			if step, ok := data["step"].(string); ok && step == "payment-processed" {
				mu.Lock()
				eventOrder = append(eventOrder, "payment-processed")
				mu.Unlock()

				// Publish saga completed event
				completedEvent := &saga.SagaEvent{
					ID:        "saga-completed-" + event.SagaID,
					SagaID:    event.SagaID,
					Type:      saga.EventSagaCompleted,
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"status": "completed",
					},
				}
				return publisher.PublishEvent(ctx, completedEvent)
			}
			return nil
		},
	}

	// Register handlers
	if err := coordinator.RegisterEventHandler(orderCreatedHandler); err != nil {
		t.Fatalf("Failed to register order handler: %v", err)
	}
	if err := coordinator.RegisterEventHandler(inventoryHandler); err != nil {
		t.Fatalf("Failed to register inventory handler: %v", err)
	}
	if err := coordinator.RegisterEventHandler(paymentHandler); err != nil {
		t.Fatalf("Failed to register payment handler: %v", err)
	}

	// Start the coordinator
	ctx := context.Background()
	if err := coordinator.Start(ctx); err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}
	defer coordinator.Stop(ctx)

	// Trigger the saga by publishing initial event
	startEvent := &saga.SagaEvent{
		ID:        "saga-start-event",
		SagaID:    "test-saga-001",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"order_id": "ORDER-001",
			"amount":   100.0,
		},
	}

	if err := publisher.PublishEvent(ctx, startEvent); err != nil {
		t.Fatalf("Failed to publish start event: %v", err)
	}

	// Wait for all events to be processed
	time.Sleep(2 * time.Second)

	// Verify event processing order
	mu.Lock()
	expectedOrder := []string{"order-created", "inventory-reserved", "payment-processed"}
	if len(eventOrder) != len(expectedOrder) {
		t.Errorf("Expected %d events, got %d", len(expectedOrder), len(eventOrder))
	} else {
		for i, expected := range expectedOrder {
			if eventOrder[i] != expected {
				t.Errorf("Event order mismatch at index %d: expected %s, got %s", i, expected, eventOrder[i])
			}
		}
	}
	mu.Unlock()

	// Verify metrics
	metrics := coordinator.GetMetrics()
	if metrics.TotalEvents == 0 {
		t.Error("Expected events to be published")
	}
	if metrics.ProcessedEvents == 0 {
		t.Error("Expected events to be handled")
	}
}

// TestChoreographyCoordinator_MultiServiceCoordination tests multiple services coordinating via events.
func TestChoreographyCoordinator_MultiServiceCoordination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 20,
		EventTimeout:          5 * time.Second,
		EnableMetrics:         true,
	}

	coordinator, err := NewChoreographyCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create choreography coordinator: %v", err)
	}
	defer coordinator.Close()

	// Track service calls
	var serviceCalls atomic.Int32

	// Simulate multiple services handling the same event type
	for i := 0; i < 5; i++ {
		handler := &testEventHandler{
			id:       "service-handler-" + string(rune('A'+i)),
			types:    []saga.SagaEventType{saga.EventSagaStarted},
			priority: i,
			handler: func(ctx context.Context, event *saga.SagaEvent) error {
				serviceCalls.Add(1)
				return nil
			},
		}
		if err := coordinator.RegisterEventHandler(handler); err != nil {
			t.Fatalf("Failed to register handler %d: %v", i, err)
		}
	}

	ctx := context.Background()
	if err := coordinator.Start(ctx); err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}
	defer coordinator.Stop(ctx)

	// Publish event
	event := &saga.SagaEvent{
		ID:        "multi-service-event",
		SagaID:    "saga-002",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	if err := publisher.PublishEvent(ctx, event); err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// Wait for processing
	time.Sleep(time.Second)

	// Verify all services were called
	calls := serviceCalls.Load()
	if calls != 5 {
		t.Errorf("Expected 5 service calls, got %d", calls)
	}
}

// TestChoreographyCoordinator_EventOrdering tests that events are processed in correct order.
func TestChoreographyCoordinator_EventOrdering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 1, // Sequential processing
		EventTimeout:          5 * time.Second,
	}

	coordinator, err := NewChoreographyCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	var processedEvents []string
	var mu sync.Mutex

	handler := &testEventHandler{
		id:       "ordering-handler",
		types:    []saga.SagaEventType{saga.EventSagaStarted, saga.EventSagaStepStarted, saga.EventSagaStepCompleted},
		priority: 1,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			mu.Lock()
			processedEvents = append(processedEvents, event.ID)
			mu.Unlock()
			return nil
		},
	}

	if err := coordinator.RegisterEventHandler(handler); err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	ctx := context.Background()
	if err := coordinator.Start(ctx); err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}
	defer coordinator.Stop(ctx)

	// Publish events in order
	eventIDs := []string{"event-1", "event-2", "event-3", "event-4", "event-5"}
	for _, id := range eventIDs {
		event := &saga.SagaEvent{
			ID:        id,
			SagaID:    "saga-003",
			Type:      saga.EventSagaStarted,
			Timestamp: time.Now(),
		}
		if err := publisher.PublishEvent(ctx, event); err != nil {
			t.Fatalf("Failed to publish event %s: %v", id, err)
		}
	}

	// Wait for processing
	time.Sleep(time.Second)

	// Verify order
	mu.Lock()
	defer mu.Unlock()
	if len(processedEvents) != len(eventIDs) {
		t.Errorf("Expected %d events, got %d", len(eventIDs), len(processedEvents))
	}
	for i, expectedID := range eventIDs {
		if i < len(processedEvents) && processedEvents[i] != expectedID {
			t.Errorf("Event order mismatch at index %d: expected %s, got %s", i, expectedID, processedEvents[i])
		}
	}
}

// TestChoreographyCoordinator_CompensationFlow tests compensation in choreography mode.
func TestChoreographyCoordinator_CompensationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 10,
		EventTimeout:          5 * time.Second,
	}

	coordinator, err := NewChoreographyCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	var compensationSteps []string
	var mu sync.Mutex

	// Handler that triggers compensation
	failureHandler := &testEventHandler{
		id:       "failure-trigger-handler",
		types:    []saga.SagaEventType{saga.EventSagaStepStarted},
		priority: 10,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			// Trigger failure event
			failureEvent := &saga.SagaEvent{
				ID:        "failure-event",
				SagaID:    event.SagaID,
				Type:      saga.EventSagaFailed,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"error": "simulated failure",
				},
			}
			return publisher.PublishEvent(ctx, failureEvent)
		},
	}

	// Compensation handler
	compensationHandler := &testEventHandler{
		id:       "compensation-handler",
		types:    []saga.SagaEventType{saga.EventSagaFailed},
		priority: 5,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			mu.Lock()
			compensationSteps = append(compensationSteps, "compensation-executed")
			mu.Unlock()

			// Publish compensated event
			compensatedEvent := &saga.SagaEvent{
				ID:        "compensated-event",
				SagaID:    event.SagaID,
				Type:      saga.EventCompensationCompleted,
				Timestamp: time.Now(),
			}
			return publisher.PublishEvent(ctx, compensatedEvent)
		},
	}

	if err := coordinator.RegisterEventHandler(failureHandler); err != nil {
		t.Fatalf("Failed to register failure handler: %v", err)
	}
	if err := coordinator.RegisterEventHandler(compensationHandler); err != nil {
		t.Fatalf("Failed to register compensation handler: %v", err)
	}

	ctx := context.Background()
	if err := coordinator.Start(ctx); err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}
	defer coordinator.Stop(ctx)

	// Trigger failure scenario
	startEvent := &saga.SagaEvent{
		ID:        "compensation-test-start",
		SagaID:    "saga-004",
		Type:      saga.EventSagaStepStarted,
		Timestamp: time.Now(),
	}

	if err := publisher.PublishEvent(ctx, startEvent); err != nil {
		t.Fatalf("Failed to publish start event: %v", err)
	}

	// Wait for compensation
	time.Sleep(2 * time.Second)

	// Verify compensation was executed
	mu.Lock()
	defer mu.Unlock()
	if len(compensationSteps) == 0 {
		t.Error("Expected compensation to be executed")
	}
}

// TestChoreographyCoordinator_ConcurrentSagas tests concurrent saga execution.
func TestChoreographyCoordinator_ConcurrentSagas(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 50,
		EventTimeout:          5 * time.Second,
	}

	coordinator, err := NewChoreographyCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	var processedSagas sync.Map

	handler := &testEventHandler{
		id:       "concurrent-handler",
		types:    []saga.SagaEventType{saga.EventSagaStarted},
		priority: 1,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			processedSagas.Store(event.SagaID, true)
			return nil
		},
	}

	if err := coordinator.RegisterEventHandler(handler); err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	ctx := context.Background()
	if err := coordinator.Start(ctx); err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}
	defer coordinator.Stop(ctx)

	// Start multiple sagas concurrently
	numSagas := 20
	var wg sync.WaitGroup
	wg.Add(numSagas)

	for i := 0; i < numSagas; i++ {
		go func(index int) {
			defer wg.Done()
			event := &saga.SagaEvent{
				ID:        "concurrent-event-" + string(rune('0'+index)),
				SagaID:    "saga-" + string(rune('A'+index)),
				Type:      saga.EventSagaStarted,
				Timestamp: time.Now(),
			}
			if err := publisher.PublishEvent(ctx, event); err != nil {
				t.Errorf("Failed to publish event %d: %v", index, err)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(2 * time.Second)

	// Verify all sagas were processed
	count := 0
	processedSagas.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	if count != numSagas {
		t.Errorf("Expected %d sagas to be processed, got %d", numSagas, count)
	}
}

// testEventHandler is a test implementation of ChoreographyEventHandler
type testEventHandler struct {
	id       string
	types    []saga.SagaEventType
	priority int
	handler  func(ctx context.Context, event *saga.SagaEvent) error
}

func (h *testEventHandler) HandleEvent(ctx context.Context, event *saga.SagaEvent) error {
	if h.handler != nil {
		return h.handler(ctx, event)
	}
	return nil
}

func (h *testEventHandler) GetHandlerID() string {
	return h.id
}

func (h *testEventHandler) GetSupportedEventTypes() []saga.SagaEventType {
	return h.types
}

func (h *testEventHandler) GetPriority() int {
	return h.priority
}

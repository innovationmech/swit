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

// TestE2E_OrderSaga_Choreography tests complete order Saga flow using choreography.
// This test simulates a real-world order processing workflow with multiple services:
// 1. Create order
// 2. Inventory service reserves stock
// 3. Payment service processes payment
// 4. Shipping service creates delivery
// 5. Verify final state
func TestE2E_OrderSaga_Choreography(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup
	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 10,
		EventTimeout:          10 * time.Second,
		EnableMetrics:         true,
	}

	coordinator, err := NewChoreographyCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Track saga state
	sagaState := &OrderSagaState{
		OrderID:           "ORDER-001",
		CustomerID:        "CUST-123",
		Amount:            100.0,
		InventoryReserved: false,
		PaymentProcessed:  false,
		ShippingCreated:   false,
		Completed:         false,
	}
	var stateMu sync.Mutex

	// Order Created Handler
	orderHandler := &testEventHandler{
		id:       "order-service",
		types:    []saga.SagaEventType{saga.EventSagaStarted},
		priority: 100,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			t.Logf("Order created: %s", sagaState.OrderID)

			// Publish inventory reservation request
			inventoryEvent := &saga.SagaEvent{
				ID:        "inventory-reserve-event",
				SagaID:    event.SagaID,
				Type:      saga.EventSagaStepStarted,
				StepID:    "inventory-reserve",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"order_id": sagaState.OrderID,
					"action":   "reserve-inventory",
				},
			}
			return publisher.PublishEvent(ctx, inventoryEvent)
		},
	}

	// Inventory Service Handler
	inventoryHandler := &testEventHandler{
		id:       "inventory-service",
		types:    []saga.SagaEventType{saga.EventSagaStepStarted},
		priority: 90,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			data := event.Data.(map[string]interface{})
			if action, ok := data["action"].(string); ok && action == "reserve-inventory" {
				stateMu.Lock()
				sagaState.InventoryReserved = true
				stateMu.Unlock()

				t.Log("Inventory reserved")

				// Publish inventory reserved event
				completedEvent := &saga.SagaEvent{
					ID:        "inventory-reserved-event",
					SagaID:    event.SagaID,
					Type:      saga.EventSagaStepCompleted,
					StepID:    "inventory-reserve",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"order_id": sagaState.OrderID,
						"action":   "process-payment",
					},
				}
				return publisher.PublishEvent(ctx, completedEvent)
			}
			return nil
		},
	}

	// Payment Service Handler
	paymentHandler := &testEventHandler{
		id:       "payment-service",
		types:    []saga.SagaEventType{saga.EventSagaStepCompleted},
		priority: 80,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			data := event.Data.(map[string]interface{})
			if action, ok := data["action"].(string); ok && action == "process-payment" {
				stateMu.Lock()
				sagaState.PaymentProcessed = true
				stateMu.Unlock()

				t.Log("Payment processed")

				// Publish payment completed event
				shippingEvent := &saga.SagaEvent{
					ID:        "shipping-create-event",
					SagaID:    event.SagaID,
					Type:      saga.EventSagaStepCompleted,
					StepID:    "payment-process",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"order_id": sagaState.OrderID,
						"action":   "create-shipping",
					},
				}
				return publisher.PublishEvent(ctx, shippingEvent)
			}
			return nil
		},
	}

	// Shipping Service Handler
	shippingHandler := &testEventHandler{
		id:       "shipping-service",
		types:    []saga.SagaEventType{saga.EventSagaStepCompleted},
		priority: 70,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			data := event.Data.(map[string]interface{})
			if action, ok := data["action"].(string); ok && action == "create-shipping" {
				stateMu.Lock()
				sagaState.ShippingCreated = true
				stateMu.Unlock()

				t.Log("Shipping created")

				// Publish saga completed event
				completedEvent := &saga.SagaEvent{
					ID:        "saga-completed-event",
					SagaID:    event.SagaID,
					Type:      saga.EventSagaCompleted,
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"order_id": sagaState.OrderID,
					},
				}
				return publisher.PublishEvent(ctx, completedEvent)
			}
			return nil
		},
	}

	// Final State Handler
	completionHandler := &testEventHandler{
		id:       "completion-service",
		types:    []saga.SagaEventType{saga.EventSagaCompleted},
		priority: 60,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			stateMu.Lock()
			sagaState.Completed = true
			stateMu.Unlock()

			t.Log("Saga completed successfully")
			return nil
		},
	}

	// Register all handlers
	handlers := []ChoreographyEventHandler{
		orderHandler,
		inventoryHandler,
		paymentHandler,
		shippingHandler,
		completionHandler,
	}
	for _, handler := range handlers {
		if err := coordinator.RegisterEventHandler(handler); err != nil{
			t.Fatalf("Failed to register handler: %v", err)
		}
	}

	// Start coordinator
	ctx := context.Background()
	if err := coordinator.Start(ctx); err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}
	defer coordinator.Stop(ctx)

	// Start the saga
	startEvent := &saga.SagaEvent{
		ID:        "order-saga-start",
		SagaID:    "order-saga-001",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"order_id":    sagaState.OrderID,
			"customer_id": sagaState.CustomerID,
			"amount":      sagaState.Amount,
		},
	}

	if err := publisher.PublishEvent(ctx, startEvent); err != nil {
		t.Fatalf("Failed to publish start event: %v", err)
	}

	// Wait for saga completion
	timeout := time.After(15 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Saga execution timed out")
		case <-ticker.C:
			stateMu.Lock()
			completed := sagaState.Completed
			stateMu.Unlock()

			if completed {
				// Verify final state
				stateMu.Lock()
				if !sagaState.InventoryReserved {
					t.Error("Expected inventory to be reserved")
				}
				if !sagaState.PaymentProcessed {
					t.Error("Expected payment to be processed")
				}
				if !sagaState.ShippingCreated {
					t.Error("Expected shipping to be created")
				}
				stateMu.Unlock()

				t.Log("All steps completed successfully")
				return
			}
		}
	}
}

// TestE2E_OrderSaga_CompensationFlow tests compensation flow when a step fails.
func TestE2E_OrderSaga_CompensationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 10,
		EventTimeout:          10 * time.Second,
	}

	coordinator, err := NewChoreographyCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Track compensation state
	var compensationSteps []string
	var compensationMu sync.Mutex

	// Order service
	orderHandler := &testEventHandler{
		id:       "order-service",
		types:    []saga.SagaEventType{saga.EventSagaStarted},
		priority: 100,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			// Request inventory reservation
			inventoryEvent := &saga.SagaEvent{
				ID:        "inventory-reserve-event",
				SagaID:    event.SagaID,
				Type:      saga.EventSagaStepStarted,
				StepID:    "inventory-reserve",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"action": "reserve-inventory",
				},
			}
			return publisher.PublishEvent(ctx, inventoryEvent)
		},
	}

	// Inventory service - succeeds
	inventoryHandler := &testEventHandler{
		id:       "inventory-service",
		types:    []saga.SagaEventType{saga.EventSagaStepStarted},
		priority: 90,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			data := event.Data.(map[string]interface{})
			if action, ok := data["action"].(string); ok && action == "reserve-inventory" {
				t.Log("Inventory reserved (will need compensation)")

				completedEvent := &saga.SagaEvent{
					ID:        "inventory-reserved-event",
					SagaID:    event.SagaID,
					Type:      saga.EventSagaStepCompleted,
					StepID:    "inventory-reserve",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"action": "process-payment",
					},
				}
				return publisher.PublishEvent(ctx, completedEvent)
			}
			return nil
		},
	}

	// Payment service - fails
	paymentHandler := &testEventHandler{
		id:       "payment-service",
		types:    []saga.SagaEventType{saga.EventSagaStepCompleted},
		priority: 80,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			data := event.Data.(map[string]interface{})
			if action, ok := data["action"].(string); ok && action == "process-payment" {
				t.Log("Payment failed - triggering compensation")

				// Trigger failure
				failureEvent := &saga.SagaEvent{
					ID:        "payment-failed-event",
					SagaID:    event.SagaID,
					Type:      saga.EventSagaFailed,
					StepID:    "payment-process",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"error":  "insufficient funds",
						"action": "compensate",
					},
				}
				return publisher.PublishEvent(ctx, failureEvent)
			}
			return nil
		},
	}

	// Compensation handler
	compensationHandler := &testEventHandler{
		id:       "compensation-service",
		types:    []saga.SagaEventType{saga.EventSagaFailed},
		priority: 70,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			data := event.Data.(map[string]interface{})
			if action, ok := data["action"].(string); ok && action == "compensate" {
				// Compensate inventory
				compensationMu.Lock()
				compensationSteps = append(compensationSteps, "release-inventory")
				compensationMu.Unlock()

				t.Log("Compensating: releasing inventory")

				compensatedEvent := &saga.SagaEvent{
					ID:        "saga-compensated-event",
					SagaID:    event.SagaID,
					Type:      saga.EventCompensationCompleted,
					Timestamp: time.Now(),
				}
				return publisher.PublishEvent(ctx, compensatedEvent)
			}
			return nil
		},
	}

	// Register handlers
	handlers := []ChoreographyEventHandler{
		orderHandler,
		inventoryHandler,
		paymentHandler,
		compensationHandler,
	}
	for _, handler := range handlers {
		if err := coordinator.RegisterEventHandler(handler); err != nil{
			t.Fatalf("Failed to register handler: %v", err)
		}
	}

	ctx := context.Background()
	if err := coordinator.Start(ctx); err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}
	defer coordinator.Stop(ctx)

	// Start saga
	startEvent := &saga.SagaEvent{
		ID:        "compensation-saga-start",
		SagaID:    "compensation-saga-001",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"order_id": "ORDER-002",
		},
	}

	if err := publisher.PublishEvent(ctx, startEvent); err != nil {
		t.Fatalf("Failed to publish start event: %v", err)
	}

	// Wait for compensation
	time.Sleep(5 * time.Second)

	// Verify compensation was executed
	compensationMu.Lock()
	defer compensationMu.Unlock()

	if len(compensationSteps) == 0 {
		t.Error("Expected compensation steps to be executed")
	}
	if compensationSteps[0] != "release-inventory" {
		t.Errorf("Expected first compensation step to be 'release-inventory', got '%s'", compensationSteps[0])
	}

	t.Log("Compensation flow completed successfully")
}

// TestE2E_MultiServiceConcurrent tests concurrent multi-service saga execution.
func TestE2E_MultiServiceConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 50,
		EventTimeout:          10 * time.Second,
	}

	coordinator, err := NewChoreographyCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Track completed sagas
	var completedSagas atomic.Int32

	// Simple handler that completes saga immediately
	handler := &testEventHandler{
		id:       "multi-saga-handler",
		types:    []saga.SagaEventType{saga.EventSagaStarted},
		priority: 100,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			// Complete saga immediately
			completedEvent := &saga.SagaEvent{
				ID:        event.ID + "-completed",
				SagaID:    event.SagaID,
				Type:      saga.EventSagaCompleted,
				Timestamp: time.Now(),
			}
			if err := publisher.PublishEvent(ctx, completedEvent); err != nil {
				return err
			}
			completedSagas.Add(1)
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
	numSagas := 50
	var wg sync.WaitGroup
	wg.Add(numSagas)

	for i := 0; i < numSagas; i++ {
		go func(index int) {
			defer wg.Done()

			sagaID := "concurrent-saga-" + string(rune('A'+index%26)) + string(rune('0'+index/26))
			startEvent := &saga.SagaEvent{
				ID:        sagaID + "-start",
				SagaID:    sagaID,
				Type:      saga.EventSagaStarted,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"index": index,
				},
			}

			if err := publisher.PublishEvent(ctx, startEvent); err != nil {
				t.Errorf("Failed to publish start event for saga %d: %v", index, err)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(5 * time.Second)

	// Verify all sagas completed
	completed := completedSagas.Load()
	if completed != int32(numSagas) {
		t.Errorf("Expected %d sagas to complete, got %d", numSagas, completed)
	}

	t.Logf("Successfully completed %d concurrent sagas", completed)
}

// TestE2E_TimeoutHandling tests timeout handling in saga execution.
func TestE2E_TimeoutHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	storage := NewInMemoryStateStorage()
	publisher := NewInMemoryEventPublisher()

	config := &ChoreographyConfig{
		EventPublisher:        publisher,
		StateStorage:          storage,
		MaxConcurrentHandlers: 10,
		EventTimeout:          2 * time.Second, // Short timeout
	}

	coordinator, err := NewChoreographyCoordinator(config)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coordinator.Close()

	// Track timeout
	var timedOut atomic.Bool

	// Slow handler that exceeds timeout
	slowHandler := &testEventHandler{
		id:       "slow-service",
		types:    []saga.SagaEventType{saga.EventSagaStarted},
		priority: 100,
		handler: func(ctx context.Context, event *saga.SagaEvent) error {
			// Simulate long operation
			select {
			case <-time.After(5 * time.Second):
				return nil
			case <-ctx.Done():
				timedOut.Store(true)
				t.Log("Handler timed out as expected")

				// Publish timeout event
				timeoutEvent := &saga.SagaEvent{
					ID:        "saga-timeout-event",
					SagaID:    event.SagaID,
					Type:      saga.EventSagaTimedOut,
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"error": ctx.Err().Error(),
					},
				}
				return publisher.PublishEvent(context.Background(), timeoutEvent)
			}
		},
	}

	if err := coordinator.RegisterEventHandler(slowHandler); err != nil{
		t.Fatalf("Failed to register handler: %v", err)
	}

	ctx := context.Background()
	if err := coordinator.Start(ctx); err != nil {
		t.Fatalf("Failed to start coordinator: %v", err)
	}
	defer coordinator.Stop(ctx)

	// Start saga
	startEvent := &saga.SagaEvent{
		ID:        "timeout-saga-start",
		SagaID:    "timeout-saga-001",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
	}

	if err := publisher.PublishEvent(ctx, startEvent); err != nil {
		t.Fatalf("Failed to publish start event: %v", err)
	}

	// Wait for timeout
	time.Sleep(4 * time.Second)

	// Verify timeout occurred
	if !timedOut.Load() {
		t.Error("Expected handler to timeout")
	}

	t.Log("Timeout handling verified successfully")
}

// OrderSagaState represents the state of an order saga
type OrderSagaState struct {
	OrderID           string
	CustomerID        string
	Amount            float64
	InventoryReserved bool
	PaymentProcessed  bool
	ShippingCreated   bool
	Completed         bool
}


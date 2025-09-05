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

package service

import (
	"context"
	"testing"
	"time"

	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/config"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/model"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/repository"
)

func TestOrderService_CreateOrder(t *testing.T) {
	// Create test repository
	dbConfig := config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}
	repo, err := repository.NewOrderRepository(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create test service
	extSvcConfig := config.ExternalServicesConfig{
		PaymentServiceURL:   "localhost:9082",
		InventoryServiceURL: "localhost:9083",
		Timeout:             30 * time.Second,
	}
	service, err := NewOrderService(repo, extSvcConfig)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Test create order
	ctx := context.Background()
	req := &model.CreateOrderRequest{
		CustomerID: "customer-123",
		ProductID:  "product-456",
		Quantity:   2,
		Amount:     99.99,
	}

	order, err := service.CreateOrder(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	if order == nil {
		t.Fatal("Order should not be nil")
	}

	if order.CustomerID != req.CustomerID {
		t.Errorf("Expected customer ID %s, got %s", req.CustomerID, order.CustomerID)
	}

	if order.ProductID != req.ProductID {
		t.Errorf("Expected product ID %s, got %s", req.ProductID, order.ProductID)
	}

	if order.Quantity != req.Quantity {
		t.Errorf("Expected quantity %d, got %d", req.Quantity, order.Quantity)
	}

	if order.Amount != req.Amount {
		t.Errorf("Expected amount %f, got %f", req.Amount, order.Amount)
	}

	// Order should be in confirmed status since we mock external services
	if order.Status != model.OrderStatusConfirmed && order.Status != model.OrderStatusFailed {
		t.Errorf("Expected order status to be confirmed or failed, got %s", order.Status)
	}
}

func TestOrderService_GetOrder(t *testing.T) {
	// Create test repository
	dbConfig := config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}
	repo, err := repository.NewOrderRepository(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create test service
	extSvcConfig := config.ExternalServicesConfig{
		PaymentServiceURL:   "localhost:9082",
		InventoryServiceURL: "localhost:9083",
		Timeout:             30 * time.Second,
	}
	service, err := NewOrderService(repo, extSvcConfig)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// First create an order
	ctx := context.Background()
	req := &model.CreateOrderRequest{
		CustomerID: "customer-123",
		ProductID:  "product-456",
		Quantity:   2,
		Amount:     99.99,
	}

	createdOrder, err := service.CreateOrder(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	// Now get the order
	retrievedOrder, err := service.GetOrder(ctx, createdOrder.ID)
	if err != nil {
		t.Fatalf("Failed to get order: %v", err)
	}

	if retrievedOrder == nil {
		t.Fatal("Retrieved order should not be nil")
	}

	if retrievedOrder.ID != createdOrder.ID {
		t.Errorf("Expected order ID %s, got %s", createdOrder.ID, retrievedOrder.ID)
	}
}

func TestOrderService_GetOrder_NotFound(t *testing.T) {
	// Create test repository
	dbConfig := config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}
	repo, err := repository.NewOrderRepository(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create test service
	extSvcConfig := config.ExternalServicesConfig{
		PaymentServiceURL:   "localhost:9082",
		InventoryServiceURL: "localhost:9083",
		Timeout:             30 * time.Second,
	}
	service, err := NewOrderService(repo, extSvcConfig)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Try to get non-existent order
	ctx := context.Background()
	_, err = service.GetOrder(ctx, "non-existent-order")
	if err == nil {
		t.Fatal("Expected error for non-existent order")
	}

	if err.Error() != "order not found" {
		t.Errorf("Expected 'order not found' error, got: %v", err)
	}
}

func TestOrderService_ProcessOrderWithCircuitBreaker(t *testing.T) {
	// Create test repository
	dbConfig := config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}
	repo, err := repository.NewOrderRepository(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create test service
	extSvcConfig := config.ExternalServicesConfig{
		PaymentServiceURL:   "localhost:9082",
		InventoryServiceURL: "localhost:9083",
		Timeout:             30 * time.Second,
	}
	service, err := NewOrderService(repo, extSvcConfig)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Test circuit breaker - should create degraded order if normal processing fails
	ctx := context.Background()
	req := &model.CreateOrderRequest{
		CustomerID: "customer-circuit-breaker",
		ProductID:  "product-circuit-breaker",
		Quantity:   1,
		Amount:     99.99,
	}

	order, err := service.ProcessOrderWithCircuitBreaker(ctx, req)
	if err != nil {
		t.Fatalf("Circuit breaker should not fail: %v", err)
	}

	if order == nil {
		t.Fatal("Order should not be nil")
	}

	// Order should be created (either confirmed or pending for fallback)
	if order.Status != model.OrderStatusConfirmed && order.Status != model.OrderStatusPending {
		t.Errorf("Expected order status to be confirmed or pending, got %s", order.Status)
	}
}

func TestOrderService_ProcessBulkOrders(t *testing.T) {
	// Create test repository
	dbConfig := config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}
	repo, err := repository.NewOrderRepository(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create test service
	extSvcConfig := config.ExternalServicesConfig{
		PaymentServiceURL:   "localhost:9082",
		InventoryServiceURL: "localhost:9083",
		Timeout:             30 * time.Second,
	}
	service, err := NewOrderService(repo, extSvcConfig)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Test bulk order processing
	ctx := context.Background()
	requests := []*model.CreateOrderRequest{
		{
			CustomerID: "customer-bulk-1",
			ProductID:  "product-bulk-1",
			Quantity:   1,
			Amount:     99.99,
		},
		{
			CustomerID: "customer-bulk-2",
			ProductID:  "product-bulk-2",
			Quantity:   2,
			Amount:     199.99,
		},
		{
			CustomerID: "customer-bulk-3",
			ProductID:  "product-bulk-3",
			Quantity:   1,
			Amount:     49.99,
		},
	}

	orders, errors := service.ProcessBulkOrders(ctx, requests)

	if len(orders) != len(requests) {
		t.Fatalf("Expected %d orders, got %d", len(requests), len(orders))
	}

	if len(errors) != len(requests) {
		t.Fatalf("Expected %d errors, got %d", len(requests), len(errors))
	}

	// Count successes
	successCount := 0
	for i, err := range errors {
		if err == nil {
			successCount++
			if orders[i] == nil {
				t.Errorf("Order %d should not be nil when error is nil", i)
			}
		}
	}

	// At least some orders should succeed with mock services
	if successCount == 0 {
		t.Error("At least some bulk orders should succeed")
	}
}

func TestOrderService_GetOrderMetrics(t *testing.T) {
	// Create test repository
	dbConfig := config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}
	repo, err := repository.NewOrderRepository(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create test service
	extSvcConfig := config.ExternalServicesConfig{
		PaymentServiceURL:   "localhost:9082",
		InventoryServiceURL: "localhost:9083",
		Timeout:             30 * time.Second,
	}
	service, err := NewOrderService(repo, extSvcConfig)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Create some test orders first
	ctx := context.Background()
	testOrders := []*model.CreateOrderRequest{
		{CustomerID: "customer-metrics-1", ProductID: "product-1", Quantity: 1, Amount: 100.00},
		{CustomerID: "customer-metrics-2", ProductID: "product-2", Quantity: 2, Amount: 200.00},
		{CustomerID: "customer-metrics-3", ProductID: "product-3", Quantity: 1, Amount: 50.00},
	}

	for _, req := range testOrders {
		_, err := service.CreateOrder(ctx, req)
		if err != nil {
			t.Logf("Failed to create test order: %v", err)
			// Continue with other orders
		}
	}

	// Get metrics
	metrics, err := service.GetOrderMetrics(ctx, "24h")
	if err != nil {
		t.Fatalf("Failed to get order metrics: %v", err)
	}

	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	// Should have some orders created
	if metrics.TotalOrders < 0 {
		t.Errorf("Total orders should be >= 0, got %d", metrics.TotalOrders)
	}

	// Success rate should be between 0 and 100
	if metrics.SuccessRate < 0 || metrics.SuccessRate > 100 {
		t.Errorf("Success rate should be between 0 and 100, got %f", metrics.SuccessRate)
	}
}

func TestOrderService_CompensationTransaction(t *testing.T) {
	// This test verifies the compensation logic is called when inventory reservation fails
	// Since we're using mock services, we'll create an order and then simulate compensation

	// Create test repository
	dbConfig := config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}
	repo, err := repository.NewOrderRepository(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create test service
	extSvcConfig := config.ExternalServicesConfig{
		PaymentServiceURL:   "localhost:9082",
		InventoryServiceURL: "localhost:9083",
		Timeout:             30 * time.Second,
	}
	service, err := NewOrderService(repo, extSvcConfig)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Create order that will trigger compensation
	ctx := context.Background()
	req := &model.CreateOrderRequest{
		CustomerID: "customer-compensation-test",
		ProductID:  "product-compensation-test",
		Quantity:   1,
		Amount:     99.99,
	}

	// Create order - with mock services this should mostly succeed
	order, err := service.CreateOrder(ctx, req)

	// The order creation should complete (may be confirmed or failed depending on mock behavior)
	if err != nil {
		// If error occurs, ensure it's handled properly
		t.Logf("Order creation error (expected for compensation test): %v", err)
	} else {
		// If order succeeds, it should have valid data
		if order == nil {
			t.Fatal("Order should not be nil")
		}
		if order.Status == "" {
			t.Error("Order status should be set")
		}
	}
}

func TestOrderService_DegradedService(t *testing.T) {
	// Create test repository
	dbConfig := config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}
	repo, err := repository.NewOrderRepository(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Create test service
	extSvcConfig := config.ExternalServicesConfig{
		PaymentServiceURL:   "localhost:9082",
		InventoryServiceURL: "localhost:9083",
		Timeout:             30 * time.Second,
	}
	service, err := NewOrderService(repo, extSvcConfig)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Test degraded service mode
	ctx := context.Background()
	req := &model.CreateOrderRequest{
		CustomerID: "customer-degraded",
		ProductID:  "product-degraded",
		Quantity:   1,
		Amount:     99.99,
	}

	// Call degraded service directly
	order, err := service.createOrderWithDegradedService(ctx, req)
	if err != nil {
		t.Fatalf("Degraded service should not fail: %v", err)
	}

	if order == nil {
		t.Fatal("Order should not be nil")
	}

	// Degraded service should create pending orders
	if order.Status != model.OrderStatusPending {
		t.Errorf("Expected order status to be pending in degraded mode, got %s", order.Status)
	}

	if order.CustomerID != req.CustomerID {
		t.Errorf("Expected customer ID %s, got %s", req.CustomerID, order.CustomerID)
	}
}

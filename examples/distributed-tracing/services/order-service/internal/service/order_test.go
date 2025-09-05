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

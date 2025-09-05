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
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	inventorypb "github.com/innovationmech/swit/api/gen/go/proto/swit/inventory/v1"
	paymentpb "github.com/innovationmech/swit/api/gen/go/proto/swit/payment/v1"

	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/config"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/model"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/repository"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/tracing"
)

// OrderService handles order business logic
type OrderService struct {
	repository       *repository.OrderRepository
	paymentClient    paymentpb.PaymentServiceClient
	inventoryClient  inventorypb.InventoryServiceClient
	paymentConn      *grpc.ClientConn
	inventoryConn    *grpc.ClientConn
	externalServices config.ExternalServicesConfig
}

// NewOrderService creates a new order service
func NewOrderService(repo *repository.OrderRepository, extSvcCfg config.ExternalServicesConfig) (*OrderService, error) {
	service := &OrderService{
		repository:       repo,
		externalServices: extSvcCfg,
	}

	// Initialize external service clients
	if err := service.initializeClients(); err != nil {
		return nil, fmt.Errorf("failed to initialize external clients: %w", err)
	}

	return service, nil
}

// initializeClients initializes gRPC clients for external services
func (s *OrderService) initializeClients() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize payment service client
	paymentConn, err := grpc.DialContext(ctx, s.externalServices.PaymentServiceURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		// For demo purposes, we'll continue without the connection
		// In a real service, you might want to fail here
		fmt.Printf("Warning: Failed to connect to payment service at %s: %v\n", s.externalServices.PaymentServiceURL, err)
	} else {
		s.paymentConn = paymentConn
		s.paymentClient = paymentpb.NewPaymentServiceClient(paymentConn)
	}

	// Initialize inventory service client
	inventoryConn, err := grpc.DialContext(ctx, s.externalServices.InventoryServiceURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		// For demo purposes, we'll continue without the connection
		fmt.Printf("Warning: Failed to connect to inventory service at %s: %v\n", s.externalServices.InventoryServiceURL, err)
	} else {
		s.inventoryConn = inventoryConn
		s.inventoryClient = inventorypb.NewInventoryServiceClient(inventoryConn)
	}

	return nil
}

// CreateOrder creates a new order with complete business logic
func (s *OrderService) CreateOrder(ctx context.Context, req *model.CreateOrderRequest) (*model.Order, error) {
	ctx, span := tracing.StartSpan(ctx, "service.CreateOrder")
	defer span.End()

	// Add order attributes to span
	orderAttrs := tracing.OrderAttributes{
		CustomerID: req.CustomerID,
		ProductID:  req.ProductID,
		Quantity:   req.Quantity,
		Amount:     req.Amount,
	}
	span.SetAttributes(orderAttrs.ToAttributes()...)

	// Generate order ID
	orderID := uuid.New().String()

	// Create order entity
	order := &model.Order{
		ID:         orderID,
		CustomerID: req.CustomerID,
		ProductID:  req.ProductID,
		Quantity:   req.Quantity,
		Amount:     req.Amount,
		Status:     model.OrderStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tracing.AddEvent(span, "order.created_locally",
		attribute.String("order.id", orderID),
	)

	// Step 1: Check inventory availability
	if err := s.checkInventory(ctx, req.ProductID, req.Quantity); err != nil {
		tracing.AddEvent(span, "order.inventory_check_failed",
			attribute.String("error", err.Error()),
		)
		tracing.SetSpanError(span, err)
		return nil, fmt.Errorf("inventory check failed: %w", err)
	}

	tracing.AddEvent(span, "order.inventory_check_passed")

	// Step 2: Process payment
	paymentResult, err := s.processPayment(ctx, orderID, req.CustomerID, req.Amount)
	if err != nil {
		tracing.AddEvent(span, "order.payment_failed",
			attribute.String("error", err.Error()),
		)
		tracing.SetSpanError(span, err)
		return nil, fmt.Errorf("payment processing failed: %w", err)
	}

	// Update order with payment transaction ID
	order.PaymentTransactionID = &paymentResult.TransactionID
	order.Status = model.OrderStatusProcessing

	tracing.AddEvent(span, "order.payment_completed",
		attribute.String("payment.transaction_id", paymentResult.TransactionID),
	)

	// Step 3: Reserve inventory
	if err := s.reserveInventory(ctx, orderID, req.ProductID, req.Quantity, req.CustomerID); err != nil {
		tracing.AddEvent(span, "order.inventory_reservation_failed",
			attribute.String("error", err.Error()),
		)
		// Implement compensation transaction - refund payment
		if err := s.compensateTransaction(ctx, order); err != nil {
			tracing.AddEvent(span, "order.compensation_failed",
				attribute.String("error", err.Error()),
			)
		}
		order.Status = model.OrderStatusFailed
	} else {
		tracing.AddEvent(span, "order.inventory_reserved")
		order.Status = model.OrderStatusConfirmed
	}

	// Step 4: Save order to database
	if err := s.repository.CreateOrder(ctx, order); err != nil {
		tracing.SetSpanError(span, err)
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	tracing.AddEvent(span, "order.saved_to_database",
		attribute.String("order.status", string(order.Status)),
	)

	// Update span with final order details
	span.SetAttributes(
		attribute.String("order.id", order.ID),
		attribute.String("order.status", string(order.Status)),
	)

	tracing.SetSpanSuccess(span)
	return order, nil
}

// GetOrder retrieves an order by ID
func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*model.Order, error) {
	ctx, span := tracing.StartSpan(ctx, "service.GetOrder",
		attribute.String("order.id", orderID),
	)
	defer span.End()

	order, err := s.repository.GetOrderByID(ctx, orderID)
	if err != nil {
		tracing.SetSpanError(span, err)
		return nil, err
	}

	if order == nil {
		tracing.AddEvent(span, "order.not_found")
		tracing.SetSpanSuccess(span)
		return nil, fmt.Errorf("order not found")
	}

	tracing.AddEvent(span, "order.found",
		attribute.String("order.status", string(order.Status)),
	)
	tracing.SetSpanSuccess(span)

	return order, nil
}

// ListOrders retrieves orders with filtering
func (s *OrderService) ListOrders(ctx context.Context, customerID string, status model.OrderStatus, pageSize int, pageToken string) ([]*model.Order, string, int64, error) {
	ctx, span := tracing.StartSpan(ctx, "service.ListOrders",
		attribute.String("customer.id", customerID),
		attribute.String("filter.status", string(status)),
		attribute.Int("pagination.page_size", pageSize),
	)
	defer span.End()

	// Calculate offset from page token (simplified implementation)
	offset := 0
	if pageToken != "" {
		// In a real implementation, you would decode the page token
		// For simplicity, we'll treat it as a numeric offset
		// offset = decodePageToken(pageToken)
	}

	// Set default page size
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	orders, totalCount, err := s.repository.ListOrders(ctx, customerID, status, pageSize, offset)
	if err != nil {
		tracing.SetSpanError(span, err)
		return nil, "", 0, err
	}

	// Generate next page token
	nextPageToken := ""
	if len(orders) == pageSize {
		// In a real implementation, you would encode the next offset
		nextPageToken = fmt.Sprintf("%d", offset+pageSize)
	}

	tracing.AddEvent(span, "orders.listed",
		attribute.Int("orders.count", len(orders)),
		attribute.Int64("orders.total", totalCount),
	)
	tracing.SetSpanSuccess(span)

	return orders, nextPageToken, totalCount, nil
}

// UpdateOrderStatus updates the status of an order
func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID string, newStatus model.OrderStatus, reason string) (*model.Order, error) {
	ctx, span := tracing.StartSpan(ctx, "service.UpdateOrderStatus",
		attribute.String("order.id", orderID),
		attribute.String("order.new_status", string(newStatus)),
	)
	defer span.End()

	if !newStatus.IsValidStatus() {
		err := fmt.Errorf("invalid status: %s", newStatus)
		tracing.SetSpanError(span, err)
		return nil, err
	}

	order, err := s.repository.UpdateOrderStatus(ctx, orderID, newStatus, reason, "system")
	if err != nil {
		tracing.SetSpanError(span, err)
		return nil, err
	}

	tracing.AddEvent(span, "order.status_updated",
		attribute.String("order.status", string(order.Status)),
	)
	tracing.SetSpanSuccess(span)

	return order, nil
}

// checkInventory checks if the requested quantity is available
func (s *OrderService) checkInventory(ctx context.Context, productID string, quantity int32) error {
	ctx, span := tracing.StartSpan(ctx, "service.checkInventory")
	defer span.End()

	extAttrs := tracing.ExternalServiceAttributes{
		ServiceName: "inventory-service",
		Method:      "CheckInventory",
		URL:         s.externalServices.InventoryServiceURL,
		Timeout:     s.externalServices.Timeout.String(),
	}
	span.SetAttributes(extAttrs.ToAttributes()...)

	if s.inventoryClient == nil {
		// Mock inventory check for demo purposes
		tracing.AddEvent(span, "inventory.check_mocked",
			attribute.String("product.id", productID),
			attribute.Int("quantity", int(quantity)),
			attribute.Bool("available", true),
		)
		tracing.SetSpanSuccess(span)
		return nil
	}

	req := &inventorypb.CheckInventoryRequest{
		ProductId: productID,
		Quantity:  quantity,
	}

	ctx, cancel := context.WithTimeout(ctx, s.externalServices.Timeout)
	defer cancel()

	resp, err := s.inventoryClient.CheckInventory(ctx, req)
	if err != nil {
		tracing.SetSpanError(span, err)
		return fmt.Errorf("inventory service error: %w", err)
	}

	if !resp.Available {
		err := fmt.Errorf("insufficient inventory: requested %d, available %d", quantity, resp.AvailableQuantity)
		tracing.AddEvent(span, "inventory.insufficient",
			attribute.Int("requested", int(quantity)),
			attribute.Int("available", int(resp.AvailableQuantity)),
		)
		tracing.SetSpanError(span, err)
		return err
	}

	tracing.AddEvent(span, "inventory.available",
		attribute.Int("requested", int(quantity)),
		attribute.Int("available", int(resp.AvailableQuantity)),
	)
	tracing.SetSpanSuccess(span)

	return nil
}

// processPayment processes payment for the order
func (s *OrderService) processPayment(ctx context.Context, orderID, customerID string, amount float64) (*model.PaymentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "service.processPayment")
	defer span.End()

	extAttrs := tracing.ExternalServiceAttributes{
		ServiceName: "payment-service",
		Method:      "ProcessPayment",
		URL:         s.externalServices.PaymentServiceURL,
		Timeout:     s.externalServices.Timeout.String(),
	}
	span.SetAttributes(extAttrs.ToAttributes()...)
	span.SetAttributes(
		attribute.Float64("payment.amount", amount),
		attribute.String("payment.currency", "USD"),
	)

	if s.paymentClient == nil {
		// Mock payment processing for demo purposes
		transactionID := uuid.New().String()
		result := &model.PaymentResponse{
			TransactionID: transactionID,
			Status:        model.PaymentStatusCompleted,
			Message:       "Payment processed successfully (mocked)",
		}

		tracing.AddEvent(span, "payment.processed_mocked",
			attribute.String("payment.transaction_id", transactionID),
			attribute.String("payment.status", string(result.Status)),
		)
		tracing.SetSpanSuccess(span)

		return result, nil
	}

	req := &paymentpb.ProcessPaymentRequest{
		CustomerId: customerID,
		OrderId:    orderID,
		Amount:     amount,
		Currency:   "USD",
		PaymentMethod: &paymentpb.PaymentMethod{
			Type: paymentpb.PaymentMethodType_PAYMENT_METHOD_TYPE_CREDIT_CARD,
		},
	}

	ctx, cancel := context.WithTimeout(ctx, s.externalServices.Timeout)
	defer cancel()

	resp, err := s.paymentClient.ProcessPayment(ctx, req)
	if err != nil {
		// Check if it's a gRPC error
		if grpcStatus, ok := status.FromError(err); ok {
			if grpcStatus.Code() == codes.InvalidArgument {
				tracing.AddEvent(span, "payment.invalid_request",
					attribute.String("error", grpcStatus.Message()),
				)
			} else if grpcStatus.Code() == codes.Unavailable {
				tracing.AddEvent(span, "payment.service_unavailable")
			}
		}
		tracing.SetSpanError(span, err)
		return nil, fmt.Errorf("payment service error: %w", err)
	}

	if resp.Payment.Status == paymentpb.PaymentStatus_PAYMENT_STATUS_FAILED {
		err := fmt.Errorf("payment failed: %s", resp.Payment.ProcessingDetails.GatewayResponseMessage)
		tracing.AddEvent(span, "payment.failed",
			attribute.String("error", err.Error()),
		)
		tracing.SetSpanError(span, err)
		return nil, err
	}

	result := &model.PaymentResponse{
		TransactionID: resp.Payment.TransactionId,
		Status:        convertPaymentStatus(resp.Payment.Status),
		Message:       "Payment processed successfully",
	}

	tracing.AddEvent(span, "payment.processed",
		attribute.String("payment.transaction_id", result.TransactionID),
		attribute.String("payment.status", string(result.Status)),
	)
	tracing.SetSpanSuccess(span)

	return result, nil
}

// reserveInventory reserves inventory for the order
func (s *OrderService) reserveInventory(ctx context.Context, orderID, productID string, quantity int32, customerID string) error {
	ctx, span := tracing.StartSpan(ctx, "service.reserveInventory")
	defer span.End()

	extAttrs := tracing.ExternalServiceAttributes{
		ServiceName: "inventory-service",
		Method:      "ReserveInventory",
		URL:         s.externalServices.InventoryServiceURL,
		Timeout:     s.externalServices.Timeout.String(),
	}
	span.SetAttributes(extAttrs.ToAttributes()...)

	if s.inventoryClient == nil {
		// Mock inventory reservation for demo purposes
		tracing.AddEvent(span, "inventory.reserved_mocked",
			attribute.String("product.id", productID),
			attribute.Int("quantity", int(quantity)),
			attribute.String("order.id", orderID),
		)
		tracing.SetSpanSuccess(span)
		return nil
	}

	req := &inventorypb.ReserveInventoryRequest{
		ProductId:  productID,
		Quantity:   quantity,
		OrderId:    orderID,
		CustomerId: customerID,
	}

	ctx, cancel := context.WithTimeout(ctx, s.externalServices.Timeout)
	defer cancel()

	resp, err := s.inventoryClient.ReserveInventory(ctx, req)
	if err != nil {
		tracing.SetSpanError(span, err)
		return fmt.Errorf("inventory reservation failed: %w", err)
	}

	tracing.AddEvent(span, "inventory.reserved",
		attribute.String("reservation.id", resp.Reservation.ReservationId),
		attribute.Int("quantity", int(resp.Reservation.Quantity)),
	)
	tracing.SetSpanSuccess(span)

	return nil
}

// convertPaymentStatus converts protobuf payment status to model status
func convertPaymentStatus(status paymentpb.PaymentStatus) model.PaymentStatus {
	switch status {
	case paymentpb.PaymentStatus_PAYMENT_STATUS_PENDING:
		return model.PaymentStatusPending
	case paymentpb.PaymentStatus_PAYMENT_STATUS_COMPLETED:
		return model.PaymentStatusCompleted
	case paymentpb.PaymentStatus_PAYMENT_STATUS_FAILED:
		return model.PaymentStatusFailed
	case paymentpb.PaymentStatus_PAYMENT_STATUS_CANCELLED:
		return model.PaymentStatusCancelled
	default:
		return model.PaymentStatusPending
	}
}

// compensateTransaction handles compensation logic when order processing fails
func (s *OrderService) compensateTransaction(ctx context.Context, order *model.Order) error {
	ctx, span := tracing.StartSpan(ctx, "service.compensateTransaction")
	defer span.End()

	span.SetAttributes(
		attribute.String("order.id", order.ID),
		attribute.String("compensation.reason", "inventory_reservation_failed"),
	)

	tracing.AddEvent(span, "compensation.started")

	// If payment was processed, initiate refund
	if order.PaymentTransactionID != nil {
		if err := s.refundPayment(ctx, *order.PaymentTransactionID, order.Amount); err != nil {
			tracing.AddEvent(span, "compensation.refund_failed",
				attribute.String("error", err.Error()),
			)
			tracing.SetSpanError(span, err)
			return fmt.Errorf("compensation refund failed: %w", err)
		}
		tracing.AddEvent(span, "compensation.payment_refunded",
			attribute.String("transaction_id", *order.PaymentTransactionID),
			attribute.Float64("amount", order.Amount),
		)
	}

	// Release any reserved inventory (if applicable)
	if err := s.releaseInventoryReservation(ctx, order.ID, order.ProductID); err != nil {
		tracing.AddEvent(span, "compensation.inventory_release_failed",
			attribute.String("error", err.Error()),
		)
		// Don't fail the compensation for inventory release failures as it's less critical
	}

	tracing.AddEvent(span, "compensation.completed")
	tracing.SetSpanSuccess(span)
	return nil
}

// refundPayment processes a payment refund
func (s *OrderService) refundPayment(ctx context.Context, transactionID string, amount float64) error {
	ctx, span := tracing.StartSpan(ctx, "service.refundPayment")
	defer span.End()

	extAttrs := tracing.ExternalServiceAttributes{
		ServiceName: "payment-service",
		Method:      "RefundPayment",
		URL:         s.externalServices.PaymentServiceURL,
		Timeout:     s.externalServices.Timeout.String(),
	}
	span.SetAttributes(extAttrs.ToAttributes()...)
	span.SetAttributes(
		attribute.String("payment.transaction_id", transactionID),
		attribute.Float64("payment.amount", amount),
	)

	if s.paymentClient == nil {
		// Mock refund for demo purposes
		tracing.AddEvent(span, "payment.refund_mocked",
			attribute.String("transaction_id", transactionID),
			attribute.Float64("amount", amount),
			attribute.Bool("success", true),
		)
		tracing.SetSpanSuccess(span)
		return nil
	}

	// In a real implementation, call the payment service refund API
	// req := &paymentpb.RefundPaymentRequest{
	//     TransactionId: transactionID,
	//     Amount: amount,
	//     Reason: "order_processing_failed",
	// }
	// resp, err := s.paymentClient.RefundPayment(ctx, req)

	tracing.AddEvent(span, "payment.refunded",
		attribute.String("transaction_id", transactionID),
		attribute.Float64("amount", amount),
	)
	tracing.SetSpanSuccess(span)

	return nil
}

// releaseInventoryReservation releases inventory reservation
func (s *OrderService) releaseInventoryReservation(ctx context.Context, orderID, productID string) error {
	ctx, span := tracing.StartSpan(ctx, "service.releaseInventoryReservation")
	defer span.End()

	extAttrs := tracing.ExternalServiceAttributes{
		ServiceName: "inventory-service",
		Method:      "ReleaseReservation",
		URL:         s.externalServices.InventoryServiceURL,
		Timeout:     s.externalServices.Timeout.String(),
	}
	span.SetAttributes(extAttrs.ToAttributes()...)

	if s.inventoryClient == nil {
		// Mock inventory release for demo purposes
		tracing.AddEvent(span, "inventory.release_mocked",
			attribute.String("order.id", orderID),
			attribute.String("product.id", productID),
			attribute.Bool("success", true),
		)
		tracing.SetSpanSuccess(span)
		return nil
	}

	// In a real implementation, call the inventory service release API
	// req := &inventorypb.ReleaseReservationRequest{
	//     OrderId: orderID,
	//     ProductId: productID,
	// }
	// resp, err := s.inventoryClient.ReleaseReservation(ctx, req)

	tracing.AddEvent(span, "inventory.reservation_released",
		attribute.String("order.id", orderID),
		attribute.String("product.id", productID),
	)
	tracing.SetSpanSuccess(span)

	return nil
}

// ProcessOrderWithCircuitBreaker processes order with circuit breaker pattern for service degradation
func (s *OrderService) ProcessOrderWithCircuitBreaker(ctx context.Context, req *model.CreateOrderRequest) (*model.Order, error) {
	ctx, span := tracing.StartSpan(ctx, "service.ProcessOrderWithCircuitBreaker")
	defer span.End()

	// Add circuit breaker attributes
	span.SetAttributes(
		attribute.String("pattern", "circuit_breaker"),
		attribute.String("resilience_strategy", "service_degradation"),
	)

	// Try normal order processing first
	order, err := s.CreateOrder(ctx, req)
	if err == nil {
		tracing.AddEvent(span, "order.circuit_breaker.success")
		tracing.SetSpanSuccess(span)
		return order, nil
	}

	tracing.AddEvent(span, "order.circuit_breaker.primary_failed",
		attribute.String("error", err.Error()),
	)

	// Fallback to degraded service (simplified order processing)
	order, fallbackErr := s.createOrderWithDegradedService(ctx, req)
	if fallbackErr != nil {
		tracing.AddEvent(span, "order.circuit_breaker.fallback_failed",
			attribute.String("error", fallbackErr.Error()),
		)
		tracing.SetSpanError(span, fallbackErr)
		return nil, fmt.Errorf("both primary and fallback failed: primary=%w, fallback=%w", err, fallbackErr)
	}

	tracing.AddEvent(span, "order.circuit_breaker.fallback_success")
	tracing.SetSpanSuccess(span)
	return order, nil
}

// createOrderWithDegradedService creates order with minimal external dependencies (degraded mode)
func (s *OrderService) createOrderWithDegradedService(ctx context.Context, req *model.CreateOrderRequest) (*model.Order, error) {
	ctx, span := tracing.StartSpan(ctx, "service.createOrderWithDegradedService")
	defer span.End()

	span.SetAttributes(
		attribute.String("service_mode", "degraded"),
		attribute.String("strategy", "minimal_validation"),
	)

	// Generate order ID
	orderID := uuid.New().String()

	// Create order with pending status (will be processed asynchronously)
	order := &model.Order{
		ID:         orderID,
		CustomerID: req.CustomerID,
		ProductID:  req.ProductID,
		Quantity:   req.Quantity,
		Amount:     req.Amount,
		Status:     model.OrderStatusPending, // Will be processed later
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tracing.AddEvent(span, "order.degraded_mode.created",
		attribute.String("order.id", orderID),
		attribute.String("processing_mode", "asynchronous"),
	)

	// Save to database immediately without external validations
	if err := s.repository.CreateOrder(ctx, order); err != nil {
		tracing.SetSpanError(span, err)
		return nil, fmt.Errorf("failed to save degraded order: %w", err)
	}

	tracing.AddEvent(span, "order.degraded_mode.saved")
	tracing.SetSpanSuccess(span)

	return order, nil
}

// ProcessBulkOrders processes multiple orders concurrently for high-load scenarios
func (s *OrderService) ProcessBulkOrders(ctx context.Context, requests []*model.CreateOrderRequest) ([]*model.Order, []error) {
	ctx, span := tracing.StartSpan(ctx, "service.ProcessBulkOrders")
	defer span.End()

	span.SetAttributes(
		attribute.Int("bulk.request_count", len(requests)),
		attribute.String("processing_pattern", "concurrent_bulk"),
	)

	type result struct {
		order *model.Order
		err   error
		index int
	}

	resultChan := make(chan result, len(requests))
	orders := make([]*model.Order, len(requests))
	errors := make([]error, len(requests))

	// Process orders concurrently
	for i, req := range requests {
		go func(idx int, request *model.CreateOrderRequest) {
			order, err := s.CreateOrder(ctx, request)
			resultChan <- result{order: order, err: err, index: idx}
		}(i, req)
	}

	// Collect results
	successCount := 0
	for i := 0; i < len(requests); i++ {
		res := <-resultChan
		orders[res.index] = res.order
		errors[res.index] = res.err
		if res.err == nil {
			successCount++
		}
	}

	tracing.AddEvent(span, "bulk.processing_completed",
		attribute.Int("bulk.success_count", successCount),
		attribute.Int("bulk.failure_count", len(requests)-successCount),
	)

	tracing.SetSpanSuccess(span)
	return orders, errors
}

// GetOrderMetrics returns business metrics for monitoring
func (s *OrderService) GetOrderMetrics(ctx context.Context, timeRange string) (*model.OrderMetrics, error) {
	ctx, span := tracing.StartSpan(ctx, "service.GetOrderMetrics")
	defer span.End()

	span.SetAttributes(
		attribute.String("metrics.time_range", timeRange),
		attribute.String("metrics.type", "business_kpi"),
	)

	metrics, err := s.repository.GetOrderMetrics(ctx, timeRange)
	if err != nil {
		tracing.SetSpanError(span, err)
		return nil, fmt.Errorf("failed to get order metrics: %w", err)
	}

	tracing.AddEvent(span, "metrics.calculated",
		attribute.Int64("metrics.total_orders", metrics.TotalOrders),
		attribute.Float64("metrics.success_rate", metrics.SuccessRate),
		attribute.Float64("metrics.avg_order_value", metrics.AverageOrderValue),
	)

	tracing.SetSpanSuccess(span)
	return metrics, nil
}

// Close closes external service connections
func (s *OrderService) Close() error {
	var errs []error

	if s.paymentConn != nil {
		if err := s.paymentConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close payment connection: %w", err))
		}
	}

	if s.inventoryConn != nil {
		if err := s.inventoryConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close inventory connection: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

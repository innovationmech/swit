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
		// TODO: Implement compensation - refund payment
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

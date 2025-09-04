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

package handler

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	orderpb "github.com/innovationmech/swit/api/gen/go/proto/swit/order/v1"

	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/model"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/service"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/tracing"
)

// OrderGRPCService implements the gRPC service interface
type OrderGRPCService struct {
	service *service.OrderService
	orderpb.UnimplementedOrderServiceServer
}

// NewOrderGRPCService creates a new gRPC service handler
func NewOrderGRPCService(service *service.OrderService) *OrderGRPCService {
	return &OrderGRPCService{
		service: service,
	}
}

// RegisterGRPC registers the gRPC service with the server
func (s *OrderGRPCService) RegisterGRPC(server interface{}) error {
	grpcServer, ok := server.(*grpc.Server)
	if !ok {
		return nil // Skip if not a gRPC server
	}

	orderpb.RegisterOrderServiceServer(grpcServer, s)
	return nil
}

// GetServiceName returns the service name
func (s *OrderGRPCService) GetServiceName() string {
	return "order-service-grpc"
}

// CreateOrder creates a new order
func (s *OrderGRPCService) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "grpc.CreateOrder")
	defer span.End()

	start := time.Now()

	// Validate request
	if req.CustomerId == "" {
		err := status.Error(codes.InvalidArgument, "customer_id is required")
		tracing.SetSpanError(span, err)
		return nil, err
	}
	if req.ProductId == "" {
		err := status.Error(codes.InvalidArgument, "product_id is required")
		tracing.SetSpanError(span, err)
		return nil, err
	}
	if req.Quantity <= 0 {
		err := status.Error(codes.InvalidArgument, "quantity must be greater than 0")
		tracing.SetSpanError(span, err)
		return nil, err
	}
	if req.Amount <= 0 {
		err := status.Error(codes.InvalidArgument, "amount must be greater than 0")
		tracing.SetSpanError(span, err)
		return nil, err
	}

	// Add request details to span
	span.SetAttributes(
		attribute.String("grpc.method", "CreateOrder"),
		attribute.String("order.customer_id", req.CustomerId),
		attribute.String("order.product_id", req.ProductId),
		attribute.Int("order.quantity", int(req.Quantity)),
		attribute.Float64("order.amount", req.Amount),
	)

	// Convert to service model
	serviceReq := &model.CreateOrderRequest{
		CustomerID: req.CustomerId,
		ProductID:  req.ProductId,
		Quantity:   req.Quantity,
		Amount:     req.Amount,
	}

	tracing.AddEvent(span, "request.validated")

	// Create order
	order, err := s.service.CreateOrder(ctx, serviceReq)
	if err != nil {
		tracing.SetSpanError(span, err)

		// Convert to appropriate gRPC status
		if err.Error() == "inventory check failed" {
			return nil, status.Error(codes.FailedPrecondition, "insufficient inventory")
		} else if err.Error() == "payment processing failed" {
			return nil, status.Error(codes.FailedPrecondition, "payment failed")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert to protobuf response
	pbOrder := convertOrderToProto(order)

	tracing.AddEvent(span, "order.created",
		attribute.String("order.id", order.ID),
		attribute.String("order.status", string(order.Status)),
	)

	// Track response time
	duration := time.Since(start)
	span.SetAttributes(
		attribute.Float64("grpc.request.duration", duration.Seconds()),
		attribute.String("grpc.status", "OK"),
	)

	tracing.SetSpanSuccess(span)

	return &orderpb.CreateOrderResponse{
		Order: pbOrder,
	}, nil
}

// GetOrder retrieves an order by ID
func (s *OrderGRPCService) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "grpc.GetOrder")
	defer span.End()

	// Validate request
	if req.OrderId == "" {
		err := status.Error(codes.InvalidArgument, "order_id is required")
		tracing.SetSpanError(span, err)
		return nil, err
	}

	span.SetAttributes(
		attribute.String("grpc.method", "GetOrder"),
		attribute.String("order.id", req.OrderId),
	)

	order, err := s.service.GetOrder(ctx, req.OrderId)
	if err != nil {
		tracing.SetSpanError(span, err)
		if err.Error() == "order not found" {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert to protobuf response
	pbOrder := convertOrderToProto(order)

	tracing.AddEvent(span, "order.retrieved",
		attribute.String("order.id", order.ID),
		attribute.String("order.status", string(order.Status)),
	)

	tracing.SetSpanSuccess(span)

	return &orderpb.GetOrderResponse{
		Order: pbOrder,
	}, nil
}

// ListOrders retrieves orders with filtering
func (s *OrderGRPCService) ListOrders(ctx context.Context, req *orderpb.ListOrdersRequest) (*orderpb.ListOrdersResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "grpc.ListOrders")
	defer span.End()

	span.SetAttributes(
		attribute.String("grpc.method", "ListOrders"),
		attribute.String("filter.customer_id", req.CustomerId),
		attribute.String("filter.status", req.Status.String()),
	)

	// Set default page size
	pageSize := int(req.Pagination.PageSize)
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	// Convert status from proto to model
	statusFilter := convertStatusFromProto(req.Status)

	orders, nextPageToken, totalCount, err := s.service.ListOrders(
		ctx,
		req.CustomerId,
		statusFilter,
		pageSize,
		req.Pagination.PageToken,
	)
	if err != nil {
		tracing.SetSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert to protobuf orders
	pbOrders := make([]*orderpb.Order, len(orders))
	for i, order := range orders {
		pbOrders[i] = convertOrderToProto(order)
	}

	tracing.AddEvent(span, "orders.listed",
		attribute.Int("orders.count", len(orders)),
		attribute.Int64("orders.total", totalCount),
	)

	tracing.SetSpanSuccess(span)

	return &orderpb.ListOrdersResponse{
		Orders: pbOrders,
		Pagination: &orderpb.PaginationResponse{
			NextPageToken: nextPageToken,
			TotalCount:    totalCount,
			PageSize:      int32(pageSize),
		},
	}, nil
}

// UpdateOrderStatus updates the status of an order
func (s *OrderGRPCService) UpdateOrderStatus(ctx context.Context, req *orderpb.UpdateOrderStatusRequest) (*orderpb.UpdateOrderStatusResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "grpc.UpdateOrderStatus")
	defer span.End()

	// Validate request
	if req.OrderId == "" {
		err := status.Error(codes.InvalidArgument, "order_id is required")
		tracing.SetSpanError(span, err)
		return nil, err
	}
	if req.Status == orderpb.OrderStatus_ORDER_STATUS_UNSPECIFIED {
		err := status.Error(codes.InvalidArgument, "status is required")
		tracing.SetSpanError(span, err)
		return nil, err
	}

	span.SetAttributes(
		attribute.String("grpc.method", "UpdateOrderStatus"),
		attribute.String("order.id", req.OrderId),
		attribute.String("order.new_status", req.Status.String()),
		attribute.String("order.reason", req.Reason),
	)

	// Convert status from proto to model
	newStatus := convertStatusFromProto(req.Status)

	order, err := s.service.UpdateOrderStatus(ctx, req.OrderId, newStatus, req.Reason)
	if err != nil {
		tracing.SetSpanError(span, err)
		if err.Error() == "order not found" {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert to protobuf response
	pbOrder := convertOrderToProto(order)

	tracing.AddEvent(span, "order.status_updated",
		attribute.String("order.id", order.ID),
		attribute.String("order.status", string(order.Status)),
	)

	tracing.SetSpanSuccess(span)

	return &orderpb.UpdateOrderStatusResponse{
		Order: pbOrder,
	}, nil
}

// convertOrderToProto converts a model.Order to protobuf Order
func convertOrderToProto(order *model.Order) *orderpb.Order {
	pbOrder := &orderpb.Order{
		Id:         order.ID,
		CustomerId: order.CustomerID,
		ProductId:  order.ProductID,
		Quantity:   order.Quantity,
		Amount:     order.Amount,
		Status:     convertStatusToProto(order.Status),
		CreatedAt:  timestamppb.New(order.CreatedAt),
		UpdatedAt:  timestamppb.New(order.UpdatedAt),
	}

	if order.PaymentTransactionID != nil {
		pbOrder.PaymentTransactionId = *order.PaymentTransactionID
	}

	// Convert status history
	if len(order.StatusHistory) > 0 {
		pbOrder.StatusHistory = make([]*orderpb.OrderStatusHistory, len(order.StatusHistory))
		for i, history := range order.StatusHistory {
			pbHistory := &orderpb.OrderStatusHistory{
				ToStatus:  convertStatusToProto(history.ToStatus),
				Reason:    history.Reason,
				ChangedAt: timestamppb.New(history.ChangedAt),
				ChangedBy: history.ChangedBy,
			}
			if history.FromStatus != nil {
				pbHistory.FromStatus = convertStatusToProto(*history.FromStatus)
			}
			pbOrder.StatusHistory[i] = pbHistory
		}
	}

	return pbOrder
}

// convertStatusToProto converts model.OrderStatus to protobuf OrderStatus
func convertStatusToProto(status model.OrderStatus) orderpb.OrderStatus {
	switch status {
	case model.OrderStatusPending:
		return orderpb.OrderStatus_ORDER_STATUS_PENDING
	case model.OrderStatusProcessing:
		return orderpb.OrderStatus_ORDER_STATUS_PROCESSING
	case model.OrderStatusConfirmed:
		return orderpb.OrderStatus_ORDER_STATUS_CONFIRMED
	case model.OrderStatusShipped:
		return orderpb.OrderStatus_ORDER_STATUS_SHIPPED
	case model.OrderStatusDelivered:
		return orderpb.OrderStatus_ORDER_STATUS_DELIVERED
	case model.OrderStatusCancelled:
		return orderpb.OrderStatus_ORDER_STATUS_CANCELLED
	case model.OrderStatusFailed:
		return orderpb.OrderStatus_ORDER_STATUS_FAILED
	case model.OrderStatusRefunded:
		return orderpb.OrderStatus_ORDER_STATUS_REFUNDED
	default:
		return orderpb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

// convertStatusFromProto converts protobuf OrderStatus to model.OrderStatus
func convertStatusFromProto(status orderpb.OrderStatus) model.OrderStatus {
	switch status {
	case orderpb.OrderStatus_ORDER_STATUS_PENDING:
		return model.OrderStatusPending
	case orderpb.OrderStatus_ORDER_STATUS_PROCESSING:
		return model.OrderStatusProcessing
	case orderpb.OrderStatus_ORDER_STATUS_CONFIRMED:
		return model.OrderStatusConfirmed
	case orderpb.OrderStatus_ORDER_STATUS_SHIPPED:
		return model.OrderStatusShipped
	case orderpb.OrderStatus_ORDER_STATUS_DELIVERED:
		return model.OrderStatusDelivered
	case orderpb.OrderStatus_ORDER_STATUS_CANCELLED:
		return model.OrderStatusCancelled
	case orderpb.OrderStatus_ORDER_STATUS_FAILED:
		return model.OrderStatusFailed
	case orderpb.OrderStatus_ORDER_STATUS_REFUNDED:
		return model.OrderStatusRefunded
	default:
		return ""
	}
}

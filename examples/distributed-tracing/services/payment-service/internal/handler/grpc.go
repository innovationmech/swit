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

	paymentpb "github.com/innovationmech/swit/api/gen/go/proto/swit/payment/v1"

	"github.com/innovationmech/swit/examples/distributed-tracing/services/payment-service/internal/model"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/payment-service/internal/service"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/payment-service/internal/tracing"
)

// PaymentGRPCService implements the gRPC service interface
type PaymentGRPCService struct {
	service *service.PaymentService
	paymentpb.UnimplementedPaymentServiceServer
}

// NewPaymentGRPCService creates a new gRPC service handler
func NewPaymentGRPCService(service *service.PaymentService) *PaymentGRPCService {
	return &PaymentGRPCService{
		service: service,
	}
}

// RegisterGRPC registers the gRPC service with the server
func (s *PaymentGRPCService) RegisterGRPC(server interface{}) error {
	grpcServer, ok := server.(*grpc.Server)
	if !ok {
		return nil // Skip if not a gRPC server
	}

	paymentpb.RegisterPaymentServiceServer(grpcServer, s)
	return nil
}

// GetServiceName returns the service name
func (s *PaymentGRPCService) GetServiceName() string {
	return "payment-service-grpc"
}

// ProcessPayment processes a payment
func (s *PaymentGRPCService) ProcessPayment(ctx context.Context, req *paymentpb.ProcessPaymentRequest) (*paymentpb.ProcessPaymentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "grpc.ProcessPayment")
	defer span.End()

	start := time.Now()

	// Validate request
	if req.CustomerId == "" {
		err := status.Error(codes.InvalidArgument, "customer_id is required")
		tracing.SetSpanError(span, err)
		return nil, err
	}
	if req.OrderId == "" {
		err := status.Error(codes.InvalidArgument, "order_id is required")
		tracing.SetSpanError(span, err)
		return nil, err
	}
	if req.Amount <= 0 {
		err := status.Error(codes.InvalidArgument, "amount must be greater than 0")
		tracing.SetSpanError(span, err)
		return nil, err
	}
	if req.PaymentMethod == nil {
		err := status.Error(codes.InvalidArgument, "payment_method is required")
		tracing.SetSpanError(span, err)
		return nil, err
	}

	// Add request details to span
	span.SetAttributes(
		attribute.String("grpc.method", "ProcessPayment"),
		attribute.String("payment.customer_id", req.CustomerId),
		attribute.String("payment.order_id", req.OrderId),
		attribute.Float64("payment.amount", req.Amount),
		attribute.String("payment.currency", req.Currency),
	)

	// Convert to service model
	serviceReq := &model.ProcessPaymentRequest{
		CustomerID:    req.CustomerId,
		OrderID:       req.OrderId,
		Amount:        req.Amount,
		Currency:      req.Currency,
		PaymentMethod: convertPaymentMethodFromProto(req.PaymentMethod),
	}

	tracing.AddEvent(span, "request.validated")

	// Process payment
	payment, err := s.service.ProcessPayment(ctx, serviceReq)
	if err != nil {
		tracing.SetSpanError(span, err)

		// Convert to appropriate gRPC status
		if err.Error() == "payment validation failed" {
			return nil, status.Error(codes.InvalidArgument, "payment validation failed")
		} else if err.Error() == "payment blocked due to high risk score" {
			return nil, status.Error(codes.FailedPrecondition, "payment blocked due to risk assessment")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert to protobuf response
	pbPayment := convertPaymentToProto(payment)

	tracing.AddEvent(span, "payment.processed",
		attribute.String("payment.transaction_id", payment.TransactionID),
		attribute.String("payment.status", string(payment.Status)),
	)

	// Track response time
	duration := time.Since(start)
	span.SetAttributes(
		attribute.Float64("grpc.request.duration", duration.Seconds()),
		attribute.String("grpc.status", "OK"),
	)

	tracing.SetSpanSuccess(span)

	return &paymentpb.ProcessPaymentResponse{
		Payment: pbPayment,
	}, nil
}

// ValidatePayment validates payment information
func (s *PaymentGRPCService) ValidatePayment(ctx context.Context, req *paymentpb.ValidatePaymentRequest) (*paymentpb.ValidatePaymentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "grpc.ValidatePayment")
	defer span.End()

	// Validate request
	if req.CustomerId == "" {
		err := status.Error(codes.InvalidArgument, "customer_id is required")
		tracing.SetSpanError(span, err)
		return nil, err
	}
	if req.PaymentMethod == nil {
		err := status.Error(codes.InvalidArgument, "payment_method is required")
		tracing.SetSpanError(span, err)
		return nil, err
	}

	span.SetAttributes(
		attribute.String("grpc.method", "ValidatePayment"),
		attribute.String("validation.customer_id", req.CustomerId),
		attribute.Float64("validation.amount", req.Amount),
		attribute.String("validation.currency", req.Currency),
	)

	// Convert to service model
	serviceReq := &model.ValidatePaymentRequest{
		CustomerID:    req.CustomerId,
		Amount:        req.Amount,
		Currency:      req.Currency,
		PaymentMethod: convertPaymentMethodFromProto(req.PaymentMethod),
	}

	response, err := s.service.ValidatePayment(ctx, serviceReq)
	if err != nil {
		tracing.SetSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert errors to protobuf ErrorDetail
	var pbErrors []*paymentpb.ErrorDetail
	if !response.IsValid {
		for _, errMsg := range response.Errors {
			pbErrors = append(pbErrors, &paymentpb.ErrorDetail{
				Code:    paymentpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
				Message: errMsg,
			})
		}
	}

	tracing.AddEvent(span, "payment.validated",
		attribute.Bool("validation.is_valid", response.IsValid),
		attribute.Int("validation.error_count", len(response.Errors)),
	)
	tracing.SetSpanSuccess(span)

	return &paymentpb.ValidatePaymentResponse{
		IsValid: response.IsValid,
		Errors:  pbErrors,
	}, nil
}

// GetPaymentStatus retrieves payment status
func (s *PaymentGRPCService) GetPaymentStatus(ctx context.Context, req *paymentpb.GetPaymentStatusRequest) (*paymentpb.GetPaymentStatusResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "grpc.GetPaymentStatus")
	defer span.End()

	// Validate request
	if req.TransactionId == "" {
		err := status.Error(codes.InvalidArgument, "transaction_id is required")
		tracing.SetSpanError(span, err)
		return nil, err
	}

	span.SetAttributes(
		attribute.String("grpc.method", "GetPaymentStatus"),
		attribute.String("payment.transaction_id", req.TransactionId),
	)

	payment, err := s.service.GetPaymentStatus(ctx, req.TransactionId)
	if err != nil {
		tracing.SetSpanError(span, err)
		if err.Error() == "payment not found" {
			return nil, status.Error(codes.NotFound, "payment not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert to protobuf response
	pbPayment := convertPaymentToProto(payment)

	tracing.AddEvent(span, "payment.status_retrieved",
		attribute.String("payment.transaction_id", payment.TransactionID),
		attribute.String("payment.status", string(payment.Status)),
	)
	tracing.SetSpanSuccess(span)

	return &paymentpb.GetPaymentStatusResponse{
		Payment: pbPayment,
	}, nil
}

// RefundPayment processes a refund
func (s *PaymentGRPCService) RefundPayment(ctx context.Context, req *paymentpb.RefundPaymentRequest) (*paymentpb.RefundPaymentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "grpc.RefundPayment")
	defer span.End()

	// Validate request
	if req.TransactionId == "" {
		err := status.Error(codes.InvalidArgument, "transaction_id is required")
		tracing.SetSpanError(span, err)
		return nil, err
	}
	if req.Amount <= 0 {
		err := status.Error(codes.InvalidArgument, "amount must be greater than 0")
		tracing.SetSpanError(span, err)
		return nil, err
	}

	span.SetAttributes(
		attribute.String("grpc.method", "RefundPayment"),
		attribute.String("refund.original_transaction_id", req.TransactionId),
		attribute.Float64("refund.amount", req.Amount),
		attribute.String("refund.reason", req.Reason),
	)

	// Convert to service model
	serviceReq := &model.RefundPaymentRequest{
		TransactionID: req.TransactionId,
		Amount:        req.Amount,
		Reason:        req.Reason,
	}

	refund, err := s.service.RefundPayment(ctx, serviceReq)
	if err != nil {
		tracing.SetSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert to protobuf response
	pbRefund := convertRefundToProto(refund)

	tracing.AddEvent(span, "refund.processed",
		attribute.String("refund.id", refund.RefundID),
		attribute.String("refund.status", string(refund.Status)),
	)
	tracing.SetSpanSuccess(span)

	return &paymentpb.RefundPaymentResponse{
		Refund: pbRefund,
	}, nil
}

// convertPaymentToProto converts a model.Payment to protobuf Payment
func convertPaymentToProto(payment *model.Payment) *paymentpb.Payment {
	return &paymentpb.Payment{
		TransactionId: payment.TransactionID,
		CustomerId:    payment.CustomerID,
		OrderId:       payment.OrderID,
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		Status:        convertPaymentStatusToProto(payment.Status),
		PaymentMethod: convertPaymentMethodToProto(payment.PaymentMethod),
		ProcessingDetails: &paymentpb.PaymentProcessingDetails{
			ProcessorId:            payment.ProcessingDetails.ProcessorID,
			ProcessorTransactionId: payment.ProcessingDetails.ProcessorTransactionID,
			ProcessingFee:          payment.ProcessingDetails.ProcessingFee,
			GatewayResponseCode:    payment.ProcessingDetails.GatewayResponseCode,
			GatewayResponseMessage: payment.ProcessingDetails.GatewayResponseMessage,
			RiskScore:              payment.ProcessingDetails.RiskScore,
		},
		CreatedAt: timestamppb.New(payment.CreatedAt),
		UpdatedAt: timestamppb.New(payment.UpdatedAt),
	}
}

// convertRefundToProto converts a model.PaymentRefund to protobuf PaymentRefund
func convertRefundToProto(refund *model.PaymentRefund) *paymentpb.PaymentRefund {
	return &paymentpb.PaymentRefund{
		RefundId:              refund.RefundID,
		OriginalTransactionId: refund.OriginalTransactionID,
		Amount:                refund.Amount,
		Currency:              refund.Currency,
		Reason:                refund.Reason,
		Status:                convertRefundStatusToProto(refund.Status),
		CreatedAt:             timestamppb.New(refund.CreatedAt),
	}
}

// convertPaymentStatusToProto converts model.PaymentStatus to protobuf PaymentStatus
func convertPaymentStatusToProto(status model.PaymentStatus) paymentpb.PaymentStatus {
	switch status {
	case model.PaymentStatusPending:
		return paymentpb.PaymentStatus_PAYMENT_STATUS_PENDING
	case model.PaymentStatusProcessing:
		return paymentpb.PaymentStatus_PAYMENT_STATUS_PROCESSING
	case model.PaymentStatusCompleted:
		return paymentpb.PaymentStatus_PAYMENT_STATUS_COMPLETED
	case model.PaymentStatusFailed:
		return paymentpb.PaymentStatus_PAYMENT_STATUS_FAILED
	case model.PaymentStatusCancelled:
		return paymentpb.PaymentStatus_PAYMENT_STATUS_CANCELLED
	case model.PaymentStatusRefunded:
		return paymentpb.PaymentStatus_PAYMENT_STATUS_REFUNDED
	case model.PaymentStatusPartiallyRefunded:
		return paymentpb.PaymentStatus_PAYMENT_STATUS_PARTIALLY_REFUNDED
	default:
		return paymentpb.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

// convertRefundStatusToProto converts model.RefundStatus to protobuf RefundStatus
func convertRefundStatusToProto(status model.RefundStatus) paymentpb.RefundStatus {
	switch status {
	case model.RefundStatusPending:
		return paymentpb.RefundStatus_REFUND_STATUS_PENDING
	case model.RefundStatusProcessing:
		return paymentpb.RefundStatus_REFUND_STATUS_PROCESSING
	case model.RefundStatusCompleted:
		return paymentpb.RefundStatus_REFUND_STATUS_COMPLETED
	case model.RefundStatusFailed:
		return paymentpb.RefundStatus_REFUND_STATUS_FAILED
	case model.RefundStatusCancelled:
		return paymentpb.RefundStatus_REFUND_STATUS_CANCELLED
	default:
		return paymentpb.RefundStatus_REFUND_STATUS_UNSPECIFIED
	}
}

// convertPaymentMethodFromProto converts protobuf PaymentMethod to model.PaymentMethod
func convertPaymentMethodFromProto(pm *paymentpb.PaymentMethod) model.PaymentMethod {
	method := model.PaymentMethod{
		Type: convertPaymentMethodTypeFromProto(pm.Type),
	}

	if pm.CardInfo != nil {
		method.CardInfo = &model.CardInfo{
			LastFourDigits: pm.CardInfo.LastFourDigits,
			Brand:          pm.CardInfo.Brand,
			ExpiryMonth:    pm.CardInfo.ExpiryMonth,
			ExpiryYear:     pm.CardInfo.ExpiryYear,
			CardholderName: pm.CardInfo.CardholderName,
		}
	}

	if pm.BankAccountInfo != nil {
		method.BankAccountInfo = &model.BankAccountInfo{
			BankName:          pm.BankAccountInfo.BankName,
			AccountHolderName: pm.BankAccountInfo.AccountHolderName,
			LastFourDigits:    pm.BankAccountInfo.LastFourDigits,
			RoutingNumber:     pm.BankAccountInfo.RoutingNumber,
		}
	}

	if pm.DigitalWalletInfo != nil {
		method.DigitalWalletInfo = &model.DigitalWalletInfo{
			Provider:  pm.DigitalWalletInfo.Provider,
			AccountID: pm.DigitalWalletInfo.AccountId,
		}
	}

	return method
}

// convertPaymentMethodToProto converts model.PaymentMethod to protobuf PaymentMethod
func convertPaymentMethodToProto(pm model.PaymentMethod) *paymentpb.PaymentMethod {
	method := &paymentpb.PaymentMethod{
		Type: convertPaymentMethodTypeToProto(pm.Type),
	}

	if pm.CardInfo != nil {
		method.CardInfo = &paymentpb.CardInfo{
			LastFourDigits: pm.CardInfo.LastFourDigits,
			Brand:          pm.CardInfo.Brand,
			ExpiryMonth:    pm.CardInfo.ExpiryMonth,
			ExpiryYear:     pm.CardInfo.ExpiryYear,
			CardholderName: pm.CardInfo.CardholderName,
		}
	}

	if pm.BankAccountInfo != nil {
		method.BankAccountInfo = &paymentpb.BankAccountInfo{
			BankName:          pm.BankAccountInfo.BankName,
			AccountHolderName: pm.BankAccountInfo.AccountHolderName,
			LastFourDigits:    pm.BankAccountInfo.LastFourDigits,
			RoutingNumber:     pm.BankAccountInfo.RoutingNumber,
		}
	}

	if pm.DigitalWalletInfo != nil {
		method.DigitalWalletInfo = &paymentpb.DigitalWalletInfo{
			Provider:  pm.DigitalWalletInfo.Provider,
			AccountId: pm.DigitalWalletInfo.AccountID,
		}
	}

	return method
}

// convertPaymentMethodTypeFromProto converts protobuf PaymentMethodType to model.PaymentMethodType
func convertPaymentMethodTypeFromProto(pmt paymentpb.PaymentMethodType) model.PaymentMethodType {
	switch pmt {
	case paymentpb.PaymentMethodType_PAYMENT_METHOD_TYPE_CREDIT_CARD:
		return model.PaymentMethodTypeCreditCard
	case paymentpb.PaymentMethodType_PAYMENT_METHOD_TYPE_DEBIT_CARD:
		return model.PaymentMethodTypeDebitCard
	case paymentpb.PaymentMethodType_PAYMENT_METHOD_TYPE_BANK_TRANSFER:
		return model.PaymentMethodTypeBankTransfer
	case paymentpb.PaymentMethodType_PAYMENT_METHOD_TYPE_DIGITAL_WALLET:
		return model.PaymentMethodTypeDigitalWallet
	case paymentpb.PaymentMethodType_PAYMENT_METHOD_TYPE_CASH_ON_DELIVERY:
		return model.PaymentMethodTypeCashOnDelivery
	default:
		return model.PaymentMethodTypeCreditCard // Default
	}
}

// convertPaymentMethodTypeToProto converts model.PaymentMethodType to protobuf PaymentMethodType
func convertPaymentMethodTypeToProto(pmt model.PaymentMethodType) paymentpb.PaymentMethodType {
	switch pmt {
	case model.PaymentMethodTypeCreditCard:
		return paymentpb.PaymentMethodType_PAYMENT_METHOD_TYPE_CREDIT_CARD
	case model.PaymentMethodTypeDebitCard:
		return paymentpb.PaymentMethodType_PAYMENT_METHOD_TYPE_DEBIT_CARD
	case model.PaymentMethodTypeBankTransfer:
		return paymentpb.PaymentMethodType_PAYMENT_METHOD_TYPE_BANK_TRANSFER
	case model.PaymentMethodTypeDigitalWallet:
		return paymentpb.PaymentMethodType_PAYMENT_METHOD_TYPE_DIGITAL_WALLET
	case model.PaymentMethodTypeCashOnDelivery:
		return paymentpb.PaymentMethodType_PAYMENT_METHOD_TYPE_CASH_ON_DELIVERY
	default:
		return paymentpb.PaymentMethodType_PAYMENT_METHOD_TYPE_UNSPECIFIED
	}
}

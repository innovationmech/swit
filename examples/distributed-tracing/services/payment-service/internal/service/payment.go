package service

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/innovationmech/swit/examples/distributed-tracing/services/payment-service/internal/config"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/payment-service/internal/model"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/payment-service/internal/tracing"
)

// PaymentService handles payment business logic
type PaymentService struct {
	config   config.PaymentConfig
	payments map[string]*model.Payment // In-memory storage for demo
	refunds  map[string]*model.PaymentRefund
	mutex    sync.RWMutex
	rng      *rand.Rand
}

// NewPaymentService creates a new payment service
func NewPaymentService(cfg config.PaymentConfig) *PaymentService {
	return &PaymentService{
		config:   cfg,
		payments: make(map[string]*model.Payment),
		refunds:  make(map[string]*model.PaymentRefund),
		mutex:    sync.RWMutex{},
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ProcessPayment processes a payment request
func (s *PaymentService) ProcessPayment(ctx context.Context, req *model.ProcessPaymentRequest) (*model.Payment, error) {
	ctx, span := tracing.StartSpan(ctx, "service.ProcessPayment")
	defer span.End()

	// Add payment attributes to span
	paymentAttrs := tracing.PaymentAttributes{
		CustomerID:    req.CustomerID,
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		PaymentMethod: string(req.PaymentMethod.Type),
	}
	span.SetAttributes(paymentAttrs.ToAttributes()...)

	// Generate transaction ID
	transactionID := uuid.New().String()

	tracing.AddEvent(span, "payment.transaction_generated",
		attribute.String("payment.transaction_id", transactionID),
	)

	// Create payment entity
	payment := &model.Payment{
		TransactionID: transactionID,
		CustomerID:    req.CustomerID,
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Status:        model.PaymentStatusPending,
		PaymentMethod: req.PaymentMethod,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		ProcessingDetails: model.PaymentProcessingDetails{
			ProcessorID: "demo-processor-001",
		},
	}

	// Store payment
	s.mutex.Lock()
	s.payments[transactionID] = payment
	s.mutex.Unlock()

	// Step 1: Validate payment information
	if err := s.validatePayment(ctx, req); err != nil {
		payment.Status = model.PaymentStatusFailed
		payment.ProcessingDetails.GatewayResponseCode = "VALIDATION_ERROR"
		payment.ProcessingDetails.GatewayResponseMessage = err.Error()
		payment.UpdatedAt = time.Now()

		tracing.AddEvent(span, "payment.validation_failed",
			attribute.String("error", err.Error()),
		)
		tracing.SetSpanError(span, err)
		return payment, fmt.Errorf("payment validation failed: %w", err)
	}

	tracing.AddEvent(span, "payment.validation_passed")

	// Step 2: Perform risk assessment
	riskScore := s.performRiskAssessment(ctx, req)
	payment.ProcessingDetails.RiskScore = riskScore

	tracing.AddEvent(span, "payment.risk_assessed",
		attribute.Int("payment.risk_score", int(riskScore)),
	)

	if riskScore > 80 {
		payment.Status = model.PaymentStatusFailed
		payment.ProcessingDetails.GatewayResponseCode = "HIGH_RISK"
		payment.ProcessingDetails.GatewayResponseMessage = "Transaction blocked due to high risk score"
		payment.UpdatedAt = time.Now()

		err := fmt.Errorf("payment blocked due to high risk score: %d", riskScore)
		tracing.AddEvent(span, "payment.blocked_high_risk",
			attribute.Int("risk_score", int(riskScore)),
		)
		tracing.SetSpanError(span, err)
		return payment, err
	}

	// Step 3: Process payment with external gateway (simulated)
	if err := s.processWithGateway(ctx, payment); err != nil {
		payment.Status = model.PaymentStatusFailed
		payment.UpdatedAt = time.Now()

		tracing.AddEvent(span, "payment.gateway_failed",
			attribute.String("error", err.Error()),
		)
		tracing.SetSpanError(span, err)
		return payment, fmt.Errorf("gateway processing failed: %w", err)
	}

	// Update final status
	payment.Status = model.PaymentStatusCompleted
	payment.UpdatedAt = time.Now()

	// Update span with final payment details
	span.SetAttributes(
		attribute.String("payment.transaction_id", payment.TransactionID),
		attribute.String("payment.status", string(payment.Status)),
		attribute.String("payment.gateway_code", payment.ProcessingDetails.GatewayResponseCode),
	)

	tracing.AddEvent(span, "payment.completed",
		attribute.String("payment.transaction_id", transactionID),
		attribute.String("payment.status", string(payment.Status)),
	)

	tracing.SetSpanSuccess(span)
	return payment, nil
}

// ValidatePayment validates payment information
func (s *PaymentService) ValidatePayment(ctx context.Context, req *model.ValidatePaymentRequest) (*model.ValidatePaymentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "service.ValidatePayment")
	defer span.End()

	span.SetAttributes(
		attribute.String("validation.customer_id", req.CustomerID),
		attribute.Float64("validation.amount", req.Amount),
		attribute.String("validation.currency", req.Currency),
	)

	var errors []string

	// Validate customer ID
	if req.CustomerID == "" {
		errors = append(errors, "customer_id is required")
	}

	// Validate amount
	if req.Amount <= 0 {
		errors = append(errors, "amount must be greater than 0")
	}
	if req.Amount > 10000 { // Demo limit
		errors = append(errors, "amount exceeds maximum limit of $10,000")
	}

	// Validate currency
	if req.Currency == "" {
		errors = append(errors, "currency is required")
	} else if req.Currency != "USD" && req.Currency != "EUR" && req.Currency != "GBP" {
		errors = append(errors, "unsupported currency")
	}

	// Validate payment method
	if req.PaymentMethod.Type == "" {
		errors = append(errors, "payment method type is required")
	}

	isValid := len(errors) == 0

	tracing.AddEvent(span, "payment.validation_completed",
		attribute.Bool("validation.is_valid", isValid),
		attribute.Int("validation.error_count", len(errors)),
	)

	if isValid {
		tracing.SetSpanSuccess(span)
	}

	return &model.ValidatePaymentResponse{
		IsValid: isValid,
		Errors:  errors,
	}, nil
}

// GetPaymentStatus retrieves payment status
func (s *PaymentService) GetPaymentStatus(ctx context.Context, transactionID string) (*model.Payment, error) {
	ctx, span := tracing.StartSpan(ctx, "service.GetPaymentStatus",
		attribute.String("payment.transaction_id", transactionID),
	)
	defer span.End()

	s.mutex.RLock()
	payment, exists := s.payments[transactionID]
	s.mutex.RUnlock()

	if !exists {
		err := fmt.Errorf("payment not found")
		tracing.AddEvent(span, "payment.not_found")
		tracing.SetSpanError(span, err)
		return nil, err
	}

	tracing.AddEvent(span, "payment.found",
		attribute.String("payment.status", string(payment.Status)),
	)
	tracing.SetSpanSuccess(span)

	return payment, nil
}

// RefundPayment processes a refund request
func (s *PaymentService) RefundPayment(ctx context.Context, req *model.RefundPaymentRequest) (*model.PaymentRefund, error) {
	ctx, span := tracing.StartSpan(ctx, "service.RefundPayment")
	defer span.End()

	refundAttrs := tracing.RefundAttributes{
		OriginalTransactionID: req.TransactionID,
		Amount:                req.Amount,
		Reason:                req.Reason,
	}
	span.SetAttributes(refundAttrs.ToAttributes()...)

	// Get original payment
	s.mutex.RLock()
	payment, exists := s.payments[req.TransactionID]
	s.mutex.RUnlock()

	if !exists {
		err := fmt.Errorf("original payment not found")
		tracing.SetSpanError(span, err)
		return nil, err
	}

	if payment.Status != model.PaymentStatusCompleted {
		err := fmt.Errorf("can only refund completed payments")
		tracing.SetSpanError(span, err)
		return nil, err
	}

	// Generate refund ID
	refundID := uuid.New().String()

	// Create refund
	refund := &model.PaymentRefund{
		RefundID:              refundID,
		OriginalTransactionID: req.TransactionID,
		Amount:                req.Amount,
		Currency:              payment.Currency,
		Reason:                req.Reason,
		Status:                model.RefundStatusProcessing,
		CreatedAt:             time.Now(),
	}

	// Simulate refund processing
	if s.config.SimulationMode {
		time.Sleep(time.Second) // Simulate processing time
		refund.Status = model.RefundStatusCompleted
	}

	// Store refund
	s.mutex.Lock()
	s.refunds[refundID] = refund

	// Update original payment status if full refund
	if req.Amount >= payment.Amount {
		payment.Status = model.PaymentStatusRefunded
	} else {
		payment.Status = model.PaymentStatusPartiallyRefunded
	}
	payment.UpdatedAt = time.Now()
	s.mutex.Unlock()

	tracing.AddEvent(span, "refund.completed",
		attribute.String("refund.id", refundID),
		attribute.String("refund.status", string(refund.Status)),
	)
	tracing.SetSpanSuccess(span)

	return refund, nil
}

// validatePayment performs payment validation
func (s *PaymentService) validatePayment(ctx context.Context, req *model.ProcessPaymentRequest) error {
	ctx, span := tracing.StartSpan(ctx, "service.validatePayment")
	defer span.End()

	processingAttrs := tracing.ProcessingAttributes{
		Step: "validation",
	}
	span.SetAttributes(processingAttrs.ToAttributes()...)

	// Basic validation
	if req.CustomerID == "" || req.OrderID == "" || req.Amount <= 0 || req.Currency == "" {
		err := fmt.Errorf("missing required fields")
		tracing.SetSpanError(span, err)
		return err
	}

	// Simulate validation processing time
	if s.config.SimulationMode {
		time.Sleep(100 * time.Millisecond)
	}

	tracing.SetSpanSuccess(span)
	return nil
}

// performRiskAssessment performs risk assessment for the payment
func (s *PaymentService) performRiskAssessment(ctx context.Context, req *model.ProcessPaymentRequest) int32 {
	ctx, span := tracing.StartSpan(ctx, "service.performRiskAssessment")
	defer span.End()

	processingAttrs := tracing.ProcessingAttributes{
		Step: "risk_assessment",
	}
	span.SetAttributes(processingAttrs.ToAttributes()...)

	// Simulate risk assessment
	var riskScore int32 = 10 // Base risk score

	// Higher amounts have higher risk
	if req.Amount > 1000 {
		riskScore += 20
	}
	if req.Amount > 5000 {
		riskScore += 30
	}

	// Add some randomness for demo purposes
	if s.config.SimulationMode {
		riskScore += int32(s.rng.Intn(40)) // Add 0-40 points randomly
		time.Sleep(200 * time.Millisecond) // Simulate processing time
	}

	tracing.AddEvent(span, "risk_assessment.completed",
		attribute.Int("risk_score", int(riskScore)),
	)
	tracing.SetSpanSuccess(span)

	return riskScore
}

// processWithGateway simulates payment processing with external gateway
func (s *PaymentService) processWithGateway(ctx context.Context, payment *model.Payment) error {
	ctx, span := tracing.StartSpan(ctx, "service.processWithGateway")
	defer span.End()

	processingAttrs := tracing.ProcessingAttributes{
		Step:        "gateway_processing",
		ProcessorID: payment.ProcessingDetails.ProcessorID,
	}
	span.SetAttributes(processingAttrs.ToAttributes()...)

	start := time.Now()

	// Simulate payment processing
	if s.config.SimulationMode {
		// Simulate processing time
		time.Sleep(s.config.ProcessingTime)

		// Simulate random failure based on failure rate
		if s.rng.Float64() < s.config.FailureRate {
			failureReasons := []string{
				"Insufficient funds",
				"Card expired",
				"Invalid card number",
				"Transaction declined by bank",
				"Network timeout",
			}

			reason := failureReasons[s.rng.Intn(len(failureReasons))]
			payment.ProcessingDetails.GatewayResponseCode = "DECLINED"
			payment.ProcessingDetails.GatewayResponseMessage = reason

			err := fmt.Errorf("payment declined: %s", reason)
			tracing.AddEvent(span, "gateway.payment_declined",
				attribute.String("decline_reason", reason),
			)
			tracing.SetSpanError(span, err)
			return err
		}

		// Success case
		payment.ProcessingDetails.ProcessorTransactionID = uuid.New().String()
		payment.ProcessingDetails.GatewayResponseCode = "SUCCESS"
		payment.ProcessingDetails.GatewayResponseMessage = "Payment approved"
		payment.ProcessingDetails.ProcessingFee = payment.Amount * 0.029 // 2.9% fee
		payment.ProcessingDetails.ProcessingTime = time.Since(start)
	}

	tracing.AddEvent(span, "gateway.payment_approved",
		attribute.String("gateway_code", payment.ProcessingDetails.GatewayResponseCode),
		attribute.String("processor_transaction_id", payment.ProcessingDetails.ProcessorTransactionID),
	)
	tracing.SetSpanSuccess(span)

	return nil
}

// HealthCheck performs a health check
func (s *PaymentService) HealthCheck(ctx context.Context) error {
	ctx, span := tracing.StartSpan(ctx, "service.HealthCheck")
	defer span.End()

	// Simple health check - verify service is responsive
	s.mutex.RLock()
	paymentCount := len(s.payments)
	s.mutex.RUnlock()

	tracing.AddEvent(span, "health_check.completed",
		attribute.Int("payments.count", paymentCount),
	)
	tracing.SetSpanSuccess(span)

	return nil
}

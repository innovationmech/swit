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

package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/innovationmech/swit/examples/distributed-tracing/services/payment-service/internal/config"
)

// InitTracing initializes OpenTelemetry tracing with Jaeger exporter
func InitTracing(cfg config.TracingConfig) (func(context.Context) error, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	// Create Jaeger exporter
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.JaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("service.component", "payment-service"),
			attribute.String("service.environment", "distributed-tracing-demo"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Return shutdown function
	return tp.Shutdown, nil
}

// GetTracer returns a tracer for the payment service
func GetTracer() trace.Tracer {
	return otel.Tracer("payment-service")
}

// StartSpan starts a new span with the given context and operation name
func StartSpan(ctx context.Context, operationName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := GetTracer()
	ctx, span := tracer.Start(ctx, operationName)

	// Add attributes if provided
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}

	return ctx, span
}

// AddEvent adds an event to the current span
func AddEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanError sets error status on the span
func SetSpanError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(trace.Status{
		Code:        trace.StatusCodeError,
		Description: err.Error(),
	})
}

// SetSpanSuccess sets success status on the span
func SetSpanSuccess(span trace.Span) {
	span.SetStatus(trace.Status{
		Code: trace.StatusCodeOk,
	})
}

// PaymentAttributes contains common payment-related tracing attributes
type PaymentAttributes struct {
	TransactionID string
	CustomerID    string
	OrderID       string
	Amount        float64
	Currency      string
	Status        string
	PaymentMethod string
}

// ToAttributes converts PaymentAttributes to OpenTelemetry attributes
func (pa PaymentAttributes) ToAttributes() []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 7)

	if pa.TransactionID != "" {
		attrs = append(attrs, attribute.String("payment.transaction_id", pa.TransactionID))
	}
	if pa.CustomerID != "" {
		attrs = append(attrs, attribute.String("payment.customer_id", pa.CustomerID))
	}
	if pa.OrderID != "" {
		attrs = append(attrs, attribute.String("payment.order_id", pa.OrderID))
	}
	if pa.Amount > 0 {
		attrs = append(attrs, attribute.Float64("payment.amount", pa.Amount))
	}
	if pa.Currency != "" {
		attrs = append(attrs, attribute.String("payment.currency", pa.Currency))
	}
	if pa.Status != "" {
		attrs = append(attrs, attribute.String("payment.status", pa.Status))
	}
	if pa.PaymentMethod != "" {
		attrs = append(attrs, attribute.String("payment.method", pa.PaymentMethod))
	}

	return attrs
}

// ProcessingAttributes contains attributes for payment processing steps
type ProcessingAttributes struct {
	Step           string
	ProcessorID    string
	RiskScore      int32
	ProcessingTime string
	GatewayCode    string
}

// ToAttributes converts ProcessingAttributes to OpenTelemetry attributes
func (pa ProcessingAttributes) ToAttributes() []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 5)

	if pa.Step != "" {
		attrs = append(attrs, attribute.String("payment.processing.step", pa.Step))
	}
	if pa.ProcessorID != "" {
		attrs = append(attrs, attribute.String("payment.processor.id", pa.ProcessorID))
	}
	if pa.RiskScore > 0 {
		attrs = append(attrs, attribute.Int("payment.risk_score", int(pa.RiskScore)))
	}
	if pa.ProcessingTime != "" {
		attrs = append(attrs, attribute.String("payment.processing.time", pa.ProcessingTime))
	}
	if pa.GatewayCode != "" {
		attrs = append(attrs, attribute.String("payment.gateway.code", pa.GatewayCode))
	}

	return attrs
}

// RefundAttributes contains attributes for refund operations
type RefundAttributes struct {
	RefundID              string
	OriginalTransactionID string
	Amount                float64
	Currency              string
	Reason                string
	Status                string
}

// ToAttributes converts RefundAttributes to OpenTelemetry attributes
func (ra RefundAttributes) ToAttributes() []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 6)

	if ra.RefundID != "" {
		attrs = append(attrs, attribute.String("refund.id", ra.RefundID))
	}
	if ra.OriginalTransactionID != "" {
		attrs = append(attrs, attribute.String("refund.original_transaction_id", ra.OriginalTransactionID))
	}
	if ra.Amount > 0 {
		attrs = append(attrs, attribute.Float64("refund.amount", ra.Amount))
	}
	if ra.Currency != "" {
		attrs = append(attrs, attribute.String("refund.currency", ra.Currency))
	}
	if ra.Reason != "" {
		attrs = append(attrs, attribute.String("refund.reason", ra.Reason))
	}
	if ra.Status != "" {
		attrs = append(attrs, attribute.String("refund.status", ra.Status))
	}

	return attrs
}

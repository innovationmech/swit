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

	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/config"
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
			attribute.String("service.component", "order-service"),
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

// GetTracer returns a tracer for the order service
func GetTracer() trace.Tracer {
	return otel.Tracer("order-service")
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

// OrderAttributes contains common order-related tracing attributes
type OrderAttributes struct {
	OrderID    string
	CustomerID string
	ProductID  string
	Quantity   int32
	Amount     float64
	Status     string
}

// ToAttributes converts OrderAttributes to OpenTelemetry attributes
func (oa OrderAttributes) ToAttributes() []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 6)

	if oa.OrderID != "" {
		attrs = append(attrs, attribute.String("order.id", oa.OrderID))
	}
	if oa.CustomerID != "" {
		attrs = append(attrs, attribute.String("order.customer_id", oa.CustomerID))
	}
	if oa.ProductID != "" {
		attrs = append(attrs, attribute.String("order.product_id", oa.ProductID))
	}
	if oa.Quantity > 0 {
		attrs = append(attrs, attribute.Int("order.quantity", int(oa.Quantity)))
	}
	if oa.Amount > 0 {
		attrs = append(attrs, attribute.Float64("order.amount", oa.Amount))
	}
	if oa.Status != "" {
		attrs = append(attrs, attribute.String("order.status", oa.Status))
	}

	return attrs
}

// ExternalServiceAttributes contains attributes for external service calls
type ExternalServiceAttributes struct {
	ServiceName string
	Method      string
	URL         string
	Timeout     string
}

// ToAttributes converts ExternalServiceAttributes to OpenTelemetry attributes
func (esa ExternalServiceAttributes) ToAttributes() []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 4)

	if esa.ServiceName != "" {
		attrs = append(attrs, attribute.String("external.service.name", esa.ServiceName))
	}
	if esa.Method != "" {
		attrs = append(attrs, attribute.String("external.method", esa.Method))
	}
	if esa.URL != "" {
		attrs = append(attrs, attribute.String("external.url", esa.URL))
	}
	if esa.Timeout != "" {
		attrs = append(attrs, attribute.String("external.timeout", esa.Timeout))
	}

	return attrs
}

// DatabaseAttributes contains attributes for database operations
type DatabaseAttributes struct {
	Operation string
	Table     string
	Query     string
}

// ToAttributes converts DatabaseAttributes to OpenTelemetry attributes
func (da DatabaseAttributes) ToAttributes() []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 3)

	if da.Operation != "" {
		attrs = append(attrs, attribute.String("db.operation", da.Operation))
	}
	if da.Table != "" {
		attrs = append(attrs, attribute.String("db.table", da.Table))
	}
	if da.Query != "" {
		attrs = append(attrs, attribute.String("db.statement", da.Query))
	}

	return attrs
}

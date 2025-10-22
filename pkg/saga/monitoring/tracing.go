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

// Package monitoring provides distributed tracing integration for Saga execution.
// It supports OpenTelemetry, Jaeger, and Zipkin exporters with full context propagation.
package monitoring

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/innovationmech/swit/pkg/tracing"
)

// SagaTracingManager manages distributed tracing for Saga execution.
// It integrates with OpenTelemetry and supports Jaeger and Zipkin exporters.
type SagaTracingManager struct {
	tracingManager tracing.TracingManager
	enabled        bool
}

// SagaTracingConfig holds configuration for Saga distributed tracing.
type SagaTracingConfig struct {
	// Enabled determines whether tracing is active
	Enabled bool

	// TracingManager is the underlying tracing implementation
	TracingManager tracing.TracingManager
}

// NewSagaTracingManager creates a new Saga tracing manager with the given configuration.
//
// Parameters:
//   - config: Configuration for the tracing manager. If nil, tracing will be disabled.
//
// Returns:
//   - A configured SagaTracingManager ready to trace Saga operations.
//
// Example:
//
//	tm := tracing.NewTracingManager()
//	if err := tm.Initialize(ctx, tracingConfig); err != nil {
//	    return err
//	}
//	sagaTracing := NewSagaTracingManager(&SagaTracingConfig{
//	    Enabled:        true,
//	    TracingManager: tm,
//	})
func NewSagaTracingManager(config *SagaTracingConfig) *SagaTracingManager {
	if config == nil {
		return &SagaTracingManager{enabled: false}
	}

	return &SagaTracingManager{
		tracingManager: config.TracingManager,
		enabled:        config.Enabled && config.TracingManager != nil,
	}
}

// StartSagaSpan starts a new span for a Saga execution.
// This is the top-level span for the entire Saga lifecycle.
//
// Parameters:
//   - ctx: Context for the span
//   - sagaID: Unique identifier for the Saga
//   - sagaType: Type/name of the Saga (e.g., "OrderSaga", "PaymentSaga")
//
// Returns:
//   - A new context with the span attached
//   - The created span (use defer span.End() to complete it)
//
// Example:
//
//	ctx, span := sagaTracing.StartSagaSpan(ctx, sagaID, "OrderSaga")
//	defer span.End()
func (stm *SagaTracingManager) StartSagaSpan(ctx context.Context, sagaID string, sagaType string) (context.Context, tracing.Span) {
	if !stm.enabled {
		return ctx, &noOpSpan{}
	}

	return stm.tracingManager.StartSpan(
		ctx,
		fmt.Sprintf("saga.%s", sagaType),
		tracing.WithSpanKind(oteltrace.SpanKindInternal),
		tracing.WithAttributes(
			attribute.String("saga.id", sagaID),
			attribute.String("saga.type", sagaType),
			attribute.String("saga.phase", "execution"),
		),
	)
}

// StartStepSpan starts a new span for a Saga step execution.
// This span is a child of the Saga span and represents a single step.
//
// Parameters:
//   - ctx: Context containing the parent Saga span
//   - sagaID: Unique identifier for the Saga
//   - stepName: Name of the step being executed
//   - stepType: Type of step ("forward" or "compensate")
//
// Returns:
//   - A new context with the span attached
//   - The created span (use defer span.End() to complete it)
//
// Example:
//
//	ctx, span := sagaTracing.StartStepSpan(ctx, sagaID, "CreateOrder", "forward")
//	defer span.End()
func (stm *SagaTracingManager) StartStepSpan(ctx context.Context, sagaID string, stepName string, stepType string) (context.Context, tracing.Span) {
	if !stm.enabled {
		return ctx, &noOpSpan{}
	}

	return stm.tracingManager.StartSpan(
		ctx,
		fmt.Sprintf("saga.step.%s", stepName),
		tracing.WithSpanKind(oteltrace.SpanKindInternal),
		tracing.WithAttributes(
			attribute.String("saga.id", sagaID),
			attribute.String("saga.step.name", stepName),
			attribute.String("saga.step.type", stepType),
		),
	)
}

// RecordSagaEvent records an event in the current Saga span.
// Events represent important occurrences during Saga execution.
//
// Parameters:
//   - ctx: Context containing the current span
//   - eventName: Name of the event (e.g., "saga.started", "saga.completed")
//   - attributes: Additional attributes for the event
//
// Example:
//
//	sagaTracing.RecordSagaEvent(ctx, "saga.started", map[string]interface{}{
//	    "saga.initial_state": "pending",
//	})
func (stm *SagaTracingManager) RecordSagaEvent(ctx context.Context, eventName string, attributes map[string]interface{}) {
	if !stm.enabled {
		return
	}

	span := stm.tracingManager.SpanFromContext(ctx)
	if span == nil {
		return
	}

	// Convert map to attribute key-values
	attrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		attrs = append(attrs, convertToAttribute(k, v))
	}

	span.AddEvent(eventName, oteltrace.WithAttributes(attrs...))
}

// RecordSagaSuccess marks the current Saga span as successful.
// This should be called when the Saga completes successfully.
//
// Parameters:
//   - ctx: Context containing the current span
//   - sagaID: Unique identifier for the Saga
//
// Example:
//
//	sagaTracing.RecordSagaSuccess(ctx, sagaID)
func (stm *SagaTracingManager) RecordSagaSuccess(ctx context.Context, sagaID string) {
	if !stm.enabled {
		return
	}

	span := stm.tracingManager.SpanFromContext(ctx)
	if span == nil {
		return
	}

	span.SetStatus(codes.Ok, "Saga completed successfully")
	span.SetAttribute("saga.status", "completed")
	span.AddEvent("saga.completed", oteltrace.WithAttributes(
		attribute.String("saga.id", sagaID),
		attribute.String("saga.result", "success"),
	))
}

// RecordSagaFailure marks the current Saga span as failed.
// This should be called when the Saga fails and cannot complete.
//
// Parameters:
//   - ctx: Context containing the current span
//   - sagaID: Unique identifier for the Saga
//   - err: The error that caused the failure
//   - reason: Human-readable reason for the failure
//
// Example:
//
//	sagaTracing.RecordSagaFailure(ctx, sagaID, err, "payment processing failed")
func (stm *SagaTracingManager) RecordSagaFailure(ctx context.Context, sagaID string, err error, reason string) {
	if !stm.enabled {
		return
	}

	span := stm.tracingManager.SpanFromContext(ctx)
	if span == nil {
		return
	}

	span.SetStatus(codes.Error, reason)
	span.SetAttribute("saga.status", "failed")
	span.SetAttribute("saga.failure.reason", reason)

	if err != nil {
		span.RecordError(err, oteltrace.WithAttributes(
			attribute.String("saga.id", sagaID),
			attribute.String("error.type", fmt.Sprintf("%T", err)),
		))
	}

	span.AddEvent("saga.failed", oteltrace.WithAttributes(
		attribute.String("saga.id", sagaID),
		attribute.String("saga.result", "failure"),
		attribute.String("saga.failure.reason", reason),
	))
}

// RecordSagaCompensated marks the current Saga span as compensated.
// This should be called when the Saga successfully rolls back after a failure.
//
// Parameters:
//   - ctx: Context containing the current span
//   - sagaID: Unique identifier for the Saga
//   - reason: Reason why compensation was needed
//
// Example:
//
//	sagaTracing.RecordSagaCompensated(ctx, sagaID, "payment failed, inventory released")
func (stm *SagaTracingManager) RecordSagaCompensated(ctx context.Context, sagaID string, reason string) {
	if !stm.enabled {
		return
	}

	span := stm.tracingManager.SpanFromContext(ctx)
	if span == nil {
		return
	}

	span.SetStatus(codes.Ok, "Saga compensated successfully")
	span.SetAttribute("saga.status", "compensated")
	span.SetAttribute("saga.compensation.reason", reason)
	span.AddEvent("saga.compensated", oteltrace.WithAttributes(
		attribute.String("saga.id", sagaID),
		attribute.String("saga.result", "compensated"),
		attribute.String("saga.compensation.reason", reason),
	))
}

// RecordStepSuccess marks the current step span as successful.
//
// Parameters:
//   - ctx: Context containing the current span
//   - stepName: Name of the step
//
// Example:
//
//	sagaTracing.RecordStepSuccess(ctx, "CreateOrder")
func (stm *SagaTracingManager) RecordStepSuccess(ctx context.Context, stepName string) {
	if !stm.enabled {
		return
	}

	span := stm.tracingManager.SpanFromContext(ctx)
	if span == nil {
		return
	}

	span.SetStatus(codes.Ok, "Step completed successfully")
	span.SetAttribute("saga.step.status", "success")
	span.AddEvent("saga.step.completed", oteltrace.WithAttributes(
		attribute.String("saga.step.name", stepName),
		attribute.String("saga.step.result", "success"),
	))
}

// RecordStepFailure marks the current step span as failed.
//
// Parameters:
//   - ctx: Context containing the current span
//   - stepName: Name of the step
//   - err: The error that caused the failure
//   - reason: Human-readable reason for the failure
//
// Example:
//
//	sagaTracing.RecordStepFailure(ctx, "ProcessPayment", err, "insufficient funds")
func (stm *SagaTracingManager) RecordStepFailure(ctx context.Context, stepName string, err error, reason string) {
	if !stm.enabled {
		return
	}

	span := stm.tracingManager.SpanFromContext(ctx)
	if span == nil {
		return
	}

	span.SetStatus(codes.Error, reason)
	span.SetAttribute("saga.step.status", "failed")
	span.SetAttribute("saga.step.failure.reason", reason)

	if err != nil {
		span.RecordError(err, oteltrace.WithAttributes(
			attribute.String("saga.step.name", stepName),
			attribute.String("error.type", fmt.Sprintf("%T", err)),
		))
	}

	span.AddEvent("saga.step.failed", oteltrace.WithAttributes(
		attribute.String("saga.step.name", stepName),
		attribute.String("saga.step.result", "failure"),
		attribute.String("saga.step.failure.reason", reason),
	))
}

// InjectContext injects tracing context into a carrier (e.g., message headers).
// This enables cross-service trace propagation.
//
// Parameters:
//   - ctx: Context containing the tracing information
//   - carrier: Map to inject the context into (e.g., message headers)
//
// Example:
//
//	headers := make(map[string]string)
//	sagaTracing.InjectContext(ctx, headers)
//	// Send message with headers
func (stm *SagaTracingManager) InjectContext(ctx context.Context, carrier map[string]string) {
	if !stm.enabled {
		return
	}

	// OpenTelemetry uses HTTP headers for propagation
	// We need to adapt the map to http.Header format
	// For now, use a simple propagation approach
	span := stm.tracingManager.SpanFromContext(ctx)
	if span == nil {
		return
	}

	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return
	}

	// W3C Trace Context format
	carrier["traceparent"] = fmt.Sprintf("00-%s-%s-%02x",
		spanCtx.TraceID().String(),
		spanCtx.SpanID().String(),
		spanCtx.TraceFlags())

	if spanCtx.TraceState().String() != "" {
		carrier["tracestate"] = spanCtx.TraceState().String()
	}
}

// ExtractContext extracts tracing context from a carrier (e.g., message headers).
// This enables cross-service trace propagation.
//
// Parameters:
//   - ctx: Base context to attach extracted trace to
//   - carrier: Map containing the tracing context (e.g., message headers)
//
// Returns:
//   - A new context with the extracted tracing information
//
// Example:
//
//	ctx := sagaTracing.ExtractContext(ctx, message.Headers)
//	// Continue Saga execution with propagated trace
func (stm *SagaTracingManager) ExtractContext(ctx context.Context, carrier map[string]string) context.Context {
	if !stm.enabled {
		return ctx
	}

	// For now, return the context as-is
	// In a full implementation, we would parse the traceparent header
	// and create a new span context
	return ctx
}

// IsEnabled returns whether tracing is enabled.
func (stm *SagaTracingManager) IsEnabled() bool {
	return stm.enabled
}

// convertToAttribute converts a value to an OpenTelemetry attribute.
func convertToAttribute(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}

// noOpSpan is a no-operation span implementation for when tracing is disabled.
type noOpSpan struct{}

func (s *noOpSpan) SetAttribute(key string, value interface{})           {}
func (s *noOpSpan) SetAttributes(attrs ...attribute.KeyValue)            {}
func (s *noOpSpan) AddEvent(name string, opts ...oteltrace.EventOption)  {}
func (s *noOpSpan) SetStatus(code codes.Code, description string)        {}
func (s *noOpSpan) End(opts ...oteltrace.SpanEndOption)                  {}
func (s *noOpSpan) RecordError(err error, opts ...oteltrace.EventOption) {}
func (s *noOpSpan) SpanContext() oteltrace.SpanContext                   { return oteltrace.SpanContext{} }

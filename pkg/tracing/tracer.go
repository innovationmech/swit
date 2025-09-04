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
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// TracingManager defines the interface for tracing management
type TracingManager interface {
	// Initialize initializes the tracer with the given configuration
	Initialize(ctx context.Context, config *TracingConfig) error

	// StartSpan creates a new span with the given operation name
	StartSpan(ctx context.Context, operationName string, opts ...SpanOption) (context.Context, Span)

	// SpanFromContext retrieves the current span from context
	SpanFromContext(ctx context.Context) Span

	// InjectHTTPHeaders injects tracing context into HTTP headers
	InjectHTTPHeaders(ctx context.Context, headers http.Header)

	// ExtractHTTPHeaders extracts tracing context from HTTP headers
	ExtractHTTPHeaders(headers http.Header) context.Context

	// Shutdown gracefully shuts down the tracing system
	Shutdown(ctx context.Context) error
}

// Span defines the interface for a tracing span
type Span interface {
	// SetAttribute sets a single attribute on the span
	SetAttribute(key string, value interface{})

	// SetAttributes sets multiple attributes on the span
	SetAttributes(attrs ...attribute.KeyValue)

	// AddEvent adds an event to the span
	AddEvent(name string, opts ...oteltrace.EventOption)

	// SetStatus sets the status of the span
	SetStatus(code codes.Code, description string)

	// End ends the span
	End(opts ...oteltrace.SpanEndOption)

	// RecordError records an error as an event on the span
	RecordError(err error, opts ...oteltrace.EventOption)

	// SpanContext returns the span context
	SpanContext() oteltrace.SpanContext
}

// SpanOption represents an option for creating spans
type SpanOption func(*spanConfig)

// spanConfig holds the configuration for creating spans
type spanConfig struct {
	spanKind   oteltrace.SpanKind
	attributes []attribute.KeyValue
}

// WithSpanKind sets the span kind
func WithSpanKind(kind oteltrace.SpanKind) SpanOption {
	return func(c *spanConfig) {
		c.spanKind = kind
	}
}

// WithAttributes sets span attributes
func WithAttributes(attrs ...attribute.KeyValue) SpanOption {
	return func(c *spanConfig) {
		c.attributes = append(c.attributes, attrs...)
	}
}

// tracingManager implements the TracingManager interface
type tracingManager struct {
	provider   *trace.TracerProvider
	tracer     oteltrace.Tracer
	propagator propagation.TextMapPropagator
	config     *TracingConfig
}

// NewTracingManager creates a new tracing manager
func NewTracingManager() TracingManager {
	return &tracingManager{}
}

// Initialize initializes the tracer with the given configuration
func (tm *tracingManager) Initialize(ctx context.Context, config *TracingConfig) error {
	if config == nil {
		return fmt.Errorf("tracing config cannot be nil")
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid tracing config: %w", err)
	}

	tm.config = config

	if !config.Enabled {
		// Initialize no-op implementations
		tm.tracer = otel.Tracer(config.ServiceName)
		tm.propagator = propagation.NewCompositeTextMapPropagator()
		return nil
	}

	// Create resource
	res, err := tm.createResource(config)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter
	exporter, err := tm.createExporter(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create span processor
	var processor trace.SpanProcessor
	if config.Exporter.Type == "console" {
		// Use simple processor for console output for better debugging
		processor = CreateSimpleSpanProcessor(exporter)
	} else {
		// Use batch processor for production exporters
		processor = CreateBatchSpanProcessor(exporter)
	}

	// Create sampler
	sampler, err := tm.createSampler(config.Sampling)
	if err != nil {
		return fmt.Errorf("failed to create sampler: %w", err)
	}

	// Create tracer provider
	tm.provider = trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithSpanProcessor(processor),
		trace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tm.provider)

	// Create tracer
	tm.tracer = tm.provider.Tracer(config.ServiceName)

	// Setup propagator
	tm.propagator = tm.createPropagator(config.Propagators)
	otel.SetTextMapPropagator(tm.propagator)

	return nil
}

// createResource creates an OpenTelemetry resource
func (tm *tracingManager) createResource(config *TracingConfig) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(config.ServiceName),
	}

	// Add configured resource attributes
	for key, value := range config.ResourceAttributes {
		attrs = append(attrs, attribute.String(key, value))
	}

	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			attrs...,
		),
	)
}

// createExporter creates the appropriate exporter based on configuration
func (tm *tracingManager) createExporter(ctx context.Context, config *TracingConfig) (trace.SpanExporter, error) {
	factory := NewExporterFactory()
	return factory.CreateSpanExporter(ctx, config.Exporter)
}

// createSampler creates the appropriate sampler based on configuration
func (tm *tracingManager) createSampler(config SamplingConfig) (trace.Sampler, error) {
	switch config.Type {
	case "always_on":
		return trace.AlwaysSample(), nil
	case "always_off":
		return trace.NeverSample(), nil
	case "traceidratio":
		return trace.TraceIDRatioBased(config.Rate), nil
	default:
		return nil, fmt.Errorf("unsupported sampling type: %s", config.Type)
	}
}

// createPropagator creates the text map propagator
func (tm *tracingManager) createPropagator(propagatorTypes []string) propagation.TextMapPropagator {
	var propagators []propagation.TextMapPropagator

	for _, propType := range propagatorTypes {
		switch propType {
		case "tracecontext":
			propagators = append(propagators, propagation.TraceContext{})
		case "baggage":
			propagators = append(propagators, propagation.Baggage{})
		case "b3":
			// b3 propagator would require additional import
			// For now, skip unsupported propagators
		}
	}

	if len(propagators) == 0 {
		// Default to trace context if none specified
		propagators = append(propagators, propagation.TraceContext{})
	}

	return propagation.NewCompositeTextMapPropagator(propagators...)
}

// StartSpan creates a new span with the given operation name
func (tm *tracingManager) StartSpan(ctx context.Context, operationName string, opts ...SpanOption) (context.Context, Span) {
	if tm.tracer == nil {
		// Return no-op span if not initialized
		return ctx, &noOpSpan{}
	}

	config := &spanConfig{}
	for _, opt := range opts {
		opt(config)
	}

	spanOpts := []oteltrace.SpanStartOption{
		oteltrace.WithSpanKind(config.spanKind),
	}

	if len(config.attributes) > 0 {
		spanOpts = append(spanOpts, oteltrace.WithAttributes(config.attributes...))
	}

	ctx, otelSpan := tm.tracer.Start(ctx, operationName, spanOpts...)

	return ctx, &spanWrapper{span: otelSpan}
}

// SpanFromContext retrieves the current span from context
func (tm *tracingManager) SpanFromContext(ctx context.Context) Span {
	otelSpan := oteltrace.SpanFromContext(ctx)
	if otelSpan == nil || !otelSpan.SpanContext().IsValid() {
		return &noOpSpan{}
	}
	return &spanWrapper{span: otelSpan}
}

// InjectHTTPHeaders injects tracing context into HTTP headers
func (tm *tracingManager) InjectHTTPHeaders(ctx context.Context, headers http.Header) {
	if tm.propagator != nil {
		tm.propagator.Inject(ctx, propagation.HeaderCarrier(headers))
	}
}

// ExtractHTTPHeaders extracts tracing context from HTTP headers
func (tm *tracingManager) ExtractHTTPHeaders(headers http.Header) context.Context {
	if tm.propagator != nil {
		return tm.propagator.Extract(context.Background(), propagation.HeaderCarrier(headers))
	}
	return context.Background()
}

// Shutdown gracefully shuts down the tracing system
func (tm *tracingManager) Shutdown(ctx context.Context) error {
	if tm.provider != nil {
		return tm.provider.Shutdown(ctx)
	}
	return nil
}

// spanWrapper wraps an OpenTelemetry span to implement our Span interface
type spanWrapper struct {
	span oteltrace.Span
}

// SetAttribute sets a single attribute on the span
func (s *spanWrapper) SetAttribute(key string, value interface{}) {
	switch v := value.(type) {
	case string:
		s.span.SetAttributes(attribute.String(key, v))
	case int:
		s.span.SetAttributes(attribute.Int(key, v))
	case int64:
		s.span.SetAttributes(attribute.Int64(key, v))
	case float64:
		s.span.SetAttributes(attribute.Float64(key, v))
	case bool:
		s.span.SetAttributes(attribute.Bool(key, v))
	default:
		s.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

// SetAttributes sets multiple attributes on the span
func (s *spanWrapper) SetAttributes(attrs ...attribute.KeyValue) {
	s.span.SetAttributes(attrs...)
}

// AddEvent adds an event to the span
func (s *spanWrapper) AddEvent(name string, opts ...oteltrace.EventOption) {
	s.span.AddEvent(name, opts...)
}

// SetStatus sets the status of the span
func (s *spanWrapper) SetStatus(code codes.Code, description string) {
	s.span.SetStatus(code, description)
}

// End ends the span
func (s *spanWrapper) End(opts ...oteltrace.SpanEndOption) {
	s.span.End(opts...)
}

// RecordError records an error as an event on the span
func (s *spanWrapper) RecordError(err error, opts ...oteltrace.EventOption) {
	s.span.RecordError(err, opts...)
}

// SpanContext returns the span context
func (s *spanWrapper) SpanContext() oteltrace.SpanContext {
	return s.span.SpanContext()
}

// noOpSpan is a no-operation span implementation
type noOpSpan struct{}

func (s *noOpSpan) SetAttribute(key string, value interface{})           {}
func (s *noOpSpan) SetAttributes(attrs ...attribute.KeyValue)            {}
func (s *noOpSpan) AddEvent(name string, opts ...oteltrace.EventOption)  {}
func (s *noOpSpan) SetStatus(code codes.Code, description string)        {}
func (s *noOpSpan) End(opts ...oteltrace.SpanEndOption)                  {}
func (s *noOpSpan) RecordError(err error, opts ...oteltrace.EventOption) {}
func (s *noOpSpan) SpanContext() oteltrace.SpanContext {
	return oteltrace.SpanContext{}
}

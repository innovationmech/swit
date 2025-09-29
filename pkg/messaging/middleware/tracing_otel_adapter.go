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

package middleware

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// OTelTracerAdapter adapts OpenTelemetry to the Tracer interface used by the
// messaging distributed tracing middleware.
type OTelTracerAdapter struct {
	tracer     oteltrace.Tracer
	propagator propagation.TextMapPropagator
}

// NewOTelTracerAdapter creates an adapter using the global OTel provider/propagator.
// If serviceName is empty, a sensible default name is used.
func NewOTelTracerAdapter(serviceName string) *OTelTracerAdapter {
	if serviceName == "" {
		serviceName = "swit-messaging"
	}
	adapter := &OTelTracerAdapter{
		tracer:     otel.Tracer(serviceName),
		propagator: otel.GetTextMapPropagator(),
	}
	if adapter.propagator == nil {
		adapter.propagator = propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	}
	return adapter
}

// StartSpan starts an OTel span and returns a wrapped span.
func (a *OTelTracerAdapter) StartSpan(ctx context.Context, spanName string, opts ...SpanStartOption) (context.Context, Span) {
	config := &SpanStartConfig{Kind: SpanKindInternal}
	for _, opt := range opts {
		opt(config)
	}

	otelOpts := make([]oteltrace.SpanStartOption, 0, 4)
	// Map span kind
	switch config.Kind {
	case SpanKindServer:
		otelOpts = append(otelOpts, oteltrace.WithSpanKind(oteltrace.SpanKindServer))
	case SpanKindClient:
		otelOpts = append(otelOpts, oteltrace.WithSpanKind(oteltrace.SpanKindClient))
	case SpanKindProducer:
		otelOpts = append(otelOpts, oteltrace.WithSpanKind(oteltrace.SpanKindProducer))
	case SpanKindConsumer:
		otelOpts = append(otelOpts, oteltrace.WithSpanKind(oteltrace.SpanKindConsumer))
	default:
		otelOpts = append(otelOpts, oteltrace.WithSpanKind(oteltrace.SpanKindInternal))
	}

	// Map attributes
	if len(config.Attributes) > 0 {
		attrs := make([]attribute.KeyValue, 0, len(config.Attributes))
		for k, v := range config.Attributes {
			switch val := v.(type) {
			case string:
				attrs = append(attrs, attribute.String(k, val))
			case int:
				attrs = append(attrs, attribute.Int(k, val))
			case int64:
				attrs = append(attrs, attribute.Int64(k, val))
			case float64:
				attrs = append(attrs, attribute.Float64(k, val))
			case bool:
				attrs = append(attrs, attribute.Bool(k, val))
			default:
				attrs = append(attrs, attribute.String(k, fmt.Sprintf("%v", val)))
			}
		}
		otelOpts = append(otelOpts, oteltrace.WithAttributes(attrs...))
	}

	ctx2, span := a.tracer.Start(ctx, spanName, otelOpts...)
	return ctx2, &otelSpanWrapper{span: span}
}

// Extract extracts propagation context from carrier into ctx.
func (a *OTelTracerAdapter) Extract(ctx context.Context, carrier map[string]string) context.Context {
	if a.propagator == nil {
		return ctx
	}
	return a.propagator.Extract(ctx, propagation.MapCarrier(carrier))
}

// Inject injects propagation context from ctx into carrier.
func (a *OTelTracerAdapter) Inject(ctx context.Context, carrier map[string]string) {
	if a.propagator == nil {
		return
	}
	a.propagator.Inject(ctx, propagation.MapCarrier(carrier))
}

// otelSpanWrapper adapts an OTel span to the Span interface.
type otelSpanWrapper struct {
	span oteltrace.Span
}

func (w *otelSpanWrapper) SetAttributes(attributes map[string]interface{}) {
	if w.span == nil {
		return
	}
	if len(attributes) == 0 {
		return
	}
	attrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		switch val := v.(type) {
		case string:
			attrs = append(attrs, attribute.String(k, val))
		case int:
			attrs = append(attrs, attribute.Int(k, val))
		case int64:
			attrs = append(attrs, attribute.Int64(k, val))
		case float64:
			attrs = append(attrs, attribute.Float64(k, val))
		case bool:
			attrs = append(attrs, attribute.Bool(k, val))
		default:
			attrs = append(attrs, attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
	w.span.SetAttributes(attrs...)
}

func (w *otelSpanWrapper) SetAttribute(key string, value interface{}) {
	if w.span == nil {
		return
	}
	switch v := value.(type) {
	case string:
		w.span.SetAttributes(attribute.String(key, v))
	case int:
		w.span.SetAttributes(attribute.Int(key, v))
	case int64:
		w.span.SetAttributes(attribute.Int64(key, v))
	case float64:
		w.span.SetAttributes(attribute.Float64(key, v))
	case bool:
		w.span.SetAttributes(attribute.Bool(key, v))
	default:
		w.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

func (w *otelSpanWrapper) SetStatus(status SpanStatus, description string) {
	if w.span == nil {
		return
	}
	switch status {
	case SpanStatusOK:
		w.span.SetStatus(codes.Ok, description)
	case SpanStatusError:
		w.span.SetStatus(codes.Error, description)
	default:
		// leave as UNSET
	}
}

func (w *otelSpanWrapper) AddEvent(name string, attributes map[string]interface{}) {
	if w.span == nil {
		return
	}
	if len(attributes) == 0 {
		w.span.AddEvent(name)
		return
	}
	attrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		switch val := v.(type) {
		case string:
			attrs = append(attrs, attribute.String(k, val))
		case int:
			attrs = append(attrs, attribute.Int(k, val))
		case int64:
			attrs = append(attrs, attribute.Int64(k, val))
		case float64:
			attrs = append(attrs, attribute.Float64(k, val))
		case bool:
			attrs = append(attrs, attribute.Bool(k, val))
		default:
			attrs = append(attrs, attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
	w.span.AddEvent(name, oteltrace.WithAttributes(attrs...))
}

func (w *otelSpanWrapper) End() {
	if w.span != nil {
		w.span.End()
	}
}

func (w *otelSpanWrapper) IsRecording() bool {
	if w.span == nil {
		return false
	}
	return w.span.IsRecording()
}

func (w *otelSpanWrapper) GetTraceID() string {
	if w.span == nil {
		return ""
	}
	return w.span.SpanContext().TraceID().String()
}

func (w *otelSpanWrapper) GetSpanID() string {
	if w.span == nil {
		return ""
	}
	return w.span.SpanContext().SpanID().String()
}

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
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// SpanStatus represents the status of a tracing span.
type SpanStatus int

const (
	SpanStatusUnset SpanStatus = iota
	SpanStatusOK
	SpanStatusError
)

// String returns the string representation of the span status.
func (s SpanStatus) String() string {
	switch s {
	case SpanStatusUnset:
		return "UNSET"
	case SpanStatusOK:
		return "OK"
	case SpanStatusError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// SpanKind represents the kind of span.
type SpanKind int

const (
	SpanKindUnspecified SpanKind = iota
	SpanKindInternal
	SpanKindServer
	SpanKindClient
	SpanKindProducer
	SpanKindConsumer
)

// String returns the string representation of the span kind.
func (s SpanKind) String() string {
	switch s {
	case SpanKindUnspecified:
		return "UNSPECIFIED"
	case SpanKindInternal:
		return "INTERNAL"
	case SpanKindServer:
		return "SERVER"
	case SpanKindClient:
		return "CLIENT"
	case SpanKindProducer:
		return "PRODUCER"
	case SpanKindConsumer:
		return "CONSUMER"
	default:
		return "UNKNOWN"
	}
}

// TraceContext represents trace context information.
type TraceContext struct {
	TraceID    string
	SpanID     string
	TraceFlags byte
	TraceState string
}

// IsValid returns true if the trace context has valid trace and span IDs.
func (tc *TraceContext) IsValid() bool {
	return tc.TraceID != "" && tc.SpanID != ""
}

// Tracer defines the interface for distributed tracing operations.
type Tracer interface {
	// StartSpan starts a new span with the given name and options.
	StartSpan(ctx context.Context, spanName string, opts ...SpanStartOption) (context.Context, Span)

	// Extract extracts trace context from carrier (e.g., message headers).
	Extract(ctx context.Context, carrier map[string]string) context.Context

	// Inject injects trace context into carrier (e.g., message headers).
	Inject(ctx context.Context, carrier map[string]string)
}

// Span defines the interface for tracing spans.
type Span interface {
	// SetAttributes sets attributes on the span.
	SetAttributes(attributes map[string]interface{})

	// SetAttribute sets a single attribute on the span.
	SetAttribute(key string, value interface{})

	// SetStatus sets the status of the span.
	SetStatus(status SpanStatus, description string)

	// AddEvent adds an event to the span.
	AddEvent(name string, attributes map[string]interface{})

	// End ends the span.
	End()

	// IsRecording returns true if the span is recording.
	IsRecording() bool

	// GetTraceID returns the trace ID.
	GetTraceID() string

	// GetSpanID returns the span ID.
	GetSpanID() string
}

// SpanStartOption defines options for starting spans.
type SpanStartOption func(*SpanStartConfig)

// SpanStartConfig holds configuration for starting spans.
type SpanStartConfig struct {
	Kind       SpanKind
	Attributes map[string]interface{}
	StartTime  time.Time
}

// WithSpanKind sets the span kind.
func WithSpanKind(kind SpanKind) SpanStartOption {
	return func(config *SpanStartConfig) {
		config.Kind = kind
	}
}

// WithAttributes sets the span attributes.
func WithAttributes(attributes map[string]interface{}) SpanStartOption {
	return func(config *SpanStartConfig) {
		if config.Attributes == nil {
			config.Attributes = make(map[string]interface{})
		}
		for k, v := range attributes {
			config.Attributes[k] = v
		}
	}
}

// WithStartTime sets the span start time.
func WithStartTime(startTime time.Time) SpanStartOption {
	return func(config *SpanStartConfig) {
		config.StartTime = startTime
	}
}

// NoOpTracer provides a no-op implementation of the Tracer interface.
type NoOpTracer struct{}

// NewNoOpTracer creates a new no-op tracer.
func NewNoOpTracer() *NoOpTracer {
	return &NoOpTracer{}
}

// StartSpan starts a no-op span.
func (n *NoOpTracer) StartSpan(ctx context.Context, spanName string, opts ...SpanStartOption) (context.Context, Span) {
	return ctx, &NoOpSpan{}
}

// Extract performs no-op extraction.
func (n *NoOpTracer) Extract(ctx context.Context, carrier map[string]string) context.Context {
	return ctx
}

// Inject performs no-op injection.
func (n *NoOpTracer) Inject(ctx context.Context, carrier map[string]string) {
	// No-op
}

// NoOpSpan provides a no-op implementation of the Span interface.
type NoOpSpan struct{}

// SetAttributes performs no-op.
func (n *NoOpSpan) SetAttributes(attributes map[string]interface{}) {}

// SetAttribute performs no-op.
func (n *NoOpSpan) SetAttribute(key string, value interface{}) {}

// SetStatus performs no-op.
func (n *NoOpSpan) SetStatus(status SpanStatus, description string) {}

// AddEvent performs no-op.
func (n *NoOpSpan) AddEvent(name string, attributes map[string]interface{}) {}

// End performs no-op.
func (n *NoOpSpan) End() {}

// IsRecording returns false for no-op span.
func (n *NoOpSpan) IsRecording() bool {
	return false
}

// GetTraceID returns empty string for no-op span.
func (n *NoOpSpan) GetTraceID() string {
	return ""
}

// GetSpanID returns empty string for no-op span.
func (n *NoOpSpan) GetSpanID() string {
	return ""
}

// InMemoryTracer provides an in-memory implementation for testing and development.
type InMemoryTracer struct {
	spans []SpanData
	mutex sync.RWMutex
}

// SpanData represents captured span data.
type SpanData struct {
	TraceID     string
	SpanID      string
	ParentID    string
	Name        string
	Kind        SpanKind
	StartTime   time.Time
	EndTime     time.Time
	Status      SpanStatus
	Description string
	Attributes  map[string]interface{}
	Events      []EventData
}

// EventData represents span event data.
type EventData struct {
	Name       string
	Time       time.Time
	Attributes map[string]interface{}
}

// NewInMemoryTracer creates a new in-memory tracer.
func NewInMemoryTracer() *InMemoryTracer {
	return &InMemoryTracer{
		spans: make([]SpanData, 0),
	}
}

// StartSpan starts a new span and returns the context and span.
func (im *InMemoryTracer) StartSpan(ctx context.Context, spanName string, opts ...SpanStartOption) (context.Context, Span) {
	config := &SpanStartConfig{
		Kind:       SpanKindInternal,
		Attributes: make(map[string]interface{}),
		StartTime:  time.Now(),
	}

	for _, opt := range opts {
		opt(config)
	}

	span := &InMemorySpan{
		tracer:     im,
		name:       spanName,
		kind:       config.Kind,
		startTime:  config.StartTime,
		traceID:    generateTraceID(),
		spanID:     generateSpanID(),
		attributes: config.Attributes,
		events:     make([]EventData, 0),
	}

	// Store span context in the context
	newCtx := context.WithValue(ctx, "tracing.span", span)

	return newCtx, span
}

// Extract extracts trace context from carrier.
func (im *InMemoryTracer) Extract(ctx context.Context, carrier map[string]string) context.Context {
	// Simple extraction - look for standard headers
	traceParent, hasTraceParent := carrier["traceparent"]
	if hasTraceParent {
		traceContext := parseTraceParent(traceParent)
		if traceContext.IsValid() {
			return context.WithValue(ctx, "tracing.context", traceContext)
		}
	}
	return ctx
}

// Inject injects trace context into carrier.
func (im *InMemoryTracer) Inject(ctx context.Context, carrier map[string]string) {
	if span, ok := ctx.Value("tracing.span").(*InMemorySpan); ok {
		// Inject W3C trace context format
		carrier["traceparent"] = fmt.Sprintf("00-%s-%s-01", span.traceID, span.spanID)
	}
}

// GetSpans returns all captured spans.
func (im *InMemoryTracer) GetSpans() []SpanData {
	im.mutex.RLock()
	defer im.mutex.RUnlock()
	spansCopy := make([]SpanData, len(im.spans))
	copy(spansCopy, im.spans)
	return spansCopy
}

// InMemorySpan provides an in-memory span implementation.
type InMemorySpan struct {
	tracer      *InMemoryTracer
	name        string
	kind        SpanKind
	traceID     string
	spanID      string
	parentID    string
	startTime   time.Time
	endTime     time.Time
	status      SpanStatus
	description string
	attributes  map[string]interface{}
	events      []EventData
	ended       bool
	mutex       sync.RWMutex
}

// SetAttributes sets attributes on the span.
func (im *InMemorySpan) SetAttributes(attributes map[string]interface{}) {
	im.mutex.Lock()
	defer im.mutex.Unlock()
	for k, v := range attributes {
		im.attributes[k] = v
	}
}

// SetAttribute sets a single attribute on the span.
func (im *InMemorySpan) SetAttribute(key string, value interface{}) {
	im.mutex.Lock()
	defer im.mutex.Unlock()
	im.attributes[key] = value
}

// SetStatus sets the status of the span.
func (im *InMemorySpan) SetStatus(status SpanStatus, description string) {
	im.mutex.Lock()
	defer im.mutex.Unlock()
	im.status = status
	im.description = description
}

// AddEvent adds an event to the span.
func (im *InMemorySpan) AddEvent(name string, attributes map[string]interface{}) {
	im.mutex.Lock()
	defer im.mutex.Unlock()
	event := EventData{
		Name:       name,
		Time:       time.Now(),
		Attributes: attributes,
	}
	im.events = append(im.events, event)
}

// End ends the span.
func (im *InMemorySpan) End() {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	if im.ended {
		return
	}

	im.ended = true
	im.endTime = time.Now()

	// Store span data in tracer
	spanData := SpanData{
		TraceID:     im.traceID,
		SpanID:      im.spanID,
		ParentID:    im.parentID,
		Name:        im.name,
		Kind:        im.kind,
		StartTime:   im.startTime,
		EndTime:     im.endTime,
		Status:      im.status,
		Description: im.description,
		Attributes:  make(map[string]interface{}),
		Events:      make([]EventData, len(im.events)),
	}

	// Copy attributes
	for k, v := range im.attributes {
		spanData.Attributes[k] = v
	}

	// Copy events
	copy(spanData.Events, im.events)

	im.tracer.mutex.Lock()
	im.tracer.spans = append(im.tracer.spans, spanData)
	im.tracer.mutex.Unlock()
}

// IsRecording returns true if the span is recording.
func (im *InMemorySpan) IsRecording() bool {
	im.mutex.RLock()
	defer im.mutex.RUnlock()
	return !im.ended
}

// GetTraceID returns the trace ID.
func (im *InMemorySpan) GetTraceID() string {
	return im.traceID
}

// GetSpanID returns the span ID.
func (im *InMemorySpan) GetSpanID() string {
	return im.spanID
}

// TracingConfig holds configuration for the tracing middleware.
type TracingConfig struct {
	Tracer               Tracer
	SpanName             string
	SpanKind             SpanKind
	RecordMessageContent bool
	RecordHeaders        bool
	RecordRetryAttempts  bool
	MaxContentLength     int
	SensitiveHeaders     []string
}

// DefaultTracingConfig returns a default tracing configuration.
func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		Tracer:               NewNoOpTracer(),
		SpanName:             "",
		SpanKind:             SpanKindConsumer,
		RecordMessageContent: false, // Don't record content by default for security
		RecordHeaders:        true,
		RecordRetryAttempts:  true,
		MaxContentLength:     1024,
		SensitiveHeaders:     []string{"authorization", "x-api-key", "x-auth-token"},
	}
}

// DistributedTracingMiddleware provides OpenTelemetry-compatible distributed tracing.
type DistributedTracingMiddleware struct {
	config *TracingConfig
}

// NewDistributedTracingMiddleware creates a new distributed tracing middleware.
func NewDistributedTracingMiddleware(config *TracingConfig) *DistributedTracingMiddleware {
	if config == nil {
		config = DefaultTracingConfig()
	}
	if config.Tracer == nil {
		config.Tracer = NewNoOpTracer()
	}
	return &DistributedTracingMiddleware{
		config: config,
	}
}

// Name returns the middleware name.
func (dtm *DistributedTracingMiddleware) Name() string {
	return "distributed-tracing"
}

// Wrap wraps a handler with distributed tracing functionality.
func (dtm *DistributedTracingMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
	return messaging.MessageHandlerFunc(func(ctx context.Context, message *messaging.Message) error {
		// Extract trace context from message headers
		ctx = dtm.config.Tracer.Extract(ctx, message.Headers)

		// Generate span name
		spanName := dtm.config.SpanName
		if spanName == "" {
			spanName = fmt.Sprintf("process_message_%s", message.Topic)
		}

		// Start span
		spanCtx, span := dtm.config.Tracer.StartSpan(ctx, spanName,
			WithSpanKind(dtm.config.SpanKind),
			WithAttributes(dtm.buildSpanAttributes(message)),
		)
		defer span.End()

		// Add message processing event
		span.AddEvent("message.processing.start", map[string]interface{}{
			"message.id": message.ID,
			"timestamp":  time.Now().UnixNano(),
		})

		// Process the message
		err := next.Handle(spanCtx, message)

		// Set span status based on result
		if err != nil {
			span.SetStatus(SpanStatusError, err.Error())
			span.AddEvent("message.processing.error", map[string]interface{}{
				"error.message": err.Error(),
				"error.type":    fmt.Sprintf("%T", err),
			})

			// Add messaging error details if available
			if msgErr, ok := err.(*messaging.MessagingError); ok {
				span.SetAttribute("error.code", string(msgErr.Code))
				span.SetAttribute("error.retryable", msgErr.Retryable)
				if msgErr.Details != nil {
					if retryAfter, ok := msgErr.Details["retry_after"]; ok {
						span.SetAttribute("error.retry_after", retryAfter)
					}
				}
			}
		} else {
			span.SetStatus(SpanStatusOK, "")
			span.AddEvent("message.processing.success", map[string]interface{}{
				"timestamp": time.Now().UnixNano(),
			})
		}

		return err
	})
}

// buildSpanAttributes constructs span attributes from message information.
func (dtm *DistributedTracingMiddleware) buildSpanAttributes(message *messaging.Message) map[string]interface{} {
	attributes := map[string]interface{}{
		"messaging.system":           "swit",
		"messaging.destination":      message.Topic,
		"messaging.message.id":       message.ID,
		"messaging.operation":        "process",
		"messaging.destination.kind": "topic",
	}

	// Add correlation ID if present
	if message.CorrelationID != "" {
		attributes["messaging.correlation.id"] = message.CorrelationID
	}

	// Add partition key if present
	if len(message.Key) > 0 {
		attributes["messaging.partition.id"] = string(message.Key)
	}

	// Add retry information if enabled and present
	if dtm.config.RecordRetryAttempts {
		if message.DeliveryAttempt > 0 {
			attributes["messaging.retry.count"] = message.DeliveryAttempt
		}
	}

	// Add headers if enabled
	if dtm.config.RecordHeaders && len(message.Headers) > 0 {
		sanitizedHeaders := dtm.sanitizeHeaders(message.Headers)
		for k, v := range sanitizedHeaders {
			attributes[fmt.Sprintf("messaging.header.%s", k)] = v
		}
	}

	// Add message content if enabled
	if dtm.config.RecordMessageContent && len(message.Payload) > 0 {
		content := string(message.Payload)
		if len(content) > dtm.config.MaxContentLength {
			content = content[:dtm.config.MaxContentLength] + "...[TRUNCATED]"
		}
		attributes["messaging.message.payload"] = content
	}

	// Add message size
	attributes["messaging.message.payload.size"] = len(message.Payload)

	return attributes
}

// sanitizeHeaders removes sensitive header values.
func (dtm *DistributedTracingMiddleware) sanitizeHeaders(headers map[string]string) map[string]string {
	if len(dtm.config.SensitiveHeaders) == 0 {
		return headers
	}

	sanitized := make(map[string]string)
	for k, v := range headers {
		if dtm.isSensitiveHeader(k) {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}

// isSensitiveHeader checks if a header should be sanitized.
func (dtm *DistributedTracingMiddleware) isSensitiveHeader(headerName string) bool {
	lowercaseHeader := toLowerSimple(headerName)
	for _, sensitiveHeader := range dtm.config.SensitiveHeaders {
		if toLowerSimple(sensitiveHeader) == lowercaseHeader {
			return true
		}
	}
	return false
}

// toLowerSimple converts a string to lowercase.
func toLowerSimple(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + ('a' - 'A')
		} else {
			result[i] = b
		}
	}
	return string(result)
}

// Helper functions for generating trace and span IDs
func generateTraceID() string {
	return fmt.Sprintf("%032x", time.Now().UnixNano())
}

func generateSpanID() string {
	return fmt.Sprintf("%016x", time.Now().UnixNano())
}

// parseTraceParent parses a W3C traceparent header.
func parseTraceParent(traceParent string) *TraceContext {
	// Simple parsing - in a real implementation, this would be more robust
	if len(traceParent) < 55 {
		return &TraceContext{}
	}

	return &TraceContext{
		TraceID:    traceParent[3:35],
		SpanID:     traceParent[36:52],
		TraceFlags: 1,
	}
}

// CreateTracingMiddleware is a factory function to create tracing middleware from configuration.
func CreateTracingMiddleware(config map[string]interface{}) (messaging.Middleware, error) {
	tracingConfig := DefaultTracingConfig()

	// Configure tracer type
	if tracerType, ok := config["tracer_type"]; ok {
		if tracerTypeStr, ok := tracerType.(string); ok {
			switch tracerTypeStr {
			case "noop":
				tracingConfig.Tracer = NewNoOpTracer()
			case "in_memory":
				tracingConfig.Tracer = NewInMemoryTracer()
			default:
				tracingConfig.Tracer = NewNoOpTracer()
			}
		}
	}

	// Configure span name
	if spanName, ok := config["span_name"]; ok {
		if spanNameStr, ok := spanName.(string); ok {
			tracingConfig.SpanName = spanNameStr
		}
	}

	// Configure span kind
	if spanKind, ok := config["span_kind"]; ok {
		if spanKindStr, ok := spanKind.(string); ok {
			switch spanKindStr {
			case "consumer":
				tracingConfig.SpanKind = SpanKindConsumer
			case "producer":
				tracingConfig.SpanKind = SpanKindProducer
			case "internal":
				tracingConfig.SpanKind = SpanKindInternal
			default:
				tracingConfig.SpanKind = SpanKindConsumer
			}
		}
	}

	// Configure message content recording
	if recordContent, ok := config["record_message_content"]; ok {
		if recordContentBool, ok := recordContent.(bool); ok {
			tracingConfig.RecordMessageContent = recordContentBool
		}
	}

	return NewDistributedTracingMiddleware(tracingConfig), nil
}

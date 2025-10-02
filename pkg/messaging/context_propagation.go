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

package messaging

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

// ContextPropagator defines the interface for context propagation across transport layers.
// It provides methods to extract and inject context information between HTTP, gRPC, and messaging systems.
type ContextPropagator interface {
	// ExtractFromHTTP extracts context information from HTTP request
	ExtractFromHTTP(ctx context.Context, r *http.Request) context.Context

	// ExtractFromGin extracts context information from Gin context
	ExtractFromGin(c *gin.Context) context.Context

	// ExtractFromGRPC extracts context information from gRPC metadata
	ExtractFromGRPC(ctx context.Context) context.Context

	// ExtractFromMessage extracts context information from message headers
	ExtractFromMessage(ctx context.Context, message *Message) context.Context

	// InjectToHTTP injects context information into HTTP headers
	InjectToHTTP(ctx context.Context, header http.Header)

	// InjectToGRPC injects context information into gRPC metadata
	InjectToGRPC(ctx context.Context) (context.Context, metadata.MD)

	// InjectToMessage injects context information into message headers
	InjectToMessage(ctx context.Context, message *Message)
}

// standardContextPropagator implements ContextPropagator with standard header names.
type standardContextPropagator struct {
	traceIDHeader       string
	spanIDHeader        string
	correlationIDHeader string
	requestIDHeader     string
	userIDHeader        string
	tenantIDHeader      string
	sessionIDHeader     string
}

// NewStandardContextPropagator creates a new standard context propagator with default header names.
func NewStandardContextPropagator() ContextPropagator {
	return &standardContextPropagator{
		traceIDHeader:       "X-Trace-ID",
		spanIDHeader:        "X-Span-ID",
		correlationIDHeader: "X-Correlation-ID",
		requestIDHeader:     "X-Request-ID",
		userIDHeader:        "X-User-ID",
		tenantIDHeader:      "X-Tenant-ID",
		sessionIDHeader:     "X-Session-ID",
	}
}

// NewContextPropagatorWithHeaders creates a context propagator with custom header names.
func NewContextPropagatorWithHeaders(headers map[string]string) ContextPropagator {
	cp := &standardContextPropagator{
		traceIDHeader:       headers["trace_id"],
		spanIDHeader:        headers["span_id"],
		correlationIDHeader: headers["correlation_id"],
		requestIDHeader:     headers["request_id"],
		userIDHeader:        headers["user_id"],
		tenantIDHeader:      headers["tenant_id"],
		sessionIDHeader:     headers["session_id"],
	}

	// Set defaults for missing headers
	if cp.traceIDHeader == "" {
		cp.traceIDHeader = "X-Trace-ID"
	}
	if cp.spanIDHeader == "" {
		cp.spanIDHeader = "X-Span-ID"
	}
	if cp.correlationIDHeader == "" {
		cp.correlationIDHeader = "X-Correlation-ID"
	}
	if cp.requestIDHeader == "" {
		cp.requestIDHeader = "X-Request-ID"
	}
	if cp.userIDHeader == "" {
		cp.userIDHeader = "X-User-ID"
	}
	if cp.tenantIDHeader == "" {
		cp.tenantIDHeader = "X-Tenant-ID"
	}
	if cp.sessionIDHeader == "" {
		cp.sessionIDHeader = "X-Session-ID"
	}

	return cp
}

// ExtractFromHTTP extracts context information from HTTP request.
func (cp *standardContextPropagator) ExtractFromHTTP(ctx context.Context, r *http.Request) context.Context {
	if r == nil {
		return ctx
	}

	// Extract trace information from OpenTelemetry span context if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		ctx = context.WithValue(ctx, ContextKeyTraceID, span.SpanContext().TraceID().String())
		ctx = context.WithValue(ctx, "span_id", span.SpanContext().SpanID().String())
	}

	// Extract custom headers
	if correlationID := r.Header.Get(cp.correlationIDHeader); correlationID != "" {
		ctx = context.WithValue(ctx, ContextKeyCorrelationID, correlationID)
	}

	if requestID := r.Header.Get(cp.requestIDHeader); requestID != "" {
		ctx = context.WithValue(ctx, ContextKeyRequestID, requestID)
	}

	if userID := r.Header.Get(cp.userIDHeader); userID != "" {
		ctx = context.WithValue(ctx, ContextKeyUserID, userID)
	}

	if tenantID := r.Header.Get(cp.tenantIDHeader); tenantID != "" {
		ctx = context.WithValue(ctx, ContextKeyTenantID, tenantID)
	}

	if sessionID := r.Header.Get(cp.sessionIDHeader); sessionID != "" {
		ctx = context.WithValue(ctx, "session_id", sessionID)
	}

	return ctx
}

// ExtractFromGin extracts context information from Gin context.
func (cp *standardContextPropagator) ExtractFromGin(c *gin.Context) context.Context {
	ctx := c.Request.Context()

	// Extract from HTTP request
	ctx = cp.ExtractFromHTTP(ctx, c.Request)

	// Also check Gin-specific context values
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok && uid != "" {
			ctx = context.WithValue(ctx, ContextKeyUserID, uid)
		}
	}

	if tenantID, exists := c.Get("tenant_id"); exists {
		if tid, ok := tenantID.(string); ok && tid != "" {
			ctx = context.WithValue(ctx, ContextKeyTenantID, tid)
		}
	}

	return ctx
}

// ExtractFromGRPC extracts context information from gRPC metadata.
func (cp *standardContextPropagator) ExtractFromGRPC(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	// Extract trace information from OpenTelemetry span context if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		ctx = context.WithValue(ctx, ContextKeyTraceID, span.SpanContext().TraceID().String())
		ctx = context.WithValue(ctx, "span_id", span.SpanContext().SpanID().String())
	}

	// Extract correlation ID
	if values := md.Get(cp.correlationIDHeader); len(values) > 0 {
		ctx = context.WithValue(ctx, ContextKeyCorrelationID, values[0])
	}

	// Extract request ID
	if values := md.Get(cp.requestIDHeader); len(values) > 0 {
		ctx = context.WithValue(ctx, ContextKeyRequestID, values[0])
	}

	// Extract user ID
	if values := md.Get(cp.userIDHeader); len(values) > 0 {
		ctx = context.WithValue(ctx, ContextKeyUserID, values[0])
	}

	// Extract tenant ID
	if values := md.Get(cp.tenantIDHeader); len(values) > 0 {
		ctx = context.WithValue(ctx, ContextKeyTenantID, values[0])
	}

	// Extract session ID
	if values := md.Get(cp.sessionIDHeader); len(values) > 0 {
		ctx = context.WithValue(ctx, "session_id", values[0])
	}

	return ctx
}

// ExtractFromMessage extracts context information from message headers.
func (cp *standardContextPropagator) ExtractFromMessage(ctx context.Context, message *Message) context.Context {
	if message == nil || message.Headers == nil {
		return ctx
	}

	// Extract trace ID
	if traceID, ok := message.Headers["trace_id"]; ok {
		ctx = context.WithValue(ctx, ContextKeyTraceID, traceID)
	}

	// Extract span ID
	if spanID, ok := message.Headers["span_id"]; ok {
		ctx = context.WithValue(ctx, "span_id", spanID)
	}

	// Extract correlation ID
	if message.CorrelationID != "" {
		ctx = context.WithValue(ctx, ContextKeyCorrelationID, message.CorrelationID)
	}

	// Extract request ID
	if requestID, ok := message.Headers["request_id"]; ok {
		ctx = context.WithValue(ctx, ContextKeyRequestID, requestID)
	}

	// Extract user ID
	if userID, ok := message.Headers["user_id"]; ok {
		ctx = context.WithValue(ctx, ContextKeyUserID, userID)
	}

	// Extract tenant ID
	if tenantID, ok := message.Headers["tenant_id"]; ok {
		ctx = context.WithValue(ctx, ContextKeyTenantID, tenantID)
	}

	// Extract session ID
	if sessionID, ok := message.Headers["session_id"]; ok {
		ctx = context.WithValue(ctx, "session_id", sessionID)
	}

	return ctx
}

// InjectToHTTP injects context information into HTTP headers.
func (cp *standardContextPropagator) InjectToHTTP(ctx context.Context, header http.Header) {
	if header == nil {
		return
	}

	// Inject trace ID
	if traceID, ok := ctx.Value(ContextKeyTraceID).(string); ok && traceID != "" {
		header.Set(cp.traceIDHeader, traceID)
	}

	// Inject span ID
	if spanID, ok := ctx.Value("span_id").(string); ok && spanID != "" {
		header.Set(cp.spanIDHeader, spanID)
	}

	// Inject correlation ID
	if correlationID, ok := ctx.Value(ContextKeyCorrelationID).(string); ok && correlationID != "" {
		header.Set(cp.correlationIDHeader, correlationID)
	}

	// Inject request ID
	if requestID, ok := ctx.Value(ContextKeyRequestID).(string); ok && requestID != "" {
		header.Set(cp.requestIDHeader, requestID)
	}

	// Inject user ID
	if userID, ok := ctx.Value(ContextKeyUserID).(string); ok && userID != "" {
		header.Set(cp.userIDHeader, userID)
	}

	// Inject tenant ID
	if tenantID, ok := ctx.Value(ContextKeyTenantID).(string); ok && tenantID != "" {
		header.Set(cp.tenantIDHeader, tenantID)
	}

	// Inject session ID
	if sessionID, ok := ctx.Value("session_id").(string); ok && sessionID != "" {
		header.Set(cp.sessionIDHeader, sessionID)
	}
}

// InjectToGRPC injects context information into gRPC metadata.
func (cp *standardContextPropagator) InjectToGRPC(ctx context.Context) (context.Context, metadata.MD) {
	md := metadata.MD{}

	// Inject trace ID
	if traceID, ok := ctx.Value(ContextKeyTraceID).(string); ok && traceID != "" {
		md.Set(cp.traceIDHeader, traceID)
	}

	// Inject span ID
	if spanID, ok := ctx.Value("span_id").(string); ok && spanID != "" {
		md.Set(cp.spanIDHeader, spanID)
	}

	// Inject correlation ID
	if correlationID, ok := ctx.Value(ContextKeyCorrelationID).(string); ok && correlationID != "" {
		md.Set(cp.correlationIDHeader, correlationID)
	}

	// Inject request ID
	if requestID, ok := ctx.Value(ContextKeyRequestID).(string); ok && requestID != "" {
		md.Set(cp.requestIDHeader, requestID)
	}

	// Inject user ID
	if userID, ok := ctx.Value(ContextKeyUserID).(string); ok && userID != "" {
		md.Set(cp.userIDHeader, userID)
	}

	// Inject tenant ID
	if tenantID, ok := ctx.Value(ContextKeyTenantID).(string); ok && tenantID != "" {
		md.Set(cp.tenantIDHeader, tenantID)
	}

	// Inject session ID
	if sessionID, ok := ctx.Value("session_id").(string); ok && sessionID != "" {
		md.Set(cp.sessionIDHeader, sessionID)
	}

	// Merge with existing metadata if present
	if existingMD, ok := metadata.FromOutgoingContext(ctx); ok {
		md = metadata.Join(existingMD, md)
	}

	return metadata.NewOutgoingContext(ctx, md), md
}

// InjectToMessage injects context information into message headers.
func (cp *standardContextPropagator) InjectToMessage(ctx context.Context, message *Message) {
	if message == nil {
		return
	}

	// Initialize headers if nil
	if message.Headers == nil {
		message.Headers = make(map[string]string)
	}

	// Inject trace ID
	if traceID, ok := ctx.Value(ContextKeyTraceID).(string); ok && traceID != "" {
		message.Headers["trace_id"] = traceID
	}

	// Inject span ID
	if spanID, ok := ctx.Value("span_id").(string); ok && spanID != "" {
		message.Headers["span_id"] = spanID
	}

	// Inject correlation ID
	if correlationID, ok := ctx.Value(ContextKeyCorrelationID).(string); ok && correlationID != "" {
		message.CorrelationID = correlationID
		message.Headers["correlation_id"] = correlationID
	}

	// Inject request ID
	if requestID, ok := ctx.Value(ContextKeyRequestID).(string); ok && requestID != "" {
		message.Headers["request_id"] = requestID
	}

	// Inject user ID
	if userID, ok := ctx.Value(ContextKeyUserID).(string); ok && userID != "" {
		message.Headers["user_id"] = userID
	}

	// Inject tenant ID
	if tenantID, ok := ctx.Value(ContextKeyTenantID).(string); ok && tenantID != "" {
		message.Headers["tenant_id"] = tenantID
	}

	// Inject session ID
	if sessionID, ok := ctx.Value("session_id").(string); ok && sessionID != "" {
		message.Headers["session_id"] = sessionID
	}
}

// GlobalContextPropagator provides a global instance for context propagation.
var GlobalContextPropagator = NewStandardContextPropagator()

// Helper functions for common use cases

// PropagateHTTPToMessage propagates context from HTTP request to message.
func PropagateHTTPToMessage(ctx context.Context, r *http.Request, message *Message) context.Context {
	ctx = GlobalContextPropagator.ExtractFromHTTP(ctx, r)
	GlobalContextPropagator.InjectToMessage(ctx, message)
	return ctx
}

// PropagateGinToMessage propagates context from Gin context to message.
func PropagateGinToMessage(c *gin.Context, message *Message) context.Context {
	ctx := GlobalContextPropagator.ExtractFromGin(c)
	GlobalContextPropagator.InjectToMessage(ctx, message)
	return ctx
}

// PropagateGRPCToMessage propagates context from gRPC context to message.
func PropagateGRPCToMessage(ctx context.Context, message *Message) context.Context {
	ctx = GlobalContextPropagator.ExtractFromGRPC(ctx)
	GlobalContextPropagator.InjectToMessage(ctx, message)
	return ctx
}

// PropagateMessageToHTTP propagates context from message to HTTP headers.
func PropagateMessageToHTTP(ctx context.Context, message *Message, header http.Header) context.Context {
	ctx = GlobalContextPropagator.ExtractFromMessage(ctx, message)
	GlobalContextPropagator.InjectToHTTP(ctx, header)
	return ctx
}

// PropagateMessageToGRPC propagates context from message to gRPC metadata.
func PropagateMessageToGRPC(ctx context.Context, message *Message) (context.Context, metadata.MD) {
	ctx = GlobalContextPropagator.ExtractFromMessage(ctx, message)
	return GlobalContextPropagator.InjectToGRPC(ctx)
}

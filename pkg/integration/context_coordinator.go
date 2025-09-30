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

package integration

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
)

// UnifiedContext represents a unified context that can be used across HTTP, gRPC, and Messaging transports.
// It provides a consistent interface for context propagation regardless of the underlying transport mechanism.
type UnifiedContext interface {
	context.Context

	// GetTraceID returns the distributed tracing trace ID
	GetTraceID() string

	// GetSpanID returns the distributed tracing span ID
	GetSpanID() string

	// GetCorrelationID returns the correlation ID for request correlation
	GetCorrelationID() string

	// GetRequestID returns the request ID
	GetRequestID() string

	// GetUserID returns the user ID if available
	GetUserID() string

	// GetTenantID returns the tenant ID for multi-tenant applications
	GetTenantID() string

	// GetMetadata returns all metadata as a map
	GetMetadata() map[string]string

	// WithValue creates a new UnifiedContext with the given key-value pair
	WithValue(key, value string) UnifiedContext

	// Clone creates a copy of the UnifiedContext
	Clone() UnifiedContext
}

// unifiedContext implements UnifiedContext interface.
type unifiedContext struct {
	context.Context
	metadata map[string]string
	mu       sync.RWMutex
}

// NewUnifiedContext creates a new UnifiedContext from a base context.
func NewUnifiedContext(ctx context.Context) UnifiedContext {
	if ctx == nil {
		ctx = context.Background()
	}

	return &unifiedContext{
		Context:  ctx,
		metadata: make(map[string]string),
	}
}

// GetTraceID returns the trace ID from the context.
func (uc *unifiedContext) GetTraceID() string {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	return uc.metadata["trace_id"]
}

// GetSpanID returns the span ID from the context.
func (uc *unifiedContext) GetSpanID() string {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	return uc.metadata["span_id"]
}

// GetCorrelationID returns the correlation ID from the context.
func (uc *unifiedContext) GetCorrelationID() string {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	return uc.metadata["correlation_id"]
}

// GetRequestID returns the request ID from the context.
func (uc *unifiedContext) GetRequestID() string {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	return uc.metadata["request_id"]
}

// GetUserID returns the user ID from the context.
func (uc *unifiedContext) GetUserID() string {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	return uc.metadata["user_id"]
}

// GetTenantID returns the tenant ID from the context.
func (uc *unifiedContext) GetTenantID() string {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	return uc.metadata["tenant_id"]
}

// GetMetadata returns all metadata as a map.
func (uc *unifiedContext) GetMetadata() map[string]string {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	// Return a copy to prevent external modifications
	result := make(map[string]string, len(uc.metadata))
	for k, v := range uc.metadata {
		result[k] = v
	}
	return result
}

// WithValue creates a new UnifiedContext with the given key-value pair.
func (uc *unifiedContext) WithValue(key, value string) UnifiedContext {
	newCtx := &unifiedContext{
		Context:  uc.Context,
		metadata: make(map[string]string),
	}

	// Copy existing metadata
	uc.mu.RLock()
	for k, v := range uc.metadata {
		newCtx.metadata[k] = v
	}
	uc.mu.RUnlock()

	// Set new value
	newCtx.metadata[key] = value

	return newCtx
}

// Clone creates a copy of the UnifiedContext.
func (uc *unifiedContext) Clone() UnifiedContext {
	newCtx := &unifiedContext{
		Context:  uc.Context,
		metadata: make(map[string]string),
	}

	// Copy all metadata
	uc.mu.RLock()
	for k, v := range uc.metadata {
		newCtx.metadata[k] = v
	}
	uc.mu.RUnlock()

	return newCtx
}

// ContextCoordinator coordinates context propagation across HTTP, gRPC, and Messaging transports.
// It provides a unified interface for extracting and injecting context across different transport mechanisms.
type ContextCoordinator struct {
	propagator propagation.TextMapPropagator
	extractors map[TransportType]ContextExtractor
	injectors  map[TransportType]ContextInjector
	mu         sync.RWMutex
}

// TransportType represents the type of transport (HTTP, gRPC, or Messaging).
type TransportType string

const (
	// TransportHTTP represents HTTP transport
	TransportHTTP TransportType = "http"

	// TransportGRPC represents gRPC transport
	TransportGRPC TransportType = "grpc"

	// TransportMessaging represents messaging transport
	TransportMessaging TransportType = "messaging"
)

// ContextExtractor extracts context from a transport-specific carrier.
type ContextExtractor interface {
	// Extract extracts UnifiedContext from the carrier
	Extract(ctx context.Context, carrier interface{}) (UnifiedContext, error)
}

// ContextInjector injects context into a transport-specific carrier.
type ContextInjector interface {
	// Inject injects UnifiedContext into the carrier
	Inject(ctx UnifiedContext, carrier interface{}) error
}

// NewContextCoordinator creates a new ContextCoordinator with OpenTelemetry propagator.
func NewContextCoordinator(propagator propagation.TextMapPropagator) *ContextCoordinator {
	if propagator == nil {
		// Use default composite propagator if none provided
		propagator = propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	}

	cc := &ContextCoordinator{
		propagator: propagator,
		extractors: make(map[TransportType]ContextExtractor),
		injectors:  make(map[TransportType]ContextInjector),
	}

	// Register default extractors and injectors
	cc.RegisterExtractor(TransportHTTP, &HTTPContextExtractor{})
	cc.RegisterExtractor(TransportGRPC, &GRPCContextExtractor{})
	cc.RegisterExtractor(TransportMessaging, &MessagingContextExtractor{})

	cc.RegisterInjector(TransportHTTP, &HTTPContextInjector{})
	cc.RegisterInjector(TransportGRPC, &GRPCContextInjector{})
	cc.RegisterInjector(TransportMessaging, &MessagingContextInjector{})

	return cc
}

// RegisterExtractor registers a context extractor for a transport type.
func (cc *ContextCoordinator) RegisterExtractor(transportType TransportType, extractor ContextExtractor) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.extractors[transportType] = extractor
}

// RegisterInjector registers a context injector for a transport type.
func (cc *ContextCoordinator) RegisterInjector(transportType TransportType, injector ContextInjector) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.injectors[transportType] = injector
}

// Extract extracts UnifiedContext from a transport-specific carrier.
func (cc *ContextCoordinator) Extract(ctx context.Context, transportType TransportType, carrier interface{}) (UnifiedContext, error) {
	cc.mu.RLock()
	extractor, ok := cc.extractors[transportType]
	cc.mu.RUnlock()

	if !ok {
		// If no extractor found, return a new unified context
		return NewUnifiedContext(ctx), nil
	}

	return extractor.Extract(ctx, carrier)
}

// Inject injects UnifiedContext into a transport-specific carrier.
func (cc *ContextCoordinator) Inject(ctx UnifiedContext, transportType TransportType, carrier interface{}) error {
	cc.mu.RLock()
	injector, ok := cc.injectors[transportType]
	cc.mu.RUnlock()

	if !ok {
		// If no injector found, do nothing
		return nil
	}

	return injector.Inject(ctx, carrier)
}

// HTTPContextExtractor extracts context from HTTP requests.
type HTTPContextExtractor struct{}

// Extract extracts UnifiedContext from a Gin context.
func (e *HTTPContextExtractor) Extract(ctx context.Context, carrier interface{}) (UnifiedContext, error) {
	ginCtx, ok := carrier.(*gin.Context)
	if !ok {
		return NewUnifiedContext(ctx), nil
	}

	uCtx := NewUnifiedContext(ctx)

	// Extract standard headers
	if traceID := ginCtx.GetHeader("X-Trace-ID"); traceID != "" {
		uCtx = uCtx.WithValue("trace_id", traceID)
	}
	if spanID := ginCtx.GetHeader("X-Span-ID"); spanID != "" {
		uCtx = uCtx.WithValue("span_id", spanID)
	}
	if correlationID := ginCtx.GetHeader("X-Correlation-ID"); correlationID != "" {
		uCtx = uCtx.WithValue("correlation_id", correlationID)
	}
	if requestID := ginCtx.GetHeader("X-Request-ID"); requestID != "" {
		uCtx = uCtx.WithValue("request_id", requestID)
	}
	if userID := ginCtx.GetHeader("X-User-ID"); userID != "" {
		uCtx = uCtx.WithValue("user_id", userID)
	}
	if tenantID := ginCtx.GetHeader("X-Tenant-ID"); tenantID != "" {
		uCtx = uCtx.WithValue("tenant_id", tenantID)
	}

	// Extract OpenTelemetry trace context
	httpCarrier := &HTTPHeaderCarrier{headers: ginCtx.Request.Header}
	otelCtx := propagation.TraceContext{}.Extract(ctx, httpCarrier)

	// Update the unified context with OpenTelemetry context
	if uc, ok := uCtx.(*unifiedContext); ok {
		uc.Context = otelCtx
	}

	return uCtx, nil
}

// HTTPContextInjector injects context into HTTP requests.
type HTTPContextInjector struct{}

// Inject injects UnifiedContext into a Gin context.
func (i *HTTPContextInjector) Inject(ctx UnifiedContext, carrier interface{}) error {
	ginCtx, ok := carrier.(*gin.Context)
	if !ok {
		return nil
	}

	metadata := ctx.GetMetadata()

	// Inject standard headers
	if traceID := metadata["trace_id"]; traceID != "" {
		ginCtx.Header("X-Trace-ID", traceID)
	}
	if spanID := metadata["span_id"]; spanID != "" {
		ginCtx.Header("X-Span-ID", spanID)
	}
	if correlationID := metadata["correlation_id"]; correlationID != "" {
		ginCtx.Header("X-Correlation-ID", correlationID)
	}
	if requestID := metadata["request_id"]; requestID != "" {
		ginCtx.Header("X-Request-ID", requestID)
	}
	if userID := metadata["user_id"]; userID != "" {
		ginCtx.Header("X-User-ID", userID)
	}
	if tenantID := metadata["tenant_id"]; tenantID != "" {
		ginCtx.Header("X-Tenant-ID", tenantID)
	}

	// Inject OpenTelemetry trace context
	httpCarrier := &HTTPHeaderCarrier{headers: ginCtx.Request.Header}
	propagation.TraceContext{}.Inject(ctx, httpCarrier)

	return nil
}

// GRPCContextExtractor extracts context from gRPC metadata.
type GRPCContextExtractor struct{}

// Extract extracts UnifiedContext from gRPC incoming metadata.
func (e *GRPCContextExtractor) Extract(ctx context.Context, carrier interface{}) (UnifiedContext, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return NewUnifiedContext(ctx), nil
	}

	uCtx := NewUnifiedContext(ctx)

	// Extract metadata
	if values := md.Get("trace_id"); len(values) > 0 {
		uCtx = uCtx.WithValue("trace_id", values[0])
	}
	if values := md.Get("span_id"); len(values) > 0 {
		uCtx = uCtx.WithValue("span_id", values[0])
	}
	if values := md.Get("correlation_id"); len(values) > 0 {
		uCtx = uCtx.WithValue("correlation_id", values[0])
	}
	if values := md.Get("request_id"); len(values) > 0 {
		uCtx = uCtx.WithValue("request_id", values[0])
	}
	if values := md.Get("user_id"); len(values) > 0 {
		uCtx = uCtx.WithValue("user_id", values[0])
	}
	if values := md.Get("tenant_id"); len(values) > 0 {
		uCtx = uCtx.WithValue("tenant_id", values[0])
	}

	// Extract OpenTelemetry trace context
	grpcCarrier := &GRPCMetadataCarrier{md: md}
	otelCtx := propagation.TraceContext{}.Extract(ctx, grpcCarrier)

	// Update the unified context with OpenTelemetry context
	if uc, ok := uCtx.(*unifiedContext); ok {
		uc.Context = otelCtx
	}

	return uCtx, nil
}

// GRPCContextInjector injects context into gRPC metadata.
type GRPCContextInjector struct{}

// Inject injects UnifiedContext into gRPC outgoing metadata.
func (i *GRPCContextInjector) Inject(ctx UnifiedContext, carrier interface{}) error {
	md := metadata.New(nil)

	metadata := ctx.GetMetadata()

	// Inject metadata
	if traceID := metadata["trace_id"]; traceID != "" {
		md.Set("trace_id", traceID)
	}
	if spanID := metadata["span_id"]; spanID != "" {
		md.Set("span_id", spanID)
	}
	if correlationID := metadata["correlation_id"]; correlationID != "" {
		md.Set("correlation_id", correlationID)
	}
	if requestID := metadata["request_id"]; requestID != "" {
		md.Set("request_id", requestID)
	}
	if userID := metadata["user_id"]; userID != "" {
		md.Set("user_id", userID)
	}
	if tenantID := metadata["tenant_id"]; tenantID != "" {
		md.Set("tenant_id", tenantID)
	}

	// Inject OpenTelemetry trace context
	grpcCarrier := &GRPCMetadataCarrier{md: md}
	propagation.TraceContext{}.Inject(ctx, grpcCarrier)

	return nil
}

// MessagingContextExtractor extracts context from message headers.
type MessagingContextExtractor struct{}

// Extract extracts UnifiedContext from message headers.
func (e *MessagingContextExtractor) Extract(ctx context.Context, carrier interface{}) (UnifiedContext, error) {
	headers, ok := carrier.(map[string]string)
	if !ok {
		return NewUnifiedContext(ctx), nil
	}

	uCtx := NewUnifiedContext(ctx)

	// Extract headers
	if traceID, ok := headers["trace_id"]; ok {
		uCtx = uCtx.WithValue("trace_id", traceID)
	}
	if spanID, ok := headers["span_id"]; ok {
		uCtx = uCtx.WithValue("span_id", spanID)
	}
	if correlationID, ok := headers["correlation_id"]; ok {
		uCtx = uCtx.WithValue("correlation_id", correlationID)
	}
	if requestID, ok := headers["request_id"]; ok {
		uCtx = uCtx.WithValue("request_id", requestID)
	}
	if userID, ok := headers["user_id"]; ok {
		uCtx = uCtx.WithValue("user_id", userID)
	}
	if tenantID, ok := headers["tenant_id"]; ok {
		uCtx = uCtx.WithValue("tenant_id", tenantID)
	}

	// Extract OpenTelemetry trace context
	msgCarrier := &MessagingHeaderCarrier{headers: headers}
	otelCtx := propagation.TraceContext{}.Extract(ctx, msgCarrier)

	// Update the unified context with OpenTelemetry context
	if uc, ok := uCtx.(*unifiedContext); ok {
		uc.Context = otelCtx
	}

	return uCtx, nil
}

// MessagingContextInjector injects context into message headers.
type MessagingContextInjector struct{}

// Inject injects UnifiedContext into message headers.
func (i *MessagingContextInjector) Inject(ctx UnifiedContext, carrier interface{}) error {
	headers, ok := carrier.(map[string]string)
	if !ok {
		return nil
	}

	metadata := ctx.GetMetadata()

	// Inject headers
	if traceID := metadata["trace_id"]; traceID != "" {
		headers["trace_id"] = traceID
	}
	if spanID := metadata["span_id"]; spanID != "" {
		headers["span_id"] = spanID
	}
	if correlationID := metadata["correlation_id"]; correlationID != "" {
		headers["correlation_id"] = correlationID
	}
	if requestID := metadata["request_id"]; requestID != "" {
		headers["request_id"] = requestID
	}
	if userID := metadata["user_id"]; userID != "" {
		headers["user_id"] = userID
	}
	if tenantID := metadata["tenant_id"]; tenantID != "" {
		headers["tenant_id"] = tenantID
	}

	// Inject OpenTelemetry trace context
	msgCarrier := &MessagingHeaderCarrier{headers: headers}
	propagation.TraceContext{}.Inject(ctx, msgCarrier)

	return nil
}

// HTTPHeaderCarrier implements propagation.TextMapCarrier for HTTP headers.
type HTTPHeaderCarrier struct {
	headers interface {
		Get(key string) string
		Set(key, value string)
	}
}

// Get returns the value associated with the key.
func (c *HTTPHeaderCarrier) Get(key string) string {
	return c.headers.Get(key)
}

// Set sets the key-value pair.
func (c *HTTPHeaderCarrier) Set(key, value string) {
	c.headers.Set(key, value)
}

// Keys lists the keys stored in the carrier.
func (c *HTTPHeaderCarrier) Keys() []string {
	// HTTP headers don't have a Keys method, so we return common trace context keys
	return []string{"traceparent", "tracestate"}
}

// GRPCMetadataCarrier implements propagation.TextMapCarrier for gRPC metadata.
type GRPCMetadataCarrier struct {
	md metadata.MD
}

// Get returns the value associated with the key.
func (c *GRPCMetadataCarrier) Get(key string) string {
	values := c.md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// Set sets the key-value pair.
func (c *GRPCMetadataCarrier) Set(key, value string) {
	c.md.Set(key, value)
}

// Keys lists the keys stored in the carrier.
func (c *GRPCMetadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.md))
	for key := range c.md {
		keys = append(keys, key)
	}
	return keys
}

// MessagingHeaderCarrier implements propagation.TextMapCarrier for message headers.
type MessagingHeaderCarrier struct {
	headers map[string]string
}

// Get returns the value associated with the key.
func (c *MessagingHeaderCarrier) Get(key string) string {
	return c.headers[key]
}

// Set sets the key-value pair.
func (c *MessagingHeaderCarrier) Set(key, value string) {
	c.headers[key] = value
}

// Keys lists the keys stored in the carrier.
func (c *MessagingHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.headers))
	for key := range c.headers {
		keys = append(keys, key)
	}
	return keys
}

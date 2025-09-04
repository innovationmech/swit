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
	"path"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/innovationmech/swit/pkg/tracing"
)

// GRPCTracingConfig holds configuration for gRPC tracing interceptors
type GRPCTracingConfig struct {
	SkipMethods      []string // Methods to skip tracing
	RecordReqPayload bool     // Whether to record request payload
	RecordRespPayload bool    // Whether to record response payload
	MaxPayloadSize   int      // Maximum size of payload to record
}

// DefaultGRPCTracingConfig returns default gRPC tracing configuration
func DefaultGRPCTracingConfig() *GRPCTracingConfig {
	return &GRPCTracingConfig{
		SkipMethods:       []string{"/grpc.health.v1.Health/Check", "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo"},
		RecordReqPayload:  false,
		RecordRespPayload: false,
		MaxPayloadSize:    4096,
	}
}

// UnaryServerInterceptor returns a gRPC unary server interceptor with tracing
func UnaryServerInterceptor(tm tracing.TracingManager) grpc.UnaryServerInterceptor {
	return UnaryServerInterceptorWithConfig(tm, DefaultGRPCTracingConfig())
}

// UnaryServerInterceptorWithConfig returns a gRPC unary server interceptor with custom configuration
func UnaryServerInterceptorWithConfig(tm tracing.TracingManager, config *GRPCTracingConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip tracing for configured methods
		for _, skipMethod := range config.SkipMethods {
			if info.FullMethod == skipMethod {
				return handler(ctx, req)
			}
		}

		// Extract tracing context from gRPC metadata
		ctx = extractTracingFromMetadata(ctx, tm)

		// Extract method and service names
		service, method := splitMethodName(info.FullMethod)
		
		// Start span
		ctx, span := tm.StartSpan(
			ctx,
			info.FullMethod,
			tracing.WithSpanKind(trace.SpanKindServer),
			tracing.WithAttributes(
				semconv.RPCSystemKey.String("grpc"),
				semconv.RPCServiceKey.String(service),
				semconv.RPCMethodKey.String(method),
			),
		)
		defer span.End()

		// Add peer information if available
		if p, ok := peer.FromContext(ctx); ok {
			span.SetAttribute("net.peer.addr", p.Addr.String())
		}

		// Record request payload if configured
		if config.RecordReqPayload {
			span.SetAttribute("rpc.request.payload", formatPayload(req, config.MaxPayloadSize))
		}

		// Call the handler
		resp, err := handler(ctx, req)

		// Set span status and record error if any
		if err != nil {
			grpcStatus := status.Convert(err)
			span.SetAttribute(string(semconv.RPCGRPCStatusCodeKey), int(grpcStatus.Code()))
			span.SetStatus(codes.Error, grpcStatus.Message())
			span.RecordError(err)
			
			// Add error details
			span.AddEvent("grpc.error", trace.WithAttributes(
				attribute.String("grpc.error.message", grpcStatus.Message()),
				attribute.String("grpc.error.code", grpcStatus.Code().String()),
			))
		} else {
			span.SetAttribute(string(semconv.RPCGRPCStatusCodeKey), 0) // OK
			span.SetStatus(codes.Ok, "")
			
			// Record response payload if configured
			if config.RecordRespPayload {
				span.SetAttribute("rpc.response.payload", formatPayload(resp, config.MaxPayloadSize))
			}
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor with tracing
func StreamServerInterceptor(tm tracing.TracingManager) grpc.StreamServerInterceptor {
	return StreamServerInterceptorWithConfig(tm, DefaultGRPCTracingConfig())
}

// StreamServerInterceptorWithConfig returns a gRPC stream server interceptor with custom configuration
func StreamServerInterceptorWithConfig(tm tracing.TracingManager, config *GRPCTracingConfig) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip tracing for configured methods
		for _, skipMethod := range config.SkipMethods {
			if info.FullMethod == skipMethod {
				return handler(srv, stream)
			}
		}

		// Extract tracing context from gRPC metadata
		ctx := extractTracingFromMetadata(stream.Context(), tm)

		// Extract method and service names
		service, method := splitMethodName(info.FullMethod)
		
		// Start span
		ctx, span := tm.StartSpan(
			ctx,
			info.FullMethod,
			tracing.WithSpanKind(trace.SpanKindServer),
			tracing.WithAttributes(
				semconv.RPCSystemKey.String("grpc"),
				semconv.RPCServiceKey.String(service),
				semconv.RPCMethodKey.String(method),
			),
		)
		defer span.End()

		// Add peer information if available
		if p, ok := peer.FromContext(ctx); ok {
			span.SetAttribute("net.peer.addr", p.Addr.String())
		}

		// Wrap the stream with tracing context
		wrappedStream := &tracedServerStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		// Call the handler
		err := handler(srv, wrappedStream)

		// Set span status and record error if any
		if err != nil {
			grpcStatus := status.Convert(err)
			span.SetAttribute(string(semconv.RPCGRPCStatusCodeKey), int(grpcStatus.Code()))
			span.SetStatus(codes.Error, grpcStatus.Message())
			span.RecordError(err)
		} else {
			span.SetAttribute(string(semconv.RPCGRPCStatusCodeKey), 0) // OK
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

// UnaryClientInterceptor returns a gRPC unary client interceptor with tracing
func UnaryClientInterceptor(tm tracing.TracingManager) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Extract method and service names
		service, methodName := splitMethodName(method)
		
		// Start span
		ctx, span := tm.StartSpan(
			ctx,
			method,
			tracing.WithSpanKind(trace.SpanKindClient),
			tracing.WithAttributes(
				semconv.RPCSystemKey.String("grpc"),
				semconv.RPCServiceKey.String(service),
				semconv.RPCMethodKey.String(methodName),
			),
		)
		defer span.End()

		// Add target information
		span.SetAttribute("rpc.grpc.target", cc.Target())

		// Inject tracing context into gRPC metadata
		ctx = injectTracingIntoMetadata(ctx, tm)

		// Make the call
		err := invoker(ctx, method, req, reply, cc, opts...)

		// Set span status and record error if any
		if err != nil {
			grpcStatus := status.Convert(err)
			span.SetAttribute(string(semconv.RPCGRPCStatusCodeKey), int(grpcStatus.Code()))
			span.SetStatus(codes.Error, grpcStatus.Message())
			span.RecordError(err)
		} else {
			span.SetAttribute(string(semconv.RPCGRPCStatusCodeKey), 0) // OK
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

// StreamClientInterceptor returns a gRPC stream client interceptor with tracing
func StreamClientInterceptor(tm tracing.TracingManager) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Extract method and service names
		service, methodName := splitMethodName(method)
		
		// Start span
		ctx, span := tm.StartSpan(
			ctx,
			method,
			tracing.WithSpanKind(trace.SpanKindClient),
			tracing.WithAttributes(
				semconv.RPCSystemKey.String("grpc"),
				semconv.RPCServiceKey.String(service),
				semconv.RPCMethodKey.String(methodName),
			),
		)

		// Add target information
		span.SetAttribute("rpc.grpc.target", cc.Target())

		// Inject tracing context into gRPC metadata
		ctx = injectTracingIntoMetadata(ctx, tm)

		// Create the stream
		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			grpcStatus := status.Convert(err)
			span.SetAttribute(string(semconv.RPCGRPCStatusCodeKey), int(grpcStatus.Code()))
			span.SetStatus(codes.Error, grpcStatus.Message())
			span.RecordError(err)
			span.End()
			return nil, err
		}

		// Wrap the stream to handle span completion
		return &tracedClientStream{
			ClientStream: stream,
			span:         span,
		}, nil
	}
}

// tracedServerStream wraps grpc.ServerStream with tracing context
type tracedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the tracing context
func (s *tracedServerStream) Context() context.Context {
	return s.ctx
}

// tracedClientStream wraps grpc.ClientStream with span management
type tracedClientStream struct {
	grpc.ClientStream
	span tracing.Span
}

// CloseSend closes the send side and ends the span
func (s *tracedClientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	if err != nil {
		grpcStatus := status.Convert(err)
		s.span.SetAttribute(string(semconv.RPCGRPCStatusCodeKey), int(grpcStatus.Code()))
		s.span.SetStatus(codes.Error, grpcStatus.Message())
		s.span.RecordError(err)
	} else {
		s.span.SetAttribute(string(semconv.RPCGRPCStatusCodeKey), 0) // OK
		s.span.SetStatus(codes.Ok, "")
	}
	s.span.End()
	return err
}

// extractTracingFromMetadata extracts tracing context from gRPC metadata
func extractTracingFromMetadata(ctx context.Context, tm tracing.TracingManager) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	// Convert metadata to HTTP headers format for extraction
	headers := make(map[string][]string)
	for k, v := range md {
		headers[k] = v
	}

	// Use the tracing manager's extraction method (we'll need to adapt this)
	// For now, just return the context as-is since we need header-based extraction
	return ctx
}

// injectTracingIntoMetadata injects tracing context into gRPC metadata
func injectTracingIntoMetadata(ctx context.Context, tm tracing.TracingManager) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}

	// We need to adapt this to work with the tracing manager's injection method
	// For now, just return the context as-is
	return metadata.NewOutgoingContext(ctx, md)
}

// splitMethodName splits a gRPC method name into service and method parts
func splitMethodName(fullMethodName string) (string, string) {
	fullMethodName = strings.TrimPrefix(fullMethodName, "/")
	parts := strings.Split(fullMethodName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	// Fallback: use the last part as service, full name as method
	service := path.Base(fullMethodName)
	return service, fullMethodName
}

// formatPayload formats a payload for recording in spans
func formatPayload(payload interface{}, maxSize int) string {
	if payload == nil {
		return ""
	}
	
	// This is a simple string representation
	// In a real implementation, you might want to use JSON marshaling
	// with size limits and sanitization
	str := string(fmt.Sprintf("%v", payload))
	if len(str) > maxSize {
		return str[:maxSize] + "..."
	}
	return str
}
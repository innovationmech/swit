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
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/innovationmech/swit/pkg/logger"
)

// GRPCLoggingConfig holds configuration for gRPC logging interceptor
type GRPCLoggingConfig struct {
	SkipMethods    []string // Methods to skip logging
	LogPayload     bool     // Whether to log request/response payloads
	MaxPayloadSize int      // Maximum size of payload to log
}

// DefaultGRPCLoggingConfig returns default gRPC logging configuration
func DefaultGRPCLoggingConfig() *GRPCLoggingConfig {
	return &GRPCLoggingConfig{
		SkipMethods:    []string{"/grpc.health.v1.Health/Check", "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo"},
		LogPayload:     false,
		MaxPayloadSize: 1024,
	}
}

// GRPCLoggingInterceptor returns a gRPC unary server interceptor for logging
func GRPCLoggingInterceptor() grpc.UnaryServerInterceptor {
	return GRPCLoggingInterceptorWithConfig(DefaultGRPCLoggingConfig())
}

// GRPCLoggingInterceptorWithConfig returns a gRPC unary server interceptor with custom config
func GRPCLoggingInterceptorWithConfig(config *GRPCLoggingConfig) grpc.UnaryServerInterceptor {
	if config == nil {
		config = DefaultGRPCLoggingConfig()
	}

	skipMethods := make(map[string]bool)
	for _, method := range config.SkipMethods {
		skipMethods[method] = true
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip logging for configured methods
		if skipMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		start := time.Now()

		// Extract request ID from metadata or generate new one
		requestID := extractRequestID(ctx)

		// Add request ID to context for downstream logging
		ctx = context.WithValue(ctx, logger.ContextKeyRequestID, requestID)

		// Extract peer information
		var clientIP string
		if p, ok := peer.FromContext(ctx); ok {
			clientIP = p.Addr.String()
		}

		// Build initial log fields
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", info.FullMethod),
			zap.String("client_ip", clientIP),
		}

		// Add request payload if configured
		if config.LogPayload && req != nil {
			fields = append(fields, zap.Any("request", req))
		}

		// Call the handler
		resp, err := handler(ctx, req)

		// Calculate latency
		latency := time.Since(start)
		latencyMS := float64(latency.Nanoseconds()) / 1e6

		// Add latency to fields
		fields = append(fields, zap.Float64("latency_ms", latencyMS))

		// Add response payload if configured and no error
		if config.LogPayload && resp != nil && err == nil {
			fields = append(fields, zap.Any("response", resp))
		}

		// Log based on error status
		logger := logger.GetLogger()
		if err != nil {
			st, _ := status.FromError(err)
			fields = append(fields,
				zap.String("grpc_code", st.Code().String()),
				zap.String("grpc_message", st.Message()),
				zap.Error(err),
			)

			// Log at different levels based on error code
			switch st.Code() {
			case codes.Unknown, codes.Internal, codes.DataLoss:
				logger.Error("gRPC request failed", fields...)
			case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.PermissionDenied, codes.Unauthenticated:
				logger.Warn("gRPC request client error", fields...)
			default:
				logger.Info("gRPC request completed with error", fields...)
			}
		} else {
			fields = append(fields, zap.String("grpc_code", codes.OK.String()))
			logger.Info("gRPC request completed", fields...)
		}

		return resp, err
	}
}

// GRPCStreamLoggingInterceptor returns a gRPC stream server interceptor for logging
func GRPCStreamLoggingInterceptor() grpc.StreamServerInterceptor {
	return GRPCStreamLoggingInterceptorWithConfig(DefaultGRPCLoggingConfig())
}

// GRPCStreamLoggingInterceptorWithConfig returns a gRPC stream server interceptor with custom config
func GRPCStreamLoggingInterceptorWithConfig(config *GRPCLoggingConfig) grpc.StreamServerInterceptor {
	if config == nil {
		config = DefaultGRPCLoggingConfig()
	}

	skipMethods := make(map[string]bool)
	for _, method := range config.SkipMethods {
		skipMethods[method] = true
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip logging for configured methods
		if skipMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		start := time.Now()
		ctx := ss.Context()

		// Extract request ID from metadata or generate new one
		requestID := extractRequestID(ctx)

		// Extract peer information
		var clientIP string
		if p, ok := peer.FromContext(ctx); ok {
			clientIP = p.Addr.String()
		}

		// Build log fields
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", info.FullMethod),
			zap.String("client_ip", clientIP),
			zap.Bool("is_server_stream", info.IsServerStream),
			zap.Bool("is_client_stream", info.IsClientStream),
		}

		logger := logger.GetLogger()
		logger.Info("gRPC stream started", fields...)

		// Call the handler
		err := handler(srv, ss)

		// Calculate latency
		latency := time.Since(start)
		latencyMS := float64(latency.Nanoseconds()) / 1e6
		fields = append(fields, zap.Float64("latency_ms", latencyMS))

		// Log based on error status
		if err != nil {
			st, _ := status.FromError(err)
			fields = append(fields,
				zap.String("grpc_code", st.Code().String()),
				zap.String("grpc_message", st.Message()),
				zap.Error(err),
			)
			logger.Error("gRPC stream failed", fields...)
		} else {
			fields = append(fields, zap.String("grpc_code", codes.OK.String()))
			logger.Info("gRPC stream completed", fields...)
		}

		return err
	}
}

// GRPCRecoveryInterceptor returns a gRPC unary server interceptor for panic recovery
func GRPCRecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// Extract request ID if available
				var requestID string
				if val := ctx.Value(logger.ContextKeyRequestID); val != nil {
					requestID, _ = val.(string)
				}

				logger.GetLogger().Error("gRPC panic recovered",
					zap.String("request_id", requestID),
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.Stack("stack"),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// GRPCValidationInterceptor returns a gRPC unary server interceptor for request validation
func GRPCValidationInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Add request validation logic here if needed
		// For now, just pass through
		return handler(ctx, req)
	}
}

// extractRequestID extracts request ID from gRPC metadata
func extractRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.New().String()
	}

	requestIDs := md.Get("request-id")
	if len(requestIDs) > 0 {
		return requestIDs[0]
	}

	return uuid.New().String()
}

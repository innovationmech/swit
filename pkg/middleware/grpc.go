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
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCLoggingInterceptor returns a gRPC unary server interceptor for logging
func GRPCLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Extract request ID from metadata or generate new one
		requestID := extractRequestID(ctx)

		// Add request ID to context
		type contextKey string
		ctx = context.WithValue(ctx, contextKey("request_id"), requestID)

		logger.Logger.Info("gRPC request started",
			zap.String("method", info.FullMethod),
			zap.String("request_id", requestID),
		)

		resp, err := handler(ctx, req)

		duration := time.Since(start)

		if err != nil {
			logger.Logger.Error("gRPC request failed",
				zap.String("method", info.FullMethod),
				zap.String("request_id", requestID),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
		} else {
			logger.Logger.Info("gRPC request completed",
				zap.String("method", info.FullMethod),
				zap.String("request_id", requestID),
				zap.Duration("duration", duration),
			)
		}

		return resp, err
	}
}

// GRPCRecoveryInterceptor returns a gRPC unary server interceptor for panic recovery
func GRPCRecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Logger.Error("gRPC panic recovered",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
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

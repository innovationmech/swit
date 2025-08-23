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
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// SentryHTTPMiddleware creates Gin middleware for Sentry error capture and performance monitoring
type SentryHTTPMiddleware struct {
	ignorePaths       []string
	ignoreStatusCodes []int
}

// SentryHTTPMiddlewareOptions configures the Sentry HTTP middleware
type SentryHTTPMiddlewareOptions struct {
	IgnorePaths       []string
	IgnoreStatusCodes []int
}

// NewSentryHTTPMiddleware creates a new Sentry HTTP middleware with the given options
func NewSentryHTTPMiddleware(options SentryHTTPMiddlewareOptions) *SentryHTTPMiddleware {
	return &SentryHTTPMiddleware{
		ignorePaths:       options.IgnorePaths,
		ignoreStatusCodes: options.IgnoreStatusCodes,
	}
}

// GinMiddleware returns the Gin middleware handler
func (m *SentryHTTPMiddleware) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if path should be ignored
		if m.shouldIgnorePath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Create a new Sentry hub for this request
		hub := sentry.GetHubFromContext(c.Request.Context())
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}

		// Configure scope with request information
		hub.ConfigureScope(func(scope *sentry.Scope) {
			// Set request context
			scope.SetRequest(c.Request)

			// Set additional tags
			scope.SetTag("method", c.Request.Method)
			scope.SetTag("url", c.Request.URL.String())
			scope.SetTag("remote_addr", c.ClientIP())

			// Set user information if available
			if userID := c.GetString("user_id"); userID != "" {
				scope.SetUser(sentry.User{
					ID:        userID,
					IPAddress: c.ClientIP(),
				})
			}

			// Set custom context
			scope.SetContext("http", map[string]interface{}{
				"method":     c.Request.Method,
				"url":        c.Request.URL.String(),
				"user_agent": c.Request.UserAgent(),
				"referer":    c.Request.Referer(),
				"headers":    flattenHeaders(c.Request.Header),
			})
		})

		// Add request to context
		ctx := sentry.SetHubOnContext(c.Request.Context(), hub)
		c.Request = c.Request.WithContext(ctx)

		// Start transaction for performance monitoring
		transaction := sentry.StartTransaction(ctx, fmt.Sprintf("%s %s", c.Request.Method, c.FullPath()))
		defer transaction.Finish()

		// Add breadcrumb
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Type:      "http",
			Category:  "http",
			Message:   fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
			Level:     sentry.LevelInfo,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"method": c.Request.Method,
				"url":    c.Request.URL.String(),
			},
		}, nil)

		// Setup panic recovery
		defer func() {
			if err := recover(); err != nil {
				// Configure scope for panic
				hub.ConfigureScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelFatal)
					scope.SetTag("panic", "true")
				})

				// Capture panic
				hub.Recover(err)
				sentry.Flush(2 * time.Second)

				// Re-panic to let Gin's recovery middleware handle it
				panic(err)
			}
		}()

		// Process request
		c.Next()

		// Check response status and capture errors if needed
		statusCode := c.Writer.Status()
		if m.shouldCaptureStatus(statusCode) {
			// Create event for HTTP error
			event := &sentry.Event{
				Level:   m.getSentryLevelForStatus(statusCode),
				Message: fmt.Sprintf("HTTP %d: %s %s", statusCode, c.Request.Method, c.Request.URL.Path),
				Tags: map[string]string{
					"http.status_code": strconv.Itoa(statusCode),
					"http.method":      c.Request.Method,
					"http.path":        c.Request.URL.Path,
				},
				Extra: map[string]interface{}{
					"response_size": c.Writer.Size(),
				},
			}

			hub.CaptureEvent(event)
		}

		// Set transaction status based on HTTP status
		transaction.Status = m.getSpanStatusForHTTPStatus(statusCode)
		transaction.SetTag("http.status_code", strconv.Itoa(statusCode))
		transaction.SetData("http.response.status_code", statusCode)
	}
}

// shouldIgnorePath checks if the request path should be ignored
func (m *SentryHTTPMiddleware) shouldIgnorePath(path string) bool {
	for _, ignorePath := range m.ignorePaths {
		if path == ignorePath {
			return true
		}
	}
	return false
}

// shouldCaptureStatus determines if an HTTP status code should be captured
func (m *SentryHTTPMiddleware) shouldCaptureStatus(statusCode int) bool {
	// Don't capture successful responses
	if statusCode < 400 {
		return false
	}

	// Check ignore list
	for _, ignoreCode := range m.ignoreStatusCodes {
		if statusCode == ignoreCode {
			return false
		}
	}

	return true
}

// getSentryLevelForStatus returns the appropriate Sentry level for HTTP status
func (m *SentryHTTPMiddleware) getSentryLevelForStatus(statusCode int) sentry.Level {
	switch {
	case statusCode >= 500:
		return sentry.LevelError
	case statusCode >= 400:
		return sentry.LevelWarning
	default:
		return sentry.LevelInfo
	}
}

// getSpanStatusForHTTPStatus converts HTTP status to Sentry span status
func (m *SentryHTTPMiddleware) getSpanStatusForHTTPStatus(statusCode int) sentry.SpanStatus {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return sentry.SpanStatusOK
	case statusCode >= 400 && statusCode < 500:
		return sentry.SpanStatusInvalidArgument
	case statusCode >= 500:
		return sentry.SpanStatusInternalError
	default:
		return sentry.SpanStatusUnknown
	}
}

// flattenHeaders converts HTTP headers to a flat map for Sentry
func flattenHeaders(headers http.Header) map[string]interface{} {
	result := make(map[string]interface{})
	for key, values := range headers {
		if len(values) == 1 {
			result[key] = values[0]
		} else {
			result[key] = values
		}
	}
	return result
}

// SentryGRPCInterceptor provides gRPC interceptors for Sentry integration
type SentryGRPCInterceptor struct {
	ignoreStatusCodes []codes.Code
}

// SentryGRPCInterceptorOptions configures the Sentry gRPC interceptor
type SentryGRPCInterceptorOptions struct {
	IgnoreStatusCodes []codes.Code
}

// NewSentryGRPCInterceptor creates a new Sentry gRPC interceptor
func NewSentryGRPCInterceptor(options SentryGRPCInterceptorOptions) *SentryGRPCInterceptor {
	return &SentryGRPCInterceptor{
		ignoreStatusCodes: options.IgnoreStatusCodes,
	}
}

// UnaryServerInterceptor returns a unary server interceptor for Sentry
func (i *SentryGRPCInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Create a new Sentry hub for this request
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}

		// Configure scope with gRPC information
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("grpc.method", info.FullMethod)
			scope.SetTag("grpc.service", extractServiceFromMethod(info.FullMethod))

			// Extract metadata if available
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				scope.SetContext("grpc", map[string]interface{}{
					"method":   info.FullMethod,
					"metadata": flattenMetadata(md),
				})

				// Set user information if available in metadata
				if userID := extractFromMetadata(md, "user-id"); userID != "" {
					scope.SetUser(sentry.User{
						ID: userID,
					})
				}
			}
		})

		// Add hub to context
		ctx = sentry.SetHubOnContext(ctx, hub)

		// Start transaction
		transaction := sentry.StartTransaction(ctx, info.FullMethod)
		defer transaction.Finish()

		// Add breadcrumb
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Type:      "rpc",
			Category:  "grpc",
			Message:   fmt.Sprintf("gRPC call: %s", info.FullMethod),
			Level:     sentry.LevelInfo,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"method": info.FullMethod,
			},
		}, nil)

		// Setup panic recovery
		defer func() {
			if err := recover(); err != nil {
				hub.ConfigureScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelFatal)
					scope.SetTag("panic", "true")
				})

				hub.Recover(err)
				sentry.Flush(2 * time.Second)
				panic(err)
			}
		}()

		// Call the handler
		resp, err := handler(ctx, req)

		// Handle response and errors
		if err != nil {
			grpcStatus := status.Convert(err)
			if i.shouldCaptureGRPCStatus(grpcStatus.Code()) {
				hub.ConfigureScope(func(scope *sentry.Scope) {
					scope.SetLevel(i.getSentryLevelForGRPCCode(grpcStatus.Code()))
					scope.SetTag("grpc.status_code", grpcStatus.Code().String())
				})

				hub.CaptureException(err)
			}

			transaction.Status = i.getSpanStatusForGRPCCode(grpcStatus.Code())
			transaction.SetTag("grpc.status_code", grpcStatus.Code().String())
		} else {
			transaction.Status = sentry.SpanStatusOK
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a stream server interceptor for Sentry
func (i *SentryGRPCInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		// Create a new Sentry hub for this stream
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}

		// Configure scope with gRPC information
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("grpc.method", info.FullMethod)
			scope.SetTag("grpc.service", extractServiceFromMethod(info.FullMethod))
			scope.SetTag("grpc.stream", "true")

			// Extract metadata if available
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				scope.SetContext("grpc", map[string]interface{}{
					"method":   info.FullMethod,
					"metadata": flattenMetadata(md),
					"stream":   true,
				})
			}
		})

		// Add hub to context
		ctx = sentry.SetHubOnContext(ctx, hub)

		// Start transaction
		transaction := sentry.StartTransaction(ctx, info.FullMethod+" (stream)")
		defer transaction.Finish()

		// Add breadcrumb
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Type:      "rpc",
			Category:  "grpc",
			Message:   fmt.Sprintf("gRPC stream: %s", info.FullMethod),
			Level:     sentry.LevelInfo,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"method": info.FullMethod,
				"stream": true,
			},
		}, nil)

		// Setup panic recovery
		defer func() {
			if err := recover(); err != nil {
				hub.ConfigureScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelFatal)
					scope.SetTag("panic", "true")
				})

				hub.Recover(err)
				sentry.Flush(2 * time.Second)
				panic(err)
			}
		}()

		// Create wrapped stream with context
		wrappedStream := &sentryServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Call the handler
		err := handler(srv, wrappedStream)

		// Handle errors
		if err != nil {
			grpcStatus := status.Convert(err)
			if i.shouldCaptureGRPCStatus(grpcStatus.Code()) {
				hub.CaptureException(err)
			}

			transaction.Status = i.getSpanStatusForGRPCCode(grpcStatus.Code())
			transaction.SetTag("grpc.status_code", grpcStatus.Code().String())
		} else {
			transaction.Status = sentry.SpanStatusOK
		}

		return err
	}
}

// shouldCaptureGRPCStatus determines if a gRPC status code should be captured
func (i *SentryGRPCInterceptor) shouldCaptureGRPCStatus(code codes.Code) bool {
	// Don't capture OK status
	if code == codes.OK {
		return false
	}

	// Check ignore list
	for _, ignoreCode := range i.ignoreStatusCodes {
		if code == ignoreCode {
			return false
		}
	}

	return true
}

// getSentryLevelForGRPCCode returns the appropriate Sentry level for gRPC status
func (i *SentryGRPCInterceptor) getSentryLevelForGRPCCode(code codes.Code) sentry.Level {
	switch code {
	case codes.Internal, codes.DataLoss, codes.Unknown:
		return sentry.LevelError
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists,
		codes.PermissionDenied, codes.FailedPrecondition, codes.Aborted,
		codes.OutOfRange, codes.Unimplemented, codes.Unauthenticated:
		return sentry.LevelWarning
	case codes.DeadlineExceeded, codes.ResourceExhausted:
		return sentry.LevelInfo
	default:
		return sentry.LevelWarning
	}
}

// getSpanStatusForGRPCCode converts gRPC status to Sentry span status
func (i *SentryGRPCInterceptor) getSpanStatusForGRPCCode(code codes.Code) sentry.SpanStatus {
	switch code {
	case codes.OK:
		return sentry.SpanStatusOK
	case codes.Canceled:
		return sentry.SpanStatusCanceled
	case codes.InvalidArgument:
		return sentry.SpanStatusInvalidArgument
	case codes.DeadlineExceeded:
		return sentry.SpanStatusDeadlineExceeded
	case codes.NotFound:
		return sentry.SpanStatusNotFound
	case codes.AlreadyExists:
		return sentry.SpanStatusAlreadyExists
	case codes.PermissionDenied:
		return sentry.SpanStatusPermissionDenied
	case codes.ResourceExhausted:
		return sentry.SpanStatusResourceExhausted
	case codes.FailedPrecondition:
		return sentry.SpanStatusFailedPrecondition
	case codes.Aborted:
		return sentry.SpanStatusAborted
	case codes.OutOfRange:
		return sentry.SpanStatusOutOfRange
	case codes.Unimplemented:
		return sentry.SpanStatusUnimplemented
	case codes.Internal:
		return sentry.SpanStatusInternalError
	case codes.Unavailable:
		return sentry.SpanStatusUnavailable
	case codes.DataLoss:
		return sentry.SpanStatusDataLoss
	case codes.Unauthenticated:
		return sentry.SpanStatusUnauthenticated
	default:
		return sentry.SpanStatusUnknown
	}
}

// extractServiceFromMethod extracts service name from full method name
func extractServiceFromMethod(fullMethod string) string {
	parts := strings.Split(fullMethod, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "unknown"
}

// extractFromMetadata extracts a value from gRPC metadata
func extractFromMetadata(md metadata.MD, key string) string {
	if values := md.Get(key); len(values) > 0 {
		return values[0]
	}
	return ""
}

// flattenMetadata converts gRPC metadata to a flat map for Sentry
func flattenMetadata(md metadata.MD) map[string]interface{} {
	result := make(map[string]interface{})
	for key, values := range md {
		if len(values) == 1 {
			result[key] = values[0]
		} else {
			result[key] = values
		}
	}
	return result
}

// sentryServerStream wraps grpc.ServerStream to provide context propagation
type sentryServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *sentryServerStream) Context() context.Context {
	return s.ctx
}

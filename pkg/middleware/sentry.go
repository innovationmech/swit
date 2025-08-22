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
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SentryMiddleware creates a Gin middleware that integrates with Sentry for error reporting
func SentryMiddleware(repanic bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start a new Sentry transaction for request tracing
		ctx := c.Request.Context()
		transaction := sentry.StartTransaction(ctx, fmt.Sprintf("%s %s", c.Request.Method, c.FullPath()))
		transaction.Op = "http.server"
		transaction.SetData("http.method", c.Request.Method)
		transaction.SetData("http.url", c.Request.URL.String())
		transaction.SetData("http.route", c.FullPath())
		
		// Update context with transaction
		ctx = transaction.Context()
		c.Request = c.Request.WithContext(ctx)

		// Set Sentry scope with request context
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
			ctx = sentry.SetHubOnContext(ctx, hub)
			c.Request = c.Request.WithContext(ctx)
		}

		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("component", "http")
			scope.SetTag("http.method", c.Request.Method)
			scope.SetTag("http.url", c.Request.URL.String())
			scope.SetTag("http.route", c.FullPath())
			scope.SetContext("request", map[string]interface{}{
				"method":     c.Request.Method,
				"url":        c.Request.URL.String(),
				"headers":    c.Request.Header,
				"query":      c.Request.URL.RawQuery,
				"user_agent": c.Request.UserAgent(),
				"remote_ip":  c.ClientIP(),
			})
		})

		defer func() {
			if rval := recover(); rval != nil {
				// Capture the panic with Sentry
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelFatal)
					scope.SetTag("panic", "true")
					scope.SetExtra("stack_trace", string(debug.Stack()))
					
					var err error
					switch x := rval.(type) {
					case string:
						err = fmt.Errorf("panic: %s", x)
					case error:
						err = fmt.Errorf("panic: %w", x)
					default:
						err = fmt.Errorf("panic: %v", x)
					}
					
					hub.CaptureException(err)
				})

				// Finish transaction with error
				transaction.Status = sentry.SpanStatusInternalError
				transaction.Finish()

				// Flush events before panic propagation
				sentry.Flush(2 * time.Second)

				if repanic {
					panic(rval)
				} else {
					// Return 500 error
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "Internal Server Error",
					})
				}
			}
		}()

		// Process request
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Set transaction status based on response
		if c.Writer.Status() >= 400 {
			if c.Writer.Status() >= 500 {
				transaction.Status = sentry.SpanStatusInternalError
			} else {
				transaction.Status = sentry.SpanStatusInvalidArgument
			}

			// Capture error for 4xx and 5xx responses
			hub.WithScope(func(scope *sentry.Scope) {
				var level sentry.Level
				if c.Writer.Status() >= 500 {
					level = sentry.LevelError
				} else {
					level = sentry.LevelWarning
				}
				
				scope.SetLevel(level)
				scope.SetTag("http.status_code", fmt.Sprintf("%d", c.Writer.Status()))
				scope.SetExtra("response_time_ms", duration.Milliseconds())
				
				// Get error from gin context if available
				if len(c.Errors) > 0 {
					for _, ginErr := range c.Errors {
						scope.SetExtra("gin_error", ginErr.Error())
						hub.CaptureException(ginErr.Err)
					}
				} else {
					// Create a generic error for non-2xx responses
					err := fmt.Errorf("HTTP %d: %s", c.Writer.Status(), http.StatusText(c.Writer.Status()))
					hub.CaptureException(err)
				}
			})
		} else {
			transaction.Status = sentry.SpanStatusOK
		}

		// Add response metadata to transaction
		transaction.SetData("http.status_code", c.Writer.Status())
		transaction.SetData("http.response_size", c.Writer.Size())
		transaction.SetData("http.response_time_ms", duration.Milliseconds())

		// Finish the transaction
		transaction.Finish()
	}
}

// GRPCUnaryServerInterceptor creates a gRPC unary server interceptor for Sentry integration
func GRPCUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Start Sentry transaction
		transaction := sentry.StartTransaction(ctx, info.FullMethod)
		transaction.Op = "grpc.server"
		transaction.SetData("grpc.method", info.FullMethod)
		
		// Update context with transaction
		ctx = transaction.Context()

		// Configure Sentry scope
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
			ctx = sentry.SetHubOnContext(ctx, hub)
		}

		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("component", "grpc")
			scope.SetTag("grpc.method", info.FullMethod)
			scope.SetContext("grpc", map[string]interface{}{
				"method": info.FullMethod,
			})
		})

		defer func() {
			if rval := recover(); rval != nil {
				// Capture panic with Sentry
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelFatal)
					scope.SetTag("panic", "true")
					scope.SetExtra("stack_trace", string(debug.Stack()))
					
					var err error
					switch x := rval.(type) {
					case string:
						err = fmt.Errorf("grpc panic: %s", x)
					case error:
						err = fmt.Errorf("grpc panic: %w", x)
					default:
						err = fmt.Errorf("grpc panic: %v", x)
					}
					
					hub.CaptureException(err)
				})

				transaction.Status = sentry.SpanStatusInternalError
				transaction.Finish()
				
				// Flush events
				sentry.Flush(2 * time.Second)
				
				// Re-panic to maintain gRPC error handling
				panic(rval)
			}
		}()

		// Execute the handler
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		// Set transaction data
		transaction.SetData("grpc.response_time_ms", duration.Milliseconds())

		if err != nil {
			// Convert gRPC error to status
			grpcStatus, _ := status.FromError(err)
			transaction.SetData("grpc.status_code", grpcStatus.Code().String())
			
			// Set transaction status based on gRPC code
			switch grpcStatus.Code() {
			case codes.OK:
				transaction.Status = sentry.SpanStatusOK
			case codes.Canceled, codes.DeadlineExceeded, codes.Aborted:
				transaction.Status = sentry.SpanStatusCanceled
			case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.PermissionDenied, codes.Unauthenticated:
				transaction.Status = sentry.SpanStatusInvalidArgument
			case codes.ResourceExhausted:
				transaction.Status = sentry.SpanStatusResourceExhausted
			case codes.FailedPrecondition, codes.OutOfRange, codes.Unimplemented, codes.Unavailable:
				transaction.Status = sentry.SpanStatusFailedPrecondition
			default:
				transaction.Status = sentry.SpanStatusInternalError
			}

			// Capture error with Sentry if it's a server error
			if grpcStatus.Code() == codes.Internal || grpcStatus.Code() == codes.Unknown || grpcStatus.Code() == codes.DataLoss {
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelError)
					scope.SetTag("grpc.code", grpcStatus.Code().String())
					scope.SetExtra("grpc.message", grpcStatus.Message())
					scope.SetExtra("response_time_ms", duration.Milliseconds())
					hub.CaptureException(err)
				})
			}
		} else {
			transaction.Status = sentry.SpanStatusOK
			transaction.SetData("grpc.status_code", codes.OK.String())
		}

		// Finish transaction
		transaction.Finish()

		return resp, err
	}
}

// GRPCStreamServerInterceptor creates a gRPC stream server interceptor for Sentry integration
func GRPCStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Start Sentry transaction
		ctx := stream.Context()
		transaction := sentry.StartTransaction(ctx, info.FullMethod)
		transaction.Op = "grpc.server.stream"
		transaction.SetData("grpc.method", info.FullMethod)
		transaction.SetData("grpc.stream", true)
		
		// Configure Sentry scope
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
			ctx = sentry.SetHubOnContext(ctx, hub)
		}

		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("component", "grpc.stream")
			scope.SetTag("grpc.method", info.FullMethod)
			scope.SetTag("grpc.stream", "true")
			scope.SetContext("grpc", map[string]interface{}{
				"method": info.FullMethod,
				"stream": true,
			})
		})

		// Wrap the stream with the updated context
		wrappedStream := &sentryStreamWrapper{ServerStream: stream, ctx: transaction.Context()}

		defer func() {
			if rval := recover(); rval != nil {
				// Capture panic with Sentry
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelFatal)
					scope.SetTag("panic", "true")
					scope.SetExtra("stack_trace", string(debug.Stack()))
					
					var err error
					switch x := rval.(type) {
					case string:
						err = fmt.Errorf("grpc stream panic: %s", x)
					case error:
						err = fmt.Errorf("grpc stream panic: %w", x)
					default:
						err = fmt.Errorf("grpc stream panic: %v", x)
					}
					
					hub.CaptureException(err)
				})

				transaction.Status = sentry.SpanStatusInternalError
				transaction.Finish()
				
				// Flush events
				sentry.Flush(2 * time.Second)
				
				// Re-panic
				panic(rval)
			}
		}()

		// Execute the handler
		start := time.Now()
		err := handler(srv, wrappedStream)
		duration := time.Since(start)

		// Set transaction data
		transaction.SetData("grpc.response_time_ms", duration.Milliseconds())

		if err != nil {
			grpcStatus, _ := status.FromError(err)
			transaction.SetData("grpc.status_code", grpcStatus.Code().String())
			
			// Set appropriate transaction status and capture errors
			if grpcStatus.Code() == codes.Internal || grpcStatus.Code() == codes.Unknown || grpcStatus.Code() == codes.DataLoss {
				transaction.Status = sentry.SpanStatusInternalError
				
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelError)
					scope.SetTag("grpc.code", grpcStatus.Code().String())
					scope.SetExtra("grpc.message", grpcStatus.Message())
					scope.SetExtra("response_time_ms", duration.Milliseconds())
					hub.CaptureException(err)
				})
			} else {
				transaction.Status = sentry.SpanStatusInvalidArgument
			}
		} else {
			transaction.Status = sentry.SpanStatusOK
			transaction.SetData("grpc.status_code", codes.OK.String())
		}

		// Finish transaction
		transaction.Finish()

		return err
	}
}

// sentryStreamWrapper wraps grpc.ServerStream to provide updated context
type sentryStreamWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *sentryStreamWrapper) Context() context.Context {
	return s.ctx
}

// SentryErrorReportingLogger creates a zap logger that reports errors to Sentry
func SentryErrorReportingLogger(logger *zap.Logger) *zap.Logger {
	return logger.WithOptions(zap.Hooks(func(entry zapcore.Entry) error {
		// Only report errors and above to Sentry
		if entry.Level >= zapcore.ErrorLevel {
			hub := sentry.CurrentHub()
			
			var level sentry.Level
			switch entry.Level {
			case zapcore.ErrorLevel:
				level = sentry.LevelError
			case zapcore.FatalLevel, zapcore.PanicLevel:
				level = sentry.LevelFatal
			default:
				level = sentry.LevelError
			}

			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetLevel(level)
				scope.SetTag("component", "logger")
				scope.SetTag("logger", entry.LoggerName)
				scope.SetExtra("message", entry.Message)
				scope.SetExtra("caller", entry.Caller.String())
				scope.SetExtra("stack_trace", entry.Stack)
				
				// Capture as message (not exception since it's a log entry)
				hub.CaptureMessage(entry.Message)
			})
		}
		return nil
	}))
}
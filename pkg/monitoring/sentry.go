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

package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SentryConfig holds Sentry configuration options
type SentryConfig struct {
	Enabled          bool              `yaml:"enabled" json:"enabled"`
	DSN              string            `yaml:"dsn" json:"dsn"`
	Environment      string            `yaml:"environment" json:"environment"`
	Release          string            `yaml:"release" json:"release"`
	SampleRate       float64           `yaml:"sample_rate" json:"sample_rate"`
	TracesSampleRate float64           `yaml:"traces_sample_rate" json:"traces_sample_rate"`
	Debug            bool              `yaml:"debug" json:"debug"`
	AttachStacktrace bool              `yaml:"attach_stacktrace" json:"attach_stacktrace"`
	Tags             map[string]string `yaml:"tags" json:"tags"`
}

// SentryManager manages Sentry initialization and integration
type SentryManager struct {
	config     SentryConfig
	logger     *zap.Logger
	initialized bool
}

// NewSentryManager creates a new Sentry manager
func NewSentryManager(config SentryConfig, logger *zap.Logger) *SentryManager {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}

	return &SentryManager{
		config: config,
		logger: logger,
	}
}

// Initialize initializes Sentry with the provided configuration
func (sm *SentryManager) Initialize() error {
	if !sm.config.Enabled {
		sm.logger.Info("Sentry monitoring is disabled")
		return nil
	}

	if sm.config.DSN == "" {
		return fmt.Errorf("sentry DSN is required when enabled")
	}

	// Initialize Sentry
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              sm.config.DSN,
		Environment:      sm.config.Environment,
		Release:          sm.config.Release,
		SampleRate:       sm.config.SampleRate,
		TracesSampleRate: sm.config.TracesSampleRate,
		Debug:            sm.config.Debug,
		AttachStacktrace: sm.config.AttachStacktrace,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Add custom logic here if needed
			sm.logger.Debug("Sending error to Sentry", 
				zap.String("event_id", string(event.EventID)),
				zap.String("level", string(event.Level)),
				zap.String("message", event.Message))
			return event
		},
	})

	if err != nil {
		return fmt.Errorf("failed to initialize Sentry: %w", err)
	}

	// Set global tags
	for key, value := range sm.config.Tags {
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag(key, value)
		})
	}

	sm.initialized = true
	sm.logger.Info("Sentry monitoring initialized successfully", 
		zap.String("environment", sm.config.Environment),
		zap.String("release", sm.config.Release))

	return nil
}

// Shutdown gracefully shuts down Sentry
func (sm *SentryManager) Shutdown(timeout time.Duration) {
	if !sm.initialized {
		return
	}

	sm.logger.Info("Shutting down Sentry monitoring")
	sentry.Flush(timeout)
}

// CaptureError captures an error and sends it to Sentry
func (sm *SentryManager) CaptureError(err error, tags map[string]string, extra map[string]interface{}) {
	if !sm.initialized || err == nil {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		// Add tags
		for key, value := range tags {
			scope.SetTag(key, value)
		}

		// Add extra data
		for key, value := range extra {
			scope.SetExtra(key, value)
		}

		sentry.CaptureException(err)
	})
}

// CaptureMessage captures a message and sends it to Sentry
func (sm *SentryManager) CaptureMessage(message string, level sentry.Level, tags map[string]string, extra map[string]interface{}) {
	if !sm.initialized {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		// Add tags
		for key, value := range tags {
			scope.SetTag(key, value)
		}

		// Add extra data
		for key, value := range extra {
			scope.SetExtra(key, value)
		}

		scope.SetLevel(level)
		sentry.CaptureMessage(message)
	})
}

// HTTPMiddleware returns a Gin middleware that captures errors and sends them to Sentry
func (sm *SentryManager) HTTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.initialized {
			c.Next()
			return
		}

		// Create a new scope for this request
		hub := sentry.GetHubFromContext(c.Request.Context())
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
			c.Request = c.Request.WithContext(sentry.SetHubOnContext(c.Request.Context(), hub))
		}

		hub.Scope().SetTag("request.method", c.Request.Method)
		hub.Scope().SetTag("request.path", c.Request.URL.Path)
		hub.Scope().SetTag("request.remote_addr", c.ClientIP())

		defer func() {
			if err := recover(); err != nil {
				// Capture panic
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetTag("error.type", "panic")
					scope.SetExtra("request.headers", c.Request.Header)
					scope.SetExtra("request.params", c.Params)
					
					if errType, ok := err.(error); ok {
						sentry.CaptureException(errType)
					} else {
						sentry.CaptureMessage(fmt.Sprintf("Panic: %v", err))
					}
				})

				// Re-panic to let Gin handle it
				panic(err)
			}
		}()

		c.Next()

		// Capture HTTP errors
		status := c.Writer.Status()
		if status >= 400 {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetTag("http.status_code", fmt.Sprintf("%d", status))
				scope.SetTag("error.type", "http_error")
				scope.SetExtra("request.headers", c.Request.Header)
				scope.SetExtra("request.params", c.Params)

				level := sentry.LevelInfo
				if status >= 500 {
					level = sentry.LevelError
				} else if status >= 400 {
					level = sentry.LevelWarning
				}

				scope.SetLevel(level)
				sentry.CaptureMessage(fmt.Sprintf("HTTP %d: %s %s", status, c.Request.Method, c.Request.URL.Path))
			})
		}
	}
}

// GRPCUnaryInterceptor returns a gRPC unary interceptor that captures errors and sends them to Sentry
func (sm *SentryManager) GRPCUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !sm.initialized {
			return handler(ctx, req)
		}

		// Create a new scope for this request
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
			ctx = sentry.SetHubOnContext(ctx, hub)
		}

		hub.Scope().SetTag("grpc.method", info.FullMethod)
		hub.Scope().SetTag("grpc.type", "unary")

		resp, err := handler(ctx, req)

		// Capture gRPC errors
		if err != nil {
			grpcStatus, ok := status.FromError(err)
			if ok && grpcStatus.Code() != codes.OK {
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetTag("grpc.code", grpcStatus.Code().String())
					scope.SetTag("error.type", "grpc_error")
					scope.SetExtra("grpc.message", grpcStatus.Message())

					level := sentry.LevelError
					// Don't treat client errors as high priority
					if grpcStatus.Code() == codes.InvalidArgument ||
						grpcStatus.Code() == codes.NotFound ||
						grpcStatus.Code() == codes.PermissionDenied ||
						grpcStatus.Code() == codes.Unauthenticated {
						level = sentry.LevelWarning
					}

					scope.SetLevel(level)
					sentry.CaptureException(err)
				})
			}
		}

		return resp, err
	}
}

// GRPCStreamInterceptor returns a gRPC stream interceptor that captures errors and sends them to Sentry
func (sm *SentryManager) GRPCStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !sm.initialized {
			return handler(srv, ss)
		}

		// Create a new scope for this request
		ctx := ss.Context()
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
			ctx = sentry.SetHubOnContext(ctx, hub)
		}

		hub.Scope().SetTag("grpc.method", info.FullMethod)
		hub.Scope().SetTag("grpc.type", "stream")

		// Wrap the stream to capture context
		wrappedStream := &sentryServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		err := handler(srv, wrappedStream)

		// Capture gRPC errors
		if err != nil {
			grpcStatus, ok := status.FromError(err)
			if ok && grpcStatus.Code() != codes.OK {
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetTag("grpc.code", grpcStatus.Code().String())
					scope.SetTag("error.type", "grpc_stream_error")
					scope.SetExtra("grpc.message", grpcStatus.Message())

					level := sentry.LevelError
					// Don't treat client errors as high priority
					if grpcStatus.Code() == codes.InvalidArgument ||
						grpcStatus.Code() == codes.NotFound ||
						grpcStatus.Code() == codes.PermissionDenied ||
						grpcStatus.Code() == codes.Unauthenticated {
						level = sentry.LevelWarning
					}

					scope.SetLevel(level)
					sentry.CaptureException(err)
				})
			}
		}

		return err
	}
}

// sentryServerStream wraps grpc.ServerStream to provide context for Sentry
type sentryServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *sentryServerStream) Context() context.Context {
	return s.ctx
}

// IsEnabled returns whether Sentry monitoring is enabled
func (sm *SentryManager) IsEnabled() bool {
	return sm.config.Enabled && sm.initialized
}
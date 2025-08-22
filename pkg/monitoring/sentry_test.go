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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewSentryManager(t *testing.T) {
	config := SentryConfig{
		Enabled:     true,
		DSN:         "https://test@sentry.io/123",
		Environment: "test",
	}

	logger, _ := zap.NewDevelopment()
	manager := NewSentryManager(config, logger)

	assert.NotNil(t, manager)
	assert.Equal(t, config, manager.config)
	assert.Equal(t, logger, manager.logger)
	assert.False(t, manager.initialized)
}

func TestNewSentryManagerWithNilLogger(t *testing.T) {
	config := SentryConfig{
		Enabled: true,
		DSN:     "https://test@sentry.io/123",
	}

	manager := NewSentryManager(config, nil)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.logger) // Should create a default logger
}

func TestSentryManagerInitialize_Disabled(t *testing.T) {
	config := SentryConfig{
		Enabled: false,
	}

	logger, _ := zap.NewDevelopment()
	manager := NewSentryManager(config, logger)

	err := manager.Initialize()
	assert.NoError(t, err)
	assert.False(t, manager.initialized)
}

func TestSentryManagerInitialize_EmptyDSN(t *testing.T) {
	config := SentryConfig{
		Enabled: true,
		DSN:     "",
	}

	logger, _ := zap.NewDevelopment()
	manager := NewSentryManager(config, logger)

	err := manager.Initialize()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sentry DSN is required")
	assert.False(t, manager.initialized)
}

func TestSentryManagerInitialize_Success(t *testing.T) {
	config := SentryConfig{
		Enabled:          true,
		DSN:              "https://test@sentry.io/123",
		Environment:      "test",
		Release:          "1.0.0",
		SampleRate:       1.0,
		TracesSampleRate: 0.1,
		Debug:            false,
		AttachStacktrace: true,
		Tags:             map[string]string{"service": "test"},
	}

	logger, _ := zap.NewDevelopment()
	manager := NewSentryManager(config, logger)

	err := manager.Initialize()
	assert.NoError(t, err)
	assert.True(t, manager.initialized)

	// Clean up
	manager.Shutdown(time.Second)
}

func TestSentryManagerIsEnabled(t *testing.T) {
	tests := []struct {
		name        string
		config      SentryConfig
		initialize  bool
		expected    bool
	}{
		{
			name: "disabled",
			config: SentryConfig{
				Enabled: false,
			},
			initialize: false,
			expected:   false,
		},
		{
			name: "enabled but not initialized",
			config: SentryConfig{
				Enabled: true,
				DSN:     "https://test@sentry.io/123",
			},
			initialize: false,
			expected:   false,
		},
		{
			name: "enabled and initialized",
			config: SentryConfig{
				Enabled: true,
				DSN:     "https://test@sentry.io/123",
			},
			initialize: true,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := zap.NewDevelopment()
			manager := NewSentryManager(tt.config, logger)

			if tt.initialize {
				err := manager.Initialize()
				require.NoError(t, err)
				defer manager.Shutdown(time.Second)
			}

			assert.Equal(t, tt.expected, manager.IsEnabled())
		})
	}
}

func TestSentryManagerCaptureError(t *testing.T) {
	config := SentryConfig{
		Enabled: true,
		DSN:     "https://test@sentry.io/123",
	}

	logger, _ := zap.NewDevelopment()
	manager := NewSentryManager(config, logger)

	// Test with uninitialized manager
	manager.CaptureError(errors.New("test error"), nil, nil)
	// Should not panic or error

	// Test with initialized manager
	err := manager.Initialize()
	require.NoError(t, err)
	defer manager.Shutdown(time.Second)

	// This should not error but we can't easily test if it actually sends to Sentry
	manager.CaptureError(errors.New("test error"), 
		map[string]string{"tag": "value"}, 
		map[string]interface{}{"extra": "data"})

	// Test with nil error
	manager.CaptureError(nil, nil, nil)
	// Should not panic
}

func TestSentryManagerCaptureMessage(t *testing.T) {
	config := SentryConfig{
		Enabled: true,
		DSN:     "https://test@sentry.io/123",
	}

	logger, _ := zap.NewDevelopment()
	manager := NewSentryManager(config, logger)

	// Test with uninitialized manager
	manager.CaptureMessage("test message", sentry.LevelInfo, nil, nil)
	// Should not panic or error

	// Test with initialized manager
	err := manager.Initialize()
	require.NoError(t, err)
	defer manager.Shutdown(time.Second)

	// This should not error but we can't easily test if it actually sends to Sentry
	manager.CaptureMessage("test message", 
		sentry.LevelWarning,
		map[string]string{"tag": "value"}, 
		map[string]interface{}{"extra": "data"})
}

func TestSentryManagerHTTPMiddleware(t *testing.T) {
	config := SentryConfig{
		Enabled: true,
		DSN:     "https://test@sentry.io/123",
	}

	logger, _ := zap.NewDevelopment()
	manager := NewSentryManager(config, logger)

	// Test with uninitialized manager
	middleware := manager.HTTPMiddleware()
	assert.NotNil(t, middleware)

	// Set up test server
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware)

	// Test normal request
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	// Test error request
	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
	})

	req = httptest.NewRequest("GET", "/error", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestSentryManagerHTTPMiddleware_WithInitialized(t *testing.T) {
	config := SentryConfig{
		Enabled: true,
		DSN:     "https://test@sentry.io/123",
	}

	logger, _ := zap.NewDevelopment()
	manager := NewSentryManager(config, logger)

	err := manager.Initialize()
	require.NoError(t, err)
	defer manager.Shutdown(time.Second)

	// Test with initialized manager
	middleware := manager.HTTPMiddleware()
	assert.NotNil(t, middleware)

	// Set up test server
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware)

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestSentryManagerGRPCUnaryInterceptor(t *testing.T) {
	config := SentryConfig{
		Enabled: true,
		DSN:     "https://test@sentry.io/123",
	}

	logger, _ := zap.NewDevelopment()
	manager := NewSentryManager(config, logger)

	interceptor := manager.GRPCUnaryInterceptor()
	assert.NotNil(t, interceptor)

	// Test successful request
	ctx := context.Background()
	req := "test request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.NoError(t, err)
	assert.Equal(t, "success", resp)

	// Test error request
	handlerWithError := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, status.Error(codes.Internal, "internal error")
	}

	resp, err = interceptor(ctx, req, info, handlerWithError)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestSentryManagerGRPCStreamInterceptor(t *testing.T) {
	config := SentryConfig{
		Enabled: true,
		DSN:     "https://test@sentry.io/123",
	}

	logger, _ := zap.NewDevelopment()
	manager := NewSentryManager(config, logger)

	interceptor := manager.GRPCStreamInterceptor()
	assert.NotNil(t, interceptor)

	// Create a mock server stream
	ctx := context.Background()
	stream := &mockServerStream{ctx: ctx}
	
	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/StreamMethod",
	}

	// Test successful stream
	handler := func(srv interface{}, ss grpc.ServerStream) error {
		return nil
	}

	err := interceptor(nil, stream, info, handler)
	assert.NoError(t, err)

	// Test error stream
	handlerWithError := func(srv interface{}, ss grpc.ServerStream) error {
		return status.Error(codes.InvalidArgument, "invalid argument")
	}

	err = interceptor(nil, stream, info, handlerWithError)
	assert.Error(t, err)
}

func TestSentryManagerShutdown(t *testing.T) {
	config := SentryConfig{
		Enabled: true,
		DSN:     "https://test@sentry.io/123",
	}

	logger, _ := zap.NewDevelopment()
	manager := NewSentryManager(config, logger)

	// Test shutdown with uninitialized manager
	manager.Shutdown(time.Second)
	// Should not panic

	// Test shutdown with initialized manager
	err := manager.Initialize()
	require.NoError(t, err)

	manager.Shutdown(time.Second)
	// Should not panic
}

// mockServerStream is a mock implementation of grpc.ServerStream for testing
type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockServerStream) RecvMsg(msg interface{}) error {
	return nil
}
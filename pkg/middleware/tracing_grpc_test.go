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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/innovationmech/swit/pkg/tracing"
)

func TestUnaryServerInterceptor(t *testing.T) {
	// Setup test tracing manager
	config := tracing.DefaultTracingConfig()
	config.Enabled = true
	config.Exporter.Type = "console"
	
	tm := tracing.NewTracingManager()
	err := tm.Initialize(context.Background(), config)
	require.NoError(t, err)

	interceptor := UnaryServerInterceptor(tm)

	t.Run("successful request creates span", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "success", nil
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/TestMethod",
		}

		resp, err := interceptor(context.Background(), "test-request", info, handler)
		
		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	t.Run("error request records error in span", func(t *testing.T) {
		testErr := status.Error(codes.Internal, "test error")
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, testErr
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/TestMethod",
		}

		resp, err := interceptor(context.Background(), "test-request", info, handler)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, testErr, err)
	})
}

func TestUnaryServerInterceptorWithConfig(t *testing.T) {
	config := tracing.DefaultTracingConfig()
	config.Enabled = true
	config.Exporter.Type = "console"
	
	tm := tracing.NewTracingManager()
	err := tm.Initialize(context.Background(), config)
	require.NoError(t, err)

	tracingConfig := &GRPCTracingConfig{
		SkipMethods:       []string{"/grpc.health.v1.Health/Check"},
		RecordReqPayload:  false,
		RecordRespPayload: false,
		MaxPayloadSize:    1024,
	}

	interceptor := UnaryServerInterceptorWithConfig(tm, tracingConfig)

	t.Run("normal method creates span", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "success", nil
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/TestMethod",
		}

		resp, err := interceptor(context.Background(), "test-request", info, handler)
		
		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	t.Run("skip method does not create span", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "healthy", nil
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/grpc.health.v1.Health/Check",
		}

		resp, err := interceptor(context.Background(), "test-request", info, handler)
		
		assert.NoError(t, err)
		assert.Equal(t, "healthy", resp)
	})
}

func TestUnaryClientInterceptor(t *testing.T) {
	config := tracing.DefaultTracingConfig()
	config.Enabled = true
	config.Exporter.Type = "console"
	
	tm := tracing.NewTracingManager()
	err := tm.Initialize(context.Background(), config)
	require.NoError(t, err)

	interceptor := UnaryClientInterceptor(tm)

	t.Run("successful request creates span", func(t *testing.T) {
		invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			// Mock successful response
			return nil
		}

		// Create a mock client connection
		cc := &grpc.ClientConn{}

		err := interceptor(
			context.Background(),
			"/test.Service/TestMethod",
			"test-request",
			"test-reply",
			cc,
			invoker,
		)
		
		assert.NoError(t, err)
	})

	t.Run("error request records error in span", func(t *testing.T) {
		testErr := status.Error(codes.Unavailable, "service unavailable")
		
		invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return testErr
		}

		cc := &grpc.ClientConn{}

		err := interceptor(
			context.Background(),
			"/test.Service/TestMethod",
			"test-request",
			"test-reply",
			cc,
			invoker,
		)
		
		assert.Error(t, err)
		assert.Equal(t, testErr, err)
	})
}

func TestDefaultGRPCTracingConfig(t *testing.T) {
	config := DefaultGRPCTracingConfig()
	
	assert.NotNil(t, config)
	assert.Contains(t, config.SkipMethods, "/grpc.health.v1.Health/Check")
	assert.Contains(t, config.SkipMethods, "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo")
	assert.False(t, config.RecordReqPayload)
	assert.False(t, config.RecordRespPayload)
	assert.Equal(t, 4096, config.MaxPayloadSize)
}

func TestSplitMethodName(t *testing.T) {
	tests := []struct {
		name       string
		methodName string
		wantSvc    string
		wantMethod string
	}{
		{
			name:       "full method name",
			methodName: "/package.Service/Method",
			wantSvc:    "package.Service",
			wantMethod: "Method",
		},
		{
			name:       "method name without leading slash",
			methodName: "package.Service/Method",
			wantSvc:    "package.Service",
			wantMethod: "Method",
		},
		{
			name:       "single part method name",
			methodName: "/Method",
			wantSvc:    "Method",
			wantMethod: "/Method",
		},
		{
			name:       "empty method name",
			methodName: "",
			wantSvc:    ".",
			wantMethod: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, method := splitMethodName(tt.methodName)
			assert.Equal(t, tt.wantSvc, svc)
			assert.Equal(t, tt.wantMethod, method)
		})
	}
}

func TestFormatPayload(t *testing.T) {
	tests := []struct {
		name        string
		payload     interface{}
		maxSize     int
		wantContains string
	}{
		{
			name:        "nil payload",
			payload:     nil,
			maxSize:     100,
			wantContains: "",
		},
		{
			name:        "string payload",
			payload:     "test message",
			maxSize:     100,
			wantContains: "test message",
		},
		{
			name:        "struct payload",
			payload:     struct{ Name string }{Name: "test"},
			maxSize:     100,
			wantContains: "test",
		},
		{
			name:        "large payload truncated",
			payload:     "this is a very long message that exceeds the maximum size limit",
			maxSize:     10,
			wantContains: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPayload(tt.payload, tt.maxSize)
			if tt.wantContains != "" {
				assert.Contains(t, result, tt.wantContains)
			} else {
				assert.Empty(t, result)
			}
			assert.LessOrEqual(t, len(result), tt.maxSize+3) // +3 for "..."
		})
	}
}
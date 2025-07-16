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

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func init() {
	logger.Logger, _ = zap.NewDevelopment()
}

func TestGRPCLoggingInterceptor(t *testing.T) {
	interceptor := GRPCLoggingInterceptor()
	assert.NotNil(t, interceptor)

	// Mock handler
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	// Mock server info
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Test the interceptor
	resp, err := interceptor(context.Background(), "request", info, handler)
	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
}

func TestGRPCRecoveryInterceptor(t *testing.T) {
	interceptor := GRPCRecoveryInterceptor()
	assert.NotNil(t, interceptor)

	// Mock handler that panics
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test panic")
	}

	// Mock server info
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Test the interceptor
	resp, err := interceptor(context.Background(), "request", info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "internal server error")
}

func TestGRPCValidationInterceptor(t *testing.T) {
	interceptor := GRPCValidationInterceptor()
	assert.NotNil(t, interceptor)

	// Mock handler
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	// Mock server info
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	// Test the interceptor
	resp, err := interceptor(context.Background(), "request", info, handler)
	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
}

func TestExtractRequestID(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantLen int
	}{
		{
			name:    "context without metadata",
			ctx:     context.Background(),
			wantLen: 36, // UUID length
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestID := extractRequestID(tt.ctx)
			assert.Len(t, requestID, tt.wantLen)
		})
	}
}

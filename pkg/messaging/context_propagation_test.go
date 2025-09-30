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

package messaging

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestStandardContextPropagator_ExtractFromHTTP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected map[ContextKey]string
	}{
		{
			name: "extract all headers",
			headers: map[string]string{
				"X-Correlation-ID": "corr-123",
				"X-Request-ID":     "req-456",
				"X-User-ID":        "user-789",
				"X-Tenant-ID":      "tenant-001",
			},
			expected: map[ContextKey]string{
				ContextKeyCorrelationID: "corr-123",
				ContextKeyRequestID:     "req-456",
				ContextKeyUserID:        "user-789",
				ContextKeyTenantID:      "tenant-001",
			},
		},
		{
			name: "extract partial headers",
			headers: map[string]string{
				"X-Correlation-ID": "corr-123",
				"X-User-ID":        "user-789",
			},
			expected: map[ContextKey]string{
				ContextKeyCorrelationID: "corr-123",
				ContextKeyUserID:        "user-789",
			},
		},
		{
			name:     "no headers",
			headers:  map[string]string{},
			expected: map[ContextKey]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propagator := NewStandardContextPropagator()
			req := httptest.NewRequest("GET", "/test", nil)

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ctx := propagator.ExtractFromHTTP(context.Background(), req)

			for key, expectedValue := range tt.expected {
				actualValue := ctx.Value(key)
				assert.Equal(t, expectedValue, actualValue, "Expected %s to be %s", key, expectedValue)
			}
		})
	}
}

func TestStandardContextPropagator_ExtractFromGin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		headers      map[string]string
		ginValues    map[string]interface{}
		expectedKeys map[ContextKey]string
	}{
		{
			name: "extract from headers and gin context",
			headers: map[string]string{
				"X-Correlation-ID": "corr-123",
				"X-Request-ID":     "req-456",
			},
			ginValues: map[string]interface{}{
				"user_id":   "user-from-gin",
				"tenant_id": "tenant-from-gin",
			},
			expectedKeys: map[ContextKey]string{
				ContextKeyCorrelationID: "corr-123",
				ContextKeyRequestID:     "req-456",
				ContextKeyUserID:        "user-from-gin",
				ContextKeyTenantID:      "tenant-from-gin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			for k, v := range tt.headers {
				c.Request.Header.Set(k, v)
			}

			for k, v := range tt.ginValues {
				c.Set(k, v)
			}

			propagator := NewStandardContextPropagator()
			ctx := propagator.ExtractFromGin(c)

			for key, expectedValue := range tt.expectedKeys {
				actualValue := ctx.Value(key)
				assert.Equal(t, expectedValue, actualValue, "Expected %s to be %s", key, expectedValue)
			}
		})
	}
}

func TestStandardContextPropagator_ExtractFromGRPC(t *testing.T) {
	tests := []struct {
		name         string
		metadata     map[string]string
		expectedKeys map[ContextKey]string
	}{
		{
			name: "extract all metadata",
			metadata: map[string]string{
				"x-correlation-id": "corr-123",
				"x-request-id":     "req-456",
				"x-user-id":        "user-789",
				"x-tenant-id":      "tenant-001",
			},
			expectedKeys: map[ContextKey]string{
				ContextKeyCorrelationID: "corr-123",
				ContextKeyRequestID:     "req-456",
				ContextKeyUserID:        "user-789",
				ContextKeyTenantID:      "tenant-001",
			},
		},
		{
			name:         "no metadata",
			metadata:     map[string]string{},
			expectedKeys: map[ContextKey]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := metadata.New(tt.metadata)
			ctx := metadata.NewIncomingContext(context.Background(), md)

			propagator := NewStandardContextPropagator()
			ctx = propagator.ExtractFromGRPC(ctx)

			for key, expectedValue := range tt.expectedKeys {
				actualValue := ctx.Value(key)
				assert.Equal(t, expectedValue, actualValue, "Expected %s to be %s", key, expectedValue)
			}
		})
	}
}

func TestStandardContextPropagator_ExtractFromMessage(t *testing.T) {
	tests := []struct {
		name         string
		message      *Message
		expectedKeys map[ContextKey]string
	}{
		{
			name: "extract from message headers",
			message: &Message{
				ID:            "msg-123",
				CorrelationID: "corr-123",
				Headers: map[string]string{
					"trace_id":   "trace-456",
					"request_id": "req-789",
					"user_id":    "user-001",
					"tenant_id":  "tenant-002",
				},
			},
			expectedKeys: map[ContextKey]string{
				ContextKeyCorrelationID: "corr-123",
				ContextKeyTraceID:       "trace-456",
				ContextKeyRequestID:     "req-789",
				ContextKeyUserID:        "user-001",
				ContextKeyTenantID:      "tenant-002",
			},
		},
		{
			name: "empty message",
			message: &Message{
				ID: "msg-123",
			},
			expectedKeys: map[ContextKey]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propagator := NewStandardContextPropagator()
			ctx := propagator.ExtractFromMessage(context.Background(), tt.message)

			for key, expectedValue := range tt.expectedKeys {
				actualValue := ctx.Value(key)
				assert.Equal(t, expectedValue, actualValue, "Expected %s to be %s", key, expectedValue)
			}
		})
	}
}

func TestStandardContextPropagator_InjectToHTTP(t *testing.T) {
	tests := []struct {
		name            string
		contextValues   map[ContextKey]string
		expectedHeaders map[string]string
	}{
		{
			name: "inject all values",
			contextValues: map[ContextKey]string{
				ContextKeyCorrelationID: "corr-123",
				ContextKeyRequestID:     "req-456",
				ContextKeyUserID:        "user-789",
				ContextKeyTenantID:      "tenant-001",
			},
			expectedHeaders: map[string]string{
				"X-Correlation-Id": "corr-123",
				"X-Request-Id":     "req-456",
				"X-User-Id":        "user-789",
				"X-Tenant-Id":      "tenant-001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			for k, v := range tt.contextValues {
				ctx = context.WithValue(ctx, k, v)
			}

			header := http.Header{}
			propagator := NewStandardContextPropagator()
			propagator.InjectToHTTP(ctx, header)

			for expectedKey, expectedValue := range tt.expectedHeaders {
				actualValue := header.Get(expectedKey)
				assert.Equal(t, expectedValue, actualValue, "Expected header %s to be %s", expectedKey, expectedValue)
			}
		})
	}
}

func TestStandardContextPropagator_InjectToGRPC(t *testing.T) {
	tests := []struct {
		name             string
		contextValues    map[ContextKey]string
		expectedMetadata map[string]string
	}{
		{
			name: "inject all values",
			contextValues: map[ContextKey]string{
				ContextKeyCorrelationID: "corr-123",
				ContextKeyRequestID:     "req-456",
				ContextKeyUserID:        "user-789",
				ContextKeyTenantID:      "tenant-001",
			},
			expectedMetadata: map[string]string{
				"x-correlation-id": "corr-123",
				"x-request-id":     "req-456",
				"x-user-id":        "user-789",
				"x-tenant-id":      "tenant-001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			for k, v := range tt.contextValues {
				ctx = context.WithValue(ctx, k, v)
			}

			propagator := NewStandardContextPropagator()
			ctx, md := propagator.InjectToGRPC(ctx)

			for expectedKey, expectedValue := range tt.expectedMetadata {
				values := md.Get(expectedKey)
				require.NotEmpty(t, values, "Expected metadata %s to exist", expectedKey)
				assert.Equal(t, expectedValue, values[0], "Expected metadata %s to be %s", expectedKey, expectedValue)
			}
		})
	}
}

func TestStandardContextPropagator_InjectToMessage(t *testing.T) {
	tests := []struct {
		name            string
		contextValues   map[ContextKey]string
		expectedHeaders map[string]string
	}{
		{
			name: "inject all values",
			contextValues: map[ContextKey]string{
				ContextKeyCorrelationID: "corr-123",
				ContextKeyTraceID:       "trace-456",
				ContextKeyRequestID:     "req-789",
				ContextKeyUserID:        "user-001",
				ContextKeyTenantID:      "tenant-002",
			},
			expectedHeaders: map[string]string{
				"correlation_id": "corr-123",
				"trace_id":       "trace-456",
				"request_id":     "req-789",
				"user_id":        "user-001",
				"tenant_id":      "tenant-002",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			for k, v := range tt.contextValues {
				ctx = context.WithValue(ctx, k, v)
			}

			message := &Message{
				ID: "msg-test",
			}

			propagator := NewStandardContextPropagator()
			propagator.InjectToMessage(ctx, message)

			for expectedKey, expectedValue := range tt.expectedHeaders {
				actualValue, exists := message.Headers[expectedKey]
				require.True(t, exists, "Expected header %s to exist", expectedKey)
				assert.Equal(t, expectedValue, actualValue, "Expected header %s to be %s", expectedKey, expectedValue)
			}

			// Check correlation ID is set on message
			assert.Equal(t, "corr-123", message.CorrelationID)
		})
	}
}

func TestPropagateHTTPToMessage(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Correlation-ID", "corr-123")
	req.Header.Set("X-User-ID", "user-789")

	message := &Message{
		ID: "msg-test",
	}

	ctx := PropagateHTTPToMessage(context.Background(), req, message)

	assert.NotNil(t, ctx)
	assert.Equal(t, "corr-123", message.CorrelationID)
	assert.Equal(t, "corr-123", message.Headers["correlation_id"])
	assert.Equal(t, "user-789", message.Headers["user_id"])
}

func TestPropagateGinToMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Correlation-ID", "corr-123")
	c.Set("user_id", "user-from-gin")

	message := &Message{
		ID: "msg-test",
	}

	ctx := PropagateGinToMessage(c, message)

	assert.NotNil(t, ctx)
	assert.Equal(t, "corr-123", message.CorrelationID)
	assert.Equal(t, "user-from-gin", message.Headers["user_id"])
}

func TestPropagateGRPCToMessage(t *testing.T) {
	md := metadata.New(map[string]string{
		"x-correlation-id": "corr-123",
		"x-user-id":        "user-789",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	message := &Message{
		ID: "msg-test",
	}

	ctx = PropagateGRPCToMessage(ctx, message)

	assert.NotNil(t, ctx)
	assert.Equal(t, "corr-123", message.CorrelationID)
	assert.Equal(t, "user-789", message.Headers["user_id"])
}

func TestPropagateMessageToHTTP(t *testing.T) {
	message := &Message{
		ID:            "msg-test",
		CorrelationID: "corr-123",
		Headers: map[string]string{
			"user_id":   "user-789",
			"tenant_id": "tenant-001",
		},
	}

	header := http.Header{}
	ctx := PropagateMessageToHTTP(context.Background(), message, header)

	assert.NotNil(t, ctx)
	assert.Equal(t, "corr-123", header.Get("X-Correlation-ID"))
	assert.Equal(t, "user-789", header.Get("X-User-ID"))
	assert.Equal(t, "tenant-001", header.Get("X-Tenant-ID"))
}

func TestPropagateMessageToGRPC(t *testing.T) {
	message := &Message{
		ID:            "msg-test",
		CorrelationID: "corr-123",
		Headers: map[string]string{
			"user_id":   "user-789",
			"tenant_id": "tenant-001",
		},
	}

	ctx, md := PropagateMessageToGRPC(context.Background(), message)

	assert.NotNil(t, ctx)
	assert.Equal(t, "corr-123", md.Get("x-correlation-id")[0])
	assert.Equal(t, "user-789", md.Get("x-user-id")[0])
	assert.Equal(t, "tenant-001", md.Get("x-tenant-id")[0])
}

func TestContextPropagation_RoundTrip(t *testing.T) {
	// Test that context can be propagated through multiple layers
	originalCtx := context.Background()
	originalCtx = context.WithValue(originalCtx, ContextKeyCorrelationID, "corr-123")
	originalCtx = context.WithValue(originalCtx, ContextKeyUserID, "user-789")

	// HTTP -> Message
	message := &Message{ID: "msg-test"}
	GlobalContextPropagator.InjectToMessage(originalCtx, message)

	// Message -> gRPC
	grpcCtx := GlobalContextPropagator.ExtractFromMessage(context.Background(), message)
	grpcCtx, grpcMD := GlobalContextPropagator.InjectToGRPC(grpcCtx)

	// gRPC -> Message
	message2 := &Message{ID: "msg-test-2"}
	grpcCtx2 := metadata.NewIncomingContext(context.Background(), grpcMD)
	grpcCtx2 = GlobalContextPropagator.ExtractFromGRPC(grpcCtx2)
	GlobalContextPropagator.InjectToMessage(grpcCtx2, message2)

	// Verify correlation ID and user ID are preserved
	assert.Equal(t, "corr-123", message2.CorrelationID)
	assert.Equal(t, "user-789", message2.Headers["user_id"])
}

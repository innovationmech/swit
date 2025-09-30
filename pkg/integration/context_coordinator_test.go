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

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
)

func TestNewUnifiedContext(t *testing.T) {
	ctx := context.Background()
	uCtx := NewUnifiedContext(ctx)

	require.NotNil(t, uCtx)
	assert.Equal(t, "", uCtx.GetTraceID())
	assert.Equal(t, "", uCtx.GetCorrelationID())
}

func TestUnifiedContextWithValue(t *testing.T) {
	ctx := context.Background()
	uCtx := NewUnifiedContext(ctx)

	uCtx = uCtx.WithValue("trace_id", "test-trace-123")
	uCtx = uCtx.WithValue("correlation_id", "test-correlation-456")

	assert.Equal(t, "test-trace-123", uCtx.GetTraceID())
	assert.Equal(t, "test-correlation-456", uCtx.GetCorrelationID())
}

func TestUnifiedContextClone(t *testing.T) {
	ctx := context.Background()
	uCtx := NewUnifiedContext(ctx).
		WithValue("trace_id", "test-trace-123").
		WithValue("user_id", "user-456")

	clonedCtx := uCtx.Clone()

	assert.Equal(t, uCtx.GetTraceID(), clonedCtx.GetTraceID())
	assert.Equal(t, uCtx.GetUserID(), clonedCtx.GetUserID())

	// Modify clone and ensure original is unchanged
	clonedCtx = clonedCtx.WithValue("trace_id", "new-trace-789")
	assert.Equal(t, "test-trace-123", uCtx.GetTraceID())
	assert.Equal(t, "new-trace-789", clonedCtx.GetTraceID())
}

func TestUnifiedContextGetMetadata(t *testing.T) {
	ctx := context.Background()
	uCtx := NewUnifiedContext(ctx).
		WithValue("trace_id", "test-trace-123").
		WithValue("user_id", "user-456").
		WithValue("tenant_id", "tenant-789")

	metadata := uCtx.GetMetadata()

	assert.Equal(t, 3, len(metadata))
	assert.Equal(t, "test-trace-123", metadata["trace_id"])
	assert.Equal(t, "user-456", metadata["user_id"])
	assert.Equal(t, "tenant-789", metadata["tenant_id"])
}

func TestNewContextCoordinator(t *testing.T) {
	cc := NewContextCoordinator(nil)

	require.NotNil(t, cc)
	assert.NotNil(t, cc.propagator)
	assert.NotNil(t, cc.extractors)
	assert.NotNil(t, cc.injectors)
}

func TestHTTPContextExtraction(t *testing.T) {
	// Setup Gin in test mode
	gin.SetMode(gin.TestMode)

	// Create a test HTTP request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Trace-ID", "test-trace-123")
	req.Header.Set("X-Correlation-ID", "test-correlation-456")
	req.Header.Set("X-Request-ID", "test-request-789")
	req.Header.Set("X-User-ID", "user-123")
	req.Header.Set("X-Tenant-ID", "tenant-456")

	// Create Gin context
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = req

	// Extract context
	coordinator := NewContextCoordinator(nil)
	uCtx, err := coordinator.Extract(context.Background(), TransportHTTP, ginCtx)

	require.NoError(t, err)
	require.NotNil(t, uCtx)

	assert.Equal(t, "test-trace-123", uCtx.GetTraceID())
	assert.Equal(t, "test-correlation-456", uCtx.GetCorrelationID())
	assert.Equal(t, "test-request-789", uCtx.GetRequestID())
	assert.Equal(t, "user-123", uCtx.GetUserID())
	assert.Equal(t, "tenant-456", uCtx.GetTenantID())
}

func TestHTTPContextInjection(t *testing.T) {
	// Setup Gin in test mode
	gin.SetMode(gin.TestMode)

	// Create unified context with metadata
	ctx := context.Background()
	uCtx := NewUnifiedContext(ctx).
		WithValue("trace_id", "test-trace-123").
		WithValue("correlation_id", "test-correlation-456").
		WithValue("request_id", "test-request-789").
		WithValue("user_id", "user-123").
		WithValue("tenant_id", "tenant-456")

	// Create Gin context
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = req

	// Inject context
	coordinator := NewContextCoordinator(nil)
	err := coordinator.Inject(uCtx, TransportHTTP, ginCtx)

	require.NoError(t, err)

	// Verify headers were set
	assert.Equal(t, "test-trace-123", w.Header().Get("X-Trace-ID"))
	assert.Equal(t, "test-correlation-456", w.Header().Get("X-Correlation-ID"))
	assert.Equal(t, "test-request-789", w.Header().Get("X-Request-ID"))
	assert.Equal(t, "user-123", w.Header().Get("X-User-ID"))
	assert.Equal(t, "tenant-456", w.Header().Get("X-Tenant-ID"))
}

func TestGRPCContextExtraction(t *testing.T) {
	// Create gRPC metadata
	md := metadata.Pairs(
		"trace_id", "test-trace-123",
		"correlation_id", "test-correlation-456",
		"request_id", "test-request-789",
		"user_id", "user-123",
		"tenant_id", "tenant-456",
	)

	// Create context with metadata
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// Extract context
	coordinator := NewContextCoordinator(nil)
	uCtx, err := coordinator.Extract(ctx, TransportGRPC, nil)

	require.NoError(t, err)
	require.NotNil(t, uCtx)

	assert.Equal(t, "test-trace-123", uCtx.GetTraceID())
	assert.Equal(t, "test-correlation-456", uCtx.GetCorrelationID())
	assert.Equal(t, "test-request-789", uCtx.GetRequestID())
	assert.Equal(t, "user-123", uCtx.GetUserID())
	assert.Equal(t, "tenant-456", uCtx.GetTenantID())
}

func TestGRPCContextInjection(t *testing.T) {
	// Create unified context with metadata
	ctx := context.Background()
	uCtx := NewUnifiedContext(ctx).
		WithValue("trace_id", "test-trace-123").
		WithValue("correlation_id", "test-correlation-456").
		WithValue("request_id", "test-request-789").
		WithValue("user_id", "user-123").
		WithValue("tenant_id", "tenant-456")

	// Inject context (note: injection into gRPC metadata is more complex in real scenarios)
	coordinator := NewContextCoordinator(nil)
	err := coordinator.Inject(uCtx, TransportGRPC, nil)

	require.NoError(t, err)
}

func TestMessagingContextExtraction(t *testing.T) {
	// Create message headers
	headers := map[string]string{
		"trace_id":       "test-trace-123",
		"correlation_id": "test-correlation-456",
		"request_id":     "test-request-789",
		"user_id":        "user-123",
		"tenant_id":      "tenant-456",
	}

	// Extract context
	coordinator := NewContextCoordinator(nil)
	uCtx, err := coordinator.Extract(context.Background(), TransportMessaging, headers)

	require.NoError(t, err)
	require.NotNil(t, uCtx)

	assert.Equal(t, "test-trace-123", uCtx.GetTraceID())
	assert.Equal(t, "test-correlation-456", uCtx.GetCorrelationID())
	assert.Equal(t, "test-request-789", uCtx.GetRequestID())
	assert.Equal(t, "user-123", uCtx.GetUserID())
	assert.Equal(t, "tenant-456", uCtx.GetTenantID())
}

func TestMessagingContextInjection(t *testing.T) {
	// Create unified context with metadata
	ctx := context.Background()
	uCtx := NewUnifiedContext(ctx).
		WithValue("trace_id", "test-trace-123").
		WithValue("correlation_id", "test-correlation-456").
		WithValue("request_id", "test-request-789").
		WithValue("user_id", "user-123").
		WithValue("tenant_id", "tenant-456")

	// Create empty headers
	headers := make(map[string]string)

	// Inject context
	coordinator := NewContextCoordinator(nil)
	err := coordinator.Inject(uCtx, TransportMessaging, headers)

	require.NoError(t, err)

	// Verify headers were set
	assert.Equal(t, "test-trace-123", headers["trace_id"])
	assert.Equal(t, "test-correlation-456", headers["correlation_id"])
	assert.Equal(t, "test-request-789", headers["request_id"])
	assert.Equal(t, "user-123", headers["user_id"])
	assert.Equal(t, "tenant-456", headers["tenant_id"])
}

func TestContextCoordinatorWithCustomPropagator(t *testing.T) {
	// Create coordinator with custom propagator
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	coordinator := NewContextCoordinator(propagator)

	require.NotNil(t, coordinator)
	assert.NotNil(t, coordinator.propagator)
}

func TestRegisterCustomExtractor(t *testing.T) {
	coordinator := NewContextCoordinator(nil)

	// Create a custom extractor
	customExtractor := &customTestExtractor{}

	// Register it
	coordinator.RegisterExtractor("custom", customExtractor)

	// Verify it was registered (by using it)
	uCtx, err := coordinator.Extract(context.Background(), "custom", nil)
	require.NoError(t, err)
	assert.NotNil(t, uCtx)
}

func TestRegisterCustomInjector(t *testing.T) {
	coordinator := NewContextCoordinator(nil)

	// Create a custom injector
	customInjector := &customTestInjector{}

	// Register it
	coordinator.RegisterInjector("custom", customInjector)

	// Verify it was registered (by using it)
	ctx := NewUnifiedContext(context.Background())
	err := coordinator.Inject(ctx, "custom", nil)
	require.NoError(t, err)
}

// Custom test extractor for testing
type customTestExtractor struct{}

func (e *customTestExtractor) Extract(ctx context.Context, carrier interface{}) (UnifiedContext, error) {
	return NewUnifiedContext(ctx), nil
}

// Custom test injector for testing
type customTestInjector struct{}

func (i *customTestInjector) Inject(ctx UnifiedContext, carrier interface{}) error {
	return nil
}

func TestHTTPHeaderCarrier(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("test-key", "test-value")

	carrier := &HTTPHeaderCarrier{headers: req.Header}

	assert.Equal(t, "test-value", carrier.Get("test-key"))

	carrier.Set("new-key", "new-value")
	assert.Equal(t, "new-value", req.Header.Get("new-key"))

	keys := carrier.Keys()
	assert.NotNil(t, keys)
}

func TestGRPCMetadataCarrier(t *testing.T) {
	md := metadata.Pairs("test-key", "test-value")
	carrier := &GRPCMetadataCarrier{md: md}

	assert.Equal(t, "test-value", carrier.Get("test-key"))

	carrier.Set("new-key", "new-value")
	values := md.Get("new-key")
	assert.Equal(t, 1, len(values))
	assert.Equal(t, "new-value", values[0])

	keys := carrier.Keys()
	assert.Contains(t, keys, "test-key")
	assert.Contains(t, keys, "new-key")
}

func TestMessagingHeaderCarrier(t *testing.T) {
	headers := map[string]string{
		"test-key": "test-value",
	}
	carrier := &MessagingHeaderCarrier{headers: headers}

	assert.Equal(t, "test-value", carrier.Get("test-key"))

	carrier.Set("new-key", "new-value")
	assert.Equal(t, "new-value", headers["new-key"])

	keys := carrier.Keys()
	assert.Contains(t, keys, "test-key")
	assert.Contains(t, keys, "new-key")
}

func TestCrossTransportContextPropagation(t *testing.T) {
	coordinator := NewContextCoordinator(nil)

	// Step 1: Extract from HTTP
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Trace-ID", "cross-transport-trace-123")
	req.Header.Set("X-Correlation-ID", "cross-transport-correlation-456")
	req.Header.Set("X-User-ID", "user-cross-789")

	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = req

	httpCtx, err := coordinator.Extract(context.Background(), TransportHTTP, ginCtx)
	require.NoError(t, err)

	// Step 2: Inject into Messaging
	messageHeaders := make(map[string]string)
	err = coordinator.Inject(httpCtx, TransportMessaging, messageHeaders)
	require.NoError(t, err)

	// Step 3: Extract from Messaging
	messagingCtx, err := coordinator.Extract(context.Background(), TransportMessaging, messageHeaders)
	require.NoError(t, err)

	// Verify context was propagated correctly
	assert.Equal(t, "cross-transport-trace-123", messagingCtx.GetTraceID())
	assert.Equal(t, "cross-transport-correlation-456", messagingCtx.GetCorrelationID())
	assert.Equal(t, "user-cross-789", messagingCtx.GetUserID())
}

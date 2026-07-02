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
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// capturingTransport is a sentry.Transport implementation that records events in memory.
type capturingTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *capturingTransport) Configure(options sentry.ClientOptions) {}

func (t *capturingTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}

func (t *capturingTransport) Flush(timeout time.Duration) bool { return true }

func (t *capturingTransport) FlushWithContext(ctx context.Context) bool { return true }

func (t *capturingTransport) Close() {}

func (t *capturingTransport) Events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*sentry.Event, len(t.events))
	copy(out, t.events)
	return out
}

// setupSentry installs a capturing transport on the current hub and restores
// the previous client after the test.
func setupSentry(t *testing.T) *capturingTransport {
	t.Helper()
	transport := &capturingTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "",
		Transport: transport,
	})
	require.NoError(t, err)

	prevClient := sentry.CurrentHub().Client()
	sentry.CurrentHub().BindClient(client)
	t.Cleanup(func() {
		sentry.CurrentHub().BindClient(prevClient)
	})
	return transport
}

func newSentryTestRouter(m *SentryHTTPMiddleware) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(m.GinMiddleware())
	router.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	router.GET("/error", func(c *gin.Context) { c.String(http.StatusInternalServerError, "boom") })
	router.GET("/bad-request", func(c *gin.Context) { c.String(http.StatusBadRequest, "bad") })
	router.GET("/health", func(c *gin.Context) { c.String(http.StatusInternalServerError, "unhealthy") })
	return router
}

func TestSentryHTTPMiddleware_CapturesServerError(t *testing.T) {
	transport := setupSentry(t)
	m := NewSentryHTTPMiddleware(SentryHTTPMiddlewareOptions{})
	router := newSentryTestRouter(m)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	events := transport.Events()
	require.NotEmpty(t, events, "expected at least one captured event for 5xx response")
	event := events[0]
	assert.Equal(t, sentry.LevelError, event.Level)
	assert.Contains(t, event.Message, "HTTP 500")
	assert.Equal(t, "500", event.Tags["http.status_code"])
	assert.Equal(t, http.MethodGet, event.Tags["http.method"])
	assert.Equal(t, "/error", event.Tags["http.path"])
}

func TestSentryHTTPMiddleware_SuccessNotCaptured(t *testing.T) {
	transport := setupSentry(t)
	m := NewSentryHTTPMiddleware(SentryHTTPMiddlewareOptions{})
	router := newSentryTestRouter(m)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, transport.Events(), "2xx responses must not be captured")
}

func TestSentryHTTPMiddleware_IgnorePaths(t *testing.T) {
	transport := setupSentry(t)
	m := NewSentryHTTPMiddleware(SentryHTTPMiddlewareOptions{
		IgnorePaths: []string{"/health"},
	})
	router := newSentryTestRouter(m)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Empty(t, transport.Events(), "ignored paths must not be captured even on error")
}

func TestSentryHTTPMiddleware_IgnoreStatusCodes(t *testing.T) {
	transport := setupSentry(t)
	m := NewSentryHTTPMiddleware(SentryHTTPMiddlewareOptions{
		IgnoreStatusCodes: []int{http.StatusBadRequest},
	})
	router := newSentryTestRouter(m)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/bad-request", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, transport.Events(), "ignored status codes must not be captured")
}

func TestSentryHTTPMiddleware_PanicRecapturedAndRethrown(t *testing.T) {
	transport := setupSentry(t)
	m := NewSentryHTTPMiddleware(SentryHTTPMiddlewareOptions{})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Recovery must be registered before (outer of) the sentry middleware so it
	// catches the re-panic.
	router.Use(gin.Recovery())
	router.Use(m.GinMiddleware())
	router.GET("/panic", func(c *gin.Context) { panic("kaboom") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	events := transport.Events()
	require.NotEmpty(t, events, "panic must be captured before re-panicking")
	assert.Equal(t, sentry.LevelFatal, events[0].Level)
	assert.Equal(t, "true", events[0].Tags["panic"])
}

func TestSentryHTTPMiddleware_ShouldIgnorePath(t *testing.T) {
	m := NewSentryHTTPMiddleware(SentryHTTPMiddlewareOptions{
		IgnorePaths: []string{"/health", "/metrics"},
	})

	assert.True(t, m.shouldIgnorePath("/health"))
	assert.True(t, m.shouldIgnorePath("/metrics"))
	assert.False(t, m.shouldIgnorePath("/api/v1/users"))
	assert.False(t, m.shouldIgnorePath("/health/sub"))
}

func TestSentryHTTPMiddleware_ShouldCaptureStatus(t *testing.T) {
	m := NewSentryHTTPMiddleware(SentryHTTPMiddlewareOptions{
		IgnoreStatusCodes: []int{http.StatusNotFound},
	})

	assert.False(t, m.shouldCaptureStatus(http.StatusOK))
	assert.False(t, m.shouldCaptureStatus(http.StatusMovedPermanently))
	assert.False(t, m.shouldCaptureStatus(http.StatusNotFound), "ignored code must not be captured")
	assert.True(t, m.shouldCaptureStatus(http.StatusBadRequest))
	assert.True(t, m.shouldCaptureStatus(http.StatusInternalServerError))
}

func TestSentryHTTPMiddleware_GetSentryLevelForStatus(t *testing.T) {
	m := NewSentryHTTPMiddleware(SentryHTTPMiddlewareOptions{})

	assert.Equal(t, sentry.LevelError, m.getSentryLevelForStatus(http.StatusInternalServerError))
	assert.Equal(t, sentry.LevelError, m.getSentryLevelForStatus(http.StatusBadGateway))
	assert.Equal(t, sentry.LevelWarning, m.getSentryLevelForStatus(http.StatusBadRequest))
	assert.Equal(t, sentry.LevelWarning, m.getSentryLevelForStatus(http.StatusNotFound))
	assert.Equal(t, sentry.LevelInfo, m.getSentryLevelForStatus(http.StatusOK))
}

func TestSentryHTTPMiddleware_GetSpanStatusForHTTPStatus(t *testing.T) {
	m := NewSentryHTTPMiddleware(SentryHTTPMiddlewareOptions{})

	assert.Equal(t, sentry.SpanStatusOK, m.getSpanStatusForHTTPStatus(http.StatusOK))
	assert.Equal(t, sentry.SpanStatusOK, m.getSpanStatusForHTTPStatus(http.StatusNoContent))
	assert.Equal(t, sentry.SpanStatusInvalidArgument, m.getSpanStatusForHTTPStatus(http.StatusBadRequest))
	assert.Equal(t, sentry.SpanStatusInternalError, m.getSpanStatusForHTTPStatus(http.StatusInternalServerError))
	assert.Equal(t, sentry.SpanStatusUnknown, m.getSpanStatusForHTTPStatus(http.StatusMovedPermanently))
}

func TestFlattenHeaders(t *testing.T) {
	headers := http.Header{
		"Content-Type": []string{"application/json"},
		"Accept":       []string{"application/json", "text/plain"},
	}

	flat := flattenHeaders(headers)
	assert.Equal(t, "application/json", flat["Content-Type"])
	assert.Equal(t, []string{"application/json", "text/plain"}, flat["Accept"])
}

func TestSentryGRPCInterceptor_UnaryCapturesError(t *testing.T) {
	transport := setupSentry(t)
	interceptor := NewSentryGRPCInterceptor(SentryGRPCInterceptorOptions{})

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, status.Error(codes.Internal, "internal failure")
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("user-id", "user-42"))
	info := &grpc.UnaryServerInfo{FullMethod: "/swit.user.v1.UserService/GetUser"}

	resp, err := interceptor.UnaryServerInterceptor()(ctx, nil, info, handler)
	assert.Nil(t, resp)
	require.Error(t, err)

	events := transport.Events()
	require.NotEmpty(t, events, "expected error to be captured")
	assert.Equal(t, "swit.user.v1.UserService", events[0].Tags["grpc.service"])
	assert.Equal(t, "/swit.user.v1.UserService/GetUser", events[0].Tags["grpc.method"])
}

func TestSentryGRPCInterceptor_UnarySuccessNotCaptured(t *testing.T) {
	transport := setupSentry(t)
	interceptor := NewSentryGRPCInterceptor(SentryGRPCInterceptorOptions{})

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}
	resp, err := interceptor.UnaryServerInterceptor()(context.Background(), nil, info, handler)
	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.Empty(t, transport.Events())
}

func TestSentryGRPCInterceptor_UnaryIgnoredCode(t *testing.T) {
	transport := setupSentry(t)
	interceptor := NewSentryGRPCInterceptor(SentryGRPCInterceptorOptions{
		IgnoreStatusCodes: []codes.Code{codes.NotFound},
	})

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, status.Error(codes.NotFound, "missing")
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}
	_, err := interceptor.UnaryServerInterceptor()(context.Background(), nil, info, handler)
	require.Error(t, err)
	assert.Empty(t, transport.Events(), "ignored gRPC codes must not be captured")
}

func TestSentryGRPCInterceptor_UnaryPanicRecapturedAndRethrown(t *testing.T) {
	transport := setupSentry(t)
	interceptor := NewSentryGRPCInterceptor(SentryGRPCInterceptorOptions{})

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("grpc kaboom")
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}
	assert.PanicsWithValue(t, "grpc kaboom", func() {
		_, _ = interceptor.UnaryServerInterceptor()(context.Background(), nil, info, handler)
	})

	events := transport.Events()
	require.NotEmpty(t, events, "panic must be captured before re-panicking")
	assert.Equal(t, sentry.LevelFatal, events[0].Level)
}

// fakeServerStream is a minimal grpc.ServerStream for interceptor testing.
type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeServerStream) Context() context.Context { return f.ctx }

func TestSentryGRPCInterceptor_StreamCapturesError(t *testing.T) {
	transport := setupSentry(t)
	interceptor := NewSentryGRPCInterceptor(SentryGRPCInterceptorOptions{})

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return status.Error(codes.Unavailable, "stream failure")
	}

	stream := &fakeServerStream{ctx: context.Background()}
	info := &grpc.StreamServerInfo{FullMethod: "/svc/StreamMethod"}

	err := interceptor.StreamServerInterceptor()(nil, stream, info, handler)
	require.Error(t, err)
	assert.NotEmpty(t, transport.Events())
}

func TestSentryGRPCInterceptor_StreamSuccessAndContextPropagation(t *testing.T) {
	transport := setupSentry(t)
	interceptor := NewSentryGRPCInterceptor(SentryGRPCInterceptorOptions{})

	var handlerCtx context.Context
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		handlerCtx = stream.Context()
		return nil
	}

	stream := &fakeServerStream{ctx: context.Background()}
	info := &grpc.StreamServerInfo{FullMethod: "/svc/StreamMethod"}

	err := interceptor.StreamServerInterceptor()(nil, stream, info, handler)
	assert.NoError(t, err)
	assert.Empty(t, transport.Events())

	// The wrapped stream must expose a context carrying the sentry hub.
	require.NotNil(t, handlerCtx)
	assert.NotNil(t, sentry.GetHubFromContext(handlerCtx))
}

func TestSentryGRPCInterceptor_ShouldCaptureGRPCStatus(t *testing.T) {
	interceptor := NewSentryGRPCInterceptor(SentryGRPCInterceptorOptions{
		IgnoreStatusCodes: []codes.Code{codes.Canceled},
	})

	assert.False(t, interceptor.shouldCaptureGRPCStatus(codes.OK))
	assert.False(t, interceptor.shouldCaptureGRPCStatus(codes.Canceled))
	assert.True(t, interceptor.shouldCaptureGRPCStatus(codes.Internal))
	assert.True(t, interceptor.shouldCaptureGRPCStatus(codes.NotFound))
}

func TestSentryGRPCInterceptor_GetSentryLevelForGRPCCode(t *testing.T) {
	interceptor := NewSentryGRPCInterceptor(SentryGRPCInterceptorOptions{})

	assert.Equal(t, sentry.LevelError, interceptor.getSentryLevelForGRPCCode(codes.Internal))
	assert.Equal(t, sentry.LevelError, interceptor.getSentryLevelForGRPCCode(codes.DataLoss))
	assert.Equal(t, sentry.LevelError, interceptor.getSentryLevelForGRPCCode(codes.Unknown))
	assert.Equal(t, sentry.LevelWarning, interceptor.getSentryLevelForGRPCCode(codes.InvalidArgument))
	assert.Equal(t, sentry.LevelWarning, interceptor.getSentryLevelForGRPCCode(codes.NotFound))
	assert.Equal(t, sentry.LevelInfo, interceptor.getSentryLevelForGRPCCode(codes.DeadlineExceeded))
	assert.Equal(t, sentry.LevelInfo, interceptor.getSentryLevelForGRPCCode(codes.ResourceExhausted))
	assert.Equal(t, sentry.LevelWarning, interceptor.getSentryLevelForGRPCCode(codes.Canceled))
}

func TestSentryGRPCInterceptor_GetSpanStatusForGRPCCode(t *testing.T) {
	interceptor := NewSentryGRPCInterceptor(SentryGRPCInterceptorOptions{})

	tests := []struct {
		code codes.Code
		want sentry.SpanStatus
	}{
		{codes.OK, sentry.SpanStatusOK},
		{codes.Canceled, sentry.SpanStatusCanceled},
		{codes.InvalidArgument, sentry.SpanStatusInvalidArgument},
		{codes.DeadlineExceeded, sentry.SpanStatusDeadlineExceeded},
		{codes.NotFound, sentry.SpanStatusNotFound},
		{codes.AlreadyExists, sentry.SpanStatusAlreadyExists},
		{codes.PermissionDenied, sentry.SpanStatusPermissionDenied},
		{codes.ResourceExhausted, sentry.SpanStatusResourceExhausted},
		{codes.FailedPrecondition, sentry.SpanStatusFailedPrecondition},
		{codes.Aborted, sentry.SpanStatusAborted},
		{codes.OutOfRange, sentry.SpanStatusOutOfRange},
		{codes.Unimplemented, sentry.SpanStatusUnimplemented},
		{codes.Internal, sentry.SpanStatusInternalError},
		{codes.Unavailable, sentry.SpanStatusUnavailable},
		{codes.DataLoss, sentry.SpanStatusDataLoss},
		{codes.Unauthenticated, sentry.SpanStatusUnauthenticated},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, interceptor.getSpanStatusForGRPCCode(tt.code), "code %s", tt.code)
	}
}

func TestExtractServiceFromMethod(t *testing.T) {
	assert.Equal(t, "swit.user.v1.UserService", extractServiceFromMethod("/swit.user.v1.UserService/GetUser"))
	assert.Equal(t, "svc", extractServiceFromMethod("/svc/Method"))
	assert.Equal(t, "unknown", extractServiceFromMethod("no-slashes"))
}

func TestExtractFromMetadata(t *testing.T) {
	md := metadata.Pairs("user-id", "42", "multi", "a", "multi", "b")

	assert.Equal(t, "42", extractFromMetadata(md, "user-id"))
	assert.Equal(t, "a", extractFromMetadata(md, "multi"))
	assert.Equal(t, "", extractFromMetadata(md, "missing"))
}

func TestFlattenMetadata(t *testing.T) {
	md := metadata.Pairs("single", "one", "multi", "a", "multi", "b")

	flat := flattenMetadata(md)
	assert.Equal(t, "one", flat["single"])
	assert.Equal(t, []string{"a", "b"}, flat["multi"])
}

func TestSentryServerStream_Context(t *testing.T) {
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "value")
	wrapped := &sentryServerStream{ServerStream: &fakeServerStream{ctx: context.Background()}, ctx: ctx}

	assert.Equal(t, "value", wrapped.Context().Value(ctxKey{}))
}

func TestSentryGRPCInterceptor_NonStatusError(t *testing.T) {
	transport := setupSentry(t)
	interceptor := NewSentryGRPCInterceptor(SentryGRPCInterceptorOptions{})

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, errors.New("plain error")
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}
	_, err := interceptor.UnaryServerInterceptor()(context.Background(), nil, info, handler)
	require.Error(t, err)

	// Plain errors map to codes.Unknown which should be captured.
	assert.NotEmpty(t, transport.Events())
}

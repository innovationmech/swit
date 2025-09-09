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

package middleware

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// TestLoggingMiddleware tests the structured logging middleware.
func TestLoggingMiddleware(t *testing.T) {
	// Create a test message
	message := &messaging.Message{
		ID:            "test-message-1",
		Topic:         "test-topic",
		Payload:       []byte(`{"data": "test"}`),
		Headers:       map[string]string{"x-test": "value"},
		CorrelationID: "correlation-123",
		Timestamp:     time.Now(),
	}

	// Create logging middleware with test configuration
	config := DefaultLoggingConfig()
	config.LogMessageContent = true
	config.MaxMessageSize = 1024

	middleware := NewStructuredLoggingMiddleware(config)

	// Create a test handler
	handlerCalled := false
	testHandler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		handlerCalled = true
		return nil
	})

	// Wrap the handler with logging middleware
	wrappedHandler := middleware.Wrap(testHandler)

	// Execute the wrapped handler
	ctx := context.Background()
	err := wrappedHandler.Handle(ctx, message)

	// Verify results
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
}

// TestLoggingMiddlewareError tests logging middleware error handling.
func TestLoggingMiddlewareError(t *testing.T) {
	message := &messaging.Message{
		ID:    "test-message-error",
		Topic: "test-topic",
	}

	config := DefaultLoggingConfig()
	middleware := NewStructuredLoggingMiddleware(config)

	// Create a handler that returns an error
	testError := messaging.NewProcessingError("test error", nil)
	testHandler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		return testError
	})

	wrappedHandler := middleware.Wrap(testHandler)

	ctx := context.Background()
	err := wrappedHandler.Handle(ctx, message)

	if err == nil {
		t.Error("Expected error to be returned")
	}
	if err != testError {
		t.Errorf("Expected error %v, got %v", testError, err)
	}
}

// TestMetricsMiddleware tests the metrics collection middleware.
func TestMetricsMiddleware(t *testing.T) {
	message := &messaging.Message{
		ID:      "test-message-metrics",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	config := DefaultMetricsConfig()
	middleware := NewStandardMetricsMiddleware(config)

	handlerCalled := false
	testHandler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		handlerCalled = true
		return nil
	})

	wrappedHandler := middleware.Wrap(testHandler)

	ctx := context.Background()
	err := wrappedHandler.Handle(ctx, message)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}

	// Verify metrics were collected
	metrics := middleware.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Expected metrics to be collected")
	}
}

// TestTracingMiddleware tests the distributed tracing middleware.
func TestTracingMiddleware(t *testing.T) {
	message := &messaging.Message{
		ID:            "test-message-tracing",
		Topic:         "test-topic",
		Headers:       map[string]string{},
		CorrelationID: "trace-correlation-123",
	}

	config := DefaultTracingConfig()
	config.Tracer = NewInMemoryTracer()
	config.RecordMessageContent = true

	middleware := NewDistributedTracingMiddleware(config)

	handlerCalled := false
	testHandler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		handlerCalled = true
		return nil
	})

	wrappedHandler := middleware.Wrap(testHandler)

	ctx := context.Background()
	err := wrappedHandler.Handle(ctx, message)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}

	// Verify spans were created
	inMemoryTracer, ok := config.Tracer.(*InMemoryTracer)
	if !ok {
		t.Fatal("Expected InMemoryTracer")
	}

	spans := inMemoryTracer.GetSpans()
	if len(spans) == 0 {
		t.Error("Expected spans to be created")
	}

	span := spans[0]
	if !strings.Contains(span.Name, "process_message") {
		t.Errorf("Expected span name to contain 'process_message', got %s", span.Name)
	}
}

// TestRateLimitMiddleware tests the rate limiting middleware.
func TestRateLimitMiddleware(t *testing.T) {
	message := &messaging.Message{
		ID:    "test-message-ratelimit",
		Topic: "test-topic",
	}

	config := DefaultRateLimitConfig()
	config.DefaultLimit = 2
	config.Window = time.Second

	middleware := NewAdaptiveRateLimitMiddleware(config)

	handlerCalled := 0
	testHandler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		handlerCalled++
		return nil
	})

	wrappedHandler := middleware.Wrap(testHandler)
	ctx := context.Background()

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		err := wrappedHandler.Handle(ctx, message)
		if err != nil {
			t.Errorf("Expected no error on request %d, got %v", i+1, err)
		}
	}

	// Third request should be rate limited
	err := wrappedHandler.Handle(ctx, message)
	if err == nil {
		t.Error("Expected rate limit error")
	}

	if handlerCalled != 2 {
		t.Errorf("Expected handler to be called 2 times, got %d", handlerCalled)
	}
}

// TestCorrelationMiddleware tests the correlation ID middleware.
func TestCorrelationMiddleware(t *testing.T) {
	message := &messaging.Message{
		ID:    "test-message-correlation",
		Topic: "test-topic",
	}

	config := DefaultCorrelationConfig()
	middleware := NewCorrelationMiddleware(config)

	var capturedCorrelation *CorrelationContext
	testHandler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		capturedCorrelation = GetCorrelationFromContext(ctx)
		return nil
	})

	wrappedHandler := middleware.Wrap(testHandler)

	ctx := context.Background()
	err := wrappedHandler.Handle(ctx, message)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify correlation ID was generated and propagated
	if capturedCorrelation == nil {
		t.Error("Expected correlation context to be available")
	} else {
		if capturedCorrelation.ID == "" {
			t.Error("Expected correlation ID to be generated")
		}
		if !strings.HasPrefix(capturedCorrelation.ID, "corr-") {
			t.Errorf("Expected correlation ID to have 'corr-' prefix, got %s", capturedCorrelation.ID)
		}
	}

	// Verify correlation ID was set on message
	if message.CorrelationID == "" {
		t.Error("Expected message correlation ID to be set")
	}
}

// TestCorrelationMiddlewareExistingID tests correlation middleware with existing ID.
func TestCorrelationMiddlewareExistingID(t *testing.T) {
	existingID := "existing-correlation-123"
	message := &messaging.Message{
		ID:            "test-message-existing-correlation",
		Topic:         "test-topic",
		CorrelationID: existingID,
		Headers:       map[string]string{"x-correlation-id": existingID},
	}

	config := DefaultCorrelationConfig()
	middleware := NewCorrelationMiddleware(config)

	var capturedCorrelation *CorrelationContext
	testHandler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		capturedCorrelation = GetCorrelationFromContext(ctx)
		return nil
	})

	wrappedHandler := middleware.Wrap(testHandler)

	ctx := context.Background()
	err := wrappedHandler.Handle(ctx, message)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify existing correlation ID was preserved
	if capturedCorrelation == nil {
		t.Error("Expected correlation context to be available")
	} else {
		if capturedCorrelation.ID != existingID {
			t.Errorf("Expected correlation ID to be %s, got %s", existingID, capturedCorrelation.ID)
		}
	}
}

// TestMiddlewareChaining tests chaining multiple middleware together.
func TestMiddlewareChaining(t *testing.T) {
	message := &messaging.Message{
		ID:      "test-message-chain",
		Topic:   "test-topic",
		Payload: []byte("test data"),
	}

	// Create middleware components
	loggingConfig := DefaultLoggingConfig()
	loggingMiddleware := NewStructuredLoggingMiddleware(loggingConfig)

	metricsConfig := DefaultMetricsConfig()
	metricsMiddleware := NewStandardMetricsMiddleware(metricsConfig)

	correlationConfig := DefaultCorrelationConfig()
	correlationMiddleware := NewCorrelationMiddleware(correlationConfig)

	// Create test handler
	handlerCalled := false
	var capturedContext context.Context
	testHandler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		handlerCalled = true
		capturedContext = ctx
		return nil
	})

	// Chain middleware: correlation -> metrics -> logging -> handler
	chainedHandler := loggingMiddleware.Wrap(
		metricsMiddleware.Wrap(
			correlationMiddleware.Wrap(testHandler),
		),
	)

	ctx := context.Background()
	err := chainedHandler.Handle(ctx, message)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}

	// Verify correlation was propagated through the chain
	correlation := GetCorrelationFromContext(capturedContext)
	if correlation == nil {
		t.Error("Expected correlation to be available in context")
	}

	// Verify metrics were collected
	metrics := metricsMiddleware.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Expected metrics to be collected")
	}

	// Verify correlation ID was set on message
	if message.CorrelationID == "" {
		t.Error("Expected message correlation ID to be set")
	}
}

// TestMiddlewareErrorHandling tests error handling through middleware chain.
func TestMiddlewareErrorHandling(t *testing.T) {
	message := &messaging.Message{
		ID:    "test-message-error-chain",
		Topic: "test-topic",
	}

	// Create middleware components
	loggingConfig := DefaultLoggingConfig()
	loggingMiddleware := NewStructuredLoggingMiddleware(loggingConfig)

	metricsConfig := DefaultMetricsConfig()
	metricsMiddleware := NewStandardMetricsMiddleware(metricsConfig)

	// Create test handler that returns an error
	testError := messaging.NewProcessingError("chain test error", nil)
	testHandler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		return testError
	})

	// Chain middleware
	chainedHandler := loggingMiddleware.Wrap(
		metricsMiddleware.Wrap(testHandler),
	)

	ctx := context.Background()
	err := chainedHandler.Handle(ctx, message)

	if err == nil {
		t.Error("Expected error to be returned")
	}
	if err != testError {
		t.Errorf("Expected error %v, got %v", testError, err)
	}
}

// TestRateLimitSlidingWindow tests sliding window rate limiting.
func TestRateLimitSlidingWindow(t *testing.T) {
	config := DefaultRateLimitConfig()
	config.Strategy = RateLimitStrategySlidingWindow
	config.DefaultLimit = 3
	config.Window = 100 * time.Millisecond

	middleware := NewAdaptiveRateLimitMiddleware(config)

	message := &messaging.Message{
		ID:    "test-sliding-window",
		Topic: "test-topic",
	}

	handlerCallCount := 0
	testHandler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		handlerCallCount++
		return nil
	})

	wrappedHandler := middleware.Wrap(testHandler)
	ctx := context.Background()

	// Send 3 requests quickly - should all succeed
	for i := 0; i < 3; i++ {
		err := wrappedHandler.Handle(ctx, message)
		if err != nil {
			t.Errorf("Expected no error on request %d, got %v", i+1, err)
		}
	}

	// 4th request should be rate limited
	err := wrappedHandler.Handle(ctx, message)
	if err == nil {
		t.Error("Expected rate limit error")
	}

	if handlerCallCount != 3 {
		t.Errorf("Expected handler to be called 3 times, got %d", handlerCallCount)
	}

	// Wait for window to slide
	time.Sleep(150 * time.Millisecond)

	// Should be able to send more requests
	err = wrappedHandler.Handle(ctx, message)
	if err != nil {
		t.Errorf("Expected no error after window slide, got %v", err)
	}

	if handlerCallCount != 4 {
		t.Errorf("Expected handler to be called 4 times after window slide, got %d", handlerCallCount)
	}
}

// BenchmarkMiddlewareChain benchmarks the performance of chained middleware.
func BenchmarkMiddlewareChain(b *testing.B) {
	message := &messaging.Message{
		ID:      "benchmark-message",
		Topic:   "benchmark-topic",
		Payload: make([]byte, 1024),
	}

	// Create middleware chain
	loggingConfig := DefaultLoggingConfig()
	loggingConfig.Level = LogLevelError // Reduce logging overhead
	loggingMiddleware := NewStructuredLoggingMiddleware(loggingConfig)

	metricsMiddleware := NewStandardMetricsMiddleware(DefaultMetricsConfig())
	correlationMiddleware := NewCorrelationMiddleware(DefaultCorrelationConfig())

	testHandler := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		return nil
	})

	chainedHandler := loggingMiddleware.Wrap(
		metricsMiddleware.Wrap(
			correlationMiddleware.Wrap(testHandler),
		),
	)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chainedHandler.Handle(ctx, message)
	}
}

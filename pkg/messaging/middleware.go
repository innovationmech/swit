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

package messaging

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// MiddlewareFunc is a function adapter that implements Middleware interface.
// It allows using functions as middleware for simple use cases.
type MiddlewareFunc func(next MessageHandler) MessageHandler

// Name returns a default name for the middleware function.
func (mf MiddlewareFunc) Name() string {
	return "middleware-func"
}

// Wrap implements Middleware interface for MiddlewareFunc.
func (mf MiddlewareFunc) Wrap(next MessageHandler) MessageHandler {
	return mf(next)
}

// MiddlewareChain represents a chain of middleware that can be applied to handlers.
type MiddlewareChain struct {
	middleware []Middleware
	mutex      sync.RWMutex
}

// NewMiddlewareChain creates a new middleware chain with the given middleware.
func NewMiddlewareChain(middleware ...Middleware) *MiddlewareChain {
	return &MiddlewareChain{
		middleware: append([]Middleware(nil), middleware...),
	}
}

// Add adds middleware to the end of the chain.
func (mc *MiddlewareChain) Add(middleware ...Middleware) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.middleware = append(mc.middleware, middleware...)
}

// Prepend adds middleware to the beginning of the chain.
func (mc *MiddlewareChain) Prepend(middleware ...Middleware) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.middleware = append(middleware, mc.middleware...)
}

// Remove removes middleware by name from the chain.
func (mc *MiddlewareChain) Remove(name string) bool {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	for i, mw := range mc.middleware {
		if mw.Name() == name {
			mc.middleware = append(mc.middleware[:i], mc.middleware[i+1:]...)
			return true
		}
	}
	return false
}

// Build creates a handler with the middleware chain applied.
func (mc *MiddlewareChain) Build(handler MessageHandler) MessageHandler {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	// Apply middleware in reverse order so the first middleware is outermost
	for i := len(mc.middleware) - 1; i >= 0; i-- {
		handler = mc.middleware[i].Wrap(handler)
	}
	return handler
}

// GetMiddleware returns a copy of the middleware slice.
func (mc *MiddlewareChain) GetMiddleware() []Middleware {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	result := make([]Middleware, len(mc.middleware))
	copy(result, mc.middleware)
	return result
}

// Len returns the number of middleware in the chain.
func (mc *MiddlewareChain) Len() int {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return len(mc.middleware)
}

// LoggingMiddleware logs message processing events.
type LoggingMiddleware struct {
	logger Logger
	level  LogLevel
}

// Logger interface for logging middleware.
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// LogLevel defines logging levels.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// NewLoggingMiddleware creates a new logging middleware.
func NewLoggingMiddleware(logger Logger, level LogLevel) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
		level:  level,
	}
}

// Name returns the middleware name.
func (lm *LoggingMiddleware) Name() string {
	return "logging"
}

// Wrap wraps a handler with logging functionality.
func (lm *LoggingMiddleware) Wrap(next MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		start := time.Now()

		if lm.level <= LogLevelDebug {
			lm.logger.Debug("processing message",
				"message_id", message.ID,
				"topic", message.Topic,
				"correlation_id", message.CorrelationID)
		}

		err := next.Handle(ctx, message)

		processingTime := time.Since(start)

		if err != nil {
			if lm.level <= LogLevelError {
				lm.logger.Error("message processing failed",
					"message_id", message.ID,
					"topic", message.Topic,
					"error", err,
					"processing_time", processingTime)
			}
		} else {
			if lm.level <= LogLevelInfo {
				lm.logger.Info("message processed successfully",
					"message_id", message.ID,
					"topic", message.Topic,
					"processing_time", processingTime)
			}
		}

		return err
	})
}

// MetricsMiddleware collects metrics about message processing.
type MetricsMiddleware struct {
	metricsCollector MetricsCollector
	mutex            sync.RWMutex
	counters         map[string]*Counter
	histograms       map[string]*Histogram
}

// MetricsCollector interface for collecting metrics.
type MetricsCollector interface {
	IncrementCounter(name string, labels map[string]string, value float64)
	ObserveHistogram(name string, labels map[string]string, value float64)
	SetGauge(name string, labels map[string]string, value float64)
}

// Counter represents a metric counter.
type Counter struct {
	value int64
	mutex sync.RWMutex
}

// Histogram represents a metric histogram.
type Histogram struct {
	observations []float64
	mutex        sync.RWMutex
}

// NewMetricsMiddleware creates a new metrics middleware.
func NewMetricsMiddleware(collector MetricsCollector) *MetricsMiddleware {
	return &MetricsMiddleware{
		metricsCollector: collector,
		counters:         make(map[string]*Counter),
		histograms:       make(map[string]*Histogram),
	}
}

// Name returns the middleware name.
func (mm *MetricsMiddleware) Name() string {
	return "metrics"
}

// Wrap wraps a handler with metrics collection functionality.
func (mm *MetricsMiddleware) Wrap(next MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		start := time.Now()

		labels := map[string]string{
			"topic": message.Topic,
		}

		// Increment received counter
		mm.metricsCollector.IncrementCounter("messages_received_total", labels, 1)

		err := next.Handle(ctx, message)

		processingTime := time.Since(start).Seconds()

		if err != nil {
			// Increment failed counter
			mm.metricsCollector.IncrementCounter("messages_failed_total", labels, 1)

			// Track error types
			errorLabels := map[string]string{
				"topic":      message.Topic,
				"error_type": getErrorType(err),
			}
			mm.metricsCollector.IncrementCounter("message_errors_by_type_total", errorLabels, 1)
		} else {
			// Increment processed counter
			mm.metricsCollector.IncrementCounter("messages_processed_total", labels, 1)
		}

		// Record processing time histogram
		mm.metricsCollector.ObserveHistogram("message_processing_duration_seconds", labels, processingTime)

		return err
	})
}

// getErrorType extracts error type from error for metrics labeling.
func getErrorType(err error) string {
	if msgErr, ok := err.(*MessagingError); ok {
		return string(msgErr.Code)
	}
	return "unknown"
}

// TimeoutMiddleware adds timeout functionality to message processing.
type TimeoutMiddleware struct {
	timeout time.Duration
}

// NewTimeoutMiddleware creates a new timeout middleware.
func NewTimeoutMiddleware(timeout time.Duration) *TimeoutMiddleware {
	return &TimeoutMiddleware{
		timeout: timeout,
	}
}

// Name returns the middleware name.
func (tm *TimeoutMiddleware) Name() string {
	return "timeout"
}

// Wrap wraps a handler with timeout functionality.
func (tm *TimeoutMiddleware) Wrap(next MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		timeoutCtx, cancel := context.WithTimeout(ctx, tm.timeout)
		defer cancel()

		type result struct {
			err error
		}

		resultCh := make(chan result, 1)

		go func() {
			err := next.Handle(timeoutCtx, message)
			resultCh <- result{err: err}
		}()

		select {
		case res := <-resultCh:
			return res.err
		case <-timeoutCtx.Done():
			return NewProcessingError(fmt.Sprintf("message processing timeout after %v", tm.timeout), timeoutCtx.Err())
		}
	})
}

// RetryMiddleware provides retry functionality for failed message processing.
type RetryMiddleware struct {
	config RetryConfig
}

// NewRetryMiddleware creates a new retry middleware.
func NewRetryMiddleware(config RetryConfig) *RetryMiddleware {
	return &RetryMiddleware{
		config: config,
	}
}

// Name returns the middleware name.
func (rm *RetryMiddleware) Name() string {
	return "retry"
}

// Wrap wraps a handler with retry functionality.
func (rm *RetryMiddleware) Wrap(next MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		var lastErr error

		for attempt := 0; attempt <= rm.config.MaxAttempts; attempt++ {
			if attempt > 0 {
				// Calculate delay with exponential backoff and jitter
				delay := rm.calculateDelay(attempt)

				select {
				case <-time.After(delay):
					// Continue with retry
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			err := next.Handle(ctx, message)
			if err == nil {
				return nil
			}

			lastErr = err

			// Check if error is retryable
			if msgErr, ok := err.(*MessagingError); ok && !msgErr.Retryable {
				return err
			}

			// Check if we should continue retrying based on context
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}

		return NewProcessingError(fmt.Sprintf("max retry attempts (%d) exceeded", rm.config.MaxAttempts), lastErr)
	})
}

// calculateDelay calculates the delay for retry attempts with exponential backoff and jitter.
func (rm *RetryMiddleware) calculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	delay := rm.config.InitialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * rm.config.Multiplier)
		if delay > rm.config.MaxDelay {
			delay = rm.config.MaxDelay
			break
		}
	}

	// Add jitter to prevent thundering herd
	if rm.config.Jitter > 0 {
		jitterAmount := time.Duration(float64(delay) * rm.config.Jitter)
		// Simple jitter: random value between -jitterAmount/2 and +jitterAmount/2
		// For simplicity, we'll use a fixed jitter here
		delay += jitterAmount / 2
	}

	return delay
}

// RecoveryMiddleware provides panic recovery for message handlers.
type RecoveryMiddleware struct {
	logger Logger
}

// NewRecoveryMiddleware creates a new recovery middleware.
func NewRecoveryMiddleware(logger Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		logger: logger,
	}
}

// Name returns the middleware name.
func (rm *RecoveryMiddleware) Name() string {
	return "recovery"
}

// Wrap wraps a handler with panic recovery functionality.
func (rm *RecoveryMiddleware) Wrap(next MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, message *Message) (err error) {
		defer func() {
			if r := recover(); r != nil {
				if rm.logger != nil {
					rm.logger.Error("panic recovered in message handler",
						"message_id", message.ID,
						"topic", message.Topic,
						"panic", r)
				} else {
					log.Printf("panic recovered in message handler: %v (message_id: %s, topic: %s)",
						r, message.ID, message.Topic)
				}

				err = NewProcessingError(fmt.Sprintf("panic recovered: %v", r), nil)
			}
		}()

		return next.Handle(ctx, message)
	})
}

// TracingMiddleware provides distributed tracing support for message processing.
type TracingMiddleware struct {
	tracer   Tracer
	spanName string
}

// Tracer interface for tracing functionality.
type Tracer interface {
	StartSpan(ctx context.Context, operationName string, tags map[string]interface{}) (context.Context, Span)
}

// Span interface for tracing spans.
type Span interface {
	SetTag(key string, value interface{})
	SetError(err error)
	Finish()
}

// NewTracingMiddleware creates a new tracing middleware.
func NewTracingMiddleware(tracer Tracer, spanName string) *TracingMiddleware {
	return &TracingMiddleware{
		tracer:   tracer,
		spanName: spanName,
	}
}

// Name returns the middleware name.
func (tm *TracingMiddleware) Name() string {
	return "tracing"
}

// Wrap wraps a handler with tracing functionality.
func (tm *TracingMiddleware) Wrap(next MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		spanName := tm.spanName
		if spanName == "" {
			spanName = fmt.Sprintf("process_message_%s", message.Topic)
		}

		tags := map[string]interface{}{
			"message.id":             message.ID,
			"message.topic":          message.Topic,
			"message.correlation_id": message.CorrelationID,
			"component":              "messaging",
		}

		spanCtx, span := tm.tracer.StartSpan(ctx, spanName, tags)
		defer span.Finish()

		err := next.Handle(spanCtx, message)

		if err != nil {
			span.SetError(err)
			span.SetTag("error", true)
		}

		return err
	})
}

// ValidationMiddleware provides message validation functionality.
type ValidationMiddleware struct {
	validator MessageValidator
}

// MessageValidator interface for message validation.
type MessageValidator interface {
	Validate(ctx context.Context, message *Message) error
}

// NewValidationMiddleware creates a new validation middleware.
func NewValidationMiddleware(validator MessageValidator) *ValidationMiddleware {
	return &ValidationMiddleware{
		validator: validator,
	}
}

// Name returns the middleware name.
func (vm *ValidationMiddleware) Name() string {
	return "validation"
}

// Wrap wraps a handler with validation functionality.
func (vm *ValidationMiddleware) Wrap(next MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		if err := vm.validator.Validate(ctx, message); err != nil {
			return NewProcessingError(fmt.Sprintf("message validation failed: %v", err), err)
		}

		return next.Handle(ctx, message)
	})
}

// RateLimitMiddleware provides rate limiting functionality for message processing.
type RateLimitMiddleware struct {
	limiter RateLimiter
}

// RateLimiter interface for rate limiting.
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	Wait(ctx context.Context, key string) error
}

// NewRateLimitMiddleware creates a new rate limit middleware.
func NewRateLimitMiddleware(limiter RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
	}
}

// Name returns the middleware name.
func (rlm *RateLimitMiddleware) Name() string {
	return "rate_limit"
}

// Wrap wraps a handler with rate limiting functionality.
func (rlm *RateLimitMiddleware) Wrap(next MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		// Use topic as rate limit key, but could be customized
		key := message.Topic

		if err := rlm.limiter.Wait(ctx, key); err != nil {
			return NewProcessingError(fmt.Sprintf("rate limit exceeded for topic %s", message.Topic), err)
		}

		return next.Handle(ctx, message)
	})
}

// ContextMiddleware provides context enrichment functionality.
type ContextMiddleware struct {
	enricher ContextEnricher
}

// ContextEnricher interface for context enrichment.
type ContextEnricher interface {
	Enrich(ctx context.Context, message *Message) context.Context
}

// NewContextMiddleware creates a new context middleware.
func NewContextMiddleware(enricher ContextEnricher) *ContextMiddleware {
	return &ContextMiddleware{
		enricher: enricher,
	}
}

// Name returns the middleware name.
func (cm *ContextMiddleware) Name() string {
	return "context"
}

// Wrap wraps a handler with context enrichment functionality.
func (cm *ContextMiddleware) Wrap(next MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		enrichedCtx := cm.enricher.Enrich(ctx, message)
		return next.Handle(enrichedCtx, message)
	})
}

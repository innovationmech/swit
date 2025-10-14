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

package retry

import (
	"context"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// RetryableFunc is a function that can be retried.
type RetryableFunc func(ctx context.Context) (interface{}, error)

// RetryPolicy defines the interface for retry policies.
// This interface is compatible with the saga.RetryPolicy interface.
type RetryPolicy interface {
	// ShouldRetry determines if an operation should be retried.
	ShouldRetry(err error, attempt int) bool

	// GetRetryDelay returns the delay before the next retry attempt.
	GetRetryDelay(attempt int) time.Duration

	// GetMaxAttempts returns the maximum number of retry attempts.
	GetMaxAttempts() int
}

// ErrorClassifier classifies errors to determine if they are retryable.
type ErrorClassifier interface {
	// IsRetryable determines if an error is retryable.
	IsRetryable(err error) bool

	// IsPermanent determines if an error is permanent (non-retryable).
	IsPermanent(err error) bool
}

// DefaultErrorClassifier is the default error classifier that treats all errors as retryable.
type DefaultErrorClassifier struct{}

// IsRetryable returns true for all non-nil errors.
func (d *DefaultErrorClassifier) IsRetryable(err error) bool {
	return err != nil
}

// IsPermanent returns false for all errors (none are considered permanent by default).
func (d *DefaultErrorClassifier) IsPermanent(err error) bool {
	return false
}

// Executor executes operations with retry logic based on a retry policy.
type Executor struct {
	// policy is the retry policy to use
	policy RetryPolicy

	// circuitBreaker is an optional circuit breaker to prevent cascading failures
	circuitBreaker *CircuitBreaker

	// errorClassifier classifies errors as retryable or permanent
	errorClassifier ErrorClassifier

	// logger is the logger for retry operations
	logger *zap.Logger

	// metrics collectors
	metricsCollector *RetryMetricsCollector

	// onRetry is called before each retry attempt (optional)
	onRetry func(attempt int, err error, delay time.Duration)

	// onSuccess is called when the operation succeeds (optional)
	onSuccess func(attempt int, duration time.Duration, result interface{})

	// onFailure is called when all retries are exhausted (optional)
	onFailure func(attempts int, duration time.Duration, lastErr error)
}

// ExecutorOption is a functional option for configuring the Executor.
type ExecutorOption func(*Executor)

// WithCircuitBreaker adds a circuit breaker to the executor.
func WithCircuitBreaker(cb *CircuitBreaker) ExecutorOption {
	return func(e *Executor) {
		e.circuitBreaker = cb
	}
}

// WithErrorClassifier sets a custom error classifier.
func WithErrorClassifier(classifier ErrorClassifier) ExecutorOption {
	return func(e *Executor) {
		e.errorClassifier = classifier
	}
}

// WithLogger sets a custom logger.
func WithLogger(l *zap.Logger) ExecutorOption {
	return func(e *Executor) {
		e.logger = l
	}
}

// WithMetrics adds metrics collection to the executor.
func WithMetrics(collector *RetryMetricsCollector) ExecutorOption {
	return func(e *Executor) {
		e.metricsCollector = collector
	}
}

// NewExecutor creates a new retry executor with the given policy and options.
func NewExecutor(policy RetryPolicy, opts ...ExecutorOption) *Executor {
	if policy == nil {
		policy = NewFixedIntervalPolicy(DefaultRetryConfig(), 100*time.Millisecond, 0)
	}

	executor := &Executor{
		policy:          policy,
		errorClassifier: &DefaultErrorClassifier{},
		logger:          logger.GetLogger().Named("retry"),
	}

	// Apply options
	for _, opt := range opts {
		opt(executor)
	}

	return executor
}

// OnRetry sets a callback that is called before each retry attempt.
func (e *Executor) OnRetry(callback func(attempt int, err error, delay time.Duration)) *Executor {
	e.onRetry = callback
	return e
}

// OnSuccess sets a callback that is called when the operation succeeds.
func (e *Executor) OnSuccess(callback func(attempt int, duration time.Duration, result interface{})) *Executor {
	e.onSuccess = callback
	return e
}

// OnFailure sets a callback that is called when all retries are exhausted.
func (e *Executor) OnFailure(callback func(attempts int, duration time.Duration, lastErr error)) *Executor {
	e.onFailure = callback
	return e
}

// Execute executes the given function with retry logic.
func (e *Executor) Execute(ctx context.Context, fn RetryableFunc) (*RetryResult, error) {
	startTime := time.Now()
	var lastErr error
	var result interface{}

	maxAttempts := e.policy.GetMaxAttempts()

	// Log retry execution start
	if e.logger != nil {
		e.logger.Debug("starting retry execution",
			zap.Int("max_attempts", maxAttempts),
		)
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptStart := time.Now()

		// Check context before attempting
		select {
		case <-ctx.Done():
			if e.logger != nil {
				e.logger.Warn("retry execution cancelled by context",
					zap.Int("attempt", attempt-1),
					zap.Error(ctx.Err()),
				)
			}
			if e.metricsCollector != nil {
				e.metricsCollector.RecordRetryAborted("context_cancelled")
			}
			return &RetryResult{
				Success:             false,
				Error:               ctx.Err(),
				Attempts:            attempt - 1,
				TotalDuration:       time.Since(startTime),
				LastAttemptDuration: 0,
			}, ctx.Err()
		default:
		}

		// Check circuit breaker state before executing
		if e.circuitBreaker != nil {
			state := e.circuitBreaker.GetState()
			if state == CircuitStateOpen {
				if e.logger != nil {
					e.logger.Warn("circuit breaker is open, failing fast")
				}
				if e.metricsCollector != nil {
					e.metricsCollector.RecordRetryAborted("circuit_breaker_open")
				}
				return &RetryResult{
					Success:       false,
					Error:         ErrCircuitBreakerOpen,
					Attempts:      attempt - 1,
					TotalDuration: time.Since(startTime),
				}, ErrCircuitBreakerOpen
			}
		}

		// Execute the function
		if e.logger != nil {
			e.logger.Debug("executing attempt",
				zap.Int("attempt", attempt),
			)
		}
		if e.metricsCollector != nil {
			e.metricsCollector.RecordRetryAttempt(attempt)
		}

		result, lastErr = fn(ctx)
		attemptDuration := time.Since(attemptStart)

		// Success!
		if lastErr == nil {
			if e.logger != nil {
				e.logger.Info("retry execution succeeded",
					zap.Int("attempt", attempt),
					zap.Duration("duration", time.Since(startTime)),
				)
			}
			if e.metricsCollector != nil {
				e.metricsCollector.RecordRetrySuccess(attempt, time.Since(startTime))
			}
			if e.onSuccess != nil {
				e.onSuccess(attempt, time.Since(startTime), result)
			}

			return &RetryResult{
				Success:             true,
				Result:              result,
				Error:               nil,
				Attempts:            attempt,
				TotalDuration:       time.Since(startTime),
				LastAttemptDuration: attemptDuration,
			}, nil
		}

		// Check if error is permanent (non-retryable)
		if e.errorClassifier != nil && e.errorClassifier.IsPermanent(lastErr) {
			if e.logger != nil {
				e.logger.Warn("permanent error detected, not retrying",
					zap.Int("attempt", attempt),
					zap.Error(lastErr),
				)
			}
			if e.metricsCollector != nil {
				e.metricsCollector.RecordRetryFailure(attempt, time.Since(startTime), "permanent_error")
			}
			if e.onFailure != nil {
				e.onFailure(attempt, time.Since(startTime), lastErr)
			}
			return &RetryResult{
				Success:             false,
				Error:               lastErr,
				Attempts:            attempt,
				TotalDuration:       time.Since(startTime),
				LastAttemptDuration: attemptDuration,
			}, lastErr
		}

		// Check if we should retry (using policy and circuit breaker)
		shouldRetry := false
		if e.circuitBreaker != nil {
			shouldRetry = e.circuitBreaker.ShouldRetry(lastErr, attempt)
		} else {
			shouldRetry = e.policy.ShouldRetry(lastErr, attempt)
		}

		if !shouldRetry {
			if e.logger != nil {
				e.logger.Warn("retry policy indicates no more retries",
					zap.Int("attempt", attempt),
					zap.Error(lastErr),
				)
			}
			// Not retryable or max attempts reached
			break
		}

		// Calculate retry delay
		delay := e.policy.GetRetryDelay(attempt)

		// Log retry attempt
		if e.logger != nil {
			e.logger.Info("retrying after error",
				zap.Int("attempt", attempt),
				zap.Error(lastErr),
				zap.Duration("delay", delay),
			)
		}

		// Call retry callback
		if e.onRetry != nil {
			e.onRetry(attempt, lastErr, delay)
		}

		// Wait before retrying
		if delay > 0 {
			select {
			case <-ctx.Done():
				if e.logger != nil {
					e.logger.Warn("retry execution cancelled during delay",
						zap.Int("attempt", attempt),
						zap.Error(ctx.Err()),
					)
				}
				if e.metricsCollector != nil {
					e.metricsCollector.RecordRetryAborted("context_cancelled")
				}
				return &RetryResult{
					Success:             false,
					Error:               ctx.Err(),
					Attempts:            attempt,
					TotalDuration:       time.Since(startTime),
					LastAttemptDuration: attemptDuration,
				}, ctx.Err()
			case <-time.After(delay):
				// Continue to next retry
			}
		}
	}

	// All retries exhausted
	totalDuration := time.Since(startTime)

	if e.logger != nil {
		e.logger.Error("all retry attempts exhausted",
			zap.Int("attempts", maxAttempts),
			zap.Duration("total_duration", totalDuration),
			zap.Error(lastErr),
		)
	}

	if e.metricsCollector != nil {
		e.metricsCollector.RecordRetryFailure(maxAttempts, totalDuration, "max_attempts_exceeded")
	}

	if e.onFailure != nil {
		e.onFailure(maxAttempts, totalDuration, lastErr)
	}

	return &RetryResult{
		Success:       false,
		Error:         fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr),
		Attempts:      maxAttempts,
		TotalDuration: totalDuration,
	}, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// ExecuteWithTimeout executes the given function with retry logic and an overall timeout.
func (e *Executor) ExecuteWithTimeout(ctx context.Context, timeout time.Duration, fn RetryableFunc) (*RetryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return e.Execute(ctx, fn)
}

// Do is a convenience method that executes a function with retry and returns only the result and error.
func (e *Executor) Do(ctx context.Context, fn RetryableFunc) (interface{}, error) {
	result, err := e.Execute(ctx, fn)
	if err != nil {
		return nil, err
	}
	return result.Result, nil
}

// DoWithTimeout is a convenience method that executes a function with retry and timeout.
func (e *Executor) DoWithTimeout(ctx context.Context, timeout time.Duration, fn RetryableFunc) (interface{}, error) {
	result, err := e.ExecuteWithTimeout(ctx, timeout, fn)
	if err != nil {
		return nil, err
	}
	return result.Result, nil
}

// ExecuteWithResult is a generic version of Execute that returns typed results.
// This method provides type-safe retry execution for functions that return specific types.
func ExecuteWithResult[T any](ctx context.Context, e *Executor, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	// Wrap the typed function in RetryableFunc
	wrappedFn := func(ctx context.Context) (interface{}, error) {
		result, err := fn(ctx)
		if err != nil {
			return zero, err
		}
		return result, nil
	}

	result, err := e.Execute(ctx, wrappedFn)
	if err != nil {
		return zero, err
	}

	// Type assert the result
	if result.Result == nil {
		return zero, nil
	}

	typedResult, ok := result.Result.(T)
	if !ok {
		return zero, fmt.Errorf("unexpected result type: got %T, want %T", result.Result, zero)
	}

	return typedResult, nil
}

// RetryMetricsCollector defines the interface for collecting retry metrics.
type RetryMetricsCollector struct {
	// Prometheus metrics
	attemptsTotal     *prometheus.CounterVec
	successTotal      *prometheus.CounterVec
	failureTotal      *prometheus.CounterVec
	abortedTotal      *prometheus.CounterVec
	attemptsGauge     prometheus.Gauge
	durationHistogram *prometheus.HistogramVec

	registry *prometheus.Registry
}

// NewRetryMetricsCollector creates a new retry metrics collector.
func NewRetryMetricsCollector(namespace, subsystem string, registry *prometheus.Registry) (*RetryMetricsCollector, error) {
	if namespace == "" {
		namespace = "saga"
	}
	if subsystem == "" {
		subsystem = "retry"
	}
	if registry == nil {
		registry = prometheus.NewRegistry()
	}

	collector := &RetryMetricsCollector{
		registry: registry,
	}

	// Initialize metrics
	collector.attemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "attempts_total",
			Help:      "Total number of retry attempts",
		},
		[]string{"attempt"},
	)

	collector.successTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "success_total",
			Help:      "Total number of successful retries",
		},
		[]string{"attempts"},
	)

	collector.failureTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "failure_total",
			Help:      "Total number of failed retries",
		},
		[]string{"reason"},
	)

	collector.abortedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "aborted_total",
			Help:      "Total number of aborted retries",
		},
		[]string{"reason"},
	)

	collector.attemptsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "attempts_current",
			Help:      "Current number of retry attempts in progress",
		},
	)

	collector.durationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "duration_seconds",
			Help:      "Duration of retry execution in seconds",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0},
		},
		[]string{"status"},
	)

	// Register all metrics
	metrics := []prometheus.Collector{
		collector.attemptsTotal,
		collector.successTotal,
		collector.failureTotal,
		collector.abortedTotal,
		collector.attemptsGauge,
		collector.durationHistogram,
	}

	for _, metric := range metrics {
		if err := registry.Register(metric); err != nil {
			return nil, fmt.Errorf("failed to register metric: %w", err)
		}
	}

	return collector, nil
}

// RecordRetryAttempt records a retry attempt.
func (c *RetryMetricsCollector) RecordRetryAttempt(attempt int) {
	attemptLabel := fmt.Sprintf("%d", attempt)
	if attempt > 5 {
		attemptLabel = "5+"
	}
	c.attemptsTotal.WithLabelValues(attemptLabel).Inc()
	c.attemptsGauge.Inc()
}

// RecordRetrySuccess records a successful retry execution.
func (c *RetryMetricsCollector) RecordRetrySuccess(attempts int, duration time.Duration) {
	attemptsLabel := fmt.Sprintf("%d", attempts)
	if attempts > 5 {
		attemptsLabel = "5+"
	}
	c.successTotal.WithLabelValues(attemptsLabel).Inc()
	c.durationHistogram.WithLabelValues("success").Observe(duration.Seconds())
	c.attemptsGauge.Dec()
}

// RecordRetryFailure records a failed retry execution.
func (c *RetryMetricsCollector) RecordRetryFailure(attempts int, duration time.Duration, reason string) {
	c.failureTotal.WithLabelValues(reason).Inc()
	c.durationHistogram.WithLabelValues("failure").Observe(duration.Seconds())
	c.attemptsGauge.Dec()
}

// RecordRetryAborted records an aborted retry execution.
func (c *RetryMetricsCollector) RecordRetryAborted(reason string) {
	c.abortedTotal.WithLabelValues(reason).Inc()
	c.attemptsGauge.Dec()
}

// GetRegistry returns the Prometheus registry.
func (c *RetryMetricsCollector) GetRegistry() *prometheus.Registry {
	return c.registry
}

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
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// RateLimitResult represents the result of a rate limit check.
type RateLimitResult struct {
	Allowed    bool
	Remaining  int64
	ResetTime  time.Time
	RetryAfter time.Duration
}

// RateLimiter defines the interface for rate limiting operations.
type RateLimiter interface {
	// Allow checks if an operation is allowed under the rate limit.
	Allow(ctx context.Context, key string) (*RateLimitResult, error)

	// Wait blocks until the operation is allowed or the context is cancelled.
	Wait(ctx context.Context, key string) error

	// Reset resets the rate limit for the given key.
	Reset(key string) error

	// GetLimits returns the current rate limit configuration for a key.
	GetLimits(key string) (limit int64, window time.Duration)
}

// TokenBucketLimiter implements a token bucket rate limiter.
type TokenBucketLimiter struct {
	buckets      map[string]*TokenBucket
	defaultLimit int64
	refillPeriod time.Duration
	mutex        sync.RWMutex
}

// TokenBucket represents a token bucket for rate limiting.
type TokenBucket struct {
	capacity   int64
	tokens     int64
	refillRate int64
	lastRefill time.Time
	mutex      sync.RWMutex
}

// NewTokenBucketLimiter creates a new token bucket rate limiter.
func NewTokenBucketLimiter(defaultLimit int64, refillPeriod time.Duration) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		buckets:      make(map[string]*TokenBucket),
		defaultLimit: defaultLimit,
		refillPeriod: refillPeriod,
	}
}

// Allow checks if an operation is allowed under the rate limit.
func (tbl *TokenBucketLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	bucket := tbl.getBucket(key)

	bucket.refill()

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	if bucket.tokens > 0 {
		bucket.tokens--
		return &RateLimitResult{
			Allowed:   true,
			Remaining: bucket.tokens,
			ResetTime: bucket.lastRefill.Add(tbl.refillPeriod),
		}, nil
	}

	// Calculate retry after time
	timeToRefill := bucket.lastRefill.Add(tbl.refillPeriod).Sub(time.Now())
	if timeToRefill < 0 {
		timeToRefill = 0
	}

	return &RateLimitResult{
		Allowed:    false,
		Remaining:  0,
		ResetTime:  bucket.lastRefill.Add(tbl.refillPeriod),
		RetryAfter: timeToRefill,
	}, nil
}

// Wait blocks until the operation is allowed.
func (tbl *TokenBucketLimiter) Wait(ctx context.Context, key string) error {
	for {
		result, err := tbl.Allow(ctx, key)
		if err != nil {
			return err
		}

		if result.Allowed {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(result.RetryAfter):
			// Continue to next iteration
		}
	}
}

// Reset resets the rate limit for the given key.
func (tbl *TokenBucketLimiter) Reset(key string) error {
	tbl.mutex.Lock()
	defer tbl.mutex.Unlock()

	if bucket, exists := tbl.buckets[key]; exists {
		bucket.mutex.Lock()
		bucket.tokens = bucket.capacity
		bucket.lastRefill = time.Now()
		bucket.mutex.Unlock()
	}

	return nil
}

// GetLimits returns the current rate limit configuration.
func (tbl *TokenBucketLimiter) GetLimits(key string) (limit int64, window time.Duration) {
	return tbl.defaultLimit, tbl.refillPeriod
}

// getBucket gets or creates a token bucket for the given key.
func (tbl *TokenBucketLimiter) getBucket(key string) *TokenBucket {
	tbl.mutex.RLock()
	bucket, exists := tbl.buckets[key]
	tbl.mutex.RUnlock()

	if exists {
		return bucket
	}

	tbl.mutex.Lock()
	defer tbl.mutex.Unlock()

	// Double-check after acquiring write lock
	if bucket, exists = tbl.buckets[key]; exists {
		return bucket
	}

	bucket = &TokenBucket{
		capacity:   tbl.defaultLimit,
		tokens:     tbl.defaultLimit,
		refillRate: tbl.defaultLimit,
		lastRefill: time.Now(),
	}

	tbl.buckets[key] = bucket
	return bucket
}

// refill refills the token bucket based on elapsed time.
func (tb *TokenBucket) refill() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	if elapsed <= 0 {
		return
	}

	// Calculate tokens to add based on elapsed time
	tokensToAdd := int64(elapsed.Seconds()) * tb.refillRate / int64(time.Second.Seconds())

	tb.tokens += tokensToAdd
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	tb.lastRefill = now
}

// SlidingWindowLimiter implements a sliding window rate limiter.
type SlidingWindowLimiter struct {
	windows      map[string]*SlidingWindow
	defaultLimit int64
	windowSize   time.Duration
	mutex        sync.RWMutex
}

// SlidingWindow represents a sliding window for rate limiting.
type SlidingWindow struct {
	limit      int64
	windowSize time.Duration
	requests   []time.Time
	mutex      sync.RWMutex
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter.
func NewSlidingWindowLimiter(defaultLimit int64, windowSize time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windows:      make(map[string]*SlidingWindow),
		defaultLimit: defaultLimit,
		windowSize:   windowSize,
	}
}

// Allow checks if an operation is allowed under the rate limit.
func (swl *SlidingWindowLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	window := swl.getWindow(key)

	window.mutex.Lock()
	defer window.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-window.windowSize)

	// Remove expired requests
	validRequests := make([]time.Time, 0, len(window.requests))
	for _, reqTime := range window.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	window.requests = validRequests

	if int64(len(window.requests)) < window.limit {
		window.requests = append(window.requests, now)
		remaining := window.limit - int64(len(window.requests))

		return &RateLimitResult{
			Allowed:   true,
			Remaining: remaining,
			ResetTime: now.Add(window.windowSize),
		}, nil
	}

	// Calculate when the oldest request will expire
	oldestRequest := window.requests[0]
	resetTime := oldestRequest.Add(window.windowSize)
	retryAfter := resetTime.Sub(now)
	if retryAfter < 0 {
		retryAfter = 0
	}

	return &RateLimitResult{
		Allowed:    false,
		Remaining:  0,
		ResetTime:  resetTime,
		RetryAfter: retryAfter,
	}, nil
}

// Wait blocks until the operation is allowed.
func (swl *SlidingWindowLimiter) Wait(ctx context.Context, key string) error {
	for {
		result, err := swl.Allow(ctx, key)
		if err != nil {
			return err
		}

		if result.Allowed {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(result.RetryAfter):
			// Continue to next iteration
		}
	}
}

// Reset resets the rate limit for the given key.
func (swl *SlidingWindowLimiter) Reset(key string) error {
	swl.mutex.Lock()
	defer swl.mutex.Unlock()

	if window, exists := swl.windows[key]; exists {
		window.mutex.Lock()
		window.requests = nil
		window.mutex.Unlock()
	}

	return nil
}

// GetLimits returns the current rate limit configuration.
func (swl *SlidingWindowLimiter) GetLimits(key string) (limit int64, window time.Duration) {
	return swl.defaultLimit, swl.windowSize
}

// getWindow gets or creates a sliding window for the given key.
func (swl *SlidingWindowLimiter) getWindow(key string) *SlidingWindow {
	swl.mutex.RLock()
	window, exists := swl.windows[key]
	swl.mutex.RUnlock()

	if exists {
		return window
	}

	swl.mutex.Lock()
	defer swl.mutex.Unlock()

	// Double-check after acquiring write lock
	if window, exists = swl.windows[key]; exists {
		return window
	}

	window = &SlidingWindow{
		limit:      swl.defaultLimit,
		windowSize: swl.windowSize,
		requests:   make([]time.Time, 0),
	}

	swl.windows[key] = window
	return window
}

// RateLimitStrategy defines the strategy for rate limiting.
type RateLimitStrategy string

const (
	RateLimitStrategyTokenBucket   RateLimitStrategy = "token_bucket"
	RateLimitStrategySlidingWindow RateLimitStrategy = "sliding_window"
)

// RateLimitConfig holds configuration for the rate limiting middleware.
type RateLimitConfig struct {
	Limiter         RateLimiter
	Strategy        RateLimitStrategy
	DefaultLimit    int64
	Window          time.Duration
	KeyExtractor    KeyExtractor
	OnLimitExceeded func(ctx context.Context, message *messaging.Message, result *RateLimitResult) error
	SkipOnError     bool
}

// KeyExtractor defines how to extract rate limiting keys from messages.
type KeyExtractor func(message *messaging.Message) string

// DefaultKeyExtractor extracts the topic as the rate limiting key.
func DefaultKeyExtractor(message *messaging.Message) string {
	return message.Topic
}

// CorrelationIDKeyExtractor extracts the correlation ID as the rate limiting key.
func CorrelationIDKeyExtractor(message *messaging.Message) string {
	if message.CorrelationID != "" {
		return message.CorrelationID
	}
	return message.Topic
}

// PartitionKeyExtractor extracts the partition key as the rate limiting key.
func PartitionKeyExtractor(message *messaging.Message) string {
	if len(message.Key) > 0 {
		return string(message.Key)
	}
	return message.Topic
}

// DefaultRateLimitConfig returns a default rate limiting configuration.
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Strategy:        RateLimitStrategyTokenBucket,
		DefaultLimit:    100,
		Window:          time.Minute,
		KeyExtractor:    DefaultKeyExtractor,
		SkipOnError:     false,
		OnLimitExceeded: nil,
	}
}

// AdaptiveRateLimitMiddleware provides configurable rate limiting for message processing.
type AdaptiveRateLimitMiddleware struct {
	config *RateLimitConfig
}

// NewAdaptiveRateLimitMiddleware creates a new adaptive rate limit middleware.
func NewAdaptiveRateLimitMiddleware(config *RateLimitConfig) *AdaptiveRateLimitMiddleware {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	if config.Limiter == nil {
		switch config.Strategy {
		case RateLimitStrategySlidingWindow:
			config.Limiter = NewSlidingWindowLimiter(config.DefaultLimit, config.Window)
		default:
			config.Limiter = NewTokenBucketLimiter(config.DefaultLimit, config.Window)
		}
	}

	if config.KeyExtractor == nil {
		config.KeyExtractor = DefaultKeyExtractor
	}

	return &AdaptiveRateLimitMiddleware{
		config: config,
	}
}

// Name returns the middleware name.
func (arlm *AdaptiveRateLimitMiddleware) Name() string {
	return "adaptive-rate-limit"
}

// Wrap wraps a handler with rate limiting functionality.
func (arlm *AdaptiveRateLimitMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
	return messaging.MessageHandlerFunc(func(ctx context.Context, message *messaging.Message) error {
		// Extract rate limiting key
		key := arlm.config.KeyExtractor(message)

		// Check rate limit
		result, err := arlm.config.Limiter.Allow(ctx, key)
		if err != nil {
			if arlm.config.SkipOnError {
				return next.Handle(ctx, message)
			}
			return fmt.Errorf("rate limit check failed: %w", err)
		}

		if !result.Allowed {
			if arlm.config.OnLimitExceeded != nil {
				return arlm.config.OnLimitExceeded(ctx, message, result)
			}

			return messaging.NewProcessingError(
				fmt.Sprintf("rate limit exceeded for key %s", key),
				fmt.Errorf("rate limit: %d requests exceeded", arlm.config.DefaultLimit),
			)
		}

		return next.Handle(ctx, message)
	})
}

// GetLimiter returns the underlying rate limiter.
func (arlm *AdaptiveRateLimitMiddleware) GetLimiter() RateLimiter {
	return arlm.config.Limiter
}

// GetStats returns rate limiting statistics.
func (arlm *AdaptiveRateLimitMiddleware) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	limit, window := arlm.config.Limiter.GetLimits("")
	stats["default_limit"] = limit
	stats["window_duration"] = window.String()
	stats["strategy"] = string(arlm.config.Strategy)

	return stats
}

// CircuitBreakerRateLimiter combines rate limiting with circuit breaker patterns.
type CircuitBreakerRateLimiter struct {
	limiter        RateLimiter
	circuitBreaker *CircuitBreaker
	mutex          sync.RWMutex
}

// CircuitBreaker represents a simple circuit breaker.
type CircuitBreaker struct {
	maxFailures     int64
	resetTimeout    time.Duration
	state           CircuitState
	failures        int64
	lastFailureTime time.Time
	mutex           sync.RWMutex
}

// CircuitState represents the state of the circuit breaker.
type CircuitState int

const (
	CircuitStateClosed CircuitState = iota
	CircuitStateOpen
	CircuitStateHalfOpen
)

// NewCircuitBreakerRateLimiter creates a new circuit breaker rate limiter.
func NewCircuitBreakerRateLimiter(limiter RateLimiter, maxFailures int64, resetTimeout time.Duration) *CircuitBreakerRateLimiter {
	return &CircuitBreakerRateLimiter{
		limiter: limiter,
		circuitBreaker: &CircuitBreaker{
			maxFailures:  maxFailures,
			resetTimeout: resetTimeout,
			state:        CircuitStateClosed,
		},
	}
}

// Allow checks if an operation is allowed considering both rate limit and circuit breaker.
func (cbrl *CircuitBreakerRateLimiter) Allow(ctx context.Context, key string) (*RateLimitResult, error) {
	// Check circuit breaker first
	if !cbrl.circuitBreaker.allowRequest() {
		return &RateLimitResult{
			Allowed:    false,
			Remaining:  0,
			RetryAfter: cbrl.circuitBreaker.resetTimeout,
		}, nil
	}

	result, err := cbrl.limiter.Allow(ctx, key)

	if err != nil {
		cbrl.circuitBreaker.recordFailure()
		return result, err
	}

	if result.Allowed {
		cbrl.circuitBreaker.recordSuccess()
	} else {
		cbrl.circuitBreaker.recordFailure()
	}

	return result, nil
}

// Wait blocks until the operation is allowed.
func (cbrl *CircuitBreakerRateLimiter) Wait(ctx context.Context, key string) error {
	return cbrl.limiter.Wait(ctx, key)
}

// Reset resets both the rate limiter and circuit breaker.
func (cbrl *CircuitBreakerRateLimiter) Reset(key string) error {
	cbrl.circuitBreaker.reset()
	return cbrl.limiter.Reset(key)
}

// GetLimits returns the rate limiter limits.
func (cbrl *CircuitBreakerRateLimiter) GetLimits(key string) (limit int64, window time.Duration) {
	return cbrl.limiter.GetLimits(key)
}

// allowRequest checks if the circuit breaker allows the request.
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case CircuitStateClosed:
		return true
	case CircuitStateOpen:
		if now.Sub(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = CircuitStateHalfOpen
			return true
		}
		return false
	case CircuitStateHalfOpen:
		return true
	default:
		return false
	}
}

// recordSuccess records a successful operation.
func (cb *CircuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures = 0
	cb.state = CircuitStateClosed
}

// recordFailure records a failed operation.
func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.state = CircuitStateOpen
	}
}

// reset resets the circuit breaker.
func (cb *CircuitBreaker) reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures = 0
	cb.state = CircuitStateClosed
	cb.lastFailureTime = time.Time{}
}

// CreateRateLimitMiddleware is a factory function to create rate limiting middleware from configuration.
func CreateRateLimitMiddleware(config map[string]interface{}) (messaging.Middleware, error) {
	rateLimitConfig := DefaultRateLimitConfig()

	// Configure strategy
	if strategy, ok := config["strategy"]; ok {
		if strategyStr, ok := strategy.(string); ok {
			switch strategyStr {
			case "token_bucket":
				rateLimitConfig.Strategy = RateLimitStrategyTokenBucket
			case "sliding_window":
				rateLimitConfig.Strategy = RateLimitStrategySlidingWindow
			default:
				rateLimitConfig.Strategy = RateLimitStrategyTokenBucket
			}
		}
	}

	// Configure limit
	if limit, ok := config["limit"]; ok {
		if limitInt64, ok := limit.(int64); ok {
			rateLimitConfig.DefaultLimit = limitInt64
		} else if limitInt, ok := limit.(int); ok {
			rateLimitConfig.DefaultLimit = int64(limitInt)
		}
	}

	// Configure window
	if window, ok := config["window"]; ok {
		if windowStr, ok := window.(string); ok {
			if parsedWindow, err := time.ParseDuration(windowStr); err == nil {
				rateLimitConfig.Window = parsedWindow
			}
		} else if windowDuration, ok := window.(time.Duration); ok {
			rateLimitConfig.Window = windowDuration
		}
	}

	// Configure key extractor
	if keyExtractor, ok := config["key_extractor"]; ok {
		if keyExtractorStr, ok := keyExtractor.(string); ok {
			switch keyExtractorStr {
			case "topic":
				rateLimitConfig.KeyExtractor = DefaultKeyExtractor
			case "correlation_id":
				rateLimitConfig.KeyExtractor = CorrelationIDKeyExtractor
			case "partition_key":
				rateLimitConfig.KeyExtractor = PartitionKeyExtractor
			default:
				rateLimitConfig.KeyExtractor = DefaultKeyExtractor
			}
		}
	}

	return NewAdaptiveRateLimitMiddleware(rateLimitConfig), nil
}

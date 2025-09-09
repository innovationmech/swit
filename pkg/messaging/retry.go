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
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

// RetryPolicy defines configuration for retry operations.
// It integrates with the error classification system to provide appropriate
// retry strategies based on error types and classification.
type RetryPolicy struct {
	// MaxRetries defines the maximum number of retry attempts
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// InitialDelay is the delay before the first retry
	InitialDelay time.Duration `json:"initial_delay" yaml:"initial_delay"`

	// MaxDelay caps the maximum delay between retries
	MaxDelay time.Duration `json:"max_delay" yaml:"max_delay"`

	// BackoffMultiplier controls the rate of delay increase for exponential backoff
	BackoffMultiplier float64 `json:"backoff_multiplier" yaml:"backoff_multiplier"`

	// Strategy determines the backoff strategy to use
	Strategy RetryStrategy `json:"strategy" yaml:"strategy"`

	// JitterEnabled adds randomness to delay calculations to prevent thundering herd
	JitterEnabled bool `json:"jitter_enabled" yaml:"jitter_enabled"`

	// JitterPercent controls the amount of jitter (0-100)
	JitterPercent float64 `json:"jitter_percent" yaml:"jitter_percent"`

	// ErrorClassifier used to determine if errors are retryable
	ErrorClassifier *ErrorClassifier `json:"-" yaml:"-"`

	// RetryableFunc allows custom logic to determine if an error is retryable
	RetryableFunc func(error) bool `json:"-" yaml:"-"`

	// OnRetry callback executed before each retry attempt
	OnRetry func(attempt int, err error, delay time.Duration) `json:"-" yaml:"-"`

	// OnExhausted callback executed when all retries are exhausted
	OnExhausted func(err error, attempts int) `json:"-" yaml:"-"`

	// rng is a per-instance random number generator for thread safety
	rng *rand.Rand `json:"-" yaml:"-"`
}

// NewRetryPolicy creates a new retry policy with sensible defaults.
func NewRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:        3,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		Strategy:          RetryStrategyExponential,
		JitterEnabled:     true,
		JitterPercent:     10.0,
		ErrorClassifier:   NewErrorClassifier(),
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewRetryPolicyFromClassification creates a retry policy from error classification result.
func NewRetryPolicyFromClassification(result ErrorClassificationResult) *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:        result.MaxRetries,
		InitialDelay:      result.InitialDelay,
		MaxDelay:          result.MaxDelay,
		BackoffMultiplier: result.BackoffMultiplier,
		Strategy:          result.RetryStrategy,
		JitterEnabled:     true,
		JitterPercent:     10.0,
		ErrorClassifier:   NewErrorClassifier(),
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// WithMaxRetries sets the maximum number of retries.
func (rp *RetryPolicy) WithMaxRetries(maxRetries int) *RetryPolicy {
	rp.MaxRetries = maxRetries
	return rp
}

// WithInitialDelay sets the initial delay.
func (rp *RetryPolicy) WithInitialDelay(delay time.Duration) *RetryPolicy {
	rp.InitialDelay = delay
	return rp
}

// WithMaxDelay sets the maximum delay.
func (rp *RetryPolicy) WithMaxDelay(delay time.Duration) *RetryPolicy {
	rp.MaxDelay = delay
	return rp
}

// WithBackoffMultiplier sets the backoff multiplier.
func (rp *RetryPolicy) WithBackoffMultiplier(multiplier float64) *RetryPolicy {
	rp.BackoffMultiplier = multiplier
	return rp
}

// WithStrategy sets the retry strategy.
func (rp *RetryPolicy) WithStrategy(strategy RetryStrategy) *RetryPolicy {
	rp.Strategy = strategy
	return rp
}

// WithJitter enables or disables jitter and sets the percentage.
func (rp *RetryPolicy) WithJitter(enabled bool, percent float64) *RetryPolicy {
	rp.JitterEnabled = enabled
	rp.JitterPercent = percent
	return rp
}

// WithErrorClassifier sets a custom error classifier.
func (rp *RetryPolicy) WithErrorClassifier(classifier *ErrorClassifier) *RetryPolicy {
	rp.ErrorClassifier = classifier
	return rp
}

// WithRetryableFunc sets a custom retryable function.
func (rp *RetryPolicy) WithRetryableFunc(fn func(error) bool) *RetryPolicy {
	rp.RetryableFunc = fn
	return rp
}

// WithOnRetry sets the retry callback.
func (rp *RetryPolicy) WithOnRetry(fn func(attempt int, err error, delay time.Duration)) *RetryPolicy {
	rp.OnRetry = fn
	return rp
}

// WithOnExhausted sets the exhausted callback.
func (rp *RetryPolicy) WithOnExhausted(fn func(err error, attempts int)) *RetryPolicy {
	rp.OnExhausted = fn
	return rp
}

// IsRetryable determines if an error should be retried.
func (rp *RetryPolicy) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Context errors should never be retryable regardless of classifier
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Use custom retryable function if provided
	if rp.RetryableFunc != nil {
		return rp.RetryableFunc(err)
	}

	// Use error classifier if available
	if rp.ErrorClassifier != nil {
		result := rp.ErrorClassifier.ClassifyError(err)
		return result.ShouldRetry(0) // Check if any retries are allowed
	}

	// Check if it's a structured messaging error
	var msgErr *BaseMessagingError
	if errors.As(err, &msgErr) {
		return msgErr.Retryable
	}

	// Default to not retryable for unknown errors
	return false
}

// CalculateDelay calculates the delay for a given attempt number.
func (rp *RetryPolicy) CalculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return rp.InitialDelay
	}

	var delay time.Duration

	switch rp.Strategy {
	case RetryStrategyFixed:
		delay = rp.InitialDelay

	case RetryStrategyLinear:
		delay = time.Duration(attempt) * rp.InitialDelay

	case RetryStrategyExponential:
		multiplier := math.Pow(rp.BackoffMultiplier, float64(attempt))
		delay = time.Duration(float64(rp.InitialDelay) * multiplier)

	case RetryStrategyJittered:
		// Exponential backoff with jitter
		multiplier := math.Pow(rp.BackoffMultiplier, float64(attempt))
		baseDelay := time.Duration(float64(rp.InitialDelay) * multiplier)
		delay = rp.addJitter(baseDelay)

	default:
		delay = rp.InitialDelay
	}

	// Apply jitter if enabled and strategy is not already jittered
	if rp.JitterEnabled && rp.Strategy != RetryStrategyJittered {
		delay = rp.addJitter(delay)
	}

	// Apply maximum delay limit
	if rp.MaxDelay > 0 && delay > rp.MaxDelay {
		delay = rp.MaxDelay
	}

	return delay
}

// addJitter adds randomness to the delay calculation.
func (rp *RetryPolicy) addJitter(baseDelay time.Duration) time.Duration {
	if rp.JitterPercent <= 0 {
		return baseDelay
	}

	// Calculate jitter range
	jitterRange := float64(baseDelay) * (rp.JitterPercent / 100.0)

	// Add random jitter between -jitterRange/2 and +jitterRange/2
	// Initialize random generator if not already done (for struct created without constructor)
	if rp.rng == nil {
		rp.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	jitter := (rp.rng.Float64() - 0.5) * jitterRange

	finalDelay := time.Duration(float64(baseDelay) + jitter)

	// Ensure delay is never negative
	if finalDelay < 0 {
		finalDelay = baseDelay / 2
	}

	return finalDelay
}

// Validate validates the retry policy configuration.
func (rp *RetryPolicy) Validate() error {
	if rp.MaxRetries < 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("max_retries cannot be negative").
			Build()
	}

	if rp.InitialDelay < 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("initial_delay cannot be negative").
			Build()
	}

	if rp.MaxDelay < 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("max_delay cannot be negative").
			Build()
	}

	if rp.BackoffMultiplier <= 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("backoff_multiplier must be positive").
			Build()
	}

	if rp.JitterPercent < 0 || rp.JitterPercent > 100 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("jitter_percent must be between 0 and 100").
			Build()
	}

	return nil
}

// RetryExecutor provides a framework for executing operations with retry logic.
type RetryExecutor struct {
	policy *RetryPolicy
	mutex  sync.RWMutex
}

// NewRetryExecutor creates a new retry executor with the given policy.
func NewRetryExecutor(policy *RetryPolicy) (*RetryExecutor, error) {
	if policy == nil {
		return nil, NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
			Message("retry policy cannot be nil").
			Build()
	}

	if err := policy.Validate(); err != nil {
		return nil, NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("invalid retry policy configuration").
			Cause(err).
			Build()
	}

	return &RetryExecutor{
		policy: policy,
	}, nil
}

// Execute executes the given operation with retry logic.
func (re *RetryExecutor) Execute(ctx context.Context, operation func() error) error {
	return re.ExecuteWithResult(ctx, func() (interface{}, error) {
		err := operation()
		return nil, err
	}, nil)
}

// ExecuteWithResult executes an operation that returns a result and error with retry logic.
func (re *RetryExecutor) ExecuteWithResult(ctx context.Context, operation func() (interface{}, error), result *interface{}) error {
	re.mutex.RLock()
	policy := re.policy
	re.mutex.RUnlock()

	var lastErr error
	attempts := 0

	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return NewError(ErrorTypeInternal, ErrCodeOperationAborted).
				Message("operation cancelled by context").
				Cause(ctx.Err()).
				Build()
		default:
		}

		// Execute the operation
		res, err := operation()
		if err == nil {
			// Success
			if result != nil {
				*result = res
			}
			return nil
		}

		lastErr = err
		attempts++

		// Check if we should retry
		if !policy.IsRetryable(err) {
			break
		}

		if attempts > policy.MaxRetries {
			break
		}

		// Calculate delay for next attempt
		delay := policy.CalculateDelay(attempts)

		// Call retry callback if provided
		if policy.OnRetry != nil {
			policy.OnRetry(attempts, err, delay)
		}

		// Wait for the calculated delay
		select {
		case <-ctx.Done():
			return NewError(ErrorTypeInternal, ErrCodeOperationAborted).
				Message("operation cancelled during retry delay").
				Cause(ctx.Err()).
				Details(map[string]interface{}{
					"attempts":   attempts,
					"last_error": lastErr.Error(),
				}).
				Build()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted
	if policy.OnExhausted != nil {
		policy.OnExhausted(lastErr, attempts)
	}

	// Wrap the final error with retry information
	return NewError(ErrorTypeInternal, ErrCodeInternal).
		Message("operation failed after all retry attempts").
		Cause(lastErr).
		Details(map[string]interface{}{
			"max_retries": policy.MaxRetries,
			"attempts":    attempts,
			"final_error": lastErr.Error(),
		}).
		Build()
}

// UpdatePolicy updates the retry policy (thread-safe).
func (re *RetryExecutor) UpdatePolicy(policy *RetryPolicy) error {
	if policy == nil {
		return NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
			Message("retry policy cannot be nil").
			Build()
	}

	if err := policy.Validate(); err != nil {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("invalid retry policy configuration").
			Cause(err).
			Build()
	}

	re.mutex.Lock()
	re.policy = policy
	re.mutex.Unlock()

	return nil
}

// GetPolicy returns a copy of the current policy (thread-safe).
func (re *RetryExecutor) GetPolicy() *RetryPolicy {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	// Return a copy to prevent external modifications
	policyCopy := *re.policy
	return &policyCopy
}

// RetryStatistics tracks retry operation statistics.
type RetryStatistics struct {
	TotalOperations   uint64        `json:"total_operations"`
	SuccessfulOps     uint64        `json:"successful_operations"`
	FailedOps         uint64        `json:"failed_operations"`
	TotalRetries      uint64        `json:"total_retries"`
	AverageRetries    float64       `json:"average_retries"`
	MaxRetries        uint64        `json:"max_retries_used"`
	TotalDelay        time.Duration `json:"total_delay"`
	AverageDelay      time.Duration `json:"average_delay"`
	LastOperationTime time.Time     `json:"last_operation_time"`
}

// retryStatisticsTracker is the internal tracker with mutex.
type retryStatisticsTracker struct {
	stats RetryStatistics
	mutex sync.RWMutex
}

// NewRetryStatistics creates a new retry statistics tracker.
func NewRetryStatistics() *retryStatisticsTracker {
	return &retryStatisticsTracker{}
}

// RecordSuccess records a successful operation.
func (rst *retryStatisticsTracker) RecordSuccess(retries uint64, totalDelay time.Duration) {
	rst.mutex.Lock()
	defer rst.mutex.Unlock()

	rst.stats.TotalOperations++
	rst.stats.SuccessfulOps++
	rst.stats.TotalRetries += retries
	rst.stats.TotalDelay += totalDelay
	rst.stats.LastOperationTime = time.Now()

	if retries > rst.stats.MaxRetries {
		rst.stats.MaxRetries = retries
	}

	rst.updateAverages()
}

// RecordFailure records a failed operation.
func (rst *retryStatisticsTracker) RecordFailure(retries uint64, totalDelay time.Duration) {
	rst.mutex.Lock()
	defer rst.mutex.Unlock()

	rst.stats.TotalOperations++
	rst.stats.FailedOps++
	rst.stats.TotalRetries += retries
	rst.stats.TotalDelay += totalDelay
	rst.stats.LastOperationTime = time.Now()

	if retries > rst.stats.MaxRetries {
		rst.stats.MaxRetries = retries
	}

	rst.updateAverages()
}

// updateAverages recalculates average statistics (must be called with lock held).
func (rst *retryStatisticsTracker) updateAverages() {
	if rst.stats.TotalOperations > 0 {
		rst.stats.AverageRetries = float64(rst.stats.TotalRetries) / float64(rst.stats.TotalOperations)
		rst.stats.AverageDelay = time.Duration(int64(rst.stats.TotalDelay) / int64(rst.stats.TotalOperations))
	}
}

// GetSnapshot returns a snapshot of current statistics.
func (rst *retryStatisticsTracker) GetSnapshot() RetryStatistics {
	rst.mutex.RLock()
	defer rst.mutex.RUnlock()

	return rst.stats
}

// Reset resets all statistics.
func (rst *retryStatisticsTracker) Reset() {
	rst.mutex.Lock()
	defer rst.mutex.Unlock()

	rst.stats = RetryStatistics{}
}

// GetSuccessRate returns the success rate as a percentage.
func (rst *retryStatisticsTracker) GetSuccessRate() float64 {
	rst.mutex.RLock()
	defer rst.mutex.RUnlock()

	if rst.stats.TotalOperations == 0 {
		return 0
	}
	return (float64(rst.stats.SuccessfulOps) / float64(rst.stats.TotalOperations)) * 100
}

// InstrumentedRetryExecutor adds statistics tracking to retry operations.
type InstrumentedRetryExecutor struct {
	*RetryExecutor
	statistics *retryStatisticsTracker
}

// NewInstrumentedRetryExecutor creates a new instrumented retry executor.
func NewInstrumentedRetryExecutor(policy *RetryPolicy) (*InstrumentedRetryExecutor, error) {
	executor, err := NewRetryExecutor(policy)
	if err != nil {
		return nil, err
	}

	return &InstrumentedRetryExecutor{
		RetryExecutor: executor,
		statistics:    NewRetryStatistics(),
	}, nil
}

// Execute executes the operation with statistics tracking.
func (ire *InstrumentedRetryExecutor) Execute(ctx context.Context, operation func() error) error {
	startTime := time.Now()
	attempts := uint64(0)

	// Wrap the operation to count attempts
	wrappedOp := func() error {
		attempts++
		return operation()
	}

	err := ire.RetryExecutor.Execute(ctx, wrappedOp)
	totalDelay := time.Since(startTime)

	// Record statistics
	if err == nil {
		ire.statistics.RecordSuccess(attempts-1, totalDelay) // attempts-1 = retries
	} else {
		ire.statistics.RecordFailure(attempts-1, totalDelay)
	}

	return err
}

// GetStatistics returns the current statistics.
func (ire *InstrumentedRetryExecutor) GetStatistics() RetryStatistics {
	return ire.statistics.GetSnapshot()
}

// ResetStatistics resets the statistics.
func (ire *InstrumentedRetryExecutor) ResetStatistics() {
	ire.statistics.Reset()
}

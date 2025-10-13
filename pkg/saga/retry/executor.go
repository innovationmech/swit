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

// Executor executes operations with retry logic based on a retry policy.
type Executor struct {
	// policy is the retry policy to use
	policy RetryPolicy

	// onRetry is called before each retry attempt (optional)
	onRetry func(attempt int, err error, delay time.Duration)

	// onSuccess is called when the operation succeeds (optional)
	onSuccess func(attempt int, duration time.Duration, result interface{})

	// onFailure is called when all retries are exhausted (optional)
	onFailure func(attempts int, duration time.Duration, lastErr error)
}

// NewExecutor creates a new retry executor with the given policy.
func NewExecutor(policy RetryPolicy) *Executor {
	if policy == nil {
		policy = NewFixedIntervalPolicy(DefaultRetryConfig(), 100*time.Millisecond, 0)
	}

	return &Executor{
		policy: policy,
	}
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

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptStart := time.Now()

		// Check context before attempting
		select {
		case <-ctx.Done():
			return &RetryResult{
				Success:             false,
				Error:               ctx.Err(),
				Attempts:            attempt - 1,
				TotalDuration:       time.Since(startTime),
				LastAttemptDuration: 0,
			}, ctx.Err()
		default:
		}

		// Execute the function
		result, lastErr = fn(ctx)
		attemptDuration := time.Since(attemptStart)

		// Success!
		if lastErr == nil {
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

		// Check if we should retry
		if !e.policy.ShouldRetry(lastErr, attempt) {
			// Not retryable or max attempts reached
			break
		}

		// Calculate retry delay
		delay := e.policy.GetRetryDelay(attempt)

		// Call retry callback
		if e.onRetry != nil {
			e.onRetry(attempt, lastErr, delay)
		}

		// Wait before retrying
		if delay > 0 {
			select {
			case <-ctx.Done():
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

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

// Package retry provides flexible retry strategies and backoff algorithms
// for reliable error recovery in Saga distributed transactions.
package retry

import (
	"errors"
	"time"
)

// Common errors returned by retry policies.
var (
	// ErrMaxRetriesExceeded is returned when the maximum number of retry attempts is exceeded.
	ErrMaxRetriesExceeded = errors.New("maximum retry attempts exceeded")

	// ErrRetryAborted is returned when retry is explicitly aborted.
	ErrRetryAborted = errors.New("retry aborted")

	// ErrInvalidConfig is returned when the retry configuration is invalid.
	ErrInvalidConfig = errors.New("invalid retry configuration")

	// ErrCircuitBreakerOpen is returned when the circuit breaker is open.
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
)

// RetryConfig defines the configuration for retry policies.
// It provides common settings that can be shared across different retry strategies.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts (including the initial attempt).
	// Must be >= 1. A value of 1 means no retries.
	MaxAttempts int

	// InitialDelay is the initial delay before the first retry.
	// Must be >= 0. A value of 0 means no delay.
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries.
	// Must be >= InitialDelay. A value of 0 means no maximum.
	MaxDelay time.Duration

	// Timeout is the overall timeout for all retry attempts.
	// A value of 0 means no timeout.
	Timeout time.Duration

	// RetryableErrors is a list of errors that should trigger a retry.
	// If nil or empty, all errors are considered retryable.
	RetryableErrors []error

	// NonRetryableErrors is a list of errors that should not trigger a retry.
	// Takes precedence over RetryableErrors.
	NonRetryableErrors []error
}

// Validate validates the retry configuration.
func (c *RetryConfig) Validate() error {
	if c.MaxAttempts < 1 {
		return ErrInvalidConfig
	}
	if c.InitialDelay < 0 {
		return ErrInvalidConfig
	}
	if c.MaxDelay > 0 && c.MaxDelay < c.InitialDelay {
		return ErrInvalidConfig
	}
	return nil
}

// DefaultRetryConfig returns a default retry configuration.
// - MaxAttempts: 3 (1 initial attempt + 2 retries)
// - InitialDelay: 100ms
// - MaxDelay: 30s
// - Timeout: 5m
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Timeout:      5 * time.Minute,
	}
}

// NoRetryConfig returns a retry configuration that disables retries.
func NoRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  1,
		InitialDelay: 0,
		MaxDelay:     0,
		Timeout:      0,
	}
}

// IsRetryableError checks if an error should trigger a retry.
// It checks against the configured retryable and non-retryable error lists.
func (c *RetryConfig) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check non-retryable errors first (takes precedence)
	for _, nonRetryable := range c.NonRetryableErrors {
		if errors.Is(err, nonRetryable) {
			return false
		}
	}

	// If no retryable errors specified, all errors are retryable
	if len(c.RetryableErrors) == 0 {
		return true
	}

	// Check if error is in retryable list
	for _, retryable := range c.RetryableErrors {
		if errors.Is(err, retryable) {
			return true
		}
	}

	return false
}

// RetryResult represents the result of a retry execution.
type RetryResult struct {
	// Success indicates if the operation succeeded.
	Success bool

	// Result is the result of the operation (nil if failed).
	Result interface{}

	// Error is the error that occurred (nil if successful).
	Error error

	// Attempts is the total number of attempts made.
	Attempts int

	// TotalDuration is the total time spent on all attempts including delays.
	TotalDuration time.Duration

	// LastAttemptDuration is the duration of the last attempt.
	LastAttemptDuration time.Duration
}

// RetryContext provides context information for retry decisions.
type RetryContext struct {
	// Attempt is the current attempt number (1-indexed).
	Attempt int

	// Error is the error from the last attempt.
	Error error

	// StartTime is when the first attempt started.
	StartTime time.Time

	// LastAttemptTime is when the last attempt started.
	LastAttemptTime time.Time

	// ElapsedTime is the total time elapsed since the first attempt.
	ElapsedTime time.Duration

	// Metadata provides additional context for retry decisions.
	Metadata map[string]interface{}
}


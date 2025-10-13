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

package retry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/saga/retry"
)

// ExampleExponentialBackoffPolicy demonstrates using exponential backoff retry.
func ExampleExponentialBackoffPolicy() {
	// Create retry configuration
	config := &retry.RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
	}

	// Create exponential backoff policy with 2x multiplier and 10% jitter
	policy := retry.NewExponentialBackoffPolicy(config, 2.0, 0.1)

	// Create executor
	executor := retry.NewExecutor(policy)

	// Execute with retry
	ctx := context.Background()
	result, err := executor.Do(ctx, func(ctx context.Context) (interface{}, error) {
		// Your business logic here
		return "success", nil
	})

	if err != nil {
		fmt.Printf("Failed: %v\n", err)
		return
	}

	fmt.Printf("Success: %v\n", result)
	// Output: Success: success
}

// ExampleLinearBackoffPolicy demonstrates using linear backoff retry.
func ExampleLinearBackoffPolicy() {
	config := &retry.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	// Linear backoff with 200ms increment
	policy := retry.NewLinearBackoffPolicy(config, 200*time.Millisecond, 0)

	executor := retry.NewExecutor(policy)

	ctx := context.Background()
	result, err := executor.Do(ctx, func(ctx context.Context) (interface{}, error) {
		return "done", nil
	})

	if err != nil {
		fmt.Printf("Failed: %v\n", err)
		return
	}

	fmt.Printf("Result: %v\n", result)
	// Output: Result: done
}

// ExampleFixedIntervalPolicy demonstrates using fixed interval retry.
func ExampleFixedIntervalPolicy() {
	config := &retry.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 500 * time.Millisecond,
	}

	// Fixed 500ms interval between retries
	policy := retry.NewFixedIntervalPolicy(config, 500*time.Millisecond, 0)

	executor := retry.NewExecutor(policy)

	ctx := context.Background()
	result, err := executor.Do(ctx, func(ctx context.Context) (interface{}, error) {
		return "completed", nil
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %v\n", result)
	// Output: Result: completed
}

// ExampleCircuitBreaker demonstrates using circuit breaker with retry.
func ExampleCircuitBreaker() {
	// Circuit breaker configuration
	cbConfig := &retry.CircuitBreakerConfig{
		MaxFailures:         3,
		ResetTimeout:        5 * time.Second,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	// Base retry policy
	retryConfig := &retry.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
	}
	basePolicy := retry.NewFixedIntervalPolicy(retryConfig, 1*time.Second, 0)

	// Wrap with circuit breaker
	cb := retry.NewCircuitBreaker(cbConfig, basePolicy)

	executor := retry.NewExecutor(cb)

	ctx := context.Background()
	result, err := executor.Do(ctx, func(ctx context.Context) (interface{}, error) {
		// Call external service
		return "response", nil
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Circuit state: %v\n", cb.GetState())
	fmt.Printf("Result: %v\n", result)
	// Output:
	// Circuit state: closed
	// Result: response
}

// ExampleExecutor_OnRetry demonstrates using retry callbacks.
func ExampleExecutor_OnRetry() {
	config := &retry.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := retry.NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)

	attemptCount := 0

	executor := retry.NewExecutor(policy).
		OnRetry(func(attempt int, err error, delay time.Duration) {
			fmt.Printf("Retrying attempt %d after error: %v\n", attempt+1, err)
		}).
		OnSuccess(func(attempt int, duration time.Duration, result interface{}) {
			fmt.Printf("Succeeded on attempt %d\n", attempt)
		})

	ctx := context.Background()
	_, err := executor.Do(ctx, func(ctx context.Context) (interface{}, error) {
		attemptCount++
		if attemptCount < 2 {
			return nil, errors.New("temporary failure")
		}
		return "success", nil
	})

	if err != nil {
		fmt.Printf("Failed: %v\n", err)
	}

	// Output:
	// Retrying attempt 2 after error: temporary failure
	// Succeeded on attempt 2
}

// ExampleExecutor_ExecuteWithTimeout demonstrates using timeout with retry.
func ExampleExecutor_ExecuteWithTimeout() {
	config := &retry.RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
	}

	policy := retry.NewFixedIntervalPolicy(config, 100*time.Millisecond, 0)
	executor := retry.NewExecutor(policy)

	ctx := context.Background()

	// This will timeout before all retries complete
	result, err := executor.ExecuteWithTimeout(ctx, 500*time.Millisecond, func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("failure")
	})

	if err != nil {
		fmt.Printf("Operation failed after %d attempts\n", result.Attempts)
	}

	// Output: Operation failed after 5 attempts
}

// ExampleRetryConfig_IsRetryableError demonstrates error classification.
func ExampleRetryConfig_IsRetryableError() {
	errNetwork := errors.New("network timeout")
	errValidation := errors.New("invalid input")

	config := &retry.RetryConfig{
		MaxAttempts: 3,
		RetryableErrors: []error{
			errNetwork,
		},
		NonRetryableErrors: []error{
			errValidation,
		},
	}

	fmt.Printf("Network error retryable: %v\n", config.IsRetryableError(errNetwork))
	fmt.Printf("Validation error retryable: %v\n", config.IsRetryableError(errValidation))

	// Output:
	// Network error retryable: true
	// Validation error retryable: false
}

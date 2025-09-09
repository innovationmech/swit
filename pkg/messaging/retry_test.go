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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryPolicy_Validation(t *testing.T) {
	tests := []struct {
		name        string
		policy      *RetryPolicy
		expectError bool
	}{
		{
			name: "valid policy",
			policy: &RetryPolicy{
				MaxRetries:        3,
				InitialDelay:      100 * time.Millisecond,
				MaxDelay:          5 * time.Second,
				BackoffMultiplier: 2.0,
				JitterPercent:     10.0,
			},
			expectError: false,
		},
		{
			name: "negative max retries",
			policy: &RetryPolicy{
				MaxRetries:        -1,
				InitialDelay:      100 * time.Millisecond,
				MaxDelay:          5 * time.Second,
				BackoffMultiplier: 2.0,
				JitterPercent:     10.0,
			},
			expectError: true,
		},
		{
			name: "negative initial delay",
			policy: &RetryPolicy{
				MaxRetries:        3,
				InitialDelay:      -100 * time.Millisecond,
				MaxDelay:          5 * time.Second,
				BackoffMultiplier: 2.0,
				JitterPercent:     10.0,
			},
			expectError: true,
		},
		{
			name: "negative max delay",
			policy: &RetryPolicy{
				MaxRetries:        3,
				InitialDelay:      100 * time.Millisecond,
				MaxDelay:          -5 * time.Second,
				BackoffMultiplier: 2.0,
				JitterPercent:     10.0,
			},
			expectError: true,
		},
		{
			name: "zero backoff multiplier",
			policy: &RetryPolicy{
				MaxRetries:        3,
				InitialDelay:      100 * time.Millisecond,
				MaxDelay:          5 * time.Second,
				BackoffMultiplier: 0,
				JitterPercent:     10.0,
			},
			expectError: true,
		},
		{
			name: "invalid jitter percent",
			policy: &RetryPolicy{
				MaxRetries:        3,
				InitialDelay:      100 * time.Millisecond,
				MaxDelay:          5 * time.Second,
				BackoffMultiplier: 2.0,
				JitterPercent:     -10.0,
			},
			expectError: true,
		},
		{
			name: "jitter percent too high",
			policy: &RetryPolicy{
				MaxRetries:        3,
				InitialDelay:      100 * time.Millisecond,
				MaxDelay:          5 * time.Second,
				BackoffMultiplier: 2.0,
				JitterPercent:     150.0,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRetryPolicy_DelayCalculation(t *testing.T) {
	tests := []struct {
		name          string
		policy        *RetryPolicy
		attempt       int
		expectedDelay time.Duration
		allowVariance bool
	}{
		{
			name: "fixed strategy",
			policy: &RetryPolicy{
				InitialDelay:  100 * time.Millisecond,
				Strategy:      RetryStrategyFixed,
				JitterEnabled: false,
			},
			attempt:       2,
			expectedDelay: 100 * time.Millisecond,
		},
		{
			name: "linear strategy",
			policy: &RetryPolicy{
				InitialDelay:  100 * time.Millisecond,
				Strategy:      RetryStrategyLinear,
				JitterEnabled: false,
			},
			attempt:       2,
			expectedDelay: 200 * time.Millisecond,
		},
		{
			name: "exponential strategy",
			policy: &RetryPolicy{
				InitialDelay:      100 * time.Millisecond,
				BackoffMultiplier: 2.0,
				Strategy:          RetryStrategyExponential,
				JitterEnabled:     false,
			},
			attempt:       2,
			expectedDelay: 400 * time.Millisecond, // 100 * 2^2
		},
		{
			name: "exponential with max delay",
			policy: &RetryPolicy{
				InitialDelay:      100 * time.Millisecond,
				MaxDelay:          300 * time.Millisecond,
				BackoffMultiplier: 2.0,
				Strategy:          RetryStrategyExponential,
				JitterEnabled:     false,
			},
			attempt:       3,
			expectedDelay: 300 * time.Millisecond, // capped at max delay
		},
		{
			name: "jittered strategy",
			policy: &RetryPolicy{
				InitialDelay:      100 * time.Millisecond,
				BackoffMultiplier: 2.0,
				Strategy:          RetryStrategyJittered,
				JitterPercent:     25.0,
			},
			attempt:       1,
			allowVariance: true, // Jitter makes delay variable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := tt.policy.CalculateDelay(tt.attempt)

			if tt.allowVariance {
				// For jittered strategies, check that delay is within reasonable bounds
				baseDelay := time.Duration(float64(tt.policy.InitialDelay) * 2.0) // attempt 1 with multiplier 2.0
				minDelay := time.Duration(float64(baseDelay) * 0.75)              // 25% jitter down
				maxDelay := time.Duration(float64(baseDelay) * 1.25)              // 25% jitter up
				assert.True(t, delay >= minDelay && delay <= maxDelay,
					"Delay %v should be between %v and %v", delay, minDelay, maxDelay)
			} else {
				assert.Equal(t, tt.expectedDelay, delay)
			}
		})
	}
}

func TestRetryPolicy_IsRetryable(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name      string
		policy    *RetryPolicy
		err       error
		retryable bool
	}{
		{
			name: "nil error",
			policy: &RetryPolicy{
				ErrorClassifier: classifier,
			},
			err:       nil,
			retryable: false,
		},
		{
			name: "retryable messaging error",
			policy: &RetryPolicy{
				ErrorClassifier: classifier,
			},
			err: NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
				Message("connection timeout").
				Retryable(true).
				Build(),
			retryable: true,
		},
		{
			name: "non-retryable messaging error",
			policy: &RetryPolicy{
				ErrorClassifier: classifier,
			},
			err: NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
				Message("invalid configuration").
				Retryable(false).
				Build(),
			retryable: false,
		},
		{
			name: "context canceled",
			policy: &RetryPolicy{
				ErrorClassifier: classifier,
			},
			err:       context.Canceled,
			retryable: false,
		},
		{
			name: "context deadline exceeded",
			policy: &RetryPolicy{
				ErrorClassifier: classifier,
			},
			err:       context.DeadlineExceeded,
			retryable: false,
		},
		{
			name: "custom retryable function - true",
			policy: &RetryPolicy{
				RetryableFunc: func(err error) bool {
					return err.Error() == "retryable error"
				},
			},
			err:       errors.New("retryable error"),
			retryable: true,
		},
		{
			name: "custom retryable function - false",
			policy: &RetryPolicy{
				RetryableFunc: func(err error) bool {
					return err.Error() == "retryable error"
				},
			},
			err:       errors.New("non-retryable error"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryable := tt.policy.IsRetryable(tt.err)
			assert.Equal(t, tt.retryable, retryable)
		})
	}
}

func TestRetryExecutor_Execute(t *testing.T) {
	t.Run("successful operation", func(t *testing.T) {
		policy := NewRetryPolicy().WithMaxRetries(3)
		executor, err := NewRetryExecutor(policy)
		require.NoError(t, err)

		callCount := 0
		operation := func() error {
			callCount++
			return nil
		}

		ctx := context.Background()
		err = executor.Execute(ctx, operation)

		assert.NoError(t, err)
		assert.Equal(t, 1, callCount) // Should only be called once
	})

	t.Run("retryable error with eventual success", func(t *testing.T) {
		policy := NewRetryPolicy().
			WithMaxRetries(3).
			WithInitialDelay(10 * time.Millisecond).
			WithStrategy(RetryStrategyFixed)

		executor, err := NewRetryExecutor(policy)
		require.NoError(t, err)

		callCount := 0
		operation := func() error {
			callCount++
			if callCount < 3 {
				return NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
					Message("connection timeout").
					Retryable(true).
					Build()
			}
			return nil
		}

		ctx := context.Background()
		start := time.Now()
		err = executor.Execute(ctx, operation)
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.Equal(t, 3, callCount)
		// Should have delayed at least 2 times (for 2 retries)
		assert.True(t, elapsed >= 20*time.Millisecond)
	})

	t.Run("non-retryable error", func(t *testing.T) {
		policy := NewRetryPolicy().WithMaxRetries(3)
		executor, err := NewRetryExecutor(policy)
		require.NoError(t, err)

		callCount := 0
		operation := func() error {
			callCount++
			return NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
				Message("invalid configuration").
				Retryable(false).
				Build()
		}

		ctx := context.Background()
		err = executor.Execute(ctx, operation)

		assert.Error(t, err)
		assert.Equal(t, 1, callCount) // Should only be called once
	})

	t.Run("retry exhaustion", func(t *testing.T) {
		policy := NewRetryPolicy().
			WithMaxRetries(2).
			WithInitialDelay(1 * time.Millisecond)

		executor, err := NewRetryExecutor(policy)
		require.NoError(t, err)

		callCount := 0
		operation := func() error {
			callCount++
			return NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
				Message("connection timeout").
				Retryable(true).
				Build()
		}

		ctx := context.Background()
		err = executor.Execute(ctx, operation)

		assert.Error(t, err)
		assert.Equal(t, 3, callCount) // Initial call + 2 retries

		// Should be our wrapped error
		var msgErr *BaseMessagingError
		assert.True(t, errors.As(err, &msgErr))
		assert.Equal(t, ErrorTypeInternal, msgErr.Type)
	})

	t.Run("context cancellation", func(t *testing.T) {
		policy := NewRetryPolicy().
			WithMaxRetries(5).
			WithInitialDelay(100 * time.Millisecond)

		executor, err := NewRetryExecutor(policy)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		operation := func() error {
			return NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
				Message("connection timeout").
				Retryable(true).
				Build()
		}

		err = executor.Execute(ctx, operation)

		assert.Error(t, err)
		// Should be a context cancellation error
		var msgErr *BaseMessagingError
		assert.True(t, errors.As(err, &msgErr))
		assert.Equal(t, ErrorTypeInternal, msgErr.Type)
	})
}

func TestRetryExecutor_WithCallbacks(t *testing.T) {
	retryCount := 0
	exhaustedCalled := false

	policy := NewRetryPolicy().
		WithMaxRetries(2).
		WithInitialDelay(1 * time.Millisecond).
		WithOnRetry(func(attempt int, err error, delay time.Duration) {
			retryCount++
		}).
		WithOnExhausted(func(err error, attempts int) {
			exhaustedCalled = true
		})

	executor, err := NewRetryExecutor(policy)
	require.NoError(t, err)

	operation := func() error {
		return NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
			Message("connection timeout").
			Retryable(true).
			Build()
	}

	ctx := context.Background()
	err = executor.Execute(ctx, operation)

	assert.Error(t, err)
	assert.Equal(t, 2, retryCount)
	assert.True(t, exhaustedCalled)
}

func TestInstrumentedRetryExecutor(t *testing.T) {
	policy := NewRetryPolicy().
		WithMaxRetries(3).
		WithInitialDelay(1 * time.Millisecond)

	executor, err := NewInstrumentedRetryExecutor(policy)
	require.NoError(t, err)

	// Test successful operation
	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
				Message("connection timeout").
				Retryable(true).
				Build()
		}
		return nil
	}

	ctx := context.Background()
	err = executor.Execute(ctx, operation)
	assert.NoError(t, err)

	stats := executor.GetStatistics()
	assert.Equal(t, uint64(1), stats.TotalOperations)
	assert.Equal(t, uint64(1), stats.SuccessfulOps)
	assert.Equal(t, uint64(0), stats.FailedOps)
	assert.Equal(t, uint64(2), stats.TotalRetries) // 2 retries before success
	assert.Equal(t, 100.0, executor.statistics.GetSuccessRate())

	// Test failed operation
	callCount = 0
	failedOperation := func() error {
		callCount++
		return NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
			Message("connection timeout").
			Retryable(true).
			Build()
	}

	err = executor.Execute(ctx, failedOperation)
	assert.Error(t, err)

	stats = executor.GetStatistics()
	assert.Equal(t, uint64(2), stats.TotalOperations)
	assert.Equal(t, uint64(1), stats.SuccessfulOps)
	assert.Equal(t, uint64(1), stats.FailedOps)
	assert.Equal(t, 50.0, executor.statistics.GetSuccessRate())
}

func TestRetryPolicyBuilderMethods(t *testing.T) {
	policy := NewRetryPolicy().
		WithMaxRetries(5).
		WithInitialDelay(200*time.Millisecond).
		WithMaxDelay(10*time.Second).
		WithBackoffMultiplier(1.5).
		WithStrategy(RetryStrategyLinear).
		WithJitter(true, 20.0)

	assert.Equal(t, 5, policy.MaxRetries)
	assert.Equal(t, 200*time.Millisecond, policy.InitialDelay)
	assert.Equal(t, 10*time.Second, policy.MaxDelay)
	assert.Equal(t, 1.5, policy.BackoffMultiplier)
	assert.Equal(t, RetryStrategyLinear, policy.Strategy)
	assert.True(t, policy.JitterEnabled)
	assert.Equal(t, 20.0, policy.JitterPercent)
}

func TestNewRetryPolicyFromClassification(t *testing.T) {
	classification := ErrorClassificationResult{
		Classification:    ClassificationTransient,
		RetryStrategy:     RetryStrategyExponential,
		MaxRetries:        4,
		InitialDelay:      500 * time.Millisecond,
		MaxDelay:          15 * time.Second,
		BackoffMultiplier: 3.0,
	}

	policy := NewRetryPolicyFromClassification(classification)

	assert.Equal(t, 4, policy.MaxRetries)
	assert.Equal(t, 500*time.Millisecond, policy.InitialDelay)
	assert.Equal(t, 15*time.Second, policy.MaxDelay)
	assert.Equal(t, 3.0, policy.BackoffMultiplier)
	assert.Equal(t, RetryStrategyExponential, policy.Strategy)
	assert.True(t, policy.JitterEnabled)
}

func TestRetryStatistics(t *testing.T) {
	stats := NewRetryStatistics()

	// Record some operations
	stats.RecordSuccess(2, 100*time.Millisecond)
	stats.RecordSuccess(1, 50*time.Millisecond)
	stats.RecordFailure(3, 200*time.Millisecond)

	snapshot := stats.GetSnapshot()
	assert.Equal(t, uint64(3), snapshot.TotalOperations)
	assert.Equal(t, uint64(2), snapshot.SuccessfulOps)
	assert.Equal(t, uint64(1), snapshot.FailedOps)
	assert.Equal(t, uint64(6), snapshot.TotalRetries) // 2+1+3
	assert.Equal(t, uint64(3), snapshot.MaxRetries)
	assert.Equal(t, 350*time.Millisecond, snapshot.TotalDelay)

	// Check success rate
	successRate := stats.GetSuccessRate()
	assert.InDelta(t, 66.67, successRate, 0.01) // 2/3 * 100

	// Reset and verify
	stats.Reset()
	snapshot = stats.GetSnapshot()
	assert.Equal(t, uint64(0), snapshot.TotalOperations)
	assert.Equal(t, 0.0, stats.GetSuccessRate())
}

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
	"errors"
	"testing"
	"time"
)

func TestExecutor_SuccessOnFirstAttempt(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)
	executor := NewExecutor(policy)

	// Function that succeeds immediately
	fn := func(ctx context.Context) (interface{}, error) {
		return "success", nil
	}

	result, err := executor.Execute(context.Background(), fn)

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if !result.Success {
		t.Error("expected Success = true")
	}
	if result.Result != "success" {
		t.Errorf("Result = %v, want 'success'", result.Result)
	}
	if result.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", result.Attempts)
	}
}

func TestExecutor_SuccessAfterRetries(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)
	executor := NewExecutor(policy)

	attemptCount := 0
	testErr := errors.New("temporary error")

	// Function that fails twice then succeeds
	fn := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		if attemptCount < 3 {
			return nil, testErr
		}
		return "success", nil
	}

	result, err := executor.Execute(context.Background(), fn)

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if !result.Success {
		t.Error("expected Success = true")
	}
	if result.Attempts != 3 {
		t.Errorf("Attempts = %d, want 3", result.Attempts)
	}
}

func TestExecutor_MaxRetriesExceeded(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)
	executor := NewExecutor(policy)

	testErr := errors.New("persistent error")

	// Function that always fails
	fn := func(ctx context.Context) (interface{}, error) {
		return nil, testErr
	}

	result, err := executor.Execute(context.Background(), fn)

	if err == nil {
		t.Error("Execute() error = nil, want error")
	}
	if !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Errorf("Execute() error = %v, want ErrMaxRetriesExceeded", err)
	}
	if result.Success {
		t.Error("expected Success = false")
	}
	if result.Attempts != 3 {
		t.Errorf("Attempts = %d, want 3", result.Attempts)
	}
}

func TestExecutor_ContextCancellation(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 50 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 50*time.Millisecond, 0)
	executor := NewExecutor(policy)

	ctx, cancel := context.WithCancel(context.Background())
	testErr := errors.New("error")

	attemptCount := 0

	// Function that fails
	fn := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		if attemptCount == 2 {
			cancel() // Cancel after second attempt
		}
		return nil, testErr
	}

	result, err := executor.Execute(ctx, fn)

	if err == nil {
		t.Error("Execute() error = nil, want context error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Execute() error = %v, want context.Canceled", err)
	}
	if result.Success {
		t.Error("expected Success = false")
	}
}

func TestExecutor_OnRetryCallback(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)

	retryCallbacks := 0
	executor := NewExecutor(policy).OnRetry(func(attempt int, err error, delay time.Duration) {
		retryCallbacks++
	})

	testErr := errors.New("error")

	// Function that always fails
	fn := func(ctx context.Context) (interface{}, error) {
		return nil, testErr
	}

	_, _ = executor.Execute(context.Background(), fn)

	// Should have 2 retry callbacks (before 2nd and 3rd attempts)
	if retryCallbacks != 2 {
		t.Errorf("retry callbacks = %d, want 2", retryCallbacks)
	}
}

func TestExecutor_OnSuccessCallback(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)

	successCalled := false
	var successAttempt int

	executor := NewExecutor(policy).OnSuccess(func(attempt int, duration time.Duration, result interface{}) {
		successCalled = true
		successAttempt = attempt
	})

	attemptCount := 0
	testErr := errors.New("error")

	// Function that succeeds on 2nd attempt
	fn := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		if attemptCount < 2 {
			return nil, testErr
		}
		return "success", nil
	}

	_, _ = executor.Execute(context.Background(), fn)

	if !successCalled {
		t.Error("expected OnSuccess callback to be called")
	}
	if successAttempt != 2 {
		t.Errorf("success attempt = %d, want 2", successAttempt)
	}
}

func TestExecutor_OnFailureCallback(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)

	failureCalled := false
	var failureAttempts int

	executor := NewExecutor(policy).OnFailure(func(attempts int, duration time.Duration, lastErr error) {
		failureCalled = true
		failureAttempts = attempts
	})

	testErr := errors.New("error")

	// Function that always fails
	fn := func(ctx context.Context) (interface{}, error) {
		return nil, testErr
	}

	_, _ = executor.Execute(context.Background(), fn)

	if !failureCalled {
		t.Error("expected OnFailure callback to be called")
	}
	if failureAttempts != 3 {
		t.Errorf("failure attempts = %d, want 3", failureAttempts)
	}
}

func TestExecutor_ExecuteWithTimeout(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 50 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 50*time.Millisecond, 0)
	executor := NewExecutor(policy)

	testErr := errors.New("error")

	// Function that always fails
	fn := func(ctx context.Context) (interface{}, error) {
		return nil, testErr
	}

	// Timeout should kick in before all retries complete
	timeout := 100 * time.Millisecond
	result, err := executor.ExecuteWithTimeout(context.Background(), timeout, fn)

	if err == nil {
		t.Error("ExecuteWithTimeout() error = nil, want timeout error")
	}
	if result.Success {
		t.Error("expected Success = false")
	}
	// Should have attempted fewer times due to timeout
	if result.Attempts >= 5 {
		t.Errorf("Attempts = %d, want < 5 due to timeout", result.Attempts)
	}
}

func TestExecutor_Do(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)
	executor := NewExecutor(policy)

	// Test successful execution
	fn := func(ctx context.Context) (interface{}, error) {
		return "result", nil
	}

	result, err := executor.Do(context.Background(), fn)

	if err != nil {
		t.Errorf("Do() error = %v, want nil", err)
	}
	if result != "result" {
		t.Errorf("Do() result = %v, want 'result'", result)
	}
}

func TestExecutor_DoWithTimeout(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)
	executor := NewExecutor(policy)

	// Test successful execution
	fn := func(ctx context.Context) (interface{}, error) {
		return "result", nil
	}

	result, err := executor.DoWithTimeout(context.Background(), 1*time.Second, fn)

	if err != nil {
		t.Errorf("DoWithTimeout() error = %v, want nil", err)
	}
	if result != "result" {
		t.Errorf("DoWithTimeout() result = %v, want 'result'", result)
	}
}

func TestNewExecutor_NilPolicy(t *testing.T) {
	executor := NewExecutor(nil)

	if executor.policy == nil {
		t.Error("expected default policy, got nil")
	}

	// Should work with default policy
	fn := func(ctx context.Context) (interface{}, error) {
		return "ok", nil
	}

	result, err := executor.Execute(context.Background(), fn)

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if !result.Success {
		t.Error("expected Success = true")
	}
}

func TestExecutor_RetryDelays(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 50 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 50*time.Millisecond, 0)
	executor := NewExecutor(policy)

	testErr := errors.New("error")

	// Track retry times
	var retryTimes []time.Time

	fn := func(ctx context.Context) (interface{}, error) {
		retryTimes = append(retryTimes, time.Now())
		return nil, testErr
	}

	_, _ = executor.Execute(context.Background(), fn)

	// Verify delays between attempts
	if len(retryTimes) < 2 {
		t.Fatal("not enough attempts recorded")
	}

	for i := 1; i < len(retryTimes); i++ {
		delay := retryTimes[i].Sub(retryTimes[i-1])
		// Allow some tolerance for timing
		if delay < 40*time.Millisecond || delay > 70*time.Millisecond {
			t.Errorf("delay between attempt %d and %d = %v, want ~50ms", i, i+1, delay)
		}
	}
}

func TestExecutor_TotalDuration(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 20 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 20*time.Millisecond, 0)
	executor := NewExecutor(policy)

	testErr := errors.New("error")

	fn := func(ctx context.Context) (interface{}, error) {
		return nil, testErr
	}

	result, _ := executor.Execute(context.Background(), fn)

	// Total duration should be at least 2 delays (between 3 attempts)
	minExpected := 40 * time.Millisecond
	if result.TotalDuration < minExpected {
		t.Errorf("TotalDuration = %v, want >= %v", result.TotalDuration, minExpected)
	}
}

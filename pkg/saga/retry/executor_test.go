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

// Test new features

func TestExecutor_WithCircuitBreaker(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)

	cbConfig := &CircuitBreakerConfig{
		MaxFailures:         2,
		ResetTimeout:        100 * time.Millisecond,
		HalfOpenMaxRequests: 1,
		SuccessThreshold:    1,
	}

	cb := NewCircuitBreaker(cbConfig, policy)
	executor := NewExecutor(policy, WithCircuitBreaker(cb))

	testErr := errors.New("error")
	attemptCount := 0

	// Function that always fails
	fn := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		return nil, testErr
	}

	// First execution should exhaust retries and open circuit
	result, err := executor.Execute(context.Background(), fn)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result.Success {
		t.Error("expected failure")
	}

	// Reset attempt count
	attemptCount = 0

	// Second execution should fail fast due to open circuit
	result2, err2 := executor.Execute(context.Background(), fn)
	if !errors.Is(err2, ErrCircuitBreakerOpen) {
		t.Errorf("expected ErrCircuitBreakerOpen, got %v", err2)
	}
	if attemptCount > 0 {
		t.Errorf("expected no attempts due to open circuit, got %d", attemptCount)
	}
	if result2.Attempts != 0 {
		t.Errorf("expected 0 attempts, got %d", result2.Attempts)
	}
}

type permanentError struct {
	msg string
}

func (e *permanentError) Error() string {
	return e.msg
}

type customErrorClassifier struct{}

func (c *customErrorClassifier) IsRetryable(err error) bool {
	_, ok := err.(*permanentError)
	return !ok
}

func (c *customErrorClassifier) IsPermanent(err error) bool {
	_, ok := err.(*permanentError)
	return ok
}

func TestExecutor_WithErrorClassifier(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)
	classifier := &customErrorClassifier{}
	executor := NewExecutor(policy, WithErrorClassifier(classifier))

	permErr := &permanentError{msg: "permanent failure"}
	attemptCount := 0

	// Function that returns permanent error
	fn := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		return nil, permErr
	}

	result, err := executor.Execute(context.Background(), fn)

	// Should fail immediately without retries
	if err == nil {
		t.Error("expected error, got nil")
	}
	if result.Success {
		t.Error("expected failure")
	}
	if attemptCount != 1 {
		t.Errorf("expected 1 attempt for permanent error, got %d", attemptCount)
	}
	if result.Attempts != 1 {
		t.Errorf("expected 1 attempt in result, got %d", result.Attempts)
	}
}

func TestExecutor_WithMetrics(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)

	collector, err := NewRetryMetricsCollector("test", "retry", nil)
	if err != nil {
		t.Fatalf("failed to create metrics collector: %v", err)
	}

	executor := NewExecutor(policy, WithMetrics(collector))

	// Test successful execution
	fn := func(ctx context.Context) (interface{}, error) {
		return "success", nil
	}

	result, err := executor.Execute(context.Background(), fn)
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	// Metrics should be recorded (we can't easily verify Prometheus metrics in tests,
	// but we can verify no panics occurred)
}

func TestExecutor_WithLogger(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  2,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)

	// Use the global logger (already initialized in tests)
	executor := NewExecutor(policy)

	testErr := errors.New("temporary error")
	attemptCount := 0

	fn := func(ctx context.Context) (interface{}, error) {
		attemptCount++
		if attemptCount < 2 {
			return nil, testErr
		}
		return "success", nil
	}

	result, err := executor.Execute(context.Background(), fn)
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if !result.Success {
		t.Error("expected success")
	}

	// Logger should have logged without panicking
}

func TestExecuteWithResult_Success(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)
	executor := NewExecutor(policy)

	// Test with string type
	fn := func(ctx context.Context) (string, error) {
		return "typed result", nil
	}

	result, err := ExecuteWithResult(context.Background(), executor, fn)
	if err != nil {
		t.Errorf("ExecuteWithResult() error = %v, want nil", err)
	}
	if result != "typed result" {
		t.Errorf("result = %v, want 'typed result'", result)
	}
}

func TestExecuteWithResult_StructType(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)
	executor := NewExecutor(policy)

	type Response struct {
		ID      int
		Message string
	}

	// Test with struct type
	fn := func(ctx context.Context) (*Response, error) {
		return &Response{ID: 42, Message: "hello"}, nil
	}

	result, err := ExecuteWithResult(context.Background(), executor, fn)
	if err != nil {
		t.Errorf("ExecuteWithResult() error = %v, want nil", err)
	}
	if result.ID != 42 {
		t.Errorf("result.ID = %v, want 42", result.ID)
	}
	if result.Message != "hello" {
		t.Errorf("result.Message = %v, want 'hello'", result.Message)
	}
}

func TestExecuteWithResult_WithRetry(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)
	executor := NewExecutor(policy)

	attemptCount := 0
	testErr := errors.New("temporary error")

	fn := func(ctx context.Context) (int, error) {
		attemptCount++
		if attemptCount < 3 {
			return 0, testErr
		}
		return 42, nil
	}

	result, err := ExecuteWithResult(context.Background(), executor, fn)
	if err != nil {
		t.Errorf("ExecuteWithResult() error = %v, want nil", err)
	}
	if result != 42 {
		t.Errorf("result = %v, want 42", result)
	}
	if attemptCount != 3 {
		t.Errorf("attemptCount = %d, want 3", attemptCount)
	}
}

func TestExecuteWithResult_Failure(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  2,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)
	executor := NewExecutor(policy)

	testErr := errors.New("persistent error")

	fn := func(ctx context.Context) (string, error) {
		return "", testErr
	}

	result, err := ExecuteWithResult(context.Background(), executor, fn)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Errorf("expected ErrMaxRetriesExceeded, got %v", err)
	}
	if result != "" {
		t.Errorf("result = %v, want empty string", result)
	}
}

func TestDefaultErrorClassifier_IsRetryable(t *testing.T) {
	classifier := &DefaultErrorClassifier{}

	// Nil error should not be retryable
	if classifier.IsRetryable(nil) {
		t.Error("expected nil error to not be retryable")
	}

	// Non-nil error should be retryable
	err := errors.New("some error")
	if !classifier.IsRetryable(err) {
		t.Error("expected error to be retryable")
	}
}

func TestDefaultErrorClassifier_IsPermanent(t *testing.T) {
	classifier := &DefaultErrorClassifier{}

	// Default classifier treats no errors as permanent
	err := errors.New("some error")
	if classifier.IsPermanent(err) {
		t.Error("expected error to not be permanent")
	}

	if classifier.IsPermanent(nil) {
		t.Error("expected nil error to not be permanent")
	}
}

func TestRetryMetricsCollector_Creation(t *testing.T) {
	// Test with default parameters
	collector, err := NewRetryMetricsCollector("", "", nil)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	if collector == nil {
		t.Error("expected non-nil collector")
	}
	if collector.GetRegistry() == nil {
		t.Error("expected non-nil registry")
	}

	// Test with custom parameters
	collector2, err := NewRetryMetricsCollector("custom", "subsys", nil)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	if collector2 == nil {
		t.Error("expected non-nil collector")
	}
}

func TestExecutor_CircuitBreakerPreventsExecution(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)

	// Circuit breaker opens after 2 failures (it opens on the 2nd failure)
	// So with MaxAttempts=5, we should get 2 actual executions before circuit opens
	cbConfig := &CircuitBreakerConfig{
		MaxFailures:         2,
		ResetTimeout:        200 * time.Millisecond,
		HalfOpenMaxRequests: 1,
		SuccessThreshold:    1,
	}

	cb := NewCircuitBreaker(cbConfig, policy)
	executor := NewExecutor(policy, WithCircuitBreaker(cb))

	testErr := errors.New("error")
	totalExecutions := 0

	fn := func(ctx context.Context) (interface{}, error) {
		totalExecutions++
		return nil, testErr
	}

	// First execution - will try until circuit opens (after 2 failures)
	executor.Execute(context.Background(), fn)

	firstExecCount := totalExecutions
	// Circuit breaker opens after recording 2 failures
	if firstExecCount < 2 {
		t.Errorf("expected at least 2 executions before circuit opens, got %d", firstExecCount)
	}

	// Second execution - should fail immediately due to open circuit
	result, err := executor.Execute(context.Background(), fn)

	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Errorf("expected ErrCircuitBreakerOpen, got %v", err)
	}

	if totalExecutions != firstExecCount {
		t.Errorf("expected no additional executions due to open circuit, got %d total (was %d)",
			totalExecutions, firstExecCount)
	}

	if result.Attempts != 0 {
		t.Errorf("expected 0 attempts due to circuit breaker, got %d", result.Attempts)
	}
}

func TestExecutor_ErrorClassifierStopsPermanentError(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
	}

	policy := NewFixedIntervalPolicy(config, 10*time.Millisecond, 0)
	classifier := &customErrorClassifier{}
	executor := NewExecutor(policy, WithErrorClassifier(classifier))

	permErr := &permanentError{msg: "permanent"}
	callCount := 0

	fn := func(ctx context.Context) (interface{}, error) {
		callCount++
		return nil, permErr
	}

	result, err := executor.Execute(context.Background(), fn)

	if err == nil {
		t.Error("expected error, got nil")
	}

	// Should only call once for permanent error
	if callCount != 1 {
		t.Errorf("expected 1 call for permanent error, got %d", callCount)
	}

	if result.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", result.Attempts)
	}
}

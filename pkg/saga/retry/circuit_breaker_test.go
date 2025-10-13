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
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         3,
		ResetTimeout:        100 * time.Millisecond,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	retryConfig := &RetryConfig{
		MaxAttempts:  10, // High limit so we don't hit it
		InitialDelay: 10 * time.Millisecond,
	}
	basePolicy := NewFixedIntervalPolicy(retryConfig, 10*time.Millisecond, 0)
	cb := NewCircuitBreaker(config, basePolicy)

	testErr := errors.New("test error")

	// Initial state should be closed
	if cb.GetState() != CircuitStateClosed {
		t.Errorf("initial state = %v, want %v", cb.GetState(), CircuitStateClosed)
	}

	// Record failures to open the circuit
	for i := 1; i <= 3; i++ {
		shouldRetry := cb.ShouldRetry(testErr, i)
		if !shouldRetry {
			t.Errorf("attempt %d: ShouldRetry = false, want true", i)
		}
	}

	// Circuit should be open now
	if cb.GetState() != CircuitStateOpen {
		t.Errorf("after failures, state = %v, want %v", cb.GetState(), CircuitStateOpen)
	}

	// Should fail fast while open
	if cb.ShouldRetry(testErr, 4) {
		t.Error("ShouldRetry = true while circuit open, want false")
	}

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Check state after timeout (should transition when we make next call)
	// Make a successful call to transition to half-open and stay there
	cb.ShouldRetry(nil, 1)

	if cb.GetState() != CircuitStateHalfOpen {
		t.Errorf("after reset timeout with success, state = %v, want %v", cb.GetState(), CircuitStateHalfOpen)
	}

	// Another success in half-open state should increment success counter
	cb.ShouldRetry(nil, 1)

	// After enough successes, should transition to closed
	if cb.GetState() != CircuitStateClosed {
		t.Errorf("after successes, state = %v, want %v", cb.GetState(), CircuitStateClosed)
	}
}

func TestCircuitBreaker_FailureInHalfOpen(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	retryConfig := &RetryConfig{
		MaxAttempts:  10, // High limit so we don't hit it
		InitialDelay: 10 * time.Millisecond,
	}
	basePolicy := NewFixedIntervalPolicy(retryConfig, 10*time.Millisecond, 0)
	cb := NewCircuitBreaker(config, basePolicy)

	testErr := errors.New("test error")

	// Open the circuit
	cb.ShouldRetry(testErr, 1)
	cb.ShouldRetry(testErr, 2)

	if cb.GetState() != CircuitStateOpen {
		t.Errorf("state = %v, want %v", cb.GetState(), CircuitStateOpen)
	}

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Transition to half-open with a successful call first
	cb.ShouldRetry(nil, 1)

	if cb.GetState() != CircuitStateHalfOpen {
		t.Errorf("state = %v, want %v", cb.GetState(), CircuitStateHalfOpen)
	}

	// Failure in half-open should reopen the circuit
	cb.ShouldRetry(testErr, 1)

	if cb.GetState() != CircuitStateOpen {
		t.Errorf("after failure in half-open, state = %v, want %v", cb.GetState(), CircuitStateOpen)
	}
}

func TestCircuitBreaker_HalfOpenRequestLimit(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    3,
	}

	basePolicy := NewFixedIntervalPolicy(DefaultRetryConfig(), 10*time.Millisecond, 0)
	cb := NewCircuitBreaker(config, basePolicy)

	testErr := errors.New("test error")

	// Open the circuit
	cb.ShouldRetry(testErr, 1)
	cb.ShouldRetry(testErr, 2)

	// Wait for reset
	time.Sleep(60 * time.Millisecond)

	// Transition to half-open
	cb.ShouldRetry(testErr, 3)

	// Should allow up to HalfOpenMaxRequests
	count := 0
	for i := 0; i < 5; i++ {
		if cb.ShouldRetry(testErr, i+4) {
			count++
		}
	}

	// Should have limited to HalfOpenMaxRequests
	if count > config.HalfOpenMaxRequests {
		t.Errorf("allowed %d requests in half-open, want max %d", count, config.HalfOpenMaxRequests)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		ResetTimeout:        1 * time.Minute, // long timeout
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	basePolicy := NewFixedIntervalPolicy(DefaultRetryConfig(), 10*time.Millisecond, 0)
	cb := NewCircuitBreaker(config, basePolicy)

	testErr := errors.New("test error")

	// Open the circuit
	cb.ShouldRetry(testErr, 1)
	cb.ShouldRetry(testErr, 2)

	if cb.GetState() != CircuitStateOpen {
		t.Errorf("state = %v, want %v", cb.GetState(), CircuitStateOpen)
	}

	// Manual reset
	cb.Reset()

	if cb.GetState() != CircuitStateClosed {
		t.Errorf("after reset, state = %v, want %v", cb.GetState(), CircuitStateClosed)
	}

	// Should allow retries again
	if !cb.ShouldRetry(testErr, 1) {
		t.Error("ShouldRetry = false after reset, want true")
	}
}

func TestCircuitBreaker_GetMetrics(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         3,
		ResetTimeout:        100 * time.Millisecond,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	basePolicy := NewFixedIntervalPolicy(DefaultRetryConfig(), 10*time.Millisecond, 0)
	cb := NewCircuitBreaker(config, basePolicy)

	testErr := errors.New("test error")

	// Record some failures
	cb.ShouldRetry(testErr, 1)
	cb.ShouldRetry(testErr, 2)

	metrics := cb.GetMetrics()

	if metrics.State != CircuitStateClosed {
		t.Errorf("metrics.State = %v, want %v", metrics.State, CircuitStateClosed)
	}
	if metrics.ConsecutiveFailures != 2 {
		t.Errorf("metrics.ConsecutiveFailures = %d, want 2", metrics.ConsecutiveFailures)
	}
	if metrics.LastFailureTime.IsZero() {
		t.Error("expected LastFailureTime to be set")
	}
}

func TestCircuitBreaker_OnStateChange(t *testing.T) {
	var stateChanges []CircuitState

	config := &CircuitBreakerConfig{
		MaxFailures:         2,
		ResetTimeout:        50 * time.Millisecond,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
		OnStateChange: func(from, to CircuitState) {
			stateChanges = append(stateChanges, to)
		},
	}

	basePolicy := NewFixedIntervalPolicy(DefaultRetryConfig(), 10*time.Millisecond, 0)
	cb := NewCircuitBreaker(config, basePolicy)

	testErr := errors.New("test error")

	// Open the circuit
	cb.ShouldRetry(testErr, 1)
	cb.ShouldRetry(testErr, 2)

	// Wait for callback to be called
	time.Sleep(10 * time.Millisecond)

	if len(stateChanges) < 1 {
		t.Error("expected state change callback to be called")
	}
	if len(stateChanges) > 0 && stateChanges[0] != CircuitStateOpen {
		t.Errorf("first state change = %v, want %v", stateChanges[0], CircuitStateOpen)
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state CircuitState
		want  string
	}{
		{CircuitStateClosed, "closed"},
		{CircuitStateOpen, "open"},
		{CircuitStateHalfOpen, "half-open"},
		{CircuitState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.state.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig()

	if config.MaxFailures != 5 {
		t.Errorf("MaxFailures = %d, want 5", config.MaxFailures)
	}
	if config.ResetTimeout != 60*time.Second {
		t.Errorf("ResetTimeout = %v, want 60s", config.ResetTimeout)
	}
	if config.HalfOpenMaxRequests != 3 {
		t.Errorf("HalfOpenMaxRequests = %d, want 3", config.HalfOpenMaxRequests)
	}
	if config.SuccessThreshold != 2 {
		t.Errorf("SuccessThreshold = %d, want 2", config.SuccessThreshold)
	}
}

func TestNewCircuitBreaker_NilConfig(t *testing.T) {
	basePolicy := NewFixedIntervalPolicy(DefaultRetryConfig(), 10*time.Millisecond, 0)
	cb := NewCircuitBreaker(nil, basePolicy)

	if cb.config == nil {
		t.Error("expected default config, got nil")
	}
	if cb.GetState() != CircuitStateClosed {
		t.Errorf("initial state = %v, want %v", cb.GetState(), CircuitStateClosed)
	}
}

func TestCircuitBreaker_SuccessInClosedState(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxFailures:         3,
		ResetTimeout:        100 * time.Millisecond,
		HalfOpenMaxRequests: 2,
		SuccessThreshold:    2,
	}

	basePolicy := NewFixedIntervalPolicy(DefaultRetryConfig(), 10*time.Millisecond, 0)
	cb := NewCircuitBreaker(config, basePolicy)

	testErr := errors.New("test error")

	// Record some failures
	cb.ShouldRetry(testErr, 1)
	cb.ShouldRetry(testErr, 2)

	metrics := cb.GetMetrics()
	if metrics.ConsecutiveFailures != 2 {
		t.Errorf("ConsecutiveFailures = %d, want 2", metrics.ConsecutiveFailures)
	}

	// Record success - should reset failure counter
	cb.ShouldRetry(nil, 1)

	metrics = cb.GetMetrics()
	if metrics.ConsecutiveFailures != 0 {
		t.Errorf("after success, ConsecutiveFailures = %d, want 0", metrics.ConsecutiveFailures)
	}
}

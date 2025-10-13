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

func TestExponentialBackoffPolicy_GetRetryDelay(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
	}

	policy := NewExponentialBackoffPolicy(config, 2.0, 0) // no jitter for testing

	tests := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{"first retry", 1, 100 * time.Millisecond},           // 100ms * 2^0
		{"second retry", 2, 200 * time.Millisecond},          // 100ms * 2^1
		{"third retry", 3, 400 * time.Millisecond},           // 100ms * 2^2
		{"fourth retry", 4, 800 * time.Millisecond},          // 100ms * 2^3
		{"fifth retry", 5, 1600 * time.Millisecond},          // 100ms * 2^4
		{"sixth retry (capped)", 6, 3200 * time.Millisecond}, // 100ms * 2^5
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.GetRetryDelay(tt.attempt)
			if got != tt.want {
				t.Errorf("GetRetryDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestExponentialBackoffPolicy_MaxDelayCap(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	policy := NewExponentialBackoffPolicy(config, 2.0, 0)

	// After a few attempts, delay should be capped at MaxDelay
	for attempt := 1; attempt <= 10; attempt++ {
		delay := policy.GetRetryDelay(attempt)
		if delay > config.MaxDelay {
			t.Errorf("GetRetryDelay(%d) = %v, exceeds MaxDelay %v", attempt, delay, config.MaxDelay)
		}
	}
}

func TestExponentialBackoffPolicy_ShouldRetry(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts: 3,
	}

	policy := NewExponentialBackoffPolicy(config, 2.0, 0)
	testErr := errors.New("test error")

	tests := []struct {
		name    string
		err     error
		attempt int
		want    bool
	}{
		{"nil error", nil, 1, false},
		{"first attempt", testErr, 1, true},
		{"second attempt", testErr, 2, true},
		{"third attempt (max)", testErr, 3, false},
		{"beyond max", testErr, 4, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.ShouldRetry(tt.err, tt.attempt)
			if got != tt.want {
				t.Errorf("ShouldRetry(%v, %d) = %v, want %v", tt.err, tt.attempt, got, tt.want)
			}
		})
	}
}

func TestExponentialBackoffPolicy_WithJitter(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
	}

	policy := NewExponentialBackoffPolicy(config, 2.0, 0.5)

	// With jitter, delays should vary but be within expected range
	for attempt := 1; attempt <= 5; attempt++ {
		delay := policy.GetRetryDelay(attempt)
		if delay < 0 {
			t.Errorf("GetRetryDelay(%d) returned negative delay: %v", attempt, delay)
		}
		if delay > config.MaxDelay {
			t.Errorf("GetRetryDelay(%d) = %v, exceeds MaxDelay %v", attempt, delay, config.MaxDelay)
		}
	}
}

func TestLinearBackoffPolicy_GetRetryDelay(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	increment := 50 * time.Millisecond
	policy := NewLinearBackoffPolicy(config, increment, 0) // no jitter

	tests := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{"first retry", 1, 100 * time.Millisecond},  // 100ms + 50ms*0
		{"second retry", 2, 150 * time.Millisecond}, // 100ms + 50ms*1
		{"third retry", 3, 200 * time.Millisecond},  // 100ms + 50ms*2
		{"fourth retry", 4, 250 * time.Millisecond}, // 100ms + 50ms*3
		{"fifth retry", 5, 300 * time.Millisecond},  // 100ms + 50ms*4
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.GetRetryDelay(tt.attempt)
			if got != tt.want {
				t.Errorf("GetRetryDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestLinearBackoffPolicy_MaxDelayCap(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
	}

	increment := 100 * time.Millisecond
	policy := NewLinearBackoffPolicy(config, increment, 0)

	// Delays should be capped at MaxDelay
	for attempt := 1; attempt <= 10; attempt++ {
		delay := policy.GetRetryDelay(attempt)
		if delay > config.MaxDelay {
			t.Errorf("GetRetryDelay(%d) = %v, exceeds MaxDelay %v", attempt, delay, config.MaxDelay)
		}
	}
}

func TestLinearBackoffPolicy_ShouldRetry(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts: 3,
	}

	policy := NewLinearBackoffPolicy(config, 100*time.Millisecond, 0)
	testErr := errors.New("test error")

	tests := []struct {
		name    string
		err     error
		attempt int
		want    bool
	}{
		{"nil error", nil, 1, false},
		{"first attempt", testErr, 1, true},
		{"second attempt", testErr, 2, true},
		{"third attempt (max)", testErr, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.ShouldRetry(tt.err, tt.attempt)
			if got != tt.want {
				t.Errorf("ShouldRetry(%v, %d) = %v, want %v", tt.err, tt.attempt, got, tt.want)
			}
		})
	}
}

func TestFixedIntervalPolicy_GetRetryDelay(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
	}

	interval := 200 * time.Millisecond
	policy := NewFixedIntervalPolicy(config, interval, 0) // no jitter

	// All delays should be the same
	for attempt := 1; attempt <= 5; attempt++ {
		got := policy.GetRetryDelay(attempt)
		if got != interval {
			t.Errorf("GetRetryDelay(%d) = %v, want %v", attempt, got, interval)
		}
	}
}

func TestFixedIntervalPolicy_DefaultInterval(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 150 * time.Millisecond,
	}

	// Passing 0 should use InitialDelay
	policy := NewFixedIntervalPolicy(config, 0, 0)

	delay := policy.GetRetryDelay(1)
	if delay != config.InitialDelay {
		t.Errorf("GetRetryDelay(1) = %v, want %v", delay, config.InitialDelay)
	}
}

func TestFixedIntervalPolicy_ShouldRetry(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts: 3,
	}

	policy := NewFixedIntervalPolicy(config, 100*time.Millisecond, 0)
	testErr := errors.New("test error")

	tests := []struct {
		name    string
		err     error
		attempt int
		want    bool
	}{
		{"nil error", nil, 1, false},
		{"first attempt", testErr, 1, true},
		{"second attempt", testErr, 2, true},
		{"third attempt (max)", testErr, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.ShouldRetry(tt.err, tt.attempt)
			if got != tt.want {
				t.Errorf("ShouldRetry(%v, %d) = %v, want %v", tt.err, tt.attempt, got, tt.want)
			}
		})
	}
}

func TestNewExponentialBackoffPolicy_Defaults(t *testing.T) {
	// Test with nil config
	policy := NewExponentialBackoffPolicy(nil, 0, -1)

	if policy.Config == nil {
		t.Error("expected default config, got nil")
	}
	if policy.Multiplier != 2.0 {
		t.Errorf("expected default multiplier 2.0, got %f", policy.Multiplier)
	}
	if policy.Jitter != 0.0 {
		t.Errorf("expected jitter 0.0 (clamped), got %f", policy.Jitter)
	}
}

func TestNewLinearBackoffPolicy_Defaults(t *testing.T) {
	policy := NewLinearBackoffPolicy(nil, 0, -1)

	if policy.Config == nil {
		t.Error("expected default config, got nil")
	}
	if policy.Increment != 100*time.Millisecond {
		t.Errorf("expected default increment 100ms, got %v", policy.Increment)
	}
	if policy.Jitter != 0.0 {
		t.Errorf("expected jitter 0.0 (clamped), got %f", policy.Jitter)
	}
}

func TestNewFixedIntervalPolicy_Defaults(t *testing.T) {
	policy := NewFixedIntervalPolicy(nil, 0, 2.0)

	if policy.Config == nil {
		t.Error("expected default config, got nil")
	}
	if policy.Jitter != 1.0 {
		t.Errorf("expected jitter 1.0 (clamped), got %f", policy.Jitter)
	}
}

func TestJitterTypes(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
	}

	jitterTypes := []JitterType{JitterTypeFull, JitterTypeEqual, JitterTypeDecorelated}

	for _, jitterType := range jitterTypes {
		t.Run(jitterType.String(), func(t *testing.T) {
			policy := NewExponentialBackoffPolicy(config, 2.0, 0.5)
			policy.JitterType = jitterType

			// Just verify it doesn't panic and returns reasonable values
			for attempt := 1; attempt <= 3; attempt++ {
				delay := policy.GetRetryDelay(attempt)
				if delay < 0 {
					t.Errorf("negative delay for jitter type %v: %v", jitterType, delay)
				}
			}
		})
	}
}

func (jt JitterType) String() string {
	switch jt {
	case JitterTypeFull:
		return "Full"
	case JitterTypeEqual:
		return "Equal"
	case JitterTypeDecorelated:
		return "Decorrelated"
	default:
		return "Unknown"
	}
}

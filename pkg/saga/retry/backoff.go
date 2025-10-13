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
	"math"
	"math/rand"
	"time"
)

// ExponentialBackoffPolicy implements exponential backoff retry strategy.
// The delay grows exponentially with each retry attempt to reduce load on failing services.
// Formula: delay = InitialDelay * (Multiplier ^ (attempt - 1)) + jitter
type ExponentialBackoffPolicy struct {
	// Config is the base retry configuration
	Config *RetryConfig

	// Multiplier is the factor by which the delay increases with each attempt.
	// Must be >= 1.0. Typical values are 2.0 (doubling) or 1.5.
	Multiplier float64

	// Jitter adds randomness to delays to avoid thundering herd problem.
	// Value between 0.0 (no jitter) and 1.0 (max jitter).
	Jitter float64

	// JitterType specifies how jitter is applied (full, equal, decorrelated)
	JitterType JitterType
}

// JitterType defines different jittering strategies.
type JitterType int

const (
	// JitterTypeFull applies jitter to the full delay: delay * random(0, 1)
	JitterTypeFull JitterType = iota

	// JitterTypeEqual splits delay in half, adds jitter to one half: delay/2 + random(0, delay/2)
	JitterTypeEqual

	// JitterTypeDecorelated uses previous delay to calculate next delay with randomness
	JitterTypeDecorelated
)

// NewExponentialBackoffPolicy creates a new exponential backoff retry policy.
func NewExponentialBackoffPolicy(config *RetryConfig, multiplier, jitter float64) *ExponentialBackoffPolicy {
	if config == nil {
		config = DefaultRetryConfig()
	}
	if multiplier < 1.0 {
		multiplier = 2.0 // default doubling
	}
	if jitter < 0.0 {
		jitter = 0.0
	}
	if jitter > 1.0 {
		jitter = 1.0
	}

	return &ExponentialBackoffPolicy{
		Config:     config,
		Multiplier: multiplier,
		Jitter:     jitter,
		JitterType: JitterTypeFull,
	}
}

// ShouldRetry determines if an operation should be retried.
func (p *ExponentialBackoffPolicy) ShouldRetry(err error, attempt int) bool {
	if err == nil {
		return false
	}
	if attempt >= p.Config.MaxAttempts {
		return false
	}
	return p.Config.IsRetryableError(err)
}

// GetRetryDelay calculates the exponential backoff delay with jitter.
func (p *ExponentialBackoffPolicy) GetRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate base delay: InitialDelay * (Multiplier ^ (attempt - 1))
	baseDelay := float64(p.Config.InitialDelay) * math.Pow(p.Multiplier, float64(attempt-1))

	// Apply max delay cap
	if p.Config.MaxDelay > 0 && baseDelay > float64(p.Config.MaxDelay) {
		baseDelay = float64(p.Config.MaxDelay)
	}

	// Apply jitter
	delay := p.applyJitter(baseDelay, attempt)

	return time.Duration(delay)
}

// GetMaxAttempts returns the maximum number of retry attempts.
func (p *ExponentialBackoffPolicy) GetMaxAttempts() int {
	return p.Config.MaxAttempts
}

// applyJitter applies jitter to the delay based on the configured jitter type.
func (p *ExponentialBackoffPolicy) applyJitter(baseDelay float64, attempt int) float64 {
	if p.Jitter == 0.0 {
		return baseDelay
	}

	switch p.JitterType {
	case JitterTypeFull:
		// Full jitter: random(0, delay)
		return rand.Float64() * baseDelay

	case JitterTypeEqual:
		// Equal jitter: delay/2 + random(0, delay/2)
		half := baseDelay / 2
		return half + (rand.Float64() * half)

	case JitterTypeDecorelated:
		// Decorrelated jitter: random(InitialDelay, delay * 3)
		minDelay := float64(p.Config.InitialDelay)
		maxDelay := baseDelay * 3
		return minDelay + (rand.Float64() * (maxDelay - minDelay))

	default:
		return baseDelay
	}
}

// LinearBackoffPolicy implements linear backoff retry strategy.
// The delay increases linearly with each retry attempt.
// Formula: delay = InitialDelay + (Increment * (attempt - 1)) + jitter
type LinearBackoffPolicy struct {
	// Config is the base retry configuration
	Config *RetryConfig

	// Increment is the amount by which the delay increases with each attempt.
	Increment time.Duration

	// Jitter adds randomness to delays to avoid thundering herd problem.
	// Value between 0.0 (no jitter) and 1.0 (max jitter).
	Jitter float64
}

// NewLinearBackoffPolicy creates a new linear backoff retry policy.
func NewLinearBackoffPolicy(config *RetryConfig, increment time.Duration, jitter float64) *LinearBackoffPolicy {
	if config == nil {
		config = DefaultRetryConfig()
	}
	if increment <= 0 {
		increment = 100 * time.Millisecond // default increment
	}
	if jitter < 0.0 {
		jitter = 0.0
	}
	if jitter > 1.0 {
		jitter = 1.0
	}

	return &LinearBackoffPolicy{
		Config:    config,
		Increment: increment,
		Jitter:    jitter,
	}
}

// ShouldRetry determines if an operation should be retried.
func (p *LinearBackoffPolicy) ShouldRetry(err error, attempt int) bool {
	if err == nil {
		return false
	}
	if attempt >= p.Config.MaxAttempts {
		return false
	}
	return p.Config.IsRetryableError(err)
}

// GetRetryDelay calculates the linear backoff delay with jitter.
func (p *LinearBackoffPolicy) GetRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate base delay: InitialDelay + (Increment * (attempt - 1))
	baseDelay := p.Config.InitialDelay + (p.Increment * time.Duration(attempt-1))

	// Apply max delay cap
	if p.Config.MaxDelay > 0 && baseDelay > p.Config.MaxDelay {
		baseDelay = p.Config.MaxDelay
	}

	// Apply jitter
	delay := p.applyJitter(baseDelay)

	return delay
}

// GetMaxAttempts returns the maximum number of retry attempts.
func (p *LinearBackoffPolicy) GetMaxAttempts() int {
	return p.Config.MaxAttempts
}

// applyJitter applies full jitter to the delay.
func (p *LinearBackoffPolicy) applyJitter(baseDelay time.Duration) time.Duration {
	if p.Jitter == 0.0 {
		return baseDelay
	}

	// Full jitter: delay +/- (delay * jitter * random(-1, 1))
	jitterAmount := float64(baseDelay) * p.Jitter * (rand.Float64()*2 - 1)
	delay := float64(baseDelay) + jitterAmount

	// Ensure delay is not negative
	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}

// FixedIntervalPolicy implements fixed interval retry strategy.
// The delay remains constant for all retry attempts.
type FixedIntervalPolicy struct {
	// Config is the base retry configuration
	Config *RetryConfig

	// Interval is the fixed delay between retry attempts.
	// If not specified, Config.InitialDelay is used.
	Interval time.Duration

	// Jitter adds randomness to delays to avoid thundering herd problem.
	// Value between 0.0 (no jitter) and 1.0 (max jitter).
	Jitter float64
}

// NewFixedIntervalPolicy creates a new fixed interval retry policy.
func NewFixedIntervalPolicy(config *RetryConfig, interval time.Duration, jitter float64) *FixedIntervalPolicy {
	if config == nil {
		config = DefaultRetryConfig()
	}
	if interval <= 0 {
		interval = config.InitialDelay
	}
	if jitter < 0.0 {
		jitter = 0.0
	}
	if jitter > 1.0 {
		jitter = 1.0
	}

	return &FixedIntervalPolicy{
		Config:   config,
		Interval: interval,
		Jitter:   jitter,
	}
}

// ShouldRetry determines if an operation should be retried.
func (p *FixedIntervalPolicy) ShouldRetry(err error, attempt int) bool {
	if err == nil {
		return false
	}
	if attempt >= p.Config.MaxAttempts {
		return false
	}
	return p.Config.IsRetryableError(err)
}

// GetRetryDelay returns the fixed interval delay with optional jitter.
func (p *FixedIntervalPolicy) GetRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	baseDelay := p.Interval

	// Apply jitter
	delay := p.applyJitter(baseDelay)

	return delay
}

// GetMaxAttempts returns the maximum number of retry attempts.
func (p *FixedIntervalPolicy) GetMaxAttempts() int {
	return p.Config.MaxAttempts
}

// applyJitter applies full jitter to the delay.
func (p *FixedIntervalPolicy) applyJitter(baseDelay time.Duration) time.Duration {
	if p.Jitter == 0.0 {
		return baseDelay
	}

	// Full jitter: delay +/- (delay * jitter * random(-1, 1))
	jitterAmount := float64(baseDelay) * p.Jitter * (rand.Float64()*2 - 1)
	delay := float64(baseDelay) + jitterAmount

	// Ensure delay is not negative
	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}

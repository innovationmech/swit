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

package resilience

import (
	"context"
	"errors"
	"time"
)

// ShouldRetryFn reports whether an error is retryable.
type ShouldRetryFn func(error) bool

// OnRetryFn is invoked before each retry attempt with the attempt number and planned delay.
type OnRetryFn func(attempt int, err error, delay time.Duration)

// OnExhaustedFn is invoked when all retries are exhausted.
type OnExhaustedFn func(err error, attempts int)

// Executor performs retry for operations according to the provided Config and Strategy.
type Executor struct {
	cfg         Config
	calc        *Calculator
	shouldRetry ShouldRetryFn
	onRetry     OnRetryFn
	onExhausted OnExhaustedFn
}

// NewExecutor creates a new Executor after validating the provided Config.
func NewExecutor(cfg Config, opts ...Option) (*Executor, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	ex := &Executor{
		cfg:  cfg,
		calc: NewCalculator(cfg),
		shouldRetry: func(err error) bool {
			if err == nil {
				return false
			}
			// Do not retry on context cancellation/deadline errors
			return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
		},
	}
	for _, opt := range opts {
		opt(ex)
	}
	return ex, nil
}

// Option configures an Executor optional behavior.
type Option func(*Executor)

// WithShouldRetry overrides the retryable error decision function.
func WithShouldRetry(fn ShouldRetryFn) Option {
	return func(e *Executor) { e.shouldRetry = fn }
}

// WithOnRetry sets the retry callback.
func WithOnRetry(fn OnRetryFn) Option {
	return func(e *Executor) { e.onRetry = fn }
}

// WithOnExhausted sets the on-exhausted callback.
func WithOnExhausted(fn OnExhaustedFn) Option {
	return func(e *Executor) { e.onExhausted = fn }
}

// Do runs an operation without result and retries on failure according to the policy.
func (e *Executor) Do(ctx context.Context, op func() error) error {
	var lastErr error
	attempts := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := op()
		if err == nil {
			return nil
		}
		lastErr = err

		if !e.shouldRetry(err) || attempts >= e.cfg.MaxRetries || e.cfg.Strategy == StrategyNone {
			if e.onExhausted != nil {
				e.onExhausted(lastErr, attempts)
			}
			return lastErr
		}

		attempts++
		delay := e.calc.Delay(attempts)
		if e.onRetry != nil {
			e.onRetry(attempts, err, delay)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

// DoWithResult runs an operation with a result and retries on failure.
// It is provided as a package-level generic helper to avoid generic method restrictions.
func DoWithResult[T any](ctx context.Context, e *Executor, op func() (T, error)) (T, error) {
	var zero T
	var lastErr error
	attempts := 0

	for {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		res, err := op()
		if err == nil {
			return res, nil
		}
		lastErr = err

		if !e.shouldRetry(err) || attempts >= e.cfg.MaxRetries || e.cfg.Strategy == StrategyNone {
			if e.onExhausted != nil {
				e.onExhausted(lastErr, attempts)
			}
			return zero, lastErr
		}

		attempts++
		delay := e.calc.Delay(attempts)
		if e.onRetry != nil {
			e.onRetry(attempts, err, delay)
		}

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}
	}
}

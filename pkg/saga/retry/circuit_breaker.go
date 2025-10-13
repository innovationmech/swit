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
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// CircuitStateClosed indicates the circuit is closed (normal operation).
	CircuitStateClosed CircuitState = iota

	// CircuitStateOpen indicates the circuit is open (failing fast).
	CircuitStateOpen

	// CircuitStateHalfOpen indicates the circuit is half-open (testing recovery).
	CircuitStateHalfOpen
)

// String returns the string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case CircuitStateClosed:
		return "closed"
	case CircuitStateOpen:
		return "open"
	case CircuitStateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig defines the configuration for a circuit breaker.
type CircuitBreakerConfig struct {
	// MaxFailures is the maximum number of consecutive failures before opening the circuit.
	MaxFailures int

	// ResetTimeout is how long the circuit stays open before transitioning to half-open.
	ResetTimeout time.Duration

	// HalfOpenMaxRequests is the maximum number of requests allowed in half-open state.
	HalfOpenMaxRequests int

	// SuccessThreshold is the number of consecutive successes required to close the circuit
	// from half-open state.
	SuccessThreshold int

	// OnStateChange is called when the circuit state changes (optional).
	OnStateChange func(from, to CircuitState)
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration.
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		MaxFailures:         5,
		ResetTimeout:        60 * time.Second,
		HalfOpenMaxRequests: 3,
		SuccessThreshold:    2,
	}
}

// CircuitBreaker implements the circuit breaker pattern for retry policies.
// It prevents cascading failures by failing fast when a service is unhealthy.
type CircuitBreaker struct {
	// config is the circuit breaker configuration
	config *CircuitBreakerConfig

	// basePolicy is the underlying retry policy used when circuit is closed
	basePolicy interface {
		ShouldRetry(err error, attempt int) bool
		GetRetryDelay(attempt int) time.Duration
		GetMaxAttempts() int
	}

	// mu protects the circuit breaker state
	mu sync.RWMutex

	// state is the current circuit state
	state CircuitState

	// consecutiveFailures tracks consecutive failures in closed state
	consecutiveFailures int

	// consecutiveSuccesses tracks consecutive successes in half-open state
	consecutiveSuccesses int

	// halfOpenRequests tracks the number of requests in half-open state
	halfOpenRequests int

	// lastStateChange is when the state last changed
	lastStateChange time.Time

	// lastFailureTime is when the last failure occurred
	lastFailureTime time.Time
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration and base policy.
func NewCircuitBreaker(config *CircuitBreakerConfig, basePolicy interface {
	ShouldRetry(err error, attempt int) bool
	GetRetryDelay(attempt int) time.Duration
	GetMaxAttempts() int
}) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}

	return &CircuitBreaker{
		config:          config,
		basePolicy:      basePolicy,
		state:           CircuitStateClosed,
		lastStateChange: time.Now(),
	}
}

// ShouldRetry determines if an operation should be retried based on circuit state.
func (cb *CircuitBreaker) ShouldRetry(err error, attempt int) bool {
	// Check if circuit needs to transition from open to half-open
	cb.mu.RLock()
	state := cb.state
	cb.mu.RUnlock()

	if state == CircuitStateOpen {
		cb.checkResetTimeout()
		cb.mu.RLock()
		state = cb.state
		cb.mu.RUnlock()
	}

	// Handle success case
	if err == nil {
		cb.recordSuccess()
		return false
	}

	// If circuit is open, fail fast
	if state == CircuitStateOpen {
		return false
	}

	// If circuit is half-open, limit requests
	if state == CircuitStateHalfOpen {
		cb.mu.Lock()
		if cb.halfOpenRequests >= cb.config.HalfOpenMaxRequests {
			cb.mu.Unlock()
			return false
		}
		cb.halfOpenRequests++
		cb.mu.Unlock()
	}

	// Delegate to base policy first
	shouldRetry := cb.basePolicy.ShouldRetry(err, attempt)

	// Only record failure if we're going to retry
	// (this prevents recording failure when attempt limit is reached)
	if shouldRetry {
		cb.recordFailure()
	}

	return shouldRetry
}

// GetRetryDelay delegates to the base policy.
func (cb *CircuitBreaker) GetRetryDelay(attempt int) time.Duration {
	return cb.basePolicy.GetRetryDelay(attempt)
}

// GetMaxAttempts delegates to the base policy.
func (cb *CircuitBreaker) GetMaxAttempts() int {
	return cb.basePolicy.GetMaxAttempts()
}

// GetState returns the current circuit breaker state.
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// recordSuccess records a successful operation.
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitStateClosed:
		// Reset failure counter
		cb.consecutiveFailures = 0

	case CircuitStateHalfOpen:
		// Increment success counter
		cb.consecutiveSuccesses++
		cb.halfOpenRequests--

		// Check if we should close the circuit
		if cb.consecutiveSuccesses >= cb.config.SuccessThreshold {
			cb.transitionTo(CircuitStateClosed)
			cb.consecutiveFailures = 0
			cb.consecutiveSuccesses = 0
			cb.halfOpenRequests = 0
		}
	}
}

// recordFailure records a failed operation.
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitStateClosed:
		cb.consecutiveFailures++

		// Check if we should open the circuit
		if cb.consecutiveFailures >= cb.config.MaxFailures {
			cb.transitionTo(CircuitStateOpen)
		}

	case CircuitStateHalfOpen:
		// Any failure in half-open state reopens the circuit
		cb.transitionTo(CircuitStateOpen)
		cb.consecutiveSuccesses = 0
		cb.halfOpenRequests = 0
	}
}

// checkResetTimeout checks if enough time has passed to transition from open to half-open.
func (cb *CircuitBreaker) checkResetTimeout() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state != CircuitStateOpen {
		return
	}

	if time.Since(cb.lastStateChange) >= cb.config.ResetTimeout {
		cb.transitionTo(CircuitStateHalfOpen)
		cb.halfOpenRequests = 0
		cb.consecutiveSuccesses = 0
	}
}

// transitionTo transitions the circuit to a new state and notifies listeners.
// Must be called with lock held.
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	// Call state change callback if configured
	if cb.config.OnStateChange != nil {
		// Call callback without holding lock to prevent deadlocks
		go cb.config.OnStateChange(oldState, newState)
	}
}

// Reset manually resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.transitionTo(CircuitStateClosed)
	cb.consecutiveFailures = 0
	cb.consecutiveSuccesses = 0
	cb.halfOpenRequests = 0
}

// GetMetrics returns current metrics about the circuit breaker.
func (cb *CircuitBreaker) GetMetrics() CircuitBreakerMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerMetrics{
		State:                cb.state,
		ConsecutiveFailures:  cb.consecutiveFailures,
		ConsecutiveSuccesses: cb.consecutiveSuccesses,
		HalfOpenRequests:     cb.halfOpenRequests,
		LastStateChange:      cb.lastStateChange,
		LastFailureTime:      cb.lastFailureTime,
	}
}

// CircuitBreakerMetrics contains metrics about circuit breaker operation.
type CircuitBreakerMetrics struct {
	State                CircuitState
	ConsecutiveFailures  int
	ConsecutiveSuccesses int
	HalfOpenRequests     int
	LastStateChange      time.Time
	LastFailureTime      time.Time
}

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
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState string

const (
	// StateClosed indicates the circuit is closed and requests pass through
	StateClosed CircuitState = "CLOSED"

	// StateOpen indicates the circuit is open and requests are rejected
	StateOpen CircuitState = "OPEN"

	// StateHalfOpen indicates the circuit is testing if it should close
	StateHalfOpen CircuitState = "HALF_OPEN"
)

// CircuitCounts holds statistics about circuit breaker operations.
type CircuitCounts struct {
	Requests             uint64 `json:"requests"`
	TotalSuccesses       uint64 `json:"total_successes"`
	TotalFailures        uint64 `json:"total_failures"`
	ConsecutiveSuccesses uint64 `json:"consecutive_successes"`
	ConsecutiveFailures  uint64 `json:"consecutive_failures"`
}

// CircuitBreakerConfig defines configuration for circuit breaker behavior.
type CircuitBreakerConfig struct {
	// Name is the identifier for this circuit breaker
	Name string `json:"name" yaml:"name"`

	// MaxRequests is the maximum number of requests allowed in half-open state
	MaxRequests uint64 `json:"max_requests" yaml:"max_requests"`

	// Interval is the cyclic period for clearing counts in closed state
	Interval time.Duration `json:"interval" yaml:"interval"`

	// Timeout is the period the circuit stays open before transitioning to half-open
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// FailureThreshold is the number of consecutive failures needed to open the circuit
	FailureThreshold uint64 `json:"failure_threshold" yaml:"failure_threshold"`

	// SuccessThreshold is the number of consecutive successes needed to close from half-open
	SuccessThreshold uint64 `json:"success_threshold" yaml:"success_threshold"`

	// ErrorClassifier used to determine if errors should trip the circuit
	ErrorClassifier *ErrorClassifier `json:"-" yaml:"-"`

	// ShouldTrip is a custom function to determine if the circuit should trip
	ShouldTrip func(counts CircuitCounts) bool `json:"-" yaml:"-"`

	// IsSuccessful determines if an error should be counted as success
	IsSuccessful func(err error) bool `json:"-" yaml:"-"`

	// OnStateChange callback invoked when circuit state changes
	OnStateChange func(name string, from CircuitState, to CircuitState) `json:"-" yaml:"-"`
}

// NewCircuitBreakerConfig creates a new circuit breaker configuration with defaults.
func NewCircuitBreakerConfig(name string) *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		Name:             name,
		MaxRequests:      1,
		Interval:         60 * time.Second,
		Timeout:          60 * time.Second,
		FailureThreshold: 5,
		SuccessThreshold: 1,
		ErrorClassifier:  NewErrorClassifier(),
	}
}

// Validate validates the circuit breaker configuration.
func (cbc *CircuitBreakerConfig) Validate() error {
	if cbc.Name == "" {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("circuit breaker name cannot be empty").
			Build()
	}

	if cbc.MaxRequests == 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("max_requests must be positive").
			Build()
	}

	if cbc.Timeout <= 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("timeout must be positive").
			Build()
	}

	if cbc.FailureThreshold == 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("failure_threshold must be positive").
			Build()
	}

	if cbc.SuccessThreshold == 0 {
		return NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("success_threshold must be positive").
			Build()
	}

	return nil
}

// CircuitBreaker implements the circuit breaker pattern to prevent cascading failures.
// It maintains state and statistics to make decisions about allowing or rejecting requests.
type CircuitBreaker struct {
	config        *CircuitBreakerConfig
	state         CircuitState
	counts        CircuitCounts
	stateChanged  time.Time
	mutex         sync.RWMutex
	requestsMutex sync.Mutex
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(config *CircuitBreakerConfig) (*CircuitBreaker, error) {
	if config == nil {
		return nil, NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
			Message("circuit breaker config cannot be nil").
			Build()
	}

	if err := config.Validate(); err != nil {
		return nil, NewError(ErrorTypeConfiguration, ErrCodeConfigValidation).
			Message("invalid circuit breaker configuration").
			Cause(err).
			Build()
	}

	return &CircuitBreaker{
		config:       config,
		state:        StateClosed,
		stateChanged: time.Now(),
	}, nil
}

// Execute executes the given operation if the circuit breaker allows it.
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	// Check if we can proceed
	canProceed, err := cb.canProceed()
	if !canProceed {
		return err
	}

	// Execute the operation
	operationErr := operation()

	// Record the result
	cb.recordResult(operationErr)

	return operationErr
}

// ExecuteWithResult executes an operation that returns a result and error.
func (cb *CircuitBreaker) ExecuteWithResult(ctx context.Context, operation func() (interface{}, error)) (interface{}, error) {
	// Check if we can proceed
	canProceed, err := cb.canProceed()
	if !canProceed {
		return nil, err
	}

	// Execute the operation
	result, operationErr := operation()

	// Record the result
	cb.recordResult(operationErr)

	return result, operationErr
}

// canProceed determines if a request can proceed based on circuit state.
func (cb *CircuitBreaker) canProceed() (bool, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		// Check if we need to clear counts due to interval
		if cb.config.Interval > 0 && now.Sub(cb.stateChanged) > cb.config.Interval {
			cb.clearCounts()
		}
		return true, nil

	case StateOpen:
		// Check if timeout has passed and we should transition to half-open
		if now.Sub(cb.stateChanged) > cb.config.Timeout {
			cb.transitionTo(StateHalfOpen, now)
			return true, nil
		}
		return false, NewError(ErrorTypeResource, ErrCodeServiceUnavailable).
			Message("circuit breaker is open").
			Component("circuit-breaker").
			Operation("request").
			Context(ErrorContext{
				Component: cb.config.Name,
				Metadata: map[string]interface{}{
					"state":                string(cb.state),
					"time_until_half_open": cb.config.Timeout - now.Sub(cb.stateChanged),
				},
			}).
			SuggestedAction("Wait for circuit breaker to enter half-open state").
			Build()

	case StateHalfOpen:
		// Check if we've exceeded max requests in half-open state
		if cb.counts.Requests >= cb.config.MaxRequests {
			return false, NewError(ErrorTypeResource, ErrCodeServiceUnavailable).
				Message("circuit breaker half-open request limit exceeded").
				Component("circuit-breaker").
				Operation("request").
				Context(ErrorContext{
					Component: cb.config.Name,
					Metadata: map[string]interface{}{
						"state":        string(cb.state),
						"requests":     cb.counts.Requests,
						"max_requests": cb.config.MaxRequests,
					},
				}).
				SuggestedAction("Wait for circuit breaker to complete half-open evaluation").
				Build()
		}
		return true, nil

	default:
		return false, NewError(ErrorTypeInternal, ErrCodeInvalidState).
			Message("circuit breaker in unknown state").
			Component("circuit-breaker").
			Context(ErrorContext{
				Component: cb.config.Name,
				Metadata: map[string]interface{}{
					"state": string(cb.state),
				},
			}).
			Build()
	}
}

// recordResult records the result of an operation and updates circuit state accordingly.
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	isSuccess := cb.isSuccessful(err)

	// Update counts
	cb.counts.Requests++

	if isSuccess {
		cb.counts.TotalSuccesses++
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0
	} else {
		cb.counts.TotalFailures++
		cb.counts.ConsecutiveFailures++
		cb.counts.ConsecutiveSuccesses = 0
	}

	// Handle state transitions
	switch cb.state {
	case StateClosed:
		if !isSuccess && cb.shouldTrip() {
			cb.transitionTo(StateOpen, now)
		}

	case StateHalfOpen:
		if isSuccess {
			// Check if we have enough successes to close
			if cb.counts.ConsecutiveSuccesses >= cb.config.SuccessThreshold {
				cb.transitionTo(StateClosed, now)
			}
		} else {
			// Any failure in half-open state opens the circuit
			cb.transitionTo(StateOpen, now)
		}
	}
}

// isSuccessful determines if an operation result should be considered successful.
func (cb *CircuitBreaker) isSuccessful(err error) bool {
	if err == nil {
		return true
	}

	// Use custom success function if provided
	if cb.config.IsSuccessful != nil {
		return cb.config.IsSuccessful(err)
	}

	// Use error classifier to determine if error should trip circuit
	if cb.config.ErrorClassifier != nil {
		result := cb.config.ErrorClassifier.ClassifyError(err)
		// If circuit breaker is explicitly disabled for this error type, treat as success
		// If not specified or enabled, treat as failure (default behavior)
		if result.MatchedRule != nil && !result.CircuitBreakerEnabled {
			return true
		}
	}

	// Context cancellation should not trip the circuit
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Default: all errors are failures
	return false
}

// shouldTrip determines if the circuit should trip based on current counts.
func (cb *CircuitBreaker) shouldTrip() bool {
	// Use custom trip function if provided
	if cb.config.ShouldTrip != nil {
		return cb.config.ShouldTrip(cb.counts)
	}

	// Default: trip on consecutive failures threshold
	return cb.counts.ConsecutiveFailures >= cb.config.FailureThreshold
}

// transitionTo transitions the circuit to a new state.
func (cb *CircuitBreaker) transitionTo(newState CircuitState, now time.Time) {
	oldState := cb.state
	cb.state = newState
	cb.stateChanged = now

	// Clear counts on state transitions
	if newState == StateClosed || newState == StateHalfOpen {
		cb.clearCounts()
	}

	// Invoke state change callback if provided
	if cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(cb.config.Name, oldState, newState)
	}
}

// clearCounts resets all counts to zero.
func (cb *CircuitBreaker) clearCounts() {
	cb.counts = CircuitCounts{}
}

// GetState returns the current state of the circuit breaker.
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetCounts returns a copy of the current counts.
func (cb *CircuitBreaker) GetCounts() CircuitCounts {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.counts
}

// GetStateSince returns how long the circuit has been in its current state.
func (cb *CircuitBreaker) GetStateSince() time.Duration {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return time.Since(cb.stateChanged)
}

// Reset manually resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.stateChanged = time.Now()
	cb.clearCounts()

	// Invoke state change callback if state actually changed
	if oldState != StateClosed && cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(cb.config.Name, oldState, StateClosed)
	}
}

// ForceOpen manually forces the circuit breaker to open state.
func (cb *CircuitBreaker) ForceOpen() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	oldState := cb.state
	cb.state = StateOpen
	cb.stateChanged = time.Now()

	// Invoke state change callback if state actually changed
	if oldState != StateOpen && cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(cb.config.Name, oldState, StateOpen)
	}
}

// GetConfig returns a copy of the circuit breaker configuration.
func (cb *CircuitBreaker) GetConfig() CircuitBreakerConfig {
	return *cb.config
}

// CircuitBreakerManager manages multiple circuit breakers.
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager.
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one.
func (cbm *CircuitBreakerManager) GetOrCreate(name string, config *CircuitBreakerConfig) (*CircuitBreaker, error) {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	if breaker, exists := cbm.breakers[name]; exists {
		return breaker, nil
	}

	// Create new circuit breaker
	if config == nil {
		config = NewCircuitBreakerConfig(name)
	}
	config.Name = name // Ensure name matches

	breaker, err := NewCircuitBreaker(config)
	if err != nil {
		return nil, err
	}

	cbm.breakers[name] = breaker
	return breaker, nil
}

// Get retrieves an existing circuit breaker by name.
func (cbm *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	breaker, exists := cbm.breakers[name]
	return breaker, exists
}

// Remove removes a circuit breaker by name.
func (cbm *CircuitBreakerManager) Remove(name string) bool {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	if _, exists := cbm.breakers[name]; exists {
		delete(cbm.breakers, name)
		return true
	}
	return false
}

// List returns a copy of all circuit breaker names.
func (cbm *CircuitBreakerManager) List() []string {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	names := make([]string, 0, len(cbm.breakers))
	for name := range cbm.breakers {
		names = append(names, name)
	}
	return names
}

// GetAllStates returns the states of all circuit breakers.
func (cbm *CircuitBreakerManager) GetAllStates() map[string]CircuitState {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	states := make(map[string]CircuitState, len(cbm.breakers))
	for name, breaker := range cbm.breakers {
		states[name] = breaker.GetState()
	}
	return states
}

// ResetAll resets all circuit breakers to closed state.
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mutex.RLock()
	breakers := make([]*CircuitBreaker, 0, len(cbm.breakers))
	for _, breaker := range cbm.breakers {
		breakers = append(breakers, breaker)
	}
	cbm.mutex.RUnlock()

	// Reset all breakers without holding the manager lock
	for _, breaker := range breakers {
		breaker.Reset()
	}
}

// CircuitBreakerStatistics tracks circuit breaker operation statistics.
type CircuitBreakerStatistics struct {
	Name              string        `json:"name"`
	State             CircuitState  `json:"state"`
	StateSince        time.Duration `json:"state_since"`
	Counts            CircuitCounts `json:"counts"`
	TotalStateChanges uint64        `json:"total_state_changes"`
	TimesClosed       uint64        `json:"times_closed"`
	TimesOpen         uint64        `json:"times_open"`
	TimesHalfOpen     uint64        `json:"times_half_open"`
	TotalTimeOpen     time.Duration `json:"total_time_open"`
	TotalTimeHalfOpen time.Duration `json:"total_time_half_open"`
	LastStateChange   time.Time     `json:"last_state_change"`
}

// circuitBreakerStatisticsTracker is the internal tracker with mutex.
type circuitBreakerStatisticsTracker struct {
	stats CircuitBreakerStatistics
	mutex sync.RWMutex
}

// NewCircuitBreakerStatistics creates a new statistics tracker.
func NewCircuitBreakerStatistics(name string) *circuitBreakerStatisticsTracker {
	return &circuitBreakerStatisticsTracker{
		stats: CircuitBreakerStatistics{
			Name:  name,
			State: StateClosed,
		},
	}
}

// RecordStateChange records a state change.
func (cbst *circuitBreakerStatisticsTracker) RecordStateChange(from, to CircuitState, when time.Time) {
	cbst.mutex.Lock()
	defer cbst.mutex.Unlock()

	cbst.stats.TotalStateChanges++

	// Update time in previous state
	if !cbst.stats.LastStateChange.IsZero() {
		duration := when.Sub(cbst.stats.LastStateChange)
		switch from {
		case StateOpen:
			cbst.stats.TotalTimeOpen += duration
		case StateHalfOpen:
			cbst.stats.TotalTimeHalfOpen += duration
		}
	}

	cbst.stats.LastStateChange = when

	// Update counters for new state
	switch to {
	case StateClosed:
		cbst.stats.TimesClosed++
	case StateOpen:
		cbst.stats.TimesOpen++
	case StateHalfOpen:
		cbst.stats.TimesHalfOpen++
	}

	cbst.stats.State = to
}

// GetSnapshot returns a snapshot of current statistics.
func (cbst *circuitBreakerStatisticsTracker) GetSnapshot() CircuitBreakerStatistics {
	cbst.mutex.RLock()
	defer cbst.mutex.RUnlock()

	snapshot := cbst.stats
	if !cbst.stats.LastStateChange.IsZero() {
		snapshot.StateSince = time.Since(cbst.stats.LastStateChange)
	}
	return snapshot
}

// GetAvailabilityPercentage calculates availability as percentage of time not open.
func (cbst *circuitBreakerStatisticsTracker) GetAvailabilityPercentage() float64 {
	cbst.mutex.RLock()
	defer cbst.mutex.RUnlock()

	if cbst.stats.LastStateChange.IsZero() {
		return 100.0 // No state changes, assume fully available
	}

	totalTime := time.Since(cbst.stats.LastStateChange)
	if totalTime == 0 {
		return 100.0
	}

	openTime := cbst.stats.TotalTimeOpen
	if cbst.stats.State == StateOpen {
		openTime += time.Since(cbst.stats.LastStateChange)
	}

	availability := float64(totalTime-openTime) / float64(totalTime) * 100.0
	if availability < 0 {
		availability = 0
	}
	if availability > 100 {
		availability = 100
	}

	return availability
}

// InstrumentedCircuitBreaker adds statistics tracking to circuit breaker operations.
type InstrumentedCircuitBreaker struct {
	*CircuitBreaker
	statistics *circuitBreakerStatisticsTracker
}

// NewInstrumentedCircuitBreaker creates a new instrumented circuit breaker.
func NewInstrumentedCircuitBreaker(config *CircuitBreakerConfig) (*InstrumentedCircuitBreaker, error) {
	stats := NewCircuitBreakerStatistics(config.Name)

	// Wrap the original state change callback
	originalCallback := config.OnStateChange
	config.OnStateChange = func(name string, from, to CircuitState) {
		stats.RecordStateChange(from, to, time.Now())
		if originalCallback != nil {
			originalCallback(name, from, to)
		}
	}

	breaker, err := NewCircuitBreaker(config)
	if err != nil {
		return nil, err
	}

	return &InstrumentedCircuitBreaker{
		CircuitBreaker: breaker,
		statistics:     stats,
	}, nil
}

// GetStatistics returns the current statistics.
func (icb *InstrumentedCircuitBreaker) GetStatistics() CircuitBreakerStatistics {
	stats := icb.statistics.GetSnapshot()

	// Update current state information from circuit breaker
	stats.State = icb.GetState()
	stats.Counts = icb.GetCounts()
	stats.StateSince = icb.GetStateSince()

	return stats
}

// ResilienceExecutor combines retry logic with circuit breaker protection.
type ResilienceExecutor struct {
	retryExecutor  *RetryExecutor
	circuitBreaker *CircuitBreaker

	// Configuration
	enableRetryOnCircuitOpen bool
	enableCircuitBreaker     bool
	enableRetry              bool
}

// ResilienceConfig configures the resilience executor.
type ResilienceConfig struct {
	RetryPolicy              *RetryPolicy          `json:"retry_policy" yaml:"retry_policy"`
	CircuitBreakerConfig     *CircuitBreakerConfig `json:"circuit_breaker_config" yaml:"circuit_breaker_config"`
	EnableRetryOnCircuitOpen bool                  `json:"enable_retry_on_circuit_open" yaml:"enable_retry_on_circuit_open"`
	EnableCircuitBreaker     bool                  `json:"enable_circuit_breaker" yaml:"enable_circuit_breaker"`
	EnableRetry              bool                  `json:"enable_retry" yaml:"enable_retry"`
}

// NewResilienceExecutor creates a new resilience executor.
func NewResilienceExecutor(config *ResilienceConfig) (*ResilienceExecutor, error) {
	if config == nil {
		return nil, NewError(ErrorTypeConfiguration, ErrCodeInvalidConfig).
			Message("resilience config cannot be nil").
			Build()
	}

	executor := &ResilienceExecutor{
		enableRetryOnCircuitOpen: config.EnableRetryOnCircuitOpen,
		enableCircuitBreaker:     config.EnableCircuitBreaker,
		enableRetry:              config.EnableRetry,
	}

	// Initialize retry executor if enabled
	if config.EnableRetry && config.RetryPolicy != nil {
		retryExecutor, err := NewRetryExecutor(config.RetryPolicy)
		if err != nil {
			return nil, fmt.Errorf("failed to create retry executor: %w", err)
		}
		executor.retryExecutor = retryExecutor
	}

	// Initialize circuit breaker if enabled
	if config.EnableCircuitBreaker && config.CircuitBreakerConfig != nil {
		circuitBreaker, err := NewCircuitBreaker(config.CircuitBreakerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create circuit breaker: %w", err)
		}
		executor.circuitBreaker = circuitBreaker
	}

	return executor, nil
}

// Execute executes an operation with combined retry and circuit breaker protection.
func (re *ResilienceExecutor) Execute(ctx context.Context, operation func() error) error {
	// If no resilience mechanisms are enabled, execute directly
	if !re.enableRetry && !re.enableCircuitBreaker {
		return operation()
	}

	// Wrap operation with circuit breaker if enabled
	wrappedOp := operation
	if re.enableCircuitBreaker && re.circuitBreaker != nil {
		wrappedOp = func() error {
			return re.circuitBreaker.Execute(ctx, operation)
		}
	}

	// Execute with retry if enabled
	if re.enableRetry && re.retryExecutor != nil {
		return re.retryExecutor.Execute(ctx, wrappedOp)
	}

	// Execute with just circuit breaker
	return wrappedOp()
}

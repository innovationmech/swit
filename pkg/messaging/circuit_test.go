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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreakerConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      *CircuitBreakerConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &CircuitBreakerConfig{
				Name:             "test-breaker",
				MaxRequests:      5,
				Timeout:          60 * time.Second,
				FailureThreshold: 3,
				SuccessThreshold: 2,
			},
			expectError: false,
		},
		{
			name: "empty name",
			config: &CircuitBreakerConfig{
				Name:             "",
				MaxRequests:      5,
				Timeout:          60 * time.Second,
				FailureThreshold: 3,
				SuccessThreshold: 2,
			},
			expectError: true,
		},
		{
			name: "zero max requests",
			config: &CircuitBreakerConfig{
				Name:             "test-breaker",
				MaxRequests:      0,
				Timeout:          60 * time.Second,
				FailureThreshold: 3,
				SuccessThreshold: 2,
			},
			expectError: true,
		},
		{
			name: "zero timeout",
			config: &CircuitBreakerConfig{
				Name:             "test-breaker",
				MaxRequests:      5,
				Timeout:          0,
				FailureThreshold: 3,
				SuccessThreshold: 2,
			},
			expectError: true,
		},
		{
			name: "zero failure threshold",
			config: &CircuitBreakerConfig{
				Name:             "test-breaker",
				MaxRequests:      5,
				Timeout:          60 * time.Second,
				FailureThreshold: 0,
				SuccessThreshold: 2,
			},
			expectError: true,
		},
		{
			name: "zero success threshold",
			config: &CircuitBreakerConfig{
				Name:             "test-breaker",
				MaxRequests:      5,
				Timeout:          60 * time.Second,
				FailureThreshold: 3,
				SuccessThreshold: 0,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	t.Run("closed to open transition", func(t *testing.T) {
		config := NewCircuitBreakerConfig("test-breaker")
		config.FailureThreshold = 3
		config.Timeout = 100 * time.Millisecond
		config.ErrorClassifier = nil // Disable classifier to treat all errors as failures

		cb, err := NewCircuitBreaker(config)
		require.NoError(t, err)

		ctx := context.Background()

		// Circuit should start closed
		assert.Equal(t, StateClosed, cb.GetState())

		// Simulate failures to trigger opening
		for i := 0; i < 3; i++ {
			err := cb.Execute(ctx, func() error {
				return errors.New("test error")
			})
			assert.Error(t, err)
		}

		// Circuit should now be open
		assert.Equal(t, StateOpen, cb.GetState())
	})

	t.Run("open to half-open transition", func(t *testing.T) {
		config := NewCircuitBreakerConfig("test-breaker")
		config.FailureThreshold = 2
		config.SuccessThreshold = 5 // Set high so circuit doesn't close immediately
		config.Timeout = 10 * time.Millisecond
		config.ErrorClassifier = nil // Disable classifier to treat all errors as failures

		cb, err := NewCircuitBreaker(config)
		require.NoError(t, err)

		ctx := context.Background()

		// Trigger circuit opening
		for i := 0; i < 2; i++ {
			cb.Execute(ctx, func() error {
				return errors.New("test error")
			})
		}
		assert.Equal(t, StateOpen, cb.GetState())

		// Wait for timeout
		time.Sleep(15 * time.Millisecond)

		// Next call should transition to half-open
		err = cb.Execute(ctx, func() error {
			return nil // Success
		})
		assert.NoError(t, err)
		assert.Equal(t, StateHalfOpen, cb.GetState())
	})

	t.Run("half-open to closed transition", func(t *testing.T) {
		config := NewCircuitBreakerConfig("test-breaker")
		config.FailureThreshold = 2
		config.SuccessThreshold = 2
		config.Timeout = 10 * time.Millisecond
		config.MaxRequests = 10      // Allow more requests in half-open state
		config.ErrorClassifier = nil // Disable classifier to treat all errors as failures

		cb, err := NewCircuitBreaker(config)
		require.NoError(t, err)

		ctx := context.Background()

		// Force circuit to open
		cb.ForceOpen()
		assert.Equal(t, StateOpen, cb.GetState())

		// Wait for timeout and transition to half-open
		time.Sleep(15 * time.Millisecond)

		// First successful call to enter half-open
		err = cb.Execute(ctx, func() error {
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, StateHalfOpen, cb.GetState())

		// Second successful call to close circuit
		err = cb.Execute(ctx, func() error {
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, StateClosed, cb.GetState())
	})

	t.Run("half-open to open on failure", func(t *testing.T) {
		config := NewCircuitBreakerConfig("test-breaker")
		config.FailureThreshold = 2
		config.Timeout = 10 * time.Millisecond
		config.ErrorClassifier = nil // Disable classifier to treat all errors as failures

		cb, err := NewCircuitBreaker(config)
		require.NoError(t, err)

		ctx := context.Background()

		// Force circuit to open
		cb.ForceOpen()

		// Wait for timeout and transition to half-open
		time.Sleep(15 * time.Millisecond)

		// First call enters half-open but fails
		err = cb.Execute(ctx, func() error {
			return errors.New("test error")
		})
		assert.Error(t, err)
		assert.Equal(t, StateOpen, cb.GetState())
	})
}

func TestCircuitBreaker_RequestLimits(t *testing.T) {
	config := NewCircuitBreakerConfig("test-breaker")
	config.MaxRequests = 2
	config.FailureThreshold = 1
	config.SuccessThreshold = 5 // Set high so circuit doesn't close after 2 successes
	config.Timeout = 10 * time.Millisecond
	config.ErrorClassifier = nil // Disable classifier to treat all errors as failures

	cb, err := NewCircuitBreaker(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Force to open state
	cb.ForceOpen()

	// Wait for timeout to allow half-open
	time.Sleep(15 * time.Millisecond)

	// First request should be allowed (enters half-open)
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.GetState())

	// Second request should be allowed
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.NoError(t, err)

	// Third request should be rejected (exceeds max requests)
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.Error(t, err)

	var msgErr *BaseMessagingError
	assert.True(t, errors.As(err, &msgErr))
	assert.Equal(t, ErrorTypeResource, msgErr.Type)
}

func TestCircuitBreaker_CustomConditions(t *testing.T) {
	t.Run("custom should trip function", func(t *testing.T) {
		config := NewCircuitBreakerConfig("test-breaker")
		config.ErrorClassifier = nil // Disable classifier to treat all errors as failures
		config.ShouldTrip = func(counts CircuitCounts) bool {
			// Trip if failure rate > 50%
			if counts.Requests < 2 {
				return false
			}
			return float64(counts.TotalFailures)/float64(counts.Requests) > 0.5
		}

		cb, err := NewCircuitBreaker(config)
		require.NoError(t, err)

		ctx := context.Background()

		// One failure, one success - should not trip
		cb.Execute(ctx, func() error { return errors.New("error") })
		cb.Execute(ctx, func() error { return nil })
		assert.Equal(t, StateClosed, cb.GetState())

		// Two more failures - should trip (3 failures out of 4 = 75% > 50%)
		cb.Execute(ctx, func() error { return errors.New("error") })
		cb.Execute(ctx, func() error { return errors.New("error") })
		assert.Equal(t, StateOpen, cb.GetState())
	})

	t.Run("custom success function", func(t *testing.T) {
		config := NewCircuitBreakerConfig("test-breaker")
		config.FailureThreshold = 2
		config.ErrorClassifier = nil // Disable classifier to use custom success function
		config.IsSuccessful = func(err error) bool {
			// Treat specific errors as success
			return err == nil || err.Error() == "acceptable error"
		}

		cb, err := NewCircuitBreaker(config)
		require.NoError(t, err)

		ctx := context.Background()

		// These should be treated as successes
		cb.Execute(ctx, func() error { return errors.New("acceptable error") })
		cb.Execute(ctx, func() error { return nil })
		assert.Equal(t, StateClosed, cb.GetState())

		// These should be treated as failures
		cb.Execute(ctx, func() error { return errors.New("real error") })
		cb.Execute(ctx, func() error { return errors.New("another error") })
		assert.Equal(t, StateOpen, cb.GetState())
	})
}

func TestCircuitBreaker_StateCallbacks(t *testing.T) {
	var stateChanges []string
	var mu sync.Mutex

	config := NewCircuitBreakerConfig("test-breaker")
	config.FailureThreshold = 2
	config.Timeout = 10 * time.Millisecond
	config.ErrorClassifier = nil // Disable classifier to treat all errors as failures
	config.OnStateChange = func(name string, from, to CircuitState) {
		mu.Lock()
		stateChanges = append(stateChanges, string(from)+"->"+string(to))
		mu.Unlock()
	}

	cb, err := NewCircuitBreaker(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Trigger state changes
	cb.Execute(ctx, func() error { return errors.New("error") })
	cb.Execute(ctx, func() error { return errors.New("error") })

	// Should transition to open
	assert.Equal(t, StateOpen, cb.GetState())

	// Small delay to allow callback goroutine to complete
	time.Sleep(20 * time.Millisecond)

	// Should transition to half-open, then back to closed
	cb.Execute(ctx, func() error { return nil })
	cb.Execute(ctx, func() error { return nil })

	// Wait for all callback goroutines to complete
	time.Sleep(10 * time.Millisecond)

	// Verify state changes were recorded
	mu.Lock()
	stateChangesCopy := make([]string, len(stateChanges))
	copy(stateChangesCopy, stateChanges)
	mu.Unlock()

	assert.Contains(t, stateChangesCopy, "CLOSED->OPEN")
	assert.Contains(t, stateChangesCopy, "OPEN->HALF_OPEN")
	assert.Contains(t, stateChangesCopy, "HALF_OPEN->CLOSED")
}

func TestCircuitBreaker_ContextErrors(t *testing.T) {
	config := NewCircuitBreakerConfig("test-breaker")
	config.FailureThreshold = 2
	config.ErrorClassifier = nil // Disable classifier - context errors handled specially

	cb, err := NewCircuitBreaker(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Context cancellation should not trip the circuit
	cb.Execute(ctx, func() error { return context.Canceled })
	cb.Execute(ctx, func() error { return context.DeadlineExceeded })

	assert.Equal(t, StateClosed, cb.GetState())

	counts := cb.GetCounts()
	assert.Equal(t, uint64(2), counts.TotalSuccesses) // Context errors treated as success
	assert.Equal(t, uint64(0), counts.TotalFailures)
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := NewCircuitBreakerConfig("test-breaker")
	config.FailureThreshold = 2
	config.ErrorClassifier = nil // Disable classifier to treat all errors as failures

	cb, err := NewCircuitBreaker(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Force circuit open
	cb.Execute(ctx, func() error { return errors.New("error") })
	cb.Execute(ctx, func() error { return errors.New("error") })
	assert.Equal(t, StateOpen, cb.GetState())

	// Reset should close the circuit
	cb.Reset()
	assert.Equal(t, StateClosed, cb.GetState())

	counts := cb.GetCounts()
	assert.Equal(t, uint64(0), counts.Requests)
	assert.Equal(t, uint64(0), counts.TotalFailures)
}

func TestCircuitBreakerManager(t *testing.T) {
	manager := NewCircuitBreakerManager()

	t.Run("get or create circuit breaker", func(t *testing.T) {
		config := NewCircuitBreakerConfig("test-breaker-1")

		cb1, err := manager.GetOrCreate("test-breaker-1", config)
		require.NoError(t, err)
		assert.Equal(t, "test-breaker-1", cb1.GetConfig().Name)

		// Getting the same breaker should return the existing one
		cb2, err := manager.GetOrCreate("test-breaker-1", nil)
		require.NoError(t, err)
		assert.Same(t, cb1, cb2)
	})

	t.Run("list circuit breakers", func(t *testing.T) {
		config2 := NewCircuitBreakerConfig("test-breaker-2")
		manager.GetOrCreate("test-breaker-2", config2)

		names := manager.List()
		assert.Contains(t, names, "test-breaker-1")
		assert.Contains(t, names, "test-breaker-2")
		assert.Len(t, names, 2)
	})

	t.Run("get existing circuit breaker", func(t *testing.T) {
		cb, exists := manager.Get("test-breaker-1")
		assert.True(t, exists)
		assert.NotNil(t, cb)

		cb, exists = manager.Get("non-existent")
		assert.False(t, exists)
		assert.Nil(t, cb)
	})

	t.Run("remove circuit breaker", func(t *testing.T) {
		removed := manager.Remove("test-breaker-1")
		assert.True(t, removed)

		removed = manager.Remove("test-breaker-1")
		assert.False(t, removed)

		_, exists := manager.Get("test-breaker-1")
		assert.False(t, exists)
	})

	t.Run("get all states", func(t *testing.T) {
		states := manager.GetAllStates()
		assert.Contains(t, states, "test-breaker-2")
		assert.Equal(t, StateClosed, states["test-breaker-2"])
	})

	t.Run("reset all", func(t *testing.T) {
		// Force one breaker to open
		cb, _ := manager.Get("test-breaker-2")
		cb.ForceOpen()
		assert.Equal(t, StateOpen, cb.GetState())

		// Reset all should close it
		manager.ResetAll()
		assert.Equal(t, StateClosed, cb.GetState())
	})
}

func TestCircuitBreakerStatistics(t *testing.T) {
	stats := NewCircuitBreakerStatistics("test-breaker")

	t.Run("record state changes", func(t *testing.T) {
		now := time.Now()

		stats.RecordStateChange(StateClosed, StateOpen, now)
		snapshot := stats.GetSnapshot()

		assert.Equal(t, StateOpen, snapshot.State)
		assert.Equal(t, uint64(1), snapshot.TotalStateChanges)
		assert.Equal(t, uint64(1), snapshot.TimesOpen)
		assert.Equal(t, uint64(0), snapshot.TimesClosed)

		// Record transition back to closed after some time
		later := now.Add(5 * time.Second)
		stats.RecordStateChange(StateOpen, StateClosed, later)

		snapshot = stats.GetSnapshot()
		assert.Equal(t, StateClosed, snapshot.State)
		assert.Equal(t, uint64(2), snapshot.TotalStateChanges)
		assert.Equal(t, uint64(1), snapshot.TimesOpen)
		assert.Equal(t, uint64(1), snapshot.TimesClosed)
		assert.Equal(t, 5*time.Second, snapshot.TotalTimeOpen)
	})

	t.Run("availability calculation", func(t *testing.T) {
		// Reset for clean test
		stats = NewCircuitBreakerStatistics("test-breaker")

		availability := stats.GetAvailabilityPercentage()
		assert.Equal(t, 100.0, availability) // No state changes = 100% available

		// Simple test: circuit is currently open
		now := time.Now()
		stats.RecordStateChange(StateClosed, StateOpen, now)

		// Since circuit is currently open, availability should be 0%
		availability = stats.GetAvailabilityPercentage()
		assert.Equal(t, 0.0, availability) // Currently open, so 0% available

		// Close the circuit - should now be 100% available
		stats.RecordStateChange(StateOpen, StateClosed, now.Add(1*time.Second))
		availability = stats.GetAvailabilityPercentage()
		assert.Equal(t, 100.0, availability) // Currently closed, so 100% available
	})
}

func TestInstrumentedCircuitBreaker(t *testing.T) {
	config := NewCircuitBreakerConfig("test-breaker")
	config.FailureThreshold = 2
	config.Timeout = 1 * time.Second // Longer timeout to ensure circuit stays open
	config.ErrorClassifier = nil     // Disable error classifier to treat all errors as failures

	icb, err := NewInstrumentedCircuitBreaker(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Execute some operations
	icb.Execute(ctx, func() error { return nil })
	icb.Execute(ctx, func() error { return errors.New("error") })
	icb.Execute(ctx, func() error { return errors.New("error") })

	// Small delay to allow the async callback to complete
	time.Sleep(10 * time.Millisecond)

	stats := icb.GetStatistics()
	assert.Equal(t, "test-breaker", stats.Name)
	assert.Equal(t, StateOpen, stats.State)
	assert.Equal(t, uint64(1), stats.TotalStateChanges) // Closed -> Open
	assert.Equal(t, uint64(1), stats.TimesOpen)
}

func TestResilienceExecutor(t *testing.T) {
	t.Run("with retry only", func(t *testing.T) {
		retryPolicy := NewRetryPolicy().
			WithMaxRetries(2).
			WithInitialDelay(1 * time.Millisecond)

		config := &ResilienceConfig{
			RetryPolicy:          retryPolicy,
			EnableRetry:          true,
			EnableCircuitBreaker: false,
		}

		executor, err := NewResilienceExecutor(config)
		require.NoError(t, err)

		callCount := 0
		operation := func() error {
			callCount++
			if callCount < 3 {
				return NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
					Retryable(true).
					Build()
			}
			return nil
		}

		ctx := context.Background()
		err = executor.Execute(ctx, operation)

		assert.NoError(t, err)
		assert.Equal(t, 3, callCount)
	})

	t.Run("with circuit breaker only", func(t *testing.T) {
		cbConfig := NewCircuitBreakerConfig("test")
		cbConfig.FailureThreshold = 2

		config := &ResilienceConfig{
			CircuitBreakerConfig: cbConfig,
			EnableRetry:          false,
			EnableCircuitBreaker: true,
		}

		executor, err := NewResilienceExecutor(config)
		require.NoError(t, err)

		ctx := context.Background()

		// First two calls should fail and open circuit
		err = executor.Execute(ctx, func() error { return errors.New("error") })
		assert.Error(t, err)

		err = executor.Execute(ctx, func() error { return errors.New("error") })
		assert.Error(t, err)

		// Third call should be rejected by circuit breaker
		err = executor.Execute(ctx, func() error { return nil })
		assert.Error(t, err)

		var msgErr *BaseMessagingError
		assert.True(t, errors.As(err, &msgErr))
		assert.Equal(t, ErrorTypeResource, msgErr.Type)
	})

	t.Run("with both retry and circuit breaker", func(t *testing.T) {
		retryPolicy := NewRetryPolicy().
			WithMaxRetries(1).
			WithInitialDelay(1 * time.Millisecond)

		cbConfig := NewCircuitBreakerConfig("test")
		cbConfig.FailureThreshold = 3

		config := &ResilienceConfig{
			RetryPolicy:          retryPolicy,
			CircuitBreakerConfig: cbConfig,
			EnableRetry:          true,
			EnableCircuitBreaker: true,
		}

		executor, err := NewResilienceExecutor(config)
		require.NoError(t, err)

		callCount := 0
		operation := func() error {
			callCount++
			if callCount < 2 {
				return NewError(ErrorTypeConnection, ErrCodeConnectionTimeout).
					Retryable(true).
					Build()
			}
			return nil
		}

		ctx := context.Background()
		err = executor.Execute(ctx, operation)

		assert.NoError(t, err)
		assert.Equal(t, 2, callCount) // Original + 1 retry
	})

	t.Run("with no resilience mechanisms", func(t *testing.T) {
		config := &ResilienceConfig{
			EnableRetry:          false,
			EnableCircuitBreaker: false,
		}

		executor, err := NewResilienceExecutor(config)
		require.NoError(t, err)

		callCount := 0
		operation := func() error {
			callCount++
			return errors.New("error")
		}

		ctx := context.Background()
		err = executor.Execute(ctx, operation)

		assert.Error(t, err)
		assert.Equal(t, 1, callCount) // No retries
	})
}

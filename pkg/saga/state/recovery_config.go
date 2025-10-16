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

package state

import (
	"errors"
	"time"
)

// RecoveryConfig defines the configuration for the RecoveryManager.
// It controls how frequently recovery checks are performed, timeouts,
// and concurrency limits for recovery operations.
type RecoveryConfig struct {
	// CheckInterval is the interval between recovery checks.
	// Default: 30 seconds
	CheckInterval time.Duration `json:"check_interval" yaml:"check_interval"`

	// RecoveryTimeout is the maximum time allowed for a single recovery operation.
	// Default: 5 minutes
	RecoveryTimeout time.Duration `json:"recovery_timeout" yaml:"recovery_timeout"`

	// MaxConcurrentRecoveries is the maximum number of concurrent recovery operations.
	// This prevents overwhelming the system during mass recovery scenarios.
	// Default: 10
	MaxConcurrentRecoveries int `json:"max_concurrent_recoveries" yaml:"max_concurrent_recoveries"`

	// MinRecoveryAge is the minimum age of a Saga before it's considered for recovery.
	// This prevents premature recovery attempts on newly created Sagas.
	// Default: 1 minute
	MinRecoveryAge time.Duration `json:"min_recovery_age" yaml:"min_recovery_age"`

	// MaxRecoveryAttempts is the maximum number of times to attempt recovery for a single Saga.
	// After this limit is reached, the Saga is marked as failed.
	// Default: 3
	MaxRecoveryAttempts int `json:"max_recovery_attempts" yaml:"max_recovery_attempts"`

	// RecoveryBackoff is the backoff duration between recovery attempts for the same Saga.
	// Default: 1 minute
	RecoveryBackoff time.Duration `json:"recovery_backoff" yaml:"recovery_backoff"`

	// EnableAutoRecovery enables automatic recovery checks.
	// If false, recovery must be triggered manually.
	// Default: true
	EnableAutoRecovery bool `json:"enable_auto_recovery" yaml:"enable_auto_recovery"`

	// RecoveryBatchSize is the maximum number of Sagas to recover in a single batch.
	// Default: 100
	RecoveryBatchSize int `json:"recovery_batch_size" yaml:"recovery_batch_size"`

	// EnableMetrics enables recovery metrics collection.
	// Default: true
	EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics"`
}

// DefaultRecoveryConfig returns a RecoveryConfig with sensible default values.
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		CheckInterval:           30 * time.Second,
		RecoveryTimeout:         5 * time.Minute,
		MaxConcurrentRecoveries: 10,
		MinRecoveryAge:          1 * time.Minute,
		MaxRecoveryAttempts:     3,
		RecoveryBackoff:         1 * time.Minute,
		EnableAutoRecovery:      true,
		RecoveryBatchSize:       100,
		EnableMetrics:           true,
	}
}

// Validate checks if the RecoveryConfig is valid and sets defaults for missing values.
func (c *RecoveryConfig) Validate() error {
	if c == nil {
		return errors.New("recovery config is nil")
	}

	// Validate explicit negative or too-small values before applying defaults
	if c.CheckInterval != 0 && c.CheckInterval < time.Second {
		return errors.New("check interval must be at least 1 second")
	}

	if c.RecoveryTimeout != 0 && c.RecoveryTimeout < time.Second {
		return errors.New("recovery timeout must be at least 1 second")
	}

	if c.MinRecoveryAge < 0 {
		return errors.New("min recovery age cannot be negative")
	}

	if c.RecoveryBackoff < 0 {
		return errors.New("recovery backoff cannot be negative")
	}

	// Apply defaults if not set
	if c.CheckInterval == 0 {
		c.CheckInterval = 30 * time.Second
	}

	if c.RecoveryTimeout == 0 {
		c.RecoveryTimeout = 5 * time.Minute
	}

	if c.MaxConcurrentRecoveries == 0 {
		c.MaxConcurrentRecoveries = 10
	}

	if c.MinRecoveryAge == 0 {
		c.MinRecoveryAge = 1 * time.Minute
	}

	if c.MaxRecoveryAttempts == 0 {
		c.MaxRecoveryAttempts = 3
	}

	if c.RecoveryBackoff == 0 {
		c.RecoveryBackoff = 1 * time.Minute
	}

	if c.RecoveryBatchSize == 0 {
		c.RecoveryBatchSize = 100
	}

	return nil
}

// WithCheckInterval sets the check interval and returns the config for chaining.
func (c *RecoveryConfig) WithCheckInterval(interval time.Duration) *RecoveryConfig {
	c.CheckInterval = interval
	return c
}

// WithRecoveryTimeout sets the recovery timeout and returns the config for chaining.
func (c *RecoveryConfig) WithRecoveryTimeout(timeout time.Duration) *RecoveryConfig {
	c.RecoveryTimeout = timeout
	return c
}

// WithMaxConcurrentRecoveries sets the max concurrent recoveries and returns the config for chaining.
func (c *RecoveryConfig) WithMaxConcurrentRecoveries(max int) *RecoveryConfig {
	c.MaxConcurrentRecoveries = max
	return c
}

// WithMinRecoveryAge sets the minimum recovery age and returns the config for chaining.
func (c *RecoveryConfig) WithMinRecoveryAge(age time.Duration) *RecoveryConfig {
	c.MinRecoveryAge = age
	return c
}

// WithMaxRecoveryAttempts sets the max recovery attempts and returns the config for chaining.
func (c *RecoveryConfig) WithMaxRecoveryAttempts(max int) *RecoveryConfig {
	c.MaxRecoveryAttempts = max
	return c
}

// WithRecoveryBackoff sets the recovery backoff and returns the config for chaining.
func (c *RecoveryConfig) WithRecoveryBackoff(backoff time.Duration) *RecoveryConfig {
	c.RecoveryBackoff = backoff
	return c
}

// WithAutoRecovery enables or disables auto recovery and returns the config for chaining.
func (c *RecoveryConfig) WithAutoRecovery(enable bool) *RecoveryConfig {
	c.EnableAutoRecovery = enable
	return c
}

// WithRecoveryBatchSize sets the recovery batch size and returns the config for chaining.
func (c *RecoveryConfig) WithRecoveryBatchSize(size int) *RecoveryConfig {
	c.RecoveryBatchSize = size
	return c
}

// WithMetrics enables or disables metrics collection and returns the config for chaining.
func (c *RecoveryConfig) WithMetrics(enable bool) *RecoveryConfig {
	c.EnableMetrics = enable
	return c
}

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

package testing

import (
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/retry"
)

// ==========================
// TestConfig
// ==========================

// TestConfig provides configuration for Saga tests.
type TestConfig struct {
	// Timeouts
	DefaultTimeout      time.Duration
	StepTimeout         time.Duration
	CompensationTimeout time.Duration

	// Retry settings
	RetryEnabled           bool
	MaxRetries             int
	RetryDelay             time.Duration
	RetryBackoffMultiplier float64

	// Concurrency
	MaxConcurrentSagas int
	StepConcurrency    int

	// Storage settings
	StorageType      string
	StorageCleanup   bool
	StorageRetention time.Duration

	// Event settings
	EventBufferSize       int
	EventPublishTimeout   time.Duration
	EventSubscribeTimeout time.Duration

	// Logging
	EnableVerboseLogging bool
	LogLevel             string
	LogStepDetails       bool
	LogEventDetails      bool

	// Performance
	EnableMetrics   bool
	EnableTracing   bool
	EnableProfiling bool

	// Test behavior
	StopOnFirstFailure bool
	ContinueOnError    bool
	FailFast           bool

	// Cleanup
	AutoCleanup  bool
	CleanupDelay time.Duration
}

// DefaultTestConfig returns a test configuration with sensible defaults.
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		// Timeouts
		DefaultTimeout:      5 * time.Minute,
		StepTimeout:         30 * time.Second,
		CompensationTimeout: 1 * time.Minute,

		// Retry settings
		RetryEnabled:           true,
		MaxRetries:             3,
		RetryDelay:             100 * time.Millisecond,
		RetryBackoffMultiplier: 2.0,

		// Concurrency
		MaxConcurrentSagas: 10,
		StepConcurrency:    1,

		// Storage settings
		StorageType:      "memory",
		StorageCleanup:   true,
		StorageRetention: 1 * time.Hour,

		// Event settings
		EventBufferSize:       100,
		EventPublishTimeout:   5 * time.Second,
		EventSubscribeTimeout: 5 * time.Second,

		// Logging
		EnableVerboseLogging: false,
		LogLevel:             "info",
		LogStepDetails:       true,
		LogEventDetails:      false,

		// Performance
		EnableMetrics:   true,
		EnableTracing:   false,
		EnableProfiling: false,

		// Test behavior
		StopOnFirstFailure: false,
		ContinueOnError:    true,
		FailFast:           false,

		// Cleanup
		AutoCleanup:  true,
		CleanupDelay: 100 * time.Millisecond,
	}
}

// QuickTestConfig returns a configuration optimized for quick unit tests.
func QuickTestConfig() *TestConfig {
	config := DefaultTestConfig()
	config.DefaultTimeout = 10 * time.Second
	config.StepTimeout = 1 * time.Second
	config.CompensationTimeout = 5 * time.Second
	config.MaxRetries = 1
	config.RetryDelay = 10 * time.Millisecond
	config.EnableVerboseLogging = false
	config.EnableMetrics = false
	config.EnableTracing = false
	config.AutoCleanup = true
	return config
}

// IntegrationTestConfig returns a configuration for integration tests.
func IntegrationTestConfig() *TestConfig {
	config := DefaultTestConfig()
	config.DefaultTimeout = 10 * time.Minute
	config.StepTimeout = 1 * time.Minute
	config.CompensationTimeout = 2 * time.Minute
	config.MaxRetries = 5
	config.RetryDelay = 500 * time.Millisecond
	config.EnableVerboseLogging = true
	config.LogEventDetails = true
	config.EnableMetrics = true
	config.EnableTracing = true
	config.ContinueOnError = false
	return config
}

// PerformanceTestConfig returns a configuration for performance/load tests.
func PerformanceTestConfig() *TestConfig {
	config := DefaultTestConfig()
	config.MaxConcurrentSagas = 100
	config.EnableVerboseLogging = false
	config.LogStepDetails = false
	config.LogEventDetails = false
	config.EnableMetrics = true
	config.EnableTracing = false
	config.EnableProfiling = true
	config.AutoCleanup = false
	return config
}

// DebugTestConfig returns a configuration with maximum debugging information.
func DebugTestConfig() *TestConfig {
	config := DefaultTestConfig()
	config.EnableVerboseLogging = true
	config.LogLevel = "debug"
	config.LogStepDetails = true
	config.LogEventDetails = true
	config.EnableTracing = true
	config.StopOnFirstFailure = true
	config.FailFast = true
	return config
}

// ==========================
// TestConfigBuilder
// ==========================

// TestConfigBuilder provides a fluent API for building test configurations.
type TestConfigBuilder struct {
	config *TestConfig
}

// NewTestConfigBuilder creates a new test configuration builder.
func NewTestConfigBuilder() *TestConfigBuilder {
	return &TestConfigBuilder{
		config: DefaultTestConfig(),
	}
}

// WithTimeout sets the default saga timeout.
func (b *TestConfigBuilder) WithTimeout(timeout time.Duration) *TestConfigBuilder {
	b.config.DefaultTimeout = timeout
	return b
}

// WithStepTimeout sets the step timeout.
func (b *TestConfigBuilder) WithStepTimeout(timeout time.Duration) *TestConfigBuilder {
	b.config.StepTimeout = timeout
	return b
}

// WithCompensationTimeout sets the compensation timeout.
func (b *TestConfigBuilder) WithCompensationTimeout(timeout time.Duration) *TestConfigBuilder {
	b.config.CompensationTimeout = timeout
	return b
}

// WithRetry enables retry with the specified settings.
func (b *TestConfigBuilder) WithRetry(maxRetries int, delay time.Duration) *TestConfigBuilder {
	b.config.RetryEnabled = true
	b.config.MaxRetries = maxRetries
	b.config.RetryDelay = delay
	return b
}

// WithoutRetry disables retry.
func (b *TestConfigBuilder) WithoutRetry() *TestConfigBuilder {
	b.config.RetryEnabled = false
	return b
}

// WithConcurrency sets the maximum concurrent sagas.
func (b *TestConfigBuilder) WithConcurrency(maxConcurrent int) *TestConfigBuilder {
	b.config.MaxConcurrentSagas = maxConcurrent
	return b
}

// WithVerboseLogging enables verbose logging.
func (b *TestConfigBuilder) WithVerboseLogging() *TestConfigBuilder {
	b.config.EnableVerboseLogging = true
	b.config.LogStepDetails = true
	b.config.LogEventDetails = true
	return b
}

// WithMetrics enables metrics collection.
func (b *TestConfigBuilder) WithMetrics() *TestConfigBuilder {
	b.config.EnableMetrics = true
	return b
}

// WithTracing enables distributed tracing.
func (b *TestConfigBuilder) WithTracing() *TestConfigBuilder {
	b.config.EnableTracing = true
	return b
}

// WithProfiling enables profiling.
func (b *TestConfigBuilder) WithProfiling() *TestConfigBuilder {
	b.config.EnableProfiling = true
	return b
}

// WithAutoCleanup enables automatic cleanup after tests.
func (b *TestConfigBuilder) WithAutoCleanup(enabled bool) *TestConfigBuilder {
	b.config.AutoCleanup = enabled
	return b
}

// WithFailFast enables fail-fast behavior.
func (b *TestConfigBuilder) WithFailFast() *TestConfigBuilder {
	b.config.FailFast = true
	b.config.StopOnFirstFailure = true
	return b
}

// WithStorage sets the storage type.
func (b *TestConfigBuilder) WithStorage(storageType string) *TestConfigBuilder {
	b.config.StorageType = storageType
	return b
}

// WithEventBufferSize sets the event buffer size.
func (b *TestConfigBuilder) WithEventBufferSize(size int) *TestConfigBuilder {
	b.config.EventBufferSize = size
	return b
}

// Build returns the constructed test configuration.
func (b *TestConfigBuilder) Build() *TestConfig {
	return b.config
}

// ==========================
// Configuration Helpers
// ==========================

// GetRetryPolicy returns a retry policy based on the test configuration.
func (c *TestConfig) GetRetryPolicy() saga.RetryPolicy {
	if !c.RetryEnabled {
		// Return a policy with no retries
		return retry.NewExponentialBackoffPolicy(retry.NoRetryConfig(), 1.0, 0.0)
	}

	retryConfig := &retry.RetryConfig{
		MaxAttempts:  c.MaxRetries,
		InitialDelay: c.RetryDelay,
		MaxDelay:     c.RetryDelay * time.Duration(c.MaxRetries),
	}

	return retry.NewExponentialBackoffPolicy(retryConfig, c.RetryBackoffMultiplier, 0.1)
}

// Clone creates a copy of the test configuration.
func (c *TestConfig) Clone() *TestConfig {
	clone := *c
	return &clone
}

// Validate validates the test configuration.
func (c *TestConfig) Validate() error {
	if c.DefaultTimeout <= 0 {
		return NewValidationError("DefaultTimeout must be positive")
	}
	if c.StepTimeout <= 0 {
		return NewValidationError("StepTimeout must be positive")
	}
	if c.CompensationTimeout <= 0 {
		return NewValidationError("CompensationTimeout must be positive")
	}
	if c.MaxRetries < 0 {
		return NewValidationError("MaxRetries cannot be negative")
	}
	if c.MaxConcurrentSagas <= 0 {
		return NewValidationError("MaxConcurrentSagas must be positive")
	}
	if c.EventBufferSize < 0 {
		return NewValidationError("EventBufferSize cannot be negative")
	}
	return nil
}

// ==========================
// Validation Error
// ==========================

// ValidationError represents a configuration validation error.
type ValidationError struct {
	message string
}

// NewValidationError creates a new validation error.
func NewValidationError(message string) *ValidationError {
	return &ValidationError{message: message}
}

func (e *ValidationError) Error() string {
	return "configuration validation error: " + e.message
}

// ==========================
// Environment-based Configuration
// ==========================

// ConfigFromEnvironment loads test configuration from environment variables.
// This is useful for CI/CD environments where configuration can be controlled externally.
func ConfigFromEnvironment() *TestConfig {
	// For simplicity, we return a default config here
	// In a real implementation, you would read from environment variables
	return DefaultTestConfig()
}

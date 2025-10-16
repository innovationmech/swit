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
	"testing"
	"time"
)

func TestDefaultRecoveryConfig(t *testing.T) {
	config := DefaultRecoveryConfig()

	if config == nil {
		t.Fatal("DefaultRecoveryConfig() returned nil")
	}

	// Verify default values
	if config.CheckInterval != 30*time.Second {
		t.Errorf("expected CheckInterval = 30s, got %v", config.CheckInterval)
	}
	if config.RecoveryTimeout != 5*time.Minute {
		t.Errorf("expected RecoveryTimeout = 5m, got %v", config.RecoveryTimeout)
	}
	if config.MaxConcurrentRecoveries != 10 {
		t.Errorf("expected MaxConcurrentRecoveries = 10, got %d", config.MaxConcurrentRecoveries)
	}
	if config.MinRecoveryAge != 1*time.Minute {
		t.Errorf("expected MinRecoveryAge = 1m, got %v", config.MinRecoveryAge)
	}
	if config.MaxRecoveryAttempts != 3 {
		t.Errorf("expected MaxRecoveryAttempts = 3, got %d", config.MaxRecoveryAttempts)
	}
	if config.RecoveryBackoff != 1*time.Minute {
		t.Errorf("expected RecoveryBackoff = 1m, got %v", config.RecoveryBackoff)
	}
	if !config.EnableAutoRecovery {
		t.Error("expected EnableAutoRecovery = true, got false")
	}
	if config.RecoveryBatchSize != 100 {
		t.Errorf("expected RecoveryBatchSize = 100, got %d", config.RecoveryBatchSize)
	}
	if !config.EnableMetrics {
		t.Error("expected EnableMetrics = true, got false")
	}
}

func TestRecoveryConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *RecoveryConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "recovery config is nil",
		},
		{
			name:    "valid default config",
			config:  DefaultRecoveryConfig(),
			wantErr: false,
		},
		{
			name: "empty config gets defaults",
			config: &RecoveryConfig{
				EnableAutoRecovery: true,
			},
			wantErr: false,
		},
		{
			name: "check interval too small",
			config: &RecoveryConfig{
				CheckInterval: 500 * time.Millisecond,
			},
			wantErr: true,
			errMsg:  "check interval must be at least 1 second",
		},
		{
			name: "recovery timeout too small",
			config: &RecoveryConfig{
				CheckInterval:   30 * time.Second,
				RecoveryTimeout: 500 * time.Millisecond,
			},
			wantErr: true,
			errMsg:  "recovery timeout must be at least 1 second",
		},
		{
			name: "max concurrent recoveries uses default when zero",
			config: &RecoveryConfig{
				CheckInterval:           30 * time.Second,
				RecoveryTimeout:         5 * time.Minute,
				MaxConcurrentRecoveries: 0, // This will get the default value
			},
			wantErr: false,
		},
		{
			name: "negative min recovery age",
			config: &RecoveryConfig{
				CheckInterval:           30 * time.Second,
				RecoveryTimeout:         5 * time.Minute,
				MaxConcurrentRecoveries: 10,
				MinRecoveryAge:          -1 * time.Minute,
			},
			wantErr: true,
			errMsg:  "min recovery age cannot be negative",
		},
		{
			name: "max recovery attempts uses default when zero",
			config: &RecoveryConfig{
				CheckInterval:           30 * time.Second,
				RecoveryTimeout:         5 * time.Minute,
				MaxConcurrentRecoveries: 10,
				MinRecoveryAge:          1 * time.Minute,
				MaxRecoveryAttempts:     0, // This will get the default value
			},
			wantErr: false,
		},
		{
			name: "negative recovery backoff",
			config: &RecoveryConfig{
				CheckInterval:           30 * time.Second,
				RecoveryTimeout:         5 * time.Minute,
				MaxConcurrentRecoveries: 10,
				MinRecoveryAge:          1 * time.Minute,
				MaxRecoveryAttempts:     3,
				RecoveryBackoff:         -1 * time.Minute,
			},
			wantErr: true,
			errMsg:  "recovery backoff cannot be negative",
		},
		{
			name: "recovery batch size uses default when zero",
			config: &RecoveryConfig{
				CheckInterval:           30 * time.Second,
				RecoveryTimeout:         5 * time.Minute,
				MaxConcurrentRecoveries: 10,
				MinRecoveryAge:          1 * time.Minute,
				MaxRecoveryAttempts:     3,
				RecoveryBackoff:         1 * time.Minute,
				RecoveryBatchSize:       0, // This will get the default value
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify defaults were applied
			if tt.config.CheckInterval == 0 {
				t.Error("CheckInterval should be set to default")
			}
		})
	}
}

func TestRecoveryConfigBuilderMethods(t *testing.T) {
	config := DefaultRecoveryConfig()

	// Test WithCheckInterval
	interval := 60 * time.Second
	result := config.WithCheckInterval(interval)
	if result != config {
		t.Error("WithCheckInterval should return the same config for chaining")
	}
	if config.CheckInterval != interval {
		t.Errorf("expected CheckInterval = %v, got %v", interval, config.CheckInterval)
	}

	// Test WithRecoveryTimeout
	timeout := 10 * time.Minute
	result = config.WithRecoveryTimeout(timeout)
	if result != config {
		t.Error("WithRecoveryTimeout should return the same config for chaining")
	}
	if config.RecoveryTimeout != timeout {
		t.Errorf("expected RecoveryTimeout = %v, got %v", timeout, config.RecoveryTimeout)
	}

	// Test WithMaxConcurrentRecoveries
	maxRecoveries := 20
	result = config.WithMaxConcurrentRecoveries(maxRecoveries)
	if result != config {
		t.Error("WithMaxConcurrentRecoveries should return the same config for chaining")
	}
	if config.MaxConcurrentRecoveries != maxRecoveries {
		t.Errorf("expected MaxConcurrentRecoveries = %d, got %d", maxRecoveries, config.MaxConcurrentRecoveries)
	}

	// Test WithMinRecoveryAge
	minAge := 2 * time.Minute
	result = config.WithMinRecoveryAge(minAge)
	if result != config {
		t.Error("WithMinRecoveryAge should return the same config for chaining")
	}
	if config.MinRecoveryAge != minAge {
		t.Errorf("expected MinRecoveryAge = %v, got %v", minAge, config.MinRecoveryAge)
	}

	// Test WithMaxRecoveryAttempts
	maxAttempts := 5
	result = config.WithMaxRecoveryAttempts(maxAttempts)
	if result != config {
		t.Error("WithMaxRecoveryAttempts should return the same config for chaining")
	}
	if config.MaxRecoveryAttempts != maxAttempts {
		t.Errorf("expected MaxRecoveryAttempts = %d, got %d", maxAttempts, config.MaxRecoveryAttempts)
	}

	// Test WithRecoveryBackoff
	backoff := 2 * time.Minute
	result = config.WithRecoveryBackoff(backoff)
	if result != config {
		t.Error("WithRecoveryBackoff should return the same config for chaining")
	}
	if config.RecoveryBackoff != backoff {
		t.Errorf("expected RecoveryBackoff = %v, got %v", backoff, config.RecoveryBackoff)
	}

	// Test WithAutoRecovery
	result = config.WithAutoRecovery(false)
	if result != config {
		t.Error("WithAutoRecovery should return the same config for chaining")
	}
	if config.EnableAutoRecovery {
		t.Error("expected EnableAutoRecovery = false, got true")
	}

	// Test WithRecoveryBatchSize
	batchSize := 200
	result = config.WithRecoveryBatchSize(batchSize)
	if result != config {
		t.Error("WithRecoveryBatchSize should return the same config for chaining")
	}
	if config.RecoveryBatchSize != batchSize {
		t.Errorf("expected RecoveryBatchSize = %d, got %d", batchSize, config.RecoveryBatchSize)
	}

	// Test WithMetrics
	result = config.WithMetrics(false)
	if result != config {
		t.Error("WithMetrics should return the same config for chaining")
	}
	if config.EnableMetrics {
		t.Error("expected EnableMetrics = false, got true")
	}
}

func TestRecoveryConfigChaining(t *testing.T) {
	config := DefaultRecoveryConfig().
		WithCheckInterval(45 * time.Second).
		WithRecoveryTimeout(8 * time.Minute).
		WithMaxConcurrentRecoveries(15).
		WithMinRecoveryAge(2 * time.Minute).
		WithMaxRecoveryAttempts(5).
		WithRecoveryBackoff(90 * time.Second).
		WithAutoRecovery(false).
		WithRecoveryBatchSize(150).
		WithMetrics(false)

	// Verify all values were set correctly
	if config.CheckInterval != 45*time.Second {
		t.Errorf("expected CheckInterval = 45s, got %v", config.CheckInterval)
	}
	if config.RecoveryTimeout != 8*time.Minute {
		t.Errorf("expected RecoveryTimeout = 8m, got %v", config.RecoveryTimeout)
	}
	if config.MaxConcurrentRecoveries != 15 {
		t.Errorf("expected MaxConcurrentRecoveries = 15, got %d", config.MaxConcurrentRecoveries)
	}
	if config.MinRecoveryAge != 2*time.Minute {
		t.Errorf("expected MinRecoveryAge = 2m, got %v", config.MinRecoveryAge)
	}
	if config.MaxRecoveryAttempts != 5 {
		t.Errorf("expected MaxRecoveryAttempts = 5, got %d", config.MaxRecoveryAttempts)
	}
	if config.RecoveryBackoff != 90*time.Second {
		t.Errorf("expected RecoveryBackoff = 90s, got %v", config.RecoveryBackoff)
	}
	if config.EnableAutoRecovery {
		t.Error("expected EnableAutoRecovery = false, got true")
	}
	if config.RecoveryBatchSize != 150 {
		t.Errorf("expected RecoveryBatchSize = 150, got %d", config.RecoveryBatchSize)
	}
	if config.EnableMetrics {
		t.Error("expected EnableMetrics = false, got true")
	}

	// Verify it's still valid
	if err := config.Validate(); err != nil {
		t.Errorf("chained config should be valid, got error: %v", err)
	}
}

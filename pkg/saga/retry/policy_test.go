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

func TestRetryConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *RetryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 100 * time.Millisecond,
				MaxDelay:     1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid max attempts",
			config: &RetryConfig{
				MaxAttempts:  0,
				InitialDelay: 100 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "negative initial delay",
			config: &RetryConfig{
				MaxAttempts:  3,
				InitialDelay: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "max delay less than initial delay",
			config: &RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 1 * time.Second,
				MaxDelay:     100 * time.Millisecond,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts = 3, got %d", config.MaxAttempts)
	}
	if config.InitialDelay != 100*time.Millisecond {
		t.Errorf("expected InitialDelay = 100ms, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay = 30s, got %v", config.MaxDelay)
	}
	if config.Timeout != 5*time.Minute {
		t.Errorf("expected Timeout = 5m, got %v", config.Timeout)
	}

	if err := config.Validate(); err != nil {
		t.Errorf("DefaultRetryConfig should be valid, got error: %v", err)
	}
}

func TestNoRetryConfig(t *testing.T) {
	config := NoRetryConfig()

	if config.MaxAttempts != 1 {
		t.Errorf("expected MaxAttempts = 1, got %d", config.MaxAttempts)
	}
	if config.InitialDelay != 0 {
		t.Errorf("expected InitialDelay = 0, got %v", config.InitialDelay)
	}

	if err := config.Validate(); err != nil {
		t.Errorf("NoRetryConfig should be valid, got error: %v", err)
	}
}

func TestRetryConfig_IsRetryableError(t *testing.T) {
	errNetwork := errors.New("network error")
	errValidation := errors.New("validation error")
	errUnknown := errors.New("unknown error")

	tests := []struct {
		name   string
		config *RetryConfig
		err    error
		want   bool
	}{
		{
			name: "nil error",
			config: &RetryConfig{
				MaxAttempts: 3,
			},
			err:  nil,
			want: false,
		},
		{
			name: "no retryable errors specified - all retryable",
			config: &RetryConfig{
				MaxAttempts: 3,
			},
			err:  errUnknown,
			want: true,
		},
		{
			name: "error in retryable list",
			config: &RetryConfig{
				MaxAttempts:     3,
				RetryableErrors: []error{errNetwork},
			},
			err:  errNetwork,
			want: true,
		},
		{
			name: "error not in retryable list",
			config: &RetryConfig{
				MaxAttempts:     3,
				RetryableErrors: []error{errNetwork},
			},
			err:  errValidation,
			want: false,
		},
		{
			name: "error in non-retryable list",
			config: &RetryConfig{
				MaxAttempts:        3,
				NonRetryableErrors: []error{errValidation},
			},
			err:  errValidation,
			want: false,
		},
		{
			name: "non-retryable takes precedence",
			config: &RetryConfig{
				MaxAttempts:        3,
				RetryableErrors:    []error{errNetwork},
				NonRetryableErrors: []error{errNetwork},
			},
			err:  errNetwork,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsRetryableError(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryResult(t *testing.T) {
	result := &RetryResult{
		Success:             true,
		Result:              "test result",
		Error:               nil,
		Attempts:            2,
		TotalDuration:       200 * time.Millisecond,
		LastAttemptDuration: 50 * time.Millisecond,
	}

	if !result.Success {
		t.Error("expected Success = true")
	}
	if result.Result != "test result" {
		t.Errorf("expected Result = 'test result', got %v", result.Result)
	}
	if result.Error != nil {
		t.Errorf("expected Error = nil, got %v", result.Error)
	}
	if result.Attempts != 2 {
		t.Errorf("expected Attempts = 2, got %d", result.Attempts)
	}
}

func TestRetryContext(t *testing.T) {
	startTime := time.Now()
	ctx := &RetryContext{
		Attempt:         2,
		Error:           errors.New("test error"),
		StartTime:       startTime,
		LastAttemptTime: startTime.Add(100 * time.Millisecond),
		ElapsedTime:     100 * time.Millisecond,
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	if ctx.Attempt != 2 {
		t.Errorf("expected Attempt = 2, got %d", ctx.Attempt)
	}
	if ctx.Error == nil {
		t.Error("expected Error to be set")
	}
	if ctx.Metadata["key"] != "value" {
		t.Errorf("expected Metadata[key] = 'value', got %v", ctx.Metadata["key"])
	}
}


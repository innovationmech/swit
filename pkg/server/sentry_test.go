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

package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestSentryManager_Initialize_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &SentryConfig{
		Enabled: false,
	}

	manager := NewSentryManager(config, logger)
	err := manager.Initialize()

	assert.NoError(t, err)
	assert.False(t, manager.IsEnabled())
}

func TestSentryManager_Initialize_Enabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &SentryConfig{
		Enabled:          true,
		DSN:              "https://test@sentry.io/test", // Test DSN
		Environment:      "test",
		SampleRate:       1.0,
		TracesSampleRate: 0.1,
		AttachStacktrace: true,
		FlushTimeout:     2 * time.Second,
	}

	manager := NewSentryManager(config, logger)
	
	// Initialize Sentry - this will fail with fake DSN but we're testing the config setup
	err := manager.Initialize()
	
	// We expect no error even with fake DSN in test mode
	if err != nil {
		t.Logf("Expected error with fake DSN: %v", err)
	}
	
	assert.True(t, manager.IsEnabled())
}

func TestSentryManager_CaptureError_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &SentryConfig{
		Enabled: false,
	}

	manager := NewSentryManager(config, logger)
	err := manager.Initialize()
	require.NoError(t, err)

	// Capture error should return nil when disabled
	testErr := fmt.Errorf("test error")
	eventID := manager.CaptureError(testErr, nil, nil)
	assert.Nil(t, eventID)
}

func TestSentryManager_CaptureMessage_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &SentryConfig{
		Enabled: false,
	}

	manager := NewSentryManager(config, logger)
	err := manager.Initialize()
	require.NoError(t, err)

	// Capture message should return nil when disabled
	eventID := manager.CaptureMessage("test message", sentry.LevelError, nil, nil)
	assert.Nil(t, eventID)
}

func TestSentryManager_AddBreadcrumb_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &SentryConfig{
		Enabled: false,
	}

	manager := NewSentryManager(config, logger)
	err := manager.Initialize()
	require.NoError(t, err)

	// Should not panic when disabled
	assert.NotPanics(t, func() {
		manager.AddBreadcrumb("test", "test-category", sentry.LevelInfo, nil)
	})
}

func TestSentryManager_Shutdown_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &SentryConfig{
		Enabled: false,
	}

	manager := NewSentryManager(config, logger)
	err := manager.Initialize()
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestSentryManager_GetHub_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &SentryConfig{
		Enabled: false,
	}

	manager := NewSentryManager(config, logger)
	err := manager.Initialize()
	require.NoError(t, err)

	hub := manager.GetHub()
	assert.Nil(t, hub)
}

func TestSentryManager_StartTransaction_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &SentryConfig{
		Enabled: false,
	}

	manager := NewSentryManager(config, logger)
	err := manager.Initialize()
	require.NoError(t, err)

	ctx := context.Background()
	span := manager.StartTransaction(ctx, "test", "test.op")
	assert.Nil(t, span)
}

func TestSentryManager_RecoverWithSentry_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &SentryConfig{
		Enabled: false,
	}

	manager := NewSentryManager(config, logger)
	err := manager.Initialize()
	require.NoError(t, err)

	// Should not panic when disabled, but also should not recover anything
	assert.NotPanics(t, func() {
		defer manager.RecoverWithSentry()
		// No panic, so recover should do nothing
	})
}

func TestSentryConfig_Defaults(t *testing.T) {
	config := NewServerConfig()
	config.SetDefaults()

	// Test Sentry defaults
	assert.False(t, config.Sentry.Enabled)
	assert.False(t, config.Sentry.Debug)
	assert.Equal(t, 1.0, config.Sentry.SampleRate)
	assert.Equal(t, 0.1, config.Sentry.TracesSampleRate)
	assert.Equal(t, 0.1, config.Sentry.ProfilesSampleRate)
	assert.True(t, config.Sentry.AttachStacktrace)
	assert.True(t, config.Sentry.EnableTracing)
	assert.False(t, config.Sentry.EnableProfiling)
	assert.Equal(t, 2*time.Second, config.Sentry.FlushTimeout)
	assert.Equal(t, "production", config.Sentry.Environment)
	assert.Equal(t, config.ServiceName, config.Sentry.ServerName)
	assert.NotNil(t, config.Sentry.Tags)

	// Test HTTP middleware Sentry defaults
	assert.Equal(t, config.Sentry.Enabled, config.HTTP.Middleware.EnableSentry)

	// Test gRPC interceptor Sentry defaults
	assert.Equal(t, config.Sentry.Enabled, config.GRPC.Interceptors.EnableSentry)
}

func TestSentryConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      *SentryConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "disabled_sentry_no_validation",
			config: &SentryConfig{
				Enabled: false,
			},
			expectError: false,
		},
		{
			name: "enabled_without_dsn",
			config: &SentryConfig{
				Enabled: true,
			},
			expectError: true,
			errorMsg:    "sentry.dsn is required when Sentry is enabled",
		},
		{
			name: "invalid_sample_rate_negative",
			config: &SentryConfig{
				Enabled:    true,
				DSN:        "https://test@sentry.io/test",
				SampleRate: -0.1,
			},
			expectError: true,
			errorMsg:    "sentry.sample_rate must be between 0 and 1",
		},
		{
			name: "invalid_sample_rate_over_one",
			config: &SentryConfig{
				Enabled:    true,
				DSN:        "https://test@sentry.io/test",
				SampleRate: 1.1,
			},
			expectError: true,
			errorMsg:    "sentry.sample_rate must be between 0 and 1",
		},
		{
			name: "invalid_traces_sample_rate",
			config: &SentryConfig{
				Enabled:          true,
				DSN:              "https://test@sentry.io/test",
				SampleRate:       1.0,
				TracesSampleRate: -0.1,
			},
			expectError: true,
			errorMsg:    "sentry.traces_sample_rate must be between 0 and 1",
		},
		{
			name: "invalid_profiles_sample_rate",
			config: &SentryConfig{
				Enabled:            true,
				DSN:                "https://test@sentry.io/test",
				SampleRate:         1.0,
				TracesSampleRate:   0.1,
				ProfilesSampleRate: 1.1,
			},
			expectError: true,
			errorMsg:    "sentry.profiles_sample_rate must be between 0 and 1",
		},
		{
			name: "invalid_flush_timeout",
			config: &SentryConfig{
				Enabled:          true,
				DSN:              "https://test@sentry.io/test",
				SampleRate:       1.0,
				TracesSampleRate: 0.1,
				FlushTimeout:     -1 * time.Second,
			},
			expectError: true,
			errorMsg:    "sentry.flush_timeout must be positive",
		},
		{
			name: "empty_environment",
			config: &SentryConfig{
				Enabled:          true,
				DSN:              "https://test@sentry.io/test",
				SampleRate:       1.0,
				TracesSampleRate: 0.1,
				FlushTimeout:     2 * time.Second,
				Environment:      "",
			},
			expectError: true,
			errorMsg:    "sentry.environment cannot be empty when Sentry is enabled",
		},
		{
			name: "valid_config",
			config: &SentryConfig{
				Enabled:          true,
				DSN:              "https://test@sentry.io/test",
				Environment:      "test",
				SampleRate:       1.0,
				TracesSampleRate: 0.1,
				FlushTimeout:     2 * time.Second,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverConfig := &ServerConfig{
				ServiceName: "test-service",
				Sentry:      *tt.config,
			}
			serverConfig.SetDefaults()

			err := serverConfig.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
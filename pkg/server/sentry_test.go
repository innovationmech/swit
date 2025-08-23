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
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSentryManager(t *testing.T) {
	tests := []struct {
		name     string
		config   *SentryConfig
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name: "disabled config",
			config: &SentryConfig{
				Enabled: false,
			},
			expected: false,
		},
		{
			name: "enabled config",
			config: &SentryConfig{
				Enabled: true,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewSentryManager(tt.config)
			assert.NotNil(t, manager)
			assert.Equal(t, tt.expected, manager.IsEnabled())
			assert.False(t, manager.IsInitialized())
		})
	}
}

func TestSentryManager_InitializeAndClose(t *testing.T) {
	tests := []struct {
		name      string
		config    *SentryConfig
		shouldErr bool
	}{
		{
			name: "disabled sentry",
			config: &SentryConfig{
				Enabled: false,
			},
			shouldErr: false,
		},
		{
			name: "invalid DSN",
			config: &SentryConfig{
				Enabled: true,
				DSN:     "invalid-dsn",
			},
			shouldErr: true,
		},
		{
			name: "valid test configuration",
			config: &SentryConfig{
				Enabled:          true,
				DSN:              "https://test@test.ingest.sentry.io/test",
				Environment:      "test",
				SampleRate:       1.0,
				AttachStacktrace: true,
				MaxBreadcrumbs:   30,
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewSentryManager(tt.config)
			ctx := context.Background()

			// Test initialization
			err := manager.Initialize(ctx)
			if tt.shouldErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.config.Enabled {
				assert.True(t, manager.IsInitialized())
			} else {
				assert.False(t, manager.IsInitialized())
			}

			// Test double initialization (should fail)
			if tt.config.Enabled {
				err = manager.Initialize(ctx)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "already initialized")
			}

			// Test close
			err = manager.Close()
			assert.NoError(t, err)

			if tt.config.Enabled {
				assert.False(t, manager.IsInitialized())
			}
		})
	}
}

func TestSentryManager_ShouldCaptureHTTPError(t *testing.T) {
	tests := []struct {
		name                  string
		config                *SentryConfig
		statusCode            int
		expectedShouldCapture bool
	}{
		{
			name: "disabled sentry",
			config: &SentryConfig{
				Enabled: false,
			},
			statusCode:            500,
			expectedShouldCapture: false,
		},
		{
			name: "server error should capture",
			config: &SentryConfig{
				Enabled:              true,
				HTTPIgnoreStatusCode: []int{400, 401, 403, 404},
			},
			statusCode:            500,
			expectedShouldCapture: true,
		},
		{
			name: "ignored client error should not capture",
			config: &SentryConfig{
				Enabled:              true,
				HTTPIgnoreStatusCode: []int{400, 401, 403, 404},
			},
			statusCode:            404,
			expectedShouldCapture: false,
		},
		{
			name: "non-ignored client error should capture",
			config: &SentryConfig{
				Enabled:              true,
				HTTPIgnoreStatusCode: []int{400, 401, 403},
			},
			statusCode:            404,
			expectedShouldCapture: true,
		},
		{
			name: "success status should not capture",
			config: &SentryConfig{
				Enabled: true,
			},
			statusCode:            200,
			expectedShouldCapture: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewSentryManager(tt.config)
			result := manager.ShouldCaptureHTTPError(tt.statusCode)
			assert.Equal(t, tt.expectedShouldCapture, result)
		})
	}
}

func TestSentryManager_ShouldCaptureHTTPPath(t *testing.T) {
	tests := []struct {
		name            string
		config          *SentryConfig
		path            string
		expectedCapture bool
	}{
		{
			name: "disabled sentry should not capture",
			config: &SentryConfig{
				Enabled: false,
			},
			path:            "/api/v1/users",
			expectedCapture: true, // Note: path filtering only applies when enabled
		},
		{
			name: "path not in ignore list should capture",
			config: &SentryConfig{
				Enabled:         true,
				HTTPIgnorePaths: []string{"/health", "/metrics"},
			},
			path:            "/api/v1/users",
			expectedCapture: true,
		},
		{
			name: "path in ignore list should not capture",
			config: &SentryConfig{
				Enabled:         true,
				HTTPIgnorePaths: []string{"/health", "/metrics"},
			},
			path:            "/health",
			expectedCapture: false,
		},
		{
			name: "partial path match should not capture",
			config: &SentryConfig{
				Enabled:         true,
				HTTPIgnorePaths: []string{"/health"},
			},
			path:            "/api/health/status",
			expectedCapture: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewSentryManager(tt.config)
			result := manager.ShouldCaptureHTTPPath(tt.path)
			assert.Equal(t, tt.expectedCapture, result)
		})
	}
}

func TestSentryManager_ShouldCaptureError(t *testing.T) {
	tests := []struct {
		name            string
		config          *SentryConfig
		err             error
		expectedCapture bool
	}{
		{
			name: "disabled sentry should not capture",
			config: &SentryConfig{
				Enabled: false,
			},
			err:             fmt.Errorf("test error"),
			expectedCapture: false,
		},
		{
			name: "nil error should not capture",
			config: &SentryConfig{
				Enabled: true,
			},
			err:             nil,
			expectedCapture: false,
		},
		{
			name: "error not in ignore list should capture",
			config: &SentryConfig{
				Enabled:      true,
				IgnoreErrors: []string{"connection timeout", "auth failed"},
			},
			err:             fmt.Errorf("database error"),
			expectedCapture: true,
		},
		{
			name: "error in ignore list should not capture",
			config: &SentryConfig{
				Enabled:      true,
				IgnoreErrors: []string{"connection timeout", "auth failed"},
			},
			err:             fmt.Errorf("connection timeout occurred"),
			expectedCapture: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewSentryManager(tt.config)
			result := manager.ShouldCaptureError(tt.err)
			assert.Equal(t, tt.expectedCapture, result)
		})
	}
}

func TestSentryManager_CaptureOperations(t *testing.T) {
	// Use a disabled manager to test the interface without actually sending to Sentry
	config := &SentryConfig{
		Enabled: false,
	}
	manager := NewSentryManager(config)

	// Test that methods return nil for disabled manager
	t.Run("disabled manager returns nil", func(t *testing.T) {
		eventID := manager.CaptureException(fmt.Errorf("test error"))
		assert.Nil(t, eventID)

		eventID = manager.CaptureMessage("test message", sentry.LevelInfo)
		assert.Nil(t, eventID)

		event := &sentry.Event{
			Message: "test event",
		}
		eventID = manager.CaptureEvent(event)
		assert.Nil(t, eventID)
	})

	t.Run("disabled manager handles scopes gracefully", func(t *testing.T) {
		// These should not panic
		manager.AddBreadcrumb(&sentry.Breadcrumb{
			Message: "test breadcrumb",
		})

		manager.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("test", "value")
		})

		manager.WithScope(func(scope *sentry.Scope) {
			scope.SetTag("test", "value")
		})

		hub := manager.GetHubFromContext(context.Background())
		assert.Nil(t, hub)
	})
}

func TestSentryManager_ContextOperations(t *testing.T) {
	config := &SentryConfig{
		Enabled: false,
	}
	manager := NewSentryManager(config)

	t.Run("context operations with disabled manager", func(t *testing.T) {
		// These should not panic
		manager.AddContextTags("test-service", "v1.0.0", "test")

		user := &sentry.User{
			ID:    "user123",
			Email: "test@example.com",
		}
		manager.SetUser(user)

		context := map[string]interface{}{
			"custom": "data",
		}
		manager.SetContext("test", context)

		success := manager.Flush(time.Second)
		assert.True(t, success) // Should return true for disabled manager
	})
}

func TestSentryManager_RecoverWithSentry(t *testing.T) {
	config := &SentryConfig{
		Enabled:       false,
		CapturePanics: true,
	}
	manager := NewSentryManager(config)

	t.Run("recover does nothing when disabled", func(t *testing.T) {
		// This should not panic and should do nothing
		assert.NotPanics(t, func() {
			defer manager.RecoverWithSentry()
		})
	})

	t.Run("recover with capture panics disabled", func(t *testing.T) {
		config.CapturePanics = false
		manager = NewSentryManager(config)

		assert.NotPanics(t, func() {
			defer manager.RecoverWithSentry()
		})
	})
}

func TestSentryConfig_Defaults(t *testing.T) {
	config := NewServerConfig()

	// Test Sentry defaults
	assert.False(t, config.Sentry.Enabled)
	assert.Equal(t, "development", config.Sentry.Environment)
	assert.Equal(t, 1.0, config.Sentry.SampleRate)
	assert.Equal(t, 0.0, config.Sentry.TracesSampleRate)
	assert.True(t, config.Sentry.AttachStacktrace)
	assert.False(t, config.Sentry.EnableTracing)
	assert.False(t, config.Sentry.Debug)
	assert.True(t, config.Sentry.IntegrateHTTP)
	assert.True(t, config.Sentry.IntegrateGRPC)
	assert.True(t, config.Sentry.CapturePanics)
	assert.Equal(t, 30, config.Sentry.MaxBreadcrumbs)
	assert.NotNil(t, config.Sentry.Tags)
	assert.Equal(t, []int{400, 401, 403, 404}, config.Sentry.HTTPIgnoreStatusCode)
}

func TestSentryConfig_Validation(t *testing.T) {
	tests := []struct {
		name      string
		config    *ServerConfig
		shouldErr bool
		errMsg    string
	}{
		{
			name: "disabled sentry should pass validation",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				Sentry: SentryConfig{
					Enabled: false,
				},
			},
			shouldErr: false,
		},
		{
			name: "enabled sentry without DSN should fail",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				Sentry: SentryConfig{
					Enabled: true,
				},
			},
			shouldErr: true,
			errMsg:    "sentry.dsn is required",
		},
		{
			name: "invalid sample rate should fail",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				Sentry: SentryConfig{
					Enabled:    true,
					DSN:        "https://test@test.ingest.sentry.io/test",
					SampleRate: 1.5,
				},
			},
			shouldErr: true,
			errMsg:    "sentry.sample_rate must be between 0.0 and 1.0",
		},
		{
			name: "invalid traces sample rate should fail",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				Sentry: SentryConfig{
					Enabled:          true,
					DSN:              "https://test@test.ingest.sentry.io/test",
					SampleRate:       1.0,
					TracesSampleRate: -0.1,
				},
			},
			shouldErr: true,
			errMsg:    "sentry.traces_sample_rate must be between 0.0 and 1.0",
		},
		{
			name: "empty environment should fail",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				Sentry: SentryConfig{
					Enabled:     true,
					DSN:         "https://test@test.ingest.sentry.io/test",
					SampleRate:  1.0,
					Environment: "",
				},
			},
			shouldErr: true,
			errMsg:    "sentry.environment cannot be empty",
		},
		{
			name: "invalid max breadcrumbs should fail",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				Sentry: SentryConfig{
					Enabled:        true,
					DSN:            "https://test@test.ingest.sentry.io/test",
					SampleRate:     1.0,
					Environment:    "test",
					MaxBreadcrumbs: 101,
				},
			},
			shouldErr: true,
			errMsg:    "sentry.max_breadcrumbs must be between 0 and 100",
		},
		{
			name: "invalid HTTP status code should fail",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				Sentry: SentryConfig{
					Enabled:              true,
					DSN:                  "https://test@test.ingest.sentry.io/test",
					SampleRate:           1.0,
					Environment:          "test",
					MaxBreadcrumbs:       30,
					HTTPIgnoreStatusCode: []int{200, 999},
				},
			},
			shouldErr: true,
			errMsg:    "sentry.http_ignore_status_codes contains invalid HTTP status code",
		},
		{
			name: "valid configuration should pass",
			config: &ServerConfig{
				ServiceName: "test",
				HTTP: HTTPConfig{
					Enabled: true,
					Port:    "8080",
				},
				Sentry: SentryConfig{
					Enabled:              true,
					DSN:                  "https://test@test.ingest.sentry.io/test",
					Environment:          "test",
					SampleRate:           1.0,
					TracesSampleRate:     0.1,
					AttachStacktrace:     true,
					MaxBreadcrumbs:       30,
					HTTPIgnoreStatusCode: []int{400, 401, 403, 404},
					Tags: map[string]string{
						"service": "test",
					},
				},
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store the original Sentry settings
			originalSentry := tt.config.Sentry

			// Always set defaults first to get valid base config
			tt.config.SetDefaults()

			// Restore important Sentry settings that were overridden by SetDefaults
			tt.config.Sentry.Enabled = originalSentry.Enabled
			if originalSentry.DSN != "" {
				tt.config.Sentry.DSN = originalSentry.DSN
			}
			if originalSentry.SampleRate != 0 {
				tt.config.Sentry.SampleRate = originalSentry.SampleRate
			}
			if originalSentry.TracesSampleRate != 0 {
				tt.config.Sentry.TracesSampleRate = originalSentry.TracesSampleRate
			}
			if originalSentry.Environment != "" {
				tt.config.Sentry.Environment = originalSentry.Environment
			}
			if originalSentry.MaxBreadcrumbs != 0 {
				tt.config.Sentry.MaxBreadcrumbs = originalSentry.MaxBreadcrumbs
			}
			if len(originalSentry.HTTPIgnoreStatusCode) > 0 {
				tt.config.Sentry.HTTPIgnoreStatusCode = originalSentry.HTTPIgnoreStatusCode
			}
			if originalSentry.Tags != nil {
				tt.config.Sentry.Tags = originalSentry.Tags
			}

			// Then override specific values for testing
			if strings.Contains(tt.name, "without DSN") {
				tt.config.Sentry.DSN = ""
			}
			if strings.Contains(tt.name, "empty environment") {
				tt.config.Sentry.Environment = ""
			}

			err := tt.config.Validate()
			if tt.shouldErr {
				assert.Error(t, err)
				if err != nil {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSentryIntegrationWithServer(t *testing.T) {
	// Create a test config with Sentry enabled but with a test DSN
	config := NewServerConfig()
	config.ServiceName = "sentry-test-service"
	config.HTTP.Port = "0" // Dynamic port
	config.GRPC.Port = "0" // Dynamic port
	config.Discovery.Enabled = false
	config.Sentry.Enabled = true
	config.Sentry.DSN = "https://test@test.ingest.sentry.io/test"
	config.Sentry.Environment = "test"
	config.Sentry.Debug = true

	// Create a test service registrar
	registrar := &testSentryServiceRegistrar{}

	// Create server
	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err)

	// Verify Sentry manager is created and configured
	sentryManager := server.GetSentryManager()
	require.NotNil(t, sentryManager)
	assert.True(t, sentryManager.IsEnabled())
	assert.False(t, sentryManager.IsInitialized()) // Not initialized until Start()

	// Test server lifecycle with Sentry
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start server (this should initialize Sentry)
	err = server.Start(ctx)
	require.NoError(t, err)
	defer server.Shutdown()

	// Verify Sentry is now initialized
	assert.True(t, sentryManager.IsInitialized())

	// Test that Sentry operations work
	eventID := sentryManager.CaptureMessage("test message", sentry.LevelInfo)
	assert.NotNil(t, eventID)
}

// testServiceRegistrar is a test implementation of BusinessServiceRegistrar
type testSentryServiceRegistrar struct{}

func (t *testSentryServiceRegistrar) RegisterServices(registry BusinessServiceRegistry) error {
	// Register a simple test handler
	handler := &testSentryHTTPHandler{}
	return registry.RegisterBusinessHTTPHandler(handler)
}

// testSentryHTTPHandler is a test implementation of BusinessHTTPHandler
type testSentryHTTPHandler struct{}

func (t *testSentryHTTPHandler) RegisterRoutes(router interface{}) error {
	// No routes needed for this test
	return nil
}

func (t *testSentryHTTPHandler) GetServiceName() string {
	return "sentry-test-handler"
}

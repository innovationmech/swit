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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerWithSentryDisabled(t *testing.T) {
	config := NewServerConfig()
	config.ServiceName = "test-service"
	config.HTTP.Port = "0" // Random port
	config.GRPC.Port = "0" // Random port
	config.Sentry.Enabled = false

	// Create a simple service registrar for testing
	registrar := &testServiceRegistrar{}

	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err)
	require.NotNil(t, server)

	// Verify Sentry manager exists but is disabled
	assert.NotNil(t, server.sentryManager)
	assert.False(t, server.sentryManager.IsEnabled())
}

func TestServerWithSentryEnabled(t *testing.T) {
	config := NewServerConfig()
	config.ServiceName = "test-service"
	config.HTTP.Port = "0" // Random port
	config.GRPC.Port = "0" // Random port
	config.Sentry.Enabled = true
	config.Sentry.DSN = "https://test@sentry.io/123"
	config.Sentry.Environment = "test"

	// Create a simple service registrar for testing
	registrar := &testServiceRegistrar{}

	server, err := NewBusinessServerCore(config, registrar, nil)
	require.NoError(t, err)
	require.NotNil(t, server)

	// Verify Sentry manager exists and is enabled
	assert.NotNil(t, server.sentryManager)
	assert.True(t, server.sentryManager.IsEnabled())

	// Test server lifecycle with Sentry
	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test shutdown
	err = server.Shutdown()
	assert.NoError(t, err)
}

func TestServerSentryConfigurationValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func(*ServerConfig)
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid Sentry config",
			setupConfig: func(c *ServerConfig) {
				c.Sentry.Enabled = true
				c.Sentry.DSN = "https://test@sentry.io/123"
				c.Sentry.SampleRate = 1.0
				c.Sentry.TracesSampleRate = 0.1
			},
			expectError: false,
		},
		{
			name: "Sentry enabled without DSN",
			setupConfig: func(c *ServerConfig) {
				c.Sentry.Enabled = true
				c.Sentry.DSN = ""
			},
			expectError: true,
			errorMsg:    "sentry.dsn is required",
		},
		{
			name: "invalid sample rate",
			setupConfig: func(c *ServerConfig) {
				c.Sentry.Enabled = true
				c.Sentry.DSN = "https://test@sentry.io/123"
				c.Sentry.SampleRate = 1.5
			},
			expectError: true,
			errorMsg:    "sentry.sample_rate must be between 0 and 1",
		},
		{
			name: "invalid traces sample rate",
			setupConfig: func(c *ServerConfig) {
				c.Sentry.Enabled = true
				c.Sentry.DSN = "https://test@sentry.io/123"
				c.Sentry.SampleRate = 1.0
				c.Sentry.TracesSampleRate = -0.1
			},
			expectError: true,
			errorMsg:    "sentry.traces_sample_rate must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewServerConfig()
			config.ServiceName = "test-service"
			config.HTTP.Port = "0"
			config.GRPC.Port = "0"
			tt.setupConfig(config)

			registrar := &testServiceRegistrar{}

			server, err := NewBusinessServerCore(config, registrar, nil)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, server)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, server)
			}
		})
	}
}

func TestSentryConfigDefaults(t *testing.T) {
	config := NewServerConfig()
	
	// Check Sentry defaults
	assert.Equal(t, 1.0, config.Sentry.SampleRate)
	assert.Equal(t, 0.1, config.Sentry.TracesSampleRate)
	assert.True(t, config.Sentry.AttachStacktrace)
	assert.Equal(t, "development", config.Sentry.Environment)
	assert.NotNil(t, config.Sentry.Tags)
}

// testServiceRegistrar is a minimal service registrar for testing
type testServiceRegistrar struct{}

func (t *testServiceRegistrar) RegisterServices(registry BusinessServiceRegistry) error {
	return nil
}
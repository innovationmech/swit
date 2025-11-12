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

package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Set Gin to test mode for all tests
	gin.SetMode(gin.TestMode)
}

// TestDefaultServerConfig tests the default server configuration.
func TestDefaultServerConfig(t *testing.T) {
	config := DefaultServerConfig()

	assert.NotNil(t, config)
	assert.Equal(t, ":8090", config.Address)
	assert.Equal(t, "8090", config.Port)
	assert.False(t, config.EnableTLS)
	assert.Equal(t, 30*time.Second, config.ReadTimeout)
	assert.Equal(t, 30*time.Second, config.WriteTimeout)
	assert.Equal(t, 60*time.Second, config.IdleTimeout)
	assert.Equal(t, 1<<20, config.MaxHeaderBytes)
	assert.Equal(t, gin.ReleaseMode, config.GinMode)
	assert.Equal(t, "/api/health", config.HealthCheckPath)
	assert.True(t, config.EnableGracefulShutdown)
	assert.Equal(t, 30*time.Second, config.GracefulShutdownTimeout)
}

// TestServerConfigValidation tests configuration validation.
func TestServerConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *ServerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with address",
			config: &ServerConfig{
				Address: ":8080",
			},
			expectError: false,
		},
		{
			name: "valid config with port",
			config: &ServerConfig{
				Port: "9090",
			},
			expectError: false,
		},
		{
			name:        "invalid - no address or port",
			config:      &ServerConfig{},
			expectError: true,
			errorMsg:    "either Address or Port must be specified",
		},
		{
			name: "invalid - TLS enabled but no config",
			config: &ServerConfig{
				Address:   ":8080",
				EnableTLS: true,
			},
			expectError: true,
			errorMsg:    "TLS is enabled but TLSConfig is nil",
		},
		{
			name: "invalid - TLS enabled but no cert files",
			config: &ServerConfig{
				Address:   ":8080",
				EnableTLS: true,
				TLSConfig: &TLSConfig{},
			},
			expectError: true,
			errorMsg:    "TLS certificate and key files must be specified",
		},
		{
			name: "invalid - negative timeout",
			config: &ServerConfig{
				Address:     ":8080",
				ReadTimeout: -1 * time.Second,
			},
			expectError: true,
			errorMsg:    "timeouts cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestServerConfigGetAddress tests the GetAddress method.
func TestServerConfigGetAddress(t *testing.T) {
	tests := []struct {
		name           string
		address        string
		port           string
		expectedResult string
	}{
		{
			name:           "port takes precedence",
			address:        ":8080",
			port:           "9090",
			expectedResult: ":9090",
		},
		{
			name:           "use address when no port",
			address:        "0.0.0.0:8080",
			port:           "",
			expectedResult: "0.0.0.0:8080",
		},
		{
			name:           "only port specified",
			address:        "",
			port:           "7070",
			expectedResult: ":7070",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ServerConfig{
				Address: tt.address,
				Port:    tt.port,
			}

			result := config.GetAddress()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestNewMonitoringServer tests server creation.
func TestNewMonitoringServer(t *testing.T) {
	t.Run("create with default config", func(t *testing.T) {
		server, err := NewMonitoringServer(nil)

		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.NotNil(t, server.router)
		assert.NotNil(t, server.config)
		assert.NotNil(t, server.middleware)
		assert.NotNil(t, server.routeManager)
		assert.False(t, server.IsRunning())
	})

	t.Run("create with custom config", func(t *testing.T) {
		config := DefaultServerConfig()
		config.Port = "9999"
		config.GinMode = gin.TestMode

		server, err := NewMonitoringServer(config)

		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, "9999", server.config.Port)
	})

	t.Run("fail with invalid config", func(t *testing.T) {
		config := &ServerConfig{
			// No address or port
		}

		server, err := NewMonitoringServer(config)

		assert.Error(t, err)
		assert.Nil(t, server)
	})
}

// TestServerStartStop tests starting and stopping the server.
func TestServerStartStop(t *testing.T) {
	config := DefaultServerConfig()
	config.Port = "0" // Use random available port
	config.GinMode = gin.TestMode

	server, err := NewMonitoringServer(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test Start
	err = server.Start(ctx)
	require.NoError(t, err)
	assert.True(t, server.IsRunning())
	assert.NotEmpty(t, server.GetAddress())

	// Wait a bit for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Test that we can't start again
	err = server.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test Stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Stop(stopCtx)
	require.NoError(t, err)
	assert.False(t, server.IsRunning())

	// Test that we can stop again (no error)
	err = server.Stop(stopCtx)
	assert.NoError(t, err)
}

// TestHealthCheckEndpoint tests the health check endpoint.
func TestHealthCheckEndpoint(t *testing.T) {
	config := DefaultServerConfig()
	config.Port = "0"
	config.GinMode = gin.TestMode

	server, err := NewMonitoringServer(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = server.Stop(context.Background())
	}()

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Test health check endpoint
	resp, err := http.Get(fmt.Sprintf("http://%s/api/health", server.GetAddress()))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse response
	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "healthy", result["status"])
	assert.Equal(t, "saga-monitoring", result["service"])
	assert.NotNil(t, result["timestamp"])
}

// TestRootEndpoint tests the root endpoint.
func TestRootEndpoint(t *testing.T) {
	config := DefaultServerConfig()
	config.Port = "0"
	config.GinMode = gin.TestMode

	server, err := NewMonitoringServer(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = server.Stop(context.Background())
	}()

	time.Sleep(100 * time.Millisecond)

	// Test root endpoint
	resp, err := http.Get(fmt.Sprintf("http://%s/", server.GetAddress()))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "Saga Monitoring Dashboard", result["service"])
	assert.Equal(t, "1.0.0", result["version"])
	assert.Equal(t, "running", result["status"])
	assert.NotNil(t, result["links"])
}

// TestAPIInfoEndpoint tests the API info endpoint.
func TestAPIInfoEndpoint(t *testing.T) {
	config := DefaultServerConfig()
	config.Port = "0"
	config.GinMode = gin.TestMode

	server, err := NewMonitoringServer(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = server.Stop(context.Background())
	}()

	time.Sleep(100 * time.Millisecond)

	// Test API info endpoint
	resp, err := http.Get(fmt.Sprintf("http://%s/api/info", server.GetAddress()))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "Saga Monitoring API", result["name"])
	assert.Equal(t, "v1", result["version"])
	assert.NotNil(t, result["endpoints"])
}

// TestLivenessProbe tests the Kubernetes liveness probe endpoint.
func TestLivenessProbe(t *testing.T) {
	config := DefaultServerConfig()
	config.Port = "0"
	config.GinMode = gin.TestMode

	server, err := NewMonitoringServer(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = server.Stop(context.Background())
	}()

	time.Sleep(100 * time.Millisecond)

	// Test liveness probe
	resp, err := http.Get(fmt.Sprintf("http://%s/api/health/live", server.GetAddress()))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "alive", result["status"])
}

// TestReadinessProbe tests the Kubernetes readiness probe endpoint.
func TestReadinessProbe(t *testing.T) {
	config := DefaultServerConfig()
	config.Port = "0"
	config.GinMode = gin.TestMode

	server, err := NewMonitoringServer(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = server.Stop(context.Background())
	}()

	time.Sleep(100 * time.Millisecond)

	// Test readiness probe
	resp, err := http.Get(fmt.Sprintf("http://%s/api/health/ready", server.GetAddress()))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "ready", result["status"])
}

// TestRegisterCustomRoute tests custom route registration.
func TestRegisterCustomRoute(t *testing.T) {
	config := DefaultServerConfig()
	config.Port = "0"
	config.GinMode = gin.TestMode

	server, err := NewMonitoringServer(config)
	require.NoError(t, err)

	// Register custom route
	server.RegisterCustomRoute("GET", "/api/custom", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Custom endpoint",
		})
	})

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = server.Stop(context.Background())
	}()

	time.Sleep(100 * time.Millisecond)

	// Test custom route
	resp, err := http.Get(fmt.Sprintf("http://%s/api/custom", server.GetAddress()))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "Custom endpoint", result["message"])
}

// TestServerWithHealthManager tests server with health manager integration.
func TestServerWithHealthManager(t *testing.T) {
	config := DefaultServerConfig()
	config.Port = "0"
	config.GinMode = gin.TestMode

	// Create health manager
	healthManager := NewSagaHealthManager(nil)

	config.HealthManager = healthManager

	server, err := NewMonitoringServer(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = server.Stop(context.Background())
	}()

	time.Sleep(100 * time.Millisecond)

	// Test health check with health manager
	resp, err := http.Get(fmt.Sprintf("http://%s/api/health", server.GetAddress()))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, "healthy", result["status"])
	assert.NotNil(t, result["components"])
}

// TestMiddlewareConfiguration tests middleware configuration.
func TestMiddlewareConfiguration(t *testing.T) {
	config := DefaultServerConfig()
	config.Port = "0"
	config.GinMode = gin.TestMode

	server, err := NewMonitoringServer(config)
	require.NoError(t, err)

	// Verify middleware is configured
	assert.NotNil(t, server.middleware)
	assert.NotNil(t, server.middleware.config)
	assert.True(t, server.middleware.config.EnableLogging)
	assert.True(t, server.middleware.config.EnableCORS)
	assert.True(t, server.middleware.config.EnableErrorHandle)
	assert.True(t, server.middleware.config.EnableRecovery)
}

// TestGracefulShutdown tests graceful shutdown behavior.
func TestGracefulShutdown(t *testing.T) {
	config := DefaultServerConfig()
	config.Port = "0"
	config.GinMode = gin.TestMode
	config.EnableGracefulShutdown = true
	config.GracefulShutdownTimeout = 5 * time.Second

	server, err := NewMonitoringServer(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Stop with graceful shutdown
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Stop(stopCtx)
	assert.NoError(t, err)
	assert.False(t, server.IsRunning())
}

// TestCORSHeaders tests CORS headers in responses.
func TestCORSHeaders(t *testing.T) {
	config := DefaultServerConfig()
	config.Port = "0"
	config.GinMode = gin.TestMode

	server, err := NewMonitoringServer(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = server.Stop(context.Background())
	}()

	time.Sleep(100 * time.Millisecond)

	// Make an OPTIONS request to test CORS
	client := &http.Client{}
	req, err := http.NewRequest("OPTIONS", fmt.Sprintf("http://%s/api/health", server.GetAddress()), nil)
	require.NoError(t, err)

	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check CORS headers are present
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
}

// TestServerStartupFailure tests server startup failure scenarios.
func TestServerStartupFailure(t *testing.T) {
	t.Run("TLS certificate file not found", func(t *testing.T) {
		config := DefaultServerConfig()
		config.Port = "0"
		config.GinMode = gin.TestMode
		config.EnableTLS = true
		config.TLSConfig = &TLSConfig{
			CertFile: "/nonexistent/cert.pem",
			KeyFile:  "/nonexistent/key.pem",
		}

		server, err := NewMonitoringServer(config)
		require.NoError(t, err)

		ctx := context.Background()
		err = server.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server startup failed")
		assert.False(t, server.IsRunning())
	})

	t.Run("port already in use", func(t *testing.T) {
		// Start a server on a specific port
		config1 := DefaultServerConfig()
		config1.Port = "54321" // Use a fixed port
		config1.GinMode = gin.TestMode

		server1, err := NewMonitoringServer(config1)
		require.NoError(t, err)

		ctx := context.Background()
		err = server1.Start(ctx)
		require.NoError(t, err)
		defer func() {
			_ = server1.Stop(context.Background())
		}()

		// Try to start another server on the same port
		config2 := DefaultServerConfig()
		config2.Port = "54321"
		config2.GinMode = gin.TestMode

		server2, err := NewMonitoringServer(config2)
		require.NoError(t, err)

		err = server2.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "address already in use")
		assert.False(t, server2.IsRunning())
	})

	t.Run("invalid address", func(t *testing.T) {
		config := DefaultServerConfig()
		config.Address = "invalid-address-format"
		config.Port = ""
		config.GinMode = gin.TestMode

		server, err := NewMonitoringServer(config)
		require.NoError(t, err)

		ctx := context.Background()
		err = server.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing port in address")
		assert.False(t, server.IsRunning())
	})
}

// TestServerRestartAfterFailure tests that server can be restarted after a failure.
func TestServerRestartAfterFailure(t *testing.T) {
	config := DefaultServerConfig()
	config.Port = "54322" // Use a fixed port
	config.GinMode = gin.TestMode

	server, err := NewMonitoringServer(config)
	require.NoError(t, err)

	ctx := context.Background()

	// First, start a server on the port
	server1, err := NewMonitoringServer(config)
	require.NoError(t, err)
	err = server1.Start(ctx)
	require.NoError(t, err)

	// Try to start our server - should fail
	err = server.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "address already in use")
	assert.False(t, server.IsRunning())

	// Stop the first server
	err = server1.Stop(context.Background())
	require.NoError(t, err)

	// Wait for the port to be released with retry logic
	// In CI environments, port release can take longer
	var startErr error
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		time.Sleep(200 * time.Millisecond)
		startErr = server.Start(ctx)
		if startErr == nil {
			break
		}
		// If it's still "address already in use", retry
		if !strings.Contains(startErr.Error(), "address already in use") {
			break // Different error, don't retry
		}
	}

	// Now our server should be able to start
	require.NoError(t, startErr)
	assert.True(t, server.IsRunning())

	// Clean up
	err = server.Stop(context.Background())
	require.NoError(t, err)
}

// TestSanitizeForLog tests the sanitization function for log injection prevention.
func TestSanitizeForLog(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "/api/users/123",
			expected: "/api/users/123",
		},
		{
			name:     "string with newline",
			input:    "/api/users/123\nLog Injection Attempt",
			expected: "/api/users/123Log Injection Attempt",
		},
		{
			name:     "string with carriage return",
			input:    "/api/users/123\rLog Injection Attempt",
			expected: "/api/users/123Log Injection Attempt",
		},
		{
			name:     "string with tab",
			input:    "/api/users/123\tLog Injection Attempt",
			expected: "/api/users/123Log Injection Attempt",
		},
		{
			name:     "string with all control characters",
			input:    "/api/users/123\n\r\tLog Injection Attempt",
			expected: "/api/users/123Log Injection Attempt",
		},
		{
			name:     "long string gets truncated",
			input:    string(make([]byte, 600)), // 600 bytes
			expected: string(make([]byte, 500)) + "... [truncated]",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeForLog(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

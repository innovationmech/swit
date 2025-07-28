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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORSSecurityValidation(t *testing.T) {
	tests := []struct {
		name          string
		config        CORSConfig
		expectError   bool
		errorContains string
	}{
		{
			name: "secure configuration - specific origins with credentials",
			config: CORSConfig{
				AllowOrigins:     []string{"http://localhost:3000", "https://example.com"},
				AllowCredentials: true,
				MaxAge:           86400,
			},
			expectError: false,
		},
		{
			name: "SECURITY VIOLATION - wildcard with credentials",
			config: CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowCredentials: true,
				MaxAge:           86400,
			},
			expectError:   true,
			errorContains: "CORS security violation: cannot use wildcard origin '*' with AllowCredentials=true",
		},
		{
			name: "SECURITY VIOLATION - mixed origins with wildcard and credentials",
			config: CORSConfig{
				AllowOrigins:     []string{"http://localhost:3000", "*", "https://example.com"},
				AllowCredentials: true,
				MaxAge:           86400,
			},
			expectError:   true,
			errorContains: "CORS security violation: cannot use wildcard origin '*' with AllowCredentials=true",
		},
		{
			name: "safe configuration - wildcard without credentials",
			config: CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowCredentials: false,
				MaxAge:           86400,
			},
			expectError: false,
		},
		{
			name: "invalid origin format",
			config: CORSConfig{
				AllowOrigins:     []string{"localhost:3000"}, // missing protocol
				AllowCredentials: true,
				MaxAge:           86400,
			},
			expectError:   true,
			errorContains: "invalid origin format",
		},
		{
			name: "empty origin with credentials",
			config: CORSConfig{
				AllowOrigins:     []string{""},
				AllowCredentials: true,
				MaxAge:           86400,
			},
			expectError:   true,
			errorContains: "origin at index 0 cannot be empty",
		},
		{
			name: "negative MaxAge",
			config: CORSConfig{
				AllowOrigins:     []string{"http://localhost:3000"},
				AllowCredentials: true,
				MaxAge:           -1,
			},
			expectError:   true,
			errorContains: "MaxAge cannot be negative",
		},
		{
			name: "excessive MaxAge warning",
			config: CORSConfig{
				AllowOrigins:     []string{"http://localhost:3000"},
				AllowCredentials: true,
				MaxAge:           86400 * 8, // 8 days, exceeds 7-day recommendation
			},
			expectError:   true,
			errorContains: "exceeds recommended maximum of 7 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server config with the CORS config
			serverConfig := &ServerConfig{
				ServiceName: "test-service",
				HTTP: HTTPConfig{
					Port:    "8080",
					Enabled: true,
					Middleware: HTTPMiddleware{
						EnableCORS: true,
						CORSConfig: tt.config,
					},
				},
			}

			// Validate the configuration
			err := serverConfig.validateCORSConfig()

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorContains != "" {
					assert.True(t, strings.Contains(err.Error(), tt.errorContains),
						"Expected error to contain '%s', but got: %s", tt.errorContains, err.Error())
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

func TestServerConfigValidation_WithCORSSecurity(t *testing.T) {
	// Test that the main Validate() method calls CORS validation
	config := NewServerConfig()

	// Set an insecure CORS configuration
	config.HTTP.Middleware.CORSConfig.AllowOrigins = []string{"*"}
	config.HTTP.Middleware.CORSConfig.AllowCredentials = true

	// Validate should fail due to CORS security violation
	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CORS security violation")
}

func TestDefaultCORSConfiguration_Security(t *testing.T) {
	// Test that default configuration is secure
	config := NewServerConfig()

	// Verify default CORS configuration is secure
	assert.NotContains(t, config.HTTP.Middleware.CORSConfig.AllowOrigins, "*",
		"Default CORS configuration should not contain wildcard origin")

	// Verify default configuration passes validation
	err := config.Validate()
	assert.NoError(t, err, "Default configuration should pass CORS security validation")

	// Verify all default origins are localhost for development safety
	for _, origin := range config.HTTP.Middleware.CORSConfig.AllowOrigins {
		assert.True(t,
			strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1"),
			"Default origins should be localhost for development safety, got: %s", origin)
	}
}

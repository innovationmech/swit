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
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/security/audit"
	"github.com/innovationmech/swit/pkg/security/oauth2"
	"github.com/innovationmech/swit/pkg/security/opa"
	"github.com/innovationmech/swit/pkg/security/scanner"
	"github.com/innovationmech/swit/pkg/security/secrets"
)

func TestNewSecurityManager(t *testing.T) {
	tests := []struct {
		name    string
		config  *SecurityConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "disabled security",
			config: &SecurityConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "enabled with minimal config",
			config: &SecurityConfig{
				Enabled: true,
			},
			wantErr: false,
		},
		{
			name: "enabled with audit config",
			config: &SecurityConfig{
				Enabled: true,
				Audit: &audit.AuditLoggerConfig{
					Enabled:    true,
					OutputType: audit.OutputTypeStdout,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm, err := NewSecurityManager(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, sm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, sm)
			}
		})
	}
}

func TestSecurityConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *SecurityConfig
		wantErr bool
	}{
		{
			name: "disabled config",
			config: &SecurityConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "valid minimal config",
			config: &SecurityConfig{
				Enabled: true,
			},
			wantErr: false,
		},
		{
			name: "invalid oauth2 config",
			config: &SecurityConfig{
				Enabled: true,
				OAuth2: &oauth2.Config{
					Enabled: true,
					// Missing required fields
				},
			},
			wantErr: true,
		},
		{
			name: "invalid opa config",
			config: &SecurityConfig{
				Enabled: true,
				OPA: &opa.Config{
					Mode: "invalid_mode",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid scanner config - no tools",
			config: &SecurityConfig{
				Enabled: true,
				Scanner: &scanner.ScannerConfig{
					Enabled: true,
					Tools:   []string{}, // Empty tools
				},
			},
			wantErr: true,
		},
		{
			name: "invalid audit config",
			config: &SecurityConfig{
				Enabled: true,
				Audit: &audit.AuditLoggerConfig{
					Enabled:    true,
					OutputType: "invalid_type",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecurityConfig_SetDefaults(t *testing.T) {
	config := &SecurityConfig{
		Enabled: true,
	}

	config.SetDefaults()

	assert.NotNil(t, config.Audit)
	assert.NotNil(t, config.Scanner)
	assert.False(t, config.Scanner.Enabled) // Scanner should be disabled by default
}

func TestSecurityManager_InitializeSecurity(t *testing.T) {
	tests := []struct {
		name    string
		config  *SecurityConfig
		wantErr bool
	}{
		{
			name: "disabled security",
			config: &SecurityConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "enabled with audit only",
			config: &SecurityConfig{
				Enabled: true,
				Audit: &audit.AuditLoggerConfig{
					Enabled:    true,
					OutputType: audit.OutputTypeStdout,
				},
			},
			wantErr: false,
		},
		{
			name: "enabled with secrets manager",
			config: &SecurityConfig{
				Enabled: true,
				Audit: &audit.AuditLoggerConfig{
					Enabled:    true,
					OutputType: audit.OutputTypeStdout,
				},
				Secrets: &secrets.ManagerConfig{
					Providers: []*secrets.ProviderConfig{
						{
							Type:    secrets.ProviderTypeMemory,
							Enabled: true,
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm, err := NewSecurityManager(tt.config)
			require.NoError(t, err)
			require.NotNil(t, sm)

			ctx := context.Background()
			err = sm.InitializeSecurity(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.config.Enabled {
					assert.True(t, sm.IsInitialized())
				}
			}
		})
	}
}

func TestSecurityManager_InitializeSecurity_DoubleInit(t *testing.T) {
	config := &SecurityConfig{
		Enabled: true,
		Audit: &audit.AuditLoggerConfig{
			Enabled:    true,
			OutputType: audit.OutputTypeStdout,
		},
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = sm.InitializeSecurity(ctx)
	require.NoError(t, err)

	// Second initialization should fail
	err = sm.InitializeSecurity(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already initialized")
}

func TestSecurityManager_RegisterSecurityMiddleware(t *testing.T) {
	config := &SecurityConfig{
		Enabled: true,
		Audit: &audit.AuditLoggerConfig{
			Enabled:    true,
			OutputType: audit.OutputTypeStdout,
		},
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = sm.InitializeSecurity(ctx)
	require.NoError(t, err)

	// Test with gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	err = sm.RegisterSecurityMiddleware(router, nil)
	assert.NoError(t, err)

	// Test with grpc server
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()

	err = sm.RegisterSecurityMiddleware(nil, grpcServer)
	assert.NoError(t, err)
}

func TestSecurityManager_RegisterSecurityMiddleware_NotInitialized(t *testing.T) {
	config := &SecurityConfig{
		Enabled: true,
		Audit: &audit.AuditLoggerConfig{
			Enabled:    true,
			OutputType: audit.OutputTypeStdout,
		},
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)

	// Try to register middleware without initialization
	gin.SetMode(gin.TestMode)
	router := gin.New()
	err = sm.RegisterSecurityMiddleware(router, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestSecurityManager_Getters(t *testing.T) {
	config := &SecurityConfig{
		Enabled: true,
		Audit: &audit.AuditLoggerConfig{
			Enabled:    true,
			OutputType: audit.OutputTypeStdout,
		},
		Secrets: &secrets.ManagerConfig{
			Providers: []*secrets.ProviderConfig{
				{
					Type:    secrets.ProviderTypeMemory,
					Enabled: true,
				},
			},
		},
		Scanner: &scanner.ScannerConfig{
			Enabled: false,
		},
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = sm.InitializeSecurity(ctx)
	require.NoError(t, err)

	// Test getters
	assert.NotNil(t, sm.GetAuditLogger())
	assert.NotNil(t, sm.GetSecretsManager())
	assert.Nil(t, sm.GetOAuth2Client())    // Not configured
	assert.Nil(t, sm.GetOPAClient())       // Not configured
	assert.Nil(t, sm.GetSecurityScanner()) // Disabled

	assert.True(t, sm.IsEnabled())
	assert.True(t, sm.IsInitialized())
}

func TestSecurityManager_Shutdown(t *testing.T) {
	config := &SecurityConfig{
		Enabled: true,
		Audit: &audit.AuditLoggerConfig{
			Enabled:    true,
			OutputType: audit.OutputTypeStdout,
		},
		Secrets: &secrets.ManagerConfig{
			Providers: []*secrets.ProviderConfig{
				{
					Type:    secrets.ProviderTypeMemory,
					Enabled: true,
				},
			},
		},
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = sm.InitializeSecurity(ctx)
	require.NoError(t, err)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = sm.Shutdown(ctx)
	// Ignore stdout sync errors
	if err != nil && !strings.Contains(err.Error(), "bad file descriptor") {
		assert.NoError(t, err)
	}

	// Double shutdown should be safe
	err = sm.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestSecurityManager_Shutdown_AfterClose(t *testing.T) {
	config := &SecurityConfig{
		Enabled: true,
		Audit: &audit.AuditLoggerConfig{
			Enabled:    true,
			OutputType: audit.OutputTypeStdout,
		},
		Secrets: &secrets.ManagerConfig{
			Providers: []*secrets.ProviderConfig{
				{
					Type:    secrets.ProviderTypeMemory,
					Enabled: true,
				},
			},
		},
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = sm.InitializeSecurity(ctx)
	require.NoError(t, err)

	// Close once
	err = sm.Shutdown(ctx)
	// Ignore stdout sync errors
	if err != nil && !strings.Contains(err.Error(), "bad file descriptor") {
		require.NoError(t, err)
	}

	// Try to initialize after close should fail
	err = sm.InitializeSecurity(ctx)
	assert.Error(t, err)
	// Should fail because already initialized or closed
	assert.True(t, strings.Contains(err.Error(), "is closed") || strings.Contains(err.Error(), "already initialized"))
}

func TestSecurityManager_DisabledSecurity(t *testing.T) {
	config := &SecurityConfig{
		Enabled: false,
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Initialize should succeed but do nothing
	err = sm.InitializeSecurity(ctx)
	assert.NoError(t, err)

	// All getters should return nil
	assert.Nil(t, sm.GetOAuth2Client())
	assert.Nil(t, sm.GetOPAClient())
	assert.Nil(t, sm.GetSecurityScanner())
	assert.Nil(t, sm.GetSecretsManager())
	assert.Nil(t, sm.GetAuditLogger())

	assert.False(t, sm.IsEnabled())
	assert.True(t, sm.IsInitialized()) // Should be initialized (but with no-op)

	// Shutdown should succeed
	err = sm.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestSecurityManager_ConcurrentAccess(t *testing.T) {
	config := &SecurityConfig{
		Enabled: true,
		Audit: &audit.AuditLoggerConfig{
			Enabled:    true,
			OutputType: audit.OutputTypeStdout,
		},
	}

	sm, err := NewSecurityManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = sm.InitializeSecurity(ctx)
	require.NoError(t, err)

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			sm.IsEnabled()
			sm.IsInitialized()
			sm.GetAuditLogger()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

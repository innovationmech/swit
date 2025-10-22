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

package security

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthCredentials_Validate(t *testing.T) {
	tests := []struct {
		name    string
		creds   *AuthCredentials
		wantErr bool
		errType error
	}{
		{
			name: "valid JWT credentials",
			creds: &AuthCredentials{
				Type:  AuthTypeJWT,
				Token: "valid-token",
			},
			wantErr: false,
		},
		{
			name: "valid API key credentials",
			creds: &AuthCredentials{
				Type:   AuthTypeAPIKey,
				APIKey: "valid-api-key",
			},
			wantErr: false,
		},
		{
			name: "valid none type",
			creds: &AuthCredentials{
				Type: AuthTypeNone,
			},
			wantErr: false,
		},
		{
			name: "missing JWT token",
			creds: &AuthCredentials{
				Type: AuthTypeJWT,
			},
			wantErr: true,
			errType: ErrMissingCredentials,
		},
		{
			name: "missing API key",
			creds: &AuthCredentials{
				Type: AuthTypeAPIKey,
			},
			wantErr: true,
			errType: ErrMissingCredentials,
		},
		{
			name: "invalid auth type",
			creds: &AuthCredentials{
				Type: "invalid",
			},
			wantErr: true,
			errType: ErrInvalidAuthType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creds.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthCredentials_IsExpired(t *testing.T) {
	tests := []struct {
		name    string
		creds   *AuthCredentials
		expired bool
	}{
		{
			name: "not expired",
			creds: &AuthCredentials{
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(1 * time.Hour)
					return &t
				}(),
			},
			expired: false,
		},
		{
			name: "expired",
			creds: &AuthCredentials{
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(-1 * time.Hour)
					return &t
				}(),
			},
			expired: true,
		},
		{
			name: "no expiration",
			creds: &AuthCredentials{
				ExpiresAt: nil,
			},
			expired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expired, tt.creds.IsExpired())
		})
	}
}

func TestAuthCredentials_HasScope(t *testing.T) {
	creds := &AuthCredentials{
		Scopes: []string{"saga:execute", "saga:read"},
	}

	tests := []struct {
		name     string
		scope    string
		expected bool
	}{
		{
			name:     "has scope",
			scope:    "saga:execute",
			expected: true,
		},
		{
			name:     "does not have scope",
			scope:    "saga:admin",
			expected: false,
		},
		{
			name:     "empty scope",
			scope:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, creds.HasScope(tt.scope))
		})
	}
}

func TestNewAuthContext(t *testing.T) {
	creds := &AuthCredentials{
		Type:   AuthTypeJWT,
		Token:  "test-token",
		UserID: "user-123",
	}

	sagaID := "saga-123"
	authCtx := NewAuthContext(creds, sagaID)

	assert.NotNil(t, authCtx)
	assert.Equal(t, creds, authCtx.Credentials)
	assert.Equal(t, sagaID, authCtx.SagaID)
	assert.Equal(t, "saga", authCtx.Source)
	assert.False(t, authCtx.Timestamp.IsZero())
}

func createTestJWTToken(secret []byte, userID string, scopes []string, expiresIn time.Duration) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"scopes":  scopes,
		"iss":     "test-issuer",
		"aud":     "test-audience",
		"exp":     time.Now().Add(expiresIn).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)
	return tokenString
}

func TestJWTAuthProvider_Authenticate(t *testing.T) {
	secret := []byte("test-secret")
	config := &JWTAuthProviderConfig{
		Secret:       string(secret),
		Issuer:       "test-issuer",
		Audience:     "test-audience",
		CacheEnabled: false,
	}

	provider, err := NewJWTAuthProvider(config)
	require.NoError(t, err)

	validToken := createTestJWTToken(secret, "user-123", []string{"saga:execute"}, 1*time.Hour)

	tests := []struct {
		name        string
		credentials *AuthCredentials
		wantErr     bool
		errType     error
	}{
		{
			name: "valid JWT authentication",
			credentials: &AuthCredentials{
				Type:  AuthTypeJWT,
				Token: validToken,
			},
			wantErr: false,
		},
		{
			name: "invalid auth type",
			credentials: &AuthCredentials{
				Type:   AuthTypeAPIKey,
				APIKey: "some-key",
			},
			wantErr: true,
			errType: ErrInvalidAuthType,
		},
		{
			name: "missing token",
			credentials: &AuthCredentials{
				Type: AuthTypeJWT,
			},
			wantErr: true,
			errType: ErrMissingCredentials,
		},
		{
			name: "invalid token",
			credentials: &AuthCredentials{
				Type:  AuthTypeJWT,
				Token: "invalid-token",
			},
			wantErr: true,
			errType: ErrAuthenticationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			authCtx, err := provider.Authenticate(ctx, tt.credentials)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				assert.Nil(t, authCtx)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, authCtx)
				assert.Equal(t, "user-123", authCtx.Credentials.UserID)
				assert.Contains(t, authCtx.Credentials.Scopes, "saga:execute")
			}
		})
	}
}

func TestJWTAuthProvider_ValidateToken(t *testing.T) {
	secret := []byte("test-secret")
	config := &JWTAuthProviderConfig{
		Secret:       string(secret),
		Issuer:       "test-issuer",
		Audience:     "test-audience",
		CacheEnabled: false,
	}

	provider, err := NewJWTAuthProvider(config)
	require.NoError(t, err)

	validToken := createTestJWTToken(secret, "user-123", []string{"saga:execute"}, 1*time.Hour)
	expiredToken := createTestJWTToken(secret, "user-123", []string{"saga:execute"}, -1*time.Hour)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "expired token",
			token:   expiredToken,
			wantErr: true,
		},
		{
			name:    "invalid token",
			token:   "invalid-token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			creds, err := provider.ValidateToken(ctx, tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, creds)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "user-123", creds.UserID)
			}
		})
	}
}

func TestJWTAuthProvider_IsHealthy(t *testing.T) {
	config := &JWTAuthProviderConfig{
		Secret:       "test-secret",
		CacheEnabled: false,
	}

	provider, err := NewJWTAuthProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	assert.True(t, provider.IsHealthy(ctx))
}

func TestAPIKeyAuthProvider_Authenticate(t *testing.T) {
	config := &APIKeyAuthProviderConfig{
		APIKeys: map[string]string{
			"valid-key-1": "user-123",
			"valid-key-2": "user-456",
		},
		CacheEnabled: false,
	}

	provider, err := NewAPIKeyAuthProvider(config)
	require.NoError(t, err)

	tests := []struct {
		name        string
		credentials *AuthCredentials
		wantErr     bool
		errType     error
		expectedUID string
	}{
		{
			name: "valid API key",
			credentials: &AuthCredentials{
				Type:   AuthTypeAPIKey,
				APIKey: "valid-key-1",
			},
			wantErr:     false,
			expectedUID: "user-123",
		},
		{
			name: "another valid API key",
			credentials: &AuthCredentials{
				Type:   AuthTypeAPIKey,
				APIKey: "valid-key-2",
			},
			wantErr:     false,
			expectedUID: "user-456",
		},
		{
			name: "invalid auth type",
			credentials: &AuthCredentials{
				Type:  AuthTypeJWT,
				Token: "some-token",
			},
			wantErr: true,
			errType: ErrInvalidAuthType,
		},
		{
			name: "missing API key",
			credentials: &AuthCredentials{
				Type: AuthTypeAPIKey,
			},
			wantErr: true,
			errType: ErrMissingCredentials,
		},
		{
			name: "invalid API key",
			credentials: &AuthCredentials{
				Type:   AuthTypeAPIKey,
				APIKey: "invalid-key",
			},
			wantErr: true,
			errType: ErrInvalidAPIKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			authCtx, err := provider.Authenticate(ctx, tt.credentials)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				assert.Nil(t, authCtx)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, authCtx)
				assert.Equal(t, tt.expectedUID, authCtx.Credentials.UserID)
				assert.Contains(t, authCtx.Credentials.Scopes, "saga:execute")
			}
		})
	}
}

func TestAPIKeyAuthProvider_ValidateToken(t *testing.T) {
	config := &APIKeyAuthProviderConfig{
		APIKeys: map[string]string{
			"valid-key": "user-123",
		},
		CacheEnabled: false,
	}

	provider, err := NewAPIKeyAuthProvider(config)
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid API key",
			token:   "valid-key",
			wantErr: false,
		},
		{
			name:    "invalid API key",
			token:   "invalid-key",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			creds, err := provider.ValidateToken(ctx, tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, creds)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "user-123", creds.UserID)
			}
		})
	}
}

func TestAPIKeyAuthProvider_IsHealthy(t *testing.T) {
	config := &APIKeyAuthProviderConfig{
		APIKeys: map[string]string{
			"key": "user",
		},
		CacheEnabled: false,
	}

	provider, err := NewAPIKeyAuthProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	assert.True(t, provider.IsHealthy(ctx))
}

func TestAuthCache(t *testing.T) {
	ttl := 100 * time.Millisecond
	cache := NewAuthCache(ttl)

	authCtx := NewAuthContext(&AuthCredentials{
		Type:   AuthTypeJWT,
		UserID: "user-123",
	}, "saga-123")

	// Test Set and Get
	cache.Set("test-key", authCtx)
	retrieved, found := cache.Get("test-key")
	assert.True(t, found)
	assert.Equal(t, authCtx.Credentials.UserID, retrieved.Credentials.UserID)

	// Test expiration
	time.Sleep(150 * time.Millisecond)
	_, found = cache.Get("test-key")
	assert.False(t, found)

	// Test Delete
	cache.Set("test-key-2", authCtx)
	cache.Delete("test-key-2")
	_, found = cache.Get("test-key-2")
	assert.False(t, found)
}

func TestAuthManager_Authenticate(t *testing.T) {
	jwtProvider, err := NewJWTAuthProvider(&JWTAuthProviderConfig{
		Secret:       "test-secret",
		Issuer:       "test-issuer",
		Audience:     "test-audience",
		CacheEnabled: false,
	})
	require.NoError(t, err)

	apiKeyProvider, err := NewAPIKeyAuthProvider(&APIKeyAuthProviderConfig{
		APIKeys: map[string]string{
			"valid-key": "user-123",
		},
		CacheEnabled: false,
	})
	require.NoError(t, err)

	config := &AuthManagerConfig{
		DefaultProvider: "jwt",
		Providers: map[string]AuthProvider{
			"jwt":    jwtProvider,
			"apikey": apiKeyProvider,
		},
		CacheEnabled: false,
	}

	manager, err := NewAuthManager(config)
	require.NoError(t, err)

	secret := []byte("test-secret")
	validToken := createTestJWTToken(secret, "user-123", []string{"saga:execute"}, 1*time.Hour)

	tests := []struct {
		name        string
		credentials *AuthCredentials
		wantErr     bool
	}{
		{
			name: "authenticate with JWT",
			credentials: &AuthCredentials{
				Type:  AuthTypeJWT,
				Token: validToken,
			},
			wantErr: false,
		},
		{
			name: "authenticate with API key",
			credentials: &AuthCredentials{
				Type:   AuthTypeAPIKey,
				APIKey: "valid-key",
			},
			wantErr: false,
		},
		{
			name:        "authenticate with nil credentials",
			credentials: nil,
			wantErr:     false,
		},
		{
			name: "authenticate with invalid credentials",
			credentials: &AuthCredentials{
				Type:  AuthTypeJWT,
				Token: "invalid-token",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			authCtx, err := manager.Authenticate(ctx, tt.credentials)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, authCtx)
			}
		})
	}
}

func TestAuthManager_AddRemoveProvider(t *testing.T) {
	jwtProvider, err := NewJWTAuthProvider(&JWTAuthProviderConfig{
		Secret:       "test-secret",
		CacheEnabled: false,
	})
	require.NoError(t, err)

	config := &AuthManagerConfig{
		DefaultProvider: "jwt",
		Providers: map[string]AuthProvider{
			"jwt": jwtProvider,
		},
		CacheEnabled: false,
	}

	manager, err := NewAuthManager(config)
	require.NoError(t, err)

	// Test AddProvider
	apiKeyProvider, err := NewAPIKeyAuthProvider(&APIKeyAuthProviderConfig{
		APIKeys: map[string]string{
			"key": "user",
		},
		CacheEnabled: false,
	})
	require.NoError(t, err)

	err = manager.AddProvider("apikey", apiKeyProvider)
	assert.NoError(t, err)

	provider, exists := manager.GetProvider("apikey")
	assert.True(t, exists)
	assert.NotNil(t, provider)

	// Test duplicate provider
	err = manager.AddProvider("apikey", apiKeyProvider)
	assert.Error(t, err)

	// Test RemoveProvider
	err = manager.RemoveProvider("apikey")
	assert.NoError(t, err)

	_, exists = manager.GetProvider("apikey")
	assert.False(t, exists)

	// Test remove default provider
	err = manager.RemoveProvider("jwt")
	assert.Error(t, err)

	// Test remove non-existent provider
	err = manager.RemoveProvider("non-existent")
	assert.Error(t, err)
}

func TestAuthManager_IsHealthy(t *testing.T) {
	jwtProvider, err := NewJWTAuthProvider(&JWTAuthProviderConfig{
		Secret:       "test-secret",
		CacheEnabled: false,
	})
	require.NoError(t, err)

	config := &AuthManagerConfig{
		DefaultProvider: "jwt",
		Providers: map[string]AuthProvider{
			"jwt": jwtProvider,
		},
		CacheEnabled: false,
	}

	manager, err := NewAuthManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	assert.True(t, manager.IsHealthy(ctx))
}

func TestSagaAuthMiddleware_Authenticate(t *testing.T) {
	jwtProvider, err := NewJWTAuthProvider(&JWTAuthProviderConfig{
		Secret:       "test-secret",
		Issuer:       "test-issuer",
		Audience:     "test-audience",
		CacheEnabled: false,
	})
	require.NoError(t, err)

	authManager, err := NewAuthManager(&AuthManagerConfig{
		DefaultProvider: "jwt",
		Providers: map[string]AuthProvider{
			"jwt": jwtProvider,
		},
		CacheEnabled: false,
	})
	require.NoError(t, err)

	secret := []byte("test-secret")
	validToken := createTestJWTToken(secret, "user-123", []string{"saga:execute"}, 1*time.Hour)
	expiredToken := createTestJWTToken(secret, "user-123", []string{"saga:execute"}, -1*time.Hour)

	tests := []struct {
		name        string
		middleware  *SagaAuthMiddleware
		credentials *AuthCredentials
		wantErr     bool
		errType     error
	}{
		{
			name: "successful authentication",
			middleware: NewSagaAuthMiddleware(&SagaAuthMiddlewareConfig{
				AuthManager: authManager,
				Required:    true,
			}),
			credentials: &AuthCredentials{
				Type:  AuthTypeJWT,
				Token: validToken,
			},
			wantErr: false,
		},
		{
			name: "authentication not required with nil credentials",
			middleware: NewSagaAuthMiddleware(&SagaAuthMiddlewareConfig{
				AuthManager: authManager,
				Required:    false,
			}),
			credentials: nil,
			wantErr:     false,
		},
		{
			name: "authentication required with nil credentials",
			middleware: NewSagaAuthMiddleware(&SagaAuthMiddlewareConfig{
				AuthManager: authManager,
				Required:    true,
			}),
			credentials: nil,
			wantErr:     true,
			errType:     ErrMissingCredentials,
		},
		{
			name: "expired token",
			middleware: NewSagaAuthMiddleware(&SagaAuthMiddlewareConfig{
				AuthManager: authManager,
				Required:    true,
			}),
			credentials: &AuthCredentials{
				Type:  AuthTypeJWT,
				Token: expiredToken,
			},
			wantErr: true,
			errType: ErrAuthenticationFailed,
		},
		{
			name: "invalid token",
			middleware: NewSagaAuthMiddleware(&SagaAuthMiddlewareConfig{
				AuthManager: authManager,
				Required:    true,
			}),
			credentials: &AuthCredentials{
				Type:  AuthTypeJWT,
				Token: "invalid-token",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			authCtx, err := tt.middleware.Authenticate(ctx, tt.credentials)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, authCtx)
			}
		})
	}
}

func TestSagaAuthMiddleware_RequireScope(t *testing.T) {
	jwtProvider, err := NewJWTAuthProvider(&JWTAuthProviderConfig{
		Secret:       "test-secret",
		CacheEnabled: false,
	})
	require.NoError(t, err)

	authManager, err := NewAuthManager(&AuthManagerConfig{
		DefaultProvider: "jwt",
		Providers: map[string]AuthProvider{
			"jwt": jwtProvider,
		},
		CacheEnabled: false,
	})
	require.NoError(t, err)

	middleware := NewSagaAuthMiddleware(&SagaAuthMiddlewareConfig{
		AuthManager: authManager,
		Required:    true,
	})

	tests := []struct {
		name          string
		authCtx       *AuthContext
		requiredScope string
		wantErr       bool
		errType       error
	}{
		{
			name: "has required scope",
			authCtx: NewAuthContext(&AuthCredentials{
				Scopes: []string{"saga:execute", "saga:read"},
			}, "saga-123"),
			requiredScope: "saga:execute",
			wantErr:       false,
		},
		{
			name: "missing required scope",
			authCtx: NewAuthContext(&AuthCredentials{
				Scopes: []string{"saga:read"},
			}, "saga-123"),
			requiredScope: "saga:admin",
			wantErr:       true,
			errType:       ErrInsufficientScope,
		},
		{
			name:          "nil auth context",
			authCtx:       nil,
			requiredScope: "saga:execute",
			wantErr:       true,
			errType:       ErrMissingCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := middleware.RequireScope(tt.authCtx, tt.requiredScope)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSagaAuthMiddleware_IsHealthy(t *testing.T) {
	jwtProvider, err := NewJWTAuthProvider(&JWTAuthProviderConfig{
		Secret:       "test-secret",
		CacheEnabled: false,
	})
	require.NoError(t, err)

	authManager, err := NewAuthManager(&AuthManagerConfig{
		DefaultProvider: "jwt",
		Providers: map[string]AuthProvider{
			"jwt": jwtProvider,
		},
		CacheEnabled: false,
	})
	require.NoError(t, err)

	middleware := NewSagaAuthMiddleware(&SagaAuthMiddlewareConfig{
		AuthManager: authManager,
		Required:    true,
	})

	ctx := context.Background()
	assert.True(t, middleware.IsHealthy(ctx))
}

func TestContextWithAuth(t *testing.T) {
	authCtx := NewAuthContext(&AuthCredentials{
		Type:   AuthTypeJWT,
		UserID: "user-123",
	}, "saga-123")

	ctx := context.Background()
	ctx = ContextWithAuth(ctx, authCtx)

	retrieved, ok := AuthFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, authCtx.Credentials.UserID, retrieved.Credentials.UserID)
	assert.Equal(t, authCtx.SagaID, retrieved.SagaID)
}

func TestAuthFromContext_NotFound(t *testing.T) {
	ctx := context.Background()

	retrieved, ok := AuthFromContext(ctx)
	assert.False(t, ok)
	assert.Nil(t, retrieved)
}

func TestJWTAuthProvider_WithCache(t *testing.T) {
	secret := []byte("test-secret")
	config := &JWTAuthProviderConfig{
		Secret:       string(secret),
		Issuer:       "test-issuer",
		Audience:     "test-audience",
		CacheEnabled: true,
		CacheTTL:     1 * time.Second,
	}

	provider, err := NewJWTAuthProvider(config)
	require.NoError(t, err)

	validToken := createTestJWTToken(secret, "user-123", []string{"saga:execute"}, 1*time.Hour)

	credentials := &AuthCredentials{
		Type:  AuthTypeJWT,
		Token: validToken,
	}

	ctx := context.Background()

	// First authentication should hit the provider
	authCtx1, err := provider.Authenticate(ctx, credentials)
	assert.NoError(t, err)
	assert.NotNil(t, authCtx1)

	// Second authentication should hit the cache
	authCtx2, err := provider.Authenticate(ctx, credentials)
	assert.NoError(t, err)
	assert.NotNil(t, authCtx2)

	// Wait for cache expiration
	time.Sleep(1100 * time.Millisecond)

	// Third authentication should hit the provider again
	authCtx3, err := provider.Authenticate(ctx, credentials)
	assert.NoError(t, err)
	assert.NotNil(t, authCtx3)
}

func TestAPIKeyAuthProvider_WithCache(t *testing.T) {
	config := &APIKeyAuthProviderConfig{
		APIKeys: map[string]string{
			"valid-key": "user-123",
		},
		CacheEnabled: true,
		CacheTTL:     1 * time.Second,
	}

	provider, err := NewAPIKeyAuthProvider(config)
	require.NoError(t, err)

	credentials := &AuthCredentials{
		Type:   AuthTypeAPIKey,
		APIKey: "valid-key",
	}

	ctx := context.Background()

	// First authentication should hit the provider
	authCtx1, err := provider.Authenticate(ctx, credentials)
	assert.NoError(t, err)
	assert.NotNil(t, authCtx1)

	// Second authentication should hit the cache
	authCtx2, err := provider.Authenticate(ctx, credentials)
	assert.NoError(t, err)
	assert.NotNil(t, authCtx2)
}

func TestNewJWTAuthProvider_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *JWTAuthProviderConfig
	}{
		{
			name: "missing secret",
			config: &JWTAuthProviderConfig{
				Issuer:   "test-issuer",
				Audience: "test-audience",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewJWTAuthProvider(tt.config)
			assert.Error(t, err)
			assert.Nil(t, provider)
		})
	}
}

func TestNewAPIKeyAuthProvider_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *APIKeyAuthProviderConfig
	}{
		{
			name: "no API keys",
			config: &APIKeyAuthProviderConfig{
				APIKeys: map[string]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAPIKeyAuthProvider(tt.config)
			assert.Error(t, err)
			assert.Nil(t, provider)
		})
	}
}

func TestNewAuthManager_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *AuthManagerConfig
	}{
		{
			name: "missing default provider",
			config: &AuthManagerConfig{
				Providers: map[string]AuthProvider{},
			},
		},
		{
			name: "default provider not found",
			config: &AuthManagerConfig{
				DefaultProvider: "jwt",
				Providers:       map[string]AuthProvider{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewAuthManager(tt.config)
			assert.Error(t, err)
			assert.Nil(t, manager)
		})
	}
}

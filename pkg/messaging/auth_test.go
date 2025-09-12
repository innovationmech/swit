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

package messaging

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthCredentials_Validate(t *testing.T) {
	tests := []struct {
		name        string
		credentials *AuthCredentials
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid JWT credentials",
			credentials: &AuthCredentials{
				Type:  AuthTypeJWT,
				Token: "test-token",
			},
			expectError: false,
		},
		{
			name: "Invalid JWT - missing token",
			credentials: &AuthCredentials{
				Type: AuthTypeJWT,
			},
			expectError: true,
			errorMsg:    "token is required for JWT authentication",
		},
		{
			name: "Valid API key credentials",
			credentials: &AuthCredentials{
				Type:   AuthTypeAPIKey,
				APIKey: "test-api-key",
			},
			expectError: false,
		},
		{
			name: "Invalid API key - missing key",
			credentials: &AuthCredentials{
				Type: AuthTypeAPIKey,
			},
			expectError: true,
			errorMsg:    "api_key is required for API key authentication",
		},
		{
			name: "Valid certificate credentials",
			credentials: &AuthCredentials{
				Type:        AuthTypeCertificate,
				Certificate: &tls.Certificate{},
			},
			expectError: false,
		},
		{
			name: "Invalid certificate - missing certificate",
			credentials: &AuthCredentials{
				Type: AuthTypeCertificate,
			},
			expectError: true,
			errorMsg:    "certificate is required for certificate authentication",
		},
		{
			name: "Invalid auth type",
			credentials: &AuthCredentials{
				Type: "invalid",
			},
			expectError: true,
			errorMsg:    "unsupported authentication type: invalid",
		},
		{
			name: "None authentication",
			credentials: &AuthCredentials{
				Type: AuthTypeNone,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.credentials.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthCredentials_IsExpired(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)
	past := now.Add(-1 * time.Hour)

	tests := []struct {
		name        string
		credentials *AuthCredentials
		expected    bool
	}{
		{
			name: "No expiration time",
			credentials: &AuthCredentials{
				Type: AuthTypeJWT,
			},
			expected: false,
		},
		{
			name: "Not expired",
			credentials: &AuthCredentials{
				Type:      AuthTypeJWT,
				ExpiresAt: &future,
			},
			expected: false,
		},
		{
			name: "Expired",
			credentials: &AuthCredentials{
				Type:      AuthTypeJWT,
				ExpiresAt: &past,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.credentials.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthCredentials_HasScope(t *testing.T) {
	credentials := &AuthCredentials{
		Type:   AuthTypeJWT,
		Scopes: []string{"messaging:publish", "messaging:consume", "admin"},
	}

	tests := []struct {
		name     string
		scope    string
		expected bool
	}{
		{
			name:     "Has scope",
			scope:    "messaging:publish",
			expected: true,
		},
		{
			name:     "Does not have scope",
			scope:    "messaging:delete",
			expected: false,
		},
		{
			name:     "Empty scope",
			scope:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := credentials.HasScope(tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthContext_ToHeaders(t *testing.T) {
	token := "test-jwt-token"
	apiKey := "test-api-key"
	expiresAt := time.Now().Add(1 * time.Hour)

	tests := []struct {
		name        string
		authContext *AuthContext
		expected    map[string]string
	}{
		{
			name: "JWT authentication",
			authContext: &AuthContext{
				Credentials: &AuthCredentials{
					Type:      AuthTypeJWT,
					Token:     token,
					UserID:    "user123",
					Scopes:    []string{"messaging:publish"},
					ExpiresAt: &expiresAt,
					Metadata:  map[string]string{"role": "admin"},
				},
				Timestamp:     time.Now(),
				MessageID:     "msg123",
				CorrelationID: "corr123",
			},
			expected: map[string]string{
				"x-auth-type":           "jwt",
				"x-auth-user-id":        "user123",
				"x-auth-timestamp":      time.Now().Format(time.RFC3339),
				"x-auth-message-id":     "msg123",
				"x-auth-correlation-id": "corr123",
				"authorization":         "Bearer " + token,
				"x-auth-scopes":         "messaging:publish",
				"x-auth-meta-role":      "admin",
			},
		},
		{
			name: "API key authentication",
			authContext: &AuthContext{
				Credentials: &AuthCredentials{
					Type:   AuthTypeAPIKey,
					APIKey: apiKey,
					UserID: "user456",
					Scopes: []string{"messaging:consume"},
				},
				Timestamp: time.Now(),
				MessageID: "msg456",
			},
			expected: map[string]string{
				"x-auth-type":       "apikey",
				"x-auth-user-id":    "user456",
				"x-auth-timestamp":  time.Now().Format(time.RFC3339),
				"x-auth-message-id": "msg456",
				"x-api-key":         apiKey,
				"x-auth-scopes":     "messaging:consume",
			},
		},
		{
			name: "No authentication",
			authContext: &AuthContext{
				Credentials: &AuthCredentials{
					Type: AuthTypeNone,
				},
				Timestamp: time.Now(),
				MessageID: "msg789",
			},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := tt.authContext.ToHeaders()

			// Check that all expected headers are present
			for key, expectedValue := range tt.expected {
				assert.Contains(t, headers, key)
				assert.Equal(t, expectedValue, headers[key])
			}
		})
	}
}

func TestAuthContextFromHeaders(t *testing.T) {
	tests := []struct {
		name        string
		headers     map[string]string
		expectError bool
		expectType  AuthType
	}{
		{
			name: "JWT authentication headers",
			headers: map[string]string{
				"x-auth-type":       "jwt",
				"x-auth-user-id":    "user123",
				"x-auth-timestamp":  time.Now().Format(time.RFC3339),
				"x-auth-message-id": "msg123",
				"authorization":     "Bearer test-token",
				"x-auth-scopes":     "messaging:publish",
				"x-auth-meta-role":  "admin",
			},
			expectError: false,
			expectType:  AuthTypeJWT,
		},
		{
			name: "API key authentication headers",
			headers: map[string]string{
				"x-auth-type":       "apikey",
				"x-auth-user-id":    "user456",
				"x-auth-timestamp":  time.Now().Format(time.RFC3339),
				"x-auth-message-id": "msg456",
				"x-api-key":         "test-api-key",
				"x-auth-scopes":     "messaging:consume",
			},
			expectError: false,
			expectType:  AuthTypeAPIKey,
		},
		{
			name: "No authentication headers",
			headers: map[string]string{
				"x-auth-message-id": "msg789",
			},
			expectError: false,
			expectType:  AuthTypeNone,
		},
		{
			name: "Invalid JWT - missing token",
			headers: map[string]string{
				"x-auth-type":       "jwt",
				"x-auth-user-id":    "user123",
				"x-auth-message-id": "msg123",
			},
			expectError: true,
			expectType:  AuthTypeNone,
		},
		{
			name: "Invalid API key - missing API key",
			headers: map[string]string{
				"x-auth-type":       "apikey",
				"x-auth-user-id":    "user456",
				"x-auth-message-id": "msg456",
			},
			expectError: true,
			expectType:  AuthTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authCtx, err := AuthContextFromHeaders(tt.headers)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, authCtx)
				assert.Equal(t, tt.expectType, authCtx.Credentials.Type)

				if tt.expectType != AuthTypeNone {
					assert.NotEmpty(t, authCtx.Credentials.UserID)
				}
			}
		})
	}
}

func TestJWTAuthProvider(t *testing.T) {
	secret := "test-secret-123"
	issuer := "test-issuer"

	config := &JWTAuthProviderConfig{
		Secret:       secret,
		Issuer:       issuer,
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
	}

	provider, err := NewJWTAuthProvider(config)
	require.NoError(t, err)
	assert.Equal(t, "jwt", provider.Name())

	ctx := context.Background()

	// Test with valid JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "test-user",
		"scopes":  []interface{}{"messaging:publish"},
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
		"iss":     issuer,
	})

	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	credentials := &AuthCredentials{
		Type:  AuthTypeJWT,
		Token: tokenString,
	}

	authCtx, err := provider.Authenticate(ctx, credentials)
	assert.NoError(t, err)
	assert.NotNil(t, authCtx)
	assert.Equal(t, "test-user", authCtx.Credentials.UserID)
	assert.Contains(t, authCtx.Credentials.Scopes, "messaging:publish")

	// Test cache
	cachedAuthCtx, err := provider.Authenticate(ctx, credentials)
	assert.NoError(t, err)
	assert.Equal(t, authCtx.MessageID, cachedAuthCtx.MessageID)

	// Test health check
	assert.True(t, provider.IsHealthy(ctx))

	// Test with invalid token
	invalidCredentials := &AuthCredentials{
		Type:  AuthTypeJWT,
		Token: "invalid-token",
	}

	_, err = provider.Authenticate(ctx, invalidCredentials)
	assert.Error(t, err)
}

func TestAPIKeyAuthProvider(t *testing.T) {
	config := &APIKeyAuthProviderConfig{
		APIKeys: map[string]string{
			"key1": "user1",
			"key2": "user2",
		},
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
	}

	provider, err := NewAPIKeyAuthProvider(config)
	require.NoError(t, err)
	assert.Equal(t, "apikey", provider.Name())

	ctx := context.Background()

	// Test with valid API key
	credentials := &AuthCredentials{
		Type:   AuthTypeAPIKey,
		APIKey: "key1",
	}

	authCtx, err := provider.Authenticate(ctx, credentials)
	assert.NoError(t, err)
	assert.NotNil(t, authCtx)
	assert.Equal(t, "user1", authCtx.Credentials.UserID)
	assert.Contains(t, authCtx.Credentials.Scopes, "messaging:publish")

	// Test cache
	cachedAuthCtx, err := provider.Authenticate(ctx, credentials)
	assert.NoError(t, err)
	assert.Equal(t, authCtx.MessageID, cachedAuthCtx.MessageID)

	// Test health check
	assert.True(t, provider.IsHealthy(ctx))

	// Test with invalid API key
	invalidCredentials := &AuthCredentials{
		Type:   AuthTypeAPIKey,
		APIKey: "invalid-key",
	}

	_, err = provider.Authenticate(ctx, invalidCredentials)
	assert.Error(t, err)
}

func TestAuthCache(t *testing.T) {
	cache := NewAuthCache(100 * time.Millisecond)

	authCtx := &AuthContext{
		Credentials: &AuthCredentials{
			Type:   AuthTypeJWT,
			UserID: "test-user",
		},
		Timestamp: time.Now(),
		MessageID: "msg123",
	}

	// Test Set and Get
	cache.Set("test-key", authCtx)

	retrieved, found := cache.Get("test-key")
	assert.True(t, found)
	assert.Equal(t, authCtx.MessageID, retrieved.MessageID)

	// Test cache expiration
	time.Sleep(150 * time.Millisecond)

	_, found = cache.Get("test-key")
	assert.False(t, found)

	// Test Delete
	cache.Set("test-key2", authCtx)
	cache.Delete("test-key2")

	_, found = cache.Get("test-key2")
	assert.False(t, found)
}

func TestAuthManager(t *testing.T) {
	// Create JWT provider
	jwtConfig := &JWTAuthProviderConfig{
		Secret: "test-secret",
	}
	jwtProvider, err := NewJWTAuthProvider(jwtConfig)
	require.NoError(t, err)

	// Create API key provider
	apiKeyConfig := &APIKeyAuthProviderConfig{
		APIKeys: map[string]string{
			"test-key": "test-user",
		},
	}
	apiKeyProvider, err := NewAPIKeyAuthProvider(apiKeyConfig)
	require.NoError(t, err)

	// Create auth manager
	config := &AuthManagerConfig{
		DefaultProvider: "jwt",
		Providers: map[string]AuthProvider{
			"jwt":    jwtProvider,
			"apikey": apiKeyProvider,
		},
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
	}

	manager, err := NewAuthManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test with JWT credentials
	jwtCredentials := &AuthCredentials{
		Type:  AuthTypeJWT,
		Token: "test-token", // This will fail validation but tests the routing
	}

	_, err = manager.Authenticate(ctx, jwtCredentials)
	assert.Error(t, err) // Expected to fail due to invalid token

	// Test with API key credentials
	apiKeyCredentials := &AuthCredentials{
		Type:   AuthTypeAPIKey,
		APIKey: "test-key",
	}

	authCtx, err := manager.Authenticate(ctx, apiKeyCredentials)
	assert.NoError(t, err)
	assert.Equal(t, "test-user", authCtx.Credentials.UserID)

	// Test health check
	assert.True(t, manager.IsHealthy(ctx))

	// Test provider management
	provider, exists := manager.GetProvider("jwt")
	assert.True(t, exists)
	assert.Equal(t, jwtProvider, provider)

	providers := manager.GetProviders()
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, "jwt")
	assert.Contains(t, providers, "apikey")

	// Test adding provider
	mockProvider := &MockAuthProvider{name: "mock"}
	err = manager.AddProvider("mock", mockProvider)
	assert.NoError(t, err)

	provider, exists = manager.GetProvider("mock")
	assert.True(t, exists)
	assert.Equal(t, mockProvider, provider)

	// Test removing provider
	err = manager.RemoveProvider("mock")
	assert.NoError(t, err)

	_, exists = manager.GetProvider("mock")
	assert.False(t, exists)
}

func TestAuthMiddleware(t *testing.T) {
	// Create auth manager
	jwtConfig := &JWTAuthProviderConfig{
		Secret: "test-secret",
	}
	jwtProvider, err := NewJWTAuthProvider(jwtConfig)
	require.NoError(t, err)

	authManagerConfig := &AuthManagerConfig{
		DefaultProvider: "jwt",
		Providers: map[string]AuthProvider{
			"jwt": jwtProvider,
		},
	}

	authManager, err := NewAuthManager(authManagerConfig)
	require.NoError(t, err)

	// Create middleware
	middlewareConfig := &AuthMiddlewareConfig{
		AuthManager: authManager,
		Required:    true,
	}

	middleware := NewAuthMiddleware(middlewareConfig)
	assert.Equal(t, "authentication", middleware.Name())

	// Create mock handler
	mockHandler := &MockMessageHandler{
		handleFunc: func(ctx context.Context, message *Message) error {
			return nil
		},
	}

	// Wrap handler with middleware
	wrappedHandler := middleware.Wrap(mockHandler)

	ctx := context.Background()

	// Test with authenticated message
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "test-user",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	})

	tokenString, _ := validToken.SignedString([]byte("test-secret"))

	message := &Message{
		ID: "msg123",
		Headers: map[string]string{
			"x-auth-type":   "jwt",
			"authorization": "Bearer " + tokenString,
		},
	}

	err = wrappedHandler.Handle(ctx, message)
	assert.NoError(t, err)

	// Test with unauthenticated message
	message = &Message{
		ID: "msg456",
		Headers: map[string]string{
			"x-auth-type":   "jwt",
			"authorization": "Bearer invalid-token",
		},
	}

	err = wrappedHandler.Handle(ctx, message)
	assert.Error(t, err)
}

// Mock implementations for testing

type MockAuthProvider struct {
	name string
}

func (m *MockAuthProvider) Name() string { return m.name }
func (m *MockAuthProvider) Authenticate(ctx context.Context, credentials *AuthCredentials) (*AuthContext, error) {
	return NewAuthContext(credentials, "mock-msg"), nil
}
func (m *MockAuthProvider) ValidateToken(ctx context.Context, token string) (*AuthCredentials, error) {
	return &AuthCredentials{Type: AuthTypeNone}, nil
}
func (m *MockAuthProvider) RefreshToken(ctx context.Context, credentials *AuthCredentials) (*AuthCredentials, error) {
	return credentials, nil
}
func (m *MockAuthProvider) IsHealthy(ctx context.Context) bool { return true }

type MockMessageHandler struct {
	handleFunc func(ctx context.Context, message *Message) error
}

func (m *MockMessageHandler) Handle(ctx context.Context, message *Message) error {
	return m.handleFunc(ctx, message)
}

func (m *MockMessageHandler) OnError(ctx context.Context, message *Message, err error) ErrorAction {
	return ErrorActionRetry
}

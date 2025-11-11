// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	secjwt "github.com/innovationmech/swit/pkg/security/jwt"
)

// Mock gRPC handler for testing
type mockGRPCUnaryHandler struct {
	called  bool
	request interface{}
	err     error
}

func (m *mockGRPCUnaryHandler) handle(ctx context.Context, req interface{}) (interface{}, error) {
	m.called = true
	m.request = req
	return "success", m.err
}

// Mock stream for testing
type mockGRPCServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockGRPCServerStream) Context() context.Context {
	return m.ctx
}

// Mock stream handler for testing
type mockGRPCStreamHandler struct {
	called bool
	err    error
}

func (m *mockGRPCStreamHandler) handle(srv interface{}, stream grpc.ServerStream) error {
	m.called = true
	return m.err
}

// Helper function to create a valid JWT token for testing
func createGRPCTestToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	return tokenString
}

// Helper function to create a JWT validator for testing
func createTestValidator(t *testing.T, secret string) *secjwt.Validator {
	t.Helper()

	config := &secjwt.Config{
		Secret:            secret,
		AllowedAlgorithms: []string{"HS256"},
		SkipIssuer:        true,
		SkipExpiry:        true,
	}

	validator, err := secjwt.NewValidator(config)
	require.NoError(t, err)

	return validator
}

func TestUnaryAuthInterceptor(t *testing.T) {
	secret := "test-secret"
	validator := createTestValidator(t, secret)

	claims := jwt.MapClaims{
		"sub":      "user123",
		"username": "testuser",
		"email":    "test@example.com",
		"roles":    []interface{}{"admin", "user"},
		"exp":      time.Now().Add(time.Hour).Unix(),
	}
	tokenString := createGRPCTestToken(t, secret, claims)

	tests := []struct {
		name           string
		setupMetadata  func() metadata.MD
		config         *GRPCAuthConfig
		expectError    bool
		expectCode     codes.Code
		expectCalled   bool
		validateResult func(t *testing.T, ctx context.Context)
	}{
		{
			name: "valid token",
			setupMetadata: func() metadata.MD {
				return metadata.Pairs("authorization", "Bearer "+tokenString)
			},
			config: &GRPCAuthConfig{
				JWTValidator: validator,
			},
			expectError:  false,
			expectCalled: true,
			validateResult: func(t *testing.T, ctx context.Context) {
				userInfo, err := GetGRPCUserInfo(ctx)
				require.NoError(t, err)
				assert.Equal(t, "user123", userInfo.Subject)
				assert.Equal(t, "testuser", userInfo.Username)
				assert.Equal(t, "test@example.com", userInfo.Email)
			},
		},
		{
			name: "missing metadata",
			setupMetadata: func() metadata.MD {
				return nil
			},
			config: &GRPCAuthConfig{
				JWTValidator: validator,
			},
			expectError:  true,
			expectCode:   codes.Unauthenticated,
			expectCalled: false,
		},
		{
			name: "missing token",
			setupMetadata: func() metadata.MD {
				return metadata.Pairs()
			},
			config: &GRPCAuthConfig{
				JWTValidator: validator,
			},
			expectError:  true,
			expectCode:   codes.Unauthenticated,
			expectCalled: false,
		},
		{
			name: "invalid token format",
			setupMetadata: func() metadata.MD {
				return metadata.Pairs("authorization", "InvalidFormat")
			},
			config: &GRPCAuthConfig{
				JWTValidator: validator,
			},
			expectError:  true,
			expectCode:   codes.Unauthenticated,
			expectCalled: false,
		},
		{
			name: "optional auth with missing token",
			setupMetadata: func() metadata.MD {
				return metadata.Pairs()
			},
			config: &GRPCAuthConfig{
				JWTValidator: validator,
				Optional:     true,
			},
			expectError:  false,
			expectCalled: true,
			validateResult: func(t *testing.T, ctx context.Context) {
				_, err := GetGRPCUserInfo(ctx)
				assert.Error(t, err) // No user info should be present
			},
		},
		{
			name: "skip method",
			setupMetadata: func() metadata.MD {
				return metadata.Pairs() // No token
			},
			config: &GRPCAuthConfig{
				JWTValidator: validator,
				SkipMethods:  []string{"/test.Service/TestMethod"},
			},
			expectError:  false,
			expectCalled: true,
		},
		{
			name: "wildcard skip method",
			setupMetadata: func() metadata.MD {
				return metadata.Pairs() // No token
			},
			config: &GRPCAuthConfig{
				JWTValidator: validator,
				SkipMethods:  []string{"/test.Service/*"},
			},
			expectError:  false,
			expectCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create interceptor
			interceptor := UnaryAuthInterceptorWithConfig(tt.config)

			// Create mock handler
			mockHandler := &mockGRPCUnaryHandler{}

			// Create context with metadata
			ctx := context.Background()
			if md := tt.setupMetadata(); md != nil {
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			// Create mock info
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			// Call interceptor
			result, err := interceptor(ctx, "request", info, mockHandler.handle)

			// Validate results
			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, "success", result)
			}

			assert.Equal(t, tt.expectCalled, mockHandler.called)

			if tt.validateResult != nil && mockHandler.called {
				// Extract context from handler call
				// Note: We need to capture the context passed to the handler
				// For simplicity, we'll use the returned context from successful calls
				if !tt.expectError {
					// Re-run with a capturing handler to get the context
					capturedCtx := context.Background()
					capturingHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
						capturedCtx = ctx
						return "success", nil
					}
					_, _ = interceptor(ctx, "request", info, capturingHandler)
					tt.validateResult(t, capturedCtx)
				}
			}
		})
	}
}

func TestStreamAuthInterceptor(t *testing.T) {
	secret := "test-secret"
	validator := createTestValidator(t, secret)

	claims := jwt.MapClaims{
		"sub":      "user123",
		"username": "testuser",
		"exp":      time.Now().Add(time.Hour).Unix(),
	}
	tokenString := createGRPCTestToken(t, secret, claims)

	tests := []struct {
		name          string
		setupMetadata func() metadata.MD
		config        *GRPCAuthConfig
		expectError   bool
		expectCode    codes.Code
		expectCalled  bool
	}{
		{
			name: "valid token",
			setupMetadata: func() metadata.MD {
				return metadata.Pairs("authorization", "Bearer "+tokenString)
			},
			config: &GRPCAuthConfig{
				JWTValidator: validator,
			},
			expectError:  false,
			expectCalled: true,
		},
		{
			name: "missing metadata",
			setupMetadata: func() metadata.MD {
				return nil
			},
			config: &GRPCAuthConfig{
				JWTValidator: validator,
			},
			expectError:  true,
			expectCode:   codes.Unauthenticated,
			expectCalled: false,
		},
		{
			name: "optional auth with missing token",
			setupMetadata: func() metadata.MD {
				return metadata.Pairs()
			},
			config: &GRPCAuthConfig{
				JWTValidator: validator,
				Optional:     true,
			},
			expectError:  false,
			expectCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create interceptor
			interceptor := StreamAuthInterceptorWithConfig(tt.config)

			// Create mock handler
			mockHandler := &mockGRPCStreamHandler{}

			// Create context with metadata
			ctx := context.Background()
			if md := tt.setupMetadata(); md != nil {
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			// Create mock stream
			mockStream := &mockGRPCServerStream{ctx: ctx}

			// Create mock info
			info := &grpc.StreamServerInfo{
				FullMethod: "/test.Service/TestStream",
			}

			// Call interceptor
			err := interceptor(nil, mockStream, info, mockHandler.handle)

			// Validate results
			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectCalled, mockHandler.called)
		})
	}
}

func TestUnaryRequireRoles(t *testing.T) {
	tests := []struct {
		name          string
		userRoles     []string
		requiredRoles []string
		expectError   bool
		expectCode    codes.Code
	}{
		{
			name:          "has required role",
			userRoles:     []string{"admin", "user"},
			requiredRoles: []string{"admin"},
			expectError:   false,
		},
		{
			name:          "has one of required roles",
			userRoles:     []string{"admin", "user"},
			requiredRoles: []string{"superadmin", "admin"},
			expectError:   false,
		},
		{
			name:          "missing required role",
			userRoles:     []string{"user"},
			requiredRoles: []string{"admin"},
			expectError:   true,
			expectCode:    codes.PermissionDenied,
		},
		{
			name:          "no required roles",
			userRoles:     []string{"user"},
			requiredRoles: []string{},
			expectError:   false,
		},
		{
			name:          "user not authenticated",
			userRoles:     nil,
			requiredRoles: []string{"admin"},
			expectError:   true,
			expectCode:    codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create interceptor
			interceptor := UnaryRequireRoles(tt.requiredRoles...)

			// Create context with user info
			ctx := context.Background()
			if tt.userRoles != nil {
				userInfo := &GRPCUserInfo{
					Subject: "user123",
					Roles:   tt.userRoles,
				}
				ctx = context.WithValue(ctx, GRPCContextKeyUserInfo, userInfo)
			}

			// Create mock handler
			mockHandler := &mockGRPCUnaryHandler{}

			// Create mock info
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			// Call interceptor
			_, err := interceptor(ctx, "request", info, mockHandler.handle)

			// Validate results
			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
				assert.False(t, mockHandler.called)
			} else {
				require.NoError(t, err)
				assert.True(t, mockHandler.called)
			}
		})
	}
}

func TestStreamRequireRoles(t *testing.T) {
	tests := []struct {
		name          string
		userRoles     []string
		requiredRoles []string
		expectError   bool
		expectCode    codes.Code
	}{
		{
			name:          "has required role",
			userRoles:     []string{"admin", "user"},
			requiredRoles: []string{"admin"},
			expectError:   false,
		},
		{
			name:          "missing required role",
			userRoles:     []string{"user"},
			requiredRoles: []string{"admin"},
			expectError:   true,
			expectCode:    codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create interceptor
			interceptor := StreamRequireRoles(tt.requiredRoles...)

			// Create context with user info
			ctx := context.Background()
			if tt.userRoles != nil {
				userInfo := &GRPCUserInfo{
					Subject: "user123",
					Roles:   tt.userRoles,
				}
				ctx = context.WithValue(ctx, GRPCContextKeyUserInfo, userInfo)
			}

			// Create mock stream
			mockStream := &mockGRPCServerStream{ctx: ctx}

			// Create mock handler
			mockHandler := &mockGRPCStreamHandler{}

			// Create mock info
			info := &grpc.StreamServerInfo{
				FullMethod: "/test.Service/TestStream",
			}

			// Call interceptor
			err := interceptor(nil, mockStream, info, mockHandler.handle)

			// Validate results
			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
				assert.False(t, mockHandler.called)
			} else {
				require.NoError(t, err)
				assert.True(t, mockHandler.called)
			}
		})
	}
}

func TestUnaryRequireScopes(t *testing.T) {
	tests := []struct {
		name           string
		userScopes     []string
		requiredScopes []string
		expectError    bool
		expectCode     codes.Code
	}{
		{
			name:           "has all required scopes",
			userScopes:     []string{"read", "write", "delete"},
			requiredScopes: []string{"read", "write"},
			expectError:    false,
		},
		{
			name:           "missing one scope",
			userScopes:     []string{"read"},
			requiredScopes: []string{"read", "write"},
			expectError:    true,
			expectCode:     codes.PermissionDenied,
		},
		{
			name:           "no required scopes",
			userScopes:     []string{"read"},
			requiredScopes: []string{},
			expectError:    false,
		},
		{
			name:           "user not authenticated",
			userScopes:     nil,
			requiredScopes: []string{"read"},
			expectError:    true,
			expectCode:     codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create interceptor
			interceptor := UnaryRequireScopes(tt.requiredScopes...)

			// Create context with user info
			ctx := context.Background()
			if tt.userScopes != nil {
				userInfo := &GRPCUserInfo{
					Subject: "user123",
					Scopes:  tt.userScopes,
				}
				ctx = context.WithValue(ctx, GRPCContextKeyUserInfo, userInfo)
			}

			// Create mock handler
			mockHandler := &mockGRPCUnaryHandler{}

			// Create mock info
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.Service/TestMethod",
			}

			// Call interceptor
			_, err := interceptor(ctx, "request", info, mockHandler.handle)

			// Validate results
			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
				assert.False(t, mockHandler.called)
			} else {
				require.NoError(t, err)
				assert.True(t, mockHandler.called)
			}
		})
	}
}

func TestStreamRequireScopes(t *testing.T) {
	tests := []struct {
		name           string
		userScopes     []string
		requiredScopes []string
		expectError    bool
		expectCode     codes.Code
	}{
		{
			name:           "has all required scopes",
			userScopes:     []string{"read", "write"},
			requiredScopes: []string{"read", "write"},
			expectError:    false,
		},
		{
			name:           "missing scope",
			userScopes:     []string{"read"},
			requiredScopes: []string{"read", "write"},
			expectError:    true,
			expectCode:     codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create interceptor
			interceptor := StreamRequireScopes(tt.requiredScopes...)

			// Create context with user info
			ctx := context.Background()
			if tt.userScopes != nil {
				userInfo := &GRPCUserInfo{
					Subject: "user123",
					Scopes:  tt.userScopes,
				}
				ctx = context.WithValue(ctx, GRPCContextKeyUserInfo, userInfo)
			}

			// Create mock stream
			mockStream := &mockGRPCServerStream{ctx: ctx}

			// Create mock handler
			mockHandler := &mockGRPCStreamHandler{}

			// Create mock info
			info := &grpc.StreamServerInfo{
				FullMethod: "/test.Service/TestStream",
			}

			// Call interceptor
			err := interceptor(nil, mockStream, info, mockHandler.handle)

			// Validate results
			if tt.expectError {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
				assert.False(t, mockHandler.called)
			} else {
				require.NoError(t, err)
				assert.True(t, mockHandler.called)
			}
		})
	}
}

func TestMapAuthError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		expectCode codes.Code
	}{
		{
			name:       "nil error",
			err:        nil,
			expectCode: codes.OK,
		},
		{
			name:       "token expired",
			err:        secjwt.ErrTokenExpired,
			expectCode: codes.Unauthenticated,
		},
		{
			name:       "token not yet valid",
			err:        secjwt.ErrTokenNotYetValid,
			expectCode: codes.Unauthenticated,
		},
		{
			name:       "invalid token",
			err:        secjwt.ErrInvalidToken,
			expectCode: codes.Unauthenticated,
		},
		{
			name:       "invalid issuer",
			err:        secjwt.ErrInvalidIssuer,
			expectCode: codes.Unauthenticated,
		},
		{
			name:       "invalid audience",
			err:        secjwt.ErrInvalidAudience,
			expectCode: codes.Unauthenticated,
		},
		{
			name:       "generic error",
			err:        errors.New("some error"),
			expectCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapAuthError(tt.err, nil)

			if tt.err == nil {
				assert.NoError(t, result)
			} else {
				require.Error(t, result)
				st, ok := status.FromError(result)
				require.True(t, ok, "error should be a gRPC status")
				assert.Equal(t, tt.expectCode, st.Code())
			}
		})
	}
}

func TestShouldSkipMethod(t *testing.T) {
	tests := []struct {
		name        string
		fullMethod  string
		skipMethods []string
		expected    bool
	}{
		{
			name:        "exact match",
			fullMethod:  "/test.Service/TestMethod",
			skipMethods: []string{"/test.Service/TestMethod"},
			expected:    true,
		},
		{
			name:        "wildcard match",
			fullMethod:  "/test.Service/TestMethod",
			skipMethods: []string{"/test.Service/*"},
			expected:    true,
		},
		{
			name:        "no match",
			fullMethod:  "/test.Service/TestMethod",
			skipMethods: []string{"/other.Service/OtherMethod"},
			expected:    false,
		},
		{
			name:        "empty skip methods",
			fullMethod:  "/test.Service/TestMethod",
			skipMethods: []string{},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipMethod(tt.fullMethod, tt.skipMethods)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGRPCHasAnyRole(t *testing.T) {
	tests := []struct {
		name          string
		userRoles     []string
		requiredRoles []string
		expected      bool
	}{
		{
			name:          "has role",
			userRoles:     []string{"admin", "user"},
			requiredRoles: []string{"admin"},
			expected:      true,
		},
		{
			name:          "has one of many",
			userRoles:     []string{"user"},
			requiredRoles: []string{"admin", "user"},
			expected:      true,
		},
		{
			name:          "no matching role",
			userRoles:     []string{"guest"},
			requiredRoles: []string{"admin", "user"},
			expected:      false,
		},
		{
			name:          "empty required roles",
			userRoles:     []string{"user"},
			requiredRoles: []string{},
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasAnyRole(tt.userRoles, tt.requiredRoles)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasAllScopes(t *testing.T) {
	tests := []struct {
		name           string
		userScopes     []string
		requiredScopes []string
		expected       bool
	}{
		{
			name:           "has all scopes",
			userScopes:     []string{"read", "write", "delete"},
			requiredScopes: []string{"read", "write"},
			expected:       true,
		},
		{
			name:           "missing one scope",
			userScopes:     []string{"read"},
			requiredScopes: []string{"read", "write"},
			expected:       false,
		},
		{
			name:           "empty required scopes",
			userScopes:     []string{"read"},
			requiredScopes: []string{},
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasAllScopes(tt.userScopes, tt.requiredScopes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextHelpers(t *testing.T) {
	t.Run("GetGRPCUserInfo", func(t *testing.T) {
		ctx := context.Background()
		userInfo := &GRPCUserInfo{
			Subject:  "user123",
			Username: "testuser",
		}
		ctx = context.WithValue(ctx, GRPCContextKeyUserInfo, userInfo)

		result, err := GetGRPCUserInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, userInfo, result)
	})

	t.Run("GetGRPCUserInfo - missing", func(t *testing.T) {
		ctx := context.Background()
		_, err := GetGRPCUserInfo(ctx)
		require.Error(t, err)
	})

	t.Run("GetGRPCClaims", func(t *testing.T) {
		ctx := context.Background()
		claims := jwt.MapClaims{
			"sub": "user123",
		}
		ctx = context.WithValue(ctx, GRPCContextKeyClaims, claims)

		result, err := GetGRPCClaims(ctx)
		require.NoError(t, err)
		assert.Equal(t, claims, result)
	})

	t.Run("GetGRPCClaims - missing", func(t *testing.T) {
		ctx := context.Background()
		_, err := GetGRPCClaims(ctx)
		require.Error(t, err)
	})

	t.Run("GetGRPCTokenString", func(t *testing.T) {
		ctx := context.Background()
		token := "test-token"
		ctx = context.WithValue(ctx, GRPCContextKeyTokenString, token)

		result, err := GetGRPCTokenString(ctx)
		require.NoError(t, err)
		assert.Equal(t, token, result)
	})

	t.Run("GetGRPCTokenString - missing", func(t *testing.T) {
		ctx := context.Background()
		_, err := GetGRPCTokenString(ctx)
		require.Error(t, err)
	})

	t.Run("MustGetGRPCUserInfo - success", func(t *testing.T) {
		ctx := context.Background()
		userInfo := &GRPCUserInfo{Subject: "user123"}
		ctx = context.WithValue(ctx, GRPCContextKeyUserInfo, userInfo)

		result := MustGetGRPCUserInfo(ctx)
		assert.Equal(t, userInfo, result)
	})

	t.Run("MustGetGRPCUserInfo - panic", func(t *testing.T) {
		ctx := context.Background()
		assert.Panics(t, func() {
			MustGetGRPCUserInfo(ctx)
		})
	})

	t.Run("MustGetGRPCClaims - success", func(t *testing.T) {
		ctx := context.Background()
		claims := jwt.MapClaims{"sub": "user123"}
		ctx = context.WithValue(ctx, GRPCContextKeyClaims, claims)

		result := MustGetGRPCClaims(ctx)
		assert.Equal(t, claims, result)
	})

	t.Run("MustGetGRPCClaims - panic", func(t *testing.T) {
		ctx := context.Background()
		assert.Panics(t, func() {
			MustGetGRPCClaims(ctx)
		})
	})
}

func TestDefaultGRPCTokenExtractor(t *testing.T) {
	tests := []struct {
		name        string
		metadata    metadata.MD
		keys        []string
		expectError bool
		expectedVal string
	}{
		{
			name: "extract from authorization header",
			metadata: metadata.Pairs(
				"authorization", "Bearer test-token",
			),
			keys:        []string{"authorization"},
			expectError: false,
			expectedVal: "test-token",
		},
		{
			name: "extract from custom header",
			metadata: metadata.Pairs(
				"x-auth-token", "Bearer custom-token",
			),
			keys:        []string{"x-auth-token"},
			expectError: false,
			expectedVal: "custom-token",
		},
		{
			name:        "missing token",
			metadata:    metadata.Pairs(),
			keys:        []string{"authorization"},
			expectError: true,
		},
		{
			name: "invalid format",
			metadata: metadata.Pairs(
				"authorization", "InvalidFormat",
			),
			keys:        []string{"authorization"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := defaultGRPCTokenExtractor(tt.keys)
			token, err := extractor(tt.metadata)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedVal, token)
			}
		})
	}
}

func TestAuthenticatedServerStream(t *testing.T) {
	originalCtx := context.Background()
	userInfo := &GRPCUserInfo{Subject: "user123"}
	authCtx := context.WithValue(originalCtx, GRPCContextKeyUserInfo, userInfo)

	mockStream := &mockGRPCServerStream{ctx: originalCtx}
	wrappedStream := &authenticatedServerStream{
		ServerStream: mockStream,
		ctx:          authCtx,
	}

	// Test that Context() returns the authenticated context
	ctx := wrappedStream.Context()
	result, err := GetGRPCUserInfo(ctx)
	require.NoError(t, err)
	assert.Equal(t, userInfo, result)
}

func TestGRPCAuthConfigDefaults(t *testing.T) {
	config := &GRPCAuthConfig{}
	setGRPCAuthConfigDefaults(config)

	assert.NotNil(t, config.ErrorHandler)
	assert.NotNil(t, config.TokenExtractor)
	assert.Equal(t, []string{DefaultAuthorizationKey}, config.MetadataKeys)
}

func TestGRPCCustomErrorHandler(t *testing.T) {
	customHandlerCalled := false
	customError := status.Error(codes.Internal, "custom error")

	config := &GRPCAuthConfig{
		JWTValidator: createTestValidator(t, "secret"),
		ErrorHandler: func(ctx context.Context, err error) error {
			customHandlerCalled = true
			return customError
		},
	}

	interceptor := UnaryAuthInterceptorWithConfig(config)

	ctx := context.Background()
	md := metadata.Pairs("authorization", "Bearer invalid-token")
	ctx = metadata.NewIncomingContext(ctx, md)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	mockHandler := &mockGRPCUnaryHandler{}
	_, err := interceptor(ctx, "request", info, mockHandler.handle)

	require.Error(t, err)
	assert.True(t, customHandlerCalled)
	assert.Equal(t, customError, err)
}

func TestCustomTokenExtractor(t *testing.T) {
	extractorCalled := false
	testToken := "custom-extracted-token"

	secret := "test-secret"
	validator := createTestValidator(t, secret)
	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenString := createGRPCTestToken(t, secret, claims)

	config := &GRPCAuthConfig{
		JWTValidator: validator,
		TokenExtractor: func(md metadata.MD) (string, error) {
			extractorCalled = true
			// Return the pre-created valid token
			return tokenString, nil
		},
	}

	interceptor := UnaryAuthInterceptorWithConfig(config)

	ctx := context.Background()
	md := metadata.Pairs("custom-header", testToken)
	ctx = metadata.NewIncomingContext(ctx, md)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	mockHandler := &mockGRPCUnaryHandler{}
	_, err := interceptor(ctx, "request", info, mockHandler.handle)

	require.NoError(t, err)
	assert.True(t, extractorCalled)
	assert.True(t, mockHandler.called)
}

func TestUnaryAuthInterceptorSimple(t *testing.T) {
	secret := "test-secret"
	validator := createTestValidator(t, secret)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenString := createGRPCTestToken(t, secret, claims)

	// Test the simple wrapper
	interceptor := UnaryAuthInterceptor(nil, validator)

	ctx := context.Background()
	md := metadata.Pairs("authorization", "Bearer "+tokenString)
	ctx = metadata.NewIncomingContext(ctx, md)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	mockHandler := &mockGRPCUnaryHandler{}
	_, err := interceptor(ctx, "request", info, mockHandler.handle)

	require.NoError(t, err)
	assert.True(t, mockHandler.called)
}

func TestStreamAuthInterceptorSimple(t *testing.T) {
	secret := "test-secret"
	validator := createTestValidator(t, secret)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenString := createGRPCTestToken(t, secret, claims)

	// Test the simple wrapper
	interceptor := StreamAuthInterceptor(nil, validator)

	ctx := context.Background()
	md := metadata.Pairs("authorization", "Bearer "+tokenString)
	ctx = metadata.NewIncomingContext(ctx, md)

	mockStream := &mockGRPCServerStream{ctx: ctx}
	mockHandler := &mockGRPCStreamHandler{}

	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/TestStream",
	}

	err := interceptor(nil, mockStream, info, mockHandler.handle)

	require.NoError(t, err)
	assert.True(t, mockHandler.called)
}

func TestMultipleMetadataKeys(t *testing.T) {
	secret := "test-secret"
	validator := createTestValidator(t, secret)

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenString := createGRPCTestToken(t, secret, claims)

	config := &GRPCAuthConfig{
		JWTValidator: validator,
		MetadataKeys: []string{"x-api-key", "authorization"},
	}

	interceptor := UnaryAuthInterceptorWithConfig(config)

	// Test with x-api-key
	ctx := context.Background()
	md := metadata.Pairs("x-api-key", "Bearer "+tokenString)
	ctx = metadata.NewIncomingContext(ctx, md)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	mockHandler := &mockGRPCUnaryHandler{}
	_, err := interceptor(ctx, "request", info, mockHandler.handle)

	require.NoError(t, err)
	assert.True(t, mockHandler.called)
}

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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthorizationPermission_String(t *testing.T) {
	tests := []struct {
		name       string
		permission *AuthorizationPermission
		expected   string
	}{
		{
			name: "Permission with scope",
			permission: &AuthorizationPermission{
				Resource: AccessResourceTypeTopic,
				Action:   PermissionPublish,
				Scope:    "orders",
			},
			expected: "topic:publish:orders",
		},
		{
			name: "Permission without scope",
			permission: &AuthorizationPermission{
				Resource: AccessResourceTypeTopic,
				Action:   PermissionConsume,
			},
			expected: "topic:consume",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.permission.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthorizationPermission_Matches(t *testing.T) {
	tests := []struct {
		name       string
		permission *AuthorizationPermission
		required   *AuthorizationPermission
		expected   bool
	}{
		{
			name: "Exact match",
			permission: &AuthorizationPermission{
				Resource: AccessResourceTypeTopic,
				Action:   PermissionPublish,
				Scope:    "orders",
			},
			required: &AuthorizationPermission{
				Resource: AccessResourceTypeTopic,
				Action:   PermissionPublish,
				Scope:    "orders",
			},
			expected: true,
		},
		{
			name: "Wildcard resource",
			permission: &AuthorizationPermission{
				Resource: "*",
				Action:   PermissionPublish,
				Scope:    "orders",
			},
			required: &AuthorizationPermission{
				Resource: AccessResourceTypeTopic,
				Action:   PermissionPublish,
				Scope:    "orders",
			},
			expected: true,
		},
		{
			name: "Wildcard action",
			permission: &AuthorizationPermission{
				Resource: AccessResourceTypeTopic,
				Action:   "*",
				Scope:    "orders",
			},
			required: &AuthorizationPermission{
				Resource: AccessResourceTypeTopic,
				Action:   PermissionPublish,
				Scope:    "orders",
			},
			expected: true,
		},
		{
			name: "Wildcard scope",
			permission: &AuthorizationPermission{
				Resource: AccessResourceTypeTopic,
				Action:   PermissionPublish,
				Scope:    "*",
			},
			required: &AuthorizationPermission{
				Resource: AccessResourceTypeTopic,
				Action:   PermissionPublish,
				Scope:    "orders",
			},
			expected: true,
		},
		{
			name: "Prefix scope match",
			permission: &AuthorizationPermission{
				Resource: AccessResourceTypeTopic,
				Action:   PermissionPublish,
				Scope:    "orders.*",
			},
			required: &AuthorizationPermission{
				Resource: AccessResourceTypeTopic,
				Action:   PermissionPublish,
				Scope:    "orders.created",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.permission.Matches(tt.required)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRBACAuthorizationProvider_Authorize(t *testing.T) {
	config := &RBACAuthorizationProviderConfig{
		Roles: map[string][]*AuthorizationPermission{
			"publisher": {
				{Resource: AccessResourceTypeTopic, Action: PermissionPublish, Scope: "*"},
			},
			"consumer": {
				{Resource: AccessResourceTypeTopic, Action: PermissionConsume, Scope: "*"},
			},
		},
		UserRoles: map[string][]string{
			"user1": {"publisher"},
			"user2": {"consumer"},
		},
	}
	provider := NewRBACAuthorizationProvider(config)

	tests := []struct {
		name           string
		authzCtx       *AuthorizationContext
		expectedResult AuthorizationDecision
	}{
		{
			name: "User with publish permission - allow",
			authzCtx: &AuthorizationContext{
				AuthContext: &AuthContext{
					Credentials: &AuthCredentials{
						UserID: "user1",
					},
				},
				Permission: &AuthorizationPermission{
					Resource: AccessResourceTypeTopic,
					Action:   PermissionPublish,
					Scope:    "orders",
				},
			},
			expectedResult: DecisionAllow,
		},
		{
			name: "User without permission - deny",
			authzCtx: &AuthorizationContext{
				AuthContext: &AuthContext{
					Credentials: &AuthCredentials{
						UserID: "user1",
					},
				},
				Permission: &AuthorizationPermission{
					Resource: AccessResourceTypeTopic,
					Action:   PermissionConsume,
					Scope:    "orders",
				},
			},
			expectedResult: DecisionDeny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := provider.Authorize(ctx, tt.authzCtx)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestAuthorizationManager_Authorize(t *testing.T) {
	rbacProvider := NewRBACAuthorizationProvider(&RBACAuthorizationProviderConfig{
		Roles: map[string][]*AuthorizationPermission{
			"publisher": {
				{Resource: AccessResourceTypeTopic, Action: PermissionPublish, Scope: "*"},
			},
		},
		UserRoles: map[string][]string{
			"user1": {"publisher"},
		},
	})

	manager, err := NewAuthorizationManager(&AuthorizationManagerConfig{
		DefaultProvider: "rbac",
		Providers: map[string]AuthorizationProvider{
			"rbac": rbacProvider,
		},
		DenyByDefault: true,
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		authzCtx       *AuthorizationContext
		expectedResult AuthorizationDecision
	}{
		{
			name: "Authorized user - allow",
			authzCtx: &AuthorizationContext{
				AuthContext: &AuthContext{
					Credentials: &AuthCredentials{
						UserID: "user1",
					},
				},
				Permission: &AuthorizationPermission{
					Resource: AccessResourceTypeTopic,
					Action:   PermissionPublish,
					Scope:    "orders",
				},
			},
			expectedResult: DecisionAllow,
		},
		{
			name: "Unauthorized user - deny",
			authzCtx: &AuthorizationContext{
				AuthContext: &AuthContext{
					Credentials: &AuthCredentials{
						UserID: "user1",
					},
				},
				Permission: &AuthorizationPermission{
					Resource: AccessResourceTypeTopic,
					Action:   PermissionConsume,
					Scope:    "orders",
				},
			},
			expectedResult: DecisionDeny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := manager.Authorize(ctx, tt.authzCtx)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestAuthorizationMiddleware_Handle(t *testing.T) {
	rbacProvider := NewRBACAuthorizationProvider(&RBACAuthorizationProviderConfig{
		Roles: map[string][]*AuthorizationPermission{
			"consumer": {
				{Resource: AccessResourceTypeTopic, Action: PermissionConsume, Scope: "orders.*"},
			},
		},
		UserRoles: map[string][]string{
			"user1": {"consumer"},
		},
	})

	authzManager, err := NewAuthorizationManager(&AuthorizationManagerConfig{
		DefaultProvider: "rbac",
		Providers: map[string]AuthorizationProvider{
			"rbac": rbacProvider,
		},
		DenyByDefault: true,
	})
	require.NoError(t, err)

	middleware := NewAuthorizationMiddleware(&AuthorizationMiddlewareConfig{
		AuthzManager: authzManager,
	})

	handlerCalled := false
	mockHandler := MessageHandlerFunc(func(ctx context.Context, msg *Message) error {
		handlerCalled = true
		return nil
	})

	wrappedHandler := middleware.Wrap(mockHandler)

	tests := []struct {
		name          string
		message       *Message
		authContext   *AuthContext
		expectError   bool
		expectHandler bool
	}{
		{
			name: "Authorized user - success",
			message: &Message{
				ID:    "msg-1",
				Topic: "orders.created",
			},
			authContext: &AuthContext{
				Credentials: &AuthCredentials{
					UserID: "user1",
				},
			},
			expectError:   false,
			expectHandler: true,
		},
		{
			name: "No auth context - denied",
			message: &Message{
				ID:    "msg-3",
				Topic: "orders.created",
			},
			authContext:   nil,
			expectError:   true,
			expectHandler: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false
			ctx := context.Background()

			if tt.authContext != nil {
				ctx = context.WithValue(ctx, "auth_context", tt.authContext)
			}

			err := wrappedHandler.Handle(ctx, tt.message)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectHandler, handlerCalled)
		})
	}
}

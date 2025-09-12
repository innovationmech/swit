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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource(t *testing.T) {
	t.Run("NewResource", func(t *testing.T) {
		resource := NewResource(AccessResourceTypeTopic, "test-topic")
		assert.Equal(t, AccessResourceTypeTopic, resource.Type)
		assert.Equal(t, "test-topic", resource.Name)
		assert.NotNil(t, resource.Metadata)
	})

	t.Run("ResourcePattern", func(t *testing.T) {
		resource := ResourcePattern(AccessResourceTypeTopic, "orders.*")
		assert.Equal(t, AccessResourceTypeTopic, resource.Type)
		assert.Equal(t, "orders.*", resource.Pattern)
		assert.NotNil(t, resource.Metadata)
	})

	t.Run("Matches", func(t *testing.T) {
		tests := []struct {
			name     string
			resource *Resource
			target   *Resource
			expected bool
		}{
			{
				name:     "Exact match",
				resource: NewResource(AccessResourceTypeTopic, "orders"),
				target:   NewResource(AccessResourceTypeTopic, "orders"),
				expected: true,
			},
			{
				name:     "Type mismatch",
				resource: NewResource(AccessResourceTypeTopic, "orders"),
				target:   NewResource(AccessResourceTypeQueue, "orders"),
				expected: false,
			},
			{
				name:     "Name mismatch",
				resource: NewResource(AccessResourceTypeTopic, "orders"),
				target:   NewResource(AccessResourceTypeTopic, "payments"),
				expected: false,
			},
			{
				name:     "Pattern match prefix",
				resource: ResourcePattern(AccessResourceTypeTopic, "orders.*"),
				target:   NewResource(AccessResourceTypeTopic, "orders.processed"),
				expected: true,
			},
			{
				name:     "Pattern match exact",
				resource: ResourcePattern(AccessResourceTypeTopic, "orders.*"),
				target:   NewResource(AccessResourceTypeTopic, "orders.*"),
				expected: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.resource.Matches(tt.target)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

func TestAccessPolicy(t *testing.T) {
	t.Run("CreatePolicy", func(t *testing.T) {
		policy := &AccessPolicy{
			Name:        "test-policy",
			Description: "Test policy",
			Resources: []*Resource{
				NewResource(AccessResourceTypeTopic, "test-topic"),
			},
			Permissions: []Permission{PermissionRead, PermissionWrite},
			Priority:    100,
			Enabled:     true,
		}

		assert.Equal(t, "test-policy", policy.Name)
		assert.Equal(t, "Test policy", policy.Description)
		assert.Len(t, policy.Resources, 1)
		assert.Len(t, policy.Permissions, 2)
		assert.Equal(t, 100, policy.Priority)
		assert.True(t, policy.Enabled)
	})

	t.Run("TimeRestrictions", func(t *testing.T) {
		policy := &AccessPolicy{
			Name: "time-restricted-policy",
			TimeRestrictions: &TimeRestrictions{
				AllowedDays: []int{1, 2, 3, 4, 5}, // Weekdays
				AllowedTimeRanges: []TimeRange{
					{Start: "09:00", End: "17:00"},
				},
			},
		}

		assert.NotNil(t, policy.TimeRestrictions)
		assert.Equal(t, []int{1, 2, 3, 4, 5}, policy.TimeRestrictions.AllowedDays)
		assert.Len(t, policy.TimeRestrictions.AllowedTimeRanges, 1)
	})
}

func TestRole(t *testing.T) {
	t.Run("CreateRole", func(t *testing.T) {
		policy := &AccessPolicy{
			Name:        "role-policy",
			Permissions: []Permission{PermissionRead},
		}

		role := &Role{
			Name:        "test-role",
			Description: "Test role",
			Policies:    []*AccessPolicy{policy},
			Active:      true,
		}

		assert.Equal(t, "test-role", role.Name)
		assert.Equal(t, "Test role", role.Description)
		assert.Len(t, role.Policies, 1)
		assert.True(t, role.Active)
	})
}

func TestAccessControlEntry(t *testing.T) {
	t.Run("CreateEntry", func(t *testing.T) {
		entry := &AccessControlEntry{
			EntityType: "user",
			EntityID:   "user123",
			Resources: []*Resource{
				NewResource(AccessResourceTypeTopic, "user-topic"),
			},
			Permissions: []Permission{PermissionPublish},
			Effect:      "allow",
		}

		assert.Equal(t, "user", entry.EntityType)
		assert.Equal(t, "user123", entry.EntityID)
		assert.Len(t, entry.Resources, 1)
		assert.Len(t, entry.Permissions, 1)
		assert.Equal(t, "allow", entry.Effect)
	})

	t.Run("ExpiredEntry", func(t *testing.T) {
		expiredTime := time.Now().Add(-1 * time.Hour)
		entry := &AccessControlEntry{
			EntityType: "user",
			EntityID:   "user123",
			ExpiresAt:  &expiredTime,
		}

		assert.True(t, entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt))
	})
}

func TestAccessContext(t *testing.T) {
	t.Run("NewAccessContext", func(t *testing.T) {
		roles := []string{"admin", "publisher"}
		ctx := NewAccessContext("user123", roles)

		assert.Equal(t, "user123", ctx.UserID)
		assert.Equal(t, roles, ctx.Roles)
		assert.NotNil(t, ctx.Context)
		assert.False(t, ctx.RequestTime.IsZero())
	})

	t.Run("WithAuthContext", func(t *testing.T) {
		authCtx := &AuthContext{
			Credentials: &AuthCredentials{
				UserID: "user123",
			},
		}

		accessCtx := NewAccessContext("user123", []string{"admin"})
		accessCtx.AuthContext = authCtx

		assert.Equal(t, authCtx, accessCtx.AuthContext)
	})
}

func TestMemoryAccessControlProvider(t *testing.T) {
	t.Run("NewProvider", func(t *testing.T) {
		provider := NewMemoryAccessControlProvider(5 * time.Minute)
		assert.Equal(t, "memory", provider.Name())
		assert.NotNil(t, provider.cache)
		assert.True(t, provider.IsHealthy(context.Background()))
	})

	t.Run("CheckAccessWithEntries", func(t *testing.T) {
		provider := NewMemoryAccessControlProvider(0) // No cache for testing

		// Add an access control entry
		entry := &AccessControlEntry{
			EntityType: "user",
			EntityID:   "user123",
			Resources: []*Resource{
				NewResource(AccessResourceTypeTopic, "test-topic"),
			},
			Permissions: []Permission{PermissionPublish},
			Effect:      "allow",
		}

		err := provider.AddAccessControlEntry(context.Background(), entry)
		require.NoError(t, err)

		accessCtx := NewAccessContext("user123", nil)
		resource := NewResource(AccessResourceTypeTopic, "test-topic")

		decision, err := provider.CheckAccess(context.Background(), accessCtx, resource, PermissionPublish)
		require.NoError(t, err)
		assert.True(t, decision.Allowed)
		assert.Equal(t, "Access allow by entry user123", decision.Reason)
	})

	t.Run("CheckAccessWithRoles", func(t *testing.T) {
		provider := NewMemoryAccessControlProvider(0)

		// Create a role with a policy
		policy := &AccessPolicy{
			Name: "publisher-policy",
			Resources: []*Resource{
				ResourcePattern(AccessResourceTypeTopic, "orders.*"),
			},
			Permissions: []Permission{PermissionPublish},
			Priority:    100,
			Enabled:     true,
		}

		role := &Role{
			Name:     "publisher",
			Policies: []*AccessPolicy{policy},
			Active:   true,
		}

		err := provider.AddRole(context.Background(), role)
		require.NoError(t, err)

		accessCtx := NewAccessContext("user123", []string{"publisher"})
		resource := NewResource(AccessResourceTypeTopic, "orders.processed")

		decision, err := provider.CheckAccess(context.Background(), accessCtx, resource, PermissionPublish)
		require.NoError(t, err)
		assert.True(t, decision.Allowed)
		assert.Equal(t, "Access allowed by policy publisher-policy in role publisher", decision.Reason)
	})

	t.Run("DefaultDeny", func(t *testing.T) {
		provider := NewMemoryAccessControlProvider(0)

		accessCtx := NewAccessContext("user123", nil)
		resource := NewResource(AccessResourceTypeTopic, "restricted-topic")

		decision, err := provider.CheckAccess(context.Background(), accessCtx, resource, PermissionPublish)
		require.NoError(t, err)
		assert.False(t, decision.Allowed)
		assert.Equal(t, "Access denied by default", decision.Reason)
	})

	t.Run("GetPermissions", func(t *testing.T) {
		provider := NewMemoryAccessControlProvider(0)

		// Add role with permissions
		policy := &AccessPolicy{
			Name: "multi-permission-policy",
			Resources: []*Resource{
				NewResource(AccessResourceTypeTopic, "shared-topic"),
			},
			Permissions: []Permission{PermissionRead, PermissionWrite, PermissionPublish},
			Enabled:     true,
		}

		role := &Role{
			Name:     "multi-role",
			Policies: []*AccessPolicy{policy},
			Active:   true,
		}

		err := provider.AddRole(context.Background(), role)
		require.NoError(t, err)

		resource := NewResource(AccessResourceTypeTopic, "shared-topic")
		permissions, err := provider.GetPermissions(context.Background(), "user123", []string{"multi-role"}, resource)
		require.NoError(t, err)

		assert.Len(t, permissions, 3)
		assert.Contains(t, permissions, PermissionRead)
		assert.Contains(t, permissions, PermissionWrite)
		assert.Contains(t, permissions, PermissionPublish)
	})
}

func TestAccessControlCache(t *testing.T) {
	t.Run("CacheOperations", func(t *testing.T) {
		cache := NewAccessControlCache(100 * time.Millisecond)

		decision := &AccessDecision{
			Allowed:    true,
			Reason:     "cached decision",
			Resource:   NewResource(AccessResourceTypeTopic, "cached-topic"),
			Permission: PermissionRead,
			Timestamp:  time.Now(),
		}

		// Set and get
		cache.Set("user123", "cached-topic", PermissionRead, decision)
		cached, found := cache.Get("user123", "cached-topic", PermissionRead)

		assert.True(t, found)
		assert.Equal(t, decision.Allowed, cached.Allowed)
		assert.Equal(t, decision.Reason, cached.Reason)
	})

	t.Run("CacheExpiration", func(t *testing.T) {
		cache := NewAccessControlCache(50 * time.Millisecond)

		decision := &AccessDecision{
			Allowed:    true,
			Reason:     "expiring decision",
			Resource:   NewResource(AccessResourceTypeTopic, "expiring-topic"),
			Permission: PermissionRead,
			Timestamp:  time.Now(),
		}

		cache.Set("user123", "expiring-topic", PermissionRead, decision)

		// Should be found immediately
		_, found := cache.Get("user123", "expiring-topic", PermissionRead)
		assert.True(t, found)

		// Wait for expiration
		time.Sleep(100 * time.Millisecond)

		// Should not be found after expiration
		_, found = cache.Get("user123", "expiring-topic", PermissionRead)
		assert.False(t, found)
	})

	t.Run("CacheClear", func(t *testing.T) {
		cache := NewAccessControlCache(100 * time.Millisecond)

		decision := &AccessDecision{
			Allowed:    true,
			Resource:   NewResource(AccessResourceTypeTopic, "clearable-topic"),
			Permission: PermissionRead,
			Timestamp:  time.Now(),
		}

		cache.Set("user123", "clearable-topic", PermissionRead, decision)

		// Should be found before clear
		_, found := cache.Get("user123", "clearable-topic", PermissionRead)
		assert.True(t, found)

		// Clear cache
		cache.Clear()

		// Should not be found after clear
		_, found = cache.Get("user123", "clearable-topic", PermissionRead)
		assert.False(t, found)
	})
}

func TestAccessControlManager(t *testing.T) {
	t.Run("CreateManager", func(t *testing.T) {
		provider := NewMemoryAccessControlProvider(0)
		authManager, err := NewAuthManager(&AuthManagerConfig{
			DefaultProvider: "jwt",
			Providers: map[string]AuthProvider{
				"jwt": &MockAuthProvider{name: "jwt"},
			},
		})
		require.NoError(t, err)

		config := &AccessControlManagerConfig{
			Provider:    provider,
			AuthManager: authManager,
			Enabled:     true,
			DefaultDeny: true,
		}

		manager, err := NewAccessControlManager(config)
		require.NoError(t, err)
		assert.True(t, manager.IsHealthy(context.Background()))
	})

	t.Run("CheckAccessDisabled", func(t *testing.T) {
		provider := NewMemoryAccessControlProvider(0)
		authManager, err := NewAuthManager(&AuthManagerConfig{
			DefaultProvider: "jwt",
			Providers: map[string]AuthProvider{
				"jwt": &MockAuthProvider{name: "jwt"},
			},
		})
		require.NoError(t, err)

		config := &AccessControlManagerConfig{
			Provider:    provider,
			AuthManager: authManager,
			Enabled:     false, // Disabled
		}

		manager, err := NewAccessControlManager(config)
		require.NoError(t, err)

		credentials := &AuthCredentials{Type: AuthTypeNone}
		resource := NewResource(AccessResourceTypeTopic, "any-topic")

		decision, err := manager.CheckAccess(context.Background(), credentials, resource, PermissionPublish)
		require.NoError(t, err)
		assert.True(t, decision.Allowed)
		assert.Equal(t, "Access control disabled", decision.Reason)
	})

	t.Run("CheckTopicAccess", func(t *testing.T) {
		provider := NewMemoryAccessControlProvider(0)
		authManager, err := NewAuthManager(&AuthManagerConfig{
			DefaultProvider: "jwt",
			Providers: map[string]AuthProvider{
				"jwt": &MockAuthProvider{name: "jwt"},
			},
		})
		require.NoError(t, err)

		config := &AccessControlManagerConfig{
			Provider:    provider,
			AuthManager: authManager,
			Enabled:     true,
		}

		manager, err := NewAccessControlManager(config)
		require.NoError(t, err)

		// Add a policy
		policy := &AccessPolicy{
			Name: "topic-access-policy",
			Resources: []*Resource{
				NewResource(AccessResourceTypeTopic, "test-topic"),
			},
			Permissions: []Permission{PermissionPublish},
			Enabled:     true,
		}

		err = manager.AddPolicy(context.Background(), policy)
		require.NoError(t, err)

		credentials := &AuthCredentials{
			Type:   AuthTypeNone,
			UserID: "user123",
			Scopes: []string{"publisher"},
		}

		decision, err := manager.CheckTopicAccess(context.Background(), credentials, "test-topic", PermissionPublish)
		require.NoError(t, err)
		// Note: This will be denied by default since we haven't set up role-based access
		assert.False(t, decision.Allowed)
	})

	t.Run("CreateDefaultRoles", func(t *testing.T) {
		provider := NewMemoryAccessControlProvider(0)
		authManager, err := NewAuthManager(&AuthManagerConfig{
			DefaultProvider: "jwt",
			Providers: map[string]AuthProvider{
				"jwt": &MockAuthProvider{name: "jwt"},
			},
		})
		require.NoError(t, err)

		config := &AccessControlManagerConfig{
			Provider:    provider,
			AuthManager: authManager,
			Enabled:     true,
		}

		manager, err := NewAccessControlManager(config)
		require.NoError(t, err)

		err = manager.CreateDefaultRoles(context.Background())
		require.NoError(t, err)

		// Verify roles were created
		resource := NewResource(AccessResourceTypeTopic, "any-topic")
		permissions, err := manager.GetPermissions(context.Background(), "admin", []string{"admin"}, resource)
		require.NoError(t, err)
		assert.Contains(t, permissions, PermissionManage)
		assert.Contains(t, permissions, PermissionPublish)
		assert.Contains(t, permissions, PermissionConsume)
	})
}

func TestAccessControlMiddleware(t *testing.T) {
	t.Run("NewMiddleware", func(t *testing.T) {
		provider := NewMemoryAccessControlProvider(0)
		authManager, err := NewAuthManager(&AuthManagerConfig{
			DefaultProvider: "jwt",
			Providers: map[string]AuthProvider{
				"jwt": &MockAuthProvider{name: "jwt"},
			},
		})
		require.NoError(t, err)

		accessManager, err := NewAccessControlManager(&AccessControlManagerConfig{
			Provider:    provider,
			AuthManager: authManager,
			Enabled:     false, // Disabled for middleware testing
		})
		require.NoError(t, err)

		config := &AccessControlMiddlewareConfig{
			AccessControlManager: accessManager,
			Required:             false,
		}

		middleware := NewAccessControlMiddleware(config)
		assert.Equal(t, "access-control", middleware.Name())
	})

	t.Run("MiddlewareWrap", func(t *testing.T) {
		provider := NewMemoryAccessControlProvider(0)
		authManager, err := NewAuthManager(&AuthManagerConfig{
			DefaultProvider: "jwt",
			Providers: map[string]AuthProvider{
				"jwt": &MockAuthProvider{name: "jwt"},
			},
		})
		require.NoError(t, err)

		accessManager, err := NewAccessControlManager(&AccessControlManagerConfig{
			Provider:    provider,
			AuthManager: authManager,
			Enabled:     false, // Disabled to allow access
		})
		require.NoError(t, err)

		config := &AccessControlMiddlewareConfig{
			AccessControlManager: accessManager,
			Required:             false,
		}

		middleware := NewAccessControlMiddleware(config)

		// Create a mock next handler
		mockHandler := &MockMessageHandler{}
		wrappedHandler := middleware.Wrap(mockHandler)

		assert.NotNil(t, wrappedHandler)
		assert.IsType(t, &accessControlMessageHandler{}, wrappedHandler)
	})
}

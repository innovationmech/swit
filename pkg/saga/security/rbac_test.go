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
	"fmt"
	"testing"
	"time"
)

func TestPermission_Validate(t *testing.T) {
	tests := []struct {
		name       string
		permission Permission
		wantErr    bool
	}{
		{
			name:       "valid permission",
			permission: PermissionSagaExecute,
			wantErr:    false,
		},
		{
			name:       "empty permission",
			permission: "",
			wantErr:    true,
		},
		{
			name:       "invalid format - no colon",
			permission: "invalidformat",
			wantErr:    true,
		},
		{
			name:       "invalid format - empty resource",
			permission: ":action",
			wantErr:    true,
		},
		{
			name:       "invalid format - empty action",
			permission: "resource:",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.permission.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Permission.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPermission_Matches(t *testing.T) {
	tests := []struct {
		name       string
		permission Permission
		other      Permission
		want       bool
	}{
		{
			name:       "exact match",
			permission: PermissionSagaExecute,
			other:      PermissionSagaExecute,
			want:       true,
		},
		{
			name:       "wildcard action matches",
			permission: "saga:*",
			other:      PermissionSagaExecute,
			want:       true,
		},
		{
			name:       "different resource",
			permission: PermissionSagaExecute,
			other:      PermissionStepExecute,
			want:       false,
		},
		{
			name:       "different action",
			permission: PermissionSagaExecute,
			other:      PermissionSagaRead,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.permission.Matches(tt.other); got != tt.want {
				t.Errorf("Permission.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPermissionSet(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		ps := NewPermissionSet()

		// Test Add
		ps.Add(PermissionSagaExecute, PermissionSagaRead)
		if ps.Size() != 2 {
			t.Errorf("expected size 2, got %d", ps.Size())
		}

		// Test Has
		if !ps.Has(PermissionSagaExecute) {
			t.Error("expected to have PermissionSagaExecute")
		}

		// Test Remove
		ps.Remove(PermissionSagaRead)
		if ps.Size() != 1 {
			t.Errorf("expected size 1 after removal, got %d", ps.Size())
		}

		// Test Clear
		ps.Clear()
		if ps.Size() != 0 {
			t.Errorf("expected size 0 after clear, got %d", ps.Size())
		}
	})

	t.Run("HasAll", func(t *testing.T) {
		ps := NewPermissionSet(PermissionSagaExecute, PermissionSagaRead, PermissionSagaCancel)

		if !ps.HasAll(PermissionSagaExecute, PermissionSagaRead) {
			t.Error("expected HasAll to return true")
		}

		if ps.HasAll(PermissionSagaExecute, PermissionStepExecute) {
			t.Error("expected HasAll to return false")
		}
	})

	t.Run("HasAny", func(t *testing.T) {
		ps := NewPermissionSet(PermissionSagaExecute)

		if !ps.HasAny(PermissionSagaExecute, PermissionSagaRead) {
			t.Error("expected HasAny to return true")
		}

		if ps.HasAny(PermissionSagaRead, PermissionStepExecute) {
			t.Error("expected HasAny to return false")
		}
	})

	t.Run("Clone", func(t *testing.T) {
		original := NewPermissionSet(PermissionSagaExecute, PermissionSagaRead)
		clone := original.Clone()

		clone.Add(PermissionSagaCancel)

		if original.Size() == clone.Size() {
			t.Error("clone should be independent of original")
		}
	})

	t.Run("Union", func(t *testing.T) {
		ps1 := NewPermissionSet(PermissionSagaExecute, PermissionSagaRead)
		ps2 := NewPermissionSet(PermissionSagaRead, PermissionSagaCancel)

		union := ps1.Union(ps2)

		if union.Size() != 3 {
			t.Errorf("expected union size 3, got %d", union.Size())
		}
	})

	t.Run("Intersect", func(t *testing.T) {
		ps1 := NewPermissionSet(PermissionSagaExecute, PermissionSagaRead)
		ps2 := NewPermissionSet(PermissionSagaRead, PermissionSagaCancel)

		intersect := ps1.Intersect(ps2)

		if intersect.Size() != 1 {
			t.Errorf("expected intersect size 1, got %d", intersect.Size())
		}
		if !intersect.Has(PermissionSagaRead) {
			t.Error("expected intersect to have PermissionSagaRead")
		}
	})

	t.Run("Difference", func(t *testing.T) {
		ps1 := NewPermissionSet(PermissionSagaExecute, PermissionSagaRead)
		ps2 := NewPermissionSet(PermissionSagaRead, PermissionSagaCancel)

		diff := ps1.Difference(ps2)

		if diff.Size() != 1 {
			t.Errorf("expected difference size 1, got %d", diff.Size())
		}
		if !diff.Has(PermissionSagaExecute) {
			t.Error("expected difference to have PermissionSagaExecute")
		}
	})
}

func TestRole(t *testing.T) {
	t.Run("creation", func(t *testing.T) {
		role := NewRole("test-role", "Test Role", PermissionSagaExecute, PermissionSagaRead)

		if role.Name != "test-role" {
			t.Errorf("expected name 'test-role', got %s", role.Name)
		}
		if role.Permissions.Size() != 2 {
			t.Errorf("expected 2 permissions, got %d", role.Permissions.Size())
		}
	})

	t.Run("add and remove permissions", func(t *testing.T) {
		role := NewRole("test-role", "Test Role")

		role.AddPermission(PermissionSagaExecute)
		if !role.HasPermission(PermissionSagaExecute) {
			t.Error("expected role to have permission after adding")
		}

		role.RemovePermission(PermissionSagaExecute)
		if role.HasPermission(PermissionSagaExecute) {
			t.Error("expected role to not have permission after removing")
		}
	})

	t.Run("parent roles", func(t *testing.T) {
		role := NewRole("test-role", "Test Role")

		role.AddParent("parent-role")
		if len(role.ParentRoles) != 1 {
			t.Errorf("expected 1 parent role, got %d", len(role.ParentRoles))
		}

		// Adding same parent twice should not duplicate
		role.AddParent("parent-role")
		if len(role.ParentRoles) != 1 {
			t.Errorf("expected 1 parent role after duplicate add, got %d", len(role.ParentRoles))
		}

		role.RemoveParent("parent-role")
		if len(role.ParentRoles) != 0 {
			t.Errorf("expected 0 parent roles after removal, got %d", len(role.ParentRoles))
		}
	})

	t.Run("clone", func(t *testing.T) {
		original := NewRole("test-role", "Test Role", PermissionSagaExecute)
		original.AddParent("parent-role")
		original.Metadata["key"] = "value"

		clone := original.Clone()

		if clone.Name != original.Name {
			t.Error("clone should have same name")
		}
		if clone.Permissions.Size() != original.Permissions.Size() {
			t.Error("clone should have same permissions")
		}

		clone.AddPermission(PermissionSagaRead)
		if original.Permissions.Size() == clone.Permissions.Size() {
			t.Error("clone should be independent of original")
		}
	})
}

func TestUserRoles(t *testing.T) {
	t.Run("creation", func(t *testing.T) {
		ur := NewUserRoles("user-123", "admin", "operator")

		if ur.UserID != "user-123" {
			t.Errorf("expected user ID 'user-123', got %s", ur.UserID)
		}
		if len(ur.Roles) != 2 {
			t.Errorf("expected 2 roles, got %d", len(ur.Roles))
		}
	})

	t.Run("add and remove roles", func(t *testing.T) {
		ur := NewUserRoles("user-123")

		ur.AddRole("admin")
		if !ur.HasRole("admin") {
			t.Error("expected user to have admin role after adding")
		}

		// Adding duplicate should not create duplicate
		ur.AddRole("admin")
		if len(ur.Roles) != 1 {
			t.Errorf("expected 1 role after duplicate add, got %d", len(ur.Roles))
		}

		ur.RemoveRole("admin")
		if ur.HasRole("admin") {
			t.Error("expected user to not have admin role after removing")
		}
	})
}

func TestRBACManager_RoleManagement(t *testing.T) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: true,
		CacheTTL:     1 * time.Minute,
		CacheMaxSize: 100,
	})

	t.Run("create role", func(t *testing.T) {
		role := NewRole("custom-role", "Custom Role", PermissionSagaExecute)
		err := manager.CreateRole(role)
		if err != nil {
			t.Fatalf("failed to create role: %v", err)
		}

		retrieved, err := manager.GetRole("custom-role")
		if err != nil {
			t.Fatalf("failed to get role: %v", err)
		}
		if retrieved.Name != "custom-role" {
			t.Errorf("expected role name 'custom-role', got %s", retrieved.Name)
		}
	})

	t.Run("create duplicate role", func(t *testing.T) {
		role := NewRole("duplicate-role", "Duplicate Role")
		_ = manager.CreateRole(role)

		err := manager.CreateRole(role)
		if err == nil {
			t.Error("expected error when creating duplicate role")
		}
	})

	t.Run("update role", func(t *testing.T) {
		role := NewRole("update-role", "Update Role", PermissionSagaRead)
		_ = manager.CreateRole(role)

		role.AddPermission(PermissionSagaExecute)
		err := manager.UpdateRole(role)
		if err != nil {
			t.Fatalf("failed to update role: %v", err)
		}

		retrieved, _ := manager.GetRole("update-role")
		if retrieved.Permissions.Size() != 2 {
			t.Errorf("expected 2 permissions after update, got %d", retrieved.Permissions.Size())
		}
	})

	t.Run("delete role", func(t *testing.T) {
		role := NewRole("delete-role", "Delete Role")
		_ = manager.CreateRole(role)

		err := manager.DeleteRole("delete-role")
		if err != nil {
			t.Fatalf("failed to delete role: %v", err)
		}

		_, err = manager.GetRole("delete-role")
		if err == nil {
			t.Error("expected error when getting deleted role")
		}
	})

	t.Run("list roles", func(t *testing.T) {
		roles := manager.ListRoles()
		if len(roles) == 0 {
			t.Error("expected at least predefined roles")
		}
	})
}

func TestRBACManager_UserRoleAssignment(t *testing.T) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: false,
	})

	t.Run("assign role", func(t *testing.T) {
		err := manager.AssignRole("user-123", "admin")
		if err != nil {
			t.Fatalf("failed to assign role: %v", err)
		}

		roles, err := manager.GetUserRoles("user-123")
		if err != nil {
			t.Fatalf("failed to get user roles: %v", err)
		}
		if len(roles) != 1 {
			t.Errorf("expected 1 role, got %d", len(roles))
		}
	})

	t.Run("assign non-existent role", func(t *testing.T) {
		err := manager.AssignRole("user-123", "non-existent-role")
		if err == nil {
			t.Error("expected error when assigning non-existent role")
		}
	})

	t.Run("revoke role", func(t *testing.T) {
		_ = manager.AssignRole("user-456", "admin")

		err := manager.RevokeRole("user-456", "admin")
		if err != nil {
			t.Fatalf("failed to revoke role: %v", err)
		}

		roles, _ := manager.GetUserRoles("user-456")
		if len(roles) != 0 {
			t.Errorf("expected 0 roles after revoke, got %d", len(roles))
		}
	})
}

func TestRBACManager_PermissionChecking(t *testing.T) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: false,
	})

	ctx := context.Background()

	// Assign admin role to user
	_ = manager.AssignRole("user-admin", "admin")
	_ = manager.AssignRole("user-viewer", "viewer")

	t.Run("check permission - has permission", func(t *testing.T) {
		err := manager.CheckPermission(ctx, "user-admin", PermissionAdminFull)
		if err != nil {
			t.Errorf("expected admin to have admin:full permission: %v", err)
		}
	})

	t.Run("check permission - no permission", func(t *testing.T) {
		err := manager.CheckPermission(ctx, "user-viewer", PermissionSagaCreate)
		if err == nil {
			t.Error("expected viewer to not have saga:create permission")
		}
	})

	t.Run("check permissions - multiple", func(t *testing.T) {
		err := manager.CheckPermissions(ctx, "user-viewer",
			PermissionSagaRead, PermissionSagaList)
		if err != nil {
			t.Errorf("expected viewer to have read and list permissions: %v", err)
		}
	})

	t.Run("check any permission", func(t *testing.T) {
		err := manager.CheckAnyPermission(ctx, "user-viewer",
			PermissionSagaCreate, PermissionSagaRead)
		if err != nil {
			t.Errorf("expected viewer to have at least one permission: %v", err)
		}
	})
}

func TestRBACManager_PermissionInheritance(t *testing.T) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: false,
	})

	ctx := context.Background()

	// Create parent role
	parentRole := NewRole("parent-role", "Parent Role", PermissionSagaExecute)
	_ = manager.CreateRole(parentRole)

	// Create child role with parent
	childRole := NewRole("child-role", "Child Role", PermissionSagaRead)
	childRole.AddParent("parent-role")
	_ = manager.CreateRole(childRole)

	// Assign child role to user
	_ = manager.AssignRole("user-child", "child-role")

	t.Run("inherited permission", func(t *testing.T) {
		// User should have permission from parent role
		err := manager.CheckPermission(ctx, "user-child", PermissionSagaExecute)
		if err != nil {
			t.Error("expected user to have inherited permission from parent role")
		}

		// User should also have direct permission
		err = manager.CheckPermission(ctx, "user-child", PermissionSagaRead)
		if err != nil {
			t.Error("expected user to have direct permission from child role")
		}
	})

	t.Run("get user permissions includes inherited", func(t *testing.T) {
		permissions, err := manager.GetUserPermissions("user-child")
		if err != nil {
			t.Fatalf("failed to get user permissions: %v", err)
		}

		if !permissions.Has(PermissionSagaExecute) {
			t.Error("expected permissions to include inherited permission")
		}
		if !permissions.Has(PermissionSagaRead) {
			t.Error("expected permissions to include direct permission")
		}
	})
}

func TestRBACManager_CircularInheritance(t *testing.T) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: false,
	})

	// Create role A
	roleA := NewRole("role-a", "Role A")
	_ = manager.CreateRole(roleA)

	// Create role B with A as parent
	roleB := NewRole("role-b", "Role B")
	roleB.AddParent("role-a")
	_ = manager.CreateRole(roleB)

	// Try to update role A to have B as parent (creates cycle)
	roleA.AddParent("role-b")
	err := manager.UpdateRole(roleA)

	if err == nil {
		t.Error("expected error when creating circular inheritance")
	}
}

func TestRBACManager_Cache(t *testing.T) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: true,
		CacheTTL:     1 * time.Minute,
		CacheMaxSize: 100,
	})

	_ = manager.AssignRole("user-cached", "admin")

	t.Run("first call populates cache", func(t *testing.T) {
		_, err := manager.GetUserPermissions("user-cached")
		if err != nil {
			t.Fatalf("failed to get user permissions: %v", err)
		}

		stats := manager.GetCacheStats()
		if stats["enabled"] != true {
			t.Error("expected cache to be enabled")
		}
	})

	t.Run("second call uses cache", func(t *testing.T) {
		_, err := manager.GetUserPermissions("user-cached")
		if err != nil {
			t.Fatalf("failed to get user permissions: %v", err)
		}

		stats := manager.GetCacheStats()
		if stats["hits"].(uint64) == 0 {
			t.Error("expected cache hit on second call")
		}
	})

	t.Run("cache invalidated on role assignment", func(t *testing.T) {
		_ = manager.AssignRole("user-cached", "operator")

		stats := manager.GetCacheStats()
		// Cache for user should be invalidated
		if size, ok := stats["size"].(int); ok && size > 0 {
			// Check if permissions changed
			perms, _ := manager.GetUserPermissions("user-cached")
			if !perms.Has(PermissionSagaExecute) {
				t.Error("expected user to have operator permissions")
			}
		}
	})
}

func TestRBACMiddleware(t *testing.T) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: false,
	})

	middleware := NewRBACMiddleware(&RBACMiddlewareConfig{
		RBACManager: manager,
	})

	// Assign roles
	_ = manager.AssignRole("user-admin", "admin")
	_ = manager.AssignRole("user-viewer", "viewer")

	t.Run("require permission - with auth context", func(t *testing.T) {
		authCtx := &AuthContext{
			Credentials: &AuthCredentials{
				UserID: "user-admin",
			},
		}
		ctx := ContextWithAuth(context.Background(), authCtx)

		err := middleware.RequirePermission(ctx, PermissionAdminFull)
		if err != nil {
			t.Errorf("expected admin to have permission: %v", err)
		}
	})

	t.Run("require permission - without auth context", func(t *testing.T) {
		ctx := context.Background()

		err := middleware.RequirePermission(ctx, PermissionAdminFull)
		if err == nil {
			t.Error("expected error when no auth context")
		}
	})

	t.Run("require role", func(t *testing.T) {
		authCtx := &AuthContext{
			Credentials: &AuthCredentials{
				UserID: "user-admin",
			},
		}
		ctx := ContextWithAuth(context.Background(), authCtx)

		err := middleware.RequireRole(ctx, "admin")
		if err != nil {
			t.Errorf("expected user to have admin role: %v", err)
		}
	})

	t.Run("has permission helper", func(t *testing.T) {
		authCtx := &AuthContext{
			Credentials: &AuthCredentials{
				UserID: "user-viewer",
			},
		}
		ctx := ContextWithAuth(context.Background(), authCtx)

		if !middleware.HasPermission(ctx, PermissionSagaRead) {
			t.Error("expected viewer to have read permission")
		}

		if middleware.HasPermission(ctx, PermissionSagaCreate) {
			t.Error("expected viewer to not have create permission")
		}
	})
}

func TestRBACMiddleware_RequirePermissions(t *testing.T) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: false,
	})

	middleware := NewRBACMiddleware(&RBACMiddlewareConfig{
		RBACManager: manager,
	})

	_ = manager.AssignRole("user-operator", "operator")

	authCtx := &AuthContext{
		Credentials: &AuthCredentials{
			UserID: "user-operator",
		},
	}
	ctx := ContextWithAuth(context.Background(), authCtx)

	err := middleware.RequirePermissions(ctx, PermissionSagaExecute, PermissionSagaRead)
	if err != nil {
		t.Errorf("expected operator to have permissions: %v", err)
	}
}

func TestRBACMiddleware_RequireAnyPermission(t *testing.T) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: false,
	})

	middleware := NewRBACMiddleware(&RBACMiddlewareConfig{
		RBACManager: manager,
	})

	_ = manager.AssignRole("user-viewer", "viewer")

	authCtx := &AuthContext{
		Credentials: &AuthCredentials{
			UserID: "user-viewer",
		},
	}
	ctx := ContextWithAuth(context.Background(), authCtx)

	err := middleware.RequireAnyPermission(ctx, PermissionSagaCreate, PermissionSagaRead)
	if err != nil {
		t.Errorf("expected viewer to have at least one permission: %v", err)
	}
}

func TestRBACMiddleware_GetUserPermissions(t *testing.T) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: false,
	})

	middleware := NewRBACMiddleware(&RBACMiddlewareConfig{
		RBACManager: manager,
	})

	_ = manager.AssignRole("user-admin", "admin")

	authCtx := &AuthContext{
		Credentials: &AuthCredentials{
			UserID: "user-admin",
		},
	}
	ctx := ContextWithAuth(context.Background(), authCtx)

	perms, err := middleware.GetUserPermissions(ctx)
	if err != nil {
		t.Fatalf("failed to get user permissions: %v", err)
	}

	if !perms.Has(PermissionAdminFull) {
		t.Error("expected admin to have admin:full permission")
	}
}

func TestRBACMiddleware_Helpers(t *testing.T) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: false,
	})

	middleware := NewRBACMiddleware(&RBACMiddlewareConfig{
		RBACManager: manager,
	})

	_ = manager.AssignRole("user-operator", "operator")

	authCtx := &AuthContext{
		Credentials: &AuthCredentials{
			UserID: "user-operator",
		},
	}
	ctx := ContextWithAuth(context.Background(), authCtx)

	t.Run("has any permission", func(t *testing.T) {
		if !middleware.HasAnyPermission(ctx, PermissionSagaCreate, PermissionSagaExecute) {
			t.Error("expected operator to have at least one permission")
		}
	})

	t.Run("has all permissions", func(t *testing.T) {
		if !middleware.HasAllPermissions(ctx, PermissionSagaExecute, PermissionSagaRead) {
			t.Error("expected operator to have all permissions")
		}
	})

	t.Run("has role", func(t *testing.T) {
		if !middleware.HasRole(ctx, "operator") {
			t.Error("expected user to have operator role")
		}
	})

	t.Run("rbac manager", func(t *testing.T) {
		if middleware.RBACManager() != manager {
			t.Error("expected middleware to return correct RBAC manager")
		}
	})
}

func TestPermissionCache_Operations(t *testing.T) {
	cache := NewPermissionCache(&PermissionCacheConfig{
		TTL:             1 * time.Minute,
		MaxSize:         10,
		CleanupInterval: 100 * time.Millisecond,
	})
	defer cache.Close()

	t.Run("size", func(t *testing.T) {
		if cache.Size() != 0 {
			t.Errorf("expected empty cache, got size %d", cache.Size())
		}

		perms := NewPermissionSet(PermissionSagaExecute)
		cache.Set("user-1", perms)

		if cache.Size() != 1 {
			t.Errorf("expected cache size 1, got %d", cache.Size())
		}
	})

	t.Run("eviction on max size", func(t *testing.T) {
		cache.Clear()

		// Fill cache to max size
		for i := 0; i < 10; i++ {
			perms := NewPermissionSet(PermissionSagaExecute)
			cache.Set(fmt.Sprintf("user-%d", i), perms)
		}

		// Adding one more should trigger eviction
		perms := NewPermissionSet(PermissionSagaExecute)
		cache.Set("user-new", perms)

		if cache.Size() > 10 {
			t.Errorf("expected cache size <= 10, got %d", cache.Size())
		}
	})

	t.Run("cleanup expired", func(t *testing.T) {
		shortCache := NewPermissionCache(&PermissionCacheConfig{
			TTL:             10 * time.Millisecond,
			MaxSize:         10,
			CleanupInterval: 50 * time.Millisecond,
		})
		defer shortCache.Close()

		perms := NewPermissionSet(PermissionSagaExecute)
		shortCache.Set("user-expired", perms)

		// Wait for expiration and cleanup
		time.Sleep(200 * time.Millisecond)

		// Try to get expired entry
		_, found := shortCache.Get("user-expired")
		if found {
			t.Error("expected expired entry to be removed")
		}
	})
}

func TestAllPermissions(t *testing.T) {
	allPerms := AllPermissions()
	if len(allPerms) == 0 {
		t.Error("expected AllPermissions to return non-empty slice")
	}

	// Check if it includes some key permissions
	found := false
	for _, p := range allPerms {
		if p == PermissionSagaExecute {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected AllPermissions to include PermissionSagaExecute")
	}
}

func BenchmarkRBACManager_CheckPermission(b *testing.B) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
		CacheMaxSize: 1000,
	})

	_ = manager.AssignRole("bench-user", "admin")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.CheckPermission(ctx, "bench-user", PermissionAdminFull)
	}
}

func BenchmarkRBACManager_GetUserPermissions(b *testing.B) {
	manager := NewRBACManager(&RBACManagerConfig{
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
		CacheMaxSize: 1000,
	})

	_ = manager.AssignRole("bench-user", "admin")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GetUserPermissions("bench-user")
	}
}

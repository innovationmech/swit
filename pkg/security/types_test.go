// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package security

import (
	"testing"
	"time"
)

func TestSecurityContext_IsValid(t *testing.T) {
	tests := []struct {
		name string
		sc   *SecurityContext
		want bool
	}{
		{
			name: "valid context",
			sc: &SecurityContext{
				UserID:    "user123",
				Username:  "testuser",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			want: true,
		},
		{
			name: "nil context",
			sc:   nil,
			want: false,
		},
		{
			name: "empty user ID",
			sc: &SecurityContext{
				UserID:    "",
				Username:  "testuser",
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			want: false,
		},
		{
			name: "expired token",
			sc: &SecurityContext{
				UserID:    "user123",
				Username:  "testuser",
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			want: false,
		},
		{
			name: "zero expiry time (never expires)",
			sc: &SecurityContext{
				UserID:   "user123",
				Username: "testuser",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sc.IsValid(); got != tt.want {
				t.Errorf("SecurityContext.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecurityContext_HasRole(t *testing.T) {
	sc := &SecurityContext{
		UserID: "user123",
		Roles:  []string{"admin", "user", "editor"},
	}

	tests := []struct {
		name string
		role string
		want bool
	}{
		{"has admin role", "admin", true},
		{"has user role", "user", true},
		{"has editor role", "editor", true},
		{"does not have superuser role", "superuser", false},
		{"empty role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sc.HasRole(tt.role); got != tt.want {
				t.Errorf("SecurityContext.HasRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestSecurityContext_HasPermission(t *testing.T) {
	sc := &SecurityContext{
		UserID:      "user123",
		Permissions: []string{"read:users", "write:users", "delete:posts"},
	}

	tests := []struct {
		name       string
		permission string
		want       bool
	}{
		{"has read:users permission", "read:users", true},
		{"has write:users permission", "write:users", true},
		{"has delete:posts permission", "delete:posts", true},
		{"does not have admin permission", "admin:all", false},
		{"empty permission", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sc.HasPermission(tt.permission); got != tt.want {
				t.Errorf("SecurityContext.HasPermission(%q) = %v, want %v", tt.permission, got, tt.want)
			}
		})
	}
}

func TestSecurityContext_HasAnyRole(t *testing.T) {
	sc := &SecurityContext{
		UserID: "user123",
		Roles:  []string{"admin", "user"},
	}

	tests := []struct {
		name  string
		roles []string
		want  bool
	}{
		{"has one of the roles", []string{"admin", "superuser"}, true},
		{"has all of the roles", []string{"admin", "user"}, true},
		{"has none of the roles", []string{"superuser", "moderator"}, false},
		{"empty roles list", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sc.HasAnyRole(tt.roles...); got != tt.want {
				t.Errorf("SecurityContext.HasAnyRole(%v) = %v, want %v", tt.roles, got, tt.want)
			}
		})
	}
}

func TestSecurityContext_HasAllRoles(t *testing.T) {
	sc := &SecurityContext{
		UserID: "user123",
		Roles:  []string{"admin", "user", "editor"},
	}

	tests := []struct {
		name  string
		roles []string
		want  bool
	}{
		{"has all specified roles", []string{"admin", "user"}, true},
		{"has all three roles", []string{"admin", "user", "editor"}, true},
		{"missing one role", []string{"admin", "user", "superuser"}, false},
		{"empty roles list", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sc.HasAllRoles(tt.roles...); got != tt.want {
				t.Errorf("SecurityContext.HasAllRoles(%v) = %v, want %v", tt.roles, got, tt.want)
			}
		})
	}
}

func TestSecurityContext_HasAnyPermission(t *testing.T) {
	sc := &SecurityContext{
		UserID:      "user123",
		Permissions: []string{"read:users", "write:users"},
	}

	tests := []struct {
		name        string
		permissions []string
		want        bool
	}{
		{"has one of the permissions", []string{"read:users", "admin:all"}, true},
		{"has all of the permissions", []string{"read:users", "write:users"}, true},
		{"has none of the permissions", []string{"admin:all", "delete:users"}, false},
		{"empty permissions list", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sc.HasAnyPermission(tt.permissions...); got != tt.want {
				t.Errorf("SecurityContext.HasAnyPermission(%v) = %v, want %v", tt.permissions, got, tt.want)
			}
		})
	}
}

func TestSecurityContext_HasAllPermissions(t *testing.T) {
	sc := &SecurityContext{
		UserID:      "user123",
		Permissions: []string{"read:users", "write:users", "delete:posts"},
	}

	tests := []struct {
		name        string
		permissions []string
		want        bool
	}{
		{"has all specified permissions", []string{"read:users", "write:users"}, true},
		{"has all three permissions", []string{"read:users", "write:users", "delete:posts"}, true},
		{"missing one permission", []string{"read:users", "write:users", "admin:all"}, false},
		{"empty permissions list", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sc.HasAllPermissions(tt.permissions...); got != tt.want {
				t.Errorf("SecurityContext.HasAllPermissions(%v) = %v, want %v", tt.permissions, got, tt.want)
			}
		})
	}
}

func TestSecurityContext_NilChecks(t *testing.T) {
	var sc *SecurityContext

	if sc.IsValid() {
		t.Error("nil SecurityContext should not be valid")
	}

	if sc.HasRole("admin") {
		t.Error("nil SecurityContext should not have any roles")
	}

	if sc.HasPermission("read:users") {
		t.Error("nil SecurityContext should not have any permissions")
	}

	if sc.HasAnyRole("admin", "user") {
		t.Error("nil SecurityContext should not have any roles")
	}

	if sc.HasAllRoles("admin", "user") {
		t.Error("nil SecurityContext should not have all roles")
	}

	if sc.HasAnyPermission("read:users", "write:users") {
		t.Error("nil SecurityContext should not have any permissions")
	}

	if sc.HasAllPermissions("read:users", "write:users") {
		t.Error("nil SecurityContext should not have all permissions")
	}
}

// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestCustomClaims_Valid(t *testing.T) {
	tests := []struct {
		name    string
		claims  *CustomClaims
		wantErr bool
	}{
		{
			name: "valid claims",
			claims: &CustomClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
					NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
				},
				UserID: "user123",
			},
			wantErr: false,
		},
		{
			name: "expired claims",
			claims: &CustomClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
				},
				UserID: "user123",
			},
			wantErr: true,
		},
		{
			name: "not yet valid claims",
			claims: &CustomClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
					NotBefore: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				},
				UserID: "user123",
			},
			wantErr: true,
		},
		{
			name: "missing user ID",
			claims: &CustomClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.claims.Valid()
			if (err != nil) != tt.wantErr {
				t.Errorf("Valid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCustomClaims_HasRole(t *testing.T) {
	claims := &CustomClaims{
		Roles: []string{"admin", "user", "editor"},
	}

	tests := []struct {
		name string
		role string
		want bool
	}{
		{"has admin role", "admin", true},
		{"has user role", "user", true},
		{"does not have viewer role", "viewer", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := claims.HasRole(tt.role); got != tt.want {
				t.Errorf("HasRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomClaims_HasPermission(t *testing.T) {
	claims := &CustomClaims{
		Permissions: []string{"read", "write", "delete"},
	}

	tests := []struct {
		name       string
		permission string
		want       bool
	}{
		{"has read permission", "read", true},
		{"has write permission", "write", true},
		{"does not have execute permission", "execute", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := claims.HasPermission(tt.permission); got != tt.want {
				t.Errorf("HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomClaims_HasAnyRole(t *testing.T) {
	claims := &CustomClaims{
		Roles: []string{"admin", "user"},
	}

	tests := []struct {
		name  string
		roles []string
		want  bool
	}{
		{"has one of the roles", []string{"admin", "superadmin"}, true},
		{"has none of the roles", []string{"viewer", "guest"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := claims.HasAnyRole(tt.roles...); got != tt.want {
				t.Errorf("HasAnyRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomClaims_HasAllRoles(t *testing.T) {
	claims := &CustomClaims{
		Roles: []string{"admin", "user", "editor"},
	}

	tests := []struct {
		name  string
		roles []string
		want  bool
	}{
		{"has all roles", []string{"admin", "user"}, true},
		{"missing one role", []string{"admin", "viewer"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := claims.HasAllRoles(tt.roles...); got != tt.want {
				t.Errorf("HasAllRoles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomClaims_HasAnyPermission(t *testing.T) {
	claims := &CustomClaims{
		Permissions: []string{"read", "write"},
	}

	tests := []struct {
		name        string
		permissions []string
		want        bool
	}{
		{"has one of the permissions", []string{"read", "delete"}, true},
		{"has none of the permissions", []string{"execute", "admin"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := claims.HasAnyPermission(tt.permissions...); got != tt.want {
				t.Errorf("HasAnyPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomClaims_HasAllPermissions(t *testing.T) {
	claims := &CustomClaims{
		Permissions: []string{"read", "write", "delete"},
	}

	tests := []struct {
		name        string
		permissions []string
		want        bool
	}{
		{"has all permissions", []string{"read", "write"}, true},
		{"missing one permission", []string{"read", "execute"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := claims.HasAllPermissions(tt.permissions...); got != tt.want {
				t.Errorf("HasAllPermissions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClaimsBuilder(t *testing.T) {
	now := time.Now()
	builder := NewClaimsBuilder().
		WithUserID("user123").
		WithUsername("testuser").
		WithEmail("test@example.com").
		WithRoles("admin", "user").
		WithPermissions("read", "write").
		WithSubject("subject123").
		WithIssuer("https://auth.example.com").
		WithAudience("https://api.example.com").
		WithExpiresAt(now.Add(1*time.Hour)).
		WithNotBefore(now).
		WithIssuedAt(now).
		WithID("jti123").
		WithMetadata("key1", "value1").
		WithMetadata("key2", 42)

	claims := builder.Build()

	if claims.UserID != "user123" {
		t.Errorf("Expected UserID to be user123, got %s", claims.UserID)
	}
	if claims.Username != "testuser" {
		t.Errorf("Expected Username to be testuser, got %s", claims.Username)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Expected Email to be test@example.com, got %s", claims.Email)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != "admin" {
		t.Errorf("Expected Roles to contain admin and user")
	}
	if len(claims.Permissions) != 2 || claims.Permissions[0] != "read" {
		t.Errorf("Expected Permissions to contain read and write")
	}
	if claims.Subject != "subject123" {
		t.Errorf("Expected Subject to be subject123, got %s", claims.Subject)
	}
	if claims.Issuer != "https://auth.example.com" {
		t.Errorf("Expected Issuer to be https://auth.example.com, got %s", claims.Issuer)
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != "https://api.example.com" {
		t.Errorf("Expected Audience to contain https://api.example.com")
	}
	if claims.ID != "jti123" {
		t.Errorf("Expected ID to be jti123, got %s", claims.ID)
	}
	if claims.Metadata["key1"] != "value1" {
		t.Errorf("Expected Metadata key1 to be value1")
	}
	if claims.Metadata["key2"] != 42 {
		t.Errorf("Expected Metadata key2 to be 42")
	}
}

func TestValidator_ParseCustomClaims(t *testing.T) {
	secret := "test-secret-key-for-testing"
	config := &Config{
		Secret: secret,
	}

	validator, err := NewValidator(config)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	customClaims := &CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:   "user123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"admin"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	parsedClaims, err := validator.ParseCustomClaims(tokenString)
	if err != nil {
		t.Errorf("ParseCustomClaims() error = %v", err)
	}
	if parsedClaims == nil {
		t.Fatal("Expected parsed claims to be non-nil")
	}
	if parsedClaims.UserID != "user123" {
		t.Errorf("Expected UserID to be user123, got %s", parsedClaims.UserID)
	}
}

func TestExtractRolesFromMapClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims jwt.MapClaims
		want   []string
	}{
		{
			name:   "roles as string slice",
			claims: jwt.MapClaims{"roles": []string{"admin", "user"}},
			want:   []string{"admin", "user"},
		},
		{
			name:   "roles as interface slice",
			claims: jwt.MapClaims{"roles": []interface{}{"admin", "user"}},
			want:   []string{"admin", "user"},
		},
		{
			name:   "groups claim",
			claims: jwt.MapClaims{"groups": []string{"group1", "group2"}},
			want:   []string{"group1", "group2"},
		},
		{
			name: "keycloak realm_access format",
			claims: jwt.MapClaims{
				"realm_access.roles": map[string]interface{}{
					"roles": []interface{}{"admin", "user"},
				},
			},
			want: []string{"admin", "user"},
		},
		{
			name:   "no roles",
			claims: jwt.MapClaims{"sub": "user123"},
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractRolesFromMapClaims(tt.claims)
			if !stringSlicesEqual(got, tt.want) {
				t.Errorf("ExtractRolesFromMapClaims() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractPermissionsFromMapClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims jwt.MapClaims
		want   []string
	}{
		{
			name:   "permissions as string slice",
			claims: jwt.MapClaims{"permissions": []string{"read", "write"}},
			want:   []string{"read", "write"},
		},
		{
			name:   "permissions as interface slice",
			claims: jwt.MapClaims{"permissions": []interface{}{"read", "write"}},
			want:   []string{"read", "write"},
		},
		{
			name:   "scopes claim",
			claims: jwt.MapClaims{"scopes": []string{"openid", "profile"}},
			want:   []string{"openid", "profile"},
		},
		{
			name:   "authorities claim",
			claims: jwt.MapClaims{"authorities": []string{"ROLE_ADMIN", "ROLE_USER"}},
			want:   []string{"ROLE_ADMIN", "ROLE_USER"},
		},
		{
			name:   "scopes as string (OAuth2 format)",
			claims: jwt.MapClaims{"scopes": "openid profile"},
			want:   []string{"openid profile"},
		},
		{
			name:   "no permissions",
			claims: jwt.MapClaims{"sub": "user123"},
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPermissionsFromMapClaims(tt.claims)
			if !stringSlicesEqual(got, tt.want) {
				t.Errorf("ExtractPermissionsFromMapClaims() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapClaimsToCustomClaims(t *testing.T) {
	now := time.Now()
	mapClaims := jwt.MapClaims{
		"sub":         "user123",
		"iss":         "https://auth.example.com",
		"aud":         []interface{}{"https://api.example.com"},
		"exp":         float64(now.Add(1 * time.Hour).Unix()),
		"iat":         float64(now.Unix()),
		"nbf":         float64(now.Unix()),
		"jti":         "jti123",
		"user_id":     "user123",
		"username":    "testuser",
		"email":       "test@example.com",
		"roles":       []interface{}{"admin", "user"},
		"permissions": []interface{}{"read", "write"},
		"custom_key":  "custom_value",
	}

	customClaims, err := MapClaimsToCustomClaims(mapClaims)
	if err != nil {
		t.Fatalf("MapClaimsToCustomClaims() error = %v", err)
	}

	if customClaims.Subject != "user123" {
		t.Errorf("Expected Subject to be user123, got %s", customClaims.Subject)
	}
	if customClaims.Issuer != "https://auth.example.com" {
		t.Errorf("Expected Issuer to be https://auth.example.com, got %s", customClaims.Issuer)
	}
	if customClaims.UserID != "user123" {
		t.Errorf("Expected UserID to be user123, got %s", customClaims.UserID)
	}
	if customClaims.Username != "testuser" {
		t.Errorf("Expected Username to be testuser, got %s", customClaims.Username)
	}
	if customClaims.Email != "test@example.com" {
		t.Errorf("Expected Email to be test@example.com, got %s", customClaims.Email)
	}
	if len(customClaims.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(customClaims.Roles))
	}
	if len(customClaims.Permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(customClaims.Permissions))
	}
	if customClaims.Metadata["custom_key"] != "custom_value" {
		t.Error("Expected custom_key in metadata")
	}
}

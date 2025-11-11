// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims extends jwt.RegisteredClaims with additional custom fields.
type CustomClaims struct {
	jwt.RegisteredClaims
	UserID      string                 `json:"user_id,omitempty"`
	Username    string                 `json:"username,omitempty"`
	Email       string                 `json:"email,omitempty"`
	Roles       []string               `json:"roles,omitempty"`
	Permissions []string               `json:"permissions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Valid validates the custom claims.
func (c *CustomClaims) Valid() error {
	// Validate expiration time
	if c.ExpiresAt != nil {
		if !c.ExpiresAt.Time.After(time.Now()) {
			return fmt.Errorf("jwt: token has expired")
		}
	}

	// Validate not before time
	if c.NotBefore != nil {
		if c.NotBefore.Time.After(time.Now()) {
			return fmt.Errorf("jwt: token is not yet valid")
		}
	}

	// Add custom validation logic here if needed
	if c.UserID == "" {
		return fmt.Errorf("jwt: user_id is required")
	}

	return nil
}

// HasRole checks if the claims contain a specific role.
func (c *CustomClaims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the claims contain a specific permission.
func (c *CustomClaims) HasPermission(permission string) bool {
	for _, p := range c.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the claims contain any of the specified roles.
func (c *CustomClaims) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if c.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if the claims contain all of the specified roles.
func (c *CustomClaims) HasAllRoles(roles ...string) bool {
	for _, role := range roles {
		if !c.HasRole(role) {
			return false
		}
	}
	return true
}

// HasAnyPermission checks if the claims contain any of the specified permissions.
func (c *CustomClaims) HasAnyPermission(permissions ...string) bool {
	for _, permission := range permissions {
		if c.HasPermission(permission) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the claims contain all of the specified permissions.
func (c *CustomClaims) HasAllPermissions(permissions ...string) bool {
	for _, permission := range permissions {
		if !c.HasPermission(permission) {
			return false
		}
	}
	return true
}

// ClaimsBuilder provides a fluent interface for building JWT claims.
type ClaimsBuilder struct {
	claims *CustomClaims
}

// NewClaimsBuilder creates a new claims builder.
func NewClaimsBuilder() *ClaimsBuilder {
	return &ClaimsBuilder{
		claims: &CustomClaims{
			Metadata: make(map[string]interface{}),
		},
	}
}

// WithUserID sets the user ID.
func (b *ClaimsBuilder) WithUserID(userID string) *ClaimsBuilder {
	b.claims.UserID = userID
	return b
}

// WithUsername sets the username.
func (b *ClaimsBuilder) WithUsername(username string) *ClaimsBuilder {
	b.claims.Username = username
	return b
}

// WithEmail sets the email.
func (b *ClaimsBuilder) WithEmail(email string) *ClaimsBuilder {
	b.claims.Email = email
	return b
}

// WithRoles sets the roles.
func (b *ClaimsBuilder) WithRoles(roles ...string) *ClaimsBuilder {
	b.claims.Roles = roles
	return b
}

// WithPermissions sets the permissions.
func (b *ClaimsBuilder) WithPermissions(permissions ...string) *ClaimsBuilder {
	b.claims.Permissions = permissions
	return b
}

// WithSubject sets the subject claim.
func (b *ClaimsBuilder) WithSubject(subject string) *ClaimsBuilder {
	b.claims.Subject = subject
	return b
}

// WithIssuer sets the issuer claim.
func (b *ClaimsBuilder) WithIssuer(issuer string) *ClaimsBuilder {
	b.claims.Issuer = issuer
	return b
}

// WithAudience sets the audience claim.
func (b *ClaimsBuilder) WithAudience(audience ...string) *ClaimsBuilder {
	b.claims.Audience = audience
	return b
}

// WithExpiresAt sets the expiration time.
func (b *ClaimsBuilder) WithExpiresAt(expiresAt time.Time) *ClaimsBuilder {
	b.claims.ExpiresAt = jwt.NewNumericDate(expiresAt)
	return b
}

// WithNotBefore sets the not-before time.
func (b *ClaimsBuilder) WithNotBefore(notBefore time.Time) *ClaimsBuilder {
	b.claims.NotBefore = jwt.NewNumericDate(notBefore)
	return b
}

// WithIssuedAt sets the issued-at time.
func (b *ClaimsBuilder) WithIssuedAt(issuedAt time.Time) *ClaimsBuilder {
	b.claims.IssuedAt = jwt.NewNumericDate(issuedAt)
	return b
}

// WithID sets the JWT ID claim.
func (b *ClaimsBuilder) WithID(id string) *ClaimsBuilder {
	b.claims.ID = id
	return b
}

// WithMetadata adds custom metadata.
func (b *ClaimsBuilder) WithMetadata(key string, value interface{}) *ClaimsBuilder {
	if b.claims.Metadata == nil {
		b.claims.Metadata = make(map[string]interface{})
	}
	b.claims.Metadata[key] = value
	return b
}

// Build returns the built claims.
func (b *ClaimsBuilder) Build() *CustomClaims {
	return b.claims
}

// ParseCustomClaims parses a token string into custom claims.
func (v *Validator) ParseCustomClaims(tokenString string) (*CustomClaims, error) {
	claims := &CustomClaims{}
	if err := v.ValidateTokenWithClaims(tokenString, claims); err != nil {
		return nil, err
	}
	return claims, nil
}

// ExtractRolesFromMapClaims extracts roles from jwt.MapClaims.
// It handles different formats used by various identity providers.
func ExtractRolesFromMapClaims(claims jwt.MapClaims) []string {
	// Try common role claim names
	claimNames := []string{"roles", "groups", "realm_access.roles", "resource_access"}

	for _, claimName := range claimNames {
		val, ok := claims[claimName]
		if !ok {
			continue
		}

		switch v := val.(type) {
		case []string:
			return v
		case []interface{}:
			roles := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					roles = append(roles, str)
				}
			}
			return roles
		case map[string]interface{}:
			// For Keycloak realm_access structure
			if rolesVal, ok := v["roles"]; ok {
				if rolesList, ok := rolesVal.([]interface{}); ok {
					roles := make([]string, 0, len(rolesList))
					for _, item := range rolesList {
						if str, ok := item.(string); ok {
							roles = append(roles, str)
						}
					}
					return roles
				}
			}
		}
	}

	return nil
}

// ExtractPermissionsFromMapClaims extracts permissions from jwt.MapClaims.
func ExtractPermissionsFromMapClaims(claims jwt.MapClaims) []string {
	// Try common permission claim names
	claimNames := []string{"permissions", "scopes", "authorities"}

	for _, claimName := range claimNames {
		val, ok := claims[claimName]
		if !ok {
			continue
		}

		switch v := val.(type) {
		case []string:
			return v
		case []interface{}:
			permissions := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					permissions = append(permissions, str)
				}
			}
			return permissions
		case string:
			// Space-separated scopes (OAuth2 format)
			return []string{v}
		}
	}

	return nil
}

// MapClaimsToCustomClaims converts jwt.MapClaims to CustomClaims.
func MapClaimsToCustomClaims(mapClaims jwt.MapClaims) (*CustomClaims, error) {
	claims := &CustomClaims{
		Metadata: make(map[string]interface{}),
	}

	// Extract standard claims
	if sub, ok := GetClaimString(mapClaims, "sub"); ok {
		claims.Subject = sub
	}
	if iss, ok := GetClaimString(mapClaims, "iss"); ok {
		claims.Issuer = iss
	}
	if jti, ok := GetClaimString(mapClaims, "jti"); ok {
		claims.ID = jti
	}

	// Extract time claims
	if exp, ok := GetClaimTime(mapClaims, "exp"); ok {
		claims.ExpiresAt = jwt.NewNumericDate(exp)
	}
	if iat, ok := GetClaimTime(mapClaims, "iat"); ok {
		claims.IssuedAt = jwt.NewNumericDate(iat)
	}
	if nbf, ok := GetClaimTime(mapClaims, "nbf"); ok {
		claims.NotBefore = jwt.NewNumericDate(nbf)
	}

	// Extract audience (can be string or []string)
	if aud, ok := mapClaims["aud"]; ok {
		switch v := aud.(type) {
		case string:
			claims.Audience = []string{v}
		case []string:
			claims.Audience = v
		case []interface{}:
			auds := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					auds = append(auds, str)
				}
			}
			claims.Audience = auds
		}
	}

	// Extract custom claims
	if userID, ok := GetClaimString(mapClaims, "user_id"); ok {
		claims.UserID = userID
	}
	if username, ok := GetClaimString(mapClaims, "username"); ok {
		claims.Username = username
	}
	if email, ok := GetClaimString(mapClaims, "email"); ok {
		claims.Email = email
	}

	// Extract roles
	claims.Roles = ExtractRolesFromMapClaims(mapClaims)

	// Extract permissions
	claims.Permissions = ExtractPermissionsFromMapClaims(mapClaims)

	// Store remaining claims in metadata
	excludedKeys := map[string]bool{
		"sub": true, "iss": true, "aud": true, "exp": true, "nbf": true,
		"iat": true, "jti": true, "user_id": true, "username": true,
		"email": true, "roles": true, "permissions": true, "groups": true,
		"scopes": true, "authorities": true,
	}
	for key, value := range mapClaims {
		if !excludedKeys[key] {
			claims.Metadata[key] = value
		}
	}

	return claims, nil
}

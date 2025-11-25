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
	"time"
)

// TokenType represents the type of security token.
type TokenType string

const (
	// TokenTypeBearer represents a bearer token type (OAuth2/JWT).
	TokenTypeBearer TokenType = "Bearer"
	// TokenTypeBasic represents a basic authentication token type.
	TokenTypeBasic TokenType = "Basic"
	// TokenTypeAPIKey represents an API key token type.
	TokenTypeAPIKey TokenType = "APIKey"
)

// SecurityContext represents the security context for a request.
// It contains authentication and authorization information.
type SecurityContext struct {
	// UserID is the unique identifier of the authenticated user.
	UserID string
	// Username is the username of the authenticated user.
	Username string
	// Email is the email address of the authenticated user.
	Email string
	// Roles is the list of roles assigned to the user.
	Roles []string
	// Permissions is the list of permissions granted to the user.
	Permissions []string
	// TokenType is the type of token used for authentication.
	TokenType TokenType
	// IssuedAt is the time when the token was issued.
	IssuedAt time.Time
	// ExpiresAt is the time when the token expires.
	ExpiresAt time.Time
	// Claims contains additional claims from the token.
	Claims map[string]interface{}
}

// IsValid checks if the security context is valid.
func (sc *SecurityContext) IsValid() bool {
	if sc == nil {
		return false
	}
	if sc.UserID == "" {
		return false
	}
	if !sc.ExpiresAt.IsZero() && time.Now().After(sc.ExpiresAt) {
		return false
	}
	return true
}

// HasRole checks if the user has a specific role.
func (sc *SecurityContext) HasRole(role string) bool {
	if sc == nil {
		return false
	}
	for _, r := range sc.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the user has a specific permission.
func (sc *SecurityContext) HasPermission(permission string) bool {
	if sc == nil {
		return false
	}
	for _, p := range sc.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the specified roles.
func (sc *SecurityContext) HasAnyRole(roles ...string) bool {
	if sc == nil {
		return false
	}
	for _, role := range roles {
		if sc.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if the user has all of the specified roles.
func (sc *SecurityContext) HasAllRoles(roles ...string) bool {
	if sc == nil {
		return false
	}
	for _, role := range roles {
		if !sc.HasRole(role) {
			return false
		}
	}
	return true
}

// HasAnyPermission checks if the user has any of the specified permissions.
func (sc *SecurityContext) HasAnyPermission(permissions ...string) bool {
	if sc == nil {
		return false
	}
	for _, permission := range permissions {
		if sc.HasPermission(permission) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the user has all of the specified permissions.
func (sc *SecurityContext) HasAllPermissions(permissions ...string) bool {
	if sc == nil {
		return false
	}
	for _, permission := range permissions {
		if !sc.HasPermission(permission) {
			return false
		}
	}
	return true
}

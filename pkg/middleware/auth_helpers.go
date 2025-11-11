// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequireRoles creates a middleware that requires the user to have at least one of the specified roles
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userInfo, exists := GetUserInfo(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "user information not found in context",
			})
			c.Abort()
			return
		}

		if !hasAnyRole(userInfo.Roles, roles) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permissions",
				"message": fmt.Sprintf("requires one of roles: %s", strings.Join(roles, ", ")),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAllRoles creates a middleware that requires the user to have all of the specified roles
func RequireAllRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userInfo, exists := GetUserInfo(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "user information not found in context",
			})
			c.Abort()
			return
		}

		if !hasAllRoles(userInfo.Roles, roles) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permissions",
				"message": fmt.Sprintf("requires all roles: %s", strings.Join(roles, ", ")),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireScopes creates a middleware that requires the user to have at least one of the specified scopes
func RequireScopes(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userInfo, exists := GetUserInfo(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "user information not found in context",
			})
			c.Abort()
			return
		}

		if !hasAnyScope(userInfo.Scopes, scopes) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_scope",
				"message": fmt.Sprintf("requires one of scopes: %s", strings.Join(scopes, ", ")),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAllScopes creates a middleware that requires the user to have all of the specified scopes
func RequireAllScopes(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userInfo, exists := GetUserInfo(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "user information not found in context",
			})
			c.Abort()
			return
		}

		if !hasAllScopes(userInfo.Scopes, scopes) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_scope",
				"message": fmt.Sprintf("requires all scopes: %s", strings.Join(scopes, ", ")),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermissions creates a middleware that requires the user to have at least one of the specified permissions
// Note: This is an alias for RequireScopes for semantic clarity
func RequirePermissions(permissions ...string) gin.HandlerFunc {
	return RequireScopes(permissions...)
}

// RequireAllPermissions creates a middleware that requires the user to have all of the specified permissions
// Note: This is an alias for RequireAllScopes for semantic clarity
func RequireAllPermissions(permissions ...string) gin.HandlerFunc {
	return RequireAllScopes(permissions...)
}

// RequireRoleOrScope creates a middleware that requires the user to have at least one of the specified roles OR scopes
func RequireRoleOrScope(rolesOrScopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userInfo, exists := GetUserInfo(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "user information not found in context",
			})
			c.Abort()
			return
		}

		// Check if user has any of the roles or scopes
		if !hasAnyRole(userInfo.Roles, rolesOrScopes) && !hasAnyScope(userInfo.Scopes, rolesOrScopes) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permissions",
				"message": fmt.Sprintf("requires one of roles or scopes: %s", strings.Join(rolesOrScopes, ", ")),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAuthenticatedUser creates a middleware that simply requires any authenticated user
func RequireAuthenticatedUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := GetUserInfo(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "authentication is required to access this resource",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireCustomClaim creates a middleware that requires a custom claim with a specific value
func RequireCustomClaim(claimName string, requiredValue interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := GetClaims(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "claims not found in context",
			})
			c.Abort()
			return
		}

		claimValue, ok := claims[claimName]
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permissions",
				"message": fmt.Sprintf("missing required claim: %s", claimName),
			})
			c.Abort()
			return
		}

		if claimValue != requiredValue {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permissions",
				"message": fmt.Sprintf("claim %s does not match required value", claimName),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireClaimValidator creates a middleware that validates a claim using a custom validator function
func RequireClaimValidator(validator func(claims map[string]interface{}) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := GetClaims(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "authentication_required",
				"message": "claims not found in context",
			})
			c.Abort()
			return
		}

		if err := validator(claims); err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permissions",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// hasAnyRole checks if userRoles contains at least one of the required roles
func hasAnyRole(userRoles, requiredRoles []string) bool {
	if len(requiredRoles) == 0 {
		return true
	}

	roleMap := make(map[string]bool, len(userRoles))
	for _, role := range userRoles {
		roleMap[role] = true
	}

	for _, required := range requiredRoles {
		if roleMap[required] {
			return true
		}
	}

	return false
}

// hasAllRoles checks if userRoles contains all of the required roles
func hasAllRoles(userRoles, requiredRoles []string) bool {
	if len(requiredRoles) == 0 {
		return true
	}

	roleMap := make(map[string]bool, len(userRoles))
	for _, role := range userRoles {
		roleMap[role] = true
	}

	for _, required := range requiredRoles {
		if !roleMap[required] {
			return false
		}
	}

	return true
}

// hasAnyScope checks if userScopes contains at least one of the required scopes
func hasAnyScope(userScopes, requiredScopes []string) bool {
	if len(requiredScopes) == 0 {
		return true
	}

	scopeMap := make(map[string]bool, len(userScopes))
	for _, scope := range userScopes {
		scopeMap[scope] = true
	}

	for _, required := range requiredScopes {
		if scopeMap[required] {
			return true
		}
	}

	return false
}

// hasAllScopes checks if userScopes contains all of the required scopes
func hasAllScopes(userScopes, requiredScopes []string) bool {
	if len(requiredScopes) == 0 {
		return true
	}

	scopeMap := make(map[string]bool, len(userScopes))
	for _, scope := range userScopes {
		scopeMap[scope] = true
	}

	for _, required := range requiredScopes {
		if !scopeMap[required] {
			return false
		}
	}

	return true
}

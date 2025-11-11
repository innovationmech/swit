// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	secjwt "github.com/innovationmech/swit/pkg/security/jwt"
	"github.com/innovationmech/swit/pkg/security/oauth2"
)

const (
	// ContextKeyUserInfo is the key used to store user information in gin.Context
	ContextKeyUserInfo = "oauth2:user_info"
	// ContextKeyClaims is the key used to store JWT claims in gin.Context
	ContextKeyClaims = "oauth2:claims"
	// ContextKeyToken is the key used to store raw token in gin.Context
	ContextKeyToken = "oauth2:token"
	// ContextKeyTokenString is the key used to store raw token string in gin.Context
	ContextKeyTokenString = "oauth2:token_string"
)

// OAuth2MiddlewareConfig holds configuration for OAuth2 middleware
type OAuth2MiddlewareConfig struct {
	// OAuth2Client is the OAuth2 client used for token validation
	OAuth2Client *oauth2.Client

	// JWTValidator is the JWT validator used for local token validation
	JWTValidator *secjwt.Validator

	// UseIntrospection enables token introspection instead of local validation
	UseIntrospection bool

	// SkipPaths is a list of paths that should skip authentication
	SkipPaths []string

	// ErrorHandler is a custom error handler function
	ErrorHandler func(*gin.Context, error)

	// TokenExtractor is a custom function to extract token from request
	TokenExtractor func(*gin.Context) (string, error)

	// CookieName is the name of the cookie containing the token (optional)
	CookieName string

	// Optional makes authentication optional (doesn't abort on missing/invalid token)
	Optional bool
}

// UserInfo represents authenticated user information stored in context
type UserInfo struct {
	Subject  string                 `json:"sub"`
	Username string                 `json:"username,omitempty"`
	Email    string                 `json:"email,omitempty"`
	Roles    []string               `json:"roles,omitempty"`
	Scopes   []string               `json:"scopes,omitempty"`
	Claims   map[string]interface{} `json:"claims,omitempty"`
}

// OAuth2Middleware creates an OAuth2 authentication middleware with default configuration
func OAuth2Middleware(client *oauth2.Client, validator *secjwt.Validator) gin.HandlerFunc {
	config := &OAuth2MiddlewareConfig{
		OAuth2Client: client,
		JWTValidator: validator,
	}
	return OAuth2MiddlewareWithConfig(config)
}

// OAuth2MiddlewareWithConfig creates an OAuth2 authentication middleware with custom configuration
func OAuth2MiddlewareWithConfig(config *OAuth2MiddlewareConfig) gin.HandlerFunc {
	if config == nil {
		panic("oauth2: middleware config cannot be nil")
	}

	// Set default error handler
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultOAuth2ErrorHandler
	}

	// Set default token extractor
	if config.TokenExtractor == nil {
		config.TokenExtractor = defaultTokenExtractor(config.CookieName)
	}

	return func(c *gin.Context) {
		// Check if path should skip authentication
		if shouldSkipPath(c.Request.URL.Path, config.SkipPaths) {
			c.Next()
			return
		}

		// Extract token from request
		tokenString, err := config.TokenExtractor(c)
		if err != nil {
			if config.Optional {
				c.Next()
				return
			}
			config.ErrorHandler(c, fmt.Errorf("oauth2: %w", err))
			return
		}

		// Store raw token string in context
		c.Set(ContextKeyTokenString, tokenString)

		// Validate token
		userInfo, claims, err := validateOAuth2Token(c.Request.Context(), config, tokenString)
		if err != nil {
			if config.Optional {
				c.Next()
				return
			}
			config.ErrorHandler(c, err)
			return
		}

		// Store user info and claims in context
		c.Set(ContextKeyUserInfo, userInfo)
		c.Set(ContextKeyClaims, claims)

		c.Next()
	}
}

// OptionalAuth creates an optional authentication middleware
// It extracts and validates tokens but doesn't abort on failure
func OptionalAuth(client *oauth2.Client, validator *secjwt.Validator) gin.HandlerFunc {
	config := &OAuth2MiddlewareConfig{
		OAuth2Client: client,
		JWTValidator: validator,
		Optional:     true,
	}
	return OAuth2MiddlewareWithConfig(config)
}

// validateOAuth2Token validates the token using either introspection or local JWT validation
func validateOAuth2Token(ctx context.Context, config *OAuth2MiddlewareConfig, tokenString string) (*UserInfo, jwt.MapClaims, error) {
	if config.UseIntrospection {
		return validateTokenWithIntrospection(ctx, config, tokenString)
	}
	return validateTokenLocally(ctx, config, tokenString)
}

// validateTokenWithIntrospection validates token using OAuth2 token introspection
func validateTokenWithIntrospection(ctx context.Context, config *OAuth2MiddlewareConfig, tokenString string) (*UserInfo, jwt.MapClaims, error) {
	if config.OAuth2Client == nil {
		return nil, nil, fmt.Errorf("oauth2: client not configured for introspection")
	}

	introspection, err := config.OAuth2Client.IntrospectToken(ctx, tokenString)
	if err != nil {
		return nil, nil, fmt.Errorf("oauth2: token introspection failed: %w", err)
	}

	if !introspection.Active {
		return nil, nil, fmt.Errorf("oauth2: token is not active")
	}

	if introspection.IsExpired() {
		return nil, nil, fmt.Errorf("oauth2: token has expired")
	}

	// Build user info from introspection response
	userInfo := &UserInfo{
		Subject:  introspection.Sub,
		Username: introspection.Username,
		Claims:   make(map[string]interface{}),
	}

	// Parse scopes (OAuth2 scopes are space-delimited)
	if introspection.Scope != "" {
		userInfo.Scopes = splitScopes(introspection.Scope)
	}

	// Build claims map
	claims := jwt.MapClaims{
		"sub":        introspection.Sub,
		"username":   introspection.Username,
		"client_id":  introspection.ClientID,
		"token_type": introspection.TokenType,
		"iss":        introspection.Iss,
		"aud":        introspection.Aud,
		"exp":        introspection.Exp,
		"iat":        introspection.Iat,
		"nbf":        introspection.Nbf,
		"jti":        introspection.Jti,
		"scope":      introspection.Scope,
	}

	// Add extensions to claims and user info
	for key, value := range introspection.Extensions {
		claims[key] = value
		userInfo.Claims[key] = value
	}

	return userInfo, claims, nil
}

// validateTokenLocally validates token using local JWT validation
func validateTokenLocally(ctx context.Context, config *OAuth2MiddlewareConfig, tokenString string) (*UserInfo, jwt.MapClaims, error) {
	if config.JWTValidator == nil {
		return nil, nil, fmt.Errorf("oauth2: JWT validator not configured")
	}

	token, err := config.JWTValidator.ValidateWithContext(ctx, tokenString)
	if err != nil {
		return nil, nil, fmt.Errorf("oauth2: token validation failed: %w", err)
	}

	// Extract claims
	claims, err := secjwt.ExtractClaims(token)
	if err != nil {
		return nil, nil, fmt.Errorf("oauth2: failed to extract claims: %w", err)
	}

	// Convert to custom claims for easier access
	customClaims, err := secjwt.MapClaimsToCustomClaims(claims)
	if err != nil {
		return nil, nil, fmt.Errorf("oauth2: failed to convert claims: %w", err)
	}

	// Build user info from claims
	userInfo := &UserInfo{
		Subject:  customClaims.Subject,
		Username: customClaims.Username,
		Email:    customClaims.Email,
		Roles:    customClaims.Roles,
		Scopes:   expandScopes(customClaims.Permissions), // Expand space-delimited scopes
		Claims:   customClaims.Metadata,
	}

	return userInfo, claims, nil
}

// defaultTokenExtractor is the default function to extract bearer token from Authorization header or cookie
func defaultTokenExtractor(cookieName string) func(*gin.Context) (string, error) {
	return func(c *gin.Context) (string, error) {
		// Try Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			return extractBearerToken(authHeader)
		}

		// Try cookie if cookie name is configured
		if cookieName != "" {
			token, err := c.Cookie(cookieName)
			if err == nil && token != "" {
				return token, nil
			}
		}

		return "", fmt.Errorf("missing authentication token")
	}
}

// extractBearerToken extracts the token from Bearer authorization header
func extractBearerToken(authHeader string) (string, error) {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid authorization header format")
	}

	scheme := strings.ToLower(parts[0])
	if scheme != "bearer" {
		return "", fmt.Errorf("unsupported authorization scheme: %s", scheme)
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("empty bearer token")
	}

	return token, nil
}

// shouldSkipPath checks if the path should skip authentication
func shouldSkipPath(path string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
		// Support wildcard matching
		if strings.HasSuffix(skipPath, "*") {
			prefix := strings.TrimSuffix(skipPath, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}
	return false
}

// defaultOAuth2ErrorHandler is the default error handler for authentication failures
func defaultOAuth2ErrorHandler(c *gin.Context, err error) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"error":   "authentication_failed",
		"message": err.Error(),
	})
	c.Abort()
}

// GetUserInfo retrieves user information from gin context
func GetUserInfo(c *gin.Context) (*UserInfo, bool) {
	userInfo, exists := c.Get(ContextKeyUserInfo)
	if !exists {
		return nil, false
	}

	info, ok := userInfo.(*UserInfo)
	return info, ok
}

// GetClaims retrieves JWT claims from gin context
func GetClaims(c *gin.Context) (jwt.MapClaims, bool) {
	claims, exists := c.Get(ContextKeyClaims)
	if !exists {
		return nil, false
	}

	mapClaims, ok := claims.(jwt.MapClaims)
	return mapClaims, ok
}

// GetTokenString retrieves the raw token string from gin context
func GetTokenString(c *gin.Context) (string, bool) {
	token, exists := c.Get(ContextKeyTokenString)
	if !exists {
		return "", false
	}

	tokenStr, ok := token.(string)
	return tokenStr, ok
}

// MustGetUserInfo retrieves user information from context or panics
func MustGetUserInfo(c *gin.Context) *UserInfo {
	userInfo, ok := GetUserInfo(c)
	if !ok {
		panic("oauth2: user info not found in context")
	}
	return userInfo
}

// MustGetClaims retrieves claims from context or panics
func MustGetClaims(c *gin.Context) jwt.MapClaims {
	claims, ok := GetClaims(c)
	if !ok {
		panic("oauth2: claims not found in context")
	}
	return claims
}

// splitScopes splits a space-delimited scope string into individual scopes.
// This handles the standard OAuth2 scope format (RFC 6749).
func splitScopes(scopeString string) []string {
	if scopeString == "" {
		return nil
	}

	scopes := strings.Split(scopeString, " ")
	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// expandScopes expands a slice of scopes that may contain space-delimited strings
// into individual scope values. This handles both array format and OAuth2 standard
// space-delimited format.
func expandScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return nil
	}

	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		// Check if this scope contains spaces (OAuth2 standard format)
		if strings.Contains(scope, " ") {
			// Split and add individual scopes
			expanded := splitScopes(scope)
			result = append(result, expanded...)
		} else {
			// Single scope, add directly
			result = append(result, scope)
		}
	}
	return result
}

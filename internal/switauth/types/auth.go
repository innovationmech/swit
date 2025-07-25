// Copyright 2024 Innovation Mechanism. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package types

import (
	"time"
)

// AuthResponse represents the authentication response with tokens
// Used for both login and token refresh responses
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ValidateTokenRequest represents a token validation request
type ValidateTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	Token string `json:"token" binding:"required"`
}

// TokenValidationResponse represents the response for token validation
type TokenValidationResponse struct {
	Valid  bool   `json:"valid"`
	UserID string `json:"user_id,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ValidateTokenResponse represents the response for token validation endpoint
type ValidateTokenResponse struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

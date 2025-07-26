// Copyright 2024 Innovation Mechanism. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package interfaces

import (
	"context"

	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/types"
)

// AuthService defines the interface for authentication business logic
// including login, logout, token refresh, and token validation operations
type AuthService interface {
	// Login authenticates a user and returns access and refresh tokens
	Login(ctx context.Context, username, password string) (*types.AuthResponse, error)

	// RefreshToken generates new tokens using a refresh token
	RefreshToken(ctx context.Context, refreshToken string) (*types.AuthResponse, error)

	// ValidateToken validates an access token and returns token details
	ValidateToken(ctx context.Context, tokenString string) (*model.Token, error)

	// Logout invalidates a token and logs out the user
	Logout(ctx context.Context, tokenString string) error
}

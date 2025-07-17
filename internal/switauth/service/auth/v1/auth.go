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

package v1

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/innovationmech/swit/internal/switauth/client"
	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/repository"
	"github.com/innovationmech/swit/pkg/utils"
)

// AuthResponse represents the authentication response with tokens
// Used for both login and token refresh responses
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// AuthSrv defines the interface for authentication business logic
// including login, logout, token refresh, and token validation operations
type AuthSrv interface {
	// Login authenticates a user and returns access and refresh tokens
	Login(ctx context.Context, username, password string) (*AuthResponse, error)

	// RefreshToken generates new tokens using a refresh token
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)

	// ValidateToken validates an access token and returns token details
	ValidateToken(ctx context.Context, tokenString string) (*model.Token, error)

	// Logout invalidates a token and logs out the user
	Logout(ctx context.Context, tokenString string) error
}

// AuthServiceConfig auth service config
type AuthServiceConfig struct {
	UserClient client.UserClient
	TokenRepo  repository.TokenRepository
	// Logger      logger.Logger
	// Cache       cache.Cache
	// EventBus    eventbus.EventBus
	// Validator   validator.Validator
}

// AuthServiceOption auth service option function type
type AuthServiceOption func(*AuthServiceConfig)

// authService implements the AuthSrv interface
type authService struct {
	config *AuthServiceConfig
}

// WithUserClient set the user client dependency
func WithUserClient(userClient client.UserClient) AuthServiceOption {
	return func(config *AuthServiceConfig) {
		config.UserClient = userClient
	}
}

// WithTokenRepository set the token repository dependency
func WithTokenRepository(tokenRepo repository.TokenRepository) AuthServiceOption {
	return func(config *AuthServiceConfig) {
		config.TokenRepo = tokenRepo
	}
}

// can add more options functions, such as:
// func WithLogger(logger logger.Logger) AuthServiceOption {
//     return func(config *AuthServiceConfig) {
//         config.Logger = logger
//     }
// }

// func WithCache(cache cache.Cache) AuthServiceOption {
//     return func(config *AuthServiceConfig) {
//         config.Cache = cache
//     }
// }

// NewAuthSrv creates a new auth service using options pattern.
func NewAuthSrv(opts ...AuthServiceOption) (AuthSrv, error) {
	config := &AuthServiceConfig{}

	// apply options
	for _, opt := range opts {
		opt(config)
	}

	// check if the required dependencies are provided
	if config.UserClient == nil {
		return nil, errors.New("user client is required")
	}
	if config.TokenRepo == nil {
		return nil, errors.New("token repository is required")
	}

	return &authService{
		config: config,
	}, nil
}

// NewAuthSrvWithConfig creates a new auth service using config struct
func NewAuthSrvWithConfig(config *AuthServiceConfig) (AuthSrv, error) {
	if config.UserClient == nil {
		return nil, errors.New("user client is required")
	}
	if config.TokenRepo == nil {
		return nil, errors.New("token repository is required")
	}

	return &authService{
		config: config,
	}, nil
}

// Login authenticates a user with username and password
func (s *authService) Login(ctx context.Context, username, password string) (*AuthResponse, error) {
	// Validate user credentials
	user, err := s.config.UserClient.ValidateUserCredentials(ctx, username, password)
	if err != nil {
		return nil, err
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is not active")
	}

	// Generate access and refresh tokens
	accessToken, accessExpiresAt, err := utils.GenerateAccessToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	refreshToken, refreshExpiresAt, err := utils.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	// Create token record
	token := &model.Token{
		UserID:           user.ID,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
		IsValid:          true,
	}

	// Store token in repository
	if err := s.config.TokenRepo.Create(ctx, token); err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiresAt,
		TokenType:    "Bearer",
	}, nil
}

// RefreshToken generates new access and refresh tokens
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// Validate refresh token
	claims, err := utils.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid token")
	}

	// Get existing token
	token, err := s.config.TokenRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// Check if refresh token is still valid
	if !token.IsValid || time.Now().After(token.RefreshExpiresAt) {
		return nil, fmt.Errorf("refresh token has expired")
	}

	// Generate new tokens
	newAccessToken, accessExpiresAt, err := utils.GenerateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	newRefreshToken, refreshExpiresAt, err := utils.GenerateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	// Update token record
	token.AccessToken = newAccessToken
	token.RefreshToken = newRefreshToken
	token.AccessExpiresAt = accessExpiresAt
	token.RefreshExpiresAt = refreshExpiresAt

	if err := s.config.TokenRepo.Update(ctx, token); err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    accessExpiresAt,
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken validates an access token
func (s *authService) ValidateToken(ctx context.Context, tokenString string) (*model.Token, error) {
	// Validate token signature and claims
	claims, err := utils.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Get token from repository
	token, err := s.config.TokenRepo.GetByAccessToken(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	// Check token validity
	if !token.IsValid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Check expiration
	if time.Now().After(token.AccessExpiresAt) {
		return nil, fmt.Errorf("access token has expired")
	}

	// Verify user ID matches token claims
	userID, ok := claims["user_id"].(string)
	if !ok || userID != token.UserID.String() {
		return nil, fmt.Errorf("token is invalid")
	}

	return token, nil
}

// Logout invalidates a token
func (s *authService) Logout(ctx context.Context, tokenString string) error {
	// Invalidate the token
	return s.config.TokenRepo.InvalidateToken(ctx, tokenString)
}

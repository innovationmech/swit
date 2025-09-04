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

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/innovationmech/swit/internal/switauth/client"
	"github.com/innovationmech/swit/internal/switauth/interfaces"
	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/repository"
	"github.com/innovationmech/swit/internal/switauth/types"
	"github.com/innovationmech/swit/pkg/tracing"
	"github.com/innovationmech/swit/pkg/utils"
)

// authService implements the interfaces.AuthService interface
type authService struct {
	config *AuthServiceConfig
}

// AuthServiceConfig auth service config
type AuthServiceConfig struct {
	UserClient     client.UserClient
	TokenRepo      repository.TokenRepository
	TracingManager tracing.TracingManager
	// Logger      logger.Logger
	// Cache       cache.Cache
	// EventBus    eventbus.EventBus
	// Validator   validator.Validator
}

// AuthServiceOption auth service option function type
type AuthServiceOption func(*AuthServiceConfig)

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

// WithTracingManager set the tracing manager dependency
func WithTracingManager(tracingManager tracing.TracingManager) AuthServiceOption {
	return func(config *AuthServiceConfig) {
		config.TracingManager = tracingManager
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
func NewAuthSrv(opts ...AuthServiceOption) (interfaces.AuthService, error) {
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
func NewAuthSrvWithConfig(config *AuthServiceConfig) (interfaces.AuthService, error) {
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
func (s *authService) Login(ctx context.Context, username, password string) (*types.AuthResponse, error) {
	// Create tracing span
	var span tracing.Span
	if s.config.TracingManager != nil {
		ctx, span = s.config.TracingManager.StartSpan(ctx, "AuthService.Login",
			tracing.WithAttributes(
				attribute.String("auth.username", username),
			),
		)
		defer span.End()
	}

	// Validate user credentials span (cross-service call)
	var user *model.User
	var err error
	if s.config.TracingManager != nil {
		_, credentialsSpan := s.config.TracingManager.StartSpan(ctx, "validate_user_credentials")
		user, err = s.config.UserClient.ValidateUserCredentials(ctx, username, password)
		if err != nil {
			credentialsSpan.SetStatus(codes.Error, err.Error())
			span.SetStatus(codes.Error, "credential validation failed")
			credentialsSpan.End()
			return nil, err
		}
		credentialsSpan.End()
	} else {
		// Fallback credential validation if no tracing
		user, err = s.config.UserClient.ValidateUserCredentials(ctx, username, password)
		if err != nil {
			return nil, err
		}
	}

	// Check if user is active
	if !user.IsActive {
		if span != nil {
			span.SetStatus(codes.Error, "user account is not active")
			span.SetAttribute("error.type", "inactive_user")
		}
		return nil, fmt.Errorf("user account is not active")
	}

	// Generate tokens span
	var accessToken, refreshToken string
	var accessExpiresAt, refreshExpiresAt time.Time
	if s.config.TracingManager != nil {
		_, tokenSpan := s.config.TracingManager.StartSpan(ctx, "generate_tokens")
		
		accessToken, accessExpiresAt, err = utils.GenerateAccessToken(user.ID.String())
		if err != nil {
			tokenSpan.SetStatus(codes.Error, err.Error())
			span.SetStatus(codes.Error, "token generation failed")
			tokenSpan.End()
			return nil, err
		}

		refreshToken, refreshExpiresAt, err = utils.GenerateRefreshToken(user.ID.String())
		if err != nil {
			tokenSpan.SetStatus(codes.Error, err.Error())
			span.SetStatus(codes.Error, "token generation failed")
			tokenSpan.End()
			return nil, err
		}
		
		tokenSpan.SetAttribute("token.user_id", user.ID.String())
		tokenSpan.End()
	} else {
		// Fallback token generation if no tracing
		accessToken, accessExpiresAt, err = utils.GenerateAccessToken(user.ID.String())
		if err != nil {
			return nil, err
		}

		refreshToken, _, err = utils.GenerateRefreshToken(user.ID.String())
		if err != nil {
			return nil, err
		}
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

	// Store token in repository span
	if s.config.TracingManager != nil {
		_, storeSpan := s.config.TracingManager.StartSpan(ctx, "store_token")
		
		if err := s.config.TokenRepo.Create(ctx, token); err != nil {
			storeSpan.SetStatus(codes.Error, err.Error())
			span.SetStatus(codes.Error, "token storage failed")
			storeSpan.End()
			return nil, err
		}
		
		storeSpan.SetAttribute("operation.success", true)
		storeSpan.SetAttribute("token.user_id", user.ID.String())
		storeSpan.End()
	} else {
		// Fallback token storage if no tracing
		if err := s.config.TokenRepo.Create(ctx, token); err != nil {
			return nil, err
		}
	}

	if span != nil {
		span.SetAttribute("auth.user_id", user.ID.String())
		span.SetAttribute("operation.success", true)
	}

	return &types.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiresAt,
		TokenType:    "Bearer",
	}, nil
}

// RefreshToken generates new access and refresh tokens
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*types.AuthResponse, error) {
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

	return &types.AuthResponse{
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

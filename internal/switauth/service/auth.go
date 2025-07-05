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

package service

import (
	"context"
	"errors"
	"time"

	"github.com/innovationmech/swit/pkg/utils"

	"github.com/innovationmech/swit/internal/switauth/client"
	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/repository"
)

type AuthService interface {
	Login(ctx context.Context, username, password string) (string, string, error)
	RefreshToken(ctx context.Context, refreshToken string) (string, string, error)
	ValidateToken(ctx context.Context, tokenString string) (*model.Token, error)
	Logout(ctx context.Context, tokenString string) error
}

type authService struct {
	userClient client.UserClient
	tokenRepo  repository.TokenRepository
}

func (s *authService) Login(ctx context.Context, username, password string) (string, string, error) {
	user, err := s.userClient.ValidateUserCredentials(ctx, username, password)
	if err != nil {
		return "", "", err
	}
	if !user.IsActive {
		return "", "", errors.New("user is not active")
	}

	// Generate access token and refresh token
	accessToken, accessExpiresAt, err := utils.GenerateAccessToken(user.ID.String())
	if err != nil {
		return "", "", err
	}

	refreshToken, refreshExpiresAt, err := utils.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return "", "", err
	}

	token := &model.Token{
		UserID:           user.ID,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
		IsValid:          true,
	}

	if err := s.tokenRepo.Create(ctx, token); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	// Validate refresh token
	claims, err := utils.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}

	userID := claims["user_id"].(string)

	// Get refresh token using TokenRepository
	token, err := s.tokenRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", err
	}

	// Check if refresh token has expired
	if time.Now().After(token.RefreshExpiresAt) {
		return "", "", errors.New("refresh token has expired")
	}

	// Generate new access token and refresh token
	newAccessToken, accessExpiresAt, err := utils.GenerateAccessToken(userID)
	if err != nil {
		return "", "", err
	}

	newRefreshToken, refreshExpiresAt, err := utils.GenerateRefreshToken(userID)
	if err != nil {
		return "", "", err
	}

	// Update Token instance
	token.AccessToken = newAccessToken
	token.RefreshToken = newRefreshToken
	token.AccessExpiresAt = accessExpiresAt
	token.RefreshExpiresAt = refreshExpiresAt

	// Update token using TokenRepository
	if err := s.tokenRepo.Update(ctx, token); err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *authService) ValidateToken(ctx context.Context, tokenString string) (*model.Token, error) {
	// Validate access token
	_, err := utils.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Get token using TokenRepository
	token, err := s.tokenRepo.GetByAccessToken(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	// Check if token has expired
	if time.Now().After(token.AccessExpiresAt) {
		return nil, errors.New("token has expired")
	}

	return token, nil
}

func (s *authService) Logout(ctx context.Context, tokenString string) error {
	if err := s.tokenRepo.InvalidateToken(ctx, tokenString); err != nil {
		return err
	}
	return nil
}

func NewAuthService(userClient client.UserClient, tokenRepo repository.TokenRepository) AuthService {
	return &authService{userClient: userClient, tokenRepo: tokenRepo}
}

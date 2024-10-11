package service

import (
	"errors"
	"time"

	"github.com/innovationmech/swit/internal/pkg/utils"
	"github.com/innovationmech/swit/internal/switauth/client"
	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/repository"
)

type AuthService interface {
	Login(username, password string) (string, string, error)
	RefreshToken(refreshToken string) (string, string, error)
	ValidateToken(tokenString string) (*model.Token, error)
	Logout(tokenString string) error
}

type authService struct {
	userClient client.UserClient
	tokenRepo  repository.TokenRepository
}

func (s *authService) Login(username, password string) (string, string, error) {
	// Validate user logic
	user, err := s.userClient.GetUserByUsername(username)
	if err != nil {
		return "", "", err
	}
	if !user.IsActive {
		return "", "", errors.New("user is not active")
	}
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", "", errors.New("invalid password")
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

	// Create Token instance
	token := &model.Token{
		UserID:           user.ID,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
		IsValid:          true,
	}

	// Save token using TokenRepository
	if err := s.tokenRepo.Create(token); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *authService) RefreshToken(refreshToken string) (string, string, error) {
	// Validate refresh token
	claims, err := utils.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}

	userID := claims["user_id"].(string)

	// Get refresh token using TokenRepository
	token, err := s.tokenRepo.GetByRefreshToken(refreshToken)
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
	if err := s.tokenRepo.Update(token); err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *authService) ValidateToken(tokenString string) (*model.Token, error) {
	// Validate access token
	_, err := utils.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Get token using TokenRepository
	token, err := s.tokenRepo.GetByAccessToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Check if token has expired
	if time.Now().After(token.AccessExpiresAt) {
		return nil, errors.New("token has expired")
	}

	return token, nil
}

func (s *authService) Logout(tokenString string) error {
	if err := s.tokenRepo.InvalidateToken(tokenString); err != nil {
		return err
	}
	return nil
}

func NewAuthService(userClient client.UserClient, tokenRepo repository.TokenRepository) AuthService {
	return &authService{userClient: userClient, tokenRepo: tokenRepo}
}

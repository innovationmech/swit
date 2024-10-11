package repository

import (
	"errors"
	"github.com/innovationmech/swit/internal/switauth/model"
	"gorm.io/gorm"
)

type TokenRepository interface {
	Create(token *model.Token) error
	GetByAccessToken(tokenString string) (*model.Token, error)
	GetByRefreshToken(tokenString string) (*model.Token, error)
	Update(token *model.Token) error
	InvalidateToken(tokenString string) error
}

type tokenRepository struct {
	db *gorm.DB
}

func (r *tokenRepository) Create(token *model.Token) error {
	return r.db.Create(token).Error
}

func (r *tokenRepository) GetByAccessToken(tokenString string) (*model.Token, error) {
	var token model.Token
	err := r.db.Where("access_token = ? AND is_valid = ?", tokenString, true).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid or expired token")
		}
		return nil, err
	}
	return &token, nil
}

func (r *tokenRepository) GetByRefreshToken(refreshToken string) (*model.Token, error) {
	var token model.Token
	err := r.db.Where("refresh_token = ? AND is_valid = ?", refreshToken, true).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid or expired refresh token")
		}
		return nil, err
	}
	return &token, nil
}

func (r *tokenRepository) Update(token *model.Token) error {
	return r.db.Save(token).Error
}

func (r *tokenRepository) InvalidateToken(tokenString string) error {
	return r.db.Model(&model.Token{}).Where("access_token = ?", tokenString).Update("is_valid", false).Error
}

func NewTokenRepository(db *gorm.DB) TokenRepository {
	return &tokenRepository{db: db}
}

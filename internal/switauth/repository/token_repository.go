// Copyright © 2023 jackelyj <dreamerlyj@gmail.com>
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

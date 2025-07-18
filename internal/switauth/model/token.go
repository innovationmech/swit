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

package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/innovationmech/swit/pkg/utils"
	"gorm.io/gorm"
)

// Token represents a JWT authentication token pair with access and refresh tokens
// stored in the database for user authentication sessions
type Token struct {
	ID               uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	UserID           uuid.UUID `json:"user_id" gorm:"type:char(36);not null"`
	AccessToken      string    `json:"access_token" gorm:"type:text;not null"`
	RefreshToken     string    `json:"refresh_token" gorm:"type:text;not null"`
	AccessExpiresAt  time.Time `json:"access_expires_at" gorm:"type:timestamp;not null"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at" gorm:"type:timestamp;not null"`
	IsValid          bool      `json:"is_valid" gorm:"type:boolean;not null;default:true"`
	CreatedAt        time.Time `json:"created_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime"`
}

// TableName specifies the database table name
func (Token) TableName() string {
	return "tokens"
}

// BeforeCreate is a GORM hook, called before creating a record
func (token *Token) BeforeCreate(tx *gorm.DB) (err error) {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	return
}

// BeforeSave is a GORM hook, called before saving a record
// It encrypts the tokens before storing them in the database
func (token *Token) BeforeSave(tx *gorm.DB) (err error) {
	// Encrypt access token if it's not already encrypted
	if token.AccessToken != "" {
		encrypted, err := utils.EncryptToken(token.AccessToken)
		if err != nil {
			return err
		}
		token.AccessToken = encrypted
	}
	
	// Encrypt refresh token if it's not already encrypted
	if token.RefreshToken != "" {
		encrypted, err := utils.EncryptToken(token.RefreshToken)
		if err != nil {
			return err
		}
		token.RefreshToken = encrypted
	}
	
	return nil
}

// AfterFind is a GORM hook, called after loading a record from database
// It decrypts the tokens after loading them from the database
func (token *Token) AfterFind(tx *gorm.DB) (err error) {
	// Decrypt access token
	if token.AccessToken != "" {
		decrypted, err := utils.DecryptToken(token.AccessToken)
		if err != nil {
			return err
		}
		token.AccessToken = decrypted
	}
	
	// Decrypt refresh token
	if token.RefreshToken != "" {
		decrypted, err := utils.DecryptToken(token.RefreshToken)
		if err != nil {
			return err
		}
		token.RefreshToken = decrypted
	}
	
	return nil
}

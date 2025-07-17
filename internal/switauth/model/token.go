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
	
	// Encrypt tokens before saving
	if err := token.encryptTokens(); err != nil {
		return err
	}
	
	return
}

// BeforeUpdate is a GORM hook, called before updating a record
func (token *Token) BeforeUpdate(tx *gorm.DB) (err error) {
	// Encrypt tokens before updating
	if err := token.encryptTokens(); err != nil {
		return err
	}
	return
}

// AfterFind is a GORM hook, called after finding a record
func (token *Token) AfterFind(tx *gorm.DB) (err error) {
	// Decrypt tokens after loading from database
	if err := token.decryptTokens(); err != nil {
		return err
	}
	return
}

// encryptTokens encrypts the access and refresh tokens
func (token *Token) encryptTokens() error {
	// Only encrypt if tokens are not already encrypted (avoid double encryption)
	if token.AccessToken != "" && !token.isEncrypted(token.AccessToken) {
		encryptedAccessToken, err := utils.EncryptToken(token.AccessToken)
		if err != nil {
			return err
		}
		token.AccessToken = encryptedAccessToken
	}
	
	if token.RefreshToken != "" && !token.isEncrypted(token.RefreshToken) {
		encryptedRefreshToken, err := utils.EncryptToken(token.RefreshToken)
		if err != nil {
			return err
		}
		token.RefreshToken = encryptedRefreshToken
	}
	
	return nil
}

// decryptTokens decrypts the access and refresh tokens
func (token *Token) decryptTokens() error {
	// Only decrypt if tokens are encrypted
	if token.AccessToken != "" && token.isEncrypted(token.AccessToken) {
		decryptedAccessToken, err := utils.DecryptToken(token.AccessToken)
		if err != nil {
			return err
		}
		token.AccessToken = decryptedAccessToken
	}
	
	if token.RefreshToken != "" && token.isEncrypted(token.RefreshToken) {
		decryptedRefreshToken, err := utils.DecryptToken(token.RefreshToken)
		if err != nil {
			return err
		}
		token.RefreshToken = decryptedRefreshToken
	}
	
	return nil
}

// isEncrypted checks if a token is encrypted by checking if it's base64 encoded
// This is a simple heuristic - encrypted tokens are base64 encoded
func (token *Token) isEncrypted(tokenStr string) bool {
	// JWT tokens start with "ey" (base64 encoded header)
	// Encrypted tokens are longer base64 strings without the JWT format
	return len(tokenStr) > 100 && !isJWTFormat(tokenStr)
}

// isJWTFormat checks if a string looks like a JWT token (has 3 parts separated by dots)
func isJWTFormat(tokenStr string) bool {
	parts := 0
	for _, char := range tokenStr {
		if char == '.' {
			parts++
		}
	}
	return parts == 2 // JWT has 3 parts separated by 2 dots
}

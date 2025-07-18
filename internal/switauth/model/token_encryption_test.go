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
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/innovationmech/swit/pkg/utils"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestToken_EncryptionIdempotency(t *testing.T) {
	// Initialize encryption key for testing
	os.Setenv("TOKEN_ENCRYPTION_KEY", "test-key-for-encryption-testing")
	err := utils.InitEncryptionKey()
	assert.NoError(t, err)

	// Create a test token
	token := &Token{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		AccessToken:      "test-access-token",
		RefreshToken:     "test-refresh-token",
		AccessExpiresAt:  time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		IsValid:          true,
	}

	originalAccessToken := token.AccessToken
	originalRefreshToken := token.RefreshToken

	// Test multiple calls to encryptTokens should not double-encrypt
	err = token.encryptTokens()
	assert.NoError(t, err)
	assert.True(t, token.accessTokenEncrypted)
	assert.True(t, token.refreshTokenEncrypted)
	assert.NotEqual(t, originalAccessToken, token.AccessToken)
	assert.NotEqual(t, originalRefreshToken, token.RefreshToken)

	firstEncryptedAccess := token.AccessToken
	firstEncryptedRefresh := token.RefreshToken

	// Second call should not change the encrypted tokens
	err = token.encryptTokens()
	assert.NoError(t, err)
	assert.Equal(t, firstEncryptedAccess, token.AccessToken)
	assert.Equal(t, firstEncryptedRefresh, token.RefreshToken)

	// Test multiple calls to decryptTokens should not double-decrypt
	err = token.decryptTokens()
	assert.NoError(t, err)
	assert.False(t, token.accessTokenEncrypted)
	assert.False(t, token.refreshTokenEncrypted)
	assert.Equal(t, originalAccessToken, token.AccessToken)
	assert.Equal(t, originalRefreshToken, token.RefreshToken)

	// Second call should not change the decrypted tokens
	err = token.decryptTokens()
	assert.NoError(t, err)
	assert.Equal(t, originalAccessToken, token.AccessToken)
	assert.Equal(t, originalRefreshToken, token.RefreshToken)
}

func TestToken_BeforeSaveIdempotency(t *testing.T) {
	// Initialize encryption key for testing
	os.Setenv("TOKEN_ENCRYPTION_KEY", "test-key-for-encryption-testing")
	err := utils.InitEncryptionKey()
	assert.NoError(t, err)

	// Create a test token
	token := &Token{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		AccessToken:      "test-access-token",
		RefreshToken:     "test-refresh-token",
		AccessExpiresAt:  time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		IsValid:          true,
	}

	originalAccessToken := token.AccessToken
	originalRefreshToken := token.RefreshToken

	// First call to BeforeSave should encrypt
	err = token.BeforeSave(&gorm.DB{})
	assert.NoError(t, err)
	assert.True(t, token.accessTokenEncrypted)
	assert.True(t, token.refreshTokenEncrypted)
	assert.NotEqual(t, originalAccessToken, token.AccessToken)
	assert.NotEqual(t, originalRefreshToken, token.RefreshToken)

	firstEncryptedAccess := token.AccessToken
	firstEncryptedRefresh := token.RefreshToken

	// Second call should not re-encrypt
	err = token.BeforeSave(&gorm.DB{})
	assert.NoError(t, err)
	assert.Equal(t, firstEncryptedAccess, token.AccessToken)
	assert.Equal(t, firstEncryptedRefresh, token.RefreshToken)
}

func TestToken_AfterFindIdempotency(t *testing.T) {
	// Initialize encryption key for testing
	os.Setenv("TOKEN_ENCRYPTION_KEY", "test-key-for-encryption-testing")
	err := utils.InitEncryptionKey()
	assert.NoError(t, err)

	// Create a token with encrypted data (simulating DB load)
	token := &Token{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		AccessExpiresAt:  time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		IsValid:          true,
	}

	// Encrypt some test tokens first
	originalAccessToken := "test-access-token"
	originalRefreshToken := "test-refresh-token"
	encryptedAccess, err := utils.EncryptToken(originalAccessToken)
	assert.NoError(t, err)
	encryptedRefresh, err := utils.EncryptToken(originalRefreshToken)
	assert.NoError(t, err)

	token.AccessToken = encryptedAccess
	token.RefreshToken = encryptedRefresh

	// First call to AfterFind should decrypt
	err = token.AfterFind(&gorm.DB{})
	assert.NoError(t, err)
	assert.False(t, token.accessTokenEncrypted)
	assert.False(t, token.refreshTokenEncrypted)
	assert.Equal(t, originalAccessToken, token.AccessToken)
	assert.Equal(t, originalRefreshToken, token.RefreshToken)

	// Second call should not re-decrypt (tokens are already decrypted)
	// Reset the encrypted tokens to simulate another DB load
	token.AccessToken = encryptedAccess
	token.RefreshToken = encryptedRefresh
	err = token.AfterFind(&gorm.DB{})
	assert.NoError(t, err)
	assert.Equal(t, originalAccessToken, token.AccessToken)
	assert.Equal(t, originalRefreshToken, token.RefreshToken)
}

func TestToken_DisabledEncryption(t *testing.T) {
	// Set environment to disable encryption
	os.Setenv("DISABLE_TOKEN_ENCRYPTION", "true")
	defer os.Unsetenv("DISABLE_TOKEN_ENCRYPTION")

	// Create a test token
	token := &Token{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		AccessToken:      "test-access-token",
		RefreshToken:     "test-refresh-token",
		AccessExpiresAt:  time.Now().Add(time.Hour),
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		IsValid:          true,
	}

	originalAccessToken := token.AccessToken
	originalRefreshToken := token.RefreshToken

	// Encryption should be skipped
	err := token.encryptTokens()
	assert.NoError(t, err)
	assert.True(t, token.accessTokenEncrypted) // State should still be tracked
	assert.True(t, token.refreshTokenEncrypted)
	assert.Equal(t, originalAccessToken, token.AccessToken) // But tokens unchanged
	assert.Equal(t, originalRefreshToken, token.RefreshToken)

	// Decryption should also be skipped
	err = token.decryptTokens()
	assert.NoError(t, err)
	assert.False(t, token.accessTokenEncrypted)
	assert.False(t, token.refreshTokenEncrypted)
	assert.Equal(t, originalAccessToken, token.AccessToken)
	assert.Equal(t, originalRefreshToken, token.RefreshToken)
}

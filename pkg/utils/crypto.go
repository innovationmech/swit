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

package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

// TokenCrypto handles encryption and decryption of tokens
type TokenCrypto struct {
	gcm cipher.AEAD
}

// NewTokenCrypto creates a new token crypto instance
func NewTokenCrypto() (*TokenCrypto, error) {
	// Get encryption key from environment variable
	encryptionKey := os.Getenv("TOKEN_ENCRYPTION_KEY")
	if encryptionKey == "" {
		return nil, errors.New("TOKEN_ENCRYPTION_KEY environment variable is required")
	}

	// Create a 32-byte key from the encryption key using SHA256
	hasher := sha256.New()
	hasher.Write([]byte(encryptionKey))
	key := hasher.Sum(nil)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &TokenCrypto{gcm: gcm}, nil
}

// Encrypt encrypts a token string and returns base64 encoded ciphertext
func (tc *TokenCrypto) Encrypt(plaintext string) (string, error) {
	// Generate a random nonce
	nonce := make([]byte, tc.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the plaintext
	ciphertext := tc.gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64 encoded ciphertext
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64 encoded ciphertext and returns the original token
func (tc *TokenCrypto) Decrypt(ciphertext string) (string, error) {
	// Decode base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	// Check minimum length (nonce + some data)
	nonceSize := tc.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := tc.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// Global token crypto instance
var tokenCrypto *TokenCrypto

// GetTokenCrypto returns the global token crypto instance
func GetTokenCrypto() (*TokenCrypto, error) {
	if tokenCrypto == nil {
		var err error
		tokenCrypto, err = NewTokenCrypto()
		if err != nil {
			return nil, err
		}
	}
	return tokenCrypto, nil
}

// EncryptToken encrypts a token using the global crypto instance
func EncryptToken(token string) (string, error) {
	crypto, err := GetTokenCrypto()
	if err != nil {
		return "", err
	}
	return crypto.Encrypt(token)
}

// DecryptToken decrypts a token using the global crypto instance
func DecryptToken(encryptedToken string) (string, error) {
	crypto, err := GetTokenCrypto()
	if err != nil {
		return "", err
	}
	return crypto.Decrypt(encryptedToken)
}
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

// Package security provides encryption and decryption functionality for Saga data.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Common encryption errors
var (
	ErrInvalidKeySize    = errors.New("invalid key size: must be 16, 24, or 32 bytes")
	ErrEncryptionFailed  = errors.New("encryption failed")
	ErrDecryptionFailed  = errors.New("decryption failed")
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	ErrKeyNotFound       = errors.New("encryption key not found")
	ErrKeyExpired        = errors.New("encryption key expired")
	ErrInvalidNonce      = errors.New("invalid nonce size")
	ErrEmptyPlaintext    = errors.New("plaintext cannot be empty")
	ErrEmptyCiphertext   = errors.New("ciphertext cannot be empty")
)

// EncryptionAlgorithm represents the encryption algorithm type
type EncryptionAlgorithm string

const (
	// AlgorithmAES256GCM represents AES-256-GCM encryption
	AlgorithmAES256GCM EncryptionAlgorithm = "aes-256-gcm"
)

// EncryptedData represents encrypted data with metadata
type EncryptedData struct {
	// Ciphertext is the encrypted data (base64 encoded)
	Ciphertext string

	// Algorithm used for encryption
	Algorithm EncryptionAlgorithm

	// KeyID identifies which key was used
	KeyID string

	// Nonce used for encryption (base64 encoded)
	Nonce string

	// Timestamp when encryption occurred
	Timestamp time.Time

	// Metadata additional information
	Metadata map[string]string
}

// Encryptor defines the interface for encryption operations
type Encryptor interface {
	// Encrypt encrypts plaintext data
	Encrypt(plaintext []byte) (*EncryptedData, error)

	// Decrypt decrypts ciphertext data
	Decrypt(encrypted *EncryptedData) ([]byte, error)

	// RotateKey rotates the encryption key
	RotateKey() error

	// GetKeyID returns the current key ID
	GetKeyID() string

	// IsHealthy checks if the encryptor is healthy
	IsHealthy() bool
}

// AESGCMEncryptor implements AES-256-GCM encryption
type AESGCMEncryptor struct {
	keyManager KeyManager
	logger     *zap.Logger
	mu         sync.RWMutex

	// Performance metrics
	encryptCount uint64
	decryptCount uint64
	errorCount   uint64
}

// AESGCMEncryptorConfig configures the AES-GCM encryptor
type AESGCMEncryptorConfig struct {
	KeyManager KeyManager
	Logger     *zap.Logger
}

// NewAESGCMEncryptor creates a new AES-256-GCM encryptor
func NewAESGCMEncryptor(config *AESGCMEncryptorConfig) (*AESGCMEncryptor, error) {
	if config.KeyManager == nil {
		return nil, fmt.Errorf("key manager is required")
	}

	encryptor := &AESGCMEncryptor{
		keyManager: config.KeyManager,
		logger:     config.Logger,
	}

	// Initialize logger if not set
	if encryptor.logger == nil {
		encryptor.logger = logger.Logger
		if encryptor.logger == nil {
			encryptor.logger = zap.NewNop()
		}
	}

	return encryptor, nil
}

// Encrypt implements Encryptor interface
func (e *AESGCMEncryptor) Encrypt(plaintext []byte) (*EncryptedData, error) {
	if len(plaintext) == 0 {
		return nil, ErrEmptyPlaintext
	}

	startTime := time.Now()

	// Get the current encryption key
	key, err := e.keyManager.GetCurrentKey()
	if err != nil {
		e.incrementErrorCount()
		e.logger.Error("Failed to get encryption key",
			zap.Error(err))
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	// Validate key size (AES-256 requires 32 bytes)
	if len(key.Key) != 32 {
		e.incrementErrorCount()
		return nil, fmt.Errorf("%w: got %d bytes, expected 32 bytes for AES-256", ErrInvalidKeySize, len(key.Key))
	}

	// Create AES cipher
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		e.incrementErrorCount()
		e.logger.Error("Failed to create AES cipher",
			zap.String("key_id", key.ID),
			zap.Error(err))
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		e.incrementErrorCount()
		e.logger.Error("Failed to create GCM mode",
			zap.String("key_id", key.ID),
			zap.Error(err))
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		e.incrementErrorCount()
		e.logger.Error("Failed to generate nonce",
			zap.String("key_id", key.ID),
			zap.Error(err))
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	e.incrementEncryptCount()

	elapsed := time.Since(startTime)
	e.logger.Debug("Encryption successful",
		zap.String("key_id", key.ID),
		zap.String("algorithm", string(AlgorithmAES256GCM)),
		zap.Int("plaintext_size", len(plaintext)),
		zap.Int("ciphertext_size", len(ciphertext)),
		zap.Duration("elapsed", elapsed))

	return &EncryptedData{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Algorithm:  AlgorithmAES256GCM,
		KeyID:      key.ID,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Timestamp:  time.Now(),
		Metadata:   make(map[string]string),
	}, nil
}

// Decrypt implements Encryptor interface
func (e *AESGCMEncryptor) Decrypt(encrypted *EncryptedData) ([]byte, error) {
	if encrypted == nil || encrypted.Ciphertext == "" {
		e.incrementErrorCount()
		return nil, ErrEmptyCiphertext
	}

	startTime := time.Now()

	// Validate algorithm
	if encrypted.Algorithm != AlgorithmAES256GCM {
		e.incrementErrorCount()
		return nil, fmt.Errorf("%w: unsupported algorithm %s", ErrDecryptionFailed, encrypted.Algorithm)
	}

	// Get the encryption key by ID
	key, err := e.keyManager.GetKey(encrypted.KeyID)
	if err != nil {
		e.incrementErrorCount()
		e.logger.Error("Failed to get decryption key",
			zap.String("key_id", encrypted.KeyID),
			zap.Error(err))
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	// Decode base64 ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Ciphertext)
	if err != nil {
		e.incrementErrorCount()
		e.logger.Error("Failed to decode ciphertext",
			zap.String("key_id", encrypted.KeyID),
			zap.Error(err))
		return nil, fmt.Errorf("%w: invalid base64 encoding", ErrInvalidCiphertext)
	}

	// Decode base64 nonce
	nonce, err := base64.StdEncoding.DecodeString(encrypted.Nonce)
	if err != nil {
		e.incrementErrorCount()
		e.logger.Error("Failed to decode nonce",
			zap.String("key_id", encrypted.KeyID),
			zap.Error(err))
		return nil, fmt.Errorf("%w: invalid nonce encoding", ErrInvalidNonce)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		e.incrementErrorCount()
		e.logger.Error("Failed to create AES cipher for decryption",
			zap.String("key_id", encrypted.KeyID),
			zap.Error(err))
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		e.incrementErrorCount()
		e.logger.Error("Failed to create GCM mode for decryption",
			zap.String("key_id", encrypted.KeyID),
			zap.Error(err))
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	// Validate nonce size
	if len(nonce) != gcm.NonceSize() {
		e.incrementErrorCount()
		return nil, fmt.Errorf("%w: got %d bytes, expected %d bytes", ErrInvalidNonce, len(nonce), gcm.NonceSize())
	}

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		e.incrementErrorCount()
		e.logger.Error("Failed to decrypt data",
			zap.String("key_id", encrypted.KeyID),
			zap.Error(err))
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	e.incrementDecryptCount()

	elapsed := time.Since(startTime)
	e.logger.Debug("Decryption successful",
		zap.String("key_id", encrypted.KeyID),
		zap.String("algorithm", string(encrypted.Algorithm)),
		zap.Int("ciphertext_size", len(ciphertext)),
		zap.Int("plaintext_size", len(plaintext)),
		zap.Duration("elapsed", elapsed))

	return plaintext, nil
}

// RotateKey implements Encryptor interface
func (e *AESGCMEncryptor) RotateKey() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.keyManager.RotateKey(); err != nil {
		e.logger.Error("Failed to rotate encryption key",
			zap.Error(err))
		return fmt.Errorf("key rotation failed: %w", err)
	}

	e.logger.Info("Encryption key rotated successfully",
		zap.String("new_key_id", e.keyManager.GetCurrentKeyID()))

	return nil
}

// GetKeyID implements Encryptor interface
func (e *AESGCMEncryptor) GetKeyID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.keyManager.GetCurrentKeyID()
}

// IsHealthy implements Encryptor interface
func (e *AESGCMEncryptor) IsHealthy() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check if key manager is healthy
	if !e.keyManager.IsHealthy() {
		return false
	}

	// Check if we can get the current key
	_, err := e.keyManager.GetCurrentKey()
	return err == nil
}

// GetMetrics returns encryption metrics
func (e *AESGCMEncryptor) GetMetrics() map[string]uint64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]uint64{
		"encrypt_count": e.encryptCount,
		"decrypt_count": e.decryptCount,
		"error_count":   e.errorCount,
	}
}

// incrementEncryptCount increments the encryption counter
func (e *AESGCMEncryptor) incrementEncryptCount() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.encryptCount++
}

// incrementDecryptCount increments the decryption counter
func (e *AESGCMEncryptor) incrementDecryptCount() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.decryptCount++
}

// incrementErrorCount increments the error counter
func (e *AESGCMEncryptor) incrementErrorCount() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.errorCount++
}

// EncryptString is a convenience method to encrypt a string
func (e *AESGCMEncryptor) EncryptString(plaintext string) (*EncryptedData, error) {
	return e.Encrypt([]byte(plaintext))
}

// DecryptString is a convenience method to decrypt to a string
func (e *AESGCMEncryptor) DecryptString(encrypted *EncryptedData) (string, error) {
	plaintext, err := e.Decrypt(encrypted)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

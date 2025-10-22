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

package security

import (
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAESGCMEncryptor(t *testing.T) {
	tests := []struct {
		name    string
		config  *AESGCMEncryptorConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &AESGCMEncryptorConfig{
				KeyManager: createTestKeyManager(t),
			},
			wantErr: false,
		},
		{
			name: "missing key manager",
			config: &AESGCMEncryptorConfig{
				KeyManager: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptor, err := NewAESGCMEncryptor(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, encryptor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, encryptor)
			}
		})
	}
}

func TestAESGCMEncryptor_Encrypt(t *testing.T) {
	encryptor := createTestEncryptor(t)

	tests := []struct {
		name      string
		plaintext []byte
		wantErr   bool
		errType   error
	}{
		{
			name:      "valid encryption",
			plaintext: []byte("sensitive saga data"),
			wantErr:   false,
		},
		{
			name:      "encrypt long data",
			plaintext: []byte(strings.Repeat("test data ", 1000)),
			wantErr:   false,
		},
		{
			name:      "empty plaintext",
			plaintext: []byte{},
			wantErr:   true,
			errType:   ErrEmptyPlaintext,
		},
		{
			name:      "nil plaintext",
			plaintext: nil,
			wantErr:   true,
			errType:   ErrEmptyPlaintext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encryptor.Encrypt(tt.plaintext)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				assert.Nil(t, encrypted)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, encrypted)
				assert.NotEmpty(t, encrypted.Ciphertext)
				assert.Equal(t, AlgorithmAES256GCM, encrypted.Algorithm)
				assert.NotEmpty(t, encrypted.KeyID)
				assert.NotEmpty(t, encrypted.Nonce)
				assert.False(t, encrypted.Timestamp.IsZero())
			}
		})
	}
}

func TestAESGCMEncryptor_Decrypt(t *testing.T) {
	encryptor := createTestEncryptor(t)
	plaintext := []byte("test saga context data")

	// First, encrypt some data
	encrypted, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	tests := []struct {
		name      string
		encrypted *EncryptedData
		wantErr   bool
		errType   error
	}{
		{
			name:      "valid decryption",
			encrypted: encrypted,
			wantErr:   false,
		},
		{
			name:      "nil encrypted data",
			encrypted: nil,
			wantErr:   true,
			errType:   ErrEmptyCiphertext,
		},
		{
			name: "empty ciphertext",
			encrypted: &EncryptedData{
				Ciphertext: "",
			},
			wantErr: true,
			errType: ErrEmptyCiphertext,
		},
		{
			name: "invalid algorithm",
			encrypted: &EncryptedData{
				Ciphertext: encrypted.Ciphertext,
				Algorithm:  "invalid",
				KeyID:      encrypted.KeyID,
				Nonce:      encrypted.Nonce,
			},
			wantErr: true,
			errType: ErrDecryptionFailed,
		},
		{
			name: "invalid key id",
			encrypted: &EncryptedData{
				Ciphertext: encrypted.Ciphertext,
				Algorithm:  AlgorithmAES256GCM,
				KeyID:      "non-existent-key",
				Nonce:      encrypted.Nonce,
			},
			wantErr: true,
			errType: ErrDecryptionFailed,
		},
		{
			name: "invalid base64 ciphertext",
			encrypted: &EncryptedData{
				Ciphertext: "not-valid-base64!!!",
				Algorithm:  AlgorithmAES256GCM,
				KeyID:      encrypted.KeyID,
				Nonce:      encrypted.Nonce,
			},
			wantErr: true,
			errType: ErrInvalidCiphertext,
		},
		{
			name: "invalid base64 nonce",
			encrypted: &EncryptedData{
				Ciphertext: encrypted.Ciphertext,
				Algorithm:  AlgorithmAES256GCM,
				KeyID:      encrypted.KeyID,
				Nonce:      "not-valid-base64!!!",
			},
			wantErr: true,
			errType: ErrInvalidNonce,
		},
		{
			name: "corrupted ciphertext",
			encrypted: &EncryptedData{
				Ciphertext: "YWJjZGVmZ2hpamts", // valid base64 but wrong data
				Algorithm:  AlgorithmAES256GCM,
				KeyID:      encrypted.KeyID,
				Nonce:      encrypted.Nonce,
			},
			wantErr: true,
			errType: ErrDecryptionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decrypted, err := encryptor.Decrypt(tt.encrypted)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, plaintext, decrypted)
			}
		})
	}
}

func TestAESGCMEncryptor_EncryptDecrypt_RoundTrip(t *testing.T) {
	encryptor := createTestEncryptor(t)

	testData := [][]byte{
		[]byte("short"),
		[]byte("medium length test data for saga"),
		[]byte(strings.Repeat("long ", 1000)),
		[]byte("special characters: !@#$%^&*()_+-=[]{}|;:',.<>?/"),
		[]byte("unicode: 你好世界 🌍"),
	}

	for i, plaintext := range testData {
		t.Run(string(rune(i)), func(t *testing.T) {
			// Encrypt
			encrypted, err := encryptor.Encrypt(plaintext)
			require.NoError(t, err)

			// Decrypt
			decrypted, err := encryptor.Decrypt(encrypted)
			require.NoError(t, err)

			// Verify
			assert.Equal(t, plaintext, decrypted)
		})
	}
}

func TestAESGCMEncryptor_EncryptString(t *testing.T) {
	encryptor := createTestEncryptor(t)

	plaintext := "test saga state data"
	encrypted, err := encryptor.EncryptString(plaintext)

	assert.NoError(t, err)
	assert.NotNil(t, encrypted)
	assert.NotEmpty(t, encrypted.Ciphertext)
}

func TestAESGCMEncryptor_DecryptString(t *testing.T) {
	encryptor := createTestEncryptor(t)

	plaintext := "test saga context"
	encrypted, err := encryptor.EncryptString(plaintext)
	require.NoError(t, err)

	decrypted, err := encryptor.DecryptString(encrypted)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestAESGCMEncryptor_RotateKey(t *testing.T) {
	keyManager := createTestKeyManager(t)
	encryptor, err := NewAESGCMEncryptor(&AESGCMEncryptorConfig{
		KeyManager: keyManager,
	})
	require.NoError(t, err)

	// Get original key ID
	originalKeyID := encryptor.GetKeyID()

	// Encrypt with original key
	plaintext := []byte("test data before rotation")
	encrypted1, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)
	assert.Equal(t, originalKeyID, encrypted1.KeyID)

	// Rotate key
	err = encryptor.RotateKey()
	assert.NoError(t, err)

	// Get new key ID
	newKeyID := encryptor.GetKeyID()
	assert.NotEqual(t, originalKeyID, newKeyID)

	// Encrypt with new key
	encrypted2, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)
	assert.Equal(t, newKeyID, encrypted2.KeyID)

	// Should still be able to decrypt data encrypted with old key
	decrypted1, err := encryptor.Decrypt(encrypted1)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	// Should be able to decrypt data encrypted with new key
	decrypted2, err := encryptor.Decrypt(encrypted2)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)
}

func TestAESGCMEncryptor_IsHealthy(t *testing.T) {
	encryptor := createTestEncryptor(t)
	assert.True(t, encryptor.IsHealthy())
}

func TestAESGCMEncryptor_GetMetrics(t *testing.T) {
	encryptor := createTestEncryptor(t)

	// Initial metrics
	metrics := encryptor.GetMetrics()
	assert.Equal(t, uint64(0), metrics["encrypt_count"])
	assert.Equal(t, uint64(0), metrics["decrypt_count"])
	assert.Equal(t, uint64(0), metrics["error_count"])

	// Perform operations
	plaintext := []byte("test data")
	encrypted, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	_, err = encryptor.Decrypt(encrypted)
	require.NoError(t, err)

	// Trigger errors
	_, _ = encryptor.Decrypt(nil)
	_, _ = encryptor.Encrypt(nil)

	// Check updated metrics
	metrics = encryptor.GetMetrics()
	assert.Equal(t, uint64(1), metrics["encrypt_count"])
	assert.Equal(t, uint64(1), metrics["decrypt_count"])
	assert.GreaterOrEqual(t, metrics["error_count"], uint64(1))
}

func TestAESGCMEncryptor_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	encryptor := createTestEncryptor(t)
	plaintext := []byte("test saga data for performance measurement")

	// Test encryption performance
	encryptStart := time.Now()
	for i := 0; i < 1000; i++ {
		_, err := encryptor.Encrypt(plaintext)
		require.NoError(t, err)
	}
	encryptDuration := time.Since(encryptStart)
	avgEncryptTime := encryptDuration / 1000

	// Encryption should be fast (< 10ms per operation)
	assert.Less(t, avgEncryptTime, 10*time.Millisecond,
		"Average encryption time should be less than 10ms, got %v", avgEncryptTime)

	// Test decryption performance
	encrypted, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	decryptStart := time.Now()
	for i := 0; i < 1000; i++ {
		_, err := encryptor.Decrypt(encrypted)
		require.NoError(t, err)
	}
	decryptDuration := time.Since(decryptStart)
	avgDecryptTime := decryptDuration / 1000

	// Decryption should be fast (< 10ms per operation)
	assert.Less(t, avgDecryptTime, 10*time.Millisecond,
		"Average decryption time should be less than 10ms, got %v", avgDecryptTime)

	t.Logf("Average encryption time: %v", avgEncryptTime)
	t.Logf("Average decryption time: %v", avgDecryptTime)
}

func TestAESGCMEncryptor_ConcurrentOperations(t *testing.T) {
	encryptor := createTestEncryptor(t)
	plaintext := []byte("concurrent test data")

	// Encrypt data once
	encrypted, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	// Run concurrent encrypt/decrypt operations
	const numGoroutines = 100
	errChan := make(chan error, numGoroutines*2)
	done := make(chan bool)

	for i := 0; i < numGoroutines; i++ {
		// Concurrent encryption
		go func() {
			_, err := encryptor.Encrypt(plaintext)
			errChan <- err
		}()

		// Concurrent decryption
		go func() {
			_, err := encryptor.Decrypt(encrypted)
			errChan <- err
		}()
	}

	// Collect results
	go func() {
		for i := 0; i < numGoroutines*2; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("Concurrent operations timed out")
	}
}

func TestAESGCMEncryptor_InvalidKeySize(t *testing.T) {
	// Create key manager with invalid key size
	invalidKey := make([]byte, 16) // 16 bytes instead of 32
	_, err := rand.Read(invalidKey)
	require.NoError(t, err)

	keyManager, err := NewInMemoryKeyManager(&InMemoryKeyManagerConfig{
		InitialKey: invalidKey,
	})
	assert.Error(t, err)
	assert.Nil(t, keyManager)
}

// Helper functions

func createTestKeyManager(t *testing.T) KeyManager {
	keyManager, err := NewInMemoryKeyManager(&InMemoryKeyManagerConfig{
		MaxKeys: 10,
	})
	require.NoError(t, err)
	return keyManager
}

func createTestEncryptor(t *testing.T) *AESGCMEncryptor {
	keyManager := createTestKeyManager(t)
	encryptor, err := NewAESGCMEncryptor(&AESGCMEncryptorConfig{
		KeyManager: keyManager,
	})
	require.NoError(t, err)
	return encryptor
}

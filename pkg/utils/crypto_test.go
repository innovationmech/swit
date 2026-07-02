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
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setEncryptionKey initializes the package-level encryption key from the given
// passphrase and registers cleanup to restore the previous state.
func setEncryptionKey(t *testing.T, passphrase string) {
	t.Helper()
	prev := encryptionKey
	t.Setenv("TOKEN_ENCRYPTION_KEY", passphrase)
	require.NoError(t, InitEncryptionKey())
	t.Cleanup(func() {
		encryptionKey = prev
	})
}

// clearEncryptionKey resets the package-level key and restores it after the test.
func clearEncryptionKey(t *testing.T) {
	t.Helper()
	prev := encryptionKey
	encryptionKey = nil
	t.Cleanup(func() {
		encryptionKey = prev
	})
}

func TestInitEncryptionKey(t *testing.T) {
	t.Run("missing env var returns error", func(t *testing.T) {
		clearEncryptionKey(t)
		t.Setenv("TOKEN_ENCRYPTION_KEY", "")
		err := InitEncryptionKey()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "TOKEN_ENCRYPTION_KEY")
	})

	t.Run("valid env var derives 32-byte key", func(t *testing.T) {
		clearEncryptionKey(t)
		t.Setenv("TOKEN_ENCRYPTION_KEY", "my-secret-passphrase")
		require.NoError(t, InitEncryptionKey())
		assert.Len(t, encryptionKey, 32)
	})

	t.Run("same passphrase derives deterministic key", func(t *testing.T) {
		clearEncryptionKey(t)
		t.Setenv("TOKEN_ENCRYPTION_KEY", "deterministic")
		require.NoError(t, InitEncryptionKey())
		first := make([]byte, len(encryptionKey))
		copy(first, encryptionKey)

		require.NoError(t, InitEncryptionKey())
		assert.Equal(t, first, encryptionKey)
	})
}

func TestEncryptDecryptToken_RoundTrip(t *testing.T) {
	setEncryptionKey(t, "round-trip-key")

	tests := []struct {
		name  string
		token string
	}{
		{name: "simple token", token: "my-jwt-token"},
		{name: "empty token", token: ""},
		{name: "unicode token", token: "令牌-\u00e9\u00e8\u00ea-🎫"},
		{name: "long token", token: strings.Repeat("abcdef0123456789", 256)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := EncryptToken(tt.token)
			require.NoError(t, err)
			assert.NotEqual(t, tt.token, encrypted)

			// Ciphertext must be valid base64
			_, err = base64.StdEncoding.DecodeString(encrypted)
			require.NoError(t, err)

			decrypted, err := DecryptToken(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.token, decrypted)
		})
	}
}

func TestEncryptToken_RandomNonceProducesDistinctCiphertexts(t *testing.T) {
	setEncryptionKey(t, "nonce-key")

	first, err := EncryptToken("same-token")
	require.NoError(t, err)
	second, err := EncryptToken("same-token")
	require.NoError(t, err)

	assert.NotEqual(t, first, second, "AES-GCM with random nonce must not produce identical ciphertexts")
}

func TestEncryptToken_KeyNotInitialized(t *testing.T) {
	clearEncryptionKey(t)

	_, err := EncryptToken("token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestDecryptToken_KeyNotInitialized(t *testing.T) {
	clearEncryptionKey(t)

	_, err := DecryptToken("whatever")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestDecryptToken_WrongKey(t *testing.T) {
	setEncryptionKey(t, "correct-key")
	encrypted, err := EncryptToken("sensitive-token")
	require.NoError(t, err)

	// Re-derive the key from a different passphrase; GCM authentication must fail.
	setEncryptionKey(t, "wrong-key")
	_, err = DecryptToken(encrypted)
	assert.Error(t, err, "decrypting with a different key must fail GCM authentication")
}

func TestDecryptToken_InvalidInput(t *testing.T) {
	setEncryptionKey(t, "invalid-input-key")

	t.Run("invalid base64", func(t *testing.T) {
		_, err := DecryptToken("not-valid-base64!!!")
		assert.Error(t, err)
	})

	t.Run("ciphertext shorter than nonce", func(t *testing.T) {
		short := base64.StdEncoding.EncodeToString([]byte{0x01, 0x02, 0x03})
		_, err := DecryptToken(short)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ciphertext too short")
	})

	t.Run("tampered ciphertext fails authentication", func(t *testing.T) {
		encrypted, err := EncryptToken("tamper-me")
		require.NoError(t, err)

		raw, err := base64.StdEncoding.DecodeString(encrypted)
		require.NoError(t, err)
		raw[len(raw)-1] ^= 0xFF
		_, err = DecryptToken(base64.StdEncoding.EncodeToString(raw))
		assert.Error(t, err)
	})
}

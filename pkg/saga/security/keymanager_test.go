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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInMemoryKeyManager(t *testing.T) {
	tests := []struct {
		name    string
		config  *InMemoryKeyManagerConfig
		wantErr bool
	}{
		{
			name: "valid config with generated key",
			config: &InMemoryKeyManagerConfig{
				MaxKeys: 5,
			},
			wantErr: false,
		},
		{
			name: "valid config with provided key",
			config: &InMemoryKeyManagerConfig{
				InitialKey: make32ByteKey(t),
				MaxKeys:    10,
			},
			wantErr: false,
		},
		{
			name: "invalid initial key size",
			config: &InMemoryKeyManagerConfig{
				InitialKey: make([]byte, 16), // Wrong size
				MaxKeys:    5,
			},
			wantErr: true,
		},
		{
			name: "with key TTL",
			config: &InMemoryKeyManagerConfig{
				MaxKeys: 5,
				KeyTTL:  1 * time.Hour,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewInMemoryKeyManager(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				assert.True(t, manager.IsHealthy())
				assert.NotEmpty(t, manager.GetCurrentKeyID())
			}
		})
	}
}

func TestInMemoryKeyManager_GetCurrentKey(t *testing.T) {
	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys: 5,
	})

	key, err := manager.GetCurrentKey()
	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.NotEmpty(t, key.ID)
	assert.Len(t, key.Key, 32)
	assert.Equal(t, KeyStatusActive, key.Status)
	assert.False(t, key.CreatedAt.IsZero())
}

func TestInMemoryKeyManager_GetKey(t *testing.T) {
	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys: 5,
	})

	// Get current key ID
	currentKeyID := manager.GetCurrentKeyID()

	tests := []struct {
		name    string
		keyID   string
		wantErr bool
		errType error
	}{
		{
			name:    "valid key ID",
			keyID:   currentKeyID,
			wantErr: false,
		},
		{
			name:    "invalid key ID",
			keyID:   "non-existent-key",
			wantErr: true,
			errType: ErrKeyNotFound,
		},
		{
			name:    "empty key ID",
			keyID:   "",
			wantErr: true,
			errType: ErrInvalidKeyID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := manager.GetKey(tt.keyID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				assert.Nil(t, key)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, key)
				assert.Equal(t, tt.keyID, key.ID)
			}
		})
	}
}

func TestInMemoryKeyManager_RotateKey(t *testing.T) {
	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys: 5,
	})

	// Get original key
	originalKeyID := manager.GetCurrentKeyID()
	originalKey, err := manager.GetCurrentKey()
	require.NoError(t, err)

	// Rotate key
	err = manager.RotateKey()
	assert.NoError(t, err)

	// Get new key
	newKeyID := manager.GetCurrentKeyID()
	newKey, err := manager.GetCurrentKey()
	require.NoError(t, err)

	// Verify rotation
	assert.NotEqual(t, originalKeyID, newKeyID)
	assert.NotEqual(t, originalKey.Key, newKey.Key)
	assert.Equal(t, KeyStatusActive, newKey.Status)
	assert.Greater(t, newKey.Version, originalKey.Version)

	// Old key should be retired but still accessible
	oldKey, err := manager.GetKey(originalKeyID)
	assert.NoError(t, err)
	assert.Equal(t, KeyStatusRetired, oldKey.Status)
}

func TestInMemoryKeyManager_MultipleRotations(t *testing.T) {
	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys: 3,
	})

	keyIDs := make([]string, 5)
	keyIDs[0] = manager.GetCurrentKeyID()

	// Rotate keys multiple times
	for i := 1; i < 5; i++ {
		err := manager.RotateKey()
		require.NoError(t, err)
		keyIDs[i] = manager.GetCurrentKeyID()
	}

	// All key IDs should be different
	uniqueKeys := make(map[string]bool)
	for _, id := range keyIDs {
		uniqueKeys[id] = true
	}
	assert.Len(t, uniqueKeys, 5)

	// Should have at most MaxKeys
	assert.LessOrEqual(t, manager.GetKeyCount(), 3)

	// Oldest keys should have been removed
	_, err := manager.GetKey(keyIDs[0])
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestInMemoryKeyManager_ListKeys(t *testing.T) {
	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys: 5,
	})

	// Initially should have 1 key
	keys := manager.ListKeys()
	assert.Len(t, keys, 1)

	// Rotate to add more keys
	err := manager.RotateKey()
	require.NoError(t, err)

	keys = manager.ListKeys()
	assert.Len(t, keys, 2)

	// Check that we have 1 active and 1 retired
	activeCount := 0
	retiredCount := 0
	for _, key := range keys {
		if key.Status == KeyStatusActive {
			activeCount++
		} else if key.Status == KeyStatusRetired {
			retiredCount++
		}
	}
	assert.Equal(t, 1, activeCount)
	assert.Equal(t, 1, retiredCount)
}

func TestInMemoryKeyManager_IsHealthy(t *testing.T) {
	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys: 5,
	})

	// Should be healthy initially
	assert.True(t, manager.IsHealthy())

	// Should remain healthy after rotation
	err := manager.RotateKey()
	require.NoError(t, err)
	assert.True(t, manager.IsHealthy())
}

func TestInMemoryKeyManager_GetKeyCount(t *testing.T) {
	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys: 5,
	})

	// Initially should have 1 key
	assert.Equal(t, 1, manager.GetKeyCount())

	// Add more keys
	for i := 0; i < 3; i++ {
		err := manager.RotateKey()
		require.NoError(t, err)
	}

	assert.Equal(t, 4, manager.GetKeyCount())
}

func TestInMemoryKeyManager_KeyExpiration(t *testing.T) {
	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys: 5,
		KeyTTL:  100 * time.Millisecond,
	})

	// Get initial key
	key, err := manager.GetCurrentKey()
	require.NoError(t, err)
	assert.NotNil(t, key.ExpiresAt)

	// Wait for key to expire
	time.Sleep(150 * time.Millisecond)

	// Key should be expired
	assert.True(t, key.IsExpired())

	// Getting current key should fail
	_, err = manager.GetCurrentKey()
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrKeyExpired)
}

func TestInMemoryKeyManager_CleanupExpiredKeys(t *testing.T) {
	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys: 10,
		KeyTTL:  50 * time.Millisecond,
	})

	// Rotate to create multiple keys
	for i := 0; i < 3; i++ {
		err := manager.RotateKey()
		require.NoError(t, err)
		time.Sleep(20 * time.Millisecond)
	}

	// Should have 4 keys now
	assert.Equal(t, 4, manager.GetKeyCount())

	// Wait for keys to expire
	time.Sleep(100 * time.Millisecond)

	// Cleanup expired keys
	removed := manager.CleanupExpiredKeys()
	assert.Greater(t, removed, 0)
	assert.Less(t, manager.GetKeyCount(), 4)
}

func TestInMemoryKeyManager_AutoRotate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping auto-rotation test in short mode")
	}

	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys:          10,
		AutoRotate:       true,
		RotationInterval: 100 * time.Millisecond,
	})

	// Get initial key ID
	initialKeyID := manager.GetCurrentKeyID()

	// Wait for auto-rotation to occur
	time.Sleep(250 * time.Millisecond)

	// Key should have rotated
	newKeyID := manager.GetCurrentKeyID()
	assert.NotEqual(t, initialKeyID, newKeyID)

	// Should have multiple keys now
	assert.Greater(t, manager.GetKeyCount(), 1)
}

func TestInMemoryKeyManager_MaxKeysLimit(t *testing.T) {
	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys: 3,
	})

	// Rotate to reach max keys
	for i := 0; i < 5; i++ {
		err := manager.RotateKey()
		require.NoError(t, err)
	}

	// Should not exceed max keys
	assert.LessOrEqual(t, manager.GetKeyCount(), 3)
}

func TestEncryptionKey_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		key    *EncryptionKey
		active bool
	}{
		{
			name: "active key without expiration",
			key: &EncryptionKey{
				Status:    KeyStatusActive,
				ExpiresAt: nil,
			},
			active: true,
		},
		{
			name: "active key not expired",
			key: &EncryptionKey{
				Status: KeyStatusActive,
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(1 * time.Hour)
					return &t
				}(),
			},
			active: true,
		},
		{
			name: "active key but expired",
			key: &EncryptionKey{
				Status: KeyStatusActive,
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(-1 * time.Hour)
					return &t
				}(),
			},
			active: false,
		},
		{
			name: "retired key",
			key: &EncryptionKey{
				Status:    KeyStatusRetired,
				ExpiresAt: nil,
			},
			active: false,
		},
		{
			name: "expired key",
			key: &EncryptionKey{
				Status:    KeyStatusExpired,
				ExpiresAt: nil,
			},
			active: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.active, tt.key.IsActive())
		})
	}
}

func TestEncryptionKey_IsExpired(t *testing.T) {
	tests := []struct {
		name    string
		key     *EncryptionKey
		expired bool
	}{
		{
			name: "no expiration",
			key: &EncryptionKey{
				ExpiresAt: nil,
			},
			expired: false,
		},
		{
			name: "not expired",
			key: &EncryptionKey{
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(1 * time.Hour)
					return &t
				}(),
			},
			expired: false,
		},
		{
			name: "expired",
			key: &EncryptionKey{
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(-1 * time.Hour)
					return &t
				}(),
			},
			expired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expired, tt.key.IsExpired())
		})
	}
}

func TestGenerateKeyID(t *testing.T) {
	// Generate multiple IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateKeyID()
		assert.NotEmpty(t, id)
		assert.Contains(t, id, "key-")

		// All IDs should be unique
		assert.False(t, ids[id], "Duplicate key ID generated: %s", id)
		ids[id] = true
	}
}

func TestInMemoryKeyManager_ConcurrentAccess(t *testing.T) {
	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys: 10,
	})

	const numGoroutines = 50
	errChan := make(chan error, numGoroutines*3)
	done := make(chan bool)

	for i := 0; i < numGoroutines; i++ {
		// Concurrent GetCurrentKey
		go func() {
			_, err := manager.GetCurrentKey()
			errChan <- err
		}()

		// Concurrent GetCurrentKeyID
		go func() {
			_ = manager.GetCurrentKeyID()
			errChan <- nil
		}()

		// Concurrent ListKeys
		go func() {
			_ = manager.ListKeys()
			errChan <- nil
		}()
	}

	// Collect results
	go func() {
		for i := 0; i < numGoroutines*3; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent operations timed out")
	}
}

func TestInMemoryKeyManager_RotateConcurrently(t *testing.T) {
	manager := createTestKeyManagerWithConfig(t, &InMemoryKeyManagerConfig{
		MaxKeys: 20,
	})

	const numRotations = 10
	errChan := make(chan error, numRotations)
	done := make(chan bool)

	// Perform concurrent rotations
	for i := 0; i < numRotations; i++ {
		go func() {
			errChan <- manager.RotateKey()
		}()
	}

	// Collect results
	go func() {
		for i := 0; i < numRotations; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}
		done <- true
	}()

	select {
	case <-done:
		// Should have at least multiple keys now
		assert.Greater(t, manager.GetKeyCount(), 1)
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent rotations timed out")
	}
}

// Helper functions

func make32ByteKey(t *testing.T) []byte {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func createTestKeyManagerWithConfig(t *testing.T, config *InMemoryKeyManagerConfig) *InMemoryKeyManager {
	manager, err := NewInMemoryKeyManager(config)
	require.NoError(t, err)
	return manager
}

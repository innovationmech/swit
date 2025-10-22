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
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Common key management errors
var (
	ErrNoKeysAvailable     = errors.New("no encryption keys available")
	ErrInvalidKeyID        = errors.New("invalid key ID")
	ErrKeyRotationDisabled = errors.New("key rotation is disabled")
	ErrMaxKeysReached      = errors.New("maximum number of keys reached")
)

// EncryptionKey represents an encryption key with metadata
type EncryptionKey struct {
	// ID is the unique identifier for this key
	ID string

	// Key is the actual encryption key bytes (32 bytes for AES-256)
	Key []byte

	// CreatedAt is when the key was created
	CreatedAt time.Time

	// ExpiresAt is when the key expires (nil means no expiration)
	ExpiresAt *time.Time

	// Status indicates if the key is active, retired, etc.
	Status KeyStatus

	// Version is the key version number
	Version int
}

// KeyStatus represents the status of an encryption key
type KeyStatus string

const (
	// KeyStatusActive indicates the key is active and can be used for encryption/decryption
	KeyStatusActive KeyStatus = "active"

	// KeyStatusRetired indicates the key is retired and can only be used for decryption
	KeyStatusRetired KeyStatus = "retired"

	// KeyStatusExpired indicates the key has expired and should not be used
	KeyStatusExpired KeyStatus = "expired"
)

// IsActive checks if the key is active
func (k *EncryptionKey) IsActive() bool {
	return k.Status == KeyStatusActive && !k.IsExpired()
}

// IsExpired checks if the key has expired
func (k *EncryptionKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// KeyManager defines the interface for encryption key management
type KeyManager interface {
	// GetCurrentKey returns the current active encryption key
	GetCurrentKey() (*EncryptionKey, error)

	// GetKey returns a specific key by ID
	GetKey(keyID string) (*EncryptionKey, error)

	// RotateKey generates a new key and retires the current one
	RotateKey() error

	// GetCurrentKeyID returns the ID of the current active key
	GetCurrentKeyID() string

	// ListKeys returns all available keys
	ListKeys() []*EncryptionKey

	// IsHealthy checks if the key manager is healthy
	IsHealthy() bool
}

// InMemoryKeyManager implements KeyManager with in-memory storage
type InMemoryKeyManager struct {
	keys          map[string]*EncryptionKey
	currentKeyID  string
	maxKeys       int
	keyTTL        time.Duration
	autoRotate    bool
	rotationMutex sync.Mutex
	mu            sync.RWMutex
	logger        *zap.Logger
	version       int
}

// InMemoryKeyManagerConfig configures the in-memory key manager
type InMemoryKeyManagerConfig struct {
	// InitialKey is the initial encryption key (32 bytes for AES-256)
	// If nil, a random key will be generated
	InitialKey []byte

	// MaxKeys is the maximum number of keys to retain
	MaxKeys int

	// KeyTTL is how long before keys expire (0 means never expire)
	KeyTTL time.Duration

	// AutoRotate enables automatic key rotation
	AutoRotate bool

	// RotationInterval is how often to rotate keys (if AutoRotate is true)
	RotationInterval time.Duration

	// Logger for logging
	Logger *zap.Logger
}

// NewInMemoryKeyManager creates a new in-memory key manager
func NewInMemoryKeyManager(config *InMemoryKeyManagerConfig) (*InMemoryKeyManager, error) {
	if config.MaxKeys <= 0 {
		config.MaxKeys = 10 // Default maximum keys
	}

	manager := &InMemoryKeyManager{
		keys:       make(map[string]*EncryptionKey),
		maxKeys:    config.MaxKeys,
		keyTTL:     config.KeyTTL,
		autoRotate: config.AutoRotate,
		logger:     config.Logger,
		version:    1,
	}

	// Initialize logger if not set
	if manager.logger == nil {
		manager.logger = logger.Logger
		if manager.logger == nil {
			manager.logger = zap.NewNop()
		}
	}

	// Create initial key
	var initialKey []byte
	if config.InitialKey != nil {
		if len(config.InitialKey) != 32 {
			return nil, fmt.Errorf("%w: initial key must be 32 bytes for AES-256", ErrInvalidKeySize)
		}
		initialKey = config.InitialKey
	} else {
		// Generate random key
		initialKey = make([]byte, 32)
		if _, err := rand.Read(initialKey); err != nil {
			return nil, fmt.Errorf("failed to generate initial key: %w", err)
		}
	}

	// Add initial key
	key := &EncryptionKey{
		ID:        generateKeyID(),
		Key:       initialKey,
		CreatedAt: time.Now(),
		Status:    KeyStatusActive,
		Version:   manager.version,
	}

	if config.KeyTTL > 0 {
		expiresAt := time.Now().Add(config.KeyTTL)
		key.ExpiresAt = &expiresAt
	}

	manager.keys[key.ID] = key
	manager.currentKeyID = key.ID

	manager.logger.Info("Key manager initialized",
		zap.String("key_id", key.ID),
		zap.Int("version", key.Version),
		zap.Bool("auto_rotate", config.AutoRotate))

	// Start auto-rotation if enabled
	if config.AutoRotate && config.RotationInterval > 0 {
		go manager.autoRotateLoop(config.RotationInterval)
	}

	return manager, nil
}

// GetCurrentKey implements KeyManager interface
func (m *InMemoryKeyManager) GetCurrentKey() (*EncryptionKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentKeyID == "" {
		return nil, ErrNoKeysAvailable
	}

	key, exists := m.keys[m.currentKeyID]
	if !exists {
		return nil, fmt.Errorf("%w: current key not found", ErrKeyNotFound)
	}

	// Check if key is expired
	if key.IsExpired() {
		return nil, fmt.Errorf("%w: current key expired", ErrKeyExpired)
	}

	return key, nil
}

// GetKey implements KeyManager interface
func (m *InMemoryKeyManager) GetKey(keyID string) (*EncryptionKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if keyID == "" {
		return nil, ErrInvalidKeyID
	}

	key, exists := m.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, keyID)
	}

	return key, nil
}

// RotateKey implements KeyManager interface
func (m *InMemoryKeyManager) RotateKey() error {
	m.rotationMutex.Lock()
	defer m.rotationMutex.Unlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we've reached max keys
	if len(m.keys) >= m.maxKeys {
		// Remove oldest retired key
		if err := m.cleanupOldestRetiredKey(); err != nil {
			return fmt.Errorf("%w: %v", ErrMaxKeysReached, err)
		}
	}

	// Retire current key
	if m.currentKeyID != "" {
		if currentKey, exists := m.keys[m.currentKeyID]; exists {
			currentKey.Status = KeyStatusRetired
			m.logger.Info("Retired encryption key",
				zap.String("key_id", m.currentKeyID),
				zap.Int("version", currentKey.Version))
		}
	}

	// Generate new key
	newKey := make([]byte, 32)
	if _, err := rand.Read(newKey); err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	m.version++
	key := &EncryptionKey{
		ID:        generateKeyID(),
		Key:       newKey,
		CreatedAt: time.Now(),
		Status:    KeyStatusActive,
		Version:   m.version,
	}

	if m.keyTTL > 0 {
		expiresAt := time.Now().Add(m.keyTTL)
		key.ExpiresAt = &expiresAt
	}

	m.keys[key.ID] = key
	m.currentKeyID = key.ID

	m.logger.Info("Created new encryption key",
		zap.String("key_id", key.ID),
		zap.Int("version", key.Version),
		zap.Int("total_keys", len(m.keys)))

	return nil
}

// GetCurrentKeyID implements KeyManager interface
func (m *InMemoryKeyManager) GetCurrentKeyID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentKeyID
}

// ListKeys implements KeyManager interface
func (m *InMemoryKeyManager) ListKeys() []*EncryptionKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]*EncryptionKey, 0, len(m.keys))
	for _, key := range m.keys {
		keys = append(keys, key)
	}

	return keys
}

// IsHealthy implements KeyManager interface
func (m *InMemoryKeyManager) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentKeyID == "" {
		return false
	}

	key, exists := m.keys[m.currentKeyID]
	if !exists {
		return false
	}

	return key.IsActive()
}

// cleanupOldestRetiredKey removes the oldest retired key
// Must be called with write lock held
func (m *InMemoryKeyManager) cleanupOldestRetiredKey() error {
	var oldestKey *EncryptionKey
	var oldestKeyID string

	// Find oldest retired key
	for id, key := range m.keys {
		if key.Status == KeyStatusRetired {
			if oldestKey == nil || key.CreatedAt.Before(oldestKey.CreatedAt) {
				oldestKey = key
				oldestKeyID = id
			}
		}
	}

	if oldestKeyID == "" {
		return errors.New("no retired keys to cleanup")
	}

	delete(m.keys, oldestKeyID)
	m.logger.Info("Cleaned up old encryption key",
		zap.String("key_id", oldestKeyID),
		zap.Int("version", oldestKey.Version))

	return nil
}

// autoRotateLoop automatically rotates keys at the specified interval
func (m *InMemoryKeyManager) autoRotateLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := m.RotateKey(); err != nil {
			m.logger.Error("Automatic key rotation failed",
				zap.Error(err))
		} else {
			m.logger.Info("Automatic key rotation completed",
				zap.String("new_key_id", m.GetCurrentKeyID()))
		}
	}
}

// CleanupExpiredKeys removes expired keys
func (m *InMemoryKeyManager) CleanupExpiredKeys() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, key := range m.keys {
		if key.IsExpired() && key.Status != KeyStatusActive {
			delete(m.keys, id)
			count++
			m.logger.Info("Removed expired encryption key",
				zap.String("key_id", id),
				zap.Int("version", key.Version))
		}
	}

	return count
}

// GetKeyCount returns the number of keys currently managed
func (m *InMemoryKeyManager) GetKeyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.keys)
}

// generateKeyID generates a unique key ID
func generateKeyID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("key-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("key-%s", hex.EncodeToString(b))
}

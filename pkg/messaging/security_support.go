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

package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// KeyStore defines the interface for cryptographic key storage
type KeyStore interface {
	// GetKey retrieves a key from the store
	GetKey(ctx context.Context, keyName string) ([]byte, error)

	// SetKey stores a key in the store
	SetKey(ctx context.Context, keyName string, key []byte) error

	// DeleteKey removes a key from the store
	DeleteKey(ctx context.Context, keyName string) error

	// ListKeys returns all key names in the store
	ListKeys(ctx context.Context) ([]string, error)
}

// InMemoryKeyStore implements an in-memory key store
type InMemoryKeyStore struct {
	keys map[string][]byte
	mu   sync.RWMutex
}

// NewInMemoryKeyStore creates a new in-memory key store
func NewInMemoryKeyStore() *InMemoryKeyStore {
	return &InMemoryKeyStore{
		keys: make(map[string][]byte),
	}
}

// GetKey implements KeyStore interface
func (s *InMemoryKeyStore) GetKey(ctx context.Context, keyName string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, exists := s.keys[keyName]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyName)
	}

	// Return a copy to prevent external modification
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)
	return keyCopy, nil
}

// SetKey implements KeyStore interface
func (s *InMemoryKeyStore) SetKey(ctx context.Context, keyName string, key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store a copy to prevent external modification
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)
	s.keys[keyName] = keyCopy

	return nil
}

// DeleteKey implements KeyStore interface
func (s *InMemoryKeyStore) DeleteKey(ctx context.Context, keyName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.keys[keyName]; !exists {
		return fmt.Errorf("key not found: %s", keyName)
	}

	delete(s.keys, keyName)
	return nil
}

// ListKeys implements KeyStore interface
func (s *InMemoryKeyStore) ListKeys(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.keys))
	for key := range s.keys {
		keys = append(keys, key)
	}

	return keys, nil
}

// EncryptionCache provides caching for encryption operations
type EncryptionCache struct {
	items map[string][]byte
	ttl   time.Duration
	mu    sync.RWMutex
}

// NewEncryptionCache creates a new encryption cache
func NewEncryptionCache() *EncryptionCache {
	cache := &EncryptionCache{
		items: make(map[string][]byte),
		ttl:   time.Hour, // Default TTL
	}

	// Start cleanup routine
	go cache.cleanup()

	return cache
}

// Get retrieves an encrypted result from the cache
func (c *EncryptionCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// For now, just check existence without TTL for simplicity
	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	return item, true
}

// Set stores an encrypted result in the cache
func (c *EncryptionCache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store a copy
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	c.items[key] = valueCopy
}

// cleanup periodically removes old items from the cache
func (c *EncryptionCache) cleanup() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		// Simple cleanup - clear all items for now
		c.items = make(map[string][]byte)
		c.mu.Unlock()
	}
}

// SecurityEventType defines types of security events
type SecurityEventType string

const (
	// SecurityEventTypeEncrypt represents encryption events
	SecurityEventTypeEncrypt SecurityEventType = "encrypt"
	// SecurityEventTypeDecrypt represents decryption events
	SecurityEventTypeDecrypt SecurityEventType = "decrypt"
	// SecurityEventTypeVerify represents signature verification events
	SecurityEventTypeVerify SecurityEventType = "verify"
	// SecurityEventTypeSign represents signing events
	SecurityEventTypeSign SecurityEventType = "sign"
	// SecurityEventTypeKeyRotation represents key rotation events
	SecurityEventTypeKeyRotation SecurityEventType = "key_rotation"
)

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	// Type of security event
	Type SecurityEventType `json:"type"`

	// Message ID associated with the event
	MessageID string `json:"message_id"`

	// Name of the security policy used
	PolicyName string `json:"policy_name"`

	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Success indicates if the operation was successful
	Success bool `json:"success"`

	// Error if the operation failed
	Error error `json:"error,omitempty"`

	// Security metadata associated with the event
	Metadata *SecurityMetadata `json:"metadata,omitempty"`

	// Message type for classification
	MessageType string `json:"message_type,omitempty"`
}

// SecurityMetrics represents security-related metrics
type SecurityMetrics struct {
	// Total number of encryption operations
	EncryptionCount int64 `json:"encryption_count"`

	// Total number of decryption operations
	DecryptionCount int64 `json:"decryption_count"`

	// Total number of signature verification operations
	VerificationCount int64 `json:"verification_count"`

	// Total number of signing operations
	SigningCount int64 `json:"signing_count"`

	// Number of failed encryption operations
	EncryptionFailures int64 `json:"encryption_failures"`

	// Number of failed decryption operations
	DecryptionFailures int64 `json:"decryption_failures"`

	// Number of failed verification operations
	VerificationFailures int64 `json:"verification_failures"`

	// Number of failed signing operations
	SigningFailures int64 `json:"signing_failures"`

	// Average encryption time in milliseconds
	AverageEncryptionTime float64 `json:"average_encryption_time_ms"`

	// Average decryption time in milliseconds
	AverageDecryptionTime float64 `json:"average_decryption_time_ms"`

	// Total keys managed
	TotalKeys int64 `json:"total_keys"`

	// Keys rotated
	KeysRotated int64 `json:"keys_rotated"`

	// Timestamp when metrics were collected
	CollectedAt time.Time `json:"collected_at"`
}

// SecurityAuditor handles security audit logging
type SecurityAuditor struct {
	config *AuditConfig
	logger *zap.Logger
	mu     sync.RWMutex

	// Metrics tracking
	metrics *SecurityMetrics

	// Event log
	events []*SecurityEvent
}

// NewSecurityAuditor creates a new security auditor
func NewSecurityAuditor(config *AuditConfig) *SecurityAuditor {
	if config == nil {
		config = &AuditConfig{Enabled: false}
	}

	return &SecurityAuditor{
		config:  config,
		logger:  zap.NewNop(),
		metrics: &SecurityMetrics{CollectedAt: time.Now()},
		events:  make([]*SecurityEvent, 0),
	}
}

// LogSecurityEvent logs a security event
func (a *SecurityAuditor) LogSecurityEvent(ctx context.Context, event *SecurityEvent) {
	if !a.config.Enabled {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Update metrics
	switch event.Type {
	case SecurityEventTypeEncrypt:
		a.metrics.EncryptionCount++
		if !event.Success {
			a.metrics.EncryptionFailures++
		}
	case SecurityEventTypeDecrypt:
		a.metrics.DecryptionCount++
		if !event.Success {
			a.metrics.DecryptionFailures++
		}
	case SecurityEventTypeVerify:
		a.metrics.VerificationCount++
		if !event.Success {
			a.metrics.VerificationFailures++
		}
	case SecurityEventTypeSign:
		a.metrics.SigningCount++
		if !event.Success {
			a.metrics.SigningFailures++
		}
	}

	// Add to event log
	a.events = append(a.events, event)

	// Log based on configured level
	logFunc := a.logger.Info
	switch a.config.LogLevel {
	case "debug":
		logFunc = a.logger.Debug
	case "warn":
		logFunc = a.logger.Warn
	case "error":
		logFunc = a.logger.Error
	}

	// Create log entry
	logFields := []zap.Field{
		zap.String("event_type", string(event.Type)),
		zap.String("message_id", event.MessageID),
		zap.String("policy_name", event.PolicyName),
		zap.Bool("success", event.Success),
		zap.Time("timestamp", event.Timestamp),
	}

	if event.Error != nil {
		logFields = append(logFields, zap.Error(event.Error))
	}

	if event.Metadata != nil {
		logFields = append(logFields,
			zap.String("encryption_algo", event.Metadata.EncryptionAlgo),
			zap.String("signing_algo", event.Metadata.SigningAlgo),
			zap.Bool("compression_used", event.Metadata.CompressionUsed),
		)
	}

	if event.MessageType != "" {
		logFields = append(logFields, zap.String("message_type", event.MessageType))
	}

	logFunc("Security event", logFields...)
}

// GetMetrics returns current security metrics
func (a *SecurityAuditor) GetMetrics() *SecurityMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Return a copy to prevent external modification
	metrics := *a.metrics
	metrics.CollectedAt = time.Now()
	return &metrics
}

// GetEvents returns security events within the retention period
func (a *SecurityAuditor) GetEvents() []*SecurityEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.config.RetentionPeriod <= 0 {
		return a.events
	}

	// Filter events by retention period
	cutoff := time.Now().Add(-a.config.RetentionPeriod)
	var filtered []*SecurityEvent

	for _, event := range a.events {
		if event.Timestamp.After(cutoff) {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

// SetLogger sets the logger for the auditor
func (a *SecurityAuditor) SetLogger(logger *zap.Logger) {
	a.logger = logger
}

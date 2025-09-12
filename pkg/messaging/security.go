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
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SecurityPolicy defines the security configuration for message encryption and signing
type SecurityPolicy struct {
	// Name of the security policy
	Name string `json:"name" yaml:"name" validate:"required"`

	// Encryption configuration
	Encryption *EncryptionConfig `json:"encryption,omitempty" yaml:"encryption,omitempty"`

	// Signing configuration
	Signing *SigningConfig `json:"signing,omitempty" yaml:"signing,omitempty"`

	// Key management configuration
	KeyManagement *KeyManagementConfig `json:"key_management,omitempty" yaml:"key_management,omitempty"`

	// Audit logging configuration
	Audit *AuditConfig `json:"audit,omitempty" yaml:"audit,omitempty"`

	// Topics this policy applies to (empty means all topics)
	Topics []string `json:"topics,omitempty" yaml:"topics,omitempty"`

	// Message types this policy applies to (empty means all types)
	MessageTypes []string `json:"message_types,omitempty" yaml:"message_types,omitempty"`

	// Enabled indicates if this policy is active
	Enabled bool `json:"enabled" yaml:"enabled" default:"true"`
}

// EncryptionConfig defines encryption settings
type EncryptionConfig struct {
	// Enabled indicates if encryption is enabled
	Enabled bool `json:"enabled" yaml:"enabled" default:"true"`

	// Algorithm to use for encryption
	Algorithm EncryptionAlgorithm `json:"algorithm" yaml:"algorithm" default:"A256GCM"`

	// Key size in bits
	KeySize int `json:"key_size" yaml:"key_size" default:"256"`

	// Key rotation interval
	KeyRotationInterval time.Duration `json:"key_rotation_interval" yaml:"key_rotation_interval" default:"168h"`

	// Compression settings
	Compression *CompressionConfig `json:"compression,omitempty" yaml:"compression,omitempty"`
}

// SigningConfig defines message signing settings
type SigningConfig struct {
	// Enabled indicates if signing is enabled
	Enabled bool `json:"enabled" yaml:"enabled" default:"true"`

	// Algorithm to use for signing
	Algorithm SigningAlgorithm `json:"algorithm" yaml:"algorithm" default:"RS512"`

	// Key ID for signing keys
	KeyID string `json:"key_id,omitempty" yaml:"key_id,omitempty"`

	// Include timestamp in signature
	IncludeTimestamp bool `json:"include_timestamp" yaml:"include_timestamp" default:"true"`

	// Include message ID in signature
	IncludeMessageID bool `json:"include_message_id" yaml:"include_message_id" default:"true"`
}

// KeyManagementConfig defines key management settings
type KeyManagementConfig struct {
	// Provider for key management
	Provider KeyProvider `json:"provider" yaml:"provider" default:"static"`

	// Key store configuration
	KeyStore *KeyStoreConfig `json:"key_store,omitempty" yaml:"key_store,omitempty"`

	// Key derivation configuration
	KeyDerivation *KeyDerivationConfig `json:"key_derivation,omitempty" yaml:"key_derivation,omitempty"`

	// Cache configuration
	Cache *KeyCacheConfig `json:"cache,omitempty" yaml:"cache,omitempty"`
}

// AuditConfig defines audit logging settings
type AuditConfig struct {
	// Enabled indicates if audit logging is enabled
	Enabled bool `json:"enabled" yaml:"enabled" default:"true"`

	// Log level for audit events
	LogLevel string `json:"log_level" yaml:"log_level" default:"info"`

	// Include message content in audit logs
	IncludeContent bool `json:"include_content" yaml:"include_content" default:"false"`

	// Include encryption metadata in audit logs
	IncludeMetadata bool `json:"include_metadata" yaml:"include_metadata" default:"true"`

	// Retention period for audit logs
	RetentionPeriod time.Duration `json:"retention_period" yaml:"retention_period" default:"168h"`
}

// CompressionConfig defines compression settings
type CompressionConfig struct {
	// Enabled indicates if compression is enabled
	Enabled bool `json:"enabled" yaml:"enabled" default:"true"`

	// Algorithm to use for compression
	Algorithm CompressionAlgorithm `json:"algorithm" yaml:"algorithm" default:"gzip"`

	// Threshold for compression (messages smaller than this won't be compressed)
	Threshold int `json:"threshold" yaml:"threshold" default:"1024"`
}

// KeyStoreConfig defines key store settings
type KeyStoreConfig struct {
	// Type of key store
	Type KeyStoreType `json:"type" yaml:"type" default:"memory"`

	// Path for file-based key stores
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// Connection string for database key stores
	ConnectionString string `json:"connection_string,omitempty" yaml:"connection_string,omitempty"`

	// Encryption for key store at rest
	AtRestEncryption bool `json:"at_rest_encryption" yaml:"at_rest_encryption" default:"true"`
}

// KeyDerivationConfig defines key derivation settings
type KeyDerivationConfig struct {
	// Algorithm for key derivation
	Algorithm KeyDerivationAlgorithm `json:"algorithm" yaml:"algorithm" default:"HKDF"`

	// Salt for key derivation
	Salt string `json:"salt,omitempty" yaml:"salt,omitempty"`

	// Info for key derivation
	Info string `json:"info,omitempty" yaml:"info,omitempty"`

	// Iterations for key derivation
	Iterations int `json:"iterations" yaml:"iterations" default:"100000"`
}

// KeyCacheConfig defines key cache settings
type KeyCacheConfig struct {
	// Enabled indicates if key caching is enabled
	Enabled bool `json:"enabled" yaml:"enabled" default:"true"`

	// TTL for cached keys
	TTL time.Duration `json:"ttl" yaml:"ttl" default:"1h"`

	// Maximum number of cached keys
	MaxSize int `json:"max_size" yaml:"max_size" default:"1000"`

	// Cleanup interval for expired keys
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval" default:"10m"`
}

// EncryptionAlgorithm defines supported encryption algorithms
type EncryptionAlgorithm string

const (
	// EncryptionAlgorithmA128GCM represents AES-128-GCM
	EncryptionAlgorithmA128GCM EncryptionAlgorithm = "A128GCM"
	// EncryptionAlgorithmA192GCM represents AES-192-GCM
	EncryptionAlgorithmA192GCM EncryptionAlgorithm = "A192GCM"
	// EncryptionAlgorithmA256GCM represents AES-256-GCM
	EncryptionAlgorithmA256GCM EncryptionAlgorithm = "A256GCM"
	// EncryptionAlgorithmA256CBC represents AES-256-CBC
	EncryptionAlgorithmA256CBC EncryptionAlgorithm = "A256CBC-HS512"
)

// SigningAlgorithm defines supported signing algorithms
type SigningAlgorithm string

const (
	// SigningAlgorithmRS256 represents RSA-SHA256
	SigningAlgorithmRS256 SigningAlgorithm = "RS256"
	// SigningAlgorithmRS384 represents RSA-SHA384
	SigningAlgorithmRS384 SigningAlgorithm = "RS384"
	// SigningAlgorithmRS512 represents RSA-SHA512
	SigningAlgorithmRS512 SigningAlgorithm = "RS512"
	// SigningAlgorithmES256 represents ECDSA-SHA256
	SigningAlgorithmES256 SigningAlgorithm = "ES256"
	// SigningAlgorithmES384 represents ECDSA-SHA384
	SigningAlgorithmES384 SigningAlgorithm = "ES384"
	// SigningAlgorithmES512 represents ECDSA-SHA512
	SigningAlgorithmES512 SigningAlgorithm = "ES512"
	// SigningAlgorithmHS256 represents HMAC-SHA256
	SigningAlgorithmHS256 SigningAlgorithm = "HS256"
	// SigningAlgorithmHS384 represents HMAC-SHA384
	SigningAlgorithmHS384 SigningAlgorithm = "HS384"
	// SigningAlgorithmHS512 represents HMAC-SHA512
	SigningAlgorithmHS512 SigningAlgorithm = "HS512"
)

// KeyProvider defines key management providers
type KeyProvider string

const (
	// KeyProviderStatic uses static keys
	KeyProviderStatic KeyProvider = "static"
	// KeyProviderVault uses HashiCorp Vault
	KeyProviderVault KeyProvider = "vault"
	// KeyProviderAWS uses AWS KMS
	KeyProviderAWS KeyProvider = "aws"
	// KeyProviderGCP uses Google Cloud KMS
	KeyProviderGCP KeyProvider = "gcp"
	// KeyProviderAzure uses Azure Key Vault
	KeyProviderAzure KeyProvider = "azure"
)

// KeyStoreType defines key store types
type KeyStoreType string

const (
	// KeyStoreTypeMemory uses in-memory storage
	KeyStoreTypeMemory KeyStoreType = "memory"
	// KeyStoreTypeFile uses file-based storage
	KeyStoreTypeFile KeyStoreType = "file"
	// KeyStoreTypeDatabase uses database storage
	KeyStoreTypeDatabase KeyStoreType = "database"
	// KeyStoreTypeRedis uses Redis storage
	KeyStoreTypeRedis KeyStoreType = "redis"
)

// KeyDerivationAlgorithm defines key derivation algorithms
type KeyDerivationAlgorithm string

const (
	// KeyDerivationAlgorithmHKDF uses HKDF
	KeyDerivationAlgorithmHKDF KeyDerivationAlgorithm = "HKDF"
	// KeyDerivationAlgorithmPBKDF2 uses PBKDF2
	KeyDerivationAlgorithmPBKDF2 KeyDerivationAlgorithm = "PBKDF2"
	// KeyDerivationAlgorithmScrypt uses scrypt
	KeyDerivationAlgorithmScrypt KeyDerivationAlgorithm = "scrypt"
)

// CompressionAlgorithm defines compression algorithms
type CompressionAlgorithm string

const (
	// CompressionAlgorithmGzip uses gzip compression
	CompressionAlgorithmGzip CompressionAlgorithm = "gzip"
	// CompressionAlgorithmZlib uses zlib compression
	CompressionAlgorithmZlib CompressionAlgorithm = "zlib"
	// CompressionAlgorithmNone disables compression
	CompressionAlgorithmNone CompressionAlgorithm = "none"
)

// SecurityManager manages message security operations
type SecurityManager struct {
	policies  map[string]*SecurityPolicy
	encryptor *MessageEncryptor
	signer    *MessageSigner
	auditor   *SecurityAuditor
	logger    *zap.Logger
	mu        sync.RWMutex
}

// SecurityManagerConfig configures the security manager
type SecurityManagerConfig struct {
	// Default security policy
	DefaultPolicy *SecurityPolicy `json:"default_policy" yaml:"default_policy"`

	// Additional security policies
	Policies map[string]*SecurityPolicy `json:"policies,omitempty" yaml:"policies,omitempty"`

	// Logger for security operations
	Logger *zap.Logger `json:"-" yaml:"-"`
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config *SecurityManagerConfig) (*SecurityManager, error) {
	if config.DefaultPolicy == nil {
		return nil, fmt.Errorf("default security policy is required")
	}

	// Initialize policies
	policies := make(map[string]*SecurityPolicy)
	policies["default"] = config.DefaultPolicy

	// Add additional policies
	for name, policy := range config.Policies {
		policies[name] = policy
	}

	// Initialize components
	encryptor := NewMessageEncryptor(config.DefaultPolicy.Encryption)
	signer := NewMessageSigner(config.DefaultPolicy.Signing)
	auditor := NewSecurityAuditor(config.DefaultPolicy.Audit)

	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &SecurityManager{
		policies:  policies,
		encryptor: encryptor,
		signer:    signer,
		auditor:   auditor,
		logger:    logger,
	}, nil
}

// GetPolicy returns a security policy by name
func (sm *SecurityManager) GetPolicy(name string) (*SecurityPolicy, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	policy, exists := sm.policies[name]
	return policy, exists
}

// GetPolicyForMessage returns the security policy for a given message
func (sm *SecurityManager) GetPolicyForMessage(message *Message) (*SecurityPolicy, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Check if there's a specific policy for this topic
	for _, policy := range sm.policies {
		if !policy.Enabled {
			continue
		}

		// Check topic match
		if len(policy.Topics) > 0 {
			topicMatch := false
			for _, topic := range policy.Topics {
				if message.Topic == topic {
					topicMatch = true
					break
				}
			}
			if !topicMatch {
				continue
			}
		}

		// Check message type match
		if len(policy.MessageTypes) > 0 {
			typeMatch := false
			for _, msgType := range policy.MessageTypes {
				if message.Headers["message_type"] == msgType {
					typeMatch = true
					break
				}
			}
			if !typeMatch {
				continue
			}
		}

		return policy, nil
	}

	// Return default policy
	return sm.policies["default"], nil
}

// SecureMessage applies security transformations to a message
func (sm *SecurityManager) SecureMessage(ctx context.Context, message *Message) (*SecureMessage, error) {
	policy, err := sm.GetPolicyForMessage(message)
	if err != nil {
		return nil, fmt.Errorf("failed to get security policy: %w", err)
	}

	secureMsg := &SecureMessage{
		OriginalMessage: message,
		SecurityMetadata: &SecurityMetadata{
			PolicyName:      policy.Name,
			EncryptionAlgo:  "",
			SigningAlgo:     "",
			KeyID:           "",
			Timestamp:       time.Now(),
			CompressionUsed: false,
		},
	}

	// Apply compression if enabled
	if policy.Encryption != nil && policy.Encryption.Compression != nil && policy.Encryption.Compression.Enabled {
		if len(message.Payload) >= policy.Encryption.Compression.Threshold {
			compressed, err := sm.compressMessage(message.Payload, string(policy.Encryption.Compression.Algorithm))
			if err != nil {
				sm.logger.Error("Failed to compress message",
					zap.String("message_id", message.ID),
					zap.Error(err))
				return nil, fmt.Errorf("message compression failed: %w", err)
			}
			message.Payload = compressed
			secureMsg.SecurityMetadata.CompressionUsed = true
		}
	}

	// Apply encryption if enabled
	if policy.Encryption != nil && policy.Encryption.Enabled {
		encrypted, err := sm.encryptor.Encrypt(ctx, message.Payload, policy)
		if err != nil {
			sm.logger.Error("Failed to encrypt message",
				zap.String("message_id", message.ID),
				zap.Error(err))
			return nil, fmt.Errorf("message encryption failed: %w", err)
		}
		message.Payload = encrypted
		secureMsg.SecurityMetadata.EncryptionAlgo = string(policy.Encryption.Algorithm)
	}

	// Apply signing if enabled
	if policy.Signing != nil && policy.Signing.Enabled {
		signature, err := sm.signer.Sign(ctx, message, policy)
		if err != nil {
			sm.logger.Error("Failed to sign message",
				zap.String("message_id", message.ID),
				zap.Error(err))
			return nil, fmt.Errorf("message signing failed: %w", err)
		}
		secureMsg.Signature = signature
		secureMsg.SecurityMetadata.SigningAlgo = string(policy.Signing.Algorithm)
		secureMsg.SecurityMetadata.KeyID = policy.Signing.KeyID
	}

	// Log audit event
	if policy.Audit != nil && policy.Audit.Enabled {
		sm.auditor.LogSecurityEvent(ctx, &SecurityEvent{
			Type:        SecurityEventTypeEncrypt,
			MessageID:   message.ID,
			PolicyName:  policy.Name,
			Timestamp:   time.Now(),
			Success:     true,
			Metadata:    secureMsg.SecurityMetadata,
			MessageType: message.Headers["message_type"],
		})
	}

	return secureMsg, nil
}

// VerifyAndDecrypt verifies and decrypts a secure message
func (sm *SecurityManager) VerifyAndDecrypt(ctx context.Context, secureMessage *SecureMessage) (*Message, error) {
	policy, exists := sm.GetPolicy(secureMessage.SecurityMetadata.PolicyName)
	if !exists {
		return nil, fmt.Errorf("security policy not found: %s", secureMessage.SecurityMetadata.PolicyName)
	}

	message := secureMessage.OriginalMessage

	// Verify signature if present
	if secureMessage.Signature != "" && policy.Signing != nil && policy.Signing.Enabled {
		err := sm.signer.Verify(ctx, secureMessage, policy)
		if err != nil {
			sm.logger.Error("Failed to verify message signature",
				zap.String("message_id", message.ID),
				zap.Error(err))

			// Log audit event
			if policy.Audit != nil && policy.Audit.Enabled {
				sm.auditor.LogSecurityEvent(ctx, &SecurityEvent{
					Type:        SecurityEventTypeVerify,
					MessageID:   message.ID,
					PolicyName:  policy.Name,
					Timestamp:   time.Now(),
					Success:     false,
					Error:       err,
					Metadata:    secureMessage.SecurityMetadata,
					MessageType: message.Headers["message_type"],
				})
			}

			return nil, fmt.Errorf("message signature verification failed: %w", err)
		}
	}

	// Decrypt message if encrypted
	if secureMessage.SecurityMetadata.EncryptionAlgo != "" && policy.Encryption != nil && policy.Encryption.Enabled {
		decrypted, err := sm.encryptor.Decrypt(ctx, message.Payload, policy)
		if err != nil {
			sm.logger.Error("Failed to decrypt message",
				zap.String("message_id", message.ID),
				zap.Error(err))

			// Log audit event
			if policy.Audit != nil && policy.Audit.Enabled {
				sm.auditor.LogSecurityEvent(ctx, &SecurityEvent{
					Type:        SecurityEventTypeDecrypt,
					MessageID:   message.ID,
					PolicyName:  policy.Name,
					Timestamp:   time.Now(),
					Success:     false,
					Error:       err,
					Metadata:    secureMessage.SecurityMetadata,
					MessageType: message.Headers["message_type"],
				})
			}

			return nil, fmt.Errorf("message decryption failed: %w", err)
		}
		message.Payload = decrypted
	}

	// Apply decompression if needed
	if secureMessage.SecurityMetadata.CompressionUsed && policy.Encryption != nil && policy.Encryption.Compression != nil {
		decompressed, err := sm.decompressMessage(message.Payload, string(policy.Encryption.Compression.Algorithm))
		if err != nil {
			sm.logger.Error("Failed to decompress message",
				zap.String("message_id", message.ID),
				zap.Error(err))
			return nil, fmt.Errorf("message decompression failed: %w", err)
		}
		message.Payload = decompressed
	}

	// Log audit event
	if policy.Audit != nil && policy.Audit.Enabled {
		sm.auditor.LogSecurityEvent(ctx, &SecurityEvent{
			Type:        SecurityEventTypeDecrypt,
			MessageID:   message.ID,
			PolicyName:  policy.Name,
			Timestamp:   time.Now(),
			Success:     true,
			Metadata:    secureMessage.SecurityMetadata,
			MessageType: message.Headers["message_type"],
		})
	}

	return message, nil
}

// AddPolicy adds a new security policy
func (sm *SecurityManager) AddPolicy(name string, policy *SecurityPolicy) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.policies[name]; exists {
		return fmt.Errorf("security policy already exists: %s", name)
	}

	// Validate policy
	if err := policy.Validate(); err != nil {
		return fmt.Errorf("invalid security policy: %w", err)
	}

	sm.policies[name] = policy
	return nil
}

// RemovePolicy removes a security policy
func (sm *SecurityManager) RemovePolicy(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if name == "default" {
		return fmt.Errorf("cannot remove default security policy")
	}

	if _, exists := sm.policies[name]; !exists {
		return fmt.Errorf("security policy not found: %s", name)
	}

	delete(sm.policies, name)
	return nil
}

// GetMetrics returns security manager metrics
func (sm *SecurityManager) GetMetrics() *SecurityMetrics {
	return sm.auditor.GetMetrics()
}

// compressMessage compresses data using the specified algorithm
func (sm *SecurityManager) compressMessage(data []byte, algorithm string) ([]byte, error) {
	switch algorithm {
	case "gzip":
		var buf bytes.Buffer
		writer := gzip.NewWriter(&buf)
		if _, err := writer.Write(data); err != nil {
			return nil, fmt.Errorf("gzip compression failed: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("gzip writer close failed: %w", err)
		}
		return buf.Bytes(), nil
	case "zlib":
		var buf bytes.Buffer
		writer := zlib.NewWriter(&buf)
		if _, err := writer.Write(data); err != nil {
			return nil, fmt.Errorf("zlib compression failed: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("zlib writer close failed: %w", err)
		}
		return buf.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algorithm)
	}
}

// decompressMessage decompresses data using the specified algorithm
func (sm *SecurityManager) decompressMessage(data []byte, algorithm string) ([]byte, error) {
	switch algorithm {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("gzip reader creation failed: %w", err)
		}
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("gzip decompression failed: %w", err)
		}
		return decompressed, nil
	case "zlib":
		reader, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("zlib reader creation failed: %w", err)
		}
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("zlib decompression failed: %w", err)
		}
		return decompressed, nil
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algorithm)
	}
}

// Validate validates a security policy
func (p *SecurityPolicy) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	if p.Encryption != nil {
		if err := p.Encryption.Validate(); err != nil {
			return fmt.Errorf("invalid encryption config: %w", err)
		}
	}

	if p.Signing != nil {
		if err := p.Signing.Validate(); err != nil {
			return fmt.Errorf("invalid signing config: %w", err)
		}
	}

	if p.KeyManagement != nil {
		if err := p.KeyManagement.Validate(); err != nil {
			return fmt.Errorf("invalid key management config: %w", err)
		}
	}

	if p.Audit != nil {
		if err := p.Audit.Validate(); err != nil {
			return fmt.Errorf("invalid audit config: %w", err)
		}
	}

	return nil
}

// Validate validates encryption configuration
func (c *EncryptionConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	validAlgorithms := []EncryptionAlgorithm{
		EncryptionAlgorithmA128GCM,
		EncryptionAlgorithmA192GCM,
		EncryptionAlgorithmA256GCM,
		EncryptionAlgorithmA256CBC,
	}

	valid := false
	for _, algo := range validAlgorithms {
		if c.Algorithm == algo {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid encryption algorithm: %s", c.Algorithm)
	}

	if c.KeySize <= 0 {
		return fmt.Errorf("key size must be positive")
	}

	if c.KeyRotationInterval <= 0 {
		return fmt.Errorf("key rotation interval must be positive")
	}

	if c.Compression != nil {
		if err := c.Compression.Validate(); err != nil {
			return fmt.Errorf("invalid compression config: %w", err)
		}
	}

	return nil
}

// Validate validates signing configuration
func (c *SigningConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	validAlgorithms := []SigningAlgorithm{
		SigningAlgorithmRS256, SigningAlgorithmRS384, SigningAlgorithmRS512,
		SigningAlgorithmES256, SigningAlgorithmES384, SigningAlgorithmES512,
		SigningAlgorithmHS256, SigningAlgorithmHS384, SigningAlgorithmHS512,
	}

	valid := false
	for _, algo := range validAlgorithms {
		if c.Algorithm == algo {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid signing algorithm: %s", c.Algorithm)
	}

	return nil
}

// Validate validates key management configuration
func (c *KeyManagementConfig) Validate() error {
	validProviders := []KeyProvider{
		KeyProviderStatic, KeyProviderVault, KeyProviderAWS,
		KeyProviderGCP, KeyProviderAzure,
	}

	valid := false
	for _, provider := range validProviders {
		if c.Provider == provider {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid key provider: %s", c.Provider)
	}

	if c.KeyStore != nil {
		if err := c.KeyStore.Validate(); err != nil {
			return fmt.Errorf("invalid key store config: %w", err)
		}
	}

	if c.KeyDerivation != nil {
		if err := c.KeyDerivation.Validate(); err != nil {
			return fmt.Errorf("invalid key derivation config: %w", err)
		}
	}

	if c.Cache != nil {
		if err := c.Cache.Validate(); err != nil {
			return fmt.Errorf("invalid key cache config: %w", err)
		}
	}

	return nil
}

// Validate validates audit configuration
func (c *AuditConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	validLogLevels := []string{"debug", "info", "warn", "error"}
	valid := false
	for _, level := range validLogLevels {
		if c.LogLevel == level {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	if c.RetentionPeriod <= 0 {
		return fmt.Errorf("retention period must be positive")
	}

	return nil
}

// Validate validates compression configuration
func (c *CompressionConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	validAlgorithms := []CompressionAlgorithm{
		CompressionAlgorithmGzip, CompressionAlgorithmZlib, CompressionAlgorithmNone,
	}

	valid := false
	for _, algo := range validAlgorithms {
		if c.Algorithm == algo {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid compression algorithm: %s", c.Algorithm)
	}

	if c.Threshold < 0 {
		return fmt.Errorf("compression threshold cannot be negative")
	}

	return nil
}

// Validate validates key store configuration
func (c *KeyStoreConfig) Validate() error {
	validTypes := []KeyStoreType{
		KeyStoreTypeMemory, KeyStoreTypeFile, KeyStoreTypeDatabase, KeyStoreTypeRedis,
	}

	valid := false
	for _, storeType := range validTypes {
		if c.Type == storeType {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid key store type: %s", c.Type)
	}

	return nil
}

// Validate validates key derivation configuration
func (c *KeyDerivationConfig) Validate() error {
	validAlgorithms := []KeyDerivationAlgorithm{
		KeyDerivationAlgorithmHKDF, KeyDerivationAlgorithmPBKDF2, KeyDerivationAlgorithmScrypt,
	}

	valid := false
	for _, algo := range validAlgorithms {
		if c.Algorithm == algo {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid key derivation algorithm: %s", c.Algorithm)
	}

	if c.Iterations <= 0 {
		return fmt.Errorf("iterations must be positive")
	}

	return nil
}

// Validate validates key cache configuration
func (c *KeyCacheConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.TTL <= 0 {
		return fmt.Errorf("cache TTL must be positive")
	}

	if c.MaxSize <= 0 {
		return fmt.Errorf("max cache size must be positive")
	}

	if c.CleanupInterval <= 0 {
		return fmt.Errorf("cleanup interval must be positive")
	}

	return nil
}

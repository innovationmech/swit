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

// Package security provides unified configuration for all security features.
package security

import (
	"errors"
	"fmt"
	"time"
)

// Common errors for security configuration
var (
	ErrInvalidConfig            = errors.New("invalid security configuration")
	ErrMissingAuthConfig        = errors.New("authentication configuration is required")
	ErrMissingEncryptionConfig  = errors.New("encryption configuration is required")
	ErrInvalidEncryptionKeySize = errors.New("encryption key size must be 16, 24, or 32 bytes")
	ErrInvalidCacheTTL          = errors.New("cache TTL must be positive")
	ErrInvalidMaxFileSize       = errors.New("max file size must be positive")
)

// SecurityConfig is the unified configuration structure for all security features.
// It provides a centralized way to configure authentication, authorization, encryption,
// audit logging, and access control.
type SecurityConfig struct {
	// Authentication configures authentication providers and settings
	Authentication *AuthenticationConfig `json:"authentication" yaml:"authentication"`

	// Authorization configures RBAC and ACL settings
	Authorization *AuthorizationConfig `json:"authorization" yaml:"authorization"`

	// Encryption configures data encryption settings
	Encryption *EncryptionConfig `json:"encryption" yaml:"encryption"`

	// Audit configures audit logging and tracking
	Audit *AuditConfig `json:"audit" yaml:"audit"`

	// DataProtection configures sensitive data protection and masking
	DataProtection *DataProtectionConfig `json:"data_protection" yaml:"data_protection"`
}

// AuthenticationConfig configures authentication providers and settings
type AuthenticationConfig struct {
	// Enabled enables authentication (default: true in production)
	Enabled bool `json:"enabled" yaml:"enabled"`

	// DefaultProvider specifies the default authentication provider (jwt, apikey)
	DefaultProvider string `json:"default_provider" yaml:"default_provider"`

	// JWT configures JWT authentication
	JWT *JWTConfig `json:"jwt,omitempty" yaml:"jwt,omitempty"`

	// APIKey configures API key authentication
	APIKey *APIKeyConfig `json:"api_key,omitempty" yaml:"api_key,omitempty"`

	// Cache configures authentication cache settings
	Cache *CacheConfig `json:"cache,omitempty" yaml:"cache,omitempty"`
}

// JWTConfig configures JWT authentication
type JWTConfig struct {
	// Secret is the JWT signing secret (required)
	Secret string `json:"secret" yaml:"secret"`

	// Issuer is the JWT issuer claim
	Issuer string `json:"issuer,omitempty" yaml:"issuer,omitempty"`

	// Audience is the JWT audience claim
	Audience string `json:"audience,omitempty" yaml:"audience,omitempty"`

	// TokenExpiry is the token expiration duration (default: 1 hour)
	TokenExpiry time.Duration `json:"token_expiry,omitempty" yaml:"token_expiry,omitempty"`
}

// APIKeyConfig configures API key authentication
type APIKeyConfig struct {
	// Keys maps API keys to user IDs
	Keys map[string]string `json:"keys,omitempty" yaml:"keys,omitempty"`

	// KeysFile is a path to a file containing API keys (alternative to Keys)
	KeysFile string `json:"keys_file,omitempty" yaml:"keys_file,omitempty"`
}

// CacheConfig configures authentication and authorization cache settings
type CacheConfig struct {
	// Enabled enables caching (default: true)
	Enabled bool `json:"enabled" yaml:"enabled"`

	// TTL is the cache time-to-live (default: 5 minutes)
	TTL time.Duration `json:"ttl,omitempty" yaml:"ttl,omitempty"`

	// MaxSize is the maximum number of cache entries (default: 1000)
	MaxSize int `json:"max_size,omitempty" yaml:"max_size,omitempty"`
}

// AuthorizationConfig configures RBAC and ACL settings
type AuthorizationConfig struct {
	// Enabled enables authorization (default: true in production)
	Enabled bool `json:"enabled" yaml:"enabled"`

	// RBAC configures role-based access control
	RBAC *RBACConfig `json:"rbac,omitempty" yaml:"rbac,omitempty"`

	// ACL configures access control lists
	ACL *ACLConfig `json:"acl,omitempty" yaml:"acl,omitempty"`

	// Cache configures permission cache settings
	Cache *CacheConfig `json:"cache,omitempty" yaml:"cache,omitempty"`
}

// RBACConfig configures role-based access control
type RBACConfig struct {
	// Enabled enables RBAC (default: true)
	Enabled bool `json:"enabled" yaml:"enabled"`

	// RolesFile is a path to a file defining roles and permissions
	RolesFile string `json:"roles_file,omitempty" yaml:"roles_file,omitempty"`

	// PredefinedRoles enables initialization of predefined admin/user roles
	PredefinedRoles bool `json:"predefined_roles" yaml:"predefined_roles"`
}

// ACLConfig configures access control lists
type ACLConfig struct {
	// Enabled enables ACL (default: false, opt-in)
	Enabled bool `json:"enabled" yaml:"enabled"`

	// DefaultEffect is the default decision when no rules match (allow, deny)
	DefaultEffect string `json:"default_effect" yaml:"default_effect"`

	// RulesFile is a path to a file containing ACL rules
	RulesFile string `json:"rules_file,omitempty" yaml:"rules_file,omitempty"`

	// EnableMetrics enables ACL performance metrics
	EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics"`
}

// EncryptionConfig configures data encryption settings
type EncryptionConfig struct {
	// Enabled enables data encryption (default: true in production)
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Algorithm specifies the encryption algorithm (aes-gcm, aes-cbc)
	// Default: aes-gcm
	Algorithm string `json:"algorithm" yaml:"algorithm"`

	// KeySize specifies the encryption key size in bytes (16, 24, or 32)
	// Default: 32 (AES-256)
	KeySize int `json:"key_size,omitempty" yaml:"key_size,omitempty"`

	// KeyManager configures the key management strategy
	KeyManager *KeyManagerConfig `json:"key_manager,omitempty" yaml:"key_manager,omitempty"`
}

// KeyManagerConfig configures the encryption key manager
type KeyManagerConfig struct {
	// Type specifies the key manager type (memory, file, external)
	// Default: memory
	Type string `json:"type" yaml:"type"`

	// InitialKey is the initial encryption key (hex-encoded or base64)
	InitialKey string `json:"initial_key,omitempty" yaml:"initial_key,omitempty"`

	// KeyFile is the path to a file containing the encryption key
	KeyFile string `json:"key_file,omitempty" yaml:"key_file,omitempty"`

	// RotationInterval is the key rotation interval (0 = disabled)
	RotationInterval time.Duration `json:"rotation_interval,omitempty" yaml:"rotation_interval,omitempty"`
}

// AuditConfig configures audit logging and tracking
type AuditConfig struct {
	// Enabled enables audit logging (default: true in production)
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Storage configures the audit storage backend
	Storage *AuditStorageConfig `json:"storage,omitempty" yaml:"storage,omitempty"`

	// Levels specifies which audit levels to log (info, warning, error, critical)
	// Default: all levels
	Levels []string `json:"levels,omitempty" yaml:"levels,omitempty"`

	// Categories specifies which categories to audit (saga, auth, data, config, security, system)
	// Default: all categories
	Categories []string `json:"categories,omitempty" yaml:"categories,omitempty"`

	// IncludeSensitive includes sensitive data in audit logs (use with caution)
	// Default: false
	IncludeSensitive bool `json:"include_sensitive" yaml:"include_sensitive"`
}

// AuditStorageConfig configures the audit storage backend
type AuditStorageConfig struct {
	// Type specifies the storage type (memory, file, database)
	// Default: file
	Type string `json:"type" yaml:"type"`

	// File configures file-based audit storage
	File *FileStorageConfig `json:"file,omitempty" yaml:"file,omitempty"`

	// Database configures database-based audit storage
	Database *DatabaseStorageConfig `json:"database,omitempty" yaml:"database,omitempty"`

	// RetentionDays is the number of days to retain audit logs (0 = forever)
	RetentionDays int `json:"retention_days,omitempty" yaml:"retention_days,omitempty"`
}

// FileStorageConfig configures file-based audit storage
type FileStorageConfig struct {
	// Path is the file path for audit logs
	Path string `json:"path" yaml:"path"`

	// MaxFileSize is the maximum file size in bytes before rotation (default: 100MB)
	MaxFileSize int64 `json:"max_file_size,omitempty" yaml:"max_file_size,omitempty"`

	// MaxBackups is the maximum number of backup files to keep (default: 10)
	MaxBackups int `json:"max_backups,omitempty" yaml:"max_backups,omitempty"`

	// Compress enables compression of rotated files
	Compress bool `json:"compress" yaml:"compress"`
}

// DatabaseStorageConfig configures database-based audit storage
type DatabaseStorageConfig struct {
	// Driver is the database driver (postgres, mysql, sqlite3)
	Driver string `json:"driver" yaml:"driver"`

	// DSN is the database connection string
	DSN string `json:"dsn" yaml:"dsn"`

	// TableName is the audit log table name (default: audit_logs)
	TableName string `json:"table_name,omitempty" yaml:"table_name,omitempty"`
}

// DataProtectionConfig configures sensitive data protection and masking
type DataProtectionConfig struct {
	// Enabled enables data protection (default: true in production)
	Enabled bool `json:"enabled" yaml:"enabled"`

	// MaskingRules configures data masking rules
	MaskingRules *MaskingRulesConfig `json:"masking_rules,omitempty" yaml:"masking_rules,omitempty"`

	// SensitiveFields is a list of field names to treat as sensitive
	SensitiveFields []string `json:"sensitive_fields,omitempty" yaml:"sensitive_fields,omitempty"`
}

// MaskingRulesConfig configures data masking rules
type MaskingRulesConfig struct {
	// DefaultMaskChar is the character used for masking (default: *)
	DefaultMaskChar string `json:"default_mask_char,omitempty" yaml:"default_mask_char,omitempty"`

	// EmailStrategy specifies email masking strategy (full, partial, domain)
	EmailStrategy string `json:"email_strategy,omitempty" yaml:"email_strategy,omitempty"`

	// PhoneStrategy specifies phone number masking strategy (full, partial, last4)
	PhoneStrategy string `json:"phone_strategy,omitempty" yaml:"phone_strategy,omitempty"`

	// CreditCardStrategy specifies credit card masking strategy (full, last4)
	CreditCardStrategy string `json:"credit_card_strategy,omitempty" yaml:"credit_card_strategy,omitempty"`
}

// Validate validates the security configuration
func (c *SecurityConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: security config is nil", ErrInvalidConfig)
	}

	// Validate authentication config
	if c.Authentication != nil && c.Authentication.Enabled {
		if err := c.Authentication.Validate(); err != nil {
			return fmt.Errorf("authentication config: %w", err)
		}
	}

	// Validate authorization config
	if c.Authorization != nil && c.Authorization.Enabled {
		if err := c.Authorization.Validate(); err != nil {
			return fmt.Errorf("authorization config: %w", err)
		}
	}

	// Validate encryption config
	if c.Encryption != nil && c.Encryption.Enabled {
		if err := c.Encryption.Validate(); err != nil {
			return fmt.Errorf("encryption config: %w", err)
		}
	}

	// Validate audit config
	if c.Audit != nil && c.Audit.Enabled {
		if err := c.Audit.Validate(); err != nil {
			return fmt.Errorf("audit config: %w", err)
		}
	}

	// Validate data protection config
	if c.DataProtection != nil && c.DataProtection.Enabled {
		if err := c.DataProtection.Validate(); err != nil {
			return fmt.Errorf("data protection config: %w", err)
		}
	}

	return nil
}

// Validate validates the authentication configuration
func (c *AuthenticationConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.DefaultProvider == "" {
		return fmt.Errorf("default_provider is required when authentication is enabled")
	}

	switch c.DefaultProvider {
	case "jwt":
		if c.JWT == nil {
			return fmt.Errorf("jwt config is required when default_provider is jwt")
		}
		if err := c.JWT.Validate(); err != nil {
			return err
		}
	case "apikey":
		if c.APIKey == nil {
			return fmt.Errorf("api_key config is required when default_provider is apikey")
		}
		if err := c.APIKey.Validate(); err != nil {
			return err
		}
	case "none":
		// Allow "none" for development/testing
	default:
		return fmt.Errorf("%w: %s (supported: jwt, apikey, none)", ErrInvalidAuthType, c.DefaultProvider)
	}

	// Validate cache config
	if c.Cache != nil && c.Cache.Enabled {
		if err := c.Cache.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the JWT configuration
func (c *JWTConfig) Validate() error {
	if c.Secret == "" {
		return fmt.Errorf("jwt secret is required")
	}

	if len(c.Secret) < 32 {
		return fmt.Errorf("jwt secret must be at least 32 characters for security")
	}

	return nil
}

// Validate validates the API key configuration
func (c *APIKeyConfig) Validate() error {
	if len(c.Keys) == 0 && c.KeysFile == "" {
		return fmt.Errorf("either keys or keys_file must be provided")
	}

	return nil
}

// Validate validates the cache configuration
func (c *CacheConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.TTL < 0 {
		return ErrInvalidCacheTTL
	}

	if c.MaxSize < 0 {
		return fmt.Errorf("cache max_size must be non-negative")
	}

	return nil
}

// Validate validates the authorization configuration
func (c *AuthorizationConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	// Validate RBAC config
	if c.RBAC != nil && c.RBAC.Enabled {
		if err := c.RBAC.Validate(); err != nil {
			return err
		}
	}

	// Validate ACL config
	if c.ACL != nil && c.ACL.Enabled {
		if err := c.ACL.Validate(); err != nil {
			return err
		}
	}

	// Validate cache config
	if c.Cache != nil && c.Cache.Enabled {
		if err := c.Cache.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the RBAC configuration
func (c *RBACConfig) Validate() error {
	// No required fields for RBAC
	return nil
}

// Validate validates the ACL configuration
func (c *ACLConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.DefaultEffect != "" && c.DefaultEffect != "allow" && c.DefaultEffect != "deny" {
		return fmt.Errorf("invalid default_effect: %s (must be 'allow' or 'deny')", c.DefaultEffect)
	}

	return nil
}

// Validate validates the encryption configuration
func (c *EncryptionConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Algorithm != "" && c.Algorithm != "aes-gcm" && c.Algorithm != "aes-cbc" {
		return fmt.Errorf("invalid encryption algorithm: %s (supported: aes-gcm, aes-cbc)", c.Algorithm)
	}

	if c.KeySize != 0 && c.KeySize != 16 && c.KeySize != 24 && c.KeySize != 32 {
		return ErrInvalidEncryptionKeySize
	}

	// Validate key manager config
	if c.KeyManager != nil {
		if err := c.KeyManager.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the key manager configuration
func (c *KeyManagerConfig) Validate() error {
	if c.Type != "" && c.Type != "memory" && c.Type != "file" && c.Type != "external" {
		return fmt.Errorf("invalid key manager type: %s (supported: memory, file, external)", c.Type)
	}

	if c.Type == "file" && c.KeyFile == "" {
		return fmt.Errorf("key_file is required when key manager type is 'file'")
	}

	if c.RotationInterval < 0 {
		return fmt.Errorf("rotation_interval must be non-negative")
	}

	return nil
}

// Validate validates the audit configuration
func (c *AuditConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	// Validate storage config
	if c.Storage != nil {
		if err := c.Storage.Validate(); err != nil {
			return err
		}
	}

	// Validate levels
	validLevels := map[string]bool{"info": true, "warning": true, "error": true, "critical": true}
	for _, level := range c.Levels {
		if !validLevels[level] {
			return fmt.Errorf("invalid audit level: %s (supported: info, warning, error, critical)", level)
		}
	}

	// Validate categories
	validCategories := map[string]bool{
		"saga": true, "auth": true, "data": true,
		"config": true, "security": true, "system": true,
	}
	for _, category := range c.Categories {
		if !validCategories[category] {
			return fmt.Errorf("invalid audit category: %s", category)
		}
	}

	return nil
}

// Validate validates the audit storage configuration
func (c *AuditStorageConfig) Validate() error {
	if c.Type != "" && c.Type != "memory" && c.Type != "file" && c.Type != "database" {
		return fmt.Errorf("invalid audit storage type: %s (supported: memory, file, database)", c.Type)
	}

	// Validate file storage config
	if c.Type == "file" && c.File != nil {
		if err := c.File.Validate(); err != nil {
			return err
		}
	}

	// Validate database storage config
	if c.Type == "database" && c.Database != nil {
		if err := c.Database.Validate(); err != nil {
			return err
		}
	}

	if c.RetentionDays < 0 {
		return fmt.Errorf("retention_days must be non-negative")
	}

	return nil
}

// Validate validates the file storage configuration
func (c *FileStorageConfig) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("file path is required for file-based audit storage")
	}

	if c.MaxFileSize < 0 {
		return ErrInvalidMaxFileSize
	}

	if c.MaxBackups < 0 {
		return fmt.Errorf("max_backups must be non-negative")
	}

	return nil
}

// Validate validates the database storage configuration
func (c *DatabaseStorageConfig) Validate() error {
	if c.Driver == "" {
		return fmt.Errorf("database driver is required")
	}

	if c.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}

	validDrivers := map[string]bool{"postgres": true, "mysql": true, "sqlite3": true}
	if !validDrivers[c.Driver] {
		return fmt.Errorf("unsupported database driver: %s (supported: postgres, mysql, sqlite3)", c.Driver)
	}

	return nil
}

// Validate validates the data protection configuration
func (c *DataProtectionConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	// Validate masking rules
	if c.MaskingRules != nil {
		if err := c.MaskingRules.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the masking rules configuration
func (c *MaskingRulesConfig) Validate() error {
	validEmailStrategies := map[string]bool{"full": true, "partial": true, "domain": true}
	if c.EmailStrategy != "" && !validEmailStrategies[c.EmailStrategy] {
		return fmt.Errorf("invalid email strategy: %s (supported: full, partial, domain)", c.EmailStrategy)
	}

	validPhoneStrategies := map[string]bool{"full": true, "partial": true, "last4": true}
	if c.PhoneStrategy != "" && !validPhoneStrategies[c.PhoneStrategy] {
		return fmt.Errorf("invalid phone strategy: %s (supported: full, partial, last4)", c.PhoneStrategy)
	}

	validCCStrategies := map[string]bool{"full": true, "last4": true}
	if c.CreditCardStrategy != "" && !validCCStrategies[c.CreditCardStrategy] {
		return fmt.Errorf("invalid credit card strategy: %s (supported: full, last4)", c.CreditCardStrategy)
	}

	return nil
}

// SetDefaults applies default values to the security configuration
func (c *SecurityConfig) SetDefaults() {
	if c.Authentication != nil {
		c.Authentication.SetDefaults()
	}

	if c.Authorization != nil {
		c.Authorization.SetDefaults()
	}

	if c.Encryption != nil {
		c.Encryption.SetDefaults()
	}

	if c.Audit != nil {
		c.Audit.SetDefaults()
	}

	if c.DataProtection != nil {
		c.DataProtection.SetDefaults()
	}
}

// SetDefaults applies default values to the authentication configuration
func (c *AuthenticationConfig) SetDefaults() {
	if c.JWT != nil {
		c.JWT.SetDefaults()
	}

	if c.Cache == nil {
		c.Cache = &CacheConfig{
			Enabled: true,
			TTL:     5 * time.Minute,
			MaxSize: 1000,
		}
	} else {
		c.Cache.SetDefaults()
	}
}

// SetDefaults applies default values to the JWT configuration
func (c *JWTConfig) SetDefaults() {
	if c.TokenExpiry == 0 {
		c.TokenExpiry = 1 * time.Hour
	}
}

// SetDefaults applies default values to the cache configuration
func (c *CacheConfig) SetDefaults() {
	if c.Enabled && c.TTL == 0 {
		c.TTL = 5 * time.Minute
	}

	if c.Enabled && c.MaxSize == 0 {
		c.MaxSize = 1000
	}
}

// SetDefaults applies default values to the authorization configuration
func (c *AuthorizationConfig) SetDefaults() {
	if c.RBAC != nil {
		c.RBAC.SetDefaults()
	}

	if c.ACL != nil {
		c.ACL.SetDefaults()
	}

	if c.Cache == nil {
		c.Cache = &CacheConfig{
			Enabled: true,
			TTL:     5 * time.Minute,
			MaxSize: 1000,
		}
	} else {
		c.Cache.SetDefaults()
	}
}

// SetDefaults applies default values to the RBAC configuration
func (c *RBACConfig) SetDefaults() {
	// RBAC defaults are set in NewRBACManager
}

// SetDefaults applies default values to the ACL configuration
func (c *ACLConfig) SetDefaults() {
	if c.DefaultEffect == "" {
		c.DefaultEffect = "deny"
	}
}

// SetDefaults applies default values to the encryption configuration
func (c *EncryptionConfig) SetDefaults() {
	if c.Algorithm == "" {
		c.Algorithm = "aes-gcm"
	}

	if c.KeySize == 0 {
		c.KeySize = 32 // AES-256
	}

	if c.KeyManager != nil {
		c.KeyManager.SetDefaults()
	}
}

// SetDefaults applies default values to the key manager configuration
func (c *KeyManagerConfig) SetDefaults() {
	if c.Type == "" {
		c.Type = "memory"
	}
}

// SetDefaults applies default values to the audit configuration
func (c *AuditConfig) SetDefaults() {
	if c.Storage == nil {
		c.Storage = &AuditStorageConfig{
			Type: "file",
			File: &FileStorageConfig{
				Path:        "/var/log/swit/audit.log",
				MaxFileSize: 100 * 1024 * 1024, // 100MB
				MaxBackups:  10,
				Compress:    true,
			},
		}
	} else {
		c.Storage.SetDefaults()
	}

	if len(c.Levels) == 0 {
		c.Levels = []string{"info", "warning", "error", "critical"}
	}

	if len(c.Categories) == 0 {
		c.Categories = []string{"saga", "auth", "data", "config", "security", "system"}
	}
}

// SetDefaults applies default values to the audit storage configuration
func (c *AuditStorageConfig) SetDefaults() {
	if c.Type == "" {
		c.Type = "file"
	}

	if c.Type == "file" && c.File != nil {
		c.File.SetDefaults()
	}

	if c.Type == "database" && c.Database != nil {
		c.Database.SetDefaults()
	}
}

// SetDefaults applies default values to the file storage configuration
func (c *FileStorageConfig) SetDefaults() {
	if c.MaxFileSize == 0 {
		c.MaxFileSize = 100 * 1024 * 1024 // 100MB
	}

	if c.MaxBackups == 0 {
		c.MaxBackups = 10
	}
}

// SetDefaults applies default values to the database storage configuration
func (c *DatabaseStorageConfig) SetDefaults() {
	if c.TableName == "" {
		c.TableName = "audit_logs"
	}
}

// SetDefaults applies default values to the data protection configuration
func (c *DataProtectionConfig) SetDefaults() {
	if c.MaskingRules == nil {
		c.MaskingRules = &MaskingRulesConfig{
			DefaultMaskChar:    "*",
			EmailStrategy:      "partial",
			PhoneStrategy:      "last4",
			CreditCardStrategy: "last4",
		}
	} else {
		c.MaskingRules.SetDefaults()
	}

	if len(c.SensitiveFields) == 0 {
		c.SensitiveFields = []string{
			"password", "secret", "token", "api_key",
			"credit_card", "ssn", "email", "phone",
		}
	}
}

// SetDefaults applies default values to the masking rules configuration
func (c *MaskingRulesConfig) SetDefaults() {
	if c.DefaultMaskChar == "" {
		c.DefaultMaskChar = "*"
	}

	if c.EmailStrategy == "" {
		c.EmailStrategy = "partial"
	}

	if c.PhoneStrategy == "" {
		c.PhoneStrategy = "last4"
	}

	if c.CreditCardStrategy == "" {
		c.CreditCardStrategy = "last4"
	}
}

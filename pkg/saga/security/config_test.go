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
	"errors"
	"testing"
	"time"
)

func TestSecurityConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *SecurityConfig
		wantErr bool
		errType error
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errType: ErrInvalidConfig,
		},
		{
			name: "valid minimal config",
			config: &SecurityConfig{
				Authentication: &AuthenticationConfig{
					Enabled:         true,
					DefaultProvider: "jwt",
					JWT: &JWTConfig{
						Secret: "this-is-a-very-secure-secret-key-32chars",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid full config",
			config: &SecurityConfig{
				Authentication: &AuthenticationConfig{
					Enabled:         true,
					DefaultProvider: "jwt",
					JWT: &JWTConfig{
						Secret:      "this-is-a-very-secure-secret-key-32chars",
						Issuer:      "swit",
						Audience:    "saga",
						TokenExpiry: 1 * time.Hour,
					},
					Cache: &CacheConfig{
						Enabled: true,
						TTL:     5 * time.Minute,
						MaxSize: 1000,
					},
				},
				Authorization: &AuthorizationConfig{
					Enabled: true,
					RBAC: &RBACConfig{
						Enabled:         true,
						PredefinedRoles: true,
					},
					ACL: &ACLConfig{
						Enabled:       true,
						DefaultEffect: "deny",
						EnableMetrics: true,
					},
				},
				Encryption: &EncryptionConfig{
					Enabled:   true,
					Algorithm: "aes-gcm",
					KeySize:   32,
					KeyManager: &KeyManagerConfig{
						Type: "memory",
					},
				},
				Audit: &AuditConfig{
					Enabled: true,
					Storage: &AuditStorageConfig{
						Type: "file",
						File: &FileStorageConfig{
							Path:        "/var/log/audit.log",
							MaxFileSize: 100 * 1024 * 1024,
							MaxBackups:  10,
							Compress:    true,
						},
					},
					Levels:     []string{"info", "warning", "error", "critical"},
					Categories: []string{"saga", "auth", "data"},
				},
				DataProtection: &DataProtectionConfig{
					Enabled: true,
					MaskingRules: &MaskingRulesConfig{
						DefaultMaskChar:    "*",
						EmailStrategy:      "partial",
						PhoneStrategy:      "last4",
						CreditCardStrategy: "last4",
					},
					SensitiveFields: []string{"password", "secret"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid authentication config",
			config: &SecurityConfig{
				Authentication: &AuthenticationConfig{
					Enabled:         true,
					DefaultProvider: "invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SecurityConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errType != nil && err != nil && !errors.Is(err, tt.errType) {
				t.Errorf("SecurityConfig.Validate() error type = %T, want %T", err, tt.errType)
			}
		})
	}
}

func TestAuthenticationConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *AuthenticationConfig
		wantErr bool
	}{
		{
			name: "disabled authentication",
			config: &AuthenticationConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "valid JWT auth",
			config: &AuthenticationConfig{
				Enabled:         true,
				DefaultProvider: "jwt",
				JWT: &JWTConfig{
					Secret: "this-is-a-very-secure-secret-key-32chars",
				},
			},
			wantErr: false,
		},
		{
			name: "valid API key auth",
			config: &AuthenticationConfig{
				Enabled:         true,
				DefaultProvider: "apikey",
				APIKey: &APIKeyConfig{
					Keys: map[string]string{"key1": "user1"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing default provider",
			config: &AuthenticationConfig{
				Enabled: true,
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			config: &AuthenticationConfig{
				Enabled:         true,
				DefaultProvider: "invalid",
			},
			wantErr: true,
		},
		{
			name: "JWT provider without config",
			config: &AuthenticationConfig{
				Enabled:         true,
				DefaultProvider: "jwt",
			},
			wantErr: true,
		},
		{
			name: "API key provider without config",
			config: &AuthenticationConfig{
				Enabled:         true,
				DefaultProvider: "apikey",
			},
			wantErr: true,
		},
		{
			name: "valid 'none' provider",
			config: &AuthenticationConfig{
				Enabled:         true,
				DefaultProvider: "none",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthenticationConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJWTConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *JWTConfig
		wantErr bool
	}{
		{
			name: "valid JWT config",
			config: &JWTConfig{
				Secret: "this-is-a-very-secure-secret-key-32chars",
			},
			wantErr: false,
		},
		{
			name: "missing secret",
			config: &JWTConfig{
				Secret: "",
			},
			wantErr: true,
		},
		{
			name: "secret too short",
			config: &JWTConfig{
				Secret: "short",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("JWTConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAPIKeyConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *APIKeyConfig
		wantErr bool
	}{
		{
			name: "valid with keys",
			config: &APIKeyConfig{
				Keys: map[string]string{"key1": "user1"},
			},
			wantErr: false,
		},
		{
			name: "valid with keys file",
			config: &APIKeyConfig{
				KeysFile: "/path/to/keys",
			},
			wantErr: false,
		},
		{
			name:    "missing both keys and file",
			config:  &APIKeyConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("APIKeyConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCacheConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *CacheConfig
		wantErr bool
	}{
		{
			name: "valid cache config",
			config: &CacheConfig{
				Enabled: true,
				TTL:     5 * time.Minute,
				MaxSize: 1000,
			},
			wantErr: false,
		},
		{
			name: "disabled cache",
			config: &CacheConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "negative TTL",
			config: &CacheConfig{
				Enabled: true,
				TTL:     -1 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "negative max size",
			config: &CacheConfig{
				Enabled: true,
				MaxSize: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CacheConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthorizationConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *AuthorizationConfig
		wantErr bool
	}{
		{
			name: "valid authorization config",
			config: &AuthorizationConfig{
				Enabled: true,
				RBAC: &RBACConfig{
					Enabled: true,
				},
				ACL: &ACLConfig{
					Enabled:       true,
					DefaultEffect: "deny",
				},
			},
			wantErr: false,
		},
		{
			name: "disabled authorization",
			config: &AuthorizationConfig{
				Enabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthorizationConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestACLConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ACLConfig
		wantErr bool
	}{
		{
			name: "valid ACL with allow effect",
			config: &ACLConfig{
				Enabled:       true,
				DefaultEffect: "allow",
			},
			wantErr: false,
		},
		{
			name: "valid ACL with deny effect",
			config: &ACLConfig{
				Enabled:       true,
				DefaultEffect: "deny",
			},
			wantErr: false,
		},
		{
			name: "invalid default effect",
			config: &ACLConfig{
				Enabled:       true,
				DefaultEffect: "invalid",
			},
			wantErr: true,
		},
		{
			name: "disabled ACL",
			config: &ACLConfig{
				Enabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ACLConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *EncryptionConfig
		wantErr bool
	}{
		{
			name: "valid encryption config",
			config: &EncryptionConfig{
				Enabled:   true,
				Algorithm: "aes-gcm",
				KeySize:   32,
			},
			wantErr: false,
		},
		{
			name: "valid with AES-CBC",
			config: &EncryptionConfig{
				Enabled:   true,
				Algorithm: "aes-cbc",
				KeySize:   32,
			},
			wantErr: false,
		},
		{
			name: "invalid algorithm",
			config: &EncryptionConfig{
				Enabled:   true,
				Algorithm: "des",
			},
			wantErr: true,
		},
		{
			name: "invalid key size",
			config: &EncryptionConfig{
				Enabled: true,
				KeySize: 20,
			},
			wantErr: true,
		},
		{
			name: "valid AES-128",
			config: &EncryptionConfig{
				Enabled: true,
				KeySize: 16,
			},
			wantErr: false,
		},
		{
			name: "valid AES-192",
			config: &EncryptionConfig{
				Enabled: true,
				KeySize: 24,
			},
			wantErr: false,
		},
		{
			name: "disabled encryption",
			config: &EncryptionConfig{
				Enabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptionConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKeyManagerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *KeyManagerConfig
		wantErr bool
	}{
		{
			name: "valid memory key manager",
			config: &KeyManagerConfig{
				Type: "memory",
			},
			wantErr: false,
		},
		{
			name: "valid file key manager",
			config: &KeyManagerConfig{
				Type:    "file",
				KeyFile: "/path/to/key",
			},
			wantErr: false,
		},
		{
			name: "file key manager without key file",
			config: &KeyManagerConfig{
				Type: "file",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			config: &KeyManagerConfig{
				Type: "invalid",
			},
			wantErr: true,
		},
		{
			name: "negative rotation interval",
			config: &KeyManagerConfig{
				Type:             "memory",
				RotationInterval: -1 * time.Hour,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("KeyManagerConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuditConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *AuditConfig
		wantErr bool
	}{
		{
			name: "valid audit config",
			config: &AuditConfig{
				Enabled: true,
				Storage: &AuditStorageConfig{
					Type: "file",
					File: &FileStorageConfig{
						Path: "/var/log/audit.log",
					},
				},
				Levels:     []string{"info", "error"},
				Categories: []string{"saga", "auth"},
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			config: &AuditConfig{
				Enabled: true,
				Levels:  []string{"invalid"},
			},
			wantErr: true,
		},
		{
			name: "invalid category",
			config: &AuditConfig{
				Enabled:    true,
				Categories: []string{"invalid"},
			},
			wantErr: true,
		},
		{
			name: "disabled audit",
			config: &AuditConfig{
				Enabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AuditConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuditStorageConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *AuditStorageConfig
		wantErr bool
	}{
		{
			name: "valid file storage",
			config: &AuditStorageConfig{
				Type: "file",
				File: &FileStorageConfig{
					Path: "/var/log/audit.log",
				},
			},
			wantErr: false,
		},
		{
			name: "valid database storage",
			config: &AuditStorageConfig{
				Type: "database",
				Database: &DatabaseStorageConfig{
					Driver: "postgres",
					DSN:    "postgres://user:pass@localhost/db",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			config: &AuditStorageConfig{
				Type: "invalid",
			},
			wantErr: true,
		},
		{
			name: "negative retention days",
			config: &AuditStorageConfig{
				Type:          "memory",
				RetentionDays: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AuditStorageConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileStorageConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *FileStorageConfig
		wantErr bool
	}{
		{
			name: "valid file storage",
			config: &FileStorageConfig{
				Path:        "/var/log/audit.log",
				MaxFileSize: 100 * 1024 * 1024,
				MaxBackups:  10,
			},
			wantErr: false,
		},
		{
			name: "missing path",
			config: &FileStorageConfig{
				MaxFileSize: 100 * 1024 * 1024,
			},
			wantErr: true,
		},
		{
			name: "negative max file size",
			config: &FileStorageConfig{
				Path:        "/var/log/audit.log",
				MaxFileSize: -1,
			},
			wantErr: true,
		},
		{
			name: "negative max backups",
			config: &FileStorageConfig{
				Path:       "/var/log/audit.log",
				MaxBackups: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FileStorageConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDatabaseStorageConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *DatabaseStorageConfig
		wantErr bool
	}{
		{
			name: "valid postgres",
			config: &DatabaseStorageConfig{
				Driver: "postgres",
				DSN:    "postgres://user:pass@localhost/db",
			},
			wantErr: false,
		},
		{
			name: "valid mysql",
			config: &DatabaseStorageConfig{
				Driver: "mysql",
				DSN:    "user:pass@tcp(localhost:3306)/db",
			},
			wantErr: false,
		},
		{
			name: "missing driver",
			config: &DatabaseStorageConfig{
				DSN: "postgres://user:pass@localhost/db",
			},
			wantErr: true,
		},
		{
			name: "missing DSN",
			config: &DatabaseStorageConfig{
				Driver: "postgres",
			},
			wantErr: true,
		},
		{
			name: "unsupported driver",
			config: &DatabaseStorageConfig{
				Driver: "mongodb",
				DSN:    "mongodb://localhost:27017",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DatabaseStorageConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataProtectionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *DataProtectionConfig
		wantErr bool
	}{
		{
			name: "valid data protection",
			config: &DataProtectionConfig{
				Enabled: true,
				MaskingRules: &MaskingRulesConfig{
					EmailStrategy: "partial",
				},
			},
			wantErr: false,
		},
		{
			name: "disabled data protection",
			config: &DataProtectionConfig{
				Enabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DataProtectionConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaskingRulesConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *MaskingRulesConfig
		wantErr bool
	}{
		{
			name: "valid masking rules",
			config: &MaskingRulesConfig{
				EmailStrategy:      "partial",
				PhoneStrategy:      "last4",
				CreditCardStrategy: "last4",
			},
			wantErr: false,
		},
		{
			name: "invalid email strategy",
			config: &MaskingRulesConfig{
				EmailStrategy: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid phone strategy",
			config: &MaskingRulesConfig{
				PhoneStrategy: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid credit card strategy",
			config: &MaskingRulesConfig{
				CreditCardStrategy: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MaskingRulesConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config *SecurityConfig
		check  func(*testing.T, *SecurityConfig)
	}{
		{
			name: "set authentication defaults",
			config: &SecurityConfig{
				Authentication: &AuthenticationConfig{
					JWT: &JWTConfig{
						Secret: "this-is-a-very-secure-secret-key-32chars",
					},
				},
			},
			check: func(t *testing.T, c *SecurityConfig) {
				if c.Authentication.JWT.TokenExpiry != 1*time.Hour {
					t.Errorf("expected default token expiry 1h, got %v", c.Authentication.JWT.TokenExpiry)
				}
				if c.Authentication.Cache == nil {
					t.Error("expected cache to be initialized")
				}
			},
		},
		{
			name: "set encryption defaults",
			config: &SecurityConfig{
				Encryption: &EncryptionConfig{
					Enabled: true,
				},
			},
			check: func(t *testing.T, c *SecurityConfig) {
				if c.Encryption.Algorithm != "aes-gcm" {
					t.Errorf("expected default algorithm aes-gcm, got %s", c.Encryption.Algorithm)
				}
				if c.Encryption.KeySize != 32 {
					t.Errorf("expected default key size 32, got %d", c.Encryption.KeySize)
				}
			},
		},
		{
			name: "set audit defaults",
			config: &SecurityConfig{
				Audit: &AuditConfig{
					Enabled: true,
				},
			},
			check: func(t *testing.T, c *SecurityConfig) {
				if c.Audit.Storage == nil {
					t.Error("expected storage to be initialized")
				}
				if len(c.Audit.Levels) == 0 {
					t.Error("expected default levels to be set")
				}
				if len(c.Audit.Categories) == 0 {
					t.Error("expected default categories to be set")
				}
			},
		},
		{
			name: "set data protection defaults",
			config: &SecurityConfig{
				DataProtection: &DataProtectionConfig{
					Enabled: true,
				},
			},
			check: func(t *testing.T, c *SecurityConfig) {
				if c.DataProtection.MaskingRules == nil {
					t.Error("expected masking rules to be initialized")
				}
				if len(c.DataProtection.SensitiveFields) == 0 {
					t.Error("expected default sensitive fields to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.SetDefaults()
			tt.check(t, tt.config)
		})
	}
}

func TestCacheConfig_SetDefaults(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
	}

	config.SetDefaults()

	if config.TTL != 5*time.Minute {
		t.Errorf("expected default TTL 5m, got %v", config.TTL)
	}

	if config.MaxSize != 1000 {
		t.Errorf("expected default MaxSize 1000, got %d", config.MaxSize)
	}
}

func TestACLConfig_SetDefaults(t *testing.T) {
	config := &ACLConfig{}

	config.SetDefaults()

	if config.DefaultEffect != "deny" {
		t.Errorf("expected default effect 'deny', got %s", config.DefaultEffect)
	}
}

func TestFileStorageConfig_SetDefaults(t *testing.T) {
	config := &FileStorageConfig{
		Path: "/var/log/audit.log",
	}

	config.SetDefaults()

	if config.MaxFileSize != 100*1024*1024 {
		t.Errorf("expected default MaxFileSize 100MB, got %d", config.MaxFileSize)
	}

	if config.MaxBackups != 10 {
		t.Errorf("expected default MaxBackups 10, got %d", config.MaxBackups)
	}
}

func TestDatabaseStorageConfig_SetDefaults(t *testing.T) {
	config := &DatabaseStorageConfig{
		Driver: "postgres",
		DSN:    "postgres://localhost/db",
	}

	config.SetDefaults()

	if config.TableName != "audit_logs" {
		t.Errorf("expected default TableName 'audit_logs', got %s", config.TableName)
	}
}

func TestMaskingRulesConfig_SetDefaults(t *testing.T) {
	config := &MaskingRulesConfig{}

	config.SetDefaults()

	if config.DefaultMaskChar != "*" {
		t.Errorf("expected default mask char '*', got %s", config.DefaultMaskChar)
	}

	if config.EmailStrategy != "partial" {
		t.Errorf("expected default email strategy 'partial', got %s", config.EmailStrategy)
	}

	if config.PhoneStrategy != "last4" {
		t.Errorf("expected default phone strategy 'last4', got %s", config.PhoneStrategy)
	}

	if config.CreditCardStrategy != "last4" {
		t.Errorf("expected default credit card strategy 'last4', got %s", config.CreditCardStrategy)
	}
}

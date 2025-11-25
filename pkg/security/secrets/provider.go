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

package secrets

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrSecretNotFound is returned when a secret is not found.
	ErrSecretNotFound = errors.New("secret not found")

	// ErrProviderNotInitialized is returned when the provider is not initialized.
	ErrProviderNotInitialized = errors.New("provider not initialized")

	// ErrInvalidSecretKey is returned when the secret key is invalid or empty.
	ErrInvalidSecretKey = errors.New("invalid secret key")

	// ErrOperationNotSupported is returned when an operation is not supported by the provider.
	ErrOperationNotSupported = errors.New("operation not supported")
)

// Secret represents a secret with metadata.
type Secret struct {
	// Key is the unique identifier of the secret.
	Key string

	// Value is the secret value.
	Value string

	// Metadata contains additional information about the secret.
	Metadata map[string]string

	// CreatedAt is the time when the secret was created.
	CreatedAt time.Time

	// UpdatedAt is the time when the secret was last updated.
	UpdatedAt time.Time

	// ExpiresAt is the time when the secret expires (optional).
	ExpiresAt *time.Time

	// Version is the version of the secret (for versioned backends like Vault KV v2).
	Version int
}

// IsExpired checks if the secret has expired.
func (s *Secret) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// SecretsProvider defines the interface for secret management providers.
// Implementations can provide secrets from various sources such as environment
// variables, files, HashiCorp Vault, AWS Secrets Manager, etc.
type SecretsProvider interface {
	// GetSecret retrieves a secret by key.
	// Returns ErrSecretNotFound if the secret does not exist.
	GetSecret(ctx context.Context, key string) (*Secret, error)

	// GetSecretValue retrieves only the value of a secret by key.
	// This is a convenience method that returns only the secret value without metadata.
	// Returns ErrSecretNotFound if the secret does not exist.
	GetSecretValue(ctx context.Context, key string) (string, error)

	// SetSecret stores or updates a secret.
	// Returns ErrOperationNotSupported if the provider does not support write operations.
	SetSecret(ctx context.Context, key string, value string) error

	// SetSecretWithMetadata stores or updates a secret with metadata.
	// Returns ErrOperationNotSupported if the provider does not support write operations.
	SetSecretWithMetadata(ctx context.Context, secret *Secret) error

	// DeleteSecret removes a secret by key.
	// Returns ErrOperationNotSupported if the provider does not support delete operations.
	DeleteSecret(ctx context.Context, key string) error

	// ListSecrets returns a list of all secret keys.
	// Some providers may not support this operation and will return ErrOperationNotSupported.
	ListSecrets(ctx context.Context, prefix string) ([]string, error)

	// Close closes the provider and releases any resources.
	Close() error

	// Name returns the name of the provider (e.g., "env", "file", "vault").
	Name() string

	// IsReadOnly returns true if the provider only supports read operations.
	IsReadOnly() bool
}

// ProviderType represents the type of secrets provider.
type ProviderType string

const (
	// ProviderTypeEnv represents environment variable provider.
	ProviderTypeEnv ProviderType = "env"

	// ProviderTypeFile represents file-based provider.
	ProviderTypeFile ProviderType = "file"

	// ProviderTypeVault represents HashiCorp Vault provider.
	ProviderTypeVault ProviderType = "vault"

	// ProviderTypeMemory represents in-memory provider (for testing).
	ProviderTypeMemory ProviderType = "memory"
)

// ProviderConfig holds common configuration for all providers.
type ProviderConfig struct {
	// Type specifies the provider type.
	Type ProviderType `json:"type" yaml:"type" mapstructure:"type"`

	// Enabled indicates whether the provider is enabled.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// Priority determines the order in which providers are queried (lower values = higher priority).
	Priority int `json:"priority,omitempty" yaml:"priority,omitempty" mapstructure:"priority"`

	// Env contains configuration for environment variable provider.
	Env *EnvProviderConfig `json:"env,omitempty" yaml:"env,omitempty" mapstructure:"env"`

	// File contains configuration for file-based provider.
	File *FileProviderConfig `json:"file,omitempty" yaml:"file,omitempty" mapstructure:"file"`

	// Vault contains configuration for Vault provider.
	Vault *VaultProviderConfig `json:"vault,omitempty" yaml:"vault,omitempty" mapstructure:"vault"`
}

// Validate validates the provider configuration.
func (c *ProviderConfig) Validate() error {
	if c.Type == "" {
		return errors.New("provider type is required")
	}

	switch c.Type {
	case ProviderTypeEnv:
		if c.Env != nil {
			return c.Env.Validate()
		}
	case ProviderTypeFile:
		if c.File == nil {
			return errors.New("file provider configuration is required")
		}
		return c.File.Validate()
	case ProviderTypeVault:
		if c.Vault == nil {
			return errors.New("vault provider configuration is required")
		}
		return c.Vault.Validate()
	case ProviderTypeMemory:
		// Memory provider doesn't require configuration
		return nil
	default:
		return errors.New("unsupported provider type: " + string(c.Type))
	}

	return nil
}

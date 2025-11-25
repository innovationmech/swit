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
	"os"
	"strings"
	"time"
)

// EnvProviderConfig holds configuration for the environment variable provider.
type EnvProviderConfig struct {
	// Prefix is an optional prefix for environment variable names.
	// For example, if prefix is "APP_", the provider will look for "APP_<key>".
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty" mapstructure:"prefix"`

	// AllowList is an optional list of allowed secret keys.
	// If specified, only keys in this list can be retrieved.
	AllowList []string `json:"allow_list,omitempty" yaml:"allow_list,omitempty" mapstructure:"allow_list"`

	// DenyList is an optional list of denied secret keys.
	// Keys in this list cannot be retrieved even if they exist.
	DenyList []string `json:"deny_list,omitempty" yaml:"deny_list,omitempty" mapstructure:"deny_list"`
}

// Validate validates the environment provider configuration.
func (c *EnvProviderConfig) Validate() error {
	// No validation errors for env provider
	return nil
}

// EnvProvider provides secrets from environment variables.
type EnvProvider struct {
	config *EnvProviderConfig
}

// NewEnvProvider creates a new environment variable provider.
func NewEnvProvider(config *EnvProviderConfig) (*EnvProvider, error) {
	if config == nil {
		config = &EnvProviderConfig{}
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &EnvProvider{
		config: config,
	}, nil
}

// GetSecret retrieves a secret from environment variables.
func (p *EnvProvider) GetSecret(ctx context.Context, key string) (*Secret, error) {
	if key == "" {
		return nil, ErrInvalidSecretKey
	}

	// Check if key is allowed
	if !p.isKeyAllowed(key) {
		return nil, ErrSecretNotFound
	}

	// Build the environment variable name
	envKey := p.buildEnvKey(key)

	// Get the value from environment
	value, exists := os.LookupEnv(envKey)
	if !exists {
		return nil, ErrSecretNotFound
	}

	return &Secret{
		Key:       key,
		Value:     value,
		Metadata:  map[string]string{"source": "environment"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// GetSecretValue retrieves only the value of a secret.
func (p *EnvProvider) GetSecretValue(ctx context.Context, key string) (string, error) {
	secret, err := p.GetSecret(ctx, key)
	if err != nil {
		return "", err
	}
	return secret.Value, nil
}

// SetSecret is not supported for environment provider.
func (p *EnvProvider) SetSecret(ctx context.Context, key string, value string) error {
	return ErrOperationNotSupported
}

// SetSecretWithMetadata is not supported for environment provider.
func (p *EnvProvider) SetSecretWithMetadata(ctx context.Context, secret *Secret) error {
	return ErrOperationNotSupported
}

// DeleteSecret is not supported for environment provider.
func (p *EnvProvider) DeleteSecret(ctx context.Context, key string) error {
	return ErrOperationNotSupported
}

// ListSecrets returns a list of all environment variable keys matching the prefix.
func (p *EnvProvider) ListSecrets(ctx context.Context, prefix string) ([]string, error) {
	var keys []string

	// Get all environment variables
	environ := os.Environ()

	// Build the search prefix
	searchPrefix := p.buildEnvKey(prefix)

	for _, env := range environ {
		// Split into key=value
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		envKey := parts[0]

		// Check if it matches our prefix
		if !strings.HasPrefix(envKey, searchPrefix) {
			continue
		}

		// Extract the original key by removing the provider prefix
		originalKey := p.extractOriginalKey(envKey)

		// Check if key is allowed
		if !p.isKeyAllowed(originalKey) {
			continue
		}

		keys = append(keys, originalKey)
	}

	return keys, nil
}

// Close releases resources (no-op for environment provider).
func (p *EnvProvider) Close() error {
	return nil
}

// Name returns the provider name.
func (p *EnvProvider) Name() string {
	return string(ProviderTypeEnv)
}

// IsReadOnly returns true as environment provider is read-only.
func (p *EnvProvider) IsReadOnly() bool {
	return true
}

// buildEnvKey builds the full environment variable key with prefix.
func (p *EnvProvider) buildEnvKey(key string) string {
	if p.config.Prefix == "" {
		return key
	}
	return p.config.Prefix + key
}

// extractOriginalKey extracts the original key by removing the provider prefix.
func (p *EnvProvider) extractOriginalKey(envKey string) string {
	if p.config.Prefix == "" {
		return envKey
	}
	return strings.TrimPrefix(envKey, p.config.Prefix)
}

// isKeyAllowed checks if a key is allowed to be accessed.
func (p *EnvProvider) isKeyAllowed(key string) bool {
	// Check deny list first
	if len(p.config.DenyList) > 0 {
		for _, deniedKey := range p.config.DenyList {
			if key == deniedKey {
				return false
			}
		}
	}

	// If allow list is empty, all keys are allowed (except denied ones)
	if len(p.config.AllowList) == 0 {
		return true
	}

	// Check if key is in allow list
	for _, allowedKey := range p.config.AllowList {
		if key == allowedKey {
			return true
		}
	}

	return false
}

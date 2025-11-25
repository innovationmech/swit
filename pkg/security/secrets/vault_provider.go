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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
)

// VaultProviderConfig holds configuration for the HashiCorp Vault provider.
type VaultProviderConfig struct {
	// Address is the Vault server address (e.g., "https://vault.example.com:8200").
	Address string `json:"address" yaml:"address" mapstructure:"address"`

	// Token is the Vault authentication token.
	Token string `json:"token,omitempty" yaml:"token,omitempty" mapstructure:"token"`

	// TokenFile is the path to a file containing the Vault token.
	// Takes precedence over Token if both are specified.
	TokenFile string `json:"token_file,omitempty" yaml:"token_file,omitempty" mapstructure:"token_file"`

	// Path is the base path in Vault where secrets are stored (e.g., "secret/data/myapp").
	Path string `json:"path" yaml:"path" mapstructure:"path"`

	// KVVersion specifies the KV secrets engine version (1 or 2).
	// Defaults to version 2.
	KVVersion int `json:"kv_version,omitempty" yaml:"kv_version,omitempty" mapstructure:"kv_version"`

	// Namespace is the Vault namespace (Vault Enterprise feature).
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty" mapstructure:"namespace"`

	// MaxRetries is the maximum number of retries for API calls.
	// Defaults to 3.
	MaxRetries int `json:"max_retries,omitempty" yaml:"max_retries,omitempty" mapstructure:"max_retries"`

	// Timeout is the timeout for API calls.
	// Defaults to 60 seconds.
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`

	// TLS configuration for Vault connection.
	TLS *VaultTLSConfig `json:"tls,omitempty" yaml:"tls,omitempty" mapstructure:"tls"`
}

// VaultTLSConfig holds TLS configuration for Vault.
type VaultTLSConfig struct {
	// Enabled indicates whether TLS is enabled.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// CACert is the path to the CA certificate file.
	CACert string `json:"ca_cert,omitempty" yaml:"ca_cert,omitempty" mapstructure:"ca_cert"`

	// ClientCert is the path to the client certificate file.
	ClientCert string `json:"client_cert,omitempty" yaml:"client_cert,omitempty" mapstructure:"client_cert"`

	// ClientKey is the path to the client key file.
	ClientKey string `json:"client_key,omitempty" yaml:"client_key,omitempty" mapstructure:"client_key"`

	// InsecureSkipVerify controls whether to skip certificate verification.
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty" yaml:"insecure_skip_verify,omitempty" mapstructure:"insecure_skip_verify"`
}

// Validate validates the Vault provider configuration.
func (c *VaultProviderConfig) Validate() error {
	if c.Address == "" {
		return errors.New("vault address is required")
	}

	if c.Token == "" && c.TokenFile == "" {
		return errors.New("vault token or token file is required")
	}

	if c.Path == "" {
		return errors.New("vault path is required")
	}

	if c.KVVersion != 0 && c.KVVersion != 1 && c.KVVersion != 2 {
		return fmt.Errorf("invalid KV version: %d (supported: 1, 2)", c.KVVersion)
	}

	return nil
}

// VaultProvider provides secrets from HashiCorp Vault.
type VaultProvider struct {
	config *VaultProviderConfig
	client *api.Client
}

// NewVaultProvider creates a new Vault provider.
func NewVaultProvider(config *VaultProviderConfig) (*VaultProvider, error) {
	if config == nil {
		return nil, errors.New("vault provider configuration is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Set defaults
	if config.KVVersion == 0 {
		config.KVVersion = 2
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	// Create Vault client configuration
	vaultConfig := api.DefaultConfig()
	vaultConfig.Address = config.Address
	vaultConfig.MaxRetries = config.MaxRetries
	vaultConfig.Timeout = config.Timeout

	// Configure TLS
	if config.TLS != nil && config.TLS.Enabled {
		tlsConfig := &api.TLSConfig{
			CACert:     config.TLS.CACert,
			ClientCert: config.TLS.ClientCert,
			ClientKey:  config.TLS.ClientKey,
			Insecure:   config.TLS.InsecureSkipVerify,
		}
		if err := vaultConfig.ConfigureTLS(tlsConfig); err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
	}

	// Create Vault client
	client, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// Set namespace if specified
	if config.Namespace != "" {
		client.SetNamespace(config.Namespace)
	}

	// Set token
	token := config.Token
	if config.TokenFile != "" {
		// Read token from file
		tokenBytes, err := readTokenFromFile(config.TokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read token file: %w", err)
		}
		token = strings.TrimSpace(string(tokenBytes))
	}
	client.SetToken(token)

	return &VaultProvider{
		config: config,
		client: client,
	}, nil
}

// GetSecret retrieves a secret from Vault.
func (p *VaultProvider) GetSecret(ctx context.Context, key string) (*Secret, error) {
	if key == "" {
		return nil, ErrInvalidSecretKey
	}

	// Build the full path
	path := p.buildPath(key)

	// Read secret from Vault
	vaultSecret, err := p.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret from vault: %w", err)
	}

	if vaultSecret == nil {
		return nil, ErrSecretNotFound
	}

	// Extract data based on KV version
	var data map[string]interface{}
	var metadata map[string]interface{}
	var version int

	if p.config.KVVersion == 2 {
		// KV v2 stores data under "data" key
		dataRaw, ok := vaultSecret.Data["data"]
		if !ok {
			return nil, fmt.Errorf("no data field in vault response")
		}
		data, ok = dataRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid data format in vault response")
		}

		// Extract metadata
		metadataRaw, ok := vaultSecret.Data["metadata"]
		if ok {
			metadata, _ = metadataRaw.(map[string]interface{})
		}

		// Extract version
		if versionRaw, ok := metadata["version"]; ok {
			if versionInt, ok := versionRaw.(int); ok {
				version = versionInt
			}
		}
	} else {
		// KV v1 stores data directly
		data = vaultSecret.Data
	}

	// Extract the value (assuming it's stored under "value" key)
	valueRaw, ok := data["value"]
	if !ok {
		return nil, fmt.Errorf("no value field in secret data")
	}

	value, ok := valueRaw.(string)
	if !ok {
		return nil, fmt.Errorf("value is not a string")
	}

	// Build secret metadata
	secretMetadata := make(map[string]string)
	secretMetadata["source"] = "vault"
	secretMetadata["path"] = path
	if p.config.KVVersion == 2 && metadata != nil {
		if createdTime, ok := metadata["created_time"].(string); ok {
			secretMetadata["created_time"] = createdTime
		}
		if updatedTime, ok := metadata["updated_time"].(string); ok {
			secretMetadata["updated_time"] = updatedTime
		}
	}

	secret := &Secret{
		Key:       key,
		Value:     value,
		Metadata:  secretMetadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   version,
	}

	return secret, nil
}

// GetSecretValue retrieves only the value of a secret.
func (p *VaultProvider) GetSecretValue(ctx context.Context, key string) (string, error) {
	secret, err := p.GetSecret(ctx, key)
	if err != nil {
		return "", err
	}
	return secret.Value, nil
}

// SetSecret stores or updates a secret in Vault.
func (p *VaultProvider) SetSecret(ctx context.Context, key string, value string) error {
	if key == "" {
		return ErrInvalidSecretKey
	}

	// Build the full path
	path := p.buildPath(key)

	// Prepare data based on KV version
	var data map[string]interface{}
	if p.config.KVVersion == 2 {
		// KV v2 requires data under "data" key
		data = map[string]interface{}{
			"data": map[string]interface{}{
				"value": value,
			},
		}
	} else {
		// KV v1 stores data directly
		data = map[string]interface{}{
			"value": value,
		}
	}

	// Write secret to Vault
	_, err := p.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return fmt.Errorf("failed to write secret to vault: %w", err)
	}

	return nil
}

// SetSecretWithMetadata stores or updates a secret with metadata.
func (p *VaultProvider) SetSecretWithMetadata(ctx context.Context, secret *Secret) error {
	if secret == nil || secret.Key == "" {
		return ErrInvalidSecretKey
	}

	// Build the full path
	path := p.buildPath(secret.Key)

	// Prepare data with all fields
	secretData := map[string]interface{}{
		"value": secret.Value,
	}

	// Add custom metadata fields
	for k, v := range secret.Metadata {
		if k != "source" && k != "path" && k != "created_time" && k != "updated_time" {
			secretData[k] = v
		}
	}

	// Prepare data based on KV version
	var data map[string]interface{}
	if p.config.KVVersion == 2 {
		data = map[string]interface{}{
			"data": secretData,
		}
	} else {
		data = secretData
	}

	// Write secret to Vault
	_, err := p.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return fmt.Errorf("failed to write secret to vault: %w", err)
	}

	return nil
}

// DeleteSecret removes a secret from Vault.
func (p *VaultProvider) DeleteSecret(ctx context.Context, key string) error {
	if key == "" {
		return ErrInvalidSecretKey
	}

	// Build the full path
	path := p.buildPath(key)

	// For KV v2, we need to use the delete endpoint
	if p.config.KVVersion == 2 {
		// Convert "secret/data/..." to "secret/metadata/..."
		metadataPath := strings.Replace(path, "/data/", "/metadata/", 1)
		path = metadataPath
	}

	// Delete secret from Vault
	_, err := p.client.Logical().DeleteWithContext(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to delete secret from vault: %w", err)
	}

	return nil
}

// ListSecrets returns a list of secret keys from Vault.
func (p *VaultProvider) ListSecrets(ctx context.Context, prefix string) ([]string, error) {
	// Build the list path
	listPath := p.buildListPath(prefix)

	// List secrets from Vault
	vaultSecret, err := p.client.Logical().ListWithContext(ctx, listPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets from vault: %w", err)
	}

	if vaultSecret == nil || vaultSecret.Data == nil {
		return []string{}, nil
	}

	// Extract keys
	keysRaw, ok := vaultSecret.Data["keys"]
	if !ok {
		return []string{}, nil
	}

	keysSlice, ok := keysRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid keys format in vault response")
	}

	keys := make([]string, 0, len(keysSlice))
	for _, keyRaw := range keysSlice {
		if key, ok := keyRaw.(string); ok {
			// Remove trailing slash (directories)
			key = strings.TrimSuffix(key, "/")
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Close releases resources.
func (p *VaultProvider) Close() error {
	// Vault client doesn't require explicit cleanup
	return nil
}

// Name returns the provider name.
func (p *VaultProvider) Name() string {
	return string(ProviderTypeVault)
}

// IsReadOnly returns false as Vault supports write operations.
func (p *VaultProvider) IsReadOnly() bool {
	return false
}

// buildPath builds the full Vault path for a secret key.
func (p *VaultProvider) buildPath(key string) string {
	// Remove leading slash from key
	key = strings.TrimPrefix(key, "/")

	// Combine base path with key
	path := strings.TrimSuffix(p.config.Path, "/") + "/" + key

	return path
}

// buildListPath builds the Vault path for listing secrets.
func (p *VaultProvider) buildListPath(prefix string) string {
	// For KV v2, convert "secret/data/..." to "secret/metadata/..."
	basePath := p.config.Path
	if p.config.KVVersion == 2 {
		basePath = strings.Replace(basePath, "/data/", "/metadata/", 1)
	}

	// Remove leading slash from prefix
	prefix = strings.TrimPrefix(prefix, "/")

	if prefix == "" {
		return basePath
	}

	// Combine base path with prefix
	return strings.TrimSuffix(basePath, "/") + "/" + prefix
}

// readTokenFromFile reads a token from a file.
func readTokenFromFile(path string) ([]byte, error) {
	// This function is implemented in a separate helper to make testing easier
	return os.ReadFile(path) // nolint:gosec
}

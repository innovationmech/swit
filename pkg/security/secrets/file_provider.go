// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// FileFormat represents the format of the secrets file.
type FileFormat string

const (
	// FileFormatJSON represents JSON format.
	FileFormatJSON FileFormat = "json"

	// FileFormatYAML represents YAML format.
	FileFormatYAML FileFormat = "yaml"
)

// FileProviderConfig holds configuration for the file-based provider.
type FileProviderConfig struct {
	// Path is the path to the secrets file.
	Path string `json:"path" yaml:"path" mapstructure:"path"`

	// Format specifies the file format (json or yaml).
	// Defaults to JSON.
	Format FileFormat `json:"format,omitempty" yaml:"format,omitempty" mapstructure:"format"`

	// ReadOnly indicates if the file should be read-only.
	// If true, write operations will fail.
	ReadOnly bool `json:"read_only,omitempty" yaml:"read_only,omitempty" mapstructure:"read_only"`

	// AutoReload enables automatic reloading of the file when it changes.
	AutoReload bool `json:"auto_reload,omitempty" yaml:"auto_reload,omitempty" mapstructure:"auto_reload"`

	// ReloadInterval specifies how often to check for file changes (if AutoReload is true).
	// Defaults to 30 seconds.
	ReloadInterval time.Duration `json:"reload_interval,omitempty" yaml:"reload_interval,omitempty" mapstructure:"reload_interval"`
}

// Validate validates the file provider configuration.
func (c *FileProviderConfig) Validate() error {
	if c.Path == "" {
		return errors.New("file path is required")
	}

	if c.Format != "" && c.Format != FileFormatJSON && c.Format != FileFormatYAML {
		return fmt.Errorf("unsupported file format: %s (supported: json, yaml)", c.Format)
	}

	return nil
}

// FileProvider provides secrets from a file (JSON or YAML).
type FileProvider struct {
	config      *FileProviderConfig
	secrets     map[string]*Secret
	mu          sync.RWMutex
	lastModTime time.Time
	stopReload  chan struct{}
}

// NewFileProvider creates a new file-based provider.
func NewFileProvider(config *FileProviderConfig) (*FileProvider, error) {
	if config == nil {
		return nil, errors.New("file provider configuration is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Set defaults
	if config.Format == "" {
		config.Format = FileFormatJSON
	}
	if config.ReloadInterval == 0 {
		config.ReloadInterval = 30 * time.Second
	}

	p := &FileProvider{
		config:     config,
		secrets:    make(map[string]*Secret),
		stopReload: make(chan struct{}),
	}

	// Load initial secrets
	if err := p.reload(); err != nil {
		return nil, fmt.Errorf("failed to load secrets file: %w", err)
	}

	// Start auto-reload if enabled
	if config.AutoReload {
		go p.autoReloadLoop()
	}

	return p, nil
}

// GetSecret retrieves a secret from the file.
func (p *FileProvider) GetSecret(ctx context.Context, key string) (*Secret, error) {
	if key == "" {
		return nil, ErrInvalidSecretKey
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	secret, exists := p.secrets[key]
	if !exists {
		return nil, ErrSecretNotFound
	}

	// Check if secret has expired
	if secret.IsExpired() {
		return nil, ErrSecretNotFound
	}

	// Return a copy to prevent external modifications
	return p.copySecret(secret), nil
}

// GetSecretValue retrieves only the value of a secret.
func (p *FileProvider) GetSecretValue(ctx context.Context, key string) (string, error) {
	secret, err := p.GetSecret(ctx, key)
	if err != nil {
		return "", err
	}
	return secret.Value, nil
}

// SetSecret stores or updates a secret in the file.
func (p *FileProvider) SetSecret(ctx context.Context, key string, value string) error {
	if p.config.ReadOnly {
		return ErrOperationNotSupported
	}

	if key == "" {
		return ErrInvalidSecretKey
	}

	secret := &Secret{
		Key:       key,
		Value:     value,
		Metadata:  map[string]string{"source": "file"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return p.SetSecretWithMetadata(ctx, secret)
}

// SetSecretWithMetadata stores or updates a secret with metadata.
func (p *FileProvider) SetSecretWithMetadata(ctx context.Context, secret *Secret) error {
	if p.config.ReadOnly {
		return ErrOperationNotSupported
	}

	if secret == nil || secret.Key == "" {
		return ErrInvalidSecretKey
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Update timestamp
	secret.UpdatedAt = time.Now()
	if secret.CreatedAt.IsZero() {
		secret.CreatedAt = time.Now()
	}

	// Store secret
	p.secrets[secret.Key] = p.copySecret(secret)

	// Write to file
	return p.writeSecretsToFile()
}

// DeleteSecret removes a secret from the file.
func (p *FileProvider) DeleteSecret(ctx context.Context, key string) error {
	if p.config.ReadOnly {
		return ErrOperationNotSupported
	}

	if key == "" {
		return ErrInvalidSecretKey
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.secrets[key]; !exists {
		return ErrSecretNotFound
	}

	delete(p.secrets, key)

	// Write to file
	return p.writeSecretsToFile()
}

// ListSecrets returns a list of all secret keys.
func (p *FileProvider) ListSecrets(ctx context.Context, prefix string) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var keys []string
	for key, secret := range p.secrets {
		// Skip expired secrets
		if secret.IsExpired() {
			continue
		}

		// Filter by prefix if specified
		if prefix != "" && len(key) >= len(prefix) && key[:len(prefix)] != prefix {
			continue
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// Close stops auto-reload and releases resources.
func (p *FileProvider) Close() error {
	if p.config.AutoReload {
		close(p.stopReload)
	}
	return nil
}

// Name returns the provider name.
func (p *FileProvider) Name() string {
	return string(ProviderTypeFile)
}

// IsReadOnly returns the read-only status.
func (p *FileProvider) IsReadOnly() bool {
	return p.config.ReadOnly
}

// reload loads secrets from the file.
func (p *FileProvider) reload() error {
	// Get file info
	fileInfo, err := os.Stat(p.config.Path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, create it if not read-only
			if !p.config.ReadOnly {
				return p.writeSecretsToFile()
			}
			return fmt.Errorf("secrets file not found: %s", p.config.Path)
		}
		return fmt.Errorf("failed to stat secrets file: %w", err)
	}

	// Check if file has been modified
	if !p.lastModTime.IsZero() && !fileInfo.ModTime().After(p.lastModTime) {
		return nil // No changes
	}

	// Read file
	data, err := os.ReadFile(p.config.Path)
	if err != nil {
		return fmt.Errorf("failed to read secrets file: %w", err)
	}

	// Parse based on format
	var secretsMap map[string]*Secret
	switch p.config.Format {
	case FileFormatJSON:
		if err := json.Unmarshal(data, &secretsMap); err != nil {
			return fmt.Errorf("failed to parse JSON secrets file: %w", err)
		}
	case FileFormatYAML:
		if err := yaml.Unmarshal(data, &secretsMap); err != nil {
			return fmt.Errorf("failed to parse YAML secrets file: %w", err)
		}
	default:
		return fmt.Errorf("unsupported file format: %s", p.config.Format)
	}

	// Update secrets
	p.mu.Lock()
	p.secrets = secretsMap
	p.lastModTime = fileInfo.ModTime()
	p.mu.Unlock()

	return nil
}

// writeSecretsToFile writes secrets to the file.
// Must be called with lock held.
func (p *FileProvider) writeSecretsToFile() error {
	// Ensure directory exists
	dir := filepath.Dir(p.config.Path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create secrets directory: %w", err)
	}

	var data []byte
	var err error

	// Serialize based on format
	switch p.config.Format {
	case FileFormatJSON:
		data, err = json.MarshalIndent(p.secrets, "", "  ")
	case FileFormatYAML:
		data, err = yaml.Marshal(p.secrets)
	default:
		return fmt.Errorf("unsupported file format: %s", p.config.Format)
	}

	if err != nil {
		return fmt.Errorf("failed to serialize secrets: %w", err)
	}

	// Write to file
	if err := os.WriteFile(p.config.Path, data, 0600); err != nil {
		return fmt.Errorf("failed to write secrets file: %w", err)
	}

	// Update last modified time
	fileInfo, _ := os.Stat(p.config.Path)
	if fileInfo != nil {
		p.lastModTime = fileInfo.ModTime()
	}

	return nil
}

// autoReloadLoop periodically checks for file changes.
func (p *FileProvider) autoReloadLoop() {
	ticker := time.NewTicker(p.config.ReloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = p.reload() // Ignore errors in background reload
		case <-p.stopReload:
			return
		}
	}
}

// copySecret creates a deep copy of a secret.
func (p *FileProvider) copySecret(s *Secret) *Secret {
	if s == nil {
		return nil
	}

	// Copy metadata
	metadata := make(map[string]string, len(s.Metadata))
	for k, v := range s.Metadata {
		metadata[k] = v
	}

	copy := &Secret{
		Key:       s.Key,
		Value:     s.Value,
		Metadata:  metadata,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		Version:   s.Version,
	}

	if s.ExpiresAt != nil {
		expiresAt := *s.ExpiresAt
		copy.ExpiresAt = &expiresAt
	}

	return copy
}

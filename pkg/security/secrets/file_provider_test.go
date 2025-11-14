// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secrets

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileProvider_GetSecret(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	secretsFile := filepath.Join(tmpDir, "secrets.json")

	// Create provider
	provider, err := NewFileProvider(&FileProviderConfig{
		Path:     secretsFile,
		Format:   FileFormatJSON,
		ReadOnly: false,
	})
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// Set a secret first
	err = provider.SetSecret(ctx, "test-key", "test-value")
	if err != nil {
		t.Fatalf("SetSecret() error = %v", err)
	}

	// Get the secret
	secret, err := provider.GetSecret(ctx, "test-key")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	if secret.Value != "test-value" {
		t.Errorf("GetSecret() value = %v, want %v", secret.Value, "test-value")
	}

	// Try to get non-existent secret
	_, err = provider.GetSecret(ctx, "non-existent")
	if err != ErrSecretNotFound {
		t.Errorf("GetSecret() error = %v, want %v", err, ErrSecretNotFound)
	}
}

func TestFileProvider_SetSecret(t *testing.T) {
	tmpDir := t.TempDir()
	secretsFile := filepath.Join(tmpDir, "secrets.json")

	provider, err := NewFileProvider(&FileProviderConfig{
		Path:     secretsFile,
		Format:   FileFormatJSON,
		ReadOnly: false,
	})
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// Set multiple secrets
	tests := []struct {
		key   string
		value string
	}{
		{"db_password", "secret123"},
		{"api_key", "key456"},
	}

	for _, tt := range tests {
		err := provider.SetSecret(ctx, tt.key, tt.value)
		if err != nil {
			t.Errorf("SetSecret(%s) error = %v", tt.key, err)
		}
	}

	// Verify secrets are persisted
	for _, tt := range tests {
		secret, err := provider.GetSecret(ctx, tt.key)
		if err != nil {
			t.Errorf("GetSecret(%s) error = %v", tt.key, err)
			continue
		}
		if secret.Value != tt.value {
			t.Errorf("GetSecret(%s) value = %v, want %v", tt.key, secret.Value, tt.value)
		}
	}
}

func TestFileProvider_DeleteSecret(t *testing.T) {
	tmpDir := t.TempDir()
	secretsFile := filepath.Join(tmpDir, "secrets.json")

	provider, err := NewFileProvider(&FileProviderConfig{
		Path:     secretsFile,
		Format:   FileFormatJSON,
		ReadOnly: false,
	})
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// Set a secret
	err = provider.SetSecret(ctx, "temp-key", "temp-value")
	if err != nil {
		t.Fatalf("SetSecret() error = %v", err)
	}

	// Delete the secret
	err = provider.DeleteSecret(ctx, "temp-key")
	if err != nil {
		t.Fatalf("DeleteSecret() error = %v", err)
	}

	// Verify it's deleted
	_, err = provider.GetSecret(ctx, "temp-key")
	if err != ErrSecretNotFound {
		t.Errorf("GetSecret() error = %v, want %v", err, ErrSecretNotFound)
	}
}

func TestFileProvider_ListSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	secretsFile := filepath.Join(tmpDir, "secrets.json")

	provider, err := NewFileProvider(&FileProviderConfig{
		Path:     secretsFile,
		Format:   FileFormatJSON,
		ReadOnly: false,
	})
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// Set multiple secrets
	secrets := map[string]string{
		"app_secret1": "value1",
		"app_secret2": "value2",
		"db_password": "value3",
	}

	for key, value := range secrets {
		err := provider.SetSecret(ctx, key, value)
		if err != nil {
			t.Fatalf("SetSecret() error = %v", err)
		}
	}

	// List all secrets
	keys, err := provider.ListSecrets(ctx, "")
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}

	if len(keys) != len(secrets) {
		t.Errorf("ListSecrets() returned %d keys, want %d", len(keys), len(secrets))
	}

	// List secrets with prefix
	keys, err = provider.ListSecrets(ctx, "app_")
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("ListSecrets(app_) returned %d keys, want 2", len(keys))
	}
}

func TestFileProvider_ReadOnly(t *testing.T) {
	tmpDir := t.TempDir()
	secretsFile := filepath.Join(tmpDir, "secrets.json")

	// Create an initial secrets file
	initialProvider, err := NewFileProvider(&FileProviderConfig{
		Path:     secretsFile,
		Format:   FileFormatJSON,
		ReadOnly: false,
	})
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}

	ctx := context.Background()
	err = initialProvider.SetSecret(ctx, "key1", "value1")
	if err != nil {
		t.Fatalf("SetSecret() error = %v", err)
	}
	initialProvider.Close()

	// Create read-only provider
	provider, err := NewFileProvider(&FileProviderConfig{
		Path:     secretsFile,
		Format:   FileFormatJSON,
		ReadOnly: true,
	})
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}
	defer provider.Close()

	// Verify read works
	secret, err := provider.GetSecret(ctx, "key1")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}
	if secret.Value != "value1" {
		t.Errorf("GetSecret() value = %v, want %v", secret.Value, "value1")
	}

	// Verify write fails
	err = provider.SetSecret(ctx, "key2", "value2")
	if err != ErrOperationNotSupported {
		t.Errorf("SetSecret() error = %v, want %v", err, ErrOperationNotSupported)
	}
}

func TestFileProvider_YAMLFormat(t *testing.T) {
	tmpDir := t.TempDir()
	secretsFile := filepath.Join(tmpDir, "secrets.yaml")

	provider, err := NewFileProvider(&FileProviderConfig{
		Path:     secretsFile,
		Format:   FileFormatYAML,
		ReadOnly: false,
	})
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// Set a secret
	err = provider.SetSecret(ctx, "yaml-key", "yaml-value")
	if err != nil {
		t.Fatalf("SetSecret() error = %v", err)
	}

	// Get the secret
	secret, err := provider.GetSecret(ctx, "yaml-key")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	if secret.Value != "yaml-value" {
		t.Errorf("GetSecret() value = %v, want %v", secret.Value, "yaml-value")
	}

	// Verify file exists and is YAML
	data, err := os.ReadFile(secretsFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// YAML files should contain the key in a readable format
	if len(data) == 0 {
		t.Error("YAML file is empty")
	}
}

func TestFileProvider_SecretExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	secretsFile := filepath.Join(tmpDir, "secrets.json")

	provider, err := NewFileProvider(&FileProviderConfig{
		Path:     secretsFile,
		Format:   FileFormatJSON,
		ReadOnly: false,
	})
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	// Set a secret with expiration
	expiresAt := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago
	secret := &Secret{
		Key:       "expired-key",
		Value:     "expired-value",
		ExpiresAt: &expiresAt,
	}

	err = provider.SetSecretWithMetadata(ctx, secret)
	if err != nil {
		t.Fatalf("SetSecretWithMetadata() error = %v", err)
	}

	// Try to get expired secret
	_, err = provider.GetSecret(ctx, "expired-key")
	if err != ErrSecretNotFound {
		t.Errorf("GetSecret() error = %v, want %v", err, ErrSecretNotFound)
	}
}

func TestFileProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *FileProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &FileProviderConfig{
				Path:   "/tmp/secrets.json",
				Format: FileFormatJSON,
			},
			wantErr: false,
		},
		{
			name: "missing path",
			config: &FileProviderConfig{
				Format: FileFormatJSON,
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			config: &FileProviderConfig{
				Path:   "/tmp/secrets.json",
				Format: "xml",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

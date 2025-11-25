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
	"testing"
	"time"
)

func TestManager_GetSecret(t *testing.T) {
	// Create manager with memory provider
	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
		},
		Cache: &CacheConfig{
			Enabled: true,
			TTL:     1 * time.Minute,
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Set a secret
	err = manager.SetSecret(ctx, "test-key", "test-value")
	if err != nil {
		t.Fatalf("SetSecret() error = %v", err)
	}

	// Get the secret
	secret, err := manager.GetSecret(ctx, "test-key")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	if secret.Value != "test-value" {
		t.Errorf("GetSecret() value = %v, want %v", secret.Value, "test-value")
	}

	// Try to get non-existent secret
	_, err = manager.GetSecret(ctx, "non-existent")
	if err != ErrSecretNotFound {
		t.Errorf("GetSecret() error = %v, want %v", err, ErrSecretNotFound)
	}
}

func TestManager_CachingBehavior(t *testing.T) {
	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
		},
		Cache: &CacheConfig{
			Enabled: true,
			TTL:     100 * time.Millisecond,
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Set a secret
	err = manager.SetSecret(ctx, "cached-key", "cached-value")
	if err != nil {
		t.Fatalf("SetSecret() error = %v", err)
	}

	// First get - should cache
	_, err = manager.GetSecret(ctx, "cached-key")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	// Verify cache hit
	cachedSecret := manager.getFromCache("cached-key")
	if cachedSecret == nil {
		t.Error("Secret not found in cache")
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Cache should be expired
	cachedSecret = manager.getFromCache("cached-key")
	if cachedSecret != nil {
		t.Error("Expired secret still in cache")
	}
}

func TestManager_MultipleProviders(t *testing.T) {
	tmpDir := t.TempDir()

	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:     ProviderTypeMemory,
				Enabled:  true,
				Priority: 1,
			},
			{
				Type:     ProviderTypeFile,
				Enabled:  true,
				Priority: 2,
				File: &FileProviderConfig{
					Path:     tmpDir + "/secrets.json",
					Format:   FileFormatJSON,
					ReadOnly: false,
				},
			},
		},
		Cache: &CacheConfig{
			Enabled: false,
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Set secret in memory provider
	err = manager.SetSecret(ctx, "memory-key", "memory-value")
	if err != nil {
		t.Fatalf("SetSecret() error = %v", err)
	}

	// Get the secret - should come from memory provider
	secret, err := manager.GetSecret(ctx, "memory-key")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	if secret.Value != "memory-value" {
		t.Errorf("GetSecret() value = %v, want %v", secret.Value, "memory-value")
	}
}

func TestManager_SetSecret(t *testing.T) {
	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	tests := []struct {
		key   string
		value string
	}{
		{"key1", "value1"},
		{"key2", "value2"},
		{"key3", "value3"},
	}

	for _, tt := range tests {
		err := manager.SetSecret(ctx, tt.key, tt.value)
		if err != nil {
			t.Errorf("SetSecret(%s) error = %v", tt.key, err)
		}

		// Verify secret was set
		secret, err := manager.GetSecret(ctx, tt.key)
		if err != nil {
			t.Errorf("GetSecret(%s) error = %v", tt.key, err)
			continue
		}

		if secret.Value != tt.value {
			t.Errorf("GetSecret(%s) value = %v, want %v", tt.key, secret.Value, tt.value)
		}
	}
}

func TestManager_DeleteSecret(t *testing.T) {
	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Set a secret
	err = manager.SetSecret(ctx, "delete-key", "delete-value")
	if err != nil {
		t.Fatalf("SetSecret() error = %v", err)
	}

	// Delete the secret
	err = manager.DeleteSecret(ctx, "delete-key")
	if err != nil {
		t.Fatalf("DeleteSecret() error = %v", err)
	}

	// Verify it's deleted
	_, err = manager.GetSecret(ctx, "delete-key")
	if err != ErrSecretNotFound {
		t.Errorf("GetSecret() error = %v, want %v", err, ErrSecretNotFound)
	}
}

func TestManager_DeleteSecret_NotFound(t *testing.T) {
	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Try to delete a non-existent secret
	err = manager.DeleteSecret(ctx, "non-existent-key")
	if err != ErrSecretNotFound {
		t.Errorf("DeleteSecret() error = %v, want %v (secret doesn't exist)", err, ErrSecretNotFound)
	}
}

func TestManager_DeleteSecret_ReadOnlyProvider(t *testing.T) {
	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeEnv,
				Enabled: true,
				Env:     &EnvProviderConfig{},
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Try to delete from read-only provider
	err = manager.DeleteSecret(ctx, "some-key")
	if err != ErrOperationNotSupported {
		t.Errorf("DeleteSecret() error = %v, want %v (no writable providers)", err, ErrOperationNotSupported)
	}
}

func TestManager_DeleteSecret_MixedProviders(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file provider with an existing secret
	fileProvider, err := NewFileProvider(&FileProviderConfig{
		Path:     tmpDir + "/secrets.json",
		Format:   FileFormatJSON,
		ReadOnly: false,
	})
	if err != nil {
		t.Fatalf("NewFileProvider() error = %v", err)
	}

	ctx := context.Background()
	err = fileProvider.SetSecret(ctx, "file-key", "file-value")
	if err != nil {
		t.Fatalf("SetSecret() error = %v", err)
	}
	fileProvider.Close()

	// Create manager with memory provider (empty) and file provider
	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
			{
				Type:    ProviderTypeFile,
				Enabled: true,
				File: &FileProviderConfig{
					Path:     tmpDir + "/secrets.json",
					Format:   FileFormatJSON,
					ReadOnly: false,
				},
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	// Delete a key that exists in file provider but not in memory provider
	// Memory provider will return ErrSecretNotFound, but file provider will succeed
	err = manager.DeleteSecret(ctx, "file-key")
	if err != nil {
		t.Errorf("DeleteSecret() error = %v, want nil (should succeed when at least one provider deletes)", err)
	}

	// Verify it's deleted from both providers
	_, err = manager.GetSecret(ctx, "file-key")
	if err != ErrSecretNotFound {
		t.Errorf("GetSecret() error = %v, want %v", err, ErrSecretNotFound)
	}
}

func TestManager_DeleteSecret_AllProvidersNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
			{
				Type:    ProviderTypeFile,
				Enabled: true,
				File: &FileProviderConfig{
					Path:     tmpDir + "/secrets.json",
					Format:   FileFormatJSON,
					ReadOnly: false,
				},
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Try to delete a key that doesn't exist in any provider
	err = manager.DeleteSecret(ctx, "non-existent")
	if err != ErrSecretNotFound {
		t.Errorf("DeleteSecret() error = %v, want %v (all providers report not found)", err, ErrSecretNotFound)
	}
}

func TestManager_ListSecrets(t *testing.T) {
	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Set multiple secrets
	secrets := map[string]string{
		"app_key1": "value1",
		"app_key2": "value2",
		"db_pass":  "value3",
	}

	for key, value := range secrets {
		err := manager.SetSecret(ctx, key, value)
		if err != nil {
			t.Fatalf("SetSecret() error = %v", err)
		}
	}

	// List all secrets
	keys, err := manager.ListSecrets(ctx, "")
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}

	if len(keys) != len(secrets) {
		t.Errorf("ListSecrets() returned %d keys, want %d", len(keys), len(secrets))
	}

	// List secrets with prefix
	keys, err = manager.ListSecrets(ctx, "app_")
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("ListSecrets(app_) returned %d keys, want 2", len(keys))
	}
}

func TestManager_Refresh(t *testing.T) {
	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
		},
		Cache: &CacheConfig{
			Enabled: true,
			TTL:     1 * time.Minute,
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Set a secret
	err = manager.SetSecret(ctx, "refresh-key", "original-value")
	if err != nil {
		t.Fatalf("SetSecret() error = %v", err)
	}

	// Get secret to cache it
	_, err = manager.GetSecret(ctx, "refresh-key")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	// Update the secret in the provider
	memProvider := manager.providers[0].(*MemoryProvider)
	err = memProvider.SetSecret(ctx, "refresh-key", "updated-value")
	if err != nil {
		t.Fatalf("Provider SetSecret() error = %v", err)
	}

	// Refresh the secret
	err = manager.Refresh(ctx, "refresh-key")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	// Get the secret - should have updated value
	secret, err := manager.GetSecret(ctx, "refresh-key")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	if secret.Value != "updated-value" {
		t.Errorf("GetSecret() value = %v, want %v", secret.Value, "updated-value")
	}
}

func TestManager_CacheEviction(t *testing.T) {
	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
		},
		Cache: &CacheConfig{
			Enabled: true,
			TTL:     1 * time.Minute,
			MaxSize: 2,
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Set and get three secrets (cache size is 2)
	for i := 1; i <= 3; i++ {
		key := "key" + string(rune('0'+i))
		value := "value" + string(rune('0'+i))

		err := manager.SetSecret(ctx, key, value)
		if err != nil {
			t.Fatalf("SetSecret() error = %v", err)
		}

		_, err = manager.GetSecret(ctx, key)
		if err != nil {
			t.Fatalf("GetSecret() error = %v", err)
		}
	}

	// Cache should have at most 2 entries
	manager.cacheMu.RLock()
	cacheSize := len(manager.cache)
	manager.cacheMu.RUnlock()

	if cacheSize > 2 {
		t.Errorf("Cache size = %d, want <= 2", cacheSize)
	}
}

func TestManagerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ManagerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ManagerConfig{
				Providers: []*ProviderConfig{
					{
						Type:    ProviderTypeMemory,
						Enabled: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no providers",
			config: &ManagerConfig{
				Providers: []*ProviderConfig{},
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			config: &ManagerConfig{
				Providers: []*ProviderConfig{
					{
						Type:    ProviderTypeFile,
						Enabled: true,
						File:    nil, // Missing required file config
					},
				},
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

func TestManager_SecretLevelExpiration(t *testing.T) {
	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
		},
		Cache: &CacheConfig{
			Enabled: true,
			TTL:     5 * time.Minute, // Long cache TTL
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Create a secret with a short expiration (100ms)
	shortExpiry := time.Now().Add(100 * time.Millisecond)
	secretWithExpiry := &Secret{
		Key:       "short-lived-key",
		Value:     "short-lived-value",
		Metadata:  map[string]string{"source": "test"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: &shortExpiry,
	}

	// Set the secret directly in the memory provider
	memProvider := manager.providers[0].(*MemoryProvider)
	err = memProvider.SetSecretWithMetadata(ctx, secretWithExpiry)
	if err != nil {
		t.Fatalf("SetSecretWithMetadata() error = %v", err)
	}

	// First get should succeed and cache the secret
	secret, err := manager.GetSecret(ctx, "short-lived-key")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}
	if secret.Value != "short-lived-value" {
		t.Errorf("GetSecret() value = %v, want short-lived-value", secret.Value)
	}

	// Verify it's in cache
	cachedSecret := manager.getFromCache("short-lived-key")
	if cachedSecret == nil {
		t.Error("Secret not found in cache")
	}

	// Wait for secret to expire
	time.Sleep(150 * time.Millisecond)

	// Cache should now respect the secret's expiration and return nil
	cachedSecret = manager.getFromCache("short-lived-key")
	if cachedSecret != nil {
		t.Error("Expired secret still returned from cache")
	}

	// GetSecret should also fail since the secret is expired
	_, err = manager.GetSecret(ctx, "short-lived-key")
	if err != ErrSecretNotFound {
		t.Errorf("GetSecret() error = %v, want %v (expired secret should not be found)", err, ErrSecretNotFound)
	}
}

func TestManager_CacheRespectsSecretExpiry(t *testing.T) {
	config := &ManagerConfig{
		Providers: []*ProviderConfig{
			{
				Type:    ProviderTypeMemory,
				Enabled: true,
			},
		},
		Cache: &CacheConfig{
			Enabled: true,
			TTL:     10 * time.Minute, // Long cache TTL
		},
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Create a secret with expiration shorter than cache TTL
	shortExpiry := time.Now().Add(200 * time.Millisecond)
	secretWithExpiry := &Secret{
		Key:       "test-key",
		Value:     "test-value",
		Metadata:  map[string]string{"source": "test"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: &shortExpiry,
	}

	// Set the secret
	memProvider := manager.providers[0].(*MemoryProvider)
	err = memProvider.SetSecretWithMetadata(ctx, secretWithExpiry)
	if err != nil {
		t.Fatalf("SetSecretWithMetadata() error = %v", err)
	}

	// Get the secret to cache it
	_, err = manager.GetSecret(ctx, "test-key")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	// Check cache entry expiration time
	manager.cacheMu.RLock()
	entry := manager.cache["test-key"]
	manager.cacheMu.RUnlock()

	if entry == nil {
		t.Fatal("Cache entry not found")
	}

	// Cache entry should expire at secret's expiry time, not cache TTL
	expectedExpiry := shortExpiry
	if entry.expiresAt.After(expectedExpiry.Add(10 * time.Millisecond)) {
		t.Errorf("Cache entry expiresAt = %v, should be around %v (secret expiry, not cache TTL)",
			entry.expiresAt, expectedExpiry)
	}
}

func TestMemoryProvider(t *testing.T) {
	provider := NewMemoryProvider()
	ctx := context.Background()

	// Test basic operations
	err := provider.SetSecret(ctx, "key1", "value1")
	if err != nil {
		t.Fatalf("SetSecret() error = %v", err)
	}

	secret, err := provider.GetSecret(ctx, "key1")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	if secret.Value != "value1" {
		t.Errorf("GetSecret() value = %v, want value1", secret.Value)
	}

	// Test delete
	err = provider.DeleteSecret(ctx, "key1")
	if err != nil {
		t.Fatalf("DeleteSecret() error = %v", err)
	}

	_, err = provider.GetSecret(ctx, "key1")
	if err != ErrSecretNotFound {
		t.Errorf("GetSecret() error = %v, want %v", err, ErrSecretNotFound)
	}

	// Test list
	_ = provider.SetSecret(ctx, "app_key1", "value1")
	_ = provider.SetSecret(ctx, "app_key2", "value2")
	_ = provider.SetSecret(ctx, "db_pass", "value3")

	keys, err := provider.ListSecrets(ctx, "app_")
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("ListSecrets() returned %d keys, want 2", len(keys))
	}

	// Test properties
	if provider.Name() != string(ProviderTypeMemory) {
		t.Errorf("Name() = %v, want %v", provider.Name(), ProviderTypeMemory)
	}

	if provider.IsReadOnly() {
		t.Error("IsReadOnly() = true, want false")
	}
}

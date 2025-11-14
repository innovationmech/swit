// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secrets

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ManagerConfig holds configuration for the secrets manager.
type ManagerConfig struct {
	// Providers is a list of provider configurations.
	Providers []*ProviderConfig `json:"providers" yaml:"providers" mapstructure:"providers"`

	// Cache configuration
	Cache *CacheConfig `json:"cache,omitempty" yaml:"cache,omitempty" mapstructure:"cache"`

	// RefreshConfig configures automatic secret refresh.
	Refresh *RefreshConfig `json:"refresh,omitempty" yaml:"refresh,omitempty" mapstructure:"refresh"`
}

// CacheConfig holds cache configuration.
type CacheConfig struct {
	// Enabled indicates whether caching is enabled.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// TTL is the default time-to-live for cached secrets.
	// Defaults to 5 minutes.
	TTL time.Duration `json:"ttl,omitempty" yaml:"ttl,omitempty" mapstructure:"ttl"`

	// MaxSize is the maximum number of secrets to cache.
	// 0 means unlimited.
	MaxSize int `json:"max_size,omitempty" yaml:"max_size,omitempty" mapstructure:"max_size"`
}

// RefreshConfig holds automatic refresh configuration.
type RefreshConfig struct {
	// Enabled indicates whether automatic refresh is enabled.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// Interval is the interval for checking and refreshing secrets.
	// Defaults to 10 minutes.
	Interval time.Duration `json:"interval,omitempty" yaml:"interval,omitempty" mapstructure:"interval"`

	// Keys is a list of secret keys to automatically refresh.
	Keys []string `json:"keys,omitempty" yaml:"keys,omitempty" mapstructure:"keys"`
}

// Validate validates the manager configuration.
func (c *ManagerConfig) Validate() error {
	if len(c.Providers) == 0 {
		return errors.New("at least one provider is required")
	}

	for i, provider := range c.Providers {
		if err := provider.Validate(); err != nil {
			return fmt.Errorf("provider %d validation failed: %w", i, err)
		}
	}

	return nil
}

// cacheEntry represents a cached secret.
type cacheEntry struct {
	secret    *Secret
	expiresAt time.Time
}

// isExpired checks if the cache entry has expired.
func (e *cacheEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// Manager manages secrets from multiple providers with caching and refresh.
type Manager struct {
	config      *ManagerConfig
	providers   []SecretsProvider
	cache       map[string]*cacheEntry
	cacheMu     sync.RWMutex
	stopRefresh chan struct{}
	wg          sync.WaitGroup
}

// NewManager creates a new secrets manager.
func NewManager(config *ManagerConfig) (*Manager, error) {
	if config == nil {
		return nil, errors.New("manager configuration is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Set defaults
	if config.Cache == nil {
		config.Cache = &CacheConfig{Enabled: true, TTL: 5 * time.Minute}
	}
	if config.Cache.TTL == 0 {
		config.Cache.TTL = 5 * time.Minute
	}

	if config.Refresh == nil {
		config.Refresh = &RefreshConfig{Enabled: false}
	}
	if config.Refresh.Interval == 0 {
		config.Refresh.Interval = 10 * time.Minute
	}

	m := &Manager{
		config:      config,
		providers:   make([]SecretsProvider, 0, len(config.Providers)),
		cache:       make(map[string]*cacheEntry),
		stopRefresh: make(chan struct{}),
	}

	// Initialize providers
	for _, providerConfig := range config.Providers {
		if !providerConfig.Enabled {
			continue
		}

		provider, err := m.createProvider(providerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %s: %w", providerConfig.Type, err)
		}

		m.providers = append(m.providers, provider)
	}

	if len(m.providers) == 0 {
		return nil, errors.New("no enabled providers configured")
	}

	// Start refresh goroutine if enabled
	if config.Refresh.Enabled {
		m.wg.Add(1)
		go m.refreshLoop()
	}

	return m, nil
}

// GetSecret retrieves a secret by key, checking cache first then providers.
func (m *Manager) GetSecret(ctx context.Context, key string) (*Secret, error) {
	if key == "" {
		return nil, ErrInvalidSecretKey
	}

	// Check cache first
	if m.config.Cache.Enabled {
		if secret := m.getFromCache(key); secret != nil {
			return secret, nil
		}
	}

	// Try each provider in order
	var lastErr error
	for _, provider := range m.providers {
		secret, err := provider.GetSecret(ctx, key)
		if err == nil {
			// Cache the secret
			if m.config.Cache.Enabled {
				m.putInCache(key, secret)
			}
			return secret, nil
		}

		if err != ErrSecretNotFound {
			lastErr = err
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, ErrSecretNotFound
}

// GetSecretValue retrieves only the value of a secret.
func (m *Manager) GetSecretValue(ctx context.Context, key string) (string, error) {
	secret, err := m.GetSecret(ctx, key)
	if err != nil {
		return "", err
	}
	return secret.Value, nil
}

// SetSecret stores or updates a secret in the first writable provider.
func (m *Manager) SetSecret(ctx context.Context, key string, value string) error {
	if key == "" {
		return ErrInvalidSecretKey
	}

	// Find first writable provider
	for _, provider := range m.providers {
		if provider.IsReadOnly() {
			continue
		}

		err := provider.SetSecret(ctx, key, value)
		if err == nil {
			// Invalidate cache
			m.invalidateCache(key)
			return nil
		}

		if err != ErrOperationNotSupported {
			return err
		}
	}

	return ErrOperationNotSupported
}

// SetSecretWithMetadata stores or updates a secret with metadata.
func (m *Manager) SetSecretWithMetadata(ctx context.Context, secret *Secret) error {
	if secret == nil || secret.Key == "" {
		return ErrInvalidSecretKey
	}

	// Find first writable provider
	for _, provider := range m.providers {
		if provider.IsReadOnly() {
			continue
		}

		err := provider.SetSecretWithMetadata(ctx, secret)
		if err == nil {
			// Invalidate cache
			m.invalidateCache(secret.Key)
			return nil
		}

		if err != ErrOperationNotSupported {
			return err
		}
	}

	return ErrOperationNotSupported
}

// DeleteSecret removes a secret from all providers.
func (m *Manager) DeleteSecret(ctx context.Context, key string) error {
	if key == "" {
		return ErrInvalidSecretKey
	}

	// Try to delete from all writable providers
	var lastErr error
	deleted := false
	hasWritableProvider := false
	allNotFound := true

	for _, provider := range m.providers {
		if provider.IsReadOnly() {
			continue
		}

		hasWritableProvider = true

		err := provider.DeleteSecret(ctx, key)
		if err == nil {
			deleted = true
			allNotFound = false
		} else if err == ErrSecretNotFound {
			// Provider supports deletion but key doesn't exist
			// Continue to check other providers
		} else if err == ErrOperationNotSupported {
			// Provider doesn't support deletion
			// Continue to check other providers
		} else {
			// Real error (e.g., network error, permission denied)
			lastErr = err
			allNotFound = false
		}
	}

	// Invalidate cache
	m.invalidateCache(key)

	// If we successfully deleted from at least one provider, succeed
	if deleted {
		return nil
	}

	// If we encountered a real error, return it
	if lastErr != nil {
		return lastErr
	}

	// If no writable providers exist, operation is not supported
	if !hasWritableProvider {
		return ErrOperationNotSupported
	}

	// If all writable providers reported ErrSecretNotFound, the secret doesn't exist
	if allNotFound {
		return ErrSecretNotFound
	}

	// All writable providers returned ErrOperationNotSupported
	return ErrOperationNotSupported
}

// ListSecrets returns a combined list of secret keys from all providers.
func (m *Manager) ListSecrets(ctx context.Context, prefix string) ([]string, error) {
	keysMap := make(map[string]bool)

	// Collect keys from all providers
	for _, provider := range m.providers {
		keys, err := provider.ListSecrets(ctx, prefix)
		if err != nil && err != ErrOperationNotSupported {
			return nil, err
		}

		for _, key := range keys {
			keysMap[key] = true
		}
	}

	// Convert map to slice
	keys := make([]string, 0, len(keysMap))
	for key := range keysMap {
		keys = append(keys, key)
	}

	return keys, nil
}

// Refresh manually refreshes specific secrets in the cache.
func (m *Manager) Refresh(ctx context.Context, keys ...string) error {
	if !m.config.Cache.Enabled {
		return nil
	}

	for _, key := range keys {
		// Invalidate cache
		m.invalidateCache(key)

		// Pre-load into cache
		_, _ = m.GetSecret(ctx, key)
	}

	return nil
}

// Close closes all providers and stops background tasks.
func (m *Manager) Close() error {
	// Stop refresh loop
	if m.config.Refresh.Enabled {
		close(m.stopRefresh)
		m.wg.Wait()
	}

	// Close all providers
	var errs []error
	for _, provider := range m.providers {
		if err := provider.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close providers: %v", errs)
	}

	return nil
}

// getFromCache retrieves a secret from cache.
func (m *Manager) getFromCache(key string) *Secret {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()

	entry, exists := m.cache[key]
	if !exists || entry.isExpired() {
		return nil
	}

	// Also check if the secret itself has expired
	if entry.secret.IsExpired() {
		return nil
	}

	return entry.secret
}

// putInCache stores a secret in cache.
func (m *Manager) putInCache(key string, secret *Secret) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	// Check max size
	if m.config.Cache.MaxSize > 0 && len(m.cache) >= m.config.Cache.MaxSize {
		// Simple eviction: remove expired entries first
		m.evictExpired()

		// If still at capacity, remove oldest entry
		if len(m.cache) >= m.config.Cache.MaxSize {
			m.evictOldest()
		}
	}

	// Calculate cache expiration time respecting secret-level expiration
	cacheExpiresAt := time.Now().Add(m.config.Cache.TTL)

	// If the secret has its own expiration, use the minimum of cache TTL and secret expiry
	if secret.ExpiresAt != nil && secret.ExpiresAt.Before(cacheExpiresAt) {
		cacheExpiresAt = *secret.ExpiresAt
	}

	m.cache[key] = &cacheEntry{
		secret:    secret,
		expiresAt: cacheExpiresAt,
	}
}

// invalidateCache removes a secret from cache.
func (m *Manager) invalidateCache(key string) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	delete(m.cache, key)
}

// evictExpired removes all expired entries from cache.
// Must be called with lock held.
func (m *Manager) evictExpired() {
	for key, entry := range m.cache {
		if entry.isExpired() {
			delete(m.cache, key)
		}
	}
}

// evictOldest removes the oldest entry from cache.
// Must be called with lock held.
func (m *Manager) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range m.cache {
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		delete(m.cache, oldestKey)
	}
}

// refreshLoop periodically refreshes secrets.
func (m *Manager) refreshLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.Refresh.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.refreshSecrets()
		case <-m.stopRefresh:
			return
		}
	}
}

// refreshSecrets refreshes all configured secrets.
func (m *Manager) refreshSecrets() {
	ctx := context.Background()

	for _, key := range m.config.Refresh.Keys {
		_ = m.Refresh(ctx, key)
	}

	// Also refresh any cached secrets that are about to expire
	m.cacheMu.RLock()
	keysToRefresh := make([]string, 0)
	threshold := time.Now().Add(m.config.Cache.TTL / 2)

	for key, entry := range m.cache {
		if entry.expiresAt.Before(threshold) {
			keysToRefresh = append(keysToRefresh, key)
		}
	}
	m.cacheMu.RUnlock()

	for _, key := range keysToRefresh {
		_ = m.Refresh(ctx, key)
	}
}

// createProvider creates a provider from configuration.
func (m *Manager) createProvider(config *ProviderConfig) (SecretsProvider, error) {
	switch config.Type {
	case ProviderTypeEnv:
		return NewEnvProvider(config.Env)
	case ProviderTypeFile:
		return NewFileProvider(config.File)
	case ProviderTypeVault:
		return NewVaultProvider(config.Vault)
	case ProviderTypeMemory:
		return NewMemoryProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

// MemoryProvider is a simple in-memory provider for testing.
type MemoryProvider struct {
	secrets map[string]*Secret
	mu      sync.RWMutex
}

// NewMemoryProvider creates a new in-memory provider.
func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{
		secrets: make(map[string]*Secret),
	}
}

// GetSecret retrieves a secret.
func (p *MemoryProvider) GetSecret(ctx context.Context, key string) (*Secret, error) {
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

	return secret, nil
}

// GetSecretValue retrieves only the value.
func (p *MemoryProvider) GetSecretValue(ctx context.Context, key string) (string, error) {
	secret, err := p.GetSecret(ctx, key)
	if err != nil {
		return "", err
	}
	return secret.Value, nil
}

// SetSecret stores a secret.
func (p *MemoryProvider) SetSecret(ctx context.Context, key string, value string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.secrets[key] = &Secret{
		Key:       key,
		Value:     value,
		Metadata:  map[string]string{"source": "memory"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return nil
}

// SetSecretWithMetadata stores a secret with metadata.
func (p *MemoryProvider) SetSecretWithMetadata(ctx context.Context, secret *Secret) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.secrets[secret.Key] = secret
	return nil
}

// DeleteSecret removes a secret.
func (p *MemoryProvider) DeleteSecret(ctx context.Context, key string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.secrets[key]; !exists {
		return ErrSecretNotFound
	}

	delete(p.secrets, key)
	return nil
}

// ListSecrets returns all keys.
func (p *MemoryProvider) ListSecrets(ctx context.Context, prefix string) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	keys := make([]string, 0, len(p.secrets))
	for key := range p.secrets {
		if prefix == "" || len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Close releases resources.
func (p *MemoryProvider) Close() error {
	return nil
}

// Name returns the provider name.
func (p *MemoryProvider) Name() string {
	return string(ProviderTypeMemory)
}

// IsReadOnly returns false.
func (p *MemoryProvider) IsReadOnly() bool {
	return false
}

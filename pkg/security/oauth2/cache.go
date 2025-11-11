// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oauth2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// TokenCache provides an interface for caching OAuth2 tokens.
type TokenCache interface {
	// Get retrieves a token from the cache.
	Get(ctx context.Context, key string) (*oauth2.Token, error)

	// Set stores a token in the cache.
	Set(ctx context.Context, key string, token *oauth2.Token) error

	// Delete removes a token from the cache.
	Delete(ctx context.Context, key string) error

	// Clear removes all tokens from the cache.
	Clear(ctx context.Context) error
}

// MemoryTokenCache is an in-memory implementation of TokenCache.
type MemoryTokenCache struct {
	mu     sync.RWMutex
	tokens map[string]*cachedToken
	ttl    time.Duration
}

type cachedToken struct {
	token     *oauth2.Token
	expiresAt time.Time
}

// NewMemoryTokenCache creates a new in-memory token cache.
// ttl specifies how long tokens should be cached. If 0, tokens are cached indefinitely.
func NewMemoryTokenCache(ttl time.Duration) *MemoryTokenCache {
	cache := &MemoryTokenCache{
		tokens: make(map[string]*cachedToken),
		ttl:    ttl,
	}

	// Start cleanup goroutine if TTL is set
	if ttl > 0 {
		go cache.cleanupLoop()
	}

	return cache
}

// Get retrieves a token from the cache.
func (c *MemoryTokenCache) Get(ctx context.Context, key string) (*oauth2.Token, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.tokens[key]
	if !ok {
		return nil, fmt.Errorf("token not found in cache")
	}

	// Check if cache entry has expired
	if !cached.expiresAt.IsZero() && time.Now().After(cached.expiresAt) {
		return nil, fmt.Errorf("cached token has expired")
	}

	// Check if the token itself has expired
	if IsTokenExpired(cached.token) {
		return nil, fmt.Errorf("token has expired")
	}

	return cached.token, nil
}

// Set stores a token in the cache.
func (c *MemoryTokenCache) Set(ctx context.Context, key string, token *oauth2.Token) error {
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	cached := &cachedToken{
		token: token,
	}

	// Set cache expiration time
	if c.ttl > 0 {
		cached.expiresAt = time.Now().Add(c.ttl)
	}

	c.tokens[key] = cached
	return nil
}

// Delete removes a token from the cache.
func (c *MemoryTokenCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.tokens, key)
	return nil
}

// Clear removes all tokens from the cache.
func (c *MemoryTokenCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokens = make(map[string]*cachedToken)
	return nil
}

// cleanupLoop periodically removes expired tokens from the cache.
func (c *MemoryTokenCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired tokens from the cache.
func (c *MemoryTokenCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, cached := range c.tokens {
		// Remove if cache entry has expired
		if !cached.expiresAt.IsZero() && now.After(cached.expiresAt) {
			delete(c.tokens, key)
			continue
		}

		// Remove if token has expired
		if IsTokenExpired(cached.token) {
			delete(c.tokens, key)
		}
	}
}

// Size returns the number of tokens in the cache.
func (c *MemoryTokenCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.tokens)
}

// CachingTokenSource wraps an oauth2.TokenSource with caching.
type CachingTokenSource struct {
	source oauth2.TokenSource
	cache  TokenCache
	key    string
	mu     sync.Mutex
}

// NewCachingTokenSource creates a new caching token source.
func NewCachingTokenSource(source oauth2.TokenSource, cache TokenCache, key string) *CachingTokenSource {
	return &CachingTokenSource{
		source: source,
		cache:  cache,
		key:    key,
	}
}

// Token returns a token from the cache if available and valid,
// otherwise fetches a new token from the underlying source.
func (s *CachingTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	// Try to get token from cache
	token, err := s.cache.Get(ctx, s.key)
	if err == nil && !IsTokenExpiring(token, 5*time.Minute) {
		return token, nil
	}

	// Fetch new token from source
	token, err = s.source.Token()
	if err != nil {
		return nil, err
	}

	// Store in cache
	if err := s.cache.Set(ctx, s.key, token); err != nil {
		// Log error but don't fail
		// The token is still valid even if caching fails
	}

	return token, nil
}

// TokenCacheConfig represents the configuration for token caching.
type TokenCacheConfig struct {
	// Enabled indicates whether token caching is enabled.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// Type is the type of cache to use (e.g., "memory", "redis").
	Type string `json:"type" yaml:"type" mapstructure:"type"`

	// TTL is the time-to-live for cached tokens.
	TTL time.Duration `json:"ttl" yaml:"ttl" mapstructure:"ttl"`
}

// SetDefaults sets default values for the cache configuration.
func (c *TokenCacheConfig) SetDefaults() {
	if c.Type == "" {
		c.Type = "memory"
	}

	if c.TTL <= 0 {
		c.TTL = 1 * time.Hour
	}
}

// Validate validates the cache configuration.
func (c *TokenCacheConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Type != "memory" {
		return fmt.Errorf("only 'memory' cache type is currently supported")
	}

	if c.TTL <= 0 {
		return fmt.Errorf("ttl must be greater than 0")
	}

	return nil
}

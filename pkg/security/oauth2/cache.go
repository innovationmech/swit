// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oauth2

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/oauth2"
)

// TokenCache provides an interface for caching OAuth2 tokens.
type TokenCache interface {
	// Get retrieves a token from the cache.
	Get(ctx context.Context, key string) (*oauth2.Token, error)

	// Set stores a token in the cache with optional TTL.
	Set(ctx context.Context, key string, token *oauth2.Token, ttl ...time.Duration) error

	// Delete removes a token from the cache.
	Delete(ctx context.Context, key string) error

	// Clear removes all tokens from the cache.
	Clear(ctx context.Context) error

	// Stats returns cache statistics.
	Stats() CacheStats

	// Warmup preloads tokens into the cache.
	Warmup(ctx context.Context, tokens map[string]*oauth2.Token) error
}

// CacheStats holds cache statistics.
type CacheStats struct {
	// Size is the current number of tokens in the cache.
	Size int

	// MaxSize is the maximum number of tokens the cache can hold.
	MaxSize int

	// Hits is the number of cache hits.
	Hits uint64

	// Misses is the number of cache misses.
	Misses uint64

	// Evictions is the number of evicted entries.
	Evictions uint64

	// HitRate is the cache hit rate (0.0 - 1.0).
	HitRate float64
}

// MemoryTokenCache is an in-memory LRU cache implementation of TokenCache.
type MemoryTokenCache struct {
	mu         sync.RWMutex
	tokens     map[string]*cacheEntry
	lruList    *list.List
	maxSize    int
	defaultTTL time.Duration

	// Statistics (atomic counters for thread safety)
	hits      atomic.Uint64
	misses    atomic.Uint64
	evictions atomic.Uint64

	// Cleanup
	stopCleanup chan struct{}
	cleanupWg   sync.WaitGroup
}

type cacheEntry struct {
	key        string
	token      *oauth2.Token
	expiresAt  time.Time
	lruElement *list.Element
}

// LRU list element value
type lruEntry struct {
	key string
}

// NewMemoryTokenCache creates a new in-memory token cache with default max size (1000).
// ttl specifies how long tokens should be cached. If 0, tokens are cached indefinitely.
func NewMemoryTokenCache(ttl time.Duration) *MemoryTokenCache {
	return NewMemoryTokenCacheWithSize(ttl, 1000)
}

// NewMemoryTokenCacheWithSize creates a new in-memory token cache with specified max size.
// ttl specifies how long tokens should be cached. If 0, tokens are cached indefinitely.
// maxSize specifies the maximum number of tokens to cache. If 0, no limit is enforced.
func NewMemoryTokenCacheWithSize(ttl time.Duration, maxSize int) *MemoryTokenCache {
	cache := &MemoryTokenCache{
		tokens:      make(map[string]*cacheEntry),
		lruList:     list.New(),
		maxSize:     maxSize,
		defaultTTL:  ttl,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine if TTL is set
	if ttl > 0 {
		cache.cleanupWg.Add(1)
		go cache.cleanupLoop()
	}

	return cache
}

// Close stops the cleanup goroutine and releases resources.
func (c *MemoryTokenCache) Close() error {
	if c.defaultTTL > 0 {
		close(c.stopCleanup)
		c.cleanupWg.Wait()
	}
	return nil
}

// Get retrieves a token from the cache and updates LRU order.
func (c *MemoryTokenCache) Get(ctx context.Context, key string) (*oauth2.Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.tokens[key]
	if !ok {
		c.misses.Add(1)
		return nil, fmt.Errorf("token not found in cache")
	}

	// Check if cache entry has expired
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		// Remove expired entry
		c.removeEntryLocked(key)
		c.misses.Add(1)
		return nil, fmt.Errorf("cached token has expired")
	}

	// Check if the token itself has expired
	if IsTokenExpired(entry.token) {
		c.removeEntryLocked(key)
		c.misses.Add(1)
		return nil, fmt.Errorf("token has expired")
	}

	// Update LRU - move to front
	c.lruList.MoveToFront(entry.lruElement)
	c.hits.Add(1)

	return entry.token, nil
}

// Set stores a token in the cache with optional TTL override.
func (c *MemoryTokenCache) Set(ctx context.Context, key string, token *oauth2.Token, ttl ...time.Duration) error {
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Determine TTL (use override if provided, otherwise use default)
	effectiveTTL := c.defaultTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		effectiveTTL = ttl[0]
	}

	// Check if entry already exists
	if existing, ok := c.tokens[key]; ok {
		// Update existing entry
		existing.token = token
		if effectiveTTL > 0 {
			existing.expiresAt = time.Now().Add(effectiveTTL)
		} else {
			existing.expiresAt = time.Time{}
		}
		// Move to front of LRU
		c.lruList.MoveToFront(existing.lruElement)
		return nil
	}

	// Evict LRU entry if cache is full
	if c.maxSize > 0 && len(c.tokens) >= c.maxSize {
		c.evictLRULocked()
	}

	// Create new entry
	entry := &cacheEntry{
		key:   key,
		token: token,
	}

	if effectiveTTL > 0 {
		entry.expiresAt = time.Now().Add(effectiveTTL)
	}

	// Add to LRU list
	lruElem := c.lruList.PushFront(&lruEntry{key: key})
	entry.lruElement = lruElem

	c.tokens[key] = entry
	return nil
}

// Delete removes a token from the cache.
func (c *MemoryTokenCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.removeEntryLocked(key)
	return nil
}

// removeEntryLocked removes an entry from the cache without locking.
// Must be called with c.mu held.
func (c *MemoryTokenCache) removeEntryLocked(key string) {
	if entry, ok := c.tokens[key]; ok {
		// Remove from LRU list
		if entry.lruElement != nil {
			c.lruList.Remove(entry.lruElement)
		}
		delete(c.tokens, key)
	}
}

// Clear removes all tokens from the cache.
func (c *MemoryTokenCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokens = make(map[string]*cacheEntry)
	c.lruList.Init()
	return nil
}

// evictLRULocked removes the least recently used entry from the cache.
// Must be called with c.mu held.
func (c *MemoryTokenCache) evictLRULocked() {
	elem := c.lruList.Back()
	if elem == nil {
		return
	}

	lruEntry := elem.Value.(*lruEntry)
	c.removeEntryLocked(lruEntry.key)
	c.evictions.Add(1)
}

// Stats returns current cache statistics.
func (c *MemoryTokenCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return CacheStats{
		Size:      len(c.tokens),
		MaxSize:   c.maxSize,
		Hits:      hits,
		Misses:    misses,
		Evictions: c.evictions.Load(),
		HitRate:   hitRate,
	}
}

// Warmup preloads tokens into the cache.
func (c *MemoryTokenCache) Warmup(ctx context.Context, tokens map[string]*oauth2.Token) error {
	for key, token := range tokens {
		if err := c.Set(ctx, key, token); err != nil {
			return fmt.Errorf("failed to warmup token %s: %w", key, err)
		}
	}
	return nil
}

// cleanupLoop periodically removes expired tokens from the cache.
func (c *MemoryTokenCache) cleanupLoop() {
	defer c.cleanupWg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanup removes expired tokens from the cache.
func (c *MemoryTokenCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	keysToRemove := make([]string, 0)

	for key, entry := range c.tokens {
		// Check if cache entry has expired
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
			keysToRemove = append(keysToRemove, key)
			continue
		}

		// Check if token has expired
		if IsTokenExpired(entry.token) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	// Remove expired entries
	for _, key := range keysToRemove {
		c.removeEntryLocked(key)
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

	// Store in cache (no TTL override, use cache default)
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

	// MaxSize is the maximum number of tokens to cache (0 = unlimited).
	MaxSize int `json:"max_size" yaml:"max_size" mapstructure:"max_size"`
}

// SetDefaults sets default values for the cache configuration.
func (c *TokenCacheConfig) SetDefaults() {
	if c.Type == "" {
		c.Type = "memory"
	}

	if c.TTL <= 0 {
		c.TTL = 1 * time.Hour
	}

	if c.MaxSize <= 0 {
		c.MaxSize = 1000
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

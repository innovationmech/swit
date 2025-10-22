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
	"sync"
	"time"
)

// PermissionCacheEntry represents a cached permission set with expiration
type PermissionCacheEntry struct {
	Permissions *PermissionSet
	ExpiresAt   time.Time
}

// IsExpired checks if the cache entry has expired
func (e *PermissionCacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// PermissionCache provides caching for user permissions
type PermissionCache struct {
	// entries maps user IDs to cached permission sets
	entries map[string]*PermissionCacheEntry

	// ttl is the time-to-live for cache entries
	ttl time.Duration

	// maxSize is the maximum number of entries in the cache
	maxSize int

	// hits counts cache hits
	hits uint64

	// misses counts cache misses
	misses uint64

	// evictions counts cache evictions
	evictions uint64

	// mu protects concurrent access to the cache
	mu sync.RWMutex

	// cleanupTicker triggers periodic cleanup of expired entries
	cleanupTicker *time.Ticker

	// stopCleanup signals the cleanup goroutine to stop
	stopCleanup chan struct{}
}

// PermissionCacheConfig configures the permission cache
type PermissionCacheConfig struct {
	// TTL is the time-to-live for cache entries
	TTL time.Duration

	// MaxSize is the maximum number of entries in the cache
	MaxSize int

	// CleanupInterval is how often to clean up expired entries
	CleanupInterval time.Duration
}

// NewPermissionCache creates a new permission cache
func NewPermissionCache(config *PermissionCacheConfig) *PermissionCache {
	if config == nil {
		config = &PermissionCacheConfig{
			TTL:             5 * time.Minute,
			MaxSize:         1000,
			CleanupInterval: 1 * time.Minute,
		}
	}

	// Set default cleanup interval if not provided or invalid
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 1 * time.Minute
	}

	cache := &PermissionCache{
		entries:       make(map[string]*PermissionCacheEntry),
		ttl:           config.TTL,
		maxSize:       config.MaxSize,
		stopCleanup:   make(chan struct{}),
		cleanupTicker: time.NewTicker(config.CleanupInterval),
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Get retrieves a permission set from the cache
func (c *PermissionCache) Get(userID string) (*PermissionSet, bool) {
	c.mu.RLock()
	entry, exists := c.entries[userID]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	if entry.IsExpired() {
		c.mu.Lock()
		delete(c.entries, userID)
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	c.mu.Lock()
	c.hits++
	c.mu.Unlock()

	return entry.Permissions.Clone(), true
}

// Set stores a permission set in the cache
func (c *PermissionCache) Set(userID string, permissions *PermissionSet) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict entries
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[userID] = &PermissionCacheEntry{
		Permissions: permissions.Clone(),
		ExpiresAt:   time.Now().Add(c.ttl),
	}
}

// Delete removes a permission set from the cache
func (c *PermissionCache) Delete(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, userID)
}

// Clear removes all entries from the cache
func (c *PermissionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*PermissionCacheEntry)
}

// Size returns the number of entries in the cache
func (c *PermissionCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// Stats returns cache statistics
func (c *PermissionCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return map[string]interface{}{
		"enabled":   true,
		"size":      len(c.entries),
		"max_size":  c.maxSize,
		"hits":      c.hits,
		"misses":    c.misses,
		"hit_rate":  hitRate,
		"evictions": c.evictions,
		"ttl_ms":    c.ttl.Milliseconds(),
	}
}

// evictOldest removes the oldest entry from the cache
func (c *PermissionCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.ExpiresAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.evictions++
	}
}

// cleanupLoop periodically removes expired entries
func (c *PermissionCache) cleanupLoop() {
	for {
		select {
		case <-c.cleanupTicker.C:
			c.cleanupExpired()
		case <-c.stopCleanup:
			c.cleanupTicker.Stop()
			return
		}
	}
}

// cleanupExpired removes all expired entries
func (c *PermissionCache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
			c.evictions++
		}
	}
}

// Close stops the cache cleanup goroutine
func (c *PermissionCache) Close() {
	close(c.stopCleanup)
}

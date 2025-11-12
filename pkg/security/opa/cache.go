// Copyright (c) 2024 Six-Thirty Labs, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opa

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Cache 决策缓存接口
type Cache interface {
	// Get 获取缓存的决策
	Get(ctx context.Context, path string, input interface{}) (*Result, bool)

	// Set 设置缓存
	Set(ctx context.Context, path string, input interface{}, result *Result)

	// Clear 清除所有缓存
	Clear(ctx context.Context)

	// Delete 删除特定路径的缓存
	Delete(ctx context.Context, path string)

	// Close 关闭缓存
	Close(ctx context.Context) error

	// Stats 获取缓存统计信息
	Stats() CacheStats
}

// CacheStats 缓存统计信息
type CacheStats struct {
	// Hits 缓存命中次数
	Hits int64 `json:"hits"`

	// Misses 缓存未命中次数
	Misses int64 `json:"misses"`

	// Evictions 缓存驱逐次数
	Evictions int64 `json:"evictions"`

	// Size 当前缓存条目数
	Size int `json:"size"`

	// HitRate 缓存命中率
	HitRate float64 `json:"hit_rate"`
}

// cacheEntry 缓存条目
type cacheEntry struct {
	result    *Result
	expiresAt time.Time
}

// memoryCache 内存缓存实现
type memoryCache struct {
	config  *CacheConfig
	entries map[string]*cacheEntry
	mu      sync.RWMutex

	// 统计
	hits      int64
	misses    int64
	evictions int64
}

// NewCache 创建缓存
func NewCache(config *CacheConfig) (Cache, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	config.SetDefaults()

	return &memoryCache{
		config:  config,
		entries: make(map[string]*cacheEntry),
	}, nil
}

// Get 获取缓存的决策
func (c *memoryCache) Get(ctx context.Context, path string, input interface{}) (*Result, bool) {
	key, err := c.generateKey(path, input)
	if err != nil {
		return nil, false
	}

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	// 检查是否过期
	if c.config.TTL > 0 && time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	c.mu.Lock()
	c.hits++
	c.mu.Unlock()

	return entry.result, true
}

// Set 设置缓存
func (c *memoryCache) Set(ctx context.Context, path string, input interface{}, result *Result) {
	key, err := c.generateKey(path, input)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否需要驱逐
	if len(c.entries) >= c.config.MaxSize {
		// 简单的 LRU：删除第一个找到的过期条目
		for k, entry := range c.entries {
			if c.config.TTL > 0 && time.Now().After(entry.expiresAt) {
				delete(c.entries, k)
				c.evictions++
				break
			}
		}

		// 如果还是满的，删除第一个条目
		if len(c.entries) >= c.config.MaxSize {
			for k := range c.entries {
				delete(c.entries, k)
				c.evictions++
				break
			}
		}
	}

	entry := &cacheEntry{
		result: result,
	}

	if c.config.TTL > 0 {
		entry.expiresAt = time.Now().Add(c.config.TTL)
	}

	c.entries[key] = entry
}

// Clear 清除所有缓存
func (c *memoryCache) Clear(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
}

// Delete 删除特定路径的缓存
func (c *memoryCache) Delete(ctx context.Context, path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 删除所有以该路径开头的缓存键
	for key := range c.entries {
		if len(key) > len(path) && key[:len(path)] == path {
			delete(c.entries, key)
		}
	}
}

// Close 关闭缓存
func (c *memoryCache) Close(ctx context.Context) error {
	c.Clear(ctx)
	return nil
}

// Stats 获取缓存统计信息
func (c *memoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return CacheStats{
		Hits:      c.hits,
		Misses:    c.misses,
		Evictions: c.evictions,
		Size:      len(c.entries),
		HitRate:   hitRate,
	}
}

// generateKey 生成缓存键
func (c *memoryCache) generateKey(path string, input interface{}) (string, error) {
	// 将 input 序列化为 JSON
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal input: %w", err)
	}

	// 使用 SHA256 生成键
	hash := sha256.Sum256([]byte(path + string(inputJSON)))
	return fmt.Sprintf("%x", hash), nil
}

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
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		MaxSize: 100,
		TTL:     5 * time.Minute,
	}

	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	if cache == nil {
		t.Fatal("Expected cache to be created")
	}
}

func TestCacheGetSet(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		MaxSize: 100,
		TTL:     5 * time.Minute,
	}

	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	path := "test/allow"
	input := map[string]interface{}{
		"user":   "alice",
		"action": "read",
	}

	result := &Result{
		Decision: true,
		Allowed:  true,
	}

	// 测试 Set
	cache.Set(ctx, path, input, result)

	// 测试 Get
	cached, ok := cache.Get(ctx, path, input)
	if !ok {
		t.Fatal("Expected to find cached result")
	}

	if cached.Allowed != result.Allowed {
		t.Errorf("Expected cached result to match, got %v, want %v", cached.Allowed, result.Allowed)
	}
}

func TestCacheMiss(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		MaxSize: 100,
		TTL:     5 * time.Minute,
	}

	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	path := "test/allow"
	input := map[string]interface{}{
		"user":   "alice",
		"action": "read",
	}

	// 测试缓存未命中
	_, ok := cache.Get(ctx, path, input)
	if ok {
		t.Fatal("Expected cache miss")
	}
}

func TestCacheTTL(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		MaxSize: 100,
		TTL:     100 * time.Millisecond,
	}

	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	path := "test/allow"
	input := map[string]interface{}{
		"user":   "alice",
		"action": "read",
	}

	result := &Result{
		Decision: true,
		Allowed:  true,
	}

	cache.Set(ctx, path, input, result)

	// 立即获取应该成功
	_, ok := cache.Get(ctx, path, input)
	if !ok {
		t.Fatal("Expected to find cached result")
	}

	// 等待 TTL 过期
	time.Sleep(150 * time.Millisecond)

	// 应该缓存未命中
	_, ok = cache.Get(ctx, path, input)
	if ok {
		t.Fatal("Expected cache to expire")
	}
}

func TestCacheEviction(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		MaxSize: 2,
		TTL:     5 * time.Minute,
	}

	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()

	result := &Result{
		Decision: true,
		Allowed:  true,
	}

	// 添加两个条目
	cache.Set(ctx, "path1", map[string]interface{}{"key": "1"}, result)
	cache.Set(ctx, "path2", map[string]interface{}{"key": "2"}, result)

	stats := cache.Stats()
	if stats.Size != 2 {
		t.Errorf("Expected cache size 2, got %d", stats.Size)
	}

	// 添加第三个条目，应该触发驱逐
	cache.Set(ctx, "path3", map[string]interface{}{"key": "3"}, result)

	stats = cache.Stats()
	if stats.Size > 2 {
		t.Errorf("Expected cache size <= 2, got %d", stats.Size)
	}

	if stats.Evictions == 0 {
		t.Error("Expected at least one eviction")
	}
}

func TestCacheClear(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		MaxSize: 100,
		TTL:     5 * time.Minute,
	}

	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()

	result := &Result{
		Decision: true,
		Allowed:  true,
	}

	// 添加条目
	cache.Set(ctx, "path1", map[string]interface{}{"key": "1"}, result)
	cache.Set(ctx, "path2", map[string]interface{}{"key": "2"}, result)

	stats := cache.Stats()
	if stats.Size != 2 {
		t.Errorf("Expected cache size 2, got %d", stats.Size)
	}

	// 清除缓存
	cache.Clear(ctx)

	stats = cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", stats.Size)
	}
}

func TestCacheStats(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		MaxSize: 100,
		TTL:     5 * time.Minute,
	}

	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	path := "test/allow"
	input := map[string]interface{}{
		"user":   "alice",
		"action": "read",
	}

	result := &Result{
		Decision: true,
		Allowed:  true,
	}

	// 缓存未命中
	cache.Get(ctx, path, input)

	// 设置缓存
	cache.Set(ctx, path, input, result)

	// 缓存命中
	cache.Get(ctx, path, input)
	cache.Get(ctx, path, input)

	stats := cache.Stats()

	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	expectedHitRate := 2.0 / 3.0
	if stats.HitRate < expectedHitRate-0.01 || stats.HitRate > expectedHitRate+0.01 {
		t.Errorf("Expected hit rate ~%.2f, got %.2f", expectedHitRate, stats.HitRate)
	}
}


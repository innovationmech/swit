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

// BenchmarkCacheGet 测试缓存 Get 性能
func BenchmarkCacheGet(b *testing.B) {
	config := &CacheConfig{
		Enabled: true,
		MaxSize: 10000,
		TTL:     5 * time.Minute,
	}

	cache, err := NewCache(config)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
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

	// 预填充缓存
	cache.Set(ctx, path, input, result)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(ctx, path, input)
	}
}

// BenchmarkCacheSet 测试缓存 Set 性能
func BenchmarkCacheSet(b *testing.B) {
	config := &CacheConfig{
		Enabled: true,
		MaxSize: 10000,
		TTL:     5 * time.Minute,
	}

	cache, err := NewCache(config)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	result := &Result{
		Decision: true,
		Allowed:  true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := "test/allow"
		input := map[string]interface{}{
			"user":   "alice",
			"action": "read",
			"seq":    i, // 变化的输入确保不同的键
		}
		cache.Set(ctx, path, input, result)
	}
}

// BenchmarkCacheGetSetConcurrent 测试并发缓存操作性能
func BenchmarkCacheGetSetConcurrent(b *testing.B) {
	config := &CacheConfig{
		Enabled: true,
		MaxSize: 10000,
		TTL:     5 * time.Minute,
	}

	cache, err := NewCache(config)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			path := "test/allow"
			input := map[string]interface{}{
				"user":   "alice",
				"action": "read",
				"seq":    i,
			}

			result := &Result{
				Decision: true,
				Allowed:  true,
			}

			// 50% 读，50% 写
			if i%2 == 0 {
				cache.Get(ctx, path, input)
			} else {
				cache.Set(ctx, path, input, result)
			}
			i++
		}
	})
}

// TestCachePerformanceP99 测试缓存 P99 性能 (< 5ms)
func TestCachePerformanceP99(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		MaxSize: 10000,
		TTL:     5 * time.Minute,
	}

	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	path := "test/allow"

	result := &Result{
		Decision: true,
		Allowed:  true,
	}

	// 预填充缓存
	for i := 0; i < 100; i++ {
		input := map[string]interface{}{
			"user":   "alice",
			"action": "read",
			"seq":    i,
		}
		cache.Set(ctx, path, input, result)
	}

	// 测量延迟
	const iterations = 10000
	latencies := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		input := map[string]interface{}{
			"user":   "alice",
			"action": "read",
			"seq":    i % 100, // 90% 缓存命中率
		}

		start := time.Now()
		cache.Get(ctx, path, input)
		latencies[i] = time.Since(start)
	}

	// 计算 P99
	p99 := calculateP99(latencies)

	// 验收标准：P99 < 5ms
	maxP99 := 5 * time.Millisecond
	if p99 > maxP99 {
		t.Errorf("P99 latency %v exceeds limit %v", p99, maxP99)
	}

	t.Logf("Cache P99 latency: %v", p99)
}

// calculateP99 计算 P99 延迟
func calculateP99(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// 简单排序法计算 P99
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// 使用插入排序（对小数组效率足够）
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > key {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}

	// 计算 P99 索引
	p99Index := int(float64(len(sorted)) * 0.99)
	if p99Index >= len(sorted) {
		p99Index = len(sorted) - 1
	}

	return sorted[p99Index]
}

// TestCacheDelete 测试缓存删除功能
func TestCacheDelete(t *testing.T) {
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

	// 添加不同路径的条目
	cache.Set(ctx, "path1/allow", map[string]interface{}{"key": "1"}, result)
	cache.Set(ctx, "path1/deny", map[string]interface{}{"key": "2"}, result)
	cache.Set(ctx, "path2/allow", map[string]interface{}{"key": "3"}, result)

	// 删除 path1 的所有缓存
	cache.Delete(ctx, "path1")

	// 验证 path1 的缓存已删除，path2 的缓存仍存在
	_, ok := cache.Get(ctx, "path1/allow", map[string]interface{}{"key": "1"})
	if ok {
		t.Error("Expected path1/allow to be deleted")
	}

	_, ok = cache.Get(ctx, "path2/allow", map[string]interface{}{"key": "3"})
	if !ok {
		t.Error("Expected path2/allow to still exist")
	}
}

// TestCacheThreadSafety 测试缓存线程安全性
func TestCacheThreadSafety(t *testing.T) {
	config := &CacheConfig{
		Enabled: true,
		MaxSize: 1000,
		TTL:     5 * time.Minute,
	}

	cache, err := NewCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	ctx := context.Background()
	const goroutines = 100
	const operations = 100

	done := make(chan bool, goroutines)

	// 启动多个 goroutine 并发操作缓存
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer func() {
				done <- true
			}()

			for i := 0; i < operations; i++ {
				path := "test/allow"
				input := map[string]interface{}{
					"user":   "alice",
					"action": "read",
					"id":     id,
					"seq":    i,
				}

				result := &Result{
					Decision: true,
					Allowed:  true,
				}

				// 执行各种操作
				cache.Set(ctx, path, input, result)
				cache.Get(ctx, path, input)

				if i%10 == 0 {
					cache.Stats()
				}

				if i%20 == 0 {
					cache.Delete(ctx, path)
				}
			}
		}(g)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// 验证缓存仍然可用
	stats := cache.Stats()
	if stats.Size < 0 {
		t.Errorf("Cache size should not be negative: %d", stats.Size)
	}

	t.Logf("Concurrent operations completed. Cache stats: Hits=%d, Misses=%d, Size=%d",
		stats.Hits, stats.Misses, stats.Size)
}

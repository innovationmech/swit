// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oauth2

import (
	"context"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestMemoryTokenCache_BasicOperations(t *testing.T) {
	cache := NewMemoryTokenCache(1 * time.Hour)
	defer cache.Close()

	ctx := context.Background()

	// Test Set and Get
	token := &oauth2.Token{
		AccessToken: "test-token-1",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	err := cache.Set(ctx, "key1", token)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("Expected token %v, got %v", token.AccessToken, retrieved.AccessToken)
	}

	// Test Get non-existent key
	_, err = cache.Get(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent key, got nil")
	}

	// Test Delete
	err = cache.Delete(ctx, "key1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = cache.Get(ctx, "key1")
	if err == nil {
		t.Error("Expected error after delete, got nil")
	}
}

func TestMemoryTokenCache_TTL(t *testing.T) {
	// Use a short TTL for testing
	cache := NewMemoryTokenCache(100 * time.Millisecond)
	defer cache.Close()

	ctx := context.Background()

	token := &oauth2.Token{
		AccessToken: "test-token-ttl",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	err := cache.Set(ctx, "key-ttl", token)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should be available immediately
	_, err = cache.Get(ctx, "key-ttl")
	if err != nil {
		t.Errorf("Get failed immediately after set: %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should be expired now
	_, err = cache.Get(ctx, "key-ttl")
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}

func TestMemoryTokenCache_TTLOverride(t *testing.T) {
	// Default TTL is 1 hour
	cache := NewMemoryTokenCache(1 * time.Hour)
	defer cache.Close()

	ctx := context.Background()

	token := &oauth2.Token{
		AccessToken: "test-token-override",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	// Set with custom TTL of 100ms
	err := cache.Set(ctx, "key-override", token, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Set with TTL override failed: %v", err)
	}

	// Should be available immediately
	_, err = cache.Get(ctx, "key-override")
	if err != nil {
		t.Errorf("Get failed immediately after set: %v", err)
	}

	// Wait for custom TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should be expired now
	_, err = cache.Get(ctx, "key-override")
	if err == nil {
		t.Error("Expected error for expired token with custom TTL, got nil")
	}
}

func TestMemoryTokenCache_LRUEviction(t *testing.T) {
	// Create cache with max size of 3
	cache := NewMemoryTokenCacheWithSize(1*time.Hour, 3)
	defer cache.Close()

	ctx := context.Background()

	// Add 3 tokens
	for i := 1; i <= 3; i++ {
		token := &oauth2.Token{
			AccessToken: "token-" + string(rune('0'+i)),
			Expiry:      time.Now().Add(1 * time.Hour),
		}
		err := cache.Set(ctx, "key"+string(rune('0'+i)), token)
		if err != nil {
			t.Fatalf("Set failed for key%d: %v", i, err)
		}
	}

	// Verify all 3 are in cache
	stats := cache.Stats()
	if stats.Size != 3 {
		t.Errorf("Expected cache size 3, got %d", stats.Size)
	}

	// Add a 4th token, should evict the LRU (key1)
	token4 := &oauth2.Token{
		AccessToken: "token-4",
		Expiry:      time.Now().Add(1 * time.Hour),
	}
	err := cache.Set(ctx, "key4", token4)
	if err != nil {
		t.Fatalf("Set failed for key4: %v", err)
	}

	// Verify size is still 3
	stats = cache.Stats()
	if stats.Size != 3 {
		t.Errorf("Expected cache size 3 after eviction, got %d", stats.Size)
	}

	// Verify key1 was evicted
	_, err = cache.Get(ctx, "key1")
	if err == nil {
		t.Error("Expected key1 to be evicted, but it's still in cache")
	}

	// Verify key2, key3, key4 are still there
	for _, key := range []string{"key2", "key3", "key4"} {
		_, err := cache.Get(ctx, key)
		if err != nil {
			t.Errorf("Expected %s to be in cache, but got error: %v", key, err)
		}
	}

	// Verify eviction count
	if stats.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got %d", stats.Evictions)
	}
}

func TestMemoryTokenCache_LRUAccessOrder(t *testing.T) {
	// Create cache with max size of 2
	cache := NewMemoryTokenCacheWithSize(1*time.Hour, 2)
	defer cache.Close()

	ctx := context.Background()

	// Add 2 tokens
	token1 := &oauth2.Token{AccessToken: "token-1", Expiry: time.Now().Add(1 * time.Hour)}
	token2 := &oauth2.Token{AccessToken: "token-2", Expiry: time.Now().Add(1 * time.Hour)}

	cache.Set(ctx, "key1", token1)
	cache.Set(ctx, "key2", token2)

	// Access key1 to make it most recently used
	cache.Get(ctx, "key1")

	// Add key3, should evict key2 (least recently used)
	token3 := &oauth2.Token{AccessToken: "token-3", Expiry: time.Now().Add(1 * time.Hour)}
	cache.Set(ctx, "key3", token3)

	// Verify key2 was evicted
	_, err := cache.Get(ctx, "key2")
	if err == nil {
		t.Error("Expected key2 to be evicted, but it's still in cache")
	}

	// Verify key1 and key3 are still there
	_, err = cache.Get(ctx, "key1")
	if err != nil {
		t.Errorf("Expected key1 to be in cache: %v", err)
	}

	_, err = cache.Get(ctx, "key3")
	if err != nil {
		t.Errorf("Expected key3 to be in cache: %v", err)
	}
}

func TestMemoryTokenCache_Stats(t *testing.T) {
	cache := NewMemoryTokenCacheWithSize(1*time.Hour, 10)
	defer cache.Close()

	ctx := context.Background()

	// Initial stats
	stats := cache.Stats()
	if stats.Size != 0 || stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("Expected empty cache stats, got %+v", stats)
	}

	// Add some tokens
	for i := 0; i < 5; i++ {
		token := &oauth2.Token{
			AccessToken: "token-" + string(rune('0'+i)),
			Expiry:      time.Now().Add(1 * time.Hour),
		}
		cache.Set(ctx, "key"+string(rune('0'+i)), token)
	}

	// Check hits and misses
	cache.Get(ctx, "key0")            // hit
	cache.Get(ctx, "key1")            // hit
	cache.Get(ctx, "key-nonexistent") // miss

	stats = cache.Stats()
	if stats.Size != 5 {
		t.Errorf("Expected size 5, got %d", stats.Size)
	}
	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	// Check hit rate
	expectedHitRate := 2.0 / 3.0
	if stats.HitRate < expectedHitRate-0.01 || stats.HitRate > expectedHitRate+0.01 {
		t.Errorf("Expected hit rate ~%.2f, got %.2f", expectedHitRate, stats.HitRate)
	}
}

func TestMemoryTokenCache_Clear(t *testing.T) {
	cache := NewMemoryTokenCache(1 * time.Hour)
	defer cache.Close()

	ctx := context.Background()

	// Add some tokens
	for i := 0; i < 5; i++ {
		token := &oauth2.Token{
			AccessToken: "token-" + string(rune('0'+i)),
			Expiry:      time.Now().Add(1 * time.Hour),
		}
		cache.Set(ctx, "key"+string(rune('0'+i)), token)
	}

	stats := cache.Stats()
	if stats.Size != 5 {
		t.Errorf("Expected size 5 before clear, got %d", stats.Size)
	}

	// Clear cache
	err := cache.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	stats = cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Expected size 0 after clear, got %d", stats.Size)
	}

	// Verify tokens are gone
	_, err = cache.Get(ctx, "key0")
	if err == nil {
		t.Error("Expected error after clear, got nil")
	}
}

func TestMemoryTokenCache_Warmup(t *testing.T) {
	cache := NewMemoryTokenCache(1 * time.Hour)
	defer cache.Close()

	ctx := context.Background()

	// Prepare tokens for warmup
	tokens := make(map[string]*oauth2.Token)
	for i := 0; i < 5; i++ {
		tokens["key"+string(rune('0'+i))] = &oauth2.Token{
			AccessToken: "token-" + string(rune('0'+i)),
			Expiry:      time.Now().Add(1 * time.Hour),
		}
	}

	// Warmup cache
	err := cache.Warmup(ctx, tokens)
	if err != nil {
		t.Fatalf("Warmup failed: %v", err)
	}

	// Verify all tokens are in cache
	stats := cache.Stats()
	if stats.Size != 5 {
		t.Errorf("Expected size 5 after warmup, got %d", stats.Size)
	}

	for key, expectedToken := range tokens {
		token, err := cache.Get(ctx, key)
		if err != nil {
			t.Errorf("Expected %s to be in cache after warmup: %v", key, err)
		}
		if token.AccessToken != expectedToken.AccessToken {
			t.Errorf("Token mismatch for %s: expected %s, got %s",
				key, expectedToken.AccessToken, token.AccessToken)
		}
	}
}

func TestMemoryTokenCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryTokenCacheWithSize(1*time.Hour, 100)
	defer cache.Close()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Number of concurrent goroutines
	numGoroutines := 10
	numOperations := 100

	// Concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				token := &oauth2.Token{
					AccessToken: "token-" + string(rune('0'+id)) + "-" + string(rune('0'+j)),
					Expiry:      time.Now().Add(1 * time.Hour),
				}
				cache.Set(ctx, "key-"+string(rune('0'+id))+"-"+string(rune('0'+j)), token)
			}
		}(i)
	}

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cache.Get(ctx, "key-"+string(rune('0'+id))+"-"+string(rune('0'+j)))
			}
		}(i)
	}

	wg.Wait()

	// No assertions - just checking for race conditions
	// Run with: go test -race
}

func TestMemoryTokenCache_AutoCleanup(t *testing.T) {
	// Use very short TTL for testing
	cache := NewMemoryTokenCache(100 * time.Millisecond)
	defer cache.Close()

	ctx := context.Background()

	// Add some tokens
	for i := 0; i < 5; i++ {
		token := &oauth2.Token{
			AccessToken: "token-" + string(rune('0'+i)),
			Expiry:      time.Now().Add(1 * time.Hour),
		}
		cache.Set(ctx, "key"+string(rune('0'+i)), token)
	}

	stats := cache.Stats()
	if stats.Size != 5 {
		t.Errorf("Expected size 5 before cleanup, got %d", stats.Size)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Manually trigger cleanup to verify expired tokens are removed
	cache.cleanup()

	// All tokens should be expired and cleaned up
	stats = cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Expected size 0 after cleanup, got %d", stats.Size)
	}
}

func TestMemoryTokenCache_ExpiredTokenDetection(t *testing.T) {
	cache := NewMemoryTokenCache(1 * time.Hour)
	defer cache.Close()

	ctx := context.Background()

	// Add token that's already expired
	expiredToken := &oauth2.Token{
		AccessToken: "expired-token",
		Expiry:      time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	err := cache.Set(ctx, "expired-key", expiredToken)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Try to get expired token
	_, err = cache.Get(ctx, "expired-key")
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}

	// Verify it's counted as a miss
	stats := cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss for expired token, got %d", stats.Misses)
	}
}

func TestMemoryTokenCache_Update(t *testing.T) {
	cache := NewMemoryTokenCache(1 * time.Hour)
	defer cache.Close()

	ctx := context.Background()

	// Add initial token
	token1 := &oauth2.Token{
		AccessToken: "token-1",
		Expiry:      time.Now().Add(1 * time.Hour),
	}
	cache.Set(ctx, "key1", token1)

	// Update with new token
	token2 := &oauth2.Token{
		AccessToken: "token-2",
		Expiry:      time.Now().Add(2 * time.Hour),
	}
	cache.Set(ctx, "key1", token2)

	// Verify updated token
	retrieved, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.AccessToken != token2.AccessToken {
		t.Errorf("Expected updated token %v, got %v", token2.AccessToken, retrieved.AccessToken)
	}

	// Verify size didn't change
	stats := cache.Stats()
	if stats.Size != 1 {
		t.Errorf("Expected size 1 after update, got %d", stats.Size)
	}
}

func TestCachingTokenSource(t *testing.T) {
	cache := NewMemoryTokenCache(1 * time.Hour)
	defer cache.Close()

	// Create a mock token source
	mockSource := &mockTokenSource{
		token: &oauth2.Token{
			AccessToken: "mock-token",
			Expiry:      time.Now().Add(1 * time.Hour),
		},
	}

	cachingSource := NewCachingTokenSource(mockSource, cache, "test-key")

	// First call should fetch from source
	token1, err := cachingSource.Token()
	if err != nil {
		t.Fatalf("Token() failed: %v", err)
	}

	if mockSource.callCount != 1 {
		t.Errorf("Expected 1 call to source, got %d", mockSource.callCount)
	}

	// Second call should get from cache
	token2, err := cachingSource.Token()
	if err != nil {
		t.Fatalf("Token() failed: %v", err)
	}

	if mockSource.callCount != 1 {
		t.Errorf("Expected still 1 call to source (cached), got %d", mockSource.callCount)
	}

	// Verify same token
	if token1.AccessToken != token2.AccessToken {
		t.Errorf("Token mismatch: expected %v, got %v", token1.AccessToken, token2.AccessToken)
	}
}

func TestTokenCacheConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *TokenCacheConfig
		setDefaults bool
		wantErr     bool
	}{
		{
			name: "valid config",
			config: &TokenCacheConfig{
				Enabled: true,
				Type:    "memory",
				TTL:     1 * time.Hour,
				MaxSize: 100,
			},
			setDefaults: true,
			wantErr:     false,
		},
		{
			name: "disabled config",
			config: &TokenCacheConfig{
				Enabled: false,
			},
			setDefaults: true,
			wantErr:     false,
		},
		{
			name: "invalid type",
			config: &TokenCacheConfig{
				Enabled: true,
				Type:    "redis", // not yet supported
				TTL:     1 * time.Hour,
			},
			setDefaults: false,
			wantErr:     true,
		},
		{
			name: "zero ttl gets default",
			config: &TokenCacheConfig{
				Enabled: true,
				Type:    "memory",
				TTL:     0,
			},
			setDefaults: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setDefaults {
				tt.config.SetDefaults()
			}
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Mock token source for testing
type mockTokenSource struct {
	token     *oauth2.Token
	callCount int
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	m.callCount++
	return m.token, nil
}

// Benchmark tests
func BenchmarkMemoryTokenCache_Set(b *testing.B) {
	cache := NewMemoryTokenCache(1 * time.Hour)
	defer cache.Close()

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "benchmark-token",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(ctx, "key", token)
	}
}

func BenchmarkMemoryTokenCache_Get(b *testing.B) {
	cache := NewMemoryTokenCache(1 * time.Hour)
	defer cache.Close()

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "benchmark-token",
		Expiry:      time.Now().Add(1 * time.Hour),
	}
	cache.Set(ctx, "key", token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(ctx, "key")
	}
}

func BenchmarkMemoryTokenCache_Concurrent(b *testing.B) {
	cache := NewMemoryTokenCache(1 * time.Hour)
	defer cache.Close()

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "benchmark-token",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Set(ctx, "key", token)
			cache.Get(ctx, "key")
		}
	})
}

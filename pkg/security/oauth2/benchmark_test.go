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

package oauth2

import (
	"context"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// ============================================================================
// OAuth2 Token Cache Benchmarks
// Target: < 10ms P99 for cache operations
// ============================================================================

// BenchmarkTokenCache_GetHit benchmarks cache retrieval performance.
func BenchmarkTokenCache_GetHit(b *testing.B) {
	cache := NewMemoryTokenCacheWithSize(time.Hour, 1000)
	defer cache.Close()

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "test-access-token-12345",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	// Pre-populate the cache
	if err := cache.Set(ctx, "test-key", token); err != nil {
		b.Fatalf("Failed to set token: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := cache.Get(ctx, "test-key")
		if err != nil {
			b.Fatalf("Get() error = %v", err)
		}
	}
}

// BenchmarkTokenCache_SetNew benchmarks cache storage performance.
func BenchmarkTokenCache_SetNew(b *testing.B) {
	cache := NewMemoryTokenCacheWithSize(time.Hour, 10000)
	defer cache.Close()

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "test-access-token-12345",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := cache.Set(ctx, "test-key", token)
		if err != nil {
			b.Fatalf("Set() error = %v", err)
		}
	}
}

// BenchmarkTokenCache_GetMiss benchmarks cache miss performance.
func BenchmarkTokenCache_GetMiss(b *testing.B) {
	cache := NewMemoryTokenCacheWithSize(time.Hour, 1000)
	defer cache.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(ctx, "non-existent-key")
	}
}

// BenchmarkMemoryTokenCache_SetWithEviction benchmarks cache with LRU eviction.
func BenchmarkMemoryTokenCache_SetWithEviction(b *testing.B) {
	// Small cache to trigger frequent evictions
	cache := NewMemoryTokenCacheWithSize(time.Hour, 100)
	defer cache.Close()

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "test-access-token-12345",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := string(rune(i % 200)) // Force evictions
		err := cache.Set(ctx, key, token)
		if err != nil {
			b.Fatalf("Set() error = %v", err)
		}
	}
}

// BenchmarkMemoryTokenCache_ConcurrentAccess benchmarks concurrent cache access.
func BenchmarkMemoryTokenCache_ConcurrentAccess(b *testing.B) {
	cache := NewMemoryTokenCacheWithSize(time.Hour, 1000)
	defer cache.Close()

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "test-access-token-12345",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	// Pre-populate the cache
	for i := 0; i < 100; i++ {
		key := string(rune(i))
		if err := cache.Set(ctx, key, token); err != nil {
			b.Fatalf("Failed to set token: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := string(rune(i % 100))
			if i%2 == 0 {
				_, _ = cache.Get(ctx, key)
			} else {
				_ = cache.Set(ctx, key, token)
			}
			i++
		}
	})
}

// BenchmarkMemoryTokenCache_Stats benchmarks stats retrieval.
func BenchmarkMemoryTokenCache_Stats(b *testing.B) {
	cache := NewMemoryTokenCacheWithSize(time.Hour, 1000)
	defer cache.Close()

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "test-access-token-12345",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	// Pre-populate the cache
	for i := 0; i < 500; i++ {
		key := string(rune(i))
		if err := cache.Set(ctx, key, token); err != nil {
			b.Fatalf("Failed to set token: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = cache.Stats()
	}
}

// BenchmarkMemoryTokenCache_Delete benchmarks cache deletion.
func BenchmarkMemoryTokenCache_Delete(b *testing.B) {
	cache := NewMemoryTokenCacheWithSize(time.Hour, 10000)
	defer cache.Close()

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "test-access-token-12345",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	// Pre-populate the cache
	for i := 0; i < b.N; i++ {
		key := string(rune(i))
		if err := cache.Set(ctx, key, token); err != nil {
			b.Fatalf("Failed to set token: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := string(rune(i))
		_ = cache.Delete(ctx, key)
	}
}

// BenchmarkMemoryTokenCache_Warmup benchmarks cache warmup.
func BenchmarkMemoryTokenCache_Warmup(b *testing.B) {
	ctx := context.Background()

	// Prepare tokens for warmup
	tokens := make(map[string]*oauth2.Token)
	for i := 0; i < 100; i++ {
		key := string(rune(i))
		tokens[key] = &oauth2.Token{
			AccessToken: "test-access-token-" + key,
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(time.Hour),
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache := NewMemoryTokenCacheWithSize(time.Hour, 1000)
		err := cache.Warmup(ctx, tokens)
		if err != nil {
			b.Fatalf("Warmup() error = %v", err)
		}
		cache.Close()
	}
}

// ============================================================================
// Token Helper Functions Benchmarks
// ============================================================================

// BenchmarkToken_GetClaim benchmarks claim retrieval from token.
func BenchmarkToken_GetClaim(b *testing.B) {
	token := &Token{
		Token: &oauth2.Token{
			AccessToken: "test-access-token",
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(time.Hour),
		},
		Claims: map[string]interface{}{
			"sub":   "user123",
			"email": "user@example.com",
			"roles": []string{"admin", "user"},
			"iat":   time.Now().Unix(),
			"exp":   time.Now().Add(time.Hour).Unix(),
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = token.GetClaim("sub")
	}
}

// BenchmarkToken_GetStringClaim benchmarks string claim retrieval.
func BenchmarkToken_GetStringClaim(b *testing.B) {
	token := &Token{
		Token: &oauth2.Token{
			AccessToken: "test-access-token",
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(time.Hour),
		},
		Claims: map[string]interface{}{
			"sub":   "user123",
			"email": "user@example.com",
			"name":  "John Doe",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = token.GetStringClaim("sub")
	}
}

// BenchmarkToken_GetRoles benchmarks roles extraction from token.
func BenchmarkToken_GetRoles(b *testing.B) {
	token := &Token{
		Token: &oauth2.Token{
			AccessToken: "test-access-token",
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(time.Hour),
		},
		Claims: map[string]interface{}{
			"sub":   "user123",
			"roles": []interface{}{"admin", "user", "moderator"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = token.GetRoles()
	}
}

// BenchmarkToken_GetSubject benchmarks subject extraction.
func BenchmarkToken_GetSubject(b *testing.B) {
	token := &Token{
		Token: &oauth2.Token{
			AccessToken: "test-access-token",
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(time.Hour),
		},
		Claims: map[string]interface{}{
			"sub": "user123",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = token.GetSubject()
	}
}

// BenchmarkToken_ToJSON benchmarks token serialization.
func BenchmarkToken_ToJSON(b *testing.B) {
	token := &Token{
		Token: &oauth2.Token{
			AccessToken:  "test-access-token-12345",
			RefreshToken: "test-refresh-token-67890",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(time.Hour),
		},
		IDToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
		Claims: map[string]interface{}{
			"sub":   "user123",
			"email": "user@example.com",
			"roles": []string{"admin", "user"},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := token.ToJSON()
		if err != nil {
			b.Fatalf("ToJSON() error = %v", err)
		}
	}
}

// BenchmarkToken_FromJSON benchmarks token deserialization.
func BenchmarkToken_FromJSON(b *testing.B) {
	token := &Token{
		Token: &oauth2.Token{
			AccessToken:  "test-access-token-12345",
			RefreshToken: "test-refresh-token-67890",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(time.Hour),
		},
		IDToken: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
		Claims: map[string]interface{}{
			"sub":   "user123",
			"email": "user@example.com",
		},
	}

	data, err := token.ToJSON()
	if err != nil {
		b.Fatalf("ToJSON() error = %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := FromJSON(data)
		if err != nil {
			b.Fatalf("FromJSON() error = %v", err)
		}
	}
}

// ============================================================================
// Token Introspection Response Benchmarks
// ============================================================================

// BenchmarkTokenIntrospection_IsExpired benchmarks expiration check.
func BenchmarkTokenIntrospection_IsExpired(b *testing.B) {
	introspection := &TokenIntrospection{
		Active: true,
		Sub:    "user123",
		Exp:    time.Now().Add(time.Hour).Unix(),
		Iat:    time.Now().Unix(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = introspection.IsExpired()
	}
}

// BenchmarkTokenIntrospection_ExpiresAt benchmarks expiration time retrieval.
func BenchmarkTokenIntrospection_ExpiresAt(b *testing.B) {
	introspection := &TokenIntrospection{
		Active: true,
		Sub:    "user123",
		Exp:    time.Now().Add(time.Hour).Unix(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = introspection.ExpiresAt()
	}
}

// ============================================================================
// Token Expiry Helper Benchmarks
// ============================================================================

// BenchmarkIsTokenExpired benchmarks token expiration check.
func BenchmarkIsTokenExpired(b *testing.B) {
	token := &oauth2.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = IsTokenExpired(token)
	}
}

// BenchmarkIsTokenExpiring benchmarks token expiring-soon check.
func BenchmarkIsTokenExpiring(b *testing.B) {
	token := &oauth2.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = IsTokenExpiring(token, 5*time.Minute)
	}
}

// ============================================================================
// CachingTokenSource Benchmarks
// ============================================================================

// benchMockTokenSource is a mock implementation for benchmarking.
type benchMockTokenSource struct {
	token *oauth2.Token
}

func (m *benchMockTokenSource) Token() (*oauth2.Token, error) {
	return m.token, nil
}

// BenchmarkCachingTokenSource_TokenCacheHit benchmarks token source with cache hit.
func BenchmarkCachingTokenSource_TokenCacheHit(b *testing.B) {
	cache := NewMemoryTokenCacheWithSize(time.Hour, 1000)
	defer cache.Close()

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "test-access-token-12345",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	// Pre-populate the cache
	if err := cache.Set(ctx, "test-key", token); err != nil {
		b.Fatalf("Failed to set token: %v", err)
	}

	source := &benchMockTokenSource{token: token}
	cachingSource := NewCachingTokenSource(source, cache, "test-key")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := cachingSource.Token()
		if err != nil {
			b.Fatalf("Token() error = %v", err)
		}
	}
}

// BenchmarkCachingTokenSource_TokenCacheMiss benchmarks token source with cache miss.
func BenchmarkCachingTokenSource_TokenCacheMiss(b *testing.B) {
	cache := NewMemoryTokenCacheWithSize(time.Hour, 1000)
	defer cache.Close()

	token := &oauth2.Token{
		AccessToken: "test-access-token-12345",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	source := &benchMockTokenSource{token: token}
	cachingSource := NewCachingTokenSource(source, cache, "test-key")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := cachingSource.Token()
		if err != nil {
			b.Fatalf("Token() error = %v", err)
		}
	}
}

// ============================================================================
// Large Scale Benchmarks
// ============================================================================

// BenchmarkMemoryTokenCache_LargeScale benchmarks cache with many entries.
func BenchmarkMemoryTokenCache_LargeScale(b *testing.B) {
	cache := NewMemoryTokenCacheWithSize(time.Hour, 100000)
	defer cache.Close()

	ctx := context.Background()

	// Pre-populate with 10000 tokens
	for i := 0; i < 10000; i++ {
		key := string(rune(i))
		token := &oauth2.Token{
			AccessToken: "test-access-token-" + key,
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(time.Hour),
		}
		if err := cache.Set(ctx, key, token); err != nil {
			b.Fatalf("Failed to set token: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := string(rune(i % 10000))
			_, _ = cache.Get(ctx, key)
			i++
		}
	})
}

// BenchmarkMemoryTokenCache_HighContention benchmarks cache under high contention.
func BenchmarkMemoryTokenCache_HighContention(b *testing.B) {
	cache := NewMemoryTokenCacheWithSize(time.Hour, 1000)
	defer cache.Close()

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "test-access-token-12345",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	// All goroutines access the same key (high contention)
	if err := cache.Set(ctx, "hot-key", token); err != nil {
		b.Fatalf("Failed to set token: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cache.Get(ctx, "hot-key")
		}
	})
}

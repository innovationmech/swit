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

package resilience

import (
	"sync"
	"testing"
	"time"
)

func TestTokenBucket_Basics(t *testing.T) {
	tb := NewTokenBucket(10, 5.0)
	if tb.Capacity() != 10 {
		t.Errorf("Capacity() = %d, want 10", tb.Capacity())
	}
	if tb.RefillRate() != 5.0 {
		t.Errorf("RefillRate() = %f, want 5.0", tb.RefillRate())
	}
	// 初始时桶是满的
	if got := tb.Tokens(); got < 9.9 || got > 10.0 {
		t.Errorf("Tokens() = %f, want ~10", got)
	}
}

func TestTokenBucket_AllowBurst(t *testing.T) {
	tb := NewTokenBucket(5, 1.0)

	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if tb.Allow() {
		t.Error("request beyond capacity should be denied")
	}
}

func TestTokenBucket_AllowN(t *testing.T) {
	tb := NewTokenBucket(10, 5.0)

	if !tb.AllowN(5) {
		t.Error("AllowN(5) should succeed with full bucket")
	}
	if tb.AllowN(6) {
		t.Error("AllowN(6) should fail with 5 tokens remaining")
	}
	if !tb.AllowN(5) {
		t.Error("AllowN(5) should succeed with 5 tokens remaining")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	tb := NewTokenBucket(10, 100.0) // 每秒补充 100 个令牌

	if !tb.AllowN(10) {
		t.Fatal("draining bucket should succeed")
	}
	if tb.Allow() {
		t.Fatal("empty bucket should deny")
	}

	time.Sleep(50 * time.Millisecond) // 补充约 5 个令牌
	if !tb.Allow() {
		t.Error("bucket should allow after refill")
	}
}

func TestLeakyBucket_Basics(t *testing.T) {
	lb := NewLeakyBucket(10, 5.0)
	if lb.Capacity() != 10 {
		t.Errorf("Capacity() = %d, want 10", lb.Capacity())
	}
	if lb.LeakRate() != 5.0 {
		t.Errorf("LeakRate() = %f, want 5.0", lb.LeakRate())
	}
	// 初始时桶是空的
	if got := lb.Water(); got != 0 {
		t.Errorf("Water() = %f, want 0", got)
	}
}

func TestLeakyBucket_AllowUntilFull(t *testing.T) {
	lb := NewLeakyBucket(3, 0.001) // 几乎不漏水

	for i := 0; i < 3; i++ {
		if !lb.Allow() {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if lb.Allow() {
		t.Error("request beyond capacity should be denied")
	}
}

func TestLeakyBucket_Leak(t *testing.T) {
	lb := NewLeakyBucket(5, 100.0) // 每秒漏出 100

	if !lb.AllowN(5) {
		t.Fatal("filling bucket should succeed")
	}
	if lb.Allow() {
		t.Fatal("full bucket should deny")
	}

	time.Sleep(50 * time.Millisecond) // 漏出约 5
	if !lb.Allow() {
		t.Error("bucket should allow after leaking")
	}
}

func TestKeyedRateLimiter_IndependentKeys(t *testing.T) {
	krl := NewKeyedTokenBucketLimiter(3, 1.0)

	for i := 0; i < 3; i++ {
		if !krl.Allow("client-a") {
			t.Fatalf("client-a request %d should be allowed", i)
		}
	}
	if krl.Allow("client-a") {
		t.Error("client-a should be rate limited")
	}

	// client-b 独立计数
	if !krl.Allow("client-b") {
		t.Error("client-b should be allowed independently")
	}

	if krl.Len() != 2 {
		t.Errorf("Len() = %d, want 2", krl.Len())
	}
}

func TestKeyedRateLimiter_LeakyBucketBackend(t *testing.T) {
	krl := NewKeyedLeakyBucketLimiter(2, 0.001)

	if !krl.Allow("client-a") || !krl.Allow("client-a") {
		t.Fatal("first two requests should be allowed")
	}
	if krl.Allow("client-a") {
		t.Error("third request should be denied")
	}
}

func TestKeyedRateLimiter_Cleanup(t *testing.T) {
	krl := NewKeyedTokenBucketLimiter(10, 5.0)

	krl.Allow("client-a")
	krl.Allow("client-b")
	krl.Allow("client-c")

	if krl.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", krl.Len())
	}

	removed := krl.Cleanup(0) // 立即清理所有空闲桶
	if removed != 3 {
		t.Errorf("Cleanup(0) removed %d, want 3", removed)
	}
	if krl.Len() != 0 {
		t.Errorf("Len() after cleanup = %d, want 0", krl.Len())
	}
}

func TestKeyedRateLimiter_CleanupKeepsActive(t *testing.T) {
	krl := NewKeyedTokenBucketLimiter(10, 5.0)

	krl.Allow("client-a")
	removed := krl.Cleanup(time.Hour)
	if removed != 0 {
		t.Errorf("Cleanup(1h) removed %d, want 0", removed)
	}
	if krl.Len() != 1 {
		t.Errorf("Len() = %d, want 1", krl.Len())
	}
}

func TestKeyedRateLimiter_ConcurrentAccess(t *testing.T) {
	krl := NewKeyedTokenBucketLimiter(100, 50.0)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				krl.Allow(key)
			}
		}(string(rune('A' + i)))
	}

	wg.Wait()
	if krl.Len() != 10 {
		t.Errorf("Len() = %d, want 10", krl.Len())
	}
}

func TestKeyedRateLimiter_CustomFactory(t *testing.T) {
	created := 0
	krl := NewKeyedRateLimiter(func() RateLimiter {
		created++
		return NewTokenBucket(1, 1.0)
	})

	krl.Allow("a")
	krl.Allow("a")
	krl.Allow("b")

	if created != 2 {
		t.Errorf("factory called %d times, want 2", created)
	}
}

func BenchmarkTokenBucket_Allow(b *testing.B) {
	tb := NewTokenBucket(1000000, 1000000.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow()
	}
}

func BenchmarkKeyedRateLimiter_Allow(b *testing.B) {
	krl := NewKeyedTokenBucketLimiter(1000000, 1000000.0)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			krl.Allow("client-1")
		}
	})
}

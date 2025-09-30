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

package middleware

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTokenBucket(t *testing.T) {
	tb := NewTokenBucket(10, 5.0)
	assert.NotNil(t, tb)
	assert.Equal(t, int64(10), tb.capacity)
	assert.Equal(t, 5.0, tb.refillRate)
	assert.Equal(t, float64(10), tb.tokens) // 初始时桶是满的
}

func TestTokenBucket_Allow(t *testing.T) {
	tests := []struct {
		name     string
		capacity int64
		rate     float64
		requests int
		wait     time.Duration
		expected []bool
	}{
		{
			name:     "允许突发流量",
			capacity: 5,
			rate:     1.0,
			requests: 6,
			wait:     0,
			expected: []bool{true, true, true, true, true, false},
		},
		{
			name:     "令牌补充后允许新请求",
			capacity: 2,
			rate:     10.0, // 每秒补充 10 个令牌
			requests: 3,
			wait:     150 * time.Millisecond, // 等待补充约 1.5 个令牌
			expected: []bool{true, true, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTokenBucket(tt.capacity, tt.rate)

			for i := 0; i < tt.requests; i++ {
				if i == 2 && tt.wait > 0 {
					time.Sleep(tt.wait)
				}
				result := tb.Allow()
				assert.Equal(t, tt.expected[i], result, "request %d", i)
			}
		})
	}
}

func TestTokenBucket_AllowN(t *testing.T) {
	tb := NewTokenBucket(10, 5.0)

	// 尝试取出 5 个令牌，应该成功
	assert.True(t, tb.AllowN(5))
	assert.InDelta(t, float64(5), tb.Tokens(), 0.1) // 允许小的浮点误差

	// 尝试取出 6 个令牌，应该失败
	assert.False(t, tb.AllowN(6))
	assert.InDelta(t, float64(5), tb.Tokens(), 0.1) // 令牌数不变

	// 尝试取出 5 个令牌，应该成功
	assert.True(t, tb.AllowN(5))
	assert.InDelta(t, float64(0), tb.Tokens(), 0.1)
}

func TestTokenBucket_Refill(t *testing.T) {
	tb := NewTokenBucket(10, 10.0) // 每秒补充 10 个令牌

	// 用完所有令牌
	assert.True(t, tb.AllowN(10))
	assert.InDelta(t, float64(0), tb.Tokens(), 0.1)

	// 等待 0.5 秒，应该补充约 5 个令牌
	time.Sleep(500 * time.Millisecond)
	tokens := tb.Tokens()
	assert.InDelta(t, 5.0, tokens, 1.0) // 允许 1 个令牌的误差

	// 等待 1 秒，应该达到最大容量
	time.Sleep(1 * time.Second)
	tokens = tb.Tokens()
	assert.Equal(t, float64(10), tokens)
}

func TestTokenBucket_ConcurrentAccess(t *testing.T) {
	tb := NewTokenBucket(100, 50.0)
	var wg sync.WaitGroup
	successCount := 0
	var mutex sync.Mutex

	// 100 个 goroutine 并发请求
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.Allow() {
				mutex.Lock()
				successCount++
				mutex.Unlock()
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, 100, successCount) // 初始容量为 100，全部应该成功
}

func TestNewTokenBucketLimiter(t *testing.T) {
	limiter := NewTokenBucketLimiter(10, 5.0)
	assert.NotNil(t, limiter)
	assert.Equal(t, int64(10), limiter.capacity)
	assert.Equal(t, 5.0, limiter.refillRate)
	assert.NotNil(t, limiter.buckets)
}

func TestTokenBucketLimiter_Allow(t *testing.T) {
	limiter := NewTokenBucketLimiter(3, 1.0)

	// 客户端 A 的前 3 个请求应该成功
	assert.True(t, limiter.Allow("client-a"))
	assert.True(t, limiter.Allow("client-a"))
	assert.True(t, limiter.Allow("client-a"))
	assert.False(t, limiter.Allow("client-a")) // 第 4 个失败

	// 客户端 B 的请求应该独立计数
	assert.True(t, limiter.Allow("client-b"))
	assert.True(t, limiter.Allow("client-b"))
	assert.True(t, limiter.Allow("client-b"))
	assert.False(t, limiter.Allow("client-b"))
}

func TestTokenBucketLimiter_Cleanup(t *testing.T) {
	limiter := NewTokenBucketLimiter(10, 5.0)

	// 创建一些桶
	limiter.Allow("client-a")
	limiter.Allow("client-b")
	limiter.Allow("client-c")

	assert.Equal(t, 3, len(limiter.buckets))

	// 清理空闲时间超过 0 的桶（立即清理）
	limiter.Cleanup(0)

	// 所有桶应该被清理
	assert.Equal(t, 0, len(limiter.buckets))
}

func TestTokenBucketLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewTokenBucketLimiter(100, 50.0)
	var wg sync.WaitGroup

	// 多个 goroutine 并发访问不同客户端
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(clientID string) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				limiter.Allow(clientID)
			}
		}(string(rune('A' + i)))
	}

	wg.Wait()
	// 应该有 10 个不同的客户端桶
	assert.Equal(t, 10, len(limiter.buckets))
}

func BenchmarkTokenBucket_Allow(b *testing.B) {
	tb := NewTokenBucket(1000000, 1000000.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow()
	}
}

func BenchmarkTokenBucketLimiter_Allow(b *testing.B) {
	limiter := NewTokenBucketLimiter(1000000, 1000000.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow("client-1")
	}
}

func BenchmarkTokenBucketLimiter_ConcurrentAllow(b *testing.B) {
	limiter := NewTokenBucketLimiter(1000000, 1000000.0)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow("client-1")
		}
	})
}

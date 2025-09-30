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
	"time"
)

// TokenBucket 实现令牌桶算法
// Token bucket 允许突发流量，但平均速率受限
type TokenBucket struct {
	capacity       int64      // 桶的最大容量
	tokens         float64    // 当前令牌数量
	refillRate     float64    // 每秒补充的令牌数
	lastRefillTime time.Time  // 上次补充令牌的时间
	mutex          sync.Mutex // 保护并发访问
}

// NewTokenBucket 创建一个新的令牌桶
// capacity: 桶的最大容量（允许的突发请求数）
// refillRate: 每秒补充的令牌数（平均速率）
func NewTokenBucket(capacity int64, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:       capacity,
		tokens:         float64(capacity), // 初始时桶是满的
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}
}

// Allow 尝试从桶中取出一个令牌
// 如果成功返回 true，否则返回 false
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN 尝试从桶中取出 n 个令牌
func (tb *TokenBucket) AllowN(n int64) bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// 先补充令牌
	tb.refill()

	// 检查是否有足够的令牌
	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}

	return false
}

// refill 根据时间流逝补充令牌
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime).Seconds()

	// 计算需要补充的令牌数
	tokensToAdd := elapsed * tb.refillRate
	tb.tokens += tokensToAdd

	// 确保不超过最大容量
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}

	tb.lastRefillTime = now
}

// Tokens 返回当前可用的令牌数
func (tb *TokenBucket) Tokens() float64 {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refill()
	return tb.tokens
}

// TokenBucketLimiter 为多个客户端（如IP地址）提供令牌桶速率限制
type TokenBucketLimiter struct {
	capacity   int64
	refillRate float64
	buckets    map[string]*TokenBucket
	mutex      sync.RWMutex
}

// NewTokenBucketLimiter 创建一个新的令牌桶限制器
func NewTokenBucketLimiter(capacity int64, refillRate float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		capacity:   capacity,
		refillRate: refillRate,
		buckets:    make(map[string]*TokenBucket),
	}
}

// Allow 检查指定客户端是否允许发送请求
func (tbl *TokenBucketLimiter) Allow(clientID string) bool {
	return tbl.AllowN(clientID, 1)
}

// AllowN 检查指定客户端是否允许发送 n 个请求
func (tbl *TokenBucketLimiter) AllowN(clientID string, n int64) bool {
	bucket := tbl.getOrCreateBucket(clientID)
	return bucket.AllowN(n)
}

// getOrCreateBucket 获取或创建指定客户端的令牌桶
func (tbl *TokenBucketLimiter) getOrCreateBucket(clientID string) *TokenBucket {
	// 先尝试只读锁
	tbl.mutex.RLock()
	bucket, exists := tbl.buckets[clientID]
	tbl.mutex.RUnlock()

	if exists {
		return bucket
	}

	// 需要创建新桶，使用写锁
	tbl.mutex.Lock()
	defer tbl.mutex.Unlock()

	// 双重检查，防止并发创建
	bucket, exists = tbl.buckets[clientID]
	if exists {
		return bucket
	}

	bucket = NewTokenBucket(tbl.capacity, tbl.refillRate)
	tbl.buckets[clientID] = bucket
	return bucket
}

// Cleanup 清理长时间未使用的桶以防止内存泄漏
func (tbl *TokenBucketLimiter) Cleanup(maxIdleTime time.Duration) {
	tbl.mutex.Lock()
	defer tbl.mutex.Unlock()

	now := time.Now()
	for clientID, bucket := range tbl.buckets {
		bucket.mutex.Lock()
		if now.Sub(bucket.lastRefillTime) > maxIdleTime {
			delete(tbl.buckets, clientID)
		}
		bucket.mutex.Unlock()
	}
}

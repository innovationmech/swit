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
	"time"
)

// RateLimiter 定义与传输层无关的限流器核心接口。
// 该接口可同时被服务端中间件（pkg/middleware）与客户端调用
// 侧的弹性组件（pkg/resilience）复用。
type RateLimiter interface {
	// Allow 尝试获取一个配额，成功返回 true。
	Allow() bool

	// AllowN 尝试一次性获取 n 个配额，成功返回 true。
	AllowN(n int64) bool

	// LastActivity 返回限流器最后一次活动的时间，
	// 供空闲清理（Cleanup）判断使用。
	LastActivity() time.Time
}

// TokenBucket 实现令牌桶算法。
// 令牌桶允许突发流量，但平均速率受限。
type TokenBucket struct {
	capacity       int64      // 桶的最大容量
	tokens         float64    // 当前令牌数量
	refillRate     float64    // 每秒补充的令牌数
	lastRefillTime time.Time  // 上次补充令牌的时间
	mutex          sync.Mutex // 保护并发访问
}

// NewTokenBucket 创建一个新的令牌桶。
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

// Allow 尝试从桶中取出一个令牌。
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN 尝试从桶中取出 n 个令牌。
func (tb *TokenBucket) AllowN(n int64) bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refill()

	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}

	return false
}

// refill 根据时间流逝补充令牌。
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime).Seconds()

	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}

	tb.lastRefillTime = now
}

// Tokens 返回当前可用的令牌数。
func (tb *TokenBucket) Tokens() float64 {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refill()
	return tb.tokens
}

// Capacity 返回桶的最大容量。
func (tb *TokenBucket) Capacity() int64 {
	return tb.capacity
}

// RefillRate 返回每秒补充的令牌数。
func (tb *TokenBucket) RefillRate() float64 {
	return tb.refillRate
}

// LastActivity 实现 RateLimiter 接口。
func (tb *TokenBucket) LastActivity() time.Time {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	return tb.lastRefillTime
}

// LeakyBucket 实现漏桶算法。
// 漏桶以恒定速率处理请求，用于平滑流量。
type LeakyBucket struct {
	capacity     int64      // 桶的最大容量（队列大小）
	leakRate     float64    // 每秒漏出的请求数
	water        float64    // 当前桶中的水量
	lastLeakTime time.Time  // 上次漏水的时间
	mutex        sync.Mutex // 保护并发访问
}

// NewLeakyBucket 创建一个新的漏桶。
// capacity: 桶的最大容量（最大队列大小）
// leakRate: 每秒漏出的请求数（处理速率）
func NewLeakyBucket(capacity int64, leakRate float64) *LeakyBucket {
	return &LeakyBucket{
		capacity:     capacity,
		leakRate:     leakRate,
		water:        0,
		lastLeakTime: time.Now(),
	}
}

// Allow 尝试向桶中加入一个请求。
func (lb *LeakyBucket) Allow() bool {
	return lb.AllowN(1)
}

// AllowN 尝试向桶中加入 n 个请求。
func (lb *LeakyBucket) AllowN(n int64) bool {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.leak()

	if lb.water+float64(n) <= float64(lb.capacity) {
		lb.water += float64(n)
		return true
	}

	return false
}

// leak 根据时间流逝漏出请求。
func (lb *LeakyBucket) leak() {
	now := time.Now()
	elapsed := now.Sub(lb.lastLeakTime).Seconds()

	lb.water -= elapsed * lb.leakRate
	if lb.water < 0 {
		lb.water = 0
	}

	lb.lastLeakTime = now
}

// Water 返回当前桶中的水量。
func (lb *LeakyBucket) Water() float64 {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.leak()
	return lb.water
}

// Capacity 返回桶的最大容量。
func (lb *LeakyBucket) Capacity() int64 {
	return lb.capacity
}

// LeakRate 返回每秒漏出的请求数。
func (lb *LeakyBucket) LeakRate() float64 {
	return lb.leakRate
}

// LastActivity 实现 RateLimiter 接口。
func (lb *LeakyBucket) LastActivity() time.Time {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	return lb.lastLeakTime
}

// KeyedRateLimiter 为多个客户端（如 IP、服务名、方法名）提供
// 按 key 分桶的限流能力。底层限流器由 factory 创建，可以是
// TokenBucket、LeakyBucket 或任意 RateLimiter 实现。
type KeyedRateLimiter struct {
	factory  func() RateLimiter
	limiters map[string]RateLimiter
	mutex    sync.RWMutex
}

// NewKeyedRateLimiter 创建一个按 key 分桶的限流器。
// factory 用于为新出现的 key 创建底层限流器。
func NewKeyedRateLimiter(factory func() RateLimiter) *KeyedRateLimiter {
	return &KeyedRateLimiter{
		factory:  factory,
		limiters: make(map[string]RateLimiter),
	}
}

// NewKeyedTokenBucketLimiter 创建以令牌桶为底层实现的按 key 限流器。
func NewKeyedTokenBucketLimiter(capacity int64, refillRate float64) *KeyedRateLimiter {
	return NewKeyedRateLimiter(func() RateLimiter {
		return NewTokenBucket(capacity, refillRate)
	})
}

// NewKeyedLeakyBucketLimiter 创建以漏桶为底层实现的按 key 限流器。
func NewKeyedLeakyBucketLimiter(capacity int64, leakRate float64) *KeyedRateLimiter {
	return NewKeyedRateLimiter(func() RateLimiter {
		return NewLeakyBucket(capacity, leakRate)
	})
}

// Allow 检查指定 key 是否允许一个请求。
func (krl *KeyedRateLimiter) Allow(key string) bool {
	return krl.AllowN(key, 1)
}

// AllowN 检查指定 key 是否允许 n 个请求。
func (krl *KeyedRateLimiter) AllowN(key string, n int64) bool {
	return krl.getOrCreate(key).AllowN(n)
}

// getOrCreate 获取或创建指定 key 的限流器。
func (krl *KeyedRateLimiter) getOrCreate(key string) RateLimiter {
	krl.mutex.RLock()
	limiter, exists := krl.limiters[key]
	krl.mutex.RUnlock()

	if exists {
		return limiter
	}

	krl.mutex.Lock()
	defer krl.mutex.Unlock()

	// 双重检查，防止并发创建
	if limiter, exists = krl.limiters[key]; exists {
		return limiter
	}

	limiter = krl.factory()
	krl.limiters[key] = limiter
	return limiter
}

// Cleanup 清理空闲时间超过 maxIdleTime 的限流器以防止内存泄漏，
// 返回被清理的数量。
func (krl *KeyedRateLimiter) Cleanup(maxIdleTime time.Duration) int {
	krl.mutex.Lock()
	defer krl.mutex.Unlock()

	now := time.Now()
	removed := 0
	for key, limiter := range krl.limiters {
		if now.Sub(limiter.LastActivity()) > maxIdleTime {
			delete(krl.limiters, key)
			removed++
		}
	}
	return removed
}

// Len 返回当前活跃的 key 数量。
func (krl *KeyedRateLimiter) Len() int {
	krl.mutex.RLock()
	defer krl.mutex.RUnlock()
	return len(krl.limiters)
}

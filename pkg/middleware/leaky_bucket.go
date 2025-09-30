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

// LeakyBucket 实现漏桶算法
// Leaky bucket 以恒定速率处理请求，平滑流量
type LeakyBucket struct {
	capacity     int64      // 桶的最大容量（队列大小）
	leakRate     float64    // 每秒漏出的请求数
	water        float64    // 当前桶中的水量
	lastLeakTime time.Time  // 上次漏水的时间
	mutex        sync.Mutex // 保护并发访问
}

// NewLeakyBucket 创建一个新的漏桶
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

// Allow 尝试向桶中加入一个请求
// 如果桶未满返回 true，否则返回 false
func (lb *LeakyBucket) Allow() bool {
	return lb.AllowN(1)
}

// AllowN 尝试向桶中加入 n 个请求
func (lb *LeakyBucket) AllowN(n int64) bool {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// 先漏出已处理的请求
	lb.leak()

	// 检查是否有足够的空间
	if lb.water+float64(n) <= float64(lb.capacity) {
		lb.water += float64(n)
		return true
	}

	return false
}

// leak 根据时间流逝漏出请求
func (lb *LeakyBucket) leak() {
	now := time.Now()
	elapsed := now.Sub(lb.lastLeakTime).Seconds()

	// 计算漏出的请求数
	leaked := elapsed * lb.leakRate
	lb.water -= leaked

	// 确保不会小于0
	if lb.water < 0 {
		lb.water = 0
	}

	lb.lastLeakTime = now
}

// Water 返回当前桶中的水量
func (lb *LeakyBucket) Water() float64 {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.leak()
	return lb.water
}

// LeakyBucketLimiter 为多个客户端（如IP地址）提供漏桶速率限制
type LeakyBucketLimiter struct {
	capacity int64
	leakRate float64
	buckets  map[string]*LeakyBucket
	mutex    sync.RWMutex
}

// NewLeakyBucketLimiter 创建一个新的漏桶限制器
func NewLeakyBucketLimiter(capacity int64, leakRate float64) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{
		capacity: capacity,
		leakRate: leakRate,
		buckets:  make(map[string]*LeakyBucket),
	}
}

// Allow 检查指定客户端是否允许发送请求
func (lbl *LeakyBucketLimiter) Allow(clientID string) bool {
	return lbl.AllowN(clientID, 1)
}

// AllowN 检查指定客户端是否允许发送 n 个请求
func (lbl *LeakyBucketLimiter) AllowN(clientID string, n int64) bool {
	bucket := lbl.getOrCreateBucket(clientID)
	return bucket.AllowN(n)
}

// getOrCreateBucket 获取或创建指定客户端的漏桶
func (lbl *LeakyBucketLimiter) getOrCreateBucket(clientID string) *LeakyBucket {
	// 先尝试只读锁
	lbl.mutex.RLock()
	bucket, exists := lbl.buckets[clientID]
	lbl.mutex.RUnlock()

	if exists {
		return bucket
	}

	// 需要创建新桶，使用写锁
	lbl.mutex.Lock()
	defer lbl.mutex.Unlock()

	// 双重检查，防止并发创建
	bucket, exists = lbl.buckets[clientID]
	if exists {
		return bucket
	}

	bucket = NewLeakyBucket(lbl.capacity, lbl.leakRate)
	lbl.buckets[clientID] = bucket
	return bucket
}

// Cleanup 清理长时间未使用的桶以防止内存泄漏
func (lbl *LeakyBucketLimiter) Cleanup(maxIdleTime time.Duration) {
	lbl.mutex.Lock()
	defer lbl.mutex.Unlock()

	now := time.Now()
	for clientID, bucket := range lbl.buckets {
		bucket.mutex.Lock()
		if now.Sub(bucket.lastLeakTime) > maxIdleTime {
			delete(lbl.buckets, clientID)
		}
		bucket.mutex.Unlock()
	}
}

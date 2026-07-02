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
	"time"

	"github.com/innovationmech/swit/pkg/resilience"
)

// TokenBucket 是 resilience 包中令牌桶实现的别名。
// 限流核心算法统一维护在 pkg/resilience，服务端中间件与
// 客户端调用限流共用同一实现。
type TokenBucket = resilience.TokenBucket

// NewTokenBucket 创建一个新的令牌桶。
// capacity: 桶的最大容量（允许的突发请求数）
// refillRate: 每秒补充的令牌数（平均速率）
func NewTokenBucket(capacity int64, refillRate float64) *TokenBucket {
	return resilience.NewTokenBucket(capacity, refillRate)
}

// TokenBucketLimiter 为多个客户端（如 IP 地址）提供令牌桶速率限制。
// 底层复用 resilience.KeyedRateLimiter。
type TokenBucketLimiter struct {
	keyed *resilience.KeyedRateLimiter
}

// NewTokenBucketLimiter 创建一个新的令牌桶限制器
func NewTokenBucketLimiter(capacity int64, refillRate float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		keyed: resilience.NewKeyedTokenBucketLimiter(capacity, refillRate),
	}
}

// Allow 检查指定客户端是否允许发送请求
func (tbl *TokenBucketLimiter) Allow(clientID string) bool {
	return tbl.keyed.Allow(clientID)
}

// AllowN 检查指定客户端是否允许发送 n 个请求
func (tbl *TokenBucketLimiter) AllowN(clientID string, n int64) bool {
	return tbl.keyed.AllowN(clientID, n)
}

// Cleanup 清理长时间未使用的桶以防止内存泄漏
func (tbl *TokenBucketLimiter) Cleanup(maxIdleTime time.Duration) {
	tbl.keyed.Cleanup(maxIdleTime)
}

// Len 返回当前活跃的客户端桶数量
func (tbl *TokenBucketLimiter) Len() int {
	return tbl.keyed.Len()
}

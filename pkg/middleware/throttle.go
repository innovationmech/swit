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
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ThrottleStrategy 定义节流策略类型
type ThrottleStrategy string

const (
	// ThrottleStrategyTokenBucket 令牌桶策略 - 允许突发流量
	ThrottleStrategyTokenBucket ThrottleStrategy = "token_bucket"
	// ThrottleStrategyLeakyBucket 漏桶策略 - 平滑流量
	ThrottleStrategyLeakyBucket ThrottleStrategy = "leaky_bucket"
)

// ThrottleConfig 节流中间件配置
type ThrottleConfig struct {
	// Strategy 节流策略：token_bucket 或 leaky_bucket
	Strategy ThrottleStrategy `json:"strategy" yaml:"strategy"`

	// Capacity 桶容量
	// 对于 token bucket: 最大突发请求数
	// 对于 leaky bucket: 最大队列大小
	Capacity int64 `json:"capacity" yaml:"capacity"`

	// Rate 速率（每秒请求数）
	// 对于 token bucket: 令牌补充速率
	// 对于 leaky bucket: 请求处理速率
	Rate float64 `json:"rate" yaml:"rate"`

	// KeyFunc 提取客户端标识的函数（默认使用 IP）
	KeyFunc func(*gin.Context) string `json:"-" yaml:"-"`

	// CleanupInterval 清理未使用桶的间隔（默认 5 分钟）
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`

	// MaxIdleTime 桶最大空闲时间（默认 10 分钟）
	MaxIdleTime time.Duration `json:"max_idle_time" yaml:"max_idle_time"`

	// ErrorHandler 自定义错误处理函数
	ErrorHandler func(*gin.Context) `json:"-" yaml:"-"`
}

// Validate 验证配置
func (c *ThrottleConfig) Validate() error {
	if c.Strategy != ThrottleStrategyTokenBucket && c.Strategy != ThrottleStrategyLeakyBucket {
		return fmt.Errorf("invalid strategy: %s, must be token_bucket or leaky_bucket", c.Strategy)
	}
	if c.Capacity <= 0 {
		return fmt.Errorf("capacity must be positive, got %d", c.Capacity)
	}
	if c.Rate <= 0 {
		return fmt.Errorf("rate must be positive, got %f", c.Rate)
	}
	return nil
}

// DefaultThrottleConfig 返回默认节流配置（令牌桶，100 req/s，容量 200）
func DefaultThrottleConfig() ThrottleConfig {
	return ThrottleConfig{
		Strategy:        ThrottleStrategyTokenBucket,
		Capacity:        200,
		Rate:            100.0,
		KeyFunc:         defaultKeyFunc,
		CleanupInterval: 5 * time.Minute,
		MaxIdleTime:     10 * time.Minute,
		ErrorHandler:    defaultErrorHandler,
	}
}

// defaultKeyFunc 默认使用客户端 IP 作为键
func defaultKeyFunc(c *gin.Context) string {
	return c.ClientIP()
}

// defaultErrorHandler 默认错误处理函数
func defaultErrorHandler(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":   "rate limit exceeded",
		"message": "too many requests, please try again later",
	})
	c.Abort()
}

// Throttler 节流器接口
type Throttler interface {
	Allow(clientID string) bool
	Cleanup(maxIdleTime time.Duration)
}

// ThrottleMiddleware 速率限制和节流中间件
type ThrottleMiddleware struct {
	config    ThrottleConfig
	throttler Throttler
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewThrottleMiddleware 创建一个新的节流中间件
func NewThrottleMiddleware(config ThrottleConfig) (*ThrottleMiddleware, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid throttle config: %w", err)
	}

	// 设置默认值
	if config.KeyFunc == nil {
		config.KeyFunc = defaultKeyFunc
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	if config.MaxIdleTime == 0 {
		config.MaxIdleTime = 10 * time.Minute
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}

	// 根据策略创建相应的限流器
	var throttler Throttler
	switch config.Strategy {
	case ThrottleStrategyTokenBucket:
		throttler = NewTokenBucketLimiter(config.Capacity, config.Rate)
	case ThrottleStrategyLeakyBucket:
		throttler = NewLeakyBucketLimiter(config.Capacity, config.Rate)
	default:
		return nil, fmt.Errorf("unsupported strategy: %s", config.Strategy)
	}

	ctx, cancel := context.WithCancel(context.Background())
	tm := &ThrottleMiddleware{
		config:    config,
		throttler: throttler,
		ctx:       ctx,
		cancel:    cancel,
	}

	// 启动清理 goroutine
	go tm.cleanup()

	return tm, nil
}

// Handler 返回 Gin 中间件处理函数
func (tm *ThrottleMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := tm.config.KeyFunc(c)

		if !tm.throttler.Allow(clientID) {
			tm.config.ErrorHandler(c)
			return
		}

		c.Next()
	}
}

// cleanup 定期清理未使用的桶
func (tm *ThrottleMiddleware) cleanup() {
	ticker := time.NewTicker(tm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tm.ctx.Done():
			return
		case <-ticker.C:
			tm.throttler.Cleanup(tm.config.MaxIdleTime)
		}
	}
}

// Stop 停止中间件并清理资源
func (tm *ThrottleMiddleware) Stop() {
	tm.cancel()
}

// NewTokenBucketMiddleware 创建一个令牌桶中间件的快捷函数
func NewTokenBucketMiddleware(capacity int64, rate float64) (*ThrottleMiddleware, error) {
	config := DefaultThrottleConfig()
	config.Strategy = ThrottleStrategyTokenBucket
	config.Capacity = capacity
	config.Rate = rate
	return NewThrottleMiddleware(config)
}

// NewLeakyBucketMiddleware 创建一个漏桶中间件的快捷函数
func NewLeakyBucketMiddleware(capacity int64, rate float64) (*ThrottleMiddleware, error) {
	config := DefaultThrottleConfig()
	config.Strategy = ThrottleStrategyLeakyBucket
	config.Capacity = capacity
	config.Rate = rate
	return NewThrottleMiddleware(config)
}

// TokenBucketMiddleware 创建一个令牌桶中间件（使用默认配置）
func TokenBucketMiddleware() gin.HandlerFunc {
	tm, err := NewTokenBucketMiddleware(200, 100.0)
	if err != nil {
		panic(fmt.Sprintf("failed to create token bucket middleware: %v", err))
	}
	return tm.Handler()
}

// LeakyBucketMiddleware 创建一个漏桶中间件（使用默认配置）
func LeakyBucketMiddleware() gin.HandlerFunc {
	tm, err := NewLeakyBucketMiddleware(100, 50.0)
	if err != nil {
		panic(fmt.Sprintf("failed to create leaky bucket middleware: %v", err))
	}
	return tm.Handler()
}

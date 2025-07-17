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
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig holds the configuration for rate limiting
type RateLimitConfig struct {
	MaxRequests int           // Maximum number of requests allowed
	Window      time.Duration // Time window for rate limiting
	KeyFunc     func(*gin.Context) string // Function to extract the key for rate limiting
}

// RateLimiter represents a rate limiter instance
type RateLimiter struct {
	config     *RateLimitConfig
	requests   map[string]*requestInfo
	mutex      sync.RWMutex
	cleanupTicker *time.Ticker
}

// requestInfo tracks request count and window start time
type requestInfo struct {
	count     int
	windowStart time.Time
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:   config,
		requests: make(map[string]*requestInfo),
	}
	
	// Start cleanup goroutine to remove expired entries
	rl.cleanupTicker = time.NewTicker(config.Window)
	go rl.cleanup()
	
	return rl
}

// cleanup removes expired entries from the rate limiter map
func (rl *RateLimiter) cleanup() {
	for range rl.cleanupTicker.C {
		rl.mutex.Lock()
		now := time.Now()
		for key, info := range rl.requests {
			if now.Sub(info.windowStart) > rl.config.Window {
				delete(rl.requests, key)
			}
		}
		rl.mutex.Unlock()
	}
}

// Allow checks if a request should be allowed based on rate limiting
func (rl *RateLimiter) Allow(key string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	info, exists := rl.requests[key]
	
	if !exists {
		// First request from this key
		rl.requests[key] = &requestInfo{
			count:       1,
			windowStart: now,
		}
		return true
	}
	
	// Check if the window has expired
	if now.Sub(info.windowStart) > rl.config.Window {
		// Reset the window
		info.count = 1
		info.windowStart = now
		return true
	}
	
	// Check if we've exceeded the limit
	if info.count >= rl.config.MaxRequests {
		return false
	}
	
	// Increment the count
	info.count++
	return true
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
}

// RateLimitMiddleware creates a Gin middleware for rate limiting
func RateLimitMiddleware(config *RateLimitConfig) gin.HandlerFunc {
	limiter := NewRateLimiter(config)
	
	return func(c *gin.Context) {
		key := config.KeyFunc(c)
		
		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
				"message": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// GetClientIP extracts the client IP for rate limiting
func GetClientIP(c *gin.Context) string {
	// Try to get real IP from headers (for proxy scenarios)
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}
	return c.ClientIP()
}

// AuthRateLimitMiddleware creates a rate limiting middleware specifically for authentication endpoints
func AuthRateLimitMiddleware() gin.HandlerFunc {
	config := &RateLimitConfig{
		MaxRequests: 5,                  // 5 requests per minute
		Window:      time.Minute,        // 1 minute window
		KeyFunc:     GetClientIP,        // Rate limit by client IP
	}
	return RateLimitMiddleware(config)
}
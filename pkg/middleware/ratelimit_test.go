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
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)
	defer rl.Stop()

	assert.NotNil(t, rl)
	assert.Equal(t, 10, rl.limit)
	assert.Equal(t, time.Minute, rl.window)
	assert.NotNil(t, rl.requests)
	assert.NotNil(t, rl.ctx)
	assert.NotNil(t, rl.cancel)
	assert.NotNil(t, rl.done)
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(2, time.Second)
	defer rl.Stop()

	// First request should be allowed
	assert.True(t, rl.allow("192.168.1.1"))

	// Second request should be allowed
	assert.True(t, rl.allow("192.168.1.1"))

	// Third request should be denied (limit exceeded)
	assert.False(t, rl.allow("192.168.1.1"))

	// Different IP should be allowed
	assert.True(t, rl.allow("192.168.1.2"))
}

func TestRateLimiter_TimeWindow(t *testing.T) {
	rl := NewRateLimiter(1, 100*time.Millisecond)
	defer rl.Stop()

	// First request should be allowed
	assert.True(t, rl.allow("192.168.1.1"))

	// Second request should be denied (within time window)
	assert.False(t, rl.allow("192.168.1.1"))

	// Wait for time window to expire
	time.Sleep(150 * time.Millisecond)

	// Request should be allowed again
	assert.True(t, rl.allow("192.168.1.1"))
}

func TestRateLimiter_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter(2, time.Second)
	defer rl.Stop()

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request should succeed
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request should succeed
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12346"
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Third request should be rate limited
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.1:12347"
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
	assert.Contains(t, w3.Body.String(), "Rate limit exceeded")
}

func TestRateLimiter_Stop(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)

	// Verify the rate limiter is running
	assert.NotNil(t, rl.ctx)
	assert.NoError(t, rl.ctx.Err())

	// Stop the rate limiter
	rl.Stop()

	// Verify the context is cancelled
	assert.Error(t, rl.ctx.Err())

	// Verify the done channel is closed
	select {
	case <-rl.done:
		// Channel is closed, which is expected
	default:
		t.Error("done channel should be closed after Stop()")
	}
}

func TestRateLimiter_NoGoroutineLeak(t *testing.T) {
	// Get initial goroutine count
	initialGoroutines := runtime.NumGoroutine()

	// Create multiple rate limiters
	var rateLimiters []*RateLimiter
	for i := 0; i < 10; i++ {
		rl := NewRateLimiter(10, time.Minute)
		rateLimiters = append(rateLimiters, rl)
	}

	// Allow some time for goroutines to start
	time.Sleep(10 * time.Millisecond)

	// Stop all rate limiters
	for _, rl := range rateLimiters {
		rl.Stop()
	}

	// Allow some time for goroutines to clean up
	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	// Check that goroutine count is back to initial or close to it
	finalGoroutines := runtime.NumGoroutine()
	// Allow for some variance in goroutine count
	assert.LessOrEqual(t, finalGoroutines, initialGoroutines+2,
		"Goroutine leak detected: initial=%d, final=%d", initialGoroutines, finalGoroutines)
}

func TestRateLimiter_CleanupFunctionality(t *testing.T) {
	// Create a rate limiter with a very short time window for testing
	rl := NewRateLimiter(1, 50*time.Millisecond)
	defer rl.Stop()

	// Add some requests
	rl.allow("192.168.1.1")
	rl.allow("192.168.1.2")

	// Verify requests are stored
	rl.mutex.RLock()
	initialCount := len(rl.requests)
	rl.mutex.RUnlock()
	assert.Equal(t, 2, initialCount)

	// Wait for requests to expire
	time.Sleep(100 * time.Millisecond)

	// Make new requests to trigger cleanup logic in allow method
	// This should clean up expired entries for existing IPs
	rl.allow("192.168.1.1") // This should clean up old entry for this IP
	rl.allow("192.168.1.2") // This should clean up old entry for this IP
	rl.allow("192.168.1.3") // New IP

	// Check that we have the expected number of IPs
	rl.mutex.RLock()
	finalCount := len(rl.requests)
	rl.mutex.RUnlock()
	// Should have 3 IPs with fresh requests
	assert.Equal(t, 3, finalCount)

	// Verify that each IP has only recent requests
	rl.mutex.RLock()
	for ip, requests := range rl.requests {
		assert.LessOrEqual(t, len(requests), 1, "IP %s should have at most 1 recent request", ip)
	}
	rl.mutex.RUnlock()
}

func TestNewAuthRateLimiter(t *testing.T) {
	rl := NewAuthRateLimiter()
	defer rl.Stop()

	assert.NotNil(t, rl)
	assert.Equal(t, 5, rl.limit)
	assert.Equal(t, time.Minute, rl.window)
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(100, time.Second)
	defer rl.Stop()

	// Test concurrent access to the rate limiter
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				rl.allow("192.168.1.1")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify the rate limiter is still functional
	assert.False(t, rl.allow("192.168.1.1")) // Should be rate limited
}

func BenchmarkRateLimiter_Allow(b *testing.B) {
	rl := NewRateLimiter(1000, time.Minute)
	defer rl.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.allow("192.168.1.1")
	}
}

func BenchmarkRateLimiter_Middleware(b *testing.B) {
	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter(1000, time.Minute)
	defer rl.Stop()

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}

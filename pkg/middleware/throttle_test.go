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
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestThrottleConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ThrottleConfig
		wantErr bool
	}{
		{
			name: "有效的 token bucket 配置",
			config: ThrottleConfig{
				Strategy: ThrottleStrategyTokenBucket,
				Capacity: 100,
				Rate:     50.0,
			},
			wantErr: false,
		},
		{
			name: "有效的 leaky bucket 配置",
			config: ThrottleConfig{
				Strategy: ThrottleStrategyLeakyBucket,
				Capacity: 100,
				Rate:     50.0,
			},
			wantErr: false,
		},
		{
			name: "无效的策略",
			config: ThrottleConfig{
				Strategy: "invalid",
				Capacity: 100,
				Rate:     50.0,
			},
			wantErr: true,
		},
		{
			name: "无效的容量",
			config: ThrottleConfig{
				Strategy: ThrottleStrategyTokenBucket,
				Capacity: -1,
				Rate:     50.0,
			},
			wantErr: true,
		},
		{
			name: "无效的速率",
			config: ThrottleConfig{
				Strategy: ThrottleStrategyTokenBucket,
				Capacity: 100,
				Rate:     0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultThrottleConfig(t *testing.T) {
	config := DefaultThrottleConfig()
	assert.Equal(t, ThrottleStrategyTokenBucket, config.Strategy)
	assert.Equal(t, int64(200), config.Capacity)
	assert.Equal(t, 100.0, config.Rate)
	assert.NotNil(t, config.KeyFunc)
	assert.Equal(t, 5*time.Minute, config.CleanupInterval)
	assert.Equal(t, 10*time.Minute, config.MaxIdleTime)
	assert.NotNil(t, config.ErrorHandler)
}

func TestNewThrottleMiddleware(t *testing.T) {
	tests := []struct {
		name    string
		config  ThrottleConfig
		wantErr bool
	}{
		{
			name: "有效的 token bucket 中间件",
			config: ThrottleConfig{
				Strategy: ThrottleStrategyTokenBucket,
				Capacity: 100,
				Rate:     50.0,
			},
			wantErr: false,
		},
		{
			name: "有效的 leaky bucket 中间件",
			config: ThrottleConfig{
				Strategy: ThrottleStrategyLeakyBucket,
				Capacity: 100,
				Rate:     50.0,
			},
			wantErr: false,
		},
		{
			name: "无效配置",
			config: ThrottleConfig{
				Strategy: "invalid",
				Capacity: 100,
				Rate:     50.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, err := NewThrottleMiddleware(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tm)
				tm.Stop()
			}
		})
	}
}

func TestThrottleMiddleware_TokenBucket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := ThrottleConfig{
		Strategy: ThrottleStrategyTokenBucket,
		Capacity: 3,
		Rate:     1.0,
	}

	tm, err := NewThrottleMiddleware(config)
	assert.NoError(t, err)
	defer tm.Stop()

	router := gin.New()
	router.Use(tm.Handler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 前 3 个请求应该成功（容量为 3）
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should succeed", i)
	}

	// 第 4 个请求应该被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "rate limit exceeded")
}

func TestThrottleMiddleware_LeakyBucket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := ThrottleConfig{
		Strategy: ThrottleStrategyLeakyBucket,
		Capacity: 3,
		Rate:     1.0,
	}

	tm, err := NewThrottleMiddleware(config)
	assert.NoError(t, err)
	defer tm.Stop()

	router := gin.New()
	router.Use(tm.Handler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 前 3 个请求应该成功（容量为 3）
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should succeed", i)
	}

	// 第 4 个请求应该被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "rate limit exceeded")
}

func TestThrottleMiddleware_DifferentClients(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := ThrottleConfig{
		Strategy: ThrottleStrategyTokenBucket,
		Capacity: 2,
		Rate:     1.0,
	}

	tm, err := NewThrottleMiddleware(config)
	assert.NoError(t, err)
	defer tm.Stop()

	router := gin.New()
	router.Use(tm.Handler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 客户端 A 的前 2 个请求应该成功
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 客户端 A 的第 3 个请求应该被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// 客户端 B 的请求应该独立计数
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestThrottleMiddleware_CustomKeyFunc(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := ThrottleConfig{
		Strategy: ThrottleStrategyTokenBucket,
		Capacity: 2,
		Rate:     1.0,
		KeyFunc: func(c *gin.Context) string {
			// 使用 User-Agent 作为键
			return c.GetHeader("User-Agent")
		},
	}

	tm, err := NewThrottleMiddleware(config)
	assert.NoError(t, err)
	defer tm.Stop()

	router := gin.New()
	router.Use(tm.Handler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 同一 User-Agent 的前 2 个请求应该成功
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "test-agent")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 第 3 个请求应该被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// 不同 User-Agent 的请求应该独立计数
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "another-agent")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNewTokenBucketMiddleware(t *testing.T) {
	tm, err := NewTokenBucketMiddleware(100, 50.0)
	assert.NoError(t, err)
	assert.NotNil(t, tm)
	assert.Equal(t, ThrottleStrategyTokenBucket, tm.config.Strategy)
	assert.Equal(t, int64(100), tm.config.Capacity)
	assert.Equal(t, 50.0, tm.config.Rate)
	tm.Stop()
}

func TestNewLeakyBucketMiddleware(t *testing.T) {
	tm, err := NewLeakyBucketMiddleware(100, 50.0)
	assert.NoError(t, err)
	assert.NotNil(t, tm)
	assert.Equal(t, ThrottleStrategyLeakyBucket, tm.config.Strategy)
	assert.Equal(t, int64(100), tm.config.Capacity)
	assert.Equal(t, 50.0, tm.config.Rate)
	tm.Stop()
}

func TestTokenBucketMiddleware_DefaultConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TokenBucketMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 测试基本功能
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLeakyBucketMiddleware_DefaultConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(LeakyBucketMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 测试基本功能
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func BenchmarkThrottleMiddleware_TokenBucket(b *testing.B) {
	gin.SetMode(gin.TestMode)

	tm, _ := NewTokenBucketMiddleware(1000000, 1000000.0)
	defer tm.Stop()

	router := gin.New()
	router.Use(tm.Handler())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}

func BenchmarkThrottleMiddleware_LeakyBucket(b *testing.B) {
	gin.SetMode(gin.TestMode)

	tm, _ := NewLeakyBucketMiddleware(1000000, 1000000.0)
	defer tm.Stop()

	router := gin.New()
	router.Use(tm.Handler())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}

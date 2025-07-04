// Copyright © 2023 jackelyj <dreamerlyj@gmail.com>
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

package middleware

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewGlobalMiddlewareRegistrar(t *testing.T) {
	registrar := NewGlobalMiddlewareRegistrar()

	assert.NotNil(t, registrar)
	assert.IsType(t, &GlobalMiddlewareRegistrar{}, registrar)
}

func TestGlobalMiddlewareRegistrar_GetName(t *testing.T) {
	registrar := NewGlobalMiddlewareRegistrar()

	name := registrar.GetName()
	assert.Equal(t, "global-middleware", name)
}

func TestGlobalMiddlewareRegistrar_GetPriority(t *testing.T) {
	registrar := NewGlobalMiddlewareRegistrar()

	priority := registrar.GetPriority()
	assert.Equal(t, 1, priority)
}

func TestGlobalMiddlewareRegistrar_RegisterMiddleware(t *testing.T) {
	t.Run("successful_registration", func(t *testing.T) {
		router := gin.New()
		registrar := NewGlobalMiddlewareRegistrar()

		err := registrar.RegisterMiddleware(router)
		assert.NoError(t, err)

		// Verify that the router has middlewares registered
		assert.NotEmpty(t, router.Handlers)
	})
}

// TestTimeoutMiddleware_Success 测试正常请求（未超时）
func TestTimeoutMiddleware_Success(t *testing.T) {
	router := gin.New()
	router.Use(TimeoutMiddleware(1 * time.Second))

	router.GET("/test", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

// TestTimeoutMiddleware_Timeout 测试超时情况
func TestTimeoutMiddleware_Timeout(t *testing.T) {
	router := gin.New()
	router.Use(TimeoutMiddleware(100 * time.Millisecond))

	router.GET("/test", func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "should not see this"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusGatewayTimeout, w.Code)
	assert.Contains(t, w.Body.String(), "Request timeout")
}

// TestTimeoutMiddleware_ContextAware 测试处理器是否能感知context取消
func TestTimeoutMiddleware_ContextAware(t *testing.T) {
	router := gin.New()
	router.Use(TimeoutMiddleware(200 * time.Millisecond))

	handlerStarted := make(chan struct{})
	handlerStopped := make(chan struct{})

	router.GET("/test", func(c *gin.Context) {
		close(handlerStarted)
		ctx := c.Request.Context()

		// 模拟一个可以响应context取消的长时间操作
		select {
		case <-time.After(1 * time.Second):
			c.JSON(http.StatusOK, gin.H{"message": "completed"})
		case <-ctx.Done():
			// 正确响应了context取消
			close(handlerStopped)
			return
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// 等待确认handler已启动
	<-handlerStarted

	// 等待handler停止（最多等待500ms）
	select {
	case <-handlerStopped:
		// 好，handler正确响应了context取消
	case <-time.After(500 * time.Millisecond):
		t.Error("Handler did not respond to context cancellation")
	}

	assert.Equal(t, http.StatusGatewayTimeout, w.Code)
}

// TestTimeoutMiddleware_Panic 测试panic处理
func TestTimeoutMiddleware_Panic(t *testing.T) {
	router := gin.New()
	router.Use(TimeoutMiddleware(1 * time.Second))

	router.GET("/test", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	// 捕获panic，避免测试崩溃
	defer func() {
		if r := recover(); r != nil {
			// 预期会重新抛出panic
			assert.Equal(t, "test panic", r)
		}
	}()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestTimeoutMiddleware_Concurrent 测试并发请求安全性
func TestTimeoutMiddleware_Concurrent(t *testing.T) {
	router := gin.New()
	router.Use(TimeoutMiddleware(500 * time.Millisecond))

	requestCount := 50
	successCount := 0
	timeoutCount := 0
	var mu sync.Mutex

	router.GET("/test/:delay", func(c *gin.Context) {
		delay := c.Param("delay")
		if delay == "slow" {
			time.Sleep(600 * time.Millisecond)
		} else {
			time.Sleep(100 * time.Millisecond)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	var wg sync.WaitGroup
	wg.Add(requestCount)

	for i := 0; i < requestCount; i++ {
		go func(index int) {
			defer wg.Done()

			w := httptest.NewRecorder()
			path := "/test/fast"
			if index%2 == 0 {
				path = "/test/slow"
			}

			req, _ := http.NewRequest("GET", path, nil)
			router.ServeHTTP(w, req)

			mu.Lock()
			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusGatewayTimeout {
				timeoutCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// 验证结果 - 允许一些误差
	assert.Greater(t, successCount, requestCount/4, "应该有一些快速请求成功")
	assert.Greater(t, timeoutCount, requestCount/4, "应该有一些慢速请求超时")
	assert.Equal(t, requestCount, successCount+timeoutCount, "所有请求都应该得到响应")
}

// TestTimeoutMiddleware_WriteAfterTimeout 测试超时后的写入是否被忽略
func TestTimeoutMiddleware_WriteAfterTimeout(t *testing.T) {
	router := gin.New()
	router.Use(TimeoutMiddleware(100 * time.Millisecond))

	writeDone := make(chan struct{})

	router.GET("/test", func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond)

		// 尝试在超时后写入
		c.JSON(http.StatusOK, gin.H{"message": "late response"})
		close(writeDone)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// 等待handler完成写入尝试
	<-writeDone

	// 验证响应是超时错误，而不是handler的响应
	assert.Equal(t, http.StatusGatewayTimeout, w.Code)
	assert.Contains(t, w.Body.String(), "Request timeout")
	assert.NotContains(t, w.Body.String(), "late response")
}

// TestTimeoutMiddleware_HeadersPreserved 测试headers是否正确保留
func TestTimeoutMiddleware_HeadersPreserved(t *testing.T) {
	router := gin.New()
	router.Use(TimeoutMiddleware(1 * time.Second))

	router.GET("/test", func(c *gin.Context) {
		c.Header("X-Custom-Header", "test-value")
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-value", w.Header().Get("X-Custom-Header"))
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

// TestTimeoutMiddleware_NoGoroutineLeak 测试是否有goroutine泄露
func TestTimeoutMiddleware_NoGoroutineLeak(t *testing.T) {
	router := gin.New()
	router.Use(TimeoutMiddleware(100 * time.Millisecond))

	router.GET("/test", func(c *gin.Context) {
		// 模拟一个可以响应context取消的长时间操作
		select {
		case <-time.After(10 * time.Second):
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		case <-c.Request.Context().Done():
			// 正确响应了context取消
			return
		}
	})

	initialGoroutines := countGoroutines()

	// 发送多个会超时的请求
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusGatewayTimeout, w.Code)
	}

	// 等待一段时间让goroutine清理
	time.Sleep(500 * time.Millisecond)

	finalGoroutines := countGoroutines()

	// 允许有少量波动，但不应该有明显的泄露
	assert.LessOrEqual(t, finalGoroutines, initialGoroutines+5,
		"Goroutine count increased significantly, possible leak")
}

// TestTimeoutWriter_HeaderOperation 测试timeoutWriter的header操作安全性
func TestTimeoutWriter_HeaderOperation(t *testing.T) {
	router := gin.New()
	router.Use(TimeoutMiddleware(1 * time.Second))

	router.GET("/test", func(c *gin.Context) {
		// 正常的header操作
		c.Header("X-Custom-Header", "test-value")
		c.Header("X-Request-ID", "12345")
		c.Header("X-API-Version", "v1")

		// 测试重复设置相同header
		c.Header("X-Custom-Header", "updated-value")

		c.JSON(http.StatusOK, gin.H{"message": "header operation test"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "updated-value", w.Header().Get("X-Custom-Header"))
	assert.Equal(t, "12345", w.Header().Get("X-Request-ID"))
	assert.Equal(t, "v1", w.Header().Get("X-API-Version"))
}

// 辅助函数：统计当前goroutine数量
func countGoroutines() int {
	return runtime.NumGoroutine()
}

// BenchmarkTimeoutMiddleware 基准测试
func BenchmarkTimeoutMiddleware(b *testing.B) {
	router := gin.New()
	router.Use(TimeoutMiddleware(1 * time.Second))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "benchmark"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

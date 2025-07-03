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

func TestTimeoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		timeout        time.Duration
		handlerDelay   time.Duration
		expectedStatus int
		expectTimeout  bool
	}{
		{
			name:           "request_completes_within_timeout",
			timeout:        100 * time.Millisecond,
			handlerDelay:   50 * time.Millisecond,
			expectedStatus: http.StatusOK,
			expectTimeout:  false,
		},
		{
			name:           "request_exceeds_timeout",
			timeout:        50 * time.Millisecond,
			handlerDelay:   100 * time.Millisecond,
			expectedStatus: http.StatusRequestTimeout,
			expectTimeout:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(TimeoutMiddleware(tt.timeout))
			router.GET("/test", func(c *gin.Context) {
				time.Sleep(tt.handlerDelay)
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectTimeout {
				assert.Contains(t, w.Body.String(), "Request timeout")
			} else {
				assert.Contains(t, w.Body.String(), "success")
			}
		})
	}
}

func TestContextTimeoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		timeout        time.Duration
		handlerDelay   time.Duration
		expectedStatus int
		expectTimeout  bool
	}{
		{
			name:           "context_timeout_success",
			timeout:        100 * time.Millisecond,
			handlerDelay:   50 * time.Millisecond,
			expectedStatus: http.StatusOK,
			expectTimeout:  false,
		},
		{
			name:           "context_timeout_exceeded",
			timeout:        50 * time.Millisecond,
			handlerDelay:   100 * time.Millisecond,
			expectedStatus: http.StatusRequestTimeout,
			expectTimeout:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ContextTimeoutMiddleware(tt.timeout))
			router.GET("/test", func(c *gin.Context) {
				// Check if context has timeout
				ctx := c.Request.Context()
				if deadline, ok := ctx.Deadline(); ok {
					assert.True(t, deadline.After(time.Now()))
				}

				time.Sleep(tt.handlerDelay)
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectTimeout {
				assert.Contains(t, w.Body.String(), "Request timeout")
			} else {
				assert.Contains(t, w.Body.String(), "success")
			}
		})
	}
}

func TestTimeoutWithConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customHandler := func(c *gin.Context) {
		c.JSON(http.StatusRequestTimeout, gin.H{
			"error":   "Custom timeout",
			"message": "Custom timeout message",
		})
	}

	tests := []struct {
		name           string
		config         TimeoutConfig
		requestPath    string
		handlerDelay   time.Duration
		expectedStatus int
		expectTimeout  bool
		expectedMsg    string
	}{
		{
			name: "custom_timeout_config",
			config: TimeoutConfig{
				Timeout:      100 * time.Millisecond,
				ErrorMessage: "Custom timeout",
				Handler:      customHandler,
				SkipPaths:    []string{"/health"},
			},
			requestPath:    "/test",
			handlerDelay:   150 * time.Millisecond,
			expectedStatus: http.StatusRequestTimeout,
			expectTimeout:  true,
			expectedMsg:    "Custom timeout",
		},
		{
			name: "skip_path_no_timeout",
			config: TimeoutConfig{
				Timeout:      50 * time.Millisecond,
				ErrorMessage: "Custom timeout",
				Handler:      customHandler,
				SkipPaths:    []string{"/health"},
			},
			requestPath:    "/health",
			handlerDelay:   100 * time.Millisecond,
			expectedStatus: http.StatusOK,
			expectTimeout:  false,
			expectedMsg:    "success",
		},
		{
			name:   "default_config_used",
			config: TimeoutConfig{
				// Empty config should use defaults
			},
			requestPath:    "/test",
			handlerDelay:   100 * time.Millisecond,
			expectedStatus: http.StatusOK,
			expectTimeout:  false,
			expectedMsg:    "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(TimeoutWithConfig(tt.config))

			router.GET("/test", func(c *gin.Context) {
				time.Sleep(tt.handlerDelay)
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			router.GET("/health", func(c *gin.Context) {
				time.Sleep(tt.handlerDelay)
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req, _ := http.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedMsg)
		})
	}
}

func TestDefaultTimeoutConfig(t *testing.T) {
	config := DefaultTimeoutConfig()

	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, "Request timeout", config.ErrorMessage)
	assert.NotNil(t, config.Handler)
	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipPaths, "/metrics")
}

func TestTimeoutRegistrar(t *testing.T) {
	tests := []struct {
		name     string
		config   TimeoutConfig
		expected TimeoutConfig
	}{
		{
			name: "custom_config",
			config: TimeoutConfig{
				Timeout:      10 * time.Second,
				ErrorMessage: "Custom message",
			},
			expected: TimeoutConfig{
				Timeout:      10 * time.Second,
				ErrorMessage: "Custom message",
			},
		},
		{
			name:     "empty_config_uses_default",
			config:   TimeoutConfig{},
			expected: DefaultTimeoutConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrar := NewTimeoutRegistrar(tt.config)

			assert.Equal(t, "timeout-middleware", registrar.GetName())
			assert.Equal(t, 10, registrar.GetPriority())

			if tt.config.Timeout == 0 {
				assert.Equal(t, tt.expected.Timeout, registrar.config.Timeout)
				assert.Equal(t, tt.expected.ErrorMessage, registrar.config.ErrorMessage)
			} else {
				assert.Equal(t, tt.expected.Timeout, registrar.config.Timeout)
				assert.Equal(t, tt.expected.ErrorMessage, registrar.config.ErrorMessage)
			}

			// Test RegisterMiddleware doesn't panic
			router := gin.New()
			err := registrar.RegisterMiddleware(router)
			assert.NoError(t, err)
		})
	}
}

func TestTimeoutWithCustomHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customHandler := func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Service temporarily unavailable",
			"code":  503,
		})
	}

	router := gin.New()
	router.Use(TimeoutWithCustomHandler(50*time.Millisecond, customHandler))
	router.GET("/test", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "Service temporarily unavailable")
}

func BenchmarkTimeoutMiddleware(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TimeoutMiddleware(1 * time.Second))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkContextTimeoutMiddleware(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ContextTimeoutMiddleware(1 * time.Second))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

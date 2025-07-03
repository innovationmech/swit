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
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// timeoutWriter wraps gin.ResponseWriter to capture response during timeout
type timeoutWriter struct {
	gin.ResponseWriter
	h        http.Header
	wbuf     []byte
	code     int
	mu       sync.Mutex
	timedOut bool
}

func (tw *timeoutWriter) Header() http.Header {
	return tw.h
}

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}

	tw.wbuf = append(tw.wbuf, p...)
	return len(p), nil
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return
	}

	tw.code = code
}

// TimeoutMiddleware creates a middleware that sets a timeout for the entire request
// Note: This implementation focuses on setting context timeouts. Handlers should
// respect context cancellation for proper timeout behavior.
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return TimeoutWithCustomHandler(timeout, func(c *gin.Context) {
		c.JSON(http.StatusRequestTimeout, gin.H{
			"error":   "Request timeout",
			"message": "The request took too long to process",
			"code":    http.StatusRequestTimeout,
		})
	})
}

// timeoutHandler returns a handler that responds with a timeout error
func timeoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusRequestTimeout, gin.H{
			"error":   "Request timeout",
			"message": "The request took too long to process",
			"code":    http.StatusRequestTimeout,
		})
	}
}

// TimeoutWithCustomHandler creates a middleware with a custom timeout handler
func TimeoutWithCustomHandler(timeout time.Duration, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace the request context with the timeout context
		c.Request = c.Request.WithContext(ctx)

		// Use a buffered writer to capture the response
		tw := &timeoutWriter{
			ResponseWriter: c.Writer,
			h:              make(http.Header),
		}
		c.Writer = tw

		// Track timing
		start := time.Now()

		// Process the request normally
		c.Next()

		// Check if we exceeded the timeout
		elapsed := time.Since(start)
		if elapsed > timeout {
			// Timeout occurred - discard response and send timeout
			tw.mu.Lock()
			tw.timedOut = true
			tw.mu.Unlock()

			// Reset writer and send timeout response
			c.Writer = tw.ResponseWriter
			c.Abort()
			handler(c)
		} else {
			// Normal completion - copy buffered response
			tw.mu.Lock()
			defer tw.mu.Unlock()

			if !tw.timedOut {
				// Restore original writer
				c.Writer = tw.ResponseWriter

				// Copy headers
				for k, v := range tw.h {
					c.Writer.Header()[k] = v
				}

				// Write status code
				if tw.code != 0 {
					c.Writer.WriteHeader(tw.code)
				}

				// Write body
				if len(tw.wbuf) > 0 {
					c.Writer.Write(tw.wbuf)
				}
			}
		}
	}
}

// ContextTimeoutMiddleware creates a middleware that adds timeout to the request context
// This is useful when you want to pass the timeout context to downstream services
func ContextTimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return TimeoutWithCustomHandler(timeout, func(c *gin.Context) {
		c.JSON(http.StatusRequestTimeout, gin.H{
			"error":   "Request timeout",
			"message": "The request took too long to process",
			"code":    http.StatusRequestTimeout,
		})
	})
}

// TimeoutConfig holds configuration for timeout middleware
type TimeoutConfig struct {
	// Timeout duration for the request
	Timeout time.Duration
	// ErrorMessage custom error message for timeout
	ErrorMessage string
	// Handler custom handler for timeout responses
	Handler gin.HandlerFunc
	// SkipPaths paths to skip timeout middleware
	SkipPaths []string
}

// DefaultTimeoutConfig returns a default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout:      30 * time.Second,
		ErrorMessage: "Request timeout",
		Handler:      timeoutHandler(),
		SkipPaths:    []string{"/health", "/metrics"},
	}
}

// TimeoutWithConfig creates a timeout middleware with custom configuration
func TimeoutWithConfig(config TimeoutConfig) gin.HandlerFunc {
	// Use default config if not provided
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.ErrorMessage == "" {
		config.ErrorMessage = "Request timeout"
	}
	if config.Handler == nil {
		config.Handler = timeoutHandler()
	}

	return func(c *gin.Context) {
		// Skip timeout for certain paths
		for _, path := range config.SkipPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		// Apply timeout using our custom timeout middleware
		TimeoutWithCustomHandler(config.Timeout, config.Handler)(c)
	}
}

// TimeoutRegistrar is a middleware registrar for timeout middleware
type TimeoutRegistrar struct {
	config TimeoutConfig
}

// NewTimeoutRegistrar creates a new timeout middleware registrar
func NewTimeoutRegistrar(config TimeoutConfig) *TimeoutRegistrar {
	if config.Timeout == 0 {
		config = DefaultTimeoutConfig()
	}
	return &TimeoutRegistrar{
		config: config,
	}
}

// RegisterMiddleware implements the MiddlewareRegistrar interface
func (tr *TimeoutRegistrar) RegisterMiddleware(router *gin.Engine) error {
	router.Use(TimeoutWithConfig(tr.config))
	return nil
}

// GetName implements the MiddlewareRegistrar interface
func (tr *TimeoutRegistrar) GetName() string {
	return "timeout-middleware"
}

// GetPriority implements the MiddlewareRegistrar interface
func (tr *TimeoutRegistrar) GetPriority() int {
	return 10 // High priority, should be applied early
}

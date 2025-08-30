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
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

// RequestLoggerConfig holds configuration for request logging middleware
type RequestLoggerConfig struct {
	SkipPaths          []string // Paths to skip logging
	LogRequestBody     bool     // Whether to log request body
	LogResponseBody    bool     // Whether to log response body
	MaxBodyLogSize     int      // Maximum size of body to log (in bytes)
	IncludeQueryParams bool     // Whether to include query parameters in path
	SensitiveHeaders   []string // Headers to exclude from logging
}

// DefaultRequestLoggerConfig returns default configuration
func DefaultRequestLoggerConfig() *RequestLoggerConfig {
	return &RequestLoggerConfig{
		SkipPaths:          []string{"/health", "/ready", "/metrics"},
		LogRequestBody:     false,
		LogResponseBody:    false,
		MaxBodyLogSize:     1024,
		IncludeQueryParams: true,
		SensitiveHeaders:   []string{"Authorization", "X-Api-Key", "Cookie"},
	}
}

// RequestLogger is a middleware function for logging HTTP requests
func RequestLogger() gin.HandlerFunc {
	return RequestLoggerWithConfig(DefaultRequestLoggerConfig())
}

// RequestLoggerWithConfig creates a request logger middleware with custom configuration
func RequestLoggerWithConfig(config *RequestLoggerConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultRequestLoggerConfig()
	}

	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip logging for configured paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Generate request ID if not present
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Header("X-Request-ID", requestID)
		}

		// Set request ID in context for downstream logging
		c.Set(string(logger.ContextKeyRequestID), requestID)

		// Capture request body if configured
		var requestBody []byte
		if config.LogRequestBody && c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			if len(requestBody) > config.MaxBodyLogSize {
				requestBody = requestBody[:config.MaxBodyLogSize]
			}
		}

		// Capture response body if configured
		var responseWriter *responseBodyWriter
		if config.LogResponseBody {
			responseWriter = &responseBodyWriter{
				ResponseWriter: c.Writer,
				body:           &bytes.Buffer{},
				maxSize:        config.MaxBodyLogSize,
			}
			c.Writer = responseWriter
		}

		// Start time
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		latencyMS := float64(latency.Nanoseconds()) / 1e6

		// Collect request information
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		userAgent := c.Request.UserAgent()
		referer := c.Request.Referer()
		contentLength := c.Request.ContentLength

		// Build path with query parameters if configured
		if config.IncludeQueryParams && raw != "" {
			path = path + "?" + raw
		}

		// Build log fields
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Float64("latency_ms", latencyMS),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
			zap.String("host", c.Request.Host),
			zap.Int64("content_length", contentLength),
		}

		// Add optional fields
		if referer != "" {
			fields = append(fields, zap.String("referer", referer))
		}

		// Add request body if configured
		if config.LogRequestBody && len(requestBody) > 0 {
			fields = append(fields, zap.ByteString("request_body", requestBody))
		}

		// Add response body if configured
		if config.LogResponseBody && responseWriter != nil {
			fields = append(fields, zap.ByteString("response_body", responseWriter.body.Bytes()))
		}

		// Add response size
		fields = append(fields, zap.Int("response_size", c.Writer.Size()))

		// Add error if present
		if len(c.Errors) > 0 {
			errors := make([]string, len(c.Errors))
			for i, err := range c.Errors {
				errors[i] = err.Error()
			}
			fields = append(fields, zap.Strings("errors", errors))
		}

		// Log based on status code
		logger := logger.GetLogger()
		if statusCode >= 500 {
			logger.Error("HTTP request failed", fields...)
		} else if statusCode >= 400 {
			logger.Warn("HTTP request client error", fields...)
		} else {
			logger.Info("HTTP request completed", fields...)
		}
	}
}

// responseBodyWriter wraps gin.ResponseWriter to capture response body
type responseBodyWriter struct {
	gin.ResponseWriter
	body    *bytes.Buffer
	maxSize int
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	if w.body.Len() < w.maxSize {
		remaining := w.maxSize - w.body.Len()
		if len(b) > remaining {
			w.body.Write(b[:remaining])
		} else {
			w.body.Write(b)
		}
	}
	return w.ResponseWriter.Write(b)
}

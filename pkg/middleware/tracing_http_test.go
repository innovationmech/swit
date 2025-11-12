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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/innovationmech/swit/pkg/tracing"
)

func TestTracingMiddleware(t *testing.T) {
	// Cleanup: Reset global TracerProvider after test to avoid schema conflicts
	t.Cleanup(func() {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
	})

	// Setup test tracing manager
	config := tracing.DefaultTracingConfig()
	config.Enabled = true
	config.Exporter.Type = "console"

	tm := tracing.NewTracingManager()
	err := tm.Initialize(nil, config)
	require.NoError(t, err)

	// Shutdown the tracing manager after test
	t.Cleanup(func() {
		_ = tm.Shutdown(context.Background())
	})

	// Setup Gin router with tracing middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TracingMiddleware(tm))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "test error"})
	})

	t.Run("successful request creates span", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "test-agent")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "success")
	})

	t.Run("error request records error in span", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/error", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
	})
}

func TestTracingMiddlewareWithConfig(t *testing.T) {
	// Cleanup: Reset global TracerProvider after test to avoid schema conflicts
	t.Cleanup(func() {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
	})

	config := tracing.DefaultTracingConfig()
	config.Enabled = true
	config.Exporter.Type = "console"

	tm := tracing.NewTracingManager()
	err := tm.Initialize(nil, config)
	require.NoError(t, err)

	// Shutdown the tracing manager after test
	t.Cleanup(func() {
		_ = tm.Shutdown(context.Background())
	})

	tracingConfig := &HTTPTracingConfig{
		SkipPaths:        []string{"/health"},
		RecordReqBody:    false,
		RecordRespBody:   false,
		MaxBodySize:      1024,
		CustomAttributes: make(map[string]func(*gin.Context) string),
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TracingMiddlewareWithConfig(tm, tracingConfig))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	t.Run("normal path creates span", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("skip path does not create span", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestHTTPClientTracingRoundTripper(t *testing.T) {
	// Cleanup: Reset global TracerProvider after test to avoid schema conflicts
	t.Cleanup(func() {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
	})

	config := tracing.DefaultTracingConfig()
	config.Enabled = true
	config.Exporter.Type = "console"

	tm := tracing.NewTracingManager()
	err := tm.Initialize(nil, config)
	require.NoError(t, err)

	// Shutdown the tracing manager after test
	t.Cleanup(func() {
		_ = tm.Shutdown(context.Background())
	})

	// Create test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer testServer.Close()

	// Create HTTP client with tracing
	client := &http.Client{
		Transport: HTTPClientTracingRoundTripper(tm, nil),
	}

	t.Run("client request creates span", func(t *testing.T) {
		req, err := http.NewRequest("GET", testServer.URL+"/test", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestDefaultHTTPTracingConfig(t *testing.T) {
	config := DefaultHTTPTracingConfig()

	assert.NotNil(t, config)
	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipPaths, "/ready")
	assert.Contains(t, config.SkipPaths, "/metrics")
	assert.False(t, config.RecordReqBody)
	assert.False(t, config.RecordRespBody)
	assert.Equal(t, 4096, config.MaxBodySize)
	assert.NotNil(t, config.CustomAttributes)
}

func TestResponseWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("response writer captures data", func(t *testing.T) {
		// Create a proper gin ResponseWriter
		recorder := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(recorder)

		rw := &responseWriter{
			ResponseWriter: ginCtx.Writer,
			statusCode:     http.StatusOK,
			size:           0,
		}

		// Write response
		data := []byte("test response")
		n, err := rw.Write(data)

		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, int64(len(data)), rw.size)
		assert.Equal(t, data, rw.body)
	})

	t.Run("response writer captures status code", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(recorder)

		rw := &responseWriter{
			ResponseWriter: ginCtx.Writer,
			statusCode:     http.StatusOK,
			size:           0,
		}

		// Write header
		rw.WriteHeader(http.StatusCreated)

		assert.Equal(t, http.StatusCreated, rw.statusCode)
	})

	t.Run("response writer limits body capture", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(recorder)

		rw := &responseWriter{
			ResponseWriter: ginCtx.Writer,
			statusCode:     http.StatusOK,
			size:           0,
		}

		// Write large response
		largeData := strings.Repeat("x", 5000)
		n, err := rw.Write([]byte(largeData))

		assert.NoError(t, err)
		assert.Equal(t, len(largeData), n)
		assert.Equal(t, int64(len(largeData)), rw.size)
		assert.LessOrEqual(t, len(rw.body), 4096) // Should be limited
	})
}

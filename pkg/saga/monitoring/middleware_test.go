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

package monitoring

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestDefaultMiddlewareConfig(t *testing.T) {
	config := DefaultMiddlewareConfig()

	if config == nil {
		t.Fatal("DefaultMiddlewareConfig() returned nil")
	}

	if !config.EnableLogging {
		t.Error("EnableLogging should be true by default")
	}
	if !config.EnableCORS {
		t.Error("EnableCORS should be true by default")
	}
	if !config.EnableErrorHandle {
		t.Error("EnableErrorHandle should be true by default")
	}
	if !config.EnableRecovery {
		t.Error("EnableRecovery should be true by default")
	}
	if config.CORSConfig == nil {
		t.Error("CORSConfig should not be nil by default")
	}
}

func TestDefaultCORSConfig(t *testing.T) {
	config := DefaultCORSConfig()

	if config == nil {
		t.Fatal("DefaultCORSConfig() returned nil")
	}

	if len(config.AllowOrigins) == 0 {
		t.Error("AllowOrigins should not be empty")
	}
	if len(config.AllowMethods) == 0 {
		t.Error("AllowMethods should not be empty")
	}
	if len(config.AllowHeaders) == 0 {
		t.Error("AllowHeaders should not be empty")
	}
	if config.MaxAge != 12*time.Hour {
		t.Errorf("MaxAge should be 12 hours, got %v", config.MaxAge)
	}
}

func TestNewMonitoringMiddleware(t *testing.T) {
	tests := []struct {
		name   string
		config *MiddlewareConfig
	}{
		{
			name:   "with nil config",
			config: nil,
		},
		{
			name:   "with default config",
			config: DefaultMiddlewareConfig(),
		},
		{
			name: "with custom config",
			config: &MiddlewareConfig{
				EnableLogging:     false,
				EnableCORS:        true,
				EnableErrorHandle: false,
				EnableRecovery:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewMonitoringMiddleware(tt.config)
			if middleware == nil {
				t.Fatal("NewMonitoringMiddleware() returned nil")
			}
			if middleware.config == nil {
				t.Error("Middleware config is nil")
			}
		})
	}
}

func TestMiddlewareGetConfig(t *testing.T) {
	config := DefaultMiddlewareConfig()
	middleware := NewMonitoringMiddleware(config)

	gotConfig := middleware.GetConfig()
	if gotConfig == nil {
		t.Error("GetConfig() returned nil")
	}
	if gotConfig != middleware.config {
		t.Error("GetConfig() should return the same config instance")
	}
}

// Note: TestSanitizeForLog already exists in server_test.go

func TestLoggingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	middleware := NewMonitoringMiddleware(&MiddlewareConfig{
		EnableLogging: true,
	})
	middleware.ApplyToRouter(router)

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test?param=value", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	middleware := NewMonitoringMiddleware(&MiddlewareConfig{
		EnableCORS: true,
		CORSConfig: DefaultCORSConfig(),
	})
	middleware.ApplyToRouter(router)

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// CORS middleware should add appropriate headers
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("CORS headers should be present")
	}
}

func TestErrorHandlingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	middleware := NewMonitoringMiddleware(&MiddlewareConfig{
		EnableErrorHandle: true,
	})
	middleware.ApplyToRouter(router)

	router.GET("/error", func(c *gin.Context) {
		_ = c.Error(errors.New("test error"))
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "error") {
		t.Error("Response should contain error message")
	}
}

func TestErrorHandlingMiddlewareBindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	middleware := NewMonitoringMiddleware(&MiddlewareConfig{
		EnableErrorHandle: true,
	})
	middleware.ApplyToRouter(router)

	type RequestBody struct {
		Value int `json:"value" binding:"required"`
	}

	router.POST("/bind", func(c *gin.Context) {
		var body RequestBody
		if err := c.ShouldBindJSON(&body); err != nil {
			_ = c.Error(err).SetType(gin.ErrorTypeBind)
			return
		}
		c.JSON(http.StatusOK, body)
	})

	req := httptest.NewRequest(http.MethodPost, "/bind", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for bind error, got %d", w.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	middleware := NewMonitoringMiddleware(&MiddlewareConfig{
		EnableRecovery: true,
	})
	middleware.ApplyToRouter(router)

	router.GET("/panic", func(c *gin.Context) {
		panic("intentional panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Internal server error") {
		t.Error("Response should contain internal server error message")
	}
}

func TestApplyToRouterWithAllMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	middleware := NewMonitoringMiddleware(DefaultMiddlewareConfig())
	middleware.ApplyToRouter(router)

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestApplyToRouterWithSelectiveMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	middleware := NewMonitoringMiddleware(&MiddlewareConfig{
		EnableLogging:     true,
		EnableCORS:        false,
		EnableErrorHandle: false,
		EnableRecovery:    false,
	})
	middleware.ApplyToRouter(router)

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

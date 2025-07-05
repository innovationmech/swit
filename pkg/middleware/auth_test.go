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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_WhiteList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "healthy"})
	})
	router.GET("/stop", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "stopped"})
	})

	tests := []struct {
		name string
		path string
	}{
		{"health_endpoint", "/health"},
		{"stop_endpoint", "/stop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "authentication header required")
}

func TestAuthMiddleware_InvalidAuthorizationHeaderFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		authHeader    string
		expectedError string
	}{
		{
			name:          "missing_bearer_prefix",
			authHeader:    "token123",
			expectedError: "invalid authentication header format",
		},
		{
			name:          "wrong_prefix",
			authHeader:    "Basic token123",
			expectedError: "invalid authentication header format",
		},
		{
			name:          "empty_token",
			authHeader:    "Bearer",
			expectedError: "invalid authentication header format",
		},
		{
			name:          "too_many_parts",
			authHeader:    "Bearer token123 extra",
			expectedError: "invalid authentication header format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware())
			router.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req, _ := http.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", tt.authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedError)
		})
	}
}

func TestAuthMiddlewareWithWhiteList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customWhiteList := []string{"/users/create", "/public"}
	router := gin.New()
	router.Use(AuthMiddlewareWithWhiteList(customWhiteList))
	router.GET("/users/create", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "user created"})
	})
	router.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "public"})
	})

	tests := []struct {
		name string
		path string
	}{
		{"users_create_endpoint", "/users/create"},
		{"public_endpoint", "/public"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestAuthMiddlewareWithConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &AuthConfig{
		WhiteList:       []string{"/custom/health"},
		AuthServiceName: "custom-auth",
		AuthEndpoint:    "/custom/validate",
	}

	router := gin.New()
	router.Use(AuthMiddlewareWithConfig(config))
	router.GET("/custom/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "custom health"})
	})

	req, _ := http.NewRequest("GET", "/custom/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIsWhitelisted(t *testing.T) {
	whiteList := []string{"/health", "/stop", "/users/create"}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"health_path", "/health", true},
		{"stop_path", "/stop", true},
		{"users_create_path", "/users/create", true},
		{"non_whitelisted_path", "/protected", false},
		{"partial_match", "/heal", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWhitelisted(tt.path, whiteList)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		expectedToken string
		expectError   bool
	}{
		{
			name:          "valid_bearer_token",
			authHeader:    "Bearer token123",
			expectedToken: "token123",
			expectError:   false,
		},
		{
			name:        "empty_header",
			authHeader:  "",
			expectError: true,
		},
		{
			name:        "invalid_format",
			authHeader:  "token123",
			expectError: true,
		},
		{
			name:        "wrong_prefix",
			authHeader:  "Basic token123",
			expectError: true,
		},
		{
			name:        "missing_token",
			authHeader:  "Bearer",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := extractTokenFromHeader(tt.authHeader)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}

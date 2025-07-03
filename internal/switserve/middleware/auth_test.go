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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_WhitelistedPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "users_create_whitelisted",
			path:           "/users/create",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "health_whitelisted",
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "stop_whitelisted",
			path:           "/stop",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware())
			router.Any(tt.path, func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req, _ := http.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), "success")
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
	assert.Contains(t, w.Body.String(), "Authentication header required")
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
			expectedError: "Invalid authentication header format",
		},
		{
			name:          "wrong_prefix",
			authHeader:    "Basic token123",
			expectedError: "Invalid authentication header format",
		},
		{
			name:          "empty_token",
			authHeader:    "Bearer",
			expectedError: "Invalid authentication header format",
		},
		{
			name:          "too_many_parts",
			authHeader:    "Bearer token123 extra",
			expectedError: "Invalid authentication header format",
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

func TestAuthMiddleware_ServiceDiscoveryError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Skip("Skipping test that requires config file - this test will fail because service discovery initialization requires actual config")
}

func TestWhiteList(t *testing.T) {
	expectedPaths := []string{
		"/users/create",
		"/health",
		"/stop",
	}

	assert.Equal(t, expectedPaths, whiteList)
}

func TestWhiteListContains(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "path_in_whitelist_users_create",
			path:     "/users/create",
			expected: true,
		},
		{
			name:     "path_in_whitelist_health",
			path:     "/health",
			expected: true,
		},
		{
			name:     "path_in_whitelist_stop",
			path:     "/stop",
			expected: true,
		},
		{
			name:     "path_not_in_whitelist",
			path:     "/api/users",
			expected: false,
		},
		{
			name:     "partial_match_not_allowed",
			path:     "/health/check",
			expected: false,
		},
		{
			name:     "empty_path",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, whitelistedPath := range whiteList {
				if tt.path == whitelistedPath {
					found = true
					break
				}
			}
			assert.Equal(t, tt.expected, found)
		})
	}
}

func TestAuthMiddleware_PathMatching(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectAuth     bool
	}{
		{
			name:           "exact_whitelist_match",
			path:           "/health",
			expectedStatus: http.StatusOK,
			expectAuth:     false,
		},
		{
			name:           "partial_match_requires_auth",
			path:           "/health/detailed",
			expectedStatus: http.StatusUnauthorized,
			expectAuth:     true,
		},
		{
			name:           "case_sensitive_match",
			path:           "/HEALTH",
			expectedStatus: http.StatusUnauthorized,
			expectAuth:     true,
		},
		{
			name:           "trailing_slash_different",
			path:           "/health/",
			expectedStatus: http.StatusUnauthorized,
			expectAuth:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware())
			router.Any(tt.path, func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req, _ := http.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectAuth {
				assert.Contains(t, w.Body.String(), "Authentication header required")
			} else {
				assert.Contains(t, w.Body.String(), "success")
			}
		})
	}
}

func BenchmarkAuthMiddleware_WhitelistedPath(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkAuthMiddleware_InvalidHeader(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkAuthMiddleware_HeaderParsing(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer sometoken")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

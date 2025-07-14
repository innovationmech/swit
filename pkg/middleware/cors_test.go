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

func TestCORSMiddleware(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		origin          string
		method          string
		expectedOrigin  string
		expectedMethods string
		expectedHeaders string
		expectedCreds   string
	}{
		{
			name:            "Valid origin",
			origin:          "http://localhost:3000",
			method:          "GET",
			expectedOrigin:  "http://localhost:3000",
			expectedMethods: "GET,POST,PUT,PATCH,DELETE,HEAD,OPTIONS",
			expectedHeaders: "Origin,Content-Length,Content-Type,Authorization",
			expectedCreds:   "true",
		},
		{
			name:            "Invalid origin",
			origin:          "http://malicious.com",
			method:          "GET",
			expectedOrigin:  "",
			expectedMethods: "",
			expectedHeaders: "",
			expectedCreds:   "",
		},
		{
			name:            "OPTIONS preflight request",
			origin:          "http://localhost:3000",
			method:          "OPTIONS",
			expectedOrigin:  "http://localhost:3000",
			expectedMethods: "GET,POST,PUT,PATCH,DELETE,HEAD,OPTIONS",
			expectedHeaders: "Origin,Content-Length,Content-Type,Authorization",
			expectedCreds:   "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new Gin router with CORS middleware
			router := gin.New()
			router.Use(CORSMiddleware())

			// Add a simple test route
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Create a test request
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			// Create a response recorder
			w := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(w, req)

			// Check CORS headers
			if tt.expectedOrigin != "" {
				assert.Equal(t, tt.expectedOrigin, w.Header().Get("Access-Control-Allow-Origin"))
			} else {
				assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
			}

			// Allow-Methods and Allow-Headers are mainly for preflight OPTIONS requests
			// For simple requests, these headers may not be present
			if tt.method == "OPTIONS" {
				assert.Equal(t, tt.expectedMethods, w.Header().Get("Access-Control-Allow-Methods"))
				assert.Equal(t, tt.expectedHeaders, w.Header().Get("Access-Control-Allow-Headers"))
			}

			if tt.expectedCreds != "" && tt.origin != "" {
				assert.Equal(t, tt.expectedCreds, w.Header().Get("Access-Control-Allow-Credentials"))
			}
		})
	}
}

func TestCORSMiddlewareWithDifferentRoutes(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new Gin router with CORS middleware
	router := gin.New()
	router.Use(CORSMiddleware())

	// Add various test routes
	router.GET("/api/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": []string{"user1", "user2"}})
	})

	router.POST("/api/users", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"id": 1, "name": "new user"})
	})

	router.PUT("/api/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id"), "updated": true})
	})

	router.DELETE("/api/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id"), "deleted": true})
	})

	tests := []struct {
		name   string
		method string
		path   string
		origin string
	}{
		{
			name:   "GET request",
			method: "GET",
			path:   "/api/users",
			origin: "http://localhost:3000",
		},
		{
			name:   "POST request",
			method: "POST",
			path:   "/api/users",
			origin: "http://localhost:3000",
		},
		{
			name:   "PUT request",
			method: "PUT",
			path:   "/api/users/1",
			origin: "http://localhost:3000",
		},
		{
			name:   "DELETE request",
			method: "DELETE",
			path:   "/api/users/1",
			origin: "http://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Origin", tt.origin)

			// Create a response recorder
			w := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(w, req)

			// Check CORS headers are present for all methods
			assert.Equal(t, tt.origin, w.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))

			// Allow-Methods and Allow-Headers are mainly for preflight OPTIONS requests
			// For simple requests, these headers may not be present
		})
	}
}

func TestCORSPreflightRequest(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new Gin router with CORS middleware
	router := gin.New()
	router.Use(CORSMiddleware())

	// Add a test route that supports OPTIONS
	router.OPTIONS("/api/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// Create a preflight OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")

	// Create a response recorder
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Check preflight response headers
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET,POST,PUT,PATCH,DELETE,HEAD,OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Origin,Content-Length,Content-Type,Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSWithoutOrigin(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new Gin router with CORS middleware
	router := gin.New()
	router.Use(CORSMiddleware())

	// Add a simple test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Create a test request without Origin header
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Check that CORS headers are not set when no origin is provided
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCORSConfigurationValues(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new Gin router with CORS middleware
	router := gin.New()
	router.Use(CORSMiddleware())

	// Add a simple test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test with allowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify specific configuration values
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

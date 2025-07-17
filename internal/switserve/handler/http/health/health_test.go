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

package health

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthHandler(t *testing.T) {
	// Set gin to test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:           "successful health check",
			method:         "GET",
			path:           "/health",
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]interface{}{"message": "pong"},
		},
		{
			name:           "health check with wrong method",
			method:         "POST",
			path:           "/health",
			expectedStatus: http.StatusNotFound,
			expectedBody:   nil,
		},
		{
			name:           "health check with wrong path",
			method:         "GET",
			path:           "/health/invalid",
			expectedStatus: http.StatusNotFound,
			expectedBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new gin router
			router := gin.New()
			router.GET("/health", Handler)

			// Create a new HTTP request
			req, err := http.NewRequest(tt.method, tt.path, nil)
			assert.NoError(t, err)

			// Create a response recorder
			w := httptest.NewRecorder()

			// Perform the request
			router.ServeHTTP(w, req)

			// Check the status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check the response body for successful health check
			if tt.expectedBody != nil {
				assert.JSONEq(t, `{"message":"pong"}`, w.Body.String())
			}
		})
	}
}

func TestHealthRouteRegistration(t *testing.T) {
	// Set gin to test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedRoutes []string
	}{
		{
			name:           "register health routes",
			expectedRoutes: []string{"GET:/health"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new gin router
			router := gin.New()

			// Register the routes
			router.GET("/health", Handler)

			// Check if the routes are registered correctly
			for _, route := range tt.expectedRoutes {
				method, path := parseRoute(route)
				found := false

				// Iterate through the registered routes
				for _, r := range router.Routes() {
					if r.Method == method && r.Path == path {
						found = true
						break
					}
				}

				assert.True(t, found, "Route %s %s not found", method, path)
			}
		})
	}
}

// TestHealthHandlerConcurrent tests the health handler under concurrent access
func TestHealthHandlerConcurrent(t *testing.T) {
	// Set gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new gin router
	router := gin.New()
	router.GET("/health", Handler)

	// Number of concurrent requests
	const concurrentRequests = 100

	// Channel to collect results
	results := make(chan bool, concurrentRequests)

	// Launch concurrent requests
	for i := 0; i < concurrentRequests; i++ {
		go func() {
			req, _ := http.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check if the response is correct
			results <- w.Code == http.StatusOK && w.Body.String() == `{"message":"pong"}`
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < concurrentRequests; i++ {
		if <-results {
			successCount++
		}
	}

	// All requests should succeed
	assert.Equal(t, concurrentRequests, successCount)
}

// TestHealthHandlerWithMiddleware tests the health handler with middleware
func TestHealthHandlerWithMiddleware(t *testing.T) {
	// Set gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new gin router with a simple middleware
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Header("X-Test-Middleware", "applied")
		c.Next()
	})

	// Register the routes
	router.GET("/health", Handler)

	// Create a new HTTP request
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the status code
	assert.Equal(t, http.StatusOK, w.Code)

	// Check the response body
	assert.JSONEq(t, `{"message":"pong"}`, w.Body.String())

	// Check if middleware was applied
	assert.Equal(t, "applied", w.Header().Get("X-Test-Middleware"))
}

// Helper function to parse route string
func parseRoute(route string) (method, path string) {
	parts := []string{}
	for i, c := range route {
		if c == ':' {
			parts = append(parts, route[:i], route[i+1:])
			break
		}
	}
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

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

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/pkg/logger"
)

func init() {
	logger.Logger, _ = zap.NewDevelopment()
	gin.SetMode(gin.TestMode)

	// Change to project root for config file access
	wd, _ := os.Getwd()
	if filepath.Base(wd) == "handler" {
		os.Chdir("../../..")
	}
}

func TestNewHealthRouteRegistrar(t *testing.T) {
	registrar := NewHealthRouteRegistrar()

	assert.NotNil(t, registrar)
	assert.IsType(t, &HealthRouteRegistrar{}, registrar)
}

func TestHealthRouteRegistrar_GetName(t *testing.T) {
	registrar := NewHealthRouteRegistrar()

	assert.Equal(t, "health-check", registrar.GetName())
}

func TestHealthRouteRegistrar_GetVersion(t *testing.T) {
	registrar := NewHealthRouteRegistrar()

	assert.Equal(t, "root", registrar.GetVersion())
}

func TestHealthRouteRegistrar_GetPrefix(t *testing.T) {
	registrar := NewHealthRouteRegistrar()

	assert.Equal(t, "", registrar.GetPrefix())
}

func TestHealthRouteRegistrar_RegisterRoutes(t *testing.T) {
	tests := []struct {
		name           string
		expectedRoutes []string
		expectedMethod string
	}{
		{
			name:           "successful route registration",
			expectedRoutes: []string{"/health"},
			expectedMethod: "GET",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test router
			router := gin.New()
			routerGroup := router.Group("")

			// Create registrar
			registrar := NewHealthRouteRegistrar()

			// Register routes
			err := registrar.RegisterRoutes(routerGroup)
			assert.NoError(t, err)

			// Verify routes are registered
			routes := router.Routes()
			require.Len(t, routes, 1, "Expected exactly one route to be registered")

			route := routes[0]
			assert.Equal(t, tt.expectedRoutes[0], route.Path)
			assert.Equal(t, tt.expectedMethod, route.Method)
		})
	}
}

func TestHealthRouteRegistrar_RegisterRoutes_HTTPMethod(t *testing.T) {
	registrar := NewHealthRouteRegistrar()

	// Create test router
	router := gin.New()
	routerGroup := router.Group("")

	// Register routes
	err := registrar.RegisterRoutes(routerGroup)
	require.NoError(t, err)

	// Verify HTTP method for health route
	routes := router.Routes()
	require.Len(t, routes, 1)

	route := routes[0]
	assert.Equal(t, "/health", route.Path)
	assert.Equal(t, "GET", route.Method)
}

func TestHealthRouteRegistrar_IntegrationTest(t *testing.T) {
	registrar := NewHealthRouteRegistrar()

	// Create test router and register routes
	router := gin.New()
	routerGroup := router.Group("")
	err := registrar.RegisterRoutes(routerGroup)
	require.NoError(t, err)

	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Verify response body
	var response model.HealthResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "pong", response.Message)
}

func TestHealthRouteRegistrar_RouteGrouping(t *testing.T) {
	registrar := NewHealthRouteRegistrar()

	// Create test router with a parent group
	router := gin.New()
	parentGroup := router.Group("/api/v1")

	// Register routes under parent group
	err := registrar.RegisterRoutes(parentGroup)
	require.NoError(t, err)

	// Verify route is prefixed correctly
	routes := router.Routes()
	require.Len(t, routes, 1)

	route := routes[0]
	assert.Equal(t, "/api/v1/health", route.Path)
	assert.Equal(t, "GET", route.Method)

	// Test the actual endpoint
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.HealthResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "pong", response.Message)
}

func TestHealthRouteRegistrar_MultipleRegistrations(t *testing.T) {
	registrar := NewHealthRouteRegistrar()

	// Create test router
	router := gin.New()
	routerGroup1 := router.Group("/v1")
	routerGroup2 := router.Group("/v2")

	// Register routes in multiple groups
	err1 := registrar.RegisterRoutes(routerGroup1)
	err2 := registrar.RegisterRoutes(routerGroup2)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Verify both sets of routes exist
	routes := router.Routes()
	assert.Len(t, routes, 2)

	routePaths := make([]string, len(routes))
	for i, route := range routes {
		routePaths[i] = route.Path
	}

	// Check both health routes exist
	assert.Contains(t, routePaths, "/v1/health")
	assert.Contains(t, routePaths, "/v2/health")

	// Test both endpoints
	req1 := httptest.NewRequest("GET", "/v1/health", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	req2 := httptest.NewRequest("GET", "/v2/health", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestHealthRouteRegistrar_Interface(t *testing.T) {
	registrar := NewHealthRouteRegistrar()

	// Verify it implements the RouteRegistrar interface by testing all methods
	assert.Equal(t, "health-check", registrar.GetName())
	assert.Equal(t, "root", registrar.GetVersion())
	assert.Equal(t, "", registrar.GetPrefix())

	// Test RegisterRoutes returns no error
	router := gin.New()
	routerGroup := router.Group("")
	err := registrar.RegisterRoutes(routerGroup)
	assert.NoError(t, err)
}

func TestHealthRouteRegistrar_ResponseFormat(t *testing.T) {
	registrar := NewHealthRouteRegistrar()

	// Create test router and register routes
	router := gin.New()
	routerGroup := router.Group("")
	err := registrar.RegisterRoutes(routerGroup)
	require.NoError(t, err)

	// Test health endpoint response format
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify JSON response structure
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check response contains expected fields
	assert.Contains(t, response, "message")
	assert.Equal(t, "pong", response["message"])
}

func TestHealthRouteRegistrar_HTTPMethods(t *testing.T) {
	registrar := NewHealthRouteRegistrar()
	router := gin.New()
	routerGroup := router.Group("")

	err := registrar.RegisterRoutes(routerGroup)
	require.NoError(t, err)

	// Test only GET method is allowed (others return 404 in Gin)
	testCases := []struct {
		method         string
		expectedStatus int
	}{
		{"GET", http.StatusOK},
		{"POST", http.StatusNotFound}, // Gin returns 404 for non-registered routes
		{"PUT", http.StatusNotFound},
		{"DELETE", http.StatusNotFound},
		{"PATCH", http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/health", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestHealthRouteRegistrar_ConcurrentAccess(t *testing.T) {
	registrar := NewHealthRouteRegistrar()
	router := gin.New()
	routerGroup := router.Group("")

	err := registrar.RegisterRoutes(routerGroup)
	require.NoError(t, err)

	// Test concurrent access to health endpoint
	const numRequests = 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

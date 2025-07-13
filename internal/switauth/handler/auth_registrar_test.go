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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

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

// MockAuthService is already defined in login_test.go, so we'll use that existing one

// createTestAuthController creates a test auth controller with mock service
func createTestAuthController() *AuthController {
	mockService := &MockAuthService{}
	return NewAuthController(mockService)
}

func TestNewAuthRouteRegistrar(t *testing.T) {
	tests := []struct {
		name       string
		controller *AuthController
		expectNil  bool
	}{
		{
			name:       "successful creation with valid controller",
			controller: createTestAuthController(),
			expectNil:  false,
		},
		{
			name:       "creation with nil controller",
			controller: nil,
			expectNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrar := NewAuthRouteRegistrar(tt.controller)

			if tt.expectNil {
				assert.Nil(t, registrar)
			} else {
				assert.NotNil(t, registrar)
				assert.Equal(t, tt.controller, registrar.controller)
			}
		})
	}
}

func TestAuthRouteRegistrar_GetName(t *testing.T) {
	controller := createTestAuthController()
	registrar := NewAuthRouteRegistrar(controller)

	assert.Equal(t, "auth-api", registrar.GetName())
}

func TestAuthRouteRegistrar_GetVersion(t *testing.T) {
	controller := createTestAuthController()
	registrar := NewAuthRouteRegistrar(controller)

	assert.Equal(t, "root", registrar.GetVersion())
}

func TestAuthRouteRegistrar_GetPrefix(t *testing.T) {
	controller := createTestAuthController()
	registrar := NewAuthRouteRegistrar(controller)

	assert.Equal(t, "", registrar.GetPrefix())
}

func TestAuthRouteRegistrar_RegisterRoutes(t *testing.T) {
	tests := []struct {
		name           string
		controller     *AuthController
		expectError    bool
		expectedRoutes []string
	}{
		{
			name:        "successful route registration",
			controller:  createTestAuthController(),
			expectError: false,
			expectedRoutes: []string{
				"/auth/login",
				"/auth/logout",
				"/auth/refresh",
				"/auth/validate",
			},
		},
		{
			name:        "registration with nil controller",
			controller:  nil,
			expectError: false, // Should not error, but routes won't work
			expectedRoutes: []string{
				"/auth/login",
				"/auth/logout",
				"/auth/refresh",
				"/auth/validate",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test router
			router := gin.New()
			routerGroup := router.Group("")

			// Create registrar
			registrar := NewAuthRouteRegistrar(tt.controller)

			// Register routes
			err := registrar.RegisterRoutes(routerGroup)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify routes are registered
				routes := router.Routes()
				routePaths := make([]string, len(routes))
				for i, route := range routes {
					routePaths[i] = route.Path
				}

				for _, expectedRoute := range tt.expectedRoutes {
					assert.Contains(t, routePaths, expectedRoute, "Route %s should be registered", expectedRoute)
				}
			}
		})
	}
}

func TestAuthRouteRegistrar_RegisterRoutes_HTTPMethods(t *testing.T) {
	controller := createTestAuthController()
	registrar := NewAuthRouteRegistrar(controller)

	// Create test router
	router := gin.New()
	routerGroup := router.Group("")

	// Register routes
	err := registrar.RegisterRoutes(routerGroup)
	require.NoError(t, err)

	// Verify HTTP methods for each route
	routes := router.Routes()
	methodMap := make(map[string]string)
	for _, route := range routes {
		methodMap[route.Path] = route.Method
	}

	expectedMethods := map[string]string{
		"/auth/login":    "POST",
		"/auth/logout":   "POST",
		"/auth/refresh":  "POST",
		"/auth/validate": "GET",
	}

	for path, expectedMethod := range expectedMethods {
		actualMethod, exists := methodMap[path]
		assert.True(t, exists, "Route %s should exist", path)
		assert.Equal(t, expectedMethod, actualMethod, "Route %s should use %s method", path, expectedMethod)
	}
}

func TestAuthRouteRegistrar_IntegrationTest(t *testing.T) {
	// Create mock service with expectations
	mockService := &MockAuthService{}
	mockService.On("Login", mock.Anything, "testuser", "password123").Return(
		"access_token", "refresh_token", nil)

	// Create controller and registrar
	controller := NewAuthController(mockService)
	registrar := NewAuthRouteRegistrar(controller)

	// Create test router and register routes
	router := gin.New()
	routerGroup := router.Group("")
	err := registrar.RegisterRoutes(routerGroup)
	require.NoError(t, err)

	// Test login endpoint
	loginJSON := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(loginJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "access_token")

	// Verify mock expectations
	mockService.AssertExpectations(t)
}

func TestAuthRouteRegistrar_RouteGrouping(t *testing.T) {
	controller := createTestAuthController()
	registrar := NewAuthRouteRegistrar(controller)

	// Create test router with a parent group
	router := gin.New()
	parentGroup := router.Group("/api/v1")

	// Register routes under parent group
	err := registrar.RegisterRoutes(parentGroup)
	require.NoError(t, err)

	// Verify routes are prefixed correctly
	routes := router.Routes()
	routePaths := make([]string, len(routes))
	for i, route := range routes {
		routePaths[i] = route.Path
	}

	expectedRoutes := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/logout",
		"/api/v1/auth/refresh",
		"/api/v1/auth/validate",
	}

	for _, expectedRoute := range expectedRoutes {
		assert.Contains(t, routePaths, expectedRoute, "Route %s should be registered under parent group", expectedRoute)
	}
}

func TestAuthRouteRegistrar_MultipleRegistrations(t *testing.T) {
	controller := createTestAuthController()
	registrar := NewAuthRouteRegistrar(controller)

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
	routePaths := make([]string, len(routes))
	for i, route := range routes {
		routePaths[i] = route.Path
	}

	// Check v1 routes
	v1Routes := []string{"/v1/auth/login", "/v1/auth/logout", "/v1/auth/refresh", "/v1/auth/validate"}
	for _, route := range v1Routes {
		assert.Contains(t, routePaths, route)
	}

	// Check v2 routes
	v2Routes := []string{"/v2/auth/login", "/v2/auth/logout", "/v2/auth/refresh", "/v2/auth/validate"}
	for _, route := range v2Routes {
		assert.Contains(t, routePaths, route)
	}
}

func TestAuthRouteRegistrar_Interface(t *testing.T) {
	controller := createTestAuthController()
	registrar := NewAuthRouteRegistrar(controller)

	// Verify it implements the RouteRegistrar interface by testing all methods
	assert.Equal(t, "auth-api", registrar.GetName())
	assert.Equal(t, "root", registrar.GetVersion())
	assert.Equal(t, "", registrar.GetPrefix())

	// Test RegisterRoutes returns no error
	router := gin.New()
	routerGroup := router.Group("")
	err := registrar.RegisterRoutes(routerGroup)
	assert.NoError(t, err)
}

func TestAuthRouteRegistrar_NilController(t *testing.T) {
	registrar := NewAuthRouteRegistrar(nil)

	// Should still implement interface methods correctly
	assert.Equal(t, "auth-api", registrar.GetName())
	assert.Equal(t, "root", registrar.GetVersion())
	assert.Equal(t, "", registrar.GetPrefix())

	// Route registration should not panic
	router := gin.New()
	routerGroup := router.Group("")

	assert.NotPanics(t, func() {
		err := registrar.RegisterRoutes(routerGroup)
		assert.NoError(t, err)
	})
}

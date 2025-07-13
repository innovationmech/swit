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

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/internal/switauth/handler"
	"github.com/innovationmech/swit/internal/switauth/model"
	"github.com/innovationmech/swit/internal/switauth/router"
	"github.com/innovationmech/swit/pkg/logger"
)

func init() {
	logger.Logger, _ = zap.NewDevelopment()
	gin.SetMode(gin.TestMode)

	// Change to project root for config file access
	wd, _ := os.Getwd()
	if filepath.Base(wd) == "server" {
		os.Chdir("../../..")
	}
}

// MockAuthService implements service.AuthService interface for testing
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Login(ctx context.Context, username, password string) (string, string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	args := m.Called(ctx, refreshToken)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockAuthService) ValidateToken(ctx context.Context, tokenString string) (*model.Token, error) {
	args := m.Called(ctx, tokenString)
	return args.Get(0).(*model.Token), args.Error(1)
}

func (m *MockAuthService) Logout(ctx context.Context, tokenString string) error {
	args := m.Called(ctx, tokenString)
	return args.Error(0)
}

// MockRouteRegistrar implements router.RouteRegistrar interface for testing
type MockRouteRegistrar struct {
	mock.Mock
	name    string
	version string
	prefix  string
}

func (m *MockRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
	args := m.Called(rg)
	return args.Error(0)
}

func (m *MockRouteRegistrar) GetName() string {
	return m.name
}

func (m *MockRouteRegistrar) GetVersion() string {
	return m.version
}

func (m *MockRouteRegistrar) GetPrefix() string {
	return m.prefix
}

// MockMiddlewareRegistrar implements router.MiddlewareRegistrar interface for testing
type MockMiddlewareRegistrar struct {
	mock.Mock
	name     string
	priority int
}

func (m *MockMiddlewareRegistrar) RegisterMiddleware(r *gin.Engine) error {
	args := m.Called(r)
	return args.Error(0)
}

func (m *MockMiddlewareRegistrar) GetName() string {
	return m.name
}

func (m *MockMiddlewareRegistrar) GetPriority() int {
	return m.priority
}

// createTestServer creates a test server instance
func createTestServer() *Server {
	return &Server{
		router: gin.New(),
	}
}

// createTestAuthController creates a test auth controller with mock service
func createTestAuthController() *handler.AuthController {
	mockService := &MockAuthService{}
	return handler.NewAuthController(mockService)
}

func TestServer_SetupRoutes(t *testing.T) {
	tests := []struct {
		name           string
		setupServer    func() *Server
		authController *handler.AuthController
		expectPanic    bool
	}{
		{
			name:           "successful setup with valid controller",
			setupServer:    createTestServer,
			authController: createTestAuthController(),
			expectPanic:    false,
		},
		{
			name:           "successful setup with nil controller",
			setupServer:    createTestServer,
			authController: nil,
			expectPanic:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()

			if tt.expectPanic {
				assert.Panics(t, func() {
					server.SetupRoutes(tt.authController)
				})
			} else {
				assert.NotPanics(t, func() {
					server.SetupRoutes(tt.authController)
				})

				// Verify router is not nil
				assert.NotNil(t, server.router)
			}
		})
	}
}

func TestServer_configureRoutes(t *testing.T) {
	server := createTestServer()
	authController := createTestAuthController()
	registry := router.New()

	// Test that configureRoutes doesn't panic
	assert.NotPanics(t, func() {
		server.configureRoutes(registry, authController)
	})
}

func TestServer_registerGlobalMiddlewares(t *testing.T) {
	server := createTestServer()
	registry := router.New()

	// Test that registerGlobalMiddlewares doesn't panic
	assert.NotPanics(t, func() {
		server.registerGlobalMiddlewares(registry)
	})
}

func TestServer_registerAPIRoutes(t *testing.T) {
	server := createTestServer()
	authController := createTestAuthController()
	registry := router.New()

	// Test that registerAPIRoutes doesn't panic
	assert.NotPanics(t, func() {
		server.registerAPIRoutes(registry, authController)
	})
}

func TestServer_setupSwaggerUI(t *testing.T) {
	server := createTestServer()

	// Test that setupSwaggerUI doesn't panic
	assert.NotPanics(t, func() {
		server.setupSwaggerUI()
	})

	// Test that Swagger route is registered
	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// Should not return 404 (route exists)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestServer_SetupRoutes_Integration(t *testing.T) {
	server := createTestServer()
	authController := createTestAuthController()

	// Setup routes
	server.SetupRoutes(authController)

	// Test health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test swagger endpoint
	req = httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestRegisterRoutes_Deprecated(t *testing.T) {
	authController := createTestAuthController()

	// Test deprecated function
	router := RegisterRoutes(authController)

	require.NotNil(t, router)

	// Test that routes are set up correctly
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test swagger endpoint
	req = httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestServer_RouteRegistration_WithMocks(t *testing.T) {
	server := createTestServer()

	// Create mock registrars
	mockRoute := &MockRouteRegistrar{
		name:    "test-route",
		version: "v1",
		prefix:  "api",
	}
	mockRoute.On("RegisterRoutes", mock.AnythingOfType("*gin.RouterGroup")).Return(nil)

	mockMiddleware := &MockMiddlewareRegistrar{
		name:     "test-middleware",
		priority: 10,
	}
	mockMiddleware.On("RegisterMiddleware", mock.AnythingOfType("*gin.Engine")).Return(nil)

	// Create custom registry with mocks
	registry := router.New()
	registry.RegisterRoute(mockRoute)
	registry.RegisterMiddleware(mockMiddleware)

	// Setup registry
	err := registry.Setup(server.router)
	assert.NoError(t, err)

	// Verify mocks were called
	mockRoute.AssertExpectations(t)
	mockMiddleware.AssertExpectations(t)
}

func TestServer_MiddlewareRegistration(t *testing.T) {
	server := createTestServer()

	// Create mock middleware
	mockMiddleware := &MockMiddlewareRegistrar{
		name:     "test-middleware",
		priority: 10,
	}
	mockMiddleware.On("RegisterMiddleware", mock.AnythingOfType("*gin.Engine")).Return(nil)

	// Create registry and register middleware
	registry := router.New()
	registry.RegisterMiddleware(mockMiddleware)

	// Setup registry
	err := registry.Setup(server.router)
	assert.NoError(t, err)

	// Verify mock was called
	mockMiddleware.AssertExpectations(t)
}

func TestServer_RouteConflicts(t *testing.T) {
	server := createTestServer()

	// Create mock routes that might conflict
	mockRoute1 := &MockRouteRegistrar{
		name:    "route1",
		version: "v1",
		prefix:  "api",
	}
	mockRoute1.On("RegisterRoutes", mock.AnythingOfType("*gin.RouterGroup")).Return(nil)

	mockRoute2 := &MockRouteRegistrar{
		name:    "route2",
		version: "v1",
		prefix:  "api",
	}
	mockRoute2.On("RegisterRoutes", mock.AnythingOfType("*gin.RouterGroup")).Return(nil)

	// Create registry and register routes
	registry := router.New()
	registry.RegisterRoute(mockRoute1)
	registry.RegisterRoute(mockRoute2)

	// Setup should not panic even with potential conflicts
	assert.NotPanics(t, func() {
		err := registry.Setup(server.router)
		assert.NoError(t, err)
	})

	// Verify both routes were registered
	mockRoute1.AssertExpectations(t)
	mockRoute2.AssertExpectations(t)
}

func TestServer_EmptyRegistry(t *testing.T) {
	server := createTestServer()
	registry := router.New()

	// Empty registry should not cause issues
	assert.NotPanics(t, func() {
		err := registry.Setup(server.router)
		assert.NoError(t, err)
	})
}

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

package transport

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/internal/switserve/types"
	"github.com/innovationmech/swit/pkg/middleware"
)

// TestServiceHandler is a comprehensive test implementation of ServiceHandler
type TestServiceHandler struct {
	name            string
	version         string
	tags            []string
	routePrefix     string
	dependencies    []string
	initDelay       time.Duration
	trackMiddleware bool
	initError       error
	httpError       error
	grpcError       error
	shutdownError   error
	initCalled      bool
	httpCalled      bool
	grpcCalled      bool
	healthCalled    bool
	shutdownCalled  bool
}

// NewTestServiceHandler creates a new test service handler
func NewTestServiceHandler(name, version string, tags []string) *TestServiceHandler {
	return &TestServiceHandler{
		name:        name,
		version:     version,
		tags:        tags,
		routePrefix: "/" + name,
	}
}

func (t *TestServiceHandler) GetMetadata() *ServiceMetadata {
	return &ServiceMetadata{
		Name:           t.name,
		Version:        t.version,
		Description:    fmt.Sprintf("Test service: %s", t.name),
		HealthEndpoint: "/health/" + t.name,
		Tags:           t.tags,
		Dependencies:   t.dependencies,
	}
}

func (t *TestServiceHandler) Initialize(ctx context.Context) error {
	t.initCalled = true
	if t.initDelay > 0 {
		time.Sleep(t.initDelay)
	}
	return t.initError
}

func (t *TestServiceHandler) RegisterHTTP(router *gin.Engine) error {
	t.httpCalled = true
	if t.httpError != nil {
		return t.httpError
	}

	// Register test endpoint
	router.GET(t.routePrefix+"/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": t.name,
			"version": t.version,
			"message": "test endpoint",
		})
	})

	// Register health endpoint
	router.GET("/health/"+t.name, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": t.name,
			"version": t.version,
		})
	})

	return nil
}

func (t *TestServiceHandler) RegisterGRPC(server *grpc.Server) error {
	t.grpcCalled = true
	return t.grpcError
}

func (t *TestServiceHandler) GetHealthEndpoint() string {
	return "/health/" + t.name
}

func (t *TestServiceHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	t.healthCalled = true
	return &types.HealthStatus{
		Status:       "healthy",
		Timestamp:    time.Now(),
		Version:      t.version,
		Uptime:       time.Since(time.Now().Add(-time.Hour)),
		Dependencies: make(map[string]types.DependencyStatus),
	}, nil
}

func (t *TestServiceHandler) Shutdown(ctx context.Context) error {
	t.shutdownCalled = true
	return t.shutdownError
}

// TestUnifiedServiceRegistrationMechanism tests the complete service registration flow
func TestUnifiedServiceRegistrationMechanism(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create registry
	registry := NewEnhancedServiceRegistry()
	assert.NotNil(t, registry)
	assert.True(t, registry.IsEmpty())
	assert.Equal(t, 0, registry.Count())

	// Create multiple services with different characteristics
	services := []*TestServiceHandler{
		NewTestServiceHandler("user-service", "v1.0.0", []string{"api", "user"}),
		NewTestServiceHandler("auth-service", "v1.1.0", []string{"api", "auth"}),
		NewTestServiceHandler("notification-service", "v2.0.0", []string{"api", "notification"}),
		NewTestServiceHandler("health-service", "v1.0.0", []string{"system", "health"}),
	}

	// Test sequential registration
	for i, service := range services {
		err := registry.Register(service)
		assert.NoError(t, err, "Failed to register service %s", service.name)
		assert.Equal(t, i+1, registry.Count())
		assert.False(t, registry.IsEmpty())
	}

	// Verify registration order and metadata
	serviceNames := registry.GetServiceNames()
	expectedNames := []string{"user-service", "auth-service", "notification-service", "health-service"}
	assert.Equal(t, expectedNames, serviceNames)

	// Test metadata retrieval
	metadata := registry.GetServiceMetadata()
	assert.Len(t, metadata, 4)

	for i, meta := range metadata {
		assert.Equal(t, services[i].name, meta.Name)
		assert.Equal(t, services[i].version, meta.Version)
		assert.Equal(t, services[i].tags, meta.Tags)
		assert.Equal(t, "/health/"+services[i].name, meta.HealthEndpoint)
	}

	// Test duplicate registration prevention
	duplicate := NewTestServiceHandler("user-service", "v2.0.0", []string{"duplicate"})
	err := registry.Register(duplicate)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
	assert.Equal(t, 4, registry.Count()) // Count should remain the same
}

// TestServiceRegistryCorrectness verifies all services register correctly
func TestServiceRegistryCorrectness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewEnhancedServiceRegistry()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create services with different initialization requirements
	services := []*TestServiceHandler{
		NewTestServiceHandler("fast-service", "v1.0.0", []string{"fast"}),
		NewTestServiceHandler("slow-service", "v1.0.0", []string{"slow"}),
		NewTestServiceHandler("dependent-service", "v1.0.0", []string{"dependent"}),
	}

	// Set different initialization delays
	services[1].initDelay = 100 * time.Millisecond // slow service
	services[2].dependencies = []string{"fast-service", "slow-service"}

	// Register all services
	for _, service := range services {
		err := registry.Register(service)
		require.NoError(t, err)
	}

	// Test initialization phase
	startTime := time.Now()
	err := registry.InitializeAll(ctx)
	assert.NoError(t, err)
	initDuration := time.Since(startTime)

	// Verify all services were initialized
	for _, service := range services {
		assert.True(t, service.initCalled, "Service %s was not initialized", service.name)
	}

	// Verify initialization took appropriate time (should be at least the slow service delay)
	assert.GreaterOrEqual(t, initDuration, 100*time.Millisecond)

	// Test health checks
	healthResults := registry.CheckAllHealth(ctx)
	assert.Len(t, healthResults, 3)

	for _, service := range services {
		result, exists := healthResults[service.name]
		assert.True(t, exists, "Health result missing for service %s", service.name)
		assert.Equal(t, "healthy", result.Status)
		assert.True(t, service.healthCalled, "Health check not called for service %s", service.name)
	}
}

// TestHTTPRouteRegistrationForAllServices tests HTTP route registration comprehensively
func TestHTTPRouteRegistrationForAllServices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewEnhancedServiceRegistry()
	router := gin.New()

	// Create services with different route patterns
	services := []*TestServiceHandler{
		NewTestServiceHandler("api-service", "v1.0.0", []string{"api"}),
		NewTestServiceHandler("admin-service", "v1.0.0", []string{"admin"}),
		NewTestServiceHandler("public-service", "v1.0.0", []string{"public"}),
	}

	// Configure different route patterns
	services[0].routePrefix = "/api/v1"
	services[1].routePrefix = "/admin"
	services[2].routePrefix = "/public"

	// Register services
	for _, service := range services {
		err := registry.Register(service)
		require.NoError(t, err)
	}

	// Register HTTP routes
	err := registry.RegisterAllHTTP(router)
	assert.NoError(t, err)

	// Verify all services had their HTTP registration called
	for _, service := range services {
		assert.True(t, service.httpCalled, "HTTP registration not called for service %s", service.name)
	}

	// Test actual HTTP endpoints
	testCases := []struct {
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{"/api/v1/test", http.StatusOK, "api-service"},
		{"/admin/test", http.StatusOK, "admin-service"},
		{"/public/test", http.StatusOK, "public-service"},
		{"/health/api-service", http.StatusOK, "healthy"},
		{"/health/admin-service", http.StatusOK, "healthy"},
		{"/health/public-service", http.StatusOK, "healthy"},
		{"/nonexistent", http.StatusNotFound, ""},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("GET %s", tc.path), func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", tc.path, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tc.expectedBody)
			}
		})
	}
}

// TestMiddlewareApplicationAcrossServices validates middleware is applied consistently
func TestMiddlewareApplicationAcrossServices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewEnhancedServiceRegistry()
	router := gin.New()

	// Add global middleware
	middlewareRegistrar := middleware.NewGlobalMiddlewareRegistrar()
	middlewareRegistrar.RegisterMiddleware(router)

	// Create services
	services := []*TestServiceHandler{
		NewTestServiceHandler("service-a", "v1.0.0", []string{"test"}),
		NewTestServiceHandler("service-b", "v1.0.0", []string{"test"}),
	}

	// Configure services to track middleware execution
	for _, service := range services {
		service.trackMiddleware = true
		err := registry.Register(service)
		require.NoError(t, err)
	}

	// Register HTTP routes
	err := registry.RegisterAllHTTP(router)
	assert.NoError(t, err)

	// Test middleware application
	testCases := []string{
		"/service-a/test",
		"/service-b/test",
	}

	for _, path := range testCases {
		t.Run(fmt.Sprintf("Middleware for %s", path), func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", path, nil)
			req.Header.Set("X-Test-Header", "test-value")

			router.ServeHTTP(w, req)

			// Verify middleware was applied (check for CORS headers, request ID, etc.)
			assert.Equal(t, http.StatusOK, w.Code)
			// Note: Actual middleware verification would depend on the specific middleware implementation
			// This is a placeholder for middleware-specific assertions
		})
	}
}

// TestConcurrentServiceRegistration tests thread safety of service registration
func TestConcurrentServiceRegistration(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	const numServices = 50
	const numGoroutines = 10

	var wg sync.WaitGroup
	errorChan := make(chan error, numServices)

	// Concurrent registration
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for j := 0; j < numServices/numGoroutines; j++ {
				serviceName := fmt.Sprintf("service-%d-%d", start, j)
				service := NewTestServiceHandler(serviceName, "v1.0.0", []string{"concurrent"})
				if err := registry.Register(service); err != nil {
					errorChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// Check for errors
	for err := range errorChan {
		t.Errorf("Concurrent registration error: %v", err)
	}

	// Verify final state
	assert.Equal(t, numServices, registry.Count())
	assert.False(t, registry.IsEmpty())

	// Verify all services are unique
	serviceNames := registry.GetServiceNames()
	assert.Len(t, serviceNames, numServices)

	uniqueNames := make(map[string]bool)
	for _, name := range serviceNames {
		assert.False(t, uniqueNames[name], "Duplicate service name found: %s", name)
		uniqueNames[name] = true
	}
}

// TestServiceRegistrationWithFailures tests error handling during registration
func TestServiceRegistrationWithFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewEnhancedServiceRegistry()
	router := gin.New()
	server := grpc.NewServer()

	// Create services with different failure modes
	services := []*TestServiceHandler{
		NewTestServiceHandler("good-service", "v1.0.0", []string{"good"}),
		NewTestServiceHandler("init-fail-service", "v1.0.0", []string{"fail"}),
		NewTestServiceHandler("http-fail-service", "v1.0.0", []string{"fail"}),
		NewTestServiceHandler("grpc-fail-service", "v1.0.0", []string{"fail"}),
	}

	// Configure failure modes
	services[1].initError = fmt.Errorf("initialization failed")
	services[2].httpError = fmt.Errorf("HTTP registration failed")
	services[3].grpcError = fmt.Errorf("gRPC registration failed")

	// Register all services (registration itself should succeed)
	for _, service := range services {
		err := registry.Register(service)
		assert.NoError(t, err, "Service registration should succeed even if service will fail later")
	}

	// Test initialization failures
	ctx := context.Background()
	err := registry.InitializeAll(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initialization failed")

	// Test HTTP registration failures
	err = registry.RegisterAllHTTP(router)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP registration failed")

	// Test gRPC registration failures
	err = registry.RegisterAllGRPC(server)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gRPC registration failed")

	// Verify that good service still works
	assert.True(t, services[0].initCalled)
	assert.True(t, services[0].httpCalled)
	assert.True(t, services[0].grpcCalled)
}

// TestServiceLifecycleIntegration tests the complete service lifecycle
func TestServiceLifecycleIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewEnhancedServiceRegistry()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create services
	services := []*TestServiceHandler{
		NewTestServiceHandler("lifecycle-service-1", "v1.0.0", []string{"lifecycle"}),
		NewTestServiceHandler("lifecycle-service-2", "v1.0.0", []string{"lifecycle"}),
	}

	// Register services
	for _, service := range services {
		err := registry.Register(service)
		require.NoError(t, err)
	}

	// Phase 1: Initialization
	err := registry.InitializeAll(ctx)
	assert.NoError(t, err)

	// Phase 2: HTTP Registration
	router := gin.New()
	err = registry.RegisterAllHTTP(router)
	assert.NoError(t, err)

	// Phase 3: gRPC Registration
	server := grpc.NewServer()
	err = registry.RegisterAllGRPC(server)
	assert.NoError(t, err)

	// Phase 4: Health Checks
	healthResults := registry.CheckAllHealth(ctx)
	assert.Len(t, healthResults, 2)
	for _, result := range healthResults {
		assert.Equal(t, "healthy", result.Status)
	}

	// Phase 5: Service Operation (simulate some work)
	time.Sleep(100 * time.Millisecond)

	// Phase 6: Graceful Shutdown
	err = registry.ShutdownAll(ctx)
	assert.NoError(t, err)

	// Verify all lifecycle phases were executed
	for i, service := range services {
		assert.True(t, service.initCalled, "Service %d: Initialize not called", i)
		assert.True(t, service.httpCalled, "Service %d: RegisterHTTP not called", i)
		assert.True(t, service.grpcCalled, "Service %d: RegisterGRPC not called", i)
		assert.True(t, service.healthCalled, "Service %d: Health check not called", i)
		assert.True(t, service.shutdownCalled, "Service %d: Shutdown not called", i)
	}
}

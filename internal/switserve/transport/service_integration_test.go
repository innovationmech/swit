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
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/internal/switserve/types"
)

// Helper function to create a mock service handler with version
func NewMockServiceHandlerWithVersion(name, version string) *MockServiceHandler {
	handler := NewMockServiceHandler(name)
	handler.version = version
	return handler
}

// TestServiceRegistrationIntegration tests the complete service registration flow
func TestServiceRegistrationIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create registry and services
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandlerWithVersion("user-service", "v1.0.0")
	handler2 := NewMockServiceHandlerWithVersion("auth-service", "v1.1.0")

	// Test service registration
	err := registry.Register(handler1)
	assert.NoError(t, err)

	err = registry.Register(handler2)
	assert.NoError(t, err)

	// Verify registration order
	names := registry.GetServiceNames()
	assert.Equal(t, []string{"user-service", "auth-service"}, names)

	// Test initialization
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = registry.InitializeAll(ctx)
	assert.NoError(t, err)

	// Verify initialization was called
	assert.True(t, handler1.initCalled)
	assert.True(t, handler2.initCalled)

	// Test HTTP registration
	router := gin.New()
	err = registry.RegisterAllHTTP(router)
	assert.NoError(t, err)

	// Verify HTTP registration was called
	assert.True(t, handler1.httpCalled)
	assert.True(t, handler2.httpCalled)

	// Test gRPC registration
	server := grpc.NewServer()
	err = registry.RegisterAllGRPC(server)
	assert.NoError(t, err)

	// Verify gRPC registration was called
	assert.True(t, handler1.grpcCalled)
	assert.True(t, handler2.grpcCalled)
}

// TestServiceHealthCheckIntegration tests health check functionality
func TestServiceHealthCheckIntegration(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service1")
	handler2 := NewMockServiceHandler("service2")

	// Register services
	registry.Register(handler1)
	registry.Register(handler2)

	// Test health checks
	ctx := context.Background()
	results := registry.CheckAllHealth(ctx)

	assert.Len(t, results, 2)
	assert.Contains(t, results, "service1")
	assert.Contains(t, results, "service2")
	assert.Equal(t, "healthy", results["service1"].Status)
	assert.Equal(t, "healthy", results["service2"].Status)

	// Verify health check was called
	assert.True(t, handler1.healthCalled)
	assert.True(t, handler2.healthCalled)
}

// TestServiceShutdownIntegration tests graceful shutdown
func TestServiceShutdownIntegration(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service1")
	handler2 := NewMockServiceHandler("service2")

	// Register services
	registry.Register(handler1)
	registry.Register(handler2)

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := registry.ShutdownAll(ctx)
	assert.NoError(t, err)

	// Verify shutdown was called
	assert.True(t, handler1.shutdownCalled)
	assert.True(t, handler2.shutdownCalled)
}

// NilMetadataHandler is a handler that returns nil metadata for testing
type NilMetadataHandler struct{}

func (n *NilMetadataHandler) GetMetadata() *ServiceMetadata {
	return nil
}

func (n *NilMetadataHandler) Initialize(ctx context.Context) error {
	return nil
}

func (n *NilMetadataHandler) RegisterHTTP(router *gin.Engine) error {
	return nil
}

func (n *NilMetadataHandler) RegisterGRPC(server *grpc.Server) error {
	return nil
}

func (n *NilMetadataHandler) GetHealthEndpoint() string {
	return "/health/nil"
}

func (n *NilMetadataHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	return &types.HealthStatus{
		Status:       "healthy",
		Timestamp:    time.Now(),
		Version:      "1.0.0",
		Uptime:       time.Since(time.Now().Add(-time.Hour)),
		Dependencies: make(map[string]types.DependencyStatus),
	}, nil
}

func (n *NilMetadataHandler) Shutdown(ctx context.Context) error {
	return nil
}

// TestServiceRegistrationErrors tests error handling during registration
func TestServiceRegistrationErrors(t *testing.T) {
	registry := NewEnhancedServiceRegistry()

	// Test registering service with nil metadata
	handlerWithNilMetadata := &NilMetadataHandler{}
	err := registry.Register(handlerWithNilMetadata)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata cannot be nil")

	// Test registering service with empty name
	handlerWithEmptyName := NewMockServiceHandler("")
	err = registry.Register(handlerWithEmptyName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service name cannot be empty")

	// Test duplicate service registration
	handler1 := NewMockServiceHandler("duplicate-service")
	handler2 := NewMockServiceHandler("duplicate-service")

	err = registry.Register(handler1)
	assert.NoError(t, err)

	err = registry.Register(handler2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

// TestHTTPRouteRegistrationIntegration tests HTTP route registration with actual HTTP calls
func TestHTTPRouteRegistrationIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewEnhancedServiceRegistry()
	router := gin.New()

	// Create a real service handler that registers actual routes
	realHandler := &RealTestServiceHandler{
		name:        "test-service",
		version:     "v1.0.0",
		description: "Test service for integration testing",
	}

	// Register the service
	err := registry.Register(realHandler)
	assert.NoError(t, err)

	// Register HTTP routes
	err = registry.RegisterAllHTTP(router)
	assert.NoError(t, err)

	// Test the registered routes
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/hello", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Hello from test-service")

	// Test health endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/health/test-service", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

// RealTestServiceHandler is a real implementation for integration testing
type RealTestServiceHandler struct {
	name        string
	version     string
	description string
}

func (r *RealTestServiceHandler) RegisterHTTP(router *gin.Engine) error {
	v1 := router.Group("/test")
	{
		v1.GET("/hello", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Hello from test-service",
				"service": r.name,
				"version": r.version,
			})
		})
	}

	// Register health endpoint
	router.GET("/health/"+r.name, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": r.name,
			"version": r.version,
		})
	})

	return nil
}

func (r *RealTestServiceHandler) RegisterGRPC(server *grpc.Server) error {
	// No gRPC services for this test handler
	return nil
}

func (r *RealTestServiceHandler) GetMetadata() *ServiceMetadata {
	return &ServiceMetadata{
		Name:           r.name,
		Version:        r.version,
		Description:    r.description,
		HealthEndpoint: "/health/" + r.name,
		Tags:           []string{"test", "integration"},
		Dependencies:   []string{},
	}
}

func (r *RealTestServiceHandler) GetHealthEndpoint() string {
	return "/health/" + r.name
}

func (r *RealTestServiceHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	return &types.HealthStatus{
		Status:       "healthy",
		Timestamp:    time.Now(),
		Version:      r.version,
		Uptime:       time.Since(time.Now().Add(-time.Hour)),
		Dependencies: make(map[string]types.DependencyStatus),
	}, nil
}

func (r *RealTestServiceHandler) Initialize(ctx context.Context) error {
	// No initialization needed for test handler
	return nil
}

func (r *RealTestServiceHandler) Shutdown(ctx context.Context) error {
	// No cleanup needed for test handler
	return nil
}

// TestServiceMetadataIntegration tests service metadata functionality
func TestServiceMetadataIntegration(t *testing.T) {
	registry := NewEnhancedServiceRegistry()

	// Create services with different metadata
	handler1 := NewMockServiceHandler("user-service")
	handler2 := NewMockServiceHandler("auth-service")

	// Register services
	registry.Register(handler1)
	registry.Register(handler2)

	// Test metadata retrieval
	metadata := registry.GetServiceMetadata()
	assert.Len(t, metadata, 2)

	// Verify first service metadata
	assert.Equal(t, "user-service", metadata[0].Name)
	assert.Equal(t, "1.0.0", metadata[0].Version)
	assert.Contains(t, metadata[0].Tags, "test")
	assert.Contains(t, metadata[0].Tags, "mock")

	// Verify second service metadata
	assert.Equal(t, "auth-service", metadata[1].Name)
	assert.Equal(t, "1.0.0", metadata[1].Version)
	assert.Contains(t, metadata[1].Tags, "test")
	assert.Contains(t, metadata[1].Tags, "mock")
}

// TestConcurrentServiceOperations tests thread safety of service operations
func TestConcurrentServiceOperations(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	done := make(chan bool, 20)

	// Concurrent service registrations
	for i := 0; i < 10; i++ {
		go func(index int) {
			handler := NewMockServiceHandler(fmt.Sprintf("service-%d", index))
			err := registry.Register(handler)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Concurrent metadata reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = registry.GetServiceMetadata()
			_ = registry.GetServiceNames()
			done <- true
		}()
	}

	// Wait for all operations to complete
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify final state
	assert.Equal(t, 10, registry.Count())
	assert.False(t, registry.IsEmpty())
}

// TestServiceInitializationErrors tests error handling during initialization
func TestServiceInitializationErrors(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler := NewMockServiceHandler("failing-service")

	// Set initialization error
	handler.SetInitError(fmt.Errorf("initialization failed"))

	// Register service
	err := registry.Register(handler)
	assert.NoError(t, err)

	// Test initialization with error
	ctx := context.Background()
	err = registry.InitializeAll(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize service")
	assert.Contains(t, err.Error(), "failing-service")
}

// TestServiceHTTPRegistrationErrors tests error handling during HTTP registration
func TestServiceHTTPRegistrationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := NewEnhancedServiceRegistry()
	handler := NewMockServiceHandler("failing-service")

	// Set HTTP registration error
	handler.SetHTTPError(fmt.Errorf("HTTP registration failed"))

	// Register service
	err := registry.Register(handler)
	assert.NoError(t, err)

	// Test HTTP registration with error
	router := gin.New()
	err = registry.RegisterAllHTTP(router)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register HTTP routes")
	assert.Contains(t, err.Error(), "failing-service")
}

// TestServiceGRPCRegistrationErrors tests error handling during gRPC registration
func TestServiceGRPCRegistrationErrors(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler := NewMockServiceHandler("failing-service")

	// Set gRPC registration error
	handler.SetGRPCError(fmt.Errorf("gRPC registration failed"))

	// Register service
	err := registry.Register(handler)
	assert.NoError(t, err)

	// Test gRPC registration with error
	server := grpc.NewServer()
	err = registry.RegisterAllGRPC(server)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register gRPC services")
	assert.Contains(t, err.Error(), "failing-service")
}

// TestServiceShutdownErrors tests error handling during shutdown
func TestServiceShutdownErrors(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler := NewMockServiceHandler("failing-service")

	// Set shutdown error
	handler.SetShutdownError(fmt.Errorf("shutdown failed"))

	// Register service
	err := registry.Register(handler)
	assert.NoError(t, err)

	// Test shutdown with error
	ctx := context.Background()
	err = registry.ShutdownAll(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to shutdown service")
	assert.Contains(t, err.Error(), "failing-service")
}

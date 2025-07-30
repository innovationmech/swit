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
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/transport"
	"github.com/innovationmech/swit/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// ServiceMetadata represents service metadata information
type ServiceMetadata struct {
	Name        string
	Version     string
	Description string
}

// TestServiceHandler implements ServiceHandler for integration testing
type TestServiceHandler struct {
	name           string
	healthy        bool
	initializeFunc func(context.Context) error
	httpRoutes     map[string]gin.HandlerFunc
	grpcServices   []func(*grpc.Server)
	shutdownFunc   func(context.Context) error
}

func NewTestServiceHandler(name string) *TestServiceHandler {
	return &TestServiceHandler{
		name:       name,
		healthy:    true,
		httpRoutes: make(map[string]gin.HandlerFunc),
	}
}

func (t *TestServiceHandler) RegisterHTTP(router gin.IRouter) error {
	// Register custom HTTP routes
	for path, handler := range t.httpRoutes {
		router.GET(path, handler)
	}

	// Register health check route
	healthPath := t.GetHealthEndpoint()
	router.GET(healthPath, func(c *gin.Context) {
		status, err := t.IsHealthy(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return appropriate status code based on health status
		if status.Status == "healthy" {
			c.JSON(http.StatusOK, status)
		} else {
			c.JSON(http.StatusServiceUnavailable, status)
		}
	})

	return nil
}

func (t *TestServiceHandler) RegisterGRPC(server *grpc.Server) error {
	for _, registerFunc := range t.grpcServices {
		registerFunc(server)
	}
	return nil
}

func (t *TestServiceHandler) GetMetadata() *ServiceMetadata {
	return &ServiceMetadata{
		Name:        t.name,
		Version:     "1.0.0",
		Description: fmt.Sprintf("Test service: %s", t.name),
	}
}

func (t *TestServiceHandler) GetHealthEndpoint() string {
	return fmt.Sprintf("/health/%s", t.name)
}

func (t *TestServiceHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	if !t.healthy {
		return &types.HealthStatus{
			Status:    "unhealthy",
			Timestamp: time.Now(),
		}, nil
	}
	return &types.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
	}, nil
}

func (t *TestServiceHandler) Initialize(ctx context.Context) error {
	if t.initializeFunc != nil {
		return t.initializeFunc(ctx)
	}
	return nil
}

func (t *TestServiceHandler) Shutdown(ctx context.Context) error {
	if t.shutdownFunc != nil {
		return t.shutdownFunc(ctx)
	}
	return nil
}

func (t *TestServiceHandler) AddHTTPRoute(path string, handler gin.HandlerFunc) {
	t.httpRoutes[path] = handler
}

func (t *TestServiceHandler) SetHealthy(healthy bool) {
	t.healthy = healthy
}

func (t *TestServiceHandler) SetInitializeFunc(fn func(context.Context) error) {
	t.initializeFunc = fn
}

func (t *TestServiceHandler) SetShutdownFunc(fn func(context.Context) error) {
	t.shutdownFunc = fn
}

// TestServiceRegistrar implements ServiceRegistrar for testing
type TestServiceRegistrar struct {
	services []ServiceHandler
}

func NewTestServiceRegistrar() *TestServiceRegistrar {
	return &TestServiceRegistrar{
		services: make([]ServiceHandler, 0),
	}
}

func (r *TestServiceRegistrar) AddService(service ServiceHandler) {
	r.services = append(r.services, service)
}

func (r *TestServiceRegistrar) RegisterServices(registry ServiceRegistry) error {
	for _, service := range r.services {
		// Create adapter to bridge ServiceHandler to transport.HandlerRegister
		adapter := &testServiceAdapter{service: service}
		// Register the adapter with transport manager directly
		if err := registry.(*serviceRegistry).transportManager.RegisterHandler("http", adapter); err != nil {
			return err
		}

		// Register HTTP handler
		httpHandler := &testHTTPHandler{service: service}
		if err := registry.RegisterHTTPHandler(httpHandler); err != nil {
			return err
		}

		// Register health check
		healthCheck := &testHealthCheck{service: service}
		if err := registry.RegisterHealthCheck(healthCheck); err != nil {
			return err
		}
	}
	return nil
}

// testHTTPHandler implements HTTPHandler for testing
type testHTTPHandler struct {
	service ServiceHandler
}

func (h *testHTTPHandler) RegisterRoutes(router gin.IRouter) error {
	return h.service.RegisterHTTP(router)
}

func (h *testHTTPHandler) GetServiceName() string {
	return h.service.GetMetadata().Name
}

func (h *testHTTPHandler) GetVersion() string {
	return h.service.GetMetadata().Version
}

// testHealthCheck implements HealthCheck for testing
type testServiceAdapter struct {
	service ServiceHandler
}

// Implement transport.HandlerRegister interface
func (a *testServiceAdapter) RegisterHTTP(router *gin.Engine) error {
	return a.service.RegisterHTTP(router)
}

func (a *testServiceAdapter) RegisterGRPC(server *grpc.Server) error {
	return a.service.RegisterGRPC(server)
}

func (a *testServiceAdapter) GetMetadata() *transport.HandlerMetadata {
	meta := a.service.GetMetadata()
	return &transport.HandlerMetadata{
		Name:        meta.Name,
		Version:     meta.Version,
		Description: meta.Description,
	}
}

func (a *testServiceAdapter) GetHealthEndpoint() string {
	return a.service.GetHealthEndpoint()
}

func (a *testServiceAdapter) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	return a.service.IsHealthy(ctx)
}

func (a *testServiceAdapter) Initialize(ctx context.Context) error {
	return a.service.Initialize(ctx)
}

func (a *testServiceAdapter) Shutdown(ctx context.Context) error {
	return a.service.Shutdown(ctx)
}

type testHealthCheck struct {
	service ServiceHandler
}

func (h *testHealthCheck) Check(ctx context.Context) *types.HealthStatus {
	status, _ := h.service.IsHealthy(ctx)
	return status
}

func (h *testHealthCheck) GetName() string {
	return h.service.GetMetadata().Name
}

// ServiceHandler interface for testing
type ServiceHandler interface {
	RegisterHTTP(router gin.IRouter) error
	RegisterGRPC(server *grpc.Server) error
	GetMetadata() *ServiceMetadata
	GetHealthEndpoint() string
	IsHealthy(ctx context.Context) (*types.HealthStatus, error)
	Initialize(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// TestBaseServerIntegration tests the complete base server framework integration
func TestBaseServerIntegration(t *testing.T) {
	// Create server configuration
	config := NewServerConfig()
	config.ServiceName = "integration-test-server"
	config.HTTPPort = "8080" // Use fixed port
	config.GRPCPort = "9090" // Use fixed port
	config.EnableHTTP = true
	config.EnableGRPC = true
	config.EnableReady = true // Enable ready channel for testing
	config.SetDefaults()

	// Create test service handlers
	userService := NewTestServiceHandler("user")
	userService.AddHTTPRoute("/api/v1/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": "user", "status": "ok"})
	})

	authService := NewTestServiceHandler("auth")
	authService.AddHTTPRoute("/api/v1/auth", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": "auth", "status": "ok"})
	})

	// Create service registrar
	registrar := NewTestServiceRegistrar()
	registrar.AddService(userService)
	registrar.AddService(authService)

	// Create base server with registrar
	server, err := NewServer(config, registrar, nil)
	require.NoError(t, err)
	require.NotNil(t, server)

	// Start server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Start(ctx)
	require.NoError(t, err)

	// Wait for HTTP transport to be ready
	for _, tr := range server.transportManager.GetTransports() {
		if tr.GetName() == "http" {
			if httpTransport, ok := tr.(*transport.HTTPTransport); ok {
				select {
				case <-httpTransport.WaitReady():
					// HTTP transport is ready
				case <-time.After(5 * time.Second):
					t.Fatal("HTTP transport failed to become ready within timeout")
				}
			}
			break
		}
	}

	// Verify server is running
	httpAddr := server.GetHTTPAddress()
	assert.NotEmpty(t, httpAddr)
	assert.NotEmpty(t, server.GetGRPCAddress())
	assert.Equal(t, "integration-test-server", server.GetServiceName())

	// Test HTTP endpoints
	resp, err := http.Get(fmt.Sprintf("http://%s/api/v1/users", httpAddr))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp, err = http.Get(fmt.Sprintf("http://%s/api/v1/auth", httpAddr))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Test health endpoints
	resp, err = http.Get(fmt.Sprintf("http://%s/health/user", httpAddr))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp, err = http.Get(fmt.Sprintf("http://%s/health/auth", httpAddr))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Stop server
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	err = server.Stop(stopCtx)
	assert.NoError(t, err)
}

// TestMultiServiceRegistration tests registration of multiple services
func TestMultiServiceRegistration(t *testing.T) {
	config := NewServerConfig()
	config.ServiceName = "multi-service-test"
	config.HTTPPort = "8081"
	config.GRPCPort = "9091"
	config.SetDefaults()

	// Create service registrar
	registrar := NewTestServiceRegistrar()

	// Register multiple services
	for i := 0; i < 5; i++ {
		service := NewTestServiceHandler(fmt.Sprintf("service-%d", i))
		service.AddHTTPRoute(fmt.Sprintf("/api/v1/service-%d", i), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"service": fmt.Sprintf("service-%d", i), "status": "ok"})
		})
		registrar.AddService(service)
	}

	server, err := NewServer(config, registrar, nil)
	require.NoError(t, err)

	// Start server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Start(ctx)
	require.NoError(t, err)

	// Test all services are accessible
	httpAddr := server.GetHTTPAddress()
	for i := 0; i < 5; i++ {
		resp, err := http.Get(fmt.Sprintf("http://%s/api/v1/service-%d", httpAddr, i))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		// Test health endpoint
		resp, err = http.Get(fmt.Sprintf("http://%s/health/service-%d", httpAddr, i))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	}

	// Stop server
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	err = server.Stop(stopCtx)
	assert.NoError(t, err)
}

// TestServiceInitializationFailure tests handling of service initialization failures
func TestServiceInitializationFailure(t *testing.T) {
	config := NewServerConfig()
	config.ServiceName = "init-failure-test"
	config.HTTPPort = "8082"
	config.GRPCPort = "9092"
	config.SetDefaults()

	// Create service that fails initialization
	failingService := NewTestServiceHandler("failing-service")
	failingService.SetInitializeFunc(func(ctx context.Context) error {
		return fmt.Errorf("initialization failed")
	})

	// Create normal service
	normalService := NewTestServiceHandler("normal-service")
	normalService.AddHTTPRoute("/api/v1/normal", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": "normal", "status": "ok"})
	})

	// Create service registrar and register services
	registrar := NewTestServiceRegistrar()
	registrar.AddService(failingService)
	registrar.AddService(normalService)

	server, err := NewServer(config, registrar, nil)
	require.NoError(t, err)

	// Start server should fail due to initialization failure
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Start(ctx)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "initialization failed")
	}
}

// TestServiceHealthCheck tests service health check functionality
func TestServiceHealthCheck(t *testing.T) {
	config := NewServerConfig()
	config.ServiceName = "health-check-test"
	config.HTTPPort = "8083"
	config.GRPCPort = "9093"
	config.SetDefaults()

	// Create healthy service
	healthyService := NewTestServiceHandler("healthy-service")
	healthyService.SetHealthy(true)

	// Create unhealthy service
	unhealthyService := NewTestServiceHandler("unhealthy-service")
	unhealthyService.SetHealthy(false)

	// Create service registrar and register services
	registrar := NewTestServiceRegistrar()
	registrar.AddService(healthyService)
	registrar.AddService(unhealthyService)

	server, err := NewServer(config, registrar, nil)
	require.NoError(t, err)

	// Start server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Start(ctx)
	require.NoError(t, err)

	// Test health endpoints
	httpAddr := server.GetHTTPAddress()

	// Healthy service should return 200
	resp, err := http.Get(fmt.Sprintf("http://%s/health/healthy-service", httpAddr))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Unhealthy service should return 503
	resp, err = http.Get(fmt.Sprintf("http://%s/health/unhealthy-service", httpAddr))
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	resp.Body.Close()

	// Stop server
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	err = server.Stop(stopCtx)
	assert.NoError(t, err)
}

// TestServerLifecycleIntegration tests complete server lifecycle
func TestServerLifecycleIntegration(t *testing.T) {
	config := NewServerConfig()
	config.ServiceName = "lifecycle-test"
	config.HTTPPort = "8084"
	config.GRPCPort = "9094"
	config.SetDefaults()

	// Create service with lifecycle tracking
	var initCalled, shutdownCalled bool
	lifecycleService := NewTestServiceHandler("lifecycle-service")
	lifecycleService.SetInitializeFunc(func(ctx context.Context) error {
		initCalled = true
		return nil
	})
	lifecycleService.SetShutdownFunc(func(ctx context.Context) error {
		shutdownCalled = true
		return nil
	})

	// Create service registrar and register service
	registrar := NewTestServiceRegistrar()
	registrar.AddService(lifecycleService)

	server, err := NewServer(config, registrar, nil)
	require.NoError(t, err)

	// Start server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Start(ctx)
	require.NoError(t, err)
	assert.True(t, initCalled, "Initialize should be called during start")

	// Stop server
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	err = server.Stop(stopCtx)
	assert.NoError(t, err)
	assert.True(t, shutdownCalled, "Shutdown should be called during stop")
}

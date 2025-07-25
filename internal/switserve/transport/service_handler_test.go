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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/internal/switserve/interfaces"
	"github.com/innovationmech/swit/internal/switserve/types"
)

// MockServiceHandler implements ServiceHandler for testing
type MockServiceHandler struct {
	name           string
	version        string
	description    string
	initError      error
	healthError    error
	httpError      error
	grpcError      error
	shutdownErr    error
	initCalled     bool
	healthCalled   bool
	httpCalled     bool
	grpcCalled     bool
	shutdownCalled bool
}

func NewMockServiceHandler(name string) *MockServiceHandler {
	return &MockServiceHandler{
		name:        name,
		version:     "1.0.0",
		description: "Mock service for testing",
	}
}

func (m *MockServiceHandler) GetMetadata() *ServiceMetadata {
	return &ServiceMetadata{
		Name:        m.name,
		Version:     m.version,
		Description: m.description,
		Tags:        []string{"test", "mock"},
	}
}

func (m *MockServiceHandler) Initialize(ctx context.Context) error {
	m.initCalled = true
	return m.initError
}

func (m *MockServiceHandler) RegisterHTTP(router *gin.Engine) error {
	m.httpCalled = true
	return m.httpError
}

func (m *MockServiceHandler) RegisterGRPC(server *grpc.Server) error {
	m.grpcCalled = true
	return m.grpcError
}

func (m *MockServiceHandler) HealthCheck(ctx context.Context) *interfaces.HealthStatus {
	m.healthCalled = true
	if m.healthError != nil {
		return &interfaces.HealthStatus{
			Status:    "unhealthy",
			Timestamp: time.Now().Unix(),
			Details:   map[string]string{"error": m.healthError.Error()},
		}
	}
	return &interfaces.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		Details:   map[string]string{"service": m.name},
	}
}

// HealthCheck is a convenience method for testing that wraps IsHealthy
func (m *MockServiceHandler) HealthCheckCompat(ctx context.Context) *interfaces.HealthStatus {
	status, _ := m.IsHealthy(ctx)
	if status == nil {
		return &interfaces.HealthStatus{
			Status:    "unhealthy",
			Timestamp: time.Now().Unix(),
			Details:   map[string]string{"error": "health check failed"},
		}
	}
	return &interfaces.HealthStatus{
		Status:    status.Status,
		Timestamp: status.Timestamp.Unix(),
		Details:   map[string]string{"service": m.name},
	}
}

func (m *MockServiceHandler) GetHealthEndpoint() string {
	return "/health/" + m.name
}

// IsHealthy implements the ServiceHandler interface
func (m *MockServiceHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	m.healthCalled = true
	if m.healthError != nil {
		return &types.HealthStatus{
			Status:       "unhealthy",
			Timestamp:    time.Now(),
			Version:      m.version,
			Uptime:       time.Since(time.Now().Add(-time.Hour)),
			Dependencies: make(map[string]types.DependencyStatus),
		}, m.healthError
	}
	return &types.HealthStatus{
		Status:       "healthy",
		Timestamp:    time.Now(),
		Version:      m.version,
		Uptime:       time.Since(time.Now().Add(-time.Hour)),
		Dependencies: make(map[string]types.DependencyStatus),
	}, nil
}

func (m *MockServiceHandler) Shutdown(ctx context.Context) error {
	m.shutdownCalled = true
	return m.shutdownErr
}

// Test helper methods
func (m *MockServiceHandler) SetInitError(err error) {
	m.initError = err
}

func (m *MockServiceHandler) SetHealthError(err error) {
	m.healthError = err
}

func (m *MockServiceHandler) SetHTTPError(err error) {
	m.httpError = err
}

func (m *MockServiceHandler) SetGRPCError(err error) {
	m.grpcError = err
}

func (m *MockServiceHandler) SetShutdownError(err error) {
	m.shutdownErr = err
}

func TestServiceMetadata(t *testing.T) {
	metadata := &ServiceMetadata{
		Name:        "test-service",
		Version:     "1.0.0",
		Description: "Test service",
		Tags:        []string{"test"},
	}

	assert.Equal(t, "test-service", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Equal(t, "Test service", metadata.Description)
	assert.Equal(t, []string{"test"}, metadata.Tags)
}

func TestMockServiceHandler(t *testing.T) {
	handler := NewMockServiceHandler("test-service")

	// Test metadata
	metadata := handler.GetMetadata()
	assert.Equal(t, "test-service", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Equal(t, "Mock service for testing", metadata.Description)
	assert.Equal(t, []string{"test", "mock"}, metadata.Tags)

	// Test initialize
	ctx := context.Background()
	err := handler.Initialize(ctx)
	assert.NoError(t, err)
	assert.True(t, handler.initCalled)

	// Test health check
	health := handler.HealthCheck(ctx)
	assert.Equal(t, "healthy", health.Status)
	assert.True(t, handler.healthCalled)

	// Test HTTP registration
	router := gin.New()
	err = handler.RegisterHTTP(router)
	assert.NoError(t, err)
	assert.True(t, handler.httpCalled)

	// Test gRPC registration
	server := grpc.NewServer()
	err = handler.RegisterGRPC(server)
	assert.NoError(t, err)
	assert.True(t, handler.grpcCalled)

	// Test shutdown
	err = handler.Shutdown(ctx)
	assert.NoError(t, err)
	assert.True(t, handler.shutdownCalled)
}

func TestMockServiceHandlerErrors(t *testing.T) {
	handler := NewMockServiceHandler("error-service")
	ctx := context.Background()

	// Test initialize error
	initErr := errors.New("init failed")
	handler.SetInitError(initErr)
	err := handler.Initialize(ctx)
	assert.Equal(t, initErr, err)

	// Test health check error
	healthErr := errors.New("health check failed")
	handler.SetHealthError(healthErr)
	health := handler.HealthCheck(ctx)
	assert.Equal(t, "unhealthy", health.Status)
	assert.Equal(t, healthErr.Error(), health.Details["error"])

	// Test HTTP error
	httpErr := errors.New("http registration failed")
	handler.SetHTTPError(httpErr)
	router := gin.New()
	err = handler.RegisterHTTP(router)
	assert.Equal(t, httpErr, err)

	// Test gRPC error
	grpcErr := errors.New("grpc registration failed")
	handler.SetGRPCError(grpcErr)
	server := grpc.NewServer()
	err = handler.RegisterGRPC(server)
	assert.Equal(t, grpcErr, err)

	// Test shutdown error
	shutdownErr := errors.New("shutdown failed")
	handler.SetShutdownError(shutdownErr)
	err = handler.Shutdown(ctx)
	assert.Equal(t, shutdownErr, err)
}

func TestNewEnhancedServiceRegistry(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	assert.NotNil(t, registry)
	assert.True(t, registry.IsEmpty())
	assert.Equal(t, 0, registry.Count())
}

func TestEnhancedServiceRegistryRegister(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler := NewMockServiceHandler("test-service")

	// Test successful registration
	err := registry.Register(handler)
	assert.NoError(t, err)
	assert.False(t, registry.IsEmpty())
	assert.Equal(t, 1, registry.Count())

	// Test duplicate registration
	err = registry.Register(handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
	assert.Equal(t, 1, registry.Count())
}

func TestEnhancedServiceRegistryUnregister(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler := NewMockServiceHandler("test-service")

	// Test unregister non-existent service
	err := registry.Unregister("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")

	// Register and then unregister
	err = registry.Register(handler)
	assert.NoError(t, err)

	err = registry.Unregister("test-service")
	assert.NoError(t, err)
	assert.True(t, registry.IsEmpty())
	assert.Equal(t, 0, registry.Count())
}

func TestEnhancedServiceRegistryGetHandler(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler := NewMockServiceHandler("test-service")

	// Test get non-existent handler
	_, err := registry.GetHandler("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")

	// Register and get handler
	err = registry.Register(handler)
	assert.NoError(t, err)

	retrieved, err := registry.GetHandler("test-service")
	assert.NoError(t, err)
	assert.Equal(t, handler, retrieved)
}

func TestEnhancedServiceRegistryGetAllHandlers(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service-1")
	handler2 := NewMockServiceHandler("service-2")

	// Test empty registry
	handlers := registry.GetAllHandlers()
	assert.Empty(t, handlers)

	// Register handlers
	err := registry.Register(handler1)
	assert.NoError(t, err)
	err = registry.Register(handler2)
	assert.NoError(t, err)

	// Get all handlers
	handlers = registry.GetAllHandlers()
	assert.Len(t, handlers, 2)
	assert.Equal(t, handler1, handlers[0])
	assert.Equal(t, handler2, handlers[1])
}

func TestEnhancedServiceRegistryGetServiceNames(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service-1")
	handler2 := NewMockServiceHandler("service-2")

	// Test empty registry
	names := registry.GetServiceNames()
	assert.Empty(t, names)

	// Register handlers
	err := registry.Register(handler1)
	assert.NoError(t, err)
	err = registry.Register(handler2)
	assert.NoError(t, err)

	// Get service names
	names = registry.GetServiceNames()
	assert.Len(t, names, 2)
	assert.Equal(t, "service-1", names[0])
	assert.Equal(t, "service-2", names[1])
}

func TestEnhancedServiceRegistryGetServiceMetadata(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler := NewMockServiceHandler("test-service")

	// Test empty registry
	metadata := registry.GetServiceMetadata()
	assert.Empty(t, metadata)

	// Register handler
	err := registry.Register(handler)
	assert.NoError(t, err)

	// Get metadata
	metadata = registry.GetServiceMetadata()
	assert.Len(t, metadata, 1)
	assert.Equal(t, "test-service", metadata[0].Name)
	assert.Equal(t, "1.0.0", metadata[0].Version)
}

func TestEnhancedServiceRegistryInitializeAll(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service-1")
	handler2 := NewMockServiceHandler("service-2")

	// Register handlers
	err := registry.Register(handler1)
	assert.NoError(t, err)
	err = registry.Register(handler2)
	assert.NoError(t, err)

	// Initialize all
	ctx := context.Background()
	err = registry.InitializeAll(ctx)
	assert.NoError(t, err)
	assert.True(t, handler1.initCalled)
	assert.True(t, handler2.initCalled)

	// Test with error
	handler2.SetInitError(errors.New("init failed"))
	err = registry.InitializeAll(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "init failed")
}

func TestEnhancedServiceRegistryRegisterAllHTTP(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service-1")
	handler2 := NewMockServiceHandler("service-2")

	// Register handlers
	err := registry.Register(handler1)
	assert.NoError(t, err)
	err = registry.Register(handler2)
	assert.NoError(t, err)

	// Register all HTTP
	router := gin.New()
	err = registry.RegisterAllHTTP(router)
	assert.NoError(t, err)
	assert.True(t, handler1.httpCalled)
	assert.True(t, handler2.httpCalled)

	// Test with error
	handler2.SetHTTPError(errors.New("http failed"))
	err = registry.RegisterAllHTTP(router)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "http failed")
}

func TestEnhancedServiceRegistryRegisterAllGRPC(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service-1")
	handler2 := NewMockServiceHandler("service-2")

	// Register handlers
	err := registry.Register(handler1)
	assert.NoError(t, err)
	err = registry.Register(handler2)
	assert.NoError(t, err)

	// Register all gRPC
	server := grpc.NewServer()
	err = registry.RegisterAllGRPC(server)
	assert.NoError(t, err)
	assert.True(t, handler1.grpcCalled)
	assert.True(t, handler2.grpcCalled)

	// Test with error
	handler2.SetGRPCError(errors.New("grpc failed"))
	err = registry.RegisterAllGRPC(server)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "grpc failed")
}

func TestEnhancedServiceRegistryCheckAllHealth(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service-1")
	handler2 := NewMockServiceHandler("service-2")

	// Register handlers
	err := registry.Register(handler1)
	assert.NoError(t, err)
	err = registry.Register(handler2)
	assert.NoError(t, err)

	// Check all health
	ctx := context.Background()
	healthResults := registry.CheckAllHealth(ctx)
	assert.Len(t, healthResults, 2)
	assert.Equal(t, "healthy", healthResults["service-1"].Status)
	assert.Equal(t, "healthy", healthResults["service-2"].Status)
	assert.True(t, handler1.healthCalled)
	assert.True(t, handler2.healthCalled)

	// Test with health error
	handler2.SetHealthError(errors.New("health failed"))
	healthResults = registry.CheckAllHealth(ctx)
	assert.Equal(t, "unhealthy", healthResults["service-2"].Status)
}

func TestEnhancedServiceRegistryShutdownAll(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service-1")
	handler2 := NewMockServiceHandler("service-2")

	// Register handlers
	err := registry.Register(handler1)
	assert.NoError(t, err)
	err = registry.Register(handler2)
	assert.NoError(t, err)

	// Shutdown all
	ctx := context.Background()
	err = registry.ShutdownAll(ctx)
	assert.NoError(t, err)
	assert.True(t, handler1.shutdownCalled)
	assert.True(t, handler2.shutdownCalled)

	// Test with error
	handler1.SetShutdownError(errors.New("shutdown failed"))
	err = registry.ShutdownAll(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shutdown failed")
}

func TestEnhancedServiceRegistryConcurrency(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	ctx := context.Background()

	// Test concurrent registration
	go func() {
		for i := 0; i < 10; i++ {
			handler := NewMockServiceHandler(fmt.Sprintf("service-%d", i))
			registry.Register(handler)
		}
	}()

	go func() {
		for i := 10; i < 20; i++ {
			handler := NewMockServiceHandler(fmt.Sprintf("service-%d", i))
			registry.Register(handler)
		}
	}()

	// Wait a bit for goroutines to complete
	time.Sleep(100 * time.Millisecond)

	// Test concurrent reads
	go func() {
		for i := 0; i < 5; i++ {
			registry.GetAllHandlers()
			registry.GetServiceNames()
			registry.Count()
		}
	}()

	go func() {
		for i := 0; i < 5; i++ {
			registry.CheckAllHealth(ctx)
			registry.GetServiceMetadata()
		}
	}()

	// Wait for concurrent operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify final state
	assert.True(t, registry.Count() > 0)
	assert.False(t, registry.IsEmpty())
}

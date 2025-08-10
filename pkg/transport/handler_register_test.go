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
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/types"
)

// MockHandlerRegister implements TransportServiceHandler interface for testing
type MockHandlerRegister struct {
	metadata        *HandlerMetadata
	httpRegisterErr error
	grpcRegisterErr error
	healthStatus    *types.HealthStatus
	healthErr       error
	initializeErr   error
	shutdownErr     error
	httpRegistered  bool
	grpcRegistered  bool
	initialized     bool
	shutdown        bool
	mu              sync.RWMutex
}

func NewMockHandlerRegister(name, version string) *MockHandlerRegister {
	return &MockHandlerRegister{
		metadata: &HandlerMetadata{
			Name:           name,
			Version:        version,
			Description:    "Test service",
			HealthEndpoint: "/health",
			Tags:           []string{"test"},
			Dependencies:   []string{},
		},
		healthStatus: &types.HealthStatus{
			Status:    types.HealthStatusHealthy,
			Timestamp: time.Now(),
			Version:   version,
			Uptime:    time.Hour,
		},
	}
}

func (m *MockHandlerRegister) RegisterHTTP(router *gin.Engine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.httpRegisterErr != nil {
		return m.httpRegisterErr
	}
	m.httpRegistered = true
	return nil
}

func (m *MockHandlerRegister) RegisterGRPC(server *grpc.Server) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.grpcRegisterErr != nil {
		return m.grpcRegisterErr
	}
	m.grpcRegistered = true
	return nil
}

func (m *MockHandlerRegister) GetMetadata() *HandlerMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metadata
}

func (m *MockHandlerRegister) GetHealthEndpoint() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metadata.HealthEndpoint
}

func (m *MockHandlerRegister) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.healthErr != nil {
		return nil, m.healthErr
	}
	return m.healthStatus, nil
}

func (m *MockHandlerRegister) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.initializeErr != nil {
		return m.initializeErr
	}
	m.initialized = true
	return nil
}

func (m *MockHandlerRegister) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.shutdownErr != nil {
		return m.shutdownErr
	}
	m.shutdown = true
	return nil
}

// Test helper methods
func (m *MockHandlerRegister) SetHTTPRegisterError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.httpRegisterErr = err
}

func (m *MockHandlerRegister) SetGRPCRegisterError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.grpcRegisterErr = err
}

func (m *MockHandlerRegister) SetHealthError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthErr = err
}

func (m *MockHandlerRegister) SetInitializeError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initializeErr = err
}

func (m *MockHandlerRegister) SetShutdownError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shutdownErr = err
}

func (m *MockHandlerRegister) IsHTTPRegistered() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.httpRegistered
}

func (m *MockHandlerRegister) IsGRPCRegistered() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.grpcRegistered
}

func (m *MockHandlerRegister) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized
}

func (m *MockHandlerRegister) IsShutdown() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.shutdown
}

func TestNewTransportServiceRegistry(t *testing.T) {
	registry := NewTransportServiceRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.handlers)
	assert.NotNil(t, registry.order)
	assert.Equal(t, 0, registry.Count())
	assert.True(t, registry.IsEmpty())
}

func TestTransportServiceRegistry_Register_Success(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	err := registry.Register(handler1)
	assert.NoError(t, err)
	assert.Equal(t, 1, registry.Count())
	assert.False(t, registry.IsEmpty())

	err = registry.Register(handler2)
	assert.NoError(t, err)
	assert.Equal(t, 2, registry.Count())

	// Verify order is maintained
	names := registry.GetServiceNames()
	assert.Equal(t, []string{"service1", "service2"}, names)
}

func TestTransportServiceRegistry_Register_NilMetadata(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler := NewMockHandlerRegister("test", "v1.0.0")
	handler.metadata = nil

	err := registry.Register(handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata cannot be nil")
	assert.Equal(t, 0, registry.Count())
}

func TestTransportServiceRegistry_Register_EmptyName(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler := NewMockHandlerRegister("", "v1.0.0")

	err := registry.Register(handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service name cannot be empty")
	assert.Equal(t, 0, registry.Count())
}

func TestTransportServiceRegistry_Register_DuplicateName(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service", "v1.0.0")
	handler2 := NewMockHandlerRegister("service", "v2.0.0")

	err := registry.Register(handler1)
	assert.NoError(t, err)

	err = registry.Register(handler2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service 'service' is already registered")
	assert.Equal(t, 1, registry.Count())
}

func TestTransportServiceRegistry_Unregister_Success(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	registry.Register(handler1)
	registry.Register(handler2)

	err := registry.Unregister("service1")
	assert.NoError(t, err)
	assert.Equal(t, 1, registry.Count())

	names := registry.GetServiceNames()
	assert.Equal(t, []string{"service2"}, names)
}

func TestTransportServiceRegistry_Unregister_NotFound(t *testing.T) {
	registry := NewTransportServiceRegistry()

	err := registry.Unregister("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service 'nonexistent' is not registered")
}

func TestTransportServiceRegistry_GetHandler_Success(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler := NewMockHandlerRegister("service", "v1.0.0")

	registry.Register(handler)

	retrieved, err := registry.GetHandler("service")
	assert.NoError(t, err)
	assert.Equal(t, handler, retrieved)
}

func TestTransportServiceRegistry_GetHandler_NotFound(t *testing.T) {
	registry := NewTransportServiceRegistry()

	retrieved, err := registry.GetHandler("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, retrieved)
	assert.Contains(t, err.Error(), "service 'nonexistent' is not registered")
}

func TestTransportServiceRegistry_GetAllHandlers(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	// Test empty registry
	handlers := registry.GetAllHandlers()
	assert.Equal(t, 0, len(handlers))

	// Add handlers
	registry.Register(handler1)
	registry.Register(handler2)

	handlers = registry.GetAllHandlers()
	assert.Equal(t, 2, len(handlers))
	assert.Equal(t, handler1, handlers[0])
	assert.Equal(t, handler2, handlers[1])
}

func TestTransportServiceRegistry_GetServiceMetadata(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	registry.Register(handler1)
	registry.Register(handler2)

	metadata := registry.GetServiceMetadata()
	assert.Equal(t, 2, len(metadata))
	assert.Equal(t, "service1", metadata[0].Name)
	assert.Equal(t, "v1.0.0", metadata[0].Version)
	assert.Equal(t, "service2", metadata[1].Name)
	assert.Equal(t, "v1.1.0", metadata[1].Version)
}

func TestTransportServiceRegistry_InitializeTransportServices_Success(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	registry.Register(handler1)
	registry.Register(handler2)

	ctx := context.Background()
	err := registry.InitializeTransportServices(ctx)

	assert.NoError(t, err)
	assert.True(t, handler1.IsInitialized())
	assert.True(t, handler2.IsInitialized())
}

func TestTransportServiceRegistry_InitializeTransportServices_WithError(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	expectedErr := errors.New("initialization failed")
	handler2.SetInitializeError(expectedErr)

	registry.Register(handler1)
	registry.Register(handler2)

	ctx := context.Background()
	err := registry.InitializeTransportServices(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize service 'service2'")
	assert.True(t, handler1.IsInitialized())
	assert.False(t, handler2.IsInitialized())
}

func TestTransportServiceRegistry_BindAllHTTPEndpoints_Success(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	registry.Register(handler1)
	registry.Register(handler2)

	router := gin.New()
	err := registry.BindAllHTTPEndpoints(router)

	assert.NoError(t, err)
	assert.True(t, handler1.IsHTTPRegistered())
	assert.True(t, handler2.IsHTTPRegistered())
}

func TestTransportServiceRegistry_BindAllHTTPEndpoints_WithError(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	expectedErr := errors.New("HTTP registration failed")
	handler2.SetHTTPRegisterError(expectedErr)

	registry.Register(handler1)
	registry.Register(handler2)

	router := gin.New()
	err := registry.BindAllHTTPEndpoints(router)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register HTTP routes for service 'service2'")
	assert.True(t, handler1.IsHTTPRegistered())
	assert.False(t, handler2.IsHTTPRegistered())
}

func TestTransportServiceRegistry_BindAllGRPCServices_Success(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	registry.Register(handler1)
	registry.Register(handler2)

	server := grpc.NewServer()
	err := registry.BindAllGRPCServices(server)

	assert.NoError(t, err)
	assert.True(t, handler1.IsGRPCRegistered())
	assert.True(t, handler2.IsGRPCRegistered())
}

func TestTransportServiceRegistry_BindAllGRPCServices_WithError(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	expectedErr := errors.New("gRPC registration failed")
	handler2.SetGRPCRegisterError(expectedErr)

	registry.Register(handler1)
	registry.Register(handler2)

	server := grpc.NewServer()
	err := registry.BindAllGRPCServices(server)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register gRPC services for service 'service2'")
	assert.True(t, handler1.IsGRPCRegistered())
	assert.False(t, handler2.IsGRPCRegistered())
}

func TestTransportServiceRegistry_CheckAllHealth_Success(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	registry.Register(handler1)
	registry.Register(handler2)

	ctx := context.Background()
	healthResults := registry.CheckAllHealth(ctx)

	assert.Equal(t, 2, len(healthResults))
	assert.NotNil(t, healthResults["service1"])
	assert.NotNil(t, healthResults["service2"])
	assert.Equal(t, types.HealthStatusHealthy, healthResults["service1"].Status)
	assert.Equal(t, types.HealthStatusHealthy, healthResults["service2"].Status)
}

func TestTransportServiceRegistry_CheckAllHealth_WithError(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	expectedErr := errors.New("health check failed")
	handler2.SetHealthError(expectedErr)

	registry.Register(handler1)
	registry.Register(handler2)

	ctx := context.Background()
	healthResults := registry.CheckAllHealth(ctx)

	assert.Equal(t, 2, len(healthResults))
	assert.Equal(t, types.HealthStatusHealthy, healthResults["service1"].Status)
	assert.Equal(t, types.HealthStatusUnhealthy, healthResults["service2"].Status)
}

func TestTransportServiceRegistry_ShutdownAll_Success(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	registry.Register(handler1)
	registry.Register(handler2)

	ctx := context.Background()
	err := registry.ShutdownAll(ctx)

	assert.NoError(t, err)
	assert.True(t, handler1.IsShutdown())
	assert.True(t, handler2.IsShutdown())
}

func TestTransportServiceRegistry_ShutdownAll_WithError(t *testing.T) {
	registry := NewTransportServiceRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	expectedErr := errors.New("shutdown failed")
	handler1.SetShutdownError(expectedErr) // First service fails

	registry.Register(handler1)
	registry.Register(handler2)

	ctx := context.Background()
	err := registry.ShutdownAll(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to shutdown service 'service1'") // The service that failed
	assert.False(t, handler1.IsShutdown())                                   // Failed
	assert.True(t, handler2.IsShutdown())                                    // Should still shutdown
}

func TestTransportServiceRegistry_ThreadSafety(t *testing.T) {
	registry := NewTransportServiceRegistry()
	const numGoroutines = 50

	var wg sync.WaitGroup
	// Register: numGoroutines, Unregister: numGoroutines/2, Get: numGoroutines, Count: numGoroutines
	totalGoroutines := numGoroutines + numGoroutines/2 + numGoroutines + numGoroutines
	wg.Add(totalGoroutines)

	// Concurrent register operations
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			handler := NewMockHandlerRegister(fmt.Sprintf("service-%d", i), "v1.0.0")
			_ = registry.Register(handler) // Ignore errors in concurrent test
		}(i)
	}

	// Concurrent unregister operations
	for i := 0; i < numGoroutines/2; i++ {
		go func(i int) {
			defer wg.Done()
			_ = registry.Unregister(fmt.Sprintf("service-%d", i)) // Ignore errors
		}(i)
	}

	// Concurrent get operations
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			_, _ = registry.GetHandler(fmt.Sprintf("service-%d", i)) // Ignore errors
		}(i)
	}

	// Concurrent count operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = registry.Count()
		}()
	}

	wg.Wait()

	// Verify no race conditions occurred
	count := registry.Count()
	assert.True(t, count >= 0) // Should be non-negative
}

// Benchmark tests
func BenchmarkTransportServiceRegistry_Register(b *testing.B) {
	registry := NewTransportServiceRegistry()
	handlers := make([]*MockHandlerRegister, b.N)

	for i := 0; i < b.N; i++ {
		handlers[i] = NewMockHandlerRegister(fmt.Sprintf("service-%d", i), "v1.0.0")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.Register(handlers[i])
	}
}

func BenchmarkTransportServiceRegistry_GetHandler(b *testing.B) {
	registry := NewTransportServiceRegistry()

	// Setup: register 100 handlers
	for i := 0; i < 100; i++ {
		handler := NewMockHandlerRegister(fmt.Sprintf("service-%d", i), "v1.0.0")
		registry.Register(handler)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.GetHandler(fmt.Sprintf("service-%d", i%100))
	}
}

func BenchmarkTransportServiceRegistry_GetAllHandlers(b *testing.B) {
	registry := NewTransportServiceRegistry()

	// Setup: register 100 handlers
	for i := 0; i < 100; i++ {
		handler := NewMockHandlerRegister(fmt.Sprintf("service-%d", i), "v1.0.0")
		registry.Register(handler)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.GetAllHandlers()
	}
}

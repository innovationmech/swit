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
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/internal/switauth/types"
)

// MockServiceHandler is a mock implementation of HandlerRegister interface
type MockServiceHandler struct {
	mock.Mock
	metadata *HandlerMetadata
}

func NewMockServiceHandler(name, version, description string) *MockServiceHandler {
	return &MockServiceHandler{
		metadata: &HandlerMetadata{
			Name:           name,
			Version:        version,
			Description:    description,
			HealthEndpoint: "/health",
			Tags:           []string{"test"},
			Dependencies:   []string{},
		},
	}
}

func (m *MockServiceHandler) RegisterHTTP(router *gin.Engine) error {
	args := m.Called(router)
	return args.Error(0)
}

func (m *MockServiceHandler) RegisterGRPC(server *grpc.Server) error {
	args := m.Called(server)
	return args.Error(0)
}

func (m *MockServiceHandler) GetMetadata() *HandlerMetadata {
	return m.metadata
}

func (m *MockServiceHandler) GetHealthEndpoint() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockServiceHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.HealthStatus), args.Error(1)
}

func (m *MockServiceHandler) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockServiceHandler) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// TestServiceMetadata tests the HandlerMetadata struct
func TestServiceMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata *HandlerMetadata
		want     *HandlerMetadata
	}{
		{
			name: "complete metadata",
			metadata: &HandlerMetadata{
				Name:           "test-service",
				Version:        "v1.0.0",
				Description:    "Test service",
				HealthEndpoint: "/health",
				Tags:           []string{"test", "api"},
				Dependencies:   []string{"db", "cache"},
			},
			want: &HandlerMetadata{
				Name:           "test-service",
				Version:        "v1.0.0",
				Description:    "Test service",
				HealthEndpoint: "/health",
				Tags:           []string{"test", "api"},
				Dependencies:   []string{"db", "cache"},
			},
		},
		{
			name: "minimal metadata",
			metadata: &HandlerMetadata{
				Name:           "minimal-service",
				Version:        "v1.0.0",
				Description:    "Minimal service",
				HealthEndpoint: "/health",
			},
			want: &HandlerMetadata{
				Name:           "minimal-service",
				Version:        "v1.0.0",
				Description:    "Minimal service",
				HealthEndpoint: "/health",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.metadata)
		})
	}
}

// TestNewEnhancedServiceRegistry tests the creation of a new registry
func TestNewEnhancedServiceRegistry(t *testing.T) {
	registry := NewEnhancedServiceRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.handlers)
	assert.NotNil(t, registry.order)
	assert.Equal(t, 0, len(registry.handlers))
	assert.Equal(t, 0, len(registry.order))
	assert.True(t, registry.IsEmpty())
	assert.Equal(t, 0, registry.Count())
}

// TestEnhancedServiceRegistry_Register tests service registration
func TestEnhancedServiceRegistry_Register(t *testing.T) {
	tests := []struct {
		name       string
		handler    HandlerRegister
		wantErr    bool
		errorMsg   string
		setupMocks func(*MockServiceHandler)
	}{
		{
			name:    "successful registration",
			handler: NewMockServiceHandler("test-service", "v1.0.0", "Test service"),
			wantErr: false,
		},
		{
			name: "nil metadata",
			handler: func() HandlerRegister {
				mock := &MockServiceHandler{metadata: nil}
				return mock
			}(),
			wantErr:  true,
			errorMsg: "service handler metadata cannot be nil",
		},
		{
			name: "empty service name",
			handler: func() HandlerRegister {
				mock := &MockServiceHandler{
					metadata: &HandlerMetadata{
						Name:        "",
						Version:     "v1.0.0",
						Description: "Test service",
					},
				}
				return mock
			}(),
			wantErr:  true,
			errorMsg: "service name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewEnhancedServiceRegistry()
			err := registry.Register(tt.handler)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 1, registry.Count())
				assert.False(t, registry.IsEmpty())
			}
		})
	}
}

// TestEnhancedServiceRegistry_RegisterDuplicate tests duplicate service registration
func TestEnhancedServiceRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("test-service", "v1.0.0", "Test service 1")
	handler2 := NewMockServiceHandler("test-service", "v2.0.0", "Test service 2")

	// First registration should succeed
	err := registry.Register(handler1)
	assert.NoError(t, err)

	// Second registration with same name should fail
	err = registry.Register(handler2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service 'test-service' is already registered")
	assert.Equal(t, 1, registry.Count())
}

// TestEnhancedServiceRegistry_Unregister tests service unregistration
func TestEnhancedServiceRegistry_Unregister(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler := NewMockServiceHandler("test-service", "v1.0.0", "Test service")

	// Register a service first
	err := registry.Register(handler)
	assert.NoError(t, err)
	assert.Equal(t, 1, registry.Count())

	// Unregister the service
	err = registry.Unregister("test-service")
	assert.NoError(t, err)
	assert.Equal(t, 0, registry.Count())
	assert.True(t, registry.IsEmpty())

	// Try to unregister non-existent service
	err = registry.Unregister("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service 'non-existent' is not registered")
}

// TestEnhancedServiceRegistry_GetHandler tests getting a specific handler
func TestEnhancedServiceRegistry_GetHandler(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler := NewMockServiceHandler("test-service", "v1.0.0", "Test service")

	// Register a service
	err := registry.Register(handler)
	assert.NoError(t, err)

	// Get existing handler
	retrievedHandler, err := registry.GetHandler("test-service")
	assert.NoError(t, err)
	assert.Equal(t, handler, retrievedHandler)

	// Try to get non-existent handler
	_, err = registry.GetHandler("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service 'non-existent' is not registered")
}

// TestEnhancedServiceRegistry_GetAllHandlers tests getting all handlers
func TestEnhancedServiceRegistry_GetAllHandlers(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service1", "v1.0.0", "Service 1")
	handler2 := NewMockServiceHandler("service2", "v1.0.0", "Service 2")
	handler3 := NewMockServiceHandler("service3", "v1.0.0", "Service 3")

	// Register services in order
	assert.NoError(t, registry.Register(handler1))
	assert.NoError(t, registry.Register(handler2))
	assert.NoError(t, registry.Register(handler3))

	// Get all handlers
	handlers := registry.GetAllHandlers()
	assert.Equal(t, 3, len(handlers))
	assert.Equal(t, handler1, handlers[0])
	assert.Equal(t, handler2, handlers[1])
	assert.Equal(t, handler3, handlers[2])
}

// TestEnhancedServiceRegistry_GetServiceNames tests getting service names
func TestEnhancedServiceRegistry_GetServiceNames(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service1", "v1.0.0", "Service 1")
	handler2 := NewMockServiceHandler("service2", "v1.0.0", "Service 2")

	// Register services
	assert.NoError(t, registry.Register(handler1))
	assert.NoError(t, registry.Register(handler2))

	// Get service names
	names := registry.GetServiceNames()
	assert.Equal(t, 2, len(names))
	assert.Equal(t, "service1", names[0])
	assert.Equal(t, "service2", names[1])
}

// TestEnhancedServiceRegistry_GetServiceMetadata tests getting service metadata
func TestEnhancedServiceRegistry_GetServiceMetadata(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service1", "v1.0.0", "Service 1")
	handler2 := NewMockServiceHandler("service2", "v2.0.0", "Service 2")

	// Register services
	assert.NoError(t, registry.Register(handler1))
	assert.NoError(t, registry.Register(handler2))

	// Get service metadata
	metadata := registry.GetServiceMetadata()
	assert.Equal(t, 2, len(metadata))
	assert.Equal(t, "service1", metadata[0].Name)
	assert.Equal(t, "v1.0.0", metadata[0].Version)
	assert.Equal(t, "service2", metadata[1].Name)
	assert.Equal(t, "v2.0.0", metadata[1].Version)
}

// TestEnhancedServiceRegistry_InitializeAll tests initializing all services
func TestEnhancedServiceRegistry_InitializeAll(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func([]*MockServiceHandler)
		wantErr    bool
		errorMsg   string
	}{
		{
			name: "successful initialization",
			setupMocks: func(handlers []*MockServiceHandler) {
				for _, handler := range handlers {
					handler.On("Initialize", mock.Anything).Return(nil)
				}
			},
			wantErr: false,
		},
		{
			name: "initialization failure",
			setupMocks: func(handlers []*MockServiceHandler) {
				handlers[0].On("Initialize", mock.Anything).Return(nil)
				handlers[1].On("Initialize", mock.Anything).Return(errors.New("init failed"))
			},
			wantErr:  true,
			errorMsg: "failed to initialize service 'service2'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewEnhancedServiceRegistry()
			handler1 := NewMockServiceHandler("service1", "v1.0.0", "Service 1")
			handler2 := NewMockServiceHandler("service2", "v1.0.0", "Service 2")
			handlers := []*MockServiceHandler{handler1, handler2}

			tt.setupMocks(handlers)

			assert.NoError(t, registry.Register(handler1))
			assert.NoError(t, registry.Register(handler2))

			ctx := context.Background()
			err := registry.InitializeAll(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			for _, handler := range handlers {
				handler.AssertExpectations(t)
			}
		})
	}
}

// TestEnhancedServiceRegistry_RegisterAllHTTP tests HTTP registration for all services
func TestEnhancedServiceRegistry_RegisterAllHTTP(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func([]*MockServiceHandler)
		wantErr    bool
		errorMsg   string
	}{
		{
			name: "successful HTTP registration",
			setupMocks: func(handlers []*MockServiceHandler) {
				for _, handler := range handlers {
					handler.On("RegisterHTTP", mock.Anything).Return(nil)
				}
			},
			wantErr: false,
		},
		{
			name: "HTTP registration failure",
			setupMocks: func(handlers []*MockServiceHandler) {
				handlers[0].On("RegisterHTTP", mock.Anything).Return(nil)
				handlers[1].On("RegisterHTTP", mock.Anything).Return(errors.New("HTTP registration failed"))
			},
			wantErr:  true,
			errorMsg: "failed to register HTTP routes for service 'service2'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewEnhancedServiceRegistry()
			handler1 := NewMockServiceHandler("service1", "v1.0.0", "Service 1")
			handler2 := NewMockServiceHandler("service2", "v1.0.0", "Service 2")
			handlers := []*MockServiceHandler{handler1, handler2}

			tt.setupMocks(handlers)

			assert.NoError(t, registry.Register(handler1))
			assert.NoError(t, registry.Register(handler2))

			router := gin.New()
			err := registry.RegisterAllHTTP(router)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			for _, handler := range handlers {
				handler.AssertExpectations(t)
			}
		})
	}
}

// TestEnhancedServiceRegistry_RegisterAllGRPC tests gRPC registration for all services
func TestEnhancedServiceRegistry_RegisterAllGRPC(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func([]*MockServiceHandler)
		wantErr    bool
		errorMsg   string
	}{
		{
			name: "successful gRPC registration",
			setupMocks: func(handlers []*MockServiceHandler) {
				for _, handler := range handlers {
					handler.On("RegisterGRPC", mock.Anything).Return(nil)
				}
			},
			wantErr: false,
		},
		{
			name: "gRPC registration failure",
			setupMocks: func(handlers []*MockServiceHandler) {
				handlers[0].On("RegisterGRPC", mock.Anything).Return(nil)
				handlers[1].On("RegisterGRPC", mock.Anything).Return(errors.New("gRPC registration failed"))
			},
			wantErr:  true,
			errorMsg: "failed to register gRPC services for service 'service2'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewEnhancedServiceRegistry()
			handler1 := NewMockServiceHandler("service1", "v1.0.0", "Service 1")
			handler2 := NewMockServiceHandler("service2", "v1.0.0", "Service 2")
			handlers := []*MockServiceHandler{handler1, handler2}

			tt.setupMocks(handlers)

			assert.NoError(t, registry.Register(handler1))
			assert.NoError(t, registry.Register(handler2))

			server := grpc.NewServer()
			err := registry.RegisterAllGRPC(server)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			for _, handler := range handlers {
				handler.AssertExpectations(t)
			}
		})
	}
}

// TestEnhancedServiceRegistry_CheckAllHealth tests health checking for all services
func TestEnhancedServiceRegistry_CheckAllHealth(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	handler1 := NewMockServiceHandler("service1", "v1.0.0", "Service 1")
	handler2 := NewMockServiceHandler("service2", "v1.0.0", "Service 2")

	// Setup mocks
	healthyStatus := &types.HealthStatus{
		Status:    types.HealthStatusHealthy,
		Timestamp: time.Now(),
		Version:   "v1.0.0",
		Uptime:    time.Hour,
	}

	handler1.On("IsHealthy", mock.Anything).Return(healthyStatus, nil)
	handler2.On("IsHealthy", mock.Anything).Return(nil, errors.New("health check failed"))

	assert.NoError(t, registry.Register(handler1))
	assert.NoError(t, registry.Register(handler2))

	ctx := context.Background()
	results := registry.CheckAllHealth(ctx)

	assert.Equal(t, 2, len(results))
	assert.Equal(t, types.HealthStatusHealthy, results["service1"].Status)
	assert.Equal(t, types.HealthStatusUnhealthy, results["service2"].Status)

	handler1.AssertExpectations(t)
	handler2.AssertExpectations(t)
}

// TestEnhancedServiceRegistry_ShutdownAll tests shutdown for all services
func TestEnhancedServiceRegistry_ShutdownAll(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func([]*MockServiceHandler)
		wantErr    bool
		errorMsg   string
	}{
		{
			name: "successful shutdown",
			setupMocks: func(handlers []*MockServiceHandler) {
				for _, handler := range handlers {
					handler.On("Shutdown", mock.Anything).Return(nil)
				}
			},
			wantErr: false,
		},
		{
			name: "shutdown failure",
			setupMocks: func(handlers []*MockServiceHandler) {
				// Note: shutdown happens in reverse order, so service2 shuts down first
				handlers[1].On("Shutdown", mock.Anything).Return(errors.New("shutdown failed"))
			},
			wantErr:  true,
			errorMsg: "failed to shutdown service 'service2'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewEnhancedServiceRegistry()
			handler1 := NewMockServiceHandler("service1", "v1.0.0", "Service 1")
			handler2 := NewMockServiceHandler("service2", "v1.0.0", "Service 2")
			handlers := []*MockServiceHandler{handler1, handler2}

			tt.setupMocks(handlers)

			assert.NoError(t, registry.Register(handler1))
			assert.NoError(t, registry.Register(handler2))

			ctx := context.Background()
			err := registry.ShutdownAll(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			for _, handler := range handlers {
				handler.AssertExpectations(t)
			}
		})
	}
}

// TestEnhancedServiceRegistry_ConcurrentAccess tests concurrent access to the registry
func TestEnhancedServiceRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	var wg sync.WaitGroup
	const numGoroutines = 10

	// Concurrent registration
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			handler := NewMockServiceHandler(
				fmt.Sprintf("service%d", id),
				"v1.0.0",
				fmt.Sprintf("Service %d", id),
			)
			err := registry.Register(handler)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all services were registered
	assert.Equal(t, numGoroutines, registry.Count())
	assert.False(t, registry.IsEmpty())

	// Concurrent read operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			serviceName := fmt.Sprintf("service%d", id)
			handler, err := registry.GetHandler(serviceName)
			assert.NoError(t, err)
			assert.NotNil(t, handler)
			assert.Equal(t, serviceName, handler.GetMetadata().Name)
		}(i)
	}

	wg.Wait()
}

// TestEnhancedServiceRegistry_OrderPreservation tests that registration order is preserved
func TestEnhancedServiceRegistry_OrderPreservation(t *testing.T) {
	registry := NewEnhancedServiceRegistry()
	serviceNames := []string{"alpha", "beta", "gamma", "delta"}

	// Register services in specific order
	for _, name := range serviceNames {
		handler := NewMockServiceHandler(name, "v1.0.0", fmt.Sprintf("Service %s", name))
		assert.NoError(t, registry.Register(handler))
	}

	// Verify order is preserved
	retrievedNames := registry.GetServiceNames()
	assert.Equal(t, serviceNames, retrievedNames)

	// Verify handlers are returned in the same order
	handlers := registry.GetAllHandlers()
	for i, handler := range handlers {
		assert.Equal(t, serviceNames[i], handler.GetMetadata().Name)
	}

	// Verify metadata is returned in the same order
	metadata := registry.GetServiceMetadata()
	for i, meta := range metadata {
		assert.Equal(t, serviceNames[i], meta.Name)
	}
}

// TestEnhancedServiceRegistry_EmptyRegistry tests operations on empty registry
func TestEnhancedServiceRegistry_EmptyRegistry(t *testing.T) {
	registry := NewEnhancedServiceRegistry()

	// Test empty state
	assert.True(t, registry.IsEmpty())
	assert.Equal(t, 0, registry.Count())

	// Test operations on empty registry
	handlers := registry.GetAllHandlers()
	assert.Equal(t, 0, len(handlers))

	names := registry.GetServiceNames()
	assert.Equal(t, 0, len(names))

	metadata := registry.GetServiceMetadata()
	assert.Equal(t, 0, len(metadata))

	// Test error cases
	_, err := registry.GetHandler("non-existent")
	assert.Error(t, err)

	err = registry.Unregister("non-existent")
	assert.Error(t, err)

	// Test operations that should succeed on empty registry
	ctx := context.Background()
	err = registry.InitializeAll(ctx)
	assert.NoError(t, err)

	router := gin.New()
	err = registry.RegisterAllHTTP(router)
	assert.NoError(t, err)

	server := grpc.NewServer()
	err = registry.RegisterAllGRPC(server)
	assert.NoError(t, err)

	results := registry.CheckAllHealth(ctx)
	assert.Equal(t, 0, len(results))

	err = registry.ShutdownAll(ctx)
	assert.NoError(t, err)
}

// BenchmarkEnhancedServiceRegistry_Register benchmarks service registration
func BenchmarkEnhancedServiceRegistry_Register(b *testing.B) {
	registry := NewEnhancedServiceRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler := NewMockServiceHandler(
			fmt.Sprintf("service%d", i),
			"v1.0.0",
			fmt.Sprintf("Service %d", i),
		)
		_ = registry.Register(handler)
	}
}

// BenchmarkEnhancedServiceRegistry_GetHandler benchmarks handler retrieval
func BenchmarkEnhancedServiceRegistry_GetHandler(b *testing.B) {
	registry := NewEnhancedServiceRegistry()
	const numServices = 1000

	// Setup
	for i := 0; i < numServices; i++ {
		handler := NewMockServiceHandler(
			fmt.Sprintf("service%d", i),
			"v1.0.0",
			fmt.Sprintf("Service %d", i),
		)
		_ = registry.Register(handler)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		serviceName := fmt.Sprintf("service%d", i%numServices)
		_, _ = registry.GetHandler(serviceName)
	}
}

// BenchmarkEnhancedServiceRegistry_GetAllHandlers benchmarks getting all handlers
func BenchmarkEnhancedServiceRegistry_GetAllHandlers(b *testing.B) {
	registry := NewEnhancedServiceRegistry()
	const numServices = 100

	// Setup
	for i := 0; i < numServices; i++ {
		handler := NewMockServiceHandler(
			fmt.Sprintf("service%d", i),
			"v1.0.0",
			fmt.Sprintf("Service %d", i),
		)
		_ = registry.Register(handler)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.GetAllHandlers()
	}
}

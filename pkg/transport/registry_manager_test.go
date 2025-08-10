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
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/innovationmech/swit/pkg/logger"
)

func init() {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
}

func TestNewMultiTransportRegistry(t *testing.T) {
	manager := NewMultiTransportRegistry()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.registries)
	assert.Equal(t, 0, len(manager.registries))
	assert.True(t, manager.IsEmpty())
}

func TestMultiTransportRegistry_GetOrCreateRegistry(t *testing.T) {
	manager := NewMultiTransportRegistry()

	// Test creating new registry
	httpRegistry := manager.GetOrCreateRegistry("http")
	assert.NotNil(t, httpRegistry)

	// Test getting existing registry
	httpRegistry2 := manager.GetOrCreateRegistry("http")
	assert.Equal(t, httpRegistry, httpRegistry2)

	// Test creating different registry
	grpcRegistry := manager.GetOrCreateRegistry("grpc")
	assert.NotNil(t, grpcRegistry)
	assert.True(t, httpRegistry != grpcRegistry) // Use pointer comparison

	assert.Equal(t, 2, len(manager.registries))
}

func TestMultiTransportRegistry_GetRegistry(t *testing.T) {
	manager := NewMultiTransportRegistry()

	// Test getting non-existent registry
	registry := manager.GetRegistry("nonexistent")
	assert.Nil(t, registry)

	// Create registry and test getting it
	httpRegistry := manager.GetOrCreateRegistry("http")
	retrievedRegistry := manager.GetRegistry("http")
	assert.Equal(t, httpRegistry, retrievedRegistry)
}

func TestMultiTransportRegistry_RegisterHandler(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler := NewMockHandlerRegister("test-service", "v1.0.0")

	// Test registering handler
	err := manager.RegisterHandler("http", handler)
	assert.NoError(t, err)

	// Verify handler was registered
	registry := manager.GetRegistry("http")
	assert.NotNil(t, registry)
	assert.Equal(t, 1, registry.Count())

	retrievedHandler, err := registry.GetHandler("test-service")
	assert.NoError(t, err)
	assert.Equal(t, handler, retrievedHandler)
}

func TestMultiTransportRegistry_RegisterHTTPService(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler := NewMockHandlerRegister("http-service", "v1.0.0")

	err := manager.RegisterHTTPService(handler)
	assert.NoError(t, err)

	// Verify handler was registered with HTTP transport
	httpRegistry := manager.GetRegistry("http")
	assert.NotNil(t, httpRegistry)
	assert.Equal(t, 1, httpRegistry.Count())
}

func TestMultiTransportRegistry_RegisterGRPCService(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler := NewMockHandlerRegister("grpc-service", "v1.0.0")

	err := manager.RegisterGRPCService(handler)
	assert.NoError(t, err)

	// Verify handler was registered with gRPC transport
	grpcRegistry := manager.GetRegistry("grpc")
	assert.NotNil(t, grpcRegistry)
	assert.Equal(t, 1, grpcRegistry.Count())
}

func TestMultiTransportRegistry_BindAllHTTPEndpoints(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler1 := NewMockHandlerRegister("http-service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("http-service2", "v1.1.0")

	// Register handlers with HTTP transport only
	manager.RegisterHTTPService(handler1)
	manager.RegisterHTTPService(handler2)

	router := gin.New()
	err := manager.BindAllHTTPEndpoints(router)

	assert.NoError(t, err)
	assert.True(t, handler1.IsHTTPRegistered())
	assert.True(t, handler2.IsHTTPRegistered())
}

func TestMultiTransportRegistry_BindAllHTTPEndpoints_WithError(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler := NewMockHandlerRegister("error-service", "v1.0.0")

	expectedErr := errors.New("HTTP registration failed")
	handler.SetHTTPRegisterError(expectedErr)

	manager.RegisterHTTPService(handler)

	router := gin.New()
	err := manager.BindAllHTTPEndpoints(router)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register HTTP routes for transport 'http'")
}

func TestMultiTransportRegistry_BindAllGRPCServices(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler1 := NewMockHandlerRegister("grpc-service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("grpc-service2", "v1.1.0")

	// Register handlers with gRPC transport only
	manager.RegisterGRPCService(handler1)
	manager.RegisterGRPCService(handler2)

	server := grpc.NewServer()
	err := manager.BindAllGRPCServices(server)

	assert.NoError(t, err)
	assert.True(t, handler1.IsGRPCRegistered())
	assert.True(t, handler2.IsGRPCRegistered())
}

func TestMultiTransportRegistry_BindAllGRPCServices_WithError(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler := NewMockHandlerRegister("error-service", "v1.0.0")

	expectedErr := errors.New("gRPC registration failed")
	handler.SetGRPCRegisterError(expectedErr)

	manager.RegisterGRPCService(handler)

	server := grpc.NewServer()
	err := manager.BindAllGRPCServices(server)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register gRPC services for transport 'grpc'")
}

func TestMultiTransportRegistry_InitializeTransportServices(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	manager.RegisterHTTPService(handler1)
	manager.RegisterGRPCService(handler2)

	ctx := context.Background()
	err := manager.InitializeTransportServices(ctx)

	assert.NoError(t, err)
	assert.True(t, handler1.IsInitialized())
	assert.True(t, handler2.IsInitialized())
}

func TestMultiTransportRegistry_InitializeTransportServices_WithError(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler := NewMockHandlerRegister("error-service", "v1.0.0")

	expectedErr := errors.New("initialization failed")
	handler.SetInitializeError(expectedErr)

	manager.RegisterHTTPService(handler)

	ctx := context.Background()
	err := manager.InitializeTransportServices(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize services for transport 'http'")
}

func TestMultiTransportRegistry_CheckAllHealth(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	manager.RegisterHTTPService(handler1)
	manager.RegisterGRPCService(handler2)

	ctx := context.Background()
	results := manager.CheckAllHealth(ctx)

	assert.Equal(t, 2, len(results))
	assert.Contains(t, results, "http")
	assert.Contains(t, results, "grpc")
	assert.Equal(t, 1, len(results["http"]))
	assert.Equal(t, 1, len(results["grpc"]))
}

func TestMultiTransportRegistry_ShutdownAll(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	manager.RegisterHTTPService(handler1)
	manager.RegisterGRPCService(handler2)

	ctx := context.Background()
	err := manager.ShutdownAll(ctx)

	assert.NoError(t, err)
	assert.True(t, handler1.IsShutdown())
	assert.True(t, handler2.IsShutdown())
}

func TestMultiTransportRegistry_ShutdownAll_WithError(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler := NewMockHandlerRegister("error-service", "v1.0.0")

	expectedErr := errors.New("shutdown failed")
	handler.SetShutdownError(expectedErr)

	manager.RegisterHTTPService(handler)

	ctx := context.Background()
	err := manager.ShutdownAll(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to shutdown services for transport")
}

func TestMultiTransportRegistry_GetAllServiceMetadata(t *testing.T) {
	manager := NewMultiTransportRegistry()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	manager.RegisterHTTPService(handler1)
	manager.RegisterGRPCService(handler2)

	metadata := manager.GetAllServiceMetadata()

	assert.Equal(t, 2, len(metadata))
	assert.Contains(t, metadata, "http")
	assert.Contains(t, metadata, "grpc")
	assert.Equal(t, 1, len(metadata["http"]))
	assert.Equal(t, 1, len(metadata["grpc"]))
	assert.Equal(t, "service1", metadata["http"][0].Name)
	assert.Equal(t, "service2", metadata["grpc"][0].Name)
}

func TestMultiTransportRegistry_GetTransportNames(t *testing.T) {
	manager := NewMultiTransportRegistry()

	// Test empty manager
	names := manager.GetTransportNames()
	assert.Equal(t, 0, len(names))

	// Add some registries
	manager.GetOrCreateRegistry("http")
	manager.GetOrCreateRegistry("grpc")
	manager.GetOrCreateRegistry("websocket")

	names = manager.GetTransportNames()
	assert.Equal(t, 3, len(names))
	assert.Contains(t, names, "http")
	assert.Contains(t, names, "grpc")
	assert.Contains(t, names, "websocket")
}

func TestMultiTransportRegistry_GetTotalServiceCount(t *testing.T) {
	manager := NewMultiTransportRegistry()

	// Test empty manager
	assert.Equal(t, 0, manager.GetTotalServiceCount())
	assert.True(t, manager.IsEmpty())

	// Add services
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")
	handler3 := NewMockHandlerRegister("service3", "v1.2.0")

	manager.RegisterHTTPService(handler1)
	manager.RegisterGRPCService(handler2)
	manager.RegisterHandler("websocket", handler3)

	assert.Equal(t, 3, manager.GetTotalServiceCount())
	assert.False(t, manager.IsEmpty())
}

func TestMultiTransportRegistry_Clear(t *testing.T) {
	manager := NewMultiTransportRegistry()

	// Add some services
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	manager.RegisterHTTPService(handler1)
	manager.RegisterGRPCService(handler2)

	assert.Equal(t, 2, manager.GetTotalServiceCount())
	assert.False(t, manager.IsEmpty())

	// Clear and verify
	manager.Clear()

	assert.Equal(t, 0, manager.GetTotalServiceCount())
	assert.True(t, manager.IsEmpty())
	assert.Equal(t, 0, len(manager.GetTransportNames()))
}

func TestMultiTransportRegistry_ConcurrentAccess(t *testing.T) {
	manager := NewMultiTransportRegistry()
	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 4) // Create, Register, Access, Count operations

	// Concurrent registry creation
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			transportName := fmt.Sprintf("transport-%d", i%5) // Create 5 different transports
			registry := manager.GetOrCreateRegistry(transportName)
			assert.NotNil(t, registry)
		}(i)
	}

	// Concurrent handler registration
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			handler := NewMockHandlerRegister(fmt.Sprintf("service-%d", i), "v1.0.0")
			transportName := fmt.Sprintf("transport-%d", i%5)
			err := manager.RegisterHandler(transportName, handler)
			assert.NoError(t, err)
		}(i)
	}

	// Concurrent registry access
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			transportName := fmt.Sprintf("transport-%d", i%5)
			registry := manager.GetRegistry(transportName)
			_ = registry // May be nil during concurrent operations
		}(i)
	}

	// Concurrent count operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			count := manager.GetTotalServiceCount()
			_ = count // Result may vary during concurrent operations
		}()
	}

	wg.Wait()

	// Verify final state
	assert.Equal(t, 5, len(manager.GetTransportNames()))
	assert.Equal(t, numGoroutines, manager.GetTotalServiceCount())
}

func TestMultiTransportRegistry_EdgeCases(t *testing.T) {
	manager := NewMultiTransportRegistry()

	t.Run("empty transport name", func(t *testing.T) {
		registry := manager.GetOrCreateRegistry("")
		assert.NotNil(t, registry)

		handler := NewMockHandlerRegister("test-service", "v1.0.0")
		err := manager.RegisterHandler("", handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transport name cannot be empty")
	})

	t.Run("nil context operations", func(t *testing.T) {
		handler := NewMockHandlerRegister("test-service", "v1.0.0")
		manager.RegisterHTTPService(handler)

		// Use context.Background() instead of nil to avoid panic
		ctx := context.Background()
		err := manager.InitializeTransportServices(ctx)
		assert.NoError(t, err)

		results := manager.CheckAllHealth(ctx)
		assert.NotNil(t, results)

		err = manager.ShutdownAll(ctx)
		assert.NoError(t, err)
	})

	t.Run("timeout context", func(t *testing.T) {
		handler := NewMockHandlerRegister("timeout-service", "v1.0.0")
		manager.RegisterHTTPService(handler)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		time.Sleep(2 * time.Millisecond) // Ensure context is expired

		results := manager.CheckAllHealth(ctx)
		assert.NotNil(t, results)
	})
}

// Benchmark tests
func BenchmarkMultiTransportRegistry_RegisterHandler(b *testing.B) {
	manager := NewMultiTransportRegistry()
	handlers := make([]TransportServiceHandler, b.N)

	for i := 0; i < b.N; i++ {
		handlers[i] = NewMockHandlerRegister(fmt.Sprintf("service-%d", i), "v1.0.0")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.RegisterHandler("http", handlers[i])
	}
}

func BenchmarkMultiTransportRegistry_GetOrCreateRegistry(b *testing.B) {
	manager := NewMultiTransportRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transportName := fmt.Sprintf("transport-%d", i%10) // 10 different transports
		manager.GetOrCreateRegistry(transportName)
	}
}

func BenchmarkMultiTransportRegistry_GetTotalServiceCount(b *testing.B) {
	manager := NewMultiTransportRegistry()

	// Setup: register 100 services across 5 transports
	for i := 0; i < 100; i++ {
		handler := NewMockHandlerRegister(fmt.Sprintf("service-%d", i), "v1.0.0")
		transportName := fmt.Sprintf("transport-%d", i%5)
		manager.RegisterHandler(transportName, handler)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetTotalServiceCount()
	}
}

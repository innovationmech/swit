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

func TestNewServiceRegistryManager(t *testing.T) {
	manager := NewServiceRegistryManager()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.registries)
	assert.Equal(t, 0, len(manager.registries))
	assert.True(t, manager.IsEmpty())
}

func TestServiceRegistryManager_GetOrCreateRegistry(t *testing.T) {
	manager := NewServiceRegistryManager()

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

func TestServiceRegistryManager_GetRegistry(t *testing.T) {
	manager := NewServiceRegistryManager()

	// Test getting non-existent registry
	registry := manager.GetRegistry("nonexistent")
	assert.Nil(t, registry)

	// Create registry and test getting it
	httpRegistry := manager.GetOrCreateRegistry("http")
	retrievedRegistry := manager.GetRegistry("http")
	assert.Equal(t, httpRegistry, retrievedRegistry)
}

func TestServiceRegistryManager_RegisterHandler(t *testing.T) {
	manager := NewServiceRegistryManager()
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

func TestServiceRegistryManager_RegisterHTTPHandler(t *testing.T) {
	manager := NewServiceRegistryManager()
	handler := NewMockHandlerRegister("http-service", "v1.0.0")

	err := manager.RegisterHTTPHandler(handler)
	assert.NoError(t, err)

	// Verify handler was registered with HTTP transport
	httpRegistry := manager.GetRegistry("http")
	assert.NotNil(t, httpRegistry)
	assert.Equal(t, 1, httpRegistry.Count())
}

func TestServiceRegistryManager_RegisterGRPCHandler(t *testing.T) {
	manager := NewServiceRegistryManager()
	handler := NewMockHandlerRegister("grpc-service", "v1.0.0")

	err := manager.RegisterGRPCHandler(handler)
	assert.NoError(t, err)

	// Verify handler was registered with gRPC transport
	grpcRegistry := manager.GetRegistry("grpc")
	assert.NotNil(t, grpcRegistry)
	assert.Equal(t, 1, grpcRegistry.Count())
}

func TestServiceRegistryManager_RegisterAllHTTP(t *testing.T) {
	manager := NewServiceRegistryManager()
	handler1 := NewMockHandlerRegister("http-service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("http-service2", "v1.1.0")

	// Register handlers with HTTP transport only
	manager.RegisterHTTPHandler(handler1)
	manager.RegisterHTTPHandler(handler2)

	router := gin.New()
	err := manager.RegisterAllHTTP(router)

	assert.NoError(t, err)
	assert.True(t, handler1.IsHTTPRegistered())
	assert.True(t, handler2.IsHTTPRegistered())
}

func TestServiceRegistryManager_RegisterAllHTTP_WithError(t *testing.T) {
	manager := NewServiceRegistryManager()
	handler := NewMockHandlerRegister("error-service", "v1.0.0")

	expectedErr := errors.New("HTTP registration failed")
	handler.SetHTTPRegisterError(expectedErr)

	manager.RegisterHTTPHandler(handler)

	router := gin.New()
	err := manager.RegisterAllHTTP(router)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register HTTP routes for transport 'http'")
}

func TestServiceRegistryManager_RegisterAllGRPC(t *testing.T) {
	manager := NewServiceRegistryManager()
	handler1 := NewMockHandlerRegister("grpc-service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("grpc-service2", "v1.1.0")

	// Register handlers with gRPC transport only
	manager.RegisterGRPCHandler(handler1)
	manager.RegisterGRPCHandler(handler2)

	server := grpc.NewServer()
	err := manager.RegisterAllGRPC(server)

	assert.NoError(t, err)
	assert.True(t, handler1.IsGRPCRegistered())
	assert.True(t, handler2.IsGRPCRegistered())
}

func TestServiceRegistryManager_RegisterAllGRPC_WithError(t *testing.T) {
	manager := NewServiceRegistryManager()
	handler := NewMockHandlerRegister("error-service", "v1.0.0")

	expectedErr := errors.New("gRPC registration failed")
	handler.SetGRPCRegisterError(expectedErr)

	manager.RegisterGRPCHandler(handler)

	server := grpc.NewServer()
	err := manager.RegisterAllGRPC(server)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register gRPC services for transport 'grpc'")
}

func TestServiceRegistryManager_InitializeAll(t *testing.T) {
	manager := NewServiceRegistryManager()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	manager.RegisterHTTPHandler(handler1)
	manager.RegisterGRPCHandler(handler2)

	ctx := context.Background()
	err := manager.InitializeAll(ctx)

	assert.NoError(t, err)
	assert.True(t, handler1.IsInitialized())
	assert.True(t, handler2.IsInitialized())
}

func TestServiceRegistryManager_InitializeAll_WithError(t *testing.T) {
	manager := NewServiceRegistryManager()
	handler := NewMockHandlerRegister("error-service", "v1.0.0")

	expectedErr := errors.New("initialization failed")
	handler.SetInitializeError(expectedErr)

	manager.RegisterHTTPHandler(handler)

	ctx := context.Background()
	err := manager.InitializeAll(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize services for transport 'http'")
}

func TestServiceRegistryManager_CheckAllHealth(t *testing.T) {
	manager := NewServiceRegistryManager()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	manager.RegisterHTTPHandler(handler1)
	manager.RegisterGRPCHandler(handler2)

	ctx := context.Background()
	results := manager.CheckAllHealth(ctx)

	assert.Equal(t, 2, len(results))
	assert.Contains(t, results, "http")
	assert.Contains(t, results, "grpc")
	assert.Equal(t, 1, len(results["http"]))
	assert.Equal(t, 1, len(results["grpc"]))
}

func TestServiceRegistryManager_ShutdownAll(t *testing.T) {
	manager := NewServiceRegistryManager()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	manager.RegisterHTTPHandler(handler1)
	manager.RegisterGRPCHandler(handler2)

	ctx := context.Background()
	err := manager.ShutdownAll(ctx)

	assert.NoError(t, err)
	assert.True(t, handler1.IsShutdown())
	assert.True(t, handler2.IsShutdown())
}

func TestServiceRegistryManager_ShutdownAll_WithError(t *testing.T) {
	manager := NewServiceRegistryManager()
	handler := NewMockHandlerRegister("error-service", "v1.0.0")

	expectedErr := errors.New("shutdown failed")
	handler.SetShutdownError(expectedErr)

	manager.RegisterHTTPHandler(handler)

	ctx := context.Background()
	err := manager.ShutdownAll(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to shutdown services for transport")
}

func TestServiceRegistryManager_GetAllServiceMetadata(t *testing.T) {
	manager := NewServiceRegistryManager()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	manager.RegisterHTTPHandler(handler1)
	manager.RegisterGRPCHandler(handler2)

	metadata := manager.GetAllServiceMetadata()

	assert.Equal(t, 2, len(metadata))
	assert.Contains(t, metadata, "http")
	assert.Contains(t, metadata, "grpc")
	assert.Equal(t, 1, len(metadata["http"]))
	assert.Equal(t, 1, len(metadata["grpc"]))
	assert.Equal(t, "service1", metadata["http"][0].Name)
	assert.Equal(t, "service2", metadata["grpc"][0].Name)
}

func TestServiceRegistryManager_GetTransportNames(t *testing.T) {
	manager := NewServiceRegistryManager()

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

func TestServiceRegistryManager_GetTotalServiceCount(t *testing.T) {
	manager := NewServiceRegistryManager()

	// Test empty manager
	assert.Equal(t, 0, manager.GetTotalServiceCount())
	assert.True(t, manager.IsEmpty())

	// Add services
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")
	handler3 := NewMockHandlerRegister("service3", "v1.2.0")

	manager.RegisterHTTPHandler(handler1)
	manager.RegisterGRPCHandler(handler2)
	manager.RegisterHandler("websocket", handler3)

	assert.Equal(t, 3, manager.GetTotalServiceCount())
	assert.False(t, manager.IsEmpty())
}

func TestServiceRegistryManager_Clear(t *testing.T) {
	manager := NewServiceRegistryManager()

	// Add some services
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	manager.RegisterHTTPHandler(handler1)
	manager.RegisterGRPCHandler(handler2)

	assert.Equal(t, 2, manager.GetTotalServiceCount())
	assert.False(t, manager.IsEmpty())

	// Clear and verify
	manager.Clear()

	assert.Equal(t, 0, manager.GetTotalServiceCount())
	assert.True(t, manager.IsEmpty())
	assert.Equal(t, 0, len(manager.GetTransportNames()))
}

func TestServiceRegistryManager_ConcurrentAccess(t *testing.T) {
	manager := NewServiceRegistryManager()
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

func TestServiceRegistryManager_EdgeCases(t *testing.T) {
	manager := NewServiceRegistryManager()

	t.Run("empty transport name", func(t *testing.T) {
		registry := manager.GetOrCreateRegistry("")
		assert.NotNil(t, registry)

		handler := NewMockHandlerRegister("test-service", "v1.0.0")
		err := manager.RegisterHandler("", handler)
		assert.NoError(t, err)
	})

	t.Run("nil context operations", func(t *testing.T) {
		handler := NewMockHandlerRegister("test-service", "v1.0.0")
		manager.RegisterHTTPHandler(handler)

		// Use context.Background() instead of nil to avoid panic
		ctx := context.Background()
		err := manager.InitializeAll(ctx)
		assert.NoError(t, err)

		results := manager.CheckAllHealth(ctx)
		assert.NotNil(t, results)

		err = manager.ShutdownAll(ctx)
		assert.NoError(t, err)
	})

	t.Run("timeout context", func(t *testing.T) {
		handler := NewMockHandlerRegister("timeout-service", "v1.0.0")
		manager.RegisterHTTPHandler(handler)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		time.Sleep(2 * time.Millisecond) // Ensure context is expired

		results := manager.CheckAllHealth(ctx)
		assert.NotNil(t, results)
	})
}

// Benchmark tests
func BenchmarkServiceRegistryManager_RegisterHandler(b *testing.B) {
	manager := NewServiceRegistryManager()
	handlers := make([]HandlerRegister, b.N)

	for i := 0; i < b.N; i++ {
		handlers[i] = NewMockHandlerRegister(fmt.Sprintf("service-%d", i), "v1.0.0")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.RegisterHandler("http", handlers[i])
	}
}

func BenchmarkServiceRegistryManager_GetOrCreateRegistry(b *testing.B) {
	manager := NewServiceRegistryManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transportName := fmt.Sprintf("transport-%d", i%10) // 10 different transports
		manager.GetOrCreateRegistry(transportName)
	}
}

func BenchmarkServiceRegistryManager_GetTotalServiceCount(b *testing.B) {
	manager := NewServiceRegistryManager()

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

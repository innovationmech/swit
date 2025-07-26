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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

func init() {
	// Set gin to test mode to reduce noise in test output
	gin.SetMode(gin.TestMode)
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
}

func TestNewHTTPTransport(t *testing.T) {
	transport := NewHTTPTransport()

	assert.NotNil(t, transport)
	assert.NotNil(t, transport.router)
	assert.NotNil(t, transport.serviceRegistry)
	assert.Equal(t, "http", transport.GetName())
}

func TestNewHTTPTransportWithConfig(t *testing.T) {
	config := &HTTPTransportConfig{
		Address:     ":9999",
		Port:        "9999",
		EnableReady: false,
	}

	transport := NewHTTPTransportWithConfig(config)

	assert.NotNil(t, transport)
	assert.Equal(t, config, transport.config)
	assert.Equal(t, "http", transport.GetName())
}

func TestNewHTTPTransportWithConfig_NilConfig(t *testing.T) {
	transport := NewHTTPTransportWithConfig(nil)

	assert.NotNil(t, transport)
	assert.NotNil(t, transport.config)
	assert.Equal(t, ":8080", transport.config.Address)
	assert.True(t, transport.config.EnableReady)
}

func TestHTTPTransport_Start_Success(t *testing.T) {
	config := &HTTPTransportConfig{
		Address:     ":0", // Use dynamic port
		EnableReady: true,
	}
	transport := NewHTTPTransportWithConfig(config)

	ctx := context.Background()
	err := transport.Start(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, transport.GetAddress())
	assert.True(t, transport.GetPort() > 0)

	// Wait for ready signal
	select {
	case <-transport.WaitReady():
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for ready signal")
	}

	// Clean up
	err = transport.Stop(context.Background())
	assert.NoError(t, err)
}

func TestHTTPTransport_Start_AlreadyStarted(t *testing.T) {
	config := &HTTPTransportConfig{
		Address:     ":0",
		EnableReady: true,
	}
	transport := NewHTTPTransportWithConfig(config)

	ctx := context.Background()

	// Start first time
	err := transport.Start(ctx)
	assert.NoError(t, err)

	// Start second time - should not error
	err = transport.Start(ctx)
	assert.NoError(t, err)

	// Clean up
	err = transport.Stop(context.Background())
	assert.NoError(t, err)
}

func TestHTTPTransport_Stop_Success(t *testing.T) {
	config := &HTTPTransportConfig{
		Address:     ":0",
		EnableReady: true,
	}
	transport := NewHTTPTransportWithConfig(config)

	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	<-transport.WaitReady()

	err = transport.Stop(ctx)
	assert.NoError(t, err)
}

func TestHTTPTransport_Stop_NotStarted(t *testing.T) {
	transport := NewHTTPTransport()

	ctx := context.Background()
	err := transport.Stop(ctx)
	assert.NoError(t, err) // Should not error when stopping non-started transport
}

func TestHTTPTransport_GetPort(t *testing.T) {
	config := &HTTPTransportConfig{
		Address:     ":0",
		EnableReady: true,
	}
	transport := NewHTTPTransportWithConfig(config)

	// Before start
	assert.Equal(t, 0, transport.GetPort())

	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	<-transport.WaitReady()

	// After start
	port := transport.GetPort()
	assert.True(t, port > 0)

	// Clean up
	transport.Stop(context.Background())
}

func TestHTTPTransport_SetTestPort(t *testing.T) {
	transport := NewHTTPTransport()

	transport.SetTestPort("9999")

	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	assert.Contains(t, transport.GetAddress(), ":9999")

	// Clean up
	transport.Stop(context.Background())
}

func TestHTTPTransport_SetAddress(t *testing.T) {
	transport := NewHTTPTransport()

	transport.SetAddress(":7777")

	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	assert.Contains(t, transport.GetAddress(), ":7777")

	// Clean up
	transport.Stop(context.Background())
}

func TestHTTPTransport_RegisterHandler_Success(t *testing.T) {
	transport := NewHTTPTransport()
	handler := NewMockHandlerRegister("test-service", "v1.0.0")

	err := transport.RegisterHandler(handler)
	assert.NoError(t, err)

	registry := transport.GetServiceRegistry()
	assert.Equal(t, 1, registry.Count())

	retrieved, err := registry.GetHandler("test-service")
	assert.NoError(t, err)
	assert.Equal(t, handler, retrieved)
}

func TestHTTPTransport_RegisterService_Alias(t *testing.T) {
	transport := NewHTTPTransport()
	handler := NewMockHandlerRegister("test-service", "v1.0.0")

	err := transport.RegisterService(handler) // Test alias method
	assert.NoError(t, err)

	registry := transport.GetServiceRegistry()
	assert.Equal(t, 1, registry.Count())
}

func TestHTTPTransport_InitializeServices_Success(t *testing.T) {
	transport := NewHTTPTransport()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	transport.RegisterHandler(handler1)
	transport.RegisterHandler(handler2)

	ctx := context.Background()
	err := transport.InitializeServices(ctx)

	assert.NoError(t, err)
	assert.True(t, handler1.IsInitialized())
	assert.True(t, handler2.IsInitialized())
}

func TestHTTPTransport_RegisterAllRoutes_Success(t *testing.T) {
	transport := NewHTTPTransport()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	transport.RegisterHandler(handler1)
	transport.RegisterHandler(handler2)

	err := transport.RegisterAllRoutes()

	assert.NoError(t, err)
	assert.True(t, handler1.IsHTTPRegistered())
	assert.True(t, handler2.IsHTTPRegistered())
}

func TestHTTPTransport_ShutdownServices_Success(t *testing.T) {
	transport := NewHTTPTransport()
	handler1 := NewMockHandlerRegister("service1", "v1.0.0")
	handler2 := NewMockHandlerRegister("service2", "v1.1.0")

	transport.RegisterHandler(handler1)
	transport.RegisterHandler(handler2)

	ctx := context.Background()
	err := transport.ShutdownServices(ctx)

	assert.NoError(t, err)
	assert.True(t, handler1.IsShutdown())
	assert.True(t, handler2.IsShutdown())
}

func TestHTTPTransport_WaitReady_DisabledReady(t *testing.T) {
	config := &HTTPTransportConfig{
		Address:     ":0",
		EnableReady: false,
	}
	transport := NewHTTPTransportWithConfig(config)

	// Should return closed channel immediately
	select {
	case <-transport.WaitReady():
		// Success - channel is closed
	case <-time.After(100 * time.Millisecond):
		t.Fatal("WaitReady should return immediately when ready is disabled")
	}
}

func TestHTTPTransport_FullLifecycle(t *testing.T) {
	config := &HTTPTransportConfig{
		Address:     ":0",
		EnableReady: true,
	}
	transport := NewHTTPTransportWithConfig(config)
	handler := NewMockHandlerRegister("test-service", "v1.0.0")

	// Register handler
	err := transport.RegisterHandler(handler)
	require.NoError(t, err)

	// Initialize services
	ctx := context.Background()
	err = transport.InitializeServices(ctx)
	require.NoError(t, err)

	// Register routes
	err = transport.RegisterAllRoutes()
	require.NoError(t, err)

	// Start transport
	err = transport.Start(ctx)
	require.NoError(t, err)

	<-transport.WaitReady()

	// Verify server is running
	client := &http.Client{Timeout: 5 * time.Second}
	addr := transport.GetAddress()
	if strings.HasPrefix(addr, "[::") {
		// IPv6 address, use localhost with port only
		port := strings.Split(addr, "]:")
		if len(port) == 2 {
			addr = ":" + port[1]
		}
	}
	resp, err := client.Get("http://localhost" + addr + "/nonexistent")
	if err == nil {
		resp.Body.Close()
		// Server is responsive (404 is expected for nonexistent route)
	}

	// Stop transport
	err = transport.Stop(ctx)
	assert.NoError(t, err)

	// Verify services were shut down
	assert.True(t, handler.IsShutdown())
}

func TestHTTPTransport_ConcurrentAccess(t *testing.T) {
	transport := NewHTTPTransport()
	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3)

	// Concurrent RegisterHandler operations
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			handler := NewMockHandlerRegister(fmt.Sprintf("service-%d", i), "v1.0.0")
			_ = transport.RegisterHandler(handler) // Ignore errors in concurrent test
		}(i)
	}

	// Concurrent GetRouter operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			router := transport.GetRouter()
			assert.NotNil(t, router)
		}()
	}

	// Concurrent GetServiceRegistry operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			registry := transport.GetServiceRegistry()
			assert.NotNil(t, registry)
		}()
	}

	wg.Wait()

	// Verify no race conditions
	registry := transport.GetServiceRegistry()
	assert.Equal(t, numGoroutines, registry.Count())
}

func TestHTTPTransport_StartStop_Multiple(t *testing.T) {
	config := &HTTPTransportConfig{
		Address:     ":0",
		EnableReady: true,
	}
	transport := NewHTTPTransportWithConfig(config)

	ctx := context.Background()

	// Test multiple start/stop cycles
	for i := 0; i < 3; i++ {
		err := transport.Start(ctx)
		require.NoError(t, err)

		<-transport.WaitReady()

		err = transport.Stop(ctx)
		require.NoError(t, err)
	}
}

func TestHTTPTransport_GetRouter(t *testing.T) {
	transport := NewHTTPTransport()

	router1 := transport.GetRouter()
	router2 := transport.GetRouter()

	assert.NotNil(t, router1)
	assert.Equal(t, router1, router2) // Should return same router instance
}

func TestHTTPTransport_ActualHTTPRequests(t *testing.T) {
	// Create a transport that can handle real HTTP requests
	config := &HTTPTransportConfig{
		Address:     ":0",
		EnableReady: true,
	}
	transport := NewHTTPTransportWithConfig(config)

	// Add a test route
	router := transport.GetRouter()
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test response"})
	})

	// Start the transport
	ctx := context.Background()
	err := transport.Start(ctx)
	require.NoError(t, err)

	<-transport.WaitReady()

	// Make an actual HTTP request
	client := &http.Client{Timeout: 5 * time.Second}
	addr := transport.GetAddress()
	if strings.HasPrefix(addr, "[::") {
		// IPv6 address, use localhost with port only
		port := strings.Split(addr, "]:")
		if len(port) == 2 {
			addr = ":" + port[1]
		}
	}
	resp, err := client.Get("http://localhost" + addr + "/test")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Clean up
	err = transport.Stop(ctx)
	assert.NoError(t, err)
}

func TestHTTPTransport_determineAddress(t *testing.T) {
	tests := []struct {
		name         string
		config       *HTTPTransportConfig
		expectedAddr string
	}{
		{
			name: "test port override",
			config: &HTTPTransportConfig{
				Address:  ":8080",
				TestMode: true,
				TestPort: "9999",
			},
			expectedAddr: ":9999",
		},
		{
			name: "configured address",
			config: &HTTPTransportConfig{
				Address: ":7777",
			},
			expectedAddr: ":7777",
		},
		{
			name:         "nil config",
			config:       nil,
			expectedAddr: ":8080",
		},
		{
			name: "empty address",
			config: &HTTPTransportConfig{
				Address: "",
			},
			expectedAddr: ":8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewHTTPTransportWithConfig(tt.config)
			addr := transport.determineAddress()
			assert.Equal(t, tt.expectedAddr, addr)
		})
	}
}

// Benchmark tests
func BenchmarkHTTPTransport_RegisterHandler(b *testing.B) {
	transport := NewHTTPTransport()
	handlers := make([]*MockHandlerRegister, b.N)

	for i := 0; i < b.N; i++ {
		handlers[i] = NewMockHandlerRegister(fmt.Sprintf("service-%d", i), "v1.0.0")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transport.RegisterHandler(handlers[i])
	}
}

func BenchmarkHTTPTransport_GetRouter(b *testing.B) {
	transport := NewHTTPTransport()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transport.GetRouter()
	}
}

func BenchmarkHTTPTransport_GetServiceRegistry(b *testing.B) {
	transport := NewHTTPTransport()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transport.GetServiceRegistry()
	}
}

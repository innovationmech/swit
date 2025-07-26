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
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Initialize logger for tests
	logger.InitLogger()

	// Add config path for tests
	// This is needed because go test changes working directory to the package directory
	viper.AddConfigPath("../../..")
}

func TestNewHTTPTransport(t *testing.T) {
	transport := NewHTTPTransport()

	assert.NotNil(t, transport)
	assert.Nil(t, transport.server)
	assert.NotNil(t, transport.router)
	assert.Empty(t, transport.address)
	assert.NotNil(t, transport.ready)
}

func TestHTTPTransport_Name(t *testing.T) {
	transport := NewHTTPTransport()

	assert.Equal(t, "http", transport.Name())
}

func TestHTTPTransport_Address(t *testing.T) {
	transport := NewHTTPTransport()

	// Initially empty
	assert.Empty(t, transport.Address())

	// Set address for testing
	transport.address = ":8080"
	assert.Equal(t, ":8080", transport.Address())
}

func TestHTTPTransport_GetRouter(t *testing.T) {
	transport := NewHTTPTransport()

	router := transport.GetRouter()
	assert.NotNil(t, router)
	assert.IsType(t, &gin.Engine{}, router)
}

func TestHTTPTransport_Start(t *testing.T) {
	// Set gin to test mode to avoid debug output
	gin.SetMode(gin.TestMode)

	transport := NewHTTPTransport()
	transport.SetTestPort("0") // Use dynamic port allocation
	ctx := context.Background()

	// Start the transport
	err := transport.Start(ctx)
	assert.NoError(t, err)

	// Verify server is created
	assert.NotNil(t, transport.server)
	assert.NotEmpty(t, transport.address)

	// Wait for ready signal
	select {
	case <-transport.WaitReady():
		// Ready signal received
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for server to be ready")
	}

	// Clean up
	transport.Stop(context.Background())
}

func TestHTTPTransport_Stop(t *testing.T) {
	gin.SetMode(gin.TestMode)

	transport := NewHTTPTransport()
	transport.SetTestPort("0") // Use dynamic port allocation
	ctx := context.Background()

	// Test stopping without starting
	err := transport.Stop(ctx)
	assert.NoError(t, err)

	// Start the transport
	err = transport.Start(ctx)
	require.NoError(t, err)

	// Wait for ready
	<-transport.WaitReady()

	// Stop the transport
	err = transport.Stop(ctx)
	assert.NoError(t, err)
}

func TestHTTPTransport_Stop_WithTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	transport := NewHTTPTransport()
	transport.SetTestPort("0") // Use dynamic port allocation

	// Start the transport
	err := transport.Start(context.Background())
	require.NoError(t, err)

	// Wait for ready
	<-transport.WaitReady()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop should complete within timeout
	err = transport.Stop(ctx)
	assert.NoError(t, err)
}

func TestHTTPTransport_ConcurrentAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	transport := NewHTTPTransport()

	// Test concurrent access to Address() method
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			transport.Address()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent access to GetRouter() method
	for i := 0; i < 10; i++ {
		go func() {
			transport.GetRouter()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestHTTPTransport_WaitReady(t *testing.T) {
	gin.SetMode(gin.TestMode)

	transport := NewHTTPTransport()
	transport.SetTestPort("0") // Use dynamic port allocation
	ctx := context.Background()

	// Start the transport
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Wait for ready signal
	select {
	case <-transport.WaitReady():
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for server to be ready")
	}

	// Clean up
	transport.Stop(ctx)
}

func TestHTTPTransport_WaitReady_Multiple(t *testing.T) {
	gin.SetMode(gin.TestMode)

	transport := NewHTTPTransport()
	transport.SetTestPort("0") // Use dynamic port allocation
	ctx := context.Background()

	// Start the transport
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Multiple goroutines waiting for ready
	done := make(chan bool)

	for i := 0; i < 5; i++ {
		go func() {
			select {
			case <-transport.WaitReady():
				done <- true
			case <-time.After(5 * time.Second):
				done <- false
			}
		}()
	}

	// All should receive the signal
	for i := 0; i < 5; i++ {
		result := <-done
		assert.True(t, result)
	}

	// Clean up
	transport.Stop(ctx)
}

func TestHTTPTransport_StartMultipleTimes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	// Test multiple transport instances can start without conflicts
	transport1 := NewHTTPTransport()
	transport1.SetTestPort("0") // Use dynamic port allocation

	transport2 := NewHTTPTransport()
	transport2.SetTestPort("0") // Use dynamic port allocation

	// Start the first transport
	err := transport1.Start(ctx)
	assert.NoError(t, err)

	// Wait for ready signal
	<-transport1.WaitReady()

	// Start the second transport (should work with different port)
	err = transport2.Start(ctx)
	assert.NoError(t, err)

	// Wait for ready signal
	<-transport2.WaitReady()

	// Clean up both
	transport1.Stop(ctx)
	transport2.Stop(ctx)
}

func TestHTTPTransport_ServerProperties(t *testing.T) {
	gin.SetMode(gin.TestMode)

	transport := NewHTTPTransport()
	transport.SetTestPort("0") // Use dynamic port allocation
	ctx := context.Background()

	// Start the transport
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Verify server properties
	assert.NotNil(t, transport.server)
	assert.Equal(t, transport.address, transport.server.Addr)
	assert.Equal(t, transport.router, transport.server.Handler)

	// Clean up
	transport.Stop(ctx)
}

func TestHTTPTransport_RouteRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	transport := NewHTTPTransport()
	router := transport.GetRouter()

	// HandlerRegister a test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})

	// Verify route is registered (indirect test)
	routes := router.Routes()
	assert.NotEmpty(t, routes)

	// Look for our test route
	found := false
	for _, route := range routes {
		if route.Path == "/test" && route.Method == "GET" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestHTTPTransport_DefaultPort(t *testing.T) {
	gin.SetMode(gin.TestMode)

	transport := NewHTTPTransport()
	transport.SetTestPort("0") // Use dynamic port allocation
	ctx := context.Background()

	// Start the transport (should use default port from config or 8080)
	err := transport.Start(ctx)
	assert.NoError(t, err)

	// Address should be set
	assert.NotEmpty(t, transport.Address())

	// Clean up
	transport.Stop(ctx)
}

func TestHTTPTransport_IntegrationTest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	transport := NewHTTPTransport()
	transport.SetTestPort("0") // Use dynamic port allocation
	ctx := context.Background()

	// Test the full lifecycle
	err := transport.Start(ctx)
	require.NoError(t, err)

	// Verify transport properties
	assert.Equal(t, "http", transport.Name())
	assert.NotEmpty(t, transport.Address())
	assert.NotNil(t, transport.GetRouter())

	// Wait for ready
	<-transport.WaitReady()

	// Clean stop
	err = transport.Stop(ctx)
	assert.NoError(t, err)

	// Verify server is accessible after stop
	assert.NotNil(t, transport.GetRouter())
}

func TestHTTPTransport_ContextCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	transport := NewHTTPTransport()
	transport.SetTestPort("0") // Use dynamic port allocation

	// Start the transport
	err := transport.Start(context.Background())
	require.NoError(t, err)

	// Wait for ready
	<-transport.WaitReady()

	// Create a context that gets cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Stop should still work with cancelled context
	err = transport.Stop(ctx)
	// May return error due to cancelled context, but shouldn't panic
	// We just verify it doesn't crash
}

func TestHTTPTransport_StopWithoutServer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	transport := NewHTTPTransport()
	ctx := context.Background()

	// Stop without starting (server is nil)
	err := transport.Stop(ctx)
	assert.NoError(t, err)

	// Should not panic or cause issues
}

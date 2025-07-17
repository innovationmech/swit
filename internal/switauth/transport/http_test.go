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

package transport_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switauth/transport"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	// Initialize logger for tests
	logger.Logger, _ = zap.NewDevelopment()
}

func TestNewHTTPTransport(t *testing.T) {
	httpTransport := transport.NewHTTPTransport()
	assert.NotNil(t, httpTransport)
	assert.Equal(t, "http", httpTransport.GetName())
	assert.NotNil(t, httpTransport.GetRouter())
}

func TestHTTPTransport_GetName(t *testing.T) {
	httpTransport := transport.NewHTTPTransport()
	assert.Equal(t, "http", httpTransport.GetName())
}

func TestHTTPTransport_SetAndGetAddress(t *testing.T) {
	httpTransport := transport.NewHTTPTransport()

	// Test default address
	assert.Empty(t, httpTransport.GetAddress())

	// Test setting address
	addr := ":8090"
	httpTransport.SetAddress(addr)
	assert.Equal(t, addr, httpTransport.GetAddress())
}

func TestHTTPTransport_GetRouter(t *testing.T) {
	httpTransport := transport.NewHTTPTransport()
	router := httpTransport.GetRouter()
	assert.NotNil(t, router)
	assert.IsType(t, &gin.Engine{}, router)
}

func TestHTTPTransport_GetPort(t *testing.T) {
	httpTransport := transport.NewHTTPTransport()

	// Test empty address
	assert.Equal(t, 0, httpTransport.GetPort())

	// Test valid address
	httpTransport.SetAddress(":8090")
	assert.Equal(t, 8090, httpTransport.GetPort())

	// Test invalid address
	httpTransport.SetAddress("invalid")
	assert.Equal(t, 0, httpTransport.GetPort())
}

func TestHTTPTransport_StartAndStop(t *testing.T) {
	// Set gin to test mode to reduce log output
	gin.SetMode(gin.TestMode)

	httpTransport := transport.NewHTTPTransport()
	httpTransport.SetAddress(":0") // Use random port

	// Add a simple route for testing
	router := httpTransport.GetRouter()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	ctx := context.Background()

	// Test start
	err := httpTransport.Start(ctx)
	require.NoError(t, err)

	// Verify server is running
	assert.NotEmpty(t, httpTransport.GetAddress())
	assert.Greater(t, httpTransport.GetPort(), 0)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test stop
	err = httpTransport.Stop(ctx)
	assert.NoError(t, err)
}

func TestHTTPTransport_StartWithDefaultPort(t *testing.T) {
	// Set gin to test mode to reduce log output
	gin.SetMode(gin.TestMode)

	httpTransport := transport.NewHTTPTransport()
	// Don't set address, should use default :8080

	ctx := context.Background()

	// Test start with default port
	err := httpTransport.Start(ctx)
	require.NoError(t, err)

	// Verify default port is used
	assert.Contains(t, httpTransport.GetAddress(), ":8080")

	// Clean up
	err = httpTransport.Stop(ctx)
	assert.NoError(t, err)
}

func TestHTTPTransport_StopWithoutStart(t *testing.T) {
	httpTransport := transport.NewHTTPTransport()
	ctx := context.Background()

	// Test stop without start should not error
	err := httpTransport.Stop(ctx)
	assert.NoError(t, err)
}

func TestHTTPTransport_StartError(t *testing.T) {
	httpTransport := transport.NewHTTPTransport()
	httpTransport.SetAddress("invalid-address")

	ctx := context.Background()

	// Test start with invalid address
	err := httpTransport.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create HTTP listener")
}

func TestHTTPTransport_ConcurrentAccess(t *testing.T) {
	httpTransport := transport.NewHTTPTransport()

	// Test concurrent access to address and router
	go func() {
		for i := 0; i < 100; i++ {
			httpTransport.SetAddress(":8090")
		}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = httpTransport.GetAddress()
		}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = httpTransport.GetRouter()
		}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = httpTransport.GetPort()
		}
	}()

	// Give goroutines time to complete
	time.Sleep(100 * time.Millisecond)

	// Should not panic
	assert.Equal(t, ":8090", httpTransport.GetAddress())
	assert.NotNil(t, httpTransport.GetRouter())
}

func TestHTTPTransport_RouterFunctionality(t *testing.T) {
	// Set gin to test mode to reduce log output
	gin.SetMode(gin.TestMode)

	httpTransport := transport.NewHTTPTransport()
	router := httpTransport.GetRouter()

	// Add routes to test router functionality
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	router.POST("/data", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"created": true})
	})

	// Verify routes are registered (this is more of a smoke test)
	assert.NotNil(t, router)
	assert.IsType(t, &gin.Engine{}, router)
}

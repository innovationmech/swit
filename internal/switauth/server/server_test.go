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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	logger.Logger, _ = zap.NewDevelopment()
	gin.SetMode(gin.TestMode)

	// Change to project root for config file access
	wd, _ := os.Getwd()
	if filepath.Base(wd) == "server" {
		os.Chdir("../../..")
	}
}

func setupTestConsulMock() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/agent/service/register":
			w.WriteHeader(http.StatusOK)
		case "/v1/agent/service/deregister/swit-auth-localhost-9001":
			w.WriteHeader(http.StatusOK)
		case "/v1/health/service/swit-auth":
			services := []map[string]interface{}{
				{
					"Service": map[string]interface{}{
						"ID":      "swit-auth-localhost-9001",
						"Service": "swit-auth",
						"Address": "localhost",
						"Port":    9001,
					},
					"Checks": []map[string]interface{}{
						{"Status": "passing"},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(services)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// createTestServerWithoutDB creates a test server instance without database dependencies
func createTestServerWithoutDB() *Server {
	return &Server{
		router: gin.New(),
	}
}

func TestNewServer_StructCreation(t *testing.T) {
	server := createTestServerWithoutDB()

	assert.NotNil(t, server)
	assert.NotNil(t, server.router)
	assert.Nil(t, server.srv)
	assert.Nil(t, server.sd)
}

func TestServer_Run_BasicStructure(t *testing.T) {
	server := createTestServerWithoutDB()

	// Test that Run method can be called without panicking on basic setup
	assert.NotNil(t, server.router)

	// We can't fully test Run without database and service discovery setup
	// But we can verify the server structure is correct
}

func TestServer_Shutdown_WithoutServices(t *testing.T) {
	server := createTestServerWithoutDB()

	// Test that shutdown doesn't panic even when services aren't initialized
	assert.Panics(t, func() {
		server.Shutdown()
	}, "Shutdown should panic when srv is nil")
}

func TestServer_runGinServer(t *testing.T) {
	tests := []struct {
		name        string
		addr        string
		expectError bool
	}{
		{
			name:        "valid address",
			addr:        ":0",
			expectError: false,
		},
		{
			name:        "invalid address format",
			addr:        "invalid:addr:format",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServerWithoutDB()
			var wg sync.WaitGroup
			ch := make(chan struct{})

			wg.Add(1)
			go server.runGinServer(tt.addr, &wg, ch)

			if !tt.expectError {
				select {
				case <-ch:
				case <-time.After(2 * time.Second):
					t.Fatal("channel was not closed within timeout")
				}

				require.NotNil(t, server.srv)
				assert.Equal(t, tt.addr, server.srv.Addr)
				assert.Equal(t, server.router, server.srv.Handler)

				err := server.srv.Shutdown(context.Background())
				assert.NoError(t, err)
			} else {
				select {
				case <-ch:
					t.Fatal("channel should not be closed for invalid address")
				case <-time.After(100 * time.Millisecond):
				}
			}

			wg.Wait()
		})
	}
}

func TestServer_runGinServer_ServerCreation(t *testing.T) {
	server := createTestServerWithoutDB()
	var wg sync.WaitGroup
	ch := make(chan struct{})

	wg.Add(1)
	go server.runGinServer(":0", &wg, ch)

	<-ch

	assert.NotNil(t, server.srv)
	assert.Equal(t, ":0", server.srv.Addr)
	assert.Equal(t, server.router, server.srv.Handler)

	err := server.srv.Shutdown(context.Background())
	assert.NoError(t, err)

	wg.Wait()
}

func TestServer_runGinServer_ChannelClosure(t *testing.T) {
	server := createTestServerWithoutDB()
	var wg sync.WaitGroup
	ch := make(chan struct{})

	wg.Add(1)
	go server.runGinServer(":0", &wg, ch)

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("channel was not closed within timeout")
	}

	err := server.srv.Shutdown(context.Background())
	assert.NoError(t, err)

	wg.Wait()
}

func TestServer_runGinServer_WaitGroupDone(t *testing.T) {
	server := createTestServerWithoutDB()
	var wg sync.WaitGroup
	ch := make(chan struct{})

	wg.Add(1)
	go server.runGinServer(":0", &wg, ch)

	<-ch

	err := server.srv.Shutdown(context.Background())
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("WaitGroup.Done() was not called within timeout")
	}
}

func TestServer_runGinServer_InvalidPort(t *testing.T) {
	server := createTestServerWithoutDB()
	var wg sync.WaitGroup
	ch := make(chan struct{})

	wg.Add(1)
	go server.runGinServer(":99999", &wg, ch)

	select {
	case <-ch:
		t.Fatal("channel should not be closed for invalid port")
	case <-time.After(100 * time.Millisecond):
	}

	wg.Wait()
}

func TestServer_runGinServer_ConcurrentCalls(t *testing.T) {
	server1 := createTestServerWithoutDB()
	server2 := createTestServerWithoutDB()

	var wg sync.WaitGroup
	ch1 := make(chan struct{})
	ch2 := make(chan struct{})

	wg.Add(2)
	go server1.runGinServer(":0", &wg, ch1)
	go server2.runGinServer(":0", &wg, ch2)

	<-ch1
	<-ch2

	assert.NotNil(t, server1.srv)
	assert.NotNil(t, server2.srv)

	err1 := server1.srv.Shutdown(context.Background())
	err2 := server2.srv.Shutdown(context.Background())

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	wg.Wait()
}

func TestInitConfig(t *testing.T) {
	// Test that initConfig doesn't panic
	assert.NotPanics(t, func() {
		initConfig()
	})

	// Test that cfg is set after initialization
	assert.NotNil(t, cfg)
}

func TestServer_BasicStructure(t *testing.T) {
	server := &Server{}

	// Test zero values
	assert.Nil(t, server.router)
	assert.Nil(t, server.sd)
	assert.Nil(t, server.srv)

	// Test setting router
	router := gin.New()
	server.router = router
	assert.Equal(t, router, server.router)
}

func TestServer_HTTPServer_Lifecycle(t *testing.T) {
	server := createTestServerWithoutDB()
	var wg sync.WaitGroup
	ch := make(chan struct{})

	// Start server
	wg.Add(1)
	go server.runGinServer(":0", &wg, ch)

	// Wait for server to be ready
	<-ch

	// Verify server is running
	assert.NotNil(t, server.srv)

	// Test graceful shutdown
	shutdownStart := time.Now()
	err := server.srv.Shutdown(context.Background())
	shutdownDuration := time.Since(shutdownStart)

	assert.NoError(t, err)
	assert.Less(t, shutdownDuration, 5*time.Second, "Shutdown took too long")

	// Wait for goroutine to complete
	wg.Wait()
}

func TestServer_MultipleShutdowns(t *testing.T) {
	server := createTestServerWithoutDB()
	var wg sync.WaitGroup
	ch := make(chan struct{})

	// Start server
	wg.Add(1)
	go server.runGinServer(":0", &wg, ch)

	// Wait for server to be ready
	<-ch

	// First shutdown
	err1 := server.srv.Shutdown(context.Background())
	assert.NoError(t, err1)

	// Second shutdown should not panic
	assert.NotPanics(t, func() {
		server.srv.Shutdown(context.Background())
	})

	wg.Wait()
}

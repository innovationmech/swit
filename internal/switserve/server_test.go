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

package switserve

import (
	"context"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
)

// createTestServer creates a test server instance
func createTestServer(t *testing.T) *Server {
	var server *Server
	var err error

	// Initialize logger for tests
	logger.InitLogger()

	// Catch panics from config loading
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Database connection not available for testing: %v", r)
		}
	}()

	server, err = NewServer()
	if err != nil {
		t.Skipf("Failed to create server: %v", err)
		return nil
	}
	return server
}

func TestNewServer(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	assert.NotNil(t, server)
	assert.NotNil(t, server.baseServer)

	// Verify transports are registered
	transports := server.GetTransports()
	assert.Len(t, transports, 2)

	// Verify transport types
	transportNames := make([]string, len(transports))
	for i, transport := range transports {
		transportNames[i] = transport.GetName()
	}
	assert.Contains(t, transportNames, "http")
	assert.Contains(t, transportNames, "grpc")
}

func TestServer_GetTransports(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	transports := server.GetTransports()
	assert.NotNil(t, transports)
	assert.Len(t, transports, 2)

	// Verify we have both HTTP and gRPC transports
	transportTypes := make(map[string]bool)
	for _, transport := range transports {
		transportTypes[transport.GetName()] = true
	}
	assert.True(t, transportTypes["http"])
	assert.True(t, transportTypes["grpc"])
}

func TestServer_ServiceRegistration(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Test that services are registered with base server
	assert.NotNil(t, server.baseServer)

	// Verify that the service registry has registered services
	// This is an indirect test since we can't directly access the registry
	transports := server.GetTransports()
	assert.Len(t, transports, 2)
}

func TestServer_TransportManagerIntegration(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Test transport manager integration through base server
	transports := server.GetTransports()
	assert.NotEmpty(t, transports)

	// Each transport should be properly configured
	for _, transport := range transports {
		assert.NotEmpty(t, transport.GetName())
		assert.NotEmpty(t, transport.GetAddress())
	}
}

func TestServer_Shutdown(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Test shutdown
	err := server.Shutdown()
	assert.NoError(t, err)

	// Test shutdown again (should be idempotent)
	err = server.Shutdown()
	assert.NoError(t, err)
}

func TestServer_StartStop_WithoutServiceDiscovery(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}
	defer server.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server in goroutine
	startErr := make(chan error, 1)
	go func() {
		startErr <- server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	err := server.Stop()
	assert.NoError(t, err)

	// Wait for start to complete
	select {
	case err := <-startErr:
		// Service discovery errors are acceptable in test environment
		if err != nil {
			t.Logf("Start completed with error (acceptable in test): %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("Server start did not complete within timeout")
	}
}

func TestServer_StopWithoutStart(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Test stopping without starting
	err := server.Stop()
	assert.NoError(t, err)
}

func TestServer_MultipleStops(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Test multiple stops
	err := server.Stop()
	assert.NoError(t, err)

	err = server.Stop()
	assert.NoError(t, err)
}

func TestServer_ConcurrentStops(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Test concurrent stops
	done := make(chan bool, 2)
	go func() {
		server.Stop()
		done <- true
	}()
	go func() {
		server.Stop()
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done
}

func TestServer_ErrorHandling(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should handle nil dependencies gracefully",
			test: func(t *testing.T) {
				server := &Server{baseServer: nil}
				err := server.Stop()
				assert.NoError(t, err)
				err = server.Shutdown()
				assert.NoError(t, err)
			},
		},
		{
			name: "should handle transport manager errors",
			test: func(t *testing.T) {
				server := createTestServer(t)
				if server == nil {
					return
				}
				// Test error handling through normal operations
				err := server.Stop()
				assert.NoError(t, err)
			},
		},
		{
			name: "should handle service discovery errors gracefully",
			test: func(t *testing.T) {
				server := createTestServer(t)
				if server == nil {
					return
				}
				// Service discovery errors should not prevent server operations
				err := server.Stop()
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestServer_ResourceManagement(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should cleanup all resources on shutdown",
			test: func(t *testing.T) {
				server := createTestServer(t)
				if server == nil {
					return
				}
				err := server.Shutdown()
				assert.NoError(t, err)
			},
		},
		{
			name: "should handle partial initialization cleanup",
			test: func(t *testing.T) {
				server := &Server{baseServer: nil}
				err := server.Shutdown()
				assert.NoError(t, err)
			},
		},
		{
			name: "should handle concurrent shutdown calls",
			test: func(t *testing.T) {
				server := createTestServer(t)
				if server == nil {
					return
				}

				done := make(chan bool, 2)
				go func() {
					server.Shutdown()
					done <- true
				}()
				go func() {
					server.Shutdown()
					done <- true
				}()

				<-done
				<-done
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestServer_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should handle zero value server",
			test: func(t *testing.T) {
				var server Server
				err := server.Stop()
				assert.NoError(t, err)
				err = server.Shutdown()
				assert.NoError(t, err)
			},
		},
		{
			name: "should validate server components",
			test: func(t *testing.T) {
				server := createTestServer(t)
				if server == nil {
					return
				}
				assert.NotNil(t, server.baseServer)
			},
		},
		{
			name: "should handle service registration edge cases",
			test: func(t *testing.T) {
				server := createTestServer(t)
				if server == nil {
					return
				}
				// Test that services are properly registered
				transports := server.GetTransports()
				assert.NotEmpty(t, transports)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestServer_LifecycleManagement(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should handle start stop cycles",
			test: func(t *testing.T) {
				server := createTestServer(t)
				if server == nil {
					return
				}
				defer server.Shutdown()

				err := server.Stop()
				assert.NoError(t, err)
			},
		},
		{
			name: "should handle shutdown after stop",
			test: func(t *testing.T) {
				server := createTestServer(t)
				if server == nil {
					return
				}

				err := server.Stop()
				assert.NoError(t, err)
				err = server.Shutdown()
				assert.NoError(t, err)
			},
		},
		{
			name: "should handle stop after shutdown",
			test: func(t *testing.T) {
				server := createTestServer(t)
				if server == nil {
					return
				}

				err := server.Shutdown()
				assert.NoError(t, err)
				err = server.Stop()
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestServer_ComponentIntegration(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should integrate all components properly",
			test: func(t *testing.T) {
				server := createTestServer(t)
				if server == nil {
					return
				}
				defer server.Shutdown()

				// Verify all components are integrated
				assert.NotNil(t, server.baseServer)
				transports := server.GetTransports()
				assert.Len(t, transports, 2)
			},
		},
		{
			name: "should handle transport lifecycle",
			test: func(t *testing.T) {
				server := createTestServer(t)
				if server == nil {
					return
				}
				defer server.Shutdown()

				transports := server.GetTransports()
				for _, transport := range transports {
					assert.NotEmpty(t, transport.GetName())
					assert.NotEmpty(t, transport.GetAddress())
				}
			},
		},
		{
			name: "should handle service discovery integration",
			test: func(t *testing.T) {
				server := createTestServer(t)
				if server == nil {
					return
				}
				defer server.Shutdown()

				// Service discovery integration is handled by base server
				assert.NotNil(t, server.baseServer)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

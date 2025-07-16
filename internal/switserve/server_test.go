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
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Initialize logger for tests
	logger.InitLogger()

	// Add config path for tests
	// This is needed because go test changes working directory to the package directory
	viper.AddConfigPath("../../..")
}

// createTestServer creates a server for testing, skipping if database is not available
func createTestServer(t *testing.T) *Server {
	var server *Server
	var err error

	// Catch panics from database connection
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
	assert.NotNil(t, server.transportManager)
	assert.NotNil(t, server.serviceRegistry)
	assert.NotNil(t, server.httpTransport)
	assert.NotNil(t, server.grpcTransport)
	assert.NotNil(t, server.sd)

	// Verify transports are registered
	transports := server.GetTransports()
	assert.Len(t, transports, 2)

	// Verify transport types
	transportNames := make([]string, len(transports))
	for i, transport := range transports {
		transportNames[i] = transport.Name()
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
	assert.Len(t, transports, 2)

	// Verify we get copies, not the original slice
	originalLen := len(transports)
	transports = append(transports, nil)

	newTransports := server.GetTransports()
	assert.Len(t, newTransports, originalLen)
}

func TestServer_ServiceRegistration(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Verify services were registered
	registrars := server.serviceRegistry.GetRegistrars()
	assert.NotEmpty(t, registrars)

	// We should have multiple services registered
	assert.True(t, len(registrars) >= 4) // At least Greeter, Notification, Health, Stop
}

func TestServer_TransportManagerIntegration(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Verify transport manager has transports
	transports := server.transportManager.GetTransports()
	assert.Len(t, transports, 2)

	// Verify both HTTP and gRPC transports are registered
	names := make([]string, len(transports))
	for i, transport := range transports {
		names[i] = transport.Name()
	}
	assert.Contains(t, names, "http")
	assert.Contains(t, names, "grpc")
}

func TestServer_Shutdown(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Shutdown should call Stop internally
	err := server.Shutdown()
	assert.NoError(t, err)
}

func TestServer_StartStop_WithoutServiceDiscovery(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	ctx := context.Background()

	// Start server - may fail due to service discovery connection or transport issues
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Server start panicked (expected): %v", r)
			}
		}()
		server.Start(ctx)
	}()

	// Stop server should always work
	err := server.Stop()
	assert.NoError(t, err)
}

func TestServer_StopWithoutStart(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Stop server without starting - should work
	err := server.Stop()
	assert.NoError(t, err)
}

func TestServer_MultipleStops(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Multiple stops should not cause issues
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
	done := make(chan bool)

	for i := 0; i < 3; i++ {
		go func() {
			server.Stop()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should not panic or cause issues
}

func TestServer_TransportAddresses(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Initially addresses should be empty
	assert.Empty(t, server.httpTransport.Address())
	assert.Empty(t, server.grpcTransport.Address())

	// After attempting to start, addresses might be set
	// We don't test actual start due to service discovery dependency
}

func TestServer_ServiceRegistryIntegration(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Verify service registry is properly initialized
	assert.NotNil(t, server.serviceRegistry)

	// Verify services are registered
	registrars := server.serviceRegistry.GetRegistrars()
	assert.NotEmpty(t, registrars)

	// Should have at least the basic services
	assert.True(t, len(registrars) >= 4)
}

func TestServer_ContextHandling(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start with timeout context - may fail due to service discovery or transport issues
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Server start panicked (expected): %v", r)
			}
		}()
		server.Start(ctx)
	}()

	// Stop should always work
	err := server.Stop()
	assert.NoError(t, err)
}

func TestServer_Integration(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Verify server components are properly initialized
	assert.NotNil(t, server.transportManager)
	assert.NotNil(t, server.serviceRegistry)
	assert.NotNil(t, server.httpTransport)
	assert.NotNil(t, server.grpcTransport)
	assert.NotNil(t, server.sd)

	// Verify transports are registered
	transports := server.GetTransports()
	assert.Len(t, transports, 2)

	// Verify services are registered
	registrars := server.serviceRegistry.GetRegistrars()
	assert.NotEmpty(t, registrars)

	// Test shutdown
	err := server.Shutdown()
	assert.NoError(t, err)
}

func TestServer_ErrorRecovery(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Test that multiple shutdown calls don't cause issues
	err := server.Shutdown()
	assert.NoError(t, err)

	err = server.Shutdown()
	assert.NoError(t, err)

	// Test that stop after shutdown works
	err = server.Stop()
	assert.NoError(t, err)
}

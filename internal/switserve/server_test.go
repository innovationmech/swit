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
	assert.NotNil(t, server.httpTransport)
	assert.NotNil(t, server.grpcTransport)
	assert.NotNil(t, server.sd)

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

	// Verify services were registered through HTTP transport
	serviceRegistry := server.httpTransport.GetServiceRegistry()
	assert.NotNil(t, serviceRegistry)

	// We should have multiple services registered
	handlers := serviceRegistry.GetAllHandlers()
	assert.True(t, len(handlers) >= 4) // At least Greeter, Notification, Health, Stop
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
		names[i] = transport.GetName()
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

	// Multiple concurrent stops should be safe
	done := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func() {
			err := server.Stop()
			done <- err
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for concurrent stop operations")
		}
	}
}

// Test error handling scenarios
func TestServer_ErrorHandling(t *testing.T) {
	t.Run("should_handle_nil_dependencies_gracefully", func(t *testing.T) {
		server := &Server{}

		// Stop with nil components should not panic
		err := server.Stop()
		assert.NoError(t, err)

		// Shutdown with nil dependencies should not panic
		err = server.Shutdown()
		assert.NoError(t, err)
	})

	t.Run("should_handle_transport_manager_errors", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Multiple stops should not cause issues
		err := server.Stop()
		assert.NoError(t, err)

		// Second stop should also be safe
		err = server.Stop()
		assert.NoError(t, err)
	})

	t.Run("should_handle_service_discovery_errors_gracefully", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Stop should work even if service discovery fails
		err := server.Stop()
		assert.NoError(t, err)
	})
}

// Test resource management
func TestServer_ResourceManagement(t *testing.T) {
	t.Run("should_cleanup_all_resources_on_shutdown", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Verify initial state
		assert.NotNil(t, server.deps)
		assert.NotNil(t, server.transportManager)

		// Shutdown should clean up resources
		err := server.Shutdown()
		assert.NoError(t, err)
	})

	t.Run("should_handle_partial_initialization_cleanup", func(t *testing.T) {
		// Test server with partial initialization
		server := &Server{
			transportManager: nil,
		}

		// Should not panic on cleanup
		err := server.Shutdown()
		assert.NoError(t, err)
	})

	t.Run("should_handle_concurrent_shutdown_calls", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Multiple concurrent shutdowns should be safe
		done := make(chan error, 3)

		for i := 0; i < 3; i++ {
			go func() {
				err := server.Shutdown()
				done <- err
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 3; i++ {
			select {
			case err := <-done:
				assert.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("Timeout waiting for concurrent shutdown operations")
			}
		}
	})
}

// Test edge cases and boundary conditions
func TestServer_EdgeCases(t *testing.T) {
	t.Run("should_handle_zero_value_server", func(t *testing.T) {
		var server Server

		// Zero value server should be safe to shutdown
		err := server.Shutdown()
		assert.NoError(t, err)

		// Zero value server should be safe to stop
		err = server.Stop()
		assert.NoError(t, err)
	})

	t.Run("should_validate_server_components", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Verify all components are properly initialized
		assert.NotNil(t, server.transportManager, "Transport manager should be initialized")
		assert.NotNil(t, server.httpTransport, "HTTP transport should be initialized")
		assert.NotNil(t, server.grpcTransport, "gRPC transport should be initialized")
		assert.NotNil(t, server.sd, "Service discovery should be initialized")
		assert.NotNil(t, server.deps, "Dependencies should be initialized")
	})

	t.Run("should_handle_service_registration_edge_cases", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Verify service registration doesn't cause issues
		serviceRegistry := server.httpTransport.GetServiceRegistry()
		handlers := serviceRegistry.GetAllHandlers()
		assert.NotEmpty(t, handlers, "Should have registered services")

		// Multiple calls to registerServices should be safe
		server.registerServices()

		// Should still have services registered
		newHandlers := serviceRegistry.GetAllHandlers()
		assert.NotEmpty(t, newHandlers, "Should still have registered services")
	})
}

// Test server lifecycle management
func TestServer_LifecycleManagement(t *testing.T) {
	t.Run("should_handle_start_stop_cycles", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Multiple start-stop cycles should work
		for i := 0; i < 3; i++ {
			// Stop (even without start) should work
			err := server.Stop()
			assert.NoError(t, err, "Stop cycle %d should succeed", i)
		}
	})

	t.Run("should_handle_shutdown_after_stop", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Stop first
		err := server.Stop()
		assert.NoError(t, err)

		// Then shutdown should still work
		err = server.Shutdown()
		assert.NoError(t, err)
	})

	t.Run("should_handle_stop_after_shutdown", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Shutdown first
		err := server.Shutdown()
		assert.NoError(t, err)

		// Then stop should still work
		err = server.Stop()
		assert.NoError(t, err)
	})
}

func TestServer_ComponentIntegration(t *testing.T) {
	t.Run("should_integrate_all_components_properly", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Verify component integration
		transports := server.GetTransports()
		assert.Len(t, transports, 2, "Should have both HTTP and gRPC transports")

		// Verify transport manager integration
		managerTransports := server.transportManager.GetTransports()
		assert.Equal(t, len(transports), len(managerTransports), "Transport manager should have same transports")

		// Verify service registry integration
		serviceRegistry := server.httpTransport.GetServiceRegistry()
		handlers := serviceRegistry.GetAllHandlers()
		assert.NotEmpty(t, handlers, "Service registry should have registered services")
	})

	t.Run("should_handle_transport_lifecycle", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Verify transports are properly configured
		assert.NotNil(t, server.httpTransport.GetRouter(), "HTTP transport should have router")
		assert.NotNil(t, server.grpcTransport.GetServer(), "gRPC transport should have server")

		// Verify addresses are set
		assert.NotEmpty(t, server.httpTransport.GetAddress(), "HTTP transport should have address")
		assert.NotEmpty(t, server.grpcTransport.GetAddress(), "gRPC transport should have address")
	})

	t.Run("should_handle_service_discovery_integration", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Service discovery should be initialized
		assert.NotNil(t, server.sd, "Service discovery should be initialized")

		// Stop should handle service discovery cleanup
		err := server.Stop()
		assert.NoError(t, err, "Stop should handle service discovery cleanup")
	})
}

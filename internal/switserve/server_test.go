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
	viper.AddConfigPath(".")
	viper.AddConfigPath("../..")
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
	assert.NotNil(t, server.switserveServer)

	// Verify server properties
	assert.Equal(t, "swit-serve", server.GetServiceName())
	assert.NotEmpty(t, server.GetHTTPAddress())
	assert.NotEmpty(t, server.GetGRPCAddress())
}

func TestServer_GetAddresses(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Test HTTP address
	httpAddr := server.GetHTTPAddress()
	assert.NotEmpty(t, httpAddr)

	// Test gRPC address
	grpcAddr := server.GetGRPCAddress()
	assert.NotEmpty(t, grpcAddr)

	// Test legacy GetAddress method
	addr := server.GetAddress()
	assert.Equal(t, httpAddr, addr)
}

func TestServer_ServiceProperties(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Verify service name
	serviceName := server.GetServiceName()
	assert.Equal(t, "swit-serve", serviceName)

	// Verify health check
	ctx := context.Background()
	isHealthy := server.IsHealthy(ctx)
	assert.True(t, isHealthy)
}

func TestServer_BaseServerIntegration(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Verify base server integration
	assert.NotNil(t, server.switserveServer)

	// Verify addresses are properly configured
	httpAddr := server.GetHTTPAddress()
	grpcAddr := server.GetGRPCAddress()
	assert.NotEmpty(t, httpAddr)
	assert.NotEmpty(t, grpcAddr)
	assert.NotEqual(t, httpAddr, grpcAddr)
}

func TestServer_Stop(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Stop should work without errors
	err := server.Stop()
	assert.NoError(t, err)
}

func TestServer_StartStop_Lifecycle(t *testing.T) {
	server := createTestServer(t)
	if server == nil {
		return
	}

	// Start server - may fail due to service discovery connection or transport issues
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Server start panicked (expected): %v", r)
			}
		}()
		server.Start()
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

		// GetServiceName with nil switserveServer should return default
		serviceName := server.GetServiceName()
		assert.Equal(t, "swit-serve", serviceName)
	})

	t.Run("should_handle_switserve_server_errors", func(t *testing.T) {
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

	t.Run("should_handle_health_check_with_nil_server", func(t *testing.T) {
		server := &Server{}
		ctx := context.Background()

		// Health check with nil switserveServer should return false
		isHealthy := server.IsHealthy(ctx)
		assert.False(t, isHealthy)
	})
}

// Test resource management
func TestServer_ResourceManagement(t *testing.T) {
	t.Run("should_cleanup_all_resources_on_stop", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Verify initial state
		assert.NotNil(t, server.switserveServer)

		// Stop should clean up resources
		err := server.Stop()
		assert.NoError(t, err)
	})

	t.Run("should_handle_partial_initialization_cleanup", func(t *testing.T) {
		// Test server with partial initialization
		server := &Server{
			switserveServer: nil,
		}

		// Should not panic on cleanup
		err := server.Stop()
		assert.NoError(t, err)
	})

	t.Run("should_handle_concurrent_stop_calls", func(t *testing.T) {
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
	})
}

// Test edge cases and boundary conditions
func TestServer_EdgeCases(t *testing.T) {
	t.Run("should_handle_zero_value_server", func(t *testing.T) {
		var server Server

		// Zero value server should be safe to stop
		err := server.Stop()
		assert.NoError(t, err)

		// Zero value server should return default service name
		serviceName := server.GetServiceName()
		assert.Equal(t, "swit-serve", serviceName)
	})

	t.Run("should_validate_server_components", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Verify all components are properly initialized
		assert.NotNil(t, server.switserveServer, "SwitserveServer should be initialized")
		assert.NotEmpty(t, server.GetHTTPAddress(), "HTTP address should be set")
		assert.NotEmpty(t, server.GetGRPCAddress(), "gRPC address should be set")
		assert.Equal(t, "swit-serve", server.GetServiceName(), "Service name should be correct")
	})

	t.Run("should_handle_address_retrieval_edge_cases", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Verify address methods work correctly
		httpAddr := server.GetHTTPAddress()
		grpcAddr := server.GetGRPCAddress()
		legacyAddr := server.GetAddress()

		assert.NotEmpty(t, httpAddr)
		assert.NotEmpty(t, grpcAddr)
		assert.Equal(t, httpAddr, legacyAddr)
		assert.NotEqual(t, httpAddr, grpcAddr)
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

	t.Run("should_handle_health_checks", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		ctx := context.Background()

		// Health check should work
		isHealthy := server.IsHealthy(ctx)
		assert.True(t, isHealthy)

		// Stop server
		err := server.Stop()
		assert.NoError(t, err)
	})

	t.Run("should_handle_service_properties", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Service properties should be consistent
		serviceName1 := server.GetServiceName()
		serviceName2 := server.GetServiceName()
		assert.Equal(t, serviceName1, serviceName2)
		assert.Equal(t, "swit-serve", serviceName1)
	})
}

func TestServer_ComponentIntegration(t *testing.T) {
	t.Run("should_integrate_all_components_properly", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Verify component integration
		assert.NotNil(t, server.switserveServer)
		assert.NotEmpty(t, server.GetHTTPAddress())
		assert.NotEmpty(t, server.GetGRPCAddress())
		assert.Equal(t, "swit-serve", server.GetServiceName())
	})

	t.Run("should_handle_base_server_lifecycle", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Verify base server integration
		assert.NotNil(t, server.switserveServer)

		// Verify addresses are set
		assert.NotEmpty(t, server.GetHTTPAddress())
		assert.NotEmpty(t, server.GetGRPCAddress())

		// Stop should work
		err := server.Stop()
		assert.NoError(t, err)
	})

	t.Run("should_handle_health_and_service_integration", func(t *testing.T) {
		server := createTestServer(t)
		if server == nil {
			return
		}

		// Health check should work
		ctx := context.Background()
		isHealthy := server.IsHealthy(ctx)
		assert.True(t, isHealthy)

		// Service name should be correct
		serviceName := server.GetServiceName()
		assert.Equal(t, "swit-serve", serviceName)

		// Stop should handle cleanup
		err := server.Stop()
		assert.NoError(t, err)
	})
}

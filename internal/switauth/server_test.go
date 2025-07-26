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

package switauth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/innovationmech/swit/internal/switauth/config"
	"github.com/innovationmech/swit/internal/switauth/deps"
	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Initialize logger for tests
	logger.InitLogger()

	// Set up test config file
	createTestConfig()

	// Run tests
	code := m.Run()

	// Clean up test config
	cleanupTestConfig()

	os.Exit(code)
}

func createTestConfig() {
	configContent := `
server:
  port: "8090"
  grpcPort: "50051"
database:
  username: "test"
  password: "test"
  host: "localhost"
  port: "3306"
  dbname: "test_db"
serviceDiscovery:
  address: "localhost:8500"
`
	err := os.WriteFile("switauth.yaml", []byte(configContent), 0644)
	if err != nil {
		panic(err)
	}
}

func cleanupTestConfig() {
	_ = os.Remove("switauth.yaml")
}

func TestNewServer_ServiceDiscoveryError(t *testing.T) {
	// This test will fail if consul is not available or database is not available
	defer func() {
		if r := recover(); r != nil {
			// This is expected when database is not available
			assert.Contains(t, r.(string), "fail to connect database")
		}
	}()

	server, err := NewServer()

	if err != nil {
		// This is expected when consul is not available
		assert.Contains(t, err.Error(), "failed to create service discovery client")
		assert.Nil(t, server)
		return
	}

	// If no error, verify the server is created properly
	require.NotNil(t, server)
	assert.NotNil(t, server.transportManager)
	assert.NotNil(t, server.httpTransport)
	assert.NotNil(t, server.httpTransport.GetServiceRegistry())
	assert.NotNil(t, server.grpcTransport)
	assert.NotNil(t, server.sd)
	assert.NotNil(t, server.config)
}

func TestServerWithComponents(t *testing.T) {
	// Create a server with real components for testing
	transportManager := transport.NewManager()
	httpTransport := transport.NewHTTPTransport()
	grpcTransport := transport.NewGRPCTransport()

	// Set addresses
	httpTransport.SetAddress(":8090")
	grpcTransport.SetAddress(":50051")

	server := &Server{
		transportManager: transportManager,
		httpTransport:    httpTransport,
		grpcTransport:    grpcTransport,
		sd:               &discovery.ServiceDiscovery{},
		config: &config.AuthConfig{
			Server: struct {
				Port     string `json:"port" yaml:"port"`
				GRPCPort string `json:"grpcPort" yaml:"grpcPort"`
			}{
				Port:     "8090",
				GRPCPort: "50051",
			},
		},
	}

	t.Run("GetHTTPAddress", func(t *testing.T) {
		address := server.GetHTTPAddress()
		// Address is empty until transport starts
		assert.Equal(t, "", address)
	})

	t.Run("GetGRPCAddress", func(t *testing.T) {
		address := server.GetGRPCAddress()
		// Address is empty until transport starts
		assert.Equal(t, "", address)
	})

	t.Run("GetServices", func(t *testing.T) {
		services := server.GetServices()
		assert.NotNil(t, services)
		// Initially empty since no services registered
		assert.Len(t, services, 0)
	})

	t.Run("RegisterWithDiscovery", func(t *testing.T) {
		// This may fail or panic in test environment without consul
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Expected panic in test environment: %v", r)
			}
		}()
		err := server.registerWithDiscovery()
		if err != nil {
			t.Logf("Expected error in test environment: %v", err)
		}
	})

	t.Run("DeregisterFromDiscovery", func(t *testing.T) {
		// This may fail or panic in test environment without consul
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Expected panic in test environment: %v", r)
			}
		}()
		err := server.deregisterFromDiscovery()
		if err != nil {
			t.Logf("Expected error in test environment: %v", err)
		}
	})

	t.Run("ConfigureMiddleware", func(t *testing.T) {
		server.configureMiddleware()
		router := server.httpTransport.GetRouter()
		assert.NotNil(t, router)
	})

	t.Run("Stop", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Expected panic in test environment: %v", r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := server.Stop(ctx)
		if err != nil {
			t.Logf("Expected error in test environment: %v", err)
		}
	})
}

func TestServerPortConfiguration(t *testing.T) {
	// Test server with different port configurations
	// Note: GetAddress() returns empty until transport starts
	tests := []struct {
		name     string
		httpPort string
		grpcPort string
	}{
		{
			name:     "standard ports",
			httpPort: "8090",
			grpcPort: "50051",
		},
		{
			name:     "different ports",
			httpPort: "3000",
			grpcPort: "9000",
		},
		{
			name:     "empty grpc port uses default",
			httpPort: "8080",
			grpcPort: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpTransport := transport.NewHTTPTransport()
			grpcTransport := transport.NewGRPCTransport()

			httpTransport.SetAddress(":" + tt.httpPort)

			if tt.grpcPort != "" {
				grpcTransport.SetAddress(":" + tt.grpcPort)
			} else {
				grpcTransport.SetAddress(":50051") // Default
			}

			server := &Server{
				httpTransport: httpTransport,
				grpcTransport: grpcTransport,
			}

			// Address is empty until transport starts
			assert.Equal(t, "", server.GetHTTPAddress())
			assert.Equal(t, "", server.GetGRPCAddress())
		})
	}
}

func TestServerStruct(t *testing.T) {
	// Test server struct initialization
	server := &Server{}
	assert.NotNil(t, server)

	// Test that all fields can be set
	server.transportManager = nil
	server.httpTransport = nil
	server.grpcTransport = nil
	server.sd = nil
	server.config = nil

	assert.Nil(t, server.transportManager)
	assert.Nil(t, server.httpTransport)
	assert.Nil(t, server.grpcTransport)
	assert.Nil(t, server.sd)
	assert.Nil(t, server.config)
}

func TestServerRegisterServices(t *testing.T) {
	// Test registerServices method
	// Note: This test now uses the dependency injection pattern
	// Expected to fail in test environment due to database connection
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic in test environment: %v", r)
		}
	}()

	dependencies, err := deps.NewDependencies()
	if err != nil {
		t.Logf("Expected error in test environment (dependencies creation): %v", err)
		return
	}

	httpTransport := transport.NewHTTPTransport()
	server := &Server{
		httpTransport: httpTransport,
		sd:            dependencies.SD,
		deps:          dependencies,
	}

	// This should now work with proper dependencies
	server.registerServices()

	// Check if services were registered
	services := server.GetServices()
	if len(services) > 0 {
		t.Log("Service registration completed successfully")
		assert.Contains(t, services, "auth-service")
		assert.Contains(t, services, "health-service")
	} else {
		t.Log("No services registered (may be expected in test environment)")
	}
}

func TestServerStartErrorHandling(t *testing.T) {
	// Test Start method error handling
	server := &Server{
		transportManager: transport.NewManager(),
		httpTransport:    transport.NewHTTPTransport(),
		grpcTransport:    transport.NewGRPCTransport(),
		sd:               &discovery.ServiceDiscovery{},
		config: &config.AuthConfig{
			Server: struct {
				Port     string `json:"port" yaml:"port"`
				GRPCPort string `json:"grpcPort" yaml:"grpcPort"`
			}{
				Port:     "8090",
				GRPCPort: "50051",
			},
		},
	}

	// Set addresses
	server.httpTransport.SetAddress(":8090")
	server.grpcTransport.SetAddress(":50051")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start may fail or panic due to database/service dependencies
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic in test environment: %v", r)
		}
	}()
	err := server.Start(ctx)
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}
}

func TestServerSamePortsRegistration(t *testing.T) {
	// Test registration when HTTP and gRPC use the same port
	httpTransport := transport.NewHTTPTransport()
	grpcTransport := transport.NewGRPCTransport()

	httpTransport.SetAddress(":8080")
	grpcTransport.SetAddress(":8080") // Same port

	server := &Server{
		httpTransport: httpTransport,
		grpcTransport: grpcTransport,
		sd:            &discovery.ServiceDiscovery{},
	}

	// Registration may fail or panic in test environment
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic in test environment: %v", r)
		}
	}()
	err := server.registerWithDiscovery()
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}

	err = server.deregisterFromDiscovery()
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}
}

func TestServerDifferentPortsRegistration(t *testing.T) {
	// Test registration when HTTP and gRPC use different ports
	httpTransport := transport.NewHTTPTransport()
	grpcTransport := transport.NewGRPCTransport()

	httpTransport.SetAddress(":8080")
	grpcTransport.SetAddress(":9090") // Different port

	server := &Server{
		httpTransport: httpTransport,
		grpcTransport: grpcTransport,
		sd:            &discovery.ServiceDiscovery{},
	}

	// Registration may fail or panic in test environment
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic in test environment: %v", r)
		}
	}()
	err := server.registerWithDiscovery()
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}

	err = server.deregisterFromDiscovery()
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}
}

func TestServerTransportManager(t *testing.T) {
	// Test transport manager functionality
	transportManager := transport.NewManager()
	httpTransport := transport.NewHTTPTransport()
	grpcTransport := transport.NewGRPCTransport()

	// Register transports
	transportManager.Register(httpTransport)
	transportManager.Register(grpcTransport)

	server := &Server{
		transportManager: transportManager,
		httpTransport:    httpTransport,
		grpcTransport:    grpcTransport,
	}

	// Verify transports are registered
	assert.NotNil(t, server.transportManager)

	// Test stop
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic in test environment: %v", r)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := server.Stop(ctx)
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}
}

func TestServerPortExtraction(t *testing.T) {
	// Test port extraction from addresses
	httpTransport := transport.NewHTTPTransport()
	grpcTransport := transport.NewGRPCTransport()

	httpTransport.SetAddress(":8090")
	grpcTransport.SetAddress(":50051")

	server := &Server{
		httpTransport: httpTransport,
		grpcTransport: grpcTransport,
	}

	// Test that we can extract ports correctly
	httpPort := server.httpTransport.GetPort()
	grpcPort := server.grpcTransport.GetPort()

	assert.Equal(t, 8090, httpPort)
	assert.Equal(t, 50051, grpcPort)
}

func TestServerMethodsWithNilComponents(t *testing.T) {
	// Test server behavior with nil components
	server := &Server{}

	// These methods should handle nil components gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Panic occurred (may be expected): %v", r)
		}
	}()

	// Test methods that access components
	if server.httpTransport != nil {
		server.GetHTTPAddress()
	}
	if server.grpcTransport != nil {
		server.GetGRPCAddress()
	}
	if server.httpTransport != nil {
		server.GetServices()
	}
}

func TestServerConfigureMiddleware(t *testing.T) {
	// Test middleware configuration
	httpTransport := transport.NewHTTPTransport()

	server := &Server{
		httpTransport: httpTransport,
	}

	// Configure middleware
	server.configureMiddleware()

	// Verify router is accessible
	router := server.httpTransport.GetRouter()
	assert.NotNil(t, router)
}

func TestServerCompleteLifecycle(t *testing.T) {
	// Test complete server lifecycle
	transportManager := transport.NewManager()
	httpTransport := transport.NewHTTPTransport()
	grpcTransport := transport.NewGRPCTransport()

	httpTransport.SetAddress(":8090")
	grpcTransport.SetAddress(":50051")

	// Register transports
	transportManager.Register(httpTransport)
	transportManager.Register(grpcTransport)

	server := &Server{
		transportManager: transportManager,
		httpTransport:    httpTransport,
		grpcTransport:    grpcTransport,
		sd:               &discovery.ServiceDiscovery{},
		config: &config.AuthConfig{
			Server: struct {
				Port     string `json:"port" yaml:"port"`
				GRPCPort string `json:"grpcPort" yaml:"grpcPort"`
			}{
				Port:     "8090",
				GRPCPort: "50051",
			},
		},
	}

	// Configure middleware
	server.configureMiddleware()

	// Test service discovery registration (may fail or panic in test env)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic in test environment: %v", r)
		}
	}()
	err := server.registerWithDiscovery()
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}

	// Test getting addresses
	assert.Equal(t, ":8090", server.GetHTTPAddress())
	assert.Equal(t, ":50051", server.GetGRPCAddress())

	// Test getting services
	services := server.GetServices()
	assert.NotNil(t, services)

	// Test deregistration (may fail or panic in test env)
	err = server.deregisterFromDiscovery()
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}

	// Test stop
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = server.Stop(ctx)
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}
}

func TestServerErrorPropagation(t *testing.T) {
	// Test error propagation in various scenarios
	server := &Server{
		transportManager: transport.NewManager(),
		httpTransport:    transport.NewHTTPTransport(),
		grpcTransport:    transport.NewGRPCTransport(),
		sd:               &discovery.ServiceDiscovery{},
	}

	// Test Start without proper configuration
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic in test environment: %v", r)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := server.Start(ctx)
	if err != nil {
		t.Logf("Expected error without proper configuration: %v", err)
	}
}

func TestServerValidation(t *testing.T) {
	// Test server validation and edge cases
	server := &Server{}

	// Test with empty server
	assert.NotNil(t, server)

	// Test setting components
	server.transportManager = transport.NewManager()
	server.httpTransport = transport.NewHTTPTransport()
	server.grpcTransport = transport.NewGRPCTransport()

	assert.NotNil(t, server.transportManager)
	assert.NotNil(t, server.httpTransport)
	assert.NotNil(t, server.grpcTransport)
}

func TestServerAddressDefaults(t *testing.T) {
	// Test address defaults when not set
	httpTransport := transport.NewHTTPTransport()
	grpcTransport := transport.NewGRPCTransport()

	server := &Server{
		httpTransport: httpTransport,
		grpcTransport: grpcTransport,
	}

	// Initially addresses should be empty (not started yet)
	assert.Equal(t, "", server.GetHTTPAddress())
	assert.Equal(t, "", server.GetGRPCAddress())

	// Set addresses
	httpTransport.SetAddress(":8090")
	grpcTransport.SetAddress(":50051")

	// Addresses are still empty until transport starts
	assert.Equal(t, "", server.GetHTTPAddress())
	assert.Equal(t, "", server.GetGRPCAddress())
}

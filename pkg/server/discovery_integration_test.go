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
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/pkg/discovery"
	"github.com/innovationmech/swit/pkg/logger"
)

// DiscoveryIntegrationTestSuite provides integration tests for service discovery
type DiscoveryIntegrationTestSuite struct {
	suite.Suite
	consulAddress string
	server        *BusinessServerImpl
	config        *ServerConfig
	httpPort      string
	grpcPort      string
	testService   *TestServiceRegistrar
	testDeps      *TestDependencyContainer
}

// SetupSuite initializes the test suite
func (suite *DiscoveryIntegrationTestSuite) SetupSuite() {
	// Initialize logger for tests
	logger.InitLogger()

	// Check if Consul is available for testing
	suite.consulAddress = os.Getenv("CONSUL_ADDRESS")
	if suite.consulAddress == "" {
		suite.consulAddress = "localhost:8500"
	}

	// Test Consul connectivity
	if !suite.isConsulAvailable() {
		suite.T().Skip("Consul not available for integration testing. Set CONSUL_ADDRESS environment variable or start Consul on localhost:8500")
	}
}

// SetupTest sets up each test case
func (suite *DiscoveryIntegrationTestSuite) SetupTest() {
	// Find available ports
	suite.httpPort = suite.findAvailablePort()
	suite.grpcPort = suite.findAvailablePort()

	// Create test configuration with discovery enabled
	suite.config = &ServerConfig{
		ServiceName: "discovery-test-service",
		HTTP: HTTPConfig{
			Port:         suite.httpPort,
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: GRPCConfig{
			Port:                suite.grpcPort,
			EnableReflection:    true,
			EnableHealthService: true,
			Enabled:             true,
			MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
			MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
			KeepaliveParams: GRPCKeepaliveParams{
				MaxConnectionIdle:     15 * time.Minute,
				MaxConnectionAge:      30 * time.Minute,
				MaxConnectionAgeGrace: 5 * time.Minute,
				Time:                  5 * time.Minute,
				Timeout:               1 * time.Minute,
			},
			KeepalivePolicy: GRPCKeepalivePolicy{
				MinTime:             5 * time.Minute,
				PermitWithoutStream: false,
			},
		},
		ShutdownTimeout: 5 * time.Second,
		Discovery: DiscoveryConfig{
			Enabled:     true,
			Address:     suite.consulAddress,
			ServiceName: "discovery-test-service",
			Tags:        []string{"test", "integration"},
		},
		Middleware: MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
	}

	// Create test dependencies
	suite.testDeps = NewTestDependencyContainer()

	// Create test service registrar
	suite.testService = NewTestServiceRegistrar()

	// Add test HTTP handler
	httpHandler := NewTestHTTPHandler("discovery-test-http")
	httpHandler.AddRoute("/discovery-test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "discovery test response"})
	})
	suite.testService.AddHTTPHandler(httpHandler)

	// Add test gRPC service
	grpcService := NewTestGRPCService("discovery-test-grpc")
	suite.testService.AddGRPCService(grpcService)
}

// TearDownTest cleans up after each test case
func (suite *DiscoveryIntegrationTestSuite) TearDownTest() {
	if suite.server != nil {
		// Stop the server
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := suite.server.Stop(ctx); err != nil {
			suite.T().Logf("Error stopping server: %v", err)
		}

		// Shutdown the server
		if err := suite.server.Shutdown(); err != nil {
			suite.T().Logf("Error shutting down server: %v", err)
		}

		// Clean up any remaining service registrations
		suite.cleanupConsulServices()
	}
}

// isConsulAvailable checks if Consul is available for testing
func (suite *DiscoveryIntegrationTestSuite) isConsulAvailable() bool {
	conn, err := net.DialTimeout("tcp", suite.consulAddress, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// findAvailablePort finds an available port for testing
func (suite *DiscoveryIntegrationTestSuite) findAvailablePort() string {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(suite.T(), err)
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("%d", addr.Port)
}

// cleanupConsulServices removes any test services from Consul
func (suite *DiscoveryIntegrationTestSuite) cleanupConsulServices() {
	client, err := discovery.GetServiceDiscoveryByAddress(suite.consulAddress)
	if err != nil {
		suite.T().Logf("Failed to create discovery client for cleanup: %v", err)
		return
	}

	// Try to deregister test services
	serviceName := suite.config.Discovery.ServiceName
	httpPort := suite.httpPort
	grpcPort := suite.grpcPort

	if httpPort != "" {
		if err := client.DeregisterService(serviceName, "localhost", mustParseInt(httpPort)); err != nil {
			suite.T().Logf("Failed to cleanup HTTP service: %v", err)
		}
	}

	if grpcPort != "" {
		if err := client.DeregisterService(serviceName+"-grpc", "localhost", mustParseInt(grpcPort)); err != nil {
			suite.T().Logf("Failed to cleanup gRPC service: %v", err)
		}
	}
}

// mustParseInt converts string to int, panics on error (for test cleanup)
func mustParseInt(s string) int {
	if s == "" {
		return 0
	}
	port := 0
	fmt.Sscanf(s, "%d", &port)
	return port
}

// TestServiceDiscoveryRegistration tests service registration with Consul
func (suite *DiscoveryIntegrationTestSuite) TestServiceDiscoveryRegistration() {
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	require.NoError(suite.T(), err)

	// Wait for registration to complete
	time.Sleep(2 * time.Second)

	// Verify services are registered in Consul using Consul API directly
	consulConfig := api.DefaultConfig()
	consulConfig.Address = suite.consulAddress
	consulClient, err := api.NewClient(consulConfig)
	require.NoError(suite.T(), err)
	services, err := consulClient.Agent().Services()
	require.NoError(suite.T(), err)

	// Look for our HTTP service
	httpServiceID := fmt.Sprintf("%s-localhost-%s", suite.config.Discovery.ServiceName, suite.httpPort)
	httpService, found := services[httpServiceID]
	assert.True(suite.T(), found, "HTTP service should be registered")
	if found {
		assert.Equal(suite.T(), suite.config.Discovery.ServiceName, httpService.Service)
		assert.Equal(suite.T(), "localhost", httpService.Address)
		assert.Equal(suite.T(), mustParseInt(suite.httpPort), httpService.Port)
	}

	// Check gRPC service registration (if on different port)
	if suite.grpcPort != suite.httpPort {
		grpcServiceID := fmt.Sprintf("%s-grpc-localhost-%s", suite.config.Discovery.ServiceName, suite.grpcPort)
		grpcService, found := services[grpcServiceID]
		assert.True(suite.T(), found, "gRPC service should be registered")
		if found {
			assert.Equal(suite.T(), suite.config.Discovery.ServiceName+"-grpc", grpcService.Service)
			assert.Equal(suite.T(), "localhost", grpcService.Address)
			assert.Equal(suite.T(), mustParseInt(suite.grpcPort), grpcService.Port)
		}
	}
}

// TestServiceDiscoveryDeregistration tests service deregistration from Consul
func (suite *DiscoveryIntegrationTestSuite) TestServiceDiscoveryDeregistration() {
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	require.NoError(suite.T(), err)

	// Wait for registration to complete
	time.Sleep(2 * time.Second)

	// Verify services are registered
	consulConfig := api.DefaultConfig()
	consulConfig.Address = suite.consulAddress
	consulClient, err := api.NewClient(consulConfig)
	require.NoError(suite.T(), err)

	services, err := consulClient.Agent().Services()
	require.NoError(suite.T(), err)

	httpServiceID := fmt.Sprintf("%s-localhost-%s", suite.config.Discovery.ServiceName, suite.httpPort)
	_, found := services[httpServiceID]
	assert.True(suite.T(), found, "HTTP service should be registered before stop")

	// Stop server
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()

	err = server.Stop(stopCtx)
	require.NoError(suite.T(), err)

	// Wait for deregistration to complete
	time.Sleep(2 * time.Second)

	// Verify services are deregistered
	services, err = consulClient.Agent().Services()
	require.NoError(suite.T(), err)

	// Check that our specific service instance is no longer registered
	_, found = services[httpServiceID]
	assert.False(suite.T(), found, "HTTP service should be deregistered after stop")

	// Check gRPC service deregistration (if on different port)
	if suite.grpcPort != suite.httpPort {
		grpcServiceID := fmt.Sprintf("%s-grpc-localhost-%s", suite.config.Discovery.ServiceName, suite.grpcPort)
		_, found = services[grpcServiceID]
		assert.False(suite.T(), found, "gRPC service should be deregistered after stop")
	}
}

// TestServiceDiscoveryHealthChecks tests health check integration with Consul
func (suite *DiscoveryIntegrationTestSuite) TestServiceDiscoveryHealthChecks() {
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	require.NoError(suite.T(), err)

	// Wait for registration and health checks to stabilize
	time.Sleep(5 * time.Second)

	// Check service health in Consul using Consul health API
	consulConfig := api.DefaultConfig()
	consulConfig.Address = suite.consulAddress
	consulClient, err := api.NewClient(consulConfig)
	require.NoError(suite.T(), err)

	healthyServices, _, err := consulClient.Health().Service(suite.config.Discovery.ServiceName, "", true, nil)
	require.NoError(suite.T(), err)

	// Verify our service is healthy
	found := false
	for _, service := range healthyServices {
		if service.Service.Port == mustParseInt(suite.httpPort) && service.Service.Address == "localhost" {
			found = true
			break
		}
	}
	assert.True(suite.T(), found, "HTTP service should be healthy in Consul")
}

// TestServiceDiscoveryFailureHandling tests graceful handling of discovery failures
func (suite *DiscoveryIntegrationTestSuite) TestServiceDiscoveryFailureHandling() {
	// Create configuration with invalid Consul address
	invalidConfig := &ServerConfig{
		ServiceName: "failure-test-service",
		HTTP: HTTPConfig{
			Port:         suite.findAvailablePort(),
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: GRPCConfig{
			Port:                suite.findAvailablePort(),
			EnableReflection:    true,
			EnableHealthService: true,
			Enabled:             true,
			MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
			MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
			KeepaliveParams: GRPCKeepaliveParams{
				MaxConnectionIdle:     15 * time.Minute,
				MaxConnectionAge:      30 * time.Minute,
				MaxConnectionAgeGrace: 5 * time.Minute,
				Time:                  5 * time.Minute,
				Timeout:               1 * time.Minute,
			},
			KeepalivePolicy: GRPCKeepalivePolicy{
				MinTime:             5 * time.Minute,
				PermitWithoutStream: false,
			},
		},
		ShutdownTimeout: 5 * time.Second,
		Discovery: DiscoveryConfig{
			Enabled:     true,
			Address:     "invalid-consul-address:8500", // Invalid address
			ServiceName: "failure-test-service",
			Tags:        []string{"test", "failure"},
		},
		Middleware: MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
	}

	server, err := NewBusinessServerCore(invalidConfig, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	// Start server - should succeed despite discovery failure
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	require.NoError(suite.T(), err, "Server should start successfully even with discovery failure")

	// Verify server is running
	assert.True(suite.T(), server.started)
	assert.NotEmpty(suite.T(), server.GetHTTPAddress())
	assert.NotEmpty(suite.T(), server.GetGRPCAddress())

	// Check discovery manager health
	discoveryManager := server.discoveryManager
	assert.NotNil(suite.T(), discoveryManager)

	// Discovery should report as unhealthy due to connection failure
	isHealthy := discoveryManager.IsHealthy(ctx)
	assert.False(suite.T(), isHealthy, "Discovery should be unhealthy with invalid address")
}

// TestMultipleEndpointRegistration tests registration of multiple endpoints
func (suite *DiscoveryIntegrationTestSuite) TestMultipleEndpointRegistration() {
	// Use different ports for HTTP and gRPC to ensure multiple registrations
	suite.config.HTTP.Port = suite.findAvailablePort()
	suite.config.GRPC.Port = suite.findAvailablePort()

	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	require.NoError(suite.T(), err)

	// Wait for registration to complete
	time.Sleep(2 * time.Second)

	// Verify both HTTP and gRPC services are registered
	consulConfig := api.DefaultConfig()
	consulConfig.Address = suite.consulAddress
	consulClient, err := api.NewClient(consulConfig)
	require.NoError(suite.T(), err)

	services, err := consulClient.Agent().Services()
	require.NoError(suite.T(), err)

	// Check HTTP service
	httpServiceID := fmt.Sprintf("%s-localhost-%s", suite.config.Discovery.ServiceName, suite.config.HTTP.Port)
	httpService, httpFound := services[httpServiceID]
	assert.True(suite.T(), httpFound, "HTTP service should be registered")

	// Check gRPC service
	grpcServiceID := fmt.Sprintf("%s-grpc-localhost-%s", suite.config.Discovery.ServiceName, suite.config.GRPC.Port)
	grpcService, grpcFound := services[grpcServiceID]
	assert.True(suite.T(), grpcFound, "gRPC service should be registered")

	// Verify service metadata
	if httpFound {
		assert.Equal(suite.T(), mustParseInt(suite.config.HTTP.Port), httpService.Port)
		assert.Equal(suite.T(), "localhost", httpService.Address)
	}

	if grpcFound {
		assert.Equal(suite.T(), mustParseInt(suite.config.GRPC.Port), grpcService.Port)
		assert.Equal(suite.T(), "localhost", grpcService.Address)
	}
}

// TestDiscoveryManagerHealthStatus tests discovery manager health status reporting
func (suite *DiscoveryIntegrationTestSuite) TestDiscoveryManagerHealthStatus() {
	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	require.NoError(suite.T(), err)

	// Wait for registration to complete
	time.Sleep(2 * time.Second)

	// Get discovery manager
	discoveryManager := server.discoveryManager
	require.NotNil(suite.T(), discoveryManager)

	// Check health status
	isHealthy := discoveryManager.IsHealthy(ctx)
	assert.True(suite.T(), isHealthy, "Discovery should be healthy with valid Consul")

	// Get detailed health status (if available)
	if impl, ok := discoveryManager.(*DiscoveryManagerImpl); ok {
		healthStatus := impl.GetHealthStatus()
		assert.NotNil(suite.T(), healthStatus)
		assert.True(suite.T(), healthStatus.Available)
		assert.Equal(suite.T(), 0, healthStatus.ConsecutiveErrors)
		assert.NotZero(suite.T(), healthStatus.LastCheck)
	}
}

// TestServiceDiscoveryRetryMechanism tests retry mechanism for discovery operations
func (suite *DiscoveryIntegrationTestSuite) TestServiceDiscoveryRetryMechanism() {
	// This test is more complex and would require temporarily disrupting Consul
	// For now, we'll test the retry configuration and basic retry logic

	server, err := NewBusinessServerCore(suite.config, suite.testService, suite.testDeps)
	require.NoError(suite.T(), err)
	suite.server = server

	// Verify discovery manager has retry configuration
	discoveryManager := server.discoveryManager
	require.NotNil(suite.T(), discoveryManager)

	if impl, ok := discoveryManager.(*DiscoveryManagerImpl); ok {
		assert.NotNil(suite.T(), impl.retryConfig)
		assert.Greater(suite.T(), impl.retryConfig.MaxRetries, 0)
		assert.Greater(suite.T(), impl.retryConfig.InitialInterval, time.Duration(0))
		assert.Greater(suite.T(), impl.retryConfig.MaxInterval, impl.retryConfig.InitialInterval)
		assert.Greater(suite.T(), impl.retryConfig.Multiplier, 1.0)
	}
}

// Run the discovery integration test suite
func TestDiscoveryIntegrationSuite(t *testing.T) {
	suite.Run(t, new(DiscoveryIntegrationTestSuite))
}

// TestServiceDiscoveryIntegrationWithConsul tests service discovery integration with actual Consul instance
func TestServiceDiscoveryIntegrationWithConsul(t *testing.T) {
	// Check if Consul is available
	consulAddress := os.Getenv("CONSUL_ADDRESS")
	if consulAddress == "" {
		consulAddress = "localhost:8500"
	}

	conn, err := net.DialTimeout("tcp", consulAddress, 2*time.Second)
	if err != nil {
		t.Skip("Consul not available for integration testing. Set CONSUL_ADDRESS environment variable or start Consul on localhost:8500")
	}
	conn.Close()

	// Initialize logger
	logger.InitLogger()

	// Find available ports
	httpPort := findAvailablePort(t)
	grpcPort := findAvailablePort(t)

	// Create configuration with discovery enabled
	config := &ServerConfig{
		ServiceName: "consul-integration-test",
		HTTP: HTTPConfig{
			Port:         httpPort,
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: GRPCConfig{
			Port:                grpcPort,
			EnableReflection:    true,
			EnableHealthService: true,
			Enabled:             true,
			MaxRecvMsgSize:      4 * 1024 * 1024, // 4MB
			MaxSendMsgSize:      4 * 1024 * 1024, // 4MB
			KeepaliveParams: GRPCKeepaliveParams{
				MaxConnectionIdle:     15 * time.Minute,
				MaxConnectionAge:      30 * time.Minute,
				MaxConnectionAgeGrace: 5 * time.Minute,
				Time:                  5 * time.Minute,
				Timeout:               1 * time.Minute,
			},
			KeepalivePolicy: GRPCKeepalivePolicy{
				MinTime:             5 * time.Minute,
				PermitWithoutStream: false,
			},
		},
		ShutdownTimeout: 5 * time.Second,
		Discovery: DiscoveryConfig{
			Enabled:     true,
			Address:     consulAddress,
			ServiceName: "consul-integration-test",
			Tags:        []string{"integration", "test", "consul"},
		},
		Middleware: MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
	}

	// Create test service registrar
	registrar := NewTestServiceRegistrar()

	// Add HTTP handler
	httpHandler := NewTestHTTPHandler("consul-test")
	httpHandler.AddRoute("/consul-test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "consul integration test"})
	})
	registrar.AddHTTPHandler(httpHandler)

	// Add gRPC service
	grpcService := NewTestGRPCService("consul-grpc-test")
	registrar.AddGRPCService(grpcService)

	// Create dependencies
	deps := NewTestDependencyContainer()

	// Create and start server
	server, err := NewBusinessServerCore(config, registrar, deps)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx)
	require.NoError(t, err)

	// Wait for registration
	time.Sleep(3 * time.Second)

	// Verify registration in Consul

	consulConfig := api.DefaultConfig()
	consulConfig.Address = consulAddress
	consulClient, err := api.NewClient(consulConfig)
	require.NoError(t, err)

	services, err := consulClient.Agent().Services()
	require.NoError(t, err)

	serviceID := fmt.Sprintf("consul-integration-test-localhost-%s", httpPort)
	service, found := services[serviceID]
	assert.True(t, found, "Service should be registered in Consul")

	// Verify service details
	if found {
		assert.Equal(t, "consul-integration-test", service.Service)
		assert.Equal(t, "localhost", service.Address)
		assert.Equal(t, mustParseInt(httpPort), service.Port)
	}

	// Test graceful shutdown and deregistration
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	err = server.Stop(shutdownCtx)
	require.NoError(t, err)

	err = server.Shutdown()
	require.NoError(t, err)

	// Wait for deregistration
	time.Sleep(2 * time.Second)

	// Verify deregistration
	services, err = consulClient.Agent().Services()
	require.NoError(t, err)

	_, found = services[serviceID]
	assert.False(t, found, "Service should be deregistered from Consul")
}

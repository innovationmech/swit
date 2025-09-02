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
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// EndToEndTestSuite provides comprehensive end-to-end tests for the base server framework
type EndToEndTestSuite struct {
	suite.Suite
	consulAddress string
	servers       []*BusinessServerImpl
	configs       []*ServerConfig
	services      []*TestServiceRegistrar
	dependencies  []*TestDependencyContainer
}

// SetupSuite initializes the test suite
func (suite *EndToEndTestSuite) SetupSuite() {
	// Initialize logger for tests
	logger.InitLogger()
	gin.SetMode(gin.TestMode)

	// Check if Consul is available for testing
	suite.consulAddress = os.Getenv("CONSUL_ADDRESS")
	if suite.consulAddress == "" {
		suite.consulAddress = "localhost:8500"
	}

	// Initialize slices
	suite.servers = make([]*BusinessServerImpl, 0)
	suite.configs = make([]*ServerConfig, 0)
	suite.services = make([]*TestServiceRegistrar, 0)
	suite.dependencies = make([]*TestDependencyContainer, 0)
}

// SetupTest sets up each test case
func (suite *EndToEndTestSuite) SetupTest() {
	// Clear any previous test data
	suite.servers = suite.servers[:0]
	suite.configs = suite.configs[:0]
	suite.services = suite.services[:0]
	suite.dependencies = suite.dependencies[:0]
}

// TearDownTest cleans up after each test case
func (suite *EndToEndTestSuite) TearDownTest() {
	// Stop all servers
	for _, server := range suite.servers {
		if server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := server.Stop(ctx); err != nil {
				suite.T().Logf("Error stopping server: %v", err)
			}
			cancel()

			if err := server.Shutdown(); err != nil {
				suite.T().Logf("Error shutting down server: %v", err)
			}
		}
	}

	// Clean up Consul services if available
	suite.cleanupConsulServices()
}

// isConsulAvailable checks if Consul is available for testing
func (suite *EndToEndTestSuite) isConsulAvailable() bool {
	conn, err := net.DialTimeout("tcp", suite.consulAddress, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// findAvailablePort finds an available port for testing
func (suite *EndToEndTestSuite) findAvailablePort() string {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(suite.T(), err)
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("%d", addr.Port)
}

// cleanupConsulServices removes any test services from Consul
func (suite *EndToEndTestSuite) cleanupConsulServices() {
	if !suite.isConsulAvailable() {
		return
	}

	consulConfig := api.DefaultConfig()
	consulConfig.Address = suite.consulAddress
	consulClient, err := api.NewClient(consulConfig)
	if err != nil {
		suite.T().Logf("Failed to create Consul client for cleanup: %v", err)
		return
	}

	// Get all services and remove test services
	services, err := consulClient.Agent().Services()
	if err != nil {
		suite.T().Logf("Failed to get services for cleanup: %v", err)
		return
	}

	for serviceID, service := range services {
		if service.Service == "e2e-test-service" || service.Service == "e2e-test-service-grpc" {
			if err := consulClient.Agent().ServiceDeregister(serviceID); err != nil {
				suite.T().Logf("Failed to cleanup service %s: %v", serviceID, err)
			}
		}
	}
}

// createTestServer creates a test server with the given configuration
func (suite *EndToEndTestSuite) createTestServer(serviceName string, httpPort, grpcPort string, enableDiscovery bool) (*BusinessServerImpl, *TestServiceRegistrar, *TestDependencyContainer) {
	// Create configuration
	config := &ServerConfig{
		ServiceName: serviceName,
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
			Enabled:     enableDiscovery && suite.isConsulAvailable(),
			Address:     suite.consulAddress,
			ServiceName: serviceName,
			Tags:        []string{"e2e", "test"},
		},
		Middleware: MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
	}

	// Create test dependencies
	deps := NewTestDependencyContainer()
	deps.AddService("test-db", fmt.Sprintf("mock-database-%s", serviceName))

	// Create test service registrar
	service := NewTestServiceRegistrar()

	// Add HTTP handler
	httpHandler := NewTestHTTPHandler(fmt.Sprintf("%s-http", serviceName))
	httpHandler.AddRoute("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "test response",
			"service": serviceName,
		})
	})
	service.AddHTTPHandler(httpHandler)

	// Add gRPC service
	grpcService := NewTestGRPCService(fmt.Sprintf("%s-grpc", serviceName))
	service.AddGRPCService(grpcService)

	// Add health check
	healthCheck := NewTestHealthCheck(serviceName, true)
	service.AddHealthCheck(healthCheck)

	// Create server
	server, err := NewBusinessServerCore(config, service, deps)
	require.NoError(suite.T(), err)

	// Store references for cleanup
	suite.servers = append(suite.servers, server)
	suite.configs = append(suite.configs, config)
	suite.services = append(suite.services, service)
	suite.dependencies = append(suite.dependencies, deps)

	return server, service, deps
}

// TestCompleteServerLifecycle tests the complete server lifecycle from creation to shutdown
func (suite *EndToEndTestSuite) TestCompleteServerLifecycle() {
	httpPort := suite.findAvailablePort()
	grpcPort := suite.findAvailablePort()

	server, service, deps := suite.createTestServer("e2e-test-service", httpPort, grpcPort, true)

	// Test server creation
	assert.NotNil(suite.T(), server)
	assert.NotNil(suite.T(), service)
	assert.False(suite.T(), server.started)
	assert.False(suite.T(), deps.IsInitialized())

	// Test server start
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server.Start(ctx)
	require.NoError(suite.T(), err)

	// Verify server is started
	assert.True(suite.T(), server.started)
	assert.True(suite.T(), deps.IsInitialized())
	assert.False(suite.T(), deps.IsClosed())

	// Verify addresses are available
	httpAddr := server.GetHTTPAddress()
	grpcAddr := server.GetGRPCAddress()
	assert.NotEmpty(suite.T(), httpAddr)
	assert.NotEmpty(suite.T(), grpcAddr)
	assert.Contains(suite.T(), httpAddr, httpPort)
	assert.Contains(suite.T(), grpcAddr, grpcPort)

	// Test HTTP endpoint
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/test", httpAddr))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Test gRPC endpoint
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(suite.T(), err)
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer healthCancel()

	healthResp, err := healthClient.Check(healthCtx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), grpc_health_v1.HealthCheckResponse_SERVING, healthResp.Status)

	// Test service discovery registration (if Consul is available)
	if suite.isConsulAvailable() {
		time.Sleep(2 * time.Second) // Wait for registration

		consulConfig := api.DefaultConfig()
		consulConfig.Address = suite.consulAddress
		consulClient, err := api.NewClient(consulConfig)
		require.NoError(suite.T(), err)

		services, err := consulClient.Agent().Services()
		require.NoError(suite.T(), err)

		serviceID := fmt.Sprintf("e2e-test-service-localhost-%s", httpPort)
		_, found := services[serviceID]
		assert.True(suite.T(), found, "Service should be registered in Consul")
	}

	// Test graceful stop
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()

	err = server.Stop(stopCtx)
	require.NoError(suite.T(), err)

	// Test shutdown
	err = server.Shutdown()
	require.NoError(suite.T(), err)

	// Verify cleanup
	assert.False(suite.T(), server.started)
	assert.True(suite.T(), deps.IsClosed())

	// Verify service discovery deregistration (if Consul is available)
	if suite.isConsulAvailable() {
		time.Sleep(2 * time.Second) // Wait for deregistration

		consulConfig := api.DefaultConfig()
		consulConfig.Address = suite.consulAddress
		consulClient, err := api.NewClient(consulConfig)
		require.NoError(suite.T(), err)

		services, err := consulClient.Agent().Services()
		require.NoError(suite.T(), err)

		serviceID := fmt.Sprintf("e2e-test-service-localhost-%s", httpPort)
		_, found := services[serviceID]
		assert.False(suite.T(), found, "Service should be deregistered from Consul")
	}
}

// TestMultipleServersWithDiscovery tests multiple servers with service discovery
func (suite *EndToEndTestSuite) TestMultipleServersWithDiscovery() {
	if !suite.isConsulAvailable() {
		suite.T().Skip("Consul not available for multiple servers test")
	}

	// Create multiple servers
	server1, _, _ := suite.createTestServer("e2e-test-service", suite.findAvailablePort(), suite.findAvailablePort(), true)
	server2, _, _ := suite.createTestServer("e2e-test-service", suite.findAvailablePort(), suite.findAvailablePort(), true)

	// Start both servers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server1.Start(ctx)
	require.NoError(suite.T(), err)

	err = server2.Start(ctx)
	require.NoError(suite.T(), err)

	// Wait for registration
	time.Sleep(3 * time.Second)

	// Verify both services are registered
	consulConfig := api.DefaultConfig()
	consulConfig.Address = suite.consulAddress
	consulClient, err := api.NewClient(consulConfig)
	require.NoError(suite.T(), err)

	services, err := consulClient.Agent().Services()
	require.NoError(suite.T(), err)

	// Count services with our service name
	serviceCount := 0
	for _, service := range services {
		if service.Service == "e2e-test-service" {
			serviceCount++
		}
	}

	assert.GreaterOrEqual(suite.T(), serviceCount, 2, "Should have at least 2 service instances registered")

	// Test service discovery health
	healthyServices, _, err := consulClient.Health().Service("e2e-test-service", "", true, nil)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(healthyServices), 2, "Should have at least 2 healthy service instances")

	// Test both servers are accessible
	client := &http.Client{Timeout: 5 * time.Second}

	resp1, err := client.Get(fmt.Sprintf("http://%s/test", server1.GetHTTPAddress()))
	require.NoError(suite.T(), err)
	defer resp1.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp1.StatusCode)

	resp2, err := client.Get(fmt.Sprintf("http://%s/test", server2.GetHTTPAddress()))
	require.NoError(suite.T(), err)
	defer resp2.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp2.StatusCode)
}

// TestServerRecoveryFromFailure tests server behavior during and after failures
func (suite *EndToEndTestSuite) TestServerRecoveryFromFailure() {
	httpPort := suite.findAvailablePort()
	grpcPort := suite.findAvailablePort()

	server, service, _ := suite.createTestServer("e2e-test-service", httpPort, grpcPort, false)

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server.Start(ctx)
	require.NoError(suite.T(), err)

	// Verify server is working
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/test", server.GetHTTPAddress()))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Simulate health check failure
	healthChecks := service.healthChecks
	if len(healthChecks) > 0 {
		if testHealthCheck, ok := healthChecks[0].(*TestHealthCheck); ok {
			testHealthCheck.SetHealthy(false)
		}
	}

	// Server should still be accessible (health check failure doesn't stop server)
	resp, err = client.Get(fmt.Sprintf("http://%s/test", server.GetHTTPAddress()))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Restore health
	if len(healthChecks) > 0 {
		if testHealthCheck, ok := healthChecks[0].(*TestHealthCheck); ok {
			testHealthCheck.SetHealthy(true)
		}
	}

	// Verify server is still working
	resp, err = client.Get(fmt.Sprintf("http://%s/test", server.GetHTTPAddress()))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

// TestConcurrentServerOperations tests concurrent operations on multiple servers
func (suite *EndToEndTestSuite) TestConcurrentServerOperations() {
	const numServers = 3
	const numRequestsPerServer = 10

	servers := make([]*BusinessServerImpl, numServers)

	// Create and start multiple servers
	for i := 0; i < numServers; i++ {
		httpPort := suite.findAvailablePort()
		grpcPort := suite.findAvailablePort()
		serviceName := fmt.Sprintf("e2e-concurrent-test-%d", i)

		server, _, _ := suite.createTestServer(serviceName, httpPort, grpcPort, false)
		servers[i] = server

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := server.Start(ctx)
		require.NoError(suite.T(), err)
	}

	// Make concurrent requests to all servers
	client := &http.Client{Timeout: 5 * time.Second}
	results := make(chan error, numServers*numRequestsPerServer)

	for i, server := range servers {
		for j := 0; j < numRequestsPerServer; j++ {
			go func(serverIndex, requestIndex int, srv *BusinessServerImpl) {
				resp, err := client.Get(fmt.Sprintf("http://%s/test", srv.GetHTTPAddress()))
				if err != nil {
					results <- fmt.Errorf("server %d request %d failed: %w", serverIndex, requestIndex, err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					results <- fmt.Errorf("server %d request %d returned status %d", serverIndex, requestIndex, resp.StatusCode)
					return
				}

				results <- nil
			}(i, j, server)
		}
	}

	// Collect results
	for i := 0; i < numServers*numRequestsPerServer; i++ {
		err := <-results
		assert.NoError(suite.T(), err)
	}
}

// Run the end-to-end test suite
func TestEndToEndSuite(t *testing.T) {
	suite.Run(t, new(EndToEndTestSuite))
}

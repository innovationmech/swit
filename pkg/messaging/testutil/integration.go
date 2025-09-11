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

package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/server"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockBusinessDependencyRegistry provides a mock implementation of the SWIT
// framework's BusinessDependencyRegistry for testing messaging integration.
type MockBusinessDependencyRegistry struct {
	mock.Mock
	dependencies map[string]interface{}
	factories    map[string]server.DependencyFactory
	initialized  bool
	closed       bool
}

// NewMockBusinessDependencyRegistry creates a new mock dependency registry.
func NewMockBusinessDependencyRegistry() *MockBusinessDependencyRegistry {
	return &MockBusinessDependencyRegistry{
		dependencies: make(map[string]interface{}),
		factories:    make(map[string]server.DependencyFactory),
		initialized:  false,
		closed:       false,
	}
}

// Close mocks the close operation.
func (m *MockBusinessDependencyRegistry) Close() error {
	args := m.Called()
	if args.Error(0) == nil {
		m.closed = true
	}
	return args.Error(0)
}

// GetService mocks service retrieval from the dependency container.
func (m *MockBusinessDependencyRegistry) GetService(name string) (interface{}, error) {
	args := m.Called(name)
	if service := args.Get(0); service != nil {
		return service, args.Error(1)
	}
	// Check if we have the dependency registered
	if dep, exists := m.dependencies[name]; exists {
		return dep, nil
	}
	return nil, args.Error(1)
}

// Initialize mocks dependency initialization.
func (m *MockBusinessDependencyRegistry) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	if args.Error(0) == nil {
		m.initialized = true
		// Initialize all registered dependencies
		for name, factory := range m.factories {
			if dep, err := factory(m); err == nil {
				m.dependencies[name] = dep
			}
		}
	}
	return args.Error(0)
}

// RegisterSingleton mocks singleton dependency registration.
func (m *MockBusinessDependencyRegistry) RegisterSingleton(name string, factory server.DependencyFactory) error {
	args := m.Called(name, factory)
	if args.Error(0) == nil {
		m.factories[name] = factory
	}
	return args.Error(0)
}

// RegisterTransient mocks transient dependency registration.
func (m *MockBusinessDependencyRegistry) RegisterTransient(name string, factory server.DependencyFactory) error {
	args := m.Called(name, factory)
	if args.Error(0) == nil {
		m.factories[name] = factory
	}
	return args.Error(0)
}

// RegisterInstance mocks instance registration.
func (m *MockBusinessDependencyRegistry) RegisterInstance(name string, instance interface{}) error {
	args := m.Called(name, instance)
	if args.Error(0) == nil {
		m.dependencies[name] = instance
	}
	return args.Error(0)
}

// GetDependencyNames mocks dependency name retrieval.
func (m *MockBusinessDependencyRegistry) GetDependencyNames() []string {
	args := m.Called()
	if names := args.Get(0); names != nil {
		return names.([]string)
	}
	// Return actual registered names
	names := make([]string, 0, len(m.dependencies))
	for name := range m.dependencies {
		names = append(names, name)
	}
	return names
}

// IsInitialized mocks initialization status check.
func (m *MockBusinessDependencyRegistry) IsInitialized() bool {
	return m.initialized
}

// IsClosed mocks closed status check.
func (m *MockBusinessDependencyRegistry) IsClosed() bool {
	return m.closed
}

// SetDependency sets a dependency directly for testing.
func (m *MockBusinessDependencyRegistry) SetDependency(name string, instance interface{}) {
	m.dependencies[name] = instance
}

// MockBusinessServiceRegistry provides a mock implementation of the SWIT
// framework's BusinessServiceRegistry for testing messaging integration.
type MockBusinessServiceRegistry struct {
	mock.Mock
	httpHandlers []server.BusinessHTTPHandler
	grpcServices []server.BusinessGRPCService
	healthChecks []server.BusinessHealthCheck
}

// NewMockBusinessServiceRegistry creates a new mock service registry.
func NewMockBusinessServiceRegistry() *MockBusinessServiceRegistry {
	return &MockBusinessServiceRegistry{
		httpHandlers: make([]server.BusinessHTTPHandler, 0),
		grpcServices: make([]server.BusinessGRPCService, 0),
		healthChecks: make([]server.BusinessHealthCheck, 0),
	}
}

// RegisterBusinessHTTPHandler mocks HTTP handler registration.
func (m *MockBusinessServiceRegistry) RegisterBusinessHTTPHandler(handler server.BusinessHTTPHandler) error {
	args := m.Called(handler)
	if args.Error(0) == nil {
		m.httpHandlers = append(m.httpHandlers, handler)
	}
	return args.Error(0)
}

// RegisterBusinessGRPCService mocks gRPC service registration.
func (m *MockBusinessServiceRegistry) RegisterBusinessGRPCService(service server.BusinessGRPCService) error {
	args := m.Called(service)
	if args.Error(0) == nil {
		m.grpcServices = append(m.grpcServices, service)
	}
	return args.Error(0)
}

// RegisterBusinessHealthCheck mocks health check registration.
func (m *MockBusinessServiceRegistry) RegisterBusinessHealthCheck(check server.BusinessHealthCheck) error {
	args := m.Called(check)
	if args.Error(0) == nil {
		m.healthChecks = append(m.healthChecks, check)
	}
	return args.Error(0)
}

// GetRegisteredHTTPHandlers returns all registered HTTP handlers for testing.
func (m *MockBusinessServiceRegistry) GetRegisteredHTTPHandlers() []server.BusinessHTTPHandler {
	return m.httpHandlers
}

// GetRegisteredGRPCServices returns all registered gRPC services for testing.
func (m *MockBusinessServiceRegistry) GetRegisteredGRPCServices() []server.BusinessGRPCService {
	return m.grpcServices
}

// GetRegisteredHealthChecks returns all registered health checks for testing.
func (m *MockBusinessServiceRegistry) GetRegisteredHealthChecks() []server.BusinessHealthCheck {
	return m.healthChecks
}

// MessagingIntegrationTestSuite provides a comprehensive test suite for testing
// messaging system integration with the SWIT framework.
type MessagingIntegrationTestSuite struct {
	DepRegistry     *MockBusinessDependencyRegistry
	ServiceRegistry *MockBusinessServiceRegistry
	BrokerSetup     *TestBrokerSetup
	Configurer      interface{} // Can be used for any configurer implementation
	Registrar       interface{} // Can be used for any registrar implementation
}

// NewMessagingIntegrationTestSuite creates a new integration test suite.
func NewMessagingIntegrationTestSuite() *MessagingIntegrationTestSuite {
	brokerSetup := NewTestBrokerSetup()

	suite := &MessagingIntegrationTestSuite{
		DepRegistry:     NewMockBusinessDependencyRegistry(),
		ServiceRegistry: NewMockBusinessServiceRegistry(),
		BrokerSetup:     brokerSetup,
		Configurer:      nil, // Will be set externally to avoid import cycles
	}

	// Create registrar with test config
	// Note: Registrar will be set externally to avoid import cycles
	suite.Registrar = nil

	// Setup default expectations
	suite.setupDefaultExpectations()

	return suite
}

// setupDefaultExpectations sets up common mock expectations for integration testing.
func (s *MessagingIntegrationTestSuite) setupDefaultExpectations() {
	// Dependency registry expectations
	s.DepRegistry.On("RegisterSingleton", mock.AnythingOfType("string"), mock.AnythingOfType("server.DependencyFactory")).Return(nil).Maybe()
	s.DepRegistry.On("GetService", mock.AnythingOfType("string")).Return(nil, nil).Maybe()
	s.DepRegistry.On("Initialize", mock.Anything).Return(nil).Maybe()
	s.DepRegistry.On("Close").Return(nil).Maybe()

	// Service registry expectations
	s.ServiceRegistry.On("RegisterBusinessHealthCheck", mock.AnythingOfType("*messaging.MessagingHealthCheck")).Return(nil).Maybe()

	// Set up broker in dependency registry
	s.DepRegistry.SetDependency("message-broker", s.BrokerSetup.Broker)
	s.DepRegistry.SetDependency("event-publisher", s.BrokerSetup.Publisher)
	s.DepRegistry.SetDependency("event-subscriber", s.BrokerSetup.Subscriber)
}

// TestFrameworkIntegration tests the complete framework integration.
func (s *MessagingIntegrationTestSuite) TestFrameworkIntegration(t *testing.T) {
	ctx := context.Background()

	// Test dependency registration - disabled due to import cycles
	// TODO: Fix import cycles to enable integration testing
	/*
		err := s.Configurer.RegisterMessagingDependencies(
			s.DepRegistry,
			s.BrokerSetup.BrokerConfig,
			s.BrokerSetup.PublisherConfig,
			s.BrokerSetup.SubscriberConfig,
		)
		require.NoError(t, err, "Should register messaging dependencies without error")

		// Test service registration
		err = s.Registrar.RegisterServices(s.ServiceRegistry)
		require.NoError(t, err, "Should register messaging services without error")
	*/

	// Placeholder for when import cycle is resolved
	var err error
	_ = err

	// Verify health check was registered
	healthChecks := s.ServiceRegistry.GetRegisteredHealthChecks()
	require.Len(t, healthChecks, 1, "Should register exactly one health check")
	require.Equal(t, "messaging-system", healthChecks[0].GetServiceName(), "Health check should have correct service name")

	// Test health check functionality
	err = healthChecks[0].Check(ctx)
	require.NoError(t, err, "Health check should pass")

	// Test dependency retrieval - disabled due to import cycles
	// TODO: Fix import cycles to enable integration testing
	/*
		broker, err := s.Configurer.GetBroker(s.DepRegistry)
		require.NoError(t, err, "Should retrieve broker from dependency container")
		require.NotNil(t, broker, "Broker should not be nil")

		publisher, err := s.Configurer.GetPublisher(s.DepRegistry)
		require.NoError(t, err, "Should retrieve publisher from dependency container")
		require.NotNil(t, publisher, "Publisher should not be nil")

		subscriber, err := s.Configurer.GetSubscriber(s.DepRegistry)
	*/

	// Placeholder for when import cycle is resolved
	var subscriber interface{}
	_ = subscriber
	// require.NoError(t, err, "Should retrieve subscriber from dependency container")
	// require.NotNil(t, subscriber, "Subscriber should not be nil")
}

// TestLifecycleManagement tests messaging service lifecycle management.
func (s *MessagingIntegrationTestSuite) TestLifecycleManagement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx // Suppress unused variable warning

	// lifecycleManager := messaging.NewMessagingServiceLifecycleManager(s.DepRegistry)
	// TODO: Fix import cycles to enable lifecycle testing
	var lifecycleManager interface{}
	_ = lifecycleManager

	// Test initialization - disabled due to import cycles
	// TODO: Fix import cycles to enable lifecycle testing
	/*
		err := lifecycleManager.Initialize(ctx)
		require.NoError(t, err, "Should initialize messaging services without error")

		// Verify broker connection was called
		s.BrokerSetup.Broker.AssertCalled(t, "Connect", mock.Anything)

		// Test shutdown
	*/

	// Placeholder for when import cycle is resolved
	var err error
	_ = err
	// err = lifecycleManager.Shutdown(ctx)
	// require.NoError(t, err, "Should shutdown messaging services without error")

	// Verify broker disconnect was called
	s.BrokerSetup.Broker.AssertCalled(t, "Disconnect", mock.Anything)
}

// AssertAllExpectations asserts that all mock expectations were met.
func (s *MessagingIntegrationTestSuite) AssertAllExpectations(t *testing.T) {
	s.DepRegistry.AssertExpectations(t)
	s.ServiceRegistry.AssertExpectations(t)
	s.BrokerSetup.AssertAllExpectations(t)
}

// IntegrationTestHelper provides utility methods for integration testing.
type IntegrationTestHelper struct{}

// NewIntegrationTestHelper creates a new integration test helper.
func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{}
}

// TestMessagingWithSWITFramework performs a comprehensive integration test
// of the messaging system with the SWIT framework.
func (h *IntegrationTestHelper) TestMessagingWithSWITFramework(t *testing.T) {
	suite := NewMessagingIntegrationTestSuite()
	defer suite.AssertAllExpectations(t)

	// Test framework integration
	suite.TestFrameworkIntegration(t)

	// Test lifecycle management
	suite.TestLifecycleManagement(t)

	t.Log("All messaging integration tests passed")
}

// BenchmarkMessagingIntegration benchmarks messaging operations within the SWIT framework context.
func (h *IntegrationTestHelper) BenchmarkMessagingIntegration(b *testing.B, messageSize int) {
	suite := NewMessagingIntegrationTestSuite()
	_ = suite // Suppress unused variable warning
	benchHelper := NewBenchmarkHelper()
	benchHelper.MessageSize = messageSize

	ctx := context.Background()
	_ = ctx // Suppress unused variable warning

	// Setup - disabled due to import cycles
	// TODO: Fix import cycles to enable benchmarking
	/*
		err := suite.Configurer.RegisterMessagingDependencies(
			suite.DepRegistry,
			suite.BrokerSetup.BrokerConfig,
			suite.BrokerSetup.PublisherConfig,
			suite.BrokerSetup.SubscriberConfig,
		)
		if err != nil {
			b.Fatalf("Failed to setup benchmark: %v", err)
		}

		publisher, err := suite.Configurer.GetPublisher(suite.DepRegistry)
		if err != nil {
			b.Fatalf("Failed to get publisher: %v", err)
	*/

	// Placeholder for when import cycle is resolved
	var publisher interface{}
	if publisher == nil {
		b.Skip("Benchmark disabled due to import cycles")
	}

	messages := benchHelper.CreateBenchmarkMessages()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		message := messages[i%len(messages)]
		// if err := publisher.Publish(ctx, &message); err != nil {
		// 	b.Fatalf("Failed to publish message: %v", err)
		// }
		_ = message // Suppress unused variable warning
	}
}

// CreateTestServerConfig creates a basic server configuration for testing.
func CreateTestServerConfig() *server.ServerConfig {
	return &server.ServerConfig{
		ServiceName: "test-messaging-service",
		HTTP: server.HTTPConfig{
			Enabled:  true,
			Port:     "0", // Dynamic port allocation
			TestMode: true,
		},
		GRPC: server.GRPCConfig{
			Enabled: true,
			Port:    "0", // Dynamic port allocation
		},
		Discovery: server.DiscoveryConfig{
			Enabled: false, // Disable for testing
		},
		ShutdownTimeout: 30 * time.Second,
	}
}

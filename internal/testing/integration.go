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

package testing

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/server"
)

// MessagingIntegrationTestSuite provides comprehensive integration testing
// for messaging framework components with the SWIT server framework
type MessagingIntegrationTestSuite struct {
	BaseServer       *server.BusinessServerImpl
	MessagingCoord   messaging.MessagingCoordinator
	TestBroker      *MockMessageBroker
	TestHandlers     []*MockEventHandler
	ServerConfig     *server.ServerConfig
	TestDependencies *TestDependencyContainer
	ServiceRegistrar *MessagingServiceRegistrar
	HTTPPort         string
	GRPCPort         string
	CancelFunc       context.CancelFunc
	ServerCtx        context.Context
	mock.Mock
}

// NewMessagingIntegrationTestSuite creates a new integration test suite
func NewMessagingIntegrationTestSuite() *MessagingIntegrationTestSuite {
	return &MessagingIntegrationTestSuite{
		TestHandlers: make([]*MockEventHandler, 0),
	}
}

// SetupSuite initializes the test suite
func (suite *MessagingIntegrationTestSuite) SetupSuite() {
	// Find available ports
	suite.HTTPPort = findAvailablePort()
	suite.GRPCPort = findAvailablePort()

	// Create test configuration
	suite.ServerConfig = &server.ServerConfig{
		ServiceName: "messaging-integration-test",
		HTTP: server.HTTPConfig{
			Port:         suite.HTTPPort,
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: server.GRPCConfig{
			Port:                suite.GRPCPort,
			EnableKeepalive:     true,
			EnableReflection:    true,
			EnableHealthService: true,
			Enabled:             true,
			MaxRecvMsgSize:      4 * 1024 * 1024,
			MaxSendMsgSize:      4 * 1024 * 1024,
		},
		ShutdownTimeout: 30 * time.Second,
		Discovery: server.DiscoveryConfig{
			Enabled: false,
		},
	}

	// Create messaging coordinator
	suite.MessagingCoord = messaging.NewMessagingCoordinator()
	suite.TestBroker = NewMockMessageBroker()

	// Create test dependencies
	suite.TestDependencies = NewTestDependencyContainer()
	suite.TestDependencies.AddService("test-messaging-db", "mock-messaging-database")

	// Create service registrar with messaging support
	suite.ServiceRegistrar = NewMessagingServiceRegistrar()

	// Add test handlers
	suite.addTestHandlers()

	// Create server context
	suite.ServerCtx, suite.CancelFunc = context.WithCancel(context.Background())
}

// TearDownSuite cleans up after the test suite
func (suite *MessagingIntegrationTestSuite) TearDownSuite() {
	if suite.CancelFunc != nil {
		suite.CancelFunc()
	}
}

// SetupTest sets up each test case
func (suite *MessagingIntegrationTestSuite) SetupTest() {
	// Reset mocks
	suite.TestBroker.ResetMocks()
	for _, handler := range suite.TestHandlers {
		handler.ResetMocks()
	}

	// Create fresh messaging coordinator for each test
	suite.MessagingCoord = messaging.NewMessagingCoordinator()
	suite.TestBroker = NewMockMessageBroker()

	// Create fresh service registrar
	suite.ServiceRegistrar = NewMessagingServiceRegistrar()
	suite.addTestHandlers()

	// Create fresh dependencies
	suite.TestDependencies = NewTestDependencyContainer()
	suite.TestDependencies.AddService("test-messaging-db", "mock-messaging-database")
}

// TearDownTest cleans up after each test case
func (suite *MessagingIntegrationTestSuite) TearDownTest() {
	if suite.BaseServer != nil {
		// Stop the server
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := suite.BaseServer.Stop(ctx); err != nil {
			suite.TestLogf("Error stopping server: %v", err)
		}

		if err := suite.BaseServer.Shutdown(); err != nil {
			suite.TestLogf("Error shutting down server: %v", err)
		}
	}

	if suite.MessagingCoord != nil && suite.MessagingCoord.IsStarted() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := suite.MessagingCoord.Stop(ctx); err != nil {
			suite.TestLogf("Error stopping messaging coordinator: %v", err)
		}
	}
}

// StartMessagingServer starts the base server with messaging integration
func (suite *MessagingIntegrationTestSuite) StartMessagingServer() error {
	var err error
	suite.BaseServer, err = server.NewBusinessServerCore(
		suite.ServerConfig,
		suite.ServiceRegistrar,
		suite.TestDependencies,
	)
	if err != nil {
		return fmt.Errorf("failed to create base server: %w", err)
	}

	// Start the base server
	if err := suite.BaseServer.Start(suite.ServerCtx); err != nil {
		return fmt.Errorf("failed to start base server: %w", err)
	}

	// Verify server is running
	if !suite.BaseServer.IsStarted() {
		return fmt.Errorf("server did not start properly")
	}

	return nil
}

// StartMessagingCoordinator starts the messaging coordinator with test components
func (suite *MessagingIntegrationTestSuite) StartMessagingCoordinator() error {
	// Register test broker
	if err := suite.MessagingCoord.RegisterBroker("test-broker", suite.TestBroker); err != nil {
		return fmt.Errorf("failed to register test broker: %w", err)
	}

	// Register test handlers
	for _, handler := range suite.TestHandlers {
		if err := suite.MessagingCoord.RegisterEventHandler(handler); err != nil {
			return fmt.Errorf("failed to register handler %s: %w", handler.GetHandlerID(), err)
		}
	}

	// Start the coordinator
	if err := suite.MessagingCoord.Start(suite.ServerCtx); err != nil {
		return fmt.Errorf("failed to start messaging coordinator: %w", err)
	}

	// Verify coordinator is started
	if !suite.MessagingCoord.IsStarted() {
		return fmt.Errorf("messaging coordinator did not start properly")
	}

	return nil
}

// StartFullIntegration starts both the server and messaging coordinator
func (suite *MessagingIntegrationTestSuite) StartFullIntegration() error {
	// Start messaging coordinator first
	if err := suite.StartMessagingCoordinator(); err != nil {
		return err
	}

	// Inject messaging coordinator into dependencies
	suite.TestDependencies.AddService("messaging-coordinator", suite.MessagingCoord)

	// Start the server
	if err := suite.StartMessagingServer(); err != nil {
		return err
	}

	return nil
}

// GetHTTPClient creates an HTTP client for testing server endpoints
func (suite *MessagingIntegrationTestSuite) GetHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

// GetGRPCConnection creates a gRPC connection for testing
func (suite *MessagingIntegrationTestSuite) GetGRPCConnection() (*grpc.ClientConn, error) {
	if suite.BaseServer == nil {
		return nil, fmt.Errorf("server not started")
	}

	grpcAddr := suite.BaseServer.GetGRPCAddress()
	return grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

// TestLogf logs test messages (equivalent to t.Logf)
func (suite *MessagingIntegrationTestSuite) TestLogf(format string, args ...interface{}) {
	suite.Called(fmt.Sprintf(format, args...))
}

// addTestHandlers adds standard test handlers to the suite
func (suite *MessagingIntegrationTestSuite) addTestHandlers() {
	// User event handler
	userHandler := &MockEventHandler{
		id:     "user-events-handler",
		topics: []string{"user.created", "user.updated", "user.deleted"},
		broker: "test-broker",
	}
	suite.TestHandlers = append(suite.TestHandlers, userHandler)
	suite.ServiceRegistrar.AddMessagingHandler(userHandler)

	// Order event handler
	orderHandler := &MockEventHandler{
		id:     "order-events-handler",
		topics: []string{"order.created", "order.cancelled", "order.completed"},
		broker: "test-broker",
	}
	suite.TestHandlers = append(suite.TestHandlers, orderHandler)
	suite.ServiceRegistrar.AddMessagingHandler(orderHandler)

	// Notification event handler
	notificationHandler := &MockEventHandler{
		id:     "notification-events-handler",
		topics: []string{"notification.send", "notification.delivered"},
		broker: "test-broker",
	}
	suite.TestHandlers = append(suite.TestHandlers, notificationHandler)
	suite.ServiceRegistrar.AddMessagingHandler(notificationHandler)
}

// MessagingServiceRegistrar extends TestServiceRegistrar with messaging support
type MessagingServiceRegistrar struct {
	httpHandlers     []server.BusinessHTTPHandler
	grpcServices     []server.BusinessGRPCService
	healthChecks     []server.BusinessHealthCheck
	messagingHandlers []messaging.EventHandler
}

// NewMessagingServiceRegistrar creates a new messaging service registrar
func NewMessagingServiceRegistrar() *MessagingServiceRegistrar {
	return &MessagingServiceRegistrar{
		httpHandlers:     make([]server.BusinessHTTPHandler, 0),
		grpcServices:     make([]server.BusinessGRPCService, 0),
		healthChecks:     make([]server.BusinessHealthCheck, 0),
		messagingHandlers: make([]messaging.EventHandler, 0),
	}
}

// RegisterServices implements server.BusinessServiceRegistrar
func (r *MessagingServiceRegistrar) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register HTTP handlers
	for _, handler := range r.httpHandlers {
		if err := registry.RegisterBusinessHTTPHandler(handler); err != nil {
			return fmt.Errorf("failed to register HTTP handler %s: %w", handler.GetServiceName(), err)
		}
	}

	// Register gRPC services
	for _, service := range r.grpcServices {
		if err := registry.RegisterBusinessGRPCService(service); err != nil {
			return fmt.Errorf("failed to register gRPC service %s: %w", service.GetServiceName(), err)
		}
	}

	// Register health checks
	for _, check := range r.healthChecks {
		if err := registry.RegisterBusinessHealthCheck(check); err != nil {
			return fmt.Errorf("failed to register health check %s: %w", check.GetServiceName(), err)
		}
	}

	return nil
}

// AddHTTPHandler adds an HTTP handler
func (r *MessagingServiceRegistrar) AddHTTPHandler(handler server.BusinessHTTPHandler) {
	r.httpHandlers = append(r.httpHandlers, handler)
}

// AddGRPCService adds a gRPC service
func (r *MessagingServiceRegistrar) AddGRPCService(service server.BusinessGRPCService) {
	r.grpcServices = append(r.grpcServices, service)
}

// AddHealthCheck adds a health check
func (r *MessagingServiceRegistrar) AddHealthCheck(check server.BusinessHealthCheck) {
	r.healthChecks = append(r.healthChecks, check)
}

// AddMessagingHandler adds a messaging handler
func (r *MessagingServiceRegistrar) AddMessagingHandler(handler messaging.EventHandler) {
	r.messagingHandlers = append(r.messagingHandlers, handler)
}

// GetMessagingHandlers returns all registered messaging handlers
func (r *MessagingServiceRegistrar) GetMessagingHandlers() []messaging.EventHandler {
	return r.messagingHandlers
}

// MockEventHandler extends messaging.MockEventHandler for integration testing
type MockEventHandler struct {
	messaging.MockEventHandler
	id           string
	topics       []string
	broker       string
	initialized  bool
	shutdown     bool
	messageCount int
	errorCount   int
	mu           sync.RWMutex
}

// NewMockEventHandler creates a new mock event handler
func NewMockEventHandler(id string, topics []string, broker string) *MockEventHandler {
	return &MockEventHandler{
		id:          id,
		topics:      topics,
		broker:      broker,
		initialized: false,
		shutdown:    false,
	}
}

// GetHandlerID returns the handler ID
func (m *MockEventHandler) GetHandlerID() string {
	return m.id
}

// GetTopics returns the handler topics
func (m *MockEventHandler) GetTopics() []string {
	return m.topics
}

// GetBrokerRequirement returns the broker requirement
func (m *MockEventHandler) GetBrokerRequirement() string {
	return m.broker
}

// Initialize initializes the handler
func (m *MockEventHandler) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initialized = true
	return nil
}

// Shutdown shuts down the handler
func (m *MockEventHandler) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shutdown = true
	return nil
}

// Handle handles a message
func (m *MockEventHandler) Handle(ctx context.Context, message *messaging.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messageCount++
	return nil
}

// OnError handles errors
func (m *MockEventHandler) OnError(ctx context.Context, message *messaging.Message, err error) messaging.ErrorAction {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCount++
	return messaging.ErrorActionAck
}

// GetMessageCount returns the number of messages processed
func (m *MockEventHandler) GetMessageCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.messageCount
}

// GetErrorCount returns the number of errors encountered
func (m *MockEventHandler) GetErrorCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorCount
}

// IsInitialized returns whether the handler is initialized
func (m *MockEventHandler) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized
}

// IsShutdown returns whether the handler is shut down
func (m *MockEventHandler) IsShutdown() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.shutdown
}

// MockMessageBroker extends messaging.MockMessageBroker for integration testing
type MockMessageBroker struct {
	messaging.MockMessageBroker
	connected    bool
	metrics      *messaging.BrokerMetrics
	healthStatus *messaging.HealthStatus
	capabilities *messaging.BrokerCapabilities
	mu           sync.RWMutex
}

// NewMockMessageBroker creates a new mock message broker
func NewMockMessageBroker() *MockMessageBroker {
	return &MockMessageBroker{
		connected: false,
		metrics: &messaging.BrokerMetrics{
			MessagesPublished:  0,
			MessagesConsumed:   0,
			ConnectionFailures: 0,
			PublishErrors:      0,
			ConsumeErrors:      0,
		},
		healthStatus: &messaging.HealthStatus{
			Status:  messaging.HealthStatusHealthy,
			Message: "Mock broker is healthy",
		},
		capabilities: &messaging.BrokerCapabilities{
			Publish:           true,
			Subscribe:         true,
			Topics:            true,
			Queues:            true,
			Persistence:       true,
			ExactlyOnce:       true,
			OrderingGuarantee: true,
		},
	}
}

// Connect connects the broker
func (m *MockMessageBroker) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

// Disconnect disconnects the broker
func (m *MockMessageBroker) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

// IsConnected returns whether the broker is connected
func (m *MockMessageBroker) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// GetMetrics returns broker metrics
func (m *MockMessageBroker) GetMetrics() *messaging.BrokerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
}

// HealthCheck performs a health check
func (m *MockMessageBroker) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.healthStatus, nil
}

// GetCapabilities returns broker capabilities
func (m *MockMessageBroker) GetCapabilities() *messaging.BrokerCapabilities {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.capabilities
}

// SetHealthStatus sets the health status
func (m *MockMessageBroker) SetHealthStatus(status messaging.HealthStatusType, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthStatus.Status = status
	m.healthStatus.Message = message
}

// IncrementMessageCount increments message metrics
func (m *MockMessageBroker) IncrementMessageCount(published, consumed int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics.MessagesPublished += int64(published)
	m.metrics.MessagesConsumed += int64(consumed)
}

// IncrementErrorCount increments error metrics
func (m *MockMessageBroker) IncrementErrorCount(connection, publish, consume int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics.ConnectionFailures += int64(connection)
	m.metrics.PublishErrors += int64(publish)
	m.metrics.ConsumeErrors += int64(consume)
}

// TestDependencyContainer extends server.TestDependencyContainer for messaging
type TestDependencyContainer struct {
	services    map[string]interface{}
	initialized bool
	closed      bool
	mu          sync.RWMutex
}

// NewTestDependencyContainer creates a new test dependency container
func NewTestDependencyContainer() *TestDependencyContainer {
	return &TestDependencyContainer{
		services:    make(map[string]interface{}),
		initialized: false,
		closed:      false,
	}
}

// GetService implements server.BusinessDependencyContainer
func (d *TestDependencyContainer) GetService(name string) (interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	if service, exists := d.services[name]; exists {
		return service, nil
	}
	return nil, fmt.Errorf("service %s not found", name)
}

// Close implements server.BusinessDependencyContainer
func (d *TestDependencyContainer) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	return nil
}

// Initialize implements server.BusinessDependencyRegistry
func (d *TestDependencyContainer) Initialize(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.initialized = true
	return nil
}

// AddService adds a service to the container
func (d *TestDependencyContainer) AddService(name string, service interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.services[name] = service
}

// IsInitialized returns whether the container is initialized
func (d *TestDependencyContainer) IsInitialized() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.initialized
}

// IsClosed returns whether the container is closed
func (d *TestDependencyContainer) IsClosed() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.closed
}

// findAvailablePort finds an available port for testing
func findAvailablePort() string {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(fmt.Sprintf("Failed to find available port: %v", err))
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("%d", addr.Port)
}
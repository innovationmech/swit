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

package messaging_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/messaging/testutil"
	"github.com/innovationmech/swit/pkg/server"
)

// Initialize logger for tests
func init() {
	logger.Logger, _ = zap.NewDevelopment()
}

// isCIEnvironment detects if we're running in a CI environment
func isCIEnvironment() bool {
	ciVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", 
		"TRAVIS", "JENKINS_URL", "CIRCLECI", "GITLAB_CI", 
		"AZURE_PIPELINES", "BUILDKITE",
	}
	for _, envVar := range ciVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	if os.Getenv("SWIT_TEST_FAST") != "" {
		return true
	}
	return false
}

// DependencyInjectionTestSuite tests messaging framework integration with dependency injection
type DependencyInjectionTestSuite struct {
	suite.Suite
	container        *server.SimpleBusinessDependencyContainer
	coordinator      messaging.MessagingCoordinator
	testServices     map[string]interface{}
	cleanupFunctions []func()
	isCI             bool
}

// BrokerTypeMock is a mock broker type for testing
var BrokerTypeMock messaging.BrokerType = "mock"

// SetupSuite initializes the test suite
func (suite *DependencyInjectionTestSuite) SetupSuite() {
	// Initialize logger for tests
	logger, _ := zap.NewDevelopment()

	// Test services that will be injected into messaging components
	suite.testServices = map[string]interface{}{
		"database": &MockDatabaseService{},
		"cache":    &MockCacheService{},
		"logger":   logger,
		"config":   &MockConfigService{},
		"metrics":  &MockMetricsService{},
	}
}

// TearDownSuite cleans up after the test suite
func (suite *DependencyInjectionTestSuite) TearDownSuite() {
	// Run all cleanup functions
	for _, cleanup := range suite.cleanupFunctions {
		cleanup()
	}
}

// getTestTimeout returns appropriate timeout based on environment
func (suite *DependencyInjectionTestSuite) getTestTimeout(normal, ci time.Duration) time.Duration {
	if suite.isCI {
		return ci
	}
	return normal
}

// SetupTest creates a fresh container for each test
func (suite *DependencyInjectionTestSuite) SetupTest() {
	suite.isCI = isCIEnvironment()
	suite.container = server.NewSimpleBusinessDependencyContainer()

	// Register a mock factory for inmemory broker type for testing
	mockBroker := testutil.NewMockMessageBroker()
	messaging.RegisterBrokerFactory(messaging.BrokerTypeInMemory, func(config *messaging.BrokerConfig) (messaging.MessageBroker, error) {
		return mockBroker, nil
	})
}

// TearDownTest cleans up after each test
func (suite *DependencyInjectionTestSuite) TearDownTest() {
	if suite.container != nil {
		suite.container.Close()
	}
}

// TestDependencyInjectionMessagingCoordinator tests that MessagingCoordinator can be created with dependency injection
func (suite *DependencyInjectionTestSuite) TestDependencyInjectionMessagingCoordinator() {
	// Test: Create messaging coordinator with dependency injection support

	// Register test services in container
	suite.registerTestServices()

	// Register messaging coordinator factory with correct type conversion
	coordinatorFactory := func(container server.BusinessDependencyContainer) (interface{}, error) {
		// Use the messaging factory function
		diContainer := &dependencyContainerAdapter{container: container}
		return messaging.CreateMessagingCoordinatorFactory()(diContainer)
	}

	err := suite.container.RegisterMessagingCoordinator(coordinatorFactory)
	require.NoError(suite.T(), err, "Should register messaging coordinator factory")

	// Initialize container
	initTimeout := suite.getTestTimeout(30*time.Second, 10*time.Second)
	initCtx, cancel := context.WithTimeout(context.Background(), initTimeout)
	defer cancel()
	
	err = suite.container.Initialize(initCtx)
	require.NoError(suite.T(), err, "Should initialize dependency container")

	// Get messaging coordinator
	coordinator, err := suite.container.GetMessagingCoordinator()
	require.NoError(suite.T(), err, "Should get messaging coordinator")
	assert.NotNil(suite.T(), coordinator, "Messaging coordinator should not be nil")

	// Test that coordinator implements the interface
	_, ok := coordinator.(messaging.MessagingCoordinator)
	assert.True(suite.T(), ok, "Coordinator should implement MessagingCoordinator interface")
}

// dependencyContainerAdapter adapts server.BusinessDependencyContainer to messaging.DependencyContainerProvider
type dependencyContainerAdapter struct {
	container server.BusinessDependencyContainer
}

func (a *dependencyContainerAdapter) GetService(name string) (interface{}, error) {
	return a.container.GetService(name)
}

func (a *dependencyContainerAdapter) Close() error {
	return a.container.Close()
}

// TestDependencyInjectionBroker tests that message brokers can be created with dependency injection
func (suite *DependencyInjectionTestSuite) TestDependencyInjectionBroker() {
	// Test: Create message broker with dependency injection support

	// Register test services
	suite.registerTestServices()

	// Register broker factory with correct type conversion
	brokerFactory := func(container server.BusinessDependencyContainer, config interface{}) (interface{}, error) {
		diContainer := &dependencyContainerAdapter{container: container}
		return messaging.CreateBrokerFactory("inmemory")(diContainer, config)
	}

	err := suite.container.RegisterBrokerFactory("mock", brokerFactory)
	require.NoError(suite.T(), err, "Should register broker factory")

	// Create broker config with proper defaults
	brokerConfig := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeInMemory,
		Endpoints: []string{"mock://localhost"},
		Connection: messaging.ConnectionConfig{
			Timeout:     30 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    10,
			IdleTimeout: 5 * time.Minute,
		},
		Retry: messaging.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
		Monitoring: messaging.MonitoringConfig{
			Enabled:             true,
			MetricsInterval:     30 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
		},
	}

	// Create broker using dependency injection
	broker, err := suite.container.CreateBroker("mock", brokerConfig)
	require.NoError(suite.T(), err, "Should create broker with dependency injection")
	assert.NotNil(suite.T(), broker, "Broker should not be nil")

	// Test that broker implements the interface
	_, ok := broker.(messaging.MessageBroker)
	assert.True(suite.T(), ok, "Broker should implement MessageBroker interface")
}

// TestDependencyInjectionEventHandler tests that event handlers can be created with dependency injection
func (suite *DependencyInjectionTestSuite) TestDependencyInjectionEventHandler() {
	// Test: Create event handler with dependency injection support

	// Register test services
	suite.registerTestServices()

	// Register event handler as singleton dependency
	err := suite.container.RegisterSingleton("di-handler", func(container server.BusinessDependencyContainer) (interface{}, error) {
		diContainer := &dependencyContainerAdapter{container: container}
		return NewDependencyAwareEventHandler(diContainer)
	})
	require.NoError(suite.T(), err, "Should register event handler")

	// Initialize container with timeout
	initTimeout := suite.getTestTimeout(30*time.Second, 10*time.Second)
	initCtx, cancel := context.WithTimeout(context.Background(), initTimeout)
	defer cancel()
	
	err = suite.container.Initialize(initCtx)
	require.NoError(suite.T(), err, "Should initialize dependency container")

	// Get event handler
	handler, err := suite.container.GetService("di-handler")
	require.NoError(suite.T(), err, "Should get event handler")
	assert.NotNil(suite.T(), handler, "Event handler should not be nil")

	// Test that handler implements the interface
	_, ok := handler.(messaging.EventHandler)
	assert.True(suite.T(), ok, "Handler should implement EventHandler interface")
}

// TestDependencyInjectionIntegration tests complete dependency injection integration for messaging
func (suite *DependencyInjectionTestSuite) TestDependencyInjectionIntegration() {
	// Test: Complete dependency injection integration with messaging coordinator

	// Register all components
	suite.registerTestServices()

	// Register messaging coordinator with correct type conversion
	coordinatorFactory := func(container server.BusinessDependencyContainer) (interface{}, error) {
		diContainer := &dependencyContainerAdapter{container: container}
		return messaging.CreateMessagingCoordinatorFactory()(diContainer)
	}
	err := suite.container.RegisterMessagingCoordinator(coordinatorFactory)
	require.NoError(suite.T(), err, "Should register messaging coordinator")

	// Register broker factory with correct type conversion
	brokerFactory := func(container server.BusinessDependencyContainer, config interface{}) (interface{}, error) {
		diContainer := &dependencyContainerAdapter{container: container}
		return messaging.CreateBrokerFactory("inmemory")(diContainer, config)
	}
	err = suite.container.RegisterBrokerFactory("mock", brokerFactory)
	require.NoError(suite.T(), err, "Should register broker factory")

	// Register event handler as singleton dependency
	err = suite.container.RegisterSingleton("di-handler", func(container server.BusinessDependencyContainer) (interface{}, error) {
		diContainer := &dependencyContainerAdapter{container: container}
		return NewDependencyAwareEventHandler(diContainer)
	})
	require.NoError(suite.T(), err, "Should register event handler")

	// Initialize container
	initTimeout := suite.getTestTimeout(30*time.Second, 10*time.Second)
	initCtx, cancel := context.WithTimeout(context.Background(), initTimeout)
	defer cancel()
	
	err = suite.container.Initialize(initCtx)
	require.NoError(suite.T(), err, "Should initialize dependency container")

	// Get messaging coordinator and start it
	coordinator, err := suite.container.GetMessagingCoordinator()
	require.NoError(suite.T(), err, "Should get messaging coordinator")

	coordinatorImpl, ok := coordinator.(messaging.MessagingCoordinator)
	require.True(suite.T(), ok, "Should be able to cast to MessagingCoordinator")

	// Start coordinator
	err = coordinatorImpl.Start(context.Background())
	require.NoError(suite.T(), err, "Should start coordinator")
	defer coordinatorImpl.Stop(context.Background())

	// Create and register a broker
	brokerConfig := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeInMemory,
		Endpoints: []string{"mock://localhost"},
		Connection: messaging.ConnectionConfig{
			Timeout:     30 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    10,
			IdleTimeout: 5 * time.Minute,
		},
		Retry: messaging.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
		Monitoring: messaging.MonitoringConfig{
			Enabled:             true,
			MetricsInterval:     30 * time.Second,
			HealthCheckInterval: 30 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
		},
	}

	broker, err := suite.container.CreateBroker("mock", brokerConfig)
	require.NoError(suite.T(), err, "Should create broker")

	brokerImpl, ok := broker.(messaging.MessageBroker)
	require.True(suite.T(), ok, "Should be able to cast to MessageBroker")

	// Register broker with coordinator
	err = coordinatorImpl.RegisterBroker("test-broker", brokerImpl)
	require.NoError(suite.T(), err, "Should register broker with coordinator")

	// Get and register event handler
	handler, err := suite.container.GetService("di-handler")
	require.NoError(suite.T(), err, "Should get event handler")

	handlerImpl, ok := handler.(messaging.EventHandler)
	require.True(suite.T(), ok, "Should be able to cast to EventHandler")

	// Register handler with coordinator
	err = coordinatorImpl.RegisterEventHandler(handlerImpl)
	require.NoError(suite.T(), err, "Should register event handler with coordinator")

	// Verify coordinator metrics
	metrics := coordinatorImpl.GetMetrics()
	assert.NotNil(suite.T(), metrics.StartedAt, "Coordinator should be started")
	assert.Equal(suite.T(), 1, metrics.BrokerCount, "Should have 1 broker")
	assert.Equal(suite.T(), 1, metrics.HandlerCount, "Should have 1 handler")
}

// TestDependencyInjectionLifecycle tests dependency injection lifecycle management
func (suite *DependencyInjectionTestSuite) TestDependencyInjectionLifecycle() {
	// Test: Dependency injection lifecycle management for messaging components

	// Register test services
	suite.registerTestServices()

	// Register a messaging coordinator with lifecycle-aware dependencies
	coordinatorFactory := func(container server.BusinessDependencyContainer) (interface{}, error) {
		diContainer := &dependencyContainerAdapter{container: container}
		return messaging.CreateMessagingCoordinatorFactory()(diContainer)
	}
	err := suite.container.RegisterMessagingCoordinator(coordinatorFactory)
	require.NoError(suite.T(), err, "Should register messaging coordinator factory")

	// Initialize container
	initTimeout := suite.getTestTimeout(30*time.Second, 10*time.Second)
	initCtx, cancel := context.WithTimeout(context.Background(), initTimeout)
	defer cancel()
	
	err = suite.container.Initialize(initCtx)
	require.NoError(suite.T(), err, "Should initialize dependency container")

	// Verify dependencies are initialized
	assert.True(suite.T(), suite.container.IsInitialized(), "Container should be initialized")

	// Get coordinator and start it
	coordinator, err := suite.container.GetMessagingCoordinator()
	require.NoError(suite.T(), err, "Should get messaging coordinator")

	coordinatorImpl, ok := coordinator.(messaging.MessagingCoordinator)
	require.True(suite.T(), ok, "Should be able to cast to MessagingCoordinator")

	// Start coordinator
	err = coordinatorImpl.Start(context.Background())
	require.NoError(suite.T(), err, "Should start coordinator")

	// Stop coordinator
	err = coordinatorImpl.Stop(context.Background())
	require.NoError(suite.T(), err, "Should stop coordinator")

	// Close container
	err = suite.container.Close()
	require.NoError(suite.T(), err, "Should close container")

	// Verify container is closed
	assert.True(suite.T(), suite.container.IsClosed(), "Container should be closed")
}

// TestDependencyInjectionErrorHandling tests error handling in dependency injection scenarios
func (suite *DependencyInjectionTestSuite) TestDependencyInjectionErrorHandling() {
	// Test: Error handling for dependency injection scenarios

	// Try to get non-existent service
	_, err := suite.container.GetService("non-existent")
	assert.Error(suite.T(), err, "Should return error for non-existent service")

	// Try to register duplicate service
	err = suite.container.RegisterSingleton("test", func(container server.BusinessDependencyContainer) (interface{}, error) {
		return "test-value", nil
	})
	require.NoError(suite.T(), err, "Should register service")

	// Try to register same service again
	err = suite.container.RegisterSingleton("test", func(container server.BusinessDependencyContainer) (interface{}, error) {
		return "test-value-2", nil
	})
	assert.Error(suite.T(), err, "Should return error for duplicate registration")

	// Try to register service after container is closed
	suite.container.Close()
	err = suite.container.RegisterSingleton("after-close", func(container server.BusinessDependencyContainer) (interface{}, error) {
		return "after-close-value", nil
	})
	assert.Error(suite.T(), err, "Should return error when registering after container is closed")
}

// TestDependencyInjectionConfiguration tests configuration injection for messaging
func (suite *DependencyInjectionTestSuite) TestDependencyInjectionConfiguration() {
	// Test: Configuration injection for messaging components

	// Create messaging configuration
	messagingConfig := &messaging.MessagingConfig{
		Brokers: map[string]*messaging.BrokerConfig{
			"test-broker": {
				Type:      messaging.BrokerTypeInMemory,
				Endpoints: []string{"mock://localhost"},
				Connection: messaging.ConnectionConfig{
					Timeout:     30 * time.Second,
					KeepAlive:   30 * time.Second,
					MaxAttempts: 3,
					PoolSize:    10,
					IdleTimeout: 5 * time.Minute,
				},
				Retry: messaging.RetryConfig{
					MaxAttempts:  3,
					InitialDelay: 1 * time.Second,
					MaxDelay:     30 * time.Second,
					Multiplier:   2.0,
					Jitter:       0.1,
				},
				Monitoring: messaging.MonitoringConfig{
					Enabled:             true,
					MetricsInterval:     30 * time.Second,
					HealthCheckInterval: 30 * time.Second,
					HealthCheckTimeout:  5 * time.Second,
				},
			},
		},
	}

	// Register configuration provider with type conversion
	err := suite.container.RegisterSingleton("messaging-config", func(container server.BusinessDependencyContainer) (interface{}, error) {
		diContainer := &dependencyContainerAdapter{container: container}
		return messaging.CreateMessagingConfigurationProvider(messagingConfig)(diContainer)
	})
	require.NoError(suite.T(), err, "Should register messaging configuration")

	// Register broker configurations with type conversion
	err = suite.container.RegisterSingleton("broker-configs", func(container server.BusinessDependencyContainer) (interface{}, error) {
		diContainer := &dependencyContainerAdapter{container: container}
		return messaging.CreateBrokerConfigurationProvider(messagingConfig.Brokers)(diContainer)
	})
	require.NoError(suite.T(), err, "Should register broker configurations")

	// Initialize container
	initTimeout := suite.getTestTimeout(30*time.Second, 10*time.Second)
	initCtx, cancel := context.WithTimeout(context.Background(), initTimeout)
	defer cancel()
	
	err = suite.container.Initialize(initCtx)
	require.NoError(suite.T(), err, "Should initialize dependency container")

	// Get and verify messaging configuration
	config, err := suite.container.GetService("messaging-config")
	require.NoError(suite.T(), err, "Should get messaging configuration")

	messagingConfigRetrieved, ok := config.(*messaging.MessagingConfig)
	require.True(suite.T(), ok, "Should be able to cast to MessagingConfig")
	assert.Equal(suite.T(), 1, len(messagingConfigRetrieved.Brokers))

	// Get and verify broker configurations
	brokerConfigs, err := suite.container.GetService("broker-configs")
	require.NoError(suite.T(), err, "Should get broker configurations")

	brokerConfigsRetrieved, ok := brokerConfigs.(map[string]*messaging.BrokerConfig)
	require.True(suite.T(), ok, "Should be able to cast to broker configs map")
	assert.Equal(suite.T(), 1, len(brokerConfigsRetrieved))
	assert.Equal(suite.T(), messaging.BrokerTypeInMemory, brokerConfigsRetrieved["test-broker"].Type)
}

// registerTestServices registers test services in the dependency container
func (suite *DependencyInjectionTestSuite) registerTestServices() {
	// Register database service
	err := suite.container.RegisterSingleton("database", func(container server.BusinessDependencyContainer) (interface{}, error) {
		return &MockDatabaseService{}, nil
	})
	require.NoError(suite.T(), err, "Should register database service")

	// Register cache service
	err = suite.container.RegisterSingleton("cache", func(container server.BusinessDependencyContainer) (interface{}, error) {
		return &MockCacheService{}, nil
	})
	require.NoError(suite.T(), err, "Should register cache service")

	// Register logger service
	err = suite.container.RegisterInstance("logger", suite.testServices["logger"])
	require.NoError(suite.T(), err, "Should register logger service")

	// Register config service
	err = suite.container.RegisterSingleton("config", func(container server.BusinessDependencyContainer) (interface{}, error) {
		return &MockConfigService{}, nil
	})
	require.NoError(suite.T(), err, "Should register config service")

	// Register metrics service
	err = suite.container.RegisterSingleton("metrics", func(container server.BusinessDependencyContainer) (interface{}, error) {
		return &MockMetricsService{}, nil
	})
	require.NoError(suite.T(), err, "Should register metrics service")
}

// MockDatabaseService is a mock database service for dependency injection testing
type MockDatabaseService struct {
	connected bool
	mu        sync.Mutex
}

func (m *MockDatabaseService) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *MockDatabaseService) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *MockDatabaseService) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// MockCacheService is a mock cache service for dependency injection testing
type MockCacheService struct {
	cache map[string]interface{}
	mu    sync.RWMutex
}

func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		cache: make(map[string]interface{}),
	}
}

func (m *MockCacheService) Set(key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[key] = value
	return nil
}

func (m *MockCacheService) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.cache[key]
	return value, exists
}

func (m *MockCacheService) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, key)
	return nil
}

// MockConfigService is a mock configuration service for dependency injection testing
type MockConfigService struct {
	config map[string]interface{}
	mu     sync.RWMutex
}

func NewMockConfigService() *MockConfigService {
	return &MockConfigService{
		config: make(map[string]interface{}),
	}
}

func (m *MockConfigService) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.config[key]
	return value, exists
}

func (m *MockConfigService) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config[key] = value
}

// MockMetricsService is a mock metrics service for dependency injection testing
type MockMetricsService struct {
	metrics map[string]float64
	mu      sync.RWMutex
}

func NewMockMetricsService() *MockMetricsService {
	return &MockMetricsService{
		metrics: make(map[string]float64),
	}
}

func (m *MockMetricsService) RecordMetric(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics[name] = value
}

func (m *MockMetricsService) GetMetric(name string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.metrics[name]
	return value, exists
}

func (m *MockMetricsService) GetAllMetrics() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]float64)
	for k, v := range m.metrics {
		result[k] = v
	}
	return result
}

// DependencyAwareEventHandler is an event handler that uses dependency injection
type DependencyAwareEventHandler struct {
	handlerID   string
	container   messaging.DependencyContainerProvider
	database    *MockDatabaseService
	cache       *MockCacheService
	logger      *zap.Logger
	config      *MockConfigService
	metrics     *MockMetricsService
	initialized bool
	mu          sync.Mutex
}

func NewDependencyAwareEventHandler(container messaging.DependencyContainerProvider) (*DependencyAwareEventHandler, error) {
	return &DependencyAwareEventHandler{
		handlerID: "di-aware-handler",
		container: container,
	}, nil
}

func (h *DependencyAwareEventHandler) GetHandlerID() string {
	return h.handlerID
}

func (h *DependencyAwareEventHandler) GetTopics() []string {
	return []string{"test.topic", "di.topic"}
}

func (h *DependencyAwareEventHandler) GetBrokerRequirement() string {
	return "" // No specific broker requirement
}

func (h *DependencyAwareEventHandler) OnError(ctx context.Context, message *messaging.Message, err error) messaging.ErrorAction {
	// Default error action: retry retryable errors, dead letter others
	var msgErr *messaging.MessagingError
	if errAs, ok := err.(*messaging.MessagingError); ok {
		msgErr = errAs
	} else {
		msgErr = &messaging.MessagingError{
			Code:    messaging.ErrInternal,
			Message: err.Error(),
			Cause:   err,
		}
	}

	if !msgErr.Retryable {
		return messaging.ErrorActionDeadLetter
	}
	return messaging.ErrorActionRetry
}

func (h *DependencyAwareEventHandler) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.initialized = false
	return nil
}

func (h *DependencyAwareEventHandler) Handle(ctx context.Context, message *messaging.Message) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.initialized {
		return fmt.Errorf("handler not initialized")
	}

	// Use injected dependencies
	if h.logger != nil {
		h.logger.Info("Processing message with injected dependencies",
			zap.String("handler_id", h.handlerID),
			zap.String("topic", message.Topic),
			zap.String("message_id", message.ID))
	}

	if h.metrics != nil {
		h.metrics.RecordMetric("messages_processed", 1.0)
	}

	if h.cache != nil {
		h.cache.Set("last_message_id", message.ID, time.Hour)
	}

	return nil
}

func (h *DependencyAwareEventHandler) Initialize(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Inject dependencies
	if err := h.injectDependencies(); err != nil {
		return fmt.Errorf("failed to inject dependencies: %w", err)
	}

	h.initialized = true
	return nil
}

func (h *DependencyAwareEventHandler) injectDependencies() error {
	// Get database service
	dbService, err := h.container.GetService("database")
	if err != nil {
		return fmt.Errorf("failed to get database service: %w", err)
	}
	if db, ok := dbService.(*MockDatabaseService); ok {
		h.database = db
	}

	// Get cache service
	cacheService, err := h.container.GetService("cache")
	if err != nil {
		return fmt.Errorf("failed to get cache service: %w", err)
	}
	if cache, ok := cacheService.(*MockCacheService); ok {
		h.cache = cache
	}

	// Get logger
	loggerService, err := h.container.GetService("logger")
	if err != nil {
		return fmt.Errorf("failed to get logger service: %w", err)
	}
	if logger, ok := loggerService.(*zap.Logger); ok {
		h.logger = logger
	}

	// Get config
	configService, err := h.container.GetService("config")
	if err != nil {
		return fmt.Errorf("failed to get config service: %w", err)
	}
	if config, ok := configService.(*MockConfigService); ok {
		h.config = config
	}

	// Get metrics
	metricsService, err := h.container.GetService("metrics")
	if err != nil {
		return fmt.Errorf("failed to get metrics service: %w", err)
	}
	if metrics, ok := metricsService.(*MockMetricsService); ok {
		h.metrics = metrics
	}

	return nil
}

// InjectDependencies implements the DependencyAware interface
func (h *DependencyAwareEventHandler) InjectDependencies(container messaging.DependencyContainerProvider) error {
	h.container = container
	return h.injectDependencies()
}

// Run the dependency injection integration test suite
func TestDependencyInjectionTestSuite(t *testing.T) {
	suite.Run(t, new(DependencyInjectionTestSuite))
}

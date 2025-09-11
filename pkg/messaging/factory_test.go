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

package messaging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMessageBrokerFactory tests the factory constructor.
func TestNewMessageBrokerFactory(t *testing.T) {
	factory := NewMessageBrokerFactory()
	if factory == nil {
		t.Error("Expected non-nil factory")
	}

	// Should be empty initially
	types := factory.GetSupportedBrokerTypes()
	if len(types) != 0 {
		t.Errorf("Expected empty supported types, got: %v", types)
	}
}

// TestGetDefaultFactory tests the default factory.
func TestGetDefaultFactory(t *testing.T) {
	factory := GetDefaultFactory()
	if factory == nil {
		t.Error("Expected non-nil default factory")
	}

	// Should be the same instance
	factory2 := GetDefaultFactory()
	if factory != factory2 {
		t.Error("Expected same factory instance")
	}
}

// TestRegisterBrokerFactory tests broker factory registration.
func TestRegisterBrokerFactory(t *testing.T) {
	factory := NewMessageBrokerFactory()

	// Register a mock factory
	mockFactory := func(config *BrokerConfig) (MessageBroker, error) {
		return &mockMessageBroker{}, nil
	}

	factory.(*messageBrokerFactoryImpl).RegisterBrokerFactory(BrokerTypeInMemory, mockFactory)

	// Check supported types
	types := factory.GetSupportedBrokerTypes()
	if len(types) != 1 {
		t.Errorf("Expected 1 supported type, got: %d", len(types))
	}

	if types[0] != BrokerTypeInMemory {
		t.Errorf("Expected %v, got: %v", BrokerTypeInMemory, types[0])
	}
}

// TestCreateBroker tests broker creation.
func TestCreateBroker(t *testing.T) {
	factory := NewMessageBrokerFactory()

	// Test with nil config
	_, err := factory.CreateBroker(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}

	// Register a mock factory
	mockFactory := func(config *BrokerConfig) (MessageBroker, error) {
		return &mockMessageBroker{}, nil
	}

	factory.(*messageBrokerFactoryImpl).RegisterBrokerFactory(BrokerTypeInMemory, mockFactory)

	// Test with valid config
	config := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:8080"},
		Connection: ConnectionConfig{
			Timeout:     10 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    10,
			IdleTimeout: 5 * time.Minute,
		},
		Retry: RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
	}

	broker, err := factory.CreateBroker(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if broker == nil {
		t.Error("Expected non-nil broker")
	}

	// Test with unsupported broker type
	config.Type = BrokerTypeKafka
	_, err = factory.CreateBroker(config)
	if err == nil {
		t.Error("Expected error for unsupported broker type")
	}

	if !IsConfigurationError(err) {
		t.Errorf("Expected configuration error, got: %v", err)
	}
}

// TestCreateBrokerWithFailingFactory tests broker creation with a failing factory.
func TestCreateBrokerWithFailingFactory(t *testing.T) {
	factory := NewMessageBrokerFactory()

	// Register a failing factory
	failingFactory := func(config *BrokerConfig) (MessageBroker, error) {
		return nil, errors.New("factory failed")
	}

	factory.(*messageBrokerFactoryImpl).RegisterBrokerFactory(BrokerTypeInMemory, failingFactory)

	config := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:8080"},
		Connection: ConnectionConfig{
			Timeout:     10 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    10,
			IdleTimeout: 5 * time.Minute,
		},
		Retry: RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
	}

	_, err := factory.CreateBroker(config)
	if err == nil {
		t.Error("Expected error from failing factory")
	}
}

// TestValidateConfig tests configuration validation.
func TestValidateConfig(t *testing.T) {
	factory := NewMessageBrokerFactory()

	// Test with nil config
	err := factory.ValidateConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}

	// Register a mock factory
	mockFactory := func(config *BrokerConfig) (MessageBroker, error) {
		return &mockMessageBroker{}, nil
	}

	factory.(*messageBrokerFactoryImpl).RegisterBrokerFactory(BrokerTypeInMemory, mockFactory)

	// Test with valid config
	config := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:8080"},
		Connection: ConnectionConfig{
			Timeout:     10 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    10,
			IdleTimeout: 5 * time.Minute,
		},
		Retry: RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
	}

	err = factory.ValidateConfig(config)
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}

	// Test with unsupported broker type
	config.Type = BrokerTypeKafka
	err = factory.ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for unsupported broker type")
	}

	// Test with invalid config
	config.Type = BrokerTypeInMemory
	config.Endpoints = nil // Invalid: no endpoints
	err = factory.ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

// TestGlobalFunctions tests the global convenience functions.
func TestGlobalFunctions(t *testing.T) {
	// Clean up the default factory for testing
	originalFactory := defaultFactory
	defer func() {
		defaultFactory = originalFactory
	}()

	// Replace with a test factory
	testFactory := NewMessageBrokerFactory()
	defaultFactory = testFactory.(*messageBrokerFactoryImpl)

	// Register a mock factory
	mockFactory := func(config *BrokerConfig) (MessageBroker, error) {
		return &mockMessageBroker{}, nil
	}

	RegisterBrokerFactory(BrokerTypeInMemory, mockFactory)

	// Test GetSupportedBrokerTypes
	types := GetSupportedBrokerTypes()
	if len(types) != 1 || types[0] != BrokerTypeInMemory {
		t.Errorf("Expected [%v], got: %v", BrokerTypeInMemory, types)
	}

	// Test NewMessageBroker
	config := &BrokerConfig{
		Type:      BrokerTypeInMemory,
		Endpoints: []string{"localhost:8080"},
		Connection: ConnectionConfig{
			Timeout:     10 * time.Second,
			KeepAlive:   30 * time.Second,
			MaxAttempts: 3,
			PoolSize:    10,
			IdleTimeout: 5 * time.Minute,
		},
		Retry: RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       0.1,
		},
	}

	broker, err := NewMessageBroker(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if broker == nil {
		t.Error("Expected non-nil broker")
	}

	// Test ValidateBrokerConfig
	err = ValidateBrokerConfig(config)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Test with invalid config
	config.Endpoints = nil
	err = ValidateBrokerConfig(config)
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

// TestConcurrentFactoryAccess tests concurrent access to the factory.
func TestConcurrentFactoryAccess(t *testing.T) {
	factory := NewMessageBrokerFactory()

	// Register multiple factories concurrently
	done := make(chan bool, 4)

	go func() {
		factory.(*messageBrokerFactoryImpl).RegisterBrokerFactory(BrokerTypeKafka, func(config *BrokerConfig) (MessageBroker, error) {
			return &mockMessageBroker{}, nil
		})
		done <- true
	}()

	go func() {
		factory.(*messageBrokerFactoryImpl).RegisterBrokerFactory(BrokerTypeNATS, func(config *BrokerConfig) (MessageBroker, error) {
			return &mockMessageBroker{}, nil
		})
		done <- true
	}()

	go func() {
		types := factory.GetSupportedBrokerTypes()
		// Should not panic
		_ = len(types)
		done <- true
	}()

	go func() {
		config := &BrokerConfig{
			Type:      BrokerTypeInMemory,
			Endpoints: []string{"localhost:8080"},
		}
		_ = factory.ValidateConfig(config)
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 4; i++ {
		<-done
	}

	// Verify both factories were registered
	types := factory.GetSupportedBrokerTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 registered factories, got: %d", len(types))
	}
}

// TestMessagingDependencyInjection tests the messaging dependency injection factory patterns
func TestMessagingDependencyInjection(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{"CreateMessagingCoordinatorFactory", testCreateMessagingCoordinatorFactory},
		{"CreateBrokerFactoryWithDI", testCreateBrokerFactoryWithDI},
		{"CreateEventHandlerFactoryWithDI", testCreateEventHandlerFactoryWithDI},
		{"DependencyInjectedCoordinator", testDependencyInjectedCoordinator},
		{"DependencyInjectedBroker", testDependencyInjectedBroker},
		{"DependencyInjectedEventHandler", testDependencyInjectedEventHandler},
		{"MessagingConfigurationProvider", testMessagingConfigurationProvider},
		{"BrokerConfigurationProvider", testBrokerConfigurationProvider},
	}

	for _, test := range tests {
		t.Run(test.name, test.test)
	}
}

func testCreateMessagingCoordinatorFactory(t *testing.T) {
	factory := CreateMessagingCoordinatorFactory()
	require.NotNil(t, factory)

	container := &mockDIContainer{}

	// Test successful creation
	coordinator, err := factory(container)
	assert.NoError(t, err)
	assert.NotNil(t, coordinator)

	// Verify it's wrapped
	wrapper, ok := coordinator.(*dependencyInjectedCoordinator)
	assert.True(t, ok)
	assert.NotNil(t, wrapper.MessagingCoordinator)
	assert.Equal(t, container, wrapper.container)
}

func testCreateBrokerFactoryWithDI(t *testing.T) {
	factory := CreateBrokerFactory("inmemory")
	require.NotNil(t, factory)

	container := &mockDIContainer{}

	// Test with valid config
	config := &BrokerConfig{
		Type: BrokerTypeInMemory,
	}

	broker, err := factory(container, config)
	if err != nil {
		t.Skipf("Skipping test as inmemory broker not registered: %v", err)
		return
	}
	assert.NotNil(t, broker)

	// Verify it's wrapped
	wrapper, ok := broker.(*dependencyInjectedBroker)
	assert.True(t, ok)
	assert.NotNil(t, wrapper.MessageBroker)
	assert.Equal(t, container, wrapper.container)

	// Test with invalid config type
	broker, err = factory(container, "invalid-config")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
	assert.Nil(t, broker)
}

func testCreateEventHandlerFactoryWithDI(t *testing.T) {
	handlerConstructor := func(container DependencyContainerProvider) (EventHandler, error) {
		return &mockDIEventHandler{handlerID: "test-handler"}, nil
	}

	factory := CreateEventHandlerFactory("test", handlerConstructor)
	require.NotNil(t, factory)

	container := &mockDIContainer{}

	// Test successful creation
	handler, err := factory(container)
	assert.NoError(t, err)
	assert.NotNil(t, handler)

	// Verify it's wrapped
	wrapper, ok := handler.(*dependencyInjectedEventHandler)
	assert.True(t, ok)
	assert.NotNil(t, wrapper.EventHandler)
	assert.Equal(t, container, wrapper.container)
	assert.Equal(t, "test-handler", wrapper.EventHandler.GetHandlerID())
}

func testDependencyInjectedCoordinator(t *testing.T) {
	baseCoordinator := &mockDIMessagingCoordinator{}
	container := &mockDIContainer{}

	wrapper := &dependencyInjectedCoordinator{
		MessagingCoordinator: baseCoordinator,
		container:            container,
	}

	// Test Start
	ctx := context.Background()
	err := wrapper.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, baseCoordinator.started)

	// Test RegisterEventHandler with DependencyAware handler
	dependencyAwareHandler := &mockDependencyAwareEventHandler{}
	err = wrapper.RegisterEventHandler(dependencyAwareHandler)
	assert.NoError(t, err)
	assert.True(t, dependencyAwareHandler.dependenciesInjected)
	assert.Contains(t, baseCoordinator.registeredHandlers, dependencyAwareHandler.GetHandlerID())

	// Test RegisterEventHandler with regular handler
	regularHandler := &mockDIEventHandler{handlerID: "regular"}
	err = wrapper.RegisterEventHandler(regularHandler)
	assert.NoError(t, err)
	assert.Contains(t, baseCoordinator.registeredHandlers, regularHandler.GetHandlerID())
}

func testDependencyInjectedBroker(t *testing.T) {
	baseBroker := &mockDIBroker{}
	container := &mockDIContainer{}

	wrapper := &dependencyInjectedBroker{
		MessageBroker: baseBroker,
		container:     container,
	}

	// Test Connect with non-DI broker
	ctx := context.Background()
	err := wrapper.Connect(ctx)
	assert.NoError(t, err)
	assert.True(t, baseBroker.connected)

	// Test Connect with DI-aware broker
	diBroker := &mockDependencyAwareBroker{}
	wrapper2 := &dependencyInjectedBroker{
		MessageBroker: diBroker,
		container:     container,
	}

	err = wrapper2.Connect(ctx)
	assert.NoError(t, err)
	assert.True(t, diBroker.connected)
	assert.True(t, diBroker.dependenciesInjected)
}

func testDependencyInjectedEventHandler(t *testing.T) {
	baseHandler := &mockDIEventHandler{handlerID: "test"}
	container := &mockDIContainer{}

	wrapper := &dependencyInjectedEventHandler{
		EventHandler: baseHandler,
		container:    container,
	}

	// Test Initialize with non-DI handler
	ctx := context.Background()
	err := wrapper.Initialize(ctx)
	assert.NoError(t, err)
	assert.True(t, baseHandler.initialized)

	// Test Initialize with DI-aware handler
	diHandler := &mockDependencyAwareEventHandler{}
	wrapper2 := &dependencyInjectedEventHandler{
		EventHandler: diHandler,
		container:    container,
	}

	err = wrapper2.Initialize(ctx)
	assert.NoError(t, err)
	assert.True(t, diHandler.initialized)
	assert.True(t, diHandler.dependenciesInjected)
}

func testMessagingConfigurationProvider(t *testing.T) {
	config := &MessagingConfig{
		Enabled: true,
		Brokers: map[string]*BrokerConfig{
			"kafka": {Type: BrokerTypeKafka},
		},
	}

	factory := CreateMessagingConfigurationProvider(config)
	require.NotNil(t, factory)

	container := &mockDIContainer{}

	// Test successful creation
	result, err := factory(container)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	resultConfig, ok := result.(*MessagingConfig)
	assert.True(t, ok)
	assert.True(t, resultConfig.Enabled)
	assert.Len(t, resultConfig.Brokers, 1)

	// Verify it's a copy (modification shouldn't affect original)
	resultConfig.Enabled = false
	assert.True(t, config.Enabled) // Original should be unchanged

	// Test with nil config
	factory2 := CreateMessagingConfigurationProvider(nil)
	result, err = factory2(container)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messaging configuration cannot be nil")
	assert.Nil(t, result)
}

func testBrokerConfigurationProvider(t *testing.T) {
	configs := map[string]*BrokerConfig{
		"kafka": {Type: BrokerTypeKafka},
		"nats":  {Type: BrokerTypeNATS},
	}

	factory := CreateBrokerConfigurationProvider(configs)
	require.NotNil(t, factory)

	container := &mockDIContainer{}

	// Test successful creation
	result, err := factory(container)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	resultConfigs, ok := result.(map[string]*BrokerConfig)
	assert.True(t, ok)
	assert.Len(t, resultConfigs, 2)
	assert.NotNil(t, resultConfigs["kafka"])
	assert.NotNil(t, resultConfigs["nats"])

	// Verify it's a copy (modification shouldn't affect original)
	resultConfigs["kafka"].Type = BrokerTypeInMemory
	assert.Equal(t, BrokerTypeKafka, configs["kafka"].Type) // Original should be unchanged

	// Test with nil configs
	factory2 := CreateBrokerConfigurationProvider(nil)
	result, err = factory2(container)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "broker configurations cannot be nil")
	assert.Nil(t, result)
}

// Mock implementations for dependency injection testing

type mockDIContainer struct {
	services map[string]interface{}
	closed   bool
}

func (m *mockDIContainer) GetService(name string) (interface{}, error) {
	if m.services == nil {
		m.services = make(map[string]interface{})
	}

	service, exists := m.services[name]
	if !exists {
		return nil, errors.New("service not found")
	}

	return service, nil
}

func (m *mockDIContainer) Close() error {
	m.closed = true
	return nil
}

type mockDIMessagingCoordinator struct {
	started            bool
	registeredHandlers map[string]EventHandler
}

func (m *mockDIMessagingCoordinator) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *mockDIMessagingCoordinator) Stop(ctx context.Context) error {
	m.started = false
	return nil
}

func (m *mockDIMessagingCoordinator) RegisterEventHandler(handler EventHandler) error {
	if m.registeredHandlers == nil {
		m.registeredHandlers = make(map[string]EventHandler)
	}
	m.registeredHandlers[handler.GetHandlerID()] = handler
	return nil
}

func (m *mockDIMessagingCoordinator) UnregisterEventHandler(handlerID string) error {
	if m.registeredHandlers != nil {
		delete(m.registeredHandlers, handlerID)
	}
	return nil
}

func (m *mockDIMessagingCoordinator) GetRegisteredHandlers() []string {
	handlers := make([]string, 0, len(m.registeredHandlers))
	for id := range m.registeredHandlers {
		handlers = append(handlers, id)
	}
	return handlers
}

func (m *mockDIMessagingCoordinator) IsStarted() bool {
	return m.started
}

func (m *mockDIMessagingCoordinator) RegisterBroker(name string, broker MessageBroker) error {
	return nil
}

func (m *mockDIMessagingCoordinator) GetBroker(name string) (MessageBroker, error) {
	return nil, nil
}

func (m *mockDIMessagingCoordinator) GetRegisteredBrokers() []string {
	return nil
}

func (m *mockDIMessagingCoordinator) GetMetrics() *MessagingCoordinatorMetrics {
	return &MessagingCoordinatorMetrics{}
}

func (m *mockDIMessagingCoordinator) HealthCheck(ctx context.Context) (*MessagingHealthStatus, error) {
	return &MessagingHealthStatus{Overall: "healthy"}, nil
}

func (m *mockDIMessagingCoordinator) GracefulShutdown(ctx context.Context, config *ShutdownConfig) (*GracefulShutdownManager, error) {
	manager := NewGracefulShutdownManager(m, config)
	return manager, nil
}

type mockDIBroker struct {
	connected bool
}

func (m *mockDIBroker) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *mockDIBroker) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *mockDIBroker) IsConnected() bool {
	return m.connected
}

func (m *mockDIBroker) Close() error {
	m.connected = false
	return nil
}

func (m *mockDIBroker) CreatePublisher(config PublisherConfig) (EventPublisher, error) {
	return nil, nil
}

func (m *mockDIBroker) CreateSubscriber(config SubscriberConfig) (EventSubscriber, error) {
	return nil, nil
}

func (m *mockDIBroker) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	return &HealthStatus{Status: HealthStatusHealthy}, nil
}

func (m *mockDIBroker) GetMetrics() *BrokerMetrics {
	return &BrokerMetrics{}
}

func (m *mockDIBroker) GetCapabilities() *BrokerCapabilities {
	return &BrokerCapabilities{}
}

type mockDIEventHandler struct {
	handlerID   string
	initialized bool
}

func (m *mockDIEventHandler) GetHandlerID() string {
	return m.handlerID
}

func (m *mockDIEventHandler) GetTopics() []string {
	return []string{"test-topic"}
}

func (m *mockDIEventHandler) GetBrokerRequirement() string {
	return ""
}

func (m *mockDIEventHandler) Initialize(ctx context.Context) error {
	m.initialized = true
	return nil
}

func (m *mockDIEventHandler) Shutdown(ctx context.Context) error {
	m.initialized = false
	return nil
}

func (m *mockDIEventHandler) Handle(ctx context.Context, message *Message) error {
	return nil
}

func (m *mockDIEventHandler) OnError(ctx context.Context, message *Message, err error) ErrorAction {
	return ErrorActionRetry
}

// DependencyAware mocks for dependency injection testing

type mockDependencyAwareBroker struct {
	mockDIBroker
	dependenciesInjected bool
}

func (m *mockDependencyAwareBroker) InjectDependencies(container DependencyContainerProvider) error {
	m.dependenciesInjected = true
	return nil
}

type mockDependencyAwareEventHandler struct {
	mockDIEventHandler
	dependenciesInjected bool
}

func (m *mockDependencyAwareEventHandler) InjectDependencies(container DependencyContainerProvider) error {
	m.dependenciesInjected = true
	return nil
}

func (m *mockDependencyAwareEventHandler) GetHandlerID() string {
	return "dependency-aware-handler"
}

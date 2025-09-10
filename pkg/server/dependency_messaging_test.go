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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessagingDependencyRegistry tests the messaging-specific dependency injection functionality
func TestMessagingDependencyRegistry(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{"RegisterMessagingCoordinator", testRegisterMessagingCoordinator},
		{"RegisterBrokerFactory", testRegisterBrokerFactory},
		{"RegisterEventHandlerFactory", testRegisterEventHandlerFactory},
		{"CreateBroker", testCreateBroker},
		{"GetMessagingCoordinator", testGetMessagingCoordinator},
		{"MessagingIntegration", testMessagingIntegration},
	}

	for _, test := range tests {
		t.Run(test.name, test.test)
	}
}

func testRegisterMessagingCoordinator(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()
	defer container.Close()

	// Test successful registration
	factory := func(container BusinessDependencyContainer) (interface{}, error) {
		return &mockMessagingCoordinator{}, nil
	}

	err := container.RegisterMessagingCoordinator(factory)
	assert.NoError(t, err)

	// Test duplicate registration
	err = container.RegisterMessagingCoordinator(factory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test registration after closing
	container.Close()
	err = container.RegisterMessagingCoordinator(factory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container is closed")
}

func testRegisterBrokerFactory(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()
	defer container.Close()

	// Test successful registration
	factory := func(container BusinessDependencyContainer, config interface{}) (interface{}, error) {
		return &mockBroker{}, nil
	}

	err := container.RegisterBrokerFactory("kafka", factory)
	assert.NoError(t, err)

	// Test duplicate registration
	err = container.RegisterBrokerFactory("kafka", factory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test empty broker type
	err = container.RegisterBrokerFactory("", factory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "broker type cannot be empty")

	// Test nil factory
	err = container.RegisterBrokerFactory("nats", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "broker factory cannot be nil")

	// Test registration after closing
	container.Close()
	err = container.RegisterBrokerFactory("rabbitmq", factory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container is closed")
}

func testRegisterEventHandlerFactory(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()
	defer container.Close()

	// Test successful registration
	factory := func(container BusinessDependencyContainer) (interface{}, error) {
		return &mockDIEventHandler{}, nil
	}

	err := container.RegisterEventHandlerFactory("user-handler", factory)
	assert.NoError(t, err)

	// Test duplicate registration
	err = container.RegisterEventHandlerFactory("user-handler", factory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test empty handler ID
	err = container.RegisterEventHandlerFactory("", factory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler ID cannot be empty")

	// Test nil factory
	err = container.RegisterEventHandlerFactory("order-handler", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event handler factory cannot be nil")

	// Test registration after closing
	container.Close()
	err = container.RegisterEventHandlerFactory("payment-handler", factory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container is closed")
}

func testCreateBroker(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()
	defer container.Close()

	// Register a broker factory
	factory := func(container BusinessDependencyContainer, config interface{}) (interface{}, error) {
		return &mockBroker{}, nil
	}

	err := container.RegisterBrokerFactory("kafka", factory)
	require.NoError(t, err)

	// Test successful broker creation
	broker, err := container.CreateBroker("kafka", &mockBrokerConfig{})
	assert.NoError(t, err)
	assert.NotNil(t, broker)

	// Test broker creation with unregistered type
	broker, err = container.CreateBroker("unregistered", &mockBrokerConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no factory registered")
	assert.Nil(t, broker)
}

func testGetMessagingCoordinator(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()
	defer container.Close()

	// Register messaging coordinator
	factory := func(container BusinessDependencyContainer) (interface{}, error) {
		return &mockMessagingCoordinator{}, nil
	}

	err := container.RegisterMessagingCoordinator(factory)
	require.NoError(t, err)

	// Initialize container
	ctx := context.Background()
	err = container.Initialize(ctx)
	require.NoError(t, err)

	// Test successful retrieval
	coordinator, err := container.GetMessagingCoordinator()
	assert.NoError(t, err)
	assert.NotNil(t, coordinator)

	// Test retrieval without registration
	container2 := NewSimpleBusinessDependencyContainer()
	defer container2.Close()

	err = container2.Initialize(ctx)
	require.NoError(t, err)

	coordinator, err = container2.GetMessagingCoordinator()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Nil(t, coordinator)
}

func testMessagingIntegration(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()
	defer container.Close()

	// Register messaging components
	coordinatorFactory := func(container BusinessDependencyContainer) (interface{}, error) {
		return &mockMessagingCoordinator{}, nil
	}

	brokerFactory := func(container BusinessDependencyContainer, config interface{}) (interface{}, error) {
		return &mockBroker{}, nil
	}

	handlerFactory := func(container BusinessDependencyContainer) (interface{}, error) {
		return &mockDIEventHandler{}, nil
	}

	// Register all components
	err := container.RegisterMessagingCoordinator(coordinatorFactory)
	require.NoError(t, err)

	err = container.RegisterBrokerFactory("kafka", brokerFactory)
	require.NoError(t, err)

	err = container.RegisterEventHandlerFactory("user-handler", handlerFactory)
	require.NoError(t, err)

	// Initialize container
	ctx := context.Background()
	err = container.Initialize(ctx)
	require.NoError(t, err)

	// Test that all components can be retrieved
	coordinator, err := container.GetMessagingCoordinator()
	assert.NoError(t, err)
	assert.NotNil(t, coordinator)

	broker, err := container.CreateBroker("kafka", &mockBrokerConfig{})
	assert.NoError(t, err)
	assert.NotNil(t, broker)

	// Verify service names are registered
	dependencyNames := container.GetDependencyNames()
	assert.Contains(t, dependencyNames, "messaging-coordinator")
}

// Mock implementations for testing

type mockMessagingCoordinator struct {
	started bool
}

func (m *mockMessagingCoordinator) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *mockMessagingCoordinator) Stop(ctx context.Context) error {
	m.started = false
	return nil
}

func (m *mockMessagingCoordinator) IsStarted() bool {
	return m.started
}

type mockBroker struct {
	connected bool
}

func (m *mockBroker) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *mockBroker) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *mockBroker) IsConnected() bool {
	return m.connected
}

func (m *mockBroker) Close() error {
	m.connected = false
	return nil
}

type mockDIEventHandler struct {
	initialized bool
	handlerID   string
}

func (m *mockDIEventHandler) Initialize(ctx context.Context) error {
	m.initialized = true
	return nil
}

func (m *mockDIEventHandler) Shutdown(ctx context.Context) error {
	m.initialized = false
	return nil
}

func (m *mockDIEventHandler) GetHandlerID() string {
	if m.handlerID == "" {
		return "mock-handler"
	}
	return m.handlerID
}

func (m *mockDIEventHandler) GetTopics() []string {
	return []string{"test-topic"}
}

func (m *mockDIEventHandler) GetBrokerRequirement() string {
	return ""
}

func (m *mockDIEventHandler) Handle(ctx context.Context, message interface{}) error {
	return nil
}

func (m *mockDIEventHandler) OnError(ctx context.Context, message interface{}, err error) interface{} {
	return nil
}

type mockBrokerConfig struct{}

// TestMessagingDependencyLifecycle tests the lifecycle management of messaging dependencies
func TestMessagingDependencyLifecycle(t *testing.T) {
	container := NewSimpleBusinessDependencyContainer()

	// Register messaging coordinator with lifecycle dependency
	lifecycleCoordinator := &mockLifecycleMessagingCoordinator{}
	coordinatorFactory := func(container BusinessDependencyContainer) (interface{}, error) {
		return lifecycleCoordinator, nil
	}

	err := container.RegisterMessagingCoordinator(coordinatorFactory)
	require.NoError(t, err)

	// Initialize container (should call Initialize on lifecycle dependencies)
	ctx := context.Background()
	err = container.Initialize(ctx)
	require.NoError(t, err)
	assert.True(t, lifecycleCoordinator.initialized)

	// Close container (should call Shutdown on lifecycle dependencies)
	err = container.Close()
	require.NoError(t, err)
	assert.True(t, lifecycleCoordinator.shutdown)
}

// Mock implementation with lifecycle support
type mockLifecycleMessagingCoordinator struct {
	mockMessagingCoordinator
	initialized bool
	shutdown    bool
}

func (m *mockLifecycleMessagingCoordinator) Initialize(ctx context.Context) error {
	m.initialized = true
	return nil
}

func (m *mockLifecycleMessagingCoordinator) Shutdown(ctx context.Context) error {
	m.shutdown = true
	return nil
}

// TestMessagingDependencyContainerBuilder tests the builder pattern with messaging capabilities
func TestMessagingDependencyContainerBuilder(t *testing.T) {
	// Create builder and add messaging components
	builder := NewBusinessDependencyContainerBuilder()

	// Add regular dependency
	builder.AddSingleton("database", func(container BusinessDependencyContainer) (interface{}, error) {
		return &mockDatabase{}, nil
	})

	// Build container and cast to messaging registry
	container := builder.Build().(*SimpleBusinessDependencyContainer)
	defer container.Close()

	// Register messaging components
	coordinatorFactory := func(container BusinessDependencyContainer) (interface{}, error) {
		return &mockMessagingCoordinator{}, nil
	}

	err := container.RegisterMessagingCoordinator(coordinatorFactory)
	require.NoError(t, err)

	// Initialize and verify both regular and messaging dependencies
	ctx := context.Background()
	err = container.Initialize(ctx)
	require.NoError(t, err)

	database, err := container.GetService("database")
	assert.NoError(t, err)
	assert.NotNil(t, database)

	coordinator, err := container.GetMessagingCoordinator()
	assert.NoError(t, err)
	assert.NotNil(t, coordinator)
}

type mockDatabase struct{}

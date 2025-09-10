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

package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEventHandler implements the EventHandler interface for testing
type mockEventHandler struct {
	handlerID         string
	topics            []string
	brokerRequirement string
}

func (m *mockEventHandler) GetHandlerID() string {
	return m.handlerID
}

func (m *mockEventHandler) GetTopics() []string {
	return m.topics
}

func (m *mockEventHandler) GetBrokerRequirement() string {
	return m.brokerRequirement
}

func (m *mockEventHandler) Initialize(ctx context.Context) error {
	return nil
}

func (m *mockEventHandler) Shutdown(ctx context.Context) error {
	return nil
}

func (m *mockEventHandler) Handle(ctx context.Context, message interface{}) error {
	return nil
}

func (m *mockEventHandler) OnError(ctx context.Context, message interface{}, err error) interface{} {
	return nil
}

// mockEventHandlerRegistry implements the EventHandlerRegistry interface for testing
type mockEventHandlerRegistry struct {
	handlers map[string]EventHandler
}

func newMockEventHandlerRegistry() *mockEventHandlerRegistry {
	return &mockEventHandlerRegistry{
		handlers: make(map[string]EventHandler),
	}
}

func (m *mockEventHandlerRegistry) RegisterEventHandler(handler EventHandler) error {
	if handler == nil {
		return assert.AnError
	}
	handlerID := handler.GetHandlerID()
	if handlerID == "" {
		return assert.AnError
	}
	if _, exists := m.handlers[handlerID]; exists {
		return assert.AnError
	}
	m.handlers[handlerID] = handler
	return nil
}

func (m *mockEventHandlerRegistry) UnregisterEventHandler(handlerID string) error {
	if handlerID == "" {
		return assert.AnError
	}
	if _, exists := m.handlers[handlerID]; !exists {
		return assert.AnError
	}
	delete(m.handlers, handlerID)
	return nil
}

func (m *mockEventHandlerRegistry) GetRegisteredHandlers() []string {
	var handlers []string
	for id := range m.handlers {
		handlers = append(handlers, id)
	}
	return handlers
}

// simpleBusinessServiceRegistry implements BusinessServiceRegistry for testing
type simpleBusinessServiceRegistry struct{}

func (m *simpleBusinessServiceRegistry) RegisterBusinessHTTPHandler(handler BusinessHTTPHandler) error {
	return nil
}

func (m *simpleBusinessServiceRegistry) RegisterBusinessGRPCService(service BusinessGRPCService) error {
	return nil
}

func (m *simpleBusinessServiceRegistry) RegisterBusinessHealthCheck(check BusinessHealthCheck) error {
	return nil
}

// mockMessagingServiceRegistrar implements MessagingServiceRegistrar for testing
type mockMessagingServiceRegistrar struct {
	metadata *EventHandlerMetadata
}

func (m *mockMessagingServiceRegistrar) RegisterServices(registry BusinessServiceRegistry) error {
	return nil
}

func (m *mockMessagingServiceRegistrar) RegisterEventHandlers(registry EventHandlerRegistry) error {
	handler := &mockEventHandler{
		handlerID: "test-handler",
		topics:    []string{"test-topic"},
	}
	return registry.RegisterEventHandler(handler)
}

func (m *mockMessagingServiceRegistrar) GetEventHandlerMetadata() *EventHandlerMetadata {
	if m.metadata == nil {
		return &EventHandlerMetadata{
			HandlerCount:       1,
			Topics:             []string{"test-topic"},
			BrokerRequirements: []string{},
			Description:        "Test event handler",
		}
	}
	return m.metadata
}

func TestEventHandlerRegistry(t *testing.T) {
	registry := newMockEventHandlerRegistry()

	t.Run("RegisterEventHandler", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "test-handler-1",
			topics:    []string{"topic1", "topic2"},
		}

		err := registry.RegisterEventHandler(handler)
		assert.NoError(t, err)

		handlers := registry.GetRegisteredHandlers()
		assert.Contains(t, handlers, "test-handler-1")
	})

	t.Run("RegisterEventHandler_NilHandler", func(t *testing.T) {
		err := registry.RegisterEventHandler(nil)
		assert.Error(t, err)
	})

	t.Run("RegisterEventHandler_EmptyID", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "",
			topics:    []string{"topic1"},
		}

		err := registry.RegisterEventHandler(handler)
		assert.Error(t, err)
	})

	t.Run("RegisterEventHandler_DuplicateID", func(t *testing.T) {
		handler1 := &mockEventHandler{
			handlerID: "duplicate-handler",
			topics:    []string{"topic1"},
		}
		handler2 := &mockEventHandler{
			handlerID: "duplicate-handler",
			topics:    []string{"topic2"},
		}

		err := registry.RegisterEventHandler(handler1)
		assert.NoError(t, err)

		err = registry.RegisterEventHandler(handler2)
		assert.Error(t, err)
	})

	t.Run("UnregisterEventHandler", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "handler-to-remove",
			topics:    []string{"topic1"},
		}

		err := registry.RegisterEventHandler(handler)
		require.NoError(t, err)

		err = registry.UnregisterEventHandler("handler-to-remove")
		assert.NoError(t, err)

		handlers := registry.GetRegisteredHandlers()
		assert.NotContains(t, handlers, "handler-to-remove")
	})

	t.Run("UnregisterEventHandler_EmptyID", func(t *testing.T) {
		err := registry.UnregisterEventHandler("")
		assert.Error(t, err)
	})

	t.Run("UnregisterEventHandler_NotFound", func(t *testing.T) {
		err := registry.UnregisterEventHandler("non-existent-handler")
		assert.Error(t, err)
	})
}

func TestMessagingServiceRegistrar(t *testing.T) {
	t.Run("MessagingServiceRegistrar_Interface", func(t *testing.T) {
		registrar := &mockMessagingServiceRegistrar{}

		// Test that it implements BusinessServiceRegistrar
		var businessRegistrar BusinessServiceRegistrar = registrar
		assert.NotNil(t, businessRegistrar)

		// Test that it implements MessagingServiceRegistrar
		var messagingRegistrar MessagingServiceRegistrar = registrar
		assert.NotNil(t, messagingRegistrar)
	})

	t.Run("RegisterEventHandlers", func(t *testing.T) {
		registrar := &mockMessagingServiceRegistrar{}
		registry := newMockEventHandlerRegistry()

		err := registrar.RegisterEventHandlers(registry)
		assert.NoError(t, err)

		handlers := registry.GetRegisteredHandlers()
		assert.Contains(t, handlers, "test-handler")
	})

	t.Run("GetEventHandlerMetadata", func(t *testing.T) {
		registrar := &mockMessagingServiceRegistrar{}

		metadata := registrar.GetEventHandlerMetadata()
		assert.NotNil(t, metadata)
		assert.Equal(t, 1, metadata.HandlerCount)
		assert.Contains(t, metadata.Topics, "test-topic")
		assert.Equal(t, "Test event handler", metadata.Description)
	})

	t.Run("GetEventHandlerMetadata_Custom", func(t *testing.T) {
		customMetadata := &EventHandlerMetadata{
			HandlerCount:       3,
			Topics:             []string{"custom-topic-1", "custom-topic-2"},
			BrokerRequirements: []string{"kafka"},
			Description:        "Custom event handlers",
		}

		registrar := &mockMessagingServiceRegistrar{metadata: customMetadata}

		metadata := registrar.GetEventHandlerMetadata()
		assert.NotNil(t, metadata)
		assert.Equal(t, 3, metadata.HandlerCount)
		assert.Equal(t, 2, len(metadata.Topics))
		assert.Contains(t, metadata.Topics, "custom-topic-1")
		assert.Contains(t, metadata.Topics, "custom-topic-2")
		assert.Contains(t, metadata.BrokerRequirements, "kafka")
		assert.Equal(t, "Custom event handlers", metadata.Description)
	})
}

func TestEventHandlerMetadata(t *testing.T) {
	t.Run("EventHandlerMetadata_Creation", func(t *testing.T) {
		metadata := &EventHandlerMetadata{
			HandlerCount:       5,
			Topics:             []string{"orders", "payments", "notifications"},
			BrokerRequirements: []string{"kafka", "rabbitmq"},
			Description:        "E-commerce event handlers",
		}

		assert.Equal(t, 5, metadata.HandlerCount)
		assert.Equal(t, 3, len(metadata.Topics))
		assert.Contains(t, metadata.Topics, "orders")
		assert.Contains(t, metadata.Topics, "payments")
		assert.Contains(t, metadata.Topics, "notifications")
		assert.Equal(t, 2, len(metadata.BrokerRequirements))
		assert.Contains(t, metadata.BrokerRequirements, "kafka")
		assert.Contains(t, metadata.BrokerRequirements, "rabbitmq")
		assert.Equal(t, "E-commerce event handlers", metadata.Description)
	})

	t.Run("EventHandlerMetadata_EmptyValues", func(t *testing.T) {
		metadata := &EventHandlerMetadata{}

		assert.Equal(t, 0, metadata.HandlerCount)
		assert.Nil(t, metadata.Topics)
		assert.Nil(t, metadata.BrokerRequirements)
		assert.Equal(t, "", metadata.Description)
	})
}

func TestEventHandler_Interface(t *testing.T) {
	t.Run("EventHandler_Implementation", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID:         "test-handler",
			topics:            []string{"test-topic"},
			brokerRequirement: "kafka",
		}

		assert.Equal(t, "test-handler", handler.GetHandlerID())
		assert.Equal(t, []string{"test-topic"}, handler.GetTopics())
		assert.Equal(t, "kafka", handler.GetBrokerRequirement())

		ctx := context.Background()
		assert.NoError(t, handler.Initialize(ctx))
		assert.NoError(t, handler.Shutdown(ctx))
		assert.NoError(t, handler.Handle(ctx, "test-message"))
		assert.Nil(t, handler.OnError(ctx, "test-message", assert.AnError))
	})

	t.Run("EventHandler_EmptyBrokerRequirement", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID:         "flexible-handler",
			topics:            []string{"any-topic"},
			brokerRequirement: "",
		}

		assert.Equal(t, "", handler.GetBrokerRequirement())
	})

	t.Run("EventHandler_MultipleTopics", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "multi-topic-handler",
			topics:    []string{"topic1", "topic2", "topic3"},
		}

		topics := handler.GetTopics()
		assert.Equal(t, 3, len(topics))
		assert.Contains(t, topics, "topic1")
		assert.Contains(t, topics, "topic2")
		assert.Contains(t, topics, "topic3")
	})
}

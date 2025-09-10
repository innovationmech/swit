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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEventHandler implements EventHandler for testing
type mockEventHandler struct {
	handlerID         string
	topics            []string
	brokerRequirement string
	initializeError   error
	shutdownError     error
	handleError       error
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
	return m.initializeError
}

func (m *mockEventHandler) Shutdown(ctx context.Context) error {
	return m.shutdownError
}

func (m *mockEventHandler) Handle(ctx context.Context, message *Message) error {
	return m.handleError
}

func (m *mockEventHandler) OnError(ctx context.Context, message *Message, err error) ErrorAction {
	return ErrorActionRetry
}

// mockMessagingCoordinator implements MessagingCoordinator for testing
type mockMessagingCoordinator struct {
	handlers map[string]EventHandler
	started  bool
}

func newMockMessagingCoordinator() *mockMessagingCoordinator {
	return &mockMessagingCoordinator{
		handlers: make(map[string]EventHandler),
		started:  false,
	}
}

func (m *mockMessagingCoordinator) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *mockMessagingCoordinator) Stop(ctx context.Context) error {
	m.started = false
	return nil
}

func (m *mockMessagingCoordinator) RegisterBroker(name string, broker MessageBroker) error {
	return nil
}

func (m *mockMessagingCoordinator) GetBroker(name string) (MessageBroker, error) {
	return nil, nil
}

func (m *mockMessagingCoordinator) GetRegisteredBrokers() []string {
	return []string{}
}

func (m *mockMessagingCoordinator) RegisterEventHandler(handler EventHandler) error {
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

func (m *mockMessagingCoordinator) UnregisterEventHandler(handlerID string) error {
	if handlerID == "" {
		return assert.AnError
	}
	if _, exists := m.handlers[handlerID]; !exists {
		return assert.AnError
	}
	delete(m.handlers, handlerID)
	return nil
}

func (m *mockMessagingCoordinator) GetRegisteredHandlers() []string {
	var handlers []string
	for id := range m.handlers {
		handlers = append(handlers, id)
	}
	return handlers
}

func (m *mockMessagingCoordinator) IsStarted() bool {
	return m.started
}

func (m *mockMessagingCoordinator) GetMetrics() *MessagingCoordinatorMetrics {
	return &MessagingCoordinatorMetrics{}
}

func (m *mockMessagingCoordinator) HealthCheck(ctx context.Context) (*MessagingHealthStatus, error) {
	return &MessagingHealthStatus{}, nil
}

func TestNewEventHandlerRegistry(t *testing.T) {
	coordinator := newMockMessagingCoordinator()
	registry := NewEventHandlerRegistry(coordinator)

	assert.NotNil(t, registry)
	assert.Equal(t, coordinator, registry.GetCoordinator())
}

func TestEventHandlerRegistry_RegisterEventHandler(t *testing.T) {
	coordinator := newMockMessagingCoordinator()
	registry := NewEventHandlerRegistry(coordinator)

	t.Run("ValidHandler", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "test-handler",
			topics:    []string{"test-topic"},
		}

		err := registry.RegisterEventHandler(handler)
		assert.NoError(t, err)

		handlers := registry.GetRegisteredHandlers()
		assert.Contains(t, handlers, "test-handler")
	})

	t.Run("NilHandler", func(t *testing.T) {
		err := registry.RegisterEventHandler(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("EmptyHandlerID", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "",
			topics:    []string{"test-topic"},
		}

		err := registry.RegisterEventHandler(handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-empty ID")
	})

	t.Run("EmptyTopics", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "test-handler-no-topics",
			topics:    []string{},
		}

		err := registry.RegisterEventHandler(handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one topic")
	})
}

func TestEventHandlerRegistry_UnregisterEventHandler(t *testing.T) {
	coordinator := newMockMessagingCoordinator()
	registry := NewEventHandlerRegistry(coordinator)

	// Register a handler first
	handler := &mockEventHandler{
		handlerID: "handler-to-remove",
		topics:    []string{"test-topic"},
	}
	err := registry.RegisterEventHandler(handler)
	require.NoError(t, err)

	t.Run("ValidUnregister", func(t *testing.T) {
		err := registry.UnregisterEventHandler("handler-to-remove")
		assert.NoError(t, err)

		handlers := registry.GetRegisteredHandlers()
		assert.NotContains(t, handlers, "handler-to-remove")
	})

	t.Run("EmptyHandlerID", func(t *testing.T) {
		err := registry.UnregisterEventHandler("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("NonExistentHandler", func(t *testing.T) {
		err := registry.UnregisterEventHandler("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unregister")
	})
}

func TestEventHandlerRegistry_ValidateHandler(t *testing.T) {
	coordinator := newMockMessagingCoordinator()
	registry := NewEventHandlerRegistry(coordinator)

	t.Run("ValidHandler", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "valid-handler-123",
			topics:    []string{"valid-topic"},
		}

		err := registry.ValidateHandler(handler)
		assert.NoError(t, err)
	})

	t.Run("NilHandler", func(t *testing.T) {
		err := registry.ValidateHandler(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("EmptyHandlerID", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "",
			topics:    []string{"test-topic"},
		}

		err := registry.ValidateHandler(handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-empty ID")
	})

	t.Run("InvalidHandlerIDCharacters", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "invalid@handler!",
			topics:    []string{"test-topic"},
		}

		err := registry.ValidateHandler(handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "letters, numbers, hyphens, and underscores")
	})

	t.Run("EmptyTopics", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "handler-no-topics",
			topics:    []string{},
		}

		err := registry.ValidateHandler(handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one topic")
	})

	t.Run("EmptyTopicName", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "handler-empty-topic",
			topics:    []string{"valid-topic", ""},
		}

		err := registry.ValidateHandler(handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("DuplicateHandlerID", func(t *testing.T) {
		// Register a handler first
		handler1 := &mockEventHandler{
			handlerID: "duplicate-handler",
			topics:    []string{"topic1"},
		}
		err := registry.RegisterEventHandler(handler1)
		require.NoError(t, err)

		// Try to validate another handler with the same ID
		handler2 := &mockEventHandler{
			handlerID: "duplicate-handler",
			topics:    []string{"topic2"},
		}

		err = registry.ValidateHandler(handler2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})
}

func TestEventHandlerRegistry_RegisterEventHandlerWithValidation(t *testing.T) {
	coordinator := newMockMessagingCoordinator()
	registry := NewEventHandlerRegistry(coordinator)

	t.Run("ValidHandlerRegistration", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "validated-handler",
			topics:    []string{"test-topic"},
		}

		err := registry.RegisterEventHandlerWithValidation(handler)
		assert.NoError(t, err)

		handlers := registry.GetRegisteredHandlers()
		assert.Contains(t, handlers, "validated-handler")
	})

	t.Run("InvalidHandlerValidation", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "",
			topics:    []string{"test-topic"},
		}

		err := registry.RegisterEventHandlerWithValidation(handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})
}

func TestEventHandlerRegistry_GetHandlerCount(t *testing.T) {
	coordinator := newMockMessagingCoordinator()
	registry := NewEventHandlerRegistry(coordinator)

	assert.Equal(t, 0, registry.GetHandlerCount())

	handler1 := &mockEventHandler{
		handlerID: "handler1",
		topics:    []string{"topic1"},
	}
	err := registry.RegisterEventHandler(handler1)
	require.NoError(t, err)

	assert.Equal(t, 1, registry.GetHandlerCount())

	handler2 := &mockEventHandler{
		handlerID: "handler2",
		topics:    []string{"topic2"},
	}
	err = registry.RegisterEventHandler(handler2)
	require.NoError(t, err)

	assert.Equal(t, 2, registry.GetHandlerCount())
}

func TestEventHandlerRegistry_Clear(t *testing.T) {
	coordinator := newMockMessagingCoordinator()
	registry := NewEventHandlerRegistry(coordinator)

	// Register multiple handlers
	handler1 := &mockEventHandler{
		handlerID: "handler1",
		topics:    []string{"topic1"},
	}
	handler2 := &mockEventHandler{
		handlerID: "handler2",
		topics:    []string{"topic2"},
	}

	err := registry.RegisterEventHandler(handler1)
	require.NoError(t, err)
	err = registry.RegisterEventHandler(handler2)
	require.NoError(t, err)

	assert.Equal(t, 2, registry.GetHandlerCount())

	// Clear all handlers
	err = registry.Clear()
	assert.NoError(t, err)

	assert.Equal(t, 0, registry.GetHandlerCount())
	handlers := registry.GetRegisteredHandlers()
	assert.Empty(t, handlers)
}

func TestEventHandlerRegistry_ThreadSafety(t *testing.T) {
	coordinator := newMockMessagingCoordinator()
	registry := NewEventHandlerRegistry(coordinator)

	// This test ensures the registry doesn't panic under concurrent access
	// We don't test for race conditions as that would require more complex testing
	t.Run("ConcurrentAccess", func(t *testing.T) {
		handler := &mockEventHandler{
			handlerID: "concurrent-handler",
			topics:    []string{"concurrent-topic"},
		}

		// Register handler
		err := registry.RegisterEventHandler(handler)
		require.NoError(t, err)

		// Access methods concurrently (basic smoke test)
		go func() {
			registry.GetRegisteredHandlers()
		}()

		go func() {
			registry.GetHandlerCount()
		}()

		go func() {
			registry.GetCoordinator()
		}()

		// No assertions needed - test passes if no panic occurs
	})
}

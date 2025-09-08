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

package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// MockHandler is a test implementation of MessageHandler
type MockHandler struct {
	handleFunc  func(ctx context.Context, message *Message) error
	onErrorFunc func(ctx context.Context, message *Message, err error) ErrorAction
	callCount   int
	lastMessage *Message
	mutex       sync.RWMutex
}

func NewMockHandler() *MockHandler {
	return &MockHandler{
		handleFunc: func(ctx context.Context, message *Message) error {
			return nil
		},
		onErrorFunc: func(ctx context.Context, message *Message, err error) ErrorAction {
			return ErrorActionRetry
		},
	}
}

func (m *MockHandler) Handle(ctx context.Context, message *Message) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.callCount++
	m.lastMessage = message

	if m.handleFunc != nil {
		return m.handleFunc(ctx, message)
	}
	return nil
}

func (m *MockHandler) OnError(ctx context.Context, message *Message, err error) ErrorAction {
	if m.onErrorFunc != nil {
		return m.onErrorFunc(ctx, message, err)
	}
	return ErrorActionRetry
}

func (m *MockHandler) GetCallCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.callCount
}

func (m *MockHandler) GetLastMessage() *Message {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.lastMessage
}

func TestSubscriberState_String(t *testing.T) {
	tests := []struct {
		state    SubscriberState
		expected string
	}{
		{SubscriberStateUnknown, "unknown"},
		{SubscriberStateConnecting, "connecting"},
		{SubscriberStateSubscribed, "subscribed"},
		{SubscriberStatePaused, "paused"},
		{SubscriberStateUnsubscribing, "unsubscribing"},
		{SubscriberStateClosed, "closed"},
		{SubscriberState(999), "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("SubscriberState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewBaseSubscriber(t *testing.T) {
	config := SubscriberConfig{
		Topics:        []string{"test-topic"},
		ConsumerGroup: "test-group",
	}

	subscriber := NewBaseSubscriber(config)

	if subscriber == nil {
		t.Fatal("NewBaseSubscriber() returned nil")
	}

	if subscriber.GetState() != SubscriberStateUnknown {
		t.Errorf("Expected initial state to be Unknown, got %v", subscriber.GetState())
	}

	if subscriber.handlers == nil {
		t.Error("handlers map should be initialized")
	}

	if subscriber.metrics == nil {
		t.Error("metrics should be initialized")
	}

	if subscriber.stateMetrics == nil {
		t.Error("stateMetrics should be initialized")
	}

	if subscriber.globalMiddleware == nil {
		t.Error("globalMiddleware should be initialized")
	}
}

func TestBaseSubscriber_RegisterHandler(t *testing.T) {
	subscriber := NewBaseSubscriber(SubscriberConfig{})
	handler := NewMockHandler()

	// Test successful registration
	err := subscriber.RegisterHandler("test-topic", handler)
	if err != nil {
		t.Fatalf("RegisterHandler() failed: %v", err)
	}

	// Verify handler was registered
	registrations := subscriber.GetRegisteredHandlers("test-topic")
	if len(registrations) != 1 {
		t.Errorf("Expected 1 registered handler, got %d", len(registrations))
	}

	if registrations[0].Handler != handler {
		t.Error("Registered handler doesn't match original")
	}

	if registrations[0].Topic != "test-topic" {
		t.Errorf("Expected topic 'test-topic', got %s", registrations[0].Topic)
	}

	// Test error cases
	err = subscriber.RegisterHandler("", handler)
	if err == nil {
		t.Error("Expected error for empty topic")
	}

	err = subscriber.RegisterHandler("test-topic", nil)
	if err == nil {
		t.Error("Expected error for nil handler")
	}
}

func TestBaseSubscriber_RegisterMultipleHandlers(t *testing.T) {
	subscriber := NewBaseSubscriber(SubscriberConfig{})
	handler1 := NewMockHandler()
	handler2 := NewMockHandler()

	// Register multiple handlers for same topic
	err := subscriber.RegisterHandler("test-topic", handler1)
	if err != nil {
		t.Fatalf("RegisterHandler() failed: %v", err)
	}

	err = subscriber.RegisterHandler("test-topic", handler2)
	if err != nil {
		t.Fatalf("RegisterHandler() failed: %v", err)
	}

	// Verify both handlers are registered
	registrations := subscriber.GetRegisteredHandlers("test-topic")
	if len(registrations) != 2 {
		t.Errorf("Expected 2 registered handlers, got %d", len(registrations))
	}
}

func TestBaseSubscriber_UnregisterHandler(t *testing.T) {
	subscriber := NewBaseSubscriber(SubscriberConfig{})
	handler := NewMockHandler()

	// Register handler first
	err := subscriber.RegisterHandler("test-topic", handler)
	if err != nil {
		t.Fatalf("RegisterHandler() failed: %v", err)
	}

	registrations := subscriber.GetRegisteredHandlers("test-topic")
	if len(registrations) != 1 {
		t.Fatalf("Expected 1 registered handler, got %d", len(registrations))
	}

	handlerID := registrations[0].HandlerID

	// Test successful unregistration
	err = subscriber.UnregisterHandler("test-topic", handlerID)
	if err != nil {
		t.Fatalf("UnregisterHandler() failed: %v", err)
	}

	// Verify handler was unregistered
	registrations = subscriber.GetRegisteredHandlers("test-topic")
	if len(registrations) != 0 {
		t.Errorf("Expected 0 registered handlers after unregistration, got %d", len(registrations))
	}

	// Test error cases
	err = subscriber.UnregisterHandler("non-existent-topic", "some-id")
	if err == nil {
		t.Error("Expected error for non-existent topic")
	}

	err = subscriber.UnregisterHandler("test-topic", "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent handler ID")
	}
}

func TestBaseSubscriber_AddGlobalMiddleware(t *testing.T) {
	subscriber := NewBaseSubscriber(SubscriberConfig{})

	// Create mock middleware
	middleware1 := &MockMiddleware{name: "middleware1"}
	middleware2 := &MockMiddleware{name: "middleware2"}

	// Add global middleware
	subscriber.AddGlobalMiddleware(middleware1, middleware2)

	// Register a handler and verify middleware is applied
	handler := NewMockHandler()
	err := subscriber.RegisterHandler("test-topic", handler)
	if err != nil {
		t.Fatalf("RegisterHandler() failed: %v", err)
	}

	registrations := subscriber.GetRegisteredHandlers("test-topic")
	if len(registrations) != 1 {
		t.Fatalf("Expected 1 registered handler, got %d", len(registrations))
	}

	// Verify global middleware was prepended to handler middleware
	if len(registrations[0].Middleware) < 2 {
		t.Errorf("Expected at least 2 middleware, got %d", len(registrations[0].Middleware))
	}

	if registrations[0].Middleware[0].Name() != "middleware1" {
		t.Errorf("Expected first middleware to be 'middleware1', got %s", registrations[0].Middleware[0].Name())
	}

	if registrations[0].Middleware[1].Name() != "middleware2" {
		t.Errorf("Expected second middleware to be 'middleware2', got %s", registrations[0].Middleware[1].Name())
	}
}

func TestBaseSubscriber_GetState(t *testing.T) {
	subscriber := NewBaseSubscriber(SubscriberConfig{})

	// Test initial state
	if state := subscriber.GetState(); state != SubscriberStateUnknown {
		t.Errorf("Expected initial state to be Unknown, got %v", state)
	}

	// Test state changes
	subscriber.setState(SubscriberStateConnecting)
	if state := subscriber.GetState(); state != SubscriberStateConnecting {
		t.Errorf("Expected state to be Connecting, got %v", state)
	}

	subscriber.setState(SubscriberStateSubscribed)
	if state := subscriber.GetState(); state != SubscriberStateSubscribed {
		t.Errorf("Expected state to be Subscribed, got %v", state)
	}
}

func TestBaseSubscriber_GetMetrics(t *testing.T) {
	subscriber := NewBaseSubscriber(SubscriberConfig{})

	metrics := subscriber.GetMetrics()
	if metrics == nil {
		t.Fatal("GetMetrics() returned nil")
	}

	// Test that metrics reflect current state
	subscriber.setState(SubscriberStateSubscribed)
	metrics = subscriber.GetMetrics()

	// Verify that metrics are returned (exact values depend on the existing SubscriberMetrics structure)
	if metrics.MessagesConsumed < 0 {
		t.Error("MessagesConsumed should not be negative")
	}
}

func TestBaseSubscriber_ProcessMessage(t *testing.T) {
	subscriber := NewBaseSubscriber(SubscriberConfig{})
	handler := NewMockHandler()

	// Register handler
	err := subscriber.RegisterHandler("test-topic", handler)
	if err != nil {
		t.Fatalf("RegisterHandler() failed: %v", err)
	}

	registrations := subscriber.GetRegisteredHandlers("test-topic")
	if len(registrations) != 1 {
		t.Fatalf("Expected 1 registered handler, got %d", len(registrations))
	}

	// Create test message
	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	// Process message
	ctx := context.Background()
	err = subscriber.processMessage(ctx, message, registrations[0])
	if err != nil {
		t.Errorf("processMessage() failed: %v", err)
	}

	// Verify handler was called
	if handler.GetCallCount() != 1 {
		t.Errorf("Expected handler to be called once, got %d calls", handler.GetCallCount())
	}

	if handler.GetLastMessage().ID != "test-id" {
		t.Errorf("Expected message ID 'test-id', got %s", handler.GetLastMessage().ID)
	}
}

func TestBaseSubscriber_ProcessMessageWithError(t *testing.T) {
	subscriber := NewBaseSubscriber(SubscriberConfig{})
	handler := NewMockHandler()

	// Configure handler to return an error
	testError := errors.New("test error")
	handler.handleFunc = func(ctx context.Context, message *Message) error {
		return testError
	}

	// Register handler
	err := subscriber.RegisterHandler("test-topic", handler)
	if err != nil {
		t.Fatalf("RegisterHandler() failed: %v", err)
	}

	registrations := subscriber.GetRegisteredHandlers("test-topic")
	if len(registrations) != 1 {
		t.Fatalf("Expected 1 registered handler, got %d", len(registrations))
	}

	// Create test message
	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	// Process message
	ctx := context.Background()
	err = subscriber.processMessage(ctx, message, registrations[0])
	if err == nil {
		t.Error("Expected processMessage() to return error")
	}

	if err != testError {
		t.Errorf("Expected error to be %v, got %v", testError, err)
	}
}

func TestBaseSubscriber_BuildHandlerChain(t *testing.T) {
	subscriber := NewBaseSubscriber(SubscriberConfig{})
	baseHandler := NewMockHandler()

	// Create mock middleware
	middleware1 := &MockMiddleware{name: "middleware1"}
	middleware2 := &MockMiddleware{name: "middleware2"}

	// Build handler chain
	chain := subscriber.buildHandlerChain(baseHandler, []Middleware{middleware1, middleware2})

	if chain == nil {
		t.Fatal("buildHandlerChain() returned nil")
	}

	// Test that the chain works
	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	err := chain.Handle(context.Background(), message)
	if err != nil {
		t.Errorf("Handler chain failed: %v", err)
	}

	// Verify middleware was called
	if !middleware1.called {
		t.Error("Middleware1 was not called")
	}

	if !middleware2.called {
		t.Error("Middleware2 was not called")
	}

	// Verify base handler was called
	if baseHandler.GetCallCount() != 1 {
		t.Errorf("Expected base handler to be called once, got %d calls", baseHandler.GetCallCount())
	}
}

func TestBaseSubscriber_Close(t *testing.T) {
	subscriber := NewBaseSubscriber(SubscriberConfig{})

	err := subscriber.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	if subscriber.GetState() != SubscriberStateClosed {
		t.Errorf("Expected state to be Closed after Close(), got %v", subscriber.GetState())
	}
}

// Mock middleware for testing
type MockMiddleware struct {
	name   string
	called bool
}

func (m *MockMiddleware) Name() string {
	return m.name
}

func (m *MockMiddleware) Wrap(next MessageHandler) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		m.called = true
		return next.Handle(ctx, message)
	})
}

// Benchmark tests
func BenchmarkBaseSubscriber_RegisterHandler(b *testing.B) {
	subscriber := NewBaseSubscriber(SubscriberConfig{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler := NewMockHandler()
		topic := "test-topic"
		err := subscriber.RegisterHandler(topic, handler)
		if err != nil {
			b.Fatalf("RegisterHandler() failed: %v", err)
		}
	}
}

func BenchmarkBaseSubscriber_ProcessMessage(b *testing.B) {
	subscriber := NewBaseSubscriber(SubscriberConfig{})
	handler := NewMockHandler()

	err := subscriber.RegisterHandler("test-topic", handler)
	if err != nil {
		b.Fatalf("RegisterHandler() failed: %v", err)
	}

	registrations := subscriber.GetRegisteredHandlers("test-topic")
	if len(registrations) != 1 {
		b.Fatalf("Expected 1 registered handler, got %d", len(registrations))
	}

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := subscriber.processMessage(ctx, message, registrations[0])
		if err != nil {
			b.Fatalf("processMessage() failed: %v", err)
		}
	}
}

func TestSubscriberMessageHandlerFunc(t *testing.T) {
	callCount := 0
	handlerFunc := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		callCount++
		return nil
	})

	message := &Message{
		ID:      "test-id",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	err := handlerFunc.Handle(context.Background(), message)
	if err != nil {
		t.Errorf("MessageHandlerFunc.Handle() failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d calls", callCount)
	}

	// Test OnError method
	testError := errors.New("test error")
	action := handlerFunc.OnError(context.Background(), message, testError)
	if action != ErrorActionRetry {
		t.Errorf("Expected ErrorActionRetry, got %v", action)
	}

	// Test OnError with non-retryable error
	nonRetryableError := &MessagingError{
		Code:      ErrProcessingFailed,
		Message:   "test error",
		Retryable: false,
	}

	action = handlerFunc.OnError(context.Background(), message, nonRetryableError)
	if action != ErrorActionDeadLetter {
		t.Errorf("Expected ErrorActionDeadLetter for non-retryable error, got %v", action)
	}
}

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

// Package testutil provides comprehensive test utilities and mock implementations
// for the messaging system, enabling effective unit testing and integration testing
// of messaging components within the SWIT framework.
package testutil

import (
	"context"
	"fmt"
	"sync"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/stretchr/testify/mock"
)

// MockMessageBroker provides a comprehensive mock implementation of the MessageBroker interface
// for testing messaging system components.
type MockMessageBroker struct {
	mock.Mock
	connected    bool
	mu           sync.RWMutex
	publishers   []messaging.EventPublisher
	subscribers  []messaging.EventSubscriber
	metrics      *messaging.BrokerMetrics
	capabilities *messaging.BrokerCapabilities
}

// NewMockMessageBroker creates a new mock message broker instance.
func NewMockMessageBroker() *MockMessageBroker {
	return &MockMessageBroker{
		connected:   false,
		publishers:  make([]messaging.EventPublisher, 0),
		subscribers: make([]messaging.EventSubscriber, 0),
		metrics: &messaging.BrokerMetrics{
			ConnectionStatus:   "connected",
			ActivePublishers:   0,
			ActiveSubscribers:  0,
		},
		capabilities: &messaging.BrokerCapabilities{
			SupportsTransactions:    false,
			SupportsOrdering:        true,
			SupportsPartitioning:    true,
			SupportsDeadLetter:      true,
			SupportsDelayedDelivery: false,
		},
	}
}

// Connect mocks the broker connection.
func (m *MockMessageBroker) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	if args.Error(0) == nil {
		m.mu.Lock()
		m.connected = true
		m.mu.Unlock()
	}
	return args.Error(0)
}

// Disconnect mocks the broker disconnection.
func (m *MockMessageBroker) Disconnect(ctx context.Context) error {
	args := m.Called(ctx)
	if args.Error(0) == nil {
		m.mu.Lock()
		m.connected = false
		m.mu.Unlock()
	}
	return args.Error(0)
}

// Close mocks the broker close operation.
func (m *MockMessageBroker) Close() error {
	args := m.Called()
	if args.Error(0) == nil {
		m.mu.Lock()
		m.connected = false
		m.mu.Unlock()
	}
	return args.Error(0)
}

// IsConnected returns the mock connection status.
func (m *MockMessageBroker) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// CreatePublisher creates a mock publisher.
func (m *MockMessageBroker) CreatePublisher(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
	args := m.Called(config)
	if publisher := args.Get(0); publisher != nil {
		mockPublisher := publisher.(messaging.EventPublisher)
		m.mu.Lock()
		m.publishers = append(m.publishers, mockPublisher)
		m.mu.Unlock()
		return mockPublisher, args.Error(1)
	}
	return nil, args.Error(1)
}

// CreateSubscriber creates a mock subscriber.
func (m *MockMessageBroker) CreateSubscriber(config messaging.SubscriberConfig) (messaging.EventSubscriber, error) {
	args := m.Called(config)
	if subscriber := args.Get(0); subscriber != nil {
		mockSubscriber := subscriber.(messaging.EventSubscriber)
		m.mu.Lock()
		m.subscribers = append(m.subscribers, mockSubscriber)
		m.mu.Unlock()
		return mockSubscriber, args.Error(1)
	}
	return nil, args.Error(1)
}

// HealthCheck mocks the health check operation.
func (m *MockMessageBroker) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	args := m.Called(ctx)
	if status := args.Get(0); status != nil {
		return status.(*messaging.HealthStatus), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetMetrics returns mock broker metrics.
func (m *MockMessageBroker) GetMetrics() *messaging.BrokerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
}

// GetCapabilities returns mock broker capabilities.
func (m *MockMessageBroker) GetCapabilities() *messaging.BrokerCapabilities {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.capabilities
}

// SetConnected sets the connection status for testing.
func (m *MockMessageBroker) SetConnected(connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = connected
}

// GetPublishers returns all created publishers for testing verification.
func (m *MockMessageBroker) GetPublishers() []messaging.EventPublisher {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]messaging.EventPublisher, len(m.publishers))
	copy(result, m.publishers)
	return result
}

// GetSubscribers returns all created subscribers for testing verification.
func (m *MockMessageBroker) GetSubscribers() []messaging.EventSubscriber {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]messaging.EventSubscriber, len(m.subscribers))
	copy(result, m.subscribers)
	return result
}

// MockEventPublisher provides a mock implementation of the EventPublisher interface.
type MockEventPublisher struct {
	mock.Mock
	publishedMessages []messaging.Message
	mu                sync.RWMutex
}

// NewMockEventPublisher creates a new mock event publisher.
func NewMockEventPublisher() *MockEventPublisher {
	return &MockEventPublisher{
		publishedMessages: make([]messaging.Message, 0),
	}
}

// Publish mocks message publishing.
func (m *MockEventPublisher) Publish(ctx context.Context, message messaging.Message) error {
	args := m.Called(ctx, message)
	if args.Error(0) == nil {
		m.mu.Lock()
		m.publishedMessages = append(m.publishedMessages, message)
		m.mu.Unlock()
	}
	return args.Error(0)
}

// PublishBatch mocks batch message publishing.
func (m *MockEventPublisher) PublishBatch(ctx context.Context, messages []messaging.Message) error {
	args := m.Called(ctx, messages)
	if args.Error(0) == nil {
		m.mu.Lock()
		m.publishedMessages = append(m.publishedMessages, messages...)
		m.mu.Unlock()
	}
	return args.Error(0)
}

// PublishAsync mocks asynchronous message publishing.
func (m *MockEventPublisher) PublishAsync(ctx context.Context, message messaging.Message) (<-chan error, error) {
	args := m.Called(ctx, message)
	if args.Error(1) == nil {
		m.mu.Lock()
		m.publishedMessages = append(m.publishedMessages, message)
		m.mu.Unlock()
	}

	if ch := args.Get(0); ch != nil {
		return ch.(<-chan error), args.Error(1)
	}
	return nil, args.Error(1)
}

// Close mocks publisher close operation.
func (m *MockEventPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

// GetPublishedMessages returns all messages published through this mock.
func (m *MockEventPublisher) GetPublishedMessages() []messaging.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]messaging.Message, len(m.publishedMessages))
	copy(result, m.publishedMessages)
	return result
}

// ClearPublishedMessages clears the published messages list for testing.
func (m *MockEventPublisher) ClearPublishedMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishedMessages = m.publishedMessages[:0]
}

// MockEventSubscriber provides a mock implementation of the EventSubscriber interface.
type MockEventSubscriber struct {
	mock.Mock
	handlers     []messaging.MessageHandler
	mu           sync.RWMutex
	subscribing  bool
	messageCount int64
}

// NewMockEventSubscriber creates a new mock event subscriber.
func NewMockEventSubscriber() *MockEventSubscriber {
	return &MockEventSubscriber{
		handlers:    make([]messaging.MessageHandler, 0),
		subscribing: false,
	}
}

// Subscribe mocks subscription to messages.
func (m *MockEventSubscriber) Subscribe(ctx context.Context, handler messaging.MessageHandler) error {
	args := m.Called(ctx, handler)
	if args.Error(0) == nil {
		m.mu.Lock()
		m.handlers = append(m.handlers, handler)
		m.subscribing = true
		m.mu.Unlock()
	}
	return args.Error(0)
}

// Unsubscribe mocks unsubscription from messages.
func (m *MockEventSubscriber) Unsubscribe(ctx context.Context) error {
	args := m.Called(ctx)
	if args.Error(0) == nil {
		m.mu.Lock()
		m.subscribing = false
		m.mu.Unlock()
	}
	return args.Error(0)
}

// Close mocks subscriber close operation.
func (m *MockEventSubscriber) Close() error {
	args := m.Called()
	if args.Error(0) == nil {
		m.mu.Lock()
		m.subscribing = false
		m.mu.Unlock()
	}
	return args.Error(0)
}

// IsSubscribed returns the mock subscription status.
func (m *MockEventSubscriber) IsSubscribed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.subscribing
}

// SimulateMessage simulates receiving a message for testing.
func (m *MockEventSubscriber) SimulateMessage(ctx context.Context, message messaging.Message) error {
	m.mu.RLock()
	handlers := make([]messaging.MessageHandler, len(m.handlers))
	copy(handlers, m.handlers)
	m.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler.Handle(ctx, &message); err != nil {
			return fmt.Errorf("handler failed: %w", err)
		}
	}

	m.mu.Lock()
	m.messageCount++
	m.mu.Unlock()

	return nil
}

// GetHandlers returns all registered message handlers.
func (m *MockEventSubscriber) GetHandlers() []messaging.MessageHandler {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]messaging.MessageHandler, len(m.handlers))
	copy(result, m.handlers)
	return result
}

// GetMessageCount returns the number of messages processed.
func (m *MockEventSubscriber) GetMessageCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.messageCount
}

// MockMessageHandler provides a mock implementation of the MessageHandler interface.
type MockMessageHandler struct {
	mock.Mock
	handledMessages []messaging.Message
	mu              sync.RWMutex
}

// NewMockMessageHandler creates a new mock message handler.
func NewMockMessageHandler() *MockMessageHandler {
	return &MockMessageHandler{
		handledMessages: make([]messaging.Message, 0),
	}
}

// Handle mocks message handling.
func (m *MockMessageHandler) Handle(ctx context.Context, message *messaging.Message) error {
	args := m.Called(ctx, message)
	if args.Error(0) == nil {
		m.mu.Lock()
		m.handledMessages = append(m.handledMessages, *message)
		m.mu.Unlock()
	}
	return args.Error(0)
}

// OnError mocks error handling.
func (m *MockMessageHandler) OnError(ctx context.Context, message *messaging.Message, err error) messaging.ErrorAction {
	args := m.Called(ctx, message, err)
	if action := args.Get(0); action != nil {
		return action.(messaging.ErrorAction)
	}
	return messaging.ErrorActionRetry
}

// GetHandledMessages returns all messages handled by this mock.
func (m *MockMessageHandler) GetHandledMessages() []messaging.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]messaging.Message, len(m.handledMessages))
	copy(result, m.handledMessages)
	return result
}

// ClearHandledMessages clears the handled messages list for testing.
func (m *MockMessageHandler) ClearHandledMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handledMessages = m.handledMessages[:0]
}

// MockMiddleware provides a mock implementation of middleware for testing.
type MockMiddleware struct {
	mock.Mock
	name string
}

// NewMockMiddleware creates a new mock middleware.
func NewMockMiddleware() *MockMiddleware {
	return &MockMiddleware{
		name: "mock-middleware",
	}
}

// Name returns the middleware name.
func (m *MockMiddleware) Name() string {
	args := m.Called()
	if name := args.String(0); name != "" {
		return name
	}
	return m.name
}

// Wrap wraps a handler with middleware logic.
func (m *MockMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
	args := m.Called(next)
	if handler := args.Get(0); handler != nil {
		return handler.(messaging.MessageHandler)
	}
	return next
}

// SetName sets the middleware name for testing.
func (m *MockMiddleware) SetName(name string) {
	m.name = name
}

// MockTransaction provides a mock implementation of the Transaction interface.
type MockTransaction struct {
	mock.Mock
	committed bool
	aborted   bool
	mu        sync.RWMutex
}

// NewMockTransaction creates a new mock transaction.
func NewMockTransaction() *MockTransaction {
	return &MockTransaction{}
}

// Publish mocks transaction publish.
func (m *MockTransaction) Publish(ctx context.Context, message *messaging.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

// Commit mocks transaction commit.
func (m *MockTransaction) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	if args.Error(0) == nil {
		m.mu.Lock()
		m.committed = true
		m.mu.Unlock()
	}
	return args.Error(0)
}

// Rollback mocks transaction rollback.
func (m *MockTransaction) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	if args.Error(0) == nil {
		m.mu.Lock()
		m.aborted = true
		m.mu.Unlock()
	}
	return args.Error(0)
}

// IsCommitted returns whether the transaction was committed.
func (m *MockTransaction) IsCommitted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.committed
}

// IsAborted returns whether the transaction was aborted.
func (m *MockTransaction) IsAborted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.aborted
}

// MockBrokerFactory provides a mock implementation of the MessageBrokerFactory interface.
type MockBrokerFactory struct {
	mock.Mock
	createdBrokers []messaging.MessageBroker
	mu             sync.RWMutex
}

// NewMockBrokerFactory creates a new mock broker factory.
func NewMockBrokerFactory() *MockBrokerFactory {
	return &MockBrokerFactory{
		createdBrokers: make([]messaging.MessageBroker, 0),
	}
}

// CreateBroker mocks broker creation.
func (m *MockBrokerFactory) CreateBroker(config *messaging.BrokerConfig) (messaging.MessageBroker, error) {
	args := m.Called(config)
	if broker := args.Get(0); broker != nil {
		mockBroker := broker.(messaging.MessageBroker)
		m.mu.Lock()
		m.createdBrokers = append(m.createdBrokers, mockBroker)
		m.mu.Unlock()
		return mockBroker, args.Error(1)
	}
	return nil, args.Error(1)
}

// GetSupportedBrokerTypes mocks supported broker types.
func (m *MockBrokerFactory) GetSupportedBrokerTypes() []messaging.BrokerType {
	args := m.Called()
	if types := args.Get(0); types != nil {
		return types.([]messaging.BrokerType)
	}
	return nil
}

// ValidateConfig mocks configuration validation.
func (m *MockBrokerFactory) ValidateConfig(config *messaging.BrokerConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

// GetCreatedBrokers returns all brokers created by this factory.
func (m *MockBrokerFactory) GetCreatedBrokers() []messaging.MessageBroker {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]messaging.MessageBroker, len(m.createdBrokers))
	copy(result, m.createdBrokers)
	return result
}

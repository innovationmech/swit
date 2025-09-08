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
	"testing"
	"time"
)

// TestMessageHandlerFunc tests the MessageHandlerFunc adapter.
func TestMessageHandlerFunc(t *testing.T) {
	called := false
	handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		called = true
		return nil
	})

	msg := &Message{
		ID:      "test-123",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	err := handler.Handle(context.Background(), msg)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !called {
		t.Error("Expected handler function to be called")
	}
}

// TestMessageHandlerFuncOnError tests the default error handling for MessageHandlerFunc.
func TestMessageHandlerFuncOnError(t *testing.T) {
	handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		return nil
	})

	msg := &Message{
		ID:      "test-123",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	// Test with retryable error
	retryableErr := &MessagingError{
		Code:      ErrConnectionFailed,
		Message:   "connection failed",
		Retryable: true,
		Timestamp: time.Now(),
	}

	action := handler.OnError(context.Background(), msg, retryableErr)
	if action != ErrorActionRetry {
		t.Errorf("Expected ErrorActionRetry for retryable error, got: %v", action)
	}

	// Test with non-retryable error
	nonRetryableErr := &MessagingError{
		Code:      ErrProcessingFailed,
		Message:   "processing failed",
		Retryable: false,
		Timestamp: time.Now(),
	}

	action = handler.OnError(context.Background(), msg, nonRetryableErr)
	if action != ErrorActionDeadLetter {
		t.Errorf("Expected ErrorActionDeadLetter for non-retryable error, got: %v", action)
	}

	// Test with generic error
	genericErr := errors.New("generic error")
	action = handler.OnError(context.Background(), msg, genericErr)
	if action != ErrorActionRetry {
		t.Errorf("Expected ErrorActionRetry for generic error, got: %v", action)
	}
}

// mockMessageBroker is a mock implementation for testing.
type mockMessageBroker struct {
	connected        bool
	capabilities     *BrokerCapabilities
	metrics          *BrokerMetrics
	connectError     error
	disconnectError  error
	publisherError   error
	subscriberError  error
	healthCheckError error
}

func (m *mockMessageBroker) Connect(ctx context.Context) error {
	if m.connectError != nil {
		return m.connectError
	}
	m.connected = true
	return nil
}

func (m *mockMessageBroker) Disconnect(ctx context.Context) error {
	if m.disconnectError != nil {
		return m.disconnectError
	}
	m.connected = false
	return nil
}

func (m *mockMessageBroker) Close() error {
	m.connected = false
	return nil
}

func (m *mockMessageBroker) IsConnected() bool {
	return m.connected
}

func (m *mockMessageBroker) CreatePublisher(config PublisherConfig) (EventPublisher, error) {
	if m.publisherError != nil {
		return nil, m.publisherError
	}
	return &mockEventPublisher{}, nil
}

func (m *mockMessageBroker) CreateSubscriber(config SubscriberConfig) (EventSubscriber, error) {
	if m.subscriberError != nil {
		return nil, m.subscriberError
	}
	return &mockEventSubscriber{}, nil
}

func (m *mockMessageBroker) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if m.healthCheckError != nil {
		return nil, m.healthCheckError
	}
	return &HealthStatus{
		Status:       HealthStatusHealthy,
		Message:      "Broker is healthy",
		LastChecked:  time.Now(),
		ResponseTime: 10 * time.Millisecond,
	}, nil
}

func (m *mockMessageBroker) GetMetrics() *BrokerMetrics {
	if m.metrics == nil {
		return &BrokerMetrics{
			ConnectionStatus:   "connected",
			ConnectionUptime:   time.Hour,
			PublishersCreated:  1,
			ActivePublishers:   1,
			SubscribersCreated: 1,
			ActiveSubscribers:  1,
			MessagesPublished:  100,
			MessagesConsumed:   90,
			LastMetricsUpdate:  time.Now(),
		}
	}
	return m.metrics
}

func (m *mockMessageBroker) GetCapabilities() *BrokerCapabilities {
	if m.capabilities == nil {
		return &BrokerCapabilities{
			SupportsTransactions:        true,
			SupportsOrdering:            true,
			SupportsPartitioning:        true,
			SupportsDeadLetter:          true,
			SupportsDelayedDelivery:     false,
			SupportsPriority:            false,
			SupportsStreaming:           true,
			SupportsSeek:                true,
			SupportsConsumerGroups:      true,
			MaxMessageSize:              1024 * 1024,
			MaxBatchSize:                1000,
			MaxTopicNameLength:          255,
			SupportedCompressionTypes:   []CompressionType{CompressionNone, CompressionGZIP},
			SupportedSerializationTypes: []SerializationType{SerializationJSON, SerializationProtobuf},
		}
	}
	return m.capabilities
}

// mockEventPublisher is a mock implementation for testing.
type mockEventPublisher struct {
	closed bool
}

func (m *mockEventPublisher) Publish(ctx context.Context, message *Message) error {
	if m.closed {
		return ErrPublisherClosedError
	}
	return nil
}

func (m *mockEventPublisher) PublishBatch(ctx context.Context, messages []*Message) error {
	if m.closed {
		return ErrPublisherClosedError
	}
	return nil
}

func (m *mockEventPublisher) PublishWithConfirm(ctx context.Context, message *Message) (*PublishConfirmation, error) {
	if m.closed {
		return nil, ErrPublisherClosedError
	}
	return &PublishConfirmation{
		MessageID: message.ID,
		Partition: 0,
		Offset:    123,
		Timestamp: time.Now(),
		Metadata:  map[string]string{"test": "value"},
	}, nil
}

func (m *mockEventPublisher) PublishAsync(ctx context.Context, message *Message, callback PublishCallback) error {
	if m.closed {
		return ErrPublisherClosedError
	}
	// Simulate async callback
	go func() {
		time.Sleep(10 * time.Millisecond)
		callback(&PublishConfirmation{
			MessageID: message.ID,
			Timestamp: time.Now(),
		}, nil)
	}()
	return nil
}

func (m *mockEventPublisher) BeginTransaction(ctx context.Context) (Transaction, error) {
	if m.closed {
		return nil, ErrPublisherClosedError
	}
	return &mockTransaction{}, nil
}

func (m *mockEventPublisher) Flush(ctx context.Context) error {
	if m.closed {
		return ErrPublisherClosedError
	}
	return nil
}

func (m *mockEventPublisher) Close() error {
	m.closed = true
	return nil
}

func (m *mockEventPublisher) GetMetrics() *PublisherMetrics {
	return &PublisherMetrics{
		MessagesPublished: 100,
		MessagesQueued:    5,
		MessagesFailed:    2,
		AvgPublishLatency: 10 * time.Millisecond,
		MessagesPerSecond: 50.0,
		ConnectionStatus:  "connected",
		LastActivity:      time.Now(),
	}
}

// mockEventSubscriber is a mock implementation for testing.
type mockEventSubscriber struct {
	closed bool
	paused bool
}

func (m *mockEventSubscriber) Subscribe(ctx context.Context, handler MessageHandler) error {
	if m.closed {
		return ErrSubscriberClosedError
	}
	return nil
}

func (m *mockEventSubscriber) SubscribeWithMiddleware(ctx context.Context, handler MessageHandler, middleware ...Middleware) error {
	if m.closed {
		return ErrSubscriberClosedError
	}
	return nil
}

func (m *mockEventSubscriber) Unsubscribe(ctx context.Context) error {
	return nil
}

func (m *mockEventSubscriber) Pause(ctx context.Context) error {
	if m.closed {
		return ErrSubscriberClosedError
	}
	m.paused = true
	return nil
}

func (m *mockEventSubscriber) Resume(ctx context.Context) error {
	if m.closed {
		return ErrSubscriberClosedError
	}
	m.paused = false
	return nil
}

func (m *mockEventSubscriber) Seek(ctx context.Context, position SeekPosition) error {
	if m.closed {
		return ErrSubscriberClosedError
	}
	return nil
}

func (m *mockEventSubscriber) GetLag(ctx context.Context) (int64, error) {
	if m.closed {
		return -1, ErrSubscriberClosedError
	}
	return 42, nil
}

func (m *mockEventSubscriber) Close() error {
	m.closed = true
	return nil
}

func (m *mockEventSubscriber) GetMetrics() *SubscriberMetrics {
	return &SubscriberMetrics{
		MessagesConsumed:  90,
		MessagesProcessed: 88,
		MessagesFailed:    2,
		AvgProcessingTime: 15 * time.Millisecond,
		MessagesPerSecond: 30.0,
		CurrentLag:        42,
		ConsumerGroup:     "test-group",
		ConnectionStatus:  "connected",
		LastActivity:      time.Now(),
	}
}

// mockTransaction is a mock implementation for testing.
type mockTransaction struct {
	id        string
	committed bool
	aborted   bool
}

func (m *mockTransaction) Publish(ctx context.Context, message *Message) error {
	if m.committed || m.aborted {
		return NewTransactionError("transaction already completed", nil)
	}
	return nil
}

func (m *mockTransaction) Commit(ctx context.Context) error {
	if m.committed || m.aborted {
		return NewTransactionError("transaction already completed", nil)
	}
	m.committed = true
	return nil
}

func (m *mockTransaction) Rollback(ctx context.Context) error {
	if m.committed || m.aborted {
		return NewTransactionError("transaction already completed", nil)
	}
	m.aborted = true
	return nil
}

func (m *mockTransaction) GetID() string {
	if m.id == "" {
		m.id = "tx-123"
	}
	return m.id
}

// TestMessageBrokerInterface tests the MessageBroker interface using mocks.
func TestMessageBrokerInterface(t *testing.T) {
	broker := &mockMessageBroker{}

	// Test Connect
	ctx := context.Background()
	err := broker.Connect(ctx)
	if err != nil {
		t.Errorf("Expected no error on connect, got: %v", err)
	}

	if !broker.IsConnected() {
		t.Error("Expected broker to be connected")
	}

	// Test CreatePublisher
	pubConfig := PublisherConfig{Topic: "test-topic"}
	publisher, err := broker.CreatePublisher(pubConfig)
	if err != nil {
		t.Errorf("Expected no error creating publisher, got: %v", err)
	}
	if publisher == nil {
		t.Error("Expected non-nil publisher")
	}

	// Test CreateSubscriber
	subConfig := SubscriberConfig{
		Topics:        []string{"test-topic"},
		ConsumerGroup: "test-group",
	}
	subscriber, err := broker.CreateSubscriber(subConfig)
	if err != nil {
		t.Errorf("Expected no error creating subscriber, got: %v", err)
	}
	if subscriber == nil {
		t.Error("Expected non-nil subscriber")
	}

	// Test HealthCheck
	health, err := broker.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Expected no error on health check, got: %v", err)
	}
	if health == nil || health.Status != HealthStatusHealthy {
		t.Error("Expected healthy status")
	}

	// Test GetMetrics
	metrics := broker.GetMetrics()
	if metrics == nil {
		t.Error("Expected non-nil metrics")
	}

	// Test GetCapabilities
	capabilities := broker.GetCapabilities()
	if capabilities == nil {
		t.Error("Expected non-nil capabilities")
	}

	// Test Disconnect
	err = broker.Disconnect(ctx)
	if err != nil {
		t.Errorf("Expected no error on disconnect, got: %v", err)
	}

	if broker.IsConnected() {
		t.Error("Expected broker to be disconnected")
	}

	// Test Close
	err = broker.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got: %v", err)
	}
}

// TestEventPublisherInterface tests the EventPublisher interface using mocks.
func TestEventPublisherInterface(t *testing.T) {
	publisher := &mockEventPublisher{}

	msg := &Message{
		ID:      "test-123",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	ctx := context.Background()

	// Test Publish
	err := publisher.Publish(ctx, msg)
	if err != nil {
		t.Errorf("Expected no error on publish, got: %v", err)
	}

	// Test PublishBatch
	messages := []*Message{msg, msg}
	err = publisher.PublishBatch(ctx, messages)
	if err != nil {
		t.Errorf("Expected no error on batch publish, got: %v", err)
	}

	// Test PublishWithConfirm
	confirmation, err := publisher.PublishWithConfirm(ctx, msg)
	if err != nil {
		t.Errorf("Expected no error on publish with confirm, got: %v", err)
	}
	if confirmation == nil || confirmation.MessageID != msg.ID {
		t.Error("Expected valid confirmation")
	}

	// Test PublishAsync
	callbackCalled := false
	callback := func(confirmation *PublishConfirmation, err error) {
		callbackCalled = true
		if err != nil {
			t.Errorf("Expected no error in callback, got: %v", err)
		}
		if confirmation == nil || confirmation.MessageID != msg.ID {
			t.Error("Expected valid confirmation in callback")
		}
	}

	err = publisher.PublishAsync(ctx, msg, callback)
	if err != nil {
		t.Errorf("Expected no error on async publish, got: %v", err)
	}

	// Wait for callback
	time.Sleep(20 * time.Millisecond)
	if !callbackCalled {
		t.Error("Expected callback to be called")
	}

	// Test BeginTransaction
	tx, err := publisher.BeginTransaction(ctx)
	if err != nil {
		t.Errorf("Expected no error on begin transaction, got: %v", err)
	}
	if tx == nil {
		t.Error("Expected non-nil transaction")
	}

	// Test transaction operations
	err = tx.Publish(ctx, msg)
	if err != nil {
		t.Errorf("Expected no error on transaction publish, got: %v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		t.Errorf("Expected no error on transaction commit, got: %v", err)
	}

	// Test Flush
	err = publisher.Flush(ctx)
	if err != nil {
		t.Errorf("Expected no error on flush, got: %v", err)
	}

	// Test GetMetrics
	metrics := publisher.GetMetrics()
	if metrics == nil {
		t.Error("Expected non-nil metrics")
	}

	// Test Close
	err = publisher.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got: %v", err)
	}

	// Test operations after close should return error
	err = publisher.Publish(ctx, msg)
	if !errors.Is(err, ErrPublisherClosedError) {
		t.Errorf("Expected ErrPublisherClosed after close, got: %v", err)
	}
}

// TestEventSubscriberInterface tests the EventSubscriber interface using mocks.
func TestEventSubscriberInterface(t *testing.T) {
	subscriber := &mockEventSubscriber{}
	ctx := context.Background()

	handler := MessageHandlerFunc(func(ctx context.Context, message *Message) error {
		return nil
	})

	// Test Subscribe
	err := subscriber.Subscribe(ctx, handler)
	if err != nil {
		t.Errorf("Expected no error on subscribe, got: %v", err)
	}

	// Test SubscribeWithMiddleware
	err = subscriber.SubscribeWithMiddleware(ctx, handler)
	if err != nil {
		t.Errorf("Expected no error on subscribe with middleware, got: %v", err)
	}

	// Test Pause
	err = subscriber.Pause(ctx)
	if err != nil {
		t.Errorf("Expected no error on pause, got: %v", err)
	}

	// Test Resume
	err = subscriber.Resume(ctx)
	if err != nil {
		t.Errorf("Expected no error on resume, got: %v", err)
	}

	// Test Seek
	seekPos := SeekPosition{
		Type:   SeekTypeBeginning,
		Offset: 0,
	}
	err = subscriber.Seek(ctx, seekPos)
	if err != nil {
		t.Errorf("Expected no error on seek, got: %v", err)
	}

	// Test GetLag
	lag, err := subscriber.GetLag(ctx)
	if err != nil {
		t.Errorf("Expected no error on get lag, got: %v", err)
	}
	if lag != 42 {
		t.Errorf("Expected lag of 42, got: %d", lag)
	}

	// Test GetMetrics
	metrics := subscriber.GetMetrics()
	if metrics == nil {
		t.Error("Expected non-nil metrics")
	}

	// Test Unsubscribe
	err = subscriber.Unsubscribe(ctx)
	if err != nil {
		t.Errorf("Expected no error on unsubscribe, got: %v", err)
	}

	// Test Close
	err = subscriber.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got: %v", err)
	}

	// Test operations after close should return error
	err = subscriber.Subscribe(ctx, handler)
	if !errors.Is(err, ErrSubscriberClosedError) {
		t.Errorf("Expected ErrSubscriberClosed after close, got: %v", err)
	}
}

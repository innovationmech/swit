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

package msgtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// AssertMessagesEqual asserts that two messages are equal in content.
func AssertMessagesEqual(t testing.TB, expected, actual *messaging.Message) {
	t.Helper()
	require.NotNil(t, expected, "expected message should not be nil")
	require.NotNil(t, actual, "actual message should not be nil")
	assert.Equal(t, expected.ID, actual.ID, "Message ID should match")
	assert.Equal(t, expected.Topic, actual.Topic, "Message topic should match")
	assert.Equal(t, expected.Headers, actual.Headers, "Message headers should match")
	assert.Equal(t, expected.Payload, actual.Payload, "Message payload should match")
	assert.Equal(t, expected.Key, actual.Key, "Message key should match")
}

// AssertMessageSlicesEqual asserts that two message slices are equal.
func AssertMessageSlicesEqual(t testing.TB, expected, actual []*messaging.Message) {
	t.Helper()
	require.Equal(t, len(expected), len(actual), "Message slice lengths should match")
	for i, expectedMsg := range expected {
		AssertMessagesEqual(t, expectedMsg, actual[i])
	}
}

// WaitForCondition waits for a condition to become true within a timeout
// period, polling every 10ms. It fails the test on timeout.
func WaitForCondition(t testing.TB, condition func() bool, timeout time.Duration, message string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for condition: %s", message)
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// WaitForMessages waits for a specific number of messages to be received.
func WaitForMessages(t testing.TB, getCount func() int, expectedCount int, timeout time.Duration) {
	t.Helper()
	WaitForCondition(t, func() bool {
		return getCount() >= expectedCount
	}, timeout, fmt.Sprintf("expected %d messages", expectedCount))
}

// BrokerSetup provides a complete pre-wired mock setup for message broker
// testing: broker, publisher, subscriber, handler, factory and configs.
type BrokerSetup struct {
	Broker           *MockMessageBroker
	Publisher        *MockEventPublisher
	Subscriber       *MockEventSubscriber
	Handler          *MockMessageHandler
	Factory          *MockBrokerFactory
	BrokerConfig     *messaging.BrokerConfig
	PublisherConfig  *messaging.PublisherConfig
	SubscriberConfig *messaging.SubscriberConfig
}

// NewBrokerSetup creates a complete test setup with all necessary mocks and
// default (optional) expectations already registered.
func NewBrokerSetup() *BrokerSetup {
	setup := &BrokerSetup{
		Broker:           NewMockMessageBroker(),
		Publisher:        NewMockEventPublisher(),
		Subscriber:       NewMockEventSubscriber(),
		Handler:          NewMockMessageHandler(),
		Factory:          NewMockBrokerFactory(),
		BrokerConfig:     NewTestBrokerConfig(),
		PublisherConfig:  NewTestPublisherConfig(),
		SubscriberConfig: NewTestSubscriberConfig(),
	}

	setup.SetupDefaultExpectations()

	return setup
}

// SetupDefaultExpectations sets up common optional mock expectations so
// tests only need to add expectations for the behavior under test.
func (s *BrokerSetup) SetupDefaultExpectations() {
	// Default broker expectations
	s.Broker.On("Connect", mock.Anything).Return(nil).Maybe()
	s.Broker.On("Disconnect", mock.Anything).Return(nil).Maybe()
	s.Broker.On("Close").Return(nil).Maybe()
	s.Broker.On("HealthCheck", mock.Anything).Return(NewHealthyStatus(), nil).Maybe()
	s.Broker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(s.Publisher, nil).Maybe()
	s.Broker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(s.Subscriber, nil).Maybe()

	// Default publisher expectations
	s.Publisher.On("Publish", mock.Anything, mock.AnythingOfType("*messaging.Message")).Return(nil).Maybe()
	s.Publisher.On("PublishBatch", mock.Anything, mock.AnythingOfType("[]*messaging.Message")).Return(nil).Maybe()
	s.Publisher.On("Flush", mock.Anything).Return(nil).Maybe()
	s.Publisher.On("Close").Return(nil).Maybe()

	// Default subscriber expectations
	s.Subscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil).Maybe()
	s.Subscriber.On("Unsubscribe", mock.Anything).Return(nil).Maybe()
	s.Subscriber.On("Close").Return(nil).Maybe()

	// Default handler expectations
	s.Handler.On("Handle", mock.Anything, mock.AnythingOfType("*messaging.Message")).Return(nil).Maybe()

	// Default factory expectations
	s.Factory.On("CreateBroker", mock.AnythingOfType("*messaging.BrokerConfig")).Return(s.Broker, nil).Maybe()
	s.Factory.On("GetSupportedBrokerTypes").Return([]messaging.BrokerType{messaging.BrokerTypeInMemory}).Maybe()
	s.Factory.On("ValidateConfig", mock.AnythingOfType("*messaging.BrokerConfig")).Return(nil).Maybe()
}

// AssertAllExpectations asserts that all mock expectations were met.
func (s *BrokerSetup) AssertAllExpectations(t *testing.T) {
	t.Helper()
	s.Broker.AssertExpectations(t)
	s.Publisher.AssertExpectations(t)
	s.Subscriber.AssertExpectations(t)
	s.Handler.AssertExpectations(t)
	s.Factory.AssertExpectations(t)
}

// BenchmarkHelper provides utilities for messaging performance benchmarking.
type BenchmarkHelper struct {
	MessageCount int
	MessageSize  int
	BatchSize    int
}

// NewBenchmarkHelper creates a new benchmark helper with default values.
func NewBenchmarkHelper() *BenchmarkHelper {
	return &BenchmarkHelper{
		MessageCount: 1000,
		MessageSize:  1024,
		BatchSize:    10,
	}
}

// CreateBenchmarkMessages creates messages suitable for benchmarking.
func (b *BenchmarkHelper) CreateBenchmarkMessages() []*messaging.Message {
	messages := make([]*messaging.Message, b.MessageCount)

	payload := make([]byte, b.MessageSize)
	for i := range payload {
		payload[i] = byte('A' + (i % 26))
	}

	for i := 0; i < b.MessageCount; i++ {
		messages[i] = &messaging.Message{
			ID:        fmt.Sprintf("bench-msg-%d", i),
			Headers:   map[string]string{"benchmark": "true"},
			Payload:   payload,
			Topic:     "benchmark-topic",
			Key:       []byte(fmt.Sprintf("key-%d", i%10)),
			Timestamp: time.Now(),
		}
	}

	return messages
}

// CreateBenchmarkBatches creates message batches for batch benchmarking.
func (b *BenchmarkHelper) CreateBenchmarkBatches() [][]*messaging.Message {
	messages := b.CreateBenchmarkMessages()
	batches := make([][]*messaging.Message, 0, (len(messages)+b.BatchSize-1)/b.BatchSize)

	for i := 0; i < len(messages); i += b.BatchSize {
		end := i + b.BatchSize
		if end > len(messages) {
			end = len(messages)
		}
		batches = append(batches, messages[i:end])
	}

	return batches
}

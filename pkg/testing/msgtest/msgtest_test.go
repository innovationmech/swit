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
	"errors"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMockMessageBrokerLifecycle(t *testing.T) {
	broker := NewMockMessageBroker()
	broker.On("Connect", mock.Anything).Return(nil)
	broker.On("Disconnect", mock.Anything).Return(nil)

	ctx := context.Background()
	require.False(t, broker.IsConnected())

	require.NoError(t, broker.Connect(ctx))
	assert.True(t, broker.IsConnected())

	require.NoError(t, broker.Disconnect(ctx))
	assert.False(t, broker.IsConnected())

	assert.NotNil(t, broker.GetMetrics())
	assert.NotNil(t, broker.GetCapabilities())
	broker.AssertExpectations(t)
}

func TestMockMessageBrokerCreatePublisherAndSubscriber(t *testing.T) {
	broker := NewMockMessageBroker()
	publisher := NewMockEventPublisher()
	subscriber := NewMockEventSubscriber()

	broker.On("CreatePublisher", mock.AnythingOfType("messaging.PublisherConfig")).Return(publisher, nil)
	broker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(subscriber, nil)

	pub, err := broker.CreatePublisher(*NewTestPublisherConfig())
	require.NoError(t, err)
	assert.Same(t, publisher, pub)

	sub, err := broker.CreateSubscriber(*NewTestSubscriberConfig())
	require.NoError(t, err)
	assert.Same(t, subscriber, sub)

	assert.Len(t, broker.GetPublishers(), 1)
	assert.Len(t, broker.GetSubscribers(), 1)
	broker.AssertExpectations(t)
}

func TestMockEventPublisherRecordsMessages(t *testing.T) {
	publisher := NewMockEventPublisher()
	publisher.On("Publish", mock.Anything, mock.AnythingOfType("*messaging.Message")).Return(nil)
	publisher.On("PublishBatch", mock.Anything, mock.AnythingOfType("[]*messaging.Message")).Return(nil)

	ctx := context.Background()
	msg := NewTestMessage("msg-1", "test.event", map[string]interface{}{"k": "v"})
	require.NoError(t, publisher.Publish(ctx, msg))

	batch := NewTestMessages(3, "test.batch")
	require.NoError(t, publisher.PublishBatch(ctx, batch))

	published := publisher.GetPublishedMessages()
	require.Len(t, published, 4)
	AssertMessagesEqual(t, msg, published[0])
	AssertMessageSlicesEqual(t, batch, published[1:])

	publisher.ClearPublishedMessages()
	assert.Empty(t, publisher.GetPublishedMessages())
	publisher.AssertExpectations(t)
}

func TestMockEventPublisherPublishAsyncInvokesCallback(t *testing.T) {
	publisher := NewMockEventPublisher()
	publisher.On("PublishAsync", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	var confirmed *messaging.PublishConfirmation
	msg := NewTestMessage("async-1", "test.async", nil)
	err := publisher.PublishAsync(context.Background(), msg, func(confirmation *messaging.PublishConfirmation, err error) {
		confirmed = confirmation
	})
	require.NoError(t, err)
	require.NotNil(t, confirmed)
	assert.Equal(t, "async-1", confirmed.MessageID)
}

func TestMockEventSubscriberSimulateMessage(t *testing.T) {
	subscriber := NewMockEventSubscriber()
	handler := NewMockMessageHandler()

	subscriber.On("Subscribe", mock.Anything, mock.Anything).Return(nil)
	handler.On("Handle", mock.Anything, mock.AnythingOfType("*messaging.Message")).Return(nil)

	ctx := context.Background()
	require.NoError(t, subscriber.Subscribe(ctx, handler))
	assert.True(t, subscriber.IsSubscribed())

	msg := NewTestMessage("sim-1", "test.sim", nil)
	require.NoError(t, subscriber.SimulateMessage(ctx, msg))

	WaitForMessages(t, func() int { return len(handler.GetHandledMessages()) }, 1, time.Second)
	assert.Equal(t, int64(1), subscriber.GetMessageCount())
	AssertMessagesEqual(t, msg, handler.GetHandledMessages()[0])
}

func TestMockEventSubscriberSubscribeWithMiddleware(t *testing.T) {
	subscriber := NewMockEventSubscriber()
	handler := NewMockMessageHandler()
	middleware := NewMockMiddleware()

	subscriber.On("SubscribeWithMiddleware", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	middleware.On("Wrap", mock.Anything).Return(nil) // nil means pass-through
	handler.On("Handle", mock.Anything, mock.AnythingOfType("*messaging.Message")).Return(nil)

	ctx := context.Background()
	require.NoError(t, subscriber.SubscribeWithMiddleware(ctx, handler, middleware))
	require.NoError(t, subscriber.SimulateMessage(ctx, NewTestMessage("mw-1", "test.mw", nil)))
	assert.Len(t, handler.GetHandledMessages(), 1)
	middleware.AssertExpectations(t)
}

func TestMockTransactionStateTracking(t *testing.T) {
	tx := NewMockTransaction()
	tx.On("Publish", mock.Anything, mock.Anything).Return(nil)
	tx.On("Commit", mock.Anything).Return(nil)

	ctx := context.Background()
	require.NoError(t, tx.Publish(ctx, NewTestMessage("tx-1", "test.tx", nil)))
	require.NoError(t, tx.Commit(ctx))

	assert.True(t, tx.IsCommitted())
	assert.False(t, tx.IsAborted())
	assert.Equal(t, "mock-transaction", tx.GetID())
}

func TestMockBrokerFactory(t *testing.T) {
	factory := NewMockBrokerFactory()
	broker := NewMockMessageBroker()

	factory.On("CreateBroker", mock.AnythingOfType("*messaging.BrokerConfig")).Return(broker, nil)
	factory.On("ValidateConfig", mock.AnythingOfType("*messaging.BrokerConfig")).Return(nil)

	config := NewTestBrokerConfig()
	require.NoError(t, factory.ValidateConfig(config))

	created, err := factory.CreateBroker(config)
	require.NoError(t, err)
	assert.Same(t, broker, created)
	assert.Len(t, factory.GetCreatedBrokers(), 1)
	factory.AssertExpectations(t)
}

func TestBrokerSetupDefaultExpectations(t *testing.T) {
	setup := NewBrokerSetup()
	ctx := context.Background()

	require.NoError(t, setup.Broker.Connect(ctx))

	pub, err := setup.Broker.CreatePublisher(*setup.PublisherConfig)
	require.NoError(t, err)
	require.NoError(t, pub.Publish(ctx, NewTestMessage("setup-1", "test.setup", nil)))

	sub, err := setup.Broker.CreateSubscriber(*setup.SubscriberConfig)
	require.NoError(t, err)
	require.NoError(t, sub.Subscribe(ctx, setup.Handler))

	status, err := setup.Broker.HealthCheck(ctx)
	require.NoError(t, err)
	assert.Equal(t, messaging.HealthStatusHealthy, status.Status)

	setup.AssertAllExpectations(t)
}

func TestMockMessageHandlerOnError(t *testing.T) {
	handler := NewMockMessageHandler()
	handler.On("Handle", mock.Anything, mock.Anything).Return(errors.New("boom"))
	handler.On("OnError", mock.Anything, mock.Anything, mock.Anything).Return(messaging.ErrorActionDeadLetter)

	ctx := context.Background()
	msg := NewTestMessage("err-1", "test.err", nil)
	err := handler.Handle(ctx, msg)
	require.Error(t, err)
	assert.Empty(t, handler.GetHandledMessages())

	action := handler.OnError(ctx, msg, err)
	assert.Equal(t, messaging.ErrorActionDeadLetter, action)
}

func TestWaitForConditionSucceeds(t *testing.T) {
	start := time.Now()
	WaitForCondition(t, func() bool {
		return time.Since(start) > 20*time.Millisecond
	}, time.Second, "condition should eventually be true")
}

func TestBenchmarkHelperCreatesMessagesAndBatches(t *testing.T) {
	helper := NewBenchmarkHelper()
	helper.MessageCount = 25
	helper.BatchSize = 10

	messages := helper.CreateBenchmarkMessages()
	require.Len(t, messages, 25)
	assert.Equal(t, "benchmark-topic", messages[0].Topic)

	batches := helper.CreateBenchmarkBatches()
	require.Len(t, batches, 3)
	assert.Len(t, batches[0], 10)
	assert.Len(t, batches[2], 5)
}

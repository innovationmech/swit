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

package inmemory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newConnectedBroker(t *testing.T) *Broker {
	t.Helper()
	b := New(&messaging.BrokerConfig{Type: messaging.BrokerTypeInMemory})
	require.NoError(t, b.Connect(context.Background()))
	t.Cleanup(func() { _ = b.Close() })
	return b
}

// collectingHandler records handled messages and exposes them via a channel.
type collectingHandler struct {
	mu       sync.Mutex
	messages []*messaging.Message
	notify   chan *messaging.Message
}

func newCollectingHandler(buffer int) *collectingHandler {
	return &collectingHandler{notify: make(chan *messaging.Message, buffer)}
}

func (h *collectingHandler) Handle(_ context.Context, msg *messaging.Message) error {
	h.mu.Lock()
	h.messages = append(h.messages, msg)
	h.mu.Unlock()
	h.notify <- msg
	return nil
}

func (h *collectingHandler) OnError(_ context.Context, _ *messaging.Message, _ error) messaging.ErrorAction {
	return messaging.ErrorActionRetry
}

func (h *collectingHandler) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.messages)
}

func waitForMessages(t *testing.T, ch <-chan *messaging.Message, n int) []*messaging.Message {
	t.Helper()
	received := make([]*messaging.Message, 0, n)
	timeout := time.After(3 * time.Second)
	for len(received) < n {
		select {
		case msg := <-ch:
			received = append(received, msg)
		case <-timeout:
			t.Fatalf("timed out waiting for %d messages, got %d", n, len(received))
		}
	}
	return received
}

func TestBrokerLifecycle(t *testing.T) {
	b := New(nil)
	assert.False(t, b.IsConnected())

	require.NoError(t, b.Connect(context.Background()))
	assert.True(t, b.IsConnected())

	// Connect is idempotent.
	require.NoError(t, b.Connect(context.Background()))

	status, err := b.HealthCheck(context.Background())
	require.NoError(t, err)
	assert.Equal(t, messaging.HealthStatusHealthy, status.Status)

	require.NoError(t, b.Disconnect(context.Background()))
	assert.False(t, b.IsConnected())

	status, err = b.HealthCheck(context.Background())
	require.NoError(t, err)
	assert.Equal(t, messaging.HealthStatusUnhealthy, status.Status)

	// Disconnect is idempotent, Close maps to Disconnect.
	require.NoError(t, b.Disconnect(context.Background()))
	require.NoError(t, b.Close())
}

func TestBrokerCapabilities(t *testing.T) {
	b := New(nil)
	caps := b.GetCapabilities()
	require.NotNil(t, caps)
	assert.True(t, caps.SupportsTransactions)
	assert.True(t, caps.SupportsSeek)
	assert.True(t, caps.SupportsConsumerGroups)
}

func TestCreatePublisherValidation(t *testing.T) {
	b := New(nil)

	_, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "orders"})
	assert.ErrorIs(t, err, messaging.ErrBrokerNotConnected)

	require.NoError(t, b.Connect(context.Background()))
	t.Cleanup(func() { _ = b.Close() })

	_, err = b.CreatePublisher(messaging.PublisherConfig{})
	assert.Error(t, err)

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "orders"})
	require.NoError(t, err)
	assert.NotNil(t, pub)
}

func TestCreateSubscriberValidation(t *testing.T) {
	b := New(nil)

	_, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"orders"}})
	assert.ErrorIs(t, err, messaging.ErrBrokerNotConnected)

	require.NoError(t, b.Connect(context.Background()))
	t.Cleanup(func() { _ = b.Close() })

	_, err = b.CreateSubscriber(messaging.SubscriberConfig{})
	assert.Error(t, err)

	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"orders"}, ConsumerGroup: "g1"})
	require.NoError(t, err)
	assert.NotNil(t, sub)
}

func TestPublishSubscribeRoundtrip(t *testing.T) {
	b := newConnectedBroker(t)

	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"orders"}, ConsumerGroup: "g1"})
	require.NoError(t, err)
	handler := newCollectingHandler(10)
	require.NoError(t, sub.Subscribe(context.Background(), handler))
	t.Cleanup(func() { _ = sub.Close() })

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "orders"})
	require.NoError(t, err)

	require.NoError(t, pub.Publish(context.Background(), &messaging.Message{Payload: []byte("hello")}))

	msgs := waitForMessages(t, handler.notify, 1)
	assert.Equal(t, "orders", msgs[0].Topic)
	assert.Equal(t, []byte("hello"), msgs[0].Payload)
	assert.NotEmpty(t, msgs[0].ID)
	assert.False(t, msgs[0].Timestamp.IsZero())

	lag, err := sub.GetLag(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), lag)
}

func TestPublishBatchAndConfirm(t *testing.T) {
	b := newConnectedBroker(t)

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "batch"})
	require.NoError(t, err)

	batch := []*messaging.Message{
		{Payload: []byte("m1")},
		{Payload: []byte("m2")},
		{Payload: []byte("m3")},
	}
	require.NoError(t, pub.PublishBatch(context.Background(), batch))

	confirmation, err := pub.PublishWithConfirm(context.Background(), &messaging.Message{Payload: []byte("m4")})
	require.NoError(t, err)
	assert.NotEmpty(t, confirmation.MessageID)
	assert.Equal(t, int64(3), confirmation.Offset)

	metrics := pub.GetMetrics()
	assert.Equal(t, int64(4), metrics.MessagesPublished)
	assert.Equal(t, int64(1), metrics.BatchesPublished)
}

func TestPublishAsyncCallback(t *testing.T) {
	b := newConnectedBroker(t)

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "async"})
	require.NoError(t, err)

	done := make(chan *messaging.PublishConfirmation, 1)
	err = pub.PublishAsync(context.Background(), &messaging.Message{Payload: []byte("a1")}, func(confirmation *messaging.PublishConfirmation, err error) {
		require.NoError(t, err)
		done <- confirmation
	})
	require.NoError(t, err)

	select {
	case confirmation := <-done:
		assert.NotEmpty(t, confirmation.MessageID)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for async publish callback")
	}
}

func TestPublisherClosedRejectsOperations(t *testing.T) {
	b := newConnectedBroker(t)

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "closed"})
	require.NoError(t, err)
	require.NoError(t, pub.Close())

	err = pub.Publish(context.Background(), &messaging.Message{Payload: []byte("x")})
	assert.ErrorIs(t, err, messaging.ErrPublisherAlreadyClosed)

	_, err = pub.BeginTransaction(context.Background())
	assert.ErrorIs(t, err, messaging.ErrPublisherAlreadyClosed)

	assert.ErrorIs(t, pub.Flush(context.Background()), messaging.ErrPublisherAlreadyClosed)
}

func TestConsumerGroupsFanOutAndQueueSemantics(t *testing.T) {
	b := newConnectedBroker(t)

	// Two distinct groups: each receives every message.
	subA, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"events"}, ConsumerGroup: "group-a"})
	require.NoError(t, err)
	handlerA := newCollectingHandler(10)
	require.NoError(t, subA.Subscribe(context.Background(), handlerA))
	t.Cleanup(func() { _ = subA.Close() })

	subB, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"events"}, ConsumerGroup: "group-b"})
	require.NoError(t, err)
	handlerB := newCollectingHandler(10)
	require.NoError(t, subB.Subscribe(context.Background(), handlerB))
	t.Cleanup(func() { _ = subB.Close() })

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "events"})
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		require.NoError(t, pub.Publish(context.Background(), &messaging.Message{Payload: []byte{byte(i)}}))
	}

	waitForMessages(t, handlerA.notify, 3)
	waitForMessages(t, handlerB.notify, 3)

	// Two subscribers in the same group: messages are split, each message
	// is delivered exactly once within the group.
	shared := newCollectingHandler(20)
	subC, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"queue"}, ConsumerGroup: "workers"})
	require.NoError(t, err)
	require.NoError(t, subC.Subscribe(context.Background(), shared))
	t.Cleanup(func() { _ = subC.Close() })

	subD, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"queue"}, ConsumerGroup: "workers"})
	require.NoError(t, err)
	require.NoError(t, subD.Subscribe(context.Background(), shared))
	t.Cleanup(func() { _ = subD.Close() })

	queuePub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "queue"})
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		require.NoError(t, queuePub.Publish(context.Background(), &messaging.Message{Payload: []byte{byte(i)}}))
	}

	received := waitForMessages(t, shared.notify, 10)
	seen := make(map[string]bool, len(received))
	for _, msg := range received {
		assert.False(t, seen[msg.ID], "message %s delivered more than once within group", msg.ID)
		seen[msg.ID] = true
	}
}

func TestSubscribeEarliestReplaysHistory(t *testing.T) {
	b := newConnectedBroker(t)

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "history"})
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		require.NoError(t, pub.Publish(context.Background(), &messaging.Message{Payload: []byte{byte(i)}}))
	}

	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{
		Topics:        []string{"history"},
		ConsumerGroup: "replay",
		Offset:        messaging.OffsetConfig{Initial: messaging.OffsetEarliest},
	})
	require.NoError(t, err)
	handler := newCollectingHandler(10)
	require.NoError(t, sub.Subscribe(context.Background(), handler))
	t.Cleanup(func() { _ = sub.Close() })

	waitForMessages(t, handler.notify, 3)
}

func TestSeekBeginningReplaysMessages(t *testing.T) {
	b := newConnectedBroker(t)

	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"seek"}, ConsumerGroup: "g1"})
	require.NoError(t, err)
	handler := newCollectingHandler(10)
	require.NoError(t, sub.Subscribe(context.Background(), handler))
	t.Cleanup(func() { _ = sub.Close() })

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "seek"})
	require.NoError(t, err)
	require.NoError(t, pub.Publish(context.Background(), &messaging.Message{Payload: []byte("first")}))
	waitForMessages(t, handler.notify, 1)

	require.NoError(t, sub.Seek(context.Background(), messaging.SeekPosition{Type: messaging.SeekTypeBeginning}))
	replayed := waitForMessages(t, handler.notify, 1)
	assert.Equal(t, []byte("first"), replayed[0].Payload)

	// Seeking a topic not covered by the subscription fails.
	err = sub.Seek(context.Background(), messaging.SeekPosition{Type: messaging.SeekTypeBeginning, Topic: "other"})
	assert.Error(t, err)
}

func TestGetLagWithLatestOffset(t *testing.T) {
	b := newConnectedBroker(t)

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "lag"})
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		require.NoError(t, pub.Publish(context.Background(), &messaging.Message{Payload: []byte{byte(i)}}))
	}

	// Group not yet created: lag counts the full log.
	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"lag"}, ConsumerGroup: "lag-group"})
	require.NoError(t, err)
	lag, err := sub.GetLag(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(5), lag)
}

func TestPauseResume(t *testing.T) {
	b := newConnectedBroker(t)

	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"pause"}, ConsumerGroup: "g1"})
	require.NoError(t, err)
	handler := newCollectingHandler(10)
	require.NoError(t, sub.Subscribe(context.Background(), handler))
	t.Cleanup(func() { _ = sub.Close() })

	require.NoError(t, sub.Pause(context.Background()))

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "pause"})
	require.NoError(t, err)
	require.NoError(t, pub.Publish(context.Background(), &messaging.Message{Payload: []byte("queued")}))

	select {
	case <-handler.notify:
		t.Fatal("received message while paused")
	case <-time.After(100 * time.Millisecond):
	}

	require.NoError(t, sub.Resume(context.Background()))
	waitForMessages(t, handler.notify, 1)
}

// failingHandler always fails and routes errors to the dead-letter queue.
type failingHandler struct {
	attempts chan int
}

func (h *failingHandler) Handle(_ context.Context, msg *messaging.Message) error {
	h.attempts <- msg.DeliveryAttempt
	return errors.New("handler failure")
}

func (h *failingHandler) OnError(_ context.Context, _ *messaging.Message, _ error) messaging.ErrorAction {
	return messaging.ErrorActionDeadLetter
}

func TestDeadLetterRouting(t *testing.T) {
	b := newConnectedBroker(t)

	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{
		Topics:        []string{"work"},
		ConsumerGroup: "workers",
		DeadLetter:    messaging.DeadLetterConfig{Enabled: true, Topic: "work.dlq"},
	})
	require.NoError(t, err)
	failing := &failingHandler{attempts: make(chan int, 10)}
	require.NoError(t, sub.Subscribe(context.Background(), failing))
	t.Cleanup(func() { _ = sub.Close() })

	// Independent subscriber on the DLQ topic.
	dlqSub, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"work.dlq"}, ConsumerGroup: "dlq-watchers"})
	require.NoError(t, err)
	dlqHandler := newCollectingHandler(10)
	require.NoError(t, dlqSub.Subscribe(context.Background(), dlqHandler))
	t.Cleanup(func() { _ = dlqSub.Close() })

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "work"})
	require.NoError(t, err)
	require.NoError(t, pub.Publish(context.Background(), &messaging.Message{Payload: []byte("poison")}))

	dlqMsgs := waitForMessages(t, dlqHandler.notify, 1)
	assert.Equal(t, "work.dlq", dlqMsgs[0].Topic)
	assert.Equal(t, []byte("poison"), dlqMsgs[0].Payload)

	metrics := sub.GetMetrics()
	assert.Equal(t, int64(1), metrics.MessagesDeadLettered)
}

// retryThenSucceedHandler fails a fixed number of times before succeeding.
type retryThenSucceedHandler struct {
	mu        sync.Mutex
	failures  int
	succeeded chan *messaging.Message
}

func (h *retryThenSucceedHandler) Handle(_ context.Context, msg *messaging.Message) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.failures > 0 {
		h.failures--
		return errors.New("transient failure")
	}
	h.succeeded <- msg
	return nil
}

func (h *retryThenSucceedHandler) OnError(_ context.Context, _ *messaging.Message, _ error) messaging.ErrorAction {
	return messaging.ErrorActionRetry
}

func TestRetryEventuallySucceeds(t *testing.T) {
	b := newConnectedBroker(t)

	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{
		Topics:        []string{"retry"},
		ConsumerGroup: "g1",
		Retry:         messaging.RetryConfig{MaxAttempts: 3},
	})
	require.NoError(t, err)
	handler := &retryThenSucceedHandler{failures: 2, succeeded: make(chan *messaging.Message, 1)}
	require.NoError(t, sub.Subscribe(context.Background(), handler))
	t.Cleanup(func() { _ = sub.Close() })

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "retry"})
	require.NoError(t, err)
	require.NoError(t, pub.Publish(context.Background(), &messaging.Message{Payload: []byte("flaky")}))

	select {
	case msg := <-handler.succeeded:
		assert.Equal(t, 3, msg.DeliveryAttempt)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for retried message to succeed")
	}

	metrics := sub.GetMetrics()
	assert.Equal(t, int64(2), metrics.MessagesRetried)
}

func TestTransactionCommitAndRollback(t *testing.T) {
	b := newConnectedBroker(t)

	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"tx"}, ConsumerGroup: "g1"})
	require.NoError(t, err)
	handler := newCollectingHandler(10)
	require.NoError(t, sub.Subscribe(context.Background(), handler))
	t.Cleanup(func() { _ = sub.Close() })

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "tx"})
	require.NoError(t, err)

	// Rollback: messages never become visible.
	tx, err := pub.BeginTransaction(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, tx.GetID())
	require.NoError(t, tx.Publish(context.Background(), &messaging.Message{Payload: []byte("discarded")}))
	require.NoError(t, tx.Rollback(context.Background()))

	select {
	case <-handler.notify:
		t.Fatal("received message from rolled back transaction")
	case <-time.After(100 * time.Millisecond):
	}

	// Commit: messages become visible atomically.
	tx, err = pub.BeginTransaction(context.Background())
	require.NoError(t, err)
	require.NoError(t, tx.Publish(context.Background(), &messaging.Message{Payload: []byte("tx1")}))
	require.NoError(t, tx.Publish(context.Background(), &messaging.Message{Payload: []byte("tx2")}))
	require.NoError(t, tx.Commit(context.Background()))

	msgs := waitForMessages(t, handler.notify, 2)
	assert.Equal(t, []byte("tx1"), msgs[0].Payload)
	assert.Equal(t, []byte("tx2"), msgs[1].Payload)

	// Completed transactions reject further use.
	assert.Error(t, tx.Commit(context.Background()))
	assert.Error(t, tx.Publish(context.Background(), &messaging.Message{Payload: []byte("late")}))
}

// upperMiddleware tags messages processed through the middleware chain.
type headerMiddleware struct{ name string }

func (m *headerMiddleware) Name() string { return m.name }

func (m *headerMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
	return messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		if msg.Headers == nil {
			msg.Headers = map[string]string{}
		}
		msg.Headers["middleware"] = m.name
		return next.Handle(ctx, msg)
	})
}

func TestSubscribeWithMiddleware(t *testing.T) {
	b := newConnectedBroker(t)

	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"mw"}, ConsumerGroup: "g1"})
	require.NoError(t, err)
	handler := newCollectingHandler(10)
	require.NoError(t, sub.SubscribeWithMiddleware(context.Background(), handler, &headerMiddleware{name: "outer"}))
	t.Cleanup(func() { _ = sub.Close() })

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "mw"})
	require.NoError(t, err)
	require.NoError(t, pub.Publish(context.Background(), &messaging.Message{Payload: []byte("m")}))

	msgs := waitForMessages(t, handler.notify, 1)
	assert.Equal(t, "outer", msgs[0].Headers["middleware"])
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	b := newConnectedBroker(t)

	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"stop"}, ConsumerGroup: "g1"})
	require.NoError(t, err)
	handler := newCollectingHandler(10)
	require.NoError(t, sub.Subscribe(context.Background(), handler))

	require.NoError(t, sub.Unsubscribe(context.Background()))

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "stop"})
	require.NoError(t, err)
	require.NoError(t, pub.Publish(context.Background(), &messaging.Message{Payload: []byte("late")}))

	select {
	case <-handler.notify:
		t.Fatal("received message after unsubscribe")
	case <-time.After(100 * time.Millisecond):
	}

	require.NoError(t, sub.Close())

	// Closed subscribers reject new subscriptions.
	err = sub.Subscribe(context.Background(), handler)
	assert.ErrorIs(t, err, messaging.ErrSubscriberAlreadyClosed)
}

func TestDisconnectClosesChildren(t *testing.T) {
	b := New(nil)
	require.NoError(t, b.Connect(context.Background()))

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "shutdown"})
	require.NoError(t, err)

	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"shutdown"}, ConsumerGroup: "g1"})
	require.NoError(t, err)
	handler := newCollectingHandler(10)
	require.NoError(t, sub.Subscribe(context.Background(), handler))

	require.NoError(t, b.Disconnect(context.Background()))

	err = pub.Publish(context.Background(), &messaging.Message{Payload: []byte("x")})
	assert.ErrorIs(t, err, messaging.ErrPublisherAlreadyClosed)
}

func TestBrokerMetrics(t *testing.T) {
	b := newConnectedBroker(t)

	pub, err := b.CreatePublisher(messaging.PublisherConfig{Topic: "metrics"})
	require.NoError(t, err)

	sub, err := b.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"metrics"}, ConsumerGroup: "g1"})
	require.NoError(t, err)
	handler := newCollectingHandler(10)
	require.NoError(t, sub.Subscribe(context.Background(), handler))
	t.Cleanup(func() { _ = sub.Close() })

	require.NoError(t, pub.Publish(context.Background(), &messaging.Message{Payload: []byte("m")}))
	waitForMessages(t, handler.notify, 1)

	metrics := b.GetMetrics()
	assert.Equal(t, "connected", metrics.ConnectionStatus)
	assert.Equal(t, int64(1), metrics.PublishersCreated)
	assert.Equal(t, int64(1), metrics.SubscribersCreated)
	assert.Equal(t, int64(1), metrics.MessagesPublished)
	assert.Equal(t, int64(1), metrics.MessagesConsumed)
}

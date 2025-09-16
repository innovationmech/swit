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

package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"
)

type retryAlwaysHandler struct{}

func (retryAlwaysHandler) Handle(ctx context.Context, msg *messaging.Message) error {
	return messaging.NewProcessingError("fail", nil)
}

func (retryAlwaysHandler) OnError(ctx context.Context, msg *messaging.Message, err error) messaging.ErrorAction {
	return messaging.ErrorActionRetry
}

// Test retry path: handler returns retry, message is republished to <queue>.retry and original acked
func TestRabbitConsumerRetryRepublish(t *testing.T) {
	base := &messaging.BrokerConfig{Type: messaging.BrokerTypeRabbitMQ, Endpoints: []string{"amqp://localhost:5672"}}
	base.Connection.PoolSize = 1
	rabbitCfg := DefaultConfig()
	broker := newRabbitBroker(base, rabbitCfg)

	mconn := newMockConnection(func() amqpChannel { return newMockChannel() })
	broker.pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) { return mconn, nil }
	require.NoError(t, broker.Connect(context.Background()))

	sub, err := broker.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"orders"}, ConsumerGroup: "cg", Processing: messaging.ProcessingConfig{AckMode: messaging.AckModeManual}})
	require.NoError(t, err)
	rsub := sub.(*rabbitSubscriber)

	go func() { _ = rsub.Subscribe(context.Background(), retryAlwaysHandler{}) }()

	// Wait a moment for consumer to start
	time.Sleep(50 * time.Millisecond)

	// Enqueue a delivery
	mc := mconn.channels[0].(*mockChannel)
	mc.enqueueDelivery(amqp.Delivery{Body: []byte("x"), DeliveryTag: 1, MessageId: "id-1"})

	// Give time to process
	time.Sleep(50 * time.Millisecond)

	// Inspect publish calls: should have published to orders.retry
	mc2 := mconn.channels[0].(*mockChannel)
	require.NotEmpty(t, mc2.publishCalls)
	last := mc2.lastPublish()
	require.NotNil(t, last)
	require.Equal(t, "orders.retry", last.routingKey)
}

// Test DLQ path: when max retries exceeded, message goes to <queue>.dlq
func TestRabbitConsumerDeadLetterRepublish(t *testing.T) {
	base := &messaging.BrokerConfig{Type: messaging.BrokerTypeRabbitMQ, Endpoints: []string{"amqp://localhost:5672"}}
	base.Connection.PoolSize = 1
	rabbitCfg := DefaultConfig()
	broker := newRabbitBroker(base, rabbitCfg)

	mconn := newMockConnection(func() amqpChannel { return newMockChannel() })
	broker.pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) { return mconn, nil }
	require.NoError(t, broker.Connect(context.Background()))

	cfg := messaging.SubscriberConfig{Topics: []string{"orders"}, ConsumerGroup: "cg", Processing: messaging.ProcessingConfig{AckMode: messaging.AckModeManual}}
	cfg.DeadLetter.MaxRetries = 1
	sub, err := broker.CreateSubscriber(cfg)
	require.NoError(t, err)
	rsub := sub.(*rabbitSubscriber)

	h := messaging.MessageHandlerFunc(func(ctx context.Context, msg *messaging.Message) error {
		return messaging.NewProcessingError("fail", nil)
	})
	go func() { _ = rsub.Subscribe(context.Background(), h) }()
	time.Sleep(50 * time.Millisecond)

	mc := mconn.channels[0].(*mockChannel)
	// First failure -> retry
	mc.enqueueDelivery(amqp.Delivery{Body: []byte("x"), DeliveryTag: 1, MessageId: "id-1"})
	time.Sleep(30 * time.Millisecond)
	// Simulate consumption of retry message (with x-retry-attempt=1) -> now exceed and DLQ
	mc.enqueueDelivery(amqp.Delivery{Body: []byte("x"), DeliveryTag: 2, MessageId: "id-1", Headers: amqp.Table{"x-retry-attempt": 1}})
	time.Sleep(50 * time.Millisecond)

	last := mc.lastPublish()
	require.NotNil(t, last)
	require.Equal(t, "orders.dlq", last.routingKey)
}

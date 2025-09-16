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

func TestRabbitPublisherPublishSimple(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	base.Connection.PoolSize = 1

	rabbitCfg := DefaultConfig()
	broker := newRabbitBroker(base, rabbitCfg)

	var conn *mockConnection
	broker.pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) {
		conn = newMockConnection(func() amqpChannel { return newMockChannel() })
		return conn, nil
	}

	require.NoError(t, broker.Connect(context.Background()))
	require.NotNil(t, conn)

	pubAny, err := broker.CreatePublisher(messaging.PublisherConfig{Topic: "orders"})
	require.NoError(t, err)

	publisher, ok := pubAny.(*rabbitPublisher)
	require.True(t, ok)

	channel := newMockChannel()
	conn.enqueueChannel(channel)

	msg := &messaging.Message{
		ID:        "message-1",
		Topic:     "orders",
		Payload:   []byte("hello"),
		Timestamp: time.Now(),
	}

	require.NoError(t, publisher.Publish(context.Background(), msg))

	call := channel.lastPublish()
	require.NotNil(t, call)
	require.Equal(t, "orders", call.routingKey)
	require.Equal(t, "", call.exchange)
}

func TestRabbitPublisherPublishWithConfirmAck(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	base.Connection.PoolSize = 1

	rabbitCfg := DefaultConfig()
	broker := newRabbitBroker(base, rabbitCfg)

	var conn *mockConnection
	broker.pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) {
		conn = newMockConnection(func() amqpChannel { return newMockChannel() })
		return conn, nil
	}

	require.NoError(t, broker.Connect(context.Background()))
	require.NotNil(t, conn)

	cfg := messaging.PublisherConfig{
		Topic: "events",
		Confirmation: messaging.ConfirmationConfig{
			Required: true,
			Timeout:  50 * time.Millisecond,
		},
	}

	pubAny, err := broker.CreatePublisher(cfg)
	require.NoError(t, err)
	publisher := pubAny.(*rabbitPublisher)

	channel := newMockChannel()
	channel.queueConfirmation(amqp.Confirmation{DeliveryTag: 42, Ack: true})
	conn.enqueueChannel(channel)

	msg := &messaging.Message{
		ID:        "confirm-1",
		Topic:     "events",
		Payload:   []byte("payload"),
		Timestamp: time.Now(),
	}

	confirmation, err := publisher.PublishWithConfirm(context.Background(), msg)
	require.NoError(t, err)
	require.NotNil(t, confirmation)
	require.Equal(t, "confirm-1", confirmation.MessageID)
	require.Equal(t, "42", confirmation.Metadata["delivery_tag"])
	require.Equal(t, "events", confirmation.Metadata["routing_key"])
}

func TestRabbitPublisherPublishWithConfirmRetryOnNack(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	base.Connection.PoolSize = 1

	rabbitCfg := DefaultConfig()
	broker := newRabbitBroker(base, rabbitCfg)

	var conn *mockConnection
	broker.pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) {
		conn = newMockConnection(func() amqpChannel { return newMockChannel() })
		return conn, nil
	}

	require.NoError(t, broker.Connect(context.Background()))
	require.NotNil(t, conn)

	cfg := messaging.PublisherConfig{
		Topic: "orders",
		Confirmation: messaging.ConfirmationConfig{
			Required: true,
			Timeout:  50 * time.Millisecond,
			Retries:  1,
		},
	}

	pubAny, err := broker.CreatePublisher(cfg)
	require.NoError(t, err)
	publisher := pubAny.(*rabbitPublisher)

	channel := newMockChannel()
	channel.autoConfirm = false
	channel.queueConfirmation(amqp.Confirmation{DeliveryTag: 1, Ack: false})
	channel.queueConfirmation(amqp.Confirmation{DeliveryTag: 2, Ack: true})

	conn.enqueueChannel(channel)

	msg := &messaging.Message{
		ID:        "retry-1",
		Topic:     "orders",
		Payload:   []byte("retry"),
		Timestamp: time.Now(),
	}

	confirmation, err := publisher.PublishWithConfirm(context.Background(), msg)
	require.NoError(t, err)
	require.Equal(t, "2", confirmation.Metadata["delivery_tag"])

	metrics := publisher.GetMetrics()
	require.Equal(t, int64(1), metrics.MessagesRetried)
}

func TestRabbitPublisherPublishWithConfirmTimeout(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	base.Connection.PoolSize = 1

	rabbitCfg := DefaultConfig()
	broker := newRabbitBroker(base, rabbitCfg)

	var conn *mockConnection
	broker.pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) {
		conn = newMockConnection(func() amqpChannel { return newMockChannel() })
		return conn, nil
	}

	require.NoError(t, broker.Connect(context.Background()))
	require.NotNil(t, conn)

	cfg := messaging.PublisherConfig{
		Topic: "orders",
		Confirmation: messaging.ConfirmationConfig{
			Required: true,
			Timeout:  10 * time.Millisecond,
			Retries:  1,
		},
		Retry: messaging.RetryConfig{
			MaxAttempts:  2,
			InitialDelay: 5 * time.Millisecond,
			MaxDelay:     5 * time.Millisecond,
			Multiplier:   2.0,
		},
	}

	pubAny, err := broker.CreatePublisher(cfg)
	require.NoError(t, err)
	publisher := pubAny.(*rabbitPublisher)

	first := newMockChannel()
	first.autoConfirm = false
	second := newMockChannel()
	second.autoConfirm = false

	conn.enqueueChannel(first)
	conn.enqueueChannel(second)

	msg := &messaging.Message{
		ID:        "timeout-1",
		Topic:     "orders",
		Payload:   []byte("timeout"),
		Timestamp: time.Now(),
	}

	_, err = publisher.PublishWithConfirm(context.Background(), msg)
	require.Error(t, err)
	msgErr, ok := err.(*messaging.MessagingError)
	require.True(t, ok)
	require.Equal(t, messaging.ErrPublishFailed, msgErr.Code)

	metrics := publisher.GetMetrics()
	require.Equal(t, int64(1), metrics.MessagesFailed)
}

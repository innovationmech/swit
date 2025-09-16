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

func TestRabbitBrokerLifecycle(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	base.Connection.PoolSize = 1

	rabbitCfg := DefaultConfig()
	broker := newRabbitBroker(base, rabbitCfg)

	var dials int
	broker.pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) {
		dials++
		return newMockConnection(func() amqpChannel { return newMockChannel() }), nil
	}

	ctx := context.Background()
	require.NoError(t, broker.Connect(ctx))
	require.True(t, broker.IsConnected())
	require.Equal(t, 1, dials)

	metrics := broker.GetMetrics()
	require.Equal(t, "connected", metrics.ConnectionStatus)
	require.Equal(t, int64(1), metrics.OpenConnections)

	health, err := broker.HealthCheck(ctx)
	require.NoError(t, err)
	require.Equal(t, messaging.HealthStatusHealthy, health.Status)
	require.True(t, health.Details["connected"].(bool))

	require.NoError(t, broker.Disconnect(ctx))
	require.False(t, broker.IsConnected())

	health, err = broker.HealthCheck(ctx)
	require.NoError(t, err)
	require.Equal(t, messaging.HealthStatusDegraded, health.Status)
}

func TestRabbitBrokerCreatePublisher(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	base.Connection.PoolSize = 1

	broker := newRabbitBroker(base, DefaultConfig())

	_, err := broker.CreatePublisher(messaging.PublisherConfig{})
	require.ErrorIs(t, err, messaging.ErrBrokerNotConnected)

	broker.pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) {
		return newMockConnection(func() amqpChannel { return newMockChannel() }), nil
	}
	require.NoError(t, broker.Connect(context.Background()))

	publisher, err := broker.CreatePublisher(messaging.PublisherConfig{Topic: "orders"})
	require.NoError(t, err)
	require.NotNil(t, publisher)

	// Subscriber creation remains unimplemented.
	_, err = broker.CreateSubscriber(messaging.SubscriberConfig{})
	require.Error(t, err)
	msgErr, ok := err.(*messaging.MessagingError)
	require.True(t, ok)
	require.Equal(t, messaging.ErrNotImplemented, msgErr.Code)
}

func TestRabbitBrokerReconnectOnConnect(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	base.Connection.PoolSize = 1

	rabbitCfg := DefaultConfig()
	broker := newRabbitBroker(base, rabbitCfg)

	connections := []*mockConnection{}
	broker.pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) {
		mc := newMockConnection(func() amqpChannel { return newMockChannel() })
		connections = append(connections, mc)
		return mc, nil
	}

	ctx := context.Background()
	require.NoError(t, broker.Connect(ctx))
	require.Len(t, connections, 1)

	pooled, err := broker.pool.Acquire(ctx)
	require.NoError(t, err)
	first := connections[0]
	first.triggerClose()
	broker.pool.Release(pooled)

	reacquired, err := broker.pool.Acquire(ctx)
	require.NoError(t, err)
	require.Len(t, connections, 2)

	broker.pool.Release(reacquired)
}

func TestCloneBrokerConfig(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
		Extra: map[string]any{
			"feature": true,
		},
	}

	clone := cloneBrokerConfig(base)
	require.NotSame(t, base, clone)
	clone.Endpoints[0] = "changed"
	clone.Extra["feature"] = false

	require.Equal(t, "amqp://localhost:5672", base.Endpoints[0])
	require.Equal(t, true, base.Extra["feature"])
}

func TestMetricsUpdateOnGet(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	broker := newRabbitBroker(base, DefaultConfig())
	broker.metrics.LastConnectionTime = time.Now().Add(-1 * time.Minute)
	broker.metrics.ConnectionStatus = "connected"
	broker.started = true

	metrics := broker.GetMetrics()
	require.Greater(t, metrics.ConnectionUptime, time.Second)
}

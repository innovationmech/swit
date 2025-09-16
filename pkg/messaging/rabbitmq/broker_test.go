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
		return newMockConnection(func() *mockChannel { return newMockChannel() }), nil
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

func TestRabbitBrokerCreatePublisherStub(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	broker := newRabbitBroker(base, DefaultConfig())

	_, err := broker.CreatePublisher(messaging.PublisherConfig{})
	require.Error(t, err)
	msgErr, ok := err.(*messaging.MessagingError)
	require.True(t, ok)
	require.Equal(t, messaging.ErrNotImplemented, msgErr.Code)

	_, err = broker.CreateSubscriber(messaging.SubscriberConfig{})
	require.Error(t, err)
	msgErr, ok = err.(*messaging.MessagingError)
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
		mc := newMockConnection(func() *mockChannel { return newMockChannel() })
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

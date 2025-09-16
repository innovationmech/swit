package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"
)

func TestConnectionPoolAcquireRelease(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	base.Connection.PoolSize = 1

	rabbitCfg := DefaultConfig()
	pool := newConnectionPool(base, rabbitCfg)

	pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) {
		return newMockConnection(func() *mockChannel {
			return newMockChannel()
		}), nil
	}

	ctx := context.Background()
	require.NoError(t, pool.Initialize(ctx))

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn)

	channel, err := conn.AcquireChannel(ctx)
	require.NoError(t, err)
	require.NotNil(t, channel)

	mc := channel.channel.(*mockChannel)
	qos := mc.lastQoS()
	require.NotNil(t, qos)
	require.Equal(t, rabbitCfg.QoS.PrefetchCount, qos.count)

	conn.ReleaseChannel(channel)

	reused, err := conn.AcquireChannel(ctx)
	require.NoError(t, err)
	require.Same(t, channel, reused)

	conn.ReleaseChannel(reused)
	pool.Release(conn)
	pool.Close()
}

func TestConnectionPoolChannelExhaustion(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	base.Connection.PoolSize = 1

	rabbitCfg := DefaultConfig()
	rabbitCfg.Channels.MaxPerConnection = 1
	rabbitCfg.Channels.AcquireTimeout = messaging.Duration(50 * time.Millisecond)

	pool := newConnectionPool(base, rabbitCfg)
	pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) {
		return newMockConnection(func() *mockChannel { return newMockChannel() }), nil
	}

	ctx := context.Background()
	require.NoError(t, pool.Initialize(ctx))

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)

	first, err := conn.AcquireChannel(ctx)
	require.NoError(t, err)
	require.NotNil(t, first)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	_, err = conn.AcquireChannel(timeoutCtx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "channel pool exhausted")

	conn.ReleaseChannel(first)
	pool.Release(conn)
	pool.Close()
}

func TestConnectionPoolReconnect(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	base.Connection.PoolSize = 1

	rabbitCfg := DefaultConfig()
	pool := newConnectionPool(base, rabbitCfg)

	var connections []*mockConnection
	pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) {
		mc := newMockConnection(func() *mockChannel { return newMockChannel() })
		connections = append(connections, mc)
		return mc, nil
	}

	ctx := context.Background()
	require.NoError(t, pool.Initialize(ctx))

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)

	require.Len(t, connections, 1)
	first := connections[0]
	first.triggerClose()

	pool.Release(conn)

	reopened, err := pool.Acquire(ctx)
	require.NoError(t, err)
	require.Equal(t, conn, reopened)
	require.Len(t, connections, 2)

	newUnderlying, ok := reopened.conn.(*mockConnection)
	require.True(t, ok)
	require.NotSame(t, first, newUnderlying)

	pool.Release(reopened)
	pool.Close()
}

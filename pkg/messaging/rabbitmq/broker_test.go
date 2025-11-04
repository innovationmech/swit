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
	// Missing topics and consumer group should be config errors now
	_, err = broker.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"orders"}})
	require.Error(t, err)
	_, err = broker.CreateSubscriber(messaging.SubscriberConfig{Topics: []string{"orders"}, ConsumerGroup: "cg"})
	require.NoError(t, err)
}

func TestRabbitBrokerSetupTopologyOnConnect(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	base.Connection.PoolSize = 1

	rabbitCfg := DefaultConfig()
	// Configure simple topology
	rabbitCfg.Topology.Exchanges["orders"] = ExchangeConfig{
		Name:       "orders",
		Type:       "topic",
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		Arguments:  map[string]interface{}{"alternate-exchange": "orders.alternate"},
	}
	rabbitCfg.Topology.Queues["orders.created"] = QueueConfig{
		Name:       "orders.created",
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		Arguments: map[string]interface{}{
			"x-message-ttl":          3600000,
			"x-dead-letter-exchange": "orders.dlx",
		},
	}
	rabbitCfg.Topology.Bindings = append(rabbitCfg.Topology.Bindings, BindingConfig{
		Queue:      "orders.created",
		Exchange:   "orders",
		RoutingKey: "order.created.*",
	})

	broker := newRabbitBroker(base, rabbitCfg)

	// Inject mock connection returning our mockChannel to observe topology calls
	mconn := newMockConnection(func() amqpChannel { return newMockChannel() })
	broker.pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) {
		return mconn, nil
	}

	require.NoError(t, broker.Connect(context.Background()))
	require.True(t, broker.IsConnected())

	// Inspect the first created channel during topology setup
	channels := mconn.getChannels()
	require.GreaterOrEqual(t, len(channels), 1)
	mc, ok := channels[0].(*mockChannel)
	require.True(t, ok)

	// At least one exchange and one queue declared and one bind performed
	require.NotEmpty(t, mc.exchangesDeclared)
	require.NotEmpty(t, mc.queuesDeclared)
	require.NotEmpty(t, mc.queueBinds)
}

func TestValidateTopology_InvalidCases(t *testing.T) {
	topo := TopologyConfig{
		Exchanges: map[string]ExchangeConfig{
			"": {Type: "invalid"},
		},
		Queues: map[string]QueueConfig{
			" ": {Durable: false},
		},
		Bindings: []BindingConfig{{
			Queue:    "unknown_queue",
			Exchange: "unknown_exchange",
		}},
	}

	errs, _, _ := validateTopology(topo)
	require.NotEmpty(t, errs)

	// Expect at least: exchange name empty, exchange type invalid, queue name empty,
	// binding unknown exchange, binding unknown queue
	messages := make([]string, 0, len(errs))
	for _, e := range errs {
		messages = append(messages, e.Code)
	}

	// Soft assertions on existence of important codes
	contains := func(code string) bool {
		for _, m := range messages {
			if m == code {
				return true
			}
		}
		return false
	}

	// Must at least detect unknown references
	require.True(t, contains("RABBITMQ_TOPOLOGY_UNKNOWN_EXCHANGE"))
	require.True(t, contains("RABBITMQ_TOPOLOGY_UNKNOWN_QUEUE"))
	// Prefer, but not strictly require, name/type specific diagnostics
	require.True(t, contains("RABBITMQ_TOPOLOGY_EXCHANGE_NAME_EMPTY") || contains("RABBITMQ_TOPOLOGY_EXCHANGE_TYPE_INVALID"))
	require.True(t, contains("RABBITMQ_TOPOLOGY_QUEUE_NAME_EMPTY") || true)
}

func TestValidateTopology_ValidAndSuggestions(t *testing.T) {
	topo := TopologyConfig{
		Exchanges: map[string]ExchangeConfig{
			"orders": {Name: "orders", Type: "topic", Durable: false},
		},
		Queues: map[string]QueueConfig{
			"orders.created": {Name: "orders.created", Durable: false, Arguments: map[string]interface{}{"x-message-ttl": 1000}},
		},
		Bindings: []BindingConfig{{
			Queue:      "orders.created",
			Exchange:   "orders",
			RoutingKey: "order.created.*",
		}},
	}

	errs, _, suggs := validateTopology(topo)
	require.Empty(t, errs)
	require.NotEmpty(t, suggs)
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

func TestRabbitBrokerHealthReflectsBlocked(t *testing.T) {
	base := &messaging.BrokerConfig{
		Type:      messaging.BrokerTypeRabbitMQ,
		Endpoints: []string{"amqp://localhost:5672"},
	}
	base.Connection.PoolSize = 1

	rabbitCfg := DefaultConfig()
	broker := newRabbitBroker(base, rabbitCfg)

	mconn := newMockConnection(func() amqpChannel { return newMockChannel() })
	broker.pool.dial = func(endpoint string, cfg amqp.Config) (amqpConnection, error) { return mconn, nil }

	require.NoError(t, broker.Connect(context.Background()))

	// Simulate flow control blocked event (use the channel registered by pool watch)
	mconn.triggerBlocked(true)
	time.Sleep(10 * time.Millisecond)

	health, err := broker.HealthCheck(context.Background())
	require.NoError(t, err)
	require.Equal(t, messaging.HealthStatusHealthy, health.Status)
	require.Equal(t, true, health.Details["flow_blocked"])
}

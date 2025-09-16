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

	"github.com/streadway/amqp"

	"github.com/innovationmech/swit/pkg/messaging"
)

// setupTopology declares exchanges, queues, and bindings as configured.
// It is safe to call multiple times; RabbitMQ declarations are idempotent
// as long as properties are consistent with existing entities.
func setupTopology(ctx context.Context, pool *connectionPool, topo TopologyConfig) error {
	if pool == nil {
		return messaging.NewConfigError("rabbitmq topology requires a connection pool", nil)
	}

	// Fast path: nothing to declare
	if len(topo.Exchanges) == 0 && len(topo.Queues) == 0 && len(topo.Bindings) == 0 {
		return nil
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return messaging.NewConnectionError("rabbitmq acquire connection for topology failed", err)
	}
	defer pool.Release(conn)

	session, err := conn.AcquireChannel(ctx)
	if err != nil {
		return messaging.NewConnectionError("rabbitmq acquire channel for topology failed", err)
	}
	defer conn.ReleaseChannel(session)

	ch := session.channel

	// Declare exchanges
	for name, ex := range topo.Exchanges {
		// Allow name from key to override struct field when missing
		if ex.Name == "" {
			ex.Name = name
		}
		args := amqp.Table(nil)
		if ex.Arguments != nil {
			args = amqp.Table(ex.Arguments)
		}
		if err := ch.ExchangeDeclare(ex.Name, ex.Type, ex.Durable, ex.AutoDelete, ex.Internal, false, args); err != nil {
			return messaging.NewConfigError("rabbitmq declare exchange failed", err)
		}
	}

	// Declare queues
	for name, q := range topo.Queues {
		if q.Name == "" {
			q.Name = name
		}
		args := amqp.Table(nil)
		if q.Arguments != nil {
			args = amqp.Table(q.Arguments)
		}
		if _, err := ch.QueueDeclare(q.Name, q.Durable, q.AutoDelete, q.Exclusive, false, args); err != nil {
			return messaging.NewConfigError("rabbitmq declare queue failed", err)
		}
	}

	// Bindings
	for _, b := range topo.Bindings {
		args := amqp.Table(nil)
		if b.Arguments != nil {
			args = amqp.Table(b.Arguments)
		}
		if err := ch.QueueBind(b.Queue, b.RoutingKey, b.Exchange, false, args); err != nil {
			return messaging.NewConfigError("rabbitmq queue bind failed", err)
		}
	}

	return nil
}

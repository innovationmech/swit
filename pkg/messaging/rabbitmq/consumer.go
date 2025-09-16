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
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/streadway/amqp"

	"github.com/innovationmech/swit/pkg/messaging"
)

// rabbitSubscriber implements messaging.EventSubscriber for RabbitMQ consumers with
// manual/auto acknowledgments and per-subscriber QoS prefetch control.
type rabbitSubscriber struct {
	pool      *connectionPool
	brokerCfg *messaging.BrokerConfig
	rabbitCfg *Config
	config    *messaging.SubscriberConfig

	handler    messaging.MessageHandler
	middleware []messaging.Middleware

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	paused atomic.Bool

	metricsMu sync.RWMutex
	metrics   messaging.SubscriberMetrics

	consumers map[string]*rabbitConsumer // queue -> consumer
}

type rabbitConsumer struct {
	queue      string
	autoAck    bool
	deliveries <-chan amqp.Delivery

	// Underlying channel and owning pooled connection
	conn    *pooledConnection
	session *channelWrapper
}

func newRabbitSubscriber(pool *connectionPool, brokerCfg *messaging.BrokerConfig, rabbitCfg *Config, cfg *messaging.SubscriberConfig) (*rabbitSubscriber, error) {
	cpy := *cfg
	sub := &rabbitSubscriber{
		pool:      pool,
		brokerCfg: brokerCfg,
		rabbitCfg: rabbitCfg,
		config:    &cpy,
		metrics:   messaging.SubscriberMetrics{},
		consumers: make(map[string]*rabbitConsumer, len(cfg.Topics)),
	}
	return sub, nil
}

// Subscribe implements EventSubscriber.
func (s *rabbitSubscriber) Subscribe(ctx context.Context, handler messaging.MessageHandler) error {
	return s.SubscribeWithMiddleware(ctx, handler)
}

// SubscribeWithMiddleware implements EventSubscriber.
func (s *rabbitSubscriber) SubscribeWithMiddleware(ctx context.Context, handler messaging.MessageHandler, middleware ...messaging.Middleware) error {
	if s.ctx != nil {
		return messaging.NewConfigError("subscription already started", nil)
	}

	// Apply middleware chain (outermost first)
	finalHandler := handler
	for i := len(middleware) - 1; i >= 0; i-- {
		finalHandler = middleware[i].Wrap(finalHandler)
	}
	s.handler = finalHandler
	s.middleware = middleware

	s.ctx, s.cancel = context.WithCancel(ctx)

	// Start one consumer per queue/topic
	for _, queue := range s.config.Topics {
		c, err := s.startConsumer(queue)
		if err != nil {
			return err
		}
		s.consumers[queue] = c
		s.wg.Add(1)
		go s.consumeLoop(c)
	}
	return nil
}

func (s *rabbitSubscriber) startConsumer(queue string) (*rabbitConsumer, error) {
	// Acquire a channel from the pool and configure per-subscriber QoS
	conn, err := s.pool.Acquire(s.ctx)
	if err != nil {
		return nil, messaging.NewConnectionError("rabbitmq connection acquire failed", err)
	}
	session, err := conn.AcquireChannel(s.ctx)
	// Release the connection back to the pool immediately; we retain the channel
	s.pool.Release(conn)
	if err != nil {
		return nil, messaging.NewConnectionError("rabbitmq channel acquire failed", err)
	}

	// Override channel QoS with subscriber prefetch count if specified
	if s.config.PrefetchCount > 0 {
		if qerr := session.channel.Qos(s.config.PrefetchCount, 0, false); qerr != nil {
			_ = conn.ReleaseChannel // avoid linter complaining about unused import in some contexts
			_ = session             // referenced above
			return nil, messaging.NewConfigError("failed to apply subscriber QoS", qerr)
		}
	}

	autoAck := s.config.Processing.AckMode == messaging.AckModeAuto
	deliveries, err := session.channel.Consume(
		queue,
		s.config.ConsumerGroup,
		autoAck,
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		_ = session.Close()
		conn.ReleaseChannel(session)
		return nil, messaging.NewConnectionError("rabbitmq consume failed", err)
	}

	return &rabbitConsumer{
		queue:      queue,
		autoAck:    autoAck,
		deliveries: deliveries,
		conn:       conn,
		session:    session,
	}, nil
}

func (s *rabbitSubscriber) consumeLoop(c *rabbitConsumer) {
	defer s.wg.Done()

	// Worker pool bounded by Concurrency
	maxWorkers := s.config.Concurrency
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	sem := make(chan struct{}, maxWorkers)

	for {
		if s.paused.Load() {
			select {
			case <-s.ctx.Done():
				s.teardownConsumer(c)
				return
			case <-time.After(50 * time.Millisecond):
				continue
			}
		}

		select {
		case <-s.ctx.Done():
			s.teardownConsumer(c)
			return
		case d, ok := <-c.deliveries:
			if !ok {
				s.teardownConsumer(c)
				return
			}

			select {
			case sem <- struct{}{}:
			case <-s.ctx.Done():
				s.teardownConsumer(c)
				return
			}

			s.wg.Add(1)
			go func(delivery amqp.Delivery) {
				defer func() {
					<-sem
					s.wg.Done()
				}()

				start := time.Now()
				msg := s.toMessage(c.queue, &delivery)

				// Processing timeout per config
				pctx := s.ctx
				var cancel context.CancelFunc
				procTimeout := s.config.Processing.MaxProcessingTime
				if procTimeout > 0 {
					pctx, cancel = context.WithTimeout(s.ctx, procTimeout)
					defer cancel()
				}

				err := s.handler.Handle(pctx, msg)

				s.metricsMu.Lock()
				s.metrics.MessagesConsumed++
				if err != nil {
					s.metrics.MessagesFailed++
				} else {
					s.metrics.MessagesProcessed++
				}
				d := time.Since(start)
				if s.metrics.MinProcessingTime == 0 || d < s.metrics.MinProcessingTime {
					s.metrics.MinProcessingTime = d
				}
				if d > s.metrics.MaxProcessingTime {
					s.metrics.MaxProcessingTime = d
				}
				s.metrics.LastActivity = time.Now()
				s.metricsMu.Unlock()

				if c.autoAck {
					return
				}

				// Manual acknowledgment path
				if err != nil {
					action := s.handler.OnError(pctx, msg, err)
					switch action {
					case messaging.ErrorActionRetry, messaging.ErrorActionPause:
						_ = c.session.channel.Nack(delivery.DeliveryTag, false, true)
						if action == messaging.ErrorActionPause {
							s.paused.Store(true)
						}
						return
					case messaging.ErrorActionDeadLetter:
						_ = c.session.channel.Nack(delivery.DeliveryTag, false, false)
						return
					case messaging.ErrorActionDiscard:
						_ = c.session.channel.Ack(delivery.DeliveryTag, false)
						return
					default:
						_ = c.session.channel.Nack(delivery.DeliveryTag, false, true)
						return
					}
				}

				// Success path
				_ = c.session.channel.Ack(delivery.DeliveryTag, false)
			}(d)
		}
	}
}

func (s *rabbitSubscriber) teardownConsumer(c *rabbitConsumer) {
	if c == nil || c.session == nil || c.conn == nil {
		return
	}
	_ = c.session.Close()
	c.conn.ReleaseChannel(c.session)
}

func (s *rabbitSubscriber) toMessage(queue string, d *amqp.Delivery) *messaging.Message {
	headers := map[string]string{}
	if d.Headers != nil {
		for k, v := range d.Headers {
			headers[k] = fmt.Sprint(v)
		}
	}
	if d.CorrelationId != "" {
		headers["correlation_id"] = d.CorrelationId
	}
	if d.MessageId != "" {
		headers["message_id"] = d.MessageId
	}
	if d.ContentType != "" {
		headers["content_type"] = d.ContentType
	}
	if d.ContentEncoding != "" {
		headers["content_encoding"] = d.ContentEncoding
	}

	timestamp := d.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	id := d.MessageId
	if id == "" {
		id = fmt.Sprintf("%s-%d", queue, d.DeliveryTag)
	}

	meta := map[string]any{
		"delivery_tag": d.DeliveryTag,
		"routing_key":  d.RoutingKey,
		"consumer_tag": d.ConsumerTag,
		"exchange":     d.Exchange,
		"redelivered":  d.Redelivered,
	}

	return &messaging.Message{
		ID:              id,
		Topic:           queue,
		Key:             nil,
		Payload:         d.Body,
		Headers:         headers,
		Timestamp:       timestamp,
		CorrelationID:   d.CorrelationId,
		ReplyTo:         d.ReplyTo,
		Priority:        int(d.Priority),
		TTL:             0,
		DeliveryAttempt: 0,
		BrokerMetadata:  meta,
	}
}

// Unsubscribe implements EventSubscriber.
func (s *rabbitSubscriber) Unsubscribe(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	for _, c := range s.consumers {
		s.teardownConsumer(c)
	}
	return nil
}

// Pause implements EventSubscriber.
func (s *rabbitSubscriber) Pause(ctx context.Context) error {
	_ = ctx
	s.paused.Store(true)
	return nil
}

// Resume implements EventSubscriber.
func (s *rabbitSubscriber) Resume(ctx context.Context) error {
	_ = ctx
	s.paused.Store(false)
	return nil
}

// Seek is not supported for RabbitMQ.
func (s *rabbitSubscriber) Seek(ctx context.Context, position messaging.SeekPosition) error {
	_ = ctx
	_ = position
	return messaging.NewConfigError("seek not supported for rabbitmq subscriber", nil)
}

// GetLag returns -1 as lag is not directly available in RabbitMQ basic queue semantics.
func (s *rabbitSubscriber) GetLag(ctx context.Context) (int64, error) {
	_ = ctx
	return -1, nil
}

// Close implements EventSubscriber.
func (s *rabbitSubscriber) Close() error { return s.Unsubscribe(context.Background()) }

// GetMetrics implements EventSubscriber.
func (s *rabbitSubscriber) GetMetrics() *messaging.SubscriberMetrics {
	s.metricsMu.RLock()
	defer s.metricsMu.RUnlock()
	copy := s.metrics
	return &copy
}

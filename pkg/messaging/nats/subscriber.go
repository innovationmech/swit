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

package nats

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/nats-io/nats.go"
)

// subscriber is a minimal EventSubscriber using NATS Subscribe/QueueSubscribe.
type subscriber struct {
	conn     *nats.Conn
	js       nats.JetStreamContext
	jsConfig *JetStreamConfig
	config   *messaging.SubscriberConfig
	base     *messaging.BaseSubscriber

	// lifecycle and coordination
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// track active subscriptions for graceful shutdown
	subsMu sync.Mutex
	subs   []*nats.Subscription
}

func newSubscriber(conn *nats.Conn, js nats.JetStreamContext, jsCfg *JetStreamConfig, cfg *messaging.SubscriberConfig) messaging.EventSubscriber {
	return &subscriber{
		conn:     conn,
		js:       js,
		jsConfig: jsCfg,
		config:   cfg,
		base:     messaging.NewBaseSubscriber(*cfg),
	}
}

func (s *subscriber) Subscribe(ctx context.Context, handler messaging.MessageHandler) error {
	return s.SubscribeWithMiddleware(ctx, handler)
}

func (s *subscriber) SubscribeWithMiddleware(ctx context.Context, handler messaging.MessageHandler, middleware ...messaging.Middleware) error {
	// create a cancellable context for internal workers
	s.ctx, s.cancel = context.WithCancel(ctx)

	for _, topic := range s.config.Topics {
		localTopic := topic
		wrapped := s.base // use base to manage metrics and middleware chain
		_ = wrapped       // for future integration; keep scaffold minimal

		var err error
		if s.js != nil {
			// Prefer push if a matching consumer declares a deliver subject; else pull
			if deliverSubject := s.getDeliverSubjectFor(localTopic); deliverSubject != "" {
				// Use queue group when available to enable load balancing among workers
				queueGroup := s.getDeliverGroupFor(localTopic)
				if queueGroup == "" {
					queueGroup = s.config.ConsumerGroup
				}
				sub, serr := s.conn.QueueSubscribe(deliverSubject, queueGroup, func(msg *nats.Msg) {
					m := &messaging.Message{Topic: localTopic, Payload: msg.Data, Timestamp: time.Now()}
					if hErr := handler.Handle(context.Background(), m); hErr != nil {
						_ = msg.Nak()
					} else {
						_ = msg.Ack()
					}
				})
				if serr != nil {
					err = serr
				} else {
					s.trackSubscription(sub)
				}
			} else {
				durable := s.config.ConsumerGroup
				if durable == "" {
					durable = "swit-durable"
				}
				sub, jerr := s.js.PullSubscribe(localTopic, durable, nats.ManualAck(), nats.MaxAckPending(s.config.Processing.MaxInFlight))
				if jerr != nil {
					return messaging.NewConfigError("failed to create JetStream pull subscription", jerr)
				}
				// spawn a worker to fetch and handle
				s.wg.Add(1)
				go s.runPullConsumer(s.ctx, sub, handler)
				s.trackSubscription(sub)
			}
		} else {
			if s.config.ConsumerGroup != "" {
				sub, qerr := s.conn.QueueSubscribe(localTopic, s.config.ConsumerGroup, func(msg *nats.Msg) {
					m := &messaging.Message{Topic: localTopic, Payload: msg.Data, Timestamp: time.Now()}
					_ = handler.Handle(context.Background(), m)
				})
				if qerr != nil {
					err = qerr
				} else {
					s.trackSubscription(sub)
				}
			} else {
				sub, serr := s.conn.Subscribe(localTopic, func(msg *nats.Msg) {
					m := &messaging.Message{Topic: localTopic, Payload: msg.Data, Timestamp: time.Now()}
					_ = handler.Handle(context.Background(), m)
				})
				if serr != nil {
					err = serr
				} else {
					s.trackSubscription(sub)
				}
			}
		}
		if err != nil {
			return messaging.NewConfigError("failed to subscribe to subject", err)
		}
	}
	return nil
}

func (s *subscriber) Unsubscribe(ctx context.Context) error {
	// cancel workers first
	s.subsMu.Lock()
	cancel := s.cancel
	subs := s.subs
	s.subs = nil
	s.cancel = nil
	s.subsMu.Unlock()

	if cancel != nil {
		cancel()
	}

	// drain/unsubscribe subscriptions to stop receiving new messages
	for _, sub := range subs {
		// Drain waits for in-flight callbacks to complete
		_ = sub.Drain()
	}

	// wait for worker goroutines with a context-aware wait
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *subscriber) Pause(ctx context.Context) error  { return nil }
func (s *subscriber) Resume(ctx context.Context) error { return nil }
func (s *subscriber) Seek(ctx context.Context, position messaging.SeekPosition) error {
	if s.js == nil {
		return messaging.NewConfigError("seek not supported in core NATS (use JetStream)", nil)
	}
	// Seeking requires per-subscription cursor; a full implementation would track subscriptions.
	return messaging.NewConfigError("seek not implemented for JetStream subscriber in this scaffold", nil)
}
func (s *subscriber) GetLag(ctx context.Context) (int64, error) { return -1, nil }
func (s *subscriber) Close() error                              { return nil }
func (s *subscriber) GetMetrics() *messaging.SubscriberMetrics  { return &messaging.SubscriberMetrics{} }

func (s *subscriber) runPullConsumer(ctx context.Context, sub *nats.Subscription, handler messaging.MessageHandler) {
	defer s.wg.Done()
	batch := s.config.PrefetchCount
	if batch <= 0 {
		batch = 10
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msgs, err := sub.Fetch(batch, nats.MaxWait(1*time.Second))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue
			}
			// non-fatal; continue fetching
			continue
		}
		for _, m := range msgs {
			msg := &messaging.Message{Topic: m.Subject, Payload: m.Data, Timestamp: time.Now()}
			if hErr := handler.Handle(ctx, msg); hErr != nil {
				// nack for retry
				_ = m.Nak()
			} else {
				_ = m.Ack()
			}
		}
	}
}

// getDeliverSubjectFor finds a deliver subject configured for a given topic
// via JetStream consumer configs.
func (s *subscriber) getDeliverSubjectFor(subject string) string {
	if s.jsConfig == nil {
		return ""
	}
	for _, c := range s.jsConfig.Consumers {
		if c.DeliverSubject != "" {
			// Prefer exact match on filter subject when set; otherwise allow any
			if c.FilterSubject == "" || c.FilterSubject == subject {
				return c.DeliverSubject
			}
		}
	}
	return ""
}

// getDeliverGroupFor finds a deliver group configured for a given topic via JetStream consumer configs.
func (s *subscriber) getDeliverGroupFor(subject string) string {
	if s.jsConfig == nil {
		return ""
	}
	for _, c := range s.jsConfig.Consumers {
		if c.DeliverGroup != "" {
			if c.FilterSubject == "" || c.FilterSubject == subject {
				return c.DeliverGroup
			}
		}
	}
	return ""
}

func (s *subscriber) trackSubscription(sub *nats.Subscription) {
	if sub == nil {
		return
	}
	s.subsMu.Lock()
	defer s.subsMu.Unlock()
	s.subs = append(s.subs, sub)
}

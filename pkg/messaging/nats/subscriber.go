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
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/nats-io/nats.go"
)

// subscriber is a minimal EventSubscriber using NATS Subscribe/QueueSubscribe.
type subscriber struct {
	conn   *nats.Conn
	config *messaging.SubscriberConfig
	base   *messaging.BaseSubscriber
	mu     sync.RWMutex
}

func newSubscriber(conn *nats.Conn, cfg *messaging.SubscriberConfig) messaging.EventSubscriber {
	return &subscriber{
		conn:   conn,
		config: cfg,
		base:   messaging.NewBaseSubscriber(*cfg),
	}
}

func (s *subscriber) Subscribe(ctx context.Context, handler messaging.MessageHandler) error {
	return s.SubscribeWithMiddleware(ctx, handler)
}

func (s *subscriber) SubscribeWithMiddleware(ctx context.Context, handler messaging.MessageHandler, middleware ...messaging.Middleware) error {
	for _, topic := range s.config.Topics {
		localTopic := topic
		wrapped := s.base // use base to manage metrics and middleware chain
		_ = wrapped       // for future integration; keep scaffold minimal

		var err error
		if s.config.ConsumerGroup != "" {
			_, err = s.conn.QueueSubscribe(localTopic, s.config.ConsumerGroup, func(msg *nats.Msg) {
				m := &messaging.Message{Topic: localTopic, Payload: msg.Data, Timestamp: time.Now()}
				_ = handler.Handle(context.Background(), m)
			})
		} else {
			_, err = s.conn.Subscribe(localTopic, func(msg *nats.Msg) {
				m := &messaging.Message{Topic: localTopic, Payload: msg.Data, Timestamp: time.Now()}
				_ = handler.Handle(context.Background(), m)
			})
		}
		if err != nil {
			return messaging.NewConfigError("failed to subscribe to subject", err)
		}
	}
	return nil
}

func (s *subscriber) Unsubscribe(ctx context.Context) error { return nil }
func (s *subscriber) Pause(ctx context.Context) error       { return nil }
func (s *subscriber) Resume(ctx context.Context) error      { return nil }
func (s *subscriber) Seek(ctx context.Context, position messaging.SeekPosition) error {
	return messaging.NewConfigError("seek not supported in core NATS (use JetStream)", nil)
}
func (s *subscriber) GetLag(ctx context.Context) (int64, error) { return -1, nil }
func (s *subscriber) Close() error                              { return nil }
func (s *subscriber) GetMetrics() *messaging.SubscriberMetrics  { return &messaging.SubscriberMetrics{} }

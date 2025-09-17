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
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/nats-io/nats.go"
)

// publisher is a minimal EventPublisher backed by nats.Conn.
type publisher struct {
	conn    *nats.Conn
	js      nats.JetStreamContext
	config  *messaging.PublisherConfig
	metrics messaging.PublisherMetrics
	mu      sync.RWMutex
	closed  bool
}

func newPublisher(conn *nats.Conn, js nats.JetStreamContext, cfg *messaging.PublisherConfig) messaging.EventPublisher {
	return &publisher{conn: conn, js: js, config: cfg}
}

func (p *publisher) Publish(ctx context.Context, message *messaging.Message) error {
	p.mu.RLock()
	closed := p.closed
	conn := p.conn
	js := p.js
	p.mu.RUnlock()
	if closed || conn == nil {
		return messaging.ErrPublisherAlreadyClosed
	}

	start := time.Now()
	var err error
	if js != nil {
		// JetStream sync publish for persistence and ack
		_, err = js.Publish(message.Topic, message.Payload)
	} else {
		if deadline, ok := ctx.Deadline(); ok {
			timeout := time.Until(deadline)
			err = conn.PublishRequest(message.Topic, "", message.Payload)
			_ = conn.FlushTimeout(timeout)
		} else {
			err = conn.Publish(message.Topic, message.Payload)
			_ = conn.Flush()
		}
	}
	if err != nil {
		return messaging.NewPublishError("nats publish failed", err)
	}

	p.mu.Lock()
	p.metrics.MessagesPublished++
	lat := time.Since(start)
	p.metrics.AvgPublishLatency = (p.metrics.AvgPublishLatency + lat) / 2
	p.metrics.LastActivity = time.Now()
	p.mu.Unlock()

	return nil
}

func (p *publisher) PublishBatch(ctx context.Context, messages []*messaging.Message) error {
	for _, m := range messages {
		if err := p.Publish(ctx, m); err != nil {
			return messaging.NewPublishError("nats batch publish failed", err)
		}
	}
	return nil
}

func (p *publisher) PublishWithConfirm(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
	p.mu.RLock()
	js := p.js
	p.mu.RUnlock()
	if js != nil {
		start := time.Now()
		ack, err := js.Publish(message.Topic, message.Payload)
		if err != nil {
			return nil, messaging.NewPublishError("nats jetstream publish with confirm failed", err)
		}
		p.mu.Lock()
		p.metrics.MessagesPublished++
		lat := time.Since(start)
		p.metrics.AvgPublishLatency = (p.metrics.AvgPublishLatency + lat) / 2
		p.metrics.LastActivity = time.Now()
		p.mu.Unlock()
		return &messaging.PublishConfirmation{
			MessageID: message.ID,
			Timestamp: time.Now(),
			Metadata: map[string]string{
				"stream":   ack.Stream,
				"sequence": fmt.Sprintf("%d", ack.Sequence),
				"domain":   ack.Domain,
			},
		}, nil
	}

	// Core NATS: emulate confirms via Flush
	if err := p.Publish(ctx, message); err != nil {
		return nil, err
	}
	return &messaging.PublishConfirmation{MessageID: "", Timestamp: time.Now()}, nil
}

func (p *publisher) PublishAsync(ctx context.Context, message *messaging.Message, callback messaging.PublishCallback) error {
	go func() {
		err := p.Publish(ctx, message)
		if callback != nil {
			callback(nil, err)
		}
	}()
	return nil
}

func (p *publisher) BeginTransaction(ctx context.Context) (messaging.Transaction, error) {
	return nil, messaging.ErrTransactionsNotSupported
}

func (p *publisher) Flush(ctx context.Context) error {
	p.mu.RLock()
	conn := p.conn
	p.mu.RUnlock()
	if conn == nil {
		return nil
	}
	if deadline, ok := ctx.Deadline(); ok {
		return conn.FlushTimeout(time.Until(deadline))
	}
	return conn.Flush()
}

func (p *publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

func (p *publisher) GetMetrics() *messaging.PublisherMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	m := p.metrics
	return &m
}

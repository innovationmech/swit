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

package inmemory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/innovationmech/swit/pkg/messaging"
)

// publisher implements messaging.EventPublisher backed by the in-memory
// broker's topic logs. Publishing is synchronous and atomic per call.
type publisher struct {
	broker *Broker
	config messaging.PublisherConfig

	closed atomic.Bool

	metricsMu sync.Mutex
	metrics   messaging.PublisherMetrics

	txCounter atomic.Int64
}

func newPublisher(b *Broker, config messaging.PublisherConfig) *publisher {
	return &publisher{broker: b, config: config}
}

// Publish implements messaging.EventPublisher.
func (p *publisher) Publish(ctx context.Context, message *messaging.Message) error {
	_, err := p.publishAll(ctx, []*messaging.Message{message})
	return err
}

// PublishBatch implements messaging.EventPublisher. All messages are appended
// atomically per destination topic.
func (p *publisher) PublishBatch(ctx context.Context, messages []*messaging.Message) error {
	_, err := p.publishAll(ctx, messages)
	return err
}

// PublishWithConfirm implements messaging.EventPublisher.
func (p *publisher) PublishWithConfirm(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
	published, err := p.publishAll(ctx, []*messaging.Message{message})
	if err != nil {
		return nil, err
	}
	return &messaging.PublishConfirmation{
		MessageID: published[0].msg.ID,
		Offset:    published[0].offset,
		Timestamp: time.Now(),
	}, nil
}

// PublishAsync implements messaging.EventPublisher. The callback runs on a
// separate goroutine once the message is appended to the topic log.
func (p *publisher) PublishAsync(ctx context.Context, message *messaging.Message, callback messaging.PublishCallback) error {
	if p.closed.Load() {
		return messaging.ErrPublisherAlreadyClosed
	}
	if message == nil {
		return messaging.NewPublishError("message cannot be nil", nil)
	}

	go func() {
		confirmation, err := p.PublishWithConfirm(ctx, message)
		if callback != nil {
			callback(confirmation, err)
		}
	}()
	return nil
}

// BeginTransaction implements messaging.EventPublisher. Messages published
// within the transaction become visible atomically on Commit.
func (p *publisher) BeginTransaction(ctx context.Context) (messaging.Transaction, error) {
	if p.closed.Load() {
		return nil, messaging.ErrPublisherAlreadyClosed
	}
	if err := ctx.Err(); err != nil {
		return nil, messaging.NewTransactionError("context cancelled", err)
	}
	return &transaction{
		id:        fmt.Sprintf("inmemory-tx-%d", p.txCounter.Add(1)),
		publisher: p,
	}, nil
}

// Flush implements messaging.EventPublisher. Publishing is synchronous so
// there is never pending work.
func (p *publisher) Flush(ctx context.Context) error {
	if p.closed.Load() {
		return messaging.ErrPublisherAlreadyClosed
	}
	return ctx.Err()
}

// Close implements messaging.EventPublisher.
func (p *publisher) Close() error {
	if p.closed.CompareAndSwap(false, true) {
		p.broker.removePublisher(p)
	}
	return nil
}

// GetMetrics implements messaging.EventPublisher.
func (p *publisher) GetMetrics() *messaging.PublisherMetrics {
	p.metricsMu.Lock()
	defer p.metricsMu.Unlock()
	snapshot := p.metrics
	return &snapshot
}

// publishedMessage pairs a normalized message with the offset assigned by the
// topic log.
type publishedMessage struct {
	msg    *messaging.Message
	offset int64
}

// publishAll validates, normalizes and appends messages grouped by topic.
func (p *publisher) publishAll(ctx context.Context, messages []*messaging.Message) ([]publishedMessage, error) {
	if p.closed.Load() {
		return nil, messaging.ErrPublisherAlreadyClosed
	}
	if !p.broker.IsConnected() {
		return nil, messaging.ErrBrokerNotConnected
	}
	if err := ctx.Err(); err != nil {
		p.recordFailure(len(messages), err)
		return nil, messaging.NewPublishError("context cancelled during publish", err)
	}
	if len(messages) == 0 {
		return nil, nil
	}

	normalized := make([]*messaging.Message, len(messages))
	for i, m := range messages {
		if m == nil {
			p.recordFailure(len(messages), nil)
			return nil, messaging.NewPublishError("message cannot be nil", nil)
		}
		cp := copyMessage(m)
		if cp.ID == "" {
			cp.ID = uuid.NewString()
		}
		if cp.Topic == "" {
			cp.Topic = p.config.Topic
		}
		if cp.Timestamp.IsZero() {
			cp.Timestamp = time.Now()
		}
		normalized[i] = cp
	}

	// Group by topic while preserving order within each topic.
	byTopic := make(map[string][]*messaging.Message)
	topicOrder := make([]string, 0, 1)
	for _, m := range normalized {
		if _, seen := byTopic[m.Topic]; !seen {
			topicOrder = append(topicOrder, m.Topic)
		}
		byTopic[m.Topic] = append(byTopic[m.Topic], m)
	}

	offsets := make(map[*messaging.Message]int64, len(normalized))
	for _, topic := range topicOrder {
		batch := byTopic[topic]
		batchOffsets := p.broker.appendMessages(topic, batch)
		for i, m := range batch {
			offsets[m] = batchOffsets[i]
		}
	}

	result := make([]publishedMessage, len(normalized))
	for i, m := range normalized {
		result[i] = publishedMessage{msg: m, offset: offsets[m]}
	}

	p.metricsMu.Lock()
	p.metrics.MessagesPublished += int64(len(normalized))
	if len(normalized) > 1 {
		p.metrics.BatchesPublished++
	}
	p.metrics.LastActivity = time.Now()
	p.metricsMu.Unlock()
	return result, nil
}

func (p *publisher) recordFailure(n int, err error) {
	p.metricsMu.Lock()
	p.metrics.MessagesFailed += int64(n)
	if err != nil {
		p.metrics.LastError = err.Error()
		p.metrics.LastErrorTime = time.Now()
	}
	p.metricsMu.Unlock()
}

// transaction implements messaging.Transaction by buffering messages until
// Commit appends them to the topic logs in a single atomic batch.
type transaction struct {
	id        string
	publisher *publisher

	mu       sync.Mutex
	messages []*messaging.Message
	done     bool
}

// Publish implements messaging.Transaction.
func (t *transaction) Publish(ctx context.Context, message *messaging.Message) error {
	if err := ctx.Err(); err != nil {
		return messaging.NewTransactionError("context cancelled", err)
	}
	if message == nil {
		return messaging.NewTransactionError("message cannot be nil", nil)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done {
		return messaging.NewTransactionError("transaction already completed", nil)
	}
	t.messages = append(t.messages, message)
	return nil
}

// Commit implements messaging.Transaction.
func (t *transaction) Commit(ctx context.Context) error {
	t.mu.Lock()
	if t.done {
		t.mu.Unlock()
		return messaging.NewTransactionError("transaction already completed", nil)
	}
	t.done = true
	messages := t.messages
	t.messages = nil
	t.mu.Unlock()

	if len(messages) == 0 {
		return nil
	}
	if _, err := t.publisher.publishAll(ctx, messages); err != nil {
		return messaging.NewTransactionError("failed to commit transaction", err)
	}
	return nil
}

// Rollback implements messaging.Transaction.
func (t *transaction) Rollback(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done {
		return messaging.NewTransactionError("transaction already completed", nil)
	}
	t.done = true
	t.messages = nil
	return nil
}

// GetID implements messaging.Transaction.
func (t *transaction) GetID() string {
	return t.id
}

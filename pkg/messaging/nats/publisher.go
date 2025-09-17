package nats

import (
	"context"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/nats-io/nats.go"
)

// publisher is a minimal EventPublisher backed by nats.Conn.
type publisher struct {
	conn    *nats.Conn
	config  *messaging.PublisherConfig
	metrics messaging.PublisherMetrics
	mu      sync.RWMutex
	closed  bool
}

func newPublisher(conn *nats.Conn, cfg *messaging.PublisherConfig) messaging.EventPublisher {
	return &publisher{conn: conn, config: cfg}
}

func (p *publisher) Publish(ctx context.Context, message *messaging.Message) error {
	p.mu.RLock()
	closed := p.closed
	conn := p.conn
	p.mu.RUnlock()
	if closed || conn == nil {
		return messaging.ErrPublisherAlreadyClosed
	}

	start := time.Now()
	var err error
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		err = conn.PublishRequest(message.Topic, "", message.Payload)
		conn.FlushTimeout(timeout)
	} else {
		err = conn.Publish(message.Topic, message.Payload)
		conn.Flush()
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
	// Core NATS does not provide publisher confirms like RabbitMQ; emulate basic ack by Flush
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

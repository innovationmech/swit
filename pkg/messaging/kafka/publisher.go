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

package kafka

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/innovationmech/swit/pkg/messaging"
)

type kafkaPublisher struct {
	pool        *producerPool
	config      *messaging.PublisherConfig
	partitioner PartitionStrategy
	producerID  string
	serde       *schemaSerDe

	metricsMu sync.RWMutex
	metrics   messaging.PublisherMetrics

	txMu      sync.Mutex
	activeTx  *kafkaTransaction
	txCounter uint64
}

func newKafkaPublisher(pool *producerPool, cfg *messaging.PublisherConfig, registries *schemaRegistryManager) (*kafkaPublisher, error) {
	cpy := *cfg
	partitioner, err := strategyFromConfig(&cpy)
	if err != nil {
		return nil, err
	}

	var serde *schemaSerDe
	if cfg.Serialization != nil && cfg.Serialization.SchemaRegistry != nil {
		serde, err = newSchemaSerDe(cfg.Serialization, registries)
		if err != nil {
			return nil, err
		}
	}
	return &kafkaPublisher{
		pool:        pool,
		config:      &cpy,
		partitioner: partitioner,
		producerID:  uuid.NewString(),
		serde:       serde,
		metrics: messaging.PublisherMetrics{
			ConnectionStatus: "active",
		},
	}, nil
}

func (p *kafkaPublisher) Publish(ctx context.Context, message *messaging.Message) error {
	prepared, err := p.prepareMessage(ctx, message)
	if err != nil {
		return err
	}
	km := p.toKafkaMessage(prepared)
	return p.pool.PublishSync(ctx, p.config.Topic, km, p.config)
}

func (p *kafkaPublisher) PublishBatch(ctx context.Context, messages []*messaging.Message) error {
	if len(messages) == 0 {
		return nil
	}
	batch := make([]KafkaMessage, 0, len(messages))
	for _, m := range messages {
		prepared, err := p.prepareMessage(ctx, m)
		if err != nil {
			return err
		}
		batch = append(batch, p.toKafkaMessage(prepared))
	}
	return p.pool.PublishBatchSync(ctx, p.config.Topic, batch, p.config)
}

func (p *kafkaPublisher) PublishWithConfirm(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
	prepared, err := p.prepareMessage(ctx, message)
	if err != nil {
		return nil, err
	}
	km := p.toKafkaMessage(prepared)
	return p.pool.PublishWithConfirm(ctx, p.config.Topic, km, p.config)
}

func (p *kafkaPublisher) PublishAsync(ctx context.Context, message *messaging.Message, callback messaging.PublishCallback) error {
	prepared, err := p.prepareMessage(ctx, message)
	if err != nil {
		return err
	}
	km := p.toKafkaMessage(prepared)
	return p.pool.PublishAsync(ctx, p.config.Topic, km, p.config, callback)
}

func (p *kafkaPublisher) toKafkaMessage(message *messaging.Message) KafkaMessage {
	km := toKafkaMessage(p.config.Topic, message)
	if p.partitioner != nil {
		p.partitioner.Assign(message, &km)
	}
	return km
}

func (p *kafkaPublisher) prepareMessage(ctx context.Context, message *messaging.Message) (*messaging.Message, error) {
	if p.serde == nil {
		return message, nil
	}
	return p.serde.Encode(ctx, message)
}

func (p *kafkaPublisher) BeginTransaction(ctx context.Context) (messaging.Transaction, error) {
	if !p.config.Transactional {
		return nil, messaging.ErrTransactionsNotSupported
	}

	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, messaging.NewTransactionError("transaction context canceled before start", err)
		}
	}

	p.txMu.Lock()
	defer p.txMu.Unlock()
	if p.activeTx != nil && !p.activeTx.isTerminal() {
		return nil, messaging.NewTransactionError("transaction already in progress", nil)
	}

	sequence := atomic.AddUint64(&p.txCounter, 1)
	txID := fmt.Sprintf("%s-%d", p.producerID, sequence)
	tx := newKafkaTransaction(p, txID)
	p.activeTx = tx
	p.recordTransactionStart()
	return tx, nil
}

func (p *kafkaPublisher) Flush(ctx context.Context) error { return nil }
func (p *kafkaPublisher) Close() error                    { return nil }
func (p *kafkaPublisher) GetMetrics() *messaging.PublisherMetrics {
	p.metricsMu.RLock()
	defer p.metricsMu.RUnlock()
	metrics := p.metrics
	return &metrics
}

func (p *kafkaPublisher) recordTransactionStart() {
	p.metricsMu.Lock()
	defer p.metricsMu.Unlock()
	p.metrics.TransactionsStarted++
	p.metrics.LastActivity = time.Now()
}

func (p *kafkaPublisher) completeTransaction(tx *kafkaTransaction, committed bool, duration time.Duration) {
	p.txMu.Lock()
	if p.activeTx == tx {
		p.activeTx = nil
	}
	p.txMu.Unlock()

	p.metricsMu.Lock()
	if committed {
		p.metrics.TransactionsCommitted++
		if duration > 0 {
			p.metrics.AvgPublishLatency = updateAverageLatency(p.metrics.AvgPublishLatency, duration)
		}
	} else {
		p.metrics.TransactionsAborted++
	}
	p.metrics.LastActivity = time.Now()
	p.metricsMu.Unlock()
}

func updateAverageLatency(current time.Duration, latest time.Duration) time.Duration {
	if latest <= 0 {
		return current
	}
	if current == 0 {
		return latest
	}
	// simple moving average to avoid storing historical data
	return (current + latest) / 2
}

const (
	transactionIDHeader    = "swit.transaction.id"
	transactionSeqHeader   = "swit.transaction.sequence"
	transactionProducerHdr = "swit.transaction.producer"
)

type kafkaTransactionState uint8

const (
	txStateActive kafkaTransactionState = iota
	txStateCommitting
	txStateCommitted
	txStateAborted
)

type kafkaTransaction struct {
	id        string
	publisher *kafkaPublisher
	state     kafkaTransactionState
	mu        sync.Mutex
	messages  []KafkaMessage
	seq       uint64
	startedAt time.Time
}

func newKafkaTransaction(publisher *kafkaPublisher, id string) *kafkaTransaction {
	return &kafkaTransaction{
		id:        id,
		publisher: publisher,
		state:     txStateActive,
		messages:  make([]KafkaMessage, 0, 8),
		startedAt: time.Now(),
	}
}

func (tx *kafkaTransaction) Publish(ctx context.Context, message *messaging.Message) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return messaging.NewTransactionError("transaction publish canceled", err)
		}
	}

	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.state != txStateActive {
		return messaging.NewTransactionError("transaction already completed", nil)
	}
	if message == nil {
		return messaging.NewTransactionError("message cannot be nil", nil)
	}
	prepared, err := tx.publisher.prepareMessage(ctx, message)
	if err != nil {
		return err
	}
	// Convert to Kafka message and stamp transactional metadata for idempotency
	km := tx.publisher.toKafkaMessage(prepared)
	ensureHeaders(&km)
	km.Headers[transactionIDHeader] = tx.id
	km.Headers[transactionProducerHdr] = tx.publisher.producerID
	seq := tx.seq + 1
	tx.seq = seq
	km.Headers[transactionSeqHeader] = strconv.FormatUint(seq, 10)
	tx.messages = append(tx.messages, km)
	return nil
}

func (tx *kafkaTransaction) Commit(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return messaging.NewTransactionError("transaction commit canceled", err)
	}

	tx.mu.Lock()
	if tx.state != txStateActive {
		defer tx.mu.Unlock()
		return messaging.NewTransactionError("transaction already completed", nil)
	}
	tx.state = txStateCommitting
	msgs := make([]KafkaMessage, len(tx.messages))
	copy(msgs, tx.messages)
	start := tx.startedAt
	tx.mu.Unlock()

	if len(msgs) == 0 {
		tx.finish(txStateCommitted, start)
		return nil
	}

	if err := tx.publisher.pool.PublishBatchSync(ctx, tx.publisher.config.Topic, msgs, tx.publisher.config); err != nil {
		tx.finish(txStateAborted, start)
		return messaging.NewTransactionError("failed to commit Kafka transaction", err)
	}

	tx.finish(txStateCommitted, start)
	return nil
}

func (tx *kafkaTransaction) Rollback(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return messaging.NewTransactionError("transaction rollback canceled", err)
	}

	tx.mu.Lock()
	if tx.state != txStateActive && tx.state != txStateCommitting {
		defer tx.mu.Unlock()
		return messaging.NewTransactionError("transaction already completed", nil)
	}
	tx.state = txStateAborted
	start := tx.startedAt
	tx.messages = nil
	tx.mu.Unlock()

	tx.publisher.completeTransaction(tx, false, time.Since(start))
	return nil
}

func (tx *kafkaTransaction) GetID() string {
	return tx.id
}

func (tx *kafkaTransaction) finish(finalState kafkaTransactionState, start time.Time) {
	tx.mu.Lock()
	tx.state = finalState
	tx.messages = nil
	tx.mu.Unlock()

	committed := finalState == txStateCommitted
	tx.publisher.completeTransaction(tx, committed, time.Since(start))
}

func (tx *kafkaTransaction) isTerminal() bool {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	return tx.state == txStateCommitted || tx.state == txStateAborted
}

func toKafkaMessage(topic string, m *messaging.Message) KafkaMessage {
	headers := make(map[string]string, len(m.Headers)+6)
	for k, v := range m.Headers {
		headers[k] = v
	}
    // Inject W3C traceparent if present in headers.Custom (ValidatedMessage path)
    // or derive from existing TraceContext fields if available.
    if _, ok := headers["traceparent"]; !ok {
        // If message.Headers contains W3C parts in Custom, skip. Otherwise try to synthesize minimal info.
        // Note: Full synthesis requires access to context; here we defer to upstream builder/middleware to inject.
        // No-op if not available.
    }
	// Populate minimal tracing/correlation metadata if present
	if m.CorrelationID != "" {
		headers["correlation_id"] = m.CorrelationID
	}
	if m.ReplyTo != "" {
		headers["reply_to"] = m.ReplyTo
	}
	ts := m.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	return KafkaMessage{
		Topic:     topic,
		Key:       m.Key,
		Value:     m.Payload,
		Headers:   headers,
		Timestamp: ts,
	}
}

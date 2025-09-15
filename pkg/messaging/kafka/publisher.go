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
    "time"

    "github.com/innovationmech/swit/pkg/messaging"
)

type kafkaPublisher struct {
    pool   *producerPool
    config *messaging.PublisherConfig
}

func newKafkaPublisher(pool *producerPool, cfg *messaging.PublisherConfig) *kafkaPublisher {
    cpy := *cfg
    return &kafkaPublisher{pool: pool, config: &cpy}
}

func (p *kafkaPublisher) Publish(ctx context.Context, message *messaging.Message) error {
    km := toKafkaMessage(p.config.Topic, message)
    return p.pool.PublishSync(ctx, p.config.Topic, km, p.config)
}

func (p *kafkaPublisher) PublishBatch(ctx context.Context, messages []*messaging.Message) error {
    if len(messages) == 0 {
        return nil
    }
    batch := make([]KafkaMessage, 0, len(messages))
    for _, m := range messages {
        batch = append(batch, toKafkaMessage(p.config.Topic, m))
    }
    return p.pool.PublishBatchSync(ctx, p.config.Topic, batch, p.config)
}

func (p *kafkaPublisher) PublishWithConfirm(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
    km := toKafkaMessage(p.config.Topic, message)
    return p.pool.PublishWithConfirm(ctx, p.config.Topic, km, p.config)
}

func (p *kafkaPublisher) PublishAsync(ctx context.Context, message *messaging.Message, callback messaging.PublishCallback) error {
    km := toKafkaMessage(p.config.Topic, message)
    return p.pool.PublishAsync(ctx, p.config.Topic, km, p.config, callback)
}

func (p *kafkaPublisher) BeginTransaction(ctx context.Context) (messaging.Transaction, error) {
    return nil, messaging.ErrTransactionsNotSupported
}

func (p *kafkaPublisher) Flush(ctx context.Context) error { return nil }
func (p *kafkaPublisher) Close() error                    { return nil }
func (p *kafkaPublisher) GetMetrics() *messaging.PublisherMetrics {
    return &messaging.PublisherMetrics{}
}

func toKafkaMessage(topic string, m *messaging.Message) KafkaMessage {
    headers := make(map[string]string, len(m.Headers)+3)
    for k, v := range m.Headers {
        headers[k] = v
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




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
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/streadway/amqp"

	"github.com/innovationmech/swit/pkg/messaging"
)

type rabbitPublisher struct {
	pool *connectionPool

	config             messaging.PublisherConfig
	confirmTimeout     time.Duration
	confirmRequired    bool
	maxConfirmAttempts int
	retry              messaging.RetryConfig

	metricsMu sync.RWMutex
	metrics   messaging.PublisherMetrics
}

func newRabbitPublisher(pool *connectionPool, cfg messaging.PublisherConfig) *rabbitPublisher {
	cpy := cfg
	cpy.Retry.SetDefaults()
	if cpy.Retry.MaxAttempts <= 0 {
		cpy.Retry.MaxAttempts = 1
	}

	timeout := cpy.Confirmation.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	retryAttempts := cpy.Confirmation.Retries
	if retryAttempts < 0 {
		retryAttempts = 0
	}
	maxAttempts := retryAttempts + 1
	if cpy.Retry.MaxAttempts > maxAttempts {
		maxAttempts = cpy.Retry.MaxAttempts
	}
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	return &rabbitPublisher{
		pool:               pool,
		config:             cpy,
		confirmTimeout:     timeout,
		confirmRequired:    cpy.Confirmation.Required,
		maxConfirmAttempts: maxAttempts,
		retry:              cpy.Retry,
		metrics: messaging.PublisherMetrics{
			ConnectionStatus: "active",
		},
	}
}

func (p *rabbitPublisher) Publish(ctx context.Context, message *messaging.Message) error {
	if p.confirmRequired {
		_, err := p.sendWithConfirm(ctx, message)
		return err
	}
	return p.publishSimple(ctx, message)
}

func (p *rabbitPublisher) PublishBatch(ctx context.Context, messages []*messaging.Message) error {
	if len(messages) == 0 {
		return nil
	}

	for _, msg := range messages {
		if p.confirmRequired {
			if _, err := p.sendWithConfirm(ctx, msg); err != nil {
				return err
			}
			continue
		}
		if err := p.publishSimple(ctx, msg); err != nil {
			return err
		}
	}

	return nil
}

func (p *rabbitPublisher) PublishWithConfirm(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
	return p.sendWithConfirm(ctx, message)
}

func (p *rabbitPublisher) PublishAsync(ctx context.Context, message *messaging.Message, callback messaging.PublishCallback) error {
	if callback == nil {
		return messaging.NewPublishError("rabbitmq async publish requires callback", nil)
	}

	go func() {
		if p.confirmRequired {
			confirmation, err := p.sendWithConfirm(ctx, message)
			callback(confirmation, err)
			return
		}
		err := p.publishSimple(ctx, message)
		callback(nil, err)
	}()

	return nil
}

func (p *rabbitPublisher) BeginTransaction(context.Context) (messaging.Transaction, error) {
	return nil, messaging.ErrTransactionsNotSupported
}

func (p *rabbitPublisher) Flush(context.Context) error { return nil }

func (p *rabbitPublisher) Close() error { return nil }

func (p *rabbitPublisher) GetMetrics() *messaging.PublisherMetrics {
	p.metricsMu.RLock()
	defer p.metricsMu.RUnlock()
	metrics := p.metrics
	return &metrics
}

func (p *rabbitPublisher) publishSimple(ctx context.Context, message *messaging.Message) error {
	if message == nil {
		return messaging.NewPublishError("rabbitmq publish requires a message", nil)
	}

	exchange, routingKey, err := p.resolveRouting(message)
	if err != nil {
		return err
	}

	start := time.Now()
	pub := p.toPublishing(message)
    err = p.withChannel(ctx, func(ch *channelWrapper) error {
        return ch.channel.Publish(exchange, routingKey, false, false, pub)
    })
    if err != nil {
        p.recordFailure(err)
        return messaging.NewPublishError("rabbitmq publish failed", err)
    }

	p.recordSuccess(time.Since(start))
	return nil
}

func (p *rabbitPublisher) sendWithConfirm(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
	if message == nil {
		return nil, messaging.NewPublishError("rabbitmq publish requires a message", nil)
	}

	exchange, routingKey, err := p.resolveRouting(message)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 1; attempt <= p.maxConfirmAttempts; attempt++ {
		start := time.Now()
		pub := p.toPublishing(message)
		confirmation, err := p.publishOnceWithConfirm(ctx, exchange, routingKey, pub, message)
		if err == nil {
			p.recordSuccess(time.Since(start))
			return confirmation, nil
		}

		if errors.Is(err, context.Canceled) {
			return nil, err
		}

		var msgErr *messaging.MessagingError
		if errors.As(err, &msgErr) && !msgErr.Retryable {
			return nil, err
		}

		lastErr = err
		if attempt < p.maxConfirmAttempts {
			p.recordRetry()
			delay := p.backoffDelay(attempt)
			if delay > 0 {
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			continue
		}

		p.recordFailure(err)
	}

	if lastErr == nil {
		lastErr = messaging.NewPublishError("rabbitmq publish confirmation failed", nil)
	}
	return nil, lastErr
}

func (p *rabbitPublisher) publishOnceWithConfirm(
	ctx context.Context,
	exchange, routingKey string,
	publish amqp.Publishing,
	message *messaging.Message,
) (*messaging.PublishConfirmation, error) {
	var result *messaging.PublishConfirmation
	err := p.withChannel(ctx, func(ch *channelWrapper) error {
        if err := ch.channel.Confirm(false); err != nil {
            return messaging.NewPublishError("rabbitmq enable confirms failed", err)
        }

		confirms := ch.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

        if err := ch.channel.Publish(exchange, routingKey, false, false, publish); err != nil {
            return messaging.NewPublishError("rabbitmq publish failed", err)
        }

        confirm, err := p.awaitConfirmation(ctx, confirms)
        if err != nil {
            return err
        }
		if !confirm.Ack {
			return messaging.NewPublishError("rabbitmq publish negatively acknowledged", nil)
		}

		metadata := map[string]string{
			"routing_key": routingKey,
		}
		if exchange != "" {
			metadata["exchange"] = exchange
		}
		metadata["delivery_tag"] = strconv.FormatUint(confirm.DeliveryTag, 10)

		result = &messaging.PublishConfirmation{
			MessageID: message.ID,
			Timestamp: time.Now(),
			Metadata:  metadata,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *rabbitPublisher) awaitConfirmation(ctx context.Context, confirms chan amqp.Confirmation) (amqp.Confirmation, error) {
	timer := time.NewTimer(p.confirmTimeout)
	defer timer.Stop()

	select {
	case confirm, ok := <-confirms:
		if !ok {
			return amqp.Confirmation{}, messaging.NewPublishError("rabbitmq confirmation channel closed", nil)
		}
		return confirm, nil
	case <-timer.C:
		return amqp.Confirmation{}, messaging.NewPublishError("rabbitmq publish confirmation timeout", context.DeadlineExceeded)
	case <-ctx.Done():
		return amqp.Confirmation{}, ctx.Err()
	}
}

func (p *rabbitPublisher) withChannel(ctx context.Context, fn func(*channelWrapper) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return messaging.NewConnectionError("rabbitmq connection acquire failed", err)
	}
	defer p.pool.Release(conn)

	session, err := conn.AcquireChannel(ctx)
	if err != nil {
		return messaging.NewConnectionError("rabbitmq channel acquire failed", err)
	}
	defer conn.ReleaseChannel(session)

	return fn(session)
}

func (p *rabbitPublisher) resolveRouting(message *messaging.Message) (string, string, error) {
	routingKey := p.config.Topic
	if message.Topic != "" {
		routingKey = message.Topic
	}
	if routingKey == "" {
		return "", "", messaging.NewConfigError("rabbitmq publisher requires a topic or routing key", nil)
	}
	return "", routingKey, nil
}

func (p *rabbitPublisher) toPublishing(message *messaging.Message) amqp.Publishing {
	headers := amqp.Table{}
	if len(message.Headers) > 0 {
		for k, v := range message.Headers {
			headers[k] = v
		}
	}
	if message.CorrelationID != "" {
		headers["correlation_id"] = message.CorrelationID
	}
	if message.ID != "" {
		headers["message_id"] = message.ID
	}

	timestamp := message.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	publishing := amqp.Publishing{
		Headers:       headers,
		Body:          message.Payload,
		MessageId:     message.ID,
		Timestamp:     timestamp,
		CorrelationId: message.CorrelationID,
		ReplyTo:       message.ReplyTo,
		Priority:      uint8(message.Priority),
		DeliveryMode:  amqp.Persistent,
	}

	if contentType, ok := message.Headers["content_type"]; ok {
		publishing.ContentType = contentType
	}
	if contentEncoding, ok := message.Headers["content_encoding"]; ok {
		publishing.ContentEncoding = contentEncoding
	}
	if message.TTL > 0 {
		publishing.Expiration = strconv.Itoa(message.TTL * 1000)
	}

	return publishing
}

func (p *rabbitPublisher) backoffDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	delay := p.retry.InitialDelay
	if delay <= 0 {
		delay = 100 * time.Millisecond
	}

	value := delay
	for i := 1; i < attempt; i++ {
		value = time.Duration(float64(value) * p.retry.Multiplier)
		if p.retry.MaxDelay > 0 && value > p.retry.MaxDelay {
			value = p.retry.MaxDelay
			break
		}
	}

	if p.retry.MaxDelay > 0 && value > p.retry.MaxDelay {
		value = p.retry.MaxDelay
	}

	return value
}

func (p *rabbitPublisher) recordSuccess(duration time.Duration) {
	p.metricsMu.Lock()
	defer p.metricsMu.Unlock()

	p.metrics.MessagesPublished++
	p.metrics.LastActivity = time.Now()
	p.metrics.AvgPublishLatency = updateLatencyAverage(p.metrics.AvgPublishLatency, duration)
	if duration > p.metrics.MaxPublishLatency {
		p.metrics.MaxPublishLatency = duration
	}
	if p.metrics.MinPublishLatency == 0 || (duration > 0 && duration < p.metrics.MinPublishLatency) {
		p.metrics.MinPublishLatency = duration
	}
}

func (p *rabbitPublisher) recordFailure(err error) {
	p.metricsMu.Lock()
	defer p.metricsMu.Unlock()

	p.metrics.MessagesFailed++
	p.metrics.LastError = err.Error()
	p.metrics.LastErrorTime = time.Now()
	p.metrics.LastActivity = time.Now()
}

func (p *rabbitPublisher) recordRetry() {
	p.metricsMu.Lock()
	p.metrics.MessagesRetried++
	p.metrics.LastActivity = time.Now()
	p.metricsMu.Unlock()
}

func updateLatencyAverage(current, latest time.Duration) time.Duration {
	if latest <= 0 {
		return current
	}
	if current == 0 {
		return latest
	}
	return (current + latest) / 2
}

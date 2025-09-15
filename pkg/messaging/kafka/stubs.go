package kafka

import (
	"context"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// stubPublisher is a minimal no-op publisher that compiles.
type stubPublisher struct{}

func (p *stubPublisher) Publish(ctx context.Context, message *messaging.Message) error {
	return nil
}

func (p *stubPublisher) PublishBatch(ctx context.Context, messages []*messaging.Message) error {
	return nil
}

func (p *stubPublisher) PublishWithConfirm(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
	return &messaging.PublishConfirmation{MessageID: message.ID, Timestamp: time.Now()}, nil
}

func (p *stubPublisher) PublishAsync(ctx context.Context, message *messaging.Message, callback messaging.PublishCallback) error {
	go func() { callback(&messaging.PublishConfirmation{MessageID: message.ID, Timestamp: time.Now()}, nil) }()
	return nil
}

func (p *stubPublisher) BeginTransaction(ctx context.Context) (messaging.Transaction, error) {
	return nil, messaging.ErrTransactionsNotSupported
}

func (p *stubPublisher) Flush(ctx context.Context) error { return nil }
func (p *stubPublisher) Close() error { return nil }
func (p *stubPublisher) GetMetrics() *messaging.PublisherMetrics { return &messaging.PublisherMetrics{} }

// stubSubscriber is a minimal no-op subscriber that compiles.
type stubSubscriber struct{}

func (s *stubSubscriber) Subscribe(ctx context.Context, handler messaging.MessageHandler) error { return nil }

func (s *stubSubscriber) SubscribeWithMiddleware(ctx context.Context, handler messaging.MessageHandler, middleware ...messaging.Middleware) error {
	return nil
}

func (s *stubSubscriber) Unsubscribe(ctx context.Context) error { return nil }
func (s *stubSubscriber) Pause(ctx context.Context) error        { return nil }
func (s *stubSubscriber) Resume(ctx context.Context) error       { return nil }

func (s *stubSubscriber) Seek(ctx context.Context, position messaging.SeekPosition) error {
	return messaging.NewConfigError("seek not supported in stub", nil)
}

func (s *stubSubscriber) GetLag(ctx context.Context) (int64, error) { return -1, nil }
func (s *stubSubscriber) Close() error                              { return nil }
func (s *stubSubscriber) GetMetrics() *messaging.SubscriberMetrics  { return &messaging.SubscriberMetrics{} }



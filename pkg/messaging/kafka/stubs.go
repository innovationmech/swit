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
func (p *stubPublisher) Close() error                    { return nil }
func (p *stubPublisher) GetMetrics() *messaging.PublisherMetrics {
	return &messaging.PublisherMetrics{}
}

// stubSubscriber is a minimal no-op subscriber that compiles.
type stubSubscriber struct{}

func (s *stubSubscriber) Subscribe(ctx context.Context, handler messaging.MessageHandler) error {
	return nil
}

func (s *stubSubscriber) SubscribeWithMiddleware(ctx context.Context, handler messaging.MessageHandler, middleware ...messaging.Middleware) error {
	return nil
}

func (s *stubSubscriber) Unsubscribe(ctx context.Context) error { return nil }
func (s *stubSubscriber) Pause(ctx context.Context) error       { return nil }
func (s *stubSubscriber) Resume(ctx context.Context) error      { return nil }

func (s *stubSubscriber) Seek(ctx context.Context, position messaging.SeekPosition) error {
	return messaging.NewConfigError("seek not supported in stub", nil)
}

func (s *stubSubscriber) GetLag(ctx context.Context) (int64, error) { return -1, nil }
func (s *stubSubscriber) Close() error                              { return nil }
func (s *stubSubscriber) GetMetrics() *messaging.SubscriberMetrics {
	return &messaging.SubscriberMetrics{}
}

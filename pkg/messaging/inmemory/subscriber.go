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
	"sync"
	"sync/atomic"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// subscriber implements messaging.EventSubscriber on top of the broker's
// per-topic logs. One worker goroutine per subscribed topic consumes messages
// through the consumer-group cursor.
type subscriber struct {
	broker *Broker
	config messaging.SubscriberConfig

	mu      sync.Mutex
	started bool
	closed  bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	paused  atomic.Bool
	stopped atomic.Bool

	metricsMu sync.Mutex
	metrics   messaging.SubscriberMetrics
}

func newSubscriber(b *Broker, config messaging.SubscriberConfig) *subscriber {
	return &subscriber{broker: b, config: config}
}

// Subscribe implements messaging.EventSubscriber. It starts background
// workers and returns immediately; processing continues until the context is
// cancelled or Unsubscribe/Close is called.
func (s *subscriber) Subscribe(ctx context.Context, handler messaging.MessageHandler) error {
	return s.SubscribeWithMiddleware(ctx, handler)
}

// SubscribeWithMiddleware implements messaging.EventSubscriber.
func (s *subscriber) SubscribeWithMiddleware(ctx context.Context, handler messaging.MessageHandler, middleware ...messaging.Middleware) error {
	if handler == nil {
		return messaging.NewConfigError("message handler cannot be nil", nil)
	}

	// Apply middleware so the first middleware is the outermost wrapper.
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i].Wrap(handler)
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return messaging.ErrSubscriberAlreadyClosed
	}
	if s.started {
		s.mu.Unlock()
		return messaging.NewConfigError("subscriber is already subscribed", nil)
	}
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.started = true
	s.stopped.Store(false)
	s.mu.Unlock()

	for _, topic := range s.config.Topics {
		t := s.broker.getTopic(topic)
		group := s.acquireGroup(t)

		s.wg.Add(1)
		go s.runLoop(workerCtx, t, group, handler)
	}

	// Wake blocked workers when the context is cancelled.
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-workerCtx.Done()
		s.stopped.Store(true)
		s.broadcastTopics()
	}()

	return nil
}

// acquireGroup returns the consumer-group state for this subscriber within a
// topic, initializing the cursor according to the configured initial offset.
func (s *subscriber) acquireGroup(t *topicState) *groupState {
	t.mu.Lock()
	defer t.mu.Unlock()

	group, ok := t.groups[s.config.ConsumerGroup]
	if !ok {
		group = &groupState{}
		if s.config.Offset.Initial == messaging.OffsetEarliest {
			group.cursor = 0
		} else {
			group.cursor = len(t.messages)
		}
		t.groups[s.config.ConsumerGroup] = group
	}
	return group
}

// runLoop consumes messages from a topic for a consumer group until stopped.
func (s *subscriber) runLoop(ctx context.Context, t *topicState, group *groupState, handler messaging.MessageHandler) {
	defer s.wg.Done()

	for {
		// Honor pause before pulling the next message.
		for s.paused.Load() && !s.stopped.Load() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Millisecond):
			}
		}

		t.mu.Lock()
		for !s.stopped.Load() && group.cursor >= len(t.messages) {
			t.cond.Wait()
		}
		if s.stopped.Load() {
			t.mu.Unlock()
			return
		}
		msg := t.messages[group.cursor]
		group.cursor++
		t.mu.Unlock()

		s.dispatch(ctx, handler, copyMessage(msg))
	}
}

// dispatch processes a single message with retry and dead-letter semantics.
func (s *subscriber) dispatch(ctx context.Context, handler messaging.MessageHandler, msg *messaging.Message) {
	maxAttempts := s.config.Retry.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	s.metricsMu.Lock()
	s.metrics.MessagesConsumed++
	s.metricsMu.Unlock()
	s.broker.recordConsumed(1)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		msg.DeliveryAttempt = attempt

		start := time.Now()
		err := handler.Handle(ctx, msg)
		elapsed := time.Since(start)

		if err == nil {
			s.metricsMu.Lock()
			s.metrics.MessagesProcessed++
			s.metrics.LastActivity = time.Now()
			if elapsed > s.metrics.MaxProcessingTime {
				s.metrics.MaxProcessingTime = elapsed
			}
			s.metricsMu.Unlock()
			return
		}

		s.metricsMu.Lock()
		s.metrics.MessagesFailed++
		s.metrics.LastError = err.Error()
		s.metrics.LastErrorTime = time.Now()
		s.metricsMu.Unlock()

		switch handler.OnError(ctx, msg, err) {
		case messaging.ErrorActionRetry:
			if attempt < maxAttempts {
				s.metricsMu.Lock()
				s.metrics.MessagesRetried++
				s.metricsMu.Unlock()
				continue
			}
			s.deadLetter(msg)
			return
		case messaging.ErrorActionDeadLetter:
			s.deadLetter(msg)
			return
		case messaging.ErrorActionPause:
			_ = s.Pause(ctx)
			return
		default: // messaging.ErrorActionDiscard
			return
		}
	}
}

// deadLetter forwards the message to the configured dead-letter topic.
func (s *subscriber) deadLetter(msg *messaging.Message) {
	if !s.config.DeadLetter.Enabled || s.config.DeadLetter.Topic == "" {
		return
	}
	dlqMsg := copyMessage(msg)
	dlqMsg.Topic = s.config.DeadLetter.Topic
	s.broker.appendMessages(s.config.DeadLetter.Topic, []*messaging.Message{dlqMsg})

	s.metricsMu.Lock()
	s.metrics.MessagesDeadLettered++
	s.metricsMu.Unlock()
}

// Unsubscribe implements messaging.EventSubscriber. It stops all workers and
// waits for in-flight processing to finish.
func (s *subscriber) Unsubscribe(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = false
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()

	s.stopped.Store(true)
	if cancel != nil {
		cancel()
	}
	s.broadcastTopics()

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

// Pause implements messaging.EventSubscriber.
func (s *subscriber) Pause(ctx context.Context) error {
	s.paused.Store(true)
	return nil
}

// Resume implements messaging.EventSubscriber.
func (s *subscriber) Resume(ctx context.Context) error {
	s.paused.Store(false)
	return nil
}

// Seek implements messaging.EventSubscriber. It repositions the consumer
// group cursor on the matching topics.
func (s *subscriber) Seek(ctx context.Context, position messaging.SeekPosition) error {
	if err := ctx.Err(); err != nil {
		return messaging.NewConfigError("context cancelled during seek", err)
	}

	matched := false
	for _, topic := range s.config.Topics {
		if position.Topic != "" && position.Topic != topic {
			continue
		}
		matched = true

		t := s.broker.getTopic(topic)
		t.mu.Lock()
		group, ok := t.groups[s.config.ConsumerGroup]
		if !ok {
			group = &groupState{}
			t.groups[s.config.ConsumerGroup] = group
		}
		switch position.Type {
		case messaging.SeekTypeBeginning:
			group.cursor = 0
		case messaging.SeekTypeEnd:
			group.cursor = len(t.messages)
		case messaging.SeekTypeOffset:
			cursor := int(position.Offset)
			if cursor < 0 {
				cursor = 0
			}
			if cursor > len(t.messages) {
				cursor = len(t.messages)
			}
			group.cursor = cursor
		case messaging.SeekTypeTimestamp:
			cursor := len(t.messages)
			for i, m := range t.messages {
				if !m.Timestamp.Before(position.Timestamp) {
					cursor = i
					break
				}
			}
			group.cursor = cursor
		default:
			t.mu.Unlock()
			return messaging.NewConfigError("unsupported seek position type", nil)
		}
		t.cond.Broadcast()
		t.mu.Unlock()
	}

	if !matched {
		return messaging.NewConfigError("seek topic does not match any subscribed topic", nil)
	}
	return nil
}

// GetLag implements messaging.EventSubscriber. It returns the total number of
// messages not yet consumed by this subscriber's consumer group.
func (s *subscriber) GetLag(ctx context.Context) (int64, error) {
	var lag int64
	for _, topic := range s.config.Topics {
		t := s.broker.getTopic(topic)
		t.mu.Lock()
		if group, ok := t.groups[s.config.ConsumerGroup]; ok {
			lag += int64(len(t.messages) - group.cursor)
		} else {
			lag += int64(len(t.messages))
		}
		t.mu.Unlock()
	}
	return lag, nil
}

// Close implements messaging.EventSubscriber.
func (s *subscriber) Close() error {
	if err := s.Unsubscribe(context.Background()); err != nil {
		return err
	}

	s.mu.Lock()
	alreadyClosed := s.closed
	s.closed = true
	s.mu.Unlock()

	if !alreadyClosed {
		s.broker.removeSubscriber(s)
	}
	return nil
}

// GetMetrics implements messaging.EventSubscriber.
func (s *subscriber) GetMetrics() *messaging.SubscriberMetrics {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	snapshot := s.metrics
	snapshot.ConsumerGroup = s.config.ConsumerGroup
	return &snapshot
}

// broadcastTopics wakes all workers blocked on subscribed topics.
func (s *subscriber) broadcastTopics() {
	for _, topic := range s.config.Topics {
		t := s.broker.getTopic(topic)
		t.mu.Lock()
		t.cond.Broadcast()
		t.mu.Unlock()
	}
}

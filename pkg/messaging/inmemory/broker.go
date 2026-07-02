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

// Package inmemory provides an in-process implementation of the messaging
// abstractions (MessageBroker, EventPublisher, EventSubscriber). It requires
// no external middleware and is intended for local development, unit tests
// and CI pipelines.
//
// Messages are retained in per-topic logs guarded by mutexes; subscribers
// consume through goroutines coordinated with condition variables, giving
// ordered delivery, consumer-group distribution, seek and lag semantics.
package inmemory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/innovationmech/swit/pkg/messaging"
)

// Broker implements messaging.MessageBroker entirely in process memory.
// It is safe for concurrent use.
type Broker struct {
	mu        sync.RWMutex
	config    *messaging.BrokerConfig
	connected bool
	topics    map[string]*topicState

	publishers  []*publisher
	subscribers []*subscriber

	metricsMu sync.Mutex
	metrics   messaging.BrokerMetrics
}

// topicState holds the retained message log and consumer-group cursors for a
// single topic. cond is signalled whenever new messages are appended so that
// blocked subscribers wake up.
type topicState struct {
	mu       sync.Mutex
	cond     *sync.Cond
	messages []*messaging.Message
	groups   map[string]*groupState
}

// groupState tracks the shared read cursor of a consumer group within a
// topic. Subscribers in the same group compete for messages; distinct groups
// each receive every message.
type groupState struct {
	cursor int
}

func newTopicState() *topicState {
	t := &topicState{groups: make(map[string]*groupState)}
	t.cond = sync.NewCond(&t.mu)
	return t
}

// New creates a new in-memory broker with the provided configuration.
// The configuration may be nil; endpoints are not required.
func New(config *messaging.BrokerConfig) *Broker {
	if config == nil {
		config = &messaging.BrokerConfig{Type: messaging.BrokerTypeInMemory}
	}
	return &Broker{
		config: config,
		topics: make(map[string]*topicState),
	}
}

// Connect implements messaging.MessageBroker. It is idempotent.
func (b *Broker) Connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return messaging.NewConnectionError("context cancelled during connect", err)
	}

	b.mu.Lock()
	b.connected = true
	b.mu.Unlock()

	b.metricsMu.Lock()
	b.metrics.ConnectionStatus = "connected"
	b.metrics.ConnectionAttempts++
	b.metrics.LastConnectionTime = time.Now()
	b.metricsMu.Unlock()
	return nil
}

// Disconnect implements messaging.MessageBroker. It closes all publishers
// and subscribers created from this broker and releases waiting goroutines.
func (b *Broker) Disconnect(ctx context.Context) error {
	b.mu.Lock()
	if !b.connected {
		b.mu.Unlock()
		return nil
	}
	b.connected = false
	pubs := b.publishers
	subs := b.subscribers
	b.publishers = nil
	b.subscribers = nil
	topics := make([]*topicState, 0, len(b.topics))
	for _, t := range b.topics {
		topics = append(topics, t)
	}
	b.mu.Unlock()

	for _, p := range pubs {
		_ = p.Close()
	}
	for _, s := range subs {
		_ = s.Close()
	}

	// Wake any goroutine still blocked on a topic condition variable.
	for _, t := range topics {
		t.mu.Lock()
		t.cond.Broadcast()
		t.mu.Unlock()
	}

	b.metricsMu.Lock()
	b.metrics.ConnectionStatus = "disconnected"
	b.metricsMu.Unlock()
	return nil
}

// Close implements messaging.MessageBroker (io.Closer semantics).
func (b *Broker) Close() error {
	return b.Disconnect(context.Background())
}

// IsConnected implements messaging.MessageBroker.
func (b *Broker) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

// CreatePublisher implements messaging.MessageBroker.
func (b *Broker) CreatePublisher(config messaging.PublisherConfig) (messaging.EventPublisher, error) {
	if !b.IsConnected() {
		return nil, messaging.ErrBrokerNotConnected
	}
	if config.Topic == "" {
		return nil, messaging.NewConfigError("publisher topic is required", nil)
	}

	p := newPublisher(b, config)
	b.mu.Lock()
	b.publishers = append(b.publishers, p)
	b.mu.Unlock()

	b.metricsMu.Lock()
	b.metrics.PublishersCreated++
	b.metrics.ActivePublishers++
	b.metricsMu.Unlock()
	return p, nil
}

// CreateSubscriber implements messaging.MessageBroker.
func (b *Broker) CreateSubscriber(config messaging.SubscriberConfig) (messaging.EventSubscriber, error) {
	if !b.IsConnected() {
		return nil, messaging.ErrBrokerNotConnected
	}
	if len(config.Topics) == 0 {
		return nil, messaging.NewConfigError("subscriber topics are required", nil)
	}
	if config.ConsumerGroup == "" {
		// Assign a unique group so ungrouped subscribers receive every
		// message (broadcast semantics), mirroring core NATS behavior.
		config.ConsumerGroup = "anonymous-" + uuid.NewString()
	}

	s := newSubscriber(b, config)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, s)
	b.mu.Unlock()

	b.metricsMu.Lock()
	b.metrics.SubscribersCreated++
	b.metrics.ActiveSubscribers++
	b.metricsMu.Unlock()
	return s, nil
}

// HealthCheck implements messaging.MessageBroker.
func (b *Broker) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	start := time.Now()
	status := &messaging.HealthStatus{
		Status:      messaging.HealthStatusHealthy,
		Message:     "in-memory broker healthy",
		LastChecked: start,
		Details: map[string]any{
			"connected": b.IsConnected(),
			"topics":    b.topicCount(),
		},
	}
	if !b.IsConnected() {
		status.Status = messaging.HealthStatusUnhealthy
		status.Message = "in-memory broker not connected"
	}
	status.ResponseTime = time.Since(start)
	return status, nil
}

// GetMetrics implements messaging.MessageBroker.
func (b *Broker) GetMetrics() *messaging.BrokerMetrics {
	b.metricsMu.Lock()
	defer b.metricsMu.Unlock()
	return b.metrics.GetSnapshot()
}

// GetCapabilities implements messaging.MessageBroker.
func (b *Broker) GetCapabilities() *messaging.BrokerCapabilities {
	caps, _ := messaging.GetCapabilityProfile(messaging.BrokerTypeInMemory)
	return caps
}

func (b *Broker) topicCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.topics)
}

// getTopic returns the state for a topic, creating it lazily.
func (b *Broker) getTopic(name string) *topicState {
	b.mu.RLock()
	t, ok := b.topics[name]
	b.mu.RUnlock()
	if ok {
		return t
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if t, ok = b.topics[name]; ok {
		return t
	}
	t = newTopicState()
	b.topics[name] = t
	return t
}

// appendMessages appends messages to a topic log atomically and returns the
// offset assigned to each message. Waiting subscribers are woken up.
func (b *Broker) appendMessages(topic string, msgs []*messaging.Message) []int64 {
	t := b.getTopic(topic)

	t.mu.Lock()
	offsets := make([]int64, len(msgs))
	for i, m := range msgs {
		offsets[i] = int64(len(t.messages))
		t.messages = append(t.messages, m)
	}
	t.cond.Broadcast()
	t.mu.Unlock()

	b.metricsMu.Lock()
	b.metrics.MessagesPublished += int64(len(msgs))
	b.metricsMu.Unlock()
	return offsets
}

// removePublisher drops the publisher from broker tracking.
func (b *Broker) removePublisher(p *publisher) {
	b.mu.Lock()
	for i, cur := range b.publishers {
		if cur == p {
			b.publishers = append(b.publishers[:i], b.publishers[i+1:]...)
			break
		}
	}
	b.mu.Unlock()

	b.metricsMu.Lock()
	if b.metrics.ActivePublishers > 0 {
		b.metrics.ActivePublishers--
	}
	b.metricsMu.Unlock()
}

// removeSubscriber drops the subscriber from broker tracking.
func (b *Broker) removeSubscriber(s *subscriber) {
	b.mu.Lock()
	for i, cur := range b.subscribers {
		if cur == s {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			break
		}
	}
	b.mu.Unlock()

	b.metricsMu.Lock()
	if b.metrics.ActiveSubscribers > 0 {
		b.metrics.ActiveSubscribers--
	}
	b.metricsMu.Unlock()
}

// recordConsumed updates broker-level consumption metrics.
func (b *Broker) recordConsumed(n int64) {
	b.metricsMu.Lock()
	b.metrics.MessagesConsumed += n
	b.metricsMu.Unlock()
}

// copyMessage returns a shallow copy of a message with its own headers map so
// handlers cannot mutate the shared log entry.
func copyMessage(m *messaging.Message) *messaging.Message {
	cp := *m
	if m.Headers != nil {
		cp.Headers = make(map[string]string, len(m.Headers))
		for k, v := range m.Headers {
			cp.Headers[k] = v
		}
	}
	return &cp
}

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

package messaging

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// 本文件为 messaging 包内的基准测试，避免引入依赖环导致的 import cycle，
// 使用最小可用的本地 mock 实现。

type benchHandler struct {
	id     string
	topics []string
	workNs int64
	count  int64
}

func newBenchHandler(id string, work time.Duration) *benchHandler {
	return &benchHandler{id: id, topics: []string{"benchmark-topic"}, workNs: int64(work)}
}

func (h *benchHandler) GetHandlerID() string                 { return h.id }
func (h *benchHandler) GetTopics() []string                  { return h.topics }
func (h *benchHandler) GetBrokerRequirement() string         { return "" }
func (h *benchHandler) Initialize(ctx context.Context) error { return nil }
func (h *benchHandler) Shutdown(ctx context.Context) error   { return nil }
func (h *benchHandler) Handle(ctx context.Context, _ *Message) error {
	if h.workNs > 0 {
		deadline := time.Now().Add(time.Duration(h.workNs))
		for time.Now().Before(deadline) {
		}
	}
	atomic.AddInt64(&h.count, 1)
	return nil
}
func (h *benchHandler) OnError(ctx context.Context, _ *Message, _ error) ErrorAction {
	return ErrorActionRetry
}

type benchSubscriber struct {
	handlers []MessageHandler
}

func (s *benchSubscriber) Subscribe(ctx context.Context, h MessageHandler) error {
	s.handlers = append(s.handlers, h)
	return nil
}
func (s *benchSubscriber) SubscribeWithMiddleware(ctx context.Context, h MessageHandler, _ ...Middleware) error {
	return s.Subscribe(ctx, h)
}
func (s *benchSubscriber) Unsubscribe(ctx context.Context) error          { return nil }
func (s *benchSubscriber) Pause(ctx context.Context) error                { return nil }
func (s *benchSubscriber) Resume(ctx context.Context) error               { return nil }
func (s *benchSubscriber) Seek(ctx context.Context, _ SeekPosition) error { return nil }
func (s *benchSubscriber) GetLag(ctx context.Context) (int64, error)      { return 0, nil }
func (s *benchSubscriber) Close() error                                   { return nil }
func (s *benchSubscriber) GetMetrics() *SubscriberMetrics                 { return &SubscriberMetrics{} }

type benchBroker struct{}

func (b *benchBroker) Connect(ctx context.Context) error    { return nil }
func (b *benchBroker) Disconnect(ctx context.Context) error { return nil }
func (b *benchBroker) Close() error                         { return nil }
func (b *benchBroker) IsConnected() bool                    { return true }
func (b *benchBroker) CreatePublisher(config PublisherConfig) (EventPublisher, error) {
	return nil, nil
}
func (b *benchBroker) CreateSubscriber(config SubscriberConfig) (EventSubscriber, error) {
	return &benchSubscriber{}, nil
}
func (b *benchBroker) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	return &HealthStatus{Status: HealthStatusHealthy, Message: "ok", LastChecked: time.Now()}, nil
}
func (b *benchBroker) GetMetrics() *BrokerMetrics { return &BrokerMetrics{} }
func (b *benchBroker) GetCapabilities() *BrokerCapabilities {
	return &BrokerCapabilities{SupportsConsumerGroups: true}
}

type benchHarness struct {
	coord    MessagingCoordinator
	broker   *benchBroker
	handlers []*benchHandler
	sub      *benchSubscriber
	messages []Message
}

func newBenchHarness(handlerCount int, work time.Duration) *benchHarness {
	h := &benchHarness{
		coord:    NewMessagingCoordinator(),
		broker:   &benchBroker{},
		handlers: make([]*benchHandler, 0, handlerCount),
		sub:      &benchSubscriber{},
	}
	for i := 0; i < handlerCount; i++ {
		h.handlers = append(h.handlers, newBenchHandler(fmt.Sprintf("h-%d", i+1), work))
	}
	return h
}

func (h *benchHarness) register() error {
	if err := h.coord.RegisterBroker("default", h.broker); err != nil {
		return err
	}
	for _, handler := range h.handlers {
		if err := h.coord.RegisterEventHandler(handler); err != nil {
			return err
		}
	}
	return nil
}

func (h *benchHarness) start(ctx context.Context) error { return h.coord.Start(ctx) }
func (h *benchHarness) stop(ctx context.Context) error  { return h.coord.Stop(ctx) }

func (h *benchHarness) genMessages(n int, size int) []Message {
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = byte('A' + (i % 26))
	}
	msgs := make([]Message, n)
	now := time.Now()
	for i := 0; i < n; i++ {
		msgs[i] = Message{ID: fmt.Sprintf("m-%d", i), Headers: map[string]string{"bench": "true"}, Payload: payload, Topic: "benchmark-topic", Key: []byte(fmt.Sprintf("k-%d", i%16)), Timestamp: now}
	}
	return msgs
}

func (h *benchHarness) simulate(n int) {
	if len(h.sub.handlers) == 0 {
		// 在 coordinator.createSubscriptionForHandler 中会创建 subscriber 并在 goroutine 中调用 Subscribe。
		// 简化起见，这里直接手动注册处理器列表。
		for _, hd := range h.handlers {
			h.sub.handlers = append(h.sub.handlers, hd)
		}
	}
	if h.messages == nil || len(h.messages) < n {
		h.messages = h.genMessages(n, 512)
	}
	ctx := context.Background()
	for i := 0; i < n; i++ {
		m := h.messages[i]
		for _, hd := range h.sub.handlers {
			_ = hd.Handle(ctx, &m)
		}
	}
}

func BenchmarkMessagingThroughput(b *testing.B) {
	h := newBenchHarness(4, 0)
	if err := h.register(); err != nil {
		b.Fatal(err)
	}
	if err := h.start(context.Background()); err != nil {
		b.Fatal(err)
	}
	defer h.stop(context.Background())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.simulate(1000)
	}
}

func BenchmarkMessagingLatency(b *testing.B) {
	h := newBenchHarness(2, 50*time.Microsecond)
	if err := h.register(); err != nil {
		b.Fatal(err)
	}
	if err := h.start(context.Background()); err != nil {
		b.Fatal(err)
	}
	defer h.stop(context.Background())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.simulate(100)
	}
}

func BenchmarkMessagingMemory(b *testing.B) {
	h := newBenchHarness(1, 0)
	if err := h.register(); err != nil {
		b.Fatal(err)
	}
	if err := h.start(context.Background()); err != nil {
		b.Fatal(err)
	}
	defer h.stop(context.Background())

	capture := func(fn func()) uint64 {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		fn()
		runtime.GC()
		runtime.ReadMemStats(&m2)
		if m2.Alloc > m1.Alloc {
			return m2.Alloc - m1.Alloc
		}
		return 0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		delta := capture(func() { h.simulate(5000) })
		b.ReportMetric(float64(delta), "bytes/op")
	}
}

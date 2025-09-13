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
package testingx

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	msgtest "github.com/innovationmech/swit/pkg/messaging/testutil"
	"github.com/stretchr/testify/mock"
)

// BenchmarkHandler 是用于消息处理基准测试的轻量事件处理器
type BenchmarkHandler struct {
	id              string
	topics          []string
	simulatedWorkNs int64
	processed       int64
}

func NewBenchmarkHandler(id string, topics []string, simulatedWork time.Duration) *BenchmarkHandler {
	return &BenchmarkHandler{id: id, topics: topics, simulatedWorkNs: int64(simulatedWork)}
}

func (h *BenchmarkHandler) GetHandlerID() string                 { return h.id }
func (h *BenchmarkHandler) GetTopics() []string                  { return h.topics }
func (h *BenchmarkHandler) GetBrokerRequirement() string         { return "" }
func (h *BenchmarkHandler) Initialize(ctx context.Context) error { return nil }
func (h *BenchmarkHandler) Shutdown(ctx context.Context) error   { return nil }

// Handle 模拟固定耗时的处理逻辑（自旋等待），避免 sleep 带来的调度噪音
func (h *BenchmarkHandler) Handle(ctx context.Context, _ *messaging.Message) error {
	if h.simulatedWorkNs > 0 {
		deadline := time.Now().Add(time.Duration(h.simulatedWorkNs))
		for time.Now().Before(deadline) {
			// 自旋到达时间点，避免 sleep
		}
	}
	atomic.AddInt64(&h.processed, 1)
	return nil
}

func (h *BenchmarkHandler) OnError(ctx context.Context, _ *messaging.Message, _ error) messaging.ErrorAction {
	return messaging.ErrorActionRetry
}

func (h *BenchmarkHandler) Processed() int64 { return atomic.LoadInt64(&h.processed) }

// MessagingBenchmarkHarness 封装用于基准测试的协调器/代理/订阅者
type MessagingBenchmarkHarness struct {
	Coordinator messaging.MessagingCoordinator
	Broker      *msgtest.MockMessageBroker
	Subscriber  *msgtest.MockEventSubscriber
	Handlers    []messaging.EventHandler
}

// NewMessagingBenchmarkHarness 创建带 Mock Broker 与可模拟消息的 Subscriber
func NewMessagingBenchmarkHarness() *MessagingBenchmarkHarness {
	coord := messaging.NewMessagingCoordinator()
	broker := msgtest.NewMockMessageBroker()
	subscriber := msgtest.NewMockEventSubscriber()

	// 宽松匹配期望
	broker.On("Connect", mock.Anything).Return(nil).Maybe()
	broker.On("Disconnect", mock.Anything).Return(nil).Maybe()
	broker.On("Close").Return(nil).Maybe()
	broker.On("IsConnected").Return(true).Maybe()
	broker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(subscriber, nil).Maybe()

	return &MessagingBenchmarkHarness{
		Coordinator: coord,
		Broker:      broker,
		Subscriber:  subscriber,
		Handlers:    make([]messaging.EventHandler, 0),
	}
}

// WithHandlers 添加 N 个 BenchmarkHandler
func (h *MessagingBenchmarkHarness) WithHandlers(count int, simulatedWork time.Duration) *MessagingBenchmarkHarness {
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("bench-handler-%d", i+1)
		h.Handlers = append(h.Handlers, NewBenchmarkHandler(id, []string{"benchmark-topic"}, simulatedWork))
	}
	return h
}

// Register 注册 Broker 与 Handlers
func (h *MessagingBenchmarkHarness) Register() error {
	if err := h.Coordinator.RegisterBroker("default", h.Broker); err != nil {
		return err
	}
	for _, handler := range h.Handlers {
		if err := h.Coordinator.RegisterEventHandler(handler); err != nil {
			return err
		}
	}
	return nil
}

// Start 启动协调器
func (h *MessagingBenchmarkHarness) Start(ctx context.Context) error {
	return h.Coordinator.Start(ctx)
}

// Stop 关闭协调器
func (h *MessagingBenchmarkHarness) Stop(ctx context.Context) error {
	return h.Coordinator.Stop(ctx)
}

// SimulateMessages 通过 Mock Subscriber 触发处理，用于吞吐/延迟测试
func (h *MessagingBenchmarkHarness) SimulateMessages(b *testing.B, total int) {
	helper := msgtest.NewBenchmarkHelper()
	msgs := helper.CreateBenchmarkMessages()
	ctx := context.Background()

	for i := 0; i < total; i++ {
		msg := msgs[i%len(msgs)]
		_ = h.Subscriber.SimulateMessage(ctx, msg)
	}
}

// CaptureMemoryDelta 返回执行 fn 前后内存增长（字节）
func CaptureMemoryDelta(fn func()) uint64 {
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

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

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
	msgtest "github.com/innovationmech/swit/pkg/messaging/testutil"
	"github.com/stretchr/testify/mock"
)

// SimpleEventHandler 是最小可用的事件处理器实现，用于生命周期测试。
type SimpleEventHandler struct {
	id     string
	topics []string
}

func NewSimpleEventHandler(id string, topics []string) *SimpleEventHandler {
	return &SimpleEventHandler{id: id, topics: topics}
}

func (h *SimpleEventHandler) GetHandlerID() string                                   { return h.id }
func (h *SimpleEventHandler) GetTopics() []string                                    { return h.topics }
func (h *SimpleEventHandler) GetBrokerRequirement() string                           { return "" }
func (h *SimpleEventHandler) Initialize(ctx context.Context) error                   { return nil }
func (h *SimpleEventHandler) Shutdown(ctx context.Context) error                     { return nil }
func (h *SimpleEventHandler) Handle(ctx context.Context, _ *messaging.Message) error { return nil }
func (h *SimpleEventHandler) OnError(ctx context.Context, _ *messaging.Message, _ error) messaging.ErrorAction {
	return messaging.ErrorActionRetry
}

// LifecycleHarness 封装了快速搭建 MessagingCoordinator + Broker + Handlers 的逻辑。
type LifecycleHarness struct {
	Coordinator messaging.MessagingCoordinator
	Broker      *msgtest.MockMessageBroker
	Handlers    []messaging.EventHandler
}

// NewLifecycleHarness 创建带有默认模拟 Broker/Subscriber 的测试夹具。
func NewLifecycleHarness() *LifecycleHarness {
	coord := messaging.NewMessagingCoordinator()
	broker := msgtest.NewMockMessageBroker()
	sub := &dummySubscriber{}

	// 宽松匹配的期望，避免参数类型不一致导致的测试脆弱
	broker.On("Connect", mock.Anything).Return(nil).Maybe()
	broker.On("Disconnect", mock.Anything).Return(nil).Maybe()
	broker.On("Close").Return(nil).Maybe()
	broker.On("IsConnected").Return(true).Maybe()
	broker.On("CreateSubscriber", mock.AnythingOfType("messaging.SubscriberConfig")).Return(sub, nil).Maybe()

	return &LifecycleHarness{
		Coordinator: coord,
		Broker:      broker,
		Handlers:    make([]messaging.EventHandler, 0),
	}
}

// dummySubscriber 实现 messaging.EventSubscriber 接口，供生命周期测试使用。
type dummySubscriber struct{}

func (*dummySubscriber) Subscribe(ctx context.Context, _ messaging.MessageHandler) error { return nil }
func (*dummySubscriber) SubscribeWithMiddleware(ctx context.Context, _ messaging.MessageHandler, _ ...messaging.Middleware) error {
	return nil
}
func (*dummySubscriber) Unsubscribe(ctx context.Context) error                    { return nil }
func (*dummySubscriber) Pause(ctx context.Context) error                          { return nil }
func (*dummySubscriber) Resume(ctx context.Context) error                         { return nil }
func (*dummySubscriber) Seek(ctx context.Context, _ messaging.SeekPosition) error { return nil }
func (*dummySubscriber) GetLag(ctx context.Context) (int64, error)                { return 0, nil }
func (*dummySubscriber) Close() error                                             { return nil }
func (*dummySubscriber) GetMetrics() *messaging.SubscriberMetrics {
	return &messaging.SubscriberMetrics{}
}

// WithHandlers 添加给定数量的简单事件处理器。
func (h *LifecycleHarness) WithHandlers(count int) *LifecycleHarness {
	for i := 0; i < count; i++ {
		id := "handler-" + strconv.Itoa(i+1)
		h.Handlers = append(h.Handlers, NewSimpleEventHandler(id, []string{"test-topic"}))
	}
	return h
}

// Register 注册 Broker 与 Handlers 到协调器。
func (h *LifecycleHarness) Register() error {
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

// Start 启动协调器。
func (h *LifecycleHarness) Start(ctx context.Context) error {
	return h.Coordinator.Start(ctx)
}

// ShutdownGracefully 触发优雅关停并返回管理器。
func (h *LifecycleHarness) ShutdownGracefully(ctx context.Context, cfg *messaging.ShutdownConfig) (*messaging.GracefulShutdownManager, error) {
	return h.Coordinator.GracefulShutdown(ctx, cfg)
}

// SimulateInflight 模拟一定数量的在途消息在 duration 后完成。
func (h *LifecycleHarness) SimulateInflight(mgr *messaging.GracefulShutdownManager, count int, duration time.Duration) {
	for i := 0; i < count; i++ {
		mgr.IncrementInflightMessages()
		go func() {
			time.Sleep(duration)
			mgr.DecrementInflightMessages()
		}()
	}
}

// String helpers（保留以便后续扩展）。
func formatKV(k string, v interface{}) string { return fmt.Sprintf("%s=%v", k, v) }

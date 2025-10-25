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

package chaos

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestMessageLoss_EventPublishing tests Saga behavior when events are lost during publishing.
func TestMessageLoss_EventPublishing(t *testing.T) {
	tests := []struct {
		name        string
		lossRate    float64
		eventType   saga.SagaEventType
		expectLoss  bool
		description string
	}{
		{
			name:        "100%消息丢失率",
			lossRate:    1.0,
			eventType:   saga.EventSagaStarted,
			expectLoss:  true,
			description: "所有消息都会丢失",
		},
		{
			name:        "50%消息丢失率",
			lossRate:    0.5,
			eventType:   saga.EventSagaStepCompleted,
			expectLoss:  false,
			description: "部分消息会丢失",
		},
		{
			name:        "补偿事件丢失",
			lossRate:    1.0,
			eventType:   saga.EventCompensationStarted,
			expectLoss:  true,
			description: "补偿事件丢失",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建故障注入器
			injector := NewFaultInjector()
			injector.AddFault("message-loss", &FaultConfig{
				Type:        FaultTypeMessageLoss,
				Probability: tt.lossRate,
				TargetStep:  string(tt.eventType),
			})

			// 创建 mock 事件发布器
			mockPublisher := &trackingEventPublisher{}
			chaosPublisher := NewChaosEventPublisher(mockPublisher, injector)

			// 发布多个测试事件
			ctx := context.Background()
			numEvents := 10
			var lostEvents int

			for i := 0; i < numEvents; i++ {
				event := &saga.SagaEvent{
					ID:     string(rune(i)),
					SagaID: "saga-1",
					Type:   tt.eventType,
				}

				err := chaosPublisher.PublishEvent(ctx, event)
				if err != nil && errors.Is(err, ErrMessageLoss) {
					lostEvents++
				}
			}

			// 验证消息丢失
			actualPublished := mockPublisher.GetPublishCount()
			t.Logf("发送 %d 个事件，实际发布 %d 个，丢失 %d 个", numEvents, actualPublished, lostEvents)

			if tt.expectLoss && lostEvents == 0 {
				t.Error("期望有消息丢失，但所有消息都成功发布")
			}

			// 验证注入统计
			stats := injector.GetStats()
			t.Logf("故障注入统计: %+v", stats)
		})
	}
}

// TestMessageLoss_EventOrdering tests that message loss doesn't violate event ordering guarantees.
func TestMessageLoss_EventOrdering(t *testing.T) {
	// 创建故障注入器，30% 消息丢失率
	injector := NewFaultInjector()
	injector.AddFault("random-message-loss", &FaultConfig{
		Type:        FaultTypeMessageLoss,
		Probability: 0.3,
	})

	// 创建追踪事件发布器
	mockPublisher := &trackingEventPublisher{}
	chaosPublisher := NewChaosEventPublisher(mockPublisher, injector)

	// 发布一系列有序事件
	ctx := context.Background()
	events := []saga.SagaEventType{
		saga.EventSagaStarted,
		saga.EventSagaStepStarted,
		saga.EventSagaStepCompleted,
		saga.EventSagaStepStarted,
		saga.EventSagaStepCompleted,
		saga.EventSagaCompleted,
	}

	var publishedSequence []saga.SagaEventType
	for i, eventType := range events {
		event := &saga.SagaEvent{
			ID:     string(rune(i)),
			SagaID: "saga-ordering-test",
			Type:   eventType,
		}

		err := chaosPublisher.PublishEvent(ctx, event)
		if err == nil {
			publishedSequence = append(publishedSequence, eventType)
		}
	}

	t.Logf("原始序列: %d 个事件", len(events))
	t.Logf("实际发布: %d 个事件", len(publishedSequence))
	t.Logf("丢失: %d 个事件", len(events)-len(publishedSequence))

	// 验证发布的事件保持相对顺序
	if len(publishedSequence) > 0 {
		// 第一个应该是 Started
		if publishedSequence[0] != saga.EventSagaStarted {
			t.Errorf("期望第一个事件为 SagaStarted，实际为: %s", publishedSequence[0])
		}

		// 最后一个（如果完成事件没丢失）应该是 Completed
		if publishedSequence[len(publishedSequence)-1] == saga.EventSagaCompleted {
			t.Log("Saga 完成事件成功发布")
		}
	}

	// 验证注入统计
	stats := injector.GetStats()
	t.Logf("故障注入统计: %+v", stats)
}

// TestMessageLoss_RetryMechanism tests message publishing with retry mechanism.
func TestMessageLoss_RetryMechanism(t *testing.T) {
	// 创建故障注入器，50% 消息丢失率
	injector := NewFaultInjector()
	injector.AddFault("retry-message-loss", &FaultConfig{
		Type:        FaultTypeMessageLoss,
		Probability: 0.5,
	})

	// 创建追踪事件发布器
	mockPublisher := &trackingEventPublisher{}
	chaosPublisher := NewChaosEventPublisher(mockPublisher, injector)

	// 发布事件，带重试机制
	ctx := context.Background()
	event := &saga.SagaEvent{
		ID:     "event-with-retry",
		SagaID: "saga-retry-test",
		Type:   saga.EventSagaStarted,
	}

	// 重试策略：最多重试 5 次
	maxRetries := 5
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := chaosPublisher.PublishEvent(ctx, event)
		if err == nil {
			t.Logf("事件在第 %d 次尝试后成功发布", attempt+1)
			break
		}
		lastErr = err
		t.Logf("第 %d 次发布尝试失败: %v", attempt+1, err)
		time.Sleep(100 * time.Millisecond)
	}

	if lastErr != nil {
		t.Logf("经过 %d 次重试后，事件仍未成功发布", maxRetries)
	}

	// 验证最终发布结果
	if mockPublisher.GetPublishCount() > 0 {
		t.Log("事件最终成功发布")
	} else {
		t.Log("事件未能成功发布")
	}
}

// TestMessageLoss_IdempotencyCheck tests idempotency when messages are retried after loss.
func TestMessageLoss_IdempotencyCheck(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	injector.Disable() // 初始禁用

	// 创建去重事件发布器
	mockPublisher := &deduplicatingEventPublisher{
		seen: make(map[string]bool),
	}
	chaosPublisher := NewChaosEventPublisher(mockPublisher, injector)

	ctx := context.Background()
	event := &saga.SagaEvent{
		ID:     "idempotent-event-1",
		SagaID: "saga-idempotency-test",
		Type:   saga.EventSagaStarted,
	}

	// 第一次发布（成功）
	err := chaosPublisher.PublishEvent(ctx, event)
	if err != nil {
		t.Fatalf("第一次发布失败: %v", err)
	}

	// 启用消息丢失
	injector.AddFault("idempotency-loss", &FaultConfig{
		Type:        FaultTypeMessageLoss,
		Probability: 1.0,
	})
	injector.Enable()

	// 第二次发布（会丢失）
	err = chaosPublisher.PublishEvent(ctx, event)
	if !errors.Is(err, ErrMessageLoss) {
		t.Errorf("期望消息丢失错误，实际为: %v", err)
	}

	// 禁用消息丢失并重试
	injector.Disable()
	err = chaosPublisher.PublishEvent(ctx, event)
	if err != nil {
		t.Fatalf("重试发布失败: %v", err)
	}

	// 验证去重
	if mockPublisher.GetPublishCount() != 1 {
		t.Errorf("期望去重后发布计数为 1，实际为: %d", mockPublisher.GetPublishCount())
	}

	if mockPublisher.GetDuplicateCount() != 1 {
		t.Errorf("期望重复计数为 1，实际为: %d", mockPublisher.GetDuplicateCount())
	}

	t.Log("幂等性检查通过：重复事件被正确识别和过滤")
}

// TestMessageLoss_DeadLetterQueue tests that lost messages can be recovered from DLQ.
func TestMessageLoss_DeadLetterQueue(t *testing.T) {
	// 创建故障注入器，100% 消息丢失率
	injector := NewFaultInjector()
	injector.AddFault("dlq-message-loss", &FaultConfig{
		Type:        FaultTypeMessageLoss,
		Probability: 1.0,
	})

	// 创建带 DLQ 的事件发布器
	dlq := &deadLetterQueue{
		messages: make([]*saga.SagaEvent, 0),
	}
	mockPublisher := &dlqEventPublisher{
		dlq: dlq,
	}
	chaosPublisher := NewChaosEventPublisher(mockPublisher, injector)

	// 发布多个事件（都会丢失并进入 DLQ）
	ctx := context.Background()
	events := []*saga.SagaEvent{
		{ID: "event-1", SagaID: "saga-1", Type: saga.EventSagaStarted},
		{ID: "event-2", SagaID: "saga-1", Type: saga.EventSagaStepStarted},
		{ID: "event-3", SagaID: "saga-1", Type: saga.EventSagaStepCompleted},
	}

	for _, event := range events {
		err := chaosPublisher.PublishEvent(ctx, event)
		if err != nil {
			// 模拟：失败的消息进入 DLQ
			dlq.Add(event)
		}
	}

	// 验证 DLQ 中的消息
	dlqMessages := dlq.GetMessages()
	if len(dlqMessages) != len(events) {
		t.Errorf("期望 DLQ 中有 %d 条消息，实际为: %d", len(events), len(dlqMessages))
	}

	t.Logf("DLQ 中有 %d 条待恢复的消息", len(dlqMessages))

	// 禁用故障注入并重新发布 DLQ 中的消息
	injector.Disable()
	var recovered int
	for _, msg := range dlqMessages {
		err := chaosPublisher.PublishEvent(ctx, msg)
		if err == nil {
			recovered++
		}
	}

	t.Logf("从 DLQ 恢复并成功发布 %d 条消息", recovered)

	if recovered != len(events) {
		t.Errorf("期望恢复 %d 条消息，实际恢复: %d", len(events), recovered)
	}
}

// trackingEventPublisher tracks the number of published events.
type trackingEventPublisher struct {
	publishCount atomic.Int32
}

func (p *trackingEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	p.publishCount.Add(1)
	return nil
}

func (p *trackingEventPublisher) GetPublishCount() int {
	return int(p.publishCount.Load())
}

func (p *trackingEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	return nil, nil
}

func (p *trackingEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	return nil
}

func (p *trackingEventPublisher) Close() error {
	return nil
}

// deduplicatingEventPublisher deduplicates events based on ID.
type deduplicatingEventPublisher struct {
	seen           map[string]bool
	publishCount   int
	duplicateCount int
}

func (p *deduplicatingEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	if p.seen[event.ID] {
		p.duplicateCount++
		return nil // Already published
	}
	p.seen[event.ID] = true
	p.publishCount++
	return nil
}

func (p *deduplicatingEventPublisher) GetPublishCount() int {
	return p.publishCount
}

func (p *deduplicatingEventPublisher) GetDuplicateCount() int {
	return p.duplicateCount
}

func (p *deduplicatingEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	return nil, nil
}

func (p *deduplicatingEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	return nil
}

func (p *deduplicatingEventPublisher) Close() error {
	return nil
}

// deadLetterQueue simulates a dead letter queue for failed messages.
type deadLetterQueue struct {
	messages []*saga.SagaEvent
}

func (q *deadLetterQueue) Add(event *saga.SagaEvent) {
	q.messages = append(q.messages, event)
}

func (q *deadLetterQueue) GetMessages() []*saga.SagaEvent {
	return q.messages
}

// dlqEventPublisher publishes events with DLQ support.
type dlqEventPublisher struct {
	dlq          *deadLetterQueue
	publishCount int
}

func (p *dlqEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	p.publishCount++
	return nil
}

func (p *dlqEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	return nil, nil
}

func (p *dlqEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	return nil
}

func (p *dlqEventPublisher) Close() error {
	return nil
}


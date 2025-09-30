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

package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// mockPublisher 模拟消息发布器
type mockPublisher struct {
	mu             sync.Mutex
	published      []*messaging.Message
	shouldFail     bool
	failAfterCount int
	callCount      int
}

func (m *mockPublisher) Publish(ctx context.Context, message *messaging.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++
	if m.shouldFail {
		if m.failAfterCount == 0 || m.callCount <= m.failAfterCount {
			return errors.New("mock publish error")
		}
	}

	m.published = append(m.published, message)
	return nil
}

func (m *mockPublisher) PublishBatch(ctx context.Context, messages []*messaging.Message) error {
	return nil
}

func (m *mockPublisher) PublishWithConfirm(ctx context.Context, message *messaging.Message) (*messaging.PublishConfirmation, error) {
	return nil, nil
}

func (m *mockPublisher) PublishAsync(ctx context.Context, message *messaging.Message, callback messaging.PublishCallback) error {
	return nil
}

func (m *mockPublisher) BeginTransaction(ctx context.Context) (messaging.Transaction, error) {
	return nil, nil
}

func (m *mockPublisher) Flush(ctx context.Context) error {
	return nil
}

func (m *mockPublisher) Close() error {
	return nil
}

func (m *mockPublisher) GetMetrics() *messaging.PublisherMetrics {
	return nil
}

func (m *mockPublisher) GetPublished() []*messaging.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*messaging.Message{}, m.published...)
}

func (m *mockPublisher) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = nil
	m.callCount = 0
}

func TestProcessor_ProcessOnce(t *testing.T) {
	tests := []struct {
		name           string
		entries        []*OutboxEntry
		expectPublish  int
		publisherFails bool
	}{
		{
			name:          "empty outbox",
			entries:       []*OutboxEntry{},
			expectPublish: 0,
		},
		{
			name: "single entry",
			entries: []*OutboxEntry{
				{
					ID:          "entry-1",
					AggregateID: "agg-1",
					EventType:   "test.event",
					Topic:       "test.topic",
					Payload:     []byte(`{"test": "data"}`),
					Headers:     map[string]string{"key": "value"},
					CreatedAt:   time.Now(),
				},
			},
			expectPublish: 1,
		},
		{
			name: "multiple entries",
			entries: []*OutboxEntry{
				{
					ID:          "entry-1",
					AggregateID: "agg-1",
					EventType:   "test.event1",
					Topic:       "test.topic",
					Payload:     []byte(`{"test": "data1"}`),
					CreatedAt:   time.Now(),
				},
				{
					ID:          "entry-2",
					AggregateID: "agg-2",
					EventType:   "test.event2",
					Topic:       "test.topic",
					Payload:     []byte(`{"test": "data2"}`),
					CreatedAt:   time.Now(),
				},
			},
			expectPublish: 2,
		},
		{
			name: "publisher fails",
			entries: []*OutboxEntry{
				{
					ID:          "entry-1",
					AggregateID: "agg-1",
					EventType:   "test.event",
					Topic:       "test.topic",
					Payload:     []byte(`{"test": "data"}`),
					CreatedAt:   time.Now(),
				},
			},
			expectPublish:  0,
			publisherFails: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewInMemoryStorage()
			mockPub := &mockPublisher{shouldFail: tt.publisherFails}

			// 保存测试条目
			for _, entry := range tt.entries {
				if err := storage.Save(context.Background(), entry); err != nil {
					t.Fatalf("failed to save entry: %v", err)
				}
			}

			config := ProcessorConfig{
				BatchSize:  100,
				MaxRetries: 3,
			}

			proc := NewProcessor(storage, mockPub, config)

			// 执行处理
			if err := proc.ProcessOnce(context.Background()); err != nil {
				t.Fatalf("ProcessOnce failed: %v", err)
			}

			// 验证发布的消息数量
			published := mockPub.GetPublished()
			if len(published) != tt.expectPublish {
				t.Errorf("expected %d published messages, got %d", tt.expectPublish, len(published))
			}

			// 验证已处理的条目
			if !tt.publisherFails {
				unprocessed, _ := storage.FetchUnprocessed(context.Background(), 100)
				if len(unprocessed) != 0 {
					t.Errorf("expected 0 unprocessed entries, got %d", len(unprocessed))
				}
			}
		})
	}
}

func TestProcessor_MaxRetries(t *testing.T) {
	storage := NewInMemoryStorage()
	mockPub := &mockPublisher{shouldFail: true}

	entry := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data"}`),
		CreatedAt:   time.Now(),
	}

	if err := storage.Save(context.Background(), entry); err != nil {
		t.Fatalf("failed to save entry: %v", err)
	}

	config := ProcessorConfig{
		BatchSize:  100,
		MaxRetries: 3,
	}

	proc := NewProcessor(storage, mockPub, config)

	// 第一次处理 - 失败，重试计数+1
	_ = proc.ProcessOnce(context.Background())

	entries, _ := storage.FetchUnprocessed(context.Background(), 100)
	if len(entries) != 1 || entries[0].RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", entries[0].RetryCount)
	}

	// 第二次处理 - 失败，重试计数+1
	_ = proc.ProcessOnce(context.Background())

	entries, _ = storage.FetchUnprocessed(context.Background(), 100)
	if len(entries) != 1 || entries[0].RetryCount != 2 {
		t.Errorf("expected retry count 2, got %d", entries[0].RetryCount)
	}

	// 第三次处理 - 失败，重试计数+1
	_ = proc.ProcessOnce(context.Background())

	entries, _ = storage.FetchUnprocessed(context.Background(), 100)
	if len(entries) != 1 || entries[0].RetryCount != 3 {
		t.Errorf("expected retry count 3, got %d", entries[0].RetryCount)
	}

	// 第四次处理 - 应该标记为失败，不再重试
	_ = proc.ProcessOnce(context.Background())

	entries, _ = storage.FetchUnprocessed(context.Background(), 100)
	if len(entries) != 0 {
		t.Errorf("expected 0 unprocessed entries after max retries, got %d", len(entries))
	}

	// 验证条目被标记为失败
	all := storage.GetAll()
	if len(all) != 1 || all[0].ProcessedAt == nil {
		t.Error("entry should be marked as failed")
	}
}

func TestProcessor_StartStop(t *testing.T) {
	storage := NewInMemoryStorage()
	mockPub := &mockPublisher{}

	config := ProcessorConfig{
		PollInterval: 100 * time.Millisecond,
		BatchSize:    100,
		MaxRetries:   3,
	}

	proc := NewProcessor(storage, mockPub, config)

	ctx := context.Background()

	// 启动处理器
	if err := proc.Start(ctx); err != nil {
		t.Fatalf("failed to start processor: %v", err)
	}

	// 重复启动应该失败
	if err := proc.Start(ctx); err != ErrProcessorAlreadyStarted {
		t.Errorf("expected ErrProcessorAlreadyStarted, got %v", err)
	}

	// 添加一些条目
	entry := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data"}`),
		CreatedAt:   time.Now(),
	}
	if err := storage.Save(ctx, entry); err != nil {
		t.Fatalf("failed to save entry: %v", err)
	}

	// 等待处理器处理
	time.Sleep(300 * time.Millisecond)

	// 停止处理器
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := proc.Stop(stopCtx); err != nil {
		t.Fatalf("failed to stop processor: %v", err)
	}

	// 验证消息被发布
	published := mockPub.GetPublished()
	if len(published) == 0 {
		t.Error("expected at least one published message")
	}
}

func TestPublisher_SaveForPublish(t *testing.T) {
	tests := []struct {
		name        string
		entry       *OutboxEntry
		expectError bool
	}{
		{
			name: "valid entry",
			entry: &OutboxEntry{
				ID:          "entry-1",
				AggregateID: "agg-1",
				EventType:   "test.event",
				Topic:       "test.topic",
				Payload:     []byte(`{"test": "data"}`),
			},
			expectError: false,
		},
		{
			name:        "nil entry",
			entry:       nil,
			expectError: true,
		},
		{
			name: "empty ID",
			entry: &OutboxEntry{
				AggregateID: "agg-1",
				EventType:   "test.event",
				Topic:       "test.topic",
				Payload:     []byte(`{"test": "data"}`),
			},
			expectError: true,
		},
		{
			name: "empty topic",
			entry: &OutboxEntry{
				ID:          "entry-1",
				AggregateID: "agg-1",
				EventType:   "test.event",
				Payload:     []byte(`{"test": "data"}`),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewInMemoryStorage()
			pub := NewPublisher(storage)

			err := pub.SaveForPublish(context.Background(), tt.entry)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				// 验证条目已保存
				entries, _ := storage.FetchUnprocessed(context.Background(), 100)
				if len(entries) != 1 {
					t.Errorf("expected 1 entry, got %d", len(entries))
				}
			}
		})
	}
}

func TestPublisher_SaveWithTransaction(t *testing.T) {
	storage := NewInMemoryStorage()
	pub := NewPublisher(storage)

	// 开始事务
	tx, err := storage.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	entry := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data"}`),
	}

	// 保存到事务中
	if err := pub.SaveWithTransaction(context.Background(), tx, entry); err != nil {
		t.Fatalf("failed to save with transaction: %v", err)
	}

	// 提交前，条目不应该可见
	entries, _ := storage.FetchUnprocessed(context.Background(), 100)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries before commit, got %d", len(entries))
	}

	// 提交事务
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("failed to commit transaction: %v", err)
	}

	// 提交后，条目应该可见
	entries, _ = storage.FetchUnprocessed(context.Background(), 100)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry after commit, got %d", len(entries))
	}
}

func TestPublisher_SaveWithTransaction_Rollback(t *testing.T) {
	storage := NewInMemoryStorage()
	pub := NewPublisher(storage)

	// 开始事务
	tx, err := storage.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	entry := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data"}`),
	}

	// 保存到事务中
	if err := pub.SaveWithTransaction(context.Background(), tx, entry); err != nil {
		t.Fatalf("failed to save with transaction: %v", err)
	}

	// 回滚事务
	if err := tx.Rollback(context.Background()); err != nil {
		t.Fatalf("failed to rollback transaction: %v", err)
	}

	// 回滚后，条目不应该可见
	entries, _ := storage.FetchUnprocessed(context.Background(), 100)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after rollback, got %d", len(entries))
	}
}

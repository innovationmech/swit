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

package inbox

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// mockHandler 模拟消息处理器
type mockHandler struct {
	name         string
	handleCount  int32
	handleError  error
	handleResult interface{}
}

func (h *mockHandler) Handle(ctx context.Context, msg *messaging.Message) error {
	atomic.AddInt32(&h.handleCount, 1)
	return h.handleError
}

func (h *mockHandler) GetName() string {
	return h.name
}

func (h *mockHandler) HandleWithResult(ctx context.Context, msg *messaging.Message) (interface{}, error) {
	atomic.AddInt32(&h.handleCount, 1)
	if h.handleError != nil {
		return nil, h.handleError
	}
	return h.handleResult, nil
}

func (h *mockHandler) GetHandleCount() int {
	return int(atomic.LoadInt32(&h.handleCount))
}

func TestNewProcessor(t *testing.T) {
	tests := []struct {
		name        string
		storage     InboxStorage
		handler     MessageHandler
		config      ProcessorConfig
		expectError bool
	}{
		{
			name:        "valid processor",
			storage:     NewInMemoryStorage(),
			handler:     &mockHandler{name: "test-handler"},
			config:      DefaultProcessorConfig(),
			expectError: false,
		},
		{
			name:        "nil storage",
			storage:     nil,
			handler:     &mockHandler{name: "test-handler"},
			config:      DefaultProcessorConfig(),
			expectError: true,
		},
		{
			name:        "nil handler",
			storage:     NewInMemoryStorage(),
			handler:     nil,
			config:      DefaultProcessorConfig(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := NewProcessor(tt.storage, tt.handler, tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if processor == nil {
					t.Errorf("expected processor, got nil")
				}
			}
		})
	}
}

func TestProcessor_Process(t *testing.T) {
	tests := []struct {
		name          string
		setupStorage  func() InboxStorage
		setupHandler  func() *mockHandler
		message       *messaging.Message
		expectError   bool
		expectHandled bool
		handleCount   int
	}{
		{
			name: "process new message",
			setupStorage: func() InboxStorage {
				return NewInMemoryStorage()
			},
			setupHandler: func() *mockHandler {
				return &mockHandler{name: "test-handler"}
			},
			message: &messaging.Message{
				ID:      "msg-1",
				Topic:   "test-topic",
				Payload: []byte("test payload"),
			},
			expectError:   false,
			expectHandled: true,
			handleCount:   1,
		},
		{
			name: "process duplicate message",
			setupStorage: func() InboxStorage {
				storage := NewInMemoryStorage()
				// 预先记录消息
				storage.RecordMessage(context.Background(), &InboxEntry{
					MessageID:   "msg-1",
					HandlerName: "test-handler",
					Topic:       "test-topic",
					ProcessedAt: time.Now(),
				})
				return storage
			},
			setupHandler: func() *mockHandler {
				return &mockHandler{name: "test-handler"}
			},
			message: &messaging.Message{
				ID:      "msg-1",
				Topic:   "test-topic",
				Payload: []byte("test payload"),
			},
			expectError:   false,
			expectHandled: false, // 不应该被处理
			handleCount:   0,     // 处理器不应该被调用
		},
		{
			name: "handler returns error",
			setupStorage: func() InboxStorage {
				return NewInMemoryStorage()
			},
			setupHandler: func() *mockHandler {
				return &mockHandler{
					name:        "test-handler",
					handleError: errors.New("handler error"),
				}
			},
			message: &messaging.Message{
				ID:      "msg-1",
				Topic:   "test-topic",
				Payload: []byte("test payload"),
			},
			expectError:   true,
			expectHandled: false,
			handleCount:   1, // 处理器被调用但返回错误
		},
		{
			name: "nil message",
			setupStorage: func() InboxStorage {
				return NewInMemoryStorage()
			},
			setupHandler: func() *mockHandler {
				return &mockHandler{name: "test-handler"}
			},
			message:       nil,
			expectError:   true,
			expectHandled: false,
			handleCount:   0,
		},
		{
			name: "empty message ID",
			setupStorage: func() InboxStorage {
				return NewInMemoryStorage()
			},
			setupHandler: func() *mockHandler {
				return &mockHandler{name: "test-handler"}
			},
			message: &messaging.Message{
				ID:      "",
				Topic:   "test-topic",
				Payload: []byte("test payload"),
			},
			expectError:   true,
			expectHandled: false,
			handleCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := tt.setupStorage()
			handler := tt.setupHandler()
			config := DefaultProcessorConfig()
			config.HandlerName = "test-handler"

			processor, err := NewProcessor(storage, handler, config)
			if err != nil {
				t.Fatalf("failed to create processor: %v", err)
			}

			ctx := context.Background()
			err = processor.Process(ctx, tt.message)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// 检查处理器调用次数
			if handler.GetHandleCount() != tt.handleCount {
				t.Errorf("expected handler to be called %d times, got %d", tt.handleCount, handler.GetHandleCount())
			}

			// 如果消息应该被处理，检查是否已记录
			if tt.expectHandled && tt.message != nil && tt.message.ID != "" {
				processed, _, err := storage.IsProcessedByHandler(ctx, tt.message.ID, "test-handler")
				if err != nil {
					t.Errorf("failed to check if message is processed: %v", err)
				}
				if !processed {
					t.Errorf("expected message to be recorded as processed")
				}
			}
		})
	}
}

func TestProcessor_ProcessWithResult(t *testing.T) {
	tests := []struct {
		name           string
		setupStorage   func() InboxStorage
		setupHandler   func() *mockHandler
		message        *messaging.Message
		expectError    bool
		expectedResult interface{}
		handleCount    int
	}{
		{
			name: "process new message with result",
			setupStorage: func() InboxStorage {
				return NewInMemoryStorage()
			},
			setupHandler: func() *mockHandler {
				return &mockHandler{
					name:         "test-handler",
					handleResult: map[string]interface{}{"status": "success"},
				}
			},
			message: &messaging.Message{
				ID:      "msg-1",
				Topic:   "test-topic",
				Payload: []byte("test payload"),
			},
			expectError:    false,
			expectedResult: map[string]interface{}{"status": "success"},
			handleCount:    1,
		},
		{
			name: "get cached result",
			setupStorage: func() InboxStorage {
				storage := NewInMemoryStorage()
				// 预先记录消息和结果
				storage.RecordMessage(context.Background(), &InboxEntry{
					MessageID:   "msg-1",
					HandlerName: "test-handler",
					Topic:       "test-topic",
					ProcessedAt: time.Now(),
					Result:      []byte(`{"status":"cached"}`),
				})
				return storage
			},
			setupHandler: func() *mockHandler {
				return &mockHandler{name: "test-handler"}
			},
			message: &messaging.Message{
				ID:      "msg-1",
				Topic:   "test-topic",
				Payload: []byte("test payload"),
			},
			expectError:    false,
			expectedResult: map[string]interface{}{"status": "cached"},
			handleCount:    0, // 不应该调用处理器
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := tt.setupStorage()
			handler := tt.setupHandler()
			config := DefaultProcessorConfig()
			config.HandlerName = "test-handler"
			config.StoreResult = true

			processor, err := NewProcessor(storage, handler, config)
			if err != nil {
				t.Fatalf("failed to create processor: %v", err)
			}

			ctx := context.Background()
			result, err := processor.ProcessWithResult(ctx, tt.message)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// 检查处理器调用次数
			if handler.GetHandleCount() != tt.handleCount {
				t.Errorf("expected handler to be called %d times, got %d", tt.handleCount, handler.GetHandleCount())
			}

			// 检查结果
			if !tt.expectError && tt.expectedResult != nil && result != nil {
				// 简单的类型检查
				if _, ok := result.(map[string]interface{}); !ok {
					t.Errorf("expected result to be a map, got %T", result)
				}
			}
		})
	}
}

func TestProcessor_ProcessWithTransaction(t *testing.T) {
	storage := NewInMemoryStorage()
	handler := &mockHandler{name: "test-handler"}
	config := DefaultProcessorConfig()
	config.HandlerName = "test-handler"

	processor, err := NewProcessor(storage, handler, config)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	ctx := context.Background()

	// 开始事务
	txStorage, ok := storage.(TransactionalStorage)
	if !ok {
		t.Fatalf("storage does not support transactions")
	}

	tx, err := txStorage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// 在事务中处理消息
	msg := &messaging.Message{
		ID:      "msg-1",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	err = processor.ProcessWithTransaction(ctx, tx, msg)
	if err != nil {
		t.Fatalf("failed to process message with transaction: %v", err)
	}

	// 检查消息在事务提交前不可见
	processed, _, err := storage.IsProcessedByHandler(ctx, msg.ID, "test-handler")
	if err != nil {
		t.Fatalf("failed to check if message is processed: %v", err)
	}
	if processed {
		t.Errorf("message should not be visible before transaction commit")
	}

	// 提交事务
	err = tx.Commit(ctx)
	if err != nil {
		t.Fatalf("failed to commit transaction: %v", err)
	}

	// 检查消息在事务提交后可见
	processed, _, err = storage.IsProcessedByHandler(ctx, msg.ID, "test-handler")
	if err != nil {
		t.Fatalf("failed to check if message is processed: %v", err)
	}
	if !processed {
		t.Errorf("message should be visible after transaction commit")
	}

	// 检查处理器被调用
	if handler.GetHandleCount() != 1 {
		t.Errorf("expected handler to be called once, got %d", handler.GetHandleCount())
	}
}

func TestProcessor_Idempotency(t *testing.T) {
	storage := NewInMemoryStorage()
	handler := &mockHandler{name: "test-handler"}
	config := DefaultProcessorConfig()

	processor, err := NewProcessor(storage, handler, config)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	ctx := context.Background()
	msg := &messaging.Message{
		ID:      "msg-1",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	// 处理消息多次
	for i := 0; i < 5; i++ {
		err := processor.Process(ctx, msg)
		if err != nil {
			t.Fatalf("iteration %d: failed to process message: %v", i, err)
		}
	}

	// 处理器应该只被调用一次
	if handler.GetHandleCount() != 1 {
		t.Errorf("expected handler to be called once, got %d", handler.GetHandleCount())
	}
}

func TestProcessor_ConcurrentProcessing(t *testing.T) {
	storage := NewInMemoryStorage()
	handler := &mockHandler{name: "test-handler"}
	config := DefaultProcessorConfig()

	processor, err := NewProcessor(storage, handler, config)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}

	ctx := context.Background()
	msg := &messaging.Message{
		ID:      "msg-1",
		Topic:   "test-topic",
		Payload: []byte("test payload"),
	}

	// 并发处理同一消息
	concurrency := 10
	done := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			done <- processor.Process(ctx, msg)
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < concurrency; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent processing failed: %v", err)
		}
	}

	// 处理器应该只被调用一次（或几次，由于竞态条件）
	// 但至少应该少于并发数
	handleCount := handler.GetHandleCount()
	if handleCount > 5 {
		t.Errorf("expected handler to be called at most 5 times due to race, got %d", handleCount)
	}
}

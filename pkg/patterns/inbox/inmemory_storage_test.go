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
	"testing"
	"time"
)

func TestInMemoryStorage_RecordMessage(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	entry := &InboxEntry{
		MessageID:   "msg-1",
		HandlerName: "test-handler",
		Topic:       "test-topic",
		ProcessedAt: time.Now(),
		Result:      []byte("test result"),
		Metadata:    map[string]string{"key": "value"},
	}

	err := storage.RecordMessage(ctx, entry)
	if err != nil {
		t.Fatalf("failed to record message: %v", err)
	}

	// 检查消息是否已记录
	processed, retrievedEntry, err := storage.IsProcessed(ctx, "msg-1")
	if err != nil {
		t.Fatalf("failed to check if message is processed: %v", err)
	}

	if !processed {
		t.Errorf("expected message to be processed")
	}

	if retrievedEntry == nil {
		t.Fatalf("expected to retrieve entry, got nil")
	}

	if retrievedEntry.MessageID != entry.MessageID {
		t.Errorf("expected MessageID %s, got %s", entry.MessageID, retrievedEntry.MessageID)
	}
}

func TestInMemoryStorage_RecordMessage_NilEntry(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	err := storage.RecordMessage(ctx, nil)
	if err == nil {
		t.Errorf("expected error for nil entry, got nil")
	}
}

func TestInMemoryStorage_RecordMessage_EmptyMessageID(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	entry := &InboxEntry{
		MessageID: "",
		Topic:     "test-topic",
	}

	err := storage.RecordMessage(ctx, entry)
	if err == nil {
		t.Errorf("expected error for empty MessageID, got nil")
	}
}

func TestInMemoryStorage_IsProcessed(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// 检查不存在的消息
	processed, entry, err := storage.IsProcessed(ctx, "non-existent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Errorf("expected message to not be processed")
	}
	if entry != nil {
		t.Errorf("expected nil entry for non-existent message")
	}

	// 记录消息
	testEntry := &InboxEntry{
		MessageID:   "msg-1",
		HandlerName: "test-handler",
		Topic:       "test-topic",
		ProcessedAt: time.Now(),
	}
	err = storage.RecordMessage(ctx, testEntry)
	if err != nil {
		t.Fatalf("failed to record message: %v", err)
	}

	// 检查已存在的消息
	processed, entry, err = storage.IsProcessed(ctx, "msg-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Errorf("expected message to be processed")
	}
	if entry == nil {
		t.Errorf("expected entry for processed message")
	}
}

func TestInMemoryStorage_IsProcessedByHandler(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// 记录消息
	entry := &InboxEntry{
		MessageID:   "msg-1",
		HandlerName: "handler-1",
		Topic:       "test-topic",
		ProcessedAt: time.Now(),
	}
	err := storage.RecordMessage(ctx, entry)
	if err != nil {
		t.Fatalf("failed to record message: %v", err)
	}

	// 检查正确的处理器
	processed, _, err := storage.IsProcessedByHandler(ctx, "msg-1", "handler-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Errorf("expected message to be processed by handler-1")
	}

	// 检查不同的处理器
	processed, _, err = storage.IsProcessedByHandler(ctx, "msg-1", "handler-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Errorf("expected message to not be processed by handler-2")
	}

	// 检查不存在的消息
	processed, _, err = storage.IsProcessedByHandler(ctx, "non-existent", "handler-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Errorf("expected non-existent message to not be processed")
	}
}

func TestInMemoryStorage_DeleteBefore(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	now := time.Now()

	// 记录旧消息
	oldEntry := &InboxEntry{
		MessageID:   "old-msg",
		HandlerName: "test-handler",
		Topic:       "test-topic",
		ProcessedAt: now.Add(-48 * time.Hour),
	}
	err := storage.RecordMessage(ctx, oldEntry)
	if err != nil {
		t.Fatalf("failed to record old message: %v", err)
	}

	// 记录新消息
	newEntry := &InboxEntry{
		MessageID:   "new-msg",
		HandlerName: "test-handler",
		Topic:       "test-topic",
		ProcessedAt: now,
	}
	err = storage.RecordMessage(ctx, newEntry)
	if err != nil {
		t.Fatalf("failed to record new message: %v", err)
	}

	// 删除 24 小时前的消息
	err = storage.DeleteBefore(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("failed to delete old messages: %v", err)
	}

	// 检查旧消息已被删除
	processed, _, err := storage.IsProcessed(ctx, "old-msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Errorf("expected old message to be deleted")
	}

	// 检查新消息仍然存在
	processed, _, err = storage.IsProcessed(ctx, "new-msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Errorf("expected new message to still exist")
	}
}

func TestInMemoryStorage_DeleteByMessageID(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// 记录多个处理器的消息
	entry1 := &InboxEntry{
		MessageID:   "msg-1",
		HandlerName: "handler-1",
		Topic:       "test-topic",
		ProcessedAt: time.Now(),
	}
	err := storage.RecordMessage(ctx, entry1)
	if err != nil {
		t.Fatalf("failed to record message 1: %v", err)
	}

	entry2 := &InboxEntry{
		MessageID:   "msg-1",
		HandlerName: "handler-2",
		Topic:       "test-topic",
		ProcessedAt: time.Now(),
	}
	err = storage.RecordMessage(ctx, entry2)
	if err != nil {
		t.Fatalf("failed to record message 2: %v", err)
	}

	// 删除消息
	err = storage.DeleteByMessageID(ctx, "msg-1")
	if err != nil {
		t.Fatalf("failed to delete message: %v", err)
	}

	// 检查所有处理器的记录都被删除
	processed, _, err := storage.IsProcessedByHandler(ctx, "msg-1", "handler-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Errorf("expected message to be deleted for handler-1")
	}

	processed, _, err = storage.IsProcessedByHandler(ctx, "msg-1", "handler-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Errorf("expected message to be deleted for handler-2")
	}
}

func TestInMemoryStorage_Transaction(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	txStorage, ok := storage.(TransactionalStorage)
	if !ok {
		t.Fatalf("storage does not implement TransactionalStorage")
	}

	// 开始事务
	tx, err := txStorage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// 在事务中记录消息
	entry := &InboxEntry{
		MessageID:   "msg-1",
		HandlerName: "test-handler",
		Topic:       "test-topic",
		ProcessedAt: time.Now(),
	}
	err = tx.RecordMessage(ctx, entry)
	if err != nil {
		t.Fatalf("failed to record message in transaction: %v", err)
	}

	// 提交前消息不可见
	processed, _, err := storage.IsProcessed(ctx, "msg-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Errorf("message should not be visible before commit")
	}

	// 提交事务
	err = tx.Commit(ctx)
	if err != nil {
		t.Fatalf("failed to commit transaction: %v", err)
	}

	// 提交后消息可见
	processed, _, err = storage.IsProcessed(ctx, "msg-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Errorf("message should be visible after commit")
	}
}

func TestInMemoryStorage_Transaction_Rollback(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	txStorage, ok := storage.(TransactionalStorage)
	if !ok {
		t.Fatalf("storage does not implement TransactionalStorage")
	}

	// 开始事务
	tx, err := txStorage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// 在事务中记录消息
	entry := &InboxEntry{
		MessageID:   "msg-1",
		HandlerName: "test-handler",
		Topic:       "test-topic",
		ProcessedAt: time.Now(),
	}
	err = tx.RecordMessage(ctx, entry)
	if err != nil {
		t.Fatalf("failed to record message in transaction: %v", err)
	}

	// 回滚事务
	err = tx.Rollback(ctx)
	if err != nil {
		t.Fatalf("failed to rollback transaction: %v", err)
	}

	// 回滚后消息不可见
	processed, _, err := storage.IsProcessed(ctx, "msg-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Errorf("message should not be visible after rollback")
	}
}

func TestInMemoryStorage_Transaction_DoubleCommit(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	txStorage, ok := storage.(TransactionalStorage)
	if !ok {
		t.Fatalf("storage does not implement TransactionalStorage")
	}

	tx, err := txStorage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		t.Fatalf("failed to commit transaction: %v", err)
	}

	// 第二次提交应该失败
	err = tx.Commit(ctx)
	if err == nil {
		t.Errorf("expected error on double commit, got nil")
	}
}

func TestInMemoryStorage_Transaction_CommitAfterRollback(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	txStorage, ok := storage.(TransactionalStorage)
	if !ok {
		t.Fatalf("storage does not implement TransactionalStorage")
	}

	tx, err := txStorage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	err = tx.Rollback(ctx)
	if err != nil {
		t.Fatalf("failed to rollback transaction: %v", err)
	}

	// 回滚后提交应该失败
	err = tx.Commit(ctx)
	if err == nil {
		t.Errorf("expected error on commit after rollback, got nil")
	}
}

func TestInMemoryStorage_MultipleHandlers(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	messageID := "msg-1"

	// 记录第一个处理器处理的消息
	entry1 := &InboxEntry{
		MessageID:   messageID,
		HandlerName: "handler-1",
		Topic:       "test-topic",
		ProcessedAt: time.Now(),
	}
	err := storage.RecordMessage(ctx, entry1)
	if err != nil {
		t.Fatalf("failed to record message for handler-1: %v", err)
	}

	// 记录第二个处理器处理的消息
	entry2 := &InboxEntry{
		MessageID:   messageID,
		HandlerName: "handler-2",
		Topic:       "test-topic",
		ProcessedAt: time.Now(),
	}
	err = storage.RecordMessage(ctx, entry2)
	if err != nil {
		t.Fatalf("failed to record message for handler-2: %v", err)
	}

	// 检查两个处理器都记录了消息
	processed1, _, err := storage.IsProcessedByHandler(ctx, messageID, "handler-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed1 {
		t.Errorf("expected message to be processed by handler-1")
	}

	processed2, _, err := storage.IsProcessedByHandler(ctx, messageID, "handler-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed2 {
		t.Errorf("expected message to be processed by handler-2")
	}

	// 检查通用的 IsProcessed 方法
	processed, _, err := storage.IsProcessed(ctx, messageID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Errorf("expected message to be processed")
	}
}

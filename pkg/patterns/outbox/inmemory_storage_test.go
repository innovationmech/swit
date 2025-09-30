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
	"testing"
	"time"
)

func TestInMemoryStorage_Save(t *testing.T) {
	storage := NewInMemoryStorage()

	entry := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data"}`),
		Headers:     map[string]string{"key": "value"},
		CreatedAt:   time.Now(),
	}

	err := storage.Save(context.Background(), entry)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 验证条目已保存
	entries, err := storage.FetchUnprocessed(context.Background(), 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].ID != entry.ID {
		t.Errorf("expected ID %s, got %s", entry.ID, entries[0].ID)
	}
}

func TestInMemoryStorage_SaveBatch(t *testing.T) {
	storage := NewInMemoryStorage()

	entries := []*OutboxEntry{
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
	}

	err := storage.SaveBatch(context.Background(), entries)
	if err != nil {
		t.Fatalf("SaveBatch failed: %v", err)
	}

	// 验证条目已保存
	saved, err := storage.FetchUnprocessed(context.Background(), 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed failed: %v", err)
	}

	if len(saved) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(saved))
	}
}

func TestInMemoryStorage_FetchUnprocessed(t *testing.T) {
	storage := NewInMemoryStorage()

	// 保存3个条目
	for i := 1; i <= 3; i++ {
		entry := &OutboxEntry{
			ID:          "entry-" + string(rune('0'+i)),
			AggregateID: "agg-1",
			EventType:   "test.event",
			Topic:       "test.topic",
			Payload:     []byte(`{"test": "data"}`),
			CreatedAt:   time.Now(),
		}
		_ = storage.Save(context.Background(), entry)
	}

	// 标记一个为已处理
	_ = storage.MarkAsProcessed(context.Background(), "entry-2")

	// 获取未处理的条目
	unprocessed, err := storage.FetchUnprocessed(context.Background(), 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed failed: %v", err)
	}

	if len(unprocessed) != 2 {
		t.Fatalf("expected 2 unprocessed entries, got %d", len(unprocessed))
	}

	// 测试 limit
	limited, err := storage.FetchUnprocessed(context.Background(), 1)
	if err != nil {
		t.Fatalf("FetchUnprocessed failed: %v", err)
	}

	if len(limited) != 1 {
		t.Fatalf("expected 1 limited entry, got %d", len(limited))
	}
}

func TestInMemoryStorage_MarkAsProcessed(t *testing.T) {
	storage := NewInMemoryStorage()

	entry := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data"}`),
		CreatedAt:   time.Now(),
	}

	_ = storage.Save(context.Background(), entry)

	err := storage.MarkAsProcessed(context.Background(), entry.ID)
	if err != nil {
		t.Fatalf("MarkAsProcessed failed: %v", err)
	}

	// 验证条目已标记为已处理
	unprocessed, _ := storage.FetchUnprocessed(context.Background(), 10)
	if len(unprocessed) != 0 {
		t.Errorf("expected 0 unprocessed entries, got %d", len(unprocessed))
	}

	// 验证 ProcessedAt 已设置
	all := storage.GetAll()
	if len(all) != 1 || all[0].ProcessedAt == nil {
		t.Error("ProcessedAt should be set")
	}
}

func TestInMemoryStorage_MarkAsFailed(t *testing.T) {
	storage := NewInMemoryStorage()

	entry := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data"}`),
		CreatedAt:   time.Now(),
	}

	_ = storage.Save(context.Background(), entry)

	errMsg := "test error"
	err := storage.MarkAsFailed(context.Background(), entry.ID, errMsg)
	if err != nil {
		t.Fatalf("MarkAsFailed failed: %v", err)
	}

	// 验证条目已标记为失败
	all := storage.GetAll()
	if len(all) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(all))
	}

	if all[0].LastError != errMsg {
		t.Errorf("expected error %s, got %s", errMsg, all[0].LastError)
	}

	if all[0].ProcessedAt == nil {
		t.Error("ProcessedAt should be set for failed entry")
	}
}

func TestInMemoryStorage_IncrementRetry(t *testing.T) {
	storage := NewInMemoryStorage()

	entry := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data"}`),
		CreatedAt:   time.Now(),
		RetryCount:  0,
	}

	_ = storage.Save(context.Background(), entry)

	// 增加重试计数
	_ = storage.IncrementRetry(context.Background(), entry.ID)

	all := storage.GetAll()
	if len(all) != 1 || all[0].RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", all[0].RetryCount)
	}

	// 再次增加
	_ = storage.IncrementRetry(context.Background(), entry.ID)

	all = storage.GetAll()
	if len(all) != 1 || all[0].RetryCount != 2 {
		t.Errorf("expected retry count 2, got %d", all[0].RetryCount)
	}
}

func TestInMemoryStorage_Delete(t *testing.T) {
	storage := NewInMemoryStorage()

	entry := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data"}`),
		CreatedAt:   time.Now(),
	}

	_ = storage.Save(context.Background(), entry)

	err := storage.Delete(context.Background(), entry.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证条目已删除
	all := storage.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 entries after delete, got %d", len(all))
	}

	// 尝试删除不存在的条目
	err = storage.Delete(context.Background(), "non-existent")
	if err != ErrEntryNotFound {
		t.Errorf("expected ErrEntryNotFound, got %v", err)
	}
}

func TestInMemoryStorage_DeleteProcessedBefore(t *testing.T) {
	storage := NewInMemoryStorage()

	now := time.Now()

	// 保存并手动设置为2小时前已处理
	entry1 := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data1"}`),
		CreatedAt:   now.Add(-2 * time.Hour),
	}
	_ = storage.Save(context.Background(), entry1)
	// 手动设置 ProcessedAt 为过去时间
	storage.mu.Lock()
	processedAt1 := now.Add(-2 * time.Hour)
	storage.entries[entry1.ID].ProcessedAt = &processedAt1
	storage.mu.Unlock()

	// 保存但不标记为已处理
	entry2 := &OutboxEntry{
		ID:          "entry-2",
		AggregateID: "agg-2",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data2"}`),
		CreatedAt:   now.Add(-1 * time.Hour),
	}
	_ = storage.Save(context.Background(), entry2)

	// 保存并标记为现在处理
	entry3 := &OutboxEntry{
		ID:          "entry-3",
		AggregateID: "agg-3",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data3"}`),
		CreatedAt:   now,
	}
	_ = storage.Save(context.Background(), entry3)
	_ = storage.MarkAsProcessed(context.Background(), entry3.ID)

	// 删除1小时之前处理的条目
	before := now.Add(-30 * time.Minute)
	err := storage.DeleteProcessedBefore(context.Background(), before)
	if err != nil {
		t.Fatalf("DeleteProcessedBefore failed: %v", err)
	}

	// 验证只有 entry1 被删除
	all := storage.GetAll()
	if len(all) != 2 {
		t.Fatalf("expected 2 entries after cleanup, got %d", len(all))
	}

	// 验证 entry2 和 entry3 仍然存在
	found2 := false
	found3 := false
	for _, e := range all {
		if e.ID == "entry-2" {
			found2 = true
		}
		if e.ID == "entry-3" {
			found3 = true
		}
	}

	if !found2 || !found3 {
		t.Error("entry2 and entry3 should still exist")
	}
}

func TestInMemoryStorage_Transaction(t *testing.T) {
	storage := NewInMemoryStorage()

	tx, err := storage.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}

	entry := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data"}`),
		CreatedAt:   time.Now(),
	}

	// 在事务中保存
	err = tx.Save(context.Background(), entry)
	if err != nil {
		t.Fatalf("tx.Save failed: %v", err)
	}

	// 提交前不应该可见
	all := storage.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 entries before commit, got %d", len(all))
	}

	// 提交事务
	err = tx.Commit(context.Background())
	if err != nil {
		t.Fatalf("tx.Commit failed: %v", err)
	}

	// 提交后应该可见
	all = storage.GetAll()
	if len(all) != 1 {
		t.Errorf("expected 1 entry after commit, got %d", len(all))
	}

	// 尝试在已提交的事务上再次操作
	err = tx.Save(context.Background(), entry)
	if err != ErrTransactionNotActive {
		t.Errorf("expected ErrTransactionNotActive, got %v", err)
	}
}

func TestInMemoryStorage_Transaction_Rollback(t *testing.T) {
	storage := NewInMemoryStorage()

	tx, err := storage.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}

	entry := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data"}`),
		CreatedAt:   time.Now(),
	}

	// 在事务中保存
	err = tx.Save(context.Background(), entry)
	if err != nil {
		t.Fatalf("tx.Save failed: %v", err)
	}

	// 回滚事务
	err = tx.Rollback(context.Background())
	if err != nil {
		t.Fatalf("tx.Rollback failed: %v", err)
	}

	// 回滚后不应该可见
	all := storage.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 entries after rollback, got %d", len(all))
	}
}

func TestInMemoryStorage_CopyEntry(t *testing.T) {
	storage := NewInMemoryStorage()

	original := &OutboxEntry{
		ID:          "entry-1",
		AggregateID: "agg-1",
		EventType:   "test.event",
		Topic:       "test.topic",
		Payload:     []byte(`{"test": "data"}`),
		Headers:     map[string]string{"key": "value"},
		CreatedAt:   time.Now(),
		RetryCount:  1,
	}

	_ = storage.Save(context.Background(), original)

	// 修改原始对象
	original.Payload[0] = 'X'
	original.Headers["key"] = "modified"
	original.RetryCount = 99

	// 获取保存的副本
	entries, _ := storage.FetchUnprocessed(context.Background(), 10)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	saved := entries[0]

	// 验证保存的副本未被修改
	if saved.Payload[0] == 'X' {
		t.Error("payload should not be affected by external modification")
	}

	if saved.Headers["key"] == "modified" {
		t.Error("headers should not be affected by external modification")
	}

	if saved.RetryCount == 99 {
		t.Error("retry count should not be affected by external modification")
	}
}

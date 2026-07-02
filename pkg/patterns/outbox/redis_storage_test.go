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
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedisStorage(t *testing.T) *RedisStorage {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	storage, err := NewRedisStorage(client)
	if err != nil {
		t.Fatalf("failed to create redis storage: %v", err)
	}
	return storage
}

func TestNewRedisStorage_Validation(t *testing.T) {
	if _, err := NewRedisStorage(nil); err == nil {
		t.Error("expected error for nil client")
	}

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	if _, err := NewRedisStorage(client, WithRedisKeyPrefix("")); err == nil {
		t.Error("expected error for empty key prefix")
	}
	if _, err := NewRedisStorage(client, WithRedisKeyPrefix("custom")); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRedisStorage_SaveAndFetch(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	base := time.Now().Add(-time.Hour)
	entries := []*OutboxEntry{
		{ID: "id-2", Topic: "orders", CreatedAt: base.Add(time.Minute), Headers: map[string]string{"h": "2"}},
		{ID: "id-1", Topic: "orders", CreatedAt: base, Payload: []byte("payload-1")},
	}
	if err := storage.SaveBatch(ctx, entries); err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}

	fetched, err := storage.FetchUnprocessed(ctx, 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed() error = %v", err)
	}
	if len(fetched) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(fetched))
	}
	// 应按创建时间排序
	if fetched[0].ID != "id-1" || fetched[1].ID != "id-2" {
		t.Errorf("expected order [id-1, id-2], got [%s, %s]", fetched[0].ID, fetched[1].ID)
	}
	if string(fetched[0].Payload) != "payload-1" {
		t.Errorf("unexpected payload: %s", fetched[0].Payload)
	}
	if fetched[1].Headers["h"] != "2" {
		t.Errorf("unexpected headers: %v", fetched[1].Headers)
	}

	// limit 生效
	limited, err := storage.FetchUnprocessed(ctx, 1)
	if err != nil {
		t.Fatalf("FetchUnprocessed(1) error = %v", err)
	}
	if len(limited) != 1 || limited[0].ID != "id-1" {
		t.Errorf("expected [id-1], got %v", limited)
	}
}

func TestRedisStorage_Save_Validation(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	if err := storage.Save(ctx, nil); err == nil {
		t.Error("expected error for nil entry")
	}
	if err := storage.Save(ctx, &OutboxEntry{}); err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestRedisStorage_MarkAsProcessed(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	entry := &OutboxEntry{ID: "id-1", Topic: "orders", CreatedAt: time.Now()}
	if err := storage.Save(ctx, entry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := storage.MarkAsProcessed(ctx, "id-1"); err != nil {
		t.Fatalf("MarkAsProcessed() error = %v", err)
	}

	fetched, err := storage.FetchUnprocessed(ctx, 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed() error = %v", err)
	}
	if len(fetched) != 0 {
		t.Errorf("expected no unprocessed entries, got %d", len(fetched))
	}

	if err := storage.MarkAsProcessed(ctx, "missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Errorf("expected ErrEntryNotFound, got %v", err)
	}
}

func TestRedisStorage_MarkAsFailed(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	entry := &OutboxEntry{ID: "id-1", Topic: "orders", CreatedAt: time.Now()}
	if err := storage.Save(ctx, entry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := storage.MarkAsFailed(ctx, "id-1", "boom"); err != nil {
		t.Fatalf("MarkAsFailed() error = %v", err)
	}

	fetched, err := storage.FetchUnprocessed(ctx, 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed() error = %v", err)
	}
	if len(fetched) != 0 {
		t.Errorf("failed entry should not be pending, got %d entries", len(fetched))
	}
}

func TestRedisStorage_IncrementRetry(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	entry := &OutboxEntry{ID: "id-1", Topic: "orders", CreatedAt: time.Now()}
	if err := storage.Save(ctx, entry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := storage.IncrementRetry(ctx, "id-1"); err != nil {
		t.Fatalf("IncrementRetry() error = %v", err)
	}
	if err := storage.IncrementRetry(ctx, "id-1"); err != nil {
		t.Fatalf("IncrementRetry() error = %v", err)
	}

	fetched, err := storage.FetchUnprocessed(ctx, 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed() error = %v", err)
	}
	if len(fetched) != 1 || fetched[0].RetryCount != 2 {
		t.Errorf("expected retry count 2, got %+v", fetched)
	}

	if err := storage.IncrementRetry(ctx, "missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Errorf("expected ErrEntryNotFound, got %v", err)
	}
}

func TestRedisStorage_Delete(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	entry := &OutboxEntry{ID: "id-1", Topic: "orders", CreatedAt: time.Now()}
	if err := storage.Save(ctx, entry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := storage.Delete(ctx, "id-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := storage.Delete(ctx, "id-1"); !errors.Is(err, ErrEntryNotFound) {
		t.Errorf("expected ErrEntryNotFound, got %v", err)
	}

	fetched, err := storage.FetchUnprocessed(ctx, 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed() error = %v", err)
	}
	if len(fetched) != 0 {
		t.Errorf("expected no entries after delete, got %d", len(fetched))
	}
}

func TestRedisStorage_DeleteProcessedBefore(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	old := &OutboxEntry{ID: "old", Topic: "orders", CreatedAt: time.Now().Add(-2 * time.Hour)}
	fresh := &OutboxEntry{ID: "fresh", Topic: "orders", CreatedAt: time.Now()}
	if err := storage.SaveBatch(ctx, []*OutboxEntry{old, fresh}); err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}
	if err := storage.MarkAsProcessed(ctx, "old"); err != nil {
		t.Fatalf("MarkAsProcessed() error = %v", err)
	}

	// old 的 processed_at 是现在，用未来时间作为界限应删除它
	if err := storage.DeleteProcessedBefore(ctx, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("DeleteProcessedBefore() error = %v", err)
	}

	if err := storage.MarkAsProcessed(ctx, "old"); !errors.Is(err, ErrEntryNotFound) {
		t.Errorf("expected old entry to be deleted, got %v", err)
	}
	// fresh 未处理，不应被清理
	fetched, err := storage.FetchUnprocessed(ctx, 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed() error = %v", err)
	}
	if len(fetched) != 1 || fetched[0].ID != "fresh" {
		t.Errorf("expected [fresh], got %v", fetched)
	}
}

func TestRedisStorage_Transaction(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	entry := &OutboxEntry{ID: "id-1", Topic: "orders", CreatedAt: time.Now()}
	if err := tx.Save(ctx, entry); err != nil {
		t.Fatalf("tx.Save() error = %v", err)
	}

	// 提交前不可见
	fetched, err := storage.FetchUnprocessed(ctx, 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed() error = %v", err)
	}
	if len(fetched) != 0 {
		t.Errorf("entry should not be visible before commit, got %d", len(fetched))
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("tx.Commit() error = %v", err)
	}

	fetched, err = storage.FetchUnprocessed(ctx, 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed() error = %v", err)
	}
	if len(fetched) != 1 || fetched[0].ID != "id-1" {
		t.Errorf("expected [id-1] after commit, got %v", fetched)
	}

	// 提交后再操作应报错
	if err := tx.Save(ctx, entry); !errors.Is(err, ErrTransactionNotActive) {
		t.Errorf("expected ErrTransactionNotActive, got %v", err)
	}
	if err := tx.Commit(ctx); !errors.Is(err, ErrTransactionNotActive) {
		t.Errorf("expected ErrTransactionNotActive, got %v", err)
	}
}

func TestRedisStorage_Transaction_Rollback(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	if err := tx.Save(ctx, &OutboxEntry{ID: "id-1", Topic: "orders"}); err != nil {
		t.Fatalf("tx.Save() error = %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("tx.Rollback() error = %v", err)
	}

	fetched, err := storage.FetchUnprocessed(ctx, 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed() error = %v", err)
	}
	if len(fetched) != 0 {
		t.Errorf("expected no entries after rollback, got %d", len(fetched))
	}
}

func TestRedisStorage_Transaction_ExecNotSupported(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	if _, err := tx.Exec(ctx, "SELECT 1"); !errors.Is(err, ErrRedisExecNotSupported) {
		t.Errorf("expected ErrRedisExecNotSupported, got %v", err)
	}
}

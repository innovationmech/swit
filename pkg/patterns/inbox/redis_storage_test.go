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
}

func TestRedisStorage_RecordAndCheck(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	entry := &InboxEntry{
		MessageID:   "msg-1",
		HandlerName: "handler-1",
		Topic:       "orders",
		ProcessedAt: time.Now(),
		Result:      []byte("result"),
		Metadata:    map[string]string{"k": "v"},
	}
	if err := storage.RecordMessage(ctx, entry); err != nil {
		t.Fatalf("RecordMessage() error = %v", err)
	}

	processed, got, err := storage.IsProcessed(ctx, "msg-1")
	if err != nil {
		t.Fatalf("IsProcessed() error = %v", err)
	}
	if !processed {
		t.Fatal("expected message to be processed")
	}
	if got.MessageID != "msg-1" || got.Metadata["k"] != "v" || string(got.Result) != "result" {
		t.Errorf("unexpected entry: %+v", got)
	}

	processed, got, err = storage.IsProcessedByHandler(ctx, "msg-1", "handler-1")
	if err != nil {
		t.Fatalf("IsProcessedByHandler() error = %v", err)
	}
	if !processed || got.HandlerName != "handler-1" {
		t.Errorf("unexpected handler result: processed=%v entry=%+v", processed, got)
	}

	// 未处理的消息与处理器
	processed, _, err = storage.IsProcessed(ctx, "missing")
	if err != nil || processed {
		t.Errorf("expected not processed, got processed=%v err=%v", processed, err)
	}
	processed, _, err = storage.IsProcessedByHandler(ctx, "msg-1", "other-handler")
	if err != nil || processed {
		t.Errorf("expected not processed by other handler, got processed=%v err=%v", processed, err)
	}
}

func TestRedisStorage_RecordMessage_Validation(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	if err := storage.RecordMessage(ctx, nil); err == nil {
		t.Error("expected error for nil entry")
	}
	if err := storage.RecordMessage(ctx, &InboxEntry{}); err == nil {
		t.Error("expected error for empty MessageID")
	}
}

func TestRedisStorage_RecordMessage_Idempotent(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	first := &InboxEntry{
		MessageID:   "msg-1",
		HandlerName: "handler-1",
		Result:      []byte("first"),
		ProcessedAt: time.Now(),
	}
	if err := storage.RecordMessage(ctx, first); err != nil {
		t.Fatalf("RecordMessage() error = %v", err)
	}

	// 重复记录不应覆盖已有结果
	second := &InboxEntry{
		MessageID:   "msg-1",
		HandlerName: "handler-1",
		Result:      []byte("second"),
		ProcessedAt: time.Now(),
	}
	if err := storage.RecordMessage(ctx, second); err != nil {
		t.Fatalf("duplicate RecordMessage() error = %v", err)
	}

	_, got, err := storage.IsProcessedByHandler(ctx, "msg-1", "handler-1")
	if err != nil {
		t.Fatalf("IsProcessedByHandler() error = %v", err)
	}
	if string(got.Result) != "first" {
		t.Errorf("expected original result to be preserved, got %s", got.Result)
	}
}

func TestRedisStorage_MultipleHandlers(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	for _, handler := range []string{"handler-1", "handler-2"} {
		entry := &InboxEntry{
			MessageID:   "msg-1",
			HandlerName: handler,
			ProcessedAt: time.Now(),
		}
		if err := storage.RecordMessage(ctx, entry); err != nil {
			t.Fatalf("RecordMessage(%s) error = %v", handler, err)
		}
	}

	for _, handler := range []string{"handler-1", "handler-2"} {
		processed, _, err := storage.IsProcessedByHandler(ctx, "msg-1", handler)
		if err != nil {
			t.Fatalf("IsProcessedByHandler(%s) error = %v", handler, err)
		}
		if !processed {
			t.Errorf("expected msg-1 processed by %s", handler)
		}
	}
}

func TestRedisStorage_DeleteByMessageID(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	for _, handler := range []string{"handler-1", "handler-2"} {
		entry := &InboxEntry{MessageID: "msg-1", HandlerName: handler, ProcessedAt: time.Now()}
		if err := storage.RecordMessage(ctx, entry); err != nil {
			t.Fatalf("RecordMessage() error = %v", err)
		}
	}

	if err := storage.DeleteByMessageID(ctx, "msg-1"); err != nil {
		t.Fatalf("DeleteByMessageID() error = %v", err)
	}

	processed, _, err := storage.IsProcessed(ctx, "msg-1")
	if err != nil || processed {
		t.Errorf("expected msg-1 deleted, got processed=%v err=%v", processed, err)
	}
	for _, handler := range []string{"handler-1", "handler-2"} {
		processed, _, err := storage.IsProcessedByHandler(ctx, "msg-1", handler)
		if err != nil || processed {
			t.Errorf("expected handler entry deleted for %s, got processed=%v err=%v", handler, processed, err)
		}
	}
}

func TestRedisStorage_DeleteBefore(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	old := &InboxEntry{MessageID: "old", HandlerName: "h", ProcessedAt: time.Now().Add(-2 * time.Hour)}
	fresh := &InboxEntry{MessageID: "fresh", HandlerName: "h", ProcessedAt: time.Now()}
	if err := storage.RecordMessage(ctx, old); err != nil {
		t.Fatalf("RecordMessage(old) error = %v", err)
	}
	if err := storage.RecordMessage(ctx, fresh); err != nil {
		t.Fatalf("RecordMessage(fresh) error = %v", err)
	}

	if err := storage.DeleteBefore(ctx, time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("DeleteBefore() error = %v", err)
	}

	processed, _, err := storage.IsProcessed(ctx, "old")
	if err != nil || processed {
		t.Errorf("expected old entry deleted, got processed=%v err=%v", processed, err)
	}
	processed, _, err = storage.IsProcessed(ctx, "fresh")
	if err != nil || !processed {
		t.Errorf("expected fresh entry to remain, got processed=%v err=%v", processed, err)
	}
}

func TestRedisStorage_Transaction(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	txStorage, ok := InboxStorage(storage).(TransactionalStorage)
	if !ok {
		t.Fatal("RedisStorage should implement TransactionalStorage")
	}

	tx, err := txStorage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	entry := &InboxEntry{MessageID: "msg-1", HandlerName: "h", ProcessedAt: time.Now()}
	if err := tx.RecordMessage(ctx, entry); err != nil {
		t.Fatalf("tx.RecordMessage() error = %v", err)
	}

	// 提交前不可见
	processed, _, err := storage.IsProcessed(ctx, "msg-1")
	if err != nil || processed {
		t.Errorf("expected not visible before commit, got processed=%v err=%v", processed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("tx.Commit() error = %v", err)
	}

	processed, _, err = storage.IsProcessed(ctx, "msg-1")
	if err != nil || !processed {
		t.Errorf("expected visible after commit, got processed=%v err=%v", processed, err)
	}

	// 提交后再操作应报错
	if err := tx.RecordMessage(ctx, entry); err == nil {
		t.Error("expected error after commit")
	}
}

func TestRedisStorage_Transaction_Rollback(t *testing.T) {
	storage := newTestRedisStorage(t)
	ctx := context.Background()

	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	entry := &InboxEntry{MessageID: "msg-1", HandlerName: "h", ProcessedAt: time.Now()}
	if err := tx.RecordMessage(ctx, entry); err != nil {
		t.Fatalf("tx.RecordMessage() error = %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("tx.Rollback() error = %v", err)
	}

	processed, _, err := storage.IsProcessed(ctx, "msg-1")
	if err != nil || processed {
		t.Errorf("expected not recorded after rollback, got processed=%v err=%v", processed, err)
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

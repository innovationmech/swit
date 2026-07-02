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

	"github.com/DATA-DOG/go-sqlmock"
)

func newMockPostgresStorage(t *testing.T) (*PostgresStorage, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	storage, err := NewPostgresStorage(db)
	if err != nil {
		t.Fatalf("failed to create postgres storage: %v", err)
	}
	return storage, mock
}

func TestNewPostgresStorage_Validation(t *testing.T) {
	if _, err := NewPostgresStorage(nil); err == nil {
		t.Error("expected error for nil db")
	}

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	if _, err := NewPostgresStorage(db, WithPostgresTableName("bad; DROP")); err == nil {
		t.Error("expected error for invalid table name")
	}
	if _, err := NewPostgresStorage(db, WithPostgresTableName("app.inbox")); err != nil {
		t.Errorf("unexpected error for schema-qualified name: %v", err)
	}
}

func TestPostgresStorage_RecordMessage(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectExec(`INSERT INTO inbox_entries`).
		WithArgs("msg-1", "handler-1", "orders", sqlmock.AnyArg(), []byte("result"), []byte(`{"k":"v"}`)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	entry := &InboxEntry{
		MessageID:   "msg-1",
		HandlerName: "handler-1",
		Topic:       "orders",
		Result:      []byte("result"),
		Metadata:    map[string]string{"k": "v"},
	}
	if err := storage.RecordMessage(context.Background(), entry); err != nil {
		t.Fatalf("RecordMessage() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPostgresStorage_RecordMessage_Validation(t *testing.T) {
	storage, _ := newMockPostgresStorage(t)

	if err := storage.RecordMessage(context.Background(), nil); err == nil {
		t.Error("expected error for nil entry")
	}
	if err := storage.RecordMessage(context.Background(), &InboxEntry{}); err == nil {
		t.Error("expected error for empty MessageID")
	}
}

func TestPostgresStorage_RecordMessage_Idempotent(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	// ON CONFLICT DO NOTHING：冲突时影响 0 行，但不报错
	mock.ExpectExec(`INSERT INTO inbox_entries`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	entry := &InboxEntry{MessageID: "msg-1", ProcessedAt: time.Now()}
	if err := storage.RecordMessage(context.Background(), entry); err != nil {
		t.Fatalf("duplicate RecordMessage() should not error, got %v", err)
	}
}

func TestPostgresStorage_IsProcessed(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	processedAt := time.Now()
	rows := sqlmock.NewRows([]string{"message_id", "handler_name", "topic", "processed_at", "result", "metadata"}).
		AddRow("msg-1", "handler-1", "orders", processedAt, []byte("result"), []byte(`{"k":"v"}`))

	mock.ExpectQuery(`SELECT .+ FROM inbox_entries WHERE message_id = \$1 ORDER BY`).
		WithArgs("msg-1").
		WillReturnRows(rows)

	processed, entry, err := storage.IsProcessed(context.Background(), "msg-1")
	if err != nil {
		t.Fatalf("IsProcessed() error = %v", err)
	}
	if !processed {
		t.Fatal("expected message to be processed")
	}
	if entry.MessageID != "msg-1" || entry.HandlerName != "handler-1" {
		t.Errorf("unexpected entry: %+v", entry)
	}
	if entry.Metadata["k"] != "v" {
		t.Errorf("expected metadata to be unmarshaled, got %v", entry.Metadata)
	}
}

func TestPostgresStorage_IsProcessed_NotFound(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectQuery(`SELECT .+ FROM inbox_entries WHERE message_id = \$1 ORDER BY`).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"message_id", "handler_name", "topic", "processed_at", "result", "metadata"}))

	processed, entry, err := storage.IsProcessed(context.Background(), "missing")
	if err != nil {
		t.Fatalf("IsProcessed() error = %v", err)
	}
	if processed || entry != nil {
		t.Errorf("expected not processed, got processed=%v entry=%+v", processed, entry)
	}
}

func TestPostgresStorage_IsProcessedByHandler(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	rows := sqlmock.NewRows([]string{"message_id", "handler_name", "topic", "processed_at", "result", "metadata"}).
		AddRow("msg-1", "handler-2", "orders", time.Now(), nil, nil)

	mock.ExpectQuery(`SELECT .+ FROM inbox_entries WHERE message_id = \$1 AND handler_name = \$2`).
		WithArgs("msg-1", "handler-2").
		WillReturnRows(rows)

	processed, entry, err := storage.IsProcessedByHandler(context.Background(), "msg-1", "handler-2")
	if err != nil {
		t.Fatalf("IsProcessedByHandler() error = %v", err)
	}
	if !processed || entry.HandlerName != "handler-2" {
		t.Errorf("unexpected result: processed=%v entry=%+v", processed, entry)
	}
}

func TestPostgresStorage_DeleteBefore(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	before := time.Now()
	mock.ExpectExec(`DELETE FROM inbox_entries WHERE processed_at < \$1`).
		WithArgs(before).
		WillReturnResult(sqlmock.NewResult(0, 5))

	if err := storage.DeleteBefore(context.Background(), before); err != nil {
		t.Fatalf("DeleteBefore() error = %v", err)
	}
}

func TestPostgresStorage_DeleteByMessageID(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectExec(`DELETE FROM inbox_entries WHERE message_id = \$1`).
		WithArgs("msg-1").
		WillReturnResult(sqlmock.NewResult(0, 2))

	if err := storage.DeleteByMessageID(context.Background(), "msg-1"); err != nil {
		t.Fatalf("DeleteByMessageID() error = %v", err)
	}
}

func TestPostgresStorage_EnsureSchema(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS inbox_entries`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := storage.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
}

func TestPostgresStorage_Transaction_Commit(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE accounts`).
		WithArgs("acc-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO inbox_entries`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	affected, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance + 1 WHERE id = $1", "acc-1")
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if affected != 1 {
		t.Errorf("expected 1 row affected, got %d", affected)
	}

	entry := &InboxEntry{MessageID: "msg-1", HandlerName: "handler-1", ProcessedAt: time.Now()}
	if err := tx.RecordMessage(ctx, entry); err != nil {
		t.Fatalf("tx.RecordMessage() error = %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("tx.Commit() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPostgresStorage_Transaction_Rollback(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectBegin()
	mock.ExpectRollback()

	ctx := context.Background()
	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("tx.Rollback() error = %v", err)
	}
}

func TestPostgresStorage_Transaction_Reads(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT .+ FROM inbox_entries WHERE message_id = \$1 ORDER BY`).
		WithArgs("msg-1").
		WillReturnRows(sqlmock.NewRows([]string{"message_id", "handler_name", "topic", "processed_at", "result", "metadata"}))
	mock.ExpectQuery(`SELECT .+ FROM inbox_entries WHERE message_id = \$1 AND handler_name = \$2`).
		WithArgs("msg-1", "handler-1").
		WillReturnRows(sqlmock.NewRows([]string{"message_id", "handler_name", "topic", "processed_at", "result", "metadata"}))
	mock.ExpectCommit()

	ctx := context.Background()
	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	processed, _, err := tx.IsProcessed(ctx, "msg-1")
	if err != nil {
		t.Fatalf("tx.IsProcessed() error = %v", err)
	}
	if processed {
		t.Error("expected not processed")
	}

	processed, _, err = tx.IsProcessedByHandler(ctx, "msg-1", "handler-1")
	if err != nil {
		t.Fatalf("tx.IsProcessedByHandler() error = %v", err)
	}
	if processed {
		t.Error("expected not processed by handler")
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("tx.Commit() error = %v", err)
	}
}

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
	tests := []struct {
		name    string
		table   string
		wantErr bool
	}{
		{name: "default table", table: DefaultPostgresTableName, wantErr: false},
		{name: "schema qualified", table: "app.outbox", wantErr: false},
		{name: "sql injection", table: "outbox; DROP TABLE users", wantErr: true},
		{name: "empty", table: "", wantErr: true},
	}

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPostgresStorage(db, WithPostgresTableName(tt.table))
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPostgresStorage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	if _, err := NewPostgresStorage(nil); err == nil {
		t.Error("expected error for nil db")
	}
}

func TestPostgresStorage_Save(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectExec(`INSERT INTO outbox_entries`).
		WithArgs("id-1", "agg-1", "order.created", "orders", []byte(`{"k":"v"}`),
			[]byte(`{"h":"1"}`), sqlmock.AnyArg(), nil, 0, "").
		WillReturnResult(sqlmock.NewResult(0, 1))

	entry := &OutboxEntry{
		ID:          "id-1",
		AggregateID: "agg-1",
		EventType:   "order.created",
		Topic:       "orders",
		Payload:     []byte(`{"k":"v"}`),
		Headers:     map[string]string{"h": "1"},
	}
	if err := storage.Save(context.Background(), entry); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPostgresStorage_Save_Validation(t *testing.T) {
	storage, _ := newMockPostgresStorage(t)

	if err := storage.Save(context.Background(), nil); err == nil {
		t.Error("expected error for nil entry")
	}
	if err := storage.Save(context.Background(), &OutboxEntry{}); err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestPostgresStorage_SaveBatch(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO outbox_entries`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO outbox_entries`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	entries := []*OutboxEntry{
		{ID: "id-1", Topic: "orders", CreatedAt: time.Now()},
		{ID: "id-2", Topic: "orders", CreatedAt: time.Now()},
	}
	if err := storage.SaveBatch(context.Background(), entries); err != nil {
		t.Fatalf("SaveBatch() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPostgresStorage_SaveBatch_Empty(t *testing.T) {
	storage, _ := newMockPostgresStorage(t)
	if err := storage.SaveBatch(context.Background(), nil); err != nil {
		t.Errorf("SaveBatch(nil) error = %v", err)
	}
}

func TestPostgresStorage_FetchUnprocessed(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	createdAt := time.Now().Add(-time.Minute)
	rows := sqlmock.NewRows([]string{
		"id", "aggregate_id", "event_type", "topic", "payload",
		"headers", "created_at", "processed_at", "retry_count", "last_error",
	}).AddRow("id-1", "agg-1", "order.created", "orders", []byte("payload"),
		[]byte(`{"h":"1"}`), createdAt, nil, 2, "")

	mock.ExpectQuery(`SELECT .+ FROM outbox_entries WHERE processed_at IS NULL`).
		WithArgs(10).
		WillReturnRows(rows)

	entries, err := storage.FetchUnprocessed(context.Background(), 10)
	if err != nil {
		t.Fatalf("FetchUnprocessed() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.ID != "id-1" || e.Topic != "orders" || e.RetryCount != 2 {
		t.Errorf("unexpected entry: %+v", e)
	}
	if e.Headers["h"] != "1" {
		t.Errorf("expected headers to be unmarshaled, got %v", e.Headers)
	}
	if e.ProcessedAt != nil {
		t.Errorf("expected nil ProcessedAt, got %v", e.ProcessedAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPostgresStorage_MarkAsProcessed(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectExec(`UPDATE outbox_entries SET processed_at`).
		WithArgs("id-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := storage.MarkAsProcessed(context.Background(), "id-1"); err != nil {
		t.Fatalf("MarkAsProcessed() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPostgresStorage_MarkAsProcessed_NotFound(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectExec(`UPDATE outbox_entries SET processed_at`).
		WithArgs("missing", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := storage.MarkAsProcessed(context.Background(), "missing")
	if !errors.Is(err, ErrEntryNotFound) {
		t.Errorf("expected ErrEntryNotFound, got %v", err)
	}
}

func TestPostgresStorage_MarkAsFailed(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectExec(`UPDATE outbox_entries SET processed_at .+ last_error`).
		WithArgs("id-1", sqlmock.AnyArg(), "boom").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := storage.MarkAsFailed(context.Background(), "id-1", "boom"); err != nil {
		t.Fatalf("MarkAsFailed() error = %v", err)
	}
}

func TestPostgresStorage_IncrementRetry(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectExec(`UPDATE outbox_entries SET retry_count = retry_count \+ 1`).
		WithArgs("id-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := storage.IncrementRetry(context.Background(), "id-1"); err != nil {
		t.Fatalf("IncrementRetry() error = %v", err)
	}
}

func TestPostgresStorage_Delete(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectExec(`DELETE FROM outbox_entries WHERE id`).
		WithArgs("id-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := storage.Delete(context.Background(), "id-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	mock.ExpectExec(`DELETE FROM outbox_entries WHERE id`).
		WithArgs("missing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := storage.Delete(context.Background(), "missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Errorf("expected ErrEntryNotFound, got %v", err)
	}
}

func TestPostgresStorage_DeleteProcessedBefore(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	before := time.Now()
	mock.ExpectExec(`DELETE FROM outbox_entries WHERE processed_at IS NOT NULL`).
		WithArgs(before).
		WillReturnResult(sqlmock.NewResult(0, 3))

	if err := storage.DeleteProcessedBefore(context.Background(), before); err != nil {
		t.Fatalf("DeleteProcessedBefore() error = %v", err)
	}
}

func TestPostgresStorage_EnsureSchema(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS outbox_entries`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := storage.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
}

func TestPostgresStorage_Transaction_Commit(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO orders`).
		WithArgs("order-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO outbox_entries`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	affected, err := tx.Exec(ctx, "INSERT INTO orders (id) VALUES ($1)", "order-1")
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if affected != 1 {
		t.Errorf("expected 1 row affected, got %d", affected)
	}

	entry := &OutboxEntry{ID: "id-1", Topic: "orders", CreatedAt: time.Now()}
	if err := tx.Save(ctx, entry); err != nil {
		t.Fatalf("tx.Save() error = %v", err)
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
	mock.ExpectExec(`INSERT INTO outbox_entries`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()

	ctx := context.Background()
	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	entry := &OutboxEntry{ID: "id-1", Topic: "orders", CreatedAt: time.Now()}
	if err := tx.Save(ctx, entry); err != nil {
		t.Fatalf("tx.Save() error = %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("tx.Rollback() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPostgresStorage_Transaction_SaveBatch(t *testing.T) {
	storage, mock := newMockPostgresStorage(t)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO outbox_entries`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO outbox_entries`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	tx, err := storage.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	entries := []*OutboxEntry{
		{ID: "id-1", Topic: "orders", CreatedAt: time.Now()},
		{ID: "id-2", Topic: "orders", CreatedAt: time.Now()},
	}
	if err := tx.SaveBatch(ctx, entries); err != nil {
		t.Fatalf("tx.SaveBatch() error = %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("tx.Commit() error = %v", err)
	}
}

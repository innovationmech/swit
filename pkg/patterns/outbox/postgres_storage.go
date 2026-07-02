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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// DefaultPostgresTableName 是 outbox 表的默认名称
const DefaultPostgresTableName = "outbox_entries"

// validTableName 校验表名，防止 SQL 注入（表名无法使用占位符）
var validTableName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)

// PostgresStorage 提供基于 PostgreSQL 的 outbox 存储实现
//
// 与业务数据共用同一个 *sql.DB 连接池，从而可以通过 BeginTx 在
// 同一个数据库事务中原子地保存业务数据与 outbox 条目。
//
// 线程安全：底层依赖 database/sql 连接池，可并发使用。
type PostgresStorage struct {
	db    *sql.DB
	table string
}

// PostgresOption 定义 PostgresStorage 的可选配置
type PostgresOption func(*PostgresStorage)

// WithPostgresTableName 自定义 outbox 表名（可包含 schema 前缀，如 "app.outbox"）
func WithPostgresTableName(table string) PostgresOption {
	return func(s *PostgresStorage) {
		s.table = table
	}
}

// NewPostgresStorage 基于已有的数据库连接池创建 PostgreSQL outbox 存储
//
// db 通常是业务服务已经持有的连接池，这样 outbox 条目才能与业务数据
// 落在同一个数据库中，保证事务原子性。
func NewPostgresStorage(db *sql.DB, opts ...PostgresOption) (*PostgresStorage, error) {
	if db == nil {
		return nil, errors.New("db cannot be nil")
	}

	s := &PostgresStorage{
		db:    db,
		table: DefaultPostgresTableName,
	}
	for _, opt := range opts {
		opt(s)
	}

	if !validTableName.MatchString(s.table) {
		return nil, fmt.Errorf("invalid table name: %q", s.table)
	}

	return s, nil
}

// EnsureSchema 创建 outbox 表及索引（如果不存在）
//
// 生产环境建议通过独立的数据库迁移工具管理 schema，此方法主要
// 便于开发与测试环境快速搭建。
func (s *PostgresStorage) EnsureSchema(ctx context.Context) error {
	ddl := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    id           VARCHAR(255) PRIMARY KEY,
    aggregate_id VARCHAR(255) NOT NULL,
    event_type   VARCHAR(255) NOT NULL,
    topic        VARCHAR(255) NOT NULL,
    payload      BYTEA        NOT NULL,
    headers      JSONB,
    created_at   TIMESTAMPTZ  NOT NULL,
    processed_at TIMESTAMPTZ,
    retry_count  INT          NOT NULL DEFAULT 0,
    last_error   TEXT         NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_%s_unprocessed ON %s (created_at) WHERE processed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_%s_processed_at ON %s (processed_at) WHERE processed_at IS NOT NULL;
`, s.table, sanitizeIndexName(s.table), s.table, sanitizeIndexName(s.table), s.table)

	if _, err := s.db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("failed to ensure outbox schema: %w", err)
	}
	return nil
}

// sanitizeIndexName 将可能带 schema 前缀的表名转换为合法的索引名片段
func sanitizeIndexName(table string) string {
	out := make([]byte, 0, len(table))
	for i := 0; i < len(table); i++ {
		c := table[i]
		if c == '.' {
			out = append(out, '_')
		} else {
			out = append(out, c)
		}
	}
	return string(out)
}

// Save 保存一条 outbox 条目（按 ID 幂等，重复保存不会报错）
func (s *PostgresStorage) Save(ctx context.Context, entry *OutboxEntry) error {
	query, args, err := s.insertStatement(entry)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to save outbox entry: %w", err)
	}
	return nil
}

// SaveBatch 批量保存 outbox 条目
func (s *PostgresStorage) SaveBatch(ctx context.Context, entries []*OutboxEntry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin batch transaction: %w", err)
	}
	for _, entry := range entries {
		query, args, err := s.insertStatement(entry)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to save outbox entry %s: %w", entry.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}
	return nil
}

// FetchUnprocessed 按创建时间顺序获取未处理的条目
func (s *PostgresStorage) FetchUnprocessed(ctx context.Context, limit int) ([]*OutboxEntry, error) {
	query := fmt.Sprintf(`SELECT id, aggregate_id, event_type, topic, payload, headers, created_at, processed_at, retry_count, last_error
FROM %s WHERE processed_at IS NULL ORDER BY created_at ASC LIMIT $1`, s.table)

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unprocessed entries: %w", err)
	}
	defer rows.Close()

	var result []*OutboxEntry
	for rows.Next() {
		entry, err := scanOutboxEntry(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate outbox rows: %w", err)
	}
	return result, nil
}

// MarkAsProcessed 标记条目为已处理
func (s *PostgresStorage) MarkAsProcessed(ctx context.Context, id string) error {
	query := fmt.Sprintf(`UPDATE %s SET processed_at = $2 WHERE id = $1`, s.table)
	return s.execExpectingRow(ctx, query, id, time.Now().UTC())
}

// MarkAsFailed 标记条目为失败并记录错误
//
// 与内存实现保持一致：失败的条目也会设置 processed_at，从而不再被
// FetchUnprocessed 返回；错误信息保留在 last_error 中供排查。
func (s *PostgresStorage) MarkAsFailed(ctx context.Context, id string, errMsg string) error {
	query := fmt.Sprintf(`UPDATE %s SET processed_at = $2, last_error = $3 WHERE id = $1`, s.table)
	return s.execExpectingRow(ctx, query, id, time.Now().UTC(), errMsg)
}

// IncrementRetry 增加重试计数
func (s *PostgresStorage) IncrementRetry(ctx context.Context, id string) error {
	query := fmt.Sprintf(`UPDATE %s SET retry_count = retry_count + 1 WHERE id = $1`, s.table)
	return s.execExpectingRow(ctx, query, id)
}

// Delete 删除指定条目
func (s *PostgresStorage) Delete(ctx context.Context, id string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, s.table)
	return s.execExpectingRow(ctx, query, id)
}

// DeleteProcessedBefore 删除指定时间之前已处理的条目
func (s *PostgresStorage) DeleteProcessedBefore(ctx context.Context, before time.Time) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE processed_at IS NOT NULL AND processed_at < $1`, s.table)
	if _, err := s.db.ExecContext(ctx, query, before); err != nil {
		return fmt.Errorf("failed to delete processed entries: %w", err)
	}
	return nil
}

// BeginTx 开始一个数据库事务
//
// 返回的事务可同时用于保存业务数据（Transaction.Exec）与 outbox 条目
// （Transaction.Save），两者在 Commit 时原子提交。
func (s *PostgresStorage) BeginTx(ctx context.Context) (Transaction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &postgresTransaction{tx: tx, storage: s}, nil
}

// execExpectingRow 执行更新并要求至少影响一行，否则返回 ErrEntryNotFound
func (s *PostgresStorage) execExpectingRow(ctx context.Context, query string, args ...interface{}) error {
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute outbox update: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}
	if affected == 0 {
		return ErrEntryNotFound
	}
	return nil
}

// insertStatement 构造插入语句；按 ID 冲突时忽略以保证幂等
func (s *PostgresStorage) insertStatement(entry *OutboxEntry) (string, []interface{}, error) {
	if entry == nil {
		return "", nil, errors.New("entry cannot be nil")
	}
	if entry.ID == "" {
		return "", nil, errors.New("entry ID cannot be empty")
	}

	headers, err := marshalHeaders(entry.Headers)
	if err != nil {
		return "", nil, err
	}

	createdAt := entry.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	query := fmt.Sprintf(`INSERT INTO %s (id, aggregate_id, event_type, topic, payload, headers, created_at, processed_at, retry_count, last_error)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT (id) DO NOTHING`, s.table)

	args := []interface{}{
		entry.ID,
		entry.AggregateID,
		entry.EventType,
		entry.Topic,
		entry.Payload,
		headers,
		createdAt,
		entry.ProcessedAt,
		entry.RetryCount,
		entry.LastError,
	}
	return query, args, nil
}

// marshalHeaders 将 headers 序列化为 JSON；nil 存储为 SQL NULL
func marshalHeaders(headers map[string]string) (interface{}, error) {
	if headers == nil {
		return nil, nil
	}
	data, err := json.Marshal(headers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal headers: %w", err)
	}
	return data, nil
}

// rowScanner 抽象 *sql.Row 和 *sql.Rows 的 Scan 方法
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanOutboxEntry 从查询结果扫描一条 outbox 条目
func scanOutboxEntry(scanner rowScanner) (*OutboxEntry, error) {
	var (
		entry       OutboxEntry
		headers     []byte
		processedAt sql.NullTime
	)
	if err := scanner.Scan(
		&entry.ID,
		&entry.AggregateID,
		&entry.EventType,
		&entry.Topic,
		&entry.Payload,
		&headers,
		&entry.CreatedAt,
		&processedAt,
		&entry.RetryCount,
		&entry.LastError,
	); err != nil {
		return nil, fmt.Errorf("failed to scan outbox entry: %w", err)
	}
	if len(headers) > 0 {
		if err := json.Unmarshal(headers, &entry.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}
	}
	if processedAt.Valid {
		t := processedAt.Time
		entry.ProcessedAt = &t
	}
	return &entry, nil
}

// postgresTransaction 基于 *sql.Tx 的事务实现
type postgresTransaction struct {
	tx      *sql.Tx
	storage *PostgresStorage
}

// Save 在事务中保存 outbox 条目
func (t *postgresTransaction) Save(ctx context.Context, entry *OutboxEntry) error {
	query, args, err := t.storage.insertStatement(entry)
	if err != nil {
		return err
	}
	if _, err := t.tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to save outbox entry in transaction: %w", err)
	}
	return nil
}

// SaveBatch 在事务中批量保存条目
func (t *postgresTransaction) SaveBatch(ctx context.Context, entries []*OutboxEntry) error {
	for _, entry := range entries {
		if err := t.Save(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

// Commit 提交事务
func (t *postgresTransaction) Commit(ctx context.Context) error {
	if err := t.tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Rollback 回滚事务
func (t *postgresTransaction) Rollback(ctx context.Context) error {
	if err := t.tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}

// Exec 在事务中执行自定义 SQL（用于保存业务数据）
func (t *postgresTransaction) Exec(ctx context.Context, query string, args ...interface{}) (int64, error) {
	res, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to execute statement in transaction: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return affected, nil
}

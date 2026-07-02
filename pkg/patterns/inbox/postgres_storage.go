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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// DefaultPostgresTableName 是 inbox 表的默认名称
const DefaultPostgresTableName = "inbox_entries"

// validTableName 校验表名，防止 SQL 注入（表名无法使用占位符）
var validTableName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)

// PostgresStorage 提供基于 PostgreSQL 的 inbox 存储实现
//
// 与业务数据共用同一个 *sql.DB 连接池，从而可以通过 BeginTx 在
// 同一个数据库事务中原子地保存业务数据与幂等性记录。
//
// 幂等性通过 (message_id, handler_name) 复合主键与
// ON CONFLICT DO NOTHING 保证。
type PostgresStorage struct {
	db    *sql.DB
	table string
}

// PostgresOption 定义 PostgresStorage 的可选配置
type PostgresOption func(*PostgresStorage)

// WithPostgresTableName 自定义 inbox 表名（可包含 schema 前缀，如 "app.inbox"）
func WithPostgresTableName(table string) PostgresOption {
	return func(s *PostgresStorage) {
		s.table = table
	}
}

// NewPostgresStorage 基于已有的数据库连接池创建 PostgreSQL inbox 存储
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

// EnsureSchema 创建 inbox 表及索引（如果不存在）
//
// 生产环境建议通过独立的数据库迁移工具管理 schema。
func (s *PostgresStorage) EnsureSchema(ctx context.Context) error {
	ddl := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    message_id   VARCHAR(255) NOT NULL,
    handler_name VARCHAR(255) NOT NULL DEFAULT '',
    topic        VARCHAR(255) NOT NULL DEFAULT '',
    processed_at TIMESTAMPTZ  NOT NULL,
    result       BYTEA,
    metadata     JSONB,
    PRIMARY KEY (message_id, handler_name)
);
CREATE INDEX IF NOT EXISTS idx_%s_processed_at ON %s (processed_at);
`, s.table, sanitizeIndexName(s.table), s.table)

	if _, err := s.db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("failed to ensure inbox schema: %w", err)
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

// RecordMessage 记录一条消息为已处理（按主键幂等，重复记录返回 nil）
func (s *PostgresStorage) RecordMessage(ctx context.Context, entry *InboxEntry) error {
	query, args, err := s.insertStatement(entry)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to record inbox message: %w", err)
	}
	return nil
}

// IsProcessed 检查消息是否已被处理（任意处理器）
func (s *PostgresStorage) IsProcessed(ctx context.Context, messageID string) (bool, *InboxEntry, error) {
	query := fmt.Sprintf(`SELECT message_id, handler_name, topic, processed_at, result, metadata
FROM %s WHERE message_id = $1 ORDER BY processed_at ASC LIMIT 1`, s.table)
	return s.queryEntry(ctx, query, messageID)
}

// IsProcessedByHandler 检查消息是否已被特定处理器处理
func (s *PostgresStorage) IsProcessedByHandler(ctx context.Context, messageID, handlerName string) (bool, *InboxEntry, error) {
	query := fmt.Sprintf(`SELECT message_id, handler_name, topic, processed_at, result, metadata
FROM %s WHERE message_id = $1 AND handler_name = $2 LIMIT 1`, s.table)
	return s.queryEntry(ctx, query, messageID, handlerName)
}

// DeleteBefore 删除指定时间之前的记录
func (s *PostgresStorage) DeleteBefore(ctx context.Context, before time.Time) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE processed_at < $1`, s.table)
	if _, err := s.db.ExecContext(ctx, query, before); err != nil {
		return fmt.Errorf("failed to delete old inbox entries: %w", err)
	}
	return nil
}

// DeleteByMessageID 删除指定消息的所有记录
func (s *PostgresStorage) DeleteByMessageID(ctx context.Context, messageID string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE message_id = $1`, s.table)
	if _, err := s.db.ExecContext(ctx, query, messageID); err != nil {
		return fmt.Errorf("failed to delete inbox entries: %w", err)
	}
	return nil
}

// BeginTx 开始一个数据库事务
//
// 返回的事务可同时用于保存业务数据（Transaction.Exec）与幂等性记录
// （Transaction.RecordMessage），两者在 Commit 时原子提交。
func (s *PostgresStorage) BeginTx(ctx context.Context) (Transaction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &postgresTransaction{tx: tx, storage: s}, nil
}

// queryEntry 查询单条记录；未找到时返回 (false, nil, nil)
func (s *PostgresStorage) queryEntry(ctx context.Context, query string, args ...interface{}) (bool, *InboxEntry, error) {
	return scanInboxEntry(s.db.QueryRowContext(ctx, query, args...))
}

// insertStatement 构造幂等插入语句
func (s *PostgresStorage) insertStatement(entry *InboxEntry) (string, []interface{}, error) {
	if entry == nil {
		return "", nil, errors.New("entry cannot be nil")
	}
	if entry.MessageID == "" {
		return "", nil, errors.New("entry MessageID cannot be empty")
	}

	var metadata interface{}
	if entry.Metadata != nil {
		data, err := json.Marshal(entry.Metadata)
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadata = data
	}

	processedAt := entry.ProcessedAt
	if processedAt.IsZero() {
		processedAt = time.Now().UTC()
	}

	query := fmt.Sprintf(`INSERT INTO %s (message_id, handler_name, topic, processed_at, result, metadata)
VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (message_id, handler_name) DO NOTHING`, s.table)

	args := []interface{}{
		entry.MessageID,
		entry.HandlerName,
		entry.Topic,
		processedAt,
		entry.Result,
		metadata,
	}
	return query, args, nil
}

// rowScanner 抽象 *sql.Row 和 *sql.Rows 的 Scan 方法
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanInboxEntry 从查询结果扫描一条 inbox 记录；无结果时返回 (false, nil, nil)
func scanInboxEntry(scanner rowScanner) (bool, *InboxEntry, error) {
	var (
		entry    InboxEntry
		metadata []byte
	)
	err := scanner.Scan(
		&entry.MessageID,
		&entry.HandlerName,
		&entry.Topic,
		&entry.ProcessedAt,
		&entry.Result,
		&metadata,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to scan inbox entry: %w", err)
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &entry.Metadata); err != nil {
			return false, nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	return true, &entry, nil
}

// postgresTransaction 基于 *sql.Tx 的事务实现
type postgresTransaction struct {
	tx      *sql.Tx
	storage *PostgresStorage
}

// RecordMessage 在事务中记录消息
func (t *postgresTransaction) RecordMessage(ctx context.Context, entry *InboxEntry) error {
	query, args, err := t.storage.insertStatement(entry)
	if err != nil {
		return err
	}
	if _, err := t.tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to record inbox message in transaction: %w", err)
	}
	return nil
}

// IsProcessed 在事务中检查消息是否已处理
func (t *postgresTransaction) IsProcessed(ctx context.Context, messageID string) (bool, *InboxEntry, error) {
	query := fmt.Sprintf(`SELECT message_id, handler_name, topic, processed_at, result, metadata
FROM %s WHERE message_id = $1 ORDER BY processed_at ASC LIMIT 1`, t.storage.table)
	return scanInboxEntry(t.tx.QueryRowContext(ctx, query, messageID))
}

// IsProcessedByHandler 在事务中检查消息是否已被特定处理器处理
func (t *postgresTransaction) IsProcessedByHandler(ctx context.Context, messageID, handlerName string) (bool, *InboxEntry, error) {
	query := fmt.Sprintf(`SELECT message_id, handler_name, topic, processed_at, result, metadata
FROM %s WHERE message_id = $1 AND handler_name = $2 LIMIT 1`, t.storage.table)
	return scanInboxEntry(t.tx.QueryRowContext(ctx, query, messageID, handlerName))
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

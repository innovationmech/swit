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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultRedisKeyPrefix 是 Redis outbox 存储的默认键前缀
const DefaultRedisKeyPrefix = "outbox"

// ErrRedisExecNotSupported Redis 存储不支持 SQL 执行
var ErrRedisExecNotSupported = errors.New("redis outbox storage does not support SQL execution")

// RedisStorage 提供基于 Redis 的 outbox 存储实现
//
// 键布局：
//   - {prefix}:entry:{id}   -> JSON 序列化的 OutboxEntry
//   - {prefix}:pending      -> ZSET，member 为条目 ID，score 为 CreatedAt（纳秒）
//   - {prefix}:processed    -> ZSET，member 为条目 ID，score 为 ProcessedAt（纳秒）
//
// 注意：Redis 不提供跨业务数据的 ACID 事务，因此 Redis 存储适用于
// 业务状态本身就在 Redis 中、或对原子性要求较弱的场景。需要与关系型
// 数据库业务数据保持严格原子性时，请使用 PostgresStorage。
type RedisStorage struct {
	client    redis.UniversalClient
	keyPrefix string
}

// RedisOption 定义 RedisStorage 的可选配置
type RedisOption func(*RedisStorage)

// WithRedisKeyPrefix 自定义键前缀
func WithRedisKeyPrefix(prefix string) RedisOption {
	return func(s *RedisStorage) {
		s.keyPrefix = prefix
	}
}

// NewRedisStorage 基于已有的 Redis 客户端创建 outbox 存储
func NewRedisStorage(client redis.UniversalClient, opts ...RedisOption) (*RedisStorage, error) {
	if client == nil {
		return nil, errors.New("redis client cannot be nil")
	}

	s := &RedisStorage{
		client:    client,
		keyPrefix: DefaultRedisKeyPrefix,
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.keyPrefix == "" {
		return nil, errors.New("key prefix cannot be empty")
	}
	return s, nil
}

func (s *RedisStorage) entryKey(id string) string {
	return s.keyPrefix + ":entry:" + id
}

func (s *RedisStorage) pendingKey() string {
	return s.keyPrefix + ":pending"
}

func (s *RedisStorage) processedKey() string {
	return s.keyPrefix + ":processed"
}

// Save 保存一条 outbox 条目
func (s *RedisStorage) Save(ctx context.Context, entry *OutboxEntry) error {
	return s.saveEntries(ctx, s.client, []*OutboxEntry{entry})
}

// SaveBatch 批量保存 outbox 条目（通过 pipeline 原子写入）
func (s *RedisStorage) SaveBatch(ctx context.Context, entries []*OutboxEntry) error {
	if len(entries) == 0 {
		return nil
	}
	return s.saveEntries(ctx, s.client, entries)
}

// saveEntries 将条目写入 Redis；使用 TxPipeline 保证批量写入的原子性
func (s *RedisStorage) saveEntries(ctx context.Context, client redis.UniversalClient, entries []*OutboxEntry) error {
	pipe := client.TxPipeline()
	for _, entry := range entries {
		if err := s.queueSave(ctx, pipe, entry); err != nil {
			return err
		}
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to save outbox entries to redis: %w", err)
	}
	return nil
}

// queueSave 将单条保存操作加入 pipeline
func (s *RedisStorage) queueSave(ctx context.Context, pipe redis.Pipeliner, entry *OutboxEntry) error {
	if entry == nil {
		return errors.New("entry cannot be nil")
	}
	if entry.ID == "" {
		return errors.New("entry ID cannot be empty")
	}

	toSave := *entry
	if toSave.CreatedAt.IsZero() {
		toSave.CreatedAt = time.Now().UTC()
	}

	data, err := json.Marshal(&toSave)
	if err != nil {
		return fmt.Errorf("failed to marshal outbox entry: %w", err)
	}

	pipe.Set(ctx, s.entryKey(toSave.ID), data, 0)
	if toSave.ProcessedAt == nil {
		pipe.ZAdd(ctx, s.pendingKey(), redis.Z{
			Score:  float64(toSave.CreatedAt.UnixNano()),
			Member: toSave.ID,
		})
	} else {
		pipe.ZRem(ctx, s.pendingKey(), toSave.ID)
		pipe.ZAdd(ctx, s.processedKey(), redis.Z{
			Score:  float64(toSave.ProcessedAt.UnixNano()),
			Member: toSave.ID,
		})
	}
	return nil
}

// FetchUnprocessed 按创建时间顺序获取未处理的条目
func (s *RedisStorage) FetchUnprocessed(ctx context.Context, limit int) ([]*OutboxEntry, error) {
	if limit <= 0 {
		return nil, nil
	}

	ids, err := s.client.ZRange(ctx, s.pendingKey(), 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read pending outbox ids: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = s.entryKey(id)
	}
	values, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read outbox entries: %w", err)
	}

	result := make([]*OutboxEntry, 0, len(values))
	for _, v := range values {
		raw, ok := v.(string)
		if !ok {
			// 条目已被删除但索引尚未清理，跳过
			continue
		}
		var entry OutboxEntry
		if err := json.Unmarshal([]byte(raw), &entry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal outbox entry: %w", err)
		}
		result = append(result, &entry)
	}
	return result, nil
}

// MarkAsProcessed 标记条目为已处理
func (s *RedisStorage) MarkAsProcessed(ctx context.Context, id string) error {
	return s.updateEntry(ctx, id, func(entry *OutboxEntry) {
		now := time.Now().UTC()
		entry.ProcessedAt = &now
	})
}

// MarkAsFailed 标记条目为失败并记录错误
func (s *RedisStorage) MarkAsFailed(ctx context.Context, id string, errMsg string) error {
	return s.updateEntry(ctx, id, func(entry *OutboxEntry) {
		now := time.Now().UTC()
		entry.ProcessedAt = &now
		entry.LastError = errMsg
	})
}

// IncrementRetry 增加重试计数
func (s *RedisStorage) IncrementRetry(ctx context.Context, id string) error {
	return s.updateEntry(ctx, id, func(entry *OutboxEntry) {
		entry.RetryCount++
	})
}

// updateEntry 读取-修改-写回一条条目，并同步维护索引
func (s *RedisStorage) updateEntry(ctx context.Context, id string, mutate func(*OutboxEntry)) error {
	raw, err := s.client.Get(ctx, s.entryKey(id)).Result()
	if errors.Is(err, redis.Nil) {
		return ErrEntryNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to read outbox entry: %w", err)
	}

	var entry OutboxEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return fmt.Errorf("failed to unmarshal outbox entry: %w", err)
	}

	mutate(&entry)

	data, err := json.Marshal(&entry)
	if err != nil {
		return fmt.Errorf("failed to marshal outbox entry: %w", err)
	}

	pipe := s.client.TxPipeline()
	pipe.Set(ctx, s.entryKey(id), data, 0)
	if entry.ProcessedAt != nil {
		pipe.ZRem(ctx, s.pendingKey(), id)
		pipe.ZAdd(ctx, s.processedKey(), redis.Z{
			Score:  float64(entry.ProcessedAt.UnixNano()),
			Member: id,
		})
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to update outbox entry: %w", err)
	}
	return nil
}

// Delete 删除指定条目
func (s *RedisStorage) Delete(ctx context.Context, id string) error {
	deleted, err := s.client.Del(ctx, s.entryKey(id)).Result()
	if err != nil {
		return fmt.Errorf("failed to delete outbox entry: %w", err)
	}
	if deleted == 0 {
		return ErrEntryNotFound
	}

	pipe := s.client.TxPipeline()
	pipe.ZRem(ctx, s.pendingKey(), id)
	pipe.ZRem(ctx, s.processedKey(), id)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to clean outbox indexes: %w", err)
	}
	return nil
}

// DeleteProcessedBefore 删除指定时间之前已处理的条目
func (s *RedisStorage) DeleteProcessedBefore(ctx context.Context, before time.Time) error {
	maxScore := strconv.FormatInt(before.UnixNano(), 10)
	ids, err := s.client.ZRangeByScore(ctx, s.processedKey(), &redis.ZRangeBy{
		Min: "-inf",
		Max: "(" + maxScore,
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to find processed entries: %w", err)
	}
	if len(ids) == 0 {
		return nil
	}

	keys := make([]string, len(ids))
	members := make([]interface{}, len(ids))
	for i, id := range ids {
		keys[i] = s.entryKey(id)
		members[i] = id
	}

	pipe := s.client.TxPipeline()
	pipe.Del(ctx, keys...)
	pipe.ZRem(ctx, s.processedKey(), members...)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete processed entries: %w", err)
	}
	return nil
}

// BeginTx 开始一个事务
//
// Redis 事务通过 TxPipeline 实现：所有 Save 操作在 Commit 时原子写入。
// 注意：无法像关系型数据库那样与任意业务数据共享事务，Exec 不受支持。
func (s *RedisStorage) BeginTx(ctx context.Context) (Transaction, error) {
	return &redisTransaction{
		storage: s,
		active:  true,
	}, nil
}

// redisTransaction 基于 TxPipeline 的事务实现
type redisTransaction struct {
	storage *RedisStorage
	pending []*OutboxEntry
	active  bool
}

// Save 在事务中保存 outbox 条目
func (t *redisTransaction) Save(ctx context.Context, entry *OutboxEntry) error {
	if !t.active {
		return ErrTransactionNotActive
	}
	if entry == nil {
		return errors.New("entry cannot be nil")
	}
	if entry.ID == "" {
		return errors.New("entry ID cannot be empty")
	}
	t.pending = append(t.pending, entry)
	return nil
}

// SaveBatch 在事务中批量保存条目
func (t *redisTransaction) SaveBatch(ctx context.Context, entries []*OutboxEntry) error {
	for _, entry := range entries {
		if err := t.Save(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

// Commit 提交事务：将暂存的条目通过 TxPipeline 原子写入
func (t *redisTransaction) Commit(ctx context.Context) error {
	if !t.active {
		return ErrTransactionNotActive
	}
	t.active = false

	if len(t.pending) == 0 {
		return nil
	}
	entries := t.pending
	t.pending = nil
	return t.storage.saveEntries(ctx, t.storage.client, entries)
}

// Rollback 回滚事务
func (t *redisTransaction) Rollback(ctx context.Context) error {
	if !t.active {
		return ErrTransactionNotActive
	}
	t.active = false
	t.pending = nil
	return nil
}

// Exec Redis 存储不支持 SQL 执行
func (t *redisTransaction) Exec(ctx context.Context, query string, args ...interface{}) (int64, error) {
	if !t.active {
		return 0, ErrTransactionNotActive
	}
	return 0, ErrRedisExecNotSupported
}

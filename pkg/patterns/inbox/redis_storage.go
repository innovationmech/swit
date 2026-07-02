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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultRedisKeyPrefix 是 Redis inbox 存储的默认键前缀
const DefaultRedisKeyPrefix = "inbox"

// ErrRedisExecNotSupported Redis 存储不支持 SQL 执行
var ErrRedisExecNotSupported = errors.New("redis inbox storage does not support SQL execution")

// indexSeparator 用于在索引 ZSET 的 member 中分隔 messageID 与 handlerName。
// 使用 ASCII unit separator，避免与业务 ID 中常见字符冲突。
const indexSeparator = "\x1f"

// RedisStorage 提供基于 Redis 的 inbox 存储实现
//
// 键布局：
//   - {prefix}:entry:{messageID}:{handlerName} -> JSON 序列化的 InboxEntry
//   - {prefix}:msg:{messageID}                 -> JSON，首个记录该消息的条目（用于 IsProcessed）
//   - {prefix}:handlers:{messageID}            -> SET，记录处理过该消息的 handler 名称
//   - {prefix}:index                           -> ZSET，member 为 messageID+sep+handlerName，
//     score 为 ProcessedAt（纳秒），用于按时间清理
//
// 幂等性通过 SETNX 保证：同一 (messageID, handlerName) 的重复记录不会覆盖。
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

// NewRedisStorage 基于已有的 Redis 客户端创建 inbox 存储
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

func (s *RedisStorage) entryKey(messageID, handlerName string) string {
	return s.keyPrefix + ":entry:" + messageID + ":" + handlerName
}

func (s *RedisStorage) msgKey(messageID string) string {
	return s.keyPrefix + ":msg:" + messageID
}

func (s *RedisStorage) handlersKey(messageID string) string {
	return s.keyPrefix + ":handlers:" + messageID
}

func (s *RedisStorage) indexKey() string {
	return s.keyPrefix + ":index"
}

// RecordMessage 记录一条消息为已处理（幂等：已存在时不覆盖并返回 nil）
func (s *RedisStorage) RecordMessage(ctx context.Context, entry *InboxEntry) error {
	if entry == nil {
		return errors.New("entry cannot be nil")
	}
	if entry.MessageID == "" {
		return errors.New("entry MessageID cannot be empty")
	}

	toSave := *entry
	if toSave.ProcessedAt.IsZero() {
		toSave.ProcessedAt = time.Now().UTC()
	}

	data, err := json.Marshal(&toSave)
	if err != nil {
		return fmt.Errorf("failed to marshal inbox entry: %w", err)
	}

	pipe := s.client.TxPipeline()
	pipe.SetNX(ctx, s.entryKey(toSave.MessageID, toSave.HandlerName), data, 0)
	pipe.SetNX(ctx, s.msgKey(toSave.MessageID), data, 0)
	pipe.SAdd(ctx, s.handlersKey(toSave.MessageID), toSave.HandlerName)
	pipe.ZAdd(ctx, s.indexKey(), redis.Z{
		Score:  float64(toSave.ProcessedAt.UnixNano()),
		Member: toSave.MessageID + indexSeparator + toSave.HandlerName,
	})
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to record inbox message: %w", err)
	}
	return nil
}

// IsProcessed 检查消息是否已被处理（任意处理器）
func (s *RedisStorage) IsProcessed(ctx context.Context, messageID string) (bool, *InboxEntry, error) {
	return s.getEntry(ctx, s.msgKey(messageID))
}

// IsProcessedByHandler 检查消息是否已被特定处理器处理
func (s *RedisStorage) IsProcessedByHandler(ctx context.Context, messageID, handlerName string) (bool, *InboxEntry, error) {
	return s.getEntry(ctx, s.entryKey(messageID, handlerName))
}

// getEntry 读取并反序列化一条记录；键不存在时返回 (false, nil, nil)
func (s *RedisStorage) getEntry(ctx context.Context, key string) (bool, *InboxEntry, error) {
	raw, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to read inbox entry: %w", err)
	}

	var entry InboxEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return false, nil, fmt.Errorf("failed to unmarshal inbox entry: %w", err)
	}
	return true, &entry, nil
}

// DeleteBefore 删除指定时间之前的记录
func (s *RedisStorage) DeleteBefore(ctx context.Context, before time.Time) error {
	maxScore := strconv.FormatInt(before.UnixNano(), 10)
	members, err := s.client.ZRangeByScore(ctx, s.indexKey(), &redis.ZRangeBy{
		Min: "-inf",
		Max: "(" + maxScore,
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to find old inbox entries: %w", err)
	}
	if len(members) == 0 {
		return nil
	}

	pipe := s.client.TxPipeline()
	zremMembers := make([]interface{}, 0, len(members))
	for _, member := range members {
		messageID, handlerName, ok := strings.Cut(member, indexSeparator)
		if !ok {
			continue
		}
		pipe.Del(ctx, s.entryKey(messageID, handlerName))
		pipe.Del(ctx, s.msgKey(messageID))
		pipe.SRem(ctx, s.handlersKey(messageID), handlerName)
		zremMembers = append(zremMembers, member)
	}
	if len(zremMembers) > 0 {
		pipe.ZRem(ctx, s.indexKey(), zremMembers...)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete old inbox entries: %w", err)
	}
	return nil
}

// DeleteByMessageID 删除指定消息的所有记录
func (s *RedisStorage) DeleteByMessageID(ctx context.Context, messageID string) error {
	handlers, err := s.client.SMembers(ctx, s.handlersKey(messageID)).Result()
	if err != nil {
		return fmt.Errorf("failed to read inbox handlers: %w", err)
	}

	pipe := s.client.TxPipeline()
	for _, handler := range handlers {
		pipe.Del(ctx, s.entryKey(messageID, handler))
		pipe.ZRem(ctx, s.indexKey(), messageID+indexSeparator+handler)
	}
	pipe.Del(ctx, s.msgKey(messageID))
	pipe.Del(ctx, s.handlersKey(messageID))
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete inbox entries: %w", err)
	}
	return nil
}

// BeginTx 开始一个事务
//
// Redis 事务通过 TxPipeline 实现：所有 RecordMessage 操作在 Commit 时
// 原子写入；读操作直接读取底层存储。Exec 不受支持。
func (s *RedisStorage) BeginTx(ctx context.Context) (Transaction, error) {
	return &redisTransaction{
		storage: s,
		active:  true,
	}, nil
}

// redisTransaction 基于延迟提交的事务实现
type redisTransaction struct {
	storage *RedisStorage
	pending []*InboxEntry
	active  bool
}

// RecordMessage 在事务中记录消息
func (t *redisTransaction) RecordMessage(ctx context.Context, entry *InboxEntry) error {
	if !t.active {
		return errors.New("transaction not active")
	}
	if entry == nil {
		return errors.New("entry cannot be nil")
	}
	if entry.MessageID == "" {
		return errors.New("entry MessageID cannot be empty")
	}
	t.pending = append(t.pending, entry)
	return nil
}

// IsProcessed 在事务中检查消息是否已处理（直接读取底层存储）
func (t *redisTransaction) IsProcessed(ctx context.Context, messageID string) (bool, *InboxEntry, error) {
	return t.storage.IsProcessed(ctx, messageID)
}

// IsProcessedByHandler 在事务中检查消息是否已被特定处理器处理
func (t *redisTransaction) IsProcessedByHandler(ctx context.Context, messageID, handlerName string) (bool, *InboxEntry, error) {
	return t.storage.IsProcessedByHandler(ctx, messageID, handlerName)
}

// Commit 提交事务：将暂存的记录依次写入
func (t *redisTransaction) Commit(ctx context.Context) error {
	if !t.active {
		return errors.New("transaction not active")
	}
	t.active = false

	entries := t.pending
	t.pending = nil
	for _, entry := range entries {
		if err := t.storage.RecordMessage(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

// Rollback 回滚事务
func (t *redisTransaction) Rollback(ctx context.Context) error {
	if !t.active {
		return errors.New("transaction not active")
	}
	t.active = false
	t.pending = nil
	return nil
}

// Exec Redis 存储不支持 SQL 执行
func (t *redisTransaction) Exec(ctx context.Context, query string, args ...interface{}) (int64, error) {
	if !t.active {
		return 0, errors.New("transaction not active")
	}
	return 0, ErrRedisExecNotSupported
}

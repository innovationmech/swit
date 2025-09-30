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
	"sync"
	"time"
)

var (
	// ErrEntryNotFound 条目不存在
	ErrEntryNotFound = errors.New("outbox entry not found")
	// ErrTransactionNotActive 事务未激活
	ErrTransactionNotActive = errors.New("transaction not active")
)

// InMemoryStorage 提供基于内存的 outbox 存储实现
//
// 线程安全：内部使用互斥锁保护状态
// 用途：测试和开发环境，不适合生产环境
type InMemoryStorage struct {
	mu      sync.RWMutex
	entries map[string]*OutboxEntry
}

// NewInMemoryStorage 创建一个内存存储实例
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		entries: make(map[string]*OutboxEntry),
	}
}

// Save 保存一条 outbox 条目
func (s *InMemoryStorage) Save(ctx context.Context, entry *OutboxEntry) error {
	if entry == nil {
		return errors.New("entry cannot be nil")
	}
	if entry.ID == "" {
		return errors.New("entry ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 创建副本以避免外部修改
	copied := s.copyEntry(entry)
	s.entries[entry.ID] = copied

	return nil
}

// SaveBatch 批量保存 outbox 条目
func (s *InMemoryStorage) SaveBatch(ctx context.Context, entries []*OutboxEntry) error {
	if len(entries) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entry := range entries {
		if entry == nil {
			return errors.New("entry cannot be nil")
		}
		if entry.ID == "" {
			return errors.New("entry ID cannot be empty")
		}
		copied := s.copyEntry(entry)
		s.entries[entry.ID] = copied
	}

	return nil
}

// FetchUnprocessed 获取未处理的条目
func (s *InMemoryStorage) FetchUnprocessed(ctx context.Context, limit int) ([]*OutboxEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*OutboxEntry
	for _, entry := range s.entries {
		if entry.ProcessedAt == nil {
			result = append(result, s.copyEntry(entry))
			if len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// MarkAsProcessed 标记条目为已处理
func (s *InMemoryStorage) MarkAsProcessed(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.entries[id]
	if !exists {
		return ErrEntryNotFound
	}

	now := time.Now()
	entry.ProcessedAt = &now
	return nil
}

// MarkAsFailed 标记条目为失败并记录错误
func (s *InMemoryStorage) MarkAsFailed(ctx context.Context, id string, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.entries[id]
	if !exists {
		return ErrEntryNotFound
	}

	entry.LastError = errMsg
	now := time.Now()
	entry.ProcessedAt = &now
	return nil
}

// IncrementRetry 增加重试计数
func (s *InMemoryStorage) IncrementRetry(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.entries[id]
	if !exists {
		return ErrEntryNotFound
	}

	entry.RetryCount++
	return nil
}

// Delete 删除指定条目
func (s *InMemoryStorage) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.entries[id]; !exists {
		return ErrEntryNotFound
	}

	delete(s.entries, id)
	return nil
}

// DeleteProcessedBefore 删除指定时间之前已处理的条目
func (s *InMemoryStorage) DeleteProcessedBefore(ctx context.Context, before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, entry := range s.entries {
		if entry.ProcessedAt != nil && entry.ProcessedAt.Before(before) {
			delete(s.entries, id)
		}
	}

	return nil
}

// BeginTx 开始一个新事务
func (s *InMemoryStorage) BeginTx(ctx context.Context) (Transaction, error) {
	return &inMemoryTransaction{
		storage: s,
		pending: make(map[string]*OutboxEntry),
		active:  true,
	}, nil
}

// copyEntry 创建条目的深拷贝
func (s *InMemoryStorage) copyEntry(entry *OutboxEntry) *OutboxEntry {
	copied := &OutboxEntry{
		ID:          entry.ID,
		AggregateID: entry.AggregateID,
		EventType:   entry.EventType,
		Topic:       entry.Topic,
		Payload:     make([]byte, len(entry.Payload)),
		Headers:     make(map[string]string),
		CreatedAt:   entry.CreatedAt,
		RetryCount:  entry.RetryCount,
		LastError:   entry.LastError,
	}

	copy(copied.Payload, entry.Payload)

	for k, v := range entry.Headers {
		copied.Headers[k] = v
	}

	if entry.ProcessedAt != nil {
		t := *entry.ProcessedAt
		copied.ProcessedAt = &t
	}

	return copied
}

// GetAll 获取所有条目（仅用于测试）
func (s *InMemoryStorage) GetAll() []*OutboxEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*OutboxEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		result = append(result, s.copyEntry(entry))
	}
	return result
}

// inMemoryTransaction 内存事务实现
type inMemoryTransaction struct {
	storage *InMemoryStorage
	pending map[string]*OutboxEntry
	mu      sync.Mutex
	active  bool
}

// Save 在事务中保存 outbox 条目
func (tx *inMemoryTransaction) Save(ctx context.Context, entry *OutboxEntry) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if !tx.active {
		return ErrTransactionNotActive
	}

	if entry == nil {
		return errors.New("entry cannot be nil")
	}
	if entry.ID == "" {
		return errors.New("entry ID cannot be empty")
	}

	// 暂存到待提交列表
	tx.pending[entry.ID] = tx.storage.copyEntry(entry)
	return nil
}

// SaveBatch 在事务中批量保存条目
func (tx *inMemoryTransaction) SaveBatch(ctx context.Context, entries []*OutboxEntry) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if !tx.active {
		return ErrTransactionNotActive
	}

	for _, entry := range entries {
		if entry == nil {
			return errors.New("entry cannot be nil")
		}
		if entry.ID == "" {
			return errors.New("entry ID cannot be empty")
		}
		tx.pending[entry.ID] = tx.storage.copyEntry(entry)
	}

	return nil
}

// Commit 提交事务
func (tx *inMemoryTransaction) Commit(ctx context.Context) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if !tx.active {
		return ErrTransactionNotActive
	}

	// 将待提交的条目写入存储
	tx.storage.mu.Lock()
	defer tx.storage.mu.Unlock()

	for id, entry := range tx.pending {
		tx.storage.entries[id] = entry
	}

	tx.active = false
	tx.pending = nil

	return nil
}

// Rollback 回滚事务
func (tx *inMemoryTransaction) Rollback(ctx context.Context) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if !tx.active {
		return ErrTransactionNotActive
	}

	tx.active = false
	tx.pending = nil

	return nil
}

// Exec 在事务中执行自定义 SQL
// 内存实现中此方法为空操作，仅用于兼容接口
func (tx *inMemoryTransaction) Exec(ctx context.Context, query string, args ...interface{}) (int64, error) {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if !tx.active {
		return 0, ErrTransactionNotActive
	}

	// 内存实现不支持 SQL 执行
	return 0, nil
}

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
	"fmt"
	"sync"
	"time"
)

// inMemoryStorage 内存存储实现，用于测试和开发
type inMemoryStorage struct {
	mu      sync.RWMutex
	entries map[string]*InboxEntry // key: messageID
	// 支持多处理器场景，key: messageID:handlerName
	handlerEntries map[string]*InboxEntry
}

// NewInMemoryStorage 创建一个新的内存存储
func NewInMemoryStorage() InboxStorage {
	return &inMemoryStorage{
		entries:        make(map[string]*InboxEntry),
		handlerEntries: make(map[string]*InboxEntry),
	}
}

// RecordMessage 记录一条消息为已处理
func (s *inMemoryStorage) RecordMessage(ctx context.Context, entry *InboxEntry) error {
	if entry == nil {
		return fmt.Errorf("entry cannot be nil")
	}
	if entry.MessageID == "" {
		return fmt.Errorf("entry MessageID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 复制 entry 以避免外部修改
	entryCopy := &InboxEntry{
		MessageID:   entry.MessageID,
		HandlerName: entry.HandlerName,
		Topic:       entry.Topic,
		ProcessedAt: entry.ProcessedAt,
		Result:      make([]byte, len(entry.Result)),
		Metadata:    make(map[string]string),
	}

	// 复制数据
	if len(entry.Result) > 0 {
		copy(entryCopy.Result, entry.Result)
	}
	for k, v := range entry.Metadata {
		entryCopy.Metadata[k] = v
	}

	// 存储到两个 map 中
	s.entries[entry.MessageID] = entryCopy

	// 如果有处理器名称，也存储到 handlerEntries
	if entry.HandlerName != "" {
		key := fmt.Sprintf("%s:%s", entry.MessageID, entry.HandlerName)
		s.handlerEntries[key] = entryCopy
	}

	return nil
}

// IsProcessed 检查消息是否已被处理
func (s *inMemoryStorage) IsProcessed(ctx context.Context, messageID string) (bool, *InboxEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.entries[messageID]
	if !exists {
		return false, nil, nil
	}

	return true, entry, nil
}

// IsProcessedByHandler 检查消息是否已被特定处理器处理
func (s *inMemoryStorage) IsProcessedByHandler(ctx context.Context, messageID, handlerName string) (bool, *InboxEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", messageID, handlerName)
	entry, exists := s.handlerEntries[key]
	if !exists {
		return false, nil, nil
	}

	return true, entry, nil
}

// DeleteBefore 删除指定时间之前的记录
func (s *inMemoryStorage) DeleteBefore(ctx context.Context, before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 删除 entries 中的旧记录
	for messageID, entry := range s.entries {
		if entry.ProcessedAt.Before(before) {
			delete(s.entries, messageID)
		}
	}

	// 删除 handlerEntries 中的旧记录
	for key, entry := range s.handlerEntries {
		if entry.ProcessedAt.Before(before) {
			delete(s.handlerEntries, key)
		}
	}

	return nil
}

// DeleteByMessageID 删除指定消息的记录
func (s *inMemoryStorage) DeleteByMessageID(ctx context.Context, messageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 从 entries 中删除
	delete(s.entries, messageID)

	// 从 handlerEntries 中删除所有相关记录
	for key := range s.handlerEntries {
		// key 格式为 "messageID:handlerName"
		// 检查是否以 messageID 开头
		if len(key) > len(messageID) && key[:len(messageID)] == messageID && key[len(messageID)] == ':' {
			delete(s.handlerEntries, key)
		}
	}

	return nil
}

// inMemoryTransaction 内存事务实现
type inMemoryTransaction struct {
	storage *inMemoryStorage
	mu      sync.Mutex
	// 事务中的操作
	operations []func() error
	committed  bool
	rolledBack bool
}

// BeginTx 开始一个新事务
func (s *inMemoryStorage) BeginTx(ctx context.Context) (Transaction, error) {
	return &inMemoryTransaction{
		storage:    s,
		operations: make([]func() error, 0),
	}, nil
}

// RecordMessage 在事务中记录消息
func (tx *inMemoryTransaction) RecordMessage(ctx context.Context, entry *InboxEntry) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction already rolled back")
	}

	// 记录操作，在 Commit 时执行
	tx.operations = append(tx.operations, func() error {
		return tx.storage.RecordMessage(ctx, entry)
	})

	return nil
}

// IsProcessed 在事务中检查消息是否已处理
func (tx *inMemoryTransaction) IsProcessed(ctx context.Context, messageID string) (bool, *InboxEntry, error) {
	// 事务中的读操作直接从存储中读取
	return tx.storage.IsProcessed(ctx, messageID)
}

// IsProcessedByHandler 在事务中检查消息是否已被特定处理器处理
func (tx *inMemoryTransaction) IsProcessedByHandler(ctx context.Context, messageID, handlerName string) (bool, *InboxEntry, error) {
	// 事务中的读操作直接从存储中读取
	return tx.storage.IsProcessedByHandler(ctx, messageID, handlerName)
}

// Commit 提交事务
func (tx *inMemoryTransaction) Commit(ctx context.Context) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction already rolled back")
	}

	// 执行所有操作
	for _, op := range tx.operations {
		if err := op(); err != nil {
			return err
		}
	}

	tx.committed = true
	return nil
}

// Rollback 回滚事务
func (tx *inMemoryTransaction) Rollback(ctx context.Context) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction already rolled back")
	}

	// 清空操作列表
	tx.operations = nil
	tx.rolledBack = true
	return nil
}

// Exec 在事务中执行自定义 SQL
// 注意：内存存储不支持 SQL，此方法仅用于满足接口
func (tx *inMemoryTransaction) Exec(ctx context.Context, query string, args ...interface{}) (int64, error) {
	return 0, fmt.Errorf("in-memory storage does not support SQL execution")
}

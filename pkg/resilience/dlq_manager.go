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

package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DeadLetterPolicy 定义死信队列的处理策略。
type DeadLetterPolicy string

const (
	// DeadLetterPolicyDiscard 丢弃消息，不进入死信队列
	DeadLetterPolicyDiscard DeadLetterPolicy = "discard"

	// DeadLetterPolicyStore 存储到死信队列
	DeadLetterPolicyStore DeadLetterPolicy = "store"

	// DeadLetterPolicyRetry 尝试有限次重试后再存储
	DeadLetterPolicyRetry DeadLetterPolicy = "retry"

	// DeadLetterPolicyNotify 存储并发送通知
	DeadLetterPolicyNotify DeadLetterPolicy = "notify"
)

// DeadLetterMessage 表示一个死信消息。
type DeadLetterMessage struct {
	// ID 消息唯一标识
	ID string

	// OriginalTopic 原始主题
	OriginalTopic string

	// Payload 消息负载
	Payload []byte

	// Headers 消息头
	Headers map[string]string

	// FirstFailureTime 首次失败时间
	FirstFailureTime time.Time

	// LastFailureTime 最后失败时间
	LastFailureTime time.Time

	// RetryCount 重试次数
	RetryCount int

	// FailureReason 失败原因
	FailureReason string

	// StackTrace 错误堆栈
	StackTrace string
}

// DLQManagerConfig 配置死信队列管理器。
type DLQManagerConfig struct {
	// Enabled 是否启用 DLQ
	Enabled bool

	// Policy 处理策略
	Policy DeadLetterPolicy

	// MaxRetries 最大重试次数
	MaxRetries int

	// RetryInterval 重试间隔
	RetryInterval time.Duration

	// TTL 消息在 DLQ 中的存活时间
	TTL time.Duration

	// MaxQueueSize DLQ 最大容量（0 表示无限制）
	MaxQueueSize int

	// NotifyCallback 通知回调函数
	NotifyCallback func(msg *DeadLetterMessage)
}

// DefaultDLQManagerConfig 返回默认配置。
func DefaultDLQManagerConfig() DLQManagerConfig {
	return DLQManagerConfig{
		Enabled:        true,
		Policy:         DeadLetterPolicyStore,
		MaxRetries:     3,
		RetryInterval:  5 * time.Second,
		TTL:            7 * 24 * time.Hour, // 7 days
		MaxQueueSize:   10000,
		NotifyCallback: nil,
	}
}

// Validate 验证配置。
func (c *DLQManagerConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}

	if c.RetryInterval < 0 {
		return fmt.Errorf("retry_interval cannot be negative")
	}

	if c.TTL <= 0 {
		return fmt.Errorf("ttl must be positive")
	}

	if c.MaxQueueSize < 0 {
		return fmt.Errorf("max_queue_size cannot be negative")
	}

	return nil
}

// DLQManager 管理死信消息的路由和处理。
type DLQManager struct {
	config  DLQManagerConfig
	metrics *DLQMetrics

	mu       sync.RWMutex
	messages map[string]*DeadLetterMessage // 按消息 ID 索引
	queue    []*DeadLetterMessage          // FIFO 队列

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewDLQManager 创建新的死信队列管理器。
func NewDLQManager(config DLQManagerConfig) (*DLQManager, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	mgr := &DLQManager{
		config:   config,
		metrics:  NewDLQMetrics(),
		messages: make(map[string]*DeadLetterMessage),
		queue:    make([]*DeadLetterMessage, 0),
		ctx:      ctx,
		cancel:   cancel,
	}

	// 启动后台清理任务
	mgr.wg.Add(1)
	go mgr.cleanupExpiredMessages()

	return mgr, nil
}

// Route 将失败的消息路由到死信队列。
func (m *DLQManager) Route(ctx context.Context, msg *DeadLetterMessage) error {
	if !m.config.Enabled {
		return nil
	}

	// 更新度量指标
	m.metrics.RecordMessage()

	switch m.config.Policy {
	case DeadLetterPolicyDiscard:
		m.metrics.RecordDiscard()
		return nil

	case DeadLetterPolicyStore:
		return m.store(msg)

	case DeadLetterPolicyRetry:
		if msg.RetryCount < m.config.MaxRetries {
			m.metrics.RecordRetry()
			// 这里可以接入重试逻辑
			return nil
		}
		return m.store(msg)

	case DeadLetterPolicyNotify:
		if err := m.store(msg); err != nil {
			return err
		}
		if m.config.NotifyCallback != nil {
			m.config.NotifyCallback(msg)
		}
		m.metrics.RecordNotification()
		return nil

	default:
		return fmt.Errorf("unknown dead letter policy: %s", m.config.Policy)
	}
}

// store 存储消息到内部队列。
func (m *DLQManager) store(msg *DeadLetterMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查容量限制
	if m.config.MaxQueueSize > 0 && len(m.queue) >= m.config.MaxQueueSize {
		// 移除最旧的消息
		oldestMsg := m.queue[0]
		delete(m.messages, oldestMsg.ID)
		m.queue = m.queue[1:]
		m.metrics.RecordEviction()
	}

	// 添加或更新消息
	if existing, ok := m.messages[msg.ID]; ok {
		existing.LastFailureTime = time.Now()
		existing.RetryCount = msg.RetryCount
		existing.FailureReason = msg.FailureReason
		existing.StackTrace = msg.StackTrace
	} else {
		msg.LastFailureTime = time.Now()
		if msg.FirstFailureTime.IsZero() {
			msg.FirstFailureTime = msg.LastFailureTime
		}
		m.messages[msg.ID] = msg
		m.queue = append(m.queue, msg)
	}

	m.metrics.RecordStore()
	return nil
}

// GetMessage 根据 ID 获取死信消息。
func (m *DLQManager) GetMessage(id string) (*DeadLetterMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	msg, ok := m.messages[id]
	if !ok {
		return nil, fmt.Errorf("message not found: %s", id)
	}

	return msg, nil
}

// ListMessages 列出所有死信消息。
func (m *DLQManager) ListMessages() []*DeadLetterMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*DeadLetterMessage, len(m.queue))
	copy(result, m.queue)
	return result
}

// RemoveMessage 从队列中移除消息。
func (m *DLQManager) RemoveMessage(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.messages[id]; !ok {
		return fmt.Errorf("message not found: %s", id)
	}

	delete(m.messages, id)

	// 从队列中移除
	for i, msg := range m.queue {
		if msg.ID == id {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			break
		}
	}

	m.metrics.RecordRemoval()
	return nil
}

// Purge 清空所有死信消息。
func (m *DLQManager) Purge() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := len(m.messages)
	m.messages = make(map[string]*DeadLetterMessage)
	m.queue = make([]*DeadLetterMessage, 0)

	for i := 0; i < count; i++ {
		m.metrics.RecordRemoval()
	}

	return nil
}

// GetMetrics 返回当前的度量指标。
func (m *DLQManager) GetMetrics() *DLQMetrics {
	return m.metrics
}

// GetQueueSize 返回当前队列大小。
func (m *DLQManager) GetQueueSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.queue)
}

// cleanupExpiredMessages 后台任务：清理过期消息。
func (m *DLQManager) cleanupExpiredMessages() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.removeExpiredMessages()
		}
	}
}

// removeExpiredMessages 移除过期的消息。
func (m *DLQManager) removeExpiredMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var toRemove []string

	for id, msg := range m.messages {
		if now.Sub(msg.FirstFailureTime) > m.config.TTL {
			toRemove = append(toRemove, id)
		}
	}

	// 移除过期消息
	for _, id := range toRemove {
		delete(m.messages, id)
		m.metrics.RecordExpiration()
	}

	// 重建队列
	if len(toRemove) > 0 {
		newQueue := make([]*DeadLetterMessage, 0, len(m.queue)-len(toRemove))
		for _, msg := range m.queue {
			if _, ok := m.messages[msg.ID]; ok {
				newQueue = append(newQueue, msg)
			}
		}
		m.queue = newQueue
	}
}

// Close 关闭 DLQ 管理器。
func (m *DLQManager) Close() error {
	m.cancel()
	m.wg.Wait()
	return nil
}

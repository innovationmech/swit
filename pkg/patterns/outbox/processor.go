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
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

var (
	// ErrProcessorNotStarted 处理器未启动
	ErrProcessorNotStarted = errors.New("outbox processor not started")
	// ErrProcessorAlreadyStarted 处理器已经启动
	ErrProcessorAlreadyStarted = errors.New("outbox processor already started")
)

// processor 实现 OutboxProcessor 接口
type processor struct {
	storage   OutboxStorage
	publisher messaging.EventPublisher
	config    ProcessorConfig

	mu        sync.RWMutex
	started   bool
	stopChan  chan struct{}
	doneChan  chan struct{}
	cleanupWg sync.WaitGroup
}

// NewProcessor 创建一个新的 outbox 处理器
func NewProcessor(storage OutboxStorage, publisher messaging.EventPublisher, config ProcessorConfig) OutboxProcessor {
	if config.PollInterval == 0 {
		config.PollInterval = 5 * time.Second
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.WorkerCount == 0 {
		config.WorkerCount = 1
	}

	return &processor{
		storage:   storage,
		publisher: publisher,
		config:    config,
		stopChan:  make(chan struct{}),
		doneChan:  make(chan struct{}),
	}
}

// Start 启动 outbox 处理器
func (p *processor) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return ErrProcessorAlreadyStarted
	}
	p.started = true
	p.mu.Unlock()

	// 启动 worker 协程
	p.cleanupWg.Add(1)
	go p.runWorker(ctx)

	// 如果配置了清理间隔，启动清理协程
	if p.config.CleanupInterval > 0 {
		p.cleanupWg.Add(1)
		go p.runCleanup(ctx)
	}

	return nil
}

// Stop 停止 outbox 处理器
func (p *processor) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return ErrProcessorNotStarted
	}
	p.mu.Unlock()

	// 发送停止信号
	close(p.stopChan)

	// 等待所有协程完成
	doneChan := make(chan struct{})
	go func() {
		p.cleanupWg.Wait()
		close(doneChan)
	}()

	// 等待完成或超时
	select {
	case <-doneChan:
		p.mu.Lock()
		p.started = false
		p.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ProcessOnce 手动触发一次处理
func (p *processor) ProcessOnce(ctx context.Context) error {
	return p.processBatch(ctx)
}

// runWorker 运行后台 worker
func (p *processor) runWorker(ctx context.Context) {
	defer p.cleanupWg.Done()

	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.processBatch(ctx); err != nil {
				// 记录错误但继续运行
				// 生产环境中应该使用日志记录
				_ = err
			}
		}
	}
}

// runCleanup 运行清理协程
func (p *processor) runCleanup(ctx context.Context) {
	defer p.cleanupWg.Done()

	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			before := time.Now().Add(-p.config.CleanupAfter)
			if err := p.storage.DeleteProcessedBefore(ctx, before); err != nil {
				// 记录错误但继续运行
				_ = err
			}
		}
	}
}

// processBatch 处理一批未发布的消息
func (p *processor) processBatch(ctx context.Context) error {
	// 获取未处理的条目
	entries, err := p.storage.FetchUnprocessed(ctx, p.config.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to fetch unprocessed entries: %w", err)
	}

	if len(entries) == 0 {
		return nil
	}

	// 处理每个条目
	for _, entry := range entries {
		if err := p.processEntry(ctx, entry); err != nil {
			// 记录错误但继续处理其他条目
			_ = err
		}
	}

	return nil
}

// processEntry 处理单个 outbox 条目
func (p *processor) processEntry(ctx context.Context, entry *OutboxEntry) error {
	// 检查是否超过最大重试次数
	if entry.RetryCount >= p.config.MaxRetries {
		return p.storage.MarkAsFailed(ctx, entry.ID, "max retries exceeded")
	}

	// 构造消息
	msg := &messaging.Message{
		ID:      entry.ID,
		Topic:   entry.Topic,
		Payload: entry.Payload,
		Headers: make(map[string]string),
	}

	// 复制头部信息
	for k, v := range entry.Headers {
		msg.Headers[k] = v
	}

	// 添加额外的元数据
	msg.Headers["aggregate_id"] = entry.AggregateID
	msg.Headers["event_type"] = entry.EventType
	msg.Headers["created_at"] = entry.CreatedAt.Format(time.RFC3339)

	// 发布消息
	if err := p.publisher.Publish(ctx, msg); err != nil {
		// 增加重试计数
		if err := p.storage.IncrementRetry(ctx, entry.ID); err != nil {
			return fmt.Errorf("failed to increment retry count: %w", err)
		}
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// 标记为已处理
	if err := p.storage.MarkAsProcessed(ctx, entry.ID); err != nil {
		return fmt.Errorf("failed to mark as processed: %w", err)
	}

	return nil
}

// publisher 实现 OutboxPublisher 接口
type publisher struct {
	storage OutboxStorage
}

// NewPublisher 创建一个新的 outbox 发布器
func NewPublisher(storage OutboxStorage) OutboxPublisher {
	return &publisher{
		storage: storage,
	}
}

// SaveForPublish 保存消息到 outbox
func (p *publisher) SaveForPublish(ctx context.Context, entry *OutboxEntry) error {
	if entry == nil {
		return errors.New("entry cannot be nil")
	}
	if entry.ID == "" {
		return errors.New("entry ID cannot be empty")
	}
	if entry.Topic == "" {
		return errors.New("entry topic cannot be empty")
	}

	// 设置创建时间
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	return p.storage.Save(ctx, entry)
}

// SaveForPublishBatch 批量保存消息到 outbox
func (p *publisher) SaveForPublishBatch(ctx context.Context, entries []*OutboxEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// 验证所有条目
	for i, entry := range entries {
		if entry == nil {
			return fmt.Errorf("entry at index %d is nil", i)
		}
		if entry.ID == "" {
			return fmt.Errorf("entry at index %d has empty ID", i)
		}
		if entry.Topic == "" {
			return fmt.Errorf("entry at index %d has empty topic", i)
		}
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = time.Now()
		}
	}

	return p.storage.SaveBatch(ctx, entries)
}

// SaveWithTransaction 在指定的事务中保存消息
func (p *publisher) SaveWithTransaction(ctx context.Context, tx Transaction, entry *OutboxEntry) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}
	if entry == nil {
		return errors.New("entry cannot be nil")
	}
	if entry.ID == "" {
		return errors.New("entry ID cannot be empty")
	}
	if entry.Topic == "" {
		return errors.New("entry topic cannot be empty")
	}

	// 设置创建时间
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	return tx.Save(ctx, entry)
}

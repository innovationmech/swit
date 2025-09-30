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
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

var (
	// ErrMessageAlreadyProcessed 消息已经被处理
	ErrMessageAlreadyProcessed = errors.New("message already processed")
	// ErrHandlerNil 处理器为空
	ErrHandlerNil = errors.New("handler cannot be nil")
	// ErrStorageNil 存储为空
	ErrStorageNil = errors.New("storage cannot be nil")
	// ErrMessageNil 消息为空
	ErrMessageNil = errors.New("message cannot be nil")
	// ErrMessageIDEmpty 消息ID为空
	ErrMessageIDEmpty = errors.New("message ID cannot be empty")
)

// processor 实现 InboxProcessor 接口
type processor struct {
	storage InboxStorage
	handler MessageHandler
	config  ProcessorConfig

	mu             sync.RWMutex
	cleanupStarted bool
	stopChan       chan struct{}
	cleanupWg      sync.WaitGroup
}

// NewProcessor 创建一个新的 inbox 处理器
func NewProcessor(storage InboxStorage, handler MessageHandler, config ProcessorConfig) (InboxProcessor, error) {
	if storage == nil {
		return nil, ErrStorageNil
	}
	if handler == nil {
		return nil, ErrHandlerNil
	}

	// 如果没有指定处理器名称，使用 handler 的名称
	if config.HandlerName == "" {
		config.HandlerName = handler.GetName()
	}

	// 设置默认值
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 24 * time.Hour
	}
	if config.CleanupAfter == 0 {
		config.CleanupAfter = 7 * 24 * time.Hour
	}

	p := &processor{
		storage:  storage,
		handler:  handler,
		config:   config,
		stopChan: make(chan struct{}),
	}

	// 如果启用自动清理，启动清理协程
	if config.EnableAutoCleanup {
		p.startCleanup()
	}

	return p, nil
}

// Process 处理消息（幂等）
func (p *processor) Process(ctx context.Context, msg *messaging.Message) error {
	if msg == nil {
		return ErrMessageNil
	}
	if msg.ID == "" {
		return ErrMessageIDEmpty
	}

	// 检查消息是否已处理
	processed, entry, err := p.storage.IsProcessedByHandler(ctx, msg.ID, p.config.HandlerName)
	if err != nil {
		return fmt.Errorf("failed to check if message is processed: %w", err)
	}

	if processed {
		// 消息已处理，直接返回
		return nil
	}

	// 处理消息
	if err := p.handler.Handle(ctx, msg); err != nil {
		return fmt.Errorf("failed to handle message: %w", err)
	}

	// 记录消息为已处理
	inboxEntry := &InboxEntry{
		MessageID:   msg.ID,
		HandlerName: p.config.HandlerName,
		Topic:       msg.Topic,
		ProcessedAt: time.Now(),
		Metadata:    make(map[string]string),
	}

	// 复制消息头部作为元数据
	for k, v := range msg.Headers {
		inboxEntry.Metadata[k] = v
	}

	if err := p.storage.RecordMessage(ctx, inboxEntry); err != nil {
		// 记录失败但消息已处理，记录错误但不返回失败
		// 这样可以避免消息被重复处理
		return fmt.Errorf("message processed but failed to record in inbox: %w", err)
	}

	// 如果之前的 entry 不为 nil，说明有旧记录（不应该发生）
	_ = entry

	return nil
}

// ProcessWithTransaction 在事务中处理消息
func (p *processor) ProcessWithTransaction(ctx context.Context, tx Transaction, msg *messaging.Message) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}
	if msg == nil {
		return ErrMessageNil
	}
	if msg.ID == "" {
		return ErrMessageIDEmpty
	}

	// 在事务中检查消息是否已处理
	processed, entry, err := tx.IsProcessedByHandler(ctx, msg.ID, p.config.HandlerName)
	if err != nil {
		return fmt.Errorf("failed to check if message is processed: %w", err)
	}

	if processed {
		// 消息已处理，直接返回
		return nil
	}

	// 处理消息
	if err := p.handler.Handle(ctx, msg); err != nil {
		return fmt.Errorf("failed to handle message: %w", err)
	}

	// 在事务中记录消息为已处理
	inboxEntry := &InboxEntry{
		MessageID:   msg.ID,
		HandlerName: p.config.HandlerName,
		Topic:       msg.Topic,
		ProcessedAt: time.Now(),
		Metadata:    make(map[string]string),
	}

	// 复制消息头部作为元数据
	for k, v := range msg.Headers {
		inboxEntry.Metadata[k] = v
	}

	if err := tx.RecordMessage(ctx, inboxEntry); err != nil {
		return fmt.Errorf("failed to record message in inbox: %w", err)
	}

	// 如果之前的 entry 不为 nil，说明有旧记录（不应该发生）
	_ = entry

	return nil
}

// ProcessWithResult 处理消息并返回结果
func (p *processor) ProcessWithResult(ctx context.Context, msg *messaging.Message) (interface{}, error) {
	if msg == nil {
		return nil, ErrMessageNil
	}
	if msg.ID == "" {
		return nil, ErrMessageIDEmpty
	}

	// 检查消息是否已处理
	processed, entry, err := p.storage.IsProcessedByHandler(ctx, msg.ID, p.config.HandlerName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if message is processed: %w", err)
	}

	if processed && entry != nil && len(entry.Result) > 0 {
		// 消息已处理，返回之前的结果
		var result interface{}
		if err := json.Unmarshal(entry.Result, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result: %w", err)
		}
		return result, nil
	}

	// 处理消息
	var result interface{}

	// 检查处理器是否支持返回结果
	if handlerWithResult, ok := p.handler.(MessageHandlerWithResult); ok {
		result, err = handlerWithResult.HandleWithResult(ctx, msg)
		if err != nil {
			return nil, fmt.Errorf("failed to handle message: %w", err)
		}
	} else {
		// 处理器不支持返回结果，只调用 Handle
		if err := p.handler.Handle(ctx, msg); err != nil {
			return nil, fmt.Errorf("failed to handle message: %w", err)
		}
	}

	// 记录消息为已处理
	inboxEntry := &InboxEntry{
		MessageID:   msg.ID,
		HandlerName: p.config.HandlerName,
		Topic:       msg.Topic,
		ProcessedAt: time.Now(),
		Metadata:    make(map[string]string),
	}

	// 复制消息头部作为元数据
	for k, v := range msg.Headers {
		inboxEntry.Metadata[k] = v
	}

	// 如果配置了存储结果，序列化并存储
	if p.config.StoreResult && result != nil {
		resultJSON, err := json.Marshal(result)
		if err == nil {
			inboxEntry.Result = resultJSON
		}
	}

	if err := p.storage.RecordMessage(ctx, inboxEntry); err != nil {
		// 记录失败但消息已处理，记录错误但不返回失败
		// 这样可以避免消息被重复处理
		return result, fmt.Errorf("message processed but failed to record in inbox: %w", err)
	}

	return result, nil
}

// startCleanup 启动清理协程
func (p *processor) startCleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cleanupStarted {
		return
	}

	p.cleanupStarted = true
	p.cleanupWg.Add(1)
	go p.runCleanup()
}

// StopCleanup 停止清理协程
func (p *processor) StopCleanup(ctx context.Context) error {
	p.mu.Lock()
	if !p.cleanupStarted {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	// 发送停止信号
	close(p.stopChan)

	// 等待清理协程完成
	doneChan := make(chan struct{})
	go func() {
		p.cleanupWg.Wait()
		close(doneChan)
	}()

	// 等待完成或超时
	select {
	case <-doneChan:
		p.mu.Lock()
		p.cleanupStarted = false
		p.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// runCleanup 运行清理协程
func (p *processor) runCleanup() {
	defer p.cleanupWg.Done()

	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			// 清理过期的记录
			before := time.Now().Add(-p.config.CleanupAfter)
			ctx := context.Background()
			if err := p.storage.DeleteBefore(ctx, before); err != nil {
				// 记录错误但继续运行
				// 生产环境中应该使用日志记录
				_ = err
			}
		}
	}
}

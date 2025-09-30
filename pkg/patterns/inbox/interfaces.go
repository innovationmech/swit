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
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// InboxEntry 表示 inbox 表中的一条记录
//
// 用于记录已处理的消息，防止重复处理
type InboxEntry struct {
	// MessageID 消息唯一标识符（幂等性键）
	MessageID string
	// HandlerName 处理器名称
	HandlerName string
	// Topic 消息来源的主题
	Topic string
	// ProcessedAt 处理时间
	ProcessedAt time.Time
	// Result 处理结果（可选，用于存储处理结果）
	Result []byte
	// Metadata 额外的元数据
	Metadata map[string]string
}

// InboxStorage 定义 inbox 存储的抽象接口
//
// 实现此接口可以支持不同的存储后端（SQL数据库、NoSQL、缓存等）
type InboxStorage interface {
	// RecordMessage 记录一条消息为已处理
	// 如果消息已存在，应该返回 nil（幂等操作）
	RecordMessage(ctx context.Context, entry *InboxEntry) error

	// IsProcessed 检查消息是否已被处理
	// 返回：是否已处理、处理结果、错误
	IsProcessed(ctx context.Context, messageID string) (bool, *InboxEntry, error)

	// IsProcessedByHandler 检查消息是否已被特定处理器处理
	// 用于支持同一消息被多个处理器处理的场景
	IsProcessedByHandler(ctx context.Context, messageID, handlerName string) (bool, *InboxEntry, error)

	// DeleteBefore 删除指定时间之前的记录
	// 用于清理历史数据
	DeleteBefore(ctx context.Context, before time.Time) error

	// DeleteByMessageID 删除指定消息的记录
	DeleteByMessageID(ctx context.Context, messageID string) error
}

// TransactionalStorage 定义支持事务的存储接口
//
// 用于在业务事务中记录消息处理状态，确保业务数据和幂等性记录的原子性
type TransactionalStorage interface {
	InboxStorage

	// BeginTx 开始一个新事务
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction 定义事务接口
type Transaction interface {
	// RecordMessage 在事务中记录消息
	RecordMessage(ctx context.Context, entry *InboxEntry) error

	// IsProcessed 在事务中检查消息是否已处理
	IsProcessed(ctx context.Context, messageID string) (bool, *InboxEntry, error)

	// IsProcessedByHandler 在事务中检查消息是否已被特定处理器处理
	IsProcessedByHandler(ctx context.Context, messageID, handlerName string) (bool, *InboxEntry, error)

	// Commit 提交事务
	Commit(ctx context.Context) error

	// Rollback 回滚事务
	Rollback(ctx context.Context) error

	// Exec 在事务中执行自定义 SQL（用于保存业务数据）
	// 返回值为受影响的行数和错误
	Exec(ctx context.Context, query string, args ...interface{}) (int64, error)
}

// InboxProcessor 定义 inbox 处理器的接口
//
// 负责检查消息是否已处理，如果未处理则调用处理器处理
type InboxProcessor interface {
	// Process 处理消息（幂等）
	// 如果消息已处理，直接返回 nil
	// 如果消息未处理，调用处理器处理并记录
	Process(ctx context.Context, msg *messaging.Message) error

	// ProcessWithTransaction 在事务中处理消息
	// 确保业务逻辑和幂等性记录的原子性
	ProcessWithTransaction(ctx context.Context, tx Transaction, msg *messaging.Message) error

	// ProcessWithResult 处理消息并返回结果
	// 如果消息已处理，返回之前的结果
	ProcessWithResult(ctx context.Context, msg *messaging.Message) (interface{}, error)
}

// MessageHandler 定义消息处理器接口
type MessageHandler interface {
	// Handle 处理消息
	Handle(ctx context.Context, msg *messaging.Message) error

	// GetName 返回处理器名称（用于支持多处理器场景）
	GetName() string
}

// MessageHandlerWithResult 定义带返回值的消息处理器接口
type MessageHandlerWithResult interface {
	MessageHandler

	// HandleWithResult 处理消息并返回结果
	HandleWithResult(ctx context.Context, msg *messaging.Message) (interface{}, error)
}

// ProcessorConfig 定义 inbox 处理器的配置
type ProcessorConfig struct {
	// HandlerName 处理器名称
	// 如果为空，使用 handler.GetName()
	HandlerName string

	// EnableAutoCleanup 是否启用自动清理
	EnableAutoCleanup bool

	// CleanupInterval 清理间隔
	CleanupInterval time.Duration

	// CleanupAfter 清理多久之前的记录
	CleanupAfter time.Duration

	// StoreResult 是否存储处理结果
	StoreResult bool
}

// DefaultProcessorConfig 返回默认的处理器配置
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		EnableAutoCleanup: false,
		CleanupInterval:   24 * time.Hour,
		CleanupAfter:      7 * 24 * time.Hour,
		StoreResult:       false,
	}
}

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
	"time"
)

// OutboxEntry 表示 outbox 表中的一条记录
type OutboxEntry struct {
	// ID 唯一标识符
	ID string
	// AggregateID 聚合根ID（业务实体ID）
	AggregateID string
	// EventType 事件类型
	EventType string
	// Topic 消息发布的目标主题
	Topic string
	// Payload 消息负载（JSON 序列化后的数据）
	Payload []byte
	// Headers 消息头部信息
	Headers map[string]string
	// CreatedAt 创建时间
	CreatedAt time.Time
	// ProcessedAt 处理时间（nil 表示未处理）
	ProcessedAt *time.Time
	// RetryCount 重试次数
	RetryCount int
	// LastError 最后一次错误信息
	LastError string
}

// OutboxStorage 定义 outbox 存储的抽象接口
//
// 实现此接口可以支持不同的存储后端（SQL数据库、NoSQL等）
type OutboxStorage interface {
	// Save 保存一条 outbox 条目
	// 通常在业务事务中调用，以确保原子性
	Save(ctx context.Context, entry *OutboxEntry) error

	// SaveBatch 批量保存 outbox 条目
	SaveBatch(ctx context.Context, entries []*OutboxEntry) error

	// FetchUnprocessed 获取未处理的条目
	// limit 限制返回条目数量
	FetchUnprocessed(ctx context.Context, limit int) ([]*OutboxEntry, error)

	// MarkAsProcessed 标记条目为已处理
	MarkAsProcessed(ctx context.Context, id string) error

	// MarkAsFailed 标记条目为失败并记录错误
	MarkAsFailed(ctx context.Context, id string, errMsg string) error

	// IncrementRetry 增加重试计数
	IncrementRetry(ctx context.Context, id string) error

	// Delete 删除指定条目
	Delete(ctx context.Context, id string) error

	// DeleteProcessedBefore 删除指定时间之前已处理的条目
	// 用于清理历史数据
	DeleteProcessedBefore(ctx context.Context, before time.Time) error
}

// TransactionalStorage 定义支持事务的存储接口
//
// 用于在业务事务中保存 outbox 条目，确保业务数据和事件发布的原子性
type TransactionalStorage interface {
	OutboxStorage

	// BeginTx 开始一个新事务
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction 定义事务接口
type Transaction interface {
	// Save 在事务中保存 outbox 条目
	Save(ctx context.Context, entry *OutboxEntry) error

	// SaveBatch 在事务中批量保存条目
	SaveBatch(ctx context.Context, entries []*OutboxEntry) error

	// Commit 提交事务
	Commit(ctx context.Context) error

	// Rollback 回滚事务
	Rollback(ctx context.Context) error

	// Exec 在事务中执行自定义 SQL（用于保存业务数据）
	// 返回值为受影响的行数和错误
	Exec(ctx context.Context, query string, args ...interface{}) (int64, error)
}

// OutboxProcessor 定义 outbox 处理器的接口
//
// 负责从存储中获取未处理的条目，并发布到消息代理
type OutboxProcessor interface {
	// Start 启动 outbox 处理器
	// 会启动后台 worker 定期轮询并处理未发布的消息
	Start(ctx context.Context) error

	// Stop 停止 outbox 处理器
	// 优雅关闭，等待当前处理完成
	Stop(ctx context.Context) error

	// ProcessOnce 手动触发一次处理
	// 用于测试或按需处理
	ProcessOnce(ctx context.Context) error
}

// OutboxPublisher 定义 outbox 发布器接口
//
// 提供事务性消息发布能力
type OutboxPublisher interface {
	// SaveForPublish 保存消息到 outbox，待后续异步发布
	// 通常在业务事务中调用
	SaveForPublish(ctx context.Context, entry *OutboxEntry) error

	// SaveForPublishBatch 批量保存消息到 outbox
	SaveForPublishBatch(ctx context.Context, entries []*OutboxEntry) error

	// SaveWithTransaction 在指定的事务中保存消息
	// tx 必须是从 TransactionalStorage.BeginTx 获取的事务
	SaveWithTransaction(ctx context.Context, tx Transaction, entry *OutboxEntry) error
}

// ProcessorConfig 定义 outbox 处理器的配置
type ProcessorConfig struct {
	// PollInterval 轮询间隔
	PollInterval time.Duration
	// BatchSize 每次处理的批次大小
	BatchSize int
	// MaxRetries 最大重试次数
	MaxRetries int
	// WorkerCount 并发 worker 数量
	WorkerCount int
	// CleanupInterval 清理已处理消息的间隔（0 表示不清理）
	CleanupInterval time.Duration
	// CleanupAfter 清理多久之前的已处理消息
	CleanupAfter time.Duration
}

// DefaultProcessorConfig 返回默认的处理器配置
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		PollInterval:    5 * time.Second,
		BatchSize:       100,
		MaxRetries:      3,
		WorkerCount:     1,
		CleanupInterval: 24 * time.Hour,
		CleanupAfter:    7 * 24 * time.Hour,
	}
}

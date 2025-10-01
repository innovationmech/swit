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

package transaction

import (
	"context"
	"time"
)

// TransactionState 表示事务的当前状态
type TransactionState int

const (
	// StateInitial 初始状态
	StateInitial TransactionState = iota
	// StatePreparing 准备阶段（2PC Phase 1）
	StatePreparing
	// StatePrepared 准备完成
	StatePrepared
	// StateCommitting 提交阶段（2PC Phase 2）
	StateCommitting
	// StateCommitted 已提交
	StateCommitted
	// StateRollingBack 回滚中
	StateRollingBack
	// StateRolledBack 已回滚
	StateRolledBack
	// StateFailed 失败
	StateFailed
)

// String 返回状态的字符串表示
func (s TransactionState) String() string {
	switch s {
	case StateInitial:
		return "INITIAL"
	case StatePreparing:
		return "PREPARING"
	case StatePrepared:
		return "PREPARED"
	case StateCommitting:
		return "COMMITTING"
	case StateCommitted:
		return "COMMITTED"
	case StateRollingBack:
		return "ROLLING_BACK"
	case StateRolledBack:
		return "ROLLED_BACK"
	case StateFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// TransactionParticipant 定义事务参与者接口
//
// 参与者需要实现三阶段操作：Prepare（准备）、Commit（提交）、Rollback（回滚）
type TransactionParticipant interface {
	// Prepare 准备事务，确保参与者可以提交
	// 返回 nil 表示准备成功，可以提交；返回 error 表示准备失败，需要回滚
	Prepare(ctx context.Context, txID string) error

	// Commit 提交事务，使更改永久生效
	Commit(ctx context.Context, txID string) error

	// Rollback 回滚事务，撤销所有更改
	Rollback(ctx context.Context, txID string) error

	// GetName 返回参与者名称（用于日志和调试）
	GetName() string
}

// TransactionStrategy 定义事务协调策略
type TransactionStrategy interface {
	// Execute 执行事务
	Execute(ctx context.Context, coordinator *TransactionExecution, fn func() error) error

	// GetName 返回策略名称
	GetName() string
}

// TransactionExecution 跟踪事务执行的元数据
type TransactionExecution struct {
	ID           string
	State        TransactionState
	Participants []TransactionParticipant
	Strategy     TransactionStrategy
	StartedAt    time.Time
	UpdatedAt    time.Time
	CompletedAt  *time.Time
	Error        error
	Metadata     map[string]interface{}
}

// TransactionCoordinator 定义分布式事务协调器接口
type TransactionCoordinator interface {
	// Begin 开始一个新事务
	Begin(ctx context.Context, participants []TransactionParticipant) (*TransactionExecution, error)

	// Execute 执行事务（包含业务逻辑）
	Execute(ctx context.Context, txExec *TransactionExecution, fn func() error) error

	// GetTransaction 获取事务状态
	GetTransaction(txID string) (*TransactionExecution, error)

	// SetStrategy 设置协调策略（2PC 或 Compensation）
	SetStrategy(strategy TransactionStrategy)
}

// TransactionConfig 事务配置
type TransactionConfig struct {
	// Timeout 事务超时时间
	Timeout time.Duration

	// MaxRetries 最大重试次数
	MaxRetries int

	// RetryDelay 重试延迟
	RetryDelay time.Duration

	// EnableLogging 是否启用日志
	EnableLogging bool

	// DefaultStrategy 默认协调策略
	DefaultStrategy string // "2pc" or "compensation"
}

// DefaultTransactionConfig 返回默认配置
func DefaultTransactionConfig() *TransactionConfig {
	return &TransactionConfig{
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		RetryDelay:      time.Second,
		EnableLogging:   true,
		DefaultStrategy: "2pc",
	}
}

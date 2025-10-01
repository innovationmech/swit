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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrTransactionNotFound 事务未找到错误
var ErrTransactionNotFound = errors.New("transaction not found")

// ErrInvalidStrategy 无效策略错误
var ErrInvalidStrategy = errors.New("invalid transaction strategy")

// DefaultCoordinator 提供分布式事务协调器的默认实现
//
// 支持可插拔的事务策略：
// - 2PC (Two-Phase Commit)：适用于需要强一致性的场景
// - Compensation：适用于长事务和松耦合系统
//
// 线程安全：使用互斥锁保护内部状态
type DefaultCoordinator struct {
	config   *TransactionConfig
	strategy TransactionStrategy
	store    map[string]*TransactionExecution
	mu       sync.RWMutex
	logger   Logger
}

// NewCoordinator 创建默认协调器实例
func NewCoordinator(config *TransactionConfig) *DefaultCoordinator {
	if config == nil {
		config = DefaultTransactionConfig()
	}

	coordinator := &DefaultCoordinator{
		config: config,
		store:  make(map[string]*TransactionExecution),
		logger: &noopLogger{},
	}

	// 设置默认策略
	if config.DefaultStrategy == "compensation" {
		coordinator.strategy = NewCompensationStrategy(config)
	} else {
		coordinator.strategy = NewTwoPhaseCommitStrategy(config)
	}

	return coordinator
}

// SetLogger 设置日志器
func (c *DefaultCoordinator) SetLogger(logger Logger) {
	c.logger = logger
	if strategy, ok := c.strategy.(*TwoPhaseCommitStrategy); ok {
		strategy.SetLogger(logger)
	}
	if strategy, ok := c.strategy.(*CompensationStrategy); ok {
		strategy.SetLogger(logger)
	}
}

// SetStrategy 设置事务策略
func (c *DefaultCoordinator) SetStrategy(strategy TransactionStrategy) {
	if strategy == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.strategy = strategy
	c.logger.Info("Transaction strategy changed", "strategy", strategy.GetName())
}

// Begin 开始一个新事务
func (c *DefaultCoordinator) Begin(ctx context.Context, participants []TransactionParticipant) (*TransactionExecution, error) {
	if len(participants) == 0 {
		return nil, fmt.Errorf("no participants provided")
	}

	txID := uuid.NewString()
	txExec := &TransactionExecution{
		ID:           txID,
		State:        StateInitial,
		Participants: participants,
		Strategy:     c.strategy,
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	c.mu.Lock()
	c.store[txID] = txExec
	c.mu.Unlock()

	c.logger.Info("Transaction started", "tx_id", txID, "participants", len(participants), "strategy", c.strategy.GetName())

	return txExec, nil
}

// Execute 执行事务
func (c *DefaultCoordinator) Execute(ctx context.Context, txExec *TransactionExecution, fn func() error) error {
	if txExec == nil {
		return fmt.Errorf("transaction execution is nil")
	}

	if txExec.State != StateInitial {
		return fmt.Errorf("transaction %s is not in initial state: %s", txExec.ID, txExec.State)
	}

	// 使用当前策略执行事务
	c.mu.RLock()
	strategy := c.strategy
	c.mu.RUnlock()

	err := strategy.Execute(ctx, txExec, fn)

	// 更新存储中的事务状态
	c.mu.Lock()
	if stored, exists := c.store[txExec.ID]; exists {
		stored.State = txExec.State
		stored.UpdatedAt = txExec.UpdatedAt
		stored.CompletedAt = txExec.CompletedAt
		stored.Error = txExec.Error
	}
	c.mu.Unlock()

	return err
}

// GetTransaction 获取事务状态
func (c *DefaultCoordinator) GetTransaction(txID string) (*TransactionExecution, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	txExec, exists := c.store[txID]
	if !exists {
		return nil, ErrTransactionNotFound
	}

	// 返回副本以避免并发修改
	return c.copyExecution(txExec), nil
}

// copyExecution 创建事务执行的副本
func (c *DefaultCoordinator) copyExecution(txExec *TransactionExecution) *TransactionExecution {
	metadata := make(map[string]interface{})
	for k, v := range txExec.Metadata {
		metadata[k] = v
	}

	return &TransactionExecution{
		ID:           txExec.ID,
		State:        txExec.State,
		Participants: txExec.Participants,
		Strategy:     txExec.Strategy,
		StartedAt:    txExec.StartedAt,
		UpdatedAt:    txExec.UpdatedAt,
		CompletedAt:  txExec.CompletedAt,
		Error:        txExec.Error,
		Metadata:     metadata,
	}
}

// BeginAndExecute 便捷方法：开始并执行事务
func (c *DefaultCoordinator) BeginAndExecute(ctx context.Context, participants []TransactionParticipant, fn func() error) error {
	txExec, err := c.Begin(ctx, participants)
	if err != nil {
		return err
	}

	return c.Execute(ctx, txExec, fn)
}

// ListTransactions 列出所有事务
func (c *DefaultCoordinator) ListTransactions() []*TransactionExecution {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*TransactionExecution, 0, len(c.store))
	for _, txExec := range c.store {
		result = append(result, c.copyExecution(txExec))
	}

	return result
}

// CleanupCompletedTransactions 清理已完成的事务
func (c *DefaultCoordinator) CleanupCompletedTransactions(olderThan time.Duration) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	count := 0

	for txID, txExec := range c.store {
		if txExec.CompletedAt != nil && txExec.CompletedAt.Before(cutoff) {
			delete(c.store, txID)
			count++
		}
	}

	if count > 0 {
		c.logger.Info("Cleaned up completed transactions", "count", count)
	}

	return count
}

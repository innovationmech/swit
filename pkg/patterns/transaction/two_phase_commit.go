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
	"fmt"
	"time"
)

// TwoPhaseCommitStrategy 实现两阶段提交（2PC）协议
//
// 2PC 协议包含两个阶段：
// Phase 1 (Prepare): 协调器询问所有参与者是否可以提交
// Phase 2 (Commit/Rollback): 如果所有参与者都准备好，则提交；否则回滚
type TwoPhaseCommitStrategy struct {
	config *TransactionConfig
	logger Logger
}

// Logger 定义日志接口
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// noopLogger 空日志实现
type noopLogger struct{}

func (n *noopLogger) Debug(msg string, args ...interface{}) {}
func (n *noopLogger) Info(msg string, args ...interface{})  {}
func (n *noopLogger) Warn(msg string, args ...interface{})  {}
func (n *noopLogger) Error(msg string, args ...interface{}) {}

// NewTwoPhaseCommitStrategy 创建 2PC 策略实例
func NewTwoPhaseCommitStrategy(config *TransactionConfig) *TwoPhaseCommitStrategy {
	if config == nil {
		config = DefaultTransactionConfig()
	}

	return &TwoPhaseCommitStrategy{
		config: config,
		logger: &noopLogger{},
	}
}

// SetLogger 设置日志器
func (s *TwoPhaseCommitStrategy) SetLogger(logger Logger) {
	s.logger = logger
}

// GetName 返回策略名称
func (s *TwoPhaseCommitStrategy) GetName() string {
	return "2pc"
}

// Execute 执行 2PC 事务
func (s *TwoPhaseCommitStrategy) Execute(ctx context.Context, txExec *TransactionExecution, fn func() error) error {
	if txExec == nil {
		return fmt.Errorf("transaction execution is nil")
	}

	// 应用超时
	if s.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Phase 1: Prepare
	if err := s.prepare(ctx, txExec); err != nil {
		s.logger.Error("Prepare phase failed", "tx_id", txExec.ID, "error", err)
		s.rollback(ctx, txExec)
		txExec.State = StateFailed
		txExec.Error = err
		return err
	}

	txExec.State = StatePrepared
	txExec.UpdatedAt = time.Now()
	s.logger.Info("All participants prepared successfully", "tx_id", txExec.ID)

	// 执行业务逻辑
	if err := fn(); err != nil {
		s.logger.Error("Business logic failed", "tx_id", txExec.ID, "error", err)
		s.rollback(ctx, txExec)
		txExec.State = StateFailed
		txExec.Error = err
		return err
	}

	// Phase 2: Commit
	if err := s.commit(ctx, txExec); err != nil {
		s.logger.Error("Commit phase failed", "tx_id", txExec.ID, "error", err)
		// 尝试回滚（可能部分成功）
		s.rollback(ctx, txExec)
		txExec.State = StateFailed
		txExec.Error = err
		return err
	}

	txExec.State = StateCommitted
	txExec.UpdatedAt = time.Now()
	now := time.Now()
	txExec.CompletedAt = &now
	s.logger.Info("Transaction committed successfully", "tx_id", txExec.ID)

	return nil
}

// prepare 执行 Prepare 阶段
func (s *TwoPhaseCommitStrategy) prepare(ctx context.Context, txExec *TransactionExecution) error {
	txExec.State = StatePreparing
	txExec.UpdatedAt = time.Now()

	for _, participant := range txExec.Participants {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		s.logger.Debug("Preparing participant", "tx_id", txExec.ID, "participant", participant.GetName())

		if err := s.prepareWithRetry(ctx, participant, txExec.ID); err != nil {
			return fmt.Errorf("participant %s prepare failed: %w", participant.GetName(), err)
		}

		s.logger.Debug("Participant prepared", "tx_id", txExec.ID, "participant", participant.GetName())
	}

	return nil
}

// prepareWithRetry 带重试的 Prepare
func (s *TwoPhaseCommitStrategy) prepareWithRetry(ctx context.Context, participant TransactionParticipant, txID string) error {
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.config.RetryDelay):
			}

			s.logger.Debug("Retrying prepare", "tx_id", txID, "participant", participant.GetName(), "attempt", attempt)
		}

		if err := participant.Prepare(ctx, txID); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("prepare failed after %d retries: %w", s.config.MaxRetries, lastErr)
}

// commit 执行 Commit 阶段
func (s *TwoPhaseCommitStrategy) commit(ctx context.Context, txExec *TransactionExecution) error {
	txExec.State = StateCommitting
	txExec.UpdatedAt = time.Now()

	var firstErr error

	for _, participant := range txExec.Participants {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		s.logger.Debug("Committing participant", "tx_id", txExec.ID, "participant", participant.GetName())

		if err := s.commitWithRetry(ctx, participant, txExec.ID); err != nil {
			s.logger.Error("Participant commit failed", "tx_id", txExec.ID, "participant", participant.GetName(), "error", err)
			if firstErr == nil {
				firstErr = err
			}
			// 继续尝试提交其他参与者
			continue
		}

		s.logger.Debug("Participant committed", "tx_id", txExec.ID, "participant", participant.GetName())
	}

	return firstErr
}

// commitWithRetry 带重试的 Commit
func (s *TwoPhaseCommitStrategy) commitWithRetry(ctx context.Context, participant TransactionParticipant, txID string) error {
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.config.RetryDelay):
			}

			s.logger.Debug("Retrying commit", "tx_id", txID, "participant", participant.GetName(), "attempt", attempt)
		}

		if err := participant.Commit(ctx, txID); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("commit failed after %d retries: %w", s.config.MaxRetries, lastErr)
}

// rollback 回滚所有参与者
func (s *TwoPhaseCommitStrategy) rollback(ctx context.Context, txExec *TransactionExecution) {
	txExec.State = StateRollingBack
	txExec.UpdatedAt = time.Now()

	s.logger.Info("Rolling back transaction", "tx_id", txExec.ID)

	// 反向回滚参与者（先加入的后回滚）
	for i := len(txExec.Participants) - 1; i >= 0; i-- {
		participant := txExec.Participants[i]

		select {
		case <-ctx.Done():
			s.logger.Warn("Context canceled during rollback", "tx_id", txExec.ID)
			// 即使上下文取消，也尽力回滚
		default:
		}

		s.logger.Debug("Rolling back participant", "tx_id", txExec.ID, "participant", participant.GetName())

		if err := s.rollbackWithRetry(ctx, participant, txExec.ID); err != nil {
			s.logger.Error("Participant rollback failed", "tx_id", txExec.ID, "participant", participant.GetName(), "error", err)
			// 继续回滚其他参与者
		} else {
			s.logger.Debug("Participant rolled back", "tx_id", txExec.ID, "participant", participant.GetName())
		}
	}

	txExec.State = StateRolledBack
	txExec.UpdatedAt = time.Now()
	now := time.Now()
	txExec.CompletedAt = &now
}

// rollbackWithRetry 带重试的 Rollback
func (s *TwoPhaseCommitStrategy) rollbackWithRetry(ctx context.Context, participant TransactionParticipant, txID string) error {
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				// 即使上下文取消，也继续尝试
			case <-time.After(s.config.RetryDelay):
			}

			s.logger.Debug("Retrying rollback", "tx_id", txID, "participant", participant.GetName(), "attempt", attempt)
		}

		if err := participant.Rollback(ctx, txID); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("rollback failed after %d retries: %w", s.config.MaxRetries, lastErr)
}

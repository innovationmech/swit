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

// CompensationStrategy 实现基于补偿的事务策略
//
// 补偿策略与 2PC 不同，它不需要 Prepare 阶段：
// - 直接执行操作（Commit）
// - 失败时通过补偿操作（Rollback）撤销已执行的操作
//
// 这种策略更适合长事务和松耦合的分布式系统
type CompensationStrategy struct {
	config *TransactionConfig
	logger Logger
}

// NewCompensationStrategy 创建补偿策略实例
func NewCompensationStrategy(config *TransactionConfig) *CompensationStrategy {
	if config == nil {
		config = DefaultTransactionConfig()
	}

	return &CompensationStrategy{
		config: config,
		logger: &noopLogger{},
	}
}

// SetLogger 设置日志器
func (s *CompensationStrategy) SetLogger(logger Logger) {
	s.logger = logger
}

// GetName 返回策略名称
func (s *CompensationStrategy) GetName() string {
	return "compensation"
}

// Execute 执行补偿型事务
func (s *CompensationStrategy) Execute(ctx context.Context, txExec *TransactionExecution, fn func() error) error {
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

	// 跟踪已执行的参与者（用于补偿）
	executed := make([]TransactionParticipant, 0, len(txExec.Participants))

	// 直接提交每个参与者的操作
	txExec.State = StateCommitting
	txExec.UpdatedAt = time.Now()

	for _, participant := range txExec.Participants {
		select {
		case <-ctx.Done():
			s.logger.Error("Context canceled during execution", "tx_id", txExec.ID)
			s.compensate(ctx, executed, txExec.ID)
			txExec.State = StateFailed
			txExec.Error = ctx.Err()
			return ctx.Err()
		default:
		}

		s.logger.Debug("Executing participant action", "tx_id", txExec.ID, "participant", participant.GetName())

		if err := s.executeWithRetry(ctx, participant, txExec.ID); err != nil {
			s.logger.Error("Participant execution failed", "tx_id", txExec.ID, "participant", participant.GetName(), "error", err)
			// 补偿已执行的操作
			s.compensate(ctx, executed, txExec.ID)
			txExec.State = StateFailed
			txExec.Error = err
			return err
		}

		executed = append(executed, participant)
		s.logger.Debug("Participant executed successfully", "tx_id", txExec.ID, "participant", participant.GetName())
	}

	// 执行业务逻辑
	if err := fn(); err != nil {
		s.logger.Error("Business logic failed", "tx_id", txExec.ID, "error", err)
		// 补偿所有已执行的操作
		s.compensate(ctx, executed, txExec.ID)
		txExec.State = StateFailed
		txExec.Error = err
		return err
	}

	// 成功完成
	txExec.State = StateCommitted
	txExec.UpdatedAt = time.Now()
	now := time.Now()
	txExec.CompletedAt = &now
	s.logger.Info("Transaction completed successfully with compensation strategy", "tx_id", txExec.ID)

	return nil
}

// executeWithRetry 带重试地执行参与者操作
func (s *CompensationStrategy) executeWithRetry(ctx context.Context, participant TransactionParticipant, txID string) error {
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.config.RetryDelay):
			}

			s.logger.Debug("Retrying execution", "tx_id", txID, "participant", participant.GetName(), "attempt", attempt)
		}

		if err := participant.Commit(ctx, txID); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("execution failed after %d retries: %w", s.config.MaxRetries, lastErr)
}

// compensate 补偿已执行的操作
func (s *CompensationStrategy) compensate(ctx context.Context, executed []TransactionParticipant, txID string) {
	s.logger.Info("Compensating transaction", "tx_id", txID, "participants", len(executed))

	// 反向补偿（后执行的先补偿）
	for i := len(executed) - 1; i >= 0; i-- {
		participant := executed[i]

		select {
		case <-ctx.Done():
			s.logger.Warn("Context canceled during compensation", "tx_id", txID)
			// 即使上下文取消，也尽力补偿
		default:
		}

		s.logger.Debug("Compensating participant", "tx_id", txID, "participant", participant.GetName())

		if err := s.compensateWithRetry(ctx, participant, txID); err != nil {
			s.logger.Error("Participant compensation failed", "tx_id", txID, "participant", participant.GetName(), "error", err)
			// 继续补偿其他参与者
		} else {
			s.logger.Debug("Participant compensated", "tx_id", txID, "participant", participant.GetName())
		}
	}
}

// compensateWithRetry 带重试的补偿操作
func (s *CompensationStrategy) compensateWithRetry(ctx context.Context, participant TransactionParticipant, txID string) error {
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				// 即使上下文取消，也继续尝试
			case <-time.After(s.config.RetryDelay):
			}

			s.logger.Debug("Retrying compensation", "tx_id", txID, "participant", participant.GetName(), "attempt", attempt)
		}

		if err := participant.Rollback(ctx, txID); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("compensation failed after %d retries: %w", s.config.MaxRetries, lastErr)
}

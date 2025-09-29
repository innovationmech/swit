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

package saga

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrSagaNotFound 在查询不存在的 Saga ID 时返回
var ErrSagaNotFound = errors.New("saga not found")

// InMemoryCoordinator 提供基于内存的 Saga 协调器实现（无持久化）
//
// 线程安全：内部使用互斥锁保护状态。
// 行为：StartSaga 同步执行步骤；失败时触发反向补偿；
// 状态会记录到内存存储中，便于测试与后续扩展。
type InMemoryCoordinator struct {
	mu    sync.RWMutex
	store map[string]*SagaExecution
}

// NewInMemoryCoordinator 创建一个内存协调器实例
func NewInMemoryCoordinator() *InMemoryCoordinator {
	return &InMemoryCoordinator{
		store: make(map[string]*SagaExecution),
	}
}

// StartSaga 执行 Saga。所有步骤按顺序依次运行；任一步失败则进行反向补偿。
func (c *InMemoryCoordinator) StartSaga(ctx context.Context, def *SagaDefinition, initialData any) (*SagaExecution, error) {
	if def == nil || len(def.Steps) == 0 {
		return nil, fmt.Errorf("invalid saga definition: nil or empty steps")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	sagaID := uuid.NewString()
	exec := &SagaExecution{
		ID:          sagaID,
		Definition:  def,
		Status:      StatusRunning,
		CurrentStep: 0,
		Executed:    make([]string, 0, len(def.Steps)),
		StartedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	c.mu.Lock()
	c.store[sagaID] = exec
	c.mu.Unlock()

	currentData := initialData
	for i, step := range def.Steps {
		if err := ctx.Err(); err != nil {
			return c.failAndCompensate(ctx, exec, i, currentData, err)
		}

		c.updateStatus(exec.ID, func(e *SagaExecution) {
			e.CurrentStep = i
			e.UpdatedAt = time.Now()
		})

		if step.Handler == nil {
			return c.failAndCompensate(ctx, exec, i, currentData, fmt.Errorf("step %q has no handler", step.Name))
		}

		result, err := step.Handler(ctx, currentData)
		if err != nil {
			return c.failAndCompensate(ctx, exec, i, currentData, fmt.Errorf("step %q failed: %w", step.Name, err))
		}

		c.updateStatus(exec.ID, func(e *SagaExecution) {
			e.Executed = append(e.Executed, step.Name)
			e.Result = result
			e.UpdatedAt = time.Now()
		})
		currentData = result
	}

	c.updateStatus(exec.ID, func(e *SagaExecution) {
		e.Status = StatusCompleted
		e.UpdatedAt = time.Now()
	})

	return c.snapshot(exec.ID), nil
}

// CompensateSaga 对已执行的步骤进行反向补偿
func (c *InMemoryCoordinator) CompensateSaga(ctx context.Context, sagaID string) error {
	exec, err := c.get(sagaID)
	if err != nil {
		return err
	}
	return c.compensate(ctx, exec, exec.Result)
}

// GetSagaStatus 返回当前 Saga 执行快照
func (c *InMemoryCoordinator) GetSagaStatus(sagaID string) (*SagaExecution, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	exec, ok := c.store[sagaID]
	if !ok {
		return nil, ErrSagaNotFound
	}
	return c.clone(exec), nil
}

func (c *InMemoryCoordinator) failAndCompensate(ctx context.Context, exec *SagaExecution, failedIndex int, data any, cause error) (*SagaExecution, error) {
	c.updateStatus(exec.ID, func(e *SagaExecution) {
		e.Status = StatusCompensating
		e.Error = cause
		e.UpdatedAt = time.Now()
	})

	_ = c.compensate(ctx, exec, data)

	c.updateStatus(exec.ID, func(e *SagaExecution) {
		e.Status = StatusCompensated
		e.UpdatedAt = time.Now()
	})

	return c.snapshot(exec.ID), fmt.Errorf("saga failed at step index %d: %w", failedIndex, cause)
}

func (c *InMemoryCoordinator) compensate(ctx context.Context, exec *SagaExecution, data any) error {
	def := exec.Definition
	for i := len(exec.Executed) - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		executedStepName := exec.Executed[i]
		var step *SagaStep
		for j := range def.Steps {
			if def.Steps[j].Name == executedStepName {
				step = &def.Steps[j]
				break
			}
		}
		if step == nil || step.Compensation == nil {
			continue
		}
		_ = step.Compensation(ctx, data)
	}
	return nil
}

// snapshot 返回指定 Saga 的只读快照
func (c *InMemoryCoordinator) snapshot(id string) *SagaExecution {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if exec, ok := c.store[id]; ok {
		return c.clone(exec)
	}
	return nil
}

func (c *InMemoryCoordinator) get(id string) (*SagaExecution, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if exec, ok := c.store[id]; ok {
		return exec, nil
	}
	return nil, ErrSagaNotFound
}

func (c *InMemoryCoordinator) updateStatus(id string, mutator func(e *SagaExecution)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.store[id]; ok {
		mutator(e)
	}
}

func (c *InMemoryCoordinator) clone(src *SagaExecution) *SagaExecution {
	copyExec := *src
	copyExec.Executed = append([]string(nil), src.Executed...)
	return &copyExec
}

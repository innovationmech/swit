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
	"time"
)

// SagaStatus 表示 Saga 的当前状态
type SagaStatus string

const (
	// StatusRunning 执行中
	StatusRunning SagaStatus = "RUNNING"
	// StatusCompleted 正常完成
	StatusCompleted SagaStatus = "COMPLETED"
	// StatusFailed 执行失败（未补偿）
	StatusFailed SagaStatus = "FAILED"
	// StatusCompensating 正在进行补偿
	StatusCompensating SagaStatus = "COMPENSATING"
	// StatusCompensated 已完成补偿
	StatusCompensated SagaStatus = "COMPENSATED"
)

// StepHandler 为 Saga 步骤的执行函数
type StepHandler func(ctx context.Context, data any) (any, error)

// CompensationHandler 为 Saga 步骤的补偿函数
type CompensationHandler func(ctx context.Context, data any) error

// SagaStep 描述 Saga 中的单个步骤
type SagaStep struct {
	Name         string
	Handler      StepHandler
	Compensation CompensationHandler
}

// SagaDefinition 描述 Saga 的步骤集合
type SagaDefinition struct {
	Name  string
	Steps []SagaStep
}

// SagaExecution 跟踪 Saga 的执行元数据（内存态）
type SagaExecution struct {
	ID          string
	Definition  *SagaDefinition
	Status      SagaStatus
	CurrentStep int
	Executed    []string
	StartedAt   time.Time
	UpdatedAt   time.Time
	Error       error
	Result      any
}

// SagaCoordinator 定义 Saga 协调器能力
//
// 约定：StartSaga 同步按顺序执行所有步骤；失败时按相反顺序进行补偿
// 并更新状态。后续可扩展为异步及持久化实现。
type SagaCoordinator interface {
	StartSaga(ctx context.Context, def *SagaDefinition, initialData any) (*SagaExecution, error)
	CompensateSaga(ctx context.Context, sagaID string) error
	GetSagaStatus(sagaID string) (*SagaExecution, error)
}

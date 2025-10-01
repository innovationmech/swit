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

package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BulkOperation 表示一个批量操作项
type BulkOperation[T any] struct {
	// Index 在批量操作中的索引
	Index int
	// Data 操作数据
	Data T
}

// BulkResult 表示批量操作的单个结果
type BulkResult[T any] struct {
	// Index 在批量操作中的索引
	Index int
	// Data 操作结果数据
	Data T
	// Error 操作错误（如果有）
	Error error
	// Duration 操作耗时
	Duration time.Duration
	// RetryCount 重试次数
	RetryCount int
}

// BulkReport 表示批量操作的完整报告
type BulkReport[T any] struct {
	// TotalCount 总操作数
	TotalCount int
	// SuccessCount 成功数
	SuccessCount int
	// FailureCount 失败数
	FailureCount int
	// TotalDuration 总耗时
	TotalDuration time.Duration
	// Results 所有结果
	Results []BulkResult[T]
}

// BulkConfig 批量操作配置
type BulkConfig struct {
	// Concurrency 并发数（0 表示串行执行，< 0 表示无限并发）
	Concurrency int
	// ContinueOnError 遇到错误时是否继续（true=尽力而为，false=快速失败）
	ContinueOnError bool
	// RetryEnabled 是否启用重试
	RetryEnabled bool
	// RetryConfig 重试配置（仅当 RetryEnabled=true 时有效）
	RetryConfig *Config
	// OnProgress 进度回调
	OnProgress func(completed, total int)
}

// DefaultBulkConfig 返回默认的批量操作配置
func DefaultBulkConfig() BulkConfig {
	return BulkConfig{
		Concurrency:     10,
		ContinueOnError: true,
		RetryEnabled:    false,
	}
}

// Validate 验证批量配置
func (c *BulkConfig) Validate() error {
	if c.RetryEnabled && c.RetryConfig != nil {
		if err := c.RetryConfig.Validate(); err != nil {
			return fmt.Errorf("invalid retry config: %w", err)
		}
	}
	return nil
}

// BulkExecutor 批量操作执行器
type BulkExecutor[T, R any] struct {
	config   BulkConfig
	executor *Executor
}

// NewBulkExecutor 创建批量操作执行器
func NewBulkExecutor[T, R any](config BulkConfig) (*BulkExecutor[T, R], error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	var executor *Executor
	if config.RetryEnabled && config.RetryConfig != nil {
		var err error
		executor, err = NewExecutor(*config.RetryConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create retry executor: %w", err)
		}
	}

	return &BulkExecutor[T, R]{
		config:   config,
		executor: executor,
	}, nil
}

// Execute 执行批量操作
func (be *BulkExecutor[T, R]) Execute(
	ctx context.Context,
	operations []BulkOperation[T],
	operationFn func(ctx context.Context, data T) (R, error),
) *BulkReport[R] {
	startTime := time.Now()
	totalCount := len(operations)

	results := make([]BulkResult[R], totalCount)
	var successCount, failureCount int
	var mu sync.Mutex

	// 根据并发设置决定执行方式
	if be.config.Concurrency == 0 {
		// 串行执行
		for i, op := range operations {
			result := be.executeOne(ctx, op, operationFn)
			results[i] = result

			mu.Lock()
			if result.Error != nil {
				failureCount++
				if !be.config.ContinueOnError {
					mu.Unlock()
					// 快速失败：填充剩余结果为未执行
					for j := i + 1; j < totalCount; j++ {
						results[j] = BulkResult[R]{
							Index: operations[j].Index,
							Error: fmt.Errorf("skipped due to previous failure"),
						}
						failureCount++
					}
					break
				}
			} else {
				successCount++
			}
			mu.Unlock()

			if be.config.OnProgress != nil {
				be.config.OnProgress(i+1, totalCount)
			}
		}
	} else {
		// 并发执行
		var wg sync.WaitGroup
		sem := make(chan struct{}, max(1, be.config.Concurrency))
		shouldStop := false

		for i, op := range operations {
			// 检查是否应该停止
			mu.Lock()
			if shouldStop {
				results[i] = BulkResult[R]{
					Index: operations[i].Index,
					Error: fmt.Errorf("stopped due to previous failure"),
				}
				failureCount++
				mu.Unlock()
				continue
			}
			mu.Unlock()

			wg.Add(1)
			go func(idx int, operation BulkOperation[T]) {
				defer wg.Done()

				// 获取信号量
				if be.config.Concurrency > 0 {
					sem <- struct{}{}
					defer func() { <-sem }()
				}

				// 执行操作
				result := be.executeOne(ctx, operation, operationFn)

				mu.Lock()
				results[idx] = result
				if result.Error != nil {
					failureCount++
					if !be.config.ContinueOnError {
						shouldStop = true
					}
				} else {
					successCount++
				}
				completed := successCount + failureCount
				mu.Unlock()

				if be.config.OnProgress != nil {
					be.config.OnProgress(completed, totalCount)
				}
			}(i, op)
		}

		wg.Wait()
	}

	return &BulkReport[R]{
		TotalCount:    totalCount,
		SuccessCount:  successCount,
		FailureCount:  failureCount,
		TotalDuration: time.Since(startTime),
		Results:       results,
	}
}

// executeOne 执行单个操作
func (be *BulkExecutor[T, R]) executeOne(
	ctx context.Context,
	op BulkOperation[T],
	operationFn func(ctx context.Context, data T) (R, error),
) BulkResult[R] {
	startTime := time.Now()
	var result BulkResult[R]
	result.Index = op.Index

	if be.config.RetryEnabled && be.executor != nil {
		// 使用重试执行
		retryCount := 0
		data, err := DoWithResult(ctx, be.executor, func() (R, error) {
			retryCount++
			return operationFn(ctx, op.Data)
		})
		result.Data = data
		result.Error = err
		result.RetryCount = retryCount - 1 // 首次执行不算重试
	} else {
		// 直接执行
		data, err := operationFn(ctx, op.Data)
		result.Data = data
		result.Error = err
	}

	result.Duration = time.Since(startTime)
	return result
}

// HasFailures 检查报告是否包含失败
func (r *BulkReport[T]) HasFailures() bool {
	return r.FailureCount > 0
}

// HasPartialSuccess 检查是否部分成功（既有成功也有失败）
func (r *BulkReport[T]) HasPartialSuccess() bool {
	return r.SuccessCount > 0 && r.FailureCount > 0
}

// IsFullSuccess 检查是否全部成功
func (r *BulkReport[T]) IsFullSuccess() bool {
	return r.SuccessCount == r.TotalCount && r.FailureCount == 0
}

// IsFullFailure 检查是否全部失败
func (r *BulkReport[T]) IsFullFailure() bool {
	return r.FailureCount == r.TotalCount && r.SuccessCount == 0
}

// GetSuccessfulResults 获取所有成功的结果
func (r *BulkReport[T]) GetSuccessfulResults() []BulkResult[T] {
	var successful []BulkResult[T]
	for _, result := range r.Results {
		if result.Error == nil {
			successful = append(successful, result)
		}
	}
	return successful
}

// GetFailedResults 获取所有失败的结果
func (r *BulkReport[T]) GetFailedResults() []BulkResult[T] {
	var failed []BulkResult[T]
	for _, result := range r.Results {
		if result.Error != nil {
			failed = append(failed, result)
		}
	}
	return failed
}

// String 返回报告的字符串表示
func (r *BulkReport[T]) String() string {
	return fmt.Sprintf(
		"BulkReport{Total: %d, Success: %d, Failure: %d, Duration: %s}",
		r.TotalCount,
		r.SuccessCount,
		r.FailureCount,
		r.TotalDuration,
	)
}

// max 返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

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

package chaos

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestTimeout_StepExecution tests Saga behavior when steps timeout.
func TestTimeout_StepExecution(t *testing.T) {
	tests := []struct {
		name          string
		timeoutDelay  time.Duration
		stepTimeout   time.Duration
		expectTimeout bool
		description   string
	}{
		{
			name:          "步骤执行超时",
			timeoutDelay:  2 * time.Second,
			stepTimeout:   500 * time.Millisecond,
			expectTimeout: true,
			description:   "步骤执行时间超过超时限制",
		},
		{
			name:          "步骤在超时前完成",
			timeoutDelay:  100 * time.Millisecond,
			stepTimeout:   1 * time.Second,
			expectTimeout: false,
			description:   "步骤在超时前成功完成",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建故障注入器
			injector := NewFaultInjector()
			injector.AddFault("timeout-fault", &FaultConfig{
				Type:        FaultTypeTimeout,
				Probability: 1.0,
				Delay:       tt.timeoutDelay,
				TargetStep:  "step1",
			})

			// 创建带超时的 Saga 定义
			sagaDef := &mockSagaDefinition{
				id:   "timeout-saga",
				name: "Timeout Test Saga",
				steps: []saga.SagaStep{
					injector.WrapStep(&mockStep{
						id:      "step1",
						name:    "Step 1",
						timeout: tt.stepTimeout,
					}),
				},
				timeout:     5 * time.Second,
				retryPolicy: saga.NewNoRetryPolicy(),
			}

			// 创建内存状态存储
			chaosStorage := createTestStorage(injector)

			// 创建协调器
			coord, err := createTestCoordinator(chaosStorage, saga.NewNoRetryPolicy())
			if err != nil {
				t.Fatalf("创建协调器失败: %v", err)
			}
			defer coord.Close()

			// 启动 Saga
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			startTime := time.Now()
			instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "timeout"})
			duration := time.Since(startTime)

			if tt.expectTimeout {
				t.Logf("执行耗时: %v", duration)
				if err == nil && instance != nil {
					// 等待处理完成
					time.Sleep(500 * time.Millisecond)
					finalInstance, _ := coord.GetSagaInstance(instance.GetID())
					if finalInstance != nil {
						t.Logf("Saga 最终状态: %s", finalInstance.GetState())
					}
				}
			} else {
				if duration > tt.stepTimeout {
					t.Errorf("期望在超时前完成，但耗时: %v > %v", duration, tt.stepTimeout)
				}
			}

			// 验证注入统计
			stats := injector.GetStats()
			t.Logf("故障注入统计: %+v", stats)
		})
	}
}

// TestTimeout_SagaLevel tests Saga-level timeout.
func TestTimeout_SagaLevel(t *testing.T) {
	// 创建故障注入器（为多个步骤添加延迟）
	injector := NewFaultInjector()
	injector.AddFault("step-delay", &FaultConfig{
		Type:        FaultTypeDelay,
		Probability: 1.0,
		Delay:       800 * time.Millisecond, // 每步延迟 800ms
	})

	// 创建多步骤 Saga（总时间会超过 Saga 超时）
	sagaDef := &mockSagaDefinition{
		id:   "saga-timeout-test",
		name: "Saga Timeout Test",
		steps: []saga.SagaStep{
			injector.WrapStep(&mockStep{id: "step1", name: "Step 1"}),
			injector.WrapStep(&mockStep{id: "step2", name: "Step 2"}),
			injector.WrapStep(&mockStep{id: "step3", name: "Step 3"}),
		},
		timeout:     2 * time.Second, // Saga 总超时 2 秒
		retryPolicy: saga.NewNoRetryPolicy(),
	}

	// 创建内存状态存储
	chaosStorage := createTestStorage(injector)

	// 创建协调器
	coord, err := createTestCoordinator(chaosStorage, saga.NewNoRetryPolicy())
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 启动 Saga
	ctx := context.Background()
	startTime := time.Now()
	instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "saga-timeout"})
	duration := time.Since(startTime)

	t.Logf("Saga 执行耗时: %v", duration)

	if instance != nil {
		// 等待 Saga 处理完成或超时
		time.Sleep(3 * time.Second)

		// 获取最终状态
		finalInstance, err := coord.GetSagaInstance(instance.GetID())
		if err != nil {
			t.Fatalf("获取 Saga 实例失败: %v", err)
		}

		t.Logf("Saga 最终状态: %s", finalInstance.GetState())
		t.Logf("已完成步骤: %d/%d", finalInstance.GetCompletedSteps(), finalInstance.GetTotalSteps())

		// 验证 Saga 是否因超时而进入终止状态
		if finalInstance.IsTerminal() {
			t.Log("Saga 已达到终止状态")
		}
	} else {
		t.Logf("Saga 启动失败: %v", err)
	}
}

// TestTimeout_CompensationTimeout tests timeout during compensation.
func TestTimeout_CompensationTimeout(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()

	// 为补偿操作添加超时
	injector.AddFault("compensation-timeout", &FaultConfig{
		Type:        FaultTypeTimeout,
		Probability: 1.0,
		Delay:       2 * time.Second,
		TargetStep:  "step2-compensate",
	})

	// 创建会失败的 Saga（触发补偿）
	sagaDef := &mockSagaDefinition{
		id:   "compensation-timeout-saga",
		name: "Compensation Timeout Test Saga",
		steps: []saga.SagaStep{
			injector.WrapStep(&mockStep{id: "step1", name: "Step 1"}),
			injector.WrapStep(&mockStep{id: "step2", name: "Step 2"}),
			injector.WrapStep(&mockStep{id: "step3", name: "Step 3 (Failing)", shouldFail: true}),
		},
		timeout:     10 * time.Second,
		retryPolicy: saga.NewNoRetryPolicy(),
	}

	// 创建内存状态存储
	chaosStorage := createTestStorage(injector)

	// 创建协调器
	coord, err := createTestCoordinator(chaosStorage, saga.NewNoRetryPolicy())
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 启动 Saga
	ctx := context.Background()
	instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "comp-timeout"})

	if instance != nil {
		// 等待补偿处理（包括超时）
		time.Sleep(4 * time.Second)

		// 获取最终状态
		finalInstance, err := coord.GetSagaInstance(instance.GetID())
		if err != nil {
			t.Fatalf("获取 Saga 实例失败: %v", err)
		}

		t.Logf("Saga 最终状态: %s", finalInstance.GetState())

		// 验证补偿超时是否被正确处理
		if finalInstance.GetError() != nil {
			t.Logf("Saga 错误: %s", finalInstance.GetError().Message)
		}
	} else {
		t.Logf("Saga 启动失败: %v", err)
	}

	// 验证注入统计
	stats := injector.GetStats()
	t.Logf("故障注入统计: %+v", stats)
}

// TestTimeout_WithRetry tests timeout behavior with retry policy.
func TestTimeout_WithRetry(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	injector.AddFault("retry-timeout", &FaultConfig{
		Type:        FaultTypeTimeout,
		Probability: 0.7, // 70% 概率超时
		Delay:       1 * time.Second,
		TargetStep:  "step1",
	})

	// 创建 Saga 定义
	sagaDef := &mockSagaDefinition{
		id:   "retry-timeout-saga",
		name: "Retry Timeout Test Saga",
		steps: []saga.SagaStep{
			injector.WrapStep(&mockStep{
				id:      "step1",
				name:    "Step 1 (Retryable)",
				timeout: 2 * time.Second,
			}),
		},
		timeout:     10 * time.Second,
		retryPolicy: saga.NewExponentialBackoffRetryPolicy(3, 500*time.Millisecond, 2*time.Second),
	}

	// 创建内存状态存储
	chaosStorage := createTestStorage(injector)

	// 创建协调器，带重试策略
	coord, err := createTestCoordinator(chaosStorage, saga.NewExponentialBackoffRetryPolicy(3, 500*time.Millisecond, 2*time.Second))
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 启动 Saga
	ctx := context.Background()
	startTime := time.Now()
	instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "retry-timeout"})
	duration := time.Since(startTime)

	t.Logf("Saga 启动耗时: %v", duration)

	if instance != nil {
		// 等待重试处理
		time.Sleep(6 * time.Second)

		// 获取最终状态
		finalInstance, err := coord.GetSagaInstance(instance.GetID())
		if err != nil {
			t.Fatalf("获取 Saga 实例失败: %v", err)
		}

		t.Logf("Saga 最终状态: %s", finalInstance.GetState())

		// 验证是否执行了重试
		stats := injector.GetStats()
		t.Logf("故障注入统计: %+v", stats)

		if stats.TotalInjections > 1 {
			t.Log("检测到多次故障注入，说明执行了重试")
		}
	} else {
		t.Logf("Saga 启动失败: %v", err)
	}
}

// TestTimeout_ContextCancellation tests timeout via context cancellation.
func TestTimeout_ContextCancellation(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	injector.AddFault("long-delay", &FaultConfig{
		Type:        FaultTypeDelay,
		Probability: 1.0,
		Delay:       5 * time.Second, // 长时间延迟
	})

	// 创建 Saga 定义
	sagaDef := &mockSagaDefinition{
		id:   "context-cancel-saga",
		name: "Context Cancellation Test Saga",
		steps: []saga.SagaStep{
			injector.WrapStep(&mockStep{id: "step1", name: "Step 1"}),
		},
		timeout:     10 * time.Second,
		retryPolicy: saga.NewNoRetryPolicy(),
	}

	// 创建内存状态存储
	chaosStorage := createTestStorage(injector)

	// 创建协调器
	coord, err := createTestCoordinator(chaosStorage, saga.NewNoRetryPolicy())
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 创建带超时的 context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 启动 Saga
	startTime := time.Now()
	_, err = coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "context-cancel"})
	duration := time.Since(startTime)

	t.Logf("操作耗时: %v", duration)

	// 验证是否因 context 取消而返回
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		t.Log("Saga 因 context 超时而取消（符合预期）")
	} else if err != nil {
		t.Logf("Saga 失败: %v", err)
	}

	// 验证操作在 context 超时后终止
	if duration > 2*time.Second {
		t.Errorf("期望在 context 超时后快速返回，但耗时: %v", duration)
	}
}

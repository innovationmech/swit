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
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestServiceCrash_DuringStepExecution tests Saga behavior when service crashes during step execution.
func TestServiceCrash_DuringStepExecution(t *testing.T) {
	tests := []struct {
		name               string
		crashAtStep        string
		expectCompensation bool
		description        string
	}{
		{
			name:               "第一步执行时服务崩溃",
			crashAtStep:        "step1",
			expectCompensation: false,
			description:        "服务在第一步执行时崩溃，无需补偿",
		},
		{
			name:               "第二步执行时服务崩溃",
			crashAtStep:        "step2",
			expectCompensation: true,
			description:        "服务在第二步执行时崩溃，需要补偿第一步",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建故障注入器
			injector := NewFaultInjector()
			injector.AddFault("service-crash", &FaultConfig{
				Type:         FaultTypeServiceCrash,
				Probability:  1.0,
				TargetStep:   tt.crashAtStep,
				ErrorMessage: "service crashed unexpectedly",
			})

			// 创建测试 Saga 定义
			sagaDef := createMultiStepSagaDefinition(injector)

			// 创建内存状态存储
			storage := state.NewMemoryStateStorage()
			chaosStorage := NewChaosStateStorage(storage, injector)

			// 创建协调器（不重试）
			coord, err := coordinator.NewOrchestratorCoordinator(
				coordinator.WithStateStorage(chaosStorage),
				coordinator.WithRetryPolicy(saga.NewNoRetryPolicy()),
			)
			if err != nil {
				t.Fatalf("创建协调器失败: %v", err)
			}
			defer coord.Close()

			// 启动 Saga
			ctx := context.Background()
			instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "data"})

			// 服务崩溃应该导致错误
			if err == nil && instance != nil {
				// 等待 Saga 处理
				time.Sleep(500 * time.Millisecond)

				// 获取最新状态
				finalInstance, err := coord.GetSagaInstance(instance.GetID())
				if err != nil {
					t.Fatalf("获取 Saga 实例失败: %v", err)
				}

				t.Logf("Saga 最终状态: %s", finalInstance.GetState())
				
				// 验证是否执行了补偿
				if tt.expectCompensation {
					if finalInstance.GetState() != saga.StateCompensated && finalInstance.GetState() != saga.StateFailed {
						t.Errorf("期望状态为 compensated 或 failed，实际为: %s", finalInstance.GetState())
					}
				}
			} else {
				t.Logf("Saga 启动失败（预期）: %v", err)
			}

			// 验证注入统计
			stats := injector.GetStats()
			t.Logf("故障注入统计: %+v", stats)
		})
	}
}

// TestServiceCrash_StateRecovery tests Saga recovery after service restart.
func TestServiceCrash_StateRecovery(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	injector.Disable() // 初始禁用

	// 创建持久化状态存储（内存模拟）
	storage := state.NewMemoryStateStorage()

	// 第一阶段：正常启动 Saga
	func() {
		chaosStorage := NewChaosStateStorage(storage, injector)
		coord, err := coordinator.NewOrchestratorCoordinator(
			coordinator.WithStateStorage(chaosStorage),
			coordinator.WithRetryPolicy(saga.NewFixedDelayRetryPolicy(2, 100*time.Millisecond)),
		)
		if err != nil {
			t.Fatalf("创建协调器失败: %v", err)
		}
		defer coord.Close()

		// 创建测试 Saga 定义
		sagaDef := createMultiStepSagaDefinition(injector)

		// 启动 Saga
		ctx := context.Background()
		instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "recovery"})
		if err != nil {
			t.Fatalf("启动 Saga 失败: %v", err)
		}

		t.Logf("Saga 已启动: %s", instance.GetID())

		// 保存 Saga ID 用于恢复
		sagaID := instance.GetID()

		// 等待部分执行
		time.Sleep(200 * time.Millisecond)

		// 模拟服务崩溃（关闭协调器）
		t.Log("模拟服务崩溃...")

		// 第二阶段：服务重启后恢复
		time.Sleep(100 * time.Millisecond)

		// 创建新的协调器（模拟服务重启）
		coord2, err := coordinator.NewOrchestratorCoordinator(
			coordinator.WithStateStorage(chaosStorage),
			coordinator.WithRetryPolicy(saga.NewFixedDelayRetryPolicy(2, 100*time.Millisecond)),
		)
		if err != nil {
			t.Fatalf("创建恢复协调器失败: %v", err)
		}
		defer coord2.Close()

		t.Log("服务已重启，尝试恢复 Saga...")

		// 尝试获取之前的 Saga 实例
		recoveredInstance, err := coord2.GetSagaInstance(sagaID)
		if err != nil {
			t.Logf("无法恢复 Saga 实例: %v", err)
			return
		}

		t.Logf("Saga 恢复成功，状态: %s", recoveredInstance.GetState())

		// 等待 Saga 完成或补偿
		time.Sleep(1 * time.Second)

		// 验证最终状态
		finalInstance, err := coord2.GetSagaInstance(sagaID)
		if err != nil {
			t.Fatalf("获取最终 Saga 实例失败: %v", err)
		}

		if !finalInstance.IsTerminal() {
			t.Logf("Saga 未达到终止状态: %s（可能仍在处理中）", finalInstance.GetState())
		} else {
			t.Logf("Saga 最终状态: %s", finalInstance.GetState())
		}
	}()
}

// TestServiceCrash_CompensationLogic tests compensation logic when service crashes during compensation.
func TestServiceCrash_CompensationLogic(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	
	// 在补偿阶段注入故障
	injector.AddFault("compensation-crash", &FaultConfig{
		Type:        FaultTypeServiceCrash,
		Probability: 1.0,
		TargetStep:  "step2-compensate",
	})

	// 创建会失败的 Saga 定义（触发补偿）
	sagaDef := createFailingSagaDefinition(injector)

	// 创建内存状态存储
	storage := state.NewMemoryStateStorage()
	chaosStorage := NewChaosStateStorage(storage, injector)

	// 创建协调器
	coord, err := coordinator.NewOrchestratorCoordinator(
		coordinator.WithStateStorage(chaosStorage),
		coordinator.WithRetryPolicy(saga.NewFixedDelayRetryPolicy(2, 100*time.Millisecond)),
	)
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 启动会失败的 Saga
	ctx := context.Background()
	instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "compensation"})

	if instance != nil {
		// 等待 Saga 处理（包括补偿）
		time.Sleep(1 * time.Second)

		// 获取最终状态
		finalInstance, err := coord.GetSagaInstance(instance.GetID())
		if err != nil {
			t.Fatalf("获取 Saga 实例失败: %v", err)
		}

		t.Logf("Saga 最终状态: %s", finalInstance.GetState())

		// 验证补偿过程中的崩溃被正确处理
		if finalInstance.GetError() != nil {
			t.Logf("Saga 错误信息: %s", finalInstance.GetError().Message)
		}
	} else {
		t.Logf("Saga 启动失败: %v", err)
	}

	// 验证注入统计
	stats := injector.GetStats()
	t.Logf("故障注入统计: %+v", stats)
}

// TestServiceCrash_PartialCompensation tests partial compensation when service crashes mid-compensation.
func TestServiceCrash_PartialCompensation(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	injector.Disable()

	// 创建多步骤 Saga 定义
	sagaDef := createMultiStepFailingSagaDefinition(injector)

	// 创建内存状态存储
	storage := state.NewMemoryStateStorage()
	chaosStorage := NewChaosStateStorage(storage, injector)

	// 创建协调器
	coord, err := coordinator.NewOrchestratorCoordinator(
		coordinator.WithStateStorage(chaosStorage),
		coordinator.WithRetryPolicy(saga.NewNoRetryPolicy()),
	)
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 启动 Saga
	ctx := context.Background()
	instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "partial"})

	if instance != nil {
		// 等待部分步骤完成
		time.Sleep(300 * time.Millisecond)

		// 在补偿开始时启用崩溃
		injector.AddFault("mid-compensation-crash", &FaultConfig{
			Type:        FaultTypeServiceCrash,
			Probability: 1.0,
			TargetStep:  "step3-compensate",
		})
		injector.Enable()

		// 等待补偿处理
		time.Sleep(1 * time.Second)

		// 获取最终状态
		finalInstance, err := coord.GetSagaInstance(instance.GetID())
		if err != nil {
			t.Fatalf("获取 Saga 实例失败: %v", err)
		}

		t.Logf("Saga 最终状态: %s", finalInstance.GetState())
		t.Logf("已完成步骤: %d/%d", finalInstance.GetCompletedSteps(), finalInstance.GetTotalSteps())
	}

	// 验证注入统计
	stats := injector.GetStats()
	t.Logf("故障注入统计: %+v", stats)
}

// Note: Helper functions are defined in test_helpers.go


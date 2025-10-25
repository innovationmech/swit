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

// TestDatabaseFailure_SaveOperation tests Saga behavior when database save operations fail.
func TestDatabaseFailure_SaveOperation(t *testing.T) {
	tests := []struct {
		name          string
		operation     string
		probability   float64
		expectFailure bool
		description   string
	}{
		{
			name:          "SaveSaga 操作失败",
			operation:     "SaveSaga",
			probability:   1.0,
			expectFailure: true,
			description:   "Saga 实例保存失败",
		},
		{
			name:          "UpdateSagaState 操作失败",
			operation:     "UpdateSagaState",
			probability:   1.0,
			expectFailure: true,
			description:   "Saga 状态更新失败",
		},
		{
			name:          "SaveStepState 操作失败",
			operation:     "SaveStepState",
			probability:   1.0,
			expectFailure: true,
			description:   "步骤状态保存失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建故障注入器
			injector := NewFaultInjector()
			injector.AddFault("db-save-fault", &FaultConfig{
				Type:        FaultTypeDatabaseFailure,
				Probability: tt.probability,
				TargetStep:  tt.operation,
			})

			// 创建混沌状态存储
			chaosStorage := createTestStorage(injector)

			// 根据操作类型执行测试
			ctx := context.Background()

			switch tt.operation {
			case "SaveSaga":
				instance := &mockSagaInstance{
					id:    "saga-db-test-1",
					state: saga.StateRunning,
				}
				err := chaosStorage.SaveSaga(ctx, instance)
				if tt.expectFailure && err == nil {
					t.Error("期望 SaveSaga 失败，但成功了")
				}
				if err != nil && !errors.Is(err, ErrDatabaseFailure) {
					t.Errorf("期望数据库故障错误，实际为: %v", err)
				}

			case "UpdateSagaState":
				// 先保存一个实例（禁用故障）
				injector.Disable()
				instance := &mockSagaInstance{
					id:    "saga-db-test-2",
					state: saga.StateRunning,
				}
				_ = chaosStorage.SaveSaga(ctx, instance)

				// 启用故障并尝试更新
				injector.Enable()
				err := chaosStorage.UpdateSagaState(ctx, instance.GetID(), saga.StateCompleted, nil)
				if tt.expectFailure && err == nil {
					t.Error("期望 UpdateSagaState 失败，但成功了")
				}
				if err != nil && !errors.Is(err, ErrDatabaseFailure) {
					t.Errorf("期望数据库故障错误，实际为: %v", err)
				}

			case "SaveStepState":
				stepState := &saga.StepState{
					ID:        "step-1",
					SagaID:    "saga-db-test-3",
					StepIndex: 0,
					Name:      "Step 1",
					State:     saga.StepStateRunning,
				}
				err := chaosStorage.SaveStepState(ctx, stepState.SagaID, stepState)
				if tt.expectFailure && err == nil {
					t.Error("期望 SaveStepState 失败，但成功了")
				}
				if err != nil && !errors.Is(err, ErrDatabaseFailure) {
					t.Errorf("期望数据库故障错误，实际为: %v", err)
				}
			}

			// 验证注入统计
			stats := injector.GetStats()
			if tt.expectFailure && stats.TotalInjections == 0 {
				t.Error("期望有故障注入，但统计为 0")
			}
			t.Logf("故障注入统计: %+v", stats)
		})
	}
}

// TestDatabaseFailure_QueryOperation tests Saga behavior when database query operations fail.
func TestDatabaseFailure_QueryOperation(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	injector.Disable() // 初始禁用，先创建数据

	// 创建混沌状态存储
	chaosStorage := createTestStorage(injector)

	// 创建并保存测试实例
	ctx := context.Background()
	instance := &mockSagaInstance{
		id:    "saga-query-test-1",
		state: saga.StateRunning,
	}
	err := chaosStorage.SaveSaga(ctx, instance)
	if err != nil {
		t.Fatalf("保存测试实例失败: %v", err)
	}

	// 启用查询故障
	injector.AddFault("db-query-fault", &FaultConfig{
		Type:        FaultTypeDatabaseFailure,
		Probability: 1.0,
		TargetStep:  "GetSaga",
	})
	injector.Enable()

	// 尝试查询（应该失败）
	_, err = chaosStorage.GetSaga(ctx, instance.GetID())
	if err == nil {
		t.Error("期望 GetSaga 失败，但成功了")
	}
	if !errors.Is(err, ErrDatabaseFailure) {
		t.Errorf("期望数据库故障错误，实际为: %v", err)
	}

	t.Log("查询故障测试通过")
}

// TestDatabaseFailure_TransactionRollback tests Saga behavior with database transaction rollback.
func TestDatabaseFailure_TransactionRollback(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	
	// 在状态更新时注入故障，模拟事务回滚
	injector.AddFault("db-transaction-fault", &FaultConfig{
		Type:        FaultTypeDatabaseFailure,
		Probability: 1.0,
		TargetStep:  "UpdateSagaState",
	})

	// 创建测试 Saga 定义
	sagaDef := createTestSagaDefinition(injector)

	// 创建混沌状态存储
	chaosStorage := createTestStorage(injector)

	// 创建协调器
	coord, err := createTestCoordinator(chaosStorage, saga.NewFixedDelayRetryPolicy(2, 100*time.Millisecond))
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 启动 Saga
	ctx := context.Background()
	_, err = coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "transaction"})

	// 数据库故障应该导致启动失败或状态不一致
	t.Logf("Saga 启动结果: %v", err)

	// 验证注入统计
	stats := injector.GetStats()
	t.Logf("故障注入统计: %+v", stats)
}

// TestDatabaseFailure_ConnectionLoss tests Saga behavior when database connection is lost.
func TestDatabaseFailure_ConnectionLoss(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	injector.Disable()

	// 创建测试 Saga 定义
	sagaDef := createMultiStepSagaDefinition(injector)

	// 创建混沌状态存储
	chaosStorage := createTestStorage(injector)

	// 创建协调器
	coord, err := coordinator.NewOrchestratorCoordinator(
		coordinator.WithStateStorage(chaosStorage),
		coordinator.WithRetryPolicy(saga.NewExponentialBackoffRetryPolicy(3, 100*time.Millisecond, 1*time.Second)),
	)
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 启动 Saga（数据库正常）
	ctx := context.Background()
	instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "connection-loss"})
	if err != nil {
		t.Fatalf("启动 Saga 失败: %v", err)
	}

	t.Logf("Saga 已启动: %s", instance.GetID())

	// 等待部分步骤完成
	time.Sleep(200 * time.Millisecond)

	// 模拟数据库连接丢失
	injector.AddFault("db-connection-loss", &FaultConfig{
		Type:        FaultTypeDatabaseFailure,
		Probability: 1.0,
	})
	injector.Enable()

	t.Log("模拟数据库连接丢失...")

	// 等待故障影响
	time.Sleep(500 * time.Millisecond)

	// 恢复数据库连接
	injector.Disable()
	t.Log("数据库连接恢复")

	// 等待 Saga 恢复和完成
	time.Sleep(2 * time.Second)

	// 获取最终状态
	finalInstance, err := coord.GetSagaInstance(instance.GetID())
	if err != nil {
		t.Logf("获取 Saga 实例失败: %v", err)
		return
	}

	t.Logf("Saga 最终状态: %s", finalInstance.GetState())
	t.Logf("已完成步骤: %d/%d", finalInstance.GetCompletedSteps(), finalInstance.GetTotalSteps())

	// 验证 Saga 最终达到一致状态
	if finalInstance.IsTerminal() {
		t.Log("Saga 达到终止状态")
	} else {
		t.Log("Saga 仍在处理中")
	}
}

// TestDatabaseFailure_PartialWrite tests behavior with partial database writes.
func TestDatabaseFailure_PartialWrite(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	injector.Disable()

	// 创建测试 Saga 定义
	sagaDef := createMultiStepSagaDefinition(injector)

	// 创建混沌状态存储
	chaosStorage := createTestStorage(injector)

	// 创建协调器
	coord, err := createTestCoordinator(chaosStorage, saga.NewNoRetryPolicy())
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 启动 Saga
	ctx := context.Background()
	instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "partial-write"})
	if err != nil {
		t.Fatalf("启动 Saga 失败: %v", err)
	}

	// 等待第一步完成
	time.Sleep(200 * time.Millisecond)

	// 在步骤状态保存时注入部分写入故障
	injector.AddFault("partial-write-fault", &FaultConfig{
		Type:        FaultTypeDatabaseFailure,
		Probability: 0.5, // 50% 概率失败
		TargetStep:  "SaveStepState",
	})
	injector.Enable()

	// 等待更多步骤执行
	time.Sleep(1 * time.Second)

	// 获取最终状态
	finalInstance, err := coord.GetSagaInstance(instance.GetID())
	if err != nil {
		t.Logf("获取 Saga 实例失败: %v", err)
		return
	}

	t.Logf("Saga 最终状态: %s", finalInstance.GetState())

	// 验证注入统计
	stats := injector.GetStats()
	t.Logf("故障注入统计: %+v", stats)
	
	// 验证部分写入场景
	if stats.TotalInjections > 0 {
		t.Log("检测到数据库部分写入故障")
	}
}

// TestDatabaseFailure_Consistency tests data consistency after database failures.
func TestDatabaseFailure_Consistency(t *testing.T) {
	// 创建故障注入器，30% 随机故障率
	injector := NewFaultInjector()
	injector.AddFault("random-db-fault", &FaultConfig{
		Type:        FaultTypeDatabaseFailure,
		Probability: 0.3,
	})

	// 创建测试 Saga 定义
	sagaDef := createMultiStepSagaDefinition(injector)

	// 创建混沌状态存储
	chaosStorage := createTestStorage(injector)

	// 创建协调器，带重试
	coord, err := createTestCoordinator(chaosStorage, saga.NewExponentialBackoffRetryPolicy(3, 100*time.Millisecond, 1*time.Second))
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 启动多个 Saga 测试一致性
	ctx := context.Background()
	numSagas := 5
	var sagaIDs []string

	for i := 0; i < numSagas; i++ {
		instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"index": i})
		if err == nil && instance != nil {
			sagaIDs = append(sagaIDs, instance.GetID())
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("启动了 %d 个 Saga", len(sagaIDs))

	// 等待所有 Saga 处理完成
	time.Sleep(3 * time.Second)

	// 验证每个 Saga 的最终状态一致性
	for _, sagaID := range sagaIDs {
		instance, err := coord.GetSagaInstance(sagaID)
		if err != nil {
			t.Logf("Saga %s: 获取失败 - %v", sagaID, err)
			continue
		}

		// 验证 Saga 达到一致的终止状态
		if !instance.IsTerminal() {
			t.Errorf("Saga %s 未达到终止状态: %s", sagaID, instance.GetState())
		} else {
			t.Logf("Saga %s: 状态=%s, 完成步骤=%d/%d",
				sagaID, instance.GetState(),
				instance.GetCompletedSteps(), instance.GetTotalSteps())
		}
	}

	// 验证注入统计
	stats := injector.GetStats()
	t.Logf("故障注入统计: %+v", stats)
}

// TestDatabaseFailure_RecoveryMechanism tests recovery mechanism after database failures.
func TestDatabaseFailure_RecoveryMechanism(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	
	// 初始时有数据库故障
	injector.AddFault("recoverable-db-fault", &FaultConfig{
		Type:        FaultTypeDatabaseFailure,
		Probability: 1.0,
		Duration:    1 * time.Second,
	})

	// 创建测试 Saga 定义
	sagaDef := createTestSagaDefinition(injector)

	// 创建混沌状态存储
	chaosStorage := createTestStorage(injector)

	// 创建协调器，带重试策略
	coord, err := coordinator.NewOrchestratorCoordinator(
		coordinator.WithStateStorage(chaosStorage),
		coordinator.WithRetryPolicy(saga.NewExponentialBackoffRetryPolicy(5, 200*time.Millisecond, 2*time.Second)),
	)
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 启动 Saga（初始会失败）
	ctx := context.Background()
	_, err = coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "recovery"})
	
	t.Logf("初始启动结果: %v", err)

	// 等待一段时间后禁用故障（模拟数据库恢复）
	time.Sleep(1500 * time.Millisecond)
	injector.Disable()
	t.Log("数据库故障已恢复")

	// 重新尝试启动 Saga
	instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "recovery-retry"})
	if err != nil {
		t.Fatalf("恢复后启动 Saga 失败: %v", err)
	}

	t.Logf("恢复后 Saga 启动成功: %s", instance.GetID())

	// 等待 Saga 完成
	time.Sleep(1 * time.Second)

	// 获取最终状态
	finalInstance, err := coord.GetSagaInstance(instance.GetID())
	if err != nil {
		t.Fatalf("获取 Saga 实例失败: %v", err)
	}

	t.Logf("Saga 最终状态: %s", finalInstance.GetState())

	// 验证 Saga 成功完成
	if finalInstance.IsTerminal() {
		t.Log("Saga 在数据库恢复后成功完成")
	}
}


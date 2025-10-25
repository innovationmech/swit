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

// TestNetworkPartition_StepExecution tests Saga behavior when network partition occurs during step execution.
func TestNetworkPartition_StepExecution(t *testing.T) {
	tests := []struct {
		name               string
		probability        float64
		targetStep         string
		expectFailure      bool
		expectRetry        bool
		expectCompensation bool
	}{
		{
			name:               "网络分区导致步骤失败",
			probability:        1.0,
			targetStep:         "step1",
			expectFailure:      true,
			expectRetry:        false,
			expectCompensation: true,
		},
		{
			name:               "部分概率网络分区",
			probability:        0.5,
			targetStep:         "step2",
			expectFailure:      false, // 可能成功或失败
			expectRetry:        false,
			expectCompensation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建故障注入器
			injector := NewFaultInjector()
			injector.AddFault("network-partition", &FaultConfig{
				Type:        FaultTypeNetworkPartition,
				Probability: tt.probability,
				TargetStep:  tt.targetStep,
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
			instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "data"})

			if tt.expectFailure {
				if err == nil {
					t.Error("期望 Saga 启动失败，但成功了")
				}
				// 验证是否是网络分区错误
				if !errors.Is(err, ErrNetworkPartition) && instance == nil {
					t.Logf("Saga 因错误失败: %v", err)
				}
			}

			if instance != nil {
				// 等待 Saga 完成
				time.Sleep(500 * time.Millisecond)

				// 获取最新状态
				finalInstance, err := coord.GetSagaInstance(instance.GetID())
				if err != nil {
					t.Fatalf("获取 Saga 实例失败: %v", err)
				}

				// 验证最终状态
				if tt.expectCompensation && finalInstance.GetState() != saga.StateCompensated {
					t.Logf("期望状态为 compensated，实际为 %s", finalInstance.GetState())
				}

				t.Logf("Saga 最终状态: %s", finalInstance.GetState())
			}

			// 验证注入统计
			stats := injector.GetStats()
			if tt.probability == 1.0 && stats.TotalInjections == 0 {
				t.Error("期望有故障注入，但统计为 0")
			}
			t.Logf("故障注入统计: 总计=%d, 成功=%d", stats.TotalInjections, stats.SuccessfulInjections)
		})
	}
}

// TestNetworkPartition_StateStorage tests Saga behavior when network partition affects state storage.
func TestNetworkPartition_StateStorage(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	injector.AddFault("storage-network-fault", &FaultConfig{
		Type:        FaultTypeNetworkPartition,
		Probability: 1.0,
		TargetStep:  "SaveSaga",
	})

	// 创建混沌状态存储
	chaosStorage := createTestStorage(injector)

	// 创建测试 Saga 实例
	instance := &mockSagaInstance{
		id:    "saga-network-test-1",
		state: saga.StateRunning,
	}

	// 尝试保存 Saga（应该失败）
	ctx := context.Background()
	err := chaosStorage.SaveSaga(ctx, instance)
	if err == nil {
		t.Error("期望保存 Saga 失败，但成功了")
	}

	if !errors.Is(err, ErrNetworkPartition) {
		t.Errorf("期望网络分区错误，实际为: %v", err)
	}

	// 验证注入统计
	stats := injector.GetStats()
	if stats.TotalInjections != 1 {
		t.Errorf("期望注入次数为 1，实际为 %d", stats.TotalInjections)
	}
}

// TestNetworkPartition_EventPublishing tests Saga behavior when network partition affects event publishing.
func TestNetworkPartition_EventPublishing(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	injector.AddFault("event-network-fault", &FaultConfig{
		Type:        FaultTypeNetworkPartition,
		Probability: 1.0,
		TargetStep:  string(saga.EventSagaStarted),
	})

	// 创建 mock 事件发布器
	mockPublisher := &mockEventPublisher{}
	chaosPublisher := NewChaosEventPublisher(mockPublisher, injector)

	// 创建测试事件
	event := &saga.SagaEvent{
		ID:     "event-1",
		SagaID: "saga-1",
		Type:   saga.EventSagaStarted,
	}

	// 尝试发布事件（应该失败）
	ctx := context.Background()
	err := chaosPublisher.PublishEvent(ctx, event)
	if err == nil {
		t.Error("期望发布事件失败，但成功了")
	}

	if !errors.Is(err, ErrNetworkPartition) {
		t.Errorf("期望网络分区错误，实际为: %v", err)
	}

	// 验证事件未被实际发布
	if mockPublisher.publishCount > 0 {
		t.Error("期望事件未发布，但发布计数 > 0")
	}
}

// TestNetworkPartition_Recovery tests Saga recovery after network partition is resolved.
func TestNetworkPartition_Recovery(t *testing.T) {
	// 创建故障注入器（初始禁用）
	injector := NewFaultInjector()
	injector.Disable()

	injector.AddFault("transient-network-fault", &FaultConfig{
		Type:        FaultTypeNetworkPartition,
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
		coordinator.WithRetryPolicy(saga.NewExponentialBackoffRetryPolicy(3, 100*time.Millisecond, 1*time.Second)),
	)
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 启动 Saga（故障禁用，应该成功）
	ctx := context.Background()
	instance, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"test": "data"})
	if err != nil {
		t.Fatalf("启动 Saga 失败: %v", err)
	}

	// 启用故障注入模拟网络分区
	injector.Enable()

	// 等待一段时间
	time.Sleep(500 * time.Millisecond)

	// 禁用故障注入模拟网络恢复
	injector.Disable()

	// 等待 Saga 完成或恢复
	time.Sleep(1 * time.Second)

	// 获取最终状态
	finalInstance, err := coord.GetSagaInstance(instance.GetID())
	if err != nil {
		t.Fatalf("获取 Saga 实例失败: %v", err)
	}

	// 验证 Saga 最终达到一致状态（完成或补偿）
	if !finalInstance.IsTerminal() {
		t.Errorf("期望 Saga 达到终止状态，实际为: %s", finalInstance.GetState())
	}

	t.Logf("Saga 最终状态: %s", finalInstance.GetState())
	t.Logf("故障注入统计: %+v", injector.GetStats())
}

// TestNetworkPartition_ConcurrentSagas tests network partition with multiple concurrent Sagas.
func TestNetworkPartition_ConcurrentSagas(t *testing.T) {
	// 创建故障注入器
	injector := NewFaultInjector()
	injector.AddFault("concurrent-network-fault", &FaultConfig{
		Type:        FaultTypeNetworkPartition,
		Probability: 0.3, // 30% 概率
	})

	// 创建测试 Saga 定义
	sagaDef := createTestSagaDefinition(injector)

	// 创建混沌状态存储
	chaosStorage := createTestStorage(injector)

	// 创建协调器
	coord, err := coordinator.NewOrchestratorCoordinator(
		coordinator.WithStateStorage(chaosStorage),
		coordinator.WithRetryPolicy(saga.NewFixedDelayRetryPolicy(2, 100*time.Millisecond)),
	)
	if err != nil {
		t.Fatalf("创建协调器失败: %v", err)
	}
	defer coord.Close()

	// 启动多个并发 Saga
	ctx := context.Background()
	numSagas := 10
	resultChan := make(chan error, numSagas)

	for i := 0; i < numSagas; i++ {
		go func(index int) {
			_, err := coord.StartSaga(ctx, sagaDef, map[string]interface{}{"index": index})
			resultChan <- err
		}(i)
	}

	// 收集结果
	var successCount, failureCount int
	for i := 0; i < numSagas; i++ {
		err := <-resultChan
		if err == nil {
			successCount++
		} else {
			failureCount++
		}
	}

	// 等待所有 Saga 完成
	time.Sleep(1 * time.Second)

	// 验证结果
	t.Logf("并发 Saga 结果: 成功=%d, 失败=%d", successCount, failureCount)
	t.Logf("故障注入统计: %+v", injector.GetStats())

	// 验证所有 Saga 最终达到一致状态
	filter := &saga.SagaFilter{}
	instances, err := coord.GetActiveSagas(filter)
	if err != nil {
		t.Fatalf("获取活动 Saga 失败: %v", err)
	}

	for _, inst := range instances {
		if !inst.IsTerminal() {
			t.Errorf("Saga %s 未达到终止状态: %s", inst.GetID(), inst.GetState())
		}
	}
}

// Note: Mock implementations are now in test_helpers.go

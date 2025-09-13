package messaging_test

import (
	"context"
	"testing"
	"time"

	"github.com/innovationmech/swit/internal/testing"
	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGracefulShutdownLifecycle 覆盖：启动 → 优雅关停（含在途消息排空）。
func TestGracefulShutdownLifecycle(t *testing.T) {
	// 构建夹具：协调器 + Broker + 1个处理器
	h := testingx.NewLifecycleHarness().WithHandlers(1)
	require.NoError(t, h.Register())

	// 启动
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, h.Start(ctx))

	// 发起优雅关停
	shutdownCfg := &messaging.ShutdownConfig{
		Timeout:             5 * time.Second,
		ForceTimeout:        10 * time.Second,
		DrainTimeout:        500 * time.Millisecond,
		ReportInterval:      100 * time.Millisecond,
		MaxInflightMessages: 100,
	}

	manager, err := h.ShutdownGracefully(ctx, shutdownCfg)
	require.NoError(t, err)

	// 模拟在途消息，短时间后完成
	h.SimulateInflight(manager, 3, 100*time.Millisecond)

	// 等待完成
	require.NoError(t, manager.WaitForCompletion())

	status := manager.GetShutdownStatus()
	assert.Equal(t, messaging.ShutdownPhaseCompleted, status.Phase)
	assert.Contains(t, status.CompletedSteps, "stop_accepting_new_messages")
	assert.Contains(t, status.CompletedSteps, "drain_inflight_messages")
	assert.Contains(t, status.CompletedSteps, "close_broker_connections")
	assert.Contains(t, status.CompletedSteps, "cleanup_event_handlers")
	assert.Contains(t, status.CompletedSteps, "release_coordinator_resources")
}

// TestGracefulShutdownForceTimeout 覆盖：达到强制超时，返回错误。
func TestGracefulShutdownForceTimeout(t *testing.T) {
	h := testingx.NewLifecycleHarness().WithHandlers(1)
	require.NoError(t, h.Register())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, h.Start(ctx))

	shutdownCfg := &messaging.ShutdownConfig{
		Timeout:             300 * time.Millisecond,
		ForceTimeout:        300 * time.Millisecond,
		DrainTimeout:        5 * time.Second, // 故意大于 Timeout
		ReportInterval:      100 * time.Millisecond,
		MaxInflightMessages: 100,
	}

	manager, err := h.ShutdownGracefully(ctx, shutdownCfg)
	require.NoError(t, err)

	// 模拟长时间在途，超过强制超时
	h.SimulateInflight(manager, 1, 3*time.Second)

	// 等待应返回强制超时错误
	err = manager.WaitForCompletion()
	require.Error(t, err)
}

// TestCoordinatorStopFallback 覆盖：直接 Stop 路径（非 GracefulShutdown）。
func TestCoordinatorStopFallback(t *testing.T) {
	h := testingx.NewLifecycleHarness().WithHandlers(2)
	require.NoError(t, h.Register())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, h.Start(ctx))

	// 直接 Stop
	require.NoError(t, h.Coordinator.Stop(ctx))
	assert.False(t, h.Coordinator.IsStarted())
}

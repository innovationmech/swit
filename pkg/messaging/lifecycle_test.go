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

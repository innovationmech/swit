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

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/coordinator"
	"github.com/innovationmech/swit/pkg/saga/state"
	"github.com/innovationmech/swit/pkg/saga/state/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// noopEventPublisher is a simple no-op event publisher for the example.
type noopEventPublisher struct{}

func (p *noopEventPublisher) PublishEvent(ctx context.Context, event *saga.SagaEvent) error {
	return nil
}

func (p *noopEventPublisher) Subscribe(filter saga.EventFilter, handler saga.EventHandler) (saga.EventSubscription, error) {
	return nil, nil
}

func (p *noopEventPublisher) Unsubscribe(subscription saga.EventSubscription) error {
	return nil
}

func (p *noopEventPublisher) Close() error {
	return nil
}

func main() {
	// 初始化日志
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	logger.Info("Starting Saga Recovery Example")

	// 创建状态存储（使用内存存储作为示例）
	stateStorage := storage.NewMemoryStateStorage()

	// 创建一个简单的事件发布器（no-op 实现）
	eventPublisher := &noopEventPublisher{}

	// 创建 Saga 协调器
	orchestrator, err := coordinator.NewOrchestratorCoordinator(
		&coordinator.OrchestratorConfig{
			StateStorage:   stateStorage,
			EventPublisher: eventPublisher,
		},
	)
	if err != nil {
		logger.Fatal("Failed to create orchestrator", zap.Error(err))
	}

	// 创建恢复管理器配置
	recoveryConfig := state.DefaultRecoveryConfig()
	recoveryConfig.EnableAutoRecovery = true
	recoveryConfig.CheckInterval = 30 * time.Second
	recoveryConfig.RecoveryTimeout = 30 * time.Second

	// 创建恢复管理器
	recoveryManager, err := state.NewRecoveryManager(
		recoveryConfig,
		stateStorage,
		orchestrator,
		logger,
	)
	if err != nil {
		logger.Fatal("Failed to create recovery manager", zap.Error(err))
	}

	// 添加日志告警通知器
	logNotifier := state.NewLogAlertNotifier(logger)
	recoveryManager.AddAlertNotifier(logNotifier)

	// 注册恢复事件监听器
	recoveryManager.AddEventListener(func(event *state.RecoveryEvent) {
		logger.Info("Recovery event",
			zap.String("type", string(event.Type)),
			zap.String("saga_id", event.SagaID),
			zap.String("message", event.Message),
		)
	})

	// 启动恢复管理器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := recoveryManager.Start(ctx); err != nil {
		logger.Fatal("Failed to start recovery manager", zap.Error(err))
	}

	// 启动 Prometheus 指标端点
	go func() {
		mux := http.NewServeMux()

		// 添加恢复指标
		if metrics := recoveryManager.GetMetrics(); metrics != nil {
			if registry := metrics.GetRegistry(); registry != nil {
				mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
			}
		}

		// 添加健康检查端点
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// 添加恢复统计端点
		mux.HandleFunc("/recovery/stats", func(w http.ResponseWriter, r *http.Request) {
			if snapshot := recoveryManager.GetMetricsSnapshot(); snapshot != nil {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{
					"total_attempts": %d,
					"successful_recoveries": %d,
					"failed_recoveries": %d,
					"currently_recovering": %d,
					"success_rate": %.2f,
					"average_duration": "%s"
				}`,
					snapshot.TotalAttempts,
					snapshot.SuccessfulRecoveries,
					snapshot.FailedRecoveries,
					snapshot.CurrentlyRecovering,
					snapshot.SuccessRate,
					snapshot.AverageDuration.String(),
				)
			}
		})

		logger.Info("Starting metrics server on :9090")
		if err := http.ListenAndServe(":9090", mux); err != nil {
			logger.Error("Metrics server failed", zap.Error(err))
		}
	}()

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	logger.Info("Recovery system running. Press Ctrl+C to stop.")
	logger.Info("Metrics available at http://localhost:9090/metrics")
	logger.Info("Recovery stats available at http://localhost:9090/recovery/stats")

	<-sigCh
	logger.Info("Shutting down...")

	// 停止恢复管理器
	if err := recoveryManager.Stop(); err != nil {
		logger.Error("Error stopping recovery manager", zap.Error(err))
	}

	// 关闭恢复管理器
	if err := recoveryManager.Close(); err != nil {
		logger.Error("Error closing recovery manager", zap.Error(err))
	}

	logger.Info("Shutdown complete")
}

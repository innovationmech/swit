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

// Package main demonstrates how to use the Saga Event Publisher for reliable event publishing.
//
// This is a documentation and demonstration example. For actual running tests,
// please refer to pkg/saga/messaging/*_test.go files.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

func main() {
	log.Println("=== Saga Event Publisher 使用示例 ===")
	log.Println()
	log.Println("这是一个文档示例，展示了如何使用 Saga Event Publisher 的各种功能。")
	log.Println("要查看实际运行的测试，请参考 pkg/saga/messaging/*_test.go")
	log.Println()

	// 示例 1: 基础 API 使用
	log.Println("示例 1: 基础 API 使用")
	demonstrateBasicUsage()
	log.Println()

	// 示例 2: 批量发布
	log.Println("示例 2: 批量发布")
	demonstrateBatchPublishing()
	log.Println()

	// 示例 3: 事务性发布
	log.Println("示例 3: 事务性发布")
	demonstrateTransactionalPublishing()
	log.Println()

	// 示例 4: 可靠性配置
	log.Println("示例 4: 可靠性配置")
	demonstrateReliabilityConfig()
	log.Println()

	// 示例 5: 性能优化
	log.Println("示例 5: 性能优化")
	demonstratePerformanceOptimization()
	log.Println()

	// 示例 6: 监控和指标
	log.Println("示例 6: 监控和指标")
	demonstrateMonitoring()
	log.Println()

	log.Println("所有示例完成！")
	log.Println()
	log.Println("要运行实际的发布测试，请：")
	log.Println("1. 启动 NATS 服务器: docker run -p 4222:4222 nats:latest")
	log.Println("2. 查看实际测试: pkg/saga/messaging/*_test.go")
	log.Println("3. 运行测试: go test ./pkg/saga/messaging/... -v")
}

// demonstrateBasicUsage 演示基础 API 使用
func demonstrateBasicUsage() {
	printCodeExample(`
// 1. 创建 broker (使用 adapter 系统)
broker, err := adapters.CreateBrokerWithAdapter(&messaging.BrokerConfig{
    Type:      messaging.BrokerTypeNATS,
    Endpoints: []string{"nats://localhost:4222"},
})
if err != nil {
    return err
}

// 2. 连接 broker
ctx := context.Background()
if err := broker.Connect(ctx); err != nil {
    return err
}
defer broker.Close()

// 3. 创建 Saga 事件发布器
publisher, err := sagamessaging.NewSagaEventPublisher(broker, &sagamessaging.PublisherConfig{
    TopicPrefix:    "saga.events",
    SerializerType: "json",
    RetryAttempts:  3,
})
if err != nil {
    return err
}
defer publisher.Close()

// 4. 发布事件
event := &saga.SagaEvent{
    ID:        "evt-001",
    SagaID:    "saga-001",
    Type:      saga.EventSagaStarted,
    Timestamp: time.Now(),
    Version:   "1.0",
}

err = publisher.PublishSagaEvent(ctx, event)
	`)

	// 创建示例事件用于演示
	event := &saga.SagaEvent{
		ID:        "evt-001",
		SagaID:    "saga-001",
		Type:      saga.EventSagaStarted,
		Timestamp: time.Now(),
		Version:   "1.0",
	}

	log.Printf("✅ 创建了示例事件: %s (SagaID: %s, Type: %s)", event.ID, event.SagaID, event.Type)
}

// demonstrateBatchPublishing 演示批量发布
func demonstrateBatchPublishing() {
	printCodeExample(`
// 创建多个事件
events := []*saga.SagaEvent{
    {
        ID:        "evt-batch-001",
        SagaID:    "saga-batch-001",
        Type:      saga.EventSagaStepStarted,
        Timestamp: time.Now(),
        Version:   "1.0",
    },
    {
        ID:        "evt-batch-002",
        SagaID:    "saga-batch-001",
        Type:      saga.EventSagaStepCompleted,
        Timestamp: time.Now(),
        Version:   "1.0",
    },
    {
        ID:        "evt-batch-003",
        SagaID:    "saga-batch-001",
        Type:      saga.EventSagaCompleted,
        Timestamp: time.Now(),
        Version:   "1.0",
    },
}

// 批量发布 - 性能提升 5-10 倍
if err := publisher.PublishBatch(ctx, events); err != nil {
    log.Printf("批量发布失败: %v", err)
    return
}

log.Printf("✅ 批量发布成功: %d 个事件", len(events))
	`)

	log.Printf("💡 性能提示: 批量发布比单个发布快 5-10 倍")
}

// demonstrateTransactionalPublishing 演示事务性发布
func demonstrateTransactionalPublishing() {
	printCodeExample(`
// 使用事务发布多个相关事件
err := publisher.WithTransaction(ctx, func(ctx context.Context, txPublisher *sagamessaging.TransactionalEventPublisher) error {
    // 在事务中发布多个事件
    events := []*saga.SagaEvent{
        {
            ID:        "evt-tx-001",
            SagaID:    "saga-tx-001",
            Type:      saga.EventSagaStarted,
            Timestamp: time.Now(),
            Version:   "1.0",
        },
        {
            ID:        "evt-tx-002",
            SagaID:    "saga-tx-001",
            Type:      saga.EventSagaStepStarted,
            Timestamp: time.Now(),
            Version:   "1.0",
        },
    }

    for _, event := range events {
        if err := txPublisher.PublishEvent(ctx, event); err != nil {
            return err // 自动回滚
        }
    }

    return nil // 自动提交
})

if err != nil {
    log.Printf("事务性发布失败: %v", err)
} else {
    log.Printf("✅ 事务提交成功")
}
	`)

	log.Printf("💡 事务保证: 所有事件要么全部成功，要么全部失败")
}

// demonstrateReliabilityConfig 演示可靠性配置
func demonstrateReliabilityConfig() {
	printCodeExample(`
publisher, err := sagamessaging.NewSagaEventPublisher(broker, &sagamessaging.PublisherConfig{
    TopicPrefix:    "saga.events",
    SerializerType: "json",
    EnableConfirm:  true, // 启用发布确认
    RetryAttempts:  5,    // 最多重试 5 次
    RetryInterval:  500 * time.Millisecond,
    Timeout:        10 * time.Second,
    Reliability: &sagamessaging.ReliabilityConfig{
        // 重试配置
        EnableRetry:      true,
        MaxRetryAttempts: 5,
        InitialRetryBackoff: 500 * time.Millisecond,
        MaxRetryInterval: 5 * time.Second,
        
        // 确认配置
        EnableConfirm:  true,
        ConfirmTimeout: 10 * time.Second,
        
        // 死信队列配置
        EnableDLQ:  true,
        DLQTopic:   "saga.dlq",
    },
})
	`)

	log.Printf("✅ 可靠性机制:")
	log.Printf("   - 自动重试（指数退避）")
	log.Printf("   - 发布确认（broker 确认持久化）")
	log.Printf("   - 死信队列（保存失败消息）")
}

// demonstratePerformanceOptimization 演示性能优化
func demonstratePerformanceOptimization() {
	printCodeExample(`
// 1. 使用 Protobuf 序列化 - 更快，体积更小
&sagamessaging.PublisherConfig{
    SerializerType: "protobuf", // 比 JSON 快 2-3 倍，小 30-50%
}

// 2. 使用批量发布 - 减少网络往返
publisher.PublishBatch(ctx, events)

// 3. 使用异步发布 - 非阻塞操作
publisher.PublishBatchAsync(ctx, events, func(err error) {
    if err != nil {
        log.Printf("异步发布失败: %v", err)
    }
})

// 4. 复用发布器实例 - 避免重复创建连接
// 在应用启动时创建一次，在整个生命周期中复用
var globalPublisher *sagamessaging.SagaEventPublisher

func init() {
    // 创建并复用
    globalPublisher, _ = sagamessaging.NewSagaEventPublisher(...)
}
	`)

	log.Printf("✅ 性能优化建议:")
	log.Printf("   - Protobuf 序列化: 性能提升 2-3 倍")
	log.Printf("   - 批量发布: 性能提升 5-10 倍")
	log.Printf("   - 异步发布: 提高吞吐量")
	log.Printf("   - 复用实例: 减少连接开销")
}

// demonstrateMonitoring 演示监控和指标
func demonstrateMonitoring() {
	printCodeExample(`
// 获取发布指标
metrics := publisher.GetMetrics()

log.Printf("发布统计:")
log.Printf("  总数: %d", metrics.TotalPublished)
log.Printf("  成功: %d", metrics.TotalSuccessful)
log.Printf("  失败: %d", metrics.TotalFailed)
log.Printf("  批次数: %d", metrics.TotalBatches)
log.Printf("  平均延迟: %s", metrics.AverageLatency)
log.Printf("  平均批次大小: %.2f", metrics.AverageBatchSize)

// 获取可靠性指标
reliabilityMetrics := publisher.GetReliabilityMetrics()
if reliabilityMetrics != nil {
    log.Printf("可靠性指标:")
    log.Printf("  总重试次数: %d", reliabilityMetrics.TotalRetries)
    log.Printf("  成功的重试: %d", reliabilityMetrics.SuccessfulRetries)
    log.Printf("  失败的重试: %d", reliabilityMetrics.FailedRetries)
    log.Printf("  成功率: %.2f%%", reliabilityMetrics.GetSuccessRate()*100)
    log.Printf("  DLQ 消息数: %d", reliabilityMetrics.DLQMessagesCount)
}

// Prometheus 指标集成
// 发布器会自动注册 Prometheus 指标
// 通过 /metrics 端点暴露指标
	`)

	log.Printf("✅ 监控指标类型:")
	log.Printf("   - 发布统计（成功/失败/延迟）")
	log.Printf("   - 可靠性指标（重试/成功率/DLQ）")
	log.Printf("   - Prometheus 集成（自动注册）")
}

// printCodeExample 打印代码示例
func printCodeExample(code string) {
	fmt.Println("```go")
	fmt.Print(code)
	fmt.Println("```")
	fmt.Println()
}

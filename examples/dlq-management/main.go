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
	"log"
	"time"

	"github.com/innovationmech/swit/pkg/resilience"
)

func main() {
	// 创建 DLQ 管理器配置
	config := resilience.DLQManagerConfig{
		Enabled:       true,
		Policy:        resilience.DeadLetterPolicyNotify,
		MaxRetries:    3,
		RetryInterval: 5 * time.Second,
		TTL:           7 * 24 * time.Hour,
		MaxQueueSize:  1000,
		NotifyCallback: func(msg *resilience.DeadLetterMessage) {
			log.Printf("DLQ notification: message %s from topic %s failed with reason: %s",
				msg.ID, msg.OriginalTopic, msg.FailureReason)
		},
	}

	// 创建 DLQ 管理器
	mgr, err := resilience.NewDLQManager(config)
	if err != nil {
		log.Fatalf("failed to create DLQ manager: %v", err)
	}
	defer mgr.Close()

	// 创建 Prometheus 指标（可选）
	promMetrics := resilience.NewDLQPrometheusMetrics("example", "dlq")

	ctx := context.Background()

	// 示例 1: 存储失败的消息
	msg1 := &resilience.DeadLetterMessage{
		ID:            "msg-001",
		OriginalTopic: "orders",
		Payload:       []byte(`{"order_id": "12345", "amount": 100.50}`),
		Headers: map[string]string{
			"content-type": "application/json",
			"x-trace-id":   "abc-123",
		},
		RetryCount:    0,
		FailureReason: "database connection timeout",
		StackTrace:    "error occurred at order_service.go:42",
	}

	if err := mgr.Route(ctx, msg1); err != nil {
		log.Printf("failed to route message: %v", err)
	}
	promMetrics.RecordMessage()
	promMetrics.RecordStore(mgr.GetQueueSize())

	// 示例 2: 存储多次重试后失败的消息
	msg2 := &resilience.DeadLetterMessage{
		ID:            "msg-002",
		OriginalTopic: "payments",
		Payload:       []byte(`{"payment_id": "pay-789", "amount": 250.00}`),
		RetryCount:    3,
		FailureReason: "payment gateway unavailable",
	}

	if err := mgr.Route(ctx, msg2); err != nil {
		log.Printf("failed to route message: %v", err)
	}
	promMetrics.RecordMessage()
	promMetrics.RecordStore(mgr.GetQueueSize())

	// 示例 3: 查询死信消息
	time.Sleep(100 * time.Millisecond)

	messages := mgr.ListMessages()
	fmt.Printf("\n=== DLQ Messages (%d total) ===\n", len(messages))
	for _, msg := range messages {
		fmt.Printf("ID: %s\n", msg.ID)
		fmt.Printf("  Topic: %s\n", msg.OriginalTopic)
		fmt.Printf("  Retry Count: %d\n", msg.RetryCount)
		fmt.Printf("  Failure: %s\n", msg.FailureReason)
		fmt.Printf("  First Failed: %s\n", msg.FirstFailureTime.Format(time.RFC3339))
		fmt.Println()
	}

	// 示例 4: 获取特定消息
	if retrieved, err := mgr.GetMessage("msg-001"); err == nil {
		fmt.Printf("Retrieved message: %s\n", retrieved.ID)
		fmt.Printf("  Payload: %s\n", string(retrieved.Payload))
	}

	// 示例 5: 查看度量指标
	metrics := mgr.GetMetrics()
	fmt.Printf("\n=== DLQ Metrics ===\n")
	fmt.Printf("Total Messages: %d\n", metrics.GetTotalMessages())
	fmt.Printf("Messages Stored: %d\n", metrics.GetMessagesStored())
	fmt.Printf("Messages Discarded: %d\n", metrics.GetMessagesDiscarded())
	fmt.Printf("Messages Retried: %d\n", metrics.GetMessagesRetried())
	fmt.Printf("Messages Removed: %d\n", metrics.GetMessagesRemoved())
	fmt.Printf("Messages Expired: %d\n", metrics.GetMessagesExpired())
	fmt.Printf("Messages Evicted: %d\n", metrics.GetMessagesEvicted())
	fmt.Printf("Notifications Sent: %d\n", metrics.GetNotificationsSent())
	fmt.Printf("Current Queue Size: %d\n", mgr.GetQueueSize())

	// 示例 6: 移除已处理的消息
	if err := mgr.RemoveMessage("msg-001"); err != nil {
		log.Printf("failed to remove message: %v", err)
	} else {
		fmt.Printf("\n=== Removed message msg-001 ===\n")
		promMetrics.RecordRemoval(mgr.GetQueueSize())
	}

	fmt.Printf("Queue size after removal: %d\n", mgr.GetQueueSize())

	// 示例 7: 测试容量限制
	fmt.Printf("\n=== Testing capacity limits ===\n")
	limitedConfig := resilience.DLQManagerConfig{
		Enabled:      true,
		Policy:       resilience.DeadLetterPolicyStore,
		MaxRetries:   3,
		TTL:          24 * time.Hour,
		MaxQueueSize: 3, // 只允许 3 条消息
	}

	limitedMgr, err := resilience.NewDLQManager(limitedConfig)
	if err != nil {
		log.Fatalf("failed to create limited DLQ manager: %v", err)
	}
	defer limitedMgr.Close()

	// 添加超过容量的消息
	for i := 0; i < 5; i++ {
		msg := &resilience.DeadLetterMessage{
			ID:            fmt.Sprintf("msg-%03d", i),
			OriginalTopic: "test-topic",
			Payload:       []byte(fmt.Sprintf("payload %d", i)),
			FailureReason: "test failure",
		}
		if err := limitedMgr.Route(ctx, msg); err != nil {
			log.Printf("failed to route message: %v", err)
		}
	}

	fmt.Printf("Limited queue size (max 3): %d\n", limitedMgr.GetQueueSize())
	limitedMetrics := limitedMgr.GetMetrics()
	fmt.Printf("Messages evicted: %d\n", limitedMetrics.GetMessagesEvicted())

	// 清理
	fmt.Printf("\n=== Cleaning up ===\n")
	if err := mgr.Purge(); err != nil {
		log.Printf("failed to purge DLQ: %v", err)
	} else {
		fmt.Printf("DLQ purged successfully. Queue size: %d\n", mgr.GetQueueSize())
	}
}

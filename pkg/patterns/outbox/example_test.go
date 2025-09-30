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

package outbox_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/innovationmech/swit/pkg/patterns/outbox"
)

// Order 示例业务实体
type Order struct {
	ID     string
	UserID string
	Amount float64
	Status string
}

// OrderCreatedEvent 订单创建事件
type OrderCreatedEvent struct {
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

// Example_basicUsage 演示基本的 outbox 使用
func Example_basicUsage() {
	// 创建存储和发布器（实际应用中应使用数据库存储）
	storage := outbox.NewInMemoryStorage()
	publisher := outbox.NewPublisher(storage)

	// 创建订单事件
	event := OrderCreatedEvent{
		OrderID:   "order-123",
		UserID:    "user-456",
		Amount:    99.99,
		Timestamp: time.Now(),
	}

	// 序列化事件
	eventData, _ := json.Marshal(event)

	// 保存到 outbox
	entry := &outbox.OutboxEntry{
		ID:          uuid.NewString(),
		AggregateID: event.OrderID,
		EventType:   "order.created",
		Topic:       "orders.events",
		Payload:     eventData,
		Headers: map[string]string{
			"user_id": event.UserID,
		},
	}

	if err := publisher.SaveForPublish(context.Background(), entry); err != nil {
		log.Fatalf("Failed to save event: %v", err)
	}

	fmt.Println("Event saved to outbox for async publishing")
	// Output: Event saved to outbox for async publishing
}

// Example_transactionalPublish 演示事务性发布
func Example_transactionalPublish() {
	// 创建支持事务的存储
	storage := outbox.NewInMemoryStorage()
	publisher := outbox.NewPublisher(storage)

	// 开始事务
	tx, err := storage.BeginTx(context.Background())
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	// 模拟保存业务数据（在真实应用中，这里会执行 SQL）
	order := Order{
		ID:     "order-789",
		UserID: "user-123",
		Amount: 199.99,
		Status: "created",
	}

	// 在事务中执行业务逻辑
	// tx.Exec(ctx, "INSERT INTO orders ...")

	// 创建事件
	event := OrderCreatedEvent{
		OrderID:   order.ID,
		UserID:    order.UserID,
		Amount:    order.Amount,
		Timestamp: time.Now(),
	}

	eventData, _ := json.Marshal(event)

	// 在同一事务中保存 outbox 条目
	entry := &outbox.OutboxEntry{
		ID:          uuid.NewString(),
		AggregateID: order.ID,
		EventType:   "order.created",
		Topic:       "orders.events",
		Payload:     eventData,
	}

	if err := publisher.SaveWithTransaction(context.Background(), tx, entry); err != nil {
		_ = tx.Rollback(context.Background())
		log.Fatalf("Failed to save event in transaction: %v", err)
	}

	// 提交事务（业务数据和事件一起提交）
	if err := tx.Commit(context.Background()); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	fmt.Println("Order and event saved atomically")
	// Output: Order and event saved atomically
}

// Example_processorWithWorker 演示启动后台处理器
func Example_processorWithWorker() {
	// 注意：这个示例需要一个真实的消息发布器
	// 这里使用 nil 仅作为演示，实际应用中需要提供真实的发布器

	storage := outbox.NewInMemoryStorage()

	// 配置处理器
	config := outbox.ProcessorConfig{
		PollInterval:    5 * time.Second,  // 每5秒轮询一次
		BatchSize:       100,               // 每批处理100条
		MaxRetries:      3,                 // 最多重试3次
		WorkerCount:     1,                 // 单个 worker
		CleanupInterval: 24 * time.Hour,    // 每天清理一次
		CleanupAfter:    7 * 24 * time.Hour, // 清理7天前的已处理消息
	}

	// 创建处理器（需要提供真实的消息发布器）
	// processor := outbox.NewProcessor(storage, messagePublisher, config)

	// 启动处理器
	// ctx := context.Background()
	// if err := processor.Start(ctx); err != nil {
	//     log.Fatalf("Failed to start processor: %v", err)
	// }

	// 在应用关闭时停止处理器
	// defer func() {
	//     stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	//     defer cancel()
	//     if err := processor.Stop(stopCtx); err != nil {
	//         log.Printf("Error stopping processor: %v", err)
	//     }
	// }()

	fmt.Println("Outbox processor configuration ready")
	// Output: Outbox processor configuration ready
}

// Example_batchPublish 演示批量保存事件
func Example_batchPublish() {
	storage := outbox.NewInMemoryStorage()
	publisher := outbox.NewPublisher(storage)

	// 创建多个事件
	var entries []*outbox.OutboxEntry

	for i := 1; i <= 5; i++ {
		event := OrderCreatedEvent{
			OrderID:   fmt.Sprintf("order-%d", i),
			UserID:    "user-123",
			Amount:    float64(i) * 10.0,
			Timestamp: time.Now(),
		}

		eventData, _ := json.Marshal(event)

		entry := &outbox.OutboxEntry{
			ID:          uuid.NewString(),
			AggregateID: event.OrderID,
			EventType:   "order.created",
			Topic:       "orders.events",
			Payload:     eventData,
		}

		entries = append(entries, entry)
	}

	// 批量保存
	if err := publisher.SaveForPublishBatch(context.Background(), entries); err != nil {
		log.Fatalf("Failed to save batch: %v", err)
	}

	fmt.Printf("Saved %d events to outbox\n", len(entries))
	// Output: Saved 5 events to outbox
}

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

package inbox_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/innovationmech/swit/pkg/messaging"
	"github.com/innovationmech/swit/pkg/patterns/inbox"
)

// OrderHandler 处理订单消息
type OrderHandler struct {
	name string
}

func (h *OrderHandler) Handle(ctx context.Context, msg *messaging.Message) error {
	var order map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &order); err != nil {
		return fmt.Errorf("failed to unmarshal order: %w", err)
	}

	// 处理订单逻辑
	fmt.Printf("Processing order: %v\n", order)
	return nil
}

func (h *OrderHandler) GetName() string {
	return h.name
}

// Example_basicUsage 演示基本使用
func Example_basicUsage() {
	// 创建存储
	storage := inbox.NewInMemoryStorage()

	// 创建处理器
	handler := &OrderHandler{name: "order-handler"}
	config := inbox.DefaultProcessorConfig()

	processor, err := inbox.NewProcessor(storage, handler, config)
	if err != nil {
		log.Fatalf("Failed to create processor: %v", err)
	}

	// 创建消息
	msg := &messaging.Message{
		ID:    "order-123",
		Topic: "orders",
		Payload: []byte(`{
			"order_id": "123",
			"customer": "John Doe",
			"amount": 100.0
		}`),
	}

	ctx := context.Background()

	// 第一次处理
	if err := processor.Process(ctx, msg); err != nil {
		log.Fatalf("Failed to process message: %v", err)
	}
	fmt.Println("First processing completed")

	// 第二次处理（幂等）
	if err := processor.Process(ctx, msg); err != nil {
		log.Fatalf("Failed to process message: %v", err)
	}
	fmt.Println("Second processing completed (idempotent)")

	// Output:
	// Processing order: map[amount:100 customer:John Doe order_id:123]
	// First processing completed
	// Second processing completed (idempotent)
}

// Example_transactionalProcessing 演示事务性处理
func Example_transactionalProcessing() {
	// 创建支持事务的存储
	storage := inbox.NewInMemoryStorage()
	txStorage, ok := storage.(inbox.TransactionalStorage)
	if !ok {
		log.Fatal("Storage does not support transactions")
	}

	// 创建处理器
	handler := &OrderHandler{name: "order-handler"}
	config := inbox.DefaultProcessorConfig()

	processor, err := inbox.NewProcessor(storage, handler, config)
	if err != nil {
		log.Fatalf("Failed to create processor: %v", err)
	}

	ctx := context.Background()

	// 开始事务
	tx, err := txStorage.BeginTx(ctx)
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	// 创建消息
	msg := &messaging.Message{
		ID:    "order-456",
		Topic: "orders",
		Payload: []byte(`{
			"order_id": "456",
			"customer": "Jane Smith",
			"amount": 200.0
		}`),
	}

	// 在事务中处理消息
	if err := processor.ProcessWithTransaction(ctx, tx, msg); err != nil {
		tx.Rollback(ctx)
		log.Fatalf("Failed to process message: %v", err)
	}

	// 可以在这里执行其他业务逻辑
	// tx.Exec(ctx, "INSERT INTO orders ...")

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	fmt.Println("Transactional processing completed")

	// Output:
	// Processing order: map[amount:200 customer:Jane Smith order_id:456]
	// Transactional processing completed
}

// PaymentHandler 处理支付消息并返回结果
type PaymentHandler struct {
	name string
}

func (h *PaymentHandler) Handle(ctx context.Context, msg *messaging.Message) error {
	var payment map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payment); err != nil {
		return fmt.Errorf("failed to unmarshal payment: %w", err)
	}

	fmt.Printf("Processing payment: %v\n", payment)
	return nil
}

func (h *PaymentHandler) HandleWithResult(ctx context.Context, msg *messaging.Message) (interface{}, error) {
	var payment map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment: %w", err)
	}

	fmt.Printf("Processing payment: %v\n", payment)

	// 返回处理结果
	result := map[string]interface{}{
		"status":         "success",
		"transaction_id": "tx-789",
		"amount":         payment["amount"],
	}

	return result, nil
}

func (h *PaymentHandler) GetName() string {
	return h.name
}

// Example_withResult 演示带结果的处理
func Example_withResult() {
	// 创建存储
	storage := inbox.NewInMemoryStorage()

	// 创建处理器
	handler := &PaymentHandler{name: "payment-handler"}
	config := inbox.DefaultProcessorConfig()
	config.StoreResult = true // 启用结果存储

	processor, err := inbox.NewProcessor(storage, handler, config)
	if err != nil {
		log.Fatalf("Failed to create processor: %v", err)
	}

	// 创建消息
	msg := &messaging.Message{
		ID:    "payment-789",
		Topic: "payments",
		Payload: []byte(`{
			"payment_id": "789",
			"amount": 150.0
		}`),
	}

	ctx := context.Background()

	// 第一次处理
	result1, err := processor.ProcessWithResult(ctx, msg)
	if err != nil {
		log.Fatalf("Failed to process message: %v", err)
	}
	fmt.Printf("First result: %v\n", result1)

	// 第二次处理（返回缓存的结果）
	result2, err := processor.ProcessWithResult(ctx, msg)
	if err != nil {
		log.Fatalf("Failed to process message: %v", err)
	}
	fmt.Printf("Second result: %v\n", result2)

	// Output:
	// Processing payment: map[amount:150 payment_id:789]
	// First result: map[amount:150 status:success transaction_id:tx-789]
	// Second result: map[amount:150 status:success transaction_id:tx-789]
}

// Example_multipleHandlers 演示多处理器场景
func Example_multipleHandlers() {
	// 创建共享存储
	storage := inbox.NewInMemoryStorage()

	// 创建两个不同的处理器
	orderHandler := &OrderHandler{name: "order-handler"}
	paymentHandler := &PaymentHandler{name: "payment-handler"}

	orderConfig := inbox.DefaultProcessorConfig()
	orderConfig.HandlerName = "order-handler"

	paymentConfig := inbox.DefaultProcessorConfig()
	paymentConfig.HandlerName = "payment-handler"

	orderProcessor, err := inbox.NewProcessor(storage, orderHandler, orderConfig)
	if err != nil {
		log.Fatalf("Failed to create order processor: %v", err)
	}

	paymentProcessor, err := inbox.NewProcessor(storage, paymentHandler, paymentConfig)
	if err != nil {
		log.Fatalf("Failed to create payment processor: %v", err)
	}

	// 创建同一个消息
	msg := &messaging.Message{
		ID:    "multi-msg-1",
		Topic: "transactions",
		Payload: []byte(`{
			"transaction_id": "1",
			"type": "order",
			"amount": 100.0
		}`),
	}

	ctx := context.Background()

	// 两个处理器都可以处理同一个消息
	if err := orderProcessor.Process(ctx, msg); err != nil {
		log.Fatalf("Failed to process with order handler: %v", err)
	}
	fmt.Println("Order handler processed")

	if err := paymentProcessor.Process(ctx, msg); err != nil {
		log.Fatalf("Failed to process with payment handler: %v", err)
	}
	fmt.Println("Payment handler processed")

	// Output:
	// Processing order: map[amount:100 transaction_id:1 type:order]
	// Order handler processed
	// Processing payment: map[amount:100 transaction_id:1 type:order]
	// Payment handler processed
}

// Example_autoCleanup 演示自动清理
func Example_autoCleanup() {
	// 创建存储
	storage := inbox.NewInMemoryStorage()

	// 创建处理器，启用自动清理
	handler := &OrderHandler{name: "order-handler"}
	config := inbox.DefaultProcessorConfig()
	config.EnableAutoCleanup = true
	// config.CleanupInterval = 1 * time.Hour   // 每小时清理一次
	// config.CleanupAfter = 24 * time.Hour     // 清理24小时前的记录

	processor, err := inbox.NewProcessor(storage, handler, config)
	if err != nil {
		log.Fatalf("Failed to create processor: %v", err)
	}

	ctx := context.Background()

	// 处理消息
	msg := &messaging.Message{
		ID:      "order-999",
		Topic:   "orders",
		Payload: []byte(`{"order_id": "999"}`),
	}

	if err := processor.Process(ctx, msg); err != nil {
		log.Fatalf("Failed to process message: %v", err)
	}

	fmt.Println("Processing completed with auto-cleanup enabled")

	// 后台会自动清理过期的记录
	// 应用关闭时可以停止清理
	// processor.StopCleanup(ctx)

	// Output:
	// Processing order: map[order_id:999]
	// Processing completed with auto-cleanup enabled
}

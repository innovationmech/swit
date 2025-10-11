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

// Package main demonstrates how to use the Saga Orchestrator for distributed transaction management.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/coordinator"
)

func main() {
	// 初始化 Saga Coordinator
	sagaCoordinator, err := setupCoordinator()
	if err != nil {
		log.Fatalf("初始化 Saga Coordinator 失败: %v", err)
	}
	defer sagaCoordinator.Close()

	log.Println("=== Saga Orchestrator 示例 ===")
	log.Println()

	// 示例 1: 成功的订单处理流程
	log.Println("示例 1: 成功的订单处理流程")
	runSuccessfulOrderSaga(sagaCoordinator)
	log.Println()

	// 示例 2: 失败并触发补偿的流程
	log.Println("示例 2: 支付失败触发补偿")
	runFailedOrderSaga(sagaCoordinator)
	log.Println()

	// 示例 3: 并发执行多个 Saga
	log.Println("示例 3: 并发执行多个订单")
	runConcurrentSagas(sagaCoordinator)
	log.Println()

	// 打印最终指标
	printMetrics(sagaCoordinator)
}

// setupCoordinator 初始化 Saga Coordinator
func setupCoordinator() (saga.SagaCoordinator, error) {
	// 创建内存状态存储（生产环境建议使用 Redis 或数据库）
	stateStorage := coordinator.NewInMemoryStateStorage()

	// 创建内存事件发布器（生产环境建议使用 Kafka 或 RabbitMQ）
	eventPublisher := coordinator.NewInMemoryEventPublisher()

	// 配置 Coordinator
	config := &coordinator.OrchestratorConfig{
		StateStorage:   stateStorage,
		EventPublisher: eventPublisher,
		RetryPolicy:    saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second),
		ConcurrencyConfig: &coordinator.ConcurrencyConfig{
			MaxConcurrentSagas: 10,
			WorkerPoolSize:     5,
			AcquireTimeout:     time.Second * 5,
			ShutdownTimeout:    time.Second * 30,
		},
	}

	return coordinator.NewOrchestratorCoordinator(config)
}

// runSuccessfulOrderSaga 运行成功的订单处理 Saga
func runSuccessfulOrderSaga(coordinator saga.SagaCoordinator) {
	// 创建订单处理 Saga
	definition := NewOrderProcessingSaga(false) // false = 不模拟失败

	// 准备订单数据
	orderData := &OrderData{
		OrderID:    "ORDER-001",
		CustomerID: "CUST-12345",
		Items: []OrderItem{
			{ProductID: "PROD-001", ProductName: "笔记本电脑", Quantity: 1, Price: 999.99},
			{ProductID: "PROD-002", ProductName: "无线鼠标", Quantity: 2, Price: 29.99},
		},
		TotalAmount: 1059.97,
		Currency:    "USD",
	}

	// 启动 Saga
	ctx := context.Background()
	instance, err := coordinator.StartSaga(ctx, definition, orderData)
	if err != nil {
		log.Printf("❌ 启动 Saga 失败: %v", err)
		return
	}

	log.Printf("✓ Saga 已启动: ID=%s", instance.GetID())

	// 等待 Saga 完成
	waitForCompletion(coordinator, instance.GetID())
}

// runFailedOrderSaga 运行失败的订单处理 Saga（模拟支付失败）
func runFailedOrderSaga(coordinator saga.SagaCoordinator) {
	// 创建订单处理 Saga（模拟支付失败）
	definition := NewOrderProcessingSaga(true) // true = 模拟支付失败

	// 准备订单数据
	orderData := &OrderData{
		OrderID:    "ORDER-002",
		CustomerID: "CUST-67890",
		Items: []OrderItem{
			{ProductID: "PROD-003", ProductName: "智能手表", Quantity: 1, Price: 299.99},
		},
		TotalAmount: 299.99,
		Currency:    "USD",
	}

	// 启动 Saga
	ctx := context.Background()
	instance, err := coordinator.StartSaga(ctx, definition, orderData)
	if err != nil {
		log.Printf("❌ 启动 Saga 失败: %v", err)
		return
	}

	log.Printf("✓ Saga 已启动: ID=%s", instance.GetID())

	// 等待 Saga 完成（包括补偿）
	waitForCompletion(coordinator, instance.GetID())
}

// runConcurrentSagas 并发运行多个 Saga
func runConcurrentSagas(coordinator saga.SagaCoordinator) {
	numSagas := 3
	done := make(chan bool, numSagas)

	for i := 0; i < numSagas; i++ {
		go func(index int) {
			definition := NewOrderProcessingSaga(false)
			orderData := &OrderData{
				OrderID:    fmt.Sprintf("ORDER-%03d", 100+index),
				CustomerID: fmt.Sprintf("CUST-%05d", 10000+index),
				Items: []OrderItem{
					{ProductID: "PROD-001", ProductName: "商品A", Quantity: 1, Price: 100.0},
				},
				TotalAmount: 100.0,
				Currency:    "USD",
			}

			ctx := context.Background()
			instance, err := coordinator.StartSaga(ctx, definition, orderData)
			if err != nil {
				log.Printf("❌ 启动 Saga %d 失败: %v", index, err)
				done <- false
				return
			}

			log.Printf("✓ 并发 Saga %d 已启动: ID=%s", index, instance.GetID())
			waitForCompletion(coordinator, instance.GetID())
			done <- true
		}(i)
	}

	// 等待所有 Saga 完成
	for i := 0; i < numSagas; i++ {
		<-done
	}
}

// waitForCompletion 等待 Saga 完成
func waitForCompletion(coordinator saga.SagaCoordinator, sagaID string) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-timeout:
			log.Printf("⏱ Saga 执行超时: %s", sagaID)
			return
		case <-ticker.C:
			instance, err := coordinator.GetSagaInstance(sagaID)
			if err != nil {
				log.Printf("❌ 获取 Saga 实例失败: %v", err)
				return
			}

			if instance.IsTerminal() {
				printSagaResult(instance)
				return
			}
		}
	}
}

// printSagaResult 打印 Saga 执行结果
func printSagaResult(instance saga.SagaInstance) {
	state := instance.GetState()
	duration := instance.GetEndTime().Sub(instance.GetStartTime())

	switch state {
	case saga.StateCompleted:
		log.Printf("✅ Saga 执行成功: ID=%s, Duration=%s",
			instance.GetID(), duration)
		if result := instance.GetResult(); result != nil {
			orderData := result.(*OrderData)
			log.Printf("   订单ID: %s, 状态: %s",
				orderData.OrderID, orderData.Status)
		}
	case saga.StateCompensated:
		log.Printf("↩️  Saga 已补偿: ID=%s, Duration=%s",
			instance.GetID(), duration)
		if err := instance.GetError(); err != nil {
			log.Printf("   失败原因: %s", err.Message)
		}
	case saga.StateFailed:
		log.Printf("❌ Saga 执行失败: ID=%s, Duration=%s",
			instance.GetID(), duration)
		if err := instance.GetError(); err != nil {
			log.Printf("   错误: %s", err.Message)
		}
	default:
		log.Printf("⚠️  Saga 结束于非预期状态: ID=%s, State=%s",
			instance.GetID(), state.String())
	}
}

// printMetrics 打印 Coordinator 指标
func printMetrics(coordinator saga.SagaCoordinator) {
	metrics := coordinator.GetMetrics()

	log.Println("=== Saga Coordinator 指标 ===")
	log.Printf("总 Saga 数: %d", metrics.TotalSagas)
	log.Printf("已完成: %d", metrics.CompletedSagas)
	log.Printf("失败: %d", metrics.FailedSagas)
	log.Printf("活跃: %d", metrics.ActiveSagas)
	if metrics.AverageSagaDuration > 0 {
		log.Printf("平均执行时间: %s", metrics.AverageSagaDuration)
	}
}

// ==========================
// 订单处理 Saga 定义
// ==========================

// NewOrderProcessingSaga 创建订单处理 Saga 定义
func NewOrderProcessingSaga(simulateFailure bool) saga.SagaDefinition {
	return &OrderSagaDefinition{
		id:              "order-processing-saga",
		name:            "订单处理 Saga",
		description:     "处理电商订单的完整流程",
		simulateFailure: simulateFailure,
	}
}

// OrderSagaDefinition 订单 Saga 定义
type OrderSagaDefinition struct {
	id              string
	name            string
	description     string
	simulateFailure bool
}

func (d *OrderSagaDefinition) GetID() string          { return d.id }
func (d *OrderSagaDefinition) GetName() string        { return d.name }
func (d *OrderSagaDefinition) GetDescription() string { return d.description }
func (d *OrderSagaDefinition) GetTimeout() time.Duration {
	return 5 * time.Minute
}

func (d *OrderSagaDefinition) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

func (d *OrderSagaDefinition) GetCompensationStrategy() saga.CompensationStrategy {
	return saga.NewSequentialCompensationStrategy(30 * time.Second)
}

func (d *OrderSagaDefinition) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"domain":  "ecommerce",
		"version": "v1",
	}
}

func (d *OrderSagaDefinition) GetSteps() []saga.SagaStep {
	return []saga.SagaStep{
		NewCreateOrderStep(),
		NewReserveInventoryStep(),
		NewProcessPaymentStep(d.simulateFailure),
		NewConfirmOrderStep(),
	}
}

func (d *OrderSagaDefinition) Validate() error {
	if d.id == "" {
		return errors.New("saga ID is required")
	}
	return nil
}

// ==========================
// 订单数据结构
// ==========================

// OrderData 订单数据
type OrderData struct {
	OrderID           string
	CustomerID        string
	Items             []OrderItem
	TotalAmount       float64
	Currency          string
	Status            string
	InventoryReserved bool
	PaymentProcessed  bool
	PaymentID         string
}

// OrderItem 订单项
type OrderItem struct {
	ProductID   string
	ProductName string
	Quantity    int
	Price       float64
}

// ==========================
// Saga 步骤实现
// ==========================

// CreateOrderStep 创建订单步骤
type CreateOrderStep struct{}

func NewCreateOrderStep() *CreateOrderStep {
	return &CreateOrderStep{}
}

func (s *CreateOrderStep) GetID() string                     { return "create-order" }
func (s *CreateOrderStep) GetName() string                   { return "创建订单" }
func (s *CreateOrderStep) GetDescription() string            { return "在订单服务中创建新订单" }
func (s *CreateOrderStep) GetTimeout() time.Duration         { return 5 * time.Second }
func (s *CreateOrderStep) GetRetryPolicy() saga.RetryPolicy  { return nil }
func (s *CreateOrderStep) IsRetryable(err error) bool        { return true }
func (s *CreateOrderStep) GetMetadata() map[string]interface{} { return nil }

func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	orderData := data.(*OrderData)
	log.Printf("  [1/4] 创建订单: %s", orderData.OrderID)

	// 模拟订单创建
	time.Sleep(200 * time.Millisecond)

	orderData.Status = "CREATED"
	log.Printf("  ✓ 订单已创建: %s", orderData.OrderID)
	return orderData, nil
}

func (s *CreateOrderStep) Compensate(ctx context.Context, data interface{}) error {
	orderData := data.(*OrderData)
	log.Printf("  ↩ [补偿] 取消订单: %s", orderData.OrderID)

	// 模拟订单取消
	time.Sleep(100 * time.Millisecond)

	orderData.Status = "CANCELLED"
	log.Printf("  ✓ 订单已取消: %s", orderData.OrderID)
	return nil
}

// ReserveInventoryStep 预留库存步骤
type ReserveInventoryStep struct{}

func NewReserveInventoryStep() *ReserveInventoryStep {
	return &ReserveInventoryStep{}
}

func (s *ReserveInventoryStep) GetID() string                     { return "reserve-inventory" }
func (s *ReserveInventoryStep) GetName() string                   { return "预留库存" }
func (s *ReserveInventoryStep) GetDescription() string            { return "在库存服务中预留商品库存" }
func (s *ReserveInventoryStep) GetTimeout() time.Duration         { return 5 * time.Second }
func (s *ReserveInventoryStep) GetRetryPolicy() saga.RetryPolicy  { return nil }
func (s *ReserveInventoryStep) IsRetryable(err error) bool        { return true }
func (s *ReserveInventoryStep) GetMetadata() map[string]interface{} { return nil }

func (s *ReserveInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	orderData := data.(*OrderData)
	log.Printf("  [2/4] 预留库存: %d 个商品", len(orderData.Items))

	// 模拟库存预留
	time.Sleep(300 * time.Millisecond)

	orderData.InventoryReserved = true
	log.Printf("  ✓ 库存已预留")
	return orderData, nil
}

func (s *ReserveInventoryStep) Compensate(ctx context.Context, data interface{}) error {
	orderData := data.(*OrderData)
	log.Printf("  ↩ [补偿] 释放库存: %d 个商品", len(orderData.Items))

	// 模拟库存释放
	time.Sleep(100 * time.Millisecond)

	orderData.InventoryReserved = false
	log.Printf("  ✓ 库存已释放")
	return nil
}

// ProcessPaymentStep 处理支付步骤
type ProcessPaymentStep struct {
	simulateFailure bool
}

func NewProcessPaymentStep(simulateFailure bool) *ProcessPaymentStep {
	return &ProcessPaymentStep{simulateFailure: simulateFailure}
}

func (s *ProcessPaymentStep) GetID() string                     { return "process-payment" }
func (s *ProcessPaymentStep) GetName() string                   { return "处理支付" }
func (s *ProcessPaymentStep) GetDescription() string            { return "在支付服务中处理支付" }
func (s *ProcessPaymentStep) GetTimeout() time.Duration         { return 10 * time.Second }
func (s *ProcessPaymentStep) GetRetryPolicy() saga.RetryPolicy  { return nil }
func (s *ProcessPaymentStep) IsRetryable(err error) bool        { return false }
func (s *ProcessPaymentStep) GetMetadata() map[string]interface{} { return nil }

func (s *ProcessPaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	orderData := data.(*OrderData)
	log.Printf("  [3/4] 处理支付: %.2f %s", orderData.TotalAmount, orderData.Currency)

	// 模拟支付处理
	time.Sleep(400 * time.Millisecond)

	if s.simulateFailure {
		log.Printf("  ✗ 支付失败: 信用卡被拒绝")
		return nil, errors.New("payment declined: insufficient funds")
	}

	orderData.PaymentProcessed = true
	orderData.PaymentID = fmt.Sprintf("PAY-%d", time.Now().Unix())
	log.Printf("  ✓ 支付成功: %s", orderData.PaymentID)
	return orderData, nil
}

func (s *ProcessPaymentStep) Compensate(ctx context.Context, data interface{}) error {
	orderData := data.(*OrderData)
	log.Printf("  ↩ [补偿] 退款: %.2f %s", orderData.TotalAmount, orderData.Currency)

	// 模拟退款
	time.Sleep(200 * time.Millisecond)

	orderData.PaymentProcessed = false
	orderData.PaymentID = ""
	log.Printf("  ✓ 退款完成")
	return nil
}

// ConfirmOrderStep 确认订单步骤
type ConfirmOrderStep struct{}

func NewConfirmOrderStep() *ConfirmOrderStep {
	return &ConfirmOrderStep{}
}

func (s *ConfirmOrderStep) GetID() string                     { return "confirm-order" }
func (s *ConfirmOrderStep) GetName() string                   { return "确认订单" }
func (s *ConfirmOrderStep) GetDescription() string            { return "确认订单并准备发货" }
func (s *ConfirmOrderStep) GetTimeout() time.Duration         { return 5 * time.Second }
func (s *ConfirmOrderStep) GetRetryPolicy() saga.RetryPolicy  { return nil }
func (s *ConfirmOrderStep) IsRetryable(err error) bool        { return true }
func (s *ConfirmOrderStep) GetMetadata() map[string]interface{} { return nil }

func (s *ConfirmOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	orderData := data.(*OrderData)
	log.Printf("  [4/4] 确认订单: %s", orderData.OrderID)

	// 模拟订单确认
	time.Sleep(200 * time.Millisecond)

	orderData.Status = "CONFIRMED"
	log.Printf("  ✓ 订单已确认，准备发货")
	return orderData, nil
}

func (s *ConfirmOrderStep) Compensate(ctx context.Context, data interface{}) error {
	orderData := data.(*OrderData)
	log.Printf("  ↩ [补偿] 取消确认: %s", orderData.OrderID)

	// 模拟取消确认
	time.Sleep(100 * time.Millisecond)

	orderData.Status = "CANCELLED"
	log.Printf("  ✓ 订单确认已取消")
	return nil
}


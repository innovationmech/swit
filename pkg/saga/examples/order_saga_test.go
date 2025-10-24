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

package examples

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// ==========================
// Mock 服务实现
// ==========================

// mockOrderService 模拟订单服务
type mockOrderService struct {
	createOrderFunc  func(ctx context.Context, data *OrderData) (*OrderStepResult, error)
	cancelOrderFunc  func(ctx context.Context, orderID string, reason string) error
	confirmOrderFunc func(ctx context.Context, orderID string) (*ConfirmStepResult, error)
}

func (m *mockOrderService) CreateOrder(ctx context.Context, data *OrderData) (*OrderStepResult, error) {
	if m.createOrderFunc != nil {
		return m.createOrderFunc(ctx, data)
	}
	return &OrderStepResult{
		OrderID:     "order-123",
		OrderNumber: "ON-2025-001",
		Status:      "pending",
		CreatedAt:   time.Now(),
		Items:       data.Items,
		Metadata:    map[string]interface{}{},
	}, nil
}

func (m *mockOrderService) CancelOrder(ctx context.Context, orderID string, reason string) error {
	if m.cancelOrderFunc != nil {
		return m.cancelOrderFunc(ctx, orderID, reason)
	}
	return nil
}

func (m *mockOrderService) ConfirmOrder(ctx context.Context, orderID string) (*ConfirmStepResult, error) {
	if m.confirmOrderFunc != nil {
		return m.confirmOrderFunc(ctx, orderID)
	}
	return &ConfirmStepResult{
		OrderID:       orderID,
		ConfirmedAt:   time.Now(),
		Status:        "confirmed",
		EstimatedShip: time.Now().Add(24 * time.Hour),
		TrackingInfo:  "TRACK-001",
		Metadata:      map[string]interface{}{"order_id": orderID},
	}, nil
}

// mockInventoryService 模拟库存服务
type mockInventoryService struct {
	reserveInventoryFunc func(ctx context.Context, orderID string, items []OrderItem) (*InventoryStepResult, error)
	releaseInventoryFunc func(ctx context.Context, reservationID string) error
}

func (m *mockInventoryService) ReserveInventory(ctx context.Context, orderID string, items []OrderItem) (*InventoryStepResult, error) {
	if m.reserveInventoryFunc != nil {
		return m.reserveInventoryFunc(ctx, orderID, items)
	}
	reservedItems := make([]ReservedItem, len(items))
	for i, item := range items {
		reservedItems[i] = ReservedItem{
			ProductID:   item.ProductID,
			SKU:         item.SKU,
			Quantity:    item.Quantity,
			WarehouseID: "WH-001",
		}
	}
	return &InventoryStepResult{
		ReservationID: "RES-123",
		ReservedItems: reservedItems,
		ExpiresAt:     time.Now().Add(15 * time.Minute),
		WarehouseID:   "WH-001",
		Metadata: map[string]interface{}{
			"order_id":       orderID,
			"amount":         100.0,
			"payment_method": "credit_card",
			"payment_token":  "token-123",
		},
	}, nil
}

func (m *mockInventoryService) ReleaseInventory(ctx context.Context, reservationID string) error {
	if m.releaseInventoryFunc != nil {
		return m.releaseInventoryFunc(ctx, reservationID)
	}
	return nil
}

// mockPaymentService 模拟支付服务
type mockPaymentService struct {
	processPaymentFunc func(ctx context.Context, orderID string, amount float64, method string, token string) (*PaymentStepResult, error)
	refundPaymentFunc  func(ctx context.Context, transactionID string, reason string) error
}

func (m *mockPaymentService) ProcessPayment(ctx context.Context, orderID string, amount float64, method string, token string) (*PaymentStepResult, error) {
	if m.processPaymentFunc != nil {
		return m.processPaymentFunc(ctx, orderID, amount, method, token)
	}
	return &PaymentStepResult{
		TransactionID: "TXN-123",
		PaymentID:     "PAY-123",
		Amount:        amount,
		Currency:      "CNY",
		Status:        "success",
		PaidAt:        time.Now(),
		Provider:      "mock-payment",
		Metadata:      map[string]interface{}{"order_id": orderID},
	}, nil
}

func (m *mockPaymentService) RefundPayment(ctx context.Context, transactionID string, reason string) error {
	if m.refundPaymentFunc != nil {
		return m.refundPaymentFunc(ctx, transactionID, reason)
	}
	return nil
}

// ==========================
// 测试辅助函数
// ==========================

// createTestOrderData 创建测试用订单数据
func createTestOrderData() *OrderData {
	return &OrderData{
		CustomerID:  "CUST-001",
		TotalAmount: 299.99,
		Items: []OrderItem{
			{
				ProductID: "PROD-001",
				SKU:       "SKU-001",
				Quantity:  2,
				UnitPrice: 99.99,
				Total:     199.98,
			},
			{
				ProductID: "PROD-002",
				SKU:       "SKU-002",
				Quantity:  1,
				UnitPrice: 100.01,
				Total:     100.01,
			},
		},
		Address: ShippingAddress{
			RecipientName: "张三",
			Phone:         "13800138000",
			Province:      "北京市",
			City:          "北京市",
			District:      "朝阳区",
			Street:        "某某街道123号",
			Zipcode:       "100000",
		},
		PaymentMethod: "credit_card",
		PaymentToken:  "token-test-123",
		Metadata:      map[string]interface{}{},
	}
}

// ==========================
// CreateOrderStep 测试
// ==========================

func TestCreateOrderStep_Execute_Success(t *testing.T) {
	mockService := &mockOrderService{}
	step := &CreateOrderStep{service: mockService}
	ctx := context.Background()

	orderData := createTestOrderData()
	result, err := step.Execute(ctx, orderData)

	if err != nil {
		t.Fatalf("Execute() 失败: %v", err)
	}

	orderResult, ok := result.(*OrderStepResult)
	if !ok {
		t.Fatalf("结果类型错误，期望 *OrderStepResult")
	}

	if orderResult.OrderID == "" {
		t.Error("OrderID 不应为空")
	}
	if orderResult.Status != "pending" {
		t.Errorf("Status = %s, 期望 pending", orderResult.Status)
	}
}

func TestCreateOrderStep_Execute_ValidationError(t *testing.T) {
	mockService := &mockOrderService{}
	step := &CreateOrderStep{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name      string
		orderData *OrderData
		wantErr   bool
	}{
		{
			name: "空客户ID",
			orderData: &OrderData{
				CustomerID:  "",
				TotalAmount: 100,
				Items:       []OrderItem{{ProductID: "P1"}},
				Address:     ShippingAddress{RecipientName: "张三", Phone: "13800138000"},
			},
			wantErr: true,
		},
		{
			name: "空订单项",
			orderData: &OrderData{
				CustomerID:  "CUST-001",
				TotalAmount: 100,
				Items:       []OrderItem{},
				Address:     ShippingAddress{RecipientName: "张三", Phone: "13800138000"},
			},
			wantErr: true,
		},
		{
			name: "总金额为0",
			orderData: &OrderData{
				CustomerID:  "CUST-001",
				TotalAmount: 0,
				Items:       []OrderItem{{ProductID: "P1"}},
				Address:     ShippingAddress{RecipientName: "张三", Phone: "13800138000"},
			},
			wantErr: true,
		},
		{
			name: "收件人姓名为空",
			orderData: &OrderData{
				CustomerID:  "CUST-001",
				TotalAmount: 100,
				Items:       []OrderItem{{ProductID: "P1"}},
				Address:     ShippingAddress{RecipientName: "", Phone: "13800138000"},
			},
			wantErr: true,
		},
		{
			name: "联系电话为空",
			orderData: &OrderData{
				CustomerID:  "CUST-001",
				TotalAmount: 100,
				Items:       []OrderItem{{ProductID: "P1"}},
				Address:     ShippingAddress{RecipientName: "张三", Phone: ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := step.Execute(ctx, tt.orderData)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateOrderStep_Execute_ServiceError(t *testing.T) {
	mockService := &mockOrderService{
		createOrderFunc: func(ctx context.Context, data *OrderData) (*OrderStepResult, error) {
			return nil, errors.New("服务不可用")
		},
	}
	step := &CreateOrderStep{service: mockService}
	ctx := context.Background()

	orderData := createTestOrderData()
	_, err := step.Execute(ctx, orderData)

	if err == nil {
		t.Error("期望出现错误，但没有错误")
	}
}

func TestCreateOrderStep_Compensate_Success(t *testing.T) {
	mockService := &mockOrderService{}
	step := &CreateOrderStep{service: mockService}
	ctx := context.Background()

	result := &OrderStepResult{
		OrderID: "order-123",
	}

	err := step.Compensate(ctx, result)
	if err != nil {
		t.Errorf("Compensate() 失败: %v", err)
	}
}

func TestCreateOrderStep_Compensate_ServiceError(t *testing.T) {
	mockService := &mockOrderService{
		cancelOrderFunc: func(ctx context.Context, orderID string, reason string) error {
			return errors.New("取消订单失败")
		},
	}
	step := &CreateOrderStep{service: mockService}
	ctx := context.Background()

	result := &OrderStepResult{
		OrderID: "order-123",
	}

	err := step.Compensate(ctx, result)
	if err == nil {
		t.Error("期望出现错误，但没有错误")
	}
}

func TestCreateOrderStep_Metadata(t *testing.T) {
	step := &CreateOrderStep{}

	if step.GetID() != "create-order" {
		t.Errorf("GetID() = %s, 期望 create-order", step.GetID())
	}
	if step.GetName() != "创建订单" {
		t.Errorf("GetName() = %s, 期望 创建订单", step.GetName())
	}
	if step.GetTimeout() <= 0 {
		t.Error("GetTimeout() 应返回正数")
	}
	if step.GetRetryPolicy() == nil {
		t.Error("GetRetryPolicy() 不应为 nil")
	}
	metadata := step.GetMetadata()
	if metadata["step_type"] != "order_creation" {
		t.Errorf("Metadata step_type = %v, 期望 order_creation", metadata["step_type"])
	}
}

// ==========================
// ReserveInventoryStep 测试
// ==========================

func TestReserveInventoryStep_Execute_Success(t *testing.T) {
	mockService := &mockInventoryService{}
	step := &ReserveInventoryStep{service: mockService}
	ctx := context.Background()

	orderResult := &OrderStepResult{
		OrderID: "order-123",
		Items: []OrderItem{
			{ProductID: "P1", SKU: "SKU1", Quantity: 2},
		},
	}

	result, err := step.Execute(ctx, orderResult)
	if err != nil {
		t.Fatalf("Execute() 失败: %v", err)
	}

	invResult, ok := result.(*InventoryStepResult)
	if !ok {
		t.Fatalf("结果类型错误，期望 *InventoryStepResult")
	}

	if invResult.ReservationID == "" {
		t.Error("ReservationID 不应为空")
	}
	if len(invResult.ReservedItems) == 0 {
		t.Error("ReservedItems 不应为空")
	}
}

func TestReserveInventoryStep_Execute_InvalidDataType(t *testing.T) {
	mockService := &mockInventoryService{}
	step := &ReserveInventoryStep{service: mockService}
	ctx := context.Background()

	// 传递错误的数据类型
	_, err := step.Execute(ctx, "invalid-data")
	if err == nil {
		t.Error("期望出现错误，但没有错误")
	}
}

func TestReserveInventoryStep_Execute_ServiceError(t *testing.T) {
	mockService := &mockInventoryService{
		reserveInventoryFunc: func(ctx context.Context, orderID string, items []OrderItem) (*InventoryStepResult, error) {
			return nil, errors.New("库存不足")
		},
	}
	step := &ReserveInventoryStep{service: mockService}
	ctx := context.Background()

	orderResult := &OrderStepResult{
		OrderID: "order-123",
		Items:   []OrderItem{{ProductID: "P1"}},
	}

	_, err := step.Execute(ctx, orderResult)
	if err == nil {
		t.Error("期望出现错误，但没有错误")
	}
}

func TestReserveInventoryStep_Compensate_Success(t *testing.T) {
	mockService := &mockInventoryService{}
	step := &ReserveInventoryStep{service: mockService}
	ctx := context.Background()

	result := &InventoryStepResult{
		ReservationID: "RES-123",
	}

	err := step.Compensate(ctx, result)
	if err != nil {
		t.Errorf("Compensate() 失败: %v", err)
	}
}

func TestReserveInventoryStep_Metadata(t *testing.T) {
	step := &ReserveInventoryStep{}

	if step.GetID() != "reserve-inventory" {
		t.Errorf("GetID() = %s, 期望 reserve-inventory", step.GetID())
	}
	if step.GetName() != "预留库存" {
		t.Errorf("GetName() = %s, 期望 预留库存", step.GetName())
	}
}

// ==========================
// ProcessPaymentStep 测试
// ==========================

func TestProcessPaymentStep_Execute_Success(t *testing.T) {
	mockService := &mockPaymentService{}
	step := &ProcessPaymentStep{service: mockService}
	ctx := context.Background()

	invResult := &InventoryStepResult{
		ReservationID: "RES-123",
		Metadata: map[string]interface{}{
			"order_id":       "order-123",
			"amount":         299.99,
			"payment_method": "credit_card",
			"payment_token":  "token-123",
		},
	}

	result, err := step.Execute(ctx, invResult)
	if err != nil {
		t.Fatalf("Execute() 失败: %v", err)
	}

	payResult, ok := result.(*PaymentStepResult)
	if !ok {
		t.Fatalf("结果类型错误，期望 *PaymentStepResult")
	}

	if payResult.TransactionID == "" {
		t.Error("TransactionID 不应为空")
	}
	if payResult.Status != "success" {
		t.Errorf("Status = %s, 期望 success", payResult.Status)
	}
}

func TestProcessPaymentStep_Execute_MissingOrderID(t *testing.T) {
	mockService := &mockPaymentService{}
	step := &ProcessPaymentStep{service: mockService}
	ctx := context.Background()

	invResult := &InventoryStepResult{
		Metadata: map[string]interface{}{
			"amount": 100.0,
		},
	}

	_, err := step.Execute(ctx, invResult)
	if err == nil {
		t.Error("期望出现错误（缺少 order_id），但没有错误")
	}
}

func TestProcessPaymentStep_Execute_InvalidAmount(t *testing.T) {
	mockService := &mockPaymentService{}
	step := &ProcessPaymentStep{service: mockService}
	ctx := context.Background()

	invResult := &InventoryStepResult{
		Metadata: map[string]interface{}{
			"order_id": "order-123",
			"amount":   0.0,
		},
	}

	_, err := step.Execute(ctx, invResult)
	if err == nil {
		t.Error("期望出现错误（无效金额），但没有错误")
	}
}

func TestProcessPaymentStep_Compensate_Success(t *testing.T) {
	mockService := &mockPaymentService{}
	step := &ProcessPaymentStep{service: mockService}
	ctx := context.Background()

	result := &PaymentStepResult{
		TransactionID: "TXN-123",
	}

	err := step.Compensate(ctx, result)
	if err != nil {
		t.Errorf("Compensate() 失败: %v", err)
	}
}

func TestProcessPaymentStep_Compensate_RefundError(t *testing.T) {
	mockService := &mockPaymentService{
		refundPaymentFunc: func(ctx context.Context, transactionID string, reason string) error {
			return errors.New("退款失败")
		},
	}
	step := &ProcessPaymentStep{service: mockService}
	ctx := context.Background()

	result := &PaymentStepResult{
		TransactionID: "TXN-123",
	}

	err := step.Compensate(ctx, result)
	if err == nil {
		t.Error("期望出现错误，但没有错误")
	}
}

func TestProcessPaymentStep_Metadata(t *testing.T) {
	step := &ProcessPaymentStep{}

	if step.GetID() != "process-payment" {
		t.Errorf("GetID() = %s, 期望 process-payment", step.GetID())
	}
	metadata := step.GetMetadata()
	if metadata["critical"] != true {
		t.Error("payment step 应标记为 critical")
	}
}

// ==========================
// ConfirmOrderStep 测试
// ==========================

func TestConfirmOrderStep_Execute_Success(t *testing.T) {
	mockService := &mockOrderService{}
	step := &ConfirmOrderStep{service: mockService}
	ctx := context.Background()

	payResult := &PaymentStepResult{
		TransactionID: "TXN-123",
		Metadata: map[string]interface{}{
			"order_id": "order-123",
		},
	}

	result, err := step.Execute(ctx, payResult)
	if err != nil {
		t.Fatalf("Execute() 失败: %v", err)
	}

	confirmResult, ok := result.(*ConfirmStepResult)
	if !ok {
		t.Fatalf("结果类型错误，期望 *ConfirmStepResult")
	}

	if confirmResult.OrderID == "" {
		t.Error("OrderID 不应为空")
	}
	if confirmResult.Status != "confirmed" {
		t.Errorf("Status = %s, 期望 confirmed", confirmResult.Status)
	}
}

func TestConfirmOrderStep_Execute_MissingOrderID(t *testing.T) {
	mockService := &mockOrderService{}
	step := &ConfirmOrderStep{service: mockService}
	ctx := context.Background()

	payResult := &PaymentStepResult{
		TransactionID: "TXN-123",
		Metadata:      map[string]interface{}{},
	}

	_, err := step.Execute(ctx, payResult)
	if err == nil {
		t.Error("期望出现错误（缺少 order_id），但没有错误")
	}
}

func TestConfirmOrderStep_Compensate(t *testing.T) {
	mockService := &mockOrderService{}
	step := &ConfirmOrderStep{service: mockService}
	ctx := context.Background()

	result := &ConfirmStepResult{
		OrderID: "order-123",
	}

	// ConfirmOrderStep 的补偿通常是空操作
	err := step.Compensate(ctx, result)
	if err != nil {
		t.Errorf("Compensate() 失败: %v", err)
	}
}

func TestConfirmOrderStep_Metadata(t *testing.T) {
	step := &ConfirmOrderStep{}

	if step.GetID() != "confirm-order" {
		t.Errorf("GetID() = %s, 期望 confirm-order", step.GetID())
	}
	if step.GetName() != "确认订单" {
		t.Errorf("GetName() = %s, 期望 确认订单", step.GetName())
	}
}

// ==========================
// OrderProcessingSagaDefinition 测试
// ==========================

func TestNewOrderProcessingSaga(t *testing.T) {
	orderService := &mockOrderService{}
	inventoryService := &mockInventoryService{}
	paymentService := &mockPaymentService{}

	saga := NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	if saga == nil {
		t.Fatal("NewOrderProcessingSaga() 返回 nil")
	}

	if saga.GetID() != "order-processing-saga" {
		t.Errorf("GetID() = %s, 期望 order-processing-saga", saga.GetID())
	}

	if saga.GetName() != "订单处理Saga" {
		t.Errorf("GetName() = %s, 期望 订单处理Saga", saga.GetName())
	}

	steps := saga.GetSteps()
	if len(steps) != 4 {
		t.Errorf("步骤数量 = %d, 期望 4", len(steps))
	}

	// 验证步骤顺序
	expectedStepIDs := []string{
		"create-order",
		"reserve-inventory",
		"process-payment",
		"confirm-order",
	}

	for i, step := range steps {
		if step.GetID() != expectedStepIDs[i] {
			t.Errorf("步骤 %d ID = %s, 期望 %s", i, step.GetID(), expectedStepIDs[i])
		}
	}

	if saga.GetTimeout() <= 0 {
		t.Error("GetTimeout() 应返回正数")
	}

	if saga.GetRetryPolicy() == nil {
		t.Error("GetRetryPolicy() 不应为 nil")
	}

	if saga.GetCompensationStrategy() == nil {
		t.Error("GetCompensationStrategy() 不应为 nil")
	}

	metadata := saga.GetMetadata()
	if metadata["saga_type"] != "order_processing" {
		t.Errorf("Metadata saga_type = %v, 期望 order_processing", metadata["saga_type"])
	}
}

func TestOrderProcessingSagaDefinition_Validate_Success(t *testing.T) {
	orderService := &mockOrderService{}
	inventoryService := &mockInventoryService{}
	paymentService := &mockPaymentService{}

	saga := NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	err := saga.Validate()
	if err != nil {
		t.Errorf("Validate() 失败: %v", err)
	}
}

func TestOrderProcessingSagaDefinition_Validate_EmptyID(t *testing.T) {
	saga := &OrderProcessingSagaDefinition{
		id:      "",
		name:    "Test Saga",
		steps:   []saga.SagaStep{},
		timeout: 1 * time.Minute,
	}

	err := saga.Validate()
	if err == nil {
		t.Error("期望出现验证错误（空ID），但没有错误")
	}
}

func TestOrderProcessingSagaDefinition_Validate_EmptyName(t *testing.T) {
	saga := &OrderProcessingSagaDefinition{
		id:      "test-saga",
		name:    "",
		steps:   []saga.SagaStep{},
		timeout: 1 * time.Minute,
	}

	err := saga.Validate()
	if err == nil {
		t.Error("期望出现验证错误（空名称），但没有错误")
	}
}

func TestOrderProcessingSagaDefinition_Validate_NoSteps(t *testing.T) {
	saga := &OrderProcessingSagaDefinition{
		id:      "test-saga",
		name:    "Test Saga",
		steps:   []saga.SagaStep{},
		timeout: 1 * time.Minute,
	}

	err := saga.Validate()
	if err == nil {
		t.Error("期望出现验证错误（无步骤），但没有错误")
	}
}

func TestOrderProcessingSagaDefinition_Validate_ZeroTimeout(t *testing.T) {
	saga := &OrderProcessingSagaDefinition{
		id:      "test-saga",
		name:    "Test Saga",
		steps:   []saga.SagaStep{&CreateOrderStep{}},
		timeout: 0,
	}

	err := saga.Validate()
	if err == nil {
		t.Error("期望出现验证错误（超时为0），但没有错误")
	}
}

// ==========================
// 集成测试
// ==========================

func TestOrderProcessingSaga_Integration_HappyPath(t *testing.T) {
	// 创建所有模拟服务
	orderService := &mockOrderService{}
	inventoryService := &mockInventoryService{}
	paymentService := &mockPaymentService{}

	// 创建 Saga 定义
	sagaDef := NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	// 验证 Saga 定义
	if err := sagaDef.Validate(); err != nil {
		t.Fatalf("Saga 验证失败: %v", err)
	}

	// 创建订单数据
	orderData := createTestOrderData()

	// 模拟执行每个步骤
	ctx := context.Background()
	steps := sagaDef.GetSteps()

	var stepData interface{} = orderData

	// 执行所有步骤
	for i, step := range steps {
		t.Logf("执行步骤 %d: %s", i+1, step.GetName())

		result, err := step.Execute(ctx, stepData)
		if err != nil {
			t.Fatalf("步骤 %s 执行失败: %v", step.GetName(), err)
		}

		stepData = result
	}

	t.Log("所有步骤执行成功")
}

func TestOrderProcessingSaga_Integration_CompensationPath(t *testing.T) {
	// 模拟支付失败的场景
	orderService := &mockOrderService{}
	inventoryService := &mockInventoryService{}
	paymentService := &mockPaymentService{
		processPaymentFunc: func(ctx context.Context, orderID string, amount float64, method string, token string) (*PaymentStepResult, error) {
			return nil, errors.New("支付失败：余额不足")
		},
	}

	sagaDef := NewOrderProcessingSaga(orderService, inventoryService, paymentService)
	ctx := context.Background()
	steps := sagaDef.GetSteps()

	// 执行步骤直到支付步骤
	var stepData interface{} = createTestOrderData()
	var completedSteps []interface{}

	for i, step := range steps {
		result, err := step.Execute(ctx, stepData)
		if err != nil {
			t.Logf("步骤 %s 失败: %v，开始补偿", step.GetName(), err)

			// 执行补偿
			strategy := sagaDef.GetCompensationStrategy()
			stepsToCompensate := strategy.GetCompensationOrder(steps[:i])

			for j, compStep := range stepsToCompensate {
				t.Logf("补偿步骤 %d: %s", j+1, compStep.GetName())
				compErr := compStep.Compensate(ctx, completedSteps[len(completedSteps)-1-j])
				if compErr != nil {
					t.Logf("补偿步骤 %s 失败: %v", compStep.GetName(), compErr)
				}
			}

			return
		}

		completedSteps = append(completedSteps, result)
		stepData = result
	}

	t.Error("期望支付步骤失败，但所有步骤都成功了")
}

// ==========================
// 边界条件和错误场景测试
// ==========================

func TestCreateOrderStep_IsRetryable(t *testing.T) {
	step := &CreateOrderStep{}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "超时错误不可重试",
			err:  context.DeadlineExceeded,
			want: false,
		},
		{
			name: "一般错误可重试",
			err:  errors.New("temporary error"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := step.IsRetryable(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessPaymentStep_IsRetryable(t *testing.T) {
	step := &ProcessPaymentStep{}

	// 支付步骤默认不重试以避免重复扣款
	err := errors.New("payment error")
	if step.IsRetryable(err) {
		t.Error("支付步骤不应该重试大多数错误")
	}
}

// ==========================
// 额外的覆盖率测试
// ==========================

func TestAllSteps_GetDescription(t *testing.T) {
	steps := []struct {
		step saga.SagaStep
		name string
	}{
		{&CreateOrderStep{}, "CreateOrderStep"},
		{&ReserveInventoryStep{}, "ReserveInventoryStep"},
		{&ProcessPaymentStep{}, "ProcessPaymentStep"},
		{&ConfirmOrderStep{}, "ConfirmOrderStep"},
	}

	for _, tt := range steps {
		t.Run(tt.name, func(t *testing.T) {
			desc := tt.step.GetDescription()
			if desc == "" {
				t.Errorf("%s.GetDescription() 返回空字符串", tt.name)
			}
		})
	}
}

func TestReserveInventoryStep_Compensate_InvalidDataType(t *testing.T) {
	mockService := &mockInventoryService{}
	step := &ReserveInventoryStep{service: mockService}
	ctx := context.Background()

	// 传递错误的数据类型
	err := step.Compensate(ctx, "invalid-data")
	if err == nil {
		t.Error("期望出现错误（错误的数据类型），但没有错误")
	}
}

func TestReserveInventoryStep_Compensate_ServiceError(t *testing.T) {
	mockService := &mockInventoryService{
		releaseInventoryFunc: func(ctx context.Context, reservationID string) error {
			return errors.New("释放库存失败")
		},
	}
	step := &ReserveInventoryStep{service: mockService}
	ctx := context.Background()

	result := &InventoryStepResult{
		ReservationID: "RES-123",
	}

	err := step.Compensate(ctx, result)
	if err == nil {
		t.Error("期望出现错误，但没有错误")
	}
}

func TestProcessPaymentStep_Compensate_InvalidDataType(t *testing.T) {
	mockService := &mockPaymentService{}
	step := &ProcessPaymentStep{service: mockService}
	ctx := context.Background()

	// 传递错误的数据类型
	err := step.Compensate(ctx, "invalid-data")
	if err == nil {
		t.Error("期望出现错误（错误的数据类型），但没有错误")
	}
}

func TestCreateOrderStep_Compensate_InvalidDataType(t *testing.T) {
	mockService := &mockOrderService{}
	step := &CreateOrderStep{service: mockService}
	ctx := context.Background()

	// 传递错误的数据类型
	err := step.Compensate(ctx, "invalid-data")
	if err == nil {
		t.Error("期望出现错误（错误的数据类型），但没有错误")
	}
}

func TestCreateOrderStep_Execute_InvalidDataType(t *testing.T) {
	mockService := &mockOrderService{}
	step := &CreateOrderStep{service: mockService}
	ctx := context.Background()

	// 传递错误的数据类型
	_, err := step.Execute(ctx, "invalid-data")
	if err == nil {
		t.Error("期望出现错误（错误的数据类型），但没有错误")
	}
}

func TestProcessPaymentStep_Execute_InvalidDataType(t *testing.T) {
	mockService := &mockPaymentService{}
	step := &ProcessPaymentStep{service: mockService}
	ctx := context.Background()

	// 传递错误的数据类型
	_, err := step.Execute(ctx, "invalid-data")
	if err == nil {
		t.Error("期望出现错误（错误的数据类型），但没有错误")
	}
}

func TestConfirmOrderStep_Execute_InvalidDataType(t *testing.T) {
	mockService := &mockOrderService{}
	step := &ConfirmOrderStep{service: mockService}
	ctx := context.Background()

	// 传递错误的数据类型
	_, err := step.Execute(ctx, "invalid-data")
	if err == nil {
		t.Error("期望出现错误（错误的数据类型），但没有错误")
	}
}

func TestConfirmOrderStep_Compensate_InvalidDataType(t *testing.T) {
	mockService := &mockOrderService{}
	step := &ConfirmOrderStep{service: mockService}
	ctx := context.Background()

	// 传递错误的数据类型，但由于 ConfirmOrderStep 的补偿是空操作，不会产生错误
	err := step.Compensate(ctx, "invalid-data")
	if err != nil {
		t.Errorf("ConfirmOrderStep.Compensate() 不应该出错: %v", err)
	}
}

func TestConfirmOrderStep_Execute_ServiceError(t *testing.T) {
	mockService := &mockOrderService{
		confirmOrderFunc: func(ctx context.Context, orderID string) (*ConfirmStepResult, error) {
			return nil, errors.New("确认订单失败")
		},
	}
	step := &ConfirmOrderStep{service: mockService}
	ctx := context.Background()

	payResult := &PaymentStepResult{
		TransactionID: "TXN-123",
		Metadata: map[string]interface{}{
			"order_id": "order-123",
		},
	}

	_, err := step.Execute(ctx, payResult)
	if err == nil {
		t.Error("期望出现错误，但没有错误")
	}
}

func TestReserveInventoryStep_IsRetryable(t *testing.T) {
	step := &ReserveInventoryStep{}

	// 测试超时错误
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("超时错误不应该重试")
	}

	// 测试一般错误
	if !step.IsRetryable(errors.New("network error")) {
		t.Error("网络错误应该重试")
	}
}

func TestConfirmOrderStep_IsRetryable(t *testing.T) {
	step := &ConfirmOrderStep{}

	// 测试超时错误
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("超时错误不应该重试")
	}

	// 测试一般错误
	if !step.IsRetryable(errors.New("network error")) {
		t.Error("网络错误应该重试")
	}
}

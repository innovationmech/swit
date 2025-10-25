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

// Package integration 提供 Saga 系统的完整集成测试套件
package integration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
	"github.com/innovationmech/swit/pkg/saga/examples"
)

// ==========================
// 测试基础设施和辅助函数
// ==========================

// executeSaga 手动执行 Saga 的所有步骤
func executeSaga(ctx context.Context, sagaDef saga.SagaDefinition, initialData interface{}) (interface{}, error) {
	steps := sagaDef.GetSteps()
	var currentData interface{} = initialData
	var executedResults []interface{}

	// 执行所有步骤
	for i, step := range steps {
		result, err := step.Execute(ctx, currentData)
		if err != nil {
			// 执行补偿
			for j := i - 1; j >= 0; j-- {
				compStep := steps[j]
				_ = compStep.Compensate(ctx, executedResults[j])
			}
			return nil, fmt.Errorf("步骤 %s 执行失败: %w", step.GetName(), err)
		}
		executedResults = append(executedResults, result)
		currentData = result
	}

	return currentData, nil
}

// ==========================
// 1. 订单处理完整流程集成测试
// ==========================

// TestOrderProcessing_Integration_HappyPath 测试订单处理正常流程
func TestOrderProcessing_Integration_HappyPath(t *testing.T) {
	orderService := &mockOrderService{}
	inventoryService := &mockInventoryService{}
	paymentService := &mockPaymentService{}

	sagaDef := examples.NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	orderData := &examples.OrderData{
		CustomerID:  "CUST-INT-001",
		TotalAmount: 599.99,
		Items: []examples.OrderItem{
			{ProductID: "PROD-001", SKU: "SKU-001", Quantity: 2, UnitPrice: 299.99, Total: 599.98},
		},
		Address: examples.ShippingAddress{
			RecipientName: "集成测试用户",
			Phone:         "13900139000",
			Province:      "广东省",
			City:          "深圳市",
			District:      "南山区",
			Street:        "科技园路100号",
			Zipcode:       "518000",
		},
		PaymentMethod: "alipay",
		PaymentToken:  "token-integration-test",
		Metadata:      map[string]interface{}{"test_type": "integration"},
	}

	ctx := context.Background()
	result, err := executeSaga(ctx, sagaDef, orderData)
	if err != nil {
		t.Fatalf("Saga 执行失败: %v", err)
	}

	confirmResult, ok := result.(*examples.ConfirmStepResult)
	if !ok {
		t.Fatalf("期望最终结果为 *ConfirmStepResult，实际为 %T", result)
	}

	if confirmResult.Status != "confirmed" {
		t.Errorf("期望订单状态为 confirmed，实际为 %s", confirmResult.Status)
	}

	t.Log("订单处理集成测试 - 正常流程：通过")
}

// TestOrderProcessing_Integration_InventoryFailure 测试订单处理库存不足场景
func TestOrderProcessing_Integration_InventoryFailure(t *testing.T) {
	orderService := &mockOrderService{}
	inventoryService := &mockInventoryService{
		reserveInventoryFunc: func(ctx context.Context, orderID string, items []examples.OrderItem) (*examples.InventoryStepResult, error) {
			return nil, errors.New("库存不足")
		},
	}
	paymentService := &mockPaymentService{}

	sagaDef := examples.NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	orderData := &examples.OrderData{
		CustomerID:    "CUST-INT-002",
		TotalAmount:   999.99,
		Items:         []examples.OrderItem{{ProductID: "PROD-OUT", SKU: "SKU-OUT", Quantity: 1000, UnitPrice: 0.99, Total: 999.00}},
		Address:       examples.ShippingAddress{RecipientName: "测试用户", Phone: "13900139000"},
		PaymentMethod: "credit_card",
		PaymentToken:  "token-test",
		Metadata:      map[string]interface{}{},
	}

	ctx := context.Background()
	_, err := executeSaga(ctx, sagaDef, orderData)
	if err == nil {
		t.Fatal("期望 Saga 执行失败，但成功了")
	}

	t.Log("订单处理集成测试 - 库存不足场景：通过")
}

// TestOrderProcessing_Integration_PaymentFailure 测试订单处理支付失败场景
func TestOrderProcessing_Integration_PaymentFailure(t *testing.T) {
	compensationCount := 0
	orderService := &mockOrderService{
		cancelOrderFunc: func(ctx context.Context, orderID string, reason string) error {
			compensationCount++
			return nil
		},
	}
	inventoryService := &mockInventoryService{
		releaseInventoryFunc: func(ctx context.Context, reservationID string) error {
			compensationCount++
			return nil
		},
	}
	paymentService := &mockPaymentService{
		processPaymentFunc: func(ctx context.Context, orderID string, amount float64, method string, token string) (*examples.PaymentStepResult, error) {
			return nil, errors.New("支付失败：余额不足")
		},
	}

	sagaDef := examples.NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	orderData := &examples.OrderData{
		CustomerID:    "CUST-INT-003",
		TotalAmount:   99999.99,
		Items:         []examples.OrderItem{{ProductID: "PROD-EXPENSIVE", SKU: "SKU-EXP", Quantity: 1, UnitPrice: 99999.99, Total: 99999.99}},
		Address:       examples.ShippingAddress{RecipientName: "测试用户", Phone: "13900139000"},
		PaymentMethod: "credit_card",
		PaymentToken:  "token-invalid",
		Metadata:      map[string]interface{}{},
	}

	ctx := context.Background()
	_, _ = executeSaga(ctx, sagaDef, orderData)

	if compensationCount < 2 {
		t.Errorf("期望至少执行 2 次补偿，实际执行 %d 次", compensationCount)
	}

	t.Log("订单处理集成测试 - 支付失败场景：通过")
}

// ==========================
// 2. 用户注册流程集成测试
// ==========================

// TestUserRegistration_Integration_HappyPath 测试用户注册正常流程
func TestUserRegistration_Integration_HappyPath(t *testing.T) {
	userService := &mockUserService{}
	emailService := &mockEmailService{}
	configService := &mockConfigService{}
	resourceService := &mockResourceService{}
	analyticsService := &mockAnalyticsService{}

	sagaDef := examples.NewUserRegistrationSaga(userService, emailService, configService, resourceService, analyticsService)

	registrationData := &examples.UserRegistrationData{
		Email:        "integration@test.com",
		Username:     "integrationuser",
		Password:     "SecurePass123!",
		FirstName:    "Integration",
		LastName:     "Test",
		Phone:        "+8613900139000",
		Source:       "integration_test",
		Language:     "zh-CN",
		Timezone:     "Asia/Shanghai",
		Subscription: "free",
		AcceptedTOS:  true,
		AcceptedAt:   time.Now(),
		Metadata:     map[string]interface{}{"test": true},
	}

	ctx := context.Background()
	result, err := executeSaga(ctx, sagaDef, registrationData)
	if err != nil {
		t.Fatalf("Saga 执行失败: %v", err)
	}

	analyticsResult, ok := result.(*examples.AnalyticsTrackingResult)
	if !ok {
		t.Fatalf("期望最终结果为 *AnalyticsTrackingResult，实际为 %T", result)
	}

	if analyticsResult.EventType != "user_registered" {
		t.Errorf("期望事件类型为 user_registered，实际为 %s", analyticsResult.EventType)
	}

	t.Log("用户注册集成测试 - 正常流程：通过")
}

// TestUserRegistration_Integration_EmailFailure 测试用户注册邮件发送失败场景
func TestUserRegistration_Integration_EmailFailure(t *testing.T) {
	compensationCount := 0
	userService := &mockUserService{
		deleteUserFunc: func(ctx context.Context, userID string, reason string) error {
			compensationCount++
			return nil
		},
	}
	emailService := &mockEmailService{
		sendVerificationEmailFunc: func(ctx context.Context, userID string, email string, username string) (*examples.EmailVerificationResult, error) {
			return nil, errors.New("邮件服务不可用")
		},
	}
	configService := &mockConfigService{}
	resourceService := &mockResourceService{}
	analyticsService := &mockAnalyticsService{}

	sagaDef := examples.NewUserRegistrationSaga(userService, emailService, configService, resourceService, analyticsService)

	registrationData := &examples.UserRegistrationData{
		Email:        "failtest@test.com",
		Username:     "failtestuser",
		Password:     "SecurePass123!",
		FirstName:    "Fail",
		LastName:     "Test",
		Source:       "integration_test",
		Language:     "zh-CN",
		Timezone:     "Asia/Shanghai",
		Subscription: "free",
		AcceptedTOS:  true,
		AcceptedAt:   time.Now(),
		Metadata:     map[string]interface{}{},
	}

	ctx := context.Background()
	_, _ = executeSaga(ctx, sagaDef, registrationData)

	if compensationCount < 1 {
		t.Errorf("期望至少执行 1 次补偿，实际执行 %d 次", compensationCount)
	}

	t.Log("用户注册集成测试 - 邮件失败场景：通过")
}

// ==========================
// 3. 库存管理流程集成测试
// ==========================

// TestInventoryManagement_Integration_HappyPath 测试库存管理正常流程
func TestInventoryManagement_Integration_HappyPath(t *testing.T) {
	inventoryService := &mockMultiWarehouseInventoryService{}
	auditService := &mockAuditInventoryService{}
	notificationService := &mockNotificationInventoryService{}

	sagaDef := examples.NewInventoryManagementSaga(inventoryService, auditService, notificationService)

	requestData := &examples.InventoryRequestData{
		RequestID:           "REQ-INT-001",
		OrderID:             "ORD-INT-001",
		CustomerID:          "CUST-INT-001",
		Items:               []examples.InventoryItem{{SKU: "SKU-001", Quantity: 100}},
		PreferredWarehouses: []string{"WH-001", "WH-002"},
		AllocationStrategy:  "priority",
		Priority:            5,
		Metadata:            map[string]interface{}{},
	}

	ctx := context.Background()
	result, err := executeSaga(ctx, sagaDef, requestData)
	if err != nil {
		t.Fatalf("Saga 执行失败: %v", err)
	}

	notifyResult, ok := result.(*examples.NotifyInventoryResult)
	if !ok {
		t.Fatalf("期望最终结果为 *NotifyInventoryResult，实际为 %T", result)
	}

	if notifyResult.Status != "sent" {
		t.Errorf("期望通知状态为 sent，实际为 %s", notifyResult.Status)
	}

	t.Log("库存管理集成测试 - 正常流程：通过")
}

// TestInventoryManagement_Integration_InsufficientStock 测试库存管理库存不足场景
func TestInventoryManagement_Integration_InsufficientStock(t *testing.T) {
	inventoryService := &mockMultiWarehouseInventoryService{
		checkAvailabilityFunc: func(ctx context.Context, items []examples.InventoryItem, preferredWarehouses []string) (*examples.CheckInventoryResult, error) {
			return &examples.CheckInventoryResult{
				RequestID:      "REQ-INT-002",
				CanFulfill:     false,
				TotalAvailable: map[string]int{"SKU-001": 10},
				Shortages: []examples.InventoryShortage{
					{SKU: "SKU-001", RequestedQty: 1000, AvailableQty: 10, ShortageQty: 990},
				},
				Metadata: map[string]interface{}{},
			}, nil
		},
	}
	auditService := &mockAuditInventoryService{}
	notificationService := &mockNotificationInventoryService{}

	sagaDef := examples.NewInventoryManagementSaga(inventoryService, auditService, notificationService)

	requestData := &examples.InventoryRequestData{
		RequestID:  "REQ-INT-002",
		OrderID:    "ORD-INT-002",
		CustomerID: "CUST-INT-002",
		Items:      []examples.InventoryItem{{SKU: "SKU-001", Quantity: 1000}},
		Metadata:   map[string]interface{}{},
	}

	ctx := context.Background()
	_, err := executeSaga(ctx, sagaDef, requestData)
	if err == nil {
		t.Fatal("期望 Saga 执行失败，但成功了")
	}

	t.Log("库存管理集成测试 - 库存不足场景：通过")
}

// ==========================
// 4. 支付处理流程集成测试
// ==========================

// TestPaymentProcessing_Integration_HappyPath 测试支付处理正常流程
func TestPaymentProcessing_Integration_HappyPath(t *testing.T) {
	accountService := &mockAccountService{}
	transferService := &mockTransferService{}
	auditService := &mockAuditService{}
	notificationService := &mockNotificationService{}

	sagaDef := examples.NewPaymentProcessingSaga(accountService, transferService, auditService, notificationService)

	transferData := &examples.TransferData{
		FromAccount:  "ACC-FROM-INT-001",
		ToAccount:    "ACC-TO-INT-001",
		Amount:       1000.00,
		Currency:     "CNY",
		Purpose:      "集成测试转账",
		Description:  "Integration test transfer",
		CustomerID:   "CUST-INT-001",
		CustomerName: "集成测试用户",
		RiskLevel:    "low",
		Metadata:     map[string]interface{}{},
	}

	ctx := context.Background()
	result, err := executeSaga(ctx, sagaDef, transferData)
	if err != nil {
		t.Fatalf("Saga 执行失败: %v", err)
	}

	notifyResult, ok := result.(*examples.NotificationResult)
	if !ok {
		t.Fatalf("期望最终结果为 *NotificationResult，实际为 %T", result)
	}

	if notifyResult.Status != "sent" {
		t.Errorf("期望通知状态为 sent，实际为 %s", notifyResult.Status)
	}

	t.Log("支付处理集成测试 - 正常流程：通过")
}

// TestPaymentProcessing_Integration_FreezeFailure 测试支付处理冻结失败场景
func TestPaymentProcessing_Integration_FreezeFailure(t *testing.T) {
	accountService := &mockAccountService{
		freezeAmountFunc: func(ctx context.Context, accountID string, amount float64, reason string) (*examples.FreezeResult, error) {
			return nil, errors.New("余额不足")
		},
	}
	transferService := &mockTransferService{}
	auditService := &mockAuditService{}
	notificationService := &mockNotificationService{}

	sagaDef := examples.NewPaymentProcessingSaga(accountService, transferService, auditService, notificationService)

	transferData := &examples.TransferData{
		FromAccount:  "ACC-FROM-INT-002",
		ToAccount:    "ACC-TO-INT-002",
		Amount:       10000.00,
		Currency:     "CNY",
		Purpose:      "测试余额不足",
		CustomerID:   "CUST-INT-002",
		CustomerName: "测试用户",
		RiskLevel:    "low",
		Metadata:     map[string]interface{}{},
	}

	ctx := context.Background()
	_, err := executeSaga(ctx, sagaDef, transferData)
	if err == nil {
		t.Fatal("期望 Saga 执行失败，但成功了")
	}

	t.Log("支付处理集成测试 - 冻结失败场景：通过")
}

// ==========================
// 5. 跨服务编排场景测试
// ==========================

// TestCrossService_Integration 测试跨服务编排
func TestCrossService_Integration(t *testing.T) {
	var executionOrder []string
	mu := sync.Mutex{}

	orderService := &mockOrderService{
		createOrderFunc: func(ctx context.Context, data *examples.OrderData) (*examples.OrderStepResult, error) {
			mu.Lock()
			executionOrder = append(executionOrder, "order")
			mu.Unlock()
			return &examples.OrderStepResult{
				OrderID:     "ORD-CROSS-001",
				OrderNumber: "ON-CROSS-001",
				Status:      "pending",
				CreatedAt:   time.Now(),
				Items:       data.Items,
				Metadata:    map[string]interface{}{},
			}, nil
		},
	}

	inventoryService := &mockInventoryService{
		reserveInventoryFunc: func(ctx context.Context, orderID string, items []examples.OrderItem) (*examples.InventoryStepResult, error) {
			mu.Lock()
			executionOrder = append(executionOrder, "inventory")
			mu.Unlock()
			reservedItems := make([]examples.ReservedItem, len(items))
			for i, item := range items {
				reservedItems[i] = examples.ReservedItem{
					ProductID:   item.ProductID,
					SKU:         item.SKU,
					Quantity:    item.Quantity,
					WarehouseID: "WH-001",
				}
			}
			return &examples.InventoryStepResult{
				ReservationID: "RES-CROSS-001",
				ReservedItems: reservedItems,
				ExpiresAt:     time.Now().Add(15 * time.Minute),
				WarehouseID:   "WH-001",
				Metadata: map[string]interface{}{
					"order_id":       orderID,
					"amount":         299.99,
					"payment_method": "alipay",
					"payment_token":  "token-cross",
				},
			}, nil
		},
	}

	paymentService := &mockPaymentService{
		processPaymentFunc: func(ctx context.Context, orderID string, amount float64, method string, token string) (*examples.PaymentStepResult, error) {
			mu.Lock()
			executionOrder = append(executionOrder, "payment")
			mu.Unlock()
			return &examples.PaymentStepResult{
				TransactionID: "TXN-CROSS-001",
				PaymentID:     "PAY-CROSS-001",
				Amount:        amount,
				Currency:      "CNY",
				Status:        "success",
				PaidAt:        time.Now(),
				Provider:      "mock-payment",
				Metadata:      map[string]interface{}{"order_id": orderID},
			}, nil
		},
	}

	sagaDef := examples.NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	orderData := &examples.OrderData{
		CustomerID:    "CUST-CROSS-001",
		TotalAmount:   299.99,
		Items:         []examples.OrderItem{{ProductID: "PROD-001", SKU: "SKU-001", Quantity: 1, UnitPrice: 299.99, Total: 299.99}},
		Address:       examples.ShippingAddress{RecipientName: "跨服务测试", Phone: "13900139000"},
		PaymentMethod: "alipay",
		PaymentToken:  "token-cross",
		Metadata:      map[string]interface{}{},
	}

	ctx := context.Background()
	_, err := executeSaga(ctx, sagaDef, orderData)
	if err != nil {
		t.Fatalf("Saga 执行失败: %v", err)
	}

	expectedOrder := []string{"order", "inventory", "payment"}
	if len(executionOrder) < len(expectedOrder) {
		t.Errorf("期望至少执行 %d 个服务，实际执行 %d 个", len(expectedOrder), len(executionOrder))
	}

	for i, expected := range expectedOrder {
		if i < len(executionOrder) && executionOrder[i] != expected {
			t.Errorf("服务调用顺序错误，期望 %s 在位置 %d，实际为 %s", expected, i, executionOrder[i])
		}
	}

	t.Log("跨服务编排集成测试：通过")
}

// ==========================
// 6. 补偿流程端到端测试
// ==========================

// TestCompensation_Integration_FullRollback 测试完整的补偿流程
func TestCompensation_Integration_FullRollback(t *testing.T) {
	var compensatedSteps []string
	mu := sync.Mutex{}

	orderService := &mockOrderService{
		cancelOrderFunc: func(ctx context.Context, orderID string, reason string) error {
			mu.Lock()
			compensatedSteps = append(compensatedSteps, "order")
			mu.Unlock()
			return nil
		},
	}

	inventoryService := &mockInventoryService{
		releaseInventoryFunc: func(ctx context.Context, reservationID string) error {
			mu.Lock()
			compensatedSteps = append(compensatedSteps, "inventory")
			mu.Unlock()
			return nil
		},
	}

	paymentService := &mockPaymentService{
		processPaymentFunc: func(ctx context.Context, orderID string, amount float64, method string, token string) (*examples.PaymentStepResult, error) {
			return nil, errors.New("支付网关错误")
		},
		refundPaymentFunc: func(ctx context.Context, transactionID string, reason string) error {
			mu.Lock()
			compensatedSteps = append(compensatedSteps, "payment")
			mu.Unlock()
			return nil
		},
	}

	sagaDef := examples.NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	orderData := &examples.OrderData{
		CustomerID:    "CUST-COMP-001",
		TotalAmount:   199.99,
		Items:         []examples.OrderItem{{ProductID: "PROD-001", SKU: "SKU-001", Quantity: 1, UnitPrice: 199.99, Total: 199.99}},
		Address:       examples.ShippingAddress{RecipientName: "补偿测试", Phone: "13900139000"},
		PaymentMethod: "credit_card",
		PaymentToken:  "token-comp",
		Metadata:      map[string]interface{}{},
	}

	ctx := context.Background()
	_, _ = executeSaga(ctx, sagaDef, orderData)

	if len(compensatedSteps) < 2 {
		t.Errorf("期望至少补偿 2 个步骤（order 和 inventory），实际补偿 %d 个", len(compensatedSteps))
	}

	expectedCompensationOrder := []string{"inventory", "order"}
	for i, expected := range expectedCompensationOrder {
		if i < len(compensatedSteps) && compensatedSteps[i] != expected {
			t.Errorf("补偿顺序错误，期望 %s 在位置 %d，实际为 %s", expected, i, compensatedSteps[i])
		}
	}

	t.Log("补偿流程端到端测试：通过")
}

// ==========================
// 7. 并发执行测试
// ==========================

// TestConcurrentExecution_Integration 测试多个 Saga 并发执行
func TestConcurrentExecution_Integration(t *testing.T) {
	const concurrentCount = 20

	orderService := &mockOrderService{}
	inventoryService := &mockInventoryService{}
	paymentService := &mockPaymentService{}

	sagaDef := examples.NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			orderData := &examples.OrderData{
				CustomerID:    fmt.Sprintf("CUST-CONCURRENT-%d", index),
				TotalAmount:   100.00,
				Items:         []examples.OrderItem{{ProductID: "PROD-001", SKU: "SKU-001", Quantity: 1, UnitPrice: 100.00, Total: 100.00}},
				Address:       examples.ShippingAddress{RecipientName: "并发测试", Phone: "13900139000"},
				PaymentMethod: "alipay",
				PaymentToken:  fmt.Sprintf("token-concurrent-%d", index),
				Metadata:      map[string]interface{}{"index": index},
			}

			ctx := context.Background()
			_, err := executeSaga(ctx, sagaDef, orderData)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if successCount != concurrentCount {
		t.Errorf("并发执行失败：期望 %d 个成功，实际 %d 个", concurrentCount, successCount)
	}

	t.Logf("并发执行测试：成功执行 %d 个 Saga", successCount)
}

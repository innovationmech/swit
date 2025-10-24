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
	"sync"
	"testing"
	"time"
)

// ==========================
// 订单处理 Saga E2E 测试
// ==========================

// TestOrderProcessingSaga_E2E_HappyPath 测试订单处理的完整正常流程
func TestOrderProcessingSaga_E2E_HappyPath(t *testing.T) {
	// 创建模拟服务（使用已有的 mock）
	orderService := &mockOrderService{}
	inventoryService := &mockInventoryService{}
	paymentService := &mockPaymentService{}

	// 创建 Saga 定义
	sagaDef := NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	// 验证 Saga 定义
	if err := sagaDef.Validate(); err != nil {
		t.Fatalf("Saga 定义验证失败: %v", err)
	}

	// 准备测试数据
	orderData := &OrderData{
		CustomerID:  "CUST-E2E-001",
		TotalAmount: 999.99,
		Items: []OrderItem{
			{ProductID: "PROD-001", SKU: "SKU-001", Quantity: 2, UnitPrice: 499.99, Total: 999.98},
		},
		Address: ShippingAddress{
			RecipientName: "测试用户",
			Phone:         "13800138000",
			Province:      "北京市",
			City:          "北京市",
			District:      "朝阳区",
			Street:        "测试街道100号",
			Zipcode:       "100000",
		},
		PaymentMethod: "credit_card",
		PaymentToken:  "token-e2e-test",
		Metadata:      map[string]interface{}{"test": "e2e"},
	}

	ctx := context.Background()
	steps := sagaDef.GetSteps()

	// 模拟执行所有步骤
	var currentData interface{} = orderData
	for i, step := range steps {
		t.Logf("执行步骤 %d: %s", i+1, step.GetName())

		result, err := step.Execute(ctx, currentData)
		if err != nil {
			t.Fatalf("步骤 %s 执行失败: %v", step.GetName(), err)
		}

		if result == nil {
			t.Fatalf("步骤 %s 返回了 nil 结果", step.GetName())
		}

		currentData = result
	}

	// 验证最终结果
	confirmResult, ok := currentData.(*ConfirmStepResult)
	if !ok {
		t.Fatalf("期望最终结果为 *ConfirmStepResult，实际为 %T", currentData)
	}

	if confirmResult.Status != "confirmed" {
		t.Errorf("期望订单状态为 confirmed，实际为 %s", confirmResult.Status)
	}

	t.Log("订单处理 Saga E2E 测试 - 正常流程：通过")
}

// TestOrderProcessingSaga_E2E_InventoryFailure 测试库存不足场景
func TestOrderProcessingSaga_E2E_InventoryFailure(t *testing.T) {
	orderService := &mockOrderService{}
	inventoryService := &mockInventoryService{
		reserveInventoryFunc: func(ctx context.Context, orderID string, items []OrderItem) (*InventoryStepResult, error) {
			return nil, errors.New("库存不足")
		},
	}
	paymentService := &mockPaymentService{}

	sagaDef := NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	orderData := &OrderData{
		CustomerID:  "CUST-E2E-002",
		TotalAmount: 500.00,
		Items: []OrderItem{
			{ProductID: "PROD-OUT-OF-STOCK", SKU: "SKU-OUT", Quantity: 100, UnitPrice: 5.00, Total: 500.00},
		},
		Address:       ShippingAddress{RecipientName: "测试用户", Phone: "13800138000"},
		PaymentMethod: "credit_card",
		PaymentToken:  "token-test",
		Metadata:      map[string]interface{}{},
	}

	ctx := context.Background()
	steps := sagaDef.GetSteps()

	// 执行步骤直到失败
	var currentData interface{} = orderData
	var executedSteps []interface{}

	failedAtStep := -1
	for i, step := range steps {
		result, err := step.Execute(ctx, currentData)
		if err != nil {
			failedAtStep = i
			t.Logf("步骤 %s 按预期失败: %v", step.GetName(), err)

			// 执行补偿
			for j := i - 1; j >= 0; j-- {
				compStep := steps[j]
				t.Logf("补偿步骤: %s", compStep.GetName())
				if compErr := compStep.Compensate(ctx, executedSteps[j]); compErr != nil {
					t.Logf("补偿步骤 %s 失败: %v", compStep.GetName(), compErr)
				}
			}
			break
		}

		executedSteps = append(executedSteps, result)
		currentData = result
	}

	if failedAtStep == -1 {
		t.Fatal("期望库存步骤失败，但所有步骤都成功了")
	}

	if failedAtStep != 1 { // reserve-inventory 是第二个步骤（索引 1）
		t.Errorf("期望在步骤 1 (reserve-inventory) 失败，实际在步骤 %d 失败", failedAtStep)
	}

	t.Log("订单处理 Saga E2E 测试 - 库存不足场景：通过")
}

// TestOrderProcessingSaga_E2E_PaymentFailure 测试支付失败场景
func TestOrderProcessingSaga_E2E_PaymentFailure(t *testing.T) {
	orderService := &mockOrderService{}
	inventoryService := &mockInventoryService{}
	paymentService := &mockPaymentService{
		processPaymentFunc: func(ctx context.Context, orderID string, amount float64, method string, token string) (*PaymentStepResult, error) {
			return nil, errors.New("支付失败：余额不足")
		},
	}

	sagaDef := NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	orderData := &OrderData{
		CustomerID:    "CUST-E2E-003",
		TotalAmount:   10000.00,
		Items:         []OrderItem{{ProductID: "PROD-EXPENSIVE", SKU: "SKU-EXP", Quantity: 1, UnitPrice: 10000.00, Total: 10000.00}},
		Address:       ShippingAddress{RecipientName: "测试用户", Phone: "13800138000"},
		PaymentMethod: "credit_card",
		PaymentToken:  "token-invalid",
		Metadata:      map[string]interface{}{},
	}

	ctx := context.Background()
	steps := sagaDef.GetSteps()

	var currentData interface{} = orderData
	var executedSteps []interface{}

	failedAtStep := -1
	for i, step := range steps {
		result, err := step.Execute(ctx, currentData)
		if err != nil {
			failedAtStep = i
			t.Logf("步骤 %s 按预期失败: %v", step.GetName(), err)

			// 执行补偿
			for j := i - 1; j >= 0; j-- {
				compStep := steps[j]
				t.Logf("补偿步骤: %s", compStep.GetName())
				if compErr := compStep.Compensate(ctx, executedSteps[j]); compErr != nil {
					t.Errorf("补偿步骤 %s 失败: %v", compStep.GetName(), compErr)
				}
			}
			break
		}

		executedSteps = append(executedSteps, result)
		currentData = result
	}

	if failedAtStep == -1 {
		t.Fatal("期望支付步骤失败，但所有步骤都成功了")
	}

	if failedAtStep != 2 { // process-payment 是第三个步骤（索引 2）
		t.Errorf("期望在步骤 2 (process-payment) 失败，实际在步骤 %d 失败", failedAtStep)
	}

	// 验证补偿了两个步骤
	if len(executedSteps) != 2 {
		t.Errorf("期望补偿 2 个步骤，实际补偿 %d 个", len(executedSteps))
	}

	t.Log("订单处理 Saga E2E 测试 - 支付失败场景：通过")
}

// ==========================
// 支付处理 Saga E2E 测试
// ==========================

// TestPaymentProcessingSaga_E2E_HappyPath 测试支付处理的完整正常流程
func TestPaymentProcessingSaga_E2E_HappyPath(t *testing.T) {
	accountService := &mockAccountService{}
	transferService := &mockTransferService{}
	auditService := &mockAuditService{}
	notificationService := &mockNotificationService{}

	sagaDef := NewPaymentProcessingSaga(accountService, transferService, auditService, notificationService)

	if err := sagaDef.Validate(); err != nil {
		t.Fatalf("Saga 定义验证失败: %v", err)
	}

	transferData := &TransferData{
		FromAccount:  "ACC-FROM-001",
		ToAccount:    "ACC-TO-001",
		Amount:       500.00,
		Currency:     "CNY",
		Purpose:      "测试转账",
		Description:  "E2E 测试转账",
		CustomerID:   "CUST-E2E-001",
		CustomerName: "测试用户",
		RiskLevel:    "low",
		Metadata:     map[string]interface{}{},
	}

	ctx := context.Background()
	steps := sagaDef.GetSteps()

	var currentData interface{} = transferData
	for i, step := range steps {
		t.Logf("执行步骤 %d: %s", i+1, step.GetName())

		result, err := step.Execute(ctx, currentData)
		if err != nil {
			t.Fatalf("步骤 %s 执行失败: %v", step.GetName(), err)
		}

		currentData = result
	}

	// 验证最终结果
	notifyResult, ok := currentData.(*NotificationResult)
	if !ok {
		t.Fatalf("期望最终结果为 *NotificationResult，实际为 %T", currentData)
	}

	if notifyResult.Status != "sent" {
		t.Errorf("期望通知状态为 sent，实际为 %s", notifyResult.Status)
	}

	t.Log("支付处理 Saga E2E 测试 - 正常流程：通过")
}

// ==========================
// 库存管理 Saga E2E 测试
// ==========================

// TestInventoryManagementSaga_E2E_HappyPath 测试库存管理的完整正常流程
func TestInventoryManagementSaga_E2E_HappyPath(t *testing.T) {
	inventoryService := &mockMultiWarehouseInventoryService{}
	auditService := &mockAuditInventoryService{}
	notificationService := &mockNotificationInventoryService{}

	sagaDef := NewInventoryManagementSaga(inventoryService, auditService, notificationService)

	if err := sagaDef.Validate(); err != nil {
		t.Fatalf("Saga 定义验证失败: %v", err)
	}

	requestData := &InventoryRequestData{
		RequestID:  "REQ-E2E-001",
		OrderID:    "ORD-E2E-001",
		CustomerID: "CUST-E2E-001",
		Items: []InventoryItem{
			{SKU: "SKU-001", Quantity: 50},
		},
		PreferredWarehouses: []string{"WH-001", "WH-002"},
		AllocationStrategy:  "priority",
		Priority:            5,
		Metadata:            map[string]interface{}{},
	}

	ctx := context.Background()
	steps := sagaDef.GetSteps()

	var currentData interface{} = requestData
	for i, step := range steps {
		t.Logf("执行步骤 %d: %s", i+1, step.GetName())

		result, err := step.Execute(ctx, currentData)
		if err != nil {
			t.Fatalf("步骤 %s 执行失败: %v", step.GetName(), err)
		}

		currentData = result
	}

	// 验证最终结果
	notifyResult, ok := currentData.(*NotifyInventoryResult)
	if !ok {
		t.Fatalf("期望最终结果为 *NotifyInventoryResult，实际为 %T", currentData)
	}

	if notifyResult.Status != "sent" {
		t.Errorf("期望通知状态为 sent，实际为 %s", notifyResult.Status)
	}

	t.Log("库存管理 Saga E2E 测试 - 正常流程：通过")
}

// ==========================
// 用户注册 Saga E2E 测试
// ==========================

// TestUserRegistrationSaga_E2E_HappyPath 测试用户注册的完整正常流程
func TestUserRegistrationSaga_E2E_HappyPath(t *testing.T) {
	userService := &mockUserService{}
	emailService := &mockEmailService{}
	configService := &mockConfigService{}
	resourceService := &mockResourceService{}
	analyticsService := &mockAnalyticsService{}

	sagaDef := NewUserRegistrationSaga(userService, emailService, configService, resourceService, analyticsService)

	if err := sagaDef.Validate(); err != nil {
		t.Fatalf("Saga 定义验证失败: %v", err)
	}

	registrationData := &UserRegistrationData{
		Email:        "newuser@example.com",
		Username:     "newuser123",
		Password:     "SecurePass123!",
		FirstName:    "New",
		LastName:     "User",
		Phone:        "+8613800138000",
		Source:       "web",
		Language:     "zh-CN",
		Timezone:     "Asia/Shanghai",
		Subscription: "free",
		AcceptedTOS:  true,
		AcceptedAt:   time.Now(),
		Metadata:     map[string]interface{}{},
	}

	ctx := context.Background()
	steps := sagaDef.GetSteps()

	var currentData interface{} = registrationData
	for i, step := range steps {
		t.Logf("执行步骤 %d: %s", i+1, step.GetName())

		result, err := step.Execute(ctx, currentData)
		if err != nil {
			t.Fatalf("步骤 %s 执行失败: %v", step.GetName(), err)
		}

		currentData = result
	}

	// 验证最终结果
	analyticsResult, ok := currentData.(*AnalyticsTrackingResult)
	if !ok {
		t.Fatalf("期望最终结果为 *AnalyticsTrackingResult，实际为 %T", currentData)
	}

	if analyticsResult.EventType != "user_registered" {
		t.Errorf("期望事件类型为 user_registered，实际为 %s", analyticsResult.EventType)
	}

	t.Log("用户注册 Saga E2E 测试 - 正常流程：通过")
}

// ==========================
// 并发执行测试
// ==========================

// TestMultipleSagas_E2E_Concurrent 测试多个 Saga 并发执行
func TestMultipleSagas_E2E_Concurrent(t *testing.T) {
	const concurrentSagas = 10

	var wg sync.WaitGroup
	errors := make(chan error, concurrentSagas)

	for i := 0; i < concurrentSagas; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			orderService := &mockOrderService{}
			inventoryService := &mockInventoryService{}
			paymentService := &mockPaymentService{}

			sagaDef := NewOrderProcessingSaga(orderService, inventoryService, paymentService)

			orderData := &OrderData{
				CustomerID:    "CUST-CONCURRENT",
				TotalAmount:   100.00,
				Items:         []OrderItem{{ProductID: "PROD-001", SKU: "SKU-001", Quantity: 1, UnitPrice: 100.00, Total: 100.00}},
				Address:       ShippingAddress{RecipientName: "并发测试用户", Phone: "13800138000"},
				PaymentMethod: "credit_card",
				PaymentToken:  "token-concurrent",
				Metadata:      map[string]interface{}{"index": index},
			}

			ctx := context.Background()
			steps := sagaDef.GetSteps()

			var currentData interface{} = orderData
			for _, step := range steps {
				result, err := step.Execute(ctx, currentData)
				if err != nil {
					errors <- err
					return
				}
				currentData = result
			}

			errors <- nil
		}(i)
	}

	wg.Wait()
	close(errors)

	failedCount := 0
	for err := range errors {
		if err != nil {
			failedCount++
			t.Logf("并发 Saga 失败: %v", err)
		}
	}

	if failedCount > 0 {
		t.Errorf("%d 个并发 Saga 中有 %d 个失败", concurrentSagas, failedCount)
	}

	t.Logf("并发测试：%d 个 Saga 全部成功", concurrentSagas)
}

// ==========================
// 性能基准测试
// ==========================

// BenchmarkOrderProcessingSaga_E2E 订单处理 Saga 性能基准测试
func BenchmarkOrderProcessingSaga_E2E(b *testing.B) {
	orderService := &mockOrderService{}
	inventoryService := &mockInventoryService{}
	paymentService := &mockPaymentService{}

	sagaDef := NewOrderProcessingSaga(orderService, inventoryService, paymentService)

	orderData := &OrderData{
		CustomerID:    "CUST-BENCH-001",
		TotalAmount:   100.00,
		Items:         []OrderItem{{ProductID: "PROD-001", SKU: "SKU-001", Quantity: 1, UnitPrice: 100.00, Total: 100.00}},
		Address:       ShippingAddress{RecipientName: "基准测试用户", Phone: "13800138000"},
		PaymentMethod: "credit_card",
		PaymentToken:  "token-bench",
		Metadata:      map[string]interface{}{},
	}

	ctx := context.Background()
	steps := sagaDef.GetSteps()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var currentData interface{} = orderData
		for _, step := range steps {
			result, err := step.Execute(ctx, currentData)
			if err != nil {
				b.Fatalf("Saga 执行失败: %v", err)
			}
			currentData = result
		}
	}
}

// BenchmarkPaymentProcessingSaga_E2E 支付处理 Saga 性能基准测试
func BenchmarkPaymentProcessingSaga_E2E(b *testing.B) {
	accountService := &mockAccountService{}
	transferService := &mockTransferService{}
	auditService := &mockAuditService{}
	notificationService := &mockNotificationService{}

	sagaDef := NewPaymentProcessingSaga(accountService, transferService, auditService, notificationService)

	transferData := &TransferData{
		FromAccount:  "ACC-FROM-BENCH",
		ToAccount:    "ACC-TO-BENCH",
		Amount:       100.00,
		Currency:     "CNY",
		Purpose:      "基准测试",
		CustomerID:   "CUST-BENCH-001",
		CustomerName: "基准测试用户",
		RiskLevel:    "low",
		Metadata:     map[string]interface{}{},
	}

	ctx := context.Background()
	steps := sagaDef.GetSteps()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var currentData interface{} = transferData
		for _, step := range steps {
			result, err := step.Execute(ctx, currentData)
			if err != nil {
				b.Fatalf("Saga 执行失败: %v", err)
			}
			currentData = result
		}
	}
}

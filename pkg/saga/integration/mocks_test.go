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

package integration

import (
	"context"
	"errors"
	"time"

	"github.com/innovationmech/swit/pkg/saga/examples"
)

// ==========================
// 订单服务 Mock
// ==========================

type mockOrderService struct {
	createOrderFunc  func(ctx context.Context, data *examples.OrderData) (*examples.OrderStepResult, error)
	cancelOrderFunc  func(ctx context.Context, orderID string, reason string) error
	confirmOrderFunc func(ctx context.Context, orderID string) (*examples.ConfirmStepResult, error)
}

func (m *mockOrderService) CreateOrder(ctx context.Context, data *examples.OrderData) (*examples.OrderStepResult, error) {
	if m.createOrderFunc != nil {
		return m.createOrderFunc(ctx, data)
	}
	return &examples.OrderStepResult{
		OrderID:     "order-mock-123",
		OrderNumber: "ON-MOCK-001",
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

func (m *mockOrderService) ConfirmOrder(ctx context.Context, orderID string) (*examples.ConfirmStepResult, error) {
	if m.confirmOrderFunc != nil {
		return m.confirmOrderFunc(ctx, orderID)
	}
	return &examples.ConfirmStepResult{
		OrderID:       orderID,
		ConfirmedAt:   time.Now(),
		Status:        "confirmed",
		EstimatedShip: time.Now().Add(24 * time.Hour),
		TrackingInfo:  "TRACK-MOCK-001",
		Metadata:      map[string]interface{}{},
	}, nil
}

// ==========================
// 库存服务 Mock
// ==========================

type mockInventoryService struct {
	reserveInventoryFunc func(ctx context.Context, orderID string, items []examples.OrderItem) (*examples.InventoryStepResult, error)
	releaseInventoryFunc func(ctx context.Context, reservationID string) error
}

func (m *mockInventoryService) ReserveInventory(ctx context.Context, orderID string, items []examples.OrderItem) (*examples.InventoryStepResult, error) {
	if m.reserveInventoryFunc != nil {
		return m.reserveInventoryFunc(ctx, orderID, items)
	}
	reservedItems := make([]examples.ReservedItem, len(items))
	for i, item := range items {
		reservedItems[i] = examples.ReservedItem{
			ProductID:   item.ProductID,
			SKU:         item.SKU,
			Quantity:    item.Quantity,
			WarehouseID: "WH-MOCK-001",
		}
	}
	return &examples.InventoryStepResult{
		ReservationID: "RES-MOCK-123",
		ReservedItems: reservedItems,
		ExpiresAt:     time.Now().Add(15 * time.Minute),
		WarehouseID:   "WH-MOCK-001",
		Metadata: map[string]interface{}{
			"order_id":       orderID,
			"amount":         100.0,
			"payment_method": "credit_card",
			"payment_token":  "token-mock",
		},
	}, nil
}

func (m *mockInventoryService) ReleaseInventory(ctx context.Context, reservationID string) error {
	if m.releaseInventoryFunc != nil {
		return m.releaseInventoryFunc(ctx, reservationID)
	}
	return nil
}

// ==========================
// 支付服务 Mock
// ==========================

type mockPaymentService struct {
	processPaymentFunc func(ctx context.Context, orderID string, amount float64, method string, token string) (*examples.PaymentStepResult, error)
	refundPaymentFunc  func(ctx context.Context, transactionID string, reason string) error
}

func (m *mockPaymentService) ProcessPayment(ctx context.Context, orderID string, amount float64, method string, token string) (*examples.PaymentStepResult, error) {
	if m.processPaymentFunc != nil {
		return m.processPaymentFunc(ctx, orderID, amount, method, token)
	}
	return &examples.PaymentStepResult{
		TransactionID: "TXN-MOCK-123",
		PaymentID:     "PAY-MOCK-123",
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
// 用户服务 Mock
// ==========================

type mockUserService struct {
	createUserFunc       func(ctx context.Context, data *examples.UserRegistrationData) (*examples.UserCreationResult, error)
	deleteUserFunc       func(ctx context.Context, userID string, reason string) error
	getUserByEmailFunc   func(ctx context.Context, email string) (*examples.UserCreationResult, error)
	getUserByUsernameFunc func(ctx context.Context, username string) (*examples.UserCreationResult, error)
}

func (m *mockUserService) CreateUser(ctx context.Context, data *examples.UserRegistrationData) (*examples.UserCreationResult, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, data)
	}
	return &examples.UserCreationResult{
		UserID:       "user-mock-123",
		Username:     data.Username,
		Email:        data.Email,
		Status:       "pending",
		CreatedAt:    time.Now(),
		ActivationID: "ACT-MOCK-123",
		Metadata:     map[string]interface{}{},
	}, nil
}

func (m *mockUserService) DeleteUser(ctx context.Context, userID string, reason string) error {
	if m.deleteUserFunc != nil {
		return m.deleteUserFunc(ctx, userID, reason)
	}
	return nil
}

func (m *mockUserService) GetUserByEmail(ctx context.Context, email string) (*examples.UserCreationResult, error) {
	if m.getUserByEmailFunc != nil {
		return m.getUserByEmailFunc(ctx, email)
	}
	return nil, errors.New("user not found")
}

func (m *mockUserService) GetUserByUsername(ctx context.Context, username string) (*examples.UserCreationResult, error) {
	if m.getUserByUsernameFunc != nil {
		return m.getUserByUsernameFunc(ctx, username)
	}
	return nil, errors.New("user not found")
}

// ==========================
// 邮件服务 Mock
// ==========================

type mockEmailService struct {
	sendVerificationEmailFunc  func(ctx context.Context, userID string, email string, username string) (*examples.EmailVerificationResult, error)
	cancelVerificationEmailFunc func(ctx context.Context, verificationID string) error
	resendVerificationEmailFunc func(ctx context.Context, verificationID string) error
}

func (m *mockEmailService) SendVerificationEmail(ctx context.Context, userID string, email string, username string) (*examples.EmailVerificationResult, error) {
	if m.sendVerificationEmailFunc != nil {
		return m.sendVerificationEmailFunc(ctx, userID, email, username)
	}
	return &examples.EmailVerificationResult{
		VerificationID:   "VER-MOCK-123",
		UserID:           userID,
		Email:            email,
		VerificationCode: "MOCK-CODE-123",
		SentAt:           time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		Provider:         "mock-email",
		Metadata:         map[string]interface{}{},
	}, nil
}

func (m *mockEmailService) CancelVerificationEmail(ctx context.Context, verificationID string) error {
	if m.cancelVerificationEmailFunc != nil {
		return m.cancelVerificationEmailFunc(ctx, verificationID)
	}
	return nil
}

func (m *mockEmailService) ResendVerificationEmail(ctx context.Context, verificationID string) error {
	if m.resendVerificationEmailFunc != nil {
		return m.resendVerificationEmailFunc(ctx, verificationID)
	}
	return nil
}

// ==========================
// 配置服务 Mock
// ==========================

type mockConfigService struct {
	initializeUserConfigFunc func(ctx context.Context, userID string, language string, timezone string) (*examples.ConfigInitResult, error)
	deleteUserConfigFunc     func(ctx context.Context, userID string) error
	updateUserConfigFunc     func(ctx context.Context, userID string, settings map[string]interface{}) error
}

func (m *mockConfigService) InitializeUserConfig(ctx context.Context, userID string, language string, timezone string) (*examples.ConfigInitResult, error) {
	if m.initializeUserConfigFunc != nil {
		return m.initializeUserConfigFunc(ctx, userID, language, timezone)
	}
	return &examples.ConfigInitResult{
		UserID:        userID,
		ConfigID:      "CFG-MOCK-123",
		Settings:      map[string]interface{}{"theme": "default", "language": language, "timezone": timezone},
		Preferences:   map[string]interface{}{},
		InitializedAt: time.Now(),
		Metadata:      map[string]interface{}{},
	}, nil
}

func (m *mockConfigService) DeleteUserConfig(ctx context.Context, userID string) error {
	if m.deleteUserConfigFunc != nil {
		return m.deleteUserConfigFunc(ctx, userID)
	}
	return nil
}

func (m *mockConfigService) UpdateUserConfig(ctx context.Context, userID string, settings map[string]interface{}) error {
	if m.updateUserConfigFunc != nil {
		return m.updateUserConfigFunc(ctx, userID, settings)
	}
	return nil
}

// ==========================
// 资源服务 Mock
// ==========================

type mockResourceService struct {
	allocateResourcesFunc func(ctx context.Context, userID string, subscription string) (*examples.ResourceAllocationResult, error)
	releaseResourcesFunc  func(ctx context.Context, userID string) error
	getResourceQuotaFunc  func(ctx context.Context, subscription string) (*examples.ResourceAllocationResult, error)
}

func (m *mockResourceService) AllocateResources(ctx context.Context, userID string, subscription string) (*examples.ResourceAllocationResult, error) {
	if m.allocateResourcesFunc != nil {
		return m.allocateResourcesFunc(ctx, userID, subscription)
	}
	return &examples.ResourceAllocationResult{
		UserID:         userID,
		WalletID:       "WALLET-MOCK-123",
		StorageQuota:   10737418240, // 10GB
		APIQuota:       10000,
		ComputeQuota:   100,
		AllocatedAt:    time.Now(),
		SubscriptionID: subscription,
		Metadata:       map[string]interface{}{},
	}, nil
}

func (m *mockResourceService) ReleaseResources(ctx context.Context, userID string) error {
	if m.releaseResourcesFunc != nil {
		return m.releaseResourcesFunc(ctx, userID)
	}
	return nil
}

func (m *mockResourceService) GetResourceQuota(ctx context.Context, subscription string) (*examples.ResourceAllocationResult, error) {
	if m.getResourceQuotaFunc != nil {
		return m.getResourceQuotaFunc(ctx, subscription)
	}
	return &examples.ResourceAllocationResult{
		StorageQuota: 10737418240,
		APIQuota:     10000,
		ComputeQuota: 100,
		Metadata:     map[string]interface{}{},
	}, nil
}

// ==========================
// 分析服务 Mock
// ==========================

type mockAnalyticsService struct {
	trackRegistrationEventFunc func(ctx context.Context, userID string, data *examples.UserRegistrationData) (*examples.AnalyticsTrackingResult, error)
	deleteTrackingEventFunc    func(ctx context.Context, eventID string) error
}

func (m *mockAnalyticsService) TrackRegistrationEvent(ctx context.Context, userID string, data *examples.UserRegistrationData) (*examples.AnalyticsTrackingResult, error) {
	if m.trackRegistrationEventFunc != nil {
		return m.trackRegistrationEventFunc(ctx, userID, data)
	}
	return &examples.AnalyticsTrackingResult{
		UserID:     userID,
		EventID:    "EVT-MOCK-123",
		EventType:  "user_registered",
		TrackedAt:  time.Now(),
		SessionID:  "SESSION-MOCK-123",
		DeviceInfo: map[string]interface{}{"device": "mock"},
		Metadata:   map[string]interface{}{},
	}, nil
}

func (m *mockAnalyticsService) DeleteTrackingEvent(ctx context.Context, eventID string) error {
	if m.deleteTrackingEventFunc != nil {
		return m.deleteTrackingEventFunc(ctx, eventID)
	}
	return nil
}

// ==========================
// 多仓库库存服务 Mock
// ==========================

type mockMultiWarehouseInventoryService struct {
	checkAvailabilityFunc   func(ctx context.Context, items []examples.InventoryItem, preferredWarehouses []string) (*examples.CheckInventoryResult, error)
	reserveInventoryFunc    func(ctx context.Context, requestID string, items []examples.InventoryItem, warehouses []string) (*examples.ReserveInventoryResult, error)
	allocateInventoryFunc   func(ctx context.Context, reservationID string) (*examples.AllocateInventoryResult, error)
	releaseReservationFunc  func(ctx context.Context, reservationID string, reason string) error
	restoreInventoryFunc    func(ctx context.Context, allocationID string, items []examples.AllocatedWarehouseItem) error
	getWarehouseInfoFunc    func(ctx context.Context, warehouseID string) (*examples.WarehouseInfo, error)
	getInventoryDetailsFunc func(ctx context.Context, warehouseID string, sku string) (*examples.WarehouseInventory, error)
}

func (m *mockMultiWarehouseInventoryService) CheckAvailability(ctx context.Context, items []examples.InventoryItem, preferredWarehouses []string) (*examples.CheckInventoryResult, error) {
	if m.checkAvailabilityFunc != nil {
		return m.checkAvailabilityFunc(ctx, items, preferredWarehouses)
	}
	return &examples.CheckInventoryResult{
		RequestID: "REQ-MOCK-123",
		AvailableWarehouses: []examples.WarehouseInfo{
			{WarehouseID: "WH-001", WarehouseName: "仓库1", Status: "active", Priority: 10},
		},
		InventoryDetails: []examples.WarehouseInventory{
			{WarehouseID: "WH-001", SKU: "SKU-001", Available: 100, Reserved: 10, Total: 110},
		},
		TotalAvailable: map[string]int{"SKU-001": 100},
		CheckedAt:      time.Now(),
		CanFulfill:     true,
		Shortages:      []examples.InventoryShortage{},
		Metadata: map[string]interface{}{
			"request_id":      "REQ-MOCK-123",
			"requested_items": items,
		},
	}, nil
}

func (m *mockMultiWarehouseInventoryService) ReserveInventory(ctx context.Context, requestID string, items []examples.InventoryItem, warehouses []string) (*examples.ReserveInventoryResult, error) {
	if m.reserveInventoryFunc != nil {
		return m.reserveInventoryFunc(ctx, requestID, items, warehouses)
	}
	reservedItems := make([]examples.ReservedWarehouseItem, 0)
	for _, item := range items {
		reservedItems = append(reservedItems, examples.ReservedWarehouseItem{
			WarehouseID:      "WH-001",
			SKU:              item.SKU,
			ReservedQty:      item.Quantity,
			ReservationToken: "TOKEN-MOCK-123",
		})
	}
	return &examples.ReserveInventoryResult{
		ReservationID: "RES-MOCK-123",
		RequestID:     requestID,
		ReservedItems: reservedItems,
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		TotalReserved: 50,
		Status:        "active",
		Metadata: map[string]interface{}{
			"request_id": requestID,
		},
	}, nil
}

func (m *mockMultiWarehouseInventoryService) AllocateInventory(ctx context.Context, reservationID string) (*examples.AllocateInventoryResult, error) {
	if m.allocateInventoryFunc != nil {
		return m.allocateInventoryFunc(ctx, reservationID)
	}
	return &examples.AllocateInventoryResult{
		AllocationID:  "ALLOC-MOCK-123",
		RequestID:     "REQ-MOCK-123",
		ReservationID: reservationID,
		AllocatedItems: []examples.AllocatedWarehouseItem{
			{WarehouseID: "WH-001", SKU: "SKU-001", AllocatedQty: 50, TransactionID: "TXN-MOCK-123"},
		},
		CompletedAt: time.Now(),
		Status:      "completed",
		Metadata: map[string]interface{}{
			"request_id": "REQ-MOCK-123",
		},
	}, nil
}

func (m *mockMultiWarehouseInventoryService) ReleaseReservation(ctx context.Context, reservationID string, reason string) error {
	if m.releaseReservationFunc != nil {
		return m.releaseReservationFunc(ctx, reservationID, reason)
	}
	return nil
}

func (m *mockMultiWarehouseInventoryService) RestoreInventory(ctx context.Context, allocationID string, items []examples.AllocatedWarehouseItem) error {
	if m.restoreInventoryFunc != nil {
		return m.restoreInventoryFunc(ctx, allocationID, items)
	}
	return nil
}

func (m *mockMultiWarehouseInventoryService) GetWarehouseInfo(ctx context.Context, warehouseID string) (*examples.WarehouseInfo, error) {
	if m.getWarehouseInfoFunc != nil {
		return m.getWarehouseInfoFunc(ctx, warehouseID)
	}
	return &examples.WarehouseInfo{
		WarehouseID:   warehouseID,
		WarehouseName: "测试仓库",
		Status:        "active",
		Priority:      10,
	}, nil
}

func (m *mockMultiWarehouseInventoryService) GetInventoryDetails(ctx context.Context, warehouseID string, sku string) (*examples.WarehouseInventory, error) {
	if m.getInventoryDetailsFunc != nil {
		return m.getInventoryDetailsFunc(ctx, warehouseID, sku)
	}
	return &examples.WarehouseInventory{
		WarehouseID: warehouseID,
		SKU:         sku,
		Available:   100,
		Reserved:    10,
		Total:       110,
	}, nil
}

// ==========================
// 审计服务 Mock (库存)
// ==========================

type mockAuditInventoryService struct {
	createAuditRecordFunc func(ctx context.Context, requestID string, operation string, details map[string]interface{}) (*examples.AuditInventoryResult, error)
	updateAuditRecordFunc func(ctx context.Context, auditID string, status string, details map[string]interface{}) error
}

func (m *mockAuditInventoryService) CreateAuditRecord(ctx context.Context, requestID string, operation string, details map[string]interface{}) (*examples.AuditInventoryResult, error) {
	if m.createAuditRecordFunc != nil {
		return m.createAuditRecordFunc(ctx, requestID, operation, details)
	}
	return &examples.AuditInventoryResult{
		AuditID:    "AUD-MOCK-123",
		RequestID:  requestID,
		AuditType:  operation,
		RecordedAt: time.Now(),
		Status:     "created",
		Metadata: map[string]interface{}{
			"request_id":  requestID,
			"customer_id": "CUST-MOCK-123",
		},
	}, nil
}

func (m *mockAuditInventoryService) UpdateAuditRecord(ctx context.Context, auditID string, status string, details map[string]interface{}) error {
	if m.updateAuditRecordFunc != nil {
		return m.updateAuditRecordFunc(ctx, auditID, status, details)
	}
	return nil
}

// ==========================
// 通知服务 Mock (库存)
// ==========================

type mockNotificationInventoryService struct {
	sendInventoryNotificationFunc func(ctx context.Context, requestID string, recipients []string, message string) (*examples.NotifyInventoryResult, error)
	sendInventoryAlertFunc        func(ctx context.Context, requestID string, alertType string, details map[string]interface{}) error
}

func (m *mockNotificationInventoryService) SendInventoryNotification(ctx context.Context, requestID string, recipients []string, message string) (*examples.NotifyInventoryResult, error) {
	if m.sendInventoryNotificationFunc != nil {
		return m.sendInventoryNotificationFunc(ctx, requestID, recipients, message)
	}
	return &examples.NotifyInventoryResult{
		NotificationID: "NOTIF-MOCK-123",
		RequestID:      requestID,
		Recipients:     recipients,
		Channel:        "email",
		SentAt:         time.Now(),
		Status:         "sent",
		Metadata:       map[string]interface{}{},
	}, nil
}

func (m *mockNotificationInventoryService) SendInventoryAlert(ctx context.Context, requestID string, alertType string, details map[string]interface{}) error {
	if m.sendInventoryAlertFunc != nil {
		return m.sendInventoryAlertFunc(ctx, requestID, alertType, details)
	}
	return nil
}

// ==========================
// 账户服务 Mock
// ==========================

type mockAccountService struct {
	validateAccountFunc  func(ctx context.Context, accountID string) (*examples.AccountBalance, error)
	getBalanceFunc       func(ctx context.Context, accountID string) (*examples.AccountBalance, error)
	freezeAmountFunc     func(ctx context.Context, accountID string, amount float64, reason string) (*examples.FreezeResult, error)
	unfreezeAmountFunc   func(ctx context.Context, freezeID string, reason string) error
	deductFrozenAmountFunc func(ctx context.Context, freezeID string) error
	addAmountFunc        func(ctx context.Context, accountID string, amount float64, transactionID string) error
}

func (m *mockAccountService) ValidateAccount(ctx context.Context, accountID string) (*examples.AccountBalance, error) {
	if m.validateAccountFunc != nil {
		return m.validateAccountFunc(ctx, accountID)
	}
	return &examples.AccountBalance{
		AccountID: accountID,
		Available: 10000.00,
		Frozen:    0.00,
		Total:     10000.00,
		Currency:  "CNY",
	}, nil
}

func (m *mockAccountService) GetBalance(ctx context.Context, accountID string) (*examples.AccountBalance, error) {
	if m.getBalanceFunc != nil {
		return m.getBalanceFunc(ctx, accountID)
	}
	return &examples.AccountBalance{
		AccountID: accountID,
		Available: 10000.00,
		Frozen:    0.00,
		Total:     10000.00,
		Currency:  "CNY",
	}, nil
}

func (m *mockAccountService) FreezeAmount(ctx context.Context, accountID string, amount float64, reason string) (*examples.FreezeResult, error) {
	if m.freezeAmountFunc != nil {
		return m.freezeAmountFunc(ctx, accountID, amount, reason)
	}
	return &examples.FreezeResult{
		FreezeID:   "FRZ-MOCK-123",
		AccountID:  accountID,
		Amount:     amount,
		Currency:   "CNY",
		FrozenAt:   time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		FreezeType: "transfer",
		Metadata:   map[string]interface{}{},
	}, nil
}

func (m *mockAccountService) UnfreezeAmount(ctx context.Context, freezeID string, reason string) error {
	if m.unfreezeAmountFunc != nil {
		return m.unfreezeAmountFunc(ctx, freezeID, reason)
	}
	return nil
}

func (m *mockAccountService) DeductFrozenAmount(ctx context.Context, freezeID string) error {
	if m.deductFrozenAmountFunc != nil {
		return m.deductFrozenAmountFunc(ctx, freezeID)
	}
	return nil
}

func (m *mockAccountService) AddAmount(ctx context.Context, accountID string, amount float64, transactionID string) error {
	if m.addAmountFunc != nil {
		return m.addAmountFunc(ctx, accountID, amount, transactionID)
	}
	return nil
}

// ==========================
// 转账服务 Mock
// ==========================

type mockTransferService struct {
	createTransferFunc      func(ctx context.Context, data *examples.TransferData) (*examples.TransferResult, error)
	executeTransferFunc     func(ctx context.Context, transferID string, freezeID string) (*examples.TransferResult, error)
	cancelTransferFunc      func(ctx context.Context, transferID string, reason string) error
	getTransferStatusFunc   func(ctx context.Context, transferID string) (string, error)
}

func (m *mockTransferService) CreateTransfer(ctx context.Context, data *examples.TransferData) (*examples.TransferResult, error) {
	if m.createTransferFunc != nil {
		return m.createTransferFunc(ctx, data)
	}
	return &examples.TransferResult{
		TransactionID:   "TXN-MOCK-123",
		TransferID:      "TXF-MOCK-123",
		FromAccount:     data.FromAccount,
		ToAccount:       data.ToAccount,
		Amount:          data.Amount,
		Currency:        data.Currency,
		Status:          "pending",
		CompletedAt:     time.Now(),
		TransactionHash: "HASH-MOCK-123",
		Metadata:        map[string]interface{}{},
	}, nil
}

func (m *mockTransferService) ExecuteTransfer(ctx context.Context, transferID string, freezeID string) (*examples.TransferResult, error) {
	if m.executeTransferFunc != nil {
		return m.executeTransferFunc(ctx, transferID, freezeID)
	}
	return &examples.TransferResult{
		TransactionID:   "TXN-MOCK-123",
		TransferID:      transferID,
		FromAccount:     "ACC-FROM-MOCK",
		ToAccount:       "ACC-TO-MOCK",
		Amount:          1000.00,
		Currency:        "CNY",
		Status:          "completed",
		CompletedAt:     time.Now(),
		TransactionHash: "HASH-MOCK-123",
		Metadata: map[string]interface{}{
			"freeze_id": freezeID,
		},
	}, nil
}

func (m *mockTransferService) CancelTransfer(ctx context.Context, transferID string, reason string) error {
	if m.cancelTransferFunc != nil {
		return m.cancelTransferFunc(ctx, transferID, reason)
	}
	return nil
}

func (m *mockTransferService) GetTransferStatus(ctx context.Context, transferID string) (string, error) {
	if m.getTransferStatusFunc != nil {
		return m.getTransferStatusFunc(ctx, transferID)
	}
	return "completed", nil
}

// ==========================
// 审计服务 Mock (支付)
// ==========================

type mockAuditService struct {
	createAuditRecordFunc func(ctx context.Context, transferID string, data *examples.TransferData, result *examples.TransferResult) (*examples.AuditRecordResult, error)
	updateAuditRecordFunc func(ctx context.Context, auditID string, status string, details map[string]interface{}) error
}

func (m *mockAuditService) CreateAuditRecord(ctx context.Context, transferID string, data *examples.TransferData, result *examples.TransferResult) (*examples.AuditRecordResult, error) {
	if m.createAuditRecordFunc != nil {
		return m.createAuditRecordFunc(ctx, transferID, data, result)
	}
	return &examples.AuditRecordResult{
		AuditID:        "AUD-MOCK-123",
		TransferID:     transferID,
		RecordType:     "transfer",
		CreatedAt:      time.Now(),
		Status:         "recorded",
		ReviewRequired: false,
		Metadata:       map[string]interface{}{},
	}, nil
}

func (m *mockAuditService) UpdateAuditRecord(ctx context.Context, auditID string, status string, details map[string]interface{}) error {
	if m.updateAuditRecordFunc != nil {
		return m.updateAuditRecordFunc(ctx, auditID, status, details)
	}
	return nil
}

// ==========================
// 通知服务 Mock (支付)
// ==========================

type mockNotificationService struct {
	sendTransferNotificationFunc func(ctx context.Context, transferID string, recipients []string, data *examples.TransferData) (*examples.NotificationResult, error)
	sendFailureNotificationFunc  func(ctx context.Context, transferID string, reason string) error
}

func (m *mockNotificationService) SendTransferNotification(ctx context.Context, transferID string, recipients []string, data *examples.TransferData) (*examples.NotificationResult, error) {
	if m.sendTransferNotificationFunc != nil {
		return m.sendTransferNotificationFunc(ctx, transferID, recipients, data)
	}
	return &examples.NotificationResult{
		NotificationID: "NOTIF-MOCK-123",
		TransferID:     transferID,
		Recipients:     recipients,
		SentAt:         time.Now(),
		Status:         "sent",
		Channel:        "email",
		Metadata:       map[string]interface{}{},
	}, nil
}

func (m *mockNotificationService) SendFailureNotification(ctx context.Context, transferID string, reason string) error {
	if m.sendFailureNotificationFunc != nil {
		return m.sendFailureNotificationFunc(ctx, transferID, reason)
	}
	return nil
}


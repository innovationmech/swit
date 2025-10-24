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

// mockMultiWarehouseInventoryService 模拟多仓库库存服务
type mockMultiWarehouseInventoryService struct {
	checkAvailabilityFunc   func(ctx context.Context, items []InventoryItem, preferredWarehouses []string) (*CheckInventoryResult, error)
	reserveInventoryFunc    func(ctx context.Context, requestID string, items []InventoryItem, warehouses []string) (*ReserveInventoryResult, error)
	allocateInventoryFunc   func(ctx context.Context, reservationID string) (*AllocateInventoryResult, error)
	releaseReservationFunc  func(ctx context.Context, reservationID string, reason string) error
	restoreInventoryFunc    func(ctx context.Context, allocationID string, items []AllocatedWarehouseItem) error
	getWarehouseInfoFunc    func(ctx context.Context, warehouseID string) (*WarehouseInfo, error)
	getInventoryDetailsFunc func(ctx context.Context, warehouseID string, sku string) (*WarehouseInventory, error)
}

func (m *mockMultiWarehouseInventoryService) CheckAvailability(ctx context.Context, items []InventoryItem, preferredWarehouses []string) (*CheckInventoryResult, error) {
	if m.checkAvailabilityFunc != nil {
		return m.checkAvailabilityFunc(ctx, items, preferredWarehouses)
	}
	return &CheckInventoryResult{
		RequestID: "REQ-001",
		AvailableWarehouses: []WarehouseInfo{
			{WarehouseID: "WH-001", WarehouseName: "华东仓", Status: "active", Priority: 10},
			{WarehouseID: "WH-002", WarehouseName: "华南仓", Status: "active", Priority: 8},
		},
		InventoryDetails: []WarehouseInventory{
			{WarehouseID: "WH-001", SKU: "SKU-001", Available: 100, Reserved: 10, Total: 110},
			{WarehouseID: "WH-002", SKU: "SKU-001", Available: 50, Reserved: 5, Total: 55},
		},
		TotalAvailable: map[string]int{"SKU-001": 150},
		CheckedAt:      time.Now(),
		CanFulfill:     true,
		Shortages:      []InventoryShortage{},
		Metadata:       map[string]interface{}{},
	}, nil
}

func (m *mockMultiWarehouseInventoryService) ReserveInventory(ctx context.Context, requestID string, items []InventoryItem, warehouses []string) (*ReserveInventoryResult, error) {
	if m.reserveInventoryFunc != nil {
		return m.reserveInventoryFunc(ctx, requestID, items, warehouses)
	}
	return &ReserveInventoryResult{
		ReservationID: "RES-001",
		RequestID:     requestID,
		ReservedItems: []ReservedWarehouseItem{
			{WarehouseID: "WH-001", SKU: "SKU-001", ReservedQty: 30, ReservationToken: "TOKEN-001"},
			{WarehouseID: "WH-002", SKU: "SKU-001", ReservedQty: 20, ReservationToken: "TOKEN-002"},
		},
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		TotalReserved: 50,
		Status:        "active",
		Metadata:      map[string]interface{}{},
	}, nil
}

func (m *mockMultiWarehouseInventoryService) AllocateInventory(ctx context.Context, reservationID string) (*AllocateInventoryResult, error) {
	if m.allocateInventoryFunc != nil {
		return m.allocateInventoryFunc(ctx, reservationID)
	}
	return &AllocateInventoryResult{
		AllocationID:  "ALLOC-001",
		RequestID:     "REQ-001",
		ReservationID: reservationID,
		AllocatedItems: []AllocatedWarehouseItem{
			{WarehouseID: "WH-001", SKU: "SKU-001", AllocatedQty: 30, TransactionID: "TXN-001"},
			{WarehouseID: "WH-002", SKU: "SKU-001", AllocatedQty: 20, TransactionID: "TXN-002"},
		},
		CompletedAt: time.Now(),
		Status:      "completed",
		Metadata:    map[string]interface{}{},
	}, nil
}

func (m *mockMultiWarehouseInventoryService) ReleaseReservation(ctx context.Context, reservationID string, reason string) error {
	if m.releaseReservationFunc != nil {
		return m.releaseReservationFunc(ctx, reservationID, reason)
	}
	return nil
}

func (m *mockMultiWarehouseInventoryService) RestoreInventory(ctx context.Context, allocationID string, items []AllocatedWarehouseItem) error {
	if m.restoreInventoryFunc != nil {
		return m.restoreInventoryFunc(ctx, allocationID, items)
	}
	return nil
}

func (m *mockMultiWarehouseInventoryService) GetWarehouseInfo(ctx context.Context, warehouseID string) (*WarehouseInfo, error) {
	if m.getWarehouseInfoFunc != nil {
		return m.getWarehouseInfoFunc(ctx, warehouseID)
	}
	return &WarehouseInfo{
		WarehouseID:   warehouseID,
		WarehouseName: "测试仓库",
		Status:        "active",
		Capacity:      1000,
		Available:     800,
	}, nil
}

func (m *mockMultiWarehouseInventoryService) GetInventoryDetails(ctx context.Context, warehouseID string, sku string) (*WarehouseInventory, error) {
	if m.getInventoryDetailsFunc != nil {
		return m.getInventoryDetailsFunc(ctx, warehouseID, sku)
	}
	return &WarehouseInventory{
		WarehouseID: warehouseID,
		SKU:         sku,
		Available:   100,
		Reserved:    10,
		Total:       110,
	}, nil
}

// mockAuditInventoryService 模拟审计服务
type mockAuditInventoryService struct {
	createAuditRecordFunc func(ctx context.Context, requestID string, operation string, details map[string]interface{}) (*AuditInventoryResult, error)
	updateAuditRecordFunc func(ctx context.Context, auditID string, status string, details map[string]interface{}) error
}

func (m *mockAuditInventoryService) CreateAuditRecord(ctx context.Context, requestID string, operation string, details map[string]interface{}) (*AuditInventoryResult, error) {
	if m.createAuditRecordFunc != nil {
		return m.createAuditRecordFunc(ctx, requestID, operation, details)
	}
	return &AuditInventoryResult{
		AuditID:    "AUD-001",
		RequestID:  requestID,
		AuditType:  operation,
		RecordedAt: time.Now(),
		Status:     "created",
		Metadata:   map[string]interface{}{},
	}, nil
}

func (m *mockAuditInventoryService) UpdateAuditRecord(ctx context.Context, auditID string, status string, details map[string]interface{}) error {
	if m.updateAuditRecordFunc != nil {
		return m.updateAuditRecordFunc(ctx, auditID, status, details)
	}
	return nil
}

// mockNotificationInventoryService 模拟通知服务
type mockNotificationInventoryService struct {
	sendInventoryNotificationFunc func(ctx context.Context, requestID string, recipients []string, message string) (*NotifyInventoryResult, error)
	sendInventoryAlertFunc        func(ctx context.Context, requestID string, alertType string, details map[string]interface{}) error
}

func (m *mockNotificationInventoryService) SendInventoryNotification(ctx context.Context, requestID string, recipients []string, message string) (*NotifyInventoryResult, error) {
	if m.sendInventoryNotificationFunc != nil {
		return m.sendInventoryNotificationFunc(ctx, requestID, recipients, message)
	}
	return &NotifyInventoryResult{
		NotificationID: "NOTIF-001",
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
// CheckInventoryStep 测试
// ==========================

func TestCheckInventoryStep_Execute_Success(t *testing.T) {
	mockService := &mockMultiWarehouseInventoryService{}
	step := &CheckInventoryStep{service: mockService}

	requestData := &InventoryRequestData{
		RequestID:  "REQ-001",
		OrderID:    "ORD-001",
		CustomerID: "CUST-001",
		Items: []InventoryItem{
			{SKU: "SKU-001", Quantity: 50},
		},
		PreferredWarehouses: []string{"WH-001", "WH-002"},
		AllocationStrategy:  "priority",
		Priority:            5,
	}

	ctx := context.Background()
	result, err := step.Execute(ctx, requestData)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	checkResult, ok := result.(*CheckInventoryResult)
	if !ok {
		t.Fatalf("Expected *CheckInventoryResult, got %T", result)
	}

	if !checkResult.CanFulfill {
		t.Error("Expected CanFulfill to be true")
	}

	if len(checkResult.AvailableWarehouses) != 2 {
		t.Errorf("Expected 2 warehouses, got %d", len(checkResult.AvailableWarehouses))
	}
}

func TestCheckInventoryStep_Execute_InsufficientInventory(t *testing.T) {
	mockService := &mockMultiWarehouseInventoryService{
		checkAvailabilityFunc: func(ctx context.Context, items []InventoryItem, preferredWarehouses []string) (*CheckInventoryResult, error) {
			return &CheckInventoryResult{
				RequestID:      "REQ-001",
				CanFulfill:     false,
				TotalAvailable: map[string]int{"SKU-001": 10},
				Shortages: []InventoryShortage{
					{SKU: "SKU-001", RequestedQty: 50, AvailableQty: 10, ShortageQty: 40},
				},
				Metadata: map[string]interface{}{},
			}, nil
		},
	}
	step := &CheckInventoryStep{service: mockService}

	requestData := &InventoryRequestData{
		RequestID:  "REQ-001",
		OrderID:    "ORD-001",
		CustomerID: "CUST-001",
		Items: []InventoryItem{
			{SKU: "SKU-001", Quantity: 50},
		},
	}

	ctx := context.Background()
	_, err := step.Execute(ctx, requestData)

	if err == nil {
		t.Fatal("Expected error for insufficient inventory, got nil")
	}
}

func TestCheckInventoryStep_Compensate(t *testing.T) {
	mockService := &mockMultiWarehouseInventoryService{}
	step := &CheckInventoryStep{service: mockService}

	result := &CheckInventoryResult{
		RequestID:  "REQ-001",
		CanFulfill: true,
	}

	ctx := context.Background()
	err := step.Compensate(ctx, result)

	if err != nil {
		t.Errorf("Expected no error for compensation, got %v", err)
	}
}

func TestCheckInventoryStep_Metadata(t *testing.T) {
	step := &CheckInventoryStep{}

	metadata := step.GetMetadata()
	if metadata["step_type"] != "inventory_check" {
		t.Error("Expected step_type to be inventory_check")
	}
	if metadata["critical"] != true {
		t.Error("Expected critical to be true")
	}
}

func TestCheckInventoryStep_AllMethods(t *testing.T) {
	step := &CheckInventoryStep{}

	if step.GetID() != "check-inventory" {
		t.Errorf("Expected ID check-inventory, got %s", step.GetID())
	}

	if step.GetName() != "检查库存" {
		t.Errorf("Expected name 检查库存, got %s", step.GetName())
	}

	desc := step.GetDescription()
	if desc == "" {
		t.Error("Expected non-empty description")
	}

	if step.GetTimeout() != 15*time.Second {
		t.Errorf("Expected timeout 15s, got %v", step.GetTimeout())
	}

	policy := step.GetRetryPolicy()
	if policy == nil {
		t.Error("Expected non-nil retry policy")
	}

	// Test IsRetryable
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("Expected context.DeadlineExceeded to be not retryable")
	}
	if step.IsRetryable(ErrInsufficientInventory) {
		t.Error("Expected ErrInsufficientInventory to be not retryable")
	}
	if !step.IsRetryable(errors.New("network error")) {
		t.Error("Expected network error to be retryable")
	}
}

// ==========================
// ReserveMultiWarehouseInventoryStep 测试
// ==========================

func TestReserveMultiWarehouseInventoryStep_Execute_Success(t *testing.T) {
	requestedItems := []InventoryItem{
		{SKU: "SKU-001", Quantity: 50}, // 请求50个，而不是全部150个可用库存
	}
	
	mockService := &mockMultiWarehouseInventoryService{}
	step := &ReserveMultiWarehouseInventoryStep{service: mockService}

	checkResult := &CheckInventoryResult{
		RequestID: "REQ-001",
		AvailableWarehouses: []WarehouseInfo{
			{WarehouseID: "WH-001", Status: "active", Priority: 10},
			{WarehouseID: "WH-002", Status: "active", Priority: 8},
		},
		TotalAvailable: map[string]int{"SKU-001": 150}, // 总可用150个
		CanFulfill:     true,
		Metadata: map[string]interface{}{
			"request_id":      "REQ-001",
			"requested_items": requestedItems, // 存储原始请求的商品项
		},
	}

	ctx := context.Background()
	result, err := step.Execute(ctx, checkResult)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	reserveResult, ok := result.(*ReserveInventoryResult)
	if !ok {
		t.Fatalf("Expected *ReserveInventoryResult, got %T", result)
	}

	if reserveResult.ReservationID == "" {
		t.Error("Expected ReservationID to be set")
	}

	if len(reserveResult.ReservedItems) != 2 {
		t.Errorf("Expected 2 reserved items, got %d", len(reserveResult.ReservedItems))
	}
}

func TestReserveMultiWarehouseInventoryStep_Compensate_Success(t *testing.T) {
	releaseCalled := false
	mockService := &mockMultiWarehouseInventoryService{
		releaseReservationFunc: func(ctx context.Context, reservationID string, reason string) error {
			releaseCalled = true
			if reservationID != "RES-001" {
				t.Errorf("Expected reservationID RES-001, got %s", reservationID)
			}
			if reason != "saga_compensation" {
				t.Errorf("Expected reason saga_compensation, got %s", reason)
			}
			return nil
		},
	}
	step := &ReserveMultiWarehouseInventoryStep{service: mockService}

	result := &ReserveInventoryResult{
		ReservationID: "RES-001",
		Status:        "active",
	}

	ctx := context.Background()
	err := step.Compensate(ctx, result)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !releaseCalled {
		t.Error("Expected ReleaseReservation to be called")
	}
}

func TestReserveMultiWarehouseInventoryStep_UsesRequestedQuantity(t *testing.T) {
	// 验证预留步骤使用请求的数量，而不是总可用数量
	requestedItems := []InventoryItem{
		{SKU: "SKU-001", Quantity: 10}, // 只请求10个
	}
	
	var reservedQuantity int
	mockService := &mockMultiWarehouseInventoryService{
		reserveInventoryFunc: func(ctx context.Context, requestID string, items []InventoryItem, warehouses []string) (*ReserveInventoryResult, error) {
			// 验证传递的是请求数量，而不是可用数量
			if len(items) != 1 {
				t.Errorf("Expected 1 item, got %d", len(items))
			}
			if items[0].Quantity != 10 {
				t.Errorf("Expected quantity 10 (requested), got %d (should not be 100 which is available)", items[0].Quantity)
			}
			reservedQuantity = items[0].Quantity
			
			return &ReserveInventoryResult{
				ReservationID: "RES-001",
				RequestID:     requestID,
				ReservedItems: []ReservedWarehouseItem{
					{WarehouseID: "WH-001", SKU: "SKU-001", ReservedQty: items[0].Quantity},
				},
				Status:   "active",
				Metadata: map[string]interface{}{},
			}, nil
		},
	}
	step := &ReserveMultiWarehouseInventoryStep{service: mockService}

	checkResult := &CheckInventoryResult{
		RequestID: "REQ-001",
		AvailableWarehouses: []WarehouseInfo{
			{WarehouseID: "WH-001", Status: "active", Priority: 10},
		},
		TotalAvailable: map[string]int{"SKU-001": 100}, // 可用100个
		CanFulfill:     true,
		Metadata: map[string]interface{}{
			"request_id":      "REQ-001",
			"requested_items": requestedItems, // 但只请求10个
		},
	}

	ctx := context.Background()
	result, err := step.Execute(ctx, checkResult)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if reservedQuantity != 10 {
		t.Errorf("Expected to reserve 10 (requested quantity), but reserved %d", reservedQuantity)
	}

	reserveResult, ok := result.(*ReserveInventoryResult)
	if !ok {
		t.Fatalf("Expected *ReserveInventoryResult, got %T", result)
	}

	if reserveResult.ReservationID == "" {
		t.Error("Expected ReservationID to be set")
	}
}

func TestReserveMultiWarehouseInventoryStep_AllMethods(t *testing.T) {
	step := &ReserveMultiWarehouseInventoryStep{}

	if step.GetID() != "reserve-multi-warehouse-inventory" {
		t.Errorf("Expected ID reserve-multi-warehouse-inventory, got %s", step.GetID())
	}

	if step.GetName() != "预留多仓库库存" {
		t.Errorf("Expected name 预留多仓库库存, got %s", step.GetName())
	}

	desc := step.GetDescription()
	if desc == "" {
		t.Error("Expected non-empty description")
	}

	if step.GetTimeout() != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", step.GetTimeout())
	}

	policy := step.GetRetryPolicy()
	if policy == nil {
		t.Error("Expected non-nil retry policy")
	}

	metadata := step.GetMetadata()
	if metadata["step_type"] != "multi_warehouse_reservation" {
		t.Error("Expected step_type to be multi_warehouse_reservation")
	}

	// Test IsRetryable
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("Expected context.DeadlineExceeded to be not retryable")
	}
	if step.IsRetryable(ErrInsufficientInventory) {
		t.Error("Expected ErrInsufficientInventory to be not retryable")
	}
}

// ==========================
// AllocateMultiWarehouseInventoryStep 测试
// ==========================

func TestAllocateMultiWarehouseInventoryStep_Execute_Success(t *testing.T) {
	mockService := &mockMultiWarehouseInventoryService{}
	step := &AllocateMultiWarehouseInventoryStep{service: mockService}

	reserveResult := &ReserveInventoryResult{
		ReservationID: "RES-001",
		RequestID:     "REQ-001",
		ReservedItems: []ReservedWarehouseItem{
			{WarehouseID: "WH-001", SKU: "SKU-001", ReservedQty: 30},
		},
		Status: "active",
		Metadata: map[string]interface{}{
			"request_id": "REQ-001",
		},
	}

	ctx := context.Background()
	result, err := step.Execute(ctx, reserveResult)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	allocateResult, ok := result.(*AllocateInventoryResult)
	if !ok {
		t.Fatalf("Expected *AllocateInventoryResult, got %T", result)
	}

	if allocateResult.AllocationID == "" {
		t.Error("Expected AllocationID to be set")
	}

	if allocateResult.Status != "completed" {
		t.Errorf("Expected status completed, got %s", allocateResult.Status)
	}
}

func TestAllocateMultiWarehouseInventoryStep_Compensate_Success(t *testing.T) {
	restoreCalled := false
	mockService := &mockMultiWarehouseInventoryService{
		restoreInventoryFunc: func(ctx context.Context, allocationID string, items []AllocatedWarehouseItem) error {
			restoreCalled = true
			if allocationID != "ALLOC-001" {
				t.Errorf("Expected allocationID ALLOC-001, got %s", allocationID)
			}
			if len(items) != 2 {
				t.Errorf("Expected 2 items, got %d", len(items))
			}
			return nil
		},
	}
	step := &AllocateMultiWarehouseInventoryStep{service: mockService}

	result := &AllocateInventoryResult{
		AllocationID: "ALLOC-001",
		Status:       "completed",
		AllocatedItems: []AllocatedWarehouseItem{
			{WarehouseID: "WH-001", SKU: "SKU-001", AllocatedQty: 30},
			{WarehouseID: "WH-002", SKU: "SKU-001", AllocatedQty: 20},
		},
	}

	ctx := context.Background()
	err := step.Compensate(ctx, result)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !restoreCalled {
		t.Error("Expected RestoreInventory to be called")
	}
}

func TestAllocateMultiWarehouseInventoryStep_Compensate_PartialStatus(t *testing.T) {
	restoreCalled := false
	mockService := &mockMultiWarehouseInventoryService{
		restoreInventoryFunc: func(ctx context.Context, allocationID string, items []AllocatedWarehouseItem) error {
			restoreCalled = true
			return nil
		},
	}
	step := &AllocateMultiWarehouseInventoryStep{service: mockService}

	result := &AllocateInventoryResult{
		AllocationID: "ALLOC-001",
		Status:       "partial",
		AllocatedItems: []AllocatedWarehouseItem{
			{WarehouseID: "WH-001", SKU: "SKU-001", AllocatedQty: 30},
		},
	}

	ctx := context.Background()
	err := step.Compensate(ctx, result)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !restoreCalled {
		t.Error("Expected RestoreInventory to be called for partial status")
	}
}

func TestAllocateMultiWarehouseInventoryStep_Compensate_FailedStatus(t *testing.T) {
	restoreCalled := false
	mockService := &mockMultiWarehouseInventoryService{
		restoreInventoryFunc: func(ctx context.Context, allocationID string, items []AllocatedWarehouseItem) error {
			restoreCalled = true
			return nil
		},
	}
	step := &AllocateMultiWarehouseInventoryStep{service: mockService}

	result := &AllocateInventoryResult{
		AllocationID: "ALLOC-001",
		Status:       "failed",
	}

	ctx := context.Background()
	err := step.Compensate(ctx, result)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if restoreCalled {
		t.Error("Expected RestoreInventory NOT to be called for failed status")
	}
}

func TestAllocateMultiWarehouseInventoryStep_AllMethods(t *testing.T) {
	step := &AllocateMultiWarehouseInventoryStep{}

	if step.GetID() != "allocate-multi-warehouse-inventory" {
		t.Errorf("Expected ID allocate-multi-warehouse-inventory, got %s", step.GetID())
	}

	if step.GetName() != "分配多仓库库存" {
		t.Errorf("Expected name 分配多仓库库存, got %s", step.GetName())
	}

	desc := step.GetDescription()
	if desc == "" {
		t.Error("Expected non-empty description")
	}

	if step.GetTimeout() != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", step.GetTimeout())
	}

	policy := step.GetRetryPolicy()
	if policy == nil {
		t.Error("Expected non-nil retry policy")
	}

	metadata := step.GetMetadata()
	if metadata["step_type"] != "multi_warehouse_allocation" {
		t.Error("Expected step_type to be multi_warehouse_allocation")
	}
	if metadata["critical"] != true {
		t.Error("Expected critical to be true")
	}

	// Test IsRetryable
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("Expected context.DeadlineExceeded to be not retryable")
	}
	if step.IsRetryable(ErrReservationExpired) {
		t.Error("Expected ErrReservationExpired to be not retryable")
	}
}

// ==========================
// ReleaseReservationStep 测试
// ==========================

func TestReleaseReservationStep_Execute_Success(t *testing.T) {
	mockService := &mockMultiWarehouseInventoryService{}
	step := &ReleaseReservationStep{service: mockService}

	allocateResult := &AllocateInventoryResult{
		AllocationID:  "ALLOC-001",
		RequestID:     "REQ-001",
		ReservationID: "RES-001",
		AllocatedItems: []AllocatedWarehouseItem{
			{WarehouseID: "WH-001", SKU: "SKU-001", AllocatedQty: 30},
			{WarehouseID: "WH-002", SKU: "SKU-001", AllocatedQty: 20},
		},
		Status: "completed",
		Metadata: map[string]interface{}{
			"request_id": "REQ-001",
		},
	}

	ctx := context.Background()
	result, err := step.Execute(ctx, allocateResult)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	releaseResult, ok := result.(*ReleaseInventoryResult)
	if !ok {
		t.Fatalf("Expected *ReleaseInventoryResult, got %T", result)
	}

	if releaseResult.ReservationID != "RES-001" {
		t.Errorf("Expected ReservationID RES-001, got %s", releaseResult.ReservationID)
	}

	if len(releaseResult.ReleasedItems) != 2 {
		t.Errorf("Expected 2 released items, got %d", len(releaseResult.ReleasedItems))
	}
}

func TestReleaseReservationStep_Compensate(t *testing.T) {
	mockService := &mockMultiWarehouseInventoryService{}
	step := &ReleaseReservationStep{service: mockService}

	result := &ReleaseInventoryResult{
		ReleaseID:     "REL-001",
		ReservationID: "RES-001",
	}

	ctx := context.Background()
	err := step.Compensate(ctx, result)

	if err != nil {
		t.Errorf("Expected no error for compensation, got %v", err)
	}
}

func TestReleaseReservationStep_AllMethods(t *testing.T) {
	step := &ReleaseReservationStep{}

	if step.GetID() != "release-reservation" {
		t.Errorf("Expected ID release-reservation, got %s", step.GetID())
	}

	if step.GetName() != "释放预留" {
		t.Errorf("Expected name 释放预留, got %s", step.GetName())
	}

	desc := step.GetDescription()
	if desc == "" {
		t.Error("Expected non-empty description")
	}

	if step.GetTimeout() != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", step.GetTimeout())
	}

	policy := step.GetRetryPolicy()
	if policy == nil {
		t.Error("Expected non-nil retry policy")
	}

	metadata := step.GetMetadata()
	if metadata["step_type"] != "reservation_release" {
		t.Error("Expected step_type to be reservation_release")
	}

	// Test IsRetryable
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("Expected context.DeadlineExceeded to be not retryable")
	}
}

// ==========================
// CreateInventoryAuditRecordStep 测试
// ==========================

func TestCreateInventoryAuditRecordStep_Execute_Success(t *testing.T) {
	mockService := &mockAuditInventoryService{}
	step := &CreateInventoryAuditRecordStep{service: mockService}

	releaseResult := &ReleaseInventoryResult{
		ReleaseID:     "REL-001",
		RequestID:     "REQ-001",
		ReservationID: "RES-001",
		ReleasedItems: []ReleasedItem{
			{WarehouseID: "WH-001", SKU: "SKU-001", ReleasedQty: 30},
		},
		ReleasedAt: time.Now(),
		Metadata: map[string]interface{}{
			"request_id": "REQ-001",
		},
	}

	ctx := context.Background()
	result, err := step.Execute(ctx, releaseResult)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	auditResult, ok := result.(*AuditInventoryResult)
	if !ok {
		t.Fatalf("Expected *AuditInventoryResult, got %T", result)
	}

	if auditResult.AuditID == "" {
		t.Error("Expected AuditID to be set")
	}

	if auditResult.RequestID != "REQ-001" {
		t.Errorf("Expected RequestID REQ-001, got %s", auditResult.RequestID)
	}
}

func TestCreateInventoryAuditRecordStep_Compensate_Success(t *testing.T) {
	updateCalled := false
	mockService := &mockAuditInventoryService{
		updateAuditRecordFunc: func(ctx context.Context, auditID string, status string, details map[string]interface{}) error {
			updateCalled = true
			if auditID != "AUD-001" {
				t.Errorf("Expected auditID AUD-001, got %s", auditID)
			}
			if status != "cancelled" {
				t.Errorf("Expected status cancelled, got %s", status)
			}
			return nil
		},
	}
	step := &CreateInventoryAuditRecordStep{service: mockService}

	result := &AuditInventoryResult{
		AuditID:   "AUD-001",
		RequestID: "REQ-001",
	}

	ctx := context.Background()
	err := step.Compensate(ctx, result)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !updateCalled {
		t.Error("Expected UpdateAuditRecord to be called")
	}
}

func TestCreateInventoryAuditRecordStep_AllMethods(t *testing.T) {
	step := &CreateInventoryAuditRecordStep{}

	if step.GetID() != "create-inventory-audit-record" {
		t.Errorf("Expected ID create-inventory-audit-record, got %s", step.GetID())
	}

	if step.GetName() != "创建库存审计记录" {
		t.Errorf("Expected name 创建库存审计记录, got %s", step.GetName())
	}

	desc := step.GetDescription()
	if desc == "" {
		t.Error("Expected non-empty description")
	}

	if step.GetTimeout() != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", step.GetTimeout())
	}

	policy := step.GetRetryPolicy()
	if policy == nil {
		t.Error("Expected non-nil retry policy")
	}

	metadata := step.GetMetadata()
	if metadata["step_type"] != "inventory_audit_record" {
		t.Error("Expected step_type to be inventory_audit_record")
	}
	if metadata["critical"] != true {
		t.Error("Expected critical to be true")
	}

	// Test IsRetryable
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("Expected context.DeadlineExceeded to be not retryable")
	}
}

// ==========================
// SendInventoryNotificationStep 测试
// ==========================

func TestSendInventoryNotificationStep_Execute_Success(t *testing.T) {
	mockService := &mockNotificationInventoryService{}
	step := &SendInventoryNotificationStep{service: mockService}

	auditResult := &AuditInventoryResult{
		AuditID:   "AUD-001",
		RequestID: "REQ-001",
		Metadata: map[string]interface{}{
			"request_id":  "REQ-001",
			"customer_id": "CUST-001",
		},
	}

	ctx := context.Background()
	result, err := step.Execute(ctx, auditResult)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	notifyResult, ok := result.(*NotifyInventoryResult)
	if !ok {
		t.Fatalf("Expected *NotifyInventoryResult, got %T", result)
	}

	if notifyResult.NotificationID == "" {
		t.Error("Expected NotificationID to be set")
	}

	if notifyResult.Status != "sent" {
		t.Errorf("Expected status sent, got %s", notifyResult.Status)
	}
}

func TestSendInventoryNotificationStep_Execute_Failure(t *testing.T) {
	mockService := &mockNotificationInventoryService{
		sendInventoryNotificationFunc: func(ctx context.Context, requestID string, recipients []string, message string) (*NotifyInventoryResult, error) {
			return nil, errors.New("notification service unavailable")
		},
	}
	step := &SendInventoryNotificationStep{service: mockService}

	auditResult := &AuditInventoryResult{
		AuditID:   "AUD-001",
		RequestID: "REQ-001",
		Metadata: map[string]interface{}{
			"request_id":  "REQ-001",
			"customer_id": "CUST-001",
		},
	}

	ctx := context.Background()
	result, err := step.Execute(ctx, auditResult)

	// 通知失败不应导致错误
	if err != nil {
		t.Fatalf("Expected no error even on notification failure, got %v", err)
	}

	notifyResult, ok := result.(*NotifyInventoryResult)
	if !ok {
		t.Fatalf("Expected *NotifyInventoryResult, got %T", result)
	}

	if notifyResult.Status != "failed" {
		t.Errorf("Expected status failed, got %s", notifyResult.Status)
	}
}

func TestSendInventoryNotificationStep_Compensate_Success(t *testing.T) {
	alertCalled := false
	mockService := &mockNotificationInventoryService{
		sendInventoryAlertFunc: func(ctx context.Context, requestID string, alertType string, details map[string]interface{}) error {
			alertCalled = true
			if requestID != "REQ-001" {
				t.Errorf("Expected requestID REQ-001, got %s", requestID)
			}
			if alertType != "allocation_cancelled" {
				t.Errorf("Expected alertType allocation_cancelled, got %s", alertType)
			}
			return nil
		},
	}
	step := &SendInventoryNotificationStep{service: mockService}

	result := &NotifyInventoryResult{
		NotificationID: "NOTIF-001",
		RequestID:      "REQ-001",
	}

	ctx := context.Background()
	err := step.Compensate(ctx, result)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !alertCalled {
		t.Error("Expected SendInventoryAlert to be called")
	}
}

func TestSendInventoryNotificationStep_AllMethods(t *testing.T) {
	step := &SendInventoryNotificationStep{}

	if step.GetID() != "send-inventory-notification" {
		t.Errorf("Expected ID send-inventory-notification, got %s", step.GetID())
	}

	if step.GetName() != "发送库存通知" {
		t.Errorf("Expected name 发送库存通知, got %s", step.GetName())
	}

	desc := step.GetDescription()
	if desc == "" {
		t.Error("Expected non-empty description")
	}

	if step.GetTimeout() != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", step.GetTimeout())
	}

	policy := step.GetRetryPolicy()
	if policy == nil {
		t.Error("Expected non-nil retry policy")
	}

	metadata := step.GetMetadata()
	if metadata["step_type"] != "inventory_notification" {
		t.Error("Expected step_type to be inventory_notification")
	}
	if metadata["critical"] != false {
		t.Error("Expected critical to be false")
	}

	// Test IsRetryable
	if step.IsRetryable(context.DeadlineExceeded) {
		t.Error("Expected context.DeadlineExceeded to be not retryable")
	}
}

// ==========================
// Saga 定义测试
// ==========================

func TestNewInventoryManagementSaga(t *testing.T) {
	inventoryService := &mockMultiWarehouseInventoryService{}
	auditService := &mockAuditInventoryService{}
	notificationService := &mockNotificationInventoryService{}

	sagaDef := NewInventoryManagementSaga(inventoryService, auditService, notificationService)

	if sagaDef.GetID() != "inventory-management-saga" {
		t.Errorf("Expected ID inventory-management-saga, got %s", sagaDef.GetID())
	}

	if sagaDef.GetName() != "库存管理Saga" {
		t.Errorf("Expected name 库存管理Saga, got %s", sagaDef.GetName())
	}

	steps := sagaDef.GetSteps()
	if len(steps) != 6 {
		t.Errorf("Expected 6 steps, got %d", len(steps))
	}

	expectedStepIDs := []string{
		"check-inventory",
		"reserve-multi-warehouse-inventory",
		"allocate-multi-warehouse-inventory",
		"release-reservation",
		"create-inventory-audit-record",
		"send-inventory-notification",
	}

	for i, expectedID := range expectedStepIDs {
		if steps[i].GetID() != expectedID {
			t.Errorf("Step %d: expected ID %s, got %s", i, expectedID, steps[i].GetID())
		}
	}

	if sagaDef.GetTimeout() != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", sagaDef.GetTimeout())
	}

	metadata := sagaDef.GetMetadata()
	if metadata["saga_type"] != "inventory_management" {
		t.Error("Expected saga_type to be inventory_management")
	}
	if metadata["multi_warehouse"] != true {
		t.Error("Expected multi_warehouse to be true")
	}

	// Test additional methods
	desc := sagaDef.GetDescription()
	if desc == "" {
		t.Error("Expected non-empty description")
	}

	retryPolicy := sagaDef.GetRetryPolicy()
	if retryPolicy == nil {
		t.Error("Expected non-nil retry policy")
	}

	compensationStrategy := sagaDef.GetCompensationStrategy()
	if compensationStrategy == nil {
		t.Error("Expected non-nil compensation strategy")
	}
}

func TestInventoryManagementSagaDefinition_Validate(t *testing.T) {
	tests := []struct {
		name    string
		sagaDef *InventoryManagementSagaDefinition
		wantErr bool
	}{
		{
			name: "Valid saga definition",
			sagaDef: &InventoryManagementSagaDefinition{
				id:      "test-saga",
				name:    "Test Saga",
				steps:   []saga.SagaStep{&CheckInventoryStep{}},
				timeout: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "Missing ID",
			sagaDef: &InventoryManagementSagaDefinition{
				name:    "Test Saga",
				steps:   []saga.SagaStep{&CheckInventoryStep{}},
				timeout: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "Missing name",
			sagaDef: &InventoryManagementSagaDefinition{
				id:      "test-saga",
				steps:   []saga.SagaStep{&CheckInventoryStep{}},
				timeout: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "No steps",
			sagaDef: &InventoryManagementSagaDefinition{
				id:      "test-saga",
				name:    "Test Saga",
				steps:   []saga.SagaStep{},
				timeout: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "Invalid timeout",
			sagaDef: &InventoryManagementSagaDefinition{
				id:      "test-saga",
				name:    "Test Saga",
				steps:   []saga.SagaStep{&CheckInventoryStep{}},
				timeout: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sagaDef.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ==========================
// 辅助函数测试
// ==========================

func TestGenerateReleaseID(t *testing.T) {
	id1 := generateReleaseID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateReleaseID()

	if id1 == id2 {
		t.Error("Expected different IDs for sequential calls")
	}

	if id1 == "" || id2 == "" {
		t.Error("Expected non-empty IDs")
	}
}

// ==========================
// 错误类型测试
// ==========================

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrInsufficientInventory", ErrInsufficientInventory},
		{"ErrReservationExpired", ErrReservationExpired},
		{"ErrWarehouseUnavailable", ErrWarehouseUnavailable},
		{"ErrInvalidAllocationPlan", ErrInvalidAllocationPlan},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("Expected %s to be defined", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("Expected %s to have a non-empty error message", tt.name)
			}
		})
	}
}

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

// Package examples provides example implementations of Saga patterns for common business scenarios.
// This file demonstrates a multi-warehouse inventory management Saga, showcasing how to handle
// complex distributed inventory coordination across multiple warehouses with proper compensation logic.
package examples

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// ==========================
// 数据结构定义
// ==========================

// InventoryRequestData 包含库存管理请求所需的所有输入数据
type InventoryRequestData struct {
	// 请求基本信息
	RequestID   string // 请求ID（如果已生成）
	OrderID     string // 关联的订单ID
	CustomerID  string // 客户ID
	RequestType string // 请求类型（reservation、allocation、transfer）

	// 商品信息
	Items []InventoryItem // 需要处理的商品项

	// 仓库偏好设置
	PreferredWarehouses []string // 优先选择的仓库列表
	AllocationStrategy  string   // 分配策略（优先级、最近仓库、负载均衡）

	// 优先级设置
	Priority  int  // 优先级（1-10，10最高）
	IsUrgent  bool // 是否紧急
	ExpiredAt time.Time

	// 元数据
	Metadata map[string]interface{} // 额外元数据
}

// InventoryItem 表示一个库存商品项
type InventoryItem struct {
	ProductID   string // 商品ID
	SKU         string // 库存单位
	Quantity    int    // 数量
	Unit        string // 单位（如 piece、box、kg）
	Description string // 描述
}

// WarehouseInfo 表示仓库信息
type WarehouseInfo struct {
	WarehouseID   string    // 仓库ID
	WarehouseName string    // 仓库名称
	Location      string    // 地理位置
	Capacity      int       // 总容量
	Available     int       // 可用容量
	Status        string    // 仓库状态（active、maintenance、closed）
	Priority      int       // 优先级
	LastUpdated   time.Time // 最后更新时间
}

// WarehouseInventory 表示仓库中的库存信息
type WarehouseInventory struct {
	WarehouseID   string // 仓库ID
	ProductID     string // 商品ID
	SKU           string // 库存单位
	Available     int    // 可用数量
	Reserved      int    // 已预留数量
	Total         int    // 总数量
	ReorderPoint  int    // 重新订货点
	ReorderQty    int    // 重新订货量
	SafetyStock   int    // 安全库存
	LastRestockAt time.Time
	NextRestockAt time.Time
	WarehouseZone string // 仓库区域
	ShelfLocation string // 货架位置
}

// CheckInventoryResult 存储库存检查步骤的结果
type CheckInventoryResult struct {
	RequestID           string                 // 请求ID
	AvailableWarehouses []WarehouseInfo        // 可用的仓库列表
	InventoryDetails    []WarehouseInventory   // 各仓库库存详情
	TotalAvailable      map[string]int         // 每个SKU的总可用数量
	CheckedAt           time.Time              // 检查时间
	CanFulfill          bool                   // 是否能够满足需求
	Shortages           []InventoryShortage    // 短缺信息
	Metadata            map[string]interface{} // 元数据
}

// InventoryShortage 表示库存短缺信息
type InventoryShortage struct {
	ProductID        string    // 商品ID
	SKU              string    // 库存单位
	RequestedQty     int       // 请求数量
	AvailableQty     int       // 可用数量
	ShortageQty      int       // 短缺数量
	EstimatedRestock time.Time // 预计补货时间
}

// ReserveInventoryResult 存储库存预留步骤的结果
type ReserveInventoryResult struct {
	ReservationID   string                  // 预留ID
	RequestID       string                  // 请求ID
	ReservedItems   []ReservedWarehouseItem // 预留的商品项（跨仓库）
	ReservationPlan []WarehouseAllocation   // 仓库分配计划
	ExpiresAt       time.Time               // 预留过期时间
	TotalReserved   int                     // 总预留数量
	Status          string                  // 预留状态
	Metadata        map[string]interface{}  // 元数据
}

// ReservedWarehouseItem 表示在特定仓库预留的商品项
type ReservedWarehouseItem struct {
	WarehouseID      string    // 仓库ID
	ProductID        string    // 商品ID
	SKU              string    // 库存单位
	ReservedQty      int       // 预留数量
	ReservationToken string    // 预留令牌（用于后续操作验证）
	ReservedAt       time.Time // 预留时间
	ExpiresAt        time.Time // 过期时间
}

// WarehouseAllocation 表示仓库分配计划
type WarehouseAllocation struct {
	WarehouseID   string          // 仓库ID
	WarehouseName string          // 仓库名称
	Items         []InventoryItem // 分配的商品项
	Priority      int             // 分配优先级
	EstimatedCost float64         // 预计成本
	Distance      float64         // 距离（如果适用）
}

// AllocateInventoryResult 存储库存分配/扣减步骤的结果
type AllocateInventoryResult struct {
	AllocationID   string                   // 分配ID
	RequestID      string                   // 请求ID
	ReservationID  string                   // 预留ID
	AllocatedItems []AllocatedWarehouseItem // 已分配的商品项
	AllocationPlan []WarehouseAllocation    // 实际执行的分配计划
	CompletedAt    time.Time                // 完成时间
	Status         string                   // 分配状态（completed、partial、failed）
	Metadata       map[string]interface{}   // 元数据
}

// AllocatedWarehouseItem 表示已分配的仓库商品项
type AllocatedWarehouseItem struct {
	WarehouseID     string    // 仓库ID
	ProductID       string    // 商品ID
	SKU             string    // 库存单位
	AllocatedQty    int       // 分配数量
	TransactionID   string    // 事务ID
	AllocatedAt     time.Time // 分配时间
	PickingLocation string    // 拣货位置
}

// ReleaseInventoryResult 存储库存释放步骤的结果
type ReleaseInventoryResult struct {
	ReleaseID     string                 // 释放ID
	RequestID     string                 // 请求ID
	ReservationID string                 // 预留ID
	ReleasedItems []ReleasedItem         // 释放的商品项
	ReleasedAt    time.Time              // 释放时间
	Reason        string                 // 释放原因
	Metadata      map[string]interface{} // 元数据
}

// ReleasedItem 表示已释放的商品项
type ReleasedItem struct {
	WarehouseID string // 仓库ID
	ProductID   string // 商品ID
	SKU         string // 库存单位
	ReleasedQty int    // 释放数量
}

// AuditInventoryResult 存储库存审计步骤的结果
type AuditInventoryResult struct {
	AuditID      string                 // 审计ID
	RequestID    string                 // 请求ID
	AuditType    string                 // 审计类型
	RecordedAt   time.Time              // 记录时间
	AuditDetails map[string]interface{} // 审计详情
	Status       string                 // 审计状态
	Metadata     map[string]interface{} // 元数据
}

// NotifyInventoryResult 存储库存通知步骤的结果
type NotifyInventoryResult struct {
	NotificationID string                 // 通知ID
	RequestID      string                 // 请求ID
	Recipients     []string               // 接收者列表
	Channel        string                 // 通知渠道
	SentAt         time.Time              // 发送时间
	Status         string                 // 通知状态
	Metadata       map[string]interface{} // 元数据
}

// ==========================
// 服务接口定义（用于模拟外部服务）
// ==========================

// MultiWarehouseInventoryServiceClient 多仓库库存服务客户端接口
type MultiWarehouseInventoryServiceClient interface {
	// CheckAvailability 检查库存可用性
	CheckAvailability(ctx context.Context, items []InventoryItem, preferredWarehouses []string) (*CheckInventoryResult, error)

	// ReserveInventory 预留库存（跨多个仓库）
	ReserveInventory(ctx context.Context, requestID string, items []InventoryItem, warehouses []string) (*ReserveInventoryResult, error)

	// AllocateInventory 分配/扣减库存（实际扣减）
	AllocateInventory(ctx context.Context, reservationID string) (*AllocateInventoryResult, error)

	// ReleaseReservation 释放预留的库存
	ReleaseReservation(ctx context.Context, reservationID string, reason string) error

	// RestoreInventory 恢复已分配的库存（补偿操作）
	RestoreInventory(ctx context.Context, allocationID string, items []AllocatedWarehouseItem) error

	// GetWarehouseInfo 获取仓库信息
	GetWarehouseInfo(ctx context.Context, warehouseID string) (*WarehouseInfo, error)

	// GetInventoryDetails 获取库存详情
	GetInventoryDetails(ctx context.Context, warehouseID string, sku string) (*WarehouseInventory, error)
}

// AuditInventoryServiceClient 库存审计服务客户端接口
type AuditInventoryServiceClient interface {
	// CreateAuditRecord 创建审计记录
	CreateAuditRecord(ctx context.Context, requestID string, operation string, details map[string]interface{}) (*AuditInventoryResult, error)

	// UpdateAuditRecord 更新审计记录
	UpdateAuditRecord(ctx context.Context, auditID string, status string, details map[string]interface{}) error
}

// NotificationInventoryServiceClient 通知服务客户端接口
type NotificationInventoryServiceClient interface {
	// SendInventoryNotification 发送库存通知
	SendInventoryNotification(ctx context.Context, requestID string, recipients []string, message string) (*NotifyInventoryResult, error)

	// SendInventoryAlert 发送库存告警
	SendInventoryAlert(ctx context.Context, requestID string, alertType string, details map[string]interface{}) error
}

// ==========================
// Saga 步骤实现
// ==========================

// CheckInventoryStep 检查库存步骤
// 此步骤检查各仓库的库存可用性，为后续分配提供决策依据
type CheckInventoryStep struct {
	service MultiWarehouseInventoryServiceClient
}

// GetID 返回步骤ID
func (s *CheckInventoryStep) GetID() string {
	return "check-inventory"
}

// GetName 返回步骤名称
func (s *CheckInventoryStep) GetName() string {
	return "检查库存"
}

// GetDescription 返回步骤描述
func (s *CheckInventoryStep) GetDescription() string {
	return "检查多个仓库的库存可用性，评估是否能够满足请求，并识别潜在的短缺"
}

// Execute 执行库存检查操作
// 输入：InventoryRequestData
// 输出：CheckInventoryResult
func (s *CheckInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	requestData, ok := data.(*InventoryRequestData)
	if !ok {
		return nil, errors.New("invalid data type: expected *InventoryRequestData")
	}

	// 验证请求数据
	if err := s.validateRequestData(requestData); err != nil {
		return nil, fmt.Errorf("库存请求数据验证失败: %w", err)
	}

	// 调用库存服务检查可用性
	result, err := s.service.CheckAvailability(ctx, requestData.Items, requestData.PreferredWarehouses)
	if err != nil {
		return nil, fmt.Errorf("检查库存可用性失败: %w", err)
	}

	// 设置元数据（重要：必须存储原始请求的商品项）
	result.Metadata = map[string]interface{}{
		"request_id":           requestData.RequestID,
		"order_id":             requestData.OrderID,
		"customer_id":          requestData.CustomerID,
		"request_type":         requestData.RequestType,
		"priority":             requestData.Priority,
		"is_urgent":            requestData.IsUrgent,
		"items_count":          len(requestData.Items),
		"requested_items":      requestData.Items, // 存储原始请求的商品项（带请求数量）
		"preferred_warehouses": requestData.PreferredWarehouses,
		"allocation_strategy":  requestData.AllocationStrategy,
	}

	// 检查是否能够满足请求
	if !result.CanFulfill {
		return result, fmt.Errorf("库存不足以满足请求: 存在 %d 个短缺项", len(result.Shortages))
	}

	return result, nil
}

// Compensate 补偿操作：库存检查不需要补偿
// 因为这一步骤只是读取操作，没有修改任何状态
func (s *CheckInventoryStep) Compensate(ctx context.Context, data interface{}) error {
	// 库存检查步骤不需要补偿，因为没有修改任何状态
	return nil
}

// GetTimeout 返回步骤超时时间
func (s *CheckInventoryStep) GetTimeout() time.Duration {
	return 15 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *CheckInventoryStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *CheckInventoryStep) IsRetryable(err error) bool {
	// 网络错误和临时服务不可用错误可重试
	// 业务错误（如库存不足）不可重试
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// 检查是否为库存不足错误（不可重试）
	if errors.Is(err, ErrInsufficientInventory) {
		return false
	}
	return true
}

// GetMetadata 返回步骤元数据
func (s *CheckInventoryStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "inventory_check",
		"service":   "inventory-service",
		"critical":  true, // 标记为关键步骤
		"read_only": true, // 只读操作
	}
}

// validateRequestData 验证请求数据的有效性
func (s *CheckInventoryStep) validateRequestData(data *InventoryRequestData) error {
	if data.RequestID == "" {
		return errors.New("请求ID不能为空")
	}
	if data.OrderID == "" {
		return errors.New("订单ID不能为空")
	}
	if data.CustomerID == "" {
		return errors.New("客户ID不能为空")
	}
	if len(data.Items) == 0 {
		return errors.New("商品项不能为空")
	}
	for i, item := range data.Items {
		if item.SKU == "" {
			return fmt.Errorf("商品项 %d 的SKU不能为空", i)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("商品项 %d 的数量必须大于0", i)
		}
	}
	return nil
}

// ReserveMultiWarehouseInventoryStep 跨仓库预留库存步骤
// 此步骤在多个仓库中预留库存，实现复杂的跨仓库协调
type ReserveMultiWarehouseInventoryStep struct {
	service MultiWarehouseInventoryServiceClient
}

// GetID 返回步骤ID
func (s *ReserveMultiWarehouseInventoryStep) GetID() string {
	return "reserve-multi-warehouse-inventory"
}

// GetName 返回步骤名称
func (s *ReserveMultiWarehouseInventoryStep) GetName() string {
	return "预留多仓库库存"
}

// GetDescription 返回步骤描述
func (s *ReserveMultiWarehouseInventoryStep) GetDescription() string {
	return "在多个仓库中协调预留库存，实现跨仓库的库存分配和预留"
}

// Execute 执行库存预留操作
// 输入：CheckInventoryResult（从上一步传递）
// 输出：ReserveInventoryResult
func (s *ReserveMultiWarehouseInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	checkResult, ok := data.(*CheckInventoryResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *CheckInventoryResult")
	}

	// 从元数据中获取请求信息
	requestID, _ := checkResult.Metadata["request_id"].(string)

	// 从元数据中获取原始请求的商品项（而不是总可用数量）
	items := s.buildInventoryItems(checkResult)

	// 选择仓库（基于可用性和优先级）
	warehouses := s.selectWarehouses(checkResult)

	// 调用库存服务预留库存
	result, err := s.service.ReserveInventory(ctx, requestID, items, warehouses)
	if err != nil {
		return nil, fmt.Errorf("预留多仓库库存失败: %w", err)
	}

	// 传递元数据
	result.Metadata = checkResult.Metadata

	return result, nil
}

// Compensate 补偿操作：释放预留的库存
func (s *ReserveMultiWarehouseInventoryStep) Compensate(ctx context.Context, data interface{}) error {
	result, ok := data.(*ReserveInventoryResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *ReserveInventoryResult")
	}

	// 释放预留的库存
	err := s.service.ReleaseReservation(ctx, result.ReservationID, "saga_compensation")
	if err != nil {
		// 库存释放失败，记录日志但不阻止补偿流程
		// 预留通常会有自动过期机制
		return fmt.Errorf("释放库存预留失败: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *ReserveMultiWarehouseInventoryStep) GetTimeout() time.Duration {
	return 30 * time.Second // 跨仓库操作可能需要更长时间
}

// GetRetryPolicy 返回步骤的重试策略
func (s *ReserveMultiWarehouseInventoryStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 15*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *ReserveMultiWarehouseInventoryStep) IsRetryable(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// 库存不足错误不可重试
	if errors.Is(err, ErrInsufficientInventory) {
		return false
	}
	return true
}

// GetMetadata 返回步骤元数据
func (s *ReserveMultiWarehouseInventoryStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type":      "multi_warehouse_reservation",
		"service":        "inventory-service",
		"critical":       true,
		"multi_resource": true, // 标记为多资源操作
	}
}

// buildInventoryItems 从检查结果的元数据中提取原始请求的商品项
// 注意：必须使用原始请求的数量，而不是总可用数量，以避免过度预留
func (s *ReserveMultiWarehouseInventoryStep) buildInventoryItems(checkResult *CheckInventoryResult) []InventoryItem {
	items := []InventoryItem{}
	
	// 从元数据中获取原始请求的商品项
	// 这些是在 CheckInventoryStep 中存储的原始请求数据
	if itemsData, ok := checkResult.Metadata["requested_items"].([]InventoryItem); ok {
		// 使用原始请求的数量，而不是可用数量
		items = itemsData
	} else {
		// 向后兼容：如果没有存储原始商品项，尝试从其他元数据重建
		// 但这种情况不应该发生，因为我们已经在 CheckInventoryStep 中存储了
		// 这里保留为空，让后续逻辑处理错误
	}
	
	return items
}

// selectWarehouses 选择仓库列表
func (s *ReserveMultiWarehouseInventoryStep) selectWarehouses(checkResult *CheckInventoryResult) []string {
	warehouses := []string{}
	for _, wh := range checkResult.AvailableWarehouses {
		if wh.Status == "active" {
			warehouses = append(warehouses, wh.WarehouseID)
		}
	}

	// 按优先级排序
	sort.Slice(warehouses, func(i, j int) bool {
		return checkResult.AvailableWarehouses[i].Priority > checkResult.AvailableWarehouses[j].Priority
	})

	return warehouses
}

// AllocateMultiWarehouseInventoryStep 跨仓库分配/扣减库存步骤
// 此步骤执行实际的库存扣减操作
type AllocateMultiWarehouseInventoryStep struct {
	service MultiWarehouseInventoryServiceClient
}

// GetID 返回步骤ID
func (s *AllocateMultiWarehouseInventoryStep) GetID() string {
	return "allocate-multi-warehouse-inventory"
}

// GetName 返回步骤名称
func (s *AllocateMultiWarehouseInventoryStep) GetName() string {
	return "分配多仓库库存"
}

// GetDescription 返回步骤描述
func (s *AllocateMultiWarehouseInventoryStep) GetDescription() string {
	return "在多个仓库中执行实际的库存扣减操作，完成库存分配"
}

// Execute 执行库存分配操作
// 输入：ReserveInventoryResult（从上一步传递）
// 输出：AllocateInventoryResult
func (s *AllocateMultiWarehouseInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	reserveResult, ok := data.(*ReserveInventoryResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *ReserveInventoryResult")
	}

	// 调用库存服务分配库存
	result, err := s.service.AllocateInventory(ctx, reserveResult.ReservationID)
	if err != nil {
		return nil, fmt.Errorf("分配多仓库库存失败: %w", err)
	}

	// 传递元数据
	result.Metadata = reserveResult.Metadata

	return result, nil
}

// Compensate 补偿操作：恢复已分配的库存
// 这是关键的补偿操作，必须确保库存完全恢复以保持一致性
func (s *AllocateMultiWarehouseInventoryStep) Compensate(ctx context.Context, data interface{}) error {
	result, ok := data.(*AllocateInventoryResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *AllocateInventoryResult")
	}

	// 只有成功分配的库存才需要恢复
	if result.Status != "completed" && result.Status != "partial" {
		return nil
	}

	// 恢复已分配的库存
	err := s.service.RestoreInventory(ctx, result.AllocationID, result.AllocatedItems)
	if err != nil {
		// 库存恢复失败是严重问题，需要人工介入
		return fmt.Errorf("恢复库存失败，需要人工处理: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *AllocateMultiWarehouseInventoryStep) GetTimeout() time.Duration {
	return 30 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *AllocateMultiWarehouseInventoryStep) GetRetryPolicy() saga.RetryPolicy {
	// 分配操作重试次数较少，避免重复扣减
	return saga.NewExponentialBackoffRetryPolicy(2, 2*time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *AllocateMultiWarehouseInventoryStep) IsRetryable(err error) bool {
	// 分配操作需要谨慎重试
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// 预留已过期等错误不重试
	if errors.Is(err, ErrReservationExpired) {
		return false
	}
	return false
}

// GetMetadata 返回步骤元数据
func (s *AllocateMultiWarehouseInventoryStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type":      "multi_warehouse_allocation",
		"service":        "inventory-service",
		"critical":       true, // 最关键的步骤
		"multi_resource": true,
	}
}

// ReleaseReservationStep 释放预留步骤
// 此步骤在分配成功后清理预留记录
type ReleaseReservationStep struct {
	service MultiWarehouseInventoryServiceClient
}

// GetID 返回步骤ID
func (s *ReleaseReservationStep) GetID() string {
	return "release-reservation"
}

// GetName 返回步骤名称
func (s *ReleaseReservationStep) GetName() string {
	return "释放预留"
}

// GetDescription 返回步骤描述
func (s *ReleaseReservationStep) GetDescription() string {
	return "清理预留记录，释放预留状态"
}

// Execute 执行释放预留操作
// 输入：AllocateInventoryResult（从上一步传递）
// 输出：ReleaseInventoryResult
func (s *ReleaseReservationStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	allocateResult, ok := data.(*AllocateInventoryResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *AllocateInventoryResult")
	}

	// 构建释放结果
	result := &ReleaseInventoryResult{
		ReleaseID:     generateReleaseID(),
		RequestID:     allocateResult.RequestID,
		ReservationID: allocateResult.ReservationID,
		ReleasedAt:    time.Now(),
		Reason:        "allocation_completed",
		Metadata:      allocateResult.Metadata,
	}

	// 构建释放项列表
	for _, item := range allocateResult.AllocatedItems {
		result.ReleasedItems = append(result.ReleasedItems, ReleasedItem{
			WarehouseID: item.WarehouseID,
			ProductID:   item.ProductID,
			SKU:         item.SKU,
			ReleasedQty: item.AllocatedQty,
		})
	}

	return result, nil
}

// Compensate 补偿操作：释放预留步骤通常不需要补偿
func (s *ReleaseReservationStep) Compensate(ctx context.Context, data interface{}) error {
	// 释放预留步骤不需要补偿
	return nil
}

// GetTimeout 返回步骤超时时间
func (s *ReleaseReservationStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *ReleaseReservationStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *ReleaseReservationStep) IsRetryable(err error) bool {
	return !errors.Is(err, context.DeadlineExceeded)
}

// GetMetadata 返回步骤元数据
func (s *ReleaseReservationStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "reservation_release",
		"service":   "inventory-service",
	}
}

// CreateInventoryAuditRecordStep 创建库存审计记录步骤
type CreateInventoryAuditRecordStep struct {
	service AuditInventoryServiceClient
}

// GetID 返回步骤ID
func (s *CreateInventoryAuditRecordStep) GetID() string {
	return "create-inventory-audit-record"
}

// GetName 返回步骤名称
func (s *CreateInventoryAuditRecordStep) GetName() string {
	return "创建库存审计记录"
}

// GetDescription 返回步骤描述
func (s *CreateInventoryAuditRecordStep) GetDescription() string {
	return "创建库存操作的审计记录，记录完整的库存变更过程"
}

// Execute 执行创建审计记录操作
// 输入：ReleaseInventoryResult（从上一步传递）
// 输出：AuditInventoryResult
func (s *CreateInventoryAuditRecordStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	releaseResult, ok := data.(*ReleaseInventoryResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *ReleaseInventoryResult")
	}

	// 从元数据中获取请求信息
	requestID, _ := releaseResult.Metadata["request_id"].(string)

	// 构建审计详情
	auditDetails := map[string]interface{}{
		"request_id":     requestID,
		"reservation_id": releaseResult.ReservationID,
		"released_items": releaseResult.ReleasedItems,
		"released_at":    releaseResult.ReleasedAt,
		"operation":      "inventory_allocation",
		"status":         "completed",
	}

	// 创建审计记录
	result, err := s.service.CreateAuditRecord(ctx, requestID, "multi_warehouse_allocation", auditDetails)
	if err != nil {
		return nil, fmt.Errorf("创建库存审计记录失败: %w", err)
	}

	result.Metadata = releaseResult.Metadata

	return result, nil
}

// Compensate 补偿操作：更新审计记录状态为已取消
func (s *CreateInventoryAuditRecordStep) Compensate(ctx context.Context, data interface{}) error {
	result, ok := data.(*AuditInventoryResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *AuditInventoryResult")
	}

	// 更新审计记录状态
	err := s.service.UpdateAuditRecord(ctx, result.AuditID, "cancelled", map[string]interface{}{
		"cancelled_reason": "saga_compensation",
		"cancelled_at":     time.Now(),
	})
	if err != nil {
		// 审计记录更新失败不应阻止补偿流程
		return fmt.Errorf("更新审计记录失败: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *CreateInventoryAuditRecordStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *CreateInventoryAuditRecordStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *CreateInventoryAuditRecordStep) IsRetryable(err error) bool {
	return !errors.Is(err, context.DeadlineExceeded)
}

// GetMetadata 返回步骤元数据
func (s *CreateInventoryAuditRecordStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "inventory_audit_record",
		"service":   "audit-service",
		"critical":  true,
	}
}

// SendInventoryNotificationStep 发送库存通知步骤
type SendInventoryNotificationStep struct {
	service NotificationInventoryServiceClient
}

// GetID 返回步骤ID
func (s *SendInventoryNotificationStep) GetID() string {
	return "send-inventory-notification"
}

// GetName 返回步骤名称
func (s *SendInventoryNotificationStep) GetName() string {
	return "发送库存通知"
}

// GetDescription 返回步骤描述
func (s *SendInventoryNotificationStep) GetDescription() string {
	return "向相关方发送库存操作完成通知"
}

// Execute 执行发送通知操作
// 输入：AuditInventoryResult（从上一步传递）
// 输出：NotifyInventoryResult
func (s *SendInventoryNotificationStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	auditResult, ok := data.(*AuditInventoryResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *AuditInventoryResult")
	}

	// 从元数据中获取请求信息
	requestID, _ := auditResult.Metadata["request_id"].(string)
	customerID, _ := auditResult.Metadata["customer_id"].(string)

	// 构建通知消息
	message := fmt.Sprintf("库存操作已完成，请求ID: %s", requestID)
	recipients := []string{customerID}

	// 发送通知
	result, err := s.service.SendInventoryNotification(ctx, requestID, recipients, message)
	if err != nil {
		// 通知失败不应导致整个 Saga 失败
		result = &NotifyInventoryResult{
			NotificationID: fmt.Sprintf("failed-%s-%d", requestID, time.Now().Unix()),
			RequestID:      requestID,
			Recipients:     recipients,
			Status:         "failed",
			Metadata: map[string]interface{}{
				"error":             err.Error(),
				"retry_recommended": true,
			},
		}
	}

	// 合并元数据
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	for k, v := range auditResult.Metadata {
		if _, exists := result.Metadata[k]; !exists {
			result.Metadata[k] = v
		}
	}

	return result, nil
}

// Compensate 补偿操作：发送取消通知
func (s *SendInventoryNotificationStep) Compensate(ctx context.Context, data interface{}) error {
	result, ok := data.(*NotifyInventoryResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *NotifyInventoryResult")
	}

	// 发送取消通知
	err := s.service.SendInventoryAlert(ctx, result.RequestID, "allocation_cancelled", map[string]interface{}{
		"reason": "saga_compensation",
	})
	if err != nil {
		// 通知失败不应阻止补偿流程
		return fmt.Errorf("发送取消通知失败: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *SendInventoryNotificationStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *SendInventoryNotificationStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *SendInventoryNotificationStep) IsRetryable(err error) bool {
	return !errors.Is(err, context.DeadlineExceeded)
}

// GetMetadata 返回步骤元数据
func (s *SendInventoryNotificationStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "inventory_notification",
		"service":   "notification-service",
		"critical":  false, // 通知不是关键步骤
	}
}

// ==========================
// Saga 定义
// ==========================

// InventoryManagementSagaDefinition 库存管理 Saga 定义
type InventoryManagementSagaDefinition struct {
	id                   string
	name                 string
	description          string
	steps                []saga.SagaStep
	timeout              time.Duration
	retryPolicy          saga.RetryPolicy
	compensationStrategy saga.CompensationStrategy
	metadata             map[string]interface{}
}

// GetID 返回 Saga 定义ID
func (d *InventoryManagementSagaDefinition) GetID() string {
	return d.id
}

// GetName 返回 Saga 名称
func (d *InventoryManagementSagaDefinition) GetName() string {
	return d.name
}

// GetDescription 返回 Saga 描述
func (d *InventoryManagementSagaDefinition) GetDescription() string {
	return d.description
}

// GetSteps 返回所有步骤
func (d *InventoryManagementSagaDefinition) GetSteps() []saga.SagaStep {
	return d.steps
}

// GetTimeout 返回超时时间
func (d *InventoryManagementSagaDefinition) GetTimeout() time.Duration {
	return d.timeout
}

// GetRetryPolicy 返回重试策略
func (d *InventoryManagementSagaDefinition) GetRetryPolicy() saga.RetryPolicy {
	return d.retryPolicy
}

// GetCompensationStrategy 返回补偿策略
func (d *InventoryManagementSagaDefinition) GetCompensationStrategy() saga.CompensationStrategy {
	return d.compensationStrategy
}

// GetMetadata 返回元数据
func (d *InventoryManagementSagaDefinition) GetMetadata() map[string]interface{} {
	return d.metadata
}

// Validate 验证 Saga 定义的有效性
func (d *InventoryManagementSagaDefinition) Validate() error {
	if d.id == "" {
		return errors.New("saga ID 不能为空")
	}
	if d.name == "" {
		return errors.New("saga 名称不能为空")
	}
	if len(d.steps) == 0 {
		return errors.New("至少需要一个步骤")
	}
	if d.timeout <= 0 {
		return errors.New("超时时间必须大于0")
	}
	return nil
}

// ==========================
// 工厂函数
// ==========================

// NewInventoryManagementSaga 创建库存管理 Saga 定义
func NewInventoryManagementSaga(
	inventoryService MultiWarehouseInventoryServiceClient,
	auditService AuditInventoryServiceClient,
	notificationService NotificationInventoryServiceClient,
) *InventoryManagementSagaDefinition {
	// 创建步骤
	steps := []saga.SagaStep{
		&CheckInventoryStep{service: inventoryService},
		&ReserveMultiWarehouseInventoryStep{service: inventoryService},
		&AllocateMultiWarehouseInventoryStep{service: inventoryService},
		&ReleaseReservationStep{service: inventoryService},
		&CreateInventoryAuditRecordStep{service: auditService},
		&SendInventoryNotificationStep{service: notificationService},
	}

	return &InventoryManagementSagaDefinition{
		id:          "inventory-management-saga",
		name:        "库存管理Saga",
		description: "处理多仓库库存协调的完整流程：检查库存→预留库存→分配库存→释放预留→审计记录→发送通知",
		steps:       steps,
		timeout:     5 * time.Minute, // 整个流程5分钟超时
		retryPolicy: saga.NewExponentialBackoffRetryPolicy(
			3,              // 最多重试3次
			1*time.Second,  // 初始延迟1秒
			30*time.Second, // 最大延迟30秒
		),
		compensationStrategy: saga.NewSequentialCompensationStrategy(
			3 * time.Minute, // 补偿超时3分钟
		),
		metadata: map[string]interface{}{
			"saga_type":       "inventory_management",
			"business_domain": "warehouse",
			"version":         "1.0.0",
			"multi_warehouse": true, // 标记为多仓库场景
			"critical":        true,
		},
	}
}

// ==========================
// 错误定义
// ==========================

var (
	// ErrInsufficientInventory 库存不足错误
	ErrInsufficientInventory = errors.New("insufficient inventory")

	// ErrReservationExpired 预留已过期错误
	ErrReservationExpired = errors.New("reservation expired")

	// ErrWarehouseUnavailable 仓库不可用错误
	ErrWarehouseUnavailable = errors.New("warehouse unavailable")

	// ErrInvalidAllocationPlan 无效的分配计划错误
	ErrInvalidAllocationPlan = errors.New("invalid allocation plan")
)

// ==========================
// 辅助函数
// ==========================

// generateReleaseID 生成释放ID
func generateReleaseID() string {
	return fmt.Sprintf("REL-%d", time.Now().UnixNano())
}

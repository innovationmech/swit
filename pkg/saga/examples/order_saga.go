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
// This file demonstrates an e-commerce order processing Saga with inventory reservation,
// payment processing, and order confirmation steps.
package examples

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/saga"
)

// ==========================
// 数据结构定义
// ==========================

// OrderData 包含订单处理所需的所有输入数据
type OrderData struct {
	// 订单基本信息
	OrderID     string  // 订单ID（如果已生成）
	CustomerID  string  // 客户ID
	TotalAmount float64 // 订单总金额

	// 订单项列表
	Items []OrderItem // 订单项

	// 配送地址
	Address ShippingAddress // 配送地址

	// 支付信息
	PaymentMethod string // 支付方式
	PaymentToken  string // 支付令牌

	// 元数据
	Metadata map[string]interface{} // 额外元数据
}

// OrderItem 表示订单中的一个商品项
type OrderItem struct {
	ProductID string  // 商品ID
	SKU       string  // 库存单位
	Quantity  int     // 数量
	UnitPrice float64 // 单价
	Total     float64 // 小计
}

// ShippingAddress 表示配送地址
type ShippingAddress struct {
	RecipientName string // 收件人姓名
	Phone         string // 联系电话
	Province      string // 省份
	City          string // 城市
	District      string // 区/县
	Street        string // 街道地址
	Zipcode       string // 邮编
}

// OrderStepResult 存储订单创建步骤的结果
type OrderStepResult struct {
	OrderID     string                 // 订单ID
	OrderNumber string                 // 订单号（用于客户查看）
	Status      string                 // 订单状态
	CreatedAt   time.Time              // 创建时间
	Items       []OrderItem            // 订单项
	Metadata    map[string]interface{} // 元数据
}

// InventoryStepResult 存储库存预留步骤的结果
type InventoryStepResult struct {
	ReservationID string                 // 预留ID
	ReservedItems []ReservedItem         // 预留的商品项
	ExpiresAt     time.Time              // 预留过期时间
	WarehouseID   string                 // 仓库ID
	Metadata      map[string]interface{} // 元数据
}

// ReservedItem 表示一个被预留的商品项
type ReservedItem struct {
	ProductID   string // 商品ID
	SKU         string // 库存单位
	Quantity    int    // 预留数量
	WarehouseID string // 仓库ID
}

// PaymentStepResult 存储支付处理步骤的结果
type PaymentStepResult struct {
	TransactionID string                 // 交易ID
	PaymentID     string                 // 支付ID
	Amount        float64                // 支付金额
	Currency      string                 // 货币类型
	Status        string                 // 支付状态
	PaidAt        time.Time              // 支付时间
	Provider      string                 // 支付提供商
	Metadata      map[string]interface{} // 元数据
}

// ConfirmStepResult 存储订单确认步骤的结果
type ConfirmStepResult struct {
	OrderID       string                 // 订单ID
	ConfirmedAt   time.Time              // 确认时间
	Status        string                 // 最终状态
	EstimatedShip time.Time              // 预计发货时间
	TrackingInfo  string                 // 跟踪信息
	Metadata      map[string]interface{} // 元数据
}

// ==========================
// 服务接口定义（用于模拟外部服务）
// ==========================

// OrderServiceClient 订单服务客户端接口
type OrderServiceClient interface {
	// CreateOrder 创建订单
	CreateOrder(ctx context.Context, data *OrderData) (*OrderStepResult, error)
	// CancelOrder 取消订单
	CancelOrder(ctx context.Context, orderID string, reason string) error
	// ConfirmOrder 确认订单
	ConfirmOrder(ctx context.Context, orderID string) (*ConfirmStepResult, error)
}

// InventoryServiceClient 库存服务客户端接口
type InventoryServiceClient interface {
	// ReserveInventory 预留库存
	ReserveInventory(ctx context.Context, orderID string, items []OrderItem) (*InventoryStepResult, error)
	// ReleaseInventory 释放库存
	ReleaseInventory(ctx context.Context, reservationID string) error
}

// PaymentServiceClient 支付服务客户端接口
type PaymentServiceClient interface {
	// ProcessPayment 处理支付
	ProcessPayment(ctx context.Context, orderID string, amount float64, method string, token string) (*PaymentStepResult, error)
	// RefundPayment 退款
	RefundPayment(ctx context.Context, transactionID string, reason string) error
}

// ==========================
// Saga 步骤实现
// ==========================

// CreateOrderStep 创建订单步骤
type CreateOrderStep struct {
	service OrderServiceClient
}

// GetID 返回步骤ID
func (s *CreateOrderStep) GetID() string {
	return "create-order"
}

// GetName 返回步骤名称
func (s *CreateOrderStep) GetName() string {
	return "创建订单"
}

// GetDescription 返回步骤描述
func (s *CreateOrderStep) GetDescription() string {
	return "在订单系统中创建新订单记录，初始状态为待支付"
}

// Execute 执行创建订单操作
// 输入：OrderData
// 输出：OrderStepResult
func (s *CreateOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	orderData, ok := data.(*OrderData)
	if !ok {
		return nil, errors.New("invalid data type: expected *OrderData")
	}

	// 验证订单数据
	if err := s.validateOrderData(orderData); err != nil {
		return nil, fmt.Errorf("订单数据验证失败: %w", err)
	}

	// 调用订单服务创建订单
	result, err := s.service.CreateOrder(ctx, orderData)
	if err != nil {
		return nil, fmt.Errorf("创建订单失败: %w", err)
	}

	return result, nil
}

// Compensate 补偿操作：取消订单
func (s *CreateOrderStep) Compensate(ctx context.Context, data interface{}) error {
	result, ok := data.(*OrderStepResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *OrderStepResult")
	}

	// 调用订单服务取消订单
	err := s.service.CancelOrder(ctx, result.OrderID, "saga_compensation")
	if err != nil {
		// 订单取消失败，记录日志但不阻止补偿流程
		// 在实际生产环境中，可能需要人工介入
		return fmt.Errorf("取消订单失败: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *CreateOrderStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *CreateOrderStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *CreateOrderStep) IsRetryable(err error) bool {
	// 网络错误和临时服务不可用错误可重试
	// 验证错误和业务规则错误不可重试
	if errors.Is(err, context.DeadlineExceeded) {
		return false // 超时不重试
	}
	return true
}

// GetMetadata 返回步骤元数据
func (s *CreateOrderStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "order_creation",
		"service":   "order-service",
	}
}

// validateOrderData 验证订单数据的有效性
func (s *CreateOrderStep) validateOrderData(data *OrderData) error {
	if data.CustomerID == "" {
		return errors.New("客户ID不能为空")
	}
	if len(data.Items) == 0 {
		return errors.New("订单项不能为空")
	}
	if data.TotalAmount <= 0 {
		return errors.New("订单总金额必须大于0")
	}
	if data.Address.RecipientName == "" {
		return errors.New("收件人姓名不能为空")
	}
	if data.Address.Phone == "" {
		return errors.New("联系电话不能为空")
	}
	return nil
}

// ReserveInventoryStep 预留库存步骤
type ReserveInventoryStep struct {
	service InventoryServiceClient
}

// GetID 返回步骤ID
func (s *ReserveInventoryStep) GetID() string {
	return "reserve-inventory"
}

// GetName 返回步骤名称
func (s *ReserveInventoryStep) GetName() string {
	return "预留库存"
}

// GetDescription 返回步骤描述
func (s *ReserveInventoryStep) GetDescription() string {
	return "在库存系统中预留订单所需的商品库存，防止超卖"
}

// Execute 执行库存预留操作
// 输入：OrderStepResult（从上一步传递）
// 输出：InventoryStepResult
func (s *ReserveInventoryStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	orderResult, ok := data.(*OrderStepResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *OrderStepResult")
	}

	// 调用库存服务预留库存
	result, err := s.service.ReserveInventory(ctx, orderResult.OrderID, orderResult.Items)
	if err != nil {
		return nil, fmt.Errorf("预留库存失败: %w", err)
	}

	return result, nil
}

// Compensate 补偿操作：释放库存
func (s *ReserveInventoryStep) Compensate(ctx context.Context, data interface{}) error {
	result, ok := data.(*InventoryStepResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *InventoryStepResult")
	}

	// 调用库存服务释放预留的库存
	err := s.service.ReleaseInventory(ctx, result.ReservationID)
	if err != nil {
		// 库存释放失败，记录日志
		// 库存通常会有自动过期机制，所以补偿失败不阻止流程
		return fmt.Errorf("释放库存失败: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *ReserveInventoryStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *ReserveInventoryStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *ReserveInventoryStep) IsRetryable(err error) bool {
	// 网络错误可重试
	// 库存不足错误不可重试
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// GetMetadata 返回步骤元数据
func (s *ReserveInventoryStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "inventory_reservation",
		"service":   "inventory-service",
	}
}

// ProcessPaymentStep 处理支付步骤
type ProcessPaymentStep struct {
	service PaymentServiceClient
}

// GetID 返回步骤ID
func (s *ProcessPaymentStep) GetID() string {
	return "process-payment"
}

// GetName 返回步骤名称
func (s *ProcessPaymentStep) GetName() string {
	return "处理支付"
}

// GetDescription 返回步骤描述
func (s *ProcessPaymentStep) GetDescription() string {
	return "通过支付网关处理订单支付，扣除客户账户资金"
}

// Execute 执行支付处理操作
// 输入：InventoryStepResult（从上一步传递）
// 输出：PaymentStepResult
// 注意：这里需要访问原始 OrderData 获取支付信息，实际实现时可能需要调整数据传递方式
func (s *ProcessPaymentStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	inventoryResult, ok := data.(*InventoryStepResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *InventoryStepResult")
	}

	// 从元数据中获取订单信息
	orderID, ok := inventoryResult.Metadata["order_id"].(string)
	if !ok || orderID == "" {
		return nil, errors.New("order_id not found in metadata")
	}

	amount, ok := inventoryResult.Metadata["amount"].(float64)
	if !ok || amount <= 0 {
		return nil, errors.New("invalid amount in metadata")
	}

	method, _ := inventoryResult.Metadata["payment_method"].(string)
	token, _ := inventoryResult.Metadata["payment_token"].(string)

	// 调用支付服务处理支付
	result, err := s.service.ProcessPayment(ctx, orderID, amount, method, token)
	if err != nil {
		return nil, fmt.Errorf("处理支付失败: %w", err)
	}

	return result, nil
}

// Compensate 补偿操作：退款
func (s *ProcessPaymentStep) Compensate(ctx context.Context, data interface{}) error {
	result, ok := data.(*PaymentStepResult)
	if !ok {
		return errors.New("invalid compensation data type: expected *PaymentStepResult")
	}

	// 调用支付服务执行退款
	err := s.service.RefundPayment(ctx, result.TransactionID, "saga_compensation")
	if err != nil {
		// 退款失败需要人工介入，这是关键操作
		return fmt.Errorf("退款失败，需要人工处理: %w", err)
	}

	return nil
}

// GetTimeout 返回步骤超时时间
func (s *ProcessPaymentStep) GetTimeout() time.Duration {
	return 30 * time.Second // 支付操作可能需要更长时间
}

// GetRetryPolicy 返回步骤的重试策略
func (s *ProcessPaymentStep) GetRetryPolicy() saga.RetryPolicy {
	// 支付操作重试次数较少，避免重复扣款
	return saga.NewExponentialBackoffRetryPolicy(2, 2*time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *ProcessPaymentStep) IsRetryable(err error) bool {
	// 支付操作需要谨慎重试
	// 只有明确的网络临时错误才重试
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// 余额不足等业务错误不重试
	return false
}

// GetMetadata 返回步骤元数据
func (s *ProcessPaymentStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "payment_processing",
		"service":   "payment-service",
		"critical":  true, // 标记为关键步骤
	}
}

// ConfirmOrderStep 确认订单步骤
type ConfirmOrderStep struct {
	service OrderServiceClient
}

// GetID 返回步骤ID
func (s *ConfirmOrderStep) GetID() string {
	return "confirm-order"
}

// GetName 返回步骤名称
func (s *ConfirmOrderStep) GetName() string {
	return "确认订单"
}

// GetDescription 返回步骤描述
func (s *ConfirmOrderStep) GetDescription() string {
	return "确认订单已支付，更新订单状态为待发货，准备物流信息"
}

// Execute 执行订单确认操作
// 输入：PaymentStepResult（从上一步传递）
// 输出：ConfirmStepResult
func (s *ConfirmOrderStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	paymentResult, ok := data.(*PaymentStepResult)
	if !ok {
		return nil, errors.New("invalid data type: expected *PaymentStepResult")
	}

	// 从元数据中获取订单ID
	orderID, ok := paymentResult.Metadata["order_id"].(string)
	if !ok || orderID == "" {
		return nil, errors.New("order_id not found in payment result metadata")
	}

	// 调用订单服务确认订单
	result, err := s.service.ConfirmOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("确认订单失败: %w", err)
	}

	return result, nil
}

// Compensate 补偿操作：将订单状态回退
// 注意：此步骤通常不需要补偿，因为前面的步骤已经处理了回滚
// 但为了完整性，这里实现一个空的补偿方法
func (s *ConfirmOrderStep) Compensate(ctx context.Context, data interface{}) error {
	// 订单确认步骤通常不需要单独补偿
	// 因为订单取消（在 CreateOrderStep 的补偿中执行）已经处理了状态回退
	return nil
}

// GetTimeout 返回步骤超时时间
func (s *ConfirmOrderStep) GetTimeout() time.Duration {
	return 10 * time.Second
}

// GetRetryPolicy 返回步骤的重试策略
func (s *ConfirmOrderStep) GetRetryPolicy() saga.RetryPolicy {
	return saga.NewExponentialBackoffRetryPolicy(3, time.Second, 10*time.Second)
}

// IsRetryable 判断错误是否可重试
func (s *ConfirmOrderStep) IsRetryable(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// GetMetadata 返回步骤元数据
func (s *ConfirmOrderStep) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"step_type": "order_confirmation",
		"service":   "order-service",
	}
}

// ==========================
// Saga 定义
// ==========================

// OrderProcessingSagaDefinition 订单处理 Saga 定义
type OrderProcessingSagaDefinition struct {
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
func (d *OrderProcessingSagaDefinition) GetID() string {
	return d.id
}

// GetName 返回 Saga 名称
func (d *OrderProcessingSagaDefinition) GetName() string {
	return d.name
}

// GetDescription 返回 Saga 描述
func (d *OrderProcessingSagaDefinition) GetDescription() string {
	return d.description
}

// GetSteps 返回所有步骤
func (d *OrderProcessingSagaDefinition) GetSteps() []saga.SagaStep {
	return d.steps
}

// GetTimeout 返回超时时间
func (d *OrderProcessingSagaDefinition) GetTimeout() time.Duration {
	return d.timeout
}

// GetRetryPolicy 返回重试策略
func (d *OrderProcessingSagaDefinition) GetRetryPolicy() saga.RetryPolicy {
	return d.retryPolicy
}

// GetCompensationStrategy 返回补偿策略
func (d *OrderProcessingSagaDefinition) GetCompensationStrategy() saga.CompensationStrategy {
	return d.compensationStrategy
}

// GetMetadata 返回元数据
func (d *OrderProcessingSagaDefinition) GetMetadata() map[string]interface{} {
	return d.metadata
}

// Validate 验证 Saga 定义的有效性
func (d *OrderProcessingSagaDefinition) Validate() error {
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

// NewOrderProcessingSaga 创建订单处理 Saga 定义
func NewOrderProcessingSaga(
	orderService OrderServiceClient,
	inventoryService InventoryServiceClient,
	paymentService PaymentServiceClient,
) *OrderProcessingSagaDefinition {
	// 创建步骤
	steps := []saga.SagaStep{
		&CreateOrderStep{service: orderService},
		&ReserveInventoryStep{service: inventoryService},
		&ProcessPaymentStep{service: paymentService},
		&ConfirmOrderStep{service: orderService},
	}

	return &OrderProcessingSagaDefinition{
		id:          "order-processing-saga",
		name:        "订单处理Saga",
		description: "处理电商订单的完整流程：创建订单→预留库存→处理支付→确认订单",
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
			"saga_type":       "order_processing",
			"business_domain": "e-commerce",
			"version":         "1.0.0",
		},
	}
}

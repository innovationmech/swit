package model

import (
	"time"

	"gorm.io/gorm"
)

// Order represents an order entity in the database
type Order struct {
	ID                   string               `gorm:"primaryKey;type:varchar(36)" json:"id"`
	CustomerID           string               `gorm:"not null;type:varchar(36);index" json:"customer_id"`
	ProductID            string               `gorm:"not null;type:varchar(36);index" json:"product_id"`
	Quantity             int32                `gorm:"not null" json:"quantity"`
	Amount               float64              `gorm:"not null;type:decimal(10,2)" json:"amount"`
	Status               OrderStatus          `gorm:"not null;type:varchar(20);index" json:"status"`
	PaymentTransactionID *string              `gorm:"type:varchar(36)" json:"payment_transaction_id,omitempty"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
	DeletedAt            gorm.DeletedAt       `gorm:"index" json:"-"`
	StatusHistory        []OrderStatusHistory `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"status_history,omitempty"`
}

// OrderStatus represents the possible order statuses
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusConfirmed  OrderStatus = "confirmed"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusFailed     OrderStatus = "failed"
	OrderStatusRefunded   OrderStatus = "refunded"
)

// IsValidStatus checks if the status is valid
func (s OrderStatus) IsValidStatus() bool {
	switch s {
	case OrderStatusPending, OrderStatusProcessing, OrderStatusConfirmed,
		OrderStatusShipped, OrderStatusDelivered, OrderStatusCancelled,
		OrderStatusFailed, OrderStatusRefunded:
		return true
	}
	return false
}

// String returns string representation of the status
func (s OrderStatus) String() string {
	return string(s)
}

// OrderStatusHistory tracks the history of order status changes
type OrderStatusHistory struct {
	ID         uint         `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID    string       `gorm:"not null;type:varchar(36);index" json:"order_id"`
	FromStatus *OrderStatus `gorm:"type:varchar(20)" json:"from_status"`
	ToStatus   OrderStatus  `gorm:"not null;type:varchar(20)" json:"to_status"`
	Reason     string       `gorm:"type:text" json:"reason,omitempty"`
	ChangedBy  string       `gorm:"type:varchar(100)" json:"changed_by,omitempty"`
	ChangedAt  time.Time    `json:"changed_at"`
}

// CreateOrderRequest represents a request to create an order
type CreateOrderRequest struct {
	CustomerID string  `json:"customer_id" binding:"required"`
	ProductID  string  `json:"product_id" binding:"required"`
	Quantity   int32   `json:"quantity" binding:"required,min=1"`
	Amount     float64 `json:"amount" binding:"required,min=0"`
}

// CreateOrderResponse represents the response to order creation
type CreateOrderResponse struct {
	Order *Order `json:"order"`
}

// GetOrderResponse represents the response to get order request
type GetOrderResponse struct {
	Order *Order `json:"order"`
}

// ListOrdersRequest represents a request to list orders
type ListOrdersRequest struct {
	CustomerID string      `form:"customer_id"`
	Status     OrderStatus `form:"status"`
	PageSize   int         `form:"page_size"`
	PageToken  string      `form:"page_token"`
}

// ListOrdersResponse represents the response to list orders
type ListOrdersResponse struct {
	Orders        []*Order `json:"orders"`
	NextPageToken string   `json:"next_page_token,omitempty"`
	TotalCount    int64    `json:"total_count"`
}

// UpdateOrderStatusRequest represents a request to update order status
type UpdateOrderStatusRequest struct {
	Status OrderStatus `json:"status" binding:"required"`
	Reason string      `json:"reason"`
}

// UpdateOrderStatusResponse represents the response to status update
type UpdateOrderStatusResponse struct {
	Order *Order `json:"order"`
}

// PaymentRequest represents a request to process payment
type PaymentRequest struct {
	CustomerID string  `json:"customer_id"`
	OrderID    string  `json:"order_id"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
}

// PaymentResponse represents a response from payment service
type PaymentResponse struct {
	TransactionID string        `json:"transaction_id"`
	Status        PaymentStatus `json:"status"`
	Message       string        `json:"message,omitempty"`
}

// PaymentStatus represents payment status
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

// InventoryRequest represents a request to check/reserve inventory
type InventoryRequest struct {
	ProductID  string `json:"product_id"`
	Quantity   int32  `json:"quantity"`
	OrderID    string `json:"order_id,omitempty"`
	CustomerID string `json:"customer_id,omitempty"`
}

// InventoryResponse represents a response from inventory service
type InventoryResponse struct {
	Available         bool   `json:"available"`
	AvailableQuantity int32  `json:"available_quantity"`
	ReservationID     string `json:"reservation_id,omitempty"`
	Message           string `json:"message,omitempty"`
}

// TableName returns the table name for Order
func (Order) TableName() string {
	return "orders"
}

// TableName returns the table name for OrderStatusHistory
func (OrderStatusHistory) TableName() string {
	return "order_status_history"
}

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

package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/config"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/model"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/tracing"
)

// OrderRepository handles database operations for orders
type OrderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(cfg config.DatabaseConfig) (*OrderRepository, error) {
	// Ensure data directory exists
	dataDir := filepath.Dir(cfg.DSN)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database connection
	db, err := gorm.Open(sqlite.Open(cfg.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	repo := &OrderRepository{db: db}

	// Run migrations
	if err := repo.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	return repo, nil
}

// migrate runs database migrations
func (r *OrderRepository) migrate() error {
	return r.db.AutoMigrate(&model.Order{}, &model.OrderStatusHistory{})
}

// CreateOrder creates a new order
func (r *OrderRepository) CreateOrder(ctx context.Context, order *model.Order) error {
	ctx, span := tracing.StartSpan(ctx, "repository.CreateOrder",
		attribute.String("order.id", order.ID),
		attribute.String("order.customer_id", order.CustomerID),
		attribute.String("order.product_id", order.ProductID),
	)
	defer span.End()

	// Start transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		tracing.SetSpanError(span, tx.Error)
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Create order
	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		tracing.SetSpanError(span, err)
		return fmt.Errorf("failed to create order: %w", err)
	}

	// Create initial status history entry
	history := &model.OrderStatusHistory{
		OrderID:    order.ID,
		FromStatus: nil,
		ToStatus:   order.Status,
		Reason:     "Order created",
		ChangedBy:  "system",
		ChangedAt:  time.Now(),
	}

	if err := tx.Create(history).Error; err != nil {
		tx.Rollback()
		tracing.SetSpanError(span, err)
		return fmt.Errorf("failed to create status history: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tracing.SetSpanError(span, err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tracing.AddEvent(span, "order.created",
		attribute.String("order.id", order.ID),
		attribute.String("order.status", string(order.Status)),
	)
	tracing.SetSpanSuccess(span)

	return nil
}

// GetOrderByID retrieves an order by its ID
func (r *OrderRepository) GetOrderByID(ctx context.Context, orderID string) (*model.Order, error) {
	ctx, span := tracing.StartSpan(ctx, "repository.GetOrderByID",
		attribute.String("order.id", orderID),
	)
	defer span.End()

	var order model.Order
	err := r.db.WithContext(ctx).
		Preload("StatusHistory").
		Where("id = ?", orderID).
		First(&order).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			tracing.AddEvent(span, "order.not_found",
				attribute.String("order.id", orderID),
			)
			tracing.SetSpanSuccess(span)
			return nil, nil // Return nil for not found instead of error
		}
		tracing.SetSpanError(span, err)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	tracing.AddEvent(span, "order.found",
		attribute.String("order.id", order.ID),
		attribute.String("order.status", string(order.Status)),
	)
	tracing.SetSpanSuccess(span)

	return &order, nil
}

// ListOrders retrieves orders with filtering and pagination
func (r *OrderRepository) ListOrders(ctx context.Context, customerID string, status model.OrderStatus, limit, offset int) ([]*model.Order, int64, error) {
	ctx, span := tracing.StartSpan(ctx, "repository.ListOrders",
		attribute.String("customer.id", customerID),
		attribute.String("filter.status", string(status)),
		attribute.Int("pagination.limit", limit),
		attribute.Int("pagination.offset", offset),
	)
	defer span.End()

	query := r.db.WithContext(ctx).Model(&model.Order{})

	// Apply filters
	if customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}
	if status != "" && status.IsValidStatus() {
		query = query.Where("status = ?", status)
	}

	// Get total count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		tracing.SetSpanError(span, err)
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	// Get orders with pagination
	var orders []*model.Order
	err := query.
		Preload("StatusHistory").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error

	if err != nil {
		tracing.SetSpanError(span, err)
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}

	tracing.AddEvent(span, "orders.listed",
		attribute.Int("orders.count", len(orders)),
		attribute.Int64("orders.total", totalCount),
	)
	tracing.SetSpanSuccess(span)

	return orders, totalCount, nil
}

// UpdateOrderStatus updates the status of an order
func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, orderID string, newStatus model.OrderStatus, reason, changedBy string) (*model.Order, error) {
	ctx, span := tracing.StartSpan(ctx, "repository.UpdateOrderStatus",
		attribute.String("order.id", orderID),
		attribute.String("order.new_status", string(newStatus)),
		attribute.String("order.reason", reason),
	)
	defer span.End()

	// Start transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		tracing.SetSpanError(span, tx.Error)
		return nil, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Get current order
	var order model.Order
	if err := tx.Where("id = ?", orderID).First(&order).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			tracing.AddEvent(span, "order.not_found",
				attribute.String("order.id", orderID),
			)
			tracing.SetSpanSuccess(span)
			return nil, fmt.Errorf("order not found")
		}
		tracing.SetSpanError(span, err)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Store old status for history
	oldStatus := order.Status

	// Update order status
	order.Status = newStatus
	order.UpdatedAt = time.Now()

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		tracing.SetSpanError(span, err)
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	// Create status history entry
	history := &model.OrderStatusHistory{
		OrderID:    orderID,
		FromStatus: &oldStatus,
		ToStatus:   newStatus,
		Reason:     reason,
		ChangedBy:  changedBy,
		ChangedAt:  time.Now(),
	}

	if err := tx.Create(history).Error; err != nil {
		tx.Rollback()
		tracing.SetSpanError(span, err)
		return nil, fmt.Errorf("failed to create status history: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tracing.SetSpanError(span, err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Load updated order with status history
	if err := r.db.WithContext(ctx).Preload("StatusHistory").Where("id = ?", orderID).First(&order).Error; err != nil {
		tracing.SetSpanError(span, err)
		return nil, fmt.Errorf("failed to reload order: %w", err)
	}

	tracing.AddEvent(span, "order.status_updated",
		attribute.String("order.id", orderID),
		attribute.String("order.from_status", string(oldStatus)),
		attribute.String("order.to_status", string(newStatus)),
	)
	tracing.SetSpanSuccess(span)

	return &order, nil
}

// UpdatePaymentTransactionID updates the payment transaction ID for an order
func (r *OrderRepository) UpdatePaymentTransactionID(ctx context.Context, orderID, transactionID string) error {
	ctx, span := tracing.StartSpan(ctx, "repository.UpdatePaymentTransactionID",
		attribute.String("order.id", orderID),
		attribute.String("payment.transaction_id", transactionID),
	)
	defer span.End()

	err := r.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ?", orderID).
		Update("payment_transaction_id", transactionID).Error

	if err != nil {
		tracing.SetSpanError(span, err)
		return fmt.Errorf("failed to update payment transaction ID: %w", err)
	}

	tracing.AddEvent(span, "order.payment_updated",
		attribute.String("order.id", orderID),
		attribute.String("payment.transaction_id", transactionID),
	)
	tracing.SetSpanSuccess(span)

	return nil
}

// HealthCheck checks if the database connection is healthy
func (r *OrderRepository) HealthCheck(ctx context.Context) error {
	ctx, span := tracing.StartSpan(ctx, "repository.HealthCheck")
	defer span.End()

	sqlDB, err := r.db.DB()
	if err != nil {
		tracing.SetSpanError(span, err)
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		tracing.SetSpanError(span, err)
		return fmt.Errorf("database ping failed: %w", err)
	}

	tracing.SetSpanSuccess(span)
	return nil
}

// Close closes the database connection
func (r *OrderRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	return sqlDB.Close()
}

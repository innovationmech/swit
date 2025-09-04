package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"

	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/model"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/service"
	"github.com/innovationmech/swit/examples/distributed-tracing/services/order-service/internal/tracing"
)

// OrderHTTPHandler implements the HTTP handler interface
type OrderHTTPHandler struct {
	service *service.OrderService
}

// NewOrderHTTPHandler creates a new HTTP handler
func NewOrderHTTPHandler(service *service.OrderService) *OrderHTTPHandler {
	return &OrderHTTPHandler{
		service: service,
	}
}

// RegisterRoutes registers HTTP routes with the router
func (h *OrderHTTPHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(gin.IRouter)
	if !ok {
		return nil // Skip if not a Gin router
	}

	// Register API routes
	api := ginRouter.Group("/api/v1")
	{
		api.POST("/orders", h.createOrder)
		api.GET("/orders/:id", h.getOrder)
		api.GET("/orders", h.listOrders)
		api.PATCH("/orders/:id/status", h.updateOrderStatus)
	}

	return nil
}

// GetServiceName returns the service name
func (h *OrderHTTPHandler) GetServiceName() string {
	return "order-service-http"
}

// createOrder handles order creation
func (h *OrderHTTPHandler) createOrder(c *gin.Context) {
	ctx, span := tracing.StartSpan(c.Request.Context(), "handler.createOrder")
	defer span.End()

	start := time.Now()

	var req model.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.SetAttributes(
			attribute.String("http.method", "POST"),
			attribute.String("http.route", "/api/v1/orders"),
			attribute.String("error.type", "validation_error"),
		)
		tracing.SetSpanError(span, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Add request details to span
	span.SetAttributes(
		attribute.String("http.method", "POST"),
		attribute.String("http.route", "/api/v1/orders"),
		attribute.String("order.customer_id", req.CustomerID),
		attribute.String("order.product_id", req.ProductID),
		attribute.Int("order.quantity", int(req.Quantity)),
		attribute.Float64("order.amount", req.Amount),
	)

	tracing.AddEvent(span, "request.validated",
		attribute.String("customer.id", req.CustomerID),
		attribute.String("product.id", req.ProductID),
	)

	order, err := h.service.CreateOrder(ctx, &req)
	if err != nil {
		tracing.AddEvent(span, "order.creation_failed",
			attribute.String("error", err.Error()),
		)
		tracing.SetSpanError(span, err)

		// Determine appropriate status code based on error
		statusCode := http.StatusInternalServerError
		if err.Error() == "inventory check failed" {
			statusCode = http.StatusConflict
		} else if err.Error() == "payment processing failed" {
			statusCode = http.StatusPaymentRequired
		}

		c.JSON(statusCode, gin.H{
			"error":   "Failed to create order",
			"details": err.Error(),
		})
		return
	}

	tracing.AddEvent(span, "order.created",
		attribute.String("order.id", order.ID),
		attribute.String("order.status", string(order.Status)),
	)

	// Track response time
	duration := time.Since(start)
	span.SetAttributes(
		attribute.Float64("http.request.duration", duration.Seconds()),
		attribute.Int("http.status_code", http.StatusCreated),
	)

	tracing.SetSpanSuccess(span)

	c.JSON(http.StatusCreated, model.CreateOrderResponse{
		Order: order,
	})
}

// getOrder handles order retrieval
func (h *OrderHTTPHandler) getOrder(c *gin.Context) {
	ctx, span := tracing.StartSpan(c.Request.Context(), "handler.getOrder")
	defer span.End()

	orderID := c.Param("id")
	if orderID == "" {
		err := gin.Error{Err: gin.Error{}.Err, Type: gin.ErrorTypePublic}
		span.SetAttributes(
			attribute.String("http.method", "GET"),
			attribute.String("http.route", "/api/v1/orders/:id"),
			attribute.String("error.type", "missing_parameter"),
		)
		tracing.SetSpanError(span, &err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Order ID is required",
		})
		return
	}

	span.SetAttributes(
		attribute.String("http.method", "GET"),
		attribute.String("http.route", "/api/v1/orders/:id"),
		attribute.String("order.id", orderID),
	)

	order, err := h.service.GetOrder(ctx, orderID)
	if err != nil {
		tracing.SetSpanError(span, err)
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to get order",
				"details": err.Error(),
			})
		}
		return
	}

	tracing.AddEvent(span, "order.retrieved",
		attribute.String("order.id", order.ID),
		attribute.String("order.status", string(order.Status)),
	)

	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK))
	tracing.SetSpanSuccess(span)

	c.JSON(http.StatusOK, model.GetOrderResponse{
		Order: order,
	})
}

// listOrders handles order listing
func (h *OrderHTTPHandler) listOrders(c *gin.Context) {
	ctx, span := tracing.StartSpan(c.Request.Context(), "handler.listOrders")
	defer span.End()

	var req model.ListOrdersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		span.SetAttributes(
			attribute.String("http.method", "GET"),
			attribute.String("http.route", "/api/v1/orders"),
			attribute.String("error.type", "validation_error"),
		)
		tracing.SetSpanError(span, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	span.SetAttributes(
		attribute.String("http.method", "GET"),
		attribute.String("http.route", "/api/v1/orders"),
		attribute.String("filter.customer_id", req.CustomerID),
		attribute.String("filter.status", string(req.Status)),
		attribute.Int("pagination.page_size", req.PageSize),
	)

	orders, nextPageToken, totalCount, err := h.service.ListOrders(ctx, req.CustomerID, req.Status, req.PageSize, req.PageToken)
	if err != nil {
		tracing.SetSpanError(span, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list orders",
			"details": err.Error(),
		})
		return
	}

	tracing.AddEvent(span, "orders.listed",
		attribute.Int("orders.count", len(orders)),
		attribute.Int64("orders.total", totalCount),
	)

	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK))
	tracing.SetSpanSuccess(span)

	c.JSON(http.StatusOK, model.ListOrdersResponse{
		Orders:        orders,
		NextPageToken: nextPageToken,
		TotalCount:    totalCount,
	})
}

// updateOrderStatus handles order status updates
func (h *OrderHTTPHandler) updateOrderStatus(c *gin.Context) {
	ctx, span := tracing.StartSpan(c.Request.Context(), "handler.updateOrderStatus")
	defer span.End()

	orderID := c.Param("id")
	if orderID == "" {
		span.SetAttributes(
			attribute.String("http.method", "PATCH"),
			attribute.String("http.route", "/api/v1/orders/:id/status"),
			attribute.String("error.type", "missing_parameter"),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Order ID is required",
		})
		return
	}

	var req model.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.SetAttributes(
			attribute.String("http.method", "PATCH"),
			attribute.String("http.route", "/api/v1/orders/:id/status"),
			attribute.String("order.id", orderID),
			attribute.String("error.type", "validation_error"),
		)
		tracing.SetSpanError(span, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	span.SetAttributes(
		attribute.String("http.method", "PATCH"),
		attribute.String("http.route", "/api/v1/orders/:id/status"),
		attribute.String("order.id", orderID),
		attribute.String("order.new_status", string(req.Status)),
		attribute.String("order.reason", req.Reason),
	)

	order, err := h.service.UpdateOrderStatus(ctx, orderID, req.Status, req.Reason)
	if err != nil {
		tracing.SetSpanError(span, err)
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to update order status",
				"details": err.Error(),
			})
		}
		return
	}

	tracing.AddEvent(span, "order.status_updated",
		attribute.String("order.id", order.ID),
		attribute.String("order.status", string(order.Status)),
	)

	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK))
	tracing.SetSpanSuccess(span)

	c.JSON(http.StatusOK, model.UpdateOrderStatusResponse{
		Order: order,
	})
}

// Helper function to convert string to int with default
func parseIntWithDefault(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return defaultValue
}

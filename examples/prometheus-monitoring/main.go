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

// Package main demonstrates a comprehensive Prometheus monitoring example
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/types"
)

// MonitoringService implements comprehensive business metrics
type MonitoringService struct {
	name string
}

func NewMonitoringService(name string) *MonitoringService {
	return &MonitoringService{name: name}
}

func (s *MonitoringService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register HTTP handler with business logic
	handler := &BusinessMetricsHandler{
		serviceName:     s.name,
		orderCount:      0,
		userSessions:    make(map[string]time.Time),
		productViews:    make(map[string]int64),
		regionMetrics:   make(map[string]*RegionMetrics),
		paymentMethods:  make(map[string]int64),
		inventoryLevels: make(map[string]int64),
	}
	
	// Initialize some sample data
	handler.initializeSampleData()
	
	if err := registry.RegisterBusinessHTTPHandler(handler); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	// Register health check
	healthCheck := &MonitoringHealthCheck{serviceName: s.name}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check: %w", err)
	}

	return nil
}

// RegionMetrics holds per-region business metrics
type RegionMetrics struct {
	ActiveUsers    int64
	Revenue        float64
	Orders         int64
	AvgOrderValue  float64
	ConversionRate float64
}

// BusinessMetricsHandler demonstrates comprehensive business metrics
type BusinessMetricsHandler struct {
	serviceName string
	metricsCollector types.MetricsCollector
	
	// Business state
	mu              sync.RWMutex
	orderCount      int64
	totalRevenue    float64
	userSessions    map[string]time.Time  // user_id -> session_start
	productViews    map[string]int64      // product_id -> views
	regionMetrics   map[string]*RegionMetrics
	paymentMethods  map[string]int64      // method -> count
	inventoryLevels map[string]int64      // product_id -> quantity
	
	// Performance metrics
	cacheHits       int64
	cacheMisses     int64
	dbConnections   int64
	queueSizes      map[string]int64      // queue_name -> size
}

func (h *BusinessMetricsHandler) initializeSampleData() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Initialize regions
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
	for _, region := range regions {
		h.regionMetrics[region] = &RegionMetrics{
			ActiveUsers:    int64(rand.Intn(1000) + 100),
			Revenue:        float64(rand.Intn(50000) + 10000),
			Orders:         int64(rand.Intn(500) + 50),
			AvgOrderValue:  float64(rand.Intn(200) + 50),
			ConversionRate: float64(rand.Intn(10)+2) / 100.0,
		}
	}
	
	// Initialize payment methods
	h.paymentMethods["credit_card"] = int64(rand.Intn(1000) + 500)
	h.paymentMethods["paypal"] = int64(rand.Intn(300) + 100)
	h.paymentMethods["apple_pay"] = int64(rand.Intn(200) + 50)
	h.paymentMethods["google_pay"] = int64(rand.Intn(150) + 25)
	
	// Initialize inventory
	products := []string{"laptop", "phone", "tablet", "headphones", "keyboard"}
	for _, product := range products {
		h.inventoryLevels[product] = int64(rand.Intn(500) + 50)
	}
	
	// Initialize queue sizes
	h.queueSizes = make(map[string]int64)
	h.queueSizes["order_processing"] = 0
	h.queueSizes["email_sending"] = 0
	h.queueSizes["image_processing"] = 0
	h.queueSizes["data_export"] = 0
}

func (h *BusinessMetricsHandler) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(gin.IRouter)
	if !ok {
		return fmt.Errorf("expected gin.IRouter, got %T", router)
	}

	// Business API endpoints
	api := ginRouter.Group("/api/v1")
	{
		// E-commerce endpoints
		api.POST("/orders", h.handleCreateOrder)
		api.POST("/payments", h.handleProcessPayment)
		api.GET("/products/:id", h.handleProductView)
		api.POST("/cart/add", h.handleAddToCart)
		api.POST("/users/register", h.handleUserRegistration)
		api.POST("/users/login", h.handleUserLogin)
		
		// Analytics endpoints
		api.GET("/analytics/revenue", h.handleRevenueAnalytics)
		api.GET("/analytics/users", h.handleUserAnalytics)
		api.GET("/analytics/products", h.handleProductAnalytics)
		
		// System endpoints
		api.GET("/health", h.handleHealthCheck)
		api.GET("/status", h.handleSystemStatus)
	}

	return nil
}

func (h *BusinessMetricsHandler) GetServiceName() string {
	return h.serviceName
}

// Business endpoint implementations with comprehensive metrics

func (h *BusinessMetricsHandler) handleCreateOrder(c *gin.Context) {
	start := time.Now()
	
	var order struct {
		UserID      string  `json:"user_id" binding:"required"`
		ProductID   string  `json:"product_id" binding:"required"`
		Quantity    int     `json:"quantity" binding:"required"`
		Price       float64 `json:"price" binding:"required"`
		Region      string  `json:"region" binding:"required"`
	}

	if err := c.ShouldBindJSON(&order); err != nil {
		h.trackMetric("order_errors_total", 1, map[string]string{"error_type": "validation"})
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Business logic
	h.mu.Lock()
	h.orderCount++
	orderID := h.orderCount
	orderValue := order.Price * float64(order.Quantity)
	h.totalRevenue += orderValue
	
	// Update regional metrics
	if region, exists := h.regionMetrics[order.Region]; exists {
		region.Orders++
		region.Revenue += orderValue
		region.AvgOrderValue = region.Revenue / float64(region.Orders)
	}
	
	// Update inventory
	if currentStock, exists := h.inventoryLevels[order.ProductID]; exists {
		h.inventoryLevels[order.ProductID] = currentStock - int64(order.Quantity)
	}
	h.mu.Unlock()
	
	// Comprehensive business metrics
	h.trackMetric("orders_created_total", 1, map[string]string{
		"region": order.Region,
		"product": order.ProductID,
		"user_segment": h.getUserSegment(order.UserID),
	})
	
	h.trackHistogram("order_value_dollars", orderValue, map[string]string{
		"region": order.Region,
		"product_category": h.getProductCategory(order.ProductID),
	})
	
	h.trackHistogram("order_quantity", float64(order.Quantity), map[string]string{
		"product": order.ProductID,
	})
	
	// Update inventory gauge
	h.mu.RLock()
	currentStock := h.inventoryLevels[order.ProductID]
	h.mu.RUnlock()
	
	h.trackGauge("inventory_levels", float64(currentStock), map[string]string{
		"product": order.ProductID,
		"warehouse": order.Region,
	})
	
	// Low stock alert threshold
	if currentStock < 10 {
		h.trackMetric("low_stock_alerts_total", 1, map[string]string{
			"product": order.ProductID,
			"severity": "warning",
		})
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"order_id": fmt.Sprintf("order_%d", orderID),
		"status": "created",
		"total": orderValue,
		"timestamp": time.Now().UTC(),
	})
	
	// Track request duration
	h.trackHistogram("request_duration_seconds", time.Since(start).Seconds(), map[string]string{
		"endpoint": "/api/v1/orders",
		"method": "POST",
		"status": "success",
	})
}

func (h *BusinessMetricsHandler) handleProcessPayment(c *gin.Context) {
	start := time.Now()
	
	var payment struct {
		OrderID string  `json:"order_id" binding:"required"`
		Method  string  `json:"method" binding:"required"`
		Amount  float64 `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&payment); err != nil {
		h.trackMetric("payment_errors_total", 1, map[string]string{"error_type": "validation"})
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Simulate payment processing
	processingTime := time.Duration(rand.Intn(500)+100) * time.Millisecond
	time.Sleep(processingTime)
	
	// Simulate success/failure (95% success rate)
	success := rand.Intn(100) < 95
	
	h.mu.Lock()
	if success {
		h.paymentMethods[payment.Method]++
	}
	h.mu.Unlock()
	
	status := "success"
	if !success {
		status = "failed"
	}
	
	// Payment metrics
	h.trackMetric("payments_processed_total", 1, map[string]string{
		"method": payment.Method,
		"status": status,
	})
	
	h.trackHistogram("payment_processing_duration_seconds", processingTime.Seconds(), map[string]string{
		"method": payment.Method,
		"status": status,
	})
	
	h.trackHistogram("payment_amount_dollars", payment.Amount, map[string]string{
		"method": payment.Method,
	})
	
	// Update payment method usage
	h.mu.RLock()
	methodCount := h.paymentMethods[payment.Method]
	h.mu.RUnlock()
	
	h.trackGauge("payment_method_usage", float64(methodCount), map[string]string{
		"method": payment.Method,
	})
	
	if success {
		c.JSON(http.StatusOK, gin.H{
			"payment_id": fmt.Sprintf("pay_%d", time.Now().Unix()),
			"status": "completed",
			"processing_time_ms": processingTime.Milliseconds(),
		})
	} else {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": "payment_failed",
			"reason": "insufficient_funds",
		})
	}
	
	h.trackHistogram("request_duration_seconds", time.Since(start).Seconds(), map[string]string{
		"endpoint": "/api/v1/payments",
		"method": "POST",
		"status": status,
	})
}

func (h *BusinessMetricsHandler) handleProductView(c *gin.Context) {
	start := time.Now()
	productID := c.Param("id")
	userID := c.Query("user_id")
	
	h.mu.Lock()
	h.productViews[productID]++
	views := h.productViews[productID]
	h.mu.Unlock()
	
	// Product metrics
	h.trackMetric("product_views_total", 1, map[string]string{
		"product": productID,
		"category": h.getProductCategory(productID),
		"user_segment": h.getUserSegment(userID),
	})
	
	h.trackGauge("product_popularity", float64(views), map[string]string{
		"product": productID,
	})
	
	// Simulate cache hit/miss
	cacheHit := rand.Intn(100) < 85 // 85% cache hit rate
	h.mu.Lock()
	if cacheHit {
		h.cacheHits++
	} else {
		h.cacheMisses++
	}
	totalOps := h.cacheHits + h.cacheMisses
	hitRate := float64(h.cacheHits) / float64(totalOps)
	h.mu.Unlock()
	
	h.trackMetric("cache_operations_total", 1, map[string]string{
		"result": map[bool]string{true: "hit", false: "miss"}[cacheHit],
		"cache_type": "product_details",
	})
	
	h.trackGauge("cache_hit_rate", hitRate, map[string]string{
		"cache_type": "product_details",
	})
	
	c.JSON(http.StatusOK, gin.H{
		"product_id": productID,
		"name": fmt.Sprintf("Product %s", productID),
		"price": rand.Intn(500) + 50,
		"views": views,
		"cache_hit": cacheHit,
		"timestamp": time.Now().UTC(),
	})
	
	h.trackHistogram("request_duration_seconds", time.Since(start).Seconds(), map[string]string{
		"endpoint": "/api/v1/products",
		"method": "GET",
	})
}

func (h *BusinessMetricsHandler) handleUserLogin(c *gin.Context) {
	start := time.Now()
	
	var login struct {
		Username string `json:"username" binding:"required"`
		Region   string `json:"region"`
	}

	if err := c.ShouldBindJSON(&login); err != nil {
		h.trackMetric("login_errors_total", 1, map[string]string{"error_type": "validation"})
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Simulate authentication (90% success rate)
	success := rand.Intn(100) < 90
	
	if success {
		h.mu.Lock()
		h.userSessions[login.Username] = time.Now()
		
		// Update regional active users
		if region, exists := h.regionMetrics[login.Region]; exists {
			region.ActiveUsers++
		}
		h.mu.Unlock()
		
		// Login success metrics
		h.trackMetric("user_logins_total", 1, map[string]string{
			"status": "success",
			"region": login.Region,
			"user_type": h.getUserType(login.Username),
		})
		
		// Update active users gauge
		h.mu.RLock()
		activeUsers := int64(len(h.userSessions))
		regionUsers := h.regionMetrics[login.Region].ActiveUsers
		h.mu.RUnlock()
		
		h.trackGauge("active_users_total", float64(activeUsers), map[string]string{})
		h.trackGauge("regional_active_users", float64(regionUsers), map[string]string{
			"region": login.Region,
		})
		
		c.JSON(http.StatusOK, gin.H{
			"token": fmt.Sprintf("jwt_token_%d", time.Now().Unix()),
			"expires_in": 3600,
			"user_id": login.Username,
		})
	} else {
		h.trackMetric("user_logins_total", 1, map[string]string{
			"status": "failed",
			"region": login.Region,
			"error_type": "invalid_credentials",
		})
		
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid_credentials",
		})
	}
	
	h.trackHistogram("request_duration_seconds", time.Since(start).Seconds(), map[string]string{
		"endpoint": "/api/v1/users/login",
		"method": "POST",
	})
}

func (h *BusinessMetricsHandler) handleRevenueAnalytics(c *gin.Context) {
	start := time.Now()
	
	h.mu.RLock()
	totalRevenue := h.totalRevenue
	regionMetrics := make(map[string]*RegionMetrics)
	for k, v := range h.regionMetrics {
		regionMetrics[k] = &RegionMetrics{
			ActiveUsers: v.ActiveUsers,
			Revenue: v.Revenue,
			Orders: v.Orders,
			AvgOrderValue: v.AvgOrderValue,
			ConversionRate: v.ConversionRate,
		}
	}
	h.mu.RUnlock()
	
	// Update revenue gauges
	h.trackGauge("total_revenue_dollars", totalRevenue, map[string]string{})
	
	for region, metrics := range regionMetrics {
		h.trackGauge("regional_revenue_dollars", metrics.Revenue, map[string]string{
			"region": region,
		})
		h.trackGauge("average_order_value_dollars", metrics.AvgOrderValue, map[string]string{
			"region": region,
		})
		h.trackGauge("conversion_rate_percent", metrics.ConversionRate*100, map[string]string{
			"region": region,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"total_revenue": totalRevenue,
		"regions": regionMetrics,
		"timestamp": time.Now().UTC(),
	})
	
	h.trackHistogram("request_duration_seconds", time.Since(start).Seconds(), map[string]string{
		"endpoint": "/api/v1/analytics/revenue",
		"method": "GET",
	})
}

// Helper methods for metrics tracking
func (h *BusinessMetricsHandler) trackMetric(name string, value float64, labels map[string]string) {
	if h.metricsCollector != nil {
		h.metricsCollector.AddToCounter(name, value, labels)
	}
}

func (h *BusinessMetricsHandler) trackGauge(name string, value float64, labels map[string]string) {
	if h.metricsCollector != nil {
		h.metricsCollector.SetGauge(name, value, labels)
	}
}

func (h *BusinessMetricsHandler) trackHistogram(name string, value float64, labels map[string]string) {
	if h.metricsCollector != nil {
		h.metricsCollector.ObserveHistogram(name, value, labels)
	}
}

func (h *BusinessMetricsHandler) getUserSegment(userID string) string {
	// Simple segmentation logic
	if len(userID) > 0 {
		switch userID[0] {
		case 'a', 'b', 'c', 'd', 'e':
			return "premium"
		case 'f', 'g', 'h', 'i', 'j':
			return "standard"
		default:
			return "basic"
		}
	}
	return "unknown"
}

func (h *BusinessMetricsHandler) getUserType(username string) string {
	if len(username) > 5 {
		return "returning"
	}
	return "new"
}

func (h *BusinessMetricsHandler) getProductCategory(productID string) string {
	switch productID {
	case "laptop", "phone", "tablet":
		return "electronics"
	case "headphones", "keyboard", "mouse":
		return "accessories"
	default:
		return "other"
	}
}

// Add remaining handlers and health check
func (h *BusinessMetricsHandler) handleAddToCart(c *gin.Context) {
	// Implementation similar to above with cart-specific metrics
	c.JSON(http.StatusOK, gin.H{"status": "added"})
}

func (h *BusinessMetricsHandler) handleUserRegistration(c *gin.Context) {
	// Implementation similar to above with registration metrics
	c.JSON(http.StatusCreated, gin.H{"status": "registered"})
}

func (h *BusinessMetricsHandler) handleUserAnalytics(c *gin.Context) {
	// Implementation similar to above with user analytics
	c.JSON(http.StatusOK, gin.H{"analytics": "user_data"})
}

func (h *BusinessMetricsHandler) handleProductAnalytics(c *gin.Context) {
	// Implementation similar to above with product analytics
	c.JSON(http.StatusOK, gin.H{"analytics": "product_data"})
}

func (h *BusinessMetricsHandler) handleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func (h *BusinessMetricsHandler) handleSystemStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "operational"})
}

// MonitoringHealthCheck implements health checking
type MonitoringHealthCheck struct {
	serviceName string
}

func (h *MonitoringHealthCheck) Check(ctx context.Context) error {
	return nil
}

func (h *MonitoringHealthCheck) GetServiceName() string {
	return h.serviceName
}

// Configuration and main function
func loadConfig() *server.ServerConfig {
	configPath := "swit.yaml"
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefaultConfig()
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Failed to read config file: %v, using defaults", err)
		return createDefaultConfig()
	}
	
	var config server.ServerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Printf("Failed to parse config file: %v, using defaults", err)
		return createDefaultConfig()
	}
	
	return &config
}

func createDefaultConfig() *server.ServerConfig {
	return &server.ServerConfig{
		ServiceName: "prometheus-monitoring-service",
		HTTP: server.HTTPConfig{
			Port:         getEnv("HTTP_PORT", "8080"),
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: server.GRPCConfig{
			Enabled: false,
		},
		ShutdownTimeout: 30 * time.Second,
		Discovery: server.DiscoveryConfig{
			Enabled: false,
		},
		Middleware: server.MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
		Prometheus: *types.DefaultPrometheusConfig(),
	}
}

func main() {
	logger.InitLogger()
	
	config := loadConfig()
	if err := config.Validate(); err != nil {
		log.Fatal("Invalid configuration:", err)
	}
	
	// Create service
	service := NewMonitoringService("prometheus-monitoring-service")
	
	// Create dependency container
	deps := &SimpleDependencyContainer{
		services: make(map[string]interface{}),
	}
	deps.services["config"] = config
	deps.services["version"] = "1.0.0"
	
	// Create server
	baseServer, err := server.NewBusinessServerCore(config, service, deps)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}
	
	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	if err := baseServer.Start(ctx); err != nil {
		log.Fatal("Failed to start server:", err)
	}
	
	log.Printf("Prometheus monitoring service started on http://localhost:%s", config.HTTP.Port)
	log.Printf("Metrics available at: http://localhost:%s/metrics", config.HTTP.Port)
	log.Printf("Business endpoints:")
	log.Printf("  POST /api/v1/orders - Create orders")
	log.Printf("  POST /api/v1/payments - Process payments")
	log.Printf("  GET  /api/v1/products/:id - View products")
	log.Printf("  POST /api/v1/users/login - User login")
	log.Printf("  GET  /api/v1/analytics/revenue - Revenue analytics")
	
	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	log.Println("Shutdown signal received")
	if err := baseServer.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

// Simple dependency container
type SimpleDependencyContainer struct {
	services map[string]interface{}
	closed   bool
}

func (d *SimpleDependencyContainer) GetService(name string) (interface{}, error) {
	if d.closed {
		return nil, fmt.Errorf("container closed")
	}
	service, exists := d.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}
	return service, nil
}

func (d *SimpleDependencyContainer) Close() error {
	d.closed = true
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
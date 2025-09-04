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

// Package main demonstrates an inventory service using the swit framework with OpenTelemetry tracing
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	inventorypb "github.com/innovationmech/swit/api/gen/go/proto/swit/inventory/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/server"
	"github.com/innovationmech/swit/pkg/types"
)

// Simple inventory item for demo
type InventoryItem struct {
	ProductID         string                  `json:"product_id"`
	TotalQuantity     int32                   `json:"total_quantity"`
	ReservedQuantity  int32                   `json:"reserved_quantity"`
	AvailableQuantity int32                   `json:"available_quantity"`
	Reservations      map[string]*Reservation `json:"reservations"`
	mutex             sync.RWMutex
}

type Reservation struct {
	ID         string    `json:"id"`
	ProductID  string    `json:"product_id"`
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	Quantity   int32     `json:"quantity"`
	CreatedAt  time.Time `json:"created_at"`
}

// InventoryService implements both HTTP and gRPC handlers
type InventoryService struct {
	name      string
	inventory map[string]*InventoryItem
	mutex     sync.RWMutex
}

func NewInventoryService(name string) *InventoryService {
	service := &InventoryService{
		name:      name,
		inventory: make(map[string]*InventoryItem),
	}

	// Initialize with some demo data
	service.inventory["product-1"] = &InventoryItem{
		ProductID:         "product-1",
		TotalQuantity:     100,
		ReservedQuantity:  0,
		AvailableQuantity: 100,
		Reservations:      make(map[string]*Reservation),
	}
	service.inventory["product-2"] = &InventoryItem{
		ProductID:         "product-2",
		TotalQuantity:     50,
		ReservedQuantity:  0,
		AvailableQuantity: 50,
		Reservations:      make(map[string]*Reservation),
	}

	return service
}

// HTTP Handler methods
func (s *InventoryService) RegisterRoutes(router interface{}) error {
	ginRouter, ok := router.(gin.IRouter)
	if !ok {
		return nil
	}

	api := ginRouter.Group("/api/v1")
	{
		api.GET("/inventory/:product_id", s.getInventory)
		api.GET("/inventory/:product_id/check", s.checkInventory)
		api.POST("/inventory/reserve", s.reserveInventory)
		api.POST("/inventory/release", s.releaseInventory)
	}
	return nil
}

func (s *InventoryService) GetServiceName() string {
	return "inventory-service-http"
}

// gRPC Service methods
type InventoryGRPCService struct {
	inventoryService *InventoryService
	inventorypb.UnimplementedInventoryServiceServer
}

func (gs *InventoryGRPCService) RegisterGRPC(server interface{}) error {
	grpcServer, ok := server.(*grpc.Server)
	if !ok {
		return nil
	}
	inventorypb.RegisterInventoryServiceServer(grpcServer, gs)
	return nil
}

func (gs *InventoryGRPCService) GetServiceName() string {
	return "inventory-service-grpc"
}

// gRPC method implementations
func (gs *InventoryGRPCService) CheckInventory(ctx context.Context, req *inventorypb.CheckInventoryRequest) (*inventorypb.CheckInventoryResponse, error) {
	tracer := otel.Tracer("inventory-service")
	ctx, span := tracer.Start(ctx, "grpc.CheckInventory")
	defer span.End()

	span.SetAttributes(
		attribute.String("product.id", req.ProductId),
		attribute.Int("requested.quantity", int(req.Quantity)),
	)

	gs.inventoryService.mutex.RLock()
	item, exists := gs.inventoryService.inventory[req.ProductId]
	gs.inventoryService.mutex.RUnlock()

	if !exists {
		return nil, status.Error(codes.NotFound, "product not found")
	}

	available := item.AvailableQuantity >= req.Quantity

	span.SetAttributes(
		attribute.Bool("available", available),
		attribute.Int("available.quantity", int(item.AvailableQuantity)),
	)

	return &inventorypb.CheckInventoryResponse{
		Available:         available,
		AvailableQuantity: item.AvailableQuantity,
		Inventory: &inventorypb.ProductInventory{
			ProductId:         item.ProductID,
			TotalQuantity:     item.TotalQuantity,
			ReservedQuantity:  item.ReservedQuantity,
			AvailableQuantity: item.AvailableQuantity,
		},
	}, nil
}

func (gs *InventoryGRPCService) ReserveInventory(ctx context.Context, req *inventorypb.ReserveInventoryRequest) (*inventorypb.ReserveInventoryResponse, error) {
	tracer := otel.Tracer("inventory-service")
	ctx, span := tracer.Start(ctx, "grpc.ReserveInventory")
	defer span.End()

	span.SetAttributes(
		attribute.String("product.id", req.ProductId),
		attribute.Int("quantity", int(req.Quantity)),
		attribute.String("order.id", req.OrderId),
	)

	gs.inventoryService.mutex.Lock()
	defer gs.inventoryService.mutex.Unlock()

	item, exists := gs.inventoryService.inventory[req.ProductId]
	if !exists {
		return nil, status.Error(codes.NotFound, "product not found")
	}

	if item.AvailableQuantity < req.Quantity {
		return nil, status.Error(codes.FailedPrecondition, "insufficient inventory")
	}

	// Create reservation
	reservation := &Reservation{
		ID:         uuid.New().String(),
		ProductID:  req.ProductId,
		OrderID:    req.OrderId,
		CustomerID: req.CustomerId,
		Quantity:   req.Quantity,
		CreatedAt:  time.Now(),
	}

	// Update inventory
	item.ReservedQuantity += req.Quantity
	item.AvailableQuantity -= req.Quantity
	item.Reservations[reservation.ID] = reservation

	return &inventorypb.ReserveInventoryResponse{
		Reservation: &inventorypb.InventoryReservation{
			ReservationId: reservation.ID,
			ProductId:     reservation.ProductID,
			OrderId:       reservation.OrderID,
			CustomerId:    reservation.CustomerID,
			Quantity:      reservation.Quantity,
			Status:        inventorypb.ReservationStatus_RESERVATION_STATUS_ACTIVE,
			CreatedAt:     timestamppb.New(reservation.CreatedAt),
		},
		Inventory: &inventorypb.ProductInventory{
			ProductId:         item.ProductID,
			TotalQuantity:     item.TotalQuantity,
			ReservedQuantity:  item.ReservedQuantity,
			AvailableQuantity: item.AvailableQuantity,
		},
	}, nil
}

// HTTP handler methods
func (s *InventoryService) getInventory(c *gin.Context) {
	productID := c.Param("product_id")

	tracer := otel.Tracer("inventory-service")
	ctx, span := tracer.Start(c.Request.Context(), "http.GetInventory")
	defer span.End()

	span.SetAttributes(attribute.String("product.id", productID))

	s.mutex.RLock()
	item, exists := s.inventory[productID]
	s.mutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	c.JSON(http.StatusOK, item)
}

func (s *InventoryService) checkInventory(c *gin.Context) {
	productID := c.Param("product_id")

	tracer := otel.Tracer("inventory-service")
	ctx, span := tracer.Start(c.Request.Context(), "http.CheckInventory")
	defer span.End()

	s.mutex.RLock()
	item, exists := s.inventory[productID]
	s.mutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"product_id":         item.ProductID,
		"available_quantity": item.AvailableQuantity,
		"total_quantity":     item.TotalQuantity,
		"reserved_quantity":  item.ReservedQuantity,
	})
}

func (s *InventoryService) reserveInventory(c *gin.Context) {
	var req struct {
		ProductID  string `json:"product_id" binding:"required"`
		Quantity   int32  `json:"quantity" binding:"required"`
		OrderID    string `json:"order_id" binding:"required"`
		CustomerID string `json:"customer_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tracer := otel.Tracer("inventory-service")
	ctx, span := tracer.Start(c.Request.Context(), "http.ReserveInventory")
	defer span.End()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	item, exists := s.inventory[req.ProductID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	if item.AvailableQuantity < req.Quantity {
		c.JSON(http.StatusConflict, gin.H{"error": "insufficient inventory"})
		return
	}

	// Create reservation
	reservation := &Reservation{
		ID:         uuid.New().String(),
		ProductID:  req.ProductID,
		OrderID:    req.OrderID,
		CustomerID: req.CustomerID,
		Quantity:   req.Quantity,
		CreatedAt:  time.Now(),
	}

	// Update inventory
	item.ReservedQuantity += req.Quantity
	item.AvailableQuantity -= req.Quantity
	item.Reservations[reservation.ID] = reservation

	c.JSON(http.StatusOK, gin.H{
		"reservation_id": reservation.ID,
		"inventory":      item,
	})
}

func (s *InventoryService) releaseInventory(c *gin.Context) {
	var req struct {
		ReservationID string `json:"reservation_id"`
		ProductID     string `json:"product_id"`
		OrderID       string `json:"order_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Find reservation
	var reservation *Reservation
	var item *InventoryItem

	if req.ReservationID != "" {
		for _, inv := range s.inventory {
			if res, exists := inv.Reservations[req.ReservationID]; exists {
				reservation = res
				item = inv
				break
			}
		}
	} else if req.ProductID != "" && req.OrderID != "" {
		if inv, exists := s.inventory[req.ProductID]; exists {
			for _, res := range inv.Reservations {
				if res.OrderID == req.OrderID {
					reservation = res
					item = inv
					break
				}
			}
		}
	}

	if reservation == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "reservation not found"})
		return
	}

	// Release reservation
	item.ReservedQuantity -= reservation.Quantity
	item.AvailableQuantity += reservation.Quantity
	delete(item.Reservations, reservation.ID)

	c.JSON(http.StatusOK, gin.H{
		"message":   "reservation released",
		"inventory": item,
	})
}

// ServiceRegistrar implementation
func (s *InventoryService) RegisterServices(registry server.BusinessServiceRegistry) error {
	// Register HTTP handler
	if err := registry.RegisterBusinessHTTPHandler(s); err != nil {
		return fmt.Errorf("failed to register HTTP handler: %w", err)
	}

	// Register gRPC service
	grpcService := &InventoryGRPCService{inventoryService: s}
	if err := registry.RegisterBusinessGRPCService(grpcService); err != nil {
		return fmt.Errorf("failed to register gRPC service: %w", err)
	}

	// Register health check
	healthCheck := &InventoryHealthCheck{serviceName: s.name}
	if err := registry.RegisterBusinessHealthCheck(healthCheck); err != nil {
		return fmt.Errorf("failed to register health check: %w", err)
	}

	return nil
}

// Health check implementation
type InventoryHealthCheck struct {
	serviceName string
}

func (h *InventoryHealthCheck) Check(ctx context.Context) error {
	return nil // Always healthy for demo
}

func (h *InventoryHealthCheck) GetServiceName() string {
	return h.serviceName
}

// Dependency container
type InventoryDependencyContainer struct {
	services map[string]interface{}
	closed   bool
}

func NewInventoryDependencyContainer() *InventoryDependencyContainer {
	return &InventoryDependencyContainer{
		services: make(map[string]interface{}),
		closed:   false,
	}
}

func (d *InventoryDependencyContainer) GetService(name string) (interface{}, error) {
	if d.closed {
		return nil, fmt.Errorf("dependency container is closed")
	}

	service, exists := d.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return service, nil
}

func (d *InventoryDependencyContainer) Close() error {
	d.closed = true
	return nil
}

func (d *InventoryDependencyContainer) AddService(name string, service interface{}) {
	d.services[name] = service
}

// Initialize tracing
func initTracing() (func(context.Context) error, error) {
	jaegerEndpoint := getEnv("JAEGER_ENDPOINT", "http://localhost:14268/api/traces")
	serviceName := getEnv("SERVICE_NAME", "inventory-service")

	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(1.0)),
	)

	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

func main() {
	// Initialize logger
	logger.InitLogger()
	log := logger.GetLogger()

	// Initialize tracing
	tracingShutdown, err := initTracing()
	if err != nil {
		log.Fatal("Failed to initialize tracing", zap.Error(err))
	}

	// Create server configuration
	config := &server.ServerConfig{
		ServiceName: "inventory-service",
		HTTP: server.HTTPConfig{
			Port:         getEnv("SERVER_HTTP_PORT", "8083"),
			EnableReady:  true,
			Enabled:      true,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: server.GRPCConfig{
			Port:                getEnv("SERVER_GRPC_PORT", "9083"),
			EnableReflection:    true,
			EnableHealthService: true,
			Enabled:             true,
			MaxRecvMsgSize:      4 * 1024 * 1024,
			MaxSendMsgSize:      4 * 1024 * 1024,
		},
		ShutdownTimeout: 30 * time.Second,
		Discovery:       server.DiscoveryConfig{Enabled: false},
		Middleware: server.MiddlewareConfig{
			EnableCORS:    true,
			EnableLogging: true,
		},
		Prometheus: *types.DefaultPrometheusConfig(),
	}

	// Create dependencies
	deps := NewInventoryDependencyContainer()
	deps.AddService("tracer", tracingShutdown)

	// Create service
	service := NewInventoryService("inventory-service")

	// Create base server
	baseServer, err := server.NewBusinessServerCore(config, service, deps)
	if err != nil {
		log.Fatal("Failed to create server", zap.Error(err))
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := baseServer.Start(ctx); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}

	log.Info("Inventory service started successfully",
		zap.String("http_address", baseServer.GetHTTPAddress()),
		zap.String("grpc_address", baseServer.GetGRPCAddress()),
		zap.String("service_name", "inventory-service"))

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutdown signal received, stopping server")

	// Graceful shutdown
	if err := baseServer.Shutdown(); err != nil {
		log.Error("Error during shutdown", zap.Error(err))
	} else {
		log.Info("Server stopped gracefully")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

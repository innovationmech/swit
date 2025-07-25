# Service Development Guide

This guide demonstrates how to develop new services using the unified ServiceHandler pattern and interface-based dependency injection approach. It provides step-by-step instructions and practical examples for creating maintainable, testable services.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Creating a New Service](#creating-a-new-service)
3. [Service Implementation Pattern](#service-implementation-pattern)
4. [Handler Implementation](#handler-implementation)
5. [Service Registration](#service-registration)
6. [Testing Strategy](#testing-strategy)
7. [Best Practices](#best-practices)
8. [Common Patterns](#common-patterns)

## Architecture Overview

The new service architecture follows these principles:

- **Interface-based Design**: Services implement well-defined interfaces
- **Unified Registration**: All services use the ServiceHandler pattern
- **Shared Types**: Common types are centralized in types package
- **Clean Dependencies**: Unidirectional dependency flow
- **Testability**: Easy mocking through interfaces

### Dependency Flow

```
Types Package → All Layers (shared types)
Interfaces Package → Service & Handler Layers
Service Layer → Repository Layer
Handler Layer → Service Interfaces (not implementations)
Transport Layer → Handler Layer
```

## Creating a New Service

### Step 1: Define Service Interface

Create the service interface in the interfaces package:

**File**: `internal/switserve/interfaces/notification_service.go`

```go
package interfaces

import (
    "context"
    "github.com/innovationmech/swit/internal/switserve/model"
    "github.com/innovationmech/swit/internal/switserve/types"
)

// NotificationService defines the interface for notification operations
type NotificationService interface {
    // SendNotification sends a notification to specified recipients
    SendNotification(ctx context.Context, notification *model.Notification) error
    
    // GetNotificationHistory retrieves notification history for a user
    GetNotificationHistory(ctx context.Context, userID string, req *types.PaginationRequest) (*types.PaginationResponse, []*model.Notification, error)
    
    // MarkAsRead marks notifications as read
    MarkAsRead(ctx context.Context, notificationIDs []string) error
    
    // GetUnreadCount returns the count of unread notifications
    GetUnreadCount(ctx context.Context, userID string) (int64, error)
}
```

### Step 2: Create Data Models

Define your data models in the model package:

**File**: `internal/switserve/model/notification.go`

```go
package model

import (
    "time"
    "github.com/innovationmech/swit/internal/switserve/types"
)

// Notification represents a notification entity
type Notification struct {
    ID          string           `json:"id" gorm:"primaryKey"`
    UserID      string           `json:"user_id" gorm:"not null;index"`
    Title       string           `json:"title" gorm:"not null"`
    Message     string           `json:"message" gorm:"not null"`
    Type        NotificationType `json:"type" gorm:"not null"`
    Status      types.Status     `json:"status" gorm:"not null;default:'pending'"`
    IsRead      bool             `json:"is_read" gorm:"default:false"`
    Metadata    map[string]interface{} `json:"metadata" gorm:"type:json"`
    CreatedAt   time.Time        `json:"created_at"`
    UpdatedAt   time.Time        `json:"updated_at"`
    ReadAt      *time.Time       `json:"read_at,omitempty"`
}

// NotificationType represents the type of notification
type NotificationType string

const (
    NotificationTypeInfo    NotificationType = "info"
    NotificationTypeWarning NotificationType = "warning"
    NotificationTypeError   NotificationType = "error"
    NotificationTypeSuccess NotificationType = "success"
)
```

### Step 3: Implement Service Layer

Create the service implementation:

**File**: `internal/switserve/service/notification/v1/service.go`

```go
package v1

import (
    "context"
    "fmt"
    "log/slog"
    
    "github.com/innovationmech/swit/internal/switserve/interfaces"
    "github.com/innovationmech/swit/internal/switserve/model"
    "github.com/innovationmech/swit/internal/switserve/repository"
    "github.com/innovationmech/swit/internal/switserve/types"
)

// notificationService implements interfaces.NotificationService
type notificationService struct {
    notificationRepo repository.NotificationRepository
    logger          *slog.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(
    notificationRepo repository.NotificationRepository,
    logger *slog.Logger,
) interfaces.NotificationService {
    return &notificationService{
        notificationRepo: notificationRepo,
        logger:          logger,
    }
}

// SendNotification implements interfaces.NotificationService
func (s *notificationService) SendNotification(ctx context.Context, notification *model.Notification) error {
    s.logger.InfoContext(ctx, "Sending notification", 
        "user_id", notification.UserID,
        "type", notification.Type)
    
    // Validate notification
    if err := s.validateNotification(notification); err != nil {
        return types.NewValidationError(fmt.Sprintf("invalid notification: %v", err))
    }
    
    // Set default values
    if notification.Status == "" {
        notification.Status = types.StatusPending
    }
    
    // Save to repository
    if err := s.notificationRepo.Create(ctx, notification); err != nil {
        s.logger.ErrorContext(ctx, "Failed to save notification", "error", err)
        return fmt.Errorf("failed to save notification: %w", err)
    }
    
    // TODO: Send actual notification (email, push, etc.)
    
    return nil
}

// GetNotificationHistory implements interfaces.NotificationService
func (s *notificationService) GetNotificationHistory(
    ctx context.Context, 
    userID string, 
    req *types.PaginationRequest,
) (*types.PaginationResponse, []*model.Notification, error) {
    s.logger.InfoContext(ctx, "Getting notification history", "user_id", userID)
    
    // Get total count
    total, err := s.notificationRepo.CountByUserID(ctx, userID)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to count notifications: %w", err)
    }
    
    // Get notifications
    notifications, err := s.notificationRepo.FindByUserID(ctx, userID, req.Page, req.PageSize)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to get notifications: %w", err)
    }
    
    // Build pagination response
    pagination := &types.PaginationResponse{
        Page:       req.Page,
        PageSize:   req.PageSize,
        Total:      total,
        TotalPages: int((total + int64(req.PageSize) - 1) / int64(req.PageSize)),
    }
    
    return pagination, notifications, nil
}

// MarkAsRead implements interfaces.NotificationService
func (s *notificationService) MarkAsRead(ctx context.Context, notificationIDs []string) error {
    s.logger.InfoContext(ctx, "Marking notifications as read", "count", len(notificationIDs))
    
    if len(notificationIDs) == 0 {
        return types.NewValidationError("no notification IDs provided")
    }
    
    return s.notificationRepo.MarkAsRead(ctx, notificationIDs)
}

// GetUnreadCount implements interfaces.NotificationService
func (s *notificationService) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
    return s.notificationRepo.CountUnreadByUserID(ctx, userID)
}

// validateNotification validates notification data
func (s *notificationService) validateNotification(notification *model.Notification) error {
    if notification.UserID == "" {
        return fmt.Errorf("user ID is required")
    }
    if notification.Title == "" {
        return fmt.Errorf("title is required")
    }
    if notification.Message == "" {
        return fmt.Errorf("message is required")
    }
    return nil
}
```

### Step 4: Create HTTP Handlers

Implement HTTP handlers for your service:

**File**: `internal/switserve/handler/http/notification/v1/handler.go`

```go
package v1

import (
    "net/http"
    "strconv"
    
    "github.com/gin-gonic/gin"
    "github.com/innovationmech/swit/internal/switserve/interfaces"
    "github.com/innovationmech/swit/internal/switserve/model"
    "github.com/innovationmech/swit/internal/switserve/types"
)

// Handler handles HTTP requests for notification operations
type Handler struct {
    notificationService interfaces.NotificationService
}

// NewHandler creates a new notification handler
func NewHandler(notificationService interfaces.NotificationService) *Handler {
    return &Handler{
        notificationService: notificationService,
    }
}

// SendNotification handles notification sending requests
func (h *Handler) SendNotification(c *gin.Context) {
    var req SendNotificationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, types.ResponseStatus{
            Code:    http.StatusBadRequest,
            Message: err.Error(),
            Success: false,
        })
        return
    }
    
    // Convert request to model
    notification := req.ToModel()
    
    // Call service
    if err := h.notificationService.SendNotification(c.Request.Context(), notification); err != nil {
        c.JSON(http.StatusInternalServerError, types.ResponseStatus{
            Code:    http.StatusInternalServerError,
            Message: err.Error(),
            Success: false,
        })
        return
    }
    
    c.JSON(http.StatusCreated, types.ResponseStatus{
        Code:    http.StatusCreated,
        Message: "Notification sent successfully",
        Success: true,
    })
}

// GetNotificationHistory handles notification history requests
func (h *Handler) GetNotificationHistory(c *gin.Context) {
    userID := c.Param("user_id")
    if userID == "" {
        c.JSON(http.StatusBadRequest, types.ResponseStatus{
            Code:    http.StatusBadRequest,
            Message: "user_id is required",
            Success: false,
        })
        return
    }
    
    // Parse pagination parameters
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
    
    req := &types.PaginationRequest{
        Page:     page,
        PageSize: pageSize,
    }
    
    // Call service
    pagination, notifications, err := h.notificationService.GetNotificationHistory(c.Request.Context(), userID, req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, types.ResponseStatus{
            Code:    http.StatusInternalServerError,
            Message: err.Error(),
            Success: false,
        })
        return
    }
    
    c.JSON(http.StatusOK, NotificationHistoryResponse{
        ResponseStatus: types.ResponseStatus{
            Code:    http.StatusOK,
            Message: "Notifications retrieved successfully",
            Success: true,
        },
        Data: NotificationHistoryData{
            Notifications: notifications,
            Pagination:    pagination,
        },
    })
}

// MarkAsRead handles mark as read requests
func (h *Handler) MarkAsRead(c *gin.Context) {
    var req MarkAsReadRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, types.ResponseStatus{
            Code:    http.StatusBadRequest,
            Message: err.Error(),
            Success: false,
        })
        return
    }
    
    // Call service
    if err := h.notificationService.MarkAsRead(c.Request.Context(), req.NotificationIDs); err != nil {
        c.JSON(http.StatusInternalServerError, types.ResponseStatus{
            Code:    http.StatusInternalServerError,
            Message: err.Error(),
            Success: false,
        })
        return
    }
    
    c.JSON(http.StatusOK, types.ResponseStatus{
        Code:    http.StatusOK,
        Message: "Notifications marked as read",
        Success: true,
    })
}

// GetUnreadCount handles unread count requests
func (h *Handler) GetUnreadCount(c *gin.Context) {
    userID := c.Param("user_id")
    if userID == "" {
        c.JSON(http.StatusBadRequest, types.ResponseStatus{
            Code:    http.StatusBadRequest,
            Message: "user_id is required",
            Success: false,
        })
        return
    }
    
    // Call service
    count, err := h.notificationService.GetUnreadCount(c.Request.Context(), userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, types.ResponseStatus{
            Code:    http.StatusInternalServerError,
            Message: err.Error(),
            Success: false,
        })
        return
    }
    
    c.JSON(http.StatusOK, UnreadCountResponse{
        ResponseStatus: types.ResponseStatus{
            Code:    http.StatusOK,
            Message: "Unread count retrieved successfully",
            Success: true,
        },
        Data: UnreadCountData{
            Count: count,
        },
    })
}
```

**File**: `internal/switserve/handler/http/notification/v1/types.go`

```go
package v1

import (
    "github.com/innovationmech/swit/internal/switserve/model"
    "github.com/innovationmech/swit/internal/switserve/types"
)

// SendNotificationRequest represents a notification sending request
type SendNotificationRequest struct {
    UserID   string                     `json:"user_id" binding:"required"`
    Title    string                     `json:"title" binding:"required"`
    Message  string                     `json:"message" binding:"required"`
    Type     model.NotificationType     `json:"type" binding:"required"`
    Metadata map[string]interface{}     `json:"metadata,omitempty"`
}

// ToModel converts request to model
func (r *SendNotificationRequest) ToModel() *model.Notification {
    return &model.Notification{
        UserID:   r.UserID,
        Title:    r.Title,
        Message:  r.Message,
        Type:     r.Type,
        Metadata: r.Metadata,
    }
}

// MarkAsReadRequest represents a mark as read request
type MarkAsReadRequest struct {
    NotificationIDs []string `json:"notification_ids" binding:"required"`
}

// NotificationHistoryResponse represents notification history response
type NotificationHistoryResponse struct {
    types.ResponseStatus
    Data NotificationHistoryData `json:"data"`
}

// NotificationHistoryData represents notification history data
type NotificationHistoryData struct {
    Notifications []*model.Notification    `json:"notifications"`
    Pagination    *types.PaginationResponse `json:"pagination"`
}

// UnreadCountResponse represents unread count response
type UnreadCountResponse struct {
    types.ResponseStatus
    Data UnreadCountData `json:"data"`
}

// UnreadCountData represents unread count data
type UnreadCountData struct {
    Count int64 `json:"count"`
}
```

### Step 5: Implement Service Handler

Create the ServiceHandler implementation:

**File**: `internal/switserve/handler/notification_service_handler.go`

```go
package handler

import (
    "context"
    "log/slog"
    "time"
    
    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"
    
    "github.com/innovationmech/swit/internal/switserve/handler/http/notification/v1"
    "github.com/innovationmech/swit/internal/switserve/interfaces"
    "github.com/innovationmech/swit/internal/switserve/transport"
    "github.com/innovationmech/swit/internal/switserve/types"
    "github.com/innovationmech/swit/pkg/middleware"
)

// NotificationServiceHandler implements transport.ServiceHandler for notification service
type NotificationServiceHandler struct {
    notificationHandler *v1.Handler
    notificationService interfaces.NotificationService
    startTime          time.Time
    logger             *slog.Logger
}

// NewNotificationServiceHandler creates a new notification service handler
func NewNotificationServiceHandler(
    notificationService interfaces.NotificationService,
    logger *slog.Logger,
) transport.ServiceHandler {
    return &NotificationServiceHandler{
        notificationHandler: v1.NewHandler(notificationService),
        notificationService: notificationService,
        startTime:          time.Now(),
        logger:             logger,
    }
}

// RegisterHTTP implements transport.ServiceHandler
func (h *NotificationServiceHandler) RegisterHTTP(router *gin.Engine) error {
    h.logger.Info("Registering notification HTTP routes")
    
    // Create API v1 group
    v1Group := router.Group("/api/v1")
    
    // Notification endpoints (with auth middleware)
    notificationGroup := v1Group.Group("/notifications")
    notificationGroup.Use(middleware.AuthMiddleware())
    {
        notificationGroup.POST("/send", h.notificationHandler.SendNotification)
        notificationGroup.GET("/users/:user_id/history", h.notificationHandler.GetNotificationHistory)
        notificationGroup.POST("/mark-read", h.notificationHandler.MarkAsRead)
        notificationGroup.GET("/users/:user_id/unread-count", h.notificationHandler.GetUnreadCount)
    }
    
    return nil
}

// RegisterGRPC implements transport.ServiceHandler
func (h *NotificationServiceHandler) RegisterGRPC(server *grpc.Server) error {
    // TODO: Implement gRPC registration when proto definitions are available
    return nil
}

// GetMetadata implements transport.ServiceHandler
func (h *NotificationServiceHandler) GetMetadata() *transport.ServiceMetadata {
    return &transport.ServiceMetadata{
        Name:           "notification-service",
        Version:        "v1.0.0",
        Description:    "Notification management service",
        HealthEndpoint: "/api/v1/notifications/health",
        Tags:           []string{"notification", "messaging"},
        Dependencies:   []string{"database"},
        Metadata: map[string]string{
            "team":        "platform",
            "repository":  "swit",
            "environment": "development",
        },
    }
}

// GetHealthEndpoint implements transport.ServiceHandler
func (h *NotificationServiceHandler) GetHealthEndpoint() string {
    return "/api/v1/notifications/health"
}

// Initialize implements transport.ServiceHandler
func (h *NotificationServiceHandler) Initialize(ctx context.Context) error {
    h.logger.InfoContext(ctx, "Initializing notification service handler")
    // Add any initialization logic here
    return nil
}

// IsHealthy implements transport.ServiceHandler
func (h *NotificationServiceHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
    status := &types.HealthStatus{
        Status:    types.StatusActive,
        Timestamp: time.Now(),
        Version:   "v1.0.0",
        Uptime:    time.Since(h.startTime),
        Dependencies: map[string]types.Status{
            "database": types.StatusActive, // TODO: Check actual database status
        },
    }
    
    return status, nil
}

// Shutdown implements transport.ServiceHandler
func (h *NotificationServiceHandler) Shutdown(ctx context.Context) error {
    h.logger.InfoContext(ctx, "Shutting down notification service handler")
    // Add any cleanup logic here
    return nil
}
```

### Step 6: Register Service

Update the dependency injection and service registration:

**File**: `internal/switserve/deps/deps.go` (add notification service)

```go
// Add to Dependencies struct
type Dependencies struct {
    // ... existing dependencies
    NotificationSrv interfaces.NotificationService
}

// Add to NewDependencies function
func NewDependencies() (*Dependencies, error) {
    // ... existing initialization
    
    // Initialize notification service
    notificationRepo := repository.NewNotificationRepository(db)
    notificationSrv := notificationv1.NewNotificationService(notificationRepo, logger)
    
    return &Dependencies{
        // ... existing dependencies
        NotificationSrv: notificationSrv,
    }, nil
}
```

**File**: `internal/switserve/server.go` (register service handler)

```go
// Add to registerServices method
func (s *Server) registerServices() error {
    // ... existing registrations
    
    // Register notification service handler
    notificationHandler := handler.NewNotificationServiceHandler(s.deps.NotificationSrv, s.logger)
    if err := s.serviceRegistry.Register(notificationHandler); err != nil {
        return fmt.Errorf("failed to register notification service: %w", err)
    }
    
    return nil
}
```

## Testing Strategy

### Unit Testing Service Layer

**File**: `internal/switserve/service/notification/v1/service_test.go`

```go
package v1

import (
    "context"
    "testing"
    "log/slog"
    "os"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    
    "github.com/innovationmech/swit/internal/switserve/model"
    "github.com/innovationmech/swit/internal/switserve/types"
)

// MockNotificationRepository is a mock implementation of NotificationRepository
type MockNotificationRepository struct {
    mock.Mock
}

func (m *MockNotificationRepository) Create(ctx context.Context, notification *model.Notification) error {
    args := m.Called(ctx, notification)
    return args.Error(0)
}

func (m *MockNotificationRepository) FindByUserID(ctx context.Context, userID string, page, pageSize int) ([]*model.Notification, error) {
    args := m.Called(ctx, userID, page, pageSize)
    return args.Get(0).([]*model.Notification), args.Error(1)
}

func (m *MockNotificationRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
    args := m.Called(ctx, userID)
    return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotificationRepository) MarkAsRead(ctx context.Context, notificationIDs []string) error {
    args := m.Called(ctx, notificationIDs)
    return args.Error(0)
}

func (m *MockNotificationRepository) CountUnreadByUserID(ctx context.Context, userID string) (int64, error) {
    args := m.Called(ctx, userID)
    return args.Get(0).(int64), args.Error(1)
}

func TestNotificationService_SendNotification(t *testing.T) {
    // Setup
    mockRepo := new(MockNotificationRepository)
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    service := NewNotificationService(mockRepo, logger)
    
    ctx := context.Background()
    notification := &model.Notification{
        UserID:  "user123",
        Title:   "Test Notification",
        Message: "This is a test message",
        Type:    model.NotificationTypeInfo,
    }
    
    // Mock expectations
    mockRepo.On("Create", ctx, mock.AnythingOfType("*model.Notification")).Return(nil)
    
    // Execute
    err := service.SendNotification(ctx, notification)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, types.StatusPending, notification.Status)
    mockRepo.AssertExpectations(t)
}

func TestNotificationService_SendNotification_ValidationError(t *testing.T) {
    // Setup
    mockRepo := new(MockNotificationRepository)
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    service := NewNotificationService(mockRepo, logger)
    
    ctx := context.Background()
    notification := &model.Notification{
        // Missing required fields
        Title:   "Test Notification",
        Message: "This is a test message",
    }
    
    // Execute
    err := service.SendNotification(ctx, notification)
    
    // Assert
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "user ID is required")
    mockRepo.AssertNotCalled(t, "Create")
}
```

### Integration Testing Service Handler

**File**: `internal/switserve/handler/notification_service_handler_test.go`

```go
package handler

import (
    "context"
    "testing"
    "log/slog"
    "os"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    
    "github.com/innovationmech/swit/internal/switserve/model"
    "github.com/innovationmech/swit/internal/switserve/types"
)

// MockNotificationService is a mock implementation of NotificationService
type MockNotificationService struct {
    mock.Mock
}

func (m *MockNotificationService) SendNotification(ctx context.Context, notification *model.Notification) error {
    args := m.Called(ctx, notification)
    return args.Error(0)
}

func (m *MockNotificationService) GetNotificationHistory(ctx context.Context, userID string, req *types.PaginationRequest) (*types.PaginationResponse, []*model.Notification, error) {
    args := m.Called(ctx, userID, req)
    return args.Get(0).(*types.PaginationResponse), args.Get(1).([]*model.Notification), args.Error(2)
}

func (m *MockNotificationService) MarkAsRead(ctx context.Context, notificationIDs []string) error {
    args := m.Called(ctx, notificationIDs)
    return args.Error(0)
}

func (m *MockNotificationService) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
    args := m.Called(ctx, userID)
    return args.Get(0).(int64), args.Error(1)
}

func TestNotificationServiceHandler_RegisterHTTP(t *testing.T) {
    // Setup
    mockService := new(MockNotificationService)
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    handler := NewNotificationServiceHandler(mockService, logger)
    
    router := gin.New()
    
    // Execute
    err := handler.RegisterHTTP(router)
    
    // Assert
    assert.NoError(t, err)
    
    // Verify routes are registered (you can check router.Routes() if needed)
}

func TestNotificationServiceHandler_GetMetadata(t *testing.T) {
    // Setup
    mockService := new(MockNotificationService)
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    handler := NewNotificationServiceHandler(mockService, logger)
    
    // Execute
    metadata := handler.GetMetadata()
    
    // Assert
    assert.Equal(t, "notification-service", metadata.Name)
    assert.Equal(t, "v1.0.0", metadata.Version)
    assert.Equal(t, "/api/v1/notifications/health", metadata.HealthEndpoint)
    assert.Contains(t, metadata.Tags, "notification")
    assert.Contains(t, metadata.Dependencies, "database")
}

func TestNotificationServiceHandler_IsHealthy(t *testing.T) {
    // Setup
    mockService := new(MockNotificationService)
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    handler := NewNotificationServiceHandler(mockService, logger)
    
    ctx := context.Background()
    
    // Execute
    status, err := handler.IsHealthy(ctx)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, types.StatusActive, status.Status)
    assert.NotZero(t, status.Uptime)
    assert.Equal(t, "v1.0.0", status.Version)
}
```

## Best Practices

### 1. Interface Design

- **Keep interfaces focused**: Each interface should have a single responsibility
- **Use context**: Always pass context.Context as the first parameter
- **Return errors**: Always return errors for operations that can fail
- **Use shared types**: Leverage the types package for common structures

### 2. Service Implementation

- **Validate inputs**: Always validate input parameters
- **Use structured logging**: Include relevant context in log messages
- **Handle errors gracefully**: Wrap errors with meaningful context
- **Keep business logic pure**: Avoid side effects in business logic

### 3. Handler Implementation

- **Focus on transport**: Handlers should only handle HTTP concerns
- **Delegate to services**: Business logic should be in the service layer
- **Use consistent responses**: Follow the established response patterns
- **Validate requests**: Use binding tags for request validation

### 4. Service Handler Implementation

- **Implement all methods**: Ensure all ServiceHandler methods are implemented
- **Use meaningful metadata**: Provide accurate service metadata
- **Handle lifecycle properly**: Implement proper initialization and shutdown
- **Monitor health**: Provide accurate health status

## Common Patterns

### Error Handling Pattern

```go
// Service layer
func (s *service) DoSomething(ctx context.Context, input *Input) error {
    if err := s.validate(input); err != nil {
        return types.NewValidationError(err.Error())
    }
    
    if err := s.repository.Save(ctx, input); err != nil {
        s.logger.ErrorContext(ctx, "Failed to save", "error", err)
        return fmt.Errorf("failed to save: %w", err)
    }
    
    return nil
}

// Handler layer
func (h *Handler) DoSomething(c *gin.Context) {
    var req Request
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, types.ResponseStatus{
            Code:    http.StatusBadRequest,
            Message: err.Error(),
            Success: false,
        })
        return
    }
    
    if err := h.service.DoSomething(c.Request.Context(), req.ToModel()); err != nil {
        status := http.StatusInternalServerError
        if _, ok := err.(types.ServiceError); ok {
            status = http.StatusBadRequest
        }
        
        c.JSON(status, types.ResponseStatus{
            Code:    status,
            Message: err.Error(),
            Success: false,
        })
        return
    }
    
    c.JSON(http.StatusOK, types.ResponseStatus{
        Code:    http.StatusOK,
        Message: "Operation completed successfully",
        Success: true,
    })
}
```

### Pagination Pattern

```go
// Service layer
func (s *service) GetItems(ctx context.Context, req *types.PaginationRequest) (*types.PaginationResponse, []*Item, error) {
    total, err := s.repository.Count(ctx)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to count items: %w", err)
    }
    
    items, err := s.repository.Find(ctx, req.Page, req.PageSize)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to get items: %w", err)
    }
    
    pagination := &types.PaginationResponse{
        Page:       req.Page,
        PageSize:   req.PageSize,
        Total:      total,
        TotalPages: int((total + int64(req.PageSize) - 1) / int64(req.PageSize)),
    }
    
    return pagination, items, nil
}
```

### Health Check Pattern

```go
func (h *ServiceHandler) IsHealthy(ctx context.Context) (*types.HealthStatus, error) {
    status := &types.HealthStatus{
        Status:    types.StatusActive,
        Timestamp: time.Now(),
        Version:   h.GetMetadata().Version,
        Uptime:    time.Since(h.startTime),
        Dependencies: make(map[string]types.Status),
    }
    
    // Check dependencies
    if err := h.checkDatabase(ctx); err != nil {
        status.Dependencies["database"] = types.StatusInactive
        status.Status = types.StatusInactive
    } else {
        status.Dependencies["database"] = types.StatusActive
    }
    
    return status, nil
}
```

This guide provides a comprehensive foundation for developing services using the new architecture. Follow these patterns and practices to ensure consistency and maintainability across your service implementations.
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

package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	v1 "github.com/innovationmech/swit/internal/switserve/service/notification/v1"
	"github.com/innovationmech/swit/internal/switserve/types"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// MockNotificationService is a mock implementation of NotificationService
type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) CreateNotification(ctx context.Context, userID, title, content string) (*v1.Notification, error) {
	args := m.Called(ctx, userID, title, content)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.Notification), args.Error(1)
}

func (m *MockNotificationService) GetNotifications(ctx context.Context, userID string, limit, offset int) ([]*v1.Notification, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*v1.Notification), args.Error(1)
}

func (m *MockNotificationService) MarkAsRead(ctx context.Context, notificationID string) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockNotificationService) DeleteNotification(ctx context.Context, notificationID string) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

// Test setup
func setupTest() {
	logger.InitLogger()
	gin.SetMode(gin.TestMode)
}

func TestMain(m *testing.M) {
	setupTest()
	m.Run()
}

// TestNewHandler tests the NewNotificationHandler constructor
func TestNewHandler(t *testing.T) {
	tests := []struct {
		name        string
		service     v1.NotificationService
		description string
	}{
		{
			name:        "success_create_handler",
			service:     &MockNotificationService{},
			description: "Should successfully create a new notification handler",
		},
		{
			name:        "with_nil_service",
			service:     nil,
			description: "Should create handler even with nil service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewNotificationHandler(tt.service)

			assert.NotNil(t, handler)
			assert.Equal(t, tt.service, handler.service)
			assert.WithinDuration(t, time.Now(), handler.startTime, time.Second)
		})
	}
}

// TestHandler_CreateNotification tests the CreateNotification HTTP handler
func TestHandler_CreateNotification(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockNotificationService)
		expectedStatus int
		expectedError  string
		description    string
	}{
		{
			name: "success_create_notification",
			requestBody: map[string]string{
				"user_id": "test-user-1",
				"title":   "Test Notification",
				"content": "Test content",
			},
			mockSetup: func(m *MockNotificationService) {
				m.On("CreateNotification", mock.Anything, "test-user-1", "Test Notification", "Test content").Return(
					&v1.Notification{
						ID:        "test-id",
						UserID:    "test-user-1",
						Title:     "Test Notification",
						Content:   "Test content",
						IsRead:    false,
						CreatedAt: time.Now().Unix(),
						UpdatedAt: time.Now().Unix(),
					}, nil)
			},
			expectedStatus: http.StatusCreated,
			description:    "Should successfully create a notification",
		},
		{
			name:           "invalid_json",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
			description:    "Should return error for invalid JSON",
		},
		{
			name: "missing_user_id",
			requestBody: map[string]string{
				"title":   "Test Notification",
				"content": "Test content",
			},
			mockSetup:      func(m *MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
			description:    "Should return error when user_id is missing",
		},
		{
			name: "missing_title",
			requestBody: map[string]string{
				"user_id": "test-user-1",
				"content": "Test content",
			},
			mockSetup:      func(m *MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
			description:    "Should return error when title is missing",
		},
		{
			name: "missing_content",
			requestBody: map[string]string{
				"user_id": "test-user-1",
				"title":   "Test Notification",
			},
			mockSetup:      func(m *MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
			description:    "Should return error when content is missing",
		},
		{
			name: "service_error",
			requestBody: map[string]string{
				"user_id": "test-user-1",
				"title":   "Test Notification",
				"content": "Test content",
			},
			mockSetup: func(m *MockNotificationService) {
				m.On("CreateNotification", mock.Anything, "test-user-1", "Test Notification", "Test content").Return(
					nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal server error",
			description:    "Should return error when service fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockNotificationService{}
			tt.mockSetup(mockService)

			handler := NewNotificationHandler(mockService)
			router := gin.New()
			router.POST("/notifications", handler.CreateNotification)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/notifications", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Request-ID", "test-request-id")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			} else {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "notification")
				assert.Contains(t, response, "metadata")
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestHandler_GetNotifications tests the GetNotifications HTTP handler
func TestHandler_GetNotifications(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func(*MockNotificationService)
		expectedStatus int
		expectedError  string
		description    string
	}{
		{
			name:        "success_get_notifications",
			queryParams: "user_id=test-user-1&limit=10&offset=0",
			mockSetup: func(m *MockNotificationService) {
				m.On("GetNotifications", mock.Anything, "test-user-1", 10, 0).Return(
					[]*v1.Notification{
						{
							ID:        "test-id-1",
							UserID:    "test-user-1",
							Title:     "Test Notification 1",
							Content:   "Test content 1",
							IsRead:    false,
							CreatedAt: time.Now().Unix(),
							UpdatedAt: time.Now().Unix(),
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Should successfully get notifications",
		},
		{
			name:           "missing_user_id",
			queryParams:    "limit=10&offset=0",
			mockSetup:      func(m *MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "user_id is required",
			description:    "Should return error when user_id is missing",
		},
		{
			name:        "default_pagination",
			queryParams: "user_id=test-user-1",
			mockSetup: func(m *MockNotificationService) {
				m.On("GetNotifications", mock.Anything, "test-user-1", 20, 0).Return(
					[]*v1.Notification{}, nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Should use default pagination when not specified",
		},
		{
			name:        "service_error",
			queryParams: "user_id=test-user-1",
			mockSetup: func(m *MockNotificationService) {
				m.On("GetNotifications", mock.Anything, "test-user-1", 20, 0).Return(
					nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal server error",
			description:    "Should return error when service fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockNotificationService{}
			tt.mockSetup(mockService)

			handler := NewNotificationHandler(mockService)
			router := gin.New()
			router.GET("/notifications", handler.GetNotifications)

			req := httptest.NewRequest(http.MethodGet, "/notifications?"+tt.queryParams, nil)
			req.Header.Set("X-Request-ID", "test-request-id")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			} else {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "notifications")
				assert.Contains(t, response, "total_count")
				assert.Contains(t, response, "metadata")
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestHandler_MarkAsRead tests the MarkAsRead HTTP handler
func TestHandler_MarkAsRead(t *testing.T) {
	tests := []struct {
		name           string
		notificationID string
		mockSetup      func(*MockNotificationService)
		expectedStatus int
		expectedError  string
		description    string
	}{
		{
			name:           "success_mark_as_read",
			notificationID: "test-id",
			mockSetup: func(m *MockNotificationService) {
				m.On("MarkAsRead", mock.Anything, "test-id").Return(nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Should successfully mark notification as read",
		},
		{
			name:           "empty_notification_id",
			notificationID: "",
			mockSetup:      func(m *MockNotificationService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "notification_id is required",
			description:    "Should return 400 when notification_id is empty",
		},
		{
			name:           "notification_not_found",
			notificationID: "non-existent-id",
			mockSetup: func(m *MockNotificationService) {
				m.On("MarkAsRead", mock.Anything, "non-existent-id").Return(v1.ErrNotificationNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "notification not found",
			description:    "Should return 404 when notification is not found",
		},
		{
			name:           "service_error",
			notificationID: "test-id",
			mockSetup: func(m *MockNotificationService) {
				m.On("MarkAsRead", mock.Anything, "test-id").Return(errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal server error",
			description:    "Should return error when service fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockNotificationService{}
			tt.mockSetup(mockService)

			handler := NewNotificationHandler(mockService)
			router := gin.New()
			router.PUT("/notifications/:id/read", handler.MarkAsRead)

			url := fmt.Sprintf("/notifications/%s/read", tt.notificationID)
			req := httptest.NewRequest(http.MethodPut, url, nil)
			req.Header.Set("X-Request-ID", "test-request-id")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			} else if w.Code == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response["success"].(bool))
				assert.Contains(t, response, "metadata")
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestHandler_DeleteNotification tests the DeleteNotification HTTP handler
func TestHandler_DeleteNotification(t *testing.T) {
	tests := []struct {
		name           string
		notificationID string
		mockSetup      func(*MockNotificationService)
		expectedStatus int
		expectedError  string
		description    string
	}{
		{
			name:           "success_delete_notification",
			notificationID: "test-id",
			mockSetup: func(m *MockNotificationService) {
				m.On("DeleteNotification", mock.Anything, "test-id").Return(nil)
			},
			expectedStatus: http.StatusOK,
			description:    "Should successfully delete notification",
		},
		{
			name:           "empty_notification_id",
			notificationID: "",
			mockSetup:      func(m *MockNotificationService) {},
			expectedStatus: http.StatusNotFound,
			expectedError:  "",
			description:    "Should return 404 when notification_id is empty (route not matched)",
		},
		{
			name:           "notification_not_found",
			notificationID: "non-existent-id",
			mockSetup: func(m *MockNotificationService) {
				m.On("DeleteNotification", mock.Anything, "non-existent-id").Return(v1.ErrNotificationNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "notification not found",
			description:    "Should return 404 when notification is not found",
		},
		{
			name:           "service_error",
			notificationID: "test-id",
			mockSetup: func(m *MockNotificationService) {
				m.On("DeleteNotification", mock.Anything, "test-id").Return(errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal server error",
			description:    "Should return error when service fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockNotificationService{}
			tt.mockSetup(mockService)

			handler := NewNotificationHandler(mockService)
			router := gin.New()
			router.DELETE("/notifications/:id", handler.DeleteNotification)

			url := fmt.Sprintf("/notifications/%s", tt.notificationID)
			req := httptest.NewRequest(http.MethodDelete, url, nil)
			req.Header.Set("X-Request-ID", "test-request-id")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			} else if w.Code == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.True(t, response["success"].(bool))
				assert.Contains(t, response, "metadata")
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestHandler_RegisterHTTP tests the RegisterHTTP method
func TestHandler_RegisterHTTP(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "success_register_http_routes",
			description: "Should successfully register HTTP routes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockNotificationService{}
			handler := NewNotificationHandler(mockService)
			router := gin.New()

			err := handler.RegisterHTTP(router)

			assert.NoError(t, err)
			// Note: Cannot easily test route registration without accessing internal router state
		})
	}
}

// TestHandler_RegisterGRPC tests the RegisterGRPC method
func TestHandler_RegisterGRPC(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "success_register_grpc_services",
			description: "Should successfully register gRPC services (no-op)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockNotificationService{}
			handler := NewNotificationHandler(mockService)
			server := grpc.NewServer()

			err := handler.RegisterGRPC(server)

			assert.NoError(t, err)
		})
	}
}

// TestHandler_GetMetadata tests the GetMetadata method
func TestHandler_GetMetadata(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "success_get_metadata",
			description: "Should return correct service metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockNotificationService{}
			handler := NewNotificationHandler(mockService)

			metadata := handler.GetMetadata()

			assert.NotNil(t, metadata)
			assert.Equal(t, "notification", metadata.Name)
			assert.Equal(t, "v1", metadata.Version)
			assert.Equal(t, "Notification service providing notification management functionality", metadata.Description)
			assert.Equal(t, "/api/v1/health", metadata.HealthEndpoint)
			assert.Contains(t, metadata.Tags, "notification")
			assert.Contains(t, metadata.Tags, "messaging")
			assert.Contains(t, metadata.Tags, "api")
			assert.Empty(t, metadata.Dependencies)
		})
	}
}

// TestHandler_GetHealthEndpoint tests the GetHealthEndpoint method
func TestHandler_GetHealthEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		expected    string
		description string
	}{
		{
			name:        "success_get_health_endpoint",
			expected:    "/api/v1/health",
			description: "Should return correct health endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockNotificationService{}
			handler := NewNotificationHandler(mockService)

			endpoint := handler.GetHealthEndpoint()

			assert.Equal(t, tt.expected, endpoint)
		})
	}
}

// TestHandler_IsHealthy tests the IsHealthy method
func TestHandler_IsHealthy(t *testing.T) {
	tests := []struct {
		name           string
		service        v1.NotificationService
		expectedStatus string
		description    string
	}{
		{
			name:           "healthy_with_service",
			service:        &MockNotificationService{},
			expectedStatus: types.HealthStatusHealthy,
			description:    "Should return healthy when service is available",
		},
		{
			name:           "unhealthy_with_nil_service",
			service:        nil,
			expectedStatus: types.HealthStatusUnhealthy,
			description:    "Should return unhealthy when service is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewNotificationHandler(tt.service)
			ctx := context.Background()

			status, err := handler.IsHealthy(ctx)

			assert.NoError(t, err)
			assert.NotNil(t, status)
			assert.Equal(t, tt.expectedStatus, status.Status)
			assert.Equal(t, "v1", status.Version)
			assert.GreaterOrEqual(t, status.Uptime, time.Duration(0))
		})
	}
}

// TestHandler_Initialize tests the Initialize method
func TestHandler_Initialize(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "success_initialize",
			description: "Should successfully initialize handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockNotificationService{}
			handler := NewNotificationHandler(mockService)
			ctx := context.Background()

			err := handler.Initialize(ctx)

			assert.NoError(t, err)
		})
	}
}

// TestHandler_Shutdown tests the Shutdown method
func TestHandler_Shutdown(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "success_shutdown",
			description: "Should successfully shutdown handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockNotificationService{}
			handler := NewNotificationHandler(mockService)
			ctx := context.Background()

			err := handler.Shutdown(ctx)

			assert.NoError(t, err)
		})
	}
}

// TestHandler_ServiceHandlerInterface tests that NotificationHandler implements HandlerRegister interface
func TestHandler_ServiceHandlerInterface(t *testing.T) {
	t.Run("implements_service_handler_interface", func(t *testing.T) {
		mockService := &MockNotificationService{}
		handler := NewNotificationHandler(mockService)

		// Verify that NotificationHandler implements transport.HandlerRegister interface
		var _ transport.HandlerRegister = handler
		assert.NotNil(t, handler)
	})
}

// Integration test
func TestHandler_Integration(t *testing.T) {
	t.Run("complete_notification_lifecycle", func(t *testing.T) {
		mockService := &MockNotificationService{}
		handler := NewNotificationHandler(mockService)
		router := gin.New()

		// HandlerRegister routes
		err := handler.RegisterHTTP(router)
		require.NoError(t, err)

		// Test metadata
		metadata := handler.GetMetadata()
		assert.Equal(t, "notification", metadata.Name)

		// Test health check
		status, err := handler.IsHealthy(context.Background())
		require.NoError(t, err)
		assert.Equal(t, types.HealthStatusHealthy, status.Status)

		// Test initialization and shutdown
		err = handler.Initialize(context.Background())
		assert.NoError(t, err)

		err = handler.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

// Benchmark tests
func BenchmarkHandler_CreateNotification(b *testing.B) {
	mockService := &MockNotificationService{}
	mockService.On("CreateNotification", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		&v1.Notification{
			ID:        "test-id",
			UserID:    "test-user",
			Title:     "Test",
			Content:   "Test content",
			IsRead:    false,
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		}, nil)

	handler := NewNotificationHandler(mockService)
	router := gin.New()
	router.POST("/notifications", handler.CreateNotification)

	body := `{"user_id":"test-user","title":"Test","content":"Test content"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/notifications", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkHandler_GetNotifications(b *testing.B) {
	mockService := &MockNotificationService{}
	mockService.On("GetNotifications", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		[]*v1.Notification{}, nil)

	handler := NewNotificationHandler(mockService)
	router := gin.New()
	router.GET("/notifications", handler.GetNotifications)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/notifications?user_id=test-user", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkHandler_MarkAsRead(b *testing.B) {
	mockService := &MockNotificationService{}
	mockService.On("MarkAsRead", mock.Anything, mock.Anything).Return(nil)

	handler := NewNotificationHandler(mockService)
	router := gin.New()
	router.PUT("/notifications/:id/read", handler.MarkAsRead)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPut, "/notifications/test-id/read", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

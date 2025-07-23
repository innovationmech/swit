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

package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	notificationv1 "github.com/innovationmech/swit/internal/switserve/service/notification/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func setupTest() {
	logger.InitLogger()
	gin.SetMode(gin.TestMode)
}

func TestMain(m *testing.M) {
	setupTest()
	code := m.Run()
	os.Exit(code)
}

func TestNewServiceRegistrar(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "creates_registrar_successfully",
			description: "Should create a notification service registrar successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := notificationv1.NewService()
			registrar := NewServiceRegistrar(mockService)

			assert.NotNil(t, registrar)
			assert.NotNil(t, registrar.httpHandler)
			assert.NotNil(t, registrar.grpcHandler)
		})
	}
}

func TestServiceRegistrar_GetName(t *testing.T) {
	tests := []struct {
		name         string
		expectedName string
	}{
		{
			name:         "returns_correct_service_name",
			expectedName: "notification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := notificationv1.NewService()
			registrar := NewServiceRegistrar(mockService)

			name := registrar.GetName()
			assert.Equal(t, tt.expectedName, name)
		})
	}
}

func TestServiceRegistrar_RegisterGRPC(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		description string
	}{
		{
			name:        "success_register_grpc",
			expectError: false,
			description: "Should successfully register gRPC service without error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := notificationv1.NewService()
			registrar := NewServiceRegistrar(mockService)
			server := grpc.NewServer()

			err := registrar.RegisterGRPC(server)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServiceRegistrar_RegisterHTTP(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		description string
	}{
		{
			name:        "success_register_http",
			expectError: false,
			description: "Should successfully register HTTP routes without error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := notificationv1.NewService()
			registrar := NewServiceRegistrar(mockService)
			router := gin.New()

			err := registrar.RegisterHTTP(router)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServiceRegistrar_CreateNotificationHTTP(t *testing.T) {
	tests := []struct {
		name         string
		requestBody  map[string]interface{}
		expectStatus int
		expectError  bool
		description  string
	}{
		{
			name: "success_create_notification",
			requestBody: map[string]interface{}{
				"user_id": "test-user-1",
				"title":   "Test Notification",
				"content": "This is a test notification",
			},
			expectStatus: http.StatusCreated,
			expectError:  false,
			description:  "Should successfully create a notification",
		},
		{
			name: "missing_user_id",
			requestBody: map[string]interface{}{
				"title":   "Test Notification",
				"content": "This is a test notification",
			},
			expectStatus: http.StatusBadRequest,
			expectError:  true,
			description:  "Should return error when user_id is missing",
		},
		{
			name: "missing_title",
			requestBody: map[string]interface{}{
				"user_id": "test-user-1",
				"content": "This is a test notification",
			},
			expectStatus: http.StatusBadRequest,
			expectError:  true,
			description:  "Should return error when title is missing",
		},
		{
			name: "missing_content",
			requestBody: map[string]interface{}{
				"user_id": "test-user-1",
				"title":   "Test Notification",
			},
			expectStatus: http.StatusBadRequest,
			expectError:  true,
			description:  "Should return error when content is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := notificationv1.NewService()
			registrar := NewServiceRegistrar(mockService)
			router := gin.New()
			registrar.RegisterHTTP(router)

			bodyBytes, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)

			if !tt.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response, "notification")
				assert.Contains(t, response, "metadata")

				notification := response["notification"].(map[string]interface{})
				assert.NotEmpty(t, notification["id"])
				assert.Equal(t, tt.requestBody["user_id"], notification["user_id"])
				assert.Equal(t, tt.requestBody["title"], notification["title"])
				assert.Equal(t, tt.requestBody["content"], notification["content"])
				assert.Equal(t, false, notification["is_read"])
			}
		})
	}
}

func TestServiceRegistrar_GetNotificationsHTTP(t *testing.T) {
	tests := []struct {
		name         string
		queryParams  map[string]string
		expectStatus int
		expectError  bool
		description  string
	}{
		{
			name: "success_get_notifications",
			queryParams: map[string]string{
				"user_id": "test-user-1",
				"limit":   "10",
				"offset":  "0",
			},
			expectStatus: http.StatusOK,
			expectError:  false,
			description:  "Should successfully get notifications",
		},
		{
			name:         "missing_user_id",
			queryParams:  map[string]string{},
			expectStatus: http.StatusBadRequest,
			expectError:  true,
			description:  "Should return error when user_id is missing",
		},
		{
			name: "success_with_defaults",
			queryParams: map[string]string{
				"user_id": "test-user-1",
			},
			expectStatus: http.StatusOK,
			expectError:  false,
			description:  "Should successfully get notifications with default pagination",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := notificationv1.NewService()
			registrar := NewServiceRegistrar(mockService)
			router := gin.New()
			registrar.RegisterHTTP(router)

			url := "/api/v1/notifications"
			if len(tt.queryParams) > 0 {
				url += "?"
				for key, value := range tt.queryParams {
					url += fmt.Sprintf("%s=%s&", key, value)
				}
				url = url[:len(url)-1] // Remove trailing &
			}

			req, _ := http.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)

			if !tt.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response, "notifications")
				assert.Contains(t, response, "total_count")
				assert.Contains(t, response, "metadata")
			}
		})
	}
}

func TestServiceRegistrar_MarkAsReadHTTP(t *testing.T) {
	mockService := notificationv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()
	registrar.RegisterHTTP(router)

	// First create a notification
	createBody := map[string]interface{}{
		"user_id": "test-user-1",
		"title":   "Test Notification",
		"content": "This is a test notification",
	}
	bodyBytes, _ := json.Marshal(createBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	notification := createResponse["notification"].(map[string]interface{})
	notificationID := notification["id"].(string)

	tests := []struct {
		name           string
		notificationID string
		expectStatus   int
		expectError    bool
		description    string
	}{
		{
			name:           "success_mark_as_read",
			notificationID: notificationID,
			expectStatus:   http.StatusOK,
			expectError:    false,
			description:    "Should successfully mark notification as read",
		},
		{
			name:           "not_found_notification",
			notificationID: "non-existent-id",
			expectStatus:   http.StatusNotFound,
			expectError:    true,
			description:    "Should return 404 for non-existent notification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/notifications/%s/read", tt.notificationID)
			req, _ := http.NewRequest(http.MethodPatch, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)

			if !tt.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response, "success")
				assert.Contains(t, response, "metadata")
				assert.Equal(t, true, response["success"])
			}
		})
	}
}

func TestServiceRegistrar_DeleteNotificationHTTP(t *testing.T) {
	mockService := notificationv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()
	registrar.RegisterHTTP(router)

	// First create a notification
	createBody := map[string]interface{}{
		"user_id": "test-user-1",
		"title":   "Test Notification",
		"content": "This is a test notification",
	}
	bodyBytes, _ := json.Marshal(createBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	notification := createResponse["notification"].(map[string]interface{})
	notificationID := notification["id"].(string)

	tests := []struct {
		name           string
		notificationID string
		expectStatus   int
		expectError    bool
		description    string
	}{
		{
			name:           "success_delete_notification",
			notificationID: notificationID,
			expectStatus:   http.StatusOK,
			expectError:    false,
			description:    "Should successfully delete notification",
		},
		{
			name:           "not_found_notification",
			notificationID: "non-existent-id",
			expectStatus:   http.StatusNotFound,
			expectError:    true,
			description:    "Should return 404 for non-existent notification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/notifications/%s", tt.notificationID)
			req, _ := http.NewRequest(http.MethodDelete, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)

			if !tt.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response, "success")
				assert.Contains(t, response, "metadata")
				assert.Equal(t, true, response["success"])
			}
		})
	}
}

func TestServiceRegistrar_Integration(t *testing.T) {
	mockService := notificationv1.NewService()
	registrar := NewServiceRegistrar(mockService)
	router := gin.New()
	registrar.RegisterHTTP(router)

	t.Run("complete_notification_workflow", func(t *testing.T) {
		userID := "test-user-integration"

		// 1. Create a notification
		createBody := map[string]interface{}{
			"user_id": userID,
			"title":   "Integration Test Notification",
			"content": "This is an integration test notification",
		}
		bodyBytes, _ := json.Marshal(createBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var createResponse map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &createResponse)
		notification := createResponse["notification"].(map[string]interface{})
		notificationID := notification["id"].(string)

		// 2. Get notifications
		req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/notifications?user_id=%s", userID), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var getResponse map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &getResponse)
		notifications := getResponse["notifications"].([]interface{})
		assert.Len(t, notifications, 1)

		// 3. Mark as read
		req, _ = http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/notifications/%s/read", notificationID), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 4. Delete notification
		req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/notifications/%s", notificationID), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 5. Verify notification is deleted
		req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/notifications?user_id=%s", userID), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		json.Unmarshal(w.Body.Bytes(), &getResponse)
		notifications = getResponse["notifications"].([]interface{})
		assert.Len(t, notifications, 0)
	})
}

func TestServiceRegistrar_ConcurrentAccess(t *testing.T) {
	mockService := notificationv1.NewService()
	registrar := NewServiceRegistrar(mockService)

	const numGoroutines = 10
	results := make([]string, numGoroutines)
	done := make(chan bool, numGoroutines)

	// Test concurrent access to GetName method
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer func() { done <- true }()
			results[index] = registrar.GetName()
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all results are correct
	for i, result := range results {
		assert.Equal(t, "notification", result, "Goroutine %d should return correct service name", i)
	}
}

// Benchmark tests
func BenchmarkServiceRegistrar_GetName(b *testing.B) {
	mockService := notificationv1.NewService()
	registrar := NewServiceRegistrar(mockService)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registrar.GetName()
	}
}

func BenchmarkServiceRegistrar_CreateNotification(b *testing.B) {
	mockService := notificationv1.NewService()
	_ = NewServiceRegistrar(mockService)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test via HTTP handler would be more appropriate for benchmarking
		// But for now, we skip the actual service call in benchmarks
		_ = context.Background()
	}
}

func BenchmarkServiceRegistrar_RegisterHTTP(b *testing.B) {
	mockService := notificationv1.NewService()
	registrar := NewServiceRegistrar(mockService)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router := gin.New()
		_ = registrar.RegisterHTTP(router)
	}
}

func BenchmarkServiceRegistrar_RegisterGRPC(b *testing.B) {
	mockService := notificationv1.NewService()
	registrar := NewServiceRegistrar(mockService)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server := grpc.NewServer()
		_ = registrar.RegisterGRPC(server)
	}
}

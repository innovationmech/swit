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
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	commonv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/common/v1"
	notificationv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/communication/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewGRPCHandler(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "creates_handler_successfully",
			description: "Should create a new gRPC handler successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService()
			handler := NewGRPCHandler(service)

			assert.NotNil(t, handler)
			assert.NotNil(t, handler.service)
			assert.Equal(t, service, handler.service)
		})
	}
}

func TestGRPCHandler_CreateNotification(t *testing.T) {
	tests := []struct {
		name        string
		request     *notificationv1.CreateNotificationRequest
		expectError bool
		errorCode   codes.Code
		description string
	}{
		{
			name: "success_create_notification",
			request: &notificationv1.CreateNotificationRequest{
				UserId:  "test-user-1",
				Title:   "Test Notification",
				Content: "This is a test notification",
				Metadata: &commonv1.RequestMetadata{
					RequestId: uuid.New().String(),
				},
			},
			expectError: false,
			description: "Should successfully create a notification",
		},
		{
			name: "empty_user_id",
			request: &notificationv1.CreateNotificationRequest{
				UserId:  "",
				Title:   "Test Notification",
				Content: "This is a test notification",
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
			description: "Should return InvalidArgument when user_id is empty",
		},
		{
			name: "empty_title",
			request: &notificationv1.CreateNotificationRequest{
				UserId:  "test-user-1",
				Title:   "",
				Content: "This is a test notification",
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
			description: "Should return InvalidArgument when title is empty",
		},
		{
			name: "empty_content",
			request: &notificationv1.CreateNotificationRequest{
				UserId:  "test-user-1",
				Title:   "Test Notification",
				Content: "",
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
			description: "Should return InvalidArgument when content is empty",
		},
		{
			name: "without_metadata",
			request: &notificationv1.CreateNotificationRequest{
				UserId:  "test-user-1",
				Title:   "Test Notification",
				Content: "This is a test notification",
			},
			expectError: false,
			description: "Should successfully create notification without metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService()
			handler := NewGRPCHandler(service)
			ctx := context.Background()

			response, err := handler.CreateNotification(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, response)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.errorCode, st.Code())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotNil(t, response.Notification)
				assert.NotNil(t, response.Metadata)

				notification := response.Notification
				assert.NotEmpty(t, notification.Id)
				assert.Equal(t, tt.request.UserId, notification.UserId)
				assert.Equal(t, tt.request.Title, notification.Title)
				assert.Equal(t, tt.request.Content, notification.Content)
				assert.Equal(t, notificationv1.NotificationType_NOTIFICATION_TYPE_INFO, notification.Type)
				assert.Equal(t, notificationv1.NotificationPriority_NOTIFICATION_PRIORITY_MEDIUM, notification.Priority)
				assert.False(t, notification.IsRead)
				assert.NotNil(t, notification.CreatedAt)
				assert.NotNil(t, notification.UpdatedAt)

				metadata := response.Metadata
				assert.NotEmpty(t, metadata.RequestId)
				assert.Equal(t, "swit-serve-1", metadata.ServerId)
			}
		})
	}
}

func TestGRPCHandler_GetNotifications(t *testing.T) {
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()
	userID := "test-user-1"

	// Create test notifications
	for i := 0; i < 5; i++ {
		_, err := service.CreateNotification(ctx, userID, "Test Title", "Test Content")
		require.NoError(t, err)
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	tests := []struct {
		name        string
		request     *notificationv1.GetNotificationsRequest
		expectError bool
		errorCode   codes.Code
		expectCount int
		description string
	}{
		{
			name: "success_get_notifications",
			request: &notificationv1.GetNotificationsRequest{
				UserId: userID,
				Pagination: &commonv1.PaginationRequest{
					PageSize: 10,
					Offset:   0,
				},
				Metadata: &commonv1.RequestMetadata{
					RequestId: uuid.New().String(),
				},
			},
			expectError: false,
			expectCount: 5,
			description: "Should successfully get notifications",
		},
		{
			name: "empty_user_id",
			request: &notificationv1.GetNotificationsRequest{
				UserId: "",
				Pagination: &commonv1.PaginationRequest{
					PageSize: 10,
					Offset:   0,
				},
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
			description: "Should return InvalidArgument when user_id is empty",
		},
		{
			name: "with_limit",
			request: &notificationv1.GetNotificationsRequest{
				UserId: userID,
				Pagination: &commonv1.PaginationRequest{
					PageSize: 3,
					Offset:   0,
				},
			},
			expectError: false,
			expectCount: 3,
			description: "Should respect pagination limit",
		},
		{
			name: "with_offset",
			request: &notificationv1.GetNotificationsRequest{
				UserId: userID,
				Pagination: &commonv1.PaginationRequest{
					PageSize: 10,
					Offset:   2,
				},
			},
			expectError: false,
			expectCount: 3,
			description: "Should respect pagination offset",
		},
		{
			name: "non_existent_user",
			request: &notificationv1.GetNotificationsRequest{
				UserId: "non-existent-user",
				Pagination: &commonv1.PaginationRequest{
					PageSize: 10,
					Offset:   0,
				},
			},
			expectError: false,
			expectCount: 0,
			description: "Should return empty result for non-existent user",
		},
		{
			name: "without_pagination",
			request: &notificationv1.GetNotificationsRequest{
				UserId: userID,
			},
			expectError: false,
			expectCount: 5,
			description: "Should work without pagination (use defaults)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.GetNotifications(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, response)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.errorCode, st.Code())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotNil(t, response.Notifications)
				assert.NotNil(t, response.Pagination)
				assert.NotNil(t, response.Metadata)

				assert.Len(t, response.Notifications, tt.expectCount)

				// Verify pagination response
				pagination := response.Pagination
				assert.Equal(t, int64(tt.expectCount), pagination.TotalCount)
				assert.NotNil(t, pagination.PageSize)
				assert.NotNil(t, pagination.Offset)

				// Verify metadata
				metadata := response.Metadata
				assert.NotEmpty(t, metadata.RequestId)
				assert.Equal(t, "swit-serve-1", metadata.ServerId)

				// Verify notifications structure
				for _, notification := range response.Notifications {
					assert.NotEmpty(t, notification.Id)
					assert.Equal(t, tt.request.UserId, notification.UserId)
					assert.NotEmpty(t, notification.Title)
					assert.NotEmpty(t, notification.Content)
					assert.Equal(t, notificationv1.NotificationType_NOTIFICATION_TYPE_INFO, notification.Type)
					assert.Equal(t, notificationv1.NotificationPriority_NOTIFICATION_PRIORITY_MEDIUM, notification.Priority)
					assert.NotNil(t, notification.CreatedAt)
					assert.NotNil(t, notification.UpdatedAt)
				}
			}
		})
	}
}

func TestGRPCHandler_MarkAsRead(t *testing.T) {
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()
	userID := "test-user-1"

	// Create a test notification
	notification, err := service.CreateNotification(ctx, userID, "Test Title", "Test Content")
	require.NoError(t, err)

	tests := []struct {
		name           string
		request        *notificationv1.MarkAsReadRequest
		expectError    bool
		errorCode      codes.Code
		description    string
		setupDeletedID bool
	}{
		{
			name: "success_mark_as_read",
			request: &notificationv1.MarkAsReadRequest{
				NotificationId: notification.ID,
				Metadata: &commonv1.RequestMetadata{
					RequestId: uuid.New().String(),
				},
			},
			expectError: false,
			description: "Should successfully mark notification as read",
		},
		{
			name: "empty_notification_id",
			request: &notificationv1.MarkAsReadRequest{
				NotificationId: "",
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
			description: "Should return InvalidArgument when notification_id is empty",
		},
		{
			name: "non_existent_notification",
			request: &notificationv1.MarkAsReadRequest{
				NotificationId: "non-existent-id",
			},
			expectError: true,
			errorCode:   codes.NotFound,
			description: "Should return NotFound for non-existent notification",
		},
		{
			name: "without_metadata",
			request: &notificationv1.MarkAsReadRequest{
				NotificationId: notification.ID,
			},
			expectError: false,
			description: "Should successfully mark as read without metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.MarkAsRead(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, response)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.errorCode, st.Code())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotNil(t, response.Metadata)

				// Verify metadata
				metadata := response.Metadata
				assert.NotEmpty(t, metadata.RequestId)
				assert.Equal(t, "swit-serve-1", metadata.ServerId)

				// Note: Cannot directly verify internal state due to interface abstraction
				// In a real implementation, you would verify via another service method
			}
		})
	}
}

func TestGRPCHandler_DeleteNotification(t *testing.T) {
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()
	userID := "test-user-1"

	// Create a test notification
	notification, err := service.CreateNotification(ctx, userID, "Test Title", "Test Content")
	require.NoError(t, err)

	tests := []struct {
		name        string
		request     *notificationv1.DeleteNotificationRequest
		expectError bool
		errorCode   codes.Code
		description string
	}{
		{
			name: "success_delete_notification",
			request: &notificationv1.DeleteNotificationRequest{
				NotificationId: notification.ID,
				Metadata: &commonv1.RequestMetadata{
					RequestId: uuid.New().String(),
				},
			},
			expectError: false,
			description: "Should successfully delete notification",
		},
		{
			name: "empty_notification_id",
			request: &notificationv1.DeleteNotificationRequest{
				NotificationId: "",
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
			description: "Should return InvalidArgument when notification_id is empty",
		},
		{
			name: "non_existent_notification",
			request: &notificationv1.DeleteNotificationRequest{
				NotificationId: "non-existent-id",
			},
			expectError: true,
			errorCode:   codes.NotFound,
			description: "Should return NotFound for non-existent notification",
		},
		{
			name: "without_metadata",
			request: &notificationv1.DeleteNotificationRequest{
				NotificationId: notification.ID,
			},
			expectError: false,
			description: "Should successfully delete without metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh notification for each test (except the first success test)
			if tt.name != "success_delete_notification" {
				freshNotification, err := service.CreateNotification(ctx, userID, "Test Title", "Test Content")
				require.NoError(t, err)
				if tt.name != "empty_notification_id" && tt.name != "non_existent_notification" {
					tt.request.NotificationId = freshNotification.ID
				}
			}

			response, err := handler.DeleteNotification(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, response)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.errorCode, st.Code())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.True(t, response.Success)
				assert.NotNil(t, response.Metadata)

				// Verify metadata
				metadata := response.Metadata
				assert.NotEmpty(t, metadata.RequestId)
				assert.Equal(t, "swit-serve-1", metadata.ServerId)

				// Note: Cannot directly verify internal state due to interface abstraction
				// In a real implementation, you would verify deletion via another service method
			}
		})
	}
}

func TestGRPCHandler_StreamNotifications(t *testing.T) {
	service := NewService()
	handler := NewGRPCHandler(service)
	userID := "test-user-1"

	tests := []struct {
		name        string
		request     *notificationv1.StreamNotificationsRequest
		expectError bool
		errorCode   codes.Code
		description string
	}{
		{
			name: "empty_user_id",
			request: &notificationv1.StreamNotificationsRequest{
				UserId: "",
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
			description: "Should return InvalidArgument when user_id is empty",
		},
		{
			name: "unimplemented_stream",
			request: &notificationv1.StreamNotificationsRequest{
				UserId: userID,
			},
			expectError: true,
			errorCode:   codes.Unimplemented,
			description: "Should return Unimplemented for streaming",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock stream (this would be a real stream in actual implementation)
			err := handler.StreamNotifications(tt.request, nil)

			if tt.expectError {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.errorCode, st.Code())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGRPCHandler_ExtractRequestIDFromMetadata(t *testing.T) {
	tests := []struct {
		name            string
		metadata        *commonv1.RequestMetadata
		expectGenerated bool
		description     string
	}{
		{
			name: "with_request_id",
			metadata: &commonv1.RequestMetadata{
				RequestId: "test-request-id",
			},
			expectGenerated: false,
			description:     "Should use provided request ID",
		},
		{
			name: "without_request_id",
			metadata: &commonv1.RequestMetadata{
				RequestId: "",
			},
			expectGenerated: true,
			description:     "Should generate new request ID when empty",
		},
		{
			name:            "nil_metadata",
			metadata:        nil,
			expectGenerated: true,
			description:     "Should generate new request ID when metadata is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestID := extractRequestIDFromMetadata(tt.metadata)

			assert.NotEmpty(t, requestID)

			if tt.expectGenerated {
				// Should be a valid UUID
				_, err := uuid.Parse(requestID)
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tt.metadata.RequestId, requestID)
			}
		})
	}
}

func TestGRPCHandler_Integration(t *testing.T) {
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()
	userID := "test-user-integration"

	t.Run("complete_notification_grpc_workflow", func(t *testing.T) {
		// 1. Create notification
		createReq := &notificationv1.CreateNotificationRequest{
			UserId:  userID,
			Title:   "Integration Test",
			Content: "This is an integration test notification",
			Metadata: &commonv1.RequestMetadata{
				RequestId: uuid.New().String(),
			},
		}

		createResp, err := handler.CreateNotification(ctx, createReq)
		require.NoError(t, err)
		assert.NotNil(t, createResp)

		notificationID := createResp.Notification.Id

		// 2. Get notifications
		getReq := &notificationv1.GetNotificationsRequest{
			UserId: userID,
			Pagination: &commonv1.PaginationRequest{
				PageSize: 10,
				Offset:   0,
			},
		}

		getResp, err := handler.GetNotifications(ctx, getReq)
		require.NoError(t, err)
		assert.Len(t, getResp.Notifications, 1)
		assert.Equal(t, notificationID, getResp.Notifications[0].Id)
		assert.False(t, getResp.Notifications[0].IsRead)

		// 3. Mark as read
		markReq := &notificationv1.MarkAsReadRequest{
			NotificationId: notificationID,
		}

		markResp, err := handler.MarkAsRead(ctx, markReq)
		require.NoError(t, err)
		assert.NotNil(t, markResp)

		// 4. Verify it's marked as read
		getResp, err = handler.GetNotifications(ctx, getReq)
		require.NoError(t, err)
		assert.Len(t, getResp.Notifications, 1)
		assert.True(t, getResp.Notifications[0].IsRead)

		// 5. Delete notification
		deleteReq := &notificationv1.DeleteNotificationRequest{
			NotificationId: notificationID,
		}

		deleteResp, err := handler.DeleteNotification(ctx, deleteReq)
		require.NoError(t, err)
		assert.True(t, deleteResp.Success)

		// 6. Verify it's deleted
		getResp, err = handler.GetNotifications(ctx, getReq)
		require.NoError(t, err)
		assert.Len(t, getResp.Notifications, 0)
	})
}

func TestGRPCHandler_ErrorHandling(t *testing.T) {
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()

	t.Run("service_errors_are_handled", func(t *testing.T) {
		// This test ensures that business logic errors are properly converted to gRPC errors

		// Test create with invalid data (empty fields)
		createReq := &notificationv1.CreateNotificationRequest{
			UserId:  "",
			Title:   "Test",
			Content: "Test",
		}

		_, err := handler.CreateNotification(ctx, createReq)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())

		// Test get with invalid data
		getReq := &notificationv1.GetNotificationsRequest{
			UserId: "",
		}

		_, err = handler.GetNotifications(ctx, getReq)
		assert.Error(t, err)
		st, ok = status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())

		// Test mark as read with invalid ID
		markReq := &notificationv1.MarkAsReadRequest{
			NotificationId: "",
		}

		_, err = handler.MarkAsRead(ctx, markReq)
		assert.Error(t, err)
		st, ok = status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())

		// Test delete with invalid ID
		deleteReq := &notificationv1.DeleteNotificationRequest{
			NotificationId: "",
		}

		_, err = handler.DeleteNotification(ctx, deleteReq)
		assert.Error(t, err)
		st, ok = status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

// Benchmark tests
func BenchmarkGRPCHandler_CreateNotification(b *testing.B) {
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()

	req := &notificationv1.CreateNotificationRequest{
		UserId:  "test-user",
		Title:   "Test Title",
		Content: "Test Content",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.CreateNotification(ctx, req)
	}
}

func BenchmarkGRPCHandler_GetNotifications(b *testing.B) {
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()
	userID := "test-user"

	// Pre-populate with notifications
	for i := 0; i < 100; i++ {
		_, _ = service.CreateNotification(ctx, userID, "Test Title", "Test Content")
	}

	req := &notificationv1.GetNotificationsRequest{
		UserId: userID,
		Pagination: &commonv1.PaginationRequest{
			PageSize: 20,
			Offset:   0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.GetNotifications(ctx, req)
	}
}

func BenchmarkGRPCHandler_MarkAsRead(b *testing.B) {
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()
	userID := "test-user"

	// Pre-populate with notifications
	notifications := make([]*Notification, b.N)
	for i := 0; i < b.N; i++ {
		notification, _ := service.CreateNotification(ctx, userID, "Test Title", "Test Content")
		notifications[i] = notification
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &notificationv1.MarkAsReadRequest{
			NotificationId: notifications[i].ID,
		}
		_, _ = handler.MarkAsRead(ctx, req)
	}
}

func BenchmarkGRPCHandler_DeleteNotification(b *testing.B) {
	service := NewService()
	handler := NewGRPCHandler(service)
	ctx := context.Background()
	userID := "test-user"

	// Pre-populate with notifications
	notifications := make([]*Notification, b.N)
	for i := 0; i < b.N; i++ {
		notification, _ := service.CreateNotification(ctx, userID, "Test Title", "Test Content")
		notifications[i] = notification
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &notificationv1.DeleteNotificationRequest{
			NotificationId: notifications[i].ID,
		}
		_, _ = handler.DeleteNotification(ctx, req)
	}
}

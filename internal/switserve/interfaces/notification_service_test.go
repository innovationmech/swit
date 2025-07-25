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

package interfaces

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/innovationmech/swit/internal/switserve/types"
)

// MockNotificationService is a mock implementation of NotificationService for testing
type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) SendNotification(ctx context.Context, request *NotificationRequest) (string, error) {
	args := m.Called(ctx, request)
	return args.String(0), args.Error(1)
}

func (m *MockNotificationService) SendBulkNotifications(ctx context.Context, requests []*NotificationRequest) (map[string]*NotificationStatus, error) {
	args := m.Called(ctx, requests)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*NotificationStatus), args.Error(1)
}

func (m *MockNotificationService) GetNotificationStatus(ctx context.Context, notificationID string) (*NotificationStatus, error) {
	args := m.Called(ctx, notificationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*NotificationStatus), args.Error(1)
}

func (m *MockNotificationService) GetUserNotifications(ctx context.Context, userID string, pagination *types.PaginationRequest) (*types.PaginatedResponse, error) {
	args := m.Called(ctx, userID, pagination)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.PaginatedResponse), args.Error(1)
}

func (m *MockNotificationService) MarkAsRead(ctx context.Context, notificationID string, userID string) error {
	args := m.Called(ctx, notificationID, userID)
	return args.Error(0)
}

func (m *MockNotificationService) DeleteNotification(ctx context.Context, notificationID string) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockNotificationService) GetDeliveryStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func TestNotificationService_Interface(t *testing.T) {
	// Test that MockNotificationService implements NotificationService interface
	var _ NotificationService = (*MockNotificationService)(nil)
}

func TestNotificationChannel_Constants(t *testing.T) {
	assert.Equal(t, NotificationChannel("email"), ChannelEmail)
	assert.Equal(t, NotificationChannel("sms"), ChannelSMS)
	assert.Equal(t, NotificationChannel("push"), ChannelPush)
	assert.Equal(t, NotificationChannel("webhook"), ChannelWebhook)
}

func TestNotificationPriority_Constants(t *testing.T) {
	assert.Equal(t, NotificationPriority("low"), PriorityLow)
	assert.Equal(t, NotificationPriority("normal"), PriorityNormal)
	assert.Equal(t, NotificationPriority("high"), PriorityHigh)
	assert.Equal(t, NotificationPriority("critical"), PriorityCritical)
}

func TestNotificationRequest_Structure(t *testing.T) {
	request := &NotificationRequest{
		ID:          "notif-123",
		RecipientID: "user-456",
		Channel:     ChannelEmail,
		Priority:    PriorityHigh,
		Subject:     "Test Notification",
		Content:     "This is a test notification",
		Metadata: map[string]interface{}{
			"template_id": "welcome",
			"language":    "en",
		},
	}

	assert.Equal(t, "notif-123", request.ID)
	assert.Equal(t, "user-456", request.RecipientID)
	assert.Equal(t, ChannelEmail, request.Channel)
	assert.Equal(t, PriorityHigh, request.Priority)
	assert.Equal(t, "Test Notification", request.Subject)
	assert.Equal(t, "This is a test notification", request.Content)
	assert.Equal(t, "welcome", request.Metadata["template_id"])
}

func TestNotificationStatus_Structure(t *testing.T) {
	deliveredAt := time.Now().Unix()
	status := &NotificationStatus{
		ID:            "notif-123",
		Status:        types.StatusActive,
		DeliveredAt:   &deliveredAt,
		FailureReason: "",
		RetryCount:    0,
	}

	assert.Equal(t, "notif-123", status.ID)
	assert.Equal(t, types.StatusActive, status.Status)
	assert.Equal(t, deliveredAt, *status.DeliveredAt)
	assert.Empty(t, status.FailureReason)
	assert.Equal(t, 0, status.RetryCount)
}

func TestMockNotificationService_SendNotification(t *testing.T) {
	mockService := new(MockNotificationService)
	ctx := context.Background()
	request := &NotificationRequest{
		ID:          "notif-123",
		RecipientID: "user-456",
		Channel:     ChannelEmail,
		Priority:    PriorityNormal,
		Subject:     "Test",
		Content:     "Test content",
	}

	// Test successful send
	mockService.On("SendNotification", ctx, request).Return("notif-123", nil)
	id, err := mockService.SendNotification(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, "notif-123", id)
	mockService.AssertExpectations(t)

	// Test send failure
	mockService = new(MockNotificationService)
	expectedErr := errors.New("invalid recipient")
	mockService.On("SendNotification", ctx, request).Return("", expectedErr)
	id, err = mockService.SendNotification(ctx, request)
	assert.Error(t, err)
	assert.Empty(t, id)
	assert.Equal(t, expectedErr, err)
	mockService.AssertExpectations(t)
}

func TestMockNotificationService_GetNotificationStatus(t *testing.T) {
	mockService := new(MockNotificationService)
	ctx := context.Background()
	notificationID := "notif-123"
	deliveredAt := time.Now().Unix()

	expectedStatus := &NotificationStatus{
		ID:          notificationID,
		Status:      types.StatusActive,
		DeliveredAt: &deliveredAt,
		RetryCount:  0,
	}

	// Test successful retrieval
	mockService.On("GetNotificationStatus", ctx, notificationID).Return(expectedStatus, nil)
	status, err := mockService.GetNotificationStatus(ctx, notificationID)
	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, status)
	mockService.AssertExpectations(t)

	// Test not found
	mockService = new(MockNotificationService)
	expectedErr := errors.New("notification not found")
	mockService.On("GetNotificationStatus", ctx, notificationID).Return(nil, expectedErr)
	status, err = mockService.GetNotificationStatus(ctx, notificationID)
	assert.Error(t, err)
	assert.Nil(t, status)
	assert.Equal(t, expectedErr, err)
	mockService.AssertExpectations(t)
}

func TestMockNotificationService_MarkAsRead(t *testing.T) {
	mockService := new(MockNotificationService)
	ctx := context.Background()
	notificationID := "notif-123"
	userID := "user-456"

	// Test successful mark as read
	mockService.On("MarkAsRead", ctx, notificationID, userID).Return(nil)
	err := mockService.MarkAsRead(ctx, notificationID, userID)
	assert.NoError(t, err)
	mockService.AssertExpectations(t)

	// Test failure
	mockService = new(MockNotificationService)
	expectedErr := errors.New("notification not found")
	mockService.On("MarkAsRead", ctx, notificationID, userID).Return(expectedErr)
	err = mockService.MarkAsRead(ctx, notificationID, userID)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockService.AssertExpectations(t)
}

func TestMockNotificationService_DeleteNotification(t *testing.T) {
	mockService := new(MockNotificationService)
	ctx := context.Background()
	notificationID := "notif-123"

	// Test successful deletion
	mockService.On("DeleteNotification", ctx, notificationID).Return(nil)
	err := mockService.DeleteNotification(ctx, notificationID)
	assert.NoError(t, err)
	mockService.AssertExpectations(t)

	// Test deletion failure
	mockService = new(MockNotificationService)
	expectedErr := errors.New("notification not found")
	mockService.On("DeleteNotification", ctx, notificationID).Return(expectedErr)
	err = mockService.DeleteNotification(ctx, notificationID)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockService.AssertExpectations(t)
}

func TestMockNotificationService_GetDeliveryStats(t *testing.T) {
	mockService := new(MockNotificationService)
	ctx := context.Background()

	expectedStats := map[string]interface{}{
		"total_sent":        1000,
		"success_rate":      0.95,
		"failure_rate":      0.05,
		"avg_delivery_time": "2.5s",
		"channels": map[string]interface{}{
			"email": map[string]interface{}{
				"sent":         800,
				"success_rate": 0.98,
			},
			"sms": map[string]interface{}{
				"sent":         200,
				"success_rate": 0.85,
			},
		},
	}

	// Test successful stats retrieval
	mockService.On("GetDeliveryStats", ctx).Return(expectedStats, nil)
	stats, err := mockService.GetDeliveryStats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedStats, stats)
	assert.Equal(t, 1000, stats["total_sent"])
	assert.Equal(t, 0.95, stats["success_rate"])
	mockService.AssertExpectations(t)

	// Test stats retrieval failure
	mockService = new(MockNotificationService)
	expectedErr := errors.New("stats unavailable")
	mockService.On("GetDeliveryStats", ctx).Return(nil, expectedErr)
	stats, err = mockService.GetDeliveryStats(ctx)
	assert.Error(t, err)
	assert.Nil(t, stats)
	assert.Equal(t, expectedErr, err)
	mockService.AssertExpectations(t)
}

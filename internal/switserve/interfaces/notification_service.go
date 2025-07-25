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

	"github.com/innovationmech/swit/internal/switserve/types"
)

// NotificationChannel represents different notification delivery channels
type NotificationChannel string

const (
	// ChannelEmail represents email notification delivery
	ChannelEmail NotificationChannel = "email"
	// ChannelSMS represents SMS notification delivery
	ChannelSMS NotificationChannel = "sms"
	// ChannelPush represents push notification delivery
	ChannelPush NotificationChannel = "push"
	// ChannelWebhook represents webhook notification delivery
	ChannelWebhook NotificationChannel = "webhook"
)

// NotificationPriority represents the priority level of notifications
type NotificationPriority string

const (
	// PriorityLow represents low priority notifications
	PriorityLow NotificationPriority = "low"
	// PriorityNormal represents normal priority notifications
	PriorityNormal NotificationPriority = "normal"
	// PriorityHigh represents high priority notifications
	PriorityHigh NotificationPriority = "high"
	// PriorityCritical represents critical priority notifications
	PriorityCritical NotificationPriority = "critical"
)

// NotificationRequest represents a notification to be sent
type NotificationRequest struct {
	// ID is a unique identifier for the notification
	ID string `json:"id"`
	// RecipientID is the ID of the user receiving the notification
	RecipientID string `json:"recipient_id"`
	// Channel specifies how the notification should be delivered
	Channel NotificationChannel `json:"channel"`
	// Priority indicates the urgency of the notification
	Priority NotificationPriority `json:"priority"`
	// Subject is the notification subject/title
	Subject string `json:"subject"`
	// Content is the main notification message
	Content string `json:"content"`
	// Metadata contains additional notification data
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationStatus represents the delivery status of a notification
type NotificationStatus struct {
	// ID is the notification identifier
	ID string `json:"id"`
	// Status indicates the current delivery status
	Status types.Status `json:"status"`
	// DeliveredAt is the timestamp when the notification was delivered
	DeliveredAt *int64 `json:"delivered_at,omitempty"`
	// FailureReason contains error details if delivery failed
	FailureReason string `json:"failure_reason,omitempty"`
	// RetryCount indicates how many delivery attempts have been made
	RetryCount int `json:"retry_count"`
}

// NotificationService defines the interface for notification and messaging operations.
// This interface provides methods for sending notifications through various channels
// and tracking their delivery status.
//
// Future implementations should support:
// - Multiple delivery channels (email, SMS, push, webhook)
// - Priority-based delivery
// - Retry mechanisms for failed deliveries
// - Delivery status tracking
// - Template-based notifications
// - Bulk notification sending
type NotificationService interface {
	// SendNotification sends a single notification through the specified channel.
	// It validates the notification request and queues it for delivery.
	// Returns the notification ID for tracking purposes or an error if validation fails.
	SendNotification(ctx context.Context, request *NotificationRequest) (string, error)

	// SendBulkNotifications sends multiple notifications efficiently.
	// It processes notifications in batches and returns a map of notification IDs
	// to their initial status or any errors encountered.
	SendBulkNotifications(ctx context.Context, requests []*NotificationRequest) (map[string]*NotificationStatus, error)

	// GetNotificationStatus retrieves the current delivery status of a notification.
	// Returns the status information or an error if the notification is not found.
	GetNotificationStatus(ctx context.Context, notificationID string) (*NotificationStatus, error)

	// GetUserNotifications retrieves all notifications for a specific user.
	// Supports pagination and filtering by status, channel, or priority.
	// Returns a paginated list of notifications or an error.
	GetUserNotifications(ctx context.Context, userID string, pagination *types.PaginationRequest) (*types.PaginatedResponse, error)

	// MarkAsRead marks a notification as read by the user.
	// This is useful for tracking user engagement with notifications.
	// Returns an error if the notification is not found or cannot be updated.
	MarkAsRead(ctx context.Context, notificationID string, userID string) error

	// DeleteNotification removes a notification from the system.
	// This should be used carefully as it permanently removes notification history.
	// Returns an error if the notification is not found or cannot be deleted.
	DeleteNotification(ctx context.Context, notificationID string) error

	// GetDeliveryStats returns statistics about notification delivery performance.
	// This includes success rates, failure reasons, and delivery times by channel.
	// Useful for monitoring and optimizing notification systems.
	GetDeliveryStats(ctx context.Context) (map[string]interface{}, error)
}

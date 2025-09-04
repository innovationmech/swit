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
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/tracing"
	"go.uber.org/zap"
)

// Notification represents a notification entity
type Notification struct {
	ID        string
	UserID    string
	Title     string
	Content   string
	IsRead    bool
	CreatedAt int64
	UpdatedAt int64
}

// NotificationService defines the interface for notification business logic
type NotificationService interface {
	CreateNotification(ctx context.Context, userID, title, content string) (*Notification, error)
	GetNotifications(ctx context.Context, userID string, limit, offset int) ([]*Notification, error)
	MarkAsRead(ctx context.Context, notificationID string) error
	DeleteNotification(ctx context.Context, notificationID string) error
}

// Service implements the notification business logic
type Service struct {
	// In-memory storage for demo purposes
	// In production, this would be replaced with a database
	notifications  map[string]*Notification
	userNotifs     map[string][]string // userID -> []notificationID
	mu             sync.RWMutex
	tracingManager tracing.TracingManager
}

// NewService creates a new notification service implementation
func NewService() NotificationService {
	return &Service{
		notifications: make(map[string]*Notification),
		userNotifs:    make(map[string][]string),
	}
}

// NewServiceWithTracing creates a new notification service implementation with tracing
func NewServiceWithTracing(tracingManager tracing.TracingManager) NotificationService {
	return &Service{
		notifications:  make(map[string]*Notification),
		userNotifs:     make(map[string][]string),
		tracingManager: tracingManager,
	}
}

// CreateNotification creates a new notification
func (s *Service) CreateNotification(ctx context.Context, userID, title, content string) (*Notification, error) {
	// Create tracing span
	var span tracing.Span
	if s.tracingManager != nil {
		ctx, span = s.tracingManager.StartSpan(ctx, "NotificationService.CreateNotification",
			tracing.WithAttributes(
				attribute.String("notification.user_id", userID),
				attribute.String("notification.title", title),
			),
		)
		defer span.End()
	}

	// Validation span
	if s.tracingManager != nil {
		_, validationSpan := s.tracingManager.StartSpan(ctx, "validate_notification_input")
		if userID == "" {
			validationSpan.SetStatus(codes.Error, "userID cannot be empty")
			span.SetStatus(codes.Error, "validation failed")
			validationSpan.End()
			return nil, fmt.Errorf("userID cannot be empty")
		}
		if title == "" {
			validationSpan.SetStatus(codes.Error, "title cannot be empty")
			span.SetStatus(codes.Error, "validation failed")
			validationSpan.End()
			return nil, fmt.Errorf("title cannot be empty")
		}
		if content == "" {
			validationSpan.SetStatus(codes.Error, "content cannot be empty")
			span.SetStatus(codes.Error, "validation failed")
			validationSpan.End()
			return nil, fmt.Errorf("content cannot be empty")
		}
		validationSpan.End()
	} else {
		// Fallback validation if no tracing
		if userID == "" {
			return nil, fmt.Errorf("userID cannot be empty")
		}
		if title == "" {
			return nil, fmt.Errorf("title cannot be empty")
		}
		if content == "" {
			return nil, fmt.Errorf("content cannot be empty")
		}
	}

	// Storage operation span
	if s.tracingManager != nil {
		_, storageSpan := s.tracingManager.StartSpan(ctx, "store_notification")
		defer storageSpan.End()

		s.mu.Lock()
		defer s.mu.Unlock()

		now := time.Now().Unix()
		notification := &Notification{
			ID:        uuid.New().String(),
			UserID:    userID,
			Title:     title,
			Content:   content,
			IsRead:    false,
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Store notification
		s.notifications[notification.ID] = notification

		// Add to user's notification list
		s.userNotifs[userID] = append(s.userNotifs[userID], notification.ID)

		storageSpan.SetAttribute("notification.id", notification.ID)
		storageSpan.SetAttribute("operation.success", true)

		if span != nil {
			span.SetAttribute("notification.id", notification.ID)
			span.SetAttribute("operation.success", true)
		}

		logger.Logger.Info("Notification created",
			zap.String("notification_id", notification.ID),
			zap.String("user_id", userID),
			zap.String("title", title),
		)

		return notification, nil
	} else {
		// Fallback storage without tracing
		s.mu.Lock()
		defer s.mu.Unlock()

		now := time.Now().Unix()
		notification := &Notification{
			ID:        uuid.New().String(),
			UserID:    userID,
			Title:     title,
			Content:   content,
			IsRead:    false,
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Store notification
		s.notifications[notification.ID] = notification

		// Add to user's notification list
		s.userNotifs[userID] = append(s.userNotifs[userID], notification.ID)

		logger.Logger.Info("Notification created",
			zap.String("notification_id", notification.ID),
			zap.String("user_id", userID),
			zap.String("title", title),
		)

		return notification, nil
	}
}

// GetNotifications retrieves notifications for a user
func (s *Service) GetNotifications(ctx context.Context, userID string, limit, offset int) ([]*Notification, error) {
	// Create tracing span
	var span tracing.Span
	if s.tracingManager != nil {
		ctx, span = s.tracingManager.StartSpan(ctx, "NotificationService.GetNotifications",
			tracing.WithAttributes(
				attribute.String("notification.user_id", userID),
				attribute.Int("notification.limit", limit),
				attribute.Int("notification.offset", offset),
			),
		)
		defer span.End()
	}

	// Validation span
	if s.tracingManager != nil {
		_, validationSpan := s.tracingManager.StartSpan(ctx, "validate_get_notifications_input")
		if userID == "" {
			validationSpan.SetStatus(codes.Error, "userID cannot be empty")
			span.SetStatus(codes.Error, "validation failed")
			validationSpan.End()
			return nil, fmt.Errorf("userID cannot be empty")
		}
		validationSpan.End()
	} else {
		if userID == "" {
			return nil, fmt.Errorf("userID cannot be empty")
		}
	}

	if limit <= 0 {
		limit = 20 // default limit
	}
	if offset < 0 {
		offset = 0
	}

	// Retrieval operation span
	if s.tracingManager != nil {
		_, retrievalSpan := s.tracingManager.StartSpan(ctx, "retrieve_notifications")
		defer retrievalSpan.End()

		s.mu.RLock()
		defer s.mu.RUnlock()

		userNotificationIDs, exists := s.userNotifs[userID]
		if !exists {
			retrievalSpan.SetAttribute("notifications.found", 0)
			if span != nil {
				span.SetAttribute("notifications.found", 0)
				span.SetAttribute("operation.success", true)
			}
			return []*Notification{}, nil
		}

		// Apply pagination
		start := offset
		if start >= len(userNotificationIDs) {
			retrievalSpan.SetAttribute("notifications.found", 0)
			if span != nil {
				span.SetAttribute("notifications.found", 0)
				span.SetAttribute("operation.success", true)
			}
			return []*Notification{}, nil
		}

		end := start + limit
		if end > len(userNotificationIDs) {
			end = len(userNotificationIDs)
		}

		// Get notifications in reverse order (newest first)
		var notifications []*Notification
		for i := len(userNotificationIDs) - 1 - start; i >= len(userNotificationIDs)-end; i-- {
			if i < 0 {
				break
			}
			notifID := userNotificationIDs[i]
			if notification, exists := s.notifications[notifID]; exists {
				notifications = append(notifications, notification)
			}
		}

		retrievalSpan.SetAttribute("notifications.found", len(notifications))
		retrievalSpan.SetAttribute("notifications.total_available", len(userNotificationIDs))
		retrievalSpan.SetAttribute("operation.success", true)

		if span != nil {
			span.SetAttribute("notifications.found", len(notifications))
			span.SetAttribute("notifications.total_available", len(userNotificationIDs))
			span.SetAttribute("operation.success", true)
		}

		logger.Logger.Debug("Retrieved notifications",
			zap.String("user_id", userID),
			zap.Int("count", len(notifications)),
			zap.Int("limit", limit),
			zap.Int("offset", offset),
		)

		return notifications, nil
	} else {
		// Fallback retrieval without tracing
		s.mu.RLock()
		defer s.mu.RUnlock()

		userNotificationIDs, exists := s.userNotifs[userID]
		if !exists {
			return []*Notification{}, nil
		}

		// Apply pagination
		start := offset
		if start >= len(userNotificationIDs) {
			return []*Notification{}, nil
		}

		end := start + limit
		if end > len(userNotificationIDs) {
			end = len(userNotificationIDs)
		}

		// Get notifications in reverse order (newest first)
		var notifications []*Notification
		for i := len(userNotificationIDs) - 1 - start; i >= len(userNotificationIDs)-end; i-- {
			if i < 0 {
				break
			}
			notifID := userNotificationIDs[i]
			if notification, exists := s.notifications[notifID]; exists {
				notifications = append(notifications, notification)
			}
		}

		logger.Logger.Debug("Retrieved notifications",
			zap.String("user_id", userID),
			zap.Int("count", len(notifications)),
			zap.Int("limit", limit),
			zap.Int("offset", offset),
		)

		return notifications, nil
	}
}

// MarkAsRead marks a notification as read
func (s *Service) MarkAsRead(ctx context.Context, notificationID string) error {
	// Create tracing span
	var span tracing.Span
	if s.tracingManager != nil {
		ctx, span = s.tracingManager.StartSpan(ctx, "NotificationService.MarkAsRead",
			tracing.WithAttributes(
				attribute.String("notification.id", notificationID),
			),
		)
		defer span.End()
	}

	// Validation span
	if s.tracingManager != nil {
		_, validationSpan := s.tracingManager.StartSpan(ctx, "validate_mark_read_input")
		if notificationID == "" {
			validationSpan.SetStatus(codes.Error, "notificationID cannot be empty")
			span.SetStatus(codes.Error, "validation failed")
			validationSpan.End()
			return fmt.Errorf("notificationID cannot be empty")
		}
		validationSpan.End()
	} else {
		if notificationID == "" {
			return fmt.Errorf("notificationID cannot be empty")
		}
	}

	// Update operation span
	if s.tracingManager != nil {
		_, updateSpan := s.tracingManager.StartSpan(ctx, "mark_notification_read")
		defer updateSpan.End()

		s.mu.Lock()
		defer s.mu.Unlock()

		notification, exists := s.notifications[notificationID]
		if !exists {
			updateSpan.SetStatus(codes.Error, "notification not found")
			span.SetStatus(codes.Error, "notification not found")
			updateSpan.End()
			return ErrNotificationNotFound
		}

		wasRead := notification.IsRead
		if !notification.IsRead {
			notification.IsRead = true
			notification.UpdatedAt = time.Now().Unix()

			updateSpan.SetAttribute("notification.was_already_read", false)
			updateSpan.SetAttribute("notification.user_id", notification.UserID)

			logger.Logger.Info("Notification marked as read",
				zap.String("notification_id", notificationID),
				zap.String("user_id", notification.UserID),
			)
		} else {
			updateSpan.SetAttribute("notification.was_already_read", true)
			updateSpan.SetAttribute("notification.user_id", notification.UserID)
		}

		updateSpan.SetAttribute("operation.success", true)
		if span != nil {
			span.SetAttribute("notification.was_already_read", wasRead)
			span.SetAttribute("notification.user_id", notification.UserID)
			span.SetAttribute("operation.success", true)
		}

		return nil
	} else {
		// Fallback update without tracing
		s.mu.Lock()
		defer s.mu.Unlock()

		notification, exists := s.notifications[notificationID]
		if !exists {
			return ErrNotificationNotFound
		}

		if !notification.IsRead {
			notification.IsRead = true
			notification.UpdatedAt = time.Now().Unix()

			logger.Logger.Info("Notification marked as read",
				zap.String("notification_id", notificationID),
				zap.String("user_id", notification.UserID),
			)
		}

		return nil
	}
}

// DeleteNotification deletes a notification
func (s *Service) DeleteNotification(ctx context.Context, notificationID string) error {
	// Create tracing span
	var span tracing.Span
	if s.tracingManager != nil {
		ctx, span = s.tracingManager.StartSpan(ctx, "NotificationService.DeleteNotification",
			tracing.WithAttributes(
				attribute.String("notification.id", notificationID),
			),
		)
		defer span.End()
	}

	// Validation span
	if s.tracingManager != nil {
		_, validationSpan := s.tracingManager.StartSpan(ctx, "validate_delete_notification_input")
		if notificationID == "" {
			validationSpan.SetStatus(codes.Error, "notificationID cannot be empty")
			span.SetStatus(codes.Error, "validation failed")
			validationSpan.End()
			return fmt.Errorf("notificationID cannot be empty")
		}
		validationSpan.End()
	} else {
		if notificationID == "" {
			return fmt.Errorf("notificationID cannot be empty")
		}
	}

	// Delete operation span
	if s.tracingManager != nil {
		_, deleteSpan := s.tracingManager.StartSpan(ctx, "delete_notification")
		defer deleteSpan.End()

		s.mu.Lock()
		defer s.mu.Unlock()

		notification, exists := s.notifications[notificationID]
		if !exists {
			deleteSpan.SetStatus(codes.Error, "notification not found")
			span.SetStatus(codes.Error, "notification not found")
			deleteSpan.End()
			return ErrNotificationNotFound
		}

		userID := notification.UserID

		// Remove from notifications map
		delete(s.notifications, notificationID)

		// Remove from user's notification list
		userNotifs := s.userNotifs[userID]
		for i, id := range userNotifs {
			if id == notificationID {
				s.userNotifs[userID] = append(userNotifs[:i], userNotifs[i+1:]...)
				break
			}
		}

		deleteSpan.SetAttribute("notification.user_id", userID)
		deleteSpan.SetAttribute("notification.title", notification.Title)
		deleteSpan.SetAttribute("operation.success", true)

		if span != nil {
			span.SetAttribute("notification.user_id", userID)
			span.SetAttribute("notification.title", notification.Title)
			span.SetAttribute("operation.success", true)
		}

		logger.Logger.Info("Notification deleted",
			zap.String("notification_id", notificationID),
			zap.String("user_id", userID),
		)

		return nil
	} else {
		// Fallback delete without tracing
		s.mu.Lock()
		defer s.mu.Unlock()

		notification, exists := s.notifications[notificationID]
		if !exists {
			return ErrNotificationNotFound
		}

		// Remove from notifications map
		delete(s.notifications, notificationID)

		// Remove from user's notification list
		userNotifs := s.userNotifs[notification.UserID]
		for i, id := range userNotifs {
			if id == notificationID {
				s.userNotifs[notification.UserID] = append(userNotifs[:i], userNotifs[i+1:]...)
				break
			}
		}

		logger.Logger.Info("Notification deleted",
			zap.String("notification_id", notificationID),
			zap.String("user_id", notification.UserID),
		)

		return nil
	}
}

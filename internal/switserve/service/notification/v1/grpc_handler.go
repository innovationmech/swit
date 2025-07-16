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
	"time"

	"github.com/google/uuid"
	commonv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/common/v1"
	notificationv1 "github.com/innovationmech/swit/api/gen/go/proto/swit/communication/v1"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler implements the gRPC server interface using business logic
type GRPCHandler struct {
	notificationv1.UnimplementedNotificationServiceServer
	service *Service
}

// NewGRPCHandler creates a new gRPC handler with business logic service
func NewGRPCHandler(service *Service) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// CreateNotification implements the CreateNotification RPC
func (h *GRPCHandler) CreateNotification(ctx context.Context, req *notificationv1.CreateNotificationRequest) (*notificationv1.CreateNotificationResponse, error) {
	// Input validation
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	logger.Logger.Info("Processing CreateNotification request",
		zap.String("user_id", req.UserId),
		zap.String("title", req.Title),
	)

	// Call business logic
	notification, err := h.service.CreateNotification(ctx, req.UserId, req.Title, req.Content)
	if err != nil {
		logger.Logger.Error("Failed to create notification", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create notification")
	}

	// Extract request ID from context or metadata
	requestID := extractRequestIDFromMetadata(req.GetMetadata())

	// Convert business model to proto message
	protoNotification := &notificationv1.Notification{
		Id:        notification.ID,
		UserId:    notification.UserID,
		Title:     notification.Title,
		Content:   notification.Content,
		Type:      notificationv1.NotificationType_NOTIFICATION_TYPE_INFO,           // Default type
		Priority:  notificationv1.NotificationPriority_NOTIFICATION_PRIORITY_MEDIUM, // Default priority
		IsRead:    notification.IsRead,
		CreatedAt: timestamppb.New(time.Unix(notification.CreatedAt, 0)),
		UpdatedAt: timestamppb.New(time.Unix(notification.UpdatedAt, 0)),
	}

	// Build response
	response := &notificationv1.CreateNotificationResponse{
		Notification: protoNotification,
		Metadata: &commonv1.ResponseMetadata{
			RequestId:        requestID,
			ServerId:         "swit-serve-1",
			ProcessingTimeMs: 0, // TODO: Calculate actual processing time
		},
	}

	logger.Logger.Info("CreateNotification request completed successfully",
		zap.String("notification_id", notification.ID),
		zap.String("user_id", req.UserId),
		zap.String("request_id", requestID),
	)

	return response, nil
}

// GetNotifications implements the GetNotifications RPC
func (h *GRPCHandler) GetNotifications(ctx context.Context, req *notificationv1.GetNotificationsRequest) (*notificationv1.GetNotificationsResponse, error) {
	// Input validation
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	limit := int(req.GetPagination().GetPageSize())
	offset := int(req.GetPagination().GetOffset())

	logger.Logger.Info("Processing GetNotifications request",
		zap.String("user_id", req.UserId),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	// Call business logic
	notifications, err := h.service.GetNotifications(ctx, req.UserId, limit, offset)
	if err != nil {
		logger.Logger.Error("Failed to get notifications", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get notifications")
	}

	// Convert business models to proto messages
	protoNotifications := make([]*notificationv1.Notification, len(notifications))
	for i, notification := range notifications {
		protoNotifications[i] = &notificationv1.Notification{
			Id:        notification.ID,
			UserId:    notification.UserID,
			Title:     notification.Title,
			Content:   notification.Content,
			Type:      notificationv1.NotificationType_NOTIFICATION_TYPE_INFO,
			Priority:  notificationv1.NotificationPriority_NOTIFICATION_PRIORITY_MEDIUM,
			IsRead:    notification.IsRead,
			CreatedAt: timestamppb.New(time.Unix(notification.CreatedAt, 0)),
			UpdatedAt: timestamppb.New(time.Unix(notification.UpdatedAt, 0)),
		}
	}

	// Extract request ID
	requestID := extractRequestIDFromMetadata(req.GetMetadata())

	// Build response
	response := &notificationv1.GetNotificationsResponse{
		Notifications: protoNotifications,
		Pagination: &commonv1.PaginationResponse{
			TotalCount:    int64(len(notifications)), // In real implementation, get total count separately
			PageSize:      int32(limit),
			Offset:        int32(offset),
			NextPageToken: "", // Generate next page token if needed
		},
		Metadata: &commonv1.ResponseMetadata{
			RequestId:        requestID,
			ServerId:         "swit-serve-1",
			ProcessingTimeMs: 0,
		},
	}

	logger.Logger.Info("GetNotifications request completed successfully",
		zap.String("user_id", req.UserId),
		zap.Int("count", len(notifications)),
		zap.String("request_id", requestID),
	)

	return response, nil
}

// MarkAsRead implements the MarkAsRead RPC
func (h *GRPCHandler) MarkAsRead(ctx context.Context, req *notificationv1.MarkAsReadRequest) (*notificationv1.MarkAsReadResponse, error) {
	// Input validation
	if req.NotificationId == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id is required")
	}

	logger.Logger.Info("Processing MarkAsRead request",
		zap.String("notification_id", req.NotificationId),
	)

	// Call business logic
	err := h.service.MarkAsRead(ctx, req.NotificationId)
	if err != nil {
		logger.Logger.Error("Failed to mark notification as read", zap.Error(err))
		if err.Error() == "notification not found" {
			return nil, status.Error(codes.NotFound, "notification not found")
		}
		return nil, status.Error(codes.Internal, "failed to mark notification as read")
	}

	// Extract request ID
	requestID := extractRequestIDFromMetadata(req.GetMetadata())

	// For response, we could fetch the updated notification, but for simplicity, just return success
	response := &notificationv1.MarkAsReadResponse{
		Notification: nil, // Could fetch updated notification here
		Metadata: &commonv1.ResponseMetadata{
			RequestId:        requestID,
			ServerId:         "swit-serve-1",
			ProcessingTimeMs: 0,
		},
	}

	logger.Logger.Info("MarkAsRead request completed successfully",
		zap.String("notification_id", req.NotificationId),
		zap.String("request_id", requestID),
	)

	return response, nil
}

// DeleteNotification implements the DeleteNotification RPC
func (h *GRPCHandler) DeleteNotification(ctx context.Context, req *notificationv1.DeleteNotificationRequest) (*notificationv1.DeleteNotificationResponse, error) {
	// Input validation
	if req.NotificationId == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id is required")
	}

	logger.Logger.Info("Processing DeleteNotification request",
		zap.String("notification_id", req.NotificationId),
	)

	// Call business logic
	err := h.service.DeleteNotification(ctx, req.NotificationId)
	if err != nil {
		logger.Logger.Error("Failed to delete notification", zap.Error(err))
		if err.Error() == "notification not found" {
			return nil, status.Error(codes.NotFound, "notification not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete notification")
	}

	// Extract request ID
	requestID := extractRequestIDFromMetadata(req.GetMetadata())

	response := &notificationv1.DeleteNotificationResponse{
		Success: true,
		Metadata: &commonv1.ResponseMetadata{
			RequestId:        requestID,
			ServerId:         "swit-serve-1",
			ProcessingTimeMs: 0,
		},
	}

	logger.Logger.Info("DeleteNotification request completed successfully",
		zap.String("notification_id", req.NotificationId),
		zap.String("request_id", requestID),
	)

	return response, nil
}

// StreamNotifications implements the streaming RPC
func (h *GRPCHandler) StreamNotifications(req *notificationv1.StreamNotificationsRequest, stream notificationv1.NotificationService_StreamNotificationsServer) error {
	// Input validation
	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}

	logger.Logger.Info("Processing StreamNotifications request",
		zap.String("user_id", req.UserId),
	)

	// TODO: Implement streaming logic
	// For now, return unimplemented
	return status.Error(codes.Unimplemented, "streaming not yet implemented")
}

// extractRequestIDFromMetadata extracts request ID from metadata or generates new one
func extractRequestIDFromMetadata(metadata *commonv1.RequestMetadata) string {
	if metadata != nil && metadata.RequestId != "" {
		return metadata.RequestId
	}
	return uuid.New().String()
}

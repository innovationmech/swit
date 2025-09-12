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

package messaging

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// AccessControlManager manages access control for messaging operations
type AccessControlManager struct {
	provider       AccessControlProvider
	authManager    *AuthManager
	enabled        bool
	defaultDeny    bool
	cacheEnabled   bool
	cacheTTL       time.Duration
	mu             sync.RWMutex
	logger         *zap.Logger
	decisionLogger *zap.Logger
}

// AccessControlManagerConfig configures the access control manager
type AccessControlManagerConfig struct {
	Provider     AccessControlProvider `json:"provider" yaml:"provider"`
	AuthManager  *AuthManager          `json:"auth_manager" yaml:"auth_manager"`
	Enabled      bool                  `json:"enabled" yaml:"enabled"`
	DefaultDeny  bool                  `json:"default_deny" yaml:"default_deny"`
	CacheEnabled bool                  `json:"cache_enabled" yaml:"cache_enabled"`
	CacheTTL     time.Duration         `json:"cache_ttl" yaml:"cache_ttl"`
	LogDecisions bool                  `json:"log_decisions" yaml:"log_decisions"`
}

// NewAccessControlManager creates a new access control manager
func NewAccessControlManager(config *AccessControlManagerConfig) (*AccessControlManager, error) {
	if config.Provider == nil {
		return nil, fmt.Errorf("access control provider is required")
	}

	if config.AuthManager == nil {
		return nil, fmt.Errorf("auth manager is required")
	}

	manager := &AccessControlManager{
		provider:     config.Provider,
		authManager:  config.AuthManager,
		enabled:      config.Enabled,
		defaultDeny:  config.DefaultDeny,
		cacheEnabled: config.CacheEnabled,
		cacheTTL:     config.CacheTTL,
		logger:       logger.Logger,
	}

	if config.LogDecisions {
		manager.decisionLogger = logger.Logger.Named("access-control-decisions")
	}

	return manager, nil
}

// CheckAccess checks if the given credentials have permission to access the resource
func (m *AccessControlManager) CheckAccess(ctx context.Context, credentials *AuthCredentials, resource *Resource, permission Permission) (*AccessDecision, error) {
	if !m.enabled {
		// Access control disabled, allow all
		decision := &AccessDecision{
			Allowed:    true,
			Reason:     "Access control disabled",
			Resource:   resource,
			Permission: permission,
			Timestamp:  time.Now(),
		}
		m.logDecision(decision)
		return decision, nil
	}

	// Authenticate credentials
	authCtx, err := m.authManager.Authenticate(ctx, credentials)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Create access context
	accessCtx := NewAccessContext(authCtx.Credentials.UserID, authCtx.Credentials.Scopes)
	accessCtx.AuthContext = authCtx

	// Check access
	decision, err := m.provider.CheckAccess(ctx, accessCtx, resource, permission)
	if err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	m.logDecision(decision)

	return decision, nil
}

// CheckTopicAccess checks if the given credentials have permission to access a topic
func (m *AccessControlManager) CheckTopicAccess(ctx context.Context, credentials *AuthCredentials, topicName string, permission Permission) (*AccessDecision, error) {
	resource := NewResource(AccessResourceTypeTopic, topicName)
	return m.CheckAccess(ctx, credentials, resource, permission)
}

// CheckQueueAccess checks if the given credentials have permission to access a queue
func (m *AccessControlManager) CheckQueueAccess(ctx context.Context, credentials *AuthCredentials, queueName string, permission Permission) (*AccessDecision, error) {
	resource := NewResource(AccessResourceTypeQueue, queueName)
	return m.CheckAccess(ctx, credentials, resource, permission)
}

// CheckPublishAccess checks if the given credentials can publish to a topic
func (m *AccessControlManager) CheckPublishAccess(ctx context.Context, credentials *AuthCredentials, topicName string) (*AccessDecision, error) {
	return m.CheckTopicAccess(ctx, credentials, topicName, PermissionPublish)
}

// CheckConsumeAccess checks if the given credentials can consume from a topic/queue
func (m *AccessControlManager) CheckConsumeAccess(ctx context.Context, credentials *AuthCredentials, resourceName string, resourceType AccessResourceType) (*AccessDecision, error) {
	resource := NewResource(resourceType, resourceName)
	return m.CheckAccess(ctx, credentials, resource, PermissionConsume)
}

// CheckReadAccess checks if the given credentials have read access to a resource
func (m *AccessControlManager) CheckReadAccess(ctx context.Context, credentials *AuthCredentials, resourceName string, resourceType AccessResourceType) (*AccessDecision, error) {
	resource := NewResource(resourceType, resourceName)
	return m.CheckAccess(ctx, credentials, resource, PermissionRead)
}

// CheckWriteAccess checks if the given credentials have write access to a resource
func (m *AccessControlManager) CheckWriteAccess(ctx context.Context, credentials *AuthCredentials, resourceName string, resourceType AccessResourceType) (*AccessDecision, error) {
	resource := NewResource(resourceType, resourceName)
	return m.CheckAccess(ctx, credentials, resource, PermissionWrite)
}

// CheckManageAccess checks if the given credentials have manage access to a resource
func (m *AccessControlManager) CheckManageAccess(ctx context.Context, credentials *AuthCredentials, resourceName string, resourceType AccessResourceType) (*AccessDecision, error) {
	resource := NewResource(resourceType, resourceName)
	return m.CheckAccess(ctx, credentials, resource, PermissionManage)
}

// GetPermissions returns all permissions for a user on a resource
func (m *AccessControlManager) GetPermissions(ctx context.Context, userID string, roles []string, resource *Resource) ([]Permission, error) {
	if !m.enabled {
		// Access control disabled, return all permissions
		return []Permission{PermissionRead, PermissionWrite, PermissionDelete, PermissionManage, PermissionConsume, PermissionPublish}, nil
	}

	return m.provider.GetPermissions(ctx, userID, roles, resource)
}

// AddPolicy adds an access policy
func (m *AccessControlManager) AddPolicy(ctx context.Context, policy *AccessPolicy) error {
	if !m.enabled {
		return fmt.Errorf("access control is disabled")
	}

	return m.provider.AddPolicy(ctx, policy)
}

// RemovePolicy removes an access policy
func (m *AccessControlManager) RemovePolicy(ctx context.Context, policyName string) error {
	if !m.enabled {
		return fmt.Errorf("access control is disabled")
	}

	return m.provider.RemovePolicy(ctx, policyName)
}

// AddRole adds a role
func (m *AccessControlManager) AddRole(ctx context.Context, role *Role) error {
	if !m.enabled {
		return fmt.Errorf("access control is disabled")
	}

	return m.provider.AddRole(ctx, role)
}

// RemoveRole removes a role
func (m *AccessControlManager) RemoveRole(ctx context.Context, roleName string) error {
	if !m.enabled {
		return fmt.Errorf("access control is disabled")
	}

	return m.provider.RemoveRole(ctx, roleName)
}

// AddAccessControlEntry adds an access control entry
func (m *AccessControlManager) AddAccessControlEntry(ctx context.Context, entry *AccessControlEntry) error {
	if !m.enabled {
		return fmt.Errorf("access control is disabled")
	}

	return m.provider.AddAccessControlEntry(ctx, entry)
}

// RemoveAccessControlEntry removes an access control entry
func (m *AccessControlManager) RemoveAccessControlEntry(ctx context.Context, entryID string) error {
	if !m.enabled {
		return fmt.Errorf("access control is disabled")
	}

	return m.provider.RemoveAccessControlEntry(ctx, entryID)
}

// IsHealthy checks if the access control manager is healthy
func (m *AccessControlManager) IsHealthy(ctx context.Context) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.provider.IsHealthy(ctx) && m.authManager.IsHealthy(ctx)
}

// CreateDefaultRoles creates default roles for access control
func (m *AccessControlManager) CreateDefaultRoles(ctx context.Context) error {
	if !m.enabled {
		return nil
	}

	// Create admin role
	adminRole := &Role{
		Name:        "admin",
		Description: "Full administrative access",
		Active:      true,
	}

	adminPolicy := &AccessPolicy{
		Name:        "admin-policy",
		Description: "Full access to all resources",
		Resources: []*Resource{
			ResourcePattern(AccessResourceTypeTopic, "*"),
			ResourcePattern(AccessResourceTypeQueue, "*"),
			ResourcePattern(AccessResourceTypeBroker, "*"),
		},
		Permissions: []Permission{
			PermissionRead,
			PermissionWrite,
			PermissionDelete,
			PermissionManage,
			PermissionConsume,
			PermissionPublish,
		},
		Priority: 1000,
		Enabled:  true,
	}

	adminRole.Policies = []*AccessPolicy{adminPolicy}

	// Create publisher role
	publisherRole := &Role{
		Name:        "publisher",
		Description: "Can publish messages to topics",
		Active:      true,
	}

	publisherPolicy := &AccessPolicy{
		Name:        "publisher-policy",
		Description: "Publish access to topics",
		Resources: []*Resource{
			ResourcePattern(AccessResourceTypeTopic, "*"),
		},
		Permissions: []Permission{
			PermissionPublish,
			PermissionRead,
		},
		Priority: 100,
		Enabled:  true,
	}

	publisherRole.Policies = []*AccessPolicy{publisherPolicy}

	// Create consumer role
	consumerRole := &Role{
		Name:        "consumer",
		Description: "Can consume messages from topics/queues",
		Active:      true,
	}

	consumerPolicy := &AccessPolicy{
		Name:        "consumer-policy",
		Description: "Consume access to topics and queues",
		Resources: []*Resource{
			ResourcePattern(AccessResourceTypeTopic, "*"),
			ResourcePattern(AccessResourceTypeQueue, "*"),
		},
		Permissions: []Permission{
			PermissionConsume,
			PermissionRead,
		},
		Priority: 100,
		Enabled:  true,
	}

	consumerRole.Policies = []*AccessPolicy{consumerPolicy}

	// Add roles
	if err := m.AddRole(ctx, adminRole); err != nil {
		return fmt.Errorf("failed to create admin role: %w", err)
	}

	if err := m.AddRole(ctx, publisherRole); err != nil {
		return fmt.Errorf("failed to create publisher role: %w", err)
	}

	if err := m.AddRole(ctx, consumerRole); err != nil {
		return fmt.Errorf("failed to create consumer role: %w", err)
	}

	m.logger.Info("Created default access control roles",
		zap.String("admin", "admin"),
		zap.String("publisher", "publisher"),
		zap.String("consumer", "consumer"))

	return nil
}

// logDecision logs access control decisions
func (m *AccessControlManager) logDecision(decision *AccessDecision) {
	if m.decisionLogger == nil {
		return
	}

	decisionType := "ALLOW"
	if !decision.Allowed {
		decisionType = "DENY"
	}

	userID := "anonymous"
	if decision.Context != nil {
		userID = decision.Context.UserID
	}

	m.decisionLogger.Info("Access control decision",
		zap.String("type", decisionType),
		zap.String("user_id", userID),
		zap.String("resource_type", string(decision.Resource.Type)),
		zap.String("resource_name", decision.Resource.Name),
		zap.String("permission", string(decision.Permission)),
		zap.String("reason", decision.Reason),
		zap.Time("timestamp", decision.Timestamp))
}

// AccessControlMiddleware creates middleware for access control
type AccessControlMiddleware struct {
	accessControlManager *AccessControlManager
	required             bool
	logger               *zap.Logger
}

// AccessControlMiddlewareConfig configures the access control middleware
type AccessControlMiddlewareConfig struct {
	AccessControlManager *AccessControlManager `json:"access_control_manager" yaml:"access_control_manager"`
	Required             bool                  `json:"required" yaml:"required"`
}

// NewAccessControlMiddleware creates a new access control middleware
func NewAccessControlMiddleware(config *AccessControlMiddlewareConfig) *AccessControlMiddleware {
	return &AccessControlMiddleware{
		accessControlManager: config.AccessControlManager,
		required:             config.Required,
		logger:               logger.Logger,
	}
}

// Name implements Middleware interface
func (m *AccessControlMiddleware) Name() string {
	return "access-control"
}

// Wrap implements Middleware interface
func (m *AccessControlMiddleware) Wrap(next MessageHandler) MessageHandler {
	return &accessControlMessageHandler{
		next:                 next,
		accessControlManager: m.accessControlManager,
		required:             m.required,
		logger:               m.logger,
	}
}

// accessControlMessageHandler implements MessageHandler with access control
type accessControlMessageHandler struct {
	next                 MessageHandler
	accessControlManager *AccessControlManager
	required             bool
	logger               *zap.Logger
}

// Handle implements MessageHandler interface
func (h *accessControlMessageHandler) Handle(ctx context.Context, message *Message) error {
	// Extract authentication context from message headers
	authCtx, err := AuthContextFromHeaders(message.Headers)
	if err != nil && h.required {
		h.logger.Error("Authentication failed for access control",
			zap.String("message_id", message.ID),
			zap.Error(err))
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Determine resource type and permission based on message headers
	resourceType := AccessResourceTypeTopic
	permission := PermissionPublish

	// Check headers to determine operation type
	if operation, ok := message.Headers["x-operation"]; ok {
		if operation == "consume" || operation == "subscribe" {
			permission = PermissionConsume
		}
	}

	// Use topic as resource name, fallback to headers if needed
	resourceName := message.Topic
	if resourceName == "" {
		if queue, ok := message.Headers["x-queue"]; ok {
			resourceName = queue
			resourceType = AccessResourceTypeQueue
		} else {
			resourceName = "unknown"
		}
	}

	// Check access control
	resource := NewResource(resourceType, resourceName)
	decision, err := h.accessControlManager.CheckAccess(ctx, authCtx.Credentials, resource, permission)
	if err != nil {
		h.logger.Error("Access control check failed",
			zap.String("message_id", message.ID),
			zap.String("resource", resourceName),
			zap.String("permission", string(permission)),
			zap.Error(err))
		return fmt.Errorf("access control check failed: %w", err)
	}

	if !decision.Allowed {
		h.logger.Warn("Access denied",
			zap.String("message_id", message.ID),
			zap.String("user_id", authCtx.Credentials.UserID),
			zap.String("resource", resourceName),
			zap.String("permission", string(permission)),
			zap.String("reason", decision.Reason))
		return fmt.Errorf("access denied: %s", decision.Reason)
	}

	// Add access decision to context for logging/auditing
	ctx = context.WithValue(ctx, "access_decision", decision)

	// Call the next handler
	return h.next.Handle(ctx, message)
}

// OnError implements MessageHandler interface
func (h *accessControlMessageHandler) OnError(ctx context.Context, message *Message, err error) ErrorAction {
	// Get access decision from context
	if decision, ok := ctx.Value("access_decision").(*AccessDecision); ok {
		h.logger.Error("Message handling error",
			zap.String("message_id", message.ID),
			zap.String("resource", decision.Resource.Name),
			zap.String("permission", string(decision.Permission)),
			zap.Bool("access_allowed", decision.Allowed),
			zap.Error(err))
	} else {
		h.logger.Error("Message handling error",
			zap.String("message_id", message.ID),
			zap.Error(err))
	}

	// Determine error action based on access control status
	if errors.Is(err, NewAccessControlError(ErrAccessDenied, "")) {
		return ErrorActionDeadLetter
	}

	return ErrorActionRetry
}

// Access control errors
var (
	ErrAccessDenied ErrorCode = "ACCESS_DENIED"
)

// AccessControlError wraps ErrorCode to implement error interface
type AccessControlError struct {
	Code ErrorCode
	Msg  string
}

func (e *AccessControlError) Error() string {
	return e.Msg
}

func (e *AccessControlError) Is(target error) bool {
	if other, ok := target.(*AccessControlError); ok {
		return e.Code == other.Code
	}
	return false
}

func NewAccessControlError(code ErrorCode, msg string) *AccessControlError {
	return &AccessControlError{Code: code, Msg: msg}
}

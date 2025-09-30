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
	"strings"
	"sync"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// AuthorizationPermission represents a specific permission in the authorization system
// It combines resource, action, and scope for fine-grained authorization
type AuthorizationPermission struct {
	// Resource represents the resource being accessed (e.g., "topic", "queue")
	Resource AccessResourceType `json:"resource" yaml:"resource"`

	// Action represents the action being performed
	Action Permission `json:"action" yaml:"action"`

	// Scope is an optional scope identifier (e.g., topic name pattern)
	Scope string `json:"scope,omitempty" yaml:"scope,omitempty"`
}

// String returns a string representation of the authorization permission
func (p *AuthorizationPermission) String() string {
	if p.Scope != "" {
		return fmt.Sprintf("%s:%s:%s", p.Resource, p.Action, p.Scope)
	}
	return fmt.Sprintf("%s:%s", p.Resource, p.Action)
}

// Matches checks if this authorization permission matches the required permission
func (p *AuthorizationPermission) Matches(required *AuthorizationPermission) bool {
	// Check resource
	if p.Resource != "*" && p.Resource != required.Resource {
		return false
	}

	// Check action
	if p.Action != "*" && p.Action != required.Action {
		return false
	}

	// Check scope - wildcard matches all, exact match, or prefix match
	if p.Scope != "" && p.Scope != "*" {
		if p.Scope != required.Scope {
			// Support prefix matching (e.g., "topic.user.*" matches "topic.user.123")
			if strings.HasSuffix(p.Scope, "*") {
				prefix := strings.TrimSuffix(p.Scope, "*")
				if !strings.HasPrefix(required.Scope, prefix) {
					return false
				}
			} else {
				return false
			}
		}
	}

	return true
}

// AuthorizationDecision represents the result of an authorization check
type AuthorizationDecision int

const (
	// DecisionDeny explicitly denies access (deny-by-default)
	DecisionDeny AuthorizationDecision = iota

	// DecisionAllow explicitly allows access
	DecisionAllow

	// DecisionAbstain indicates the provider cannot make a decision
	DecisionAbstain
)

func (d AuthorizationDecision) String() string {
	switch d {
	case DecisionDeny:
		return "deny"
	case DecisionAllow:
		return "allow"
	case DecisionAbstain:
		return "abstain"
	default:
		return "unknown"
	}
}

// AuthorizationContext contains information needed for authorization decisions
type AuthorizationContext struct {
	// Authentication context containing user credentials
	AuthContext *AuthContext `json:"auth_context"`

	// Permission being requested
	Permission *AuthorizationPermission `json:"permission"`

	// Message being processed
	Message *Message `json:"message"`

	// Additional metadata for authorization decisions
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AuthorizationProvider defines the interface for authorization providers
// Providers are pluggable and can implement different authorization strategies
type AuthorizationProvider interface {
	// Name returns the provider name
	Name() string

	// Authorize makes an authorization decision for the given context
	// Returns DecisionAllow, DecisionDeny, or DecisionAbstain
	Authorize(ctx context.Context, authzCtx *AuthorizationContext) (AuthorizationDecision, error)

	// GetPermissions returns all permissions for the given user
	GetPermissions(ctx context.Context, userID string) ([]*AuthorizationPermission, error)

	// HasPermission checks if the user has a specific permission
	HasPermission(ctx context.Context, userID string, permission *AuthorizationPermission) (bool, error)

	// IsHealthy checks if the authorization provider is healthy
	IsHealthy(ctx context.Context) bool
}

// RBACAuthorizationProvider implements role-based access control (RBAC)
type RBACAuthorizationProvider struct {
	// roles maps role names to their permissions
	roles map[string][]*AuthorizationPermission

	// userRoles maps user IDs to their roles
	userRoles map[string][]string

	mu sync.RWMutex
}

// RBACAuthorizationProviderConfig configures RBAC authorization provider
type RBACAuthorizationProviderConfig struct {
	// Roles maps role names to permissions
	Roles map[string][]*AuthorizationPermission `json:"roles" yaml:"roles"`

	// UserRoles maps user IDs to role names
	UserRoles map[string][]string `json:"user_roles" yaml:"user_roles"`
}

// NewRBACAuthorizationProvider creates a new RBAC authorization provider
func NewRBACAuthorizationProvider(config *RBACAuthorizationProviderConfig) *RBACAuthorizationProvider {
	if config.Roles == nil {
		config.Roles = make(map[string][]*AuthorizationPermission)
	}
	if config.UserRoles == nil {
		config.UserRoles = make(map[string][]string)
	}

	return &RBACAuthorizationProvider{
		roles:     config.Roles,
		userRoles: config.UserRoles,
	}
}

// Name implements AuthorizationProvider interface
func (p *RBACAuthorizationProvider) Name() string {
	return "rbac"
}

// Authorize implements AuthorizationProvider interface
func (p *RBACAuthorizationProvider) Authorize(ctx context.Context, authzCtx *AuthorizationContext) (AuthorizationDecision, error) {
	// Anonymous users are denied by default
	if authzCtx.AuthContext == nil || authzCtx.AuthContext.Credentials == nil {
		return DecisionDeny, nil
	}

	userID := authzCtx.AuthContext.Credentials.UserID
	if userID == "" {
		return DecisionDeny, nil
	}

	// Check if user has the required permission
	hasPermission, err := p.HasPermission(ctx, userID, authzCtx.Permission)
	if err != nil {
		return DecisionDeny, fmt.Errorf("permission check failed: %w", err)
	}

	if hasPermission {
		return DecisionAllow, nil
	}

	return DecisionDeny, nil
}

// GetPermissions implements AuthorizationProvider interface
func (p *RBACAuthorizationProvider) GetPermissions(ctx context.Context, userID string) ([]*AuthorizationPermission, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	roles, exists := p.userRoles[userID]
	if !exists {
		return nil, nil
	}

	var permissions []*AuthorizationPermission
	seenPerms := make(map[string]bool)

	for _, roleName := range roles {
		rolePerms, exists := p.roles[roleName]
		if !exists {
			continue
		}

		for _, perm := range rolePerms {
			permStr := perm.String()
			if !seenPerms[permStr] {
				permissions = append(permissions, perm)
				seenPerms[permStr] = true
			}
		}
	}

	return permissions, nil
}

// HasPermission implements AuthorizationProvider interface
func (p *RBACAuthorizationProvider) HasPermission(ctx context.Context, userID string, permission *AuthorizationPermission) (bool, error) {
	permissions, err := p.GetPermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	// Check if any of the user's permissions match the required permission
	for _, perm := range permissions {
		if perm.Matches(permission) {
			return true, nil
		}
	}

	return false, nil
}

// IsHealthy implements AuthorizationProvider interface
func (p *RBACAuthorizationProvider) IsHealthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Basic health check - roles are configured
	return len(p.roles) > 0
}

// AddRole adds a role with its permissions
func (p *RBACAuthorizationProvider) AddRole(roleName string, permissions []*AuthorizationPermission) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.roles[roleName] = permissions
}

// RemoveRole removes a role
func (p *RBACAuthorizationProvider) RemoveRole(roleName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.roles, roleName)

	// Remove role from all users
	for userID, roles := range p.userRoles {
		newRoles := make([]string, 0)
		for _, role := range roles {
			if role != roleName {
				newRoles = append(newRoles, role)
			}
		}
		p.userRoles[userID] = newRoles
	}
}

// AssignRole assigns a role to a user
func (p *RBACAuthorizationProvider) AssignRole(userID string, roleName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if role exists
	if _, exists := p.roles[roleName]; !exists {
		return fmt.Errorf("role '%s' not found", roleName)
	}

	// Add role to user
	roles := p.userRoles[userID]
	for _, r := range roles {
		if r == roleName {
			return nil // Already assigned
		}
	}

	p.userRoles[userID] = append(roles, roleName)
	return nil
}

// RevokeRole revokes a role from a user
func (p *RBACAuthorizationProvider) RevokeRole(userID string, roleName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	roles := p.userRoles[userID]
	newRoles := make([]string, 0)
	for _, r := range roles {
		if r != roleName {
			newRoles = append(newRoles, r)
		}
	}
	p.userRoles[userID] = newRoles
}

// ACLAuthorizationProvider implements access control list (ACL) based authorization
type ACLAuthorizationProvider struct {
	// acl maps user IDs to their explicit permissions
	acl map[string][]*AuthorizationPermission

	mu sync.RWMutex
}

// ACLAuthorizationProviderConfig configures ACL authorization provider
type ACLAuthorizationProviderConfig struct {
	// ACL maps user IDs to their permissions
	ACL map[string][]*AuthorizationPermission `json:"acl" yaml:"acl"`
}

// NewACLAuthorizationProvider creates a new ACL authorization provider
func NewACLAuthorizationProvider(config *ACLAuthorizationProviderConfig) *ACLAuthorizationProvider {
	if config.ACL == nil {
		config.ACL = make(map[string][]*AuthorizationPermission)
	}

	return &ACLAuthorizationProvider{
		acl: config.ACL,
	}
}

// Name implements AuthorizationProvider interface
func (p *ACLAuthorizationProvider) Name() string {
	return "acl"
}

// Authorize implements AuthorizationProvider interface
func (p *ACLAuthorizationProvider) Authorize(ctx context.Context, authzCtx *AuthorizationContext) (AuthorizationDecision, error) {
	// Anonymous users are denied by default
	if authzCtx.AuthContext == nil || authzCtx.AuthContext.Credentials == nil {
		return DecisionDeny, nil
	}

	userID := authzCtx.AuthContext.Credentials.UserID
	if userID == "" {
		return DecisionDeny, nil
	}

	// Check if user has the required permission
	hasPermission, err := p.HasPermission(ctx, userID, authzCtx.Permission)
	if err != nil {
		return DecisionDeny, fmt.Errorf("permission check failed: %w", err)
	}

	if hasPermission {
		return DecisionAllow, nil
	}

	return DecisionDeny, nil
}

// GetPermissions implements AuthorizationProvider interface
func (p *ACLAuthorizationProvider) GetPermissions(ctx context.Context, userID string) ([]*AuthorizationPermission, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	permissions, exists := p.acl[userID]
	if !exists {
		return nil, nil
	}

	// Return a copy to prevent external modification
	result := make([]*AuthorizationPermission, len(permissions))
	copy(result, permissions)
	return result, nil
}

// HasPermission implements AuthorizationProvider interface
func (p *ACLAuthorizationProvider) HasPermission(ctx context.Context, userID string, permission *AuthorizationPermission) (bool, error) {
	permissions, err := p.GetPermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	// Check if any of the user's permissions match the required permission
	for _, perm := range permissions {
		if perm.Matches(permission) {
			return true, nil
		}
	}

	return false, nil
}

// IsHealthy implements AuthorizationProvider interface
func (p *ACLAuthorizationProvider) IsHealthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// ACL provider is always healthy
	return true
}

// GrantPermission grants a permission to a user
func (p *ACLAuthorizationProvider) GrantPermission(userID string, permission *AuthorizationPermission) {
	p.mu.Lock()
	defer p.mu.Unlock()

	permissions := p.acl[userID]
	p.acl[userID] = append(permissions, permission)
}

// RevokePermission revokes a permission from a user
func (p *ACLAuthorizationProvider) RevokePermission(userID string, permission *AuthorizationPermission) {
	p.mu.Lock()
	defer p.mu.Unlock()

	permissions := p.acl[userID]
	newPermissions := make([]*AuthorizationPermission, 0)
	permStr := permission.String()

	for _, perm := range permissions {
		if perm.String() != permStr {
			newPermissions = append(newPermissions, perm)
		}
	}

	p.acl[userID] = newPermissions
}

// AuthorizationManager manages authorization for messaging operations
// It implements a deny-by-default strategy with pluggable authorization providers
type AuthorizationManager struct {
	providers       map[string]AuthorizationProvider
	defaultProvider string

	// denyByDefault enforces deny-by-default policy
	denyByDefault bool

	mu sync.RWMutex
}

// AuthorizationManagerConfig configures the authorization manager
type AuthorizationManagerConfig struct {
	// DefaultProvider is the default authorization provider to use
	DefaultProvider string `json:"default_provider" yaml:"default_provider"`

	// Providers maps provider names to provider implementations
	Providers map[string]AuthorizationProvider `json:"providers" yaml:"providers"`

	// DenyByDefault enforces deny-by-default policy (default: true)
	DenyByDefault bool `json:"deny_by_default" yaml:"deny_by_default"`
}

// NewAuthorizationManager creates a new authorization manager
func NewAuthorizationManager(config *AuthorizationManagerConfig) (*AuthorizationManager, error) {
	if config.DefaultProvider == "" {
		return nil, fmt.Errorf("default provider is required")
	}

	if _, exists := config.Providers[config.DefaultProvider]; !exists {
		return nil, fmt.Errorf("default provider '%s' not found", config.DefaultProvider)
	}

	// Default to deny-by-default if not specified
	denyByDefault := config.DenyByDefault
	if !config.DenyByDefault && config.DenyByDefault == false {
		// Only set to true if explicitly false, otherwise default to true
		denyByDefault = true
	}

	return &AuthorizationManager{
		providers:       config.Providers,
		defaultProvider: config.DefaultProvider,
		denyByDefault:   denyByDefault,
	}, nil
}

// Authorize makes an authorization decision using the configured providers
// Implements deny-by-default strategy: access is denied unless explicitly allowed
func (m *AuthorizationManager) Authorize(ctx context.Context, authzCtx *AuthorizationContext) (AuthorizationDecision, error) {
	// Deny-by-default: start with deny decision
	finalDecision := DecisionDeny

	// Use default provider
	m.mu.RLock()
	provider, exists := m.providers[m.defaultProvider]
	m.mu.RUnlock()

	if !exists {
		return DecisionDeny, fmt.Errorf("authorization provider '%s' not found", m.defaultProvider)
	}

	decision, err := provider.Authorize(ctx, authzCtx)
	if err != nil {
		// On error, deny access (fail-safe)
		return DecisionDeny, fmt.Errorf("authorization check failed: %w", err)
	}

	// Update decision based on provider result
	switch decision {
	case DecisionAllow:
		finalDecision = DecisionAllow
	case DecisionDeny:
		finalDecision = DecisionDeny
	case DecisionAbstain:
		// Abstain means the provider cannot make a decision
		// With deny-by-default, abstain results in deny
		if m.denyByDefault {
			finalDecision = DecisionDeny
		}
	}

	return finalDecision, nil
}

// CheckPermission checks if the given user has the specified permission
func (m *AuthorizationManager) CheckPermission(ctx context.Context, userID string, permission *AuthorizationPermission) (bool, error) {
	m.mu.RLock()
	provider, exists := m.providers[m.defaultProvider]
	m.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("authorization provider '%s' not found", m.defaultProvider)
	}

	return provider.HasPermission(ctx, userID, permission)
}

// IsHealthy checks if all authorization providers are healthy
func (m *AuthorizationManager) IsHealthy(ctx context.Context) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, provider := range m.providers {
		if !provider.IsHealthy(ctx) {
			return false
		}
	}

	return true
}

// AddProvider adds an authorization provider
func (m *AuthorizationManager) AddProvider(name string, provider AuthorizationProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.providers[name]; exists {
		return fmt.Errorf("provider '%s' already exists", name)
	}

	m.providers[name] = provider
	return nil
}

// RemoveProvider removes an authorization provider
func (m *AuthorizationManager) RemoveProvider(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == m.defaultProvider {
		return fmt.Errorf("cannot remove default provider")
	}

	if _, exists := m.providers[name]; !exists {
		return fmt.Errorf("provider '%s' not found", name)
	}

	delete(m.providers, name)
	return nil
}

// GetProvider returns an authorization provider by name
func (m *AuthorizationManager) GetProvider(name string) (AuthorizationProvider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[name]
	return provider, exists
}

// AuthorizationMiddleware creates middleware for message authorization
type AuthorizationMiddleware struct {
	authzManager *AuthorizationManager
	logger       *zap.Logger

	// permissionExtractor extracts required permissions from messages
	permissionExtractor func(*Message) *AuthorizationPermission
}

// AuthorizationMiddlewareConfig configures the authorization middleware
type AuthorizationMiddlewareConfig struct {
	// AuthorizationManager is the authorization manager to use
	AuthzManager *AuthorizationManager `json:"authz_manager" yaml:"authz_manager"`

	// PermissionExtractor is an optional function to extract permissions from messages
	// If not provided, defaults to checking topic-based permissions
	PermissionExtractor func(*Message) *AuthorizationPermission `json:"-" yaml:"-"`
}

// NewAuthorizationMiddleware creates a new authorization middleware
func NewAuthorizationMiddleware(config *AuthorizationMiddlewareConfig) *AuthorizationMiddleware {
	permissionExtractor := config.PermissionExtractor
	if permissionExtractor == nil {
		// Default permission extractor: extract from topic
		permissionExtractor = func(msg *Message) *AuthorizationPermission {
			return &AuthorizationPermission{
				Resource: AccessResourceTypeTopic,
				Action:   PermissionConsume,
				Scope:    msg.Topic,
			}
		}
	}

	return &AuthorizationMiddleware{
		authzManager:        config.AuthzManager,
		permissionExtractor: permissionExtractor,
		logger:              logger.Logger,
	}
}

// Name implements Middleware interface
func (m *AuthorizationMiddleware) Name() string {
	return "authorization"
}

// Wrap implements Middleware interface
func (m *AuthorizationMiddleware) Wrap(next MessageHandler) MessageHandler {
	return &authzMessageHandler{
		next:                next,
		authzManager:        m.authzManager,
		permissionExtractor: m.permissionExtractor,
		logger:              m.logger,
	}
}

// authzMessageHandler implements MessageHandler with authorization
type authzMessageHandler struct {
	next                MessageHandler
	authzManager        *AuthorizationManager
	permissionExtractor func(*Message) *AuthorizationPermission
	logger              *zap.Logger
}

// Handle implements MessageHandler interface
func (h *authzMessageHandler) Handle(ctx context.Context, message *Message) error {
	// Extract authentication context from context
	authCtx, ok := ctx.Value("auth_context").(*AuthContext)
	if !ok || authCtx == nil {
		h.logger.Error("Authorization failed: no authentication context",
			zap.String("message_id", message.ID))
		return fmt.Errorf("authorization failed: no authentication context")
	}

	// Extract required permission from message
	requiredPermission := h.permissionExtractor(message)
	if requiredPermission == nil {
		h.logger.Error("Authorization failed: cannot extract permission",
			zap.String("message_id", message.ID))
		return fmt.Errorf("authorization failed: cannot extract permission")
	}

	// Create authorization context
	authzCtx := &AuthorizationContext{
		AuthContext: authCtx,
		Permission:  requiredPermission,
		Message:     message,
		Metadata:    make(map[string]interface{}),
	}

	// Make authorization decision
	decision, err := h.authzManager.Authorize(ctx, authzCtx)
	if err != nil {
		h.logger.Error("Authorization check failed",
			zap.String("message_id", message.ID),
			zap.String("user_id", authCtx.Credentials.UserID),
			zap.String("permission", requiredPermission.String()),
			zap.Error(err))
		return fmt.Errorf("authorization check failed: %w", err)
	}

	// Enforce decision (deny-by-default)
	if decision != DecisionAllow {
		h.logger.Warn("Authorization denied",
			zap.String("message_id", message.ID),
			zap.String("user_id", authCtx.Credentials.UserID),
			zap.String("permission", requiredPermission.String()),
			zap.String("decision", decision.String()))
		return ErrAuthzDenied
	}

	// Authorization successful - log and proceed
	h.logger.Debug("Authorization granted",
		zap.String("message_id", message.ID),
		zap.String("user_id", authCtx.Credentials.UserID),
		zap.String("permission", requiredPermission.String()))

	// Add authorization context to context for downstream handlers
	ctx = context.WithValue(ctx, "authz_context", authzCtx)

	// Call the next handler
	return h.next.Handle(ctx, message)
}

// OnError implements MessageHandler interface
func (h *authzMessageHandler) OnError(ctx context.Context, message *Message, err error) ErrorAction {
	// Get authorization context from context
	if authzCtx, ok := ctx.Value("authz_context").(*AuthorizationContext); ok {
		h.logger.Error("Message handling error",
			zap.String("message_id", message.ID),
			zap.String("user_id", authzCtx.AuthContext.Credentials.UserID),
			zap.String("permission", authzCtx.Permission.String()),
			zap.Error(err))
	} else {
		h.logger.Error("Message handling error",
			zap.String("message_id", message.ID),
			zap.Error(err))
	}

	// Authorization errors should not be retried
	if errors.Is(err, ErrAuthzDenied) {
		return ErrorActionDeadLetter
	}

	return ErrorActionRetry
}

// Authorization errors
var (
	// ErrAuthzDenied is returned when authorization is denied
	ErrAuthzDenied = errors.New("authorization denied")

	// ErrInvalidPermission is returned when permission format is invalid
	ErrInvalidPermission = errors.New("invalid permission")
)

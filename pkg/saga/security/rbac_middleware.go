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

package security

import (
	"context"
	"fmt"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// RBACMiddleware provides RBAC permission checking for Saga operations
type RBACMiddleware struct {
	rbacManager *RBACManager
	logger      *zap.Logger
}

// RBACMiddlewareConfig configures the RBAC middleware
type RBACMiddlewareConfig struct {
	RBACManager *RBACManager
}

// NewRBACMiddleware creates a new RBAC middleware
func NewRBACMiddleware(config *RBACMiddlewareConfig) *RBACMiddleware {
	if config == nil || config.RBACManager == nil {
		panic("RBAC manager is required for RBAC middleware")
	}

	middleware := &RBACMiddleware{
		rbacManager: config.RBACManager,
		logger:      logger.Logger,
	}

	if middleware.logger == nil {
		middleware.logger = zap.NewNop()
	}

	return middleware
}

// RequirePermission checks if the authenticated user has the required permission
func (m *RBACMiddleware) RequirePermission(ctx context.Context, permission Permission) error {
	// Extract auth context from context
	authCtx, ok := AuthFromContext(ctx)
	if !ok || authCtx == nil || authCtx.Credentials == nil {
		m.logger.Error("No authentication context found in request")
		return ErrMissingCredentials
	}

	userID := authCtx.Credentials.UserID
	if userID == "" {
		m.logger.Error("No user ID found in authentication context")
		return ErrMissingCredentials
	}

	// Check permission
	if err := m.rbacManager.CheckPermission(ctx, userID, permission); err != nil {
		m.logger.Warn("Permission check failed",
			zap.String("user_id", userID),
			zap.String("permission", permission.String()),
			zap.Error(err))
		return err
	}

	m.logger.Debug("Permission check passed",
		zap.String("user_id", userID),
		zap.String("permission", permission.String()))

	return nil
}

// RequirePermissions checks if the authenticated user has all required permissions
func (m *RBACMiddleware) RequirePermissions(ctx context.Context, permissions ...Permission) error {
	// Extract auth context from context
	authCtx, ok := AuthFromContext(ctx)
	if !ok || authCtx == nil || authCtx.Credentials == nil {
		m.logger.Error("No authentication context found in request")
		return ErrMissingCredentials
	}

	userID := authCtx.Credentials.UserID
	if userID == "" {
		m.logger.Error("No user ID found in authentication context")
		return ErrMissingCredentials
	}

	// Check permissions
	if err := m.rbacManager.CheckPermissions(ctx, userID, permissions...); err != nil {
		m.logger.Warn("Permission check failed (multiple)",
			zap.String("user_id", userID),
			zap.Int("required_count", len(permissions)),
			zap.Error(err))
		return err
	}

	m.logger.Debug("Permission check passed (multiple)",
		zap.String("user_id", userID),
		zap.Int("required_count", len(permissions)))

	return nil
}

// RequireAnyPermission checks if the authenticated user has any of the required permissions
func (m *RBACMiddleware) RequireAnyPermission(ctx context.Context, permissions ...Permission) error {
	// Extract auth context from context
	authCtx, ok := AuthFromContext(ctx)
	if !ok || authCtx == nil || authCtx.Credentials == nil {
		m.logger.Error("No authentication context found in request")
		return ErrMissingCredentials
	}

	userID := authCtx.Credentials.UserID
	if userID == "" {
		m.logger.Error("No user ID found in authentication context")
		return ErrMissingCredentials
	}

	// Check any permission
	if err := m.rbacManager.CheckAnyPermission(ctx, userID, permissions...); err != nil {
		m.logger.Warn("Permission check failed (any)",
			zap.String("user_id", userID),
			zap.Int("required_count", len(permissions)),
			zap.Error(err))
		return err
	}

	m.logger.Debug("Permission check passed (any)",
		zap.String("user_id", userID),
		zap.Int("required_count", len(permissions)))

	return nil
}

// RequireRole checks if the authenticated user has the specified role
func (m *RBACMiddleware) RequireRole(ctx context.Context, roleName string) error {
	// Extract auth context from context
	authCtx, ok := AuthFromContext(ctx)
	if !ok || authCtx == nil || authCtx.Credentials == nil {
		m.logger.Error("No authentication context found in request")
		return ErrMissingCredentials
	}

	userID := authCtx.Credentials.UserID
	if userID == "" {
		m.logger.Error("No user ID found in authentication context")
		return ErrMissingCredentials
	}

	// Get user roles
	roles, err := m.rbacManager.GetUserRoles(userID)
	if err != nil {
		m.logger.Error("Failed to get user roles",
			zap.String("user_id", userID),
			zap.Error(err))
		return err
	}

	// Check if user has the required role
	hasRole := false
	for _, role := range roles {
		if role == roleName {
			hasRole = true
			break
		}
	}

	if !hasRole {
		m.logger.Warn("Role check failed",
			zap.String("user_id", userID),
			zap.String("required_role", roleName))
		return fmt.Errorf("%w: user '%s' does not have role '%s'", ErrPermissionDenied, userID, roleName)
	}

	m.logger.Debug("Role check passed",
		zap.String("user_id", userID),
		zap.String("role", roleName))

	return nil
}

// GetUserPermissions returns all permissions for the authenticated user
func (m *RBACMiddleware) GetUserPermissions(ctx context.Context) (*PermissionSet, error) {
	// Extract auth context from context
	authCtx, ok := AuthFromContext(ctx)
	if !ok || authCtx == nil || authCtx.Credentials == nil {
		m.logger.Error("No authentication context found in request")
		return nil, ErrMissingCredentials
	}

	userID := authCtx.Credentials.UserID
	if userID == "" {
		m.logger.Error("No user ID found in authentication context")
		return nil, ErrMissingCredentials
	}

	return m.rbacManager.GetUserPermissions(userID)
}

// HasPermission checks if the authenticated user has a specific permission without returning an error
func (m *RBACMiddleware) HasPermission(ctx context.Context, permission Permission) bool {
	err := m.RequirePermission(ctx, permission)
	return err == nil
}

// HasAnyPermission checks if the authenticated user has any of the given permissions without returning an error
func (m *RBACMiddleware) HasAnyPermission(ctx context.Context, permissions ...Permission) bool {
	err := m.RequireAnyPermission(ctx, permissions...)
	return err == nil
}

// HasAllPermissions checks if the authenticated user has all given permissions without returning an error
func (m *RBACMiddleware) HasAllPermissions(ctx context.Context, permissions ...Permission) bool {
	err := m.RequirePermissions(ctx, permissions...)
	return err == nil
}

// HasRole checks if the authenticated user has a specific role without returning an error
func (m *RBACMiddleware) HasRole(ctx context.Context, roleName string) bool {
	err := m.RequireRole(ctx, roleName)
	return err == nil
}

// RBACManager returns the underlying RBAC manager
func (m *RBACMiddleware) RBACManager() *RBACManager {
	return m.rbacManager
}

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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Common RBAC errors
var (
	ErrRoleNotFound        = errors.New("role not found")
	ErrRoleAlreadyExists   = errors.New("role already exists")
	ErrUserNotFound        = errors.New("user not found")
	ErrPermissionDenied    = errors.New("permission denied")
	ErrInvalidRole         = errors.New("invalid role")
	ErrInvalidPermission   = errors.New("invalid permission")
	ErrCircularInheritance = errors.New("circular role inheritance detected")
)

// Role represents a role in the RBAC system
type Role struct {
	// Name is the unique identifier for the role
	Name string

	// Description provides context about the role
	Description string

	// Permissions assigned to this role
	Permissions *PermissionSet

	// Parent roles for permission inheritance
	ParentRoles []string

	// Metadata for additional role properties
	Metadata map[string]string

	// CreatedAt is when the role was created
	CreatedAt time.Time

	// UpdatedAt is when the role was last updated
	UpdatedAt time.Time
}

// NewRole creates a new role with the given name and permissions
func NewRole(name, description string, permissions ...Permission) *Role {
	now := time.Now()
	return &Role{
		Name:        name,
		Description: description,
		Permissions: NewPermissionSet(permissions...),
		ParentRoles: make([]string, 0),
		Metadata:    make(map[string]string),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// AddPermission adds a permission to the role
func (r *Role) AddPermission(permission Permission) {
	r.Permissions.Add(permission)
	r.UpdatedAt = time.Now()
}

// RemovePermission removes a permission from the role
func (r *Role) RemovePermission(permission Permission) {
	r.Permissions.Remove(permission)
	r.UpdatedAt = time.Now()
}

// HasPermission checks if the role has a specific permission (direct only, no inheritance)
func (r *Role) HasPermission(permission Permission) bool {
	return r.Permissions.Has(permission)
}

// AddParent adds a parent role for inheritance
func (r *Role) AddParent(parentName string) {
	for _, p := range r.ParentRoles {
		if p == parentName {
			return
		}
	}
	r.ParentRoles = append(r.ParentRoles, parentName)
	r.UpdatedAt = time.Now()
}

// RemoveParent removes a parent role
func (r *Role) RemoveParent(parentName string) {
	for i, p := range r.ParentRoles {
		if p == parentName {
			r.ParentRoles = append(r.ParentRoles[:i], r.ParentRoles[i+1:]...)
			r.UpdatedAt = time.Now()
			return
		}
	}
}

// Clone creates a deep copy of the role
func (r *Role) Clone() *Role {
	clone := &Role{
		Name:        r.Name,
		Description: r.Description,
		Permissions: r.Permissions.Clone(),
		ParentRoles: make([]string, len(r.ParentRoles)),
		Metadata:    make(map[string]string),
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	copy(clone.ParentRoles, r.ParentRoles)
	for k, v := range r.Metadata {
		clone.Metadata[k] = v
	}
	return clone
}

// UserRoles represents the mapping of a user to their roles
type UserRoles struct {
	UserID    string
	Roles     []string
	UpdatedAt time.Time
}

// NewUserRoles creates a new user-roles mapping
func NewUserRoles(userID string, roles ...string) *UserRoles {
	return &UserRoles{
		UserID:    userID,
		Roles:     roles,
		UpdatedAt: time.Now(),
	}
}

// HasRole checks if the user has a specific role
func (ur *UserRoles) HasRole(roleName string) bool {
	for _, r := range ur.Roles {
		if r == roleName {
			return true
		}
	}
	return false
}

// AddRole adds a role to the user
func (ur *UserRoles) AddRole(roleName string) {
	if !ur.HasRole(roleName) {
		ur.Roles = append(ur.Roles, roleName)
		ur.UpdatedAt = time.Now()
	}
}

// RemoveRole removes a role from the user
func (ur *UserRoles) RemoveRole(roleName string) {
	for i, r := range ur.Roles {
		if r == roleName {
			ur.Roles = append(ur.Roles[:i], ur.Roles[i+1:]...)
			ur.UpdatedAt = time.Now()
			return
		}
	}
}

// RBACManager manages roles and permissions in the RBAC system
type RBACManager struct {
	// roles maps role names to role definitions
	roles map[string]*Role

	// userRoles maps user IDs to their assigned roles
	userRoles map[string]*UserRoles

	// permissionCache caches computed permissions for users
	permissionCache *PermissionCache

	// cacheEnabled controls whether caching is enabled
	cacheEnabled bool

	// mu protects concurrent access to roles and userRoles
	mu sync.RWMutex

	// logger for RBAC operations
	logger *zap.Logger
}

// RBACManagerConfig configures the RBAC manager
type RBACManagerConfig struct {
	// CacheEnabled enables permission caching
	CacheEnabled bool

	// CacheTTL is the time-to-live for cached permissions
	CacheTTL time.Duration

	// CacheMaxSize is the maximum number of entries in the cache
	CacheMaxSize int
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(config *RBACManagerConfig) *RBACManager {
	if config == nil {
		config = &RBACManagerConfig{
			CacheEnabled: true,
			CacheTTL:     5 * time.Minute,
			CacheMaxSize: 1000,
		}
	}

	manager := &RBACManager{
		roles:        make(map[string]*Role),
		userRoles:    make(map[string]*UserRoles),
		cacheEnabled: config.CacheEnabled,
		logger:       logger.Logger,
	}

	if manager.logger == nil {
		manager.logger = zap.NewNop()
	}

	if config.CacheEnabled {
		manager.permissionCache = NewPermissionCache(&PermissionCacheConfig{
			TTL:     config.CacheTTL,
			MaxSize: config.CacheMaxSize,
		})
	}

	// Initialize with predefined roles
	manager.initializePredefinedRoles()

	return manager
}

// initializePredefinedRoles sets up common predefined roles
func (m *RBACManager) initializePredefinedRoles() {
	// Admin role with full permissions
	adminRole := NewRole("admin", "Administrator with full permissions", AllPermissions()...)
	m.roles["admin"] = adminRole

	// Operator role for managing sagas
	operatorRole := NewRole("operator", "Operator for managing saga operations",
		PermissionSagaExecute,
		PermissionSagaRead,
		PermissionSagaCancel,
		PermissionSagaRetry,
		PermissionSagaRollback,
		PermissionSagaCompensate,
		PermissionSagaCreate,
		PermissionSagaUpdate,
		PermissionSagaList,
		PermissionStepExecute,
		PermissionStepRead,
		PermissionStepUpdate,
		PermissionMonitorView,
	)
	m.roles["operator"] = operatorRole

	// Viewer role for read-only access
	viewerRole := NewRole("viewer", "Viewer with read-only access",
		PermissionSagaRead,
		PermissionSagaList,
		PermissionStepRead,
		PermissionMonitorView,
		PermissionConfigRead,
	)
	m.roles["viewer"] = viewerRole

	// Auditor role for audit access
	auditorRole := NewRole("auditor", "Auditor with audit log access",
		PermissionAuditView,
		PermissionAuditExport,
		PermissionMonitorView,
	)
	m.roles["auditor"] = auditorRole
}

// CreateRole creates a new role
func (m *RBACManager) CreateRole(role *Role) error {
	if role == nil {
		return ErrInvalidRole
	}

	if role.Name == "" {
		return fmt.Errorf("%w: role name cannot be empty", ErrInvalidRole)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[role.Name]; exists {
		return fmt.Errorf("%w: role '%s' already exists", ErrRoleAlreadyExists, role.Name)
	}

	// Validate parent roles exist
	for _, parentName := range role.ParentRoles {
		if _, exists := m.roles[parentName]; !exists {
			return fmt.Errorf("%w: parent role '%s' not found", ErrRoleNotFound, parentName)
		}
	}

	// Check for circular inheritance
	if err := m.checkCircularInheritance(role.Name, role.ParentRoles); err != nil {
		return err
	}

	m.roles[role.Name] = role.Clone()
	m.invalidateCache()

	m.logger.Info("Role created",
		zap.String("role", role.Name),
		zap.Int("permissions", role.Permissions.Size()))

	return nil
}

// GetRole retrieves a role by name
func (m *RBACManager) GetRole(name string) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	role, exists := m.roles[name]
	if !exists {
		return nil, fmt.Errorf("%w: role '%s' not found", ErrRoleNotFound, name)
	}

	return role.Clone(), nil
}

// UpdateRole updates an existing role
func (m *RBACManager) UpdateRole(role *Role) error {
	if role == nil {
		return ErrInvalidRole
	}

	if role.Name == "" {
		return fmt.Errorf("%w: role name cannot be empty", ErrInvalidRole)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[role.Name]; !exists {
		return fmt.Errorf("%w: role '%s' not found", ErrRoleNotFound, role.Name)
	}

	// Validate parent roles exist
	for _, parentName := range role.ParentRoles {
		if _, exists := m.roles[parentName]; !exists {
			return fmt.Errorf("%w: parent role '%s' not found", ErrRoleNotFound, parentName)
		}
	}

	// Check for circular inheritance
	if err := m.checkCircularInheritance(role.Name, role.ParentRoles); err != nil {
		return err
	}

	role.UpdatedAt = time.Now()
	m.roles[role.Name] = role.Clone()
	m.invalidateCache()

	m.logger.Info("Role updated",
		zap.String("role", role.Name),
		zap.Int("permissions", role.Permissions.Size()))

	return nil
}

// DeleteRole deletes a role
func (m *RBACManager) DeleteRole(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[name]; !exists {
		return fmt.Errorf("%w: role '%s' not found", ErrRoleNotFound, name)
	}

	// Check if role is being used by any user
	for _, ur := range m.userRoles {
		if ur.HasRole(name) {
			return fmt.Errorf("cannot delete role '%s': still assigned to user '%s'", name, ur.UserID)
		}
	}

	// Check if role is parent of any other role
	for roleName, role := range m.roles {
		for _, parentName := range role.ParentRoles {
			if parentName == name {
				return fmt.Errorf("cannot delete role '%s': is parent of role '%s'", name, roleName)
			}
		}
	}

	delete(m.roles, name)
	m.invalidateCache()

	m.logger.Info("Role deleted", zap.String("role", name))

	return nil
}

// ListRoles returns all roles
func (m *RBACManager) ListRoles() []*Role {
	m.mu.RLock()
	defer m.mu.RUnlock()

	roles := make([]*Role, 0, len(m.roles))
	for _, role := range m.roles {
		roles = append(roles, role.Clone())
	}
	return roles
}

// AssignRole assigns a role to a user
func (m *RBACManager) AssignRole(userID, roleName string) error {
	if userID == "" {
		return fmt.Errorf("%w: user ID cannot be empty", ErrUserNotFound)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if role exists
	if _, exists := m.roles[roleName]; !exists {
		return fmt.Errorf("%w: role '%s' not found", ErrRoleNotFound, roleName)
	}

	// Get or create user roles
	ur, exists := m.userRoles[userID]
	if !exists {
		ur = NewUserRoles(userID)
		m.userRoles[userID] = ur
	}

	ur.AddRole(roleName)
	m.invalidateCacheForUser(userID)

	m.logger.Info("Role assigned",
		zap.String("user_id", userID),
		zap.String("role", roleName))

	return nil
}

// RevokeRole revokes a role from a user
func (m *RBACManager) RevokeRole(userID, roleName string) error {
	if userID == "" {
		return fmt.Errorf("%w: user ID cannot be empty", ErrUserNotFound)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	ur, exists := m.userRoles[userID]
	if !exists {
		return fmt.Errorf("%w: user '%s' not found", ErrUserNotFound, userID)
	}

	ur.RemoveRole(roleName)
	m.invalidateCacheForUser(userID)

	m.logger.Info("Role revoked",
		zap.String("user_id", userID),
		zap.String("role", roleName))

	return nil
}

// GetUserRoles returns the roles assigned to a user
func (m *RBACManager) GetUserRoles(userID string) ([]string, error) {
	if userID == "" {
		return nil, fmt.Errorf("%w: user ID cannot be empty", ErrUserNotFound)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	ur, exists := m.userRoles[userID]
	if !exists {
		return []string{}, nil
	}

	result := make([]string, len(ur.Roles))
	copy(result, ur.Roles)
	return result, nil
}

// GetUserPermissions returns all permissions for a user (including inherited permissions)
func (m *RBACManager) GetUserPermissions(userID string) (*PermissionSet, error) {
	if userID == "" {
		return nil, fmt.Errorf("%w: user ID cannot be empty", ErrUserNotFound)
	}

	// Check cache first
	if m.cacheEnabled && m.permissionCache != nil {
		if cached, found := m.permissionCache.Get(userID); found {
			return cached, nil
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	ur, exists := m.userRoles[userID]
	if !exists {
		return NewPermissionSet(), nil
	}

	// Compute permissions with inheritance
	permissions := NewPermissionSet()
	visited := make(map[string]bool)

	for _, roleName := range ur.Roles {
		m.collectPermissions(roleName, permissions, visited)
	}

	// Cache the result
	if m.cacheEnabled && m.permissionCache != nil {
		m.permissionCache.Set(userID, permissions)
	}

	return permissions, nil
}

// collectPermissions recursively collects permissions from a role and its parents
func (m *RBACManager) collectPermissions(roleName string, permissions *PermissionSet, visited map[string]bool) {
	// Prevent infinite recursion
	if visited[roleName] {
		return
	}
	visited[roleName] = true

	role, exists := m.roles[roleName]
	if !exists {
		return
	}

	// Add direct permissions
	permissions.Add(role.Permissions.List()...)

	// Recursively add permissions from parent roles
	for _, parentName := range role.ParentRoles {
		m.collectPermissions(parentName, permissions, visited)
	}
}

// CheckPermission checks if a user has a specific permission
func (m *RBACManager) CheckPermission(ctx context.Context, userID string, permission Permission) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		m.logger.Debug("Permission check completed",
			zap.String("user_id", userID),
			zap.String("permission", permission.String()),
			zap.Duration("duration", duration))
	}()

	permissions, err := m.GetUserPermissions(userID)
	if err != nil {
		return err
	}

	if !permissions.Has(permission) {
		m.logger.Warn("Permission denied",
			zap.String("user_id", userID),
			zap.String("permission", permission.String()))
		return fmt.Errorf("%w: user '%s' does not have permission '%s'", ErrPermissionDenied, userID, permission)
	}

	return nil
}

// CheckPermissions checks if a user has all of the given permissions
func (m *RBACManager) CheckPermissions(ctx context.Context, userID string, permissions ...Permission) error {
	userPerms, err := m.GetUserPermissions(userID)
	if err != nil {
		return err
	}

	if !userPerms.HasAll(permissions...) {
		m.logger.Warn("Permission denied (multiple)",
			zap.String("user_id", userID),
			zap.Int("required_permissions", len(permissions)))
		return fmt.Errorf("%w: user '%s' does not have all required permissions", ErrPermissionDenied, userID)
	}

	return nil
}

// CheckAnyPermission checks if a user has any of the given permissions
func (m *RBACManager) CheckAnyPermission(ctx context.Context, userID string, permissions ...Permission) error {
	userPerms, err := m.GetUserPermissions(userID)
	if err != nil {
		return err
	}

	if !userPerms.HasAny(permissions...) {
		m.logger.Warn("Permission denied (any)",
			zap.String("user_id", userID),
			zap.Int("required_permissions", len(permissions)))
		return fmt.Errorf("%w: user '%s' does not have any of the required permissions", ErrPermissionDenied, userID)
	}

	return nil
}

// checkCircularInheritance checks for circular inheritance in role hierarchy
func (m *RBACManager) checkCircularInheritance(roleName string, parentRoles []string) error {
	visited := make(map[string]bool)
	return m.detectCycle(roleName, parentRoles, visited)
}

// detectCycle recursively detects cycles in role inheritance
func (m *RBACManager) detectCycle(currentRole string, parentRoles []string, visited map[string]bool) error {
	for _, parentName := range parentRoles {
		if parentName == currentRole {
			return fmt.Errorf("%w: role '%s' cannot inherit from itself", ErrCircularInheritance, currentRole)
		}

		if visited[parentName] {
			return fmt.Errorf("%w: circular inheritance detected involving role '%s'", ErrCircularInheritance, parentName)
		}

		visited[parentName] = true

		parent, exists := m.roles[parentName]
		if exists {
			if err := m.detectCycle(currentRole, parent.ParentRoles, visited); err != nil {
				return err
			}
		}

		delete(visited, parentName)
	}

	return nil
}

// invalidateCache invalidates the entire permission cache
func (m *RBACManager) invalidateCache() {
	if m.cacheEnabled && m.permissionCache != nil {
		m.permissionCache.Clear()
		m.logger.Debug("Permission cache invalidated")
	}
}

// invalidateCacheForUser invalidates the cache for a specific user
func (m *RBACManager) invalidateCacheForUser(userID string) {
	if m.cacheEnabled && m.permissionCache != nil {
		m.permissionCache.Delete(userID)
		m.logger.Debug("Permission cache invalidated for user", zap.String("user_id", userID))
	}
}

// GetCacheStats returns statistics about the permission cache
func (m *RBACManager) GetCacheStats() map[string]interface{} {
	if !m.cacheEnabled || m.permissionCache == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	return m.permissionCache.Stats()
}

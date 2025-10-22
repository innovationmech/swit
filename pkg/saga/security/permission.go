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
	"fmt"
	"strings"
)

// Permission represents a specific operation that can be performed
type Permission string

const (
	// Saga execution permissions
	PermissionSagaExecute    Permission = "saga:execute"
	PermissionSagaRead       Permission = "saga:read"
	PermissionSagaCancel     Permission = "saga:cancel"
	PermissionSagaRetry      Permission = "saga:retry"
	PermissionSagaRollback   Permission = "saga:rollback"
	PermissionSagaCompensate Permission = "saga:compensate"

	// Saga management permissions
	PermissionSagaCreate Permission = "saga:create"
	PermissionSagaUpdate Permission = "saga:update"
	PermissionSagaDelete Permission = "saga:delete"
	PermissionSagaList   Permission = "saga:list"

	// Step-level permissions
	PermissionStepExecute Permission = "step:execute"
	PermissionStepRead    Permission = "step:read"
	PermissionStepUpdate  Permission = "step:update"

	// Monitoring and audit permissions
	PermissionMonitorView  Permission = "monitor:view"
	PermissionMonitorAdmin Permission = "monitor:admin"
	PermissionAuditView    Permission = "audit:view"
	PermissionAuditExport  Permission = "audit:export"

	// Administrative permissions
	PermissionAdminFull   Permission = "admin:full"
	PermissionRoleManage  Permission = "role:manage"
	PermissionUserManage  Permission = "user:manage"
	PermissionConfigWrite Permission = "config:write"
	PermissionConfigRead  Permission = "config:read"
)

// AllPermissions returns all available permissions
func AllPermissions() []Permission {
	return []Permission{
		// Saga execution
		PermissionSagaExecute,
		PermissionSagaRead,
		PermissionSagaCancel,
		PermissionSagaRetry,
		PermissionSagaRollback,
		PermissionSagaCompensate,
		// Saga management
		PermissionSagaCreate,
		PermissionSagaUpdate,
		PermissionSagaDelete,
		PermissionSagaList,
		// Step-level
		PermissionStepExecute,
		PermissionStepRead,
		PermissionStepUpdate,
		// Monitoring and audit
		PermissionMonitorView,
		PermissionMonitorAdmin,
		PermissionAuditView,
		PermissionAuditExport,
		// Administrative
		PermissionAdminFull,
		PermissionRoleManage,
		PermissionUserManage,
		PermissionConfigWrite,
		PermissionConfigRead,
	}
}

// String returns the string representation of the permission
func (p Permission) String() string {
	return string(p)
}

// Validate checks if the permission is valid
func (p Permission) Validate() error {
	if p == "" {
		return fmt.Errorf("permission cannot be empty")
	}

	// Check if permission follows the pattern "resource:action"
	parts := strings.Split(string(p), ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid permission format: %s (expected 'resource:action')", p)
	}

	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid permission format: %s (resource and action cannot be empty)", p)
	}

	return nil
}

// Resource returns the resource part of the permission (before the colon)
func (p Permission) Resource() string {
	parts := strings.Split(string(p), ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// Action returns the action part of the permission (after the colon)
func (p Permission) Action() string {
	parts := strings.Split(string(p), ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// Matches checks if this permission matches another permission
// Supports wildcard matching (e.g., "saga:*" matches "saga:execute")
func (p Permission) Matches(other Permission) bool {
	if p == other {
		return true
	}

	// Check for wildcard in action
	pResource := p.Resource()
	pAction := p.Action()
	otherResource := other.Resource()
	otherAction := other.Action()

	// If resources don't match, no match
	if pResource != otherResource && pResource != "*" && otherResource != "*" {
		return false
	}

	// Check action matching with wildcard support
	if pAction == "*" || otherAction == "*" {
		return true
	}

	return pAction == otherAction
}

// PermissionSet represents a set of permissions
type PermissionSet struct {
	permissions map[Permission]bool
}

// NewPermissionSet creates a new permission set
func NewPermissionSet(permissions ...Permission) *PermissionSet {
	ps := &PermissionSet{
		permissions: make(map[Permission]bool),
	}
	ps.Add(permissions...)
	return ps
}

// Add adds permissions to the set
func (ps *PermissionSet) Add(permissions ...Permission) {
	for _, p := range permissions {
		ps.permissions[p] = true
	}
}

// Remove removes permissions from the set
func (ps *PermissionSet) Remove(permissions ...Permission) {
	for _, p := range permissions {
		delete(ps.permissions, p)
	}
}

// Has checks if the set has a permission
func (ps *PermissionSet) Has(permission Permission) bool {
	if ps.permissions[permission] {
		return true
	}

	// Check for wildcard matches
	for p := range ps.permissions {
		if p.Matches(permission) {
			return true
		}
	}

	return false
}

// HasAll checks if the set has all of the given permissions
func (ps *PermissionSet) HasAll(permissions ...Permission) bool {
	for _, p := range permissions {
		if !ps.Has(p) {
			return false
		}
	}
	return true
}

// HasAny checks if the set has any of the given permissions
func (ps *PermissionSet) HasAny(permissions ...Permission) bool {
	for _, p := range permissions {
		if ps.Has(p) {
			return true
		}
	}
	return false
}

// List returns all permissions in the set
func (ps *PermissionSet) List() []Permission {
	result := make([]Permission, 0, len(ps.permissions))
	for p := range ps.permissions {
		result = append(result, p)
	}
	return result
}

// Size returns the number of permissions in the set
func (ps *PermissionSet) Size() int {
	return len(ps.permissions)
}

// Clear removes all permissions from the set
func (ps *PermissionSet) Clear() {
	ps.permissions = make(map[Permission]bool)
}

// Clone creates a copy of the permission set
func (ps *PermissionSet) Clone() *PermissionSet {
	clone := NewPermissionSet()
	for p := range ps.permissions {
		clone.Add(p)
	}
	return clone
}

// Union returns a new permission set containing all permissions from both sets
func (ps *PermissionSet) Union(other *PermissionSet) *PermissionSet {
	result := ps.Clone()
	result.Add(other.List()...)
	return result
}

// Intersect returns a new permission set containing only common permissions
func (ps *PermissionSet) Intersect(other *PermissionSet) *PermissionSet {
	result := NewPermissionSet()
	for p := range ps.permissions {
		if other.Has(p) {
			result.Add(p)
		}
	}
	return result
}

// Difference returns a new permission set containing permissions in this set but not in the other
func (ps *PermissionSet) Difference(other *PermissionSet) *PermissionSet {
	result := NewPermissionSet()
	for p := range ps.permissions {
		if !other.Has(p) {
			result.Add(p)
		}
	}
	return result
}

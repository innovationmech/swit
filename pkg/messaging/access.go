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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Permission represents an access control permission
type Permission string

const (
	// PermissionRead grants read access to topics/queues
	PermissionRead Permission = "read"

	// PermissionWrite grants write access to topics/queues
	PermissionWrite Permission = "write"

	// PermissionDelete grants delete access to topics/queues
	PermissionDelete Permission = "delete"

	// PermissionManage grants administrative access to topics/queues
	PermissionManage Permission = "manage"

	// PermissionConsume grants consume access to queues
	PermissionConsume Permission = "consume"

	// PermissionPublish grants publish access to topics
	PermissionPublish Permission = "publish"
)

// AccessResourceType represents the type of resource being accessed
type AccessResourceType string

const (
	// AccessResourceTypeTopic represents a topic resource
	AccessResourceTypeTopic AccessResourceType = "topic"

	// AccessResourceTypeQueue represents a queue resource
	AccessResourceTypeQueue AccessResourceType = "queue"

	// AccessResourceTypeBroker represents a broker resource
	AccessResourceTypeBroker AccessResourceType = "broker"
)

// Resource represents a messaging resource that can be protected
type Resource struct {
	// Type of the resource
	Type AccessResourceType `json:"type"`

	// Name of the resource (topic name, queue name, etc.)
	Name string `json:"name"`

	// Pattern for wildcard matching (e.g., "orders.*")
	Pattern string `json:"pattern,omitempty"`

	// Description of the resource
	Description string `json:"description,omitempty"`

	// Metadata associated with the resource
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewResource creates a new resource
func NewResource(resourceType AccessResourceType, name string) *Resource {
	return &Resource{
		Type:     resourceType,
		Name:     name,
		Metadata: make(map[string]string),
	}
}

// ResourcePattern creates a resource with a wildcard pattern
func ResourcePattern(resourceType AccessResourceType, pattern string) *Resource {
	return &Resource{
		Type:     resourceType,
		Pattern:  pattern,
		Metadata: make(map[string]string),
	}
}

// Matches checks if this resource matches the target resource
func (r *Resource) Matches(target *Resource) bool {
	if r.Type != target.Type {
		return false
	}

	// Exact match
	if r.Name != "" && r.Name == target.Name {
		return true
	}

	// Pattern match
	if r.Pattern != "" {
		// Simple wildcard matching for patterns like "orders.*"
		patternParts := strings.Split(r.Pattern, "*")
		if len(patternParts) == 2 {
			return strings.HasPrefix(target.Name, patternParts[0])
		}
		// Exact pattern match
		return r.Pattern == target.Name
	}

	return false
}

// AccessPolicy represents an access control policy
type AccessPolicy struct {
	// Name of the policy
	Name string `json:"name"`

	// Description of the policy
	Description string `json:"description,omitempty"`

	// Resources this policy applies to
	Resources []*Resource `json:"resources"`

	// Permissions granted by this policy
	Permissions []Permission `json:"permissions"`

	// Conditions that must be satisfied for this policy
	Conditions map[string]interface{} `json:"conditions,omitempty"`

	// Priority of this policy (higher numbers have higher priority)
	Priority int `json:"priority"`

	// Whether this policy is enabled
	Enabled bool `json:"enabled"`

	// Time-based restrictions
	TimeRestrictions *TimeRestrictions `json:"time_restrictions,omitempty"`
}

// TimeRestrictions defines time-based access restrictions
type TimeRestrictions struct {
	// AllowedDays of the week (0=Sunday, 6=Saturday)
	AllowedDays []int `json:"allowed_days,omitempty"`

	// Time ranges when access is allowed
	AllowedTimeRanges []TimeRange `json:"allowed_time_ranges,omitempty"`
}

// TimeRange represents a time range
type TimeRange struct {
	// Start time in HH:MM format
	Start string `json:"start"`

	// End time in HH:MM format
	End string `json:"end"`
}

// Role represents a role with associated policies
type Role struct {
	// Name of the role
	Name string `json:"name"`

	// Description of the role
	Description string `json:"description,omitempty"`

	// Policies assigned to this role
	Policies []*AccessPolicy `json:"policies"`

	// Metadata associated with the role
	Metadata map[string]string `json:"metadata,omitempty"`

	// Whether this role is active
	Active bool `json:"active"`
}

// AccessControlEntry represents an access control entry for a specific entity
type AccessControlEntry struct {
	// Entity type (user, role, service)
	EntityType string `json:"entity_type"`

	// Entity ID (user ID, role name, service name)
	EntityID string `json:"entity_id"`

	// Resources this entry applies to
	Resources []*Resource `json:"resources"`

	// Permissions granted
	Permissions []Permission `json:"permissions"`

	// Whether this is an allow or deny entry
	Effect string `json:"effect"` // "allow" or "deny"

	// Expiration time for this entry
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`
}

// AccessDecision represents an access control decision
type AccessDecision struct {
	// Whether access is allowed
	Allowed bool `json:"allowed"`

	// Reason for the decision
	Reason string `json:"reason"`

	// Policy that made the decision
	Policy *AccessPolicy `json:"policy,omitempty"`

	// Resource being accessed
	Resource *Resource `json:"resource"`

	// Permission being requested
	Permission Permission `json:"permission"`

	// Context used for the decision
	Context *AccessContext `json:"context"`

	// Timestamp of the decision
	Timestamp time.Time `json:"timestamp"`
}

// AccessContext contains context for access control decisions
type AccessContext struct {
	// User ID making the request
	UserID string `json:"user_id"`

	// User roles
	Roles []string `json:"roles"`

	// Authentication context
	AuthContext *AuthContext `json:"auth_context"`

	// Client information
	ClientIP string `json:"client_ip,omitempty"`

	// Request timestamp
	RequestTime time.Time `json:"request_time"`

	// Additional context
	Context map[string]interface{} `json:"context,omitempty"`
}

// NewAccessContext creates a new access context
func NewAccessContext(userID string, roles []string) *AccessContext {
	return &AccessContext{
		UserID:      userID,
		Roles:       roles,
		RequestTime: time.Now(),
		Context:     make(map[string]interface{}),
	}
}

// AccessControlProvider defines the interface for access control providers
type AccessControlProvider interface {
	// Name returns the provider name
	Name() string

	// CheckAccess checks if the given context has permission to access the resource
	CheckAccess(ctx context.Context, accessCtx *AccessContext, resource *Resource, permission Permission) (*AccessDecision, error)

	// GetPermissions returns all permissions for a user/role on a resource
	GetPermissions(ctx context.Context, userID string, roles []string, resource *Resource) ([]Permission, error)

	// AddPolicy adds an access policy
	AddPolicy(ctx context.Context, policy *AccessPolicy) error

	// RemovePolicy removes an access policy
	RemovePolicy(ctx context.Context, policyName string) error

	// AddRole adds a role
	AddRole(ctx context.Context, role *Role) error

	// RemoveRole removes a role
	RemoveRole(ctx context.Context, roleName string) error

	// AddAccessControlEntry adds an access control entry
	AddAccessControlEntry(ctx context.Context, entry *AccessControlEntry) error

	// RemoveAccessControlEntry removes an access control entry
	RemoveAccessControlEntry(ctx context.Context, entryID string) error

	// IsHealthy checks if the access control provider is healthy
	IsHealthy(ctx context.Context) bool
}

// MemoryAccessControlProvider implements an in-memory access control provider
type MemoryAccessControlProvider struct {
	policies map[string]*AccessPolicy
	roles    map[string]*Role
	entries  map[string]*AccessControlEntry
	mu       sync.RWMutex
	cache    *AccessControlCache
	cacheTTL time.Duration
	logger   *zap.Logger
}

// NewMemoryAccessControlProvider creates a new memory-based access control provider
func NewMemoryAccessControlProvider(cacheTTL time.Duration) *MemoryAccessControlProvider {
	provider := &MemoryAccessControlProvider{
		policies: make(map[string]*AccessPolicy),
		roles:    make(map[string]*Role),
		entries:  make(map[string]*AccessControlEntry),
		cacheTTL: cacheTTL,
		logger:   logger.Logger,
	}

	if cacheTTL > 0 {
		provider.cache = NewAccessControlCache(cacheTTL)
	}

	return provider
}

// Name implements AccessControlProvider interface
func (p *MemoryAccessControlProvider) Name() string {
	return "memory"
}

// CheckAccess implements AccessControlProvider interface
func (p *MemoryAccessControlProvider) CheckAccess(ctx context.Context, accessCtx *AccessContext, resource *Resource, permission Permission) (*AccessDecision, error) {
	// Check cache first
	if p.cache != nil {
		if cached, found := p.cache.Get(accessCtx.UserID, resource.Name, permission); found {
			return cached, nil
		}
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	decision := &AccessDecision{
		Resource:   resource,
		Permission: permission,
		Context:    accessCtx,
		Timestamp:  time.Now(),
	}

	// Check direct access control entries first
	for _, entry := range p.entries {
		if p.matchEntry(entry, accessCtx, resource) {
			if p.hasPermission(entry, permission) {
				decision.Allowed = entry.Effect == "allow"
				decision.Reason = fmt.Sprintf("Access %s by entry %s", entry.Effect, entry.EntityID)

				// Cache the decision
				if p.cache != nil {
					p.cache.Set(accessCtx.UserID, resource.Name, permission, decision)
				}

				return decision, nil
			}
		}
	}

	// Check role-based policies
	for _, roleName := range accessCtx.Roles {
		if role, exists := p.roles[roleName]; exists && role.Active {
			for _, policy := range role.Policies {
				if policy.Enabled && p.matchPolicy(policy, resource) {
					if p.hasPolicyPermission(policy, permission) && p.checkConditions(policy, accessCtx) {
						decision.Allowed = true
						decision.Policy = policy
						decision.Reason = fmt.Sprintf("Access allowed by policy %s in role %s", policy.Name, roleName)

						// Cache the decision
						if p.cache != nil {
							p.cache.Set(accessCtx.UserID, resource.Name, permission, decision)
						}

						return decision, nil
					}
				}
			}
		}
	}

	// Default deny
	decision.Allowed = false
	decision.Reason = "Access denied by default"

	// Cache the decision
	if p.cache != nil {
		p.cache.Set(accessCtx.UserID, resource.Name, permission, decision)
	}

	return decision, nil
}

// GetPermissions implements AccessControlProvider interface
func (p *MemoryAccessControlProvider) GetPermissions(ctx context.Context, userID string, roles []string, resource *Resource) ([]Permission, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	permissions := make(map[Permission]bool)

	// Check direct access control entries
	for _, entry := range p.entries {
		if entry.EntityType == "user" && entry.EntityID == userID {
			for _, res := range entry.Resources {
				if res.Matches(resource) {
					if entry.Effect == "allow" {
						for _, perm := range entry.Permissions {
							permissions[perm] = true
						}
					}
				}
			}
		}
	}

	// Check role-based permissions
	for _, roleName := range roles {
		if role, exists := p.roles[roleName]; exists && role.Active {
			for _, policy := range role.Policies {
				if policy.Enabled && p.matchPolicy(policy, resource) {
					for _, perm := range policy.Permissions {
						permissions[perm] = true
					}
				}
			}
		}
	}

	// Convert map to slice
	result := make([]Permission, 0, len(permissions))
	for perm := range permissions {
		result = append(result, perm)
	}

	return result, nil
}

// AddPolicy implements AccessControlProvider interface
func (p *MemoryAccessControlProvider) AddPolicy(ctx context.Context, policy *AccessPolicy) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.policies[policy.Name] = policy

	// Clear cache when policies change
	if p.cache != nil {
		p.cache.Clear()
	}

	return nil
}

// RemovePolicy implements AccessControlProvider interface
func (p *MemoryAccessControlProvider) RemovePolicy(ctx context.Context, policyName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.policies, policyName)

	// Remove from roles
	for _, role := range p.roles {
		filtered := make([]*AccessPolicy, 0, len(role.Policies))
		for _, policy := range role.Policies {
			if policy.Name != policyName {
				filtered = append(filtered, policy)
			}
		}
		role.Policies = filtered
	}

	// Clear cache
	if p.cache != nil {
		p.cache.Clear()
	}

	return nil
}

// AddRole implements AccessControlProvider interface
func (p *MemoryAccessControlProvider) AddRole(ctx context.Context, role *Role) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.roles[role.Name] = role

	// Clear cache
	if p.cache != nil {
		p.cache.Clear()
	}

	return nil
}

// RemoveRole implements AccessControlProvider interface
func (p *MemoryAccessControlProvider) RemoveRole(ctx context.Context, roleName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.roles, roleName)

	// Clear cache
	if p.cache != nil {
		p.cache.Clear()
	}

	return nil
}

// AddAccessControlEntry implements AccessControlProvider interface
func (p *MemoryAccessControlProvider) AddAccessControlEntry(ctx context.Context, entry *AccessControlEntry) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	resourceType := ""
	if len(entry.Resources) > 0 {
		resourceType = string(entry.Resources[0].Type)
	}
	entryID := fmt.Sprintf("%s:%s:%s", entry.EntityType, entry.EntityID, resourceType)
	p.entries[entryID] = entry

	// Clear cache
	if p.cache != nil {
		p.cache.Clear()
	}

	return nil
}

// RemoveAccessControlEntry implements AccessControlProvider interface
func (p *MemoryAccessControlProvider) RemoveAccessControlEntry(ctx context.Context, entryID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.entries, entryID)

	// Clear cache
	if p.cache != nil {
		p.cache.Clear()
	}

	return nil
}

// IsHealthy implements AccessControlProvider interface
func (p *MemoryAccessControlProvider) IsHealthy(ctx context.Context) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Basic health check - ensure maps are initialized
	return p.policies != nil && p.roles != nil && p.entries != nil
}

// Helper methods

func (p *MemoryAccessControlProvider) matchEntry(entry *AccessControlEntry, accessCtx *AccessContext, resource *Resource) bool {
	// Check if entry matches the user or role
	if entry.EntityType == "user" && entry.EntityID != accessCtx.UserID {
		return false
	}

	if entry.EntityType == "role" {
		roleMatch := false
		for _, role := range accessCtx.Roles {
			if role == entry.EntityID {
				roleMatch = true
				break
			}
		}
		if !roleMatch {
			return false
		}
	}

	// Check if entry has expired
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		return false
	}

	// Check resource match
	for _, res := range entry.Resources {
		if res.Matches(resource) {
			return true
		}
	}

	return false
}

func (p *MemoryAccessControlProvider) hasPermission(entry *AccessControlEntry, permission Permission) bool {
	for _, perm := range entry.Permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

func (p *MemoryAccessControlProvider) matchPolicy(policy *AccessPolicy, resource *Resource) bool {
	for _, res := range policy.Resources {
		if res.Matches(resource) {
			return true
		}
	}
	return false
}

func (p *MemoryAccessControlProvider) hasPolicyPermission(policy *AccessPolicy, permission Permission) bool {
	for _, perm := range policy.Permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

func (p *MemoryAccessControlProvider) checkConditions(policy *AccessPolicy, accessCtx *AccessContext) bool {
	// Check time restrictions
	if policy.TimeRestrictions != nil {
		now := time.Now()

		// Check allowed days
		if len(policy.TimeRestrictions.AllowedDays) > 0 {
			weekday := int(now.Weekday())
			dayAllowed := false
			for _, allowedDay := range policy.TimeRestrictions.AllowedDays {
				if weekday == allowedDay {
					dayAllowed = true
					break
				}
			}
			if !dayAllowed {
				return false
			}
		}

		// Check time ranges
		if len(policy.TimeRestrictions.AllowedTimeRanges) > 0 {
			currentTime := now.Format("15:04")
			timeAllowed := false
			for _, timeRange := range policy.TimeRestrictions.AllowedTimeRanges {
				if currentTime >= timeRange.Start && currentTime <= timeRange.End {
					timeAllowed = true
					break
				}
			}
			if !timeAllowed {
				return false
			}
		}
	}

	// Additional conditions can be checked here
	return true
}

// AccessControlCache provides caching for access control decisions
type AccessControlCache struct {
	items map[string]*AccessDecision
	ttl   time.Duration
	mu    sync.RWMutex
}

// NewAccessControlCache creates a new access control cache
func NewAccessControlCache(ttl time.Duration) *AccessControlCache {
	cache := &AccessControlCache{
		items: make(map[string]*AccessDecision),
		ttl:   ttl,
	}

	// Start cleanup routine
	go cache.cleanup()

	return cache
}

// Get retrieves an access decision from the cache
func (c *AccessControlCache) Get(userID string, resourceName string, permission Permission) (*AccessDecision, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := fmt.Sprintf("%s:%s:%s", userID, resourceName, permission)
	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if item has expired
	if time.Since(item.Timestamp) > c.ttl {
		return nil, false
	}

	return item, true
}

// Set stores an access decision in the cache
func (c *AccessControlCache) Set(userID string, resourceName string, permission Permission, decision *AccessDecision) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", userID, resourceName, permission)
	c.items[key] = decision
}

// Clear clears all items from the cache
func (c *AccessControlCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*AccessDecision)
}

// cleanup periodically removes expired items from the cache
func (c *AccessControlCache) cleanup() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.Sub(item.Timestamp) > c.ttl {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

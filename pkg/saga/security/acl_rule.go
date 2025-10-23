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
	"time"
)

// ACLEffect represents the effect of an ACL rule (Allow or Deny)
type ACLEffect string

const (
	// ACLEffectAllow grants access
	ACLEffectAllow ACLEffect = "Allow"

	// ACLEffectDeny denies access
	ACLEffectDeny ACLEffect = "Deny"
)

// String returns the string representation of the ACL effect
func (e ACLEffect) String() string {
	return string(e)
}

// Validate checks if the ACL effect is valid
func (e ACLEffect) Validate() error {
	if e != ACLEffectAllow && e != ACLEffectDeny {
		return fmt.Errorf("invalid ACL effect: %s (must be Allow or Deny)", e)
	}
	return nil
}

// ACLRule represents an access control rule
type ACLRule struct {
	// ID is the unique identifier for the rule
	ID string

	// Priority determines the order of rule evaluation (higher priority = evaluated first)
	// Range: 0-1000, where 1000 is highest priority
	Priority int

	// Effect determines whether the rule allows or denies access
	Effect ACLEffect

	// Principal identifies who the rule applies to (user ID, role, or group)
	// Supports wildcards: "*" matches all principals
	Principal string

	// Resource identifies what resource the rule applies to
	// Examples:
	//   - "saga:order-123" - specific saga instance
	//   - "saga:order-*" - all order sagas
	//   - "saga:*" - all sagas
	//   - "*" - all resources
	Resource string

	// Action is the operation being performed
	// Examples: "execute", "read", "cancel", "retry", "*"
	Action string

	// Conditions are additional constraints for rule matching
	// Examples: time-based, IP-based, attribute-based conditions
	Conditions map[string]string

	// Description provides context about the rule
	Description string

	// Metadata for additional rule properties
	Metadata map[string]string

	// CreatedAt is when the rule was created
	CreatedAt time.Time

	// UpdatedAt is when the rule was last updated
	UpdatedAt time.Time

	// ExpiresAt is when the rule expires (nil means never expires)
	ExpiresAt *time.Time
}

// NewACLRule creates a new ACL rule
func NewACLRule(id string, priority int, effect ACLEffect, principal, resource, action string) *ACLRule {
	now := time.Now()
	return &ACLRule{
		ID:         id,
		Priority:   priority,
		Effect:     effect,
		Principal:  principal,
		Resource:   resource,
		Action:     action,
		Conditions: make(map[string]string),
		Metadata:   make(map[string]string),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// Validate checks if the ACL rule is valid
func (r *ACLRule) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("ACL rule ID cannot be empty")
	}

	if r.Priority < 0 || r.Priority > 1000 {
		return fmt.Errorf("ACL rule priority must be between 0 and 1000, got %d", r.Priority)
	}

	if err := r.Effect.Validate(); err != nil {
		return err
	}

	if r.Principal == "" {
		return fmt.Errorf("ACL rule principal cannot be empty")
	}

	if r.Resource == "" {
		return fmt.Errorf("ACL rule resource cannot be empty")
	}

	if r.Action == "" {
		return fmt.Errorf("ACL rule action cannot be empty")
	}

	return nil
}

// IsExpired checks if the rule has expired
func (r *ACLRule) IsExpired() bool {
	if r.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*r.ExpiresAt)
}

// Matches checks if the rule matches the given context
func (r *ACLRule) Matches(ctx *ACLContext) bool {
	// Check if rule has expired
	if r.IsExpired() {
		return false
	}

	// Check principal matching
	if !r.matchesPrincipal(ctx.Principal) {
		return false
	}

	// Check resource matching
	if !r.matchesResource(ctx.Resource) {
		return false
	}

	// Check action matching
	if !r.matchesAction(ctx.Action) {
		return false
	}

	// Check conditions
	if !r.matchesConditions(ctx) {
		return false
	}

	return true
}

// matchesPrincipal checks if the principal matches (supports wildcards)
func (r *ACLRule) matchesPrincipal(principal string) bool {
	return matchesPattern(r.Principal, principal)
}

// matchesResource checks if the resource matches (supports wildcards)
func (r *ACLRule) matchesResource(resource string) bool {
	return matchesPattern(r.Resource, resource)
}

// matchesAction checks if the action matches (supports wildcards)
func (r *ACLRule) matchesAction(action string) bool {
	return matchesPattern(r.Action, action)
}

// matchesConditions checks if all conditions are satisfied
func (r *ACLRule) matchesConditions(ctx *ACLContext) bool {
	if len(r.Conditions) == 0 {
		return true
	}

	for key, expectedValue := range r.Conditions {
		actualValue, exists := ctx.Attributes[key]
		if !exists {
			return false
		}

		if !matchesPattern(expectedValue, actualValue) {
			return false
		}
	}

	return true
}

// Clone creates a deep copy of the rule
func (r *ACLRule) Clone() *ACLRule {
	clone := &ACLRule{
		ID:          r.ID,
		Priority:    r.Priority,
		Effect:      r.Effect,
		Principal:   r.Principal,
		Resource:    r.Resource,
		Action:      r.Action,
		Description: r.Description,
		Conditions:  make(map[string]string),
		Metadata:    make(map[string]string),
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	for k, v := range r.Conditions {
		clone.Conditions[k] = v
	}

	for k, v := range r.Metadata {
		clone.Metadata[k] = v
	}

	if r.ExpiresAt != nil {
		expiresAt := *r.ExpiresAt
		clone.ExpiresAt = &expiresAt
	}

	return clone
}

// ACLContext represents the context for ACL evaluation
type ACLContext struct {
	// Principal is the entity requesting access (user ID, role, etc.)
	Principal string

	// Resource is the target resource being accessed
	Resource string

	// Action is the operation being performed
	Action string

	// Attributes are additional context attributes for condition evaluation
	// Examples: "ip_address", "time", "user_agent", "request_id"
	Attributes map[string]string

	// Timestamp is when the access is being evaluated
	Timestamp time.Time
}

// NewACLContext creates a new ACL context
func NewACLContext(principal, resource, action string) *ACLContext {
	return &ACLContext{
		Principal:  principal,
		Resource:   resource,
		Action:     action,
		Attributes: make(map[string]string),
		Timestamp:  time.Now(),
	}
}

// WithAttribute adds an attribute to the context
func (c *ACLContext) WithAttribute(key, value string) *ACLContext {
	c.Attributes[key] = value
	return c
}

// WithAttributes adds multiple attributes to the context
func (c *ACLContext) WithAttributes(attributes map[string]string) *ACLContext {
	for k, v := range attributes {
		c.Attributes[k] = v
	}
	return c
}

// Validate checks if the ACL context is valid
func (c *ACLContext) Validate() error {
	if c.Principal == "" {
		return fmt.Errorf("ACL context principal cannot be empty")
	}

	if c.Resource == "" {
		return fmt.Errorf("ACL context resource cannot be empty")
	}

	if c.Action == "" {
		return fmt.Errorf("ACL context action cannot be empty")
	}

	return nil
}

// matchesPattern checks if a pattern matches a value
// Supports wildcards: "*" matches anything, "prefix*" matches prefix
func matchesPattern(pattern, value string) bool {
	// Exact match
	if pattern == value {
		return true
	}

	// Wildcard matches everything
	if pattern == "*" {
		return true
	}

	// Prefix matching with wildcard
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}

	// Suffix matching with wildcard
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(value, suffix)
	}

	// No match
	return false
}

// ACLDecision represents the result of an ACL evaluation
type ACLDecision struct {
	// Allowed indicates whether access is granted
	Allowed bool

	// Effect is the final effect (Allow or Deny)
	Effect ACLEffect

	// MatchedRule is the rule that made the decision (if any)
	MatchedRule *ACLRule

	// Reason describes why the decision was made
	Reason string

	// EvaluatedRules is the number of rules that were evaluated
	EvaluatedRules int

	// EvaluationTime is how long the evaluation took
	EvaluationTime time.Duration
}

// NewACLDecision creates a new ACL decision
func NewACLDecision(allowed bool, effect ACLEffect, matchedRule *ACLRule, reason string) *ACLDecision {
	return &ACLDecision{
		Allowed:     allowed,
		Effect:      effect,
		MatchedRule: matchedRule,
		Reason:      reason,
	}
}

// String returns a string representation of the decision
func (d *ACLDecision) String() string {
	if d.Allowed {
		return fmt.Sprintf("ALLOW: %s", d.Reason)
	}
	return fmt.Sprintf("DENY: %s", d.Reason)
}

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
	"sort"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// Common ACL errors
var (
	ErrACLRuleNotFound      = errors.New("ACL rule not found")
	ErrACLRuleAlreadyExists = errors.New("ACL rule already exists")
	ErrInvalidACLRule       = errors.New("invalid ACL rule")
	ErrInvalidACLContext    = errors.New("invalid ACL context")
	ErrAccessDenied         = errors.New("access denied by ACL")
)

// ACLManager manages ACL rules and performs access control decisions
type ACLManager struct {
	// rules stores all ACL rules indexed by ID
	rules map[string]*ACLRule

	// sortedRules stores rules sorted by priority (highest first)
	sortedRules []*ACLRule

	// defaultEffect is the default decision when no rules match
	defaultEffect ACLEffect

	// rbacManager is the optional RBAC manager for role resolution
	rbacManager *RBACManager

	// mu protects concurrent access to rules
	mu sync.RWMutex

	// logger for ACL operations
	logger *zap.Logger

	// metrics tracks ACL performance
	metrics *ACLMetrics
}

// ACLManagerConfig configures the ACL manager
type ACLManagerConfig struct {
	// DefaultEffect is the default decision when no rules match (default: Deny)
	DefaultEffect ACLEffect

	// RBACManager is the optional RBAC manager for role resolution
	RBACManager *RBACManager

	// EnableMetrics enables performance metrics collection
	EnableMetrics bool
}

// ACLMetrics tracks ACL performance metrics
type ACLMetrics struct {
	mu               sync.RWMutex
	totalEvaluations int64
	allowedCount     int64
	deniedCount      int64
	averageEvalTime  time.Duration
	totalEvalTime    time.Duration
	cacheHits        int64
	cacheMisses      int64
	rulesEvaluated   int64
}

// NewACLManager creates a new ACL manager
func NewACLManager(config *ACLManagerConfig) *ACLManager {
	if config == nil {
		config = &ACLManagerConfig{
			DefaultEffect: ACLEffectDeny,
			EnableMetrics: true,
		}
	}

	if config.DefaultEffect == "" {
		config.DefaultEffect = ACLEffectDeny
	}

	manager := &ACLManager{
		rules:         make(map[string]*ACLRule),
		sortedRules:   make([]*ACLRule, 0),
		defaultEffect: config.DefaultEffect,
		rbacManager:   config.RBACManager,
		logger:        logger.Logger,
	}

	if manager.logger == nil {
		manager.logger = zap.NewNop()
	}

	if config.EnableMetrics {
		manager.metrics = &ACLMetrics{}
	}

	return manager
}

// AddRule adds a new ACL rule
func (m *ACLManager) AddRule(rule *ACLRule) error {
	if rule == nil {
		return ErrInvalidACLRule
	}

	if err := rule.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidACLRule, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.rules[rule.ID]; exists {
		return fmt.Errorf("%w: rule '%s' already exists", ErrACLRuleAlreadyExists, rule.ID)
	}

	m.rules[rule.ID] = rule.Clone()
	m.rebuildSortedRules()

	m.logger.Info("ACL rule added",
		zap.String("rule_id", rule.ID),
		zap.Int("priority", rule.Priority),
		zap.String("effect", rule.Effect.String()))

	return nil
}

// GetRule retrieves an ACL rule by ID
func (m *ACLManager) GetRule(ruleID string) (*ACLRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rule, exists := m.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("%w: rule '%s' not found", ErrACLRuleNotFound, ruleID)
	}

	return rule.Clone(), nil
}

// UpdateRule updates an existing ACL rule
func (m *ACLManager) UpdateRule(rule *ACLRule) error {
	if rule == nil {
		return ErrInvalidACLRule
	}

	if err := rule.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidACLRule, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.rules[rule.ID]; !exists {
		return fmt.Errorf("%w: rule '%s' not found", ErrACLRuleNotFound, rule.ID)
	}

	rule.UpdatedAt = time.Now()
	m.rules[rule.ID] = rule.Clone()
	m.rebuildSortedRules()

	m.logger.Info("ACL rule updated",
		zap.String("rule_id", rule.ID),
		zap.Int("priority", rule.Priority))

	return nil
}

// DeleteRule deletes an ACL rule
func (m *ACLManager) DeleteRule(ruleID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.rules[ruleID]; !exists {
		return fmt.Errorf("%w: rule '%s' not found", ErrACLRuleNotFound, ruleID)
	}

	delete(m.rules, ruleID)
	m.rebuildSortedRules()

	m.logger.Info("ACL rule deleted", zap.String("rule_id", ruleID))

	return nil
}

// ListRules returns all ACL rules
func (m *ACLManager) ListRules() []*ACLRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*ACLRule, 0, len(m.rules))
	for _, rule := range m.rules {
		rules = append(rules, rule.Clone())
	}
	return rules
}

// ListRulesByPrincipal returns all rules for a specific principal
func (m *ACLManager) ListRulesByPrincipal(principal string) []*ACLRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*ACLRule, 0)
	for _, rule := range m.rules {
		if rule.matchesPrincipal(principal) {
			rules = append(rules, rule.Clone())
		}
	}
	return rules
}

// ListRulesByResource returns all rules for a specific resource
func (m *ACLManager) ListRulesByResource(resource string) []*ACLRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*ACLRule, 0)
	for _, rule := range m.rules {
		if rule.matchesResource(resource) {
			rules = append(rules, rule.Clone())
		}
	}
	return rules
}

// Evaluate evaluates ACL rules for the given context
func (m *ACLManager) Evaluate(ctx context.Context, aclCtx *ACLContext) (*ACLDecision, error) {
	start := time.Now()

	if aclCtx == nil {
		return nil, ErrInvalidACLContext
	}

	if err := aclCtx.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidACLContext, err)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Evaluate rules in priority order (highest priority first)
	var evaluatedCount int
	for _, rule := range m.sortedRules {
		evaluatedCount++

		// Skip expired rules
		if rule.IsExpired() {
			continue
		}

		// Check if rule matches the context
		if rule.Matches(aclCtx) {
			duration := time.Since(start)
			decision := NewACLDecision(
				rule.Effect == ACLEffectAllow,
				rule.Effect,
				rule,
				fmt.Sprintf("Matched rule: %s (priority: %d)", rule.ID, rule.Priority),
			)
			decision.EvaluatedRules = evaluatedCount
			decision.EvaluationTime = duration

			m.recordMetrics(decision)

			m.logger.Debug("ACL rule matched",
				zap.String("rule_id", rule.ID),
				zap.String("principal", aclCtx.Principal),
				zap.String("resource", aclCtx.Resource),
				zap.String("action", aclCtx.Action),
				zap.String("effect", rule.Effect.String()),
				zap.Duration("duration", duration))

			return decision, nil
		}
	}

	// No matching rule found, use default effect
	duration := time.Since(start)
	decision := NewACLDecision(
		m.defaultEffect == ACLEffectAllow,
		m.defaultEffect,
		nil,
		fmt.Sprintf("No matching rule, using default effect: %s", m.defaultEffect),
	)
	decision.EvaluatedRules = evaluatedCount
	decision.EvaluationTime = duration

	m.recordMetrics(decision)

	m.logger.Debug("No ACL rule matched, using default",
		zap.String("principal", aclCtx.Principal),
		zap.String("resource", aclCtx.Resource),
		zap.String("action", aclCtx.Action),
		zap.String("default_effect", m.defaultEffect.String()),
		zap.Duration("duration", duration))

	return decision, nil
}

// CheckAccess is a convenience method that returns an error if access is denied
func (m *ACLManager) CheckAccess(ctx context.Context, aclCtx *ACLContext) error {
	decision, err := m.Evaluate(ctx, aclCtx)
	if err != nil {
		return err
	}

	if !decision.Allowed {
		m.logger.Warn("Access denied",
			zap.String("principal", aclCtx.Principal),
			zap.String("resource", aclCtx.Resource),
			zap.String("action", aclCtx.Action),
			zap.String("reason", decision.Reason))
		return fmt.Errorf("%w: %s", ErrAccessDenied, decision.Reason)
	}

	return nil
}

// CheckAccessForSaga checks access for a specific saga instance
func (m *ACLManager) CheckAccessForSaga(ctx context.Context, userID, sagaID, action string) error {
	aclCtx := NewACLContext(
		userID,
		fmt.Sprintf("saga:%s", sagaID),
		action,
	)
	return m.CheckAccess(ctx, aclCtx)
}

// CheckAccessWithRoles checks access considering user's RBAC roles
// This integrates ACL with RBAC by checking both user ID and user's roles
func (m *ACLManager) CheckAccessWithRoles(ctx context.Context, userID, resource, action string) error {
	// First check direct user ACL
	aclCtx := NewACLContext(userID, resource, action)
	decision, err := m.Evaluate(ctx, aclCtx)
	if err != nil {
		return err
	}

	if decision.Allowed {
		return nil
	}

	// If RBAC manager is available, check user's roles
	if m.rbacManager != nil {
		roles, err := m.rbacManager.GetUserRoles(userID)
		if err == nil {
			// Check ACL for each role
			for _, role := range roles {
				roleCtx := NewACLContext(fmt.Sprintf("role:%s", role), resource, action)
				roleDecision, err := m.Evaluate(ctx, roleCtx)
				if err == nil && roleDecision.Allowed {
					m.logger.Debug("Access granted via role",
						zap.String("user_id", userID),
						zap.String("role", role),
						zap.String("resource", resource))
					return nil
				}
			}
		}
	}

	m.logger.Warn("Access denied (including role check)",
		zap.String("user_id", userID),
		zap.String("resource", resource),
		zap.String("action", action))
	return fmt.Errorf("%w: user '%s' cannot perform '%s' on resource '%s'", ErrAccessDenied, userID, action, resource)
}

// CleanupExpiredRules removes expired rules
func (m *ACLManager) CleanupExpiredRules() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	var removedCount int
	for id, rule := range m.rules {
		if rule.IsExpired() {
			delete(m.rules, id)
			removedCount++
		}
	}

	if removedCount > 0 {
		m.rebuildSortedRules()
		m.logger.Info("Cleaned up expired ACL rules", zap.Int("count", removedCount))
	}

	return removedCount
}

// GetMetrics returns ACL performance metrics
func (m *ACLManager) GetMetrics() map[string]interface{} {
	if m.metrics == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	return map[string]interface{}{
		"enabled":           true,
		"total_evaluations": m.metrics.totalEvaluations,
		"allowed_count":     m.metrics.allowedCount,
		"denied_count":      m.metrics.deniedCount,
		"average_eval_time": m.metrics.averageEvalTime.String(),
		"total_eval_time":   m.metrics.totalEvalTime.String(),
		"cache_hits":        m.metrics.cacheHits,
		"cache_misses":      m.metrics.cacheMisses,
		"rules_evaluated":   m.metrics.rulesEvaluated,
		"total_rules":       len(m.rules),
	}
}

// ResetMetrics resets all performance metrics
func (m *ACLManager) ResetMetrics() {
	if m.metrics == nil {
		return
	}

	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.totalEvaluations = 0
	m.metrics.allowedCount = 0
	m.metrics.deniedCount = 0
	m.metrics.averageEvalTime = 0
	m.metrics.totalEvalTime = 0
	m.metrics.cacheHits = 0
	m.metrics.cacheMisses = 0
	m.metrics.rulesEvaluated = 0
}

// rebuildSortedRules rebuilds the sorted rules list (must be called with lock held)
func (m *ACLManager) rebuildSortedRules() {
	m.sortedRules = make([]*ACLRule, 0, len(m.rules))
	for _, rule := range m.rules {
		m.sortedRules = append(m.sortedRules, rule)
	}

	// Sort by priority (highest first), then by creation time (oldest first)
	sort.Slice(m.sortedRules, func(i, j int) bool {
		if m.sortedRules[i].Priority != m.sortedRules[j].Priority {
			return m.sortedRules[i].Priority > m.sortedRules[j].Priority
		}
		return m.sortedRules[i].CreatedAt.Before(m.sortedRules[j].CreatedAt)
	})
}

// recordMetrics records metrics for an evaluation
func (m *ACLManager) recordMetrics(decision *ACLDecision) {
	if m.metrics == nil {
		return
	}

	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.totalEvaluations++
	if decision.Allowed {
		m.metrics.allowedCount++
	} else {
		m.metrics.deniedCount++
	}

	m.metrics.totalEvalTime += decision.EvaluationTime
	m.metrics.averageEvalTime = m.metrics.totalEvalTime / time.Duration(m.metrics.totalEvaluations)
	m.metrics.rulesEvaluated += int64(decision.EvaluatedRules)
}

// ExportRules exports all rules in a format suitable for backup/restore
func (m *ACLManager) ExportRules() []*ACLRule {
	return m.ListRules()
}

// ImportRules imports rules from a backup (replaces existing rules)
func (m *ACLManager) ImportRules(rules []*ACLRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate all rules first
	for _, rule := range rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("invalid rule '%s': %w", rule.ID, err)
		}
	}

	// Clear existing rules
	m.rules = make(map[string]*ACLRule)

	// Import new rules
	for _, rule := range rules {
		m.rules[rule.ID] = rule.Clone()
	}

	m.rebuildSortedRules()

	m.logger.Info("ACL rules imported", zap.Int("count", len(rules)))

	return nil
}

// Clear removes all ACL rules
func (m *ACLManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rules = make(map[string]*ACLRule)
	m.sortedRules = make([]*ACLRule, 0)

	m.logger.Info("All ACL rules cleared")
}

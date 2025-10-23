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
	"testing"
	"time"
)

func TestNewACLRule(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		priority  int
		effect    ACLEffect
		principal string
		resource  string
		action    string
	}{
		{
			name:      "Create valid allow rule",
			id:        "rule-1",
			priority:  100,
			effect:    ACLEffectAllow,
			principal: "user:123",
			resource:  "saga:order-456",
			action:    "execute",
		},
		{
			name:      "Create valid deny rule",
			id:        "rule-2",
			priority:  200,
			effect:    ACLEffectDeny,
			principal: "user:789",
			resource:  "saga:*",
			action:    "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewACLRule(tt.id, tt.priority, tt.effect, tt.principal, tt.resource, tt.action)

			if rule.ID != tt.id {
				t.Errorf("Expected ID %s, got %s", tt.id, rule.ID)
			}
			if rule.Priority != tt.priority {
				t.Errorf("Expected Priority %d, got %d", tt.priority, rule.Priority)
			}
			if rule.Effect != tt.effect {
				t.Errorf("Expected Effect %s, got %s", tt.effect, rule.Effect)
			}
			if rule.Principal != tt.principal {
				t.Errorf("Expected Principal %s, got %s", tt.principal, rule.Principal)
			}
			if rule.Resource != tt.resource {
				t.Errorf("Expected Resource %s, got %s", tt.resource, rule.Resource)
			}
			if rule.Action != tt.action {
				t.Errorf("Expected Action %s, got %s", tt.action, rule.Action)
			}
			if rule.Conditions == nil {
				t.Error("Expected Conditions to be initialized")
			}
			if rule.Metadata == nil {
				t.Error("Expected Metadata to be initialized")
			}
		})
	}
}

func TestACLRuleValidate(t *testing.T) {
	tests := []struct {
		name      string
		rule      *ACLRule
		wantError bool
	}{
		{
			name:      "Valid rule",
			rule:      NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute"),
			wantError: false,
		},
		{
			name: "Empty ID",
			rule: &ACLRule{
				ID:        "",
				Priority:  100,
				Effect:    ACLEffectAllow,
				Principal: "user:123",
				Resource:  "saga:*",
				Action:    "execute",
			},
			wantError: true,
		},
		{
			name: "Invalid priority (negative)",
			rule: &ACLRule{
				ID:        "rule-1",
				Priority:  -1,
				Effect:    ACLEffectAllow,
				Principal: "user:123",
				Resource:  "saga:*",
				Action:    "execute",
			},
			wantError: true,
		},
		{
			name: "Invalid priority (too high)",
			rule: &ACLRule{
				ID:        "rule-1",
				Priority:  1001,
				Effect:    ACLEffectAllow,
				Principal: "user:123",
				Resource:  "saga:*",
				Action:    "execute",
			},
			wantError: true,
		},
		{
			name: "Invalid effect",
			rule: &ACLRule{
				ID:        "rule-1",
				Priority:  100,
				Effect:    ACLEffect("Invalid"),
				Principal: "user:123",
				Resource:  "saga:*",
				Action:    "execute",
			},
			wantError: true,
		},
		{
			name: "Empty principal",
			rule: &ACLRule{
				ID:        "rule-1",
				Priority:  100,
				Effect:    ACLEffectAllow,
				Principal: "",
				Resource:  "saga:*",
				Action:    "execute",
			},
			wantError: true,
		},
		{
			name: "Empty resource",
			rule: &ACLRule{
				ID:        "rule-1",
				Priority:  100,
				Effect:    ACLEffectAllow,
				Principal: "user:123",
				Resource:  "",
				Action:    "execute",
			},
			wantError: true,
		},
		{
			name: "Empty action",
			rule: &ACLRule{
				ID:        "rule-1",
				Priority:  100,
				Effect:    ACLEffectAllow,
				Principal: "user:123",
				Resource:  "saga:*",
				Action:    "",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestACLRuleMatches(t *testing.T) {
	tests := []struct {
		name    string
		rule    *ACLRule
		context *ACLContext
		want    bool
	}{
		{
			name:    "Exact match",
			rule:    NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:order-456", "execute"),
			context: NewACLContext("user:123", "saga:order-456", "execute"),
			want:    true,
		},
		{
			name:    "Wildcard action match",
			rule:    NewACLRule("rule-2", 100, ACLEffectAllow, "user:123", "saga:order-456", "*"),
			context: NewACLContext("user:123", "saga:order-456", "execute"),
			want:    true,
		},
		{
			name:    "Wildcard resource match",
			rule:    NewACLRule("rule-3", 100, ACLEffectAllow, "user:123", "saga:*", "execute"),
			context: NewACLContext("user:123", "saga:order-456", "execute"),
			want:    true,
		},
		{
			name:    "Wildcard principal match",
			rule:    NewACLRule("rule-4", 100, ACLEffectAllow, "*", "saga:order-456", "execute"),
			context: NewACLContext("user:123", "saga:order-456", "execute"),
			want:    true,
		},
		{
			name:    "All wildcards match",
			rule:    NewACLRule("rule-5", 100, ACLEffectAllow, "*", "*", "*"),
			context: NewACLContext("user:123", "saga:order-456", "execute"),
			want:    true,
		},
		{
			name:    "Principal mismatch",
			rule:    NewACLRule("rule-6", 100, ACLEffectAllow, "user:123", "saga:order-456", "execute"),
			context: NewACLContext("user:789", "saga:order-456", "execute"),
			want:    false,
		},
		{
			name:    "Resource mismatch",
			rule:    NewACLRule("rule-7", 100, ACLEffectAllow, "user:123", "saga:order-456", "execute"),
			context: NewACLContext("user:123", "saga:payment-789", "execute"),
			want:    false,
		},
		{
			name:    "Action mismatch",
			rule:    NewACLRule("rule-8", 100, ACLEffectAllow, "user:123", "saga:order-456", "execute"),
			context: NewACLContext("user:123", "saga:order-456", "cancel"),
			want:    false,
		},
		{
			name:    "Prefix wildcard resource match",
			rule:    NewACLRule("rule-9", 100, ACLEffectAllow, "user:123", "saga:order-*", "execute"),
			context: NewACLContext("user:123", "saga:order-456", "execute"),
			want:    true,
		},
		{
			name:    "Prefix wildcard principal match",
			rule:    NewACLRule("rule-10", 100, ACLEffectAllow, "user:*", "saga:order-456", "execute"),
			context: NewACLContext("user:123", "saga:order-456", "execute"),
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.Matches(tt.context)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestACLRuleMatchesWithConditions(t *testing.T) {
	rule := NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute")
	rule.Conditions["ip_address"] = "192.168.1.*"
	rule.Conditions["time"] = "business_hours"

	tests := []struct {
		name    string
		context *ACLContext
		want    bool
	}{
		{
			name: "All conditions match",
			context: NewACLContext("user:123", "saga:order-456", "execute").
				WithAttribute("ip_address", "192.168.1.100").
				WithAttribute("time", "business_hours"),
			want: true,
		},
		{
			name: "One condition mismatch",
			context: NewACLContext("user:123", "saga:order-456", "execute").
				WithAttribute("ip_address", "10.0.0.1").
				WithAttribute("time", "business_hours"),
			want: false,
		},
		{
			name: "Missing condition",
			context: NewACLContext("user:123", "saga:order-456", "execute").
				WithAttribute("ip_address", "192.168.1.100"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rule.Matches(tt.context)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestACLRuleIsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "Never expires",
			expiresAt: nil,
			want:      false,
		},
		{
			name:      "Already expired",
			expiresAt: &past,
			want:      true,
		},
		{
			name:      "Not yet expired",
			expiresAt: &future,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute")
			rule.ExpiresAt = tt.expiresAt

			got := rule.IsExpired()
			if got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestACLRuleClone(t *testing.T) {
	original := NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute")
	original.Conditions["key1"] = "value1"
	original.Metadata["meta1"] = "val1"
	future := time.Now().Add(1 * time.Hour)
	original.ExpiresAt = &future

	clone := original.Clone()

	// Test that clone has same values
	if clone.ID != original.ID {
		t.Error("Clone ID mismatch")
	}
	if clone.Priority != original.Priority {
		t.Error("Clone Priority mismatch")
	}
	if clone.Effect != original.Effect {
		t.Error("Clone Effect mismatch")
	}

	// Test that modifications to clone don't affect original
	clone.Conditions["key2"] = "value2"
	if _, exists := original.Conditions["key2"]; exists {
		t.Error("Modifying clone affected original conditions")
	}

	clone.Metadata["meta2"] = "val2"
	if _, exists := original.Metadata["meta2"]; exists {
		t.Error("Modifying clone affected original metadata")
	}
}

func TestNewACLContext(t *testing.T) {
	ctx := NewACLContext("user:123", "saga:order-456", "execute")

	if ctx.Principal != "user:123" {
		t.Errorf("Expected Principal 'user:123', got %s", ctx.Principal)
	}
	if ctx.Resource != "saga:order-456" {
		t.Errorf("Expected Resource 'saga:order-456', got %s", ctx.Resource)
	}
	if ctx.Action != "execute" {
		t.Errorf("Expected Action 'execute', got %s", ctx.Action)
	}
	if ctx.Attributes == nil {
		t.Error("Expected Attributes to be initialized")
	}
}

func TestACLContextValidate(t *testing.T) {
	tests := []struct {
		name      string
		context   *ACLContext
		wantError bool
	}{
		{
			name:      "Valid context",
			context:   NewACLContext("user:123", "saga:order-456", "execute"),
			wantError: false,
		},
		{
			name:      "Empty principal",
			context:   NewACLContext("", "saga:order-456", "execute"),
			wantError: true,
		},
		{
			name:      "Empty resource",
			context:   NewACLContext("user:123", "", "execute"),
			wantError: true,
		},
		{
			name:      "Empty action",
			context:   NewACLContext("user:123", "saga:order-456", ""),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.context.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestACLContextWithAttribute(t *testing.T) {
	ctx := NewACLContext("user:123", "saga:order-456", "execute")
	ctx.WithAttribute("ip_address", "192.168.1.1")

	if ctx.Attributes["ip_address"] != "192.168.1.1" {
		t.Error("WithAttribute failed to set attribute")
	}
}

func TestACLContextWithAttributes(t *testing.T) {
	ctx := NewACLContext("user:123", "saga:order-456", "execute")
	attrs := map[string]string{
		"ip_address": "192.168.1.1",
		"time":       "business_hours",
	}
	ctx.WithAttributes(attrs)

	if ctx.Attributes["ip_address"] != "192.168.1.1" {
		t.Error("WithAttributes failed to set ip_address")
	}
	if ctx.Attributes["time"] != "business_hours" {
		t.Error("WithAttributes failed to set time")
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		value   string
		want    bool
	}{
		{"Exact match", "user:123", "user:123", true},
		{"Wildcard match all", "*", "user:123", true},
		{"Prefix wildcard match", "user:*", "user:123", true},
		{"Prefix wildcard no match", "role:*", "user:123", false},
		{"Suffix wildcard match", "*:123", "user:123", true},
		{"Suffix wildcard no match", "*:456", "user:123", false},
		{"No match", "user:456", "user:123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesPattern(tt.pattern, tt.value)
			if got != tt.want {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}

func TestNewACLManager(t *testing.T) {
	tests := []struct {
		name   string
		config *ACLManagerConfig
	}{
		{
			name:   "Default config",
			config: nil,
		},
		{
			name: "Custom config",
			config: &ACLManagerConfig{
				DefaultEffect: ACLEffectAllow,
				EnableMetrics: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewACLManager(tt.config)
			if manager == nil {
				t.Fatal("NewACLManager returned nil")
			}
			if manager.rules == nil {
				t.Error("Rules map not initialized")
			}
			if manager.sortedRules == nil {
				t.Error("Sorted rules not initialized")
			}
		})
	}
}

func TestACLManagerAddRule(t *testing.T) {
	manager := NewACLManager(nil)

	tests := []struct {
		name      string
		rule      *ACLRule
		wantError bool
	}{
		{
			name:      "Add valid rule",
			rule:      NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute"),
			wantError: false,
		},
		{
			name:      "Add nil rule",
			rule:      nil,
			wantError: true,
		},
		{
			name: "Add invalid rule",
			rule: &ACLRule{
				ID:        "",
				Priority:  100,
				Effect:    ACLEffectAllow,
				Principal: "user:123",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.AddRule(tt.rule)
			if (err != nil) != tt.wantError {
				t.Errorf("AddRule() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && tt.rule != nil {
				// Verify rule was added
				retrieved, err := manager.GetRule(tt.rule.ID)
				if err != nil {
					t.Errorf("GetRule() after AddRule() failed: %v", err)
				}
				if retrieved.ID != tt.rule.ID {
					t.Error("Retrieved rule doesn't match added rule")
				}
			}
		})
	}
}

func TestACLManagerAddRuleDuplicate(t *testing.T) {
	manager := NewACLManager(nil)
	rule := NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute")

	// Add rule first time - should succeed
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("First AddRule() failed: %v", err)
	}

	// Add same rule again - should fail
	err := manager.AddRule(rule)
	if err == nil {
		t.Error("Expected error when adding duplicate rule, got nil")
	}
	if !errors.Is(err, ErrACLRuleAlreadyExists) {
		t.Errorf("Expected ErrACLRuleAlreadyExists, got %v", err)
	}
}

func TestACLManagerGetRule(t *testing.T) {
	manager := NewACLManager(nil)
	rule := NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute")
	_ = manager.AddRule(rule)

	tests := []struct {
		name      string
		ruleID    string
		wantError bool
	}{
		{
			name:      "Get existing rule",
			ruleID:    "rule-1",
			wantError: false,
		},
		{
			name:      "Get non-existent rule",
			ruleID:    "rule-999",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := manager.GetRule(tt.ruleID)
			if (err != nil) != tt.wantError {
				t.Errorf("GetRule() error = %v, wantError %v", err, tt.wantError)
			}
			if !tt.wantError && retrieved.ID != tt.ruleID {
				t.Errorf("GetRule() returned rule with ID %s, want %s", retrieved.ID, tt.ruleID)
			}
		})
	}
}

func TestACLManagerUpdateRule(t *testing.T) {
	manager := NewACLManager(nil)
	rule := NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute")
	_ = manager.AddRule(rule)

	// Update the rule
	rule.Priority = 200
	err := manager.UpdateRule(rule)
	if err != nil {
		t.Fatalf("UpdateRule() failed: %v", err)
	}

	// Verify the update
	retrieved, _ := manager.GetRule("rule-1")
	if retrieved.Priority != 200 {
		t.Errorf("Expected priority 200, got %d", retrieved.Priority)
	}
}

func TestACLManagerUpdateRuleNonExistent(t *testing.T) {
	manager := NewACLManager(nil)
	rule := NewACLRule("rule-999", 100, ACLEffectAllow, "user:123", "saga:*", "execute")

	err := manager.UpdateRule(rule)
	if err == nil {
		t.Error("Expected error when updating non-existent rule, got nil")
	}
	if !errors.Is(err, ErrACLRuleNotFound) {
		t.Errorf("Expected ErrACLRuleNotFound, got %v", err)
	}
}

func TestACLManagerDeleteRule(t *testing.T) {
	manager := NewACLManager(nil)
	rule := NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute")
	_ = manager.AddRule(rule)

	// Delete the rule
	err := manager.DeleteRule("rule-1")
	if err != nil {
		t.Fatalf("DeleteRule() failed: %v", err)
	}

	// Verify deletion
	_, err = manager.GetRule("rule-1")
	if err == nil {
		t.Error("Expected error when getting deleted rule, got nil")
	}
}

func TestACLManagerListRules(t *testing.T) {
	manager := NewACLManager(nil)

	// Add multiple rules
	_ = manager.AddRule(NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute"))
	_ = manager.AddRule(NewACLRule("rule-2", 200, ACLEffectDeny, "user:456", "saga:*", "cancel"))

	rules := manager.ListRules()
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}
}

func TestACLManagerListRulesByPrincipal(t *testing.T) {
	manager := NewACLManager(nil)

	_ = manager.AddRule(NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute"))
	_ = manager.AddRule(NewACLRule("rule-2", 200, ACLEffectAllow, "user:456", "saga:*", "execute"))
	_ = manager.AddRule(NewACLRule("rule-3", 150, ACLEffectAllow, "user:*", "saga:*", "read"))

	rules := manager.ListRulesByPrincipal("user:123")
	// Should match rule-1 (exact) and rule-3 (wildcard)
	if len(rules) < 2 {
		t.Errorf("Expected at least 2 matching rules, got %d", len(rules))
	}
}

func TestACLManagerListRulesByResource(t *testing.T) {
	manager := NewACLManager(nil)

	_ = manager.AddRule(NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:order-*", "execute"))
	_ = manager.AddRule(NewACLRule("rule-2", 200, ACLEffectAllow, "user:456", "saga:payment-*", "execute"))
	_ = manager.AddRule(NewACLRule("rule-3", 150, ACLEffectAllow, "user:123", "saga:*", "read"))

	rules := manager.ListRulesByResource("saga:order-123")
	// Should match rule-1 (prefix) and rule-3 (wildcard)
	if len(rules) < 2 {
		t.Errorf("Expected at least 2 matching rules, got %d", len(rules))
	}
}

func TestACLManagerEvaluate(t *testing.T) {
	manager := NewACLManager(&ACLManagerConfig{
		DefaultEffect: ACLEffectDeny,
	})

	// Add rules with different priorities
	_ = manager.AddRule(NewACLRule("rule-deny", 100, ACLEffectDeny, "user:123", "saga:*", "delete"))
	_ = manager.AddRule(NewACLRule("rule-allow", 200, ACLEffectAllow, "user:123", "saga:*", "execute"))

	tests := []struct {
		name        string
		context     *ACLContext
		wantAllowed bool
	}{
		{
			name:        "Matched allow rule",
			context:     NewACLContext("user:123", "saga:order-456", "execute"),
			wantAllowed: true,
		},
		{
			name:        "Matched deny rule",
			context:     NewACLContext("user:123", "saga:order-456", "delete"),
			wantAllowed: false,
		},
		{
			name:        "No match, use default (deny)",
			context:     NewACLContext("user:999", "saga:order-456", "execute"),
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := manager.Evaluate(context.Background(), tt.context)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if decision.Allowed != tt.wantAllowed {
				t.Errorf("Evaluate() allowed = %v, want %v", decision.Allowed, tt.wantAllowed)
			}
		})
	}
}

func TestACLManagerEvaluateRulePriority(t *testing.T) {
	manager := NewACLManager(&ACLManagerConfig{
		DefaultEffect: ACLEffectDeny,
	})

	// Add rules with different priorities - higher priority should win
	_ = manager.AddRule(NewACLRule("rule-low", 100, ACLEffectAllow, "user:123", "saga:*", "execute"))
	_ = manager.AddRule(NewACLRule("rule-high", 900, ACLEffectDeny, "user:123", "saga:*", "execute"))

	ctx := NewACLContext("user:123", "saga:order-456", "execute")
	decision, err := manager.Evaluate(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	// High priority deny rule should win
	if decision.Allowed {
		t.Error("Expected deny (high priority), got allow")
	}
	if decision.MatchedRule.ID != "rule-high" {
		t.Errorf("Expected matched rule 'rule-high', got '%s'", decision.MatchedRule.ID)
	}
}

func TestACLManagerCheckAccess(t *testing.T) {
	manager := NewACLManager(nil)
	_ = manager.AddRule(NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute"))

	tests := []struct {
		name      string
		context   *ACLContext
		wantError bool
	}{
		{
			name:      "Access allowed",
			context:   NewACLContext("user:123", "saga:order-456", "execute"),
			wantError: false,
		},
		{
			name:      "Access denied",
			context:   NewACLContext("user:999", "saga:order-456", "execute"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.CheckAccess(context.Background(), tt.context)
			if (err != nil) != tt.wantError {
				t.Errorf("CheckAccess() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && !errors.Is(err, ErrAccessDenied) {
				t.Errorf("Expected ErrAccessDenied, got %v", err)
			}
		})
	}
}

func TestACLManagerCheckAccessForSaga(t *testing.T) {
	manager := NewACLManager(nil)
	_ = manager.AddRule(NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:order-456", "execute"))

	err := manager.CheckAccessForSaga(context.Background(), "user:123", "order-456", "execute")
	if err != nil {
		t.Errorf("CheckAccessForSaga() unexpected error: %v", err)
	}

	err = manager.CheckAccessForSaga(context.Background(), "user:999", "order-456", "execute")
	if err == nil {
		t.Error("Expected error for unauthorized access, got nil")
	}
}

func TestACLManagerCheckAccessWithRoles(t *testing.T) {
	rbacManager := NewRBACManager(nil)
	_ = rbacManager.AssignRole("user:123", "operator")

	manager := NewACLManager(&ACLManagerConfig{
		RBACManager: rbacManager,
	})

	// Add ACL rule for role
	_ = manager.AddRule(NewACLRule("rule-1", 100, ACLEffectAllow, "role:operator", "saga:*", "execute"))

	// User should get access through their role
	err := manager.CheckAccessWithRoles(context.Background(), "user:123", "saga:order-456", "execute")
	if err != nil {
		t.Errorf("CheckAccessWithRoles() unexpected error: %v", err)
	}

	// User without role should not get access
	err = manager.CheckAccessWithRoles(context.Background(), "user:999", "saga:order-456", "execute")
	if err == nil {
		t.Error("Expected error for user without role, got nil")
	}
}

func TestACLManagerCleanupExpiredRules(t *testing.T) {
	manager := NewACLManager(nil)

	// Add rule that expires in the past
	past := time.Now().Add(-1 * time.Hour)
	expiredRule := NewACLRule("rule-expired", 100, ACLEffectAllow, "user:123", "saga:*", "execute")
	expiredRule.ExpiresAt = &past
	_ = manager.AddRule(expiredRule)

	// Add rule that never expires
	_ = manager.AddRule(NewACLRule("rule-active", 100, ACLEffectAllow, "user:456", "saga:*", "execute"))

	// Cleanup
	removed := manager.CleanupExpiredRules()
	if removed != 1 {
		t.Errorf("Expected 1 rule removed, got %d", removed)
	}

	// Verify expired rule is gone
	_, err := manager.GetRule("rule-expired")
	if err == nil {
		t.Error("Expected error when getting expired rule, got nil")
	}

	// Verify active rule still exists
	_, err = manager.GetRule("rule-active")
	if err != nil {
		t.Errorf("Active rule should still exist: %v", err)
	}
}

func TestACLManagerMetrics(t *testing.T) {
	manager := NewACLManager(&ACLManagerConfig{
		EnableMetrics: true,
	})

	_ = manager.AddRule(NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute"))

	// Perform some evaluations
	ctx1 := NewACLContext("user:123", "saga:order-1", "execute")
	_, _ = manager.Evaluate(context.Background(), ctx1)

	ctx2 := NewACLContext("user:999", "saga:order-2", "execute")
	_, _ = manager.Evaluate(context.Background(), ctx2)

	metrics := manager.GetMetrics()
	enabled, ok := metrics["enabled"].(bool)
	if !ok || !enabled {
		t.Error("Metrics should be enabled")
	}

	totalEval, ok := metrics["total_evaluations"].(int64)
	if !ok || totalEval != 2 {
		t.Errorf("Expected 2 total evaluations, got %v", totalEval)
	}

	// Reset metrics
	manager.ResetMetrics()
	metrics = manager.GetMetrics()
	totalEval, ok = metrics["total_evaluations"].(int64)
	if !ok || totalEval != 0 {
		t.Errorf("Expected 0 evaluations after reset, got %v", totalEval)
	}
}

func TestACLManagerExportImportRules(t *testing.T) {
	manager := NewACLManager(nil)

	// Add some rules
	_ = manager.AddRule(NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute"))
	_ = manager.AddRule(NewACLRule("rule-2", 200, ACLEffectDeny, "user:456", "saga:*", "delete"))

	// Export rules
	exported := manager.ExportRules()
	if len(exported) != 2 {
		t.Errorf("Expected 2 exported rules, got %d", len(exported))
	}

	// Create new manager and import
	newManager := NewACLManager(nil)
	err := newManager.ImportRules(exported)
	if err != nil {
		t.Fatalf("ImportRules() failed: %v", err)
	}

	// Verify imported rules
	imported := newManager.ListRules()
	if len(imported) != 2 {
		t.Errorf("Expected 2 imported rules, got %d", len(imported))
	}
}

func TestACLManagerClear(t *testing.T) {
	manager := NewACLManager(nil)

	_ = manager.AddRule(NewACLRule("rule-1", 100, ACLEffectAllow, "user:123", "saga:*", "execute"))
	_ = manager.AddRule(NewACLRule("rule-2", 200, ACLEffectAllow, "user:456", "saga:*", "execute"))

	manager.Clear()

	rules := manager.ListRules()
	if len(rules) != 0 {
		t.Errorf("Expected 0 rules after Clear(), got %d", len(rules))
	}
}

func TestACLManagerConcurrentAccess(t *testing.T) {
	manager := NewACLManager(nil)
	_ = manager.AddRule(NewACLRule("rule-1", 100, ACLEffectAllow, "user:*", "saga:*", "execute"))

	// Run concurrent evaluations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			ctx := NewACLContext("user:123", "saga:order-456", "execute")
			_, err := manager.Evaluate(context.Background(), ctx)
			if err != nil {
				t.Errorf("Concurrent Evaluate() failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkACLManagerEvaluate(b *testing.B) {
	manager := NewACLManager(nil)

	// Add rules with varying priorities
	for i := 0; i < 100; i++ {
		rule := NewACLRule(
			fmt.Sprintf("rule-%d", i),
			i*10,
			ACLEffectAllow,
			fmt.Sprintf("user:%d", i),
			"saga:*",
			"execute",
		)
		_ = manager.AddRule(rule)
	}

	ctx := NewACLContext("user:50", "saga:order-456", "execute")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.Evaluate(context.Background(), ctx)
	}
}

func BenchmarkACLRuleMatches(b *testing.B) {
	rule := NewACLRule("rule-1", 100, ACLEffectAllow, "user:*", "saga:order-*", "*")
	ctx := NewACLContext("user:123", "saga:order-456", "execute")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rule.Matches(ctx)
	}
}

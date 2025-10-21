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

// Package coordinator provides mode selection logic for hybrid Saga coordination.
package coordinator

import (
	"sort"

	"github.com/innovationmech/swit/pkg/saga"
)

// CoordinationMode represents the coordination mode for Saga execution.
type CoordinationMode string

const (
	// ModeOrchestration uses centralized orchestration-based coordination.
	ModeOrchestration CoordinationMode = "orchestration"

	// ModeChoreography uses decentralized choreography-based coordination.
	ModeChoreography CoordinationMode = "choreography"

	// ModeAuto automatically selects the best coordination mode.
	ModeAuto CoordinationMode = "auto"
)

// ModeSelector defines the interface for selecting coordination mode.
// Implementations can provide different strategies for mode selection.
type ModeSelector interface {
	// SelectMode selects the appropriate coordination mode for a Saga definition.
	// Returns the selected mode based on the definition's characteristics.
	SelectMode(def saga.SagaDefinition) CoordinationMode

	// GetReason returns a human-readable explanation for the mode selection.
	GetReason() string
}

// ModeSelectionRule defines a rule for selecting a coordination mode.
// Rules are evaluated in priority order to determine the best mode.
type ModeSelectionRule interface {
	// Matches determines if this rule applies to the given Saga definition.
	Matches(def saga.SagaDefinition) bool

	// RecommendedMode returns the mode recommended by this rule.
	RecommendedMode() CoordinationMode

	// Priority returns the priority of this rule (higher values execute first).
	Priority() int

	// GetReason returns a human-readable explanation for this rule.
	GetReason() string
}

// SmartModeSelector implements intelligent mode selection using a set of rules.
// It evaluates rules in priority order and selects the mode recommended by
// the first matching rule.
type SmartModeSelector struct {
	rules       []ModeSelectionRule
	lastReason  string
	defaultMode CoordinationMode
	sortedOnce  bool
}

// NewSmartModeSelector creates a new smart mode selector with the given rules.
// Rules are evaluated in priority order (higher priority first).
func NewSmartModeSelector(rules []ModeSelectionRule, defaultMode CoordinationMode) *SmartModeSelector {
	if defaultMode == "" {
		defaultMode = ModeOrchestration
	}

	return &SmartModeSelector{
		rules:       rules,
		defaultMode: defaultMode,
	}
}

// SelectMode selects the coordination mode by evaluating rules in priority order.
func (s *SmartModeSelector) SelectMode(def saga.SagaDefinition) CoordinationMode {
	// Sort rules by priority on first use
	if !s.sortedOnce {
		s.sortRulesByPriority()
		s.sortedOnce = true
	}

	// Evaluate rules in priority order
	for _, rule := range s.rules {
		if rule.Matches(def) {
			s.lastReason = rule.GetReason()
			return rule.RecommendedMode()
		}
	}

	// No rule matched, use default mode
	s.lastReason = "no matching rule found, using default mode"
	return s.defaultMode
}

// GetReason returns the explanation for the last mode selection.
func (s *SmartModeSelector) GetReason() string {
	return s.lastReason
}

// sortRulesByPriority sorts rules by priority (higher priority first).
func (s *SmartModeSelector) sortRulesByPriority() {
	sort.Slice(s.rules, func(i, j int) bool {
		return s.rules[i].Priority() > s.rules[j].Priority()
	})
}

// AddRule adds a new rule to the selector.
// The rules will be re-sorted on the next SelectMode call.
func (s *SmartModeSelector) AddRule(rule ModeSelectionRule) {
	s.rules = append(s.rules, rule)
	s.sortedOnce = false
}

// RemoveRule removes a rule by comparing with the given rule.
func (s *SmartModeSelector) RemoveRule(rule ModeSelectionRule) bool {
	for i, r := range s.rules {
		if r == rule {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			return true
		}
	}
	return false
}

// GetRules returns a copy of the current rules.
func (s *SmartModeSelector) GetRules() []ModeSelectionRule {
	rulesCopy := make([]ModeSelectionRule, len(s.rules))
	copy(rulesCopy, s.rules)
	return rulesCopy
}

// ==========================
// Built-in Selection Rules
// ==========================

// SimpleLinearRule recommends orchestration for simple linear workflows.
// It matches Sagas with few steps and no complex dependencies.
type SimpleLinearRule struct {
	maxSteps int
}

// NewSimpleLinearRule creates a new simple linear rule.
// Sagas with steps <= maxSteps are considered simple and linear.
func NewSimpleLinearRule(maxSteps int) *SimpleLinearRule {
	if maxSteps <= 0 {
		maxSteps = 5
	}
	return &SimpleLinearRule{maxSteps: maxSteps}
}

// Matches determines if the Saga is simple and linear.
func (r *SimpleLinearRule) Matches(def saga.SagaDefinition) bool {
	steps := def.GetSteps()
	if len(steps) == 0 || len(steps) > r.maxSteps {
		return false
	}

	// Check if workflow is linear (no parallel execution hints in metadata)
	metadata := def.GetMetadata()
	if metadata != nil {
		if parallel, ok := metadata["parallel"].(bool); ok && parallel {
			return false
		}
	}

	return true
}

// RecommendedMode returns orchestration mode for simple linear workflows.
func (r *SimpleLinearRule) RecommendedMode() CoordinationMode {
	return ModeOrchestration
}

// Priority returns the priority of this rule.
func (r *SimpleLinearRule) Priority() int {
	return 100
}

// GetReason returns the explanation for this rule.
func (r *SimpleLinearRule) GetReason() string {
	return "simple linear workflow with few steps - orchestration is more efficient"
}

// ComplexParallelRule recommends choreography for complex parallel workflows.
// It matches Sagas with many steps or parallel execution requirements.
type ComplexParallelRule struct {
	minSteps int
}

// NewComplexParallelRule creates a new complex parallel rule.
// Sagas with steps > minSteps are considered complex.
func NewComplexParallelRule(minSteps int) *ComplexParallelRule {
	if minSteps <= 0 {
		minSteps = 5
	}
	return &ComplexParallelRule{minSteps: minSteps}
}

// Matches determines if the Saga is complex or requires parallel execution.
func (r *ComplexParallelRule) Matches(def saga.SagaDefinition) bool {
	steps := def.GetSteps()
	if len(steps) > r.minSteps {
		return true
	}

	// Check for parallel execution hints in metadata
	metadata := def.GetMetadata()
	if metadata != nil {
		if parallel, ok := metadata["parallel"].(bool); ok && parallel {
			return true
		}
	}

	return false
}

// RecommendedMode returns choreography mode for complex workflows.
func (r *ComplexParallelRule) RecommendedMode() CoordinationMode {
	return ModeChoreography
}

// Priority returns the priority of this rule.
func (r *ComplexParallelRule) Priority() int {
	return 90
}

// GetReason returns the explanation for this rule.
func (r *ComplexParallelRule) GetReason() string {
	return "complex workflow with many steps or parallel execution - choreography scales better"
}

// CrossDomainRule recommends choreography for cross-domain workflows.
// It matches Sagas that span multiple service domains or teams.
type CrossDomainRule struct{}

// NewCrossDomainRule creates a new cross-domain rule.
func NewCrossDomainRule() *CrossDomainRule {
	return &CrossDomainRule{}
}

// Matches determines if the Saga spans multiple domains.
func (r *CrossDomainRule) Matches(def saga.SagaDefinition) bool {
	metadata := def.GetMetadata()
	if metadata == nil {
		return false
	}

	// Check for cross-domain flag in metadata
	if crossDomain, ok := metadata["cross_domain"].(bool); ok && crossDomain {
		return true
	}

	// Check for multiple service domains
	if domains, ok := metadata["domains"].([]string); ok && len(domains) > 1 {
		return true
	}

	return false
}

// RecommendedMode returns choreography mode for cross-domain workflows.
func (r *CrossDomainRule) RecommendedMode() CoordinationMode {
	return ModeChoreography
}

// Priority returns the priority of this rule.
func (r *CrossDomainRule) Priority() int {
	return 95
}

// GetReason returns the explanation for this rule.
func (r *CrossDomainRule) GetReason() string {
	return "workflow spans multiple service domains - choreography reduces coupling"
}

// CentralizedControlRule recommends orchestration when centralized control is needed.
// It matches Sagas that require strong consistency or centralized decision making.
type CentralizedControlRule struct{}

// NewCentralizedControlRule creates a new centralized control rule.
func NewCentralizedControlRule() *CentralizedControlRule {
	return &CentralizedControlRule{}
}

// Matches determines if the Saga requires centralized control.
func (r *CentralizedControlRule) Matches(def saga.SagaDefinition) bool {
	metadata := def.GetMetadata()
	if metadata == nil {
		return false
	}

	// Check for centralized control flag
	if centralized, ok := metadata["centralized"].(bool); ok && centralized {
		return true
	}

	// Check for strong consistency requirement
	if consistency, ok := metadata["consistency"].(string); ok && consistency == "strong" {
		return true
	}

	return false
}

// RecommendedMode returns orchestration mode for centralized control.
func (r *CentralizedControlRule) RecommendedMode() CoordinationMode {
	return ModeOrchestration
}

// Priority returns the priority of this rule.
func (r *CentralizedControlRule) Priority() int {
	return 110
}

// GetReason returns the explanation for this rule.
func (r *CentralizedControlRule) GetReason() string {
	return "workflow requires centralized control or strong consistency - orchestration provides better guarantees"
}

// ManualModeSelector implements a simple selector that always returns a fixed mode.
// Useful for testing or when manual mode selection is desired.
type ManualModeSelector struct {
	mode   CoordinationMode
	reason string
}

// NewManualModeSelector creates a new manual mode selector.
func NewManualModeSelector(mode CoordinationMode, reason string) *ManualModeSelector {
	if reason == "" {
		reason = "manually configured mode"
	}
	return &ManualModeSelector{
		mode:   mode,
		reason: reason,
	}
}

// SelectMode returns the fixed coordination mode.
func (m *ManualModeSelector) SelectMode(def saga.SagaDefinition) CoordinationMode {
	return m.mode
}

// GetReason returns the explanation for the mode selection.
func (m *ManualModeSelector) GetReason() string {
	return m.reason
}

// SetMode changes the fixed mode.
func (m *ManualModeSelector) SetMode(mode CoordinationMode, reason string) {
	m.mode = mode
	if reason != "" {
		m.reason = reason
	}
}

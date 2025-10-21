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

package coordinator

import (
	"testing"

	"github.com/innovationmech/swit/pkg/saga"
)

// TestSimpleLinearRule tests the SimpleLinearRule mode selection.
func TestSimpleLinearRule(t *testing.T) {
	tests := []struct {
		name       string
		definition saga.SagaDefinition
		wantMatch  bool
		wantMode   CoordinationMode
	}{
		{
			name: "matches simple linear workflow",
			definition: &mockSagaDefinition{
				id:   "simple-saga",
				name: "Simple Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
					&mockSagaStep{id: "step2"},
					&mockSagaStep{id: "step3"},
				},
				metadata: nil,
			},
			wantMatch: true,
			wantMode:  ModeOrchestration,
		},
		{
			name: "does not match too many steps",
			definition: &mockSagaDefinition{
				id:   "complex-saga",
				name: "Complex Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
					&mockSagaStep{id: "step2"},
					&mockSagaStep{id: "step3"},
					&mockSagaStep{id: "step4"},
					&mockSagaStep{id: "step5"},
					&mockSagaStep{id: "step6"},
				},
				metadata: nil,
			},
			wantMatch: false,
			wantMode:  ModeOrchestration,
		},
		{
			name: "does not match parallel workflow",
			definition: &mockSagaDefinition{
				id:   "parallel-saga",
				name: "Parallel Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
					&mockSagaStep{id: "step2"},
				},
				metadata: map[string]interface{}{
					"parallel": true,
				},
			},
			wantMatch: false,
			wantMode:  ModeOrchestration,
		},
		{
			name: "does not match empty steps",
			definition: &mockSagaDefinition{
				id:       "empty-saga",
				name:     "Empty Saga",
				steps:    []saga.SagaStep{},
				metadata: nil,
			},
			wantMatch: false,
			wantMode:  ModeOrchestration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewSimpleLinearRule(5)
			if got := rule.Matches(tt.definition); got != tt.wantMatch {
				t.Errorf("SimpleLinearRule.Matches() = %v, want %v", got, tt.wantMatch)
			}
			if got := rule.RecommendedMode(); got != tt.wantMode {
				t.Errorf("SimpleLinearRule.RecommendedMode() = %v, want %v", got, tt.wantMode)
			}
			if rule.Priority() <= 0 {
				t.Error("SimpleLinearRule.Priority() should be positive")
			}
			if rule.GetReason() == "" {
				t.Error("SimpleLinearRule.GetReason() should not be empty")
			}
		})
	}
}

// TestComplexParallelRule tests the ComplexParallelRule mode selection.
func TestComplexParallelRule(t *testing.T) {
	tests := []struct {
		name       string
		definition saga.SagaDefinition
		wantMatch  bool
		wantMode   CoordinationMode
	}{
		{
			name: "matches complex workflow with many steps",
			definition: &mockSagaDefinition{
				id:   "complex-saga",
				name: "Complex Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
					&mockSagaStep{id: "step2"},
					&mockSagaStep{id: "step3"},
					&mockSagaStep{id: "step4"},
					&mockSagaStep{id: "step5"},
					&mockSagaStep{id: "step6"},
				},
				metadata: nil,
			},
			wantMatch: true,
			wantMode:  ModeChoreography,
		},
		{
			name: "matches parallel workflow",
			definition: &mockSagaDefinition{
				id:   "parallel-saga",
				name: "Parallel Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
					&mockSagaStep{id: "step2"},
				},
				metadata: map[string]interface{}{
					"parallel": true,
				},
			},
			wantMatch: true,
			wantMode:  ModeChoreography,
		},
		{
			name: "does not match simple workflow",
			definition: &mockSagaDefinition{
				id:   "simple-saga",
				name: "Simple Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
					&mockSagaStep{id: "step2"},
				},
				metadata: nil,
			},
			wantMatch: false,
			wantMode:  ModeChoreography,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewComplexParallelRule(5)
			if got := rule.Matches(tt.definition); got != tt.wantMatch {
				t.Errorf("ComplexParallelRule.Matches() = %v, want %v", got, tt.wantMatch)
			}
			if got := rule.RecommendedMode(); got != tt.wantMode {
				t.Errorf("ComplexParallelRule.RecommendedMode() = %v, want %v", got, tt.wantMode)
			}
		})
	}
}

// TestCrossDomainRule tests the CrossDomainRule mode selection.
func TestCrossDomainRule(t *testing.T) {
	tests := []struct {
		name       string
		definition saga.SagaDefinition
		wantMatch  bool
		wantMode   CoordinationMode
	}{
		{
			name: "matches cross-domain workflow",
			definition: &mockSagaDefinition{
				id:   "cross-domain-saga",
				name: "Cross Domain Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
					&mockSagaStep{id: "step2"},
				},
				metadata: map[string]interface{}{
					"cross_domain": true,
				},
			},
			wantMatch: true,
			wantMode:  ModeChoreography,
		},
		{
			name: "matches multiple domains",
			definition: &mockSagaDefinition{
				id:   "multi-domain-saga",
				name: "Multi Domain Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
					&mockSagaStep{id: "step2"},
				},
				metadata: map[string]interface{}{
					"domains": []string{"domain1", "domain2"},
				},
			},
			wantMatch: true,
			wantMode:  ModeChoreography,
		},
		{
			name: "does not match single domain",
			definition: &mockSagaDefinition{
				id:   "single-domain-saga",
				name: "Single Domain Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
				},
				metadata: map[string]interface{}{
					"domains": []string{"domain1"},
				},
			},
			wantMatch: false,
			wantMode:  ModeChoreography,
		},
		{
			name: "does not match no metadata",
			definition: &mockSagaDefinition{
				id:       "no-metadata-saga",
				name:     "No Metadata Saga",
				steps:    []saga.SagaStep{&mockSagaStep{id: "step1"}},
				metadata: nil,
			},
			wantMatch: false,
			wantMode:  ModeChoreography,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewCrossDomainRule()
			if got := rule.Matches(tt.definition); got != tt.wantMatch {
				t.Errorf("CrossDomainRule.Matches() = %v, want %v", got, tt.wantMatch)
			}
			if got := rule.RecommendedMode(); got != tt.wantMode {
				t.Errorf("CrossDomainRule.RecommendedMode() = %v, want %v", got, tt.wantMode)
			}
		})
	}
}

// TestCentralizedControlRule tests the CentralizedControlRule mode selection.
func TestCentralizedControlRule(t *testing.T) {
	tests := []struct {
		name       string
		definition saga.SagaDefinition
		wantMatch  bool
		wantMode   CoordinationMode
	}{
		{
			name: "matches centralized control",
			definition: &mockSagaDefinition{
				id:   "centralized-saga",
				name: "Centralized Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
				},
				metadata: map[string]interface{}{
					"centralized": true,
				},
			},
			wantMatch: true,
			wantMode:  ModeOrchestration,
		},
		{
			name: "matches strong consistency",
			definition: &mockSagaDefinition{
				id:   "strong-consistency-saga",
				name: "Strong Consistency Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
				},
				metadata: map[string]interface{}{
					"consistency": "strong",
				},
			},
			wantMatch: true,
			wantMode:  ModeOrchestration,
		},
		{
			name: "does not match eventual consistency",
			definition: &mockSagaDefinition{
				id:   "eventual-consistency-saga",
				name: "Eventual Consistency Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
				},
				metadata: map[string]interface{}{
					"consistency": "eventual",
				},
			},
			wantMatch: false,
			wantMode:  ModeOrchestration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewCentralizedControlRule()
			if got := rule.Matches(tt.definition); got != tt.wantMatch {
				t.Errorf("CentralizedControlRule.Matches() = %v, want %v", got, tt.wantMatch)
			}
			if got := rule.RecommendedMode(); got != tt.wantMode {
				t.Errorf("CentralizedControlRule.RecommendedMode() = %v, want %v", got, tt.wantMode)
			}
		})
	}
}

// TestSmartModeSelector tests the SmartModeSelector.
func TestSmartModeSelector(t *testing.T) {
	tests := []struct {
		name       string
		rules      []ModeSelectionRule
		definition saga.SagaDefinition
		wantMode   CoordinationMode
	}{
		{
			name: "selects orchestration for simple workflow",
			rules: []ModeSelectionRule{
				NewSimpleLinearRule(5),
				NewComplexParallelRule(5),
			},
			definition: &mockSagaDefinition{
				id:   "simple-saga",
				name: "Simple Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
					&mockSagaStep{id: "step2"},
				},
			},
			wantMode: ModeOrchestration,
		},
		{
			name: "selects choreography for complex workflow",
			rules: []ModeSelectionRule{
				NewSimpleLinearRule(5),
				NewComplexParallelRule(5),
			},
			definition: &mockSagaDefinition{
				id:   "complex-saga",
				name: "Complex Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
					&mockSagaStep{id: "step2"},
					&mockSagaStep{id: "step3"},
					&mockSagaStep{id: "step4"},
					&mockSagaStep{id: "step5"},
					&mockSagaStep{id: "step6"},
				},
			},
			wantMode: ModeChoreography,
		},
		{
			name: "respects priority order - centralized over simple",
			rules: []ModeSelectionRule{
				NewSimpleLinearRule(5),
				NewCentralizedControlRule(),
			},
			definition: &mockSagaDefinition{
				id:   "simple-centralized-saga",
				name: "Simple Centralized Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
					&mockSagaStep{id: "step2"},
				},
				metadata: map[string]interface{}{
					"centralized": true,
				},
			},
			wantMode: ModeOrchestration,
		},
		{
			name:  "uses default mode when no rule matches",
			rules: []ModeSelectionRule{},
			definition: &mockSagaDefinition{
				id:   "no-match-saga",
				name: "No Match Saga",
				steps: []saga.SagaStep{
					&mockSagaStep{id: "step1"},
				},
			},
			wantMode: ModeOrchestration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewSmartModeSelector(tt.rules, ModeOrchestration)
			if got := selector.SelectMode(tt.definition); got != tt.wantMode {
				t.Errorf("SmartModeSelector.SelectMode() = %v, want %v", got, tt.wantMode)
			}
			if selector.GetReason() == "" {
				t.Error("SmartModeSelector.GetReason() should not be empty")
			}
		})
	}
}

// TestSmartModeSelectorAddRemoveRule tests rule management.
func TestSmartModeSelectorAddRemoveRule(t *testing.T) {
	selector := NewSmartModeSelector([]ModeSelectionRule{}, ModeOrchestration)

	// Initially no rules
	if len(selector.GetRules()) != 0 {
		t.Errorf("expected 0 rules, got %d", len(selector.GetRules()))
	}

	// Add rule
	rule1 := NewSimpleLinearRule(5)
	selector.AddRule(rule1)
	if len(selector.GetRules()) != 1 {
		t.Errorf("expected 1 rule, got %d", len(selector.GetRules()))
	}

	// Add another rule
	rule2 := NewComplexParallelRule(5)
	selector.AddRule(rule2)
	if len(selector.GetRules()) != 2 {
		t.Errorf("expected 2 rules, got %d", len(selector.GetRules()))
	}

	// Remove rule
	if !selector.RemoveRule(rule1) {
		t.Error("expected RemoveRule to return true")
	}
	if len(selector.GetRules()) != 1 {
		t.Errorf("expected 1 rule after removal, got %d", len(selector.GetRules()))
	}

	// Try to remove non-existent rule
	if selector.RemoveRule(rule1) {
		t.Error("expected RemoveRule to return false for non-existent rule")
	}
}

// TestManualModeSelector tests the ManualModeSelector.
func TestManualModeSelector(t *testing.T) {
	tests := []struct {
		name       string
		mode       CoordinationMode
		definition saga.SagaDefinition
		wantMode   CoordinationMode
	}{
		{
			name: "always returns orchestration mode",
			mode: ModeOrchestration,
			definition: &mockSagaDefinition{
				id:    "test-saga",
				name:  "Test Saga",
				steps: []saga.SagaStep{&mockSagaStep{id: "step1"}},
			},
			wantMode: ModeOrchestration,
		},
		{
			name: "always returns choreography mode",
			mode: ModeChoreography,
			definition: &mockSagaDefinition{
				id:    "test-saga",
				name:  "Test Saga",
				steps: []saga.SagaStep{&mockSagaStep{id: "step1"}},
			},
			wantMode: ModeChoreography,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewManualModeSelector(tt.mode, "test reason")
			if got := selector.SelectMode(tt.definition); got != tt.wantMode {
				t.Errorf("ManualModeSelector.SelectMode() = %v, want %v", got, tt.wantMode)
			}
			if selector.GetReason() == "" {
				t.Error("ManualModeSelector.GetReason() should not be empty")
			}

			// Test SetMode
			newMode := ModeChoreography
			if tt.mode == ModeChoreography {
				newMode = ModeOrchestration
			}
			selector.SetMode(newMode, "new reason")
			if got := selector.SelectMode(tt.definition); got != newMode {
				t.Errorf("ManualModeSelector.SelectMode() after SetMode = %v, want %v", got, newMode)
			}
		})
	}
}

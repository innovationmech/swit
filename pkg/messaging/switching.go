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
	"sort"
	"strings"
)

// FeatureDeltaType captures how a feature changes when switching adapters.
type FeatureDeltaType string

const (
	// FeatureDeltaTypeLoss indicates the target broker does not support a feature provided by the current broker.
	FeatureDeltaTypeLoss FeatureDeltaType = "loss"

	// FeatureDeltaTypeGain indicates the target broker introduces a new capability.
	FeatureDeltaTypeGain FeatureDeltaType = "gain"
)

// BrokerFeatureDelta describes semantic differences for a specific capability during a broker switch.
type BrokerFeatureDelta struct {
	Feature        string           `json:"feature"`
	Delta          FeatureDeltaType `json:"delta"`
	CurrentSupport CapabilityLevel  `json:"current_support"`
	TargetSupport  CapabilityLevel  `json:"target_support"`
	Impact         string           `json:"impact,omitempty"`
	Recommendation string           `json:"recommendation,omitempty"`
}

// BrokerSwitchPlan aggregates compatibility analysis, migration guidance, and operational checklist
// required to safely switch messaging adapters via configuration.
type BrokerSwitchPlan struct {
	CurrentType           BrokerType           `json:"current_type"`
	TargetType            BrokerType           `json:"target_type"`
	CompatibilityScore    int                  `json:"compatibility_score"`
	OverallCompatibility  CompatibilityLevel   `json:"overall_compatibility"`
	MigrationDifficulty   MigrationDifficulty  `json:"migration_difficulty"`
	EstimatedEffort       string               `json:"estimated_effort,omitempty"`
	FeatureDeltas         []BrokerFeatureDelta `json:"feature_deltas,omitempty"`
	Checklist             []string             `json:"checklist,omitempty"`
	Considerations        []string             `json:"considerations,omitempty"`
	Risks                 []MigrationRisk      `json:"risks,omitempty"`
	Steps                 []MigrationStep      `json:"steps,omitempty"`
	ConfigurationIssues   []ConfigurationIssue `json:"configuration_issues,omitempty"`
	ConfigurationMappings map[string]string    `json:"configuration_mappings,omitempty"`
	Recommendations       []string             `json:"recommendations,omitempty"`
	DualWriteRecommended  bool                 `json:"dual_write_recommended"`
}

// PlanBrokerSwitch analyses the semantic impact of switching from one broker adapter to another and
// produces a migration checklist based on feature parity, configuration gaps, and documented guidance.
//
// The function is intentionally read-only: the provided configurations are copied and defaulted so that
// callers can reuse their structs without unintended mutation.
func PlanBrokerSwitch(ctx context.Context, currentConfig, targetConfig *BrokerConfig) (*BrokerSwitchPlan, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if currentConfig == nil {
		return nil, fmt.Errorf("current broker configuration cannot be nil")
	}
	if targetConfig == nil {
		return nil, fmt.Errorf("target broker configuration cannot be nil")
	}
	if !currentConfig.Type.IsValid() {
		return nil, fmt.Errorf("current broker type %q is invalid", currentConfig.Type)
	}
	if !targetConfig.Type.IsValid() {
		return nil, fmt.Errorf("target broker type %q is invalid", targetConfig.Type)
	}

	currentCopy := *currentConfig
	currentCopy.SetDefaults()
	if err := currentCopy.Validate(); err != nil {
		return nil, fmt.Errorf("current broker configuration invalid: %w", err)
	}

	targetCopy := *targetConfig
	targetCopy.SetDefaults()
	if err := targetCopy.Validate(); err != nil {
		return nil, fmt.Errorf("target broker configuration invalid: %w", err)
	}

	checker := NewDefaultBrokerCompatibilityChecker()
	matrix, err := checker.CheckCompatibility(ctx, currentCopy.Type, targetCopy.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to compare broker capabilities: %w", err)
	}

	guidance, err := checker.GetMigrationGuidance(ctx, currentCopy.Type, targetCopy.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to generate migration guidance: %w", err)
	}

	configCompatibility, err := checker.ValidateConfiguration(ctx, &currentCopy, targetCopy.Type)
	if err != nil {
		return nil, fmt.Errorf("configuration compatibility check failed: %w", err)
	}

	featureDeltas := buildFeatureDeltas(currentCopy.Type, targetCopy.Type, matrix)
	recommendations := dedupeStrings(append([]string{}, configCompatibility.Recommendations...))
	recommendations = append(recommendations, extractRecommendations(featureDeltas)...)
	recommendations = dedupeStrings(recommendations)

	plan := &BrokerSwitchPlan{
		CurrentType:           currentCopy.Type,
		TargetType:            targetCopy.Type,
		CompatibilityScore:    matrix.CompatibilityScore,
		OverallCompatibility:  matrix.OverallCompatibility,
		MigrationDifficulty:   guidance.Difficulty,
		EstimatedEffort:       guidance.EstimatedEffort,
		FeatureDeltas:         featureDeltas,
		Risks:                 cloneMigrationRisks(guidance.Risks),
		Steps:                 cloneMigrationSteps(guidance.Steps),
		Considerations:        dedupeStrings(append(cloneStrings(guidance.Considerations), configCompatibility.Recommendations...)),
		ConfigurationIssues:   cloneConfigurationIssues(configCompatibility.Issues),
		ConfigurationMappings: cloneStringMap(configCompatibility.Mappings),
		Recommendations:       recommendations,
	}

	plan.DualWriteRecommended = shouldRecommendDualWrite(plan.OverallCompatibility, featureDeltas)
	plan.Checklist = buildSwitchChecklist(plan, &targetCopy)

	return plan, nil
}

func buildFeatureDeltas(currentType, targetType BrokerType, matrix *CompatibilityMatrix) []BrokerFeatureDelta {
	if matrix == nil || len(matrix.FeatureComparisons) == 0 {
		return nil
	}

	features := make([]string, 0, len(matrix.FeatureComparisons))
	for feature := range matrix.FeatureComparisons {
		features = append(features, feature)
	}
	sort.Strings(features)

	var deltas []BrokerFeatureDelta
	for _, feature := range features {
		comparison := matrix.FeatureComparisons[feature]
		if comparison == nil {
			continue
		}

		currentSupported := comparison.Broker1Support.IsSupported()
		targetSupported := comparison.Broker2Support.IsSupported()

		switch {
		case currentSupported && !targetSupported:
			deltas = append(deltas, BrokerFeatureDelta{
				Feature:        feature,
				Delta:          FeatureDeltaTypeLoss,
				CurrentSupport: comparison.Broker1Support,
				TargetSupport:  comparison.Broker2Support,
				Impact:         fmt.Sprintf("%s is only available on %s", humanizeFeatureName(feature), currentType),
				Recommendation: fallbackRecommendationForFeature(feature),
			})
		case !currentSupported && targetSupported:
			deltas = append(deltas, BrokerFeatureDelta{
				Feature:        feature,
				Delta:          FeatureDeltaTypeGain,
				CurrentSupport: comparison.Broker1Support,
				TargetSupport:  comparison.Broker2Support,
				Impact:         fmt.Sprintf("%s becomes available on %s", humanizeFeatureName(feature), targetType),
				Recommendation: fmt.Sprintf("Plan how to adopt %s capabilities after migration", humanizeFeatureName(feature)),
			})
		}
	}

	return deltas
}

func humanizeFeatureName(feature string) string {
	if feature == "" {
		return feature
	}
	parts := strings.Split(feature, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

var featureLossMitigations = map[string]string{
	"transactions":     "Use a transactional outbox pattern with idempotent consumers to preserve exactly-once semantics.",
	"ordering":         "Partition by key or design handlers to tolerate reordered deliveries; add deduplication where necessary.",
	"partitioning":     "Emulate partitions via routing keys or distinct subjects/queues per shard.",
	"dead_letter":      "Publish failures to a parking topic/queue and process them with a dedicated DLQ worker.",
	"delayed_delivery": "Schedule retries via consumer timers or job schedulers when native delays are unavailable.",
	"priority":         "Route high-priority work to dedicated topics/queues and weight consumers accordingly.",
	"seek":             "Persist offsets externally and re-consume from stored checkpoints to simulate replay.",
	"streaming":        "Adopt batch pull loops or use an external streaming component when continuous streams are required.",
	"consumer_groups":  "Coordinate consumers manually (e.g., via locking or consistent hashing) to share workload.",
}

func fallbackRecommendationForFeature(feature string) string {
	if recommendation, found := featureLossMitigations[feature]; found {
		return recommendation
	}
	return fmt.Sprintf("Document and mitigate the impact of missing %s support", humanizeFeatureName(feature))
}

func extractRecommendations(deltas []BrokerFeatureDelta) []string {
	var recommendations []string
	for _, delta := range deltas {
		if delta.Recommendation == "" {
			continue
		}
		recommendations = append(recommendations,
			fmt.Sprintf("%s: %s", humanizeFeatureName(delta.Feature), delta.Recommendation))
	}
	return recommendations
}

func shouldRecommendDualWrite(overall CompatibilityLevel, deltas []BrokerFeatureDelta) bool {
	// Recommend dual writes for lower compatibility scores or loss of critical guarantees.
	if overall <= CompatibilityLevelMedium {
		return true
	}

	criticalLosses := map[string]struct{}{
		"transactions":    {},
		"dead_letter":     {},
		"ordering":        {},
		"consumer_groups": {},
	}

	for _, delta := range deltas {
		if delta.Delta != FeatureDeltaTypeLoss {
			continue
		}
		if _, critical := criticalLosses[delta.Feature]; critical {
			return true
		}
	}

	return false
}

func buildSwitchChecklist(plan *BrokerSwitchPlan, targetConfig *BrokerConfig) []string {
	var checklist []string
	if plan == nil || targetConfig == nil {
		return checklist
	}

	checklist = append(checklist,
		fmt.Sprintf("Update broker type from %s to %s in configuration", plan.CurrentType, plan.TargetType),
	)

	if len(targetConfig.Endpoints) > 0 {
		checklist = append(checklist,
			fmt.Sprintf("Validate connectivity to target endpoints: %s", strings.Join(targetConfig.Endpoints, ", ")),
		)
	}

	if plan.DualWriteRecommended {
		checklist = append(checklist, "Enable dual writes and shadow consumers during the migration window")
	}

	for _, issue := range plan.ConfigurationIssues {
		if issue.Resolution != "" {
			checklist = append(checklist, fmt.Sprintf("%s: %s", humanizeFieldName(issue.Field), issue.Resolution))
			continue
		}
		if issue.Issue != "" {
			checklist = append(checklist, fmt.Sprintf("%s: %s", humanizeFieldName(issue.Field), issue.Issue))
		}
	}

	for _, delta := range plan.FeatureDeltas {
		if delta.Delta != FeatureDeltaTypeLoss || delta.Recommendation == "" {
			continue
		}
		checklist = append(checklist,
			fmt.Sprintf("Plan fallback for %s: %s", humanizeFeatureName(delta.Feature), delta.Recommendation),
		)
	}

	checklist = dedupeStrings(checklist)
	sort.Strings(checklist)
	return checklist
}

func humanizeFieldName(field string) string {
	if field == "" {
		return "configuration"
	}
	return humanizeFeatureName(field)
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func cloneMigrationRisks(risks []MigrationRisk) []MigrationRisk {
	if len(risks) == 0 {
		return nil
	}
	clone := make([]MigrationRisk, len(risks))
	copy(clone, risks)
	return clone
}

func cloneMigrationSteps(steps []MigrationStep) []MigrationStep {
	if len(steps) == 0 {
		return nil
	}
	clone := make([]MigrationStep, len(steps))
	copy(clone, steps)
	return clone
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	clone := make([]string, len(values))
	copy(clone, values)
	return clone
}

func cloneConfigurationIssues(issues []ConfigurationIssue) []ConfigurationIssue {
	if len(issues) == 0 {
		return nil
	}
	clone := make([]ConfigurationIssue, len(issues))
	copy(clone, issues)
	return clone
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	clone := make(map[string]string, len(input))
	for key, value := range input {
		clone[key] = value
	}
	return clone
}

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
	"sync"
)

// CompatibilityLevel defines the level of compatibility between brokers.
type CompatibilityLevel int

const (
	// CompatibilityLevelNone indicates no compatibility
	CompatibilityLevelNone CompatibilityLevel = iota

	// CompatibilityLevelLow indicates basic functionality compatibility
	CompatibilityLevelLow

	// CompatibilityLevelMedium indicates good compatibility with some feature gaps
	CompatibilityLevelMedium

	// CompatibilityLevelHigh indicates high compatibility with minor differences
	CompatibilityLevelHigh

	// CompatibilityLevelFull indicates full compatibility
	CompatibilityLevelFull
)

// String returns the string representation of CompatibilityLevel.
func (cl CompatibilityLevel) String() string {
	switch cl {
	case CompatibilityLevelNone:
		return "none"
	case CompatibilityLevelLow:
		return "low"
	case CompatibilityLevelMedium:
		return "medium"
	case CompatibilityLevelHigh:
		return "high"
	case CompatibilityLevelFull:
		return "full"
	default:
		return "unknown"
	}
}

// IsCompatible returns true if the compatibility level is sufficient.
func (cl CompatibilityLevel) IsCompatible() bool {
	return cl >= CompatibilityLevelLow
}

// CompatibilityMatrix represents compatibility comparison between two brokers.
type CompatibilityMatrix struct {
	// Broker1 is the first broker type being compared
	Broker1 BrokerType `json:"broker1"`

	// Broker2 is the second broker type being compared
	Broker2 BrokerType `json:"broker2"`

	// OverallCompatibility is the overall compatibility level
	OverallCompatibility CompatibilityLevel `json:"overall_compatibility"`

	// FeatureComparisons contains detailed feature-by-feature comparison
	FeatureComparisons map[string]*FeatureComparison `json:"feature_comparisons"`

	// CommonFeatures lists features supported by both brokers
	CommonFeatures []string `json:"common_features"`

	// UniqueFeatures maps broker type to its unique features
	UniqueFeatures map[BrokerType][]string `json:"unique_features"`

	// MigrationGuidance provides guidance for migrating between brokers
	MigrationGuidance *MigrationGuidance `json:"migration_guidance,omitempty"`

	// CompatibilityScore is a numeric score (0-100) representing compatibility
	CompatibilityScore int `json:"compatibility_score"`
}

// FeatureComparison represents the comparison of a specific feature between brokers.
type FeatureComparison struct {
	// FeatureName is the name of the feature being compared
	FeatureName string `json:"feature_name"`

	// Broker1Support is the support level in the first broker
	Broker1Support CapabilityLevel `json:"broker1_support"`

	// Broker2Support is the support level in the second broker
	Broker2Support CapabilityLevel `json:"broker2_support"`

	// CompatibilityLevel indicates how compatible the feature is between brokers
	CompatibilityLevel CompatibilityLevel `json:"compatibility_level"`

	// Notes provides additional information about the comparison
	Notes string `json:"notes,omitempty"`

	// Workarounds suggests ways to achieve compatibility
	Workarounds []string `json:"workarounds,omitempty"`
}

// MigrationGuidance provides guidance for migrating from one broker to another.
type MigrationGuidance struct {
	// Difficulty indicates the migration difficulty level
	Difficulty MigrationDifficulty `json:"difficulty"`

	// EstimatedEffort provides an effort estimate
	EstimatedEffort string `json:"estimated_effort"`

	// Steps lists the recommended migration steps
	Steps []MigrationStep `json:"steps"`

	// Considerations lists important considerations for the migration
	Considerations []string `json:"considerations"`

	// Risks lists potential risks and mitigation strategies
	Risks []MigrationRisk `json:"risks"`

	// Tools lists tools that can help with the migration
	Tools []MigrationTool `json:"tools,omitempty"`
}

// MigrationDifficulty indicates the difficulty level of a migration.
type MigrationDifficulty int

const (
	// MigrationDifficultyLow indicates a straightforward migration
	MigrationDifficultyLow MigrationDifficulty = iota

	// MigrationDifficultyMedium indicates moderate migration complexity
	MigrationDifficultyMedium

	// MigrationDifficultyHigh indicates complex migration with significant changes
	MigrationDifficultyHigh

	// MigrationDifficultyVeryHigh indicates very complex migration requiring major changes
	MigrationDifficultyVeryHigh
)

// String returns the string representation of MigrationDifficulty.
func (md MigrationDifficulty) String() string {
	switch md {
	case MigrationDifficultyLow:
		return "low"
	case MigrationDifficultyMedium:
		return "medium"
	case MigrationDifficultyHigh:
		return "high"
	case MigrationDifficultyVeryHigh:
		return "very_high"
	default:
		return "unknown"
	}
}

// MigrationStep represents a step in the migration process.
type MigrationStep struct {
	// Order indicates the sequence order of this step
	Order int `json:"order"`

	// Title is a brief description of the step
	Title string `json:"title"`

	// Description provides detailed instructions for the step
	Description string `json:"description"`

	// Category categorizes the step (preparation, implementation, validation, etc.)
	Category string `json:"category"`

	// EstimatedTime provides an estimate of how long the step will take
	EstimatedTime string `json:"estimated_time,omitempty"`

	// Prerequisites lists requirements for this step
	Prerequisites []string `json:"prerequisites,omitempty"`
}

// MigrationRisk represents a potential risk during migration.
type MigrationRisk struct {
	// Risk describes the potential problem
	Risk string `json:"risk"`

	// Impact describes the potential impact if the risk occurs
	Impact string `json:"impact"`

	// Probability indicates the likelihood of the risk occurring
	Probability string `json:"probability"`

	// Mitigation describes how to prevent or handle the risk
	Mitigation string `json:"mitigation"`
}

// MigrationTool represents a tool that can help with migration.
type MigrationTool struct {
	// Name is the tool name
	Name string `json:"name"`

	// Description describes what the tool does
	Description string `json:"description"`

	// URL provides a link to the tool
	URL string `json:"url,omitempty"`

	// Category categorizes the tool (data migration, configuration, testing, etc.)
	Category string `json:"category"`
}

// BrokerCompatibilityChecker provides methods for checking compatibility between brokers.
type BrokerCompatibilityChecker interface {
	// CheckCompatibility performs compatibility analysis between two brokers.
	CheckCompatibility(ctx context.Context, broker1, broker2 BrokerType) (*CompatibilityMatrix, error)

	// FindCompatibleBrokers finds brokers compatible with the given requirements.
	FindCompatibleBrokers(ctx context.Context, requirements []FeatureRequirement) ([]BrokerCompatibilityResult, error)

	// GetMigrationGuidance provides migration guidance between two brokers.
	GetMigrationGuidance(ctx context.Context, from, to BrokerType) (*MigrationGuidance, error)

	// ValidateConfiguration validates if a configuration is compatible between brokers.
	ValidateConfiguration(ctx context.Context, config *BrokerConfig, targetBroker BrokerType) (*ConfigurationCompatibility, error)
}

// BrokerCompatibilityResult represents the compatibility of a broker with given requirements.
type BrokerCompatibilityResult struct {
	// BrokerType is the broker type
	BrokerType BrokerType `json:"broker_type"`

	// CompatibilityScore is the compatibility score (0-100)
	CompatibilityScore int `json:"compatibility_score"`

	// ValidationResult contains detailed validation results
	ValidationResult *ValidationResult `json:"validation_result"`

	// Capabilities are the broker's capabilities
	Capabilities *BrokerCapabilities `json:"capabilities"`
}

// ConfigurationCompatibility represents compatibility of configuration between brokers.
type ConfigurationCompatibility struct {
	// Compatible indicates if the configuration is compatible
	Compatible bool `json:"compatible"`

	// Issues lists configuration issues that need to be addressed
	Issues []ConfigurationIssue `json:"issues,omitempty"`

	// Mappings provides configuration mappings between brokers
	Mappings map[string]string `json:"mappings,omitempty"`

	// Recommendations provides recommended configuration changes
	Recommendations []string `json:"recommendations,omitempty"`
}

// ConfigurationIssue represents a configuration compatibility issue.
type ConfigurationIssue struct {
	// Severity indicates the severity of the issue
	Severity IssueSeverity `json:"severity"`

	// Field is the configuration field with the issue
	Field string `json:"field"`

	// Issue describes the problem
	Issue string `json:"issue"`

	// Resolution suggests how to fix the issue
	Resolution string `json:"resolution,omitempty"`
}

// IssueSeverity indicates the severity level of a configuration issue.
type IssueSeverity string

const (
	// IssueSeverityInfo is informational
	IssueSeverityInfo IssueSeverity = "info"

	// IssueSeverityWarning is a warning that should be addressed
	IssueSeverityWarning IssueSeverity = "warning"

	// IssueSeverityError is an error that must be fixed
	IssueSeverityError IssueSeverity = "error"

	// IssueSeverityCritical is a critical issue that blocks compatibility
	IssueSeverityCritical IssueSeverity = "critical"
)

// DefaultBrokerCompatibilityChecker provides a default implementation of BrokerCompatibilityChecker.
type DefaultBrokerCompatibilityChecker struct {
	registry CapabilityRegistry
	mutex    sync.RWMutex
}

// NewDefaultBrokerCompatibilityChecker creates a new default compatibility checker.
func NewDefaultBrokerCompatibilityChecker() *DefaultBrokerCompatibilityChecker {
	return &DefaultBrokerCompatibilityChecker{
		registry: NewDefaultCapabilityRegistry(),
	}
}

// CheckCompatibility performs compatibility analysis between two brokers.
func (checker *DefaultBrokerCompatibilityChecker) CheckCompatibility(ctx context.Context, broker1, broker2 BrokerType) (*CompatibilityMatrix, error) {
	if broker1 == broker2 {
		// Same broker type - full compatibility
		capabilities, err := GetCapabilityProfile(broker1)
		if err != nil {
			return nil, fmt.Errorf("failed to get capabilities for %s: %w", broker1, err)
		}

		return &CompatibilityMatrix{
			Broker1:              broker1,
			Broker2:              broker2,
			OverallCompatibility: CompatibilityLevelFull,
			CompatibilityScore:   100,
			CommonFeatures:       getAllSupportedFeatures(capabilities),
			UniqueFeatures:       make(map[BrokerType][]string),
			FeatureComparisons:   make(map[string]*FeatureComparison),
		}, nil
	}

	caps1, err := GetCapabilityProfile(broker1)
	if err != nil {
		return nil, fmt.Errorf("failed to get capabilities for %s: %w", broker1, err)
	}

	caps2, err := GetCapabilityProfile(broker2)
	if err != nil {
		return nil, fmt.Errorf("failed to get capabilities for %s: %w", broker2, err)
	}

	return checker.compareCapabilities(broker1, caps1, broker2, caps2)
}

// FindCompatibleBrokers finds brokers compatible with the given requirements.
func (checker *DefaultBrokerCompatibilityChecker) FindCompatibleBrokers(ctx context.Context, requirements []FeatureRequirement) ([]BrokerCompatibilityResult, error) {
	var results []BrokerCompatibilityResult

	brokerTypes := []BrokerType{BrokerTypeKafka, BrokerTypeNATS, BrokerTypeRabbitMQ, BrokerTypeInMemory}

	for _, brokerType := range brokerTypes {
		capabilities, err := GetCapabilityProfile(brokerType)
		if err != nil {
			continue // Skip brokers we can't get capabilities for
		}

		validation := capabilities.ValidateRequirements(requirements)
		score := calculateCompatibilityScore(validation, len(requirements))

		results = append(results, BrokerCompatibilityResult{
			BrokerType:         brokerType,
			CompatibilityScore: score,
			ValidationResult:   validation,
			Capabilities:       capabilities,
		})
	}

	// Sort by compatibility score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].CompatibilityScore > results[j].CompatibilityScore
	})

	return results, nil
}

// GetMigrationGuidance provides migration guidance between two brokers.
func (checker *DefaultBrokerCompatibilityChecker) GetMigrationGuidance(ctx context.Context, from, to BrokerType) (*MigrationGuidance, error) {
	matrix, err := checker.CheckCompatibility(ctx, from, to)
	if err != nil {
		return nil, err
	}

	return generateMigrationGuidance(from, to, matrix)
}

// ValidateConfiguration validates if a configuration is compatible between brokers.
func (checker *DefaultBrokerCompatibilityChecker) ValidateConfiguration(ctx context.Context, config *BrokerConfig, targetBroker BrokerType) (*ConfigurationCompatibility, error) {
	if config.Type == targetBroker {
		return &ConfigurationCompatibility{
			Compatible: true,
		}, nil
	}

	// Basic validation - in a real implementation, this would be more sophisticated
	var issues []ConfigurationIssue
	mappings := make(map[string]string)
	var recommendations []string

	// Check if broker type change is supported
	issues = append(issues, ConfigurationIssue{
		Severity:   IssueSeverityWarning,
		Field:      "type",
		Issue:      fmt.Sprintf("Changing broker type from %s to %s", config.Type, targetBroker),
		Resolution: "Update client configuration and validate feature compatibility",
	})

	// Add broker-specific recommendations
	switch targetBroker {
	case BrokerTypeKafka:
		recommendations = append(recommendations, "Consider partitioning strategy for optimal performance")
		recommendations = append(recommendations, "Review replication factor settings")
	case BrokerTypeNATS:
		recommendations = append(recommendations, "Enable JetStream for persistence")
		recommendations = append(recommendations, "Consider subject naming conventions")
	case BrokerTypeRabbitMQ:
		recommendations = append(recommendations, "Configure appropriate exchange types")
		recommendations = append(recommendations, "Set up dead letter exchanges if needed")
	}

	compatible := true
	for _, issue := range issues {
		if issue.Severity == IssueSeverityError || issue.Severity == IssueSeverityCritical {
			compatible = false
			break
		}
	}

	return &ConfigurationCompatibility{
		Compatible:      compatible,
		Issues:          issues,
		Mappings:        mappings,
		Recommendations: recommendations,
	}, nil
}

// Helper functions

func (checker *DefaultBrokerCompatibilityChecker) compareCapabilities(broker1 BrokerType, caps1 *BrokerCapabilities, broker2 BrokerType, caps2 *BrokerCapabilities) (*CompatibilityMatrix, error) {
	features := []string{
		"transactions", "ordering", "partitioning", "dead_letter",
		"delayed_delivery", "priority", "streaming", "seek", "consumer_groups",
	}

	featureComparisons := make(map[string]*FeatureComparison)
	var commonFeatures []string
	uniqueFeatures := make(map[BrokerType][]string)

	totalFeatures := len(features)
	compatibleFeatures := 0

	for _, feature := range features {
		level1 := caps1.SupportsFeature(feature)
		level2 := caps2.SupportsFeature(feature)

		comparison := &FeatureComparison{
			FeatureName:    feature,
			Broker1Support: level1,
			Broker2Support: level2,
		}

		if level1.IsSupported() && level2.IsSupported() {
			commonFeatures = append(commonFeatures, feature)
			comparison.CompatibilityLevel = CompatibilityLevelHigh
			compatibleFeatures++
		} else if level1.IsSupported() && !level2.IsSupported() {
			uniqueFeatures[broker1] = append(uniqueFeatures[broker1], feature)
			comparison.CompatibilityLevel = CompatibilityLevelLow
			comparison.Notes = fmt.Sprintf("Feature only supported by %s", broker1)
		} else if !level1.IsSupported() && level2.IsSupported() {
			uniqueFeatures[broker2] = append(uniqueFeatures[broker2], feature)
			comparison.CompatibilityLevel = CompatibilityLevelLow
			comparison.Notes = fmt.Sprintf("Feature only supported by %s", broker2)
		} else {
			// Neither supports it - still compatible in the sense that both lack the feature
			comparison.CompatibilityLevel = CompatibilityLevelHigh
			compatibleFeatures++
		}

		featureComparisons[feature] = comparison
	}

	// Calculate overall compatibility
	compatibilityScore := (compatibleFeatures * 100) / totalFeatures
	var overallCompatibility CompatibilityLevel

	switch {
	case compatibilityScore >= 90:
		overallCompatibility = CompatibilityLevelFull
	case compatibilityScore >= 70:
		overallCompatibility = CompatibilityLevelHigh
	case compatibilityScore >= 50:
		overallCompatibility = CompatibilityLevelMedium
	case compatibilityScore >= 25:
		overallCompatibility = CompatibilityLevelLow
	default:
		overallCompatibility = CompatibilityLevelNone
	}

	return &CompatibilityMatrix{
		Broker1:              broker1,
		Broker2:              broker2,
		OverallCompatibility: overallCompatibility,
		FeatureComparisons:   featureComparisons,
		CommonFeatures:       commonFeatures,
		UniqueFeatures:       uniqueFeatures,
		CompatibilityScore:   compatibilityScore,
	}, nil
}

func getAllSupportedFeatures(capabilities *BrokerCapabilities) []string {
	var features []string

	if capabilities.SupportsTransactions {
		features = append(features, "transactions")
	}
	if capabilities.SupportsOrdering {
		features = append(features, "ordering")
	}
	if capabilities.SupportsPartitioning {
		features = append(features, "partitioning")
	}
	if capabilities.SupportsDeadLetter {
		features = append(features, "dead_letter")
	}
	if capabilities.SupportsDelayedDelivery {
		features = append(features, "delayed_delivery")
	}
	if capabilities.SupportsPriority {
		features = append(features, "priority")
	}
	if capabilities.SupportsStreaming {
		features = append(features, "streaming")
	}
	if capabilities.SupportsSeek {
		features = append(features, "seek")
	}
	if capabilities.SupportsConsumerGroups {
		features = append(features, "consumer_groups")
	}

	return features
}

func calculateCompatibilityScore(validation *ValidationResult, totalRequirements int) int {
	if totalRequirements == 0 {
		return 100
	}

	satisfiedCount := len(validation.Satisfied)
	return (satisfiedCount * 100) / totalRequirements
}

func generateMigrationGuidance(from, to BrokerType, matrix *CompatibilityMatrix) (*MigrationGuidance, error) {
	var difficulty MigrationDifficulty
	var estimatedEffort string
	var steps []MigrationStep
	var considerations []string
	var risks []MigrationRisk

	// Determine difficulty based on compatibility score
	switch {
	case matrix.CompatibilityScore >= 80:
		difficulty = MigrationDifficultyLow
		estimatedEffort = "1-2 weeks"
	case matrix.CompatibilityScore >= 60:
		difficulty = MigrationDifficultyMedium
		estimatedEffort = "2-4 weeks"
	case matrix.CompatibilityScore >= 40:
		difficulty = MigrationDifficultyHigh
		estimatedEffort = "1-2 months"
	default:
		difficulty = MigrationDifficultyVeryHigh
		estimatedEffort = "2+ months"
	}

	// Generate generic migration steps
	steps = []MigrationStep{
		{
			Order:         1,
			Title:         "Assessment and Planning",
			Description:   "Analyze current usage patterns and identify required features",
			Category:      "preparation",
			EstimatedTime: "1-2 days",
		},
		{
			Order:         2,
			Title:         "Setup Target Environment",
			Description:   fmt.Sprintf("Deploy and configure %s infrastructure", to),
			Category:      "infrastructure",
			EstimatedTime: "1-3 days",
		},
		{
			Order:         3,
			Title:         "Application Updates",
			Description:   "Update application code to use new broker client libraries",
			Category:      "implementation",
			EstimatedTime: "1-2 weeks",
		},
		{
			Order:         4,
			Title:         "Testing",
			Description:   "Thoroughly test the migration with staging data",
			Category:      "validation",
			EstimatedTime: "1 week",
		},
		{
			Order:         5,
			Title:         "Production Migration",
			Description:   "Execute the production migration with rollback plan",
			Category:      "deployment",
			EstimatedTime: "1-2 days",
		},
	}

	// Add general considerations
	considerations = []string{
		"Message format compatibility",
		"Client library changes",
		"Monitoring and alerting updates",
		"Backup and disaster recovery procedures",
	}

	// Add general risks
	risks = []MigrationRisk{
		{
			Risk:        "Data loss during migration",
			Impact:      "Critical business impact",
			Probability: "Low",
			Mitigation:  "Implement comprehensive backup strategy and use dual-write pattern during transition",
		},
		{
			Risk:        "Application downtime",
			Impact:      "Service interruption",
			Probability: "Medium",
			Mitigation:  "Use blue-green deployment strategy and implement circuit breakers",
		},
	}

	// Add broker-specific considerations and risks
	if len(matrix.UniqueFeatures[from]) > 0 {
		considerations = append(considerations, fmt.Sprintf("Features unique to %s that may need alternative implementations", from))

		risks = append(risks, MigrationRisk{
			Risk:        fmt.Sprintf("Loss of %s-specific features", from),
			Impact:      "Functionality reduction",
			Probability: "High",
			Mitigation:  "Implement alternative solutions or accept feature loss",
		})
	}

	return &MigrationGuidance{
		Difficulty:      difficulty,
		EstimatedEffort: estimatedEffort,
		Steps:           steps,
		Considerations:  considerations,
		Risks:           risks,
	}, nil
}

// DefaultCapabilityRegistry provides a simple in-memory implementation of CapabilityRegistry.
type DefaultCapabilityRegistry struct {
	capabilities map[BrokerType]*BrokerCapabilities
	mutex        sync.RWMutex
}

// NewDefaultCapabilityRegistry creates a new default capability registry.
func NewDefaultCapabilityRegistry() *DefaultCapabilityRegistry {
	registry := &DefaultCapabilityRegistry{
		capabilities: make(map[BrokerType]*BrokerCapabilities),
	}

	// Pre-populate with default capability profiles
	brokerTypes := []BrokerType{BrokerTypeKafka, BrokerTypeNATS, BrokerTypeRabbitMQ, BrokerTypeInMemory}
	for _, brokerType := range brokerTypes {
		if caps, err := GetCapabilityProfile(brokerType); err == nil {
			registry.capabilities[brokerType] = caps
		}
	}

	return registry
}

// RegisterBrokerCapabilities registers capabilities for a broker type.
func (registry *DefaultCapabilityRegistry) RegisterBrokerCapabilities(brokerType BrokerType, capabilities *BrokerCapabilities) error {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	if capabilities == nil {
		return fmt.Errorf("capabilities cannot be nil")
	}

	registry.capabilities[brokerType] = capabilities.Clone()
	return nil
}

// GetBrokerCapabilities retrieves capabilities for a broker type.
func (registry *DefaultCapabilityRegistry) GetBrokerCapabilities(brokerType BrokerType) (*BrokerCapabilities, error) {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	if capabilities, exists := registry.capabilities[brokerType]; exists {
		return capabilities.Clone(), nil
	}

	return nil, fmt.Errorf("no capabilities registered for broker type: %s", brokerType)
}

// FindCompatibleBrokers finds brokers that meet the specified requirements.
func (registry *DefaultCapabilityRegistry) FindCompatibleBrokers(requirements []FeatureRequirement) ([]BrokerType, error) {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	var compatibleBrokers []BrokerType

	for brokerType, capabilities := range registry.capabilities {
		validation := capabilities.ValidateRequirements(requirements)
		if validation.Valid {
			compatibleBrokers = append(compatibleBrokers, brokerType)
		}
	}

	return compatibleBrokers, nil
}

// CompareCapabilities compares capabilities between different brokers.
func (registry *DefaultCapabilityRegistry) CompareCapabilities(broker1, broker2 BrokerType) (map[string]interface{}, error) {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	caps1, exists1 := registry.capabilities[broker1]
	if !exists1 {
		return nil, fmt.Errorf("no capabilities registered for broker type: %s", broker1)
	}

	caps2, exists2 := registry.capabilities[broker2]
	if !exists2 {
		return nil, fmt.Errorf("no capabilities registered for broker type: %s", broker2)
	}

	checker := NewDefaultBrokerCompatibilityChecker()
	matrix, err := checker.compareCapabilities(broker1, caps1, broker2, caps2)
	if err != nil {
		return nil, err
	}

	// Convert to map[string]interface{} to match interface
	result := make(map[string]interface{})
	result["broker1"] = matrix.Broker1
	result["broker2"] = matrix.Broker2
	result["overall_compatibility"] = matrix.OverallCompatibility
	result["feature_comparisons"] = matrix.FeatureComparisons
	result["common_features"] = matrix.CommonFeatures
	result["unique_features"] = matrix.UniqueFeatures
	result["compatibility_score"] = matrix.CompatibilityScore

	return result, nil
}

// ListSupportedBrokers returns all registered broker types.
func (registry *DefaultCapabilityRegistry) ListSupportedBrokers() []BrokerType {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	brokers := make([]BrokerType, 0, len(registry.capabilities))
	for brokerType := range registry.capabilities {
		brokers = append(brokers, brokerType)
	}

	sort.Slice(brokers, func(i, j int) bool {
		return string(brokers[i]) < string(brokers[j])
	})

	return brokers
}

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

package messaging

import (
	"context"
	"testing"
)

func TestCompatibilityLevel(t *testing.T) {
	tests := []struct {
		level      CompatibilityLevel
		expected   string
		compatible bool
	}{
		{CompatibilityLevelNone, "none", false},
		{CompatibilityLevelLow, "low", true},
		{CompatibilityLevelMedium, "medium", true},
		{CompatibilityLevelHigh, "high", true},
		{CompatibilityLevelFull, "full", true},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			if test.level.String() != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, test.level.String())
			}
			if test.level.IsCompatible() != test.compatible {
				t.Errorf("Expected IsCompatible() to return %v", test.compatible)
			}
		})
	}
}

func TestMigrationDifficulty(t *testing.T) {
	tests := []struct {
		difficulty MigrationDifficulty
		expected   string
	}{
		{MigrationDifficultyLow, "low"},
		{MigrationDifficultyMedium, "medium"},
		{MigrationDifficultyHigh, "high"},
		{MigrationDifficultyVeryHigh, "very_high"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			if test.difficulty.String() != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, test.difficulty.String())
			}
		})
	}
}

func TestDefaultBrokerCompatibilityChecker_CheckCompatibility(t *testing.T) {
	checker := NewDefaultBrokerCompatibilityChecker()
	ctx := context.Background()

	t.Run("Same Broker Type", func(t *testing.T) {
		matrix, err := checker.CheckCompatibility(ctx, BrokerTypeKafka, BrokerTypeKafka)
		if err != nil {
			t.Fatalf("Failed to check compatibility: %v", err)
		}

		if matrix.Broker1 != BrokerTypeKafka || matrix.Broker2 != BrokerTypeKafka {
			t.Error("Broker types not set correctly")
		}
		if matrix.OverallCompatibility != CompatibilityLevelFull {
			t.Error("Same broker should have full compatibility")
		}
		if matrix.CompatibilityScore != 100 {
			t.Error("Same broker should have 100% compatibility score")
		}
	})

	t.Run("Different Broker Types", func(t *testing.T) {
		matrix, err := checker.CheckCompatibility(ctx, BrokerTypeKafka, BrokerTypeRabbitMQ)
		if err != nil {
			t.Fatalf("Failed to check compatibility: %v", err)
		}

		if matrix.Broker1 != BrokerTypeKafka || matrix.Broker2 != BrokerTypeRabbitMQ {
			t.Error("Broker types not set correctly")
		}
		if matrix.OverallCompatibility == CompatibilityLevelFull {
			t.Error("Different brokers should not have full compatibility")
		}
		if matrix.CompatibilityScore < 0 || matrix.CompatibilityScore > 100 {
			t.Errorf("Compatibility score should be 0-100, got %d", matrix.CompatibilityScore)
		}
		if len(matrix.FeatureComparisons) == 0 {
			t.Error("Feature comparisons should not be empty")
		}
	})

	t.Run("Unknown Broker Type", func(t *testing.T) {
		_, err := checker.CheckCompatibility(ctx, BrokerType("unknown"), BrokerTypeKafka)
		if err == nil {
			t.Error("Expected error for unknown broker type")
		}
	})

	t.Run("Feature Analysis", func(t *testing.T) {
		matrix, err := checker.CheckCompatibility(ctx, BrokerTypeKafka, BrokerTypeNATS)
		if err != nil {
			t.Fatalf("Failed to check compatibility: %v", err)
		}

		// Check specific feature comparison
		if comparison, exists := matrix.FeatureComparisons["partitioning"]; exists {
			if comparison.FeatureName != "partitioning" {
				t.Error("Feature name not set correctly")
			}
			// Kafka supports partitioning, NATS doesn't
			if comparison.Broker1Support == CapabilityLevelNotSupported {
				t.Error("Kafka should support partitioning")
			}
			if comparison.Broker2Support != CapabilityLevelNotSupported {
				t.Error("NATS should not support partitioning")
			}
		}
	})
}

func TestDefaultBrokerCompatibilityChecker_FindCompatibleBrokers(t *testing.T) {
	checker := NewDefaultBrokerCompatibilityChecker()
	ctx := context.Background()

	t.Run("Basic Requirements", func(t *testing.T) {
		requirements := []FeatureRequirement{
			{Name: "consumer_groups", Level: CapabilityLevelFull, Optional: false},
		}

		results, err := checker.FindCompatibleBrokers(ctx, requirements)
		if err != nil {
			t.Fatalf("Failed to find compatible brokers: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected to find at least one compatible broker")
		}

		// Results should be sorted by compatibility score
		for i := 1; i < len(results); i++ {
			if results[i-1].CompatibilityScore < results[i].CompatibilityScore {
				t.Error("Results should be sorted by compatibility score (descending)")
			}
		}

		// Check that all results have valid compatibility scores
		for _, result := range results {
			if result.CompatibilityScore < 0 || result.CompatibilityScore > 100 {
				t.Errorf("Invalid compatibility score: %d", result.CompatibilityScore)
			}
		}
	})

	t.Run("Strict Requirements", func(t *testing.T) {
		requirements := []FeatureRequirement{
			{Name: "transactions", Level: CapabilityLevelFull, Optional: false},
			{Name: "partitioning", Level: CapabilityLevelFull, Optional: false},
			{Name: "ordering", Level: CapabilityLevelFull, Optional: false},
		}

		results, err := checker.FindCompatibleBrokers(ctx, requirements)
		if err != nil {
			t.Fatalf("Failed to find compatible brokers: %v", err)
		}

		// Kafka should be highly compatible with these requirements
		var kafkaResult *BrokerCompatibilityResult
		for _, result := range results {
			if result.BrokerType == BrokerTypeKafka {
				kafkaResult = &result
				break
			}
		}

		if kafkaResult == nil {
			t.Error("Kafka should be in the results")
		} else if !kafkaResult.ValidationResult.Valid {
			t.Error("Kafka should satisfy these requirements")
		}
	})

	t.Run("Optional Requirements", func(t *testing.T) {
		requirements := []FeatureRequirement{
			{Name: "consumer_groups", Level: CapabilityLevelFull, Optional: false},
			{Name: "priority", Level: CapabilityLevelFull, Optional: true},
		}

		results, err := checker.FindCompatibleBrokers(ctx, requirements)
		if err != nil {
			t.Fatalf("Failed to find compatible brokers: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected to find compatible brokers")
		}

		// All brokers should be compatible since priority is optional
		for _, result := range results {
			if !result.ValidationResult.Valid && len(result.ValidationResult.Missing) > 0 {
				// Check if missing features are only optional ones
				for _, missing := range result.ValidationResult.Missing {
					found := false
					for _, req := range requirements {
						if req.Name == missing && req.Optional {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Non-optional feature %s is missing for broker %s", missing, result.BrokerType)
					}
				}
			}
		}
	})
}

func TestDefaultBrokerCompatibilityChecker_GetMigrationGuidance(t *testing.T) {
	checker := NewDefaultBrokerCompatibilityChecker()
	ctx := context.Background()

	t.Run("Same Broker Migration", func(t *testing.T) {
		guidance, err := checker.GetMigrationGuidance(ctx, BrokerTypeKafka, BrokerTypeKafka)
		if err != nil {
			t.Fatalf("Failed to get migration guidance: %v", err)
		}

		if guidance.Difficulty != MigrationDifficultyLow {
			t.Error("Same broker migration should be low difficulty")
		}
		if len(guidance.Steps) == 0 {
			t.Error("Migration steps should not be empty")
		}
	})

	t.Run("Different Broker Migration", func(t *testing.T) {
		guidance, err := checker.GetMigrationGuidance(ctx, BrokerTypeKafka, BrokerTypeRabbitMQ)
		if err != nil {
			t.Fatalf("Failed to get migration guidance: %v", err)
		}

		if guidance.Difficulty == MigrationDifficultyLow {
			t.Error("Different broker migration should not be low difficulty")
		}
		if len(guidance.Steps) == 0 {
			t.Error("Migration steps should not be empty")
		}
		if len(guidance.Considerations) == 0 {
			t.Error("Migration considerations should not be empty")
		}
		if len(guidance.Risks) == 0 {
			t.Error("Migration risks should not be empty")
		}

		// Verify steps are ordered
		for i, step := range guidance.Steps {
			if step.Order != i+1 {
				t.Errorf("Step order should be %d, got %d", i+1, step.Order)
			}
		}
	})

	t.Run("Complex Migration", func(t *testing.T) {
		guidance, err := checker.GetMigrationGuidance(ctx, BrokerTypeNATS, BrokerTypeKafka)
		if err != nil {
			t.Fatalf("Failed to get migration guidance: %v", err)
		}

		// Should have standard migration steps
		expectedCategories := []string{"preparation", "infrastructure", "implementation", "validation", "deployment"}
		categoryFound := make(map[string]bool)

		for _, step := range guidance.Steps {
			categoryFound[step.Category] = true
		}

		for _, category := range expectedCategories {
			if !categoryFound[category] {
				t.Errorf("Expected migration step category %s not found", category)
			}
		}
	})
}

func TestDefaultBrokerCompatibilityChecker_ValidateConfiguration(t *testing.T) {
	checker := NewDefaultBrokerCompatibilityChecker()
	ctx := context.Background()

	t.Run("Same Broker Configuration", func(t *testing.T) {
		config := &BrokerConfig{
			Type:      BrokerTypeKafka,
			Endpoints: []string{"localhost:9092"},
		}

		compatibility, err := checker.ValidateConfiguration(ctx, config, BrokerTypeKafka)
		if err != nil {
			t.Fatalf("Failed to validate configuration: %v", err)
		}

		if !compatibility.Compatible {
			t.Error("Same broker configuration should be compatible")
		}
	})

	t.Run("Different Broker Configuration", func(t *testing.T) {
		config := &BrokerConfig{
			Type:      BrokerTypeKafka,
			Endpoints: []string{"localhost:9092"},
		}

		compatibility, err := checker.ValidateConfiguration(ctx, config, BrokerTypeRabbitMQ)
		if err != nil {
			t.Fatalf("Failed to validate configuration: %v", err)
		}

		if len(compatibility.Issues) == 0 {
			t.Error("Different broker configuration should have issues")
		}
		if len(compatibility.Recommendations) == 0 {
			t.Error("Different broker configuration should have recommendations")
		}

		// Check that we have at least one issue about broker type change
		foundTypeIssue := false
		for _, issue := range compatibility.Issues {
			if issue.Field == "type" {
				foundTypeIssue = true
				break
			}
		}
		if !foundTypeIssue {
			t.Error("Expected to find issue about broker type change")
		}
	})
}

func TestDefaultCapabilityRegistry(t *testing.T) {
	registry := NewDefaultCapabilityRegistry()

	t.Run("Pre-populated Capabilities", func(t *testing.T) {
		brokers := registry.ListSupportedBrokers()
		if len(brokers) == 0 {
			t.Error("Registry should be pre-populated with broker capabilities")
		}

		// Check that we have the expected brokers
		expectedBrokers := []BrokerType{BrokerTypeKafka, BrokerTypeNATS, BrokerTypeRabbitMQ, BrokerTypeInMemory}
		for _, expected := range expectedBrokers {
			found := false
			for _, broker := range brokers {
				if broker == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected broker %s not found in registry", expected)
			}
		}
	})

	t.Run("Get Broker Capabilities", func(t *testing.T) {
		capabilities, err := registry.GetBrokerCapabilities(BrokerTypeKafka)
		if err != nil {
			t.Fatalf("Failed to get Kafka capabilities: %v", err)
		}

		if capabilities == nil {
			t.Error("Capabilities should not be nil")
		}

		// Should be a clone, not the original
		originalCaps, _ := registry.GetBrokerCapabilities(BrokerTypeKafka)
		capabilities.SupportsTransactions = !capabilities.SupportsTransactions
		if originalCaps.SupportsTransactions == capabilities.SupportsTransactions {
			t.Error("Should return cloned capabilities")
		}
	})

	t.Run("Register New Capabilities", func(t *testing.T) {
		customBroker := BrokerType("custom")
		customCaps := &BrokerCapabilities{
			SupportsTransactions: true,
		}

		err := registry.RegisterBrokerCapabilities(customBroker, customCaps)
		if err != nil {
			t.Fatalf("Failed to register capabilities: %v", err)
		}

		retrievedCaps, err := registry.GetBrokerCapabilities(customBroker)
		if err != nil {
			t.Fatalf("Failed to get registered capabilities: %v", err)
		}

		if !retrievedCaps.SupportsTransactions {
			t.Error("Custom capabilities not registered correctly")
		}
	})

	t.Run("Register Nil Capabilities", func(t *testing.T) {
		err := registry.RegisterBrokerCapabilities(BrokerType("nil_test"), nil)
		if err == nil {
			t.Error("Should not allow registering nil capabilities")
		}
	})

	t.Run("Find Compatible Brokers", func(t *testing.T) {
		requirements := []FeatureRequirement{
			{Name: "transactions", Level: CapabilityLevelFull, Optional: false},
		}

		compatible, err := registry.FindCompatibleBrokers(requirements)
		if err != nil {
			t.Fatalf("Failed to find compatible brokers: %v", err)
		}

		if len(compatible) == 0 {
			t.Error("Should find at least one compatible broker")
		}

		// Verify all returned brokers actually support transactions
		for _, broker := range compatible {
			caps, err := registry.GetBrokerCapabilities(broker)
			if err != nil {
				t.Fatalf("Failed to get capabilities for %s: %v", broker, err)
			}
			if !caps.SupportsTransactions {
				t.Errorf("Broker %s should support transactions", broker)
			}
		}
	})

	t.Run("Get Unknown Broker", func(t *testing.T) {
		_, err := registry.GetBrokerCapabilities(BrokerType("unknown"))
		if err == nil {
			t.Error("Should return error for unknown broker")
		}
	})
}

func TestCompatibilityMatrix(t *testing.T) {
	matrix := &CompatibilityMatrix{
		Broker1:              BrokerTypeKafka,
		Broker2:              BrokerTypeRabbitMQ,
		OverallCompatibility: CompatibilityLevelMedium,
		CompatibilityScore:   75,
		CommonFeatures:       []string{"transactions", "consumer_groups"},
		UniqueFeatures: map[BrokerType][]string{
			BrokerTypeKafka:    {"partitioning", "streaming"},
			BrokerTypeRabbitMQ: {"priority", "dead_letter"},
		},
		FeatureComparisons: map[string]*FeatureComparison{
			"transactions": {
				FeatureName:        "transactions",
				Broker1Support:     CapabilityLevelFull,
				Broker2Support:     CapabilityLevelFull,
				CompatibilityLevel: CompatibilityLevelHigh,
			},
		},
	}

	if matrix.Broker1 != BrokerTypeKafka {
		t.Error("Broker1 not set correctly")
	}
	if matrix.Broker2 != BrokerTypeRabbitMQ {
		t.Error("Broker2 not set correctly")
	}
	if matrix.OverallCompatibility != CompatibilityLevelMedium {
		t.Error("Overall compatibility not set correctly")
	}
	if matrix.CompatibilityScore != 75 {
		t.Error("Compatibility score not set correctly")
	}
	if len(matrix.CommonFeatures) != 2 {
		t.Error("Common features not set correctly")
	}
	if len(matrix.UniqueFeatures[BrokerTypeKafka]) != 2 {
		t.Error("Unique features for Kafka not set correctly")
	}
}

func TestMigrationGuidance(t *testing.T) {
	guidance := &MigrationGuidance{
		Difficulty:      MigrationDifficultyMedium,
		EstimatedEffort: "2-4 weeks",
		Steps: []MigrationStep{
			{Order: 1, Title: "Assessment", Category: "preparation"},
			{Order: 2, Title: "Setup", Category: "infrastructure"},
		},
		Considerations: []string{"Message format compatibility", "Performance differences"},
		Risks: []MigrationRisk{
			{Risk: "Data loss", Impact: "High", Probability: "Low", Mitigation: "Use backup strategy"},
		},
	}

	if guidance.Difficulty != MigrationDifficultyMedium {
		t.Error("Difficulty not set correctly")
	}
	if len(guidance.Steps) != 2 {
		t.Error("Steps not set correctly")
	}
	if len(guidance.Considerations) != 2 {
		t.Error("Considerations not set correctly")
	}
	if len(guidance.Risks) != 1 {
		t.Error("Risks not set correctly")
	}

	// Test step ordering
	if guidance.Steps[0].Order != 1 || guidance.Steps[1].Order != 2 {
		t.Error("Steps not ordered correctly")
	}
}

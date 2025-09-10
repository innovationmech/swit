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
	"testing"
)

func TestCapabilityLevel(t *testing.T) {
	tests := []struct {
		level     CapabilityLevel
		expected  string
		supported bool
	}{
		{CapabilityLevelNotSupported, "not_supported", false},
		{CapabilityLevelBasic, "basic", true},
		{CapabilityLevelFull, "full", true},
		{CapabilityLevelEnhanced, "enhanced", true},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			if test.level.String() != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, test.level.String())
			}
			if test.level.IsSupported() != test.supported {
				t.Errorf("Expected IsSupported() to return %v", test.supported)
			}
		})
	}
}

func TestBrokerCapabilitiesBuilder(t *testing.T) {
	t.Run("Basic Builder Usage", func(t *testing.T) {
		capabilities, err := NewBrokerCapabilitiesBuilder().
			WithTransactions(true).
			WithOrdering(true).
			WithPartitioning(false).
			WithMaxMessageSize(1024).
			WithCompressionTypes(CompressionNone, CompressionGZIP).
			WithSerializationTypes(SerializationJSON, SerializationProtobuf).
			Build()

		if err != nil {
			t.Fatalf("Failed to build capabilities: %v", err)
		}

		if !capabilities.SupportsTransactions {
			t.Error("Expected transactions to be supported")
		}
		if !capabilities.SupportsOrdering {
			t.Error("Expected ordering to be supported")
		}
		if capabilities.SupportsPartitioning {
			t.Error("Expected partitioning to not be supported")
		}
		if capabilities.MaxMessageSize != 1024 {
			t.Errorf("Expected max message size to be 1024, got %d", capabilities.MaxMessageSize)
		}
		if len(capabilities.SupportedCompressionTypes) != 2 {
			t.Errorf("Expected 2 compression types, got %d", len(capabilities.SupportedCompressionTypes))
		}
		if len(capabilities.SupportedSerializationTypes) != 2 {
			t.Errorf("Expected 2 serialization types, got %d", len(capabilities.SupportedSerializationTypes))
		}
	})

	t.Run("Extended Capabilities", func(t *testing.T) {
		capabilities, err := NewBrokerCapabilitiesBuilder().
			WithExtendedCapability("custom_feature", true).
			WithExtendedCapability("version", "1.0").
			Build()

		if err != nil {
			t.Fatalf("Failed to build capabilities: %v", err)
		}

		if val, exists := capabilities.GetExtendedCapability("custom_feature"); !exists || val != true {
			t.Error("Expected custom_feature to be true")
		}
		if val, exists := capabilities.GetExtendedCapability("version"); !exists || val != "1.0" {
			t.Error("Expected version to be 1.0")
		}
	})

	t.Run("Builder Validation Errors", func(t *testing.T) {
		_, err := NewBrokerCapabilitiesBuilder().
			WithMaxMessageSize(-1).
			WithMaxBatchSize(-10).
			Build()

		if err == nil {
			t.Error("Expected validation errors for negative values")
		}
	})

	t.Run("Default Values", func(t *testing.T) {
		capabilities, err := NewBrokerCapabilitiesBuilder().Build()
		if err != nil {
			t.Fatalf("Failed to build capabilities: %v", err)
		}

		if len(capabilities.SupportedCompressionTypes) == 0 {
			t.Error("Expected default compression types")
		}
		if len(capabilities.SupportedSerializationTypes) == 0 {
			t.Error("Expected default serialization types")
		}
	})
}

func TestGetCapabilityProfile(t *testing.T) {
	tests := []struct {
		brokerType  BrokerType
		shouldError bool
	}{
		{BrokerTypeKafka, false},
		{BrokerTypeNATS, false},
		{BrokerTypeRabbitMQ, false},
		{BrokerTypeInMemory, false},
		{BrokerType("unknown"), true},
	}

	for _, test := range tests {
		t.Run(string(test.brokerType), func(t *testing.T) {
			capabilities, err := GetCapabilityProfile(test.brokerType)

			if test.shouldError {
				if err == nil {
					t.Error("Expected error for unknown broker type")
				}
				return
			}

			if err != nil {
				t.Fatalf("Failed to get capability profile: %v", err)
			}

			if capabilities == nil {
				t.Fatal("Capabilities should not be nil")
			}

			// Verify some expected characteristics based on broker type
			switch test.brokerType {
			case BrokerTypeKafka:
				if !capabilities.SupportsTransactions {
					t.Error("Kafka should support transactions")
				}
				if !capabilities.SupportsPartitioning {
					t.Error("Kafka should support partitioning")
				}
				if !capabilities.SupportsOrdering {
					t.Error("Kafka should support ordering")
				}
			case BrokerTypeRabbitMQ:
				if !capabilities.SupportsDeadLetter {
					t.Error("RabbitMQ should support dead letter queues")
				}
				if !capabilities.SupportsPriority {
					t.Error("RabbitMQ should support message priority")
				}
			case BrokerTypeNATS:
				if !capabilities.SupportsStreaming {
					t.Error("NATS should support streaming (JetStream)")
				}
				if capabilities.SupportsPartitioning {
					t.Error("NATS should not support partitioning")
				}
			case BrokerTypeInMemory:
				// InMemory should support everything for testing
				if !capabilities.SupportsTransactions {
					t.Error("InMemory should support transactions")
				}
			}
		})
	}
}

func TestBrokerCapabilitiesSupportsFeature(t *testing.T) {
	capabilities := &BrokerCapabilities{
		SupportsTransactions: true,
		SupportsOrdering:     false,
		Extended: map[string]any{
			"custom_feature":   true,
			"disabled_feature": false,
		},
	}

	tests := []struct {
		feature  string
		expected CapabilityLevel
	}{
		{"transactions", CapabilityLevelFull},
		{"ordering", CapabilityLevelNotSupported},
		{"partitioning", CapabilityLevelNotSupported},
		{"custom_feature", CapabilityLevelFull},
		{"disabled_feature", CapabilityLevelNotSupported},
		{"unknown_feature", CapabilityLevelNotSupported},
	}

	for _, test := range tests {
		t.Run(test.feature, func(t *testing.T) {
			level := capabilities.SupportsFeature(test.feature)
			if level != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, level)
			}
		})
	}
}

func TestBrokerCapabilitiesValidateRequirements(t *testing.T) {
	capabilities := &BrokerCapabilities{
		SupportsTransactions:   true,
		SupportsOrdering:       true,
		SupportsPartitioning:   false,
		SupportsConsumerGroups: true,
	}

	requirements := []FeatureRequirement{
		{
			Name:     "transactions",
			Level:    CapabilityLevelFull,
			Optional: false,
		},
		{
			Name:     "partitioning",
			Level:    CapabilityLevelBasic,
			Optional: true,
		},
		{
			Name:     "dead_letter",
			Level:    CapabilityLevelFull,
			Optional: false,
			Fallback: []string{"consumer_groups"},
		},
		{
			Name:     "consumer_groups",
			Level:    CapabilityLevelBasic,
			Optional: false,
		},
	}

	result := capabilities.ValidateRequirements(requirements)

	// Check satisfied requirements
	if len(result.Satisfied) != 2 {
		t.Errorf("Expected 2 satisfied requirements, got %d", len(result.Satisfied))
	}
	if !contains(result.Satisfied, "transactions") {
		t.Error("Expected transactions to be satisfied")
	}
	if !contains(result.Satisfied, "consumer_groups") {
		t.Error("Expected consumer_groups to be satisfied")
	}

	// Check optional requirements
	if len(result.Optional) != 1 {
		t.Errorf("Expected 1 optional requirement, got %d", len(result.Optional))
	}
	if !contains(result.Optional, "partitioning") {
		t.Error("Expected partitioning to be optional")
	}

	// Check missing requirements
	if len(result.Missing) != 1 {
		t.Errorf("Expected 1 missing requirement, got %d", len(result.Missing))
	}
	if !contains(result.Missing, "dead_letter") {
		t.Error("Expected dead_letter to be missing")
	}

	// Check fallbacks
	if len(result.Fallbacks) != 1 {
		t.Errorf("Expected 1 fallback mapping, got %d", len(result.Fallbacks))
	}
	if fallbacks, exists := result.Fallbacks["dead_letter"]; !exists || !contains(fallbacks, "consumer_groups") {
		t.Error("Expected consumer_groups as fallback for dead_letter")
	}

	// Should not be valid due to missing required feature
	if result.Valid {
		t.Error("Expected result to be invalid due to missing required feature")
	}
}

func TestBrokerCapabilitiesClone(t *testing.T) {
	original := &BrokerCapabilities{
		SupportsTransactions:        true,
		SupportedCompressionTypes:   []CompressionType{CompressionNone, CompressionGZIP},
		SupportedSerializationTypes: []SerializationType{SerializationJSON},
		Extended: map[string]any{
			"key1": "value1",
			"key2": 42,
		},
	}

	clone := original.Clone()

	if clone == nil {
		t.Fatal("Clone should not be nil")
	}

	// Verify values are copied
	if clone.SupportsTransactions != original.SupportsTransactions {
		t.Error("SupportsTransactions not copied correctly")
	}

	// Verify it's a deep copy by modifying the clone
	clone.SupportsTransactions = false
	if original.SupportsTransactions == false {
		t.Error("Modifying clone affected original")
	}

	// Test slice deep copy
	clone.SupportedCompressionTypes[0] = CompressionSnappy
	if original.SupportedCompressionTypes[0] == CompressionSnappy {
		t.Error("Modifying clone slice affected original")
	}

	// Test map deep copy
	clone.Extended["key1"] = "modified"
	if original.Extended["key1"] == "modified" {
		t.Error("Modifying clone map affected original")
	}

	// Test cloning nil
	var nilCaps *BrokerCapabilities
	nilClone := nilCaps.Clone()
	if nilClone != nil {
		t.Error("Cloning nil should return nil")
	}
}

func TestFeatureRequirement(t *testing.T) {
	req := FeatureRequirement{
		Name:        "transactions",
		Level:       CapabilityLevelFull,
		Optional:    false,
		Fallback:    []string{"ordering"},
		Description: "Transaction support required",
	}

	if req.Name != "transactions" {
		t.Error("Name not set correctly")
	}
	if req.Level != CapabilityLevelFull {
		t.Error("Level not set correctly")
	}
	if req.Optional {
		t.Error("Optional should be false")
	}
	if len(req.Fallback) != 1 || req.Fallback[0] != "ordering" {
		t.Error("Fallback not set correctly")
	}
	if req.Description != "Transaction support required" {
		t.Error("Description not set correctly")
	}
}

func TestValidationResult(t *testing.T) {
	result := &ValidationResult{
		Valid:     false,
		Satisfied: []string{"feature1", "feature2"},
		Missing:   []string{"feature3"},
		Optional:  []string{"feature4"},
		Fallbacks: map[string][]string{
			"feature3": {"alternative1", "alternative2"},
		},
		Details: map[string]string{
			"feature1": "fully supported",
			"feature3": "not supported",
		},
	}

	if result.Valid {
		t.Error("Expected Valid to be false")
	}
	if len(result.Satisfied) != 2 {
		t.Error("Expected 2 satisfied features")
	}
	if len(result.Missing) != 1 {
		t.Error("Expected 1 missing feature")
	}
	if len(result.Optional) != 1 {
		t.Error("Expected 1 optional feature")
	}
	if len(result.Fallbacks["feature3"]) != 2 {
		t.Error("Expected 2 fallback options")
	}
	if result.Details["feature1"] != "fully supported" {
		t.Error("Expected correct detail for feature1")
	}
}

// Helper function to check if slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

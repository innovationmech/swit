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

package adapters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/innovationmech/swit/pkg/messaging"
)

// BrokerSelector provides methods for selecting appropriate brokers based on requirements.
// This enables automatic broker selection based on capabilities, performance requirements,
// and configuration preferences, supporting the adapter pattern goals.
type BrokerSelector struct {
	registry BrokerAdapterRegistry
}

// NewBrokerSelector creates a new broker selector with the given adapter registry.
func NewBrokerSelector(registry BrokerAdapterRegistry) *BrokerSelector {
	return &BrokerSelector{
		registry: registry,
	}
}

// SelectionCriteria defines criteria for broker selection.
type SelectionCriteria struct {
	// RequiredCapabilities lists capabilities that the broker must support
	RequiredCapabilities []CapabilityRequirement `json:"required_capabilities,omitempty"`

	// PreferredBrokerTypes lists broker types in order of preference
	PreferredBrokerTypes []messaging.BrokerType `json:"preferred_broker_types,omitempty"`

	// ExcludedBrokerTypes lists broker types to exclude from selection
	ExcludedBrokerTypes []messaging.BrokerType `json:"excluded_broker_types,omitempty"`

	// PerformanceRequirements specifies performance requirements
	PerformanceRequirements *PerformanceRequirements `json:"performance_requirements,omitempty"`

	// ConfigurationHints provide hints for configuration generation
	ConfigurationHints map[string]interface{} `json:"configuration_hints,omitempty"`
}

// CapabilityRequirement represents a required capability for broker selection.
type CapabilityRequirement struct {
	// Name is the capability name (e.g., "transactions", "ordering", "partitioning")
	Name string `json:"name"`

	// Required indicates if this capability is mandatory or preferred
	Required bool `json:"required"`

	// MinLevel specifies the minimum capability level required
	MinLevel string `json:"min_level,omitempty"`
}

// PerformanceRequirements specifies performance requirements for broker selection.
type PerformanceRequirements struct {
	// MinThroughput specifies minimum required throughput (messages/second)
	MinThroughput int64 `json:"min_throughput,omitempty"`

	// MaxLatency specifies maximum acceptable latency (milliseconds)
	MaxLatency int64 `json:"max_latency,omitempty"`

	// MaxMessageSize specifies maximum message size requirement (bytes)
	MaxMessageSize int64 `json:"max_message_size,omitempty"`

	// RequiresBatching indicates if batching capability is required
	RequiresBatching bool `json:"requires_batching,omitempty"`
}

// SelectionResult represents the result of broker selection.
type SelectionResult struct {
	// SelectedBrokerType is the recommended broker type
	SelectedBrokerType messaging.BrokerType `json:"selected_broker_type"`

	// SelectedAdapter is the adapter for the selected broker type
	SelectedAdapter messaging.MessageBrokerAdapter `json:"-"`

	// RecommendedConfig is a pre-configured broker configuration
	RecommendedConfig *messaging.BrokerConfig `json:"recommended_config"`

	// MatchScore indicates how well the selection matches the criteria (0-100)
	MatchScore int `json:"match_score"`

	// MatchDetails provides details about why this broker was selected
	MatchDetails []string `json:"match_details"`

	// Warnings contains any warnings about the selection
	Warnings []string `json:"warnings,omitempty"`
}

// SelectBroker selects the best broker type based on the given criteria.
func (s *BrokerSelector) SelectBroker(criteria SelectionCriteria) (*SelectionResult, error) {
	if s.registry == nil {
		return nil, messaging.NewConfigError("broker adapter registry not configured", nil)
	}

	adapters := s.registry.ListRegisteredAdapters()
	if len(adapters) == 0 {
		return nil, messaging.NewConfigError("no broker adapters registered", nil)
	}

	var candidates []brokerCandidate

	// Evaluate each adapter
	for _, adapterInfo := range adapters {
		adapter, err := s.registry.GetAdapter(adapterInfo.Name)
		if err != nil {
			continue // Skip adapters we can't retrieve
		}

		// Check each supported broker type
		for _, brokerType := range adapterInfo.SupportedBrokerTypes {
			candidate := brokerCandidate{
				BrokerType: brokerType,
				Adapter:    adapter,
				Score:      0,
				Details:    []string{},
				Warnings:   []string{},
			}

			// Evaluate the candidate
			s.evaluateCandidate(&candidate, criteria)

			// Only include candidates that meet requirements
			if candidate.Score > 0 {
				candidates = append(candidates, candidate)
			}
		}
	}

	if len(candidates) == 0 {
		return nil, messaging.NewConfigError("no brokers match the selection criteria", nil)
	}

	// Sort candidates by score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// Select the best candidate
	best := candidates[0]

	// Generate recommended configuration
	config := best.Adapter.GetDefaultConfiguration()
	config.Type = best.BrokerType

	// Apply configuration hints
	if criteria.ConfigurationHints != nil {
		s.applyConfigurationHints(config, criteria.ConfigurationHints)
	}

	return &SelectionResult{
		SelectedBrokerType: best.BrokerType,
		SelectedAdapter:    best.Adapter,
		RecommendedConfig:  config,
		MatchScore:         best.Score,
		MatchDetails:       best.Details,
		Warnings:           best.Warnings,
	}, nil
}

// brokerCandidate represents a candidate broker for selection.
type brokerCandidate struct {
	BrokerType messaging.BrokerType
	Adapter    messaging.MessageBrokerAdapter
	Score      int
	Details    []string
	Warnings   []string
}

// evaluateCandidate evaluates a broker candidate against the selection criteria.
func (s *BrokerSelector) evaluateCandidate(candidate *brokerCandidate, criteria SelectionCriteria) {
	capabilities := candidate.Adapter.GetCapabilities()

	// Check excluded types first
	for _, excluded := range criteria.ExcludedBrokerTypes {
		if candidate.BrokerType == excluded {
			candidate.Score = 0
			candidate.Details = append(candidate.Details, fmt.Sprintf("Excluded broker type: %s", candidate.BrokerType))
			return
		}
	}

	baseScore := 50 // Start with base score

	// Check preferred types
	for i, preferred := range criteria.PreferredBrokerTypes {
		if candidate.BrokerType == preferred {
			// Higher score for more preferred types
			bonus := (len(criteria.PreferredBrokerTypes) - i) * 10
			baseScore += bonus
			candidate.Details = append(candidate.Details, fmt.Sprintf("Preferred broker type (rank %d): %s", i+1, candidate.BrokerType))
			break
		}
	}

	// Check required capabilities
	capabilityScore := 0
	for _, requirement := range criteria.RequiredCapabilities {
		supported := s.checkCapabilitySupport(capabilities, requirement)
		if requirement.Required && !supported {
			// Required capability not supported - exclude
			candidate.Score = 0
			candidate.Details = append(candidate.Details, fmt.Sprintf("Missing required capability: %s", requirement.Name))
			return
		} else if supported {
			capabilityScore += 10
			candidate.Details = append(candidate.Details, fmt.Sprintf("Supports capability: %s", requirement.Name))
		}
	}

	// Check performance requirements
	perfScore := s.checkPerformanceRequirements(capabilities, criteria.PerformanceRequirements, candidate)

	candidate.Score = baseScore + capabilityScore + perfScore
}

// checkCapabilitySupport checks if a capability is supported.
func (s *BrokerSelector) checkCapabilitySupport(capabilities *messaging.BrokerCapabilities, requirement CapabilityRequirement) bool {
	switch strings.ToLower(requirement.Name) {
	case "transactions":
		return capabilities.SupportsTransactions
	case "ordering":
		return capabilities.SupportsOrdering
	case "partitioning":
		return capabilities.SupportsPartitioning
	case "dead_letter", "deadletter":
		return capabilities.SupportsDeadLetter
	case "delayed_delivery":
		return capabilities.SupportsDelayedDelivery
	case "priority":
		return capabilities.SupportsPriority
	case "streaming":
		return capabilities.SupportsStreaming
	case "seek":
		return capabilities.SupportsSeek
	case "consumer_groups":
		return capabilities.SupportsConsumerGroups
	default:
		// Check extended capabilities
		if capabilities.Extended != nil {
			if value, exists := capabilities.Extended[requirement.Name]; exists {
				if boolVal, ok := value.(bool); ok {
					return boolVal
				}
			}
		}
		return false
	}
}

// checkPerformanceRequirements evaluates performance requirements.
func (s *BrokerSelector) checkPerformanceRequirements(capabilities *messaging.BrokerCapabilities, requirements *PerformanceRequirements, candidate *brokerCandidate) int {
	if requirements == nil {
		return 0
	}

	score := 0

	// Check message size requirements
	if requirements.MaxMessageSize > 0 {
		if capabilities.MaxMessageSize > 0 && capabilities.MaxMessageSize < requirements.MaxMessageSize {
			candidate.Warnings = append(candidate.Warnings, fmt.Sprintf("Broker max message size (%d) is less than required (%d)", capabilities.MaxMessageSize, requirements.MaxMessageSize))
			score -= 10
		} else {
			score += 5
			candidate.Details = append(candidate.Details, "Meets message size requirements")
		}
	}

	// Check batching requirements
	if requirements.RequiresBatching {
		if capabilities.MaxBatchSize > 0 || capabilities.MaxBatchSize == 0 { // 0 means unlimited
			score += 5
			candidate.Details = append(candidate.Details, "Supports batching")
		} else {
			score -= 5
			candidate.Warnings = append(candidate.Warnings, "Limited or no batching support")
		}
	}

	return score
}

// applyConfigurationHints applies configuration hints to the broker configuration.
func (s *BrokerSelector) applyConfigurationHints(config *messaging.BrokerConfig, hints map[string]interface{}) {
	for key, value := range hints {
		switch key {
		case "endpoints":
			if endpoints, ok := value.([]string); ok {
				config.Endpoints = endpoints
			}
		case "timeout":
			// Handle timeout configuration (implementation depends on BrokerConfig structure)
		case "retry_attempts":
			// Handle retry configuration (implementation depends on BrokerConfig structure)
		default:
			// Store unknown hints in extended configuration if supported
			// This would require extending BrokerConfig to have an Extended field
		}
	}
}

// GetDefaultSelectionCriteria returns sensible default selection criteria.
func GetDefaultSelectionCriteria() SelectionCriteria {
	return SelectionCriteria{
		RequiredCapabilities: []CapabilityRequirement{
			{Name: "consumer_groups", Required: true},
		},
		PreferredBrokerTypes: []messaging.BrokerType{
			messaging.BrokerTypeKafka,    // Prefer Kafka for production
			messaging.BrokerTypeNATS,     // NATS for lightweight scenarios
			messaging.BrokerTypeRabbitMQ, // RabbitMQ for traditional messaging
			messaging.BrokerTypeInMemory, // In-memory for testing
		},
		PerformanceRequirements: &PerformanceRequirements{
			RequiresBatching: false, // Not required by default
		},
	}
}

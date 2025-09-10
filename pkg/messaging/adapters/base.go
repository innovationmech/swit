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
	"context"
	"fmt"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// BaseMessageBrokerAdapter provides common functionality for broker adapters.
// It implements shared logic that all adapters need, reducing code duplication
// and ensuring consistent behavior across different broker implementations.
//
// This base adapter provides:
// - Common validation patterns
// - Standard error handling
// - Consistent health check structure
// - Configuration template generation
// - Adapter metadata management
//
// Concrete adapters should embed this base adapter and override methods
// as needed for broker-specific behavior.
type BaseMessageBrokerAdapter struct {
	// info contains adapter metadata
	info *messaging.BrokerAdapterInfo

	// capabilities describes what features this adapter supports
	capabilities *messaging.BrokerCapabilities

	// defaultConfig provides a template configuration
	defaultConfig *messaging.BrokerConfig
}

// NewBaseMessageBrokerAdapter creates a new base adapter with the provided metadata.
// This is typically called by concrete adapter implementations during initialization.
//
// Parameters:
//   - info: Adapter metadata including name, version, and supported types
//   - capabilities: Capabilities supported by this adapter
//   - defaultConfig: Default configuration template for this adapter
//
// Returns:
//   - *BaseMessageBrokerAdapter: Configured base adapter ready for use
func NewBaseMessageBrokerAdapter(
	info *messaging.BrokerAdapterInfo,
	capabilities *messaging.BrokerCapabilities,
	defaultConfig *messaging.BrokerConfig,
) *BaseMessageBrokerAdapter {
	return &BaseMessageBrokerAdapter{
		info:          info,
		capabilities:  capabilities,
		defaultConfig: defaultConfig,
	}
}

// GetAdapterInfo implements MessageBrokerAdapter interface.
func (b *BaseMessageBrokerAdapter) GetAdapterInfo() *messaging.BrokerAdapterInfo {
	if b.info == nil {
		return &messaging.BrokerAdapterInfo{
			Name:        "unknown",
			Version:     "0.0.0",
			Description: "Base adapter without metadata",
		}
	}

	// Return a copy to prevent external mutation
	info := *b.info
	if b.info.Extended != nil {
		info.Extended = make(map[string]any)
		for k, v := range b.info.Extended {
			info.Extended[k] = v
		}
	}
	if b.info.SupportedBrokerTypes != nil {
		info.SupportedBrokerTypes = make([]messaging.BrokerType, len(b.info.SupportedBrokerTypes))
		copy(info.SupportedBrokerTypes, b.info.SupportedBrokerTypes)
	}

	return &info
}

// CreateBroker implements MessageBrokerAdapter interface.
// This base implementation provides common validation and error handling.
// Concrete adapters should override this method to create broker instances.
func (b *BaseMessageBrokerAdapter) CreateBroker(config *messaging.BrokerConfig) (messaging.MessageBroker, error) {
	if config == nil {
		return nil, messaging.NewConfigError("broker configuration cannot be nil", nil)
	}

	// Validate configuration using the adapter's validation logic
	result := b.ValidateConfiguration(config)
	if !result.Valid {
		return nil, b.createValidationError("configuration validation failed", result)
	}

	// Check if broker type is supported by this adapter
	if !b.isTypeSupported(config.Type) {
		return nil, messaging.NewConfigError(
			fmt.Sprintf("broker type '%s' not supported by adapter '%s'", config.Type, b.getAdapterName()),
			nil,
		)
	}

	// This base implementation cannot create brokers - concrete adapters must override
	return nil, messaging.NewConfigError(
		fmt.Sprintf("adapter '%s' does not implement broker creation", b.getAdapterName()),
		nil,
	)
}

// ValidateConfiguration implements MessageBrokerAdapter interface.
// This base implementation provides common validation patterns.
func (b *BaseMessageBrokerAdapter) ValidateConfiguration(config *messaging.BrokerConfig) *messaging.AdapterValidationResult {
	result := &messaging.AdapterValidationResult{
		Valid:       true,
		Errors:      []messaging.AdapterValidationError{},
		Warnings:    []messaging.AdapterValidationWarning{},
		Suggestions: []messaging.AdapterValidationSuggestion{},
	}

	// Validate basic configuration requirements
	if config == nil {
		b.addValidationError(result, "", "configuration cannot be nil", "CONFIG_NULL", messaging.AdapterValidationSeverityError)
		return result
	}

	// Validate broker type
	if config.Type == "" {
		b.addValidationError(result, "Type", "broker type cannot be empty", "TYPE_EMPTY", messaging.AdapterValidationSeverityError)
	} else if !config.Type.IsValid() {
		b.addValidationError(result, "Type", fmt.Sprintf("invalid broker type: %s", config.Type), "TYPE_INVALID", messaging.AdapterValidationSeverityError)
	} else if !b.isTypeSupported(config.Type) {
		b.addValidationError(result, "Type", fmt.Sprintf("broker type '%s' not supported by adapter '%s'", config.Type, b.getAdapterName()), "TYPE_UNSUPPORTED", messaging.AdapterValidationSeverityError)
	}

	// Validate endpoints
	if len(config.Endpoints) == 0 {
		b.addValidationError(result, "Endpoints", "at least one endpoint must be specified", "ENDPOINTS_EMPTY", messaging.AdapterValidationSeverityError)
	} else {
		for i, endpoint := range config.Endpoints {
			if endpoint == "" {
				b.addValidationError(result, fmt.Sprintf("Endpoints[%d]", i), "endpoint cannot be empty", "ENDPOINT_EMPTY", messaging.AdapterValidationSeverityError)
			}
		}
	}

	// Note: We skip the configuration's own validation method in the base adapter
	// since it may require fields that are broker-specific. Concrete adapters
	// can override this method to include more detailed validation.

	return result
}

// GetCapabilities implements MessageBrokerAdapter interface.
func (b *BaseMessageBrokerAdapter) GetCapabilities() *messaging.BrokerCapabilities {
	if b.capabilities == nil {
		return &messaging.BrokerCapabilities{
			// Default capabilities for base adapter (minimal)
			SupportsTransactions:        false,
			SupportsOrdering:            false,
			SupportsPartitioning:        false,
			SupportsDeadLetter:          false,
			SupportsDelayedDelivery:     false,
			SupportsPriority:            false,
			SupportsStreaming:           false,
			SupportsSeek:                false,
			SupportsConsumerGroups:      true, // Most brokers support this
			MaxMessageSize:              0,    // Unlimited by default
			MaxBatchSize:                0,    // Unlimited by default
			MaxTopicNameLength:          0,    // Unlimited by default
			SupportedCompressionTypes:   []messaging.CompressionType{messaging.CompressionNone},
			SupportedSerializationTypes: []messaging.SerializationType{messaging.SerializationJSON},
		}
	}

	// Return a copy to prevent external mutation
	caps := *b.capabilities
	if b.capabilities.Extended != nil {
		caps.Extended = make(map[string]any)
		for k, v := range b.capabilities.Extended {
			caps.Extended[k] = v
		}
	}
	if b.capabilities.SupportedCompressionTypes != nil {
		caps.SupportedCompressionTypes = make([]messaging.CompressionType, len(b.capabilities.SupportedCompressionTypes))
		copy(caps.SupportedCompressionTypes, b.capabilities.SupportedCompressionTypes)
	}
	if b.capabilities.SupportedSerializationTypes != nil {
		caps.SupportedSerializationTypes = make([]messaging.SerializationType, len(b.capabilities.SupportedSerializationTypes))
		copy(caps.SupportedSerializationTypes, b.capabilities.SupportedSerializationTypes)
	}

	return &caps
}

// GetDefaultConfiguration implements MessageBrokerAdapter interface.
func (b *BaseMessageBrokerAdapter) GetDefaultConfiguration() *messaging.BrokerConfig {
	if b.defaultConfig == nil {
		return &messaging.BrokerConfig{
			Type:      messaging.BrokerTypeInMemory, // Safe default
			Endpoints: []string{"localhost:9092"},   // Common default
		}
	}

	// Return a copy to prevent external mutation
	config := *b.defaultConfig
	if b.defaultConfig.Endpoints != nil {
		config.Endpoints = make([]string, len(b.defaultConfig.Endpoints))
		copy(config.Endpoints, b.defaultConfig.Endpoints)
	}

	return &config
}

// HealthCheck implements MessageBrokerAdapter interface.
func (b *BaseMessageBrokerAdapter) HealthCheck(ctx context.Context) (*messaging.HealthStatus, error) {
	startTime := time.Now()

	status := &messaging.HealthStatus{
		Status:       messaging.HealthStatusHealthy,
		Message:      fmt.Sprintf("Adapter '%s' is functioning normally", b.getAdapterName()),
		LastChecked:  startTime,
		ResponseTime: time.Since(startTime),
		Details:      make(map[string]any),
	}

	// Add adapter information to health check details
	status.Details["adapter_name"] = b.getAdapterName()
	status.Details["adapter_version"] = b.getAdapterVersion()
	status.Details["supported_types"] = b.getSupportedTypeStrings()

	// Check if adapter has required metadata
	if b.info == nil {
		status.Status = messaging.HealthStatusDegraded
		status.Message = "Adapter metadata is missing"
		status.Details["issues"] = []string{"missing adapter info"}
	}

	status.ResponseTime = time.Since(startTime)
	return status, nil
}

// Helper methods for internal use

// getAdapterName returns the adapter name, or a default if not set.
func (b *BaseMessageBrokerAdapter) getAdapterName() string {
	if b.info != nil && b.info.Name != "" {
		return b.info.Name
	}
	return "unknown-adapter"
}

// getAdapterVersion returns the adapter version, or a default if not set.
func (b *BaseMessageBrokerAdapter) getAdapterVersion() string {
	if b.info != nil && b.info.Version != "" {
		return b.info.Version
	}
	return "0.0.0"
}

// isTypeSupported checks if the adapter supports the given broker type.
func (b *BaseMessageBrokerAdapter) isTypeSupported(brokerType messaging.BrokerType) bool {
	if b.info == nil || b.info.SupportedBrokerTypes == nil {
		return false
	}

	for _, supportedType := range b.info.SupportedBrokerTypes {
		if supportedType == brokerType {
			return true
		}
	}
	return false
}

// getSupportedTypeStrings returns supported types as string slice for reporting.
func (b *BaseMessageBrokerAdapter) getSupportedTypeStrings() []string {
	if b.info == nil || b.info.SupportedBrokerTypes == nil {
		return []string{}
	}

	result := make([]string, len(b.info.SupportedBrokerTypes))
	for i, brokerType := range b.info.SupportedBrokerTypes {
		result[i] = string(brokerType)
	}
	return result
}

// addValidationError adds a validation error to the result.
func (b *BaseMessageBrokerAdapter) addValidationError(
	result *messaging.AdapterValidationResult,
	field, message, code string,
	severity messaging.AdapterValidationSeverity,
) {
	result.Errors = append(result.Errors, messaging.AdapterValidationError{
		Field:    field,
		Message:  message,
		Code:     code,
		Severity: severity,
	})
	result.Valid = false
}

// addValidationWarning adds a validation warning to the result.
func (b *BaseMessageBrokerAdapter) addValidationWarning(
	result *messaging.AdapterValidationResult,
	field, message, code string,
) {
	result.Warnings = append(result.Warnings, messaging.AdapterValidationWarning{
		Field:   field,
		Message: message,
		Code:    code,
	})
}

// addValidationSuggestion adds a validation suggestion to the result.
func (b *BaseMessageBrokerAdapter) addValidationSuggestion(
	result *messaging.AdapterValidationResult,
	field, message string,
	suggestedValue any,
) {
	result.Suggestions = append(result.Suggestions, messaging.AdapterValidationSuggestion{
		Field:          field,
		Message:        message,
		SuggestedValue: suggestedValue,
	})
}

// createValidationError creates a configuration error from validation results.
func (b *BaseMessageBrokerAdapter) createValidationError(message string, result *messaging.AdapterValidationResult) error {
	if result.Valid {
		return nil
	}

	var errorMessages []string
	for _, err := range result.Errors {
		if err.Field != "" {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", err.Field, err.Message))
		} else {
			errorMessages = append(errorMessages, err.Message)
		}
	}

	combinedMessage := message
	if len(errorMessages) > 0 {
		combinedMessage = fmt.Sprintf("%s: %v", message, errorMessages)
	}

	return messaging.NewConfigError(combinedMessage, nil)
}

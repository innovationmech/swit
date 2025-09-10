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
	"sort"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// BrokerAdapterRegistry provides centralized management of message broker adapters.
// It enables pluggable broker implementations through a registry pattern,
// allowing runtime discovery and instantiation of different broker types.
//
// The registry maintains thread-safe access to adapters and provides:
// - Adapter registration and discovery
// - Configuration validation across adapters
// - Broker creation with automatic adapter selection
// - Health monitoring of registered adapters
//
// This design follows the SWIT framework patterns for extensibility and consistency.
type BrokerAdapterRegistry interface {
	// RegisterAdapter registers a broker adapter for specific broker types.
	// This enables the factory to create brokers using the adapter pattern.
	//
	// Parameters:
	//   - adapter: The adapter implementation to register
	//
	// Returns:
	//   - error: RegistrationError if adapter is invalid or conflicts with existing registrations
	RegisterAdapter(adapter messaging.MessageBrokerAdapter) error

	// UnregisterAdapter removes an adapter from the registry.
	// Any broker types exclusively supported by this adapter will become unavailable.
	//
	// Parameters:
	//   - adapterName: Name of the adapter to unregister
	//
	// Returns:
	//   - error: RegistrationError if adapter not found
	UnregisterAdapter(adapterName string) error

	// GetAdapter retrieves a registered adapter by name.
	//
	// Parameters:
	//   - adapterName: Name of the adapter to retrieve
	//
	// Returns:
	//   - MessageBrokerAdapter: The requested adapter, nil if not found
	//   - error: RegistrationError if adapter not found
	GetAdapter(adapterName string) (messaging.MessageBrokerAdapter, error)

	// GetAdapterForBrokerType finds the best adapter for a specific broker type.
	// If multiple adapters support the same type, selection is deterministic.
	//
	// Parameters:
	//   - brokerType: The broker type to find an adapter for
	//
	// Returns:
	//   - MessageBrokerAdapter: The selected adapter for the broker type
	//   - error: UnsupportedTypeError if no adapter supports the type
	GetAdapterForBrokerType(brokerType messaging.BrokerType) (messaging.MessageBrokerAdapter, error)

	// ListRegisteredAdapters returns information about all registered adapters.
	//
	// Returns:
	//   - []*BrokerAdapterInfo: List of adapter information, sorted by name
	ListRegisteredAdapters() []*messaging.BrokerAdapterInfo

	// GetSupportedBrokerTypes returns all broker types supported by registered adapters.
	//
	// Returns:
	//   - []BrokerType: List of supported broker types, sorted alphabetically
	GetSupportedBrokerTypes() []messaging.BrokerType

	// ValidateConfiguration validates a broker configuration using the appropriate adapter.
	//
	// Parameters:
	//   - config: Configuration to validate
	//
	// Returns:
	//   - *AdapterValidationResult: Validation result from the appropriate adapter
	//   - error: UnsupportedTypeError if no adapter supports the config's broker type
	ValidateConfiguration(config *messaging.BrokerConfig) (*messaging.AdapterValidationResult, error)

	// CreateBroker creates a broker instance using the appropriate adapter.
	//
	// Parameters:
	//   - config: Broker configuration
	//
	// Returns:
	//   - MessageBroker: Created broker instance
	//   - error: Configuration, creation, or unsupported type error
	CreateBroker(config *messaging.BrokerConfig) (messaging.MessageBroker, error)

	// HealthCheck performs health checks on all registered adapters.
	//
	// Parameters:
	//   - ctx: Context for timeout control
	//
	// Returns:
	//   - map[string]*HealthStatus: Health status for each adapter, keyed by adapter name
	//   - error: HealthCheckError if any critical issues are found
	HealthCheck(ctx context.Context) (map[string]*messaging.HealthStatus, error)
}

// brokerAdapterRegistryImpl implements the BrokerAdapterRegistry interface.
type brokerAdapterRegistryImpl struct {
	// adapters maps adapter names to their instances
	adapters map[string]messaging.MessageBrokerAdapter

	// typeToAdapter maps broker types to their preferred adapter names
	// When multiple adapters support the same type, the first registered wins
	typeToAdapter map[messaging.BrokerType]string

	// mu protects concurrent access to the registry
	mu sync.RWMutex
}

// defaultAdapterRegistry is the default registry instance.
var defaultAdapterRegistry = &brokerAdapterRegistryImpl{
	adapters:      make(map[string]messaging.MessageBrokerAdapter),
	typeToAdapter: make(map[messaging.BrokerType]string),
}

// NewBrokerAdapterRegistry creates a new adapter registry instance.
// This allows creating custom registries for testing or specialized use cases.
func NewBrokerAdapterRegistry() BrokerAdapterRegistry {
	return &brokerAdapterRegistryImpl{
		adapters:      make(map[string]messaging.MessageBrokerAdapter),
		typeToAdapter: make(map[messaging.BrokerType]string),
	}
}

// GetDefaultAdapterRegistry returns the default adapter registry instance.
func GetDefaultAdapterRegistry() BrokerAdapterRegistry {
	return defaultAdapterRegistry
}

// RegisterAdapter implements BrokerAdapterRegistry interface.
func (r *brokerAdapterRegistryImpl) RegisterAdapter(adapter messaging.MessageBrokerAdapter) error {
	if adapter == nil {
		return messaging.NewConfigError("adapter cannot be nil", nil)
	}

	info := adapter.GetAdapterInfo()
	if info == nil {
		return messaging.NewConfigError("adapter info cannot be nil", nil)
	}

	if info.Name == "" {
		return messaging.NewConfigError("adapter name cannot be empty", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate adapter names
	if _, exists := r.adapters[info.Name]; exists {
		return messaging.NewConfigError(
			fmt.Sprintf("adapter with name '%s' already registered", info.Name),
			nil,
		)
	}

	// Register the adapter
	r.adapters[info.Name] = adapter

	// Register broker type mappings (first adapter wins for each type)
	for _, brokerType := range info.SupportedBrokerTypes {
		if _, exists := r.typeToAdapter[brokerType]; !exists {
			r.typeToAdapter[brokerType] = info.Name
		}
	}

	return nil
}

// UnregisterAdapter implements BrokerAdapterRegistry interface.
func (r *brokerAdapterRegistryImpl) UnregisterAdapter(adapterName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	adapter, exists := r.adapters[adapterName]
	if !exists {
		return messaging.NewConfigError(
			fmt.Sprintf("adapter '%s' not found", adapterName),
			nil,
		)
	}

	// Remove adapter
	delete(r.adapters, adapterName)

	// Remove type mappings for this adapter
	info := adapter.GetAdapterInfo()
	if info != nil {
		for _, brokerType := range info.SupportedBrokerTypes {
			if r.typeToAdapter[brokerType] == adapterName {
				delete(r.typeToAdapter, brokerType)
			}
		}
	}

	return nil
}

// GetAdapter implements BrokerAdapterRegistry interface.
func (r *brokerAdapterRegistryImpl) GetAdapter(adapterName string) (messaging.MessageBrokerAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[adapterName]
	if !exists {
		return nil, messaging.NewConfigError(
			fmt.Sprintf("adapter '%s' not found", adapterName),
			nil,
		)
	}

	return adapter, nil
}

// GetAdapterForBrokerType implements BrokerAdapterRegistry interface.
func (r *brokerAdapterRegistryImpl) GetAdapterForBrokerType(brokerType messaging.BrokerType) (messaging.MessageBrokerAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapterName, exists := r.typeToAdapter[brokerType]
	if !exists {
		return nil, messaging.NewConfigError(
			fmt.Sprintf("no adapter registered for broker type '%s'", brokerType),
			nil,
		)
	}

	adapter, exists := r.adapters[adapterName]
	if !exists {
		// This should not happen, but handle gracefully
		return nil, messaging.NewConfigError(
			fmt.Sprintf("adapter '%s' not found for broker type '%s'", adapterName, brokerType),
			nil,
		)
	}

	return adapter, nil
}

// ListRegisteredAdapters implements BrokerAdapterRegistry interface.
func (r *brokerAdapterRegistryImpl) ListRegisteredAdapters() []*messaging.BrokerAdapterInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapters := make([]*messaging.BrokerAdapterInfo, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		adapters = append(adapters, adapter.GetAdapterInfo())
	}

	// Sort by adapter name for consistent ordering
	sort.Slice(adapters, func(i, j int) bool {
		return adapters[i].Name < adapters[j].Name
	})

	return adapters
}

// GetSupportedBrokerTypes implements BrokerAdapterRegistry interface.
func (r *brokerAdapterRegistryImpl) GetSupportedBrokerTypes() []messaging.BrokerType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]messaging.BrokerType, 0, len(r.typeToAdapter))
	for brokerType := range r.typeToAdapter {
		types = append(types, brokerType)
	}

	// Sort for consistent ordering
	sort.Slice(types, func(i, j int) bool {
		return string(types[i]) < string(types[j])
	})

	return types
}

// ValidateConfiguration implements BrokerAdapterRegistry interface.
func (r *brokerAdapterRegistryImpl) ValidateConfiguration(config *messaging.BrokerConfig) (*messaging.AdapterValidationResult, error) {
	if config == nil {
		return &messaging.AdapterValidationResult{
			Valid: false,
			Errors: []messaging.AdapterValidationError{{
				Field:    "",
				Message:  "configuration cannot be nil",
				Code:     "CONFIG_NULL",
				Severity: messaging.AdapterValidationSeverityError,
			}},
		}, nil
	}

	adapter, err := r.GetAdapterForBrokerType(config.Type)
	if err != nil {
		return &messaging.AdapterValidationResult{
			Valid: false,
			Errors: []messaging.AdapterValidationError{{
				Field:    "Type",
				Message:  fmt.Sprintf("unsupported broker type: %s", config.Type),
				Code:     "TYPE_UNSUPPORTED",
				Severity: messaging.AdapterValidationSeverityError,
			}},
		}, nil
	}

	return adapter.ValidateConfiguration(config), nil
}

// CreateBroker implements BrokerAdapterRegistry interface.
func (r *brokerAdapterRegistryImpl) CreateBroker(config *messaging.BrokerConfig) (messaging.MessageBroker, error) {
	if config == nil {
		return nil, messaging.NewConfigError("broker configuration cannot be nil", nil)
	}

	adapter, err := r.GetAdapterForBrokerType(config.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to find adapter for broker type '%s': %w", config.Type, err)
	}

	broker, err := adapter.CreateBroker(config)
	if err != nil {
		return nil, fmt.Errorf("adapter '%s' failed to create broker: %w", adapter.GetAdapterInfo().Name, err)
	}

	return broker, nil
}

// HealthCheck implements BrokerAdapterRegistry interface.
func (r *brokerAdapterRegistryImpl) HealthCheck(ctx context.Context) (map[string]*messaging.HealthStatus, error) {
	r.mu.RLock()
	adapters := make(map[string]messaging.MessageBrokerAdapter, len(r.adapters))
	for name, adapter := range r.adapters {
		adapters[name] = adapter
	}
	r.mu.RUnlock()

	results := make(map[string]*messaging.HealthStatus)
	var criticalIssues []string

	for name, adapter := range adapters {
		status, err := adapter.HealthCheck(ctx)
		if err != nil {
			// Create error status if health check failed
			status = &messaging.HealthStatus{
				Status:      messaging.HealthStatusUnhealthy,
				Message:     fmt.Sprintf("Health check failed: %v", err),
				LastChecked: time.Now(),
			}
			criticalIssues = append(criticalIssues, fmt.Sprintf("adapter '%s': %v", name, err))
		}
		results[name] = status

		// Track critical issues
		if status.Status == messaging.HealthStatusUnhealthy {
			criticalIssues = append(criticalIssues, fmt.Sprintf("adapter '%s': %s", name, status.Message))
		}
	}

	var err error
	if len(criticalIssues) > 0 {
		err = messaging.NewConfigError(
			fmt.Sprintf("critical adapter health issues: %v", criticalIssues),
			nil,
		)
	}

	return results, err
}

// Convenience functions for global registry access

// RegisterGlobalAdapter registers an adapter with the default registry.
func RegisterGlobalAdapter(adapter messaging.MessageBrokerAdapter) error {
	return defaultAdapterRegistry.RegisterAdapter(adapter)
}

// GetGlobalAdapter retrieves an adapter from the default registry.
func GetGlobalAdapter(adapterName string) (messaging.MessageBrokerAdapter, error) {
	return defaultAdapterRegistry.GetAdapter(adapterName)
}

// CreateBrokerWithAdapter creates a broker using the default adapter registry.
func CreateBrokerWithAdapter(config *messaging.BrokerConfig) (messaging.MessageBroker, error) {
	return defaultAdapterRegistry.CreateBroker(config)
}

// ListGlobalAdapters lists all adapters in the default registry.
func ListGlobalAdapters() []*messaging.BrokerAdapterInfo {
	return defaultAdapterRegistry.ListRegisteredAdapters()
}

// GetGlobalSupportedBrokerTypes returns supported types from the default registry.
func GetGlobalSupportedBrokerTypes() []messaging.BrokerType {
	return defaultAdapterRegistry.GetSupportedBrokerTypes()
}

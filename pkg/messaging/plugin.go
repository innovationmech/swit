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
	"sync"
	"time"
)

// PluginMetadata contains information about a middleware plugin.
type PluginMetadata struct {
	// Name is the unique identifier for the plugin
	Name string `json:"name"`

	// Version is the plugin version (semantic versioning recommended)
	Version string `json:"version"`

	// Description provides a brief description of the plugin's functionality
	Description string `json:"description"`

	// Author is the plugin author or organization
	Author string `json:"author"`

	// Dependencies lists required plugins (name:version format)
	Dependencies []string `json:"dependencies,omitempty"`

	// Tags are optional labels for categorization and search
	Tags []string `json:"tags,omitempty"`

	// CreatedAt is the plugin creation timestamp
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// PluginState represents the current state of a plugin in its lifecycle.
type PluginState int

const (
	// PluginStateUninitialized indicates the plugin has been loaded but not initialized
	PluginStateUninitialized PluginState = iota

	// PluginStateInitialized indicates Initialize() has completed successfully
	PluginStateInitialized

	// PluginStateStarted indicates Start() has completed successfully
	PluginStateStarted

	// PluginStateStopped indicates Stop() has been called
	PluginStateStopped

	// PluginStateFailed indicates the plugin encountered an error
	PluginStateFailed
)

// String returns the string representation of PluginState.
func (ps PluginState) String() string {
	switch ps {
	case PluginStateUninitialized:
		return "uninitialized"
	case PluginStateInitialized:
		return "initialized"
	case PluginStateStarted:
		return "started"
	case PluginStateStopped:
		return "stopped"
	case PluginStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Plugin defines the interface that all middleware plugins must implement.
// A plugin combines middleware functionality with lifecycle management.
type Plugin interface {
	// Metadata returns the plugin's metadata information
	Metadata() PluginMetadata

	// Initialize prepares the plugin for operation
	// This is called once when the plugin is loaded
	Initialize(ctx context.Context, config map[string]interface{}) error

	// Start begins the plugin's operation
	// This is called after Initialize and may be called again after Stop
	Start(ctx context.Context) error

	// Stop halts the plugin's operation gracefully
	// Resources should be released but the plugin should remain in a restartable state
	Stop(ctx context.Context) error

	// Shutdown performs final cleanup before the plugin is unloaded
	// This is called when the plugin is being permanently removed
	Shutdown(ctx context.Context) error

	// CreateMiddleware creates a middleware instance from this plugin
	// This can be called multiple times to create multiple instances
	CreateMiddleware() (Middleware, error)

	// State returns the current lifecycle state of the plugin
	State() PluginState

	// HealthCheck performs a health check on the plugin
	// Returns nil if healthy, error otherwise
	HealthCheck(ctx context.Context) error
}

// BasePlugin provides a base implementation of Plugin interface with lifecycle management.
// Plugin developers can embed this to get default lifecycle behavior.
type BasePlugin struct {
	metadata PluginMetadata
	state    PluginState
	mutex    sync.RWMutex
	config   map[string]interface{}
}

// NewBasePlugin creates a new BasePlugin with the given metadata.
func NewBasePlugin(metadata PluginMetadata) *BasePlugin {
	return &BasePlugin{
		metadata: metadata,
		state:    PluginStateUninitialized,
		config:   make(map[string]interface{}),
	}
}

// Metadata returns the plugin's metadata.
func (bp *BasePlugin) Metadata() PluginMetadata {
	return bp.metadata
}

// Initialize initializes the plugin with the given configuration.
func (bp *BasePlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	if bp.state != PluginStateUninitialized && bp.state != PluginStateStopped {
		return fmt.Errorf("plugin %s cannot be initialized from state %s", bp.metadata.Name, bp.state)
	}

	bp.config = config
	bp.state = PluginStateInitialized
	return nil
}

// Start starts the plugin.
func (bp *BasePlugin) Start(ctx context.Context) error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	if bp.state != PluginStateInitialized && bp.state != PluginStateStopped {
		return fmt.Errorf("plugin %s cannot be started from state %s", bp.metadata.Name, bp.state)
	}

	bp.state = PluginStateStarted
	return nil
}

// Stop stops the plugin.
func (bp *BasePlugin) Stop(ctx context.Context) error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	if bp.state != PluginStateStarted {
		return fmt.Errorf("plugin %s cannot be stopped from state %s", bp.metadata.Name, bp.state)
	}

	bp.state = PluginStateStopped
	return nil
}

// Shutdown shuts down the plugin.
func (bp *BasePlugin) Shutdown(ctx context.Context) error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	bp.state = PluginStateStopped
	bp.config = nil
	return nil
}

// CreateMiddleware is a placeholder that should be overridden by concrete plugins.
func (bp *BasePlugin) CreateMiddleware() (Middleware, error) {
	return nil, fmt.Errorf("CreateMiddleware not implemented for plugin %s", bp.metadata.Name)
}

// State returns the current state of the plugin.
func (bp *BasePlugin) State() PluginState {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	return bp.state
}

// HealthCheck performs a basic health check.
func (bp *BasePlugin) HealthCheck(ctx context.Context) error {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()

	if bp.state == PluginStateFailed {
		return fmt.Errorf("plugin %s is in failed state", bp.metadata.Name)
	}

	if bp.state != PluginStateStarted {
		return fmt.Errorf("plugin %s is not started (current state: %s)", bp.metadata.Name, bp.state)
	}

	return nil
}

// GetConfig returns a copy of the plugin's configuration.
func (bp *BasePlugin) GetConfig() map[string]interface{} {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()

	configCopy := make(map[string]interface{}, len(bp.config))
	for k, v := range bp.config {
		configCopy[k] = v
	}
	return configCopy
}

// SetState sets the plugin state (protected method for plugin implementations).
func (bp *BasePlugin) SetState(state PluginState) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	bp.state = state
}

// PluginFactory is a function that creates a new plugin instance.
type PluginFactory func() (Plugin, error)

// PluginRegistry manages plugin registration and lifecycle.
type PluginRegistry struct {
	plugins   map[string]Plugin
	factories map[string]PluginFactory
	mutex     sync.RWMutex
}

// NewPluginRegistry creates a new plugin registry.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins:   make(map[string]Plugin),
		factories: make(map[string]PluginFactory),
	}
}

// RegisterPlugin registers a plugin instance.
func (pr *PluginRegistry) RegisterPlugin(plugin Plugin) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	name := plugin.Metadata().Name
	if _, exists := pr.plugins[name]; exists {
		return fmt.Errorf("plugin already registered: %s", name)
	}

	pr.plugins[name] = plugin
	return nil
}

// RegisterFactory registers a plugin factory.
func (pr *PluginRegistry) RegisterFactory(name string, factory PluginFactory) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if _, exists := pr.factories[name]; exists {
		return fmt.Errorf("plugin factory already registered: %s", name)
	}

	pr.factories[name] = factory
	return nil
}

// GetPlugin retrieves a registered plugin by name.
func (pr *PluginRegistry) GetPlugin(name string) (Plugin, error) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	plugin, exists := pr.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	return plugin, nil
}

// CreatePlugin creates a plugin instance using a registered factory.
func (pr *PluginRegistry) CreatePlugin(name string) (Plugin, error) {
	pr.mutex.RLock()
	factory, exists := pr.factories[name]
	pr.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("plugin factory not found: %s", name)
	}

	return factory()
}

// ListPlugins returns all registered plugin names.
func (pr *PluginRegistry) ListPlugins() []string {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	names := make([]string, 0, len(pr.plugins))
	for name := range pr.plugins {
		names = append(names, name)
	}
	return names
}

// ListFactories returns all registered plugin factory names.
func (pr *PluginRegistry) ListFactories() []string {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	names := make([]string, 0, len(pr.factories))
	for name := range pr.factories {
		names = append(names, name)
	}
	return names
}

// UnregisterPlugin removes a plugin from the registry.
// It will attempt to stop and shutdown the plugin before removing it.
func (pr *PluginRegistry) UnregisterPlugin(ctx context.Context, name string) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	plugin, exists := pr.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	// Try to stop and shutdown the plugin gracefully
	if plugin.State() == PluginStateStarted {
		if err := plugin.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop plugin %s: %w", name, err)
		}
	}

	if err := plugin.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown plugin %s: %w", name, err)
	}

	delete(pr.plugins, name)
	return nil
}

// UnregisterFactory removes a plugin factory from the registry.
func (pr *PluginRegistry) UnregisterFactory(name string) bool {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if _, exists := pr.factories[name]; exists {
		delete(pr.factories, name)
		return true
	}
	return false
}

// InitializeAll initializes all registered plugins.
func (pr *PluginRegistry) InitializeAll(ctx context.Context, configs map[string]map[string]interface{}) error {
	pr.mutex.RLock()
	plugins := make([]Plugin, 0, len(pr.plugins))
	for _, plugin := range pr.plugins {
		plugins = append(plugins, plugin)
	}
	pr.mutex.RUnlock()

	for _, plugin := range plugins {
		name := plugin.Metadata().Name
		config := configs[name]
		if config == nil {
			config = make(map[string]interface{})
		}

		if err := plugin.Initialize(ctx, config); err != nil {
			return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
		}
	}

	return nil
}

// StartAll starts all initialized plugins.
func (pr *PluginRegistry) StartAll(ctx context.Context) error {
	pr.mutex.RLock()
	plugins := make([]Plugin, 0, len(pr.plugins))
	for _, plugin := range pr.plugins {
		plugins = append(plugins, plugin)
	}
	pr.mutex.RUnlock()

	for _, plugin := range plugins {
		if plugin.State() == PluginStateInitialized || plugin.State() == PluginStateStopped {
			if err := plugin.Start(ctx); err != nil {
				return fmt.Errorf("failed to start plugin %s: %w", plugin.Metadata().Name, err)
			}
		}
	}

	return nil
}

// StopAll stops all started plugins.
func (pr *PluginRegistry) StopAll(ctx context.Context) error {
	pr.mutex.RLock()
	plugins := make([]Plugin, 0, len(pr.plugins))
	for _, plugin := range pr.plugins {
		plugins = append(plugins, plugin)
	}
	pr.mutex.RUnlock()

	var errs []error
	for _, plugin := range plugins {
		if plugin.State() == PluginStateStarted {
			if err := plugin.Stop(ctx); err != nil {
				errs = append(errs, fmt.Errorf("failed to stop plugin %s: %w", plugin.Metadata().Name, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping plugins: %v", errs)
	}
	return nil
}

// ShutdownAll shuts down all plugins.
func (pr *PluginRegistry) ShutdownAll(ctx context.Context) error {
	pr.mutex.RLock()
	plugins := make([]Plugin, 0, len(pr.plugins))
	for _, plugin := range pr.plugins {
		plugins = append(plugins, plugin)
	}
	pr.mutex.RUnlock()

	var errs []error
	for _, plugin := range plugins {
		if err := plugin.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown plugin %s: %w", plugin.Metadata().Name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors shutting down plugins: %v", errs)
	}
	return nil
}

// HealthCheckAll performs health checks on all plugins.
func (pr *PluginRegistry) HealthCheckAll(ctx context.Context) map[string]error {
	pr.mutex.RLock()
	plugins := make([]Plugin, 0, len(pr.plugins))
	for _, plugin := range pr.plugins {
		plugins = append(plugins, plugin)
	}
	pr.mutex.RUnlock()

	results := make(map[string]error)
	for _, plugin := range plugins {
		name := plugin.Metadata().Name
		results[name] = plugin.HealthCheck(ctx)
	}

	return results
}

// GlobalPluginRegistry provides a global instance for plugin management.
var GlobalPluginRegistry = NewPluginRegistry()

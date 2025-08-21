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

// Package plugin provides plugin system implementation for switctl.
package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// DefaultPluginManager implements the PluginManager interface.
type DefaultPluginManager struct {
	plugins     map[string]interfaces.Plugin
	registry    *PluginRegistry
	loader      PluginLoader
	pluginDirs  []string
	config      *ManagerConfig
	logger      interfaces.Logger
	mu          sync.RWMutex
	initialized bool
}

// NewPluginManager creates a new plugin manager instance.
func NewPluginManager(config *ManagerConfig, logger interfaces.Logger) *DefaultPluginManager {
	if config == nil {
		config = &ManagerConfig{
			PluginDirs:          []string{"plugins"},
			LoadTimeout:         30 * time.Second,
			EnableAutoDiscovery: true,
			EnableVersionCheck:  true,
			MaxPlugins:          50,
			IsolationMode:       "sandbox",
			SecurityPolicy: SecurityPolicy{
				AllowUnsigned:   false,
				RequireChecksum: true,
			},
		}
	}

	return &DefaultPluginManager{
		plugins:    make(map[string]interfaces.Plugin),
		registry:   NewPluginRegistry(),
		loader:     NewGoPluginLoader(logger),
		pluginDirs: config.PluginDirs,
		config:     config,
		logger:     logger,
	}
}

// Initialize initializes the plugin manager.
func (pm *DefaultPluginManager) Initialize() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.initialized {
		return nil
	}

	pm.logger.Info("Initializing plugin manager", "dirs", pm.pluginDirs)

	// Validate configuration
	if err := pm.validateConfig(); err != nil {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginInitError,
			"Invalid plugin manager configuration",
			err,
		)
	}

	// Initialize plugin directories
	if err := pm.initializePluginDirs(); err != nil {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginInitError,
			"Failed to initialize plugin directories",
			err,
		)
	}

	pm.initialized = true
	pm.logger.Info("Plugin manager initialized successfully")
	return nil
}

// LoadPlugins loads all available plugins from configured directories.
func (pm *DefaultPluginManager) LoadPlugins() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.initialized {
		if err := pm.Initialize(); err != nil {
			return err
		}
	}

	pm.logger.Info("Loading plugins from directories", "dirs", pm.pluginDirs)

	var loadErrors []error
	totalLoaded := 0

	for _, dir := range pm.pluginDirs {
		loaded, err := pm.loadPluginsFromDir(dir)
		if err != nil {
			pm.logger.Warn("Failed to load plugins from directory", "dir", dir, "error", err)
			loadErrors = append(loadErrors, err)
			continue
		}
		totalLoaded += loaded
	}

	pm.logger.Info("Plugin loading completed",
		"total_loaded", totalLoaded,
		"total_plugins", len(pm.plugins),
		"errors", len(loadErrors))

	// Return error if no plugins were loaded and there were errors
	if totalLoaded == 0 && len(loadErrors) > 0 {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginLoadError,
			"Failed to load any plugins",
			loadErrors[0],
		)
	}

	return nil
}

// GetPlugin retrieves a plugin by name.
func (pm *DefaultPluginManager) GetPlugin(name string) (interfaces.Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// ListPlugins returns all loaded plugin names.
func (pm *DefaultPluginManager) ListPlugins() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.plugins))
	for name := range pm.plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RegisterPlugin registers a plugin with the manager.
func (pm *DefaultPluginManager) RegisterPlugin(plugin interfaces.Plugin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	name := plugin.Name()
	if name == "" {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginInitError,
			"Plugin name cannot be empty",
			nil,
		)
	}

	// Check if plugin already exists
	if _, exists := pm.plugins[name]; exists {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginInitError,
			fmt.Sprintf("Plugin already registered: %s", name),
			nil,
		)
	}

	// Check plugin count limit
	if pm.config.MaxPlugins > 0 && len(pm.plugins) >= pm.config.MaxPlugins {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginInitError,
			fmt.Sprintf("Maximum plugin limit reached: %d", pm.config.MaxPlugins),
			nil,
		)
	}

	// Validate plugin
	if err := pm.validatePlugin(plugin); err != nil {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginInitError,
			"Plugin validation failed",
			err,
		)
	}

	// Initialize plugin
	pluginConfig := interfaces.PluginConfig{
		Name:    name,
		Version: plugin.Version(),
		Enabled: true,
	}

	if err := plugin.Initialize(pluginConfig); err != nil {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginInitError,
			fmt.Sprintf("Plugin initialization failed: %s", name),
			err,
		)
	}

	// Register plugin
	pm.plugins[name] = plugin
	pm.registry.RegisterPlugin(name, &PluginInfo{
		Metadata: PluginMetadata{
			Name:     name,
			Version:  plugin.Version(),
			LoadedAt: time.Now(),
		},
		Status: PluginStatusInitialized,
		Plugin: plugin,
	})

	pm.logger.Info("Plugin registered successfully", "name", name, "version", plugin.Version())
	return nil
}

// UnloadPlugin unloads a plugin by name.
func (pm *DefaultPluginManager) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginNotFound,
			fmt.Sprintf("Plugin not found: %s", name),
			nil,
		)
	}

	// Cleanup plugin
	if err := plugin.Cleanup(); err != nil {
		pm.logger.Warn("Plugin cleanup failed", "name", name, "error", err)
		// Continue with unloading even if cleanup fails
	}

	// Remove from manager
	delete(pm.plugins, name)
	pm.registry.UnregisterPlugin(name)

	pm.logger.Info("Plugin unloaded successfully", "name", name)
	return nil
}

// GetPluginInfo returns detailed information about a plugin.
func (pm *DefaultPluginManager) GetPluginInfo(name string) (*PluginInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	info, exists := pm.registry.GetPluginInfo(name)
	if !exists {
		return nil, interfaces.NewPluginError(
			interfaces.ErrCodePluginNotFound,
			fmt.Sprintf("Plugin not found: %s", name),
			nil,
		)
	}

	return info, nil
}

// EnablePlugin enables a disabled plugin.
func (pm *DefaultPluginManager) EnablePlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	info, exists := pm.registry.GetPluginInfo(name)
	if !exists {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginNotFound,
			fmt.Sprintf("Plugin not found: %s", name),
			nil,
		)
	}

	if info.Status != PluginStatusDisabled {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginInitError,
			fmt.Sprintf("Plugin is not disabled: %s", name),
			nil,
		)
	}

	info.Status = PluginStatusLoaded
	pm.logger.Info("Plugin enabled", "name", name)
	return nil
}

// DisablePlugin disables a plugin.
func (pm *DefaultPluginManager) DisablePlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	info, exists := pm.registry.GetPluginInfo(name)
	if !exists {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginNotFound,
			fmt.Sprintf("Plugin not found: %s", name),
			nil,
		)
	}

	info.Status = PluginStatusDisabled
	pm.logger.Info("Plugin disabled", "name", name)
	return nil
}

// ExecutePlugin executes a plugin with the given context and arguments.
func (pm *DefaultPluginManager) ExecutePlugin(name string, ctx context.Context, args []string) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginNotFound,
			fmt.Sprintf("Plugin not found: %s", name),
			nil,
		)
	}

	// Check if plugin is enabled
	info, _ := pm.registry.GetPluginInfo(name)
	if info != nil && info.Status == PluginStatusDisabled {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginExecError,
			fmt.Sprintf("Plugin is disabled: %s", name),
			nil,
		)
	}

	pm.logger.Debug("Executing plugin", "name", name, "args", args)

	if err := plugin.Execute(ctx, args); err != nil {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginExecError,
			fmt.Sprintf("Plugin execution failed: %s", name),
			err,
		)
	}

	return nil
}

// Shutdown shuts down the plugin manager and cleans up all plugins.
func (pm *DefaultPluginManager) Shutdown() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.logger.Info("Shutting down plugin manager")

	var errors []error
	for name, plugin := range pm.plugins {
		if err := plugin.Cleanup(); err != nil {
			pm.logger.Warn("Plugin cleanup failed during shutdown", "name", name, "error", err)
			errors = append(errors, err)
		}
	}

	pm.plugins = make(map[string]interfaces.Plugin)
	pm.registry = NewPluginRegistry()
	pm.initialized = false

	if len(errors) > 0 {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginInitError,
			"Some plugins failed to cleanup during shutdown",
			errors[0],
		)
	}

	pm.logger.Info("Plugin manager shutdown completed")
	return nil
}

// validateConfig validates the plugin manager configuration.
func (pm *DefaultPluginManager) validateConfig() error {
	if pm.config.MaxPlugins <= 0 {
		return fmt.Errorf("max_plugins must be greater than 0")
	}

	if pm.config.LoadTimeout <= 0 {
		return fmt.Errorf("load_timeout must be greater than 0")
	}

	return nil
}

// initializePluginDirs ensures plugin directories exist.
func (pm *DefaultPluginManager) initializePluginDirs() error {
	for _, dir := range pm.pluginDirs {
		if !filepath.IsAbs(dir) {
			// Convert relative path to absolute
			absDir, err := filepath.Abs(dir)
			if err != nil {
				return fmt.Errorf("failed to get absolute path for %s: %w", dir, err)
			}
			dir = absDir
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create plugin directory %s: %w", dir, err)
		}
	}

	return nil
}

// loadPluginsFromDir loads plugins from a specific directory.
func (pm *DefaultPluginManager) loadPluginsFromDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			pm.logger.Debug("Plugin directory does not exist", "dir", dir)
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read plugin directory %s: %w", dir, err)
	}

	loaded := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if file is a plugin (e.g., .so file)
		if !pm.isPluginFile(entry.Name()) {
			continue
		}

		// Check plugin count limit
		if len(pm.plugins) >= pm.config.MaxPlugins {
			pm.logger.Warn("Maximum plugin limit reached", "max", pm.config.MaxPlugins)
			break
		}

		pluginPath := filepath.Join(dir, entry.Name())
		if err := pm.loadPluginFromPath(pluginPath); err != nil {
			pm.logger.Warn("Failed to load plugin", "path", pluginPath, "error", err)
			continue
		}

		loaded++
	}

	return loaded, nil
}

// loadPluginFromPath loads a plugin from a specific file path.
func (pm *DefaultPluginManager) loadPluginFromPath(path string) error {
	pm.logger.Debug("Loading plugin from path", "path", path)

	// Security check
	if err := pm.validatePluginPath(path); err != nil {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginLoadError,
			"Plugin path validation failed",
			err,
		)
	}

	// Load plugin with timeout
	ctx, cancel := context.WithTimeout(context.Background(), pm.config.LoadTimeout)
	defer cancel()

	plugin, err := pm.loader.LoadPlugin(ctx, path)
	if err != nil {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginLoadError,
			fmt.Sprintf("Failed to load plugin from %s", path),
			err,
		)
	}

	// Initialize plugin
	config := interfaces.PluginConfig{
		Name:    plugin.Name(),
		Version: plugin.Version(),
		Enabled: true,
		Path:    path,
	}

	if err := plugin.Initialize(config); err != nil {
		return interfaces.NewPluginError(
			interfaces.ErrCodePluginInitError,
			fmt.Sprintf("Failed to initialize plugin %s", plugin.Name()),
			err,
		)
	}

	// Register plugin
	pm.plugins[plugin.Name()] = plugin
	pm.registry.RegisterPlugin(plugin.Name(), &PluginInfo{
		Metadata: PluginMetadata{
			Name:     plugin.Name(),
			Version:  plugin.Version(),
			LoadedAt: time.Now(),
			Path:     path,
		},
		Status: PluginStatusInitialized,
		Plugin: plugin,
	})

	pm.logger.Info("Plugin loaded successfully",
		"name", plugin.Name(),
		"version", plugin.Version(),
		"path", path)

	return nil
}

// validatePlugin validates a plugin before registration.
func (pm *DefaultPluginManager) validatePlugin(plugin interfaces.Plugin) error {
	if plugin.Name() == "" {
		return fmt.Errorf("plugin name is required")
	}

	if plugin.Version() == "" {
		return fmt.Errorf("plugin version is required")
	}

	// Check naming conventions
	if pm.config.AllowedPluginPrefixes != nil && len(pm.config.AllowedPluginPrefixes) > 0 {
		validPrefix := false
		for _, prefix := range pm.config.AllowedPluginPrefixes {
			if strings.HasPrefix(plugin.Name(), prefix) {
				validPrefix = true
				break
			}
		}
		if !validPrefix {
			return fmt.Errorf("plugin name does not have allowed prefix")
		}
	}

	return nil
}

// validatePluginPath validates a plugin file path for security.
func (pm *DefaultPluginManager) validatePluginPath(path string) error {
	// Check blocked patterns
	for _, pattern := range pm.config.SecurityPolicy.BlockedPatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return fmt.Errorf("plugin path matches blocked pattern: %s", pattern)
		}
	}

	// Additional security checks can be added here
	return nil
}

// isPluginFile checks if a file is a valid plugin file.
func (pm *DefaultPluginManager) isPluginFile(filename string) bool {
	ext := filepath.Ext(filename)
	return ext == ".so" || ext == ".dll" || ext == ".dylib"
}

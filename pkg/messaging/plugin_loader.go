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
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
)

// PluginLoader provides functionality to load and manage plugins from various sources.
type PluginLoader struct {
	registry      *PluginRegistry
	loadedPlugins map[string]*plugin.Plugin // Maps plugin name to loaded Go plugin
	mutex         sync.RWMutex
}

// NewPluginLoader creates a new plugin loader with the given registry.
func NewPluginLoader(registry *PluginRegistry) *PluginLoader {
	return &PluginLoader{
		registry:      registry,
		loadedPlugins: make(map[string]*plugin.Plugin),
	}
}

// LoadFromFile loads a plugin from a shared object (.so) file.
// The plugin must export a function named "NewPlugin" that returns (Plugin, error).
func (pl *PluginLoader) LoadFromFile(ctx context.Context, path string, config map[string]interface{}) error {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	// Load the plugin shared object
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", path, err)
	}

	// Look up the NewPlugin function
	symbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return fmt.Errorf("plugin %s does not export NewPlugin function: %w", path, err)
	}

	// Assert the function signature
	newPluginFunc, ok := symbol.(func() (Plugin, error))
	if !ok {
		return fmt.Errorf("plugin %s NewPlugin function has invalid signature", path)
	}

	// Create the plugin instance
	pluginInstance, err := newPluginFunc()
	if err != nil {
		return fmt.Errorf("failed to create plugin from %s: %w", path, err)
	}

	// Initialize the plugin
	if err := pluginInstance.Initialize(ctx, config); err != nil {
		return fmt.Errorf("failed to initialize plugin from %s: %w", path, err)
	}

	// Register the plugin
	if err := pl.registry.RegisterPlugin(pluginInstance); err != nil {
		return fmt.Errorf("failed to register plugin from %s: %w", path, err)
	}

	// Store the loaded plugin reference
	pl.loadedPlugins[pluginInstance.Metadata().Name] = p

	return nil
}

// LoadFromDirectory loads all plugins from a directory.
// Only files with .so extension are considered.
func (pl *PluginLoader) LoadFromDirectory(ctx context.Context, dirPath string, configs map[string]map[string]interface{}) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory %s: %w", dirPath, err)
	}

	var loadErrors []error
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only load .so files
		if !strings.HasSuffix(entry.Name(), ".so") {
			continue
		}

		pluginPath := filepath.Join(dirPath, entry.Name())
		pluginName := strings.TrimSuffix(entry.Name(), ".so")

		config := configs[pluginName]
		if config == nil {
			config = make(map[string]interface{})
		}

		if err := pl.LoadFromFile(ctx, pluginPath, config); err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("failed to load %s: %w", pluginPath, err))
		}
	}

	if len(loadErrors) > 0 {
		return fmt.Errorf("errors loading plugins: %v", loadErrors)
	}

	return nil
}

// LoadFromFactory loads a plugin using a factory function.
// This is useful for statically compiled plugins.
func (pl *PluginLoader) LoadFromFactory(ctx context.Context, name string, factory PluginFactory, config map[string]interface{}) error {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	// Register the factory
	if err := pl.registry.RegisterFactory(name, factory); err != nil {
		return fmt.Errorf("failed to register plugin factory %s: %w", name, err)
	}

	// Create the plugin instance
	pluginInstance, err := factory()
	if err != nil {
		return fmt.Errorf("failed to create plugin %s: %w", name, err)
	}

	// Initialize the plugin
	if err := pluginInstance.Initialize(ctx, config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	// Register the plugin instance
	if err := pl.registry.RegisterPlugin(pluginInstance); err != nil {
		return fmt.Errorf("failed to register plugin %s: %w", name, err)
	}

	// Add to loaded plugins map (for factory-based plugins, we use nil)
	pl.loadedPlugins[pluginInstance.Metadata().Name] = nil

	return nil
}

// LoadMultiple loads multiple plugins from factories.
func (pl *PluginLoader) LoadMultiple(ctx context.Context, plugins map[string]PluginFactory, configs map[string]map[string]interface{}) error {
	var loadErrors []error

	for name, factory := range plugins {
		config := configs[name]
		if config == nil {
			config = make(map[string]interface{})
		}

		if err := pl.LoadFromFactory(ctx, name, factory, config); err != nil {
			loadErrors = append(loadErrors, err)
		}
	}

	if len(loadErrors) > 0 {
		return fmt.Errorf("errors loading plugins: %v", loadErrors)
	}

	return nil
}

// Unload unloads a plugin by name.
func (pl *PluginLoader) Unload(ctx context.Context, name string) error {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	// Unregister from registry (this will stop and shutdown the plugin)
	if err := pl.registry.UnregisterPlugin(ctx, name); err != nil {
		return err
	}

	// Remove from loaded plugins map
	delete(pl.loadedPlugins, name)

	return nil
}

// UnloadAll unloads all plugins.
func (pl *PluginLoader) UnloadAll(ctx context.Context) error {
	pl.mutex.RLock()
	names := make([]string, 0, len(pl.loadedPlugins))
	for name := range pl.loadedPlugins {
		names = append(names, name)
	}
	pl.mutex.RUnlock()

	var errs []error
	for _, name := range names {
		if err := pl.Unload(ctx, name); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors unloading plugins: %v", errs)
	}

	return nil
}

// GetLoadedPlugins returns a list of loaded plugin names.
func (pl *PluginLoader) GetLoadedPlugins() []string {
	pl.mutex.RLock()
	defer pl.mutex.RUnlock()

	names := make([]string, 0, len(pl.loadedPlugins))
	for name := range pl.loadedPlugins {
		names = append(names, name)
	}
	return names
}

// Reload reloads a plugin by unloading and loading it again.
// Note: This only works for factory-based plugins, not file-based ones.
func (pl *PluginLoader) Reload(ctx context.Context, name string, config map[string]interface{}) error {
	// First, unload the existing plugin
	if err := pl.Unload(ctx, name); err != nil {
		return fmt.Errorf("failed to unload plugin %s for reload: %w", name, err)
	}

	// Try to create a new instance from the factory
	plugin, err := pl.registry.CreatePlugin(name)
	if err != nil {
		return fmt.Errorf("failed to create plugin %s for reload: %w", name, err)
	}

	// Initialize the new instance
	if err := plugin.Initialize(ctx, config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s for reload: %w", name, err)
	}

	// Register the new instance
	if err := pl.registry.RegisterPlugin(plugin); err != nil {
		return fmt.Errorf("failed to register plugin %s for reload: %w", name, err)
	}

	return nil
}

// PluginDiscovery provides functionality to discover plugins in the system.
type PluginDiscovery struct {
	searchPaths []string
	mutex       sync.RWMutex
}

// NewPluginDiscovery creates a new plugin discovery with the given search paths.
func NewPluginDiscovery(searchPaths ...string) *PluginDiscovery {
	return &PluginDiscovery{
		searchPaths: searchPaths,
	}
}

// AddSearchPath adds a directory to the search paths.
func (pd *PluginDiscovery) AddSearchPath(path string) {
	pd.mutex.Lock()
	defer pd.mutex.Unlock()
	pd.searchPaths = append(pd.searchPaths, path)
}

// Discover searches for plugins in all search paths and returns their file paths.
func (pd *PluginDiscovery) Discover() ([]string, error) {
	pd.mutex.RLock()
	defer pd.mutex.RUnlock()

	var pluginPaths []string
	var errors []error

	for _, searchPath := range pd.searchPaths {
		entries, err := os.ReadDir(searchPath)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to read directory %s: %w", searchPath, err))
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if strings.HasSuffix(entry.Name(), ".so") {
				pluginPaths = append(pluginPaths, filepath.Join(searchPath, entry.Name()))
			}
		}
	}

	if len(errors) > 0 {
		return pluginPaths, fmt.Errorf("errors during discovery: %v", errors)
	}

	return pluginPaths, nil
}

// DiscoverAndLoad discovers and loads all plugins from search paths.
func (pd *PluginDiscovery) DiscoverAndLoad(ctx context.Context, loader *PluginLoader, configs map[string]map[string]interface{}) error {
	pluginPaths, err := pd.Discover()
	if err != nil {
		return err
	}

	var loadErrors []error
	for _, path := range pluginPaths {
		pluginName := strings.TrimSuffix(filepath.Base(path), ".so")
		config := configs[pluginName]
		if config == nil {
			config = make(map[string]interface{})
		}

		if err := loader.LoadFromFile(ctx, path, config); err != nil {
			loadErrors = append(loadErrors, err)
		}
	}

	if len(loadErrors) > 0 {
		return fmt.Errorf("errors loading discovered plugins: %v", loadErrors)
	}

	return nil
}
